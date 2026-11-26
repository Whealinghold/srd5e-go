package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ActiveEffect is one effect in force: where it came from, who cast it, what it is on, its
// template, and where its clocks stand (FR-002). It is an immutable value; a change returns a new
// one. The zero ActiveEffect is invalid (see IsZero). It holds slices, so compare with Equal, not
// ==. No function of this package returns a CombatantID: read the creatures with CasterIs and
// OnCreature, or from the events. Safe for concurrent use.
type ActiveEffect struct {
	id           ActiveEffectID
	origin       EffectOrigin
	caster       CombatantID
	subject      EffectSubject
	binding      ConcentrationBinding
	template     EffectTemplate
	stage        int
	duration     EffectDuration // the duration of the current stage; CombatantLeft can replace it
	remaining    int            // rounds left of the stage clock, for EffectLastsRounds
	overallLeft  int            // rounds left of the overall clock, when it is EffectLastsRounds
	anchor       CombatantID    // whose turn starts count the rounds; zero outside combat
	skipsTurnEnd bool           // an effect started on its owner's turn skips that turn's end (FR-024)
	stored       dice.Expression
	held         []content.ConditionKind
}

// IsZero reports whether e is the invalid zero ActiveEffect.
func (e ActiveEffect) IsZero() bool { return e.id.IsZero() }

// ID returns the id.
func (e ActiveEffect) ID() ActiveEffectID { return e.id }

// Origin returns where the effect comes from.
func (e ActiveEffect) Origin() EffectOrigin { return e.origin }

// Subject returns what the effect is on.
func (e ActiveEffect) Subject() EffectSubject { return e.subject }

// CasterIs reports whether c cast the effect.
func (e ActiveEffect) CasterIs(c CombatantID) bool { return e.caster == c }

// OnCreature reports whether the effect is on the creature c.
func (e ActiveEffect) OnCreature(c CombatantID) bool {
	cs, ok := creatureOf(e.subject)
	return ok && cs.id == c
}

// SpellID returns the spell of a spell effect and whether there is one (FR-064).
func (e ActiveEffect) SpellID() (content.SpellID, bool) {
	o, ok := e.origin.(SpellEffectOrigin)
	return o.spell, ok
}

// CastAtLevel returns the level a spell effect was cast at and whether there is one (FR-064).
func (e ActiveEffect) CastAtLevel() (int, bool) {
	o, ok := e.origin.(SpellEffectOrigin)
	return o.level, ok
}

// Binding returns the Concentration binding and whether the effect has one.
func (e ActiveEffect) Binding() (ConcentrationBinding, bool) { return e.binding, !e.binding.IsZero() }

// Template returns the template the effect was started from.
func (e ActiveEffect) Template() EffectTemplate { return e.template }

// Potency returns the potency of the template.
func (e ActiveEffect) Potency() EffectPotency { return e.template.potency }

// Stage returns the index of the current stage.
func (e ActiveEffect) Stage() int { return e.stage }

// Duration returns how long the current stage lasts.
func (e ActiveEffect) Duration() EffectDuration { return e.duration }

// RemainingRounds returns the rounds left of the stage clock; 0 for a stage that is not
// EffectLastsRounds.
func (e ActiveEffect) RemainingRounds() int { return e.remaining }

// OverallRemainingRounds returns the rounds left of the overall clock and whether it counts
// rounds (an indefinite or absent overall clock reports false).
func (e ActiveEffect) OverallRemainingRounds() (int, bool) {
	_, ok := e.template.overall.(EffectLastsRounds)
	return e.overallLeft, ok
}

// Anchored reports whether turn starts of a combatant count the rounds of the effect.
func (e ActiveEffect) Anchored() bool { return !e.anchor.IsZero() }

// AnchoredTo reports whether turn starts of c count the rounds of the effect.
func (e ActiveEffect) AnchoredTo(c CombatantID) bool { return !c.IsZero() && e.anchor == c }

// SkipsCurrentTurnEnd reports whether the effect skips the end of the turn it started in.
func (e ActiveEffect) SkipsCurrentTurnEnd() bool { return e.skipsTurnEnd }

// StoredDamage returns the damage the effect has stored and whether it stores any.
func (e ActiveEffect) StoredDamage() (dice.Expression, bool) { return e.stored, !e.stored.IsZero() }

// Conditions returns the conditions the effect holds on its creature, as a new slice.
func (e ActiveEffect) Conditions() []content.ConditionKind { return slices.Clone(e.held) }

// Equal reports whether e and o are the same effect in the same state.
func (e ActiveEffect) Equal(o ActiveEffect) bool {
	return e.id == o.id && e.origin == o.origin && e.caster == o.caster && e.subject == o.subject &&
		e.binding == o.binding && e.template.Equal(o.template) && e.stage == o.stage && e.duration == o.duration &&
		e.remaining == o.remaining && e.overallLeft == o.overallLeft && e.anchor == o.anchor &&
		e.skipsTurnEnd == o.skipsTurnEnd && e.stored == o.stored && slices.Equal(e.held, o.held)
}

// stageDef is the current stage of the template.
func (e ActiveEffect) stageDef() EffectStage { return e.template.stages[e.stage] }

// holds reports whether the effect holds the condition kind on its creature.
func (e ActiveEffect) holds(k content.ConditionKind) bool { return slices.Contains(e.held, k) }

// enter returns the effect at the start of stage, with its clock set from the stage duration and
// anchored at active (FR-013, FR-034). A remainder stage has no clock of its own.
func (e ActiveEffect) enter(stage int, active CombatantID) ActiveEffect {
	d := e.template.stages[stage].duration
	e.stage, e.duration, e.anchor, e.remaining, e.skipsTurnEnd = stage, d, active, 0, false
	switch v := d.(type) {
	case EffectLastsRounds:
		e.remaining = v.rounds
	case EffectLastsUntilTurnBoundary:
		e.skipsTurnEnd = !active.IsZero() && active == v.owner && v.boundary == content.BoundaryEnd
	}
	return e
}

// ActiveEffects is the registry of active effects: entries in the order started, held by the host
// (Q2). The zero ActiveEffects is the empty registry. Immutable: every transition returns a new
// value. Safe for concurrent use.
type ActiveEffects struct{ entries []ActiveEffect }

// NewActiveEffects returns the registry of effects in that order, copying them. It returns an
// error matching srderr.ErrInvalidInput for a zero effect, two effects with one id or a creature holding more than 64 (plan P10). It never
// panics.
func NewActiveEffects(effects ...ActiveEffect) (ActiveEffects, error) {
	seen := map[ActiveEffectID]bool{}
	for _, e := range effects {
		if e.IsZero() || seen[e.id] {
			return ActiveEffects{}, invalidInput("a registry holds each active effect once")
		}
		seen[e.id] = true
	}
	r := ActiveEffects{slices.Clone(effects)}
	for _, e := range r.entries {
		if c, ok := creatureOf(e.subject); ok && r.heldBy(c) > maxEffectsPerCreature {
			return ActiveEffects{}, invalidInput("a creature holds at most 64 active effects (limit_exceeded)")
		}
	}
	return r, nil
}

// maxEffectsPerCreature is the cap of plan P10 on the active effects whose subject is one creature.
const maxEffectsPerCreature = 64

// heldBy counts the effects whose subject is the creature.
func (r ActiveEffects) heldBy(c CreatureEffectSubject) int {
	n := 0
	for _, e := range r.entries {
		if e.subject == EffectSubject(c) {
			n++
		}
	}
	return n
}

// Len returns the number of effects.
func (r ActiveEffects) Len() int { return len(r.entries) }

// Effects returns the effects in the order started, as a new slice.
func (r ActiveEffects) Effects() []ActiveEffect { return slices.Clone(r.entries) }

// Find returns the effect with the id and whether the registry holds it.
func (r ActiveEffects) Find(id ActiveEffectID) (ActiveEffect, bool) {
	i := r.index(id)
	if i < 0 {
		return ActiveEffect{}, false
	}
	return r.entries[i], true
}

// Position returns the position of the effect with the id in registry order (0 first) and whether
// the registry holds it (FR-064).
func (r ActiveEffects) Position(id ActiveEffectID) (int, bool) {
	i := r.index(id)
	return i, i >= 0
}

func (r ActiveEffects) index(id ActiveEffectID) int {
	return slices.IndexFunc(r.entries, func(e ActiveEffect) bool { return e.id == id })
}

// ForCreature returns the effects on the creature, in registry order, as a new slice.
func (r ActiveEffects) ForCreature(c CombatantID) []ActiveEffect {
	return r.filter(func(e ActiveEffect) bool { return e.OnCreature(c) })
}

// ForPlace returns the effects on the place, in registry order, as a new slice.
func (r ActiveEffects) ForPlace(p EffectPlaceID) []ActiveEffect {
	return r.filter(func(e ActiveEffect) bool { s, ok := e.subject.(PlaceEffectSubject); return ok && s.place == p })
}

// CastBy returns the effects cast by the creature, in registry order, as a new slice.
func (r ActiveEffects) CastBy(c CombatantID) []ActiveEffect {
	return r.filter(func(e ActiveEffect) bool { return e.caster == c })
}

// BoundTo returns the effects bound to the Concentration binding, in registry order, as a new
// slice. The zero binding matches nothing.
func (r ActiveEffects) BoundTo(b ConcentrationBinding) []ActiveEffect {
	return r.filter(func(e ActiveEffect) bool { return !b.IsZero() && e.binding == b })
}

func (r ActiveEffects) filter(keep func(ActiveEffect) bool) []ActiveEffect {
	out := []ActiveEffect{}
	for _, e := range r.entries {
		if keep(e) {
			out = append(out, e)
		}
	}
	return out
}

// Equal reports whether r and o hold equal effects in the same order.
func (r ActiveEffects) Equal(o ActiveEffects) bool {
	return slices.EqualFunc(r.entries, o.entries, ActiveEffect.Equal)
}

// ApplyingEffect is an effect that applies to its subject (FR-064) with its position in registry
// order. The zero value is invalid.
type ApplyingEffect struct {
	effect   ActiveEffect
	position int
}

// Effect returns the effect.
func (a ApplyingEffect) Effect() ActiveEffect { return a.effect }

// Position returns the position in registry order: distinct per effect and increasing with start
// order, the castOrder of a spell-granted modifier source.
func (a ApplyingEffect) Position() int { return a.position }

// EffectCombination is the stateless service of Combining Spell Effects (SRD 5.2.1, p. 106). The
// zero value is ready to use.
type EffectCombination struct{}

// combiningKey is what the effects of one spell must share to combine (FR-015).
type combiningKey struct {
	spell    content.SpellID
	onCaster bool
	who      string
}

// applyingFlags reports, for each entry, whether it applies: an effect of another origin always
// does; among the effects of one spell sharing a combining key the one with the highest potency does,
// and on equal potency the one later in the registry.
func applyingFlags(entries []ActiveEffect) []bool {
	best := map[combiningKey]int{}
	keys := make([]combiningKey, len(entries))
	for i, e := range entries {
		o, ok := e.origin.(SpellEffectOrigin)
		if !ok {
			continue
		}
		k := combiningKey{spell: o.spell, onCaster: e.template.combine == CombiningOnCaster}
		if k.onCaster {
			k.who = string(e.caster)
		} else {
			k.who = subjectKey(e.subject)
		}
		keys[i] = k
		if j, seen := best[k]; !seen || entries[j].Potency() <= e.Potency() {
			best[k] = i
		}
	}
	out := make([]bool, len(entries))
	for i, e := range entries {
		if _, ok := e.origin.(SpellEffectOrigin); !ok {
			out[i] = true
			continue
		}
		out[i] = best[keys[i]] == i
	}
	return out
}

// subjectKey is a string that tells two subjects apart.
func subjectKey(s EffectSubject) string {
	switch v := s.(type) {
	case CreatureEffectSubject:
		return "c:" + string(v.id)
	case PlaceEffectSubject:
		return "p:" + string(v.place)
	}
	return ""
}

// Applying returns the effects on subject that apply, in registry order (FR-018, FR-064): one per
// spell group, the most potent and, on a tie, the latest started. A suppressed effect stays in the
// registry and ages, and applies again when the effect that suppressed it ends. It changes nothing.
func (EffectCombination) Applying(effects ActiveEffects, subject EffectSubject) []ApplyingEffect {
	flags := applyingFlags(effects.entries)
	out := []ApplyingEffect{}
	for i, e := range effects.entries {
		if flags[i] && e.subject == subject {
			out = append(out, ApplyingEffect{e, i})
		}
	}
	return out
}
