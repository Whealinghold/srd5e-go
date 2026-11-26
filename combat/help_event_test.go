package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-060, FR-068 - the four Help events are in the HelpEvent family, have constant snake_case kinds and name the helper then the ally or enemy (HelpBenefitUsed for an attack roll: the helper, the attacker, then the enemy), as a fresh copy, with no duplicate.
// Path: happy.
func TestHelpEvents(t *testing.T) {
	t.Parallel()
	check, roll := checkHelp(t, "h", "y", athletics()), rollHelp(t, "h", "e")
	given, err := combat.NewHelpGiven(check)
	if err != nil {
		t.Fatal(err)
	}
	givenRoll, _ := combat.NewHelpGiven(roll)
	usedCheck, err := combat.NewHelpBenefitUsed(check, gid("y"))
	if err != nil {
		t.Fatal(err)
	}
	usedRoll, err := combat.NewHelpBenefitUsed(roll, gid("a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := combat.NewHelpBenefitUsed(roll, gid("z")); err != nil {
		t.Errorf("an attacker that sorts after the helper and the enemy: %v", err)
	}
	expired, _ := combat.NewHelpBenefitExpired(check)
	dropped, _ := combat.NewHelpBenefitDropped(roll)
	combattest.CheckEvent(t, given, "help_given", gid("h"), gid("y"))
	combattest.CheckEvent(t, givenRoll, "help_given", gid("h"), gid("e"))
	combattest.CheckEvent(t, usedCheck, "help_benefit_used", gid("h"), gid("y"))
	combattest.CheckEvent(t, usedRoll, "help_benefit_used", gid("h"), gid("a"), gid("e"))
	combattest.CheckEvent(t, expired, "help_benefit_expired", gid("h"), gid("y"))
	combattest.CheckEvent(t, dropped, "help_benefit_dropped", gid("h"), gid("e"))
	combattest.CheckEventKinds(t, []combat.Event{given, usedCheck, expired, dropped})
	for _, e := range []combat.HelpEvent{given, usedCheck, expired, dropped} {
		if e.(interface{ Benefit() combat.HelpBenefit }).Benefit() == nil {
			t.Errorf("%s has no benefit", e.Kind())
		}
	}
	again, _ := combat.NewHelpBenefitUsed(roll, gid("a"))
	other, _ := combat.NewHelpBenefitUsed(roll, gid("b"))
	if usedRoll != again || usedRoll == other || given == givenRoll || given == mustGiven(t, checkHelp(t, "h", "y", stealth())) {
		t.Error("HelpEvent equality must follow the benefit and the roller [FR-068]")
	}
}

func mustGiven(tb testing.TB, b combat.HelpBenefit) combat.HelpGiven {
	tb.Helper()
	e, err := combat.NewHelpGiven(b)
	if err != nil {
		tb.Fatal(err)
	}
	return e
}

// Requirement: doc/specs/attack-and-help-actions.md FR-074 - a Help event constructor rejects a nil or zero benefit; HelpBenefitUsed rejects an ability check roller other than the ally, and an attacker that is blank, the helper or the enemy, so no event holds a duplicate combatant.
// Path: unhappy (invalid input).
func TestHelpEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	check, roll := checkHelp(t, "h", "y", athletics()), rollHelp(t, "h", "e")
	checks := map[string]error{
		"given nil":         second(combat.NewHelpGiven(nil)),
		"given zero":        second(combat.NewHelpGiven(combat.AbilityCheckHelp{})),
		"used nil":          second(combat.NewHelpBenefitUsed(nil, gid("y"))),
		"used zero":         second(combat.NewHelpBenefitUsed(combat.AttackRollHelp{}, gid("y"))),
		"used check other":  second(combat.NewHelpBenefitUsed(check, gid("z"))),
		"used check helper": second(combat.NewHelpBenefitUsed(check, gid("h"))),
		"used roll blank":   second(combat.NewHelpBenefitUsed(roll, "")),
		"used roll helper":  second(combat.NewHelpBenefitUsed(roll, gid("h"))),
		"used roll enemy":   second(combat.NewHelpBenefitUsed(roll, gid("e"))),
		"expired nil":       second(combat.NewHelpBenefitExpired(nil)),
		"dropped zero":      second(combat.NewHelpBenefitDropped(combat.AttackRollHelp{})),
	}
	for name, err := range checks {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
