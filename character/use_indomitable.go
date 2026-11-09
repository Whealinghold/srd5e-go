package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseIndomitableCommand is the input of UseIndomitable.Run: the character, the
// saving throw it failed and the index of the die to replace among the dice of that
// roll (the only die, or either die under Advantage or Disadvantage). Construct it
// with a keyed literal. Run does not alter it.
type UseIndomitableCommand struct {
	Character   Character
	FailedSave  d20.CheckResult
	ReplacedDie int
}

// UseIndomitableResult is the closed set of outcomes of UseIndomitable.Run.
// Variants: IndomitableSucceeded, IndomitableStillFailed, IndomitableNotAvailable.
type UseIndomitableResult interface{ isUseIndomitableResult() }

// indomitableUsed is the body shared by the two rolled outcomes.
type indomitableUsed struct {
	character Character
	rolled    int
	check     d20.CheckResult
	events    []Event
}

// IndomitableSucceeded is the outcome of a reroll whose new total meets or exceeds
// the DC. Only Run builds one.
type IndomitableSucceeded struct{ used indomitableUsed }

// Character returns the new Character, with one more Indomitable use expended.
func (r IndomitableSucceeded) Character() Character { return r.used.character }

// Rolled returns the d20 rolled, which must be used.
func (r IndomitableSucceeded) Rolled() int { return r.used.rolled }

// NewTotal returns the recomputed total: the used die, the modifiers and the Fighter level.
func (r IndomitableSucceeded) NewTotal() int { return r.used.check.Total() }

// Check returns the recomputed saving throw, whose Roll has the replaced die.
func (r IndomitableSucceeded) Check() d20.CheckResult { return r.used.check }

// Events returns a new slice holding the IndomitableUsed event.
func (r IndomitableSucceeded) Events() []Event { return slices.Clone(r.used.events) }

// IndomitableStillFailed is the outcome of a reroll whose new total is still below
// the DC: a normal rule outcome, not an error, and the use is expended all the same.
// Only Run builds one.
type IndomitableStillFailed struct{ used indomitableUsed }

// Character returns the new Character, with one more Indomitable use expended.
func (r IndomitableStillFailed) Character() Character { return r.used.character }

// Rolled returns the d20 rolled, which must be used.
func (r IndomitableStillFailed) Rolled() int { return r.used.rolled }

// NewTotal returns the recomputed total: the used die, the modifiers and the Fighter level.
func (r IndomitableStillFailed) NewTotal() int { return r.used.check.Total() }

// Check returns the recomputed saving throw, whose Roll has the replaced die.
func (r IndomitableStillFailed) Check() d20.CheckResult { return r.used.check }

// Events returns a new slice holding the IndomitableUsed event.
func (r IndomitableStillFailed) Events() []Event { return slices.Clone(r.used.events) }

// IndomitableNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged and the roller was not called. Construct it with a keyed literal.
type IndomitableNotAvailable struct{ Reason FighterReason }

func (IndomitableSucceeded) isUseIndomitableResult()    {}
func (IndomitableStillFailed) isUseIndomitableResult()  {}
func (IndomitableNotAvailable) isUseIndomitableResult() {}

// UseIndomitable is the use case that uses a Fighter's Indomitable on a failed
// saving throw (SRD 5.2.1, "Level 9: Indomitable", p. 48): it checks the use, rolls
// 1d20 through the dice port only when it is available and emits the
// srd5e.character.feature_used signals. It holds no rules logic. Create it with
// NewUseIndomitable; the zero value is not usable. It is safe for concurrent use only
// if its roller, telemetry and clock are.
type UseIndomitable struct {
	progression FighterProgression
	roller      Roller
	sig         featureSignals
	set         bool
}

// NewUseIndomitable returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the
// zero progression or a nil roller.
func NewUseIndomitable(progression FighterProgression, roller Roller, opts ...Option) (UseIndomitable, error) {
	if progression.IsZero() || roller == nil {
		return UseIndomitable{}, invalidInput("use Indomitable without a Fighter progression or a dice roller")
	}
	return UseIndomitable{progression: progression, roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run uses Indomitable for cmd.Character on cmd.FailedSave. The reasons are checked
// in the order not a Fighter, feature not gained, not a saving throw, die not
// rolled, save not failed, no uses remaining; only an available use rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or a zero FailedSave, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns a *srderr.PortError
// (error type "dice_roller_error") with a nil result when the roller fails, panics or
// breaks its contract; the failure is recorded once and never logged.
func (u UseIndomitable) Run(ctx context.Context, cmd UseIndomitableCommand) (UseIndomitableResult, error) {
	if err := fighterCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	if cmd.FailedSave.Roll().IsZero() {
		return nil, invalidInput("use Indomitable on a zero check result")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseIndomitable, ValueClassFighter, ValueFeatureIndomitable)
	defer run.end(ctx)
	fc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason == 0 {
		if un, ok := (Indomitable{}).Availability(fc.stats, fc.state, cmd.FailedSave, cmd.ReplacedDie).(IndomitableUnavailable); ok {
			reason = un.Reason
		}
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return IndomitableNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollOne(u.roller, dice.D20())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := Indomitable{}.Resolve(fc.stats, fc.state, cmd.FailedSave, cmd.ReplacedDie, rolled)
	if err != nil { // unreachable: dice.Roll answers 1 to 20 and the use was available
		run.commandError(err)
		return nil, err
	}
	var out indomitableOutcome
	succeeded := false
	switch r := res.(type) {
	case IndomitableResolutionSucceeded:
		out, succeeded = r.outcome, true
	default:
		out = res.(IndomitableResolutionStillFailed).outcome
	}
	next, err := WithFeatureState(cmd.Character, FighterStateKey(), out.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	used := indomitableUsed{character: next, rolled: rolled, check: out.check, events: []Event{IndomitableUsed{id: next.ID(), rolled: rolled}}}
	if succeeded {
		run.succeeded(ctx)
		return IndomitableSucceeded{used: used}, nil
	}
	run.stillFailed(ctx)
	return IndomitableStillFailed{used: used}, nil
}
