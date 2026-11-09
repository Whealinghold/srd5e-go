package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// wantCall is what one class feature call must have recorded.
type wantCall struct {
	span, feature      string
	status             telemetry.Status
	outcome, rejection string
	logged             bool
	errorType          string
	// class is the srd5e.class value; the Fighter when empty.
	class string
}

// checkFeatureCall asserts the signals of exactly one call: one span with the class, feature, outcome and rejection, one feature_used increment (with
// no error.type), one duration in seconds, the errors counter once with error.type or not at all, and the FINE log once or not at all.
func checkFeatureCall(t *testing.T, rec *telemetrytest.Recorder, w wantCall) {
	t.Helper()
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Name != w.span || !spans[0].Ended || spans[0].Status != w.status {
		t.Fatalf("spans = %+v, want one ended %s with status %v", spans, w.span, w.status)
	}
	if w.class == "" {
		w.class = character.ValueClassFighter
	}
	ident := map[string]any{character.AttrClass: w.class, character.AttrFeature: w.feature}
	call := map[string]any{telemetry.AttrOutcome: w.outcome}
	for k, v := range ident {
		call[k] = v
	}
	if w.rejection != "" {
		call[character.AttrRejection] = w.rejection
	}
	if w.status == telemetry.StatusError && w.errorType != "" && len(spans[0].Errors) != 1 {
		t.Errorf("span errors = %v, want one recorded error", spans[0].Errors)
	}
	sameAttrs(t, "span", spans[0].Attrs, call)
	used := onlyOne(t, rec, character.MetricFeatureUsed)
	sameAttrs(t, "feature_used", used.Attrs, call)
	if used.Value != 1 {
		t.Errorf("feature_used value = %v, want 1", used.Value)
	}
	dur := onlyOne(t, rec, character.MetricFeatureUsedDuration)
	sameAttrs(t, "duration", dur.Attrs, ident)
	if dur.Value != 1 {
		t.Errorf("duration = %v s, want 1 from the stepping clock", dur.Value)
	}
	errs := rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)
	if w.errorType == "" {
		if len(errs) != 0 {
			t.Errorf("errors counter = %+v, want none", errs)
		}
	} else {
		if len(errs) != 1 {
			t.Fatalf("errors counter = %+v, want one", errs)
		}
		want := map[string]any{telemetry.AttrErrorType: w.errorType}
		for k, v := range ident {
			want[k] = v
		}
		sameAttrs(t, "errors", errs[0].Attrs, want)
	}
	logs := rec.Logger.Records()
	if w.logged {
		if len(logs) != 1 || logs[0].Level != telemetry.LevelDebug || logs[0].Message != character.LogClassFeatureResolved {
			t.Fatalf("logs = %+v, want one FINE %s", logs, character.LogClassFeatureResolved)
		}
		sameAttrs(t, "log", logs[0].Attrs, map[string]any{character.AttrClass: w.class, character.AttrFeature: w.feature, telemetry.AttrOutcome: w.outcome})
	} else if len(logs) != 0 {
		t.Errorf("logged %+v, want nothing", logs)
	}
	telemetrytest.RequireNoViolations(t, rec)
}

// Requirement: doc/specs/class-fighter.md FR-138, FR-139, FR-140, FR-142 - a successful call records one span (class, feature, outcome success), one feature_used increment, one duration
// in seconds with the class and feature only and one FINE log; no rule rejection sets error status; a rejection records outcome failure with the first reason and one FINE log.
// Path: per case (happy / unhappy: state error).
func TestFeatureSignalsSuccessAndRejection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("Second Wind applied", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if _, err := useSecondWind(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, character.UseSecondWindCommand{Character: fgtFighter(t, 5)}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseSecondWind, feature: "second_wind", status: telemetry.StatusUnset, outcome: "success", logged: true})
	})
	t.Run("Second Wind with no uses", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		c := fgtWith(t, fgtFighter(t, 1), fgtState(2, 0, 0))
		if _, err := useSecondWind(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, character.UseSecondWindCommand{Character: c}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseSecondWind, feature: "second_wind", status: telemetry.StatusUnset, outcome: "failure", rejection: "no_uses_remaining", logged: true})
	})
	t.Run("Action Surge on another turn", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if _, err := useActionSurge(t, opts...).Run(ctx, character.UseActionSurgeCommand{Character: fgtFighter(t, 5), Turn: character.FighterOtherTurn, PerTurn: character.FighterActionSurgeUsed}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseActionSurge, feature: "action_surge", status: telemetry.StatusUnset, outcome: "failure", rejection: "not_own_turn", logged: true})
	})
	t.Run("Action Surge applied", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if _, err := useActionSurge(t, opts...).Run(ctx, character.UseActionSurgeCommand{Character: fgtFighter(t, 5), Turn: character.FighterOwnTurn, PerTurn: character.FighterActionSurgeUnused}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseActionSurge, feature: "action_surge", status: telemetry.StatusUnset, outcome: "success", logged: true})
	})
	t.Run("a character that is not a Fighter", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if _, err := useSecondWind(t, dicetest.NewFixedRoller(4), opts...).Run(ctx, character.UseSecondWindCommand{Character: fgtChar(t, cl("wizard", 5))}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseSecondWind, feature: "second_wind", status: telemetry.StatusUnset, outcome: "failure", rejection: "not_a_fighter", logged: true})
	})
}

// Requirement: doc/specs/class-fighter.md FR-142 - Tactical Mind and Indomitable that still fail are normal rule outcomes: no error status, one FINE log, outcome failure and no rejection
// attribute; the successful ones record outcome success.
// Path: per case (unhappy: boundary at the DC).
func TestFeatureSignalsStillFailed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range []struct {
		name    string
		roll    int
		outcome string
		rej     string
	}{{"Tactical Mind still failed", 2, "failure", ""}, {"Tactical Mind succeeded", 3, "success", ""}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			if _, err := useTacticalMind(t, dicetest.NewFixedRoller(c.roll), opts...).Run(ctx, character.UseTacticalMindCommand{Character: fgtFighter(t, 2), FailedCheck: failedCheck(t, 12)}); err != nil {
				t.Fatal(err)
			}
			checkFeatureCall(t, rec, wantCall{span: character.SpanUseTacticalMind, feature: "tactical_mind", status: telemetry.StatusUnset, outcome: c.outcome, rejection: c.rej, logged: true})
		})
	}
	save := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{8}, 4, 15)
	for _, c := range []struct {
		name    string
		roll    int
		outcome string
	}{{"Indomitable still failed", 1, "failure"}, {"Indomitable succeeded", 2, "success"}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			if _, err := useIndomitable(t, dicetest.NewFixedRoller(c.roll), opts...).Run(ctx, character.UseIndomitableCommand{Character: fgtFighter(t, 9), FailedSave: save}); err != nil {
				t.Fatal(err)
			}
			checkFeatureCall(t, rec, wantCall{span: character.SpanUseIndomitable, feature: "indomitable", status: telemetry.StatusUnset, outcome: c.outcome, logged: true})
		})
	}
	t.Run("Tactical Mind rejected", func(t *testing.T) {
		t.Parallel()
		rec, opts := fgtTelemetry()
		if _, err := useTacticalMind(t, dicetest.NewFixedRoller(3), opts...).Run(ctx, character.UseTacticalMindCommand{Character: fgtFighter(t, 1), FailedCheck: failedCheck(t, 12)}); err != nil {
			t.Fatal(err)
		}
		checkFeatureCall(t, rec, wantCall{span: character.SpanUseTacticalMind, feature: "tactical_mind", status: telemetry.StatusUnset, outcome: "failure", rejection: "feature_not_gained", logged: true})
	})
}

// Requirement: doc/specs/class-fighter.md FR-139, FR-141 - a roller failure sets the span error status with the recorded error, increments feature_used once with outcome failure and no error.type
// or rejection, increments feature_used.errors exactly once with error.type dice_roller_error, and logs nothing.
// Path: per case (unhappy: collaborator failure).
func TestFeatureSignalsRollerFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range []struct {
		name string
		run  func(*testing.T, []character.Option) error
		span string
		feat string
	}{
		{"Second Wind", func(t *testing.T, o []character.Option) error {
			t.Helper()
			_, err := useSecondWind(t, dicetest.NewFixedRoller(), o...).Run(ctx, character.UseSecondWindCommand{Character: fgtFighter(t, 5)})
			return err
		}, character.SpanUseSecondWind, "second_wind"},
		{"Tactical Mind", func(t *testing.T, o []character.Option) error {
			t.Helper()
			_, err := useTacticalMind(t, dicetest.NewFixedRoller(), o...).Run(ctx, character.UseTacticalMindCommand{Character: fgtFighter(t, 5), FailedCheck: failedCheck(t, 5)})
			return err
		}, character.SpanUseTacticalMind, "tactical_mind"},
		{"Indomitable", func(t *testing.T, o []character.Option) error {
			t.Helper()
			_, err := useIndomitable(t, dicetest.NewFixedRoller(), o...).Run(ctx, character.UseIndomitableCommand{Character: fgtFighter(t, 9), FailedSave: fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)})
			return err
		}, character.SpanUseIndomitable, "indomitable"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			if err := c.run(t, opts); err == nil {
				t.Fatal("want the roller failure")
			}
			checkFeatureCall(t, rec, wantCall{span: c.span, feature: c.feat, status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-138, FR-140, FR-141, FR-142, FR-143 - ApplyFighterOptionChoice records feature fighting_style or weapon_mastery by option; its span and duration cover
// the lookups; content-missing sets error status, records outcome failure with the rejection content_missing, no error.type and no log (the duration is still recorded); a source
// failure sets error status, records the error once with error.type content_error and logs nothing; a rejection with several reasons records the first one as the single
// rejection attribute.
// Path: per case (happy / unhappy: missing content, collaborator failure, rule interaction).
func TestFeatureSignalsApplyFighterOption(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	run := func(t *testing.T, src character.ContentSource, c character.Character, choice character.FighterOptionChoice, w wantCall) {
		t.Helper()
		rec, opts := fgtTelemetry()
		_, _ = applyUseCase(t, src, opts...).Run(ctx, character.ApplyFighterOptionChoiceCommand{Character: c, Choice: choice})
		w.span = character.SpanApplyFighterOption
		checkFeatureCall(t, rec, w)
	}
	style := character.FighterChooseFightingStyle{Choice: styleChoice("archery")}
	masteries := character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}
	t.Run("a style applied", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t), fgtFighter(t, 1), style, wantCall{feature: "fighting_style", outcome: "success", logged: true})
	})
	t.Run("masteries applied", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t), fgtFighter(t, 1), masteries, wantCall{feature: "weapon_mastery", outcome: "success", logged: true})
	})
	t.Run("a swap with a missing weapon", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t), fgtFighter(t, 1), character.FighterSwapWeaponMastery{Remove: "mace", Add: "aklys"}, wantCall{feature: "weapon_mastery", status: telemetry.StatusError, outcome: "failure", rejection: "content_missing"})
	})
	t.Run("a missing feat", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t), fgtFighter(t, 1), character.FighterChooseFightingStyle{Choice: styleChoice("no_such_style")}, wantCall{feature: "fighting_style", status: telemetry.StatusError, outcome: "failure", rejection: "content_missing"})
	})
	t.Run("a source failure", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t).FailLookup(errBoom), fgtFighter(t, 1), style, wantCall{feature: "fighting_style", status: telemetry.StatusError, outcome: "failure", errorType: "content_error"})
	})
	t.Run("a contract violation of the source", func(t *testing.T) {
		t.Parallel()
		run(t, fgtWrongSource{w: weaponFixtures(t)["mace"]}, fgtFighter(t, 1), style, wantCall{feature: "fighting_style", status: telemetry.StatusError, outcome: "failure", errorType: "content_error"})
	})
	t.Run("the first of several reasons", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t), fgtFighter(t, 1), character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "mace", "mace", "club")},
			wantCall{feature: "weapon_mastery", outcome: "failure", rejection: "wrong_mastery_count", logged: true})
	})
	t.Run("not a Fighter", func(t *testing.T) {
		t.Parallel()
		run(t, fgtSource(t), fgtChar(t, cl("wizard", 4)), style, wantCall{feature: "fighting_style", outcome: "failure", rejection: "not_a_fighter", logged: true})
	})
}

// Requirement: doc/specs/class-fighter.md FR-141 - a stored state of another type under a Fighter key (or the shared Weapon Mastery key) is a command error recorded on the span only
// (error status and the recorded error): no feature_used increment, no errors increment and no log; the duration is still recorded.
// Path: per case (unhappy: state error).
func TestFeatureSignalsCommandError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	foreign := func(name string, level int) character.Character {
		return must(character.WithFeatureState(fgtFighter(t, level), character.MustFeatureStateKey[character.ChosenSubclass](name), must(character.NewChosenSubclass("champion"))))
	}
	save := fgtCheck(t, d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)
	for _, c := range []struct {
		name string
		run  func(opts []character.Option) error
	}{
		{"Second Wind", func(o []character.Option) error {
			_, err := useSecondWind(t, dicetest.NewFixedRoller(4), o...).Run(ctx, character.UseSecondWindCommand{Character: foreign("fighter", 5)})
			return err
		}},
		{"Tactical Mind", func(o []character.Option) error {
			_, err := useTacticalMind(t, dicetest.NewFixedRoller(4), o...).Run(ctx, character.UseTacticalMindCommand{Character: foreign("fighter", 5), FailedCheck: failedCheck(t, 5)})
			return err
		}},
		{"Indomitable", func(o []character.Option) error {
			_, err := useIndomitable(t, dicetest.NewFixedRoller(4), o...).Run(ctx, character.UseIndomitableCommand{Character: foreign("fighter", 9), FailedSave: save})
			return err
		}},
		{"Action Surge", func(o []character.Option) error {
			_, err := useActionSurge(t, o...).Run(ctx, character.UseActionSurgeCommand{Character: foreign("fighter", 5), Turn: character.FighterOwnTurn, PerTurn: character.FighterActionSurgeUnused})
			return err
		}},
		{"a foreign Fighter state in a choice", func(o []character.Option) error {
			_, err := applyUseCase(t, fgtSource(t), o...).Run(ctx, character.ApplyFighterOptionChoiceCommand{Character: foreign("fighter", 1), Choice: character.FighterChooseFightingStyle{Choice: styleChoice("archery")}})
			return err
		}},
		{"a foreign Weapon Mastery state in a choice", func(o []character.Option) error {
			_, err := applyUseCase(t, fgtSource(t), o...).Run(ctx, character.ApplyFighterOptionChoiceCommand{Character: foreign("weapon_mastery_choice", 1), Choice: character.FighterChooseWeaponMasteries{Kinds: ids("greatsword", "longbow", "mace")}})
			return err
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := fgtTelemetry()
			wantInvalid(t, c.run(opts), c.name)
			spans := rec.Tracer.Spans()
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with the error recorded", spans)
			}
			if n := len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) + len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)); n != 0 {
				t.Errorf("%d counter increments, want none", n)
			}
			onlyOne(t, rec, character.MetricFeatureUsedDuration)
			noLogsOrViolations(t, rec)
		})
	}
}
