package character

// RangerReason is why a Ranger operation refused or did not apply a feature. The values are declared in the order rejections
// are reported when several hold (class-ranger.md section 4, FR-105). The zero value is not a reason.
type RangerReason int

// The reasons a Ranger operation is refused or not applicable.
const (
	// RangerNotARanger: the character has no Ranger levels.
	RangerNotARanger RangerReason = iota + 1
	// RangerFeatureNotAvailable: the Ranger level is below the feature's level.
	RangerFeatureNotAvailable
	// RangerNotHunter: the feature belongs to the Hunter subclass, which the character has not chosen.
	RangerNotHunter
	// RangerNoUsesRemaining: every use of the feature is expended.
	RangerNoUsesRemaining
	// RangerWrongMoment: a replacement is attempted at a moment the feature does not allow.
	RangerWrongMoment
	// RangerNothingToReplace: a replacement is attempted when no choice was made.
	RangerNothingToReplace
	// RangerAlreadyChosen: the choice is already made and cannot be made again.
	RangerAlreadyChosen
	// RangerSameOption: the replacement option is the one already held.
	RangerSameOption
	// RangerUnknownOption: the option id is not one of the feature's options.
	RangerUnknownOption
	// RangerNotProficient: the skill is not one the character is proficient in.
	RangerNotProficient
	// RangerAlreadyHasExpertise: the skill already has Expertise.
	RangerAlreadyHasExpertise
	// RangerNoGrantRemaining: every feat grant of that category is used.
	RangerNoGrantRemaining
	// RangerInvalidChoice: counts, distinctness, membership or the feat and mastery rules refuse the choice.
	RangerInvalidChoice
	// RangerNoReaction: the feature takes a Reaction and none is available.
	RangerNoReaction
)

func rangerReasonNames() []string {
	return []string{"not_a_ranger", "feature_not_available", "not_hunter", "no_uses_remaining", "wrong_moment", "nothing_to_replace",
		"already_chosen", "same_option", "unknown_option", "not_proficient", "already_has_expertise", "no_grant_remaining", "invalid_choice", "no_reaction"}
}

// String returns the snake_case name ("no_uses_remaining", ...), the srd5e.character.rejection value, or "unknown".
func (r RangerReason) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return rangerReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r RangerReason) Valid() bool { return r >= 1 && int(r) <= len(rangerReasonNames()) }

// RangerRiderReason is why a Ranger rider or extra attack is not eligible (class-ranger.md T9). The zero value is not a reason.
type RangerRiderReason int

// The reasons a rider or extra attack is not eligible, in the order they are checked.
const (
	// RangerRiderFeatureNotAvailable: the Ranger level is below the feature's, or the Hunter is not chosen.
	RangerRiderFeatureNotAvailable RangerRiderReason = iota + 1
	// RangerRiderOptionNotHeld: the Hunter's Prey option is not the one the rider belongs to.
	RangerRiderOptionNotHeld
	// RangerRiderNotWeaponAttack: the attack is not made with a weapon.
	RangerRiderNotWeaponAttack
	// RangerRiderNotMarked: the first target is not marked by Hunter's Mark.
	RangerRiderNotMarked
	// RangerRiderUsedThisTurn: the once-per-turn use is spent.
	RangerRiderUsedThisTurn
	// RangerRiderSameCreature: the second target is the first.
	RangerRiderSameCreature
	// RangerRiderTooFar: the second target is farther than the feature allows.
	RangerRiderTooFar
	// RangerRiderOutOfRange: the second target is outside the weapon's range.
	RangerRiderOutOfRange
	// RangerRiderAlreadyAttacked: the second target was already attacked this turn.
	RangerRiderAlreadyAttacked
	// RangerRiderNotVisible: the second target cannot be seen.
	RangerRiderNotVisible
)

func rangerRiderReasonNames() []string {
	return []string{"feature_not_available", "option_not_held", "not_weapon_attack", "not_marked", "used_this_turn", "same_creature",
		"too_far", "out_of_range", "already_attacked", "not_visible"}
}

// String returns the snake_case name, or "unknown".
func (r RangerRiderReason) String() string {
	if r < 1 || int(r) > len(rangerRiderReasonNames()) {
		return "unknown"
	}
	return rangerRiderReasonNames()[r-1]
}
