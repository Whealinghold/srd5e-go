package combat

import "slices"

// RosterStep is the closed set of results of a tracker roster operation (combatant-roster-changes.md
// FR-039). Variants: RosterStepApplied, RosterStepRejected.
type RosterStep interface{ isRosterStep() }

// RosterStepApplied: the operation took effect. It carries the new tracker and the subject's
// standing: its id and its count after the operation. The operation emits no TurnEvent.
type RosterStepApplied struct {
	tracker  TurnTracker
	standing InitiativeStanding
}

// RosterStepRejected: the operation was refused; nothing was applied. Reason is
// RosterUnknownCombatantRejection or RosterDuplicateCombatantRejection.
type RosterStepRejected struct{ Reason RosterRejection }

func (RosterStepApplied) isRosterStep()  {}
func (RosterStepRejected) isRosterStep() {}

// Tracker returns the new tracker.
func (s RosterStepApplied) Tracker() TurnTracker { return s.tracker }

// Standing returns the subject's id and its count after the operation.
func (s RosterStepApplied) Standing() InitiativeStanding { return s.standing }

// The roster operations are pure, synchronous and silent, emit no TurnEvent and leave the receiver
// unchanged (FR-039). They return an error matching srderr.ErrInvalidInput only for a zero tracker, a
// blank id, a size that is not defined or a position that is not valid; a refusal is a
// RosterStepRejected. They never change the round, the active combatant, its resources, or
// another entry's Reaction, Dodge, readied action or speeds (FR-005). They do not refuse the
// active combatant: that is the Encounter's decision (FR-040).

// InsertCombatant adds the combatant of the profile at the position (SRD 5.2.1, "Initiative
// Order", p. 13). The joiner starts with its Reaction available, not Dodging and with no readied
// action (FR-007). It returns RosterDuplicateCombatantRejection for an id that is in the
// tracker and RosterUnknownCombatantRejection for an anchor that is not.
func (t TurnTracker) InsertCombatant(p CombatantProfile, pos RosterPosition) (RosterStep, error) {
	if t.IsZero() || p.ID.IsZero() || !p.Size.Valid() || !validRosterPosition(pos) {
		return nil, invalidInput("an insert needs a tracker, a profile and a position")
	}
	if _, dup := t.profiles[p.ID]; dup {
		return RosterStepRejected{RosterDuplicateCombatantRejection{p.ID}}, nil
	}
	i, count, rej := placeIn(t.order, pos)
	if rej != nil {
		return RosterStepRejected{rej}, nil
	}
	n := t.clone()
	standing := InitiativeStanding{p.ID, count}
	n.order = slices.Insert(n.order, i, standing)
	if i <= t.active {
		n.active++
	}
	n.profiles[p.ID] = p
	return RosterStepApplied{n, standing}, nil
}

// RemoveCombatant drops the combatant's entry, profile, spent Reaction, Dodge, readied action and
// feat uses (FR-016). The index of the active combatant follows it: after a removal before it the
// active combatant is the same. Removing the active entry leaves the next entry active, without a
// turn start; the Encounter refuses that first (FR-019). Removing the last entry leaves the zero
// tracker. It returns RosterUnknownCombatantRejection for an id that is not in the tracker.
func (t TurnTracker) RemoveCombatant(id CombatantID) (RosterStep, error) {
	if t.IsZero() || id.IsZero() {
		return nil, invalidInput("a removal needs a tracker and an id")
	}
	i := t.orderIndex(id)
	if i < 0 {
		return RosterStepRejected{RosterUnknownCombatantRejection{id}}, nil
	}
	n := t.clone()
	standing := t.order[i]
	n.order = slices.Delete(n.order, i, i+1)
	switch {
	case i < t.active:
		n.active--
	case i == t.active && n.active >= len(n.order):
		n.active = 0
	}
	delete(n.profiles, id)
	delete(n.spent, id)
	delete(n.dodging, id)
	delete(n.readied, id)
	for k := range n.feats {
		if k.id == id {
			delete(n.feats, k)
		}
	}
	return RosterStepApplied{n, standing}, nil
}

// MoveCombatant moves the combatant's entry to the position and sets its count accordingly (SRD
// 5.2.1, "Controlling a Mount", p. 16). The place is computed on the order without the subject, so
// it never ties with itself (FR-003). Its Reaction, Dodge, readied action and speeds are kept
// (FR-030). It returns RosterUnknownCombatantRejection for a subject, then an anchor, that is not
// in the tracker, and an error matching srderr.ErrInvalidInput for an anchor that is the subject.
func (t TurnTracker) MoveCombatant(id CombatantID, pos RosterPosition) (RosterStep, error) {
	if t.IsZero() || id.IsZero() || !validRosterPosition(pos) {
		return nil, invalidInput("a move needs a tracker, an id and a position")
	}
	if a, ok := pos.(RosterAfterCombatant); ok && a.Anchor == id {
		return nil, invalidInput("a combatant cannot be placed after itself")
	}
	i := t.orderIndex(id)
	if i < 0 {
		return RosterStepRejected{RosterUnknownCombatantRejection{id}}, nil
	}
	rest := slices.Delete(slices.Clone(t.order), i, i+1)
	at, count, rej := placeIn(rest, pos)
	if rej != nil {
		return RosterStepRejected{rej}, nil
	}
	n := t.clone()
	standing := InitiativeStanding{id, count}
	n.order = slices.Insert(rest, at, standing)
	active := t.order[t.active].Combatant
	n.active = n.orderIndex(active)
	return RosterStepApplied{n, standing}, nil
}

// orderIndex returns the position of id in the order, or -1.
func (t TurnTracker) orderIndex(id CombatantID) int {
	return slices.IndexFunc(t.order, func(s InitiativeStanding) bool { return s.Combatant == id })
}
