package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// ssHurt applies amount fire damage to the target of a, as the host does before calling Assess.
func ssHurt(tb testing.TB, status combat.HitPointStatus, amount int) combat.HitPointStatus {
	tb.Helper()
	tr, err := status.ApplyDamage(gid("hero"), []combat.Damage{fire(amount)}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		tb.Fatal(err)
	}
	return tr.(combat.HitPointTransitionApplied).Status()
}

// ssAssess returns whether Assess says the transformation of a continues after the status.
func ssAssess(tb testing.TB, t combat.ActiveTransformation, status combat.HitPointStatus) combat.AssessShapeShiftResult {
	tb.Helper()
	res, err := (combat.ShapeShiftSpells{}).Assess(t, status)
	if err != nil {
		tb.Fatal(err)
	}
	return res
}

// ssEnd ends t for cause against hp and requires the ending.
func ssEnd(tb testing.TB, t combat.ActiveTransformation, cause combat.ShapeShiftEndCause, hp combat.TargetHitPoints) combat.EndShapeShiftApplied {
	tb.Helper()
	res, err := (combat.ShapeShiftSpells{}).End(combat.EndShapeShiftCommand{State: t, Cause: cause, HitPoints: hp, ConcentrationOf: t.Parties().Caster})
	if err != nil {
		tb.Fatal(err)
	}
	a, ok := res.(combat.EndShapeShiftApplied)
	if !ok {
		tb.Fatalf("End(%v) = %#v, want EndShapeShiftApplied", cause, res)
	}
	return a
}

// Requirement: per case (table T of doc/specs/shape-changing-spells.md, with the real HitPointStatus operations ApplyDamage, GainTemporary and EndTemporary; each case is one row)
// Path: per case.
func TestTemporaryHitPointsOfAForm(t *testing.T) {
	t.Parallel()
	poly := func(formHP, held int, choice combat.TemporaryHitPointsChoice) combat.BeginShapeShiftCommand {
		c := ssPolymorph(t)
		c.Form = ssBeast(t, "1", formHP)
		c.HitPoints = ssTracked(t, 20, 30, held, choice)
		return c
	}
	cases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
		wantTemp                int
		wantProvenance          combat.FormTemporaryHitPoints
		wantEvent               bool
	}{
		{"none held, accept", "FR-027, FR-028", "happy", poly(11, 0, combat.AcceptNew), 11, combat.TemporaryFromForm, true},
		{"none held, keep", "FR-027, FR-028", "happy", poly(11, 0, combat.KeepCurrent), 11, combat.TemporaryFromForm, true},
		{"more offered", "FR-028", "unhappy (rule interaction)", poly(12, 10, combat.AcceptNew), 12, combat.TemporaryFromForm, true},
		{"keep larger", "FR-028", "unhappy (rule interaction)", poly(12, 10, combat.KeepCurrent), 10, combat.TemporaryDeclined, false},
		{"equal amount, accept", "FR-059", "unhappy (boundary)", poly(12, 12, combat.AcceptNew), 12, combat.TemporaryDeclined, false},
		{"equal amount, keep", "FR-059", "unhappy (boundary)", poly(12, 12, combat.KeepCurrent), 12, combat.TemporaryDeclined, false},
		{"one more offered", "FR-028", "unhappy (boundary)", poly(13, 12, combat.AcceptNew), 13, combat.TemporaryFromForm, true},
		{"Animal Shapes", "FR-027", "happy", ssAnimalShapes(t), 20, combat.TemporaryFromForm, true},
		{"Shapechange", "FR-027", "happy", ssShapechange(t), 40, combat.TemporaryFromForm, true},
		{"True Polymorph creature into creature", "FR-027", "happy", ssTruePolymorph(t, combat.CreatureIntoCreature), 30, combat.TemporaryFromForm, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := ssBegin(t, c.cmd)
			if got := a.Status().HitPoints().Temporary(); got != c.wantTemp || a.Transformation().TemporaryHitPoints() != c.wantProvenance ||
				a.GainedTemporaryHitPointsEvent() != c.wantEvent {
				t.Errorf("temp %d, provenance %v, event %v; want %d, %v, %v [%s]", got, a.Transformation().TemporaryHitPoints(),
					a.GainedTemporaryHitPointsEvent(), c.wantTemp, c.wantProvenance, c.wantEvent, c.requirement)
			}
			if want := statusOf(c.cmd.HitPoints).HitPoints().Current(); a.Status().HitPoints().Current() != want {
				t.Errorf("current Hit Points = %d, want the retained %d", a.Status().HitPoints().Current(), want)
			}
			hasEvent := slices.Contains(ssEventKinds(a.Events()), "temporary_hit_points_gained")
			if hasEvent != c.wantEvent {
				t.Errorf("events %v, want a Temporary Hit Point event: %v", ssEventKinds(a.Events()), c.wantEvent)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-056 - ShapeShiftBegan comes before TemporaryHitPointsGained.
// Path: unhappy (rule interaction).
func TestBeginEmitsTheBeginningBeforeTheTemporaryHitPoints(t *testing.T) {
	t.Parallel()
	got := ssEventKinds(ssBegin(t, ssPolymorph(t)).Events())
	if want := []string{"shape_shift_began", "temporary_hit_points_gained"}; !slices.Equal(got, want) {
		t.Errorf("events = %v, want %v [FR-056]", got, want)
	}
}

// Requirement: per case (table T rows about damage and Assess; the real ApplyDamage takes the damage from the Temporary Hit Points first)
// Path: per case.
func TestAssessAfterDamage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
		damage                  int
		wantEnds                bool
		wantCurrent, wantTemp   int
	}{
		{"damage within temp", "FR-032", "happy", ssPolymorph(t), 7, false, 20, 4},
		{"damage exactly temp", "FR-032", "unhappy (boundary)", ssPolymorph(t), 11, true, 20, 0},
		{"damage one short of temp", "FR-032", "unhappy (boundary)", ssPolymorph(t), 10, false, 20, 1},
		{"damage past temp", "FR-033", "unhappy (rule interaction)", ssPolymorph(t), 15, true, 16, 0},
		{"declined, then 0", "FR-032", "unhappy (rule interaction)", func() combat.BeginShapeShiftCommand {
			c := ssPolymorph(t)
			c.Form, c.HitPoints = ssBeast(t, "1", 12), ssTracked(t, 20, 30, 10, combat.KeepCurrent)
			return c
		}(), 10, true, 20, 0},
		{"True Polymorph at 0 temp keeps going", "FR-032", "unhappy (rule interaction)", ssTruePolymorph(t, combat.CreatureIntoCreature), 30, false, 20, 0},
		{"Shapechange at 0 temp keeps going", "FR-032", "unhappy (rule interaction)", ssShapechange(t), 40, false, 30, 0},
		{"Animal Shapes at 0 temp keeps going", "FR-032", "unhappy (rule interaction)", ssAnimalShapes(t), 20, false, 20, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := ssBegin(t, c.cmd)
			hurt := ssHurt(t, a.Status(), c.damage)
			before := hurt
			res := ssAssess(t, a.Transformation(), hurt)
			ended, isEnded := res.(combat.AssessShapeShiftEnded)
			if isEnded != c.wantEnds || (isEnded && ended.Cause() != combat.CauseFormHitPointsDepleted) {
				t.Errorf("Assess = %#v, want ends %v [%s]", res, c.wantEnds, c.requirement)
			}
			if hurt != before || hurt.HitPoints().Current() != c.wantCurrent || hurt.HitPoints().Temporary() != c.wantTemp {
				t.Errorf("Hit Points %d, Temporary %d; want %d, %d: Assess must not change them [FR-033]", hurt.HitPoints().Current(), hurt.HitPoints().Temporary(), c.wantCurrent, c.wantTemp)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-032, FR-053 - Assess reads the actual Temporary Hit Points whatever their source, and never ends a transformation for a Dead target, a permanent or a missing one.
// Path: per case.
func TestAssessReadsTheActualTemporaryHitPoints(t *testing.T) {
	t.Parallel()
	polymorph := ssBegin(t, ssPolymorph(t)).Transformation()
	other := pcStatus(t, 20, 30, 5)
	permanent := ssTruePolymorph(t, combat.CreatureIntoCreature)
	permanentState := luMust((combat.ShapeShiftSpells{}).ElapseFullDuration(ssBegin(t, permanent).Transformation(), true, permanent.HitPoints)).(combat.ElapseShapeShiftMadePermanent).Transformation()
	cases := []struct {
		name, requirement, path string
		t                       combat.ActiveTransformation
		status                  combat.HitPointStatus
		wantEnds                bool
	}{
		{"another source's points are enough", "FR-032", "unhappy (rule interaction)", polymorph, other, false},
		{"no points at all", "FR-032", "unhappy (boundary)", polymorph, pcStatus(t, 20, 30, 0), true},
		{"a Dead target", "FR-053", "unhappy (state error)", polymorph, deadStatus(t, combat.DeathSavingThrowFailures), false},
		{"no transformation", "FR-053", "unhappy (empty and minimal)", combat.ActiveTransformation{}, pcStatus(t, 20, 30, 0), false},
		{"a permanent transformation", "FR-032", "unhappy (state error)", permanentState, pcStatus(t, 20, 30, 0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, ends := ssAssess(t, c.t, c.status).(combat.AssessShapeShiftEnded)
			if ends != c.wantEnds {
				t.Errorf("Assess ends = %v, want %v [%s]", ends, c.wantEnds, c.requirement)
			}
		})
	}
	if _, err := (combat.ShapeShiftSpells{}).Assess(polymorph, combat.HitPointStatus{}); err == nil {
		t.Error("Assess accepted the zero status")
	}
}

// Requirement: per case (table T rows about ending; the real EndTemporary)
// Path: per case.
func TestEndingEndsTheTemporaryHitPointsOfTheForm(t *testing.T) {
	t.Parallel()
	declined := ssPolymorph(t)
	declined.Form, declined.HitPoints = ssBeast(t, "1", 12), ssTracked(t, 20, 30, 10, combat.KeepCurrent)
	equal := ssPolymorph(t)
	equal.Form, equal.HitPoints = ssBeast(t, "1", 12), ssTracked(t, 20, 30, 12, combat.AcceptNew)
	cases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
		damage                  int
		cause                   combat.ShapeShiftEndCause
		wantTemp                int
		wantKinds               []string
	}{
		{"form temp held", "FR-034", "happy", ssAnimalShapes(t), 14, combat.CauseEndedByTarget, 0, []string{"temporary_hit_points_ended", "shape_shift_ended"}},
		{"declined", "FR-034", "unhappy (rule interaction)", declined, 0, combat.CauseDispelled, 10, []string{"shape_shift_ended"}},
		{"equal amount", "FR-059", "unhappy (boundary)", equal, 0, combat.CauseDispelled, 12, []string{"shape_shift_ended"}},
		{"none left to end", "FR-034", "unhappy (boundary)", ssPolymorph(t), 11, combat.CauseFormHitPointsDepleted, 0, []string{"shape_shift_ended"}},
		{"duration", "FR-034", "happy", ssShapechange(t), 0, combat.CauseDurationElapsed, 0, []string{"temporary_hit_points_ended", "shape_shift_ended"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			begun := ssBegin(t, c.cmd)
			status := begun.Status()
			if c.damage > 0 {
				status = ssHurt(t, status, c.damage)
			}
			ended := ssEnd(t, begun.Transformation(), c.cause, combat.TrackedHitPoints{Status: status})
			if got := ended.Status().HitPoints().Temporary(); got != c.wantTemp || !slices.Equal(ssEventKinds(ended.Events()), c.wantKinds) {
				t.Errorf("temp %d, events %v; want %d, %v [%s]", got, ssEventKinds(ended.Events()), c.wantTemp, c.wantKinds, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-035 and table T row "end, replaced" - RecordTemporaryHitPointsReplaced sets FromForm to Replaced, so the end leaves the other effect's points; any other provenance and the zero value are unchanged.
// Path: per case.
func TestRecordTemporaryHitPointsReplaced(t *testing.T) {
	t.Parallel()
	s := combat.ShapeShiftSpells{}
	begun := ssBegin(t, ssAnimalShapes(t))
	replaced := s.RecordTemporaryHitPointsReplaced(begun.Transformation())
	if replaced.TemporaryHitPoints() != combat.TemporaryReplaced {
		t.Fatalf("provenance = %v, want Replaced", replaced.TemporaryHitPoints())
	}
	status := pcStatus(t, 20, 30, 9) // another effect's points took over
	ended := ssEnd(t, replaced, combat.CauseEndedByTarget, combat.TrackedHitPoints{Status: status})
	if ended.Status().HitPoints().Temporary() != 9 || slices.Contains(ssEventKinds(ended.Events()), "temporary_hit_points_ended") {
		t.Errorf("the end took the other effect's points: %v, temp %d", ssEventKinds(ended.Events()), ended.Status().HitPoints().Temporary())
	}
	if again := s.RecordTemporaryHitPointsReplaced(replaced); again != replaced {
		t.Error("Replaced changed again")
	}
	declined := ssBegin(t, ssTruePolymorph(t, combat.CreatureIntoObject)).Transformation()
	if s.RecordTemporaryHitPointsReplaced(declined) != declined {
		t.Error("NotGranted changed")
	}
	if got := s.RecordTemporaryHitPointsReplaced(combat.ActiveTransformation{}); !got.IsZero() {
		t.Error("the zero transformation changed")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-031 - a later ChangeForm gains no further Temporary Hit Points: the amount is that of the first form (table T row "form change").
// Path: unhappy (rule interaction).
func TestChangeFormGainsNoFurtherTemporaryHitPoints(t *testing.T) {
	t.Parallel()
	begun := ssBegin(t, ssShapechange(t))
	if begun.Status().HitPoints().Temporary() != 40 {
		t.Fatalf("first form temp = %d, want 40", begun.Status().HitPoints().Temporary())
	}
	second := ssForm(t, content.CreatureTypeDragon, "5", 70, content.SizeLarge)
	res, err := (combat.ShapeShiftSpells{}).ChangeForm(combat.ChangeFormCommand{Actor: gid("wizard"), State: begun.Transformation(), Form: second,
		Rating: ssLevel(17), HasSeenKind: true, Equipment: combat.ResizesToFit})
	if err != nil {
		t.Fatal(err)
	}
	changed := res.(combat.ChangeFormApplied)
	if changed.Transformation().TemporaryHitPoints() != combat.TemporaryFromForm || changed.Transformation().Form() != second {
		t.Errorf("provenance %v, form %v", changed.Transformation().TemporaryHitPoints(), changed.Transformation().Form())
	}
	for _, e := range changed.Events() {
		if e.Kind() == "temporary_hit_points_gained" {
			t.Error("ChangeForm gained Temporary Hit Points")
		}
	}
}

// Requirement: per case (doc/specs/shape-changing-spells.md Q-3, FR-032, FR-053 - a living target at 0 current Hit Points, Alive or Dying, stays shifted: True Polymorph, Shapechange and Animal Shapes continue, and Polymorph ends only with FormHitPointsDepleted, that is when it holds no Temporary Hit Points; SRD 5.2.1, "Shape-Shifting", p. 187)
// Path: per case.
func TestAssessForALivingTargetAtZeroHitPoints(t *testing.T) {
	t.Parallel()
	zero, dying := ssStableAtZero(t, 0), dyingStatus(t, 30, 1, 1)
	zeroWithTemp := ssStableAtZero(t, 5)
	cases := []struct {
		name, requirement, path string
		cmd                     combat.BeginShapeShiftCommand
		status                  combat.HitPointStatus
		wantEnds                bool
	}{
		{"True Polymorph, 0 Hit Points", "Q-3, FR-032", "unhappy (boundary)", ssTruePolymorph(t, combat.CreatureIntoCreature), zero, false},
		{"True Polymorph, Dying", "Q-3, FR-032", "unhappy (state error)", ssTruePolymorph(t, combat.CreatureIntoCreature), dying, false},
		{"Shapechange, 0 Hit Points", "Q-3, FR-032", "unhappy (boundary)", ssShapechange(t), zero, false},
		{"Shapechange, Dying", "Q-3, FR-032", "unhappy (state error)", ssShapechange(t), dying, false},
		{"Animal Shapes, 0 Hit Points", "Q-3, FR-032", "unhappy (boundary)", ssAnimalShapes(t), zero, false},
		{"Animal Shapes, Dying", "Q-3, FR-032", "unhappy (state error)", ssAnimalShapes(t), dying, false},
		{"Polymorph, 0 Hit Points and no Temporary Hit Points", "FR-032", "unhappy (boundary)", ssPolymorph(t), zero, true},
		{"Polymorph, Dying and no Temporary Hit Points", "FR-032", "unhappy (state error)", ssPolymorph(t), dying, true},
		{"Polymorph, 0 Hit Points and Temporary Hit Points left", "FR-032", "unhappy (boundary)", ssPolymorph(t), zeroWithTemp, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := ssBegin(t, c.cmd)
			res := ssAssess(t, a.Transformation(), c.status)
			ended, isEnded := res.(combat.AssessShapeShiftEnded)
			if isEnded != c.wantEnds || (isEnded && ended.Cause() != combat.CauseFormHitPointsDepleted) {
				t.Errorf("Assess = %#v, want ends %v with FormHitPointsDepleted only [%s]", res, c.wantEnds, c.requirement)
			}
		})
	}
}

// ssStableAtZero is a living, Stable creature at 0 current Hit Points holding temp Temporary Hit Points (an Alive
// creature cannot hold 0, so a living target at 0 is Dying or Stable).
func ssStableAtZero(tb testing.TB, temp int) combat.HitPointStatus {
	tb.Helper()
	return status(tb, 0, 30, temp, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities())
}
