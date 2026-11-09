package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case equipSpec FR-011, FR-012, FR-014 (SRD 5.2.1, "Carrying
// Capacity", p. 178) - the capacity table rows: Carry (as half pounds) and
// Drag/Lift/Push.
// Path: per case (happy; unhappy boundary).
func TestCarryingCapacityRows(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		size      content.CreatureSize
		strength  int
		carryHalf int
		drag      int
	}{
		{"Tiny, 10", content.SizeTiny, 10, 150, 150}, {"Small, 10", content.SizeSmall, 10, 300, 300},
		{"Medium, 10", content.SizeMedium, 10, 300, 300}, {"Medium, 15", content.SizeMedium, 15, 450, 450},
		{"Large, 20", content.SizeLarge, 20, 1200, 1200}, {"Huge, 25", content.SizeHuge, 25, 3000, 3000},
		{"Tiny, 1", content.SizeTiny, 1, 15, 15}, {"Tiny, 3", content.SizeTiny, 3, 45, 45},
		{"Tiny, 15", content.SizeTiny, 15, 225, 225}, {"Medium, 1", content.SizeMedium, 1, 30, 30},
		{"Gargantuan, 30", content.SizeGargantuan, 30, 7200, 7200},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.NewCarryingCapacity(c.size, score(t, c.strength))
			if got.CarryHalfPounds() != c.carryHalf || got.DragLiftPushPounds() != c.drag {
				t.Errorf("capacity = %d half-pounds / %d, want %d / %d", got.CarryHalfPounds(), got.DragLiftPushPounds(), c.carryHalf, c.drag)
			}
		})
	}
}

// Requirement: equipSpec FR-013 - Small and Medium give equal capacities for
// every score.
// Path: unhappy (rule interaction).
func TestSmallEqualsMedium(t *testing.T) {
	t.Parallel()
	for s := ability.MinScore; s <= ability.MaxScore; s++ {
		if character.NewCarryingCapacity(content.SizeSmall, score(t, s)) != character.NewCarryingCapacity(content.SizeMedium, score(t, s)) {
			t.Errorf("Small and Medium differ at Strength %d", s)
		}
	}
}

// Requirement: equipSpec FR-084 (SRD 5.2.1, "Powerful Build", p. 86) - the size
// is used as given: the host passes one size larger for a Goliath (Medium row
// gives Large, Large Form gives Huge, Gargantuan stays).
// Path: unhappy (rule interaction).
func TestCarryingCapacityUsesTheSizeGiven(t *testing.T) {
	t.Parallel()
	s := score(t, 15)
	cases := []struct {
		size  content.CreatureSize
		carry int
	}{{content.SizeLarge, 450}, {content.SizeHuge, 900}, {content.SizeGargantuan, 1800}}
	for _, c := range cases {
		if got := character.NewCarryingCapacity(c.size, s).CarryHalfPounds(); got != c.carry*2 {
			t.Errorf("%v: Carry %d half-pounds, want %d", c.size, got, c.carry*2)
		}
	}
	if got := character.NewCarryingCapacity(0, s); got != (character.CarryingCapacity{}) {
		t.Errorf("invalid size = %v, want zero capacity", got)
	}
}

// Requirement: per case equipSpec FR-015, FR-016, FR-017, FR-018, FR-019,
// FR-020 (SRD 5.2.1, "Carrying Capacity", p. 178) - the load assessment rows.
// Path: per case (unhappy boundary, empty and minimal, invalid input).
func TestAssessLoad(t *testing.T) {
	t.Parallel()
	const (
		within  = "within"
		limited = "limited"
		beyond  = "beyond"
	)
	medium := character.NewCarryingCapacity(content.SizeMedium, score(t, 10))
	tiny := character.NewCarryingCapacity(content.SizeTiny, score(t, 15))
	cases := []struct {
		name  string
		cap   character.CarryingCapacity
		load  int
		carry bool
		kind  string
	}{
		{"Medium 0", medium, 0, true, within}, {"Medium 150", medium, 150, true, within},
		{"Medium 151", medium, 151, false, limited}, {"Medium 300", medium, 300, false, limited},
		{"Medium 301", medium, 301, false, beyond},
		{"Tiny 112", tiny, 112, true, within}, {"Tiny 113", tiny, 113, false, limited},
		{"Tiny 225", tiny, 225, false, limited}, {"Tiny 226", tiny, 226, false, beyond},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			carry, err := c.cap.CanCarry(c.load)
			if err != nil || carry != c.carry {
				t.Errorf("CanCarry = %v, %v, want %v", carry, err, c.carry)
			}
			got, err := c.cap.Assess(c.load)
			if err != nil {
				t.Fatal(err)
			}
			switch v := got.(type) {
			case character.LoadWithinCarry:
				if c.kind != within {
					t.Errorf("Assess = within, want %s", c.kind)
				}
			case character.LoadSpeedLimited:
				if c.kind != limited || v.LimitFeet() != 5 {
					t.Errorf("Assess = limited (%d feet), want %s and 5", v.LimitFeet(), c.kind)
				}
			case character.LoadBeyondDragLiftPush:
				if c.kind != beyond {
					t.Errorf("Assess = beyond, want %s", c.kind)
				}
			}
		})
	}
}

// Requirement: equipSpec FR-020 - a negative load is an srderr.ErrInvalidInput
// error from both CanCarry and Assess.
// Path: unhappy (invalid input).
func TestNegativeLoad(t *testing.T) {
	t.Parallel()
	c := character.NewCarryingCapacity(content.SizeMedium, score(t, 10))
	_, err := c.CanCarry(-1)
	wantInvalid(t, err, "CanCarry")
	_, err = c.Assess(-1)
	wantInvalid(t, err, "Assess")
}

// Requirement: equipSpec FR-019 - LoadSpeedLimited exposes LimitFeet (5) and
// applies it to nothing.
// Path: unhappy (boundary).
func TestLoadSpeedLimitedFeet(t *testing.T) {
	t.Parallel()
	got, err := character.NewCarryingCapacity(content.SizeMedium, score(t, 10)).Assess(151)
	if l, ok := got.(character.LoadSpeedLimited); err != nil || !ok || l.LimitFeet() != 5 {
		t.Errorf("load 151 = %v, %v", got, err)
	}
}
