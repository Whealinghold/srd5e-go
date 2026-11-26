package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// MountCreatureInput holds the values NewMountCreatureCommand validates. Build it with a
// keyed literal. The engine models no map: the host reports the sizes, the distance and the
// facts about the mount (SRD 5.2.1, "Mounted Combat", p. 15).
type MountCreatureInput struct {
	Rider, Mount CombatantID
	RiderSize    content.CreatureSize
	MountSize    content.CreatureSize
	// Status is the rider's current status; a rider already mounted is refused.
	Status RiderStatus
	Facts  MountFacts
	Turn   MountingTurnFacts
	// DistanceFeet is the distance between rider and mount, 0 or more.
	DistanceFeet int
	// Control is the control the rider asks for.
	Control MountControl
}

// MountCreatureCommand is the immutable input of MountCreature.Run. Build it with
// NewMountCreatureCommand; the zero value is invalid. It declares no TurnTracker, resource,
// condition or field about the mount's own turn.
type MountCreatureCommand struct {
	rider, mount         CombatantID
	riderSize, mountSize content.CreatureSize
	status               RiderStatus
	facts                MountFacts
	turn                 MountingTurnFacts
	distanceFeet         int
	control              MountControl
}

// NewMountCreatureCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a blank id, the same id for rider and mount, a size that is not
// valid, facts or turn facts that are not from their constructors, a negative distance or a
// control that is not valid. It never panics. Ids are opaque: it does not check that they
// are in any tracker.
func NewMountCreatureCommand(in MountCreatureInput) (MountCreatureCommand, error) {
	switch {
	case in.Rider.IsZero() || in.Mount.IsZero() || in.Rider == in.Mount:
		return MountCreatureCommand{}, invalidInput("mounting needs two different creatures")
	case !in.RiderSize.Valid() || !in.MountSize.Valid():
		return MountCreatureCommand{}, invalidInput("mounting needs two valid sizes")
	case !in.Facts.valid() || !in.Turn.valid() || !in.Control.Valid():
		return MountCreatureCommand{}, invalidInput("mounting needs mount facts, turn facts and a control")
	case in.DistanceFeet < 0:
		return MountCreatureCommand{}, invalidInput("mounting distance is negative")
	}
	return MountCreatureCommand{in.Rider, in.Mount, in.RiderSize, in.MountSize, in.Status, in.Facts, in.Turn, in.DistanceFeet, in.Control}, nil
}

// MountCreatureResult is the closed set of outcomes of MountCreature.Run. Variants:
// MountCreatureApplied, MountCreatureRejected.
type MountCreatureResult interface{ isMountCreatureResult() }

// MountCreatureApplied: the rider is on the mount. The caller charges MovementCostFeet.
type MountCreatureApplied struct {
	status   RiderStatus
	costFeet int
	events   []MountEvent
}

// RiderStatus returns the new status, MountedOn the mount with the control asked for.
func (r MountCreatureApplied) RiderStatus() RiderStatus { return r.status }

// MovementCostFeet returns half the walking Speed, rounded down.
func (r MountCreatureApplied) MovementCostFeet() int { return r.costFeet }

// Events returns a copy of the events: one CreatureMounted.
func (r MountCreatureApplied) Events() []MountEvent { return slices.Clone(r.events) }

// MountCreatureRejected: the rider cannot mount.
type MountCreatureRejected struct{ reason MountRejection }

// Reason returns why, the first in the order of FR-017.
func (r MountCreatureRejected) Reason() MountRejection { return r.reason }

func (MountCreatureApplied) isMountCreatureResult()  {}
func (MountCreatureRejected) isMountCreatureResult() {}

// MountCreature is the use case of mounting a creature during the rider's move (SRD
// 5.2.1, "Mounting and Dismounting", p. 15). It reads and charges no tracker: it returns
// the movement cost as data. Create it with NewMountCreature; the zero value is usable and
// silent. Safe for concurrent use if its telemetry and clock are.
type MountCreature struct{ mountingUse }

// NewMountCreature returns the use case. Options: WithTelemetry, WithClock.
func NewMountCreature(opts ...Option) MountCreature { return MountCreature{newMountingUse(opts)} }

// Run decides whether the rider can mount and, if so, returns the new status, the movement
// cost and one CreatureMounted event. A refusal is a result and leaves every input unchanged.
//
// It records the shared signals with srd5e.combat.mount_control (the control asked for) and,
// for an accepted mount, srd5e.combat.mounting_movement_spent. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero command, and ctx.Err() unwrapped,
// before any signal, for a cancelled ctx. It rolls no dice and never fails otherwise.
func (u MountCreature) Run(ctx context.Context, cmd MountCreatureCommand) (MountCreatureResult, error) {
	if cmd.rider.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationMountCreature, spanMountCreature}
	res, err := run(ctx, u.in, op, func(context.Context) (MountCreatureResult, error) {
		cost, reason := mountRefusal(cmd)
		if reason != nil {
			return MountCreatureRejected{reason}, nil
		}
		initiative, turn := ridingOutcome(cmd.control, cmd.rider)
		return MountCreatureApplied{RiderStatus{cmd.mount, cmd.control}, cost,
			[]MountEvent{CreatureMounted{mountEventBase{cmd.rider, cmd.mount}, cmd.control, cost, initiative, turn}}}, nil
	}, func(r MountCreatureResult) description {
		extras := controlExtras(cmd.control, true)
		switch v := r.(type) {
		case MountCreatureApplied:
			return description{Class: resultApplied, Name: valueResultApplied, Events: widen(v.events), Extras: extras}
		case MountCreatureRejected:
			return description{Class: resultRefused, Name: v.reason.String(), Extras: extras}
		default:
			panic(fmt.Errorf("combat: unreachable mount result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(MountCreatureApplied); ok {
		u.recordMovementSpent(ctx, op, v.costFeet)
	}
	return res, nil
}
