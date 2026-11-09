package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-wizard.md FR-105 - WizardCreationChoices seeds the initial WizardResources under wizard_resources and leaves other keys intact, names no content key,
// and NewWizardCreationChoices validates skills, equipment, cantrips and spellbook, joining every failing part in that order, each matching ErrInvalidInput.
// Path: per case (happy / unhappy: invalid input, one defect per part and several at once).
func TestNewWizardCreationChoices(t *testing.T) {
	t.Parallel()
	def := wzDef(t)
	skills := []content.Skill{content.SkillArcana, content.SkillHistory}
	cantrips := wzFacts("c", 0, 3)
	book := wzFacts("b", 1, 6)
	good, err := character.NewWizardCreationChoices(def, skills, "A", cantrips, book)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(good.Skills(), skills) || good.EquipmentOption() != "A" || len(good.Cantrips()) != 3 || len(good.Spellbook().Spells()) != 6 {
		t.Errorf("choices = %+v", good)
	}
	good.Skills()[0] = content.SkillNature
	good.Cantrips()[0] = "zz"
	if good.Skills()[0] != content.SkillArcana || good.Cantrips()[0] != "ca" {
		t.Error("an accessor returned shared state")
	}
	if keys := good.ContentKeys(); keys == nil || len(keys) != 0 {
		t.Errorf("ContentKeys = %v, want an empty non-nil slice", keys)
	}
	other := must(character.SetFeatureState(character.FeatureStates{}, character.FighterStateKey(), character.InitialFighterState()))
	seeded := good.SeedFeatureStates(other)
	if r, ok, err := character.ReadFeatureState(seeded, character.WizardResourcesKey()); err != nil || !ok || r != (character.WizardResources{}) || seeded.Len() != 2 || other.Len() != 1 {
		t.Errorf("SeedFeatureStates = %+v (%v, %v)", seeded, ok, err)
	}
	if clash := (character.WizardCreationChoices{}).SeedFeatureStates(seeded); clash.Len() != 2 {
		t.Error("seeding twice must keep one entry")
	}
	type part = character.WizardCreationRejectionKind
	for _, c := range []struct {
		name     string
		skills   []content.Skill
		option   string
		cantrips []character.WizardSpellFacts
		book     []character.WizardSpellFacts
		want     []part
	}{
		{"skills", skills[:1], "A", cantrips, book, []part{character.WizardCreationSkillsInvalid}},
		{"equipment", skills, "C", cantrips, book, []part{character.WizardCreationEquipmentInvalid}},
		{"cantrips", skills, "B", cantrips[:2], book, []part{character.WizardCreationCantripsInvalid}},
		{"spellbook", skills, "A", cantrips, book[:5], []part{character.WizardCreationSpellbookInvalid}},
		{"all four", nil, "", nil, nil, []part{character.WizardCreationSkillsInvalid, character.WizardCreationEquipmentInvalid, character.WizardCreationCantripsInvalid, character.WizardCreationSpellbookInvalid}},
	} {
		_, err := character.NewWizardCreationChoices(def, c.skills, c.option, c.cantrips, c.book)
		if !srderrIsInvalid(err) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", c.name, err)
			continue
		}
		var got []part
		for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
			var ce character.WizardCreationError
			if !errors.As(e, &ce) {
				t.Fatalf("%s: %v is not a WizardCreationError", c.name, e)
			}
			r := ce.Rejection
			got = append(got, r.Kind)
			if r.String() == "unknown" || ce.Error() == "" {
				t.Errorf("%s: rendering = %q", c.name, r.String())
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: parts = %v, want %v", c.name, got, c.want)
		}
	}
	if _, err := character.NewWizardCreationChoices(content.ClassDefinition{}, skills, "A", cantrips, book); !srderrIsInvalid(err) {
		t.Errorf("a zero definition: err = %v", err)
	}
}
