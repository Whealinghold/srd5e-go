package combat

import "github.com/Whealinghold/srd5e-go/content"

// ZoneTriggerEvent is a host-reported event about one creature and a persistent
// zone (persistent-spell-zones Table E). Variants: CreatureEntered,
// CreatureEndedTurn, CreatureStartedTurn, ZoneMovedIntoSpace, CreatureTravelled.
// The host reports each from its own geometry; the event carries the creature's
// facts so the zone needs no lookup. Each variant embeds zoneTriggerBase, which
// carries those facts.
type ZoneTriggerEvent interface {
	isZoneTriggerEvent()
	// Facts returns the creature the event is about and its facts.
	Facts() ZoneCreatureFacts
}

// zoneTriggerBase carries the creature of every trigger event.
type zoneTriggerBase struct {
	facts ZoneCreatureFacts
}

func (zoneTriggerBase) isZoneTriggerEvent() {}

// Facts returns the creature the event is about and its facts.
func (b zoneTriggerBase) Facts() ZoneCreatureFacts { return b.facts }

// CreatureEntered is a creature's own arrival into the zone. It never matches a
// creature that is already inside when its turn starts, nor a zone that moved
// onto it (ZoneMovedIntoSpace).
type CreatureEntered struct{ zoneTriggerBase }

// NewCreatureEntered returns the event of f's creature entering the zone.
func NewCreatureEntered(f ZoneCreatureFacts) CreatureEntered {
	return CreatureEntered{zoneTriggerBase{facts: f}}
}

// CreatureEndedTurn is a creature ending its turn inside the zone.
type CreatureEndedTurn struct{ zoneTriggerBase }

// NewCreatureEndedTurn returns the event of f's creature ending its turn inside.
func NewCreatureEndedTurn(f ZoneCreatureFacts) CreatureEndedTurn {
	return CreatureEndedTurn{zoneTriggerBase{facts: f}}
}

// CreatureStartedTurn is a creature starting its turn inside the zone.
type CreatureStartedTurn struct{ zoneTriggerBase }

// NewCreatureStartedTurn returns the event of f's creature starting its turn inside.
func NewCreatureStartedTurn(f ZoneCreatureFacts) CreatureStartedTurn {
	return CreatureStartedTurn{zoneTriggerBase{facts: f}}
}

// ZoneMovedIntoSpace is a creature whose space the zone moved into (a drift, or
// the origin's move). It never satisfies CreatureEntered (SRD 5.2.1, "Moonbeam",
// p. 151).
type ZoneMovedIntoSpace struct{ zoneTriggerBase }

// NewZoneMovedIntoSpace returns the event of the zone moving into f's creature's space.
func NewZoneMovedIntoSpace(f ZoneCreatureFacts) ZoneMovedIntoSpace {
	return ZoneMovedIntoSpace{zoneTriggerBase{facts: f}}
}

// CreatureTravelled is a creature that travelled feetInside feet inside the zone
// in one reported segment (SRD 5.2.1, "Spike Growth", p. 164). The cause of the
// movement is not part of the event: forced and own movement count alike.
type CreatureTravelled struct {
	zoneTriggerBase
	feetInside int
}

// NewCreatureTravelled returns the event of f's creature travelling feetInside
// feet inside the zone.
//
// It returns an *ZoneInvalidError for a negative distance. It never panics.
func NewCreatureTravelled(f ZoneCreatureFacts, feetInside int) (CreatureTravelled, error) {
	if feetInside < 0 {
		return CreatureTravelled{}, &ZoneInvalidError{What: "feet travelled inside a zone"}
	}
	return CreatureTravelled{zoneTriggerBase: zoneTriggerBase{facts: f}, feetInside: feetInside}, nil
}

// FeetInside returns the feet travelled inside the zone in the segment.
func (e CreatureTravelled) FeetInside() int { return e.feetInside }

// ZoneMove is a move a host reports for a persistent zone. Variants: CasterMove
// (the zone moved with its caster's action), DriftMove (the zone drifted at the
// start of the caster's turn).
type ZoneMove interface{ isZoneMove() }

// CasterMove moves a zone by the caster's action: DistanceFeet, spent as how
// (content.ZoneActionKind). The zero value is invalid; build with [NewCasterMove].
type CasterMove struct {
	feet int
	how  content.ZoneActionKind
}

// NewCasterMove returns the move of feet feet spent as how.
//
// It returns an *ZoneInvalidError for a distance of 0 or below. It never panics.
func NewCasterMove(feet int, how content.ZoneActionKind) (CasterMove, error) {
	if feet <= 0 {
		return CasterMove{}, &ZoneInvalidError{What: "caster move distance"}
	}
	return CasterMove{feet: feet, how: how}, nil
}

// Feet returns the distance moved.
func (m CasterMove) Feet() int { return m.feet }

// How returns the action kind the move is spent as.
func (m CasterMove) How() content.ZoneActionKind { return m.how }

func (CasterMove) isZoneMove() {}

// DriftMove moves a zone by its drift at the start of its caster's turn. The zero
// value is invalid; build with [NewDriftMove].
type DriftMove struct {
	feet int
}

// NewDriftMove returns the drift of feet feet.
//
// It returns an *ZoneInvalidError for a distance of 0 or below. It never panics.
func NewDriftMove(feet int) (DriftMove, error) {
	if feet <= 0 {
		return DriftMove{}, &ZoneInvalidError{What: "drift move distance"}
	}
	return DriftMove{feet: feet}, nil
}

// Feet returns the distance moved.
func (m DriftMove) Feet() int { return m.feet }

func (DriftMove) isZoneMove() {}

// ZoneDriftDue is the drift a zone owes at the start of its caster's turn
// (SRD 5.2.1, "Cloudkill", p. 116). The host owns the geometry and reports the
// move as a DriftMove no further than Feet.
type ZoneDriftDue struct {
	zone      PersistentZoneID
	caster    CombatantID
	feet      int
	direction content.DriftDirectionRule
}

// ZoneID returns the zone that drifts.
func (d ZoneDriftDue) ZoneID() PersistentZoneID { return d.zone }

// Combatants returns the caster whose turn starts.
func (d ZoneDriftDue) Combatants() []CombatantID { return []CombatantID{d.caster} }

// Feet returns the distance the zone drifts.
func (d ZoneDriftDue) Feet() int { return d.feet }

// Direction returns the direction rule: away from the caster, or a direction the
// caster chooses.
func (d ZoneDriftDue) Direction() content.DriftDirectionRule { return d.direction }
