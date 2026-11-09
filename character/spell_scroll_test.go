package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: spell-scroll-use.md FR-023, FR-024 (SRD 5.2.1, "Scribing Spell Scrolls", p. 103) - the stats take the
// DC, attack bonus and level of the casting class as they were at completion.
// Path: happy.
func TestScribedScrollStatsKeepTheScribesNumbers(t *testing.T) {
	t.Parallel()
	s := scribedStats(t, 11)
	if s.SaveDC() != 15 || s.AttackBonus() != 7 || s.CharacterLevel() != character.MustLevel(11) {
		t.Errorf("stats = DC %d %+d level %d, want 15 +7 level 11", s.SaveDC(), s.AttackBonus(), s.CharacterLevel().Value())
	}
}

// Requirement: spell-scroll-use.md FR-024, B3 - a later level-up of the scribe does not change a scroll already completed.
// Path: unhappy (rule interaction).
func TestScribedScrollStatsAreFixedAtCompletion(t *testing.T) {
	t.Parallel()
	scroll := scrollOf(t, 1, scribedOrigin(t, 11), "wizard")
	later := scribedStats(t, 13) // the scribe's numbers have grown to DC 16
	if later.SaveDC() != 16 {
		t.Fatalf("fixture: level 13 DC = %d, want 16", later.SaveDC())
	}
	got := scroll.Origin().(character.ScribedScrollOrigin).Stats()
	if got.SaveDC() != 15 || got.CharacterLevel().Value() != 11 {
		t.Errorf("scroll stats = DC %d level %d, want 15 and 11", got.SaveDC(), got.CharacterLevel().Value())
	}
}

// Requirement: spell-scroll-use.md FR-024, FR-023 - zero stats, level and origins are errors, never panics.
// Path: unhappy (invalid input).
func TestScribedScrollConstructorsRejectZeroValues(t *testing.T) {
	t.Parallel()
	l := character.MustLevel(5)
	ok := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(l))
	cases := []struct {
		name string
		err  error
	}{
		{"zero stats", func() error { _, err := character.NewScribedScrollStats(character.SpellcastingStats{}, l); return err }()},
		{"zero level", func() error { _, err := character.NewScribedScrollStats(ok, character.Level{}); return err }()},
		{"zero stats in an origin", func() error { _, err := character.NewScribedScrollOrigin(character.ScribedScrollStats{}); return err }()},
	}
	for _, c := range cases {
		if !errors.Is(c.err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", c.name, c.err)
		}
	}
}

// Requirement: spell-scroll-use.md FR-023 - NewSpellScroll rejects a nil origin and a zero scribed origin.
// Path: unhappy (invalid input).
func TestNewSpellScrollRejectsInvalidOrigins(t *testing.T) {
	t.Parallel()
	sp := spellMap(t, spellSpec{id: "x_spell", level: 1, classes: []string{"wizard"}})[content.MustSpellID("x_spell")]
	for name, origin := range map[string]character.SpellScrollOrigin{"nil": nil, "zero scribed": character.ScribedScrollOrigin{}} {
		if _, err := character.NewSpellScroll(sp, origin); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s origin: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: spell-scroll-use.md FR-034 - Classes returns a copy that cannot corrupt the scroll.
// Path: unhappy (immutability).
func TestSpellScrollClassesAreCopied(t *testing.T) {
	t.Parallel()
	s := scrollOf(t, 3, character.FoundScroll{}, "wizard", "cleric")
	if s.SpellID() != content.MustSpellID("scroll_spell") || s.SpellLevel() != 3 {
		t.Errorf("scroll = %v level %d", s.SpellID(), s.SpellLevel())
	}
	s.Classes()[0] = content.MustClassID("bard")
	if got := s.Classes(); len(got) != 2 || got[0] == content.MustClassID("bard") {
		t.Errorf("Classes() = %v after mutating a copy", got)
	}
}

// Requirement: spell-scroll-use.md FR-014 - the caster level variants report their level.
// Path: happy.
func TestScrollCasterLevels(t *testing.T) {
	t.Parallel()
	if got := (character.ScrollLowestCasterLevel{}).Level(); got != character.ItemCasterLevelLowest {
		t.Errorf("lowest = %v", got)
	}
	if got := character.NewScribedCantripCasterLevel(character.MustLevel(11)).Level(); got.Value() != 11 {
		t.Errorf("scribed cantrip level = %d, want 11", got.Value())
	}
}
