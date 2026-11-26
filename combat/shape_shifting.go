package combat

import (
	"errors"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
)

// ShapeShiftState is whether a creature is in its true form or in a form an effect
// gave it. The zero value is TrueForm(). Build a shifted state with ShiftedForm.
// Comparable and safe for concurrent use.
type ShapeShiftState struct{ effect EffectSource }

// TrueForm returns the state of a creature in its true form: the zero value.
func TrueForm() ShapeShiftState { return ShapeShiftState{} }

// ShiftedForm returns the state of a creature shifted by effect. It returns an
// error matching srderr.ErrInvalidInput for the zero source. It never panics.
func ShiftedForm(effect EffectSource) (ShapeShiftState, error) {
	if effect.IsZero() {
		return ShapeShiftState{}, invalidInput("a shifted form needs an effect source")
	}
	return ShapeShiftState{effect}, nil
}

// IsShifted reports whether the creature is in a shifted form.
func (s ShapeShiftState) IsShifted() bool { return !s.effect.IsZero() }

// Effect returns the effect that shifted the creature, the zero source in true form.
func (s ShapeShiftState) Effect() EffectSource { return s.effect }

// CarryOverRule is the closed set of what carries over when a creature changes form:
// everything (the default) or everything except what is named (SRD 5.2.1,
// "Shape-Shifting", p. 187: "Unless that description says otherwise, any ongoing
// effects on you ... carry over"). A nil CarryOverRule means CarryOverAll. Variants:
// CarryOverAll, CarryOverExcept.
type CarryOverRule interface{ isCarryOverRule() }

// CarryOverAll: every condition, spell and curse carries over.
type CarryOverAll struct{}

// CarryOverExcept: everything carries over except the named explicit conditions and
// ongoing effects. Build it with NewCarryOverExcept; it holds slices, so it is not
// comparable with ==.
type CarryOverExcept struct {
	conditions []content.ConditionKind
	effects    []OngoingEffectID
}

// NewCarryOverExcept returns the rule. Both lists are copied and repeats are kept
// once. It returns an error matching srderr.ErrInvalidInput for an invalid condition
// kind or a blank effect id, and a LimitExceededError for more than 256 conditions or 256 effect ids. It never panics.
func NewCarryOverExcept(conditions []content.ConditionKind, effects []OngoingEffectID) (CarryOverExcept, error) {
	if err := errors.Join(overCap("carry-over conditions", len(conditions), maxInputList), overCap("carry-over effect ids", len(effects), maxInputList)); err != nil {
		return CarryOverExcept{}, err
	}
	var r CarryOverExcept
	for _, k := range conditions {
		if !k.Valid() {
			return CarryOverExcept{}, invalidInput("invalid condition kind")
		}
		if !slices.Contains(r.conditions, k) {
			r.conditions = append(r.conditions, k)
		}
	}
	for _, id := range effects {
		if strings.TrimSpace(string(id)) == "" {
			return CarryOverExcept{}, invalidInput("blank ongoing effect id")
		}
		if !slices.Contains(r.effects, id) {
			r.effects = append(r.effects, id)
		}
	}
	return r, nil
}

// Conditions returns a copy of the condition kinds that do not carry over.
func (r CarryOverExcept) Conditions() []content.ConditionKind { return slices.Clone(r.conditions) }

// Effects returns a copy of the effect ids that do not carry over.
func (r CarryOverExcept) Effects() []OngoingEffectID { return slices.Clone(r.effects) }

func (CarryOverAll) isCarryOverRule()    {}
func (CarryOverExcept) isCarryOverRule() {}

// NotCarriedOver lists what a change of form removed: condition kinds and effect
// ids, each once. The zero value lists nothing.
type NotCarriedOver struct {
	conditions []content.ConditionKind
	effects    []OngoingEffectID
}

// Conditions returns a copy of the removed condition kinds in the order the rule
// named them, except that Unconscious comes first.
func (n NotCarriedOver) Conditions() []content.ConditionKind { return slices.Clone(n.conditions) }

// Effects returns a copy of the removed effect ids in the order the rule named them.
func (n NotCarriedOver) Effects() []OngoingEffectID { return slices.Clone(n.effects) }

// Empty reports whether nothing was removed.
func (n NotCarriedOver) Empty() bool { return len(n.conditions) == 0 && len(n.effects) == 0 }

// ShiftShapeCommand is the host's facts for a shift. Construct it with a keyed literal.
type ShiftShapeCommand struct {
	// Creature is the shifting creature.
	Creature CombatantID
	// State is the creature's form before the shift.
	State ShapeShiftState
	// Conditions and Effects are what the creature has before the shift.
	Conditions ConditionSet
	Effects    OngoingEffects
	// Effect is the effect that shifts the creature; the zero source is invalid.
	Effect EffectSource
	// Rule says what carries over; nil means CarryOverAll.
	Rule CarryOverRule
}

// ShiftShapeResult is the closed set of answers of ShapeShifting.Shift. Variants:
// ShiftShapeApplied, ShiftShapeExceptionNotHeld.
type ShiftShapeResult interface{ isShiftShapeResult() }

// shapeChange is the state after a change of form.
type shapeChange struct {
	state      ShapeShiftState
	conditions ConditionSet
	effects    OngoingEffects
	omitted    NotCarriedOver
	events     []MagicalEffectEvent
}

// State returns the form after the change.
func (c shapeChange) State() ShapeShiftState { return c.state }

// Conditions returns the creature's conditions after the change.
func (c shapeChange) Conditions() ConditionSet { return c.conditions }

// Effects returns the creature's ongoing effects after the change.
func (c shapeChange) Effects() OngoingEffects { return c.effects }

// NotCarriedOver returns what the change removed.
func (c shapeChange) NotCarriedOver() NotCarriedOver { return c.omitted }

// Events returns a copy of the events, in order.
func (c shapeChange) Events() []MagicalEffectEvent { return cloneMagicalEvents(c.events) }

// notHeld is the part of a result that refuses a CarryOverExcept naming what the
// creature does not hold.
type notHeld struct {
	conditions []content.ConditionKind
	effects    []OngoingEffectID
}

// Conditions returns a copy of the named condition kinds that are not held as
// explicit conditions (absent, or held only by implication).
func (n notHeld) Conditions() []content.ConditionKind { return slices.Clone(n.conditions) }

// Effects returns a copy of the named effect ids that are not in the list.
func (n notHeld) Effects() []OngoingEffectID { return slices.Clone(n.effects) }

// ShiftShapeApplied: the creature is in the shifted form. State is ShiftedForm of
// the effect (replacing an earlier shift), Conditions and Effects are what carried
// over, and Events holds one ShapeShifted.
type ShiftShapeApplied struct{ shapeChange }

// ShiftShapeExceptionNotHeld: the CarryOverExcept names a condition kind the
// creature does not hold as an explicit condition, or an effect id it does not
// have; nothing changed and no event.
type ShiftShapeExceptionNotHeld struct{ notHeld }

func (ShiftShapeApplied) isShiftShapeResult()          {}
func (ShiftShapeExceptionNotHeld) isShiftShapeResult() {}

// ShapeShifting is the stateless service for Shape-Shifting. The zero value is
// ready to use. Safe for concurrent use.
type ShapeShifting struct{}

// Shift shifts the creature into a form (SRD 5.2.1, "Shape-Shifting", p. 187). The
// rule decides what carries over: CarryOverAll leaves the conditions and effects as
// given; CarryOverExcept removes each named explicit condition and effect and lists
// them in NotCarriedOver. A shift while shifted succeeds and replaces the effect. A
// named condition kind that is not explicit (absent, or held only because another
// condition implies it) or an effect id not in the list gives
// ShiftShapeExceptionNotHeld. A named Exhaustion is removed at every level and
// listed once.
//
// It returns an error matching srderr.ErrInvalidInput, with the nil result, for a
// zero creature or the zero effect source. It does not change its arguments and
// never panics.
func (ShapeShifting) Shift(c ShiftShapeCommand) (ShiftShapeResult, error) {
	if c.Creature.IsZero() || c.Effect.IsZero() {
		return nil, invalidInput("a shift needs a creature and an effect")
	}
	change, missing := applyCarryOver(c.Creature, c.Conditions, c.Effects, c.Rule)
	if missing != nil {
		return ShiftShapeExceptionNotHeld{*missing}, nil
	}
	change.state = ShapeShiftState{c.Effect}
	change.events = []MagicalEffectEvent{ShapeShifted{magicalEffectEventBase{c.Creature}, c.Effect}}
	return ShiftShapeApplied{change}, nil
}

// RevertToTrueFormCommand is the host's facts for a return to true form. Construct
// it with a keyed literal.
type RevertToTrueFormCommand struct {
	// Creature is the reverting creature.
	Creature CombatantID
	// State is the creature's form before the reversion.
	State ShapeShiftState
	// Conditions and Effects are what the creature has before the reversion.
	Conditions ConditionSet
	Effects    OngoingEffects
	// Rule says what carries back, from the effect's description; nil means CarryOverAll.
	Rule CarryOverRule
}

// RevertToTrueFormResult is the closed set of answers of ShapeShifting.Revert.
// Variants: RevertToTrueFormApplied, RevertToTrueFormExceptionNotHeld.
type RevertToTrueFormResult interface{ isRevertToTrueFormResult() }

// RevertToTrueFormApplied: the creature is in its true form. A creature already in
// true form gets the same state, conditions and effects and no event; a shifted one
// gets TrueForm, what carried back and one RevertedToTrueForm.
type RevertToTrueFormApplied struct{ shapeChange }

// RevertToTrueFormExceptionNotHeld: as ShiftShapeExceptionNotHeld, for the rule of a reversion.
type RevertToTrueFormExceptionNotHeld struct{ notHeld }

func (RevertToTrueFormApplied) isRevertToTrueFormResult()          {}
func (RevertToTrueFormExceptionNotHeld) isRevertToTrueFormResult() {}

// Revert returns the creature to its true form, carrying ongoing effects back by
// the rule of a shift ("from one form to the other", SRD 5.2.1, "Shape-Shifting",
// p. 187). A creature in true form is unchanged. It returns an error matching
// srderr.ErrInvalidInput, with the nil result, for a zero creature. It does not
// change its arguments and never panics.
func (ShapeShifting) Revert(c RevertToTrueFormCommand) (RevertToTrueFormResult, error) {
	if c.Creature.IsZero() {
		return nil, invalidInput("a reversion needs a creature")
	}
	if !c.State.IsShifted() {
		return RevertToTrueFormApplied{shapeChange{state: c.State, conditions: c.Conditions, effects: c.Effects}}, nil
	}
	change, missing := applyCarryOver(c.Creature, c.Conditions, c.Effects, c.Rule)
	if missing != nil {
		return RevertToTrueFormExceptionNotHeld{*missing}, nil
	}
	change.events = []MagicalEffectEvent{RevertedToTrueForm{magicalEffectEventBase{c.Creature}, RevertCauseEnded}}
	return RevertToTrueFormApplied{change}, nil
}

// RevertOnDeathResult is the closed set of answers of ShapeShifting.RevertOnDeath.
// Variants: RevertOnDeathApplied, RevertOnDeathNotDead.
type RevertOnDeathResult interface{ isRevertOnDeathResult() }

// RevertOnDeathApplied: the dead creature is in its true form. Events holds one
// RevertedToTrueForm with cause RevertCauseDied when it was shifted, none otherwise.
type RevertOnDeathApplied struct {
	events []MagicalEffectEvent
}

// State returns TrueForm.
func (RevertOnDeathApplied) State() ShapeShiftState { return ShapeShiftState{} }

// Events returns a copy of the events.
func (r RevertOnDeathApplied) Events() []MagicalEffectEvent { return cloneMagicalEvents(r.events) }

// RevertOnDeathNotDead: the creature is not Dead; nothing changed and no event.
type RevertOnDeathNotDead struct{}

func (RevertOnDeathApplied) isRevertOnDeathResult() {}
func (RevertOnDeathNotDead) isRevertOnDeathResult() {}

// RevertOnDeath returns a creature that died to its true form (SRD 5.2.1,
// "Shape-Shifting", p. 187: "You revert to your true form if you die"). The host
// calls it after Died. A creature whose life state is not a valid Dead (the zero Dead has no cause) gets
// RevertOnDeathNotDead whatever its form; a Dead creature in true form gets Applied
// with no event. A zero creature is not a Dead creature. It never panics.
func (ShapeShifting) RevertOnDeath(creature CombatantID, state ShapeShiftState, life LifeState) RevertOnDeathResult {
	if d, dead := life.(Dead); !dead || !d.cause.Valid() || creature.IsZero() {
		return RevertOnDeathNotDead{}
	}
	if !state.IsShifted() {
		return RevertOnDeathApplied{}
	}
	return RevertOnDeathApplied{[]MagicalEffectEvent{RevertedToTrueForm{magicalEffectEventBase{creature}, RevertCauseDied}}}
}

// applyCarryOver applies the rule to the conditions and effects. It returns the
// change (state and events unset), or what the rule names that is not held.
func applyCarryOver(creature CombatantID, conditions ConditionSet, effects OngoingEffects, rule CarryOverRule) (shapeChange, *notHeld) {
	change := shapeChange{conditions: conditions, effects: effects}
	except, ok := rule.(CarryOverExcept)
	if !ok {
		return change, nil
	}
	held := map[content.ConditionKind]bool{}
	exhaustion := 0
	for _, c := range conditions.Explicit() {
		held[c.Kind()] = true
		if e, isExhaustion := c.(ExhaustionCondition); isExhaustion {
			exhaustion = e.Level.Value()
		}
	}
	var missing notHeld
	for _, k := range except.conditions {
		if !held[k] {
			missing.conditions = append(missing.conditions, k)
		}
	}
	for _, id := range except.effects {
		if _, found := effects.Find(id); !found {
			missing.effects = append(missing.effects, id)
		}
	}
	if len(missing.conditions) > 0 || len(missing.effects) > 0 {
		return shapeChange{}, &missing
	}
	// Unconscious first: removing it leaves the creature Prone, so a named Prone is removed after it.
	kinds := slices.Clone(except.conditions)
	if i := slices.Index(kinds, content.ConditionUnconscious); i > 0 {
		kinds = slices.Insert(slices.Delete(kinds, i, i+1), 0, content.ConditionUnconscious)
	}
	for _, k := range kinds {
		if k == content.ConditionExhaustion {
			// The single-level Remove would only lower it, so take every level off at once.
			if r, err := change.conditions.ReduceExhaustion(creature, exhaustion); err == nil {
				change.conditions = r.Set()
			}
		} else {
			change.conditions = change.conditions.Remove(creature, k).Set()
		}
		change.omitted.conditions = append(change.omitted.conditions, k)
	}
	change.effects = effects.without(except.effects)
	change.omitted.effects = slices.Clone(except.effects)
	return change, nil
}
