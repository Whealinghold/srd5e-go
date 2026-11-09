package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// PointCostBudget is the number of points a Point Cost assignment may spend (SRD
// 5.2.1, "Point Cost", p. 21).
const PointCostBudget = 27

// StandardArrayScores returns the Standard Array as a new slice: 15, 14, 13, 12,
// 10 and 8 (SRD 5.2.1, "Standard Array", p. 21).
func StandardArrayScores() []int { return []int{15, 14, 13, 12, 10, 8} }

// PointCostOf returns the points a score costs under the Point Cost method:
// 0, 1, 2, 3, 4, 5, 7 and 9 for the scores 8 to 15 (SRD 5.2.1, Ability Score Point
// Costs table, p. 21). It returns an error matching srderr.ErrInvalidInput for a
// score below 8 or above 15. It never panics.
func PointCostOf(score int) (int, error) {
	switch {
	case score >= 8 && score <= 13:
		return score - 8, nil
	case score == 14:
		return 7, nil
	case score == 15:
		return 9, nil
	}
	return 0, invalidInput("point cost of a score outside 8 to 15")
}

// AbilityScoreAssignmentProblem is one reason an assignment of ability scores is
// refused (SRD 5.2.1, "Generate Your Scores", p. 21). Every value is an error
// matching srderr.ErrInvalidInput. The zero value is invalid.
//
//nolint:errname // a typed enum problem named by the spec
type AbilityScoreAssignmentProblem int

// The problems of NewAbilityScoreAssignment, in the order they are reported.
const (
	// AbilityMissing: an ability has no score.
	AbilityMissing AbilityScoreAssignmentProblem = iota + 1 //nolint:errname // a member of the typed enum, not a sentinel
	// NotStandardArray: the scores are not a permutation of the Standard Array.
	NotStandardArray //nolint:errname // a member of the typed enum, not a sentinel
	// ScoreOutsideRolledRange: a Random Generation score is not from 3 to 18.
	ScoreOutsideRolledRange //nolint:errname // a member of the typed enum, not a sentinel
	// ScoreOutsidePointCostRange: a Point Cost score is not from 8 to 15.
	ScoreOutsidePointCostRange //nolint:errname // a member of the typed enum, not a sentinel
	// PointCostExceedsBudget: the Point Cost total is above 27.
	PointCostExceedsBudget //nolint:errname // a member of the typed enum, not a sentinel
)

// String returns the snake_case name of p or "unknown".
func (p AbilityScoreAssignmentProblem) String() string {
	names := [...]string{"unknown", "ability_missing", "not_standard_array", "score_outside_rolled_range", "score_outside_point_cost_range", "point_cost_exceeds_budget"}
	if p < 1 || int(p) >= len(names) {
		return names[0]
	}
	return names[p]
}

// Error renders the problem. The text is not a contract.
func (p AbilityScoreAssignmentProblem) Error() string {
	return "character: invalid ability score assignment: " + p.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (AbilityScoreAssignmentProblem) Unwrap() error { return srderr.ErrInvalidInput }

// AbilityScoreAssignment is a method and the six scores assigned with it. It is
// an immutable value, comparable with ==. The zero value is invalid; see IsZero.
type AbilityScoreAssignment struct {
	method AbilityScoreGenerationMethod
	scores ability.Scores
}

// NewAbilityScoreAssignment returns the assignment of raw, the host's integers by
// ability, under method.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid method, and
// otherwise the errors.Join of every AbilityScoreAssignmentProblem found, each
// once and in declaration order. An ability without an entry is AbilityMissing;
// an integer below 1 or above 30 (not an ability.Score) is the problem of its
// method. The permutation check of the Standard Array and the budget check of
// Point Cost run only when every ability has a score. It never panics.
func NewAbilityScoreAssignment(method AbilityScoreGenerationMethod, raw map[ability.Ability]int) (AbilityScoreAssignment, error) {
	if !method.Valid() {
		return AbilityScoreAssignment{}, invalidInput("ability score assignment with an invalid method")
	}
	var found [6]bool
	add := func(p AbilityScoreAssignmentProblem) { found[p] = true }
	values := make([]int, 0, 6)
	missing := false
	for _, a := range ability.All() {
		v, ok := raw[a]
		if !ok {
			missing = true
			continue
		}
		values = append(values, v)
	}
	if missing {
		add(AbilityMissing)
	}
	switch method {
	case StandardArray:
		if !missing && !isStandardArray(values) {
			add(NotStandardArray)
		}
	case RandomGeneration:
		if slices.ContainsFunc(values, func(v int) bool { return v < 3 || v > 18 }) {
			add(ScoreOutsideRolledRange)
		}
	case PointCost:
		total, inRange := 0, true
		for _, v := range values {
			cost, err := PointCostOf(v)
			if err != nil {
				inRange = false
			}
			total += cost
		}
		if !inRange {
			add(ScoreOutsidePointCostRange)
		} else if !missing && total > PointCostBudget {
			add(PointCostExceedsBudget)
		}
	}
	var problems []error
	for p := AbilityMissing; p <= PointCostExceedsBudget; p++ {
		if found[p] {
			problems = append(problems, p)
		}
	}
	if len(problems) > 0 {
		return AbilityScoreAssignment{}, errors.Join(problems...)
	}
	m := map[ability.Ability]ability.Score{}
	for _, a := range ability.All() {
		s, err := ability.NewScore(raw[a])
		if err != nil { // unreachable: every method range lies inside 1 to 30
			return AbilityScoreAssignment{}, err
		}
		m[a] = s
	}
	scores, err := ability.NewScores(m)
	if err != nil { // unreachable: all six abilities are present
		return AbilityScoreAssignment{}, err
	}
	return AbilityScoreAssignment{method: method, scores: scores}, nil
}

// MustAbilityScoreAssignment is like NewAbilityScoreAssignment but panics with
// its error when it returns one.
func MustAbilityScoreAssignment(method AbilityScoreGenerationMethod, raw map[ability.Ability]int) AbilityScoreAssignment {
	a, err := NewAbilityScoreAssignment(method, raw)
	if err != nil {
		panic(err)
	}
	return a
}

// Method returns the method the scores were generated with.
func (a AbilityScoreAssignment) Method() AbilityScoreGenerationMethod { return a.method }

// Scores returns the assigned scores.
func (a AbilityScoreAssignment) Scores() ability.Scores { return a.scores }

// IsZero reports whether a is the invalid zero assignment.
func (a AbilityScoreAssignment) IsZero() bool { return a.method == 0 }

// isStandardArray reports whether values is a permutation of the Standard Array.
func isStandardArray(values []int) bool {
	got := slices.Clone(values)
	slices.Sort(got)
	want := StandardArrayScores()
	slices.Sort(want)
	return slices.Equal(got, want)
}

// RolledAbilityScore is one Random Generation roll: four d6 and the total of the
// highest three (SRD 5.2.1, "Random Generation", p. 21). It is an immutable value,
// comparable with ==. Only RollAbilityScores builds one; the zero value has no
// dice.
type RolledAbilityScore struct {
	dice  [4]int
	total int
}

// Dice returns a new slice of the four dice, in roll order.
func (s RolledAbilityScore) Dice() []int { return slices.Clone(s.dice[:]) }

// Total returns the sum of the highest three dice, 3 to 18.
func (s RolledAbilityScore) Total() int { return s.total }

// RolledAbilityScores are the six rolls of RollAbilityScores, in roll order. It is
// an immutable value, comparable with ==; the zero value holds no rolls.
type RolledAbilityScores struct{ scores [6]RolledAbilityScore }

// Scores returns a new slice of the six rolls, in roll order.
func (s RolledAbilityScores) Scores() []RolledAbilityScore { return slices.Clone(s.scores[:]) }

// Totals returns a new slice of the six totals, in roll order.
func (s RolledAbilityScores) Totals() []int {
	out := make([]int, len(s.scores))
	for i, r := range s.scores {
		out[i] = r.total
	}
	return out
}
