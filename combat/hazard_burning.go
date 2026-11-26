package combat

import "github.com/Whealinghold/srd5e-go/dice"

// BurningState says whether a creature is on fire (SRD 5.2.1, "Burning
// [Hazard]", p. 178). Burning is a hazard, not a condition. The zero value is
// BurningStateNotBurning.
type BurningState int

// The two states.
const (
	// BurningStateNotBurning: the creature is not on fire (the zero value).
	BurningStateNotBurning BurningState = iota
	// BurningStateBurning: the creature is on fire and takes damage each turn.
	BurningStateBurning
)

// Valid reports whether s is one of the two states.
func (s BurningState) Valid() bool { return s == BurningStateNotBurning || s == BurningStateBurning }

// String returns "not_burning", "burning" or "unknown".
func (s BurningState) String() string {
	switch s {
	case BurningStateNotBurning:
		return "not_burning"
	case BurningStateBurning:
		return "burning"
	}
	return "unknown"
}

// BurningRejection is why a Burning operation was refused. Refusals are results,
// not errors. The zero value is not a rejection.
type BurningRejection int

// The rejections.
const (
	// BurningRejectionNotBurning: the creature is not on fire.
	BurningRejectionNotBurning BurningRejection = iota + 1
	// BurningRejectionExtinguishBlockedByImmunity: the creature is immune to
	// Prone, so rolling on the ground cannot put the fire out.
	BurningRejectionExtinguishBlockedByImmunity
)

// String returns the snake_case name of r (the srd5e.combat.result value), or
// "unknown".
func (r BurningRejection) String() string {
	return enumName([]string{"not_burning", "blocked_by_immunity"}, int(r))
}

// ExtinguishMethod is how a burning creature puts the fire out (SRD 5.2.1,
// "Burning [Hazard]", p. 178). The zero value is invalid.
type ExtinguishMethod int

// The methods.
const (
	// ExtinguishMethodRollOnTheGround: giving oneself the Prone condition.
	ExtinguishMethodRollOnTheGround ExtinguishMethod = iota + 1
	// ExtinguishMethodDoused: being doused with water.
	ExtinguishMethodDoused
	// ExtinguishMethodSubmerged: being submerged.
	ExtinguishMethodSubmerged
	// ExtinguishMethodSuffocated: the fire being starved of air.
	ExtinguishMethodSuffocated
)

// Valid reports whether m is one of the four methods.
func (m ExtinguishMethod) Valid() bool {
	return m >= ExtinguishMethodRollOnTheGround && m <= ExtinguishMethodSuffocated
}

// String returns the snake_case name of m, or "unknown".
func (m ExtinguishMethod) String() string {
	return enumName([]string{"roll_on_the_ground", "doused", "submerged", "suffocated"}, int(m))
}

// ProneApplicationOutcome is the optional Prone part of ExtinguishBurning.
// Variants: ProneApplied, ProneNotApplied.
type ProneApplicationOutcome interface{ isProneApplicationOutcome() }

// ProneApplied: Prone was applied through ConditionSet.Apply; Application is
// that call's result (ConditionAdded, ConditionAlreadyHeld or ConditionImmune).
type ProneApplied struct{ Application ApplyConditionToSetResult }

// ProneNotApplied: the method does not involve Prone, so no call was made.
type ProneNotApplied struct{}

func (ProneApplied) isProneApplicationOutcome()    {}
func (ProneNotApplied) isProneApplicationOutcome() {}

// burningDamageDie is the 1d4 Fire damage of a burning creature's turn.
func burningDamageDie() dice.Expression { return dice.MustExpression(1, dice.D4(), 0) }

// extinguish is the rule of ExtinguishBurning: it decides the new state, the
// condition set and the condition events. It returns the rejection, 0 when the
// fire went out.
func extinguish(state BurningState, method ExtinguishMethod, set ConditionSet, target CombatantID) (
	BurningRejection, ConditionSet, ProneApplicationOutcome, []Event) {
	if state != BurningStateBurning {
		return BurningRejectionNotBurning, set, ProneNotApplied{}, nil
	}
	if method != ExtinguishMethodRollOnTheGround {
		return 0, set, ProneNotApplied{}, nil
	}
	app := set.Apply(target, Prone{})
	switch a := app.(type) {
	case ConditionImmune:
		return BurningRejectionExtinguishBlockedByImmunity, set, ProneNotApplied{}, nil
	case ConditionAdded:
		var events []Event
		for _, e := range a.events {
			events = append(events, e)
		}
		return 0, a.set, ProneApplied{app}, events
	}
	return 0, app.Set(), ProneApplied{app}, nil
}
