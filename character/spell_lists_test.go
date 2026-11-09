package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spellcasting.md FR-097, FR-024 - NewGrantedCantrip rejects an
// invalid spell, class or list with ErrInvalidInput; Spellbook drops repeats.
// Path: unhappy (invalid input).
func TestGrantedCantripAndSpellbook(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		spell content.SpellID
		class character.SpellcastingClass
		list  content.ClassID
	}{"spell": {"", character.CasterWizard, "wizard"}, "class": {"a", 0, "wizard"}, "list": {"a", character.CasterWizard, ""}} {
		if _, err := character.NewGrantedCantrip(c.spell, c.class, c.list); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	g := mustGranted(t, "a", character.CasterWizard, "cleric")
	if g.Spell() != "a" || g.Class() != character.CasterWizard || g.ListClass() != "cleric" {
		t.Errorf("accessors = %v", g)
	}
	b := character.NewSpellbook([]content.SpellID{"a", "b", "a"})
	if len(b.Spells()) != 2 || !b.Contains("b") || b.Contains("c") {
		t.Errorf("spellbook = %v", b.Spells())
	}
	if (character.Spellbook{}).Contains("a") {
		t.Error("zero Spellbook contains a spell")
	}
}
