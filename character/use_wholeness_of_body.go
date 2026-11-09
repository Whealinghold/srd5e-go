package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseWholenessOfBodyCommand is the input of UseWholenessOfBody.Run. Construct it with a
// keyed literal. Run does not alter it.
type UseWholenessOfBodyCommand struct {
	Character Character
	// Subclass is the Monk's chosen subclass, or empty for none; the Character holds none.
	Subclass content.SubclassID
}

// UseWholenessOfBodyResult is the closed set of outcomes of UseWholenessOfBody.Run.
// Variants: WholenessOfBodyApplied, WholenessOfBodyNotAvailable.
type UseWholenessOfBodyResult interface{ isUseWholenessOfBodyResult() }

// WholenessOfBodyApplied is the outcome of a use of Wholeness of Body. Only Run builds one.
type WholenessOfBodyApplied struct {
	character  Character
	resolution WholenessOfBodyResolution
	events     []Event
}

// Character returns the new Character, with one more use spent.
func (r WholenessOfBodyApplied) Character() Character { return r.character }

// Rolled returns the Martial Arts die rolled.
func (r WholenessOfBodyApplied) Rolled() int { return r.resolution.rolled }

// HitPointsToRegain returns the die plus the Wisdom modifier, at least 1, uncapped: the
// consumer caps it at the Hit Point maximum.
func (r WholenessOfBodyApplied) HitPointsToRegain() int { return r.resolution.hitPoints }

// Events returns a new slice holding the WholenessOfBodyUsed event.
func (r WholenessOfBodyApplied) Events() []Event { return slices.Clone(r.events) }

// WholenessOfBodyNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged, there are no events and the roller was not called. Construct it with a keyed literal.
type WholenessOfBodyNotAvailable struct{ Reason MonkReason }

func (WholenessOfBodyApplied) isUseWholenessOfBodyResult()      {}
func (WholenessOfBodyNotAvailable) isUseWholenessOfBodyResult() {}

// UseWholenessOfBody is the use case that uses the Warrior of the Open Hand's Wholeness of
// Body (SRD 5.2.1, "Level 6: Wholeness of Body", p. 52): it checks the use, rolls the
// Martial Arts die through the dice port only when it is available and emits the
// srd5e.character.feature_used signals. The Wisdom modifier is the Character's. It holds no
// rules logic. Create it with NewUseWholenessOfBody; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type UseWholenessOfBody struct{ monkUse }

// NewUseWholenessOfBody returns the use case. Options: WithTelemetry, WithClock; the others
// are ignored. It returns an error matching srderr.ErrInvalidInput for the zero progression
// or a nil roller.
func NewUseWholenessOfBody(progression MonkProgression, roller Roller, opts ...Option) (UseWholenessOfBody, error) {
	u, err := newMonkUse(progression, roller, opts)
	return UseWholenessOfBody{u}, err
}

// Run uses Wholeness of Body for cmd.Character. A character with no Monk levels is
// WholenessOfBodyNotAvailable with MonkNotAMonk; otherwise the use is checked with
// WholenessOfBody.Availability, and only an available use rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use
// case or a zero Character, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or breaks its contract; the Character is unchanged.
func (u UseWholenessOfBody) Run(ctx context.Context, cmd UseWholenessOfBodyCommand) (UseWholenessOfBodyResult, error) {
	if err := u.check(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run, mc, err := u.open(ctx, cmd.Character, SpanUseWholenessOfBody, ValueFeatureWholenessOfBody)
	defer run.end(ctx)
	if err != nil {
		return nil, err
	}
	wisdom := cmd.Character.AbilityScores().Modifier(ability.Wisdom)
	if reason := monkReasonOf((WholenessOfBody{}).Availability(mc.stats, mc.state, cmd.Subclass, wisdom)); reason != 0 {
		run.rejected(ctx, reason.String())
		return WholenessOfBodyNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollMartialArts(u.roller, mc.stats, 1)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := WholenessOfBody{}.Resolve(mc.stats, mc.state, cmd.Subclass, wisdom, rolled)
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
	return WholenessOfBodyApplied{
		character: next, resolution: res, events: []Event{WholenessOfBodyUsed{id: next.ID(), rolled: res.rolled, hitPoints: res.hitPoints}},
	}, nil
}
