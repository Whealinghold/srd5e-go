package d20_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// row is one end-to-end case: a command, the scripted dice and the expected
// result.
type row struct {
	name     string
	cmd      d20.Command
	rolls    []int // scripted dice; empty means no roller request is expected
	wantDice int   // expected size of the one d20 request
	want     want
}

type want struct {
	kind     d20.Kind
	total    int
	outcome  d20.Outcome
	critical bool
	reason   d20.ForcedFailureReason
	mode     modifier.RollMode
	target   int
}

func rollAttempt(abilityMod int, circ ...modifier.RollCircumstance) d20.RollAttempt {
	return d20.RollAttempt{Modifiers: abilityMods(abilityMod), Circumstances: circ}
}

func caseTable() []row {
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	check := func(a ability.Ability, dc d20.DifficultyClass, at d20.CheckAttempt) d20.Command {
		return d20.AbilityCheckCommand{Ability: a, Target: dc, Attempt: at}
	}
	save := func(a ability.Ability, dc d20.DifficultyClass, at d20.SaveAttempt) d20.Command {
		return d20.SavingThrowCommand{Ability: a, Target: dc, Attempt: at}
	}
	attack := func(ac d20.ArmorClass, at d20.RollAttempt) d20.Command {
		return d20.AttackRollCommand{Ability: ability.Strength, Target: ac, Attempt: at}
	}
	const (
		c = d20.AbilityCheck
		s = d20.SavingThrow
		a = d20.AttackRoll
	)
	ok, no := d20.Success, d20.Failure
	return []row{
		{"check meets DC", check(ability.Strength, 15, rollAttempt(5)), []int{10}, 1, want{kind: c, total: 15, outcome: ok, target: 15}},
		{"save fails", save(ability.Dexterity, 10, rollAttempt(1)), []int{8}, 1, want{kind: s, total: 9, outcome: no, target: 10}},
		{"attack hits", attack(15, rollAttempt(3)), []int{12}, 1, want{kind: a, total: 15, outcome: ok, target: 15}},
		{"total above 30", check(ability.Strength, 30, rollAttempt(15)), []int{20}, 1, want{kind: c, total: 35, outcome: ok, target: 30}},
		{"total below zero", check(ability.Strength, 5, d20.RollAttempt{Modifiers: abilityMods(-5).WithOthers(modifier.New(-2))}), []int{1}, 1, want{kind: c, total: -6, outcome: no, target: 5}},
		{"extreme low, normal", check(ability.Strength, 5, rollAttempt(0)), []int{1}, 1, want{kind: c, total: 1, outcome: no, target: 5}},
		{"extreme high, normal", attack(30, rollAttempt(0)), []int{20}, 1, want{kind: a, total: 20, outcome: ok, critical: true, target: 30}},
		{"advantage uses the higher die", check(ability.Strength, 15, rollAttempt(0, adv)), []int{18, 3}, 2, want{kind: c, total: 18, outcome: ok, mode: modifier.WithAdvantage, target: 15}},
		{"disadvantage uses the lower die", check(ability.Strength, 15, rollAttempt(0, dis)), []int{18, 3}, 2, want{kind: c, total: 3, outcome: no, mode: modifier.WithDisadvantage, target: 15}},
		{"advantage, both dice 1", attack(5, rollAttempt(20, adv)), []int{1, 1}, 2, want{kind: a, total: 21, outcome: no, mode: modifier.WithAdvantage, target: 5}},
		{"disadvantage, both dice 20", attack(30, rollAttempt(-3, dis)), []int{20, 20}, 2, want{kind: a, total: 17, outcome: ok, critical: true, mode: modifier.WithDisadvantage, target: 30}},
		{"advantage, both dice 20", attack(30, rollAttempt(-3, adv)), []int{20, 20}, 2, want{kind: a, total: 17, outcome: ok, critical: true, mode: modifier.WithAdvantage, target: 30}},
		{"disadvantage, both dice 1", attack(5, rollAttempt(20, dis)), []int{1, 1}, 2, want{kind: a, total: 21, outcome: no, mode: modifier.WithDisadvantage, target: 5}},
		{"disadvantage, natural 1 used", attack(5, rollAttempt(10, dis)), []int{20, 1}, 2, want{kind: a, total: 11, outcome: no, mode: modifier.WithDisadvantage, target: 5}},
		{"advantage and disadvantage cancel", check(ability.Strength, 15, rollAttempt(0, adv, dis)), []int{14}, 1, want{kind: c, total: 14, outcome: no, target: 15}},
		{"many sources of advantage stay advantage", check(ability.Strength, 15, rollAttempt(0, adv, adv)), []int{2, 16}, 2, want{kind: c, total: 16, outcome: ok, mode: modifier.WithAdvantage, target: 15}},
		{"chosen to fail a save", save(ability.Dexterity, 10, d20.ChooseToFail{}), nil, 0, want{kind: s, outcome: no, reason: d20.ChosenToFail, target: 10}},
		{"automatic failure on a check", check(ability.Wisdom, 10, d20.AutomaticFailure{}), nil, 0, want{kind: c, outcome: no, reason: d20.Automatic, target: 10}},
		{"automatic failure on a save", save(ability.Constitution, 12, d20.AutomaticFailure{}), nil, 0, want{kind: s, outcome: no, reason: d20.Automatic, target: 12}},
	}
}

func newResolver(t *testing.T, roller d20.Roller, opts ...d20.Option) d20.Resolver {
	t.Helper()
	r, err := d20.NewResolver(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// recorded is a Resolver wired to a scripted roller and recording telemetry.
type recorded struct {
	resolver d20.Resolver
	roller   *dicetest.FixedRoller
	rec      *telemetrytest.Recorder
}

func newRecorded(t *testing.T, rolls ...int) recorded {
	t.Helper()
	rec := telemetrytest.NewRecorder()
	roller := dicetest.NewFixedRoller(rolls...)
	clock := telemetrytest.NewClock(time.Unix(0, 0), 250*time.Millisecond)
	return recorded{newResolver(t, roller, d20.WithTelemetry(rec.Telemetry()), d20.WithClock(clock.Now)), roller, rec}
}

func resolve(t *testing.T, r d20.Resolver, cmd d20.Command) d20.Result {
	t.Helper()
	res, err := r.Resolve(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	return res
}

func attrMap(attrs []telemetry.Attr) map[string]any {
	m := make(map[string]any, len(attrs))
	for _, a := range attrs {
		m[a.Key()] = a.Value()
	}
	return m
}

func d20Expr(n int) dice.Expression { return dice.MustExpression(n, dice.D20(), 0) }

func newFixed(rolls []int) *dicetest.FixedRoller { return dicetest.NewFixedRoller(rolls...) }
