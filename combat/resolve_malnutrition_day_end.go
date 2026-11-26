package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ResolveMalnutritionDayEndCommand is the input of ResolveMalnutritionDayEnd.Run: the
// creature's size, the food it ate that day, the Constitution saving throw (rolled by
// the host once Run asks for it), its conditions and id, and its hazard record. Build
// it with a keyed literal; the zero value is invalid.
type ResolveMalnutritionDayEndCommand struct {
	Size        content.CreatureSize
	Food        FoodAmount
	SavingThrow FoodSavingThrow
	Conditions  ConditionSet
	Target      CombatantID
	Record      HazardRecord
}

// ResolveMalnutritionDayEndResult is the closed set of outcomes of
// ResolveMalnutritionDayEnd.Run. Variants: MalnutritionDayEnded,
// MalnutritionSaveRequired.
type ResolveMalnutritionDayEndResult interface{ isResolveMalnutritionDayEndResult() }

// MalnutritionDayEnded is the creature at the end of the day: its conditions, its
// hazard record, what happened to its Exhaustion and the events (the condition events,
// then MalnutritionRecovered when the full amount unlocked levels). At Exhaustion 6
// the host calls IsDeadFromExhaustion.
type MalnutritionDayEnded struct {
	conditions ConditionSet
	record     HazardRecord
	outcome    HazardExhaustionOutcome
	events     []Event
}

// Conditions returns the creature's conditions after the call.
func (r MalnutritionDayEnded) Conditions() ConditionSet { return r.conditions }

// Record returns the hazard record after the call.
func (r MalnutritionDayEnded) Record() HazardRecord { return r.record }

// Outcome returns what happened to the creature's Exhaustion.
func (r MalnutritionDayEnded) Outcome() HazardExhaustionOutcome { return r.outcome }

// Events returns a copy of the events, in the order they happened.
func (r MalnutritionDayEnded) Events() []Event { return copyEvents(r.events) }

// MalnutritionSaveRequired: the creature ate some food but less than half, so it needs
// a saving throw; nothing was changed. The host rolls it with d20 and calls again with
// the outcome as a FoodSaveRolled.
type MalnutritionSaveRequired struct {
	dc      d20.DifficultyClass
	ability ability.Ability
}

// DC returns the Difficulty Class of the saving throw, 10.
func (r MalnutritionSaveRequired) DC() d20.DifficultyClass { return r.dc }

// Ability returns the ability of the saving throw, Constitution.
func (r MalnutritionSaveRequired) Ability() ability.Ability { return r.ability }

func (MalnutritionDayEnded) isResolveMalnutritionDayEndResult()     {}
func (MalnutritionSaveRequired) isResolveMalnutritionDayEndResult() {}

// ResolveMalnutritionDayEnd is the use case the host calls at the end of each day: a
// creature that ate less than half its daily food makes a Constitution saving throw or
// gains an Exhaustion level, one that ate nothing for 5 days gains one without a save,
// and eating the full amount unlocks the levels Malnutrition holds (SRD 5.2.1,
// "Malnutrition [Hazard]", p. 185). It rolls no die. Create it with
// NewResolveMalnutritionDayEnd; the zero value is silent and works. Safe for concurrent
// use.
type ResolveMalnutritionDayEnd struct{ environmentUse }

// NewResolveMalnutritionDayEnd returns the use case. Options: WithTelemetry, WithClock.
func NewResolveMalnutritionDayEnd(opts ...Option) ResolveMalnutritionDayEnd {
	return ResolveMalnutritionDayEnd{newEnvironmentUse(opts)}
}

// Run resolves the day and records the shared signals plus
// srd5e.combat.hazard.exhaustion_gained (malnutrition) when a level was gained. A
// FoodSaveRolled is ignored when no save is required. The record is clamped so its
// levels do not exceed the Exhaustion the set holds; a record that disagrees with the
// set never panics.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target, a size that is not valid or a SavingThrow that is not a known variant, and
// ctx.Err() unwrapped, before any signal, for a cancelled ctx.
func (u ResolveMalnutritionDayEnd) Run(ctx context.Context, cmd ResolveMalnutritionDayEndCommand) (ResolveMalnutritionDayEndResult, error) {
	switch cmd.SavingThrow.(type) {
	case nil, FoodSaveNotRolled, FoodSaveRolled:
	default:
		return nil, invalidInput("food saving throw is not a known variant")
	}
	if !cmd.Size.Valid() {
		return nil, invalidInput("invalid creature size")
	}
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationResolveMalnutritionDayEnd, spanResolveMalnutritionDayEnd},
		func(context.Context) (ResolveMalnutritionDayEndResult, error) {
			d, needsSave := malnutritionDayEnd(needsEighths(cmd.Size), cmd.Food.eighths, cmd.SavingThrow, cmd.Conditions, cmd.Target, cmd.Record)
			if needsSave {
				dc, ab := malnutritionSave()
				return MalnutritionSaveRequired{dc, ab}, nil
			}
			return MalnutritionDayEnded{d.set, d.record, d.outcome, d.events}, nil
		},
		func(r ResolveMalnutritionDayEndResult) description {
			if v, ok := r.(MalnutritionDayEnded); ok {
				return exhaustionDescription(v.outcome, v.events)
			}
			return description{Class: resultApplied, Name: valueResultSaveRequired}
		})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(MalnutritionDayEnded); ok {
		u.recordExhaustionGained(ctx, HazardMalnutrition, v.outcome)
	}
	return res, nil
}
