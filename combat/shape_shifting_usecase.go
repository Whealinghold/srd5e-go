package combat

import "context"

// BeginShapeShiftingCommand is the input of BeginShapeShifting.Run: the command of ShapeShiftSpells.Begin.
type BeginShapeShiftingCommand BeginShapeShiftCommand

// BeginShapeShifting is the use case of casting Polymorph, True Polymorph, Shapechange or Animal Shapes: it wraps
// ShapeShiftSpells.Begin and returns its BeginShapeShiftingResult (the spec's BeginShapeShift, renamed because that is
// the owner's command). The result is completed (applied), shape_shift_resisted (applied) or the name of the rejection
// (target_is_dead, target_not_willing, shapechange_not_self or form_ineligible: refused). Create it with
// NewBeginShapeShifting.
type BeginShapeShifting struct{ in instrumenter }

// NewBeginShapeShifting returns the use case. Options: WithTelemetry, WithClock.
func NewBeginShapeShifting(opts ...Option) BeginShapeShifting {
	return BeginShapeShifting{newWired(opts)}
}

// Run begins the transformation. It returns the errors of ShapeShiftSpells.Begin (srderr.ErrInvalidInput for an
// invalid command) and those described for the package.
func (u BeginShapeShifting) Run(ctx context.Context, cmd BeginShapeShiftingCommand) (BeginShapeShiftingResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationBeginShapeShift, spanBeginShapeShift},
		func() (BeginShapeShiftingResult, error) { return ShapeShiftSpells{}.Begin(BeginShapeShiftCommand(cmd)) },
		func(r BeginShapeShiftingResult) description {
			switch v := r.(type) {
			case BeginShapeShiftApplied:
				return appliedWith(valueResultCompleted, v.Events()...)
			case BeginShapeShiftResisted:
				return appliedWith(valueResultShapeShiftResisted)
			case BeginShapeShiftRefused:
				return refusedWith(v.Rejection.String())
			default:
				panic(unreachableError("begin shape shift result", r))
			}
		})
}

// ChangeShapeShiftFormCommand is the input of ChangeShapeShiftForm.Run: the command of ShapeShiftSpells.ChangeForm.
type ChangeShapeShiftFormCommand ChangeFormCommand

// ChangeShapeShiftForm is the use case of assuming another form with the Magic action: it wraps
// ShapeShiftSpells.ChangeForm and returns its ChangeShapeShiftFormResult. The result is completed (applied) or the name of the
// rejection (target_is_dead, form_change_not_allowed, not_transformed, actor_not_caster or form_ineligible: refused).
// Create it with NewChangeShapeShiftForm.
type ChangeShapeShiftForm struct{ in instrumenter }

// NewChangeShapeShiftForm returns the use case. Options: WithTelemetry, WithClock.
func NewChangeShapeShiftForm(opts ...Option) ChangeShapeShiftForm {
	return ChangeShapeShiftForm{newWired(opts)}
}

// Run changes the form. It returns the errors of ShapeShiftSpells.ChangeForm and those described for the package.
func (u ChangeShapeShiftForm) Run(ctx context.Context, cmd ChangeShapeShiftFormCommand) (ChangeShapeShiftFormResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationChangeShapeShiftForm, spanChangeShapeShiftForm},
		func() (ChangeShapeShiftFormResult, error) {
			return ShapeShiftSpells{}.ChangeForm(ChangeFormCommand(cmd))
		},
		func(r ChangeShapeShiftFormResult) description {
			switch v := r.(type) {
			case ChangeFormApplied:
				return appliedWith(valueResultCompleted, v.Events()...)
			case ChangeFormRefused:
				return refusedWith(v.Rejection.String())
			default:
				panic(unreachableError("change form result", r))
			}
		})
}

// AssessShapeShiftCommand is the input of AssessShapeShift.Run: the arguments of ShapeShiftSpells.Assess. The host
// makes the call with the new HitPointStatus after any call that changed Hit Points; no damage use case does (FR-007).
type AssessShapeShiftCommand struct {
	State  ActiveTransformation
	Status HitPointStatus
}

// AssessShapeShift is the use case of asking whether a transformation ends because the form's Temporary Hit
// Points are gone or the creature dropped to 0 Hit Points: it wraps ShapeShiftSpells.Assess and returns its
// AssessShapeShiftResult (the spec's AssessShapeShift, renamed because that is the owner's result). The result is
// completed (AssessShapeShiftContinues) or shape_shift_ended (AssessShapeShiftEnded), both applied. Create it with
// NewAssessShapeShift.
type AssessShapeShift struct{ in instrumenter }

// NewAssessShapeShift returns the use case. Options: WithTelemetry, WithClock.
func NewAssessShapeShift(opts ...Option) AssessShapeShift {
	return AssessShapeShift{newWired(opts)}
}

// Run assesses the transformation. It returns the errors of ShapeShiftSpells.Assess and those described for the package.
func (u AssessShapeShift) Run(ctx context.Context, cmd AssessShapeShiftCommand) (AssessShapeShiftResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationAssessShapeShift, spanAssessShapeShift},
		func() (AssessShapeShiftResult, error) { return ShapeShiftSpells{}.Assess(cmd.State, cmd.Status) },
		func(r AssessShapeShiftResult) description {
			switch r.(type) {
			case AssessShapeShiftContinues:
				return appliedWith(valueResultCompleted)
			case AssessShapeShiftEnded:
				return appliedWith(valueResultShapeShiftEnded)
			default:
				panic(unreachableError("assess shape shift result", r))
			}
		})
}

// EndShapeShiftingCommand is the input of EndShapeShifting.Run: the command of ShapeShiftSpells.End.
type EndShapeShiftingCommand EndShapeShiftCommand

// EndShapeShifting is the use case of ending a transformation: it wraps ShapeShiftSpells.End and returns its
// EndShapeShiftingResult (the spec's EndShapeShift, renamed because that is the owner's command). The result is completed
// (EndShapeShiftApplied) or shape_shift_unaffected (EndShapeShiftUnaffected), both applied, or the name of the rejection
// (cannot_end_at_will or target_is_dead: refused). Create it with NewEndShapeShifting.
type EndShapeShifting struct{ in instrumenter }

// NewEndShapeShifting returns the use case. Options: WithTelemetry, WithClock.
func NewEndShapeShifting(opts ...Option) EndShapeShifting { return EndShapeShifting{newWired(opts)} }

// Run ends the transformation. It returns the errors of ShapeShiftSpells.End and those described for the package.
func (u EndShapeShifting) Run(ctx context.Context, cmd EndShapeShiftingCommand) (EndShapeShiftingResult, error) {
	return wired(ctx, u.in, combatOperation{valueOperationEndShapeShift, spanEndShapeShift},
		func() (EndShapeShiftingResult, error) { return ShapeShiftSpells{}.End(EndShapeShiftCommand(cmd)) },
		func(r EndShapeShiftingResult) description {
			switch v := r.(type) {
			case EndShapeShiftApplied:
				return appliedWith(valueResultCompleted, v.Events()...)
			case EndShapeShiftUnaffected:
				return appliedWith(valueResultShapeShiftUnaffected)
			case EndShapeShiftRefused:
				return refusedWith(v.Rejection.String())
			default:
				panic(unreachableError("end shape shift result", r))
			}
		})
}
