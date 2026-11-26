package combat

// ShapeShiftEvent is the closed family of events of the shape-shifting spells. Each is an
// immutable value that carries the spell and never display text, with a constant snake_case
// Kind and a Combatants that lists the target, then the caster when they differ, as a fresh
// copy. The zero value of each variant is invalid; the operations build them. Variants:
// ShapeShiftBegan, ShapeShiftFormChanged, ShapeShiftMadePermanent, ShapeShiftEnded. Each
// embeds shapeShiftEventBase.
type ShapeShiftEvent interface {
	Event
	shapeShiftEvent()
}

// shapeShiftEventBase carries what every shape-shift event is about and puts the event in
// the Event and ShapeShiftEvent families.
type shapeShiftEventBase struct {
	target, caster CombatantID
	spell          ShapeShiftSpell
}

func (shapeShiftEventBase) event()           {}
func (shapeShiftEventBase) shapeShiftEvent() {}

// Combatants returns a new slice holding the target, then the caster when it is another creature.
func (b shapeShiftEventBase) Combatants() []CombatantID {
	if b.caster == b.target {
		return []CombatantID{b.target}
	}
	return []CombatantID{b.target, b.caster}
}

// Spell returns the spell.
func (b shapeShiftEventBase) Spell() ShapeShiftSpell { return b.spell }

// Parties returns the transformed creature and the creature that cast the spell.
func (b shapeShiftEventBase) Parties() ShapeShiftParties {
	return ShapeShiftParties{b.target, b.caster}
}

func shapeShiftBaseOf(t ActiveTransformation) shapeShiftEventBase {
	return shapeShiftEventBase{target: t.target, caster: t.caster, spell: t.spell}
}

// ShapeShiftBegan: the spell shape-shifted the target. Its Kind is shape_shift_began.
type ShapeShiftBegan struct{ shapeShiftEventBase }

// Kind returns "shape_shift_began".
func (ShapeShiftBegan) Kind() string { return "shape_shift_began" }

// ShapeShiftFormChanged: the target took another form under the same spell. Its Kind is
// shape_shift_form_changed.
type ShapeShiftFormChanged struct{ shapeShiftEventBase }

// Kind returns "shape_shift_form_changed".
func (ShapeShiftFormChanged) Kind() string { return "shape_shift_form_changed" }

// ShapeShiftMadePermanent: Concentration held for the full duration made the True Polymorph
// last until dispelled. Its Kind is shape_shift_made_permanent.
type ShapeShiftMadePermanent struct {
	shapeShiftEventBase
	controlLost bool
}

// Kind returns "shape_shift_made_permanent".
func (ShapeShiftMadePermanent) Kind() string { return "shape_shift_made_permanent" }

// ControlLost reports that the caster no longer controls the creature: an object turned
// into a creature that stays one for more than an hour (SRD 5.2.1, "True Polymorph", p. 170).
func (e ShapeShiftMadePermanent) ControlLost() bool { return e.controlLost }

// ShapeShiftEnded: the transformation ended and the target reverts to its true form. Its
// Kind is shape_shift_ended.
type ShapeShiftEnded struct {
	shapeShiftEventBase
	cause      ShapeShiftEndCause
	memoryLost bool
}

// Kind returns "shape_shift_ended".
func (ShapeShiftEnded) Kind() string { return "shape_shift_ended" }

// Cause returns why the transformation ended.
func (e ShapeShiftEnded) Cause() ShapeShiftEndCause { return e.cause }

// RevertsToTrueForm reports that the target returns to its true form: always true (SRD 5.2.1,
// "Shape-Shifting", p. 187). The host passes CarryOverAll to ShapeShifting.Revert.
func (ShapeShiftEnded) RevertsToTrueForm() bool { return true }

// MemoryOfObjectFormLost reports that a creature turned into an object no longer remembers
// being one (SRD 5.2.1, "True Polymorph", p. 170).
func (e ShapeShiftEnded) MemoryOfObjectFormLost() bool { return e.memoryLost }
