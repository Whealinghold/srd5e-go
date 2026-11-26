package combat

import (
	"math"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// MovementMode is how a creature moves along one segment (SRD 5.2.1, "Speed",
// p. 188). The zero MovementMode is invalid.
type MovementMode int

// The movement modes.
const (
	MovementModeWalk MovementMode = iota + 1
	MovementModeBurrow
	MovementModeClimb
	MovementModeCrawl
	MovementModeFly
	MovementModeSwim
)

func movementModeNames() []string {
	return []string{"walk", "burrow", "climb", "crawl", "fly", "swim"}
}

// Valid reports whether m is one of the six modes.
func (m MovementMode) Valid() bool { return m >= MovementModeWalk && m <= MovementModeSwim }

// String returns "walk", "burrow", "climb", "crawl", "fly", "swim" or "unknown".
func (m MovementMode) String() string { return enumName(movementModeNames(), int(m)) }

// SpeedUse says whether a climb or swim segment uses the Climb or Swim speed.
// The zero value is SpeedUseWalking.
type SpeedUse int

// The two uses.
const (
	// SpeedUseWalking: the walking Speed pays the extra cost.
	SpeedUseWalking SpeedUse = iota
	// SpeedUseSpecial: the Climb or Swim speed, which removes the extra cost.
	SpeedUseSpecial
)

// Terrain is the ground of a segment. The zero value is TerrainOpen.
type Terrain int

// The two kinds of ground.
const (
	TerrainOpen Terrain = iota
	TerrainDifficult
)

// SegmentEnd says whether a segment ends in a space another creature occupies.
// The zero value is SegmentEndFree.
type SegmentEnd int

// The two ends.
const (
	SegmentEndFree SegmentEnd = iota
	SegmentEndOccupiedByCreature
)

// OccupantRelation says whether an occupant is the mover's ally. The zero value
// is OccupantOther.
type OccupantRelation int

// The two relations.
const (
	OccupantOther OccupantRelation = iota
	OccupantAlly
)

// Occupant is a creature in a space a segment passes through (SRD 5.2.1, "Moving
// around Other Creatures", p. 14). Build it with a keyed literal; NewMoveSegment
// validates it. Comparable with ==.
type Occupant struct {
	Size         content.CreatureSize
	Relation     OccupantRelation
	Availability ConditionActionAvailability
}

// MovementCharge is the movement a move spends in one mode. Comparable with ==.
type MovementCharge struct {
	Mode MovementMode
	Feet int
}

// MoveSegmentInput holds the values NewMoveSegment validates. Build it with a
// keyed literal. ExtraCostPerFoot is the host-supplied extra cost, for example
// dragging a Grappled creature (grappling.md FR-062).
type MoveSegmentInput struct {
	DistanceFeet     int
	Mode             MovementMode
	SpeedUse         SpeedUse
	Terrain          Terrain
	Occupants        []Occupant
	End              SegmentEnd
	ExtraCostPerFoot int
}

// MoveSegment is one stretch of a move in one mode over one kind of ground. The
// zero MoveSegment is invalid. Immutable; not comparable with == (use Equal).
type MoveSegment struct {
	distance, extra int
	mode            MovementMode
	use             SpeedUse
	terrain         Terrain
	occupants       []Occupant
	end             SegmentEnd
}

// NewMoveSegment returns the segment. It returns an error matching
// srderr.ErrInvalidInput for a distance of 0 or less, a negative extra cost, a
// mode, use, terrain, end or relation that is not defined, or an occupant with an
// invalid size or availability. The occupants are copied. It never panics.
func NewMoveSegment(in MoveSegmentInput) (MoveSegment, error) {
	if in.DistanceFeet <= 0 || in.ExtraCostPerFoot < 0 {
		return MoveSegment{}, invalidInput("segment needs a positive distance and no negative extra cost")
	}
	if !in.Mode.Valid() || (in.SpeedUse != SpeedUseWalking && in.SpeedUse != SpeedUseSpecial) ||
		(in.Terrain != TerrainOpen && in.Terrain != TerrainDifficult) ||
		(in.End != SegmentEndFree && in.End != SegmentEndOccupiedByCreature) {
		return MoveSegment{}, invalidInput("segment mode, speed use, terrain or end is not defined")
	}
	for _, o := range in.Occupants {
		if !o.Size.Valid() || (o.Relation != OccupantOther && o.Relation != OccupantAlly) ||
			(o.Availability != ConditionActionAvailabilityAvailable && o.Availability != ConditionActionAvailabilityNone) {
			return MoveSegment{}, invalidInput("occupant needs a size, a relation and an availability")
		}
	}
	return MoveSegment{in.DistanceFeet, in.ExtraCostPerFoot, in.Mode, in.SpeedUse, in.Terrain, slices.Clone(in.Occupants), in.End}, nil
}

// IsZero reports whether s is the invalid zero segment.
func (s MoveSegment) IsZero() bool { return s.distance == 0 }

// DistanceFeet returns the distance of the segment.
func (s MoveSegment) DistanceFeet() int { return s.distance }

// Mode returns how the segment is moved.
func (s MoveSegment) Mode() MovementMode { return s.mode }

// Equal reports whether s and o are the same segment.
func (s MoveSegment) Equal(o MoveSegment) bool {
	return s.distance == o.distance && s.extra == o.extra && s.mode == o.mode && s.use == o.use &&
		s.terrain == o.terrain && s.end == o.end && slices.Equal(s.occupants, o.occupants)
}

// segmentsEqual reports whether two lists of segments are equal in order.
func segmentsEqual(a, b []MoveSegment) bool { return slices.EqualFunc(a, b, MoveSegment.Equal) }

// usesSpecialSpeed reports whether the segment climbs or swims by the Climb or
// Swim speed, which removes the whole extra cost.
func (s MoveSegment) usesSpecialSpeed() bool {
	return s.use == SpeedUseSpecial && (s.mode == MovementModeClimb || s.mode == MovementModeSwim)
}

// isDifficult reports whether the ground is Difficult Terrain: declared so or
// held by an occupant that is neither Tiny nor an ally. However many things make
// it so, the extra cost is one foot per foot (SRD 5.2.1, "Difficult Terrain",
// p. 181).
func (s MoveSegment) isDifficult() bool {
	return s.terrain == TerrainDifficult || slices.ContainsFunc(s.occupants, func(o Occupant) bool {
		return o.Size != content.SizeTiny && o.Relation != OccupantAlly
	})
}

// costPerFoot is the movement one foot of the segment costs (FR-077).
func (s MoveSegment) costPerFoot() int {
	c := 1
	if !s.usesSpecialSpeed() {
		if s.mode == MovementModeClimb || s.mode == MovementModeSwim || s.mode == MovementModeCrawl {
			c++
		}
		if s.isDifficult() {
			c++
		}
	}
	return satAdd(c, s.extra)
}

// cost is the movement the whole segment costs, saturating.
func (s MoveSegment) cost() int { return satMulPositive(s.distance, s.costPerFoot()) }

// satMulPositive multiplies two positive ints, saturating at math.MaxInt.
func satMulPositive(a, b int) int {
	if a > math.MaxInt/b {
		return math.MaxInt
	}
	return a * b
}

// sizeGap is the number of sizes between a and b.
func sizeGap(a, b content.CreatureSize) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// canPassThrough reports whether a mover of size mover may move through the
// space of o: an ally, an Incapacitated creature, a Tiny creature or one at least
// two sizes larger or smaller (SRD 5.2.1, "Moving around Other Creatures", p. 14;
// "at least" is an interpretation).
func canPassThrough(mover content.CreatureSize, o Occupant) bool {
	return o.Relation == OccupantAlly || o.Availability == ConditionActionAvailabilityNone ||
		o.Size == content.SizeTiny || sizeGap(mover, o.Size) >= 2
}

// governing returns the speed that governs the segment: Fly for fly, Burrow for
// burrow, Climb or Swim for climb or swim by the special speed, else walking.
func (s MoveSegment) governing() SpeedName {
	switch {
	case s.mode == MovementModeFly:
		return SpeedNameFly
	case s.mode == MovementModeBurrow:
		return SpeedNameBurrow
	case s.mode == MovementModeClimb && s.use == SpeedUseSpecial:
		return SpeedNameClimb
	case s.mode == MovementModeSwim && s.use == SpeedUseSpecial:
		return SpeedNameSwim
	}
	return SpeedNameWalk
}

// moveJudgement is the verdict of judgeMove: the charges and totals of an
// affordable move, or the first refusal.
type moveJudgement struct {
	charges   []MovementCharge
	total     int
	moved     int
	rejection TurnRejection
}

// judgeMove judges segments in order for a mover of size mover with speeds, having
// moved feet already and a Dash extra of dash (FR-115): first every speed is
// checked, then every passage, then each segment is afforded in turn, then the end
// of the move. The first refusal wins.
func judgeMove(speeds MovementSpeeds, mover content.CreatureSize, moved, dash int, segs []MoveSegment) moveJudgement {
	for _, s := range segs {
		if _, ok := speeds.Of(s.governing()); !ok {
			return moveJudgement{rejection: TurnSpeedUnavailableRejection{Speed: s.governing()}}
		}
	}
	for i, s := range segs {
		if slices.ContainsFunc(s.occupants, func(o Occupant) bool { return !canPassThrough(mover, o) }) {
			return moveJudgement{rejection: TurnBlockedByCreatureRejection{SegmentIndex: i}}
		}
	}
	j := moveJudgement{moved: moved}
	for i, s := range segs {
		feet, _ := speeds.Of(s.governing())
		left := satAdd(satAdd(feet, dash), -j.moved)
		cost := s.cost()
		switch {
		case left <= 0:
			return moveJudgement{rejection: TurnNoMovementRemainingRejection{SegmentIndex: i, MovementLeft: left}}
		case left < cost:
			return moveJudgement{rejection: TurnInsufficientMovementRejection{SegmentIndex: i, ShortfallFeet: cost - left}}
		}
		j.moved = satAdd(j.moved, cost)
		j.total = satAdd(j.total, cost)
		j.charges = append(j.charges, MovementCharge{s.mode, cost})
	}
	if last := len(segs) - 1; last >= 0 && segs[last].end == SegmentEndOccupiedByCreature {
		return moveJudgement{rejection: TurnEndsInOccupiedSpaceRejection{SegmentIndex: last}}
	}
	return j
}
