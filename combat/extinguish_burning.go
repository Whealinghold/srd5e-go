package combat

import "context"

// ExtinguishBurningCommand is the input of ExtinguishBurning.Run: the creature's
// state, how it puts the fire out, its conditions and its id. Build it with a keyed
// literal; the zero value is invalid.
type ExtinguishBurningCommand struct {
	State      BurningState
	Method     ExtinguishMethod
	Conditions ConditionSet
	Target     CombatantID
}

// ExtinguishBurningResult is the closed set of outcomes of ExtinguishBurning.Run.
// Variants: BurningExtinguished, BurningExtinguishRejected.
type ExtinguishBurningResult interface{ isExtinguishBurningResult() }

// BurningExtinguished: the fire went out. Rolling on the ground also applied Prone.
type BurningExtinguished struct {
	state            BurningState
	conditions       ConditionSet
	proneApplication ProneApplicationOutcome
	events           []Event
}

// State returns the state after the call, BurningStateNotBurning.
func (r BurningExtinguished) State() BurningState { return r.state }

// Conditions returns the creature's conditions after the call.
func (r BurningExtinguished) Conditions() ConditionSet { return r.conditions }

// ProneApplication returns whether Prone was applied, and with what result.
func (r BurningExtinguished) ProneApplication() ProneApplicationOutcome { return r.proneApplication }

// Events returns a copy of the events in the order they happened: the condition
// events, then BurningEnded.
func (r BurningExtinguished) Events() []Event { return copyEvents(r.events) }

// BurningExtinguishRejected: the creature is not burning, or is immune to Prone and so
// cannot put the fire out by rolling. Construct it with a keyed literal.
type BurningExtinguishRejected struct{ Rejection BurningRejection }

func (BurningExtinguished) isExtinguishBurningResult()       {}
func (BurningExtinguishRejected) isExtinguishBurningResult() {}

// ExtinguishBurning is the use case that puts a creature's fire out, by rolling on
// the ground (which gives it the Prone condition), dousing, submerging or
// suffocating it (SRD 5.2.1, "Burning [Hazard]", p. 178). It rolls no die. Create it
// with NewExtinguishBurning; the zero value is silent and works. Safe for concurrent
// use.
type ExtinguishBurning struct{ environmentUse }

// NewExtinguishBurning returns the use case. Options: WithTelemetry, WithClock.
func NewExtinguishBurning(opts ...Option) ExtinguishBurning {
	return ExtinguishBurning{newEnvironmentUse(opts)}
}

// Run puts the fire out and records the shared signals.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target, a State or Method that is not valid, and ctx.Err() unwrapped, before any
// signal, for a cancelled ctx. A creature that is not burning, or is immune to Prone
// and rolls on the ground, is a BurningExtinguishRejected result, never an error.
func (u ExtinguishBurning) Run(ctx context.Context, cmd ExtinguishBurningCommand) (ExtinguishBurningResult, error) {
	if !cmd.State.Valid() || !cmd.Method.Valid() {
		return nil, invalidInput("invalid burning state or extinguish method")
	}
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationExtinguishBurning, spanExtinguishBurning},
		func(context.Context) (ExtinguishBurningResult, error) {
			rejection, set, prone, events := extinguish(cmd.State, cmd.Method, cmd.Conditions, cmd.Target)
			if rejection != 0 {
				return BurningExtinguishRejected{rejection}, nil
			}
			ended, err := NewBurningEnded(cmd.Target, cmd.Method)
			if err != nil {
				return nil, err
			}
			return BurningExtinguished{BurningStateNotBurning, set, prone, append(events, ended)}, nil
		},
		func(r ExtinguishBurningResult) description {
			if v, ok := r.(BurningExtinguishRejected); ok {
				return description{Class: resultRefused, Name: v.Rejection.String()}
			}
			return description{Class: resultApplied, Name: valueResultExtinguished, Events: r.(BurningExtinguished).events}
		})
}
