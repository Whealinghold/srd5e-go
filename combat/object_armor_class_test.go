package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-066 (table 4.9a) - the Armor Class of each of the seven substances; an invalid substance is an ErrInvalidInput error.
// Path: per case.
func TestObjectArmorClassTable(t *testing.T) {
	t.Parallel()
	want := map[combat.ObjectSubstance]struct {
		ac   int
		name string
	}{combat.ObjectSubstanceClothPaperRope: {11, "cloth_paper_rope"}, combat.ObjectSubstanceCrystalGlassIce: {13, "crystal_glass_ice"},
		combat.ObjectSubstanceWood: {15, "wood"}, combat.ObjectSubstanceStone: {17, "stone"}, combat.ObjectSubstanceIronSteel: {19, "iron_steel"},
		combat.ObjectSubstanceMithral: {21, "mithral"}, combat.ObjectSubstanceAdamantine: {23, "adamantine"}}
	for s, w := range want {
		got, err := (combat.ObjectArmorClassTable{}).Of(s)
		if err != nil || int(got) != w.ac || s.String() != w.name || !s.Valid() {
			t.Errorf("Of(%s) = %d, %v, want %d", s, got, err, w.ac)
		}
	}
	for _, s := range []combat.ObjectSubstance{0, 8, -1} {
		if _, err := (combat.ObjectArmorClassTable{}).Of(s); !errors.Is(err, srderr.ErrInvalidInput) || s.Valid() || s.String() != "unknown" {
			t.Errorf("Of(%d) = %v, want ErrInvalidInput", int(s), err)
		}
	}
}
