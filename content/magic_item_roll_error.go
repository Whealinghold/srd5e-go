package content

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollRangeError reports a roll outside the range of a static table (a d100
// miscibility roll, a d10 communication roll). Roll is the rejected roll and
// Min and Max the range the table covers. It matches srderr.ErrInvalidInput
// with errors.Is. Do not branch on the rendered text.
type RollRangeError struct {
	Roll, Min, Max int
}

// Error renders the roll and the range. The text is not a contract.
func (e *RollRangeError) Error() string {
	return "content: roll " + strconv.Itoa(e.Roll) + " is outside " + strconv.Itoa(e.Min) + " to " + strconv.Itoa(e.Max)
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *RollRangeError) Unwrap() error { return srderr.ErrInvalidInput }

// checkRoll returns a *RollRangeError unless 1 <= roll <= max: every table of
// this package starts at 1.
func checkRoll(roll, max int) error {
	if roll < 1 || roll > max {
		return &RollRangeError{Roll: roll, Min: 1, Max: max}
	}
	return nil
}
