package character

// ZeroHitPointsOutcome is what happened when a creature was reduced to 0 Hit Points,
// supplied by the host from combat, a closed set. Variants: ZeroHitPointsNotKilled,
// ZeroHitPointsKilledOutright.
type ZeroHitPointsOutcome interface{ isZeroHitPointsOutcome() }

// ZeroHitPointsNotKilled: the creature was reduced to 0 Hit Points and not killed
// outright.
type ZeroHitPointsNotKilled struct{}

// ZeroHitPointsKilledOutright: the damage killed the creature outright (Massive
// Damage, or a Hit Point maximum of 0; SRD 5.2.1, "Instant Death", p. 17).
type ZeroHitPointsKilledOutright struct{}

func (ZeroHitPointsNotKilled) isZeroHitPointsOutcome()      {}
func (ZeroHitPointsKilledOutright) isZeroHitPointsOutcome() {}

// RelentlessEndurance is the Orc trait that drops a creature to 1 Hit Point instead
// of 0 (SRD 5.2.1, "Relentless Endurance", p. 86). The zero value is ready to use. It
// is stateless and safe for concurrent use.
type RelentlessEndurance struct{}

// Resolve returns the Hit Points the creature has instead and true when the trait
// triggers (the creature was not killed outright): 1. It returns 0 and false for a
// creature killed outright or a nil outcome.
func (RelentlessEndurance) Resolve(o ZeroHitPointsOutcome) (hitPointsInstead int, triggered bool) {
	if _, ok := o.(ZeroHitPointsNotKilled); ok {
		return 1, true
	}
	return 0, false
}
