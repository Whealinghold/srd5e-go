package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// UseDarkOnesOwnLuckCommand is the input of UseDarkOnesOwnLuck.Run. Charisma is the character's Charisma modifier, Kind the kind of
// the D20 Test the die is for and OnRoll whether the feature was already used on that roll. Construct it with a keyed literal. Run
// does not alter it.
type UseDarkOnesOwnLuckCommand struct {
	Character Character
	Charisma  ability.Modifier
	Kind      d20.Kind
	OnRoll    LuckOnRoll
}

// UseDarkOnesOwnLuckResult is the closed set of outcomes of UseDarkOnesOwnLuck.Run. Variants: UseDarkOnesOwnLuckApplied,
// UseDarkOnesOwnLuckRejected.
type UseDarkOnesOwnLuckResult interface{ isUseDarkOnesOwnLuckResult() }

// UseDarkOnesOwnLuckApplied is the outcome of a use of the feature.
type UseDarkOnesOwnLuckApplied struct {
	character Character
	d10       modifier.Modifier
	events    []Event
}

// Character returns the new Character, with one more use spent.
func (r UseDarkOnesOwnLuckApplied) Character() Character { return r.character }

// Bonus returns the d10 rolled as a Modifier for the host to add to the roll.
func (r UseDarkOnesOwnLuckApplied) Bonus() modifier.Modifier { return r.d10 }

// Events returns a new slice holding the DarkOnesOwnLuckUsed event.
func (r UseDarkOnesOwnLuckApplied) Events() []Event { return slices.Clone(r.events) }

// UseDarkOnesOwnLuckRejected is the outcome of a use a rule refused: the Character is unchanged and the roller was not called.
type UseDarkOnesOwnLuckRejected struct{ Rejection WarlockRejection }

func (UseDarkOnesOwnLuckApplied) isUseDarkOnesOwnLuckResult()  {}
func (UseDarkOnesOwnLuckRejected) isUseDarkOnesOwnLuckResult() {}

// UseDarkOnesOwnLuck is the use case that uses a Fiend Patron Warlock's Dark One's Own Luck (SRD 5.2.1, "Level 6: Dark One's Own
// Luck", p. 76; class-warlock.md FR-127 to FR-130): it rolls 1d10 through the dice port only when the use is allowed. It reads no
// content. Create it with NewUseDarkOnesOwnLuck; the zero value is not usable.
type UseDarkOnesOwnLuck struct {
	warlockUse
	roller Roller
}

// NewUseDarkOnesOwnLuck returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// for a nil roller.
func NewUseDarkOnesOwnLuck(roller Roller, opts ...Option) (UseDarkOnesOwnLuck, error) {
	if roller == nil {
		return UseDarkOnesOwnLuck{}, invalidInput("use Dark One's Own Luck without a dice roller")
	}
	u, err := newWarlockUse("use Dark One's Own Luck", nil, false, opts)
	return UseDarkOnesOwnLuck{warlockUse: u, roller: roller}, err
}

// Run uses the feature. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character, and, recorded on the span, for an invalid roll kind; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract; the
// Character is unchanged and the failure is recorded once and never logged. A refusal is a UseDarkOnesOwnLuckRejected result and the
// roller is not called; the profile is read for the rules, so a character with no Warlock level is WarlockNotAWarlock.
func (u UseDarkOnesOwnLuck) Run(ctx context.Context, cmd UseDarkOnesOwnLuckCommand) (UseDarkOnesOwnLuckResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanUseDarkOnesOwnLuck, ValueFeatureDarkOnesOwnLuck, cmd.Character, false)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return UseDarkOnesOwnLuckRejected{Rejection: rej}, nil
	}
	// A rolled value of 1 stands in for the die while the rules decide, so a refusal never reaches the roller.
	probe, err := AddDarkOnesOwnLuck(cmd.Character.ID(), call.state, call.profile, cmd.Charisma, cmd.Kind, cmd.OnRoll, 1)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	if refused, ok := probe.(WarlockRefused); ok {
		call.run.rejected(ctx, refused.rejection.Kind.String())
		return UseDarkOnesOwnLuckRejected{Rejection: refused.rejection}, nil
	}
	rolled, err := rollOne(u.roller, dice.D10())
	if err != nil {
		call.run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	t, err := AddDarkOnesOwnLuck(cmd.Character.ID(), call.state, call.profile, cmd.Charisma, cmd.Kind, cmd.OnRoll, rolled)
	if err != nil { // unreachable: dice.Roll answers 1 to 10 and the kind was valid
		call.run.commandError(err)
		return nil, err
	}
	next, changed, _, err := call.commit(ctx, cmd.Character, t)
	if err != nil {
		return nil, err
	}
	bonus, _ := changed.Bonus()
	return UseDarkOnesOwnLuckApplied{character: next, d10: bonus, events: changed.Events()}, nil
}
