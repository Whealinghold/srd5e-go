package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/rests-and-hit-dice.md FR-001 - RestKind has exactly two
// valid values and the zero value is invalid; the telemetry spelling is short or
// long (SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185).
// Path: unhappy (boundary: zero and one past the last value).
func TestRestKindHasExactlyTwoValues(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		kind  character.RestKind
		valid bool
		name  string
	}{
		{0, false, "unknown"}, {character.ShortRest, true, "short"}, {character.LongRest, true, "long"}, {3, false, "unknown"},
	} {
		if c.kind.Valid() != c.valid || c.kind.String() != c.name {
			t.Errorf("RestKind(%d) = valid %v %q, want %v %q", int(c.kind), c.kind.Valid(), c.kind.String(), c.valid, c.name)
		}
	}
	if character.StandardLongRest.Valid() != true || character.TranceLongRest.Valid() != true || character.LongRestVariant(0).Valid() || character.LongRestVariant(3).Valid() {
		t.Error("LongRestVariant validity is wrong")
	}
	if !character.RestFinished.Valid() || !character.LongRestInterrupted.Valid() || character.RestSource(0).Valid() || character.RestSource(3).Valid() {
		t.Error("RestSource validity is wrong")
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-002, FR-003, FR-004 - the
// minimum hours are 1 for a Short Rest (whatever the variant), 8 for a standard
// Long Rest and 4 with Trance (SRD 5.2.1, "Short Rest", p. 187; "Long Rest",
// p. 185; "Elf", Trance, p. 85).
// Path: per case.
func TestRestKindMinimumHours(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		variant                 character.LongRestVariant
		want                    int
	}{
		{"short rest", "FR-002", "happy", character.ShortRest, character.StandardLongRest, 1},
		{"short rest ignores trance", "FR-004", "unhappy (rule interaction)", character.ShortRest, character.TranceLongRest, 1},
		{"standard long rest", "FR-003", "happy", character.LongRest, character.StandardLongRest, 8},
		{"trance long rest", "FR-004", "happy", character.LongRest, character.TranceLongRest, 4},
		{"invalid variant", "FR-003", "unhappy (invalid input)", character.LongRest, 0, 0},
		{"invalid kind", "FR-001", "unhappy (invalid input)", 0, character.StandardLongRest, 0},
		{"invalid kind with trance", "FR-001", "unhappy (invalid input)", 0, character.TranceLongRest, 0},
		{"undefined kind past the last", "FR-001", "unhappy (boundary)", 3, character.StandardLongRest, 0},
		{"undefined kind past the last with trance", "FR-001", "unhappy (boundary)", 3, character.TranceLongRest, 0},
		{"undefined variant past the last", "FR-003", "unhappy (boundary)", character.LongRest, 3, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.kind.MinimumHours(c.variant); got != c.want {
				t.Errorf("MinimumHours = %d, want %d [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-005, FR-006, FR-022, table 4.8a
// SI-4 - there are exactly four interruptions; each stops a Long Rest, and each
// stops a Short Rest (PhysicalExertion by inference); an undefined one stops
// nothing (SRD 5.2.1, "Long Rest", p. 185; "Short Rest", p. 187).
// Path: per case.
func TestRestInterruptionStops(t *testing.T) {
	t.Parallel()
	for _, i := range []character.RestInterruption{character.RollingInitiative, character.CastingSpellOtherThanCantrip, character.TakingDamage, character.PhysicalExertion} {
		for _, k := range []character.RestKind{character.ShortRest, character.LongRest} {
			if !i.Valid() || !i.Stops(k) || i.String() == "unknown" {
				t.Errorf("%v does not stop %v", i, k)
			}
		}
	}
	for _, i := range []character.RestInterruption{0, 5} {
		if i.Valid() || i.Stops(character.LongRest) || i.String() != "unknown" {
			t.Errorf("RestInterruption(%d) is valid or stops a rest", int(i))
		}
	}
	for i, want := range map[character.RestInterruption]string{
		character.RollingInitiative: "rolling_initiative", character.CastingSpellOtherThanCantrip: "casting_spell_other_than_cantrip",
		character.TakingDamage: "taking_damage", character.PhysicalExertion: "physical_exertion",
	} {
		if i.String() != want {
			t.Errorf("RestInterruption(%d).String() = %q, want %q", int(i), i.String(), want)
		}
	}
	if character.TakingDamage.Stops(0) {
		t.Error("an interruption stops an invalid rest kind")
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-032, FR-065, table 4.8f - one
// test per cell: a Long Rest is not also a Short Rest, and either recharges a
// "Short or Long Rest" feature (SRD 5.2.1, "Short Rest", "Special Feature",
// p. 187; "Long Rest", "Special Feature", p. 185).
// Path: per case.
func TestRestKindRecharges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		kind                    character.RestKind
		trigger                 content.RechargeTrigger
		want                    bool
	}{
		{"RW-1 short rest recharges short trigger", "RW-1", "happy", character.ShortRest, content.RechargeShortRest, true},
		{"RW-1 long rest does not recharge short trigger", "RW-1", "unhappy (rule interaction: a Long Rest is not also a Short Rest)", character.LongRest, content.RechargeShortRest, false},
		{"RW-2 short rest does not recharge long trigger", "RW-2", "unhappy (rule interaction)", character.ShortRest, content.RechargeLongRest, false},
		{"RW-2 long rest recharges long trigger", "RW-2", "happy", character.LongRest, content.RechargeLongRest, true},
		{"RW-3 short rest recharges either trigger", "RW-3", "happy", character.ShortRest, content.RechargeShortOrLongRest, true},
		{"RW-3 long rest recharges either trigger", "RW-3", "happy", character.LongRest, content.RechargeShortOrLongRest, true},
		{"invalid trigger", "FR-065", "unhappy (invalid input)", character.LongRest, 0, false},
		{"invalid kind", "FR-065", "unhappy (invalid input)", 0, content.RechargeShortOrLongRest, false},
		{"invalid kind against a short trigger", "FR-065", "unhappy (invalid input)", 0, content.RechargeShortRest, false},
		{"undefined kind against a long trigger", "FR-065", "unhappy (boundary)", 3, content.RechargeLongRest, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.kind.Recharges(c.trigger); got != c.want {
				t.Errorf("%v.Recharges(%v) = %v, want %v [%s]", c.kind, c.trigger, got, c.want, c.requirement)
			}
		})
	}
}
