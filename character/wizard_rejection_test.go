package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-wizard.md FR-097, FR-100 - each rejection Kind has a distinct snake_case name (the srd5e.character.rejection value); the zero and an out-of-range Kind are "unknown"; a creation rejection
// names its part and cause, renders and matches ErrInvalidInput.
// Path: per case (happy / unhappy: boundary at the ends of each enum).
func TestWizardRejectionNames(t *testing.T) {
	t.Parallel()
	names := wzKindNames()
	if len(names) != 3+1+4+8+5+4+8+9+6+3 {
		t.Fatalf("%d kind names", len(names))
	}
	for i, n := range names {
		if n == "" || n == "unknown" {
			t.Errorf("name %d = %q", i, n)
		}
	}
	for _, unknown := range []string{
		character.SkillsRejectionKind(0).String(), character.SkillsRejectionKind(4).String(), character.EquipmentRejectionKind(0).String(), character.CantripChoiceRejectionKind(5).String(),
		character.SpellbookRejectionKind(0).String(), character.ArcaneRecoveryRejectionKind(6).String(), character.ScholarRejectionKind(0).String(), character.MemorizeRejectionKind(9).String(),
		character.SpellMasteryRejectionKind(0).String(), character.SignatureSpellsRejectionKind(7).String(), character.RitualAdeptRejectionKind(4).String(), character.WizardCreationRejectionKind(0).String(),
	} {
		if unknown != "unknown" {
			t.Errorf("out of range name = %q", unknown)
		}
	}
	for _, r := range []character.WizardCreationRejection{
		{Kind: character.WizardCreationSkillsInvalid, Skills: character.SkillsRejection{Kind: character.SkillsWrongSkillCount}},
		{Kind: character.WizardCreationEquipmentInvalid, Equipment: character.EquipmentRejection{Kind: character.EquipmentUnknownEquipmentOption}},
		{Kind: character.WizardCreationCantripsInvalid, Cantrips: character.CantripChoiceRejection{Kind: character.CantripChoiceNotACantrip}},
		{Kind: character.WizardCreationSpellbookInvalid, Spellbook: character.SpellbookRejection{Kind: character.SpellbookDuplicateSpell}},
	} {
		if err := (character.WizardCreationError{Rejection: r}); r.String() == "unknown" || !errors.Is(err, srderr.ErrInvalidInput) || err.Error() == "" {
			t.Errorf("%+v renders %q", r, r.String())
		}
	}
	if (character.WizardCreationRejection{}).String() != "unknown" {
		t.Error("the zero creation rejection must render unknown")
	}
}
