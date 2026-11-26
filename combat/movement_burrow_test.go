package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/movement-lifting-and-flight.md FR-048 - BurrowMedium has exactly five values in the order Sand, Earth, Mud, Ice, SolidRock (SRD 5.2.1, "Burrow Speed", p. 178).
// Path: happy.
func TestBurrowMediumValues(t *testing.T) {
	t.Parallel()
	for i, w := range []string{"sand", "earth", "mud", "ice", "solid_rock"} {
		m := combat.BurrowMedium(i + 1)
		if !m.Valid() || m.String() != w {
			t.Errorf("medium %d = %q valid %v, want %q", i+1, m, m.Valid(), w)
		}
	}
	for _, m := range []combat.BurrowMedium{0, combat.BurrowMediumSolidRock + 1} {
		if m.Valid() || m.String() != "unknown" {
			t.Errorf("medium %d: Valid %v, String %q", m, m.Valid(), m)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestBurrowPermissionFor(t *testing.T) {
	t.Parallel()
	const r = `doc/specs/movement-lifting-and-flight.md cases 4.6a; SRD 5.2.1, "Burrow Speed", p. 178`
	allowed, rock := combat.BurrowAllowed{}, combat.BurrowRefused{Reason: combat.BurrowRefusalSolidRock}
	cases := []struct {
		name, path string
		medium     combat.BurrowMedium
		access     combat.SolidRockAccess
		want       combat.BurrowPermission
	}{
		{"sand (FR-049)", "happy", combat.BurrowMediumSand, combat.SolidRockAccessNone, allowed},
		{"earth (FR-049)", "happy", combat.BurrowMediumEarth, combat.SolidRockAccessNone, allowed},
		{"mud (FR-049)", "happy", combat.BurrowMediumMud, combat.SolidRockAccessNone, allowed},
		{"ice (FR-049)", "happy", combat.BurrowMediumIce, combat.SolidRockAccessNone, allowed},
		{"earth, trait irrelevant (FR-049)", "unhappy (rule interaction)", combat.BurrowMediumEarth, combat.SolidRockAccessAllowedByTrait, allowed},
		{"solid rock (FR-050)", "unhappy (rule interaction)", combat.BurrowMediumSolidRock, combat.SolidRockAccessNone, rock},
		{"solid rock with a trait (FR-051)", "unhappy (rule interaction)", combat.BurrowMediumSolidRock, combat.SolidRockAccessAllowedByTrait, allowed},
		{"an invalid medium is refused with the zero reason", "unhappy (invalid input)", 0, combat.SolidRockAccessAllowedByTrait, combat.BurrowRefused{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.BurrowPermissionFor(c.medium, c.access); got != c.want {
				t.Errorf("BurrowPermissionFor(%v, %v) = %#v, want %#v [%s, %s]", c.medium, c.access, got, c.want, r, c.path)
			}
		})
	}
}
