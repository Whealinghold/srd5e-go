package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// withConditions commits a new ConditionSet and its events for the combatant at i; mark
// changes the combatant too when it is not nil.
func (e Encounter) withConditions(i int, set ConditionSet, events []ConditionEvent, mark func(*Combatant)) encounterTransition {
	c := e.combatants[i]
	c.conditions = set
	if mark != nil {
		mark(&c)
	}
	return e.replace(i, c).accept(e.wrapCondition(events))
}

// ApplyCondition applies a condition through ConditionSet.Apply to a living combatant
// (SRD 5.2.1, "Condition", p. 179): Exhaustion raises its level by 1, and level 6 kills
// the combatant through HitPointStatus.RecordExhaustionDeath, with the wrapped Died
// after the wrapped ConditionApplied. The encounter never ends a grapple by itself.
//
// It returns, first that applies, EncounterAlreadyEndedRejection,
// UnknownCombatantRejection, CombatantIsDeadRejection, ImmuneToConditionRejection and
// AlreadyHasConditionRejection. It returns an error matching srderr.ErrInvalidInput for
// an invalid encounter or id or a nil or invalid condition.
func (e Encounter) ApplyCondition(id CombatantID, condition Condition) (encounterTransition, error) {
	if !validCondition(condition) {
		return nil, invalidInput("apply condition needs a valid condition")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	set := e.combatants[i].conditions
	switch r := set.Apply(id, condition).(type) {
	case ConditionAdded:
		c := e.combatants[i]
		c.conditions = r.set
		events := e.wrapCondition(r.events)
		if (ConditionEffects{}).IsDeadFromExhaustion(r.set) {
			// The error is only for a zero subject or status, which target ruled out.
			death, _ := c.hp.RecordExhaustionDeath(id)
			a := death.(HitPointTransitionApplied)
			c.hp = a.status
			events = append(events, e.wrapHitPoint(a.events)...)
		}
		return e.replace(i, c).accept(events), nil
	case ConditionImmune:
		return e.reject(ImmuneToConditionRejection{id, condition.Kind()}), nil
	case ConditionAlreadyHeld:
		return e.reject(AlreadyHasConditionRejection{id, condition.Kind()}), nil
	default: // ExhaustionAtMaximum: the combatant is Dead, which is refused first
		return e.reject(CombatantIsDeadRejection{id}), nil
	}
}

// RemoveCondition removes a condition through ConditionSet.Remove from a living
// combatant (SRD 5.2.1, "Condition", p. 179). Removing Exhaustion lowers it by one level.
// Removing Unconscious leaves the combatant Prone and clears the knocked-out marker.
//
// It returns the rejections of ApplyCondition up to CombatantIsDeadRejection, then
// DoesNotHaveConditionRejection and ConditionImpliedByRejection (a kind held only
// because another condition implies it). It returns an error matching
// srderr.ErrInvalidInput for an invalid encounter or id or an invalid kind.
func (e Encounter) RemoveCondition(id CombatantID, kind content.ConditionKind) (encounterTransition, error) {
	if !kind.Valid() {
		return nil, invalidInput("remove condition needs a valid kind")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	switch r := e.combatants[i].conditions.Remove(id, kind).(type) {
	case ConditionCleared:
		return e.withConditions(i, r.set, r.events, func(c *Combatant) { c.knockedOut = c.knockedOut && kind != content.ConditionUnconscious }), nil
	case ExhaustionLowered:
		return e.withConditions(i, r.set, r.events, nil), nil
	case ConditionRemovalImpliedBy:
		return e.reject(ConditionImpliedByRejection{id, kind, r.ImpliedBy()}), nil
	default: // ConditionRemovalNotPresent
		return e.reject(DoesNotHaveConditionRejection{id, kind}), nil
	}
}

// ReduceExhaustion lowers a living combatant's Exhaustion by levels through
// ConditionSet.ReduceExhaustion (SRD 5.2.1, "Exhaustion [Condition]", p. 181). It returns
// the rejections of ApplyCondition up to CombatantIsDeadRejection and
// DoesNotHaveConditionRejection for a combatant without Exhaustion. It returns an error
// matching srderr.ErrInvalidInput for an invalid encounter or id or levels below 1.
func (e Encounter) ReduceExhaustion(id CombatantID, levels int) (encounterTransition, error) {
	if levels < 1 {
		return nil, invalidInput("levels must be at least 1")
	}
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	res, err := e.combatants[i].conditions.ReduceExhaustion(id, levels)
	if err != nil {
		return nil, err
	}
	switch r := res.(type) {
	case ExhaustionReduced:
		return e.withConditions(i, r.set, r.events, nil), nil
	case ExhaustionEnded:
		return e.withConditions(i, r.set, r.events, nil), nil
	default: // ExhaustionReductionNotPresent
		return e.reject(DoesNotHaveConditionRejection{id, content.ConditionExhaustion}), nil
	}
}

// grappleTarget is target for the grapple commands: the combatant must hold Grappled.
// It returns the position and the grappler named by the condition when it holds one.
func (e Encounter) grappleTarget(id CombatantID) (int, EncounterRejection, error) {
	i, rej, err := e.target(id)
	if err != nil || rej != nil {
		return i, rej, err
	}
	if _, ok := e.combatants[i].conditions.slots[content.ConditionGrappled].(Grappled); !ok {
		return i, DoesNotHaveConditionRejection{id, content.ConditionGrappled}, nil
	}
	return i, nil, nil
}

// endGrapple ends the grapple of the combatant at i with the cause through
// GrappleRules.EndOnSet: Grappled is removed and ConditionRemoved then GrappleEnded are
// wrapped. It never ends a grapple with ConditionSet.Remove.
func (e Encounter) endGrapple(i int, id CombatantID, cause GrappleEndCause) encounterTransition {
	r, ok := (GrappleRules{}).EndOnSet(e.combatants[i].conditions, id, cause).(GrappleEndOnSetApplied)
	if !ok { // unreachable: Grappled was read from the same set
		return e.reject(DoesNotHaveConditionRejection{id, content.ConditionGrappled})
	}
	var events []Event
	for _, ev := range r.events {
		switch v := ev.(type) {
		case ConditionEvent:
			events = append(events, ConditionEventOccurred{e.base(), v})
		case GrappleEvent:
			events = append(events, GrappleEventOccurred{e.base(), v})
		}
	}
	c := e.combatants[i]
	c.conditions = r.conditions
	return e.replace(i, c).accept(events)
}

// CheckGrapple asks whether the grapple holding the combatant still holds (SRD 5.2.1,
// "Ending a Grapple", p. 182): it reads the grappler's ConditionSet and calls
// GrappleRules.Hold with the distance and range the host supplies. When it does not
// hold the grapple ends with the matching cause; when it holds nothing changes. A dead
// grappler's hold is not ended here.
//
// It returns, first that applies, EncounterAlreadyEndedRejection,
// UnknownCombatantRejection (the combatant, or the grappler named by its Grappled),
// CombatantIsDeadRejection and DoesNotHaveConditionRejection (Grappled). It returns an
// error matching srderr.ErrInvalidInput for an invalid encounter or id or a negative
// distance or range.
func (e Encounter) CheckGrapple(id CombatantID, distanceFeet, rangeFeet int) (encounterTransition, error) {
	if distanceFeet < 0 || rangeFeet < 0 || id.IsZero() || !e.usable() {
		return nil, invalidInput("a grapple check needs a valid encounter, an id and a distance and range of 0 or more")
	}
	i := e.index(id)
	if e.ended() {
		return e.reject(EncounterAlreadyEndedRejection{}), nil
	}
	if i < 0 {
		return e.reject(UnknownCombatantRejection{id}), nil
	}
	c := e.combatants[i]
	g, held := c.conditions.slots[content.ConditionGrappled].(Grappled)
	grappler := e.index(g.Grappler)
	switch {
	case held && grappler < 0:
		return e.reject(UnknownCombatantRejection{g.Grappler}), nil
	case c.hp.isDead():
		return e.reject(CombatantIsDeadRejection{id}), nil
	case !held:
		return e.reject(DoesNotHaveConditionRejection{id, content.ConditionGrappled}), nil
	}
	// The numbers were checked above, so Hold returns no error.
	hold, _ := (GrappleRules{}).Hold(e.combatants[grappler].conditions, distanceFeet, rangeFeet)
	switch hold {
	case GrappleHoldGrapplerIncapacitated:
		return e.endGrapple(i, id, GrappleEndCauseGrapplerIncapacitated), nil
	case GrappleHoldOutOfRange:
		return e.endGrapple(i, id, GrappleEndCauseOutOfRange), nil
	case GrappleHoldHolds:
	}
	return e.accept(nil), nil
}

// ReleaseGrapple ends the grapple holding the combatant with the cause Released at no
// cost of action (SRD 5.2.1, "Ending a Grapple", p. 182), through GrappleRules.EndOnSet.
// The grappler is not looked up. It returns the rejections of CheckGrapple except the
// grappler's, and an error matching srderr.ErrInvalidInput for an invalid encounter or id.
func (e Encounter) ReleaseGrapple(id CombatantID) (encounterTransition, error) {
	i, rej, err := e.grappleTarget(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	return e.endGrapple(i, id, GrappleEndCauseReleased), nil
}

// EscapeGrapple applies the outcome of the host's escape check: on success the grapple
// ends with the cause Escaped through GrappleRules.EndOnSet; on failure nothing changes
// (SRD 5.2.1, "Ending a Grapple", p. 182). It returns the rejections of ReleaseGrapple
// and an error matching srderr.ErrInvalidInput for an invalid encounter or id or an
// outcome that is neither d20.Success nor d20.Failure.
func (e Encounter) EscapeGrapple(id CombatantID, outcome d20.Outcome) (encounterTransition, error) {
	if outcome != d20.Success && outcome != d20.Failure {
		return nil, invalidInput("escape needs a success or failure")
	}
	i, rej, err := e.grappleTarget(id)
	if err != nil || rej != nil {
		return e.rejected(rej, err)
	}
	if outcome == d20.Failure {
		return e.accept(nil), nil
	}
	return e.endGrapple(i, id, GrappleEndCauseEscaped), nil
}
