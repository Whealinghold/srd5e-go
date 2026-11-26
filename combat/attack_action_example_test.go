package combat_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A Fighter 5 / Barbarian 5 has two attacks per Attack action; the host passes the count
// in, draws a weapon, attacks twice and is between attacks once.
func ExampleAttackActionProgress() {
	allowance, _ := combat.NewAttackAllowance(2, 2)
	progress, _ := combat.StartAttackActionProgress(combat.MustCombatantID("hero"), allowance)
	if out, ok := progress.HandleWeapon(combat.WeaponChangeDrawFromSheath, combat.WeaponChangeBeforeAttack).(combat.AttackActionWeaponChangeRecorded); ok {
		progress = out.Progress()
	}
	out := progress.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRecorded)
	progress = out.Progress()
	fmt.Println(out.Events()[0].Kind(), progress.IsBetweenAttacks())
	progress = progress.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRecorded).Progress()
	fmt.Println(progress.IsBetweenAttacks(), progress.AttacksRemaining())
	_, refused := progress.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused)
	fmt.Println(refused)
	// Output:
	// attack_made_in_attack_action true
	// false 0
	// true
}

// A helper gives an ally Advantage on the next Athletics check; the check spends it.
func ExampleHelpBenefits() {
	athletics := combat.AssistedSkill(content.SkillAthletics)
	holds, _ := combat.NewHelpGiverProficiencies([]content.Skill{content.SkillAthletics}, nil)
	var benefits combat.HelpBenefits
	given := benefits.GiveAbilityCheckHelp(combat.MustCombatantID("rogue"), combat.MustCombatantID("fighter"),
		athletics, holds, combat.AssistanceJudgementPossible).(combat.AbilityCheckHelpGiven)
	used := given.Benefits().ConsumeForAbilityCheck(combat.MustCombatantID("fighter"), athletics).(combat.AbilityCheckHelpUsed)
	fmt.Println(modifier.Resolve(used.Circumstances()...), used.Benefits().Len())
	// Output:
	// advantage 0
}
