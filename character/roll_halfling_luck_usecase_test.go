package character_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func luckCmd(t *testing.T, traits character.SpeciesTraits, roll d20.Roll, die int, rerolled ...int) character.RollHalflingLuckCommand {
	t.Helper()
	return must(character.NewRollHalflingLuckCommand(spChar(t, 5, 10), traits, roll, die, rerolled))
}

// Requirement: doc/specs/species-trait-behaviour.md FR-042 to FR-045, FR-054, FR-060, Table H (SRD 5.2.1, "Luck", p. 86; "Interactions with Rerolls", p. 8) - a die showing 1 is rerolled with one request of 1d20 and the new face stands even when equal or lower; under Advantage the chosen die is replaced and the natural value re-selected; a die that is not 1 makes no request; a die already rerolled, a non-Halfling and a zero species are rejected without a request; equal with the no-op bundle; success with result resolved or not_applicable, or failure with the rejection.
// Path: per case.
func TestRollHalflingLuck(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name     string
		traits   character.SpeciesTraits
		roll     d20.Roll
		die      int
		rerolled []int
		face     int
		result   string
		reject   string
		natural  int
		dice     []int
	}{
		{"single 1", spHalfling(t), d20.MustRoll(modifier.Normal, 1), 0, nil, 14, "resolved", "", 14, []int{14}},
		{"advantage, the 1 rerolled to 3", spHalfling(t), d20.MustRoll(modifier.WithAdvantage, 1, 14), 0, nil, 3, "resolved", "", 14, []int{3, 14}},
		{"advantage, the 1 rerolled to 19", spHalfling(t), d20.MustRoll(modifier.WithAdvantage, 1, 14), 0, nil, 19, "resolved", "", 19, []int{19, 14}},
		{"two 1s, one rerolled", spHalfling(t), d20.MustRoll(modifier.WithAdvantage, 1, 1), 1, nil, 9, "resolved", "", 9, []int{1, 9}},
		{"new roll stands when it is 1 again", spHalfling(t), d20.MustRoll(modifier.Normal, 1), 0, nil, 1, "resolved", "", 1, []int{1}},
		{"single 2", spHalfling(t), d20.MustRoll(modifier.Normal, 2), 0, nil, 14, "not_applicable", "", 2, []int{2}},
		{"the other die is not a 1", spHalfling(t), d20.MustRoll(modifier.WithAdvantage, 1, 14), 1, nil, 5, "not_applicable", "", 14, []int{1, 14}},
		{"die already rerolled", spHalfling(t), d20.MustRoll(modifier.Normal, 1), 0, []int{0}, 14, "rejected", "die_already_rerolled", 1, []int{1}},
		{"after Inspiration, other die free", spHalfling(t), d20.MustRoll(modifier.WithAdvantage, 1, 1), 0, []int{1}, 7, "resolved", "", 7, []int{7, 1}},
		{"not a Halfling", spHuman(t), d20.MustRoll(modifier.Normal, 1), 0, nil, 14, "rejected", "species_use_not_offered", 1, []int{1}},
		{"zero traits", character.SpeciesTraits{}, d20.MustRoll(modifier.Normal, 1), 0, nil, 14, "rejected", "species_behaviour_unknown", 1, []int{1}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller, plain := dicetest.NewFixedRoller(c.face), dicetest.NewFixedRoller(c.face)
			cmd := luckCmd(t, c.traits, c.roll, c.die, c.rerolled...)
			rec, opts := wzTelemetry()
			res, err := must(character.NewRollHalflingLuck(roller, opts...)).Run(context.Background(), cmd)
			again, err2 := must(character.NewRollHalflingLuck(plain)).Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, again) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, again, err2)
			}
			wantRequests := 0
			attrs := map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: c.result}
			switch r := res.(type) {
			case character.RollHalflingLuckRerolled:
				wantRequests = 1
				if r.NewValue() != c.face || r.Roll().Natural() != c.natural || !slices.Equal(r.Roll().Dice(), c.dice) || !slices.Contains(r.RerolledIndexes(), c.die) ||
					r.Event().DieIndex() != c.die || r.Event().NewValue() != c.face || r.Event().CharacterID() != spChar(t, 5, 10).ID() {
					t.Errorf("= %#v", r)
				}
			case character.RollHalflingLuckNotApplicable:
				if c.result != "not_applicable" || r.Face() != c.roll.Dice()[c.die] {
					t.Errorf("= %#v", r)
				}
			case character.RollHalflingLuckRejected:
				if r.Rejection.String() != c.reject {
					t.Errorf("rejection = %v, want %s", r.Rejection, c.reject)
				}
				attrs = map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.reject}
			}
			if got := roller.Requests(); len(got) != wantRequests || wantRequests == 1 && got[0] != dice.MustExpression(1, dice.D20(), 0) {
				t.Errorf("requests = %v, want %d of 1d20", got, wantRequests)
			}
			checkOneCall(t, rec, character.SpanRollHalflingLuck, "roll_halfling_luck", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-043, Table H "second Luck call" (SRD 5.2.1, "Luck", p. 86) - the returned RerolledIndexes passed back refuse a second reroll of that die with DieAlreadyRerolled, the new roll standing; the returned slice is a copy.
// Path: unhappy (state error).
func TestRollHalflingLuckSecondCall(t *testing.T) {
	t.Parallel()
	uc := must(character.NewRollHalflingLuck(dicetest.NewFixedRoller(1, 8)))
	first := must(uc.Run(context.Background(), luckCmd(t, spHalfling(t), d20.MustRoll(modifier.Normal, 1), 0))).(character.RollHalflingLuckRerolled)
	if first.NewValue() != 1 || first.Roll().Natural() != 1 {
		t.Fatalf("first = %#v", first)
	}
	idx := first.RerolledIndexes()
	idx[0] = 5
	if first.RerolledIndexes()[0] != 0 {
		t.Error("RerolledIndexes is not a copy")
	}
	second := must(uc.Run(context.Background(), luckCmd(t, spHalfling(t), first.Roll(), 0, first.RerolledIndexes()...)))
	if rj, ok := second.(character.RollHalflingLuckRejected); !ok || rj.Rejection != (character.DieAlreadyRerolled{Index: 0}) {
		t.Errorf("second call = %#v", second)
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-062, FR-065 - a roller that fails, panics or answers another expression is a *srderr.PortError (dice_roller_error) with a nil result, recorded once with error status; the cause stays reachable.
// Path: unhappy (collaborator failure).
func TestRollHalflingLuckPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	for name, r := range failingRollers(cause) {
		rec, opts := wzTelemetry()
		res, err := must(character.NewRollHalflingLuck(r, opts...)).Run(context.Background(), luckCmd(t, spHalfling(t), d20.MustRoll(modifier.Normal, 1), 0))
		wantPortFailure(t, name, "roll_halfling_luck", rec, res, err)
		if name == "fails" && !errors.Is(err, cause) {
			t.Error("the cause is lost")
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-052, FR-060 - NewRollHalflingLuckCommand rejects a zero character or roll, a die index outside the roll and a rerolled index outside the roll; the zero command, the zero use case and a nil roller are refused before any signal or roll; a cancelled context is returned unwrapped.
// Path: unhappy (invalid input).
func TestRollHalflingLuckInvalidInput(t *testing.T) {
	t.Parallel()
	pc, roll := spChar(t, 5, 10), d20.MustRoll(modifier.WithAdvantage, 1, 14)
	for name, args := range map[string]struct {
		c        character.Character
		roll     d20.Roll
		die      int
		rerolled []int
	}{
		"zero character": {character.Character{}, roll, 0, nil}, "zero roll": {pc, d20.Roll{}, 0, nil},
		"negative die": {pc, roll, -1, nil}, "die past the roll": {pc, roll, 2, nil},
		"rerolled past the roll": {pc, roll, 0, []int{2}}, "negative rerolled": {pc, roll, 0, []int{-1}},
	} {
		if _, err := character.NewRollHalflingLuckCommand(args.c, spHalfling(t), args.roll, args.die, args.rerolled); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	roller := dicetest.NewFixedRoller(5)
	rec, opts := wzTelemetry()
	uc := must(character.NewRollHalflingLuck(roller, opts...))
	if res, err := uc.Run(context.Background(), character.RollHalflingLuckCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v, %v", res, err)
	}
	if res, err := (character.RollHalflingLuck{}).Run(context.Background(), luckCmd(t, spHalfling(t), roll, 0)); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	if _, err := character.NewRollHalflingLuck(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(cancelled, luckCmd(t, spHalfling(t), roll, 0)); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Error("a refused call emitted a signal or rolled")
	}
}
