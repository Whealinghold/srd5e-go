package character

import "math"

// Bounds that keep the half-mile conversion exact in a float64.
const (
	maxVehicleHalfMiles = 1 << 52
	maxVehicleMiles     = 1 << 51
)

// VehicleSpeed is the Speed of a vehicle in miles per hour, held as a whole
// number of half-miles per hour from 1 to 2^52 (SRD 5.2.1, "Airborne and
// Waterborne Vehicles", p. 101). A host rounds an adjusted speed down to a
// multiple of 0.5 mph before building one. The zero VehicleSpeed is invalid; see
// IsZero. Comparable and safe for concurrent use.
type VehicleSpeed struct{ halfMiles int64 }

// NewVehicleSpeed returns the speed of halfMilesPerHour half-miles per hour. It
// returns a *ExplorationError (matching srderr.ErrInvalidInput) for 0, a
// negative value or a value above 2^52. It never panics.
func NewVehicleSpeed(halfMilesPerHour int64) (VehicleSpeed, error) {
	if halfMilesPerHour < 1 || halfMilesPerHour > maxVehicleHalfMiles {
		return VehicleSpeed{}, &ExplorationError{Input: ExplorationVehicleSpeed}
	}
	return VehicleSpeed{halfMiles: halfMilesPerHour}, nil
}

// MustVehicleSpeed is NewVehicleSpeed for constants known to be valid. It panics
// with the error NewVehicleSpeed returns, and only then.
func MustVehicleSpeed(halfMilesPerHour int64) VehicleSpeed {
	s, err := NewVehicleSpeed(halfMilesPerHour)
	if err != nil {
		panic(err)
	}
	return s
}

// VehicleSpeedFromMilesPerHour converts a speed in miles per hour, such as a
// decoded content value. It accepts only a finite positive multiple of 0.5 at
// most 2^51 and returns a *ExplorationError (matching srderr.ErrInvalidInput)
// otherwise (NaN, infinity, 0, a negative value, 0.3, a value above 2^51). It
// never panics.
func VehicleSpeedFromMilesPerHour(mph float64) (VehicleSpeed, error) {
	half := mph * 2
	// NaN and the infinities fail the range test or the whole-number test.
	if !(mph > 0 && mph <= maxVehicleMiles) || half != math.Trunc(half) {
		return VehicleSpeed{}, &ExplorationError{Input: ExplorationVehicleSpeed}
	}
	return VehicleSpeed{halfMiles: int64(half)}, nil
}

// HalfMilesPerHour returns the speed in half-miles per hour.
func (s VehicleSpeed) HalfMilesPerHour() int64 { return s.halfMiles }

// MilesPerHour returns the speed in miles per hour, exact for every valid speed.
func (s VehicleSpeed) MilesPerHour() float64 { return float64(s.halfMiles) / 2 }

// IsZero reports whether s is the invalid zero VehicleSpeed.
func (s VehicleSpeed) IsZero() bool { return s.halfMiles == 0 }
