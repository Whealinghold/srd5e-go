package combat

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// ActiveTransformation is one shape-shifting spell on one target: the value the host holds
// per transformed creature, keyed by the target's CombatantID (shape-changing-spells.md
// FR-062). It is immutable and comparable with ==. The zero value is "no transformation";
// IsZero reports it. Build one with NewActiveTransformation; the operations of
// ShapeShiftSpells return the new state. Safe for concurrent use.
type ActiveTransformation struct {
	spell      ShapeShiftSpell
	mode       TrueTransformation
	terms      ShapechangeTerms
	target     CombatantID
	caster     CombatantID
	form       AssumedForm
	maintained MaintainedBy
	permanent  bool
	temporary  FormTemporaryHitPoints
}

// ActiveTransformationFields are the parts of an ActiveTransformation. Construct it with a
// keyed literal.
type ActiveTransformationFields struct {
	// Spell is the spell.
	Spell ShapeShiftSpell
	// Mode is the mode of True Polymorph and must be the zero value for any other spell.
	Mode TrueTransformation
	// Terms are the terms of Shapechange (the zero value is StandardShapechange).
	Terms ShapechangeTerms
	// Target is the transformed creature (the host's id for an object turned into a creature).
	Target CombatantID
	// Caster is the creature that cast the spell.
	Caster CombatantID
	// Form is the form assumed.
	Form AssumedForm
	// MaintainedBy says whether the caster's Concentration keeps the spell.
	MaintainedBy MaintainedBy
	// Permanent is whether the spell lasts until dispelled (True Polymorph only).
	Permanent bool
	// TemporaryHitPoints is where the Temporary Hit Points came from.
	TemporaryHitPoints FormTemporaryHitPoints
}

// ActiveTransformationProblem is one reason an ActiveTransformation could not be built. The
// zero value is not a problem.
type ActiveTransformationProblem int

// ActiveTransformationError reports one problem of NewActiveTransformation. It matches
// srderr.ErrInvalidInput with errors.Is, and another *ActiveTransformationError with the same
// Problem; it carries no display text.
type ActiveTransformationError struct{ Problem ActiveTransformationProblem }

// Error renders the error. The text is not a contract.
func (e *ActiveTransformationError) Error() string {
	return "combat: invalid transformation: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ActiveTransformationError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ActiveTransformationError with the same Problem.
func (e *ActiveTransformationError) Is(target error) bool {
	t, ok := target.(*ActiveTransformationError)
	return ok && t != nil && e != nil && t.Problem == e.Problem
}

// The problems NewActiveTransformation reports, in the order listed.
const (
	TransformationInvalidSpell ActiveTransformationProblem = iota + 1
	TransformationInvalidMode
	TransformationInvalidTerms
	TransformationBlankTarget
	TransformationBlankCaster
	TransformationNoForm
	TransformationInvalidMaintainedBy
	TransformationInvalidProvenance
	// TransformationPermanentNotTruePolymorph: permanent on a spell other than True Polymorph.
	TransformationPermanentNotTruePolymorph
	// TransformationTemporaryHitPointsNotGranted: FromForm provenance on a spell or mode that grants none.
	TransformationTemporaryHitPointsNotGranted
	// TransformationConcentrationNotAllowed: Concentration on Animal Shapes or a stat block Shapechange.
	TransformationConcentrationNotAllowed
	// TransformationShapechangeTargetNotCaster: a Shapechange whose target differs from its caster.
	TransformationShapechangeTargetNotCaster
	// TransformationObjectFormNotTruePolymorph: an object form outside True Polymorph.
	TransformationObjectFormNotTruePolymorph
)

// String returns the snake_case problem, or "unknown".
func (p ActiveTransformationProblem) String() string {
	return enumName([]string{"invalid_spell", "invalid_mode", "invalid_terms", "blank_target", "blank_caster", "no_form",
		"invalid_maintained_by", "invalid_provenance", "permanent_not_true_polymorph", "temporary_hit_points_not_granted",
		"concentration_not_allowed", "shapechange_target_not_caster", "object_form_not_true_polymorph"}, int(p))
}

// grantsTemporaryHitPoints reports whether the spell, mode and terms grant the form's Hit
// Points as Temporary Hit Points (SRD 5.2.1, "True Polymorph", p. 170; "Adult Brass Dragon", p. 269).
func grantsTemporaryHitPoints(spell ShapeShiftSpell, mode TrueTransformation, terms ShapechangeTerms) bool {
	switch spell {
	case PolymorphSpell, AnimalShapesSpell:
		return true
	case TruePolymorphSpell:
		return mode == CreatureIntoCreature
	case ShapechangeSpell:
		return terms == StandardShapechange
	}
	return false
}

// NewActiveTransformation returns the transformation of f.
//
// It returns errors.Join of every violated problem (typed ActiveTransformationProblem
// values, each matching srderr.ErrInvalidInput), in the fixed order of the enum: an invalid
// part, permanent on any spell but True Polymorph, FromForm provenance where none is
// granted, Concentration on Animal Shapes or a stat block Shapechange, a Shapechange whose
// target differs from its caster, an object form outside True Polymorph. It never panics.
func NewActiveTransformation(f ActiveTransformationFields) (ActiveTransformation, error) {
	var problems []ActiveTransformationProblem
	add := func(bad bool, p ActiveTransformationProblem) {
		if bad {
			problems = append(problems, p)
		}
	}
	truePolymorph := f.Spell == TruePolymorphSpell
	_, isObject := f.Form.(ObjectForm)
	add(!f.Spell.Valid(), TransformationInvalidSpell)
	add((truePolymorph && !f.Mode.Valid()) || (!truePolymorph && f.Mode != 0) ||
		(truePolymorph && f.Form != nil && isObject != (f.Mode == CreatureIntoObject)), TransformationInvalidMode)
	add(!f.Terms.Valid() || (f.Terms != StandardShapechange && f.Spell != ShapechangeSpell), TransformationInvalidTerms)
	add(f.Target.IsZero(), TransformationBlankTarget)
	add(f.Caster.IsZero(), TransformationBlankCaster)
	add(f.Form == nil || f.Form.Size() == 0, TransformationNoForm)
	add(!f.MaintainedBy.Valid(), TransformationInvalidMaintainedBy)
	add(!f.TemporaryHitPoints.Valid(), TransformationInvalidProvenance)
	add(f.Permanent && !truePolymorph, TransformationPermanentNotTruePolymorph)
	add(f.TemporaryHitPoints == TemporaryFromForm && f.Spell.Valid() && !grantsTemporaryHitPoints(f.Spell, f.Mode, f.Terms),
		TransformationTemporaryHitPointsNotGranted)
	add(f.MaintainedBy == MaintainedByConcentration &&
		(f.Spell == AnimalShapesSpell || (f.Spell == ShapechangeSpell && f.Terms == StatBlockShapechange)), TransformationConcentrationNotAllowed)
	add(f.Spell == ShapechangeSpell && f.Target != f.Caster, TransformationShapechangeTargetNotCaster)
	add(isObject && !truePolymorph, TransformationObjectFormNotTruePolymorph)
	if len(problems) > 0 {
		errs := make([]error, len(problems))
		for i, p := range problems {
			errs[i] = &ActiveTransformationError{p}
		}
		return ActiveTransformation{}, errors.Join(errs...)
	}
	return ActiveTransformation{f.Spell, f.Mode, f.Terms, f.Target, f.Caster, f.Form, f.MaintainedBy, f.Permanent, f.TemporaryHitPoints}, nil
}

// MustActiveTransformation is NewActiveTransformation for test literals. It panics with the
// error NewActiveTransformation returns, and only then.
func MustActiveTransformation(f ActiveTransformationFields) ActiveTransformation {
	t, err := NewActiveTransformation(f)
	if err != nil {
		panic(err)
	}
	return t
}

// IsZero reports whether t is the zero value: no transformation.
func (t ActiveTransformation) IsZero() bool { return t.spell == 0 }

// Spell returns the spell.
func (t ActiveTransformation) Spell() ShapeShiftSpell { return t.spell }

// Mode returns the mode of True Polymorph, and the zero value for the other spells.
func (t ActiveTransformation) Mode() TrueTransformation { return t.mode }

// Terms returns the terms of Shapechange.
func (t ActiveTransformation) Terms() ShapechangeTerms { return t.terms }

// Parties returns the transformed creature and the creature that cast the spell. (No function
// of the package returns a bare CombatantID, so ids are never generated.)
func (t ActiveTransformation) Parties() ShapeShiftParties {
	return ShapeShiftParties{t.target, t.caster}
}

// Form returns the form assumed.
func (t ActiveTransformation) Form() AssumedForm { return t.form }

// MaintainedBy returns whether the caster's Concentration keeps the spell.
func (t ActiveTransformation) MaintainedBy() MaintainedBy { return t.maintained }

// IsPermanent reports whether the spell lasts until dispelled.
func (t ActiveTransformation) IsPermanent() bool { return t.permanent }

// TemporaryHitPoints returns where the Temporary Hit Points of the form came from.
func (t ActiveTransformation) TemporaryHitPoints() FormTemporaryHitPoints { return t.temporary }

// RecordTemporaryHitPointsReplaced returns t with the provenance Replaced when the host
// reports that another effect's Temporary Hit Points took the place of the form's (SRD 5.2.1,
// "They Don't Stack", p. 18); it leaves t unchanged for any other provenance, and for the
// zero value. It never fails or panics.
func (ShapeShiftSpells) RecordTemporaryHitPointsReplaced(t ActiveTransformation) ActiveTransformation {
	if t.temporary == TemporaryFromForm {
		t.temporary = TemporaryReplaced
	}
	return t
}

// ShapeShiftParties are the two creatures of a transformation: the one shifted and the one that
// cast the spell (the same creature for Shapechange). Comparable.
type ShapeShiftParties struct{ Target, Caster CombatantID }
