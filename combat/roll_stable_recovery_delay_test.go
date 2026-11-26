package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-072, table 4.8a - the use case requests exactly 1d4 once and
// returns the face as hours; d4 face 1 gives 1 hour and face 4 gives 4 hours; it counts one rolled result.
// Path: unhappy (boundary).
func TestRollStableRecoveryDelay(t *testing.T) {
	t.Parallel()
	for _, face := range []int{1, 4} {
		roller := dicetest.NewFixedRoller(face)
		rec, opts := recording()
		uc, err := combat.NewRollStableRecoveryDelay(roller, opts...)
		if err != nil {
			t.Fatal(err)
		}
		res, err := uc.Run(context.Background(), combat.RollStableRecoveryDelayCommand{})
		if err != nil || res.(combat.StableRecoveryDelayRolled).Delay().Hours() != face {
			t.Errorf("face %d = %v, %v", face, res, err)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustParseExpression("1d4") {
			t.Errorf("requests = %v, want exactly [1d4]", reqs)
		}
		ops := measured(rec, "srd5e.combat.operations")
		if len(ops) != 1 || ops[0]["srd5e.combat.result"] != "rolled" || ops[0]["srd5e.combat.result_class"] != "applied" || ops[0]["srd5e.combat.operation"] != "roll_stable_recovery_delay" {
			t.Errorf("operations = %v", ops)
		}
		checkTelemetryHygiene(t, rec)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-078, FR-092 - a roller that fails, panics or answers another
// expression is a *srderr.PortError, recorded once, with a nil result.
// Path: unhappy (collaborator failure).
func TestRollStableRecoveryDelayPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	for name, roller := range map[string]combat.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) }),
		"wrong expression": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustParseExpression("1d6"), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 3)})
		}),
	} {
		rec, opts := recording()
		uc, _ := combat.NewRollStableRecoveryDelay(roller, opts...)
		res, err := uc.Run(context.Background(), combat.RollStableRecoveryDelayCommand{})
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) {
			t.Errorf("%s = %v, %v, want a *PortError", name, res, err)
		}
		spans := rec.Tracer.Spans()
		if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(measured(rec, "srd5e.combat.errors")) != 1 || len(measured(rec, "srd5e.combat.operations")) != 0 {
			t.Errorf("%s: spans %+v, errors %v", name, spans, measured(rec, "srd5e.combat.errors"))
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-094 - a nil roller is invalid input; so is the zero use case.
// Path: unhappy (invalid input).
func TestNewRollStableRecoveryDelayRejectsNilRoller(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewRollStableRecoveryDelay(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("error = %v", err)
	}
	if res, err := (combat.RollStableRecoveryDelay{}).Run(context.Background(), combat.RollStableRecoveryDelayCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v, %v", res, err)
	}
}
