package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func shipOf(t *testing.T, id string, hp int) content.AirborneOrWaterborneVehicle {
	t.Helper()
	return must(content.NewAirborneOrWaterborneVehicle(content.ItemID(id), id, content.SourceCitation{}, 1, 1, 0, 0, 11, hp, 0, content.NoWeight{}, price(gp(50))))
}

// Requirement: per case equipSpec FR-029, FR-030 (SRD 5.2.1, "Ship Repair", p.
// 101) - the "Ship repair" rows: 20 GP and 2 half-days per Hit Point, both
// halved with abundant supplies.
// Path: per case (happy; unhappy boundary).
func TestQuoteShipRepair(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		hp       int
		supplies character.SuppliesAndLabor
		gold     int
		halfDays int
	}{
		{"50 notAbundant", 50, character.NotAbundant, 1000, 100}, {"50 abundant", 50, character.Abundant, 500, 50},
		{"1 notAbundant", 1, character.NotAbundant, 20, 2}, {"1 abundant", 1, character.Abundant, 10, 1},
		{"3 abundant", 3, character.Abundant, 30, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := character.QuoteShipRepair(c.hp, c.supplies)
			if err != nil || r.Cost() != gp(c.gold) || r.HalfDays() != c.halfDays {
				t.Errorf("quote = %v, %d half-days, %v; want %d GP, %d", r.Cost(), r.HalfDays(), err, c.gold, c.halfDays)
			}
		})
	}
}

// Requirement: per case equipSpec FR-031 - fewer than 1 Hit Point, an invalid
// supplies state or a cost beyond the largest Cost is an srderr.ErrInvalidInput
// error.
// Path: unhappy (invalid input).
func TestQuoteShipRepairInvalid(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		hp       int
		supplies character.SuppliesAndLabor
	}{
		{0, character.NotAbundant}, {-1, character.Abundant}, {1, 0}, {1, 3},
		{content.MaxCostAmount/20 + 1, character.NotAbundant}, {content.MaxCostAmount/10 + 1, character.Abundant},
	} {
		_, err := character.QuoteShipRepair(c.hp, c.supplies)
		wantInvalid(t, err, "quote")
	}
	if _, err := character.QuoteShipRepair(content.MaxCostAmount/20, character.NotAbundant); err != nil {
		t.Errorf("largest quotable repair: %v", err)
	}
	if character.NotAbundant.String() != "notAbundant" || character.Abundant.String() != "abundant" || character.SuppliesAndLabor(0).String() != "unknown" {
		t.Error("SuppliesAndLabor names")
	}
}

// Requirement: per case equipSpec FR-032 (Vehicles table HP, p. 101) - a repair
// up to the ship's Hit Points is quoted, above them it is reported with both
// figures.
// Path: unhappy (boundary).
func TestQuoteShipRepairFor(t *testing.T) {
	t.Parallel()
	rowboat, sailing := shipOf(t, "rowboat_x", 50), shipOf(t, "sailing_x", 300)
	q, err := character.QuoteShipRepairFor(rowboat, 50, character.NotAbundant)
	if got, ok := q.(character.ShipRepairQuoted); err != nil || !ok || got.Repair.Cost() != gp(1000) || got.Repair.HalfDays() != 100 {
		t.Errorf("Rowboat 50 = %v, %v", q, err)
	}
	q, err = character.QuoteShipRepairFor(rowboat, 51, character.NotAbundant)
	if got, ok := q.(character.ShipRepairExceedsShip); err != nil || !ok || got.RequestedHitPoints != 51 || got.ShipHitPoints != 50 {
		t.Errorf("Rowboat 51 = %v, %v", q, err)
	}
	q, err = character.QuoteShipRepairFor(sailing, 300, character.Abundant)
	if got, ok := q.(character.ShipRepairQuoted); err != nil || !ok || got.Repair.Cost() != gp(3000) || got.Repair.HalfDays() != 300 {
		t.Errorf("Sailing Ship 300 = %v, %v", q, err)
	}
	for _, c := range []struct {
		hp       int
		supplies character.SuppliesAndLabor
	}{{0, character.Abundant}, {-1, character.Abundant}, {51, 0}, {1, 0}} {
		_, err = character.QuoteShipRepairFor(rowboat, c.hp, c.supplies)
		wantInvalid(t, err, "ForShip")
	}
}
