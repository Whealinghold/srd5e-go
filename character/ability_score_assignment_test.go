package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func rawScores(str, dex, con, intl, wis, cha int) map[ability.Ability]int {
	return map[ability.Ability]int{ability.Strength: str, ability.Dexterity: dex, ability.Constitution: con, ability.Intelligence: intl, ability.Wisdom: wis, ability.Charisma: cha}
}

// Requirement: doc/specs/character-creation.md FR-055 - the Standard Array is 15, 14, 13, 12, 10 and 8, returned as a fresh slice.
// Path: happy.
func TestStandardArrayScores(t *testing.T) {
	t.Parallel()
	got := character.StandardArrayScores()
	if !slices.Equal(got, []int{15, 14, 13, 12, 10, 8}) {
		t.Errorf("StandardArrayScores() = %v", got)
	}
	got[0] = 0
	if character.StandardArrayScores()[0] != 15 {
		t.Error("StandardArrayScores shares its slice")
	}
}

// Requirement: doc/specs/character-creation.md FR-060 and the table "Point Cost cases" - the cost of 8 to 15 is 0, 1, 2, 3, 4, 5, 7 and 9; 7 and 16 are validation errors.
// Path: per case.
func TestPointCostOf(t *testing.T) {
	t.Parallel()
	for score, want := range map[int]int{8: 0, 9: 1, 10: 2, 11: 3, 12: 4, 13: 5, 14: 7, 15: 9} {
		if got, err := character.PointCostOf(score); err != nil || got != want {
			t.Errorf("PointCostOf(%d) = %d, %v, want %d", score, got, err, want)
		}
	}
	for _, score := range []int{7, 16, 0, -1} {
		if _, err := character.PointCostOf(score); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("PointCostOf(%d) error = %v", score, err)
		}
	}
	if character.PointCostBudget != 27 {
		t.Errorf("PointCostBudget = %d", character.PointCostBudget)
	}
}

// Requirement: per case (doc/specs/character-creation.md FR-056, FR-059, FR-061, FR-062, FR-066 and the tables "Standard Array cases",
// "Random Generation cases", "Point Cost cases" and "Raw integer cases").
// Path: per case.
func TestNewAbilityScoreAssignment(t *testing.T) {
	t.Parallel()
	missing := rawScores(15, 14, 13, 12, 10, 8)
	delete(missing, ability.Charisma)
	std, rnd, pc := character.StandardArray, character.RandomGeneration, character.PointCost
	for _, c := range []struct {
		name, requirement, path string
		method                  character.AbilityScoreGenerationMethod
		raw                     map[ability.Ability]int
		want                    []character.AbilityScoreAssignmentProblem
	}{
		{"the Fighter row", "p. 21", "happy", std, rawScores(15, 14, 13, 8, 10, 12), nil},
		{"the Wizard row", "p. 21", "happy", std, rawScores(8, 12, 13, 15, 14, 10), nil},
		{"the Barbarian row", "p. 21", "happy", std, rawScores(15, 13, 14, 10, 12, 8), nil},
		{"a score twice", "p. 21", "unhappy (invalid input)", std, rawScores(15, 15, 13, 12, 10, 8), []character.AbilityScoreAssignmentProblem{character.NotStandardArray}},
		{"same total 72", "p. 21", "unhappy (boundary)", std, rawScores(14, 14, 13, 12, 10, 9), []character.AbilityScoreAssignmentProblem{character.NotStandardArray}},
		{"off by one", "p. 21", "unhappy (boundary)", std, rawScores(15, 14, 13, 12, 10, 7), []character.AbilityScoreAssignmentProblem{character.NotStandardArray}},
		{"all equal", "p. 21", "unhappy (boundary)", std, rawScores(10, 10, 10, 10, 10, 10), []character.AbilityScoreAssignmentProblem{character.NotStandardArray}},
		{"five scores", "p. 21", "unhappy (empty and minimal)", std, missing, []character.AbilityScoreAssignmentProblem{character.AbilityMissing}},
		{"no scores", "p. 21", "unhappy (empty and minimal)", std, nil, []character.AbilityScoreAssignmentProblem{character.AbilityMissing}},
		{"raw 0 under the Standard Array", "p. 5", "unhappy (boundary)", std, rawScores(0, 14, 13, 12, 10, 8), []character.AbilityScoreAssignmentProblem{character.NotStandardArray}},
		{"raw 31 under the Standard Array", "p. 5", "unhappy (boundary)", std, rawScores(31, 14, 13, 12, 10, 8), []character.AbilityScoreAssignmentProblem{character.NotStandardArray}},
		{"rolled 3 and 18", "p. 21", "unhappy (boundary)", rnd, rawScores(3, 18, 10, 10, 10, 10), nil},
		{"rolled 2", "p. 21", "unhappy (boundary)", rnd, rawScores(2, 10, 10, 10, 10, 10), []character.AbilityScoreAssignmentProblem{character.ScoreOutsideRolledRange}},
		{"rolled 19", "p. 21", "unhappy (boundary)", rnd, rawScores(19, 10, 10, 10, 10, 10), []character.AbilityScoreAssignmentProblem{character.ScoreOutsideRolledRange}},
		{"rolled 30 is a Score outside the range", "p. 5", "unhappy (boundary)", rnd, rawScores(30, 10, 10, 10, 10, 10), []character.AbilityScoreAssignmentProblem{character.ScoreOutsideRolledRange}},
		{"rolled 0", "p. 5", "unhappy (boundary)", rnd, rawScores(0, 10, 10, 10, 10, 10), []character.AbilityScoreAssignmentProblem{character.ScoreOutsideRolledRange}},
		{"rolled with a missing ability and a bad score", "p. 21", "unhappy (invalid input)", rnd, map[ability.Ability]int{ability.Strength: 40}, []character.AbilityScoreAssignmentProblem{character.AbilityMissing, character.ScoreOutsideRolledRange}},
		{"all eights", "p. 21", "happy (boundary)", pc, rawScores(8, 8, 8, 8, 8, 8), nil},
		{"three fifteens", "p. 21", "happy (boundary)", pc, rawScores(15, 15, 15, 8, 8, 8), nil},
		{"balanced", "p. 21", "happy (boundary)", pc, rawScores(14, 14, 14, 10, 10, 10), nil},
		{"the Standard Array as Point Cost", "p. 21", "unhappy (boundary)", pc, rawScores(15, 14, 13, 12, 10, 8), nil},
		{"cost 28", "p. 21", "unhappy (boundary)", pc, rawScores(15, 15, 15, 8, 8, 9), []character.AbilityScoreAssignmentProblem{character.PointCostExceedsBudget}},
		{"cost 30", "p. 21", "unhappy (boundary)", pc, rawScores(13, 13, 13, 13, 13, 13), []character.AbilityScoreAssignmentProblem{character.PointCostExceedsBudget}},
		{"cost 54", "p. 21", "unhappy (boundary)", pc, rawScores(15, 15, 15, 15, 15, 15), []character.AbilityScoreAssignmentProblem{character.PointCostExceedsBudget}},
		{"point cost 7", "p. 21", "unhappy (boundary)", pc, rawScores(7, 8, 8, 8, 8, 8), []character.AbilityScoreAssignmentProblem{character.ScoreOutsidePointCostRange}},
		{"point cost 16", "p. 21", "unhappy (boundary)", pc, rawScores(16, 8, 8, 8, 8, 8), []character.AbilityScoreAssignmentProblem{character.ScoreOutsidePointCostRange}},
		{"out of range and over budget: the range only", "p. 21", "unhappy (rule interaction)", pc, rawScores(16, 15, 15, 15, 8, 8), []character.AbilityScoreAssignmentProblem{character.ScoreOutsidePointCostRange}},
		{"raw 0 under Point Cost", "p. 5", "unhappy (boundary)", pc, rawScores(0, 8, 8, 8, 8, 8), []character.AbilityScoreAssignmentProblem{character.ScoreOutsidePointCostRange}},
		{"raw 31 under Point Cost", "p. 5", "unhappy (boundary)", pc, rawScores(31, 8, 8, 8, 8, 8), []character.AbilityScoreAssignmentProblem{character.ScoreOutsidePointCostRange}},
		{"point cost with a missing ability", "p. 21", "unhappy (rule interaction)", pc, missing, []character.AbilityScoreAssignmentProblem{character.AbilityMissing}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a, err := character.NewAbilityScoreAssignment(c.method, c.raw)
			if len(c.want) == 0 {
				if err != nil || a.IsZero() || a.Method() != c.method || a.Scores().Score(ability.Strength).Value() != c.raw[ability.Strength] {
					t.Errorf("= %+v, %v", a, err)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !a.IsZero() {
				t.Fatalf("= %+v, %v", a, err)
			}
			var got []character.AbilityScoreAssignmentProblem
			for p := character.AbilityMissing; p <= character.PointCostExceedsBudget; p++ {
				if errors.Is(err, p) {
					got = append(got, p)
				}
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-062, FR-063 - an invalid method is an invalid-input error, MustAbilityScoreAssignment
// panics with the error if and only if the constructor returns one, and the problems have distinct names.
// Path: unhappy (invalid input).
func TestMustAbilityScoreAssignment(t *testing.T) {
	t.Parallel()
	if _, err := character.NewAbilityScoreAssignment(0, rawScores(15, 14, 13, 12, 10, 8)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero method error = %v", err)
	}
	if got := character.MustAbilityScoreAssignment(character.PointCost, rawScores(8, 8, 8, 8, 8, 8)); got.IsZero() {
		t.Error("MustAbilityScoreAssignment returned the zero value")
	}
	func() {
		defer func() {
			err, _ := recover().(error)
			if !errors.Is(err, character.NotStandardArray) {
				t.Errorf("panic = %v", err)
			}
		}()
		character.MustAbilityScoreAssignment(character.StandardArray, rawScores(10, 10, 10, 10, 10, 10))
	}()
	names := map[string]bool{}
	for p := character.AbilityMissing; p <= character.PointCostExceedsBudget; p++ {
		names[p.String()] = true
		if p.Error() == "" || !errors.Is(p, srderr.ErrInvalidInput) {
			t.Errorf("problem %d: %q", p, p.Error())
		}
	}
	if len(names) != 5 || character.AbilityScoreAssignmentProblem(0).String() != "unknown" || character.AbilityScoreAssignmentProblem(9).String() != "unknown" {
		t.Errorf("names = %v", names)
	}
}

// Requirement: doc/specs/character-creation.md FR-063 - NewAbilityScoreAssignment never panics for any map of integers, and returns an
// assignment only when the scores fit the method.
// Path: unhappy (invalid input).
func FuzzNewAbilityScoreAssignment(f *testing.F) {
	f.Add(1, 15, 14, 13, 12, 10, 8)
	f.Add(2, 3, 18, 0, -1, 31, 1<<62)
	f.Add(3, 8, 8, 8, 8, 8, 8)
	f.Add(0, 0, 0, 0, 0, 0, 0)
	f.Fuzz(func(t *testing.T, method, a, b, c, d, e, g int) {
		raw := rawScores(a, b, c, d, e, g)
		if method%2 == 0 {
			delete(raw, ability.Wisdom)
		}
		got, err := character.NewAbilityScoreAssignment(character.AbilityScoreGenerationMethod(method), raw)
		if err == nil && (got.IsZero() || !got.Method().Valid()) {
			t.Fatalf("no error but assignment %+v", got)
		}
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("error %v does not match ErrInvalidInput", err)
		}
	})
}

// Requirement: doc/specs/character-creation.md section 3 ("AbilityScoreAssignmentProblem") - each problem has its own snake_case name, the
// zero value and values past the last are "unknown", and each is an error matching ErrInvalidInput.
// Path: per case.
func TestAbilityScoreAssignmentProblemNames(t *testing.T) {
	t.Parallel()
	for p, want := range map[character.AbilityScoreAssignmentProblem]string{
		character.AbilityMissing: "ability_missing", character.NotStandardArray: "not_standard_array", character.ScoreOutsideRolledRange: "score_outside_rolled_range",
		character.ScoreOutsidePointCostRange: "score_outside_point_cost_range", character.PointCostExceedsBudget: "point_cost_exceeds_budget",
		0: "unknown", 6: "unknown", -1: "unknown",
	} {
		if p.String() != want {
			t.Errorf("AbilityScoreAssignmentProblem(%d).String() = %q, want %q", p, p.String(), want)
		}
	}
	if character.AbilityMissing != 1 || character.PointCostExceedsBudget != 5 {
		t.Errorf("the problems are not numbered 1 to 5 in declaration order: %d, %d", character.AbilityMissing, character.PointCostExceedsBudget)
	}
}
