package combat

import "github.com/Whealinghold/srd5e-go/dice"

// BlinkLocation is where the caster of Blink is. The zero value is invalid.
type BlinkLocation int

const (
	// BlinkOnOtherPlane is the plane the caster cast the spell on.
	BlinkOnOtherPlane BlinkLocation = iota + 1
	// BlinkOnEtherealPlane is the Ethereal Plane.
	BlinkOnEtherealPlane
)

// Valid reports whether l is one of the two locations.
func (l BlinkLocation) Valid() bool { return l == BlinkOnOtherPlane || l == BlinkOnEtherealPlane }

// String returns "other_plane", "ethereal_plane" or "unknown".
func (l BlinkLocation) String() string {
	return enumName([]string{"other_plane", "ethereal_plane"}, int(l))
}

// blinkTable is the 1d6 split of Blink: 1 to 3 stays, 4 to 6 vanishes (SRD 5.2.1, "Blink", p. 113). The value is true
// for a vanish.
func blinkTable() outcomeTable[bool] {
	return newOutcomeTable(dice.D6(), []outcomeBand[bool]{{1, 3, false}, {4, 6, true}})
}

// BlinkVanishesOn reports whether the 1d6 face makes the caster vanish: 4 to 6 do, 1 to 3 do not. It returns an
// error matching ErrFaceOutOfRange for a face outside 1 to 6. It never panics.
func BlinkVanishesOn(face int) (bool, error) { return blinkTable().lookup(face) }

// BlinkReturn is the closed set of what happens to a caster on the Ethereal Plane when it is time to return.
// Variants: BlinkReturnsToOtherPlane, BlinkNothingToReturn.
type BlinkReturn interface{ isBlinkReturn() }

// BlinkReturnsToOtherPlane: the caster returns to the other plane, to an unoccupied space of its choice that it can
// see within 10 feet of the space it left, or else to the nearest unoccupied space (SRD 5.2.1, "Blink", p. 114).
// The result chooses no space.
type BlinkReturnsToOtherPlane struct{}

// ChosenSpaceWithinFeet returns 10: the distance from the space the caster left.
func (BlinkReturnsToOtherPlane) ChosenSpaceWithinFeet() int { return 10 }

// NearestIfNoneChosen returns true: with no such space the nearest unoccupied space is used.
func (BlinkReturnsToOtherPlane) NearestIfNoneChosen() bool { return true }

// BlinkNothingToReturn: the caster is not on the Ethereal Plane.
type BlinkNothingToReturn struct{}

func (BlinkReturnsToOtherPlane) isBlinkReturn() {}
func (BlinkNothingToReturn) isBlinkReturn()     {}

// BlinkReturnFor returns what happens at the start of the caster's next turn and when the spell ends: a return for a
// caster on the Ethereal Plane, nothing for one on the other plane (SRD 5.2.1, "Blink", p. 114).
func BlinkReturnFor(location BlinkLocation) BlinkReturn {
	if location == BlinkOnEtherealPlane {
		return BlinkReturnsToOtherPlane{}
	}
	return BlinkNothingToReturn{}
}

// BlinkTurnEndOutcome is the closed set of results of the 1d6 at the end of the caster's turn. Variants:
// BlinkStaysOnPlane, BlinkVanishesToEtherealPlane, BlinkEndsAlreadyEthereal.
type BlinkTurnEndOutcome interface{ isBlinkTurnEndOutcome() }

// BlinkStaysOnPlane: a 1 to 3. The location is unchanged.
type BlinkStaysOnPlane struct{ location BlinkLocation }

// Location returns the unchanged location.
func (o BlinkStaysOnPlane) Location() BlinkLocation { return o.location }

// BlinkVanishesToEtherealPlane: a 4 to 6 for a caster on the other plane. The new location is the Ethereal Plane.
type BlinkVanishesToEtherealPlane struct{}

// Location returns BlinkOnEtherealPlane.
func (BlinkVanishesToEtherealPlane) Location() BlinkLocation { return BlinkOnEtherealPlane }

// BlinkEndsAlreadyEthereal: a 4 to 6 for a caster already on the Ethereal Plane. The spell ends instantly, and the
// caster returns as Return says.
type BlinkEndsAlreadyEthereal struct{ ret BlinkReturn }

// Return returns BlinkReturnFor(BlinkOnEtherealPlane).
func (o BlinkEndsAlreadyEthereal) Return() BlinkReturn { return o.ret }

func (BlinkStaysOnPlane) isBlinkTurnEndOutcome()            {}
func (BlinkVanishesToEtherealPlane) isBlinkTurnEndOutcome() {}
func (BlinkEndsAlreadyEthereal) isBlinkTurnEndOutcome()     {}
