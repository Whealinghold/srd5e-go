package character

import "strconv"

// PreserveLifeTarget are the host's facts about a Bloodied creature chosen for Preserve Life and
// the Hit Points allocated to it. Key is the host's opaque key and is never an attribute. Construct
// it with a keyed literal.
type PreserveLifeTarget struct {
	Key              string
	CurrentHitPoints int
	HitPointMaximum  int
	DistanceFeet     int
	Allocation       int
}

// PreserveLifeProblem is why a Preserve Life allocation is refused: the Reason (0 when it is not),
// the target it names, and for the Hit Point reasons the numbers behind it. Construct it with a keyed
// literal.
type PreserveLifeProblem struct {
	Reason ClericReason
	Target string
	// Room is the target's room, for allocation_exceeds_room: half its maximum rounded down, less its current Hit Points.
	Room int
	// Pool and Sum are the pool and the allocated total, for allocation_exceeds_pool.
	Pool, Sum int
}

// PreserveLife is the pure service of the Life Domain's Preserve Life Channel Divinity effect (SRD
// 5.2.1, "Level 3: Preserve Life", p. 40). It is stateless; the zero value is ready to use. Safe for
// concurrent use.
type PreserveLife struct{}

// Pool returns the Hit Points to divide: five times the Cleric level.
func (PreserveLife) Pool(level ClericLevel) int { return 5 * level.n }

// Room returns how many Hit Points a creature can regain before it is above half its Hit Point maximum:
// half the maximum rounded down, less its current Hit Points.
func (PreserveLife) Room(currentHitPoints, hitPointMaximum int) int {
	return hitPointMaximum/2 - currentHitPoints
}

// Validate checks the allocation to targets for a Cleric of the level and returns the first problem in
// the order of class-cleric.md table 4.14.1: duplicate_target, then per target in command order
// target_out_of_range (more than 30 feet), target_not_bloodied (Bloodied is IsBloodied), negative_allocation
// and allocation_exceeds_room, and last allocation_exceeds_pool. A zero Reason means the allocation is valid,
// including no target at all and a total below the pool.
//
// It returns an error matching srderr.ErrInvalidInput for the zero level, a negative current Hit Point count, a
// Hit Point maximum below 1 or a negative distance. It never panics.
func (p PreserveLife) Validate(level ClericLevel, targets []PreserveLifeTarget) (PreserveLifeProblem, error) {
	if level.IsZero() {
		return PreserveLifeProblem{}, invalidInput("Preserve Life without a Cleric level")
	}
	seen := map[string]bool{}
	for _, t := range targets {
		if seen[t.Key] {
			return PreserveLifeProblem{Reason: ClericDuplicateTarget, Target: t.Key}, nil
		}
		seen[t.Key] = true
	}
	sum := 0
	for _, t := range targets {
		maximum, err := NewHitPointMaximum(t.HitPointMaximum)
		if err != nil || t.DistanceFeet < 0 {
			return PreserveLifeProblem{}, invalidInput("Preserve Life target " + strconv.Quote(t.Key) + " with an invalid Hit Point maximum or distance")
		}
		bloodied, err := IsBloodied(t.CurrentHitPoints, maximum)
		if err != nil {
			return PreserveLifeProblem{}, err
		}
		room := p.Room(t.CurrentHitPoints, t.HitPointMaximum)
		switch {
		case t.DistanceFeet > clericEffectRangeFeet:
			return PreserveLifeProblem{Reason: ClericTargetOutOfRange, Target: t.Key}, nil
		case !bloodied:
			return PreserveLifeProblem{Reason: ClericTargetNotBloodied, Target: t.Key}, nil
		case t.Allocation < 0:
			return PreserveLifeProblem{Reason: ClericNegativeAllocation, Target: t.Key}, nil
		case t.Allocation > room:
			return PreserveLifeProblem{Reason: ClericAllocationExceedsRoom, Target: t.Key, Room: room}, nil
		}
		sum += t.Allocation
	}
	if pool := p.Pool(level); sum > pool {
		return PreserveLifeProblem{Reason: ClericAllocationExceedsPool, Pool: pool, Sum: sum}, nil
	}
	return PreserveLifeProblem{}, nil
}
