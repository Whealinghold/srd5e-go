package character

import "time"

// FuelKind is a fuel a light source burns: a Candle, a Torch or a flask of Oil
// (SRD 5.2.1, "Candle", p. 96; "Torch", p. 100; "Oil", p. 98). The zero FuelKind is
// invalid. Comparable and safe for concurrent use.
type FuelKind int

// The fuel kinds.
const (
	// FuelCandle burns for 1 hour.
	FuelCandle FuelKind = iota + 1
	// FuelTorch burns for 1 hour.
	FuelTorch
	// FuelOilFlask burns for 6 hours in a Lamp or Lantern.
	FuelOilFlask
)

// Valid reports whether k is a candle, a torch or an oil flask.
func (k FuelKind) Valid() bool { return k >= FuelCandle && k <= FuelOilFlask }

// String returns "candle", "torch", "oil_flask" or "unknown".
func (k FuelKind) String() string {
	switch k {
	case FuelCandle:
		return "candle"
	case FuelTorch:
		return "torch"
	case FuelOilFlask:
		return "oil_flask"
	}
	return "unknown"
}

// Capacity returns how long a fresh supply of k burns: 1 hour for a candle and a
// torch, 6 hours for an oil flask. An invalid kind gives 0.
func (k FuelKind) Capacity() time.Duration {
	switch k {
	case FuelCandle, FuelTorch:
		return time.Hour
	case FuelOilFlask:
		return 6 * time.Hour
	}
	return 0
}

// LightFuelEvent is the closed family of events of a FuelSupply. It is an Event.
// Members: LightFuelExhausted.
type LightFuelEvent interface {
	Event
	isLightFuelEvent()
}

// LightFuelExhausted is the event of a supply of fuel burning out.
type LightFuelExhausted struct {
	id   Identifier
	kind FuelKind
}

// CharacterID returns the id of the Character whose light source this is.
func (e LightFuelExhausted) CharacterID() Identifier { return e.id }

// Kind returns the kind of fuel that burned out.
func (e LightFuelExhausted) Kind() FuelKind { return e.kind }

func (LightFuelExhausted) isEvent()          {}
func (LightFuelExhausted) isLightFuelEvent() {}

// FuelSupply is the burn time left in one Candle, Torch or oil flask of a
// Character. It tracks time burned, not whether the source is lit; burn time is
// additive and need not be consecutive (SRD 5.2.1, "Oil", "Fuel", p. 99). It is
// immutable: Burn returns a new value. Build it with NewFreshFuelSupply or
// RestoreFuelSupply; the zero value is invalid. Comparable and safe for
// concurrent use.
type FuelSupply struct {
	id        Identifier
	kind      FuelKind
	remaining time.Duration
}

// RestoreFuelSupply returns a supply of kind with remaining burn time left, for
// reloading a part-burned one. It returns a *ExplorationError (matching
// srderr.ErrInvalidInput) for the zero Identifier, an invalid kind, or a remaining
// time below zero or above the kind's Capacity. Zero remaining gives a spent supply
// and no event. It never panics.
func RestoreFuelSupply(id Identifier, kind FuelKind, remaining time.Duration) (FuelSupply, error) {
	switch {
	case id.IsZero():
		return FuelSupply{}, &ExplorationError{Input: ExplorationCharacterID}
	case !kind.Valid():
		return FuelSupply{}, &ExplorationError{Input: ExplorationFuelKind}
	case remaining < 0 || remaining > kind.Capacity():
		return FuelSupply{}, &ExplorationError{Input: ExplorationFuelRemaining}
	}
	return FuelSupply{id: id, kind: kind, remaining: remaining}, nil
}

// NewFreshFuelSupply returns a full supply of kind, equal to RestoreFuelSupply at
// the kind's Capacity. It returns a *ExplorationError for the zero Identifier or an
// invalid kind. It never panics.
func NewFreshFuelSupply(id Identifier, kind FuelKind) (FuelSupply, error) {
	return RestoreFuelSupply(id, kind, kind.Capacity())
}

// CharacterID returns the id of the Character whose light source this is.
func (s FuelSupply) CharacterID() Identifier { return s.id }

// Kind returns the kind of fuel.
func (s FuelSupply) Kind() FuelKind { return s.kind }

// Remaining returns the burn time left.
func (s FuelSupply) Remaining() time.Duration { return s.remaining }

// FuelBurnResult is the closed outcome of Burn. Variants: FuelBurnContinues,
// FuelBurnExhausted.
type FuelBurnResult interface{ isFuelBurnResult() }

// FuelBurnContinues is the outcome of a burn that leaves fuel: the new supply.
type FuelBurnContinues struct{ supply FuelSupply }

// Supply returns the supply after the burn.
func (r FuelBurnContinues) Supply() FuelSupply { return r.supply }

// FuelBurnExhausted is the outcome of a burn that used up the fuel, or of a burn on
// a supply already spent.
type FuelBurnExhausted struct {
	supply    FuelSupply
	burnedFor time.Duration
	event     LightFuelExhausted
	raised    bool
}

// Supply returns the spent supply.
func (r FuelBurnExhausted) Supply() FuelSupply { return r.supply }

// BurnedFor returns the time actually burned: the old remaining time, never more
// than the fuel held, and 0 on an already spent supply.
func (r FuelBurnExhausted) BurnedFor() time.Duration { return r.burnedFor }

// Event returns the LightFuelExhausted event, and false when the supply was already
// spent so no event was produced.
func (r FuelBurnExhausted) Event() (LightFuelExhausted, bool) { return r.event, r.raised }

func (FuelBurnContinues) isFuelBurnResult() {}
func (FuelBurnExhausted) isFuelBurnResult() {}

// Burn burns duration of the supply. A negative duration is checked first and
// returns a *ExplorationError (matching srderr.ErrInvalidInput). A supply already
// spent gives a FuelBurnExhausted with nothing burned, unchanged and no event. A
// duration below the remaining time gives a FuelBurnContinues (a zero duration
// leaves the supply unchanged); an equal or greater one gives a FuelBurnExhausted
// burning only what was left, with one LightFuelExhausted. It never panics.
func (s FuelSupply) Burn(duration time.Duration) (FuelBurnResult, error) {
	switch {
	case duration < 0:
		return nil, &ExplorationError{Input: ExplorationBurnDuration}
	case s.remaining == 0:
		return FuelBurnExhausted{supply: s}, nil
	case duration < s.remaining:
		s.remaining -= duration
		return FuelBurnContinues{supply: s}, nil
	}
	burned := s.remaining
	s.remaining = 0
	return FuelBurnExhausted{supply: s, burnedFor: burned, event: LightFuelExhausted{id: s.id, kind: s.kind}, raised: true}, nil
}
