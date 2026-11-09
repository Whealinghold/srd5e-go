package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FeatRejection is a rule's refusal of a feat operation, a closed set. String
// returns the snake_case name of the variant, the value of the
// srd5e.character.rejection attribute. Variants: FeatBehaviourUnknown,
// FeatOutsideGrant, FeatNotRepeatable, FeatPrerequisiteNotMet,
// FeatChoicesInvalid, FeatIncreaseRejected, SpellChangeNotAllowedNow,
// SpellChangeAlreadyUsed, FeatUseAlreadyExpended, FeatNotHeld,
// FeatHasNoLimitedUse, FeatNotReplaceable.
type FeatRejection interface {
	String() string
	isFeatRejection()
}

// FeatBehaviourUnknown: no behaviour is defined for the feat (only the 17 SRD
// feats have one). Construct it with a keyed literal.
type FeatBehaviourUnknown struct{ Feat content.FeatID }

// FeatOutsideGrant: the feat is not one the grant allows.
type FeatOutsideGrant struct{ Feat content.FeatID }

// FeatNotRepeatable: the character already holds the feat and it is not
// Repeatable.
type FeatNotRepeatable struct{ Feat content.FeatID }

// FeatPrerequisiteNotMet: at least one prerequisite clause is unmet. Only the
// rules build one; the zero value lists none.
type FeatPrerequisiteNotMet struct{ unmet []content.Prerequisite }

// Unmet returns a new slice of every unmet clause, in the feat's clause order.
func (r FeatPrerequisiteNotMet) Unmet() []content.Prerequisite { return slices.Clone(r.unmet) }

// FeatChoicesInvalid: the choices are wrong. Only the rules build one; the zero
// value lists no problem.
type FeatChoicesInvalid struct{ problems []FeatChoiceProblem }

// Problems returns a new slice of every problem found.
func (r FeatChoicesInvalid) Problems() []FeatChoiceProblem { return slices.Clone(r.problems) }

// FeatIncreaseRejected: the ability score increase the feat gives was refused.
// Construct it with a keyed literal.
type FeatIncreaseRejected struct{ Rejection AbilityScoreIncreaseRejection }

// SpellChangeNotAllowedNow: Spell Change is only allowed in response to the
// character gaining a level now.
type SpellChangeNotAllowedNow struct{}

// SpellChangeAlreadyUsed: Spell Change was already used at this total level.
type SpellChangeAlreadyUsed struct{}

// FeatUseAlreadyExpended: the feat's use is already expended. Construct it with
// a keyed literal.
type FeatUseAlreadyExpended struct{ Feat content.FeatID }

// FeatNotHeld: the character does not hold the feat. Construct it with a keyed
// literal.
type FeatNotHeld struct{ Feat content.FeatID }

// FeatHasNoLimitedUse: the feat has no limited-use benefit. Construct it with a
// keyed literal.
type FeatHasNoLimitedUse struct{ Feat content.FeatID }

// FeatNotReplaceable: the feat gave an ability score increase, which cannot be
// undone. Construct it with a keyed literal.
type FeatNotReplaceable struct{ Feat content.FeatID }

// String returns "feat_behaviour_unknown".
func (FeatBehaviourUnknown) String() string { return "feat_behaviour_unknown" }

// String returns "feat_outside_grant".
func (FeatOutsideGrant) String() string { return "feat_outside_grant" }

// String returns "feat_not_repeatable".
func (FeatNotRepeatable) String() string { return "feat_not_repeatable" }

// String returns "feat_prerequisite_not_met".
func (FeatPrerequisiteNotMet) String() string { return "feat_prerequisite_not_met" }

// String returns "feat_choices_invalid".
func (FeatChoicesInvalid) String() string { return "feat_choices_invalid" }

// String returns "feat_increase_rejected".
func (FeatIncreaseRejected) String() string { return "feat_increase_rejected" }

// String returns "spell_change_not_allowed_now".
func (SpellChangeNotAllowedNow) String() string { return "spell_change_not_allowed_now" }

// String returns "spell_change_already_used".
func (SpellChangeAlreadyUsed) String() string { return "spell_change_already_used" }

// String returns "feat_use_already_expended".
func (FeatUseAlreadyExpended) String() string { return "feat_use_already_expended" }

// String returns "feat_not_held".
func (FeatNotHeld) String() string { return "feat_not_held" }

// String returns "feat_has_no_limited_use".
func (FeatHasNoLimitedUse) String() string { return "feat_has_no_limited_use" }

// String returns "feat_not_replaceable".
func (FeatNotReplaceable) String() string { return "feat_not_replaceable" }

func (FeatBehaviourUnknown) isFeatRejection()     {}
func (FeatOutsideGrant) isFeatRejection()         {}
func (FeatNotRepeatable) isFeatRejection()        {}
func (FeatPrerequisiteNotMet) isFeatRejection()   {}
func (FeatChoicesInvalid) isFeatRejection()       {}
func (FeatIncreaseRejected) isFeatRejection()     {}
func (SpellChangeNotAllowedNow) isFeatRejection() {}
func (SpellChangeAlreadyUsed) isFeatRejection()   {}
func (FeatUseAlreadyExpended) isFeatRejection()   {}
func (FeatNotHeld) isFeatRejection()              {}
func (FeatHasNoLimitedUse) isFeatRejection()      {}
func (FeatNotReplaceable) isFeatRejection()       {}
