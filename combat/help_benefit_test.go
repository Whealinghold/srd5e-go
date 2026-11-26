package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-074 - NewAbilityCheckHelp rejects a helper equal to the ally, a blank id and an invalid proficiency; NewAttackRollHelp rejects a helper equal to the enemy and a blank id; distinct combatants build and report [helper, other].
// Path: per case.
func TestNewHelpBenefitsValidateTheirCombatants(t *testing.T) {
	t.Parallel()
	for name, err := range map[string]error{
		"check helper is ally":  second(combat.NewAbilityCheckHelp(gid("h"), gid("h"), athletics())),
		"check blank helper":    second(combat.NewAbilityCheckHelp("", gid("y"), athletics())),
		"check blank ally":      second(combat.NewAbilityCheckHelp(gid("h"), "", athletics())),
		"check bad proficiency": second(combat.NewAbilityCheckHelp(gid("h"), gid("y"), combat.AssistedProficiency{})),
		"roll helper is enemy":  second(combat.NewAttackRollHelp(gid("h"), gid("h"))),
		"roll blank helper":     second(combat.NewAttackRollHelp("", gid("e"))),
		"roll blank enemy":      second(combat.NewAttackRollHelp(gid("h"), "")),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	c, r := checkHelp(t, "h", "y", thievesTools()), rollHelp(t, "h", "e")
	if got := c.Combatants(); len(got) != 2 || got[0] != gid("h") || got[1] != gid("y") || c.Proficiency() != thievesTools() || !c.Valid() {
		t.Errorf("ability check help = %v, %v", got, c.Proficiency())
	}
	if got := r.Combatants(); len(got) != 2 || got[0] != gid("h") || got[1] != gid("e") || !r.Valid() {
		t.Errorf("attack roll help = %v", got)
	}
	if (combat.AbilityCheckHelp{}).Valid() || (combat.AttackRollHelp{}).Valid() {
		t.Error("a zero benefit is valid")
	}
	sameA, sameB := checkHelp(t, "h", "y", athletics()), checkHelp(t, "h", "y", athletics())
	if sameA != sameB {
		t.Error("benefits with equal fields must be equal")
	}
	if checkHelp(t, "h", "y", athletics()) == checkHelp(t, "h", "y", stealth()) ||
		checkHelp(t, "h", "y", athletics()) == checkHelp(t, "h", "z", athletics()) || rollHelp(t, "h", "e") == rollHelp(t, "h", "f") {
		t.Error("benefits must be equal by value")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-070, FR-071 - the enums and refusal reasons have snake_case names, and an undefined value is "unknown".
// Path: happy.
func TestHelpEnumNames(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{combat.AssistanceJudgementPossible.String(), "possible"},
		{combat.AssistanceJudgementNotPossible.String(), "not_possible"},
		{combat.AssistanceJudgement(0).String(), "unknown"},
		{combat.AttackAssistTargetEnemyWithinFiveFeet.String(), "enemy_within_five_feet"},
		{combat.AttackAssistTargetEnemyBeyondFiveFeet.String(), "enemy_beyond_five_feet"},
		{combat.AttackAssistTargetNotAnEnemy.String(), "not_an_enemy"},
		{combat.AttackAssistTarget(0).String(), "unknown"},
		{combat.AbilityCheckHelpRefusalCannotHelpSelf.String(), "cannot_help_self"},
		{combat.AbilityCheckHelpRefusalHelperLacksProficiency.String(), "helper_lacks_proficiency"},
		{combat.AbilityCheckHelpRefusalAssistanceNotPossible.String(), "assistance_not_possible"},
		{combat.AbilityCheckHelpRefusal(0).String(), "unknown"},
		{combat.AttackRollHelpRefusalTargetNotAnEnemy.String(), "target_not_an_enemy"},
		{combat.AttackRollHelpRefusalEnemyNotWithinFiveFeet.String(), "enemy_not_within_five_feet"},
		{combat.AttackRollHelpRefusal(0).String(), "unknown"},
	}
	for _, p := range pairs {
		if p[0] != p[1] {
			t.Errorf("String() = %q, want %q", p[0], p[1])
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-065 - AlliedHelpers drops repeats, rejects a blank id, compares as a set and its zero value is empty.
// Path: unhappy (invalid input).
func TestAlliedHelpers(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewAlliedHelpers(gid("a"), ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank id: %v", err)
	}
	a := alliedOf(t, "a", "b", "a")
	var zero combat.AlliedHelpers
	if !a.Contains(gid("a")) || !a.Contains(gid("b")) || a.Contains(gid("c")) || zero.Contains(gid("a")) {
		t.Error("Contains")
	}
	if !a.Equal(alliedOf(t, "b", "a")) || a.Equal(alliedOf(t, "a")) || a.Equal(alliedOf(t, "a", "c")) || !zero.Equal(alliedOf(t)) {
		t.Error("Equal must compare as a set")
	}
}
