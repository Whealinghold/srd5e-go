package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// effectWork is the scratch state of one pure transition of the registry: the entries as they
// change, which ones are gone, and what the transition produced, in order. It holds no port and
// rolls nothing.
type effectWork struct {
	entries  []ActiveEffect
	gone     []bool
	events   []ActiveEffectEvent
	toApply  []EffectConditionPlan
	toRemove []EffectConditionPlan
	damage   []EffectDamagePlan
	ended    []ActiveEffect
}

func newEffectWork(r ActiveEffects) *effectWork {
	return &effectWork{entries: slices.Clone(r.entries), gone: make([]bool, len(r.entries))}
}

// end removes entry i for the reason once; ending it again is harmless (FR-046). When runOnEnd, the
// onEnd actions of its current stage add their damage plans (FR-038). taker is the combatant of the tick.
func (w *effectWork) end(i int, reason EffectEndReason, runOnEnd bool, taker CombatantID) {
	if w.gone[i] {
		return
	}
	e := w.entries[i]
	w.gone[i] = true
	w.ended = append(w.ended, e)
	w.events = append(w.events, ActiveEffectEnded{baseOf(e), reason})
	if runOnEnd {
		for _, a := range e.stageDef().onEnd {
			if p, ok := planEffectDamage(e, a, taker); ok {
				w.damage = append(w.damage, p)
			}
		}
	}
}

// planEffectDamage is the damage plan of action a of e, if it plans any. A subject audience on a
// place plans nothing (FR-035), nor does stored damage that is not stored.
func planEffectDamage(e ActiveEffect, a EffectAction, taker CombatantID) (EffectDamagePlan, bool) {
	var amount dice.Expression
	var typ content.DamageType
	var save EffectDamageSave
	var audience EffectDamageAudience
	switch v := a.(type) {
	case DealDamage:
		amount, typ, save, audience = v.amount, v.typ, v.save, v.audience
	case DealStoredDamage:
		amount, typ, save, audience = e.stored, v.typ, v.save, v.audience
	default:
		return EffectDamagePlan{}, false
	}
	if amount.IsZero() {
		return EffectDamagePlan{}, false
	}
	p := EffectDamagePlan{effect: e.id, audience: audience, amount: amount, typ: typ, save: save}
	switch audience {
	case EffectAudienceSubject:
		c, ok := creatureOf(e.subject)
		if !ok {
			return EffectDamagePlan{}, false
		}
		p.targets = []CombatantID{c.id}
	case EffectAudienceTurnTaker:
		p.targets = []CombatantID{taker}
	case EffectAudienceAreaMembers:
		// the host lists the creatures when it resolves the damage
	}
	return p, true
}

// heldElsewhere reports whether a remaining effect other than skip holds kind on creature c.
func (w *effectWork) heldElsewhere(skip int, c CombatantID, kind content.ConditionKind) bool {
	for j, r := range w.entries {
		if j != skip && !w.gone[j] && r.OnCreature(c) && r.holds(kind) {
			return true
		}
	}
	return false
}

// advance moves entry i to its next stage, planning the conditions to add and the ones to remove
// (FR-034). active is the combatant active after the tick.
func (w *effectWork) advance(i int, active CombatantID) {
	e := w.entries[i]
	next := e.enter(e.stage+1, active)
	cs, onCreature := creatureOf(e.subject)
	if c := cs.id; onCreature {
		newKinds := next.stageDef().conditions
		for _, k := range e.held {
			if !slices.Contains(newKinds, k) && !w.heldElsewhere(i, c, k) {
				w.toRemove = append(w.toRemove, EffectConditionPlan{e.id, c, k})
			}
		}
		for _, k := range newKinds {
			if !slices.Contains(e.held, k) {
				w.toApply = append(w.toApply, EffectConditionPlan{e.id, c, k})
			}
		}
		next.held = slices.Clone(newKinds)
	}
	w.entries[i] = next
	w.events = append(w.events, ActiveEffectStageAdvanced{baseOf(next), next.stage})
}

// finish returns the registry without the effects that are gone, and plans the removal of the
// conditions of ended effects that no remaining effect holds (FR-045).
func (w *effectWork) finish() ActiveEffects {
	remaining := ActiveEffects{}
	for i, e := range w.entries {
		if !w.gone[i] {
			remaining.entries = append(remaining.entries, e)
		}
	}
	for _, e := range w.ended {
		cs, ok := creatureOf(e.subject)
		if !ok {
			continue
		}
		c := cs.id
		for _, k := range e.held {
			p := EffectConditionPlan{e.id, c, k}
			held := slices.ContainsFunc(remaining.entries, func(r ActiveEffect) bool { return r.OnCreature(c) && r.holds(k) })
			if !held && !slices.ContainsFunc(w.toRemove, func(q EffectConditionPlan) bool { return q.Target == c && q.Kind == k }) {
				w.toRemove = append(w.toRemove, p)
			}
		}
	}
	return remaining
}
