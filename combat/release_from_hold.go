package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// ReleaseFromHoldInput holds the values NewReleaseFromHoldCommand validates. Build it with a
// keyed literal. The engine models no map: the host reports the helper's reach, the actor's
// ability to act and the Thieves' Tools (SRD 5.2.1, pp. 96-99).
type ReleaseFromHoldInput struct {
	// Holds are the holds in force and HoldID the hold to get out of.
	Holds  GearHolds
	HoldID HoldID
	// Method is the way out tried and Actor who tries.
	Method HoldReleaseMethod
	Actor  ReleaseActor
	// Actions says whether the actor can take actions, ConditionActionAvailability of its
	// effective conditions.
	Actions ConditionActionAvailability
	// HasThievesTools says whether the actor has Thieves' Tools, read for picking a lock only.
	HasThievesTools bool
	// Attempt is how the actor makes the ability check, with its modifiers and circumstances.
	// It is ignored for a key, which takes no check, but never nil.
	Attempt d20.CheckAttempt
}

// ReleaseFromHoldCommand is the immutable input of ReleaseFromHold.Run. Build it with
// NewReleaseFromHoldCommand; the zero value is invalid.
type ReleaseFromHoldCommand struct{ in ReleaseFromHoldInput }

// NewReleaseFromHoldCommand returns the command. The attempt's circumstances are copied.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a blank
// hold id, or GearProblemInvalidValue for a method, actor or actions that is not valid or an
// attempt that is not a d20.RollAttempt or d20.AutomaticFailure. It never panics. It returns GearProblemLimitExceeded for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewReleaseFromHoldCommand(in ReleaseFromHoldInput) (ReleaseFromHoldCommand, error) {
	attempt, ok := checkCheck(in.Attempt)
	_, held := in.Actor.(ActorIsHeld)
	_, helper := in.Actor.(ActorIsHelper)
	actions := in.Actions == ConditionActionAvailabilityAvailable || in.Actions == ConditionActionAvailabilityNone
	switch {
	case in.HoldID == "":
		return ReleaseFromHoldCommand{}, gearInvalid(GearProblemBlankID)
	case !in.Method.Valid() || (!held && !helper) || !actions || !ok:
		return ReleaseFromHoldCommand{}, gearInvalid(GearProblemInvalidValue)
	case attemptOverCap(in.Attempt) != nil:
		return ReleaseFromHoldCommand{}, gearInvalid(GearProblemLimitExceeded)
	}
	in.Attempt = attempt
	return ReleaseFromHoldCommand{in}, nil
}

// ReleaseFromHoldResult is the closed set of outcomes of ReleaseFromHold.Run. Variants:
// ReleaseFromHoldReleased, ReleaseFromHoldHeld, ReleaseFromHoldRejected.
type ReleaseFromHoldResult interface{ isReleaseFromHoldResult() }

// ReleaseFromHoldReleased: the hold ended.
type ReleaseFromHoldReleased struct {
	holds         GearHolds
	test          GearTest
	actionSpent   bool
	events        []GearEvent
	restraintEnds bool
}

// Holds returns the new set, without the hold.
func (r ReleaseFromHoldReleased) Holds() GearHolds { return r.holds }

// Test returns the check as d20 resolved it; ok is false for a key, which takes none.
func (r ReleaseFromHoldReleased) Test() (GearTest, bool) { return r.test, r.test != nil }

// ActionSpent reports whether the attempt took an action: true for a rolled check, false for a
// key. How the host spends the action is not decided here.
func (r ReleaseFromHoldReleased) ActionSpent() bool { return r.actionSpent }

// Events returns a copy of the events: one GearHoldEnded.
func (r ReleaseFromHoldReleased) Events() []GearEvent { return slices.Clone(r.events) }

// RestraintEnds reports whether the host removes the Restrained condition now.
func (r ReleaseFromHoldReleased) RestraintEnds() bool { return r.restraintEnds }

// ReleaseFromHoldHeld: the check failed and the hold stays. The action is spent.
type ReleaseFromHoldHeld struct {
	test   GearTest
	damage Damage
}

// Test returns the check as d20 resolved it.
func (r ReleaseFromHoldHeld) Test() GearTest { return r.test }

// ActionSpent returns true: a failed attempt takes its action.
func (ReleaseFromHoldHeld) ActionSpent() bool { return true }

// Damage returns the damage the failed attempt deals to the held creature, for the host to pass
// to ApplyDamage: 1 Piercing for a Hunting Trap; ok is false for any other hold.
func (r ReleaseFromHoldHeld) Damage() (Damage, bool) { return r.damage, r.damage != Damage{} }

// ReleaseFromHoldRejected: the attempt cannot be made and nothing was rolled or spent.
type ReleaseFromHoldRejected struct{ reason GearRejection }

// Reason returns why: unknownHold, methodNotAvailable, helperNotAllowed, helperOutOfReach,
// noAction or toolsMissing.
func (r ReleaseFromHoldRejected) Reason() GearRejection { return r.reason }

func (ReleaseFromHoldReleased) isReleaseFromHoldResult() {}
func (ReleaseFromHoldHeld) isReleaseFromHoldResult()     {}
func (ReleaseFromHoldRejected) isReleaseFromHoldResult() {}

// ReleaseFromHold is the use case of getting out of a hold: escaping, bursting, picking a lock
// or using a key (SRD 5.2.1, pp. 96-99). It composes the pure d20 functions and never calls
// d20.Resolver or another use case. Create it with NewReleaseFromHold; the zero value is not
// usable.
type ReleaseFromHold struct {
	gearUse
	flow attackFlow
}

// NewReleaseFromHold returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewReleaseFromHold(roller Roller, opts ...Option) (ReleaseFromHold, error) {
	if roller == nil {
		return ReleaseFromHold{}, invalidInput("release without a dice roller")
	}
	return ReleaseFromHold{newGearUse(opts), attackFlow{roller}}, nil
}

// Run plans the attempt with HoldRules and, when it is allowed, makes the check: only its d20s.
// Success ends the hold; failure leaves it and, for a Hunting Trap, returns 1 Piercing damage.
// A key ends a Manacles hold with no check and no action spent. The caller applies the damage
// and the condition changes.
//
// It records the shared signals, srd5e.combat.gear.holds_in_force and
// srd5e.combat.gear.hold_ended for a released hold. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case or command, and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression; the failure is recorded once and never logged.
func (u ReleaseFromHold) Run(ctx context.Context, cmd ReleaseFromHoldCommand) (ReleaseFromHoldResult, error) {
	if cmd.in.HoldID == "" || u.flow.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in := cmd.in
	op := combatOperation{valueOperationReleaseFromHold, spanReleaseFromHold}
	res, err := run(ctx, u.in, op, func(context.Context) (ReleaseFromHoldResult, error) {
		hold, ok := in.Holds.HoldOf(in.HoldID)
		if !ok {
			return ReleaseFromHoldRejected{GearRejectionUnknownHold}, nil
		}
		plan, refusal := ReleaseRequest{hold, in.Method, in.Actor, in.Actions, in.HasThievesTools}.plan()
		if !refusal.IsZero() {
			return ReleaseFromHoldRejected{refusal}, nil
		}
		var test GearTest
		if check, rolled := plan.Check(); rolled {
			var err error
			if test, err = u.flow.gearTest(d20.AbilityCheck, in.Attempt, check.dc); err != nil {
				return nil, err
			}
			if test.Outcome() != d20.Success {
				damage, _ := HoldRules{}.FailureDamage(hold.kind)
				return ReleaseFromHoldHeld{test, damage}, nil
			}
		}
		ended := in.Holds.end(in.HoldID, plan.cause).(GearHoldsApplied)
		return ReleaseFromHoldReleased{ended.holds, test, test != nil, ended.events, ended.restraintEnds}, nil
	}, func(r ReleaseFromHoldResult) description {
		switch v := r.(type) {
		case ReleaseFromHoldReleased:
			return description{Class: resultApplied, Name: valueResultReleased, Events: widen(v.events)}
		case ReleaseFromHoldHeld:
			return description{Class: resultApplied, Name: valueResultHeld}
		case ReleaseFromHoldRejected:
			return description{Class: resultRefused, Name: v.reason.String()}
		default:
			panic(fmt.Errorf("combat: unreachable release from hold result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if released, ok := res.(ReleaseFromHoldReleased); ok {
		u.recordHolds(ctx, op, released.holds, released.events)
	}
	return res, nil
}
