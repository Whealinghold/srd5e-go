package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func lastChargeUse(t *testing.T, r character.Roller, opts ...character.Option) character.ResolveLastCharge {
	t.Helper()
	return must(character.NewResolveLastCharge(r, opts...))
}

func lastChargeCmd(charges character.ItemCharges, rule character.LastChargeRule) character.ResolveLastChargeCommand {
	return character.ResolveLastChargeCommand{CharacterID: holder, Instance: inst("wand-1"), Charges: charges, Rule: rule}
}

// Requirement: per case (doc/specs/magic-item-effects.md FR-038 to FR-042 - ResolveLastCharge with a fixed roller: faces 1, 2, 19 and 20 for each rule
// shape, one 1d20 request plus one for the regain on a 20, none for automatic and no-effect rules, the regain capped at the maximum, and a rejection
// with charges above 0 that makes no request).
// Path: per case.
func TestResolveLastCharge(t *testing.T) {
	t.Parallel()
	d20 := dice.MustExpression(1, dice.D20(), 0)
	d8plus2 := dice.MustExpression(1, dice.D8(), 2)
	d12plus1 := dice.MustExpression(1, dice.D12(), 1)
	destroyedRule := must(character.NewLastChargeRolled(character.LastChargeDestroyed, character.NoRegain()))
	powerRule := must(character.NewLastChargeRolled(character.LastChargeRetainsBonusLosesOtherProperties, character.RegainsDice(d8plus2)))
	magiRule := must(character.NewLastChargeRolled(character.LastChargeNone, character.RegainsDice(d12plus1)))
	auto := must(character.NewLastChargeAutomatic(character.LastChargeBecomesNonmagical))
	empty := func(max int) character.ItemCharges { return character.MustItemCharges(max, 0) }
	const p = `SRD 5.2.1, "Wand of Magic Missiles", p. 251; "Staff of Power", p. 246; "Staff of the Magi", p. 247`
	cases := []struct {
		name, path  string
		charges     character.ItemCharges
		rule        character.LastChargeRule
		faces       []int
		consequence character.LastChargeConsequence
		current     int
		requests    []dice.Expression
	}{
		{"destroyed on 1", "happy", empty(7), destroyedRule, []int{1}, character.LastChargeDestroyed, 0, []dice.Expression{d20}},
		{"2 changes nothing", "unhappy (boundary)", empty(7), destroyedRule, []int{2}, character.LastChargeNone, 0, []dice.Expression{d20}},
		{"19 changes nothing", "unhappy (boundary)", empty(7), destroyedRule, []int{19}, character.LastChargeNone, 0, []dice.Expression{d20}},
		{"20 with no regain", "unhappy (rule interaction)", empty(7), destroyedRule, []int{20}, character.LastChargeNone, 0, []dice.Expression{d20}},
		{"staff of power 1", "happy", empty(20), powerRule, []int{1}, character.LastChargeRetainsBonusLosesOtherProperties, 0, []dice.Expression{d20}},
		{"staff of power 2", "unhappy (boundary)", empty(20), powerRule, []int{2}, character.LastChargeNone, 0, []dice.Expression{d20}},
		{"staff of power 19", "unhappy (boundary)", empty(20), powerRule, []int{19}, character.LastChargeNone, 0, []dice.Expression{d20}},
		{"staff of power 20 regains 1d8+2", "happy", empty(20), powerRule, []int{20, 4}, character.LastChargeNone, 6, []dice.Expression{d20, d8plus2}},
		{"regain capped at the maximum", "unhappy (boundary)", empty(5), powerRule, []int{20, 8}, character.LastChargeNone, 5, []dice.Expression{d20, d8plus2}},
		{"regain exactly the maximum", "unhappy (boundary)", empty(6), powerRule, []int{20, 4}, character.LastChargeNone, 6, []dice.Expression{d20, d8plus2}},
		{"magi 1 has no consequence", "unhappy (rule interaction)", empty(50), magiRule, []int{1}, character.LastChargeNone, 0, []dice.Expression{d20}},
		{"magi 20 regains 1d12+1", "happy", empty(50), magiRule, []int{20, 3}, character.LastChargeNone, 4, []dice.Expression{d20, d12plus1}},
		{"automatic", "happy", empty(3), auto, nil, character.LastChargeBecomesNonmagical, 0, nil},
		{"no effect", "happy", empty(3), character.NoLastChargeEffect{}, nil, character.LastChargeNone, 0, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := lastChargeUse(t, roller).Do(context.Background(), lastChargeCmd(c.charges, c.rule))
			r, ok := res.(character.ResolveLastChargeResolved)
			if err != nil || !ok {
				t.Fatalf("result = %+v %v [%s]", res, err, p)
			}
			if r.Consequence() != c.consequence || r.Charges().Current() != c.current || r.Charges().Maximum() != c.charges.Maximum() {
				t.Errorf("consequence %v, charges %+v", r.Consequence(), r.Charges())
			}
			ev := r.Event()
			if ev.CharacterID() != holder || ev.InstanceID() != inst("wand-1") || ev.Consequence() != c.consequence {
				t.Errorf("event = %+v", ev)
			}
			if !reflect.DeepEqual(roller.Requests(), c.requests) {
				t.Errorf("requests = %v, want %v", roller.Requests(), c.requests)
			}
		})
	}
	for _, current := range []int{1, 3} {
		roller := dicetest.NewFixedRoller()
		res, err := lastChargeUse(t, roller).Do(context.Background(), lastChargeCmd(character.MustItemCharges(7, current), destroyedRule))
		if r, ok := res.(character.ResolveLastChargeRejected); err != nil || !ok || r.Reason() != character.ChargesRemain || len(roller.Requests()) != 0 {
			t.Errorf("current %d = %+v %v", current, res, err)
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-043, FR-067, section 7 - ResolveLastCharge records one span (with the consequence) and one counter
// increment per call, success for resolved and failure for rejected (no error status), the no-op gives the same result, a roller failure at either
// roll is recorded once as dice_roller_error with the cause preserved and logs nothing.
// Path: per case.
func TestResolveLastChargeSignalsAndFailures(t *testing.T) {
	t.Parallel()
	rule := must(character.NewLastChargeRolled(character.LastChargeDestroyed, character.NoRegain()))
	for _, c := range []struct {
		name    string
		charges character.ItemCharges
		faces   []int
		want    map[string]any
	}{
		{"resolved", character.MustItemCharges(7, 0), []int{1}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "resolved", character.AttrLastChargeConsequence: "destroyed"}},
		{"nothing happens", character.MustItemCharges(7, 0), []int{10}, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "resolved", character.AttrLastChargeConsequence: "none"}},
		{"rejected", character.MustItemCharges(7, 2), nil, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "charges_remain"}},
	} {
		cmd := lastChargeCmd(c.charges, rule)
		rec := telemetrytest.NewRecorder()
		res, err := lastChargeUse(t, dicetest.NewFixedRoller(c.faces...), character.WithTelemetry(rec.Telemetry())).Do(context.Background(), cmd)
		plain, err2 := lastChargeUse(t, dicetest.NewFixedRoller(c.faces...)).Do(context.Background(), cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ", c.name)
		}
		checkOneCall(t, rec, character.SpanResolveLastCharge, "resolve_last_charge", telemetry.StatusUnset, c.want, character.AttrLastChargeConsequence)
		noLogsOrViolations(t, rec)
	}
	cause := errors.New("boom")
	power := must(character.NewLastChargeRolled(character.LastChargeNone, character.RegainsDice(dice.MustExpression(1, dice.D8(), 2))))
	for name, r := range failingRollers(cause) {
		rec := telemetrytest.NewRecorder()
		res, err := lastChargeUse(t, r, character.WithTelemetry(rec.Telemetry())).Do(context.Background(), lastChargeCmd(character.MustItemCharges(7, 0), rule))
		wantPortFailure(t, name, "resolve_last_charge", rec, res, err)
		if name == "fails" && !errors.Is(err, cause) {
			t.Error("the cause is lost")
		}
	}
	// the regain roll fails after a 20: the first roll succeeds, the second breaks
	calls := 0
	second := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		calls++
		if calls == 1 {
			return dice.NewResult(e, []dice.DieRoll{dice.MustDieRoll(dice.D20(), 20)})
		}
		return dice.Result{}, cause
	})
	rec := telemetrytest.NewRecorder()
	res, err := lastChargeUse(t, second, character.WithTelemetry(rec.Telemetry())).Do(context.Background(), lastChargeCmd(character.MustItemCharges(7, 0), power))
	wantPortFailure(t, "regain roll", "resolve_last_charge", rec, res, err)
	if calls != 2 {
		t.Errorf("%d rolls", calls)
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-038, FR-067 - invalid commands (zero holder, blank instance, zero charges, nil or zero-valued rule),
// a zero use case, a nil roller and a cancelled context are refused before any signal.
// Path: unhappy (invalid input).
func TestResolveLastChargeInvalidInput(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller()
	use := lastChargeUse(t, roller, character.WithTelemetry(rec.Telemetry()))
	good := lastChargeCmd(character.MustItemCharges(7, 0), character.NoLastChargeEffect{})
	for name, mut := range map[string]func(*character.ResolveLastChargeCommand){
		"zero holder":      func(c *character.ResolveLastChargeCommand) { c.CharacterID = character.Identifier{} },
		"blank instance":   func(c *character.ResolveLastChargeCommand) { c.Instance = "" },
		"zero charges":     func(c *character.ResolveLastChargeCommand) { c.Charges = character.ItemCharges{} },
		"nil rule":         func(c *character.ResolveLastChargeCommand) { c.Rule = nil },
		"zero automatic":   func(c *character.ResolveLastChargeCommand) { c.Rule = character.LastChargeAutomatic{} },
		"zero rolled rule": func(c *character.ResolveLastChargeCommand) { c.Rule = character.LastChargeRolled{} },
	} {
		cmd := good
		mut(&cmd)
		if r, err := use.Do(context.Background(), cmd); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v %v", name, r, err)
		}
	}
	if r, err := (character.ResolveLastCharge{}).Do(context.Background(), good); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	if _, err := character.NewResolveLastCharge(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := use.Do(cancelled, good); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Error("a refused call emitted a signal or rolled")
	}
}
