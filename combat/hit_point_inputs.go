package combat

// ZeroHitPointsRule says what dropping to 0 Hit Points does to a creature
// (SRD 5.2.1, "Dropping to 0 Hit Points", p. 17). The host chooses it per
// creature; there is no default. The zero value is invalid.
type ZeroHitPointsRule int

const (
	// DiesInstantly: the creature dies at 0 Hit Points (Monster Death).
	DiesInstantly ZeroHitPointsRule = iota + 1
	// FallsUnconscious: the creature is Dying and makes Death Saving Throws.
	FallsUnconscious
)

// Valid reports whether r is DiesInstantly or FallsUnconscious.
func (r ZeroHitPointsRule) Valid() bool { return r == DiesInstantly || r == FallsUnconscious }

// String returns the name of r, or "unknown".
func (r ZeroHitPointsRule) String() string {
	switch r {
	case DiesInstantly:
		return "dies_instantly"
	case FallsUnconscious:
		return "falls_unconscious"
	}
	return "unknown"
}

// HitKind says whether damage came from a Critical Hit. It is a damage
// application input only (SRD 5.2.1, "Damage at 0 Hit Points", p. 18), not an
// attack outcome. The zero value is invalid.
type HitKind int

const (
	// NormalHit: not a Critical Hit.
	NormalHit HitKind = iota + 1
	// CriticalHit: a Critical Hit, which adds two Death Saving Throw failures at 0 Hit Points.
	CriticalHit
)

// Valid reports whether k is NormalHit or CriticalHit.
func (k HitKind) Valid() bool { return k == NormalHit || k == CriticalHit }

// DamageDelivery says how damage reached the creature. The zero value is
// invalid.
type DamageDelivery int

const (
	// MeleeAttackWithoutKnockOut: a melee attack that does not knock out.
	MeleeAttackWithoutKnockOut DamageDelivery = iota + 1
	// MeleeAttackKnockOut: a melee attack whose attacker chose to knock the target out (SRD 5.2.1, "Knocking Out a Creature", p. 17).
	MeleeAttackKnockOut
	// OtherSource: any other source of damage.
	OtherSource
)

// Valid reports whether d is one of the three deliveries.
func (d DamageDelivery) Valid() bool { return d >= MeleeAttackWithoutKnockOut && d <= OtherSource }

// TemporaryHitPointsChoice says which amount a creature keeps when it already
// holds Temporary Hit Points and gains more (SRD 5.2.1, "They Don't Stack",
// p. 18). The zero value is invalid.
type TemporaryHitPointsChoice int

const (
	// KeepCurrent keeps the Temporary Hit Points already held.
	KeepCurrent TemporaryHitPointsChoice = iota + 1
	// AcceptNew replaces them with the new amount.
	AcceptNew
)

// Valid reports whether c is KeepCurrent or AcceptNew.
func (c TemporaryHitPointsChoice) Valid() bool { return c == KeepCurrent || c == AcceptNew }

// HitPointRejection is why a hit point operation was refused. Refusals are
// results, not errors. The zero value is not a rejection.
type HitPointRejection int

const (
	// CreatureIsDead: the creature is Dead (SRD 5.2.1, Rules Glossary, "Dead", p. 180).
	CreatureIsDead HitPointRejection = iota + 1
	// NotDying: the creature is not Dying.
	NotDying
	// NotStable: the creature is not Stable.
	NotStable
)

// String returns the snake_case name of r, the value of
// srd5e.combat.refusal_reason, or "unknown".
func (r HitPointRejection) String() string {
	switch r {
	case CreatureIsDead:
		return "creature_is_dead"
	case NotDying:
		return "not_dying"
	case NotStable:
		return "not_stable"
	}
	return "unknown"
}

// DeathSavingThrowOutcome is what one Death Saving Throw did. The zero value is
// invalid.
type DeathSavingThrowOutcome int

const (
	// DeathSavingThrowSuccess: one success.
	DeathSavingThrowSuccess DeathSavingThrowOutcome = iota + 1
	// DeathSavingThrowFailure: one failure.
	DeathSavingThrowFailure
	// DeathSavingThrowTwoFailures: a natural 1, two failures.
	DeathSavingThrowTwoFailures
	// DeathSavingThrowRegainedHitPoint: a natural 20, the creature regains 1 Hit Point.
	DeathSavingThrowRegainedHitPoint
)

// String returns the value of srd5e.combat.death_save_result, or "unknown".
func (o DeathSavingThrowOutcome) String() string {
	switch o {
	case DeathSavingThrowSuccess:
		return valueDeathSaveSuccess
	case DeathSavingThrowFailure:
		return valueDeathSaveFailure
	case DeathSavingThrowTwoFailures:
		return valueDeathSaveCriticalFailure
	case DeathSavingThrowRegainedHitPoint:
		return valueDeathSaveCriticalSuccess
	}
	return "unknown"
}
