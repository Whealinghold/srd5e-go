package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-029 - Draw, PickUp and EquipOther equip; Sheathe, Stow, Drop and UnequipOther unequip; an undefined change has no kind.
// Path: per case.
func TestWeaponChangeKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		change combat.AttackActionWeaponChange
		kind   combat.WeaponChangeKind
		name   string
	}{
		{combat.WeaponChangeDrawFromSheath, combat.WeaponChangeKindEquip, "draw_from_sheath"},
		{combat.WeaponChangePickUp, combat.WeaponChangeKindEquip, "pick_up"},
		{combat.WeaponChangeEquipOther, combat.WeaponChangeKindEquip, "equip_other"},
		{combat.WeaponChangeSheathe, combat.WeaponChangeKindUnequip, "sheathe"},
		{combat.WeaponChangeStow, combat.WeaponChangeKindUnequip, "stow"},
		{combat.WeaponChangeDrop, combat.WeaponChangeKindUnequip, "drop"},
		{combat.WeaponChangeUnequipOther, combat.WeaponChangeKindUnequip, "unequip_other"},
		{0, 0, "unknown"},
		{8, 0, "unknown"},
	}
	for _, c := range cases {
		if c.change.Change() != c.kind || c.change.String() != c.name || c.change.Valid() != (c.kind != 0) {
			t.Errorf("%d: Change = %v, String = %q, Valid = %v", c.change, c.change.Change(), c.change.String(), c.change.Valid())
		}
	}
	if combat.WeaponChangeKindEquip.String() != "equip" || combat.WeaponChangeKindUnequip.String() != "unequip" || combat.WeaponChangeKind(0).String() != "unknown" {
		t.Error("WeaponChangeKind names")
	}
	if combat.WeaponChangeBeforeAttack.String() != "before_attack" || combat.WeaponChangeAfterAttack.String() != "after_attack" ||
		combat.AttackActionWeaponChangeTiming(0).String() != "unknown" || combat.AttackActionWeaponChangeTiming(3).Valid() {
		t.Error("timing names or validity")
	}
	if combat.WeaponChangeRefusalNoAttackMade.String() != "no_attack_made" || combat.WeaponChangeRefusalNoAttackRemaining.String() != "no_attack_remaining" ||
		combat.WeaponChangeRefusalAlreadyUsedForThisAttack.String() != "already_used_for_this_attack" || combat.WeaponChangeRefusal(0).String() != "unknown" ||
		combat.AttackRefusalLimitReached.String() != "attack_limit_reached" || combat.AttackRefusal(0).String() != "unknown" {
		t.Error("refusal names")
	}
}
