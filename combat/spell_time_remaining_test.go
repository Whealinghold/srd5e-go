package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - TimedRemaining needs at least one second (boundary and invalid input).
// Path: per case.
func TestNewSpellTimeRemainingNeedsAtLeastOneSecond(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		seconds                 int
		wantErr                 bool
	}{
		{"zero seconds is refused", "FR-003", "unhappy (boundary)", 0, true},
		{"minus one second is refused", "FR-003", "unhappy (invalid input)", -1, true},
		{"one second is the least", "FR-003", "unhappy (boundary)", 1, false},
		{"six seconds is one round", "FR-003", "happy", 6, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewSpellTimeRemaining(c.seconds)
			if c.wantErr {
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Fatalf("NewSpellTimeRemaining(%d) error = %v, want ErrInvalidInput [%s]", c.seconds, err, c.requirement)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewSpellTimeRemaining(%d) error = %v [%s]", c.seconds, err, c.requirement)
			}
			if got, ok := r.Seconds(); !ok || got != c.seconds || r.IsOpenEnded() {
				t.Errorf("Seconds() = %d, %v; want %d timed [%s]", got, ok, c.seconds, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - FromDuration converts each timed unit to seconds, the maximum for upTo and concentrationUpTo (SRD 5.2.1, "Duration", p. 106).
// Path: happy.
func TestSpellTimeRemainingFromDurationConvertsTimedUnits(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		amount int
		unit   content.DurationUnit
		limit  content.DurationLimit
		want   int
	}{
		{"rounds", 10, content.DurationUnitRounds, content.LimitFixed, 60},
		{"minutes", 2, content.DurationUnitMinutes, content.LimitFixed, 120},
		{"hours", 1, content.DurationUnitHours, content.LimitFixed, 3600},
		{"days", 1, content.DurationUnitDays, content.LimitFixed, 86400},
		{"up to the maximum in rounds", 3, content.DurationUnitRounds, content.LimitUpTo, 18},
		{"concentration up to in minutes", 1, content.DurationUnitMinutes, content.LimitConcentrationUpTo, 60},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := content.NewSpellDurationTimed(c.amount, c.unit, c.limit)
			if err != nil {
				t.Fatal(err)
			}
			r, err := combat.SpellTimeRemainingFromDuration(d)
			if err != nil {
				t.Fatalf("FromDuration error = %v [FR-003]", err)
			}
			if got, ok := r.Seconds(); !ok || got != c.want {
				t.Errorf("FromDuration seconds = %d, %v; want %d [FR-003]", got, ok, c.want)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - untilDispelled, untilDispelledOrTriggered and special have no number of seconds (SRD 5.2.1, "Duration", p. 106).
// Path: happy.
func TestSpellTimeRemainingFromDurationOpenEndedVariants(t *testing.T) {
	t.Parallel()
	for name, d := range map[string]content.SpellDuration{
		"until dispelled":              content.NewSpellDurationUntilDispelled(),
		"until dispelled or triggered": content.NewSpellDurationUntilDispelledOrTriggered(),
		"special":                      content.NewSpellDurationSpecial(),
	} {
		r, err := combat.SpellTimeRemainingFromDuration(d)
		if err != nil {
			t.Fatalf("%s: FromDuration error = %v [FR-003]", name, err)
		}
		if !r.IsOpenEnded() {
			t.Errorf("%s: IsOpenEnded() = false, want true [FR-003]", name)
		}
		if _, ok := r.Seconds(); ok {
			t.Errorf("%s: Seconds() reports a number for an open-ended spell [FR-003]", name)
		}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - an instantaneous duration is never an ongoing spell (invalid input, SRD 5.2.1, "Duration", p. 106).
// Path: unhappy (invalid input).
func TestSpellTimeRemainingFromDurationRefusesInstantaneous(t *testing.T) {
	t.Parallel()
	_, err := combat.SpellTimeRemainingFromDuration(content.NewSpellDurationInstantaneous())
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("FromDuration(instantaneous) error = %v, want ErrInvalidInput [FR-003]", err)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - a conversion that would overflow int is refused, at the boundary (arithmetic limits of the quantity).
// Path: unhappy (boundary).
func TestSpellTimeRemainingFromDurationOverflowBoundary(t *testing.T) {
	t.Parallel()
	const perDay = 86400
	cases := []struct {
		name    string
		amount  int
		wantErr bool
	}{
		{"largest amount that fits", math.MaxInt / perDay, false},
		{"one more day overflows", math.MaxInt/perDay + 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := content.NewSpellDurationTimed(c.amount, content.DurationUnitDays, content.LimitFixed)
			if err != nil {
				t.Fatalf("NewSpellDurationTimed(%d) error = %v", c.amount, err)
			}
			_, err = combat.SpellTimeRemainingFromDuration(d)
			if c.wantErr != errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("FromDuration(%d days) error = %v, want refused=%v [FR-003]", c.amount, err, c.wantErr)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - the zero remaining time is invalid and reports no number (invalid zero value).
// Path: unhappy (invalid input).
func TestSpellTimeRemainingZeroValueIsInvalid(t *testing.T) {
	t.Parallel()
	var r combat.SpellTimeRemaining
	if !r.IsZero() {
		t.Fatal("zero SpellTimeRemaining IsZero() = false, want true [FR-003]")
	}
	if got, ok := r.Seconds(); ok || got != 0 {
		t.Errorf("zero Seconds() = %d, %v; want 0, false [FR-003]", got, ok)
	}
}
