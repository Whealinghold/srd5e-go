package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Jumping (SRD 5.2.1, "Jumping", "Long Jump", "High Jump", pp. 183 to 185). Pure and
// silent: nothing is rolled, and the host charges a permitted jump to its movement ledger.

// runningStartFeet is the movement before a jump that makes it a running one.
const runningStartFeet = 10

// JumpKind is a Long Jump (horizontal) or a High Jump (vertical). The zero JumpKind is invalid.
type JumpKind int

// The two kinds.
const (
	JumpKindLong JumpKind = iota + 1
	JumpKindHigh
)

// Valid reports whether k is one of the two kinds.
func (k JumpKind) Valid() bool { return k >= JumpKindLong && k <= JumpKindHigh }

// String returns "long", "high" or "unknown".
func (k JumpKind) String() string { return enumName([]string{"long", "high"}, int(k)) }

// JumpApproach is how a jump begins. The zero JumpApproach is invalid.
type JumpApproach int

// The two approaches.
const (
	// JumpApproachRunning follows at least 10 feet of movement (on foot for a High Jump).
	JumpApproachRunning JumpApproach = iota + 1
	// JumpApproachStanding is a jump from a standing start (the SRD's word).
	JumpApproachStanding
)

// Valid reports whether a is one of the two approaches.
func (a JumpApproach) Valid() bool { return a >= JumpApproachRunning && a <= JumpApproachStanding }

// String returns "running", "standing" or "unknown".
func (a JumpApproach) String() string { return enumName([]string{"running", "standing"}, int(a)) }

// JumpApproachFor returns Running when the creature moved at least 10 feet immediately
// before the jump (for a High Jump, on foot, that is mode Walk) and Standing otherwise. It
// returns an error matching srderr.ErrInvalidInput for a negative run-up, an invalid kind
// or an invalid mode. It never panics.
func JumpApproachFor(kind JumpKind, runUpFeet int, runUpMode MovementMode) (JumpApproach, error) {
	switch {
	case !kind.Valid():
		return 0, invalidInput("invalid jump kind")
	case runUpFeet < 0:
		return 0, invalidInput("negative run-up")
	case !runUpMode.Valid():
		return 0, invalidInput("invalid run-up mode")
	}
	if runUpFeet >= runningStartFeet && (kind == JumpKindLong || runUpMode == MovementModeWalk) {
		return JumpApproachRunning, nil
	}
	return JumpApproachStanding, nil
}

// JumpMaximumFeet returns the greatest distance (Long Jump) or height (High Jump) in feet
// for the Strength score: a running Long Jump is the score, a running High Jump 3 plus the
// Strength modifier and never below 0, and a standing jump half of that rounded down (SRD
// 5.2.1, pp. 183 to 185; "Round Down", p. 187). An invalid kind or approach gives 0.
func JumpMaximumFeet(kind JumpKind, strength ability.Score, approach JumpApproach) int {
	var running int
	switch kind {
	case JumpKindLong:
		running = strength.Value()
	case JumpKindHigh:
		running = max(0, 3+strength.Modifier().Value())
	default:
		return 0
	}
	switch approach {
	case JumpApproachRunning:
		return running
	case JumpApproachStanding:
		return running / 2
	}
	return 0
}

// JumpAttempt is a jump to judge: the kind, the Strength score, the conditions, the approach
// derived from the movement made before, the distance asked and the walking movement left.
// Build it with NewJumpAttempt. The zero JumpAttempt is invalid (IsZero). Comparable with ==.
type JumpAttempt struct {
	kind         JumpKind
	strength     ability.Score
	conditions   ConditionSet
	approach     JumpApproach
	requested    int
	movementLeft int
}

// NewJumpAttempt returns the attempt, deriving the approach from the run-up. A negative
// movementLeftFeet is a legal state (a Speed drop) and is stored as 0. It returns an error
// matching srderr.ErrInvalidInput for an invalid kind or mode, a zero Strength score, or a
// negative run-up or requested distance. It never panics.
func NewJumpAttempt(kind JumpKind, strength ability.Score, conditions ConditionSet, runUpFeet int, runUpMode MovementMode, requestedFeet, movementLeftFeet int) (JumpAttempt, error) {
	approach, err := JumpApproachFor(kind, runUpFeet, runUpMode)
	switch {
	case err != nil:
		return JumpAttempt{}, err
	case strength.IsZero():
		return JumpAttempt{}, invalidInput("zero Strength score")
	case requestedFeet < 0:
		return JumpAttempt{}, invalidInput("negative jump distance")
	}
	return JumpAttempt{kind, strength, conditions, approach, requestedFeet, max(0, movementLeftFeet)}, nil
}

// IsZero reports whether a is the invalid zero attempt.
func (a JumpAttempt) IsZero() bool { return a == JumpAttempt{} }

// JumpRefusal is why a jump is refused, in order of precedence. The zero value is invalid.
type JumpRefusal int

// The refusals, first match wins.
const (
	// JumpRefusalProneRestrictedMovement: a Prone creature can only crawl or right itself (p. 186).
	JumpRefusalProneRestrictedMovement JumpRefusal = iota + 1
	// JumpRefusalExceedsMaximumDistance: the request is above the maximum.
	JumpRefusalExceedsMaximumDistance
	// JumpRefusalInsufficientMovement: the request is above the movement left.
	JumpRefusalInsufficientMovement
)

// Valid reports whether r is one of the three refusals.
func (r JumpRefusal) Valid() bool {
	return r >= JumpRefusalProneRestrictedMovement && r <= JumpRefusalInsufficientMovement
}

// String returns "prone_restricted_movement", "exceeds_maximum_distance",
// "insufficient_movement" or "unknown".
func (r JumpRefusal) String() string {
	return enumName([]string{"prone_restricted_movement", "exceeds_maximum_distance", "insufficient_movement"}, int(r))
}

// JumpAssessment is the closed set of verdicts of AssessJump. Variants: JumpPermitted,
// JumpRefused.
//
//sumtype:decl
type JumpAssessment interface{ isJumpAssessment() }

// JumpPermitted is a jump the creature may make. MovementCostFeet equals DistanceFeet, one
// foot of movement per foot jumped (SRD 5.2.1, "Long Jump", p. 185). Build it with a keyed literal.
type JumpPermitted struct {
	Kind             JumpKind
	Approach         JumpApproach
	MaximumFeet      int
	DistanceFeet     int
	MovementCostFeet int
}

// JumpRefused is a jump the creature may not make. Build it with a keyed literal.
type JumpRefused struct {
	Kind             JumpKind
	Approach         JumpApproach
	Reason           JumpRefusal
	MaximumFeet      int
	RequestedFeet    int
	MovementLeftFeet int
}

func (JumpPermitted) isJumpAssessment() {}
func (JumpRefused) isJumpAssessment()   {}

// AssessJump judges the attempt. A request of 0 feet is always permitted (a jump of 0 feet
// is no movement). Otherwise the first match refuses: a Prone creature (also by implication
// from Unconscious), a request above the maximum, a request above the movement left; a
// request equal to either limit is permitted. It rolls nothing and cannot fail. The zero
// attempt gives the zero JumpRefused.
func AssessJump(a JumpAttempt) JumpAssessment {
	if a.IsZero() {
		return JumpRefused{}
	}
	maximum := JumpMaximumFeet(a.kind, a.strength, a.approach)
	refuse := func(r JumpRefusal) JumpAssessment {
		return JumpRefused{a.kind, a.approach, r, maximum, a.requested, a.movementLeft}
	}
	switch {
	case a.requested == 0:
	case slices.Contains(a.conditions.Effective(), content.ConditionProne):
		return refuse(JumpRefusalProneRestrictedMovement)
	case a.requested > maximum:
		return refuse(JumpRefusalExceedsMaximumDistance)
	case a.requested > a.movementLeft:
		return refuse(JumpRefusalInsufficientMovement)
	}
	return JumpPermitted{a.kind, a.approach, maximum, a.requested, a.requested}
}
