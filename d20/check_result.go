package d20

import "github.com/Whealinghold/srd5e-go/modifier"

// CheckResult is the outcome of one rolled ability check or saving throw: the
// roll, the modifiers, the total and the Difficulty Class it was compared
// with. It carries everything needed to render a breakdown. There is no
// critical-hit notion: the natural 20 and 1 rules apply to attack rolls only.
//
// It is immutable, built only by [ResolveCheck] (or [CheckResult.AdjustedBy]),
// and not comparable with ==; use [CheckResult.Equal]. The zero value is not a
// result.
type CheckResult struct {
	kind      Kind
	roll      Roll
	modifiers RollModifiers
	total     int
	target    DifficultyClass
	outcome   Outcome
}

// Kind returns [AbilityCheck] or [SavingThrow].
func (c CheckResult) Kind() Kind { return c.kind }

// Roll returns the roll the test was made with.
func (c CheckResult) Roll() Roll { return c.roll }

// Modifiers returns the modifiers added to the natural value.
func (c CheckResult) Modifiers() RollModifiers { return c.modifiers }

// Total returns the natural value plus the modifiers, unclamped.
func (c CheckResult) Total() int { return c.total }

// Target returns the Difficulty Class the total was compared with.
func (c CheckResult) Target() DifficultyClass { return c.target }

// Outcome returns [Success] or [Failure].
func (c CheckResult) Outcome() Outcome { return c.outcome }

// AdjustedBy returns the result of the same test with delta applied after the
// roll (an effect that adds or subtracts, "Add Modifiers", SRD 5.2.1, p. 6):
// delta joins the other modifiers and the outcome is re-resolved. A delta of 0
// is allowed. c is unchanged. It returns the [ResolveCheck] error for the zero
// result.
func (c CheckResult) AdjustedBy(delta int) (CheckResult, error) {
	return ResolveCheck(c.kind, c.roll, c.modifiers.WithOthers(modifier.New(delta)), c.target)
}

// Equal reports whether c and o are the same result field by field.
func (c CheckResult) Equal(o CheckResult) bool {
	return c.kind == o.kind && c.roll == o.roll && c.modifiers.Equal(o.modifiers) &&
		c.total == o.total && c.target == o.target && c.outcome == o.outcome
}
