package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ResolveThrownItemInput holds the values NewResolveThrownItemCommand validates. Build it with
// a keyed literal. The engine models no map: the host reports the target's facts, the
// distance, the sight and the Cover (SRD 5.2.1, "Acid", p. 94).
type ResolveThrownItemInput struct {
	// Kind is the item thrown.
	Kind ThrownItemKind
	// Thrower is the creature throwing the item; never zero. It is named in the events.
	Thrower CombatantID
	// DexterityModifier and ProficiencyBonus are the thrower's, for the Difficulty Class.
	DexterityModifier ability.Modifier
	ProficiencyBonus  character.ProficiencyBonus
	// Target is a creature, with the facts the rules read, or an object.
	Target ThrownTarget
	// DistanceFeet is the distance to the target, 0 or more.
	DistanceFeet int
	// Sight says whether the thrower can see the target.
	Sight ThrownSight
	// Cover says whether Cover lets the target be targeted.
	Cover ThrownCover
	// Attempt is how the target makes the Dexterity saving throw: a d20.RollAttempt carrying
	// every modifier and circumstance (Cover's bonus included), d20.ChooseToFail or
	// d20.AutomaticFailure. It is never nil, and ignored when the target has no save.
	Attempt d20.SaveAttempt
}

// ResolveThrownItemCommand is the immutable input of ResolveThrownItem.Run. Build it with
// NewResolveThrownItemCommand; the zero value is invalid.
type ResolveThrownItemCommand struct{ in ResolveThrownItemInput }

// NewResolveThrownItemCommand returns the command. The attempt's circumstances are copied.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with the violated field for
// an invalid kind, a zero thrower, an invalid target, a negative distance, a sight or Cover
// that is not valid, or an attempt that is not a d20.RollAttempt, d20.ChooseToFail or
// d20.AutomaticFailure. It never panics. It returns a LimitExceededError for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewResolveThrownItemCommand(in ResolveThrownItemInput) (ResolveThrownItemCommand, error) {
	attempt, ok := checkSave(in.Attempt)
	switch {
	case !in.Kind.Valid():
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldKind, int64(in.Kind))
	case in.Thrower.IsZero():
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldThrower, 0)
	case !validThrownTarget(in.Target):
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldTarget, 0)
	case in.DistanceFeet < 0:
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldDistanceFeet, int64(in.DistanceFeet))
	case !in.Sight.Valid():
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldSight, int64(in.Sight))
	case !in.Cover.Valid():
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldCover, int64(in.Cover))
	case !ok:
		return ResolveThrownItemCommand{}, itemUseInvalid(ItemUseFieldAttempt, 0)
	case attemptOverCap(in.Attempt) != nil:
		return ResolveThrownItemCommand{}, attemptOverCap(in.Attempt)
	}
	in.Attempt = attempt
	return ResolveThrownItemCommand{in}, nil
}

// MustResolveThrownItemCommand is NewResolveThrownItemCommand for values known to be valid. It
// panics with the error NewResolveThrownItemCommand returns, and only then.
func MustResolveThrownItemCommand(in ResolveThrownItemInput) ResolveThrownItemCommand {
	c, err := NewResolveThrownItemCommand(in)
	if err != nil {
		panic(err)
	}
	return c
}

// ResolveThrownItemResult is the closed set of outcomes of ResolveThrownItem.Run. Variants:
// ThrownItemResolved, ThrownItemNotPossible.
type ResolveThrownItemResult interface{ isResolveThrownItemResult() }

// ThrownItemResolved: the item was thrown. Nothing is applied: the host applies the effect,
// spends the cost and decrements the item. Not comparable.
type ThrownItemResolved struct {
	item        ThrownItemKind
	save        ThrownSave
	effect      ThrownEffect
	consumption ItemConsumption
	events      []ItemUseEvent
}

// Item returns the item thrown.
func (r ThrownItemResolved) Item() ThrownItemKind { return r.item }

// Cost returns ItemActionCostReplacesOneAttack, for the host to record with
// ThrownItemRules.Replacement.
func (r ThrownItemResolved) Cost() ItemActionCost { return ItemUseRules{}.CostOf(r.item.ItemUseKind()) }

// Save returns the target's saving throw, or the skipped save.
func (r ThrownItemResolved) Save() ThrownSave { return r.save }

// Effect returns what the item did to the target.
func (r ThrownItemResolved) Effect() ThrownEffect { return r.effect }

// Consumption returns the item used up: one for a flask, none for a Net.
func (r ThrownItemResolved) Consumption() ItemConsumption { return r.consumption }

// Events returns a copy of the events: one ThrownItemUsed.
func (r ThrownItemResolved) Events() []ItemUseEvent { return slices.Clone(r.events) }

// ThrownItemNotPossible: the item cannot be thrown; no die was rolled and nothing consumed.
type ThrownItemNotPossible struct{ Reason ThrownNotPossibleReason }

func (ThrownItemResolved) isResolveThrownItemResult()    {}
func (ThrownItemNotPossible) isResolveThrownItemResult() {}

// ResolveThrownItem is the use case of throwing Acid, Alchemist's Fire, Holy Water, Oil or a
// Net (SRD 5.2.1, pp. 94-98). It composes the pure d20 functions and never calls d20.Resolver
// or another use case. Create it with NewResolveThrownItem; the zero value is not usable.
type ResolveThrownItem struct {
	in   instrumenter
	flow attackFlow
}

// NewResolveThrownItem returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewResolveThrownItem(roller Roller, opts ...Option) (ResolveThrownItem, error) {
	if roller == nil {
		return ResolveThrownItem{}, invalidInput("thrown item without a dice roller")
	}
	o := newOptions(opts)
	return ResolveThrownItem{newInstrumenter(o.tel, o.clock), attackFlow{roller}}, nil
}

// Run checks that the item can be thrown (ThrownItemRules.Possibility), then has the target
// make its Dexterity saving throw against ItemSaveDCOf, unless it has nothing to save against
// (Holy Water at a creature that is neither a Fiend nor an Undead; a Net at a Huge or
// Gargantuan one). Dice are requested in this order: the save's d20s (two with Advantage or
// Disadvantage; none for a forced failure or a skipped save), then, only on a failed save of an
// item that deals damage, one damage expression. A natural 20 or 1 on the save is no automatic
// success or failure, there is no half damage and no Critical Hit.
//
// It records the shared signals with the attribute srd5e.combat.item_use.item. It returns an
// error matching srderr.ErrInvalidInput, before any signal, for the zero use case or command,
// and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails,
// panics or answers another expression; the failure is recorded once and never logged.
func (u ResolveThrownItem) Run(ctx context.Context, cmd ResolveThrownItemCommand) (ResolveThrownItemResult, error) {
	if cmd.in.Thrower.IsZero() || u.flow.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	in := cmd.in
	item := telemetry.String(attrItemUseItem, in.Kind.String())
	return run(ctx, u.in, combatOperation{valueOperationResolveThrownItem, spanResolveThrownItem},
		func(context.Context) (ResolveThrownItemResult, error) { return u.resolve(in) },
		func(r ResolveThrownItemResult) description {
			switch v := r.(type) {
			case ThrownItemResolved:
				return description{Class: resultApplied, Name: thrownResultName(v), Events: widen(v.events), Extras: []telemetry.Attr{item}}
			case ThrownItemNotPossible:
				return description{Class: resultRefused, Name: v.Reason.String(), Extras: []telemetry.Attr{item}}
			default:
				panic(unreachableError("resolve thrown item result", r))
			}
		})
}

// thrownResultName is the srd5e.combat.result of a resolved throw.
func thrownResultName(r ThrownItemResolved) string {
	switch e := r.effect.(type) {
	case ThrownDamageDealt:
		return valueResultDamaged
	case ThrownDamageAndBurningStarted:
		return valueResultDamagedAndBurning
	case ThrownOilCovered:
		return valueResultCoveredInOil
	case ThrownRestrained:
		return valueResultRestrained
	case ThrownNoEffect:
		if e.reason == ThrownNoEffectSaveSucceeded {
			return valueResultSaved
		}
		return valueResultNoEffect
	default:
		panic(unreachableError("thrown effect", r.effect))
	}
}

// resolve is the work of Run for a valid command.
func (u ResolveThrownItem) resolve(in ResolveThrownItemInput) (ResolveThrownItemResult, error) {
	rules := ThrownItemRules{}
	if reason := rules.possibility(in.Kind, in.Target, in.DistanceFeet, in.Sight, in.Cover); reason != 0 {
		return ThrownItemNotPossible{reason}, nil
	}
	dc := ItemSaveDCOf(in.DexterityModifier, in.ProficiencyBonus)
	var save ThrownSave
	var effect ThrownEffect
	if skip := rules.SaveSkip(in.Kind, in.Target); skip != 0 {
		save, effect = ThrownSaveSkipped{skip}, ThrownNoEffect{skip}
	} else {
		var err error
		if save, err = u.save(in.Attempt, dc); err != nil {
			return nil, err
		}
		if effect, err = u.effect(in.Kind, save); err != nil {
			return nil, err
		}
	}
	used := ThrownItemUsed{itemUseEventBase{first: in.Thrower}, in.Kind}
	if c, ok := in.Target.(ThrownAtCreature); ok {
		used.second = c.ID
	}
	return ThrownItemResolved{in.Kind, save, effect, rules.Consumption(in.Kind), []ItemUseEvent{used}}, nil
}

// save makes the target's Dexterity saving throw: a RollAttempt resolves its roll mode once,
// rolls that many d20s and compares with d20.ResolveCheck; a forced failure draws no die.
func (u ResolveThrownItem) save(attempt d20.SaveAttempt, dc d20.DifficultyClass) (ThrownSave, error) {
	reason := d20.Automatic
	switch a := attempt.(type) {
	case d20.RollAttempt:
		roll, err := u.flow.rollD20(modifier.Resolve(a.Circumstances...))
		if err != nil {
			return nil, err
		}
		res, err := d20.ResolveCheck(d20.SavingThrow, roll, a.Modifiers, dc)
		return ThrownSaveRolled{dc, res}, err
	case d20.ChooseToFail:
		reason = d20.ChosenToFail
	}
	res, err := d20.FailWithoutRoll(d20.SavingThrow, reason)
	return ThrownSaveForced{dc, res}, err
}

// effect is what the item does after the save: nothing on a success, else the rules' effect
// with the damage rolled once as one expression.
func (u ResolveThrownItem) effect(kind ThrownItemKind, save ThrownSave) (ThrownEffect, error) {
	if save.Succeeded() {
		return ThrownNoEffect{ThrownNoEffectSaveSucceeded}, nil
	}
	total := 0
	if expr, ok := ThrownItemFactsOf(kind).Damage(); ok {
		res, err := dice.Roll(u.flow.roller, expr)
		if err != nil {
			return nil, err
		}
		total = res.Total()
	}
	return ThrownItemRules{}.OnFailedSave(kind, total)
}
