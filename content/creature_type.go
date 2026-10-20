package content

// CreatureType is one of the 14 creature types, in Glossary order (SRD 5.2.1,
// "Creature Type", p. 179). The zero CreatureType is invalid. Comparable and
// safe for concurrent use.
type CreatureType int

// The creature types.
const (
	CreatureTypeAberration CreatureType = iota + 1
	CreatureTypeBeast
	CreatureTypeCelestial
	CreatureTypeConstruct
	CreatureTypeDragon
	CreatureTypeElemental
	CreatureTypeFey
	CreatureTypeFiend
	CreatureTypeGiant
	CreatureTypeHumanoid
	CreatureTypeMonstrosity
	CreatureTypeOoze
	CreatureTypePlant
	CreatureTypeUndead
)

func creatureTypeNames() []string {
	return []string{
		"Aberration", "Beast", "Celestial", "Construct", "Dragon", "Elemental", "Fey",
		"Fiend", "Giant", "Humanoid", "Monstrosity", "Ooze", "Plant", "Undead",
	}
}

// Valid reports whether t is one of the 14 creature types.
func (t CreatureType) Valid() bool { return enumValid(creatureTypeNames(), int(t)) }

// String returns the printed name of t, or "unknown" for an undefined value.
func (t CreatureType) String() string { return enumString(creatureTypeNames(), int(t)) }

// ParseCreatureType returns the creature type spelled exactly s, such as
// "Monstrosity". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseCreatureType(s string) (CreatureType, error) {
	v, err := parseEnum("CreatureType", creatureTypeNames(), s)
	return CreatureType(v), err
}
