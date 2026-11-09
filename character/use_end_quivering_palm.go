package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// EndQuiveringPalmCommand is the input of EndQuiveringPalm.Run. Construct it with a keyed
// literal. Run does not alter it.
type EndQuiveringPalmCommand struct {
	Character Character
	// End is how the vibrations are ended.
	End QuiveringPalmEnd
	// SamePlane reports that the Monk and the target are on the same plane of existence.
	SamePlane bool
	// Outcome is the target's Constitution saving throw for an end with effect; ignored for a harmless end.
	Outcome d20.Outcome
}

// EndQuiveringPalmResult is the closed set of outcomes of EndQuiveringPalm.Run. Variants:
// QuiveringPalmEndApplied, QuiveringPalmEndNotAvailable.
type EndQuiveringPalmResult interface{ isEndQuiveringPalmResult() }

// QuiveringPalmEndApplied is the outcome of ending the vibrations. Only Run builds one.
type QuiveringPalmEndApplied struct {
	character  Character
	resolution QuiveringPalmEndResolution
	events     []Event
}

// Character returns the new Character, with no vibrations.
func (r QuiveringPalmEndApplied) Character() Character { return r.character }

// Resolution returns the way it ended, the DC used and the Force damage applied.
func (r QuiveringPalmEndApplied) Resolution() QuiveringPalmEndResolution { return r.resolution }

// Events returns a new slice holding the QuiveringPalmEnded event.
func (r QuiveringPalmEndApplied) Events() []Event { return slices.Clone(r.events) }

// QuiveringPalmEndNotAvailable is the outcome of an end a rule refused: the Character is
// unchanged, the vibrations stay, there are no events and the roller was not called.
// Construct it with a keyed literal.
type QuiveringPalmEndNotAvailable struct{ Reason MonkReason }

func (QuiveringPalmEndApplied) isEndQuiveringPalmResult()      {}
func (QuiveringPalmEndNotAvailable) isEndQuiveringPalmResult() {}

// EndQuiveringPalm is the use case that ends the Quivering Palm vibrations (SRD 5.2.1,
// "Level 17: Quivering Palm", p. 52): it checks the use, rolls 10d12 through the dice port
// only for an available end with effect and emits the srd5e.character.feature_used
// signals. It holds no rules logic. Create it with NewEndQuiveringPalm; the zero value is
// not usable. It is safe for concurrent use only if its roller, telemetry and clock are.
type EndQuiveringPalm struct{ monkUse }

// NewEndQuiveringPalm returns the use case. Options: WithTelemetry, WithClock; the others
// are ignored. It returns an error matching srderr.ErrInvalidInput for the zero progression
// or a nil roller.
func NewEndQuiveringPalm(progression MonkProgression, roller Roller, opts ...Option) (EndQuiveringPalm, error) {
	u, err := newMonkUse(progression, roller, opts)
	return EndQuiveringPalm{u}, err
}

// Run ends the vibrations for cmd.Character. A character with no Monk levels is
// QuiveringPalmEndNotAvailable with MonkNotAMonk; otherwise the end is checked with
// QuiveringPalm.EndAvailability, and only an available end with effect rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use
// case, a zero Character, or an end with effect whose Outcome is neither d20.Success nor
// d20.Failure, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It
// returns a *srderr.PortError (error type "dice_roller_error") with a nil result when the
// roller fails, panics or breaks its contract; the Character is unchanged.
func (u EndQuiveringPalm) Run(ctx context.Context, cmd EndQuiveringPalmCommand) (EndQuiveringPalmResult, error) {
	if cmd.End == QuiveringPalmEndWithEffect && cmd.Outcome != d20.Success && cmd.Outcome != d20.Failure {
		return nil, invalidInput("Quivering Palm saving throw outcome is neither success nor failure")
	}
	if err := u.check(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run, mc, err := u.open(ctx, cmd.Character, SpanEndQuiveringPalm, ValueFeatureQuiveringPalmEnd)
	defer run.end(ctx)
	if err != nil {
		return nil, err
	}
	if reason := monkReasonOf((QuiveringPalm{}).EndAvailability(mc.stats, mc.state, cmd.End, cmd.SamePlane)); reason != 0 {
		run.rejected(ctx, reason.String())
		return QuiveringPalmEndNotAvailable{Reason: reason}, nil
	}
	rolled := 0
	if cmd.End == QuiveringPalmEndWithEffect {
		if rolled, err = rollTotal(u.roller, QuiveringPalm{}.DamageDice()); err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
	}
	res, err := QuiveringPalm{}.Finish(mc.stats, mc.state, MonkAbilitiesOf(cmd.Character), cmd.End, cmd.SamePlane, cmd.Outcome, rolled)
	if err != nil { // unreachable: the end was available and the roll came from 10d12
		run.commandError(err)
		return nil, err
	}
	next, err := WithFeatureState(cmd.Character, MonkStateKey(), res.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return QuiveringPalmEndApplied{character: next, resolution: res, events: []Event{res.Event(next.ID())}}, nil
}
