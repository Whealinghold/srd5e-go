package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// The numbers of the Wish stress (SRD 5.2.1, "Wish", p. 175).
const (
	// WishRestingDayReduction is how many days a day of resting with light activity takes off the
	// remaining Strength recovery time: 2 in all, one named place (wish-spell.md section 9).
	WishRestingDayReduction = 2

	wishStressMinDays      = 2  // 2d4
	wishStressMaxDays      = 8  // 2d4
	wishLockoutPercentile  = 33 // "33 percent chance": a 1d100 of 1 to 33 locks
	wishWeakenedStrength   = 3  // "your Strength score becomes 3"
	wishStressStateKeyName = "wish_stress"
)

// WishWeakness is the Strength part of the Wish stress, a closed set. Variants:
// NoWishWeakness, WishWeakened.
//
//sumtype:decl
type WishWeakness interface{ isWishWeakness() }

// NoWishWeakness is the absence of the Strength 3 effect.
type NoWishWeakness struct{}

// WishWeakened is the Strength 3 effect with its remaining days, 1 or more. Build one with
// NewWishWeakened; the zero value is invalid.
type WishWeakened struct{ days int }

// NewWishWeakened returns the effect with days remaining.
//
// It returns an error matching srderr.ErrInvalidInput for days below 1. It never panics.
func NewWishWeakened(days int) (WishWeakened, error) {
	if days < 1 {
		return WishWeakened{}, invalidInput("wish weakness of fewer than 1 day")
	}
	return WishWeakened{days: days}, nil
}

// DaysRemaining returns the days left, 1 or more.
func (w WishWeakened) DaysRemaining() int { return w.days }

func (NoWishWeakness) isWishWeakness() {}
func (WishWeakened) isWishWeakness()   {}

// WishCastDamage is the cast damage part of the Wish stress, a closed set. Variants:
// WishCastDamageInactive, WishCastDamageActive.
//
//sumtype:decl
type WishCastDamage interface{ isWishCastDamage() }

// WishCastDamageInactive means later casts cost no Necrotic damage.
type WishCastDamageInactive struct{}

// WishCastDamageActive means each later cast costs 1d10 Necrotic damage per level of the spell,
// until a Long Rest is finished.
type WishCastDamageActive struct{}

func (WishCastDamageInactive) isWishCastDamage() {}
func (WishCastDamageActive) isWishCastDamage()   {}

// WishCastingAbility is the lockout part of the Wish stress, a closed set. Variants:
// CanCastWish, CannotCastWishAgain.
//
//sumtype:decl
type WishCastingAbility interface{ isWishCastingAbility() }

// CanCastWish means the creature may cast Wish.
type CanCastWish struct{}

// CannotCastWishAgain means the creature is unable to cast Wish ever again. No transition clears it.
type CannotCastWishAgain struct{}

func (CanCastWish) isWishCastingAbility()         {}
func (CannotCastWishAgain) isWishCastingAbility() {}

// WishStress is the persistent state of the stress of casting Wish (SRD 5.2.1, "Wish", p. 175):
// the Strength 3 effect, the Necrotic damage on later casts and the lockout. It is a FeatureState
// stored under WishStressKey. The zero value is "no stress"; a nil part given to NewWishStress
// reads as its first variant. It is immutable, comparable with == and safe for concurrent use.
type WishStress struct {
	weakenedDays int
	costly       bool
	locked       bool
}

// NewWishStress returns the stress made of the three parts; a nil part is its first variant.
//
// It returns an error matching srderr.ErrInvalidInput for a WishWeakened of fewer than 1 day (the
// zero WishWeakened). It never panics.
func NewWishStress(weakness WishWeakness, damage WishCastDamage, casting WishCastingAbility) (WishStress, error) {
	var s WishStress
	if w, ok := weakness.(WishWeakened); ok {
		if w.days < 1 {
			return WishStress{}, invalidInput("wish stress with a zero weakness")
		}
		s.weakenedDays = w.days
	}
	_, s.costly = damage.(WishCastDamageActive)
	_, s.locked = casting.(CannotCastWishAgain)
	return s, nil
}

// Weakness returns NoWishWeakness or WishWeakened.
func (s WishStress) Weakness() WishWeakness {
	if s.weakenedDays > 0 {
		return WishWeakened{days: s.weakenedDays}
	}
	return NoWishWeakness{}
}

// CastDamage returns WishCastDamageInactive or WishCastDamageActive.
func (s WishStress) CastDamage() WishCastDamage {
	if s.costly {
		return WishCastDamageActive{}
	}
	return WishCastDamageInactive{}
}

// CastingAbility returns CanCastWish or CannotCastWishAgain.
func (s WishStress) CastingAbility() WishCastingAbility {
	if s.locked {
		return CannotCastWishAgain{}
	}
	return CanCastWish{}
}

// CanCastWish reports whether the creature may cast Wish: false exactly for CannotCastWishAgain.
func (s WishStress) CanCastWish() bool { return !s.locked }

// Equal reports whether other is a WishStress of the same parts.
func (s WishStress) Equal(other FeatureState) bool {
	o, ok := other.(WishStress)
	return ok && o == s
}

// WishStressKey returns the FeatureStateKey under which a Character holds its WishStress, named
// "wish_stress". An absent state reads as the zero WishStress.
func WishStressKey() FeatureStateKey[WishStress] {
	return MustFeatureStateKey[WishStress](wishStressStateKeyName)
}

// wishStressOf reads the stress a character holds; the zero WishStress when none is stored.
func wishStressOf(c Character) (WishStress, error) {
	s, _, err := ReadFeatureState(c.FeatureStates(), WishStressKey())
	return s, err
}

// WishStressRoll is the two dice of the stress: the 2d4 days and the 1d100 percentile. The zero
// value is invalid; build one with NewWishStressRoll.
type WishStressRoll struct{ days, percentile int }

// NewWishStressRoll returns the roll of days (2 to 8) and percentile (1 to 100).
//
// It returns an error matching srderr.ErrInvalidInput for a value out of range. It never panics.
func NewWishStressRoll(days, percentile int) (WishStressRoll, error) {
	if days < wishStressMinDays || days > wishStressMaxDays || percentile < 1 || percentile > 100 {
		return WishStressRoll{}, invalidInput("wish stress roll out of range")
	}
	return WishStressRoll{days: days, percentile: percentile}, nil
}

// Days returns the 2d4, 2 to 8.
func (r WishStressRoll) Days() int { return r.days }

// Percentile returns the 1d100, 1 to 100.
func (r WishStressRoll) Percentile() int { return r.percentile }

// LocksOut reports whether the percentile is 1 to 33, the 33 percent chance of never casting Wish again.
func (r WishStressRoll) LocksOut() bool { return r.percentile <= wishLockoutPercentile }

// IsZero reports whether r is the invalid zero roll.
func (r WishStressRoll) IsZero() bool { return r == WishStressRoll{} }

// IncurWishStress returns s after the stress of roll: Strength 3 for the rolled days (the newest
// roll governs even when shorter), the cast damage active, and the lockout set when the percentile
// is 33 or less. An existing lockout is never cleared. It is pure and never panics.
func IncurWishStress(s WishStress, roll WishStressRoll) WishStress {
	return WishStress{weakenedDays: roll.days, costly: true, locked: s.locked || roll.LocksOut()}
}

// WishEffectiveScores returns scores with Strength exactly 3 while stress is WishWeakened, whatever
// the stored value ("becomes 3", not a cap), and scores unchanged otherwise. The stored scores are
// never rewritten. It is pure and never panics.
func WishEffectiveScores(scores ability.Scores, stress WishStress) ability.Scores {
	if stress.weakenedDays == 0 || scores.IsZero() {
		return scores
	}
	weakened, err := scores.With(ability.Strength, ability.MustScore(wishWeakenedStrength))
	if err != nil { // unreachable: Strength and 3 are valid
		return scores
	}
	return weakened
}

// WishRecoveryDay is how a day of Strength recovery passed, a closed enum. The zero value is an ordinary day.
type WishRecoveryDay int

// The recovery days.
const (
	// WishOrdinaryDay is a day without rest, 1 day off the remaining time.
	WishOrdinaryDay WishRecoveryDay = iota
	// WishRestingDayWithLightActivity is a day of resting and nothing more than light activity,
	// WishRestingDayReduction days off.
	WishRestingDayWithLightActivity
)

// String returns "ordinary" or "resting_with_light_activity"; any other value counts as an ordinary day and reads "ordinary".
func (d WishRecoveryDay) String() string {
	if d == WishRestingDayWithLightActivity {
		return "resting_with_light_activity"
	}
	return "ordinary"
}

// WishTransition is a WishStress after a day or a rest, with the events of the change. Only the
// transition functions build one.
type WishTransition struct {
	stress WishStress
	events []Event
}

// Stress returns the new WishStress, for the host to write back with WithFeatureState.
func (t WishTransition) Stress() WishStress { return t.stress }

// Events returns a new slice of the events, in order.
func (t WishTransition) Events() []Event { return slices.Clone(t.events) }

// WishAfterDay returns stress after one day of recovery: 1 day off the remaining Strength 3 time, or
// WishRestingDayReduction for WishRestingDayWithLightActivity, never below 0. Reaching 0 ends the
// weakness with WishWeaknessEnded; a stress without weakness returns equal and no events. A Long
// Rest does not count as a day. It is pure and never panics.
func WishAfterDay(id Identifier, stress WishStress, day WishRecoveryDay) WishTransition {
	if stress.weakenedDays == 0 {
		return WishTransition{stress: stress}
	}
	off := 1
	if day == WishRestingDayWithLightActivity {
		off = WishRestingDayReduction
	}
	stress.weakenedDays -= off
	if stress.weakenedDays > 0 {
		return WishTransition{stress: stress}
	}
	stress.weakenedDays = 0
	return WishTransition{stress: stress, events: []Event{WishWeaknessEnded{id: id}}}
}

// WishAfterRest returns stress after a rest: a finished Long Rest (BenefitsOf LongRest) ends the cast
// damage with WishCastDamageEnded; a Short Rest, an interrupted Long Rest and a stress without cast
// damage return equal and no events. The weakness and the lockout never change.
//
// It returns an error matching srderr.ErrInvalidInput when rest concerns another character than id.
// It never panics.
func WishAfterRest(id Identifier, stress WishStress, rest RestApplied) (WishTransition, error) {
	if rest.CharacterID() != id {
		return WishTransition{}, invalidInput("wish stress after the rest of another character")
	}
	if rest.BenefitsOf() != LongRest || !stress.costly {
		return WishTransition{stress: stress}, nil
	}
	stress.costly = false
	return WishTransition{stress: stress, events: []Event{WishCastDamageEnded{id: id}}}, nil
}
