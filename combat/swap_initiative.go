package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
)

// SwapInitiativeCommand is the immutable input of SwapInitiative.Run. Build it with
// NewSwapInitiativeCommand; the zero value is invalid.
type SwapInitiativeCommand struct {
	standings         []InitiativeStanding
	swapper, ally     CombatantID
	effect            character.InitiativeSwap
	swapperAv, allyAv ConditionActionAvailability
	consent           AllyConsent
}

// NewSwapInitiativeCommand returns the command: the counts as given to
// EstablishInitiativeOrder, the two combatants, the Initiative Swap effect (its partner
// and blocking condition are not re-read), each side's availability and the ally's
// consent. It returns an error matching srderr.ErrInvalidInput for no standing, a blank or
// repeated id among the standings, the same id for swapper and ally, a blank id, or an
// availability or consent that is not defined, and a LimitExceededError for more than 256 standings. The standings are copied.
func NewSwapInitiativeCommand(standings []InitiativeStanding, swapper, ally CombatantID, effect character.InitiativeSwap, swapperAv, allyAv ConditionActionAvailability, consent AllyConsent) (SwapInitiativeCommand, error) {
	if err := overCap("standings", len(standings), maxInputList); err != nil {
		return SwapInitiativeCommand{}, err
	}
	seen := map[CombatantID]bool{}
	for _, s := range standings {
		if s.Combatant.IsZero() || seen[s.Combatant] {
			return SwapInitiativeCommand{}, invalidInput("standings need distinct combatants")
		}
		seen[s.Combatant] = true
	}
	if len(standings) == 0 || swapper.IsZero() || ally.IsZero() || swapper == ally ||
		!validAvailability(swapperAv) || !validAvailability(allyAv) || !consent.Valid() {
		return SwapInitiativeCommand{}, invalidInput("a swap needs two different combatants, availabilities and a consent")
	}
	return SwapInitiativeCommand{slices.Clone(standings), swapper, ally, effect, swapperAv, allyAv, consent}, nil
}

// SwapInitiativeResult is the closed set of outcomes of SwapInitiative.Run. Variants:
// SwapInitiativeApplied, SwapInitiativeRejected.
type SwapInitiativeResult interface{ isSwapInitiativeResult() }

// SwapInitiativeApplied: the counts were exchanged; the standings keep their order.
type SwapInitiativeApplied struct {
	standings []InitiativeStanding
	events    []TurnEvent
}

// Standings returns a copy of the standings after the swap.
func (r SwapInitiativeApplied) Standings() []InitiativeStanding { return slices.Clone(r.standings) }

// Events returns a copy of the events: one InitiativeSwapped.
func (r SwapInitiativeApplied) Events() []TurnEvent { return slices.Clone(r.events) }

// SwapInitiativeRejected: the rules refuse the swap, or a combatant is not in the standings.
type SwapInitiativeRejected struct{ Reason TurnRejection }

func (SwapInitiativeApplied) isSwapInitiativeResult()  {}
func (SwapInitiativeRejected) isSwapInitiativeResult() {}

// SwapInitiative is the use case of the Alert feat's Initiative Swap (SRD 5.2.1, "Alert",
// p. 87). It is pure over its command, runs before EstablishInitiativeOrder and calls no
// other use case; the host renews the feat's uses itself.
type SwapInitiative struct{ turnUse }

// NewSwapInitiative returns the use case.
func NewSwapInitiative(opts ...Option) SwapInitiative { return SwapInitiative{newTurnUse(opts)} }

// Run exchanges the two counts. A swap that creates a tie leaves it to
// EstablishInitiativeOrder. It returns an error matching srderr.ErrInvalidInput, before
// any signal, for the zero command, and ctx.Err() unwrapped for a cancelled ctx.
func (u SwapInitiative) Run(ctx context.Context, cmd SwapInitiativeCommand) (SwapInitiativeResult, error) {
	if len(cmd.standings) == 0 {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationSwapInitiative, spanSwapInitiative},
		func(context.Context) (SwapInitiativeResult, error) {
			out, events, reason := swapStandings(cmd.standings, cmd.swapper, cmd.ally, cmd.swapperAv, cmd.allyAv, cmd.consent)
			if reason != nil {
				return SwapInitiativeRejected{reason}, nil
			}
			return SwapInitiativeApplied{out, events}, nil
		},
		func(r SwapInitiativeResult) description {
			switch v := r.(type) {
			case SwapInitiativeApplied:
				return description{Class: resultApplied, Name: valueResultApplied, Events: widen(v.events)}
			case SwapInitiativeRejected:
				return description{Class: turnRejectionClass(v.Reason), Name: v.Reason.String()}
			default:
				panic(unreachableError("swap result", r))
			}
		})
}
