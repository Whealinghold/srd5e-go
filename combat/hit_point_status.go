package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// HitPointStatus is a creature's hit point state: Hit Points, LifeState, the
// ZeroHitPointsRule and DamageAffinities. It is an immutable, comparable value
// with no identity (the host keeps one per creature and passes the CombatantID
// to each operation). Create it with NewHitPointStatus or RestoreHitPointStatus;
// the zero value is invalid (see IsZero). Invariants: a Dead, Dying or Stable
// creature has 0 current Hit Points; an Alive one has more; a maximum of 0 means
// Dead with cause HitPointMaximumZero. Safe for concurrent use.
type HitPointStatus struct {
	hp   HitPoints
	life LifeState
	rule ZeroHitPointsRule
	aff  DamageAffinities
}

// HitPointStatusProblem is one reason a HitPointStatus could not be built. The
// zero value is not a problem.
type HitPointStatusProblem int

// The problems NewHitPointStatus and RestoreHitPointStatus report, in the order
// they are listed.
const (
	// MaximumBelowOne: the maximum of a new status is below 1.
	MaximumBelowOne HitPointStatusProblem = iota + 1
	// DeadWithHitPoints: a Dead creature with current Hit Points above 0.
	DeadWithHitPoints
	// DyingWithHitPoints: a Dying creature with current Hit Points above 0.
	DyingWithHitPoints
	// StableWithHitPoints: a Stable creature with current Hit Points above 0.
	StableWithHitPoints
	// AliveAtZeroHitPoints: an Alive creature with 0 current Hit Points.
	AliveAtZeroHitPoints
	// HitPointMaximumZeroNotDead: a maximum of 0 without Dead(HitPointMaximumZero).
	HitPointMaximumZeroNotDead
)

// String returns the snake_case name of p, or "unknown".
func (p HitPointStatusProblem) String() string {
	switch p {
	case MaximumBelowOne:
		return "maximum_below_one"
	case DeadWithHitPoints:
		return "dead_with_hit_points"
	case DyingWithHitPoints:
		return "dying_with_hit_points"
	case StableWithHitPoints:
		return "stable_with_hit_points"
	case AliveAtZeroHitPoints:
		return "alive_at_zero_hit_points"
	case HitPointMaximumZeroNotDead:
		return "hit_point_maximum_zero_not_dead"
	}
	return "unknown"
}

// HitPointStatusError reports every problem found building a HitPointStatus. It
// matches srderr.ErrInvalidInput with errors.Is, is never built with no
// problems and carries no display text.
type HitPointStatusError struct{ problems []HitPointStatusProblem }

// Problems returns a copy of the problems in the order of the enum.
func (e *HitPointStatusError) Problems() []HitPointStatusProblem { return slices.Clone(e.problems) }

// Error renders the error. The text is not a contract.
func (e *HitPointStatusError) Error() string {
	names := make([]string, len(e.problems))
	for i, p := range e.problems {
		names[i] = p.String()
	}
	return "combat: invalid hit point status: " + strings.Join(names, ", ")
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *HitPointStatusError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *HitPointStatusError listing the same problems.
func (e *HitPointStatusError) Is(target error) bool {
	t, ok := target.(*HitPointStatusError)
	return ok && t != nil && e != nil && slices.Equal(t.problems, e.problems)
}

// NewHitPointStatus returns a creature at full Hit Points: current equal to
// maximum, no Temporary Hit Points and Alive.
//
// It returns a *HitPointStatusError (matching srderr.ErrInvalidInput) with
// MaximumBelowOne for a maximum below 1, and an error matching
// srderr.ErrInvalidInput for a rule that is not valid. It never panics.
func NewHitPointStatus(maximum int, rule ZeroHitPointsRule, affinities DamageAffinities) (HitPointStatus, error) {
	if maximum < 1 {
		return HitPointStatus{}, &HitPointStatusError{problems: []HitPointStatusProblem{MaximumBelowOne}}
	}
	if !rule.Valid() {
		return HitPointStatus{}, invalidInput("hit point status needs a zero hit points rule")
	}
	return HitPointStatus{HitPoints{maximum, maximum, 0}, Alive{}, rule, affinities}, nil
}

// MustHitPointStatus is NewHitPointStatus for values known to be valid. It
// panics with the error NewHitPointStatus returns, and only then.
func MustHitPointStatus(maximum int, rule ZeroHitPointsRule, affinities DamageAffinities) HitPointStatus {
	s, err := NewHitPointStatus(maximum, rule, affinities)
	if err != nil {
		panic(err)
	}
	return s
}

// RestoreHitPointStatus rebuilds a status from saved or given parts.
//
// It returns a *HitPointStatusError (matching srderr.ErrInvalidInput) listing
// every problem: DeadWithHitPoints, DyingWithHitPoints, StableWithHitPoints,
// AliveAtZeroHitPoints and HitPointMaximumZeroNotDead (a maximum of 0 needs
// Dead(HitPointMaximumZero)). It returns an error matching
// srderr.ErrInvalidInput for a nil or invalid LifeState or an invalid rule. It
// never panics.
func RestoreHitPointStatus(hp HitPoints, life LifeState, rule ZeroHitPointsRule, affinities DamageAffinities) (HitPointStatus, error) {
	if !rule.Valid() {
		return HitPointStatus{}, invalidInput("hit point status needs a zero hit points rule")
	}
	var problems []HitPointStatusProblem
	add := func(p HitPointStatusProblem) { problems = append(problems, p) }
	switch l := life.(type) {
	case Alive:
		if hp.current == 0 {
			add(AliveAtZeroHitPoints)
		}
	case Dying:
		if hp.current > 0 {
			add(DyingWithHitPoints)
		}
	case Stable:
		if hp.current > 0 {
			add(StableWithHitPoints)
		}
	case Dead:
		if !l.cause.Valid() {
			return HitPointStatus{}, invalidInput("death needs a valid cause")
		}
		if hp.current > 0 {
			add(DeadWithHitPoints)
		}
	default:
		return HitPointStatus{}, invalidInput("hit point status needs a life state")
	}
	if dead, ok := life.(Dead); hp.maximum == 0 && (!ok || dead.cause != HitPointMaximumZero) {
		add(HitPointMaximumZeroNotDead)
	}
	if len(problems) > 0 {
		slices.Sort(problems)
		return HitPointStatus{}, &HitPointStatusError{problems: problems}
	}
	return HitPointStatus{hp, life, rule, affinities}, nil
}

// HitPoints returns the Hit Points.
func (s HitPointStatus) HitPoints() HitPoints { return s.hp }

// LifeState returns the life state: Alive, Dying, Stable or Dead.
func (s HitPointStatus) LifeState() LifeState { return s.life }

// ZeroHitPointsRule returns what dropping to 0 Hit Points does to the creature.
func (s HitPointStatus) ZeroHitPointsRule() ZeroHitPointsRule { return s.rule }

// Affinities returns the Resistances, Vulnerabilities and Immunities.
func (s HitPointStatus) Affinities() DamageAffinities { return s.aff }

// IsZero reports whether s is the invalid zero HitPointStatus.
func (s HitPointStatus) IsZero() bool { return s.life == nil }

// IsUnconscious reports whether the 0-Hit-Point rules leave the creature
// Unconscious: true for Dying and Stable. A knocked-out creature (1 Hit Point,
// Alive) is Unconscious through the KnockedOut event and the conditions, not
// through this predicate.
func (s HitPointStatus) IsUnconscious() bool {
	switch s.life.(type) {
	case Dying, Stable:
		return true
	}
	return false
}

// MakesDeathSavingThrows reports whether a Death Saving Throw is due: true if
// and only if the creature is Dying, whatever kind of creature it is.
func (s HitPointStatus) MakesDeathSavingThrows() bool {
	_, ok := s.life.(Dying)
	return ok
}
