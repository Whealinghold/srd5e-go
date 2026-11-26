package combat

import (
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
)

// minBreathCapacity is the least time a creature can hold its breath.
const minBreathCapacity = 30 * time.Second

// BreathCapacity is the rule for how long a creature can hold its breath: the
// longer of 30 seconds and 1 + its Constitution modifier minutes (SRD 5.2.1,
// "Suffocation [Hazard]", p. 189). The zero value is ready to use.
type BreathCapacity struct{}

// ForConstitution returns how long a creature with the Constitution modifier can
// hold its breath.
func (BreathCapacity) ForConstitution(m ability.Modifier) time.Duration {
	return max(minBreathCapacity, time.Duration(1+m.Value())*time.Minute)
}

// SuffocationCause is why a creature is suffocating (SRD 5.2.1, "Suffocation
// [Hazard]", p. 189). The zero value is invalid.
type SuffocationCause int

// The causes.
const (
	// SuffocationRanOutOfBreath: the creature held its breath as long as it could.
	SuffocationRanOutOfBreath SuffocationCause = iota + 1
	// SuffocationChoking: the creature is choking.
	SuffocationChoking
)

// Valid reports whether c is one of the two causes.
func (c SuffocationCause) Valid() bool {
	return c == SuffocationRanOutOfBreath || c == SuffocationChoking
}

// String returns "ran_out_of_breath", "choking" or "unknown".
func (c SuffocationCause) String() string {
	return enumName([]string{"ran_out_of_breath", "choking"}, int(c))
}

// BreathState is how a creature is breathing. Variants: BreathNormal,
// BreathHolding, BreathSuffocating. A nil BreathState is BreathNormal.
type BreathState interface{ isBreathState() }

// BreathNormal: the creature breathes freely. The zero value is ready to use.
type BreathNormal struct{}

// BreathHolding: the creature holds its breath, for Elapsed of Capacity so far.
// Build it with NewBreathHolding; the zero value has no capacity and is invalid.
type BreathHolding struct{ elapsed, capacity time.Duration }

// NewBreathHolding returns the state of a creature that has held its breath for
// elapsed of capacity. It returns an error matching srderr.ErrInvalidInput for a
// negative elapsed, a capacity below 1 or an elapsed that is not below capacity
// (that creature is suffocating). It never panics.
func NewBreathHolding(elapsed, capacity time.Duration) (BreathHolding, error) {
	if elapsed < 0 || elapsed >= capacity {
		return BreathHolding{}, invalidInput("breath held needs 0 <= elapsed < capacity")
	}
	return BreathHolding{elapsed, capacity}, nil
}

// Elapsed returns how long the creature has held its breath.
func (b BreathHolding) Elapsed() time.Duration { return b.elapsed }

// Capacity returns how long the creature can hold its breath.
func (b BreathHolding) Capacity() time.Duration { return b.capacity }

// BreathSuffocating: the creature ran out of breath or is choking. Build it with
// NewBreathSuffocating; the zero value has no cause and is invalid.
type BreathSuffocating struct{ cause SuffocationCause }

// NewBreathSuffocating returns the state for cause. It returns an error matching
// srderr.ErrInvalidInput for a cause that is not valid.
func NewBreathSuffocating(cause SuffocationCause) (BreathSuffocating, error) {
	if !cause.Valid() {
		return BreathSuffocating{}, invalidInput("invalid suffocation cause")
	}
	return BreathSuffocating{cause}, nil
}

// Cause returns why the creature is suffocating.
func (b BreathSuffocating) Cause() SuffocationCause { return b.cause }

func (BreathNormal) isBreathState()      {}
func (BreathHolding) isBreathState()     {}
func (BreathSuffocating) isBreathState() {}

// validBreathState reports whether s is nil or a valid variant.
func validBreathState(s BreathState) bool {
	switch v := s.(type) {
	case nil, BreathNormal:
		return true
	case BreathHolding:
		return v.elapsed < v.capacity
	case BreathSuffocating:
		return v.cause.Valid()
	}
	return false
}

// BreathChange is a change the host reports to AdvanceBreath. Variants:
// BeginHoldingBreath, HoldBreathFor, BeginChoking.
type BreathChange interface{ isBreathChange() }

// BeginHoldingBreath: the creature starts holding its breath. Construct it with a
// keyed literal.
type BeginHoldingBreath struct{ ConstitutionModifier ability.Modifier }

// HoldBreathFor: the creature keeps holding its breath for Elapsed longer. A zero
// Elapsed changes nothing; a negative one is an error. Construct it with a keyed
// literal.
type HoldBreathFor struct{ Elapsed time.Duration }

// BeginChoking: the creature starts choking.
type BeginChoking struct{}

func (BeginHoldingBreath) isBreathChange() {}
func (HoldBreathFor) isBreathChange()      {}
func (BeginChoking) isBreathChange()       {}

// BreathRejection is why a breathing change was refused. Refusals are results, not
// errors. The zero value is not a rejection.
type BreathRejection int

// The rejections.
const (
	// BreathRejectionNotBreathing: BeginHoldingBreath, but the creature was not breathing normally.
	BreathRejectionNotBreathing BreathRejection = iota + 1
	// BreathRejectionNotHoldingBreath: HoldBreathFor, but the creature was not holding its breath.
	BreathRejectionNotHoldingBreath
	// BreathRejectionAlreadySuffocating: BeginChoking, but the creature was already suffocating.
	BreathRejectionAlreadySuffocating
)

// String returns the snake_case name of r (the srd5e.combat.result value), or
// "unknown".
func (r BreathRejection) String() string {
	return enumName([]string{"not_breathing", "not_holding_breath", "already_suffocating"}, int(r))
}

// advanceBreath is the rule of AdvanceBreath. It returns the new state and the
// event the change produced, or a rejection (non-zero) with the state unchanged.
// The error is ErrInvalidInput for a negative elapsed time or a change that is not
// one of the variants.
func advanceBreath(state BreathState, change BreathChange, target CombatantID) (BreathState, Event, BreathRejection, error) {
	base := creatureEnvironmentEventBase{target}
	_, normal := state.(BreathNormal)
	normal = normal || state == nil
	switch c := change.(type) {
	case BeginHoldingBreath:
		if !normal {
			return state, nil, BreathRejectionNotBreathing, nil
		}
		return BreathHolding{0, BreathCapacity{}.ForConstitution(c.ConstitutionModifier)}, BreathHeldStarted{base}, 0, nil
	case HoldBreathFor:
		if c.Elapsed < 0 {
			return nil, nil, 0, invalidInput("elapsed time below 0")
		}
		h, ok := state.(BreathHolding)
		if !ok {
			return state, nil, BreathRejectionNotHoldingBreath, nil
		}
		if c.Elapsed >= h.capacity-h.elapsed {
			return BreathSuffocating{SuffocationRanOutOfBreath}, RanOutOfBreath{base}, 0, nil
		}
		return BreathHolding{h.elapsed + c.Elapsed, h.capacity}, nil, 0, nil
	case BeginChoking:
		if _, suffocating := state.(BreathSuffocating); suffocating {
			return state, nil, BreathRejectionAlreadySuffocating, nil
		}
		return BreathSuffocating{SuffocationChoking}, BeganChoking{base}, 0, nil
	}
	return nil, nil, 0, invalidInput("breath change is not a known variant")
}

// suffocationTurnEnd is the rule of ResolveSuffocationTurnEnd: a suffocating
// creature gains one Exhaustion level at the end of its turn (SRD 5.2.1,
// "Suffocation [Hazard]", p. 189).
func suffocationTurnEnd(state BreathState, set ConditionSet, target CombatantID, rec HazardRecord) dayEnd {
	rec = rec.clamped(set.exhaustionLevel())
	out := dayEnd{outcome: HazardNoExhaustion, set: set, record: rec}
	if _, suffocating := state.(BreathSuffocating); !suffocating {
		return out
	}
	a := applyExhaustion(set, target)
	out.outcome, out.set, out.events = a.outcome, a.set, a.events
	if a.outcome == HazardExhaustionGained {
		out.record.suffocation++
	}
	out.record = out.record.clamped(out.set.exhaustionLevel())
	return out
}
