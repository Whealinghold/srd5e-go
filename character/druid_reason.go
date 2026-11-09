package character

// DruidRejection is why a Druid operation refused to apply a feature. The numeric order of the members is the total order of
// class-druid.md D-5: when several reasons hold for one operation the reason reported is the lowest-numbered one. The eight
// groups of D-5 come first (availability, state, casting, timing, arguments, resources, limits, the expended slot), then the
// level-gain reasons. NoSlotAvailable and NoExpendedSlot stand for the rejection of a slot operation by the spell slot pool
// (SlotOperationNoSlotAvailable, SlotOperationNoExpendedSlot). The zero value is not a reason.
type DruidRejection int

// The reasons a Druid operation is refused.
const (
	// DruidFeatureNotGranted: the feature is not granted at the Druid level, or the subclass is not held.
	DruidFeatureNotGranted DruidRejection = iota + 1
	// DruidFeatureNotChosen: the other Elemental Fury option was chosen.
	DruidFeatureNotChosen
	// DruidLandTypeNotChosen: the operation needs a land type and none is chosen.
	DruidLandTypeNotChosen
	// DruidNotInBeastForm: the Druid is not in a Beast form.
	DruidNotInBeastForm
	// DruidNoActiveSanctuary: no Nature's Sanctuary Cube is active.
	DruidNoActiveSanctuary
	// DruidSanctuaryAlreadyActive: a Nature's Sanctuary Cube is already active.
	DruidSanctuaryAlreadyActive
	// DruidPrimalOrderAlreadyChosen: the Primal Order is already chosen.
	DruidPrimalOrderAlreadyChosen
	// DruidElementalFuryAlreadyChosen: the Elemental Fury option is already chosen.
	DruidElementalFuryAlreadyChosen
	// DruidCannotCastInBeastForm: spells cannot be cast in Beast form below Druid level 18.
	DruidCannotCastInBeastForm
	// DruidMaterialComponentRestricted: the spell has a costed or consumed Material component.
	DruidMaterialComponentRestricted
	// DruidChangeNotAllowedNow: the choice is only open after a Long Rest.
	DruidChangeNotAllowedNow
	// DruidRecoveryNotAllowedNow: the recovery is only open after a Short Rest.
	DruidRecoveryNotAllowedNow
	// DruidAlreadyUsedThisTurn: the once-per-turn feature was used this turn.
	DruidAlreadyUsedThisTurn
	// DruidAlreadyUsedSinceLongRest: the once-until-Long-Rest feature was used.
	DruidAlreadyUsedSinceLongRest
	// DruidInvalidConversion: Nature Magician converts at least one use.
	DruidInvalidConversion
	// DruidAttackNotEligible: the attack is not one Primal Strike applies to.
	DruidAttackNotEligible
	// DruidNotACircleSpell: the spell is not a level 1+ Circle Spell of the current land.
	DruidNotACircleSpell
	// DruidNotADruidCantrip: the spell is not a Druid cantrip, so Potent Spellcasting adds nothing.
	DruidNotADruidCantrip
	// DruidFormNotKnown: the Beast form is not a known form.
	DruidFormNotKnown
	// DruidNotABeast: the form is not a Beast.
	DruidNotABeast
	// DruidChallengeRatingTooHigh: the form's Challenge Rating is above the maximum for the Druid level.
	DruidChallengeRatingTooHigh
	// DruidFlySpeedNotAllowed: the form has a Fly Speed and the Druid level is below 8.
	DruidFlySpeedNotAllowed
	// DruidDuplicateForm: the form is named twice or is already known.
	DruidDuplicateForm
	// DruidTooManyKnownForms: more forms than the Druid level allows.
	DruidTooManyKnownForms
	// DruidItemUnassigned: an item has no disposition.
	DruidItemUnassigned
	// DruidItemCannotBeWorn: the form cannot wear the item.
	DruidItemCannotBeWorn
	// DruidMoveTooFar: the Cube would move more than 60 feet or sit farther than 120 feet from the Druid.
	DruidMoveTooFar
	// DruidNoUsesRemaining: no use of Wild Shape remains.
	DruidNoUsesRemaining
	// DruidWildShapeUsesRemain: a use of Wild Shape remains.
	DruidWildShapeUsesRemain
	// DruidNoSlotAvailable: no spell slot of the chosen level remains (SlotOperationNoSlotAvailable).
	DruidNoSlotAvailable
	// DruidSlotLevelTooHigh: Natural Recovery recovers no slot of level 6 or higher.
	DruidSlotLevelTooHigh
	// DruidRecoveryBudgetExceeded: the recovered slots total more than half the Druid level, rounded up.
	DruidRecoveryBudgetExceeded
	// DruidTooManyReplacements: a known form was already replaced since the last Long Rest.
	DruidTooManyReplacements
	// DruidNoExpendedSlot: no spell slot of the chosen level is expended (SlotOperationNoExpendedSlot).
	DruidNoExpendedSlot
	// DruidLevelNotConsecutive: a Druid level is gained one at a time.
	DruidLevelNotConsecutive
	// DruidMaximumLevelReached: no Druid level above 20.
	DruidMaximumLevelReached
)

func druidRejectionNames() []string {
	return []string{"feature_not_granted", "feature_not_chosen", "land_type_not_chosen", "not_in_beast_form", "no_active_sanctuary",
		"sanctuary_already_active", "primal_order_already_chosen", "elemental_fury_already_chosen", "cannot_cast_in_beast_form",
		"material_component_restricted", "change_not_allowed_now", "recovery_not_allowed_now", "already_used_this_turn",
		"already_used_since_long_rest", "invalid_conversion", "attack_not_eligible", "not_a_circle_spell", "not_a_druid_cantrip", "form_not_known", "not_a_beast",
		"challenge_rating_too_high", "fly_speed_not_allowed", "duplicate_form", "too_many_known_forms", "item_unassigned",
		"item_cannot_be_worn", "move_too_far", "no_uses_remaining", "wild_shape_uses_remain", "no_slot_available", "slot_level_too_high",
		"recovery_budget_exceeded", "too_many_replacements", "no_expended_slot", "level_not_consecutive", "maximum_level_reached"}
}

// String returns the snake_case name ("feature_not_granted", ...), the srd5e.character.rejection value, or "unknown".
func (r DruidRejection) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return druidRejectionNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r DruidRejection) Valid() bool { return r >= 1 && int(r) <= len(druidRejectionNames()) }

// lowestDruidRejection returns the first reason of the total order among reasons, and false when there is none.
func lowestDruidRejection(reasons ...DruidRejection) (DruidRejection, bool) {
	first := DruidRejection(0)
	for _, r := range reasons {
		if r != 0 && (first == 0 || r < first) {
			first = r
		}
	}
	return first, first != 0
}
