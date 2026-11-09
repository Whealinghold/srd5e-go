package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// enumByName returns the 1-based position of name among names, or 0.
func enumByName(name string, names []string) int { return slices.Index(names, name) + 1 }

// DraconicAncestry is the kind of dragon a Dragonborn descends from (SRD 5.2.1,
// "Dragonborn", Draconic Ancestors, p. 84). The zero value is invalid.
// Comparable and safe for concurrent use.
type DraconicAncestry int

// The ten Draconic Ancestors.
const (
	DraconicBlack DraconicAncestry = iota + 1
	DraconicBlue
	DraconicBrass
	DraconicBronze
	DraconicCopper
	DraconicGold
	DraconicGreen
	DraconicRed
	DraconicSilver
	DraconicWhite
)

func draconicNames() []string {
	return []string{"Black", "Blue", "Brass", "Bronze", "Copper", "Gold", "Green", "Red", "Silver", "White"}
}

// Valid reports whether a is one of the ten ancestors.
func (a DraconicAncestry) Valid() bool { return a >= DraconicBlack && a <= DraconicWhite }

// String returns the printed name, or "unknown".
func (a DraconicAncestry) String() string { return featNameOf(int(a), draconicNames()...) }

// DamageType returns the damage type of the ancestry (Black and Copper Acid; Blue and
// Bronze Lightning; Brass, Gold and Red Fire; Green Poison; Silver and White Cold),
// or the zero DamageType for an invalid value.
func (a DraconicAncestry) DamageType() content.DamageType {
	if !a.Valid() {
		return 0
	}
	return [...]content.DamageType{content.DamageAcid, content.DamageLightning, content.DamageFire, content.DamageLightning,
		content.DamageAcid, content.DamageFire, content.DamagePoison, content.DamageFire, content.DamageCold, content.DamageCold}[a-1]
}

// ParseDraconicAncestry returns the ancestry printed as name (exact, case
// sensitive) and true, or the zero value and false. It never panics.
func ParseDraconicAncestry(name string) (DraconicAncestry, bool) {
	a := DraconicAncestry(enumByName(name, draconicNames()))
	return a, a.Valid()
}

// ElvenLineage is an Elven Lineage (SRD 5.2.1, "Elf", p. 85). The zero value is
// invalid. Comparable and safe for concurrent use.
type ElvenLineage int

// The Elven Lineages.
const (
	ElvenDrow ElvenLineage = iota + 1
	ElvenHighElf
	ElvenWoodElf
)

func elvenNames() []string { return []string{"Drow", "High Elf", "Wood Elf"} }

// Valid reports whether l is one of the three lineages.
func (l ElvenLineage) Valid() bool { return l >= ElvenDrow && l <= ElvenWoodElf }

// String returns the printed name, or "unknown".
func (l ElvenLineage) String() string { return featNameOf(int(l), elvenNames()...) }

// ParseElvenLineage returns the lineage printed as name (exact, case sensitive)
// and true, or the zero value and false. It never panics.
func ParseElvenLineage(name string) (ElvenLineage, bool) {
	l := ElvenLineage(enumByName(name, elvenNames()))
	return l, l.Valid()
}

// GnomishLineage is a Gnomish Lineage (SRD 5.2.1, "Gnome", p. 85). The zero
// value is invalid. Comparable and safe for concurrent use.
type GnomishLineage int

// The Gnomish Lineages.
const (
	GnomishForest GnomishLineage = iota + 1
	GnomishRock
)

func gnomishNames() []string { return []string{"Forest Gnome", "Rock Gnome"} }

// Valid reports whether l is one of the two lineages.
func (l GnomishLineage) Valid() bool { return l >= GnomishForest && l <= GnomishRock }

// String returns the printed name, or "unknown".
func (l GnomishLineage) String() string { return featNameOf(int(l), gnomishNames()...) }

// ParseGnomishLineage returns the lineage printed as name (exact, case sensitive)
// and true, or the zero value and false. It never panics.
func ParseGnomishLineage(name string) (GnomishLineage, bool) {
	l := GnomishLineage(enumByName(name, gnomishNames()))
	return l, l.Valid()
}

// GiantAncestry is a Giant Ancestry benefit of a Goliath (SRD 5.2.1, "Goliath",
// pp. 85-86); the benefit is chosen with the ancestry, so one value names both.
// The zero value is invalid. Comparable and safe for concurrent use.
type GiantAncestry int

// The six Giant Ancestry benefits.
const (
	AncestryCloud GiantAncestry = iota + 1
	AncestryFire
	AncestryFrost
	AncestryHill
	AncestryStone
	AncestryStorm
)

// giantNames are the printed headings, with the typographic apostrophe.
func giantNames() []string {
	return []string{"Cloud’s Jaunt (Cloud Giant)", "Fire’s Burn (Fire Giant)", "Frost’s Chill (Frost Giant)",
		"Hill’s Tumble (Hill Giant)", "Stone’s Endurance (Stone Giant)", "Storm’s Thunder (Storm Giant)"}
}

// Valid reports whether a is one of the six.
func (a GiantAncestry) Valid() bool { return a >= AncestryCloud && a <= AncestryStorm }

// String returns the printed heading, or "unknown".
func (a GiantAncestry) String() string { return featNameOf(int(a), giantNames()...) }

// ParseGiantAncestry returns the benefit printed as name (exact, case sensitive,
// with the descriptor and the typographic apostrophe) and true, or the zero value
// and false. It never panics.
func ParseGiantAncestry(name string) (GiantAncestry, bool) {
	a := GiantAncestry(enumByName(name, giantNames()))
	return a, a.Valid()
}

// FiendishLegacy is a Fiendish Legacy (SRD 5.2.1, "Tiefling", p. 86). The zero
// value is invalid. Comparable and safe for concurrent use.
type FiendishLegacy int

// The Fiendish Legacies.
const (
	LegacyAbyssal FiendishLegacy = iota + 1
	LegacyChthonic
	LegacyInfernal
)

func legacyNames() []string { return []string{"Abyssal", "Chthonic", "Infernal"} }

// Valid reports whether l is one of the three legacies.
func (l FiendishLegacy) Valid() bool { return l >= LegacyAbyssal && l <= LegacyInfernal }

// String returns the printed name, or "unknown".
func (l FiendishLegacy) String() string { return featNameOf(int(l), legacyNames()...) }

// Resistance returns the damage type of the legacy's Resistance (Abyssal Poison,
// Chthonic Necrotic, Infernal Fire), or the zero DamageType for an invalid value.
func (l FiendishLegacy) Resistance() content.DamageType {
	if !l.Valid() {
		return 0
	}
	return [...]content.DamageType{content.DamagePoison, content.DamageNecrotic, content.DamageFire}[l-1]
}

// ParseFiendishLegacy returns the legacy printed as name (exact, case sensitive)
// and true, or the zero value and false. It never panics.
func ParseFiendishLegacy(name string) (FiendishLegacy, bool) {
	l := FiendishLegacy(enumByName(name, legacyNames()))
	return l, l.Valid()
}
