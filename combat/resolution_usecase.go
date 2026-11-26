package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Every use case of combat action resolution takes an Encounter (and an ActionLedger where the spec says
// so), runs one thing a combatant does by composing the root's methods, the tracker operations and the
// pure services, and returns the new state and the ordered events (combat-action-resolution.md section 1).
// No use case calls another (FR-007). Each is built by NewX (a Roller first where a die is rolled, then
// Options: WithTelemetry, WithClock) and has one method Run(ctx, cmd) returning its own closed result,
// <UseCase>Completed, <UseCase>Rejected or <UseCase>Invalid. A call is atomic: a Rejected or Invalid result
// carries the encounter and ledger that came in, equal, and no event; a Completed one carries the new ones.
// A rule outcome is never an error. Run returns an error matching srderr.ErrInvalidInput, before any signal,
// for the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a
// *srderr.PortError (error type "dice_roller_error"), recorded once, when the roller fails, panics or breaks
// its contract, leaving the host's state as it was. They are safe for concurrent use if their roller,
// telemetry and clock are.

// resolutionOutcome is what every result variant of these use cases offers the instrumenter. Implemented by
// resolutionCompleted, resolutionRejected and resolutionInvalid, which every variant embeds.
type resolutionOutcome interface {
	describe() description
	Events() []Event
}

// resolutionUse is what every combat action resolution use case holds. The zero value is silent; one that
// rolls needs a roller.
type resolutionUse struct {
	in   instrumenter
	tel  telemetry.Telemetry
	flow attackFlow
}

// newResolutionUse returns the use case state for the options and the roller, which may be nil for a use
// case that rolls nothing.
func newResolutionUse(roller Roller, opts []Option) resolutionUse {
	o := newOptions(opts)
	return resolutionUse{newInstrumenter(o.tel, o.clock), o.tel, attackFlow{roller}}
}

// describeOutcome classifies a result variant.
func describeOutcome(r any) description {
	switch v := r.(type) {
	case resolutionOutcome:
		return v.describe()
	default:
		panic(unreachableError("resolution result", r))
	}
}

// outcomeEvents returns the events of a result variant.
func outcomeEvents(r any) []Event {
	if v, ok := r.(resolutionOutcome); ok {
		return v.Events()
	}
	return nil
}

// execute runs work in the instrumenter under op and counts one srd5e.combat.deaths per wrapped Died of a
// result. A cancelled ctx is returned before any signal.
func execute[R any](ctx context.Context, u resolutionUse, op combatOperation, work func() (R, error)) (R, error) {
	var zero R
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	res, err := run(ctx, u.in, op, func(context.Context) (R, error) { return work() },
		func(r R) description { return describeOutcome(r) })
	if err != nil {
		return zero, err
	}
	checks := u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricConcentrationChecks, Unit: "{check}",
		Description: "Concentration checks after damage, by outcome."})
	deaths := u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricDeaths, Unit: "{creature}",
		Description: "Creatures that died, by cause."})
	for _, ev := range outcomeEvents(res) {
		if c, ok := ev.(ConcentrationChecked); ok {
			checks.Add(ctx, 1, telemetry.String(attrConcentrationOutcome, concentrationOutcomeName(c.outcome)))
		}
		if h, ok := ev.(HitPointEventOccurred); ok {
			if d, ok := h.wrapped.(Died); ok {
				deaths.Add(ctx, 1, telemetry.String(attrDeathCause, d.cause.String()))
			}
		}
	}
	return res, nil
}

// resolutionCompleted is the completed half of every result: the new encounter and the events in the order
// they happened.
type resolutionCompleted struct {
	enc    Encounter
	events []Event
}

// Encounter returns the new encounter.
func (c resolutionCompleted) Encounter() Encounter { return c.enc }

// Events returns a copy of the events, in the order they happened.
func (c resolutionCompleted) Events() []Event { return slices.Clone(c.events) }

func (c resolutionCompleted) describe() description {
	return description{Class: resultApplied, Name: valueResultCompleted, Events: c.events}
}

// resolutionRejected is the refused half of every result: the rule that stopped the call and the encounter
// that came in.
type resolutionRejected struct {
	rejection ResolutionRejection
	enc       Encounter
}

// Rejection returns why the call stopped.
func (r resolutionRejected) Rejection() ResolutionRejection { return r.rejection }

// Encounter returns the encounter that came in, unchanged.
func (r resolutionRejected) Encounter() Encounter { return r.enc }

// Events returns no event: a refused call produces none.
func (resolutionRejected) Events() []Event { return []Event{} }

func (r resolutionRejected) describe() description {
	return description{Class: resultRefused, Name: r.rejection.String()}
}

// resolutionInvalid is the invalid half of every result: the problem with the host's input and the
// encounter that came in.
type resolutionInvalid struct {
	problem ResolutionProblem
	enc     Encounter
}

// Problem returns what was wrong with the input.
func (r resolutionInvalid) Problem() ResolutionProblem { return r.problem }

// Encounter returns the encounter that came in, unchanged.
func (r resolutionInvalid) Encounter() Encounter { return r.enc }

// Events returns no event: an invalid call produces none.
func (resolutionInvalid) Events() []Event { return []Event{} }

func (r resolutionInvalid) describe() description {
	return description{Class: resultInvalid, Name: r.problem.String()}
}

// ledgerCompleted adds the new ActionLedger to a completed result.
type ledgerCompleted struct {
	resolutionCompleted
	ledger ActionLedger
}

// Ledger returns the new ActionLedger.
func (c ledgerCompleted) Ledger() ActionLedger { return c.ledger }

// ledgerRejected adds the ActionLedger that came in to a refused result.
type ledgerRejected struct {
	resolutionRejected
	ledger ActionLedger
}

// Ledger returns the ActionLedger that came in, unchanged.
func (r ledgerRejected) Ledger() ActionLedger { return r.ledger }

// ledgerInvalid adds the ActionLedger that came in to an invalid result.
type ledgerInvalid struct {
	resolutionInvalid
	ledger ActionLedger
}

// Ledger returns the ActionLedger that came in, unchanged.
func (r ledgerInvalid) Ledger() ActionLedger { return r.ledger }

// stop is the first stop a call found: a rule (rejection) or a problem with the input; at most one is set.
type stop struct {
	rejection ResolutionRejection
	problem   ResolutionProblem
}

// stopped reports whether the call must end.
func (s stop) stopped() bool { return s.rejection != nil || s.problem != nil }

// preconditions applies steps 1 to 3 of the stop order for the ids.
func preconditions(e Encounter, ids ...CombatantID) stop {
	rej, prob := (ResolutionPrecedence{}).Preconditions(e, ids...)
	return stop{rej, prob}
}
