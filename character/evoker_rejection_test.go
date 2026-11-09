package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// evkKindNames are the String of every Evoker rejection Kind, in constant order per type.
func evkKindNames() [][]string {
	var spellbook, potent, sculpt, empowered, overchannel []string
	for k := character.EvokerSpellbookNotEvoker; k <= character.EvokerSpellbookAlreadyInSpellbook; k++ {
		spellbook = append(spellbook, k.String())
	}
	for k := character.PotentCantripNotEvoker; k <= character.PotentCantripSpellDealsNoDamage; k++ {
		potent = append(potent, k.String())
	}
	for k := character.SculptNotEvoker; k <= character.SculptTooManyCreatures; k++ {
		sculpt = append(sculpt, k.String())
	}
	for k := character.EmpoweredNotEvoker; k <= character.EmpoweredAlreadyEmpowered; k++ {
		empowered = append(empowered, k.String())
	}
	for k := character.OverchannelNotEvoker; k <= character.OverchannelSpellDealsNoDamage; k++ {
		overchannel = append(overchannel, k.String())
	}
	return [][]string{spellbook, potent, sculpt, empowered, overchannel}
}

// Requirement: doc/specs/class-wizard-evoker.md section 5 (rejection types), class-wizard.md FR-097, FR-100 - each Evoker rejection Kind has the snake_case name of its variant, in the reporting order of section 5 (the
// srd5e.character.rejection value); the zero and an out-of-range Kind are "unknown".
// Path: per case (happy / unhappy: boundary at the ends of each enum).
func TestEvokerRejectionNames(t *testing.T) {
	t.Parallel()
	want := [][]string{
		{"not_evoker", "feature_not_available", "no_new_spell_level", "wrong_spell_count", "duplicate_spell", "is_a_cantrip", "not_on_wizard_list", "not_an_evocation_spell", "above_highest_spell_level", "already_in_spellbook"},
		{"not_evoker", "feature_not_available", "unsupported_test_kind", "not_a_cantrip", "spell_deals_no_damage"},
		{"not_evoker", "feature_not_available", "not_an_evocation_spell", "caster", "creature_not_eligible", "too_many_creatures"},
		{"not_evoker", "feature_not_available", "not_a_wizard_spell", "not_an_evocation_spell", "already_empowered"},
		{"not_evoker", "feature_not_available", "not_a_wizard_spell", "no_spell_slot_used", "slot_level_out_of_range", "spell_deals_no_damage"},
	}
	got := evkKindNames()
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("type %d has %d kinds, want %d", i, len(got[i]), len(want[i]))
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("type %d kind %d = %q, want %q", i, j+1, got[i][j], want[i][j])
			}
		}
	}
	for _, unknown := range []string{
		character.EvokerSpellbookRejectionKind(0).String(), character.EvokerSpellbookRejectionKind(11).String(), character.PotentCantripRejectionKind(0).String(), character.PotentCantripRejectionKind(6).String(),
		character.SculptRejectionKind(0).String(), character.SculptRejectionKind(7).String(), character.EmpoweredRejectionKind(0).String(), character.EmpoweredRejectionKind(6).String(),
		character.OverchannelRejectionKind(0).String(), character.OverchannelRejectionKind(7).String(),
	} {
		if unknown != "unknown" {
			t.Errorf("out-of-range Kind = %q, want unknown", unknown)
		}
	}
}
