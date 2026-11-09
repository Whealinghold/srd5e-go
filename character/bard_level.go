package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	// BardClassID is the id of the Bard class.
	BardClassID content.ClassID = "bard"
	// CollegeOfLoreSubclassID is the id of the College of Lore subclass, the only Bard subclass in SRD 5.2.1.
	CollegeOfLoreSubclassID content.SubclassID = "college_of_lore"
)

// The Bard levels at which features arrive (SRD 5.2.1, "Bard Features", p. 31).
const (
	bardSubclassLevel          = 3
	bardFontOfInspirationLevel = 5
	bardCountercharmLevel      = 7
	bardMagicalSecretsLevel    = 10
	bardSuperiorLevel          = 18
	bardWordsOfCreationLevel   = 20
	bardPeerlessSkillLevel     = 14
	bardMagicalDiscoveryLevel  = 6
)

// BardLevel is a level in the Bard class, 1 to 20 (SRD 5.2.1, "Bard Features", p. 31). It counts Bard levels only,
// not the total level. It is an immutable value, comparable with ==. The zero BardLevel is invalid; see IsZero.
// Safe for concurrent use.
type BardLevel struct{ n int }

// NewBardLevel returns the Bard level n. It returns an error matching srderr.ErrInvalidInput for n below 1 or
// above 20. It never panics.
func NewBardLevel(n int) (BardLevel, error) {
	if n < 1 || n > MaxLevel {
		return BardLevel{}, invalidInput("Bard level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return BardLevel{n: n}, nil
}

// MustBardLevel is NewBardLevel for literals known to be valid. It panics with the error NewBardLevel returns,
// and only then.
func MustBardLevel(n int) BardLevel {
	l, err := NewBardLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l BardLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l BardLevel) IsZero() bool { return l.n == 0 }

// bardLevelOf returns the Bard level of c and false when it has no Bard levels. It reads ClassLevels alone.
func bardLevelOf(c Character) (BardLevel, bool) {
	n := c.ClassLevels().LevelIn(BardClassID)
	if n < 1 {
		return BardLevel{}, false
	}
	return BardLevel{n: n}, true
}

// BardicDie is the size of the Bardic Inspiration die, a closed set (SRD 5.2.1, "Bard Features", p. 31): d6, d8,
// d10 or d12. The zero value is not a die.
type BardicDie int

// The Bardic Dice.
const (
	BardicD6  BardicDie = 6
	BardicD8  BardicDie = 8
	BardicD10 BardicDie = 10
	BardicD12 BardicDie = 12
)

// BardicDieAt returns the Bardic Die at the Bard level: d6 at levels 1 to 4, d8 at 5 to 9, d10 at 10 to 14 and
// d12 at 15 to 20 (SRD 5.2.1, "Bardic Inspiration", "At Higher Levels", p. 32). The zero level gives the d6.
func BardicDieAt(l BardLevel) BardicDie {
	switch {
	case l.n >= 15:
		return BardicD12
	case l.n >= bardMagicalSecretsLevel:
		return BardicD10
	case l.n >= bardFontOfInspirationLevel:
		return BardicD8
	}
	return BardicD6
}

// Valid reports whether d is one of the four dice.
func (d BardicDie) Valid() bool {
	return d == BardicD6 || d == BardicD8 || d == BardicD10 || d == BardicD12
}

// Sides returns the number of faces, 6, 8, 10 or 12; 0 for an invalid die.
func (d BardicDie) Sides() int {
	if !d.Valid() {
		return 0
	}
	return int(d)
}

// Die returns the kernel dice.Die with the same number of sides. It panics only for an invalid BardicDie, which
// the exported API cannot build except from the zero value.
func (d BardicDie) Die() dice.Die { return dice.MustDie(d.Sides()) }

// String returns "d6", "d8", "d10", "d12" or "unknown".
func (d BardicDie) String() string {
	if !d.Valid() {
		return "unknown"
	}
	return "d" + strconv.Itoa(int(d))
}
