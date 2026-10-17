package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// A hero turns: an Advantage check, a save, an attack roll and a chosen
// failure, resolved with a scripted roller and recorded telemetry.
func TestHeroTurn(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Rolls in request order: the Advantage check takes two d20s, the save and
	// the attack one each; the chosen failure takes none.
	roller := dicetest.NewFixedRoller(15, 6, 9, 17)
	rec := telemetrytest.NewRecorder()
	resolver, err := d20.NewResolver(roller, d20.WithTelemetry(rec.Telemetry()))
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	// (1) A Strength check with Advantage against DC 15.
	res, err := resolver.Resolve(ctx, d20.AbilityCheckCommand{
		Ability: ability.Strength,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{
			Modifiers:     d20.NewRollModifiers(modifier.New(3)),
			Circumstances: []modifier.RollCircumstance{modifier.Advantage{}},
		},
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	check, ok := res.(d20.RolledCheck)
	if !ok {
		t.Fatalf("check result is %T, want RolledCheck", res)
	}
	c := check.Check()
	if c.Roll().Mode() != modifier.WithAdvantage || c.Roll().Natural() != 15 || c.Total() != 18 || c.Outcome() != d20.Success {
		t.Errorf("check = mode %v natural %d total %d outcome %v", c.Roll().Mode(), c.Roll().Natural(), c.Total(), c.Outcome())
	}

	// The kept die is rerolled with Heroic Inspiration; keeping is a value too.
	rerolled, err := c.Roll().ApplyReroll(d20.RerollDie{Index: 1, Face: 14})
	if err != nil {
		t.Fatalf("reroll: %v", err)
	}
	if d := rerolled.Dice(); d[0] != 15 || d[1] != 14 || rerolled.Natural() != 15 {
		t.Errorf("rerolled dice %v natural %d, want [15 14] and 15", d, rerolled.Natural())
	}
	if kept, err := c.Roll().ApplyReroll(d20.KeepRoll{}); err != nil || kept.Natural() != 15 {
		t.Errorf("KeepRoll = %v, %v", kept, err)
	}

	// (2) A Wisdom saving throw against a medium DC.
	res, err = resolver.Resolve(ctx, d20.SavingThrowCommand{
		Ability: ability.Wisdom,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(1))},
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if res.Kind() != d20.SavingThrow || res.Outcome() != d20.Failure {
		t.Errorf("save = %v %v, want saving throw failure (9+1 vs 15)", res.Kind(), res.Outcome())
	}

	// (3) A Strength attack roll against Armor Class 13.
	res, err = resolver.Resolve(ctx, d20.AttackRollCommand{
		Ability: ability.Strength,
		Target:  13,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(5))},
	})
	if err != nil {
		t.Fatalf("attack: %v", err)
	}
	attack, ok := res.(d20.RolledAttack)
	if !ok {
		t.Fatalf("attack result is %T, want RolledAttack", res)
	}
	if a := attack.Attack(); a.Total() != 22 || !a.Hit() || a.IsCriticalHit() {
		t.Errorf("attack total %d hit %v crit %v, want 22 true false", a.Total(), a.Hit(), a.IsCriticalHit())
	}

	// (4) The hero chooses to fail the next saving throw; a failure is a
	// normal outcome, not an error, and the roller is not consulted.
	res, err = resolver.Resolve(ctx, d20.SavingThrowCommand{
		Ability: ability.Dexterity,
		Target:  d20.DCEasy,
		Attempt: d20.ChooseToFail{},
	})
	if err != nil {
		t.Fatalf("chosen failure: %v", err)
	}
	forced, ok := res.(d20.ForcedFailure)
	if !ok || forced.Failure().Reason() != d20.ChosenToFail || forced.Target() != d20.DCEasy {
		t.Fatalf("chosen failure = %#v", res)
	}

	// Each call was one span named by the signal constant.
	spans := rec.Tracer.Spans()
	if len(spans) != 4 {
		t.Fatalf("got %d spans, want 4", len(spans))
	}
	for _, s := range spans {
		if s.Name != d20.SpanResolve {
			t.Errorf("span %q, want %q", s.Name, d20.SpanResolve)
		}
	}

	// The roller saw one request per rolled test and none for the failure.
	var got []string
	for _, r := range roller.Requests() {
		got = append(got, r.Notation())
	}
	if want := []string{"2d20", "1d20", "1d20"}; !equal(got, want) {
		t.Errorf("roller requests = %v, want %v", got, want)
	}
	telemetrytest.RequireNoViolations(t, rec)
}

// A roller outage surfaces as a port error the caller can branch on, and the
// resolver records it as an error span.
func TestResolveReportsRollerOutage(t *testing.T) {
	t.Parallel()
	down := errors.New("backend down")
	roller := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, down })
	rec := telemetrytest.NewRecorder()
	resolver, err := d20.NewResolver(roller, d20.WithTelemetry(rec.Telemetry()))
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	_, err = resolver.Resolve(context.Background(), d20.AttackRollCommand{
		Ability: ability.Strength, Target: 10, Attempt: d20.RollAttempt{},
	})

	var pe *srderr.PortError
	if !errors.As(err, &pe) || !errors.Is(err, down) {
		t.Fatalf("err = %v, want a PortError wrapping the outage", err)
	}
	if got := rec.Tracer.Spans()[0].Status; got != telemetry.StatusError {
		t.Errorf("span status = %v, want error", got)
	}
}

// Untrusted commands are rejected with a typed error; nothing panics.
func TestResolveRejectsInvalidCommands(t *testing.T) {
	t.Parallel()
	resolver, err := d20.NewResolver(dicetest.NewMinimumRoller())
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	tests := []struct {
		name string
		cmd  d20.Command
	}{
		{"nil command", nil},
		{"unknown ability", d20.AttackRollCommand{Ability: 0, Target: 10}},
		{"missing attempt", d20.AbilityCheckCommand{Ability: ability.Strength, Target: d20.DCEasy}},
		{"pointer command", &d20.AttackRollCommand{Ability: ability.Strength}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolver.Resolve(context.Background(), tc.cmd)
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
	if _, err := d20.NewResolver(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewResolver(nil) err = %v, want ErrInvalidInput", err)
	}
}

func equal(a, b []string) bool {
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
