package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// HitPoints are a creature's current Hit Points, Hit Point maximum and
// Temporary Hit Points (SRD 5.2.1, "Hit Points", p. 16; "Temporary Hit Points",
// p. 18). Current is 0 to the maximum; Temporary Hit Points are a buffer, not
// Hit Points. It is comparable. The zero value (0, 0, 0) is valid: it is the
// Hit Points of a creature whose maximum is 0. Safe for concurrent use.
type HitPoints struct{ current, maximum, temporary int }

// HitPointsProblem is one reason NewHitPoints rejected its arguments. The zero
// value is not a problem.
type HitPointsProblem int

// The problems NewHitPoints reports, in the order it lists them.
const (
	// CurrentBelowZero: current is negative.
	CurrentBelowZero HitPointsProblem = iota + 1
	// CurrentAboveMaximum: current exceeds the maximum.
	CurrentAboveMaximum
	// MaximumBelowZero: the maximum is negative.
	MaximumBelowZero
	// TemporaryBelowZero: Temporary Hit Points are negative.
	TemporaryBelowZero
)

// String returns the snake_case name of p, or "unknown".
func (p HitPointsProblem) String() string {
	switch p {
	case CurrentBelowZero:
		return "current_below_zero"
	case CurrentAboveMaximum:
		return "current_above_maximum"
	case MaximumBelowZero:
		return "maximum_below_zero"
	case TemporaryBelowZero:
		return "temporary_below_zero"
	}
	return "unknown"
}

// HitPointsError reports every problem NewHitPoints found. It matches
// srderr.ErrInvalidInput with errors.Is, is never built with no problems and
// carries no display text.
type HitPointsError struct{ problems []HitPointsProblem }

// Problems returns a copy of the problems, in the order CurrentBelowZero,
// CurrentAboveMaximum, MaximumBelowZero, TemporaryBelowZero.
func (e *HitPointsError) Problems() []HitPointsProblem { return slices.Clone(e.problems) }

// Error renders the error. The text is not a contract.
func (e *HitPointsError) Error() string {
	names := make([]string, len(e.problems))
	for i, p := range e.problems {
		names[i] = p.String()
	}
	return "combat: invalid hit points: " + strings.Join(names, ", ")
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *HitPointsError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *HitPointsError listing the same problems.
func (e *HitPointsError) Is(target error) bool {
	t, ok := target.(*HitPointsError)
	return ok && t != nil && e != nil && slices.Equal(t.problems, e.problems)
}

// NewHitPoints returns the Hit Points with the given values.
//
// It returns a *HitPointsError (matching srderr.ErrInvalidInput) listing every
// violated problem when current < 0, current > maximum, maximum < 0 or
// temporary < 0. It never panics.
func NewHitPoints(current, maximum, temporary int) (HitPoints, error) {
	var problems []HitPointsProblem
	if current < 0 {
		problems = append(problems, CurrentBelowZero)
	}
	if current > maximum {
		problems = append(problems, CurrentAboveMaximum)
	}
	if maximum < 0 {
		problems = append(problems, MaximumBelowZero)
	}
	if temporary < 0 {
		problems = append(problems, TemporaryBelowZero)
	}
	if len(problems) > 0 {
		return HitPoints{}, &HitPointsError{problems: problems}
	}
	return HitPoints{current, maximum, temporary}, nil
}

// MustHitPoints is NewHitPoints for values known to be valid. It panics with
// the error NewHitPoints returns, and only then.
func MustHitPoints(current, maximum, temporary int) HitPoints {
	h, err := NewHitPoints(current, maximum, temporary)
	if err != nil {
		panic(err)
	}
	return h
}

// Current returns the current Hit Points.
func (h HitPoints) Current() int { return h.current }

// Maximum returns the Hit Point maximum.
func (h HitPoints) Maximum() int { return h.maximum }

// Temporary returns the Temporary Hit Points.
func (h HitPoints) Temporary() int { return h.temporary }
