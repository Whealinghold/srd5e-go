package character

import "github.com/Whealinghold/srd5e-go/content"

// PaladinCreationChoices is the ClassCreationChoices of a new first-class Paladin (class-paladin.md FR-122): it seeds the level 1
// PaladinState (the Lay On Hands pool at 5, no Channel Divinity uses, every use available) under PaladinStateKey and, when the
// Weapon Mastery choice is not empty, the WeaponMasteryChoice under WeaponMasteryChoiceKey, where its owner validates and keeps
// it. It names no content. Build it with NewPaladinCreationChoices; it emits no event. Immutable; not comparable with ==.
type PaladinCreationChoices struct{ masteries WeaponMasteryChoice }

// NewPaladinCreationChoices returns the creation choices holding the weapon mastery choice (the empty choice for none).
func NewPaladinCreationChoices(masteries WeaponMasteryChoice) PaladinCreationChoices {
	return PaladinCreationChoices{masteries: WeaponMasteryChoice{kinds: masteries.Kinds()}}
}

// SeedFeatureStates returns states with the level 1 PaladinState (and the Weapon Mastery choice) added and every other entry
// unchanged; states itself is unchanged. States that already hold the Paladin key are returned unchanged: CreateCharacter starts
// from empty states, so that is a programmer error, and the package does not panic outside Must helpers.
func (c PaladinCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	if _, held, err := ReadFeatureState(states, PaladinStateKey()); held || err != nil {
		return states
	}
	next, err := SetFeatureState(states, PaladinStateKey(), InitialPaladinState(MustPaladinLevel(1), 0))
	if err == nil && len(c.masteries.kinds) > 0 {
		next, err = SetFeatureState(next, WeaponMasteryChoiceKey(), NewWeaponMasteryChoiceState(c.masteries))
	}
	if err != nil { // unreachable: both keys are valid and the states are non-nil; only a foreign type under a key fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Paladin choices name no content entry.
func (PaladinCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = PaladinCreationChoices{}
