package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// luTurnStart is a drake whose Recharge 5-6 and Recharge 6 parts are spent and that took two Legendary Actions.
func luTurnStart(t *testing.T) combat.LimitedUseState {
	t.Helper()
	s := luSpent(t, luSpent(t, luStart(t, luDrake(t), combat.OutsideLair), luSpray(t)), luLash(t))
	return luTake(t, luTake(t, s, luWindow(t, "b", 1)), luWindow(t, "c", 1))
}

func luTurnUse(t *testing.T, r combat.Roller, opts ...combat.Option) combat.ApplyMonsterTurnStart {
	t.Helper()
	return luMust(combat.NewApplyMonsterTurnStart(r, opts...))
}

func luRun(t *testing.T, r combat.Roller, s combat.LimitedUseState, opts ...combat.Option) (combat.MonsterTurnStartApplied, error) {
	t.Helper()
	res, err := luTurnUse(t, r, opts...).Run(context.Background(), combat.ApplyMonsterTurnStartCommand{State: s})
	if err != nil {
		return combat.MonsterTurnStartApplied{}, err
	}
	return res.(combat.MonsterTurnStartApplied), nil
}

// Requirement: doc/specs/monster-limited-use.md FR-035, FR-036, FR-024 - the use case rolls 1d6 once per part of RechargeRollsNeeded in printed order, then applies the faces with StartOfOwnTurn; the result holds the new state, the events in order and the rolls made.
// Path: per case.
func TestApplyMonsterTurnStart(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		state                   combat.LimitedUseState
		faces                   []int
		wantSpray, wantLash     combat.RechargeReadiness
		wantKinds               []string
		wantFaces               []int
	}{
		{"both recharge", "FR-036 FR-024", "happy", luTurnStart(t), []int{5, 6}, combat.RechargeReady, combat.RechargeReady,
			[]string{"monster_recharge_rolled", "monster_recharge_rolled", "legendary_action_uses_regained"}, []int{5, 6}},
		{"faces in printed order: the first gets the first face", "FR-036", "unhappy (boundary)", luTurnStart(t), []int{4, 6}, combat.RechargeSpent, combat.RechargeReady,
			[]string{"monster_recharge_rolled", "monster_recharge_rolled", "legendary_action_uses_regained"}, []int{4, 6}},
		{"none recharge", "FR-036", "unhappy (boundary)", luTurnStart(t), []int{4, 5}, combat.RechargeSpent, combat.RechargeSpent,
			[]string{"monster_recharge_rolled", "monster_recharge_rolled", "legendary_action_uses_regained"}, []int{4, 5}},
		{"nothing to roll for makes no roll", "FR-036", "unhappy (empty and minimal)", luStart(t, luDrake(t), combat.OutsideLair), nil, combat.RechargeReady, combat.RechargeReady, []string{}, nil},
		{"only the legendary uses return", "FR-036 FR-022", "unhappy (rule interaction)", luTake(t, luStart(t, luDrake(t), combat.OutsideLair), luWindow(t, "b", 1)), nil,
			combat.RechargeReady, combat.RechargeReady, []string{"legendary_action_uses_regained"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement
			roller := dicetest.NewFixedRoller(c.faces...)
			before := c.state
			got, err := luRun(t, roller, c.state)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if !c.state.Equal(before) {
				t.Error("the command state changed")
			}
			var kinds []string
			for _, e := range got.Events() {
				kinds = append(kinds, e.Kind())
			}
			if kinds == nil {
				kinds = []string{}
			}
			if !slices.Equal(kinds, c.wantKinds) {
				t.Errorf("event kinds = %v, want %v [%s]", kinds, c.wantKinds, req)
			}
			var faces []int
			for _, r := range got.Rolls() {
				faces = append(faces, r.Face())
			}
			if !slices.Equal(faces, c.wantFaces) {
				t.Errorf("rolls = %v, want %v [%s]", faces, c.wantFaces, req)
			}
			for _, e := range roller.Requests() {
				if e != dice.MustExpression(1, dice.D6(), 0) {
					t.Errorf("rolled %v, want 1d6 [FR-036]", e)
				}
			}
			if len(roller.Requests()) != len(c.wantFaces) {
				t.Errorf("made %d rolls, want %d (one per part that needs one) [FR-036]", len(roller.Requests()), len(c.wantFaces))
			}
			if luReadiness(t, got.State(), luSpray(t)) != c.wantSpray || luReadiness(t, got.State(), luLash(t)) != c.wantLash {
				t.Errorf("readiness = %v %v, want %v %v [%s]", luReadiness(t, got.State(), luSpray(t)), luReadiness(t, got.State(), luLash(t)), c.wantSpray, c.wantLash, req)
			}
			if leg := luLegendaryOf(t, got.State()); leg.Remaining() != leg.Maximum() {
				t.Errorf("Legendary uses = %d of %d, want full [FR-022]", leg.Remaining(), leg.Maximum())
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-037 - a dice.Roller error, a panic or an answer for another expression is a *srderr.PortError for port dice with errors.Is finding the cause, with no state or event returned, also on the second roll, and recorded once (error status and one srd5e.combat.errors count, no operations count, no roll count, no log).
// Path: unhappy (failure of collaborator).
func TestApplyMonsterTurnStartPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	secondFails := func() combat.Roller {
		calls := 0
		return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
			calls++
			if calls == 2 {
				return dice.Result{}, cause
			}
			return dicetest.NewFixedRoller(5).Roll(e)
		})
	}
	rollers := map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller(), "fails on the second roll": secondFails()}
	for name, roller := range rollers {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			state := luTurnStart(t)
			res, err := luTurnUse(t, roller, opts...).Run(context.Background(), combat.ApplyMonsterTurnStartCommand{State: state})
			envPortFailure(t, err)
			if res != nil {
				t.Errorf("result = %#v, want nil", res)
			}
			if name == "fails" && !errors.Is(err, cause) {
				t.Errorf("errors.Is(err, cause) = false for %v", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.apply_monster_turn_start")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
			}
			if got := values(measured(rec, "srd5e.combat.errors"), "error.type"); !slices.Equal(got, []string{"dice_roller_error"}) {
				t.Errorf("errors counted = %v, want one dice_roller_error", got)
			}
			for _, n := range []string{"srd5e.combat.operations", "srd5e.combat.events", "srd5e.combat.monster_recharge_rolls"} {
				if m := rec.Meter.MeasurementsNamed(n); len(m) != 0 {
					t.Errorf("%s = %v, want no sample after a port failure", n, m)
				}
			}
			checkTelemetryHygiene(t, rec)
			if !state.Equal(luTurnStart(t)) {
				t.Error("the state changed")
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-039, FR-038 - the use case records the shared combat signals under apply_monster_turn_start: class applied, srd5e.combat.result turn_start_recharged, turn_start_not_recharged or turn_start_no_rolls, one srd5e.combat.monster_recharge_rolls count per roll with the recharge outcome, the event kinds counted, no id in any attribute, no log; the result equals the Noop result.
// Path: per case.
func TestApplyMonsterTurnStartSignals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		state                   combat.LimitedUseState
		faces                   []int
		wantResult              string
		wantOutcomes            []string
		wantEvents              []string
	}{
		{"one regained, one not", "FR-039", "happy", luTurnStart(t), []int{5, 5}, "turn_start_recharged", []string{"not_regained", "regained"},
			[]string{"legendary_action_uses_regained", "monster_recharge_rolled", "monster_recharge_rolled"}},
		{"none regained", "FR-039", "unhappy (boundary)", luTurnStart(t), []int{4, 5}, "turn_start_not_recharged", []string{"not_regained", "not_regained"},
			[]string{"legendary_action_uses_regained", "monster_recharge_rolled", "monster_recharge_rolled"}},
		{"no rolls", "FR-039", "unhappy (empty and minimal)", luStart(t, luDrake(t), combat.OutsideLair), nil, "turn_start_no_rolls", []string{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement
			rec, opts := recording()
			got, err := luRun(t, dicetest.NewFixedRoller(c.faces...), c.state, opts...)
			if err != nil {
				t.Fatal(err)
			}
			plain, err := luRun(t, dicetest.NewFixedRoller(c.faces...), c.state)
			if err != nil || !plain.State().Equal(got.State()) || !slices.Equal(plain.Rolls(), got.Rolls()) || len(plain.Events()) != len(got.Events()) {
				t.Errorf("recording and Noop results differ: %v vs %v (%v) [FR-038]", plain, got, err)
			}
			want := map[string]any{"srd5e.combat.operation": "apply_monster_turn_start", "srd5e.outcome": "success", "srd5e.combat.result": c.wantResult, "srd5e.combat.result_class": "applied"}
			spans := rec.Tracer.SpansNamed("srd5e.combat.apply_monster_turn_start")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status == telemetry.StatusError || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span without error status [%s]", spans, req)
			}
			if sa := attrMap(spans[0].Attrs); len(sa) != len(want) {
				t.Errorf("span attrs = %v, want %v", sa, want)
			} else {
				for k, v := range want {
					if sa[k] != v {
						t.Errorf("span attr %s = %v, want %v [%s]", k, sa[k], v, req)
					}
				}
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || ops[0]["srd5e.combat.result"] != c.wantResult || len(ops[0]) != len(want) {
				t.Errorf("operations = %v, want one with result %s [%s]", ops, c.wantResult, req)
			}
			if got := values(measured(rec, "srd5e.combat.monster_recharge_rolls"), "srd5e.combat.recharge_outcome"); !slices.Equal(got, c.wantOutcomes) {
				t.Errorf("recharge outcomes = %v, want %v [FR-039]", got, c.wantOutcomes)
			}
			for _, m := range measured(rec, "srd5e.combat.monster_recharge_rolls") {
				if len(m) != 1 {
					t.Errorf("a roll count carries %v, want only the outcome", m)
				}
			}
			if got := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(got, c.wantEvents) {
				t.Errorf("event kinds counted = %v, want %v", got, c.wantEvents)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0", n)
			}
			checkTelemetryHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-035, FR-031 - a nil roller, the zero use case and the zero command are invalid input before any signal, a cancelled context is returned unwrapped before any signal, and the result hands out copies of its events and rolls.
// Path: unhappy (invalid input, cancellation, immutability).
func TestApplyMonsterTurnStartInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewApplyMonsterTurnStart(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewApplyMonsterTurnStart(nil) = %v, want ErrInvalidInput", err)
	}
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(5, 5)
	use := luTurnUse(t, roller, opts...)
	if _, err := (combat.ApplyMonsterTurnStart{}).Run(context.Background(), combat.ApplyMonsterTurnStartCommand{State: luTurnStart(t)}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, want ErrInvalidInput", err)
	}
	if _, err := use.Run(context.Background(), combat.ApplyMonsterTurnStartCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v, want ErrInvalidInput", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(cancelled, combat.ApplyMonsterTurnStartCommand{State: luTurnStart(t)}); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled: %v, %v, want nil and context.Canceled", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Error("an invalid call recorded a signal or rolled")
	}
	got, err := luRun(t, dicetest.NewFixedRoller(5, 5), luTurnStart(t))
	if err != nil {
		t.Fatal(err)
	}
	ev, rolls := got.Events(), got.Rolls()
	ev[0], rolls[0] = nil, combat.MonsterRechargeRolled{}
	if got.Events()[0] == nil || got.Rolls()[0] == (combat.MonsterRechargeRolled{}) {
		t.Error("Events and Rolls are not copies [FR-031]")
	}
}
