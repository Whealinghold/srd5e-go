package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollPotionOfHealingCommand is the input of RollPotionOfHealing.Do: the rarity
// of the potion. Construct it with a keyed literal.
type RollPotionOfHealingCommand struct{ Rarity content.ItemRarity }

// RollPotionOfHealingResult is the closed set of outcomes of
// RollPotionOfHealing.Do. Variants: RollPotionOfHealingRolled,
// RollPotionOfHealingNotListed, RollPotionOfHealingStandardRolledElsewhere.
type RollPotionOfHealingResult interface{ isRollPotionOfHealingResult() }

// RollPotionOfHealingRolled: the Hit Points the potion restores. The amount is
// not clamped to a Hit Point maximum; applying it is the host's.
type RollPotionOfHealingRolled struct {
	grade  PotionOfHealingGrade
	amount int
}

// Grade returns the grade.
func (r RollPotionOfHealingRolled) Grade() PotionOfHealingGrade { return r.grade }

// Amount returns the total of the roll.
func (r RollPotionOfHealingRolled) Amount() int { return r.amount }

// RollPotionOfHealingNotListed: no grade has the rarity; nothing was rolled.
type RollPotionOfHealingNotListed struct{ rarity content.ItemRarity }

// Rarity returns the rarity asked for.
func (r RollPotionOfHealingNotListed) Rarity() content.ItemRarity { return r.rarity }

// RollPotionOfHealingStandardRolledElsewhere: the Common potion's roll belongs to
// the item-use rules; nothing was rolled here.
type RollPotionOfHealingStandardRolledElsewhere struct{ grade PotionOfHealingGrade }

// Grade returns GradeStandard.
func (r RollPotionOfHealingStandardRolledElsewhere) Grade() PotionOfHealingGrade { return r.grade }

func (RollPotionOfHealingRolled) isRollPotionOfHealingResult()                  {}
func (RollPotionOfHealingNotListed) isRollPotionOfHealingResult()               {}
func (RollPotionOfHealingStandardRolledElsewhere) isRollPotionOfHealingResult() {}

// RollPotionOfHealing is the use case that rolls a Potion of Healing of the
// greater, superior or supreme grade (SRD 5.2.1, "Potions of Healing", p. 236).
// Create it with NewRollPotionOfHealing; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type RollPotionOfHealing struct {
	roller Roller
	sig    operationSignals
}

// NewRollPotionOfHealing returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollPotionOfHealing(roller Roller, opts ...Option) (RollPotionOfHealing, error) {
	if roller == nil {
		return RollPotionOfHealing{}, invalidInput("roll a potion of healing without a dice roller")
	}
	return RollPotionOfHealing{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do makes one request for the grade's expression, fixed modifier included. A
// rarity with no grade is RollPotionOfHealingNotListed and the Common grade is
// RollPotionOfHealingStandardRolledElsewhere; neither makes a request.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or an invalid rarity, and ctx.Err() unwrapped, before any signal,
// when ctx is already cancelled. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails or answers another
// expression; the failure is recorded once and never logged.
func (u RollPotionOfHealing) Do(ctx context.Context, cmd RollPotionOfHealingCommand) (RollPotionOfHealingResult, error) {
	if u.roller == nil || !cmd.Rarity.Valid() {
		return nil, invalidInput("roll potion of healing: zero use case or invalid rarity")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollPotionOfHealing, ValueOperationRollPotionOfHealing)
	defer run.end(ctx)
	found, ok := (PotionOfHealingGrades{}).ForRarity(cmd.Rarity).(PotionOfHealingGradeFound)
	if !ok {
		run.finish(ctx, ValueResultNotListed, "")
		return RollPotionOfHealingNotListed{rarity: cmd.Rarity}, nil
	}
	if found.grade == GradeStandard {
		run.finish(ctx, ValueResultRolledElsewhere, "")
		return RollPotionOfHealingStandardRolledElsewhere(found), nil
	}
	res, err := dice.Roll(u.roller, found.grade.Dice())
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	run.finish(ctx, ValueResultRolled, "")
	return RollPotionOfHealingRolled{grade: found.grade, amount: res.Total()}, nil
}

// RollSpellStoringRingLevelsCommand is the input of RollSpellStoringRingLevels.Do:
// none.
type RollSpellStoringRingLevelsCommand struct{}

// RollSpellStoringRingLevelsResult is the closed set of outcomes of
// RollSpellStoringRingLevels.Do. Variants: RollSpellStoringRingLevelsRolled.
type RollSpellStoringRingLevelsResult interface{ isRollSpellStoringRingLevelsResult() }

// RollSpellStoringRingLevelsRolled: the levels of spells a new ring holds, 0 to 5.
type RollSpellStoringRingLevelsRolled struct{ levels int }

// Levels returns the levels, the 1d6 roll minus 1.
func (r RollSpellStoringRingLevelsRolled) Levels() int { return r.levels }

func (RollSpellStoringRingLevelsRolled) isRollSpellStoringRingLevelsResult() {}

// RollSpellStoringRingLevels is the use case that rolls the levels of spells a
// found Ring of Spell Storing holds (SRD 5.2.1, "Ring of Spell Storing", p. 239).
// Create it with NewRollSpellStoringRingLevels; the zero value is not usable. It
// is safe for concurrent use only if its roller, telemetry and clock are.
type RollSpellStoringRingLevels struct {
	roller Roller
	sig    operationSignals
}

// NewRollSpellStoringRingLevels returns the use case rolling through roller.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil roller.
func NewRollSpellStoringRingLevels(roller Roller, opts ...Option) (RollSpellStoringRingLevels, error) {
	if roller == nil {
		return RollSpellStoringRingLevels{}, invalidInput("roll spell storing ring levels without a dice roller")
	}
	return RollSpellStoringRingLevels{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Do makes one request of 1d6 and returns the roll minus 1.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "dice_roller_error") with
// a nil result when the roller fails or answers another expression; the failure is
// recorded once and never logged.
func (u RollSpellStoringRingLevels) Do(ctx context.Context, _ RollSpellStoringRingLevelsCommand) (RollSpellStoringRingLevelsResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero RollSpellStoringRingLevels; use NewRollSpellStoringRingLevels")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollSpellStoringLevels, ValueOperationRollSpellStoringLevels)
	defer run.end(ctx)
	res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D6(), 0))
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	run.finish(ctx, ValueResultRolled, "")
	return RollSpellStoringRingLevelsRolled{levels: res.Total() - 1}, nil
}
