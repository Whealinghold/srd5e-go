package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func changeSignals(span, op, result, outcomeValue, class, rejection string, status telemetry.Status) wantSignals {
	a := attrs(res(result, outcomeValue), map[string]any{character.AttrClass: class})
	if rejection != "" {
		a[character.AttrRejection] = rejection
	}
	return wantSignals{span: span, operation: op, counter: character.MetricPreparedSpellChanges, status: status,
		spanAttrs: attrs(a, map[string]any{character.AttrOperation: op}), counterAttrs: a}
}

// Requirement: doc/specs/spellcasting.md FR-089, FR-117, FR-132, FR-084 to FR-088, Tables P6, P7, P17, R5 -
// ChooseAdditionalSpells maps each PreparationOutcome to its result and records it once on the
// prepared_spell_changes counter with the class (and the first reason when rejected); a content-missing
// result is a failure with error status and no rejection; Noop and recording telemetry give equal results.
// Path: per case.
func TestChooseAdditionalSpellsUseCase(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	cleric4 := castingState(t, classLevelsOf(t, "cleric", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: cat(spellIDs(1, 3), spellIDs(2, 3))})
	op := character.ValueOperationChooseAdditionalSpells
	cases := []struct {
		name     string
		prepared []content.SpellID
		want     func(character.ChooseAdditionalSpellsResult) bool
		signals  wantSignals
	}{
		{"P6 one added", spellIDs(2, 4)[3:], func(r character.ChooseAdditionalSpellsResult) bool {
			c, ok := r.(character.ChooseAdditionalSpellsChosen)
			return ok && len(c.Events()) == 1 && classOfState(t, c.State(), character.CasterCleric).Prepared().Count() == 7
		}, changeSignals(character.SpanChooseAdditionalSpells, op, "changed", "success", "cleric", "", telemetry.StatusUnset)},
		{"P17 none added", nil, func(r character.ChooseAdditionalSpellsResult) bool {
			c, ok := r.(character.ChooseAdditionalSpellsChosen)
			return ok && len(c.Events()) == 0
		}, changeSignals(character.SpanChooseAdditionalSpells, op, "changed", "success", "cleric", "", telemetry.StatusUnset)},
		{"P7 two added", spellIDs(2, 5)[3:], func(r character.ChooseAdditionalSpellsResult) bool {
			c, ok := r.(character.ChooseAdditionalSpellsRejected)
			return ok && c.Rejection().First() == character.PreparationTooManyPrepared
		}, changeSignals(character.SpanChooseAdditionalSpells, op, "rejected", "failure", "cleric", "too_many_prepared", telemetry.StatusUnset)},
		{"R5 a spell absent", []content.SpellID{"gone"}, func(r character.ChooseAdditionalSpellsResult) bool {
			c, ok := r.(character.ChooseAdditionalSpellsContentMissing)
			return ok && len(c.Missing()) == 1
		}, changeSignals(character.SpanChooseAdditionalSpells, op, "content_missing", "failure", "cleric", "", telemetry.StatusError)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ChooseAdditionalSpellsCommand{State: cleric4, Class: character.CasterCleric, Prepared: c.prepared, Spells: spells}
			rec, opts := recordedCasting()
			got, err := character.NewChooseAdditionalSpells(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewChooseAdditionalSpells().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) || !c.want(got) {
				t.Fatalf("result = %#v (%v, %v), plain %#v", got, err, err2, plain)
			}
			checkSignals(t, rec, c.signals)
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-126, FR-087 - a class not held is an ErrClassNotHeld error recorded
// on the span only; zero state, invalid class and a cancelled context are errors before any signal.
// Path: unhappy (invalid input).
func TestChooseAdditionalSpellsInvalid(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	st := castingState(t, classLevelsOf(t, "cleric", 1), spells)
	rec, opts := recordedCasting()
	uc := character.NewChooseAdditionalSpells(opts...)
	if _, err := uc.Do(context.Background(), character.ChooseAdditionalSpellsCommand{State: st, Class: character.CasterWizard, Spells: spells}); !errors.Is(err, character.ErrClassNotHeld) {
		t.Fatalf("err = %v", err)
	}
	if s := rec.Tracer.Spans(); len(s) != 1 || s[0].Status != telemetry.StatusError || len(s[0].Errors) != 1 || len(rec.Meter.MeasurementsNamed(character.MetricPreparedSpellChanges)) != 0 {
		t.Errorf("spans = %+v", s)
	}
	noLogsOrViolations(t, rec)
	rec.Clear()
	for name, cmd := range map[string]character.ChooseAdditionalSpellsCommand{"zero state": {Class: character.CasterCleric}, "invalid class": {State: st}} {
		if _, err := uc.Do(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (character.ChooseAdditionalSpells{}).Do(context.Background(), character.ChooseAdditionalSpellsCommand{State: st, Class: character.CasterCleric}); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Do(ctx, character.ChooseAdditionalSpellsCommand{State: st, Class: character.CasterCleric}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if len(rec.Tracer.Spans()) != 0 {
		t.Error("an invalid call emitted signals")
	}
}

// Requirement: doc/specs/spellcasting.md FR-090, FR-117, FR-132, FR-084 to FR-088, Tables P1, P3, R4 -
// ChangePreparedSpells maps each PreparationOutcome to its result, records it once on the
// prepared_spell_changes counter, and gives equal results with Noop and recording telemetry.
// Path: per case.
func TestChangePreparedSpellsUseCase(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	wiz5 := preparedWizard5(t, spells)
	op := character.ValueOperationChangePreparedSpells
	swap1 := cat(spellIDs(1, 4), spellIDs(1, 6)[5:], spellIDs(2, 4))
	swap2 := cat(spellIDs(1, 3), spellIDs(1, 7)[5:], spellIDs(2, 4))
	cases := []struct {
		name     string
		prepared []content.SpellID
		want     func(character.ChangePreparedSpellsResult) bool
		signals  wantSignals
	}{
		{"P1 one swapped", swap1, func(r character.ChangePreparedSpellsResult) bool {
			c, ok := r.(character.ChangePreparedSpellsChanged)
			return ok && len(c.Events()) == 1 && c.State().CharacterID() == caster
		}, changeSignals(character.SpanChangePreparedSpells, op, "changed", "success", "wizard", "", telemetry.StatusUnset)},
		{"P3 two swapped", swap2, func(r character.ChangePreparedSpellsResult) bool {
			c, ok := r.(character.ChangePreparedSpellsRejected)
			return ok && c.Rejection().First() == character.PreparationTooManyReplacements
		}, changeSignals(character.SpanChangePreparedSpells, op, "rejected", "failure", "wizard", "too_many_replacements", telemetry.StatusUnset)},
		{"R4 a spell absent", append(swap1[:8:8], "gone"), func(r character.ChangePreparedSpellsResult) bool {
			c, ok := r.(character.ChangePreparedSpellsContentMissing)
			return ok && len(c.Missing()) == 1
		}, changeSignals(character.SpanChangePreparedSpells, op, "content_missing", "failure", "wizard", "", telemetry.StatusError)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ChangePreparedSpellsCommand{State: wiz5, Class: character.CasterWizard, Trigger: character.TriggerShortRest, Change: character.PreparationChange{Prepared: c.prepared}, Spells: spells}
			rec, opts := recordedCasting()
			got, err := character.NewChangePreparedSpells(opts...).Do(context.Background(), cmd)
			plain, err2 := character.NewChangePreparedSpells().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(got, plain) || !c.want(got) {
				t.Fatalf("result = %#v (%v, %v), plain %#v", got, err, err2, plain)
			}
			checkSignals(t, rec, c.signals)
		})
	}
}

// Requirement: doc/specs/spellcasting.md FR-126, FR-087 - ErrClassNotHeld and a bad replacement are errors
// recorded on the span only; zero state, invalid class or trigger and a cancelled context are errors
// before any signal.
// Path: unhappy (invalid input).
func TestChangePreparedSpellsInvalid(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	st := castingState(t, classLevelsOf(t, "cleric", 1), spells)
	rec, opts := recordedCasting()
	uc := character.NewChangePreparedSpells(opts...)
	if _, err := uc.Do(context.Background(), character.ChangePreparedSpellsCommand{State: st, Class: character.CasterWizard, Trigger: character.TriggerLongRest, Spells: spells}); !errors.Is(err, character.ErrClassNotHeld) {
		t.Fatalf("err = %v", err)
	}
	if s := rec.Tracer.Spans(); len(s) != 1 || s[0].Status != telemetry.StatusError || len(s[0].Errors) != 1 || len(rec.Meter.MeasurementsNamed(character.MetricPreparedSpellChanges)) != 0 {
		t.Errorf("spans = %+v", s)
	}
	noLogsOrViolations(t, rec)
	rec.Clear()
	for name, cmd := range map[string]character.ChangePreparedSpellsCommand{
		"zero state": {Class: character.CasterCleric, Trigger: character.TriggerLongRest}, "invalid class": {State: st, Trigger: character.TriggerLongRest}, "invalid trigger": {State: st, Class: character.CasterCleric},
	} {
		if _, err := uc.Do(context.Background(), cmd); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (character.ChangePreparedSpells{}).Do(context.Background(), character.ChangePreparedSpellsCommand{State: st, Class: character.CasterCleric, Trigger: character.TriggerLongRest}); !isInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Do(ctx, character.ChangePreparedSpellsCommand{State: st, Class: character.CasterCleric, Trigger: character.TriggerLongRest}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
	if len(rec.Tracer.Spans()) != 0 {
		t.Error("an invalid call emitted signals")
	}
}
