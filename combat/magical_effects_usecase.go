package combat

import "context"

// The use cases of this file record the operations of magical-effects.md (RA-1, FR-148, section 7): each wraps a pure
// service of that spec in one Instrumenter operation (combat-action-resolution.md FR-170) and returns the owner's own
// closed result unchanged. They take no Encounter and no ActionLedger, roll no die and call no port and no other use
// case; the state a result returns (effects, shape, order) is the host's, as the spec's Q-11 says. An error matching
// srderr.ErrInvalidInput from the owner, a nil request or an undefined placement is returned as it is and recorded on
// the span only; a cancelled ctx is returned unwrapped before any signal.

// RemoveCombatantCurseCommand is the input of RemoveCombatantCurse.Run: the arguments of CurseRemoval.Remove. Construct
// it with a keyed literal.
type RemoveCombatantCurseCommand struct {
	Creature CombatantID
	Effects  OngoingEffects
	Curse    OngoingEffectID
	Means    CurseRemovalMeans
}

// RemoveCombatantCurse is the use case of removing a curse (SRD 5.2.1, "Curses", p. 179): it wraps CurseRemoval.Remove
// and returns its RemoveCombatantCurseResult. The result is completed (RemoveCurseApplied, applied), curse_not_present or
// curse_not_a_curse (refused). Create it with NewRemoveCombatantCurse. Safe for concurrent use if its telemetry and clock
// are.
type RemoveCombatantCurse struct{ in instrumenter }

// NewRemoveCombatantCurse returns the use case. Options: WithTelemetry, WithClock.
func NewRemoveCombatantCurse(opts ...Option) RemoveCombatantCurse {
	return RemoveCombatantCurse{newWired(opts)}
}

// Run removes the curse. It returns the errors described for the package.
func (u RemoveCombatantCurse) Run(ctx context.Context, cmd RemoveCombatantCurseCommand) (RemoveCombatantCurseResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationRemoveCombatantCurse, spanRemoveCombatantCurse},
		func() (RemoveCombatantCurseResult, error) {
			return CurseRemoval{}.Remove(cmd.Creature, cmd.Effects, cmd.Curse, cmd.Means), nil
		},
		func(r RemoveCombatantCurseResult) description {
			switch v := r.(type) {
			case RemoveCurseApplied:
				return appliedWith(valueResultCompleted, widen(v.Events())...)
			case RemoveCurseNotPresent:
				return refusedWith(valueResultCurseNotPresent)
			case RemoveCurseNotACurse:
				return refusedWith(valueResultCurseNotACurse)
			default:
				panic(unreachableError("remove curse result", r))
			}
		})
}

// ShapeShiftingRequest is the change of form a ShiftCombatantShape call carries: one variant for each operation of
// ShapeShifting. Variants: ShiftRequest, RevertRequest, RevertOnDeathRequest.
type ShapeShiftingRequest interface{ isShapeShiftingRequest() }

// ShiftRequest: the creature shifts into a form (ShapeShifting.Shift).
type ShiftRequest struct{ Command ShiftShapeCommand }

// RevertRequest: the creature returns to its true form (ShapeShifting.Revert).
type RevertRequest struct{ Command RevertToTrueFormCommand }

// RevertOnDeathRequest: the host reports a creature that died (ShapeShifting.RevertOnDeath).
type RevertOnDeathRequest struct {
	Creature CombatantID
	State    ShapeShiftState
	Life     LifeState
}

func (ShiftRequest) isShapeShiftingRequest()         {}
func (RevertRequest) isShapeShiftingRequest()        {}
func (RevertOnDeathRequest) isShapeShiftingRequest() {}

// ShiftCombatantShapeCommand is the input of ShiftCombatantShape.Run. Construct it with a keyed literal.
type ShiftCombatantShapeCommand struct{ Request ShapeShiftingRequest }

// ShiftCombatantShapeResult is the closed set of answers of ShiftCombatantShape.Run: the owner's own results, unchanged,
// of the three operations. Variants: ShiftShapeApplied, ShiftShapeExceptionNotHeld, RevertToTrueFormApplied,
// RevertToTrueFormExceptionNotHeld, RevertOnDeathApplied, RevertOnDeathNotDead.
type ShiftCombatantShapeResult interface{ isShiftCombatantShapeResult() }

func (ShiftShapeApplied) isShiftCombatantShapeResult()                {}
func (ShiftShapeExceptionNotHeld) isShiftCombatantShapeResult()       {}
func (RevertToTrueFormApplied) isShiftCombatantShapeResult()          {}
func (RevertToTrueFormExceptionNotHeld) isShiftCombatantShapeResult() {}
func (RevertOnDeathApplied) isShiftCombatantShapeResult()             {}
func (RevertOnDeathNotDead) isShiftCombatantShapeResult()             {}

// ShiftCombatantShape is the use case of Shape-Shifting (SRD 5.2.1, "Shape-Shifting", p. 187): it wraps the three
// operations of ShapeShifting and returns the owner's result of the one asked for. The result is completed (every
// Applied, applied), shift_exception_not_held (a shift or a reversion whose rule names what is not held) or
// revert_not_dead (refused). Create it with NewShiftCombatantShape. Safe for concurrent use if its telemetry and clock
// are.
type ShiftCombatantShape struct{ in instrumenter }

// NewShiftCombatantShape returns the use case. Options: WithTelemetry, WithClock.
func NewShiftCombatantShape(opts ...Option) ShiftCombatantShape {
	return ShiftCombatantShape{newWired(opts)}
}

// Run makes the change of form. It returns an error matching srderr.ErrInvalidInput for a nil or unknown request and
// for the zero creature or effect the owner rejects, and the errors described for the package.
func (u ShiftCombatantShape) Run(ctx context.Context, cmd ShiftCombatantShapeCommand) (ShiftCombatantShapeResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationShiftCombatantShape, spanShiftCombatantShape},
		func() (ShiftCombatantShapeResult, error) {
			switch v := cmd.Request.(type) {
			case ShiftRequest:
				r, err := ShapeShifting{}.Shift(v.Command)
				if err != nil {
					return nil, err
				}
				return r.(ShiftCombatantShapeResult), nil
			case RevertRequest:
				r, err := ShapeShifting{}.Revert(v.Command)
				if err != nil {
					return nil, err
				}
				return r.(ShiftCombatantShapeResult), nil
			case RevertOnDeathRequest:
				return ShapeShifting{}.RevertOnDeath(v.Creature, v.State, v.Life).(ShiftCombatantShapeResult), nil
			}
			return nil, invalidInput("a shape change needs a request")
		},
		func(r ShiftCombatantShapeResult) description {
			switch v := r.(type) {
			case ShiftShapeApplied:
				return appliedWith(valueResultCompleted, widen(v.Events())...)
			case RevertToTrueFormApplied:
				return appliedWith(valueResultCompleted, widen(v.Events())...)
			case RevertOnDeathApplied:
				return appliedWith(valueResultCompleted, widen(v.Events())...)
			case ShiftShapeExceptionNotHeld, RevertToTrueFormExceptionNotHeld:
				return refusedWith(valueResultShiftExceptionNotHeld)
			case RevertOnDeathNotDead:
				return refusedWith(valueResultRevertNotDead)
			default:
				panic(unreachableError("shape change result", r))
			}
		})
}

// TeleportCombatantCommand is the input of TeleportCombatant.Run: the command Teleportation.Resolve takes, built by
// NewTeleportCommand. Construct it with a keyed literal.
type TeleportCombatantCommand struct{ Command TeleportCommand }

// TeleportCombatant is the use case of Teleportation (SRD 5.2.1, "Teleportation", p. 190): it wraps Teleportation.Resolve
// and returns its TeleportCombatantResult. The result is completed (an arrival at or near the destination, applied),
// teleport_destination_not_seen (refused) or teleport_invalid_touched_set (invalid, error status). Create it with
// NewTeleportCombatant. Safe for concurrent use if its telemetry and clock are.
type TeleportCombatant struct{ in instrumenter }

// NewTeleportCombatant returns the use case. Options: WithTelemetry, WithClock.
func NewTeleportCombatant(opts ...Option) TeleportCombatant { return TeleportCombatant{newWired(opts)} }

// Run resolves the teleportation. It returns an error matching srderr.ErrInvalidInput for a command not built by
// NewTeleportCommand, and the errors described for the package.
func (u TeleportCombatant) Run(ctx context.Context, cmd TeleportCombatantCommand) (TeleportCombatantResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationTeleportCombatant, spanTeleportCombatant},
		func() (TeleportCombatantResult, error) { return Teleportation{}.Resolve(cmd.Command) },
		func(r TeleportCombatantResult) description {
			switch v := r.(type) {
			case TeleportArrivedAtDestination:
				return appliedWith(valueResultCompleted, widen(v.Events())...)
			case TeleportArrivedNearDestination:
				return appliedWith(valueResultCompleted, widen(v.Events())...)
			case TeleportDestinationNotSeen:
				return refusedWith(valueResultTeleportDestinationNotSeen)
			case TeleportInvalidTouchedSet:
				return invalidWith(valueResultTeleportInvalidTouchedSet)
			default:
				panic(unreachableError("teleport result", r))
			}
		})
}

// ResolveEffectOrderCommand is the input of ResolveEffectOrder.Run: the arguments of SimultaneousEffects.Order. The
// lists are not changed. Construct it with a keyed literal.
type ResolveEffectOrderCommand struct {
	Holder  TurnHolder
	Pending []PendingEffectID
	Chosen  []PendingEffectID
}

// ResolveEffectOrder is the use case of Simultaneous Effects (SRD 5.2.1, "Simultaneous Effects", p. 187): it wraps
// SimultaneousEffects.Order and returns its ResolveEffectOrderResult. The result is completed
// (OrderSimultaneousEffectsOrdered, applied) or order_rejected (invalid, error status). Create it with
// NewResolveEffectOrder. Safe for concurrent use if its telemetry and clock are.
type ResolveEffectOrder struct{ in instrumenter }

// NewResolveEffectOrder returns the use case. Options: WithTelemetry, WithClock.
func NewResolveEffectOrder(opts ...Option) ResolveEffectOrder {
	return ResolveEffectOrder{newWired(opts)}
}

// Run validates the chosen order. It returns the errors described for the package.
func (u ResolveEffectOrder) Run(ctx context.Context, cmd ResolveEffectOrderCommand) (ResolveEffectOrderResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationResolveEffectOrder, spanResolveEffectOrder},
		func() (ResolveEffectOrderResult, error) {
			return SimultaneousEffects{}.Order(cmd.Holder, cmd.Pending, cmd.Chosen), nil
		},
		func(r ResolveEffectOrderResult) description {
			switch v := r.(type) {
			case OrderSimultaneousEffectsOrdered:
				return appliedWith(valueResultCompleted)
			case OrderSimultaneousEffectsRejected:
				if len(v.Problems()) == 1 && v.Problems()[0].Kind == OrderProblemLimitExceeded {
					return invalidWith(valueResultLimitExceeded)
				}
				return invalidWith(valueResultOrderRejected)
			default:
				panic(unreachableError("order result", r))
			}
		})
}

// AreaLocationRequest is the question an AssessAreaLocation call asks: one variant for each service. Variants:
// AssessMembershipRequest, ResolveOriginRequest.
type AreaLocationRequest interface{ isAreaLocationRequest() }

// AssessMembershipRequest: is the location in the area (AreaMembership.Assess).
type AssessMembershipRequest struct{ Facts AreaLocationFacts }

// ResolveOriginRequest: where does the origin end up (AreaOrigin.Resolve).
type ResolveOriginRequest struct{ Placement OriginPlacement }

func (AssessMembershipRequest) isAreaLocationRequest() {}
func (ResolveOriginRequest) isAreaLocationRequest()    {}

// AssessAreaLocationCommand is the input of AssessAreaLocation.Run. Construct it with a keyed literal.
type AssessAreaLocationCommand struct{ Request AreaLocationRequest }

// AssessAreaLocationResult is the closed set of answers of AssessAreaLocation.Run: the owner's own results, unchanged.
// Variants: AreaMembershipIncluded, AreaMembershipExcluded, AreaMembershipNoLinesConsidered, AreaOriginResolution.
type AssessAreaLocationResult interface{ isAssessAreaLocationResult() }

func (AreaMembershipIncluded) isAssessAreaLocationResult()          {}
func (AreaMembershipExcluded) isAssessAreaLocationResult()          {}
func (AreaMembershipNoLinesConsidered) isAssessAreaLocationResult() {}
func (AreaOriginResolution) isAssessAreaLocationResult()            {}

// AssessAreaLocation is the use case of the Area of Effect rules (SRD 5.2.1, "Area of Effect", p. 177): it wraps
// AreaMembership.Assess and AreaOrigin.Resolve and returns the owner's result of the one asked for. The result is
// completed (included, or a resolved origin, applied), area_location_excluded (an excluded location is an applied
// answer, not a refusal) or area_no_lines_considered (invalid, error status). Create it with NewAssessAreaLocation.
// Safe for concurrent use if its telemetry and clock are.
type AssessAreaLocation struct{ in instrumenter }

// NewAssessAreaLocation returns the use case. Options: WithTelemetry, WithClock.
func NewAssessAreaLocation(opts ...Option) AssessAreaLocation {
	return AssessAreaLocation{newWired(opts)}
}

// Run assesses the location or resolves the origin. It returns an error matching srderr.ErrInvalidInput for a nil or
// unknown request and for an undefined placement (the owner's zero resolution), and the errors described for the
// package.
func (u AssessAreaLocation) Run(ctx context.Context, cmd AssessAreaLocationCommand) (AssessAreaLocationResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationAssessAreaLocation, spanAssessAreaLocation},
		func() (AssessAreaLocationResult, error) {
			switch v := cmd.Request.(type) {
			case AssessMembershipRequest:
				return AreaMembership{}.Assess(v.Facts).(AssessAreaLocationResult), nil
			case ResolveOriginRequest:
				res := AreaOrigin{}.Resolve(v.Placement)
				if res == 0 {
					return nil, invalidInput("an origin needs a defined placement")
				}
				return res, nil
			}
			return nil, invalidInput("an area location needs a request")
		},
		func(r AssessAreaLocationResult) description {
			switch r.(type) {
			case AreaMembershipIncluded, AreaOriginResolution:
				return appliedWith(valueResultCompleted)
			case AreaMembershipExcluded:
				return appliedWith(valueResultAreaLocationExcluded)
			case AreaMembershipNoLinesConsidered:
				return invalidWith(valueResultAreaNoLinesConsidered)
			default:
				panic(unreachableError("area location result", r))
			}
		})
}
