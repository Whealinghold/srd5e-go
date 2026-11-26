package combat

import (
	"math"

	"github.com/Whealinghold/srd5e-go/content"
)

// EffectDuration is how long an effect, or one stage of it, lasts. Variants: EffectLastsRounds,
// EffectLastsUntilTurnBoundary, EffectLastsIndefinitely, EffectLastsRemainder.
type EffectDuration interface{ isEffectDuration() }

// EffectLastsRounds lasts a number of rounds (at least 1). Build it with NewEffectLastsRounds; the
// zero value is invalid.
type EffectLastsRounds struct{ rounds int }

// NewEffectLastsRounds returns a duration of rounds rounds. It returns an error matching
// srderr.ErrInvalidInput for rounds below 1. It never panics.
func NewEffectLastsRounds(rounds int) (EffectLastsRounds, error) {
	if rounds < 1 {
		return EffectLastsRounds{}, invalidInput("an effect lasts at least 1 round")
	}
	return EffectLastsRounds{rounds}, nil
}

// Rounds returns the number of rounds.
func (d EffectLastsRounds) Rounds() int { return d.rounds }

// EffectLastsUntilTurnBoundary lasts until the start or end of a turn of its owner (SRD 5.2.1,
// "Ray of Sickness", p. 158). Build it with NewEffectLastsUntilTurnBoundary; the zero value is
// invalid.
type EffectLastsUntilTurnBoundary struct {
	owner    CombatantID
	boundary content.TurnBoundary
}

// NewEffectLastsUntilTurnBoundary returns the duration up to the boundary of a turn of owner. It
// returns an error matching srderr.ErrInvalidInput for a zero owner or an invalid boundary. It
// never panics.
func NewEffectLastsUntilTurnBoundary(owner CombatantID, boundary content.TurnBoundary) (EffectLastsUntilTurnBoundary, error) {
	if owner.IsZero() || !boundary.Valid() {
		return EffectLastsUntilTurnBoundary{}, invalidInput("a turn boundary duration needs an owner and a boundary")
	}
	return EffectLastsUntilTurnBoundary{owner, boundary}, nil
}

// OwnerIs reports whether the turn is the turn of id.
func (d EffectLastsUntilTurnBoundary) OwnerIs(id CombatantID) bool { return d.owner == id }

// Boundary returns the start or the end of the turn.
func (d EffectLastsUntilTurnBoundary) Boundary() content.TurnBoundary { return d.boundary }

// EffectLastsIndefinitely never expires by a tick (until dispelled). The zero value is valid.
type EffectLastsIndefinitely struct{}

// EffectLastsRemainder is the duration of a stage that lasts as long as the effect's overall clock
// (FR-061). It is valid only as the duration of a template stage. The zero value is valid.
type EffectLastsRemainder struct{}

func (EffectLastsRounds) isEffectDuration()            {}
func (EffectLastsUntilTurnBoundary) isEffectDuration() {}
func (EffectLastsIndefinitely) isEffectDuration()      {}
func (EffectLastsRemainder) isEffectDuration()         {}

// validEffectDuration reports whether d is a variant its constructor would accept.
func validEffectDuration(d EffectDuration) bool {
	switch v := d.(type) {
	case EffectLastsRounds:
		return v.rounds >= 1
	case EffectLastsUntilTurnBoundary:
		return !v.owner.IsZero() && v.boundary.Valid()
	case EffectLastsIndefinitely, EffectLastsRemainder:
		return true
	}
	return false
}

// RoundsInTimeSpan converts amount units of a time span to rounds: 1 per round, 10 per minute, 600
// per hour and 14,400 per day, from a round being about 6 seconds (SRD 5.2.1, "The Order of
// Combat", p. 13). The SRD says "about", so the arithmetic is an interpretation (FR-006). It
// returns an error matching srderr.ErrInvalidInput for an amount below 1, an invalid unit or a
// result that does not fit an int. It never panics.
func RoundsInTimeSpan(amount int, unit content.DurationUnit) (int, error) {
	var per int
	switch unit {
	case content.DurationUnitRounds:
		per = 1
	case content.DurationUnitMinutes:
		per = 10
	case content.DurationUnitHours:
		per = 600
	case content.DurationUnitDays:
		per = 14400
	default:
		return 0, invalidInput("unknown duration unit")
	}
	if amount < 1 || amount > math.MaxInt/per {
		return 0, invalidInput("time span out of range")
	}
	return amount * per, nil
}

// EffectDurationFromSpell maps a spell duration to an effect duration (table D): Timed to
// EffectLastsRounds, UntilDispelled, UntilDispelledOrTriggered and Special to
// EffectLastsIndefinitely. Instantaneous has no lasting effect (FR-010). It returns an error
// matching srderr.ErrInvalidInput for Instantaneous, a nil or unknown duration, or a span out of
// range. It never panics.
func EffectDurationFromSpell(d content.SpellDuration) (EffectDuration, error) {
	switch v := d.(type) {
	case content.SpellDurationTimed:
		n, err := RoundsInTimeSpan(v.Amount(), v.Unit())
		if err != nil {
			return nil, err
		}
		return EffectLastsRounds{n}, nil
	case content.SpellDurationUntilDispelled, content.SpellDurationUntilDispelledOrTriggered, content.SpellDurationSpecial:
		return EffectLastsIndefinitely{}, nil
	}
	return nil, invalidInput("a spell duration with no lasting effect")
}

// EffectDurationFromCondition maps the duration of a condition outcome to an effect duration
// (FR-011): ForSpellDuration to EffectDurationFromSpell of spellDuration, TimedConditionDuration
// to rounds, UntilNextTurnBoundary to EffectLastsUntilTurnBoundary of caster or target. It returns
// an error matching srderr.ErrInvalidInput for an unknown duration, a zero owner or a span out of
// range. It never panics.
func EffectDurationFromCondition(d content.ConditionDuration, spellDuration content.SpellDuration, caster, target CombatantID) (EffectDuration, error) {
	switch v := d.(type) {
	case content.ForSpellDuration:
		return EffectDurationFromSpell(spellDuration)
	case content.TimedConditionDuration:
		n, err := RoundsInTimeSpan(v.Amount(), v.Unit())
		if err != nil {
			return nil, err
		}
		return EffectLastsRounds{n}, nil
	case content.UntilNextTurnBoundary:
		owner := caster
		if v.Owner() == content.OwnerTarget {
			owner = target
		}
		return NewEffectLastsUntilTurnBoundary(owner, v.Boundary())
	}
	return nil, invalidInput("unknown condition duration")
}
