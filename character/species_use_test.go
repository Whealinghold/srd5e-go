package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-030, FR-032, Table U (SRD 5.2.1, pp. 84-86) - each use has the limit, minimum level and renewal of Table U, and the limits that equal the Proficiency Bonus follow it at every level (1 and 4: 2; 5 and 8: 3; 9 and 12: 4; 13 and 16: 5; 17 and 20: 6).
// Path: per case.
func TestSpeciesUseTable(t *testing.T) {
	t.Parallel()
	pb := character.BreathWeaponUse
	for _, c := range []struct {
		name  string
		use   character.SpeciesUse
		limit int // at level 5, Proficiency Bonus 3
		min   int
		renew content.RechargeTrigger
	}{
		{"Breath Weapon", pb, 3, 1, content.RechargeLongRest},
		{"Draconic Flight", character.DraconicFlightUse, 1, 5, content.RechargeLongRest},
		{"Stonecunning", character.StonecunningUse, 3, 1, content.RechargeLongRest},
		{"Giant Ancestry", character.GiantAncestryUse, 3, 1, content.RechargeLongRest},
		{"Large Form", character.LargeFormUse, 1, 5, content.RechargeLongRest},
		{"Adrenaline Rush", character.AdrenalineRushUse, 3, 1, content.RechargeShortOrLongRest},
		{"Relentless Endurance", character.RelentlessEnduranceUse, 1, 1, content.RechargeLongRest},
		{"lineage spell 3", character.LineageSpellThreeUse, 1, 3, content.RechargeLongRest},
		{"lineage spell 5", character.LineageSpellFiveUse, 1, 5, content.RechargeLongRest},
		{"Speak with Animals", character.GnomeSpellUse, 3, 1, content.RechargeLongRest},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if !c.use.Valid() || c.use.String() == "unknown" || c.use.Limit(character.MustLevel(5)) != c.limit || c.use.MinimumLevel() != c.min || c.use.Renewal() != c.renew {
				t.Errorf("%v: limit %d, minimum %d, renewal %v", c.use, c.use.Limit(character.MustLevel(5)), c.use.MinimumLevel(), c.use.Renewal())
			}
		})
	}
	for level, want := range map[int]int{1: 2, 4: 2, 5: 3, 8: 3, 9: 4, 12: 4, 13: 5, 16: 5, 17: 6, 20: 6} {
		if got := pb.Limit(character.MustLevel(level)); got != want {
			t.Errorf("Breath Weapon limit at level %d = %d, want %d", level, got, want)
		}
	}
	var none character.SpeciesUse
	if none.Valid() || none.Limit(character.MustLevel(5)) != 0 || none.MinimumLevel() != 0 || none.Renewal() != 0 || character.SpeciesUse(11).Renewal() != 0 || none.String() != "unknown" {
		t.Error("an invalid use has facts")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md section 5 (windows, actions, shapes) - the closed enums have snake_case strings, valid members and an invalid zero value.
// Path: happy.
func TestSpeciesUseEnums(t *testing.T) {
	t.Parallel()
	if !character.WindowResourceful.Valid() || !character.WindowHighElfCantripReplacement.Valid() || character.SpeciesWindow(0).Valid() ||
		character.WindowResourceful.String() != "resourceful" || character.WindowHighElfCantripReplacement.String() != "high_elf_cantrip_replacement" || character.SpeciesWindow(3).String() != "unknown" {
		t.Error("SpeciesWindow")
	}
	if character.TraitBonusAction.String() != "bonus_action" || character.TraitReaction.String() != "reaction" || character.SpeciesTraitAction(0).String() != "unknown" {
		t.Error("SpeciesTraitAction")
	}
	if !character.BreathCone15.Valid() || !character.BreathLine30.Valid() || character.BreathWeaponShape(0).Valid() || character.BreathCone15.String() != "cone_15" || character.BreathLine30.String() != "line_30" {
		t.Error("BreathWeaponShape")
	}
	if character.BreathWeaponShape(3).Valid() {
		t.Error("shape 3 is valid")
	}
}
