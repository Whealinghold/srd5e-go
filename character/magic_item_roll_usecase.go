package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// RechargeMagicItemCommand is the input of RechargeMagicItem.Do: the holder, the
// item copy, its charges, its schedule and recovery (from MagicItemFacts) and the
// moment the host reports. Construct it with a keyed literal. Do does not alter it.
type RechargeMagicItemCommand struct {
	CharacterID Identifier
	Instance    MagicItemInstanceID
	Charges     ItemCharges
	Time        content.ItemRechargeTime
	Recovery    content.ChargeRecovery
	Moment      RechargeMoment
}

// RechargeMagicItemResult is the closed set of outcomes of RechargeMagicItem.Do.
// Variants: RechargeMagicItemApplied, RechargeMagicItemNotDue.
type RechargeMagicItemResult interface{ isRechargeMagicItemResult() }

// RechargeMagicItemApplied: the item recharged.
type RechargeMagicItemApplied struct {
	charges ItemCharges
	event   MagicItemRecharged
}

// Charges returns the new charges.
func (r RechargeMagicItemApplied) Charges() ItemCharges { return r.charges }

// Event returns the MagicItemRecharged event.
func (r RechargeMagicItemApplied) Event() MagicItemRecharged { return r.event }

// RechargeMagicItemNotDue: the moment does not match the item's schedule, or it
// never recharges; the charges are unchanged.
type RechargeMagicItemNotDue struct{ charges ItemCharges }

// Charges returns the unchanged charges.
func (r RechargeMagicItemNotDue) Charges() ItemCharges { return r.charges }

func (RechargeMagicItemApplied) isRechargeMagicItemResult() {}
func (RechargeMagicItemNotDue) isRechargeMagicItemResult()  {}

// RechargeMagicItem is the use case that recharges an item when the host reports
// the moment ("the GM determines when the item recharges"). Create it with
// NewRechargeMagicItem; the zero value is not usable. It is safe for concurrent
// use only if its roller, telemetry and clock are.
type RechargeMagicItem struct {
	roller Roller
	sig    operationSignals
}

// NewRechargeMagicItem returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRechargeMagicItem(roller Roller, opts ...Option) (RechargeMagicItem, error) {
	if roller == nil {
		return RechargeMagicItem{}, invalidInput("recharge a magic item without a dice roller")
	}
	return RechargeMagicItem{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do recharges. A moment that does not match the schedule, or an item that does
// not recharge, is RechargeMagicItemNotDue and makes no request. A RegainsRolled
// recovery makes one request for its expression; the others none.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or an invalid command, and ctx.Err() unwrapped, before any
// signal, when ctx is already cancelled. It returns a *srderr.PortError (error
// type "dice_roller_error") with a nil result when the roller fails; the failure
// is recorded once and never logged.
func (u RechargeMagicItem) Do(ctx context.Context, cmd RechargeMagicItemCommand) (RechargeMagicItemResult, error) {
	if u.roller == nil || cmd.CharacterID.IsZero() || !cmd.Instance.Valid() || cmd.Charges.IsZero() || cmd.Recovery == nil ||
		cmd.Time < content.RechargesAtNextDawn || cmd.Time > content.DoesNotRecharge || cmd.Moment.String() == "unknown" {
		return nil, invalidInput("recharge magic item: zero use case, holder, instance or charges, or invalid schedule, recovery or moment")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRechargeMagicItem, ValueOperationRechargeMagicItem)
	defer run.end(ctx)
	run.span.SetAttributes(telemetry.String(AttrRechargeMoment, cmd.Moment.String()))
	if !(ChargeRecoveryRules{}).Due(cmd.Time, cmd.Moment) {
		run.finish(ctx, ValueResultNotDue, "")
		return RechargeMagicItemNotDue{charges: cmd.Charges}, nil
	}
	rolled := 0
	if r, ok := cmd.Recovery.(content.RegainsRolled); ok {
		res, err := dice.Roll(u.roller, r.Roll())
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		rolled = res.Total()
	}
	n, err := RegainedFor(cmd.Recovery, cmd.Charges, max(rolled, 0))
	if err != nil { // unreachable for a known recovery
		return nil, err
	}
	next := ItemCharges{current: cmd.Charges.current + n, maximum: cmd.Charges.maximum}
	run.finish(ctx, ValueResultChanged, "")
	return RechargeMagicItemApplied{charges: next, event: MagicItemRecharged{id: cmd.CharacterID, instance: cmd.Instance, regained: n}}, nil
}

// MaxMixedPotions is the most potions RollPotionMiscibility.Do accepts. The SRD
// sets no limit; this bounds the work of one call.
const MaxMixedPotions = 1000

// RollPotionMiscibilityCommand is the input of RollPotionMiscibility.Do: the
// number of potions mixed. Construct it with a keyed literal.
type RollPotionMiscibilityCommand struct{ Potions int }

// RollPotionMiscibilityResult is the closed set of outcomes of
// RollPotionMiscibility.Do. Variants: RollPotionMiscibilityRolled,
// RollPotionMiscibilityRejected.
type RollPotionMiscibilityResult interface{ isRollPotionMiscibilityResult() }

// RollPotionMiscibilityRolled: the effects of the mixture, one per roll.
type RollPotionMiscibilityRolled struct {
	effects []content.PotionMiscibilityEffect
}

// Effects returns a new slice of the effects in roll order.
func (r RollPotionMiscibilityRolled) Effects() []content.PotionMiscibilityEffect {
	return slices.Clone(r.effects)
}

// RollPotionMiscibilityRejected: fewer than two potions were mixed.
type RollPotionMiscibilityRejected struct{ reason PotionMixRejection }

// Reason returns PotionMixTooFewPotions.
func (r RollPotionMiscibilityRejected) Reason() PotionMixRejection { return r.reason }

// PotionMixRejection is why a mixture was refused. The zero value is not a reason.
type PotionMixRejection int

// PotionMixTooFewPotions: mixing needs at least two potions.
const PotionMixTooFewPotions PotionMixRejection = 1

func (RollPotionMiscibilityRolled) isRollPotionMiscibilityResult()   {}
func (RollPotionMiscibilityRejected) isRollPotionMiscibilityResult() {}

// RollPotionMiscibility is the use case that rolls what happens when potions are
// mixed (SRD 5.2.1, "Mixing Potions", p. 204). Create it with
// NewRollPotionMiscibility; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type RollPotionMiscibility struct {
	roller Roller
	sig    operationSignals
}

// NewRollPotionMiscibility returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollPotionMiscibility(roller Roller, opts ...Option) (RollPotionMiscibility, error) {
	if roller == nil {
		return RollPotionMiscibility{}, invalidInput("roll potion miscibility without a dice roller")
	}
	return RollPotionMiscibility{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do rolls 1d100 once for each potion after the first and maps each roll through
// the Potion Miscibility table. Fewer than two potions is
// RollPotionMiscibilityRejected and makes no request.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, for more than MaxMixedPotions potions, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with
// a nil result when the roller fails; the failure is recorded once and never logged.
func (u RollPotionMiscibility) Do(ctx context.Context, cmd RollPotionMiscibilityCommand) (RollPotionMiscibilityResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero RollPotionMiscibility; use NewRollPotionMiscibility")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cmd.Potions > MaxMixedPotions {
		return nil, invalidInput("mix more than MaxMixedPotions potions")
	}
	ctx, run := u.sig.start(ctx, SpanMixPotions, ValueOperationMixPotions)
	defer run.end(ctx)
	if cmd.Potions < 2 {
		run.finish(ctx, ValueResultRejected, ValueRejectionTooFewPotions)
		return RollPotionMiscibilityRejected{reason: PotionMixTooFewPotions}, nil
	}
	expr, err := dice.NewExpression(1, dice.D100(), 0)
	if err != nil { // unreachable: one d100
		return nil, err
	}
	effects := make([]content.PotionMiscibilityEffect, 0, cmd.Potions-1)
	for range cmd.Potions - 1 {
		res, err := dice.Roll(u.roller, expr)
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		effect, err := content.PotionMiscibilityTable{}.ForRoll(res.Total())
		if err != nil { // unreachable: dice.Roll vouches for 1 to 100
			return nil, err
		}
		effects = append(effects, effect)
	}
	run.finish(ctx, ValueResultRolled, "")
	return RollPotionMiscibilityRolled{effects: effects}, nil
}

// CheckCraftingMaterialsCommand is the input of CheckCraftingMaterials.Do: the
// settlement. Construct it with a keyed literal.
type CheckCraftingMaterialsCommand struct{ Settlement content.SettlementSize }

// CheckCraftingMaterialsResult is the closed set of outcomes of
// CheckCraftingMaterials.Do. Variants: CheckCraftingMaterialsAvailable,
// CheckCraftingMaterialsUnavailable.
type CheckCraftingMaterialsResult interface{ isCheckCraftingMaterialsResult() }

// CheckCraftingMaterialsAvailable: the raw materials were found.
type CheckCraftingMaterialsAvailable struct{}

// CheckCraftingMaterialsUnavailable: the crafter must wait at least
// MaterialsWaitDays before checking again.
type CheckCraftingMaterialsUnavailable struct{ waitAtLeastDays int }

// WaitAtLeastDays returns 7.
func (r CheckCraftingMaterialsUnavailable) WaitAtLeastDays() int { return r.waitAtLeastDays }

func (CheckCraftingMaterialsAvailable) isCheckCraftingMaterialsResult()   {}
func (CheckCraftingMaterialsUnavailable) isCheckCraftingMaterialsResult() {}

// CheckCraftingMaterials is the use case that rolls whether raw materials can be
// found in a settlement. Create it with NewCheckCraftingMaterials; the zero value
// is not usable. It is safe for concurrent use only if its roller, telemetry and
// clock are.
type CheckCraftingMaterials struct {
	roller Roller
	sig    operationSignals
}

// NewCheckCraftingMaterials returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewCheckCraftingMaterials(roller Roller, opts ...Option) (CheckCraftingMaterials, error) {
	if roller == nil {
		return CheckCraftingMaterials{}, invalidInput("check crafting materials without a dice roller")
	}
	return CheckCraftingMaterials{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do rolls 1d100 and compares it with the chance of the settlement.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or an unknown settlement, and ctx.Err() unwrapped, before any
// signal, when ctx is already cancelled. It returns a *srderr.PortError (error
// type "dice_roller_error") with a nil result when the roller fails; the failure
// is recorded once and never logged.
func (u CheckCraftingMaterials) Do(ctx context.Context, cmd CheckCraftingMaterialsCommand) (CheckCraftingMaterialsResult, error) {
	if u.roller == nil || !cmd.Settlement.Valid() {
		return nil, invalidInput("check crafting materials: zero use case or unknown settlement")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanCheckCraftingMaterials, ValueOperationCheckCraftingMaterials)
	defer run.end(ctx)
	expr, err := dice.NewExpression(1, dice.D100(), 0)
	if err != nil { // unreachable: one d100
		return nil, err
	}
	res, err := dice.Roll(u.roller, expr)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	ok, err := CraftingMaterials{}.IsAvailable(cmd.Settlement, res.Total())
	if err != nil { // unreachable: dice.Roll vouches for 1 to 100
		return nil, err
	}
	if !ok {
		run.finish(ctx, ValueResultUnavailable, "")
		return CheckCraftingMaterialsUnavailable{waitAtLeastDays: MaterialsWaitDays}, nil
	}
	run.finish(ctx, ValueResultChanged, "")
	return CheckCraftingMaterialsAvailable{}, nil
}
