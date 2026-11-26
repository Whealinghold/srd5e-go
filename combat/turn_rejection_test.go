package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/initiative-and-actions.md FR-100, FR-126 - each of the 20 reasons has the snake_case name of FR-100, the names are unique, and only the unknown combatant is class invalid.
// Path: happy.
func TestTurnRejectionNames(t *testing.T) {
	t.Parallel()
	reasons := map[combat.TurnRejection]string{
		combat.TurnUnknownCombatantRejection{}:               "unknown_combatant",
		combat.TurnIncapacitatedRejection{}:                  "incapacitated",
		combat.TurnNotActiveCombatantRejection{}:             "not_active_combatant",
		combat.TurnActionAlreadyTakenRejection{}:             "action_already_taken",
		combat.TurnNoBonusActionGrantedRejection{}:           "no_bonus_action_granted",
		combat.TurnBonusActionAlreadyTakenRejection{}:        "bonus_action_already_taken",
		combat.TurnReactionAlreadyTakenRejection{}:           "reaction_already_taken",
		combat.TurnProvokerDisengagedRejection{}:             "provoker_disengaged",
		combat.TurnProvokerNotSeenRejection{}:                "provoker_not_seen",
		combat.TurnNoMatchingReadiedActionRejection{}:        "no_matching_readied_action",
		combat.TurnNeedsUtilizeActionRejection{}:             "needs_utilize_action",
		combat.TurnSpeedUnavailableRejection{}:               "speed_unavailable",
		combat.TurnNoMovementRemainingRejection{}:            "no_movement_remaining",
		combat.TurnInsufficientMovementRejection{}:           "insufficient_movement",
		combat.TurnBlockedByCreatureRejection{}:              "blocked_by_creature",
		combat.TurnEndsInOccupiedSpaceRejection{}:            "ends_in_occupied_space",
		combat.TurnSpeedIsZeroRejection{}:                    "speed_is_zero",
		combat.TurnAdditionalActionAlreadyGrantedRejection{}: "additional_action_already_granted",
		combat.TurnFeatUseLimitedRejection{}:                 "feat_use_limited",
		combat.TurnAllyNotWillingRejection{}:                 "ally_not_willing",
	}
	if len(reasons) != 20 {
		t.Fatalf("%d reasons, want 20", len(reasons))
	}
	for r, want := range reasons {
		if r.String() != want {
			t.Errorf("%T = %q, want %q", r, r.String(), want)
		}
	}
}
