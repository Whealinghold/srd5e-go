package character_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-cleric.md section 7, FR-124 - the Cleric span names and attribute values are a public contract: their strings are pinned here, each span and value appears in
// doc/observability/signals.md or doc/rules/observability.md, and every rejection value (the String of ClericReason) is in doc/rules/observability.md.
// Path: happy.
func TestClericSignalNamesArePinnedAndCatalogued(t *testing.T) {
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
		{character.SpanUseClericChannelDivinity, "srd5e.character.cleric.use_channel_divinity"},
		{character.SpanUseDivineIntervention, "srd5e.character.cleric.use_divine_intervention"},
		{character.ValueClassCleric, "cleric"},
		{character.ValueFeatureDivineSpark, "divine_spark"},
		{character.ValueFeatureTurnUndead, "turn_undead"},
		{character.ValueFeaturePreserveLife, "preserve_life"},
		{character.ValueFeatureDivineIntervention, "divine_intervention"},
	} {
		if c.got != c.want {
			t.Errorf("constant = %q, want %q", c.got, c.want)
		}
		if !strings.Contains(string(catalogue), "`"+c.want+"`") && !strings.Contains(string(rules), "`"+c.want+"`") {
			t.Errorf("%q is in neither signals.md nor observability.md", c.want)
		}
	}
	for r := character.ClericNotACleric; r.Valid(); r++ {
		if !strings.Contains(string(rules), "`"+r.String()+"`") {
			t.Errorf("rejection value %q is not in observability.md", r.String())
		}
	}
}

func channelWant(feature string, extra func(*wantCall)) wantCall {
	w := wantCall{span: character.SpanUseClericChannelDivinity, feature: feature, class: character.ValueClassCleric, status: telemetry.StatusUnset, outcome: "success", logged: true}
	if extra != nil {
		extra(&w)
	}
	return w
}

// Requirement: doc/specs/class-cleric.md FR-124 - a successful call records one class-qualified span (class cleric, feature, outcome success), one feature_used increment, one duration in seconds and one FINE log; a rule
// rejection and NotACleric record outcome failure with the reason, no error status, no error.type and no errors increment.
// Path: per case (happy / unhappy: state error).
func TestClericUseCaseSignalsSuccessAndRejection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rejected := func(reason string) func(*wantCall) {
		return func(w *wantCall) { w.outcome, w.rejection = "failure", reason }
	}
	for _, c := range []struct {
		name string
		cmd  func() (character.UseClericChannelDivinityCommand, []int)
		want wantCall
	}{
		{"Divine Spark", func() (character.UseClericChannelDivinityCommand, []int) {
			return sparkCmd(clcCleric(t, 5), character.SparkHealing, ""), []int{4}
		}, channelWant("divine_spark", nil)},
		{"Turn Undead", func() (character.UseClericChannelDivinityCommand, []int) {
			return character.UseClericChannelDivinityCommand{Character: clcCleric(t, 5), Effect: character.TurnUndeadEffect{Targets: []character.UndeadTarget{undead("z", 5)}}}, nil
		}, channelWant("turn_undead", nil)},
		{"Preserve Life", func() (character.UseClericChannelDivinityCommand, []int) {
			return character.UseClericChannelDivinityCommand{Character: clcCleric(t, 5), Subclass: "life_domain", Effect: character.PreserveLifeEffect{}}, nil
		}, channelWant("preserve_life", nil)},
		{"no uses", func() (character.UseClericChannelDivinityCommand, []int) {
			return sparkCmd(clcWith(t, clcCleric(t, 5), clcUses(2, 0)), character.SparkHealing, ""), nil
		}, channelWant("divine_spark", rejected("no_uses_remaining"))},
		{"below level 2", func() (character.UseClericChannelDivinityCommand, []int) {
			return sparkCmd(clcCleric(t, 1), character.SparkHealing, ""), nil
		}, channelWant("divine_spark", rejected("below_level_2"))},
		{"not a Cleric", func() (character.UseClericChannelDivinityCommand, []int) {
			return sparkCmd(fgtFighter(t, 5), character.SparkHealing, ""), nil
		}, channelWant("divine_spark", rejected("not_a_cleric"))},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			cmd, rolls := c.cmd()
			if _, err := useChannel(t, dicetest.NewFixedRoller(rolls...), opts...).Run(ctx, cmd); err != nil {
				t.Fatal(err)
			}
			checkFeatureCall(t, rec, c.want)
		})
	}
	t.Run("Divine Intervention", func(t *testing.T) {
		t.Parallel()
		for _, k := range []struct {
			char   character.Character
			reason string
		}{{clcCleric(t, 10), ""}, {clcCleric(t, 9), "below_level_10"}, {fgtFighter(t, 10), "not_a_cleric"}} {
			rec, opts := fgtTelemetry()
			if _, err := useIntervention(t, dicetest.NewFixedRoller(), opts...).Run(ctx, interventionCmd(k.char, clcClericSpell(t, "cure_wounds", 1))); err != nil {
				t.Fatal(err)
			}
			w := wantCall{span: character.SpanUseDivineIntervention, feature: "divine_intervention", class: "cleric", status: telemetry.StatusUnset, outcome: "success", logged: true}
			if k.reason != "" {
				w.outcome, w.rejection = "failure", k.reason
			}
			checkFeatureCall(t, rec, w)
		}
	})
}

// Requirement: doc/specs/class-cleric.md FR-124, FR-130, table 4.14.4 CF-01 to CF-03 - a *srderr.PortError from the dice roller is recorded once: error status and error.type on the span, one feature_used increment with outcome
// failure and no error.type, one errors increment with error.type dice_roller_error, one duration, no log; an invalid command records the error on the span only, with no counter increment.
// Path: per case (unhappy: collaborator failure, invalid input).
func TestClericUseCaseSignalsFailures(t *testing.T) {
	t.Parallel()
	fail := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, errBoom })
	t.Run("CF-01 Divine Spark", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if res, err := useChannel(t, fail, opts...).Run(context.Background(), sparkCmd(clcCleric(t, 5), character.SparkHealing, "")); res != nil || err == nil {
			t.Fatalf("= %v, %v", res, err)
		}
		checkFeatureCall(t, rec, channelWant("divine_spark", func(w *wantCall) {
			w.status, w.outcome, w.logged, w.errorType = telemetry.StatusError, "failure", false, "dice_roller_error"
		}))
	})
	t.Run("CF-03 Wish", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if res, err := useIntervention(t, fail, opts...).Run(context.Background(), interventionCmd(clcCleric(t, 20), wishSpell(t))); res != nil || err == nil {
			t.Fatalf("= %v, %v", res, err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseDivineIntervention, feature: "divine_intervention", class: "cleric", status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	})
	t.Run("an invalid command", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		foreign := must(character.WithFeatureState(clcCleric(t, 5), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_resources"), must(character.NewChosenSubclass("life_domain"))))
		if _, err := useChannel(t, dicetest.NewFixedRoller(1), opts...).Run(context.Background(), sparkCmd(foreign, character.SparkHealing, "")); err == nil {
			t.Fatal("no error")
		}
		spans := rec.Tracer.Spans()
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 ||
			len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 || len(rec.Logger.Records()) != 0 {
			t.Errorf("spans %+v, counters %v", spans, rec.Meter.MeasurementsNamed(character.MetricFeatureUsed))
		}
	})
}
