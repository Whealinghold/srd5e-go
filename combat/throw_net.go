package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ThrowNetInput holds the values NewThrowNetCommand validates. Build it with a keyed literal.
// The engine models no map: the host reports the distance, the sight and the size (SRD 5.2.1,
// "Net", p. 98).
type ThrowNetInput struct {
	// Holds are the holds in force.
	Holds GearHolds
	// HoldID is the id of the hold a caught target gets; never blank.
	HoldID HoldID
	// Thrower and Target are the two creatures. The thrower is validated and named nowhere
	// else: the events name the held creature.
	Thrower, Target CombatantID
	// TargetSize is the target's size; Huge and Gargantuan succeed without a roll.
	TargetSize content.CreatureSize
	// DistanceFeet is the distance to the target, 0 or more.
	DistanceFeet int
	// Sight says whether the thrower can see the target.
	Sight ThrowerSight
	// SaveDC is the Difficulty Class of the target's Dexterity saving throw, from NetRules.SaveDC.
	SaveDC d20.DifficultyClass
	// Attempt is how the target makes the saving throw, with its modifiers and circumstances.
	// It is ignored for a target that succeeds automatically, but never nil.
	Attempt d20.SaveAttempt
	// EscapeDCOverride replaces the Net's escape Difficulty Class when it supplies one.
	EscapeDCOverride DCOverride
}

// ThrowNetCommand is the immutable input of ThrowNet.Run. Build it with NewThrowNetCommand;
// the zero value is invalid.
type ThrowNetCommand struct{ in ThrowNetInput }

// NewThrowNetCommand returns the command. The attempt's circumstances are copied.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a blank
// hold id or a zero thrower or target, GearProblemNegativeDistance for a negative distance, or
// GearProblemInvalidValue for a size or sight that is not valid or an attempt that is not a
// d20.RollAttempt, d20.ChooseToFail or d20.AutomaticFailure, and GearProblemLimitExceeded for a set of holds that already holds 256 (plan P10). It never panics. It returns GearProblemLimitExceeded for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewThrowNetCommand(in ThrowNetInput) (ThrowNetCommand, error) {
	attempt, ok := checkSave(in.Attempt)
	switch {
	case in.HoldID == "" || in.Thrower.IsZero() || in.Target.IsZero():
		return ThrowNetCommand{}, gearInvalid(GearProblemBlankID)
	case in.DistanceFeet < 0:
		return ThrowNetCommand{}, gearInvalid(GearProblemNegativeDistance)
	case !in.TargetSize.Valid() || !in.Sight.Valid() || !ok:
		return ThrowNetCommand{}, gearInvalid(GearProblemInvalidValue)
	case in.Holds.Len() >= maxInputList:
		return ThrowNetCommand{}, gearInvalid(GearProblemLimitExceeded)
	case attemptOverCap(in.Attempt) != nil:
		return ThrowNetCommand{}, gearInvalid(GearProblemLimitExceeded)
	}
	in.Attempt = attempt
	return ThrowNetCommand{in}, nil
}

// ThrowNetResult is the closed set of outcomes of ThrowNet.Run. Variants: ThrowNetHeld,
// ThrowNetSaved, ThrowNetAutoSucceeded, ThrowNetRejected.
type ThrowNetResult interface{ isThrowNetResult() }

// ThrowNetHeld: the target failed the saving throw and is held by the Net.
type ThrowNetHeld struct {
	holds           GearHolds
	test            GearTest
	events          []GearEvent
	restraintBegins bool
}

// Holds returns the new set, with the Net's hold.
func (r ThrowNetHeld) Holds() GearHolds { return r.holds }

// Test returns the saving throw as d20 resolved it.
func (r ThrowNetHeld) Test() GearTest { return r.test }

// Events returns a copy of the events: one GearHoldStarted.
func (r ThrowNetHeld) Events() []GearEvent { return slices.Clone(r.events) }

// RestraintBegins reports whether the host applies the Restrained condition now: a Net always
// restrains, but another hold may already have.
func (r ThrowNetHeld) RestraintBegins() bool { return r.restraintBegins }

// ThrowNetSaved: the target passed the saving throw and nothing holds it.
type ThrowNetSaved struct{ test GearTest }

// Test returns the saving throw as d20 resolved it.
func (r ThrowNetSaved) Test() GearTest { return r.test }

// ThrowNetAutoSucceeded: the target is Huge or Gargantuan and succeeds with no roll.
type ThrowNetAutoSucceeded struct{}

// ThrowNetRejected: the Net cannot be thrown and nothing was rolled.
type ThrowNetRejected struct{ reason GearRejection }

// Reason returns why: targetNotSeen, outOfReach, targetAlreadyHeld or idInUse.
func (r ThrowNetRejected) Reason() GearRejection { return r.reason }

func (ThrowNetHeld) isThrowNetResult()          {}
func (ThrowNetSaved) isThrowNetResult()         {}
func (ThrowNetAutoSucceeded) isThrowNetResult() {}
func (ThrowNetRejected) isThrowNetResult()      {}

// ThrowNet is the use case of throwing a Net at a creature (SRD 5.2.1, "Net", p. 98): the
// target makes a Dexterity saving throw, which a Huge or Gargantuan target passes without a
// roll. It composes the pure d20 functions and never calls d20.Resolver or another use case.
// Create it with NewThrowNet; the zero value is not usable.
type ThrowNet struct {
	gearUse
	flow attackFlow
}

// NewThrowNet returns the use case rolling through roller. Options: WithTelemetry, WithClock.
// It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewThrowNet(roller Roller, opts ...Option) (ThrowNet, error) {
	if roller == nil {
		return ThrowNet{}, invalidInput("net without a dice roller")
	}
	return ThrowNet{newGearUse(opts), attackFlow{roller}}, nil
}

// Run checks that the Net can be thrown (targetNotSeen, outOfReach, targetAlreadyHeld, idInUse,
// in that order), then lets a Huge or Gargantuan target succeed, or has the target make the
// saving throw: only its d20s. A failure begins a Net hold that restrains, with the profile's
// escape Difficulty Class or the override. The attack the throw replaces is the caller's to
// record.
//
// It records the shared signals and srd5e.combat.gear.holds_in_force for a held target. It
// returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or
// command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails,
// panics or answers another expression; the failure is recorded once and never logged.
func (u ThrowNet) Run(ctx context.Context, cmd ThrowNetCommand) (ThrowNetResult, error) {
	if cmd.in.HoldID == "" || u.flow.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in := cmd.in
	op := combatOperation{valueOperationThrowNet, spanThrowNet}
	res, err := run(ctx, u.in, op, func(context.Context) (ThrowNetResult, error) {
		if r := netPossibility(in.DistanceFeet, in.Sight); !r.IsZero() {
			return ThrowNetRejected{r}, nil
		}
		hold := GearHold{in.HoldID, HoldKindNet, in.Target, in.EscapeDCOverride.or(HoldProfileFor(HoldKindNet).escape.dc),
			HoldRestraintRestrains, "", 0, NetProfile{}.Status()}
		began := in.Holds.begin(hold)
		if rej, refused := began.(GearHoldsRejected); refused {
			return ThrowNetRejected(rej), nil
		}
		if (NetRules{}).AutoSucceeds(in.TargetSize) {
			return ThrowNetAutoSucceeded{}, nil
		}
		test, err := u.flow.gearTest(d20.SavingThrow, in.Attempt, in.SaveDC)
		if err != nil {
			return nil, err
		}
		if test.Outcome() == d20.Success {
			return ThrowNetSaved{test}, nil
		}
		applied := began.(GearHoldsApplied)
		return ThrowNetHeld{applied.holds, test, applied.events, applied.restraintBegins}, nil
	}, func(r ThrowNetResult) description {
		switch v := r.(type) {
		case ThrowNetHeld:
			return description{Class: resultApplied, Name: valueResultHeld, Events: widen(v.events)}
		case ThrowNetSaved:
			return description{Class: resultApplied, Name: valueResultSaved}
		case ThrowNetAutoSucceeded:
			return description{Class: resultApplied, Name: valueResultAutoSucceeded}
		case ThrowNetRejected:
			return description{Class: resultRefused, Name: v.reason.String()}
		default:
			panic(fmt.Errorf("combat: unreachable throw net result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if held, ok := res.(ThrowNetHeld); ok {
		u.recordHolds(ctx, op, held.holds, held.events)
	}
	return res, nil
}
