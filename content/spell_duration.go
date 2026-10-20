package content

// MinSpellDurationAmount is the least amount of a timed spell duration
// (content-spells.md FR-051, FR-111).
const MinSpellDurationAmount = 1

// ValidSpellDurationAmount reports whether n is an amount a timed spell
// duration may have, at least [MinSpellDurationAmount].
func ValidSpellDurationAmount(n int) bool { return n >= MinSpellDurationAmount }

// SpellDuration is the closed set of spell durations of SRD 5.2.1 ("Duration",
// p. 106): [SpellDurationInstantaneous], [SpellDurationTimed],
// [SpellDurationUntilDispelled], [SpellDurationUntilDispelledOrTriggered] and
// [SpellDurationSpecial]. Every variant is an immutable, comparable value and
// safe for concurrent use. Consumers switch over it exhaustively.
type SpellDuration interface{ isSpellDuration() }

// SpellDurationInstantaneous is a duration of Instantaneous. The zero value is
// valid.
type SpellDurationInstantaneous struct{}

// NewSpellDurationInstantaneous returns the Instantaneous duration.
func NewSpellDurationInstantaneous() SpellDurationInstantaneous {
	return SpellDurationInstantaneous{}
}

func (SpellDurationInstantaneous) isSpellDuration() {}

// SpellDurationTimed is a Time Span of an amount of rounds, minutes, hours or
// days, bounded as its [DurationLimit] says. The zero value is invalid; build
// it with [NewSpellDurationTimed].
type SpellDurationTimed struct {
	amount int
	unit   DurationUnit
	limit  DurationLimit
}

// NewSpellDurationTimed returns the Time Span of amount units under limit.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an amount below [MinSpellDurationAmount] and
// BlockInvalidMember for an invalid unit or limit. It never panics.
func NewSpellDurationTimed(amount int, unit DurationUnit, limit DurationLimit) (SpellDurationTimed, error) {
	if !ValidSpellDurationAmount(amount) {
		return SpellDurationTimed{}, blockError("SpellDurationTimed", BlockOutOfRange)
	}
	if !unit.Valid() || !limit.Valid() {
		return SpellDurationTimed{}, blockError("SpellDurationTimed", BlockInvalidMember)
	}
	return SpellDurationTimed{amount: amount, unit: unit, limit: limit}, nil
}

// Amount returns the number of units.
func (d SpellDurationTimed) Amount() int { return d.amount }

// Unit returns the unit.
func (d SpellDurationTimed) Unit() DurationUnit { return d.unit }

// Limit returns how the Time Span is bounded.
func (d SpellDurationTimed) Limit() DurationLimit { return d.limit }

func (SpellDurationTimed) isSpellDuration() {}

// SpellDurationUntilDispelled is a duration of Until dispelled. The zero value
// is valid.
type SpellDurationUntilDispelled struct{}

// NewSpellDurationUntilDispelled returns the Until dispelled duration.
func NewSpellDurationUntilDispelled() SpellDurationUntilDispelled {
	return SpellDurationUntilDispelled{}
}

func (SpellDurationUntilDispelled) isSpellDuration() {}

// SpellDurationUntilDispelledOrTriggered is a duration of Until dispelled or
// triggered. The zero value is valid.
type SpellDurationUntilDispelledOrTriggered struct{}

// NewSpellDurationUntilDispelledOrTriggered returns the Until dispelled or
// triggered duration.
func NewSpellDurationUntilDispelledOrTriggered() SpellDurationUntilDispelledOrTriggered {
	return SpellDurationUntilDispelledOrTriggered{}
}

func (SpellDurationUntilDispelledOrTriggered) isSpellDuration() {}

// SpellDurationSpecial is a duration the description explains (SRD 5.2.1,
// "Creation", p. 121). The zero value is valid.
type SpellDurationSpecial struct{}

// NewSpellDurationSpecial returns the Special duration.
func NewSpellDurationSpecial() SpellDurationSpecial { return SpellDurationSpecial{} }

func (SpellDurationSpecial) isSpellDuration() {}

// checkSpellDuration reports why d is not a duration its constructor would
// accept: BlockMissing for nil and for a pointer variant.
func checkSpellDuration(d SpellDuration) error {
	switch v := d.(type) {
	case SpellDurationInstantaneous, SpellDurationUntilDispelled, SpellDurationUntilDispelledOrTriggered, SpellDurationSpecial:
		return nil
	case SpellDurationTimed:
		_, err := NewSpellDurationTimed(v.amount, v.unit, v.limit)
		return err
	}
	return blockError("SpellDuration", BlockMissing)
}
