package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// limitedSpeedFeet is the Speed a creature carrying more than its Carry
// maximum can have at most (SRD 5.2.1, "Carrying Capacity", p. 178).
const limitedSpeedFeet = 5

// CarryingCapacity is the Carry and Drag/Lift/Push maximums of a creature from
// its size and Strength score (SRD 5.2.1, "Carrying Capacity", p. 178). It is
// an immutable value, comparable with ==. The zero value is a capacity of 0.
type CarryingCapacity struct {
	carryHalfPounds int
	dragPounds      int
}

// NewCarryingCapacity returns the capacity of a creature of the given size
// and Strength score. The size is used as given (a Goliath's Powerful Build
// is the host's concern). An invalid size or the zero score gives a capacity
// of 0.
//
// The Carry maximum is Strength times 7.5 (Tiny), 15 (Small, Medium), 30
// (Large), 60 (Huge) or 120 (Gargantuan) pounds; the Drag/Lift/Push maximum is
// Strength times 15, 30, 60, 120 or 240 pounds.
func NewCarryingCapacity(size content.CreatureSize, strength ability.Score) CarryingCapacity {
	// The two multipliers are the same integer: Carry in half pounds,
	// Drag/Lift/Push in pounds.
	per := 0
	switch size {
	case content.SizeTiny:
		per = 15
	case content.SizeSmall, content.SizeMedium:
		per = 30
	case content.SizeLarge:
		per = 60
	case content.SizeHuge:
		per = 120
	case content.SizeGargantuan:
		per = 240
	}
	return CarryingCapacity{carryHalfPounds: strength.Value() * per, dragPounds: strength.Value() * per}
}

// CarryHalfPounds returns the Carry maximum in half pounds, so that a Tiny
// creature's 7.5 pounds per point of Strength is exact.
func (c CarryingCapacity) CarryHalfPounds() int { return c.carryHalfPounds }

// DragLiftPushPounds returns the Drag/Lift/Push maximum in pounds.
func (c CarryingCapacity) DragLiftPushPounds() int { return c.dragPounds }

// CanCarry reports whether loadPounds does not exceed the Carry maximum. It
// returns an error matching srderr.ErrInvalidInput for a negative load.
func (c CarryingCapacity) CanCarry(loadPounds int) (bool, error) {
	if loadPounds < 0 {
		return false, invalidInput("negative load")
	}
	return int64(loadPounds)*2 <= int64(c.carryHalfPounds), nil
}

// LoadAssessment is the closed set of results of [CarryingCapacity.Assess]:
// [LoadWithinCarry], [LoadSpeedLimited] and [LoadBeyondDragLiftPush].
//
//sumtype:decl
type LoadAssessment interface{ isLoadAssessment() }

// LoadWithinCarry is a load not above the Carry maximum.
type LoadWithinCarry struct{}

// LoadSpeedLimited is a load above the Carry maximum and not above the
// Drag/Lift/Push maximum: Speed can be no more than [LoadSpeedLimited.LimitFeet].
// It does not apply the limit to a Speed; that is the movement rules' job.
type LoadSpeedLimited struct{}

// LoadBeyondDragLiftPush is a load above the Drag/Lift/Push maximum, which
// cannot be moved (a package-design reading of the table's maximum).
type LoadBeyondDragLiftPush struct{}

func (LoadWithinCarry) isLoadAssessment()        {}
func (LoadSpeedLimited) isLoadAssessment()       {}
func (LoadBeyondDragLiftPush) isLoadAssessment() {}

// LimitFeet returns 5, the most a limited Speed can be.
func (LoadSpeedLimited) LimitFeet() int { return limitedSpeedFeet }

// Assess classifies loadPounds against the two maximums. It returns an error
// matching srderr.ErrInvalidInput for a negative load; 0 is within Carry.
func (c CarryingCapacity) Assess(loadPounds int) (LoadAssessment, error) {
	carry, err := c.CanCarry(loadPounds)
	switch {
	case err != nil:
		return nil, err
	case carry:
		return LoadWithinCarry{}, nil
	case int64(loadPounds) <= int64(c.dragPounds):
		return LoadSpeedLimited{}, nil
	}
	return LoadBeyondDragLiftPush{}, nil
}
