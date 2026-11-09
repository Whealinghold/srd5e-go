package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func cantripSource(t *testing.T) *contenttest.FakeSource {
	t.Helper()
	var entries []content.Entry
	for _, s := range cantripSpells(t) {
		entries = append(entries, s)
	}
	return contenttest.NewFakeSource(entries...)
}

// Requirement: doc/specs/species-trait-behaviour.md FR-025, FR-060, FR-061, FR-065, Table L "High Elf swap", "swap rejected" - the replacement is looked up through the content port before anything else (one lookup of the spell, whatever the state), then found, missing or refused by the domain, each a result and never a panic; equal with the no-op bundle; one replace span with a load_content child, one counter increment, content_missing with error status.
// Path: per case.
func TestReplaceHighElfCantrip(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		traits  character.SpeciesTraits
		spell   content.SpellID
		result  string
		outcome string
		reject  string
		status  telemetry.Status
	}{
		{"found", openHighElf(t), "light", "changed", "success", "", telemetry.StatusUnset},
		{"missing", openHighElf(t), "unknown_spell", "content_missing", "failure", "", telemetry.StatusError},
		{"not a cantrip", openHighElf(t), "shield", "rejected", "failure", "species_choices_invalid", telemetry.StatusUnset},
		{"window closed", spElf(t, character.ElvenHighElf), "light", "rejected", "failure", "cantrip_window_closed", telemetry.StatusUnset},
		{"a Drow", spElf(t, character.ElvenDrow), "light", "rejected", "failure", "species_use_not_offered", telemetry.StatusUnset},
		{"zero traits", character.SpeciesTraits{}, "light", "rejected", "failure", "species_behaviour_unknown", telemetry.StatusUnset},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ReplaceHighElfCantripCommand{Character: spChar(t, 5, 10), Traits: c.traits, Spell: c.spell}
			src := cantripSource(t)
			rec, opts := wzTelemetry()
			res, err := must(character.NewReplaceHighElfCantrip(src, opts...)).Run(context.Background(), cmd)
			plain, err2 := must(character.NewReplaceHighElfCantrip(cantripSource(t))).Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			switch r := res.(type) {
			case character.ReplaceHighElfCantripSucceeded:
				if c.result != "changed" || r.Event().New() != "light" || r.Traits().CantripWindowOpen() {
					t.Errorf("= %#v", r)
				}
			case character.ReplaceHighElfCantripRejected:
				if r.Rejection.String() != c.reject {
					t.Errorf("rejection = %v, want %s", r.Rejection, c.reject)
				}
			case character.ReplaceHighElfCantripContentMissing:
				if c.result != "content_missing" || !slices.Equal(r.Missing(), []content.MissingContent{c.spell.Missing()}) {
					t.Errorf("= %#v", r)
				}
				r.Missing()[0] = content.MissingContent{}
				if r.Missing()[0] == (content.MissingContent{}) {
					t.Error("Missing must return a copy")
				}
			}
			if keys := lookupKeys(src); len(keys) != 1 || keys[0] != c.spell.Key() {
				t.Errorf("lookups = %v", keys)
			}
			attrs := map[string]any{telemetry.AttrOutcome: c.outcome, character.AttrResult: c.result}
			if c.reject != "" {
				attrs[character.AttrRejection] = c.reject
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Name != character.SpanReplaceHighElfCantrip || spans[1].Name != character.SpanLoadContent || spans[0].Status != c.status {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "span", spans[0].Attrs, attrs)
			counter := map[string]any{character.AttrOperation: "replace_high_elf_cantrip"}
			for k, v := range attrs {
				counter[k] = v
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, counter)
			sameAttrs(t, "duration", onlyOne(t, rec, character.MetricDuration).Attrs, map[string]any{character.AttrOperation: "replace_high_elf_cantrip"})
			noLogsOrViolations(t, rec)
		})
	}
	if (character.ReplaceHighElfCantripContentMissing{}).Missing() != nil {
		t.Error("the zero result names nothing")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-061, FR-065 - a content port failure and a spell key answered with another entry type are a *srderr.PortError (content_error) recorded once with error status and no log; the cause stays reachable; invalid commands, the zero use case, a nil source and a cancelled context are refused before any lookup or signal.
// Path: unhappy (collaborator failure).
func TestReplaceHighElfCantripFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("down")
	cmd := character.ReplaceHighElfCantripCommand{Character: spChar(t, 5, 10), Traits: openHighElf(t), Spell: "light"}
	for name, src := range map[string]character.ContentSource{"fails": contenttest.NewFakeSource().FailLookup(cause), "another type": answering{entry: alertFeat()}} {
		rec, opts := wzTelemetry()
		res, err := must(character.NewReplaceHighElfCantrip(src, opts...)).Run(context.Background(), cmd)
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != "content_error" || name == "fails" && !errors.Is(err, cause) {
			t.Fatalf("%s: %v, %v", name, res, err)
		}
		if spans := rec.Tracer.Spans(); spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
			t.Errorf("%s: span %+v", name, spans[0])
		}
		sameAttrs(t, name+" operations", onlyOne(t, rec, character.MetricOperations).Attrs, map[string]any{character.AttrOperation: "replace_high_elf_cantrip", telemetry.AttrOutcome: "failure", telemetry.AttrErrorType: "content_error"})
		noLogsOrViolations(t, rec)
	}
	src := cantripSource(t)
	rec, opts := wzTelemetry()
	uc := must(character.NewReplaceHighElfCantrip(src, opts...))
	for name, mutate := range map[string]func(*character.ReplaceHighElfCantripCommand){
		"zero character": func(c *character.ReplaceHighElfCantripCommand) { c.Character = character.Character{} },
		"invalid spell":  func(c *character.ReplaceHighElfCantripCommand) { c.Spell = "" },
	} {
		bad := cmd
		mutate(&bad)
		if res, err := uc.Run(context.Background(), bad); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.ReplaceHighElfCantrip{}).Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	if _, err := character.NewReplaceHighElfCantrip(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil source: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, cmd); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(lookupKeys(src)) != 0 {
		t.Error("a refused call looked up content or emitted a signal")
	}
}
