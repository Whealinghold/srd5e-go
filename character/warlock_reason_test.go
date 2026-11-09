package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-warlock.md FR-147, doc/rules/observability.md section 5 - every WarlockRejectionKind has its own snake_case String (the srd5e.character.rejection value), the kinds are numbered from 1 with no gap, and the zero and an out-of-range value are invalid and read "unknown".
// Path: per case (every kind; boundary: zero and one past the last).
func TestWarlockRejectionKindStrings(t *testing.T) {
	t.Parallel()
	want := map[character.WarlockRejectionKind]string{
		character.WarlockNotAWarlock: "not_a_warlock", character.WarlockFeatureNotGained: "feature_not_gained",
		character.WarlockInvocationSelectionInvalid: "invocation_selection_invalid", character.WarlockInvocationNotKnown: "invocation_not_known",
		character.WarlockNotASpellGrant: "not_a_spell_grant", character.WarlockLightLevelWrong: "light_level", character.WarlockAlreadyCast: "already_cast",
		character.WarlockMagicalCunningAlreadyUsed: "magical_cunning_already_used", character.WarlockRiteTooShort: "rite_too_short",
		character.WarlockContactPatronAlreadyUsed: "contact_patron_already_used", character.WarlockNoArcanumAtLevel: "no_arcanum_at_level",
		character.WarlockArcanumWrongSpellLevel: "wrong_spell_level", character.WarlockArcanumNotOnWarlockList: "not_on_warlock_list",
		character.WarlockArcanumTooManyReplacements: "arcanum_too_many_replacements", character.WarlockNotAnArcanumSpell: "not_an_arcanum_spell",
		character.WarlockArcanumAlreadyCast: "arcanum_already_cast", character.WarlockNotFiendPatron: "not_fiend_patron", character.WarlockNotAnEnemy: "not_an_enemy",
		character.WarlockTargetTooFar: "target_too_far", character.WarlockForceNotAllowed: "force_not_allowed", character.WarlockNotAfterRest: "not_after_rest",
		character.WarlockNotAHit: "not_a_hit", character.WarlockAlreadyUsedThisTurn: "already_used_this_turn",
		character.WarlockHurlThroughHellAlreadySpent: "hurl_through_hell_already_spent", character.WarlockHurlThroughHellUseNotSpent: "hurl_through_hell_use_not_spent",
		character.WarlockNoPactMagicSlot: "no_pact_magic_slot", character.WarlockProneNotAllowedForSize: "prone_not_allowed_for_size",
		character.WarlockNoUsesLeft: "no_uses_left", character.WarlockAlreadyUsedOnThisRoll: "already_used_on_this_roll", character.WarlockRollKindNotAllowed: "roll_kind_not_allowed",
		character.WarlockNotMeleeSimpleOrMartial: "not_melee_simple_or_martial", character.WarlockAttunedByAnother: "attuned_by_another", character.WarlockBondedByAnotherWarlock: "bonded_by_another_warlock",
		character.WarlockDamageTypeNotOffered: "damage_type_not_offered", character.WarlockNoHitPointDie: "no_hit_point_die", character.WarlockFamiliarFormNotOffered: "familiar_form_not_offered",
		character.WarlockNoFamiliar: "no_familiar", character.WarlockNoAttackToForgo: "no_attack_to_forgo", character.WarlockReactionSpent: "reaction_spent", character.WarlockNoBonusAction: "no_bonus_action",
		character.WarlockDamageNotChangeable: "damage_not_changeable", character.WarlockTargetTypeNotOffered: "target_type_not_offered", character.WarlockNotWilling: "not_willing",
		character.WarlockNotTouched: "not_touched", character.WarlockNotOnSamePlane: "not_on_same_plane", character.WarlockLinkEnded: "link_ended",
		character.WarlockCantripCountWrong: "cantrip_count_wrong", character.WarlockSpellCountWrong: "spell_count_wrong", character.WarlockBookSpellRefused: "book_spell_refused",
		character.WarlockNoPage: "no_page", character.WarlockPermissionNotGiven: "permission_not_given", character.WarlockNoActionAvailable: "no_action_available",
		character.WarlockPageFull: "page_full", character.WarlockNameAlreadyOnPage: "name_already_on_page", character.WarlockNameNotOnPage: "name_not_on_page",
		character.WarlockNoMagicAction: "no_magic_action", character.WarlockGiftAlreadySpent: "gift_of_the_protectors_already_spent", character.WarlockKilledOutright: "killed_outright",
	}
	last := character.WarlockKilledOutright
	if len(want) != int(last) {
		t.Fatalf("table has %d kinds, the last kind is %d", len(want), last)
	}
	seen := map[string]bool{}
	for k := character.WarlockNotAWarlock; k <= last; k++ {
		if !k.Valid() || k.String() != want[k] || seen[k.String()] {
			t.Errorf("kind %d = %q (valid %v), want %q once", k, k.String(), k.Valid(), want[k])
		}
		seen[k.String()] = true
	}
	for _, k := range []character.WarlockRejectionKind{0, last + 1, -1} {
		if k.Valid() || k.String() != "unknown" {
			t.Errorf("kind %d must be invalid and read unknown, got %q", k, k.String())
		}
	}
}
