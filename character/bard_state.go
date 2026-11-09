package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
)

// BardicInspirationPool is the Bard's uses of Bardic Inspiration: the uses remaining and the maximum, which is the
// Charisma modifier with a minimum of one (SRD 5.2.1, "Bardic Inspiration", "Number of Uses", p. 32). It is a
// FeatureState stored under BardicInspirationPoolKey and enforces 0 <= remaining <= maximum. It is immutable,
// comparable with ==, and the zero value is 0 of 0 uses. Safe for concurrent use.
type BardicInspirationPool struct{ max, remaining int }

// BardicInspirationPoolKey returns the FeatureStateKey named bardic_inspiration_pool (class-bard.md FR-118).
func BardicInspirationPoolKey() FeatureStateKey[BardicInspirationPool] {
	return MustFeatureStateKey[BardicInspirationPool]("bardic_inspiration_pool")
}

// bardicMaximum returns the maximum number of uses for a Charisma modifier: the modifier, at least 1.
func bardicMaximum(charisma ability.Modifier) int { return max(charisma.Value(), 1) }

// InitialBardicInspirationPool returns the full pool for a Charisma modifier (FR-121).
func InitialBardicInspirationPool(charisma ability.Modifier) BardicInspirationPool {
	n := bardicMaximum(charisma)
	return BardicInspirationPool{max: n, remaining: n}
}

// NewBardicInspirationPool returns a pool of remaining uses out of maximum. It returns an error matching
// srderr.ErrInvalidInput for a maximum below 1 or a remaining count outside 0 to the maximum. It never panics.
func NewBardicInspirationPool(maximum, remaining int) (BardicInspirationPool, error) {
	if maximum < 1 || remaining < 0 || remaining > maximum {
		return BardicInspirationPool{}, invalidInput("Bardic Inspiration " + strconv.Itoa(remaining) + " of " + strconv.Itoa(maximum) + " uses")
	}
	return BardicInspirationPool{max: maximum, remaining: remaining}, nil
}

// Maximum returns the maximum number of uses.
func (p BardicInspirationPool) Maximum() int { return p.max }

// Remaining returns the unexpended uses.
func (p BardicInspirationPool) Remaining() int { return p.remaining }

// Equal reports whether other is a BardicInspirationPool with the same uses.
func (p BardicInspirationPool) Equal(other FeatureState) bool {
	o, ok := other.(BardicInspirationPool)
	return ok && o == p
}

// WithMaximumFrom returns the pool with the maximum recomputed from the current Charisma modifier: the remaining
// uses are clamped to the new maximum and never raised (FR-121).
func (p BardicInspirationPool) WithMaximumFrom(charisma ability.Modifier) BardicInspirationPool {
	n := bardicMaximum(charisma)
	return BardicInspirationPool{max: n, remaining: min(p.remaining, n)}
}

// spend returns the pool with one use expended, never below zero; the callers check that a use remains first.
func (p BardicInspirationPool) spend() BardicInspirationPool {
	p.remaining = max(p.remaining-1, 0)
	return p
}

// restoreAll returns the pool with every use regained.
func (p BardicInspirationPool) restoreAll() BardicInspirationPool {
	p.remaining = p.max
	return p
}

// restoreOne returns the pool with one expended use regained, never above the maximum.
func (p BardicInspirationPool) restoreOne() BardicInspirationPool {
	p.remaining = min(p.remaining+1, p.max)
	return p
}

// restoreToTwo returns the pool with uses regained until two remain, or the maximum when that is lower; a pool
// holding two or more is unchanged (Superior Inspiration, FR-043, FR-108).
func (p BardicInspirationPool) restoreToTwo() BardicInspirationPool {
	p.remaining = max(p.remaining, min(2, p.max))
	return p
}

// readBardicInspirationPool returns the pool c stores, with its maximum recomputed from charisma. A character with
// no stored pool is a broken invariant for a Bard (FR-118): it returns an error matching srderr.ErrInvalidInput, as
// it does when another type of state sits under the key.
func readBardicInspirationPool(c Character, charisma ability.Modifier) (BardicInspirationPool, error) {
	s, ok, err := ReadFeatureState(c.FeatureStates(), BardicInspirationPoolKey())
	if err != nil {
		return BardicInspirationPool{}, err
	}
	if !ok {
		return BardicInspirationPool{}, invalidInput("a Bard holds no Bardic Inspiration pool")
	}
	return s.WithMaximumFrom(charisma), nil
}
