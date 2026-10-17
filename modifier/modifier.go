package modifier

import "strconv"

// Modifier is a bonus (positive), penalty (negative) or nothing (zero) added
// to a d20 number (SRD 5.2.1, "D20 Tests", "Add Modifiers", p. 6). Any int is
// valid. The zero Modifier is +0.
//
// Arithmetic is plain int arithmetic with no clamping and no overflow check;
// no SRD value comes near the limits.
type Modifier struct{ v int }

// New returns the modifier with value v.
func New(v int) Modifier { return Modifier{v} }

// Value returns the modifier as a plain integer.
func (m Modifier) Value() int { return m.v }

// Add returns the sum of m and o.
func (m Modifier) Add(o Modifier) Modifier { return Modifier{m.v + o.v} }

// ApplyTo returns d20 + m. It does not clamp or validate: the total may be
// below 1 or above 20.
func (m Modifier) ApplyTo(d20 int) int { return d20 + m.v }

// String returns the signed value, such as "+3", "+0" or "-2".
func (m Modifier) String() string {
	if m.v < 0 {
		return strconv.Itoa(m.v)
	}
	return "+" + strconv.Itoa(m.v)
}

// Sum returns the total of ms by signed addition. The sum of no modifiers is
// the zero Modifier. ms is not altered.
func Sum(ms ...Modifier) Modifier {
	var t Modifier
	for _, m := range ms {
		t = t.Add(m)
	}
	return t
}
