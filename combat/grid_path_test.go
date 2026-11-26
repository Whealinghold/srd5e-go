package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func dirs(d combat.GridDirection, n int) []combat.GridDirection {
	out := make([]combat.GridDirection, n)
	for i := range out {
		out[i] = d
	}
	return out
}

// planned plans the path and fails the test unless it is planned.
func planned(tb testing.TB, board combat.GridBoard, mover combat.GridCreature, ds []combat.GridDirection, allies []combat.CombatantID) combat.GridPathPlanned {
	tb.Helper()
	got, err := gridRules.PlanPath(squareGrid, board, mover, ds, allies, combat.MovementModeWalk, combat.SpeedUseWalking, 0)
	p, ok := got.(combat.GridPathPlanned)
	if err != nil || !ok {
		tb.Fatalf("PlanPath = %#v, %v, want GridPathPlanned", got, err)
	}
	return p
}

// Requirement: per case
// Path: per case.
func TestPlanPathAgainstTheTracker(t *testing.T) {
	t.Parallel()
	const p13 = `SRD 5.2.1, "Speed", p. 13`
	mover := gridCreature(t, "a", content.SizeMedium, 0, 0)
	foe := gridCreature(t, "foe", content.SizeMedium, 1, 0)
	cases := []struct {
		name, requirement, path string
		walk                    int
		terrain                 combat.GridTerrain
		creatures               []combat.GridCreature
		directions              []combat.GridDirection
		wantSegments            int
		wantFeet                int   // the cost the tracker charges when it accepts
		refusedAt               int   // the first segment the tracker refuses, or -1
		difficult               []int // the segment indexes planned as Difficult Terrain
	}{
		{"six open steps (Speed 30)", p13, "happy", 30, combat.GridTerrain{}, nil, dirs(combat.GridDirectionEast, 6), 6, 30, -1, nil},
		{"seven open steps", "FR-029", "unhappy (boundary)", 30, combat.GridTerrain{}, nil, dirs(combat.GridDirectionEast, 7), 7, 0, 6, nil},
		{"diagonal run", p13, "happy", 30, combat.GridTerrain{}, nil, dirs(combat.GridDirectionSouthEast, 3), 3, 15, -1, nil},
		{"one difficult square", p13, "unhappy (rule interaction)", 30, combat.MustGridTerrain(squares(t, 2, 0), nil), nil, dirs(combat.GridDirectionEast, 3), 3, 20, -1, []int{1}},
		{"Speed 32 (rounds down), six steps", p13 + `; "Round Down", p. 187`, "unhappy (boundary)", 32, combat.GridTerrain{}, nil, dirs(combat.GridDirectionEast, 6), 6, 30, -1, nil},
		{"Speed 32 (rounds down), a seventh step", p13 + `; "Round Down", p. 187`, "unhappy (boundary)", 32, combat.GridTerrain{}, nil, dirs(combat.GridDirectionEast, 7), 7, 0, 6, nil},
		{"difficult squares exactly use the Speed", p13, "unhappy (boundary)", 30, combat.MustGridTerrain(squares(t, 1, 0, 2, 0, 3, 0), nil), nil, dirs(combat.GridDirectionEast, 3), 3, 30, -1, []int{0, 1, 2}},
		{"enemy in path", "FR-029", "unhappy (rule interaction)", 30, combat.GridTerrain{}, []combat.GridCreature{foe}, dirs(combat.GridDirectionEast, 2), 2, 0, 0, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			board := gridBoard(t, c.terrain, c.creatures...)
			plan := planned(t, board, mover, c.directions, nil)
			if len(plan.Segments) != c.wantSegments {
				t.Fatalf("segments = %d, want %d [%s]", len(plan.Segments), c.wantSegments, c.requirement)
			}
			for i, s := range plan.Segments {
				if s.DistanceFeet() != 5 || s.Mode() != combat.MovementModeWalk {
					t.Errorf("segment %d = %d feet mode %v, want 5 feet walking", i, s.DistanceFeet(), s.Mode())
				}
			}
			tracker := trkSolo(t, content.SizeMedium, trkSpeeds(t, c.walk))
			// The tracker owns the mover's id, so the plan is judged for the combatant named "a".
			step, err := tracker.Move(trkA, plan.Segments)
			if err != nil {
				t.Fatal(err)
			}
			switch r := step.(type) {
			case combat.TrackerStepApplied:
				if c.refusedAt != -1 || r.Events()[0].(combat.MovementSpent).TotalFeet() != c.wantFeet {
					t.Errorf("the tracker accepted the plan (%v feet), want refusal at %d / %d feet [%s]", r.Events(), c.refusedAt, c.wantFeet, c.requirement)
				}
			case combat.TrackerStepRejected:
				if c.refusedAt == -1 {
					t.Errorf("the tracker refused the plan: %#v [%s]", r.Reason, c.requirement)
				}
				if got := rejectedSegment(r.Reason); got != c.refusedAt {
					t.Errorf("refused at segment %d (%#v), want %d [%s]", got, r.Reason, c.refusedAt, c.requirement)
				}
			}
			for i, s := range plan.Segments {
				hasDifficult := slices.Contains(c.difficult, i)
				single, err := tracker.Move(trkA, []combat.MoveSegment{s})
				if err != nil {
					t.Fatal(err)
				}
				if applied, ok := single.(combat.TrackerStepApplied); ok {
					want := 5
					if hasDifficult {
						want = 10
					}
					if got := applied.Events()[0].(combat.MovementSpent).TotalFeet(); got != want {
						t.Errorf("segment %d costs %d feet, want %d (difficult %v)", i, got, want, hasDifficult)
					}
				}
			}
		})
	}
}

// rejectedSegment is the index of the segment a rejection names, or -2 for one that names none.
func rejectedSegment(r combat.TurnRejection) int {
	switch v := r.(type) {
	case combat.TurnInsufficientMovementRejection:
		return v.SegmentIndex
	case combat.TurnBlockedByCreatureRejection:
		return v.SegmentIndex
	case combat.TurnNoMovementRemainingRejection:
		return v.SegmentIndex
	case combat.TurnEndsInOccupiedSpaceRejection:
		return v.SegmentIndex
	}
	return -2
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-028 - PlanPath applies the step rules in order, the mover's footprint following each step, and returns the final footprint with one 5-foot segment per step carrying the mode, the speed use, the extra cost and the step's terrain, occupants and end.
// Path: happy.
func TestPlanPathBuildsOneSegmentPerStep(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "mover", content.SizeMedium, 0, 0)
	ally := gridCreature(t, "ally", content.SizeSmall, 2, 1)
	board := gridBoard(t, combat.MustGridTerrain(squares(t, 1, 1), nil), ally)
	got, err := gridRules.PlanPath(squareGrid, board, mover,
		[]combat.GridDirection{combat.GridDirectionSouthEast, combat.GridDirectionEast, combat.GridDirectionEast, combat.GridDirectionNorth},
		[]combat.CombatantID{"ally"}, combat.MovementModeClimb, combat.SpeedUseSpecial, 1)
	p, ok := got.(combat.GridPathPlanned)
	if err != nil || !ok {
		t.Fatalf("PlanPath = %#v, %v", got, err)
	}
	want := []combat.MoveSegmentInput{
		{DistanceFeet: 5, Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial, Terrain: combat.TerrainDifficult, ExtraCostPerFoot: 1},
		{DistanceFeet: 5, Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial, ExtraCostPerFoot: 1, End: combat.SegmentEndOccupiedByCreature,
			Occupants: []combat.Occupant{{Size: content.SizeSmall, Relation: combat.OccupantAlly, Availability: combat.ConditionActionAvailabilityAvailable}}},
		{DistanceFeet: 5, Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial, ExtraCostPerFoot: 1},
		{DistanceFeet: 5, Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial, ExtraCostPerFoot: 1},
	}
	if p.FinalFootprint != fp(t, 3, 0, 1, 1) || len(p.Segments) != len(want) {
		t.Fatalf("final footprint %v with %d segments, want (3, 0) with %d", p.FinalFootprint, len(p.Segments), len(want))
	}
	for i, in := range want {
		seg, err := combat.NewMoveSegment(in)
		if err != nil || !seg.Equal(p.Segments[i]) {
			t.Errorf("segment %d = %#v, want %#v (%v)", i, p.Segments[i], in, err)
		}
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-028, table 4.4a - at the first blocked step PlanPath returns GridPathBlocked with the step index and the reason, and no segments, whatever the later steps.
// Path: unhappy (rule interaction).
func TestPlanPathStopsAtTheFirstBlockedStep(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "mover", content.SizeMedium, 0, 0)
	wall := gridBoard(t, combat.MustGridTerrain(nil, squares(t, 2, 0)))
	cases := []struct {
		name string
		ds   []combat.GridDirection
		want combat.GridPathBlocked
	}{
		{"blocked second step (index 1)", dirs(combat.GridDirectionEast, 4), combat.GridPathBlocked{StepIndex: 1, Reason: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"blocked first step", []combat.GridDirection{combat.GridDirectionEast, combat.GridDirectionEast}, combat.GridPathBlocked{StepIndex: 1, Reason: combat.GridStepBlockReasonEnteringFilledSquare}},
		{"cutting the corner of the wall", []combat.GridDirection{combat.GridDirectionEast, combat.GridDirectionNorthEast}, combat.GridPathBlocked{StepIndex: 1, Reason: combat.GridStepBlockReasonCuttingCorner}},
	}
	for _, c := range cases {
		got, err := gridRules.PlanPath(squareGrid, wall, mover, c.ds, nil, combat.MovementModeWalk, combat.SpeedUseWalking, 0)
		if err != nil || got != combat.GridPathPlan(c.want) {
			t.Errorf("%s: PlanPath = %#v, %v, want %#v", c.name, got, err, c.want)
		}
	}
	first := gridBoard(t, combat.MustGridTerrain(nil, squares(t, 1, 0)))
	if got, _ := gridRules.PlanPath(squareGrid, first, mover, dirs(combat.GridDirectionEast, 3), nil, combat.MovementModeWalk, combat.SpeedUseWalking, 0); got != combat.GridPathPlan(combat.GridPathBlocked{StepIndex: 0, Reason: combat.GridStepBlockReasonEnteringFilledSquare}) {
		t.Errorf("a blocked first step = %#v", got)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-028, FR-030, FR-031, FR-040 - the arguments a plan cannot be made from are ErrInvalidInput errors even when the first step would be blocked: no directions, an invalid direction, a negative extra cost, an invalid mode or speed use, the zero mover, and a step that leaves the grid range.
// Path: unhappy (invalid input).
func TestPlanPathRejects(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "mover", content.SizeMedium, 0, 0)
	walled := gridBoard(t, combat.MustGridTerrain(nil, squares(t, 1, 0)))
	east := dirs(combat.GridDirectionEast, 1)
	edge := gridCreature(t, "edge", content.SizeMedium, 1_000_000, 0)
	cases := []struct {
		name     string
		board    combat.GridBoard
		mover    combat.GridCreature
		ds       []combat.GridDirection
		mode     combat.MovementMode
		use      combat.SpeedUse
		extra    int
		fr       string
		wantBase bool
	}{
		{"no directions", walled, mover, nil, combat.MovementModeWalk, combat.SpeedUseWalking, 0, "FR-028", false},
		{"an empty list", walled, mover, []combat.GridDirection{}, combat.MovementModeWalk, combat.SpeedUseWalking, 0, "FR-028", false},
		{"an invalid direction", walled, mover, []combat.GridDirection{0}, combat.MovementModeWalk, combat.SpeedUseWalking, 0, "FR-028", false},
		{"a negative extra cost on a blocked path", walled, mover, east, combat.MovementModeWalk, combat.SpeedUseWalking, -1, "FR-030", false},
		{"the smallest extra cost", walled, mover, east, combat.MovementModeWalk, combat.SpeedUseWalking, math.MinInt, "FR-030", false},
		{"the zero mode", walled, mover, east, 0, combat.SpeedUseWalking, 0, "FR-028", false},
		{"an invalid speed use", walled, mover, east, combat.MovementModeWalk, 9, 0, "FR-028", false},
		{"the zero mover", walled, combat.GridCreature{}, east, combat.MovementModeWalk, combat.SpeedUseWalking, 0, "FR-028", false},
		{"leaving the grid range", combat.GridBoard{}, edge, east, combat.MovementModeWalk, combat.SpeedUseWalking, 0, "FR-031", false},
		{"leaving the grid range on the second step", combat.GridBoard{}, gridCreature(t, "edge", content.SizeMedium, 999_999, 0), dirs(combat.GridDirectionEast, 2), combat.MovementModeWalk, combat.SpeedUseWalking, 0, "FR-031", false},
	}
	for _, c := range cases {
		if got, err := gridRules.PlanPath(squareGrid, c.board, c.mover, c.ds, nil, c.mode, c.use, c.extra); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
			t.Errorf("%s: PlanPath = %#v, %v, want ErrInvalidInput [%s]", c.name, got, err, c.fr)
		}
	}
	if p := planned(t, combat.GridBoard{}, gridCreature(t, "edge", content.SizeMedium, 999_999, 0), east, nil); p.FinalFootprint != fp(t, 1_000_000, 0, 1, 1) {
		t.Errorf("a path ending exactly on the limit = %v", p.FinalFootprint)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-030 (SRD 5.2.1, "Entering a Square", p. 13) - a positive extra cost makes each square cost that many extra squares: the tracker charges 20 feet for two squares at 1 extra, and refuses what no longer fits.
// Path: unhappy (rule interaction).
func TestPlanPathExtraCostIsChargedByTheTracker(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "a", content.SizeMedium, 0, 0)
	got, err := gridRules.PlanPath(squareGrid, combat.GridBoard{}, mover, dirs(combat.GridDirectionEast, 2), nil, combat.MovementModeWalk, combat.SpeedUseWalking, 1)
	p, ok := got.(combat.GridPathPlanned)
	if err != nil || !ok {
		t.Fatalf("PlanPath = %#v, %v", got, err)
	}
	applied, _ := trkApplied(t)(trkSolo(t, content.SizeMedium, trkSpeeds(t, 30)).Move(trkA, p.Segments))
	if applied.Resources().MovedFeet() != 20 {
		t.Errorf("moved %d feet, want 20 (4 squares for 2 steps at 1 extra)", applied.Resources().MovedFeet())
	}
	if r := trkReason(t)(trkSolo(t, content.SizeMedium, trkSpeeds(t, 15)).Move(trkA, p.Segments)); r != (combat.TurnInsufficientMovementRejection{SegmentIndex: 1, ShortfallFeet: 5}) {
		t.Errorf("a Speed of 15 = %#v, want insufficient movement at the second segment", r)
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-029 - a plan is not judged for creatures that cannot be passed through nor for ending in an occupied space: it is returned as planned, and the tracker refuses blockedByCreature and endsInOccupiedSpace.
// Path: unhappy (rule interaction).
func TestPlanPathLeavesPassageToTheTracker(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "a", content.SizeMedium, 0, 0)
	foe := gridCreature(t, "foe", content.SizeMedium, 1, 0)
	tracker := trkSolo(t, content.SizeMedium, trkSpeeds(t, 30))
	through := planned(t, gridBoard(t, combat.GridTerrain{}, foe), mover, dirs(combat.GridDirectionEast, 2), nil)
	if r := trkReason(t)(tracker.Move(trkA, through.Segments)); r != (combat.TurnBlockedByCreatureRejection{}) {
		t.Errorf("an enemy in the path = %#v, want blockedByCreature", r)
	}
	ending := planned(t, gridBoard(t, combat.GridTerrain{}, foe), mover, dirs(combat.GridDirectionEast, 1), nil)
	if r := trkReason(t)(tracker.Move(trkA, ending.Segments)); r != (combat.TurnBlockedByCreatureRejection{}) {
		t.Errorf("ending in an enemy square = %#v", r)
	}
	ally := planned(t, gridBoard(t, combat.GridTerrain{}, foe), mover, dirs(combat.GridDirectionEast, 1), []combat.CombatantID{"foe"})
	if r := trkReason(t)(tracker.Move(trkA, ally.Segments)); r != (combat.TurnEndsInOccupiedSpaceRejection{}) {
		t.Errorf("ending in an ally's space = %#v, want endsInOccupiedSpace", r)
	}
	pastAlly := planned(t, gridBoard(t, combat.GridTerrain{}, foe), mover, dirs(combat.GridDirectionEast, 3), []combat.CombatantID{"foe"})
	if applied, _ := trkApplied(t)(tracker.Move(trkA, pastAlly.Segments)); applied.Resources().MovedFeet() != 15 {
		t.Errorf("passing through an ally moved %d feet, want 15", applied.Resources().MovedFeet())
	}
}

// Requirement: doc/specs/grid-positions-and-movement.md FR-032, FR-020 - PlanPath changes neither the board, the terrain, the mover nor the directions, gives equal outputs for equal inputs, and the host applies a plan with the mover at its final footprint; a diagonal step costs the same as an orthogonal one.
// Path: unhappy (immutability).
func TestPlanPathChangesNothingAndIsApplied(t *testing.T) {
	t.Parallel()
	mover := gridCreature(t, "a", content.SizeLarge, 0, 0)
	foe := gridCreature(t, "foe", content.SizeMedium, 8, 8)
	board := gridBoard(t, combat.MustGridTerrain(squares(t, 2, 1), nil), foe)
	ds := []combat.GridDirection{combat.GridDirectionSouthEast, combat.GridDirectionEast, combat.GridDirectionNorthEast}
	first := planned(t, board, mover, ds, nil)
	second := planned(t, board, mover, ds, nil)
	if first.FinalFootprint != second.FinalFootprint || !slices.EqualFunc(first.Segments, second.Segments, combat.MoveSegment.Equal) {
		t.Error("equal inputs gave different plans")
	}
	if first.FinalFootprint != fp(t, 3, 0, 2, 2) || mover.Footprint() != fp(t, 0, 0, 2, 2) || !slices.Equal(board.Creatures(), []combat.GridCreature{foe}) ||
		!slices.Equal(ds, []combat.GridDirection{combat.GridDirectionSouthEast, combat.GridDirectionEast, combat.GridDirectionNorthEast}) {
		t.Errorf("final %v, mover %v: the inputs changed or the path is wrong", first.FinalFootprint, mover.Footprint())
	}
	first.Segments[0] = first.Segments[1]
	if third := planned(t, board, mover, ds, nil); slices.EqualFunc(first.Segments, third.Segments, combat.MoveSegment.Equal) {
		t.Error("the plan shares its segment slice")
	}
	moved, err := mover.WithFootprint(second.FinalFootprint)
	if err != nil {
		t.Fatal(err)
	}
	after, err := board.Place(moved)
	if err != nil || after.Creatures()[0] != foe && len(after.Creatures()) != 2 {
		t.Errorf("Place of the planned mover = %v, %v", after.Creatures(), err)
	}
}
