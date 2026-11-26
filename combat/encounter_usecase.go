package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// encounterUse is what every encounter use case holds: the instrumenter for the shared
// signals and the telemetry for the signals of encounter-aggregate.md section 7. The
// zero value is silent and usable. The encounter calls no port, so it returns no
// *srderr.PortError.
type encounterUse struct {
	in  instrumenter
	tel telemetry.Telemetry
}

func newEncounterUse(opts []Option) encounterUse {
	o := newOptions(opts)
	return encounterUse{in: newInstrumenter(o.tel, o.clock), tel: o.tel}
}

// encounterApplied is the accepted half of every encounter result: the new encounter
// and the events in the order they happened.
type encounterApplied struct {
	enc    Encounter
	events []Event
}

// Encounter returns the new encounter.
func (a encounterApplied) Encounter() Encounter { return a.enc }

// Events returns a copy of the events, in the order they happened.
func (a encounterApplied) Events() []Event { return slices.Clone(a.events) }

// encounterRejected is the refused half of every encounter result: the rejection and
// the encounter, unchanged. Rejection is the field of every ...Rejected result.
type encounterRejected struct {
	Rejection EncounterRejection
	enc       Encounter
}

// Encounter returns the encounter, unchanged.
func (r encounterRejected) Encounter() Encounter { return r.enc }

// transitionOp is one encounter operation: its two names.
func transitionOp(value, span string) combatOperation { return combatOperation{value, span} }

// runTransition runs one Encounter command through the instrumenter and maps the
// transition to the use case's own result: an invalid encounter returns an error
// matching srderr.ErrInvalidInput and a cancelled ctx ctx.Err(), both before any signal.
// A rejection is recorded by its class and name; one srd5e.combat.deaths is counted per
// wrapped Died.
func runTransition[R any](ctx context.Context, u encounterUse, op combatOperation, enc Encounter,
	work func() (encounterTransition, error), applied func(encounterApplied, []DamageResolution) R, rejected func(encounterRejected) R) (R, error) {
	var zero R
	if enc.IsZero() {
		return zero, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	tr, err := run(ctx, u.in, op, func(context.Context) (encounterTransition, error) {
		tr, err := work()
		if err == nil && tr.Rejection() == nil && tr.Encounter().IsZero() {
			return nil, invalidInput("the encounter is not valid after the call")
		}
		return tr, err
	}, func(t encounterTransition) description {
		if r := t.Rejection(); r != nil {
			return description{Class: rejectionClass(r), Name: r.String()}
		}
		return description{Class: resultApplied, Name: valueResultApplied, Events: t.Events()}
	})
	if err != nil {
		return zero, err
	}
	if r := tr.Rejection(); r != nil {
		return rejected(encounterRejected{r, tr.Encounter()}), nil
	}
	deaths := u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricDeaths, Unit: "{creature}",
		Description: "Creatures that died, by cause."})
	for _, ev := range tr.Events() {
		if h, ok := ev.(HitPointEventOccurred); ok {
			if d, ok := h.wrapped.(Died); ok {
				deaths.Add(ctx, 1, telemetry.String(attrDeathCause, d.cause.String()))
			}
		}
	}
	return applied(encounterApplied{tr.Encounter(), tr.Events()}, tr.Resolutions()), nil
}

// StartEncounterCommand is the immutable input of StartEncounter.Run. Build it with
// NewStartEncounterCommand; the zero value is invalid.
type StartEncounterCommand struct {
	id          EncounterID
	established InitiativeOrderEstablished
	combatants  []Combatant
}

// NewStartEncounterCommand returns the command, taking its own copy of combatants. It
// validates nothing: untrusted input is reported by Run as StartEncounterInvalid.
func NewStartEncounterCommand(id EncounterID, established InitiativeOrderEstablished, combatants []Combatant) StartEncounterCommand {
	return StartEncounterCommand{id, established, slices.Clone(combatants)}
}

// StartEncounterResult is the closed set of outcomes of StartEncounter.Run. Variants:
// StartEncounterApplied, StartEncounterInvalid.
type StartEncounterResult interface{ isStartEncounterResult() }

// StartEncounterApplied: the encounter started.
type StartEncounterApplied struct{ encounterApplied }

// StartEncounterInvalid: the input could not start an encounter.
type StartEncounterInvalid struct{ err error }

func (StartEncounterApplied) isStartEncounterResult() {}
func (StartEncounterInvalid) isStartEncounterResult() {}

// Err returns the error Encounter.Start returned: it matches srderr.ErrInvalidInput.
func (r StartEncounterInvalid) Err() error { return r.err }

// Problems returns the EncounterProblems of the failed start, in the order found.
func (r StartEncounterInvalid) Problems() []EncounterProblem { return ProblemsOf(r.err) }

// StartEncounter is the use case of starting an encounter (Encounter.Start). Create it
// with NewStartEncounter; the zero value is silent and works.
type StartEncounter struct{ encounterUse }

// NewStartEncounter returns the use case. Options: WithTelemetry, WithClock.
func NewStartEncounter(opts ...Option) StartEncounter { return StartEncounter{newEncounterUse(opts)} }

// Run starts the encounter and records the shared signals and, for a valid start,
// srd5e.combat.encounter_size. It returns StartEncounterInvalid (class invalid, result
// invalid_encounter) for input Encounter.Start rejects. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero command (no established
// order), and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
func (u StartEncounter) Run(ctx context.Context, cmd StartEncounterCommand) (StartEncounterResult, error) {
	if cmd.established.tracker.IsZero() {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, transitionOp(valueOperationStartEncounter, spanStartEncounter),
		func(context.Context) (StartEncounterResult, error) {
			enc, events, err := (Encounter{}).Start(cmd.id, cmd.established, cmd.combatants)
			if err != nil {
				return StartEncounterInvalid{err}, nil
			}
			return StartEncounterApplied{encounterApplied{enc, events}}, nil
		}, func(r StartEncounterResult) description {
			if a, ok := r.(StartEncounterApplied); ok {
				return description{Class: resultApplied, Name: valueResultApplied, Events: a.events}
			}
			if inv, ok := r.(StartEncounterInvalid); ok && slices.Contains(inv.Problems(), EncounterProblem(CombatantLimitProblem{})) {
				return description{Class: resultInvalid, Name: valueResultLimitExceeded}
			}
			return description{Class: resultInvalid, Name: valueResultInvalidEncounter}
		})
	if err != nil {
		return nil, err
	}
	if _, ok := res.(StartEncounterApplied); ok {
		u.tel.WithDefaults().Meter.Histogram(telemetry.Descriptor{Name: metricEncounterSize, Unit: "{combatant}",
			Description: "Combatants in an encounter at start."}).Record(ctx, float64(len(cmd.combatants)))
	}
	return res, nil
}
