package character

import (
	"errors"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// HitPointMaximumBonus is a feature that raises the Hit Point maximum: a flat
// amount and an amount per level, from a level on, optionally counted in one
// class only (SRD 5.2.1, "Level 3: Draconic Resilience", p. 69; "Dwarven
// Toughness", p. 84). The caller supplies the numbers from content; this type
// holds no feature names. It is an immutable value, comparable with ==. The
// zero HitPointMaximumBonus is invalid. Safe for concurrent use.
type HitPointMaximumBonus struct {
	flat, perLevel, grantedAtLevel int
	scope                          content.ClassID
}

// HitPointMaximumBonusProblem names a violated field rule of
// NewHitPointMaximumBonus. The zero value is not a problem.
type HitPointMaximumBonusProblem int

// The Hit Point maximum bonus problems, in the order they are reported.
const (
	// HitPointMaximumBonusNegativeFlat: flat is below 0.
	HitPointMaximumBonusNegativeFlat HitPointMaximumBonusProblem = iota + 1
	// HitPointMaximumBonusNegativePerLevel: perLevel is below 0.
	HitPointMaximumBonusNegativePerLevel
	// HitPointMaximumBonusGrantedAtLevelOutOfRange: grantedAtLevel is outside 1 to 20.
	HitPointMaximumBonusGrantedAtLevelOutOfRange
)

// String returns "negativeFlat", "negativePerLevel", "grantedAtLevelOutOfRange"
// or "unknown".
func (p HitPointMaximumBonusProblem) String() string {
	switch p {
	case HitPointMaximumBonusNegativeFlat:
		return "negativeFlat"
	case HitPointMaximumBonusNegativePerLevel:
		return "negativePerLevel"
	case HitPointMaximumBonusGrantedAtLevelOutOfRange:
		return "grantedAtLevelOutOfRange"
	}
	return "unknown"
}

// HitPointMaximumBonusError reports one violated field rule. It matches
// srderr.ErrInvalidInput with errors.Is.
type HitPointMaximumBonusError struct {
	Problem HitPointMaximumBonusProblem
}

// Error renders the error. The text is not a contract.
func (e *HitPointMaximumBonusError) Error() string {
	return "character: invalid Hit Point maximum bonus: " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *HitPointMaximumBonusError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *HitPointMaximumBonusError with the same
// Problem.
func (e *HitPointMaximumBonusError) Is(target error) bool {
	t, ok := target.(*HitPointMaximumBonusError)
	return ok && t != nil && e != nil && t.Problem == e.Problem
}

// NewHitPointMaximumBonus returns a bonus of flat plus perLevel for each level
// past grantedAtLevel, counted in the levels of scope, or in the total level
// when scope is the zero ClassID.
//
// It returns an errors.Join of one *HitPointMaximumBonusError per violated
// rule, each at most once, in the order of HitPointMaximumBonusProblem, each
// matching srderr.ErrInvalidInput. It never panics.
func NewHitPointMaximumBonus(flat, perLevel, grantedAtLevel int, scope content.ClassID) (HitPointMaximumBonus, error) {
	var problems []error
	if flat < 0 {
		problems = append(problems, &HitPointMaximumBonusError{Problem: HitPointMaximumBonusNegativeFlat})
	}
	if perLevel < 0 {
		problems = append(problems, &HitPointMaximumBonusError{Problem: HitPointMaximumBonusNegativePerLevel})
	}
	if grantedAtLevel < MinLevel || grantedAtLevel > MaxLevel {
		problems = append(problems, &HitPointMaximumBonusError{Problem: HitPointMaximumBonusGrantedAtLevelOutOfRange})
	}
	if len(problems) > 0 {
		return HitPointMaximumBonus{}, errors.Join(problems...)
	}
	return HitPointMaximumBonus{flat: flat, perLevel: perLevel, grantedAtLevel: grantedAtLevel, scope: scope}, nil
}

// MustHitPointMaximumBonus is NewHitPointMaximumBonus for constants known to be
// valid. It panics with the error NewHitPointMaximumBonus returns, and only then.
func MustHitPointMaximumBonus(flat, perLevel, grantedAtLevel int, scope content.ClassID) HitPointMaximumBonus {
	b, err := NewHitPointMaximumBonus(flat, perLevel, grantedAtLevel, scope)
	if err != nil {
		panic(err)
	}
	return b
}

// IsZero reports whether b is the invalid zero HitPointMaximumBonus.
func (b HitPointMaximumBonus) IsZero() bool { return b.grantedAtLevel == 0 }

// Scope returns the class whose levels count, or the zero ClassID for the total
// level.
func (b HitPointMaximumBonus) Scope() content.ClassID { return b.scope }

// ContributionFor returns the Hit Points b adds for levels. With n the level in
// the scope (the class's level when scoped, the total level when not), it is 0
// below grantedAtLevel and flat + perLevel * (n - grantedAtLevel) otherwise.
func (b HitPointMaximumBonus) ContributionFor(levels ClassLevels) int {
	n := levels.TotalLevel().n
	if b.scope != "" {
		n = levels.LevelIn(b.scope)
	}
	if n < b.grantedAtLevel {
		return 0
	}
	return b.flat + b.perLevel*(n-b.grantedAtLevel)
}
