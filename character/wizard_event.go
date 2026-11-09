package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// WizardEvent is something that happened to a Wizard, in the past tense and immutable,
// the closed family of Wizard events. Every one is also an Event and carries the
// Identifier it concerns. The interface cannot be implemented outside this package.
// Members: WizardSkillsChosen, WizardEquipmentChosen, WizardCantripsChosen,
// ScholarExpertiseChosen, SpellsAddedToSpellbook, ArcaneRecoveryUsed, SpellMemorized,
// SpellMasteryChosen, SpellMasteryReplaced, SignatureSpellsChosen, OverchannelUsed.
type WizardEvent interface {
	Event
	isWizardEvent()
}

// WizardSkillsChosen is the event of the Wizard's two skills being chosen.
type WizardSkillsChosen struct {
	id     Identifier
	skills []content.Skill
}

// CharacterID returns the id of the Character.
func (e WizardSkillsChosen) CharacterID() Identifier { return e.id }

// Skills returns a new slice of the chosen skills.
func (e WizardSkillsChosen) Skills() []content.Skill { return slices.Clone(e.skills) }

// WizardEquipmentChosen is the event of a Starting Equipment option being chosen.
type WizardEquipmentChosen struct {
	id     Identifier
	option string
}

// CharacterID returns the id of the Character.
func (e WizardEquipmentChosen) CharacterID() Identifier { return e.id }

// Option returns "A" or "B".
func (e WizardEquipmentChosen) Option() string { return e.option }

// WizardCantripsChosen is the event of Wizard cantrips being chosen.
type WizardCantripsChosen struct {
	id       Identifier
	cantrips []content.SpellID
}

// CharacterID returns the id of the Character.
func (e WizardCantripsChosen) CharacterID() Identifier { return e.id }

// Cantrips returns a new slice of the cantrips chosen.
func (e WizardCantripsChosen) Cantrips() []content.SpellID { return slices.Clone(e.cantrips) }

// ScholarExpertiseChosen is the event of the Scholar skill being chosen.
type ScholarExpertiseChosen struct {
	id    Identifier
	skill content.Skill
}

// CharacterID returns the id of the Character.
func (e ScholarExpertiseChosen) CharacterID() Identifier { return e.id }

// Skill returns the skill the Wizard has Expertise in.
func (e ScholarExpertiseChosen) Skill() content.Skill { return e.skill }

// SpellsAddedToSpellbook is the event of spells being added to the spellbook.
type SpellsAddedToSpellbook struct {
	id     Identifier
	source SpellbookSource
	spells []content.SpellID
}

// CharacterID returns the id of the Character.
func (e SpellsAddedToSpellbook) CharacterID() Identifier { return e.id }

// Source returns why the spells were added.
func (e SpellsAddedToSpellbook) Source() SpellbookSource { return e.source }

// Spells returns a new slice of the spells added.
func (e SpellsAddedToSpellbook) Spells() []content.SpellID { return slices.Clone(e.spells) }

// ArcaneRecoveryUsed is the event of Arcane Recovery being used.
type ArcaneRecoveryUsed struct {
	id     Identifier
	levels int
}

// CharacterID returns the id of the Character.
func (e ArcaneRecoveryUsed) CharacterID() Identifier { return e.id }

// Levels returns the combined level of the slots recovered.
func (e ArcaneRecoveryUsed) Levels() int { return e.levels }

// SpellMemorized is the event of a prepared spell being replaced by Memorize Spell.
type SpellMemorized struct {
	id        Identifier
	replaced  content.SpellID
	memorized content.SpellID
}

// CharacterID returns the id of the Character.
func (e SpellMemorized) CharacterID() Identifier { return e.id }

// Replaced returns the spell no longer prepared.
func (e SpellMemorized) Replaced() content.SpellID { return e.replaced }

// Memorized returns the spell now prepared.
func (e SpellMemorized) Memorized() content.SpellID { return e.memorized }

// SpellMasteryChosen is the event of the Spell Mastery spells being chosen.
type SpellMasteryChosen struct {
	id      Identifier
	mastery SpellMastery
}

// CharacterID returns the id of the Character.
func (e SpellMasteryChosen) CharacterID() Identifier { return e.id }

// Mastery returns the choice.
func (e SpellMasteryChosen) Mastery() SpellMastery { return e.mastery }

// SpellMasteryReplaced is the event of a Spell Mastery spell being replaced after a Long Rest.
type SpellMasteryReplaced struct {
	id       Identifier
	replaced content.SpellID
	mastery  SpellMastery
}

// CharacterID returns the id of the Character.
func (e SpellMasteryReplaced) CharacterID() Identifier { return e.id }

// Replaced returns the spell no longer a Mastery spell.
func (e SpellMasteryReplaced) Replaced() content.SpellID { return e.replaced }

// Mastery returns the choice now held.
func (e SpellMasteryReplaced) Mastery() SpellMastery { return e.mastery }

// SignatureSpellsChosen is the event of the Signature Spells being chosen.
type SignatureSpellsChosen struct {
	id        Identifier
	signature SignatureSpells
}

// CharacterID returns the id of the Character.
func (e SignatureSpellsChosen) CharacterID() Identifier { return e.id }

// Signature returns the choice.
func (e SignatureSpellsChosen) Signature() SignatureSpells { return e.signature }

func (WizardSkillsChosen) isEvent()           {}
func (WizardEquipmentChosen) isEvent()        {}
func (WizardCantripsChosen) isEvent()         {}
func (ScholarExpertiseChosen) isEvent()       {}
func (SpellsAddedToSpellbook) isEvent()       {}
func (ArcaneRecoveryUsed) isEvent()           {}
func (SpellMemorized) isEvent()               {}
func (SpellMasteryChosen) isEvent()           {}
func (SpellMasteryReplaced) isEvent()         {}
func (SignatureSpellsChosen) isEvent()        {}
func (WizardSkillsChosen) isWizardEvent()     {}
func (WizardEquipmentChosen) isWizardEvent()  {}
func (WizardCantripsChosen) isWizardEvent()   {}
func (ScholarExpertiseChosen) isWizardEvent() {}
func (SpellsAddedToSpellbook) isWizardEvent() {}
func (ArcaneRecoveryUsed) isWizardEvent()     {}
func (SpellMemorized) isWizardEvent()         {}
func (SpellMasteryChosen) isWizardEvent()     {}
func (SpellMasteryReplaced) isWizardEvent()   {}
func (SignatureSpellsChosen) isWizardEvent()  {}
