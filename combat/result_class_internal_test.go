package combat

import "testing"

// Requirement: doc/specs/combat-foundations.md FR-031 - every srd5e.combat.result value a rejection family produces has exactly one result class across the families (roster, turn and encounter rejections), so a shared value such as unknown_combatant is never applied in one use case and invalid in another; every value has a class (not the zero one).
// Path: unhappy (rule interaction).
func TestRejectionResultValuesHaveOneClass(t *testing.T) {
	t.Parallel()
	type named struct {
		name  string
		class combatResultClass
	}
	var all []named
	for _, r := range []RosterRejection{RosterEncounterEndedRejection{}, RosterUnknownCombatantRejection{}, RosterDuplicateCombatantRejection{},
		RosterCombatantIsActiveRejection{}, RosterWouldEmptySideRejection{}, RosterLimitExceededRejection{}} {
		all = append(all, named{r.String(), rosterRejectionClass(r)})
	}
	for _, r := range []TurnRejection{TurnUnknownCombatantRejection{}, TurnIncapacitatedRejection{}, TurnNotActiveCombatantRejection{},
		TurnActionAlreadyTakenRejection{}, TurnNoBonusActionGrantedRejection{}, TurnBonusActionAlreadyTakenRejection{},
		TurnReactionAlreadyTakenRejection{}, TurnProvokerDisengagedRejection{}, TurnProvokerNotSeenRejection{},
		TurnNoMatchingReadiedActionRejection{}, TurnNeedsUtilizeActionRejection{}, TurnSpeedUnavailableRejection{},
		TurnNoMovementRemainingRejection{}, TurnInsufficientMovementRejection{}, TurnBlockedByCreatureRejection{},
		TurnEndsInOccupiedSpaceRejection{}, TurnSpeedIsZeroRejection{}, TurnAdditionalActionAlreadyGrantedRejection{},
		TurnFeatUseLimitedRejection{}, TurnAllyNotWillingRejection{}} {
		all = append(all, named{r.String(), turnRejectionClass(r)})
	}
	for _, r := range []EncounterRejection{EncounterAlreadyEndedRejection{}, UnknownCombatantRejection{}, CombatantIsDeadRejection{},
		AlreadyHasConditionRejection{}, ImmuneToConditionRejection{}, DoesNotHaveConditionRejection{}, ConditionImpliedByRejection{},
		HitPointRuleRejection{Reason: NotDying}, HitPointRuleRejection{Reason: NotStable}, TurnOrderChangedRejection{},
		DeathSavingThrowAlreadyMadeRejection{}} {
		all = append(all, named{r.String(), rejectionClass(r)})
	}
	classes := map[string]combatResultClass{}
	for _, n := range all {
		if n.class != resultApplied && n.class != resultRefused && n.class != resultInvalid {
			t.Errorf("%q has no result class", n.name)
		}
		if prev, seen := classes[n.name]; seen && prev != n.class {
			t.Errorf("%q is classed %v and %v", n.name, prev, n.class)
		}
		classes[n.name] = n.class
	}
	if got := classes[valueResultUnknownCombatant]; got != resultInvalid {
		t.Errorf("%q is classed %v, want invalid", valueResultUnknownCombatant, got)
	}
}
