package d20

import "github.com/Whealinghold/srd5e-go/modifier"

// AttackRollResult is the outcome of one attack roll: the roll, the modifiers,
// the total, the Armor Class, whether it hit and whether it was a Critical
// Hit ("Attack Rolls", SRD 5.2.1, p. 7).
//
// It is immutable, built only by [ResolveAttack] (or
// [AttackRollResult.AdjustedBy]), and not comparable with ==; use
// [AttackRollResult.Equal]. The zero value is not a result.
type AttackRollResult struct {
	roll          Roll
	modifiers     RollModifiers
	total         int
	target        ArmorClass
	outcome       Outcome
	critical      bool
	criticalRange int
}

// Roll returns the roll the attack was made with.
func (a AttackRollResult) Roll() Roll { return a.roll }

// Modifiers returns the modifiers added to the natural value.
func (a AttackRollResult) Modifiers() RollModifiers { return a.modifiers }

// Total returns the natural value plus the modifiers, unclamped.
func (a AttackRollResult) Total() int { return a.total }

// Target returns the Armor Class the total was compared with.
func (a AttackRollResult) Target() ArmorClass { return a.target }

// Outcome returns [Success] for a hit and [Failure] for a miss.
func (a AttackRollResult) Outcome() Outcome { return a.outcome }

// IsCriticalHit reports whether the attack was a Critical Hit. A Critical Hit
// always hits.
func (a AttackRollResult) IsCriticalHit() bool { return a.critical }

// CriticalRange returns the natural value at or above which the attack is a
// Critical Hit: 20 by default, 19 or 18 with the features at SRD 5.2.1, p. 49.
func (a AttackRollResult) CriticalRange() int { return a.criticalRange }

// Hit reports whether the attack hit. A natural 1 never does.
func (a AttackRollResult) Hit() bool { return a.outcome == Success }

// AdjustedBy returns the result of the same attack with delta applied after
// the roll ("Add Modifiers", SRD 5.2.1, p. 6): delta joins the other
// modifiers and the outcome is re-resolved with this result's critical range,
// so a natural 20 stays a hit and a natural 1 stays a miss at any total. A
// delta of 0 is allowed. a is unchanged. It returns the [ResolveAttack] error
// for the zero result.
func (a AttackRollResult) AdjustedBy(delta int) (AttackRollResult, error) {
	return ResolveAttack(a.roll, a.modifiers.WithOthers(modifier.New(delta)), a.target, a.criticalRange)
}

// Equal reports whether a and o are the same result field by field.
func (a AttackRollResult) Equal(o AttackRollResult) bool {
	return a.roll == o.roll && a.modifiers.Equal(o.modifiers) && a.total == o.total &&
		a.target == o.target && a.outcome == o.outcome && a.critical == o.critical &&
		a.criticalRange == o.criticalRange
}
