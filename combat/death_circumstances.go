package combat

import (
	"time"
)

// BodyState is what remains of a Dead creature's body (SRD 5.2.1, "Raise Dead",
// p. 157; "True Resurrection", p. 170). The zero value is Intact.
type BodyState int

// The states of a body.
const (
	// Intact: the body is whole.
	Intact BodyState = iota
	// MissingNonIntegralParts: parts are missing, none integral for survival.
	MissingNonIntegralParts
	// MissingIntegralParts: body parts or organs integral for survival are missing.
	MissingIntegralParts
	// NoBody: the original body no longer exists.
	NoBody
)

// Valid reports whether b is one of the four states.
func (b BodyState) Valid() bool { return b >= Intact && b <= NoBody }

// String returns "intact", "missing_non_integral_parts", "missing_integral_parts",
// "no_body" or "unknown".
func (b BodyState) String() string {
	if b < Intact {
		return "unknown"
	}
	return enumName([]string{"intact", "missing_non_integral_parts", "missing_integral_parts", "no_body"}, int(b)+1)
}

// DeathCircumstances are the host's facts about a death that a revival spell reads.
// Build it with NewDeathCircumstances. The zero value is valid: died this instant,
// intact, not Undead, not of old age. Immutable, comparable and safe for concurrent use.
type DeathCircumstances struct {
	timeSinceDeath time.Duration
	diedOfOldAge   bool
	wasUndead      bool
	body           BodyState
}

// NewDeathCircumstances returns the circumstances of a death.
//
// It returns a *RestorationRangeError (matching srderr.ErrInvalidInput) for a
// negative timeSinceDeath or a body that is not one of the four states, and the
// zero DeathCircumstances with it. Zero is valid: the creature died this instant.
// It never panics.
func NewDeathCircumstances(timeSinceDeath time.Duration, diedOfOldAge, wasUndead bool, body BodyState) (DeathCircumstances, error) {
	if timeSinceDeath < 0 {
		return DeathCircumstances{}, rangeError("time_since_death", int64(timeSinceDeath))
	}
	if !body.Valid() {
		return DeathCircumstances{}, rangeError("body_state", int64(body))
	}
	return DeathCircumstances{timeSinceDeath, diedOfOldAge, wasUndead, body}, nil
}

// MustDeathCircumstances is NewDeathCircumstances for known-good values. It panics
// with the error NewDeathCircumstances returns, and only then.
func MustDeathCircumstances(timeSinceDeath time.Duration, diedOfOldAge, wasUndead bool, body BodyState) DeathCircumstances {
	c, err := NewDeathCircumstances(timeSinceDeath, diedOfOldAge, wasUndead, body)
	if err != nil {
		panic(err)
	}
	return c
}

// TimeSinceDeath returns how long the creature has been dead.
func (c DeathCircumstances) TimeSinceDeath() time.Duration { return c.timeSinceDeath }

// DiedOfOldAge reports whether the creature died of old age.
func (c DeathCircumstances) DiedOfOldAge() bool { return c.diedOfOldAge }

// WasUndead reports whether the creature was Undead when it died.
func (c DeathCircumstances) WasUndead() bool { return c.wasUndead }

// Body returns what remains of the body.
func (c DeathCircumstances) Body() BodyState { return c.body }

// RevivalIneligibility is one reason a revival spell cannot work on a creature.
// The zero value is not a reason.
type RevivalIneligibility int

// The reasons, in the fixed order ReviveBySpell lists them.
const (
	// TimeWindowExceeded: the creature has been dead longer than the spell allows.
	TimeWindowExceeded RevivalIneligibility = iota + 1
	// DiedOfOldAge: the spell cannot return a creature that died of old age.
	DiedOfOldAge
	// WasUndead: the spell cannot return a creature that was Undead.
	WasUndead
	// NoBodyToTouch: the spell is cast by touch and no body remains.
	NoBodyToTouch
	// IntegralPartsMissing: Raise Dead fails when integral body parts are missing.
	IntegralPartsMissing
	// NameNotSpoken: True Resurrection needs the creature's name for a new body.
	NameNotSpoken
)

// String returns the snake_case name of r, or "unknown".
func (r RevivalIneligibility) String() string {
	return enumName([]string{"time_window_exceeded", "died_of_old_age", "was_undead", "no_body_to_touch",
		"integral_parts_missing", "name_not_spoken"}, int(r))
}

// PoisonsAtDeath says which of a creature's ongoing effects are poisons, and whether
// its Poisoned condition came from poison, because the SRD has no poison category.
// Construct it with a keyed literal; the zero value is no poison. Effects is read
// and never changed.
type PoisonsAtDeath struct {
	// Effects are the ids of the held ongoing effects that are poisons.
	Effects []OngoingEffectID
	// PoisonedConditionFromPoison is true when the Poisoned condition came from poison.
	PoisonedConditionFromPoison bool
}

// ineligibilities lists every reason spell cannot work, in the fixed order.
func (s RevivalSpell) ineligibilities(c DeathCircumstances, nameSpoken bool) []RevivalIneligibility {
	f := s.Facts()
	var out []RevivalIneligibility
	if c.timeSinceDeath > f.window {
		out = append(out, TimeWindowExceeded)
	}
	if c.diedOfOldAge && f.oldAgeBlocks {
		out = append(out, DiedOfOldAge)
	}
	if c.wasUndead && f.undeadBlocks {
		out = append(out, WasUndead)
	}
	if c.body == NoBody && !f.nameForBody {
		out = append(out, NoBodyToTouch)
	}
	if c.body == MissingIntegralParts && f.integralPartsBlock {
		out = append(out, IntegralPartsMissing)
	}
	if c.body == NoBody && f.nameForBody && !nameSpoken {
		out = append(out, NameNotSpoken)
	}
	return out
}
