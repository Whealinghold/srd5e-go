package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// Flying (SRD 5.2.1, "Flying", "Fly Speed", p. 182; "Hover", p. 183). Pure and silent:
// whether a flier falls depends on the facts alone, never on the turn or the movement left.

// HoverAbility says whether a creature can hover. The zero value is HoverAbilityNone.
type HoverAbility int

// The two values. The host reads it from the stat block or from the effect that grants it.
const (
	HoverAbilityNone HoverAbility = iota
	HoverAbilityHover
)

// Valid reports whether h is one of the two values.
func (h HoverAbility) Valid() bool { return h == HoverAbilityNone || h == HoverAbilityHover }

// FlightFallCause is why a flier falls, in the order the causes are listed. The zero value is invalid.
type FlightFallCause int

// The three causes.
const (
	FlightFallCauseIncapacitated FlightFallCause = iota + 1
	FlightFallCauseProne
	FlightFallCauseFlySpeedZero
)

// Valid reports whether c is one of the three causes.
func (c FlightFallCause) Valid() bool {
	return c >= FlightFallCauseIncapacitated && c <= FlightFallCauseFlySpeedZero
}

// String returns "incapacitated", "prone", "fly_speed_zero" or "unknown".
func (c FlightFallCause) String() string {
	return enumName([]string{"incapacitated", "prone", "fly_speed_zero"}, int(c))
}

// FlightFallCauses is a set of causes, each once. The zero value is the empty set.
// Comparable with ==.
type FlightFallCauses struct{ mask uint8 }

func (s FlightFallCauses) with(c FlightFallCause) FlightFallCauses {
	return FlightFallCauses{s.mask | 1<<c}
}

// Has reports whether c is in the set.
func (s FlightFallCauses) Has(c FlightFallCause) bool { return c.Valid() && s.mask&(1<<c) != 0 }

// Slice returns a new slice of the causes in the fixed order Incapacitated, Prone,
// FlySpeedZero; it is empty for the empty set.
func (s FlightFallCauses) Slice() []FlightFallCause {
	var out []FlightFallCause
	for c := FlightFallCauseIncapacitated; c <= FlightFallCauseFlySpeedZero; c++ {
		if s.Has(c) {
			out = append(out, c)
		}
	}
	return out
}

// FlightFacts is what decides whether a flier falls: its conditions, its base Fly Speed in
// feet (before the effect of conditions, after any change that is not a condition) and its
// HoverAbility. Build it with NewFlightFacts. The zero value is a creature with a Fly Speed
// of 0 and no conditions. Comparable with ==.
type FlightFacts struct {
	conditions       ConditionSet
	baseFlySpeedFeet int
	hover            HoverAbility
}

// NewFlightFacts returns the facts. It returns an error matching srderr.ErrInvalidInput for
// a negative base Fly Speed or an invalid hover ability. It never panics.
func NewFlightFacts(conditions ConditionSet, baseFlySpeedFeet int, hover HoverAbility) (FlightFacts, error) {
	if baseFlySpeedFeet < 0 {
		return FlightFacts{}, invalidInput("negative Fly Speed")
	}
	if !hover.Valid() {
		return FlightFacts{}, invalidInput("invalid hover ability")
	}
	return FlightFacts{conditions, baseFlySpeedFeet, hover}, nil
}

// FlightAssessment is the closed set of verdicts of AssessFlight. Variants:
// FlightStaysAloft, FlightFalls.
//
//sumtype:decl
type FlightAssessment interface{ isFlightAssessment() }

// FlightStaysAloft is a flier that stays aloft. OverriddenCauses are the causes hovering
// overrode, empty when none applied.
type FlightStaysAloft struct{ OverriddenCauses FlightFallCauses }

// FlightFalls is a flier that falls. Causes is never empty. The distance, damage and landing
// belong to the falling hazard.
type FlightFalls struct{ Causes FlightFallCauses }

func (FlightStaysAloft) isFlightAssessment() {}
func (FlightFalls) isFlightAssessment()      {}

// AssessFlight says whether a flier falls: it does when it is Incapacitated or Prone
// (also by implication) or its effective Fly Speed, the base after the conditions, is 0,
// unless it can hover. Equal facts give equal results. It cannot fail.
func AssessFlight(f FlightFacts) FlightAssessment {
	var causes FlightFallCauses
	if (ConditionEffects{}).IsIncapacitated(f.conditions) {
		causes = causes.with(FlightFallCauseIncapacitated)
	}
	if slices.Contains(f.conditions.Effective(), content.ConditionProne) {
		causes = causes.with(FlightFallCauseProne)
	}
	// The base Fly Speed is non-negative, the only way Speed fails.
	if speed, _ := (ConditionEffects{}).Speed(f.conditions, f.baseFlySpeedFeet); speed.Value == 0 {
		causes = causes.with(FlightFallCauseFlySpeedZero)
	}
	if causes == (FlightFallCauses{}) || f.hover == HoverAbilityHover {
		return FlightStaysAloft{causes}
	}
	return FlightFalls{causes}
}
