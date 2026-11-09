package character

// ClericReason is why a Cleric operation refused or did not apply a feature. The
// values follow the order of the checks of class-cleric.md table 4.14.1: the
// reason reported when several hold is the first by use case. The zero value is
// not a reason.
type ClericReason int

// The reasons a Cleric operation is refused or not applicable.
const (
	// ClericNotACleric: the character has no Cleric levels.
	ClericNotACleric ClericReason = iota + 1
	// ClericBelowLevel2: Channel Divinity needs Cleric level 2.
	ClericBelowLevel2
	// ClericEffectNotAvailable: the Channel Divinity effect is not one the Cleric has.
	ClericEffectNotAvailable
	// ClericNoUsesRemaining: every Channel Divinity use is expended.
	ClericNoUsesRemaining
	// ClericTargetIsSelf: Divine Spark targets another creature.
	ClericTargetIsSelf
	// ClericTargetNotVisible: the Cleric cannot see the target.
	ClericTargetNotVisible
	// ClericTargetOutOfRange: the target is farther than the effect's range.
	ClericTargetOutOfRange
	// ClericDamageTypeNotAllowed: the damage type is not Necrotic or Radiant.
	ClericDamageTypeNotAllowed
	// ClericSearUndeadNotGained: Sear Undead needs Cleric level 5.
	ClericSearUndeadNotGained
	// ClericTargetNotUndead: a Turn Undead target is not an Undead.
	ClericTargetNotUndead
	// ClericDuplicateTarget: a Preserve Life target is named twice.
	ClericDuplicateTarget
	// ClericTargetNotBloodied: a Preserve Life target is not Bloodied.
	ClericTargetNotBloodied
	// ClericNegativeAllocation: a Preserve Life allocation is below 0.
	ClericNegativeAllocation
	// ClericAllocationExceedsRoom: an allocation restores a target above half its Hit Point maximum.
	ClericAllocationExceedsRoom
	// ClericAllocationExceedsPool: the allocations total more than the pool.
	ClericAllocationExceedsPool
	// ClericAlreadyChosen: the Divine Order or the Blessed Strikes option is already chosen.
	ClericAlreadyChosen
	// ClericOrderMissing: no Divine Order is named.
	ClericOrderMissing
	// ClericBothOrders: both Divine Orders are named.
	ClericBothOrders
	// ClericOrderUnknown: the Divine Order name is not Protector or Thaumaturge.
	ClericOrderUnknown
	// ClericExtraCantripMissing: a Thaumaturge names no extra cantrip.
	ClericExtraCantripMissing
	// ClericExtraCantripNotClericCantrip: the extra cantrip is not a cantrip of the Cleric list.
	ClericExtraCantripNotClericCantrip
	// ClericExtraCantripAlreadyKnown: the extra cantrip is already known.
	ClericExtraCantripAlreadyKnown
	// ClericBelowLevel7: Blessed Strikes needs Cleric level 7.
	ClericBelowLevel7
	// ClericOptionMissing: no Blessed Strikes option is named.
	ClericOptionMissing
	// ClericBothOptions: both Blessed Strikes options are named.
	ClericBothOptions
	// ClericUnknownOption: the Blessed Strikes option name is unknown.
	ClericUnknownOption
	// ClericBelowLevel10: Divine Intervention needs Cleric level 10.
	ClericBelowLevel10
	// ClericNotAvailable: Divine Intervention is not available until a Long Rest.
	ClericNotAvailable
	// ClericNotClericSpell: the spell is not on the Cleric list and is not an unlocked grant.
	ClericNotClericSpell
	// ClericSpellLevelTooHigh: the spell is above level 5.
	ClericSpellLevelTooHigh
	// ClericReactionCastingTime: the chosen casting time is a Reaction.
	ClericReactionCastingTime
	// ClericNoOptionChosen: the Cleric of level 7 or higher has chosen no Blessed Strikes option.
	ClericNoOptionChosen
	// ClericOtherOptionChosen: the Cleric holds the other Blessed Strikes option.
	ClericOtherOptionChosen
	// ClericTurnSpent: Divine Strike was already used this turn.
	ClericTurnSpent
	// ClericNotWeaponAttack: the attack roll does not use a weapon.
	ClericNotWeaponAttack
	// ClericNotClericCantrip: the spell is not a Cleric cantrip.
	ClericNotClericCantrip
	// ClericNoDamageDealt: the cantrip dealt no damage.
	ClericNoDamageDealt
	// ClericBelowLevel14: Improved Blessed Strikes needs Cleric level 14.
	ClericBelowLevel14
	// ClericRecipientOutOfRange: the Temporary Hit Points recipient is farther than 60 feet.
	ClericRecipientOutOfRange
)

func clericReasonNames() []string {
	return []string{"not_a_cleric", "below_level_2", "effect_not_available", "no_uses_remaining", "target_is_self", "target_not_visible",
		"target_out_of_range", "damage_type_not_allowed", "sear_undead_not_gained", "target_not_undead", "duplicate_target", "target_not_bloodied",
		"negative_allocation", "allocation_exceeds_room", "allocation_exceeds_pool", "already_chosen", "order_missing", "both_orders", "order_unknown",
		"extra_cantrip_missing", "extra_cantrip_not_cleric_cantrip", "extra_cantrip_already_known", "below_level_7", "option_missing", "both_options",
		"unknown_option", "below_level_10", "not_available", "not_cleric_spell", "spell_level_too_high", "reaction_casting_time", "no_option_chosen",
		"other_option_chosen", "turn_spent", "not_weapon_attack", "not_cleric_cantrip", "no_damage_dealt", "below_level_14", "recipient_out_of_range"}
}

// String returns the snake_case name ("not_a_cleric", ...), the
// srd5e.character.rejection value, or "unknown".
func (r ClericReason) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return clericReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r ClericReason) Valid() bool { return r >= 1 && int(r) <= len(clericReasonNames()) }
