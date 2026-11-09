package character

// TravelMode is how a group travels, a closed set. Variants: LandTravel (a group
// that chooses a pace) and VehicleTravel (a vehicle with a Speed, which chooses
// none) (SRD 5.2.1, "Vehicles", p. 12). Switch over it exhaustively.
type TravelMode interface{ isTravelMode() }

// LandTravel is travel on foot, riding, or in a land vehicle at a chosen pace.
// Build it with a keyed literal. An invalid Pace has no game effect and no speed.
type LandTravel struct{ Pace TravelPace }

// VehicleTravel is travel in a vehicle that has a Speed; it carries no pace. Build
// it with a keyed literal. The zero Speed has no game effect and no speed.
type VehicleTravel struct{ Speed VehicleSpeed }

func (LandTravel) isTravelMode()    {}
func (VehicleTravel) isTravelMode() {}
