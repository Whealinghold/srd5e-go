package ability

import "strconv"

// Score bounds. The SRD says 1 to 20, "although some monsters have a score as
// high as 30" (SRD 5.2.1, "Ability Scores", p. 5). The adventurer cap of 20 is
// a character rule and is not checked here.
const (
	MinScore = 1
	MaxScore = 30
)

// Score is an ability score from MinScore to MaxScore. The zero Score is
// invalid; see IsZero.
type Score struct{ v int }

// Modifier is the bonus or penalty an ability score gives.
type Modifier struct{ v int }

// Value returns the modifier as a plain integer.
func (m Modifier) Value() int { return m.v }

// String returns the signed modifier, such as "+2", "+0" or "-1".
func (m Modifier) String() string {
	if m.v < 0 {
		return strconv.Itoa(m.v)
	}
	return "+" + strconv.Itoa(m.v)
}

// NewScore returns the score v. It returns an *InvalidError with
// ReasonBelowMinimum or ReasonAboveMaximum when v is outside MinScore to
// MaxScore.
func NewScore(v int) (Score, error) {
	switch {
	case v < MinScore:
		return Score{}, invalid(ReasonBelowMinimum, "score "+strconv.Itoa(v)+" is below "+strconv.Itoa(MinScore))
	case v > MaxScore:
		return Score{}, invalid(ReasonAboveMaximum, "score "+strconv.Itoa(v)+" is above "+strconv.Itoa(MaxScore))
	}
	return Score{v}, nil
}

// MustScore is NewScore for constants known to be valid. It panics with the
// error NewScore would return; do not use it on untrusted input.
func MustScore(v int) Score {
	s, err := NewScore(v)
	if err != nil {
		panic(err)
	}
	return s
}

// Value returns the score as a plain integer, or 0 for the zero Score.
func (s Score) Value() int { return s.v }

// IsZero reports whether s is the invalid zero Score.
func (s Score) IsZero() bool { return s.v == 0 }

// Modifier is the Ability Modifiers table (SRD 5.2.1, p. 6): the score minus 10, halved and rounded down. The zero Score has the zero
// Modifier.
func (s Score) Modifier() Modifier {
	if s.IsZero() {
		return Modifier{}
	}
	return Modifier{(s.v - 10) >> 1}
}

// String returns the score as a decimal number, or "invalid" for the zero Score.
func (s Score) String() string {
	if s.IsZero() {
		return "invalid"
	}
	return strconv.Itoa(s.v)
}
