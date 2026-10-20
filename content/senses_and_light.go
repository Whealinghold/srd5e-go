package content

// LightLevel is the level of light in an area, in SRD order (SRD 5.2.1,
// "Light", p. 11). Identity only: what each level does is another context's
// rule. The zero LightLevel is invalid. Comparable and safe for concurrent
// use.
type LightLevel int

// The light levels.
const (
	LightBright LightLevel = iota + 1
	LightDim
	LightDarkness
)

func lightLevelNames() []string { return []string{"Bright Light", "Dim Light", "Darkness"} }

// Valid reports whether l is one of the three light levels.
func (l LightLevel) Valid() bool { return enumValid(lightLevelNames(), int(l)) }

// String returns the printed name of l, or "unknown" for an undefined value.
func (l LightLevel) String() string { return enumString(lightLevelNames(), int(l)) }

// ParseLightLevel returns the light level spelled exactly s, such as
// "Dim Light". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseLightLevel(s string) (LightLevel, error) {
	v, err := parseEnum("LightLevel", lightLevelNames(), s)
	return LightLevel(v), err
}

// DarknessKind qualifies Darkness as nonmagical or magical (SRD 5.2.1,
// "Light", "Special Senses", p. 11; "Truesight", p. 190). Nonmagical is a
// package name, not an SRD term. The zero DarknessKind is invalid.
// Comparable and safe for concurrent use.
type DarknessKind int

// The kinds of Darkness.
const (
	DarknessNonmagical DarknessKind = iota + 1
	DarknessMagical
)

func darknessKindNames() []string { return []string{"Nonmagical", "Magical"} }

// Valid reports whether k is one of the two kinds.
func (k DarknessKind) Valid() bool { return enumValid(darknessKindNames(), int(k)) }

// String returns the wire spelling of k, or "unknown" for an undefined value.
func (k DarknessKind) String() string { return enumString(darknessKindNames(), int(k)) }

// ParseDarknessKind returns the kind spelled exactly s: "Nonmagical" or
// "Magical". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseDarknessKind(s string) (DarknessKind, error) {
	v, err := parseEnum("DarknessKind", darknessKindNames(), s)
	return DarknessKind(v), err
}

// Obscurement is how much an area hides what is in it, ordered by severity so
// the constants compare with < (SRD 5.2.1, "Obscured Areas", p. 11). None is a
// package name for "not obscured", not an SRD term. The zero Obscurement is
// invalid. Comparable and safe for concurrent use.
type Obscurement int

// The levels of obscurement, least to most severe.
const (
	ObscurementNone Obscurement = iota + 1
	ObscurementLightly
	ObscurementHeavily
)

func obscurementNames() []string {
	return []string{"None", "Lightly Obscured", "Heavily Obscured"}
}

// Valid reports whether o is one of the three levels.
func (o Obscurement) Valid() bool { return enumValid(obscurementNames(), int(o)) }

// String returns the wire spelling of o, or "unknown" for an undefined value.
func (o Obscurement) String() string { return enumString(obscurementNames(), int(o)) }

// ParseObscurement returns the level spelled exactly s, such as "Heavily
// Obscured". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseObscurement(s string) (Obscurement, error) {
	v, err := parseEnum("Obscurement", obscurementNames(), s)
	return Obscurement(v), err
}

// SpecialSense is one of the four special senses, in SRD order (SRD 5.2.1,
// "Special Senses", p. 11). Identity only: ranges and sight rules belong to
// other contexts. The zero SpecialSense is invalid. Comparable and safe for
// concurrent use.
type SpecialSense int

// The special senses.
const (
	SenseBlindsight SpecialSense = iota + 1
	SenseDarkvision
	SenseTremorsense
	SenseTruesight
)

func specialSenseNames() []string {
	return []string{"Blindsight", "Darkvision", "Tremorsense", "Truesight"}
}

// Valid reports whether s is one of the four senses.
func (s SpecialSense) Valid() bool { return enumValid(specialSenseNames(), int(s)) }

// String returns the printed name of s, or "unknown" for an undefined value.
func (s SpecialSense) String() string { return enumString(specialSenseNames(), int(s)) }

// ParseSpecialSense returns the sense spelled exactly s, such as
// "Darkvision". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseSpecialSense(s string) (SpecialSense, error) {
	v, err := parseEnum("SpecialSense", specialSenseNames(), s)
	return SpecialSense(v), err
}
