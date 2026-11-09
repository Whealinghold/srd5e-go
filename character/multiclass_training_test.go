package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	martialWeapons = `[{"category":"Simple"},{"category":"Martial"}]`
	shieldGrants   = `{"weaponProficiencies":[{"category":"Martial"}],"armorTraining":["Shield"],"toolProficiencies":[]}`
	fighterGrants  = `{"weaponProficiencies":[{"category":"Martial"}],"armorTraining":["Light","Medium","Shield"],"toolProficiencies":[]}`
)

// grantClasses are synthetic classes whose multiclass grants follow SRD 5.2.1,
// "As a Multiclass Character" (Barbarian p. 28, Fighter p. 47, Wizard p. 77).
func grantClasses(t *testing.T) map[content.ClassID]content.ClassDefinition {
	t.Helper()
	barbarian := advClass(t, advSpec{id: "barbarian", die: "D12", kind: "allOf", abilities: []string{"strength"}, weapons: martialWeapons, armor: `["Shield"]`, grants: shieldGrants})
	fighter := advClass(t, advSpec{id: "fighter", die: "D10", kind: "anyOf", abilities: []string{"strength", "dexterity"}, weapons: martialWeapons, armor: `["Light","Medium","Heavy","Shield"]`, grants: fighterGrants})
	wizard := advWizard(t)
	return map[content.ClassID]content.ClassDefinition{"barbarian": barbarian, "fighter": fighter, "wizard": wizard}
}

func grantsOf(g content.MulticlassGrants) ([]content.WeaponTrainingCategory, []content.ArmorTrainingCategory) {
	var weapons []content.WeaponTrainingCategory
	for _, w := range g.WeaponProficiencies() {
		weapons = append(weapons, w.Category())
	}
	return weapons, g.ArmorTraining()
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-112, FR-108 to
// FR-111 - Table MG, GrantsFor rows: the grants of every class after the first in
// ClassLevels order, none for a single class, the grants depending on the class only;
// a class without a definition is invalid input.
// Path: per case.
func TestMulticlassTrainingGrantsFor(t *testing.T) {
	t.Parallel()
	defs := grantClasses(t)
	for _, c := range []struct {
		name, requirement, path string
		levels                  []character.ClassLevel
		want                    []content.ClassID // classes whose grants are expected, in order
	}{
		{"two classes", aiSpec + " FR-112 (SRD 5.2.1, Multiclassing, Proficiencies, p. 25)", "happy", []character.ClassLevel{cl("fighter", 2), cl("wizard", 1)}, []content.ClassID{"wizard"}},
		{"single class", aiSpec + " FR-112", "unhappy (boundary)", []character.ClassLevel{cl("fighter", 3)}, nil},
		{"three classes", aiSpec + " FR-112", "happy", []character.ClassLevel{cl("fighter", 2), cl("wizard", 1), cl("barbarian", 1)}, []content.ClassID{"wizard", "barbarian"}},
		{"the initial class grants nothing", aiSpec + " FR-109", "unhappy (rule interaction)", []character.ClassLevel{cl("barbarian", 1), cl("fighter", 1)}, []content.ClassID{"fighter"}},
		{"order of ClassLevels, not of the map", aiSpec + " FR-111", "unhappy (determinism)", []character.ClassLevel{cl("wizard", 1), cl("barbarian", 1), cl("fighter", 1)}, []content.ClassID{"barbarian", "fighter"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.MulticlassTraining{}.GrantsFor(levels(c.levels...), defs)
			if err != nil || got == nil || len(got) != len(c.want) {
				t.Fatalf("grants = %v, %v, want %d entries [%s]", got, err, len(c.want), c.requirement)
			}
			for i, id := range c.want {
				w1, a1 := grantsOf(got[i])
				w2, a2 := grantsOf(defs[id].MulticlassGrants())
				if !slices.Equal(w1, w2) || !slices.Equal(a1, a2) {
					t.Errorf("entry %d is not the grants of %s", i, id)
				}
			}
		})
	}
	if _, err := (character.MulticlassTraining{}).GrantsFor(levels(cl("fighter", 1), cl("wizard", 1)), map[content.ClassID]content.ClassDefinition{"fighter": defs["fighter"]}); !isInvalid(err) {
		t.Errorf("missing definition: %v", err)
	}
	if _, err := (character.MulticlassTraining{}).GrantsFor(character.ClassLevels{}, defs); !isInvalid(err) {
		t.Errorf("empty levels: %v", err)
	}
	// The initial class needs no definition.
	if g, err := (character.MulticlassTraining{}).GrantsFor(levels(cl("fighter", 3)), nil); err != nil || len(g) != 0 {
		t.Errorf("single class without definitions: %v, %v", g, err)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-108, FR-110 - Table
// MG: a new class's grants are the weapon and armor training of its multiclass
// grants (Martial weapons and Shields for the Barbarian; none for the Wizard) and
// hold no saving throw proficiency.
// Path: per case.
func TestMulticlassGrantsOfSampleClasses(t *testing.T) {
	t.Parallel()
	defs := grantClasses(t)
	for _, c := range []struct {
		name, requirement string
		class             content.ClassID
		weapons           []content.WeaponTrainingCategory
		armor             []content.ArmorTrainingCategory
	}{
		{"Barbarian as a new class", aiSpec + " FR-108, SRD 5.2.1, p. 28", "barbarian", []content.WeaponTrainingCategory{content.WeaponTrainingMartial}, []content.ArmorTrainingCategory{content.ArmorTrainingShield}},
		{"Wizard as a new class", aiSpec + " FR-108, SRD 5.2.1, p. 77", "wizard", nil, nil},
	} {
		w, a := grantsOf(defs[c.class].MulticlassGrants())
		if !slices.Equal(w, c.weapons) || !slices.Equal(a, c.armor) {
			t.Errorf("%s: weapons %v armor %v [%s]", c.name, w, a, c.requirement)
		}
	}
}
