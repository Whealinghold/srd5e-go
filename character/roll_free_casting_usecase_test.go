package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func recallFeats(t *testing.T) character.TakenFeats {
	t.Helper()
	return heldFeats(t, recallFeat(), singleOf(ability.Wisdom))
}

func freeCastingCmd(t *testing.T, feats character.TakenFeats, level int) character.RollFreeCastingCommand {
	t.Helper()
	return character.RollFreeCastingCommand{Character: charAt(t, 20), Feats: feats, SlotLevel: character.MustSlotLevel(level)}
}

// Requirement: doc/specs/feats-behaviour.md FR-053 to FR-055, FR-071, FR-073, Table F - RollFreeCasting makes exactly one request of 1d4 for a slot of level 1 to 4 and resolves it; a slot of level 5 to 9 makes none and is not_applicable (a success); a character without the Boon is rejected with FeatNotHeld (a failure) and no request; the no-op and a recording bundle agree; one span, one counter increment, no log.
// Path: per case.
func TestRollFreeCasting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		feats     character.TakenFeats
		level     int
		roll      int
		requests  int
		result    string
		outcome   string
		reject    string
		resolved  character.FreeCastingOutcome
		notAbove4 bool
	}{
		{"level 1, roll 1", recallFeats(t), 1, 1, 1, "resolved", "success", "", character.FreeCastingSpared, false},
		{"level 2, roll 2", recallFeats(t), 2, 2, 1, "resolved", "success", "", character.FreeCastingSpared, false},
		{"level 3, roll 4", recallFeats(t), 3, 4, 1, "resolved", "success", "", character.FreeCastingSpent, false},
		{"level 4, roll 3", recallFeats(t), 4, 3, 1, "resolved", "success", "", character.FreeCastingSpent, false},
		{"level 5", recallFeats(t), 5, 1, 0, "not_applicable", "success", "", 0, true},
		{"level 9", recallFeats(t), 9, 1, 0, "not_applicable", "success", "", 0, true},
		{"not held", character.TakenFeats{}, 2, 2, 0, "rejected", "failure", "feat_not_held", 0, false},
		{"not held, level 5", character.TakenFeats{}, 5, 2, 0, "rejected", "failure", "feat_not_held", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roller, plainRoller := dicetest.NewFixedRoller(tc.roll), dicetest.NewFixedRoller(tc.roll)
			rec := telemetrytest.NewRecorder()
			clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
			cmd := freeCastingCmd(t, tc.feats, tc.level)
			res, err := must(character.NewRollFreeCasting(roller, character.WithTelemetry(rec.Telemetry()), character.WithClock(clock.Now))).Run(context.Background(), cmd)
			plain, err2 := must(character.NewRollFreeCasting(plainRoller)).Run(context.Background(), cmd)
			if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
				t.Fatalf("= %#v, %v (plain %#v, %v)", res, err, plain, err2)
			}
			if got := roller.Requests(); len(got) != tc.requests || tc.requests == 1 && got[0] != dice.MustExpression(1, dice.D4(), 0) {
				t.Errorf("requests = %v, want %d of 1d4", got, tc.requests)
			}
			attrs := map[string]any{telemetry.AttrOutcome: tc.outcome, character.AttrResult: tc.result}
			switch r := res.(type) {
			case character.RollFreeCastingResolved:
				if r.Roll() != tc.roll || r.Outcome() != tc.resolved {
					t.Errorf("= %#v", r)
				}
			case character.RollFreeCastingNotApplicable:
				if !tc.notAbove4 || r.SlotLevel().Value() != tc.level {
					t.Errorf("= %#v", r)
				}
			case character.RollFreeCastingRejected:
				if r.Rejection != (character.FeatNotHeld{Feat: "boon_of_spell_recall"}) {
					t.Errorf("= %#v", r)
				}
				attrs[character.AttrRejection] = tc.reject
			}
			checkOneCall(t, rec, character.SpanRollFreeCasting, "roll_free_casting", telemetry.StatusUnset, attrs)
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-056, FR-078, FR-074, Table F "roller fails" - a roller that fails, panics or answers another expression is a *srderr.PortError (dice_roller_error) with a nil result, recorded once with error status, one counter increment with error.type and no log; the cause stays reachable.
// Path: per case.
func TestRollFreeCastingPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	for name, r := range failingRollers(cause) {
		rec := telemetrytest.NewRecorder()
		res, err := must(character.NewRollFreeCasting(r, character.WithTelemetry(rec.Telemetry()))).Run(context.Background(), freeCastingCmd(t, recallFeats(t), 2))
		wantPortFailure(t, name, "roll_free_casting", rec, res, err)
		if name == "fails" && !errors.Is(err, cause) {
			t.Error("the cause is lost")
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-059, FR-077, FR-071 - the zero slot level, a zero character and the zero use case are refused before any signal or roll, a nil roller is refused by the constructor, and a cancelled context is returned unwrapped; a level outside 1 to 9 is not constructible.
// Path: unhappy (invalid input).
func TestRollFreeCastingInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller(1)
	uc := must(character.NewRollFreeCasting(roller, character.WithTelemetry(rec.Telemetry())))
	good := freeCastingCmd(t, recallFeats(t), 2)
	for name, cmd := range map[string]character.RollFreeCastingCommand{
		"zero slot level": {Character: good.Character, Feats: good.Feats},
		"zero character":  {Feats: good.Feats, SlotLevel: good.SlotLevel},
	} {
		if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, res, err)
		}
	}
	for _, n := range []int{0, 10} {
		if _, err := character.NewSlotLevel(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("slot level %d: %v", n, err)
		}
	}
	if res, err := (character.RollFreeCasting{}).Run(context.Background(), good); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, %v", res, err)
	}
	if _, err := character.NewRollFreeCasting(nil); !errors.Is(err, srderr.ErrInvalidInput) {
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
