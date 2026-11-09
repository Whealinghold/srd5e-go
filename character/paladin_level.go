package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	// PaladinClassID is the id of the Paladin class.
	PaladinClassID content.ClassID = "paladin"
	// OathOfDevotionSubclassID is the id of the Oath of Devotion subclass, the only Paladin subclass in SRD 5.2.1.
	OathOfDevotionSubclassID content.SubclassID = "oath_of_devotion"
)

// PaladinLevel is a level in the Paladin class, 1 to 20 (SRD 5.2.1, "Paladin
// Features", p. 53). It counts Paladin levels only, not the total level. It is an
// immutable value, comparable with ==. The zero PaladinLevel is invalid; see
// IsZero. Safe for concurrent use.
type PaladinLevel struct{ n int }

// NewPaladinLevel returns the Paladin level n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 20. It never panics.
func NewPaladinLevel(n int) (PaladinLevel, error) {
	if n < 1 || n > MaxLevel {
		return PaladinLevel{}, invalidInput("Paladin level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return PaladinLevel{n: n}, nil
}

// MustPaladinLevel is NewPaladinLevel for literals known to be valid. It panics
// with the error NewPaladinLevel returns, and only then.
func MustPaladinLevel(n int) PaladinLevel {
	l, err := NewPaladinLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l PaladinLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l PaladinLevel) IsZero() bool { return l.n == 0 }

// paladinLevelOf returns the Paladin level of c and false when it has no Paladin
// levels. It reads ClassLevels alone, so a stored state never makes a non-Paladin
// a Paladin.
func paladinLevelOf(c Character) (PaladinLevel, bool) {
	n := c.ClassLevels().LevelIn(PaladinClassID)
	if n < 1 {
		return PaladinLevel{}, false
	}
	return PaladinLevel{n: n}, true
}
