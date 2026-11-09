package character

import "strconv"

// RogueLevel is a level in the Rogue class, 1 to 20 (SRD 5.2.1, "Rogue
// Features", p. 62). It counts Rogue levels only, not the total level. It is an
// immutable value, comparable with ==. The zero RogueLevel is invalid; see
// IsZero. Safe for concurrent use.
type RogueLevel struct{ n int }

// NewRogueLevel returns the Rogue level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 20. It never panics.
func NewRogueLevel(n int) (RogueLevel, error) {
	if n < 1 || n > MaxLevel {
		return RogueLevel{}, invalidInput("Rogue level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return RogueLevel{n: n}, nil
}

// MustRogueLevel is NewRogueLevel for literals known to be valid. It panics
// with the error NewRogueLevel returns, and only then.
func MustRogueLevel(n int) RogueLevel {
	l, err := NewRogueLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l RogueLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l RogueLevel) IsZero() bool { return l.n == 0 }
