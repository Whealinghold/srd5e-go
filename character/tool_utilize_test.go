package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: tool-proficiency-and-crafting.md FR-018 (SRD 5.2.1, "Tools",
// Utilize, p. 93) - the listed actions in printed order, as a fresh copy.
// Path: happy.
func TestUtilizeActions(t *testing.T) {
	t.Parallel()
	r := resolvedTool(thievesTools())
	got := character.UtilizeActions(r)
	if len(got) != 2 || got[0].Description() != "Pick a lock" || got[1].Description() != "Disarm a trap" {
		t.Fatalf("actions = %v", got)
	}
	got[0] = action("Changed", 1)
	if again := character.UtilizeActions(r); again[0].Description() != "Pick a lock" {
		t.Error("UtilizeActions returned the tool's own slice")
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-019, FR-020 (SRD
// 5.2.1, "Tools", Utilize, p. 93; Cook's Utensils, p. 93) - one DC per action,
// never a combined DC, and an unlisted action has none.
// Path: per case (happy; unhappy missing content).
func TestUtilizeDC(t *testing.T) {
	t.Parallel()
	cook := resolvedTool(standaloneTool("cooks_utensils", ability.Wisdom, action("Improve flavor", 10), action("Detect spoiled or poisoned food", 15)))
	thieves := resolvedTool(thievesTools())
	cases := []struct {
		name   string
		tool   character.ResolvedTool
		action content.ToolUtilizeAction
		wantDC int
	}{
		{"Cook's Utensils, improve flavor", cook, action("Improve flavor", 10), 10},
		{"Cook's Utensils, detect spoiled food", cook, action("Detect spoiled or poisoned food", 15), 15},
		{"Thieves' Tools, forge a key (not listed)", thieves, action("Forge a key", 15), 0},
		{"a listed description at another DC is not listed", thieves, action("Pick a lock", 16), 0},
		{"the zero tool lists nothing", character.ResolvedTool{}, action("Pick a lock", 15), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch got := character.UtilizeDC(c.tool, c.action).(type) {
			case character.ToolUtilizeDC:
				if int(got.DifficultyClass()) != c.wantDC || c.wantDC == 0 {
					t.Errorf("DC = %d, want %d", got.DifficultyClass(), c.wantDC)
				}
			case character.ToolUtilizeActionNotListed:
				if c.wantDC != 0 || got.Action() != c.action {
					t.Errorf("not listed %v, want DC %d", got.Action(), c.wantDC)
				}
			}
		})
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-021 (SRD 5.2.1,
// "Tools" and "Utilize [Action]", pp. 93, 191) - the check carries the Ability,
// the action's DC and the benefits; an unlisted action and a benefits mismatch
// are results, never silently scored.
// Path: per case (happy; unhappy missing content, contradictory data).
func TestUtilizeCheck(t *testing.T) {
	t.Parallel()
	herb, thieves := resolvedTool(herbalismKit()), resolvedTool(thievesTools())
	herbBenefits := character.EvaluateToolCheck(character.ToolCheckInput{Tool: herb, Tools: character.NewToolProficiencies("herbalism_kit")})
	thievesBenefits := character.EvaluateToolCheck(character.ToolCheckInput{Tool: thieves})
	t.Run("a listed action", func(t *testing.T) {
		t.Parallel()
		got, ok := character.UtilizeCheck(herb, action("Identify a plant", 10), herbBenefits).(character.ToolUtilizeCheck)
		if !ok {
			t.Fatalf("UtilizeCheck = %#v", character.UtilizeCheck(herb, action("Identify a plant", 10), herbBenefits))
		}
		if got.Ability() != ability.Intelligence || got.DifficultyClass() != d20.DifficultyClass(10) || got.Benefits() != herbBenefits {
			t.Errorf("check = %v, %v, %v", got.Ability(), got.DifficultyClass(), got.Benefits())
		}
	})
	t.Run("an unlisted action", func(t *testing.T) {
		t.Parallel()
		if _, ok := character.UtilizeCheck(thieves, action("Forge a key", 15), thievesBenefits).(character.ToolUtilizeActionNotListed); !ok {
			t.Error("want ToolUtilizeActionNotListed")
		}
	})
	t.Run("benefits evaluated for another tool", func(t *testing.T) {
		t.Parallel()
		got, ok := character.UtilizeCheck(herb, action("Identify a plant", 10), thievesBenefits).(character.ToolUtilizeBenefitsMismatch)
		if !ok || got.ToolID() != "herbalism_kit" || got.BenefitsToolID() != "thieves_tools" {
			t.Errorf("got %#v, want a mismatch between herbalism_kit and thieves_tools", got)
		}
	})
	t.Run("benefits evaluated for another tool, the other way round", func(t *testing.T) {
		t.Parallel()
		got, ok := character.UtilizeCheck(thieves, action("Pick a lock", 15), herbBenefits).(character.ToolUtilizeBenefitsMismatch)
		if !ok || got.ToolID() != "thieves_tools" || got.BenefitsToolID() != "herbalism_kit" {
			t.Errorf("got %#v, want a mismatch between thieves_tools and herbalism_kit", got)
		}
	})
	t.Run("the Ability follows the benefits", func(t *testing.T) {
		t.Parallel()
		b := character.EvaluateToolCheck(character.ToolCheckInput{Tool: thieves, Ability: ability.Strength})
		got, _ := character.UtilizeCheck(thieves, action("Pick a lock", 15), b).(character.ToolUtilizeCheck)
		if got.Ability() != ability.Strength {
			t.Errorf("Ability = %v, want Strength", got.Ability())
		}
	})
}
