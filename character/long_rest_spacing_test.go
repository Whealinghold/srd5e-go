package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/rests-and-hit-dice.md FR-009, FR-072 - the spacing is
// whole hours since the previous finished Long Rest; negative hours are host
// input errors (SRD 5.2.1, "Long Rest", p. 185).
// Path: unhappy (boundary: 0 and -1 hours).
func TestNewLongRestFinishedHoursAgo(t *testing.T) {
	t.Parallel()
	s, err := character.NewLongRestFinishedHoursAgo(0)
	if err != nil || s.Hours() != 0 || s != (character.LongRestFinishedHoursAgo{}) {
		t.Errorf("NewLongRestFinishedHoursAgo(0) = %v, %v", s, err)
	}
	s, err = character.NewLongRestFinishedHoursAgo(15)
	if err != nil || s.Hours() != 15 {
		t.Errorf("NewLongRestFinishedHoursAgo(15) = %v, %v", s, err)
	}
	if _, err := character.NewLongRestFinishedHoursAgo(-1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewLongRestFinishedHoursAgo(-1) err = %v, want ErrInvalidInput", err)
	}
}
