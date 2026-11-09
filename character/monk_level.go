package character

import "strconv"

// MonkLevel is a level in the Monk class, 1 to 20 (SRD 5.2.1, "Monk
// Features", p. 50). It counts Monk levels only, not the total level. It is an
// immutable value, comparable with ==. The zero MonkLevel is invalid; see
// IsZero. Safe for concurrent use.
type MonkLevel struct{ n int }

// NewMonkLevel returns the Monk level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 20. It never panics.
func NewMonkLevel(n int) (MonkLevel, error) {
	if n < 1 || n > MaxLevel {
		return MonkLevel{}, invalidInput("Monk level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return MonkLevel{n: n}, nil
}

// MustMonkLevel is NewMonkLevel for literals known to be valid. It panics
// with the error NewMonkLevel returns, and only then.
func MustMonkLevel(n int) MonkLevel {
	l, err := NewMonkLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l MonkLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l MonkLevel) IsZero() bool { return l.n == 0 }
