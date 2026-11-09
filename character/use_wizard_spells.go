package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ---- UseArcaneRecovery --------------------------------------------------

// UseArcaneRecoveryCommand is the input of UseArcaneRecovery.Run: the state held, the slots
// expended and chosen (slot level to count) and the RestApplied that triggers the feature.
// Construct it with a keyed literal.
type UseArcaneRecoveryCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Resources   WizardResources
	Expended    map[int]int
	Chosen      map[int]int
	Trigger     RestApplied
}

// UseArcaneRecoveryResult is the closed set of outcomes of UseArcaneRecovery.Run. Variants:
// ArcaneRecoveryApplied, UseArcaneRecoveryRejected, UseArcaneRecoveryContentMissing,
// UseArcaneRecoveryNotAWizard.
type UseArcaneRecoveryResult interface{ isUseArcaneRecoveryResult() }

// ArcaneRecoveryApplied is the outcome of a choice the rules allowed. An empty choice is a
// no-op: nothing recovered, no event, and Arcane Recovery stays available.
type ArcaneRecoveryApplied struct {
	recovered ArcaneRecoveryChoice
	resources WizardResources
	events    []Event
}

// Recovered returns the slots recovered.
func (r ArcaneRecoveryApplied) Recovered() ArcaneRecoveryChoice { return r.recovered }

// Resources returns the new WizardResources, with Arcane Recovery marked used unless the choice was empty.
func (r ArcaneRecoveryApplied) Resources() WizardResources { return r.resources }

// Events returns a new slice holding the ArcaneRecoveryUsed event, or none for an empty choice.
func (r ArcaneRecoveryApplied) Events() []Event { return slices.Clone(r.events) }

// UseArcaneRecoveryRejected is the outcome of a choice a rule refused.
type UseArcaneRecoveryRejected struct{ Rejection ArcaneRecoveryRejection }

// UseArcaneRecoveryContentMissing is the outcome when the wizard class is not in the source.
type UseArcaneRecoveryContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r UseArcaneRecoveryContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// UseArcaneRecoveryNotAWizard is the outcome for a character with no Wizard levels.
type UseArcaneRecoveryNotAWizard struct{}

func (ArcaneRecoveryApplied) isUseArcaneRecoveryResult()           {}
func (UseArcaneRecoveryRejected) isUseArcaneRecoveryResult()       {}
func (UseArcaneRecoveryContentMissing) isUseArcaneRecoveryResult() {}
func (UseArcaneRecoveryNotAWizard) isUseArcaneRecoveryResult()     {}

// UseArcaneRecovery is the use case that applies a Wizard's Arcane Recovery after a Short Rest
// (SRD 5.2.1, "Level 1: Arcane Recovery", p. 78). Create it with NewUseArcaneRecovery.
type UseArcaneRecovery struct{ wizardUse }

// NewUseArcaneRecovery returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewUseArcaneRecovery(source ContentSource, opts ...Option) (UseArcaneRecovery, error) {
	if source == nil {
		return UseArcaneRecovery{}, invalidInput("use Arcane Recovery without a content source")
	}
	return UseArcaneRecovery{newWizardUse(source, opts)}, nil
}

// Run applies the choice, with the errors of ChooseWizardSkills.Run.
func (u UseArcaneRecovery) Run(ctx context.Context, cmd UseArcaneRecoveryCommand) (UseArcaneRecoveryResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanUseArcaneRecovery, ValueFeatureArcaneRecovery, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return UseArcaneRecoveryNotAWizard{}, nil
		}
		return UseArcaneRecoveryContentMissing{missing: wizardMissingContent()}, nil
	}
	reject := func(rej ArcaneRecoveryRejection) (UseArcaneRecoveryResult, error) {
		call.run.rejected(ctx, rej.Kind.String())
		return UseArcaneRecoveryRejected{Rejection: rej}, nil
	}
	if !shortRestFinished(cmd.Trigger) {
		return reject(ArcaneRecoveryRejection{Kind: ArcaneRecoveryNotAfterShortRest})
	}
	if cmd.Resources.ArcaneRecoveryUsed() {
		return reject(ArcaneRecoveryRejection{Kind: ArcaneRecoveryAlreadyUsed})
	}
	choice, rej := NewArcaneRecoveryChoice(cmd.Chosen, cmd.Expended, ArcaneRecovery{}.MaxLevels(call.level))
	if rej.Kind != 0 {
		return reject(rej)
	}
	call.run.succeeded(ctx)
	if choice.IsEmpty() {
		return ArcaneRecoveryApplied{resources: cmd.Resources}, nil
	}
	return ArcaneRecoveryApplied{recovered: choice, resources: cmd.Resources.WithArcaneRecoveryUsed(),
		events: []Event{ArcaneRecoveryUsed{id: cmd.CharacterID, levels: choice.Levels()}}}, nil
}

// ---- MemorizeSpell ------------------------------------------------------

// MemorizeSpellCommand is the input of MemorizeSpell.Run. Resources is read only, for the
// always-prepared set; Prepared is the whole prepared list. Construct it with a keyed literal.
type MemorizeSpellCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Resources   WizardResources
	Prepared    []content.SpellID
	Spellbook   Spellbook
	Replace     content.SpellID
	New         WizardSpellFacts
	Trigger     RestApplied
}

// MemorizeSpellResult is the closed set of outcomes of MemorizeSpell.Run. Variants:
// MemorizeApplied, MemorizeSpellRejected, MemorizeSpellContentMissing, MemorizeSpellNotAWizard.
type MemorizeSpellResult interface{ isMemorizeSpellResult() }

// MemorizeApplied is the outcome of a replacement the rules allowed.
type MemorizeApplied struct {
	prepared []content.SpellID
	events   []Event
}

// Prepared returns a new slice of the new prepared list.
func (r MemorizeApplied) Prepared() []content.SpellID { return slices.Clone(r.prepared) }

// Events returns a new slice holding the SpellMemorized event.
func (r MemorizeApplied) Events() []Event { return slices.Clone(r.events) }

// MemorizeSpellRejected is the outcome of a replacement a rule refused.
type MemorizeSpellRejected struct{ Rejection MemorizeRejection }

// MemorizeSpellContentMissing is the outcome when the wizard class is not in the source.
type MemorizeSpellContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r MemorizeSpellContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// MemorizeSpellNotAWizard is the outcome for a character with no Wizard levels.
type MemorizeSpellNotAWizard struct{}

func (MemorizeApplied) isMemorizeSpellResult()             {}
func (MemorizeSpellRejected) isMemorizeSpellResult()       {}
func (MemorizeSpellContentMissing) isMemorizeSpellResult() {}
func (MemorizeSpellNotAWizard) isMemorizeSpellResult()     {}

// MemorizeSpell is the use case that replaces a prepared spell after a Short Rest (SRD 5.2.1,
// "Level 5: Memorize Spell", p. 79). It reuses the prepared-list rules and never calls the
// ChangePreparation function. Create it with NewMemorizeSpell.
type MemorizeSpell struct{ wizardUse }

// NewMemorizeSpell returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewMemorizeSpell(source ContentSource, opts ...Option) (MemorizeSpell, error) {
	if source == nil {
		return MemorizeSpell{}, invalidInput("memorize a spell without a content source")
	}
	return MemorizeSpell{newWizardUse(source, opts)}, nil
}

// Run applies the replacement, with the errors of ChooseWizardCantrips.Run.
func (u MemorizeSpell) Run(ctx context.Context, cmd MemorizeSpellCommand) (MemorizeSpellResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.New); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanMemorizeSpell, ValueFeatureMemorizeSpell, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return MemorizeSpellNotAWizard{}, nil
		}
		return MemorizeSpellContentMissing{missing: wizardMissingContent()}, nil
	}
	highest, err := WizardRules{}.HighestSpellLevel(call.def, call.level)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	gained := wizardFeatureLevel(call.def, "wizard_memorize_spell")
	prepared, rej := checkMemorize(MemorizeRequest{Available: gained > 0 && call.level >= gained, Trigger: cmd.Trigger, Prepared: cmd.Prepared,
		AlwaysPrep: WizardAlwaysPrepared{}.Of(cmd.Resources), Spellbook: cmd.Spellbook, Replace: cmd.Replace, New: cmd.New, HighestLevel: highest})
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return MemorizeSpellRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return MemorizeApplied{prepared: prepared, events: []Event{SpellMemorized{id: cmd.CharacterID, replaced: cmd.Replace, memorized: cmd.New.ID}}}, nil
}

// ---- CheckRitualAdept ---------------------------------------------------

// CheckRitualAdeptCommand is the input of CheckRitualAdept.Run. Construct it with a keyed literal.
type CheckRitualAdeptCommand struct {
	ClassLevels ClassLevels
	Spellbook   Spellbook
	Spell       WizardSpellFacts
	// SpellbookAvailable says whether the spellbook can be read from now.
	SpellbookAvailable bool
}

// CheckRitualAdeptResult is the closed set of outcomes of CheckRitualAdept.Run. Variants:
// RitualAdeptEligible, CheckRitualAdeptRejected, CheckRitualAdeptNotAWizard.
type CheckRitualAdeptResult interface{ isCheckRitualAdeptResult() }

// RitualAdeptEligible is the outcome for a spell the Wizard can cast as a Ritual, prepared or not.
type RitualAdeptEligible struct{}

// CheckRitualAdeptRejected is the outcome for a spell the Wizard cannot cast as a Ritual.
type CheckRitualAdeptRejected struct{ Rejection RitualAdeptRejection }

// CheckRitualAdeptNotAWizard is the outcome for a character with no Wizard levels.
type CheckRitualAdeptNotAWizard struct{}

func (RitualAdeptEligible) isCheckRitualAdeptResult()        {}
func (CheckRitualAdeptRejected) isCheckRitualAdeptResult()   {}
func (CheckRitualAdeptNotAWizard) isCheckRitualAdeptResult() {}

// CheckRitualAdept is the pure, advisory use case that says whether a spell can be cast as a
// Ritual with Ritual Adept (SRD 5.2.1, "Level 1: Ritual Adept", p. 78). The host calls it, then
// CastSpell with the Ritual mode; CastSpell does not read it. Create it with NewCheckRitualAdept.
type CheckRitualAdept struct{ wizardUse }

// NewCheckRitualAdept returns the use case. Options: WithTelemetry, WithClock.
func NewCheckRitualAdept(opts ...Option) CheckRitualAdept {
	return CheckRitualAdept{newWizardUse(nil, opts)}
}

// Run answers for cmd.Spell, with the errors of CopySpellbook.Run.
func (u CheckRitualAdept) Run(ctx context.Context, cmd CheckRitualAdeptCommand) (CheckRitualAdeptResult, error) {
	if err := wizardCommandProblem(u.set, Identifier{}, false); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spell); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, _ := u.open(ctx, SpanCheckRitualAdept, ValueFeatureRitualAdept, &cmd.ClassLevels, false)
	defer call.run.end(ctx)
	if call.stopped(ctx) {
		return CheckRitualAdeptNotAWizard{}, nil
	}
	if rej := checkRitualAdept(cmd.Spellbook, cmd.Spell, cmd.SpellbookAvailable); rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return CheckRitualAdeptRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return RitualAdeptEligible{}, nil
}

// ---- ChooseSpellMastery -------------------------------------------------

// ChooseSpellMasteryCommand is the input of ChooseSpellMastery.Run: the level 1 and level 2
// spells chosen. Construct it with a keyed literal.
type ChooseSpellMasteryCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Resources   WizardResources
	Spellbook   Spellbook
	Spells      []WizardSpellFacts
}

// ChooseSpellMasteryResult is the closed set of outcomes of ChooseSpellMastery.Run. Variants:
// SpellMasteryApplied, ChooseSpellMasteryRejected, ChooseSpellMasteryContentMissing,
// ChooseSpellMasteryNotAWizard.
type ChooseSpellMasteryResult interface{ isChooseSpellMasteryResult() }

// SpellMasteryApplied is the outcome of a choice the rules allowed.
type SpellMasteryApplied struct {
	mastery   SpellMastery
	resources WizardResources
	events    []Event
}

// Mastery returns the choice.
func (r SpellMasteryApplied) Mastery() SpellMastery { return r.mastery }

// Resources returns the new WizardResources, holding the choice.
func (r SpellMasteryApplied) Resources() WizardResources { return r.resources }

// Events returns a new slice holding the SpellMasteryChosen event.
func (r SpellMasteryApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseSpellMasteryRejected is the outcome of a choice a rule refused.
type ChooseSpellMasteryRejected struct{ Rejection SpellMasteryRejection }

// ChooseSpellMasteryContentMissing is the outcome when the wizard class is not in the source.
type ChooseSpellMasteryContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseSpellMasteryContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// ChooseSpellMasteryNotAWizard is the outcome for a character with no Wizard levels.
type ChooseSpellMasteryNotAWizard struct{}

func (SpellMasteryApplied) isChooseSpellMasteryResult()              {}
func (ChooseSpellMasteryRejected) isChooseSpellMasteryResult()       {}
func (ChooseSpellMasteryContentMissing) isChooseSpellMasteryResult() {}
func (ChooseSpellMasteryNotAWizard) isChooseSpellMasteryResult()     {}

// ChooseSpellMastery is the use case that applies a Wizard's Spell Mastery choice (SRD 5.2.1,
// "Level 18: Spell Mastery", p. 79). Create it with NewChooseSpellMastery.
type ChooseSpellMastery struct{ wizardUse }

// NewChooseSpellMastery returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewChooseSpellMastery(source ContentSource, opts ...Option) (ChooseSpellMastery, error) {
	if source == nil {
		return ChooseSpellMastery{}, invalidInput("choose Spell Mastery without a content source")
	}
	return ChooseSpellMastery{newWizardUse(source, opts)}, nil
}

// Run applies the choice, with the errors of ChooseWizardCantrips.Run.
func (u ChooseSpellMastery) Run(ctx context.Context, cmd ChooseSpellMasteryCommand) (ChooseSpellMasteryResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spells...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseSpellMastery, ValueFeatureSpellMastery, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return ChooseSpellMasteryNotAWizard{}, nil
		}
		return ChooseSpellMasteryContentMissing{missing: wizardMissingContent()}, nil
	}
	reject := func(rej SpellMasteryRejection) (ChooseSpellMasteryResult, error) {
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseSpellMasteryRejected{Rejection: rej}, nil
	}
	if gained := wizardFeatureLevel(call.def, "wizard_spell_mastery"); gained == 0 || call.level < gained {
		return reject(SpellMasteryRejection{Kind: SpellMasteryFeatureNotAvailable})
	}
	if _, held := cmd.Resources.SpellMastery(); held {
		return reject(SpellMasteryRejection{Kind: SpellMasteryAlready})
	}
	mastery, rej := NewSpellMastery(cmd.Spells, cmd.Spellbook)
	if rej.Kind != 0 {
		return reject(rej)
	}
	call.run.succeeded(ctx)
	return SpellMasteryApplied{mastery: mastery, resources: cmd.Resources.withMastery(mastery), events: []Event{SpellMasteryChosen{id: cmd.CharacterID, mastery: mastery}}}, nil
}

// ---- ReplaceSpellMastery ------------------------------------------------

// ReplaceSpellMasteryCommand is the input of ReplaceSpellMastery.Run. Construct it with a keyed literal.
type ReplaceSpellMasteryCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Resources   WizardResources
	Spellbook   Spellbook
	Replace     content.SpellID
	New         WizardSpellFacts
	Trigger     RestApplied
}

// ReplaceSpellMasteryResult is the closed set of outcomes of ReplaceSpellMastery.Run. Variants:
// SpellMasteryReplaceApplied, ReplaceSpellMasteryRejected, ReplaceSpellMasteryContentMissing,
// ReplaceSpellMasteryNotAWizard.
type ReplaceSpellMasteryResult interface{ isReplaceSpellMasteryResult() }

// SpellMasteryReplaceApplied is the outcome of a replacement the rules allowed.
type SpellMasteryReplaceApplied struct {
	mastery   SpellMastery
	resources WizardResources
	events    []Event
}

// Mastery returns the choice now held.
func (r SpellMasteryReplaceApplied) Mastery() SpellMastery { return r.mastery }

// Resources returns the new WizardResources.
func (r SpellMasteryReplaceApplied) Resources() WizardResources { return r.resources }

// Events returns a new slice holding the SpellMasteryReplaced event.
func (r SpellMasteryReplaceApplied) Events() []Event { return slices.Clone(r.events) }

// ReplaceSpellMasteryRejected is the outcome of a replacement a rule refused.
type ReplaceSpellMasteryRejected struct{ Rejection SpellMasteryRejection }

// ReplaceSpellMasteryContentMissing is the outcome when the wizard class is not in the source.
type ReplaceSpellMasteryContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ReplaceSpellMasteryContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// ReplaceSpellMasteryNotAWizard is the outcome for a character with no Wizard levels.
type ReplaceSpellMasteryNotAWizard struct{}

func (SpellMasteryReplaceApplied) isReplaceSpellMasteryResult()        {}
func (ReplaceSpellMasteryRejected) isReplaceSpellMasteryResult()       {}
func (ReplaceSpellMasteryContentMissing) isReplaceSpellMasteryResult() {}
func (ReplaceSpellMasteryNotAWizard) isReplaceSpellMasteryResult()     {}

// ReplaceSpellMastery is the use case that replaces a Spell Mastery spell after a Long Rest
// (SRD 5.2.1, "Level 18: Spell Mastery", p. 79). Create it with NewReplaceSpellMastery.
type ReplaceSpellMastery struct{ wizardUse }

// NewReplaceSpellMastery returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewReplaceSpellMastery(source ContentSource, opts ...Option) (ReplaceSpellMastery, error) {
	if source == nil {
		return ReplaceSpellMastery{}, invalidInput("replace Spell Mastery without a content source")
	}
	return ReplaceSpellMastery{newWizardUse(source, opts)}, nil
}

// Run applies the replacement, with the errors of ChooseWizardCantrips.Run.
func (u ReplaceSpellMastery) Run(ctx context.Context, cmd ReplaceSpellMasteryCommand) (ReplaceSpellMasteryResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.New); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanReplaceSpellMastery, ValueFeatureSpellMasteryReplace, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return ReplaceSpellMasteryNotAWizard{}, nil
		}
		return ReplaceSpellMasteryContentMissing{missing: wizardMissingContent()}, nil
	}
	reject := func(rej SpellMasteryRejection) (ReplaceSpellMasteryResult, error) {
		call.run.rejected(ctx, rej.Kind.String())
		return ReplaceSpellMasteryRejected{Rejection: rej}, nil
	}
	held, isHeld := cmd.Resources.SpellMastery()
	switch gained := wizardFeatureLevel(call.def, "wizard_spell_mastery"); {
	case gained == 0 || call.level < gained:
		return reject(SpellMasteryRejection{Kind: SpellMasteryFeatureNotAvailable})
	case !longRestBenefits(cmd.Trigger):
		return reject(SpellMasteryRejection{Kind: SpellMasteryNotAfterLongRest})
	case !isHeld:
		return reject(SpellMasteryRejection{Kind: SpellMasteryNotHeld})
	}
	next, rej := held.replace(cmd.Replace, cmd.New, cmd.Spellbook)
	if rej.Kind != 0 {
		return reject(rej)
	}
	call.run.succeeded(ctx)
	return SpellMasteryReplaceApplied{mastery: next, resources: cmd.Resources.withMastery(next),
		events: []Event{SpellMasteryReplaced{id: cmd.CharacterID, replaced: cmd.Replace, mastery: next}}}, nil
}

// ---- ChooseSignatureSpells ----------------------------------------------

// ChooseSignatureSpellsCommand is the input of ChooseSignatureSpells.Run: the two level 3 spells
// chosen. Construct it with a keyed literal.
type ChooseSignatureSpellsCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Resources   WizardResources
	Spellbook   Spellbook
	Spells      []WizardSpellFacts
}

// ChooseSignatureSpellsResult is the closed set of outcomes of ChooseSignatureSpells.Run. Variants:
// SignatureSpellsApplied, ChooseSignatureSpellsRejected, ChooseSignatureSpellsContentMissing,
// ChooseSignatureSpellsNotAWizard.
type ChooseSignatureSpellsResult interface{ isChooseSignatureSpellsResult() }

// SignatureSpellsApplied is the outcome of a choice the rules allowed.
type SignatureSpellsApplied struct {
	signature SignatureSpells
	resources WizardResources
	events    []Event
}

// Signature returns the choice.
func (r SignatureSpellsApplied) Signature() SignatureSpells { return r.signature }

// Resources returns the new WizardResources, holding the choice with both free casts available.
func (r SignatureSpellsApplied) Resources() WizardResources { return r.resources }

// Events returns a new slice holding the SignatureSpellsChosen event.
func (r SignatureSpellsApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseSignatureSpellsRejected is the outcome of a choice a rule refused.
type ChooseSignatureSpellsRejected struct{ Rejection SignatureSpellsRejection }

// ChooseSignatureSpellsContentMissing is the outcome when the wizard class is not in the source.
type ChooseSignatureSpellsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseSignatureSpellsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// ChooseSignatureSpellsNotAWizard is the outcome for a character with no Wizard levels.
type ChooseSignatureSpellsNotAWizard struct{}

func (SignatureSpellsApplied) isChooseSignatureSpellsResult()              {}
func (ChooseSignatureSpellsRejected) isChooseSignatureSpellsResult()       {}
func (ChooseSignatureSpellsContentMissing) isChooseSignatureSpellsResult() {}
func (ChooseSignatureSpellsNotAWizard) isChooseSignatureSpellsResult()     {}

// ChooseSignatureSpells is the use case that applies a Wizard's Signature Spells choice (SRD
// 5.2.1, "Level 20: Signature Spells", p. 79). Create it with NewChooseSignatureSpells.
type ChooseSignatureSpells struct{ wizardUse }

// NewChooseSignatureSpells returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewChooseSignatureSpells(source ContentSource, opts ...Option) (ChooseSignatureSpells, error) {
	if source == nil {
		return ChooseSignatureSpells{}, invalidInput("choose Signature Spells without a content source")
	}
	return ChooseSignatureSpells{newWizardUse(source, opts)}, nil
}

// Run applies the choice, with the errors of ChooseWizardCantrips.Run.
func (u ChooseSignatureSpells) Run(ctx context.Context, cmd ChooseSignatureSpellsCommand) (ChooseSignatureSpellsResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spells...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseSignatureSpells, ValueFeatureSignatureSpells, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return ChooseSignatureSpellsNotAWizard{}, nil
		}
		return ChooseSignatureSpellsContentMissing{missing: wizardMissingContent()}, nil
	}
	reject := func(rej SignatureSpellsRejection) (ChooseSignatureSpellsResult, error) {
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseSignatureSpellsRejected{Rejection: rej}, nil
	}
	if gained := wizardFeatureLevel(call.def, "wizard_signature_spells"); gained == 0 || call.level < gained {
		return reject(SignatureSpellsRejection{Kind: SignatureSpellsFeatureNotAvailable})
	}
	if _, held := cmd.Resources.SignatureSpells(); held {
		return reject(SignatureSpellsRejection{Kind: SignatureSpellsAlready})
	}
	signature, rej := NewSignatureSpells(cmd.Spells, cmd.Spellbook)
	if rej.Kind != 0 {
		return reject(rej)
	}
	call.run.succeeded(ctx)
	return SignatureSpellsApplied{signature: signature, resources: cmd.Resources.withSignature(signature), events: []Event{SignatureSpellsChosen{id: cmd.CharacterID, signature: signature}}}, nil
}
