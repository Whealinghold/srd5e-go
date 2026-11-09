package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

func load(t *testing.T, ounces, cubicInches int) character.SpaceLoad {
	t.Helper()
	return must(character.NewSpaceLoad(ounces, cubicInches))
}

func lbFt(t *testing.T, pounds, cubicFeet int) character.SpaceLoad {
	t.Helper()
	return must(character.SpaceLoadOf(pounds, cubicFeet))
}

// Requirement: doc/specs/magic-item-effects.md FR-057, FR-058 (SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224; "Portable Hole",
// pp. 234-235) - the capacities of table "Extradimensional spaces" and every load row of "Nesting and load cases": a load is overloaded above the weight
// or the volume limit, exactly at a limit is not, a Portable Hole has no limit, and a negative load or a missing pouch is invalid input.
// Path: per case.
func TestSpaceCheckOf(t *testing.T) {
	t.Parallel()
	bag, hav, hole := character.BagOfHoldingSpace{}, character.HandyHaversackSpace{}, character.PortableHoleSpace{}
	cases := []struct {
		name       string
		space      character.ExtradimensionalSpace
		pouch      character.SpacePouch
		load       character.SpaceLoad
		overloaded bool
	}{
		{"bag at limit", bag, character.PouchMain, lbFt(t, 500, 64), false},
		{"bag one ounce over", bag, character.PouchMain, load(t, 500*16+1, 64*1728), true},
		{"bag volume over, weight under", bag, character.PouchMain, load(t, 100*16, 64*1728+1), true},
		{"bag one ounce under", bag, character.PouchMain, load(t, 500*16-1, 64*1728-1), false},
		{"bag empty", bag, character.PouchMain, character.SpaceLoad{}, false},
		{"side pouch 201 lb", hav, character.PouchSide, lbFt(t, 201, 1), true},
		{"side pouch at limit", hav, character.PouchSide, lbFt(t, 200, 25), false},
		{"side pouch volume over", hav, character.PouchSide, load(t, 0, 25*1728+1), true},
		{"central pouch at limit", hav, character.PouchCentral, lbFt(t, 500, 64), false},
		{"central pouch 201 lb is fine", hav, character.PouchCentral, lbFt(t, 201, 1), false},
		{"central pouch over", hav, character.PouchCentral, load(t, 500*16+1, 0), true},
		{"hole, any load", hole, character.PouchMain, lbFt(t, 100000, 1000), false},
	}
	for _, c := range cases {
		got := must((character.SpaceCheck{}).Of(c.space, c.pouch, c.load))
		switch r := got.(type) {
		case character.SpaceWithinCapacity:
			if c.overloaded {
				t.Errorf("%s: within capacity", c.name)
			}
		case character.SpaceOverloaded:
			if !c.overloaded || r.Pouch() != c.pouch {
				t.Errorf("%s: overloaded %v", c.name, r.Pouch())
			}
		}
	}
	wantInvalid(t, second((character.SpaceCheck{}).Of(nil, character.PouchMain, character.SpaceLoad{})), "nil space")
	wantInvalid(t, second((character.SpaceCheck{}).Of(bag, character.PouchSide, character.SpaceLoad{})), "bag has no side pouch")
	wantInvalid(t, second((character.SpaceCheck{}).Of(hav, character.PouchMain, character.SpaceLoad{})), "haversack has no main pouch")
	wantInvalid(t, second((character.SpaceCheck{}).Of(bag, 0, character.SpaceLoad{})), "pouch 0")
}

// Requirement: doc/specs/magic-item-effects.md FR-058 - a negative load is rejected by its constructor with an error matching ErrInvalidInput.
// Path: unhappy (invalid input).
func TestSpaceLoadConstruction(t *testing.T) {
	t.Parallel()
	wantInvalid(t, second(character.NewSpaceLoad(-1, 0)), "negative ounces")
	wantInvalid(t, second(character.NewSpaceLoad(0, -1)), "negative cubic inches")
	wantInvalid(t, second(character.SpaceLoadOf(-1, 0)), "negative pounds")
	l := lbFt(t, 2, 3)
	if l.Ounces() != 32 || l.CubicInches() != 5184 || (character.SpaceLoad{}).Ounces() != 0 {
		t.Errorf("load = %d oz %d in3", l.Ounces(), l.CubicInches())
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-057, FR-063, FR-064, FR-065 - the table "Extradimensional spaces": capacity per pouch, air, fixed
// weight (5 pounds for the Bag and Haversack, next to nothing for the Hole) and the Hole's escape facts.
// Path: per case.
func TestExtradimensionalSpaceFacts(t *testing.T) {
	t.Parallel()
	bag, hav, hole := character.BagOfHoldingSpace{}, character.HandyHaversackSpace{}, character.PortableHoleSpace{}
	if c, ok := bag.Capacity(character.PouchMain); !ok || c != lbFt(t, 500, 64) {
		t.Error("bag capacity")
	}
	if _, ok := bag.Capacity(character.PouchSide); ok {
		t.Error("bag side pouch")
	}
	if c, ok := hav.Capacity(character.PouchSide); !ok || c != lbFt(t, 200, 25) {
		t.Error("side capacity")
	}
	if c, ok := hav.Capacity(character.PouchCentral); !ok || c != lbFt(t, 500, 64) {
		t.Error("central capacity")
	}
	if _, ok := hav.Capacity(character.PouchMain); ok {
		t.Error("haversack main pouch")
	}
	if _, ok := hole.Capacity(character.PouchMain); ok {
		t.Error("hole has a limit")
	}
	for _, c := range []struct {
		space character.ExtradimensionalSpace
		pouch character.SpacePouch
		air   int
	}{{bag, character.PouchMain, 10}, {bag, character.PouchSide, 0}, {hav, character.PouchSide, 10}, {hav, character.PouchCentral, 10}, {hav, character.PouchMain, 0},
		{hole, character.PouchMain, 60}, {hole, character.PouchSide, 0}} {
		if got := c.space.AirMinutes(c.pouch); got != c.air {
			t.Errorf("%T pouch %d air = %d, want %d", c.space, c.pouch, got, c.air)
		}
	}
	for _, s := range []character.ExtradimensionalSpace{bag, hav} {
		if w, ok := s.Weight().(character.FixedWeight); !ok || w.Ounces() != 80 {
			t.Errorf("%T weight = %+v", s, s.Weight())
		}
	}
	if _, ok := hole.Weight().(character.WeighsNextToNothing); !ok {
		t.Error("hole weight")
	}
	if e := hole.Escape(); e.DepthFeet != 10 || e.DC != 10 || e.AppearsWithinFeet != 5 {
		t.Errorf("escape = %+v", e)
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-059 (SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224) - an overloaded, pierced or torn
// Bag of Holding scatters its contents, a Handy Haversack loses them, a Portable Hole and a similar item have no such rule, and an undamaged item is intact.
// Path: unhappy (rule interaction).
func TestContainerFates(t *testing.T) {
	t.Parallel()
	similar := must(character.NewSimilarSpace(character.WeighsNextToNothing{}, 1, must(character.NewGateArea(character.GateSphere, 5)), nil))
	type fate string
	of := func(s character.ExtradimensionalSpace, over, torn bool) fate {
		switch (character.ContainerFates{}).Of(s, over, torn).(type) {
		case character.ContainerIntact:
			return "intact"
		case character.ContainerContentsScattered:
			return "scattered"
		case character.ContainerContentsLost:
			return "lost"
		}
		return "?"
	}
	for _, c := range []struct {
		name       string
		space      character.ExtradimensionalSpace
		over, torn bool
		want       fate
	}{
		{"bag overloaded", character.BagOfHoldingSpace{}, true, false, "scattered"},
		{"bag pierced", character.BagOfHoldingSpace{}, false, true, "scattered"},
		{"bag fine", character.BagOfHoldingSpace{}, false, false, "intact"},
		{"haversack overloaded", character.HandyHaversackSpace{}, true, false, "lost"},
		{"haversack torn", character.HandyHaversackSpace{}, false, true, "lost"},
		{"haversack fine", character.HandyHaversackSpace{}, false, false, "intact"},
		{"hole overloaded and torn", character.PortableHoleSpace{}, true, true, "intact"},
		{"similar torn", similar, false, true, "intact"},
	} {
		if got := of(c.space, c.over, c.torn); got != c.want {
			t.Errorf("%s = %s, want %s", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-060, FR-061, FR-062 (SRD 5.2.1, "Bag of Holding", p. 212; "Handy Haversack", p. 224; "Portable Hole",
// p. 235) - every ordered pair of the four kinds destroys both items and opens a one-way gate that never reopens, with the placed item's area (a Sphere of
// 10 feet for a Bag, each creature within 10 feet and not behind Total Cover for a Haversack or Hole, the item's own area for a similar item); an
// ordinary object is safe.
// Path: per case.
func TestExtradimensionalNestingPlace(t *testing.T) {
	t.Parallel()
	own := must(character.NewGateArea(character.GateCreaturesWithinNotBehindTotalCover, 30))
	similar := must(character.NewSimilarSpace(character.FixedWeight{}, 5, own, nil))
	spaces := []character.ExtradimensionalSpace{character.BagOfHoldingSpace{}, character.HandyHaversackSpace{}, character.PortableHoleSpace{}, similar}
	sphere := must(character.NewGateArea(character.GateSphere, 10))
	within := must(character.NewGateArea(character.GateCreaturesWithinNotBehindTotalCover, 10))
	wantArea := []character.GateArea{sphere, within, within, own}
	pairs := 0
	for i, placed := range spaces {
		for _, into := range spaces {
			r, ok := must((character.ExtradimensionalNesting{}).Place(placed, into)).(character.NestingDestroysBoth)
			pairs++
			if !ok {
				t.Errorf("%T into %T is safe", placed, into)
				continue
			}
			g := r.Gate()
			if !g.OneWay() || g.Reopenable() || g.Area() != wantArea[i] {
				t.Errorf("%T into %T gate = %+v", placed, into, g)
			}
		}
	}
	if pairs != 16 {
		t.Errorf("%d pairs", pairs)
	}
	for _, into := range spaces {
		if _, ok := must((character.ExtradimensionalNesting{}).Place(character.OrdinaryObject{}, into)).(character.NestingSafe); !ok {
			t.Errorf("rope into %T is not safe", into)
		}
	}
	wantInvalid(t, second((character.ExtradimensionalNesting{}).Place(nil, spaces[0])), "nil placed")
	wantInvalid(t, second((character.ExtradimensionalNesting{}).Place(spaces[0], nil)), "nil into")
	if sphere.Kind() != character.GateSphere || sphere.Feet() != 10 {
		t.Error("area accessors")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-063 (SRD 5.2.1, "Bag of Holding", p. 212; "Portable Hole", p. 235; "Round Down", p. 5) - the air
// divides by the breathing creatures, rounded down: Bag with 3 creatures 3 minutes, Hole with 7 creatures 8 minutes, nobody breathing is NoOneBreathing,
// a negative count is invalid input.
// Path: unhappy (boundary).
func TestAirSuppliesOf(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		space     character.ExtradimensionalSpace
		pouch     character.SpacePouch
		creatures int
		minutes   int
		nobody    bool
	}{
		{"bag, 1", character.BagOfHoldingSpace{}, character.PouchMain, 1, 10, false},
		{"bag, 3", character.BagOfHoldingSpace{}, character.PouchMain, 3, 3, false},
		{"bag, 10", character.BagOfHoldingSpace{}, character.PouchMain, 10, 1, false},
		{"bag, 11", character.BagOfHoldingSpace{}, character.PouchMain, 11, 0, false},
		{"hole, 0", character.PortableHoleSpace{}, character.PouchMain, 0, 0, true},
		{"hole, 7", character.PortableHoleSpace{}, character.PouchMain, 7, 8, false},
		{"haversack side, 2", character.HandyHaversackSpace{}, character.PouchSide, 2, 5, false},
		{"haversack central, 4", character.HandyHaversackSpace{}, character.PouchCentral, 4, 2, false},
	} {
		switch r := must((character.AirSupplies{}).Of(c.space, c.pouch, c.creatures)).(type) {
		case character.AirForMinutes:
			if c.nobody || r.Minutes() != c.minutes {
				t.Errorf("%s = %d", c.name, r.Minutes())
			}
		case character.NoOneBreathing:
			if !c.nobody {
				t.Errorf("%s: no one breathing", c.name)
			}
		}
	}
	wantInvalid(t, second((character.AirSupplies{}).Of(character.BagOfHoldingSpace{}, character.PouchMain, -1)), "negative")
	wantInvalid(t, second((character.AirSupplies{}).Of(nil, character.PouchMain, 1)), "nil space")
	wantInvalid(t, second((character.AirSupplies{}).Of(character.BagOfHoldingSpace{}, character.PouchSide, 1)), "no such pouch")
}

// Requirement: doc/specs/magic-item-effects.md FR-062 - a similar item carries its own data: weight, air, gate area and an optional capacity; the
// constructors reject a nil weight, negative air and an invalid area.
// Path: unhappy (invalid input).
func TestSimilarSpaceAndGateArea(t *testing.T) {
	t.Parallel()
	area := must(character.NewGateArea(character.GateSphere, 5))
	lim := lbFt(t, 10, 1)
	s := must(character.NewSimilarSpace(character.FixedWeight{}, 7, area, &lim))
	if c, ok := s.Capacity(character.PouchMain); !ok || c != lim || s.AirMinutes(character.PouchMain) != 7 || s.AirMinutes(character.PouchSide) != 0 || s.GateArea() != area {
		t.Errorf("similar = %+v", s)
	}
	if _, over := must((character.SpaceCheck{}).Of(s, character.PouchMain, lbFt(t, 11, 0))).(character.SpaceOverloaded); !over {
		t.Error("similar item limit not enforced")
	}
	free := must(character.NewSimilarSpace(character.WeighsNextToNothing{}, 0, area, nil))
	if _, ok := free.Capacity(character.PouchMain); ok {
		t.Error("no capacity expected")
	}
	if _, ok := must((character.SpaceCheck{}).Of(free, character.PouchMain, lbFt(t, 1000, 0))).(character.SpaceWithinCapacity); !ok {
		t.Error("unlimited similar item overloaded")
	}
	heavy := must(character.NewSimilarSpace(must(character.NewFixedWeight(12*character.OuncesPerPound)), 1, area, nil))
	if w, ok := heavy.Weight().(character.FixedWeight); !ok || w.Ounces() != 192 {
		t.Errorf("similar weight = %v, want a fixed 192 ounces", heavy.Weight())
	}
	wantInvalid(t, second(character.NewFixedWeight(-1)), "negative fixed weight")
	wantInvalid(t, second(character.NewGateArea(0, 10)), "kind 0")
	wantInvalid(t, second(character.NewGateArea(character.GateCreaturesWithinNotBehindTotalCover+1, 10)), "kind past the set")
	wantInvalid(t, second(character.NewGateArea(character.GateSphere, 0)), "feet 0")
	wantInvalid(t, second(character.NewSimilarSpace(nil, 1, area, nil)), "nil weight")
	wantInvalid(t, second(character.NewSimilarSpace(character.FixedWeight{}, -1, area, nil)), "negative air")
	wantInvalid(t, second(character.NewSimilarSpace(character.FixedWeight{}, 1, character.GateArea{}, nil)), "zero area")
}
