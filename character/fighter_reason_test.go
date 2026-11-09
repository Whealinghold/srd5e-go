package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-fighter.md section 5 (reason tables) and observability.md section 5 - every reason has its
// snake_case name, which is the srd5e.character.rejection value; the zero and an out-of-range value are unknown and not valid.
// Path: per case.
func TestFighterReasonNames(t *testing.T) {
	t.Parallel()
	want := map[character.FighterReason]string{
		character.FighterNotAFighter: "not_a_fighter", character.FighterFeatureNotGained: "feature_not_gained",
		character.FighterNoUsesRemaining: "no_uses_remaining", character.FighterNotAnAbilityCheck: "not_an_ability_check",
		character.FighterCheckNotFailed: "check_not_failed", character.FighterNotASavingThrow: "not_a_saving_throw",
		character.FighterDieNotRolled: "die_not_rolled", character.FighterSaveNotFailed: "save_not_failed",
		character.FighterNotOwnTurn: "not_own_turn", character.FighterAlreadyUsedThisTurn: "already_used_this_turn",
		character.FighterFightingStyleAlreadyChosen: "fighting_style_already_chosen", character.FighterNoStyleToReplace: "no_style_to_replace",
		character.FighterEventNotForCharacter: "event_not_for_character", character.FighterNoLevelGained: "no_level_gained",
		character.FighterAdditionalStyleAlreadyChosen: "additional_style_already_chosen", character.FighterNotAFightingStyleFeat: "not_a_fighting_style_feat",
		character.FighterSameFeat: "same_feat", character.FighterFeatNotRepeatable: "feat_not_repeatable",
		character.FighterWrongMasteryCount: "wrong_mastery_count", character.FighterDuplicateMastery: "duplicate_mastery",
		character.FighterNotSimpleOrMartialWeapon: "not_simple_or_martial_weapon", character.FighterDropsExistingMastery: "drops_existing_mastery",
		character.FighterNoLongRestFinished: "no_long_rest_finished", character.FighterNotCurrentlyChosen: "not_currently_chosen",
		character.FighterAlreadyChosen: "already_chosen", character.FighterMasteryNotUsable: "mastery_not_usable",
		character.FighterNotInCombat: "not_in_combat", character.FighterAlreadyHasInspiration: "already_has_inspiration",
		character.FighterNoHitPoints: "no_hit_points", character.FighterNotBloodied: "not_bloodied",
	}
	if len(want) != 30 {
		t.Fatalf("table lists %d reasons, want 30", len(want))
	}
	for r, name := range want {
		if r.String() != name || !r.Valid() {
			t.Errorf("reason %d = %q valid %v, want %q valid", int(r), r.String(), r.Valid(), name)
		}
	}
	for _, r := range []character.FighterReason{0, 31, -1} {
		if r.String() != "unknown" || r.Valid() {
			t.Errorf("reason %d = %q valid %v, want unknown and not valid", int(r), r.String(), r.Valid())
		}
	}
}
