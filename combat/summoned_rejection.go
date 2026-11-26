package combat

// SummonRejection is why a summons or a change to a summoned creature was refused: the field of SummonCreatureRejected,
// ApplySummonedChangeRejected and SummonedRejected (summoned-and-companion-creatures.md FR-073). It carries data only. String is
// the snake_case reason, the suffix of the value srd5e.combat.result takes after "rejected_" (Table R). Every variant is a refusal
// except SummonLimitExceededRejection, which is host input the engine cannot use. Variants: SummonCreatureIDInUseRejection,
// SummonLinkCycleRejection, LevelBelowSpellRejection, NotNightRejection, FormNotEligibleRejection, SteedNotDisappearedRejection,
// TargetNotEligibleRejection, ObjectNotEligibleRejection, OriginalNotEligibleRejection, DuplicateTargetRejection,
// TooManyCreaturesRejection, CompositionNotAllowedRejection, ControlLapsedRejection, NothingToAnimateRejection,
// NothingToReassertRejection, ResistanceNotOfferedRejection, SpaceUnavailableRejection, SummonLimitExceededRejection,
// CommandNotYourCreatureRejection, CommandOutOfRangeRejection, CannotGiveVerbalCommandRejection, FamiliarCannotAttackRejection,
// FamiliarAlreadyDismissedRejection, FamiliarNotDismissedRejection, SensesNotSharableRejection, TouchNotDeliverableRejection,
// SimulacrumTooFarRejection, SimulacrumHealingRefusedRejection, SimulacrumRepairAmountRejection,
// SimulacrumRepairNeedsLongRestRejection, LifeBondOutOfReachRejection, TelepathyOnlyWithSummonerRejection.
type SummonRejection interface {
	isSummonRejection()
	String() string
}

// SummonCreatureIDInUseRejection: ID already names a summoned creature of the roster (FR-008).
type SummonCreatureIDInUseRejection struct{ ID CombatantID }

// SummonLinkCycleRejection: making Creature a creature of Summoner would close a cycle, because Summoner is, directly or through
// other summoners, a creature of Creature (plan P10). The chain is walked iteratively.
type SummonLinkCycleRejection struct{ Creature, Summoner CombatantID }

// LevelBelowSpellRejection: the cast level is below Minimum, the level of the spell itself (FR-035).
type LevelBelowSpellRejection struct{ Minimum int }

// NotNightRejection: Create Undead can be cast only at night (FR-039).
type NotNightRejection struct{}

// FormNotEligibleRejection: the familiar form is not one of the eligible forms for the origin (FR-043, FR-054).
type FormNotEligibleRejection struct{}

// SteedNotDisappearedRejection: the cast chose the same steed again but the summoner has no steed that disappeared, or the id of a
// steed that disappeared was given as a different steed (FR-056).
type SteedNotDisappearedRejection struct{}

// TargetNotEligibleRejection: the pile of bones or the corpse of the creature Creature is not an eligible target (FR-038, FR-039).
type TargetNotEligibleRejection struct{ Creature CombatantID }

// ObjectNotEligibleRejection: the object that would become Creature is magical, worn or carried, fixed to a surface or
// Gargantuan (FR-029, FR-031).
type ObjectNotEligibleRejection struct{ Creature CombatantID }

// OriginalNotEligibleRejection: the original is not a Beast or Humanoid within 10 feet for the entire casting (FR-060).
type OriginalNotEligibleRejection struct{}

// DuplicateTargetRejection: the pile of bones or the corpse for Creature is also the one of another creature of the cast (FR-038).
type DuplicateTargetRejection struct{ Creature CombatantID }

// TooManyCreaturesRejection: the cast names more creatures, or objects of more weight, than Maximum (FR-031, FR-038, FR-039,
// FR-041).
type TooManyCreaturesRejection struct{ Maximum int }

// CompositionNotAllowedRejection: the creatures of the cast do not fit one composition row allowed at the cast level (FR-039,
// FR-041).
type CompositionNotAllowedRejection struct{}

// ControlLapsedRejection: Creature is an undead creature whose control lapsed (FR-015, FR-041).
type ControlLapsedRejection struct{ Creature CombatantID }

// NothingToAnimateRejection: the animate request names no target (FR-042).
type NothingToAnimateRejection struct{}

// NothingToReassertRejection: the request to reassert control names no linked undead creature (FR-042).
type NothingToReassertRejection struct{}

// ResistanceField says which of the two choices of a Draconic Spirit was refused. The zero value is invalid.
type ResistanceField int

// The two choices.
const (
	// SharedResistanceField: the Resistance the summoner shares.
	SharedResistanceField ResistanceField = iota + 1
	// BreathDamageTypeField: the damage type of the Breath Weapon.
	BreathDamageTypeField
)

// String returns "shared_resistance", "breath_damage_type" or "unknown".
func (f ResistanceField) String() string {
	return enumName([]string{"shared_resistance", "breath_damage_type"}, int(f))
}

// ResistanceNotOfferedRejection: the damage type chosen for Field is not one of the five the Draconic Spirit resists (FR-033).
type ResistanceNotOfferedRejection struct{ Field ResistanceField }

// SpaceUnavailableRejection: the host reports no unoccupied space within range for the creature (FR-051, FR-056).
type SpaceUnavailableRejection struct{}

// SummonLimitExceededRejection: the call would leave a summoner with more than 32 summoned creatures (plan P10). It changes
// nothing.
type SummonLimitExceededRejection struct{}

// CommandNotYourCreatureRejection: Creature is not in the roster, is not linked to the summoner, is in the pocket dimension or is
// of another spell than the command's (FR-014). A group order is refused whole.
type CommandNotYourCreatureRejection struct{ Creature CombatantID }

// CommandOutOfRangeRejection: Creature is farther than RangeFeet, the range of the mental channel (FR-009).
type CommandOutOfRangeRejection struct {
	Creature  CombatantID
	RangeFeet int
}

// CannotGiveVerbalCommandRejection: the summoner cannot give a verbal command (FR-011).
type CannotGiveVerbalCommandRejection struct{}

// FamiliarCannotAttackRejection: a familiar cannot take the Attack action without a FamiliarAttackGrant (FR-048).
type FamiliarCannotAttackRejection struct{}

// FamiliarAlreadyDismissedRejection: the familiar is already in the pocket dimension (FR-050).
type FamiliarAlreadyDismissedRejection struct{}

// FamiliarNotDismissedRejection: the familiar is not in the pocket dimension, or the summoner has none to recall (FR-051).
type FamiliarNotDismissedRejection struct{}

// SensesNotSharableRejection: the familiar is in the pocket dimension, departed or farther than 100 feet (FR-046).
type SensesNotSharableRejection struct{}

// TouchReason is why a familiar cannot deliver a touch spell. The zero value is invalid.
type TouchReason int

// The four reasons, in the order they are checked.
const (
	// TouchNotPresent: the familiar is not in the encounter.
	TouchNotPresent TouchReason = iota + 1
	// TouchNotTouch: the spell does not have range Touch.
	TouchNotTouch
	// TouchTooFar: the familiar is farther than 100 feet from the summoner.
	TouchTooFar
	// TouchFamiliarCannotReact: the familiar cannot take a Reaction.
	TouchFamiliarCannotReact
)

// String returns "not_present", "not_touch", "too_far", "familiar_cannot_react" or "unknown".
func (r TouchReason) String() string {
	return enumName([]string{"not_present", "not_touch", "too_far", "familiar_cannot_react"}, int(r))
}

// TouchNotDeliverableRejection: the familiar cannot deliver the touch, for Reason (FR-047).
type TouchNotDeliverableRejection struct{ Reason TouchReason }

// SimulacrumTooFarRejection: the simulacrum is not within 5 feet of the summoner (FR-062).
type SimulacrumTooFarRejection struct{}

// SimulacrumHealingRefusedRejection: a simulacrum regains Hit Points only through its repair (FR-062).
type SimulacrumHealingRefusedRejection struct{}

// SimulacrumRepairAmountRejection: the repair restores 0 Hit Points or more than the Missing ones (FR-062).
type SimulacrumRepairAmountRejection struct{ Missing int }

// SimulacrumRepairNeedsLongRestRejection: the repair of a simulacrum takes a Long Rest (FR-062).
type SimulacrumRepairNeedsLongRestRejection struct{}

// LifeBondOutOfReachRejection: the steed is not within 5 feet of its summoner, so Life Bond gives it nothing (FR-057).
type LifeBondOutOfReachRejection struct{}

// TelepathyOnlyWithSummonerRejection: a steed's telepathy works only with its summoner (FR-058).
type TelepathyOnlyWithSummonerRejection struct{}

func (SummonCreatureIDInUseRejection) isSummonRejection()         {}
func (SummonLinkCycleRejection) isSummonRejection()               {}
func (LevelBelowSpellRejection) isSummonRejection()               {}
func (NotNightRejection) isSummonRejection()                      {}
func (FormNotEligibleRejection) isSummonRejection()               {}
func (SteedNotDisappearedRejection) isSummonRejection()           {}
func (TargetNotEligibleRejection) isSummonRejection()             {}
func (ObjectNotEligibleRejection) isSummonRejection()             {}
func (OriginalNotEligibleRejection) isSummonRejection()           {}
func (DuplicateTargetRejection) isSummonRejection()               {}
func (TooManyCreaturesRejection) isSummonRejection()              {}
func (CompositionNotAllowedRejection) isSummonRejection()         {}
func (ControlLapsedRejection) isSummonRejection()                 {}
func (NothingToAnimateRejection) isSummonRejection()              {}
func (NothingToReassertRejection) isSummonRejection()             {}
func (ResistanceNotOfferedRejection) isSummonRejection()          {}
func (SpaceUnavailableRejection) isSummonRejection()              {}
func (SummonLimitExceededRejection) isSummonRejection()           {}
func (CommandNotYourCreatureRejection) isSummonRejection()        {}
func (CommandOutOfRangeRejection) isSummonRejection()             {}
func (CannotGiveVerbalCommandRejection) isSummonRejection()       {}
func (FamiliarCannotAttackRejection) isSummonRejection()          {}
func (FamiliarAlreadyDismissedRejection) isSummonRejection()      {}
func (FamiliarNotDismissedRejection) isSummonRejection()          {}
func (SensesNotSharableRejection) isSummonRejection()             {}
func (TouchNotDeliverableRejection) isSummonRejection()           {}
func (SimulacrumTooFarRejection) isSummonRejection()              {}
func (SimulacrumHealingRefusedRejection) isSummonRejection()      {}
func (SimulacrumRepairAmountRejection) isSummonRejection()        {}
func (SimulacrumRepairNeedsLongRestRejection) isSummonRejection() {}
func (LifeBondOutOfReachRejection) isSummonRejection()            {}
func (TelepathyOnlyWithSummonerRejection) isSummonRejection()     {}

// String returns "summon_creature_id_in_use".
func (SummonCreatureIDInUseRejection) String() string { return valueSummonReasonIDInUse }

// String returns "summon_link_cycle".
func (SummonLinkCycleRejection) String() string { return valueSummonReasonLinkCycle }

// String returns "level_below_spell".
func (LevelBelowSpellRejection) String() string { return valueSummonReasonLevelBelowSpell }

// String returns "not_night".
func (NotNightRejection) String() string { return valueSummonReasonNotNight }

// String returns "form_not_eligible".
func (FormNotEligibleRejection) String() string { return valueSummonReasonFormNotEligible }

// String returns "steed_not_disappeared".
func (SteedNotDisappearedRejection) String() string { return valueSummonReasonSteedNotDisappeared }

// String returns "target_not_eligible".
func (TargetNotEligibleRejection) String() string { return valueSummonReasonTargetNotEligible }

// String returns "object_not_eligible".
func (ObjectNotEligibleRejection) String() string { return valueSummonReasonObjectNotEligible }

// String returns "original_not_eligible".
func (OriginalNotEligibleRejection) String() string { return valueSummonReasonOriginalNotEligible }

// String returns "duplicate_target".
func (DuplicateTargetRejection) String() string { return valueSummonReasonDuplicateTarget }

// String returns "too_many_creatures".
func (TooManyCreaturesRejection) String() string { return valueSummonReasonTooManyCreatures }

// String returns "composition_not_allowed".
func (CompositionNotAllowedRejection) String() string { return valueSummonReasonCompositionNotAllowed }

// String returns "control_lapsed".
func (ControlLapsedRejection) String() string { return valueSummonReasonControlLapsed }

// String returns "nothing_to_animate".
func (NothingToAnimateRejection) String() string { return valueSummonReasonNothingToAnimate }

// String returns "nothing_to_reassert".
func (NothingToReassertRejection) String() string { return valueSummonReasonNothingToReassert }

// String returns "resistance_not_offered".
func (ResistanceNotOfferedRejection) String() string { return valueSummonReasonResistanceNotOffered }

// String returns "space_unavailable".
func (SpaceUnavailableRejection) String() string { return valueSummonReasonSpaceUnavailable }

// String returns "limit_exceeded".
func (SummonLimitExceededRejection) String() string { return valueResultLimitExceeded }

// String returns "command_not_your_creature".
func (CommandNotYourCreatureRejection) String() string { return valueSummonReasonCommandNotYours }

// String returns "command_out_of_range".
func (CommandOutOfRangeRejection) String() string { return valueSummonReasonCommandOutOfRange }

// String returns "cannot_give_verbal_command".
func (CannotGiveVerbalCommandRejection) String() string { return valueSummonReasonCannotGiveVerbal }

// String returns "familiar_cannot_attack".
func (FamiliarCannotAttackRejection) String() string { return valueSummonReasonFamiliarCannotAttack }

// String returns "familiar_already_dismissed".
func (FamiliarAlreadyDismissedRejection) String() string { return valueSummonReasonAlreadyDismissed }

// String returns "familiar_not_dismissed".
func (FamiliarNotDismissedRejection) String() string { return valueSummonReasonNotDismissed }

// String returns "senses_not_sharable".
func (SensesNotSharableRejection) String() string { return valueSummonReasonSensesNotSharable }

// String returns "touch_not_deliverable".
func (TouchNotDeliverableRejection) String() string { return valueSummonReasonTouchNotDeliverable }

// String returns "simulacrum_too_far".
func (SimulacrumTooFarRejection) String() string { return valueSummonReasonSimulacrumTooFar }

// String returns "simulacrum_healing_refused".
func (SimulacrumHealingRefusedRejection) String() string { return valueSummonReasonHealingRefused }

// String returns "simulacrum_repair_amount".
func (SimulacrumRepairAmountRejection) String() string { return valueSummonReasonRepairAmount }

// String returns "simulacrum_repair_needs_long_rest".
func (SimulacrumRepairNeedsLongRestRejection) String() string {
	return valueSummonReasonRepairNeedsRest
}

// String returns "life_bond_out_of_reach".
func (LifeBondOutOfReachRejection) String() string { return valueSummonReasonLifeBondOutOfReach }

// String returns "telepathy_only_with_summoner".
func (TelepathyOnlyWithSummonerRejection) String() string {
	return valueSummonReasonTelepathyOnlySummoner
}

// maxSummonsPerSummoner is the cap of plan P10 on the creatures one summoner has at once.
const maxSummonsPerSummoner = 32

// summonRejectionClass is the result class of r: the limit is invalid, every other rejection a refusal.
func summonRejectionClass(r SummonRejection) combatResultClass {
	if _, over := r.(SummonLimitExceededRejection); over {
		return resultInvalid
	}
	return resultRefused
}
