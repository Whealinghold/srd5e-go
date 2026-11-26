package combat

import (
	"context"
	"fmt"
	"slices"
)

// DeployGearCommand is the immutable input of DeployGear.Run. Build it with
// NewDeployGearCommand; the zero value is invalid.
type DeployGearCommand struct {
	gears        DeployedGears
	id           DeployedGearID
	kind         DeployedGearKind
	deployer     CombatantID
	distanceFeet int
	ground       GroundLevel
}

// NewDeployGearCommand returns the command to set out gear of kind with the id for deployer,
// distanceFeet from it, on the ground (SRD 5.2.1, "Utilize", p. 191). The distance and ground
// are host facts: the engine models no map.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a blank
// id or a zero deployer, or GearProblemInvalidValue for a kind or ground that is not valid. A
// negative distance is not refused here: Run answers it with DeployGearInvalid. It never
// panics.
func NewDeployGearCommand(gears DeployedGears, id DeployedGearID, kind DeployedGearKind, deployer CombatantID,
	distanceFeet int, ground GroundLevel) (DeployGearCommand, error) {
	switch {
	case id == "" || deployer.IsZero():
		return DeployGearCommand{}, gearInvalid(GearProblemBlankID)
	case !kind.Valid() || !ground.Valid():
		return DeployGearCommand{}, gearInvalid(GearProblemInvalidValue)
	}
	return DeployGearCommand{gears, id, kind, deployer, distanceFeet, ground}, nil
}

// DeployGearResult is the closed set of outcomes of DeployGear.Run. Variants:
// DeployGearPlaced, DeployGearRejected, DeployGearInvalid.
type DeployGearResult interface{ isDeployGearResult() }

// DeployGearPlaced: the gear is set out.
type DeployGearPlaced struct {
	gears  DeployedGears
	events []GearEvent
}

// Gears returns the new set.
func (r DeployGearPlaced) Gears() DeployedGears { return r.gears }

// Events returns a copy of the events: one DeployedGearPlaced.
func (r DeployGearPlaced) Events() []GearEvent { return slices.Clone(r.events) }

// DeployGearRejected: the rules refuse; the set is unchanged.
type DeployGearRejected struct{ reason GearRejection }

// Reason returns why: idInUse, outOfReach or groundNotLevel.
func (r DeployGearRejected) Reason() GearRejection { return r.reason }

// DeployGearInvalid: the command holds a value the rules cannot use (a negative distance).
type DeployGearInvalid struct{ problem GearProblem }

// Problem returns what is wrong.
func (r DeployGearInvalid) Problem() GearProblem { return r.problem }

func (DeployGearPlaced) isDeployGearResult()   {}
func (DeployGearRejected) isDeployGearResult() {}
func (DeployGearInvalid) isDeployGearResult()  {}

// DeployGear is the use case of setting gear out: Caltrops, Ball Bearings, Oil or a Hunting
// Trap, as a Utilize action (SRD 5.2.1, pp. 96-98). It rolls nothing. Create it with
// NewDeployGear; the zero value is usable and silent. Safe for concurrent use if its
// telemetry and clock are.
type DeployGear struct{ gearUse }

// NewDeployGear returns the use case. Options: WithTelemetry, WithClock.
func NewDeployGear(opts ...Option) DeployGear { return DeployGear{newGearUse(opts)} }

// Run sets the gear out and returns the new set and one DeployedGearPlaced. A refusal or an
// invalid value is a result and leaves every input unchanged. It records the shared signals
// only.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It rolls no dice
// and never fails otherwise.
func (u DeployGear) Run(ctx context.Context, cmd DeployGearCommand) (DeployGearResult, error) {
	if cmd.id == "" {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationDeployGear, spanDeployGear}
	return run(ctx, u.in, op, func(context.Context) (DeployGearResult, error) {
		t, err := cmd.gears.Deploy(cmd.id, cmd.kind, cmd.deployer, cmd.distanceFeet, cmd.ground)
		if p, ok := gearProblem(err); ok {
			return DeployGearInvalid{p}, nil
		}
		switch v := t.(type) {
		case DeployedGearsApplied:
			return DeployGearPlaced(v), nil
		case DeployedGearsRejected:
			return DeployGearRejected(v), nil
		default:
			panic(unreachableError("deploy transition", t))
		}
	}, func(r DeployGearResult) description {
		switch v := r.(type) {
		case DeployGearPlaced:
			return description{Class: resultApplied, Name: valueResultPlaced, Events: widen(v.events)}
		case DeployGearRejected:
			return description{Class: resultRefused, Name: v.reason.String()}
		case DeployGearInvalid:
			return description{Class: resultInvalid, Name: v.problem.String()}
		default:
			panic(fmt.Errorf("combat: unreachable deploy gear result %T", r))
		}
	})
}
