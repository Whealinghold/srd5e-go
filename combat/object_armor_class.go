package combat

import "github.com/Whealinghold/srd5e-go/d20"

// ObjectSubstance is what an object is made of, for its suggested Armor Class (SRD
// 5.2.1, "Breaking Objects", Object Armor Class, p. 178). The zero value is invalid.
type ObjectSubstance int

// The seven substances.
const (
	// ObjectSubstanceClothPaperRope: cloth, paper or rope.
	ObjectSubstanceClothPaperRope ObjectSubstance = iota + 1
	// ObjectSubstanceCrystalGlassIce: crystal, glass or ice.
	ObjectSubstanceCrystalGlassIce
	// ObjectSubstanceWood: wood.
	ObjectSubstanceWood
	// ObjectSubstanceStone: stone.
	ObjectSubstanceStone
	// ObjectSubstanceIronSteel: iron or steel.
	ObjectSubstanceIronSteel
	// ObjectSubstanceMithral: mithral.
	ObjectSubstanceMithral
	// ObjectSubstanceAdamantine: adamantine.
	ObjectSubstanceAdamantine
)

// Valid reports whether s is one of the seven substances.
func (s ObjectSubstance) Valid() bool {
	return s >= ObjectSubstanceClothPaperRope && s <= ObjectSubstanceAdamantine
}

// String returns the snake_case name of s, or "unknown".
func (s ObjectSubstance) String() string {
	return enumName([]string{"cloth_paper_rope", "crystal_glass_ice", "wood", "stone", "iron_steel", "mithral", "adamantine"}, int(s))
}

// ObjectArmorClassTable is the table of suggested Armor Class by substance (SRD
// 5.2.1, p. 178). The host may give an object any Armor Class; this is a lookup.
// The zero value is ready to use.
type ObjectArmorClassTable struct{}

// Of returns the Armor Class of an object of substance s: 11, 13, 15, 17, 19, 21
// and 23 in table order. It returns an error matching srderr.ErrInvalidInput for a
// substance that is not valid.
func (ObjectArmorClassTable) Of(s ObjectSubstance) (d20.ArmorClass, error) {
	if !s.Valid() {
		return 0, invalidInput("invalid object substance")
	}
	return d20.ArmorClass(9 + 2*int(s)), nil
}
