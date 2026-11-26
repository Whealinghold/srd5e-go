package combat_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewDeathCircumstances(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		since                   time.Duration
		body                    combat.BodyState
		wantName                string
		wantValue               int64
	}{
		{"zero duration", rrSpec + "FR-008; table 4.1b just born dead", "unhappy (empty and minimal)", 0, combat.Intact, "", 0},
		{"one nanosecond", rrSpec + "FR-008", "unhappy (boundary)", 1, combat.Intact, "", 0},
		{"the largest duration", rrSpec + "FR-008", "unhappy (boundary)", math.MaxInt64, combat.NoBody, "", 0},
		{"negative one second", rrSpec + "FR-008; table 4.1b negative time", "unhappy (invalid input)", -time.Second, combat.Intact, "time_since_death", int64(-time.Second)},
		{"the smallest duration", rrSpec + "FR-008", "unhappy (invalid input)", math.MinInt64, combat.Intact, "time_since_death", math.MinInt64},
		{"a body state below the first", rrSpec + "FR-008", "unhappy (invalid input)", 0, -1, "body_state", -1},
		{"a body state above the last", rrSpec + "FR-008", "unhappy (invalid input)", 0, 4, "body_state", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewDeathCircumstances(c.since, true, true, c.body)
			if c.wantName == "" {
				if err != nil || got.TimeSinceDeath() != c.since || !got.DiedOfOldAge() || !got.WasUndead() || got.Body() != c.body {
					t.Errorf("NewDeathCircumstances(%v) = %+v, %v, want the facts back [%s; %s]", c.since, got, err, c.requirement, c.path)
				}
				return
			}
			var re *combat.RestorationRangeError
			if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &re) || re.Name != c.wantName || re.Value != c.wantValue {
				t.Errorf("NewDeathCircumstances(%v, %v) error = %v, want a range error of %s = %d [%s; %s]", c.since, c.body, err, c.wantName, c.wantValue, c.requirement, c.path)
			}
			if err.Error() == "" {
				t.Error("the range error renders as an empty string")
			}
			if got != (combat.DeathCircumstances{}) {
				t.Errorf("NewDeathCircumstances returned %+v with an error, want the zero value", got)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-008 - MustDeathCircumstances returns the value for good input and panics with the
// error of NewDeathCircumstances (matching srderr.ErrInvalidInput) for a negative time; the zero DeathCircumstances is valid and means a
// creature that died this instant, intact, not Undead, not of old age.
// Path: unhappy (invalid input).
func TestMustDeathCircumstances(t *testing.T) {
	t.Parallel()
	if got := combat.MustDeathCircumstances(time.Hour, false, false, combat.MissingNonIntegralParts); got.TimeSinceDeath() != time.Hour {
		t.Errorf("MustDeathCircumstances = %+v, want one hour", got)
	}
	zero := combat.DeathCircumstances{}
	if zero.TimeSinceDeath() != 0 || zero.DiedOfOldAge() || zero.WasUndead() || zero.Body() != combat.Intact {
		t.Errorf("zero DeathCircumstances = %+v, want died this instant, intact, not Undead, not old age", zero)
	}
	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an error matching ErrInvalidInput", err)
		}
	}()
	combat.MustDeathCircumstances(-1, false, false, combat.Intact)
}

// Requirement: doc/specs/revival-and-restoration-spells.md section 3 - the names of the body states and of the ineligibility reasons.
// Path: happy.
func TestBodyStateAndReasonNames(t *testing.T) {
	t.Parallel()
	for b, want := range map[combat.BodyState]string{combat.Intact: "intact", combat.MissingNonIntegralParts: "missing_non_integral_parts",
		combat.MissingIntegralParts: "missing_integral_parts", combat.NoBody: "no_body", 9: "unknown", -1: "unknown"} {
		if b.String() != want {
			t.Errorf("BodyState(%d).String() = %q, want %q", int(b), b.String(), want)
		}
	}
	for r, want := range map[combat.RevivalIneligibility]string{combat.TimeWindowExceeded: "time_window_exceeded", combat.DiedOfOldAge: "died_of_old_age",
		combat.WasUndead: "was_undead", combat.NoBodyToTouch: "no_body_to_touch", combat.IntegralPartsMissing: "integral_parts_missing",
		combat.NameNotSpoken: "name_not_spoken", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("RevivalIneligibility(%d).String() = %q, want %q", int(r), r.String(), want)
		}
	}
}
