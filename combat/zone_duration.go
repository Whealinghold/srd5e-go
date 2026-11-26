package combat

import (
	"math"

	"github.com/Whealinghold/srd5e-go/content"
)

// Rounds per unit of a spell's Time Span (SRD 5.2.1, "Combat Step by Step",
// p. 13: a round is about 6 seconds; a minute is 10 rounds, an hour 600).
const (
	roundsPerMinute = 10
	roundsPerHour   = 600
)

// ZoneDuration is how long a persistent zone lasts. Variants: RoundsDuration,
// HostTimedDuration, UntilDispelled. A nil ZoneDuration in a command means the
// spell's own duration applies.
type ZoneDuration interface{ isZoneDuration() }

// RoundsDuration lasts a number of rounds counted from the cast round. The zero
// value is invalid; build it with [NewRoundsDuration].
type RoundsDuration struct {
	rounds int
}

// NewRoundsDuration returns a duration of rounds rounds.
//
// It returns an *ZoneInvalidError for fewer than 1 round. It never panics.
func NewRoundsDuration(rounds int) (RoundsDuration, error) {
	if rounds < 1 {
		return RoundsDuration{}, &ZoneInvalidError{What: "rounds duration"}
	}
	return RoundsDuration{rounds: rounds}, nil
}

// Rounds returns the number of rounds.
func (d RoundsDuration) Rounds() int { return d.rounds }

func (RoundsDuration) isZoneDuration() {}

// HostTimedDuration is a duration longer than the rounds model (days): the host
// ends the zone.
type HostTimedDuration struct{}

func (HostTimedDuration) isZoneDuration() {}

// UntilDispelled lasts until the host ends it.
type UntilDispelled struct{}

func (UntilDispelled) isZoneDuration() {}

// ZoneDurationConversion is the outcome of [ZoneDurationFromSpellDuration].
// Variants: ZoneDurationConverted, ZoneDurationRefused.
type ZoneDurationConversion interface{ isZoneDurationConversion() }

// ZoneDurationConverted is a spell duration that a zone can carry.
type ZoneDurationConverted struct {
	Duration ZoneDuration
}

// ZoneDurationRefused is a spell duration a zone cannot carry: Instantaneous or
// Special (the DurationNotZone refusal).
type ZoneDurationRefused struct{}

func (ZoneDurationConverted) isZoneDurationConversion() {}
func (ZoneDurationRefused) isZoneDurationConversion()   {}

// ZoneDurationFromSpellDuration converts a spell's duration to a zone duration
// (persistent-spell-zones FR-009): a timed duration in rounds, minutes or hours
// becomes rounds (1, 10 and 600 per unit); a longer timed duration becomes
// HostTimedDuration; "until dispelled" and "until dispelled or triggered" become
// UntilDispelled; an Instantaneous or Special duration is refused. A timed
// duration that gives fewer than 1 round is refused too. It never panics.
func ZoneDurationFromSpellDuration(d content.SpellDuration) ZoneDurationConversion {
	switch v := d.(type) {
	case content.SpellDurationTimed:
		return timedZoneDuration(v)
	case content.SpellDurationUntilDispelled, content.SpellDurationUntilDispelledOrTriggered:
		return ZoneDurationConverted{Duration: UntilDispelled{}}
	}
	return ZoneDurationRefused{}
}

// timedZoneDuration converts a timed spell duration, see ZoneDurationFromSpellDuration.
// A round count past int is beyond what the zone counts, so the host times it.
func timedZoneDuration(d content.SpellDurationTimed) ZoneDurationConversion {
	var rounds int
	var ok bool
	switch d.Unit() {
	case content.DurationUnitRounds:
		rounds, ok = d.Amount(), true
	case content.DurationUnitMinutes:
		rounds, ok = roundsIn(d.Amount(), roundsPerMinute)
	case content.DurationUnitHours:
		rounds, ok = roundsIn(d.Amount(), roundsPerHour)
	case content.DurationUnitDays:
		return ZoneDurationConverted{Duration: HostTimedDuration{}}
	}
	if !ok {
		return ZoneDurationConverted{Duration: HostTimedDuration{}}
	}
	r, err := NewRoundsDuration(rounds)
	if err != nil {
		return ZoneDurationRefused{}
	}
	return ZoneDurationConverted{Duration: r}
}

// roundsIn returns amount units of perUnit rounds each. It reports false when the
// product is past int.
func roundsIn(amount, perUnit int) (int, bool) {
	if amount > math.MaxInt/perUnit {
		return 0, false
	}
	return amount * perUnit, true
}
