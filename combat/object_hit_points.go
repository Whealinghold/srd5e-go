package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ObjectDurability says how sturdy an object is, for its suggested Hit Points (SRD
// 5.2.1, "Breaking Objects", Object Hit Points, p. 178). The zero value is invalid.
type ObjectDurability int

// The two durabilities.
const (
	// ObjectFragile: fragile objects have fewer Hit Points.
	ObjectFragile ObjectDurability = iota + 1
	// ObjectResilient: resilient objects have more.
	ObjectResilient
)

// Valid reports whether d is one of the two durabilities.
func (d ObjectDurability) Valid() bool { return d == ObjectFragile || d == ObjectResilient }

// String returns "fragile", "resilient" or "unknown".
func (d ObjectDurability) String() string { return enumName([]string{"fragile", "resilient"}, int(d)) }

// ObjectHitPointsEntry is one row of the Object Hit Points table: the fixed Hit
// Points and the die expression they are the average of. It is comparable. The
// zero value is not a row.
type ObjectHitPointsEntry struct {
	hitPoints  int
	expression dice.Expression
}

// HitPoints returns the fixed Hit Points.
func (e ObjectHitPointsEntry) HitPoints() int { return e.hitPoints }

// Expression returns the die expression to roll instead.
func (e ObjectHitPointsEntry) Expression() dice.Expression { return e.expression }

// ObjectHitPointsLookup is the result of ObjectHitPointsTable.Lookup. Variants:
// ObjectHitPointsFound, ObjectHitPointsSplitRequired.
type ObjectHitPointsLookup interface{ isObjectHitPointsLookup() }

// ObjectHitPointsFound: the table has a row for the size and durability.
type ObjectHitPointsFound struct{ Entry ObjectHitPointsEntry }

// ObjectHitPointsSplitRequired: the object is Huge or Gargantuan, which the table
// does not cover: divide it into Large or smaller sections and track them
// separately (SRD 5.2.1, p. 178). Size is the size asked for.
type ObjectHitPointsSplitRequired struct{ Size content.CreatureSize }

func (ObjectHitPointsFound) isObjectHitPointsLookup()         {}
func (ObjectHitPointsSplitRequired) isObjectHitPointsLookup() {}

// ObjectHitPointsTable is the table of suggested Object Hit Points for Tiny to Large
// objects (SRD 5.2.1, p. 178). The zero value is ready to use.
type ObjectHitPointsTable struct{}

// Lookup returns the row for size and durability, or ObjectHitPointsSplitRequired
// for a Huge or Gargantuan object. It returns an error matching
// srderr.ErrInvalidInput for a size or durability that is not valid.
func (ObjectHitPointsTable) Lookup(size content.CreatureSize, d ObjectDurability) (ObjectHitPointsLookup, error) {
	if !size.Valid() || !d.Valid() {
		return nil, invalidInput("object hit points need a valid size and durability")
	}
	if size > content.SizeLarge {
		return ObjectHitPointsSplitRequired{size}, nil
	}
	die := []dice.Die{dice.D4(), dice.D6(), dice.D8(), dice.D10()}[size-content.SizeTiny]
	count := 1
	if d == ObjectResilient {
		count = int(size-content.SizeTiny) + 2
	}
	e := dice.MustExpression(count, die, 0)
	return ObjectHitPointsFound{ObjectHitPointsEntry{e.Average(), e}}, nil
}
