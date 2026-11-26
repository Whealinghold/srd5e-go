package combat

import "context"

// CreatePersistentZoneCommand is the input of CreatePersistentZone.Run: the arguments of CreateZone. Construct it with a
// keyed literal.
type CreatePersistentZoneCommand CreateZoneCommand

// CreatePersistentZone is the use case of making a persistent zone from a cast spell (persistent-spell-zones.md RA-4):
// it wraps CreateZone and returns its CreatePersistentZoneResult. The result is completed (CreateZoneCreated, applied),
// zone_create_invalid (a rejection whose reasons are anything but a missing zone fact, invalid) or zone_no_facts
// (CreateNoZoneFacts, refused). Create it with NewCreatePersistentZone.
type CreatePersistentZone struct{ in instrumenter }

// NewCreatePersistentZone returns the use case. Options: WithTelemetry, WithClock.
func NewCreatePersistentZone(opts ...Option) CreatePersistentZone {
	return CreatePersistentZone{newWired(opts)}
}

// Run makes the zone. It returns the errors described for the package.
func (u CreatePersistentZone) Run(ctx context.Context, cmd CreatePersistentZoneCommand) (CreatePersistentZoneResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationCreatePersistentZone, spanCreatePersistentZone},
		func() (CreatePersistentZoneResult, error) { return CreateZone(CreateZoneCommand(cmd)), nil },
		func(r CreatePersistentZoneResult) description {
			switch v := r.(type) {
			case CreateZoneCreated:
				return appliedWith(valueResultCompleted, v.Event)
			case CreateZoneRejected:
				if len(v.Reasons) == 1 && v.Reasons[0] == CreateNoZoneFacts {
					return refusedWith(valueResultZoneNoFacts)
				}
				return invalidWith(valueResultZoneCreateInvalid)
			default:
				panic(unreachableError("create zone result", r))
			}
		})
}

// ZoneTriggerRequest is what AssessPersistentZoneTrigger asks of the owner. Variants: AssessTrigger,
// ApplyOptionalEffect, RecordCastApplication.
type ZoneTriggerRequest interface{ isZoneTriggerRequest() }

// AssessTrigger: match a reported event against the zone's rules (AssessZoneTrigger).
type AssessTrigger struct {
	Zone  PersistentZone
	Turn  TurnMarker
	Event ZoneTriggerEvent
}

// ApplyOptionalEffect: record an optional effect the caster chose to apply (ApplyOptionalZoneEffect).
type ApplyOptionalEffect struct {
	Zone PersistentZone
	Turn TurnMarker
	Due  ZoneEffectDue
}

// RecordCastApplication: record that a creature was affected at casting (RecordZoneCastApplication).
type RecordCastApplication struct {
	Zone     PersistentZone
	Creature CombatantID
	Turn     TurnMarker
}

func (AssessTrigger) isZoneTriggerRequest()         {}
func (ApplyOptionalEffect) isZoneTriggerRequest()   {}
func (RecordCastApplication) isZoneTriggerRequest() {}

// AssessPersistentZoneTriggerCommand is the input of AssessPersistentZoneTrigger.Run. Construct it with a keyed literal.
type AssessPersistentZoneTriggerCommand struct{ Request ZoneTriggerRequest }

// AssessPersistentZoneTriggerResult is the closed set of answers of AssessPersistentZoneTrigger.Run: the owner's own
// results, unchanged. Variants: AssessZoneTriggerDue, AssessZoneTriggerEnded, AssessZoneTriggerStaleTurn,
// OptionalZoneEffectApplied, OptionalZoneEffectRefused, RecordCastApplicationRecorded, RecordCastApplicationZoneEnded,
// RecordCastApplicationNotOncePerTurn, RecordCastApplicationStaleTurn, AssessZoneTriggerLimitExceeded,
// RecordCastApplicationLimitExceeded.
type AssessPersistentZoneTriggerResult interface{ isAssessPersistentZoneTriggerResult() }

func (AssessZoneTriggerDue) isAssessPersistentZoneTriggerResult()                {}
func (AssessZoneTriggerEnded) isAssessPersistentZoneTriggerResult()              {}
func (AssessZoneTriggerStaleTurn) isAssessPersistentZoneTriggerResult()          {}
func (OptionalZoneEffectApplied) isAssessPersistentZoneTriggerResult()           {}
func (OptionalZoneEffectRefused) isAssessPersistentZoneTriggerResult()           {}
func (RecordCastApplicationRecorded) isAssessPersistentZoneTriggerResult()       {}
func (RecordCastApplicationZoneEnded) isAssessPersistentZoneTriggerResult()      {}
func (RecordCastApplicationNotOncePerTurn) isAssessPersistentZoneTriggerResult() {}
func (RecordCastApplicationStaleTurn) isAssessPersistentZoneTriggerResult()      {}
func (AssessZoneTriggerLimitExceeded) isAssessPersistentZoneTriggerResult()      {}
func (RecordCastApplicationLimitExceeded) isAssessPersistentZoneTriggerResult()  {}

// AssessPersistentZoneTrigger is the use case of assessing what a zone does when something happens in it
// (persistent-spell-zones.md RA-4): it wraps AssessZoneTrigger, ApplyOptionalZoneEffect and RecordZoneCastApplication
// (the spec's AssessZoneTrigger, renamed because that is the owner's function). The result is completed (applied),
// zone_refused (a zone that ended, a refused optional effect or a cast application a zone cannot hold: refused) or
// zone_stale_turn (a turn earlier than the zone's: invalid). Create it with NewAssessPersistentZoneTrigger.
type AssessPersistentZoneTrigger struct{ in instrumenter }

// NewAssessPersistentZoneTrigger returns the use case. Options: WithTelemetry, WithClock.
func NewAssessPersistentZoneTrigger(opts ...Option) AssessPersistentZoneTrigger {
	return AssessPersistentZoneTrigger{newWired(opts)}
}

// Run assesses the request. It returns an error matching srderr.ErrInvalidInput for a nil or unknown request, and the
// errors described for the package.
func (u AssessPersistentZoneTrigger) Run(ctx context.Context, cmd AssessPersistentZoneTriggerCommand) (AssessPersistentZoneTriggerResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationAssessZoneTrigger, spanAssessZoneTrigger},
		func() (AssessPersistentZoneTriggerResult, error) {
			switch v := cmd.Request.(type) {
			case AssessTrigger:
				return AssessZoneTrigger(v.Zone, v.Turn, v.Event).(AssessPersistentZoneTriggerResult), nil
			case ApplyOptionalEffect:
				return ApplyOptionalZoneEffect(v.Zone, v.Turn, v.Due).(AssessPersistentZoneTriggerResult), nil
			case RecordCastApplication:
				return RecordZoneCastApplication(v.Zone, v.Creature, v.Turn).(AssessPersistentZoneTriggerResult), nil
			}
			return nil, invalidInput("a zone trigger assessment needs a request")
		},
		func(r AssessPersistentZoneTriggerResult) description {
			switch v := r.(type) {
			case AssessZoneTriggerDue:
				return appliedWith(valueResultCompleted, widen(v.Events)...)
			case OptionalZoneEffectApplied, RecordCastApplicationRecorded:
				return appliedWith(valueResultCompleted)
			case AssessZoneTriggerEnded, OptionalZoneEffectRefused, RecordCastApplicationZoneEnded, RecordCastApplicationNotOncePerTurn:
				return refusedWith(valueResultZoneRefused)
			case AssessZoneTriggerStaleTurn, RecordCastApplicationStaleTurn:
				return invalidWith(valueResultZoneStaleTurn)
			case AssessZoneTriggerLimitExceeded, RecordCastApplicationLimitExceeded:
				return invalidWith(valueResultLimitExceeded)
			default:
				panic(unreachableError("zone trigger result", r))
			}
		})
}

// ZoneMoveRequest is what MovePersistentZone asks of the owner. Variants: MoveZoneRequest, StartOfTurnRequest.
type ZoneMoveRequest interface{ isZoneMoveRequest() }

// MoveZoneRequest: apply a host-reported move (MoveZone).
type MoveZoneRequest struct {
	Zone PersistentZone
	Move ZoneMove
	Turn TurnMarker
}

// StartOfTurnRequest: report the start of a turn to the zone (ZoneStartOfTurn).
type StartOfTurnRequest struct {
	Zone PersistentZone
	Turn TurnMarker
}

func (MoveZoneRequest) isZoneMoveRequest()    {}
func (StartOfTurnRequest) isZoneMoveRequest() {}

// MovePersistentZoneCommand is the input of MovePersistentZone.Run. Construct it with a keyed literal.
type MovePersistentZoneCommand struct{ Request ZoneMoveRequest }

// MovePersistentZoneResult is the closed set of answers of MovePersistentZone.Run: the owner's own results, unchanged.
// Variants: MoveZoneMoved, MoveZoneRefused, MoveZoneStaleTurn, StartOfTurnReported, StartOfTurnStaleTurn.
type MovePersistentZoneResult interface{ isMovePersistentZoneResult() }

func (MoveZoneMoved) isMovePersistentZoneResult()        {}
func (MoveZoneRefused) isMovePersistentZoneResult()      {}
func (MoveZoneStaleTurn) isMovePersistentZoneResult()    {}
func (StartOfTurnReported) isMovePersistentZoneResult()  {}
func (StartOfTurnStaleTurn) isMovePersistentZoneResult() {}

// MovePersistentZone is the use case of moving a zone and of the start of a turn in it (persistent-spell-zones.md
// RA-4): it wraps MoveZone and ZoneStartOfTurn. The result is completed (a move or a report, applied),
// zone_move_refused (refused) or zone_stale_turn (invalid). Create it with NewMovePersistentZone.
type MovePersistentZone struct{ in instrumenter }

// NewMovePersistentZone returns the use case. Options: WithTelemetry, WithClock.
func NewMovePersistentZone(opts ...Option) MovePersistentZone {
	return MovePersistentZone{newWired(opts)}
}

// Run applies the request. It returns an error matching srderr.ErrInvalidInput for a nil or unknown request, and the
// errors described for the package.
func (u MovePersistentZone) Run(ctx context.Context, cmd MovePersistentZoneCommand) (MovePersistentZoneResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationMovePersistentZone, spanMovePersistentZone},
		func() (MovePersistentZoneResult, error) {
			switch v := cmd.Request.(type) {
			case MoveZoneRequest:
				return MoveZone(v.Zone, v.Move, v.Turn).(MovePersistentZoneResult), nil
			case StartOfTurnRequest:
				return ZoneStartOfTurn(v.Zone, v.Turn).(MovePersistentZoneResult), nil
			}
			return nil, invalidInput("a zone move needs a request")
		},
		func(r MovePersistentZoneResult) description {
			switch v := r.(type) {
			case MoveZoneMoved:
				return appliedWith(valueResultCompleted, v.Event)
			case StartOfTurnReported:
				return appliedWith(valueResultCompleted)
			case MoveZoneRefused:
				return refusedWith(valueResultZoneMoveRefused)
			case MoveZoneStaleTurn, StartOfTurnStaleTurn:
				return invalidWith(valueResultZoneStaleTurn)
			default:
				panic(unreachableError("zone move result", r))
			}
		})
}

// ZoneEndRequest is what EndPersistentZone asks of the owner. Variants: EndZoneRequest, RoundStartedRequest,
// RecordDamageRequest.
type ZoneEndRequest interface{ isZoneEndRequest() }

// EndZoneRequest: end the zone for the reason (EndZone).
type EndZoneRequest struct {
	Zone   PersistentZone
	Reason ZoneEndReason
}

// RoundStartedRequest: report the start of a round to the zone (ZoneRoundStarted).
type RoundStartedRequest struct {
	Zone  PersistentZone
	Round int
}

// RecordDamageRequest: add damage dealt to the zone's budget (RecordZoneDamage).
type RecordDamageRequest struct {
	Zone   PersistentZone
	Amount int
}

func (EndZoneRequest) isZoneEndRequest()      {}
func (RoundStartedRequest) isZoneEndRequest() {}
func (RecordDamageRequest) isZoneEndRequest() {}

// EndPersistentZoneCommand is the input of EndPersistentZone.Run. Construct it with a keyed literal.
type EndPersistentZoneCommand struct{ Request ZoneEndRequest }

// EndPersistentZoneResult is the closed set of answers of EndPersistentZone.Run: the owner's own results, unchanged.
// Variants: EndZoneEnded, EndZoneAlreadyEnded, ZoneRoundResult, RecordDamageRecorded, RecordDamageNoBudget,
// RecordDamageZoneEnded.
type EndPersistentZoneResult interface{ isEndPersistentZoneResult() }

func (EndZoneEnded) isEndPersistentZoneResult()          {}
func (EndZoneAlreadyEnded) isEndPersistentZoneResult()   {}
func (ZoneRoundResult) isEndPersistentZoneResult()       {}
func (RecordDamageRecorded) isEndPersistentZoneResult()  {}
func (RecordDamageNoBudget) isEndPersistentZoneResult()  {}
func (RecordDamageZoneEnded) isEndPersistentZoneResult() {}

// EndPersistentZone is the use case of ending a zone, of a round starting and of damage dealt in it
// (persistent-spell-zones.md RA-4): it wraps EndZone, ZoneRoundStarted and RecordZoneDamage. The result is completed
// (applied, whether or not the zone ended), zone_already_ended or zone_refused (refused). Create it with
// NewEndPersistentZone.
type EndPersistentZone struct{ in instrumenter }

// NewEndPersistentZone returns the use case. Options: WithTelemetry, WithClock.
func NewEndPersistentZone(opts ...Option) EndPersistentZone { return EndPersistentZone{newWired(opts)} }

// Run applies the request. It returns an error matching srderr.ErrInvalidInput for a nil or unknown request, a round
// below the zone's cast round or a negative amount or a reason that is not an end reason, and the errors described for the package.
func (u EndPersistentZone) Run(ctx context.Context, cmd EndPersistentZoneCommand) (EndPersistentZoneResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationEndPersistentZone, spanEndPersistentZone},
		func() (EndPersistentZoneResult, error) {
			switch v := cmd.Request.(type) {
			case EndZoneRequest:
				res := EndZone(v.Zone, v.Reason)
				if _, bad := res.(EndZoneInvalidReason); bad {
					return nil, invalidInput("a zone ends for one of the six end reasons")
				}
				return res.(EndPersistentZoneResult), nil
			case RoundStartedRequest:
				r, err := ZoneRoundStarted(v.Zone, v.Round)
				return r, err
			case RecordDamageRequest:
				r, err := RecordZoneDamage(v.Zone, v.Amount)
				if err != nil {
					return nil, err
				}
				return r.(EndPersistentZoneResult), nil
			}
			return nil, invalidInput("ending a zone needs a request")
		},
		func(r EndPersistentZoneResult) description {
			switch v := r.(type) {
			case EndZoneEnded:
				return appliedWith(valueResultCompleted, v.Event)
			case ZoneRoundResult:
				return appliedWith(valueResultCompleted, widen(v.Events)...)
			case RecordDamageRecorded:
				return appliedWith(valueResultCompleted, widen(v.Events)...)
			case EndZoneAlreadyEnded:
				return refusedWith(valueResultZoneAlreadyEnded)
			case RecordDamageNoBudget, RecordDamageZoneEnded:
				return refusedWith(valueResultZoneRefused)
			default:
				panic(unreachableError("end zone result", r))
			}
		})
}
