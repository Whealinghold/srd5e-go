package d20

import "github.com/Whealinghold/srd5e-go/ability"

// Result is the resolved outcome of one D20 Test, carrying the whole
// breakdown. The variants are [RolledCheck], [RolledAttack] and
// [ForcedFailure], all values; handle them with a type switch covering all
// three. Variants hold slices, so interface values must not be compared with
// ==, which panics; compare the concrete types field by field instead.
type Result interface {
	// Ability returns the command's ability label, unchanged.
	Ability() ability.Ability
	// Kind returns the kind of D20 Test resolved.
	Kind() Kind
	// Outcome returns [Success] or [Failure].
	Outcome() Outcome
	result()
}

// RolledCheck is a rolled ability check or saving throw.
type RolledCheck struct {
	ability ability.Ability
	check   CheckResult
}

// RolledAttack is a rolled attack roll.
type RolledAttack struct {
	ability ability.Ability
	attack  AttackRollResult
}

// ForcedFailure is a check or saving throw failed without a roll. It still
// names the Difficulty Class it would have been compared with.
type ForcedFailure struct {
	ability ability.Ability
	failure ForcedFailureResult
	target  DifficultyClass
}

// Ability returns the command's ability label.
func (r RolledCheck) Ability() ability.Ability { return r.ability }

// Kind returns [AbilityCheck] or [SavingThrow].
func (r RolledCheck) Kind() Kind { return r.check.kind }

// Outcome returns the outcome of the check.
func (r RolledCheck) Outcome() Outcome { return r.check.outcome }

// Check returns the resolved check with its whole breakdown.
func (r RolledCheck) Check() CheckResult { return r.check }

// Equal reports whether r and o are the same result field by field.
func (r RolledCheck) Equal(o RolledCheck) bool {
	return r.ability == o.ability && r.check.Equal(o.check)
}

// Ability returns the command's ability label.
func (r RolledAttack) Ability() ability.Ability { return r.ability }

// Kind returns [AttackRoll].
func (r RolledAttack) Kind() Kind { return AttackRoll }

// Outcome returns [Success] for a hit and [Failure] for a miss.
func (r RolledAttack) Outcome() Outcome { return r.attack.outcome }

// Attack returns the resolved attack roll with its whole breakdown.
func (r RolledAttack) Attack() AttackRollResult { return r.attack }

// Equal reports whether r and o are the same result field by field.
func (r RolledAttack) Equal(o RolledAttack) bool {
	return r.ability == o.ability && r.attack.Equal(o.attack)
}

// Ability returns the command's ability label.
func (r ForcedFailure) Ability() ability.Ability { return r.ability }

// Kind returns [AbilityCheck] or [SavingThrow].
func (r ForcedFailure) Kind() Kind { return r.failure.kind }

// Outcome returns [Failure].
func (r ForcedFailure) Outcome() Outcome { return Failure }

// Failure returns the forced failure with its kind and reason.
func (r ForcedFailure) Failure() ForcedFailureResult { return r.failure }

// Target returns the Difficulty Class the test would have been compared with.
func (r ForcedFailure) Target() DifficultyClass { return r.target }

func (RolledCheck) result()   {}
func (RolledAttack) result()  {}
func (ForcedFailure) result() {}

var (
	_ Result = RolledCheck{}
	_ Result = RolledAttack{}
	_ Result = ForcedFailure{}
)

// NewRolledCheck wraps a resolved check with the ability label of the command
// that produced it, so a consumer can build a canned [Result] for its own
// tests. It returns an [*InvalidError] with [ReasonUnknownAbility] for an
// invalid ability and [ReasonZeroRoll] for the zero [CheckResult].
func NewRolledCheck(a ability.Ability, c CheckResult) (RolledCheck, error) {
	if !a.Valid() {
		return RolledCheck{}, invalid(ReasonUnknownAbility, a.String())
	}
	if c.roll.IsZero() {
		return RolledCheck{}, invalid(ReasonZeroRoll, "check result is the zero value")
	}
	return RolledCheck{ability: a, check: c}, nil
}

// NewRolledAttack wraps a resolved attack roll with the ability label of the
// command that produced it. It returns an [*InvalidError] with
// [ReasonUnknownAbility] for an invalid ability and [ReasonZeroRoll] for the
// zero [AttackRollResult].
func NewRolledAttack(a ability.Ability, r AttackRollResult) (RolledAttack, error) {
	if !a.Valid() {
		return RolledAttack{}, invalid(ReasonUnknownAbility, a.String())
	}
	if r.roll.IsZero() {
		return RolledAttack{}, invalid(ReasonZeroRoll, "attack result is the zero value")
	}
	return RolledAttack{ability: a, attack: r}, nil
}

// NewForcedFailure wraps a forced failure with the ability label and the
// Difficulty Class it would have been compared with. It returns an
// [*InvalidError] with [ReasonUnknownAbility] for an invalid ability and
// [ReasonUnknownKind] for the zero [ForcedFailureResult].
func NewForcedFailure(a ability.Ability, f ForcedFailureResult, target DifficultyClass) (ForcedFailure, error) {
	if !a.Valid() {
		return ForcedFailure{}, invalid(ReasonUnknownAbility, a.String())
	}
	if !f.kind.Valid() {
		return ForcedFailure{}, invalid(ReasonUnknownKind, "forced failure is the zero value")
	}
	return ForcedFailure{ability: a, failure: f, target: target}, nil
}
