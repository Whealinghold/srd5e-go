package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

type sorMode int

const (
	sorOK sorMode = iota
	sorBad
	sorNot
)

// sorCase drives one Sorcerer use case in the three modes (an accepted command, one a rule refuses, a character that is not a Sorcerer).
type sorCase struct {
	name, span, feature, rejection string
	run                            func(t *testing.T, mode sorMode, src character.ContentSource, opts ...character.Option) (any, error)
}

func sorFighter(t *testing.T) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), cl("fighter", 4))
}

func sorCases() []sorCase {
	return []sorCase{
		{name: "ChooseMetamagicOptions", span: character.SpanChooseMetamagicOptions, feature: "metamagic", rejection: "metamagic_selection_invalid",
			run: func(t *testing.T, mode sorMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				c := sorChar(t, 2, 16, "", sorFresh(sorProfile(2, "")))
				if mode == sorNot {
					c = sorFighter(t)
				}
				chosen := sorOptionIDs(character.MetamagicCareful, character.MetamagicDistant)
				if mode == sorBad {
					chosen = chosen[:1]
				}
				return must(character.NewChooseMetamagicOptions(src, opts...)).Run(t.Context(), character.ChooseMetamagicOptionsCommand{Character: c, Chosen: chosen})
			}},
		{name: "ApplySorcererLevelUp", span: character.SpanApplySorcererLevelUp, feature: "sorcerer_level_up", rejection: "metamagic_selection_invalid",
			run: func(t *testing.T, mode sorMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				c := sorChar(t, 2, 16, "", sorWith(sorProfile(2, ""), 2, nil, nil))
				if mode == sorNot {
					c, gained := fgtLevelUp(t, sorFighter(t), "fighter")
					return must(character.NewApplySorcererLevelUp(src, opts...)).Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained})
				}
				c, gained := sorLevelUp(t, c)
				var metamagic []content.FeatureOptionID
				if mode == sorBad {
					metamagic = sorOptionIDs(character.MetamagicCareful)
				}
				return must(character.NewApplySorcererLevelUp(src, opts...)).Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained, Metamagic: metamagic})
			}},
		{name: "CreateSpellSlot", span: character.SpanCreateSpellSlot, feature: "font_of_magic", rejection: "slot_level_not_creatable",
			run: func(t *testing.T, mode sorMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				c := sorChar(t, 5, 16, "", sorWith(sorProfile(5, ""), 5, nil, nil))
				if mode == sorNot {
					c = sorFighter(t)
				}
				level := 3
				if mode == sorBad {
					level = 6
				}
				return must(character.NewCreateSpellSlot(src, opts...)).Run(t.Context(), character.CreateSpellSlotCommand{Character: c, SlotLevel: level})
			}},
		{name: "UseMetamagic", span: character.SpanUseMetamagic, feature: "metamagic", rejection: "metamagic_option_not_known",
			run: func(t *testing.T, mode sorMode, src character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				c := sorChar(t, 5, 16, "", sorWith(sorProfile(5, ""), 5, []character.MetamagicEffect{character.MetamagicCareful}, nil))
				if mode == sorNot {
					c = sorFighter(t)
				}
				option := content.FeatureOptionID("careful_spell")
				if mode == sorBad {
					option = "fireball"
				}
				return must(character.NewUseMetamagic(src, opts...)).Run(t.Context(), character.UseMetamagicCommand{Character: c, Option: option, Facts: sorFacts(nil),
					Parameters: character.MetamagicParameters{Count: 1}})
			}},
	}
}

// sorWant is what one Sorcerer call records.
type sorWant struct {
	span, feature, outcome, rejection string
	status                            telemetry.Status
	logged                            bool
	errorType                         string
	loaded                            bool // a load_content child span is expected
}

// sorCheck asserts the signals of exactly one Sorcerer call: one top-level span (and the load_content child when loaded), one feature_used
// increment, one duration, the errors counter and the one FINE log.
func sorCheck(t *testing.T, rec *telemetrytest.Recorder, w sorWant) {
	t.Helper()
	spans := rec.Tracer.SpansNamed(w.span)
	if len(spans) != 1 || !spans[0].Ended || spans[0].Status != w.status || spans[0].ParentID != 0 {
		t.Fatalf("spans = %+v, want one ended top-level %s with status %v", rec.Tracer.Spans(), w.span, w.status)
	}
	if load := rec.Tracer.SpansNamed(character.SpanLoadContent); w.loaded && (len(load) != 1 || load[0].ParentID != spans[0].ID) || !w.loaded && len(load) != 0 {
		t.Errorf("load_content spans = %+v, loaded %v", load, w.loaded)
	}
	ident := map[string]any{character.AttrClass: character.ValueClassSorcerer, character.AttrFeature: w.feature}
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

func sorTypeOf(v any) string { return reflect.TypeOf(v).Name() }

// Requirement: doc/specs/class-sorcerer.md FR-124, FR-130 - every Sorcerer use case, in one call, emits one top-level span named per the spec (with the load_content child), one feature_used increment and one duration with
// srd5e.class sorcerer and its srd5e.feature; an accepted result is outcome success; a rule refusal is outcome failure with the snake_case SorcererRejectionKind, no error.type, no error status and no errors increment; a character that
// is not a Sorcerer is outcome failure with rejection not_a_sorcerer, decided before any lookup. Each result variant is named with its use case. (The shared featureRun also writes the one FINE log of a normal outcome.)
// Path: per case (happy / unhappy: state).
func TestSorcererUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range sorCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			src := sorSource(t)
			rec, opts := sorTelemetry()
			res, err := c.run(t, sorOK, src, opts...)
			if err != nil || sorTypeOf(res) != c.name+"Applied" {
				t.Fatalf("accepted = %T, %v; want %sApplied", res, err, c.name)
			}
			sorCheck(t, rec, sorWant{span: c.span, feature: c.feature, outcome: "success", status: telemetry.StatusUnset, logged: true, loaded: true})
			rec, opts = sorTelemetry()
			res, err = c.run(t, sorBad, src, opts...)
			if err != nil || sorTypeOf(res) != c.name+"Rejected" {
				t.Fatalf("refused = %T, %v; want %sRejected", res, err, c.name)
			}
			sorCheck(t, rec, sorWant{span: c.span, feature: c.feature, outcome: "failure", rejection: c.rejection, status: telemetry.StatusUnset, logged: true, loaded: true})
			rec, opts = sorTelemetry()
			res, err = c.run(t, sorNot, src, opts...)
			if err != nil || sorTypeOf(res) != c.name+"Rejected" {
				t.Fatalf("not a Sorcerer = %T, %v; want %sRejected", res, err, c.name)
			}
			if got := reflect.ValueOf(res).FieldByName("Rejection").Interface().(character.SorcererRejection); got.Kind != character.SorcererNotASorcerer {
				t.Errorf("rejection = %+v, want SorcererNotASorcerer", got)
			}
			sorCheck(t, rec, sorWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "not_a_sorcerer", status: telemetry.StatusUnset, logged: true})
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-099, FR-125 - when the sorcerer class is absent every use case returns its own ContentMissing variant naming the class, with error status on the span, rejection content_missing and no log; a source
// that returns its error or answers with another entry is a *srderr.PortError for content, recorded once (span error status, one feature_used failure, one errors increment with error.type) and propagated with no result.
// Path: per case (unhappy: missing content, collaborator failure).
func TestSorcererUseCaseContentFailures(t *testing.T) {
	t.Parallel()
	for _, c := range sorCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := sorTelemetry()
			res, err := c.run(t, sorOK, contenttest.NewFakeSource(), opts...)
			if err != nil || sorTypeOf(res) != c.name+"ContentMissing" {
				t.Fatalf("missing = %T, %v; want %sContentMissing", res, err, c.name)
			}
			missing := reflect.ValueOf(res).MethodByName("Missing").Call(nil)[0].Interface().([]content.MissingContent)
			if len(missing) != 1 || missing[0].Kind() != content.KindCharacterClass || missing[0].ID().String() != "sorcerer" {
				t.Errorf("Missing = %v", missing)
			}
			sorCheck(t, rec, sorWant{span: c.span, feature: c.feature, outcome: "failure", rejection: "content_missing", status: telemetry.StatusError, loaded: true})
			for name, src := range map[string]character.ContentSource{"an error": contenttest.NewFakeSource().FailLookup(errBoom), "wrong entry": wrongEntrySource{entry: sorSource(t)}} {
				rec, opts = sorTelemetry()
				res, err = c.run(t, sorOK, src, opts...)
				var pe *srderr.PortError
				if res != nil || !asPortError(err, &pe) || pe.Port != content.Port {
					t.Fatalf("%s: = %v, %v; want a PortError for content", name, res, err)
				}
				sorCheck(t, rec, sorWant{span: c.span, feature: c.feature, outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err), loaded: true})
				if loads := rec.Tracer.SpansNamed(character.SpanLoadContent); len(loads) != 1 || loads[0].Status != telemetry.StatusError {
					t.Errorf("%s: load_content = %+v, want error status", name, loads)
				}
			}
			if _, err := c.run(t, sorOK, contenttest.NewFakeSource().FailLookup(context.Canceled)); !errors.Is(err, context.Canceled) {
				t.Errorf("a cancelled lookup must stay visible through errors.Is: %v", err)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-100, FR-101, FR-126 - every Sorcerer use case gives an equal result with the no-op and a recording telemetry and repeated calls agree; the command inputs are not altered.
// Path: per case (unhappy: determinism).
func TestSorcererUseCasesAreDeterministic(t *testing.T) {
	t.Parallel()
	for _, c := range sorCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, mode := range []sorMode{sorOK, sorBad, sorNot} {
				plain, err1 := c.run(t, mode, sorSource(t))
				again, err3 := c.run(t, mode, sorSource(t))
				_, opts := sorTelemetry()
				recorded, err2 := c.run(t, mode, sorSource(t), opts...)
				if err1 != nil || err2 != nil || err3 != nil || !reflect.DeepEqual(plain, recorded) || !reflect.DeepEqual(plain, again) {
					t.Errorf("mode %d: %#v (%v) versus %#v (%v) and %#v (%v)", mode, plain, err1, recorded, err2, again, err3)
				}
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-099, FR-125 (programmer errors) - the zero use case, a zero Character and a cancelled context are errors matching ErrInvalidInput (or ctx.Err()) before any signal; a nil source is rejected by the constructor;
// a stored state of another type under the sorcerer key is an error recorded on the span only.
// Path: unhappy (invalid input).
func TestSorcererUseCaseInvalidCommands(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec, opts := sorTelemetry()
	src := sorSource(t)
	c := sorChar(t, 5, 16, "", sorFresh(sorProfile(5, "")))
	if _, err := (character.ChooseMetamagicOptions{}).Run(ctx, character.ChooseMetamagicOptionsCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero ChooseMetamagicOptions: %v", err)
	}
	if _, err := (character.ApplySorcererLevelUp{}).Run(ctx, character.ApplySorcererLevelUpCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero ApplySorcererLevelUp: %v", err)
	}
	if _, err := (character.CreateSpellSlot{}).Run(ctx, character.CreateSpellSlotCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero CreateSpellSlot: %v", err)
	}
	if _, err := (character.UseMetamagic{}).Run(ctx, character.UseMetamagicCommand{Character: c}); !srderrIsInvalid(err) {
		t.Errorf("zero UseMetamagic: %v", err)
	}
	if _, err := must(character.NewCreateSpellSlot(src, opts...)).Run(ctx, character.CreateSpellSlotCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero Character: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := must(character.NewUseMetamagic(src, opts...)).Run(cancelled, character.UseMetamagicCommand{Character: c}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 {
		t.Errorf("an invalid command left signals: %+v", rec.Tracer.Spans())
	}
	for i, err := range []error{errOf(character.NewChooseMetamagicOptions(nil)), errOf(character.NewApplySorcererLevelUp(nil)), errOf(character.NewCreateSpellSlot(nil)), errOf(character.NewUseMetamagic(nil))} {
		if !srderrIsInvalid(err) {
			t.Errorf("constructor %d accepted a nil source: %v", i, err)
		}
	}
	wrong := must(character.WithFeatureState(restored(t, 355000, 200, sorScores(t, 16), cl("sorcerer", 5)), character.MustFeatureStateKey[character.BardicInspirationPool]("sorcerer"), character.BardicInspirationPool{}))
	if _, err := must(character.NewCreateSpellSlot(src, opts...)).Run(ctx, character.CreateSpellSlotCommand{Character: wrong, SlotLevel: 1}); !srderrIsInvalid(err) {
		t.Errorf("another type of state: %v", err)
	}
	spans := rec.Tracer.SpansNamed(character.SpanCreateSpellSlot)
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsedErrors)) != 0 {
		t.Errorf("an invalid state must be recorded on the span only: %+v", rec.Tracer.Spans())
	}
}
