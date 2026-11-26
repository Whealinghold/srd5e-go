package combat

// RosterEvent is the closed family of events a roster change produces (combatant-roster-changes.md
// FR-033). Each is an immutable past-tense value comparable with ==, with a constant snake_case Kind
// and a Combatants list of exactly the combatant that joined, left or moved. None is an envelope.
// Variants: CombatantJoined, CombatantRemoved, CombatantRepositioned. Each embeds rosterEventBase.
// No function of this package returns a CombatantID, so the entry before a combatant is read
// from PrecededBy, a list of none or one id.
type RosterEvent interface {
	Event
	rosterEvent()
}

// rosterEventBase carries the encounter and the round every roster event happened in and puts the
// event in the Event and RosterEvent families.
type rosterEventBase struct {
	id    EncounterID
	round int
}

func (rosterEventBase) event()       {}
func (rosterEventBase) rosterEvent() {}

// EncounterID returns the id of the encounter.
func (b rosterEventBase) EncounterID() EncounterID { return b.id }

// Round returns the round the event happened in, 1 or more.
func (b rosterEventBase) Round() int { return b.round }

// precededByList is the accessor form of a precededBy standing: none for the first entry, else one id.
func precededByList(s InitiativeStanding) []CombatantID {
	if s.Combatant.IsZero() {
		return []CombatantID{}
	}
	return one(s.Combatant)
}

// CombatantJoined: a combatant joined the encounter after it started. Kind combatant_joined.
type CombatantJoined struct {
	rosterEventBase
	combatant CombatantID
	side      Side
	reason    JoinReason
	count     InitiativeCount
	before    InitiativeStanding
}

// Kind returns "combatant_joined".
func (CombatantJoined) Kind() string { return "combatant_joined" }

// Combatants returns the combatant that joined.
func (e CombatantJoined) Combatants() []CombatantID { return one(e.combatant) }

// Side returns the side the combatant joined.
func (e CombatantJoined) Side() Side { return e.side }

// Reason returns why the combatant joined.
func (e CombatantJoined) Reason() JoinReason { return e.reason }

// Count returns the combatant's Initiative count after the join.
func (e CombatantJoined) Count() InitiativeCount { return e.count }

// PrecededBy returns the entry directly before the combatant in the order: none when it is first.
func (e CombatantJoined) PrecededBy() []CombatantID { return precededByList(e.before) }

// CombatantRemoved: a combatant left the encounter. Kind combatant_removed.
type CombatantRemoved struct {
	rosterEventBase
	combatant CombatantID
	side      Side
	cause     RemovalCause
}

// Kind returns "combatant_removed".
func (CombatantRemoved) Kind() string { return "combatant_removed" }

// Combatants returns the combatant that left.
func (e CombatantRemoved) Combatants() []CombatantID { return one(e.combatant) }

// Side returns the side the combatant fought on.
func (e CombatantRemoved) Side() Side { return e.side }

// Cause returns why the combatant left.
func (e CombatantRemoved) Cause() RemovalCause { return e.cause }

// CombatantRepositioned: a combatant took another place in the order. It is produced even when the
// order does not change; then the new and previous values are equal. Kind combatant_repositioned.
type CombatantRepositioned struct {
	rosterEventBase
	combatant      CombatantID
	count          InitiativeCount
	before         InitiativeStanding
	previousCount  InitiativeCount
	previousBefore InitiativeStanding
}

// Kind returns "combatant_repositioned".
func (CombatantRepositioned) Kind() string { return "combatant_repositioned" }

// Combatants returns the combatant that moved.
func (e CombatantRepositioned) Combatants() []CombatantID { return one(e.combatant) }

// Count returns the combatant's Initiative count after the move.
func (e CombatantRepositioned) Count() InitiativeCount { return e.count }

// PrecededBy returns the entry directly before the combatant after the move: none when it is first.
func (e CombatantRepositioned) PrecededBy() []CombatantID { return precededByList(e.before) }

// PreviousCount returns the combatant's count before the move.
func (e CombatantRepositioned) PreviousCount() InitiativeCount { return e.previousCount }

// PreviousPrecededBy returns the entry directly before the combatant before the move: none when it was first.
func (e CombatantRepositioned) PreviousPrecededBy() []CombatantID {
	return precededByList(e.previousBefore)
}
