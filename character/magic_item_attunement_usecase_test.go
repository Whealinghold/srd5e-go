package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// checkOneCall asserts the single span and counter increment of a use case call and the absence of logs.
func checkOneCall(t *testing.T, rec *telemetrytest.Recorder, span, op string, status telemetry.Status, attrs map[string]any, spanOnly ...string) {
	t.Helper()
	spans := rec.Tracer.Spans()
	if len(spans) != 1 || spans[0].Name != span || !spans[0].Ended || spans[0].Status != status {
		t.Fatalf("spans = %+v", spans)
	}
	sameAttrs(t, "span", spans[0].Attrs, attrs)
	counter := map[string]any{character.AttrOperation: op}
	for k, v := range attrs {
		if !slices.Contains(spanOnly, k) {
			counter[k] = v
		}
	}
	sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, counter)
	sameAttrs(t, "duration", onlyOne(t, rec, character.MetricDuration).Attrs, map[string]any{character.AttrOperation: op})
	noLogsOrViolations(t, rec)
}

// Requirement: doc/specs/magic-item-rules.md FR-040, FR-060, FR-105, FR-109, FR-122 - an accepted attempt returns the new set and
// one MagicItemAttuned carrying the character id, a rejection returns the reason with the set unchanged and no event; the no-op
// bundle returns the same result; one span and one counter increment, outcome failure and no error status for a rejection.
// Path: per case.
func TestAttuneToMagicItemSignalsAndResults(t *testing.T) {
	t.Parallel()
	item := attunable(t, "ring", "n", content.AttunementRequired{})
	cases := []struct {
		name  string
		focus character.ShortRestFocus
		want  map[string]any
	}{
		{"accepted", focusOf(t, "n", character.RestPurposeAttune, character.InContact, true), map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}},
		{"rejected", focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "rest_interrupted"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.AttuneToMagicItemCommand{CharacterID: holder, Set: attunedSet(t, "cloak", "c1"), Item: item, Candidate: candidate(t, character.CanCast), Focus: c.focus, Limit: 3}
			rec := telemetrytest.NewRecorder()
			res, err := character.NewAttuneToMagicItem(character.WithTelemetry(rec.Telemetry())).Do(context.Background(), cmd)
			plain, err2 := character.NewAttuneToMagicItem().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("results differ: %#v %v vs %#v %v", res, err, plain, err2)
			}
			switch r := res.(type) {
			case character.AttuneToMagicItemApplied:
				ev := r.Events()
				if !r.Set().IsAttunedTo(inst("n")) || len(ev) != 1 || ev[0].CharacterID() != holder {
					t.Errorf("applied: %v %v", r.Set().Items(), ev)
				}
			case character.AttuneToMagicItemRejected:
				if r.Reason() != character.AttuneRestInterrupted {
					t.Errorf("reason = %v", r.Reason())
				}
			}
			checkOneCall(t, rec, character.SpanAttuneToMagicItem, "attune_magic_item", telemetry.StatusUnset, c.want)
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-040, FR-108 - the zero use case, an invalid command and a cancelled context are
// refused before any signal.
// Path: unhappy (invalid input).
func TestAttuneToMagicItemInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	use := character.NewAttuneToMagicItem(character.WithTelemetry(rec.Telemetry()))
	ok := character.AttuneToMagicItemCommand{CharacterID: holder, Item: attunable(t, "ring", "n", content.AttunementRequired{}), Candidate: candidate(t, character.CanCast),
		Focus: focusOf(t, "n", character.RestPurposeAttune, character.InContact, true), Limit: 3}
	if res, err := (character.AttuneToMagicItem{}).Do(context.Background(), ok); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", res, err)
	}
	if res, err := use.Do(context.Background(), character.AttuneToMagicItemCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v %v", res, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Do(cancelled, ok); res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v %v", res, err)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-052 to FR-060, FR-105, FR-109, FR-116 - EndMagicItemAttunement returns the new set
// and one MagicItemAttunementEnded per ended item carrying the character id; a rejection returns the reason with no event; both
// are recorded once and the no-op bundle returns the same result.
// Path: per case.
func TestEndMagicItemAttunementSignalsAndResults(t *testing.T) {
	t.Parallel()
	set := attunedSet(t, "ring", "r1", "cloak", "c1")
	cases := []struct {
		name  string
		means character.EndMeans
		want  map[string]any
	}{
		{"death", character.EndByTrigger{Trigger: character.AttunementEndDied{}}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}},
		{"not attuned", character.EndByTrigger{Trigger: character.AttunementEndAnotherCreatureAttuned{}}, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "not_attuned"}},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			instance := []string{"r1", "zz"}[i]
			cmd := character.EndMagicItemAttunementCommand{CharacterID: holder, Set: set, Instance: inst(instance), Curse: content.NotCursed, Means: c.means}
			rec := telemetrytest.NewRecorder()
			res, err := character.NewEndMagicItemAttunement(character.WithTelemetry(rec.Telemetry())).Do(context.Background(), cmd)
			plain, err2 := character.NewEndMagicItemAttunement().Do(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("results differ: %#v %v vs %#v %v", res, err, plain, err2)
			}
			switch r := res.(type) {
			case character.EndMagicItemAttunementApplied:
				if ev := r.Events(); r.Set().Len() != 0 || len(ev) != 2 || ev[0].CharacterID() != holder {
					t.Errorf("applied: %v %v", r.Set().Items(), ev)
				}
			case character.EndMagicItemAttunementRejected:
				if r.Reason() != character.EndNotAttuned {
					t.Errorf("reason = %v", r.Reason())
				}
			}
			checkOneCall(t, rec, character.SpanEndMagicItemAttunement, "end_magic_item_attunement", telemetry.StatusUnset, c.want)
		})
	}
	rec := telemetrytest.NewRecorder()
	use := character.NewEndMagicItemAttunement(character.WithTelemetry(rec.Telemetry()))
	if res, err := (character.EndMagicItemAttunement{}).Do(context.Background(), character.EndMagicItemAttunementCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", res, err)
	}
	if res, err := use.Do(context.Background(), character.EndMagicItemAttunementCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v %v", res, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	ok := character.EndMagicItemAttunementCommand{CharacterID: holder, Set: set, Instance: inst("r1"), Means: character.EndByTrigger{Trigger: character.AttunementEndDied{}}}
	if res, err := use.Do(cancelled, ok); res != nil || !errors.Is(err, context.Canceled) || len(rec.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v %v", res, err)
	}
}
