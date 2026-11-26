package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func rollBurning(t *testing.T, r combat.Roller, state combat.BurningState) (combat.RollBurningDamageResult, error) {
	t.Helper()
	uc, err := combat.NewRollBurningDamage(r)
	if err != nil {
		t.Fatal(err)
	}
	return uc.Run(context.Background(), combat.RollBurningDamageCommand{State: state})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-005, FR-007, FR-008, FR-095 - a burning creature makes exactly one 1d4 request and gets that many points of full, unadjusted Fire damage, still burning; the next call rolls again.
// Path: per case.
func TestRollBurningDamageRollsOneD4(t *testing.T) {
	t.Parallel()
	for face := 1; face <= 4; face++ {
		roller := dicetest.NewFixedRoller(face, face)
		for turn := 0; turn < 2; turn++ {
			res, err := rollBurning(t, roller, combat.BurningStateBurning)
			got, ok := res.(combat.BurningDamageRolled)
			if err != nil || !ok || got.Damage() != combat.MustDamage(face, content.DamageFire) || got.State() != combat.BurningStateBurning ||
				got.Roll().Total() != face {
				t.Fatalf("face %d turn %d = %#v, %v, want %d Fire damage and still burning (FR-005, FR-007)", face, turn, res, err, face)
			}
		}
		want := dice.MustParseExpression("1d4")
		if reqs := roller.Requests(); len(reqs) != 2 || reqs[0] != want || reqs[1] != want {
			t.Errorf("requests = %v, want two of 1d4 (FR-095)", reqs)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-006 - a creature that is not burning gets the NotBurning rejection, no roll and no damage.
// Path: unhappy (state error).
func TestRollBurningDamageWhenNotBurningRollsNothing(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(3)
	res, err := rollBurning(t, roller, combat.BurningStateNotBurning)
	if err != nil || res != (combat.BurningDamageRejected{Rejection: combat.BurningRejectionNotBurning}) || len(roller.Requests()) != 0 {
		t.Errorf("= %#v, %v with %d requests, want the NotBurning rejection and no roll", res, err, len(roller.Requests()))
	}
	var zero combat.RollBurningDamageCommand
	if r, err := rollBurning(t, roller, zero.State); err != nil || r.(combat.BurningDamageRejected).Rejection != combat.BurningRejectionNotBurning {
		t.Errorf("the zero state = %#v, %v, want not burning", r, err)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-094, FR-095 - a nil roller, a state that is not valid and a cancelled context are errors before any signal; a roller that fails, panics or answers another expression is a *srderr.PortError and no result.
// Path: unhappy (collaborator failure).
func TestRollBurningDamageFailures(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewRollBurningDamage(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller = %v, want ErrInvalidInput", err)
	}
	var zero combat.RollBurningDamage
	if _, err := zero.Run(context.Background(), combat.RollBurningDamageCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v, want ErrInvalidInput", err)
	}
	if _, err := rollBurning(t, dicetest.NewFixedRoller(1), combat.BurningState(9)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid state = %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	uc, _ := combat.NewRollBurningDamage(dicetest.NewFixedRoller(1))
	if _, err := uc.Run(cancelled, combat.RollBurningDamageCommand{State: combat.BurningStateBurning}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, want context.Canceled", err)
	}
	cause := errors.New("roller down")
	for name, r := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := rollBurning(t, r, combat.BurningStateBurning)
		envPortFailure(t, err)
		if res != nil {
			t.Errorf("%s: result = %#v with an error, want nil", name, res)
		}
		if name == "fails" && !errors.Is(err, cause) || name == "wrong expression" && !errors.Is(err, srderr.ErrContractViolation) {
			t.Errorf("%s: cause not reachable: %v", name, err)
		}
	}
}

func extinguishBurning(t *testing.T, state combat.BurningState, m combat.ExtinguishMethod, set combat.ConditionSet) (combat.ExtinguishBurningResult, error) {
	t.Helper()
	return combat.NewExtinguishBurning().Run(context.Background(), combat.ExtinguishBurningCommand{State: state, Method: m, Conditions: set, Target: subject()})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-009 to FR-013, FR-106, FR-120 (table 4.2a) - extinguishing a fire: rolling on the ground applies Prone through ConditionSet.Apply, the other methods leave the conditions alone, an existing Prone still ends the fire, Immunity to Prone blocks rolling, and BurningEnded comes once, last, only when the fire went out.
// Path: per case.
func TestExtinguishBurning(t *testing.T) {
	t.Parallel()
	roll, doused := combat.ExtinguishMethodRollOnTheGround, combat.ExtinguishMethodDoused
	burning, notBurning := combat.BurningStateBurning, combat.BurningStateNotBurning
	cases := []struct {
		name, requirement, path string
		state                   combat.BurningState
		method                  combat.ExtinguishMethod
		set                     combat.ConditionSet
		reject                  combat.BurningRejection // 0: extinguished
		prone                   string                  // "added", "held", "immune", "" for ProneNotApplied
		events                  []string
		proneAfter              bool
	}{
		{"roll on the ground", "FR-009, FR-106", "happy", burning, roll, combat.ConditionSet{}, 0, "added", []string{"condition_applied", "burning_ended"}, true},
		{"already Prone", "FR-010", "unhappy (rule interaction)", burning, roll, condSet(t, condConds(combat.Prone{})), 0, "held", []string{"burning_ended"}, true},
		{"Unconscious implies Prone", "FR-010", "unhappy (rule interaction)", burning, roll, condSet(t, condConds(combat.Unconscious{})), 0, "held", []string{"burning_ended"}, true},
		{"immune to Prone", "FR-011", "unhappy (rule interaction)", burning, roll, condSet(t, nil, content.ConditionProne), combat.BurningRejectionExtinguishBlockedByImmunity, "", nil, false},
		{"doused", "FR-012", "happy", burning, doused, combat.ConditionSet{}, 0, "", []string{"burning_ended"}, false},
		{"submerged", "FR-012", "happy", burning, combat.ExtinguishMethodSubmerged, combat.ConditionSet{}, 0, "", []string{"burning_ended"}, false},
		{"suffocated", "FR-012", "happy", burning, combat.ExtinguishMethodSuffocated, combat.ConditionSet{}, 0, "", []string{"burning_ended"}, false},
		{"doused while immune to Prone", "FR-012", "unhappy (rule interaction)", burning, doused, condSet(t, nil, content.ConditionProne), 0, "", []string{"burning_ended"}, false},
		{"not burning, roll", "FR-013", "unhappy (state error)", notBurning, roll, combat.ConditionSet{}, combat.BurningRejectionNotBurning, "", nil, false},
		{"not burning, doused", "FR-013", "unhappy (state error)", notBurning, doused, combat.ConditionSet{}, combat.BurningRejectionNotBurning, "", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.set
			res, err := extinguishBurning(t, c.state, c.method, c.set)
			if err != nil {
				t.Fatal(err)
			}
			if c.reject != 0 {
				if res != (combat.BurningExtinguishRejected{Rejection: c.reject}) {
					t.Errorf("= %#v, want the rejection %v [%s]", res, c.reject, c.requirement)
				}
				return
			}
			got, ok := res.(combat.BurningExtinguished)
			if !ok || got.State() != combat.BurningStateNotBurning {
				t.Fatalf("= %#v, want BurningExtinguished and not burning [%s]", res, c.requirement)
			}
			if !equalStrings(envKinds(got.Events()), c.events) {
				t.Errorf("events = %v, want %v [%s]", envKinds(got.Events()), c.events, c.requirement)
			}
			kind := ""
			switch p := got.ProneApplication().(type) {
			case combat.ProneApplied:
				switch p.Application.(type) {
				case combat.ConditionAdded:
					kind = "added"
				case combat.ConditionAlreadyHeld:
					kind = "held"
				case combat.ConditionImmune:
					kind = "immune"
				}
			case combat.ProneNotApplied:
			default:
				t.Fatalf("proneApplication = %#v", p)
			}
			if kind != c.prone {
				t.Errorf("prone application = %q, want %q [FR-120]", kind, c.prone)
			}
			hasProne := false
			for _, k := range got.Conditions().Effective() {
				hasProne = hasProne || k == content.ConditionProne
			}
			if hasProne != c.proneAfter {
				t.Errorf("Prone after = %v, want %v", hasProne, c.proneAfter)
			}
			if c.prone != "added" && got.Conditions() != before {
				t.Error("the conditions changed though Prone was not added")
			}
			ev := got.Events()
			ev[0] = nil
			if got.Events()[0] == nil {
				t.Error("Events is not a copy")
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-106, FR-105 - BurningEnded carries the target and the method; a zero target, an invalid state or method, and a cancelled context are errors.
// Path: unhappy (invalid input).
func TestExtinguishBurningInvalidInput(t *testing.T) {
	t.Parallel()
	uc := combat.NewExtinguishBurning()
	ok := combat.ExtinguishBurningCommand{State: combat.BurningStateBurning, Method: combat.ExtinguishMethodDoused, Target: subject()}
	res, err := uc.Run(context.Background(), ok)
	ended := res.(combat.BurningExtinguished).Events()[0].(combat.BurningEnded)
	if err != nil || ended.Method() != combat.ExtinguishMethodDoused || ended.Combatants()[0] != subject() {
		t.Errorf("BurningEnded = %#v, %v", ended, err)
	}
	bad := map[string]combat.ExtinguishBurningCommand{
		"zero target": {State: ok.State, Method: ok.Method}, "invalid state": {State: 7, Method: ok.Method, Target: subject()},
		"invalid method": {State: ok.State, Method: 0, Target: subject()}, "zero command": {},
	}
	for name, cmd := range bad {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, ok); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, want context.Canceled", err)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-113 - the Burning enums and rejections carry names, and an invalid value is unknown.
// Path: unhappy (invalid input).
func TestBurningEnumNames(t *testing.T) {
	t.Parallel()
	if combat.BurningStateNotBurning.String() != "not_burning" || combat.BurningStateBurning.String() != "burning" || combat.BurningState(5).String() != "unknown" ||
		combat.BurningState(5).Valid() || combat.BurningState(-1).Valid() {
		t.Error("BurningState names or validity are wrong")
	}
	if combat.BurningRejectionNotBurning.String() != "not_burning" || combat.BurningRejectionExtinguishBlockedByImmunity.String() != "blocked_by_immunity" || combat.BurningRejection(0).String() != "unknown" {
		t.Error("BurningRejection names are wrong")
	}
	names := []string{"roll_on_the_ground", "doused", "submerged", "suffocated"}
	for i, n := range names {
		m := combat.ExtinguishMethod(i + 1)
		if m.String() != n || !m.Valid() {
			t.Errorf("method %d = %q valid %v, want %q", i+1, m, m.Valid(), n)
		}
	}
	if combat.ExtinguishMethod(0).Valid() || combat.ExtinguishMethod(5).Valid() || combat.ExtinguishMethod(5).String() != "unknown" {
		t.Error("an invalid ExtinguishMethod reads as valid")
	}
}
