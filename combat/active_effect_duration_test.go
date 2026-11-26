package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/active-effects.md FR-005 - EffectLastsRounds takes 1 or more rounds, a turn boundary duration needs an owner and a boundary.
// Path: unhappy (boundary).
func TestEffectDurationConstructors(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		if _, err := combat.NewEffectLastsRounds(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewEffectLastsRounds(%d) err = %v", n, err)
		}
	}
	if d, err := combat.NewEffectLastsRounds(1); err != nil || d.Rounds() != 1 {
		t.Errorf("NewEffectLastsRounds(1) = %v, %v", d, err)
	}
	if _, err := combat.NewEffectLastsUntilTurnBoundary("", content.BoundaryEnd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero owner err = %v", err)
	}
	if _, err := combat.NewEffectLastsUntilTurnBoundary("a", 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero boundary err = %v", err)
	}
	d, err := combat.NewEffectLastsUntilTurnBoundary("a", content.BoundaryStart)
	if err != nil || !d.OwnerIs("a") || d.OwnerIs("b") || d.Boundary() != content.BoundaryStart {
		t.Errorf("until = %v, %v", d, err)
	}
}

// Requirement: doc/specs/active-effects.md FR-006 (SRD 5.2.1, "The Order of Combat", p. 13; "Duration", p. 106) - 1 round per round, 10 per minute, 600 per hour, 14,400 per day; an amount of 0 or less, an unknown unit or a result that overflows is an error.
// Path: per case.
func TestRoundsInTimeSpan(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		amount     int
		unit       content.DurationUnit
		want       int
		ok         bool
	}{
		{"one round", "happy", 1, content.DurationUnitRounds, 1, true},
		{"one minute", "happy", 1, content.DurationUnitMinutes, 10, true},
		{"eight hours", "happy", 8, content.DurationUnitHours, 4800, true},
		{"one day", "happy", 1, content.DurationUnitDays, 14400, true},
		{"zero", "unhappy (boundary)", 0, content.DurationUnitMinutes, 0, false},
		{"negative", "unhappy (boundary)", -3, content.DurationUnitRounds, 0, false},
		{"unknown unit", "unhappy (invalid input)", 1, 0, 0, false},
		{"largest day amount", "unhappy (boundary)", math.MaxInt / 14400, content.DurationUnitDays, math.MaxInt / 14400 * 14400, true},
		{"one past the largest day amount", "unhappy (boundary)", math.MaxInt/14400 + 1, content.DurationUnitDays, 0, false},
	}
	for _, c := range cases {
		got, err := combat.RoundsInTimeSpan(c.amount, c.unit)
		if (err == nil) != c.ok || got != c.want || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("%s: RoundsInTimeSpan(%d, %v) = %d, %v, want %d ok=%v [FR-006]", c.name, c.amount, c.unit, got, err, c.want, c.ok)
		}
	}
}

func timed(amount int, unit content.DurationUnit, limit content.DurationLimit) content.SpellDuration {
	return aeMust(content.NewSpellDurationTimed(amount, unit, limit))
}

// Requirement: doc/specs/active-effects.md FR-010, table D rows D1 D2 D5 D6 D7 - a spell duration maps to rounds or to indefinite; Instantaneous and an unknown duration are errors.
// Path: per case.
func TestEffectDurationFromSpell(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		in         content.SpellDuration
		want       combat.EffectDuration
	}{
		{"D1 Hold Person one minute", "happy", timed(1, content.DurationUnitMinutes, content.LimitConcentrationUpTo), aeRounds(t, 10)},
		{"D2 Hex eight hours", "happy", timed(8, content.DurationUnitHours, content.LimitConcentrationUpTo), aeRounds(t, 4800)},
		{"D5 one round fixed", "unhappy (boundary)", timed(1, content.DurationUnitRounds, content.LimitFixed), aeRounds(t, 1)},
		{"D6 until dispelled", "happy", content.NewSpellDurationUntilDispelled(), combat.EffectLastsIndefinitely{}},
		{"until dispelled or triggered", "happy", content.NewSpellDurationUntilDispelledOrTriggered(), combat.EffectLastsIndefinitely{}},
		{"special", "happy", content.NewSpellDurationSpecial(), combat.EffectLastsIndefinitely{}},
		{"D7 instantaneous", "unhappy (invalid input)", content.NewSpellDurationInstantaneous(), nil},
		{"nil duration", "unhappy (invalid input)", nil, nil},
		{"overflowing span", "unhappy (boundary)", timed(math.MaxInt, content.DurationUnitDays, content.LimitFixed), nil},
	}
	for _, c := range cases {
		got, err := combat.EffectDurationFromSpell(c.in)
		if c.want == nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
				t.Errorf("%s: = %v, %v, want an ErrInvalidInput error [FR-010]", c.name, got, err)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s: = %v, %v, want %v [FR-010]", c.name, got, err, c.want)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-011, table D rows D3 D4 - a condition duration maps for the spell, for a time span and for a turn boundary of the caster or the target.
// Path: per case.
func TestEffectDurationFromCondition(t *testing.T) {
	t.Parallel()
	minute := timed(1, content.DurationUnitMinutes, content.LimitConcentrationUpTo)
	hours := aeMust(content.NewTimedConditionDuration(2, content.DurationUnitHours))
	casterEnd := aeMust(content.NewUntilNextTurnBoundary(content.OwnerCaster, content.BoundaryEnd))
	targetStart := aeMust(content.NewUntilNextTurnBoundary(content.OwnerTarget, content.BoundaryStart))
	cases := []struct {
		name string
		in   content.ConditionDuration
		want combat.EffectDuration
		ok   bool
	}{
		{"D3 for the spell duration", content.ForSpellDuration{}, aeRounds(t, 10), true},
		{"timed", hours, aeRounds(t, 1200), true},
		{"D4 until the end of the caster's turn", casterEnd, aeUntil(t, "caster", content.BoundaryEnd), true},
		{"until the start of the target's turn", targetStart, aeUntil(t, "target", content.BoundaryStart), true},
		{"nil", nil, nil, false},
	}
	for _, c := range cases {
		got, err := combat.EffectDurationFromCondition(c.in, minute, "caster", "target")
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("%s: = %v, %v, want %v ok=%v [FR-011]", c.name, got, err, c.want, c.ok)
		}
	}
	if _, err := combat.EffectDurationFromCondition(content.ForSpellDuration{}, content.NewSpellDurationInstantaneous(), "c", "t"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("for-the-spell on an instantaneous spell err = %v", err)
	}
	if _, err := combat.EffectDurationFromCondition(casterEnd, minute, "", "t"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero caster owner err = %v", err)
	}
	overflow := aeMust(content.NewTimedConditionDuration(math.MaxInt, content.DurationUnitDays))
	if _, err := combat.EffectDurationFromCondition(overflow, minute, "c", "t"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("overflowing condition span err = %v", err)
	}
}
