package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseUncannyMetabolismCommand is the input of UseUncannyMetabolism.Run. Construct it with a
// keyed literal. Run does not alter it.
type UseUncannyMetabolismCommand struct {
	Character Character
	// RolledInitiative reports that the Monk rolled Initiative, the only trigger.
	RolledInitiative bool
}

// UseUncannyMetabolismResult is the closed set of outcomes of UseUncannyMetabolism.Run.
// Variants: UncannyMetabolismApplied, UncannyMetabolismNotAvailable.
type UseUncannyMetabolismResult interface{ isUseUncannyMetabolismResult() }

// UncannyMetabolismApplied is the outcome of a use of Uncanny Metabolism. Only Run builds one.
type UncannyMetabolismApplied struct {
	character  Character
	resolution UncannyMetabolismResolution
	events     []Event
}

// Character returns the new Character: every Focus Point regained and the feature used.
func (r UncannyMetabolismApplied) Character() Character { return r.character }

// Rolled returns the Martial Arts die rolled.
func (r UncannyMetabolismApplied) Rolled() int { return r.resolution.rolled }

// HitPointsToRegain returns the Monk level plus the die rolled, uncapped: the consumer
// caps it at the Hit Point maximum.
func (r UncannyMetabolismApplied) HitPointsToRegain() int { return r.resolution.hitPoints }

// FocusPointsRegained returns the Focus Points regained.
func (r UncannyMetabolismApplied) FocusPointsRegained() int { return r.resolution.regained }

// Events returns a new slice holding the UncannyMetabolismUsed event.
func (r UncannyMetabolismApplied) Events() []Event { return slices.Clone(r.events) }

// UncannyMetabolismNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged, there are no events and the roller was not called. Construct it with a keyed literal.
type UncannyMetabolismNotAvailable struct{ Reason MonkReason }

func (UncannyMetabolismApplied) isUseUncannyMetabolismResult()      {}
func (UncannyMetabolismNotAvailable) isUseUncannyMetabolismResult() {}

// UseUncannyMetabolism is the use case that uses a Monk's Uncanny Metabolism (SRD 5.2.1,
// "Level 2: Uncanny Metabolism", p. 51): it checks the use, rolls the Martial Arts die
// through the dice port only when it is available and emits the srd5e.character.feature_used
// signals. It holds no rules logic. Create it with NewUseUncannyMetabolism; the zero value
// is not usable. It is safe for concurrent use only if its roller, telemetry and clock are.
type UseUncannyMetabolism struct{ monkUse }

// NewUseUncannyMetabolism returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the zero
// progression or a nil roller.
func NewUseUncannyMetabolism(progression MonkProgression, roller Roller, opts ...Option) (UseUncannyMetabolism, error) {
	u, err := newMonkUse(progression, roller, opts)
	return UseUncannyMetabolism{u}, err
}

// Run uses Uncanny Metabolism for cmd.Character. A character with no Monk levels is
// UncannyMetabolismNotAvailable with MonkNotAMonk; otherwise the use is checked with
// UncannyMetabolism.Availability, and only an available use rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use
// case or a zero Character, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or breaks its contract; the Character is unchanged
// and the failure is recorded once and never logged.
func (u UseUncannyMetabolism) Run(ctx context.Context, cmd UseUncannyMetabolismCommand) (UseUncannyMetabolismResult, error) {
	if err := u.check(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run, mc, err := u.open(ctx, cmd.Character, SpanUseUncannyMetabolism, ValueFeatureUncannyMetabolism)
	defer run.end(ctx)
	if err != nil {
		return nil, err
	}
	if reason := monkReasonOf((UncannyMetabolism{}).Availability(mc.stats, mc.state, cmd.RolledInitiative)); reason != 0 {
		run.rejected(ctx, reason.String())
		return UncannyMetabolismNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollMartialArts(u.roller, mc.stats, 1)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := UncannyMetabolism{}.Resolve(mc.stats, mc.state, cmd.RolledInitiative, rolled)
	if err != nil { // unreachable: the use was available and the roll came from the Martial Arts die
		run.commandError(err)
		return nil, err
	}
	next, err := WithFeatureState(cmd.Character, MonkStateKey(), res.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return UncannyMetabolismApplied{
		character: next, resolution: res,
		events: []Event{UncannyMetabolismUsed{id: next.ID(), rolled: res.rolled, hitPoints: res.hitPoints, regained: res.regained}},
	}, nil
}
