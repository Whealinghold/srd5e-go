package content_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// fullSpecies has all three option variants: Damage on trait T, a lineage on U
// and a text option with spells on V.
const fullSpecies = `{"id":"fixture","name":"Fixture","source":{"document":"SRD 5.2.1","section":"S","page":84},"creatureType":"Humanoid","sizes":["Small","Medium"],"sizeText":"s","speed":30,` +
	`"traits":[{"name":"T","text":"a"},{"name":"U","text":"b"},{"name":"V","text":"c"}],"choices":[` +
	`{"traitName":"T","options":[{"kind":"damageType","name":"D","damageType":"Acid"}]},` +
	`{"traitName":"U","options":[{"kind":"lineage","name":"L","levelBenefits":[{"level":1,"text":"x","spellId":"spell_a"},{"level":3,"text":"y","spellId":"spell_b"},{"level":5,"text":"z","spellId":"spell_c"}]}]},` +
	`{"traitName":"V","options":[{"kind":"text","name":"X","text":"t","spellIds":["spell_d"]}]}]}`

const choiceBackground = `{"id":"bg_two","name":"Two","abilityScores":["wisdom","charisma","intelligence"],"feat":{"featId":"feat_a","choice":"Word"},"skillProficiencies":["Arcana","History"],"toolProficiency":{"kind":"choice","category":"Set"},"equipmentOptions":{"A":{"items":[{"name":"Rope","quantity":2}],"gold":8},"B":{"items":[],"gold":50}}}`

const fullFeat = `{"id":"feat_full","name":"Full","source":{"document":"SRD 5.2.1","section":"S","page":87},"category":"General","prerequisites":[{"kind":"level","minimum":4},` +
	`{"kind":"abilityScore","abilities":["strength","dexterity"],"minimum":13},{"kind":"feature","name":"F"}],"benefits":[{"name":"N","text":"x"},{"text":"y"}],"repeatable":true,"repeatableText":"r"}`

// edit parses doc, applies f to the value at the slash-separated path (array
// indexes are numbers) and renders it again; only used on fixtures.
func edit(t *testing.T, doc, path string, f func(parent any, key string)) string {
	t.Helper()
	var root any
	if err := json.Unmarshal([]byte(doc), &root); err != nil {
		t.Fatal(err)
	}
	segs := strings.Split(path, "/")
	cur := root
	for _, s := range segs[:len(segs)-1] {
		if i, err := strconv.Atoi(s); err == nil {
			cur = cur.([]any)[i]
		} else {
			cur = cur.(map[string]any)[s]
		}
	}
	f(cur, segs[len(segs)-1])
	out, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func removeKey(t *testing.T, doc, path string) string {
	t.Helper()
	return edit(t, doc, path, func(parent any, key string) { delete(parent.(map[string]any), key) })
}

func nullKey(t *testing.T, doc, path string) string {
	t.Helper()
	return edit(t, doc, path, func(parent any, key string) { parent.(map[string]any)[key] = nil })
}

func requiredPaths() []struct {
	array, entry string
	paths        []string
} {
	return []struct {
		array, entry string
		paths        []string
	}{
		{"species", fullSpecies, []string{
			"id", "name", "creatureType", "sizes", "speed", "traits", "traits/0/name", "traits/0/text",
			"choices/0/traitName", "choices/0/options", "choices/0/options/0/kind", "choices/0/options/0/name", "choices/0/options/0/damageType",
			"choices/1/options/0/levelBenefits", "choices/1/options/0/levelBenefits/1/level", "choices/1/options/0/levelBenefits/1/text", "choices/1/options/0/levelBenefits/1/spellId",
			"choices/2/options/0/text",
		}},
		{"backgrounds", backgroundJSON, []string{
			"id", "name", "abilityScores", "feat", "feat/featId", "skillProficiencies", "toolProficiency", "toolProficiency/kind", "toolProficiency/name",
			"equipmentOptions", "equipmentOptions/A", "equipmentOptions/B", "equipmentOptions/A/items", "equipmentOptions/A/gold", "equipmentOptions/B/gold",
			"equipmentOptions/A/items/0/name", "equipmentOptions/A/items/0/quantity",
		}},
		{"backgrounds", choiceBackground, []string{"toolProficiency/category"}},
		{"feats", fullFeat, []string{
			"category", "benefits", "benefits/0/text", "repeatable", "repeatableText",
			"prerequisites/0/kind", "prerequisites/0/minimum", "prerequisites/1/abilities", "prerequisites/1/minimum", "prerequisites/2/name",
		}},
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-019, FR-013 and doc/specs/content-species-backgrounds-feats.md
// section 5 - each required field of every shape, nested ones included, is one missingField at its path when
// absent, and one wrongType when null, with nothing cascading; each optional field is wrongType when null.
// Path: per case.
func TestDecodeRequiredFieldsAreRequiredAndNullIsWrongType(t *testing.T) {
	t.Parallel()
	for _, e := range requiredPaths() {
		if got := decodeProblems(wrapDoc(e.array, e.entry)); got != nil {
			t.Fatalf("the %s fixture is not valid: %q", e.array, got)
		}
		for _, path := range e.paths {
			t.Run(e.array+"/"+path, func(t *testing.T) {
				t.Parallel()
				want := "/" + e.array + "/0/" + path
				if got := decodeProblems(wrapDoc(e.array, removeKey(t, e.entry, path))); len(got) != 1 || got[0] != want+" missingField" {
					t.Errorf("absent: problems = %q, want only %s missingField [FR-019, unhappy (missing content)]", got, want)
				}
				if got := decodeProblems(wrapDoc(e.array, nullKey(t, e.entry, path))); len(got) != 1 || got[0] != want+" wrongType" {
					t.Errorf("null: problems = %q, want only %s wrongType [FR-020, unhappy (invalid input)]", got, want)
				}
			})
		}
	}
	for _, path := range []string{"source", "sizeText", "choices", "choices/2/options/0/spellIds"} {
		if got := decodeProblems(wrapDoc("species", nullKey(t, fullSpecies, path))); len(got) != 1 || got[0] != "/species/0/"+path+" wrongType" {
			t.Errorf("null optional %s: problems = %q, want one wrongType", path, got)
		}
	}
	for _, path := range []string{"feat/choice"} {
		if got := decodeProblems(wrapDoc("backgrounds", nullKey(t, choiceBackground, path))); len(got) != 1 || got[0] != "/backgrounds/0/"+path+" wrongType" {
			t.Errorf("null optional %s: problems = %q, want one wrongType", path, got)
		}
	}
	for _, path := range []string{"prerequisites", "benefits/0/name", "repeatableText"} {
		if got := decodeProblems(wrapDoc("feats", nullKey(t, fullFeat, path))); len(got) != 1 || got[0] != "/feats/0/"+path+" wrongType" {
			t.Errorf("null optional %s: problems = %q, want one wrongType", path, got)
		}
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-014, FR-016, FR-060, FR-061 - an element or
// field that is itself wrong is one problem, and a bad part never raises a second problem about its siblings.
// Path: per case.
func TestDecodeBadPartsDoNotCascade(t *testing.T) {
	t.Parallel()
	const so = "/species/0/choices/0/options"
	sp := func(choices string) string { return wrapDoc("species", speciesJSON(choices)) }
	runDocCases(t, []docCase{
		{"two blank trait names are not repeats", "FR-014", "unhappy (invalid input)",
			wrapDoc("species", mutate(mutate(speciesJSON(""), `"name":"T"`, `"name":" "`), `"name":"U"`, `"name":" "`)),
			[]string{"/species/0/traits/0/name invalidValue", "/species/0/traits/1/name invalidValue"}},
		{"blank trait name in a species with a choice", "FR-060", "unhappy (invalid input)",
			wrapDoc("species", mutate(speciesJSON(choiceJSON(textOpt("A"))), `"name":"T"`, `"name":" "`)), []string{"/species/0/traits/0/name invalidValue"}},
		{"blank choice trait name", "FR-060", "unhappy (invalid input)",
			wrapDoc("species", mutate(speciesJSON(choiceJSON(textOpt("A"))), `"traitName":"T"`, `"traitName":" "`)), []string{"/species/0/choices/0/traitName invalidValue"}},
		{"choice trait name that sorts before the traits", "FR-060", "unhappy (rule interaction)",
			wrapDoc("species", mutate(speciesJSON(choiceJSON(textOpt("A"))), `"traitName":"T"`, `"traitName":"A"`)), []string{"/species/0/choices/0/traitName constraintViolation"}},
		{"two blank option names are not repeats", "FR-016", "unhappy (invalid input)",
			sp(choiceJSON(mutate(textOpt("A"), `"name":"A"`, `"name":" "`), mutate(textOpt("B"), `"name":"B"`, `"name":" "`))), []string{so + "/0/name invalidValue", so + "/1/name invalidValue"}},
		{"text option then damage option", "FR-016", "unhappy (rule interaction)", sp(choiceJSON(textOpt("A"), damageOpt("B", "Acid"))), []string{so + "/1/kind constraintViolation"}},
		{"lineage then text option", "FR-016", "unhappy (rule interaction)", sp(choiceJSON(lineageOpt("L", 1, 3, 5), textOpt("A"))), []string{so + "/1/kind constraintViolation"}},
		{"text then lineage option", "FR-016", "unhappy (rule interaction)", sp(choiceJSON(textOpt("A"), lineageOpt("L", 1, 3, 5))), []string{so + "/1/kind constraintViolation"}},
		{"damage then lineage option", "FR-016", "unhappy (rule interaction)", sp(choiceJSON(damageOpt("A", "Acid"), lineageOpt("L", 1, 3, 5))), []string{so + "/1/kind constraintViolation"}},
		{"option that is not an object", "FR-016", "unhappy (invalid input)", sp(choiceJSON(`1`, textOpt("A"))), []string{so + "/0 wrongType"}},
		{"level benefit that is not an object", "FR-061", "unhappy (invalid input)",
			sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `{"level":1,"text":"b","spellId":"spell_0"}`, `1`))), []string{so + "/0/levelBenefits/0 wrongType"}},
		{"unknown tool kind sorting above specific", "schema FR-119", "unhappy (invalid input)",
			wrapDoc("backgrounds", mutate(backgroundJSON, `"specific"`, `"zzz"`)), []string{"/backgrounds/0/toolProficiency/kind invalidValue"}},
		{"unknown tool kind sorting below choice", "schema FR-119", "unhappy (invalid input)",
			wrapDoc("backgrounds", mutate(backgroundJSON, `"specific"`, `"a"`)), []string{"/backgrounds/0/toolProficiency/kind invalidValue"}},
		{"item that is not an object", "FR-030", "unhappy (invalid input)",
			wrapDoc("backgrounds", mutate(backgroundJSON, `{"name":"Rope","quantity":2}`, `1`)), []string{"/backgrounds/0/equipmentOptions/A/items/0 wrongType"}},
		{"feat benefit that is not an object", "FR-063", "unhappy (invalid input)",
			wrapDoc("feats", mutate(featJSON, `{"text":"x"}`, `1`)), []string{"/feats/0/benefits/0 wrongType"}},
		{"prerequisite that is not an object", "schema FR-118", "unhappy (invalid input)",
			wrapDoc("feats", mutate(featJSON, `"benefits"`, `"prerequisites":[1],"benefits"`)), []string{"/feats/0/prerequisites/0 wrongType"}},
	})
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-021, FR-025 to FR-031, FR-040 and schema FR-029,
// FR-094 - a document decodes to exactly the entry its constructors build from the same values, for every
// variant of every closed family, so no part is dropped, reordered or rewritten.
// Path: happy.
func TestDecodeBuildsTheEntriesTheConstructorsBuild(t *testing.T) {
	t.Parallel()
	src := content.MustSourceCitation("SRD 5.2.1", "S", 84)
	species := content.MustSpecies("fixture", "Fixture", src, content.SpeciesFields{
		CreatureType: content.CreatureTypeHumanoid, Sizes: []content.CreatureSize{content.SizeSmall, content.SizeMedium}, SizeText: "s", Speed: 30,
		Traits: []content.NamedText{content.MustNamedText("T", "a"), content.MustNamedText("U", "b"), content.MustNamedText("V", "c")},
		Choices: []content.SpeciesChoice{
			content.MustSpeciesChoice("T", []content.ChoiceOption{content.MustDamageTypeOption("D", content.DamageAcid)}),
			content.MustSpeciesChoice("U", []content.ChoiceOption{content.MustLineageOption("L", []content.LevelBenefit{
				content.MustLevelBenefit(1, "x", "spell_a"), content.MustLevelBenefit(3, "y", "spell_b"), content.MustLevelBenefit(5, "z", "spell_c"),
			})}),
			content.MustSpeciesChoice("V", []content.ChoiceOption{content.MustTextOption("X", "t", []content.SpellID{"spell_d"})}),
		},
	})
	rope := content.MustEquipmentItem("Rope", 2)
	bgA := content.MustBackground("fixture", "Fixture", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores: []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution}, Feat: content.MustBackgroundFeatReference("feat_a", ""),
		SkillProficiencies: []content.Skill{content.SkillAthletics, content.SkillStealth}, ToolProficiency: mustSpecificTool("Tool"),
		EquipmentOptions: content.NewEquipmentOptions(content.MustEquipmentOption([]content.EquipmentItem{rope}, 10), content.MustEquipmentOption(nil, 50)),
	})
	bgB := content.MustBackground("bg_two", "Two", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores: []ability.Ability{ability.Wisdom, ability.Charisma, ability.Intelligence}, Feat: content.MustBackgroundFeatReference("feat_a", "Word"),
		SkillProficiencies: []content.Skill{content.SkillArcana, content.SkillHistory}, ToolProficiency: mustChoiceTool("Set"),
		EquipmentOptions: content.NewEquipmentOptions(content.MustEquipmentOption([]content.EquipmentItem{rope}, 8), content.MustEquipmentOption(nil, 50)),
	})
	abilityPre, err := content.NewAbilityScorePrerequisite([]ability.Ability{ability.Strength, ability.Dexterity}, 13)
	if err != nil {
		t.Fatal(err)
	}
	featPre, err := content.NewFeaturePrerequisite("F")
	if err != nil {
		t.Fatal(err)
	}
	full := content.MustFeat("feat_full", "Full", content.MustSourceCitation("SRD 5.2.1", "S", 87), content.FeatFields{
		Category: content.FeatGeneral, Prerequisites: []content.Prerequisite{mustLevel(4), abilityPre, featPre},
		Benefits: []content.FeatBenefit{content.MustFeatBenefit("N", "x"), content.MustFeatBenefit("", "y")}, Repeatable: true, RepeatableText: "r",
	})
	plain := content.MustFeat("feat_a", "Feat A", content.SourceCitation{}, content.FeatFields{Category: content.FeatOrigin, Benefits: []content.FeatBenefit{content.MustFeatBenefit("", "x")}})
	want := content.MustSet(species, bgA, bgB, full, plain)
	doc := `{"schemaVersion":1,"species":[` + fullSpecies + `],"backgrounds":[` + backgroundJSON + `,` + choiceBackground + `],"feats":[` + fullFeat + `,` + featJSON + `]}`
	if got := decodeSet(t, doc); !got.Equal(want) {
		t.Errorf("decoded set differs from the constructed one:\n got %#v\nwant %#v", got.Entries(), want.Entries())
	}
}
