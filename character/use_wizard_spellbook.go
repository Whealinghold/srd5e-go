package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// The five spellbook use cases share the accepted outcome SpellbookChanged and each have
// their own rejected, content-missing and not-a-Wizard outcomes (a use case that loads
// nothing has no content-missing outcome).

// ---- StartSpellbook -----------------------------------------------------

// StartSpellbookCommand is the input of StartSpellbook.Run: six level 1 Wizard spells.
// Construct it with a keyed literal.
type StartSpellbookCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Spells      []WizardSpellFacts
}

// StartSpellbookResult is the closed set of outcomes of StartSpellbook.Run. Variants:
// SpellbookChanged, StartSpellbookRejected, StartSpellbookContentMissing, StartSpellbookNotAWizard.
type StartSpellbookResult interface{ isStartSpellbookResult() }

// StartSpellbookRejected is the outcome of a choice a rule refused.
type StartSpellbookRejected struct{ Rejection SpellbookRejection }

// StartSpellbookContentMissing is the outcome when the wizard class is not in the source.
type StartSpellbookContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r StartSpellbookContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// StartSpellbookNotAWizard is the outcome for a character with no Wizard levels.
type StartSpellbookNotAWizard struct{}

func (SpellbookChanged) isStartSpellbookResult()             {}
func (StartSpellbookRejected) isStartSpellbookResult()       {}
func (StartSpellbookContentMissing) isStartSpellbookResult() {}
func (StartSpellbookNotAWizard) isStartSpellbookResult()     {}

// StartSpellbook is the use case that builds a Wizard's starting spellbook (SRD 5.2.1, "Level
// 1: Spellcasting", Spellbook, p. 78). Create it with NewStartSpellbook.
type StartSpellbook struct{ wizardUse }

// NewStartSpellbook returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewStartSpellbook(source ContentSource, opts ...Option) (StartSpellbook, error) {
	if source == nil {
		return StartSpellbook{}, invalidInput("start a spellbook without a content source")
	}
	return StartSpellbook{newWizardUse(source, opts)}, nil
}

// Run builds the spellbook, with the errors of ChooseWizardCantrips.Run.
func (u StartSpellbook) Run(ctx context.Context, cmd StartSpellbookCommand) (StartSpellbookResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spells...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanStartSpellbook, ValueFeatureSpellbookStart, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return StartSpellbookNotAWizard{}, nil
		}
		return StartSpellbookContentMissing{missing: wizardMissingContent()}, nil
	}
	changed, rej := SpellbookRules{}.Start(cmd.CharacterID, cmd.Spells)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return StartSpellbookRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return changed, nil
}

// ---- AddLevelUpSpells ---------------------------------------------------

// AddLevelUpSpellsCommand is the input of AddLevelUpSpells.Run: the book and the two spells added
// on gaining the current Wizard level. Construct it with a keyed literal.
type AddLevelUpSpellsCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Spellbook   Spellbook
	Spells      []WizardSpellFacts
}

// AddLevelUpSpellsResult is the closed set of outcomes of AddLevelUpSpells.Run. Variants:
// SpellbookChanged, AddLevelUpSpellsRejected, AddLevelUpSpellsContentMissing, AddLevelUpSpellsNotAWizard.
type AddLevelUpSpellsResult interface{ isAddLevelUpSpellsResult() }

// AddLevelUpSpellsRejected is the outcome of a choice a rule refused.
type AddLevelUpSpellsRejected struct{ Rejection SpellbookRejection }

// AddLevelUpSpellsContentMissing is the outcome when the wizard class is not in the source.
type AddLevelUpSpellsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r AddLevelUpSpellsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// AddLevelUpSpellsNotAWizard is the outcome for a character with no Wizard levels.
type AddLevelUpSpellsNotAWizard struct{}

func (SpellbookChanged) isAddLevelUpSpellsResult()               {}
func (AddLevelUpSpellsRejected) isAddLevelUpSpellsResult()       {}
func (AddLevelUpSpellsContentMissing) isAddLevelUpSpellsResult() {}
func (AddLevelUpSpellsNotAWizard) isAddLevelUpSpellsResult()     {}

// AddLevelUpSpells is the use case that adds the two spells a new Wizard level owes (SRD 5.2.1,
// "Level 1: Spellcasting", Spellbook, p. 78). Create it with NewAddLevelUpSpells.
type AddLevelUpSpells struct{ wizardUse }

// NewAddLevelUpSpells returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewAddLevelUpSpells(source ContentSource, opts ...Option) (AddLevelUpSpells, error) {
	if source == nil {
		return AddLevelUpSpells{}, invalidInput("add level-up spells without a content source")
	}
	return AddLevelUpSpells{newWizardUse(source, opts)}, nil
}

// Run adds the spells, with the errors of ChooseWizardCantrips.Run.
func (u AddLevelUpSpells) Run(ctx context.Context, cmd AddLevelUpSpellsCommand) (AddLevelUpSpellsResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spells...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanAddLevelUpSpells, ValueFeatureSpellbookLevelUp, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return AddLevelUpSpellsNotAWizard{}, nil
		}
		return AddLevelUpSpellsContentMissing{missing: wizardMissingContent()}, nil
	}
	highest, err := WizardRules{}.HighestSpellLevel(call.def, call.level)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	changed, rej := SpellbookRules{}.AddLevelUp(cmd.CharacterID, call.level, cmd.Spellbook, cmd.Spells, highest)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return AddLevelUpSpellsRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return changed, nil
}

// ---- CopySpellIntoBook --------------------------------------------------

// CopySpellIntoBookCommand is the input of CopySpellIntoBook.Run. Construct it with a keyed literal.
type CopySpellIntoBookCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Spellbook   Spellbook
	Spell       WizardSpellFacts
}

// CopySpellIntoBookResult is the closed set of outcomes of CopySpellIntoBook.Run. Variants:
// SpellbookChanged (with a cost), CopySpellIntoBookRejected (no cost), CopySpellIntoBookContentMissing,
// CopySpellIntoBookNotAWizard.
type CopySpellIntoBookResult interface{ isCopySpellIntoBookResult() }

// CopySpellIntoBookRejected is the outcome of a copy a rule refused; it carries no cost.
type CopySpellIntoBookRejected struct{ Rejection SpellbookRejection }

// CopySpellIntoBookContentMissing is the outcome when the wizard class is not in the source.
type CopySpellIntoBookContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r CopySpellIntoBookContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// CopySpellIntoBookNotAWizard is the outcome for a character with no Wizard levels.
type CopySpellIntoBookNotAWizard struct{}

func (SpellbookChanged) isCopySpellIntoBookResult()                {}
func (CopySpellIntoBookRejected) isCopySpellIntoBookResult()       {}
func (CopySpellIntoBookContentMissing) isCopySpellIntoBookResult() {}
func (CopySpellIntoBookNotAWizard) isCopySpellIntoBookResult()     {}

// CopySpellIntoBook is the use case that copies a found Wizard spell into the spellbook (SRD
// 5.2.1, "Expanding and Replacing a Spellbook", p. 78). Create it with NewCopySpellIntoBook.
type CopySpellIntoBook struct{ wizardUse }

// NewCopySpellIntoBook returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewCopySpellIntoBook(source ContentSource, opts ...Option) (CopySpellIntoBook, error) {
	if source == nil {
		return CopySpellIntoBook{}, invalidInput("copy a spell into a book without a content source")
	}
	return CopySpellIntoBook{newWizardUse(source, opts)}, nil
}

// Run copies the spell, with the errors of ChooseWizardCantrips.Run.
func (u CopySpellIntoBook) Run(ctx context.Context, cmd CopySpellIntoBookCommand) (CopySpellIntoBookResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spell); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanCopySpellIntoBook, ValueFeatureSpellbookCopyIn, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return CopySpellIntoBookNotAWizard{}, nil
		}
		return CopySpellIntoBookContentMissing{missing: wizardMissingContent()}, nil
	}
	highest, err := WizardRules{}.HighestSpellLevel(call.def, call.level)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	changed, rej := SpellbookRules{}.CopyIn(cmd.CharacterID, cmd.Spellbook, cmd.Spell, highest)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return CopySpellIntoBookRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return changed, nil
}

// ---- CopySpellbook ------------------------------------------------------

// CopySpellbookCommand is the input of CopySpellbook.Run: the source book and the spells copied.
// It carries no CharacterID: copying produces no event. Construct it with a keyed literal.
type CopySpellbookCommand struct {
	ClassLevels ClassLevels
	Source      Spellbook
	Spells      []WizardSpellFacts
}

// CopySpellbookResult is the closed set of outcomes of CopySpellbook.Run. Variants:
// SpellbookChanged (with a cost), CopySpellbookRejected, CopySpellbookNotAWizard.
type CopySpellbookResult interface{ isCopySpellbookResult() }

// CopySpellbookRejected is the outcome of a copy a rule refused.
type CopySpellbookRejected struct{ Rejection SpellbookRejection }

// CopySpellbookNotAWizard is the outcome for a character with no Wizard levels.
type CopySpellbookNotAWizard struct{}

func (SpellbookChanged) isCopySpellbookResult()        {}
func (CopySpellbookRejected) isCopySpellbookResult()   {}
func (CopySpellbookNotAWizard) isCopySpellbookResult() {}

// CopySpellbook is the pure use case that copies spells out of a spellbook into another book
// (SRD 5.2.1, "Expanding and Replacing a Spellbook", Copying the Book, p. 78). It loads
// nothing. Create it with NewCopySpellbook.
type CopySpellbook struct{ wizardUse }

// NewCopySpellbook returns the use case. Options: WithTelemetry, WithClock.
func NewCopySpellbook(opts ...Option) CopySpellbook { return CopySpellbook{newWizardUse(nil, opts)} }

// Run copies the spells. It returns an error matching srderr.ErrInvalidInput for the zero use
// case or spell facts without a valid id or level, and ctx.Err() for a cancelled ctx.
func (u CopySpellbook) Run(ctx context.Context, cmd CopySpellbookCommand) (CopySpellbookResult, error) {
	if err := wizardCommandProblem(u.set, Identifier{}, false); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spells...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanCopySpellbook, ValueFeatureSpellbookCopyBook, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return CopySpellbookNotAWizard{}, nil
	}
	changed, rej := SpellbookRules{}.CopyOut(cmd.Source, cmd.Spells)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return CopySpellbookRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return changed, nil
}

// ---- ReplaceLostSpellbook -----------------------------------------------

// ReplaceLostSpellbookCommand is the input of ReplaceLostSpellbook.Run: the Wizard spells the
// character has prepared. Construct it with a keyed literal.
type ReplaceLostSpellbookCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Prepared    []WizardSpellFacts
}

// ReplaceLostSpellbookResult is the closed set of outcomes of ReplaceLostSpellbook.Run. Variants:
// SpellbookChanged (with a cost), ReplaceLostSpellbookNotAWizard. The rules refuse nothing.
type ReplaceLostSpellbookResult interface{ isReplaceLostSpellbookResult() }

// ReplaceLostSpellbookNotAWizard is the outcome for a character with no Wizard levels.
type ReplaceLostSpellbookNotAWizard struct{}

func (SpellbookChanged) isReplaceLostSpellbookResult()               {}
func (ReplaceLostSpellbookNotAWizard) isReplaceLostSpellbookResult() {}

// ReplaceLostSpellbook is the pure use case that transcribes the prepared Wizard spells into a
// new spellbook after the old one is lost (SRD 5.2.1, "Expanding and Replacing a Spellbook",
// p. 78). It loads nothing. Create it with NewReplaceLostSpellbook.
type ReplaceLostSpellbook struct{ wizardUse }

// NewReplaceLostSpellbook returns the use case. Options: WithTelemetry, WithClock.
func NewReplaceLostSpellbook(opts ...Option) ReplaceLostSpellbook {
	return ReplaceLostSpellbook{newWizardUse(nil, opts)}
}

// Run transcribes the spells, with the errors of CopySpellbook.Run and a zero CharacterID.
func (u ReplaceLostSpellbook) Run(ctx context.Context, cmd ReplaceLostSpellbookCommand) (ReplaceLostSpellbookResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Prepared...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanReplaceLostSpellbook, ValueFeatureSpellbookReplace, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return ReplaceLostSpellbookNotAWizard{}, nil
	}
	call.run.succeeded(ctx)
	return SpellbookRules{}.ReplaceLost(cmd.CharacterID, cmd.Prepared), nil
}
