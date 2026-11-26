package combat_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// escaper is the grapple record every escape test uses: a holds b, escape DC 13.
func escaper(tb testing.TB) combat.Grapples {
	tb.Helper()
	return grapplesOf(tb, held("a", "b", hand(0)))
}

// rollOf is a roll attempt with the ability modifier mod and the given circumstances.
func rollOf(mod int, cs ...modifier.RollCircumstance) d20.RollAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(mod)), Circumstances: cs}
}

// escapeCommand builds the command of b escaping with attempt.
func escapeCommand(tb testing.TB, s combat.Grapples, actions combat.ConditionActionAvailability, attempt d20.CheckAttempt) combat.EscapeGrappleFromRecordCommand {
	tb.Helper()
	cmd, err := combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), combat.GrappleEscapeOptionDexterityAcrobatics, actions, attempt)
	if err != nil {
		tb.Fatal(err)
	}
	return cmd
}

// Requirement: per case
// Path: per case.
func TestEscapeGrappleFromRecordRolls(t *testing.T) {
	t.Parallel()
	const p6 = `"D20 Tests", p. 6; "Grappling", p. 182`
	adv, dis := modifier.RollCircumstance(modifier.Advantage{}), modifier.RollCircumstance(modifier.Disadvantage{})
	cases := []struct {
		name, requirement, path string
		attempt                 d20.RollAttempt
		faces                   []int
		request                 string
		natural, total          int
		escaped                 bool
	}{
		{"total equals DC", "FR-038, FR-040, table 4.10.5, " + p6, "unhappy (boundary)", rollOf(3), []int{10}, "1d20", 10, 13, true},
		{"total one below DC", "FR-039, FR-040, table 4.10.5", "unhappy (boundary)", rollOf(3), []int{9}, "1d20", 9, 12, false},
		{"total well above", "FR-038, table 4.10.5", "happy", rollOf(3), []int{20}, "1d20", 20, 23, true},
		{"natural 1 is only a low total", "FR-040 (checks have no natural 1 rule)", "unhappy (boundary)", rollOf(30), []int{1}, "1d20", 1, 31, true},
		{"natural 20 does not beat the DC", "FR-040", "unhappy (boundary)", rollOf(-30), []int{20}, "1d20", 20, -10, false},
		{"Disadvantage keeps the lower die", `FR-046; "Advantage/Disadvantage", p. 8`, "unhappy (rule interaction)", rollOf(0, dis), []int{17, 4}, "2d20", 4, 4, false},
		{"Advantage keeps the higher die", "FR-046", "unhappy (rule interaction)", rollOf(0, adv), []int{4, 17}, "2d20", 17, 17, true},
		{"Advantage and Disadvantage cancel", "FR-046, table 4.10.5", "unhappy (rule interaction)", rollOf(1, adv, dis), []int{12}, "1d20", 12, 13, true},
		{"two Advantage sources are one", `FR-105; "Powerful Build", "Large Form", p. 86`, "unhappy (rule interaction)", rollOf(0, adv, adv), []int{4, 17}, "2d20", 17, 17, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			uc, err := combat.NewEscapeGrappleFromRecord(roller)
			if err != nil {
				t.Fatal(err)
			}
			s := escaper(t)
			res, err := uc.Run(context.Background(), escapeCommand(t, s, combat.ConditionActionAvailabilityAvailable, c.attempt))
			if err != nil {
				t.Fatal(err)
			}
			if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustParseExpression(c.request) {
				t.Errorf("requests = %v, want exactly [%s] [%s]", reqs, c.request, c.requirement)
			}
			var check combat.GrappleEscapeCheck
			switch r := res.(type) {
			case combat.EscapeGrappleFromRecordEscaped:
				check = r.Check()
				if !c.escaped || r.Grapples().Len() != 0 || r.Ending().Cause() != combat.GrappleEndCauseEscaped || r.Ending().Grapple() != held("a", "b", hand(0)) || !r.ActionSpent() {
					t.Errorf("escaped = %+v, want held-and-spent only when the check fails [%s]", r, c.requirement)
				}
			case combat.EscapeGrappleFromRecordHeld:
				check = r.Check()
				if c.escaped || !r.Grapples().Equal(s) || !r.ActionSpent() {
					t.Errorf("held = %+v, want the set unchanged and the action spent [%s]", r, c.requirement)
				}
			default:
				t.Fatalf("result = %#v", res)
			}
			got := check.(combat.GrappleEscapeRolled).Result()
			if got.Total() != c.total || got.Roll().Natural() != c.natural || got.Target() != 13 || got.Kind() != d20.AbilityCheck || !got.Modifiers().Equal(c.attempt.Modifiers) {
				t.Errorf("check = natural %d total %d target %d, want %d/%d/13 with the attempt's modifiers [FR-037, FR-040]", got.Roll().Natural(), got.Total(), got.Target(), c.natural, c.total)
			}
			want, _ := d20.ResolveCheck(d20.AbilityCheck, got.Roll(), c.attempt.Modifiers, 13)
			if !got.Equal(want) {
				t.Errorf("the check differs from d20.ResolveCheck's [FR-040]")
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-037 - the DC is the grapple's own EscapeDC, whatever it is.
// Path: unhappy (boundary).
func TestEscapeUsesTheGrapplesOwnDC(t *testing.T) {
	t.Parallel()
	g := combat.MustGrapple(gid("a"), gid("b"), hand(0), 25, 5)
	uc, _ := combat.NewEscapeGrappleFromRecord(dicetest.NewFixedRoller(20))
	res, err := uc.Run(context.Background(), escapeCommand(t, grapplesOf(t, g), combat.ConditionActionAvailabilityAvailable, rollOf(4)))
	r, ok := res.(combat.EscapeGrappleFromRecordHeld)
	if err != nil || !ok || r.Check().(combat.GrappleEscapeRolled).Result().Target() != 25 {
		t.Errorf("= %#v, %v, want a held escape against DC 25 (20 + 4 = 24)", res, err)
	}
}

// Requirement: doc/specs/grappling.md FR-045, table 4.10.5 - an AutomaticFailure draws no die and holds with the forced
// failure d20.FailWithoutRoll returns; the action is spent.
// Path: unhappy (rule interaction).
func TestEscapeAutomaticFailureDrawsNoDie(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(20)
	uc, _ := combat.NewEscapeGrappleFromRecord(roller)
	s := escaper(t)
	res, err := uc.Run(context.Background(), escapeCommand(t, s, combat.ConditionActionAvailabilityAvailable, d20.AutomaticFailure{}))
	want, _ := d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic)
	r, ok := res.(combat.EscapeGrappleFromRecordHeld)
	forced, isForced := combat.GrappleEscapeCheck(nil), false
	if ok {
		forced = r.Check()
		_, isForced = forced.(combat.GrappleEscapeForced)
	}
	if err != nil || !ok || !isForced || forced.(combat.GrappleEscapeForced).Result() != want || !r.ActionSpent() || !r.Grapples().Equal(s) {
		t.Errorf("= %#v, %v, want held with %v", res, err, want)
	}
	if n := len(roller.Requests()); n != 0 {
		t.Errorf("%d dice requested, want none", n)
	}
}

// Requirement: doc/specs/grappling.md FR-041, FR-042, FR-043, FR-044, table 4.10.5 - a target that is not grappled and a
// creature that can take no action are refused without a die and without spending the action; not grappled is
// reported first.
// Path: per case.
func TestEscapeRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		set                     combat.Grapples
		target                  string
		actions                 combat.ConditionActionAvailability
		want                    combat.GrappleRejection
	}{
		{"Incapacitated escaper", `FR-042; "Incapacitated", "Inactive", p. 184`, "unhappy (state error)", escaper(t), "b", combat.ConditionActionAvailabilityNone, combat.GrappleRejectionNoAction},
		{"not grappled", "FR-043", "unhappy (state error)", escaper(t), "zed", combat.ConditionActionAvailabilityAvailable, combat.GrappleRejectionNotGrappled},
		{"not grappled and Incapacitated", "FR-044", "unhappy (rule interaction)", escaper(t), "zed", combat.ConditionActionAvailabilityNone, combat.GrappleRejectionNotGrappled},
		{"empty set", "FR-043", "unhappy (empty and minimal)", combat.Grapples{}, "b", combat.ConditionActionAvailabilityAvailable, combat.GrappleRejectionNotGrappled},
		{"the grappler is not a held target", "FR-043", "unhappy (state error)", escaper(t), "a", combat.ConditionActionAvailabilityAvailable, combat.GrappleRejectionNotGrappled},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(20)
			uc, _ := combat.NewEscapeGrappleFromRecord(roller)
			cmd, err := combat.NewEscapeGrappleFromRecordCommand(c.set, combat.CombatantID(c.target), combat.GrappleEscapeOptionStrengthAthletics, c.actions, rollOf(0))
			if err != nil {
				t.Fatal(err)
			}
			res, err := uc.Run(context.Background(), cmd)
			if err != nil || res != (combat.EscapeGrappleFromRecordRejected{Rejection: c.want}) || res.ActionSpent() {
				t.Errorf("= %#v, %v, want %v with the action not spent [%s]", res, err, c.want, c.requirement)
			}
			if n := len(roller.Requests()); n != 0 {
				t.Errorf("%d dice requested, want none [%s]", n, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/grappling.md FR-047, FR-075 - a roller that fails, panics or answers another expression is a
// *srderr.PortError (dice_roller_error) with a nil result, recorded once on the span and once in srd5e.combat.errors,
// with no operations increment, no own signal and no log; the cause stays reachable.
// Path: unhappy (failure of collaborator).
func TestEscapePortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
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
			uc, _ := combat.NewEscapeGrappleFromRecord(roller, opts...)
			res, err := uc.Run(context.Background(), escapeCommand(t, escaper(t), combat.ConditionActionAvailabilityAvailable, rollOf(0)))
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port {
				t.Fatalf("= %v, %v, want nil and a dice_roller *PortError", res, err)
			}
			if name == "wrong expression" && !errors.Is(err, srderr.ErrContractViolation) || name == "fails" && !errors.Is(err, cause) {
				t.Errorf("cause not reachable: %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.escape_grapple_from_record")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
			}
			errs := measured(rec, "srd5e.combat.errors")
			if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != "escape_grapple_from_record" {
				t.Errorf("errors = %v, want one with error.type", errs)
			}
			for _, n := range []string{"operations", "events", "grapple.ended", "grapple.in_force"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
					t.Errorf("%s recorded %d on an error", n, len(m))
				}
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/grappling.md FR-041, FR-069 - equal commands with equal fixed dice give equal results, and the
// command is not altered.
// Path: unhappy (determinism).
func TestEscapeIsDeterministicAndLeavesTheCommandAlone(t *testing.T) {
	t.Parallel()
	cmd := escapeCommand(t, escaper(t), combat.ConditionActionAvailabilityAvailable, rollOf(3, modifier.Disadvantage{}))
	snapshot := escapeCommand(t, escaper(t), combat.ConditionActionAvailabilityAvailable, rollOf(3, modifier.Disadvantage{}))
	var results [2]combat.EscapeGrappleFromRecordResult
	for i := range results {
		uc, _ := combat.NewEscapeGrappleFromRecord(dicetest.NewFixedRoller(17, 4))
		res, err := uc.Run(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		results[i] = res
	}
	if !reflect.DeepEqual(results[0], results[1]) || !reflect.DeepEqual(cmd, snapshot) {
		t.Error("results differ or the command changed")
	}
}

// Requirement: doc/specs/grappling.md FR-035, FR-041, testing.md section 4 - the command constructor rejects an invalid option
// or action availability and an attempt that is neither a RollAttempt nor an AutomaticFailure value; a nil roller is
// rejected; the zero use case, zero command and cancelled context are refused before any signal.
// Path: unhappy (invalid input).
func TestEscapeRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	s := escaper(t)
	avail := combat.ConditionActionAvailabilityAvailable
	roll, auto := d20.CheckAttempt(rollOf(0)), d20.CheckAttempt(d20.AutomaticFailure{})
	for name, err := range map[string]error{
		"zero option":      second(combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 0, avail, roll)),
		"option past end":  second(combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 3, avail, roll)),
		"zero actions":     second(combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 1, 0, roll)),
		"actions past end": second(combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 1, 3, roll)),
		"nil attempt":      second(combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 1, avail, nil)),
		"pointer attempt":  second(combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 1, avail, &d20.RollAttempt{})),
		"nil roller":       second(combat.NewEscapeGrappleFromRecord(nil)),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if cmd, err := combat.NewEscapeGrappleFromRecordCommand(s, gid("b"), 1, avail, auto); err != nil || cmd.Option() != combat.GrappleEscapeOptionStrengthAthletics {
		t.Errorf("a valid command was rejected or lost its option: %v", err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(20)
	uc, _ := combat.NewEscapeGrappleFromRecord(roller, opts...)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, escapeCommand(t, s, avail, roll)); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx: %v", err)
	}
	if _, err := uc.Run(context.Background(), combat.EscapeGrappleFromRecordCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v", err)
	}
	var zero combat.EscapeGrappleFromRecord
	if _, err := zero.Run(context.Background(), escapeCommand(t, s, avail, roll)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v", err)
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()) + len(roller.Requests()); n != 0 {
		t.Errorf("signals or dice before validation: %d", n)
	}
}
