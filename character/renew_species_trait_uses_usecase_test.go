package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-033, FR-045, FR-060, FR-065, section 7 - RenewSpeciesTraitUses returns the renewed traits, the event and the windows a Long Rest opened (the Human Resourceful window the host acts on), a successful no-op with no event when nothing changes, and a rejection for an event of another character or a zero species; equal with the no-op bundle; one span and one counter increment.
// Path: per case.
func TestRenewSpeciesTraitUses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		traits  character.SpeciesTraits
		rest    character.RestApplied
		events  int
		windows []character.SpeciesWindow
		reject  string
	}{
		{"Long Rest clears a use", spExpended(t, spDwarf(t), character.StonecunningUse, 2), spRest(t, character.LongRest, false), 1, nil, ""},
		{"Long Rest for a Human opens Resourceful", spHuman(t), spRest(t, character.LongRest, false), 1, []character.SpeciesWindow{character.WindowResourceful}, ""},
		{"nothing to clear", spDwarf(t), spRest(t, character.LongRest, false), 0, nil, ""},
		{"another character", spDwarf(t), fgtRest(t, character.MustIdentifier("someone-else"), character.LongRest), 0, nil, "character_mismatch"},
		{"zero traits", character.SpeciesTraits{}, spRest(t, character.LongRest, false), 0, nil, "species_behaviour_unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.RenewSpeciesTraitUsesCommand{Character: spChar(t, 5, 14), Traits: c.traits, Rest: c.rest}
			rec, opts := wzTelemetry()
			res, err := character.NewRenewSpeciesTraitUses(opts...).Run(context.Background(), cmd)
			plain, err2 := character.NewRenewSpeciesTraitUses().Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			attrs := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}
			switch r := res.(type) {
			case character.RenewSpeciesTraitUsesSucceeded:
				if c.reject != "" || len(r.Events()) != c.events || !reflect.DeepEqual(append([]character.SpeciesWindow{}, r.Windows()...), append([]character.SpeciesWindow{}, c.windows...)) {
					t.Errorf("= %#v", r)
				}
				if c.events == 0 && r.Traits() != c.traits {
					t.Error("a no-op changed the traits")
				}
			case character.RenewSpeciesTraitUsesRejected:
				if r.Rejection.String() != c.reject {
					t.Errorf("rejection = %v, want %s", r.Rejection, c.reject)
				}
				attrs = map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.reject}
			}
			checkOneCall(t, rec, character.SpanRenewSpeciesTraitUses, "renew_species_trait_uses", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-052, FR-060 - the zero use case and a zero character are refused with an error matching srderr.ErrInvalidInput before any signal; a cancelled context is returned unwrapped.
// Path: unhappy (invalid input).
func TestRenewSpeciesTraitUsesInvalidInput(t *testing.T) {
	t.Parallel()
	rec, opts := wzTelemetry()
	uc := character.NewRenewSpeciesTraitUses(opts...)
	good := character.RenewSpeciesTraitUsesCommand{Character: spChar(t, 5, 10), Traits: spDwarf(t), Rest: spRest(t, character.LongRest, false)}
	if res, err := uc.Run(context.Background(), character.RenewSpeciesTraitUsesCommand{Traits: good.Traits, Rest: good.Rest}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v, %v", res, err)
	}
	if res, err := (character.RenewSpeciesTraitUses{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, good); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a refused call emitted a signal")
	}
}
