package combat

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// GearMaintenance is the closed set of upkeep of deployed gear. Variants: LightOil,
// RecoverGear, ResetHuntingTrap, ExpireBurningOil. Construct them with keyed literals.
type GearMaintenance interface{ isGearMaintenance() }

// LightOil lights the unlit poured Oil with GearID at Stamp.
type LightOil struct {
	GearID DeployedGearID
	Stamp  TurnStamp
}

// RecoverGear gathers up the Caltrops or Ball Bearings with GearID.
type RecoverGear struct{ GearID DeployedGearID }

// ResetHuntingTrap sets the sprung, empty Hunting Trap with GearID again.
type ResetHuntingTrap struct{ GearID DeployedGearID }

// ExpireBurningOil burns out every Oil that has burned through EndedTurn. The host calls it at
// the end of every turn, or Oil stays burning past its time.
type ExpireBurningOil struct{ EndedTurn TurnStamp }

func (LightOil) isGearMaintenance()         {}
func (RecoverGear) isGearMaintenance()      {}
func (ResetHuntingTrap) isGearMaintenance() {}
func (ExpireBurningOil) isGearMaintenance() {}

// validMaintenance reports whether m is a known variant whose ids and stamp are not blank or zero.
func validMaintenance(m GearMaintenance) bool {
	blank := func(id DeployedGearID) bool { return strings.TrimSpace(string(id)) == "" }
	switch v := m.(type) {
	case LightOil:
		return !blank(v.GearID) && !v.Stamp.IsZero()
	case RecoverGear:
		return !blank(v.GearID)
	case ResetHuntingTrap:
		return !blank(v.GearID)
	case ExpireBurningOil:
		return !v.EndedTurn.IsZero()
	}
	return false
}

// MaintainDeployedGearCommand is the immutable input of MaintainDeployedGear.Run. Build it with
// NewMaintainDeployedGearCommand; the zero value is invalid.
type MaintainDeployedGearCommand struct {
	gears       DeployedGears
	holds       GearHolds
	maintenance GearMaintenance
}

// NewMaintainDeployedGearCommand returns the command to do maintenance on gears. The holds are
// read by ResetHuntingTrap only, to see whether the trap is occupied.
//
// It returns a *GearError (matching srderr.ErrInvalidInput, GearProblemInvalidValue) for a
// maintenance that is nil, of another type, or holds a blank id or the zero stamp. It never
// panics.
func NewMaintainDeployedGearCommand(gears DeployedGears, holds GearHolds, m GearMaintenance) (MaintainDeployedGearCommand, error) {
	if !validMaintenance(m) {
		return MaintainDeployedGearCommand{}, gearInvalid(GearProblemInvalidValue)
	}
	return MaintainDeployedGearCommand{gears, holds, m}, nil
}

// MaintainDeployedGearResult is the closed set of outcomes of MaintainDeployedGear.Run.
// Variants: MaintainDeployedGearApplied, MaintainDeployedGearRejected.
type MaintainDeployedGearResult interface{ isMaintainDeployedGearResult() }

// MaintainDeployedGearApplied: the maintenance took effect (burning Oil that has not burned out
// is a normal result with no event).
type MaintainDeployedGearApplied struct {
	gears  DeployedGears
	events []GearEvent
}

// Gears returns the new set.
func (r MaintainDeployedGearApplied) Gears() DeployedGears { return r.gears }

// Events returns a copy of the events, in the order they happened.
func (r MaintainDeployedGearApplied) Events() []GearEvent { return slices.Clone(r.events) }

// MaintainDeployedGearRejected: the rules refuse; the set is unchanged.
type MaintainDeployedGearRejected struct{ reason GearRejection }

// Reason returns why.
func (r MaintainDeployedGearRejected) Reason() GearRejection { return r.reason }

func (MaintainDeployedGearApplied) isMaintainDeployedGearResult()  {}
func (MaintainDeployedGearRejected) isMaintainDeployedGearResult() {}

// MaintainDeployedGear is the use case of lighting Oil, recovering Caltrops and Ball Bearings,
// resetting a Hunting Trap and burning out Oil at the end of a turn (SRD 5.2.1, pp. 96-98). It
// rolls nothing. Create it with NewMaintainDeployedGear; the zero value is usable and silent.
// Safe for concurrent use if its telemetry and clock are.
type MaintainDeployedGear struct{ gearUse }

// NewMaintainDeployedGear returns the use case. Options: WithTelemetry, WithClock.
func NewMaintainDeployedGear(opts ...Option) MaintainDeployedGear {
	return MaintainDeployedGear{newGearUse(opts)}
}

// Run does the maintenance and returns the new set and the events. A refusal is a result and
// leaves every input unchanged.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero command,
// and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It rolls no dice and never
// fails otherwise.
func (u MaintainDeployedGear) Run(ctx context.Context, cmd MaintainDeployedGearCommand) (MaintainDeployedGearResult, error) {
	if cmd.maintenance == nil {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationMaintainDeployedGear, spanMaintainDeployedGear}
	return run(ctx, u.in, op, func(context.Context) (MaintainDeployedGearResult, error) {
		var t DeployedGearsTransition
		var err error
		switch m := cmd.maintenance.(type) {
		case LightOil:
			t, err = cmd.gears.Light(m.GearID, m.Stamp)
		case RecoverGear:
			t = cmd.gears.Recover(m.GearID)
		case ResetHuntingTrap:
			t = cmd.gears.Reset(m.GearID, cmd.holds)
		case ExpireBurningOil:
			var applied DeployedGearsApplied
			applied, err = cmd.gears.Expire(m.EndedTurn)
			t = applied
		}
		if err != nil {
			return nil, err
		}
		switch v := t.(type) {
		case DeployedGearsApplied:
			return MaintainDeployedGearApplied(v), nil
		case DeployedGearsRejected:
			return MaintainDeployedGearRejected(v), nil
		default:
			panic(unreachableError("maintenance transition", t))
		}
	}, func(r MaintainDeployedGearResult) description {
		switch v := r.(type) {
		case MaintainDeployedGearApplied:
			return description{Class: resultApplied, Name: valueResultApplied, Events: widen(v.events)}
		case MaintainDeployedGearRejected:
			return description{Class: resultRefused, Name: v.reason.String()}
		default:
			panic(fmt.Errorf("combat: unreachable maintain deployed gear result %T", r))
		}
	})
}
