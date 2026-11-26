package combat

import (
	"math"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellTimeRemaining is the time left on an ongoing spell: a whole number of
// seconds, or open-ended when the spell lasts until it is dispelled or
// triggered (SRD 5.2.1, "Duration", p. 106). A round is 6 seconds. Build it
// with NewSpellTimeRemaining, SpellTimeOpenEnded or SpellTimeRemainingFromDuration;
// the zero value is invalid. Comparable and safe for concurrent use.
type SpellTimeRemaining struct {
	seconds int
	open    bool
}

// secondsPerRound is the length of a round in seconds (SRD 5.2.1, "Rounds", p. 13).
const secondsPerRound = 6

// NewSpellTimeRemaining returns a timed remaining duration of seconds. It
// returns an error matching srderr.ErrInvalidInput when seconds is below 1.
// It never panics.
func NewSpellTimeRemaining(seconds int) (SpellTimeRemaining, error) {
	if seconds < 1 {
		return SpellTimeRemaining{}, invalidInput("spell time remaining needs at least one second")
	}
	return SpellTimeRemaining{seconds: seconds}, nil
}

// SpellTimeOpenEnded returns the remaining time of a spell that has no number:
// it lasts until it is dispelled, dispelled or triggered, or by special rule.
func SpellTimeOpenEnded() SpellTimeRemaining { return SpellTimeRemaining{open: true} }

// SpellTimeRemainingFromDuration converts a spell duration into the time it
// lasts. A timed duration gives its amount in seconds (rounds x 6, minutes x 60,
// hours x 3600, days x 86400); for upTo and concentrationUpTo the amount is the
// maximum. Until dispelled, until dispelled or triggered and special give
// SpellTimeOpenEnded. It returns an error matching srderr.ErrInvalidInput for an
// instantaneous duration (the effect happens once and is never ongoing), for an
// amount that overflows int when converted, and for a duration the package does
// not know. It never panics.
func SpellTimeRemainingFromDuration(d content.SpellDuration) (SpellTimeRemaining, error) {
	switch v := d.(type) {
	case content.SpellDurationTimed:
		factor, ok := secondsPerUnit(v.Unit())
		if !ok {
			return SpellTimeRemaining{}, invalidInput("spell duration needs a known unit")
		}
		if v.Amount() < 1 || v.Amount() > math.MaxInt/factor {
			return SpellTimeRemaining{}, invalidInput("spell duration amount is out of range")
		}
		return NewSpellTimeRemaining(v.Amount() * factor)
	case content.SpellDurationUntilDispelled, content.SpellDurationUntilDispelledOrTriggered, content.SpellDurationSpecial:
		return SpellTimeOpenEnded(), nil
	case content.SpellDurationInstantaneous:
		return SpellTimeRemaining{}, invalidInput("an instantaneous spell is never an ongoing spell")
	default:
		return SpellTimeRemaining{}, invalidInput("spell duration is not a known variant")
	}
}

// secondsPerUnit returns the seconds in one unit of a timed spell duration.
func secondsPerUnit(u content.DurationUnit) (int, bool) {
	switch u {
	case content.DurationUnitRounds:
		return secondsPerRound, true
	case content.DurationUnitMinutes:
		return 60, true
	case content.DurationUnitHours:
		return 3600, true
	case content.DurationUnitDays:
		return 86400, true
	default:
		return 0, false
	}
}

// IsOpenEnded reports whether the spell has no number of seconds left.
func (r SpellTimeRemaining) IsOpenEnded() bool { return r.open }

// Seconds returns the seconds left and true for a timed remaining duration, and
// 0 and false for an open-ended one or the zero value.
func (r SpellTimeRemaining) Seconds() (int, bool) {
	if r.open || r.seconds == 0 {
		return 0, false
	}
	return r.seconds, true
}

// IsZero reports whether r is the invalid zero value.
func (r SpellTimeRemaining) IsZero() bool { return r == SpellTimeRemaining{} }
