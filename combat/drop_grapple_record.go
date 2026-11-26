package combat

import (
	"context"
	"fmt"
	"slices"
)

// DropGrappleRecordCommand is the input of DropGrappleRecord.Run: the grapples in
// force and the target whose Grappled condition ended. Build it with a keyed
// literal.
type DropGrappleRecordCommand struct {
	Grapples Grapples
	Target   CombatantID
}

// DropGrappleRecordResult is the closed set of outcomes of DropGrappleRecord.Run.
// Variants: DropGrappleRecordApplied, DropGrappleRecordRejected.
type DropGrappleRecordResult interface{ isDropGrappleRecordResult() }

// DropGrappleRecordApplied: the record was dropped, with one GrappleRecordDropped.
type DropGrappleRecordApplied struct {
	grapples Grapples
	events   []Event
}

// Grapples returns the set without the dropped record.
func (r DropGrappleRecordApplied) Grapples() Grapples { return r.grapples }

// Events returns a copy of the events: one GrappleRecordDropped.
func (r DropGrappleRecordApplied) Events() []Event { return slices.Clone(r.events) }

// DropGrappleRecordRejected: no grapple holds the target. Construct it with a keyed
// literal.
type DropGrappleRecordRejected struct{ Rejection GrappleRejection }

func (DropGrappleRecordApplied) isDropGrappleRecordResult()  {}
func (DropGrappleRecordRejected) isDropGrappleRecordResult() {}

// DropGrappleRecord is the use case that drops the record of a grapple whose Grappled
// condition ended by some other rule, or whose Grappled condition could not be
// applied after BeginGrapple recorded it, so the record never outlives the
// condition. It rolls no die and touches no ConditionSet. Create it with
// NewDropGrappleRecord; the zero value is silent and works. Safe for concurrent use.
type DropGrappleRecord struct{ grappleUse }

// NewDropGrappleRecord returns the use case. Options: WithTelemetry, WithClock.
func NewDropGrappleRecord(opts ...Option) DropGrappleRecord {
	return DropGrappleRecord{newGrappleUse(opts)}
}

// Run drops the record with Grapples.ConditionEnded. It records the shared signals
// plus, for an applied result, srd5e.combat.grapple.ended (condition_ended) and
// srd5e.combat.grapple.in_force.
//
// It returns ctx.Err() unwrapped, before any signal, for a cancelled ctx. A target
// that is not grappled is a DropGrappleRecordRejected result, never an error.
func (u DropGrappleRecord) Run(ctx context.Context, cmd DropGrappleRecordCommand) (DropGrappleRecordResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationDropGrappleRecord, spanDropGrappleRecord}
	res, err := run(ctx, u.in, op, func(context.Context) (DropGrappleRecordResult, error) {
		switch t := cmd.Grapples.ConditionEnded(cmd.Target).(type) {
		case GrappleTransitionApplied:
			return DropGrappleRecordApplied{t.grapples, t.events}, nil
		case GrappleTransitionRejected:
			return DropGrappleRecordRejected(t), nil
		default:
			panic(fmt.Errorf("combat: unreachable grapple transition %T", t))
		}
	}, func(r DropGrappleRecordResult) description {
		switch v := r.(type) {
		case DropGrappleRecordApplied:
			return description{Class: resultApplied, Name: valueResultApplied, Events: v.events}
		case DropGrappleRecordRejected:
			return refusedGrapple(v.Rejection)
		default:
			panic(fmt.Errorf("combat: unreachable drop grapple record result %T", r))
		}
	})
	if err != nil {
		return nil, err
	}
	if v, ok := res.(DropGrappleRecordApplied); ok {
		u.recordEnded(ctx, valueGrappleEndConditionEnded)
		u.recordInForce(ctx, op, v.grapples)
	}
	return res, nil
}
