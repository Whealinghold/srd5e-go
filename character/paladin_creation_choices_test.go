package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-paladin.md FR-122 - PaladinCreationChoices seeds the level 1 PaladinState (pool 5, no Channel Divinity uses, every use available) under PaladinStateKey and the Weapon Mastery choice
// under WeaponMasteryChoiceKey when it is not empty, keeps every other entry, leaves its argument unchanged, names no content, and returns states already holding the Paladin key unchanged.
// Path: happy / unhappy (empty case, state error).
func TestPaladinCreationChoicesSeed(t *testing.T) {
	t.Parallel()
	empty := character.FeatureStates{}
	other := must(character.SetFeatureState(empty, character.ClericChoicesKey(), must(character.NewClericChoices(character.DivineOrderProtector, "", 0))))
	for _, masteries := range []character.WeaponMasteryChoice{{}, choiceOf(t, "longsword", "dagger")} {
		choices := character.NewPaladinCreationChoices(masteries)
		seeded := choices.SeedFeatureStates(other)
		s, ok, err := character.ReadFeatureState(seeded, character.PaladinStateKey())
		if err != nil || !ok || !s.Equal(character.InitialPaladinState(character.MustPaladinLevel(1), 0)) || s.LayOnHandsRemaining() != 5 {
			t.Fatalf("seeded state = %+v, %v, %v", s, ok, err)
		}
		if _, ok, _ := character.ReadFeatureState(seeded, character.ClericChoicesKey()); !ok {
			t.Error("the other entry was dropped")
		}
		if _, ok, _ := character.ReadFeatureState(other, character.PaladinStateKey()); ok {
			t.Error("the argument changed")
		}
		m, held, err := character.ReadFeatureState(seeded, character.WeaponMasteryChoiceKey())
		if err != nil || held != (len(masteries.Kinds()) > 0) || (held && !m.Choice().Equal(masteries)) {
			t.Errorf("mastery state = %v, %v, %v", m, held, err)
		}
		if got := choices.ContentKeys(); got == nil || len(got) != 0 {
			t.Errorf("ContentKeys = %v, want an empty slice", got)
		}
		twice := choices.SeedFeatureStates(seeded)
		if a, _, _ := character.ReadFeatureState(twice, character.PaladinStateKey()); !a.Equal(s) {
			t.Error("a second seed changed the state")
		}
	}
	var _ character.ClassCreationChoices = character.PaladinCreationChoices{}
}

// Requirement: doc/specs/class-paladin.md FR-122 - states holding a foreign type under the Paladin key are returned unchanged (the seed neither overwrites it nor adds the Weapon Mastery choice).
// Path: unhappy (state error).
func TestPaladinCreationChoicesSeedForeignState(t *testing.T) {
	t.Parallel()
	foreign := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("paladin_state"), must(character.NewChosenSubclass("oath_of_devotion"))))
	got := character.NewPaladinCreationChoices(choiceOf(t, "longsword", "dagger")).SeedFeatureStates(foreign)
	if !got.Equal(foreign) {
		t.Errorf("seeded %v over a foreign state, want it unchanged", got)
	}
	if _, held, _ := character.ReadFeatureState(got, character.WeaponMasteryChoiceKey()); held {
		t.Error("the Weapon Mastery choice was added")
	}
}
