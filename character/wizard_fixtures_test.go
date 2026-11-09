package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/srderr"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// wzID is the character id every Wizard test uses.
var wzID = character.MustIdentifier("wiz-1")

// wzSource is the bundled content source (it holds the Wizard).
func wzSource(tb testing.TB) character.ContentSource {
	tb.Helper()
	return must(data.NewBundledSource())
}

// wzDef is the bundled Wizard ClassDefinition (content/data/class_wizard.json).
func wzDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	res := must(wzSource(tb).Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("wizard"))))
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

// wzLevels is the ClassLevels of a Wizard of level n.
func wzLevels(n int) character.ClassLevels { return levels(cl("wizard", n)) }

// wzFact builds the facts of a spell of the given level on the Wizard list with the Magic action as its casting time.
func wzFact(id string, level int, mods ...func(*character.WizardSpellFacts)) character.WizardSpellFacts {
	f := character.WizardSpellFacts{ID: content.SpellID(id), Level: content.SpellLevel(level), Classes: []content.ClassID{"wizard"},
		CastingTimes: []content.CastingTime{content.NewCastingTimeMagicAction()}}
	for _, m := range mods {
		m(&f)
	}
	return f
}

func wzOffList(f *character.WizardSpellFacts) { f.Classes = []content.ClassID{"cleric"} }
func wzRitual(f *character.WizardSpellFacts)  { f.Ritual = true }
func wzMinutes(f *character.WizardSpellFacts) {
	f.CastingTimes = []content.CastingTime{must(content.NewCastingTimeMinutes(1))}
}

// wzFacts builds n numbered facts of the level with the prefix.
func wzFacts(prefix string, level, n int) []character.WizardSpellFacts {
	var out []character.WizardSpellFacts
	for i := 1; i <= n; i++ {
		out = append(out, wzFact(prefix+string(rune('a'+i-1)), level))
	}
	return out
}

func wzBook(ids ...string) character.Spellbook {
	var out []content.SpellID
	for _, id := range ids {
		out = append(out, content.SpellID(id))
	}
	return character.NewSpellbook(out)
}

func wzIDs(ids ...string) []content.SpellID {
	var out []content.SpellID
	for _, id := range ids {
		out = append(out, content.SpellID(id))
	}
	return out
}

// wzRest is the RestApplied of a rest of kind: finished, or, with interrupted, a Long Rest cut short that earned the Short Rest benefits.
func wzRest(t *testing.T, kind character.RestKind, interrupted bool) character.RestApplied {
	t.Helper()
	if !interrupted {
		return fgtRest(t, wzID, kind)
	}
	cmd := must(character.NewTakeRestCommand(wzID, must(character.NewRestReport(character.LongRest, character.StandardLongRest, stopped(character.PhysicalExertion, 3, 0))), 12, 0, nil, character.SpentHitDice{}))
	return character.NewRestTaker().TakeRest(t.Context(), cmd).(character.TakeRestAppliedAfterInterruption).Applied()
}

// wzTelemetry is a recording bundle and the options that inject it with a stepping clock.
func wzTelemetry() (*telemetrytest.Recorder, []character.Option) {
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), time.Second)
	return rec, []character.Option{character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now)}
}

// wzWant is what one Wizard call records.
type wzWant struct {
	span, feature, outcome, rejection string
	status                            telemetry.Status
	logged                            bool
	errorType                         string
	loaded                            bool // a load_content child span is expected
}

// wzCheck asserts the signals of exactly one Wizard call: one top-level span (and the load_content child when loaded), one feature_used
// increment, one duration, the errors counter and the one FINE log.
func wzCheck(t *testing.T, rec *telemetrytest.Recorder, w wzWant) {
	t.Helper()
	spans := rec.Tracer.SpansNamed(w.span)
	if len(spans) != 1 || !spans[0].Ended || spans[0].Status != w.status || spans[0].ParentID != 0 {
		t.Fatalf("spans = %+v, want one ended top-level %s with status %v", rec.Tracer.Spans(), w.span, w.status)
	}
	if load := rec.Tracer.SpansNamed(character.SpanLoadContent); w.loaded && (len(load) != 1 || load[0].ParentID != spans[0].ID) || !w.loaded && len(load) != 0 {
		t.Errorf("load_content spans = %+v, loaded %v", load, w.loaded)
	}
	ident := map[string]any{character.AttrClass: character.ValueClassWizard, character.AttrFeature: w.feature}
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

// wzMissing is a source without the Wizard.
func wzMissing() *contenttest.FakeSource { return contenttest.NewFakeSource() }

func errorsIs(err, target error) bool { return errors.Is(err, target) }

// wzMissingFailing is a source whose every lookup fails with err.
func wzMissingFailing(err error) *contenttest.FakeSource {
	return contenttest.NewFakeSource().FailLookup(err)
}

// wzWrongEntry answers the wizard key with a species, which is a contract violation.
func wzWrongEntry(t *testing.T) character.ContentSource {
	t.Helper()
	return wrongEntrySource{entry: must(data.NewBundledSource())}
}

type wrongEntrySource struct{ entry character.ContentSource }

func (s wrongEntrySource) Lookup(ctx context.Context, _ content.Key) (content.LookupResult, error) {
	return s.entry.Lookup(ctx, content.NewKey(content.KindSpecies, content.MustID("elf")))
}

func asPortError(err error, target **srderr.PortError) bool { return errors.As(err, target) }
