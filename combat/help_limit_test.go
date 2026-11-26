package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/attack-and-help-actions.md Limits (plan P10) - a restored benefit list holds at most 256 benefits.
// Path: unhappy (boundary).
func TestNewHelpBenefitsCap(t *testing.T) {
	t.Parallel()
	runCap(t, "help benefits", 256, func(n int) error {
		list := make([]combat.HelpBenefit, n)
		for i := range list {
			list[i] = rollHelp(t, fmt.Sprintf("h%d", i), "e")
		}
		_, err := combat.NewHelpBenefits(list)
		return err
	})
}

// Requirement: doc/specs/attack-and-help-actions.md Limits (plan P10) - a helper's skill list and tool list each hold at most 256 entries.
// Path: per case.
func TestNewHelpGiverProficienciesCap(t *testing.T) {
	t.Parallel()
	t.Run("skills", func(t *testing.T) {
		t.Parallel()
		runCap(t, "help skills", 256, func(n int) error {
			skills := make([]content.Skill, n)
			for i := range skills {
				skills[i] = content.SkillAthletics
			}
			_, err := combat.NewHelpGiverProficiencies(skills, nil)
			return err
		})
	})
	t.Run("tools", func(t *testing.T) {
		t.Parallel()
		runCap(t, "help tools", 256, func(n int) error {
			tools := make([]content.ItemID, n)
			for i := range tools {
				tools[i] = content.MustItemID("thieves_tools")
			}
			_, err := combat.NewHelpGiverProficiencies(nil, tools)
			return err
		})
	})
}

// Requirement: doc/specs/attack-and-help-actions.md Limits (plan P10) - an allied helper set holds at most 256 ids.
// Path: unhappy (boundary).
func TestNewAlliedHelpersCap(t *testing.T) {
	t.Parallel()
	runCap(t, "allied helpers", 256, func(n int) error {
		ids := make([]combat.CombatantID, n)
		for i := range ids {
			ids[i] = gid(fmt.Sprintf("a%d", i))
		}
		_, err := combat.NewAlliedHelpers(ids...)
		return err
	})
}

// Requirement: doc/specs/attack-and-help-actions.md Limits (plan P10) - a restored attack history holds at most 256 entries.
// Path: unhappy (boundary).
func TestNewAttackActionProgressHistoryCap(t *testing.T) {
	t.Parallel()
	runCap(t, "attack action history", 256, func(n int) error {
		history := make([]combat.AttackEntry, n)
		for i := range history {
			history[i] = combat.MadeAttack(combat.AttackMeansOtherWeapon)
		}
		_, err := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 256, 256), history, nil)
		return err
	})
}

// Requirement: doc/specs/attack-and-help-actions.md Limits (plan P10) - GiveAbilityCheckHelp and GiveAttackRollHelp refuse with the LimitExceeded reason, the shared limit_exceeded value, when 256 benefits are already pending, and leave the benefits unchanged; with 255 the help is given.
// Path: per case.
func TestGiveHelpCap(t *testing.T) {
	t.Parallel()
	held := func(n int) combat.HelpBenefits {
		list := make([]combat.HelpBenefit, n)
		for i := range list {
			list[i] = rollHelp(t, fmt.Sprintf("h%d", i), "e")
		}
		return benefitsOf(t, list...)
	}
	profs := proficienciesOf(t, []content.Skill{content.SkillAthletics}, nil)
	t.Run("ability check at the cap", func(t *testing.T) {
		t.Parallel()
		if _, ok := held(255).GiveAbilityCheckHelp(gid("a"), gid("b"), athletics(), profs, combat.AssistanceJudgementPossible).(combat.AbilityCheckHelpGiven); !ok {
			t.Error("255 pending: help not given")
		}
	})
	t.Run("ability check cap plus one", func(t *testing.T) {
		t.Parallel()
		h := held(256)
		o, ok := h.GiveAbilityCheckHelp(gid("a"), gid("b"), athletics(), profs, combat.AssistanceJudgementPossible).(combat.AbilityCheckHelpRefused)
		if !ok || o.Reason() != combat.AbilityCheckHelpRefusalLimitExceeded || o.Reason().String() != "limit_exceeded" || !o.Benefits().Equal(h) {
			t.Errorf("outcome = %v, want a limit_exceeded refusal with the benefits unchanged", o)
		}
	})
	t.Run("attack roll at the cap", func(t *testing.T) {
		t.Parallel()
		if _, ok := held(255).GiveAttackRollHelp(gid("a"), gid("z"), combat.AttackAssistTargetEnemyWithinFiveFeet).(combat.AttackRollHelpGiven); !ok {
			t.Error("255 pending: help not given")
		}
	})
	t.Run("attack roll cap plus one", func(t *testing.T) {
		t.Parallel()
		h := held(256)
		o, ok := h.GiveAttackRollHelp(gid("a"), gid("z"), combat.AttackAssistTargetEnemyWithinFiveFeet).(combat.AttackRollHelpRefused)
		if !ok || o.Reason() != combat.AttackRollHelpRefusalLimitExceeded || o.Reason().String() != "limit_exceeded" || !o.Benefits().Equal(h) {
			t.Errorf("outcome = %v, want a limit_exceeded refusal with the benefits unchanged", o)
		}
	})
}

// Requirement: doc/specs/attack-and-help-actions.md Limits (plan P10) - an attack allowance holds at most 256 general attacks and at most 256 pact weapon attacks; 257 of either is a LimitExceededError.
// Path: unhappy (boundary).
func TestNewAttackAllowanceCaps(t *testing.T) {
	t.Parallel()
	t.Run("general attacks", func(t *testing.T) {
		t.Parallel()
		runCap(t, "general attacks", 256, func(n int) error {
			_, err := combat.NewAttackAllowance(n, n)
			return err
		})
	})
	t.Run("pact weapon attacks", func(t *testing.T) {
		t.Parallel()
		runCap(t, "pact weapon attacks", 256, func(n int) error {
			_, err := combat.NewAttackAllowance(1, n)
			return err
		})
	})
}
