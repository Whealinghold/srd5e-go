package content

// WeaponProperty is one of the nine weapon properties, in the order of the
// Properties section (SRD 5.2.1, "Properties", pp. 89-90). "Range" is a
// notation qualifying Ammunition and Thrown, not a property. Identity only;
// behaviour belongs to the context that applies the rule. The zero
// WeaponProperty is invalid. Comparable and safe for concurrent use.
type WeaponProperty int

// The weapon properties.
const (
	WeaponPropertyAmmunition WeaponProperty = iota + 1
	WeaponPropertyFinesse
	WeaponPropertyHeavy
	WeaponPropertyLight
	WeaponPropertyLoading
	WeaponPropertyReach
	WeaponPropertyThrown
	WeaponPropertyTwoHanded
	WeaponPropertyVersatile
)

func weaponPropertyNames() []string {
	return []string{
		"Ammunition", "Finesse", "Heavy", "Light", "Loading", "Reach", "Thrown",
		"Two-Handed", "Versatile",
	}
}

// Valid reports whether p is one of the nine properties.
func (p WeaponProperty) Valid() bool { return enumValid(weaponPropertyNames(), int(p)) }

// String returns the printed name of p, or "unknown" for an undefined value.
func (p WeaponProperty) String() string { return enumString(weaponPropertyNames(), int(p)) }

// ParseWeaponProperty returns the property spelled exactly s, such as
// "Two-Handed". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "Range". It never panics.
func ParseWeaponProperty(s string) (WeaponProperty, error) {
	v, err := parseEnum("WeaponProperty", weaponPropertyNames(), s)
	return WeaponProperty(v), err
}

// MasteryProperty is one of the eight weapon mastery properties, in order
// (SRD 5.2.1, "Mastery Properties", p. 90). Identity only. The zero
// MasteryProperty is invalid. Comparable and safe for concurrent use.
type MasteryProperty int

// The mastery properties.
const (
	MasteryCleave MasteryProperty = iota + 1
	MasteryGraze
	MasteryNick
	MasteryPush
	MasterySap
	MasterySlow
	MasteryTopple
	MasteryVex
)

func masteryPropertyNames() []string {
	return []string{"Cleave", "Graze", "Nick", "Push", "Sap", "Slow", "Topple", "Vex"}
}

// Valid reports whether m is one of the eight mastery properties.
func (m MasteryProperty) Valid() bool { return enumValid(masteryPropertyNames(), int(m)) }

// String returns the printed name of m, or "unknown" for an undefined value.
func (m MasteryProperty) String() string { return enumString(masteryPropertyNames(), int(m)) }

// ParseMasteryProperty returns the mastery property spelled exactly s, such as
// "Cleave". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseMasteryProperty(s string) (MasteryProperty, error) {
	v, err := parseEnum("MasteryProperty", masteryPropertyNames(), s)
	return MasteryProperty(v), err
}
