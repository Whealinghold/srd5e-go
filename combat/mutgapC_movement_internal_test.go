package combat

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

func mutgapCSegment(t *testing.T, in MoveSegmentInput) MoveSegment {
	t.Helper()
	if in.DistanceFeet == 0 {
		in.DistanceFeet = 10
	}
	s, err := NewMoveSegment(in)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/initiative-and-actions.md FR-077 - Equal tells two segments apart when any one of distance, extra cost, mode, speed use, terrain or end differs, whichever is larger.
// Path: unhappy (boundary).
func TestMutgapCSegmentEqualTellsEveryFieldApart(t *testing.T) {
	t.Parallel()
	big := mutgapCSegment(t, MoveSegmentInput{DistanceFeet: 10, ExtraCostPerFoot: 2, Mode: MovementModeSwim, SpeedUse: SpeedUseSpecial, Terrain: TerrainDifficult, End: SegmentEndOccupiedByCreature})
	for name, small := range map[string]MoveSegmentInput{
		"distance": {DistanceFeet: 5, ExtraCostPerFoot: 2, Mode: MovementModeSwim, SpeedUse: SpeedUseSpecial, Terrain: TerrainDifficult, End: SegmentEndOccupiedByCreature},
		"extra":    {DistanceFeet: 10, ExtraCostPerFoot: 1, Mode: MovementModeSwim, SpeedUse: SpeedUseSpecial, Terrain: TerrainDifficult, End: SegmentEndOccupiedByCreature},
		"mode":     {DistanceFeet: 10, ExtraCostPerFoot: 2, Mode: MovementModeWalk, SpeedUse: SpeedUseSpecial, Terrain: TerrainDifficult, End: SegmentEndOccupiedByCreature},
		"use":      {DistanceFeet: 10, ExtraCostPerFoot: 2, Mode: MovementModeSwim, SpeedUse: SpeedUseWalking, Terrain: TerrainDifficult, End: SegmentEndOccupiedByCreature},
		"terrain":  {DistanceFeet: 10, ExtraCostPerFoot: 2, Mode: MovementModeSwim, SpeedUse: SpeedUseSpecial, Terrain: TerrainOpen, End: SegmentEndOccupiedByCreature},
		"end":      {DistanceFeet: 10, ExtraCostPerFoot: 2, Mode: MovementModeSwim, SpeedUse: SpeedUseSpecial, Terrain: TerrainDifficult, End: SegmentEndFree},
	} {
		other := mutgapCSegment(t, small)
		if big.Equal(other) || other.Equal(big) {
			t.Errorf("segments differing in %s must not be equal", name)
		}
	}
	if !big.Equal(mutgapCSegment(t, MoveSegmentInput{DistanceFeet: 10, ExtraCostPerFoot: 2, Mode: MovementModeSwim, SpeedUse: SpeedUseSpecial, Terrain: TerrainDifficult, End: SegmentEndOccupiedByCreature})) {
		t.Error("equal inputs must give equal segments")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-077 - a foot costs 1, plus 1 for climbing, swimming or crawling and 1 for Difficult Terrain, but a Climb or Swim speed removes both; a walk, burrow or fly foot has no mode surcharge.
// Path: happy; unhappy (boundary).
func TestMutgapCCostPerFootForEveryModeAndSpeedUse(t *testing.T) {
	t.Parallel()
	type key struct {
		mode    MovementMode
		use     SpeedUse
		terrain Terrain
	}
	want := map[key]int{}
	for _, m := range []MovementMode{MovementModeWalk, MovementModeBurrow, MovementModeClimb, MovementModeCrawl, MovementModeFly, MovementModeSwim} {
		for _, u := range []SpeedUse{SpeedUseWalking, SpeedUseSpecial} {
			for _, g := range []Terrain{TerrainOpen, TerrainDifficult} {
				c := 1
				if u != SpeedUseSpecial || (m != MovementModeClimb && m != MovementModeSwim) {
					if m == MovementModeClimb || m == MovementModeSwim || m == MovementModeCrawl {
						c++
					}
					if g == TerrainDifficult {
						c++
					}
				}
				want[key{m, u, g}] = c
			}
		}
	}
	for k, c := range want {
		s := mutgapCSegment(t, MoveSegmentInput{DistanceFeet: 5, Mode: k.mode, SpeedUse: k.use, Terrain: k.terrain})
		if got := s.costPerFoot(); got != c {
			t.Errorf("%v use %d terrain %d: costPerFoot = %d, want %d", k.mode, k.use, k.terrain, got, c)
		}
		if got := s.cost(); got != 5*c {
			t.Errorf("%v use %d terrain %d: cost = %d, want %d", k.mode, k.use, k.terrain, got, 5*c)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-077, SRD 5.2.1, "Difficult Terrain", p. 181 - a space held by a creature that is neither Tiny nor an ally is Difficult Terrain; a Tiny creature or an ally is not.
// Path: unhappy (boundary).
func TestMutgapCOccupantsMakeDifficultTerrain(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		o    Occupant
		want int
	}{
		{"medium other", Occupant{Size: content.SizeMedium, Relation: OccupantOther, Availability: ConditionActionAvailabilityAvailable}, 2},
		{"small other", Occupant{Size: content.SizeSmall, Relation: OccupantOther, Availability: ConditionActionAvailabilityAvailable}, 2},
		{"gargantuan other", Occupant{Size: content.SizeGargantuan, Relation: OccupantOther, Availability: ConditionActionAvailabilityAvailable}, 2},
		{"tiny other", Occupant{Size: content.SizeTiny, Relation: OccupantOther, Availability: ConditionActionAvailabilityAvailable}, 1},
		{"medium ally", Occupant{Size: content.SizeMedium, Relation: OccupantAlly, Availability: ConditionActionAvailabilityAvailable}, 1},
	} {
		s := mutgapCSegment(t, MoveSegmentInput{Mode: MovementModeWalk, Occupants: []Occupant{c.o}})
		if got := s.costPerFoot(); got != c.want {
			t.Errorf("%s: costPerFoot = %d, want %d", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-077 - the cost of a long segment saturates at the largest int and is exact up to it.
// Path: unhappy (boundary).
func TestMutgapCSatMulPositiveAtTheOverflowEdge(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ a, b, want int }{
		{math.MaxInt / 2, 2, math.MaxInt - 1},
		{math.MaxInt/2 + 1, 2, math.MaxInt},
		{math.MaxInt / 3, 3, math.MaxInt - 1},
		{math.MaxInt/3 + 1, 3, math.MaxInt},
		{math.MaxInt, 1, math.MaxInt},
	} {
		if got := satMulPositive(c.a, c.b); got != c.want {
			t.Errorf("satMulPositive(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-092, SRD 5.2.1, "Moving around Other Creatures", p. 14 - a mover passes an ally, an Incapacitated creature, a Tiny creature or one two or more sizes away in either direction, and no other.
// Path: unhappy (boundary).
func TestMutgapCCanPassThrough(t *testing.T) {
	t.Parallel()
	avail, down := ConditionActionAvailabilityAvailable, ConditionActionAvailabilityNone
	for _, c := range []struct {
		name  string
		mover content.CreatureSize
		o     Occupant
		want  bool
	}{
		{"one size larger", content.SizeMedium, Occupant{content.SizeLarge, OccupantOther, avail}, false},
		{"two sizes larger", content.SizeMedium, Occupant{content.SizeHuge, OccupantOther, avail}, true},
		{"one size smaller", content.SizeLarge, Occupant{content.SizeMedium, OccupantOther, avail}, false},
		{"two sizes smaller", content.SizeHuge, Occupant{content.SizeSmall, OccupantOther, avail}, true},
		{"same size", content.SizeMedium, Occupant{content.SizeMedium, OccupantOther, avail}, false},
		{"tiny beside a small mover", content.SizeSmall, Occupant{content.SizeTiny, OccupantOther, avail}, true},
		{"tiny beside a tiny mover", content.SizeTiny, Occupant{content.SizeTiny, OccupantOther, avail}, true},
		{"ally of the same size", content.SizeMedium, Occupant{content.SizeMedium, OccupantAlly, avail}, true},
		{"incapacitated of the same size", content.SizeMedium, Occupant{content.SizeMedium, OccupantOther, down}, true},
	} {
		if got := canPassThrough(c.mover, c.o); got != c.want {
			t.Errorf("%s: canPassThrough = %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-078 - the speed that governs a segment is Fly for fly, Burrow for burrow, Climb or Swim only for climb or swim on the special speed, and Walk for everything else.
// Path: happy; unhappy (boundary).
func TestMutgapCGoverningSpeedForEveryModeAndSpeedUse(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		mode    MovementMode
		walking SpeedName
		special SpeedName
	}{
		{MovementModeWalk, SpeedNameWalk, SpeedNameWalk},
		{MovementModeBurrow, SpeedNameBurrow, SpeedNameBurrow},
		{MovementModeClimb, SpeedNameWalk, SpeedNameClimb},
		{MovementModeCrawl, SpeedNameWalk, SpeedNameWalk},
		{MovementModeFly, SpeedNameFly, SpeedNameFly},
		{MovementModeSwim, SpeedNameWalk, SpeedNameSwim},
	} {
		for use, want := range map[SpeedUse]SpeedName{SpeedUseWalking: c.walking, SpeedUseSpecial: c.special} {
			if got := mutgapCSegment(t, MoveSegmentInput{Mode: c.mode, SpeedUse: use}).governing(); got != want {
				t.Errorf("%v use %d: governing = %v, want %v", c.mode, use, got, want)
			}
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-092 - a move that ends in an occupied space is refused at its last segment, however many segments precede it; an occupied end that is not last does not refuse it.
// Path: unhappy (state error).
func TestMutgapCJudgeMoveRefusesAnOccupiedEndAtTheLastSegment(t *testing.T) {
	t.Parallel()
	speeds, err := NewMovementSpeeds(60)
	if err != nil {
		t.Fatal(err)
	}
	free := mutgapCSegment(t, MoveSegmentInput{Mode: MovementModeWalk})
	taken := mutgapCSegment(t, MoveSegmentInput{Mode: MovementModeWalk, End: SegmentEndOccupiedByCreature})
	for _, c := range []struct {
		name string
		segs []MoveSegment
		want int
	}{
		{"one segment", []MoveSegment{taken}, 0},
		{"two segments", []MoveSegment{free, taken}, 1},
		{"three segments", []MoveSegment{free, free, taken}, 2},
	} {
		j := judgeMove(speeds, content.SizeMedium, 0, 0, c.segs)
		if r, ok := j.rejection.(TurnEndsInOccupiedSpaceRejection); !ok || r.SegmentIndex != c.want {
			t.Errorf("%s: rejection = %#v, want an occupied end at segment %d", c.name, j.rejection, c.want)
		}
	}
	if j := judgeMove(speeds, content.SizeMedium, 0, 0, []MoveSegment{taken, free}); j.rejection != nil {
		t.Errorf("an occupied end that is not last gave %#v, want none", j.rejection)
	}
}
