package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// RollObjectHitPointsCommand is the input of RollObjectHitPoints.Run: the object's size
// and durability. Build it with a keyed literal; the zero value is invalid.
type RollObjectHitPointsCommand struct {
	Size       content.CreatureSize
	Durability ObjectDurability
}

// RollObjectHitPointsResult is the closed set of outcomes of RollObjectHitPoints.Run.
// Variants: ObjectHitPointsRolled, ObjectHitPointsRollSplitRequired.
type RollObjectHitPointsResult interface{ isRollObjectHitPointsResult() }

// ObjectHitPointsRolled: the object's Hit Points were rolled; current and maximum are the
// total and it has no Temporary Hit Points.
type ObjectHitPointsRolled struct {
	entry     ObjectHitPointsEntry
	roll      dice.Result
	hitPoints HitPoints
}

// Entry returns the table row that was rolled.
func (r ObjectHitPointsRolled) Entry() ObjectHitPointsEntry { return r.entry }

// Roll returns the roll of the entry's expression.
func (r ObjectHitPointsRolled) Roll() dice.Result { return r.roll }

// HitPoints returns the object's Hit Points.
func (r ObjectHitPointsRolled) HitPoints() HitPoints { return r.hitPoints }

// ObjectHitPointsRollSplitRequired: the object is Huge or Gargantuan, so nothing was
// rolled: divide it into Large or smaller sections. Size is the size asked for.
// Construct it with a keyed literal.
type ObjectHitPointsRollSplitRequired struct{ Size content.CreatureSize }

func (ObjectHitPointsRolled) isRollObjectHitPointsResult()            {}
func (ObjectHitPointsRollSplitRequired) isRollObjectHitPointsResult() {}

// RollObjectHitPoints is the use case that rolls an object's Hit Points instead of
// taking the table's average (SRD 5.2.1, "Breaking Objects", Object Hit Points, p. 178).
// It uses the dice port. Create it with NewRollObjectHitPoints; the zero value is not
// usable.
type RollObjectHitPoints struct {
	environmentUse
	roller Roller
}

// NewRollObjectHitPoints returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a
// nil roller.
func NewRollObjectHitPoints(roller Roller, opts ...Option) (RollObjectHitPoints, error) {
	if roller == nil {
		return RollObjectHitPoints{}, invalidInput("object hit points without a dice roller")
	}
	return RollObjectHitPoints{newEnvironmentUse(opts), roller}, nil
}

// Run makes one request of the entry's expression and records the shared signals.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. A size or
// durability that is not valid is an error matching srderr.ErrInvalidInput that the span
// records; no operations are counted. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers
// another expression; the failure is recorded once and never logged. A Huge or
// Gargantuan object is a split-required result, with no roll.
func (u RollObjectHitPoints) Run(ctx context.Context, cmd RollObjectHitPointsCommand) (RollObjectHitPointsResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero use case")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationRollObjectHitPoints, spanRollObjectHitPoints},
		func(context.Context) (RollObjectHitPointsResult, error) {
			lookup, err := ObjectHitPointsTable{}.Lookup(cmd.Size, cmd.Durability)
			if err != nil {
				return nil, err
			}
			found, ok := lookup.(ObjectHitPointsFound)
			if !ok {
				return ObjectHitPointsRollSplitRequired{cmd.Size}, nil
			}
			roll, err := dice.Roll(u.roller, found.Entry.expression)
			if err != nil {
				return nil, err
			}
			return ObjectHitPointsRolled{found.Entry, roll, HitPoints{roll.Total(), roll.Total(), 0}}, nil
		},
		func(r RollObjectHitPointsResult) description {
			if _, ok := r.(ObjectHitPointsRollSplitRequired); ok {
				return description{Class: resultRefused, Name: valueResultSplitRequired}
			}
			return description{Class: resultApplied, Name: valueResultRolled}
		})
}
