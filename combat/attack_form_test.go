package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/attacks.md FR-087 - an AttackForm is one of five legal pairs of delivery and source; an
// Unarmed Strike is always melee (SRD 5.2.1, "Melee Attacks", "Ranged Attacks", p. 15; "Spell Attack", p. 188).
// Path: per case.
func TestAttackFormDeliveryAndSource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		form                    combat.AttackForm
		delivery                combat.AttackDelivery
		source                  combat.AttackSource
		text                    string
	}{
		{"melee weapon", "FR-087", "happy", combat.MeleeWeaponAttack, combat.AttackDeliveryMelee, combat.AttackSourceWeapon, "melee_weapon"},
		{"ranged weapon", "FR-087", "happy", combat.RangedWeaponAttack, combat.AttackDeliveryRanged, combat.AttackSourceWeapon, "ranged_weapon"},
		{"melee spell", "FR-087, FR-088", "happy", combat.MeleeSpellAttack, combat.AttackDeliveryMelee, combat.AttackSourceSpell, "melee_spell"},
		{"ranged spell", "FR-087, FR-089", "happy", combat.RangedSpellAttack, combat.AttackDeliveryRanged, combat.AttackSourceSpell, "ranged_spell"},
		{"unarmed strike", "FR-087", "happy", combat.UnarmedStrikeAttack, combat.AttackDeliveryMelee, combat.AttackSourceUnarmedStrike, "unarmed_strike"},
		{"zero value", "FR-087", "unhappy (invalid input)", 0, 0, 0, "unknown"},
		{"past the last", "FR-087", "unhappy (invalid input)", combat.UnarmedStrikeAttack + 1, 0, 0, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.form.Delivery() != c.delivery || c.form.Source() != c.source || c.form.String() != c.text ||
				c.form.Valid() != (c.text != "unknown") {
				t.Errorf("%d = %v/%v/%q valid %v, want %v/%v/%q [%s]", c.form, c.form.Delivery(), c.form.Source(), c.form,
					c.form.Valid(), c.delivery, c.source, c.text, c.requirement)
			}
		})
	}
	if combat.AttackDelivery(0).String() != "unknown" || combat.AttackSource(9).String() != "unknown" ||
		combat.AttackDeliveryMelee.String() != "melee" || combat.AttackDeliveryRanged.String() != "ranged" ||
		combat.AttackSourceWeapon.String() != "weapon" || combat.AttackSourceSpell.String() != "spell" ||
		combat.AttackSourceUnarmedStrike.String() != "unarmed_strike" {
		t.Errorf("delivery and source names are wrong")
	}
}
