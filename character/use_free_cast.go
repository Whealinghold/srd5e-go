package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// UseFreeCastCommand is the input of UseFreeCast.Run: the Paladin and the free cast to spend. Construct it with a keyed literal.
// Run does not alter it.
type UseFreeCastCommand struct {
	Character Character
	Feature   FreeCastFeature
}

// UseFreeCastResult is the closed set of outcomes of UseFreeCast.Run. Variants: UseFreeCastApplied, UseFreeCastRejected,
// UseFreeCastNotAPaladin.
type UseFreeCastResult interface{ isUseFreeCastResult() }

// UseFreeCastApplied is the outcome of an accepted use. Only Run builds one.
type UseFreeCastApplied struct {
	character Character
	spell     content.SpellID
	state     PaladinState
	events    []Event
}

// Character returns the new Character, with the free cast spent.
func (r UseFreeCastApplied) Character() Character { return r.character }

// Spell returns the spell the host now casts with CastModeFeatureGranted, spending no spell slot.
func (r UseFreeCastApplied) Spell() content.SpellID { return r.spell }

// State returns the new PaladinState.
func (r UseFreeCastApplied) State() PaladinState { return r.state }

// Events returns a new slice holding the FreeCastSpent event.
func (r UseFreeCastApplied) Events() []Event { return slices.Clone(r.events) }

// UseFreeCastRejected is the outcome of a use a rule refused: the Character is unchanged and there are no events. Only Run builds
// one.
type UseFreeCastRejected struct{ reasons []PaladinReason }

// Reasons returns every reason that holds in declaration order, as a new slice. It is never empty.
func (r UseFreeCastRejected) Reasons() []PaladinReason { return slices.Clone(r.reasons) }

// UseFreeCastNotAPaladin is the outcome for a character with no Paladin levels.
type UseFreeCastNotAPaladin struct{}

func (UseFreeCastApplied) isUseFreeCastResult()     {}
func (UseFreeCastRejected) isUseFreeCastResult()    {}
func (UseFreeCastNotAPaladin) isUseFreeCastResult() {}

// UseFreeCast is the use case that spends a Paladin's free cast of Divine Smite (Paladin’s Smite) or Find Steed (Faithful Steed)
// (SRD 5.2.1, "Level 2: Paladin’s Smite", p. 54; "Level 5: Faithful Steed", p. 55). It only tracks availability and returns the
// spell id; the cast is the host's. Create it with NewUseFreeCast; the zero value is not usable. It is safe for concurrent use
// only if its telemetry and clock are.
type UseFreeCast struct{ paladinUse }

// NewUseFreeCast returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for the zero progression.
func NewUseFreeCast(p PaladinProgression, opts ...Option) (UseFreeCast, error) {
	u, err := newPaladinUse("use a Paladin free cast", p, opts)
	return UseFreeCast{u}, err
}

// Run spends the free cast of cmd.Feature. PaladinFeatureNotGained is reported below the level of its feature entry and
// PaladinAlreadyUsedSinceLongRest when the use is spent; both are reported when both hold.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or an invalid
// feature, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. A refused use is a result.
func (u UseFreeCast) Run(ctx context.Context, cmd UseFreeCastCommand) (UseFreeCastResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	if !cmd.Feature.Valid() {
		return nil, invalidInput("use a Paladin free cast: undefined feature")
	}
	ctx, run := u.sig.start(ctx, SpanUseFreeCast, ValueClassPaladin, cmd.Feature.String())
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return UseFreeCastNotAPaladin{}, nil
	}
	var reasons []PaladinReason
	if !u.progression.has(cmd.Feature.paladinFeature(), level) {
		reasons = append(reasons, PaladinFeatureNotGained)
	}
	if state.FreeCast(cmd.Feature) == LongRestUseSpent {
		reasons = append(reasons, PaladinAlreadyUsedSinceLongRest)
	}
	if len(reasons) > 0 {
		return UseFreeCastRejected{reasons: paladinReject(ctx, run, reasons)}, nil
	}
	if cmd.Feature == FreeCastFaithfulSteed {
		state.steed = LongRestUseSpent
	} else {
		state.smite = LongRestUseSpent
	}
	next, err := paladinStore(run, cmd.Character, state)
	if err != nil {
		return nil, err
	}
	spell := u.progression.FreeCastSpell(cmd.Feature)
	run.succeeded(ctx)
	return UseFreeCastApplied{character: next, spell: spell, state: state,
		events: []Event{FreeCastSpent{id: next.ID(), feature: cmd.Feature, spell: spell}}}, nil
}
