package scenarios_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-066, Table A rows A02, A15 - the host feeds the Extra Attack combination and the
// pact weapon count of the character context into the allowance, and the Attack action counts against them: a Fighter 5 / Barbarian 5
// (counts 2 and 2) makes two attacks, a Warlock with Thirsting Blade and Devouring Blade makes three with the pact weapon but one with
// any other.
// Path: happy.
func TestAttackActionCountsComeFromTheCharacterContext(t *testing.T) {
	t.Parallel()
	hero := combat.MustCombatantID("hero")
	general, err := character.ExtraAttackCombination([]int{2, 2})
	if err != nil {
		t.Fatal(err)
	}
	allowance, err := combat.NewAttackAllowance(general, character.PactWeaponAttackCount(general, false, false))
	if err != nil {
		t.Fatal(err)
	}
	progress, _ := combat.StartAttackActionProgress(hero, allowance)
	for range 2 {
		progress = progress.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRecorded).Progress()
	}
	if _, ok := progress.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused); !ok {
		t.Error("a third attack was recorded with an allowance of 2")
	}

	one, _ := character.ExtraAttackCombination([]int{1})
	warlock, err := combat.NewAttackAllowance(one, character.PactWeaponAttackCount(one, true, true))
	if err != nil {
		t.Fatal(err)
	}
	progress, _ = combat.StartAttackActionProgress(hero, warlock)
	for range 3 {
		out, ok := progress.RecordAttack(combat.AttackMeansPactWeapon).(combat.AttackActionAttackRecorded)
		if !ok {
			t.Fatal("a pact weapon attack was refused within 3")
		}
		progress = out.Progress()
	}
	other, _ := combat.StartAttackActionProgress(hero, warlock)
	other = other.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRecorded).Progress()
	if _, ok := other.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused); !ok {
		t.Error("a second non-pact attack was recorded with a general limit of 1")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-046, FR-050, FR-043 - Help gives Advantage as a circumstance only: a Help benefit
// is used up by the ally's next Athletics check, the caller's own Disadvantage cancels it in the kernel, and the benefit is gone.
// Path: unhappy (rule interaction).
func TestHelpAdvantageIsResolvedByTheKernel(t *testing.T) {
	t.Parallel()
	rogue, fighter := combat.MustCombatantID("rogue"), combat.MustCombatantID("fighter")
	athletics := combat.AssistedSkill(content.SkillAthletics)
	holds, err := combat.NewHelpGiverProficiencies([]content.Skill{content.SkillAthletics}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var benefits combat.HelpBenefits
	given := benefits.GiveAbilityCheckHelp(rogue, fighter, athletics, holds, combat.AssistanceJudgementPossible).(combat.AbilityCheckHelpGiven)
	used := given.Benefits().ConsumeForAbilityCheck(fighter, athletics).(combat.AbilityCheckHelpUsed)
	if mode := modifier.Resolve(used.Circumstances()...); mode != modifier.WithAdvantage {
		t.Errorf("mode = %v, want Advantage", mode)
	}
	if mode := modifier.Resolve(append(used.Circumstances(), modifier.Disadvantage{})...); mode != modifier.Normal {
		t.Errorf("mode with Disadvantage = %v, want Normal", mode)
	}
	if used.Benefits().Len() != 0 {
		t.Error("the benefit survived the check")
	}
	expiry := given.Benefits().ExpireAtStartOfTurn(rogue)
	if len(expiry.Events()) != 1 || expiry.Events()[0].Kind() != "help_benefit_expired" {
		t.Errorf("expiry events = %v", expiry.Events())
	}
}
