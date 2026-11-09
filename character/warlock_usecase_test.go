package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

type warMode int

const (
	warOK warMode = iota
	warBad
	warNot
)

// warCase drives one Warlock use case in the three modes (an accepted command, one a rule refuses, a character that is not a Warlock).
type warCase struct {
	name, span, feature, rejection string
	loads                          bool
	run                            func(t *testing.T, mode warMode, src character.ContentSource, roller character.Roller, opts ...character.Option) (any, error)
}

func warFighter(t *testing.T) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), cl("fighter", 4))
}

func warCases() []warCase {
	return append([]warCase{
		{name: "ChooseWarlockInvocations", span: character.SpanChooseWarlockInvocations, feature: "eldritch_invocations", rejection: "invocation_selection_invalid", loads: true,
			run: func(t *testing.T, mode warMode, src character.ContentSource, _ character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 3, 16, "", character.WarlockState{})
				if mode == warNot {
					c = warFighter(t)
				}
				picks := []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("agonizing_blast", "eldritch_blast", ""), warPick("devils_sight", "", "")}
				if mode == warBad {
					picks = picks[:2]
				}
				return must(character.NewChooseWarlockInvocations(src, opts...)).Run(t.Context(), character.ChooseWarlockInvocationsCommand{Character: c, Picks: picks,
					KnownCantrips: []character.CantripFacts{warCantrips()["eldritch_blast"]}})
			}},
		{name: "UseDarkOnesOwnLuck", span: character.SpanUseDarkOnesOwnLuck, feature: "dark_ones_own_luck", rejection: "roll_kind_not_allowed",
			run: func(t *testing.T, mode warMode, _ character.ContentSource, roller character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 6, 16, "fiend_patron", character.WarlockState{})
				if mode == warNot {
					c = warFighter(t)
				}
				kind := d20.SavingThrow
				if mode == warBad {
					kind = d20.AttackRoll
				}
				return must(character.NewUseDarkOnesOwnLuck(roller, opts...)).Run(t.Context(), character.UseDarkOnesOwnLuckCommand{Character: c, Charisma: warMod(16), Kind: kind})
			}},
	}, warMoreCases()...)
}

// warWant is what one Warlock call records.
type warWant struct {
	span, feature, outcome, rejection string
	status                            telemetry.Status
	logged                            bool
	errorType                         string
	loaded                            bool // a load_content child span is expected
}

// warCheck asserts the signals of exactly one Warlock call: one top-level span (and the load_content child when loaded), one feature_used
// increment, one duration, the errors counter and the one FINE log.
func warCheck(t *testing.T, rec *telemetrytest.Recorder, w warWant) {
	t.Helper()
	spans := rec.Tracer.SpansNamed(w.span)
	if len(spans) != 1 || !spans[0].Ended || spans[0].Status != w.status || spans[0].ParentID != 0 {
		t.Fatalf("spans = %+v, want one ended top-level %s with status %v", rec.Tracer.Spans(), w.span, w.status)
	}
	if load := rec.Tracer.SpansNamed(character.SpanLoadContent); w.loaded && (len(load) != 1 || load[0].ParentID != spans[0].ID) || !w.loaded && len(load) != 0 {
		t.Errorf("load_content spans = %+v, loaded %v", load, w.loaded)
	}
	ident := map[string]any{character.AttrClass: character.ValueClassWarlock, character.AttrFeature: w.feature}
	call := map[string]any{telemetry.AttrOutcome: w.outcome}
	for k, v := range ident {
		call[k] = v
	}
	if w.rejection != "" {
		call[character.AttrRejection] = w.rejection
	}
	sameAttrs(t, "span", spans[0].Attrs, call)
	used := onlyOne(t, rec, character.MetricFeatureUsed)
	sameAttrs(t, "feature_used", used.Attrs, call)
	if used.Value != 1 {
		t.Errorf("feature_used value = %v, want 1", used.Value)
	}
	sameAttrs(t, "duration", onlyOne(t, rec, character.MetricFeatureUsedDuration).Attrs, ident)
	errs := rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)
	if w.errorType == "" && len(errs) != 0 || w.errorType != "" && len(errs) != 1 {
		t.Errorf("errors counter = %+v, want error type %q", errs, w.errorType)
	}
	if w.errorType != "" {
		want := map[string]any{telemetry.AttrErrorType: w.errorType}
		for k, v := range ident {
			want[k] = v
		}
		sameAttrs(t, "errors", errs[0].Attrs, want)
	}
	logs := rec.Logger.Records()
	if w.logged != (len(logs) == 1) || len(logs) > 1 {
		t.Errorf("logs = %+v, want logged=%v", logs, w.logged)
	}
	telemetrytest.RequireNoViolations(t, rec)
}

func warTypeOf(v any) string { return reflect.TypeOf(v).Name() }

// Requirement: doc/specs/class-warlock.md FR-148 - every Warlock use case, in one call, emits one top-level span named per the spec (with the load_content child where it reads content), one feature_used increment and one duration with srd5e.class warlock and its srd5e.feature; an accepted result is outcome success; a rule refusal is outcome failure with the snake_case WarlockRejectionKind, no error.type, no error status and no errors increment; a character that is not a Warlock is outcome failure with rejection not_a_warlock, decided before any lookup. Each result variant is named with its use case.
// Path: per case (happy / unhappy: state).
func TestWarlockUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range warCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			src := warSource(t)
			rec, opts := warTelemetry()
			res, err := c.run(t, warOK, src, dicetest.NewFixedRoller(7), opts...)
			if err != nil || warTypeOf(res) != c.name+"Applied" {
				t.Fatalf("accepted = %T, %v; want %sApplied", res, err, c.name)
			}
			warCheck(t, rec, warWant{span: c.span, feature: c.feature, outcome: "success", status: telemetry.StatusUnset, logged: true, loaded: c.loads})
			rec, opts = warTelemetry()
			res, err = c.run(t, warBad, src, dicetest.NewFixedRoller(7), opts...)
			if err != nil || warTypeOf(res) != c.name+"Rejected" {
				t.Fatalf("refused = %T, %v; want %sRejected", res, err, c.name)
			}
			warCheck(t, rec, warWant{span: c.span, feature: c.feature, outcome: "failure", rejection: c.rejection, status: telemetry.StatusUnset, logged: true, loaded: c.loads})
			rec, opts = warTelemetry()
			res, err = c.run(t, warNot, src, dicetest.NewFixedRoller(7), opts...)
			if err != nil || warTypeOf(res) != c.name+"Rejected" {
				t.Fatalf("not a Warlock = %T, %v; want %sRejected", res, err, c.name)
			}
			if got := reflect.ValueOf(res).FieldByName("Rejection").Interface().(character.WarlockRejection); got.Kind != character.WarlockNotAWarlock {
				t.Errorf("rejection = %+v, want WarlockNotAWarlock", got)
			}
			warCheck(t, rec, warWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "not_a_warlock", status: telemetry.StatusUnset, logged: true})
		})
	}
}

// Requirement: doc/specs/class-warlock.md FR-042, FR-141, FR-142 - when the warlock class is absent the use case that reads content returns its own ContentMissing variant naming the class, with error status on the span, rejection content_missing and no log; a source that returns its error or answers with another entry is a *srderr.PortError for content, recorded once (span error status, one feature_used failure, one errors increment with error.type) and propagated with no result and no state change; the cause stays visible to errors.Is.
// Path: per case (unhappy: missing content, collaborator failure).
func TestWarlockUseCaseContentFailures(t *testing.T) {
	t.Parallel()
	c := warCases()[0]
	rec, opts := warTelemetry()
	res, err := c.run(t, warOK, contenttest.NewFakeSource(), nil, opts...)
	if err != nil || warTypeOf(res) != c.name+"ContentMissing" {
		t.Fatalf("missing = %T, %v; want %sContentMissing", res, err, c.name)
	}
	missing := res.(character.ChooseWarlockInvocationsContentMissing).Missing()
	if len(missing) != 1 || missing[0].Kind() != content.KindCharacterClass || missing[0].ID().String() != "warlock" {
		t.Errorf("Missing = %v", missing)
	}
	missing[0] = content.MissingContent{}
	if again := res.(character.ChooseWarlockInvocationsContentMissing).Missing(); again[0].ID().String() != "warlock" {
		t.Error("Missing must return a copy")
	}
	warCheck(t, rec, warWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "content_missing", status: telemetry.StatusError, loaded: true})
	for name, src := range map[string]character.ContentSource{"an error": contenttest.NewFakeSource().FailLookup(errBoom), "wrong entry": wrongEntrySource{entry: warSource(t)}} {
		rec, opts = warTelemetry()
		res, err = c.run(t, warOK, src, nil, opts...)
		var pe *srderr.PortError
		if res != nil || !asPortError(err, &pe) || pe.Port != content.Port {
			t.Fatalf("%s: = %v, %v; want a PortError for content", name, res, err)
		}
		warCheck(t, rec, warWant{span: c.span, feature: c.feature, outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err), loaded: true})
		if loads := rec.Tracer.SpansNamed(character.SpanLoadContent); len(loads) != 1 || loads[0].Status != telemetry.StatusError {
			t.Errorf("%s: load_content = %+v, want error status", name, loads)
		}
	}
	if _, err := c.run(t, warOK, contenttest.NewFakeSource().FailLookup(context.Canceled), nil); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled lookup must stay visible through errors.Is: %v", err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-146 - every Warlock use case gives an equal result with the no-op and a recording telemetry and repeated calls agree.
// Path: per case (unhappy: determinism).
func TestWarlockUseCasesAreDeterministic(t *testing.T) {
	t.Parallel()
	for _, c := range warCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, mode := range []warMode{warOK, warBad, warNot} {
				plain, err1 := c.run(t, mode, warSource(t), dicetest.NewFixedRoller(7))
				again, err3 := c.run(t, mode, warSource(t), dicetest.NewFixedRoller(7))
				_, opts := warTelemetry()
				recorded, err2 := c.run(t, mode, warSource(t), dicetest.NewFixedRoller(7), opts...)
				if err1 != nil || err2 != nil || err3 != nil || !reflect.DeepEqual(plain, recorded) || !reflect.DeepEqual(plain, again) {
					t.Errorf("mode %d: %#v (%v) versus %#v (%v) and %#v (%v)", mode, plain, err1, recorded, err2, again, err3)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-warlock.md FR-148, FR-149 (programmer errors) - the zero use case, a zero Character and a cancelled context are errors matching ErrInvalidInput (or ctx.Err()) before any signal; a nil source or roller is rejected by the constructor; a stored state of another type under the warlock key is an error recorded on the span only.
// Path: unhappy (invalid input).
func TestWarlockUseCaseInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec, opts := warTelemetry()
	src := warSource(t)
	c := warChar(t, 6, 16, "fiend_patron", character.WarlockState{})
	if _, err := (character.ChooseWarlockInvocations{}).Run(ctx, character.ChooseWarlockInvocationsCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero ChooseWarlockInvocations: %v", err)
	}
	if _, err := (character.UseDarkOnesOwnLuck{}).Run(ctx, character.UseDarkOnesOwnLuckCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero UseDarkOnesOwnLuck: %v", err)
	}
	if _, err := must(character.NewChooseWarlockInvocations(src, opts...)).Run(ctx, character.ChooseWarlockInvocationsCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero Character: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := must(character.NewUseDarkOnesOwnLuck(dicetest.NewFixedRoller(3), opts...)).Run(cancelled, character.UseDarkOnesOwnLuckCommand{Character: c}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("an invalid command left signals: %+v", rec.Tracer.Spans())
	}
	if !srderrIsInvalid(errOf(character.NewChooseWarlockInvocations(nil))) || !srderrIsInvalid(errOf(character.NewUseDarkOnesOwnLuck(nil))) {
		t.Error("a constructor accepted a nil collaborator")
	}
	wrong := must(character.WithFeatureState(restored(t, 355000, 200, warScores(t, 16), cl("warlock", 5)), character.MustFeatureStateKey[character.BardicInspirationPool]("warlock"), character.BardicInspirationPool{}))
	if _, err := must(character.NewChooseWarlockInvocations(src, opts...)).Run(ctx, character.ChooseWarlockInvocationsCommand{Character: wrong}); !srderrIsInvalid(err) {
		t.Errorf("another type of state: %v", err)
	}
	spans := rec.Tracer.SpansNamed(character.SpanChooseWarlockInvocations)
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 {
		t.Errorf("an invalid state must be recorded on the span only: %+v", rec.Tracer.Spans())
	}
	// A Dark One's Own Luck call with an invalid roll kind is an error recorded on the span only.
	rec, opts = warTelemetry()
	roller := dicetest.NewFixedRoller(3)
	if _, err := must(character.NewUseDarkOnesOwnLuck(roller, opts...)).Run(ctx, character.UseDarkOnesOwnLuckCommand{Character: c, Charisma: warMod(16)}); !srderrIsInvalid(err) || len(roller.Requests()) != 0 {
		t.Errorf("an invalid kind: %v, requests %v", err, roller.Requests())
	}
	if len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Error("an invalid kind must not count as a use")
	}
}

var _ = dice.Port
