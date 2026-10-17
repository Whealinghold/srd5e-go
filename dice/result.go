package dice

import (
	"slices"
	"strconv"
	"strings"
)

// Result is the outcome of rolling an Expression: the expression and one
// DieRoll per die. It is immutable and, because it holds a slice, not
// comparable with ==; use Equal. The zero Result is invalid; see IsZero.
type Result struct {
	expression Expression
	rolls      []DieRoll
	diceTotal  int
}

// NewResult returns the Result of rolling expression with the given rolls. The
// rolls are copied.
//
// It returns an *InvalidError with ReasonCountOutOfRange when expression is
// the zero Expression, ReasonRollCountMismatch when len(rolls) is not
// expression.Count(), or ReasonDieMismatch when any roll is of a different die,
// checked in that order. It never panics.
func NewResult(expression Expression, rolls []DieRoll) (Result, error) {
	if expression.IsZero() {
		return Result{}, invalid(ReasonCountOutOfRange, "zero expression")
	}
	if len(rolls) != expression.count {
		return Result{}, invalid(ReasonRollCountMismatch, strconv.Itoa(len(rolls))+" rolls for "+expression.Notation())
	}
	total := 0
	for _, r := range rolls {
		if r.die != expression.die {
			return Result{}, invalid(ReasonDieMismatch, r.die.String()+" in "+expression.Notation())
		}
		total += r.value
	}
	return Result{expression, slices.Clone(rolls), total}, nil
}

// Expression returns the expression that was rolled.
func (r Result) Expression() Expression { return r.expression }

// Rolls returns a copy of the individual die rolls, in roll order.
func (r Result) Rolls() []DieRoll { return slices.Clone(r.rolls) }

// DiceTotal returns the sum of the faces, without the modifier.
func (r Result) DiceTotal() int { return r.diceTotal }

// Modifier returns the expression's modifier.
func (r Result) Modifier() int { return r.expression.modifier }

// Total returns DiceTotal plus Modifier.
func (r Result) Total() int { return r.diceTotal + r.expression.modifier }

// IsZero reports whether r is the invalid zero Result.
func (r Result) IsZero() bool { return r.expression.IsZero() }

// SumOfHighest returns the sum of the n highest faces (no modifier), for
// keep-highest rolls such as advantage.
//
// It returns an *InvalidError with ReasonSelectionOutOfRange when n is outside
// 1..Expression().Count(). It never panics.
func (r Result) SumOfHighest(n int) (int, error) {
	faces, err := r.sortedFaces(n)
	if err != nil {
		return 0, err
	}
	return sum(faces[len(faces)-n:]), nil
}

// SumOfLowest returns the sum of the n lowest faces (no modifier), for
// keep-lowest rolls such as disadvantage.
//
// It returns an *InvalidError with ReasonSelectionOutOfRange when n is outside
// 1..Expression().Count(). It never panics.
func (r Result) SumOfLowest(n int) (int, error) {
	faces, err := r.sortedFaces(n)
	if err != nil {
		return 0, err
	}
	return sum(faces[:n]), nil
}

// Equal reports whether r and o have the same expression and the same rolls in
// the same order.
func (r Result) Equal(o Result) bool {
	return r.expression == o.expression && slices.Equal(r.rolls, o.rolls)
}

// String returns the expression and rolls, for example "2d8+2 [d8=3 d8=5]".
func (r Result) String() string {
	parts := make([]string, len(r.rolls))
	for i, roll := range r.rolls {
		parts[i] = roll.String()
	}
	return r.expression.Notation() + " [" + strings.Join(parts, " ") + "]"
}

func (r Result) sortedFaces(n int) ([]int, error) {
	if n < 1 || n > r.expression.count {
		return nil, invalid(ReasonSelectionOutOfRange, "n "+strconv.Itoa(n)+" not in 1.."+strconv.Itoa(r.expression.count))
	}
	faces := make([]int, len(r.rolls))
	for i, roll := range r.rolls {
		faces[i] = roll.value
	}
	slices.Sort(faces)
	return faces, nil
}

func sum(values []int) int {
	t := 0
	for _, v := range values {
		t += v
	}
	return t
}
