package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

var (
	skillStealth = content.SkillStealth
	skillSleight = content.SkillSleightOfHand
	skillPerc    = content.SkillPerception
	skillAcro    = content.SkillAcrobatics
	skillArcana  = content.SkillArcana
)

// Requirement: doc/specs/class-rogue.md FR-023 to FR-030, FR-106 (SRD 5.2.1, "Level 1: Expertise", p. 61; "Expertise", p. 182) -
// Expertise is chosen once at Rogue level 1 and once at Rogue level 6, two distinct proficient skills each, and every problem is
// reported once in check order.
// Path: per case (the Expertise window cases and problem rows): happy / unhappy (boundary, state error, rule interaction, invalid input).
func TestRogueExpertiseSelect(t *testing.T) {
	t.Parallel()
	one, six := character.ExpertiseLevelOne, character.ExpertiseLevelSix
	prof := []content.Skill{skillStealth, skillSleight, skillPerc, skillAcro}
	pick := []content.Skill{skillStealth, skillSleight}
	P := func(ps ...character.RogueExpertiseProblem) []character.RogueExpertiseProblem { return ps }
	for _, c := range []struct {
		name     string
		level    int
		existing []content.Skill
		used     []character.RogueExpertiseWindow
		chosen   []content.Skill
		window   character.RogueExpertiseWindow
		problems []character.RogueExpertiseProblem
	}{
		{"level 1, none used", 1, nil, nil, pick, one, nil},
		{"level 1, any two proficient skills", 1, nil, nil, []content.Skill{skillPerc, skillAcro}, one, nil},
		{"level 6 after level 1", 6, pick, []character.RogueExpertiseWindow{one}, []content.Skill{skillPerc, skillAcro}, six, nil},
		{"level 1 again", 1, nil, []character.RogueExpertiseWindow{one}, pick, 0, P(character.RogueExpertiseAlready)},
		{"level 6 again", 6, pick, []character.RogueExpertiseWindow{one, six}, []content.Skill{skillPerc, skillAcro}, 0, P(character.RogueExpertiseAlready)},
		{"level 6 without level 1", 6, nil, nil, pick, 0, P(character.RogueExpertiseLevelOneNot)},
		{"level 2", 2, nil, []character.RogueExpertiseWindow{one}, pick, 0, P(character.RogueExpertiseNotDueAtThisLevel)},
		{"level 5", 5, nil, []character.RogueExpertiseWindow{one}, pick, 0, P(character.RogueExpertiseNotDueAtThisLevel)},
		{"level 7", 7, nil, []character.RogueExpertiseWindow{one, six}, pick, 0, P(character.RogueExpertiseNotDueAtThisLevel)},
		{"level 20", 20, nil, []character.RogueExpertiseWindow{one, six}, pick, 0, P(character.RogueExpertiseNotDueAtThisLevel)},
		{"zero skills", 1, nil, nil, nil, 0, P(character.RogueExpertiseWrongCount)},
		{"one skill", 1, nil, nil, []content.Skill{skillStealth}, 0, P(character.RogueExpertiseWrongCount)},
		{"three skills", 1, nil, nil, []content.Skill{skillStealth, skillSleight, skillPerc}, 0, P(character.RogueExpertiseWrongCount)},
		{"the same skill twice", 1, nil, nil, []content.Skill{skillStealth, skillStealth}, 0, P(character.RogueExpertiseDuplicate)},
		{"not proficient", 1, nil, nil, []content.Skill{skillStealth, skillArcana}, 0, P(character.RogueExpertiseNotProficient)},
		{"already has Expertise at level 6", 6, pick, []character.RogueExpertiseWindow{one}, []content.Skill{skillStealth, skillPerc}, 0, P(character.RogueExpertiseAlreadyHasExpertise)},
		{"several problems", 6, pick, []character.RogueExpertiseWindow{six}, []content.Skill{skillStealth, skillStealth, skillArcana}, 0, P(
			character.RogueExpertiseAlready, character.RogueExpertiseLevelOneNot, character.RogueExpertiseWrongCount, character.RogueExpertiseDuplicate, character.RogueExpertiseNotProficient, character.RogueExpertiseAlreadyHasExpertise)},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			chosen, used := slices.Clone(c.chosen), slices.Clone(c.used)
			sel, err := character.RogueExpertise{}.Select(rl(c.level), prof, c.existing, c.used, c.chosen)
			if !slices.Equal(chosen, c.chosen) || !slices.Equal(used, c.used) {
				t.Error("an argument changed")
			}
			if c.problems != nil {
				if got := rogProblems[character.RogueExpertiseProblem](t, err); !slices.Equal(got, c.problems) {
					t.Errorf("problems = %v, want %v", got, c.problems)
				}
				return
			}
			if err != nil || sel.Window() != c.window || !slices.Equal(sel.Skills(), c.chosen) {
				t.Errorf("selection = %v %v, %v; want %v %v", sel.Window(), sel.Skills(), err, c.window, c.chosen)
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-029 (SRD 5.2.1, "Rogue Features", p. 62; "Multiclassing", p. 24) - Expertise keys on the
// Rogue level, not the total level: a character with total level 9 and Rogue level 6 gets the second selection; total level 6 with
// Rogue level 3 does not.
// Path: unhappy (rule interaction).
func TestRogueExpertiseKeysOnRogueLevel(t *testing.T) {
	t.Parallel()
	prof := []content.Skill{skillStealth, skillSleight, skillPerc}
	used := []character.RogueExpertiseWindow{character.ExpertiseLevelOne}
	if _, err := (character.RogueExpertise{}).Select(rl(6), prof, nil, used, []content.Skill{skillPerc, skillStealth}); err != nil {
		t.Errorf("Rogue level 6 (total level 9) was refused: %v", err)
	}
	_, err := character.RogueExpertise{}.Select(rl(3), prof, nil, used, []content.Skill{skillPerc, skillStealth})
	if got := rogProblems[character.RogueExpertiseProblem](t, err); !slices.Equal(got, []character.RogueExpertiseProblem{character.RogueExpertiseNotDueAtThisLevel}) {
		t.Errorf("Rogue level 3 (total level 6) problems = %v", got)
	}
}

// Requirement: doc/specs/class-rogue.md FR-103 - Skills returns a copy of the selection and the windows render as snake_case.
// Path: unhappy (immutability).
func TestRogueExpertiseSelectionCopies(t *testing.T) {
	t.Parallel()
	sel := must(character.RogueExpertise{}.Select(rl(1), []content.Skill{skillStealth, skillSleight}, nil, nil, []content.Skill{skillStealth, skillSleight}))
	sel.Skills()[0] = skillArcana
	if sel.Skills()[0] != skillStealth {
		t.Error("Skills shares storage")
	}
	for w, want := range map[character.RogueExpertiseWindow]string{character.ExpertiseLevelOne: "level_one", character.ExpertiseLevelSix: "level_six", 0: "unknown"} {
		if w.String() != want {
			t.Errorf("window %d = %q, want %q", w, w.String(), want)
		}
	}
}
