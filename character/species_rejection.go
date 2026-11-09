package character

import "slices"

// SpeciesTraitRejection is a rule's refusal of a species trait operation, a closed
// set. String returns the snake_case name of the variant, the value of the
// srd5e.character.rejection attribute. Variants: SpeciesBehaviourUnknown,
// SpeciesChoicesInvalid, SpeciesUseNotOffered, SpeciesUseHasOwnUseCase,
// SpeciesUseNotYetAvailable, NotEnoughSpaceForLargeForm, SpeciesUseExhausted,
// CantripWindowClosed, DieAlreadyRerolled, RestNotForCharacter.
type SpeciesTraitRejection interface {
	String() string
	isSpeciesTraitRejection()
}

// SpeciesBehaviourUnknown: the traits hold no species with a behaviour (the zero
// SpeciesTraits, a house species).
type SpeciesBehaviourUnknown struct{}

// SpeciesChoicesInvalid: the replacement cantrip of a High Elf is wrong. Only the
// rules build one; the zero value lists no problem.
type SpeciesChoicesInvalid struct{ problems []SpeciesChoiceProblem }

// Problems returns a new slice of every problem found.
func (r SpeciesChoicesInvalid) Problems() []SpeciesChoiceProblem { return slices.Clone(r.problems) }

// SpeciesUseNotOffered: the species or lineage lacks the use or the benefit.
type SpeciesUseNotOffered struct{}

// SpeciesUseHasOwnUseCase: the use belongs to RollStonesEndurance or
// UseRelentlessEndurance.
type SpeciesUseHasOwnUseCase struct{}

// SpeciesUseNotYetAvailable: the character is below the level the use needs.
// Construct it with a keyed literal.
type SpeciesUseNotYetAvailable struct{ MinimumLevel int }

// NotEnoughSpaceForLargeForm: the host reports the space too small for Large Form.
type NotEnoughSpaceForLargeForm struct{}

// SpeciesUseExhausted: every use is expended.
type SpeciesUseExhausted struct{}

// CantripWindowClosed: no Long Rest has opened the High Elf swap since the last one.
type CantripWindowClosed struct{}

// DieAlreadyRerolled: the die was already rerolled by an effect. Construct it with
// a keyed literal.
type DieAlreadyRerolled struct{ Index int }

// RestNotForCharacter: the event is for another character.
type RestNotForCharacter struct{}

// String returns "species_behaviour_unknown".
func (SpeciesBehaviourUnknown) String() string { return "species_behaviour_unknown" }

// String returns "species_choices_invalid".
func (SpeciesChoicesInvalid) String() string { return "species_choices_invalid" }

// String returns "species_use_not_offered".
func (SpeciesUseNotOffered) String() string { return "species_use_not_offered" }

// String returns "species_use_has_own_use_case".
func (SpeciesUseHasOwnUseCase) String() string { return "species_use_has_own_use_case" }

// String returns "species_use_not_yet_available".
func (SpeciesUseNotYetAvailable) String() string { return "species_use_not_yet_available" }

// String returns "not_enough_space_for_large_form".
func (NotEnoughSpaceForLargeForm) String() string { return "not_enough_space_for_large_form" }

// String returns "species_use_exhausted".
func (SpeciesUseExhausted) String() string { return "species_use_exhausted" }

// String returns "cantrip_window_closed".
func (CantripWindowClosed) String() string { return "cantrip_window_closed" }

// String returns "die_already_rerolled".
func (DieAlreadyRerolled) String() string { return "die_already_rerolled" }

// String returns "character_mismatch".
func (RestNotForCharacter) String() string { return "character_mismatch" }

func (SpeciesBehaviourUnknown) isSpeciesTraitRejection()    {}
func (SpeciesChoicesInvalid) isSpeciesTraitRejection()      {}
func (SpeciesUseNotOffered) isSpeciesTraitRejection()       {}
func (SpeciesUseHasOwnUseCase) isSpeciesTraitRejection()    {}
func (SpeciesUseNotYetAvailable) isSpeciesTraitRejection()  {}
func (NotEnoughSpaceForLargeForm) isSpeciesTraitRejection() {}
func (SpeciesUseExhausted) isSpeciesTraitRejection()        {}
func (CantripWindowClosed) isSpeciesTraitRejection()        {}
func (DieAlreadyRerolled) isSpeciesTraitRejection()         {}
func (RestNotForCharacter) isSpeciesTraitRejection()        {}
