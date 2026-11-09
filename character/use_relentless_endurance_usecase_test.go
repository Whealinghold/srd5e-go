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

// Requirement: doc/specs/species-trait-behaviour.md FR-041, Table E (SRD 5.2.1, "Relentless Endurance", p. 86; "Instant Death", p. 17) - an Orc reduced to 0 Hit Points and not killed drops to 1 and spends the use; a second drop is exhausted; after a Long Rest it works again; killed outright is not triggered and spends nothing; a Dwarf is refused; equal with the no-op bundle; success with result changed, success with not_applicable, or failure with the rejection.
// Path: per case.
func TestUseRelentlessEndurance(t *testing.T) {
	t.Parallel()
	notKilled, killed := character.ZeroHitPointsNotKilled{}, character.ZeroHitPointsKilledOutright{}
	for _, c := range []struct {
		name    string
		traits  character.SpeciesTraits
		outcome character.ZeroHitPointsOutcome
		result  string
		reject  string
	}{
		{"spent", spOrc(t), notKilled, "changed", ""},
		{"used twice", spExpended(t, spOrc(t), character.RelentlessEnduranceUse, 1), notKilled, "rejected", "species_use_exhausted"},
		{"killed outright", spOrc(t), killed, "not_applicable", ""},
		{"killed outright with the use spent is still not triggered", spExpended(t, spOrc(t), character.RelentlessEnduranceUse, 1), killed, "not_applicable", ""},
		{"not an Orc", spDwarf(t), notKilled, "rejected", "species_use_not_offered"},
		{"zero traits", character.SpeciesTraits{}, notKilled, "rejected", "species_behaviour_unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := character.UseRelentlessEnduranceCommand{Character: spChar(t, 5, 14), Traits: c.traits, Outcome: c.outcome}
			rec, opts := wzTelemetry()
			res, err := character.NewUseRelentlessEndurance(opts...).Run(context.Background(), cmd)
			plain, err2 := character.NewUseRelentlessEndurance().Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			attrs := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: c.result}
			switch r := res.(type) {
			case character.UseRelentlessEnduranceUsed:
				if c.result != "changed" || r.HitPointsInstead() != 1 || r.Traits().ExpendedOf(character.RelentlessEnduranceUse) != 1 || r.Event().CharacterID() != cmd.Character.ID() {
					t.Errorf("= %#v", r)
				}
				// Endurance after a Long Rest.
				renewed := must(character.SpeciesUses{}.Renew(cmd.Character, r.Traits(), spRest(t, character.LongRest, false))).(character.SpeciesTraitsChanged).Traits()
				again, err := character.NewUseRelentlessEndurance().Run(context.Background(), character.UseRelentlessEnduranceCommand{Character: cmd.Character, Traits: renewed, Outcome: notKilled})
				if _, ok := again.(character.UseRelentlessEnduranceUsed); !ok || err != nil {
					t.Errorf("after a Long Rest = %#v, %v", again, err)
				}
			case character.UseRelentlessEnduranceNotTriggered:
				if c.result != "not_applicable" {
					t.Errorf("= %#v", r)
				}
			case character.UseRelentlessEnduranceRejected:
				if r.Rejection.String() != c.reject {
					t.Errorf("rejection = %v, want %s", r.Rejection, c.reject)
				}
				attrs = map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.reject}
			}
			checkOneCall(t, rec, character.SpanUseRelentlessEndurance, "use_relentless_endurance", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-052, FR-060 - the zero use case, a zero character and a nil outcome are refused with an error matching srderr.ErrInvalidInput before any signal; a cancelled context is returned unwrapped.
// Path: unhappy (invalid input).
func TestUseRelentlessEnduranceInvalidInput(t *testing.T) {
	t.Parallel()
	rec, opts := wzTelemetry()
	uc := character.NewUseRelentlessEndurance(opts...)
	good := character.UseRelentlessEnduranceCommand{Character: spChar(t, 5, 10), Traits: spOrc(t), Outcome: character.ZeroHitPointsNotKilled{}}
	for name, cmd := range map[string]character.UseRelentlessEnduranceCommand{
		"zero character": {Traits: good.Traits, Outcome: good.Outcome},
		"nil outcome":    {Character: good.Character, Traits: good.Traits},
	} {
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	if res, err := (character.UseRelentlessEndurance{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
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
