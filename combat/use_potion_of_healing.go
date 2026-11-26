package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// potionAdministerRangeFeet is how far from the user a patient can be and still be given a
// Potion of Healing: within 5 feet (SRD 5.2.1, "Potion of Healing", p. 99).
const potionAdministerRangeFeet = 5

// PotionUse says who the Potion of Healing is for. Variants: DrinkPotion, AdministerPotion.
type PotionUse interface{ isPotionUse() }

// DrinkPotion: the user is the patient.
type DrinkPotion struct{}

// AdministerPotion: the user gives the potion to another creature DistanceFeet away (0 or
// more; 5 is within reach). Construct with a keyed literal.
type AdministerPotion struct{ DistanceFeet int }

func (DrinkPotion) isPotionUse()      {}
func (AdministerPotion) isPotionUse() {}

// UsePotionOfHealingCommand is the immutable input of UsePotionOfHealing.Run. Build it with
// NewUsePotionOfHealingCommand; the zero value is invalid.
type UsePotionOfHealingCommand struct {
	user, patient CombatantID
	status        HitPointStatus
	use           PotionUse
}

// NewUsePotionOfHealingCommand returns the command: user gives or drinks the potion, patient
// is the creature healed, status is the patient's. Whether the user can act is the host's
// input and is not checked here.
//
// It returns an *ItemUseError (matching srderr.ErrInvalidInput) with the violated field for a
// zero user, patient or status, a use that is neither DrinkPotion nor AdministerPotion, a
// negative distance (ItemUseFieldDistanceFeet), or DrinkPotion with a user other than the
// patient (ItemUseFieldPatient). It never panics.
func NewUsePotionOfHealingCommand(user, patient CombatantID, status HitPointStatus, use PotionUse) (UsePotionOfHealingCommand, error) {
	switch u := use.(type) {
	case DrinkPotion:
		if user != patient {
			return UsePotionOfHealingCommand{}, itemUseInvalid(ItemUseFieldPatient, 0)
		}
	case AdministerPotion:
		if u.DistanceFeet < 0 {
			return UsePotionOfHealingCommand{}, itemUseInvalid(ItemUseFieldDistanceFeet, int64(u.DistanceFeet))
		}
	default:
		return UsePotionOfHealingCommand{}, itemUseInvalid(ItemUseFieldPotionUse, 0)
	}
	switch {
	case user.IsZero():
		return UsePotionOfHealingCommand{}, itemUseInvalid(ItemUseFieldUser, 0)
	case patient.IsZero():
		return UsePotionOfHealingCommand{}, itemUseInvalid(ItemUseFieldPatient, 0)
	case status.IsZero():
		return UsePotionOfHealingCommand{}, itemUseInvalid(ItemUseFieldStatus, 0)
	}
	return UsePotionOfHealingCommand{user, patient, status, use}, nil
}

// MustUsePotionOfHealingCommand is NewUsePotionOfHealingCommand for values known to be valid.
// It panics with the error NewUsePotionOfHealingCommand returns, and only then.
func MustUsePotionOfHealingCommand(user, patient CombatantID, status HitPointStatus, use PotionUse) UsePotionOfHealingCommand {
	c, err := NewUsePotionOfHealingCommand(user, patient, status, use)
	if err != nil {
		panic(err)
	}
	return c
}

// PotionNotPossibleReason is why a Potion of Healing cannot be used. The zero value is no
// reason. String returns the snake_case reason, the value of srd5e.combat.result.
type PotionNotPossibleReason int

// The reasons, in the order they are checked.
const (
	// PotionReasonNotWithinFiveFeet: the patient is more than 5 feet from the user.
	PotionReasonNotWithinFiveFeet PotionNotPossibleReason = iota + 1
	// PotionReasonPatientIsDead: the patient is Dead.
	PotionReasonPatientIsDead
)

// String returns "not_within_five_feet", "patient_is_dead", "none" for the zero value or "unknown".
func (r PotionNotPossibleReason) String() string {
	if r == 0 {
		return "none"
	}
	return enumName([]string{"not_within_five_feet", "patient_is_dead"}, int(r))
}

// UsePotionOfHealingResult is the closed set of outcomes of UsePotionOfHealing.Run. Variants:
// PotionOfHealingUsed, PotionOfHealingNotPossible.
type UsePotionOfHealingResult interface{ isUsePotionOfHealingResult() }

// PotionOfHealingUsed: the potion was used and the patient regained Hit Points (possibly 0).
// Not comparable.
type PotionOfHealingUsed struct {
	status   HitPointStatus
	events   []Event
	roll     dice.Result
	regained int
}

// Status returns the patient's new status.
func (r PotionOfHealingUsed) Status() HitPointStatus { return r.status }

// Amount returns the healing rolled, 4 to 10, unclamped by the Hit Point maximum.
func (r PotionOfHealingUsed) Amount() int { return r.roll.Total() }

// HitPointsRegained returns the Hit Points actually regained, 0 for a patient at the maximum.
func (r PotionOfHealingUsed) HitPointsRegained() int { return r.regained }

// Roll returns the dice rolled.
func (r PotionOfHealingUsed) Roll() dice.Result { return r.roll }

// Consumption returns the one potion used up, whatever the Hit Points regained.
func (PotionOfHealingUsed) Consumption() ItemConsumption { return consumedOne(ItemUsePotionOfHealing) }

// Cost returns ItemActionCostBonusAction.
func (PotionOfHealingUsed) Cost() ItemActionCost {
	return ItemUseRules{}.CostOf(ItemUsePotionOfHealing)
}

// Events returns a copy of the events in order: the hit point events of the healing
// (HitPointsRestored, and Recovered for a patient at 0 Hit Points), then one
// PotionOfHealingConsumed.
func (r PotionOfHealingUsed) Events() []Event { return slices.Clone(r.events) }

// PotionOfHealingNotPossible: the potion cannot be used; no die was rolled and the potion is
// not consumed. Construct with a keyed literal.
type PotionOfHealingNotPossible struct{ Reason PotionNotPossibleReason }

func (PotionOfHealingUsed) isUsePotionOfHealingResult()        {}
func (PotionOfHealingNotPossible) isUsePotionOfHealingResult() {}

// UsePotionOfHealing is the use case of drinking or administering a Potion of Healing (SRD
// 5.2.1, "Potion of Healing", p. 99). It composes HitPointStatus.Restore and never calls
// another use case. Create it with NewUsePotionOfHealing; the zero value is not usable.
type UsePotionOfHealing struct {
	in     instrumenter
	roller Roller
}

// NewUsePotionOfHealing returns the use case rolling through roller. Options: WithTelemetry,
// WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewUsePotionOfHealing(roller Roller, opts ...Option) (UsePotionOfHealing, error) {
	if roller == nil {
		return UsePotionOfHealing{}, invalidInput("potion without a dice roller")
	}
	o := newOptions(opts)
	return UsePotionOfHealing{newInstrumenter(o.tel, o.clock), roller}, nil
}

// Run refuses a patient more than 5 feet away (not_within_five_feet), then a Dead patient
// (patient_is_dead), each before any die is rolled. Otherwise it rolls the standard Potion
// of Healing expression (character.GradeStandard, 2d4+2) once and restores that many Hit
// Points through HitPointStatus.Restore. A patient at 0 Hit Points becomes Alive with its
// tallies reset; an Unconscious one may be given the potion; one at the Hit Point maximum
// regains 0 and the potion is still consumed.
//
// It records the shared signals with the attribute srd5e.combat.item_use.item. It returns an
// error matching srderr.ErrInvalidInput, before any signal, for the zero use case or command,
// and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails,
// panics or answers another expression; the failure is recorded once and never logged.
func (u UsePotionOfHealing) Run(ctx context.Context, cmd UsePotionOfHealingCommand) (UsePotionOfHealingResult, error) {
	if cmd.status.IsZero() || u.roller == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	item := telemetry.String(attrItemUseItem, valueItemUsePotionOfHealing)
	return run(ctx, u.in, combatOperation{valueOperationUsePotionOfHealing, spanUsePotionOfHealing},
		func(context.Context) (UsePotionOfHealingResult, error) { return u.use(cmd) },
		func(r UsePotionOfHealingResult) description {
			switch v := r.(type) {
			case PotionOfHealingUsed:
				return description{Class: resultApplied, Name: valueResultHealed, Events: slices.Clone(v.events), Extras: []telemetry.Attr{item}}
			case PotionOfHealingNotPossible:
				return description{Class: resultRefused, Name: v.Reason.String(), Extras: []telemetry.Attr{item}}
			default:
				panic(unreachableError("use potion of healing result", r))
			}
		})
}

// use is the work of Run for a valid command.
func (u UsePotionOfHealing) use(cmd UsePotionOfHealingCommand) (UsePotionOfHealingResult, error) {
	if a, ok := cmd.use.(AdministerPotion); ok && a.DistanceFeet > potionAdministerRangeFeet {
		return PotionOfHealingNotPossible{PotionReasonNotWithinFiveFeet}, nil
	}
	if cmd.status.isDead() {
		return PotionOfHealingNotPossible{PotionReasonPatientIsDead}, nil
	}
	roll, err := dice.Roll(u.roller, character.GradeStandard.Dice())
	if err != nil {
		return nil, err
	}
	tr, err := cmd.status.Restore(cmd.patient, roll.Total(), NoHealingCeiling{})
	if err != nil {
		return nil, err
	}
	switch a := tr.(type) {
	case HitPointTransitionApplied:
		events := append(asEvents(a.events), PotionOfHealingConsumed{itemUseEventBase{cmd.user, cmd.patient}})
		return PotionOfHealingUsed{a.status, events, roll, a.regained}, nil
	default:
		// A living patient is never refused Healing: the Dead were turned away above.
		panic(unreachableError("healing of a living patient", tr))
	}
}
