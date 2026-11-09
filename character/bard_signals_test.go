package character_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-bard.md section 7 - the Bard span names and attribute values are a public contract: their strings are pinned here, each span and value appears in doc/observability/signals.md or doc/rules/observability.md, and every rejection value
// (the String of BardReason) is in doc/rules/observability.md.
// Path: happy.
func TestBardSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanUseBardicInspiration, "srd5e.character.bard.use_bardic_inspiration"},
		{character.SpanUseCuttingWords, "srd5e.character.bard.use_cutting_words"},
		{character.SpanUsePeerlessSkill, "srd5e.character.bard.use_peerless_skill"},
		{character.SpanUseCountercharm, "srd5e.character.bard.use_countercharm"},
		{character.ValueClassBard, "bard"},
		{character.ValueFeatureBardicInspiration, "bardic_inspiration"},
		{character.ValueFeatureCuttingWords, "cutting_words"},
		{character.ValueFeaturePeerlessSkill, "peerless_skill"},
		{character.ValueFeatureCountercharm, "countercharm"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.BardNotABard; r.Valid(); r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
}

func bardWant(span, feature string, extra func(*wantCall)) wantCall {
	w := wantCall{span: span, feature: feature, class: character.ValueClassBard, status: telemetry.StatusUnset, outcome: "success", logged: true}
	if extra != nil {
		extra(&w)
	}
	return w
}

func rejectedWith(reason string) func(*wantCall) {
	return func(w *wantCall) { w.outcome, w.rejection = "failure", reason }
}

// Requirement: doc/specs/class-bard.md section 7 - a successful call records one class-qualified span (class bard, feature, outcome success), one feature_used increment, one duration in seconds and one FINE log; a rule rejection records outcome failure with the
// reason, no error status, no error.type and no errors increment; a Peerless Skill that still fails is an applied result, so success.
// Path: per case (happy / unhappy: state error).
func TestBardUseCaseSignalsSuccessAndRejection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	failedCheck := brdAbility(t, 5, 0, 15)
	passedCheck := character.CuttingWordsRoll{Result: brdAbility(t, 16, 0, 15)}
	for _, c := range []struct {
		name string
		run  func(opts []character.Option) error
		want wantCall
	}{
		{"Bardic Inspiration", func(o []character.Option) error {
			_, err := useInspiration(t, dicetest.NewFixedRoller(4), o...).Run(ctx, inspirationCmd(t, character.BardicD6, 0, failedCheck))
			return err
		}, bardWant(character.SpanUseBardicInspiration, "bardic_inspiration", nil)},
		{"Bardic Inspiration expired", func(o []character.Option) error {
			_, err := useInspiration(t, dicetest.NewFixedRoller(4), o...).Run(ctx, inspirationCmd(t, character.BardicD6, 2*time.Hour, failedCheck))
			return err
		}, bardWant(character.SpanUseBardicInspiration, "bardic_inspiration", rejectedWith("expired"))},
		{"Bardic Inspiration not failed", func(o []character.Option) error {
			_, err := useInspiration(t, dicetest.NewFixedRoller(4), o...).Run(ctx, inspirationCmd(t, character.BardicD6, 0, brdAbility(t, 18, 0, 15)))
			return err
		}, bardWant(character.SpanUseBardicInspiration, "bardic_inspiration", rejectedWith("not_a_failure"))},
		{"Cutting Words", func(o []character.Option) error {
			_, err := useCutting(t, dicetest.NewFixedRoller(2), o...).Run(ctx, cuttingCmd(brdBard(t, 5, 16), passedCheck))
			return err
		}, bardWant(character.SpanUseCuttingWords, "cutting_words", nil)},
		{"Cutting Words no uses", func(o []character.Option) error {
			_, err := useCutting(t, dicetest.NewFixedRoller(2), o...).Run(ctx, cuttingCmd(brdWithPool(t, brdBard(t, 5, 16), 3, 0), passedCheck))
			return err
		}, bardWant(character.SpanUseCuttingWords, "cutting_words", rejectedWith("no_uses_remaining"))},
		{"Cutting Words not a Bard", func(o []character.Option) error {
			_, err := useCutting(t, dicetest.NewFixedRoller(2), o...).Run(ctx, cuttingCmd(fgtFighter(t, 5), passedCheck))
			return err
		}, bardWant(character.SpanUseCuttingWords, "cutting_words", rejectedWith("not_a_bard"))},
		{"Peerless Skill still failing", func(o []character.Option) error {
			_, err := usePeerless(t, dicetest.NewFixedRoller(1), o...).Run(ctx, peerlessCmd(brdBard(t, 14, 16), failedCheck))
			return err
		}, bardWant(character.SpanUsePeerlessSkill, "peerless_skill", nil)},
		{"Peerless Skill below level", func(o []character.Option) error {
			_, err := usePeerless(t, dicetest.NewFixedRoller(1), o...).Run(ctx, peerlessCmd(brdBard(t, 13, 16), failedCheck))
			return err
		}, bardWant(character.SpanUsePeerlessSkill, "peerless_skill", rejectedWith("feature_unavailable"))},
		{"Countercharm", func(o []character.Option) error {
			_, err := useCountercharm(t, dicetest.NewFixedRoller(4, 9), o...).Run(ctx, countercharmCmd(brdBard(t, 7, 16)))
			return err
		}, bardWant(character.SpanUseCountercharm, "countercharm", nil)},
		{"Countercharm ineligible effect", func(o []character.Option) error {
			cmd := countercharmCmd(brdBard(t, 7, 16))
			cmd.Conditions = []content.ConditionKind{content.ConditionPoisoned}
			_, err := useCountercharm(t, dicetest.NewFixedRoller(4, 9), o...).Run(ctx, cmd)
			return err
		}, bardWant(character.SpanUseCountercharm, "countercharm", rejectedWith("ineligible_effect"))},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			if err := c.run(opts); err != nil {
				t.Fatal(err)
			}
			checkFeatureCall(t, rec, c.want)
		})
	}
}

// Requirement: doc/specs/class-bard.md section 7, FR-036, FR-101 - a *srderr.PortError from the dice roller is recorded once: error status and error.type on the span, one feature_used increment with outcome failure and no error.type, one errors increment
// with error.type dice_roller_error, one duration, no log; an invalid command records the error on the span only, with no counter increment.
// Path: per case (unhappy: collaborator failure, invalid input).
func TestBardUseCaseSignalsFailures(t *testing.T) {
	t.Parallel()
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, errBoom })
	failure := func(span, feature string) wantCall {
		return bardWant(span, feature, func(w *wantCall) {
			w.status, w.outcome, w.logged, w.errorType = telemetry.StatusError, "failure", false, "dice_roller_error"
		})
	}
	for _, c := range []struct {
		name string
		run  func(opts []character.Option) (any, error)
		want wantCall
	}{
		{"Bardic Inspiration", func(o []character.Option) (any, error) {
			return useInspiration(t, failing, o...).Run(context.Background(), inspirationCmd(t, character.BardicD6, 0, brdAbility(t, 5, 0, 15)))
		}, failure(character.SpanUseBardicInspiration, "bardic_inspiration")},
		{"Cutting Words", func(o []character.Option) (any, error) {
			return useCutting(t, failing, o...).Run(context.Background(), cuttingCmd(brdBard(t, 5, 16), character.CuttingWordsDamage{Total: 5}))
		}, failure(character.SpanUseCuttingWords, "cutting_words")},
		{"Peerless Skill", func(o []character.Option) (any, error) {
			return usePeerless(t, failing, o...).Run(context.Background(), peerlessCmd(brdBard(t, 14, 16), brdAbility(t, 5, 0, 15)))
		}, failure(character.SpanUsePeerlessSkill, "peerless_skill")},
		{"Countercharm", func(o []character.Option) (any, error) {
			return useCountercharm(t, failing, o...).Run(context.Background(), countercharmCmd(brdBard(t, 7, 16)))
		}, failure(character.SpanUseCountercharm, "countercharm")},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			if res, err := c.run(opts); res == nil && err == nil || err == nil {
				t.Fatalf("= %v, %v", res, err)
			}
			checkFeatureCall(t, rec, c.want)
		})
	}
	t.Run("an invalid command", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		noPool := restored(t, 355000, 200, flatScores(t, 10), cl("bard", 5))
		if _, err := useCutting(t, dicetest.NewFixedRoller(1), opts...).Run(context.Background(), cuttingCmd(noPool, character.CuttingWordsDamage{Total: 5})); err == nil {
			t.Fatal("no error")
		}
		spans := rec.Tracer.Spans()
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 ||
			len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 || len(rec.Logger.Records()) != 0 {
			t.Errorf("spans %+v, counters %v", spans, rec.Meter.MeasurementsNamed(character.MetricFeatureUsed))
		}
	})
}
