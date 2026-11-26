package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ssFields are the parts of a valid Polymorph transformation.
func ssFields(tb testing.TB) combat.ActiveTransformationFields {
	tb.Helper()
	return combat.ActiveTransformationFields{Spell: combat.PolymorphSpell, Target: gid("hero"), Caster: gid("druid"), Form: ssBeast(tb, "1", 11),
		MaintainedBy: combat.MaintainedByConcentration, TemporaryHitPoints: combat.TemporaryFromForm}
}

// Requirement: doc/specs/shape-changing-spells.md FR-054 - NewActiveTransformation accepts a valid set of parts and exposes them; the value is comparable and equal by value.
// Path: happy.
func TestNewActiveTransformationExposesItsParts(t *testing.T) {
	t.Parallel()
	f := ssFields(t)
	a, err := combat.NewActiveTransformation(f)
	if err != nil {
		t.Fatal(err)
	}
	b := combat.MustActiveTransformation(f)
	if a != b || a.IsZero() || a.Spell() != f.Spell || a.Parties() != (combat.ShapeShiftParties{Target: f.Target, Caster: f.Caster}) || a.Form() != f.Form ||
		a.MaintainedBy() != f.MaintainedBy || a.TemporaryHitPoints() != f.TemporaryHitPoints || a.IsPermanent() || a.Mode() != 0 || a.Terms() != combat.StandardShapechange {
		t.Errorf("transformation = %+v", a)
	}
	if !(combat.ActiveTransformation{}).IsZero() {
		t.Error("the zero transformation is not zero")
	}
}

// Requirement: per case (FR-054, one violation of the constructor per case; each error matches ErrInvalidInput and holds exactly the listed typed problems)
// Path: per case.
func TestNewActiveTransformationRejectsEachViolation(t *testing.T) {
	t.Parallel()
	mut := func(f func(*combat.ActiveTransformationFields)) combat.ActiveTransformationFields {
		fields := ssFields(t)
		f(&fields)
		return fields
	}
	statBlock := func(f *combat.ActiveTransformationFields) {
		f.Spell, f.Terms, f.Target, f.Caster = combat.ShapechangeSpell, combat.StatBlockShapechange, gid("a"), gid("a")
	}
	cases := []struct {
		name, requirement, path string
		fields                  combat.ActiveTransformationFields
		want                    []combat.ActiveTransformationProblem
	}{
		{"permanent on Polymorph", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Permanent = true }), []combat.ActiveTransformationProblem{combat.TransformationPermanentNotTruePolymorph}},
		{"FromForm where the stat block grants none", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			statBlock(f)
			f.MaintainedBy = combat.NotMaintained
		}), []combat.ActiveTransformationProblem{combat.TransformationTemporaryHitPointsNotGranted}},
		{"FromForm where True Polymorph into an object grants none", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			f.Spell, f.Mode, f.Form = combat.TruePolymorphSpell, combat.CreatureIntoObject, ssObject(t, content.SizeMedium)
		}), []combat.ActiveTransformationProblem{combat.TransformationTemporaryHitPointsNotGranted}},
		{"Concentration on Animal Shapes", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Spell = combat.AnimalShapesSpell }), []combat.ActiveTransformationProblem{combat.TransformationConcentrationNotAllowed}},
		{"Concentration on a stat block Shapechange", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			statBlock(f)
			f.TemporaryHitPoints = combat.TemporaryNotGranted
		}), []combat.ActiveTransformationProblem{combat.TransformationConcentrationNotAllowed}},
		{"a Shapechange whose target is not the caster", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Spell = combat.ShapechangeSpell }), []combat.ActiveTransformationProblem{combat.TransformationShapechangeTargetNotCaster}},
		{"an object form outside True Polymorph", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			f.Form, f.TemporaryHitPoints = ssObject(t, content.SizeMedium), combat.TemporaryNotGranted
		}), []combat.ActiveTransformationProblem{combat.TransformationObjectFormNotTruePolymorph}},
		{"no spell", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Spell = 0 }), []combat.ActiveTransformationProblem{combat.TransformationInvalidSpell}},
		{"a mode on Polymorph", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Mode = combat.CreatureIntoCreature }), []combat.ActiveTransformationProblem{combat.TransformationInvalidMode}},
		{"True Polymorph without a mode", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			f.Spell, f.TemporaryHitPoints = combat.TruePolymorphSpell, combat.TemporaryNotGranted
		}), []combat.ActiveTransformationProblem{combat.TransformationInvalidMode}},
		{"an object form for a creature mode", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			f.Spell, f.Mode, f.Form = combat.TruePolymorphSpell, combat.CreatureIntoCreature, ssObject(t, content.SizeMedium)
		}), []combat.ActiveTransformationProblem{combat.TransformationInvalidMode}},
		{"terms on Polymorph", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Terms = combat.StatBlockShapechange }), []combat.ActiveTransformationProblem{combat.TransformationInvalidTerms}},
		{"no target", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Target = "" }), []combat.ActiveTransformationProblem{combat.TransformationBlankTarget}},
		{"no caster", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Caster = "" }), []combat.ActiveTransformationProblem{combat.TransformationBlankCaster}},
		{"no form", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.Form = nil }), []combat.ActiveTransformationProblem{combat.TransformationNoForm}},
		{"the zero object form", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) {
			f.Spell, f.Mode, f.Form, f.TemporaryHitPoints = combat.TruePolymorphSpell, combat.CreatureIntoObject, combat.ObjectForm{}, combat.TemporaryNotGranted
		}), []combat.ActiveTransformationProblem{combat.TransformationNoForm}},
		{"no maintenance", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.MaintainedBy = 0 }), []combat.ActiveTransformationProblem{combat.TransformationInvalidMaintainedBy}},
		{"no provenance", "FR-054", "unhappy (invalid input)", mut(func(f *combat.ActiveTransformationFields) { f.TemporaryHitPoints = 0 }), []combat.ActiveTransformationProblem{combat.TransformationInvalidProvenance}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewActiveTransformation(c.fields)
			if !got.IsZero() || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewActiveTransformation = %+v, %v, want the zero value and ErrInvalidInput [%s]", got, err, c.requirement)
			}
			var found []combat.ActiveTransformationProblem
			for p := combat.TransformationInvalidSpell; p <= combat.TransformationObjectFormNotTruePolymorph; p++ {
				if errors.Is(err, &combat.ActiveTransformationError{Problem: p}) {
					found = append(found, p)
				}
			}
			if !slices.Equal(found, c.want) {
				t.Errorf("problems = %v, want %v [%s]", found, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-054 - when several violations apply, the error is errors.Join of every typed problem, in the fixed order of the enum; MustActiveTransformation panics with that error.
// Path: unhappy (rule interaction).
func TestNewActiveTransformationReportsEveryProblemInOrder(t *testing.T) {
	t.Parallel()
	f := ssFields(t)
	f.Spell, f.Terms, f.Permanent, f.Form, f.Caster = combat.AnimalShapesSpell, combat.StatBlockShapechange, true, ssObject(t, content.SizeLarge), ""
	_, err := combat.NewActiveTransformation(f)
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("error %T is not an errors.Join", err)
	}
	var got []combat.ActiveTransformationProblem
	for _, e := range joined.Unwrap() {
		var pe *combat.ActiveTransformationError
		if !errors.As(e, &pe) {
			t.Fatalf("joined error %T is not a typed problem", e)
		}
		got = append(got, pe.Problem)
	}
	want := []combat.ActiveTransformationProblem{combat.TransformationInvalidTerms, combat.TransformationBlankCaster, combat.TransformationPermanentNotTruePolymorph,
		combat.TransformationConcentrationNotAllowed, combat.TransformationObjectFormNotTruePolymorph}
	if !slices.Equal(got, want) {
		t.Errorf("problems = %v, want %v", got, want)
	}
	for _, p := range got {
		if p.String() == "unknown" {
			t.Errorf("problem %d: name %q", p, p.String())
		}
	}
	defer func() {
		r := recover()
		if e, isErr := r.(error); !isErr || !errors.Is(e, srderr.ErrInvalidInput) {
			t.Errorf("MustActiveTransformation panicked with %v, want the error", r)
		}
	}()
	combat.MustActiveTransformation(f)
}

// Requirement: doc/specs/shape-changing-spells.md FR-054, FR-046 - a True Polymorph may be permanent, in any mode, and keeps its form and caster.
// Path: happy.
func TestNewActiveTransformationAcceptsAPermanentTruePolymorph(t *testing.T) {
	t.Parallel()
	for _, mode := range []combat.TrueTransformation{combat.CreatureIntoCreature, combat.ObjectIntoCreature, combat.CreatureIntoObject} {
		permanent, _ := ssPermanent(t, mode)
		if !permanent.IsPermanent() || permanent.Mode() != mode || permanent.Spell() != combat.TruePolymorphSpell {
			t.Errorf("%v: permanent = %+v", mode, permanent)
		}
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-062 - an object turned into a creature is keyed by the host's id, and a creature turned into an object keeps its own; the events name those ids.
// Path: unhappy (state error).
func TestTransformationIsKeyedByTheHostsID(t *testing.T) {
	t.Parallel()
	object := ssBegin(t, ssTruePolymorph(t, combat.ObjectIntoCreature))
	if object.Transformation().Parties().Target != gid("hero") {
		t.Errorf("Target = %q, want the host's id", object.Transformation().Parties().Target)
	}
	if got := object.Events()[0].Combatants(); !slices.Equal(got, []combat.CombatantID{gid("hero"), gid("wizard")}) {
		t.Errorf("the beginning names %v", got)
	}
	creature := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoObject))
	if creature.Transformation().Parties().Target != gid("hero") {
		t.Errorf("a creature turned into an object has the id %q", creature.Transformation().Parties().Target)
	}
	ended := ssEnd(t, object.Transformation(), combat.CauseTargetDestroyed, combat.UntrackedHitPoints{})
	if got := ended.Events()[0].Combatants()[0]; got != gid("hero") {
		t.Errorf("the destruction names %q, want the host's id", got)
	}
}
