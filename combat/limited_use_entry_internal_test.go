package combat

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-limited-use.md FR-001, plan-phase-4.md P13 - content numbers are validated again before combat relies on them: a use count, a Recharge bound or a part with no usage that content's constructors would reject is invalid input, not a state. The public constructors of content cannot build such a value, so this test forges the zero values content leaves invalid.
// Path: unhappy (invalid input, missing content).
func TestUsageEntryRevalidatesContentNumbers(t *testing.T) {
	t.Parallel()
	ref := LimitedUsePartRef{MonsterPartActions, 0}
	cases := []struct {
		name  string
		usage content.PartUsage
	}{
		{"zero X/Day (count 0)", content.PartPerDay{}},
		{"zero Recharge (bounds 0-0)", content.PartRecharge{}},
		{"nil usage", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, limited, err := usageEntry(ref, c.usage, OutsideLair)
			if !errors.Is(err, srderr.ErrInvalidInput) || limited || e != nil {
				t.Errorf("usageEntry = %v, %v, %v, want no entry and ErrInvalidInput", e, limited, err)
			}
		})
	}
	for _, ok := range []content.PartUsage{content.PartUnlimited{}, content.PartUsageOther{}} {
		if e, limited, err := usageEntry(ref, ok, OutsideLair); e != nil || limited || err != nil {
			t.Errorf("usageEntry(%T) = %v, %v, %v, want no entry and no error", ok, e, limited, err)
		}
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-043, plan-phase-4.md P13 - the uses of a spell frequency and a use count are checked against content.MonsterMinUses and content.MonsterMaxNumber, both ends inclusive.
// Path: unhappy (boundary).
func TestUsesInRangeAndSpellUses(t *testing.T) {
	t.Parallel()
	over := content.MonsterMaxNumber
	over++ // wraps negative at 32-bit width, which is out of range as well
	for n, want := range map[int]bool{content.MonsterMinUses - 1: false, content.MonsterMinUses: true, content.MonsterMaxNumber: true, over: false} {
		if usesInRange(n) != want {
			t.Errorf("usesInRange(%d) = %v, want %v", n, !want, want)
		}
	}
	if n, limited := spellUses(content.SpellFrequencyPerDay{}); n != 0 || !limited {
		t.Errorf("zero N/Day = %d, %v, want 0 and limited (so Start rejects it)", n, limited)
	}
	if _, limited := spellUses(content.SpellFrequencyAtWill{}); limited {
		t.Error("At Will must not be limited")
	}
	if _, limited := spellUses(nil); limited {
		t.Error("a nil frequency must not be limited")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-004, FR-043, plan-phase-4.md P13 - a spell frequency or a Legendary Action use count outside the allowed range is invalid input, and an At Will group gives no entry; the in-range cases give one entry per spell, or the full track for the lair status.
// Path: per case.
func TestSpellEntriesAndLegendaryTrackRevalidate(t *testing.T) {
	t.Parallel()
	if _, err := spellEntries(0, content.SpellFrequencyPerDay{}, 2); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero N/Day: %v, want ErrInvalidInput", err)
	}
	if _, err := spellEntries(0, content.SpellFrequencyPerDayEach{}, 2); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero N/Day Each: %v, want ErrInvalidInput", err)
	}
	if e, err := spellEntries(0, content.SpellFrequencyAtWill{}, 2); e != nil || err != nil {
		t.Errorf("At Will: %v, %v, want no entry", e, err)
	}
	for _, c := range [][3]int{{0, 0, 1}, {-1, 0, 1}, {3, -1, 1}, {3, 0, 1}} {
		got, err := legendaryTrack(c[0], c[1], OutsideLair)
		if ok := c[0] >= 1 && c[1] >= 0; ok != (err == nil) || (!ok && !errors.Is(err, srderr.ErrInvalidInput)) || (!ok && got != nil) {
			t.Errorf("legendaryTrack(%d, %d) = %v, %v", c[0], c[1], got, err)
		}
	}
	over := content.MonsterMaxNumber
	over++ // wraps negative at 32-bit width, which is out of range as well
	for _, c := range [][2]int{{over, 0}, {3, over}} {
		if got, err := legendaryTrack(c[0], c[1], OutsideLair); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
			t.Errorf("legendaryTrack(%d, %d) = %v, %v, want ErrInvalidInput", c[0], c[1], got, err)
		}
	}
	if got, _ := legendaryTrack(3, 4, InLair); got != (LegendaryTracked{4, 4, 3, 4}) {
		t.Errorf("legendaryTrack(3, 4, InLair) = %v, want 4 of 4", got)
	}
}
