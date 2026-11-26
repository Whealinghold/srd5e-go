package combat

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

func mustPlan(t *testing.T) content.MultiattackPlan {
	t.Helper()
	must := func(v any, err error) any {
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	rule := must(content.NewReplacementRule(1, content.ReplacementScopeAnyAttack{}, []content.ReplacementOption{
		must(content.NewReplacementOption("Breath", "Scorching Ray", 3)).(content.ReplacementOption)}, content.ReplaceOnce)).(content.ReplacementRule)
	variant := must(content.NewMultiattackVariant("Hasten", []content.AttackSlotGroup{must(content.NewAttackSlotGroup(2, []string{"Claw", "Bite"})).(content.AttackSlotGroup)},
		[]content.AdditionalUse{must(content.NewAdditionalUse([]string{"Gaze"})).(content.AdditionalUse)}, []content.ReplacementRule{rule})).(content.MultiattackVariant)
	plain := must(content.NewMultiattackVariant("", []content.AttackSlotGroup{must(content.NewAttackSlotGroup(1, []string{"Claw"})).(content.AttackSlotGroup)}, nil, nil)).(content.MultiattackVariant)
	return must(content.NewMultiattackPlan([]content.MultiattackVariant{variant, plain})).(content.MultiattackPlan)
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-021 - a name that resolves to no part is MultiattackPlanPartMissing and one that resolves to two is
// MultiattackPlanPartShared, for the attack names and for the part that must have been used, and a spell the Spellcasting lacks is MultiattackPlanSpellMissing; each name is
// reported once, in document order. Content rejects the first two kinds for a monster it builds, so this internal test reaches them with injected lookups.
// Path: per case.
func TestPlanProblemsAgainstInjectedLookups(t *testing.T) {
	t.Parallel()
	plan := mustPlan(t)
	counts := func(m map[string]int) func(string) int { return func(n string) int { return m[n] } }
	cases := []struct {
		name         string
		actions      map[string]int
		used         map[string]int
		spellPresent bool
		want         []MultiattackPlanProblem
	}{
		{"everything resolves", map[string]int{"Claw": 1, "Bite": 1, "Gaze": 1, "Breath": 1}, map[string]int{"Hasten": 1}, true, nil},
		{"a missing attack part", map[string]int{"Claw": 1, "Gaze": 1, "Breath": 1}, map[string]int{"Hasten": 1}, true, []MultiattackPlanProblem{{MultiattackPlanPartMissing, "Bite"}}},
		{"a shared attack part, reported once for its two uses", map[string]int{"Claw": 2, "Bite": 1, "Gaze": 1, "Breath": 1}, map[string]int{"Hasten": 1}, true, []MultiattackPlanProblem{{MultiattackPlanPartShared, "Claw"}}},
		{"a missing required part", map[string]int{"Claw": 1, "Bite": 1, "Gaze": 1, "Breath": 1}, nil, true, []MultiattackPlanProblem{{MultiattackPlanPartMissing, "Hasten"}}},
		{"a shared required part", map[string]int{"Claw": 1, "Bite": 1, "Gaze": 1, "Breath": 1}, map[string]int{"Hasten": 2}, true, []MultiattackPlanProblem{{MultiattackPlanPartShared, "Hasten"}}},
		{"a missing spell", map[string]int{"Claw": 1, "Bite": 1, "Gaze": 1, "Breath": 1}, map[string]int{"Hasten": 1}, false, []MultiattackPlanProblem{{MultiattackPlanSpellMissing, "Scorching Ray"}}},
		{"all kinds in document order", nil, nil, false, []MultiattackPlanProblem{{MultiattackPlanPartMissing, "Claw"}, {MultiattackPlanPartMissing, "Bite"}, {MultiattackPlanPartMissing, "Gaze"},
			{MultiattackPlanPartMissing, "Breath"}, {MultiattackPlanPartMissing, "Hasten"}, {MultiattackPlanSpellMissing, "Scorching Ray"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := planProblems(plan, planLookups{actions: counts(c.actions), used: counts(c.used), spell: func(string) bool { return c.spellPresent }})
			if !slices.Equal(got, c.want) {
				t.Errorf("planProblems = %v, want %v", got, c.want)
			}
		})
	}
}
