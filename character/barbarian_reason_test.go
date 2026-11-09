package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-barbarian.md section 7 and doc/rules/observability.md section 5 - every reason has its snake_case name, which is
// the srd5e.character.rejection value, in check order; the zero and an out-of-range value are unknown and not valid.
// Path: per case.
func TestBarbarianReasonNames(t *testing.T) {
	t.Parallel()
	want := []string{"not_a_barbarian", "feature_not_gained", "not_a_berserker", "rage_already_active", "rage_not_active",
		"wearing_heavy_armor", "incapacitated", "no_bonus_action", "no_reaction", "no_rage_uses_remaining",
		"rage_began_this_turn", "rage_already_extended", "rage_lasts_without_extension", "not_first_attack_roll",
		"reckless_not_declared", "not_a_strength_attack", "attack_missed", "attack_has_disadvantage",
		"effect_not_unlocked", "duplicate_effect", "too_many_effects", "frenzy_target_already_hit", "not_dropped_to_zero",
		"persistent_rage_spent", "presence_not_restored", "presence_not_used", "source_not_within_five_feet",
		"event_not_for_character", "wrong_mastery_count", "duplicate_mastery", "not_simple_or_martial_melee_weapon",
		"drops_existing_mastery", "no_long_rest_finished", "not_currently_chosen", "already_chosen"}
	last := character.BarbarianAlreadyChosen
	if int(last) != len(want) {
		t.Fatalf("the last reason is %d, the table lists %d", int(last), len(want))
	}
	for i, name := range want {
		r := character.BarbarianReason(i + 1)
		if r.String() != name || !r.Valid() {
			t.Errorf("reason %d = %q valid %v, want %q valid", i+1, r.String(), r.Valid(), name)
		}
	}
	for _, r := range []character.BarbarianReason{0, last + 1, -1} {
		if r.String() != "unknown" || r.Valid() {
			t.Errorf("reason %d = %q valid %v, want unknown and not valid", int(r), r.String(), r.Valid())
		}
	}
}
