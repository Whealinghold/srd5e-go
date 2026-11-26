package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// BindCreatureInput holds the values NewBindCreatureCommand validates. Build it with a keyed
// literal. The engine models no map: the host reports the distance, the size, the willingness
// and the conditions (SRD 5.2.1, "Chain", "Manacles", "Rope", pp. 96, 98, 99).
type BindCreatureInput struct {
	// Holds are the holds in force.
	Holds GearHolds
	// HoldID is the id of the new hold; never blank.
	HoldID HoldID
	// Kind is how the creature is bound.
	Kind BindingKind
	// Binder and Target are the two creatures. The binder is validated and named nowhere else:
	// the events name the held creature.
	Binder, Target CombatantID
	// TargetSize is the target's size.
	TargetSize content.CreatureSize
	// TargetConditions is the target's effective conditions.
	TargetConditions ConditionSet
	// Willingness says whether the target is willing; a willing target is refused.
	Willingness Willingness
	// DistanceFeet is the distance to the target, 0 or more; ignored for a Rope.
	DistanceFeet int
	// Restraint says whether the new hold restrains: for a Chain or Rope the legs are bound, for
	// Manacles they are attached to a fixed chain or hook.
	Restraint HoldRestraint
	// Attempt is how the binder makes the ability check, with its modifiers and circumstances.
	Attempt d20.CheckAttempt
}

// BindCreatureCommand is the immutable input of BindCreature.Run. Build it with
// NewBindCreatureCommand; the zero value is invalid.
type BindCreatureCommand struct{ in BindCreatureInput }

// NewBindCreatureCommand returns the command. The attempt's circumstances are copied.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a blank
// hold id or a zero binder or target, GearProblemNegativeDistance for a negative distance, or
// GearProblemInvalidValue for a kind, size, willingness or restraint that is not valid or an
// attempt that is not a d20.RollAttempt or d20.AutomaticFailure. It never panics. It returns GearProblemLimitExceeded for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewBindCreatureCommand(in BindCreatureInput) (BindCreatureCommand, error) {
	attempt, ok := checkCheck(in.Attempt)
	switch {
	case in.HoldID == "" || in.Binder.IsZero() || in.Target.IsZero():
		return BindCreatureCommand{}, gearInvalid(GearProblemBlankID)
	case in.DistanceFeet < 0:
		return BindCreatureCommand{}, gearInvalid(GearProblemNegativeDistance)
	case !in.Kind.Valid() || !in.TargetSize.Valid() || !in.Willingness.Valid() || !in.Restraint.Valid() || !ok:
		return BindCreatureCommand{}, gearInvalid(GearProblemInvalidValue)
	case in.Holds.Len() >= maxInputList:
		return BindCreatureCommand{}, gearInvalid(GearProblemLimitExceeded)
	case attemptOverCap(in.Attempt) != nil:
		return BindCreatureCommand{}, gearInvalid(GearProblemLimitExceeded)
	}
	in.Attempt = attempt
	return BindCreatureCommand{in}, nil
}

// BindCreatureResult is the closed set of outcomes of BindCreature.Run. Variants:
// BindCreatureBound, BindCreatureFailed, BindCreatureRejected.
type BindCreatureResult interface{ isBindCreatureResult() }

// BindCreatureBound: the binder's check succeeded and the creature is held.
type BindCreatureBound struct {
	holds           GearHolds
	test            GearTest
	events          []GearEvent
	restraintBegins bool
}

// Holds returns the new set, with the new hold.
func (r BindCreatureBound) Holds() GearHolds { return r.holds }

// Test returns the check as d20 resolved it.
func (r BindCreatureBound) Test() GearTest { return r.test }

// Events returns a copy of the events: one GearHoldStarted.
func (r BindCreatureBound) Events() []GearEvent { return slices.Clone(r.events) }

// RestraintBegins reports whether the host applies the Restrained condition now.
func (r BindCreatureBound) RestraintBegins() bool { return r.restraintBegins }

// BindCreatureFailed: the binder's check failed and nothing holds the creature.
type BindCreatureFailed struct{ test GearTest }

// Test returns the check as d20 resolved it.
func (r BindCreatureFailed) Test() GearTest { return r.test }

// BindCreatureRejected: the creature cannot be bound and nothing was rolled.
type BindCreatureRejected struct{ reason GearRejection }

// Reason returns why: the first of targetWilling, outOfReach, sizeNotAllowed, targetNotHeldDown,
// targetAlreadyHeld, idInUse.
func (r BindCreatureRejected) Reason() GearRejection { return r.reason }

func (BindCreatureBound) isBindCreatureResult()    {}
func (BindCreatureFailed) isBindCreatureResult()   {}
func (BindCreatureRejected) isBindCreatureResult() {}

// BindCreature is the use case of binding an unwilling creature that is Grappled, Incapacitated
// or Restrained with a Chain, Manacles or Rope (SRD 5.2.1, pp. 96, 98, 99). It composes the
// pure d20 functions and never calls d20.Resolver or another use case. Create it with
// NewBindCreature; the zero value is not usable.
type BindCreature struct {
	gearUse
	flow attackFlow
}

// NewBindCreature returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewBindCreature(roller Roller, opts ...Option) (BindCreature, error) {
	if roller == nil {
		return BindCreature{}, invalidInput("binding without a dice roller")
	}
	return BindCreature{newGearUse(opts), attackFlow{roller}}, nil
}

// Run assesses the binding with BindingRules and, when it is allowed, makes the binder's check:
// only its d20s. Success begins a hold with the profile's escape Difficulty Class and the
// restraint asked for; failure holds nothing.
//
// It records the shared signals and srd5e.combat.gear.holds_in_force for a bound creature. It
// returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or
// command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails,
// panics or answers another expression; the failure is recorded once and never logged.
func (u BindCreature) Run(ctx context.Context, cmd BindCreatureCommand) (BindCreatureResult, error) {
	if cmd.in.HoldID == "" || u.flow.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in := cmd.in
	op := combatOperation{valueOperationBindCreature, spanBindCreature}
	res, err := run(ctx, u.in, op, func(context.Context) (BindCreatureResult, error) {
		check, refusal := BindingScene{in.Kind, in.Willingness, in.DistanceFeet, in.Target, in.TargetSize, in.TargetConditions, in.Holds, in.HoldID}.assess()
		if !refusal.IsZero() {
			return BindCreatureRejected{refusal}, nil
		}
		test, err := u.flow.gearTest(d20.AbilityCheck, in.Attempt, check.dc)
		if err != nil {
			return nil, err
		}
		if test.Outcome() != d20.Success {
			return BindCreatureFailed{test}, nil
		}
		kind := in.Kind.HoldKind()
		hold := GearHold{in.HoldID, kind, in.Target, HoldProfileFor(kind).escape.dc, in.Restraint, "", 0, ObjectStatus{}}
		applied := in.Holds.begin(hold).(GearHoldsApplied)
		return BindCreatureBound{applied.holds, test, applied.events, applied.restraintBegins}, nil
	}, func(r BindCreatureResult) description {
		switch v := r.(type) {
		case BindCreatureBound:
			return description{Class: resultApplied, Name: valueResultBound, Events: widen(v.events)}
		case BindCreatureFailed:
			return description{Class: resultApplied, Name: valueResultFailed}
		case BindCreatureRejected:
			return description{Class: resultRefused, Name: v.reason.String()}
		default:
			panic(fmt.Errorf("combat: unreachable bind creature result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if bound, ok := res.(BindCreatureBound); ok {
		u.recordHolds(ctx, op, bound.holds, bound.events)
	}
	return res, nil
}
