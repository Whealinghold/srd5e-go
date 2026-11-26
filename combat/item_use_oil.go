package combat

import (
	"math"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
)

// oilDries is how long Oil stays wet: it dries after 1 minute (SRD 5.2.1, "Oil", p. 98).
const oilDries = time.Minute

// OilCoatState is the state of an oil coat. The zero value is OilCoatWet.
type OilCoatState int

// The three states.
const (
	// OilCoatWet: the oil has not dried and has not burned.
	OilCoatWet OilCoatState = iota
	// OilCoatDried: the oil dried after 1 minute.
	OilCoatDried
	// OilCoatBurnedOff: the oil burned once.
	OilCoatBurnedOff
)

// String returns "wet", "dried", "burned_off" or "unknown".
func (s OilCoatState) String() string {
	if s < OilCoatWet || s > OilCoatBurnedOff {
		return "unknown"
	}
	return []string{"wet", "dried", "burned_off"}[s]
}

// OilCoat is a creature or object covered in oil (SRD 5.2.1, "Oil", p. 98): wet for the first
// minute, when Fire damage taken adds 5 Fire damage from burning oil, once. A coat holds only
// the time since it was applied, which the host supplies through Advance (nothing here reads a
// clock). The zero value is a fresh wet coat. Immutable and comparable.
type OilCoat struct {
	elapsed time.Duration
	state   OilCoatState
}

// NewOilCoat returns a fresh wet coat. A new failed Oil save on a target carrying any coat
// gives it a fresh coat that replaces the old one, never a second (Q-8).
func NewOilCoat() OilCoat { return OilCoat{} }

// Elapsed returns the time since the oil was applied.
func (c OilCoat) Elapsed() time.Duration { return c.elapsed }

// State returns the state of the coat.
func (c OilCoat) State() OilCoatState { return c.state }

// satAddDuration adds two non-negative durations, saturating at the largest Duration.
func satAddDuration(a, b time.Duration) time.Duration {
	if b > math.MaxInt64-a {
		return math.MaxInt64
	}
	return a + b
}

// Advance adds d to the elapsed time; a wet coat dries once the elapsed time is at least 1
// minute, so exactly 60 seconds is dried (Q-8). A dried or burned-off coat does not change
// except for its elapsed time. The sum saturates.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with ItemUseFieldElapsed for a
// negative d. It never panics.
func (c OilCoat) Advance(d time.Duration) (OilCoat, error) {
	if d < 0 {
		return OilCoat{}, itemUseInvalid(ItemUseFieldElapsed, int64(d))
	}
	c.elapsed = satAddDuration(c.elapsed, d)
	if c.state == OilCoatWet && c.elapsed >= oilDries {
		c.state = OilCoatDried
	}
	return c, nil
}

// OilIgnition is the outcome of Fire damage on an oil coat. Variants: OilIgnited, OilNotIgnited.
type OilIgnition interface{ isOilIgnition() }

// OilIgnited: the wet oil burned. The extra 5 Fire damage is a second Damage instance, so
// Resistance and Vulnerability apply to it once; the coat is burned off.
type OilIgnited struct {
	damage Damage
	coat   OilCoat
}

// Damage returns the extra 5 Fire damage from burning oil.
func (i OilIgnited) Damage() Damage { return i.damage }

// Coat returns the coat, burned off.
func (i OilIgnited) Coat() OilCoat { return i.coat }

// OilNotIgnitedReason is why oil did not burn. The zero value is no reason.
type OilNotIgnitedReason int

// The reasons.
const (
	// OilNotIgnitedDried: the oil has dried.
	OilNotIgnitedDried OilNotIgnitedReason = iota + 1
	// OilNotIgnitedBurnedOff: the oil already burned once.
	OilNotIgnitedBurnedOff
	// OilNotIgnitedNoFireDamageTaken: no Fire damage was taken.
	OilNotIgnitedNoFireDamageTaken
)

// String returns "dried", "burned_off", "no_fire_damage_taken" or "unknown".
func (r OilNotIgnitedReason) String() string {
	return enumName([]string{"dried", "burned_off", "no_fire_damage_taken"}, int(r))
}

// OilNotIgnited: the oil did not burn and the coat is unchanged.
type OilNotIgnited struct {
	reason OilNotIgnitedReason
	coat   OilCoat
}

// Reason returns why.
func (i OilNotIgnited) Reason() OilNotIgnitedReason { return i.reason }

// Coat returns the unchanged coat.
func (i OilNotIgnited) Coat() OilCoat { return i.coat }

func (OilIgnited) isOilIgnition()    {}
func (OilNotIgnited) isOilIgnition() {}

// OnFireDamage says what Fire damage does to the coat: a wet coat and fireDamageTaken of 1 or
// more give OilIgnited with 5 Fire damage and a burned-off coat; a dried or burned-off coat,
// or no Fire damage, give OilNotIgnited. fireDamageTaken is DamageTaken.Total of the Fire
// damage, so Temporary Hit Points that absorb it count and Immunity that reduces it to 0
// does not. The extra damage is not itself Fire damage taken for a second ignition.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with ItemUseFieldDamageTaken
// for a negative fireDamageTaken. It never panics.
func (c OilCoat) OnFireDamage(fireDamageTaken int) (OilIgnition, error) {
	switch {
	case fireDamageTaken < 0:
		return nil, itemUseInvalid(ItemUseFieldDamageTaken, int64(fireDamageTaken))
	case c.state == OilCoatDried:
		return OilNotIgnited{OilNotIgnitedDried, c}, nil
	case c.state == OilCoatBurnedOff:
		return OilNotIgnited{OilNotIgnitedBurnedOff, c}, nil
	case fireDamageTaken == 0:
		return OilNotIgnited{OilNotIgnitedNoFireDamageTaken, c}, nil
	}
	c.state = OilCoatBurnedOff
	return OilIgnited{MustDamage(5, content.DamageFire), c}, nil
}
