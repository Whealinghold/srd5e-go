package dice

import "strconv"

// Side limits of a Die.
const (
	// MinSides is the fewest sides a Die may have.
	MinSides = 2
	// MaxSides is the most sides a Die may have.
	MaxSides = 1000
)

// Die is a die with a fixed number of sides. It is comparable. The zero Die is
// invalid; see IsZero.
type Die struct{ sides int }

// NewDie returns a Die with the given sides.
//
// It returns an *InvalidError with ReasonSidesOutOfRange when sides is outside
// MinSides..MaxSides. It never panics.
func NewDie(sides int) (Die, error) {
	if sides < MinSides || sides > MaxSides {
		return Die{}, invalid(ReasonSidesOutOfRange, "sides "+strconv.Itoa(sides)+" not in "+
			strconv.Itoa(MinSides)+".."+strconv.Itoa(MaxSides))
	}
	return Die{sides}, nil
}

// MustDie is NewDie for known-good constants. It panics with the error NewDie
// would return; never pass it untrusted input.
func MustDie(sides int) Die {
	d, err := NewDie(sides)
	if err != nil {
		panic(err)
	}
	return d
}

// D3 returns the three-sided die.
func D3() Die { return Die{3} }

// D4 returns the four-sided die.
func D4() Die { return Die{4} }

// D6 returns the six-sided die.
func D6() Die { return Die{6} }

// D8 returns the eight-sided die.
func D8() Die { return Die{8} }

// D10 returns the ten-sided die.
func D10() Die { return Die{10} }

// D12 returns the twelve-sided die.
func D12() Die { return Die{12} }

// D20 returns the twenty-sided die.
func D20() Die { return Die{20} }

// D100 returns the hundred-sided die.
func D100() Die { return Die{100} }

// Sides returns the number of sides, or 0 for the zero Die.
func (d Die) Sides() int { return d.sides }

// IsZero reports whether d is the invalid zero Die.
func (d Die) IsZero() bool { return d.sides == 0 }

// String returns "d" followed by the sides, for example "d20".
func (d Die) String() string { return "d" + strconv.Itoa(d.sides) }
