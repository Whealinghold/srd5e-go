package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func useEndPalm(t *testing.T, roller character.Roller, opts ...character.Option) character.EndQuiveringPalm {
	t.Helper()
	return must(character.NewEndQuiveringPalm(mnkProgression(t), roller, opts...))
}

// palmMonk is a level 17 Monk with vibrations active and 4 Focus Points expended.
func palmMonk(t *testing.T) character.Character {
	t.Helper()
	return mnkWith(t, mnkMonk(t, 17), character.MustMonkState(4, false, 0, mnkVibrations(17)))
}

// Requirement: doc/specs/class-monk.md Table 4.10c, FR-117 to FR-120, FR-122, FR-141 - EndQuiveringPalm with effect rolls 10d12 (one request), the target saves against the
// Monk's Focus save DC (8 + 3 + 6 = 17): Force damage is the total on a failure and half on a success; ending harmlessly rolls nothing, deals nothing and has no DC; both clear the
// vibrations, keep the expended points and emit QuiveringPalmEnded.
// Path: per case (rows of Table 4.10c).
func TestEndQuiveringPalmApplied(t *testing.T) {
	t.Parallel()
	d12 := func(n int) []int { // ten dice whose sum is n
		out := make([]int, 10)
		for i := range out {
			out[i] = 1
		}
		for left, i := n-10, 0; left > 0; i++ {
			add := min(left, 11)
			out[i] += add
			left -= add
		}
		return out
	}
	for _, c := range []struct {
		name    string
		end     character.QuiveringPalmEnd
		outcome d20.Outcome
		total   int
		damage  int
		dc      d20.DifficultyClass
	}{
		{"failed, lowest", character.QuiveringPalmEndWithEffect, d20.Failure, 10, 10, 17}, {"failed, highest", character.QuiveringPalmEndWithEffect, d20.Failure, 120, 120, 17},
		{"successful, even total", character.QuiveringPalmEndWithEffect, d20.Success, 20, 10, 17}, {"successful, lowest", character.QuiveringPalmEndWithEffect, d20.Success, 10, 5, 17},
		{"harmless", character.QuiveringPalmEndHarmlessly, 0, 0, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := palmMonk(t)
			var roller *dicetest.FixedRoller
			if c.total > 0 {
				roller = dicetest.NewFixedRoller(d12(c.total)...)
			} else {
				roller = dicetest.NewFixedRoller()
			}
			res, err := useEndPalm(t, roller).Run(context.Background(), character.EndQuiveringPalmCommand{Character: in, End: c.end, SamePlane: true, Outcome: c.outcome})
			r, ok := res.(character.QuiveringPalmEndApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if r.Resolution().DamageApplied() != c.damage || r.Resolution().DifficultyClass() != c.dc || mnkStateOf(t, r.Character()) != mnkState(4, false, 0) || !mnkStateOf(t, in).Vibrations().Active() {
				t.Errorf("= %+v, state %v", r.Resolution(), mnkStateOf(t, r.Character()))
			}
			ev, isEv := r.Events()[0].(character.QuiveringPalmEnded)
			if len(r.Events()) != 1 || !isEv || ev.DamageApplied() != c.damage || ev.WithEffect() != (c.end == character.QuiveringPalmEndWithEffect) || ev.CharacterID() != in.ID() {
				t.Errorf("events = %#v", r.Events())
			}
			wantReqs := 0
			if c.total > 0 {
				wantReqs = 1
			}
			if reqs := roller.Requests(); len(reqs) != wantReqs || (wantReqs == 1 && reqs[0] != dice.MustExpression(10, dice.D12(), 0)) {
				t.Errorf("requests = %v", reqs)
			}
		})
	}
}

// Requirement: doc/specs/class-monk.md FR-119, FR-123, FR-023, FR-135 - ending with none active is notEligible, ending with effect on another plane is notEligible and the vibrations
// stay, no Monk levels is notAMonk; none rolls; an outcome that is neither success nor failure for an end with effect is an invalid command.
// Path: per case (unhappy: state error).
func TestEndQuiveringPalmNotAvailable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		char  character.Character
		end   character.QuiveringPalmEnd
		plane bool
		want  character.MonkReason
	}{
		{"none active", mnkMonk(t, 17), character.QuiveringPalmEndHarmlessly, true, character.MonkNotEligible},
		{"another plane", palmMonk(t), character.QuiveringPalmEndWithEffect, false, character.MonkNotEligible},
		{"an unknown way", palmMonk(t), 0, true, character.MonkNotEligible},
		{"no Monk levels", mnkChar(t, cl("fighter", 17)), character.QuiveringPalmEndHarmlessly, true, character.MonkNotAMonk},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller()
			res, err := useEndPalm(t, roller).Run(context.Background(), character.EndQuiveringPalmCommand{Character: c.char, End: c.end, SamePlane: c.plane, Outcome: d20.Failure})
			if got, ok := res.(character.QuiveringPalmEndNotAvailable); err != nil || !ok || got.Reason != c.want || len(roller.Requests()) != 0 {
				t.Fatalf("= %#v, %v; want not available %v", res, err, c.want)
			}
		})
	}
	res, err := useEndPalm(t, dicetest.NewFixedRoller()).Run(context.Background(), character.EndQuiveringPalmCommand{Character: palmMonk(t), End: character.QuiveringPalmEndWithEffect, SamePlane: true})
	wantInvalid(t, err, "outcome")
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
	foreign := must(character.WithFeatureState(mnkMonk(t, 17), character.MustFeatureStateKey[character.ChosenSubclass]("monk"), must(character.NewChosenSubclass("warrior_of_the_open_hand"))))
	_, err = useEndPalm(t, dicetest.NewFixedRoller()).Run(context.Background(), character.EndQuiveringPalmCommand{Character: foreign, End: character.QuiveringPalmEndHarmlessly, SamePlane: true})
	wantInvalid(t, err, "foreign state")
}

// Requirement: doc/specs/class-monk.md FR-124, FR-126, FR-129, FR-147 - a rest does not end the vibrations; a failing dice port is a *srderr.PortError with the cause and the vibrations stay;
// equal results with the no-op and a recording telemetry; invalid construction, the zero use case, a zero Character and a cancelled context are refused before any signal; the signals of
// success, rejection and port failure.
// Path: per case (unhappy: collaborator failure, invalid input, cancellation).
func TestEndQuiveringPalmFailuresAndSignals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cmd := character.EndQuiveringPalmCommand{Character: palmMonk(t), End: character.QuiveringPalmEndWithEffect, SamePlane: true, Outcome: d20.Failure}
	for _, kind := range []character.RestKind{character.ShortRest, character.LongRest} {
		if !mnkStateOf(t, palmMonk(t)).AfterRest(kind).Vibrations().Active() {
			t.Errorf("a %v ended the vibrations", kind)
		}
	}
	for _, c := range mnkFailingRollers() {
		res, err := useEndPalm(t, c.roller).Run(ctx, cmd)
		mnkWantPortFailure(t, res, err, c.cause)
	}
	rec, opts := mnkTelemetry()
	withRec, err := useEndPalm(t, dicetest.NewMaximumRoller(), opts...).Run(ctx, cmd)
	plain, err2 := useEndPalm(t, dicetest.NewMaximumRoller()).Run(ctx, cmd)
	if err != nil || err2 != nil || !reflect.DeepEqual(withRec, plain) {
		t.Errorf("results differ: %#v %v vs %#v %v", withRec, err, plain, err2)
	}
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanEndQuiveringPalm, feature: "quivering_palm_end", status: telemetry.StatusUnset, outcome: "success", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useEndPalm(t, dicetest.NewMaximumRoller(), opts...).Run(ctx, character.EndQuiveringPalmCommand{Character: mnkMonk(t, 17), End: character.QuiveringPalmEndHarmlessly, SamePlane: true})
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanEndQuiveringPalm, feature: "quivering_palm_end", status: telemetry.StatusUnset, outcome: "failure", rejection: "not_eligible", logged: true})
	rec, opts = mnkTelemetry()
	_, _ = useEndPalm(t, dicetest.NewFixedRoller(), opts...).Run(ctx, cmd)
	checkFeatureCall(t, rec, wantCall{class: "monk", span: character.SpanEndQuiveringPalm, feature: "quivering_palm_end", status: telemetry.StatusError, outcome: "failure", errorType: "dice_roller_error"})
	_, err = character.NewEndQuiveringPalm(character.MonkProgression{}, dicetest.NewFixedRoller(1))
	wantInvalid(t, err, "zero progression")
	_, err = character.NewEndQuiveringPalm(mnkProgression(t), nil)
	wantInvalid(t, err, "nil roller")
	_, err = character.EndQuiveringPalm{}.Run(ctx, cmd)
	wantInvalid(t, err, "zero use case")
	_, err = useEndPalm(t, dicetest.NewFixedRoller(1)).Run(ctx, character.EndQuiveringPalmCommand{End: character.QuiveringPalmEndHarmlessly})
	wantInvalid(t, err, "zero character")
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec2, opts2 := mnkTelemetry()
	res, err := useEndPalm(t, dicetest.NewFixedRoller(1), opts2...).Run(cancelled, cmd)
	if res != nil || !errors.Is(err, context.Canceled) || len(rec2.Tracer.Spans()) != 0 {
		t.Errorf("cancelled: %v, %v, spans %d", res, err, len(rec2.Tracer.Spans()))
	}
}

// Requirement: doc/specs/class-monk.md FR-141 - an end with effect whose Outcome is neither success nor failure (zero, or above them) is refused before any signal and any roll, while a harmless end
// ignores the Outcome.
// Path: per case (unhappy: invalid input, boundary).
func TestEndQuiveringPalmRefusesBadOutcomeBeforeAnySignal(t *testing.T) {
	t.Parallel()
	for _, bad := range []d20.Outcome{0, 3, -1} {
		rec, opts := mnkTelemetry()
		roller := dicetest.NewMaximumRoller()
		res, err := useEndPalm(t, roller, opts...).Run(context.Background(), character.EndQuiveringPalmCommand{Character: palmMonk(t), End: character.QuiveringPalmEndWithEffect, SamePlane: true, Outcome: bad})
		wantInvalid(t, err, "outcome")
		if res != nil || len(rec.Tracer.Spans()) != 0 || len(roller.Requests()) != 0 {
			t.Errorf("outcome %d: result %v, spans %d, requests %v; want nothing", int(bad), res, len(rec.Tracer.Spans()), roller.Requests())
		}
	}
	res, err := useEndPalm(t, dicetest.NewFixedRoller()).Run(context.Background(), character.EndQuiveringPalmCommand{Character: palmMonk(t), End: character.QuiveringPalmEndHarmlessly, SamePlane: true, Outcome: 7})
	if _, ok := res.(character.QuiveringPalmEndApplied); err != nil || !ok {
		t.Errorf("a harmless end with an unused outcome = %#v, %v", res, err)
	}
}
