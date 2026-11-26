package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-060, FR-068 - the three Attack action events are about the one combatant (a fresh copy), have constant snake_case kinds, belong to the AttackActionEvent family, and are equal by value.
// Path: happy.
func TestAttackActionEvents(t *testing.T) {
	t.Parallel()
	made, err := combat.NewAttackMadeInAttackAction(gid("hero"), combat.AttackMeansPactWeapon, 2)
	if err != nil {
		t.Fatal(err)
	}
	replaced, _ := combat.NewAttackReplacedInAttackAction(gid("hero"), replacementOf(t), 1)
	changed, _ := combat.NewWeaponChangedInAttackAction(gid("hero"), combat.WeaponChangeSheathe, combat.WeaponChangeAfterAttack, 3)
	combattest.CheckEvent(t, made, "attack_made_in_attack_action", gid("hero"))
	combattest.CheckEvent(t, replaced, "attack_replaced_in_attack_action", gid("hero"))
	combattest.CheckEvent(t, changed, "weapon_changed_in_attack_action", gid("hero"))
	combattest.CheckEventKinds(t, []combat.Event{made, replaced, changed})
	for _, e := range []combat.AttackActionEvent{made, replaced, changed} {
		if len(e.Combatants()) != 1 {
			t.Errorf("%s names %v, want the combatant alone", e.Kind(), e.Combatants())
		}
	}
	if made.Means() != combat.AttackMeansPactWeapon || made.Number() != 2 || replaced.Number() != 1 || replaced.Replacement() != replacementOf(t) ||
		changed.Change() != combat.WeaponChangeSheathe || changed.ChangeKind() != combat.WeaponChangeKindUnequip ||
		changed.Timing() != combat.WeaponChangeAfterAttack || changed.Slot() != 3 {
		t.Error("accessors disagree with the constructor arguments")
	}
	same, _ := combat.NewAttackMadeInAttackAction(gid("hero"), combat.AttackMeansPactWeapon, 2)
	diffNumber, _ := combat.NewAttackMadeInAttackAction(gid("hero"), combat.AttackMeansPactWeapon, 3)
	diffMeans, _ := combat.NewAttackMadeInAttackAction(gid("hero"), combat.AttackMeansOtherWeapon, 2)
	diffWho, _ := combat.NewAttackMadeInAttackAction(gid("villain"), combat.AttackMeansPactWeapon, 2)
	if made != same || made == diffNumber || made == diffMeans || made == diffWho {
		t.Error("AttackMadeInAttackAction equality must follow combatant, means and number [FR-068]")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-068 - an Attack action event constructor rejects a blank combatant, a number or slot below 1, and an invalid means, replacement, change or timing.
// Path: unhappy (invalid input).
func TestAttackActionEventConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	checks := map[string]error{
		"made blank":        second(combat.NewAttackMadeInAttackAction("", combat.AttackMeansPactWeapon, 1)),
		"made number 0":     second(combat.NewAttackMadeInAttackAction(gid("a"), combat.AttackMeansPactWeapon, 0)),
		"made zero means":   second(combat.NewAttackMadeInAttackAction(gid("a"), 0, 1)),
		"replaced blank":    second(combat.NewAttackReplacedInAttackAction("", replacementOf(t), 1)),
		"replaced number 0": second(combat.NewAttackReplacedInAttackAction(gid("a"), replacementOf(t), 0)),
		"replaced zero":     second(combat.NewAttackReplacedInAttackAction(gid("a"), combat.AttackReplacement{}, 1)),
		"changed blank":     second(combat.NewWeaponChangedInAttackAction("", combat.WeaponChangeDrop, combat.WeaponChangeAfterAttack, 1)),
		"changed slot 0":    second(combat.NewWeaponChangedInAttackAction(gid("a"), combat.WeaponChangeDrop, combat.WeaponChangeAfterAttack, 0)),
		"changed no change": second(combat.NewWeaponChangedInAttackAction(gid("a"), 0, combat.WeaponChangeAfterAttack, 1)),
		"changed no timing": second(combat.NewWeaponChangedInAttackAction(gid("a"), combat.WeaponChangeDrop, 0, 1)),
	}
	for name, err := range checks {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
