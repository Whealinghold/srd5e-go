package content_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// The fixtures are synthetic: made-up ids, names and texts, never SRD data.

const specSpec = "doc/specs/content-species-backgrounds-feats.md"

// mutate replaces the first old in base with repl and fails loudly (a test
// bug) when old is absent, so a case cannot silently test nothing.
func mutate(base, old, repl string) string {
	if !strings.Contains(base, old) {
		panic("fixture does not contain " + old)
	}
	return strings.Replace(base, old, repl, 1)
}

func wrapDoc(array, entry string) string {
	return `{"schemaVersion":1,"` + array + `":[` + entry + `]}`
}

func speciesJSON(choices string) string {
	s := `{"id":"fixture","name":"Fixture","creatureType":"Humanoid","sizes":["Medium"],"speed":30,"traits":[{"name":"T","text":"x"},{"name":"U","text":"y"}]`
	if choices != "" {
		s += `,"choices":[` + choices + `]`
	}
	return s + `}`
}

func choiceJSON(options ...string) string {
	return `{"traitName":"T","options":[` + strings.Join(options, ",") + `]}`
}

func damageOpt(name, dt string) string {
	return `{"kind":"damageType","name":"` + name + `","damageType":"` + dt + `"}`
}

func textOpt(name string, spells ...string) string {
	s := `{"kind":"text","name":"` + name + `","text":"x"`
	if len(spells) > 0 {
		s += `,"spellIds":["` + strings.Join(spells, `","`) + `"]`
	}
	return s + `}`
}

func lineageOpt(name string, levels ...int) string {
	var bs []string
	for i, l := range levels {
		bs = append(bs, `{"level":`+strconv.Itoa(l)+`,"text":"b","spellId":"spell_`+strconv.Itoa(i)+`"}`)
	}
	return `{"kind":"lineage","name":"` + name + `","levelBenefits":[` + strings.Join(bs, ",") + `]}`
}

const backgroundJSON = `{"id":"fixture","name":"Fixture","abilityScores":["strength","dexterity","constitution"],"feat":{"featId":"feat_a"},"skillProficiencies":["Athletics","Stealth"],"toolProficiency":{"kind":"specific","name":"Tool"},"equipmentOptions":{"A":{"items":[{"name":"Rope","quantity":2}],"gold":10},"B":{"items":[],"gold":50}}}`

const featJSON = `{"id":"feat_a","name":"Feat A","category":"Origin","benefits":[{"text":"x"}],"repeatable":false}`

func decodeProblems(doc string) []string {
	_, err := content.JSONDecoder{}.Decode([]byte(doc))
	var out []string
	for _, p := range content.Problems(err) {
		out = append(out, p.Path()+" "+p.Kind().String())
	}
	return out
}

type docCase struct {
	name, requirement, path string
	doc                     string
	want                    []string
}

func runDocCases(t *testing.T, cases []docCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := decodeProblems(c.doc); !slices.Equal(got, c.want) {
				t.Errorf("problems = %q, want %q [%s, %s]", got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestDecodeSpeciesHeader(t *testing.T) {
	t.Parallel()
	s := func(old, repl string) string { return wrapDoc("species", mutate(speciesJSON(""), old, repl)) }
	const p = "/species/0/"
	runDocCases(t, []docCase{
		{"valid", specSpec + " FR-010 to FR-014", "happy", wrapDoc("species", speciesJSON("")), nil},
		{"source present", "FR-007, FR-008", "happy", s(`"name":"Fixture",`, `"name":"Fixture","source":{"document":"SRD 5.2.1","section":"S","page":84},`), nil},
		{"blank source section", "FR-008", "unhappy (invalid input)", s(`"name":"Fixture",`, `"name":"Fixture","source":{"document":"SRD 5.2.1","section":" ","page":84},`), []string{p + "source/section invalidValue"}},
		{"source page 0", "FR-008", "unhappy (boundary)", s(`"name":"Fixture",`, `"name":"Fixture","source":{"document":"SRD 5.2.1","section":"S","page":0},`), []string{p + "source/page outOfRange"}},
		{"source page -1", "FR-008", "unhappy (boundary)", s(`"name":"Fixture",`, `"name":"Fixture","source":{"document":"SRD 5.2.1","section":"S","page":-1},`), []string{p + "source/page outOfRange"}},
		{"missing creatureType", "FR-010", "unhappy (missing content)", s(`"creatureType":"Humanoid",`, ``), []string{p + "creatureType missingField"}},
		{"missing sizes", "FR-010", "unhappy (missing content)", s(`"sizes":["Medium"],`, ``), []string{p + "sizes missingField"}},
		{"missing speed", "FR-010", "unhappy (missing content)", s(`"speed":30,`, ``), []string{p + "speed missingField"}},
		{"missing traits", "FR-010", "unhappy (missing content)", s(`,"traits":[{"name":"T","text":"x"},{"name":"U","text":"y"}]`, ``), []string{p + "traits missingField"}},
		{"unknown creature type", "FR-057", "unhappy (invalid input)", s(`"Humanoid"`, `"Robot"`), []string{p + "creatureType invalidValue"}},
		{"no sizes", "FR-012", "unhappy (boundary)", s(`["Medium"]`, `[]`), []string{p + "sizes outOfRange"}},
		{"three sizes, no element problems", "FR-012", "unhappy (boundary)", s(`["Medium"]`, `["Large","Large","Large"]`), []string{p + "sizes outOfRange"}},
		{"two sizes", "FR-012", "happy", s(`["Medium"]`, `["Small","Medium"]`), nil},
		{"repeated size at the later index", "FR-012", "unhappy (boundary)", s(`["Medium"]`, `["Small","Small"]`), []string{p + "sizes/1 duplicate"}},
		{"Large is not allowed", "FR-012", "unhappy (boundary)", s(`["Medium"]`, `["Large"]`), []string{p + "sizes/0 invalidValue"}},
		{"unknown size", "FR-012", "unhappy (invalid input)", s(`["Medium"]`, `["Huge2"]`), []string{p + "sizes/0 invalidValue"}},
		{"speed zero", "FR-013", "unhappy (boundary)", s(`"speed":30`, `"speed":0`), nil},
		{"negative speed", "FR-013", "unhappy (boundary)", s(`"speed":30`, `"speed":-5`), []string{p + "speed outOfRange"}},
		{"speed beyond the cap", "FR-013", "unhappy (boundary)", s(`"speed":30`, `"speed":2147483648`), []string{p + "speed outOfRange"}},
		{"fractional speed", "FR-013", "unhappy (boundary)", s(`"speed":30`, `"speed":30.5`), []string{p + "speed wrongType"}},
		{"string speed", "FR-013", "unhappy (invalid input)", s(`"speed":30`, `"speed":"30"`), []string{p + "speed wrongType"}},
		{"empty traits", "FR-014", "unhappy (boundary)", s(`[{"name":"T","text":"x"},{"name":"U","text":"y"}]`, `[]`), []string{p + "traits outOfRange"}},
		{"blank trait name", "FR-014", "unhappy (invalid input)", s(`"name":"T"`, `"name":" "`), []string{p + "traits/0/name invalidValue"}},
		{"blank trait text", "FR-014", "unhappy (invalid input)", s(`"text":"x"`, `"text":" "`), []string{p + "traits/0/text invalidValue"}},
		{"repeated trait name", "FR-014", "unhappy (invalid input)", s(`"name":"U"`, `"name":"T"`), []string{p + "traits/1/name duplicate"}},
		{"trait with an extra key", "FR-116", "unhappy (invalid input)", s(`"text":"x"`, `"text":"x","note":"n"`), []string{p + "traits/0/note unknownField"}},
		{"sizeText present", "FR-074", "happy", s(`"speed":30`, `"sizeText":"s","speed":30`), nil},
		{"blank sizeText", "FR-074", "unhappy (invalid input)", s(`"speed":30`, `"sizeText":" ","speed":30`), []string{p + "sizeText invalidValue"}},
		{"three independent problems in field order", "FR-057", "unhappy (invalid input)",
			s(`"creatureType":"Humanoid","sizes":["Medium"],"speed":30,"traits":[{"name":"T"`, `"creatureType":"Robot","sizes":["Medium"],"traits":[{"name":" "`),
			[]string{p + "creatureType invalidValue", p + "speed missingField", p + "traits/0/name invalidValue"}},
		{"key outside the schema", "FR-058", "unhappy (invalid input)", s(`"speed":30`, `"race":"x","speed":30`), []string{p + "race unknownField"}},
		{"repeated key", "FR-058", "unhappy (invalid input)", s(`"speed":30`, `"speed":30,"speed":31`), []string{p + "speed duplicate"}},
		{"duplicate id", "FR-005", "unhappy (invalid input)",
			`{"schemaVersion":1,"species":[` + speciesJSON("") + `,` + speciesJSON("") + `]}`, []string{"/species/1/id duplicate"}},
		{"entry is not an object", "FR-013 of the schema", "unhappy (invalid input)", `{"schemaVersion":1,"species":[1]}`, []string{"/species/0 wrongType"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeSpeciesChoices(t *testing.T) {
	t.Parallel()
	const p = "/species/0/choices/0/options"
	sp := func(choices string) string { return wrapDoc("species", speciesJSON(choices)) }
	runDocCases(t, []docCase{
		{"empty choices list", "SX-1", "happy", sp(``), nil},
		{"damage options", specSpec + " FR-016, FR-017", "happy", sp(choiceJSON(damageOpt("A", "Acid"), damageOpt("B", "Fire"))), nil},
		{"unknown damage type", "FR-017", "unhappy (invalid input)", sp(choiceJSON(damageOpt("A", "Sonic"))), []string{p + "/0/damageType invalidValue"}},
		{"text key on a damage option", "FR-017", "unhappy (invalid input)",
			sp(choiceJSON(mutate(damageOpt("A", "Acid"), `"name":"A"`, `"name":"A","text":"x"`))), []string{p + "/0/text unknownField"}},
		{"damage option without damageType", "FR-016", "unhappy (missing content)",
			sp(choiceJSON(`{"kind":"damageType","name":"A"}`)), []string{p + "/0/damageType missingField"}},
		{"lineage options", "FR-016, FR-018", "happy", sp(choiceJSON(lineageOpt("L", 1, 3, 5))), nil},
		{"lineage with two levels", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 1, 3))), []string{p + "/0/levelBenefits outOfRange"}},
		{"lineage with four levels", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 1, 3, 5, 5))), []string{p + "/0/levelBenefits outOfRange"}},
		{"repeated level", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 1, 3, 3))), []string{p + "/0/levelBenefits/2/level duplicate"}},
		{"level 2", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 1, 2, 5))), []string{p + "/0/levelBenefits/1/level invalidValue"}},
		{"level 0 and level 6", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 0, 3, 6))), []string{p + "/0/levelBenefits/0/level invalidValue", p + "/0/levelBenefits/2/level invalidValue"}},
		{"levels out of order", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 1, 5, 3))), []string{p + "/0/levelBenefits constraintViolation"}},
		{"levels in descending order", "FR-061", "unhappy (boundary)", sp(choiceJSON(lineageOpt("L", 5, 3, 1))), []string{p + "/0/levelBenefits constraintViolation"}},
		{"level wrapping to 1 on 32-bit", "FR-061", "unhappy (boundary)", sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `"level":1`, `"level":4294967297`))), []string{p + "/0/levelBenefits/0/level invalidValue"}},
		{"level wrapping to 3 on 32-bit", "FR-061", "unhappy (boundary)", sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `"level":1`, `"level":4294967299`))), []string{p + "/0/levelBenefits/0/level invalidValue"}},
		{"fractional level", "FR-061", "unhappy (invalid input)", sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `"level":1`, `"level":1.5`))), []string{p + "/0/levelBenefits/0/level wrongType"}},
		{"level benefit without spellId", "FR-061", "unhappy (missing content)",
			sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `,"spellId":"spell_0"`, ``))), []string{p + "/0/levelBenefits/0/spellId missingField"}},
		{"level benefit with a text-option key", "FR-061", "unhappy (missing content)",
			sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `"level":1,`, `"level":1,"extra":1,`))), []string{p + "/0/levelBenefits/0/extra unknownField"}},
		{"lineage without levelBenefits", "FR-016", "unhappy (missing content)", sp(choiceJSON(`{"kind":"lineage","name":"L"}`)), []string{p + "/0/levelBenefits missingField"}},
		{"text key on a lineage option", "FR-016", "unhappy (invalid input)",
			sp(choiceJSON(mutate(lineageOpt("L", 1, 3, 5), `"name":"L"`, `"name":"L","text":"x"`))), []string{p + "/0/text unknownField"}},
		{"text options with and without spells", "FR-020, FR-021", "happy", sp(choiceJSON(textOpt("A"), textOpt("B", "spell_a", "spell_b"))), nil},
		{"blank spell id", "FR-021", "unhappy (invalid input)", sp(choiceJSON(mutate(textOpt("A", "spell_a"), `"spell_a"`, `""`))), []string{p + "/0/spellIds/0 invalidValue"}},
		{"text option without text", "FR-016", "unhappy (missing content)", sp(choiceJSON(`{"kind":"text","name":"A"}`)), []string{p + "/0/text missingField"}},
		{"no options", "FR-016", "unhappy (boundary)", sp(choiceJSON()), []string{p + " outOfRange"}},
		{"repeated option name", "FR-016", "unhappy (invalid input)", sp(choiceJSON(textOpt("A"), textOpt("A"))), []string{p + "/1/name duplicate"}},
		{"mixed variants", "FR-016, FR-017", "unhappy (rule interaction)", sp(choiceJSON(damageOpt("A", "Acid"), textOpt("B"))), []string{p + "/1/kind constraintViolation"}},
		{"unknown kind", "FR-016", "unhappy (invalid input)", sp(choiceJSON(`{"kind":"mystery","name":"A","text":"x"}`)), []string{p + "/0/kind invalidValue"}},
		{"unknown kind with the fields of every variant", "FR-016", "unhappy (invalid input)", sp(choiceJSON(`{"kind":"mystery","name":"A","damageType":"Acid","levelBenefits":[],"text":"x","spellIds":[]}`)), []string{p + "/0/kind invalidValue"}},
		{"missing kind", "FR-016", "unhappy (missing content)", sp(choiceJSON(`{"name":"A","text":"x"}`)), []string{p + "/0/kind missingField"}},
		{"trait name no trait has", "FR-060", "unhappy (rule interaction)",
			wrapDoc("species", mutate(speciesJSON(choiceJSON(textOpt("A"))), `"traitName":"T"`, `"traitName":"Z"`)), []string{"/species/0/choices/0/traitName constraintViolation"}},
		{"choice with an unknown key", "SX-1", "unhappy (invalid input)",
			sp(mutate(choiceJSON(textOpt("A")), `"traitName":"T"`, `"traitName":"T","pick":1`)), []string{"/species/0/choices/0/pick unknownField"}},
		{"choice that is not an object", "SX-1", "unhappy (invalid input)", sp(`1`), []string{"/species/0/choices/0 wrongType"}},
		{"traits problem does not cascade into the link check", "FR-060", "unhappy (rule interaction)",
			wrapDoc("species", mutate(speciesJSON(choiceJSON(textOpt("A"))), `"name":"T"`, `"name":" "`)), []string{"/species/0/traits/0/name invalidValue"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeBackgrounds(t *testing.T) {
	t.Parallel()
	b := func(old, repl string) string { return wrapDoc("backgrounds", mutate(backgroundJSON, old, repl)) }
	const p = "/backgrounds/0/"
	runDocCases(t, []docCase{
		{"valid", specSpec + " FR-024 to FR-031", "happy", wrapDoc("backgrounds", backgroundJSON), nil},
		{"two abilities", "FR-024", "unhappy (boundary)", b(`["strength","dexterity","constitution"]`, `["strength","dexterity"]`), []string{p + "abilityScores outOfRange"}},
		{"four abilities", "FR-024", "unhappy (boundary)", b(`["strength","dexterity","constitution"]`, `["strength","dexterity","constitution","wisdom"]`), []string{p + "abilityScores outOfRange"}},
		{"repeated ability", "FR-024", "unhappy (boundary)", b(`"dexterity"`, `"strength"`), []string{p + "abilityScores/1 duplicate"}},
		{"unknown ability", "FR-024", "unhappy (invalid input)", b(`"dexterity"`, `"luck"`), []string{p + "abilityScores/1 invalidValue"}},
		{"ability is case sensitive", "schema FR-025", "unhappy (invalid input)", b(`"dexterity"`, `"Dexterity"`), []string{p + "abilityScores/1 invalidValue"}},
		{"ability increases are not data", "FR-032", "unhappy (rule interaction)", b(`"feat"`, `"abilityIncreases":[2,1],"feat"`), []string{p + "abilityIncreases unknownField"}},
		{"feat with a qualifier", "FR-026", "happy", b(`{"featId":"feat_a"}`, `{"featId":"feat_a","choice":"Word"}`), nil},
		{"blank qualifier", "FR-026", "unhappy (invalid input)", b(`{"featId":"feat_a"}`, `{"featId":"feat_a","choice":" "}`), []string{p + "feat/choice invalidValue"}},
		{"feat without id", "FR-025", "unhappy (missing content)", b(`{"featId":"feat_a"}`, `{}`), []string{p + "feat/featId missingField"}},
		{"malformed feat id", "FR-025", "unhappy (invalid input)", b(`"feat_a"`, `"Feat A"`), []string{p + "feat/featId invalidValue"}},
		{"one skill", "FR-027", "unhappy (boundary)", b(`["Athletics","Stealth"]`, `["Athletics"]`), []string{p + "skillProficiencies outOfRange"}},
		{"three skills", "FR-027", "unhappy (boundary)", b(`["Athletics","Stealth"]`, `["Athletics","Stealth","Arcana"]`), []string{p + "skillProficiencies outOfRange"}},
		{"repeated skill", "FR-027", "unhappy (boundary)", b(`"Stealth"`, `"Athletics"`), []string{p + "skillProficiencies/1 duplicate"}},
		{"unknown skill", "FR-027", "unhappy (invalid input)", b(`"Stealth"`, `"Luck"`), []string{p + "skillProficiencies/1 invalidValue"}},
		{"choice tool proficiency", "FR-028", "happy", b(`{"kind":"specific","name":"Tool"}`, `{"kind":"choice","category":"Set"}`), nil},
		{"unknown tool kind", "FR-028", "unhappy (invalid input)", b(`"specific"`, `"either"`), []string{p + "toolProficiency/kind invalidValue"}},
		{"specific tool without name", "FR-028", "unhappy (missing content)", b(`,"name":"Tool"`, ``), []string{p + "toolProficiency/name missingField"}},
		{"choice tool with a name key", "FR-028", "unhappy (invalid input)", b(`{"kind":"specific","name":"Tool"}`, `{"kind":"choice","name":"Tool","category":"Set"}`), []string{p + "toolProficiency/name unknownField"}},
		{"option B missing", "FR-029", "unhappy (missing content)", b(`,"B":{"items":[],"gold":50}`, ``), []string{p + "equipmentOptions/B missingField"}},
		{"extra option key", "FR-029", "unhappy (invalid input)", b(`"B":`, `"C":{"items":[],"gold":1},"B":`), []string{p + "equipmentOptions/C unknownField"}},
		{"quantity zero", "FR-030", "unhappy (boundary)", b(`"quantity":2`, `"quantity":0`), []string{p + "equipmentOptions/A/items/0/quantity outOfRange"}},
		{"quantity one", "FR-030", "unhappy (boundary)", b(`"quantity":2`, `"quantity":1`), nil},
		{"parenthetical stays in the name", "FR-030", "happy", b(`"Rope"`, `"Book (history)"`), nil},
		{"negative gold", "FR-029", "unhappy (boundary)", b(`"gold":10`, `"gold":-1`), []string{p + "equipmentOptions/A/gold outOfRange"}},
		{"missing gold", "FR-029", "unhappy (missing content)", b(`,"gold":10`, ``), []string{p + "equipmentOptions/A/gold missingField"}},
		{"item without name", "FR-030", "unhappy (missing content)", b(`"name":"Rope",`, ``), []string{p + "equipmentOptions/A/items/0/name missingField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeFeats(t *testing.T) {
	t.Parallel()
	f := func(old, repl string) string { return wrapDoc("feats", mutate(featJSON, old, repl)) }
	const p = "/feats/0/"
	runDocCases(t, []docCase{
		{"valid", specSpec + " FR-033 to FR-041", "happy", wrapDoc("feats", featJSON), nil},
		{"unknown category", "FR-034", "unhappy (invalid input)", f(`"Origin"`, `"origin"`), []string{p + "category invalidValue"}},
		{"missing category", "FR-033", "unhappy (missing content)", f(`"category":"Origin",`, ``), []string{p + "category missingField"}},
		{"no benefits", "FR-033", "unhappy (empty case)", f(`[{"text":"x"}]`, `[]`), []string{p + "benefits outOfRange"}},
		{"missing benefits", "FR-033", "unhappy (missing content)", f(`"benefits":[{"text":"x"}],`, ``), []string{p + "benefits missingField"}},
		{"missing repeatable", "FR-033", "unhappy (missing content)", f(`,"repeatable":false`, ``), []string{p + "repeatable missingField"}},
		{"repeatable is not a boolean", "FR-033", "unhappy (invalid input)", f(`false`, `"no"`), []string{p + "repeatable wrongType"}},
		{"named benefit", "FR-040", "happy", f(`{"text":"x"}`, `{"name":"N","text":"x"}`), nil},
		{"blank benefit name", "FR-063", "unhappy (invalid input)", f(`{"text":"x"}`, `{"name":" ","text":"x"}`), []string{p + "benefits/0/name invalidValue"}},
		{"blank benefit text", "FR-063", "unhappy (invalid input)", f(`"x"`, `" "`), []string{p + "benefits/0/text invalidValue"}},
		{"benefit without text", "FR-063", "unhappy (missing content)", f(`{"text":"x"}`, `{"name":"N"}`), []string{p + "benefits/0/text missingField"}},
		{"no prerequisites field", "FR-035", "happy", f(`"benefits"`, `"benefits"`), nil},
		{"level prerequisite", "FR-035, schema FR-118", "happy", f(`"benefits"`, `"prerequisites":[{"kind":"level","minimum":4}],"benefits"`), nil},
		{"empty prerequisites", "FR-035 (SX-4)", "unhappy (invalid input)", f(`"benefits"`, `"prerequisites":[],"benefits"`), []string{p + "prerequisites outOfRange"}},
		{"level 0 and level 21", "schema FR-118", "unhappy (boundary)",
			f(`"benefits"`, `"prerequisites":[{"kind":"level","minimum":0},{"kind":"level","minimum":21}],"benefits"`),
			[]string{p + "prerequisites/0/minimum outOfRange", p + "prerequisites/1/minimum outOfRange"}},
		{"ability prerequisite", "schema FR-118", "happy", f(`"benefits"`, `"prerequisites":[{"kind":"abilityScore","abilities":["strength","dexterity"],"minimum":13}],"benefits"`), nil},
		{"ability prerequisite bounds", "schema FR-118", "unhappy (boundary)",
			f(`"benefits"`, `"prerequisites":[{"kind":"abilityScore","abilities":[],"minimum":0},{"kind":"abilityScore","abilities":["strength","strength"],"minimum":31}],"benefits"`),
			[]string{p + "prerequisites/0/abilities outOfRange", p + "prerequisites/0/minimum outOfRange", p + "prerequisites/1/abilities/1 duplicate", p + "prerequisites/1/minimum outOfRange"}},
		{"feature prerequisite", "schema FR-118", "happy", f(`"benefits"`, `"prerequisites":[{"kind":"feature","name":"F"}],"benefits"`), nil},
		{"unknown prerequisite kind", "schema FR-118", "unhappy (invalid input)",
			f(`"benefits"`, `"prerequisites":[{"kind":"mystery","name":"F"}],"benefits"`), []string{p + "prerequisites/0/kind invalidValue"}},
		{"prerequisite with a foreign key", "schema FR-118", "unhappy (invalid input)",
			f(`"benefits"`, `"prerequisites":[{"kind":"feature","name":"F","minimum":1}],"benefits"`), []string{p + "prerequisites/0/minimum unknownField"}},
		{"repeatable with its text", "FR-039 (SX-2)", "happy", f(`"repeatable":false`, `"repeatable":true,"repeatableText":"r"`), nil},
		{"repeatable text without repeatable", "FR-039 (SX-2)", "unhappy (rule interaction)", f(`"repeatable":false`, `"repeatable":false,"repeatableText":"r"`), []string{p + "repeatableText constraintViolation"}},
		{"repeatable without text", "FR-039 (SX-2)", "unhappy (missing content)", f(`"repeatable":false`, `"repeatable":true`), []string{p + "repeatableText missingField"}},
		{"blank repeatable text", "FR-039 (SX-2)", "unhappy (invalid input)", f(`"repeatable":false`, `"repeatable":true,"repeatableText":" "`), []string{p + "repeatableText invalidValue"}},
		{"repeatable of the wrong type leaves the text rule unchecked", "FR-039", "unhappy (rule interaction)",
			f(`"repeatable":false`, `"repeatable":1,"repeatableText":"r"`), []string{p + "repeatable wrongType"}},
		{"key outside the schema", "FR-058", "unhappy (invalid input)", f(`"benefits"`, `"description":"d","benefits"`), []string{p + "description unknownField"}},
		{"same id under two kinds", "FR-006", "unhappy (rule interaction)",
			`{"schemaVersion":1,"backgrounds":[` + mutate(backgroundJSON, `"id":"fixture"`, `"id":"feat_a"`) + `],"feats":[` + featJSON + `]}`, nil},
		{"id repeated within the kind", "FR-005", "unhappy (invalid input)",
			`{"schemaVersion":1,"feats":[` + featJSON + `,` + featJSON + `]}`, []string{"/feats/1/id duplicate"}},
		{"arrays in revision-table order", "schema FR-016", "happy (ordering)",
			`{"schemaVersion":1,"feats":[` + mutate(featJSON, `"category":"Origin"`, `"category":"x"`) + `],"species":[` + mutate(speciesJSON(""), `"Humanoid"`, `"x"`) + `]}`,
			[]string{"/species/0/creatureType invalidValue", "/feats/0/category invalidValue"}},
	})
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-025 to FR-030 - the rules of a background row
// that sit at a bound or in a rule choice: the id registers for the duplicate check, the qualifier is a line, a
// specific tool name is a name, a category is a text, an unknown tool kind accepts both variants' keys, the feat
// object rejects other keys and the gold and quantity bounds are inclusive.
// Path: per case.
func TestDecodeBackgroundRuleChoices(t *testing.T) {
	t.Parallel()
	b := func(old, repl string) string { return wrapDoc("backgrounds", mutate(backgroundJSON, old, repl)) }
	const p = "/backgrounds/0/"
	runDocCases(t, []docCase{
		{"duplicate background id", "FR-005", "unhappy (invalid input)", `{"schemaVersion":1,"backgrounds":[` + backgroundJSON + `,` + backgroundJSON + `]}`, []string{"/backgrounds/1/id duplicate"}},
		{"qualifier with a line feed", "FR-026", "unhappy (invalid input)", b(`{"featId":"feat_a"}`, `{"featId":"feat_a","choice":"A\nB"}`), []string{p + "feat/choice invalidValue"}},
		{"feat with an extra key", "FR-025", "unhappy (invalid input)", b(`{"featId":"feat_a"}`, `{"featId":"feat_a","extra":1}`), []string{p + "feat/extra unknownField"}},
		{"specific tool name with a line feed", "FR-028", "unhappy (invalid input)", b(`"name":"Tool"`, `"name":"To\nol"`), []string{p + "toolProficiency/name invalidValue"}},
		{"category with a line feed", "FR-028", "happy", b(`{"kind":"specific","name":"Tool"}`, `{"kind":"choice","category":"Set\nTwo"}`), nil},
		{"unknown tool kind with both variants' keys", "FR-028", "unhappy (invalid input)", b(`{"kind":"specific","name":"Tool"}`, `{"kind":"either","name":"Tool","category":"Set"}`), []string{p + "toolProficiency/kind invalidValue"}},
		{"gold zero", "FR-029", "unhappy (boundary)", b(`"gold":10`, `"gold":0`), nil},
		{"gold at the cap", "FR-029", "unhappy (boundary)", b(`"gold":10`, `"gold":2147483647`), nil},
		{"gold beyond the cap", "FR-029", "unhappy (boundary)", b(`"gold":10`, `"gold":2147483648`), []string{p + "equipmentOptions/A/gold outOfRange"}},
		{"quantity at the cap", "FR-030", "unhappy (boundary)", b(`"quantity":2`, `"quantity":2147483647`), nil},
		{"quantity beyond the cap", "FR-030", "unhappy (boundary)", b(`"quantity":2`, `"quantity":2147483648`), []string{p + "equipmentOptions/A/items/0/quantity outOfRange"}},
	})
}
