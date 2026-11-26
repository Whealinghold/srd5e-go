package combat

import "context"

// LandAfterFallCommand is the input of LandAfterFall.Run: the creature's conditions
// and id, and the damage the fall finally dealt. Build it with a keyed literal; the
// zero value is invalid.
type LandAfterFallCommand struct {
	Conditions ConditionSet
	Target     CombatantID
	// DamageTaken is the final amount after Resistance, Vulnerability and Immunity, 0 or more.
	DamageTaken int
}

// LandAfterFallResult is the closed set of outcomes of LandAfterFall.Run. Variants:
// FallLandedProne, FallLandedStanding.
type LandAfterFallResult interface{ isLandAfterFallResult() }

// FallLandedProne: the fall dealt damage, so Prone was applied.
type FallLandedProne struct {
	conditions  ConditionSet
	application ApplyConditionToSetResult
	events      []Event
}

// Conditions returns the creature's conditions after the call.
func (r FallLandedProne) Conditions() ConditionSet { return r.conditions }

// Application returns the result of applying Prone: ConditionAdded, ConditionAlreadyHeld
// or ConditionImmune.
func (r FallLandedProne) Application() ApplyConditionToSetResult { return r.application }

// Events returns a copy of the condition events, in the order they happened.
func (r FallLandedProne) Events() []Event { return copyEvents(r.events) }

// FallLandedStanding: the fall dealt no damage, so the creature is not Prone.
type FallLandedStanding struct{ conditions ConditionSet }

// Conditions returns the creature's conditions, unchanged.
func (r FallLandedStanding) Conditions() ConditionSet { return r.conditions }

func (FallLandedProne) isLandAfterFallResult()    {}
func (FallLandedStanding) isLandAfterFallResult() {}

// LandAfterFall is the use case that lands a creature Prone after a fall unless it
// avoided all damage (SRD 5.2.1, "Falling [Hazard]", p. 182). It rolls no die. Create
// it with NewLandAfterFall; the zero value is silent and works. Safe for concurrent
// use.
type LandAfterFall struct{ environmentUse }

// NewLandAfterFall returns the use case. Options: WithTelemetry, WithClock.
func NewLandAfterFall(opts ...Option) LandAfterFall { return LandAfterFall{newEnvironmentUse(opts)} }

// Run applies Prone through ConditionSet.Apply for a damage of 1 or more and records
// the shared signals.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for a zero
// target, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. A negative
// DamageTaken is an error matching srderr.ErrInvalidInput that the span records; no
// operations or events are counted.
func (u LandAfterFall) Run(ctx context.Context, cmd LandAfterFallCommand) (LandAfterFallResult, error) {
	if err := u.begin(ctx, cmd.Target); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationLandAfterFall, spanLandAfterFall},
		func(context.Context) (LandAfterFallResult, error) {
			switch {
			case cmd.DamageTaken < 0:
				return nil, invalidInput("damage taken below 0")
			case cmd.DamageTaken == 0:
				return FallLandedStanding{cmd.Conditions}, nil
			}
			app := cmd.Conditions.Apply(cmd.Target, Prone{})
			var events []Event
			if added, ok := app.(ConditionAdded); ok {
				events = widen(added.events)
			}
			return FallLandedProne{app.Set(), app, events}, nil
		},
		func(r LandAfterFallResult) description {
			if v, ok := r.(FallLandedProne); ok {
				return description{Class: resultApplied, Name: valueResultLandedProne, Events: v.events}
			}
			return description{Class: resultApplied, Name: valueResultLandedStanding}
		})
}
