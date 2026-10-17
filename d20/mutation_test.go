package d20_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: none - pins current behaviour of the critical range at its
// boundaries (SRD 5.2.1, "Rolling 20 or 1", p. 7, for the natural 20 case).
// Path: a natural value at or above the range, but not 20, is a Critical Hit
// that hits whatever the AC; below the range it is compared with the AC.
func TestResolveAttackCriticalRangeBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		face     int
		critical int
		want     bool
	}{
		{"19 above range 18", 19, 18, true},
		{"18 at range 18", 18, 18, true},
		{"20 above range 18", 20, 18, true},
		{"19 at range 19", 19, 19, true},
		{"20 above range 19", 20, 19, true},
		{"18 below range 19", 18, 19, false},
		{"17 below range 18", 17, 18, false},
		{"19 below range 20", 19, 20, false},
		{"1 in range 18", 1, 18, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := d20.ResolveAttack(normalRoll(tc.face), abilityMods(-30), 30, tc.critical)
			if err != nil {
				t.Fatal(err)
			}
			if got.IsCriticalHit() != tc.want || got.Hit() != tc.want {
				t.Errorf("critical %v hit %v, want both %v", got.IsCriticalHit(), got.Hit(), tc.want)
			}
		})
	}
}

// Requirement: none - pins current behaviour of FailWithoutRoll's argument
// validation.
// Path: every reason outside ChosenToFail and Automatic is rejected, below and
// above the defined values, before the kind rules apply.
func TestFailWithoutRoll_UnknownReasons(t *testing.T) {
	t.Parallel()
	for _, reason := range []d20.ForcedFailureReason{-1, 0, 3, 99} {
		for _, kind := range []d20.Kind{d20.AbilityCheck, d20.SavingThrow, d20.AttackRoll} {
			_, err := d20.FailWithoutRoll(kind, reason)
			if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonUnknownReason}) {
				t.Errorf("kind %v reason %d: err = %v", kind, reason, err)
			}
		}
	}
	for _, kind := range []d20.Kind{-1, 0, 4, 99} {
		_, err := d20.FailWithoutRoll(kind, d20.Automatic)
		if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonUnknownKind}) {
			t.Errorf("kind %d: err = %v", kind, err)
		}
	}
}

// Requirement: none - pins current behaviour of InvalidError matching.
// Path: an InvalidError matches only an InvalidError of the same Reason,
// whichever of the two has the larger Reason.
func TestInvalidErrorIsMatchesOnlyTheSameReason(t *testing.T) {
	t.Parallel()
	low := &d20.InvalidError{Reason: d20.ReasonIndexOutOfRange}
	high := &d20.InvalidError{Reason: d20.ReasonMissingRoller}
	if errors.Is(low, high) || errors.Is(high, low) {
		t.Error("errors of different reasons match")
	}
	if !errors.Is(low, &d20.InvalidError{Reason: d20.ReasonIndexOutOfRange, Detail: "other"}) {
		t.Error("errors of the same reason do not match")
	}
}

// Requirement: none - pins current behaviour of Roll.ReplaceDie validation.
// Path: an index at, above and below the dice is rejected with a detail naming
// the valid range.
func TestReplaceDieIndexOutOfRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		roll  d20.Roll
		index int
		want  string
	}{
		{"one die, at n", normalRoll(5), 1, "index 1 not in 0..0"},
		{"one die, above n", normalRoll(5), 2, "index 2 not in 0..0"},
		{"two dice, at n", d20.MustRoll(modifier.WithAdvantage, 3, 4), 2, "index 2 not in 0..1"},
		{"two dice, above n", d20.MustRoll(modifier.WithAdvantage, 3, 4), 5, "index 5 not in 0..1"},
		{"negative", d20.MustRoll(modifier.WithAdvantage, 3, 4), -1, "index -1 not in 0..1"},
		{"zero roll", d20.Roll{}, 0, "index 0 not in 0..-1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.roll.ReplaceDie(tc.index, 7)
			if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonIndexOutOfRange}) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want reason index out of range with %q", err, tc.want)
			}
		})
	}
}

// Requirement: none - pins current behaviour of Equal on results.
// Path: results differing in exactly one field are unequal, in both argument
// orders.
func TestResultEqualFieldByField(t *testing.T) {
	t.Parallel()
	adv := d20.MustRoll(modifier.WithAdvantage, 10, 5)
	otherMods := d20.NewRollModifiers(modifier.New(0)).WithOthers(modifier.New(1))
	check := func(kind d20.Kind, roll d20.Roll, mods d20.RollModifiers, dc d20.DifficultyClass) d20.CheckResult {
		c, err := d20.ResolveCheck(kind, roll, mods, dc)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	base := check(d20.AbilityCheck, normalRoll(10), abilityMods(1), 12)
	for name, v := range map[string]d20.CheckResult{
		"kind":      check(d20.SavingThrow, normalRoll(10), abilityMods(1), 12),
		"roll":      check(d20.AbilityCheck, adv, abilityMods(1), 12),
		"modifiers": check(d20.AbilityCheck, normalRoll(10), otherMods, 12),
		"target":    check(d20.AbilityCheck, normalRoll(10), abilityMods(1), 13),
		"outcome":   check(d20.AbilityCheck, normalRoll(10), abilityMods(1), 11),
	} {
		if base.Equal(v) || v.Equal(base) {
			t.Errorf("CheckResult differing in %s is equal", name)
		}
	}

	attack := func(roll d20.Roll, mods d20.RollModifiers, ac d20.ArmorClass, cr int) d20.AttackRollResult {
		a, err := d20.ResolveAttack(roll, mods, ac, cr)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	ab := attack(normalRoll(10), abilityMods(1), 14, 20)
	for name, v := range map[string]d20.AttackRollResult{
		"roll":          attack(adv, abilityMods(1), 14, 20),
		"modifiers":     attack(normalRoll(10), otherMods, 14, 20),
		"target":        attack(normalRoll(10), abilityMods(1), 15, 20),
		"target, lower": attack(normalRoll(10), abilityMods(1), 13, 20),
		"outcome":       attack(normalRoll(10), abilityMods(1), 11, 20),
		"critical":      attack(normalRoll(20), abilityMods(-9), 14, 20),
		"range":         attack(normalRoll(10), abilityMods(1), 14, 19),
	} {
		if ab.Equal(v) || v.Equal(ab) {
			t.Errorf("AttackRollResult differing in %s is equal", name)
		}
	}
	if !ab.Equal(attack(normalRoll(10), abilityMods(1), 14, 20)) {
		t.Error("identical attack results are unequal")
	}
}

// Requirement: none - pins current behaviour of Equal on wrapped results.
// Path: a different ability label makes otherwise identical results unequal.
func TestRolledResultEqualComparesAbility(t *testing.T) {
	t.Parallel()
	c, _ := d20.ResolveCheck(d20.AbilityCheck, normalRoll(10), abilityMods(1), 12)
	a, _ := d20.ResolveAttack(normalRoll(10), abilityMods(1), 12, 20)
	c1, _ := d20.NewRolledCheck(ability.Strength, c)
	c2, _ := d20.NewRolledCheck(ability.Dexterity, c)
	a1, _ := d20.NewRolledAttack(ability.Strength, a)
	a2, _ := d20.NewRolledAttack(ability.Dexterity, a)
	if c1.Equal(c2) || c2.Equal(c1) || a1.Equal(a2) || a2.Equal(a1) {
		t.Error("results with different abilities are equal")
	}
}

// Requirement: none - pins current behaviour of WithClock.
// Path: a nil clock is ignored and the Resolver keeps working with time.Now.
func TestWithClockNilKeepsDefaultClock(t *testing.T) {
	t.Parallel()
	r := newResolver(t, dicetest.NewFixedRoller(10), d20.WithClock(nil))
	cmd := d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)}
	if res, err := r.Resolve(context.Background(), cmd); err != nil || res.Outcome() != d20.Success {
		t.Fatalf("res %v err %v", res, err)
	}
	calls := 0
	custom := func() time.Time { calls++; return time.Unix(0, 0) }
	r = newResolver(t, dicetest.NewFixedRoller(10), d20.WithClock(custom))
	resolve(t, r, cmd)
	if calls == 0 {
		t.Error("custom clock not used")
	}
}

// Requirement: none - pins current behaviour of WithClock(nil).
// Path: with a nil clock the duration is still recorded, using the default.
func TestWithClockNilStillTimesTheCall(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	r := newResolver(t, dicetest.NewFixedRoller(10), d20.WithTelemetry(rec.Telemetry()), d20.WithClock(nil))
	resolve(t, r, d20.AbilityCheckCommand{Ability: ability.Strength, Target: 10, Attempt: rollAttempt(0)})
	if n := len(rec.Meter.MeasurementsNamed(d20.MetricDuration)); n != 1 {
		t.Errorf("duration recorded %d times, want 1", n)
	}
}

// Requirement: none - pins current behaviour of Reason.String.
// Path: undefined reasons, just past the last one and far past it, read
// "unknown" without panicking.
func TestReasonStringUnknown(t *testing.T) {
	t.Parallel()
	last := d20.ReasonMissingRoller
	for _, r := range []d20.Reason{-1, 0, last + 1, last + 2, 100} {
		if got := r.String(); got != "unknown" {
			t.Errorf("Reason(%d).String() = %q, want unknown", int(r), got)
		}
	}
	if got := last.String(); got != "missing roller" {
		t.Errorf("last reason = %q", got)
	}
}

// Requirement: none - pins current behaviour of the zero results.
// Path: a zero attack result is not a hit.
func TestZeroAttackResultIsNotAHit(t *testing.T) {
	t.Parallel()
	if (d20.AttackRollResult{}).Hit() {
		t.Error("zero AttackRollResult reports a hit")
	}
}
