package character_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: doc/specs/feats-behaviour.md section 7, FR-073 - the feat signal names, attribute keys and values are a public contract: their strings are pinned here, each appears in doc/observability/signals.md and the attribute values in doc/rules/observability.md; the rejection values are the String of each FeatRejection.
// Path: happy.
func TestFeatSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanTakeFeat, "srd5e.character.take_feat"},
		{character.SpanChangeMagicInitiateSpell, "srd5e.character.change_magic_initiate_spell"},
		{character.SpanExpendFeatUse, "srd5e.character.expend_feat_use"},
		{character.SpanRenewFeatUses, "srd5e.character.renew_feat_uses"},
		{character.SpanRollFreeCasting, "srd5e.character.roll_free_casting"},
		{character.AttrFeatCategory, "srd5e.character.feat_category"},
		{character.AttrFeatSource, "srd5e.character.feat_source"},
		{character.ValueOperationTakeFeat, "take_feat"},
		{character.ValueOperationChangeMagicInitiateSpell, "change_magic_initiate_spell"},
		{character.ValueOperationExpendFeatUse, "expend_feat_use"},
		{character.ValueOperationRenewFeatUses, "renew_feat_uses"},
		{character.ValueOperationRollFreeCasting, "roll_free_casting"},
		{character.ValueResultTaken, "taken"},
		{character.ValueResultNotApplicable, "not_applicable"},
		{character.ValueFeatCategoryOrigin, "origin"},
		{character.ValueFeatCategoryGeneral, "general"},
		{character.ValueFeatCategoryFightingStyle, "fighting_style"},
		{character.ValueFeatCategoryEpicBoon, "epic_boon"},
		{character.ValueFeatSourceBackground, "background"},
		{character.ValueFeatSourceSpeciesTrait, "species_trait"},
		{character.ValueFeatSourceClassFeature, "class_feature"},
		{character.ValueFeatSourceEldritchInvocation, "eldritch_invocation"},
		{character.ValueFeatSourceBonusFeat, "bonus_feat"},
		{character.FeatBehaviourUnknown{}.String(), "feat_behaviour_unknown"},
		{character.FeatNotReplaceable{}.String(), "feat_not_replaceable"},
	}
	for _, p := range pins {
		if p.got != p.want {
			t.Errorf("constant = %q, want %q", p.got, p.want)
		}
		if !strings.Contains(string(rules), "`"+p.want+"`") && !strings.HasPrefix(p.want, "srd5e.") {
			t.Errorf("observability.md does not list %q", p.want)
		}
		if !strings.Contains(string(catalogue), "`"+p.want+"`") && strings.HasPrefix(p.want, "srd5e.") {
			t.Errorf("signals.md does not list %q", p.want)
		}
	}
	for _, p := range []string{"srd5e.character.feat_category", "srd5e.character.feat_source"} {
		if !strings.Contains(string(catalogue), "`"+p+"`") || !strings.Contains(string(rules), "`"+p+"`") {
			t.Errorf("%q is not catalogued", p)
		}
	}
	for _, s := range []string{"feat_prerequisite_not_met", "feat_choices_invalid", "feat_increase_rejected", "spell_change_not_allowed_now", "spell_change_already_used", "feat_use_already_expended", "feat_not_held", "feat_has_no_limited_use", "feat_outside_grant", "feat_not_repeatable"} {
		if !strings.Contains(string(rules), "`"+s+"`") || !strings.Contains(string(catalogue), "`"+s+"`") {
			t.Errorf("rejection %q is not catalogued", s)
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-081, FR-073 - TakeFeat records only its own signals: an Ability Score Improvement that raises Constitution yields the take_feat span with its load_content child and one operations counter increment of operation take_feat, none of ChangeAbilityScore, and the Hit Point maximum change travels as an event.
// Path: unhappy (nested telemetry).
func TestTakeFeatRecordsOnlyItsOwnSignals(t *testing.T) {
	t.Parallel()
	c := charAt(t, 8, 10, 10, 17, 10, 10, 10)
	choices := must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, 3, 0))
	uc, rec := recordedTake(t, featSource(t))
	res, err := uc.Run(context.Background(), takeCmd(c, "ability_score_improvement", anyGrant(), choices))
	ok, isOK := res.(character.TakeFeatSucceeded)
	if err != nil || !isOK || len(ok.Events()) != 3 {
		t.Fatalf("= %#v, %v", res, err)
	}
	for _, e := range ok.Events()[1:] {
		if e.CharacterID() != c.ID() {
			t.Errorf("%T", e)
		}
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 2 || spans[0].Name != character.SpanTakeFeat || spans[1].Name != character.SpanLoadContent {
		t.Fatalf("spans = %+v", spans)
	}
	if got := rec.Meter.MeasurementsNamed(character.MetricOperations); len(got) != 1 {
		t.Errorf("%d operations measurements, want 1", len(got))
	}
	telemetrytest.RequireNoViolations(t, rec)
}
