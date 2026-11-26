package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-061, FR-092, table 4.7a - the throw requests exactly 1d20
// (2d20 under Advantage or Disadvantage, the modifier never part of the expression), keeps the die the mode selects,
// and the total adds the host's modifier: the natural decides 1 and 20, the total decides 10.
// Path: per case.
func TestMakeDeathSavingThrowRolls(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		mode                    modifier.RollMode
		mod                     int
		faces                   []int
		request                 string
		outcome                 combat.DeathSavingThrowOutcome
	}{
		{"normal success", "FR-054", "happy", modifier.Normal, 0, []int{10}, "1d20", combat.DeathSavingThrowSuccess},
		{"normal failure", "FR-054", "unhappy (boundary)", modifier.Normal, 0, []int{9}, "1d20", combat.DeathSavingThrowFailure},
		{"Advantage keeps the higher die", `table 4.7a; "Roll Two D20s", p. 8`, "unhappy (rule interaction)", modifier.WithAdvantage, 0, []int{4, 15}, "2d20", combat.DeathSavingThrowSuccess},
		{"Disadvantage keeps the lower die", "FR-061", "unhappy (rule interaction)", modifier.WithDisadvantage, 0, []int{15, 4}, "2d20", combat.DeathSavingThrowFailure},
		{"Exhaustion turns a success into a failure", `table 4.7a; "Exhaustion", p. 181`, "unhappy (rule interaction)", modifier.Normal, -4, []int{11}, "1d20", combat.DeathSavingThrowFailure},
		{"a bonus lifts a failure to a success", "FR-061", "unhappy (rule interaction)", modifier.Normal, 3, []int{7}, "1d20", combat.DeathSavingThrowSuccess},
		{"natural 20 ignores a low total", "FR-056", "unhappy (boundary)", modifier.Normal, -30, []int{20}, "1d20", combat.DeathSavingThrowRegainedHitPoint},
		{"natural 1 with a high total", "FR-055", "unhappy (boundary)", modifier.Normal, 30, []int{1}, "1d20", combat.DeathSavingThrowTwoFailures},
		{"Advantage and natural 1 and 20 keeps the 20", "FR-056", "unhappy (rule interaction)", modifier.WithAdvantage, 0, []int{1, 20}, "2d20", combat.DeathSavingThrowRegainedHitPoint},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			cmd, err := combat.NewMakeDeathSavingThrowCommand(subject(), dyingStatus(t, 20, 0, 0), c.mode, modifier.New(c.mod))
			if err != nil {
				t.Fatal(err)
			}
			uc, err := combat.NewMakeDeathSavingThrow(roller)
			if err != nil {
				t.Fatal(err)
			}
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			if got := res.(combat.DeathSavingThrowApplied).Outcome(); got != c.outcome {
				t.Errorf("outcome = %v, want %v [%s]", got, c.outcome, c.requirement)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustParseExpression(c.request) {
				t.Errorf("requests = %v, want exactly [%s]", reqs, c.request)
			}
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-053, FR-068 - a creature that is not Dying is refused
// without rolling.
// Path: unhappy (state error).
func TestMakeDeathSavingThrowRefusalRollsNothing(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(12)
	cmd, _ := combat.NewMakeDeathSavingThrowCommand(subject(), pcStatus(t, 5, 20, 0), modifier.Normal, modifier.New(0))
	uc, _ := combat.NewMakeDeathSavingThrow(roller)
	res, err := uc.Run(context.Background(), cmd)
	if err != nil || res != (combat.DeathSavingThrowRejected{Rejection: combat.NotDying}) || len(roller.Requests()) != 0 {
		t.Errorf("= %v, %v with %d requests, want a refusal and no roll", res, err, len(roller.Requests()))
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-078, FR-080, FR-092 - a roller that fails, panics or answers
// another expression is a *srderr.PortError (dice_roller_error) with a nil result and no new state, recorded once
// on the span and once in srd5e.combat.errors, with no operations increment, no own signal and no log; the cause
// stays reachable.
// Path: unhappy (collaborator failure).
func TestMakeDeathSavingThrowPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	st := dyingStatus(t, 20, 1, 1)
	cmd, _ := combat.NewMakeDeathSavingThrowCommand(subject(), st, modifier.Normal, modifier.New(0))
	rollers := map[string]combat.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) }),
		"wrong expression": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustParseExpression("1d6"), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 3)})
		}),
	}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			uc, _ := combat.NewMakeDeathSavingThrow(roller, opts...)
			res, err := uc.Run(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port {
				t.Fatalf("= %v, %v, want nil and a dice_roller *PortError", res, err)
			}
			if name == "wrong expression" && !errors.Is(err, srderr.ErrContractViolation) || name == "fails" && !errors.Is(err, cause) {
				t.Errorf("cause not reachable: %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.make_death_saving_throw")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
			}
			errs := measured(rec, "srd5e.combat.errors")
			if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != "make_death_saving_throw" {
				t.Errorf("errors = %v, want one with error.type", errs)
			}
			for _, n := range []string{"operations", "events", "death_saves", "hit_point_refusals", "deaths"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
					t.Errorf("%s recorded %d on an error", n, len(m))
				}
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-108 - death_saves increments once per applied call with the
// result mapped from the outcome.
// Path: per case.
func TestMakeDeathSavingThrowCountsResults(t *testing.T) {
	t.Parallel()
	for face, want := range map[int]string{12: "success", 5: "failure", 1: "critical_failure", 20: "critical_success"} {
		rec, opts := recording()
		cmd, _ := combat.NewMakeDeathSavingThrowCommand(subject(), dyingStatus(t, 20, 0, 0), modifier.Normal, modifier.New(0))
		uc, _ := combat.NewMakeDeathSavingThrow(dicetest.NewFixedRoller(face), opts...)
		if _, err := uc.Run(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
		got := measured(rec, "srd5e.combat.death_saves")
		if len(got) != 1 || got[0]["srd5e.combat.death_save_result"] != want || len(got[0]) != 1 {
			t.Errorf("face %d: death_saves = %v, want %s", face, got, want)
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-094, architecture.md section 4 - the constructor rejects a nil
// roller, the command an undefined roll mode and a zero subject or status; the zero use case is invalid input.
// Path: unhappy (invalid input).
func TestMakeDeathSavingThrowConstructors(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewMakeDeathSavingThrow(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v", err)
	}
	dy := dyingStatus(t, 20, 0, 0)
	if _, err := combat.NewMakeDeathSavingThrowCommand(subject(), dy, modifier.RollMode(9), modifier.New(0)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("undefined mode error = %v", err)
	}
	if _, err := combat.NewMakeDeathSavingThrowCommand("", dy, modifier.Normal, modifier.New(0)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero subject error = %v", err)
	}
	cmd, _ := combat.NewMakeDeathSavingThrowCommand(subject(), dy, modifier.Normal, modifier.New(0))
	if res, err := (combat.MakeDeathSavingThrow{}).Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v, %v", res, err)
	}
}
