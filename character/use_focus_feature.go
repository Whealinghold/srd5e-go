package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseFocusFeatureCommand is the input of UseFocusFeature.Run. Construct it with a keyed
// literal. Run does not alter it.
type UseFocusFeatureCommand struct {
	Character Character
	Use       MonkFocusUse
}

// UseFocusFeatureResult is the closed set of outcomes of UseFocusFeature.Run. Variants:
// FocusFeatureApplied, FocusFeatureNotAvailable.
type UseFocusFeatureResult interface{ isUseFocusFeatureResult() }

// FocusFeatureApplied is the outcome of a use of Flurry of Blows, Patient Defense or Step
// of the Wind. Only Run builds one.
type FocusFeatureApplied struct {
	character  Character
	resolution MonkFocusResolution
	events     []Event
}

// Character returns the new Character, with the Focus Point expended when one was spent.
func (r FocusFeatureApplied) Character() Character { return r.character }

// Resolution returns what the use granted: strikes, actions, Temporary Hit Points, the carried creature.
func (r FocusFeatureApplied) Resolution() MonkFocusResolution { return r.resolution }

// Events returns a new slice holding the FocusPointsExpended event when a point was spent.
func (r FocusFeatureApplied) Events() []Event { return slices.Clone(r.events) }

// FocusFeatureNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged, there are no events and the roller was not called. Construct it with a keyed literal.
type FocusFeatureNotAvailable struct{ Reason MonkReason }

func (FocusFeatureApplied) isUseFocusFeatureResult()      {}
func (FocusFeatureNotAvailable) isUseFocusFeatureResult() {}

// UseFocusFeature is the use case that uses a Monk's Flurry of Blows, Patient Defense or
// Step of the Wind (SRD 5.2.1, "Level 2: Monk's Focus", p. 50): it checks the use, rolls
// two Martial Arts dice through the dice port only when Patient Defense gives Temporary
// Hit Points, and emits the srd5e.character.feature_used signals. It holds no rules logic.
// Create it with NewUseFocusFeature; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type UseFocusFeature struct{ monkUse }

// NewUseFocusFeature returns the use case. Options: WithTelemetry, WithClock; the others
// are ignored. It returns an error matching srderr.ErrInvalidInput for the zero progression
// or a nil roller.
func NewUseFocusFeature(progression MonkProgression, roller Roller, opts ...Option) (UseFocusFeature, error) {
	u, err := newMonkUse(progression, roller, opts)
	return UseFocusFeature{u}, err
}

// Run uses the Focus feature for cmd.Character. A character with no Monk levels is
// FocusFeatureNotAvailable with MonkNotAMonk; otherwise the use is checked with
// MonkFocus.Availability, and only an available use that gives Temporary Hit Points rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use
// case or a zero Character, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or breaks its contract; the Character is unchanged
// and the failure is recorded once and never logged. A refused use is a
// FocusFeatureNotAvailable result.
func (u UseFocusFeature) Run(ctx context.Context, cmd UseFocusFeatureCommand) (UseFocusFeatureResult, error) {
	if err := u.check(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run, mc, err := u.open(ctx, cmd.Character, SpanUseFocusFeature, cmd.Use.Feature.String())
	defer run.end(ctx)
	if err != nil {
		return nil, err
	}
	if reason := monkReasonOf((MonkFocus{}).Availability(mc.stats, mc.state, cmd.Use)); reason != 0 {
		run.rejected(ctx, reason.String())
		return FocusFeatureNotAvailable{Reason: reason}, nil
	}
	rolled := 0
	if n := (MonkFocus{}).TemporaryHitPointDice(mc.stats, cmd.Use); n > 0 {
		if rolled, err = rollMartialArts(u.roller, mc.stats, n); err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
	}
	res, err := MonkFocus{}.Resolve(mc.stats, mc.state, cmd.Use, rolled)
	if err != nil { // unreachable: the use was available and the roll came from the Martial Arts dice
		run.commandError(err)
		return nil, err
	}
	next, err := WithFeatureState(cmd.Character, MonkStateKey(), res.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	var events []Event
	if res.spent > 0 {
		events = append(events, FocusPointsExpended{id: next.ID(), feature: res.feature, count: res.spent})
	}
	run.succeeded(ctx)
	return FocusFeatureApplied{character: next, resolution: res, events: events}, nil
}
