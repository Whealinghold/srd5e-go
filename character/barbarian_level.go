package character

import "strconv"

// BarbarianLevel is a level in the Fighter class, 1 to 20 (SRD 5.2.1, "Fighter
// Features", p. 28). It counts Barbarian levels only, not the total level. It is an
// immutable value, comparable with ==. The zero BarbarianLevel is invalid; see
// IsZero. Safe for concurrent use.
type BarbarianLevel struct{ n int }

// NewBarbarianLevel returns the Barbarian level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 20. It never panics.
func NewBarbarianLevel(n int) (BarbarianLevel, error) {
	if n < 1 || n > MaxLevel {
		return BarbarianLevel{}, invalidInput("Barbarian level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return BarbarianLevel{n: n}, nil
}

// MustBarbarianLevel is NewBarbarianLevel for literals known to be valid. It panics
// with the error NewBarbarianLevel returns, and only then.
func MustBarbarianLevel(n int) BarbarianLevel {
	l, err := NewBarbarianLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l BarbarianLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l BarbarianLevel) IsZero() bool { return l.n == 0 }
