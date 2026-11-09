package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseSecondWindCommand is the input of UseSecondWind.Run. Construct it with a keyed
// literal. Run does not alter it.
type UseSecondWindCommand struct{ Character Character }

// UseSecondWindResult is the closed set of outcomes of UseSecondWind.Run. Variants:
// SecondWindApplied, SecondWindNotAvailable.
type UseSecondWindResult interface{ isUseSecondWindResult() }

// SecondWindApplied is the outcome of a use of Second Wind. Only Run builds one.
type SecondWindApplied struct {
	character  Character
	resolution SecondWindResolution
	tactical   TacticalShiftOffer
	events     []Event
}

// Character returns the new Character, with one more Second Wind use expended.
func (r SecondWindApplied) Character() Character { return r.character }

// HitPointsToRegain returns the d10 roll plus the Fighter level, uncapped: the
// consumer caps it at the Hit Point maximum and loses the excess.
func (r SecondWindApplied) HitPointsToRegain() int { return r.resolution.hitPoints }

// Rolled returns the d10 rolled.
func (r SecondWindApplied) Rolled() int { return r.resolution.rolled }

// TacticalShift returns TacticalShiftOffered from Fighter level 5, else NoTacticalShift.
func (r SecondWindApplied) TacticalShift() TacticalShiftOffer { return r.tactical }

// Cost returns FighterBonusAction.
func (SecondWindApplied) Cost() FighterActionCost { return FighterBonusAction }

// Events returns a new slice holding the SecondWindUsed event.
func (r SecondWindApplied) Events() []Event { return slices.Clone(r.events) }

// SecondWindNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged, there are no events and the roller was not called. Construct it with a
// keyed literal.
type SecondWindNotAvailable struct{ Reason FighterReason }

func (SecondWindApplied) isUseSecondWindResult()      {}
func (SecondWindNotAvailable) isUseSecondWindResult() {}

// UseSecondWind is the use case that uses a Fighter's Second Wind (SRD 5.2.1, "Level
// 1: Second Wind", p. 48): it checks the use, rolls 1d10 through the dice port only
// when it is available and emits the srd5e.character.feature_used signals. It holds
// no rules logic. Create it with NewUseSecondWind; the zero value is not usable. It
// is safe for concurrent use only if its roller, telemetry and clock are.
type UseSecondWind struct {
	progression FighterProgression
	roller      Roller
	sig         featureSignals
	set         bool
}

// NewUseSecondWind returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the
// zero progression or a nil roller.
func NewUseSecondWind(progression FighterProgression, roller Roller, opts ...Option) (UseSecondWind, error) {
	if progression.IsZero() || roller == nil {
		return UseSecondWind{}, invalidInput("use Second Wind without a Fighter progression or a dice roller")
	}
	return UseSecondWind{progression: progression, roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run uses Second Wind for cmd.Character. A character with no Fighter levels is
// SecondWindNotAvailable with FighterNotAFighter; otherwise the use is checked with
// SecondWind.Availability, and only an available use rolls.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or a zero Character, and ctx.Err() unwrapped, before any signal,
// when ctx is already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or breaks its
// contract; the Character is unchanged and the failure is recorded once and never
// logged. A refused use is a SecondWindNotAvailable result.
func (u UseSecondWind) Run(ctx context.Context, cmd UseSecondWindCommand) (UseSecondWindResult, error) {
	if err := fighterCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseSecondWind, ValueClassFighter, ValueFeatureSecondWind)
	defer run.end(ctx)
	fc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason == 0 {
		if un, ok := (SecondWind{}).Availability(fc.stats, fc.state).(SecondWindUnavailable); ok {
			reason = un.Reason
		}
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return SecondWindNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollOne(u.roller, dice.D10())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := SecondWind{}.Resolve(fc.stats, fc.state, rolled)
	if err != nil { // unreachable: dice.Roll answers 1 to 10 and the use was available
		run.commandError(err)
		return nil, err
	}
	next, err := WithFeatureState(cmd.Character, FighterStateKey(), res.state)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return SecondWindApplied{
		character: next, resolution: res, tactical: FighterEffects{}.TacticalShift(fc.stats.level),
		events: []Event{SecondWindUsed{id: next.ID(), rolled: res.rolled, hitPoints: res.hitPoints}},
	}, nil
}
