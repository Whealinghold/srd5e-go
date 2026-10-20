package content

import "math"

// Mount is a row of the Mounts and Other Animals table (SRD 5.2.1, p. 100).
// It is an [EquipmentEntry]. Immutable; the zero value is invalid.
// Comparable.
type Mount struct {
	equipmentHeader
	capacity int
	price    FixedCost
}

var _ EquipmentEntry = Mount{}

// NewMount returns the mount id called name, with a carrying capacity of at
// least 1 pound.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockOutOfRange for a capacity below 1 or above 2^31 - 1;
// BlockInvalidMember for an invalid price. It never panics.
func NewMount(id ItemID, name string, source SourceCitation, carryingCapacityPounds int, price FixedCost) (Mount, error) {
	const block = "Mount"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return Mount{}, err
	}
	switch {
	case carryingCapacityPounds < 1 || carryingCapacityPounds > equipmentMaxInt:
		return Mount{}, blockError(block, BlockOutOfRange)
	case !price.cost.Valid():
		return Mount{}, blockError(block, BlockInvalidMember)
	}
	return Mount{equipmentHeader: h, capacity: carryingCapacityPounds, price: price}, nil
}

// EquipmentCategory returns [EquipmentMount].
func (Mount) EquipmentCategory() EquipmentCategory { return EquipmentMount }

// CarryingCapacityPounds returns the carrying capacity in pounds.
func (m Mount) CarryingCapacityPounds() int { return m.capacity }

// Price returns the price.
func (m Mount) Price() FixedCost { return m.price }

// TackHarnessOrDrawnVehicle is a row of the Tack, Harness, and Drawn Vehicles
// table (SRD 5.2.1, p. 100). It is an [EquipmentEntry]. Immutable; the zero
// value is invalid. Comparable.
type TackHarnessOrDrawnVehicle struct {
	equipmentHeader
	weight WeightSpec
	price  FixedCost
}

var _ EquipmentEntry = TackHarnessOrDrawnVehicle{}

// NewTackHarnessOrDrawnVehicle returns the row id called name. The weight is a
// [Weight] or [NoWeight].
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid weight or price. It never panics.
func NewTackHarnessOrDrawnVehicle(id ItemID, name string, source SourceCitation, weight WeightSpec, price FixedCost) (TackHarnessOrDrawnVehicle, error) {
	const block = "TackHarnessOrDrawnVehicle"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return TackHarnessOrDrawnVehicle{}, err
	}
	if !validFixedWeight(weight) || !price.cost.Valid() {
		return TackHarnessOrDrawnVehicle{}, blockError(block, BlockInvalidMember)
	}
	return TackHarnessOrDrawnVehicle{equipmentHeader: h, weight: weight, price: price}, nil
}

// EquipmentCategory returns [EquipmentTackHarnessOrDrawnVehicle].
func (TackHarnessOrDrawnVehicle) EquipmentCategory() EquipmentCategory {
	return EquipmentTackHarnessOrDrawnVehicle
}

// Weight returns the weight: a [Weight] or [NoWeight].
func (t TackHarnessOrDrawnVehicle) Weight() WeightSpec { return t.weight }

// Price returns the price.
func (t TackHarnessOrDrawnVehicle) Price() FixedCost { return t.price }

// isHalfStep reports whether v is a multiple of 0.5 that is at least 0.5 (the
// speed and cargo of a vehicle).
func isHalfStep(v float64) bool { return v >= 0.5 && v <= equipmentMaxInt && v*2 == math.Trunc(v*2) }

// AirborneOrWaterborneVehicle is a row of the Airborne and Waterborne Vehicles
// table (SRD 5.2.1, p. 101). A column printed "—" is absent. It is an
// [EquipmentEntry]. Immutable; the zero value is invalid. Comparable.
type AirborneOrWaterborneVehicle struct {
	equipmentHeader
	speed           float64
	crew            int
	passengers      int     // 0 when absent
	cargoTons       float64 // 0 when absent
	armorClass      int
	hitPoints       int
	damageThreshold int // 0 when absent
	weight          WeightSpec
	price           FixedCost
}

var _ EquipmentEntry = AirborneOrWaterborneVehicle{}

// NewAirborneOrWaterborneVehicle returns the vehicle id called name. The speed
// and the cargo are multiples of 0.5 of at least 0.5; the crew, Armor Class and
// Hit Points are at least 1; passengers, cargoTons and damageThreshold are 0
// when the table prints "—", and otherwise at least 1, 0.5 and 1. The weight is
// a [Weight] or [NoWeight].
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockOutOfRange for a number outside its range or off its half step;
// BlockInvalidMember for an invalid weight or price. It never panics.
func NewAirborneOrWaterborneVehicle(id ItemID, name string, source SourceCitation, speedMph float64, crew, passengers int, cargoTons float64, armorClass, hitPoints, damageThreshold int, weight WeightSpec, price FixedCost) (AirborneOrWaterborneVehicle, error) {
	const block = "AirborneOrWaterborneVehicle"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return AirborneOrWaterborneVehicle{}, err
	}
	inRange := func(v, lo int) bool { return v >= lo && v <= equipmentMaxInt }
	switch {
	case !isHalfStep(speedMph), cargoTons != 0 && !isHalfStep(cargoTons),
		!inRange(crew, 1), passengers != 0 && !inRange(passengers, 1), !inRange(armorClass, 1),
		!inRange(hitPoints, 1), damageThreshold != 0 && !inRange(damageThreshold, 1):
		return AirborneOrWaterborneVehicle{}, blockError(block, BlockOutOfRange)
	case !validFixedWeight(weight), !price.cost.Valid():
		return AirborneOrWaterborneVehicle{}, blockError(block, BlockInvalidMember)
	}
	return AirborneOrWaterborneVehicle{
		equipmentHeader: h, speed: speedMph, crew: crew, passengers: passengers, cargoTons: cargoTons,
		armorClass: armorClass, hitPoints: hitPoints, damageThreshold: damageThreshold, weight: weight, price: price,
	}, nil
}

// EquipmentCategory returns [EquipmentAirborneOrWaterborneVehicle].
func (AirborneOrWaterborneVehicle) EquipmentCategory() EquipmentCategory {
	return EquipmentAirborneOrWaterborneVehicle
}

// SpeedMph returns the speed in miles per hour.
func (v AirborneOrWaterborneVehicle) SpeedMph() float64 { return v.speed }

// Crew returns the crew.
func (v AirborneOrWaterborneVehicle) Crew() int { return v.crew }

// Passengers returns the number of passengers and true, or 0 and false when
// the table prints "—".
func (v AirborneOrWaterborneVehicle) Passengers() (int, bool) { return v.passengers, v.passengers != 0 }

// CargoTons returns the cargo in tons and true, or 0 and false when the table
// prints "—".
func (v AirborneOrWaterborneVehicle) CargoTons() (float64, bool) {
	return v.cargoTons, v.cargoTons != 0
}

// ArmorClass returns the Armor Class.
func (v AirborneOrWaterborneVehicle) ArmorClass() int { return v.armorClass }

// HitPoints returns the Hit Points.
func (v AirborneOrWaterborneVehicle) HitPoints() int { return v.hitPoints }

// DamageThreshold returns the Damage Threshold and true, or 0 and false when
// the table prints "—".
func (v AirborneOrWaterborneVehicle) DamageThreshold() (int, bool) {
	return v.damageThreshold, v.damageThreshold != 0
}

// Weight returns the weight: a [Weight] or [NoWeight].
func (v AirborneOrWaterborneVehicle) Weight() WeightSpec { return v.weight }

// Price returns the price.
func (v AirborneOrWaterborneVehicle) Price() FixedCost { return v.price }

// LifestyleExpense is a row of the Lifestyle Expenses table with its verbatim
// description (SRD 5.2.1, p. 101). It is an [EquipmentEntry]. Immutable; the
// zero value is invalid. Comparable.
type LifestyleExpense struct {
	equipmentHeader
	price FixedCost
	text  string
}

var _ EquipmentEntry = LifestyleExpense{}

// NewLifestyleExpense returns the expense id called name with the verbatim
// text.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid price; BlockBlankText or
// BlockInvalidCharacter for a text that breaks the text rule. It never panics.
func NewLifestyleExpense(id ItemID, name string, source SourceCitation, price FixedCost, text string) (LifestyleExpense, error) {
	const block = "LifestyleExpense"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return LifestyleExpense{}, err
	}
	if !price.cost.Valid() {
		return LifestyleExpense{}, blockError(block, BlockInvalidMember)
	}
	if err := checkText(block, text); err != nil {
		return LifestyleExpense{}, err
	}
	return LifestyleExpense{equipmentHeader: h, price: price, text: text}, nil
}

// EquipmentCategory returns [EquipmentLifestyleExpense].
func (LifestyleExpense) EquipmentCategory() EquipmentCategory { return EquipmentLifestyleExpense }

// Price returns the price.
func (e LifestyleExpense) Price() FixedCost { return e.price }

// Text returns the verbatim description.
func (e LifestyleExpense) Text() string { return e.text }

// FoodDrinkOrLodging is a row of the Food, Drink, and Lodging table (SRD 5.2.1,
// pp. 101-102). It is an [EquipmentEntry]. Immutable; the zero value is
// invalid. Comparable.
type FoodDrinkOrLodging struct {
	equipmentHeader
	price FixedCost
}

var _ EquipmentEntry = FoodDrinkOrLodging{}

// NewFoodDrinkOrLodging returns the row id called name.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid price. It never panics.
func NewFoodDrinkOrLodging(id ItemID, name string, source SourceCitation, price FixedCost) (FoodDrinkOrLodging, error) {
	h, err := newPricedHeader("FoodDrinkOrLodging", id, name, source, price)
	if err != nil {
		return FoodDrinkOrLodging{}, err
	}
	return FoodDrinkOrLodging{equipmentHeader: h, price: price}, nil
}

// EquipmentCategory returns [EquipmentFoodDrinkOrLodging].
func (FoodDrinkOrLodging) EquipmentCategory() EquipmentCategory { return EquipmentFoodDrinkOrLodging }

// Price returns the price.
func (f FoodDrinkOrLodging) Price() FixedCost { return f.price }

// Hireling is a row of the Hirelings table (SRD 5.2.1, p. 102). It is an
// [EquipmentEntry]. Immutable; the zero value is invalid. Comparable.
type Hireling struct {
	equipmentHeader
	price FixedCost
}

var _ EquipmentEntry = Hireling{}

// NewHireling returns the hireling id called name.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockInvalidMember for an invalid price. It never panics.
func NewHireling(id ItemID, name string, source SourceCitation, price FixedCost) (Hireling, error) {
	h, err := newPricedHeader("Hireling", id, name, source, price)
	if err != nil {
		return Hireling{}, err
	}
	return Hireling{equipmentHeader: h, price: price}, nil
}

// EquipmentCategory returns [EquipmentHireling].
func (Hireling) EquipmentCategory() EquipmentCategory { return EquipmentHireling }

// Price returns the price.
func (h Hireling) Price() FixedCost { return h.price }

// newPricedHeader validates an identity and a price, returning the zero header
// on an error.
func newPricedHeader(block string, id ItemID, name string, source SourceCitation, price FixedCost) (equipmentHeader, error) {
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return equipmentHeader{}, err
	}
	if !price.cost.Valid() {
		return equipmentHeader{}, blockError(block, BlockInvalidMember)
	}
	return h, nil
}

// SettlementSize is the size of a settlement: a village, a town or a city (SRD
// 5.2.1, "Spellcasting Services", p. 102), in ascending order. The zero value
// is invalid. Comparable and safe for concurrent use.
type SettlementSize int

// The settlement sizes.
const (
	SettlementVillage SettlementSize = iota + 1
	SettlementTown
	SettlementCity
)

func settlementSizeNames() []string { return []string{"village", "town", "city"} }

// Valid reports whether s is a settlement size.
func (s SettlementSize) Valid() bool { return enumValid(settlementSizeNames(), int(s)) }

// String returns the lower case name of s, or "unknown".
func (s SettlementSize) String() string { return enumString(settlementSizeNames(), int(s)) }

// Availability is where a spellcasting service can be found, as printed (SRD
// 5.2.1, "Spellcasting Services", p. 102). The zero value is invalid.
// Comparable and safe for concurrent use.
type Availability int

// The availabilities.
const (
	AvailabilityVillageTownOrCity Availability = iota + 1
	AvailabilityTownOrCityOnly
	AvailabilityCityOnly
)

func availabilityNames() []string {
	return []string{"Village, town, or city", "Town or city only", "City only"}
}

// Valid reports whether a is an availability.
func (a Availability) Valid() bool { return enumValid(availabilityNames(), int(a)) }

// String returns the printed text of a, or "unknown".
func (a Availability) String() string { return enumString(availabilityNames(), int(a)) }

// MinimumSize returns the smallest settlement where the service is available:
// a village, a town or a city. It returns the zero (invalid) size for an
// invalid availability.
func (a Availability) MinimumSize() SettlementSize {
	switch a {
	case AvailabilityVillageTownOrCity:
		return SettlementVillage
	case AvailabilityTownOrCityOnly:
		return SettlementTown
	case AvailabilityCityOnly:
		return SettlementCity
	}
	return 0
}

func parseAvailability(s string) (Availability, error) {
	v, err := parseEnum("Availability", availabilityNames(), s)
	return Availability(v), err
}

// SpellcastingService is a row of the Spellcasting Services table: a range of
// spell levels, where it is available and its price (SRD 5.2.1, p. 102). Spell
// levels are plain ints 0 to 9 here (content-equipment FR-118). It is an
// [EquipmentEntry]. Immutable; the zero value is invalid. Comparable.
type SpellcastingService struct {
	equipmentHeader
	min, max     int
	availability Availability
	price        FixedCost
}

var _ EquipmentEntry = SpellcastingService{}

// NewSpellcastingService returns the service id called name for spell levels
// spellLevelMin to spellLevelMax.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockOutOfRange for a level outside 0 to 9; BlockConstraint for a minimum
// above the maximum; BlockInvalidMember for an invalid availability or price.
// It never panics.
func NewSpellcastingService(id ItemID, name string, source SourceCitation, spellLevelMin, spellLevelMax int, availability Availability, price FixedCost) (SpellcastingService, error) {
	const block = "SpellcastingService"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return SpellcastingService{}, err
	}
	switch {
	case !validSpellLevel(spellLevelMin) || !validSpellLevel(spellLevelMax):
		return SpellcastingService{}, blockError(block, BlockOutOfRange)
	case spellLevelMin > spellLevelMax:
		return SpellcastingService{}, blockError(block, BlockConstraint)
	case !availability.Valid(), !price.cost.Valid():
		return SpellcastingService{}, blockError(block, BlockInvalidMember)
	}
	return SpellcastingService{equipmentHeader: h, min: spellLevelMin, max: spellLevelMax, availability: availability, price: price}, nil
}

// EquipmentCategory returns [EquipmentSpellcastingService].
func (SpellcastingService) EquipmentCategory() EquipmentCategory { return EquipmentSpellcastingService }

// SpellLevelMin returns the lowest spell level the row covers, 0 for a cantrip.
func (s SpellcastingService) SpellLevelMin() int { return s.min }

// SpellLevelMax returns the highest spell level the row covers.
func (s SpellcastingService) SpellLevelMax() int { return s.max }

// Availability returns where the service is available.
func (s SpellcastingService) Availability() Availability { return s.availability }

// Price returns the price.
func (s SpellcastingService) Price() FixedCost { return s.price }

// SpellScrollCost is a row of the Spell Scroll Costs table: the days and the
// cost to scribe a scroll of one spell level (SRD 5.2.1, p. 103). The spell
// level is a plain int 0 to 9 (content-equipment FR-118). It is an
// [EquipmentEntry]. Immutable; the zero value is invalid. Comparable.
type SpellScrollCost struct {
	equipmentHeader
	spellLevel int
	days       int
	price      FixedCost
}

var _ EquipmentEntry = SpellScrollCost{}

// NewSpellScrollCost returns the row id called name for scrolls of spellLevel
// 0 to 9, taking at least 1 day.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError]: BlockInvalidName;
// BlockOutOfRange for a level outside 0 to 9 or days below 1 or above
// 2^31 - 1; BlockInvalidMember for an invalid price. It never panics.
func NewSpellScrollCost(id ItemID, name string, source SourceCitation, spellLevel, days int, price FixedCost) (SpellScrollCost, error) {
	const block = "SpellScrollCost"
	h, err := newEquipmentHeader(block, id, name, source)
	if err != nil {
		return SpellScrollCost{}, err
	}
	switch {
	case !validSpellLevel(spellLevel), days < 1 || days > equipmentMaxInt:
		return SpellScrollCost{}, blockError(block, BlockOutOfRange)
	case !price.cost.Valid():
		return SpellScrollCost{}, blockError(block, BlockInvalidMember)
	}
	return SpellScrollCost{equipmentHeader: h, spellLevel: spellLevel, days: days, price: price}, nil
}

// EquipmentCategory returns [EquipmentSpellScrollCost].
func (SpellScrollCost) EquipmentCategory() EquipmentCategory { return EquipmentSpellScrollCost }

// SpellLevel returns the spell level, 0 for a cantrip.
func (s SpellScrollCost) SpellLevel() int { return s.spellLevel }

// Days returns the days to scribe a scroll.
func (s SpellScrollCost) Days() int { return s.days }

// Price returns the price.
func (s SpellScrollCost) Price() FixedCost { return s.price }
