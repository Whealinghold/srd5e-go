package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/character-aggregate.md FR-070, FR-103, FR-118 - the
// signal names, attribute keys and values are a public contract: their strings
// are pinned here and each appears in doc/observability/signals.md.
// Path: happy.
func TestSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanLevelUp, "srd5e.character.level_up"},
		{character.SpanAwardExperience, "srd5e.character.award_experience"},
		{character.SpanChangeAbilityScore, "srd5e.character.change_ability_score"},
		{character.SpanLoadContent, "srd5e.character.load_content"},
		{character.MetricOperations, "srd5e.character.operations"},
		{character.MetricDuration, "srd5e.character.duration"},
		{character.MetricFeatureUsed, "srd5e.character.feature_used"},
		{character.MetricFeatureUsedDuration, "srd5e.character.feature_used.duration"},
		{character.MetricFeatureUsedErrors, "srd5e.character.feature_used.errors"},
		{character.AttrOperation, "srd5e.character.operation"},
		{character.AttrResult, "srd5e.character.result"},
		{character.AttrRejection, "srd5e.character.rejection"},
		{character.AttrHitPointChoice, "srd5e.character.hit_point_choice"},
		{character.AttrClass, "srd5e.class"},
		{character.AttrFeature, "srd5e.feature"},
		{character.ValueOperationLevelUp, "level_up"},
		{character.ValueOperationAwardExperience, "award_experience"},
		{character.ValueOperationChangeAbilityScore, "change_ability_score"},
		{character.ValueResultChanged, "changed"},
		{character.ValueResultRejected, "rejected"},
		{character.ValueResultContentMissing, "content_missing"},
		{character.ValueRejectionMaximumLevel, "maximum_level"},
		{character.ValueRejectionInsufficientExperience, "insufficient_experience"},
		{character.ValueRejectionMulticlassPrerequisitesNotMet, "multiclass_prerequisites_not_met"},
		{character.ValueRejectionExceedsMaximumScore, "exceeds_maximum_score"},
		{character.ValueRejectionConstitutionModifierDecrease, "constitution_modifier_decrease"},
		{character.ValueHitPointChoiceRolled, "rolled"},
		{character.ValueHitPointChoiceFixed, "fixed"},
	} {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md section 7, FR-057 - the rest signal
// names, attribute keys and values are a public contract: their strings are pinned
// here and each appears in doc/observability/signals.md. The rejection values that
// come from enum String methods are pinned too.
// Path: happy.
func TestRestSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanTakeRest, "srd5e.character.take_rest"},
		{character.SpanSpendHitPointDie, "srd5e.character.spend_hit_point_die"},
		{character.MetricRests, "srd5e.character.rests"},
		{character.MetricHitPointDiceSpent, "srd5e.character.hit_point_dice_spent"},
		{character.MetricHitDieHitPoints, "srd5e.character.hit_die_hit_points"},
		{character.MetricRestOperationDuration, "srd5e.character.rest_operation_duration"},
		{character.AttrRestKind, "srd5e.character.rest_kind"},
		{character.AttrRestBenefits, "srd5e.character.rest_benefits"},
		{character.AttrHitDieFaces, "srd5e.character.hit_die_faces"},
		{character.AttrRestOperation, "srd5e.character.rest_operation"},
		{character.ValueRestOperationTakeRest, "take_rest"},
		{character.ValueRestOperationSpendHitPointDie, "spend_hit_point_die"},
		{character.ValueResultApplied, "applied"},
		{character.ValueResultAppliedAfterInterruption, "applied_after_interruption"},
		{character.ValueResultStopped, "stopped"},
		{character.ValueResultIncomplete, "incomplete"},
		{character.ValueResultNotStarted, "not_started"},
		{character.ValueResultSpent, "spent"},
		{character.ValueRestBenefitsShort, "short"},
		{character.ValueRestBenefitsLong, "long"},
		{character.ValueRestBenefitsNone, "none"},
		{character.ValueRejectionNoHitPoints, "no_hit_points"},
		{character.ValueRejectionExhaustionLevelSix, "exhaustion_level_six"},
		{character.ValueRejectionTooSoonAfterLongRest, "too_soon_after_long_rest"},
		{character.ShortRestCannotBeResumed.String(), "short_rest_cannot_be_resumed"},
		{character.NoneRemaining.String(), "none_remaining"},
		{character.NotInPool.String(), "not_in_pool"},
		{character.NotAShortRestBenefit.String(), "not_a_short_rest_benefit"},
		{character.SpentExceedsPool.String(), "spent_exceeds_pool"},
		{character.ShortRest.String(), "short"},
		{character.LongRest.String(), "long"},
	} {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
	}
}

// Requirement: doc/specs/heroic-inspiration.md FR-036, section 7 - the Heroic
// Inspiration span names and the values added to the shared operation, result and
// rejection keys are a public contract: their strings are pinned here and each
// appears in doc/observability/signals.md. The rejection values that come from enum
// String methods are pinned too.
// Path: happy.
func TestHeroicInspirationSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanGainHeroicInspiration, "srd5e.character.gain_heroic_inspiration"},
		{character.SpanExpendHeroicInspiration, "srd5e.character.expend_heroic_inspiration"},
		{character.ValueOperationGain, "gain"},
		{character.ValueOperationExpend, "expend"},
		{character.ValueResultGained, "gained"},
		{character.ValueResultExcessLost, "excess_lost"},
		{character.ValueResultExcessGiven, "excess_given"},
		{character.ValueResultExpended, "expended"},
		{character.ValueRejectionNotInspired, "not_inspired"},
		{character.RecipientIsHolder.String(), "recipient_is_holder"},
		{character.RecipientAlreadyInspired.String(), "recipient_already_inspired"},
	} {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-111, section 7 - the Topple save span name and the
// values added to the shared operation and result keys are a public contract: their strings are
// pinned here and each appears in doc/observability/signals.md.
// Path: happy.
func TestToppleSaveSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ got, want string }{
		{character.SpanResolveToppleSave, "srd5e.character.resolve_topple_save"},
		{character.ValueOperationResolveToppleSave, "resolve_topple_save"},
		{character.ValueResultTargetProne, "target_prone"},
		{character.ValueResultTargetResisted, "target_resisted"},
	} {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
		if c.want != character.SpanResolveToppleSave && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/rules/observability.md section 5", c.want)
		}
	}
}

// Requirement: doc/specs/spellcasting.md section 7, FR-085 - the spellcasting signal names, attribute keys and
// values are a public contract: their strings are pinned here and each appears in doc/observability/signals.md.
// The rejection values that come from enum String methods are pinned too.
// Path: happy.
func TestSpellcastingSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	pairs := []struct{ got, want string }{
		{character.SpanCalculateSpellcasting, "srd5e.character.calculate_spellcasting"},
		{character.SpanCreateSpellcasting, "srd5e.character.create_spellcasting"},
		{character.SpanChooseAdditionalSpells, "srd5e.character.choose_additional_spells"},
		{character.SpanChangePreparedSpells, "srd5e.character.change_prepared_spells"},
		{character.SpanExpendSpellSlot, "srd5e.character.expend_spell_slot"},
		{character.SpanRegainSpellSlot, "srd5e.character.regain_spell_slot"},
		{character.SpanRestoreSpellSlots, "srd5e.character.restore_spell_slots"},
		{character.SpanAdvanceSpellcasting, "srd5e.character.advance_spellcasting"},
		{character.MetricPreparedSpellChanges, "srd5e.character.prepared_spell_changes"},
		{character.MetricSpellSlotsRestored, "srd5e.character.spell_slots_restored"},
		{character.MetricSpellSlotLevel, "srd5e.character.spell_slot_level"},
		{character.AttrPool, "srd5e.character.pool"},
		{character.ValueOperationCalculateSpellcasting, "calculate_spellcasting"},
		{character.ValueOperationCreateSpellcasting, "create_spellcasting"},
		{character.ValueOperationChooseAdditionalSpells, "choose_additional_spells"},
		{character.ValueOperationChangePreparedSpells, "change_prepared_spells"},
		{character.ValueOperationExpendSpellSlot, "expend_spell_slot"},
		{character.ValueOperationRegainSpellSlot, "regain_spell_slot"},
		{character.ValueOperationRestoreSpellSlots, "restore_spell_slots"},
		{character.ValueOperationAdvanceSpellcasting, "advance_spellcasting"},
		{character.ValueResultCalculated, "calculated"},
		{character.ValueResultCreated, "created"},
		{character.ValueResultRegained, "regained"},
		{character.ValueResultRestored, "restored"},
		{character.ValueResultAdvanced, "advanced"},
		{character.ValuePoolSpellcasting, character.PoolSpellcasting.String()},
		{character.ValuePoolPactMagic, character.PoolPactMagic.String()},
		{character.ValuePoolSpellcasting, "spellcasting"},
		{character.ValuePoolPactMagic, "pact_magic"},
		{character.SlotOperationNoSlotAvailable.String(), "no_slot_available"},
		{character.SlotOperationNoExpendedSlot.String(), "no_expended_slot"},
		{character.SpellcastingAdvanceLevelLowered.String(), "level_lowered"},
		{character.SpellcastingAdvanceClassRemoved.String(), "class_removed"},
		{character.SpellcastingAdvanceLevelsSkipped.String(), "levels_skipped"},
		{character.SpellcastingAdvanceNoChange.String(), "no_change"},
		{character.PreparationNotOnClassList.String(), "not_on_class_list"},
		{character.PreparationSpellLevelTooHigh.String(), "spell_level_too_high"},
		{character.PreparationTooManyPrepared.String(), "too_many_prepared"},
		{character.PreparationDuplicateSpell.String(), "duplicate_spell"},
		{character.PreparationCantripNotPreparable.String(), "cantrip_not_preparable"},
		{character.PreparationAlwaysPreparedNotReplaceable.String(), "always_prepared_not_replaceable"},
		{character.PreparationChangeNotAllowedNow.String(), "change_not_allowed_now"},
		{character.PreparationTooManyReplacements.String(), "too_many_replacements"},
		{character.PreparationNotInSpellbook.String(), "not_in_spellbook"},
		{character.PreparationTooManyCantrips.String(), "too_many_cantrips"},
		{character.PreparationNotACantrip.String(), "not_a_cantrip"},
	}
	for _, c := range pairs {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
	}
}
