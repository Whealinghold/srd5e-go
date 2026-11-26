package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// GrappleRules is the stateless domain service of the grapple rules (SRD 5.2.1,
// "Grappling", p. 182; "Unarmed Strike", p. 190). Its methods are pure,
// synchronous and silent, and read or change no Grapples value. The zero value is
// ready to use; a consumer that wants to replace one declares its own interface.
// Safe for concurrent use.
type GrappleRules struct{}

// GrappleDC returns 8 plus the ability modifier plus the Proficiency Bonus, the DC
// of the target's save and of every escape attempt (SRD 5.2.1, "Unarmed Strike",
// Grapple, p. 190). There is no cap or floor; the sum saturates at the limits of
// int.
func (GrappleRules) GrappleDC(abilityModifier, proficiencyBonus int) d20.DifficultyClass {
	return d20.DifficultyClass(satAdd(satAdd(8, abilityModifier), proficiencyBonus))
}

// OptionSaveAbilities returns the abilities of the saving throw the target of a
// Grapple chooses between: Strength and Dexterity, in that order (SRD 5.2.1,
// "Unarmed Strike", p. 190).
func (GrappleRules) OptionSaveAbilities() []ability.Ability {
	return []ability.Ability{ability.Strength, ability.Dexterity}
}

// EscapeOptions returns the two checks a Grappled creature may make to escape,
// Strength (Athletics) then Dexterity (Acrobatics) (SRD 5.2.1, "Grappling", p. 182).
func (GrappleRules) EscapeOptions() []GrappleEscapeOption {
	return []GrappleEscapeOption{GrappleEscapeOptionStrengthAthletics, GrappleEscapeOptionDexterityAcrobatics}
}

// GrapplePossibility is the closed set of answers of GrappleRules.Possibility.
// Variants: GrapplePossible, GrappleNotPossible.
type GrapplePossibility interface{ isGrapplePossibility() }

// GrapplePossible: the creature may use the Grapple option.
type GrapplePossible struct{}

// GrappleNotPossible: the creature may not use the Grapple option, for the reason.
// Construct it with a keyed literal.
type GrappleNotPossible struct{ Reason GrappleNotPossibleReason }

func (GrapplePossible) isGrapplePossibility()    {}
func (GrappleNotPossible) isGrapplePossibility() {}

// Possibility says whether an attacker of the size may use the Grapple option on a
// target of the other size: not when the target is more than one size larger than
// the attacker (GrappleNotPossibleReasonTargetTooLarge, checked first), not when
// hand is not HandAvailabilityHandFree (GrappleNotPossibleReasonNoFreeHand) (SRD
// 5.2.1, "Unarmed Strike", p. 190). Whether a hand is free is the host's
// statement. Sizes compare in the order of content.CreatureSize.
func (GrappleRules) Possibility(attacker, target content.CreatureSize, hand HandAvailability) GrapplePossibility {
	switch {
	case target > attacker+1:
		return GrappleNotPossible{GrappleNotPossibleReasonTargetTooLarge}
	case hand != HandAvailabilityHandFree:
		return GrappleNotPossible{GrappleNotPossibleReasonNoFreeHand}
	}
	return GrapplePossible{}
}

// Hold says whether a grapple holds (SRD 5.2.1, "Grappling", "Ending a Grapple",
// p. 182): GrappleHoldGrapplerIncapacitated when the grappler is Incapacitated,
// directly or by implication, or WithGrapplerDead(true); otherwise
// GrappleHoldOutOfRange when distanceFeet is greater than rangeFeet; otherwise
// GrappleHoldHolds. A distance equal to the range holds, and Incapacitated wins
// when both apply.
//
// It returns an error matching srderr.ErrInvalidInput, naming the distance first,
// for a negative distance or range. It never panics.
func (GrappleRules) Hold(grapplerConditions ConditionSet, distanceFeet, rangeFeet int, opts ...GrappleOption) (GrappleHold, error) {
	switch {
	case distanceFeet < 0:
		return 0, invalidInput("negative distance")
	case rangeFeet < 0:
		return 0, invalidInput("negative range")
	}
	return holdOf(grapplerConditions, distanceFeet, rangeFeet, newGrappleConfig(opts).dead), nil
}

// holdOf is Hold for numbers already checked.
func holdOf(grapplerConditions ConditionSet, distanceFeet, rangeFeet int, dead bool) GrappleHold {
	switch {
	case dead || (ConditionEffects{}).IsIncapacitated(grapplerConditions):
		return GrappleHoldGrapplerIncapacitated
	case distanceFeet > rangeFeet:
		return GrappleHoldOutOfRange
	}
	return GrappleHoldHolds
}

// MovementCost returns the extra feet each foot of movement costs a grappler
// dragging or carrying a Grappled creature: 0 when the creature is Tiny or two or
// more sizes smaller than the grappler, otherwise 1 (SRD 5.2.1, "Grappled
// [Condition]", "Movable", p. 182). An invalid size costs 1.
func (GrappleRules) MovementCost(grapplerSize, grappledSize content.CreatureSize) int {
	if grapplerSize.Valid() && grappledSize.Valid() && (grappledSize == content.SizeTiny || grappledSize <= grapplerSize-2) {
		return 0
	}
	return 1
}

// GrappleEndOnSetResult is the closed set of outcomes of GrappleRules.EndOnSet.
// Variants: GrappleEndOnSetApplied, GrappleEndOnSetNotGrappled.
type GrappleEndOnSetResult interface{ isGrappleEndOnSetResult() }

// GrappleEndOnSetApplied: Grappled was removed. It carries the new conditions and
// the events ConditionRemoved then GrappleEnded.
type GrappleEndOnSetApplied struct {
	conditions ConditionSet
	events     []Event
}

// Conditions returns the set without Grappled.
func (r GrappleEndOnSetApplied) Conditions() ConditionSet { return r.conditions }

// Events returns a copy of the events, ConditionRemoved then GrappleEnded.
func (r GrappleEndOnSetApplied) Events() []Event { return append([]Event(nil), r.events...) }

// GrappleEndOnSetNotGrappled: the set does not hold Grappled (or the input is not
// usable), so nothing changed and there is no event.
type GrappleEndOnSetNotGrappled struct{}

func (GrappleEndOnSetApplied) isGrappleEndOnSetResult()     {}
func (GrappleEndOnSetNotGrappled) isGrappleEndOnSetResult() {}

// EndOnSet ends a grapple on the target's conditions: a host or the encounter ends
// a grapple through it, never through a plain ConditionSet.Remove, so that
// GrappleEnded always follows ConditionRemoved. When set holds the explicit
// condition Grappled it returns GrappleEndOnSetApplied with the set minus Grappled
// (through ConditionSet.Remove) and the events ConditionRemoved then GrappleEnded,
// the grappler read from the removed condition, which the removal event does not
// carry. Otherwise, and for a zero target or a cause that is not one of the four, it
// returns GrappleEndOnSetNotGrappled with no event. The input set is unchanged.
func (GrappleRules) EndOnSet(set ConditionSet, target CombatantID, cause GrappleEndCause) GrappleEndOnSetResult {
	grappled, ok := set.slots[content.ConditionGrappled].(Grappled)
	if !ok || target.IsZero() || !cause.Valid() {
		return GrappleEndOnSetNotGrappled{}
	}
	cleared, ok := set.Remove(target, content.ConditionGrappled).(ConditionCleared)
	if !ok { // unreachable: an explicit condition is always cleared
		return GrappleEndOnSetNotGrappled{}
	}
	ended := GrappleEnded{grappleEventBase{target, grappled.Grappler}, cause}
	return GrappleEndOnSetApplied{cleared.set, append(asConditionEvents(cleared.events), ended)}
}
