package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ConjureBookOfShadowsCommand is the input of ConjureBookOfShadows.Run. Rest is the rest just finished, the zero RestKind for none;
// Cantrips and Spells are the spells chosen for the book; Prepared are the spells the Warlock already has prepared, known cantrips
// included. Construct it with a keyed literal. Run does not alter it.
type ConjureBookOfShadowsCommand struct {
	Character Character
	Rest      RestKind
	Cantrips  []content.SpellID
	Spells    []content.SpellID
	Prepared  []content.SpellID
}

// ConjureBookOfShadowsResult is the closed set of outcomes of ConjureBookOfShadows.Run. Variants: ConjureBookOfShadowsApplied,
// ConjureBookOfShadowsRejected, ConjureBookOfShadowsContentMissing.
type ConjureBookOfShadowsResult interface{ isConjureBookOfShadowsResult() }

// ConjureBookOfShadowsApplied is the outcome of a conjuring the rules allowed.
type ConjureBookOfShadowsApplied struct {
	character Character
	book      BookOfShadows
	events    []Event
}

// Character returns the new Character holding the new WarlockPactState.
func (r ConjureBookOfShadowsApplied) Character() Character { return r.character }

// Book returns the book conjured.
func (r ConjureBookOfShadowsApplied) Book() BookOfShadows { return r.book }

// Events returns a new slice holding the BookOfShadowsLost event for a book replaced, then the BookOfShadowsConjured event.
func (r ConjureBookOfShadowsApplied) Events() []Event { return slices.Clone(r.events) }

// ConjureBookOfShadowsRejected is the outcome of a conjuring a rule refused: the Character is unchanged.
type ConjureBookOfShadowsRejected struct{ Rejection WarlockRejection }

// ConjureBookOfShadowsContentMissing is the outcome when a chosen spell is not in the source.
type ConjureBookOfShadowsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ConjureBookOfShadowsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ConjureBookOfShadowsApplied) isConjureBookOfShadowsResult()        {}
func (ConjureBookOfShadowsRejected) isConjureBookOfShadowsResult()       {}
func (ConjureBookOfShadowsContentMissing) isConjureBookOfShadowsResult() {}

// ConjureBookOfShadows is the use case that conjures the Book of Shadows of Pact of the Tome (SRD 5.2.1, "Pact of the Tome", p. 74;
// class-warlock.md FR-042, FR-097 to FR-101): it loads the chosen spells and calls ConjureBook. Create it with NewConjureBookOfShadows;
// the zero value is not usable.
type ConjureBookOfShadows struct{ warlockUse }

// NewConjureBookOfShadows returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for
// a nil source.
func NewConjureBookOfShadows(source ContentSource, opts ...Option) (ConjureBookOfShadows, error) {
	u, err := newWarlockUse("conjure a Book of Shadows", source, true, opts)
	return ConjureBookOfShadows{u}, err
}

// Run conjures the book. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character,
// and, recorded on the span, for an invalid spell id; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError
// when the source fails. A refusal is a ConjureBookOfShadowsRejected result; every chosen spell the source lacks is listed in a ContentMissing
// result.
func (u ConjureBookOfShadows) Run(ctx context.Context, cmd ConjureBookOfShadowsCommand) (ConjureBookOfShadowsResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanConjureBookOfShadows, ValueFeatureBookOfShadows, cmd.Character, false)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return ConjureBookOfShadowsRejected{Rejection: rej}, nil
	}
	chosen := slices.Concat(cmd.Cantrips, cmd.Spells)
	keys := make([]content.Key, len(chosen))
	for i, id := range chosen {
		if !id.Valid() {
			err := invalidInput("conjure a Book of Shadows with an invalid spell id")
			call.run.commandError(err)
			return nil, err
		}
		keys[i] = id.Key()
	}
	entries, missing, err := u.lookupAll(ctx, call, ValueFeatureBookOfShadows, keys)
	if err != nil {
		return nil, err
	}
	if len(missing) != 0 {
		call.run.contentMissing(ctx)
		return ConjureBookOfShadowsContentMissing{missing: missing}, nil
	}
	spells := make([]content.Spell, len(entries))
	for i, e := range entries {
		sp, ok := e.(content.Spell)
		if !ok {
			return nil, call.wrongEntry(ctx, keys[i])
		}
		spells[i] = sp
	}
	n := min(len(cmd.Cantrips), len(spells))
	next, changed, rej, err := call.commitPact(ctx, cmd.Character, ConjureBook(cmd.Character.ID(), call.state, call.pact, cmd.Rest, spells[:n], spells[n:], cmd.Prepared))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return ConjureBookOfShadowsRejected{Rejection: rej}, nil
	}
	return ConjureBookOfShadowsApplied{character: next, book: changed.state.book, events: changed.Events()}, nil
}
