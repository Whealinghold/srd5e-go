package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RedirectDeflectedAttackCommand is the input of RedirectDeflectedAttack.Run. Construct it
// with a keyed literal. Run does not alter it.
type RedirectDeflectedAttackCommand struct {
	Character Character
	Request   MonkRedirectRequest
	// Outcome is the target's Dexterity saving throw, d20.Success or d20.Failure.
	Outcome d20.Outcome
}

// RedirectDeflectedAttackResult is the closed set of outcomes of
// RedirectDeflectedAttack.Run. Variants: RedirectApplied, RedirectNotAvailable.
type RedirectDeflectedAttackResult interface{ isRedirectDeflectedAttackResult() }

// RedirectApplied is the outcome of a redirect. Only Run builds one.
type RedirectApplied struct {
	character  Character
	resolution RedirectResolution
	events     []Event
}

// Character returns the new Character, with 1 Focus Point expended.
func (r RedirectApplied) Character() Character { return r.character }

// Resolution returns the DC used, the damage on a failed save and the damage applied.
func (r RedirectApplied) Resolution() RedirectResolution { return r.resolution }

// Events returns a new slice holding the DeflectedForceRedirected event.
func (r RedirectApplied) Events() []Event { return slices.Clone(r.events) }

// RedirectNotAvailable is the outcome of a redirect a rule refused: the Character is
// unchanged, there are no events and the roller was not called. Construct it with a keyed literal.
type RedirectNotAvailable struct{ Reason MonkReason }

func (RedirectApplied) isRedirectDeflectedAttackResult()      {}
func (RedirectNotAvailable) isRedirectDeflectedAttackResult() {}

// RedirectDeflectedAttack is the use case that redirects a deflected attack's force (SRD
// 5.2.1, "Level 3: Deflect Attacks", p. 51): it checks the use, rolls two Martial Arts dice
// through the dice port only when it is available and emits the srd5e.character.feature_used
// signals. The modifiers and the Proficiency Bonus are the Character's. It holds no rules
// logic. Create it with NewRedirectDeflectedAttack; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type RedirectDeflectedAttack struct{ monkUse }

// NewRedirectDeflectedAttack returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the zero
// progression or a nil roller.
func NewRedirectDeflectedAttack(progression MonkProgression, roller Roller, opts ...Option) (RedirectDeflectedAttack, error) {
	u, err := newMonkUse(progression, roller, opts)
	return RedirectDeflectedAttack{u}, err
}

// Run redirects the force for cmd.Character. A character with no Monk levels is
// RedirectNotAvailable with MonkNotAMonk; otherwise the use is checked with
// DeflectRedirect.Availability, and only an available redirect rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use
// case, a zero Character or an Outcome other than d20.Success or d20.Failure, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller
// fails, panics or breaks its contract; the Character is unchanged.
func (u RedirectDeflectedAttack) Run(ctx context.Context, cmd RedirectDeflectedAttackCommand) (RedirectDeflectedAttackResult, error) {
	if cmd.Outcome != d20.Success && cmd.Outcome != d20.Failure {
		return nil, invalidInput("redirect saving throw outcome is neither success nor failure")
	}
	if err := u.check(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run, mc, err := u.open(ctx, cmd.Character, SpanRedirectDeflectedAttack, ValueFeatureDeflectRedirect)
	defer run.end(ctx)
	if err != nil {
		return nil, err
	}
	if reason := monkReasonOf((DeflectRedirect{}).Availability(mc.stats, mc.state, cmd.Request)); reason != 0 {
		run.rejected(ctx, reason.String())
		return RedirectNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollMartialArts(u.roller, mc.stats, 2)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := DeflectRedirect{}.Resolve(mc.stats, mc.state, MonkAbilitiesOf(cmd.Character), cmd.Request, rolled, cmd.Outcome)
	if err != nil { // unreachable: the use was available and the roll came from two Martial Arts dice
		run.commandError(err)
		return nil, err
	}
	next, err := WithFeatureState(cmd.Character, MonkStateKey(), res.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return RedirectApplied{
		character: next, resolution: res,
		events: []Event{DeflectedForceRedirected{id: next.ID(), dc: res.dc, damageType: res.damageType, applied: res.applied}},
	}, nil
}
