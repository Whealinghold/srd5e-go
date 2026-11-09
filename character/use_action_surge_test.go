package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

func useActionSurge(t *testing.T, opts ...character.Option) character.UseActionSurge {
	t.Helper()
	return must(character.NewUseActionSurge(fgtProgression(t), opts...))
}

// Requirement: doc/specs/class-fighter.md Table E, FR-071, FR-072, FR-073, FR-075, FR-076, FR-108, FR-129 - UseActionSurge applies on the Fighter's own turn with an unexpended
// use and no earlier use this turn (AS6, AS7: a second use of level 17 on a later turn), describes one additional action that is not the Magic action, expends exactly
// one use and emits ActionSurgeUsed; the other rows are refused with the first failing reason (AS1 to AS5, AS8) and leave the Character unchanged with no event.
// Path: per case (rows AS1 to AS8).
func TestUseActionSurgeTableE(t *testing.T) {
	t.Parallel()
	own, other := character.FighterOwnTurn, character.FighterOtherTurn
	unused, used := character.FighterActionSurgeUnused, character.FighterActionSurgeUsed
	for _, c := range []struct {
		name     string
		level    int
		turn     character.FighterTurnOwnership
		perTurn  character.FighterPerTurnUse
		expended int
		want     character.FighterReason
	}{
		{"AS1", 1, own, unused, 0, character.FighterFeatureNotGained},
		{"AS2", 2, other, used, 1, character.FighterNotOwnTurn},
		{"AS3", 17, own, used, 0, character.FighterAlreadyUsedThisTurn},
		{"AS4", 17, own, used, 2, character.FighterAlreadyUsedThisTurn},
		{"AS5", 2, own, unused, 1, character.FighterNoUsesRemaining},
		{"AS6", 2, own, unused, 0, 0},
		{"AS7", 17, own, unused, 1, 0},
		{"AS8", 0, own, unused, 0, character.FighterNotAFighter},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := fgtChar(t, cl("wizard", 5))
			if c.level > 0 {
				in = fgtWith(t, fgtFighter(t, c.level), fgtState(1, c.expended, 1))
			}
			res, err := useActionSurge(t).Run(context.Background(), character.UseActionSurgeCommand{Character: in, Turn: c.turn, PerTurn: c.perTurn})
			if err != nil {
				t.Fatal(err)
			}
			switch r := res.(type) {
			case character.ActionSurgeNotAvailable:
				if r.Reason != c.want {
					t.Errorf("not available %v, want %v", r.Reason, c.want)
				}
			case character.ActionSurgeApplied:
				ev, ok := r.Events()[0].(character.ActionSurgeUsed)
				if c.want != 0 || fgtStateOf(t, r.Character()) != fgtState(1, c.expended+1, 1) || r.AdditionalAction() != character.FighterAnyActionButMagic || !ok || len(r.Events()) != 1 || ev.CharacterID() != in.ID() {
					t.Errorf("applied: state %v events %v", fgtStateOf(t, r.Character()), r.Events())
				}
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md FR-107, FR-112, FR-149, FR-073, FR-152 - the result is the same with the no-op and a recording telemetry; an absent state is the initial state;
// the constructor refuses the zero progression; Run refuses the zero use case, a zero Character, an invalid turn or per-turn value and a cancelled context, and a foreign
// state under the Fighter key.
// Path: per case (unhappy: invalid input, cancellation, state error).
func TestUseActionSurgeInvalidInputAndNoop(t *testing.T) {
	t.Parallel()
	good := character.UseActionSurgeCommand{Character: fgtFighter(t, 5), Turn: character.FighterOwnTurn, PerTurn: character.FighterActionSurgeUnused}
	_, opts := fgtTelemetry()
	withRec, err := useActionSurge(t, opts...).Run(context.Background(), good)
	plain, err2 := useActionSurge(t).Run(context.Background(), good)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v vs %#v", withRec, plain)
	}
	if got := fgtStateOf(t, withRec.(character.ActionSurgeApplied).Character()); got != fgtState(0, 1, 0) {
		t.Errorf("state from an absent one = %v", got)
	}
	_, err = character.NewUseActionSurge(character.FighterProgression{})
	wantInvalid(t, err, "zero progression")
	uc := useActionSurge(t)
	foreign := must(character.WithFeatureState(good.Character, character.MustFeatureStateKey[character.ChosenSubclass]("fighter"), must(character.NewChosenSubclass("champion"))))
	for name, run := range map[string]func() (character.UseActionSurgeResult, error){
		"zero use case": func() (character.UseActionSurgeResult, error) {
			return character.UseActionSurge{}.Run(context.Background(), good)
		},
		"zero character": func() (character.UseActionSurgeResult, error) {
			return uc.Run(context.Background(), character.UseActionSurgeCommand{Turn: good.Turn, PerTurn: good.PerTurn})
		},
		"invalid turn": func() (character.UseActionSurgeResult, error) {
			return uc.Run(context.Background(), character.UseActionSurgeCommand{Character: good.Character, PerTurn: good.PerTurn})
		},
		"invalid per turn": func() (character.UseActionSurgeResult, error) {
			return uc.Run(context.Background(), character.UseActionSurgeCommand{Character: good.Character, Turn: good.Turn})
		},
		"foreign state": func() (character.UseActionSurgeResult, error) {
			return uc.Run(context.Background(), character.UseActionSurgeCommand{Character: foreign, Turn: good.Turn, PerTurn: good.PerTurn})
		},
	} {
		if r, err := run(); r != nil {
			t.Errorf("%s: result %v", name, r)
		} else {
			wantInvalid(t, err, name)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := uc.Run(cancelled, good); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, %v", r, err)
	}
}
