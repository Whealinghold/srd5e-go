package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseTacticalMindCommand is the input of UseTacticalMind.Run: the character and the
// ability check it failed. Construct it with a keyed literal. Run does not alter it.
type UseTacticalMindCommand struct {
	Character   Character
	FailedCheck d20.CheckResult
}

// UseTacticalMindResult is the closed set of outcomes of UseTacticalMind.Run.
// Variants: TacticalMindSucceeded, TacticalMindStillFailed, TacticalMindNotAvailable.
type UseTacticalMindResult interface{ isUseTacticalMindResult() }

// tacticalMindUsed is the body shared by the two rolled outcomes.
type tacticalMindUsed struct {
	character Character
	rolled    int
	check     d20.CheckResult
	events    []Event
}

// TacticalMindSucceeded is the outcome of a use that turned the check into a
// success; a Second Wind use is expended. Only Run builds one.
type TacticalMindSucceeded struct{ used tacticalMindUsed }

// Character returns the new Character, with one more Second Wind use expended.
func (r TacticalMindSucceeded) Character() Character { return r.used.character }

// Rolled returns the d10 rolled.
func (r TacticalMindSucceeded) Rolled() int { return r.used.rolled }

// NewTotal returns the check total with the d10 added.
func (r TacticalMindSucceeded) NewTotal() int { return r.used.check.Total() }

// Check returns the check with the d10 added; its Roll is the original.
func (r TacticalMindSucceeded) Check() d20.CheckResult { return r.used.check }

// Events returns a new slice holding the TacticalMindUsed event.
func (r TacticalMindSucceeded) Events() []Event { return slices.Clone(r.used.events) }

// TacticalMindStillFailed is the outcome of a use whose new total is still below the
// DC: a normal rule outcome, not an error, and no use is expended. Only Run builds one.
type TacticalMindStillFailed struct{ used tacticalMindUsed }

// Character returns the Character, unchanged.
func (r TacticalMindStillFailed) Character() Character { return r.used.character }

// Rolled returns the d10 rolled.
func (r TacticalMindStillFailed) Rolled() int { return r.used.rolled }

// NewTotal returns the check total with the d10 added.
func (r TacticalMindStillFailed) NewTotal() int { return r.used.check.Total() }

// Check returns the check with the d10 added; its Roll is the original.
func (r TacticalMindStillFailed) Check() d20.CheckResult { return r.used.check }

// Events returns an empty slice: nothing happened to the Character.
func (r TacticalMindStillFailed) Events() []Event { return slices.Clone(r.used.events) }

// TacticalMindNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged and the roller was not called. Construct it with a keyed literal.
type TacticalMindNotAvailable struct{ Reason FighterReason }

func (TacticalMindSucceeded) isUseTacticalMindResult()    {}
func (TacticalMindStillFailed) isUseTacticalMindResult()  {}
func (TacticalMindNotAvailable) isUseTacticalMindResult() {}

// UseTacticalMind is the use case that uses a Fighter's Tactical Mind on a failed
// ability check (SRD 5.2.1, "Level 2: Tactical Mind", p. 48): it checks the use,
// rolls 1d10 through the dice port only when it is available and emits the
// srd5e.character.feature_used signals. It holds no rules logic. Create it with
// NewUseTacticalMind; the zero value is not usable. It is safe for concurrent use
// only if its roller, telemetry and clock are.
type UseTacticalMind struct {
	progression FighterProgression
	roller      Roller
	sig         featureSignals
	set         bool
}

// NewUseTacticalMind returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the
// zero progression or a nil roller.
func NewUseTacticalMind(progression FighterProgression, roller Roller, opts ...Option) (UseTacticalMind, error) {
	if progression.IsZero() || roller == nil {
		return UseTacticalMind{}, invalidInput("use Tactical Mind without a Fighter progression or a dice roller")
	}
	return UseTacticalMind{progression: progression, roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run uses Tactical Mind for cmd.Character on cmd.FailedCheck. The reasons are
// checked in the order not a Fighter, feature not gained, not an ability check,
// check not failed, no uses remaining; only an available use rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or a zero FailedCheck, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns a *srderr.PortError
// (error type "dice_roller_error") with a nil result when the roller fails, panics or
// breaks its contract; the failure is recorded once and never logged.
func (u UseTacticalMind) Run(ctx context.Context, cmd UseTacticalMindCommand) (UseTacticalMindResult, error) {
	if err := fighterCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	if cmd.FailedCheck.Roll().IsZero() {
		return nil, invalidInput("use Tactical Mind on a zero check result")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseTacticalMind, ValueClassFighter, ValueFeatureTacticalMind)
	defer run.end(ctx)
	fc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason == 0 {
		if un, ok := (TacticalMind{}).Availability(fc.stats, fc.state, cmd.FailedCheck).(TacticalMindUnavailable); ok {
			reason = un.Reason
		}
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return TacticalMindNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollOne(u.roller, dice.D10())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := TacticalMind{}.Resolve(fc.stats, fc.state, cmd.FailedCheck, rolled)
	if err != nil { // unreachable: dice.Roll answers 1 to 10 and the use was available
		run.commandError(err)
		return nil, err
	}
	switch r := res.(type) {
	case TacticalMindResolutionSucceeded:
		next, err := WithFeatureState(cmd.Character, FighterStateKey(), r.outcome.state)
		if err != nil { // unreachable: the key is valid and the state is read from the same key
			run.commandError(err)
			return nil, err
		}
		run.succeeded(ctx)
		return TacticalMindSucceeded{used: tacticalMindUsed{character: next, rolled: rolled, check: r.outcome.check,
			events: []Event{TacticalMindUsed{id: next.ID(), rolled: rolled}}}}, nil
	default:
		failed := res.(TacticalMindResolutionStillFailed)
		run.stillFailed(ctx)
		return TacticalMindStillFailed{used: tacticalMindUsed{character: cmd.Character, rolled: rolled, check: failed.outcome.check, events: []Event{}}}, nil
	}
}
