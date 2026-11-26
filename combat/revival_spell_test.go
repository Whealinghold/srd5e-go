package combat_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestRevivalSpellOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		id                      string
		want                    combat.RevivalSpell
		ok                      bool
	}{
		{"revivify", rrSpec + `FR-001; SRD 5.2.1, "Revivify", p. 159`, "happy", "revivify", combat.Revivify, true},
		{"raise dead", rrSpec + `FR-001; SRD 5.2.1, "Raise Dead", p. 157`, "happy", "raise_dead", combat.RaiseDead, true},
		{"resurrection", rrSpec + `FR-001; SRD 5.2.1, "Resurrection", pp. 158-159`, "happy", "resurrection", combat.Resurrection, true},
		{"true resurrection", rrSpec + `FR-001; SRD 5.2.1, "True Resurrection", p. 170`, "happy", "true_resurrection", combat.TrueResurrection, true},
		{"reincarnate is not a revival", rrSpec + "FR-001", "unhappy (invalid input)", "reincarnate", 0, false},
		{"revive is not a spell", rrSpec + "FR-001", "unhappy (invalid input)", "revive", 0, false},
		{"unknown id", rrSpec + "FR-001", "unhappy (invalid input)", "no_such_spell", 0, false},
		{"empty id", rrSpec + "FR-001", "unhappy (empty and minimal)", "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := combat.RevivalSpellOf(content.SpellID(c.id))
			if got != c.want || ok != c.ok {
				t.Errorf("RevivalSpellOf(%q) = %v, %v, want %v, %v [%s; %s]", c.id, got, ok, c.want, c.ok, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md table 4.1a - the static facts of each revival spell: the window
// (1 minute, 10 days, 36,500 days, 73,000 days), who it cannot return, the Hit Points, the poison, the curses, the ordeal and
// the strain threshold (SRD 5.2.1, pp. 157-159, 170).
// Path: happy.
func TestRevivalSpellFacts(t *testing.T) {
	t.Parallel()
	type row struct {
		spell                                                       combat.RevivalSpell
		window                                                      time.Duration
		oldAge, undead, all, poison, curses, ordeal, integral, name bool
		strain                                                      time.Duration
	}
	for _, r := range []row{
		{combat.Revivify, time.Minute, true, false, false, false, false, false, false, false, 0},
		{combat.RaiseDead, 10 * rrDay, false, true, false, true, false, true, true, false, 0},
		{combat.Resurrection, 36500 * rrDay, true, true, true, true, false, true, false, false, 365 * rrDay},
		{combat.TrueResurrection, 73000 * rrDay, true, false, true, true, true, false, false, true, 0},
	} {
		f := r.spell.Facts()
		got := row{r.spell, f.Window(), f.OldAgeBlocks(), f.UndeadBlocks(), f.ReturnsAllHitPoints(), f.NeutralisesPoison(), f.LiftsCurses(),
			f.GivesOrdeal(), f.IntegralPartsBlock(), f.NameForBody(), f.StrainAfter()}
		if got != r {
			t.Errorf("%v.Facts() = %+v, want %+v", r.spell, got, r)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md section 3 - the zero and an out-of-range RevivalSpell are not revival spells,
// have the zero facts and the name "unknown"; the four values have their snake_case names.
// Path: unhappy (invalid input).
func TestRevivalSpellInvalid(t *testing.T) {
	t.Parallel()
	for _, s := range []combat.RevivalSpell{0, -1, 5} {
		if s.Valid() || s.Facts() != (combat.RevivalSpellFacts{}) || s.String() != "unknown" {
			t.Errorf("RevivalSpell(%d): Valid %v, Facts %+v, String %q, want invalid, zero facts, unknown", int(s), s.Valid(), s.Facts(), s.String())
		}
	}
	for s, want := range map[combat.RevivalSpell]string{combat.Revivify: "revivify", combat.RaiseDead: "raise_dead",
		combat.Resurrection: "resurrection", combat.TrueResurrection: "true_resurrection"} {
		if s.String() != want || !s.Valid() {
			t.Errorf("String() = %q (Valid %v), want %q", s.String(), s.Valid(), want)
		}
	}
}
