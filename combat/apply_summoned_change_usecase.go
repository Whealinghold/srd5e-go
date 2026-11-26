package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// ApplySummonedChangeResult is the closed set of outcomes of ApplySummonedChange.Run. Variants: ApplySummonedChangeApplied,
// ApplySummonedChangeRejected, ApplySummonedChangeUnchanged.
type ApplySummonedChangeResult interface{ isApplySummonedChangeResult() }

// ApplySummonedChangeApplied: the change took effect. It holds the new roster, the events in the order they happened and the
// instructions for the encounter's roster (FR-077), and for a delivered touch the familiar's Reaction and for a Life Bond the Hit
// Points the steed regains. Only Run builds one.
type ApplySummonedChangeApplied struct {
	roster      SummonedRoster
	events      []SummonEvent
	actions     []SummonedRosterAction
	reaction    FeatureReaction
	hasReaction bool
	lifeBond    int
}

// Roster returns the new roster.
func (r ApplySummonedChangeApplied) Roster() SummonedRoster { return r.roster }

// Events returns a copy of the events, in the order they happened.
func (r ApplySummonedChangeApplied) Events() []SummonEvent { return slices.Clone(r.events) }

// RosterActions returns a copy of the instructions for the encounter's roster, in creation order.
func (r ApplySummonedChangeApplied) RosterActions() []SummonedRosterAction {
	return slices.Clone(r.actions)
}

// Reaction returns the Reaction a delivered touch spent, the familiar's, and whether there is one (FR-047).
func (r ApplySummonedChangeApplied) Reaction() (FeatureReaction, bool) {
	return r.reaction, r.hasReaction
}

// LifeBondHitPoints returns the Hit Points a steed regains through Life Bond, 0 when the change was not a Life Bond (FR-057).
func (r ApplySummonedChangeApplied) LifeBondHitPoints() int { return r.lifeBond }

// ApplySummonedChangeRejected: the rules refused the change. Rejection says why; the roster is as it came in and there are no
// events.
type ApplySummonedChangeRejected struct {
	Rejection SummonRejection
	roster    SummonedRoster
}

// Roster returns the roster that came in, unchanged.
func (r ApplySummonedChangeRejected) Roster() SummonedRoster { return r.roster }

// ApplySummonedChangeUnchanged: the change had nothing to change: the creature is not in the roster or already departed, or the
// spell ended nothing (FR-026). It holds the roster that came in.
type ApplySummonedChangeUnchanged struct{ roster SummonedRoster }

// Roster returns the roster that came in, unchanged.
func (r ApplySummonedChangeUnchanged) Roster() SummonedRoster { return r.roster }

func (ApplySummonedChangeApplied) isApplySummonedChangeResult()   {}
func (ApplySummonedChangeRejected) isApplySummonedChangeResult()  {}
func (ApplySummonedChangeUnchanged) isApplySummonedChangeResult() {}

// ApplySummonedChange is the use case of one report or order about summoned creatures: it composes SummonedRoster.Apply and
// records the signals of summoned-and-companion-creatures.md section 7. It is synchronous, reads no port and holds no rule. Create
// it with NewApplySummonedChange; the zero value is silent and works. It calls no other use case. It is safe for concurrent use if
// its telemetry and clock are.
type ApplySummonedChange struct {
	in  instrumenter
	tel telemetry.Telemetry
}

// NewApplySummonedChange returns the use case. Options: WithTelemetry, WithClock.
func NewApplySummonedChange(opts ...Option) ApplySummonedChange {
	o := newOptions(opts)
	return ApplySummonedChange{newInstrumenter(o.tel, o.clock), o.tel}
}

// Run applies the change. It records the shared signals under the operation apply_summoned_change with srd5e.combat.result
// commanded, departed, lapsed, elapsed, dismissed, recalled, repaired, touch_delivered, senses_shared, life_bond, turn_reported,
// task_completed, action_allowed, unchanged or rejected_<reason> (refused, or limit_exceeded, invalid), and one
// srd5e.combat.summon_endings count for each creature that left. It logs nothing. ctx is used for the span.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero command, and ctx.Err() unwrapped, before
// any signal, for a cancelled ctx. A refusal is a result, never an error.
func (u ApplySummonedChange) Run(ctx context.Context, cmd ApplySummonedChangeCommand) (ApplySummonedChangeResult, error) {
	if cmd.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationApplySummonedChange, spanApplySummonedChange},
		func(ctx context.Context) (ApplySummonedChangeResult, error) {
			switch t := cmd.roster.Apply(cmd.change, cmd.active).(type) {
			case SummonedApplied:
				recordEndings(ctx, u.tel, t.events)
				return ApplySummonedChangeApplied{t.roster, t.Events(), t.RosterActions(), t.reaction, t.hasReaction, t.lifeBond}, nil
			case SummonedRejected:
				return ApplySummonedChangeRejected{t.Rejection, cmd.roster}, nil
			}
			return ApplySummonedChangeUnchanged{cmd.roster}, nil
		},
		func(res ApplySummonedChangeResult) description {
			switch t := res.(type) {
			case ApplySummonedChangeApplied:
				return description{Class: resultApplied, Name: changeResultName(cmd.change, t.events), Events: summonEventsOf(t.events)}
			case ApplySummonedChangeRejected:
				return description{Class: summonRejectionClass(t.Rejection), Name: rejectionResultName(t.Rejection)}
			}
			return description{Class: resultApplied, Name: valueResultUnchanged}
		})
}

// changeResultName is the value of srd5e.combat.result of an applied change.
func changeResultName(change SummonedChange, events []SummonEvent) string {
	switch change.(type) {
	case IssueCommand:
		return valueResultCommanded
	case ReportZeroHitPoints, ReportSpellEnded, ReportSummonerDied, ReportLongRestFinished:
		return valueResultDeparted
	case ElapseControlTime:
		for _, e := range events {
			if _, lapsed := e.(SummonControlLapsed); lapsed {
				return valueResultLapsed
			}
		}
		return valueResultElapsed
	case DismissFamiliar:
		return valueResultDismissed
	case RecallFamiliar:
		return valueResultRecalled
	case ShareFamiliarSenses:
		return valueResultSensesShared
	case DeliverTouch:
		return valueResultTouchDelivered
	case RepairSimulacrum:
		return valueResultRepaired
	case ApplySteedLifeBond:
		return valueResultLifeBond
	case ReportTurnStarted, ReportTurnEnded:
		return valueResultTurnReported
	case ReportTaskComplete:
		return valueResultTaskCompleted
	}
	return valueResultActionAllowed
}
