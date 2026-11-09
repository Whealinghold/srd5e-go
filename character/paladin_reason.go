package character

// PaladinReason is why a Paladin operation refused or did not apply a feature.
// The values are declared in the order rejections are reported when several hold
// (class-paladin.md FR-055, FR-106). The zero value is not a reason.
type PaladinReason int

// The reasons a Paladin operation is refused or not applicable.
const (
	// PaladinNotAPaladin: the character has no Paladin levels.
	PaladinNotAPaladin PaladinReason = iota + 1
	// PaladinInvalidAmount: a Lay On Hands restore amount is negative.
	PaladinInvalidAmount
	// PaladinEmptyRequest: a Lay On Hands use restores nothing and removes no condition.
	PaladinEmptyRequest
	// PaladinConditionNotRemovable: the condition is not Poisoned or one Restoring Touch removes.
	PaladinConditionNotRemovable
	// PaladinRestoringTouchNotAvailable: Restoring Touch needs the level of its feature entry.
	PaladinRestoringTouchNotAvailable
	// PaladinExceedsPool: the request costs more than the Lay On Hands pool holds.
	PaladinExceedsPool
	// PaladinFeatureNotGained: the Paladin level is below the feature's level, or the subclass is not chosen.
	PaladinFeatureNotGained
	// PaladinOptionNotAvailable: the Channel Divinity option is not yet available.
	PaladinOptionNotAvailable
	// PaladinNotTakingAttackAction: Sacred Weapon needs the Attack action.
	PaladinNotTakingAttackAction
	// PaladinNoMeleeWeaponHeld: Sacred Weapon needs a held Melee weapon.
	PaladinNoMeleeWeaponHeld
	// PaladinNoUsesRemaining: every Channel Divinity use is expended.
	PaladinNoUsesRemaining
	// PaladinAlreadyUsedSinceLongRest: the free cast or Holy Nimbus was used and no Long Rest has been finished.
	PaladinAlreadyUsedSinceLongRest
	// PaladinNotExpended: the Holy Nimbus use is available, so there is nothing to restore.
	PaladinNotExpended
	// PaladinNoLevel5SlotAvailable: the chosen pool has no level 5 spell slot.
	PaladinNoLevel5SlotAvailable
	// PaladinLevelTooLow: the Fighting Style needs Paladin level 2.
	PaladinLevelTooLow
	// PaladinAlreadyChosen: the Fighting Style is already chosen.
	PaladinAlreadyChosen
	// PaladinNotAFightingStyleFeat: the feat is not of the Fighting Style category.
	PaladinNotAFightingStyleFeat
	// PaladinFeatNotRepeatable: the feat is already held and not Repeatable.
	PaladinFeatNotRepeatable
	// PaladinWrongCount: Blessed Warrior takes exactly two cantrips.
	PaladinWrongCount
	// PaladinDuplicate: a Blessed Warrior cantrip is named twice or already held.
	PaladinDuplicate
	// PaladinNotACantrip: the spell is not a cantrip.
	PaladinNotACantrip
	// PaladinNotOnClericList: the cantrip is not on the Cleric list.
	PaladinNotOnClericList
	// PaladinNotBlessedWarrior: a cantrip replacement needs Blessed Warrior.
	PaladinNotBlessedWarrior
	// PaladinNotHeld: the cantrip to replace is not one of the Blessed Warrior cantrips.
	PaladinNotHeld
	// PaladinNotActive: the effect to end is not active.
	PaladinNotActive
	// PaladinReasonNotApplicable: the effect cannot end for that reason.
	PaladinReasonNotApplicable
	// PaladinNotACandidate: the chosen aura is not among the candidates.
	PaladinNotACandidate
)

func paladinReasonNames() []string {
	return []string{"not_a_paladin", "invalid_amount", "empty_request", "condition_not_removable", "restoring_touch_not_available", "exceeds_pool",
		"feature_not_gained", "option_not_available", "not_taking_attack_action", "no_melee_weapon_held", "no_uses_remaining", "already_used_since_long_rest",
		"not_expended", "no_level_5_slot_available", "level_too_low", "already_chosen", "not_a_fighting_style_feat", "feat_not_repeatable", "wrong_count",
		"duplicate", "not_a_cantrip", "not_on_cleric_list", "not_blessed_warrior", "not_held", "not_active", "reason_not_applicable", "not_a_candidate"}
}

// String returns the snake_case name ("exceeds_pool", ...), the srd5e.character.rejection
// value, or "unknown".
func (r PaladinReason) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return paladinReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r PaladinReason) Valid() bool { return r >= 1 && int(r) <= len(paladinReasonNames()) }
