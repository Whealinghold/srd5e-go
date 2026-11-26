package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// AdvanceCombatTurnCommand is the immutable input of AdvanceCombatTurn.Run. Build it with
// NewAdvanceCombatTurnCommand; the zero value is invalid.
type AdvanceCombatTurnCommand struct {
	encounter Encounter
	ledger    ActionLedger
	updates   []SpeedUpdate
	occupants []content.CreatureSize
}

// NewAdvanceCombatTurnCommand returns the command, taking its own copy of the lists. updatedSpeeds replace the
// profile Speeds of combatants, in the order given; sharedSpaceOccupants are the sizes of the other creatures in
// the ending combatant's space, none when it is alone. It returns an error matching srderr.ErrInvalidInput for a
// zero encounter, a blank id in an update or a size that is not defined. An id the encounter does not hold is
// the host's input and is reported by Run as AdvanceCombatTurnInvalid. It never panics.
func NewAdvanceCombatTurnCommand(encounter Encounter, ledger ActionLedger, updatedSpeeds []SpeedUpdate, sharedSpaceOccupants []content.CreatureSize) (AdvanceCombatTurnCommand, error) {
	switch {
	case encounter.IsZero():
		return AdvanceCombatTurnCommand{}, invalidInput("an advance needs an encounter")
	case slices.ContainsFunc(updatedSpeeds, func(u SpeedUpdate) bool { return u.Combatant.IsZero() }):
		return AdvanceCombatTurnCommand{}, invalidInput("a speed update needs a combatant")
	case slices.ContainsFunc(sharedSpaceOccupants, func(s content.CreatureSize) bool { return !s.Valid() }):
		return AdvanceCombatTurnCommand{}, invalidInput("a size is not defined")
	}
	return AdvanceCombatTurnCommand{encounter, ledger, slices.Clone(updatedSpeeds), slices.Clone(sharedSpaceOccupants)}, nil
}

// AdvanceCombatTurnResult is the closed set of outcomes of AdvanceCombatTurn.Run. Variants:
// AdvanceCombatTurnCompleted, AdvanceCombatTurnRejected, AdvanceCombatTurnInvalid.
type AdvanceCombatTurnResult interface{ isAdvanceCombatTurnResult() }

// AdvanceCombatTurnCompleted: the turn advanced. It holds the new encounter and ledger and the events in order.
type AdvanceCombatTurnCompleted struct{ ledgerCompleted }

// AdvanceCombatTurnRejected: the encounter ended; the encounter and ledger are as they came in.
type AdvanceCombatTurnRejected struct{ ledgerRejected }

// AdvanceCombatTurnInvalid: an update named a combatant the encounter does not hold; the encounter and ledger
// are as they came in.
type AdvanceCombatTurnInvalid struct{ ledgerInvalid }

func (AdvanceCombatTurnCompleted) isAdvanceCombatTurnResult() {}
func (AdvanceCombatTurnRejected) isAdvanceCombatTurnResult()  {}
func (AdvanceCombatTurnInvalid) isAdvanceCombatTurnResult()   {}

// AdvanceCombatTurn is the use case of ending the active combatant's turn and starting the next (SRD 5.2.1, "The
// Order of Combat", p. 13). It plays the role of AdvanceEncounterTurn and adds what that bare transition does not:
// Prone for a creature that fell in a shared space, the end of the Attack action, the expiry of Help benefits
// and the prompts of a changed condition (combat-action-resolution.md FR-126). A dead combatant's turn is not
// skipped, and it does not roll the Death Saving Throw of a Dying combatant: DyingCombatantTurnStarted is the
// host's prompt. Create it with NewAdvanceCombatTurn; the zero value is silent and works. It rolls no die.
type AdvanceCombatTurn struct{ resolutionUse }

// NewAdvanceCombatTurn returns the use case. Options: WithTelemetry, WithClock.
func NewAdvanceCombatTurn(opts ...Option) AdvanceCombatTurn {
	return AdvanceCombatTurn{newResolutionUse(nil, opts)}
}

// Run advances the turn. It returns AdvanceCombatTurnRejected for an ended encounter and
// AdvanceCombatTurnInvalid naming the smallest unknown id of the updates. The encounter's events come first,
// then the ConditionApplied of each Prone from a shared space (an existing Prone, an Immunity or a dead combatant
// does not stop the call and gives no event), then HelpBenefitExpired for each benefit the new active combatant
// gave and did not use, then the prompts. It returns the errors described for the package.
func (u AdvanceCombatTurn) Run(ctx context.Context, cmd AdvanceCombatTurnCommand) (AdvanceCombatTurnResult, error) {
	op := combatOperation{valueOperationAdvanceCombatTurn, spanAdvanceCombatTurn}
	if cmd.encounter.IsZero() {
		return nil, invalidInput("zero command")
	}
	return execute(ctx, u.resolutionUse, op, func() (AdvanceCombatTurnResult, error) { return u.advance(cmd) })
}

// advance is the work of Run.
func (u AdvanceCombatTurn) advance(cmd AdvanceCombatTurnCommand) (AdvanceCombatTurnResult, error) {
	if s := preconditions(cmd.encounter); s.stopped() {
		return AdvanceCombatTurnRejected{ledgerRejected{resolutionRejected{s.rejection, cmd.encounter}, cmd.ledger}}, nil
	}
	tr, err := cmd.encounter.AdvanceTurn(TurnAdvance{Updates: cmd.updates, SharedSpaceOccupants: cmd.occupants})
	if err != nil {
		return nil, err
	}
	if rej := tr.Rejection(); rej != nil {
		unknown, ok := rej.(UnknownCombatantRejection)
		if !ok {
			return AdvanceCombatTurnRejected{ledgerRejected{resolutionRejected{EncounterStateRejection{rej}, cmd.encounter}, cmd.ledger}}, nil
		}
		return AdvanceCombatTurnInvalid{ledgerInvalid{resolutionInvalid{UnknownCombatantProblem(unknown), cmd.encounter}, cmd.ledger}}, nil
	}
	r := &resolution{enc: cmd.encounter}
	r.mustCommit(tr)
	for _, ev := range tr.Events() {
		t, ok := ev.(TurnEventOccurred)
		if !ok {
			continue
		}
		if fell, ok := t.Event().(FellProneInSharedSpace); ok {
			r.fallProne(fell.combatant)
		}
	}
	ledger, expired := cmd.ledger.ended(r.enc.tracker.order[r.enc.tracker.active].Combatant)
	r.add(expired...)
	r.prompts(cmd.encounter)
	// No events cap check: the events of this call are bounded by its per-call lists (plan P10), far under 4096.
	if r.exitInvalid() {
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return AdvanceCombatTurnCompleted{ledgerCompleted{resolutionCompleted{r.enc, r.events}, ledger}}, nil
}
