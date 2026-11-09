package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ChangeMagicInitiateSpellCommand is the input of ChangeMagicInitiateSpell.Run:
// the character and its feats, the Magic Initiate (named by its spell list), the
// spell replaced and its replacement, and the LevelGained event the change
// answers (the zero event is none). Construct it with a keyed literal. Run does
// not alter it.
type ChangeMagicInitiateSpellCommand struct {
	Character   Character
	Feats       TakenFeats
	List        content.ClassID
	Old         content.SpellID
	New         content.SpellID
	LevelGained LevelGained
}

// ChangeMagicInitiateSpellResult is the closed set of outcomes of
// ChangeMagicInitiateSpell.Run. Variants: ChangeMagicInitiateSpellSucceeded,
// ChangeMagicInitiateSpellRejected, ChangeMagicInitiateSpellContentMissing.
type ChangeMagicInitiateSpellResult interface{ isChangeMagicInitiateSpellResult() }

// ChangeMagicInitiateSpellSucceeded is the outcome of a Spell Change. Only Run
// builds one.
type ChangeMagicInitiateSpellSucceeded struct{ change FeatsChanged }

// Feats returns the new TakenFeats.
func (r ChangeMagicInitiateSpellSucceeded) Feats() TakenFeats { return r.change.feats }

// Event returns the FeatSpellReplaced event.
func (r ChangeMagicInitiateSpellSucceeded) Event() FeatSpellReplaced {
	return r.change.events[0].(FeatSpellReplaced)
}

// ChangeMagicInitiateSpellRejected is the outcome of a change a rule refused.
// Construct it with a keyed literal.
type ChangeMagicInitiateSpellRejected struct{ Rejection FeatRejection }

// ChangeMagicInitiateSpellContentMissing is the outcome when the source lacks the
// replacement spell. Only Run builds one; the zero value names nothing.
type ChangeMagicInitiateSpellContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChangeMagicInitiateSpellContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChangeMagicInitiateSpellSucceeded) isChangeMagicInitiateSpellResult()      {}
func (ChangeMagicInitiateSpellRejected) isChangeMagicInitiateSpellResult()       {}
func (ChangeMagicInitiateSpellContentMissing) isChangeMagicInitiateSpellResult() {}

// ChangeMagicInitiateSpell is the use case of the Spell Change of Magic Initiate:
// it looks up the replacement spell, asks MagicInitiateSpellChange and emits the
// srd5e.character.* signals. It holds no rules logic. Create it with
// NewChangeMagicInitiateSpell; the zero value is not usable. It is safe for
// concurrent use only if its source, telemetry and clock are.
type ChangeMagicInitiateSpell struct {
	source ContentSource
	sig    operationSignals
}

// NewChangeMagicInitiateSpell returns the use case reading content from source.
// Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for a nil source.
func NewChangeMagicInitiateSpell(source ContentSource, opts ...Option) (ChangeMagicInitiateSpell, error) {
	if source == nil {
		return ChangeMagicInitiateSpell{}, invalidInput("change a magic initiate spell without a content source")
	}
	return ChangeMagicInitiateSpell{source: source, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run applies cmd. Only the replacement spell is looked up.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case, a zero Character or an invalid list or spell id,
// and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It
// returns a *srderr.PortError (error type "content_error") with a nil result when
// the source fails, panics or breaks its contract; the failure is recorded once
// and never logged. A refused change is a ChangeMagicInitiateSpellRejected result.
func (u ChangeMagicInitiateSpell) Run(ctx context.Context, cmd ChangeMagicInitiateSpellCommand) (ChangeMagicInitiateSpellResult, error) {
	if u.source == nil || cmd.Character.IsZero() || !cmd.List.Valid() || !cmd.Old.Valid() || !cmd.New.Valid() {
		return nil, invalidInput("change a magic initiate spell: zero use case or character, or an invalid id")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanChangeMagicInitiateSpell, ValueOperationChangeMagicInitiateSpell)
	defer run.end(ctx)
	found, missing, err := lookupFeatEntries(ctx, u.source, u.sig.tel.Tracer, []content.Key{cmd.New.Key()})
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return ChangeMagicInitiateSpellContentMissing{missing: missing}, nil
	}
	spell, ok := found[cmd.New.Key()].(content.Spell)
	if !ok {
		err := portViolation(content.Port, "lookup", "a spell key answered with another entry type")
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	switch t := replaceMagicInitiateSpell(cmd.Character, cmd.Feats, cmd.List, cmd.Old, spell, cmd.LevelGained).(type) {
	case FeatsChanged:
		run.finish(ctx, ValueResultChanged, "")
		return ChangeMagicInitiateSpellSucceeded{change: t}, nil
	default:
		rj := t.(FeatsRejected).Rejection
		run.finish(ctx, ValueResultRejected, rj.String())
		return ChangeMagicInitiateSpellRejected{Rejection: rj}, nil
	}
}
