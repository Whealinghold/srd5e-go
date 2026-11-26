package combat

import "github.com/Whealinghold/srd5e-go/dice"

// outcomeBand is the inclusive range of faces low to high of one row of a printed table, and what the row gives.
type outcomeBand[T any] struct {
	low, high int
	value     T
}

// outcomeTable is a printed table rolled with one die: contiguous bands that cover every face once (SRD 5.2.1,
// "Teleport", p. 168; "Confusion", p. 117; "Reincarnate", p. 158). A table is fixed program data built by a function
// that returns a fresh one on each call, so no state is shared (spell-outcome-tables.md FR-001 to FR-003).
type outcomeTable[T any] struct {
	die   dice.Die
	bands []outcomeBand[T]
}

// newOutcomeTable returns the table of die over bands. It checks nothing: validate does, in the tests.
func newOutcomeTable[T any](die dice.Die, bands []outcomeBand[T]) outcomeTable[T] {
	return outcomeTable[T]{die, bands}
}

// validate reports a gap, an overlap, a first band that does not start at 1 or a last band that does not end at the
// last face of the die, as an error matching srderr.ErrInvalidInput that names the bands.
func (t outcomeTable[T]) validate() error {
	next := 1
	for _, b := range t.bands {
		if b.low != next || b.high < b.low {
			return invalidInput("bands must start at 1 and be contiguous, without overlap")
		}
		next = b.high + 1
	}
	if next-1 != t.die.Sides() {
		return invalidInput("bands must end at the last face of the die")
	}
	return nil
}

// lookup returns the value of the band that holds face. A face outside 1 to the sides of the die is an error matching
// ErrFaceOutOfRange.
func (t outcomeTable[T]) lookup(face int) (T, error) {
	var zero T
	if face < 1 || face > t.die.Sides() {
		return zero, ErrFaceOutOfRange
	}
	for _, b := range t.bands {
		if face >= b.low && face <= b.high {
			return b.value, nil
		}
	}
	return zero, invalidInput("no band holds the face")
}
