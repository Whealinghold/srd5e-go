package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// The two Evoker spellbook use cases share the accepted outcome SpellbookChanged. Both
// load the Wizard class to resolve the subclass the host names, so each has its own
// rejected, content-missing, subclass-unknown and not-a-Wizard outcomes.

// evokerSubclassOf resolves id in def. The zero id means the character has no subclass and
// gives the zero SubclassDefinition; an id def lists gives its definition; any other id
// reports false (the caller returns its SubclassUnknown).
func evokerSubclassOf(def content.ClassDefinition, id content.SubclassID) (content.SubclassDefinition, bool) {
	if id == "" {
		return content.SubclassDefinition{}, true
	}
	if found, ok := def.Subclass(id).(content.SubclassFound); ok {
		return found.Subclass(), true
	}
	return content.SubclassDefinition{}, false
}

// ---- AddEvocationSavantSpells -------------------------------------------

// AddEvocationSavantSpellsCommand is the input of AddEvocationSavantSpells.Run: the two
// Evocation spells to add. Subclass is the id of the character's subclass, or empty for none.
// Construct it with a keyed literal.
type AddEvocationSavantSpellsCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Subclass    content.SubclassID
	Spellbook   Spellbook
	Spells      []WizardSpellFacts
}

// AddEvocationSavantSpellsResult is the closed set of outcomes of AddEvocationSavantSpells.Run.
// Variants: SpellbookChanged, AddEvocationSavantSpellsRejected, AddEvocationSavantSpellsContentMissing,
// AddEvocationSavantSpellsSubclassUnknown, AddEvocationSavantSpellsNotAWizard.
type AddEvocationSavantSpellsResult interface{ isAddEvocationSavantSpellsResult() }

// AddEvocationSavantSpellsRejected is the outcome of a choice a rule refused.
type AddEvocationSavantSpellsRejected struct{ Rejection EvokerSpellbookRejection }

// AddEvocationSavantSpellsContentMissing is the outcome when the wizard class is not in the source.
type AddEvocationSavantSpellsContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r AddEvocationSavantSpellsContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// AddEvocationSavantSpellsSubclassUnknown is the outcome when the Wizard lists no such subclass.
type AddEvocationSavantSpellsSubclassUnknown struct{ id content.SubclassID }

// Subclass returns the id that names no Wizard subclass.
func (r AddEvocationSavantSpellsSubclassUnknown) Subclass() content.SubclassID { return r.id }

// AddEvocationSavantSpellsNotAWizard is the outcome for a character with no Wizard levels.
type AddEvocationSavantSpellsNotAWizard struct{}

func (SpellbookChanged) isAddEvocationSavantSpellsResult()                        {}
func (AddEvocationSavantSpellsRejected) isAddEvocationSavantSpellsResult()        {}
func (AddEvocationSavantSpellsContentMissing) isAddEvocationSavantSpellsResult()  {}
func (AddEvocationSavantSpellsSubclassUnknown) isAddEvocationSavantSpellsResult() {}
func (AddEvocationSavantSpellsNotAWizard) isAddEvocationSavantSpellsResult()      {}

// AddEvocationSavantSpells is the use case that adds the two spells of Evocation Savant (SRD
// 5.2.1, "Level 3: Evocation Savant", p. 82). Create it with NewAddEvocationSavantSpells.
type AddEvocationSavantSpells struct{ wizardUse }

// NewAddEvocationSavantSpells returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewAddEvocationSavantSpells(source ContentSource, opts ...Option) (AddEvocationSavantSpells, error) {
	if source == nil {
		return AddEvocationSavantSpells{}, invalidInput("add Evocation Savant spells without a content source")
	}
	return AddEvocationSavantSpells{newWizardUse(source, opts)}, nil
}

// Run adds the spells, with the errors of ChooseWizardCantrips.Run.
func (u AddEvocationSavantSpells) Run(ctx context.Context, cmd AddEvocationSavantSpellsCommand) (AddEvocationSavantSpellsResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spells...); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanAddEvocationSavantSpells, ValueFeatureEvocationSavant, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return AddEvocationSavantSpellsNotAWizard{}, nil
		}
		return AddEvocationSavantSpellsContentMissing{missing: wizardMissingContent()}, nil
	}
	sub, known := evokerSubclassOf(call.def, cmd.Subclass)
	if !known {
		call.run.rejected(ctx, ValueRejectionSubclassUnknown)
		return AddEvocationSavantSpellsSubclassUnknown{id: cmd.Subclass}, nil
	}
	gate := EvokerRules{}.Gate(sub, call.level, EvocationSavantID)
	changed, rej := EvokerRules{}.AddEvocationSavant(cmd.CharacterID, gate, cmd.Spellbook, cmd.Spells)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return AddEvocationSavantSpellsRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return changed, nil
}

// ---- AddEvokerFreeSpell -------------------------------------------------

// AddEvokerFreeSpellCommand is the input of AddEvokerFreeSpell.Run: the one Evocation spell
// to add on gaining a new level of Spell Slots. Subclass is the id of the character's
// subclass, or empty for none. Construct it with a keyed literal.
type AddEvokerFreeSpellCommand struct {
	CharacterID Identifier
	ClassLevels ClassLevels
	Subclass    content.SubclassID
	Spellbook   Spellbook
	Spell       WizardSpellFacts
}

// AddEvokerFreeSpellResult is the closed set of outcomes of AddEvokerFreeSpell.Run.
// Variants: SpellbookChanged, AddEvokerFreeSpellRejected, AddEvokerFreeSpellContentMissing,
// AddEvokerFreeSpellSubclassUnknown, AddEvokerFreeSpellNotAWizard.
type AddEvokerFreeSpellResult interface{ isAddEvokerFreeSpellResult() }

// AddEvokerFreeSpellRejected is the outcome of a choice a rule refused.
type AddEvokerFreeSpellRejected struct{ Rejection EvokerSpellbookRejection }

// AddEvokerFreeSpellContentMissing is the outcome when the wizard class is not in the source.
type AddEvokerFreeSpellContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r AddEvokerFreeSpellContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// AddEvokerFreeSpellSubclassUnknown is the outcome when the Wizard lists no such subclass.
type AddEvokerFreeSpellSubclassUnknown struct{ id content.SubclassID }

// Subclass returns the id that names no Wizard subclass.
func (r AddEvokerFreeSpellSubclassUnknown) Subclass() content.SubclassID { return r.id }

// AddEvokerFreeSpellNotAWizard is the outcome for a character with no Wizard levels.
type AddEvokerFreeSpellNotAWizard struct{}

func (SpellbookChanged) isAddEvokerFreeSpellResult()                  {}
func (AddEvokerFreeSpellRejected) isAddEvokerFreeSpellResult()        {}
func (AddEvokerFreeSpellContentMissing) isAddEvokerFreeSpellResult()  {}
func (AddEvokerFreeSpellSubclassUnknown) isAddEvokerFreeSpellResult() {}
func (AddEvokerFreeSpellNotAWizard) isAddEvokerFreeSpellResult()      {}

// AddEvokerFreeSpell is the use case that adds the free spell the Evoker gets on gaining
// access to a new level of Spell Slots (SRD 5.2.1, "Level 3: Evocation Savant", p. 82).
// Create it with NewAddEvokerFreeSpell.
type AddEvokerFreeSpell struct{ wizardUse }

// NewAddEvokerFreeSpell returns the use case. It returns an error matching srderr.ErrInvalidInput for a nil source.
func NewAddEvokerFreeSpell(source ContentSource, opts ...Option) (AddEvokerFreeSpell, error) {
	if source == nil {
		return AddEvokerFreeSpell{}, invalidInput("add an Evoker free spell without a content source")
	}
	return AddEvokerFreeSpell{newWizardUse(source, opts)}, nil
}

// Run adds the spell, with the errors of ChooseWizardCantrips.Run.
func (u AddEvokerFreeSpell) Run(ctx context.Context, cmd AddEvokerFreeSpellCommand) (AddEvokerFreeSpellResult, error) {
	if err := wizardCommandProblem(u.set, cmd.CharacterID, true); err != nil {
		return nil, err
	}
	if err := wizardFactsProblem(cmd.Spell); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanAddEvokerFreeSpell, ValueFeatureEvokerFreeSpell, &cmd.ClassLevels, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if call.stopped(ctx) {
		if call.stage == wizardNotAWizard {
			return AddEvokerFreeSpellNotAWizard{}, nil
		}
		return AddEvokerFreeSpellContentMissing{missing: wizardMissingContent()}, nil
	}
	sub, known := evokerSubclassOf(call.def, cmd.Subclass)
	if !known {
		call.run.rejected(ctx, ValueRejectionSubclassUnknown)
		return AddEvokerFreeSpellSubclassUnknown{id: cmd.Subclass}, nil
	}
	gate := EvokerRules{}.Gate(sub, call.level, EvocationSavantID)
	newLevel, highest, err := evokerSlotFacts(call.def, call.level)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	changed, rej := EvokerRules{}.AddEvokerFree(cmd.CharacterID, gate, call.level, newLevel, highest, cmd.Spellbook, cmd.Spell)
	if rej.Kind != 0 {
		call.run.rejected(ctx, rej.Kind.String())
		return AddEvokerFreeSpellRejected{Rejection: rej}, nil
	}
	call.run.succeeded(ctx)
	return changed, nil
}

// evokerSlotFacts returns whether the Wizard level brings a new level of Spell Slots and the highest Spell Level at it.
func evokerSlotFacts(def content.ClassDefinition, level int) (newLevel bool, highest int, err error) {
	if newLevel, err = (WizardRules{}).NewSpellLevelAt(def, level); err != nil {
		return false, 0, err
	}
	highest, err = (WizardRules{}).HighestSpellLevel(def, level)
	return newLevel, highest, err
}
