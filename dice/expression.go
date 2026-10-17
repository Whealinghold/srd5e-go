package dice

import (
	"strconv"
	"strings"
)

// Limits of an Expression.
const (
	// MaxCount is the most dice an Expression may roll.
	MaxCount = 1000
	// MaxModifier is the largest magnitude of an Expression modifier.
	MaxModifier = 1000000
)

// tooLarge stands in for a number too big to represent; it is above every
// limit, so range checks classify it correctly.
const tooLarge = 2000000000

// Expression is a die expression: count dice of one kind plus a flat modifier,
// such as "2d8+2". It is comparable. The zero Expression is invalid; see
// IsZero.
type Expression struct {
	count    int
	die      Die
	modifier int
}

// NewExpression returns the expression count x die + modifier.
//
// It returns an *InvalidError with ReasonCountOutOfRange when count is outside
// 1..MaxCount, ReasonSidesOutOfRange when die is the zero Die, or
// ReasonModifierOutOfRange when the modifier magnitude exceeds MaxModifier,
// checked in that order. It never panics.
func NewExpression(count int, die Die, modifier int) (Expression, error) {
	if count < 1 || count > MaxCount {
		return Expression{}, invalid(ReasonCountOutOfRange, "count "+strconv.Itoa(count)+" not in 1.."+strconv.Itoa(MaxCount))
	}
	if die.sides < MinSides || die.sides > MaxSides {
		return Expression{}, invalid(ReasonSidesOutOfRange, "die "+die.String())
	}
	if modifier > MaxModifier || modifier < -MaxModifier {
		return Expression{}, invalid(ReasonModifierOutOfRange, "modifier "+strconv.Itoa(modifier))
	}
	return Expression{count, die, modifier}, nil
}

// MustExpression is NewExpression for known-good constants. It panics with
// the error NewExpression would return; never pass it untrusted input.
func MustExpression(count int, die Die, modifier int) Expression {
	e, err := NewExpression(count, die, modifier)
	if err != nil {
		panic(err)
	}
	return e
}

// ParseExpression parses dice notation such as "d20", "4d6", "2d8+2",
// "2d8 + 2", "1d4 - 5" or "1d20−1" (U+2212).
//
// The grammar is [count] ("d"|"D") sides [sign modifier]. The count may be
// omitted (one die). The sign is "+", "-" or U+2212, with a space on both
// sides of it or on neither. Digits are ASCII, leading zeros are allowed and
// read as decimal, and there is no other whitespace. Parsing is strict:
// callers trim deliberately.
//
// It returns an *InvalidError with ReasonEmpty for "", ReasonMalformed for
// text that is not notation, and otherwise the first of ReasonCountOutOfRange,
// ReasonSidesOutOfRange, ReasonModifierOutOfRange that applies; numbers too
// large to represent are out of range, not malformed. It never panics.
func ParseExpression(text string) (Expression, error) {
	if text == "" {
		return Expression{}, invalid(ReasonEmpty, "")
	}
	n, ok := scan(text)
	if !ok {
		return Expression{}, invalid(ReasonMalformed, "")
	}
	count := 1
	if n.count != "" {
		count = readNumber(n.count)
	}
	modifier := readNumber(n.modifier)
	if n.negative {
		modifier = -modifier
	}
	sides := readNumber(n.sides)
	if sides < MinSides || sides > MaxSides {
		// NewExpression reports count first, so check it here to keep the
		// documented order.
		if count < 1 || count > MaxCount {
			return Expression{}, invalid(ReasonCountOutOfRange, "")
		}
		return Expression{}, invalid(ReasonSidesOutOfRange, "")
	}
	return NewExpression(count, Die{sides}, modifier)
}

// MustParseExpression is ParseExpression for known-good constants. It panics
// with the error ParseExpression would return; never pass it untrusted input.
func MustParseExpression(text string) Expression {
	e, err := ParseExpression(text)
	if err != nil {
		panic(err)
	}
	return e
}

// readNumber reads a run of ASCII digits as decimal, or 0 for the empty run.
func readNumber(digits string) int {
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return 0
	}
	if len(digits) > 9 {
		return tooLarge
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return tooLarge
	}
	return n
}

// Count returns the number of dice, or 0 for the zero Expression.
func (e Expression) Count() int { return e.count }

// Die returns the kind of die rolled.
func (e Expression) Die() Die { return e.die }

// Modifier returns the flat modifier added to the dice total.
func (e Expression) Modifier() int { return e.modifier }

// IsZero reports whether e is the invalid zero Expression.
func (e Expression) IsZero() bool { return e.count == 0 }

// Minimum returns the lowest possible total: every die shows 1.
func (e Expression) Minimum() int { return e.count + e.modifier }

// Maximum returns the highest possible total: every die shows its top face.
func (e Expression) Maximum() int { return e.count*e.die.sides + e.modifier }

// Average returns the mean total rounded down (toward negative infinity).
func (e Expression) Average() int {
	// Shifting right is floor division by two for negative values too.
	return (e.count*(e.die.sides+1) + 2*e.modifier) >> 1
}

// CanDouble reports whether Double would succeed.
func (e Expression) CanDouble() bool { return e.count*2 <= MaxCount && !e.IsZero() }

// Double returns e with twice as many dice and the same modifier, as SRD
// critical hits require.
//
// It returns an *InvalidError with ReasonCountOutOfRange when the doubled count
// would exceed MaxCount or e is the zero Expression. It never panics.
func (e Expression) Double() (Expression, error) {
	if !e.CanDouble() {
		return Expression{}, invalid(ReasonCountOutOfRange, "doubling "+strconv.Itoa(e.count)+" dice exceeds "+strconv.Itoa(MaxCount))
	}
	e.count *= 2
	return e, nil
}

// Notation returns the canonical machine form: count always present,
// lowercase d, no spaces, ASCII sign, modifier omitted when 0 ("1d20",
// "2d8+2", "1d4-5"). ParseExpression(e.Notation()) equals e for every valid e.
func (e Expression) Notation() string {
	s := strconv.Itoa(e.count) + "d" + strconv.Itoa(e.die.sides)
	if e.modifier > 0 {
		s += "+"
	}
	if e.modifier != 0 {
		s += strconv.Itoa(e.modifier)
	}
	return s
}

// String returns Notation.
func (e Expression) String() string { return e.Notation() }
