package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// rosterTransition is the closed domain-level outcome of a roster operation of the Encounter; each use
// case maps it to its own result (combatant-roster-changes.md FR-036). It is not exported. Variants:
// rosterApplied, rosterRejected.
type rosterTransition interface {
	isRosterTransition()
	// Encounter returns the new encounter, or the unchanged one for a rejection.
	Encounter() Encounter
	// Events returns a copy of the events in the order they happened; none for a rejection.
	Events() []Event
	// Rejection returns why the operation was refused, or nil when it was applied.
	Rejection() RosterRejection
}

// rosterApplied: the operation took effect.
type rosterApplied struct {
	enc    Encounter
	events []Event
}

// rosterRejected: the operation was refused; the encounter is unchanged.
type rosterRejected struct {
	enc       Encounter
	rejection RosterRejection
}

func (rosterApplied) isRosterTransition()  {}
func (rosterRejected) isRosterTransition() {}

func (t rosterApplied) Encounter() Encounter  { return t.enc }
func (t rosterRejected) Encounter() Encounter { return t.enc }

func (t rosterApplied) Events() []Event { return slices.Clone(t.events) }
func (rosterRejected) Events() []Event  { return nil }

func (rosterApplied) Rejection() RosterRejection    { return nil }
func (t rosterRejected) Rejection() RosterRejection { return t.rejection }

// rosterReject is the refusal of a roster operation: the encounter, unchanged.
func (e Encounter) rosterReject(r RosterRejection) rosterTransition { return rosterRejected{e, r} }

// inOrder returns the combatants of pool in the order of the standings.
func inOrder(order []InitiativeStanding, pool []Combatant) []Combatant {
	byID := make(map[CombatantID]Combatant, len(pool))
	for _, c := range pool {
		byID[c.ref.ID] = c
	}
	out := make([]Combatant, len(order))
	for i, s := range order {
		out[i] = byID[s.Combatant]
	}
	return out
}

// appliedStep returns the applied step of a tracker operation the encounter has already checked, so
// that a refusal cannot happen.
func appliedStep(s RosterStep) RosterStepApplied {
	switch v := s.(type) {
	case RosterStepApplied:
		return v
	default:
		panic(unreachableError("roster step", s))
	}
}

// AddCombatant adds the combatant to an encounter that has started, at the position, with the profile
// the tracker keeps for it (SRD 5.2.1, "Initiative Order", p. 13; combatant-roster-changes.md
// FR-008). The combatants list stays in the tracker's order, the status, the round and the active
// combatant are unchanged, and the one event is CombatantJoined. A joiner of any life state or
// condition is accepted unchanged.
//
// It returns, first that applies, RosterEncounterEndedRejection, RosterDuplicateCombatantRejection (the id
// is a combatant already), RosterUnknownCombatantRejection (the anchor of RosterAfterCombatant is not a
// combatant, a dead one is) and RosterLimitExceededRejection (the encounter holds 256 combatants). It returns
// an error matching srderr.ErrInvalidInput for an invalid encounter, a zero Combatant, a profile whose
// id is not the combatant's or whose size is not defined, a position that is not valid or a reason
// that is not defined. It never panics.
func (e Encounter) AddCombatant(c Combatant, p CombatantProfile, pos RosterPosition, reason JoinReason) (rosterTransition, error) {
	if e.IsZero() || c.IsZero() || c.ref.Kind == nil || !c.side.Valid() || c.hp.IsZero() ||
		p.ID != c.ref.ID || !p.Size.Valid() || !validRosterPosition(pos) || !reason.Valid() {
		return nil, invalidInput("an add needs a valid encounter, combatant, profile, position and reason")
	}
	id := c.ref.ID
	anchor, anchored := pos.(RosterAfterCombatant)
	switch {
	case e.ended():
		return e.rosterReject(RosterEncounterEndedRejection{}), nil
	case e.index(id) >= 0:
		return e.rosterReject(RosterDuplicateCombatantRejection{id}), nil
	case anchored && e.index(anchor.Anchor) < 0:
		return e.rosterReject(RosterUnknownCombatantRejection{anchor.Anchor}), nil
	case len(e.combatants) >= maxCombatantsPerEncounter:
		return e.rosterReject(RosterLimitExceededRejection{}), nil
	}
	step, err := e.tracker.InsertCombatant(p, pos)
	if err != nil {
		return nil, err
	}
	s := appliedStep(step) // the refusals of the tracker were ruled out above
	n := e
	n.tracker = s.tracker
	n.combatants = inOrder(s.tracker.order, append(slices.Clone(e.combatants), c))
	joined := CombatantJoined{rosterEventBase{e.id, e.tracker.round}, id, c.side, reason, s.standing.Count, precededBy(s.tracker.order, id)}
	return rosterApplied{n, []Event{joined}}, nil
}

// RemoveCombatant removes the combatant from an encounter that has started (SRD 5.2.1, "Ending
// Combat", p. 14; combatant-roster-changes.md FR-016). Every grapple that involves the leaver ends
// first, with GrappleRules.EndOnSet and the cause OutOfRange: the leaver's own Grappled, then, in turn
// order, each remaining combatant whose Grappled names the leaver, each wrapping ConditionRemoved and
// GrappleEnded (FR-021). Every other condition is left unchanged. The events are those endings then
// CombatantRemoved. A combatant of any life state may leave, and no Died or Hit Point event is produced.
// Effects, readied actions of others and containments are not touched (FR-048, FR-049).
//
// It returns, first that applies, RosterEncounterEndedRejection, RosterUnknownCombatantRejection,
// RosterCombatantIsActiveRejection, RosterWouldEmptySideRejection (no combatant would be left on its side; the
// encounter does not end by itself) and RosterLimitExceededRejection (the call would produce more than 4096
// events). It returns an error matching srderr.ErrInvalidInput for an invalid encounter, a blank id or
// a cause that is not defined. It never panics.
func (e Encounter) RemoveCombatant(id CombatantID, cause RemovalCause) (rosterTransition, error) {
	if e.IsZero() || id.IsZero() || !cause.Valid() {
		return nil, invalidInput("a removal needs a valid encounter, an id and a cause")
	}
	i := e.index(id)
	switch {
	case e.ended():
		return e.rosterReject(RosterEncounterEndedRejection{}), nil
	case i < 0:
		return e.rosterReject(RosterUnknownCombatantRejection{id}), nil
	case e.tracker.order[e.tracker.active].Combatant == id:
		return e.rosterReject(RosterCombatantIsActiveRejection{id}), nil
	}
	side := e.combatants[i].side
	if !slices.ContainsFunc(e.combatants, func(c Combatant) bool { return c.side == side && c.ref.ID != id }) {
		return e.rosterReject(RosterWouldEmptySideRejection{side}), nil
	}
	cs := slices.Clone(e.combatants)
	var events []Event
	if _, held := cs[i].conditions.slots[content.ConditionGrappled].(Grappled); held {
		events = e.endDeparting(cs, i, events)
	}
	for j := range cs {
		if g, held := cs[j].conditions.slots[content.ConditionGrappled].(Grappled); j != i && held && g.Grappler == id {
			events = e.endDeparting(cs, j, events)
		}
	}
	if len(events)+1 > maxEventsPerCall {
		return e.rosterReject(RosterLimitExceededRejection{}), nil
	}
	step, err := e.tracker.RemoveCombatant(id)
	if err != nil {
		return nil, err
	}
	s := appliedStep(step)
	n := e
	n.tracker = s.tracker
	n.combatants = inOrder(s.tracker.order, slices.Delete(cs, i, i+1))
	events = append(events, CombatantRemoved{rosterEventBase{e.id, e.tracker.round}, id, side, cause})
	return rosterApplied{n, events}, nil
}

// endDeparting ends the grapple holding cs[j] because its grappler or it itself left the encounter
// (SRD 5.2.1, "Ending a Grapple", p. 182; interpretation: a creature that left the battle is beyond
// any grapple's reach, combatant-roster-changes.md FR-021): Grappled is removed from cs[j] through
// GrappleRules.EndOnSet, never ConditionSet.Remove, and the wrapped events are appended.
func (e Encounter) endDeparting(cs []Combatant, j int, events []Event) []Event {
	r, ok := (GrappleRules{}).EndOnSet(cs[j].conditions, cs[j].ref.ID, GrappleEndCauseOutOfRange).(GrappleEndOnSetApplied)
	if !ok { // unreachable: Grappled was read from the same set
		return events
	}
	for _, ev := range r.events {
		switch v := ev.(type) {
		case ConditionEvent:
			events = append(events, ConditionEventOccurred{e.base(), v})
		case GrappleEvent:
			events = append(events, GrappleEventOccurred{e.base(), v})
		}
	}
	cs[j].conditions = r.conditions
	return events
}

// RepositionCombatant moves the combatant's entry to the position and sets its count (SRD 5.2.1,
// "Controlling a Mount", p. 16; combatant-roster-changes.md FR-028). The combatants list follows the new
// order; the combatant and every other entry are unchanged, the Reaction, Dodge, readied action and
// speeds of the subject included. A combatant moved from before the active one to after it may take a
// second turn this round; the engine follows the order only (FR-031). The one event is
// CombatantRepositioned, produced even when the order does not change.
//
// It returns, first that applies, RosterEncounterEndedRejection, RosterUnknownCombatantRejection (the
// subject, then the anchor) and RosterCombatantIsActiveRejection (the subject is active; an active anchor is
// allowed). It returns an error matching srderr.ErrInvalidInput for an invalid encounter, a blank id, a
// position that is not valid or an anchor that is the subject. It never panics.
func (e Encounter) RepositionCombatant(id CombatantID, pos RosterPosition) (rosterTransition, error) {
	anchor, anchored := pos.(RosterAfterCombatant)
	if e.IsZero() || id.IsZero() || !validRosterPosition(pos) || (anchored && anchor.Anchor == id) {
		return nil, invalidInput("a reposition needs a valid encounter, an id and a position other than after itself")
	}
	switch {
	case e.ended():
		return e.rosterReject(RosterEncounterEndedRejection{}), nil
	case e.index(id) < 0:
		return e.rosterReject(RosterUnknownCombatantRejection{id}), nil
	case anchored && e.index(anchor.Anchor) < 0:
		return e.rosterReject(RosterUnknownCombatantRejection{anchor.Anchor}), nil
	case e.tracker.order[e.tracker.active].Combatant == id:
		return e.rosterReject(RosterCombatantIsActiveRejection{id}), nil
	}
	step, err := e.tracker.MoveCombatant(id, pos)
	if err != nil {
		return nil, err
	}
	s := appliedStep(step)
	previous := e.tracker.order[e.tracker.orderIndex(id)]
	n := e
	n.tracker = s.tracker
	n.combatants = inOrder(s.tracker.order, e.combatants)
	moved := CombatantRepositioned{rosterEventBase{e.id, e.tracker.round}, id, s.standing.Count, precededBy(s.tracker.order, id),
		previous.Count, precededBy(e.tracker.order, id)}
	return rosterApplied{n, []Event{moved}}, nil
}
