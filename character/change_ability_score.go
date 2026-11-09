package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// ChangeAbilityScoreCommand is the input of ChangeAbilityScore.Run. Construct it
// with a keyed literal. MaximumScore is the cap a feature allows, 20 to 30; 0
// means the default of 20. Run does not alter it.
type ChangeAbilityScoreCommand struct {
	Character    Character
	Ability      ability.Ability
	Score        ability.Score
	MaximumScore int
}

// ChangeAbilityScoreResult is the closed set of outcomes of
// ChangeAbilityScore.Run. Variants: ChangeAbilityScoreSucceeded,
// ChangeAbilityScoreRejected.
type ChangeAbilityScoreResult interface{ isChangeAbilityScoreResult() }

// ChangeAbilityScoreSucceeded is the outcome of a change that took effect, a
// no-op included: the new Character and the events. Only Run builds one.
type ChangeAbilityScoreSucceeded struct {
	character Character
	events    []Event
}

// Character returns the Character after the change.
func (r ChangeAbilityScoreSucceeded) Character() Character { return r.character }

// Events returns a new slice of the events, in order.
func (r ChangeAbilityScoreSucceeded) Events() []Event { return slices.Clone(r.events) }

// ChangeAbilityScoreRejected is the outcome of a change a rule refused.
type ChangeAbilityScoreRejected struct{ Rejection AbilityScoreChangeRejection }

func (ChangeAbilityScoreSucceeded) isChangeAbilityScoreResult() {}
func (ChangeAbilityScoreRejected) isChangeAbilityScoreResult()  {}

// ChangeAbilityScore is the use case that sets one ability score of a Character
// and emits the srd5e.character.* signals. It needs no port beyond telemetry.
// Create it with NewChangeAbilityScore; the zero value is not usable. It is safe
// for concurrent use only if its telemetry and clock are.
type ChangeAbilityScore struct {
	sig operationSignals
	set bool
}

// NewChangeAbilityScore returns the use case. Options: WithTelemetry,
// WithClock; the others are ignored.
func NewChangeAbilityScore(opts ...Option) ChangeAbilityScore {
	return ChangeAbilityScore{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run applies cmd. It returns an error matching srderr.ErrInvalidInput, before
// any signal, for the zero use case or an invalid command (see
// Character.ChangeAbilityScore), and ctx.Err() unwrapped, before any signal,
// when ctx is already cancelled. A rule refusal is a ChangeAbilityScoreRejected
// result, recorded as a failure without error status; it never returns
// srderr.PortError.
func (u ChangeAbilityScore) Run(ctx context.Context, cmd ChangeAbilityScoreCommand) (ChangeAbilityScoreResult, error) {
	if !u.set {
		return nil, invalidInput("zero ChangeAbilityScore; use NewChangeAbilityScore")
	}
	limit := cmd.MaximumScore
	if limit == 0 {
		limit = DefaultScoreCap
	}
	tr, err := cmd.Character.ChangeAbilityScore(cmd.Ability, cmd.Score, limit)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanChangeAbilityScore, ValueOperationChangeAbilityScore)
	defer run.end(ctx)
	switch t := tr.(type) {
	case TransitionChanged[AbilityScoreChangeRejection]:
		run.finish(ctx, ValueResultChanged, "")
		return ChangeAbilityScoreSucceeded(t), nil
	case TransitionRejected[AbilityScoreChangeRejection]:
		run.finish(ctx, ValueResultRejected, abilityRejectionValue(t.rejection))
		return ChangeAbilityScoreRejected{Rejection: t.rejection}, nil
	default:
		panic(invalidInput("unknown transition")) // unreachable: Transition is closed
	}
}

// abilityRejectionValue maps a rejection to its ValueRejection* value.
func abilityRejectionValue(r AbilityScoreChangeRejection) string {
	if _, ok := r.(AbilityScoreChangeExceedsMaximum); ok {
		return ValueRejectionExceedsMaximumScore
	}
	return ValueRejectionConstitutionModifierDecrease
}
