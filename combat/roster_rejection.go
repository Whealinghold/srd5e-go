package combat

// RosterRejection is why a roster operation was refused: the field of every ...Rejected result and
// of RosterStepRejected (combatant-roster-changes.md FR-009, FR-019, FR-020, FR-029). String is
// the snake_case value of srd5e.combat.result. A duplicate id, an unknown combatant and a
// roster over its cap are host input the engine cannot use (class invalid); the others are
// refusals. Variants: RosterEncounterEndedRejection, RosterUnknownCombatantRejection,
// RosterDuplicateCombatantRejection, RosterCombatantIsActiveRejection,
// RosterWouldEmptySideRejection, RosterLimitExceededRejection.
type RosterRejection interface {
	isRosterRejection()
	String() string
}

// RosterEncounterEndedRejection: the encounter ended, so no roster change is accepted.
type RosterEncounterEndedRejection struct{}

// RosterUnknownCombatantRejection: ID is not a combatant of the encounter (the subject, or the
// anchor of RosterAfterCombatant).
type RosterUnknownCombatantRejection struct{ ID CombatantID }

// RosterDuplicateCombatantRejection: ID is already a combatant of the encounter.
type RosterDuplicateCombatantRejection struct{ ID CombatantID }

// RosterCombatantIsActiveRejection: ID is the active combatant, which cannot leave or move; the host
// ends its turn first with AdvanceCombatTurn.
type RosterCombatantIsActiveRejection struct{ ID CombatantID }

// RosterWouldEmptySideRejection: the removal would leave no combatant on Side. The encounter does
// not end by itself; the host calls end with EndReasonSideDefeated.
type RosterWouldEmptySideRejection struct{ Side Side }

// RosterLimitExceededRejection: the call would leave more than 256 combatants in the encounter or
// produce more than 4096 events (plan P10). It changes nothing.
type RosterLimitExceededRejection struct{}

func (RosterEncounterEndedRejection) isRosterRejection()     {}
func (RosterUnknownCombatantRejection) isRosterRejection()   {}
func (RosterDuplicateCombatantRejection) isRosterRejection() {}
func (RosterCombatantIsActiveRejection) isRosterRejection()  {}
func (RosterWouldEmptySideRejection) isRosterRejection()     {}
func (RosterLimitExceededRejection) isRosterRejection()      {}

// String returns "encounter_already_ended".
func (RosterEncounterEndedRejection) String() string { return valueResultRosterEncounterEnded }

// String returns "unknown_combatant".
func (RosterUnknownCombatantRejection) String() string { return valueResultUnknownCombatant }

// String returns "duplicate_combatant".
func (RosterDuplicateCombatantRejection) String() string { return valueResultDuplicateCombatant }

// String returns "combatant_is_active".
func (RosterCombatantIsActiveRejection) String() string { return valueResultCombatantIsActive }

// String returns "would_empty_side".
func (RosterWouldEmptySideRejection) String() string { return valueResultWouldEmptySide }

// String returns "limit_exceeded".
func (RosterLimitExceededRejection) String() string { return valueResultLimitExceeded }

// maxCombatantsPerEncounter is the cap of plan P10 on the combatants of one encounter, applied to an add.
const maxCombatantsPerEncounter = 256

// rosterRejectionClass is the result class of r: an unknown combatant, a duplicate id and a roster
// over its cap are invalid (FR-042); every other rejection is a refusal.
func rosterRejectionClass(r RosterRejection) combatResultClass {
	switch r.(type) {
	case RosterUnknownCombatantRejection, RosterDuplicateCombatantRejection, RosterLimitExceededRejection:
		return resultInvalid
	}
	return resultRefused
}
