package content_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const equipSpec = "doc/specs/content-equipment.md"

// The fixtures of the equipment tests are synthetic: made-up ids, names and
// numbers, never SRD data.

func fixedCost(amount int, coin content.Coin) content.FixedCost {
	return content.MustFixedCost(content.MustCost(amount, coin, 0))
}

// Requirement: equipSpec FR-009 (SRD 5.2.1, "Coins", p. 89) - the five coins
// in ascending order are worth 1, 10, 50, 100 and 1000 Copper Pieces; an invalid
// coin is worth 0 and does not print as a member.
// Path: per case.
func TestCoins(t *testing.T) {
	t.Parallel()
	cases := []struct {
		coin   content.Coin
		name   string
		copper int
	}{
		{content.CoinCP, "CP", 1}, {content.CoinSP, "SP", 10}, {content.CoinEP, "EP", 50},
		{content.CoinGP, "GP", 100}, {content.CoinPP, "PP", 1000},
		{content.Coin(0), "unknown", 0}, {content.Coin(6), "unknown", 0},
	}
	for _, c := range cases {
		if c.coin.String() != c.name || c.coin.ValueInCopper() != c.copper || c.coin.Valid() != (c.copper != 0) {
			t.Errorf("Coin(%d) = %q worth %d valid %v, want %q worth %d [FR-009]", int(c.coin), c.coin, c.coin.ValueInCopper(), c.coin.Valid(), c.name, c.copper)
		}
	}
	if content.CostPerDay.String() != "Day" || content.CostPerMile.String() != "Mile" || content.CostRate(0).Valid() || content.CostRate(3).String() != "unknown" {
		t.Error("CostRate members are Day and Mile; the zero rate is not one [FR-098]")
	}
}

// Requirement: equipSpec FR-010, FR-098 - Cost.ValueInCopper is the amount
// times the coin's worth, in integers, over the whole range of amounts.
// Path: per case.
func TestCostValueInCopper(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		amount int
		coin   content.Coin
		want   int64
		path   string
	}{
		{"value-cp", 1, content.CoinCP, 1, "happy"}, {"value-sp", 1, content.CoinSP, 10, "happy"},
		{"value-ep", 1, content.CoinEP, 50, "happy"}, {"value-gp", 1, content.CoinGP, 100, "happy"},
		{"value-pp", 1, content.CoinPP, 1000, "happy"}, {"hundred-cp", 100, content.CoinCP, 100, "happy"},
		{"zero-gp", 0, content.CoinGP, 0, "unhappy (boundary)"},
		{"max-gp", 100000, content.CoinGP, 10_000_000, "unhappy (boundary)"},
		{"largest amount of platinum", content.MaxCostAmount, content.CoinPP, int64(content.MaxCostAmount) * 1000, "unhappy (boundary)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cost, err := content.NewCost(c.amount, c.coin, 0)
			if err != nil || cost.ValueInCopper() != c.want || cost.Amount() != c.amount || cost.Coin() != c.coin || !cost.Valid() {
				t.Errorf("NewCost(%d, %v) = %+v (%v) worth %d, want %d [FR-010, %s]", c.amount, c.coin, cost, err, cost.ValueInCopper(), c.want, c.path)
			}
			if _, per := cost.Per(); per {
				t.Error("a cost built with the zero rate is per item")
			}
		})
	}
}

// Requirement: equipSpec FR-098 - NewCost rejects a negative or too large
// amount, an invalid coin and an invalid rate, in that order, with a typed error.
// Path: per case.
func TestNewCostRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		amount int
		coin   content.Coin
		per    content.CostRate
		want   content.BlockProblem
	}{
		{"negative", -1, content.CoinGP, 0, content.BlockOutOfRange},
		{"above the bound", beyond(), content.CoinGP, 0, content.BlockOutOfRange},
		{"zero coin", 1, 0, 0, content.BlockInvalidMember},
		{"coin out of range", 1, content.Coin(6), 0, content.BlockInvalidMember},
		{"rate out of range", 1, content.CoinGP, content.CostRate(9), content.BlockInvalidMember},
		{"amount before coin", -1, 0, 0, content.BlockOutOfRange},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.NewCost(c.amount, c.coin, c.per)
			wantBlockError(t, err, "Cost", c.want)
			if got != (content.Cost{}) || got.Valid() {
				t.Errorf("rejected Cost = %+v, want the invalid zero Cost [FR-098]", got)
			}
			assertMustPanics(t, func() { content.MustCost(c.amount, c.coin, c.per) })
		})
	}
	if content.MustCost(0, content.CoinCP, content.CostPerMile) == (content.Cost{}) {
		t.Error("a valid cost equals the zero Cost")
	}
}

// Requirement: equipSpec FR-011, FR-012 - costs are equal iff amount, coin and
// rate are equal; CompareCost orders by Copper value and refuses different rates.
// Path: per case.
func TestCompareCost(t *testing.T) {
	t.Parallel()
	cost := func(amount int, coin content.Coin, per content.CostRate) content.Cost {
		return content.MustCost(amount, coin, per)
	}
	if cost(5, content.CoinSP, 0) == cost(50, content.CoinCP, 0) {
		t.Error("5 SP equals 50 CP under ==, want not equal [FR-011 equality]")
	}
	cases := []struct {
		name string
		a, b content.Cost
		want int
		err  bool
		path string
	}{
		{"compare-equal", cost(5, content.CoinSP, 0), cost(50, content.CoinCP, 0), 0, false, "unhappy (rule interaction)"},
		{"compare-less", cost(1, content.CoinCP, 0), cost(1, content.CoinSP, 0), -1, false, "happy"},
		{"compare-greater", cost(1, content.CoinGP, 0), cost(99, content.CoinCP, 0), 1, false, "happy"},
		{"compare-per-day", cost(2, content.CoinSP, content.CostPerDay), cost(1, content.CoinGP, content.CostPerDay), -1, false, "unhappy (boundary)"},
		{"compare-rates", cost(5, content.CoinSP, content.CostPerDay), cost(5, content.CoinSP, 0), 0, true, "unhappy (invalid input)"},
		{"compare-day-and-mile", cost(5, content.CoinSP, content.CostPerDay), cost(5, content.CoinSP, content.CostPerMile), 0, true, "unhappy (invalid input)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.CompareCost(c.a, c.b)
			if c.err {
				wantBlockError(t, err, "Cost", content.BlockConstraint)
			} else if err != nil {
				t.Errorf("CompareCost error = %v [FR-011, %s]", err, c.path)
			}
			if got != c.want {
				t.Errorf("CompareCost(%+v, %+v) = %d, want %d [FR-011, %s]", c.a, c.b, got, c.want, c.path)
			}
		})
	}
	if per, ok := cost(1, content.CoinGP, content.CostPerMile).Per(); !ok || per != content.CostPerMile {
		t.Error("Per of a per-Mile cost = false")
	}
}

// Requirement: equipSpec FR-013, FR-014 - a Price is a FixedCost or CostVaries,
// and a FixedCost holds only a valid Cost.
// Path: per case.
func TestPrice(t *testing.T) {
	t.Parallel()
	f, err := content.NewFixedCost(content.MustCost(3, content.CoinGP, 0))
	if err != nil || f.Cost().Amount() != 3 {
		t.Errorf("NewFixedCost = %+v, %v [happy]", f, err)
	}
	if _, err := content.NewFixedCost(content.Cost{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewFixedCost(zero Cost) error = %v, want ErrInvalidInput [unhappy (invalid input)]", err)
	}
	wantBlockError(t, func() error { _, err := content.NewFixedCost(content.Cost{}); return err }(), "FixedCost", content.BlockInvalidMember)
	assertMustPanics(t, func() { content.MustFixedCost(content.Cost{}) })
	for _, p := range []content.Price{f, content.CostVaries{}} {
		switch p.(type) { // exhaustive: the two variants
		case content.FixedCost, content.CostVaries:
		default:
			t.Errorf("unexpected price variant %T", p)
		}
	}
}

// Requirement: equipSpec FR-013, FR-015 (SRD 5.2.1, Adventuring Gear table,
// p. 95) - a Weight holds finite pounds of 0 or more and an optional
// non-blank note; Varies and "—" are their own variants.
// Path: per case.
func TestWeight(t *testing.T) {
	t.Parallel()
	w, err := content.NewWeight(0.25, "full")
	if err != nil || w.Pounds() != 0.25 || w.Note() != "full" {
		t.Errorf("NewWeight(0.25, full) = %+v, %v [happy]", w, err)
	}
	if z, err := content.NewWeight(0, ""); err != nil || z != (content.Weight{}) {
		t.Errorf("NewWeight(0, \"\") = %+v, %v, want the zero Weight [unhappy (boundary)]", z, err)
	}
	cases := []struct {
		name   string
		pounds float64
		note   string
		want   content.BlockProblem
	}{
		{"negative", -0.25, "", content.BlockOutOfRange},
		{"infinite", math.Inf(1), "", content.BlockOutOfRange},
		{"not a number", math.NaN(), "", content.BlockOutOfRange},
		{"blank note", 1, " ", content.BlockBlankText},
		{"control character in the note", 1, "a\x00", content.BlockInvalidCharacter},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.NewWeight(c.pounds, c.note)
			wantBlockError(t, err, "Weight", c.want)
			if got != (content.Weight{}) {
				t.Errorf("rejected Weight = %+v, want zero [unhappy (invalid input)]", got)
			}
			assertMustPanics(t, func() { content.MustWeight(c.pounds, c.note) })
		})
	}
	for _, s := range []content.WeightSpec{w, content.WeightVaries{}, content.NoWeight{}} {
		switch s.(type) { // exhaustive: the three variants
		case content.Weight, content.WeightVaries, content.NoWeight:
		default:
			t.Errorf("unexpected weight variant %T", s)
		}
	}
}

// Requirement: equipSpec FR-002 - EquipmentCategory has exactly twelve members.
// Path: happy.
func TestEquipmentCategory(t *testing.T) {
	t.Parallel()
	want := []string{
		"weapon", "armor", "tool", "adventuringGear", "mount", "tackHarnessOrDrawnVehicle",
		"airborneOrWaterborneVehicle", "lifestyleExpense", "foodDrinkOrLodging", "hireling",
		"spellcastingService", "spellScrollCost",
	}
	for i, name := range want {
		c := content.EquipmentCategory(i + 1)
		if c.String() != name || !c.Valid() {
			t.Errorf("EquipmentCategory(%d) = %q valid %v, want %q [FR-002]", i+1, c, c.Valid(), name)
		}
	}
	if content.EquipmentCategory(0).Valid() || content.EquipmentCategory(len(want)+1).Valid() || content.EquipmentCategory(0).String() != "unknown" {
		t.Error("the categories outside 1 to 12 are valid [FR-002]")
	}
}

// beyond returns 2^31, which no equipment integer may reach. Where int is 32
// bits wide the conversion wraps to a negative number, which each row that
// uses it rejects as out of range too, so the same rows run at both widths.
func beyond() int {
	v := int64(1) << 31
	return int(v)
}
