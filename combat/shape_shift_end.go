package combat

// EndShapeShiftCommand is the host's facts for End. Construct it with a keyed literal.
type EndShapeShiftCommand struct {
	// State is the transformation to end; the zero value means none.
	State ActiveTransformation
	// Cause is why it ends.
	Cause ShapeShiftEndCause
	// HitPoints is the target's Hit Point side; the status is read only when the form's
	// Temporary Hit Points end with the transformation.
	HitPoints TargetHitPoints
	// ConcentrationOf is the creature whose Concentration ended, read only for
	// CauseConcentrationEnded (the host derives it from the casting rules).
	ConcentrationOf CombatantID
}

// EndShapeShiftingResult is the closed set of answers of ShapeShiftSpells.End. Variants:
// EndShapeShiftApplied, EndShapeShiftUnaffected, EndShapeShiftRefused.
type EndShapeShiftingResult interface{ isEndShapeShiftingResult() }

// EndShapeShiftApplied: the transformation ended. The transformation is cleared (the zero
// value) and the target reverts to its true form: the host calls ShapeShifting.Revert with
// CarryOver. Events holds the Temporary Hit Point event, if any, then ShapeShiftEnded.
type EndShapeShiftApplied struct {
	status HitPointStatus
	events []Event
	cost   ShapeShiftActionCost
}

// Transformation returns the cleared state: the zero ActiveTransformation.
func (EndShapeShiftApplied) Transformation() ActiveTransformation { return ActiveTransformation{} }

// Status returns the hit point status after the end; the zero status for an untracked target.
func (r EndShapeShiftApplied) Status() HitPointStatus { return r.status }

// Events returns a copy of the events, in order.
func (r EndShapeShiftApplied) Events() []Event { return append([]Event(nil), r.events...) }

// CarryOver returns the rule to pass to ShapeShifting.Revert: CarryOverAll.
func (EndShapeShiftApplied) CarryOver() CarryOverRule { return CarryOverAll{} }

// ActionCost returns the Bonus Action the target spent and true when it ended Animal Shapes
// itself (CauseEndedByTarget), and false otherwise.
func (r EndShapeShiftApplied) ActionCost() (ShapeShiftActionCost, bool) { return r.cost, r.cost != 0 }

// EndShapeShiftUnaffected: the cause does not end this transformation, or there is none.
// Nothing changes and there are no events.
type EndShapeShiftUnaffected struct{}

// EndShapeShiftRefused: the rules refuse; nothing changes. Construct it with a keyed literal.
type EndShapeShiftRefused struct{ Rejection EndShapeShiftRejection }

func (EndShapeShiftApplied) isEndShapeShiftingResult()    {}
func (EndShapeShiftUnaffected) isEndShapeShiftingResult() {}
func (EndShapeShiftRefused) isEndShapeShiftingResult()    {}

// End ends a transformation for a cause. With no transformation the answer is Unaffected. On a
// Dead target only CauseTargetDied and CauseTargetDestroyed apply and any other cause is
// refused. A permanent transformation ends only by CauseDispelled, CauseTargetDied,
// CauseTargetDestroyed or CauseReplaced; the other causes leave it Unaffected. Otherwise:
// CauseDurationElapsed ends it; CauseConcentrationEnded ends a transformation kept by that
// creature's Concentration; CauseFormHitPointsDepleted ends a Polymorph; CauseEndedByTarget
// ends Animal Shapes at the cost of a Bonus Action and is refused (CannotEndAtWill) for the
// other spells; CauseDispelled and CauseReplaced always end it; CauseTargetDied and
// CauseTargetDestroyed end it whatever the spell (SRD 5.2.1, "Shape-Shifting", p. 187). A
// transformation whose form Temporary Hit Points came FromForm loses any still held
// (HitPointStatus.EndTemporary), unless the target is Dead or died.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil result, for an invalid
// cause or Hit Point side, a missing ConcentrationOf for CauseConcentrationEnded, or an
// untracked target whose form Temporary Hit Points are FromForm. It never panics and changes
// none of its arguments.
func (ShapeShiftSpells) End(c EndShapeShiftCommand) (EndShapeShiftingResult, error) {
	t := c.State
	if !c.Cause.Valid() || !validTargetHitPoints(c.HitPoints) || (c.Cause == CauseConcentrationEnded && c.ConcentrationOf.IsZero()) {
		return nil, invalidInput("End needs a cause, Hit Points and, for Concentration, its creature")
	}
	if err := checkEndHitPoints(t, c.HitPoints); err != nil {
		return nil, err
	}
	died := c.Cause == CauseTargetDied || c.Cause == CauseTargetDestroyed
	switch {
	case t.IsZero():
		return EndShapeShiftUnaffected{}, nil
	case targetIsDead(c.HitPoints) && !died:
		return EndShapeShiftRefused{EndTargetIsDead}, nil
	case c.Cause == CauseEndedByTarget && t.spell != AnimalShapesSpell && !t.permanent:
		return EndShapeShiftRefused{CannotEndAtWill}, nil
	case !endsFor(t, c):
		return EndShapeShiftUnaffected{}, nil
	}
	status, events, err := endTransformation(t, c.Cause, c.HitPoints)
	if err != nil {
		return nil, err
	}
	r := EndShapeShiftApplied{status: status, events: events}
	if c.Cause == CauseEndedByTarget {
		r.cost = ShapeShiftBonusAction
	}
	return r, nil
}

// endsFor reports whether the cause of c ends the non-zero transformation t.
func endsFor(t ActiveTransformation, c EndShapeShiftCommand) bool {
	switch c.Cause {
	case CauseTargetDied, CauseTargetDestroyed, CauseDispelled, CauseReplaced:
		return true
	case CauseDurationElapsed:
		return !t.permanent
	case CauseConcentrationEnded:
		return !t.permanent && t.maintained == MaintainedByConcentration && t.caster == c.ConcentrationOf
	case CauseFormHitPointsDepleted:
		return !t.permanent && t.spell == PolymorphSpell
	case CauseEndedByTarget:
		return !t.permanent && t.spell == AnimalShapesSpell
	}
	return false
}

// checkEndHitPoints rejects an untracked target whose transformation holds form Temporary Hit Points.
func checkEndHitPoints(t ActiveTransformation, hp TargetHitPoints) error {
	if _, tracked := hp.(TrackedHitPoints); !tracked && t.temporary == TemporaryFromForm {
		return invalidInput("a transformation with form Temporary Hit Points needs tracked Hit Points")
	}
	return nil
}

// endTransformation ends t for cause: it ends the form's Temporary Hit Points when they are
// FromForm and the target lives, then adds ShapeShiftEnded. It returns the new status (zero
// when untracked) and the events in order.
func endTransformation(t ActiveTransformation, cause ShapeShiftEndCause, hp TargetHitPoints) (HitPointStatus, []Event, error) {
	status := statusOf(hp)
	var events []Event
	dying := targetIsDead(hp) || cause == CauseTargetDied || cause == CauseTargetDestroyed
	if t.temporary == TemporaryFromForm && !dying {
		ended, err := status.EndTemporary(t.target)
		if err != nil {
			return HitPointStatus{}, nil, err
		}
		if done, ok := ended.(HitPointTransitionApplied); ok {
			status = done.Status()
			for _, e := range done.Events() {
				events = append(events, e)
			}
		}
	}
	events = append(events, ShapeShiftEnded{shapeShiftBaseOf(t), cause, t.mode == CreatureIntoObject})
	return status, events, nil
}

// ElapseFullDurationResult is the closed set of answers of ShapeShiftSpells.ElapseFullDuration.
// Variants: ElapseShapeShiftMadePermanent, ElapseShapeShiftEnded, ElapseShapeShiftUnaffected.
type ElapseFullDurationResult interface{ isElapseFullDurationResult() }

// ElapseShapeShiftMadePermanent: Concentration held for the full duration made the True
// Polymorph last until dispelled. It carries the permanent transformation and one
// ShapeShiftMadePermanent; nothing ended.
type ElapseShapeShiftMadePermanent struct {
	transformation ActiveTransformation
	events         []Event
}

// Transformation returns the permanent ActiveTransformation.
func (r ElapseShapeShiftMadePermanent) Transformation() ActiveTransformation { return r.transformation }

// Events returns a copy of the events: one ShapeShiftMadePermanent.
func (r ElapseShapeShiftMadePermanent) Events() []Event { return append([]Event(nil), r.events...) }

// ElapseShapeShiftEnded: the duration ended the transformation, as End with CauseDurationElapsed.
type ElapseShapeShiftEnded struct{ end EndShapeShiftApplied }

// Ended returns the ending: the cleared state, the status and the events.
func (r ElapseShapeShiftEnded) Ended() EndShapeShiftApplied { return r.end }

// ElapseShapeShiftUnaffected: there is no transformation, it is already permanent, or the
// target is Dead (its death ends it through CauseTargetDied). Nothing changes.
type ElapseShapeShiftUnaffected struct{}

func (ElapseShapeShiftMadePermanent) isElapseFullDurationResult() {}
func (ElapseShapeShiftEnded) isElapseFullDurationResult()         {}
func (ElapseShapeShiftUnaffected) isElapseFullDurationResult()    {}

// ElapseFullDuration handles the end of the full duration of a transformation (SRD 5.2.1,
// "True Polymorph", p. 170). A True Polymorph kept by Concentration for the full duration
// becomes permanent (ElapseShapeShiftMadePermanent, with ControlLost for an object turned
// into a creature); otherwise, and for every other spell, the transformation ends with
// CauseDurationElapsed. A zero, permanent or Dead-target transformation is left Unaffected.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil result, for an invalid Hit
// Point side or an untracked target whose form Temporary Hit Points are FromForm. It never
// panics and changes none of its arguments.
func (s ShapeShiftSpells) ElapseFullDuration(t ActiveTransformation, concentrationHeldToTheEnd bool, hp TargetHitPoints) (ElapseFullDurationResult, error) {
	if !validTargetHitPoints(hp) {
		return nil, invalidInput("ElapseFullDuration needs Hit Points")
	}
	if err := checkEndHitPoints(t, hp); err != nil {
		return nil, err
	}
	if t.IsZero() || t.permanent || targetIsDead(hp) {
		return ElapseShapeShiftUnaffected{}, nil
	}
	if t.spell.Profile().PermanentIfHeld && concentrationHeldToTheEnd {
		t.permanent = true
		return ElapseShapeShiftMadePermanent{t, []Event{ShapeShiftMadePermanent{shapeShiftBaseOf(t), t.mode == ObjectIntoCreature}}}, nil
	}
	status, events, err := endTransformation(t, CauseDurationElapsed, hp)
	if err != nil {
		return nil, err
	}
	return ElapseShapeShiftEnded{EndShapeShiftApplied{status: status, events: events}}, nil
}

// AssessShapeShiftResult is the closed set of answers of ShapeShiftSpells.Assess. Variants:
// AssessShapeShiftContinues, AssessShapeShiftEnded.
type AssessShapeShiftResult interface{ isAssessShapeShiftResult() }

// AssessShapeShiftContinues: the transformation goes on.
type AssessShapeShiftContinues struct{}

// AssessShapeShiftEnded: the transformation must end now; the host calls End with Cause.
type AssessShapeShiftEnded struct{ cause ShapeShiftEndCause }

// Cause returns CauseFormHitPointsDepleted.
func (r AssessShapeShiftEnded) Cause() ShapeShiftEndCause { return r.cause }

func (AssessShapeShiftContinues) isAssessShapeShiftResult() {}
func (AssessShapeShiftEnded) isAssessShapeShiftResult()     {}

// Assess is called after damage: a Polymorph ends early when the target has no Temporary Hit
// Points left, whatever their source (SRD 5.2.1, "Polymorph", p. 153). It reads the status and
// changes no Hit Points: the damage beyond the Temporary Hit Points was already taken from
// the target's own Hit Points. Every other spell, a permanent or missing transformation and a
// Dead target (whose death ends it through CauseTargetDied) always continue.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil result, for the zero status.
// It never panics.
func (ShapeShiftSpells) Assess(t ActiveTransformation, status HitPointStatus) (AssessShapeShiftResult, error) {
	if status.IsZero() {
		return nil, invalidInput("Assess needs a hit point status")
	}
	_, dead := status.LifeState().(Dead)
	if t.spell.Profile().EndsWhenTemporaryHitPointsGone && !t.permanent && !dead && status.HitPoints().Temporary() == 0 {
		return AssessShapeShiftEnded{CauseFormHitPointsDepleted}, nil
	}
	return AssessShapeShiftContinues{}, nil
}
