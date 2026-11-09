package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/derived-stats.md FR-049 - the saving throw set holds
// abilities, deduplicated, in SRD order, and unions without altering either side
// (SRD 5.2.1, "Saving Throw Proficiencies", p. 9).
// Path: unhappy (rule interaction).
func TestSavingThrowProficiencies(t *testing.T) {
	t.Parallel()
	a := must(character.NewSavingThrowProficiencies(ability.Wisdom, ability.Strength, ability.Wisdom))
	b := must(character.NewSavingThrowProficiencies(ability.Charisma))
	u := a.Union(b)
	if !a.Has(ability.Wisdom) || a.Has(ability.Charisma) || !u.Has(ability.Charisma) || u.Has(ability.Dexterity) {
		t.Errorf("Has/Union wrong: %v %v", a.Abilities(), u.Abilities())
	}
	if want := []ability.Ability{ability.Strength, ability.Wisdom, ability.Charisma}; !slices.Equal(u.Abilities(), want) {
		t.Errorf("Abilities = %v, want %v", u.Abilities(), want)
	}
	if got := (character.SavingThrowProficiencies{}).Abilities(); len(got) != 0 {
		t.Errorf("empty set = %v", got)
	}
	if a.Has(0) {
		t.Error("invalid ability reported as held")
	}
	_, err := character.NewSavingThrowProficiencies(ability.Strength, 0)
	wantInvalid(t, err, "invalid ability")
}
