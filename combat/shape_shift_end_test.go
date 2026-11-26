package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ssPermanent is a True Polymorph in the given mode held for the full duration, with the status and Hit Points it ended with.
func ssPermanent(tb testing.TB, mode combat.TrueTransformation) (combat.ActiveTransformation, combat.TargetHitPoints) {
	tb.Helper()
	c := ssTruePolymorph(tb, mode)
	res, err := (combat.ShapeShiftSpells{}).ElapseFullDuration(ssBegin(tb, c).Transformation(), true, c.HitPoints)
	if err != nil {
		tb.Fatal(err)
	}
	return res.(combat.ElapseShapeShiftMadePermanent).Transformation(), c.HitPoints
}

func ssEndResult(tb testing.TB, c combat.EndShapeShiftCommand) combat.EndShapeShiftingResult {
	tb.Helper()
	res, err := (combat.ShapeShiftSpells{}).End(c)
	if err != nil {
		tb.Fatalf("End: %v", err)
	}
	return res
}

// Requirement: per case (table X of doc/specs/shape-changing-spells.md, rows of End; each case is one row)
// Path: per case.
func TestEndByCause(t *testing.T) {
	t.Parallel()
	polymorph := ssBegin(t, ssPolymorph(t))
	animal := ssBegin(t, ssAnimalShapes(t))
	statBlockCmd := ssShapechange(t)
	statBlockCmd.Terms, statBlockCmd.Form = combat.StatBlockShapechange, ssForm(t, content.CreatureTypeBeast, "5", 40, content.SizeLarge)
	statBlock := ssBegin(t, statBlockCmd)
	shapechange := ssBegin(t, ssShapechange(t))
	permanent, permanentHP := ssPermanent(t, combat.CreatureIntoCreature)
	objectCreature := ssBegin(t, ssTruePolymorph(t, combat.ObjectIntoCreature))
	creatureObject := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoObject))
	tracked := func(a combat.BeginShapeShiftApplied) combat.TargetHitPoints {
		return combat.TrackedHitPoints{Status: a.Status(), Choice: combat.AcceptNew}
	}
	deadHP := combat.TrackedHitPoints{Status: deadStatus(t, combat.DeathSavingThrowFailures)}
	type outcome int
	const (
		ends outcome = iota
		unaffected
		cannotEnd
		targetDead
	)
	cases := []struct {
		name, requirement, path string
		t                       combat.ActiveTransformation
		cause                   combat.ShapeShiftEndCause
		by                      combat.CombatantID
		hp                      combat.TargetHitPoints
		want                    outcome
	}{
		{"caster stops Concentration", "FR-049", "happy", polymorph.Transformation(), combat.CauseConcentrationEnded, gid("druid"), tracked(polymorph), ends},
		{"another creature's Concentration", "FR-049", "unhappy (rule interaction)", polymorph.Transformation(), combat.CauseConcentrationEnded, gid("other"), tracked(polymorph), unaffected},
		{"not maintained, Animal Shapes", "FR-049", "unhappy (rule interaction)", animal.Transformation(), combat.CauseConcentrationEnded, gid("druid"), tracked(animal), unaffected},
		{"stat block Shapechange", "FR-025, FR-049", "unhappy (rule interaction)", statBlock.Transformation(), combat.CauseConcentrationEnded, gid("wizard"), tracked(statBlock), unaffected},
		{"duration, Polymorph", "FR-045", "happy", polymorph.Transformation(), combat.CauseDurationElapsed, "", tracked(polymorph), ends},
		{"duration, permanent", "FR-046", "unhappy (state error)", permanent, combat.CauseDurationElapsed, "", permanentHP, unaffected},
		{"permanent, Concentration", "FR-046", "unhappy (state error)", permanent, combat.CauseConcentrationEnded, gid("wizard"), permanentHP, unaffected},
		{"permanent, dispelled", "FR-046, FR-047", "happy", permanent, combat.CauseDispelled, "", permanentHP, ends},
		{"permanent, replaced", "FR-046", "unhappy (rule interaction)", permanent, combat.CauseReplaced, "", permanentHP, ends},
		{"permanent, form temp", "FR-046", "unhappy (state error)", permanent, combat.CauseFormHitPointsDepleted, "", permanentHP, unaffected},
		{"permanent, ended by target", "FR-046", "unhappy (state error)", permanent, combat.CauseEndedByTarget, "", permanentHP, unaffected},
		{"permanent, dies", "FR-048", "unhappy (state error)", permanent, combat.CauseTargetDied, "", permanentHP, ends},
		{"permanent, destroyed", "FR-046", "unhappy (state error)", permanent, combat.CauseTargetDestroyed, "", permanentHP, ends},
		{"object-turned-creature destroyed", "FR-048, FR-062", "unhappy (state error)", objectCreature.Transformation(), combat.CauseTargetDestroyed, "", combat.UntrackedHitPoints{}, ends},
		{"death, Shapechange", "FR-048", "unhappy (state error)", shapechange.Transformation(), combat.CauseTargetDied, "", combat.TrackedHitPoints{Status: deadStatus(t, combat.MassiveDamage)}, ends},
		{"Bonus Action, Animal Shapes", "FR-050", "happy", animal.Transformation(), combat.CauseEndedByTarget, "", tracked(animal), ends},
		{"Bonus Action, Polymorph", "FR-050", "unhappy (state error)", polymorph.Transformation(), combat.CauseEndedByTarget, "", tracked(polymorph), cannotEnd},
		{"Bonus Action, Shapechange", "FR-050", "unhappy (state error)", shapechange.Transformation(), combat.CauseEndedByTarget, "", tracked(shapechange), cannotEnd},
		{"dispelled, every spell", "FR-047", "happy", shapechange.Transformation(), combat.CauseDispelled, "", tracked(shapechange), ends},
		{"replaced", "FR-026", "happy", animal.Transformation(), combat.CauseReplaced, "", tracked(animal), ends},
		{"form Hit Points depleted, Polymorph", "FR-032", "happy", polymorph.Transformation(), combat.CauseFormHitPointsDepleted, "", tracked(polymorph), ends},
		{"form Hit Points depleted, Shapechange", "FR-032", "unhappy (rule interaction)", shapechange.Transformation(), combat.CauseFormHitPointsDepleted, "", tracked(shapechange), unaffected},
		{"creature into object, dispelled", "FR-051", "unhappy (rule interaction)", creatureObject.Transformation(), combat.CauseDispelled, "", combat.UntrackedHitPoints{}, ends},
		{"no transformation", "FR-052", "unhappy (empty and minimal)", combat.ActiveTransformation{}, combat.CauseDispelled, "", tracked(polymorph), unaffected},
		{"Dead target, dispelled", "FR-052", "unhappy (state error)", polymorph.Transformation(), combat.CauseDispelled, "", deadHP, targetDead},
		{"Dead target, duration", "FR-052", "unhappy (state error)", polymorph.Transformation(), combat.CauseDurationElapsed, "", deadHP, targetDead},
		{"Dead target, dies", "FR-052", "unhappy (state error)", polymorph.Transformation(), combat.CauseTargetDied, "", deadHP, ends},
		{"Dead target, destroyed", "FR-052", "unhappy (state error)", polymorph.Transformation(), combat.CauseTargetDestroyed, "", deadHP, ends},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := ssEndResult(t, combat.EndShapeShiftCommand{State: c.t, Cause: c.cause, HitPoints: c.hp, ConcentrationOf: c.by})
			var got outcome
			switch r := res.(type) {
			case combat.EndShapeShiftApplied:
				got = ends
				if !r.Transformation().IsZero() || r.CarryOver() != (combat.CarryOverAll{}) {
					t.Errorf("Transformation = %+v, CarryOver = %v", r.Transformation(), r.CarryOver())
				}
				last := r.Events()[len(r.Events())-1]
				ev, isEnded := last.(combat.ShapeShiftEnded)
				if !isEnded || ev.Cause() != c.cause || !ev.RevertsToTrueForm() {
					t.Errorf("last event = %#v, want ShapeShiftEnded(%v) that reverts to the true form", last, c.cause)
				}
			case combat.EndShapeShiftUnaffected:
				got = unaffected
			case combat.EndShapeShiftRefused:
				got = map[combat.EndShapeShiftRejection]outcome{combat.CannotEndAtWill: cannotEnd, combat.EndTargetIsDead: targetDead}[r.Rejection]
			}
			if got != c.want {
				t.Errorf("End(%v) = %#v, want outcome %d [%s]", c.cause, res, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-050 - ending Animal Shapes at will costs a Bonus Action and ends that target's transformation alone; no other ending carries a cost.
// Path: unhappy (rule interaction).
func TestEndByTargetCostsABonusAction(t *testing.T) {
	t.Parallel()
	animal := ssBegin(t, ssAnimalShapes(t))
	byTarget := ssEnd(t, animal.Transformation(), combat.CauseEndedByTarget, combat.TrackedHitPoints{Status: animal.Status()})
	if cost, ok := byTarget.ActionCost(); !ok || cost != combat.ShapeShiftBonusAction {
		t.Errorf("ActionCost() = %v, %v, want a Bonus Action", cost, ok)
	}
	dispelled := ssEnd(t, animal.Transformation(), combat.CauseDispelled, combat.TrackedHitPoints{Status: animal.Status()})
	if cost, ok := dispelled.ActionCost(); ok || cost != 0 {
		t.Errorf("a dispel cost %v", cost)
	}
	if got := byTarget.Events()[len(byTarget.Events())-1].Combatants(); !slices.Equal(got, []combat.CombatantID{gid("hero"), gid("druid")}) {
		t.Errorf("the ending names %v, want the target and the caster", got)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-048, FR-034 - a death skips the Temporary Hit Point call: the status is unchanged and no Temporary Hit Point event is produced, even when the form's points were still held.
// Path: unhappy (state error).
func TestDeathSkipsTheTemporaryHitPointCall(t *testing.T) {
	t.Parallel()
	shapechange := ssBegin(t, ssShapechange(t))
	for _, cause := range []combat.ShapeShiftEndCause{combat.CauseTargetDied, combat.CauseTargetDestroyed} {
		ended := ssEnd(t, shapechange.Transformation(), cause, combat.TrackedHitPoints{Status: shapechange.Status()})
		if got := ssEventKinds(ended.Events()); !slices.Equal(got, []string{"shape_shift_ended"}) || ended.Status() != shapechange.Status() {
			t.Errorf("%v: events %v, status changed %v", cause, got, ended.Status() != shapechange.Status())
		}
	}
	live := ssEnd(t, shapechange.Transformation(), combat.CauseDispelled, combat.TrackedHitPoints{Status: shapechange.Status()})
	if got := ssEventKinds(live.Events()); !slices.Equal(got, []string{"temporary_hit_points_ended", "shape_shift_ended"}) {
		t.Errorf("a dispel gave %v, want the Temporary Hit Points to end first [FR-056]", got)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-051 - ending a creature turned into an object sets MemoryOfObjectFormLost; ending any other transformation does not.
// Path: per case.
func TestEndingACreatureIntoAnObjectLosesItsMemory(t *testing.T) {
	t.Parallel()
	object := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoObject))
	other := ssBegin(t, ssTruePolymorph(t, combat.ObjectIntoCreature))
	for _, c := range []struct {
		name string
		a    combat.BeginShapeShiftApplied
		want bool
	}{{"creature into object", object, true}, {"object into creature", other, false}} {
		ended := ssEnd(t, c.a.Transformation(), combat.CauseDispelled, combat.UntrackedHitPoints{})
		if got := ended.Events()[0].(combat.ShapeShiftEnded).MemoryOfObjectFormLost(); got != c.want {
			t.Errorf("%s: MemoryOfObjectFormLost = %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-043, FR-055 - End returns an error matching ErrInvalidInput and no result for an invalid cause, no Hit Point side, Concentration with no creature, or an untracked target whose transformation holds form Temporary Hit Points.
// Path: per case.
func TestEndRejectsInvalidCommands(t *testing.T) {
	t.Parallel()
	shapechange := ssBegin(t, ssShapechange(t))
	cases := []struct {
		name, requirement, path string
		cmd                     combat.EndShapeShiftCommand
	}{
		{"no cause", "FR-043", "unhappy (invalid input)", combat.EndShapeShiftCommand{State: shapechange.Transformation(), HitPoints: combat.TrackedHitPoints{Status: shapechange.Status()}}},
		{"a cause out of range", "FR-043", "unhappy (invalid input)", combat.EndShapeShiftCommand{State: shapechange.Transformation(), Cause: 9, HitPoints: combat.TrackedHitPoints{Status: shapechange.Status()}}},
		{"no Hit Point side", "FR-055", "unhappy (invalid input)", combat.EndShapeShiftCommand{State: shapechange.Transformation(), Cause: combat.CauseDispelled}},
		{"a zero status", "FR-055", "unhappy (invalid input)", combat.EndShapeShiftCommand{State: shapechange.Transformation(), Cause: combat.CauseDispelled, HitPoints: combat.TrackedHitPoints{}}},
		{"Concentration without a creature", "FR-049", "unhappy (invalid input)", combat.EndShapeShiftCommand{State: shapechange.Transformation(), Cause: combat.CauseConcentrationEnded, HitPoints: combat.TrackedHitPoints{Status: shapechange.Status()}}},
		{"untracked with form Temporary Hit Points", "FR-030", "unhappy (invalid input)", combat.EndShapeShiftCommand{State: shapechange.Transformation(), Cause: combat.CauseDispelled, HitPoints: combat.UntrackedHitPoints{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.ShapeShiftSpells{}).End(c.cmd)
			if got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("End = %#v, %v, want nil and ErrInvalidInput [%s]", got, err, c.requirement)
			}
		})
	}
}

// Requirement: per case (table X rows of ElapseFullDuration; each case is one row)
// Path: per case.
func TestElapseFullDuration(t *testing.T) {
	t.Parallel()
	tp := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoCreature))
	object := ssBegin(t, ssTruePolymorph(t, combat.ObjectIntoCreature))
	polymorph := ssBegin(t, ssPolymorph(t))
	shapechange := ssBegin(t, ssShapechange(t))
	tracked := func(a combat.BeginShapeShiftApplied) combat.TargetHitPoints {
		return combat.TrackedHitPoints{Status: a.Status()}
	}
	permanent, permanentHP := ssPermanent(t, combat.CreatureIntoCreature)
	cases := []struct {
		name, requirement, path string
		t                       combat.ActiveTransformation
		held                    bool
		hp                      combat.TargetHitPoints
		want                    string // permanent, controlLost, ended, unaffected
	}{
		{"True Polymorph held", "FR-045", "unhappy (boundary)", tp.Transformation(), true, tracked(tp), "permanent"},
		{"True Polymorph lost", "FR-045", "unhappy (boundary)", tp.Transformation(), false, tracked(tp), "ended"},
		{"permanent object to creature", "FR-046", "unhappy (rule interaction)", object.Transformation(), true, combat.UntrackedHitPoints{}, "controlLost"},
		{"permanent creature to creature", "FR-046", "unhappy (rule interaction)", tp.Transformation(), true, tracked(tp), "permanent"},
		{"Polymorph always ends", "FR-045", "happy", polymorph.Transformation(), true, tracked(polymorph), "ended"},
		{"Shapechange always ends", "FR-045", "unhappy (rule interaction)", shapechange.Transformation(), true, tracked(shapechange), "ended"},
		{"already permanent", "FR-046", "unhappy (state error)", permanent, true, permanentHP, "unaffected"},
		{"no transformation", "FR-052", "unhappy (empty and minimal)", combat.ActiveTransformation{}, true, tracked(tp), "unaffected"},
		{"Dead target", "FR-053", "unhappy (state error)", tp.Transformation(), true, combat.TrackedHitPoints{Status: deadStatus(t, combat.MassiveDamage)}, "unaffected"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := (combat.ShapeShiftSpells{}).ElapseFullDuration(c.t, c.held, c.hp)
			if err != nil {
				t.Fatal(err)
			}
			got := "unaffected"
			switch r := res.(type) {
			case combat.ElapseShapeShiftMadePermanent:
				got = "permanent"
				ev := r.Events()
				made, ok := ev[0].(combat.ShapeShiftMadePermanent)
				if len(ev) != 1 || !ok || !r.Transformation().IsPermanent() {
					t.Fatalf("events %v, permanent %v", ssEventKinds(ev), r.Transformation().IsPermanent())
				}
				if made.ControlLost() {
					got = "controlLost"
				}
			case combat.ElapseShapeShiftEnded:
				got = "ended"
				last := r.Ended().Events()
				if e := last[len(last)-1].(combat.ShapeShiftEnded); e.Cause() != combat.CauseDurationElapsed {
					t.Errorf("cause = %v", e.Cause())
				}
			}
			if got != c.want {
				t.Errorf("ElapseFullDuration = %#v, want %s [%s]", res, c.want, c.requirement)
			}
		})
	}
	if _, err := (combat.ShapeShiftSpells{}).ElapseFullDuration(tp.Transformation(), true, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ElapseFullDuration(no Hit Points) error = %v", err)
	}
	if _, err := (combat.ShapeShiftSpells{}).ElapseFullDuration(tp.Transformation(), false, combat.UntrackedHitPoints{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ElapseFullDuration(untracked, form temp) error = %v", err)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-055 - End, ElapseFullDuration and Assess change none of their inputs and give equal results for equal inputs; the event lists are copies.
// Path: unhappy (immutability).
func TestEndingOperationsArePure(t *testing.T) {
	t.Parallel()
	begun := ssBegin(t, ssAnimalShapes(t))
	cmd := combat.EndShapeShiftCommand{State: begun.Transformation(), Cause: combat.CauseDispelled, HitPoints: combat.TrackedHitPoints{Status: begun.Status()}}
	before := cmd
	a, b := ssEndResult(t, cmd), ssEndResult(t, cmd)
	if !reflect.DeepEqual(cmd, before) || !reflect.DeepEqual(a, b) {
		t.Error("End changed its command or gave unequal results")
	}
	events := a.(combat.EndShapeShiftApplied).Events()
	events[0] = nil
	if a.(combat.EndShapeShiftApplied).Events()[0] == nil {
		t.Error("Events() returned a view")
	}
}

// Requirement: per case (doc/specs/shape-changing-spells.md Q-3, FR-045, FR-048, FR-050 - a living target at 0 current Hit Points can still be released: End by the causes that apply and ElapseFullDuration work as for any other living target; SRD 5.2.1, "Shape-Shifting", p. 187)
// Path: per case.
func TestEndAndElapseForALivingTargetAtZeroHitPoints(t *testing.T) {
	t.Parallel()
	zero := combat.TrackedHitPoints{Status: ssStableAtZero(t, 0), Choice: combat.AcceptNew}
	dying := combat.TrackedHitPoints{Status: dyingStatus(t, 30, 0, 1), Choice: combat.AcceptNew}
	endCases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
		cause                   combat.ShapeShiftEndCause
		hp                      combat.TargetHitPoints
	}{
		{"Polymorph dispelled at 0 Hit Points", "FR-047", "unhappy (boundary)", ssPolymorph(t), combat.CauseDispelled, zero},
		{"Polymorph form points depleted at 0 Hit Points", "FR-032", "unhappy (boundary)", ssPolymorph(t), combat.CauseFormHitPointsDepleted, zero},
		{"Shapechange dispelled while Dying", "FR-047", "unhappy (state error)", ssShapechange(t), combat.CauseDispelled, dying},
		{"Animal Shapes ended by the target at 0 Hit Points", "FR-050", "unhappy (boundary)", ssAnimalShapes(t), combat.CauseEndedByTarget, zero},
		{"True Polymorph replaced while Dying", "FR-026", "unhappy (state error)", ssTruePolymorph(t, combat.CreatureIntoCreature), combat.CauseReplaced, dying},
	}
	for _, c := range endCases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := ssBegin(t, c.cmd)
			res := ssEndResult(t, combat.EndShapeShiftCommand{State: a.Transformation(), Cause: c.cause, HitPoints: c.hp, ConcentrationOf: gid("druid")})
			applied, ok := res.(combat.EndShapeShiftApplied)
			if !ok {
				t.Fatalf("End(%v) = %#v, want EndShapeShiftApplied [%s]", c.cause, res, c.requirement)
			}
			last := applied.Events()[len(applied.Events())-1]
			if ev, isEnded := last.(combat.ShapeShiftEnded); !isEnded || ev.Cause() != c.cause {
				t.Errorf("last event = %#v, want ShapeShiftEnded(%v) [%s]", last, c.cause, c.requirement)
			}
		})
	}
	tp := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoCreature))
	polymorph := ssBegin(t, ssPolymorph(t))
	t.Run("True Polymorph held to the full duration at 0 Hit Points is permanent", func(t *testing.T) {
		t.Parallel()
		res, err := (combat.ShapeShiftSpells{}).ElapseFullDuration(tp.Transformation(), true, zero)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := res.(combat.ElapseShapeShiftMadePermanent); !ok {
			t.Errorf("ElapseFullDuration = %#v, want made permanent [FR-045]", res)
		}
	})
	t.Run("Polymorph at the full duration while Dying ends", func(t *testing.T) {
		t.Parallel()
		res, err := (combat.ShapeShiftSpells{}).ElapseFullDuration(polymorph.Transformation(), true, dying)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := res.(combat.ElapseShapeShiftEnded); !ok {
			t.Errorf("ElapseFullDuration = %#v, want ended [FR-045]", res)
		}
	})
}
