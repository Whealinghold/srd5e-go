package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/magic-item-effects.md FR-001, FR-002 (SRD 5.2.1, "Armor, +1, +2, or +3", p. 210; "Shield", p. 243; "Weapon", p. 253;
// "Ammunition", p. 209; "Wand of the War Mage", p. 251) - every cell of table "Enhancement bonuses": a listed rarity gives its bonus, any other is
// EnhancementBonusNotListed carrying the family and rarity, never an error.
// Path: per case.
func TestEnhancementBonusLookup(t *testing.T) {
	t.Parallel()
	rarities := []content.ItemRarity{content.RarityCommon, content.RarityUncommon, content.RarityRare, content.RarityVeryRare, content.RarityLegendary, content.RarityArtifact}
	armor := []int{0, 0, 1, 2, 3, 0}
	other := []int{0, 1, 2, 3, 0, 0}
	for _, f := range []character.EnhancementFamily{character.EnhancementArmor, character.EnhancementShield, character.EnhancementWeapon, character.EnhancementAmmunition, character.EnhancementWarMageWand} {
		want := other
		if f == character.EnhancementArmor {
			want = armor
		}
		for i, r := range rarities {
			switch got := (character.EnhancementBonusLookup{}).Of(f, r).(type) {
			case character.EnhancementBonusFound:
				if got.Modifier().Value() != want[i] || want[i] == 0 {
					t.Errorf("%s %s = %+v, want %d", f, r, got, want[i])
				}
			case character.EnhancementBonusNotListed:
				if want[i] != 0 || got.Family() != f || got.Rarity() != r {
					t.Errorf("%s %s not listed = %+v, want %d", f, r, got, want[i])
				}
			}
		}
	}
	for _, bad := range []struct {
		f character.EnhancementFamily
		r content.ItemRarity
	}{{0, content.RarityRare}, {character.EnhancementWarMageWand + 1, content.RarityRare}, {character.EnhancementArmor, 0}, {character.EnhancementShield, content.RarityArtifact + 1}} {
		if _, ok := (character.EnhancementBonusLookup{}).Of(bad.f, bad.r).(character.EnhancementBonusNotListed); !ok {
			t.Errorf("%v %v: want not listed", bad.f, bad.r)
		}
	}
	if character.EnhancementArmor.String() != "armor" || character.EnhancementWarMageWand.String() != "war_mage_wand" || character.EnhancementFamily(0).String() != "unknown" || (character.EnhancementWarMageWand+1).String() != "unknown" {
		t.Error("family names")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-006 (SRD 5.2.1, "Ammunition, +1, +2, or +3", p. 209) - magic ammunition becomes nonmagical on a hit,
// stays magical on a miss, and nonmagical ammunition stays so either way.
// Path: unhappy (boundary).
func TestMagicAmmunitionAfterAttack(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		state character.AmmunitionState
		hit   bool
		want  character.AmmunitionState
	}{
		{"magical hit", character.MagicalAmmunition, true, character.NonmagicalAmmunition},
		{"magical miss", character.MagicalAmmunition, false, character.MagicalAmmunition},
		{"nonmagical hit", character.NonmagicalAmmunition, true, character.NonmagicalAmmunition},
		{"nonmagical miss", character.NonmagicalAmmunition, false, character.NonmagicalAmmunition},
	} {
		if got := (character.MagicAmmunition{}).AfterAttack(c.state, c.hit); got != c.want {
			t.Errorf("%s = %v, want %v", c.name, got, c.want)
		}
	}
}
