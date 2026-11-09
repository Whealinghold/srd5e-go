package character_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/magic-item-rules.md section 7, FR-109 - the magic item signal names, attribute keys and values are a public
// contract: their strings are pinned here and each appears in doc/observability/signals.md; the rejection values that come from enum
// String methods are pinned too.
// Path: happy.
func TestMagicItemSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	pins := []struct{ got, want string }{
		{character.SpanAttuneToMagicItem, "srd5e.character.attune_magic_item"},
		{character.SpanEndMagicItemAttunement, "srd5e.character.end_magic_item_attunement"},
		{character.SpanRechargeMagicItem, "srd5e.character.recharge_magic_item"},
		{character.SpanMixPotions, "srd5e.character.mix_potions"},
		{character.SpanCheckCraftingMaterials, "srd5e.character.check_crafting_materials"},
		{character.SpanPlanMagicItemCrafting, "srd5e.character.plan_magic_item_crafting"},
		{character.SpanGenerateSentientTraits, "srd5e.character.generate_sentient_traits"},
		{character.SpanResolveSentientConflict, "srd5e.character.resolve_sentient_conflict"},
		{character.SpanAttemptSentientControl, "srd5e.character.attempt_sentient_control"},
		{character.SpanRepeatSentientControlSave, "srd5e.character.repeat_sentient_control_save"},
		{character.AttrRechargeMoment, "srd5e.character.recharge_moment"},
		{character.ValueOperationAttuneMagicItem, "attune_magic_item"},
		{character.ValueOperationEndMagicItemAttunement, "end_magic_item_attunement"},
		{character.ValueOperationRechargeMagicItem, "recharge_magic_item"},
		{character.ValueOperationMixPotions, "mix_potions"},
		{character.ValueOperationCheckCraftingMaterials, "check_crafting_materials"},
		{character.ValueOperationPlanMagicItemCrafting, "plan_magic_item_crafting"},
		{character.ValueOperationGenerateSentientTraits, "generate_sentient_traits"},
		{character.ValueOperationResolveSentientConflict, "resolve_sentient_conflict"},
		{character.ValueOperationAttemptSentientControl, "attempt_sentient_control"},
		{character.ValueOperationRepeatSentientControl, "repeat_sentient_control_save"},
		{character.ValueResultRolled, "rolled"},
		{character.ValueResultPlanned, "planned"},
		{character.ValueResultGenerated, "generated"},
		{character.ValueResultResisted, "resisted"},
		{character.ValueResultDemandsMade, "demands_made"},
		{character.ValueResultUnchanged, "unchanged"},
		{character.ValueResultNotDue, "not_due"},
		{character.ValueResultUnavailable, "unavailable"},
		{character.ValueResultContentIncomplete, "content_incomplete"},
		{character.ValueRejectionTooFewPotions, "too_few_potions"},
		{character.ValueRejectionRarityUnresolved, "rarity_unresolved"},
		{character.ValueRejectionPurposeDetailMissing, "purpose_detail_missing"},
		{character.ValueRejectionUnalignedNotAllowed, "unaligned_not_allowed"},
		{character.ValueRejectionAlignedPurposeInvalid, "aligned_purpose_invalid"},
		{character.ValueRejectionAlignedPurposeNeedsNonNeutralAlignment, "aligned_purpose_needs_non_neutral_alignment"},
		{character.ValueRejectionDemandsInvalid, "demands_invalid"},
		{character.ValueRejectionPowerSpent, "power_spent"},
		{character.AttuneWrongRestPurpose.String(), "wrong_rest_purpose"},
		{character.AttuneRestNotFocusedOnItem.String(), "rest_not_focused"},
		{character.AttuneAlreadyAttunedToItem.String(), "already_attuned"},
		{character.AttuneDuplicateCopy.String(), "duplicate_copy"},
		{character.AttunePrerequisiteJudgementMissing.String(), "prerequisite_judgement_missing"},
		{character.AttunePrerequisiteNotMet.String(), "prerequisite_not_met"},
		{character.AttuneLimitReached.String(), "limit_reached"},
		{character.AttuneRestInterrupted.String(), "rest_interrupted"},
		{character.EndCurseBlocksEnding.String(), "curse_blocks_ending"},
		{character.EndNotAttuned.String(), "not_attuned"},
		{character.EndNotCursed.String(), "not_cursed"},
		{character.EndSpellDoesNotEndAttunement.String(), "spell_does_not_end_attunement"},
		{character.EndRestInterrupted.String(), "rest_interrupted"},
		{character.EndRestNotFocusedOnItem.String(), "rest_not_focused"},
		{character.EndWrongRestPurpose.String(), "wrong_rest_purpose"},
		{character.EndEndingConditionNotMet.String(), "ending_condition_not_met"},
		{character.ShortfallArcanaProficiency.String(), "arcana"},
		{character.ShortfallToolProficiency.String(), "tool"},
		{character.ShortfallSpellNotPrepared.String(), "spell_not_prepared"},
		{character.ShortfallTooManyAssistants.String(), "too_many_assistants"},
		{character.GeneratePurposeDetailMissing.String(), "purpose_detail_missing"},
		{character.GenerateUnalignedNotAllowed.String(), "unaligned_not_allowed"},
		{character.GenerateAlignedPurposeInvalid.String(), "aligned_purpose_invalid"},
		{character.GenerateAlignedPurposeNeedsNonNeutralAlignment.String(), "aligned_purpose_needs_non_neutral_alignment"},
	}
	for _, c := range pins {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
		if spanName := strings.HasPrefix(c.want, "srd5e.character.") && c.want != character.AttrRechargeMoment; !spanName && !strings.Contains(string(rules), c.want) {
			t.Errorf("%q is missing from doc/rules/observability.md section 5", c.want)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-122 - an id is a non-blank string; the seven events are a closed family of Events that
// carry the character id passed in.
// Path: unhappy (invalid input).
func TestMagicItemInstanceIDAndPairID(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "  ", "\t"} {
		if _, err := character.NewMagicItemInstanceID(s); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("instance %q: %v", s, err)
		}
		if _, err := character.NewPairID(s); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("pair %q: %v", s, err)
		}
	}
	if i, err := character.NewMagicItemInstanceID("a"); err != nil || i.String() != "a" || !i.Valid() || character.MagicItemInstanceID("").Valid() {
		t.Errorf("instance: %v %v", i, err)
	}
	if p, err := character.NewPairID("p"); err != nil || p.String() != "p" {
		t.Errorf("pair: %v %v", p, err)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-120, FR-122 - the event family has exactly the seven events, each an Event carrying the
// character id and its own fields.
// Path: happy.
func TestMagicItemEventsCarryTheCharacterID(t *testing.T) {
	t.Parallel()
	set := attunedSet(t, "ring", "r1")
	var events []character.MagicItemEvent
	attuned := must(character.NewAttuneToMagicItem().Do(t.Context(), character.AttuneToMagicItemCommand{CharacterID: holder, Set: attunedSet(t), Item: attunable(t, "ring", "n", content.AttunementRequired{}),
		Candidate: candidate(t, character.CanCast), Focus: focusOf(t, "n", character.RestPurposeAttune, character.InContact, true), Limit: 3}))
	events = append(events, attuned.(character.AttuneToMagicItemApplied).Events()...)
	_, ended, _ := set.EndBecauseAnotherAttuned(holder, inst("r1"))
	spent := must(character.MustItemCharges(3, 3).Spend(holder, inst("w"), 1)).(character.ItemChargesSpendApplied).Event
	used := must(character.MagicItemUse{}.Use(holder, inst("p"), true, character.ItemIntact)).(character.MagicItemUseApplied).Events()
	events = append(events, ended, spent)
	events = append(events, used...)
	for _, e := range events {
		var asEvent character.Event = e
		if asEvent.CharacterID() != holder {
			t.Errorf("%T carries %v", e, e.CharacterID())
		}
	}
	if len(events) != 4 {
		t.Errorf("events = %d", len(events))
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-060, FR-122 - a blank id is not valid whatever the whitespace, and each attunement end reason has its own name.
// Path: unhappy (invalid input).
func TestInstanceValidityAndEndReasonNames(t *testing.T) {
	t.Parallel()
	for _, s := range []character.MagicItemInstanceID{"", " ", "\t"} {
		if s.Valid() {
			t.Errorf("%q is valid", s)
		}
	}
	want := []string{"prerequisite_no_longer_met", "item_too_far_away", "died", "another_creature_attuned", "ended_by_short_rest", "curse_broken"}
	for i, w := range want {
		if got := character.AttunementEndReason(i + 1).String(); got != w {
			t.Errorf("reason %d = %q, want %q", i+1, got, w)
		}
	}
}
