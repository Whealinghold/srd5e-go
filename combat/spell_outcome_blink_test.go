package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-040, FR-044 - BlinkVanishesOn is false for 1 to 3 and true for 4 to 6, and a face outside 1 to 6 is ErrFaceOutOfRange (SRD 5.2.1, "Blink", p. 113).
// Path: per case.
func TestBlinkVanishesOn(t *testing.T) {
	t.Parallel()
	for face, want := range map[int]bool{1: false, 2: false, 3: false, 4: true, 5: true, 6: true} {
		if got, err := combat.BlinkVanishesOn(face); err != nil || got != want {
			t.Errorf("BlinkVanishesOn(%d) = %v, %v, want %v [%sFR-044]", face, got, err, want, otSpec)
		}
	}
	for _, face := range []int{0, 7, -3} {
		if _, err := combat.BlinkVanishesOn(face); !errors.Is(err, combat.ErrFaceOutOfRange) {
			t.Errorf("BlinkVanishesOn(%d) = %v, want ErrFaceOutOfRange", face, err)
		}
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-043 - the return gives a placement for a caster on the Ethereal Plane (an unoccupied space the caster can see within 10 feet of the one it left, else the nearest one) and nothing to return for one on the other plane (SRD 5.2.1, "Blink", p. 114).
// Path: per case.
func TestBlinkReturnFor(t *testing.T) {
	t.Parallel()
	back, ok := combat.BlinkReturnFor(combat.BlinkOnEtherealPlane).(combat.BlinkReturnsToOtherPlane)
	if !ok || back.ChosenSpaceWithinFeet() != 10 || !back.NearestIfNoneChosen() {
		t.Errorf("return from the Ethereal Plane = %#v, want BlinkReturnsToOtherPlane with 10 feet and the nearest space", back)
	}
	if got := combat.BlinkReturnFor(combat.BlinkOnOtherPlane); got != (combat.BlinkNothingToReturn{}) {
		t.Errorf("return from the other plane = %#v, want BlinkNothingToReturn", got)
	}
	if got := combat.BlinkReturnFor(0); got != (combat.BlinkNothingToReturn{}) {
		t.Errorf("return for an invalid location = %#v, want nothing to return", got)
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-040 to FR-042 - the locations are valid only as the two planes and have snake_case names; the vanish outcome reports the Ethereal Plane.
// Path: unhappy (invalid input).
func TestBlinkLocation(t *testing.T) {
	t.Parallel()
	if !combat.BlinkOnOtherPlane.Valid() || !combat.BlinkOnEtherealPlane.Valid() || combat.BlinkLocation(0).Valid() || combat.BlinkLocation(3).Valid() {
		t.Error("only the two planes are locations")
	}
	if combat.BlinkOnOtherPlane.String() != "other_plane" || combat.BlinkOnEtherealPlane.String() != "ethereal_plane" || combat.BlinkLocation(0).String() != "unknown" {
		t.Error("location names")
	}
	if (combat.BlinkVanishesToEtherealPlane{}).Location() != combat.BlinkOnEtherealPlane {
		t.Error("a vanish arrives on the Ethereal Plane")
	}
}
