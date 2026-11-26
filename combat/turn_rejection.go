package combat

import "github.com/Whealinghold/srd5e-go/content"

// TurnRejection is the closed set of reasons a tracker operation or a turn use case
// refuses (initiative-and-actions.md FR-126). Each is a value type of its own;
// String returns the snake_case reason, the value of srd5e.combat.result. Only
// TurnUnknownCombatantRejection is host input the engine could not use (class
// invalid); the others are normal rule outcomes. Variants:
// TurnUnknownCombatantRejection, TurnIncapacitatedRejection,
// TurnNotActiveCombatantRejection, TurnActionAlreadyTakenRejection,
// TurnNoBonusActionGrantedRejection, TurnBonusActionAlreadyTakenRejection,
// TurnReactionAlreadyTakenRejection, TurnProvokerDisengagedRejection,
// TurnProvokerNotSeenRejection, TurnNoMatchingReadiedActionRejection,
// TurnNeedsUtilizeActionRejection, TurnSpeedUnavailableRejection,
// TurnNoMovementRemainingRejection, TurnInsufficientMovementRejection,
// TurnBlockedByCreatureRejection, TurnEndsInOccupiedSpaceRejection,
// TurnSpeedIsZeroRejection, TurnAdditionalActionAlreadyGrantedRejection,
// TurnFeatUseLimitedRejection, TurnAllyNotWillingRejection.
type TurnRejection interface {
	isTurnRejection()
	String() string
}

// TurnUnknownCombatantRejection: the id is not in the tracker or standings.
type TurnUnknownCombatantRejection struct{ ID CombatantID }

// TurnIncapacitatedRejection: the combatant cannot take actions, Bonus Actions or Reactions (SRD 5.2.1, "Incapacitated [Condition]", p. 184).
type TurnIncapacitatedRejection struct{}

// TurnNotActiveCombatantRejection: it is not the combatant's turn (SRD 5.2.1, "Your Turn", p. 13).
type TurnNotActiveCombatantRejection struct{}

// TurnActionAlreadyTakenRejection: the turn's action is spent, or a Ready is already held.
type TurnActionAlreadyTakenRejection struct{}

// TurnNoBonusActionGrantedRejection: no rule grants a Bonus Action (SRD 5.2.1, "Bonus Action", p. 177).
type TurnNoBonusActionGrantedRejection struct{}

// TurnBonusActionAlreadyTakenRejection: the turn's Bonus Action is spent.
type TurnBonusActionAlreadyTakenRejection struct{}

// TurnReactionAlreadyTakenRejection: the Reaction is spent until the combatant's own turn starts.
type TurnReactionAlreadyTakenRejection struct{}

// TurnProvokerDisengagedRejection: the provoker took Disengage this turn.
type TurnProvokerDisengagedRejection struct{}

// TurnProvokerNotSeenRejection: the observer cannot see the provoker.
type TurnProvokerNotSeenRejection struct{}

// TurnNoMatchingReadiedActionRejection: the reactor holds no readied action with the trigger.
type TurnNoMatchingReadiedActionRejection struct{ Trigger ReadyTriggerID }

// TurnNeedsUtilizeActionRejection: the free interaction is spent; a second needs the Utilize action.
type TurnNeedsUtilizeActionRejection struct{}

// TurnSpeedUnavailableRejection: the combatant lacks the speed.
type TurnSpeedUnavailableRejection struct{ Speed SpeedName }

// TurnNoMovementRemainingRejection: the governing speed has nothing left at the segment; MovementLeft may be negative.
type TurnNoMovementRemainingRejection struct{ SegmentIndex, MovementLeft int }

// TurnInsufficientMovementRejection: the segment costs ShortfallFeet more than is left.
type TurnInsufficientMovementRejection struct{ SegmentIndex, ShortfallFeet int }

// TurnBlockedByCreatureRejection: the segment passes through a creature the mover cannot pass.
type TurnBlockedByCreatureRejection struct{ SegmentIndex int }

// TurnEndsInOccupiedSpaceRejection: the move ends in a space another creature occupies.
type TurnEndsInOccupiedSpaceRejection struct{ SegmentIndex int }

// TurnSpeedIsZeroRejection: the walking Speed is 0.
type TurnSpeedIsZeroRejection struct{}

// TurnAdditionalActionAlreadyGrantedRejection: one additional action is granted a turn.
type TurnAdditionalActionAlreadyGrantedRejection struct{}

// TurnFeatUseLimitedRejection: the feat's use is already recorded in its window.
type TurnFeatUseLimitedRejection struct{ Feat content.FeatID }

// TurnAllyNotWillingRejection: the ally does not consent to the swap.
type TurnAllyNotWillingRejection struct{}

func (TurnUnknownCombatantRejection) isTurnRejection()               {}
func (TurnIncapacitatedRejection) isTurnRejection()                  {}
func (TurnNotActiveCombatantRejection) isTurnRejection()             {}
func (TurnActionAlreadyTakenRejection) isTurnRejection()             {}
func (TurnNoBonusActionGrantedRejection) isTurnRejection()           {}
func (TurnBonusActionAlreadyTakenRejection) isTurnRejection()        {}
func (TurnReactionAlreadyTakenRejection) isTurnRejection()           {}
func (TurnProvokerDisengagedRejection) isTurnRejection()             {}
func (TurnProvokerNotSeenRejection) isTurnRejection()                {}
func (TurnNoMatchingReadiedActionRejection) isTurnRejection()        {}
func (TurnNeedsUtilizeActionRejection) isTurnRejection()             {}
func (TurnSpeedUnavailableRejection) isTurnRejection()               {}
func (TurnNoMovementRemainingRejection) isTurnRejection()            {}
func (TurnInsufficientMovementRejection) isTurnRejection()           {}
func (TurnBlockedByCreatureRejection) isTurnRejection()              {}
func (TurnEndsInOccupiedSpaceRejection) isTurnRejection()            {}
func (TurnSpeedIsZeroRejection) isTurnRejection()                    {}
func (TurnAdditionalActionAlreadyGrantedRejection) isTurnRejection() {}
func (TurnFeatUseLimitedRejection) isTurnRejection()                 {}
func (TurnAllyNotWillingRejection) isTurnRejection()                 {}

// String returns "unknown_combatant".
func (TurnUnknownCombatantRejection) String() string { return "unknown_combatant" }

// String returns "incapacitated".
func (TurnIncapacitatedRejection) String() string { return "incapacitated" }

// String returns "not_active_combatant".
func (TurnNotActiveCombatantRejection) String() string { return "not_active_combatant" }

// String returns "action_already_taken".
func (TurnActionAlreadyTakenRejection) String() string { return "action_already_taken" }

// String returns "no_bonus_action_granted".
func (TurnNoBonusActionGrantedRejection) String() string { return "no_bonus_action_granted" }

// String returns "bonus_action_already_taken".
func (TurnBonusActionAlreadyTakenRejection) String() string { return "bonus_action_already_taken" }

// String returns "reaction_already_taken".
func (TurnReactionAlreadyTakenRejection) String() string { return "reaction_already_taken" }

// String returns "provoker_disengaged".
func (TurnProvokerDisengagedRejection) String() string { return "provoker_disengaged" }

// String returns "provoker_not_seen".
func (TurnProvokerNotSeenRejection) String() string { return "provoker_not_seen" }

// String returns "no_matching_readied_action".
func (TurnNoMatchingReadiedActionRejection) String() string { return "no_matching_readied_action" }

// String returns "needs_utilize_action".
func (TurnNeedsUtilizeActionRejection) String() string { return "needs_utilize_action" }

// String returns "speed_unavailable".
func (TurnSpeedUnavailableRejection) String() string { return "speed_unavailable" }

// String returns "no_movement_remaining".
func (TurnNoMovementRemainingRejection) String() string { return "no_movement_remaining" }

// String returns "insufficient_movement".
func (TurnInsufficientMovementRejection) String() string { return "insufficient_movement" }

// String returns "blocked_by_creature".
func (TurnBlockedByCreatureRejection) String() string { return "blocked_by_creature" }

// String returns "ends_in_occupied_space".
func (TurnEndsInOccupiedSpaceRejection) String() string { return "ends_in_occupied_space" }

// String returns "speed_is_zero".
func (TurnSpeedIsZeroRejection) String() string { return "speed_is_zero" }

// String returns "additional_action_already_granted".
func (TurnAdditionalActionAlreadyGrantedRejection) String() string {
	return "additional_action_already_granted"
}

// String returns "feat_use_limited".
func (TurnFeatUseLimitedRejection) String() string { return "feat_use_limited" }

// String returns "ally_not_willing".
func (TurnAllyNotWillingRejection) String() string { return "ally_not_willing" }

// turnRejectionClass classifies a rejection for telemetry: an unknown combatant is
// invalid, every other reason refused.
func turnRejectionClass(r TurnRejection) combatResultClass {
	if _, ok := r.(TurnUnknownCombatantRejection); ok {
		return resultInvalid
	}
	return resultRefused
}
