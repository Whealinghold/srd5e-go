package content

// DurationUnit is a unit of a Time Span in a spell Duration: rounds, minutes,
// hours or days (SRD 5.2.1, "Duration", p. 106). The type carries no fact and
// no conversion to a time.Duration. The zero DurationUnit is invalid. Safe for
// concurrent use.
type DurationUnit int

// The duration units, in ascending order. Weeks, months and years are not
// members.
const (
	// DurationUnitRounds is a number of rounds.
	DurationUnitRounds DurationUnit = iota + 1
	// DurationUnitMinutes is a number of minutes.
	DurationUnitMinutes
	// DurationUnitHours is a number of hours.
	DurationUnitHours
	// DurationUnitDays is a number of days.
	DurationUnitDays
)

// Valid reports whether u is one of the four units.
func (u DurationUnit) Valid() bool { return enumValid(durationUnitNames(), int(u)) }

// String returns the wire spelling of u (a package tag, lower case), or
// "unknown" for an undefined value.
func (u DurationUnit) String() string { return enumString(durationUnitNames(), int(u)) }

func durationUnitNames() []string { return []string{"rounds", "minutes", "hours", "days"} }

// ParseDurationUnit returns the unit spelled exactly s: "rounds", "minutes",
// "hours" or "days". Matching is case-sensitive.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string, including "Rounds", "hour" and "years". It never panics.
func ParseDurationUnit(s string) (DurationUnit, error) {
	v, err := parseEnum("DurationUnit", durationUnitNames(), s)
	return DurationUnit(v), err
}
