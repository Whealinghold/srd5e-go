package character

// BarbarianReason is why a Barbarian feature use or option choice was refused. The
// values are in the order the checks run: the reasons of one result are reported
// in this order. The zero value is not a reason.
type BarbarianReason int

// The reasons a Barbarian feature use or option choice is refused.
const (
	// BarbarianNotABarbarian: the character has no Barbarian levels; decided before any roll or lookup.
	BarbarianNotABarbarian BarbarianReason = iota + 1
	// BarbarianFeatureNotGained: the Barbarian level does not yet give the feature.
	BarbarianFeatureNotGained
	// BarbarianNotABerserker: the feature belongs to the Path of the Berserker and the subclass is another or none.
	BarbarianNotABerserker
	// BarbarianRageAlreadyActive: a Rage is already active.
	BarbarianRageAlreadyActive
	// BarbarianRageNotActive: no Rage is active.
	BarbarianRageNotActive
	// BarbarianWearingHeavyArmor: the character wears Heavy armor.
	BarbarianWearingHeavyArmor
	// BarbarianIncapacitated: the character has the Incapacitated condition.
	BarbarianIncapacitated
	// BarbarianNoBonusAction: no Bonus Action is available this turn.
	BarbarianNoBonusAction
	// BarbarianNoReaction: the Reaction is spent.
	BarbarianNoReaction
	// BarbarianNoRageUsesRemaining: every Rage use is expended.
	BarbarianNoRageUsesRemaining
	// BarbarianRageBeganThisTurn: the Rage began this turn, so there is nothing to extend yet.
	BarbarianRageBeganThisTurn
	// BarbarianRageAlreadyExtended: the Rage was already extended this turn.
	BarbarianRageAlreadyExtended
	// BarbarianRageLastsWithoutExtension: Persistent Rage makes an extension unnecessary.
	BarbarianRageLastsWithoutExtension
	// BarbarianNotFirstAttackRoll: Reckless Attack is declared on the first attack roll of the turn only.
	BarbarianNotFirstAttackRoll
	// BarbarianRecklessNotDeclared: Reckless Attack was not used this turn.
	BarbarianRecklessNotDeclared
	// BarbarianNotAStrengthAttack: the attack roll does not use Strength.
	BarbarianNotAStrengthAttack
	// BarbarianAttackMissed: the attack roll missed.
	BarbarianAttackMissed
	// BarbarianAttackHasDisadvantage: the attack roll has Disadvantage.
	BarbarianAttackHasDisadvantage
	// BarbarianEffectNotUnlocked: a Brutal Strike effect is not yet unlocked at the level.
	BarbarianEffectNotUnlocked
	// BarbarianDuplicateEffect: the same Brutal Strike effect is named twice.
	BarbarianDuplicateEffect
	// BarbarianTooManyEffects: more Brutal Strike effects than the level allows.
	BarbarianTooManyEffects
	// BarbarianFrenzyTargetAlreadyHit: a target was already hit this turn, so Frenzy has been spent.
	BarbarianFrenzyTargetAlreadyHit
	// BarbarianNotDroppedToZero: the character did not drop to 0 Hit Points.
	BarbarianNotDroppedToZero
	// BarbarianPersistentRageSpent: Persistent Rage was used since the last Long Rest.
	BarbarianPersistentRageSpent
	// BarbarianPresenceNotRestored: Intimidating Presence was used and not restored.
	BarbarianPresenceNotRestored
	// BarbarianPresenceNotUsed: Intimidating Presence has not been used, so there is nothing to restore.
	BarbarianPresenceNotUsed
	// BarbarianSourceNotWithinFiveFeet: the source of the damage is not within 5 feet.
	BarbarianSourceNotWithinFiveFeet
	// BarbarianEventNotForCharacter: the carried event names another character.
	BarbarianEventNotForCharacter
	// BarbarianWrongMasteryCount: the choice does not name the allowed number of kinds.
	BarbarianWrongMasteryCount
	// BarbarianDuplicateMastery: a weapon kind is named twice.
	BarbarianDuplicateMastery
	// BarbarianNotSimpleOrMartialMeleeWeapon: an item is not a Simple or Martial Melee weapon.
	BarbarianNotSimpleOrMartialMeleeWeapon
	// BarbarianDropsExistingMastery: the choice drops a kind already chosen.
	BarbarianDropsExistingMastery
	// BarbarianNoLongRestFinished: the swap is not at the end of a Long Rest.
	BarbarianNoLongRestFinished
	// BarbarianNotCurrentlyChosen: the kind to remove is not chosen.
	BarbarianNotCurrentlyChosen
	// BarbarianAlreadyChosen: the kind to add is already chosen.
	BarbarianAlreadyChosen
)

func barbarianReasonNames() []string {
	return []string{"not_a_barbarian", "feature_not_gained", "not_a_berserker", "rage_already_active", "rage_not_active",
		"wearing_heavy_armor", "incapacitated", "no_bonus_action", "no_reaction", "no_rage_uses_remaining",
		"rage_began_this_turn", "rage_already_extended", "rage_lasts_without_extension", "not_first_attack_roll",
		"reckless_not_declared", "not_a_strength_attack", "attack_missed", "attack_has_disadvantage",
		"effect_not_unlocked", "duplicate_effect", "too_many_effects", "frenzy_target_already_hit", "not_dropped_to_zero",
		"persistent_rage_spent", "presence_not_restored", "presence_not_used", "source_not_within_five_feet",
		"event_not_for_character", "wrong_mastery_count", "duplicate_mastery", "not_simple_or_martial_melee_weapon",
		"drops_existing_mastery", "no_long_rest_finished", "not_currently_chosen", "already_chosen"}
}

// String returns the snake_case name ("not_a_barbarian", ...), the
// srd5e.character.rejection value, or "unknown".
func (r BarbarianReason) String() string {
	if r < 1 || int(r) > len(barbarianReasonNames()) {
		return "unknown"
	}
	return barbarianReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r BarbarianReason) Valid() bool { return r >= 1 && int(r) <= len(barbarianReasonNames()) }
