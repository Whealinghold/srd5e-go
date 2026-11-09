package character

import "github.com/Whealinghold/srd5e-go/srderr"

// ExplorationInput names the input of the exploration rules (travel, mounts,
// light sources and fuel) that a constructor or method rejected. The zero value
// is not an input.
type ExplorationInput int

// The inputs the exploration rules reject.
const (
	// ExplorationVehicleSpeed: a vehicle speed that is not a whole number of
	// half-miles per hour from 1 to 2^52, or a miles-per-hour value that is not a
	// finite positive multiple of 0.5 at most 2^51.
	ExplorationVehicleSpeed ExplorationInput = iota + 1
	// ExplorationLightRadius: a negative Bright Light or Dim Light distance.
	ExplorationLightRadius
	// ExplorationNoLight: an emission with neither Bright Light nor Dim Light.
	ExplorationNoLight
	// ExplorationLightShape: a light shape that is not radius or cone.
	ExplorationLightShape
	// ExplorationLightDistance: a negative distance from a light source.
	ExplorationLightDistance
	// ExplorationFuelKind: a fuel kind that is not candle, torch or oil flask.
	ExplorationFuelKind
	// ExplorationFuelRemaining: remaining fuel below zero or above the capacity.
	ExplorationFuelRemaining
	// ExplorationBurnDuration: a negative burn duration.
	ExplorationBurnDuration
	// ExplorationCharacterID: the zero Identifier.
	ExplorationCharacterID
)

// String returns the snake_case name of i, or "unknown".
func (i ExplorationInput) String() string {
	names := [...]string{"vehicle_speed", "light_radius", "no_light", "light_shape", "light_distance", "fuel_kind", "fuel_remaining", "burn_duration", "character_id"}
	if i < ExplorationVehicleSpeed || int(i) > len(names) {
		return "unknown"
	}
	return names[i-1]
}

// ExplorationError reports a rejected exploration input. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type ExplorationError struct{ Input ExplorationInput }

// Error renders the error. The text is not a contract.
func (e *ExplorationError) Error() string {
	return "character: invalid exploration input: " + e.Input.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *ExplorationError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ExplorationError with the same Input.
func (e *ExplorationError) Is(target error) bool {
	t, ok := target.(*ExplorationError)
	return ok && t != nil && e != nil && *t == *e
}
