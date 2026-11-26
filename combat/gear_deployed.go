package combat

import (
	"slices"
	"strings"
)

// DeployedGearID is the id of a piece of deployed gear, supplied by the host. It is a
// comparable value, not trimmed or case-folded. The zero DeployedGearID (the empty string) is
// invalid. Safe for concurrent use.
type DeployedGearID string

// NewDeployedGearID returns id unchanged. It returns a *GearError (matching
// srderr.ErrInvalidInput) with GearProblemBlankID when strings.TrimSpace(id) is empty. It
// never panics.
func NewDeployedGearID(id string) (DeployedGearID, error) {
	if strings.TrimSpace(id) == "" {
		return "", gearInvalid(GearProblemBlankID)
	}
	return DeployedGearID(id), nil
}

// MustDeployedGearID is NewDeployedGearID for literals known to be valid. It panics with the
// error NewDeployedGearID returns, and only then.
func MustDeployedGearID(id string) DeployedGearID {
	v, err := NewDeployedGearID(id)
	if err != nil {
		panic(err)
	}
	return v
}

// TurnStamp names the turn in progress: its round, counting from 1, and the creature whose
// turn it is (SRD 5.2.1, "Combat Step by Step", p. 13). It is the key of the once-per-turn
// rules and the time of Oil. It is comparable and immutable; the zero TurnStamp is invalid.
// Safe for concurrent use.
type TurnStamp struct {
	round     int
	combatant CombatantID
}

// NewTurnStamp returns the stamp of round and combatant. It returns a *GearError (matching
// srderr.ErrInvalidInput) with GearProblemNegativeRound for a round below 1, or
// GearProblemBlankID for a zero combatant. It never panics.
func NewTurnStamp(round int, combatant CombatantID) (TurnStamp, error) {
	switch {
	case round < 1:
		return TurnStamp{}, gearInvalid(GearProblemNegativeRound)
	case combatant.IsZero():
		return TurnStamp{}, gearInvalid(GearProblemBlankID)
	}
	return TurnStamp{round, combatant}, nil
}

// MustTurnStamp is NewTurnStamp for values known to be valid. It panics with the error
// NewTurnStamp returns, and only then.
func MustTurnStamp(round int, combatant CombatantID) TurnStamp {
	s, err := NewTurnStamp(round, combatant)
	if err != nil {
		panic(err)
	}
	return s
}

// IsZero reports whether s is the invalid zero TurnStamp.
func (s TurnStamp) IsZero() bool { return s.round == 0 }

// Round returns the round, 1 or more.
func (s TurnStamp) Round() int { return s.round }

// IsCombatant reports whether it is id's turn.
func (s TurnStamp) IsCombatant(id CombatantID) bool { return !s.IsZero() && s.combatant == id }

// DeployedGearState is the closed set of states deployed gear can be in. Variants:
// GearInForce, GearArmed, GearSprung, OilUnlit, OilBurning.
type DeployedGearState interface{ isDeployedGearState() }

// PouredOilState is the closed set of states poured Oil can be in, each also a
// DeployedGearState. Variants: OilUnlit, OilBurning.
type PouredOilState interface {
	DeployedGearState
	isPouredOilState()
}

// GearInForce: Caltrops or Ball Bearings lying on the ground.
type GearInForce struct{}

// GearArmed: a Hunting Trap that is set.
type GearArmed struct{}

// GearSprung: a Hunting Trap that has snapped shut.
type GearSprung struct{}

// OilUnlit: poured Oil that has not been lit.
type OilUnlit struct{}

// OilBurning: poured Oil that burns until the end of the turn it burns through (SRD 5.2.1,
// "Oil", p. 98). The zero OilBurning is invalid; only DeployedGears.Light makes one.
type OilBurning struct{ burnsThrough TurnStamp }

// BurnsThrough returns the turn whose end the Oil burns through.
func (o OilBurning) BurnsThrough() TurnStamp { return o.burnsThrough }

func (GearInForce) isDeployedGearState() {}
func (GearArmed) isDeployedGearState()   {}
func (GearSprung) isDeployedGearState()  {}
func (OilUnlit) isDeployedGearState()    {}
func (OilBurning) isDeployedGearState()  {}
func (OilUnlit) isPouredOilState()       {}
func (OilBurning) isPouredOilState()     {}

// DeployedGear is a piece of gear set out on the ground: id, kind, deployer and state. It is
// comparable on all four and immutable. The zero DeployedGear is invalid; DeployedGears.Deploy
// makes one. Safe for concurrent use.
type DeployedGear struct {
	id       DeployedGearID
	kind     DeployedGearKind
	deployer CombatantID
	state    DeployedGearState
}

// IsZero reports whether g is the invalid zero DeployedGear.
func (g DeployedGear) IsZero() bool { return g.id == "" }

// ID returns the id of the gear.
func (g DeployedGear) ID() DeployedGearID { return g.id }

// Kind returns the kind of the gear.
func (g DeployedGear) Kind() DeployedGearKind { return g.kind }

// IsDeployedBy reports whether id set the gear out.
func (g DeployedGear) IsDeployedBy(id CombatantID) bool { return !g.IsZero() && g.deployer == id }

// State returns the state of the gear.
func (g DeployedGear) State() DeployedGearState { return g.state }

// base returns the event base of the gear: the deployer and, if given, another creature.
func (g DeployedGear) base(other CombatantID) gearEventBase { return gearEventBase{g.deployer, other} }

// DeployedGearsTransition is the closed set of outcomes of a change to DeployedGears.
// Variants: DeployedGearsApplied, DeployedGearsRejected.
type DeployedGearsTransition interface{ isDeployedGearsTransition() }

// DeployedGearsApplied: the change took effect. The set it holds is the new state.
type DeployedGearsApplied struct {
	gears  DeployedGears
	events []GearEvent
}

// Gears returns the new set.
func (t DeployedGearsApplied) Gears() DeployedGears { return t.gears }

// Events returns a copy of the events, in the order they happened.
func (t DeployedGearsApplied) Events() []GearEvent { return slices.Clone(t.events) }

// DeployedGearsRejected: the rules refuse the change and nothing happened.
type DeployedGearsRejected struct{ reason GearRejection }

// Reason returns why.
func (t DeployedGearsRejected) Reason() GearRejection { return t.reason }

func (DeployedGearsApplied) isDeployedGearsTransition()  {}
func (DeployedGearsRejected) isDeployedGearsTransition() {}

// DeployedGears is the set of gear set out in a fight, in deployment order. It is immutable:
// every change returns a new set. The zero value is the empty set. Compare with Equal. Safe
// for concurrent use.
type DeployedGears struct{ items []DeployedGear }

// NewDeployedGears returns the set of gears in the given order, copying the slice. It
// returns an error matching srderr.ErrInvalidInput for a zero gear or two gears with one id.
// It returns a LimitExceededError for more than 256 gears (plan P10), before it reads one. It never panics.
func NewDeployedGears(gears []DeployedGear) (DeployedGears, error) {
	if err := overCap("deployed gears", len(gears), maxInputList); err != nil {
		return DeployedGears{}, err
	}
	for i, g := range gears {
		if g.IsZero() || slices.ContainsFunc(gears[:i], func(o DeployedGear) bool { return o.id == g.id }) {
			return DeployedGears{}, invalidInput("deployed gears need non-zero gears with distinct ids")
		}
	}
	return DeployedGears{slices.Clone(gears)}, nil
}

// Gears returns a copy of the gears in deployment order.
func (s DeployedGears) Gears() []DeployedGear { return slices.Clone(s.items) }

// Len returns the number of gears.
func (s DeployedGears) Len() int { return len(s.items) }

// Find returns the gear with the id; ok is false when there is none.
func (s DeployedGears) Find(id DeployedGearID) (gear DeployedGear, ok bool) {
	i := s.index(id)
	if i < 0 {
		return DeployedGear{}, false
	}
	return s.items[i], true
}

// Equal reports whether s and o hold the same gears, whatever the order they were set out in.
func (s DeployedGears) Equal(o DeployedGears) bool {
	return len(s.items) == len(o.items) && !slices.ContainsFunc(s.items, func(g DeployedGear) bool { return !slices.Contains(o.items, g) })
}

func (s DeployedGears) index(id DeployedGearID) int {
	return slices.IndexFunc(s.items, func(g DeployedGear) bool { return g.id == id })
}

// with returns the set with item i replaced by g, or removed when g is the zero gear.
func (s DeployedGears) with(i int, g DeployedGear) DeployedGears {
	items := slices.Clone(s.items)
	if g.IsZero() {
		return DeployedGears{slices.Delete(items, i, i+1)}
	}
	items[i] = g
	return DeployedGears{items}
}

func appliedGears(s DeployedGears, events ...GearEvent) DeployedGearsTransition {
	return DeployedGearsApplied{s, events}
}

func rejectedGears(r GearRejection) DeployedGearsTransition { return DeployedGearsRejected{r} }

// Deploy sets out a gear of kind for deployer, distanceFeet from it, on the ground. It
// returns a DeployedGearsApplied with the new set and one DeployedGearPlaced, or a
// DeployedGearsRejected with the first of idInUse, outOfReach, groundNotLevel that applies.
// The initial state is GearArmed for the Hunting Trap, OilUnlit for Oil and GearInForce for
// Caltrops and Ball Bearings (SRD 5.2.1, pp. 96-98).
//
// It returns a *GearError (matching srderr.ErrInvalidInput) for a blank id, a zero deployer
// (GearProblemBlankID), a kind or ground that is not valid (GearProblemInvalidValue) or a
// negative distance (GearProblemNegativeDistance). A set that already holds 256 gears is
// GearProblemLimitExceeded (plan P10). It never panics.
func (s DeployedGears) Deploy(id DeployedGearID, kind DeployedGearKind, deployer CombatantID,
	distanceFeet int, ground GroundLevel) (DeployedGearsTransition, error) {
	switch {
	case strings.TrimSpace(string(id)) == "" || deployer.IsZero():
		return nil, gearInvalid(GearProblemBlankID)
	case !kind.Valid() || !ground.Valid():
		return nil, gearInvalid(GearProblemInvalidValue)
	case distanceFeet < 0:
		return nil, gearInvalid(GearProblemNegativeDistance)
	case len(s.items) >= maxInputList:
		return nil, gearInvalid(GearProblemLimitExceeded)
	}
	p := DeployedGearProfileFor(kind)
	switch {
	case s.index(id) >= 0:
		return rejectedGears(GearRejectionIDInUse), nil
	case !p.reach.Allows(distanceFeet):
		return rejectedGears(GearRejectionOutOfReach), nil
	case p.levelGround && ground == GroundLevelNotLevel:
		return rejectedGears(GearRejectionGroundNotLevel), nil
	}
	var state DeployedGearState
	switch kind {
	case DeployedGearKindCaltrops, DeployedGearKindBallBearings:
		state = GearInForce{}
	case DeployedGearKindPouredOil:
		state = OilUnlit{}
	case DeployedGearKindHuntingTrap:
		state = GearArmed{}
	}
	g := DeployedGear{id, kind, deployer, state}
	return appliedGears(DeployedGears{append(slices.Clone(s.items), g)}, DeployedGearPlaced{g.base(""), id, kind}), nil
}

// Recover gathers up Caltrops or Ball Bearings (SRD 5.2.1, p. 96: "It takes 10 minutes to
// recover"; the engine reads no clock, see DeployedGearProfile.RecoveryMinutes). It returns a
// DeployedGearsApplied with the gear removed and one DeployedGearRecovered, or a
// DeployedGearsRejected with unknownGear or notRecoverable (any other kind). It never
// panics.
func (s DeployedGears) Recover(id DeployedGearID) DeployedGearsTransition {
	i := s.index(id)
	switch {
	case i < 0:
		return rejectedGears(GearRejectionUnknownGear)
	case DeployedGearProfileFor(s.items[i].kind).recoveryMinutes == 0:
		return rejectedGears(GearRejectionNotRecoverable)
	}
	g := s.items[i]
	return appliedGears(s.with(i, DeployedGear{}), DeployedGearRecovered{g.base(""), id, g.kind})
}

// Light lights unlit poured Oil at stamp: it burns through the end of the turn two rounds
// later of the same creature (SRD 5.2.1, "Oil", p. 98: "until the end of the turn 2 rounds
// from when the oil was lit"). The round saturates at the largest int. It returns a
// DeployedGearsApplied with one PouredOilLit, or a DeployedGearsRejected with the first of
// unknownGear, notPouredOil, alreadyLit that applies.
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for the
// zero stamp. It never panics.
func (s DeployedGears) Light(id DeployedGearID, stamp TurnStamp) (DeployedGearsTransition, error) {
	if stamp.IsZero() {
		return nil, gearInvalid(GearProblemInvalidValue)
	}
	i := s.index(id)
	if i < 0 {
		return rejectedGears(GearRejectionUnknownGear), nil
	}
	g := s.items[i]
	switch {
	case g.kind != DeployedGearKindPouredOil:
		return rejectedGears(GearRejectionNotPouredOil), nil
	case g.state != DeployedGearState(OilUnlit{}):
		return rejectedGears(GearRejectionAlreadyLit), nil
	}
	through := TurnStamp{satAdd(stamp.round, 2), stamp.combatant}
	lit := DeployedGear{g.id, g.kind, g.deployer, OilBurning{through}}
	return appliedGears(s.with(i, lit), PouredOilLit{g.base(stamp.combatant), id, through}), nil
}

// Expire burns out every burning Oil whose turn has ended: the Oil burns through endedTurn,
// or through a round before it. Each is removed, with one PouredOilBurnedOut in deployment
// order. Other gear is unchanged; when nothing burns out the set is returned as it is, with
// no events. The host calls it at the end of every turn (SRD 5.2.1, "Oil", p. 98).
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for the
// zero stamp. It never panics.
func (s DeployedGears) Expire(endedTurn TurnStamp) (DeployedGearsApplied, error) {
	if endedTurn.IsZero() {
		return DeployedGearsApplied{}, gearInvalid(GearProblemInvalidValue)
	}
	var kept []DeployedGear
	var events []GearEvent
	for _, g := range s.items {
		if o, ok := g.state.(OilBurning); ok && (o.burnsThrough == endedTurn || o.burnsThrough.round < endedTurn.round) {
			events = append(events, PouredOilBurnedOut{g.base(o.burnsThrough.combatant), g.id})
			continue
		}
		kept = append(kept, g)
	}
	if len(events) == 0 {
		return DeployedGearsApplied{s, nil}, nil
	}
	return DeployedGearsApplied{DeployedGears{kept}, events}, nil
}

// Reset sets a sprung Hunting Trap again, which is a Utilize action (SRD 5.2.1, "Hunting
// Trap", p. 97; interpretation, Q-3). It returns a DeployedGearsApplied with the trap armed
// and one HuntingTrapReset, or a DeployedGearsRejected with unknownGear, notSprung (an armed
// trap, or gear that is not a Hunting Trap) or trapOccupied (a hold in holds has the trap's
// id). It never panics.
func (s DeployedGears) Reset(id DeployedGearID, holds GearHolds) DeployedGearsTransition {
	i := s.index(id)
	if i < 0 {
		return rejectedGears(GearRejectionUnknownGear)
	}
	g := s.items[i]
	switch {
	case g.state != DeployedGearState(GearSprung{}):
		return rejectedGears(GearRejectionNotSprung)
	case holds.holdsTrap(id):
		return rejectedGears(GearRejectionTrapOccupied)
	}
	g.state = GearArmed{}
	return appliedGears(s.with(i, g), HuntingTrapReset{g.base(""), id})
}

// Spring closes an armed Hunting Trap on creature, whether or not the creature was caught.
// It returns a DeployedGearsApplied with the trap sprung and one HuntingTrapSprung, or a
// DeployedGearsRejected with unknownGear, trapAlreadySprung (a sprung
// trap) or triggerNotApplicable (gear that is not a Hunting Trap).
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemBlankID) for a zero
// creature. It never panics.
func (s DeployedGears) Spring(id DeployedGearID, creature CombatantID) (DeployedGearsTransition, error) {
	if creature.IsZero() {
		return nil, gearInvalid(GearProblemBlankID)
	}
	i := s.index(id)
	if i < 0 {
		return rejectedGears(GearRejectionUnknownGear), nil
	}
	g := s.items[i]
	switch g.state.(type) {
	case GearArmed:
	case GearSprung:
		return rejectedGears(GearRejectionTrapAlreadySprung), nil
	default:
		return rejectedGears(GearRejectionTriggerNotApplicable), nil
	}
	g.state = GearSprung{}
	return appliedGears(s.with(i, g), HuntingTrapSprung{g.base(creature), id}), nil
}
