package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/spellcasting.md FR-092, FR-127, FR-084 to FR-088, Tables P14, P16 - CreateSpellcasting
// builds a state holding the character id, full slots in both pools and the lists given, for a
// Wizard 1 (P14) and a Warlock 1 (P16); equal results with Noop and recording telemetry; one span,
// one operations increment, one duration, no log.
// Path: happy.
func TestCreateSpellcastingCreated(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	cmd := character.CreateSpellcastingCommand{Character: caster, Levels: classLevelsOf(t, "wizard", 1), Spells: spells, Classes: []character.ClassSpellcastingSetup{{
		Class: character.CasterWizard, Cantrips: spellIDs(0, 3), Prepared: spellIDs(1, 4), Spellbook: character.NewSpellbook(spellIDs(1, 6))}}}
	rec, opts := recordedCasting()
	got, err := character.NewCreateSpellcasting(opts...).Do(context.Background(), cmd)
	plain, err2 := character.NewCreateSpellcasting().Do(context.Background(), cmd)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	if !reflect.DeepEqual(got, plain) {
		t.Error("FR-086: the result differs between Noop and recording telemetry")
	}
	st := got.(character.CreateSpellcastingCreated).State()
	w, ok := st.Class(character.CasterWizard)
	if st.CharacterID() != caster || !ok || w.Level() != 1 || w.Prepared().Count() != 4 || w.Cantrips().Count() != 3 || remaining(st, character.PoolSpellcasting, 1) != 2 || st.Slots(character.PoolPactMagic) != (character.SlotPoolView{}) {
		t.Errorf("state = %+v", st)
	}
	checkSignals(t, rec, wantSignals{
		span: character.SpanCreateSpellcasting, operation: "create_spellcasting", counter: character.MetricOperations,
		spanAttrs:    attrs(res("created", "success"), map[string]any{character.AttrOperation: "create_spellcasting"}),
		counterAttrs: attrs(res("created", "success"), map[string]any{character.AttrOperation: "create_spellcasting"}),
	})

	lock, err := character.NewCreateSpellcasting().Do(context.Background(), character.CreateSpellcastingCommand{Character: caster, Levels: classLevelsOf(t, "warlock", 1), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWarlock, Cantrips: spellIDs(0, 2), Prepared: spellIDs(1, 2)}}})
	if err != nil {
		t.Fatal(err)
	}
	ls := lock.(character.CreateSpellcastingCreated).State()
	if ls.Slots(character.PoolPactMagic).Maximum(slotLevel(1)) != 1 || ls.Slots(character.PoolPactMagic).Remaining(slotLevel(1)) != 1 || ls.Slots(character.PoolSpellcasting) != (character.SlotPoolView{}) {
		t.Errorf("P16 pools: pact %+v", ls.Slots(character.PoolPactMagic))
	}
}

// Requirement: doc/specs/spellcasting.md FR-092 - a held spellcasting class with no setup starts with empty lists; a class
// without Spellcasting gets no entry.
// Path: unhappy (empty and minimal).
func TestCreateSpellcastingDefaults(t *testing.T) {
	t.Parallel()
	st := castingState(t, classLevelsOf(t, "fighter", 1, "cleric", 2, "wizard", 1), spellCatalogue(t))
	cs := st.Classes()
	if len(cs) != 2 || cs[0].Class() != character.CasterCleric || cs[1].Class() != character.CasterWizard || cs[0].Level() != 2 || cs[0].Prepared().Count() != 0 {
		t.Errorf("classes = %+v", cs)
	}
}

// Requirement: doc/specs/spellcasting.md FR-054, FR-085, Table P15 - a refused list is a
// CreateSpellcastingRejected listing every problem of every class; outcome failure, no error
// status, the first reason as srd5e.character.rejection, one counter increment, no log.
// Path: unhappy (boundary).
func TestCreateSpellcastingRejected(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	rec, opts := recordedCasting()
	got, err := character.NewCreateSpellcasting(opts...).Do(context.Background(), character.CreateSpellcastingCommand{Character: caster, Levels: classLevelsOf(t, "wizard", 1, "cleric", 1), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{
			{Class: character.CasterWizard, Prepared: spellIDs(1, 5), Spellbook: character.NewSpellbook(spellIDs(1, 5))},
			{Class: character.CasterCleric, Cantrips: spellIDs(0, 4)}}})
	if err != nil {
		t.Fatal(err)
	}
	r, ok := got.(character.CreateSpellcastingRejected)
	if !ok {
		t.Fatalf("result = %#v", got)
	}
	p := r.Rejection().Problems()
	if len(p) != 2 || p[0].Kind != character.PreparationTooManyPrepared || p[0].Class != character.CasterWizard || p[1].Kind != character.PreparationTooManyCantrips || p[1].Class != character.CasterCleric {
		t.Errorf("problems = %+v", p)
	}
	checkSignals(t, rec, wantSignals{
		span: character.SpanCreateSpellcasting, operation: "create_spellcasting", counter: character.MetricOperations,
		spanAttrs:    attrs(res("rejected", "failure"), map[string]any{character.AttrOperation: "create_spellcasting", character.AttrRejection: "too_many_prepared"}),
		counterAttrs: attrs(res("rejected", "failure"), map[string]any{character.AttrOperation: "create_spellcasting", character.AttrRejection: "too_many_prepared"}),
	})
}

// Requirement: doc/specs/spellcasting.md FR-117, FR-132, Table R3 - a spell absent from the map is
// a CreateSpellcastingContentMissing holding each absent id once in command order, taking
// precedence over a rule rejection; recorded once: failure, content_missing, error status, no
// rejection, no error.type, no log.
// Path: unhappy (invalid input).
func TestCreateSpellcastingContentMissing(t *testing.T) {
	t.Parallel()
	rec, opts := recordedCasting()
	got, err := character.NewCreateSpellcasting(opts...).Do(context.Background(), character.CreateSpellcastingCommand{Character: caster, Levels: classLevelsOf(t, "wizard", 1), Spells: spellCatalogue(t),
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Prepared: append(spellIDs(1, 5), "gone_a", "gone_a"), Cantrips: []content.SpellID{"gone_b"}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(character.CreateSpellcastingContentMissing)
	if !ok || len(m.Missing()) != 2 || m.Missing()[0] != content.SpellID("gone_a").Missing() || m.Missing()[1] != content.SpellID("gone_b").Missing() {
		t.Fatalf("result = %#v", got)
	}
	m.Missing()[0] = content.MissingContent{}
	if m.Missing()[0] == (content.MissingContent{}) {
		t.Error("Missing shares memory")
	}
	checkSignals(t, rec, wantSignals{
		span: character.SpanCreateSpellcasting, operation: "create_spellcasting", counter: character.MetricOperations, status: telemetry.StatusError,
		spanAttrs:    attrs(res("content_missing", "failure"), map[string]any{character.AttrOperation: "create_spellcasting"}),
		counterAttrs: attrs(res("content_missing", "failure"), map[string]any{character.AttrOperation: "create_spellcasting"}),
	})
}

// Requirement: doc/specs/spellcasting.md FR-126, FR-087, FR-129 - a setup or granted cantrip naming a class
// the levels lack is an ErrClassNotHeld error recorded on the span only (no counter, no log); a
// repeated or invalid setup, a granted cantrip of another class, a zero command and a cancelled
// context are errors before any result.
// Path: unhappy (invalid input).
func TestCreateSpellcastingInvalid(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	levels := classLevelsOf(t, "wizard", 1)
	rec, opts := recordedCasting()
	uc := character.NewCreateSpellcasting(opts...)
	_, err := uc.Do(context.Background(), character.CreateSpellcastingCommand{Character: caster, Levels: levels, Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterCleric}}})
	if !errors.Is(err, character.ErrClassNotHeld) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrClassNotHeld", err)
	}
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !errors.Is(spans[0].Errors[0], character.ErrClassNotHeld) {
		t.Fatalf("spans = %+v, want the error recorded once", spans)
	}
	if len(rec.Meter.MeasurementsNamed(character.MetricOperations)) != 0 || len(rec.Meter.MeasurementsNamed(character.MetricDuration)) != 1 {
		t.Error("an invalid command incremented the outcome counter or skipped the duration")
	}
	noLogsOrViolations(t, rec)
	granted := mustGranted(t, spellID(0, 1), character.CasterCleric, "cleric")
	for name, cmd := range map[string]character.CreateSpellcastingCommand{
		"granted class not held":    {Character: caster, Levels: levels, Spells: spells, Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Granted: []character.GrantedCantrip{granted}}}},
		"granted for another class": {Character: caster, Levels: classLevelsOf(t, "wizard", 1, "cleric", 1), Spells: spells, Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Granted: []character.GrantedCantrip{granted}}}},
	} {
		if _, err := uc.Do(context.Background(), cmd); err == nil || (name == "granted class not held") != errors.Is(err, character.ErrClassNotHeld) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	for name, cmd := range map[string]character.CreateSpellcastingCommand{
		"zero character": {Levels: levels}, "zero levels": {Character: caster},
		"repeated setup": {Character: caster, Levels: levels, Spells: spells, Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard}, {Class: character.CasterWizard}}},
		"invalid class":  {Character: caster, Levels: levels, Spells: spells, Classes: []character.ClassSpellcastingSetup{{}}},
		"negative extra": {Character: caster, Levels: levels, Spells: spells, Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, ExtraCantrips: -1}}},
	} {
		if _, err := uc.Do(context.Background(), cmd); !isInvalid(err) || errors.Is(err, character.ErrClassNotHeld) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := (character.CreateSpellcasting{}).Do(context.Background(), character.CreateSpellcastingCommand{Character: caster, Levels: levels}); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Do(ctx, character.CreateSpellcastingCommand{Character: caster, Levels: levels}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
}
