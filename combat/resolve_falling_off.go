package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ResolveFallingOffCommand is the immutable input of ResolveFallingOff.Run. Build it with
// NewResolveFallingOffCommand; the zero value is invalid. It declares no TurnTracker,
// resource, turn facts or condition field: the save does not depend on whose turn it is.
type ResolveFallingOffCommand struct {
	rider   CombatantID
	status  RiderStatus
	trigger FallingOffTrigger
	attempt d20.SaveAttempt
	landing LandingSpace
}

// NewResolveFallingOffCommand returns the command. The attempt is the rider's Dexterity
// saving throw, built by the caller with its modifiers and circumstances (Dodge's Advantage,
// a condition's Disadvantage, Paralyzed's automatic failure). The circumstances are copied.
//
// It returns an error matching srderr.ErrInvalidInput for a blank rider, a rider that is
// its own mount in status, a trigger or landing space that is not valid, or an attempt that
// is not a d20.RollAttempt, d20.ChooseToFail or d20.AutomaticFailure. A rider who is not
// mounted is valid here and refused by Run. It never panics. It returns a LimitExceededError for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewResolveFallingOffCommand(rider CombatantID, status RiderStatus, trigger FallingOffTrigger,
	attempt d20.SaveAttempt, landing LandingSpace) (ResolveFallingOffCommand, error) {
	if rider.IsZero() || status.IsOn(rider) || !trigger.Valid() || !landing.Valid() {
		return ResolveFallingOffCommand{}, invalidInput("falling off needs a rider, a mount other than the rider, a trigger and a landing space")
	}
	if err := attemptOverCap(attempt); err != nil {
		return ResolveFallingOffCommand{}, err
	}
	switch a := attempt.(type) {
	case d20.RollAttempt:
		a.Circumstances = slices.Clone(a.Circumstances)
		attempt = a
	case d20.ChooseToFail, d20.AutomaticFailure:
	default:
		return ResolveFallingOffCommand{}, invalidInput("falling off save attempt " + typeName(attempt))
	}
	return ResolveFallingOffCommand{rider, status, trigger, attempt, landing}, nil
}

// ResolveFallingOffResult is the closed set of outcomes of ResolveFallingOff.Run. Variants:
// ResolveFallingOffStayedMounted, ResolveFallingOffFellOff, ResolveFallingOffRejected.
type ResolveFallingOffResult interface{ isResolveFallingOffResult() }

// ResolveFallingOffStayedMounted: the save succeeded and the ride goes on. There is no event.
type ResolveFallingOffStayedMounted struct {
	status RiderStatus
	save   FallingOffSave
}

// RiderStatus returns the status the command held, equal by value.
func (r ResolveFallingOffStayedMounted) RiderStatus() RiderStatus { return r.status }

// Save returns the resolved save.
func (r ResolveFallingOffStayedMounted) Save() FallingOffSave { return r.save }

// ResolveFallingOffFellOff: the save failed. The rider is no longer mounted and lands Prone;
// the caller applies the condition FellOffMount names. A failed save is a normal result.
type ResolveFallingOffFellOff struct {
	save   FallingOffSave
	events []MountEvent
}

// RiderStatus returns NotMounted.
func (ResolveFallingOffFellOff) RiderStatus() RiderStatus { return NotMounted() }

// Save returns the resolved save.
func (r ResolveFallingOffFellOff) Save() FallingOffSave { return r.save }

// Events returns a copy of the events: one FellOffMount.
func (r ResolveFallingOffFellOff) Events() []MountEvent { return slices.Clone(r.events) }

// ResolveFallingOffRejected: the rider is not mounted, so no die was rolled.
type ResolveFallingOffRejected struct{ reason FallingOffRejection }

// Reason returns why.
func (r ResolveFallingOffRejected) Reason() FallingOffRejection { return r.reason }

func (ResolveFallingOffStayedMounted) isResolveFallingOffResult() {}
func (ResolveFallingOffFellOff) isResolveFallingOffResult()       {}
func (ResolveFallingOffRejected) isResolveFallingOffResult()      {}

// ResolveFallingOff is the use case of the Falling Off saving throw: a DC 10 Dexterity save
// when an effect moves the mount against its will or the rider or the mount is knocked Prone
// (SRD 5.2.1, "Falling Off", p. 16). It rolls through the dice port and composes the pure d20
// functions; it never calls d20.Resolver or another use case, so no srd5e.d20_test signal
// fires. Create it with NewResolveFallingOff; the zero value is not usable.
type ResolveFallingOff struct {
	mountingUse
	flow attackFlow
}

// NewResolveFallingOff returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveFallingOff(roller Roller, opts ...Option) (ResolveFallingOff, error) {
	if roller == nil {
		return ResolveFallingOff{}, invalidInput("falling off without a dice roller")
	}
	return ResolveFallingOff{newMountingUse(opts), attackFlow{roller}}, nil
}

// save makes the one saving throw against the Difficulty Class: a RollAttempt resolves its
// roll mode once, rolls that many d20s and compares with d20.ResolveCheck; the other
// attempts fail without a die.
func (u ResolveFallingOff) save(a d20.SaveAttempt) (FallingOffSave, error) {
	var reason d20.ForcedFailureReason
	switch v := a.(type) {
	case d20.RollAttempt:
		roll, err := u.flow.rollD20(modifier.Resolve(v.Circumstances...))
		if err != nil {
			return nil, err
		}
		res, err := d20.ResolveCheck(d20.SavingThrow, roll, v.Modifiers, FallingOffDifficultyClass)
		return FallingOffSaveRolled{res}, err
	case d20.ChooseToFail:
		reason = d20.ChosenToFail
	case d20.AutomaticFailure:
		reason = d20.Automatic
	default:
		return nil, invalidInput("falling off save attempt " + typeName(a))
	}
	res, err := d20.FailWithoutRoll(d20.SavingThrow, reason)
	return FallingOffSaveForced{res}, err
}

// Run makes the save for a mounted rider. A passed save keeps the rider mounted, with the
// status unchanged; a failed save, rolled or forced, ends the ride and reports one
// FellOffMount event. A rider who is not mounted is refused before any die is rolled.
//
// It records the shared signals with srd5e.combat.falling_off_trigger and, for a mounted
// rider, srd5e.combat.mount_control. It returns an error matching srderr.ErrInvalidInput,
// before any signal, for the zero use case or command, and ctx.Err() unwrapped, before any
// signal, for a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression; the failure is recorded once and never logged.
func (u ResolveFallingOff) Run(ctx context.Context, cmd ResolveFallingOffCommand) (ResolveFallingOffResult, error) {
	if cmd.rider.IsZero() || u.flow.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveFallingOff, spanResolveFallingOff}
	control, mounted := cmd.status.Control()
	res, err := run(ctx, u.in, op, func(context.Context) (ResolveFallingOffResult, error) {
		if !mounted {
			return ResolveFallingOffRejected{MountingRiderNotMountedRejection{}}, nil
		}
		save, err := u.save(cmd.attempt)
		if err != nil {
			return nil, err
		}
		if staysMounted(save) {
			return ResolveFallingOffStayedMounted{cmd.status, save}, nil
		}
		return ResolveFallingOffFellOff{save, []MountEvent{
			FellOffMount{mountEventBase{cmd.rider, cmd.status.mount}, control, cmd.trigger, cmd.landing}}}, nil
	}, func(r ResolveFallingOffResult) description {
		extras := append(controlExtras(control, mounted), telemetry.String(attrFallingOffTrigger, cmd.trigger.String()))
		switch v := r.(type) {
		case ResolveFallingOffStayedMounted:
			return description{Class: resultApplied, Name: valueResultStayedMounted, Extras: extras}
		case ResolveFallingOffFellOff:
			return description{Class: resultApplied, Name: valueResultFellOff, Events: widen(v.events), Extras: extras}
		case ResolveFallingOffRejected:
			return description{Class: resultRefused, Name: v.reason.String(), Extras: extras}
		default:
			panic(fmt.Errorf("combat: unreachable falling off result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
