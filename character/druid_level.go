package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	// DruidClassID is the id of the Druid class.
	DruidClassID content.ClassID = "druid"
	// CircleOfTheLandSubclassID is the id of the Circle of the Land subclass, the only Druid subclass in SRD 5.2.1.
	CircleOfTheLandSubclassID content.SubclassID = "circle_of_the_land"
)

// DruidLevel is a level in the Druid class, 1 to 20 (SRD 5.2.1, "Druid Features", p. 41). It counts Druid levels only,
// not the total level. It is an immutable value, comparable with ==. The zero DruidLevel is invalid; see IsZero. Safe for
// concurrent use.
type DruidLevel struct{ n int }

// NewDruidLevel returns the Druid level n. It returns an error matching srderr.ErrInvalidInput for n below 1 or above
// 20. It never panics.
func NewDruidLevel(n int) (DruidLevel, error) {
	if n < 1 || n > MaxLevel {
		return DruidLevel{}, invalidInput("Druid level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return DruidLevel{n: n}, nil
}

// MustDruidLevel is NewDruidLevel for literals known to be valid. It panics with the error NewDruidLevel returns, and
// only then.
func MustDruidLevel(n int) DruidLevel {
	l, err := NewDruidLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l DruidLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l DruidLevel) IsZero() bool { return l.n == 0 }
