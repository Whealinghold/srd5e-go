package combat

// GrappleEvent is the closed family of events the grapple rules produce. Each is
// an immutable value comparable with ==, about a target and its grappler (its
// Combatants is exactly [target, grappler]) with a constant snake_case Kind.
// Variants: GrappleStarted, GrappleEnded, GrappleRecordDropped. Each embeds
// grappleEventBase. There are no accessors for the two ids, because no function
// of this package returns a CombatantID; read them from Combatants.
type GrappleEvent interface {
	Event
	grappleEvent()
}

// grappleEventBase carries the two creatures every grapple event is about and the
// markers that put the event in the Event and GrappleEvent families.
type grappleEventBase struct{ target, grappler CombatantID }

func (grappleEventBase) event()        {}
func (grappleEventBase) grappleEvent() {}

// Combatants returns a new slice [target, grappler].
func (b grappleEventBase) Combatants() []CombatantID { return []CombatantID{b.target, b.grappler} }

func newGrappleBase(grappler, target CombatantID) (grappleEventBase, error) {
	switch {
	case grappler.IsZero() || target.IsZero():
		return grappleEventBase{}, problemError(GrappleProblemMissingValue)
	case grappler == target:
		return grappleEventBase{}, problemError(GrappleProblemSameGrapplerAndTarget)
	}
	return grappleEventBase{target: target, grappler: grappler}, nil
}

// GrappleStarted: the grappler began holding the target with the part. Its Kind is
// grapple_started.
type GrappleStarted struct {
	grappleEventBase
	part GraspingPart
}

// NewGrappleStarted returns the event. It returns a *GrappleProblemError
// (matching srderr.ErrInvalidInput) for a blank id, an invalid part or equal ids.
func NewGrappleStarted(grappler, target CombatantID, part GraspingPart) (GrappleStarted, error) {
	b, err := newGrappleBase(grappler, target)
	if err != nil {
		return GrappleStarted{}, err
	}
	if !validPart(part) {
		return GrappleStarted{}, problemError(GrappleProblemMissingValue)
	}
	return GrappleStarted{b, part}, nil
}

// Kind returns "grapple_started".
func (GrappleStarted) Kind() string { return "grapple_started" }

// Part returns the part the grappler holds the target with.
func (e GrappleStarted) Part() GraspingPart { return e.part }

// GrappleEnded: a grapple ended for the cause. It is produced only by
// GrappleRules.EndOnSet, after the ConditionRemoved of Grappled. Its Kind is
// grapple_ended.
type GrappleEnded struct {
	grappleEventBase
	cause GrappleEndCause
}

// NewGrappleEnded returns the event. It returns a *GrappleProblemError (matching
// srderr.ErrInvalidInput) for a blank id or equal ids, and an error matching
// srderr.ErrInvalidInput for a cause that is not one of the four.
func NewGrappleEnded(target, grappler CombatantID, cause GrappleEndCause) (GrappleEnded, error) {
	b, err := newGrappleBase(grappler, target)
	if err != nil {
		return GrappleEnded{}, err
	}
	if !cause.Valid() {
		return GrappleEnded{}, invalidInput("grapple ended needs a cause")
	}
	return GrappleEnded{b, cause}, nil
}

// Kind returns "grapple_ended".
func (GrappleEnded) Kind() string { return "grapple_ended" }

// Cause returns how the grapple ended.
func (e GrappleEnded) Cause() GrappleEndCause { return e.cause }

// GrappleRecordDropped: the record of a grapple was dropped because its Grappled
// condition ended by some other route. Its Kind is grapple_record_dropped.
type GrappleRecordDropped struct{ grappleEventBase }

// NewGrappleRecordDropped returns the event. It returns a *GrappleProblemError
// (matching srderr.ErrInvalidInput) for a blank id or equal ids.
func NewGrappleRecordDropped(grappler, target CombatantID) (GrappleRecordDropped, error) {
	b, err := newGrappleBase(grappler, target)
	if err != nil {
		return GrappleRecordDropped{}, err
	}
	return GrappleRecordDropped{b}, nil
}

// Kind returns "grapple_record_dropped".
func (GrappleRecordDropped) Kind() string { return "grapple_record_dropped" }
