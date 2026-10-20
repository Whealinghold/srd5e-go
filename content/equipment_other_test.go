package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: equipSpec FR-046, FR-047, FR-091 - a Mount has a carrying
// capacity of at least 1 and a price; a tack row has a Weight or NoWeight.
// Path: per case.
func TestMountAndTack(t *testing.T) {
	t.Parallel()
	price := fixedCost(5, content.CoinGP)
	m, err := content.NewMount("mount_x", "Mount X", noSource, 480, price)
	if err != nil || m.CarryingCapacityPounds() != 480 || m.Price() != price || m.EquipmentCategory() != content.EquipmentMount || m.Key().Kind() != content.KindEquipment {
		t.Errorf("NewMount = %+v, %v [FR-046 happy]", m, err)
	}
	var _ content.EquipmentEntry = m
	for _, c := range []struct {
		name     string
		capacity int
		price    content.FixedCost
		want     content.BlockProblem
	}{{"capacity 0", 0, price, content.BlockOutOfRange}, {"capacity -1", -1, price, content.BlockOutOfRange}, {"above the bound", beyond(), price, content.BlockOutOfRange}, {"zero price", 1, content.FixedCost{}, content.BlockInvalidMember}} {
		_, err := content.NewMount("mount_x", "M", noSource, c.capacity, c.price)
		wantBlockError(t, err, "Mount", c.want)
	}
	if m, err := content.NewMount("mount_x", "M", noSource, 1, price); err != nil || m.CarryingCapacityPounds() != 1 {
		t.Errorf("capacity 1 = %+v, %v [FR-091 boundary]", m, err)
	}
	_, err = content.NewMount("mount_x", "", noSource, 1, price)
	wantBlockError(t, err, "Mount", content.BlockInvalidName)

	for _, w := range []content.WeightSpec{content.MustWeight(30, ""), content.NoWeight{}} {
		tk, err := content.NewTackHarnessOrDrawnVehicle("tack_x", "Tack X", noSource, w, price)
		if err != nil || tk.Weight() != w || tk.Price() != price || tk.EquipmentCategory() != content.EquipmentTackHarnessOrDrawnVehicle {
			t.Errorf("NewTackHarnessOrDrawnVehicle(%v) = %+v, %v [FR-047 happy]", w, tk, err)
		}
		var _ content.EquipmentEntry = tk
	}
	for _, w := range []content.WeightSpec{content.WeightVaries{}, nil} {
		_, err := content.NewTackHarnessOrDrawnVehicle("tack_x", "T", noSource, w, price)
		wantBlockError(t, err, "TackHarnessOrDrawnVehicle", content.BlockInvalidMember)
	}
	_, err = content.NewTackHarnessOrDrawnVehicle("tack_x", "T", noSource, content.NoWeight{}, content.FixedCost{})
	wantBlockError(t, err, "TackHarnessOrDrawnVehicle", content.BlockInvalidMember)
	_, err = content.NewTackHarnessOrDrawnVehicle("tack_x", " ", noSource, content.NoWeight{}, price)
	wantBlockError(t, err, "TackHarnessOrDrawnVehicle", content.BlockInvalidName)
}

// Requirement: equipSpec FR-048, FR-091 (SRD 5.2.1, Airborne and Waterborne
// Vehicles table, p. 101) - a vehicle's speed and cargo are half steps of at
// least 0.5; a column printed "—" is absent, never 0.
// Path: per case.
func TestVehicle(t *testing.T) {
	t.Parallel()
	price, w := fixedCost(10, content.CoinGP), content.WeightSpec(content.NoWeight{})
	mk := func(speed float64, crew, passengers int, cargo float64, ac, hp, threshold int, weight content.WeightSpec) (content.AirborneOrWaterborneVehicle, error) {
		return content.NewAirborneOrWaterborneVehicle("boat_x", "Boat X", noSource, speed, crew, passengers, cargo, ac, hp, threshold, weight, price)
	}
	v, err := mk(1.5, 2, 3, 0.5, 11, 50, 5, content.MustWeight(100, ""))
	if err != nil {
		t.Fatal(err)
	}
	pass, hasPass := v.Passengers()
	cargo, hasCargo := v.CargoTons()
	thr, hasThr := v.DamageThreshold()
	if v.SpeedMph() != 1.5 || v.Crew() != 2 || !hasPass || pass != 3 || !hasCargo || cargo != 0.5 || v.ArmorClass() != 11 || v.HitPoints() != 50 || !hasThr || thr != 5 ||
		v.Weight() != content.WeightSpec(content.MustWeight(100, "")) || v.Price() != price || v.EquipmentCategory() != content.EquipmentAirborneOrWaterborneVehicle {
		t.Errorf("vehicle accessors = %+v [FR-048]", v)
	}
	var _ content.EquipmentEntry = v
	bare, err := mk(0.5, 1, 0, 0, 1, 1, 0, w)
	if err != nil {
		t.Fatalf("a vehicle with every optional column printed \"—\": %v [FR-048 boundary]", err)
	}
	if _, ok := bare.Passengers(); ok {
		t.Error("an absent Passengers column reads as present")
	}
	if _, ok := bare.CargoTons(); ok {
		t.Error("an absent Cargo column reads as present")
	}
	if _, ok := bare.DamageThreshold(); ok {
		t.Error("an absent Damage Threshold column reads as present")
	}
	cases := []struct {
		name string
		make func() error
		want content.BlockProblem
	}{
		{"speed off the half step", func() error { _, err := mk(1.25, 1, 0, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"speed 0.25", func() error { _, err := mk(0.25, 1, 0, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"speed 0", func() error { _, err := mk(0, 1, 0, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"speed above the bound", func() error { _, err := mk(1<<31+1, 1, 0, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"cargo off the half step", func() error { _, err := mk(1, 1, 0, 0.75, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"cargo 0.25", func() error { _, err := mk(1, 1, 0, 0.25, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"crew 0", func() error { _, err := mk(1, 0, 0, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"passengers -1", func() error { _, err := mk(1, 1, -1, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"armor class 0", func() error { _, err := mk(1, 1, 0, 0, 0, 1, 0, w); return err }, content.BlockOutOfRange},
		{"hit points 0", func() error { _, err := mk(1, 1, 0, 0, 1, 0, 0, w); return err }, content.BlockOutOfRange},
		{"threshold -1", func() error { _, err := mk(1, 1, 0, 0, 1, 1, -1, w); return err }, content.BlockOutOfRange},
		{"crew above the bound", func() error { _, err := mk(1, beyond(), 0, 0, 1, 1, 0, w); return err }, content.BlockOutOfRange},
		{"Varies weight", func() error { _, err := mk(1, 1, 0, 0, 1, 1, 0, content.WeightVaries{}); return err }, content.BlockInvalidMember},
		{"zero price", func() error {
			_, err := content.NewAirborneOrWaterborneVehicle("boat_x", "B", noSource, 1, 1, 0, 0, 1, 1, 0, w, content.FixedCost{})
			return err
		}, content.BlockInvalidMember},
		{"bad name", func() error {
			_, err := content.NewAirborneOrWaterborneVehicle("boat_x", "", noSource, 1, 1, 0, 0, 1, 1, 0, w, price)
			return err
		}, content.BlockInvalidName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			wantBlockError(t, c.make(), "AirborneOrWaterborneVehicle", c.want)
		})
	}
}

// Requirement: equipSpec FR-054 to FR-056, FR-116 - lifestyle expenses, food
// rows and hirelings hold a price; a lifestyle expense holds a text.
// Path: per case.
func TestTablesOfPrices(t *testing.T) {
	t.Parallel()
	price := content.MustFixedCost(content.MustCost(2, content.CoinSP, content.CostPerDay))
	l, err := content.NewLifestyleExpense("squalid_x", "Squalid X", noSource, price, "Text.")
	if err != nil || l.Price() != price || l.Text() != "Text." || l.EquipmentCategory() != content.EquipmentLifestyleExpense {
		t.Errorf("NewLifestyleExpense = %+v, %v [FR-054]", l, err)
	}
	var _ content.EquipmentEntry = l
	for _, c := range []struct {
		text  string
		price content.FixedCost
		want  content.BlockProblem
	}{{"", price, content.BlockBlankText}, {" ", price, content.BlockBlankText}, {"a\x00", price, content.BlockInvalidCharacter}, {"t", content.FixedCost{}, content.BlockInvalidMember}} {
		_, err := content.NewLifestyleExpense("squalid_x", "S", noSource, c.price, c.text)
		wantBlockError(t, err, "LifestyleExpense", c.want)
	}
	_, err = content.NewLifestyleExpense("squalid_x", "", noSource, price, "t")
	wantBlockError(t, err, "LifestyleExpense", content.BlockInvalidName)

	f, err := content.NewFoodDrinkOrLodging("ale_x", "Ale X", noSource, price)
	if err != nil || f.Price() != price || f.EquipmentCategory() != content.EquipmentFoodDrinkOrLodging {
		t.Errorf("NewFoodDrinkOrLodging = %+v, %v [FR-055]", f, err)
	}
	var _ content.EquipmentEntry = f
	if got, err := content.NewFoodDrinkOrLodging("ale_x", "A", noSource, content.FixedCost{}); err == nil || got != (content.FoodDrinkOrLodging{}) {
		t.Errorf("a food row with a zero price = %+v, %v, want the zero value and an error [FR-055]", got, err)
	}
	_, err = content.NewFoodDrinkOrLodging("Bad Id", "A", noSource, price)
	if err == nil {
		t.Error("a food row with an invalid id was accepted")
	}
	h, err := content.NewHireling("hire_x", "Hire X", noSource, price)
	if err != nil || h.Price() != price || h.EquipmentCategory() != content.EquipmentHireling {
		t.Errorf("NewHireling = %+v, %v [FR-056]", h, err)
	}
	var _ content.EquipmentEntry = h
	if got, err := content.NewHireling("hire_x", "H", noSource, content.FixedCost{}); err == nil || got != (content.Hireling{}) {
		t.Errorf("a hireling with a zero price = %+v, %v, want the zero value and an error [FR-056]", got, err)
	}
	_, err = content.NewHireling("hire_x", " ", noSource, price)
	wantBlockError(t, err, "Hireling", content.BlockInvalidName)
}

// Requirement: equipSpec FR-057, FR-064, FR-118 - spell levels are plain ints
// 0 to 9 on a service and a scroll cost; Availability and SettlementSize are
// closed enums.
// Path: per case.
func TestServicesAndScrollCosts(t *testing.T) {
	t.Parallel()
	for i, n := range []string{"Village, town, or city", "Town or city only", "City only"} {
		a := content.Availability(i + 1)
		if a.String() != n || !a.Valid() || a.MinimumSize() != content.SettlementSize(i+1) {
			t.Errorf("Availability(%d) = %q, minimum %v, want %q and %d [FR-057]", i+1, a, a.MinimumSize(), n, i+1)
		}
	}
	if content.Availability(0).Valid() || content.Availability(4).String() != "unknown" || content.Availability(0).MinimumSize() != 0 || content.Availability(4).MinimumSize().Valid() {
		t.Error("an invalid Availability is a member [FR-057]")
	}
	for i, n := range []string{"village", "town", "city"} {
		if s := content.SettlementSize(i + 1); s.String() != n || !s.Valid() {
			t.Errorf("SettlementSize(%d) = %q, want %q", i+1, s, n)
		}
	}
	if content.SettlementSize(0).Valid() || content.SettlementSize(4).String() != "unknown" {
		t.Error("an invalid SettlementSize is a member")
	}
	price := fixedCost(50, content.CoinGP)
	// Levels 0 and 9 are accepted, -1 and 10 rejected, on each of the three types.
	for _, level := range []int{0, 9} {
		if _, err := content.NewSpellcastingService("svc_x", "S", noSource, level, level, content.AvailabilityCityOnly, price); err != nil {
			t.Errorf("service level %d: %v [FR-118 boundary]", level, err)
		}
		if _, err := content.NewSpellScrollCost("cost_x", "C", noSource, level, 1, price); err != nil {
			t.Errorf("scroll cost level %d: %v [FR-118 boundary]", level, err)
		}
	}
	for _, level := range []int{-1, 10} {
		_, err := content.NewSpellcastingService("svc_x", "S", noSource, level, 9, content.AvailabilityCityOnly, price)
		wantBlockError(t, err, "SpellcastingService", content.BlockOutOfRange)
		_, err = content.NewSpellcastingService("svc_x", "S", noSource, 0, level, content.AvailabilityCityOnly, price)
		wantBlockError(t, err, "SpellcastingService", content.BlockOutOfRange)
		_, err = content.NewSpellScrollCost("cost_x", "C", noSource, level, 1, price)
		wantBlockError(t, err, "SpellScrollCost", content.BlockOutOfRange)
	}
	s, err := content.NewSpellcastingService("svc_x", "Svc X", noSource, 4, 5, content.AvailabilityTownOrCityOnly, price)
	if err != nil || s.SpellLevelMin() != 4 || s.SpellLevelMax() != 5 || s.Availability() != content.AvailabilityTownOrCityOnly || s.Price() != price || s.EquipmentCategory() != content.EquipmentSpellcastingService {
		t.Errorf("NewSpellcastingService = %+v, %v [FR-057]", s, err)
	}
	var _ content.EquipmentEntry = s
	_, err = content.NewSpellcastingService("svc_x", "S", noSource, 5, 4, content.AvailabilityCityOnly, price)
	wantBlockError(t, err, "SpellcastingService", content.BlockConstraint)
	_, err = content.NewSpellcastingService("svc_x", "S", noSource, 0, 1, 0, price)
	wantBlockError(t, err, "SpellcastingService", content.BlockInvalidMember)
	_, err = content.NewSpellcastingService("svc_x", "S", noSource, 0, 1, content.AvailabilityCityOnly, content.FixedCost{})
	wantBlockError(t, err, "SpellcastingService", content.BlockInvalidMember)
	_, err = content.NewSpellcastingService("svc_x", "", noSource, 0, 1, content.AvailabilityCityOnly, price)
	wantBlockError(t, err, "SpellcastingService", content.BlockInvalidName)

	c, err := content.NewSpellScrollCost("cost_x", "Cost X", noSource, 3, 2, price)
	if err != nil || c.SpellLevel() != 3 || c.Days() != 2 || c.Price() != price || c.EquipmentCategory() != content.EquipmentSpellScrollCost {
		t.Errorf("NewSpellScrollCost = %+v, %v [FR-064]", c, err)
	}
	var _ content.EquipmentEntry = c
	for _, days := range []int{0, -1, beyond()} {
		_, err := content.NewSpellScrollCost("cost_x", "C", noSource, 3, days, price)
		wantBlockError(t, err, "SpellScrollCost", content.BlockOutOfRange)
	}
	_, err = content.NewSpellScrollCost("cost_x", "C", noSource, 3, 1, content.FixedCost{})
	wantBlockError(t, err, "SpellScrollCost", content.BlockInvalidMember)
	_, err = content.NewSpellScrollCost("cost_x", " ", noSource, 3, 1, price)
	wantBlockError(t, err, "SpellScrollCost", content.BlockInvalidName)
}
