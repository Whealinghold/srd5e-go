package combat

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// EscapeGrappleFromRecordCommand is the input of EscapeGrappleFromRecord.Run: the
// grapples in force, the Grappled creature that tries to escape, the check it chose,
// whether it can act, and how the check is attempted. Build it with
// NewEscapeGrappleFromRecordCommand; the zero value is invalid.
type EscapeGrappleFromRecordCommand struct {
	grapples Grapples
	target   CombatantID
	option   GrappleEscapeOption
	actions  ConditionActionAvailability
	attempt  d20.CheckAttempt
}

// NewEscapeGrappleFromRecordCommand returns the command. The attempt carries the
// modifiers the host computed for the chosen option (the ability modifier and the
// Proficiency Bonus of the skill) and the Advantage and Disadvantage sources; the
// engine adds none and does not check them against option. It returns an error
// matching srderr.ErrInvalidInput for an invalid option or actions, or an attempt
// that is neither a d20.RollAttempt nor a d20.AutomaticFailure value. A target that
// is not grappled is a refusal of Run, not an error.
func NewEscapeGrappleFromRecordCommand(grapples Grapples, target CombatantID, option GrappleEscapeOption,
	actions ConditionActionAvailability, attempt d20.CheckAttempt) (EscapeGrappleFromRecordCommand, error) {
	switch attempt.(type) {
	case d20.RollAttempt, d20.AutomaticFailure:
	default:
		return EscapeGrappleFromRecordCommand{}, invalidInput("escape attempt must be a roll or an automatic failure")
	}
	if !option.Valid() || actions < ConditionActionAvailabilityAvailable || actions > ConditionActionAvailabilityNone {
		return EscapeGrappleFromRecordCommand{}, invalidInput("escape needs a valid option and action availability")
	}
	return EscapeGrappleFromRecordCommand{grapples, target, option, actions, attempt}, nil
}

// Option returns the check the escaper chose.
func (c EscapeGrappleFromRecordCommand) Option() GrappleEscapeOption { return c.option }

// GrappleEscapeCheck is the check of an escape attempt, as d20 resolved it.
// Variants: GrappleEscapeRolled, GrappleEscapeForced.
type GrappleEscapeCheck interface{ isGrappleEscapeCheck() }

// GrappleEscapeRolled: the check was rolled.
type GrappleEscapeRolled struct{ result d20.CheckResult }

// Result returns the check d20.ResolveCheck resolved, unchanged.
func (c GrappleEscapeRolled) Result() d20.CheckResult { return c.result }

// GrappleEscapeForced: the check failed automatically, with no die.
type GrappleEscapeForced struct{ result d20.ForcedFailureResult }

// Result returns the forced failure d20.FailWithoutRoll returned.
func (c GrappleEscapeForced) Result() d20.ForcedFailureResult { return c.result }

func (GrappleEscapeRolled) isGrappleEscapeCheck() {}
func (GrappleEscapeForced) isGrappleEscapeCheck() {}

// EscapeGrappleFromRecordResult is the closed set of outcomes of
// EscapeGrappleFromRecord.Run. Variants: EscapeGrappleFromRecordEscaped,
// EscapeGrappleFromRecordHeld, EscapeGrappleFromRecordRejected. ActionSpent says
// whether the Grappled creature's action is spent: true when a check was made, false
// for a refusal. How the host spends it is the host's (the escape is an improvised
// action, SRD 5.2.1, "Grappling", p. 182).
type EscapeGrappleFromRecordResult interface {
	isEscapeGrappleFromRecordResult()
	ActionSpent() bool
}

// EscapeGrappleFromRecordEscaped: the check succeeded. The host passes the ending's
// target and cause to GrappleRules.EndOnSet.
type EscapeGrappleFromRecordEscaped struct {
	grapples Grapples
	check    GrappleEscapeCheck
	ending   GrappleEnding
}

// EscapeGrappleFromRecordHeld: the check failed (a normal outcome); nothing changed.
type EscapeGrappleFromRecordHeld struct {
	grapples Grapples
	check    GrappleEscapeCheck
}

// EscapeGrappleFromRecordRejected: no die was rolled. Construct it with a keyed
// literal.
type EscapeGrappleFromRecordRejected struct{ Rejection GrappleRejection }

func (EscapeGrappleFromRecordEscaped) isEscapeGrappleFromRecordResult()  {}
func (EscapeGrappleFromRecordHeld) isEscapeGrappleFromRecordResult()     {}
func (EscapeGrappleFromRecordRejected) isEscapeGrappleFromRecordResult() {}

// Grapples returns the set without the grapple that was escaped.
func (r EscapeGrappleFromRecordEscaped) Grapples() Grapples { return r.grapples }

// Check returns the check.
func (r EscapeGrappleFromRecordEscaped) Check() GrappleEscapeCheck { return r.check }

// Ending returns the ending, with GrappleEndCauseEscaped.
func (r EscapeGrappleFromRecordEscaped) Ending() GrappleEnding { return r.ending }

// ActionSpent is true.
func (EscapeGrappleFromRecordEscaped) ActionSpent() bool { return true }

// Grapples returns the unchanged set.
func (r EscapeGrappleFromRecordHeld) Grapples() Grapples { return r.grapples }

// Check returns the check.
func (r EscapeGrappleFromRecordHeld) Check() GrappleEscapeCheck { return r.check }

// ActionSpent is true.
func (EscapeGrappleFromRecordHeld) ActionSpent() bool { return true }

// ActionSpent is false.
func (EscapeGrappleFromRecordRejected) ActionSpent() bool { return false }

// EscapeGrappleFromRecord is the use case of the escape check of a Grappled creature
// (SRD 5.2.1, "Grappling", "Ending a Grapple", p. 182). It composes the pure d20
// functions with the dice port and never calls d20.Resolver, so no srd5e.d20_test
// signal fires. Create it with NewEscapeGrappleFromRecord; the zero value is not
// usable. It is safe for concurrent use only if its roller, telemetry and clock are.
type EscapeGrappleFromRecord struct {
	in     instrumenter
	tel    telemetry.Telemetry
	roller Roller
}

// NewEscapeGrappleFromRecord returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a
// nil roller.
func NewEscapeGrappleFromRecord(roller Roller, opts ...Option) (EscapeGrappleFromRecord, error) {
	if roller == nil {
		return EscapeGrappleFromRecord{}, invalidInput("escape without a dice roller")
	}
	o := newOptions(opts)
	return EscapeGrappleFromRecord{newInstrumenter(o.tel, o.clock), o.tel, roller}, nil
}

// Run refuses a target that is not grappled (checked first) or whose action is none,
// both without rolling. Otherwise it makes the check against the grapple's EscapeDC:
// an automatic failure draws no die; a roll resolves its mode once, requests 1d20
// (2d20 with Advantage or Disadvantage) and compares through d20.ResolveCheck. It
// records the shared signals plus, for an escape,
// srd5e.combat.grapple.ended (escaped), and srd5e.combat.grapple.in_force for a
// result that carries a set.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
// It returns a *srderr.PortError (error type "dice_roller_error") with a nil result when
// the roller fails, panics or answers another expression; the failure is recorded once
// and never logged.
func (u EscapeGrappleFromRecord) Run(ctx context.Context, cmd EscapeGrappleFromRecordCommand) (EscapeGrappleFromRecordResult, error) {
	if u.roller == nil || cmd.attempt == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationEscapeGrappleFromRecord, spanEscapeGrappleFromRecord}
	res, err := run(ctx, u.in, op, func(context.Context) (EscapeGrappleFromRecordResult, error) { return u.escape(cmd) },
		func(r EscapeGrappleFromRecordResult) description {
			switch v := r.(type) {
			case EscapeGrappleFromRecordEscaped:
				return description{Class: resultApplied, Name: valueResultEscaped}
			case EscapeGrappleFromRecordHeld:
				return description{Class: resultApplied, Name: valueResultHeld}
			case EscapeGrappleFromRecordRejected:
				return refusedGrapple(v.Rejection)
			default:
				panic(fmt.Errorf("combat: unreachable escape grapple result %T", r))
			}
		})
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case EscapeGrappleFromRecordEscaped:
		recordGrappleEnded(ctx, u.tel, v.ending.cause.String())
		recordGrappleInForce(ctx, u.tel, op, v.grapples)
	case EscapeGrappleFromRecordHeld:
		recordGrappleInForce(ctx, u.tel, op, v.grapples)
	}
	return res, nil
}

// escape is the work of Run.
func (u EscapeGrappleFromRecord) escape(cmd EscapeGrappleFromRecordCommand) (EscapeGrappleFromRecordResult, error) {
	in, ok := cmd.grapples.GrappleOf(cmd.target).(InGrapple)
	switch {
	case !ok:
		return EscapeGrappleFromRecordRejected{GrappleRejectionNotGrappled}, nil
	case cmd.actions == ConditionActionAvailabilityNone:
		return EscapeGrappleFromRecordRejected{GrappleRejectionNoAction}, nil
	}
	var check GrappleEscapeCheck
	switch a := cmd.attempt.(type) {
	case d20.AutomaticFailure:
		f, err := d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic)
		if err != nil {
			return nil, err
		}
		return EscapeGrappleFromRecordHeld{cmd.grapples, GrappleEscapeForced{f}}, nil
	case d20.RollAttempt:
		rolled, err := u.rollCheck(a, in.grapple)
		if err != nil {
			return nil, err
		}
		check = GrappleEscapeRolled{rolled}
		if rolled.Outcome() != d20.Success {
			return EscapeGrappleFromRecordHeld{cmd.grapples, check}, nil
		}
	default:
		panic(fmt.Errorf("combat: unreachable escape attempt %T", cmd.attempt))
	}
	t := cmd.grapples.Escape(cmd.target).(GrappleTransitionApplied)
	return EscapeGrappleFromRecordEscaped{t.grapples, check, t.endings[0]}, nil
}

// rollCheck makes the one roller request of an attempt, as d20.Resolver does, and
// resolves the check against the grapple's escape DC.
func (u EscapeGrappleFromRecord) rollCheck(a d20.RollAttempt, g Grapple) (d20.CheckResult, error) {
	mode := modifier.Resolve(a.Circumstances...)
	res, err := dice.Roll(u.roller, dice.MustExpression(mode.DiceCount(), dice.D20(), 0))
	if err != nil {
		return d20.CheckResult{}, err
	}
	faces := make([]int, 0, mode.DiceCount())
	for _, r := range res.Rolls() {
		faces = append(faces, r.Value())
	}
	roll, err := d20.NewRoll(mode, faces)
	if err != nil {
		return d20.CheckResult{}, srderr.ContractViolation(dice.Port, "roll", err.Error())
	}
	return d20.ResolveCheck(d20.AbilityCheck, roll, a.Modifiers, g.EscapeDC)
}
