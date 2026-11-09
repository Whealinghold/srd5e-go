package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-sorcerer.md section 5, FR-124 and doc/observability/signals.md - each SorcererRejectionKind has its snake_case name, the srd5e.character.rejection value, in the order of the table of section 5; zero and values
// beyond the last are not kinds.
// Path: per case (happy / unhappy: out of range).
func TestSorcererRejectionKindNames(t *testing.T) {
	t.Parallel()
	names := []string{"not_a_sorcerer", "feature_not_gained", "elemental_affinity_choice_invalid", "elemental_affinity_type_mismatch",
		"elemental_affinity_already_applied", "metamagic_selection_invalid", "metamagic_replacement_invalid", "innate_sorcery_already_active",
		"innate_sorcery_no_uses_left", "innate_sorcery_uses_remain", "sorcery_incarnate_unavailable", "innate_sorcery_not_active",
		"sorcery_points_insufficient", "no_slot_to_convert", "sorcery_points_would_exceed_maximum", "slot_level_not_creatable",
		"sorcerer_level_below_minimum", "created_slot_unavailable", "created_slot_level_too_low", "created_slot_spent_this_turn",
		"metamagic_option_not_known", "metamagic_combination_refused", "quickened_restriction_applies", "metamagic_not_eligible",
		"restoration_already_used", "restoration_amount_invalid", "restoration_not_a_short_rest", "dragon_wings_already_used",
		"dragon_wings_use_not_expended", "dragon_wings_not_active", "dragon_companion_free_cast_used"}
	for i, want := range names {
		k := character.SorcererRejectionKind(i + 1)
		if k.String() != want || !k.Valid() {
			t.Errorf("kind %d = %q valid %v, want %q", i+1, k, k.Valid(), want)
		}
	}
	for _, k := range []character.SorcererRejectionKind{0, -1, character.SorcererRejectionKind(len(names) + 1)} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("kind %d = %q valid %v, want unknown", int(k), k, k.Valid())
		}
	}
	if character.SorcererNotASorcerer != 1 || character.SorcererDragonCompanionFreeCastUsed != character.SorcererRejectionKind(len(names)) {
		t.Error("the first and last kinds moved")
	}
}

// Requirement: doc/specs/class-sorcerer.md section 5 - the MetamagicIneligibility, MetamagicSelectionProblem and MetamagicFact enums have snake_case names, and zero is none of them.
// Path: per case (happy / unhappy: out of range).
func TestMetamagicEnumNames(t *testing.T) {
	t.Parallel()
	ineligible := []string{"no_saving_throw", "range_not_eligible", "duration_not_eligible", "duration_already_at_maximum", "casting_time_not_action",
		"no_missed_attack_roll", "nothing_to_remove", "damage_type_not_transmutable", "no_higher_slot_target", "count_out_of_range", "spell_fact_unknown"}
	for i, want := range ineligible {
		if got := character.MetamagicIneligibility(i + 1).String(); got != want {
			t.Errorf("ineligibility %d = %q, want %q", i+1, got, want)
		}
	}
	for i, want := range []string{"wrong_size", "duplicate", "unknown_option"} {
		if got := character.MetamagicSelectionProblem(i + 1).String(); got != want {
			t.Errorf("problem %d = %q, want %q", i+1, got, want)
		}
	}
	for i, want := range []string{"saving_throw", "range", "duration", "casting_time", "components", "attack_roll", "damage", "higher_slot_target"} {
		if got := character.MetamagicFact(i + 1).String(); got != want {
			t.Errorf("fact %d = %q, want %q", i+1, got, want)
		}
	}
	for _, got := range []string{character.MetamagicIneligibility(0).String(), character.MetamagicIneligibility(12).String(), character.MetamagicSelectionProblem(0).String(),
		character.MetamagicSelectionProblem(4).String(), character.MetamagicFact(0).String(), character.MetamagicFact(9).String()} {
		if got != "unknown" {
			t.Errorf("out of range name = %q, want unknown", got)
		}
	}
}
