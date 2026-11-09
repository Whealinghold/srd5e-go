package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
)

// CreateSpellcastingCommand is the input of CreateSpellcasting.Do. Classes holds
// at most one setup per class; a held spellcasting class with no setup starts
// with empty lists. Spells must hold every spell the setups name. Construct it
// with a keyed literal. Do does not alter it.
type CreateSpellcastingCommand struct {
	Character Identifier
	Levels    ClassLevels
	Classes   []ClassSpellcastingSetup
	Spells    map[content.SpellID]content.Spell
}

// CreateSpellcastingResult is the closed set of outcomes of CreateSpellcasting.Do.
// Variants: CreateSpellcastingCreated, CreateSpellcastingRejected,
// CreateSpellcastingContentMissing.
type CreateSpellcastingResult interface{ isCreateSpellcastingResult() }

// CreateSpellcastingCreated: the state, with full slots and no events.
type CreateSpellcastingCreated struct{ state SpellcastingState }

// State returns the new state.
func (r CreateSpellcastingCreated) State() SpellcastingState { return r.state }

// CreateSpellcastingRejected: a rule refused the lists; every problem is listed.
type CreateSpellcastingRejected struct{ rejection PreparationRejection }

// Rejection returns every problem found, class by class.
func (r CreateSpellcastingRejected) Rejection() PreparationRejection { return r.rejection }

// CreateSpellcastingContentMissing: a spell named is absent from the spells
// supplied; it takes precedence over every rejection.
type CreateSpellcastingContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the absent spells, in the order named, without repeats.
func (r CreateSpellcastingContentMissing) Missing() []content.MissingContent {
	return append([]content.MissingContent(nil), r.missing...)
}

func (CreateSpellcastingCreated) isCreateSpellcastingResult()        {}
func (CreateSpellcastingRejected) isCreateSpellcastingResult()       {}
func (CreateSpellcastingContentMissing) isCreateSpellcastingResult() {}

// CreateSpellcasting is the use case that builds a character's SpellcastingState.
// It holds no rules logic and needs no port beyond telemetry. Create it with
// NewCreateSpellcasting; the zero value is not usable. It is safe for concurrent
// use only if its telemetry and clock are.
type CreateSpellcasting struct {
	sig spellcastingSignals
	set bool
}

// NewCreateSpellcasting returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewCreateSpellcasting(opts ...Option) CreateSpellcasting {
	return CreateSpellcasting{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do builds the state with full slots and no events, checking every list as
// NewClassSpellcasting does and reporting every problem of every class.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case, a zero character id or class levels, or a repeated or invalid
// setup, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns an error matching ErrClassNotHeld, recorded on the span
// only, for a setup or granted cantrip that names a class the levels lack.
func (u CreateSpellcasting) Do(ctx context.Context, cmd CreateSpellcastingCommand) (CreateSpellcastingResult, error) {
	if !u.set || cmd.Character.IsZero() || cmd.Levels.IsZero() {
		return nil, invalidInput("create spellcasting: zero use case, character or class levels")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanCreateSpellcasting, ValueOperationCreateSpellcasting)
	defer run.end(ctx)
	state, missing, problems, err := createState(cmd.Character, cmd.Levels, cmd.Classes, cmd.Spells)
	switch {
	case err != nil:
		run.commandError(err)
		return nil, err
	case len(missing) > 0:
		run.finish(ctx, ValueResultContentMissing, "")
		return CreateSpellcastingContentMissing{missing: missing}, nil
	case len(problems) > 0:
		rej := PreparationRejection{problems: problems}
		run.finish(ctx, ValueResultRejected, rej.First().String())
		return CreateSpellcastingRejected{rejection: rej}, nil
	}
	run.finish(ctx, ValueResultCreated, "")
	return CreateSpellcastingCreated{state: state}, nil
}
