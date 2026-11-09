package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollStonesEnduranceCommand is the input of RollStonesEndurance.Run: the incoming
// damage. Build it with NewRollStonesEnduranceCommand; the zero value is invalid. Run
// does not alter it.
type RollStonesEnduranceCommand struct {
	character Character
	traits    SpeciesTraits
	damage    int
}

// NewRollStonesEnduranceCommand returns the command for damage taken by c.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or a
// negative damage. It never panics.
func NewRollStonesEnduranceCommand(c Character, t SpeciesTraits, damage int) (RollStonesEnduranceCommand, error) {
	if c.IsZero() || damage < 0 {
		return RollStonesEnduranceCommand{}, invalidInput("roll stone's endurance: zero character or a negative damage")
	}
	return RollStonesEnduranceCommand{character: c, traits: t, damage: damage}, nil
}

// RollStonesEnduranceResult is the closed set of outcomes of RollStonesEndurance.Run.
// Variants: RollStonesEnduranceRolled, RollStonesEnduranceRejected.
type RollStonesEnduranceResult interface{ isRollStonesEnduranceResult() }

// RollStonesEnduranceRolled is the outcome of a roll: the 1d12, the Constitution
// modifier added, the reduction (never below 0), the damage left, the new
// SpeciesTraits and the event. Only Run builds one.
type RollStonesEnduranceRolled struct {
	roll, modifier, reduction, after int
	traits                           SpeciesTraits
	event                            StonesEnduranceUsed
}

// Roll returns the 1d12 rolled.
func (r RollStonesEnduranceRolled) Roll() int { return r.roll }

// Modifier returns the Constitution modifier added.
func (r RollStonesEnduranceRolled) Modifier() int { return r.modifier }

// Reduction returns the damage taken off: max(0, roll + modifier).
func (r RollStonesEnduranceRolled) Reduction() int { return r.reduction }

// DamageAfter returns the damage left: max(0, damage - reduction).
func (r RollStonesEnduranceRolled) DamageAfter() int { return r.after }

// Traits returns the new SpeciesTraits, with a Giant Ancestry use spent.
func (r RollStonesEnduranceRolled) Traits() SpeciesTraits { return r.traits }

// Event returns the StonesEnduranceUsed event.
func (r RollStonesEnduranceRolled) Event() StonesEnduranceUsed { return r.event }

// RollStonesEnduranceRejected is the outcome of a use a rule refused; nothing was
// rolled. Construct it with a keyed literal.
type RollStonesEnduranceRejected struct{ Rejection SpeciesTraitRejection }

func (RollStonesEnduranceRolled) isRollStonesEnduranceResult()   {}
func (RollStonesEnduranceRejected) isRollStonesEnduranceResult() {}

// RollStonesEndurance is the use case of the Goliath benefit Stone's Endurance (SRD
// 5.2.1, p. 85). It uses the dice port and telemetry only. Create it with
// NewRollStonesEndurance; the zero value is not usable. It is safe for concurrent use
// only if its roller, telemetry and clock are.
type RollStonesEndurance struct {
	roller Roller
	sig    operationSignals
}

// NewRollStonesEndurance returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollStonesEndurance(roller Roller, opts ...Option) (RollStonesEndurance, error) {
	if roller == nil {
		return RollStonesEndurance{}, invalidInput("roll stone's endurance without a dice roller")
	}
	return RollStonesEndurance{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run checks, before any roll and in this order: the zero SpeciesTraits
// (SpeciesBehaviourUnknown), a species or ancestry that is not Stone
// (SpeciesUseNotOffered) and a spent Giant Ancestry pool (SpeciesUseExhausted); then it
// makes one request of 1d12, adds the Constitution modifier and spends a use. The
// use is spent only after a successful roll.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or a zero command, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. It returns a *srderr.PortError (error type "dice_roller_error")
// with a nil result and nothing spent when the roller fails, panics, answers another
// expression or rolls outside 1 to 12; the failure is recorded once and never logged.
func (u RollStonesEndurance) Run(ctx context.Context, cmd RollStonesEnduranceCommand) (RollStonesEnduranceResult, error) {
	if u.roller == nil || cmd.character.IsZero() {
		return nil, invalidInput("roll stone's endurance: zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollStonesEndurance, ValueOperationRollStonesEndurance)
	defer run.end(ctx)
	reject := func(r SpeciesTraitRejection) (RollStonesEnduranceResult, error) {
		run.finish(ctx, ValueResultRejected, r.String())
		return RollStonesEnduranceRejected{Rejection: r}, nil
	}
	switch {
	case cmd.traits.IsZero():
		return reject(SpeciesBehaviourUnknown{})
	case !cmd.traits.offersRequest(StonesEnduranceRequest{}, GiantAncestryUse):
		return reject(SpeciesUseNotOffered{})
	case cmd.traits.ExpendedOf(GiantAncestryUse) >= GiantAncestryUse.Limit(cmd.character.TotalLevel()):
		return reject(SpeciesUseExhausted{})
	}
	res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D12(), 0))
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	mod := cmd.character.AbilityScores().Modifier(ability.Constitution).Value()
	reduction, after := StonesEnduranceRule{}.Reduce(res.Total(), mod, cmd.damage)
	next := cmd.traits
	next.expended[GiantAncestryUse-1]++
	run.finish(ctx, ValueResultResolved, "")
	return RollStonesEnduranceRolled{roll: res.Total(), modifier: mod, reduction: reduction, after: after, traits: next,
		event: StonesEnduranceUsed{id: cmd.character.ID(), roll: res.Total(), reduction: reduction}}, nil
}
