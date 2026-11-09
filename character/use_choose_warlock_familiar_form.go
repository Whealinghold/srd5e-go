package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ChooseWarlockFamiliarFormCommand is the input of ChooseWarlockFamiliarForm.Run. Construct it with a keyed literal. Run does not alter it.
type ChooseWarlockFamiliarFormCommand struct {
	Character Character
	Form      FamiliarForm
}

// ChooseWarlockFamiliarFormResult is the closed set of outcomes of ChooseWarlockFamiliarForm.Run. Variants:
// ChooseWarlockFamiliarFormApplied, ChooseWarlockFamiliarFormRejected, ChooseWarlockFamiliarFormContentMissing.
type ChooseWarlockFamiliarFormResult interface{ isChooseWarlockFamiliarFormResult() }

// ChooseWarlockFamiliarFormApplied is the outcome of a choice the rules allowed.
type ChooseWarlockFamiliarFormApplied struct {
	character Character
	events    []Event
}

// Character returns the new Character holding the new WarlockPactState.
func (r ChooseWarlockFamiliarFormApplied) Character() Character { return r.character }

// Events returns a new slice holding the FamiliarFormChosen event.
func (r ChooseWarlockFamiliarFormApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseWarlockFamiliarFormRejected is the outcome of a choice a rule refused: the Character is unchanged.
type ChooseWarlockFamiliarFormRejected struct{ Rejection WarlockRejection }

// ChooseWarlockFamiliarFormContentMissing is the outcome when the monster of the form is not in the source.
type ChooseWarlockFamiliarFormContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseWarlockFamiliarFormContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ChooseWarlockFamiliarFormApplied) isChooseWarlockFamiliarFormResult()        {}
func (ChooseWarlockFamiliarFormRejected) isChooseWarlockFamiliarFormResult()       {}
func (ChooseWarlockFamiliarFormContentMissing) isChooseWarlockFamiliarFormResult() {}

// ChooseWarlockFamiliarForm is the use case that chooses the form of the Pact of the Chain familiar (SRD 5.2.1, "Pact of the Chain", p. 74;
// class-warlock.md FR-090, FR-141): it loads the monster of the form and calls SelectFamiliarForm. Create it with
// NewChooseWarlockFamiliarForm; the zero value is not usable.
type ChooseWarlockFamiliarForm struct{ warlockUse }

// NewChooseWarlockFamiliarForm returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// for a nil source.
func NewChooseWarlockFamiliarForm(source ContentSource, opts ...Option) (ChooseWarlockFamiliarForm, error) {
	u, err := newWarlockUse("choose a familiar form", source, true, opts)
	return ChooseWarlockFamiliarForm{u}, err
}

// Run chooses the form. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character,
// and, recorded on the span, for the zero form; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError when the
// source fails. A refusal is a ChooseWarlockFamiliarFormRejected result; a monster not in the source is a ContentMissing result.
func (u ChooseWarlockFamiliarForm) Run(ctx context.Context, cmd ChooseWarlockFamiliarFormCommand) (ChooseWarlockFamiliarFormResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseWarlockFamiliarForm, ValueFeatureFamiliarForm, cmd.Character, false)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return ChooseWarlockFamiliarFormRejected{Rejection: rej}, nil
	}
	if !cmd.Form.id.Valid() {
		err := invalidInput("choose a familiar form with an invalid monster id")
		call.run.commandError(err)
		return nil, err
	}
	entries, missing, err := u.lookupAll(ctx, call, ValueFeatureFamiliarForm, []content.Key{cmd.Form.id.Key()})
	if err != nil {
		return nil, err
	}
	if len(missing) != 0 {
		call.run.contentMissing(ctx)
		return ChooseWarlockFamiliarFormContentMissing{missing: missing}, nil
	}
	monster, ok := entries[0].(content.Monster)
	if !ok {
		return nil, call.wrongEntry(ctx, cmd.Form.id.Key())
	}
	next, changed, rej, err := call.commitPact(ctx, cmd.Character, SelectFamiliarForm(cmd.Character.ID(), call.state, call.pact, cmd.Form, monster))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return ChooseWarlockFamiliarFormRejected{Rejection: rej}, nil
	}
	return ChooseWarlockFamiliarFormApplied{character: next, events: changed.Events()}, nil
}
