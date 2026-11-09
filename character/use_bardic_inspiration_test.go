package character_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func useInspiration(t *testing.T, roller character.Roller, opts ...character.Option) character.UseBardicInspiration {
	t.Helper()
	return must(character.NewUseBardicInspiration(roller, opts...))
}

func inspirationCmd(t *testing.T, die character.BardicDie, elapsed time.Duration, failed character.AdjustableD20Result) character.UseBardicInspirationCommand {
	t.Helper()
	return character.UseBardicInspirationCommand{ID: brdID, Grant: must(character.NewBardicInspirationGrant(die)), Elapsed: elapsed, Failed: failed}
}

// Requirement: doc/specs/class-bard.md FR-030 to FR-032, FR-035, FR-119 (SRD 5.2.1, "Using Bardic Inspiration", pp. 31-32) - UseBardicInspiration rolls exactly one die of the held size through the dice port, adds it to the failed test and expends the die even
// when the test still fails; exactly one hour elapsed is accepted.
// Path: per case (happy / unhappy: boundary).
func TestUseBardicInspirationApplied(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		die     character.BardicDie
		elapsed time.Duration
		failed  character.AdjustableD20Result
		roll    int
		want    d20.Outcome
	}{
		{"turns the failure into a success", character.BardicD8, 30 * time.Minute, brdAbility(t, 10, 0, 15), 6, d20.Success},
		{"exactly one hour", character.BardicD6, time.Hour, brdAbility(t, 10, 0, 15), 5, d20.Success},
		{"still fails and the die is expended", character.BardicD10, 0, brdAbility(t, 5, 0, 15), 2, d20.Failure},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.roll)
			res, err := useInspiration(t, roller).Run(context.Background(), inspirationCmd(t, c.die, c.elapsed, c.failed))
			applied, ok := res.(character.UseBardicInspirationApplied)
			if err != nil || !ok {
				t.Fatalf("= %#v, %v", res, err)
			}
			if applied.Applied().Result().Outcome() != c.want || applied.Applied().Result().Total() != c.failed.Total()+c.roll {
				t.Errorf("result = %v total %d", applied.Applied().Result().Outcome(), applied.Applied().Result().Total())
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.MustDie(c.die.Sides()), 0) {
				t.Errorf("requests = %v, want one %v", reqs, c.die)
			}
			if ev := applied.Events(); len(ev) != 1 || ev[0].(character.BardicInspirationExpended).Rolled() != c.roll {
				t.Errorf("events = %v", ev)
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-033, FR-034, FR-116 - a test that was not failed is refused with no roll and the die stays (no grant to discard); a die held past one hour is refused as expired with no roll, carrying the grant to discard and the
// BardicInspirationExpired event, and a later attempt with the same grant is refused again.
// Path: per case (unhappy: state error, boundary).
func TestUseBardicInspirationRejected(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(4)
	uc := useInspiration(t, roller)
	notFailed := must(uc.Run(context.Background(), inspirationCmd(t, character.BardicD6, 0, brdAbility(t, 18, 0, 15)))).(character.UseBardicInspirationRejected)
	if notFailed.Reason != character.BardNotAFailure || !notFailed.Discard.IsZero() || notFailed.Expired != nil {
		t.Errorf("not failed = %+v", notFailed)
	}
	cmd := inspirationCmd(t, character.BardicD6, time.Hour+time.Second, brdAbility(t, 5, 0, 15))
	for range 2 {
		expired := must(uc.Run(context.Background(), cmd)).(character.UseBardicInspirationRejected)
		if expired.Reason != character.BardExpired || expired.Discard != cmd.Grant || expired.Expired == nil || expired.Expired.Die() != character.BardicD6 || expired.Expired.CharacterID() != brdID {
			t.Errorf("expired = %+v", expired)
		}
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("a refused use rolled: %v", roller.Requests())
	}
}

// Requirement: doc/specs/class-bard.md FR-036, FR-101 - a *srderr.PortError from the dice roller is returned unchanged with a nil result and the die unexpended; a zero use case, a zero grant, a nil result, a zero result and a cancelled context are
// errors, and a cancelled context returns ctx.Err() unwrapped.
// Path: per case (unhappy: collaborator failure, invalid input).
func TestUseBardicInspirationFailures(t *testing.T) {
	t.Parallel()
	port := &srderr.PortError{Port: "dice", Op: "roll", Err: errBoom}
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, port })
	res, err := useInspiration(t, failing).Run(context.Background(), inspirationCmd(t, character.BardicD6, 0, brdAbility(t, 5, 0, 15)))
	var perr *srderr.PortError
	if res != nil || !errors.As(err, &perr) || !errors.Is(err, errBoom) {
		t.Errorf("port failure = %v, %v", res, err)
	}
	good := inspirationCmd(t, character.BardicD6, 0, brdAbility(t, 5, 0, 15))
	uc := useInspiration(t, dicetest.NewFixedRoller(3))
	for name, run := range map[string]func() error{
		"zero use case": func() error { _, e := character.UseBardicInspiration{}.Run(context.Background(), good); return e },
		"zero grant": func() error {
			c := good
			c.Grant = character.BardicInspirationGrant{}
			_, e := uc.Run(context.Background(), c)
			return e
		},
		"nil result": func() error { c := good; c.Failed = nil; _, e := uc.Run(context.Background(), c); return e },
		"zero result": func() error {
			c := good
			c.Failed = character.AdjustableCheck{}
			_, e := uc.Run(context.Background(), c)
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, good); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: error = %v, want context.Canceled", err)
	}
	if _, err := character.NewUseBardicInspiration(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil roller: error = %v", err)
	}
}
