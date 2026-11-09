package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ResolveDeflectAttacksCommand is the input of ResolveDeflectAttacks.Run. Construct it with
// a keyed literal. Run does not alter it.
type ResolveDeflectAttacksCommand struct {
	Character Character
	// DamageTypes are the types the hitting attack's damage includes.
	DamageTypes []content.DamageType
	// TotalDamage is the attack's total damage, at least 0.
	TotalDamage int
}

// ResolveDeflectAttacksResult is the closed set of outcomes of ResolveDeflectAttacks.Run.
// Variants: DeflectAttacksApplied, DeflectAttacksNotAvailable.
type ResolveDeflectAttacksResult interface{ isResolveDeflectAttacksResult() }

// DeflectAttacksApplied is the outcome of Deflect Attacks. Only Run builds one.
type DeflectAttacksApplied struct {
	character  Character
	resolution DeflectResolution
	events     []Event
}

// Character returns the Character, unchanged: Deflect Attacks expends nothing.
func (r DeflectAttacksApplied) Character() Character { return r.character }

// Resolution returns the d10 rolled, the reduction and the damage after.
func (r DeflectAttacksApplied) Resolution() DeflectResolution { return r.resolution }

// Events returns a new slice holding the AttackDeflected event.
func (r DeflectAttacksApplied) Events() []Event { return slices.Clone(r.events) }

// DeflectAttacksNotAvailable is the outcome of a use a rule refused: there are no events and
// the roller was not called. Construct it with a keyed literal.
type DeflectAttacksNotAvailable struct{ Reason MonkReason }

func (DeflectAttacksApplied) isResolveDeflectAttacksResult()      {}
func (DeflectAttacksNotAvailable) isResolveDeflectAttacksResult() {}

// ResolveDeflectAttacks is the use case that reduces an attack's damage with Deflect
// Attacks (SRD 5.2.1, "Level 3: Deflect Attacks", p. 51): it checks the use, rolls 1d10
// through the dice port only when it is available and emits the srd5e.character.feature_used
// signals. The Dexterity modifier is the Character's base score modifier. It holds no rules
// logic. Create it with NewResolveDeflectAttacks; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type ResolveDeflectAttacks struct{ monkUse }

// NewResolveDeflectAttacks returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the zero
// progression or a nil roller.
func NewResolveDeflectAttacks(progression MonkProgression, roller Roller, opts ...Option) (ResolveDeflectAttacks, error) {
	u, err := newMonkUse(progression, roller, opts)
	return ResolveDeflectAttacks{u}, err
}

// Run deflects the attack for cmd.Character. A character with no Monk levels is
// DeflectAttacksNotAvailable with MonkNotAMonk.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use
// case, a zero Character or a negative TotalDamage, and ctx.Err() unwrapped, before any
// signal, when ctx is already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract.
func (u ResolveDeflectAttacks) Run(ctx context.Context, cmd ResolveDeflectAttacksCommand) (ResolveDeflectAttacksResult, error) {
	if cmd.TotalDamage < 0 {
		return nil, invalidInput("Deflect Attacks with a negative damage total")
	}
	if err := u.check(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run, mc, err := u.open(ctx, cmd.Character, SpanResolveDeflectAttacks, ValueFeatureDeflectAttacks)
	defer run.end(ctx)
	if err != nil {
		return nil, err
	}
	if reason := monkReasonOf((DeflectAttacks{}).Availability(mc.stats, cmd.DamageTypes)); reason != 0 {
		run.rejected(ctx, reason.String())
		return DeflectAttacksNotAvailable{Reason: reason}, nil
	}
	rolled, err := rollTotal(u.roller, deflectDie())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	res, err := DeflectAttacks{}.Resolve(mc.stats, MonkAbilitiesOf(cmd.Character), cmd.DamageTypes, cmd.TotalDamage, rolled)
	if err != nil { // unreachable: the use was available and the roll came from 1d10
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return DeflectAttacksApplied{
		character: cmd.Character, resolution: res,
		events: []Event{AttackDeflected{id: cmd.Character.ID(), rolled: res.rolled, reduction: res.reduction, after: res.after}},
	}, nil
}
