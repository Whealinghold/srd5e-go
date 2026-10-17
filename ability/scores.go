package ability

import (
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Scores holds one Score for each of the six abilities. It is immutable and
// comparable. The zero Scores is invalid; see IsZero.
type Scores struct{ held [6]Score }

// NewScores builds Scores from m. It returns an errors.Join of one
// *InvalidError per problem, all reported at once: ReasonMissingScore for each
// ability with no (or a zero) score and ReasonUnknownAbility for each invalid
// key.
func NewScores(m map[Ability]Score) (Scores, error) {
	var s Scores
	var errs []error
	for _, a := range All() {
		sc := m[a]
		if sc.IsZero() {
			errs = append(errs, invalid(ReasonMissingScore, "no score for "+a.String()))
			continue
		}
		s.held[a-Strength] = sc
	}
	// Sorted so the joined error is deterministic.
	for _, a := range slices.Sorted(maps.Keys(m)) {
		if !a.Valid() {
			errs = append(errs, invalid(ReasonUnknownAbility, "invalid ability key "+strconv.Itoa(int(a))))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return Scores{}, err
	}
	return s, nil
}

// IsZero reports whether s is the invalid zero Scores.
func (s Scores) IsZero() bool { return s == Scores{} }

// Score returns the score of a. It returns the zero Score when a is invalid or
// s is the zero Scores.
func (s Scores) Score(a Ability) Score {
	if !a.Valid() {
		return Score{}
	}
	return s.held[a-Strength]
}

// Modifier returns the modifier of a's score.
func (s Scores) Modifier(a Ability) Modifier { return s.Score(a).Modifier() }

// With returns a copy of s with the score of a replaced. It returns an
// *InvalidError with ReasonUnknownAbility when a is invalid and
// ReasonMissingScore when sc is the zero Score.
func (s Scores) With(a Ability, sc Score) (Scores, error) {
	if !a.Valid() {
		return Scores{}, invalid(ReasonUnknownAbility, "invalid ability")
	}
	if sc.IsZero() {
		return Scores{}, invalid(ReasonMissingScore, "zero score for "+a.String())
	}
	s.held[a-Strength] = sc
	return s, nil
}

// String renders each ability and score in SRD order.
func (s Scores) String() string {
	parts := make([]string, 0, len(s.held))
	for _, a := range All() {
		parts = append(parts, a.String()+": "+s.Score(a).String())
	}
	return "Scores(" + strings.Join(parts, ", ") + ")"
}
