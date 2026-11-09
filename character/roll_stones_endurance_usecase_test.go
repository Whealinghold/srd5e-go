package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/species-trait-behaviour.md FR-038, FR-054, FR-060, Table G "Stone roll cases", "Stone not offered", "Stone exhausted" (SRD 5.2.1, "Stone's Endurance", p. 85) - the use case rolls one 1d12, adds the Constitution modifier, never reduces below 0, spends a use of the shared pool only after a successful roll, and makes no request for a Dwarf, a Goliath with another ancestry, a spent pool or a zero species; equal with the no-op bundle; success with result resolved, or failure with the rejection.
// Path: per case.
func TestRollStonesEndurance(t *testing.T) {
	t.Parallel()
	stone := spGoliath(t, character.AncestryStone)
	for _, c := range []struct {
		name                       string
		traits                     character.SpeciesTraits
		con, roll, damage          int
		modifier, reduction, after int
		reject                     string
	}{
		{"roll 6, Con +2, damage 20", stone, 14, 6, 20, 2, 8, 12, ""},
		{"roll 12, Con +3, damage 10", stone, 16, 12, 10, 3, 15, 0, ""},
		{"roll 1, Con -3, damage 5", stone, 4, 1, 5, -3, 0, 5, ""},
		{"roll 1, Con -5, damage 9", stone, 1, 1, 9, -5, 0, 9, ""},
		{"damage 0", stone, 14, 5, 0, 2, 7, 0, ""},
		{"one use left of the pool", spExpended(t, stone, character.GiantAncestryUse, 1), 14, 6, 20, 2, 8, 12, ""},
		{"Dwarf", spDwarf(t), 14, 6, 20, 0, 0, 0, "species_use_not_offered"},
		{"Hill Goliath", spGoliath(t, character.AncestryHill), 14, 6, 20, 0, 0, 0, "species_use_not_offered"},
		{"pool spent", spExpended(t, stone, character.GiantAncestryUse, 2), 14, 6, 20, 0, 0, 0, "species_use_exhausted"},
		{"zero traits", character.SpeciesTraits{}, 14, 6, 20, 0, 0, 0, "species_behaviour_unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller, plain := dicetest.NewFixedRoller(c.roll), dicetest.NewFixedRoller(c.roll)
			pc := spChar(t, 1, c.con)
			cmd := must(character.NewRollStonesEnduranceCommand(pc, c.traits, c.damage))
			rec, opts := wzTelemetry()
			res, err := must(character.NewRollStonesEndurance(roller, opts...)).Run(context.Background(), cmd)
			again, err2 := must(character.NewRollStonesEndurance(plain)).Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, again) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, again, err2)
			}
			requests := 0
			attrs := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "resolved"}
			switch r := res.(type) {
			case character.RollStonesEnduranceRolled:
				requests = 1
				if r.Roll() != c.roll || r.Modifier() != c.modifier || r.Reduction() != c.reduction || r.DamageAfter() != c.after ||
					r.Traits().ExpendedOf(character.GiantAncestryUse) != c.traits.ExpendedOf(character.GiantAncestryUse)+1 ||
					r.Event().Roll() != c.roll || r.Event().Reduction() != c.reduction || r.Event().CharacterID() != pc.ID() {
					t.Errorf("= %#v", r)
				}
			case character.RollStonesEnduranceRejected:
				if r.Rejection.String() != c.reject {
					t.Errorf("rejection = %v, want %s", r.Rejection, c.reject)
				}
				attrs = map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.reject}
			}
			if got := roller.Requests(); len(got) != requests || requests == 1 && got[0] != dice.MustExpression(1, dice.D12(), 0) {
				t.Errorf("requests = %v, want %d of 1d12", got, requests)
			}
			checkOneCall(t, rec, character.SpanRollStonesEndurance, "roll_stones_endurance", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-038, FR-062, Table G "Stone dice failure" - a roller that fails, panics or answers another expression is a *srderr.PortError (dice_roller_error) with a nil result, recorded once with error status, and no use is spent; the cause stays reachable.
// Path: unhappy (collaborator failure).
func TestRollStonesEndurancePortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	cmd := must(character.NewRollStonesEnduranceCommand(spChar(t, 1, 14), spGoliath(t, character.AncestryStone), 10))
	for name, r := range failingRollers(cause) {
		rec, opts := wzTelemetry()
		res, err := must(character.NewRollStonesEndurance(r, opts...)).Run(context.Background(), cmd)
		wantPortFailure(t, name, "roll_stones_endurance", rec, res, err)
		if name == "fails" && !errors.Is(err, cause) {
			t.Error("the cause is lost")
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-052, FR-060, Table G "Stone zero and negative damage" - a negative damage and a zero character are refused by NewRollStonesEnduranceCommand with an error matching srderr.ErrInvalidInput; the zero command, the zero use case and a nil roller are refused before any signal or roll; a cancelled context is returned unwrapped.
// Path: unhappy (invalid input).
func TestRollStonesEnduranceInvalidInput(t *testing.T) {
	t.Parallel()
	stone := spGoliath(t, character.AncestryStone)
	if _, err := character.NewRollStonesEnduranceCommand(spChar(t, 1, 14), stone, -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("damage -1: %v", err)
	}
	if _, err := character.NewRollStonesEnduranceCommand(character.Character{}, stone, 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v", err)
	}
	roller := dicetest.NewFixedRoller(5)
	rec, opts := wzTelemetry()
	uc := must(character.NewRollStonesEndurance(roller, opts...))
	if res, err := uc.Run(context.Background(), character.RollStonesEnduranceCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v, %v", res, err)
	}
	good := must(character.NewRollStonesEnduranceCommand(spChar(t, 1, 14), stone, 3))
	if res, err := (character.RollStonesEndurance{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	if _, err := character.NewRollStonesEndurance(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, good); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Error("a refused call emitted a signal or rolled")
	}
}
