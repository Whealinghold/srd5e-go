package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/hiding-searching-and-senses.md FR-017, FR-097 - a SenseRangeError matches only a *SenseRangeError with the
// same Reason: not another reason, not a foreign error, not a nil target, and a nil receiver matches nothing.
// Path: unhappy (invalid input).
func TestSenseRangeErrorMatchesOnlyTheSameReason(t *testing.T) {
	t.Parallel()
	_, negative := combat.NewSenseRange(dv, -1)
	_, unknown := combat.NewSenseRange(0, 5)
	var nilTarget *combat.SenseRangeError
	var nilReceiver *combat.SenseRangeError
	sameReason := &combat.SenseRangeError{Reason: combat.SenseRangeReasonNegativeRange}
	cases := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"same reason", negative, sameReason, true},
		{"another reason", negative, &combat.SenseRangeError{Reason: combat.SenseRangeReasonUnknownSense}, false},
		{"another reason, the other way", unknown, sameReason, false},
		{"a foreign error", negative, errors.New("negative_range"), false},
		{"a nil *SenseRangeError target", negative, nilTarget, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := errors.Is(c.err, c.target); got != c.want {
				t.Errorf("errors.Is(%v, %v) = %t, want %t", c.err, c.target, got, c.want)
			}
		})
	}
	if nilReceiver.Is(sameReason) {
		t.Error("a nil *SenseRangeError matched a target, want false")
	}
}

// Requirement: doc/specs/hiding-searching-and-senses.md FR-020 to FR-024 - the zero value of each special-sense enum is invalid, so
// the named values start at 1 and run in order.
// Path: unhappy (boundary).
func TestSenseEnumsStartAtOne(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		got  []int
	}{
		{"TremorsenseContact", []int{int(combat.TremorsenseContactSharedSurfaceOrLiquid), int(combat.TremorsenseContactNoSharedSurface), int(combat.TremorsenseContactTargetInTheAir)}},
		{"IllusionAppearance", []int{int(combat.IllusionAppearanceTransparent), int(combat.IllusionAppearanceAsCreated)}},
		{"IllusionSave", []int{int(combat.IllusionSaveAutomaticSuccess), int(combat.IllusionSaveRollNormally)}},
		{"TrueFormPerception", []int{int(combat.TrueFormPerceptionDiscerned), int(combat.TrueFormPerceptionNotDiscerned)}},
		{"EtherealView", []int{int(combat.EtherealViewSeen), int(combat.EtherealViewNotSeen)}},
	}
	for _, c := range cases {
		for i, v := range c.got {
			if v != i+1 {
				t.Errorf("%s value %d = %d, want %d", c.name, i, v, i+1)
			}
		}
	}
}
