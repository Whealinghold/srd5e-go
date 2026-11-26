package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
)

// TeleportFamiliarity is the row of the Teleportation Outcome table a destination falls in (SRD 5.2.1, "Teleport",
// "Familiarity", p. 168). It is a fact the Game Master judges and the host supplies. The zero value is invalid.
type TeleportFamiliarity int

// The familiarity rows, in table order.
const (
	// PermanentCircle is a destination with a permanent teleportation circle.
	PermanentCircle TeleportFamiliarity = iota + 1
	// LinkedObject is a destination reached through an object linked to it.
	LinkedObject
	// VeryFamiliar is a place the caster knows well.
	VeryFamiliar
	// SeenCasually is a place the caster has seen casually.
	SeenCasually
	// ViewedOnceOrDescribed is a place seen once or only described.
	ViewedOnceOrDescribed
	// FalseDestination is a destination that does not exist.
	FalseDestination
)

// Valid reports whether f is one of the six rows.
func (f TeleportFamiliarity) Valid() bool { return f >= PermanentCircle && f <= FalseDestination }

// String returns the snake_case name of the row, or "unknown".
func (f TeleportFamiliarity) String() string {
	return enumName([]string{"permanent_circle", "linked_object", "very_familiar", "seen_casually",
		"viewed_once_or_described", "false_destination"}, int(f))
}

// TeleportOutcomeKind is one column of the Teleportation Outcome table (SRD 5.2.1, p. 168). The zero value is
// invalid.
type TeleportOutcomeKind int

// The columns of the table.
const (
	// TeleportMishapOutcome is the Mishap column: damage and another roll.
	TeleportMishapOutcome TeleportOutcomeKind = iota + 1
	// TeleportSimilarArea is the Similar Area column.
	TeleportSimilarArea
	// TeleportOffTarget is the Off Target column.
	TeleportOffTarget
	// TeleportOnTarget is the On Target column.
	TeleportOnTarget
)

// String returns the value of srd5e.combat.table_outcome for the column: mishap, similar_area, off_target or
// on_target, or "unknown".
func (k TeleportOutcomeKind) String() string {
	return enumName([]string{"mishap", "similar_area", "off_target", "on_target"}, int(k))
}

// teleportTable returns the 1d100 table of f, false for a familiarity that is not a row (table T of
// spell-outcome-tables.md, SRD 5.2.1, "Teleportation Outcome", p. 168; the printed "00" is face 100).
func teleportTable(f TeleportFamiliarity) (outcomeTable[TeleportOutcomeKind], bool) {
	b := func(low, high int, k TeleportOutcomeKind) outcomeBand[TeleportOutcomeKind] {
		return outcomeBand[TeleportOutcomeKind]{low, high, k}
	}
	var bands []outcomeBand[TeleportOutcomeKind]
	switch f {
	case PermanentCircle, LinkedObject:
		bands = []outcomeBand[TeleportOutcomeKind]{b(1, 100, TeleportOnTarget)}
	case VeryFamiliar:
		bands = []outcomeBand[TeleportOutcomeKind]{b(1, 5, TeleportMishapOutcome), b(6, 13, TeleportSimilarArea),
			b(14, 24, TeleportOffTarget), b(25, 100, TeleportOnTarget)}
	case SeenCasually:
		bands = []outcomeBand[TeleportOutcomeKind]{b(1, 33, TeleportMishapOutcome), b(34, 43, TeleportSimilarArea),
			b(44, 53, TeleportOffTarget), b(54, 100, TeleportOnTarget)}
	case ViewedOnceOrDescribed:
		bands = []outcomeBand[TeleportOutcomeKind]{b(1, 43, TeleportMishapOutcome), b(44, 53, TeleportSimilarArea),
			b(54, 73, TeleportOffTarget), b(74, 100, TeleportOnTarget)}
	case FalseDestination:
		bands = []outcomeBand[TeleportOutcomeKind]{b(1, 50, TeleportMishapOutcome), b(51, 100, TeleportSimilarArea)}
	default:
		return outcomeTable[TeleportOutcomeKind]{}, false
	}
	return newOutcomeTable(dice.D100(), bands), true
}

// TeleportKindFor returns the column of the Teleportation Outcome table for the familiarity row and the 1d100 face
// (SRD 5.2.1, p. 168). It returns an error matching srderr.ErrInvalidInput for a familiarity that is not a row, and
// one matching ErrFaceOutOfRange for a face outside 1 to 100. It never panics.
func TeleportKindFor(f TeleportFamiliarity, face int) (TeleportOutcomeKind, error) {
	t, ok := teleportTable(f)
	if !ok {
		return 0, invalidInput("teleport familiarity is not a row of the table")
	}
	return t.lookup(face)
}

// TeleportNeedsRoll reports whether the familiarity row can give anything but On Target. A permanent circle and a
// linked object always arrive on target, so no d100 is requested for them (spell-outcome-tables.md FR-011).
func TeleportNeedsRoll(f TeleportFamiliarity) bool {
	return f.Valid() && f != PermanentCircle && f != LinkedObject
}

// CompassDirection is one of the eight directions of the Off Target d8, in the order of the die: 1 east, 2 southeast,
// 3 south, 4 southwest, 5 west, 6 northwest, 7 north, 8 northeast (SRD 5.2.1, "Off Target", p. 168). The zero value
// is invalid.
type CompassDirection int

// The directions, in the order of the d8.
const (
	// East is face 1.
	East CompassDirection = iota + 1
	// Southeast is face 2.
	Southeast
	// South is face 3.
	South
	// Southwest is face 4.
	Southwest
	// West is face 5.
	West
	// Northwest is face 6.
	Northwest
	// North is face 7.
	North
	// Northeast is face 8.
	Northeast
)

// String returns the lowercase name of the direction, or "unknown".
func (d CompassDirection) String() string {
	return enumName([]string{"east", "southeast", "south", "southwest", "west", "northwest", "north", "northeast"}, int(d))
}

// Valid reports whether d is one of the eight directions.
func (d CompassDirection) Valid() bool { return d >= East && d <= Northeast }

// IsCardinal reports whether d is north, east, south or west.
func (d CompassDirection) IsCardinal() bool {
	return d == North || d == East || d == South || d == West
}

// TeleportTableArrival is the closed set of arrivals a Teleportation Outcome gives (the name TeleportArrival is the
// arrival of a teleport that has been placed, in the magical effect events). Variants: ArrivesOnTarget,
// ArrivesInSimilarArea, ArrivesOffTarget.
type TeleportTableArrival interface{ isTeleportTableArrival() }

// ArrivesOnTarget: the teleporters arrive where they meant to ("On Target", SRD 5.2.1, p. 168).
type ArrivesOnTarget struct{}

// ArrivesInSimilarArea: the teleporters arrive in a similar area; which one is the host's ("Similar Area", p. 168).
type ArrivesInSimilarArea struct{}

// ArrivesOffTarget: the teleporters arrive some miles away in a direction ("Off Target", p. 168). Build it with
// NewArrivesOffTarget; the zero value is invalid.
type ArrivesOffTarget struct {
	miles     int
	direction CompassDirection
}

// NewArrivesOffTarget returns the arrival miles away toward direction. It returns an error matching
// srderr.ErrInvalidInput for miles outside 2 to 24 (the total of 2d12) and a direction that is not valid. It never
// panics.
func NewArrivesOffTarget(miles int, direction CompassDirection) (ArrivesOffTarget, error) {
	if miles < 2 || miles > 24 || !direction.Valid() {
		return ArrivesOffTarget{}, invalidInput("off target arrival needs 2 to 24 miles and a direction")
	}
	return ArrivesOffTarget{miles, direction}, nil
}

// Miles returns the distance in miles.
func (a ArrivesOffTarget) Miles() int { return a.miles }

// Direction returns the direction.
func (a ArrivesOffTarget) Direction() CompassDirection { return a.direction }

func (ArrivesOnTarget) isTeleportTableArrival()      {}
func (ArrivesInSimilarArea) isTeleportTableArrival() {}
func (ArrivesOffTarget) isTeleportTableArrival()     {}

// TeleportSubject is the closed set of what teleports (SRD 5.2.1, "Teleport", p. 168). Variants:
// TeleportingCreatures, TeleportingObject.
type TeleportSubject interface{ isTeleportSubject() }

// TeleportingCreatures is one to nine distinct creatures: "you and up to eight willing creatures". Build it with
// NewTeleportingCreatures; the zero value holds none and is invalid. Immutable.
type TeleportingCreatures struct{ ids []CombatantID }

// maxTeleportingCreatures is the caster and up to eight willing creatures (SRD 5.2.1, "Teleport", p. 168).
const maxTeleportingCreatures = 9

// NewTeleportingCreatures returns the creatures, copied. It returns an error matching srderr.ErrInvalidInput for
// none, for more than 9, for a zero id and for a repeated id. It never panics.
func NewTeleportingCreatures(ids []CombatantID) (TeleportingCreatures, error) {
	if len(ids) < 1 || len(ids) > maxTeleportingCreatures || slices.ContainsFunc(ids, CombatantID.IsZero) {
		return TeleportingCreatures{}, invalidInput("teleporting creatures need 1 to 9 ids")
	}
	if len(slices.Compact(slices.Sorted(slices.Values(ids)))) != len(ids) {
		return TeleportingCreatures{}, invalidInput("teleporting creature repeated")
	}
	return TeleportingCreatures{slices.Clone(ids)}, nil
}

// IDs returns the creatures in order, as a new slice.
func (c TeleportingCreatures) IDs() []CombatantID { return slices.Clone(c.ids) }

// TeleportingObject is a single object ("or a single object", SRD 5.2.1, p. 168). Its object limits are the host's.
type TeleportingObject struct{}

func (TeleportingCreatures) isTeleportSubject() {}
func (TeleportingObject) isTeleportSubject()    {}

// teleportSubjectCount is the number of Mishap rolls one Mishap needs: one per creature, one for an object. It is 0
// for the zero or an unknown subject.
func teleportSubjectCount(s TeleportSubject) int {
	switch v := s.(type) {
	case TeleportingCreatures:
		return len(v.ids)
	case TeleportingObject:
		return 1
	}
	return 0
}

// TeleportMishap is the damage of one Mishap: one Damage per teleporting creature, in the order of the subject, one
// for an object (SRD 5.2.1, "Mishap", p. 168). Immutable.
type TeleportMishap struct{ damage []Damage }

// Damage returns the 3d10 Force damage of each teleporter, as a new slice.
func (m TeleportMishap) Damage() []Damage { return slices.Clone(m.damage) }

// TeleportOutcome is what one Teleport roll gave: the familiarity, the subject, every Mishap in order and the
// arrival. It applies nothing: the host applies each Damage and places the arrival. Immutable; compare with Equal.
type TeleportOutcome struct {
	familiarity TeleportFamiliarity
	subject     TeleportSubject
	mishaps     []TeleportMishap
	arrival     TeleportTableArrival
}

// Familiarity returns the row that was rolled.
func (o TeleportOutcome) Familiarity() TeleportFamiliarity { return o.familiarity }

// Subject returns who teleported.
func (o TeleportOutcome) Subject() TeleportSubject { return o.subject }

// Mishaps returns every Mishap, in order, as a new slice; it is empty when there was none.
func (o TeleportOutcome) Mishaps() []TeleportMishap {
	out := make([]TeleportMishap, len(o.mishaps))
	for i, m := range o.mishaps {
		out[i] = TeleportMishap{slices.Clone(m.damage)}
	}
	return out
}

// Arrival returns the arrival of the first roll that was not a Mishap.
func (o TeleportOutcome) Arrival() TeleportTableArrival { return o.arrival }

// Equal reports whether o and p are the same outcome.
func (o TeleportOutcome) Equal(p TeleportOutcome) bool {
	return o.familiarity == p.familiarity && o.arrival == p.arrival && sameSubject(o.subject, p.subject) &&
		slices.EqualFunc(o.mishaps, p.mishaps, func(a, b TeleportMishap) bool { return slices.Equal(a.damage, b.damage) })
}

func sameSubject(a, b TeleportSubject) bool {
	x, ok := a.(TeleportingCreatures)
	y, ok2 := b.(TeleportingCreatures)
	if ok || ok2 {
		return ok && ok2 && slices.Equal(x.ids, y.ids)
	}
	return a == b
}
