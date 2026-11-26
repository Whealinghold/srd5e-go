package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
)

// ExitContainerMethod is how a held creature tries to get out: by itself or pulled by another creature.
// Variants: ExitBySelf, ExitByPull.
type ExitContainerMethod interface{ isExitContainerMethod() }

// ExitBySelf: the held creature takes an action to make a Strength (Athletics) check (SRD 5.2.1,
// "Gelatinous Cube", p. 287). Construct it with a keyed literal.
type ExitBySelf struct {
	// Attempt is the check: a d20.RollAttempt carrying the host's modifiers, or d20.AutomaticFailure.
	Attempt d20.CheckAttempt
	// Actions says whether the held creature can take actions.
	Actions ConditionActionAvailability
}

// ExitByPull: another creature within 5 feet takes an action to pull the held one out (SRD 5.2.1,
// "Gelatinous Cube", p. 287; "Water Elemental", p. 337). Construct it with a keyed literal.
type ExitByPull struct {
	// Puller is the creature that pulls.
	Puller CombatantID
	// PullerWithin5Feet says whether the puller is within 5 feet of the container.
	PullerWithin5Feet bool
	// Attempt is the puller's Strength check, as for ExitBySelf.
	Attempt d20.CheckAttempt
	// Actions says whether the puller can take actions.
	Actions ConditionActionAvailability
}

func (ExitBySelf) isExitContainerMethod() {}
func (ExitByPull) isExitContainerMethod() {}

// ExitContainerInput is the raw values of an ExitContainerCommand. Build it with a keyed literal.
type ExitContainerInput struct {
	// Containments is the creatures held.
	Containments Containments
	// Grapples is the grapples in force.
	Grapples Grapples
	// Occupant is the held creature that tries to get out, or is pulled out.
	Occupant CombatantID
	// OccupantConditions are the occupant's conditions.
	OccupantConditions ConditionSet
	// Method is ExitBySelf or ExitByPull.
	Method ExitContainerMethod
}

// ExitContainerCommand is the input of ExitContainer.Run. Build it with NewExitContainerCommand; the zero
// value is invalid.
type ExitContainerCommand struct{ in ExitContainerInput }

// NewExitContainerCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a
// blank id, a method that is not ExitBySelf or ExitByPull, a pull without a puller, an attempt that is
// neither a d20.RollAttempt nor a d20.AutomaticFailure, or an invalid action availability. It copies the
// attempt and never panics. It returns a LimitExceededError for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewExitContainerCommand(in ExitContainerInput) (ExitContainerCommand, error) {
	bad := invalidInput("exit needs an occupant, a method, a check and a valid action availability")
	if in.Occupant.IsZero() {
		return ExitContainerCommand{}, bad
	}
	valid := func(a d20.CheckAttempt, actions ConditionActionAvailability) (d20.CheckAttempt, bool) {
		copied, ok := checkCheck(a)
		return copied, ok && actions >= ConditionActionAvailabilityAvailable && actions <= ConditionActionAvailabilityNone
	}
	switch m := in.Method.(type) {
	case ExitBySelf:
		if err := attemptOverCap(m.Attempt); err != nil {
			return ExitContainerCommand{}, err
		}
		a, ok := valid(m.Attempt, m.Actions)
		if !ok {
			return ExitContainerCommand{}, bad
		}
		m.Attempt = a
		in.Method = m
	case ExitByPull:
		if err := attemptOverCap(m.Attempt); err != nil {
			return ExitContainerCommand{}, err
		}
		a, ok := valid(m.Attempt, m.Actions)
		if !ok || m.Puller.IsZero() {
			return ExitContainerCommand{}, bad
		}
		m.Attempt = a
		in.Method = m
	default:
		return ExitContainerCommand{}, bad
	}
	return ExitContainerCommand{in}, nil
}

// ExitContainerResult is the closed set of outcomes of ExitContainer.Run. Variants: ExitContainerExited,
// ExitContainerHeld, ExitContainerRejected. ActionSpent says whether the acting creature's action is spent:
// true when a check was made, false for a refusal.
type ExitContainerResult interface {
	isExitContainerResult()
	ActionSpent() bool
}

// ExitContainerExited: the check succeeded and the creature is out.
type ExitContainerExited struct {
	containments Containments
	grapples     Grapples
	check        ContainmentTest
	release      ContainmentRelease
	damage       []ContainmentDamageDue
}

// Containments returns the set without the creature.
func (r ExitContainerExited) Containments() Containments { return r.containments }

// Grapples returns the grapples after the exit; an Engulfed-grappled creature's grapple has ended.
func (r ExitContainerExited) Grapples() Grapples { return r.grapples }

// Check returns the check.
func (r ExitContainerExited) Check() ContainmentTest { return r.check }

// Release returns the effect on the creature: its conditions, placement and breathing.
func (r ExitContainerExited) Release() ContainmentRelease { return r.release }

// DamageDue returns a new slice of the damage the puller takes; empty for an exit by itself.
func (r ExitContainerExited) DamageDue() []ContainmentDamageDue { return slices.Clone(r.damage) }

// Events returns a copy of the events, in the order they happened: the end of a grapple, then the
// release.
func (r ExitContainerExited) Events() []Event { return r.release.Events() }

// ActionSpent is true.
func (ExitContainerExited) ActionSpent() bool { return true }

// ExitContainerHeld: the check failed (a normal outcome); nothing changed and no damage is due.
type ExitContainerHeld struct {
	containments Containments
	check        ContainmentTest
}

// Containments returns the unchanged set.
func (r ExitContainerHeld) Containments() Containments { return r.containments }

// Check returns the check.
func (r ExitContainerHeld) Check() ContainmentTest { return r.check }

// ActionSpent is true.
func (ExitContainerHeld) ActionSpent() bool { return true }

// ExitContainerRejected: the rules refuse the attempt; no die was rolled. Construct it with a keyed
// literal.
type ExitContainerRejected struct{ Rejection ContainmentRejection }

// ActionSpent is false.
func (ExitContainerRejected) ActionSpent() bool { return false }

func (ExitContainerExited) isExitContainerResult()   {}
func (ExitContainerHeld) isExitContainerResult()     {}
func (ExitContainerRejected) isExitContainerResult() {}

// ExitContainer is the use case of getting out of a container that lives (SRD 5.2.1, "Gelatinous Cube",
// p. 287). It composes the pure d20 functions with the dice port and never calls d20.Resolver or another
// use case. Create it with NewExitContainer; the zero value is not usable.
type ExitContainer struct {
	containmentUse
	flow attackFlow
}

// NewExitContainer returns the use case rolling through roller. Options: WithTelemetry, WithClock. It
// returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewExitContainer(roller Roller, opts ...Option) (ExitContainer, error) {
	if roller == nil {
		return ExitContainer{}, invalidInput("exit without a dice roller")
	}
	return ExitContainer{newContainmentUse(opts), attackFlow{roller}}, nil
}

// Run refuses, with no die and in this order: a creature that is not held, a method with no DC in the
// container's terms (a swallow has none, and an Engulfed-grappled creature leaves by its grapple's escape),
// a puller held by the same container, a puller beyond 5 feet, and an acting creature that cannot act.
// Otherwise it makes the Strength check against the method's DC and releases the creature on a success
// (ContainmentLeaveCauseEscaped or ContainmentLeaveCausePulledOut, ending the grapple of an
// Engulfed-grappled creature as escaped); a pull also reports the puller's damage due.
//
// It records the shared signals, srd5e.combat.containment.ended for an exit and
// srd5e.combat.containment.occupants for a result with a set. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case or command, and ctx.Err() unwrapped,
// before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error")
// with a nil result when the roller fails, panics or answers another expression; the failure is recorded
// once and never logged.
func (u ExitContainer) Run(ctx context.Context, cmd ExitContainerCommand) (ExitContainerResult, error) {
	if u.flow.roller == nil || cmd.in.Method == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationExitContainer, spanExitContainer}
	res, err := runContainment(ctx, u.containmentUse, op, func(context.Context) (ExitContainerResult, error) { return u.exit(cmd.in) },
		func(r ExitContainerResult) description {
			switch v := r.(type) {
			case ExitContainerExited:
				return description{Class: resultApplied, Name: valueResultExited, Events: v.release.events}
			case ExitContainerHeld:
				return description{Class: resultApplied, Name: valueResultHeld}
			case ExitContainerRejected:
				return description{Class: resultRefused, Name: v.Rejection.String()}
			default:
				panic(unreachableError("exit container result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case ExitContainerExited:
		u.recordEnded(ctx, v.release.cause)
		u.recordSet(ctx, op, v.containments)
	case ExitContainerHeld:
		u.recordSet(ctx, op, v.containments)
	}
	return res, nil
}

// exit is the work of Run.
func (u ExitContainer) exit(in ExitContainerInput) (ExitContainerResult, error) {
	held, ok := in.Containments.ContainerOf(in.Occupant).(InContainer)
	if !ok {
		return ExitContainerRejected{ContainmentRejectionNotContained}, nil
	}
	rec := held.record
	exit, _ := rec.terms.Exit()
	var (
		dc      d20.DifficultyClass
		attempt d20.CheckAttempt
		actions ConditionActionAvailability
		cause   ContainmentLeaveCause
		puller  CombatantID
	)
	switch m := in.Method.(type) {
	case ExitBySelf:
		dc, attempt, actions, cause = exit.selfEscapeDC, m.Attempt, m.Actions, ContainmentLeaveCauseEscaped
		if rec.terms.mode == ContainmentModeEngulfedGrappled {
			dc = 0
		}
		if dc == 0 {
			return ExitContainerRejected{ContainmentRejectionNoWayOut}, nil
		}
	case ExitByPull:
		dc, attempt, actions, cause, puller = exit.pullDC, m.Attempt, m.Actions, ContainmentLeaveCausePulledOut, m.Puller
		if dc == 0 {
			return ExitContainerRejected{ContainmentRejectionNoWayOut}, nil
		}
		if peer, held := in.Containments.ContainerOf(puller).(InContainer); held && peer.record.Container == rec.Container {
			return ExitContainerRejected{ContainmentRejectionPullerIsContained}, nil
		}
		if !m.PullerWithin5Feet {
			return ExitContainerRejected{ContainmentRejectionPullerOutOfReach}, nil
		}
	default:
		panic(unreachableError("exit method", in.Method))
	}
	if actions == ConditionActionAvailabilityNone {
		return ExitContainerRejected{ContainmentRejectionNoAction}, nil
	}
	test, err := u.flow.containmentTest(d20.AbilityCheck, attempt, dc)
	if err != nil {
		return nil, err
	}
	if !test.passed() {
		return ExitContainerHeld{in.Containments, test}, nil
	}
	grapples, rel := releaseHeld(in.Grapples, in.OccupantConditions, rec, cause)
	var damage []ContainmentDamageDue
	if cause == ContainmentLeaveCausePulledOut {
		damage = dueOf(rec.Container, puller, exit.pullerDamage, false, ContainmentDamageCausePull)
	}
	return ExitContainerExited{in.Containments.without(rec.Occupant), grapples, test, rel, damage}, nil
}
