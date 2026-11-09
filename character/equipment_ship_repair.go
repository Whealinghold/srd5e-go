package character

import "github.com/Whealinghold/srd5e-go/content"

// SuppliesAndLabor says whether supplies and skilled labor are abundant, "such
// as a city shipyard" (SRD 5.2.1, "Ship Repair", p. 101). The zero value is
// invalid.
type SuppliesAndLabor int

// The supplies and labor states.
const (
	NotAbundant SuppliesAndLabor = iota + 1
	Abundant
)

// String returns "notAbundant", "abundant" or "unknown".
func (s SuppliesAndLabor) String() string {
	switch s {
	case NotAbundant:
		return "notAbundant"
	case Abundant:
		return "abundant"
	}
	return "unknown"
}

// ShipRepair is the cost and time to repair some Hit Points of a vehicle. It
// is an immutable value, comparable with ==.
type ShipRepair struct {
	cost     content.Cost
	halfDays int
}

// Cost returns the cost, in Gold Pieces.
func (r ShipRepair) Cost() content.Cost { return r.cost }

// HalfDays returns the time in half days, so that the halved time of abundant
// supplies is exact.
func (r ShipRepair) HalfDays() int { return r.halfDays }

// QuoteShipRepair returns the cost and time to repair hitPoints: 1 day and 20
// GP per Hit Point, both halved where supplies and skilled labor are abundant
// (SRD 5.2.1, "Ship Repair", p. 101). It returns an error matching
// srderr.ErrInvalidInput for hitPoints below 1, an invalid supplies state or a
// cost above [content.MaxCostAmount].
func QuoteShipRepair(hitPoints int, supplies SuppliesAndLabor) (ShipRepair, error) {
	var gold, halfDays int
	switch supplies {
	case NotAbundant:
		gold, halfDays = 20, 2
	case Abundant:
		gold, halfDays = 10, 1
	default:
		return ShipRepair{}, invalidInput("supplies and labor")
	}
	if hitPoints < 1 {
		return ShipRepair{}, invalidInput("hit points below 1")
	}
	if hitPoints > content.MaxCostAmount/gold {
		return ShipRepair{}, invalidInput("hit points out of range")
	}
	cost, err := content.NewCost(hitPoints*gold, content.CoinGP, 0)
	if err != nil {
		return ShipRepair{}, err
	}
	return ShipRepair{cost: cost, halfDays: hitPoints * halfDays}, nil
}

// ShipRepairOutcome is the closed set of results of [QuoteShipRepairFor]:
// [ShipRepairQuoted] and [ShipRepairExceedsShip].
//
//sumtype:decl
type ShipRepairOutcome interface{ isShipRepairOutcome() }

// ShipRepairQuoted holds the quote for the requested Hit Points.
type ShipRepairQuoted struct {
	Repair ShipRepair
}

// ShipRepairExceedsShip reports more Hit Points than the ship has.
type ShipRepairExceedsShip struct {
	RequestedHitPoints int
	ShipHitPoints      int
}

func (ShipRepairQuoted) isShipRepairOutcome()      {}
func (ShipRepairExceedsShip) isShipRepairOutcome() {}

// QuoteShipRepairFor is [QuoteShipRepair] bounded by the ship's Hit Points from
// the Vehicles table (SRD 5.2.1, p. 101). The SRD states no upper bound; the
// bound is package design, since the damage taken is unknown here. A caller who
// knows the real damage uses [QuoteShipRepair]. It returns an error for
// hitPoints below 1 as [QuoteShipRepair] does.
func QuoteShipRepairFor(ship content.AirborneOrWaterborneVehicle, hitPoints int, supplies SuppliesAndLabor) (ShipRepairOutcome, error) {
	if hitPoints > ship.HitPoints() && hitPoints >= 1 {
		if supplies != NotAbundant && supplies != Abundant {
			return nil, invalidInput("supplies and labor")
		}
		return ShipRepairExceedsShip{RequestedHitPoints: hitPoints, ShipHitPoints: ship.HitPoints()}, nil
	}
	r, err := QuoteShipRepair(hitPoints, supplies)
	if err != nil {
		return nil, err
	}
	return ShipRepairQuoted{Repair: r}, nil
}
