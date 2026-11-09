package character

// TravelPace is the pace a group travels at: Slow, Normal or Fast (SRD 5.2.1,
// "Travel Pace", p. 12). The values are ordered by speed, slowest first, so
// PaceSlow < PaceNormal < PaceFast; the SRD prints the table Fast first, which
// is presentation. The zero TravelPace is invalid; see Valid. Comparable and
// safe for concurrent use.
type TravelPace int

// The travel paces.
const (
	PaceSlow TravelPace = iota + 1
	PaceNormal
	PaceFast
)

// TravelPaces returns the three paces, slowest first, in a new slice.
func TravelPaces() []TravelPace { return []TravelPace{PaceSlow, PaceNormal, PaceFast} }

// Valid reports whether p is Slow, Normal or Fast.
func (p TravelPace) Valid() bool { return p >= PaceSlow && p <= PaceFast }

// String returns "slow", "normal" or "fast", or "unknown" for an undefined value.
func (p TravelPace) String() string {
	switch p {
	case PaceSlow:
		return "slow"
	case PaceNormal:
		return "normal"
	case PaceFast:
		return "fast"
	}
	return "unknown"
}

// RaisedOneStep returns the next faster pace: Slow gives Normal, Normal gives
// Fast and Fast stays Fast (SRD 5.2.1, "Good Roads", p. 192). An invalid pace
// is returned unchanged.
func (p TravelPace) RaisedOneStep() TravelPace {
	if p == PaceSlow || p == PaceNormal {
		return p + 1
	}
	return p
}

// TravelPaceRow is one row of the Travel Pace table: the distance travelled per
// minute, hour and day (SRD 5.2.1, "Travel Pace", p. 12). The figures are the
// printed ones, held as data, and are not multiples of one another. The zero
// TravelPaceRow (the row of an invalid pace) holds zeros. Comparable and safe for
// concurrent use.
type TravelPaceRow struct{ feetPerMinute, milesPerHour, milesPerDay int }

// Row returns the table row of p, or the zero TravelPaceRow for an invalid pace.
func (p TravelPace) Row() TravelPaceRow {
	switch p {
	case PaceFast:
		return TravelPaceRow{400, 4, 30}
	case PaceNormal:
		return TravelPaceRow{300, 3, 24}
	case PaceSlow:
		return TravelPaceRow{200, 2, 18}
	}
	return TravelPaceRow{}
}

// FeetPerMinute returns the feet travelled per minute.
func (r TravelPaceRow) FeetPerMinute() int { return r.feetPerMinute }

// MilesPerHour returns the miles travelled per hour.
func (r TravelPaceRow) MilesPerHour() int { return r.milesPerHour }

// MilesPerDay returns the miles travelled per day.
func (r TravelPaceRow) MilesPerDay() int { return r.milesPerDay }
