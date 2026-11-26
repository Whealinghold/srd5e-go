package combat

import "context"

// AdvanceBreathCommand is the input of AdvanceBreath.Run: the creature's breathing,
// the change the host reports and the creature's id. Build it with a keyed literal; the
// zero value is invalid.
type AdvanceBreathCommand struct {
	State  BreathState
	Change BreathChange
	Target CombatantID
}

// AdvanceBreathResult is the closed set of outcomes of AdvanceBreath.Run. Variants:
// BreathAdvanced, BreathAdvanceRejected.
type AdvanceBreathResult interface{ isAdvanceBreathResult() }

// BreathAdvanced: the change took effect. The events are BreathHeldStarted,
// RanOutOfBreath or BeganChoking when the change produced one, none for a hold that
// did not use up the capacity.
type BreathAdvanced struct {
	state  BreathState
	events []Event
}

// State returns the breathing after the change.
func (r BreathAdvanced) State() BreathState { return r.state }

// Events returns a copy of the events, in the order they happened.
func (r BreathAdvanced) Events() []Event { return copyEvents(r.events) }

// BreathAdvanceRejected: the change does not fit the creature's breathing. Construct it
// with a keyed literal.
type BreathAdvanceRejected struct{ Rejection BreathRejection }

func (BreathAdvanced) isAdvanceBreathResult()        {}
func (BreathAdvanceRejected) isAdvanceBreathResult() {}

// AdvanceBreath is the use case the host calls when a creature starts holding its
// breath, holds it for a while or starts choking (SRD 5.2.1, "Suffocation [Hazard]", p.
// 189). It rolls no die. Create it with NewAdvanceBreath; the zero value is silent and
// works. Safe for concurrent use.
type AdvanceBreath struct{ environmentUse }

// NewAdvanceBreath returns the use case. Options: WithTelemetry, WithClock.
func NewAdvanceBreath(opts ...Option) AdvanceBreath { return AdvanceBreath{newEnvironmentUse(opts)} }

// Run applies the change and records the shared signals.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target, a State that is not valid or a nil Change, and ctx.Err() unwrapped, before
// any signal, for a cancelled ctx. A HoldBreathFor with a negative Elapsed is an error
// matching srderr.ErrInvalidInput that the span records; no operations or events are
// counted. A change that does not fit the state is a BreathAdvanceRejected result.
func (u AdvanceBreath) Run(ctx context.Context, cmd AdvanceBreathCommand) (AdvanceBreathResult, error) {
	if !validBreathState(cmd.State) || cmd.Change == nil {
		return nil, invalidInput("invalid breath state or no change")
	}
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationAdvanceBreath, spanAdvanceBreath},
		func(context.Context) (AdvanceBreathResult, error) {
			state, event, rejection, err := advanceBreath(cmd.State, cmd.Change, cmd.Target)
			switch {
			case err != nil:
				return nil, err
			case rejection != 0:
				return BreathAdvanceRejected{rejection}, nil
			case event == nil:
				return BreathAdvanced{state, []Event{}}, nil
			}
			return BreathAdvanced{state, []Event{event}}, nil
		},
		func(r AdvanceBreathResult) description {
			if v, ok := r.(BreathAdvanceRejected); ok {
				return description{Class: resultRefused, Name: v.Rejection.String()}
			}
			return description{Class: resultApplied, Name: valueResultAdvanced, Events: r.(BreathAdvanced).events}
		})
}
