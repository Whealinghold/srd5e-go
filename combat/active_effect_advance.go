package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// effectSaves is what advancing asks of the use case: one saving throw. attackFlow implements it;
// the domain never sees a dice port.
type effectSaves interface {
	resolveSave(a d20.SaveAttempt, dc d20.DifficultyClass) (UnarmedStrikeSave, error)
}

// clockOutcome is what a tick did to the lifetime of one effect.
type clockOutcome int

const (
	clockNone clockOutcome = iota
	clockStageEnded
	clockOverallEnded
)

// ticked applies a tick to the clocks of e (FR-020 to FR-024, FR-061, FR-062) and says whether the
// stage or the whole effect ran out. It changes nothing else.
func (e ActiveEffect) ticked(t EffectClockTick) (ActiveEffect, clockOutcome) {
	count := func(n int) {
		if _, ok := e.duration.(EffectLastsRounds); ok {
			e.remaining -= n
		}
		if _, ok := e.template.overall.(EffectLastsRounds); ok {
			e.overallLeft -= n
		}
	}
	until, isUntil := e.duration.(EffectLastsUntilTurnBoundary)
	boundary := false
	switch v := t.(type) {
	case TurnStartReached:
		if e.anchor == v.combatant {
			count(1)
		}
		boundary = isUntil && until.owner == v.combatant && until.boundary == content.BoundaryStart
	case TurnEndReached:
		if isUntil && until.owner == v.combatant && until.boundary == content.BoundaryEnd {
			boundary = !e.skipsTurnEnd
			e.skipsTurnEnd = false
		}
	case RoundStartReached:
		if e.anchor.IsZero() {
			count(1)
		}
	case TimeElapsed:
		count(v.rounds)
	case EndOfCombatReached:
		e.anchor = ""
	case CombatantLeft:
		if e.anchor == v.combatant {
			e.anchor = ""
		}
		if isUntil && until.owner == v.combatant {
			e.duration, e.remaining, e.anchor, e.skipsTurnEnd = EffectLastsRounds{1}, 1, "", false
		}
	}
	_, overall := e.template.overall.(EffectLastsRounds)
	_, rounds := e.duration.(EffectLastsRounds)
	switch {
	case overall && e.overallLeft <= 0:
		return e, clockOverallEnded
	case boundary || rounds && e.remaining <= 0:
		return e, clockStageEnded
	}
	return e, clockNone
}

// effectAdvance is everything advanceEffects reads.
type effectAdvance struct {
	effects ActiveEffects
	tick    EffectClockTick
	area    []ActiveEffectID
	order   []ActiveEffectID
	saves   map[CombatantID]map[ability.Ability]d20.SaveAttempt
	active  CombatantID
}

// turnTick is the combatant and boundary of a turn tick. It exists because no function of this package returns a
// CombatantID.
type turnTick struct {
	combatant CombatantID
	boundary  content.TurnBoundary
}

// turnOf returns the turn tick of t, if t is the start or the end of a turn.
func turnOf(t EffectClockTick) (turnTick, bool) {
	switch v := t.(type) {
	case TurnStartReached:
		return turnTick{v.combatant, content.BoundaryStart}, true
	case TurnEndReached:
		return turnTick{v.combatant, content.BoundaryEnd}, true
	}
	return turnTick{}, false
}

// matches reports whether the trigger fires for e at the tick (FR-027).
func (t EffectTrigger) matches(e ActiveEffect, tick EffectClockTick, area []ActiveEffectID) bool {
	turn, ok := turnOf(tick)
	if !ok || turn.boundary != t.boundary {
		return false
	}
	c := turn.combatant
	switch t.whose {
	case EffectTurnOfCaster:
		return e.caster == c
	case EffectTurnOfSubject:
		return e.OnCreature(c)
	case EffectTurnOfCreatureInArea:
		return slices.Contains(area, e.id)
	}
	return false
}

// firingTriggers returns the triggers of the current stage of e that fire at the tick.
func firingTriggers(e ActiveEffect, tick EffectClockTick, area []ActiveEffectID) []EffectTrigger {
	var out []EffectTrigger
	for _, t := range e.stageDef().triggers {
		if t.matches(e, tick, area) {
			out = append(out, t)
		}
	}
	return out
}

// savePair is one saving throw a tick needs: whose and which ability.
type savePair struct {
	c CombatantID
	a ability.Ability
}

// advanceOutcome is what advanceEffects produced.
type advanceOutcome struct {
	rejection ActiveEffectRejection
	work      *effectWork
	effects   ActiveEffects
}

// advanceEffects applies one tick (FR-025): effects whose duration ends at the tick end or enter
// their next stage, with their onEnd actions and gates; then the triggers of the applying effects
// that remain fire, in the host's order or registry order (FR-028). It returns a rejection before
// any die when the order is not a permutation of the effects that fire or a needed saving throw
// has no attempt (FR-063). The effects that fire are those that apply before the tick: an effect
// that becomes the applying one because another ended at this tick fires from the next tick on.
// A failing saves leaves no partial result.
func advanceEffects(in effectAdvance, rolls effectSaves) (advanceOutcome, error) {
	w := newEffectWork(in.effects)
	outcomes := make([]clockOutcome, len(w.entries))
	for i, e := range in.effects.entries {
		w.entries[i], outcomes[i] = e.ticked(in.tick)
	}
	applying := applyingFlags(in.effects.entries)
	var firing []int
	for i, e := range w.entries {
		if outcomes[i] == clockNone && applying[i] && len(firingTriggers(e, in.tick, in.area)) > 0 {
			firing = append(firing, i)
		}
	}
	proc, bad := processingOrder(w.entries, firing, in.order)
	if len(bad) > 0 {
		return advanceOutcome{rejection: rejection(bad...), effects: in.effects}, nil
	}
	var missing []ActiveEffectProblem
	seen := map[savePair]bool{}
	need := func(e ActiveEffect, a ability.Ability) {
		cs, ok := creatureOf(e.subject)
		c := cs.id
		if p := (savePair{c, a}); ok && !seen[p] {
			seen[p] = true
			if in.saves[c][a] == nil {
				missing = append(missing, ActiveEffectProblem{EffectRejectMissingSaveAttempt, e.id, c, a})
			}
		}
	}
	for i, e := range w.entries {
		if g, ok := e.stageDef().Gate(); ok && outcomes[i] == clockStageEnded {
			need(e, g.ability)
		}
	}
	for _, i := range proc {
		for _, t := range firingTriggers(w.entries[i], in.tick, in.area) {
			for _, a := range t.actions {
				if rs, ok := a.(RepeatSave); ok {
					need(w.entries[i], rs.ability)
				}
			}
		}
	}
	if len(missing) > 0 {
		return advanceOutcome{rejection: rejection(missing...), effects: in.effects}, nil
	}
	if err := expireEffects(w, outcomes, in, rolls); err != nil {
		return advanceOutcome{}, err
	}
	turn, _ := turnOf(in.tick)
	taker := turn.combatant
	for _, i := range proc {
		if err := fireTriggers(w, i, in, taker, rolls); err != nil {
			return advanceOutcome{}, err
		}
	}
	return advanceOutcome{work: w, effects: w.finish()}, nil
}

// processingOrder returns the entries to fire in the order to process them: the host's order, or
// registry order when it gives none. It returns a problem for each id of the order that is repeated
// or does not fire, and each firing effect it omits (FR-028).
func processingOrder(entries []ActiveEffect, firing []int, order []ActiveEffectID) ([]int, []ActiveEffectProblem) {
	if len(order) == 0 {
		return firing, nil
	}
	var bad []ActiveEffectProblem
	var proc []int
	seen := map[ActiveEffectID]bool{}
	for _, id := range order {
		i := slices.IndexFunc(entries, func(e ActiveEffect) bool { return e.id == id })
		if i < 0 || !slices.Contains(firing, i) || seen[id] {
			bad = append(bad, ActiveEffectProblem{Reason: EffectRejectInvalidOrder, Effect: id})
			continue
		}
		seen[id] = true
		proc = append(proc, i)
	}
	for _, i := range firing {
		if !seen[entries[i].id] {
			bad = append(bad, ActiveEffectProblem{Reason: EffectRejectInvalidOrder, Effect: entries[i].id})
		}
	}
	return proc, bad
}

// expireEffects ends or advances, in registry order, every effect whose clock ran out at the tick
// (FR-025 step 1, FR-034, FR-061). A gate runs once; the last stage's expiry ends the effect.
func expireEffects(w *effectWork, outcomes []clockOutcome, in effectAdvance, rolls effectSaves) error {
	for i, o := range outcomes {
		e := w.entries[i]
		switch {
		case o == clockOverallEnded:
			w.end(i, EffectExpired, true, "")
		case o != clockStageEnded:
		default:
			if g, ok := e.stageDef().Gate(); ok && isCreature(e.subject) {
				saved, err := w.save(i, g, EffectSaveStageGate, in, rolls)
				if err != nil {
					return err
				}
				if saved {
					w.end(i, EffectRepeatSaveSucceeded, true, "")
					continue
				}
			}
			if e.stage == len(e.template.stages)-1 {
				w.end(i, EffectExpired, true, "")
			} else {
				w.advance(i, in.active)
			}
		}
	}
	return nil
}

// save makes the saving throw of the creature entry i is on and records it (FR-029, FR-031).
func (w *effectWork) save(i int, s RepeatSave, kind EffectSaveKind, in effectAdvance, rolls effectSaves) (bool, error) {
	e := w.entries[i]
	cs, _ := creatureOf(e.subject)
	res, err := rolls.resolveSave(in.saves[cs.id][s.ability], s.dc)
	if err != nil {
		return false, err
	}
	w.events = append(w.events, ActiveEffectRepeatSaveMade{baseOf(e), s.ability, kind, res.Outcome()})
	return res.Outcome() == d20.Success, nil
}

// fireTriggers runs the actions of the triggers of entry i that fire at the tick, in order
// (FR-025 step 2). A repeated save that succeeds ends the effect and stops its actions.
func fireTriggers(w *effectWork, i int, in effectAdvance, taker CombatantID, rolls effectSaves) error {
	for _, t := range firingTriggers(w.entries[i], in.tick, in.area) {
		for _, a := range t.actions {
			e := w.entries[i]
			switch v := a.(type) {
			case RepeatSave:
				saved, err := w.save(i, v, EffectSaveRepeat, in, rolls)
				if err != nil {
					return err
				}
				if saved {
					w.end(i, EffectRepeatSaveSucceeded, true, taker)
					return nil
				}
			case GrowStoredDamage:
				e.stored = grownDamage(e.stored, v.extra)
				w.entries[i] = e
				w.events = append(w.events, ActiveEffectStoredDamageGrew{baseOf(e), e.stored})
			default:
				if p, ok := planEffectDamage(e, a, taker); ok {
					w.damage = append(w.damage, p)
				}
			}
		}
	}
	return nil
}

// grownDamage adds extra dice to stored (FR-037), saturating at the limits of an Expression.
func grownDamage(stored, extra dice.Expression) dice.Expression {
	count := min(stored.Count()+extra.Count(), dice.MaxCount)
	mod := max(min(stored.Modifier()+extra.Modifier(), dice.MaxModifier), -dice.MaxModifier)
	if grown, err := dice.NewExpression(count, stored.Die(), mod); err == nil {
		return grown
	}
	return stored
}
