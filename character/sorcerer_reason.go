package character

// SorcererRejectionKind is why a Sorcerer feature refused to apply (class-sorcerer.md section 5). Its String is the
// srd5e.character.rejection value of the Sorcerer use cases. The zero value is not a kind and means "no rejection".
type SorcererRejectionKind int

// The kinds of Sorcerer rejection.
const (
	// SorcererNotASorcerer: the character has no Sorcerer levels, or holds no Sorcerer state and is not at class level 1.
	SorcererNotASorcerer SorcererRejectionKind = iota + 1
	// SorcererFeatureNotGained: the feature needs a higher Sorcerer level or the Draconic Sorcery subclass.
	SorcererFeatureNotGained
	// SorcererElementalAffinityChoiceInvalid: the damage type is not Acid, Cold, Fire, Lightning or Poison, or one is already chosen.
	SorcererElementalAffinityChoiceInvalid
	// SorcererElementalAffinityTypeMismatch: no type is chosen, or the damage dealt does not include the chosen type.
	SorcererElementalAffinityTypeMismatch
	// SorcererElementalAffinityAlreadyApplied: Elemental Affinity was already added to a damage roll of this spell.
	SorcererElementalAffinityAlreadyApplied
	// SorcererMetamagicSelectionInvalid: the Metamagic selection has the wrong size, a repeat or an option content does not list.
	SorcererMetamagicSelectionInvalid
	// SorcererMetamagicReplacementInvalid: the selection drops more than one known option.
	SorcererMetamagicReplacementInvalid
	// SorcererInnateSorceryAlreadyActive: Innate Sorcery is already active.
	SorcererInnateSorceryAlreadyActive
	// SorcererInnateSorceryNoUsesLeft: no use of Innate Sorcery remains and the points route is unavailable.
	SorcererInnateSorceryNoUsesLeft
	// SorcererInnateSorceryUsesRemain: the points route needs no uses left, and some remain.
	SorcererInnateSorceryUsesRemain
	// SorcererSorceryIncarnateUnavailable: the points route of Sorcery Incarnate needs Sorcerer level 7.
	SorcererSorceryIncarnateUnavailable
	// SorcererInnateSorceryNotActive: Innate Sorcery is not active, so it cannot end.
	SorcererInnateSorceryNotActive
	// SorcererSorceryPointsInsufficient: the Sorcery Points on hand are below the cost (Required and Available).
	SorcererSorceryPointsInsufficient
	// SorcererNoSlotToConvert: no spell slot of the level is available.
	SorcererNoSlotToConvert
	// SorcererSorceryPointsWouldExceedMaximum: converting the slot would take the points above the maximum (Cap).
	SorcererSorceryPointsWouldExceedMaximum
	// SorcererSlotLevelNotCreatable: Font of Magic creates no slot above level 5.
	SorcererSlotLevelNotCreatable
	// SorcererLevelBelowMinimum: the Sorcerer level is below the minimum of the slot (Required).
	SorcererLevelBelowMinimum
	// SorcererCreatedSlotUnavailable: no created slot of the level exists.
	SorcererCreatedSlotUnavailable
	// SorcererCreatedSlotLevelTooLow: the spell's level is above the created slot's.
	SorcererCreatedSlotLevelTooLow
	// SorcererCreatedSlotSpentThisTurn: a spell slot was already spent this turn.
	SorcererCreatedSlotSpentThisTurn
	// SorcererMetamagicOptionNotKnown: the character does not know the Metamagic option.
	SorcererMetamagicOptionNotKnown
	// SorcererMetamagicCombinationRefused: the option cannot join the options already applied to this casting.
	SorcererMetamagicCombinationRefused
	// SorcererQuickenedRestrictionApplies: Quickened Spell forbids this casting.
	SorcererQuickenedRestrictionApplies
	// SorcererMetamagicNotEligible: the spell does not suit the option (the Ineligibility says why).
	SorcererMetamagicNotEligible
	// SorcererRestorationAlreadyUsed: Sorcerous Restoration was used since the last Long Rest.
	SorcererRestorationAlreadyUsed
	// SorcererRestorationAmountInvalid: the amount is below 1, above the cap or above the points missing (Cap and Missing).
	SorcererRestorationAmountInvalid
	// SorcererRestorationNotAShortRest: Sorcerous Restoration was asked for with a rest that is not a Short Rest.
	SorcererRestorationNotAShortRest
	// SorcererDragonWingsAlreadyUsed: Dragon Wings was used since the last Long Rest.
	SorcererDragonWingsAlreadyUsed
	// SorcererDragonWingsUseNotExpended: the use of Dragon Wings is not expended, so there is nothing to restore.
	SorcererDragonWingsUseNotExpended
	// SorcererDragonWingsNotActive: the wings are not out, so they cannot be dismissed.
	SorcererDragonWingsNotActive
	// SorcererDragonCompanionFreeCastUsed: the slot-free cast of Summon Dragon was used since the last Long Rest.
	SorcererDragonCompanionFreeCastUsed
)

func sorcererRejectionNames() []string {
	return []string{"not_a_sorcerer", "feature_not_gained", "elemental_affinity_choice_invalid", "elemental_affinity_type_mismatch",
		"elemental_affinity_already_applied", "metamagic_selection_invalid", "metamagic_replacement_invalid", "innate_sorcery_already_active",
		"innate_sorcery_no_uses_left", "innate_sorcery_uses_remain", "sorcery_incarnate_unavailable", "innate_sorcery_not_active",
		"sorcery_points_insufficient", "no_slot_to_convert", "sorcery_points_would_exceed_maximum", "slot_level_not_creatable",
		"sorcerer_level_below_minimum", "created_slot_unavailable", "created_slot_level_too_low", "created_slot_spent_this_turn",
		"metamagic_option_not_known", "metamagic_combination_refused", "quickened_restriction_applies", "metamagic_not_eligible",
		"restoration_already_used", "restoration_amount_invalid", "restoration_not_a_short_rest", "dragon_wings_already_used",
		"dragon_wings_use_not_expended", "dragon_wings_not_active", "dragon_companion_free_cast_used"}
}

// String returns the snake_case name ("not_a_sorcerer", ...), the srd5e.character.rejection value, or "unknown".
func (k SorcererRejectionKind) String() string {
	if !k.Valid() {
		return "unknown"
	}
	return sorcererRejectionNames()[k-1]
}

// Valid reports whether k is one of the kinds.
func (k SorcererRejectionKind) Valid() bool { return k >= 1 && int(k) <= len(sorcererRejectionNames()) }

// MetamagicIneligibility is why a spell does not suit a Metamagic option (class-sorcerer.md section 5). The zero value
// is not a reason. It is telemetry-silent: the rejection attribute carries only metamagic_not_eligible.
type MetamagicIneligibility int

// The reasons a spell does not suit a Metamagic option.
const (
	// MetamagicNoSavingThrow: the spell forces no saving throw.
	MetamagicNoSavingThrow MetamagicIneligibility = iota + 1
	// MetamagicRangeNotEligible: the range has no distance to double (Self, Sight, Unlimited, Special, or below 5 feet).
	MetamagicRangeNotEligible
	// MetamagicDurationNotEligible: the duration is not a timed span of 1 minute or longer.
	MetamagicDurationNotEligible
	// MetamagicDurationAlreadyAtMaximum: the duration is already 24 hours or longer.
	MetamagicDurationAlreadyAtMaximum
	// MetamagicCastingTimeNotAction: the casting time is not an action.
	MetamagicCastingTimeNotAction
	// MetamagicNoMissedAttackRoll: the spell has no spell attack roll, or the roll did not miss.
	MetamagicNoMissedAttackRoll
	// MetamagicNothingToRemove: the spell has no component Subtle Spell could remove.
	MetamagicNothingToRemove
	// MetamagicDamageTypeNotTransmutable: the damage type named, or the one chosen, is not in the Transmuted Spell list or does not apply.
	MetamagicDamageTypeNotTransmutable
	// MetamagicNoHigherSlotTarget: the spell cannot target an additional creature with a higher-level slot.
	MetamagicNoHigherSlotTarget
	// MetamagicCountOutOfRange: the chosen count is below 1 or above the cap.
	MetamagicCountOutOfRange
	// MetamagicSpellFactUnknown: a fact the option needs is marked unknown.
	MetamagicSpellFactUnknown
)

func metamagicIneligibilityNames() []string {
	return []string{"no_saving_throw", "range_not_eligible", "duration_not_eligible", "duration_already_at_maximum", "casting_time_not_action",
		"no_missed_attack_roll", "nothing_to_remove", "damage_type_not_transmutable", "no_higher_slot_target", "count_out_of_range", "spell_fact_unknown"}
}

// String returns the snake_case name ("no_saving_throw", ...) or "unknown".
func (i MetamagicIneligibility) String() string {
	if i < 1 || int(i) > len(metamagicIneligibilityNames()) {
		return "unknown"
	}
	return metamagicIneligibilityNames()[i-1]
}

// SorcererRejection is why a Sorcerer operation was refused: the Kind and the figures that explain it. It is a value
// object held as a field of a result or transition, not a variant. Only the fields the Kind names are set. Construct
// it with a keyed literal; it is comparable with == only when Problems is nil.
type SorcererRejection struct {
	// Kind is the reason, never zero in a rejection.
	Kind SorcererRejectionKind
	// Required and Available are the points needed and on hand (SorcererSorceryPointsInsufficient) or the level needed (SorcererLevelBelowMinimum).
	Required, Available int
	// Cap is the limit that was exceeded; Missing is the points missing (SorcererRestorationAmountInvalid).
	Cap, Missing int
	// Feature names the feature not gained ("font_of_magic", "metamagic", ...) for SorcererFeatureNotGained.
	Feature string
	// Problems lists every problem of a refused Metamagic selection (SorcererMetamagicSelectionInvalid).
	Problems []MetamagicSelectionProblem
	// Ineligibility says why a spell does not suit the option (SorcererMetamagicNotEligible).
	Ineligibility MetamagicIneligibility
	// Fact names the unknown fact for MetamagicSpellFactUnknown.
	Fact MetamagicFact
}

// MetamagicSelectionProblem is one thing wrong with a Metamagic selection (class-sorcerer.md FR-057). The zero value is not a problem.
type MetamagicSelectionProblem int

// The problems a Metamagic selection can have.
const (
	// MetamagicSelectionWrongSize: the selection is not the entitled number of options.
	MetamagicSelectionWrongSize MetamagicSelectionProblem = iota + 1
	// MetamagicSelectionDuplicate: the selection names an option twice.
	MetamagicSelectionDuplicate
	// MetamagicSelectionUnknownOption: the selection names an option the class definition does not list.
	MetamagicSelectionUnknownOption
)

// String returns "wrong_size", "duplicate", "unknown_option" or "unknown".
func (p MetamagicSelectionProblem) String() string {
	names := []string{"wrong_size", "duplicate", "unknown_option"}
	if p < 1 || int(p) > len(names) {
		return "unknown"
	}
	return names[p-1]
}

func refuse(kind SorcererRejectionKind) SorcererRejection { return SorcererRejection{Kind: kind} }

func notGained(feature string) SorcererRejection {
	return SorcererRejection{Kind: SorcererFeatureNotGained, Feature: feature}
}
