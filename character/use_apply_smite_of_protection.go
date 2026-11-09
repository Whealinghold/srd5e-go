package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// ApplySmiteOfProtectionCommand is the input of ApplySmiteOfProtection.Run. Subclass is the subclass the Paladin holds ("" for
// none); DivineSmiteCast and Incapacitated are host facts. Construct it with a keyed literal. Run does not alter it.
type ApplySmiteOfProtectionCommand struct {
	Character       Character
	Subclass        content.SubclassID
	DivineSmiteCast bool
	Charisma        ability.Modifier
	Incapacitated   bool
}

// ApplySmiteOfProtectionResult is the closed set of outcomes of ApplySmiteOfProtection.Run. Variants:
// ApplySmiteOfProtectionApplied, ApplySmiteOfProtectionNoEffect, ApplySmiteOfProtectionNotAPaladin.
type ApplySmiteOfProtectionResult interface{ isApplySmiteOfProtectionResult() }

// ApplySmiteOfProtectionApplied is the outcome of a cast of Divine Smite that gives Half Cover. Only Run builds one.
type ApplySmiteOfProtectionApplied struct {
	effect SmiteOfProtectionEffect
	events []Event
}

// Effect returns the cover data for the host to apply.
func (r ApplySmiteOfProtectionApplied) Effect() SmiteOfProtectionEffect { return r.effect }

// Events returns a new slice holding the SmiteOfProtectionTriggered event.
func (r ApplySmiteOfProtectionApplied) Events() []Event { return slices.Clone(r.events) }

// ApplySmiteOfProtectionNoEffect is the outcome when Smite of Protection gives nothing: a normal success of the evaluation, with no
// event. Reasons: below the feature's level, no Oath of Devotion, the aura inactive, or no Divine Smite cast.
type ApplySmiteOfProtectionNoEffect struct{}

// ApplySmiteOfProtectionNotAPaladin is the outcome for a character with no Paladin levels.
type ApplySmiteOfProtectionNotAPaladin struct{}

func (ApplySmiteOfProtectionApplied) isApplySmiteOfProtectionResult()     {}
func (ApplySmiteOfProtectionNoEffect) isApplySmiteOfProtectionResult()    {}
func (ApplySmiteOfProtectionNotAPaladin) isApplySmiteOfProtectionResult() {}

// ApplySmiteOfProtection is the use case that evaluates the Oath of Devotion's Smite of Protection when Divine Smite is cast
// (SRD 5.2.1, "Level 15: Smite of Protection", p. 57). It changes no state. Create it with NewApplySmiteOfProtection; the zero
// value is not usable. It is safe for concurrent use only if its telemetry and clock are.
type ApplySmiteOfProtection struct{ paladinUse }

// NewApplySmiteOfProtection returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression.
func NewApplySmiteOfProtection(p PaladinProgression, opts ...Option) (ApplySmiteOfProtection, error) {
	u, err := newPaladinUse("apply Smite of Protection", p, opts)
	return ApplySmiteOfProtection{u}, err
}

// Run evaluates Smite of Protection for cmd.Character. It records outcome success for the effect and for no effect, and failure
// only for a character with no Paladin levels.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u ApplySmiteOfProtection) Run(ctx context.Context, cmd ApplySmiteOfProtectionCommand) (ApplySmiteOfProtectionResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanApplySmiteOfProtection, ValueClassPaladin, ValueFeatureSmiteOfProtection)
	defer run.end(ctx)
	level, _, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return ApplySmiteOfProtectionNotAPaladin{}, nil
	}
	oath := oathChosen(cmd.Subclass)
	aura := PaladinAuras{}.Compute(u.progression, level, oath, cmd.Charisma, cmd.Incapacitated)
	run.succeeded(ctx)
	if !smiteOfProtectionApplies(u.progression, level, oath, cmd.DivineSmiteCast, aura) {
		return ApplySmiteOfProtectionNoEffect{}, nil
	}
	return ApplySmiteOfProtectionApplied{events: []Event{SmiteOfProtectionTriggered{id: cmd.Character.ID()}}}, nil
}
