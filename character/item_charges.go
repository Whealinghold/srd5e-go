package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ItemChargesReason says why NewItemCharges refused its values. The zero value
// is not a reason.
type ItemChargesReason int

// The reasons.
const (
	ItemChargesMaximumBelowOne ItemChargesReason = iota + 1
	ItemChargesCurrentNegative
	ItemChargesCurrentAboveMaximum
)

// String returns the snake_case name of the reason, or "unknown".
func (r ItemChargesReason) String() string {
	names := []string{"maximum_below_one", "current_negative", "current_above_maximum"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// ItemChargesRejected is the error NewItemCharges returns. It matches
// srderr.ErrInvalidInput.
//
//nolint:errname // the name is the spec's (FR-119); it is an error with typed data
type ItemChargesRejected struct{ Reason ItemChargesReason }

// Error renders the error. The text is not a contract.
func (e *ItemChargesRejected) Error() string { return "character: item charges: " + e.Reason.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *ItemChargesRejected) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is an *ItemChargesRejected with the same Reason.
func (e *ItemChargesRejected) Is(target error) bool {
	t, ok := target.(*ItemChargesRejected)
	return ok && t != nil && e != nil && *t == *e
}

// ItemCharges are the charges of a magic item: a maximum of at least 1 and a
// current number from 0 to the maximum (SRD 5.2.1, "Charges", p. 206). Immutable
// and comparable; the zero value is invalid.
type ItemCharges struct{ current, maximum int }

// NewItemCharges returns the charges. It returns an *ItemChargesRejected
// (matching srderr.ErrInvalidInput) for a maximum below 1, a negative current or
// a current above the maximum. It never panics.
func NewItemCharges(maximum, current int) (ItemCharges, error) {
	switch {
	case maximum < 1:
		return ItemCharges{}, &ItemChargesRejected{Reason: ItemChargesMaximumBelowOne}
	case current < 0:
		return ItemCharges{}, &ItemChargesRejected{Reason: ItemChargesCurrentNegative}
	case current > maximum:
		return ItemCharges{}, &ItemChargesRejected{Reason: ItemChargesCurrentAboveMaximum}
	}
	return ItemCharges{current: current, maximum: maximum}, nil
}

// MustItemCharges is NewItemCharges for literals; it panics with the error.
func MustItemCharges(maximum, current int) ItemCharges {
	c, err := NewItemCharges(maximum, current)
	if err != nil {
		panic(err)
	}
	return c
}

// Current returns the charges left.
func (c ItemCharges) Current() int { return c.current }

// Maximum returns the most charges the item holds.
func (c ItemCharges) Maximum() int { return c.maximum }

// IsZero reports whether c is the invalid zero ItemCharges.
func (c ItemCharges) IsZero() bool { return c.maximum == 0 }

// ItemChargesSpendResult is the closed set of outcomes of ItemCharges.Spend.
// Variants: ItemChargesSpendApplied, MagicItemInsufficientCharges.
type ItemChargesSpendResult interface{ isItemChargesSpendResult() }

// ItemChargesSpendApplied: the charges were spent.
type ItemChargesSpendApplied struct {
	Charges ItemCharges
	Event   MagicItemChargesSpent
}

// MagicItemInsufficientCharges: more charges were asked for than the item holds;
// the charges are unchanged.
type MagicItemInsufficientCharges struct{ Current, Requested int }

func (ItemChargesSpendApplied) isItemChargesSpendResult()      {}
func (MagicItemInsufficientCharges) isItemChargesSpendResult() {}

// Spend expends n charges, from 1 to Current, with the MagicItemChargesSpent
// event for holder and instance. More than Current is MagicItemInsufficientCharges.
// It returns an error matching srderr.ErrInvalidInput for n below 1, a zero
// holder, a blank instance or zero charges.
func (c ItemCharges) Spend(holder Identifier, instance MagicItemInstanceID, n int) (ItemChargesSpendResult, error) {
	if n < 1 || holder.IsZero() || !instance.Valid() || c.IsZero() {
		return nil, invalidInput("spend charges: n below 1, zero holder, blank instance or zero charges")
	}
	if n > c.current {
		return MagicItemInsufficientCharges{Current: c.current, Requested: n}, nil
	}
	return ItemChargesSpendApplied{Charges: ItemCharges{current: c.current - n, maximum: c.maximum},
		Event: MagicItemChargesSpent{id: holder, instance: instance, spent: n}}, nil
}

// ChargeKnowledge is what a creature knows of an item's charges.
type ChargeKnowledge struct{ remainingKnown, regainKnown bool }

// KnowledgeOfCharges returns what a creature knows: the charges remaining when
// attuned or after an Identify cast, and how many it regains only when attuned.
func KnowledgeOfCharges(standing AttunementStanding, identifyCast bool) ChargeKnowledge {
	attuned := standing == StandingAttuned
	return ChargeKnowledge{remainingKnown: attuned || identifyCast, regainKnown: attuned}
}

// RemainingKnown reports whether the charges remaining are known.
func (k ChargeKnowledge) RemainingKnown() bool { return k.remainingKnown }

// RegainKnown reports whether the number regained, its recovery and schedule are known.
func (k ChargeKnowledge) RegainKnown() bool { return k.regainKnown }

// RechargeMoment is the moment the host reports. The zero value is invalid.
type RechargeMoment int

// The moments.
const (
	RechargeNextDawn RechargeMoment = iota + 1
	RechargeSpecifiedTime
)

// String returns "next_dawn", "specified_time" or "unknown".
func (m RechargeMoment) String() string {
	switch m {
	case RechargeNextDawn:
		return "next_dawn"
	case RechargeSpecifiedTime:
		return "specified_time"
	}
	return "unknown"
}

// ChargeRecoveryRules holds the recharge rules. The zero value is ready to use.
type ChargeRecoveryRules struct{}

// Due reports whether an item with schedule time recharges at moment.
func (ChargeRecoveryRules) Due(time content.ItemRechargeTime, moment RechargeMoment) bool {
	return time == content.RechargesAtNextDawn && moment == RechargeNextDawn ||
		time == content.RechargesAtSpecifiedTime && moment == RechargeSpecifiedTime
}

// RegainedFor is the number of charges regained: RegainsAll the expended count,
// RegainsFixed its number, RegainsRolled rolled, NoChargeRecovery 0; capped at
// the expended count so current never exceeds maximum. It returns an error
// matching srderr.ErrInvalidInput for a negative rolled value, zero charges or
// an unknown recovery.
func RegainedFor(recovery content.ChargeRecovery, charges ItemCharges, rolled int) (int, error) {
	if rolled < 0 || charges.IsZero() {
		return 0, invalidInput("regained charges: negative rolled value or zero charges")
	}
	expended := charges.maximum - charges.current
	var n int
	switch r := recovery.(type) {
	case content.RegainsAll:
		n = expended
	case content.RegainsFixed:
		n = r.Charges()
	case content.RegainsRolled:
		n = rolled
	case content.NoChargeRecovery:
		n = 0
	default:
		return 0, invalidInput("regained charges: nil or foreign recovery")
	}
	return min(n, expended), nil
}
