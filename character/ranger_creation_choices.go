package character

import "github.com/Whealinghold/srd5e-go/content"

// RangerCreationChoices is the ClassCreationChoices of a new first-class Ranger (class-ranger.md FR-109): it seeds the initial
// RangerState (nothing chosen, no use expended) under RangerStateKey and, when the Weapon Mastery choice is not empty, the
// WeaponMasteryChoice under WeaponMasteryChoiceKey, where its owner validates and keeps it. It names no content. Build it with
// NewRangerCreationChoices; it emits no event. Immutable; not comparable with ==.
type RangerCreationChoices struct{ masteries WeaponMasteryChoice }

// NewRangerCreationChoices returns the creation choices holding the weapon mastery choice (the empty choice for none).
func NewRangerCreationChoices(masteries WeaponMasteryChoice) RangerCreationChoices {
	return RangerCreationChoices{masteries: WeaponMasteryChoice{kinds: masteries.Kinds()}}
}

// SeedFeatureStates returns states with the initial RangerState (and the Weapon Mastery choice) added and every other entry
// unchanged; states itself is unchanged. States that already hold the Ranger key are returned unchanged: CreateCharacter starts
// from empty states, so that is a programmer error, and the package does not panic outside Must helpers.
func (c RangerCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	if _, held, err := ReadFeatureState(states, RangerStateKey()); held || err != nil {
		return states
	}
	next, err := SetFeatureState(states, RangerStateKey(), InitialRangerState())
	if err == nil && len(c.masteries.kinds) > 0 {
		next, err = SetFeatureState(next, WeaponMasteryChoiceKey(), NewWeaponMasteryChoiceState(c.masteries))
	}
	if err != nil { // unreachable: both keys are valid and the states are non-nil; only a foreign type under a key fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Ranger choices name no content entry.
func (RangerCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = RangerCreationChoices{}
