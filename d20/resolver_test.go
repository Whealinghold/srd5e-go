package d20_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func TestResolveCaseTable(t *testing.T) {
	t.Parallel()
	for _, tc := range caseTable() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(tc.rolls...)
			res := resolve(t, newResolver(t, roller), tc.cmd)

			var wantRequests []dice.Expression
			if len(tc.rolls) > 0 {
				wantRequests = []dice.Expression{d20Expr(tc.wantDice)}
			}
			if got := roller.Requests(); !slices.Equal(got, wantRequests) {
				t.Errorf("requests = %v, want %v", got, wantRequests)
			}
			if res.Ability() != commandAbility(t, tc.cmd) {
				t.Errorf("Ability() = %v, want %v", res.Ability(), commandAbility(t, tc.cmd))
			}
			if res.Kind() != tc.want.kind || res.Outcome() != tc.want.outcome {
				t.Errorf("kind %v outcome %v, want %v %v", res.Kind(), res.Outcome(), tc.want.kind, tc.want.outcome)
			}
			switch v := res.(type) {
			case d20.RolledCheck:
				c := v.Check()
				if c.Total() != tc.want.total || c.Roll().Mode() != tc.want.mode || int(c.Target()) != tc.want.target ||
					!slices.Equal(c.Roll().Dice(), tc.rolls) {
					t.Errorf("check = total %d mode %v target %d dice %v", c.Total(), c.Roll().Mode(), c.Target(), c.Roll().Dice())
				}
			case d20.RolledAttack:
				a := v.Attack()
				if a.Total() != tc.want.total || a.Roll().Mode() != tc.want.mode || int(a.Target()) != tc.want.target ||
					a.IsCriticalHit() != tc.want.critical || !slices.Equal(a.Roll().Dice(), tc.rolls) {
					t.Errorf("attack = total %d mode %v target %d crit %v", a.Total(), a.Roll().Mode(), a.Target(), a.IsCriticalHit())
				}
			case d20.ForcedFailure:
				if v.Failure().Reason() != tc.want.reason || int(v.Target()) != tc.want.target || v.Failure().Kind() != tc.want.kind {
					t.Errorf("forced = %+v target %d", v.Failure(), v.Target())
				}
			default:
				t.Fatalf("unexpected result type %T", res)
			}
		})
	}
}

func TestResultCarriesAbilityLabel(t *testing.T) {
	t.Parallel()
	for _, a := range ability.All() {
		cmds := map[string]d20.Command{
			"check":   d20.AbilityCheckCommand{Ability: a, Target: 10, Attempt: rollAttempt(0)},
			"save":    d20.SavingThrowCommand{Ability: a, Target: 10, Attempt: rollAttempt(0)},
			"attack":  d20.AttackRollCommand{Ability: a, Target: 10, Attempt: rollAttempt(0)},
			"chosen":  d20.SavingThrowCommand{Ability: a, Target: 10, Attempt: d20.ChooseToFail{}},
			"auto":    d20.SavingThrowCommand{Ability: a, Target: 10, Attempt: d20.AutomaticFailure{}},
			"autochk": d20.AbilityCheckCommand{Ability: a, Target: 10, Attempt: d20.AutomaticFailure{}},
		}
		for name, cmd := range cmds {
			res := resolve(t, newResolver(t, dicetest.NewMinimumRoller()), cmd)
			if res.Ability() != a {
				t.Errorf("%s: Ability() = %v, want %v", name, res.Ability(), a)
			}
		}
	}
}

// commandAbility reads the ability label off a command, failing on an
// unhandled command variant.
func commandAbility(t *testing.T, cmd d20.Command) ability.Ability {
	t.Helper()
	switch c := cmd.(type) {
	case d20.AbilityCheckCommand:
		return c.Ability
	case d20.SavingThrowCommand:
		return c.Ability
	case d20.AttackRollCommand:
		return c.Ability
	default:
		t.Fatalf("unhandled command variant %T", cmd)
		return 0
	}
}

func TestResultTypeSwitchIsExhaustive(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, tc := range caseTable() {
		commandAbility(t, tc.cmd) // fails on an unhandled Command variant
		res := resolve(t, newResolver(t, dicetest.NewFixedRoller(tc.rolls...)), tc.cmd)
		switch res.(type) {
		case d20.RolledCheck:
			seen["check"] = true
		case d20.RolledAttack:
			seen["attack"] = true
		case d20.ForcedFailure:
			seen["forced"] = true
		default:
			t.Fatalf("unhandled result variant %T", res)
		}
	}
	if len(seen) != 3 {
		t.Errorf("variants seen = %v, want all three", seen)
	}
}

func TestNewResultsValidate(t *testing.T) {
	t.Parallel()
	roll, err := d20.NewRoll(modifier.Normal, []int{10})
	if err != nil {
		t.Fatal(err)
	}
	check, err := d20.ResolveCheck(d20.AbilityCheck, roll, abilityMods(0), 10)
	if err != nil {
		t.Fatal(err)
	}
	attack, err := d20.ResolveAttack(roll, abilityMods(0), 10, d20.DefaultCriticalRange)
	if err != nil {
		t.Fatal(err)
	}
	forced, err := d20.FailWithoutRoll(d20.SavingThrow, d20.ChosenToFail)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d20.NewRolledCheck(ability.Wisdom, check)
	if err != nil || c.Ability() != ability.Wisdom || !c.Check().Equal(check) {
		t.Errorf("NewRolledCheck = %+v, %v", c, err)
	}
	a, err := d20.NewRolledAttack(ability.Wisdom, attack)
	if err != nil || a.Ability() != ability.Wisdom || !a.Attack().Equal(attack) {
		t.Errorf("NewRolledAttack = %+v, %v", a, err)
	}
	f, err := d20.NewForcedFailure(ability.Wisdom, forced, 12)
	if err != nil || f.Ability() != ability.Wisdom || f.Failure() != forced || f.Target() != 12 {
		t.Errorf("NewForcedFailure = %+v, %v", f, err)
	}

	bad := ability.Ability(0)
	_, e1 := d20.NewRolledCheck(bad, check)
	_, e2 := d20.NewRolledAttack(bad, attack)
	_, e3 := d20.NewForcedFailure(bad, forced, 12)
	_, e4 := d20.NewRolledCheck(ability.Wisdom, d20.CheckResult{})
	_, e5 := d20.NewRolledAttack(ability.Wisdom, d20.AttackRollResult{})
	_, e6 := d20.NewForcedFailure(ability.Wisdom, d20.ForcedFailureResult{}, 12)
	for i, tc := range []struct {
		err    error
		reason d20.Reason
	}{
		{e1, d20.ReasonUnknownAbility}, {e2, d20.ReasonUnknownAbility}, {e3, d20.ReasonUnknownAbility},
		{e4, d20.ReasonZeroRoll}, {e5, d20.ReasonZeroRoll}, {e6, d20.ReasonUnknownKind},
	} {
		if !errors.Is(tc.err, &d20.InvalidError{Reason: tc.reason}) {
			t.Errorf("case %d: err = %v, want reason %v", i, tc.err, tc.reason)
		}
	}
}

func TestResultsAreValues(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(11, 11, 20, 20)
	r := newResolver(t, roller)
	check := d20.AbilityCheckCommand{Ability: ability.Strength, Target: 15, Attempt: rollAttempt(5)}
	attack := d20.AttackRollCommand{Ability: ability.Strength, Target: 15, Attempt: rollAttempt(5)}

	c1, c2 := resolve(t, r, check).(d20.RolledCheck), resolve(t, r, check).(d20.RolledCheck)
	a1, a2 := resolve(t, r, attack).(d20.RolledAttack), resolve(t, r, attack).(d20.RolledAttack)
	if !c1.Equal(c2) || !a1.Equal(a2) {
		t.Error("identical runs gave unequal results")
	}
	other := resolve(t, newResolver(t, dicetest.NewFixedRoller(12)), check).(d20.RolledCheck)
	if c1.Equal(other) {
		t.Error("results differing in the roll are equal")
	}
	if a1.Equal(resolve(t, newResolver(t, dicetest.NewFixedRoller(5)), attack).(d20.RolledAttack)) {
		t.Error("attack results differing in the roll are equal")
	}
}

func TestResolveLeavesCommandUntouched(t *testing.T) {
	t.Parallel()
	circ := []modifier.RollCircumstance{modifier.Advantage{}}
	cmd := d20.AttackRollCommand{Ability: ability.Strength, Target: 12, Attempt: d20.RollAttempt{
		Modifiers: abilityMods(2).WithOthers(modifier.New(1)), Circumstances: circ,
	}}
	resolve(t, newResolver(t, dicetest.NewFixedRoller(3, 9)), cmd)
	if len(circ) != 1 || cmd.Attempt.Modifiers.Total().Value() != 3 {
		t.Error("command or its collections changed")
	}
}

func TestResolveIsDeterministicInAnyOrder(t *testing.T) {
	t.Parallel()
	rows := caseTable()
	run := func(tc row) d20.Result {
		return resolve(t, newResolver(t, dicetest.NewFixedRoller(tc.rolls...)), tc.cmd)
	}
	forward := make([]d20.Result, len(rows))
	for i, tc := range rows {
		forward[i] = run(tc)
	}
	for i := len(rows) - 1; i >= 0; i-- {
		if !sameResult(forward[i], run(rows[i])) {
			t.Errorf("%s: result differs between runs", rows[i].name)
		}
	}
}

func sameResult(a, b d20.Result) bool {
	switch x := a.(type) {
	case d20.RolledCheck:
		y, ok := b.(d20.RolledCheck)
		return ok && x.Equal(y)
	case d20.RolledAttack:
		y, ok := b.(d20.RolledAttack)
		return ok && x.Equal(y)
	case d20.ForcedFailure:
		y, ok := b.(d20.ForcedFailure)
		return ok && x.Ability() == y.Ability() && x.Failure() == y.Failure() && x.Target() == y.Target()
	}
	return false
}

func TestNewResolverRejectsNilRoller(t *testing.T) {
	t.Parallel()
	_, err := d20.NewResolver(nil)
	if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonMissingRoller}) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("err = %v", err)
	}
}

func TestResolveRejectsInvalidCommands(t *testing.T) {
	t.Parallel()
	good := rollAttempt(0)
	tests := []struct {
		name string
		cmd  d20.Command
		want d20.Reason
	}{
		{"nil command", nil, d20.ReasonUnknownCommand},
		{"pointer command", &d20.AttackRollCommand{Ability: ability.Strength, Attempt: good}, d20.ReasonUnknownCommand},
		{"zero ability on check", d20.AbilityCheckCommand{Attempt: good}, d20.ReasonUnknownAbility},
		{"undefined ability on save", d20.SavingThrowCommand{Ability: 99, Attempt: good}, d20.ReasonUnknownAbility},
		{"undefined ability on attack", d20.AttackRollCommand{Ability: 99, Attempt: good}, d20.ReasonUnknownAbility},
		{"nil check attempt", d20.AbilityCheckCommand{Ability: ability.Strength}, d20.ReasonMissingAttempt},
		{"nil save attempt", d20.SavingThrowCommand{Ability: ability.Strength}, d20.ReasonMissingAttempt},
		{"pointer check attempt", d20.AbilityCheckCommand{Ability: ability.Strength, Attempt: &good}, d20.ReasonMissingAttempt},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rr := newRecorded(t, 10)
			res, err := rr.resolver.Resolve(context.Background(), tc.cmd)
			if res != nil || !errors.Is(err, &d20.InvalidError{Reason: tc.want}) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("res %v err %v, want reason %v", res, err, tc.want)
			}
			if len(rr.roller.Requests()) != 0 || len(rr.rec.Tracer.Spans()) != 0 || len(rr.rec.Meter.Measurements()) != 0 {
				t.Error("an invalid command rolled dice or emitted signals")
			}
		})
	}
}

type brokenRoller struct {
	fn func(dice.Expression) (dice.Result, error)
}

func (b brokenRoller) Roll(e dice.Expression) (dice.Result, error) { return b.fn(e) }

func TestResolveRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("backend down")
	answer := func(e dice.Expression, die dice.Die, n, face int) (dice.Result, error) {
		_ = e
		rolls := make([]dice.DieRoll, n)
		for i := range rolls {
			rolls[i] = dice.MustDieRoll(die, face)
		}
		return dice.NewResult(dice.MustExpression(n, die, 0), rolls)
	}
	tests := []struct {
		name        string
		fn          func(dice.Expression) (dice.Result, error)
		wantIs      error
		wantFailure string
	}{
		{"error", func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }, cause, d20.ValueRollerUnavailable},
		{"panic", func(dice.Expression) (dice.Result, error) { panic("boom") }, srderr.ErrPortPanic, d20.ValueRollerUnavailable},
		{"different count", func(e dice.Expression) (dice.Result, error) { return answer(e, dice.D20(), 2, 5) }, srderr.ErrContractViolation, d20.ValueRollerInvalidResult},
		{"different die", func(e dice.Expression) (dice.Result, error) { return answer(e, dice.D6(), 1, 4) }, srderr.ErrContractViolation, d20.ValueRollerInvalidResult},
		{"zero result", func(dice.Expression) (dice.Result, error) { return dice.Result{}, nil }, srderr.ErrContractViolation, d20.ValueRollerInvalidResult},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			rr := newRecorded(t)
			r := newResolver(t, brokenRoller{func(e dice.Expression) (dice.Result, error) { calls++; return tc.fn(e) }},
				d20.WithTelemetry(rr.rec.Telemetry()))
			cmd := d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)}

			res, err := r.Resolve(context.Background(), cmd)
			var pe *srderr.PortError
			if res != nil || !errors.Is(err, tc.wantIs) || !errors.As(err, &pe) || pe.Port != dice.Port {
				t.Fatalf("res %v err %v, want port error matching %v", res, err, tc.wantIs)
			}
			if calls != 1 {
				t.Errorf("roller called %d times, want 1", calls)
			}

			span := rr.rec.Tracer.Spans()[0]
			if span.Status.String() != "error" || len(span.Errors) != 1 || !errors.Is(span.Errors[0], tc.wantIs) {
				t.Errorf("span status %v errors %v", span.Status, span.Errors)
			}
			if v, _ := span.AttrValue(d20.AttrRollerFailure); v != tc.wantFailure {
				t.Errorf("span roller_failure = %v, want %s", v, tc.wantFailure)
			}
			errs := rr.rec.Meter.MeasurementsNamed(d20.MetricErrors)
			if len(errs) != 1 || errs[0].Value != 1 || errs[0].Unit != "{error}" {
				t.Fatalf("errors counter = %+v", errs)
			}
			wantAttrs := map[string]any{
				d20.AttrKind:          d20.ValueKindAbilityCheck,
				"error.type":          "dice_roller_error",
				d20.AttrRollerFailure: tc.wantFailure,
			}
			if got := attrMap(errs[0].Attrs); !mapsEqual(got, wantAttrs) {
				t.Errorf("errors attrs = %v, want %v", got, wantAttrs)
			}
			if n := len(rr.rec.Meter.MeasurementsNamed(d20.MetricResolved)); n != 0 {
				t.Errorf("resolved incremented %d times for a failure", n)
			}
			if d := rr.rec.Meter.MeasurementsNamed(d20.MetricDuration); len(d) != 1 {
				t.Errorf("duration recorded %d times, want once for a failed call", len(d))
			}
		})
	}
}

func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestResolveRollerErrorIsNotAResult(t *testing.T) {
	t.Parallel()
	r := newResolver(t, dicetest.NewFixedRoller()) // exhausted at once
	_, err := r.Resolve(context.Background(), d20.AttackRollCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)})
	if !errors.Is(err, dicetest.ErrExhausted) {
		t.Errorf("err = %v, want ErrExhausted", err)
	}
}

func TestResolveExhaustedScriptIsAPortError(t *testing.T) {
	t.Parallel()
	// The script holds one die but advantage asks for two.
	r := newResolver(t, dicetest.NewFixedRoller(5))
	cmd := d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0, modifier.Advantage{})}
	res, err := r.Resolve(context.Background(), cmd)
	var pe *srderr.PortError
	if res != nil || !errors.Is(err, dicetest.ErrExhausted) || !errors.As(err, &pe) || pe.Port != dice.Port {
		t.Errorf("res %v err %v, want a dice port error wrapping ErrExhausted", res, err)
	}
}

func TestResolveCancelledContextDoesNotAffectResult(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	r := newResolver(t, newFixed([]int{10}), d20.WithTelemetry(rec.Telemetry()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := r.Resolve(ctx, d20.AbilityCheckCommand{Ability: ability.Strength, Target: 15, Attempt: rollAttempt(5)})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.Outcome() != d20.Success {
		t.Errorf("outcome = %v", res.Outcome())
	}
	if n := len(rec.Tracer.Spans()); n != 1 {
		t.Errorf("spans = %d, want 1", n)
	}
}

func TestZeroResolverReturnsError(t *testing.T) {
	t.Parallel()
	var r d20.Resolver
	_, err := r.Resolve(context.Background(), d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)})
	if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonMissingRoller}) {
		t.Errorf("err = %v", err)
	}
}
