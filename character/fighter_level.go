package character

import "strconv"

// FighterLevel is a level in the Fighter class, 1 to 20 (SRD 5.2.1, "Fighter
// Features", p. 47). It counts Fighter levels only, not the total level. It is an
// immutable value, comparable with ==. The zero FighterLevel is invalid; see
// IsZero. Safe for concurrent use.
type FighterLevel struct{ n int }

// NewFighterLevel returns the Fighter level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 20. It never panics.
func NewFighterLevel(n int) (FighterLevel, error) {
	if n < 1 || n > MaxLevel {
		return FighterLevel{}, invalidInput("Fighter level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return FighterLevel{n: n}, nil
}

// MustFighterLevel is NewFighterLevel for literals known to be valid. It panics
// with the error NewFighterLevel returns, and only then.
func MustFighterLevel(n int) FighterLevel {
	l, err := NewFighterLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l FighterLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l FighterLevel) IsZero() bool { return l.n == 0 }
