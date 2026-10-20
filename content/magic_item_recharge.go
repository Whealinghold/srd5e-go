package content

import "github.com/Whealinghold/srd5e-go/dice"

// ItemRechargeTime is when an item with charges recharges (SRD 5.2.1,
// "Charges", "The Next Dawn", p. 206), a closed set of three members. The
// zero ItemRechargeTime is invalid. Comparable and safe for concurrent use.
type ItemRechargeTime int

// The item recharge times.
const (
	RechargesAtNextDawn ItemRechargeTime = iota + 1
	RechargesAtSpecifiedTime
	DoesNotRecharge
)

func itemRechargeTimeNames() []string {
	return []string{"nextDawn", "specifiedTime", "doesNotRecharge"}
}

// Valid reports whether t is one of the three recharge times.
func (t ItemRechargeTime) Valid() bool { return enumValid(itemRechargeTimeNames(), int(t)) }

// String returns the name of t, or "unknown" for an undefined value.
func (t ItemRechargeTime) String() string { return enumString(itemRechargeTimeNames(), int(t)) }

// ChargeRecovery is how many charges an item regains when it recharges, a
// closed family with four variants: [RegainsAll], [RegainsFixed],
// [RegainsRolled] and [NoChargeRecovery]. It is data only: no member computes
// an amount, because the amount depends on the item's charges, which this
// package never reads. Consumers switch over it exhaustively. Every variant is
// an immutable, comparable value and safe for concurrent use.
type ChargeRecovery interface{ isChargeRecovery() }

// RegainsAll is the recovery of all charges. The zero value is valid.
type RegainsAll struct{}

func (RegainsAll) isChargeRecovery() {}

// RegainsFixed is the recovery of a fixed number of charges. The zero value is
// invalid; build it with [NewRegainsFixed].
type RegainsFixed struct{ charges int }

// NewRegainsFixed returns the recovery of n charges, n being at least 1.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for n below 1. It never panics.
func NewRegainsFixed(n int) (RegainsFixed, error) {
	if n < 1 {
		return RegainsFixed{}, blockError("RegainsFixed", BlockOutOfRange)
	}
	return RegainsFixed{charges: n}, nil
}

// Charges returns the number of charges regained.
func (r RegainsFixed) Charges() int { return r.charges }

func (RegainsFixed) isChargeRecovery() {}

// RegainsRolled is the recovery of a rolled number of charges. The zero value
// is invalid; build it with [NewRegainsRolled].
type RegainsRolled struct{ roll dice.Expression }

// NewRegainsRolled returns the recovery of the charges rolled by roll.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockMissing for the zero expression. It never panics.
func NewRegainsRolled(roll dice.Expression) (RegainsRolled, error) {
	if roll.IsZero() {
		return RegainsRolled{}, blockError("RegainsRolled", BlockMissing)
	}
	return RegainsRolled{roll: roll}, nil
}

// Roll returns the dice that give the number of charges.
func (r RegainsRolled) Roll() dice.Expression { return r.roll }

func (RegainsRolled) isChargeRecovery() {}

// NoChargeRecovery is an item with no stated recovery. The zero value is
// valid.
type NoChargeRecovery struct{}

func (NoChargeRecovery) isChargeRecovery() {}

// validChargeRecovery reports whether c is a variant its constructor would
// accept.
func validChargeRecovery(c ChargeRecovery) bool {
	switch v := c.(type) {
	case RegainsAll, NoChargeRecovery:
		return true
	case RegainsFixed:
		_, err := NewRegainsFixed(v.charges)
		return err == nil
	case RegainsRolled:
		_, err := NewRegainsRolled(v.roll)
		return err == nil
	}
	return false
}
