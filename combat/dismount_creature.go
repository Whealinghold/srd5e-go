package combat

import (
	"context"
	"fmt"
	"slices"
)

// DismountCreatureCommand is the immutable input of DismountCreature.Run. Build it with
// NewDismountCreatureCommand; the zero value is invalid. It declares no TurnTracker, resource,
// condition or action field.
type DismountCreatureCommand struct {
	rider  CombatantID
	status RiderStatus
	turn   MountingTurnFacts
}

// NewDismountCreatureCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a blank rider, turn facts that are not from their constructor, or
// a rider that is its own mount in status. It never panics.
func NewDismountCreatureCommand(rider CombatantID, status RiderStatus, turn MountingTurnFacts) (DismountCreatureCommand, error) {
	if rider.IsZero() || !turn.valid() || status.IsOn(rider) {
		return DismountCreatureCommand{}, invalidInput("dismounting needs a rider, turn facts and a mount other than the rider")
	}
	return DismountCreatureCommand{rider, status, turn}, nil
}

// DismountCreatureResult is the closed set of outcomes of DismountCreature.Run. Variants:
// DismountCreatureApplied, DismountCreatureRejected.
type DismountCreatureResult interface{ isDismountCreatureResult() }

// DismountCreatureApplied: the rider is no longer mounted. The caller charges MovementCostFeet.
type DismountCreatureApplied struct {
	costFeet int
	events   []MountEvent
}

// RiderStatus returns NotMounted.
func (DismountCreatureApplied) RiderStatus() RiderStatus { return NotMounted() }

// MovementCostFeet returns half the walking Speed, rounded down.
func (r DismountCreatureApplied) MovementCostFeet() int { return r.costFeet }

// Events returns a copy of the events: one CreatureDismounted.
func (r DismountCreatureApplied) Events() []MountEvent { return slices.Clone(r.events) }

// DismountCreatureRejected: the rider cannot dismount.
type DismountCreatureRejected struct{ reason DismountRejection }

// Reason returns why, the first in the order of FR-017.
func (r DismountCreatureRejected) Reason() DismountRejection { return r.reason }

func (DismountCreatureApplied) isDismountCreatureResult()  {}
func (DismountCreatureRejected) isDismountCreatureResult() {}

// DismountCreature is the use case of dismounting during the rider's move (SRD 5.2.1,
// "Mounting and Dismounting", p. 15). It reads and charges no tracker. Create it with
// NewDismountCreature; the zero value is usable and silent. Safe for concurrent use if its
// telemetry and clock are.
type DismountCreature struct{ mountingUse }

// NewDismountCreature returns the use case. Options: WithTelemetry, WithClock.
func NewDismountCreature(opts ...Option) DismountCreature {
	return DismountCreature{newMountingUse(opts)}
}

// Run decides whether the rider can dismount and, if so, returns NotMounted, the movement
// cost and one CreatureDismounted event. A refusal is a result and leaves every input unchanged.
//
// It records the shared signals with srd5e.combat.mount_control (the ride's control, absent
// for a rider who is not mounted) and, for an accepted dismount,
// srd5e.combat.mounting_movement_spent. It returns an error matching srderr.ErrInvalidInput,
// before any signal, for the zero command, and ctx.Err() unwrapped, before any signal, for a
// cancelled ctx. It rolls no dice and never fails otherwise.
func (u DismountCreature) Run(ctx context.Context, cmd DismountCreatureCommand) (DismountCreatureResult, error) {
	if cmd.rider.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationDismountCreature, spanDismountCreature}
	control, mounted := cmd.status.Control()
	res, err := run(ctx, u.in, op, func(context.Context) (DismountCreatureResult, error) {
		cost, reason := dismountRefusal(cmd)
		if reason != nil {
			return DismountCreatureRejected{reason}, nil
		}
		return DismountCreatureApplied{cost, []MountEvent{
			CreatureDismounted{mountEventBase{cmd.rider, cmd.status.mount}, control, cost}}}, nil
	}, func(r DismountCreatureResult) description {
		extras := controlExtras(control, mounted)
		switch v := r.(type) {
		case DismountCreatureApplied:
			return description{Class: resultApplied, Name: valueResultApplied, Events: widen(v.events), Extras: extras}
		case DismountCreatureRejected:
			return description{Class: resultRefused, Name: v.reason.String(), Extras: extras}
		default:
			panic(fmt.Errorf("combat: unreachable dismount result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(DismountCreatureApplied); ok {
		u.recordMovementSpent(ctx, op, v.costFeet)
	}
	return res, nil
}
