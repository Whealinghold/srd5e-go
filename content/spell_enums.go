package content

// The small closed vocabularies of the spell shape (content-spells.md FR-012,
// FR-119). Each is numbered from 1 in declaration order, 0 is invalid, and the
// wire spelling of member v is names[v-1], as in enum.go. Each is comparable
// and safe for concurrent use.

// SpellSchool is one of the eight schools of magic (SRD 5.2.1, "Schools of
// Magic", p. 105). The zero SpellSchool is invalid.
type SpellSchool int

// The schools of magic, in table order.
const (
	SchoolAbjuration SpellSchool = iota + 1
	SchoolConjuration
	SchoolDivination
	SchoolEnchantment
	SchoolEvocation
	SchoolIllusion
	SchoolNecromancy
	SchoolTransmutation
)

func spellSchoolNames() []string {
	return []string{
		"Abjuration", "Conjuration", "Divination", "Enchantment", "Evocation",
		"Illusion", "Necromancy", "Transmutation",
	}
}

// Valid reports whether s is one of the eight schools.
func (s SpellSchool) Valid() bool { return enumValid(spellSchoolNames(), int(s)) }

// String returns the printed name of s, or "unknown" for an undefined value.
func (s SpellSchool) String() string { return enumString(spellSchoolNames(), int(s)) }

// ParseSpellSchool returns the school spelled exactly s, such as "Evocation".
// Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseSpellSchool(s string) (SpellSchool, error) {
	v, err := parseEnum("SpellSchool", spellSchoolNames(), s)
	return SpellSchool(v), err
}

// DistanceUnit is the unit of a spell range distance: feet or miles (SRD
// 5.2.1, "Range", p. 105). The zero DistanceUnit is invalid.
type DistanceUnit int

// The distance units.
const (
	// DistanceFeet is a number of feet.
	DistanceFeet DistanceUnit = iota + 1
	// DistanceMiles is a number of miles.
	DistanceMiles
)

func distanceUnitNames() []string { return []string{"feet", "miles"} }

// Valid reports whether u is feet or miles.
func (u DistanceUnit) Valid() bool { return enumValid(distanceUnitNames(), int(u)) }

// String returns the wire spelling of u, or "unknown" for an undefined value.
func (u DistanceUnit) String() string { return enumString(distanceUnitNames(), int(u)) }

// ParseDistanceUnit returns the unit spelled exactly s: "feet" or "miles".
// Matching is case-sensitive.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseDistanceUnit(s string) (DistanceUnit, error) {
	v, err := parseEnum("DistanceUnit", distanceUnitNames(), s)
	return DistanceUnit(v), err
}

// DurationLimit says how a timed spell duration is bounded (SRD 5.2.1,
// "Duration", p. 106): exactly, up to, or up to while concentrating, so
// Concentration without a cap cannot be expressed. The zero DurationLimit is
// invalid.
type DurationLimit int

// The duration limits.
const (
	// LimitFixed is a duration of exactly the stated time ("1 hour").
	LimitFixed DurationLimit = iota + 1
	// LimitUpTo is a duration of up to the stated time without Concentration
	// ("Up to 1 hour").
	LimitUpTo
	// LimitConcentrationUpTo is Concentration for up to the stated time.
	LimitConcentrationUpTo
)

func durationLimitNames() []string { return []string{"fixed", "upTo", "concentrationUpTo"} }

// Valid reports whether l is one of the three limits.
func (l DurationLimit) Valid() bool { return enumValid(durationLimitNames(), int(l)) }

// String returns the wire spelling of l, or "unknown" for an undefined value.
func (l DurationLimit) String() string { return enumString(durationLimitNames(), int(l)) }

// ParseDurationLimit returns the limit spelled exactly s: "fixed", "upTo" or
// "concentrationUpTo". Matching is case-sensitive.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseDurationLimit(s string) (DurationLimit, error) {
	v, err := parseEnum("DurationLimit", durationLimitNames(), s)
	return DurationLimit(v), err
}
