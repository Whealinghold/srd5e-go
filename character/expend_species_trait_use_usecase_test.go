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

// Requirement: doc/specs/species-trait-behaviour.md FR-031, FR-060, FR-063, FR-065, FR-066, section 7 - ExpendSpeciesTraitUse returns the new traits, the declared effect and one SpeciesUseExpended on success and the rejection otherwise; the no-op and a recording bundle give equal results; one span and one counter increment, outcome success with result changed, or failure with result rejected and the rejection value and no error status; the command is not altered and nothing is logged.
// Path: per case.
func TestExpendSpeciesTraitUse(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		level  int
		traits character.SpeciesTraits
		req    character.SpeciesUseRequest
		reject string
	}{
		{"accepted", 5, spDragonborn(t, character.DraconicRed), character.BreathWeaponRequest{Shape: character.BreathCone15}, ""},
		{"not offered", 5, spDwarf(t), character.BreathWeaponRequest{Shape: character.BreathCone15}, "species_use_not_offered"},
		{"own use case", 5, spOrc(t), character.RelentlessEnduranceRequest{}, "species_use_has_own_use_case"},
		{"not yet available", 4, spDragonborn(t, character.DraconicRed), character.DraconicFlightRequest{}, "species_use_not_yet_available"},
		{"no space", 5, spGoliath(t, character.AncestryHill), character.LargeFormRequest{}, "not_enough_space_for_large_form"},
		{"exhausted", 1, spExpended(t, spDwarf(t), character.StonecunningUse, 2), character.StonecunningRequest{}, "species_use_exhausted"},
		{"unknown species", 1, character.SpeciesTraits{}, character.StonecunningRequest{}, "species_behaviour_unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.ExpendSpeciesTraitUseCommand{Character: spChar(t, c.level, 14), Traits: c.traits, Request: c.req}
			snapshot := cmd
			rec, opts := wzTelemetry()
			res, err := character.NewExpendSpeciesTraitUse(opts...).Run(context.Background(), cmd)
			plain, err2 := character.NewExpendSpeciesTraitUse().Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) || !reflect.DeepEqual(cmd, snapshot) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			attrs := map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.reject}
			switch r := res.(type) {
			case character.ExpendSpeciesTraitUseSucceeded:
				if c.reject != "" || r.Traits().ExpendedOf(character.BreathWeaponUse) != 1 || r.Event().Use() != character.BreathWeaponUse || r.Effect() == nil {
					t.Errorf("= %#v", r)
				}
				attrs = map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "changed"}
			case character.ExpendSpeciesTraitUseRejected:
				if r.Rejection.String() != c.reject {
					t.Errorf("rejection = %v, want %s", r.Rejection, c.reject)
				}
			}
			checkOneCall(t, rec, character.SpanExpendSpeciesTraitUse, "expend_species_trait_use", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-052, FR-060, FR-065 - the zero use case, a zero character, a nil request and an invalid request are refused with an error matching srderr.ErrInvalidInput before any signal, and a cancelled context is returned unwrapped.
// Path: unhappy (invalid input).
func TestExpendSpeciesTraitUseInvalidInput(t *testing.T) {
	t.Parallel()
	rec, opts := wzTelemetry()
	uc := character.NewExpendSpeciesTraitUse(opts...)
	good := character.ExpendSpeciesTraitUseCommand{Character: spChar(t, 5, 10), Traits: spDwarf(t), Request: character.StonecunningRequest{}}
	for name, mutate := range map[string]func(*character.ExpendSpeciesTraitUseCommand){
		"zero character": func(c *character.ExpendSpeciesTraitUseCommand) { c.Character = character.Character{} },
		"nil request":    func(c *character.ExpendSpeciesTraitUseCommand) { c.Request = nil },
		"zero shape":     func(c *character.ExpendSpeciesTraitUseCommand) { c.Request = character.BreathWeaponRequest{} },
	} {
		cmd := good
		mutate(&cmd)
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.ExpendSpeciesTraitUse{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
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

// Requirement: doc/specs/species-trait-behaviour.md test plan, errors-and-results.md FR-051 (architecture.md section 6) - a panic in a telemetry double is the host's bug and propagates out of the use case instead of becoming a result or an error; the same holds for the other five use cases, which share the signal helper.
// Path: unhappy (collaborator failure).
func TestSpeciesUseCasePanicInTelemetryPropagates(t *testing.T) {
	t.Parallel()
	uc := character.NewExpendSpeciesTraitUse(character.WithTelemetry(telemetry.Telemetry{Tracer: panickingTracer{}}))
	defer func() {
		if recover() == nil {
			t.Error("the telemetry panic was swallowed")
		}
	}()
	_, _ = uc.Run(context.Background(), character.ExpendSpeciesTraitUseCommand{Character: spChar(t, 5, 10), Traits: spDwarf(t), Request: character.StonecunningRequest{}})
	t.Error("Run returned")
}
