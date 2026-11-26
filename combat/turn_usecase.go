package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// turnUse is what every initiative and turn use case holds: the instrumenter for the
// shared signals and the telemetry for the signals of initiative-and-actions.md section
// 7. The zero value is silent and usable. Safe for concurrent use if its telemetry and clock are.
type turnUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newTurnUse(opts []Option) turnUse {
	o := newOptions(opts)
	return turnUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// appliedTurnStep is the accepted half of every tracker-backed result: the new tracker
// and the events in the order they happened.
type appliedTurnStep struct {
	tracker TurnTracker
	events  []TurnEvent
}

// Tracker returns the new tracker.
func (a appliedTurnStep) Tracker() TurnTracker { return a.tracker }

// Events returns a copy of the events.
func (a appliedTurnStep) Events() []TurnEvent { return slices.Clone(a.events) }

// convertStep maps a tracker step one to one onto a use case result (FR-149).
func convertStep[R any](step TrackerStep, applied func(appliedTurnStep) R, rejected func(TurnRejection) R) R {
	switch v := step.(type) {
	case TrackerStepApplied:
		return applied(appliedTurnStep(v))
	case TrackerStepRejected:
		return rejected(v.Reason)
	default:
		panic(unreachableError("tracker step", step))
	}
}

// step runs one tracker operation through the instrumenter: the zero tracker and a
// cancelled ctx return an error before any signal; a refusal is recorded as its class
// and an accepted move, stand-up or readied move records the movement it spent.
func (u turnUse) step(ctx context.Context, op combatOperation, tracker TurnTracker, work func() (TrackerStep, error)) (TrackerStep, error) {
	if tracker.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	step, err := run(ctx, u.in, op, func(context.Context) (TrackerStep, error) { return work() }, func(s TrackerStep) description {
		switch v := s.(type) {
		case TrackerStepApplied:
			return description{Class: resultApplied, Name: valueResultApplied, Events: widen(v.events)}
		case TrackerStepRejected:
			return description{Class: turnRejectionClass(v.Reason), Name: v.Reason.String()}
		default:
			panic(unreachableError("tracker step", s))
		}
	})
	if err != nil {
		return nil, err
	}
	if a, ok := step.(TrackerStepApplied); ok {
		u.recordMovement(ctx, a.events)
	}
	return step, nil
}

// recordMovement records srd5e.combat.turn_movement_spent from the events, taking every
// amount from them: one sample per mode of a Move and of a readied move, one with the
// mode posture for a stand-up (FR-104).
func (u turnUse) recordMovement(ctx context.Context, events []TurnEvent) {
	h := u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricTurnMovementSpent, Unit: "[ft_i]",
		Description: "Movement spent per accepted move, stand-up or readied move, one sample per mode."})
	sample := func(mode string, feet int) {
		h.Record(ctx, float64(feet), telemetry.String(attrMovementMode, mode))
	}
	perMode := func(charges []MovementCharge) {
		var order []MovementMode
		sums := map[MovementMode]int{}
		for _, c := range charges {
			if _, seen := sums[c.Mode]; !seen {
				order = append(order, c.Mode)
			}
			sums[c.Mode] = satAdd(sums[c.Mode], c.Feet)
		}
		for _, m := range order {
			sample(m.String(), sums[m])
		}
	}
	for _, e := range events {
		switch v := e.(type) {
		case MovementSpent:
			perMode(v.charges)
		case StoodUp:
			sample(valueMovementModePosture, v.cost)
		case ReactionTaken:
			if r, ok := v.record.(ReadiedActionRecord); ok {
				if m, ok := r.Outcome.(ReadiedMoveOutcome); ok {
					perMode(m.charges)
				}
			}
		}
	}
}
