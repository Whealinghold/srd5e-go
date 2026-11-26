package combat

import "context"

// ResolveSuffocationTurnEndCommand is the input of ResolveSuffocationTurnEnd.Run: the
// creature's breathing, its conditions and id, and its hazard record. Build it with a
// keyed literal; the zero value is invalid.
type ResolveSuffocationTurnEndCommand struct {
	State      BreathState
	Conditions ConditionSet
	Target     CombatantID
	Record     HazardRecord
}

// ResolveSuffocationTurnEndResult is the closed set of outcomes of
// ResolveSuffocationTurnEnd.Run. Variants: SuffocationTurnEnded.
type ResolveSuffocationTurnEndResult interface{ isResolveSuffocationTurnEndResult() }

// SuffocationTurnEnded is the creature at the end of its turn: its conditions, its
// hazard record, what happened to its Exhaustion and the condition events. At
// Exhaustion 6 the host calls IsDeadFromExhaustion.
type SuffocationTurnEnded struct {
	conditions ConditionSet
	record     HazardRecord
	outcome    HazardExhaustionOutcome
	events     []Event
}

// Conditions returns the creature's conditions after the call.
func (r SuffocationTurnEnded) Conditions() ConditionSet { return r.conditions }

// Record returns the hazard record after the call.
func (r SuffocationTurnEnded) Record() HazardRecord { return r.record }

// Outcome returns what happened to the creature's Exhaustion.
func (r SuffocationTurnEnded) Outcome() HazardExhaustionOutcome { return r.outcome }

// Events returns a copy of the events, in the order they happened.
func (r SuffocationTurnEnded) Events() []Event { return copyEvents(r.events) }

func (SuffocationTurnEnded) isResolveSuffocationTurnEndResult() {}

// ResolveSuffocationTurnEnd is the use case the host calls at the end of each of a
// creature's turns: a suffocating creature gains one Exhaustion level (SRD 5.2.1,
// "Suffocation [Hazard]", p. 189). It rolls no die. Create it with
// NewResolveSuffocationTurnEnd; the zero value is silent and works. Safe for concurrent
// use.
type ResolveSuffocationTurnEnd struct{ environmentUse }

// NewResolveSuffocationTurnEnd returns the use case. Options: WithTelemetry, WithClock.
func NewResolveSuffocationTurnEnd(opts ...Option) ResolveSuffocationTurnEnd {
	return ResolveSuffocationTurnEnd{newEnvironmentUse(opts)}
}

// Run resolves the turn end and records the shared signals plus
// srd5e.combat.hazard.exhaustion_gained (suffocation) when a level was gained. A
// creature that is breathing or holding its breath gains nothing. The record is
// clamped so its levels do not exceed the Exhaustion the set holds.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target or a State that is not valid, and ctx.Err() unwrapped, before any signal, for
// a cancelled ctx.
func (u ResolveSuffocationTurnEnd) Run(ctx context.Context, cmd ResolveSuffocationTurnEndCommand) (ResolveSuffocationTurnEndResult, error) {
	if !validBreathState(cmd.State) {
		return nil, invalidInput("invalid breath state")
	}
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationResolveSuffocationTurnEnd, spanResolveSuffocationTurnEnd},
		func(context.Context) (SuffocationTurnEnded, error) {
			d := suffocationTurnEnd(cmd.State, cmd.Conditions, cmd.Target, cmd.Record)
			return SuffocationTurnEnded{d.set, d.record, d.outcome, d.events}, nil
		},
		func(r SuffocationTurnEnded) description { return exhaustionDescription(r.outcome, r.events) })
	if err != nil {
		return nil, err
	}
	u.recordExhaustionGained(ctx, HazardSuffocation, res.outcome)
	return res, nil
}
