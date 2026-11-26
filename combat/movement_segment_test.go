package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// trkBig is a tracker whose mover has 1000 feet of every speed, so only the cost of a segment matters.
func trkBig(tb testing.TB, size content.CreatureSize) combat.TurnTracker {
	tb.Helper()
	all := trkSpeeds(tb, 1000, fly(1000), climb(1000), combat.MovementSpeedEntry{Name: combat.SpeedNameBurrow, Feet: 1000},
		combat.MovementSpeedEntry{Name: combat.SpeedNameSwim, Feet: 1000})
	return trkTracker(tb, trkProfile(tb, trkA, size, all), trkProfile(tb, trkB, content.SizeMedium, all))
}

// trkCost is the movement one move of seg costs, read from the MovementSpent event.
func trkCost(tb testing.TB, seg combat.MoveSegment) int {
	tb.Helper()
	_, ev := trkApplied(tb)(trkBig(tb, content.SizeMedium).Move(trkA, []combat.MoveSegment{seg}))
	return ev[0].(combat.MovementSpent).TotalFeet()
}

// Requirement: doc/specs/initiative-and-actions.md FR-087 - a segment of distance 0 or less, a negative extra cost or an undefined mode, use, terrain, end or occupant is an invalid-input error.
// Path: unhappy (invalid input).
func TestNewMoveSegmentRejectsInvalid(t *testing.T) {
	t.Parallel()
	good := combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk}
	occ := func(o combat.Occupant) func(*combat.MoveSegmentInput) {
		return func(in *combat.MoveSegmentInput) { in.Occupants = []combat.Occupant{o} }
	}
	medium := combat.Occupant{Size: content.SizeMedium, Relation: combat.OccupantOther, Availability: trkAvail}
	cases := []struct {
		name   string
		change func(*combat.MoveSegmentInput)
	}{
		{"zero distance", func(in *combat.MoveSegmentInput) { in.DistanceFeet = 0 }},
		{"negative distance", func(in *combat.MoveSegmentInput) { in.DistanceFeet = -5 }},
		{"negative extra", func(in *combat.MoveSegmentInput) { in.ExtraCostPerFoot = -1 }},
		{"no mode", func(in *combat.MoveSegmentInput) { in.Mode = 0 }},
		{"mode past the last", func(in *combat.MoveSegmentInput) { in.Mode = combat.MovementModeSwim + 1 }},
		{"speed use", func(in *combat.MoveSegmentInput) { in.SpeedUse = 2 }},
		{"terrain", func(in *combat.MoveSegmentInput) { in.Terrain = 2 }},
		{"end", func(in *combat.MoveSegmentInput) { in.End = 2 }},
		{"occupant size", occ(combat.Occupant{Relation: combat.OccupantOther, Availability: trkAvail})},
		{"occupant relation", occ(combat.Occupant{Size: content.SizeMedium, Relation: 2, Availability: trkAvail})},
		{"occupant availability", occ(combat.Occupant{Size: content.SizeMedium, Relation: combat.OccupantOther})},
		{"negative speed use", func(in *combat.MoveSegmentInput) { in.SpeedUse = -1 }},
		{"negative terrain", func(in *combat.MoveSegmentInput) { in.Terrain = -1 }},
		{"negative end", func(in *combat.MoveSegmentInput) { in.End = -1 }},
		{"negative relation", occ(combat.Occupant{Size: content.SizeMedium, Relation: -1, Availability: trkAvail})},
		{"availability past the last", occ(combat.Occupant{Size: content.SizeMedium, Availability: 3})},
		{"negative availability", occ(combat.Occupant{Size: content.SizeMedium, Availability: -1})},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := good
			c.change(&in)
			if s, err := combat.NewMoveSegment(in); !errors.Is(err, srderr.ErrInvalidInput) || !s.IsZero() {
				t.Errorf("NewMoveSegment = %+v, %v; want a zero segment and invalid input [FR-087]", s, err)
			}
		})
	}
	if s, err := combat.NewMoveSegment(good); err != nil || s.IsZero() || s.DistanceFeet() != 5 || s.Mode() != combat.MovementModeWalk {
		t.Errorf("NewMoveSegment(good) = %+v, %v", s, err)
	}
	if _ = medium; !(combat.MoveSegment{}).IsZero() {
		t.Error("the zero segment is valid")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-096 - the occupants of a segment are copied on construction and Equal compares every field.
// Path: unhappy (immutability).
func TestMoveSegmentCopiesAndCompares(t *testing.T) {
	t.Parallel()
	occ := []combat.Occupant{{Size: content.SizeSmall, Relation: combat.OccupantAlly, Availability: trkAvail}}
	in := combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: occ}
	s := trkSegment(t, in)
	occ[0].Size = content.SizeHuge
	if !s.Equal(trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeSmall, Relation: combat.OccupantAlly, Availability: trkAvail}}})) {
		t.Error("changing the caller's occupants changed the segment [FR-096]")
	}
	for name, change := range map[string]func(*combat.MoveSegmentInput){
		"distance": func(i *combat.MoveSegmentInput) { i.DistanceFeet = 6 },
		"extra":    func(i *combat.MoveSegmentInput) { i.ExtraCostPerFoot = 1 },
		"mode":     func(i *combat.MoveSegmentInput) { i.Mode = combat.MovementModeFly },
		"use":      func(i *combat.MoveSegmentInput) { i.SpeedUse = combat.SpeedUseSpecial },
		"terrain":  func(i *combat.MoveSegmentInput) { i.Terrain = combat.TerrainDifficult },
		"end":      func(i *combat.MoveSegmentInput) { i.End = combat.SegmentEndOccupiedByCreature },
		"occupants": func(i *combat.MoveSegmentInput) {
			i.Occupants = []combat.Occupant{{Size: content.SizeSmall, Relation: combat.OccupantOther, Availability: trkAvail}}
		},
	} {
		other := combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeSmall, Relation: combat.OccupantAlly, Availability: trkAvail}}}
		change(&other)
		if s.Equal(trkSegment(t, other)) {
			t.Errorf("segments differing in %s are equal", name)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestMoveSegmentCostPerFoot(t *testing.T) {
	t.Parallel()
	const open, difficult = combat.TerrainOpen, combat.TerrainDifficult
	cases := []struct {
		name, requirement, path string
		in                      combat.MoveSegmentInput
		perFoot                 int
	}{
		{"walk, open", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeWalk}, 1},
		{"walk, difficult", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Terrain: difficult}, 2},
		{"climb by walking Speed, open", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeClimb}, 2},
		{"swim by walking Speed, difficult", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeSwim, Terrain: difficult}, 3},
		{"climb by the Climb Speed", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial}, 1},
		{"swim by the Swim Speed", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeSwim, SpeedUse: combat.SpeedUseSpecial}, 1},
		{"climb by the Climb Speed, difficult (interpretation)", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial, Terrain: difficult}, 1},
		{"crawl, open", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeCrawl}, 2},
		{"crawl, difficult", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeCrawl, Terrain: difficult}, 3},
		{"special use does not help a crawl", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeCrawl, SpeedUse: combat.SpeedUseSpecial}, 2},
		{"fly, open", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeFly}, 1},
		{"fly, difficult", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeFly, Terrain: difficult}, 2},
		{"burrow, open", "FR-077", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeBurrow}, 1},
		{"walk dragging a Grappled creature", "FR-077", "unhappy (boundary)", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, ExtraCostPerFoot: 1}, 2},
		{"walk with an extra cost of 0", "FR-077", "unhappy (boundary)", combat.MoveSegmentInput{Mode: combat.MovementModeWalk}, 1},
		{"walk dragging, difficult", "FR-077", "unhappy (boundary)", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, ExtraCostPerFoot: 1, Terrain: difficult}, 3},
		{"climb dragging without the Climb Speed", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeClimb, ExtraCostPerFoot: 1}, 3},
		{"fly dragging", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeFly, ExtraCostPerFoot: 1}, 2},
		{"climb by the Climb Speed dragging", "FR-077", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeClimb, SpeedUse: combat.SpeedUseSpecial, ExtraCostPerFoot: 1}, 2},
		{"an incapacitated Medium enemy makes it difficult", "FR-079", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeMedium, Relation: combat.OccupantOther, Availability: trkNone}, {Size: content.SizeTiny, Relation: combat.OccupantOther, Availability: trkAvail}}}, 2},
		{"declared and occupant, not cumulative", "FR-078", "unhappy (rule interaction)", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Terrain: difficult, Occupants: []combat.Occupant{{Size: content.SizeMedium, Relation: combat.OccupantOther, Availability: trkNone}, {Size: content.SizeSmall, Relation: combat.OccupantOther, Availability: trkNone}}}, 2},
		{"a Tiny enemy is open ground", "FR-079", "unhappy (boundary)", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeTiny, Relation: combat.OccupantOther, Availability: trkAvail}}}, 1},
		{"a Medium ally is open ground", "FR-079", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeMedium, Relation: combat.OccupantAlly, Availability: trkAvail}}}, 1},
		{"an Incapacitated Small enemy still makes it difficult", "FR-079", "happy", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeSmall, Relation: combat.OccupantOther, Availability: trkNone}}}, 2},
		{"no occupants, open", "FR-079", "unhappy (empty case)", combat.MoveSegmentInput{Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{}}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := c.in
			in.DistanceFeet = 5
			if got := trkCost(t, trkSegment(t, in)); got != 5*c.perFoot {
				t.Errorf("5 ft cost %d, want %d [doc/specs/initiative-and-actions.md %s]", got, 5*c.perFoot, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestPassingThroughACreature(t *testing.T) {
	t.Parallel()
	enemy := func(size content.CreatureSize) combat.Occupant {
		return combat.Occupant{Size: size, Relation: combat.OccupantOther, Availability: trkAvail}
	}
	cases := []struct {
		name, path string
		mover      content.CreatureSize
		occupant   combat.Occupant
		passes     bool
	}{
		{"Medium ally, Medium", "happy", content.SizeMedium, combat.Occupant{Size: content.SizeMedium, Relation: combat.OccupantAlly, Availability: trkAvail}, true},
		{"Gargantuan ally", "happy", content.SizeMedium, combat.Occupant{Size: content.SizeGargantuan, Relation: combat.OccupantAlly, Availability: trkAvail}, true},
		{"incapacitated Medium", "happy", content.SizeMedium, combat.Occupant{Size: content.SizeMedium, Relation: combat.OccupantOther, Availability: trkNone}, true},
		{"Tiny enemy", "happy", content.SizeMedium, enemy(content.SizeTiny), true},
		{"same size enemy", "unhappy (rule interaction)", content.SizeMedium, enemy(content.SizeMedium), false},
		{"one size larger", "unhappy (boundary)", content.SizeMedium, enemy(content.SizeLarge), false},
		{"one size smaller", "unhappy (boundary)", content.SizeLarge, enemy(content.SizeMedium), false},
		{"two sizes larger", "unhappy (boundary)", content.SizeMedium, enemy(content.SizeHuge), true},
		{"two sizes smaller", "unhappy (boundary)", content.SizeLarge, enemy(content.SizeSmall), true},
		{"three sizes apart", "unhappy (boundary)", content.SizeMedium, enemy(content.SizeGargantuan), true},
		{"four sizes apart", "unhappy (boundary)", content.SizeGargantuan, enemy(content.SizeSmall), true},
		{"Gargantuan against Medium", "unhappy (boundary)", content.SizeGargantuan, enemy(content.SizeMedium), true},
		{"Huge against Large", "unhappy (boundary)", content.SizeHuge, enemy(content.SizeLarge), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			seg := trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{c.occupant}})
			step, err := trkBig(t, c.mover).Move(trkA, []combat.MoveSegment{seg})
			if c.passes {
				trkApplied(t)(step, err)
				return
			}
			if r := trkReason(t)(step, err); r != (combat.TurnBlockedByCreatureRejection{SegmentIndex: 0}) {
				t.Errorf("reason = %#v, want BlockedByCreature [FR-080]", r)
			}
		})
	}
	// Requirement: doc/specs/initiative-and-actions.md FR-080 - one blocking occupant among several blocks the segment.
	seg := trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{enemy(content.SizeTiny), enemy(content.SizeLarge)}})
	if r := trkReason(t)(trkBig(t, content.SizeMedium).Move(trkA, []combat.MoveSegment{seg})); r != (combat.TurnBlockedByCreatureRejection{}) {
		t.Errorf("reason = %#v, want BlockedByCreature for one blocker among several", r)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-077 - the cost saturates instead of overflowing for a host-supplied distance and extra cost at the limits of int.
// Path: unhappy (boundary).
func TestMoveSegmentCostSaturates(t *testing.T) {
	t.Parallel()
	seg := trkSegment(t, combat.MoveSegmentInput{DistanceFeet: math.MaxInt, Mode: combat.MovementModeWalk, ExtraCostPerFoot: math.MaxInt})
	step, err := trkBig(t, content.SizeMedium).Move(trkA, []combat.MoveSegment{seg})
	if r, ok := step.(combat.TrackerStepRejected); !ok || r.Reason != (combat.TurnInsufficientMovementRejection{SegmentIndex: 0, ShortfallFeet: math.MaxInt - 1000}) {
		t.Errorf("step = %#v, %v; want InsufficientMovement with a saturated cost", step, err)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-087 - NewMoveSegment never panics for any numbers and a valid segment has the distance given.
// Path: unhappy (invalid input).
func FuzzNewMoveSegment(f *testing.F) {
	f.Add(5, 1, 0, 0, 0, 0, 0, 3)
	f.Add(math.MaxInt, math.MaxInt, 9, 9, 9, 9, 9, 99)
	f.Fuzz(func(t *testing.T, dist, extra, mode, use, terrain, end, relation, size int) {
		in := combat.MoveSegmentInput{DistanceFeet: dist, Mode: combat.MovementMode(mode), SpeedUse: combat.SpeedUse(use), Terrain: combat.Terrain(terrain),
			End: combat.SegmentEnd(end), ExtraCostPerFoot: extra,
			Occupants: []combat.Occupant{{Size: content.CreatureSize(size), Relation: combat.OccupantRelation(relation), Availability: combat.ConditionActionAvailability(use)}}}
		s, err := combat.NewMoveSegment(in)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if s.DistanceFeet() != dist || s.IsZero() {
			t.Fatalf("segment %+v lost its distance", s)
		}
	})
}
