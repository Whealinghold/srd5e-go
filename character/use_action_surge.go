package character

import (
	"context"
	"slices"
)

// UseActionSurgeCommand is the input of UseActionSurge.Run: the character, whose
// turn it is and whether Action Surge was already used this turn. Construct it with
// a keyed literal. Run does not alter it.
type UseActionSurgeCommand struct {
	Character Character
	Turn      FighterTurnOwnership
	PerTurn   FighterPerTurnUse
}

// UseActionSurgeResult is the closed set of outcomes of UseActionSurge.Run.
// Variants: ActionSurgeApplied, ActionSurgeNotAvailable.
type UseActionSurgeResult interface{ isUseActionSurgeResult() }

// ActionSurgeApplied is the outcome of a use of Action Surge. Only Run builds one.
type ActionSurgeApplied struct {
	character Character
	events    []Event
}

// Character returns the new Character, with one more Action Surge use expended.
func (r ActionSurgeApplied) Character() Character { return r.character }

// AdditionalAction returns FighterAnyActionButMagic: one additional action that is
// not the Magic action.
func (ActionSurgeApplied) AdditionalAction() FighterAdditionalAction { return FighterAnyActionButMagic }

// Events returns a new slice holding the ActionSurgeUsed event.
func (r ActionSurgeApplied) Events() []Event { return slices.Clone(r.events) }

// ActionSurgeNotAvailable is the outcome of a use a rule refused: the Character is
// unchanged and there are no events. Construct it with a keyed literal.
type ActionSurgeNotAvailable struct{ Reason FighterReason }

func (ActionSurgeApplied) isUseActionSurgeResult()      {}
func (ActionSurgeNotAvailable) isUseActionSurgeResult() {}

// UseActionSurge is the use case that uses a Fighter's Action Surge (SRD 5.2.1,
// "Level 2: Action Surge", p. 48): it checks the use and emits the
// srd5e.character.feature_used signals. It calls no port besides telemetry and
// holds no rules logic. Create it with NewUseActionSurge; the zero value is not
// usable. It is safe for concurrent use only if its telemetry and clock are.
type UseActionSurge struct {
	progression FighterProgression
	sig         featureSignals
	set         bool
}

// NewUseActionSurge returns the use case. Options: WithTelemetry, WithClock; the
// others are ignored. It returns an error matching srderr.ErrInvalidInput for the
// zero progression.
func NewUseActionSurge(progression FighterProgression, opts ...Option) (UseActionSurge, error) {
	if progression.IsZero() {
		return UseActionSurge{}, invalidInput("use Action Surge without a Fighter progression")
	}
	return UseActionSurge{progression: progression, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

// Run uses Action Surge for cmd.Character. The reasons are checked in the order not a
// Fighter, feature not gained, not own turn, already used this turn, no uses
// remaining.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero Character or an invalid Turn or PerTurn, and ctx.Err()
// unwrapped, before any signal, when ctx is already cancelled. It never returns a
// srderr.PortError.
func (u UseActionSurge) Run(ctx context.Context, cmd UseActionSurgeCommand) (UseActionSurgeResult, error) {
	if err := fighterCommandProblem(u.set, cmd.Character); err != nil {
		return nil, err
	}
	if !cmd.Turn.Valid() || !cmd.PerTurn.Valid() {
		return nil, invalidInput("use Action Surge with an invalid turn ownership or per-turn state")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseActionSurge, ValueClassFighter, ValueFeatureActionSurge)
	defer run.end(ctx)
	fc, reason, err := u.progression.contextOf(cmd.Character)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if reason == 0 {
		switch e := (ActionSurge{}).Evaluate(fc.stats, fc.state, cmd.Turn, cmd.PerTurn).(type) {
		case ActionSurgeRefused:
			reason = e.Reason
		case ActionSurgeGranted:
			next, err := WithFeatureState(cmd.Character, FighterStateKey(), e.state)
			if err != nil { // unreachable: the key is valid and the state is read from the same key
				run.commandError(err)
				return nil, err
			}
			run.succeeded(ctx)
			return ActionSurgeApplied{character: next, events: []Event{ActionSurgeUsed{id: next.ID()}}}, nil
		}
	}
	run.rejected(ctx, reason.String())
	return ActionSurgeNotAvailable{Reason: reason}, nil
}
