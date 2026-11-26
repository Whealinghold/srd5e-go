package combat

// LifeState is where a creature stands in the rules of dropping to 0 Hit
// Points (SRD 5.2.1, "Dropping to 0 Hit Points", pp. 17-18). Variants: Alive,
// Dying, Stable, Dead.
type LifeState interface{ lifeState() }

// Alive is a creature that is not at 0 Hit Points, or has been knocked out (and
// so has 1 Hit Point).
type Alive struct{}

// Dying is a creature at 0 Hit Points making Death Saving Throws, with its
// tallies of successes and failures (each 0 to 2). The zero Dying has both
// tallies at 0. Build others with NewDying.
type Dying struct{ successes, failures int }

// Stable is a creature at 0 Hit Points that makes no Death Saving Throws (SRD
// 5.2.1, Rules Glossary, "Stable", p. 188).
type Stable struct{}

// Dead is a creature that has died, with the cause. It is terminal. The zero
// Dead has no valid cause; build one with NewDead.
type Dead struct{ cause DeathCause }

func (Alive) lifeState()  {}
func (Dying) lifeState()  {}
func (Stable) lifeState() {}
func (Dead) lifeState()   {}

// NewDying returns the Dying state with the given tallies. It returns an error
// matching srderr.ErrInvalidInput for a tally outside 0 to 2 (the third success
// or failure ends the state). It never panics.
func NewDying(successes, failures int) (Dying, error) {
	if successes < 0 || successes > 2 || failures < 0 || failures > 2 {
		return Dying{}, invalidInput("death saving throw tally outside 0 to 2")
	}
	return Dying{successes, failures}, nil
}

// Successes returns the Death Saving Throw successes, 0 to 2.
func (d Dying) Successes() int { return d.successes }

// Failures returns the Death Saving Throw failures, 0 to 2.
func (d Dying) Failures() int { return d.failures }

// NewDead returns the Dead state with the given cause. It returns an error
// matching srderr.ErrInvalidInput for a cause that is not valid.
func NewDead(cause DeathCause) (Dead, error) {
	if !cause.Valid() {
		return Dead{}, invalidInput("death needs a valid cause")
	}
	return Dead{cause}, nil
}

// Cause returns why the creature died.
func (d Dead) Cause() DeathCause { return d.cause }

// DeathCause is why a creature died. The zero value is invalid.
type DeathCause int

const (
	// MonsterDeath: dropped to 0 Hit Points with the DiesInstantly rule (SRD 5.2.1, "Monster Death", p. 17).
	MonsterDeath DeathCause = iota + 1
	// HitPointMaximumZero: the Hit Point maximum reached 0 (SRD 5.2.1, "Hit Point Maximum of 0", p. 17).
	HitPointMaximumZero
	// MassiveDamage: the damage left at 0 Hit Points equalled or exceeded the maximum (SRD 5.2.1, "Massive Damage", p. 17).
	MassiveDamage
	// DeathSavingThrowFailures: the third Death Saving Throw failure.
	DeathSavingThrowFailures
	// Exhaustion: Exhaustion reached level 6 (SRD 5.2.1, Rules Glossary, "Exhaustion", p. 181).
	Exhaustion
)

// Valid reports whether c is one of the five causes.
func (c DeathCause) Valid() bool { return c >= MonsterDeath && c <= Exhaustion }

// String returns the snake_case name of c, the value of
// srd5e.combat.death_cause, or "unknown".
func (c DeathCause) String() string {
	switch c {
	case MonsterDeath:
		return "monster_death"
	case HitPointMaximumZero:
		return "hit_point_maximum_zero"
	case MassiveDamage:
		return "massive_damage"
	case DeathSavingThrowFailures:
		return "death_saving_throw_failures"
	case Exhaustion:
		return "exhaustion"
	}
	return "unknown"
}
