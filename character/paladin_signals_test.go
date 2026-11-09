package character_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-paladin.md section 7, FR-119 - the Paladin span names and attribute values are a public contract: their strings are pinned here, each span and value appears in doc/observability/signals.md or
// doc/rules/observability.md, and every rejection value (the String of PaladinReason) is in doc/rules/observability.md.
// Path: happy.
func TestPaladinSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanUseLayOnHands, "srd5e.character.paladin.use_lay_on_hands"},
		{character.SpanUsePaladinChannelDivinity, "srd5e.character.paladin.use_channel_divinity"},
		{character.SpanUseFreeCast, "srd5e.character.paladin.use_free_cast"},
		{character.SpanUseHolyNimbus, "srd5e.character.paladin.use_holy_nimbus"},
		{character.SpanApplySmiteOfProtection, "srd5e.character.paladin.apply_smite_of_protection"},
		{character.SpanChoosePaladinOption, "srd5e.character.paladin.choose_option"},
		{character.SpanCalculatePaladinFeatures, "srd5e.character.paladin.calculate_features"},
		{character.ValueClassPaladin, "paladin"},
		{character.ValueFeatureLayOnHands, "lay_on_hands"},
		{character.ValueFeatureDivineSense, "divine_sense"},
		{character.ValueFeatureAbjureFoes, "abjure_foes"},
		{character.ValueFeatureSacredWeapon, "sacred_weapon"},
		{character.ValueFeaturePaladinsSmite, "paladins_smite"},
		{character.ValueFeatureFaithfulSteed, "faithful_steed"},
		{character.ValueFeatureHolyNimbus, "holy_nimbus"},
		{character.ValueFeatureSmiteOfProtection, "smite_of_protection"},
		{character.ValueFeatureChooseOption, "choose_option"},
		{character.ValueFeatureCalculateFeatures, "calculate_features"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.PaladinNotAPaladin; r.Valid(); r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-110, FR-115, FR-117, FR-118 - a Paladin use case whose Reasons are several records the first one as the rejection, once, with outcome failure, no error status, no error.type and no errors
// increment; a returned error for an invalid command records the error on the span only.
// Path: per case (unhappy: telemetry on failure).
func TestPaladinUseCaseSignalsOnFailure(t *testing.T) {
	t.Parallel()
	rec, opts := fgtTelemetry()
	cmd := character.UseLayOnHandsCommand{Character: pldPaladin(t, 13), HitPoints: -1, Remove: []content.ConditionKind{cProne, cBlinded}}
	res, err := useLayOnHands(t, opts...).Run(context.Background(), cmd)
	if rj, ok := res.(character.UseLayOnHandsRejected); err != nil || !ok || len(rj.Reasons()) != 3 {
		t.Fatalf("result = %#v, %v", res, err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUseLayOnHands, "lay_on_hands", "invalid_amount"))
	if len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 {
		t.Error("a rule rejection incremented the errors counter")
	}
	rec, opts = fgtTelemetry()
	_, _ = useLayOnHands(t, opts...).Run(context.Background(), character.UseLayOnHandsCommand{Character: pldForeign(t, pldPaladin(t, 5)), HitPoints: 1})
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 {
		t.Errorf("a foreign state recorded %+v", spans)
	}
}
