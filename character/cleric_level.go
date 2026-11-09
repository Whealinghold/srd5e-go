package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	// ClericClassID is the id of the Cleric class.
	ClericClassID content.ClassID = "cleric"
	// LifeDomainSubclassID is the id of the Life Domain subclass, the only Cleric subclass in SRD 5.2.1.
	LifeDomainSubclassID content.SubclassID = "life_domain"
)

// ClericLevel is a level in the Cleric class, 1 to 20 (SRD 5.2.1, "Cleric
// Features", p. 36). It counts Cleric levels only, not the total level. It is an
// immutable value, comparable with ==. The zero ClericLevel is invalid; see
// IsZero. Safe for concurrent use.
type ClericLevel struct{ n int }

// NewClericLevel returns the Cleric level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 20. It never panics.
func NewClericLevel(n int) (ClericLevel, error) {
	if n < 1 || n > MaxLevel {
		return ClericLevel{}, invalidInput("Cleric level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return ClericLevel{n: n}, nil
}

// MustClericLevel is NewClericLevel for literals known to be valid. It panics
// with the error NewClericLevel returns, and only then.
func MustClericLevel(n int) ClericLevel {
	l, err := NewClericLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l ClericLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l ClericLevel) IsZero() bool { return l.n == 0 }

// clericLevelOf returns the Cleric level of c and false when it has no Cleric
// levels. It reads ClassLevels alone, so a stored state never makes a
// non-Cleric a Cleric.
func clericLevelOf(c Character) (ClericLevel, bool) {
	n := c.ClassLevels().LevelIn(ClericClassID)
	if n < 1 {
		return ClericLevel{}, false
	}
	return ClericLevel{n: n}, true
}
