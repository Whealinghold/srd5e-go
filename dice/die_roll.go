package dice

import "strconv"

// DieRoll is the face one die showed. It is comparable. The zero DieRoll is
// invalid; see IsZero.
type DieRoll struct {
	die   Die
	value int
}

// NewDieRoll returns the roll of die showing value.
//
// It returns an *InvalidError with ReasonValueOutOfRange when value is outside
// 1..die.Sides() (which includes every value for the zero Die). It never
// panics.
func NewDieRoll(die Die, value int) (DieRoll, error) {
	if value < 1 || value > die.sides {
		return DieRoll{}, invalid(ReasonValueOutOfRange, "value "+strconv.Itoa(value)+" not on "+die.String())
	}
	return DieRoll{die, value}, nil
}

// MustDieRoll is NewDieRoll for known-good values. It panics with the error
// NewDieRoll would return; never pass it untrusted input.
func MustDieRoll(die Die, value int) DieRoll {
	r, err := NewDieRoll(die, value)
	if err != nil {
		panic(err)
	}
	return r
}

// Die returns the die that was rolled.
func (r DieRoll) Die() Die { return r.die }

// Value returns the face shown, or 0 for the zero DieRoll.
func (r DieRoll) Value() int { return r.value }

// IsZero reports whether r is the invalid zero DieRoll.
func (r DieRoll) IsZero() bool { return r.value == 0 }

// String returns the die and face, for example "d8=5".
func (r DieRoll) String() string { return r.die.String() + "=" + strconv.Itoa(r.value) }
