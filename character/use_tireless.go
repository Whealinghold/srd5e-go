package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseTirelessCommand is the input of UseTireless.Run: the Ranger and its Wisdom modifier. Construct it with a keyed literal. Run
// does not alter it.
type UseTirelessCommand struct {
	Character Character
	Wisdom    ability.Modifier
}

// UseTirelessResult is the closed set of outcomes of UseTireless.Run. Variants: UseTirelessAccepted, UseTirelessRejected,
// UseTirelessNotARanger.
type UseTirelessResult interface{ isUseTirelessResult() }

// UseTirelessAccepted is the outcome of a use of Tireless. Only Run builds one.
type UseTirelessAccepted struct {
	character Character
	state     RangerState
	rolled    int
	temporary int
	events    []Event
}

// Character returns the new Character, with one more use of Tireless expended.
func (r UseTirelessAccepted) Character() Character { return r.character }

// State returns the new RangerState.
func (r UseTirelessAccepted) State() RangerState { return r.state }

// Rolled returns the d8 rolled.
func (r UseTirelessAccepted) Rolled() int { return r.rolled }

// TemporaryHitPoints returns the roll plus the greater of 1 and the Wisdom modifier.
func (r UseTirelessAccepted) TemporaryHitPoints() int { return r.temporary }

// Cost returns RangerMagicAction.
func (UseTirelessAccepted) Cost() RangerActionCost { return RangerMagicAction }

// Events returns a new slice holding the TirelessUsed event.
func (r UseTirelessAccepted) Events() []Event { return slices.Clone(r.events) }

// UseTirelessRejected is a use a rule refused: the Character is unchanged, there are no events and the roller was not called.
// Construct it with a keyed literal.
type UseTirelessRejected struct{ Reason RangerReason }

// UseTirelessNotARanger is the outcome for a character with no Ranger levels.
type UseTirelessNotARanger struct{}

func (UseTirelessAccepted) isUseTirelessResult()   {}
func (UseTirelessRejected) isUseTirelessResult()   {}
func (UseTirelessNotARanger) isUseTirelessResult() {}

// UseTireless is the use case that uses a Ranger's Tireless Temporary Hit Points (SRD 5.2.1, "Level 10: Tireless", p. 59): it
// checks the use, rolls 1d8 through the dice port only when it is available and emits the srd5e.character.feature_used signals. It
// holds no rules logic. Create it with NewUseTireless; the zero value is not usable. It is safe for concurrent use only if its
// roller, telemetry and clock are.
type UseTireless struct {
	rangerUse
	roller Roller
}

// NewUseTireless returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for the zero progression or a nil roller.
func NewUseTireless(p RangerProgression, roller Roller, opts ...Option) (UseTireless, error) {
	if roller == nil {
		return UseTireless{}, invalidInput("use Tireless without a dice roller")
	}
	u, err := newRangerUse("use Tireless", p, opts)
	return UseTireless{rangerUse: u, roller: roller}, err
}

// Run uses Tireless for cmd.Character. A character with no Ranger levels is UseTirelessNotARanger; below the level of Tireless the
// use is rejected with RangerFeatureNotAvailable and with none left with RangerNoUsesRemaining, and neither rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract; the Character is unchanged and the
// failure is recorded once and never logged. A refused use is a result.
func (u UseTireless) Run(ctx context.Context, cmd UseTirelessCommand) (UseTirelessResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseTireless, ValueClassRanger, ValueFeatureTireless)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return UseTirelessNotARanger{}, nil
	}
	reason := RangerReason(0)
	switch {
	case !u.progression.has(RangerFeatureTireless, level):
		reason = RangerFeatureNotAvailable
	case state.TirelessRemaining(cmd.Wisdom) == 0:
		reason = RangerNoUsesRemaining
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return UseTirelessRejected{Reason: reason}, nil
	}
	rolled, err := rollOne(u.roller, dice.D8())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	temporary, err := TirelessTemporaryHitPoints(rolled, cmd.Wisdom)
	if err != nil { // unreachable: dice.Roll answers 1 to 8
		run.commandError(err)
		return nil, err
	}
	state.tirelessSpent++
	next, err := WithFeatureState(cmd.Character, RangerStateKey(), state)
	if err != nil { // unreachable: the key is valid and the state was read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return UseTirelessAccepted{character: next, state: state, rolled: rolled, temporary: temporary,
		events: []Event{TirelessUsed{id: next.ID(), rolled: rolled, temporary: temporary}}}, nil
}
