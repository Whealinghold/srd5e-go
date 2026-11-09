package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	// RangerClassID is the id of the Ranger class.
	RangerClassID content.ClassID = "ranger"
	// HunterSubclassID is the id of the Hunter subclass, the only Ranger subclass in SRD 5.2.1.
	HunterSubclassID content.SubclassID = "hunter"
)

// RangerLevel is a level in the Ranger class, 1 to 20 (SRD 5.2.1, "Ranger Features", p. 58). It counts Ranger levels only, not
// the total level. It is an immutable value, comparable with ==. The zero RangerLevel is invalid; see IsZero. Safe for
// concurrent use.
type RangerLevel struct{ n int }

// NewRangerLevel returns the Ranger level n. It returns an error matching srderr.ErrInvalidInput for n below 1 or above 20. It
// never panics.
func NewRangerLevel(n int) (RangerLevel, error) {
	if n < 1 || n > MaxLevel {
		return RangerLevel{}, invalidInput("Ranger level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return RangerLevel{n: n}, nil
}

// MustRangerLevel is NewRangerLevel for literals known to be valid. It panics with the error NewRangerLevel returns, and only
// then.
func MustRangerLevel(n int) RangerLevel {
	l, err := NewRangerLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l RangerLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l RangerLevel) IsZero() bool { return l.n == 0 }

// rangerLevelOf returns the Ranger level of c and false when it has no Ranger levels. It reads ClassLevels alone, so a stored
// state never makes a non-Ranger a Ranger.
func rangerLevelOf(c Character) (RangerLevel, bool) {
	n := c.ClassLevels().LevelIn(RangerClassID)
	if n < 1 {
		return RangerLevel{}, false
	}
	return RangerLevel{n: n}, true
}
