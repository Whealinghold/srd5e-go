package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: doc/specs/class-ranger.md FR-109, FR-110 - RangerCreationChoices seeds the initial RangerState under RangerStateKey and the Weapon Mastery choice under WeaponMasteryChoiceKey when it is not empty,
// keeps every other entry, leaves its argument unchanged, names no content, and returns states already holding the Ranger key unchanged.
// Path: happy / unhappy (empty case, state error).
func TestRangerCreationChoicesSeed(t *testing.T) {
	t.Parallel()
	empty := character.FeatureStates{}
	other := must(character.SetFeatureState(empty, character.ClericChoicesKey(), must(character.NewClericChoices(character.DivineOrderProtector, "", 0))))
	for _, masteries := range []character.WeaponMasteryChoice{{}, choiceOf(t, "longbow", "shortsword")} {
		choices := character.NewRangerCreationChoices(masteries)
		seeded := choices.SeedFeatureStates(other)
		s, ok, err := character.ReadFeatureState(seeded, character.RangerStateKey())
		if err != nil || !ok || !s.Equal(character.InitialRangerState()) {
			t.Fatalf("seeded state = %+v, %v, %v", s, ok, err)
		}
		if _, ok, _ := character.ReadFeatureState(seeded, character.ClericChoicesKey()); !ok {
			t.Error("the other entry was dropped")
		}
		if _, ok, _ := character.ReadFeatureState(other, character.RangerStateKey()); ok {
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
		if a, _, _ := character.ReadFeatureState(twice, character.RangerStateKey()); !a.Equal(s) {
			t.Error("a second seed changed the state")
		}
	}
	var _ character.ClassCreationChoices = character.RangerCreationChoices{}
}

// Requirement: doc/specs/class-ranger.md FR-109 - states holding a foreign type under the Ranger key are returned unchanged (the seed neither overwrites it nor adds the Weapon Mastery choice).
// Path: unhappy (state error).
func TestRangerCreationChoicesSeedForeignState(t *testing.T) {
	t.Parallel()
	foreign := must(character.SetFeatureState(character.FeatureStates{}, character.MustFeatureStateKey[character.ChosenSubclass]("ranger_state"), must(character.NewChosenSubclass("hunter"))))
	got := character.NewRangerCreationChoices(choiceOf(t, "longbow", "shortsword")).SeedFeatureStates(foreign)
	if !got.Equal(foreign) {
		t.Errorf("seeded %v over a foreign state, want it unchanged", got)
	}
	if _, held, _ := character.ReadFeatureState(got, character.WeaponMasteryChoiceKey()); held {
		t.Error("the Weapon Mastery choice was added")
	}
}

// Requirement: doc/specs/class-ranger.md FR-109, FR-110 - states that already hold a RangerState (here with a use expended) are returned unchanged by the seed: the held state is neither replaced by the initial one nor given the Weapon Mastery choice.
// Path: unhappy (state error: the key is already held).
func TestRangerCreationChoicesSeedKeepsAHeldState(t *testing.T) {
	t.Parallel()
	held := must(character.SetFeatureState(character.FeatureStates{}, character.RangerStateKey(), rgrState(t, character.RangerStateParts{FavoredEnemySpent: 1})))
	got := character.NewRangerCreationChoices(choiceOf(t, "longbow", "shortsword")).SeedFeatureStates(held)
	s, _, _ := character.ReadFeatureState(got, character.RangerStateKey())
	if s.FavoredEnemySpent() != 1 || !got.Equal(held) {
		t.Errorf("seeded %v over a held state, want it unchanged", got)
	}
	if _, has, _ := character.ReadFeatureState(got, character.WeaponMasteryChoiceKey()); has {
		t.Error("the Weapon Mastery choice was added over a held state")
	}
}
