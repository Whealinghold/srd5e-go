package character_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var allMethods = []character.AbilityScoreGenerationMethod{character.StandardArray, character.RandomGeneration, character.PointCost}

// Requirement: doc/specs/character-aggregate.md FR-104, FR-116 - the standard
// options permit all three methods, start at level 1 and use NoBonusFeats, and
// equal an explicit instance.
// Path: happy.
func TestStandardRuleOptions(t *testing.T) {
	t.Parallel()
	s := character.StandardRuleOptions()
	explicit := character.MustRuleOptions(allMethods, 1, character.NoBonusFeats{})
	if !s.Equal(explicit) || s.StartingLevel() != 1 || !slices.Equal(s.PermittedMethods(), allMethods) || s.BonusFeatsAtLevel20() != (character.NoBonusFeats{}) || s.IsZero() {
		t.Errorf("standard = %+v", s)
	}
	if !(character.RuleOptions{}).IsZero() {
		t.Error("zero options must report IsZero")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-105, FR-111 - every non-empty
// subset of the methods round-trips; the exposed set is a copy; an empty set is
// rejected.
// Path: per case.
func TestPermittedMethodSubsets(t *testing.T) {
	t.Parallel()
	for mask := 1; mask < 8; mask++ {
		var want []character.AbilityScoreGenerationMethod
		for i, m := range allMethods {
			if mask&(1<<i) != 0 {
				want = append(want, m)
			}
		}
		in := slices.Clone(want)
		o, err := character.NewRuleOptions(in, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		in[0] = character.PointCost // later mutation of the input must not leak
		got := o.PermittedMethods()
		if !slices.Equal(got, want) {
			t.Errorf("subset %v round-trips to %v", want, got)
		}
		got[0] = 0
		if !slices.Equal(o.PermittedMethods(), want) {
			t.Error("PermittedMethods must return a copy")
		}
	}
	_, err := character.NewRuleOptions(nil, 1, nil)
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("empty set: %v", err)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-106, FR-107, FR-108, FR-109,
// FR-110, FR-115, FR-117 - the starting level accepts 1 to 20 and rejects the rest
// with a typed problem carrying parameter, kind and value; every problem is
// reported, methods first; Must panics with the same error.
// Path: per case.
func TestRuleOptionsValidation(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		want  []character.RuleOptionsProblem
	}{
		{1, nil}, {2, nil}, {19, nil}, {20, nil},
		{0, []character.RuleOptionsProblem{{Parameter: character.StartingLevel, Kind: character.RuleOptionsBelowMinimum, Value: 0}}},
		{-1, []character.RuleOptionsProblem{{Parameter: character.StartingLevel, Kind: character.RuleOptionsBelowMinimum, Value: -1}}},
		{math.MinInt, []character.RuleOptionsProblem{{Parameter: character.StartingLevel, Kind: character.RuleOptionsBelowMinimum, Value: math.MinInt}}},
		{21, []character.RuleOptionsProblem{{Parameter: character.StartingLevel, Kind: character.RuleOptionsAboveMaximum, Value: 21}}},
		{math.MaxInt, []character.RuleOptionsProblem{{Parameter: character.StartingLevel, Kind: character.RuleOptionsAboveMaximum, Value: math.MaxInt}}},
	} {
		o, err := character.NewRuleOptions(allMethods, c.level, nil)
		_, levelErr := character.NewLevel(c.level)
		if (err != nil) != (levelErr != nil) {
			t.Errorf("level %d: options error %v disagrees with NewLevel %v", c.level, err, levelErr)
		}
		if c.want == nil {
			if err != nil || o.StartingLevel() != c.level {
				t.Errorf("level %d: %v", c.level, err)
			}
			continue
		}
		got := ruleOptionsProblems(err)
		if !slices.Equal(got, c.want) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("level %d: problems %v, want %v", c.level, got, c.want)
		}
	}
	// Both parameters bad: methods before level, every problem reported.
	_, err := character.NewRuleOptions(nil, 0, nil)
	got := ruleOptionsProblems(err)
	if len(got) != 2 || got[0].Parameter != character.AbilityScoreMethods || got[0].Kind != character.RuleOptionsEmpty || got[1].Parameter != character.StartingLevel {
		t.Errorf("problems = %v", got)
	}
	panicked := recovered(t, func() { character.MustRuleOptions(nil, 0, nil) })
	var first character.RuleOptionsProblem
	if panicked == nil || !errors.As(panicked, &first) || first.Parameter != character.AbilityScoreMethods {
		t.Errorf("Must panic = %v", panicked)
	}
	// An unknown method is a problem of its own.
	_, err = character.NewRuleOptions([]character.AbilityScoreGenerationMethod{0, character.PointCost}, 1, nil)
	if p := ruleOptionsProblems(err); len(p) != 1 || p[0].Kind != character.RuleOptionsUnknownMethod {
		t.Errorf("unknown method: %v", err)
	}
	if recovered(t, func() { character.MustRuleOptions(allMethods, 5, nil) }) != nil {
		t.Error("Must panicked on valid input")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-112 - Equal ignores method
// order; the With methods change one field, validate and leave the original.
// Path: per case.
func TestRuleOptionsEqualAndWith(t *testing.T) {
	t.Parallel()
	a := character.MustRuleOptions([]character.AbilityScoreGenerationMethod{character.PointCost, character.StandardArray}, 3, character.BonusFeatsFromXp{})
	b := character.MustRuleOptions([]character.AbilityScoreGenerationMethod{character.StandardArray, character.PointCost}, 3, character.BonusFeatsFromXp{})
	if !a.Equal(b) {
		t.Error("equal regardless of insertion order")
	}
	std := character.StandardRuleOptions()
	m, err := std.WithPermittedMethods([]character.AbilityScoreGenerationMethod{character.RandomGeneration})
	if err != nil || !slices.Equal(m.PermittedMethods(), []character.AbilityScoreGenerationMethod{character.RandomGeneration}) || m.StartingLevel() != 1 || m.Equal(std) {
		t.Errorf("WithPermittedMethods: %v %v", m, err)
	}
	l, err := std.WithStartingLevel(5)
	if err != nil || l.StartingLevel() != 5 || !slices.Equal(l.PermittedMethods(), allMethods) {
		t.Errorf("WithStartingLevel: %v", err)
	}
	f, err := std.WithBonusFeatsAtLevel20(character.BonusFeatsFromXp{})
	if err != nil || f.BonusFeatsAtLevel20() != (character.BonusFeatsFromXp{}) || f.Equal(std) {
		t.Errorf("WithBonusFeatsAtLevel20: %v", err)
	}
	if _, err := std.WithPermittedMethods(nil); err == nil {
		t.Error("empty set must be rejected")
	}
	if _, err := std.WithStartingLevel(0); err == nil {
		t.Error("level 0 must be rejected")
	}
	if !std.Equal(character.MustRuleOptions(allMethods, 1, nil)) || std.StartingLevel() != 1 {
		t.Error("the original must be unchanged")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-089 - the generation methods,
// parameters and problem kinds print their names.
// Path: happy.
func TestRuleOptionsStrings(t *testing.T) {
	t.Parallel()
	got := []string{
		character.StandardArray.String(), character.RandomGeneration.String(), character.PointCost.String(), character.AbilityScoreGenerationMethod(0).String(),
		character.AbilityScoreMethods.String(), character.StartingLevel.String(), character.RuleOptionsParameter(0).String(),
		character.RuleOptionsEmpty.String(), character.RuleOptionsBelowMinimum.String(), character.RuleOptionsAboveMaximum.String(), character.RuleOptionsUnknownMethod.String(), character.RuleOptionsProblemKind(0).String(),
	}
	want := []string{"standardArray", "randomGeneration", "pointCost", "unknown", "abilityScoreMethods", "startingLevel", "unknown", "empty", "belowMinimum", "aboveMaximum", "unknownMethod", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("strings = %v", got)
	}
	if (character.RuleOptionsProblem{Parameter: character.StartingLevel, Kind: character.RuleOptionsEmpty}).Error() == "" {
		t.Error("empty problem text")
	}
}

// problemsOf returns the RuleOptionsProblem values a joined error holds.
func ruleOptionsProblems(err error) []character.RuleOptionsProblem {
	var out []character.RuleOptionsProblem
	for _, e := range joined(err) {
		var p character.RuleOptionsProblem
		if errors.As(e, &p) {
			out = append(out, p)
		}
	}
	return out
}
