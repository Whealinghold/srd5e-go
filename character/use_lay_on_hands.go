package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// UseLayOnHandsCommand is the input of UseLayOnHands.Run: the Paladin, the Hit Points to restore and the conditions to remove.
// Construct it with a keyed literal. Run does not alter it.
type UseLayOnHandsCommand struct {
	Character Character
	HitPoints int
	Remove    []content.ConditionKind
}

// UseLayOnHandsResult is the closed set of outcomes of UseLayOnHands.Run. Variants: UseLayOnHandsApplied, UseLayOnHandsRejected,
// UseLayOnHandsNotAPaladin.
type UseLayOnHandsResult interface{ isUseLayOnHandsResult() }

// UseLayOnHandsApplied is the outcome of an accepted use. Only Run builds one.
type UseLayOnHandsApplied struct {
	character Character
	outcome   LayOnHandsOutcome
	state     PaladinState
	events    []Event
}

// Character returns the new Character, with the pool reduced.
func (r UseLayOnHandsApplied) Character() Character { return r.character }

// Outcome returns the Hit Points to restore, the conditions to remove and the action cost, for the host to apply.
func (r UseLayOnHandsApplied) Outcome() LayOnHandsOutcome { return r.outcome }

// State returns the new PaladinState.
func (r UseLayOnHandsApplied) State() PaladinState { return r.state }

// Events returns a new slice holding the LayOnHandsUsed event.
func (r UseLayOnHandsApplied) Events() []Event { return slices.Clone(r.events) }

// UseLayOnHandsRejected is the outcome of a use a rule refused: the Character is unchanged and there are no events. Only Run
// builds one.
type UseLayOnHandsRejected struct{ reasons []PaladinReason }

// Reasons returns every reason that holds in the order of class-paladin.md FR-055, as a new slice. It is never empty.
func (r UseLayOnHandsRejected) Reasons() []PaladinReason { return slices.Clone(r.reasons) }

// UseLayOnHandsNotAPaladin is the outcome for a character with no Paladin levels.
type UseLayOnHandsNotAPaladin struct{}

func (UseLayOnHandsApplied) isUseLayOnHandsResult()     {}
func (UseLayOnHandsRejected) isUseLayOnHandsResult()    {}
func (UseLayOnHandsNotAPaladin) isUseLayOnHandsResult() {}

// UseLayOnHands is the use case that uses a Paladin's Lay On Hands (SRD 5.2.1, "Level 1: Lay On Hands", p. 54; "Level 14:
// Restoring Touch", p. 55). It holds no rules logic: LayOnHandsRules decides. Create it with NewUseLayOnHands; the zero value is
// not usable. It is safe for concurrent use only if its telemetry and clock are.
type UseLayOnHands struct{ paladinUse }

// NewUseLayOnHands returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for the zero progression.
func NewUseLayOnHands(p PaladinProgression, opts ...Option) (UseLayOnHands, error) {
	u, err := newPaladinUse("use Lay On Hands", p, opts)
	return UseLayOnHands{u}, err
}

// Run uses Lay On Hands for cmd.Character. Every reason that holds is reported, in the order of FR-055; the pool is unchanged
// for a rejection.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. A refused use is a result.
func (u UseLayOnHands) Run(ctx context.Context, cmd UseLayOnHandsCommand) (UseLayOnHandsResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseLayOnHands, ValueClassPaladin, ValueFeatureLayOnHands)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return UseLayOnHandsNotAPaladin{}, nil
	}
	outcome, reasons := LayOnHandsRules{}.Use(u.progression, level, state.layRemaining, LayOnHandsRequest{HitPoints: cmd.HitPoints, Remove: cmd.Remove})
	if len(reasons) > 0 {
		return UseLayOnHandsRejected{reasons: paladinReject(ctx, run, reasons)}, nil
	}
	state.layRemaining = outcome.remaining
	next, err := paladinStore(run, cmd.Character, state)
	if err != nil {
		return nil, err
	}
	run.succeeded(ctx)
	return UseLayOnHandsApplied{character: next, outcome: outcome, state: state,
		events: []Event{LayOnHandsUsed{id: next.ID(), hitPoints: outcome.hitPoints, conditions: outcome.conditions, remaining: outcome.remaining}}}, nil
}
