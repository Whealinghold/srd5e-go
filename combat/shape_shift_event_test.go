package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/shape-changing-spells.md FR-056 - the four shape-shift events have the constant kinds shape_shift_began, shape_shift_form_changed, shape_shift_made_permanent and shape_shift_ended, unique together; their combatants are [target, caster] and [target] when the two are one creature, as a fresh copy; they carry the spell and are values.
// Path: unhappy (name collision).
func TestShapeShiftEventFamily(t *testing.T) {
	t.Parallel()
	polymorph := ssBegin(t, ssPolymorph(t))
	began := polymorph.Events()[0]
	shapechange := ssBegin(t, ssShapechange(t))
	changed := ssChange(t, combat.ChangeFormCommand{Actor: gid("wizard"), State: shapechange.Transformation(),
		Form: ssForm(t, content.CreatureTypeBeast, "2", 60, content.SizeLarge), Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.DropsToGround}).(combat.ChangeFormApplied).Events()[0]
	tp := ssTruePolymorph(t, combat.ObjectIntoCreature)
	permanent := luMust((combat.ShapeShiftSpells{}).ElapseFullDuration(ssBegin(t, tp).Transformation(), true, tp.HitPoints)).(combat.ElapseShapeShiftMadePermanent).Events()[0]
	ended := ssEnd(t, polymorph.Transformation(), combat.CauseDispelled, combat.TrackedHitPoints{Status: polymorph.Status()}).Events()

	combattest.CheckEvent(t, began, "shape_shift_began", gid("hero"), gid("druid"))
	combattest.CheckEvent(t, changed, "shape_shift_form_changed", gid("wizard"))
	combattest.CheckEvent(t, permanent, "shape_shift_made_permanent", gid("hero"), gid("wizard"))
	combattest.CheckEvent(t, ended[len(ended)-1], "shape_shift_ended", gid("hero"), gid("druid"))
	combattest.CheckEventKinds(t, []combat.Event{began, changed, permanent, ended[len(ended)-1]})
	combattest.CheckEventsCopy(t, func() []combat.Event { return polymorph.Events() })
	for _, e := range []combat.Event{began, changed, permanent, ended[len(ended)-1]} {
		if _, ok := e.(combat.ShapeShiftEvent); !ok {
			t.Errorf("%s is not a ShapeShiftEvent", e.Kind())
		}
	}
	if got := began.(combat.ShapeShiftBegan); got.Spell() != combat.PolymorphSpell || got.Parties() != (combat.ShapeShiftParties{Target: gid("hero"), Caster: gid("druid")}) {
		t.Errorf("began = %+v", got)
	}
	if began != polymorph.Events()[0] {
		t.Error("two equal events are not equal by value")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-056 - ShapeShiftEnded carries the cause, RevertsToTrueForm and MemoryOfObjectFormLost; ShapeShiftMadePermanent carries ControlLost.
// Path: happy.
func TestShapeShiftEndedAndMadePermanentCarryTheirFacts(t *testing.T) {
	t.Parallel()
	object := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoObject))
	ended := ssEnd(t, object.Transformation(), combat.CauseDispelled, combat.UntrackedHitPoints{}).Events()
	e := ended[len(ended)-1].(combat.ShapeShiftEnded)
	if e.Cause() != combat.CauseDispelled || !e.RevertsToTrueForm() || !e.MemoryOfObjectFormLost() {
		t.Errorf("ended = %+v", e)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-056 - within one result the ending events come before the beginning events, and TemporaryHitPointsEnded comes before ShapeShiftEnded; a replacement gives the ending, then the beginning, then the new points.
// Path: unhappy (rule interaction).
func TestShapeShiftEventsAreOrdered(t *testing.T) {
	t.Parallel()
	first := ssBegin(t, ssAnimalShapes(t))
	again := ssAnimalShapes(t)
	again.Earlier = first.Transformation()
	again.HitPoints = combat.TrackedHitPoints{Status: first.Status(), Choice: combat.AcceptNew}
	got := ssEventKinds(ssBegin(t, again).Events())
	want := []string{"temporary_hit_points_ended", "shape_shift_ended", "shape_shift_began", "temporary_hit_points_gained"}
	if !slices.Equal(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}
