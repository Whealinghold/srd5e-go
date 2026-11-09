package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func spSel(trait, option string) character.SpeciesTraitSelection {
	return character.SpeciesTraitSelection{Trait: trait, Option: option}
}

func spGrants(t *testing.T, skill content.Skill, abilities map[string]ability.Ability) character.SpeciesGrantSelections {
	t.Helper()
	return must(character.NewSpeciesGrantSelections(skill, "", abilities))
}

// spProblems returns the SpeciesChoiceProblems joined in err, in order.
func spProblems(err error) []character.SpeciesChoiceProblem {
	var out []character.SpeciesChoiceProblem
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range j.Unwrap() {
			var p character.SpeciesChoiceProblem
			if errors.As(e, &p) {
				out = append(out, p)
			}
		}
	}
	return out
}

// Requirement: doc/specs/species-trait-behaviour.md FR-001, FR-003, FR-004, FR-005, FR-007, Table C (SRD 5.2.1, pp. 84-86) - each species parses its own choices variant from the printed option names, the ability of the lineage trait and the skill of the grants, and from nothing else.
// Path: per case.
func TestParseSpeciesTraitChoicesHappy(t *testing.T) {
	t.Parallel()
	wis := map[string]ability.Ability{"Elven Lineage": ability.Wisdom}
	for _, c := range []struct {
		name       string
		species    content.SpeciesID
		selections []character.SpeciesTraitSelection
		grants     character.SpeciesGrantSelections
		want       character.SpeciesTraitChoices
	}{
		{"Dragonborn Gold", "dragonborn", []character.SpeciesTraitSelection{spSel("Draconic Ancestry", "Gold")}, character.SpeciesGrantSelections{}, character.DragonbornChoices{Ancestry: character.DraconicGold}},
		{"Elf High Elf", "elf", []character.SpeciesTraitSelection{spSel("Elven Lineage", "High Elf")}, spGrants(t, content.SkillPerception, wis), character.ElfChoices{Lineage: character.ElvenHighElf, Ability: ability.Wisdom, KeenSensesSkill: content.SkillPerception}},
		{"Gnome Rock", "gnome", []character.SpeciesTraitSelection{spSel("Gnomish Lineage", "Rock Gnome")}, spGrants(t, 0, map[string]ability.Ability{"Gnomish Lineage": ability.Charisma}), character.GnomeChoices{Lineage: character.GnomishRock, Ability: ability.Charisma}},
		{"Goliath Storm", "goliath", []character.SpeciesTraitSelection{spSel("Giant Ancestry", "Storm’s Thunder (Storm Giant)")}, character.SpeciesGrantSelections{}, character.GoliathChoices{Ancestry: character.AncestryStorm}},
		{"Tiefling Infernal", "tiefling", []character.SpeciesTraitSelection{spSel("Fiendish Legacy", "Infernal")}, spGrants(t, 0, map[string]ability.Ability{"Fiendish Legacy": ability.Intelligence}), character.TieflingChoices{Legacy: character.LegacyInfernal, Ability: ability.Intelligence}},
		{"Human", "human", nil, spGrants(t, content.SkillArcana, nil), character.HumanChoices{Skill: content.SkillArcana}},
		{"Dwarf", "dwarf", nil, character.SpeciesGrantSelections{}, character.NoSpeciesChoices{}},
		{"Halfling", "halfling", nil, character.SpeciesGrantSelections{}, character.NoSpeciesChoices{}},
		{"Orc", "orc", nil, character.SpeciesGrantSelections{}, character.NoSpeciesChoices{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.ParseSpeciesTraitChoices(c.species, c.selections, c.grants)
			if err != nil || got != c.want {
				t.Fatalf("Parse = %#v, %v; want %#v", got, err, c.want)
			}
			if _, err := character.NewSpeciesTraits(c.species, got); err != nil {
				t.Errorf("NewSpeciesTraits rejects the parsed choices: %v", err)
			}
			if character.MustSpeciesTraitChoices(c.species, c.selections, c.grants) != c.want {
				t.Error("MustSpeciesTraitChoices disagrees")
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-004 to FR-008, Table C unhappy rows - each wrong, missing or unexpected choice is one typed problem, all of them are listed in input order in the joined error, and the error matches srderr.ErrInvalidInput.
// Path: per case.
func TestParseSpeciesTraitChoicesProblems(t *testing.T) {
	t.Parallel()
	elfSel := []character.SpeciesTraitSelection{spSel("Elven Lineage", "Wood Elf")}
	elfAbility := map[string]ability.Ability{"Elven Lineage": ability.Wisdom}
	type p = character.SpeciesChoiceProblem
	for _, c := range []struct {
		name       string
		species    content.SpeciesID
		selections []character.SpeciesTraitSelection
		grants     character.SpeciesGrantSelections
		want       []p
	}{
		{"elf without an ability", "elf", elfSel, spGrants(t, content.SkillInsight, nil), []p{{Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemMissing}}},
		{"dwarf with an ability", "dwarf", nil, spGrants(t, 0, map[string]ability.Ability{"Elven Lineage": ability.Wisdom}), []p{{Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemUnexpected}}},
		{"halfling with an ancestry", "halfling", []character.SpeciesTraitSelection{spSel("Draconic Ancestry", "Red")}, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceDraconicAncestry, Kind: character.TraitProblemUnexpected}}},
		{"empty trait name for a species with no choice", "dwarf", []character.SpeciesTraitSelection{spSel("", "Gold")}, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceShape, Kind: character.TraitProblemUnexpected}}},
		{"unknown trait", "dwarf", []character.SpeciesTraitSelection{spSel("Sneaking", "Often")}, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceShape, Kind: character.TraitProblemUnexpected}}},
		{"dwarf with a skill", "dwarf", nil, spGrants(t, content.SkillArcana, nil), []p{{Name: character.ChoiceShape, Kind: character.TraitProblemUnexpected}}},
		{"elf skill Athletics", "elf", elfSel, spGrants(t, content.SkillAthletics, elfAbility), []p{{Name: character.ChoiceKeenSensesSkill, Kind: character.TraitProblemSkillNotAllowed}}},
		{"elf without a skill", "elf", elfSel, spGrants(t, 0, elfAbility), []p{{Name: character.ChoiceKeenSensesSkill, Kind: character.TraitProblemMissing}}},
		{"human without a skill", "human", nil, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceSkillfulSkill, Kind: character.TraitProblemMissing}}},
		{"tiefling Strength", "tiefling", []character.SpeciesTraitSelection{spSel("Fiendish Legacy", "Abyssal")}, spGrants(t, 0, map[string]ability.Ability{"Fiendish Legacy": ability.Strength}), []p{{Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemAbilityNotAllowed}}},
		{"dragonborn option Sonic", "dragonborn", []character.SpeciesTraitSelection{spSel("Draconic Ancestry", "Sonic")}, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceDraconicAncestry, Kind: character.TraitProblemOptionUnknown}}},
		{"dragonborn without an ancestry", "dragonborn", nil, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceDraconicAncestry, Kind: character.TraitProblemMissing}}},
		{"ancestry chosen twice", "dragonborn", []character.SpeciesTraitSelection{spSel("Draconic Ancestry", "Red"), spSel("Draconic Ancestry", "Blue")}, character.SpeciesGrantSelections{}, []p{{Name: character.ChoiceDraconicAncestry, Kind: character.TraitProblemDuplicate}}},
		{"ability for a lineage the species lacks", "goliath", []character.SpeciesTraitSelection{spSel("Giant Ancestry", "Hill’s Tumble (Hill Giant)")}, spGrants(t, 0, map[string]ability.Ability{"Giant Ancestry": ability.Wisdom}), []p{{Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemUnexpected}}},
		{"every problem", "elf", []character.SpeciesTraitSelection{spSel("Elven Lineage", "Moon Elf")}, spGrants(t, 0, map[string]ability.Ability{"Elven Lineage": ability.Strength}), []p{
			{Name: character.ChoiceElvenLineage, Kind: character.TraitProblemOptionUnknown},
			{Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemAbilityNotAllowed},
			{Name: character.ChoiceKeenSensesSkill, Kind: character.TraitProblemMissing}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.ParseSpeciesTraitChoices(c.species, c.selections, c.grants)
			if got != nil || err == nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("Parse = %#v, %v; want nil and an invalid input error", got, err)
			}
			ps := spProblems(err)
			if len(ps) != len(c.want) {
				t.Fatalf("problems = %v, want %v", ps, c.want)
			}
			for i := range ps {
				if ps[i] != c.want[i] || !errors.Is(err, c.want[i]) {
					t.Errorf("problem %d = %v, want %v", i, ps[i], c.want[i])
				}
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-007, FR-008, Table C "unknown species" - a species outside the nine is a typed error that matches ErrSpeciesBehaviourUnknown and srderr.ErrInvalidInput, carries the id, and never panics; MustSpeciesTraitChoices panics with that error.
// Path: unhappy (missing content).
func TestParseSpeciesTraitChoicesUnknownSpecies(t *testing.T) {
	t.Parallel()
	got, err := character.ParseSpeciesTraitChoices("house_species", nil, character.SpeciesGrantSelections{})
	var ue *character.SpeciesBehaviourUnknownError
	if got != nil || !errors.As(err, &ue) || ue.Species != "house_species" || !errors.Is(err, character.ErrSpeciesBehaviourUnknown) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("Parse = %v, %v", got, err)
	}
	if _, err := character.NewSpeciesTraits("", character.NoSpeciesChoices{}); !errors.Is(err, character.ErrSpeciesBehaviourUnknown) {
		t.Errorf("NewSpeciesTraits(\"\") = %v", err)
	}
	defer func() {
		if r, ok := recover().(error); !ok || !errors.Is(r, character.ErrSpeciesBehaviourUnknown) {
			t.Errorf("panic value = %v", r)
		}
	}()
	character.MustSpeciesTraitChoices("house_species", nil, character.SpeciesGrantSelections{})
}

// Requirement: doc/specs/species-trait-behaviour.md FR-001, FR-004, FR-005, FR-056, Table C "wrong variant" (SRD 5.2.1, pp. 84-86) - NewSpeciesTraits accepts exactly the variant of the species: another variant, a nil value and a part that is missing or not allowed are typed errors matching srderr.ErrInvalidInput.
// Path: per case.
func TestNewSpeciesTraitsValidates(t *testing.T) {
	t.Parallel()
	elf := character.ElfChoices{Lineage: character.ElvenDrow, Ability: ability.Charisma, KeenSensesSkill: content.SkillSurvival}
	for _, c := range []struct {
		name     string
		species  content.SpeciesID
		choices  character.SpeciesTraitChoices
		mismatch bool
		problem  []character.SpeciesChoiceProblem
	}{
		{"dwarf with Dragonborn choices", "dwarf", character.DragonbornChoices{Ancestry: character.DraconicRed}, true, nil},
		{"dragonborn with no choices", "dragonborn", character.NoSpeciesChoices{}, true, nil},
		{"human with Elf choices", "human", elf, true, nil},
		{"elf with Human choices", "elf", character.HumanChoices{Skill: content.SkillArcana}, true, nil},
		{"gnome with Tiefling choices", "gnome", character.TieflingChoices{}, true, nil},
		{"goliath with Gnome choices", "goliath", character.GnomeChoices{}, true, nil},
		{"tiefling with Goliath choices", "tiefling", character.GoliathChoices{}, true, nil},
		{"orc with a nil value", "orc", nil, true, nil},
		{"dragonborn zero ancestry", "dragonborn", character.DragonbornChoices{}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceDraconicAncestry, Kind: character.TraitProblemMissing}}},
		{"dragonborn undefined ancestry", "dragonborn", character.DragonbornChoices{Ancestry: 11}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceDraconicAncestry, Kind: character.TraitProblemOptionUnknown}}},
		{"elf with every part wrong", "elf", character.ElfChoices{Lineage: 9, Ability: ability.Dexterity, KeenSensesSkill: content.SkillArcana}, false, []character.SpeciesChoiceProblem{
			{Name: character.ChoiceElvenLineage, Kind: character.TraitProblemOptionUnknown}, {Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemAbilityNotAllowed}, {Name: character.ChoiceKeenSensesSkill, Kind: character.TraitProblemSkillNotAllowed}}},
		{"elf with nothing", "elf", character.ElfChoices{}, false, []character.SpeciesChoiceProblem{
			{Name: character.ChoiceElvenLineage, Kind: character.TraitProblemMissing}, {Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemMissing}, {Name: character.ChoiceKeenSensesSkill, Kind: character.TraitProblemMissing}}},
		{"gnome zero", "gnome", character.GnomeChoices{}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceGnomishLineage, Kind: character.TraitProblemMissing}, {Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemMissing}}},
		{"goliath undefined", "goliath", character.GoliathChoices{Ancestry: 7}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceGiantAncestry, Kind: character.TraitProblemOptionUnknown}}},
		{"tiefling zero", "tiefling", character.TieflingChoices{}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceFiendishLegacy, Kind: character.TraitProblemMissing}, {Name: character.ChoiceSpellcastingAbility, Kind: character.TraitProblemMissing}}},
		{"human zero", "human", character.HumanChoices{}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceSkillfulSkill, Kind: character.TraitProblemMissing}}},
		{"human undefined skill", "human", character.HumanChoices{Skill: 99}, false, []character.SpeciesChoiceProblem{{Name: character.ChoiceSkillfulSkill, Kind: character.TraitProblemSkillNotAllowed}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := character.NewSpeciesTraits(c.species, c.choices)
			if !tr.IsZero() || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewSpeciesTraits = %v, %v", tr, err)
			}
			var me *character.SpeciesMismatchError
			if errors.As(err, &me) != c.mismatch || c.mismatch && (me.Species != c.species || me.Choices != c.choices) {
				t.Errorf("mismatch error = %v", err)
			}
			ps := spProblems(err)
			if len(ps) != len(c.problem) {
				t.Fatalf("problems = %v, want %v", ps, c.problem)
			}
			for i := range ps {
				if ps[i] != c.problem[i] {
					t.Errorf("problem %d = %v, want %v", i, ps[i], c.problem[i])
				}
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-001 and architecture.md section 6 - MustSpeciesTraits panics with the error NewSpeciesTraits returns, and only then.
// Path: unhappy (invalid input).
func TestMustSpeciesTraitsPanics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r, ok := recover().(error); !ok || !errors.Is(r, srderr.ErrInvalidInput) {
			t.Errorf("MustSpeciesTraits panic = %v", r)
		}
	}()
	character.MustSpeciesTraits("dwarf", nil)
}

// Requirement: doc/specs/species-trait-behaviour.md SpeciesChoiceProblem, FR-006 - the names and kinds have snake_case strings, an undefined value is "unknown", and a problem is a comparable value that matches an equal problem with errors.Is.
// Path: happy.
func TestSpeciesChoiceProblemValues(t *testing.T) {
	t.Parallel()
	p := character.SpeciesChoiceProblem{Name: character.ChoiceKeenSensesSkill, Kind: character.TraitProblemSkillNotAllowed}
	if p.Name.String() != "keen_senses_skill" || p.Kind.String() != "skill_not_allowed" || p.Error() == "" {
		t.Errorf("problem = %v", p)
	}
	names := []character.SpeciesChoiceName{character.ChoiceDraconicAncestry, character.ChoiceElvenLineage, character.ChoiceGnomishLineage, character.ChoiceGiantAncestry, character.ChoiceFiendishLegacy,
		character.ChoiceSpellcastingAbility, character.ChoiceKeenSensesSkill, character.ChoiceSkillfulSkill, character.ChoiceReplacementCantrip, character.ChoiceShape}
	for i, n := range names {
		if n.String() == "unknown" || int(n) != i+1 {
			t.Errorf("name %d = %q", i, n)
		}
	}
	kinds := []character.SpeciesChoiceKind{character.TraitProblemMissing, character.TraitProblemUnexpected, character.TraitProblemOptionUnknown, character.TraitProblemAbilityNotAllowed,
		character.TraitProblemSkillNotAllowed, character.TraitProblemNotACantrip, character.TraitProblemNotOnSpellList, character.TraitProblemDuplicate}
	for i, k := range kinds {
		if k.String() == "unknown" || int(k) != i+1 {
			t.Errorf("kind %d = %q", i, k)
		}
	}
	if character.SpeciesChoiceName(0).String() != "unknown" || character.SpeciesChoiceKind(99).String() != "unknown" {
		t.Error("an undefined value has a name")
	}
	if !errors.Is(p, srderr.ErrInvalidInput) || errors.Is(p, character.SpeciesChoiceProblem{}) {
		t.Error("problem matching is wrong")
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-007, testing.md §10 - ParseSpeciesTraitChoices never panics, whatever the species, selections, ability or skill; a failure is always an error that matches srderr.ErrInvalidInput and is never empty.
// Path: unhappy (invalid input).
func FuzzParseSpeciesTraitChoices(f *testing.F) {
	for _, s := range []string{"dragonborn", "elf", "gnome", "goliath", "tiefling", "human", "dwarf", "house"} {
		f.Add(s, "Elven Lineage", "High Elf", 3, 7)
	}
	f.Add("elf", "", "", 0, 0)
	f.Add("goliath", "Giant Ancestry", "Cloud’s Jaunt (Cloud Giant)", 9, 99)
	f.Fuzz(func(t *testing.T, species, trait, option string, ab, skill int) {
		grants, err := character.NewSpeciesGrantSelections(content.Skill(skill), "", map[string]ability.Ability{trait: ability.Ability(ab)})
		if err != nil {
			grants = character.SpeciesGrantSelections{}
		}
		got, err := character.ParseSpeciesTraitChoices(content.SpeciesID(species), []character.SpeciesTraitSelection{spSel(trait, option)}, grants)
		if err != nil && (got != nil || !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Fatalf("Parse = %#v, %v", got, err)
		}
		if err == nil {
			if _, e := character.NewSpeciesTraits(content.SpeciesID(species), got); e != nil {
				t.Fatalf("parsed choices rejected by NewSpeciesTraits: %v", e)
			}
		}
	})
}

// Requirement: doc/specs/species-trait-behaviour.md FR-001, FR-008 - the two typed errors of NewSpeciesTraits render a message, unwrap to srderr.ErrInvalidInput and carry their data; the text is not a contract.
// Path: unhappy (invalid input).
func TestSpeciesTraitsErrorsRender(t *testing.T) {
	t.Parallel()
	_, unknown := character.NewSpeciesTraits("house_species", character.NoSpeciesChoices{})
	_, mismatch := character.NewSpeciesTraits("dwarf", character.HumanChoices{})
	for name, err := range map[string]error{"unknown": unknown, "mismatch": mismatch} {
		if err == nil || err.Error() == "" || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
