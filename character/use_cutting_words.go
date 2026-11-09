package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// UseCuttingWordsCommand is the input of UseCuttingWords.Run. Subclass is the Bard subclass the character holds ("" for
// none). Trigger is the roll or damage the Bard answers. Construct it with a keyed literal. Run does not alter it.
type UseCuttingWordsCommand struct {
	Character Character
	Subclass  content.SubclassID
	Trigger   CuttingWordsTrigger
}

// UseCuttingWordsResult is the closed set of outcomes of UseCuttingWords.Run. Variants: UseCuttingWordsApplied,
// UseCuttingWordsRejected.
type UseCuttingWordsResult interface{ isUseCuttingWordsResult() }

// UseCuttingWordsApplied is the outcome of a use: the Bardic Inspiration die was rolled and subtracted. Only Run builds one.
type UseCuttingWordsApplied struct {
	character Character
	applied   CuttingWordsApplied
}

// Character returns the new Character, with one Bardic Inspiration use expended.
func (r UseCuttingWordsApplied) Character() Character { return r.character }

// Applied returns the reduced roll or damage, the pool and the event.
func (r UseCuttingWordsApplied) Applied() CuttingWordsApplied { return r.applied }

// Events returns a new slice holding the BardicInspirationExpended event.
func (r UseCuttingWordsApplied) Events() []Event { return []Event{r.applied.event} }

// UseCuttingWordsRejected is the outcome of a use a rule refused: the Character is unchanged, nothing was rolled and no
// use is expended. Construct it with a keyed literal.
type UseCuttingWordsRejected struct{ Reason BardReason }

func (UseCuttingWordsApplied) isUseCuttingWordsResult()  {}
func (UseCuttingWordsRejected) isUseCuttingWordsResult() {}

// UseCuttingWords is the use case that uses a College of Lore Bard's Cutting Words (SRD 5.2.1, "Level 3: Cutting Words",
// p. 35): it plans the use, rolls the Bardic Die through the dice port only for a use that is allowed, applies it and
// emits the srd5e.character.feature_used signals. The Reaction cost is the host's. It holds no rules logic. Create it with
// NewUseCuttingWords; the zero value is not usable. Safe for concurrent use only if its roller, telemetry and clock are.
type UseCuttingWords struct{ bardUse }

// NewUseCuttingWords returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewUseCuttingWords(roller Roller, opts ...Option) (UseCuttingWords, error) {
	u, err := newBardUse("use Cutting Words", roller, opts)
	return UseCuttingWords{u}, err
}

// Run uses Cutting Words for cmd.Character on cmd.Trigger. The reasons are checked in the order of PlanCuttingWords.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a nil
// trigger, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns an error matching
// srderr.ErrInvalidInput for a trigger the rules cannot take or a Bard that holds no pool, and a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract; the Character is unchanged and
// the failure is recorded once and never logged. A refused use is a result.
func (u UseCuttingWords) Run(ctx context.Context, cmd UseCuttingWordsCommand) (UseCuttingWordsResult, error) {
	ctx, run, err := u.start(ctx, cmd.Character.IsZero() || cmd.Trigger == nil, SpanUseCuttingWords, ValueFeatureCuttingWords)
	if err != nil {
		return nil, err
	}
	defer run.end(ctx)
	facts, isBard, err := readBardFacts(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if !isBard {
		run.rejected(ctx, BardNotABard.String())
		return UseCuttingWordsRejected{Reason: BardNotABard}, nil
	}
	plan, reason, err := PlanCuttingWords(cmd.Character.ID(), facts.level, cmd.Subclass, facts.pool, facts.charisma, cmd.Trigger)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason != 0 {
		run.rejected(ctx, reason.String())
		return UseCuttingWordsRejected{Reason: reason}, nil
	}
	rolled, err := rollOne(u.roller, plan.Die().Die())
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	applied, err := ApplyCuttingWords(plan, rolled)
	if err != nil {
		fail(ctx, run, err)
		return nil, err
	}
	next, err := WithFeatureState(cmd.Character, BardicInspirationPoolKey(), applied.pool)
	if err != nil { // unreachable: the key is valid and the state was read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return UseCuttingWordsApplied{character: next, applied: applied}, nil
}
