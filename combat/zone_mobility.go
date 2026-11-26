package combat

import "github.com/Whealinghold/srd5e-go/content"

// MoveRefusal says why a zone cannot move. The reasons apply in the order of the
// constants (persistent-spell-zones FR-035, FR-038).
type MoveRefusal int

// The move refusals.
const (
	// RefusalZoneHasEnded: the zone has ended.
	RefusalZoneHasEnded MoveRefusal = iota + 1
	// RefusalNotMovable: the zone's mobility does not take this move.
	RefusalNotMovable
	// RefusalMovesWithOrigin: the zone follows its origin, the caster.
	RefusalMovesWithOrigin
	// RefusalStoppedForTurn: a creature moved into the zone and stopped it this turn.
	RefusalStoppedForTurn
	// RefusalWrongActionKind: the move is spent as the wrong kind of action.
	RefusalWrongActionKind
	// RefusalExceedsMaxDistance: the move is farther than the zone allows.
	RefusalExceedsMaxDistance
	// RefusalNotCastersTurn: the turn is not the caster's.
	RefusalNotCastersTurn
	// RefusalNotOnLaterTurn: the move is on the cast turn, and only later turns allow it.
	RefusalNotOnLaterTurn
	// RefusalDistanceNotPositive: the move is the zero value or has a distance of 0 or below (the constructors refuse it).
	RefusalDistanceNotPositive
)

func moveRefusalNames() []string {
	return []string{"zone_has_ended", "not_movable", "moves_with_origin", "stopped_for_turn",
		"wrong_action_kind", "exceeds_max_distance", "not_casters_turn", "not_on_later_turn", "distance_not_positive"}
}

// String returns the snake_case name of the refusal.
func (r MoveRefusal) String() string { return zoneName(moveRefusalNames(), int(r)) }

// MoveZoneResult is the closed set of outcomes of [MoveZone]. Variants:
// MoveZoneMoved, MoveZoneRefused, MoveZoneStaleTurn.
type MoveZoneResult interface{ isMoveZoneResult() }

// MoveZoneMoved is an accepted move: the zone is unchanged in state (the host owns
// the geometry), with its ZoneMoved event.
type MoveZoneMoved struct {
	Zone  PersistentZone
	Event ZoneMoved
}

// MoveZoneRefused is a move the zone refuses, with the first reason.
type MoveZoneRefused struct {
	Reason MoveRefusal
}

// MoveZoneStaleTurn is a turn earlier than the zone's cast turn.
type MoveZoneStaleTurn struct{}

func (MoveZoneMoved) isMoveZoneResult()     {}
func (MoveZoneRefused) isMoveZoneResult()   {}
func (MoveZoneStaleTurn) isMoveZoneResult() {}

// MoveZone applies a host-reported move at turn (persistent-spell-zones section
// 4.5). A caster move needs a MovedByCaster zone, the right action kind, a
// distance within the maximum, the caster's own turn and, when the zone says so,
// a later turn than its cast. A drift needs a DriftsAtTurnStart zone and a
// distance within its feet. A stationary zone refuses every move; a zone with a
// moving origin refuses a drift. A move of 0 feet or below (the zero
// value of either move) is refused with RefusalDistanceNotPositive. It never panics.
func MoveZone(zone PersistentZone, move ZoneMove, turn TurnMarker) MoveZoneResult {
	if zone.ended {
		return MoveZoneRefused{Reason: RefusalZoneHasEnded}
	}
	if turn.round < zone.cast.round {
		return MoveZoneStaleTurn{}
	}
	switch m := move.(type) {
	case CasterMove:
		if m.feet <= 0 {
			return MoveZoneRefused{Reason: RefusalDistanceNotPositive}
		}
		return moveByCaster(zone, m, turn)
	case DriftMove:
		if m.feet <= 0 {
			return MoveZoneRefused{Reason: RefusalDistanceNotPositive}
		}
		return driftZone(zone, m, turn)
	}
	return MoveZoneRefused{Reason: RefusalNotMovable}
}

// moveByCaster is the caster-move branch of MoveZone.
func moveByCaster(zone PersistentZone, m CasterMove, turn TurnMarker) MoveZoneResult {
	mv, ok := zone.facts.Mobility().(content.MovedByCaster)
	if !ok {
		return MoveZoneRefused{Reason: RefusalNotMovable}
	}
	switch {
	case zone.StoppedFor(turn):
		return MoveZoneRefused{Reason: RefusalStoppedForTurn}
	case m.How() != mv.Action():
		return MoveZoneRefused{Reason: RefusalWrongActionKind}
	case m.Feet() > mv.MaxFeet():
		return MoveZoneRefused{Reason: RefusalExceedsMaxDistance}
	case turn.active != zone.caster:
		return MoveZoneRefused{Reason: RefusalNotCastersTurn}
	case mv.LaterTurnsOnly() && turn == zone.cast:
		return MoveZoneRefused{Reason: RefusalNotOnLaterTurn}
	}
	return moved(zone, m.Feet(), m.How())
}

// driftZone is the drift branch of MoveZone.
func driftZone(zone PersistentZone, m DriftMove, turn TurnMarker) MoveZoneResult {
	switch mv := zone.facts.Mobility().(type) {
	case content.DriftsAtTurnStart:
		switch {
		case zone.StoppedFor(turn):
			return MoveZoneRefused{Reason: RefusalStoppedForTurn}
		case m.Feet() > mv.Feet():
			return MoveZoneRefused{Reason: RefusalExceedsMaxDistance}
		}
		return moved(zone, m.Feet(), 0)
	case content.MovesWithOrigin:
		return MoveZoneRefused{Reason: RefusalMovesWithOrigin}
	}
	return MoveZoneRefused{Reason: RefusalNotMovable}
}

// moved returns the accepted move, with its ZoneMoved event.
func moved(zone PersistentZone, feet int, how content.ZoneActionKind) MoveZoneMoved {
	ev := ZoneMoved{zoneEventBase: zoneEventBase{zone: zone.id}, caster: zone.caster, feet: feet, how: how}
	return MoveZoneMoved{Zone: zone, Event: ev}
}

// StartOfTurnResult is the closed set of outcomes of [ZoneStartOfTurn]. Variants:
// StartOfTurnReported, StartOfTurnStaleTurn.
type StartOfTurnResult interface{ isStartOfTurnResult() }

// StartOfTurnReported is the zone at the start of a turn, with the drift it owes
// when Drifts is true (only on the caster's turn of a drifting zone).
type StartOfTurnReported struct {
	Zone   PersistentZone
	Drift  ZoneDriftDue
	Drifts bool
}

// StartOfTurnStaleTurn is a turn earlier than the zone's cast turn.
type StartOfTurnStaleTurn struct{}

func (StartOfTurnReported) isStartOfTurnResult()  {}
func (StartOfTurnStaleTurn) isStartOfTurnResult() {}

// ZoneStartOfTurn reports the start of turn: the drift a zone owes on its caster's
// turn, and the clearing of a stop from an earlier turn (persistent-spell-zones
// FR-036, FR-043). It ends nothing. It never panics.
func ZoneStartOfTurn(zone PersistentZone, turn TurnMarker) StartOfTurnResult {
	if zone.ended {
		return StartOfTurnReported{Zone: zone}
	}
	if turn.round < zone.cast.round {
		return StartOfTurnStaleTurn{}
	}
	if zone.hasStopped && zone.stopped != turn {
		zone.hasStopped = false
	}
	if mv, ok := zone.facts.Mobility().(content.DriftsAtTurnStart); ok && turn.active == zone.caster {
		drift := ZoneDriftDue{zone: zone.id, caster: zone.caster, feet: mv.Feet(), direction: mv.Direction()}
		return StartOfTurnReported{Zone: zone, Drift: drift, Drifts: true}
	}
	return StartOfTurnReported{Zone: zone}
}
