package combat

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// ResolveDehydrationDayEndCommand is the input of ResolveDehydrationDayEnd.Run: the
// creature's size, the water it drank that day, its conditions and id, and its hazard
// record. Build it with a keyed literal; the zero value is invalid.
type ResolveDehydrationDayEndCommand struct {
	Size       content.CreatureSize
	Water      WaterAmount
	Conditions ConditionSet
	Target     CombatantID
	Record     HazardRecord
}

// ResolveDehydrationDayEndResult is the closed set of outcomes of
// ResolveDehydrationDayEnd.Run. Variants: DehydrationDayEnded.
type ResolveDehydrationDayEndResult interface{ isResolveDehydrationDayEndResult() }

// DehydrationDayEnded is the creature at the end of the day: its conditions, its hazard
// record, what happened to its Exhaustion and the events (the condition events, then
// DehydrationRecovered when the full amount unlocked levels). At Exhaustion 6 the host
// calls IsDeadFromExhaustion.
type DehydrationDayEnded struct {
	conditions ConditionSet
	record     HazardRecord
	outcome    HazardExhaustionOutcome
	events     []Event
}

// Conditions returns the creature's conditions after the call.
func (r DehydrationDayEnded) Conditions() ConditionSet { return r.conditions }

// Record returns the hazard record after the call.
func (r DehydrationDayEnded) Record() HazardRecord { return r.record }

// Outcome returns what happened to the creature's Exhaustion.
func (r DehydrationDayEnded) Outcome() HazardExhaustionOutcome { return r.outcome }

// Events returns a copy of the events, in the order they happened.
func (r DehydrationDayEnded) Events() []Event { return copyEvents(r.events) }

func (DehydrationDayEnded) isResolveDehydrationDayEndResult() {}

// ResolveDehydrationDayEnd is the use case the host calls at the end of each day: a
// creature that drank less than half its daily water gains one Exhaustion level, and
// drinking the full amount unlocks the levels Dehydration holds (SRD 5.2.1,
// "Dehydration [Hazard]", p. 181). It rolls no die. Create it with
// NewResolveDehydrationDayEnd; the zero value is silent and works. Safe for concurrent
// use.
type ResolveDehydrationDayEnd struct{ environmentUse }

// NewResolveDehydrationDayEnd returns the use case. Options: WithTelemetry, WithClock.
func NewResolveDehydrationDayEnd(opts ...Option) ResolveDehydrationDayEnd {
	return ResolveDehydrationDayEnd{newEnvironmentUse(opts)}
}

// Run resolves the day and records the shared signals plus
// srd5e.combat.hazard.exhaustion_gained (dehydration) when a level was gained. The
// record is clamped so its levels do not exceed the Exhaustion the set holds; a record
// that disagrees with the set never panics.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target or a size that is not valid, and ctx.Err() unwrapped, before any signal, for a
// cancelled ctx.
func (u ResolveDehydrationDayEnd) Run(ctx context.Context, cmd ResolveDehydrationDayEndCommand) (ResolveDehydrationDayEndResult, error) {
	if !cmd.Size.Valid() {
		return nil, invalidInput("invalid creature size")
	}
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationResolveDehydrationDayEnd, spanResolveDehydrationDayEnd},
		func(context.Context) (DehydrationDayEnded, error) {
			d := dehydrationDayEnd(needsEighths(cmd.Size), cmd.Water.eighths, cmd.Conditions, cmd.Target, cmd.Record)
			return DehydrationDayEnded{d.set, d.record, d.outcome, d.events}, nil
		},
		func(r DehydrationDayEnded) description { return exhaustionDescription(r.outcome, r.events) })
	if err != nil {
		return nil, err
	}
	u.recordExhaustionGained(ctx, HazardDehydration, res.outcome)
	return res, nil
}
