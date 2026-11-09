package character

// FighterReason is why a Fighter feature use or option choice was refused. The
// values are in the order the checks run: the reasons of one result are reported
// in this order. The zero value is not a reason.
type FighterReason int

// The reasons a Fighter feature use or option choice is refused.
const (
	// FighterNotAFighter: the character has no Fighter levels; decided before any roll or lookup.
	FighterNotAFighter FighterReason = iota + 1
	// FighterFeatureNotGained: the Fighter level, or the subclass, does not yet give the feature.
	FighterFeatureNotGained
	// FighterNoUsesRemaining: every use is expended.
	FighterNoUsesRemaining
	// FighterNotAnAbilityCheck: the carried result is not an ability check.
	FighterNotAnAbilityCheck
	// FighterCheckNotFailed: the carried ability check succeeded.
	FighterCheckNotFailed
	// FighterNotASavingThrow: the carried result is not a saving throw.
	FighterNotASavingThrow
	// FighterDieNotRolled: the die to replace does not index a die of the roll.
	FighterDieNotRolled
	// FighterSaveNotFailed: the carried saving throw succeeded.
	FighterSaveNotFailed
	// FighterNotOwnTurn: Action Surge is used on another creature's turn.
	FighterNotOwnTurn
	// FighterAlreadyUsedThisTurn: Action Surge was already used this turn.
	FighterAlreadyUsedThisTurn
	// FighterFightingStyleAlreadyChosen: the level 1 Fighting Style is already chosen.
	FighterFightingStyleAlreadyChosen
	// FighterNoStyleToReplace: there is no level 1 Fighting Style to replace.
	FighterNoStyleToReplace
	// FighterEventNotForCharacter: the carried event names another character.
	FighterEventNotForCharacter
	// FighterNoLevelGained: the carried event is not the gaining of the current Fighter level.
	FighterNoLevelGained
	// FighterAdditionalStyleAlreadyChosen: the Additional Fighting Style is already chosen.
	FighterAdditionalStyleAlreadyChosen
	// FighterNotAFightingStyleFeat: the feat is not a Fighting Style feat.
	FighterNotAFightingStyleFeat
	// FighterSameFeat: the replacement names the feat it replaces.
	FighterSameFeat
	// FighterFeatNotRepeatable: the feat is held and is not Repeatable.
	FighterFeatNotRepeatable
	// FighterWrongMasteryCount: the choice does not name the allowed number of kinds.
	FighterWrongMasteryCount
	// FighterDuplicateMastery: a weapon kind is named twice.
	FighterDuplicateMastery
	// FighterNotSimpleOrMartialWeapon: an item is not a Simple or Martial weapon.
	FighterNotSimpleOrMartialWeapon
	// FighterDropsExistingMastery: the choice drops a kind already chosen.
	FighterDropsExistingMastery
	// FighterNoLongRestFinished: the swap is not at the end of a Long Rest.
	FighterNoLongRestFinished
	// FighterNotCurrentlyChosen: the kind to remove is not chosen.
	FighterNotCurrentlyChosen
	// FighterAlreadyChosen: the kind to add is already chosen.
	FighterAlreadyChosen
	// FighterMasteryNotUsable: the weapon's mastery property cannot be used.
	FighterMasteryNotUsable
	// FighterNotInCombat: Heroic Warrior applies only during combat.
	FighterNotInCombat
	// FighterAlreadyHasInspiration: the character already has Heroic Inspiration.
	FighterAlreadyHasInspiration
	// FighterNoHitPoints: the character is at 0 Hit Points.
	FighterNoHitPoints
	// FighterNotBloodied: the character is not Bloodied.
	FighterNotBloodied
)

func fighterReasonNames() []string {
	return []string{"not_a_fighter", "feature_not_gained", "no_uses_remaining", "not_an_ability_check", "check_not_failed",
		"not_a_saving_throw", "die_not_rolled", "save_not_failed", "not_own_turn", "already_used_this_turn",
		"fighting_style_already_chosen", "no_style_to_replace", "event_not_for_character", "no_level_gained",
		"additional_style_already_chosen", "not_a_fighting_style_feat", "same_feat", "feat_not_repeatable",
		"wrong_mastery_count", "duplicate_mastery", "not_simple_or_martial_weapon", "drops_existing_mastery",
		"no_long_rest_finished", "not_currently_chosen", "already_chosen", "mastery_not_usable", "not_in_combat",
		"already_has_inspiration", "no_hit_points", "not_bloodied"}
}

// String returns the snake_case name ("not_a_fighter", ...), the
// srd5e.character.rejection value, or "unknown".
func (r FighterReason) String() string {
	if r < 1 || int(r) > len(fighterReasonNames()) {
		return "unknown"
	}
	return fighterReasonNames()[r-1]
}

// Valid reports whether r is one of the reasons.
func (r FighterReason) Valid() bool { return r >= 1 && int(r) <= len(fighterReasonNames()) }
