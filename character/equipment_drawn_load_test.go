package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case equipSpec FR-021, FR-022, FR-023, FR-025 (SRD 5.2.1,
// "Mounts and Cargo", p. 100) - the "Drawn load" rows: five times the summed
// capacities, the vehicle's weight counted, order irrelevant.
// Path: per case (unhappy boundary, rule interaction, immutability, empty and minimal).
func TestDrawnLoad(t *testing.T) {
	t.Parallel()
	m := func(c int) content.Mount { return mountOf(t, "animal_x", c) }
	cases := []struct {
		name           string
		animals        []content.Mount
		vehicle, cargo int
		limit          int
		canMove        bool
	}{
		{"Horse, Draft, Cart, 2,500", []content.Mount{m(540)}, 200, 2500, 2700, true},
		{"Horse, Draft, Cart, 2,501", []content.Mount{m(540)}, 200, 2501, 2700, false},
		{"Elephant, Carriage, 6,000", []content.Mount{m(1320)}, 600, 6000, 6600, true},
		{"Pony, Sled, 825", []content.Mount{m(225)}, 300, 825, 1125, true},
		{"Mastiff, Sled, 675", []content.Mount{m(195)}, 300, 675, 975, true},
		{"Mastiff, Sled, 676", []content.Mount{m(195)}, 300, 676, 975, false},
		{"Mule x2, Wagon", []content.Mount{m(420), m(420)}, 400, 3800, 4200, true},
		{"Riding + Draft, Carriage", []content.Mount{m(480), m(540)}, 600, 4500, 5100, true},
		{"Camel x2 + Mule, Wagon", []content.Mount{m(450), m(450), m(420)}, 400, 5200, 6600, true},
		{"other order", []content.Mount{m(420), m(450), m(450)}, 400, 5201, 6600, true},
		{"no animals, Cart", nil, 200, 0, 0, false},
		{"no animals, nothing", nil, 0, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := append([]content.Mount(nil), c.animals...)
			d := character.DrawnLoad{}
			if got := d.LimitPounds(c.animals); got != c.limit || d.LimitPounds(c.animals) != got {
				t.Errorf("limit = %d, want %d (and repeatable)", got, c.limit)
			}
			ok, err := d.CanMove(c.animals, c.vehicle, c.cargo)
			if err != nil || ok != c.canMove {
				t.Errorf("CanMove = %v, %v, want %v", ok, err, c.canMove)
			}
			for i := range before {
				if c.animals[i] != before[i] {
					t.Errorf("animals altered at %d", i)
				}
			}
		})
	}
}

// Requirement: equipSpec FR-024 - a negative vehicle or cargo weight is an
// srderr.ErrInvalidInput error.
// Path: unhappy (invalid input).
func TestDrawnLoadNegative(t *testing.T) {
	t.Parallel()
	mule := []content.Mount{mountOf(t, "mule_x", 420)}
	_, err := character.DrawnLoad{}.CanMove(mule, 200, -1)
	wantInvalid(t, err, "cargo -1")
	_, err = character.DrawnLoad{}.CanMove(mule, -1, 0)
	wantInvalid(t, err, "vehicle -1")
}
