package content

import (
	"cmp"
	"strconv"
)

// ChallengeRating is the Challenge Rating of a monster: one of 0, 1/8, 1/4,
// 1/2 and the whole numbers 1 to 30 (SRD 5.2.1, "Experience Points by
// Challenge Rating", p. 256). The zero ChallengeRating is CR 0, which is
// valid. It is comparable with ==, ordered by [CompareChallengeRating] and
// safe for concurrent use.
type ChallengeRating struct{ step uint8 }

// crFractionSteps is the number of ratings below 1: 0, 1/8, 1/4 and 1/2.
const crFractionSteps = 4

// crWholeMax is the highest whole rating.
const crWholeMax = 30

// ParseChallengeRating returns the rating spelled exactly s: "0", "1/8", "1/4",
// "1/2" or "1" to "30", with no sign, padding or white space.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput that carries the
// rejected input for any other string ("1/3", "31", "-1", "0.5", "01", "").
// It never panics.
func ParseChallengeRating(s string) (ChallengeRating, error) {
	switch s {
	case "0":
		return ChallengeRating{}, nil
	case "1/8":
		return ChallengeRating{step: 1}, nil
	case "1/4":
		return ChallengeRating{step: 2}, nil
	case "1/2":
		return ChallengeRating{step: 3}, nil
	}
	if n, ok := wholeRating(s); ok {
		return ChallengeRating{step: uint8(n + crFractionSteps - 1)}, nil //nolint:gosec // n is at most 30.
	}
	return ChallengeRating{}, &ParseError{Vocabulary: "ChallengeRating", Value: s}
}

// wholeRating reads s as a whole rating from 1 to 30: digits only, no leading
// zero.
func wholeRating(s string) (int, bool) {
	if len(s) == 0 || len(s) > 2 || s[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, _ := strconv.Atoi(s) // digits only, at most two of them
	return n, n >= 1 && n <= crWholeMax
}

// MustChallengeRating is like [ParseChallengeRating] but panics with its error
// when s is not a rating.
func MustChallengeRating(s string) ChallengeRating {
	v, err := ParseChallengeRating(s)
	if err != nil {
		panic(err)
	}
	return v
}

// AllChallengeRatings returns the 34 ratings in ascending order, as a new
// slice.
func AllChallengeRatings() []ChallengeRating {
	out := make([]ChallengeRating, crFractionSteps+crWholeMax)
	for i := range out {
		out[i] = ChallengeRating{step: uint8(i)}
	}
	return out
}

// String returns the printed spelling of c: "0", "1/8", "1/4", "1/2" or a
// whole number.
func (c ChallengeRating) String() string {
	switch c.step {
	case 0:
		return "0"
	case 1:
		return "1/8"
	case 2:
		return "1/4"
	case 3:
		return "1/2"
	}
	return strconv.Itoa(int(c.step) - crFractionSteps + 1)
}

// IsZero reports whether c is CR 0.
func (c ChallengeRating) IsZero() bool { return c.step == 0 }

// CompareChallengeRating compares a and b like cmp.Compare: -1 when a is the
// lower rating, 0 when they are equal and +1 when a is higher, in the order 0
// < 1/8 < 1/4 < 1/2 < 1 < 2 < ... < 30.
func CompareChallengeRating(a, b ChallengeRating) int { return cmp.Compare(a.step, b.step) }

// ProficiencyBonus returns the Proficiency Bonus of a monster of rating c, 2
// to 9 (SRD 5.2.1, "Proficiency Bonus by Challenge Rating", p. 256).
func (c ChallengeRating) ProficiencyBonus() int {
	if c.step < crFractionSteps {
		return 2
	}
	whole := int(c.step) - crFractionSteps + 1
	return 2 + (whole-1)/4
}
