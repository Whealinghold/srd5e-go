package d20

import "github.com/Whealinghold/srd5e-go/ability"

// Command is the input of [Resolver.Resolve]: one D20 Test of one kind,
// pairing the kind with the target number it is compared with and the attempt
// that resolves it. The variants are [AbilityCheckCommand],
// [SavingThrowCommand] and [AttackRollCommand], all values. The pairing is
// enforced by the types: a Difficulty Class cannot be paired with an attack
// roll, nor can an attack roll be forced to fail. Variants hold slices, so
// interface values must not be compared with ==, which panics.
type Command interface {
	kind() Kind
	abilityLabel() ability.Ability
	validate() error
}

// AbilityCheckCommand is an ability check ("Ability Checks", SRD 5.2.1, p. 6). Construct with keyed
// literals.
type AbilityCheckCommand struct {
	// Ability labels the check for telemetry and the result. The modifier
	// summed is the one in the attempt; the caller keeps the two consistent.
	Ability ability.Ability
	// Target is the Difficulty Class the total is compared with.
	Target DifficultyClass
	// Attempt is a RollAttempt or an AutomaticFailure.
	Attempt CheckAttempt
}

// SavingThrowCommand is a saving throw ("Saving Throws", SRD 5.2.1, p. 7). Construct with keyed
// literals.
type SavingThrowCommand struct {
	// Ability labels the save for telemetry and the result.
	Ability ability.Ability
	// Target is the Difficulty Class the total is compared with.
	Target DifficultyClass
	// Attempt is a RollAttempt, a ChooseToFail or an AutomaticFailure.
	Attempt SaveAttempt
}

// AttackRollCommand is an attack roll ("Attack Rolls", SRD 5.2.1, p. 7). It
// always rolls, with the default critical range. Construct with keyed
// literals.
type AttackRollCommand struct {
	// Ability labels the attack for telemetry and the result.
	Ability ability.Ability
	// Target is the Armor Class the total is compared with.
	Target ArmorClass
	// Attempt is the rolling attempt.
	Attempt RollAttempt
}

func (AbilityCheckCommand) kind() Kind { return AbilityCheck }
func (SavingThrowCommand) kind() Kind  { return SavingThrow }
func (AttackRollCommand) kind() Kind   { return AttackRoll }

func (c AbilityCheckCommand) abilityLabel() ability.Ability { return c.Ability }
func (c SavingThrowCommand) abilityLabel() ability.Ability  { return c.Ability }
func (c AttackRollCommand) abilityLabel() ability.Ability   { return c.Ability }

func (c AbilityCheckCommand) validate() error {
	if err := validAbility(c.Ability); err != nil {
		return err
	}
	switch c.Attempt.(type) {
	case RollAttempt, AutomaticFailure:
		return nil
	}
	return invalid(ReasonMissingAttempt, "ability check attempt is nil or unknown")
}

func (c SavingThrowCommand) validate() error {
	if err := validAbility(c.Ability); err != nil {
		return err
	}
	switch c.Attempt.(type) {
	case RollAttempt, ChooseToFail, AutomaticFailure:
		return nil
	}
	return invalid(ReasonMissingAttempt, "saving throw attempt is nil or unknown")
}

func (c AttackRollCommand) validate() error { return validAbility(c.Ability) }

func validAbility(a ability.Ability) error {
	if !a.Valid() {
		return invalid(ReasonUnknownAbility, "ability label is not one of the six abilities")
	}
	return nil
}
