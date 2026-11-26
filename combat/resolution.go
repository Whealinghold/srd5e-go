package combat

// resolution accumulates the state and the events of one combat call: the encounter after every step so
// far and the events in the order they happened (combat-action-resolution.md FR-089, FR-091). It is a local
// of one call, never stored, and holds no port. Every change to the encounter goes through the root's own
// methods (FR-008).
type resolution struct {
	enc    Encounter
	events []Event
	// steps counts the steps of the call: each committed transition and each Concentration check (a prompt is an event, counted by the event cap)
	// (the cascade step budget of plan P10).
	steps int
	// passes counts the full passes over the combatants the call made since its entry: exitInvalid is the only one, so a cascade of
	// any length counts 0 until the call ends and 1 after (plan P14; encounter_cost_internal_test.go).
	passes int
}

// maxStepsPerCall is the cascade step budget of plan P10: the steps one call may take.
const maxStepsPerCall = 1000

// commitTurn commits the tracker step through Encounter.ApplyTurnChange (FR-014). The two rejections it can
// return, a tracker with another turn order and an event naming an unknown combatant, cannot happen: the
// tracker comes from the encounter and every id passed the stops. An error matching srderr.ErrInvalidInput
// is returned as it is.
func (r *resolution) commitTurn(step TrackerStepApplied) error {
	r.steps++
	tr, err := r.enc.ApplyTurnChange(step.Tracker(), step.Events())
	if err != nil {
		return err
	}
	switch tr.(type) {
	case transitionApplied:
		r.enc, r.events = tr.Encounter(), append(r.events, tr.Events()...)
		return nil
	default:
		panic(unreachableError("turn change rejection", tr.Rejection()))
	}
}

// commit commits a transition the encounter returned: it takes the new encounter and the events and
// returns the resolutions, or the rejection and no change.
func (r *resolution) commit(tr encounterTransition) (EncounterRejection, []DamageResolution) {
	r.steps++
	if rej := tr.Rejection(); rej != nil {
		return rej, nil
	}
	r.enc, r.events = tr.Encounter(), append(r.events, tr.Events()...)
	return nil, tr.Resolutions()
}

// mustCommit commits a transition whose rejection the stops make unreachable. Anything but an applied
// transition is a broken invariant.
func (r *resolution) mustCommit(tr encounterTransition) []DamageResolution {
	rej, res := r.commit(tr)
	switch rej.(type) {
	case nil:
		return res
	default:
		panic(unreachableError("encounter rejection", rej))
	}
}

// add appends events in the order given.
func (r *resolution) add(events ...Event) { r.events = append(r.events, events...) }

// base is the id and round every ResolutionEvent of the call now carries.
func (r *resolution) base() resolutionEventBase {
	return resolutionEventBase{r.enc.id, r.enc.tracker.round}
}

// prompts appends the prompts that follow a call that changed conditions: the Speed reviews, then the
// grapple checks, in turn order (FR-085, FR-120, FR-088). before is the encounter the call started with.
func (r *resolution) prompts(before Encounter) {
	rules := ConsequenceRules{}
	for _, id := range rules.SpeedReviews(before.combatants, r.enc.combatants) {
		r.add(MovementSpeedsReviewBecameDue{r.base(), id})
	}
	for _, cue := range rules.GrappleChecks(before.combatants, r.enc.combatants) {
		r.add(GrappleCheckBecameDue{r.base(), cue.Grappled, cue.Grappler})
	}
}

// overLimit reports whether the call produced more events than the cap of plan P10 or took more steps than its budget:
// the call is then an invalid result with the shared limit_exceeded value and changes nothing.
func (r *resolution) overLimit() bool {
	return len(r.events) > maxEventsPerCall || r.steps > maxStepsPerCall
}

// exitInvalid reports whether the encounter a call produced is not valid: the root validates at entry and at
// exit of every call (plan P14), never inside the call.
func (r *resolution) exitInvalid() bool {
	r.passes++
	return r.enc.IsZero()
}

// availabilityOf returns what the conditions of a combatant allow it to do with actions (FR-016): read
// from its own ConditionSet, never from the command.
func availabilityOf(set ConditionSet) ConditionActionAvailability {
	return (ConditionEffects{}).ActiveFlags(set).Actions
}

// costStep computes, without committing it, the tracker step an attack's cost spends (FR-019): the Attack
// action, the Bonus Action or the Reaction. An Attack action that continues and an attack already paid spend
// nothing and give a nil step.
func costStep(t TurnTracker, id CombatantID, set ConditionSet, c AttackCost) (step TrackerStep, err error) {
	av := availabilityOf(set)
	switch v := c.(type) {
	case BeginsAttackAction:
		step, err = t.TakeAction(id, av, SimpleActionChoice{SimpleStandardActionAttack})
	case BonusActionCost:
		step, err = t.TakeBonusAction(id, av, bonusEntitlement(v.Grant))
	case ReactionCost:
		step, err = t.TakeReaction(id, av, v.Choice)
	}
	return step, err
}

// bonusEntitlement is the entitlement of a grant: an absent grant is not entitled (FR-051).
func bonusEntitlement(g BonusActionGrant) BonusActionEntitlement {
	if g.IsZero() {
		return NotEntitledToBonusAction{}
	}
	return EntitledToBonusAction{g}
}

// fallProne applies Prone to a creature that ended its turn in a space shared with another creature (SRD 5.2.1,
// "Moving around Other Creatures", p. 14). A refusal (it already has Prone, is Immune to it or is dead) does not
// stop the call: the advance stands and no event is produced for it (FR-065).
func (r *resolution) fallProne(id CombatantID) {
	// The only error is ErrInvalidInput for an invalid encounter, which the call already ruled out at entry; it is dropped on purpose.
	if tr, err := r.enc.ApplyCondition(id, Prone{}); err == nil {
		r.commit(tr)
	}
}
