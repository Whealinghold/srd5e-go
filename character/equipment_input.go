package character

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// invalidInput returns an error matching srderr.ErrInvalidInput for a
// caller-computed number the equipment rules cannot use (equipment-rules
// Q16): a negative load, a rate where none is allowed, a figure out of range.
func invalidInput(what string) error {
	return fmt.Errorf("character: %s: %w", what, srderr.ErrInvalidInput)
}
