package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// The three roster use cases add, remove and reposition a combatant of an Encounter after combat
// started (combatant-roster-changes.md section 6). Each wraps one Encounter method, takes its
// <UseCase>Command and returns its own closed <UseCase>Result, ...Applied or ...Rejected, the latter
// holding the unchanged encounter and the RosterRejection. No use case calls another or
// AdvanceCombatTurn; they roll no die, call no port and return no *srderr.PortError. Run returns an
// error matching srderr.ErrInvalidInput, before any signal, for the zero command, and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx; it also returns such an error, recording it
// on the span only, for an encounter that is not valid after the call. A refusal is a Rejected
// result, never an error, and a Rejected result carries the encounter that came in and no event.
// They are safe for concurrent use if their telemetry and clock are.

// rosterUse is what every roster use case holds: the instrumenter for the shared signals. The zero
// value is silent and usable.
type rosterUse struct{ in instrumenter }

func newRosterUse(opts []Option) rosterUse {
	o := newOptions(opts)
	return rosterUse{newInstrumenter(o.tel, o.clock)}
}

// rosterResultApplied is the accepted half of every roster result: the new encounter and the events
// in the order they happened.
type rosterResultApplied struct {
	enc    Encounter
	events []Event
}

// Encounter returns the new encounter.
func (a rosterResultApplied) Encounter() Encounter { return a.enc }

// Events returns a copy of the events, in the order they happened.
func (a rosterResultApplied) Events() []Event { return slices.Clone(a.events) }

// rosterResultRejected is the refused half of every roster result: the rejection and the encounter,
// unchanged. Rejection is the field of every ...Rejected result.
type rosterResultRejected struct {
	Rejection RosterRejection
	enc       Encounter
}

// Encounter returns the encounter, unchanged.
func (r rosterResultRejected) Encounter() Encounter { return r.enc }

// runRoster runs one roster operation through the instrumenter and maps its transition to the use
// case's own result. reason is the value of srd5e.combat.roster_reason, recorded for every call,
// a refused one included. A zero encounter returns an error matching srderr.ErrInvalidInput and a
// cancelled ctx ctx.Err(), both before any signal. The root validates the encounter at entry and exit
// of the call, never inside it (plan P14): an applied transition to an encounter that is not valid
// is an error recorded on the span, with no operation or event counted.
func runRoster[R any](ctx context.Context, u rosterUse, op combatOperation, reason string, enc Encounter,
	work func() (rosterTransition, error), applied func(rosterResultApplied) R, rejected func(rosterResultRejected) R) (R, error) {
	var zero R
	if enc.IsZero() {
		return zero, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	extras := []telemetry.Attr{telemetry.String(attrRosterReason, reason)}
	tr, err := run(ctx, u.in, op, func(context.Context) (rosterTransition, error) {
		tr, err := work()
		if err == nil && tr.Rejection() == nil && tr.Encounter().IsZero() {
			return nil, invalidInput("the encounter is not valid after the call")
		}
		return tr, err
	}, func(t rosterTransition) description {
		if r := t.Rejection(); r != nil {
			return description{Class: rosterRejectionClass(r), Name: r.String(), Extras: extras}
		}
		return description{Class: resultApplied, Name: valueResultApplied, Events: t.Events(), Extras: extras}
	})
	if err != nil {
		return zero, err
	}
	if r := tr.Rejection(); r != nil {
		return rejected(rosterResultRejected{r, tr.Encounter()}), nil
	}
	return applied(rosterResultApplied{tr.Encounter(), tr.Events()}), nil
}

// AddCombatantToEncounterCommand is the immutable input of AddCombatantToEncounter.Run. Build it with
// NewAddCombatantToEncounterCommand; the zero value is invalid.
type AddCombatantToEncounterCommand struct {
	encounter Encounter
	combatant Combatant
	profile   CombatantProfile
	position  RosterPosition
	reason    JoinReason
}

// NewAddCombatantToEncounterCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a zero encounter or Combatant, a profile whose id differs from the combatant's
// or whose size is not defined, a position that is not valid (nil, a blank anchor or a tie placement
// that is not defined) or a reason that is not defined. It does not check that the anchor is a
// combatant: AddCombatantToEncounter.Run does, as RosterUnknownCombatantRejection. It never panics.
func NewAddCombatantToEncounterCommand(encounter Encounter, combatant Combatant, profile CombatantProfile, position RosterPosition, reason JoinReason) (AddCombatantToEncounterCommand, error) {
	if encounter.IsZero() || combatant.IsZero() || combatant.ref.Kind == nil || !combatant.side.Valid() || combatant.hp.IsZero() ||
		profile.ID != combatant.ref.ID || !profile.Size.Valid() || !validRosterPosition(position) || !reason.Valid() {
		return AddCombatantToEncounterCommand{}, invalidInput("an add needs an encounter, a combatant, its profile, a position and a reason")
	}
	return AddCombatantToEncounterCommand{encounter, combatant, profile, position, reason}, nil
}

// AddCombatantToEncounterResult is the closed set of outcomes of AddCombatantToEncounter.Run. Variants:
// AddCombatantToEncounterApplied, AddCombatantToEncounterRejected.
type AddCombatantToEncounterResult interface{ isAddCombatantToEncounterResult() }

// AddCombatantToEncounterApplied: the combatant joined. It holds the new encounter and the one event,
// CombatantJoined.
type AddCombatantToEncounterApplied struct{ rosterResultApplied }

// AddCombatantToEncounterRejected: the roster refused the add; Rejection says why and the encounter is
// unchanged.
type AddCombatantToEncounterRejected struct{ rosterResultRejected }

func (AddCombatantToEncounterApplied) isAddCombatantToEncounterResult()  {}
func (AddCombatantToEncounterRejected) isAddCombatantToEncounterResult() {}

// AddCombatantToEncounter is the use case of adding a combatant to an encounter that has started
// (Encounter.AddCombatant). Create it with NewAddCombatantToEncounter; the zero value is silent and
// works. The host rolls a newcomer's Initiative itself; the engine rolls nothing.
type AddCombatantToEncounter struct{ rosterUse }

// NewAddCombatantToEncounter returns the use case. Options: WithTelemetry, WithClock.
func NewAddCombatantToEncounter(opts ...Option) AddCombatantToEncounter {
	return AddCombatantToEncounter{newRosterUse(opts)}
}

// Run adds the combatant and records the shared signals with srd5e.combat.roster_reason (the JoinReason). It
// returns AddCombatantToEncounterRejected for the refusals of Encounter.AddCombatant (a duplicate id, an
// unknown anchor and a roster over its cap are class invalid, an ended encounter is refused), and the
// errors described for the roster use cases.
func (u AddCombatantToEncounter) Run(ctx context.Context, cmd AddCombatantToEncounterCommand) (AddCombatantToEncounterResult, error) {
	return runRoster(ctx, u.rosterUse, combatOperation{valueOperationAddCombatantToEncounter, spanAddCombatantToEncounter},
		cmd.reason.String(), cmd.encounter,
		func() (rosterTransition, error) {
			return cmd.encounter.AddCombatant(cmd.combatant, cmd.profile, cmd.position, cmd.reason)
		},
		func(a rosterResultApplied) AddCombatantToEncounterResult { return AddCombatantToEncounterApplied{a} },
		func(r rosterResultRejected) AddCombatantToEncounterResult { return AddCombatantToEncounterRejected{r} })
}

// RemoveCombatantFromEncounterCommand is the immutable input of RemoveCombatantFromEncounter.Run. Build
// it with NewRemoveCombatantFromEncounterCommand; the zero value is invalid.
type RemoveCombatantFromEncounterCommand struct {
	encounter Encounter
	ledger    ActionLedger
	combatant CombatantID
	cause     RemovalCause
}

// NewRemoveCombatantFromEncounterCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a zero encounter, a blank id or a cause that is not defined. An id the
// encounter does not hold is the host's input and is reported by Run as RosterUnknownCombatantRejection.
// It never panics.
func NewRemoveCombatantFromEncounterCommand(encounter Encounter, ledger ActionLedger, combatantID CombatantID, cause RemovalCause) (RemoveCombatantFromEncounterCommand, error) {
	if encounter.IsZero() || combatantID.IsZero() || !cause.Valid() {
		return RemoveCombatantFromEncounterCommand{}, invalidInput("a removal needs an encounter, an id and a cause")
	}
	return RemoveCombatantFromEncounterCommand{encounter, ledger, combatantID, cause}, nil
}

// RemoveCombatantFromEncounterResult is the closed set of outcomes of RemoveCombatantFromEncounter.Run.
// Variants: RemoveCombatantFromEncounterApplied, RemoveCombatantFromEncounterRejected.
type RemoveCombatantFromEncounterResult interface{ isRemoveCombatantFromEncounterResult() }

// RemoveCombatantFromEncounterApplied: the combatant left. It holds the new encounter, the grapple endings
// and CombatantRemoved as Events, the new ActionLedger without the leaver's pending Help benefits and
// the HelpBenefitDropped events of those as LedgerEvents, apart from the encounter's events.
type RemoveCombatantFromEncounterApplied struct {
	rosterResultApplied
	ledger       ActionLedger
	ledgerEvents []HelpEvent
}

// Ledger returns the new ActionLedger.
func (r RemoveCombatantFromEncounterApplied) Ledger() ActionLedger { return r.ledger }

// LedgerEvents returns a copy of the HelpBenefitDropped events of the benefits the leaver was in. They are
// not in Events and are not counted by srd5e.combat.events.
func (r RemoveCombatantFromEncounterApplied) LedgerEvents() []HelpEvent {
	return slices.Clone(r.ledgerEvents)
}

// RemoveCombatantFromEncounterRejected: the roster refused the removal; Rejection says why. The encounter and
// the ledger are as they came in.
type RemoveCombatantFromEncounterRejected struct {
	rosterResultRejected
	ledger ActionLedger
}

// Ledger returns the ActionLedger that came in, unchanged.
func (r RemoveCombatantFromEncounterRejected) Ledger() ActionLedger { return r.ledger }

func (RemoveCombatantFromEncounterApplied) isRemoveCombatantFromEncounterResult()  {}
func (RemoveCombatantFromEncounterRejected) isRemoveCombatantFromEncounterResult() {}

// RemoveCombatantFromEncounter is the use case of removing a combatant from an encounter that has started
// (Encounter.RemoveCombatant), with the clean-up of the Help benefits the leaver was in. Create it with
// NewRemoveCombatantFromEncounter; the zero value is silent and works. It does not end the turn of the
// active combatant, which it refuses to remove: the host calls AdvanceCombatTurn first.
type RemoveCombatantFromEncounter struct{ rosterUse }

// NewRemoveCombatantFromEncounter returns the use case. Options: WithTelemetry, WithClock.
func NewRemoveCombatantFromEncounter(opts ...Option) RemoveCombatantFromEncounter {
	return RemoveCombatantFromEncounter{newRosterUse(opts)}
}

// Run removes the combatant and drops its Help benefits from the ledger, and records the shared signals with
// srd5e.combat.roster_reason (the RemovalCause). A refusal returns the ledger it was given. See
// Encounter.RemoveCombatant for the rejections, and the errors described for the roster use cases.
func (u RemoveCombatantFromEncounter) Run(ctx context.Context, cmd RemoveCombatantFromEncounterCommand) (RemoveCombatantFromEncounterResult, error) {
	return runRoster(ctx, u.rosterUse, combatOperation{valueOperationRemoveCombatantFromEncounter, spanRemoveCombatantFromEncounter},
		cmd.cause.String(), cmd.encounter,
		func() (rosterTransition, error) { return cmd.encounter.RemoveCombatant(cmd.combatant, cmd.cause) },
		func(a rosterResultApplied) RemoveCombatantFromEncounterResult {
			drop := cmd.ledger.helps.DropFor(cmd.combatant)
			return RemoveCombatantFromEncounterApplied{a, cmd.ledger.withHelps(drop.benefits), drop.events}
		},
		func(r rosterResultRejected) RemoveCombatantFromEncounterResult {
			return RemoveCombatantFromEncounterRejected{r, cmd.ledger}
		})
}

// RepositionCombatantInEncounterCommand is the immutable input of RepositionCombatantInEncounter.Run. Build
// it with NewRepositionCombatantInEncounterCommand; the zero value is invalid.
type RepositionCombatantInEncounterCommand struct {
	encounter Encounter
	combatant CombatantID
	position  RosterPosition
}

// NewRepositionCombatantInEncounterCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a zero encounter, a blank id, a position that is not valid or a
// RosterAfterCombatant whose anchor is the combatant itself. It never panics.
func NewRepositionCombatantInEncounterCommand(encounter Encounter, combatantID CombatantID, position RosterPosition) (RepositionCombatantInEncounterCommand, error) {
	anchor, anchored := position.(RosterAfterCombatant)
	if encounter.IsZero() || combatantID.IsZero() || !validRosterPosition(position) || (anchored && anchor.Anchor == combatantID) {
		return RepositionCombatantInEncounterCommand{}, invalidInput("a reposition needs an encounter, an id and a position other than after itself")
	}
	return RepositionCombatantInEncounterCommand{encounter, combatantID, position}, nil
}

// RepositionCombatantInEncounterResult is the closed set of outcomes of RepositionCombatantInEncounter.Run.
// Variants: RepositionCombatantInEncounterApplied, RepositionCombatantInEncounterRejected.
type RepositionCombatantInEncounterResult interface {
	isRepositionCombatantInEncounterResult()
}

// RepositionCombatantInEncounterApplied: the combatant took its new place. It holds the new encounter and the
// one event, CombatantRepositioned, produced even when the order did not change.
type RepositionCombatantInEncounterApplied struct{ rosterResultApplied }

// RepositionCombatantInEncounterRejected: the roster refused the move; Rejection says why and the encounter is
// unchanged.
type RepositionCombatantInEncounterRejected struct{ rosterResultRejected }

func (RepositionCombatantInEncounterApplied) isRepositionCombatantInEncounterResult()  {}
func (RepositionCombatantInEncounterRejected) isRepositionCombatantInEncounterResult() {}

// RepositionCombatantInEncounter is the use case of moving a combatant to another place in the
// Initiative order (Encounter.RepositionCombatant), as a controlled mount's Initiative changes to match
// its rider's. Create it with NewRepositionCombatantInEncounter; the zero value is silent and works.
type RepositionCombatantInEncounter struct{ rosterUse }

// NewRepositionCombatantInEncounter returns the use case. Options: WithTelemetry, WithClock.
func NewRepositionCombatantInEncounter(opts ...Option) RepositionCombatantInEncounter {
	return RepositionCombatantInEncounter{newRosterUse(opts)}
}

// Run moves the combatant and records the shared signals with srd5e.combat.roster_reason "repositioned". See
// Encounter.RepositionCombatant for the rejections, and the errors described for the roster use cases.
func (u RepositionCombatantInEncounter) Run(ctx context.Context, cmd RepositionCombatantInEncounterCommand) (RepositionCombatantInEncounterResult, error) {
	return runRoster(ctx, u.rosterUse, combatOperation{valueOperationRepositionCombatantInEncounter, spanRepositionCombatantInEncounter},
		valueRosterReasonRepositioned, cmd.encounter,
		func() (rosterTransition, error) {
			return cmd.encounter.RepositionCombatant(cmd.combatant, cmd.position)
		},
		func(a rosterResultApplied) RepositionCombatantInEncounterResult {
			return RepositionCombatantInEncounterApplied{a}
		},
		func(r rosterResultRejected) RepositionCombatantInEncounterResult {
			return RepositionCombatantInEncounterRejected{r}
		})
}
