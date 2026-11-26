package combat

import "slices"

// GrappleEnding is a grapple that ended and the cause. It is data for the host,
// not an event: the host passes the target and cause to GrappleRules.EndOnSet,
// which removes Grappled and emits the one GrappleEnded. The zero value is
// invalid. Comparable with == and safe for concurrent use.
type GrappleEnding struct {
	grapple Grapple
	cause   GrappleEndCause
}

// Grapple returns the grapple that ended.
func (e GrappleEnding) Grapple() Grapple { return e.grapple }

// Cause returns how it ended.
func (e GrappleEnding) Cause() GrappleEndCause { return e.cause }

// GrappleTransition is the closed set of outcomes of a Grapples operation.
// Variants: GrappleTransitionApplied, GrappleTransitionRejected,
// GrappleTransitionInvalid. Whatever the variant, the receiver is unchanged.
type GrappleTransition interface{ isGrappleTransition() }

// GrappleTransitionApplied: the operation took effect. It carries the new set, the
// events and the endings for the host to pass to GrappleRules.EndOnSet.
type GrappleTransitionApplied struct {
	grapples Grapples
	events   []Event
	endings  []GrappleEnding
}

// Grapples returns the set after the operation.
func (t GrappleTransitionApplied) Grapples() Grapples { return t.grapples }

// Events returns a copy of the events, in the order they happened.
func (t GrappleTransitionApplied) Events() []Event { return slices.Clone(t.events) }

// Endings returns a copy of the endings, in the order the grapples began.
func (t GrappleTransitionApplied) Endings() []GrappleEnding { return slices.Clone(t.endings) }

// GrappleTransitionRejected: the rules refuse the operation in this state. It
// carries no events. Construct it with a keyed literal.
type GrappleTransitionRejected struct{ Rejection GrappleRejection }

// GrappleTransitionInvalid: host input the engine cannot use. It carries no
// events. Construct it with a keyed literal.
type GrappleTransitionInvalid struct{ Problem GrappleProblem }

func (GrappleTransitionApplied) isGrappleTransition()  {}
func (GrappleTransitionRejected) isGrappleTransition() {}
func (GrappleTransitionInvalid) isGrappleTransition()  {}

func rejectGrapple(r GrappleRejection) GrappleTransition { return GrappleTransitionRejected{r} }

// without returns a new set holding the grapples for which drop is false.
func (s Grapples) without(drop func(Grapple) bool) Grapples {
	var kept []Grapple
	for _, g := range s.list {
		if !drop(g) {
			kept = append(kept, g)
		}
	}
	return Grapples{kept}
}

// Begin adds the grapple (SRD 5.2.1, "Grappling", p. 182). It checks, in this
// order: the grapple (GrappleTransitionInvalid with the problem NewGrapple would
// report), a part already holding a creature (GrappleRejectionPartInUse), a target
// already held by any grappler (GrappleRejectionTargetAlreadyGrappled). A set already holding 256 grapples
// is GrappleTransitionInvalid with GrappleProblemLimitExceeded (plan P10). Otherwise
// it returns GrappleTransitionApplied with one GrappleStarted and no endings. The
// grappler's conditions, sizes and the distance are not consulted.
func (s Grapples) Begin(grapple Grapple) GrappleTransition {
	if p, bad := grapple.problem(); bad {
		return GrappleTransitionInvalid{p}
	}
	if s.IsHolding(grapple.Grappler, grapple.Part) == PartStateHolding {
		return rejectGrapple(GrappleRejectionPartInUse)
	}
	if _, held := s.GrappleOf(grapple.Target).(InGrapple); held {
		return rejectGrapple(GrappleRejectionTargetAlreadyGrappled)
	}
	if len(s.list) >= maxInputList {
		return GrappleTransitionInvalid{GrappleProblemLimitExceeded}
	}
	started := GrappleStarted{grappleEventBase{grapple.Target, grapple.Grappler}, grapple.Part}
	return GrappleTransitionApplied{grapples: Grapples{append(s.All(), grapple)}, events: []Event{started}}
}

// Release removes the grapple in which the grappler holds the target, at no
// action. It returns GrappleTransitionInvalid with
// GrappleProblemSameGrapplerAndTarget for equal ids, GrappleRejectionNotHeld when
// the grappler does not hold the target, and otherwise GrappleTransitionApplied
// with no event and one GrappleEnding with GrappleEndCauseReleased.
func (s Grapples) Release(grappler, target CombatantID) GrappleTransition {
	if grappler == target {
		return GrappleTransitionInvalid{GrappleProblemSameGrapplerAndTarget}
	}
	if g, ok := s.GrappleOf(target).(InGrapple); ok && g.grapple.Grappler == grappler {
		return s.ended(g.grapple, GrappleEndCauseReleased)
	}
	return rejectGrapple(GrappleRejectionNotHeld)
}

// Escape removes the grapple holding the target, whose escape check succeeded.
// It returns GrappleRejectionNotGrappled when none holds it, otherwise
// GrappleTransitionApplied with no event and one GrappleEnding with
// GrappleEndCauseEscaped. The roll is the escape use case's.
func (s Grapples) Escape(target CombatantID) GrappleTransition {
	if g, ok := s.GrappleOf(target).(InGrapple); ok {
		return s.ended(g.grapple, GrappleEndCauseEscaped)
	}
	return rejectGrapple(GrappleRejectionNotGrappled)
}

// ended removes g and reports it as one ending.
func (s Grapples) ended(g Grapple, cause GrappleEndCause) GrappleTransition {
	return GrappleTransitionApplied{
		grapples: s.without(func(o Grapple) bool { return o == g }),
		endings:  []GrappleEnding{{g, cause}},
	}
}

// ConditionEnded removes the grapple holding the target because its Grappled
// condition ended by some other rule, so the record never outlives the condition.
// It returns GrappleRejectionNotGrappled when none holds it, otherwise
// GrappleTransitionApplied with one GrappleRecordDropped and no ending (the
// condition is already gone).
func (s Grapples) ConditionEnded(target CombatantID) GrappleTransition {
	g, ok := s.GrappleOf(target).(InGrapple)
	if !ok {
		return rejectGrapple(GrappleRejectionNotGrappled)
	}
	dropped := GrappleRecordDropped{grappleEventBase{g.grapple.Target, g.grapple.Grappler}}
	return GrappleTransitionApplied{
		grapples: s.without(func(o Grapple) bool { return o == g.grapple }),
		events:   []Event{dropped},
	}
}

// Review ends every grapple whose observation says it no longer holds
// (GrappleRules.Hold, with the observation's GrapplerDead). A grapple with no
// observation is left alone. It returns GrappleTransitionInvalid with
// GrappleProblemUnknownGrapple for an observation of a grapple the set does not
// hold (the first in the order given, checked before any duplicate), with
// GrappleProblemDuplicateObservation for two observations of one grapple and, before both, with
// GrappleProblemLimitExceeded for more than 256 observations (plan P10);
// otherwise GrappleTransitionApplied with the endings in the order the grapples
// began and no event.
func (s Grapples) Review(observations []GrappleObservation) GrappleTransition {
	if len(observations) > maxInputList {
		return GrappleTransitionInvalid{GrappleProblemLimitExceeded}
	}
	type pair struct{ grappler, target CombatantID }
	known := make(map[pair]struct{}, len(s.list))
	for _, g := range s.list {
		known[pair{g.Grappler, g.Target}] = struct{}{}
	}
	for _, o := range observations {
		if _, ok := known[pair{o.grappler, o.target}]; !ok {
			return GrappleTransitionInvalid{GrappleProblemUnknownGrapple}
		}
	}
	seen := make(map[pair]GrappleObservation, len(observations))
	for _, o := range observations {
		k := pair{o.grappler, o.target}
		if _, dup := seen[k]; dup {
			return GrappleTransitionInvalid{GrappleProblemDuplicateObservation}
		}
		seen[k] = o
	}
	var endings []GrappleEnding
	ended := map[Grapple]struct{}{}
	for _, g := range s.list {
		o, ok := seen[pair{g.Grappler, g.Target}]
		if !ok {
			continue
		}
		switch holdOf(o.conditions, o.distanceFeet, g.RangeFeet, o.dead) {
		case GrappleHoldHolds:
			continue
		case GrappleHoldGrapplerIncapacitated:
			endings = append(endings, GrappleEnding{g, GrappleEndCauseGrapplerIncapacitated})
			ended[g] = struct{}{}
		case GrappleHoldOutOfRange:
			endings = append(endings, GrappleEnding{g, GrappleEndCauseOutOfRange})
			ended[g] = struct{}{}
		}
	}
	return GrappleTransitionApplied{
		grapples: s.without(func(g Grapple) bool { _, gone := ended[g]; return gone }),
		endings:  endings,
	}
}
