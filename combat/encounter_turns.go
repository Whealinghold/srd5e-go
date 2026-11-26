package combat

import "slices"

// unknownIn returns the rejection for the smallest id (in code unit order, so the
// choice does not depend on how the command was built) that is not a combatant, and
// false when every id is one.
func (e Encounter) unknownIn(ids []CombatantID) (EncounterRejection, bool) {
	var worst CombatantID
	found := false
	for _, id := range ids {
		if e.index(id) < 0 && (!found || id < worst) {
			worst, found = id, true
		}
	}
	return UnknownCombatantRejection{worst}, found
}

// AdvanceTurn ends the active combatant's turn and starts the next (SRD 5.2.1, "The
// Order of Combat", p. 13) through TurnTracker.Advance: a dead, Unconscious or 0 Hit
// Point combatant is not skipped. The events are the tracker's, wrapped in order, then
// DyingCombatantTurnStarted when the combatant whose turn began is Dying. The Death
// Saving Throw marker of that combatant is cleared. It does not apply Prone for
// FellProneInSharedSpace; the host does.
//
// It returns UnknownCombatantRejection (the smallest id) for an update naming a
// combatant the encounter does not hold, before the tracker is called, and
// EncounterAlreadyEndedRejection for an ended encounter. It returns an error matching
// srderr.ErrInvalidInput for an invalid encounter or a size that is not defined.
func (e Encounter) AdvanceTurn(a TurnAdvance) (encounterTransition, error) {
	if !e.usable() {
		return nil, invalidInput("advance needs a valid encounter")
	}
	if e.ended() {
		return e.reject(EncounterAlreadyEndedRejection{}), nil
	}
	ids := make([]CombatantID, len(a.Updates))
	for i, u := range a.Updates {
		ids[i] = u.Combatant
	}
	if r, bad := e.unknownIn(ids); bad {
		return e.reject(r), nil
	}
	step, err := e.tracker.Advance(a)
	if err != nil {
		return nil, err
	}
	next := step.(TrackerStepApplied) // the only refusal is an unknown id, checked above
	n := e
	n.tracker = next.tracker
	i := n.index(n.tracker.order[n.tracker.active].Combatant)
	n.combatants = slices.Clone(e.combatants)
	n.combatants[i].dstMade = false
	events := append(n.wrapTurn(e.tracker.round, next.events), n.dyingTurnEvent(i)...)
	return n.accept(events), nil
}

// ApplyTurnChange replaces the tracker with one the host got from a tracker operation
// and wraps that step's events, in the order given, each in TurnEventOccurred with the
// round it happened in (SRD 5.2.1, "The Order of Combat", p. 13). It adds no event of
// its own and changes no combatant. An empty list is valid. It cannot tell whether the
// tracker succeeds the stored one: a stale tracker with an equal turn order is accepted.
//
// It returns, first that applies, EncounterAlreadyEndedRejection,
// UnknownCombatantRejection (an event naming a combatant the encounter does not hold:
// the smallest id) and TurnOrderChangedRejection (the tracker's TurnOrder is not the
// encounter's). It returns an error matching srderr.ErrInvalidInput for an invalid
// encounter, a tracker that is zero or not well formed (plan P14) or a nil event.
func (e Encounter) ApplyTurnChange(tracker TurnTracker, events []TurnEvent) (encounterTransition, error) {
	if !e.usable() || !tracker.wellFormed() || slices.Contains(events, nil) {
		return nil, invalidInput("a turn change needs a valid encounter, a tracker and events")
	}
	if e.ended() {
		return e.reject(EncounterAlreadyEndedRejection{}), nil
	}
	var named []CombatantID
	for _, ev := range events {
		named = append(named, ev.Combatants()...)
	}
	if r, bad := e.unknownIn(named); bad {
		return e.reject(r), nil
	}
	if !tracker.Order().Equal(e.tracker.Order()) {
		return e.reject(TurnOrderChangedRejection{}), nil
	}
	n := e
	n.tracker = tracker
	return n.accept(e.wrapTurn(e.tracker.round, events)), nil
}

// end ends the encounter for the reason; it does not verify a SideDefeated claim and
// the encounter never ends by itself (SRD 5.2.1, "Ending Combat", p. 14). It returns
// EncounterAlreadyEndedRejection for an ended encounter and an error matching
// srderr.ErrInvalidInput for an invalid encounter or reason.
func (e Encounter) end(reason EndReason) (encounterTransition, error) {
	if !e.usable() || !validEndReason(reason) {
		return nil, invalidInput("end needs a valid encounter and reason")
	}
	if e.ended() {
		return e.reject(EncounterAlreadyEndedRejection{}), nil
	}
	n := e
	n.status = EncounterEndedStatus{reason}
	ids := make([]CombatantID, len(e.combatants))
	for i, c := range e.combatants {
		ids[i] = c.ref.ID
	}
	return n.accept([]Event{EncounterEnded{e.base(), ids, reason}}), nil
}
