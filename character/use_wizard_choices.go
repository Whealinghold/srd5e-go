package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// open starts the span and, when levels is given, decides NotAWizard from it before any
// lookup. When load is set it then looks the Wizard definition up in a SpanLoadContent
// child span. The caller defers call.run.end(ctx) right after open, whatever err is. A
// source failure, panic or broken contract comes back as a *srderr.PortError, recorded once.
func (u wizardUse) open(ctx context.Context, span, feature string, levels *ClassLevels, load bool) (context.Context, wizardCall, error) {
	ctx, run := u.sig.start(ctx, span, ValueClassWizard, feature)
	call := wizardCall{run: run}
	if levels != nil {
		if call.level = levels.LevelIn(WizardClassID); call.level < 1 {
			call.stage = wizardNotAWizard
			return ctx, call, nil
		}
	}
	if !load {
		return ctx, call, nil
	}
	lctx, lspan := u.sig.tel.Tracer.Start(ctx, SpanLoadContent, telemetry.String(AttrClass, ValueClassWizard), telemetry.String(AttrFeature, feature))
	defer lspan.End()
	entry, found, err := lookupEntry(lctx, u.source, WizardClassID.Key())
	if err == nil && found {
		def, ok := entry.(content.ClassDefinition)
		if !ok {
			err = srderr.ContractViolation(content.Port, "lookup", "the wizard key answered with another entry type")
		}
		call.def = def
	}
	if err != nil {
		lspan.RecordError(err)
		lspan.SetStatus(telemetry.StatusError)
		call.run.portFailure(ctx, err, srderr.ErrorType(err))
		return ctx, call, err
	}
	if !found {
		call.stage = wizardMissing
	}
	return ctx, call, nil
}

// wizardFactsProblem returns an error matching srderr.ErrInvalidInput when a spell fact has
// no valid id or level (a spell id with no facts is a programmer error).
func wizardFactsProblem(facts ...WizardSpellFacts) error {
	for _, f := range facts {
		if !f.ID.Valid() || !content.ValidSpellLevel(int(f.Level)) {
			return invalidInput("Wizard use case: spell facts with an invalid id or level")
		}
	}
	return nil
}

// ---- ChooseWizardSkills -------------------------------------------------

// ChooseWizardSkillsCommand is the input of ChooseWizardSkills.Run. Construct it with a keyed literal.
type ChooseWizardSkillsCommand struct {
	CharacterID Identifier
	Skills      []content.Skill
}

// ChooseWizardSkillsResult is the closed set of outcomes of ChooseWizardSkills.Run. Variants:
// WizardSkillsApplied, ChooseWizardSkillsRejected, ChooseWizardSkillsContentMissing.
type ChooseWizardSkillsResult interface{ isChooseWizardSkillsResult() }

// WizardSkillsApplied is the outcome of a choice the rules allowed.
type WizardSkillsApplied struct {
	skills []content.Skill
	events []Event
}

// Skills returns a new slice of the chosen skills.
func (r WizardSkillsApplied) Skills() []content.Skill { return slices.Clone(r.skills) }

// Events returns a new slice holding the WizardSkillsChosen event.
func (r WizardSkillsApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseWizardSkillsRejected is the outcome of a choice a rule refused.
type ChooseWizardSkillsRejected struct{ Rejection SkillsRejection }

// ChooseWizardSkillsContentMissing is the outcome when the wizard class is not in the source.
type ChooseWizardSkillsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseWizardSkillsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (WizardSkillsApplied) isChooseWizardSkillsResult()              {}
func (ChooseWizardSkillsRejected) isChooseWizardSkillsResult()       {}
func (ChooseWizardSkillsContentMissing) isChooseWizardSkillsResult() {}

// ChooseWizardSkills is the use case that validates a Wizard's two Skill Proficiencies
// (SRD 5.2.1, "Core Wizard Traits", p. 77) against the class definition. Create it with
// NewChooseWizardSkills; the zero value is not usable.
type ChooseWizardSkills struct{ wizardUse }

// NewChooseWizardSkills returns the use case. Options: WithTelemetry, WithClock. It returns
// an error matching srderr.ErrInvalidInput for a nil source.
func NewChooseWizardSkills(source ContentSource, opts ...Option) (ChooseWizardSkills, error) {
	if source == nil {
		return ChooseWizardSkills{}, invalidInput("choose Wizard skills without a content source")
	}
	return ChooseWizardSkills{newWizardUse(source, opts)}, nil
}

// Run validates cmd.Skills. It returns an error matching srderr.ErrInvalidInput, before any
// signal, for the zero use case or a zero CharacterID; ctx.Err() unwrapped, before any signal,
// for a cancelled ctx; and a *srderr.PortError when the source fails. A refused choice is a
// ChooseWizardSkillsRejected result.
func (u ChooseWizardSkills) Run(ctx context.Context, cmd ChooseWizardSkillsCommand) (ChooseWizardSkillsResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseWizardSkills, ValueFeatureWizardSkills, nil, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		return ChooseWizardSkillsContentMissing{missing: wizardMissingContent()}, nil
	}
	rej, err := checkWizardSkills(call.def, cmd.Skills)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseWizardSkillsRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return WizardSkillsApplied{skills: slices.Clone(cmd.Skills), events: []Event{WizardSkillsChosen{id: cmd.CharacterID, skills: slices.Clone(cmd.Skills)}}}, nil
}

// ---- ChooseWizardEquipment ----------------------------------------------

// ChooseWizardEquipmentCommand is the input of ChooseWizardEquipment.Run. Option is the raw
// letter. Construct it with a keyed literal.
type ChooseWizardEquipmentCommand struct {
	CharacterID Identifier
	Option      string
}

// ChooseWizardEquipmentResult is the closed set of outcomes of ChooseWizardEquipment.Run.
// Variants: WizardEquipmentApplied, ChooseWizardEquipmentRejected, ChooseWizardEquipmentContentMissing.
type ChooseWizardEquipmentResult interface{ isChooseWizardEquipmentResult() }

// WizardEquipmentApplied is the outcome of a valid option: the definition's option as held.
type WizardEquipmentApplied struct {
	option    string
	equipment content.ClassEquipmentOption
	events    []Event
}

// Option returns "A" or "B".
func (r WizardEquipmentApplied) Option() string { return r.option }

// Equipment returns the option as the class definition holds it (item lines and GP).
func (r WizardEquipmentApplied) Equipment() content.ClassEquipmentOption { return r.equipment }

// Events returns a new slice holding the WizardEquipmentChosen event.
func (r WizardEquipmentApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseWizardEquipmentRejected is the outcome of an unknown option.
type ChooseWizardEquipmentRejected struct{ Rejection EquipmentRejection }

// ChooseWizardEquipmentContentMissing is the outcome when the wizard class is not in the source.
type ChooseWizardEquipmentContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseWizardEquipmentContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (WizardEquipmentApplied) isChooseWizardEquipmentResult()              {}
func (ChooseWizardEquipmentRejected) isChooseWizardEquipmentResult()       {}
func (ChooseWizardEquipmentContentMissing) isChooseWizardEquipmentResult() {}

// ChooseWizardEquipment is the use case that resolves a Wizard's Starting Equipment option
// (SRD 5.2.1, "Core Wizard Traits", p. 77). Create it with NewChooseWizardEquipment.
type ChooseWizardEquipment struct{ wizardUse }

// NewChooseWizardEquipment returns the use case. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewChooseWizardEquipment(source ContentSource, opts ...Option) (ChooseWizardEquipment, error) {
	if source == nil {
		return ChooseWizardEquipment{}, invalidInput("choose Wizard equipment without a content source")
	}
	return ChooseWizardEquipment{newWizardUse(source, opts)}, nil
}

// Run resolves cmd.Option, with the errors of ChooseWizardSkills.Run.
func (u ChooseWizardEquipment) Run(ctx context.Context, cmd ChooseWizardEquipmentCommand) (ChooseWizardEquipmentResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseWizardEquipment, ValueFeatureWizardEquipment, nil, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		return ChooseWizardEquipmentContentMissing{missing: wizardMissingContent()}, nil
	}
	option, rej := wizardEquipment(call.def, cmd.Option)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseWizardEquipmentRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return WizardEquipmentApplied{option: cmd.Option, equipment: option, events: []Event{WizardEquipmentChosen{id: cmd.CharacterID, option: cmd.Option}}}, nil
}

// ---- ChooseWizardCantrips -----------------------------------------------

// ChooseWizardCantripsCommand is the input of ChooseWizardCantrips.Run: the cantrips known and
// the facts of those chosen. Construct it with a keyed literal.
type ChooseWizardCantripsCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Current     []content.SpellID
	Chosen      []WizardSpellFacts
}

// ChooseWizardCantripsResult is the closed set of outcomes of ChooseWizardCantrips.Run. Variants:
// WizardCantripsApplied, ChooseWizardCantripsRejected, ChooseWizardCantripsContentMissing,
// ChooseWizardCantripsNotAWizard.
type ChooseWizardCantripsResult interface{ isChooseWizardCantripsResult() }

// WizardCantripsApplied is the outcome of a choice the rules allowed.
type WizardCantripsApplied struct {
	cantrips []content.SpellID
	events   []Event
}

// Cantrips returns a new slice of the cantrips chosen.
func (r WizardCantripsApplied) Cantrips() []content.SpellID { return slices.Clone(r.cantrips) }

// Events returns a new slice holding the WizardCantripsChosen event.
func (r WizardCantripsApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseWizardCantripsRejected is the outcome of a choice a rule refused.
type ChooseWizardCantripsRejected struct{ Rejection CantripChoiceRejection }

// ChooseWizardCantripsContentMissing is the outcome when the wizard class is not in the source.
type ChooseWizardCantripsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseWizardCantripsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// ChooseWizardCantripsNotAWizard is the outcome for a character with no Wizard levels.
type ChooseWizardCantripsNotAWizard struct{}

func (WizardCantripsApplied) isChooseWizardCantripsResult()              {}
func (ChooseWizardCantripsRejected) isChooseWizardCantripsResult()       {}
func (ChooseWizardCantripsContentMissing) isChooseWizardCantripsResult() {}
func (ChooseWizardCantripsNotAWizard) isChooseWizardCantripsResult()     {}

// ChooseWizardCantrips is the use case that validates a Wizard's cantrip choice (SRD 5.2.1,
// "Level 1: Spellcasting", Cantrips, p. 77). Create it with NewChooseWizardCantrips.
type ChooseWizardCantrips struct{ wizardUse }

// NewChooseWizardCantrips returns the use case. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewChooseWizardCantrips(source ContentSource, opts ...Option) (ChooseWizardCantrips, error) {
	if source == nil {
		return ChooseWizardCantrips{}, invalidInput("choose Wizard cantrips without a content source")
	}
	return ChooseWizardCantrips{newWizardUse(source, opts)}, nil
}

// Run validates cmd.Chosen, with the errors of ChooseWizardSkills.Run, and an error matching
// srderr.ErrInvalidInput for spell facts without a valid id or level.
func (u ChooseWizardCantrips) Run(ctx context.Context, cmd ChooseWizardCantripsCommand) (ChooseWizardCantripsResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Chosen...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseWizardCantrips, ValueFeatureWizardCantrips, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return ChooseWizardCantripsNotAWizard{}, nil
		}
		return ChooseWizardCantripsContentMissing{missing: wizardMissingContent()}, nil
	}
	rej, err := checkWizardCantrips(call.def, call.level, cmd.Current, cmd.Chosen)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseWizardCantripsRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	ids := wizardFactIDs(cmd.Chosen)
	return WizardCantripsApplied{cantrips: ids, events: []Event{WizardCantripsChosen{id: cmd.CharacterID, cantrips: slices.Clone(ids)}}}, nil
}

// ---- ChooseScholarExpertise ---------------------------------------------

// ChooseScholarExpertiseCommand is the input of ChooseScholarExpertise.Run: the state held,
// the skills the character is proficient in and the skill chosen. Construct it with a keyed literal.
type ChooseScholarExpertiseCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Resources   WizardResources
	Proficient  []content.Skill
	Skill       content.Skill
}

// ChooseScholarExpertiseResult is the closed set of outcomes of ChooseScholarExpertise.Run.
// Variants: ScholarApplied, ChooseScholarExpertiseRejected, ChooseScholarExpertiseContentMissing,
// ChooseScholarExpertiseNotAWizard.
type ChooseScholarExpertiseResult interface{ isChooseScholarExpertiseResult() }

// ScholarApplied is the outcome of a choice the rules allowed: Expertise in the skill.
type ScholarApplied struct {
	resources WizardResources
	skill     content.Skill
	events    []Event
}

// Resources returns the new WizardResources, holding the Scholar skill.
func (r ScholarApplied) Resources() WizardResources { return r.resources }

// Skill returns the skill the Wizard now has Expertise in.
func (r ScholarApplied) Skill() content.Skill { return r.skill }

// Events returns a new slice holding the ScholarExpertiseChosen event.
func (r ScholarApplied) Events() []Event { return slices.Clone(r.events) }

// ChooseScholarExpertiseRejected is the outcome of a choice a rule refused.
type ChooseScholarExpertiseRejected struct{ Rejection ScholarRejection }

// ChooseScholarExpertiseContentMissing is the outcome when the wizard class is not in the source.
type ChooseScholarExpertiseContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ChooseScholarExpertiseContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// ChooseScholarExpertiseNotAWizard is the outcome for a character with no Wizard levels.
type ChooseScholarExpertiseNotAWizard struct{}

func (ScholarApplied) isChooseScholarExpertiseResult()                       {}
func (ChooseScholarExpertiseRejected) isChooseScholarExpertiseResult()       {}
func (ChooseScholarExpertiseContentMissing) isChooseScholarExpertiseResult() {}
func (ChooseScholarExpertiseNotAWizard) isChooseScholarExpertiseResult()     {}

// ChooseScholarExpertise is the use case that applies a Wizard's Scholar choice (SRD 5.2.1,
// "Level 2: Scholar", p. 78). Create it with NewChooseScholarExpertise.
type ChooseScholarExpertise struct{ wizardUse }

// NewChooseScholarExpertise returns the use case. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewChooseScholarExpertise(source ContentSource, opts ...Option) (ChooseScholarExpertise, error) {
	if source == nil {
		return ChooseScholarExpertise{}, invalidInput("choose Scholar expertise without a content source")
	}
	return ChooseScholarExpertise{newWizardUse(source, opts)}, nil
}

// Run applies the choice, with the errors of ChooseWizardSkills.Run.
func (u ChooseScholarExpertise) Run(ctx context.Context, cmd ChooseScholarExpertiseCommand) (ChooseScholarExpertiseResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanChooseScholar, ValueFeatureScholar, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return ChooseScholarExpertiseNotAWizard{}, nil
		}
		return ChooseScholarExpertiseContentMissing{missing: wizardMissingContent()}, nil
	}
	gained := wizardFeatureLevel(call.def, "wizard_scholar")
	rej := checkScholar(gained > 0 && call.level >= gained, cmd.Resources, cmd.Skill, cmd.Proficient)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return ChooseScholarExpertiseRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return ScholarApplied{resources: cmd.Resources.withScholar(cmd.Skill), skill: cmd.Skill,
		events: []Event{ScholarExpertiseChosen{id: cmd.CharacterID, skill: cmd.Skill}}}, nil
}
