package combat

import "github.com/Whealinghold/srd5e-go/srderr"

// AttackRangeField names the field of an AttackRange that is violated.
type AttackRangeField int

// The fields. Single is the range of a single-range attack, Normal and Long
// those of a normal-and-long one.
const (
	AttackRangeFieldSingle AttackRangeField = iota + 1
	AttackRangeFieldNormal
	AttackRangeFieldLong
)

// String returns "single", "normal", "long" or "unknown".
func (f AttackRangeField) String() string {
	return enumName([]string{"single", "normal", "long"}, int(f))
}

// AttackRangeProblem is one violated field of NewAttackRange. It matches
// srderr.ErrInvalidInput with errors.Is and is comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type AttackRangeProblem struct{ Field AttackRangeField }

// Error renders the problem. The text is not a contract.
func (p AttackRangeProblem) Error() string { return problemText("attack range", p.Field.String()) }

// Unwrap returns srderr.ErrInvalidInput.
func (p AttackRangeProblem) Unwrap() error { return srderr.ErrInvalidInput }

// rangeBand says where a distance falls against an attack's range.
type rangeBand int

const (
	bandWithinNormal rangeBand = iota
	bandBeyondNormal
	bandBeyondLong
)

// AttackRange is the range of a ranged attack, in feet: a single range, or a
// normal and a long range (SRD 5.2.1, "Range", p. 15; "Range", p. 90). Build it
// with NewAttackRange. The zero value is invalid. Comparable and safe for
// concurrent use.
type AttackRange struct{ normal, long int }

// NewAttackRange returns the range with the normal distance and, when long is
// not 0, the long one; a long of 0 makes a single range of normal feet.
//
// It returns the errors.Join of every AttackRangeProblem (single or normal not
// positive, then long not larger than normal when long is not 0), matching
// srderr.ErrInvalidInput. It never panics.
func NewAttackRange(normal, long int) (AttackRange, error) {
	var ps []AttackRangeProblem
	if normal <= 0 {
		f := AttackRangeFieldNormal
		if long == 0 {
			f = AttackRangeFieldSingle
		}
		ps = append(ps, AttackRangeProblem{f})
	}
	if long != 0 && long <= normal {
		ps = append(ps, AttackRangeProblem{AttackRangeFieldLong})
	}
	if err := joinProblems(ps); err != nil {
		return AttackRange{}, err
	}
	return AttackRange{normal, long}, nil
}

// MustAttackRange is NewAttackRange for values known to be valid. It panics with
// the error NewAttackRange returns, and only then.
func MustAttackRange(normal, long int) AttackRange {
	r, err := NewAttackRange(normal, long)
	if err != nil {
		panic(err)
	}
	return r
}

// Normal returns the normal range (the only one of a single range) in feet.
func (r AttackRange) Normal() int { return r.normal }

// Long returns the long range in feet, or 0 for a single range.
func (r AttackRange) Long() int { return r.long }

// IsZero reports whether r is the invalid zero value.
func (r AttackRange) IsZero() bool { return r.normal == 0 }

// bandAt places distance: within normal range (distance 0 included), beyond
// normal but within long, or beyond long (beyond a single range).
func (r AttackRange) bandAt(distance int) rangeBand {
	switch {
	case distance <= r.normal:
		return bandWithinNormal
	case r.long != 0 && distance <= r.long:
		return bandBeyondNormal
	}
	return bandBeyondLong
}

// defaultReachFeet is the reach of a creature without a rule saying otherwise.
const defaultReachFeet = 5

// ReachField names the field of a Reach that is violated.
type ReachField int

// ReachFieldFeet is the only field.
const ReachFieldFeet ReachField = iota + 1

// String returns "feet" or "unknown".
func (f ReachField) String() string { return enumName([]string{"feet"}, int(f)) }

// ReachProblem is the violated field of NewReach. It matches
// srderr.ErrInvalidInput with errors.Is and is comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type ReachProblem struct{ Field ReachField }

// Error renders the problem. The text is not a contract.
func (p ReachProblem) Error() string { return problemText("reach", p.Field.String()) }

// Unwrap returns srderr.ErrInvalidInput.
func (p ReachProblem) Unwrap() error { return srderr.ErrInvalidInput }

// Reach is how far a melee attack reaches, in feet: 5 unless a rule says
// otherwise (SRD 5.2.1, "Reach", p. 15). The zero value is the default 5 feet. A
// supplied reach of less than 5 feet is read as 5. Comparable and safe for
// concurrent use.
type Reach struct{ feet int }

// NewReach returns a reach of feet. It returns an error matching
// srderr.ErrInvalidInput, as a ReachProblem, for feet that is not positive. It
// never panics.
func NewReach(feet int) (Reach, error) {
	if feet <= 0 {
		return Reach{}, ReachProblem{ReachFieldFeet}
	}
	return Reach{feet}, nil
}

// MustReach is NewReach for values known to be valid. It panics with the error
// NewReach returns, and only then.
func MustReach(feet int) Reach {
	r, err := NewReach(feet)
	if err != nil {
		panic(err)
	}
	return r
}

// Feet returns the reach in feet: the supplied reach when larger than 5, else 5.
func (r Reach) Feet() int { return max(r.feet, defaultReachFeet) }

// NearbyEnemyField names the field of a NearbyEnemy that is violated.
type NearbyEnemyField int

// NearbyEnemyFieldDistance is the only field.
const NearbyEnemyFieldDistance NearbyEnemyField = iota + 1

// String returns "distance" or "unknown".
func (f NearbyEnemyField) String() string { return enumName([]string{"distance"}, int(f)) }

// NearbyEnemyProblem is the violated field of NewNearbyEnemy. It matches
// srderr.ErrInvalidInput with errors.Is and is comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type NearbyEnemyProblem struct{ Field NearbyEnemyField }

// Error renders the problem. The text is not a contract.
func (p NearbyEnemyProblem) Error() string { return problemText("nearby enemy", p.Field.String()) }

// Unwrap returns srderr.ErrInvalidInput.
func (p NearbyEnemyProblem) Unwrap() error { return srderr.ErrInvalidInput }

// NearbyEnemy is an enemy of the attacker near it, as the host states: its
// distance in feet, whether it can see the attacker and whether it is
// Incapacitated (SRD 5.2.1, "Ranged Attacks in Close Combat", p. 15). Build it
// with NewNearbyEnemy. Comparable and safe for concurrent use.
type NearbyEnemy struct {
	distance       int
	canSeeAttacker bool
	incapacitated  bool
}

// NewNearbyEnemy returns the enemy. It returns an error matching
// srderr.ErrInvalidInput, as a NearbyEnemyProblem, for a negative distance. It
// never panics.
func NewNearbyEnemy(distance int, canSeeAttacker, incapacitated bool) (NearbyEnemy, error) {
	if distance < 0 {
		return NearbyEnemy{}, NearbyEnemyProblem{NearbyEnemyFieldDistance}
	}
	return NearbyEnemy{distance, canSeeAttacker, incapacitated}, nil
}

// MustNearbyEnemy is NewNearbyEnemy for values known to be valid. It panics with
// the error NewNearbyEnemy returns, and only then.
func MustNearbyEnemy(distance int, canSeeAttacker, incapacitated bool) NearbyEnemy {
	e, err := NewNearbyEnemy(distance, canSeeAttacker, incapacitated)
	if err != nil {
		panic(err)
	}
	return e
}

// Distance returns the distance in feet.
func (e NearbyEnemy) Distance() int { return e.distance }

// CanSeeAttacker reports whether the enemy can see the attacker.
func (e NearbyEnemy) CanSeeAttacker() bool { return e.canSeeAttacker }

// Incapacitated reports whether the enemy is Incapacitated.
func (e NearbyEnemy) Incapacitated() bool { return e.incapacitated }

// threatensInCloseCombat reports whether e makes a ranged attack harder: within
// 5 feet, able to see the attacker and not Incapacitated.
func (e NearbyEnemy) threatensInCloseCombat() bool {
	return e.distance <= 5 && e.canSeeAttacker && !e.incapacitated
}
