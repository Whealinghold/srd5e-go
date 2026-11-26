package combat

import "context"

// RestoreBreathingCommand is the input of RestoreBreathing.Run: the creature's
// breathing, its conditions and id, and its hazard record. Build it with a keyed
// literal; the zero value is invalid.
type RestoreBreathingCommand struct {
	State      BreathState
	Conditions ConditionSet
	Target     CombatantID
	Record     HazardRecord
}

// RestoreBreathingResult is the closed set of outcomes of RestoreBreathing.Run.
// Variants: BreathingRestored.
type RestoreBreathingResult interface{ isRestoreBreathingResult() }

// BreathingRestored is the creature able to breathe again: BreathNormal, its
// conditions and hazard record, whether the Exhaustion was reduced, the levels actually
// removed and the events (the condition events, then BreathRestored; none when the
// creature was not suffocating and the record held no Suffocation levels).
type BreathingRestored struct {
	state         BreathState
	conditions    ConditionSet
	record        HazardRecord
	reduction     ExhaustionReductionOutcome
	levelsRemoved int
	events        []Event
}

// State returns the breathing after the call, BreathNormal.
func (r BreathingRestored) State() BreathState { return r.state }

// Conditions returns the creature's conditions after the call.
func (r BreathingRestored) Conditions() ConditionSet { return r.conditions }

// Record returns the hazard record after the call, with no Suffocation levels.
func (r BreathingRestored) Record() HazardRecord { return r.record }

// Reduction returns whether ConditionSet.ReduceExhaustion was called, and its result.
func (r BreathingRestored) Reduction() ExhaustionReductionOutcome { return r.reduction }

// LevelsRemoved returns the Exhaustion levels actually removed.
func (r BreathingRestored) LevelsRemoved() int { return r.levelsRemoved }

// Events returns a copy of the events, in the order they happened.
func (r BreathingRestored) Events() []Event { return copyEvents(r.events) }

func (BreathingRestored) isRestoreBreathingResult() {}

// RestoreBreathing is the use case the host calls when a creature can breathe again: it
// loses the Exhaustion levels it gained from suffocating, and only those (SRD 5.2.1,
// "Suffocation [Hazard]", p. 189). It rolls no die. Create it with NewRestoreBreathing;
// the zero value is silent and works. Safe for concurrent use.
type RestoreBreathing struct{ environmentUse }

// NewRestoreBreathing returns the use case. Options: WithTelemetry, WithClock.
func NewRestoreBreathing(opts ...Option) RestoreBreathing {
	return RestoreBreathing{newEnvironmentUse(opts)}
}

// Run restores the breathing and records the shared signals. A record that holds more
// Suffocation levels than the set holds Exhaustion ends the Exhaustion without a
// negative level. It reduces nothing, and calls ConditionSet.ReduceExhaustion never
// with 0 levels, when the record holds no Suffocation levels.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target or a State that is not valid, and ctx.Err() unwrapped, before any signal, for
// a cancelled ctx.
func (u RestoreBreathing) Run(ctx context.Context, cmd RestoreBreathingCommand) (RestoreBreathingResult, error) {
	if !validBreathState(cmd.State) {
		return nil, invalidInput("invalid breath state")
	}
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationRestoreBreathing, spanRestoreBreathing},
		func(context.Context) (RestoreBreathingResult, error) {
			r, err := restoreBreathing(cmd.State, cmd.Conditions, cmd.Target, cmd.Record)
			if err != nil {
				return nil, err
			}
			return BreathingRestored{BreathNormal{}, r.set, r.record, r.reduction, r.removed, r.events}, nil
		},
		func(r RestoreBreathingResult) description {
			return description{Class: resultApplied, Name: valueResultRestored, Events: r.(BreathingRestored).events}
		})
}
