package combat

import "testing"

// Requirement: doc/specs/combat-action-resolution.md FR-099, FR-100, FR-106 - the eight operation names, their spans and the result values of the resolution stops are pinned to their catalogue spelling (renaming one breaks consumers' dashboards), each operation and span pair is a valid combatOperation, and the cap of plan P10 is 4096.
// Path: per case.
func TestResolutionSignalValuesArePinned(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got, want               string
	}{
		{"perform_attack", "FR-099", "happy", valueOperationPerformAttack, "perform_attack"},
		{"perform_attack span", "FR-099", "happy", spanPerformAttack, "srd5e.combat.perform_attack"},
		{"perform_unarmed_strike", "FR-099", "happy", valueOperationPerformUnarmedStrike, "perform_unarmed_strike"},
		{"perform_unarmed_strike span", "FR-099", "happy", spanPerformUnarmedStrike, "srd5e.combat.perform_unarmed_strike"},
		{"take_combatant_action", "FR-099", "happy", valueOperationTakeCombatantAction, "take_combatant_action"},
		{"take_combatant_action span", "FR-099", "happy", spanTakeCombatantAction, "srd5e.combat.take_combatant_action"},
		{"move_combatant", "FR-099", "happy", valueOperationMoveCombatant, "move_combatant"},
		{"move_combatant span", "FR-099", "happy", spanMoveCombatant, "srd5e.combat.move_combatant"},
		{"advance_combat_turn", "FR-099", "happy", valueOperationAdvanceCombatTurn, "advance_combat_turn"},
		{"advance_combat_turn span", "FR-099", "happy", spanAdvanceCombatTurn, "srd5e.combat.advance_combat_turn"},
		{"roll_combatant_death_saving_throw", "FR-099", "happy", valueOperationRollCombatantDeathSavingThrow, "roll_combatant_death_saving_throw"},
		{"roll_combatant_death_saving_throw span", "FR-099", "happy", spanRollCombatantDeathSavingThrow, "srd5e.combat.roll_combatant_death_saving_throw"},
		{"attempt_to_stabilize_combatant", "FR-099", "happy", valueOperationAttemptToStabilizeCombatant, "attempt_to_stabilize_combatant"},
		{"attempt_to_stabilize_combatant span", "FR-099", "happy", spanAttemptToStabilizeCombatant, "srd5e.combat.attempt_to_stabilize_combatant"},
		{"deal_damage_to_combatant", "FR-099", "happy", valueOperationDealDamageToCombatant, "deal_damage_to_combatant"},
		{"deal_damage_to_combatant span", "FR-099", "happy", spanDealDamageToCombatant, "srd5e.combat.deal_damage_to_combatant"},
		{"unknown_combatant", "FR-100", "unhappy (invalid input)", valueResultUnknownCombatant, "unknown_combatant"},
		{"ledger_refused", "FR-100", "unhappy (state)", valueResultLedgerRefused, "ledger_refused"},
		{"turn_refused", "FR-100", "unhappy (state)", valueResultTurnRefused, "turn_refused"},
		{"targeting_forbidden", "FR-100", "unhappy (rule interaction)", valueResultTargetingForbidden, "targeting_forbidden"},
		{"attack_not_possible", "FR-100", "unhappy (state)", valueResultAttackNotPossible, "attack_not_possible"},
		{"movement_restricted", "FR-100", "unhappy (rule interaction)", valueResultMovementRestricted, "movement_restricted"},
		{"posture_mismatch", "FR-100", "unhappy (state)", valueResultPostureMismatch, "posture_mismatch"},
		{"death_saving_throw_out_of_turn", "FR-100", "unhappy (state)", valueResultDeathSavingThrowOutOfTurn, "death_saving_throw_out_of_turn"},
		{"completed", "FR-100", "happy", valueResultCompleted, "completed"},
		{"limit_exceeded", "FR-100, plan P10", "unhappy (boundary)", valueResultLimitExceeded, "limit_exceeded"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("%s = %q, want %q [%s%s]", c.name, c.got, c.want, "doc/specs/combat-action-resolution.md ", c.requirement)
			}
		})
	}
	for _, pair := range [][2]string{
		{valueOperationPerformAttack, spanPerformAttack}, {valueOperationPerformUnarmedStrike, spanPerformUnarmedStrike},
		{valueOperationTakeCombatantAction, spanTakeCombatantAction}, {valueOperationMoveCombatant, spanMoveCombatant},
		{valueOperationAdvanceCombatTurn, spanAdvanceCombatTurn}, {valueOperationRollCombatantDeathSavingThrow, spanRollCombatantDeathSavingThrow},
		{valueOperationAttemptToStabilizeCombatant, spanAttemptToStabilizeCombatant}, {valueOperationDealDamageToCombatant, spanDealDamageToCombatant},
	} {
		if _, err := newCombatOperation(pair[0], pair[1]); err != nil {
			t.Errorf("operation %s with span %s is not valid: %v", pair[0], pair[1], err)
		}
	}
	if maxEventsPerCall != 4096 {
		t.Errorf("maxEventsPerCall = %d, want 4096 (plan P10)", maxEventsPerCall)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-100, FR-101 - a refusal is class refused (no error status) and a problem is class invalid, whatever the rejection or problem; an EncounterStateRejection is named for the encounter's own rejection.
// Path: unhappy (state).
func TestResolutionOutcomesAreClassifiedByVariant(t *testing.T) {
	t.Parallel()
	enc := Encounter{}
	for _, r := range []ResolutionRejection{
		EncounterStateRejection{EncounterAlreadyEndedRejection{}}, LedgerRejection{LedgerNoAttackActionInProgress{}}, TrackerRejection{TurnIncapacitatedRejection{}},
		TargetingForbiddenRejection{}, AttackNotPossibleRejection{AttackNotPossibleReasonTotalCover}, MovementRestrictedRejection{MovementRestrictionProneAllowsOnlyCrawl},
		PostureMismatchRejection{PostureMismatchNotProne}, DeathSavingThrowOutOfTurnRejection{},
	} {
		d := resolutionRejected{r, enc}.describe()
		if d.Class != resultRefused || d.Name != r.String() || d.Class.IsSpanError() || len(d.Events) != 0 {
			t.Errorf("%T described as %+v, want refused and named %q with no error status", r, d, r.String())
		}
	}
	for _, p := range []ResolutionProblem{UnknownCombatantProblem{ID: "x"}, LimitExceededProblem{}} {
		d := resolutionInvalid{p, enc}.describe()
		if d.Class != resultInvalid || d.Name != p.String() || !d.Class.IsSpanError() {
			t.Errorf("%T described as %+v, want invalid with error status", p, d)
		}
	}
	if d := (resolutionCompleted{enc, nil}).describe(); d.Class != resultApplied || d.Name != valueResultCompleted {
		t.Errorf("completed described as %+v", d)
	}
}
