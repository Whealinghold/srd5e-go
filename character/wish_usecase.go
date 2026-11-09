package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// WishResolver holds the two Wish use cases, ResolveWish and RollWishStressDamage. It holds only the
// dice and telemetry ports. Create it with NewWishResolver; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type WishResolver struct {
	roller Roller
	sig    operationSignals
}

// NewWishResolver returns the use cases rolling through roller. Options: WithTelemetry, WithClock;
// the others are ignored. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewWishResolver(roller Roller, opts ...Option) (WishResolver, error) {
	if roller == nil {
		return WishResolver{}, invalidInput("resolve a wish without a dice roller")
	}
	return WishResolver{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// ResolveWishCommand is the input of ResolveWish: the character and the choice. It takes no cast
// event and no slot, because Wish may be cast through a slot, a feature or an item. Construct it
// with a keyed literal. ResolveWish does not alter it.
type ResolveWishCommand struct {
	Character Character
	Choice    WishChoice
}

// ResolveWishResult is the closed set of outcomes of ResolveWish. Variants: ResolveWishDuplicated,
// ResolveWishApplied, ResolveWishRejected.
type ResolveWishResult interface{ isResolveWishResult() }

// ResolveWishDuplicated is the outcome of a duplicated spell: the Character equal to the command's,
// the WishDuplicate handoff and the event WishDuplicated. No dice were rolled and no stress added.
// Only ResolveWish builds one.
type ResolveWishDuplicated struct{ app WishApplication }

// Character returns the Character, equal to the command's.
func (r ResolveWishDuplicated) Character() Character { return r.app.character }

// Effect returns the WishDuplicate handoff.
func (r ResolveWishDuplicated) Effect() WishEffect { return r.app.effect }

// Events returns a new slice of the events.
func (r ResolveWishDuplicated) Events() []Event { return r.app.Events() }

// ResolveWishApplied is the outcome of an effect other than a duplication, with the stress rolled
// and written to the Character. Only ResolveWish builds one.
type ResolveWishApplied struct{ app WishApplication }

// Character returns the new Character, holding the new WishStress.
func (r ResolveWishApplied) Character() Character { return r.app.character }

// Feats returns the new TakenFeats for Sudden Learning, the zero TakenFeats otherwise.
func (r ResolveWishApplied) Feats() TakenFeats { return r.app.feats }

// Effect returns the effect handed off.
func (r ResolveWishApplied) Effect() WishEffect { return r.app.effect }

// Events returns a new slice of the events: those of the effect, then WishStressEndured.
func (r ResolveWishApplied) Events() []Event { return r.app.Events() }

// ResolveWishRejected is the outcome of a choice the rules refuse; nothing was rolled and nothing
// changed. Only ResolveWish builds one.
type ResolveWishRejected struct{ rejection ResolveWishRejection }

// Rejection returns why the choice was refused.
func (r ResolveWishRejected) Rejection() ResolveWishRejection { return r.rejection }

func (ResolveWishDuplicated) isResolveWishResult() {}
func (ResolveWishApplied) isResolveWishResult()    {}
func (ResolveWishRejected) isResolveWishResult()   {}

// ResolveWish resolves a cast Wish. Rejections are checked first, before any die (CheckWish); an
// accepted choice other than DuplicateSpell then rolls the 2d4 days and the 1d100 percentile of the
// stress, in that order, and ApplyWish applies the choice and the stress.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero resolver, a
// zero Character, a nil or zero choice or a stored WishStress of another type, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a *srderr.PortError (error
// type "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression; nothing is applied then, the failure is recorded once and never logged.
func (u WishResolver) ResolveWish(ctx context.Context, cmd ResolveWishCommand) (ResolveWishResult, error) {
	if u.roller == nil {
		return nil, invalidInput("resolve a wish with the zero resolver")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	check, err := CheckWish(cmd.Character, cmd.Choice)
	if err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanResolveWish, ValueOperationResolveWish, telemetry.String(AttrWishEffect, wishEffectValue(cmd.Choice)))
	defer run.end(ctx)
	if rej, ok := check.Rejection(); ok {
		run.finish(ctx, ValueResultRejected, rej.kind.String())
		return ResolveWishRejected{rejection: rej}, nil
	}
	var roll WishStressRoll
	if WishRequiresStress(cmd.Choice) {
		if roll, err = u.rollStress(); err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
	}
	app, err := ApplyWish(cmd.Character, cmd.Choice, roll)
	if err != nil { // unreachable: the choice passed CheckWish and the roll is valid
		run.commandError(err)
		return nil, err
	}
	if !WishRequiresStress(cmd.Choice) {
		run.finish(ctx, ValueResultDuplicated, "")
		return ResolveWishDuplicated{app: app}, nil
	}
	run.withAttrs(telemetry.Bool(AttrWishLockout, roll.LocksOut())).finish(ctx, ValueResultApplied, "")
	return ResolveWishApplied{app: app}, nil
}

// rollStress rolls the 2d4 days, then the 1d100 percentile.
func (u WishResolver) rollStress() (WishStressRoll, error) {
	days, err := dice.Roll(u.roller, dice.MustExpression(2, dice.D4(), 0))
	if err != nil {
		return WishStressRoll{}, err
	}
	percentile, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D100(), 0))
	if err != nil {
		return WishStressRoll{}, err
	}
	return NewWishStressRoll(days.Total(), percentile.Total())
}

// RollWishStressDamageCommand is the input of RollWishStressDamage: the character, the cast being
// paid for, its spell and whether it is the duplicate of a Wish. Construct it with a keyed literal.
// RollWishStressDamage does not alter it.
type RollWishStressDamageCommand struct {
	Character Character
	SpellCast SpellCast
	Spell     content.Spell
	Origin    WishCastOrigin
}

// RollWishStressDamageResult is the closed set of outcomes of RollWishStressDamage. Variants:
// RollWishStressDamageRolled, RollWishStressDamageNone.
type RollWishStressDamageResult interface{ isRollWishStressDamageResult() }

// RollWishStressDamageRolled is the outcome of a cast that costs Necrotic damage: the damage, for
// combat to apply without reduction, and the event WishStressDamageRolled. Only RollWishStressDamage
// builds one.
type RollWishStressDamageRolled struct {
	damage UnpreventableDamage
	events []Event
}

// Damage returns the Necrotic damage that cannot be reduced or prevented.
func (r RollWishStressDamageRolled) Damage() UnpreventableDamage { return r.damage }

// Events returns a new slice of the events.
func (r RollWishStressDamageRolled) Events() []Event { return append([]Event(nil), r.events...) }

// RollWishStressDamageNone is the outcome of a cast that costs nothing, with the reason. No die was
// rolled. Only RollWishStressDamage builds one.
type RollWishStressDamageNone struct{ reason WishDamageNoneReason }

// Reason returns why the cast costs nothing.
func (r RollWishStressDamageNone) Reason() WishDamageNoneReason { return r.reason }

func (RollWishStressDamageRolled) isRollWishStressDamageResult() {}
func (RollWishStressDamageNone) isRollWishStressDamageResult()   {}

// RollWishStressDamage rolls the Necrotic damage a cast pays while the stress is active: one d10 per
// level of the spell for that casting (WishLevelOfCast). A cantrip, an inactive stress and the
// duplicate of a Wish cost nothing and roll nothing. The Character is never changed.
//
// It returns an error matching srderr.ErrInvalidInput, before any die or signal, for the zero
// resolver, a zero Character, a cast of another character or another spell, a cast level below the
// spell's or on a cantrip, an unknown origin and a stored WishStress of another type, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It returns a *srderr.PortError (error
// type "dice_roller_error") with a nil result when the roller fails, panics or answers another
// expression; the failure is recorded once and never logged.
func (u WishResolver) RollWishStressDamage(ctx context.Context, cmd RollWishStressDamageCommand) (RollWishStressDamageResult, error) {
	if u.roller == nil {
		return nil, invalidInput("roll wish stress damage with the zero resolver")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	plan, err := planWishStressDamage(cmd.Character, cmd.SpellCast, cmd.Spell, cmd.Origin)
	if err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollWishStressDamage, ValueOperationRollWishStressDamage)
	defer run.end(ctx)
	if plan.none != 0 {
		run.finish(ctx, ValueResultNone, "")
		return RollWishStressDamageNone{reason: plan.none}, nil
	}
	rolled, err := dice.Roll(u.roller, plan.dice)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	damage, err := NewUnpreventableDamage(content.DamageNecrotic, rolled.Total())
	if err != nil { // unreachable: a roll of 1 or more d10 totals at least 1
		run.commandError(err)
		return nil, err
	}
	run.finish(ctx, ValueResultDamage, "")
	return RollWishStressDamageRolled{damage: damage, events: []Event{WishStressDamageRolled{id: cmd.Character.ID(), level: plan.level, amount: damage.amount}}}, nil
}
