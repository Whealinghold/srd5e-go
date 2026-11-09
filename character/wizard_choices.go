package character

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// checkWizardSkills validates the Wizard Skill Proficiencies choice against def
// (SRD 5.2.1, "Core Wizard Traits", p. 77): exactly the listed number of distinct
// skills from the listed ones. A rejection has a non-zero Kind. It returns an error
// matching srderr.ErrInvalidInput when def lists no skills.
func checkWizardSkills(def content.ClassDefinition, skills []content.Skill) (SkillsRejection, error) {
	listed, ok := def.SkillChoice().(content.SkillChoiceListed)
	if !ok {
		return SkillsRejection{}, invalidInput("Wizard skills: the class definition lists no skills")
	}
	if len(skills) != listed.Picks() {
		return SkillsRejection{Kind: SkillsWrongSkillCount, Count: len(skills)}, nil
	}
	for i, s := range skills {
		if slices.Contains(skills[:i], s) {
			return SkillsRejection{Kind: SkillsDuplicateSkill, Skill: s}, nil
		}
	}
	for _, s := range skills {
		if !slices.Contains(listed.Skills(), s) {
			return SkillsRejection{Kind: SkillsSkillNotAllowed, Skill: s}, nil
		}
	}
	return SkillsRejection{}, nil
}

// wizardEquipment resolves the Starting Equipment option letter against def. A
// rejection has a non-zero Kind.
func wizardEquipment(def content.ClassDefinition, option string) (content.ClassEquipmentOption, EquipmentRejection) {
	switch option {
	case "A":
		return def.StartingEquipment().A(), EquipmentRejection{}
	case "B":
		return def.StartingEquipment().B(), EquipmentRejection{}
	}
	return content.ClassEquipmentOption{}, EquipmentRejection{Kind: EquipmentUnknownEquipmentOption, Option: option}
}

// checkWizardCantrips validates a cantrip choice at the Wizard level: exactly the
// pending number (the Cantrips column less current), distinct, level 0 and on the
// Wizard list. A rejection has a non-zero Kind. It returns an error matching
// srderr.ErrInvalidInput for a level outside 1 to 20.
func checkWizardCantrips(def content.ClassDefinition, level int, current []content.SpellID, chosen []WizardSpellFacts) (CantripChoiceRejection, error) {
	allowed, err := WizardRules{}.CantripsPending(def, level, len(current))
	if err != nil {
		return CantripChoiceRejection{}, err
	}
	if len(chosen) != allowed {
		return CantripChoiceRejection{Kind: CantripChoiceWrongCantripCount, Allowed: allowed}, nil
	}
	ids := wizardFactIDs(chosen)
	if id, dup := firstRepeat(ids); dup {
		return CantripChoiceRejection{Kind: CantripChoiceDuplicateCantrip, Spell: id}, nil
	}
	for _, f := range chosen {
		if slices.Contains(current, f.ID) {
			return CantripChoiceRejection{Kind: CantripChoiceDuplicateCantrip, Spell: f.ID}, nil
		}
	}
	for _, f := range chosen {
		if !f.IsCantrip() {
			return CantripChoiceRejection{Kind: CantripChoiceNotACantrip, Spell: f.ID}, nil
		}
	}
	for _, f := range chosen {
		if !f.OnWizardList() {
			return CantripChoiceRejection{Kind: CantripChoiceNotOnWizardList, Spell: f.ID}, nil
		}
	}
	return CantripChoiceRejection{}, nil
}

// WizardCreationChoices is a validated set of level 1 Wizard choices (SRD 5.2.1, "As
// a Level 1 Character", p. 77) and the ClassCreationChoices of a new Wizard: it
// seeds the initial WizardResources under WizardResourcesKey and names no content.
// The host passes Skills and EquipmentOption to CreateCharacter and the cantrips and
// spellbook to CreateSpellcasting. Immutable; the accessors return copies. Build it
// with NewWizardCreationChoices; the zero value seeds the initial state and holds no choices.
type WizardCreationChoices struct {
	skills    []content.Skill
	option    string
	cantrips  []content.SpellID
	spellbook Spellbook
}

// NewWizardCreationChoices validates the four parts of the level 1 choices against
// the Wizard definition: the two skills, the equipment option letter, the three
// cantrips and the six spellbook spells. It returns an error joining every
// WizardCreationError, in the order skills, equipment, cantrips, spellbook (so
// the first names the first failing part); each matches srderr.ErrInvalidInput. It
// returns a plain error matching srderr.ErrInvalidInput when def has no Wizard rows.
func NewWizardCreationChoices(wizard content.ClassDefinition, skills []content.Skill, equipmentOption string, cantrips, spellbook []WizardSpellFacts) (WizardCreationChoices, error) {
	var errs []error
	sk, err := checkWizardSkills(wizard, skills)
	if err != nil {
		return WizardCreationChoices{}, err
	}
	if sk.Kind != 0 {
		errs = append(errs, WizardCreationError{WizardCreationRejection{Kind: WizardCreationSkillsInvalid, Skills: sk}})
	}
	if _, eq := wizardEquipment(wizard, equipmentOption); eq.Kind != 0 {
		errs = append(errs, WizardCreationError{WizardCreationRejection{Kind: WizardCreationEquipmentInvalid, Equipment: eq}})
	}
	ct, err := checkWizardCantrips(wizard, 1, nil, cantrips)
	if err != nil {
		return WizardCreationChoices{}, err
	}
	if ct.Kind != 0 {
		errs = append(errs, WizardCreationError{WizardCreationRejection{Kind: WizardCreationCantripsInvalid, Cantrips: ct}})
	}
	book, bk := SpellbookRules{}.Start(Identifier{}, spellbook)
	if bk.Kind != 0 {
		errs = append(errs, WizardCreationError{WizardCreationRejection{Kind: WizardCreationSpellbookInvalid, Spellbook: bk}})
	}
	if len(errs) > 0 {
		return WizardCreationChoices{}, errors.Join(errs...)
	}
	return WizardCreationChoices{skills: slices.Clone(skills), option: equipmentOption, cantrips: wizardFactIDs(cantrips), spellbook: book.Spellbook()}, nil
}

// Skills returns a new slice of the two chosen skills.
func (c WizardCreationChoices) Skills() []content.Skill { return slices.Clone(c.skills) }

// EquipmentOption returns the chosen Starting Equipment letter, "A" or "B".
func (c WizardCreationChoices) EquipmentOption() string { return c.option }

// Cantrips returns a new slice of the three chosen cantrips.
func (c WizardCreationChoices) Cantrips() []content.SpellID { return slices.Clone(c.cantrips) }

// Spellbook returns the six starting spells.
func (c WizardCreationChoices) Spellbook() Spellbook { return NewSpellbook(c.spellbook.Spells()) }

// SeedFeatureStates returns states with the initial WizardResources stored under
// WizardResourcesKey and every other entry unchanged; states itself is unchanged.
func (WizardCreationChoices) SeedFeatureStates(states FeatureStates) FeatureStates {
	next, err := SetFeatureState(states, WizardResourcesKey(), WizardResources{})
	if err != nil { // the key is valid; only a foreign type under the name fails
		return states
	}
	return next
}

// ContentKeys returns an empty slice: the Wizard choices name no content key.
func (WizardCreationChoices) ContentKeys() []content.Key { return []content.Key{} }

var _ ClassCreationChoices = WizardCreationChoices{}
