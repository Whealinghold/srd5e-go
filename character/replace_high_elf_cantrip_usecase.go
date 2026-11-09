package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ReplaceHighElfCantripCommand is the input of ReplaceHighElfCantrip.Run: the spell
// that replaces the lineage cantrip. Construct it with a keyed literal. Run does not
// alter it.
type ReplaceHighElfCantripCommand struct {
	Character Character
	Traits    SpeciesTraits
	Spell     content.SpellID
}

// ReplaceHighElfCantripResult is the closed set of outcomes of
// ReplaceHighElfCantrip.Run. Variants: ReplaceHighElfCantripSucceeded,
// ReplaceHighElfCantripRejected, ReplaceHighElfCantripContentMissing.
type ReplaceHighElfCantripResult interface{ isReplaceHighElfCantripResult() }

// ReplaceHighElfCantripSucceeded is the outcome of a swap. Only Run builds one.
type ReplaceHighElfCantripSucceeded struct{ change SpeciesTraitsChanged }

// Traits returns the new SpeciesTraits, with the window closed.
func (r ReplaceHighElfCantripSucceeded) Traits() SpeciesTraits { return r.change.traits }

// Event returns the HighElfCantripReplaced event.
func (r ReplaceHighElfCantripSucceeded) Event() HighElfCantripReplaced {
	return r.change.events[0].(HighElfCantripReplaced)
}

// ReplaceHighElfCantripRejected is the outcome of a swap a rule refused. Construct it
// with a keyed literal.
type ReplaceHighElfCantripRejected struct{ Rejection SpeciesTraitRejection }

// ReplaceHighElfCantripContentMissing is the outcome when the source lacks the spell.
// Only Run builds one; the zero value names nothing.
type ReplaceHighElfCantripContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ReplaceHighElfCantripContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ReplaceHighElfCantripSucceeded) isReplaceHighElfCantripResult()      {}
func (ReplaceHighElfCantripRejected) isReplaceHighElfCantripResult()       {}
func (ReplaceHighElfCantripContentMissing) isReplaceHighElfCantripResult() {}

// ReplaceHighElfCantrip is the use case of the High Elf cantrip swap: it looks up the
// replacement spell, asks HighElfCantripSwap and emits the srd5e.character.* signals.
// It holds no rules logic. Create it with NewReplaceHighElfCantrip; the zero value is
// not usable. It is safe for concurrent use only if its source, telemetry and clock are.
type ReplaceHighElfCantrip struct {
	source ContentSource
	sig    operationSignals
}

// NewReplaceHighElfCantrip returns the use case reading content from source. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewReplaceHighElfCantrip(source ContentSource, opts ...Option) (ReplaceHighElfCantrip, error) {
	if source == nil {
		return ReplaceHighElfCantrip{}, invalidInput("replace a high elf cantrip without a content source")
	}
	return ReplaceHighElfCantrip{source: source, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run looks up the spell, whatever the state, then applies HighElfCantripSwap.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or signal, for
// the zero use case, a zero Character or an invalid spell id, and ctx.Err() unwrapped,
// before any signal, when ctx is already cancelled. It returns a *srderr.PortError
// (error type "content_error") with a nil result when the source fails, panics or breaks
// its contract; the failure is recorded once and never logged. A missing spell is a
// ReplaceHighElfCantripContentMissing result; a refused swap is a
// ReplaceHighElfCantripRejected result.
func (u ReplaceHighElfCantrip) Run(ctx context.Context, cmd ReplaceHighElfCantripCommand) (ReplaceHighElfCantripResult, error) {
	if u.source == nil || cmd.Character.IsZero() || !cmd.Spell.Valid() {
		return nil, invalidInput("replace a high elf cantrip: zero use case or character, or an invalid spell id")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanReplaceHighElfCantrip, ValueOperationReplaceHighElfCantrip)
	defer run.end(ctx)
	found, missing, err := lookupFeatEntries(ctx, u.source, u.sig.tel.Tracer, []content.Key{cmd.Spell.Key()})
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return ReplaceHighElfCantripContentMissing{missing: missing}, nil
	}
	spell, ok := found[cmd.Spell.Key()].(content.Spell)
	if !ok {
		err := portViolation(content.Port, "lookup", "a spell key answered with another entry type")
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	tr, _ := HighElfCantripSwap{}.Replace(cmd.Character, cmd.Traits, spell) // the character is not zero: no error
	switch t := tr.(type) {
	case SpeciesTraitsChanged:
		run.finish(ctx, ValueResultChanged, "")
		return ReplaceHighElfCantripSucceeded{change: t}, nil
	default:
		rj := t.(SpeciesTraitsRejected).Rejection
		run.finish(ctx, ValueResultRejected, rj.String())
		return ReplaceHighElfCantripRejected{Rejection: rj}, nil
	}
}
