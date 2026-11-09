package character_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// castSignals is what one casting use case call must emit.
type castSignals struct {
	span, op, counter string
	attrs             map[string]any // span attributes (the counter has them without the operation unless withOp)
	withOp            bool
	slotLevel         float64 // 0: none recorded
	debugResult       string  // the FINE log result; "" for no log
}

func checkCasting(t *testing.T, rec *telemetrytest.Recorder, w castSignals) {
	t.Helper()
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Name != w.span || !spans[0].Ended || spans[0].Status != telemetry.StatusUnset {
		t.Fatalf("spans = %+v, want one ended %s without error status [FR-085]", spans, w.span)
	}
	spanAttrs := attrs(w.attrs, map[string]any{character.AttrOperation: w.op})
	sameAttrs(t, "span", spans[0].Attrs, spanAttrs)
	counterAttrs := w.attrs
	if w.withOp {
		counterAttrs = spanAttrs
	}
	sameAttrs(t, "counter "+w.counter, onlyOne(t, rec, w.counter).Attrs, counterAttrs)
	if d := onlyOne(t, rec, character.MetricDuration); d.Value != 0.25 {
		t.Errorf("duration = %v, want 0.25", d.Value)
	} else {
		sameAttrs(t, "duration", d.Attrs, map[string]any{character.AttrOperation: w.op})
	}
	levels := rec.Meter.MeasurementsNamed(character.MetricSpellSlotLevel)
	switch {
	case w.slotLevel == 0 && len(levels) != 0:
		t.Errorf("slot level recorded %+v, want none", levels)
	case w.slotLevel != 0 && (len(levels) != 1 || levels[0].Value != w.slotLevel):
		t.Errorf("slot level = %+v, want %v", levels, w.slotLevel)
	}
	logs := rec.Logger.Records()
	switch {
	case w.debugResult == "" && len(logs) != 0:
		t.Errorf("logged %+v, want nothing", logs)
	case w.debugResult != "" && (len(logs) != 1 || logs[0].Level != telemetry.LevelDebug || logs[0].Message != character.LogSpellCastResolved):
		t.Errorf("logs = %+v, want one debug %s", logs, character.LogSpellCastResolved)
	case w.debugResult != "":
		sameAttrs(t, "log", logs[0].Attrs, map[string]any{character.AttrResult: w.debugResult})
	}
	telemetrytest.RequireNoViolations(t, rec)
}

func outcomeOf(result string) string {
	if result == "rejected" {
		return "failure"
	}
	return "success"
}

// Requirement: doc/specs/casting-and-concentration.md FR-082, FR-085, FR-133, FR-137, Section 7, FR-143, FR-150 - CastSpell
// records one span, one spells_cast increment, one duration and, for a spent slot, spell_slot_level; a rejection is a
// failure without error status; a scroll, a duplicated cast or a class-less cast has no srd5e.class; the result is equal
// with Noop and recording telemetry.
// Path: per case.
func TestCastSpellSignals(t *testing.T) {
	t.Parallel()
	s := warlock11Wizard5(t)
	cases := []struct {
		name string
		c    castCase
		w    castSignals
		res  string
		src  string
		cls  string
		lvl  float64
		rej  string
	}{
		{name: "slot cast", c: castCase{spell: l1(t), slot: choice(2)}, res: "cast", src: "slot", cls: "wizard", lvl: 2},
		{name: "cantrip", c: castCase{spell: castSpell(t, cantripKnown, 0)}, res: "cast", src: "cantrip", cls: "wizard"},
		{name: "started", c: castCase{spell: l1(t, withTimes(times(timeHour))), slot: choice(1)}, res: "started", src: "slot", cls: "wizard"},
		{name: "ritual", c: castCase{spell: castSpell(t, ritualPrepared, 1, asRitual()), mode: character.CastModeRitual}, res: "started", src: "ritual", cls: "wizard"},
		{name: "rejected", c: castCase{spell: castSpell(t, preparedL3, 3), slot: choice(2), mods: mods(func(f *character.CastingFacts) { f.CanSpeak = false })}, res: "rejected", src: "slot", cls: "wizard", rej: "missing_component_verbal"},
		{name: "feature granted", c: castCase{state: s, class: character.CasterWarlock, mode: character.CastModeFeatureGranted, spell: castSpell(t, level6Spell, 6)}, res: "cast", src: "feature_granted", cls: "warlock"},
		{name: "scroll", c: castCase{state: s, noClass: true, mode: character.CastModeSpellScroll, terms: terms13(t, 0), spell: castSpell(t, preparedL3, 3)}, res: "cast", src: "scroll"},
		{name: "duplicated", c: castCase{state: s, noClass: true, mode: character.CastModeDuplicated, terms: terms13(t, 0), spell: castSpell(t, preparedL3, 3)}, res: "cast", src: "duplicated"},
		{name: "rejected scroll", c: castCase{state: s, noClass: true, mode: character.CastModeSpellScroll, terms: terms13(t, 0), spell: castSpell(t, preparedL3, 3), mods: mods(unavailable(preparedL3))}, res: "rejected", src: "scroll", rej: "spell_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := map[string]any{telemetry.AttrOutcome: outcomeOf(tc.res), character.AttrResult: tc.res, character.AttrCastSource: tc.src}
			if tc.cls != "" {
				a[character.AttrClass] = tc.cls
			}
			if tc.rej != "" {
				a[character.AttrRejection] = tc.rej
			}
			rec, opts := recordedCasting()
			cmd := tc.c.command(t)
			got, err := character.NewCastSpell(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewCastSpell().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v (%v, %v), plain %#v [telemetry must not change the result]", got, err, err2, plain)
			}
			checkCasting(t, rec, castSignals{span: character.SpanCastSpell, op: "cast_spell", counter: character.MetricSpellsCast, attrs: a, slotLevel: tc.lvl, debugResult: tc.res})
		})
	}
	t.Run("a validation error emits nothing", func(t *testing.T) {
		t.Parallel()
		rec, opts := recordedCasting()
		if _, err := character.NewCastSpell(opts...).Do(context.Background(), castCase{spell: l1(t)}.command(t)); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("err = %v", err)
		}
		if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricSpellsCast)) != 0 || len(rec.Logger.Records()) != 0 {
			t.Error("signals were emitted for invalid input [FR-087]")
		}
	})
}

// Requirement: doc/specs/casting-and-concentration.md FR-071, FR-085, FR-105, Section 7 - CompleteCasting records a
// spells_cast increment with the source and class of the casting (none when nothing was in progress), the first rejection
// reason, and spell_slot_level for a spent slot; equal results with Noop and recording telemetry.
// Path: per case.
func TestCompleteCastingSignals(t *testing.T) {
	t.Parallel()
	hour := castSpell(t, preparedL1, 1, withTimes(times(timeHour)))
	started := mustStart(t, castCase{spell: hour, slot: choice(1)}).State()
	wiz := wizard5(t)
	cases := []struct {
		name  string
		state character.SpellcastingState
		mods  []func(*character.CastingFacts)
		res   string
		a     map[string]any
		lvl   float64
	}{
		{"completed", started, nil, "cast", map[string]any{character.AttrCastSource: "slot", character.AttrClass: "wizard"}, 1},
		{"rejected re-check", started, mods(slotSpent), "rejected", map[string]any{character.AttrCastSource: "slot", character.AttrClass: "wizard", character.AttrRejection: "slot_already_spent_this_turn"}, 0},
		{"nothing in progress", wiz, nil, "rejected", map[string]any{character.AttrRejection: "casting_not_in_progress"}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := attrs(map[string]any{telemetry.AttrOutcome: outcomeOf(tc.res), character.AttrResult: tc.res}, tc.a)
			rec, opts := recordedCasting()
			cmd := character.CompleteCastingCommand{State: tc.state, Spell: hour, Context: castContext(t, tc.mods...)}
			got, err := character.NewCompleteCasting(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewCompleteCasting().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v (%v, %v), plain %#v", got, err, err2, plain)
			}
			checkCasting(t, rec, castSignals{span: character.SpanCompleteCasting, op: "complete_casting", counter: character.MetricSpellsCast, attrs: a, slotLevel: tc.lvl})
		})
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-133, Table N (N30), Section 7 - BreakConcentration and
// BeginConcentration each increment srd5e.character.operations exactly once whatever the result, record none of spells_cast,
// and carry the concentration reason on BreakConcentration; equal results with Noop and recording telemetry.
// Path: per case.
func TestConcentrationUseCaseSignals(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	onSpell := concentratingWizard(t)
	casting := castingStarted(t, wiz)
	for _, tc := range []struct {
		name, res string
		state     character.SpellcastingState
		reason    character.ConcentrationBreakReason
		rs        string
	}{
		{"ended", "ended", onSpell, character.BreakFailedSave, "failed_save"},
		{"casting failed", "casting_failed", casting, character.BreakIncapacitated, "incapacitated"},
		{"none", "none", wiz, character.BreakAnotherEffect, "another_effect"},
		{"dead", "ended", onSpell, character.BreakDead, "dead"},
		{"ended by the caster", "ended", onSpell, character.BreakEnded, "ended"},
	} {
		t.Run("break "+tc.name, func(t *testing.T) {
			t.Parallel()
			a := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: tc.res, character.AttrConcentrationReason: tc.rs}
			rec, opts := recordedCasting()
			cmd := character.BreakConcentrationCommand{State: tc.state, Reason: tc.reason}
			got, err := character.NewBreakConcentration(opts...).Do(context.Background(), cmd)
			plain, _ := character.NewBreakConcentration().Do(context.Background(), cmd)
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v, %v, plain %#v", got, err, plain)
			}
			checkCasting(t, rec, castSignals{span: character.SpanBreakConcentration, op: "break_concentration", counter: character.MetricOperations, attrs: a, withOp: true})
			if n := len(rec.Meter.MeasurementsNamed(character.MetricSpellsCast)); n != 0 {
				t.Errorf("%d spells_cast measurements, want none [FR-133]", n)
			}
		})
	}
	for _, tc := range []struct {
		name, res string
		state     character.SpellcastingState
	}{{"began", "began", wiz}, {"casting failed", "casting_failed", casting}} {
		t.Run("begin "+tc.name, func(t *testing.T) {
			t.Parallel()
			a := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: tc.res}
			rec, opts := recordedCasting()
			cmd := character.BeginConcentrationCommand{State: tc.state, Subject: featureSubject}
			got, err := character.NewBeginConcentration(opts...).Do(context.Background(), cmd)
			plain, _ := character.NewBeginConcentration().Do(context.Background(), cmd)
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Fatalf("result = %#v, %v, plain %#v", got, err, plain)
			}
			checkCasting(t, rec, castSignals{span: character.SpanBeginConcentration, op: "begin_concentration", counter: character.MetricOperations, attrs: a, withOp: true})
		})
	}
}

// Requirement: doc/specs/casting-and-concentration.md Section 7, FR-141, FR-142 - the casting signal names, attribute keys
// and values are a public contract: they are pinned here and catalogued in doc/observability/signals.md.
// Path: happy.
func TestCastingSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	pairs := []struct{ got, want string }{
		{character.SpanCastSpell, "srd5e.character.cast_spell"},
		{character.SpanCompleteCasting, "srd5e.character.complete_casting"},
		{character.SpanBreakConcentration, "srd5e.character.break_concentration"},
		{character.SpanBeginConcentration, "srd5e.character.begin_concentration"},
		{character.MetricSpellsCast, "srd5e.character.spells_cast"},
		{character.LogSpellCastResolved, "srd5e.character.spell_cast_resolved"},
		{character.AttrCastSource, "srd5e.character.cast_source"},
		{character.AttrConcentrationReason, "srd5e.character.concentration_reason"},
		{character.ValueOperationCastSpell, "cast_spell"},
		{character.ValueOperationCompleteCasting, "complete_casting"},
		{character.ValueOperationBreakConcentration, "break_concentration"},
		{character.ValueOperationBeginConcentration, "begin_concentration"},
		{character.ValueResultCast, "cast"},
		{character.ValueResultStarted, "started"},
		{character.ValueResultEnded, "ended"},
		{character.ValueResultBegan, "began"},
		{character.ValueResultNone, "none"},
		{character.ValueResultCastingFailed, "casting_failed"},
		{character.CastSourceSlot.String(), "slot"}, {character.CastSourceCantrip.String(), "cantrip"}, {character.CastSourceRitual.String(), "ritual"},
		{character.CastSourceFeatureGranted.String(), "feature_granted"}, {character.CastSourceScroll.String(), "scroll"}, {character.CastSourceDuplicated.String(), "duplicated"},
		{character.BreakIncapacitated.String(), "incapacitated"}, {character.BreakDead.String(), "dead"}, {character.BreakEnded.String(), "ended"},
		{character.BreakFailedSave.String(), "failed_save"}, {character.BreakAnotherEffect.String(), "another_effect"},
	}
	for i := character.CastRejectionCannotCastSpells; i <= character.CastRejectionCastingNotInProgress; i++ {
		pairs = append(pairs, struct{ got, want string }{i.String(), i.String()})
	}
	for _, c := range pairs {
		if c.got != c.want {
			t.Errorf("signal = %q, want %q", c.got, c.want)
		}
		if strings.Contains(c.want, "srd5e.") && !strings.Contains(string(catalogue), "`"+c.want+"`") {
			t.Errorf("%q is missing from doc/observability/signals.md", c.want)
		}
		if !strings.HasPrefix(c.want, "srd5e.character.") && c.want != "srd5e.character.cast_source" && !strings.Contains(string(rules), c.want) {
			t.Errorf("%q is missing from doc/rules/observability.md section 5", c.want)
		}
	}
}
