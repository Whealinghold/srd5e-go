package combat

import "context"

// RemoveExhaustionLevelsCommand is the input of RemoveExhaustionLevels.Run: the
// creature's conditions and id, its hazard record and the number of Exhaustion levels
// to remove. Build it with a keyed literal; the zero value is invalid.
type RemoveExhaustionLevelsCommand struct {
	Conditions ConditionSet
	Target     CombatantID
	Record     HazardRecord
	Levels     int
}

// RemoveExhaustionLevelsResult is the closed set of outcomes of
// RemoveExhaustionLevels.Run. Variants: ExhaustionLevelsRemoved,
// RemoveExhaustionLevelsRejected.
type RemoveExhaustionLevelsResult interface{ isRemoveExhaustionLevelsResult() }

// ExhaustionLevelsRemoved: levels came off, up to those the record does not lock. Some
// of the levels asked for may be Blocked.
type ExhaustionLevelsRemoved struct {
	conditions ConditionSet
	record     HazardRecord
	reduction  ReduceExhaustionOnSetResult
	removed    int
	blocked    int
	events     []Event
}

// Conditions returns the creature's conditions after the call.
func (r ExhaustionLevelsRemoved) Conditions() ConditionSet { return r.conditions }

// Record returns the hazard record after the call.
func (r ExhaustionLevelsRemoved) Record() HazardRecord { return r.record }

// Reduction returns the result of ConditionSet.ReduceExhaustion.
func (r ExhaustionLevelsRemoved) Reduction() ReduceExhaustionOnSetResult { return r.reduction }

// Removed returns the levels removed.
func (r ExhaustionLevelsRemoved) Removed() int { return r.removed }

// Blocked returns the levels asked for that a lock held back.
func (r ExhaustionLevelsRemoved) Blocked() int { return r.blocked }

// Events returns a copy of the condition events, in the order they happened.
func (r ExhaustionLevelsRemoved) Events() []Event { return copyEvents(r.events) }

// RemoveExhaustionLevelsRejected: no level could be removed, because every level is
// locked or the creature has no Exhaustion; nothing changed. Construct it with a keyed
// literal.
type RemoveExhaustionLevelsRejected struct{ Rejection ExhaustionRemovalRejection }

func (ExhaustionLevelsRemoved) isRemoveExhaustionLevelsResult()        {}
func (RemoveExhaustionLevelsRejected) isRemoveExhaustionLevelsResult() {}

// RemoveExhaustionLevels is the use case that removes Exhaustion levels without
// breaking the lock: levels Dehydration, Malnutrition or a contagion hold cannot be
// removed until their cause is gone (SRD 5.2.1, "Exhaustion", "Dehydration [Hazard]",
// p. 181; "Malnutrition [Hazard]", p. 185). A host calls it instead of
// ReduceExhaustionOnSet, for a Long Rest as well. It rolls no die. Create it with
// NewRemoveExhaustionLevels; the zero value is silent and works. Safe for concurrent
// use.
type RemoveExhaustionLevels struct{ environmentUse }

// NewRemoveExhaustionLevels returns the use case. Options: WithTelemetry, WithClock.
func NewRemoveExhaustionLevels(opts ...Option) RemoveExhaustionLevels {
	return RemoveExhaustionLevels{newEnvironmentUse(opts)}
}

// Run removes the levels and records the shared signals. The Suffocation levels of the
// record are cut to the Exhaustion that remains.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. Levels below
// 1 give an error matching srderr.ErrInvalidInput that the span records; no operations
// or events are counted. A creature whose levels are all locked, or without Exhaustion,
// is a RemoveExhaustionLevelsRejected result.
func (u RemoveExhaustionLevels) Run(ctx context.Context, cmd RemoveExhaustionLevelsCommand) (RemoveExhaustionLevelsResult, error) {
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationRemoveExhaustionLevels, spanRemoveExhaustionLevels},
		func(context.Context) (RemoveExhaustionLevelsResult, error) {
			r, err := removeExhaustion(cmd.Conditions, cmd.Target, cmd.Record, cmd.Levels)
			switch {
			case err != nil:
				return nil, err
			case r.rejection != ExhaustionRemovalRejection{}:
				return RemoveExhaustionLevelsRejected{r.rejection}, nil
			}
			return ExhaustionLevelsRemoved{r.set, r.record, r.reduction, r.removed, r.blocked, r.events}, nil
		},
		func(r RemoveExhaustionLevelsResult) description {
			if _, ok := r.(RemoveExhaustionLevelsRejected); ok {
				return description{Class: resultRefused, Name: valueResultBlocked}
			}
			return description{Class: resultApplied, Name: valueResultRemoved, Events: r.(ExhaustionLevelsRemoved).events}
		})
}
