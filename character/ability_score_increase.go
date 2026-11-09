package character

import (
	"errors"
	"maps"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// AbilityScoreIncrease is the increase an Ability Score Improvement, an Epic
// Boon or a similar feat gives: a positive amount for each of some abilities
// and the maximum score the feat allows, 20 to 30 (SRD 5.2.1, "Ability Score
// Improvement", p. 87; "Epic Boon Feats", p. 88). It is immutable and
// defensively copied. It holds a map, so compare two with Equal, not ==. The
// zero AbilityScoreIncrease is invalid; see IsZero. Safe for concurrent use.
type AbilityScoreIncrease struct {
	increases map[ability.Ability]int
	maximum   int
}

// AbilityScoreIncreaseProblem names an invariant of AbilityScoreIncrease that
// an input violates. The zero value is not a problem.
type AbilityScoreIncreaseProblem int

// The increase problems, in the order NewAbilityScoreIncrease reports them.
const (
	// IncreaseEmpty: no ability is increased.
	IncreaseEmpty AbilityScoreIncreaseProblem = iota + 1
	// IncreaseAmountBelowMinimum: an amount is below 1.
	IncreaseAmountBelowMinimum
	// IncreaseInvalidAbility: a key is not a valid ability.
	IncreaseInvalidAbility
	// IncreaseMaximumBelowDefault: the maximum score is below 20.
	IncreaseMaximumBelowDefault
	// IncreaseMaximumAboveHighest: the maximum score is above 30.
	IncreaseMaximumAboveHighest
	// IncreaseSameAbilityTwice: a two-by-one increase names one ability twice.
	IncreaseSameAbilityTwice
)

// String returns "empty", "amountBelowMinimum", "invalidAbility",
// "maximumBelowDefault", "maximumAboveHighest", "sameAbilityTwice" or
// "unknown".
func (p AbilityScoreIncreaseProblem) String() string {
	switch p {
	case IncreaseEmpty:
		return "empty"
	case IncreaseAmountBelowMinimum:
		return "amountBelowMinimum"
	case IncreaseInvalidAbility:
		return "invalidAbility"
	case IncreaseMaximumBelowDefault:
		return "maximumBelowDefault"
	case IncreaseMaximumAboveHighest:
		return "maximumAboveHighest"
	case IncreaseSameAbilityTwice:
		return "sameAbilityTwice"
	}
	return "unknown"
}

// AbilityScoreIncreaseError reports one violated invariant of
// AbilityScoreIncrease with the offending parameter name and value. It matches
// srderr.ErrInvalidInput with errors.Is.
type AbilityScoreIncreaseError struct {
	Problem AbilityScoreIncreaseProblem
	// Parameter is "increases", "maximumScore", "first" or "second".
	Parameter string
	// InvalidValue is the offending amount, maximum or ability number; 0 for
	// an empty increase.
	InvalidValue int
}

// Error renders the error. The text is not a contract.
func (e *AbilityScoreIncreaseError) Error() string {
	return "character: invalid ability score increase: " + e.Parameter + ": " + e.Problem.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *AbilityScoreIncreaseError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *AbilityScoreIncreaseError with the same
// Problem and Parameter.
func (e *AbilityScoreIncreaseError) Is(target error) bool {
	t, ok := target.(*AbilityScoreIncreaseError)
	return ok && t != nil && e != nil && t.Problem == e.Problem && t.Parameter == e.Parameter
}

// NewAbilityScoreIncrease returns the increase of the given amounts to the
// given abilities, copied, with the highest score it may reach. It is the
// extension point for other feats (for example Grappler's +1 to Strength or
// Dexterity with a maximum of 20).
//
// It returns an errors.Join of one *AbilityScoreIncreaseError per problem,
// each matching srderr.ErrInvalidInput, in the order empty, amount below 1 (in
// Ability declaration order), invalid ability, maximum below 20, maximum above
// 30; a nil error if and only if the input is valid. It never panics.
func NewAbilityScoreIncrease(increases map[ability.Ability]int, maximumScore int) (AbilityScoreIncrease, error) {
	var problems []error
	add := func(p AbilityScoreIncreaseProblem, parameter string, value int) {
		problems = append(problems, &AbilityScoreIncreaseError{Problem: p, Parameter: parameter, InvalidValue: value})
	}
	if len(increases) == 0 {
		add(IncreaseEmpty, "increases", 0)
	}
	for _, a := range ability.All() {
		if amount, ok := increases[a]; ok && amount < 1 {
			add(IncreaseAmountBelowMinimum, "increases", amount)
		}
	}
	for _, a := range slices.Sorted(maps.Keys(increases)) {
		if !a.Valid() {
			add(IncreaseInvalidAbility, "increases", int(a))
		}
	}
	if maximumScore < DefaultScoreCap {
		add(IncreaseMaximumBelowDefault, "maximumScore", maximumScore)
	}
	if maximumScore > HighestScoreCap {
		add(IncreaseMaximumAboveHighest, "maximumScore", maximumScore)
	}
	if len(problems) > 0 {
		return AbilityScoreIncrease{}, errors.Join(problems...)
	}
	return AbilityScoreIncrease{increases: maps.Clone(increases), maximum: maximumScore}, nil
}

// MustAbilityScoreIncrease is NewAbilityScoreIncrease for constants known to be
// valid. It panics with the error NewAbilityScoreIncrease returns, and only
// then.
func MustAbilityScoreIncrease(increases map[ability.Ability]int, maximumScore int) AbilityScoreIncrease {
	i, err := NewAbilityScoreIncrease(increases, maximumScore)
	if err != nil {
		panic(err)
	}
	return i
}

// NewAbilityScoreImprovementOneByTwo returns the Ability Score Improvement
// increase of 2 to one ability, up to 20 (SRD 5.2.1, "Ability Score
// Improvement", p. 87). It returns the errors of NewAbilityScoreIncrease for
// an invalid ability. It never panics.
func NewAbilityScoreImprovementOneByTwo(which ability.Ability) (AbilityScoreIncrease, error) {
	return NewAbilityScoreIncrease(map[ability.Ability]int{which: 2}, DefaultScoreCap)
}

// NewAbilityScoreImprovementTwoByOne returns the Ability Score Improvement
// increase of 1 to each of two distinct abilities, up to 20. It returns an
// error matching srderr.ErrInvalidInput, a *AbilityScoreIncreaseError with
// IncreaseSameAbilityTwice, when the abilities are equal, and otherwise the
// errors of NewAbilityScoreIncrease. It never panics.
func NewAbilityScoreImprovementTwoByOne(first, second ability.Ability) (AbilityScoreIncrease, error) {
	if first == second {
		return AbilityScoreIncrease{}, &AbilityScoreIncreaseError{Problem: IncreaseSameAbilityTwice, Parameter: "second", InvalidValue: int(second)}
	}
	return NewAbilityScoreIncrease(map[ability.Ability]int{first: 1, second: 1}, DefaultScoreCap)
}

// NewEpicBoonIncrease returns the increase of 1 to one ability, up to 30, that
// an Epic Boon feat gives (SRD 5.2.1, "Epic Boon Feats", p. 88). It returns
// the errors of NewAbilityScoreIncrease for an invalid ability. It never
// panics.
func NewEpicBoonIncrease(which ability.Ability) (AbilityScoreIncrease, error) {
	return NewAbilityScoreIncrease(map[ability.Ability]int{which: 1}, HighestScoreCap)
}

// Increases returns a new map of the amount added to each ability.
func (i AbilityScoreIncrease) Increases() map[ability.Ability]int { return maps.Clone(i.increases) }

// MaximumScore returns the highest score the increase may bring an ability to.
func (i AbilityScoreIncrease) MaximumScore() int { return i.maximum }

// IsZero reports whether i is the invalid zero AbilityScoreIncrease.
func (i AbilityScoreIncrease) IsZero() bool { return len(i.increases) == 0 }

// Equal reports whether i and o hold the same amounts and maximum.
func (i AbilityScoreIncrease) Equal(o AbilityScoreIncrease) bool {
	return i.maximum == o.maximum && maps.Equal(i.increases, o.increases)
}

// AbilityScoreIncreaseRejection is the closed set of reasons AbilityScoreIncreaser
// refuses an increase. Variants: IncreaseExceedsMaximum.
type AbilityScoreIncreaseRejection interface{ isAbilityScoreIncreaseRejection() }

// IncreaseExceedsMaximum is the rejection of an increase that would bring an
// ability above the maximum score. Only AbilityScoreIncreaser builds one.
type IncreaseExceedsMaximum struct {
	abilities []ability.Ability
	maximum   int
}

// Abilities returns a new slice of every ability that would exceed the maximum,
// in Ability declaration order.
func (r IncreaseExceedsMaximum) Abilities() []ability.Ability { return slices.Clone(r.abilities) }

// MaximumScore returns the maximum score of the refused increase.
func (r IncreaseExceedsMaximum) MaximumScore() int { return r.maximum }

func (IncreaseExceedsMaximum) isAbilityScoreIncreaseRejection() {}

// AbilityScoreIncreaser applies an AbilityScoreIncrease to a Character. The
// zero value is ready to use. It is stateless and safe for concurrent use.
type AbilityScoreIncreaser struct{}

// Apply adds the increase to the Character (SRD 5.2.1, "Ability Score
// Improvement", p. 87; "Gaining a Level" step 5, p. 23). Each listed ability
// rises by its amount and every other ability is unchanged. The whole increase
// is accepted or refused: when any result would exceed the maximum score, none
// is applied and the result is a TransitionRejected holding
// IncreaseExceedsMaximum; there is no clamping. A raised Constitution modifier
// raises the Hit Point maximum by the aggregate's rule. The events are one
// AbilityScoreChanged per ability in Ability declaration order, then at most one
// HitPointMaximumChanged from the maximum before the increase to the maximum
// after it. The level and classes of the Character are not judged, and the
// receiver is unchanged.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or
// a zero increase. It never panics.
func (AbilityScoreIncreaser) Apply(c Character, inc AbilityScoreIncrease) (Transition[AbilityScoreIncreaseRejection], error) {
	if c.IsZero() || inc.IsZero() {
		return nil, invalidInput("ability score increase of a zero character or increase")
	}
	var over []ability.Ability
	for _, a := range ability.All() {
		if amount, ok := inc.increases[a]; ok && c.scores.Score(a).Value()+amount > inc.maximum {
			over = append(over, a)
		}
	}
	if len(over) > 0 {
		return rejected[AbilityScoreIncreaseRejection](IncreaseExceedsMaximum{abilities: over, maximum: inc.maximum}), nil
	}
	cur := c
	var events []Event
	for _, a := range ability.All() {
		amount, ok := inc.increases[a]
		if !ok {
			continue
		}
		score, err := ability.NewScore(c.scores.Score(a).Value() + amount)
		if err != nil { // unreachable: the result is at most the maximum, 30 at most
			return nil, err
		}
		tr, err := cur.ChangeAbilityScore(a, score, inc.maximum)
		if err != nil {
			return nil, err
		}
		step, ok := tr.(TransitionChanged[AbilityScoreChangeRejection])
		if !ok { // unreachable: the result is within the maximum and Constitution does not fall
			return nil, invalidInput("ability score increase refused by the aggregate")
		}
		cur = step.character
		events = append(events, step.events[0])
	}
	if cur.hitPoints != c.hitPoints {
		events = append(events, HitPointMaximumChanged{id: c.id, previous: c.hitPoints, current: cur.hitPoints})
	}
	return changed[AbilityScoreIncreaseRejection](cur, events...), nil
}
