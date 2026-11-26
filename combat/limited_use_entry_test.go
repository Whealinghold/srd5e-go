package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-limited-use.md FR-002, FR-032 - an X/Day entry reports its printed numbers and the in-lair figure only when printed; entries are comparable values and differ when any field differs.
// Path: happy.
func TestEntryAccessors(t *testing.T) {
	t.Parallel()
	s := luStart(t, luDrake(t), combat.InLair)
	res := luPerDayOf(t, s, luRes(t))
	if lair, ok := res.MaximumInLair(); res.Maximum() != 3 || !ok || lair != 4 || res.Remaining() != 4 || res.Part() != luRes(t) {
		t.Errorf("Resistance entry = %d, %d (%v), left %d, want 3, 4 (true), left 4 [FR-002]", res.Maximum(), lair, ok, res.Remaining())
	}
	mind := luPerDayOf(t, s, luMind(t))
	if lair, ok := mind.MaximumInLair(); ok || lair != 0 {
		t.Errorf("Mind in-lair figure = %d, %v, want none", lair, ok)
	}
	if res == mind || res != luPerDayOf(t, luStart(t, luDrake(t), combat.InLair), luRes(t)) {
		t.Error("entries must compare by value [FR-032]")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-043, FR-032 - a spell entry reports its spell, maximum and remaining uses and compares by value.
// Path: happy.
func TestSpellEntryAccessors(t *testing.T) {
	t.Parallel()
	s := luStart(t, luMonster(t, func(f *content.MonsterFields) {
		f.Traits = []content.StatBlockPart{luSpellcasting(luSpells(luPerDayEach(3), "A"))}
	}), combat.OutsideLair)
	e := s.Entries()[0].(combat.SpellUseEntry)
	if e.Spell() != luSpell(t, 0, 0) || e.Maximum() != 3 || e.Remaining() != 3 {
		t.Errorf("entry = %v max %d left %d, want spell 0/0, 3 and 3", e.Spell(), e.Maximum(), e.Remaining())
	}
}
