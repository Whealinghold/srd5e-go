package combat

import "github.com/Whealinghold/srd5e-go/content"

// BeginShapeShiftCommand is the host's facts for Begin. Construct it with a keyed literal.
type BeginShapeShiftCommand struct {
	// Spell is the spell cast; Caster the creature that cast it; Target the creature shifted.
	// For an object turned into a creature, Target is the CombatantID the host gives the
	// creature (shape-changing-spells.md FR-062); a creature turned into an object keeps its own.
	Spell          ShapeShiftSpell
	Caster, Target CombatantID
	// Mode is the mode of True Polymorph; ignored for the other spells.
	Mode TrueTransformation
	// Terms are the terms of Shapechange; the zero value is StandardShapechange.
	Terms ShapechangeTerms
	// Form is the form chosen, with the size chosen among its options.
	Form AssumedForm
	// Response is the target's response; an object of True Polymorph is given Willing.
	Response TargetResponse
	// Rating is the target's rating (Polymorph, True Polymorph creature into creature) or the
	// caster's (Shapechange): the Challenge Rating or level the form may not exceed.
	Rating TargetRating
	// HasSeenKind is whether the caster has seen the sort of creature (Shapechange).
	HasSeenKind bool
	// HoldsSpellcastingFeature is whether the Shapechange caster has the Spellcasting feature.
	HoldsSpellcastingFeature bool
	// Object describes the object of True Polymorph object into creature; TargetSize is the
	// size of the creature of True Polymorph creature into object.
	Object     ObjectFacts
	TargetSize content.CreatureSize
	// HitPoints is the target's Hit Point side.
	HitPoints TargetHitPoints
	// Equipment is the caster's choice for Shapechange (DropsToGround or ResizesToFit) and
	// must be the zero value for the other spells.
	Equipment EquipmentOutcome
	// Earlier is the transformation the target already holds; the zero value means none.
	Earlier ActiveTransformation
}

// BeginShapeShiftingResult is the closed set of answers of ShapeShiftSpells.Begin. Variants:
// BeginShapeShiftApplied, BeginShapeShiftResisted, BeginShapeShiftRefused.
type BeginShapeShiftingResult interface{ isBeginShapeShiftingResult() }

// BeginShapeShiftApplied: the target is shape-shifted. It carries the new transformation and
// hit point status, the ordered events (the ending of an earlier transformation, then
// ShapeShiftBegan, then TemporaryHitPointsGained), and the inputs for ShapeShifting.Shift:
// Effect is SpellSource of the spell and CarryOver is CarryOverAll, because none of the four
// descriptions says otherwise (SRD 5.2.1, "Shape-Shifting", p. 187).
type BeginShapeShiftApplied struct {
	transformation ActiveTransformation
	status         HitPointStatus
	events         []Event
	effect         EffectSource
	equipment      EquipmentOutcome
	restrictions   FormRestrictions
	statistics     FormStatistics
	animated       AnimatedObjectTerms
	gained         bool
}

// Transformation returns the new ActiveTransformation.
func (r BeginShapeShiftApplied) Transformation() ActiveTransformation { return r.transformation }

// Status returns the new hit point status; the zero status for an untracked target.
func (r BeginShapeShiftApplied) Status() HitPointStatus { return r.status }

// Events returns a copy of the events, in order.
func (r BeginShapeShiftApplied) Events() []Event { return append([]Event(nil), r.events...) }

// Effect returns the effect source to pass to ShapeShifting.Shift.
func (r BeginShapeShiftApplied) Effect() EffectSource { return r.effect }

// CarryOver returns the rule to pass to ShapeShifting.Shift: CarryOverAll.
func (BeginShapeShiftApplied) CarryOver() CarryOverRule { return CarryOverAll{} }

// Equipment returns what becomes of the target's equipment.
func (r BeginShapeShiftApplied) Equipment() EquipmentOutcome { return r.equipment }

// Restrictions returns what the form forbids.
func (r BeginShapeShiftApplied) Restrictions() FormRestrictions { return r.restrictions }

// Statistics returns the statistics the target keeps.
func (r BeginShapeShiftApplied) Statistics() FormStatistics { return r.statistics }

// AnimatedObject returns the terms of an object turned into a creature and true for True
// Polymorph object into creature; the zero terms and false otherwise.
func (r BeginShapeShiftApplied) AnimatedObject() (AnimatedObjectTerms, bool) {
	return r.animated, !r.animated.IsZero()
}

// GainedTemporaryHitPointsEvent reports whether the events hold a TemporaryHitPointsGained.
// It is false when no Temporary Hit Points were granted, or when the offer equalled the held
// amount or the creature kept what it held (shape-changing-spells.md FR-059).
func (r BeginShapeShiftApplied) GainedTemporaryHitPointsEvent() bool { return r.gained }

// BeginShapeShiftResisted: the target's saving throw succeeded and it is not affected. Nothing
// changes and there are no events.
type BeginShapeShiftResisted struct{}

// BeginShapeShiftRefused: the rules refuse in this state; nothing changes. Construct it with a
// keyed literal.
type BeginShapeShiftRefused struct{ Rejection BeginShapeShiftRejection }

func (BeginShapeShiftApplied) isBeginShapeShiftingResult()  {}
func (BeginShapeShiftResisted) isBeginShapeShiftingResult() {}
func (BeginShapeShiftRefused) isBeginShapeShiftingResult()  {}

// Begin shape-shifts a target. The checks run in this order and only the first failure is
// reported: a Dead target (TargetIsDead); Shapechange on another creature
// (ShapechangeNotSelf); an unwilling target of Animal Shapes or Shapechange
// (TargetNotWilling); an ineligible form (FormIneligible, every reason); then the saving
// throw, where Unwilling with a successful save gives BeginShapeShiftResisted and Willing or a
// failed save shape-shifts. When the target already holds a transformation, it ends first
// with cause Replaced. A form that grants Temporary Hit Points equal to its printed average
// Hit Points calls HitPointStatus.GainTemporary with the command's choice.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil result, for an invalid spell,
// id, form, response or Hit Point side, an UntrackedHitPoints where Temporary Hit Points are
// granted, an Equipment that is not DropsToGround or ResizesToFit for Shapechange or is set for
// another spell, or an Earlier transformation of another target. It never panics and changes
// none of its arguments.
func (s ShapeShiftSpells) Begin(c BeginShapeShiftCommand) (BeginShapeShiftingResult, error) {
	grants := grantsTemporaryHitPoints(c.Spell, c.Mode, c.Terms)
	if err := c.validate(grants); err != nil {
		return nil, err
	}
	eligibility, err := s.Eligibility(FormQuery{Spell: c.Spell, Mode: c.Mode, Terms: c.Terms, Form: c.Form, Rating: c.Rating,
		HasSeenKind: c.HasSeenKind, Object: c.Object, TargetSize: c.TargetSize})
	if err != nil {
		return nil, err
	}
	unwilling, isUnwilling := c.Response.(Unwilling)
	switch {
	case targetIsDead(c.HitPoints):
		return BeginShapeShiftRefused{TargetIsDead{}}, nil
	case c.Spell == ShapechangeSpell && c.Target != c.Caster:
		return BeginShapeShiftRefused{ShapechangeNotSelf{}}, nil
	case isUnwilling && (c.Spell == AnimalShapesSpell || c.Spell == ShapechangeSpell):
		return BeginShapeShiftRefused{TargetNotWilling{}}, nil
	}
	if ineligible, bad := eligibility.(FormIneligible); bad {
		return BeginShapeShiftRefused{ineligible}, nil
	}
	if isUnwilling && unwilling.Save == successfulSave {
		return BeginShapeShiftResisted{}, nil
	}
	return s.apply(c, grants)
}

// apply shape-shifts a target the checks of Begin allow.
func (s ShapeShiftSpells) apply(c BeginShapeShiftCommand, grants bool) (BeginShapeShiftingResult, error) {
	effect, err := SpellSource(c.Spell.SpellID())
	if err != nil {
		return nil, err
	}
	hp := c.HitPoints
	var events []Event
	if !c.Earlier.IsZero() {
		status, ended, err := endTransformation(c.Earlier, CauseReplaced, hp)
		if err != nil {
			return nil, err
		}
		events = ended
		if tracked, ok := hp.(TrackedHitPoints); ok {
			tracked.Status = status
			hp = tracked
		}
	}
	status := statusOf(hp)
	events = append(events, ShapeShiftBegan{shapeShiftEventBase{c.Target, c.Caster, c.Spell}})
	temporary, gained := TemporaryNotGranted, false
	if grants {
		tracked := hp.(TrackedHitPoints) // validate refused UntrackedHitPoints
		gain, err := status.GainTemporary(c.Target, c.Form.(CreatureForm).facts.hitPoints, tracked.Choice)
		if err != nil {
			return nil, err
		}
		done, ok := gain.(HitPointTransitionApplied)
		if !ok {
			return BeginShapeShiftRefused{TargetIsDead{}}, nil
		}
		status = done.Status()
		for _, e := range done.Events() {
			events = append(events, e)
		}
		gained = len(done.Events()) > 0
		temporary = TemporaryDeclined
		if gained {
			temporary = TemporaryFromForm
		}
	}
	fields := ActiveTransformationFields{Spell: c.Spell, Target: c.Target, Caster: c.Caster, Form: c.Form,
		MaintainedBy: MaintainedByConcentration, TemporaryHitPoints: temporary}
	if c.Spell == TruePolymorphSpell {
		fields.Mode = c.Mode
	}
	if c.Spell == ShapechangeSpell {
		fields.Terms = c.Terms
	}
	if c.Spell == AnimalShapesSpell || (c.Spell == ShapechangeSpell && c.Terms == StatBlockShapechange) {
		fields.MaintainedBy = NotMaintained
	}
	transformation, err := NewActiveTransformation(fields)
	if err != nil {
		return nil, err
	}
	restrictions, _ := s.RestrictionsOf(c.Spell, fields.Mode, c.HoldsSpellcastingFeature) // the spell and mode are valid
	statistics, _ := s.StatisticsOf(c.Spell, fields.Mode)
	r := BeginShapeShiftApplied{transformation: transformation, status: status, events: events, effect: effect,
		equipment: equipmentOf(c.Spell, fields.Mode, c.Equipment), restrictions: restrictions, statistics: statistics, gained: gained}
	if fields.Mode == ObjectIntoCreature {
		r.animated = AnimatedObjectTerms{c.Caster}
	}
	return r, nil
}

// equipmentOf returns what becomes of the equipment: the caster's choice for Shapechange,
// transforming with the creature for a creature turned into an object, melding otherwise.
func equipmentOf(spell ShapeShiftSpell, mode TrueTransformation, chosen EquipmentOutcome) EquipmentOutcome {
	switch {
	case spell == ShapechangeSpell:
		return chosen
	case spell == TruePolymorphSpell && mode == CreatureIntoObject:
		return TransformsWithCreature
	}
	return MeldsUnusable
}

// validate rejects a command no state of the rules could make sense of.
func (c BeginShapeShiftCommand) validate(grants bool) error {
	if !c.Spell.Valid() || c.Target.IsZero() || c.Caster.IsZero() || !validTargetHitPoints(c.HitPoints) {
		return invalidInput("Begin needs a spell, a target, a caster and Hit Points")
	}
	switch r := c.Response.(type) {
	case Willing:
	case Unwilling:
		if !validSave(r.Save) || (c.Spell == TruePolymorphSpell && c.Mode == ObjectIntoCreature) {
			return invalidInput("an unwilling response needs a save outcome and an object has no response")
		}
	default:
		return invalidInput("Begin needs a response")
	}
	if tracked, ok := c.HitPoints.(TrackedHitPoints); ok && grants && !tracked.Choice.Valid() {
		return invalidInput("Temporary Hit Points need a choice")
	}
	if _, untracked := c.HitPoints.(UntrackedHitPoints); untracked && grants {
		return invalidInput("this form grants Temporary Hit Points, so the Hit Points must be tracked")
	}
	if shapechange := c.Spell == ShapechangeSpell; (shapechange && !validShapechangeChoice(c.Equipment)) || (!shapechange && c.Equipment != 0) {
		return invalidInput("only Shapechange takes an equipment choice, DropsToGround or ResizesToFit")
	}
	if !c.Earlier.IsZero() {
		if _, untracked := c.HitPoints.(UntrackedHitPoints); c.Earlier.target != c.Target || (untracked && c.Earlier.temporary == TemporaryFromForm) {
			return invalidInput("the earlier transformation is another target's or needs tracked Hit Points")
		}
	}
	return nil
}

// validShapechangeChoice reports whether o is a choice Shapechange offers.
func validShapechangeChoice(o EquipmentOutcome) bool { return o == DropsToGround || o == ResizesToFit }
