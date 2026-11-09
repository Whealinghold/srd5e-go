package character

import "github.com/Whealinghold/srd5e-go/content"

// drawnLoadFactor is how many times its carrying capacity an animal can draw
// (SRD 5.2.1, "Mounts and Cargo", p. 100).
const drawnLoadFactor = 5

// DrawnLoad is the stateless service for loads pulled by animals (SRD 5.2.1,
// "Mounts and Cargo", p. 100). The zero value is ready to use.
type DrawnLoad struct{}

// LimitPounds returns five times the sum of the animals' carrying capacities.
// The result does not depend on the order of animals, and animals is not
// altered. With no animals it is 0.
//
// shortcut: plain int arithmetic, so the sum can overflow a 32-bit int only for
// capacities no SRD animal has; widen if a host feeds such values.
func (DrawnLoad) LimitPounds(animals []content.Mount) int {
	sum := 0
	for _, m := range animals {
		sum += m.CarryingCapacityPounds()
	}
	return sum * drawnLoadFactor
}

// CanMove reports whether vehiclePounds plus cargoPounds does not exceed the
// limit of animals ("including the weight of the vehicle"). It returns an error
// matching srderr.ErrInvalidInput for a negative weight.
func (d DrawnLoad) CanMove(animals []content.Mount, vehiclePounds, cargoPounds int) (bool, error) {
	if vehiclePounds < 0 || cargoPounds < 0 {
		return false, invalidInput("negative weight")
	}
	return int64(vehiclePounds)+int64(cargoPounds) <= int64(d.LimitPounds(animals)), nil
}
