package character

// The units of SpaceLoad: whole ounces and cubic inches, so no capacity check
// compares floating-point numbers.
const (
	OuncesPerPound          = 16
	CubicInchesPerCubicFoot = 1728
)

// SpaceLoad is what is inside an extradimensional space, in ounces and cubic
// inches. Comparable; the zero value is an empty load.
type SpaceLoad struct{ ounces, cubicInches int }

// NewSpaceLoad returns the load. It returns an error matching
// srderr.ErrInvalidInput for a negative amount.
func NewSpaceLoad(ounces, cubicInches int) (SpaceLoad, error) {
	if ounces < 0 || cubicInches < 0 {
		return SpaceLoad{}, invalidInput("space load: negative amount")
	}
	return SpaceLoad{ounces, cubicInches}, nil
}

// SpaceLoadOf returns the load of pounds and cubic feet. It returns an error
// matching srderr.ErrInvalidInput for a negative amount.
func SpaceLoadOf(pounds, cubicFeet int) (SpaceLoad, error) {
	return NewSpaceLoad(pounds*OuncesPerPound, cubicFeet*CubicInchesPerCubicFoot)
}

// Ounces returns the weight in ounces.
func (l SpaceLoad) Ounces() int { return l.ounces }

// CubicInches returns the volume in cubic inches.
func (l SpaceLoad) CubicInches() int { return l.cubicInches }

// SpacePouch names a pouch of a space. The zero value is invalid.
type SpacePouch int

// The pouches: the one space of a Bag of Holding or Portable Hole, and the side
// and central pouches of a Handy Haversack.
const (
	PouchMain SpacePouch = iota + 1
	PouchSide
	PouchCentral
)

// ContainerWeight is the closed set of what an extradimensional item weighs.
// Variants: FixedWeight, WeighsNextToNothing.
type ContainerWeight interface{ isContainerWeight() }

// FixedWeight is a weight regardless of the contents.
type FixedWeight struct{ ounces int }

// NewFixedWeight returns a fixed weight in ounces. It returns an error matching
// srderr.ErrInvalidInput for a negative amount.
func NewFixedWeight(ounces int) (FixedWeight, error) {
	if ounces < 0 {
		return FixedWeight{}, invalidInput("fixed weight: negative amount")
	}
	return FixedWeight{ounces}, nil
}

// Ounces returns the weight in ounces.
func (w FixedWeight) Ounces() int { return w.ounces }

// WeighsNextToNothing: the Portable Hole states no number.
type WeighsNextToNothing struct{}

func (FixedWeight) isContainerWeight()         {}
func (WeighsNextToNothing) isContainerWeight() {}

// GateAreaKind is the shape of the area an Astral Gate affects. The zero value is invalid.
type GateAreaKind int

// The shapes.
const (
	GateSphere GateAreaKind = iota + 1
	GateCreaturesWithinNotBehindTotalCover
)

// GateArea is who an Astral Gate moves: a Sphere centered on the gate, or each
// creature within a distance and not behind Total Cover. Comparable; the zero
// value is invalid.
type GateArea struct {
	kind GateAreaKind
	feet int
}

// NewGateArea returns the area. It returns an error matching srderr.ErrInvalidInput
// for an invalid kind or non-positive distance.
func NewGateArea(kind GateAreaKind, feet int) (GateArea, error) {
	if kind < GateSphere || kind > GateCreaturesWithinNotBehindTotalCover || feet < 1 {
		return GateArea{}, invalidInput("gate area: invalid kind or distance")
	}
	return GateArea{kind, feet}, nil
}

// Kind returns the shape.
func (a GateArea) Kind() GateAreaKind { return a.kind }

// Feet returns the radius of the Sphere, or the distance.
func (a GateArea) Feet() int { return a.feet }

// ExtradimensionalSpace is the closed set of items with an extradimensional space.
// Variants: BagOfHoldingSpace, HandyHaversackSpace, PortableHoleSpace, SimilarSpace.
type ExtradimensionalSpace interface {
	NestedObject
	// Weight returns what the item weighs.
	Weight() ContainerWeight
	// Capacity returns the limit of a pouch and true, or false when the pouch has none.
	Capacity(pouch SpacePouch) (SpaceLoad, bool)
	// AirMinutes returns the minutes of air in a pouch, or 0 for a pouch the item lacks.
	AirMinutes(pouch SpacePouch) int
	// GateArea returns the area when this item is the one placed.
	GateArea() GateArea
	isExtradimensionalSpace()
}

// NestedObject is the closed set of what can be put inside an extradimensional
// space. Variants: OrdinaryObject, BagOfHoldingSpace, HandyHaversackSpace,
// PortableHoleSpace, SimilarSpace.
type NestedObject interface{ isNestedObject() }

// OrdinaryObject is anything that is not an extradimensional item.
type OrdinaryObject struct{}

func fiveOunces() FixedWeight { return FixedWeight{5 * OuncesPerPound} }

// bigPouch is 500 pounds and 64 cubic feet; sidePouch 200 pounds and 25 cubic feet.
func bigPouch() SpaceLoad  { return SpaceLoad{500 * OuncesPerPound, 64 * CubicInchesPerCubicFoot} }
func sidePouch() SpaceLoad { return SpaceLoad{200 * OuncesPerPound, 25 * CubicInchesPerCubicFoot} }

// sphereTen is a Sphere of radius 10 feet; withinTen each creature within 10 feet not behind Total Cover.
func sphereTen() GateArea { return GateArea{GateSphere, 10} }
func withinTen() GateArea { return GateArea{GateCreaturesWithinNotBehindTotalCover, 10} }

// BagOfHoldingSpace is a Bag of Holding (SRD 5.2.1, p. 212): 500 pounds and 64
// cubic feet, 10 minutes of air, 5 pounds. The zero value is that bag.
type BagOfHoldingSpace struct{}

// HandyHaversackSpace is a Handy Haversack (p. 224): two side pouches of 200
// pounds and 25 cubic feet, a central pouch of 500 pounds and 64 cubic feet, 10
// minutes of air in each, 5 pounds. The zero value is that haversack.
type HandyHaversackSpace struct{}

// PortableHoleSpace is a Portable Hole (pp. 234-235): no stated limit, 60 minutes
// of air when folded, weighs next to nothing. The zero value is that hole.
type PortableHoleSpace struct{}

// HoleEscape are the facts of climbing out of a folded Portable Hole: the depth,
// and an action for a Strength (Athletics) check against a DC after which the
// creature appears within some feet of the hole. The host makes the check.
type HoleEscape struct{ DepthFeet, DC, AppearsWithinFeet int }

// Escape returns depth 10 feet, DC 10, within 5 feet.
func (PortableHoleSpace) Escape() HoleEscape {
	return HoleEscape{DepthFeet: 10, DC: 10, AppearsWithinFeet: 5}
}

// SimilarSpace is an item "similar" to the other three, carrying its own data.
type SimilarSpace struct {
	weight   ContainerWeight
	air      int
	area     GateArea
	capacity SpaceLoad
	limited  bool
}

// NewSimilarSpace returns a similar item: its weight, minutes of air and gate
// area, and an optional capacity limit (nil for none). It returns an error
// matching srderr.ErrInvalidInput for a nil weight, negative air or an invalid area.
func NewSimilarSpace(weight ContainerWeight, airMinutes int, area GateArea, capacity *SpaceLoad) (SimilarSpace, error) {
	if weight == nil || airMinutes < 0 || area.kind == 0 {
		return SimilarSpace{}, invalidInput("similar space: nil weight, negative air or invalid area")
	}
	s := SimilarSpace{weight: weight, air: airMinutes, area: area}
	if capacity != nil {
		s.capacity, s.limited = *capacity, true
	}
	return s, nil
}

// Weight returns 5 pounds.
func (BagOfHoldingSpace) Weight() ContainerWeight { return fiveOunces() }

// Weight returns 5 pounds.
func (HandyHaversackSpace) Weight() ContainerWeight { return fiveOunces() }

// Weight returns WeighsNextToNothing.
func (PortableHoleSpace) Weight() ContainerWeight { return WeighsNextToNothing{} }

// Weight returns the item's weight.
func (s SimilarSpace) Weight() ContainerWeight { return s.weight }

// Capacity returns 500 pounds and 64 cubic feet for PouchMain.
func (BagOfHoldingSpace) Capacity(p SpacePouch) (SpaceLoad, bool) { return bigPouch(), p == PouchMain }

// Capacity returns the side or central pouch limit.
func (HandyHaversackSpace) Capacity(p SpacePouch) (SpaceLoad, bool) {
	switch p {
	case PouchSide:
		return sidePouch(), true
	case PouchCentral:
		return bigPouch(), true
	case PouchMain:
	}
	return SpaceLoad{}, false
}

// Capacity reports no limit.
func (PortableHoleSpace) Capacity(SpacePouch) (SpaceLoad, bool) { return SpaceLoad{}, false }

// Capacity returns the item's limit, if it has one.
func (s SimilarSpace) Capacity(SpacePouch) (SpaceLoad, bool) { return s.capacity, s.limited }

// AirMinutes returns 10 for PouchMain.
func (BagOfHoldingSpace) AirMinutes(p SpacePouch) int { return airFor(p == PouchMain, 10) }

// AirMinutes returns 10 for the side and central pouches.
func (HandyHaversackSpace) AirMinutes(p SpacePouch) int {
	return airFor(p == PouchSide || p == PouchCentral, 10)
}

// AirMinutes returns 60 for PouchMain.
func (PortableHoleSpace) AirMinutes(p SpacePouch) int { return airFor(p == PouchMain, 60) }

// AirMinutes returns the item's air for PouchMain.
func (s SimilarSpace) AirMinutes(p SpacePouch) int { return airFor(p == PouchMain, s.air) }

func airFor(ok bool, minutes int) int {
	if ok {
		return minutes
	}
	return 0
}

// GateArea returns a Sphere of radius 10 feet.
func (BagOfHoldingSpace) GateArea() GateArea { return sphereTen() }

// GateArea returns each creature within 10 feet and not behind Total Cover.
func (HandyHaversackSpace) GateArea() GateArea { return withinTen() }

// GateArea returns each creature within 10 feet and not behind Total Cover.
func (PortableHoleSpace) GateArea() GateArea { return withinTen() }

// GateArea returns the item's own area.
func (s SimilarSpace) GateArea() GateArea { return s.area }

func (BagOfHoldingSpace) isExtradimensionalSpace()   {}
func (HandyHaversackSpace) isExtradimensionalSpace() {}
func (PortableHoleSpace) isExtradimensionalSpace()   {}
func (SimilarSpace) isExtradimensionalSpace()        {}
func (OrdinaryObject) isNestedObject()               {}
func (BagOfHoldingSpace) isNestedObject()            {}
func (HandyHaversackSpace) isNestedObject()          {}
func (PortableHoleSpace) isNestedObject()            {}
func (SimilarSpace) isNestedObject()                 {}

// SpaceCheckResult is the closed set of outcomes of SpaceCheck.Of. Variants:
// SpaceWithinCapacity, SpaceOverloaded.
type SpaceCheckResult interface{ isSpaceCheckResult() }

// SpaceWithinCapacity: the load is at or under every limit.
type SpaceWithinCapacity struct{}

// SpaceOverloaded: the load is above the weight or the volume limit of the pouch.
type SpaceOverloaded struct{ pouch SpacePouch }

// Pouch returns the pouch that is overloaded.
func (r SpaceOverloaded) Pouch() SpacePouch { return r.pouch }

func (SpaceWithinCapacity) isSpaceCheckResult() {}
func (SpaceOverloaded) isSpaceCheckResult()     {}

// SpaceCheck checks a pouch against its limits (SRD 5.2.1, "Bag of Holding",
// p. 212). The zero value is ready to use.
type SpaceCheck struct{}

// Of reports SpaceOverloaded when load is above the weight or the volume limit of
// the pouch; a load exactly at a limit is within capacity, and a pouch with no
// limit never overloads. It returns an error matching srderr.ErrInvalidInput for
// an invalid pouch or a nil space (a negative load cannot be built, NewSpaceLoad refuses it).
func (SpaceCheck) Of(space ExtradimensionalSpace, pouch SpacePouch, load SpaceLoad) (SpaceCheckResult, error) {
	if space == nil || !hasPouch(space, pouch) {
		return nil, invalidInput("space check: nil space or no such pouch")
	}
	if limit, ok := space.Capacity(pouch); ok && (load.ounces > limit.ounces || load.cubicInches > limit.cubicInches) {
		return SpaceOverloaded{pouch: pouch}, nil
	}
	return SpaceWithinCapacity{}, nil
}

// hasPouch reports whether the space has the pouch: the item has air in it, or it
// is a similar item's main pouch with no air stated.
func hasPouch(space ExtradimensionalSpace, pouch SpacePouch) bool {
	if _, ok := space.(SimilarSpace); ok {
		return pouch == PouchMain
	}
	return space.AirMinutes(pouch) > 0
}

// ContainerFate is the closed set of what becomes of a container. Variants:
// ContainerIntact, ContainerContentsScattered, ContainerContentsLost.
type ContainerFate interface{ isContainerFate() }

// ContainerIntact: nothing destroys the item.
type ContainerIntact struct{}

// ContainerContentsScattered: the item is destroyed and its contents scatter in
// the Astral Plane.
type ContainerContentsScattered struct{}

// ContainerContentsLost: the item is destroyed and its contents are lost forever
// (an Artifact turns up again somewhere).
type ContainerContentsLost struct{}

func (ContainerIntact) isContainerFate()            {}
func (ContainerContentsScattered) isContainerFate() {}
func (ContainerContentsLost) isContainerFate()      {}

// ContainerFates decides what overload, piercing or tearing does (SRD 5.2.1,
// "Bag of Holding", p. 212; "Handy Haversack", p. 224). The zero value is ready to use.
type ContainerFates struct{}

// Of returns ContainerContentsScattered for a Bag of Holding that is overloaded,
// pierced or torn, ContainerContentsLost for a Handy Haversack in the same state
// (any pouch), and ContainerIntact for a Portable Hole or a similar item, which
// have no such rule, and for any item neither overloaded nor damaged.
func (ContainerFates) Of(space ExtradimensionalSpace, overloaded, piercedOrTorn bool) ContainerFate {
	if !overloaded && !piercedOrTorn {
		return ContainerIntact{}
	}
	switch space.(type) {
	case BagOfHoldingSpace:
		return ContainerContentsScattered{}
	case HandyHaversackSpace:
		return ContainerContentsLost{}
	}
	return ContainerIntact{}
}

// AstralGate is the gate that opens when one extradimensional item is placed in
// another: one-way, closed for good once creatures are moved, and it moves each
// creature in its area to a random place on the Astral Plane (the roll is the host's).
type AstralGate struct{ area GateArea }

// OneWay is true.
func (AstralGate) OneWay() bool { return true }

// Reopenable is false.
func (AstralGate) Reopenable() bool { return false }

// Area returns who the gate moves, following the placed item's own text.
func (g AstralGate) Area() GateArea { return g.area }

// NestingResult is the closed set of outcomes of ExtradimensionalNesting.Place.
// Variants: NestingSafe, NestingDestroysBoth.
type NestingResult interface{ isNestingResult() }

// NestingSafe: the object is not an extradimensional item.
type NestingSafe struct{}

// NestingDestroysBoth: both items are destroyed and the gate opens.
type NestingDestroysBoth struct{ gate AstralGate }

// Gate returns the Astral Gate.
func (r NestingDestroysBoth) Gate() AstralGate { return r.gate }

func (NestingSafe) isNestingResult()         {}
func (NestingDestroysBoth) isNestingResult() {}

// ExtradimensionalNesting holds the nesting rule (SRD 5.2.1, "Bag of Holding",
// p. 212; "Handy Haversack", p. 224; "Portable Hole", p. 235). The zero value is
// ready to use.
type ExtradimensionalNesting struct{}

// Place returns NestingDestroysBoth when an extradimensional item is placed
// inside another, in any ordered pair of the four kinds including the same kind
// twice (the "similar item" wording; a Hole in a Hole is an interpretation), with
// the gate area of the placed item. Any other object is NestingSafe. It returns an
// error matching srderr.ErrInvalidInput for a nil placed object or space.
func (ExtradimensionalNesting) Place(placed NestedObject, into ExtradimensionalSpace) (NestingResult, error) {
	if placed == nil || into == nil {
		return nil, invalidInput("place in extradimensional space: nil object or space")
	}
	space, ok := placed.(ExtradimensionalSpace)
	if !ok {
		return NestingSafe{}, nil
	}
	return NestingDestroysBoth{gate: AstralGate{area: space.GateArea()}}, nil
}

// AirSupplyResult is the closed set of outcomes of AirSupplies.Of. Variants:
// AirForMinutes, NoOneBreathing.
type AirSupplyResult interface{ isAirSupplyResult() }

// AirForMinutes: the air lasts this many whole minutes.
type AirForMinutes struct{ minutes int }

// Minutes returns the minutes, rounded down.
func (r AirForMinutes) Minutes() int { return r.minutes }

// NoOneBreathing: nobody is breathing, so the air does not run out.
type NoOneBreathing struct{}

func (AirForMinutes) isAirSupplyResult()  {}
func (NoOneBreathing) isAirSupplyResult() {}

// AirSupplies shares a pouch's air among the creatures in it. The zero value is ready to use.
type AirSupplies struct{}

// Of returns the pouch's minutes divided by the number of breathing creatures,
// rounded down (SRD 5.2.1, "Round Down", p. 5), or NoOneBreathing for none. It
// returns an error matching srderr.ErrInvalidInput for a negative count, a nil
// space or a pouch the space lacks.
func (AirSupplies) Of(space ExtradimensionalSpace, pouch SpacePouch, breathingCreatures int) (AirSupplyResult, error) {
	if space == nil || breathingCreatures < 0 || !hasPouch(space, pouch) {
		return nil, invalidInput("air supply: nil space, no such pouch or negative count")
	}
	if breathingCreatures == 0 {
		return NoOneBreathing{}, nil
	}
	return AirForMinutes{minutes: space.AirMinutes(pouch) / breathingCreatures}, nil
}
