package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func luSpell(tb testing.TB, g, s int) combat.SpellUseRef {
	tb.Helper()
	return luMust(combat.NewSpellUseRef(g, s))
}

// luCaster has the groups 1/Day Each: A, B; 2/Day: C, D; At Will: E, and a 2/Day action.
func luCaster(tb testing.TB) content.Monster {
	tb.Helper()
	return luMonster(tb, func(f *content.MonsterFields) {
		f.Traits = []content.StatBlockPart{luSpellcasting(
			luSpells(luPerDayEach(1), "A", "B"), luSpells(luPerDaySpells(2), "C", "D"), luSpells(content.SpellFrequencyAtWill{}, "E"))}
		f.Actions = []content.StatBlockPart{luPart("Mind", luPerDay(2, 0))}
	})
}

func luSpellSpent(tb testing.TB, s combat.LimitedUseState, r combat.SpellUseRef) combat.LimitedUseState {
	tb.Helper()
	out, ok := s.SpendSpell(r).(combat.LimitedUseSpent)
	if !ok {
		tb.Fatalf("SpendSpell(%v) was refused", r)
	}
	return out.Transition.State()
}

func luSpellLeft(tb testing.TB, s combat.LimitedUseState, r combat.SpellUseRef) int {
	tb.Helper()
	for _, e := range s.Entries() {
		if v, ok := e.(combat.SpellUseEntry); ok && v.Spell() == r {
			return v.Remaining()
		}
	}
	tb.Fatalf("no spell entry for %v", r)
	return -1
}

// Requirement: doc/specs/monster-limited-use.md FR-043, FR-044, SP-1 - a state holds one entry per spell of an N/Day or N/Day Each group, after the part entries and in printed order (group, then spell), each with n uses; At Will spells, a monster with no Spellcasting and a negative index have none (SRD 5.2.1, "Spellcasting", p. 257; "Priest", p. 316).
// Path: per case.
func TestSpellEntries(t *testing.T) {
	t.Parallel()
	single := luMonster(t, func(f *content.MonsterFields) {
		f.Traits = []content.StatBlockPart{luSpellcasting(luSpells(luPerDaySpells(2), "Single"))}
	})
	cases := []struct {
		name, requirement, path string
		monster                 content.Monster
		want                    []string
	}{
		{"per day each and per day, at will skipped", "FR-043 SP-1", "happy", luCaster(t),
			[]string{"perday actions/0 max=2 lair=0 left=2", "spell 0/0 max=1 left=1", "spell 0/1 max=1 left=1", "spell 1/0 max=2 left=2", "spell 1/1 max=2 left=2"}},
		{"per day single", "FR-043 SP-1", "happy", single, []string{"spell 0/0 max=2 left=2"}},
		{"no spellcasting", "FR-043 SP-1", "unhappy (empty and minimal)", luDrake(t),
			[]string{"perday traits/0 max=3 lair=4 left=3", "rest traits/1 ready", "perday actions/1 max=2 lair=0 left=2", "recharge actions/2 5-6 ready", "recharge actions/3 6-6 ready", "perday reactions/0 max=1 lair=0 left=1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := luDescribe(luStart(t, c.monster, combat.OutsideLair).Entries()); !slices.Equal(got, c.want) {
				t.Errorf("entries = %q, want %q [%s%s]", got, c.want, limitedUseSpec, c.requirement)
			}
		})
	}
	for _, bad := range [][2]int{{-1, 0}, {0, -1}, {-1, -1}} {
		if r, err := combat.NewSpellUseRef(bad[0], bad[1]); !errors.Is(err, srderr.ErrInvalidInput) || r != (combat.SpellUseRef{}) {
			t.Errorf("NewSpellUseRef(%d, %d) = %v, %v, want the zero ref and ErrInvalidInput [FR-044 SP-1]", bad[0], bad[1], r, err)
		}
	}
	if r := luSpell(t, 2, 3); r.GroupIndex() != 2 || r.SpellIndex() != 3 || r != luSpell(t, 2, 3) || r == luSpell(t, 3, 2) {
		t.Error("SpellUseRef must expose its indexes and compare by value [FR-044]")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-045, FR-046, FR-048, SP-1 - SpendSpell reduces one spell by exactly 1 with a LimitedUsePartSpent event naming a SpellSubject, refuses 0 uses as exhausted and an At Will spell or a position outside the stat block as not limited, and each spell has its own uses, never a pool shared by the group.
// Path: per case.
func TestSpendSpell(t *testing.T) {
	t.Parallel()
	s := luStart(t, luCaster(t), combat.OutsideLair)
	// 1/Day spell: once, then refused.
	a := luSpell(t, 0, 0)
	out := s.SpendSpell(a).(combat.LimitedUseSpent)
	e := out.Transition.Events()
	if len(e) != 1 || e[0].(combat.LimitedUsePartSpent).Subject() != (combat.SpellSubject{Spell: a}) || e[0].(combat.LimitedUsePartSpent).Remaining() != 0 {
		t.Errorf("events = %#v, want one LimitedUsePartSpent for spell A with 0 left [FR-045]", e)
	}
	s1 := out.Transition.State()
	if r, ok := s1.SpendSpell(a).(combat.LimitedUseSpendRefused); !ok || r.Reason != combat.LimitedUseRefusedExhausted {
		t.Errorf("second spend = %#v, want refused exhausted [FR-045 SP-1]", s1.SpendSpell(a))
	}
	if s1.AvailabilityOfSpell(a) != combat.UseExhausted || s.AvailabilityOfSpell(a) != combat.UseAvailable {
		t.Error("AvailabilityOfSpell must follow the uses [FR-048]")
	}
	// 2/Day spell C twice: C 0, D 2 (independent uses), and B of the same group is untouched.
	c, d := luSpell(t, 1, 0), luSpell(t, 1, 1)
	s2 := luSpellSpent(t, luSpellSpent(t, s, c), c)
	if luSpellLeft(t, s2, c) != 0 || luSpellLeft(t, s2, d) != 2 || luSpellLeft(t, s2, luSpell(t, 0, 1)) != 1 {
		t.Errorf("uses C, D, B = %d, %d, %d, want 0, 2, 1 [FR-046]", luSpellLeft(t, s2, c), luSpellLeft(t, s2, d), luSpellLeft(t, s2, luSpell(t, 0, 1)))
	}
	if r, ok := s2.SpendSpell(c).(combat.LimitedUseSpendRefused); !ok || r.Reason != combat.LimitedUseRefusedExhausted {
		t.Error("third cast of a 2/Day spell must be refused as exhausted [SP-1]")
	}
	// Not limited: At Will, a group past the list, a spell past its group, the zero ref (group 0 spell 0 is a real spell).
	for name, ref := range map[string]combat.SpellUseRef{"at will": luSpell(t, 2, 0), "group past the list": luSpell(t, 9, 0), "spell past the group": luSpell(t, 0, 5)} {
		if r, ok := s.SpendSpell(ref).(combat.LimitedUseSpendRefused); !ok || r.Reason != combat.LimitedUseRefusedNotLimited {
			t.Errorf("%s: SpendSpell = %#v, want refused not_limited_use [FR-045]", name, s.SpendSpell(ref))
		}
		if s.AvailabilityOfSpell(ref) != combat.UseNotLimited {
			t.Errorf("%s: AvailabilityOfSpell = %v, want not_limited [FR-048]", name, s.AvailabilityOfSpell(ref))
		}
	}
	// A part is not a spell: the two spend methods look in separate lists (FR-050).
	if _, ok := s.Spend(luRef(t, combat.MonsterPartTraits, 0)).(combat.LimitedUseSpendRefused); !ok {
		t.Error("Spend of the Spellcasting trait must be refused: it is not a limited part [FR-050]")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-047, FR-049, FR-027, SP-1 - a Long Rest restores every spell entry to its maximum and a Short Rest does not; one event lists a PartSubject then a SpellSubject; ChangeLair and StartOfOwnTurn leave spell entries unchanged (SRD 5.2.1, "Limited Usage", p. 257).
// Path: per case.
func TestSpellUsesAndRests(t *testing.T) {
	t.Parallel()
	spent := luSpellSpent(t, luSpent(t, luStart(t, luCaster(t), combat.OutsideLair), luRef(t, combat.MonsterPartActions, 0)), luSpell(t, 1, 0))
	a := luSpell(t, 1, 0)
	if short := spent.AfterRest(character.ShortRest); luSpellLeft(t, short.State(), a) != 1 || len(short.Events()) != 0 || !short.State().Equal(spent) {
		t.Error("a Short Rest must not restore a spell nor the X/Day part, and adds no event [FR-047]")
	}
	long := spent.AfterRest(character.LongRest)
	if luSpellLeft(t, long.State(), a) != 2 {
		t.Error("a Long Rest must restore the spell [FR-047]")
	}
	want := []combat.LimitedUseSubject{combat.PartSubject{Part: luRef(t, combat.MonsterPartActions, 0)}, combat.SpellSubject{Spell: a}}
	if ev := long.Events(); len(ev) != 1 || !slices.Equal(ev[0].(combat.LimitedUsesRestored).Subjects(), want) {
		t.Errorf("events = %#v, want one LimitedUsesRestored listing the part then the spell [FR-027]", ev)
	}
	if lair := spent.ChangeLair(combat.InLair); luSpellLeft(t, lair.State(), a) != 1 {
		t.Error("a lair change must leave spell entries unchanged [FR-049]")
	}
	turn, err := spent.StartOfOwnTurn(nil)
	if err != nil || luSpellLeft(t, turn.State(), a) != 1 || len(turn.Events()) != 0 {
		t.Errorf("the start of a turn must leave spell entries unchanged and add no event [FR-049]: %v", err)
	}
	for _, r := range spent.RechargeRollsNeeded() {
		t.Errorf("RechargeRollsNeeded lists %v: spell entries never need a roll [FR-050]", r)
	}
}
