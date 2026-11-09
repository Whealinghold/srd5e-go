package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-bard.md FR-060 to FR-062, FR-066, FR-070, FR-076, FR-092, FR-094 (SRD 5.2.1, "Bard Features", p. 31) - the choices each of the 20 levels owes: Expertise at 2 and 9, the subclass at 3 (with Bonus Proficiencies for the College of
// Lore), a feat at 4, 8, 12 and 16, a cantrip at 4 and 10, Magical Discoveries at 6 for a College of Lore Bard only, the Epic Boon feat at 19, and nothing at the other levels (5 owes no feat).
// Path: per case (the 20 levels): happy / unhappy (boundary).
func TestBardPendingChoicesAt(t *testing.T) {
	t.Parallel()
	exp, sub, bonus, feat, boon, cantrip, disc := character.BardPendingExpertise, character.BardPendingSubclass, character.BardPendingBonusProficiencies,
		character.BardPendingFeat, character.BardPendingEpicBoon, character.BardPendingCantrip, character.BardPendingMagicalDiscoveries
	type pc = []character.BardPendingChoice
	lorePending := map[int]pc{2: {exp}, 3: {sub, bonus}, 4: {feat, cantrip}, 6: {disc}, 8: {feat}, 9: {exp}, 10: {cantrip}, 12: {feat}, 16: {feat}, 19: {boon}}
	plainPending := map[int]pc{2: {exp}, 3: {sub}, 4: {feat, cantrip}, 8: {feat}, 9: {exp}, 10: {cantrip}, 12: {feat}, 16: {feat}, 19: {boon}}
	for n := 1; n <= 20; n++ {
		l := character.MustBardLevel(n)
		for _, c := range []struct {
			subclass content.SubclassID
			want     pc
		}{{lore, lorePending[n]}, {"", plainPending[n]}} {
			got := character.BardPendingChoicesAt(l, c.subclass)
			if got == nil || !slices.Equal(got, append(pc{}, c.want...)) {
				t.Errorf("level %d, subclass %q: %v, want %v", n, c.subclass, got, c.want)
			}
		}
	}
	for c, want := range map[character.BardPendingChoice]string{exp: "expertise", sub: "subclass", bonus: "bonus_proficiencies", feat: "feat", boon: "epic_boon", cantrip: "cantrip", disc: "magical_discoveries", 0: "unknown", 8: "unknown"} {
		if c.String() != want {
			t.Errorf("BardPendingChoice(%d) = %q, want %q", int(c), c.String(), want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-097, FR-099, FR-115, FR-121, FR-123 - AdvanceBard, called once the new level is recorded, gives a first-level Bard a full Bardic Inspiration pool, recomputes the maximum of a stored pool from the Charisma modifier
// at a later level (clamping, never raising), reports the choices, the Bardic Die, the prepared spells and cantrips the Features table adds (4 to 5 prepared at level 2; 1 cantrip at 4 and 10; 0 otherwise) and the always-prepared spells at level 20, and
// leaves the input Character alone.
// Path: per case (happy / unhappy: boundary).
func TestAdvanceBard(t *testing.T) {
	t.Parallel()
	flat := restored(t, 355000, 200, flatScores(t, 10), cl("bard", 1))
	first := must(character.AdvanceBard(flat, 0, "")).(character.AdvanceBardApplied)
	if p := brdPoolOf(t, first.Character()); p.Maximum() != 1 || p.Remaining() != 1 {
		t.Errorf("a first level seeded %d of %d, want 1 of 1", p.Remaining(), p.Maximum())
	}
	if _, held, _ := character.ReadFeatureState(flat.FeatureStates(), character.BardicInspirationPoolKey()); held {
		t.Error("AdvanceBard altered its argument")
	}
	if first.Level().Value() != 1 || first.BardicDie() != character.BardicD6 || first.PreparedSpellsGained() != 4 || first.CantripsGained() != 2 || len(first.PendingChoices()) != 0 || len(first.AlwaysPreparedSpellsGranted()) != 0 {
		t.Errorf("level 1 = %d, %v, +%d prepared, +%d cantrips, %v, %v", first.Level().Value(), first.BardicDie(), first.PreparedSpellsGained(), first.CantripsGained(), first.PendingChoices(), first.AlwaysPreparedSpellsGranted())
	}
	seeded := brdBard(t, 1, 16)
	if got := must(character.AdvanceBard(seeded, 0, "")).(character.AdvanceBardApplied); brdPoolOf(t, got.Character()) != brdPoolOf(t, seeded) {
		t.Error("a pool seeded at creation was replaced at the first level")
	}
	lvl2 := brdWithPool(t, restored(t, 355000, 200, flatScores(t, 10), cl("bard", 2)), 3, 3)
	got := must(character.AdvanceBard(lvl2, 1, "")).(character.AdvanceBardApplied)
	if p := brdPoolOf(t, got.Character()); p.Maximum() != 1 || p.Remaining() != 1 || got.PreparedSpellsGained() != 1 || got.CantripsGained() != 0 || !slices.Equal(got.PendingChoices(), []character.BardPendingChoice{character.BardPendingExpertise}) {
		t.Errorf("level 2 = pool %d of %d, +%d prepared, %v", p.Remaining(), p.Maximum(), got.PreparedSpellsGained(), got.PendingChoices())
	}
	if p := brdPoolOf(t, lvl2); p.Maximum() != 3 {
		t.Error("AdvanceBard altered its argument")
	}
	grown := brdWithPool(t, restored(t, 355000, 200, must(flatScores(t, 10).With(abilityCharisma, scoreOf(20))), cl("bard", 5)), 1, 0)
	if p := brdPoolOf(t, must(character.AdvanceBard(grown, 4, "")).(character.AdvanceBardApplied).Character()); p.Maximum() != 5 || p.Remaining() != 0 {
		t.Errorf("a grown maximum = %d of %d, want 0 of 5 (never raised)", p.Remaining(), p.Maximum())
	}
	gains := map[int][2]int{4: {1, 1}, 5: {2, 0}, 10: {1, 1}, 18: {1, 0}, 20: {1, 0}, 11: {1, 0}, 12: {0, 0}}
	for n, want := range gains {
		c := brdBard(t, n, 14)
		a := must(character.AdvanceBard(c, n-1, "")).(character.AdvanceBardApplied)
		if a.PreparedSpellsGained() != want[0] || a.CantripsGained() != want[1] {
			t.Errorf("level %d adds %d prepared and %d cantrips, want %v", n, a.PreparedSpellsGained(), a.CantripsGained(), want)
		}
	}
	twenty := must(character.AdvanceBard(brdBard(t, 20, 14), 19, "")).(character.AdvanceBardApplied)
	if !slices.Equal(twenty.AlwaysPreparedSpellsGranted(), []content.SpellID{"power_word_heal", "power_word_kill"}) || twenty.BardicDie() != character.BardicD12 {
		t.Errorf("level 20 = %v, %v", twenty.AlwaysPreparedSpellsGranted(), twenty.BardicDie())
	}
}

// Requirement: doc/specs/class-bard.md FR-098, FR-099, FR-115, FR-118 - AdvanceBard returns NotABard for a character with no Bard level, and an error matching ErrInvalidInput for a zero Character, a previous level that is not one below the new level
// (no multi-level jump), a Bard above level 1 that holds no pool (a broken invariant, never zero uses) and another type of state under the pool key.
// Path: per case (unhappy: state error, invalid input).
func TestAdvanceBardRefusals(t *testing.T) {
	t.Parallel()
	if got, err := character.AdvanceBard(clcCleric(t, 3), 2, ""); err != nil || got != (character.AdvanceBardNotABard{}) {
		t.Errorf("a Cleric = %v, %v, want NotABard", got, err)
	}
	foreign := must(character.WithFeatureState(restored(t, 355000, 200, flatScores(t, 10), cl("bard", 2)), character.MustFeatureStateKey[character.ChosenSubclass]("bardic_inspiration_pool"), must(character.NewChosenSubclass("college_of_lore"))))
	noPool := restored(t, 355000, 200, flatScores(t, 10), cl("bard", 2))
	for name, run := range map[string]func() error{
		"zero Character":      func() error { _, e := character.AdvanceBard(character.Character{}, 0, ""); return e },
		"a jump of two":       func() error { _, e := character.AdvanceBard(brdBard(t, 4, 14), 2, ""); return e },
		"the same level":      func() error { _, e := character.AdvanceBard(brdBard(t, 4, 14), 4, ""); return e },
		"no pool at level 2":  func() error { _, e := character.AdvanceBard(noPool, 1, ""); return e },
		"another state type":  func() error { _, e := character.AdvanceBard(foreign, 1, ""); return e },
		"another type at one": func() error { _, e := character.AdvanceBard(foreign, 0, ""); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
