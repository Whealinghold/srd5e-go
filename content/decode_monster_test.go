package content_test

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// The fixtures are synthetic: made-up names and numbers, never stat blocks.

const monAbilitiesJSON = `"abilities":{"str":{"score":10,"save":0},"dex":{"score":10,"save":0},"con":{"score":10,"save":0},"int":{"score":10,"save":0},"wis":{"score":10,"save":0},"cha":{"score":10,"save":0}}`

const monJSON = `{"id":"fixture_beast","name":"Fixture Beast","source":{"document":"Fixture","section":"Monsters A-Z","page":258},` +
	`"sizes":["Medium"],"type":"Beast","alignment":"Unaligned","armorClass":12,"initiative":{"modifier":1,"score":11},` +
	`"hitPoints":{"average":9,"count":2,"faces":8,"modifier":0},"speeds":[{"modes":["walk"],"feet":30}],` + monAbilitiesJSON + `,` +
	`"senses":{"passivePerception":10},"languages":{},"challengeRating":"0","experiencePoints":0}`

func monDoc(entries ...string) string {
	return `{"schemaVersion":1,"monsters":[` + strings.Join(entries, ",") + `]}`
}

func monWith(old, repl string) string { return monDoc(mutate(monJSON, old, repl)) }

// monPartJSON is the smallest part, with extra members appended.
func monPartJSON(name, extra string) string {
	return `{"name":"` + name + `","usage":{"kind":"unlimited"},"text":"T."` + extra + `}`
}

// monActions is the fixture with the given parts as its Actions.
func monActions(parts ...string) string {
	return monWith(`"experiencePoints":0`, `"experiencePoints":0,"actions":[`+strings.Join(parts, ",")+`]`)
}

func monDecode(t *testing.T, doc string) content.Monster {
	t.Helper()
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %q", decodeProblems(doc))
	}
	for _, e := range set.Entries() {
		if m, ok := e.(content.Monster); ok {
			return m
		}
	}
	t.Fatal("no monster in the set")
	return content.Monster{}
}

// Requirement: monSpec FR-048, FR-051, FR-052, FR-053, FR-054 - the monsters root array is registered (absent means
// none, a non-array is wrongType), the smallest monster decodes with every optional collection empty, decoding is
// deterministic and does not modify its input, and an unknown or repeated key is reported.
// Path: unhappy (empty).
func TestDecodeMonsterRoot(t *testing.T) {
	t.Parallel()
	const p = "/monsters/0"
	runDocCases(t, []docCase{
		{"absent array", monSpec + " FR-048", "happy", `{"schemaVersion":1}`, nil},
		{"empty array", "FR-048", "unhappy (boundary)", `{"schemaVersion":1,"monsters":[]}`, nil},
		{"object", "FR-048", "unhappy (boundary)", `{"schemaVersion":1,"monsters":{}}`, []string{"/monsters wrongType"}},
		{"null", "FR-048", "unhappy (boundary)", `{"schemaVersion":1,"monsters":null}`, []string{"/monsters wrongType"}},
		{"element not an object", "FR-051", "unhappy (invalid input)", `{"schemaVersion":1,"monsters":[1]}`, []string{p + " wrongType"}},
		{"unknown key", "FR-053", "unhappy (invalid input)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"extra":1`), []string{p + "/extra unknownField"}},
		{"repeated key", "FR-053", "unhappy (invalid input)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"armorClass":1`), []string{p + "/armorClass duplicate"}},
		{"non-current version", "FR-053", "unhappy (invalid input)", `{"schemaVersion":2,"monsters":[]}`, []string{"/schemaVersion unsupportedVersion"}},
		{"later duplicate id", "FR-058", "unhappy (duplicate)", monDoc(monJSON, monJSON), []string{"/monsters/1/id duplicate"}},
		{"three independent problems in field order", "FR-049", "unhappy (invalid input)",
			monDoc(
				strings.NewReplacer(`"type":"Beast"`, `"type":"x"`, `"armorClass":12`, `"armorClass":-1`, `"experiencePoints":0`, `"experiencePoints":-1`).Replace(monJSON)),
			[]string{p + "/type invalidValue", p + "/armorClass outOfRange", p + "/experiencePoints outOfRange"}},
	})
	m := monDecode(t, monDoc(monJSON))
	want := monMake(t, monFields(t))
	if !m.Equal(want) {
		t.Errorf("the smallest monster = %+v, want the one built from fields [FR-052]", m)
	}
	if len(m.Traits())+len(m.Actions())+len(m.BonusActions())+len(m.Reactions())+len(m.Tags())+len(m.Gear())+len(m.Skills())+
		len(m.Vulnerabilities())+len(m.Resistances())+len(m.Immunities())+len(m.Senses()) != 0 || len(m.Languages().Phrases()) != 0 {
		t.Error("an optional collection of the smallest monster is not empty [FR-052]")
	}
	if _, ok := m.LegendaryActions(); ok {
		t.Error("the smallest monster has legendary actions [FR-052]")
	}
	input := []byte(monDoc(monJSON))
	before := bytes.Clone(input)
	a, errA := content.JSONDecoder{}.Decode(input)
	b, errB := content.JSONDecoder{}.Decode(input)
	if errA != nil || errB != nil || !a.Equal(b) || !bytes.Equal(input, before) {
		t.Error("decoding twice differs or modified the input [FR-054]")
	}
	for i, probs := 0, 0; i < 2; i++ {
		_, err := content.JSONDecoder{}.Decode([]byte(monWith(`"experiencePoints":0`, `"experiencePoints":-1`)))
		if probs = len(content.Problems(err)); probs != 1 {
			t.Errorf("problems = %d, want 1 [FR-054]", probs)
		}
	}
}

// monRow is one row of the agreement test: the fixture with one replacement, the problems the decoder reports, and
// the same fault in the constructor's fields when the rule is a constructor rule of the monster itself.
type monRow struct {
	name, requirement, path string
	old, repl               string
	want                    []string
	ctor                    func(*content.MonsterFields)
}

// Requirement: monSpec FR-049, FR-050, FR-051, FR-095 and table 4.8a - every invalid value goes to the decoder, as
// JSON, with the problem kind and path of the table, and to the constructor, as a returned validation error; every
// valid value to both.
// Path: per case.
func TestMonsterAgreementHeaderAndHighlights(t *testing.T) {
	t.Parallel()
	const p = "/monsters/0/"
	two := func(f *content.MonsterFields) {
		f.Sizes = []content.CreatureSize{content.SizeMedium, content.SizeMedium}
	}
	rows := []monRow{
		{"valid", "FR-051", "happy", `"armorClass":12`, `"armorClass":12`, nil, nil},
		{"blank name", "FR-003", "unhappy (invalid input)", `"Fixture Beast"`, `" "`, []string{p + "name invalidValue"}, nil},
		{"id mismatch", "FR-002", "unhappy (invalid input)", `"fixture_beast"`, `"fixture_bat"`, []string{p + "id constraintViolation"}, nil},
		{"id invalid", "FR-002", "unhappy (invalid input)", `"fixture_beast"`, `"Fixture"`, []string{p + "id invalidValue"}, nil},
		{"id missing", "FR-051", "unhappy (boundary)", `"id":"fixture_beast",`, ``, []string{p + "id missingField"}, nil},
		{"page 257", "FR-004", "unhappy (boundary)", `"page":258`, `"page":257`, []string{p + "source/page outOfRange"}, nil},
		{"page 364", "FR-004", "unhappy (boundary)", `"page":258`, `"page":364`, nil, nil},
		{"page 365", "FR-004", "unhappy (boundary)", `"page":258`, `"page":365`, []string{p + "source/page outOfRange"}, nil},
		{"animals section", "FR-004", "happy", `"Monsters A-Z"`, `"Animals"`, nil, nil},
		{"other section", "FR-004", "unhappy (boundary)", `"Monsters A-Z"`, `"Equipment"`, []string{p + "source/section invalidValue"}, nil},
		{"source missing", "FR-051", "unhappy (boundary)", `"source":{"document":"Fixture","section":"Monsters A-Z","page":258},`, ``, []string{p + "source missingField"}, nil},
		{"empty sizes", "FR-008", "unhappy (empty)", `["Medium"]`, `[]`, []string{p + "sizes constraintViolation"}, func(f *content.MonsterFields) { f.Sizes = nil }},
		{"two sizes", "FR-008", "happy", `["Medium"]`, `["Medium","Small"]`, nil, nil},
		{"duplicate sizes", "FR-008", "unhappy (invalid input)", `["Medium"]`, `["Medium","Medium"]`, []string{p + "sizes/1 duplicate"}, two},
		{"unknown size", "FR-050", "unhappy (invalid input)", `["Medium"]`, `["Enormous"]`, []string{p + "sizes/0 invalidValue"}, func(f *content.MonsterFields) { f.Sizes = []content.CreatureSize{0} }},
		{"sizes not an array", "FR-051", "unhappy (invalid input)", `["Medium"]`, `"Medium"`, []string{p + "sizes wrongType"}, nil},
		{"swarm", "FR-009", "happy", `"type":"Beast"`, `"swarmOf":"Tiny","type":"Beast"`, nil, nil},
		{"unknown swarm size", "FR-050", "unhappy (invalid input)", `"type":"Beast"`, `"swarmOf":"x","type":"Beast"`, []string{p + "swarmOf invalidValue"}, nil},
		{"unknown type", "FR-050", "unhappy (invalid input)", `"Beast"`, `"not_a_creature_type"`, []string{p + "type invalidValue"}, func(f *content.MonsterFields) { f.Type = 0 }},
		{"unknown alignment", "FR-050", "unhappy (invalid input)", `"Unaligned"`, `"not_an_alignment"`, []string{p + "alignment invalidValue"}, func(f *content.MonsterFields) { f.Alignment = 0 }},
		{"tags", "FR-011", "happy", `"type":"Beast"`, `"type":"Beast","tags":["Dinosaur"]`, nil, nil},
		{"blank tag", "FR-011", "unhappy (invalid input)", `"type":"Beast"`, `"type":"Beast","tags":[" "]`, []string{p + "tags/0 invalidValue"}, func(f *content.MonsterFields) { f.Tags = []string{" "} }},
		{"AC -1", "FR-013", "unhappy (boundary)", `"armorClass":12`, `"armorClass":-1`, []string{p + "armorClass outOfRange"}, func(f *content.MonsterFields) { f.ArmorClass = -1 }},
		{"AC 0", "FR-013", "unhappy (boundary)", `"armorClass":12`, `"armorClass":0`, nil, nil},
		{"AC a string", "FR-013", "unhappy (invalid input)", `"armorClass":12`, `"armorClass":"12"`, []string{p + "armorClass wrongType"}, nil},
		{"initiative missing score", "FR-014", "unhappy (boundary)", `,"score":11`, ``, []string{p + "initiative/score missingField"}, nil},
		{"initiative missing", "FR-051", "unhappy (boundary)", `"initiative":{"modifier":1,"score":11},`, ``, []string{p + "initiative missingField"}, nil},
		{"average 0", "FR-015", "unhappy (boundary)", `"average":9`, `"average":0`, []string{p + "hitPoints/average outOfRange"}, nil},
		{"average 1", "FR-015", "unhappy (boundary)", `"average":9`, `"average":1`, nil, nil},
		{"count 0", "FR-015", "unhappy (boundary)", `"count":2`, `"count":0`, []string{p + "hitPoints/count outOfRange"}, nil},
		{"die d7", "FR-015", "unhappy (boundary)", `"faces":8`, `"faces":7`, []string{p + "hitPoints/faces outOfRange"}, nil},
		{"die d20", "FR-015", "unhappy (boundary)", `"faces":8`, `"faces":20`, nil, nil},
		{"negative modifier", "FR-015", "unhappy (boundary)", `"modifier":0}`, `"modifier":-4}`, nil, nil},
		{"hit points missing modifier", "FR-015", "unhappy (boundary)", `,"modifier":0}`, `}`, []string{p + "hitPoints/modifier missingField"}, nil},
		{"negative speed", "FR-016", "unhappy (boundary)", `"feet":30`, `"feet":-5`, []string{p + "speeds/0/feet outOfRange"}, nil},
		{"zero speed", "FR-016", "unhappy (boundary)", `"feet":30`, `"feet":0`, nil, nil},
		{"hover with fly", "FR-016", "happy", `"modes":["walk"],"feet":30`, `"modes":["fly"],"feet":30,"hover":true`, nil, nil},
		{"hover without fly", "FR-016", "unhappy (invalid input)", `"feet":30`, `"feet":30,"hover":true`, []string{p + "speeds/0/hover constraintViolation"}, nil},
		{"repeated mode", "FR-016", "unhappy (invalid input)", `["walk"]`, `["walk","walk"]`, []string{p + "speeds/0/modes/1 duplicate"}, nil},
		{"unknown mode", "FR-016", "unhappy (invalid input)", `["walk"]`, `["crawl"]`, []string{p + "speeds/0/modes/0 invalidValue"}, nil},
		{"no modes", "FR-016", "unhappy (empty)", `["walk"]`, `[]`, []string{p + "speeds/0/modes outOfRange"}, nil},
		{"duplicate entry", "FR-016", "unhappy (invalid input)", `[{"modes":["walk"],"feet":30}]`, `[{"modes":["walk"],"feet":30},{"modes":["walk"],"feet":40}]`, []string{p + "speeds/1 constraintViolation"}, func(f *content.MonsterFields) {
			f.Speeds = []content.SpeedEntry{monSpeed(t, 30), monSpeed(t, 40)}
		}},
		{"qualifier tells entries apart", "FR-017", "happy", `[{"modes":["walk"],"feet":30}]`, `[{"modes":["walk"],"feet":30},{"modes":["walk"],"feet":40,"qualifier":"wolf form only"}]`, nil, nil},
		{"no entries", "FR-016", "unhappy (empty)", `[{"modes":["walk"],"feet":30}]`, `[]`, []string{p + "speeds constraintViolation"}, func(f *content.MonsterFields) { f.Speeds = nil }},
	}
	runMonRows(t, rows)
}

func runMonRows(t *testing.T, rows []monRow) {
	t.Helper()
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			if got := decodeProblems(monWith(r.old, r.repl)); !slices.Equal(got, r.want) {
				t.Errorf("decoder problems = %q, want %q [%s, %s]", got, r.want, r.requirement, r.path)
			}
			if r.ctor == nil {
				return
			}
			f := monFields(t)
			r.ctor(&f)
			if _, err := monBuild(t, f); (err == nil) != (len(r.want) == 0) || (err != nil && !isInvalidInput(err)) {
				t.Errorf("constructor error = %v, want a verdict that agrees with %q [%s, %s]", err, r.want, r.requirement, r.path)
			}
		})
	}
}

// Requirement: monSpec FR-018 to FR-032 and table 4.8a - abilities, skills, defences, gear, senses, languages,
// Challenge Rating and experience points, with the problem kind and path of the table, and the constructor's verdict.
// Path: per case.
func TestMonsterAgreementAbilitiesAndDetails(t *testing.T) {
	t.Parallel()
	const p = "/monsters/0/"
	cond := func(c content.ConditionKind) content.DefenseEntry { return must(content.NewConditionDefense(c, "")) }
	dmg := func(d content.DamageType) content.DefenseEntry { return must(content.NewDamageTypeDefense(d, "")) }
	rows := []monRow{
		{"ability missing", "FR-018", "unhappy (boundary)", `"con":{"score":10,"save":0},`, ``, []string{p + "abilities/con missingField"}, nil},
		{"save missing", "FR-018", "unhappy (boundary)", `"str":{"score":10,"save":0}`, `"str":{"score":10}`, []string{p + "abilities/str/save missingField"}, nil},
		{"score missing", "FR-018", "unhappy (boundary)", `"str":{"score":10,"save":0}`, `"str":{"save":0}`, []string{p + "abilities/str/score missingField"}, nil},
		{"score 0", "FR-018", "unhappy (boundary)", `"str":{"score":10,`, `"str":{"score":0,`, []string{p + "abilities/str/score outOfRange"}, nil},
		{"score 1", "FR-018", "unhappy (boundary)", `"str":{"score":10,`, `"str":{"score":1,`, nil, nil},
		{"score 30", "FR-018", "unhappy (boundary)", `"str":{"score":10,`, `"str":{"score":30,`, nil, nil},
		{"score 31", "FR-018", "unhappy (boundary)", `"str":{"score":10,`, `"str":{"score":31,`, []string{p + "abilities/str/score outOfRange"}, nil},
		{"unknown ability key", "FR-018", "unhappy (invalid input)", `"cha":{"score":10,"save":0}}`, `"cha":{"score":10,"save":0},"luck":{"score":1,"save":0}}`, []string{p + "abilities/luck unknownField"}, nil},
		{"skill", "FR-021", "happy", `"senses"`, `"skills":{"Stealth":4,"Sleight of Hand":-1},"senses"`, nil, nil},
		{"unknown skill", "FR-021", "unhappy (invalid input)", `"senses"`, `"skills":{"Cooking":4},"senses"`, []string{p + "skills/Cooking invalidValue"}, func(f *content.MonsterFields) { f.Skills = map[content.Skill]int{0: 4} }},
		{"skill bonus not a number", "FR-021", "unhappy (invalid input)", `"senses"`, `"skills":{"Stealth":"4"},"senses"`, []string{p + "skills/Stealth wrongType"}, nil},
		{"repeated skill", "FR-021", "unhappy (invalid input)", `"senses"`, `"skills":{"Stealth":4,"Stealth":5},"senses"`, []string{p + "skills/Stealth duplicate"}, nil},
		{"resistance to a damage type", "FR-024", "happy", `"senses"`, `"resistances":[{"damageType":"Cold","qualifier":"from nonmagical attacks"},{"text":"Chosen type"}],"senses"`, nil, nil},
		{"condition among resistances", "FR-024", "unhappy (invalid input)", `"senses"`, `"resistances":[{"condition":"Poisoned"}],"senses"`, []string{p + "resistances/0 invalidValue"}, func(f *content.MonsterFields) {
			f.Resistances = []content.DefenseEntry{cond(content.ConditionPoisoned)}
		}},
		{"condition among vulnerabilities", "FR-024", "unhappy (invalid input)", `"senses"`, `"vulnerabilities":[{"condition":"Poisoned"}],"senses"`, []string{p + "vulnerabilities/0 invalidValue"}, func(f *content.MonsterFields) {
			f.Vulnerabilities = []content.DefenseEntry{cond(content.ConditionPoisoned)}
		}},
		{"damage then conditions", "FR-025", "happy", `"senses"`, `"immunities":[{"damageType":"Poison"},{"condition":"Poisoned","qualifier":"q"}],"senses"`, nil, nil},
		{"condition before damage", "FR-025", "unhappy (rule interaction)", `"senses"`, `"immunities":[{"condition":"Poisoned"},{"damageType":"Poison"}],"senses"`, []string{p + "immunities/1 constraintViolation"}, func(f *content.MonsterFields) {
			f.Immunities = []content.DefenseEntry{cond(content.ConditionPoisoned), dmg(content.DamagePoison)}
		}},
		{"unknown damage type", "FR-050", "unhappy (invalid input)", `"senses"`, `"immunities":[{"damageType":"Arcane"}],"senses"`, []string{p + "immunities/0/damageType invalidValue"}, nil},
		{"defence with no key", "FR-024", "unhappy (invalid input)", `"senses"`, `"resistances":[{}],"senses"`, []string{p + "resistances/0/damageType missingField"}, nil},
		{"defence with two keys", "FR-024", "unhappy (invalid input)", `"senses"`, `"resistances":[{"damageType":"Cold","text":"x"}],"senses"`, []string{p + "resistances/0 constraintViolation"}, nil},
		{"blank qualifier", "FR-024", "unhappy (invalid input)", `"senses"`, `"resistances":[{"damageType":"Cold","qualifier":" "}],"senses"`, []string{p + "resistances/0/qualifier invalidValue"}, nil},
		{"gear", "FR-026", "happy", `"senses"`, `"gear":["Item","Item"],"senses"`, nil, nil},
		{"blank gear", "FR-026", "unhappy (invalid input)", `"senses"`, `"gear":[""],"senses"`, []string{p + "gear/0 invalidValue"}, func(f *content.MonsterFields) { f.Gear = []string{""} }},
		{"sense", "FR-027", "happy", `"senses":{`, `"senses":{"special":[{"kind":"Darkvision","feet":60,"qualifier":"q"}],`, nil, nil},
		{"negative sense range", "FR-028", "unhappy (invalid input)", `"senses":{`, `"senses":{"special":[{"kind":"Darkvision","feet":-1}],`, []string{p + "senses/special/0/feet outOfRange"}, nil},
		{"unknown sense", "FR-028", "unhappy (invalid input)", `"senses":{`, `"senses":{"special":[{"kind":"Echolocation","feet":1}],`, []string{p + "senses/special/0/kind invalidValue"}, nil},
		{"passive perception missing", "FR-051", "unhappy (boundary)", `"passivePerception":10`, ``, []string{p + "senses/passivePerception missingField"}, nil},
		{"senses missing", "FR-051", "unhappy (boundary)", `"senses":{"passivePerception":10},`, ``, []string{p + "senses missingField"}, nil},
		{"languages with telepathy", "FR-029", "happy", `"languages":{}`, `"languages":{"phrases":["Deep Speech"],"telepathyFeet":120,"telepathyQualifier":"q"}`, nil, nil},
		{"qualifier without telepathy", "FR-029", "unhappy (invalid input)", `"languages":{}`, `"languages":{"telepathyQualifier":"q"}`, []string{p + "languages/telepathyQualifier constraintViolation"}, nil},
		{"blank phrase", "FR-029", "unhappy (invalid input)", `"languages":{}`, `"languages":{"phrases":["x",""]}`, []string{p + "languages/phrases/1 invalidValue"}, nil},
		{"languages missing", "FR-051", "unhappy (boundary)", `"languages":{},`, ``, []string{p + "languages missingField"}, nil},
		{"CR 1/3", "FR-030", "unhappy (boundary)", `"challengeRating":"0"`, `"challengeRating":"1/3"`, []string{p + "challengeRating invalidValue"}, nil},
		{"CR 31", "FR-030", "unhappy (boundary)", `"challengeRating":"0"`, `"challengeRating":"31"`, []string{p + "challengeRating invalidValue"}, nil},
		{"CR 30", "FR-030", "unhappy (boundary)", `"challengeRating":"0"`, `"challengeRating":"30"`, nil, nil},
		{"CR 1/8", "FR-030", "happy", `"challengeRating":"0"`, `"challengeRating":"1/8"`, nil, nil},
		{"CR a number", "FR-030", "unhappy (invalid input)", `"challengeRating":"0"`, `"challengeRating":1`, []string{p + "challengeRating wrongType"}, nil},
		{"negative XP", "FR-032", "unhappy (boundary)", `"experiencePoints":0`, `"experiencePoints":-1`, []string{p + "experiencePoints outOfRange"}, func(f *content.MonsterFields) { f.ExperiencePoints = -1 }},
		{"XP not checked against the CR", "FR-032", "unhappy (boundary)", `"challengeRating":"0","experiencePoints":0`, `"challengeRating":"12","experiencePoints":8000`, nil, nil},
		{"lair XP above XP", "FR-032", "unhappy (boundary)", `"experiencePoints":0`, `"experiencePoints":10,"experiencePointsInLair":11`, nil, nil},
		{"lair XP equal to XP", "FR-032", "unhappy (boundary)", `"experiencePoints":0`, `"experiencePoints":10,"experiencePointsInLair":10`, []string{p + "experiencePointsInLair constraintViolation"}, func(f *content.MonsterFields) { f.ExperiencePoints, f.ExperiencePointsInLair = 10, 10 }},
		{"lair XP 0", "FR-032", "unhappy (boundary)", `"experiencePoints":0`, `"experiencePoints":0,"experiencePointsInLair":0`, []string{p + "experiencePointsInLair constraintViolation"}, nil},
		{"lair XP negative", "FR-032", "unhappy (boundary)", `"experiencePoints":0`, `"experiencePoints":0,"experiencePointsInLair":-1`, []string{p + "experiencePointsInLair outOfRange"}, nil},
		{"XP missing", "FR-051", "unhappy (boundary)", `,"experiencePoints":0`, ``, []string{p + "experiencePoints missingField"}, nil},
	}
	runMonRows(t, rows)
}

// Requirement: monSpec FR-033 to FR-047, FR-100, FR-101 and tables 4.6a, 4.6b, 4.7a, 4.8a - the parts of a stat block:
// usage, attack, saving throw, clauses, riders, damage, legendary actions and spellcasting, with the problem kind and
// path of the table.
// Path: per case.
func TestDecodeMonsterParts(t *testing.T) {
	t.Parallel()
	const a = "/monsters/0/actions/0/"
	part := func(extra string) string { return monActions(monPartJSON("Bite", extra)) }
	usage := func(u string) string { return monActions(`{"name":"Bite","usage":` + u + `,"text":"T."}`) }
	clause := func(c string) string { return part(`,"clauses":[` + c + `]`) }
	damage := func(d string) string {
		return clause(`{"label":"hit","text":"T.","damage":[` + d + `]}`)
	}
	spell := func(sc string) string { return part(`,"spellcasting":` + sc) }
	const gp = `"groups":[{"frequency":{"kind":"at_will"},"spells":[{"name":"Light"}]}]`
	runDocCases(t, []docCase{
		{"part", monSpec + " FR-033", "happy", part(``), nil},
		{"blank part name", "FR-034", "unhappy (invalid input)", monActions(monPartJSON(" ", "")), []string{a + "name invalidValue"}},
		{"blank part text", "FR-034", "unhappy (invalid input)", monActions(strings.Replace(monPartJSON("Bite", ""), `"T."`, `" "`, 1)), []string{a + "text invalidValue"}},
		{"part missing usage", "FR-034", "unhappy (boundary)", monActions(`{"name":"Bite","text":"T."}`), []string{a + "usage missingField"}},
		{"part missing text", "FR-034", "unhappy (boundary)", monActions(`{"name":"Bite","usage":{"kind":"unlimited"}}`), []string{a + "text missingField"}},
		{"usage kind missing", "FR-035", "unhappy (boundary)", monActions(`{"name":"Bite","usage":{},"text":"T."}`), []string{a + "usage/kind missingField"}},
		{"usage kind not a string", "FR-035", "unhappy (invalid input)", monActions(`{"name":"Bite","usage":{"kind":1},"text":"T."}`), []string{a + "usage/kind wrongType"}},
		{"usage not an object", "FR-035", "unhappy (invalid input)", monActions(`{"name":"Bite","usage":"x","text":"T."}`), []string{a + "usage wrongType"}},
		{"part unknown key", "FR-053", "unhappy (invalid input)", part(`,"trigger":"x"`), []string{a + "trigger unknownField"}},
		{"part damage key (moved to clauses)", "FR-053", "unhappy (invalid input)", part(`,"damage":[]`), []string{a + "damage unknownField"}},
		{"per day", "FR-035", "happy", usage(`{"kind":"per_day","count":3,"inLairCount":4}`), nil},
		{"zero uses", "FR-035", "unhappy (boundary)", usage(`{"kind":"per_day","count":0}`), []string{a + "usage/count outOfRange"}},
		{"zero lair uses", "FR-035", "unhappy (boundary)", usage(`{"kind":"per_day","count":1,"inLairCount":0}`), []string{a + "usage/inLairCount outOfRange"}},
		{"recharge 4-6", "FR-035", "happy", usage(`{"kind":"recharge","low":4,"high":6}`), nil},
		{"recharge 6", "FR-035", "unhappy (boundary)", usage(`{"kind":"recharge","low":6,"high":6}`), nil},
		{"recharge low above high", "FR-035", "unhappy (boundary)", usage(`{"kind":"recharge","low":5,"high":4}`), []string{a + "usage constraintViolation"}},
		{"recharge high 7", "FR-035", "unhappy (boundary)", usage(`{"kind":"recharge","low":5,"high":7}`), []string{a + "usage/high outOfRange"}},
		{"recharge low 0", "FR-035", "unhappy (boundary)", usage(`{"kind":"recharge","low":0,"high":6}`), []string{a + "usage/low outOfRange"}},
		{"recharge after rest", "FR-035", "happy", usage(`{"kind":"recharge_after_rest"}`), nil},
		{"other usage", "FR-035", "happy", usage(`{"kind":"other","text":"1/Day; Requires Soul Bag"}`), nil},
		{"other usage blank", "FR-035", "unhappy (invalid input)", usage(`{"kind":"other","text":" "}`), []string{a + "usage/text invalidValue"}},
		{"unknown usage kind", "FR-035", "unhappy (invalid input)", usage(`{"kind":"daily"}`), []string{a + "usage/kind invalidValue"}},
		{"usage key of another kind", "FR-053", "unhappy (invalid input)", usage(`{"kind":"unlimited","count":1}`), []string{a + "usage/count unknownField"}},
		{"melee attack", "FR-037", "happy", part(`,"attack":{"kind":"melee","bonus":9,"reach":15}`), nil},
		{"ranged with range", "FR-037", "happy", part(`,"attack":{"kind":"ranged","bonus":7,"range":{"normal":80,"long":320}}`), nil},
		{"single range", "FR-037", "unhappy (boundary)", part(`,"attack":{"kind":"ranged","bonus":4,"range":{"normal":60}}`), nil},
		{"melee or ranged with qualifier", "FR-037", "happy", part(`,"attack":{"kind":"meleeOrRanged","bonus":-1,"qualifier":"q","reach":10,"range":{"normal":30,"long":120}}`), nil},
		{"unknown attack kind", "FR-037", "unhappy (invalid input)", part(`,"attack":{"kind":"thrown","bonus":1}`), []string{a + "attack/kind invalidValue"}},
		{"missing attack bonus", "FR-037", "unhappy (invalid input)", part(`,"attack":{"kind":"melee"}`), []string{a + "attack/bonus missingField"}},
		{"negative reach", "FR-037", "unhappy (boundary)", part(`,"attack":{"kind":"melee","bonus":1,"reach":-1}`), []string{a + "attack/reach outOfRange"}},
		{"reach 0", "FR-037", "unhappy (boundary)", part(`,"attack":{"kind":"melee","bonus":1,"reach":0}`), nil},
		{"range 0", "FR-037", "unhappy (boundary)", part(`,"attack":{"kind":"ranged","bonus":1,"range":{"normal":0}}`), []string{a + "attack/range/normal outOfRange"}},
		{"long range equal", "FR-037", "unhappy (boundary)", part(`,"attack":{"kind":"ranged","bonus":1,"range":{"normal":60,"long":60}}`), []string{a + "attack/range/long constraintViolation"}},
		{"long range 0", "FR-037", "unhappy (boundary)", part(`,"attack":{"kind":"ranged","bonus":1,"range":{"normal":60,"long":0}}`), []string{a + "attack/range/long outOfRange"}},
		{"saving throw", "FR-039", "happy", part(`,"savingThrow":{"ability":"wisdom","dc":15,"target":"the triggering creature","prefix":"Response—"}`), nil},
		{"saving throw without target", "FR-039", "unhappy (minimal)", part(`,"savingThrow":{"ability":"constitution","dc":12}`), nil},
		{"save DC 0", "FR-039", "unhappy (boundary)", part(`,"savingThrow":{"ability":"wisdom","dc":0}`), []string{a + "savingThrow/dc outOfRange"}},
		{"save DC 1", "FR-039", "unhappy (boundary)", part(`,"savingThrow":{"ability":"wisdom","dc":1}`), nil},
		{"unknown save ability", "FR-039", "unhappy (invalid input)", part(`,"savingThrow":{"ability":"luck","dc":10}`), []string{a + "savingThrow/ability invalidValue"}},
		{"attack and save in one part", "FR-039", "happy", part(`,"attack":{"kind":"melee","bonus":4,"reach":5},"savingThrow":{"ability":"constitution","dc":12},"clauses":[{"label":"hit","text":"x"},{"label":"firstFailure","text":"y"},{"label":"subsequentFailures","text":"z"}]`), nil},
		{"failure labels", "FR-038", "happy", clause(`{"label":"firstFailure","text":"a"},{"label":"secondFailure","text":"b"},{"label":"failure","text":"c"},{"label":"failureByFiveOrMore","text":"d"}`), nil},
		{"repeated label", "FR-038", "unhappy (invalid input)", clause(`{"label":"hit","text":"a"},{"label":"hit","text":"b"}`), []string{a + "clauses/1/label invalidValue"}},
		{"unknown label", "FR-038", "unhappy (invalid input)", clause(`{"label":"Hit","text":"a"}`), []string{a + "clauses/0/label invalidValue"}},
		{"clause missing text", "FR-038", "unhappy (boundary)", clause(`{"label":"hit"}`), []string{a + "clauses/0/text missingField"}},
		{"half damage only", "FR-100", "happy", clause(`{"label":"success","text":"Half damage only.","successRider":{"kind":"half_damage_only"}}`), nil},
		{"half damage", "FR-100", "happy", clause(`{"label":"success","text":"Half damage.","successRider":{"kind":"half_damage"}}`), nil},
		{"immunity", "FR-100", "happy", clause(`{"label":"success","text":"Immune.","successRider":{"kind":"immune_to_part","hours":24}}`), nil},
		{"hours 0", "FR-100", "unhappy (boundary)", clause(`{"label":"success","text":"x","successRider":{"kind":"immune_to_part","hours":0}}`), []string{a + "clauses/0/successRider/hours outOfRange"}},
		{"hours missing", "FR-100", "unhappy (boundary)", clause(`{"label":"success","text":"x","successRider":{"kind":"immune_to_part"}}`), []string{a + "clauses/0/successRider/hours missingField"}},
		{"unknown rider", "FR-100", "unhappy (invalid input)", clause(`{"label":"success","text":"x","successRider":{"kind":"double"}}`), []string{a + "clauses/0/successRider/kind invalidValue"}},
		{"rider on a failure", "FR-100", "unhappy (rule interaction)", clause(`{"label":"failure","text":"x","successRider":{"kind":"half_damage"}}`), []string{a + "clauses/0/successRider constraintViolation"}},
		{"damage number and dice", "FR-042", "happy", damage(`{"number":5,"dice":{"count":1,"faces":6,"modifier":2},"damageType":"Bludgeoning"}`), nil},
		{"damage fixed only", "FR-042", "unhappy (minimal)", damage(`{"number":1,"damageType":"Slashing"}`), nil},
		{"two instances", "FR-042", "happy", damage(`{"number":7,"dice":{"count":1,"faces":6,"modifier":4},"damageType":"Piercing"},{"number":17,"dice":{"count":5,"faces":6,"modifier":0},"damageType":"Poison"}`), nil},
		{"negative flat modifier", "FR-042", "unhappy (boundary)", damage(`{"number":1,"dice":{"count":1,"faces":4,"modifier":-1},"damageType":"Piercing"}`), nil},
		{"described damage type", "FR-042", "happy", damage(`{"number":7,"dice":{"count":2,"faces":6,"modifier":0},"text":"damage of the type chosen for a trait"}`), nil},
		{"unknown damage type", "FR-043", "unhappy (invalid input)", damage(`{"number":4,"damageType":"Arcane"}`), []string{a + "clauses/0/damage/0/damageType invalidValue"}},
		{"damage dice count 0", "FR-043", "unhappy (boundary)", damage(`{"number":4,"dice":{"count":0,"faces":4,"modifier":0},"damageType":"Fire"}`), []string{a + "clauses/0/damage/0/dice/count outOfRange"}},
		{"damage dice d7", "FR-043", "unhappy (boundary)", damage(`{"number":4,"dice":{"count":1,"faces":7,"modifier":0},"damageType":"Fire"}`), []string{a + "clauses/0/damage/0/dice/faces outOfRange"}},
		{"damage number -1", "FR-043", "unhappy (boundary)", damage(`{"number":-1,"damageType":"Fire"}`), []string{a + "clauses/0/damage/0/number outOfRange"}},
		{"damage with no type", "FR-042", "unhappy (invalid input)", damage(`{"number":4}`), []string{a + "clauses/0/damage/0/damageType missingField"}},
		{"damage with both types", "FR-042", "unhappy (invalid input)", damage(`{"number":4,"damageType":"Fire","text":"x"}`), []string{a + "clauses/0/damage/0 constraintViolation"}},
		{"underwater flag on Fire", "FR-101", "happy", damage(`{"number":7,"damageType":"Fire","underwaterResistanceDenied":true}`), nil},
		{"underwater flag on Cold", "FR-101", "unhappy (invalid input)", damage(`{"number":7,"damageType":"Cold","underwaterResistanceDenied":true}`), []string{a + "clauses/0/damage/0/underwaterResistanceDenied constraintViolation"}},
		{"underwater flag on a described type", "FR-101", "unhappy (invalid input)", damage(`{"number":7,"text":"x","underwaterResistanceDenied":true}`), []string{a + "clauses/0/damage/0/underwaterResistanceDenied constraintViolation"}},
		{"underwater flag false on Cold", "FR-101", "happy", damage(`{"number":7,"damageType":"Cold","underwaterResistanceDenied":false}`), nil},
		{"spellcasting", "FR-044", "happy", spell(`{"ability":"intelligence","dc":17,"attackBonus":9,"components":"requiring no Material components",` + gp + `}`), nil},
		{"spellcasting without groups", "FR-044", "unhappy (empty)", spell(`{"ability":"intelligence","groups":[]}`), []string{a + "spellcasting/groups constraintViolation"}},
		{"spell group without spells", "FR-045", "unhappy (boundary)", spell(`{"ability":"intelligence","groups":[{"frequency":{"kind":"at_will"},"spells":[]}]}`), []string{a + "spellcasting/groups/0/spells constraintViolation"}},
		{"frequency per day", "FR-045", "happy", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"per_day","count":3},"spells":[{"name":"Light"}]},{"frequency":{"kind":"per_day_each","count":1},"spells":[{"name":"Fly","note":"n","castAtLevel":1}]}]}`), nil},
		{"frequency count 0", "FR-045", "unhappy (boundary)", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"per_day","count":0},"spells":[{"name":"Light"}]}]}`), []string{a + "spellcasting/groups/0/frequency/count outOfRange"}},
		{"frequency kind", "FR-045", "unhappy (invalid input)", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"daily"},"spells":[{"name":"Light"}]}]}`), []string{a + "spellcasting/groups/0/frequency/kind invalidValue"}},
		{"castAtLevel 0", "FR-046", "unhappy (boundary)", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"at_will"},"spells":[{"name":"Light","castAtLevel":0}]}]}`), []string{a + "spellcasting/groups/0/spells/0/castAtLevel outOfRange"}},
		{"castAtLevel 9", "FR-046", "unhappy (boundary)", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"at_will"},"spells":[{"name":"Light","castAtLevel":9}]}]}`), nil},
		{"castAtLevel 10", "FR-046", "unhappy (boundary)", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"at_will"},"spells":[{"name":"Light","castAtLevel":10}]}]}`), []string{a + "spellcasting/groups/0/spells/0/castAtLevel outOfRange"}},
		{"spell name blank", "FR-046", "unhappy (invalid input)", spell(`{"ability":"wisdom","groups":[{"frequency":{"kind":"at_will"},"spells":[{"name":" "}]}]}`), []string{a + "spellcasting/groups/0/spells/0/name invalidValue"}},
		{"spellcasting ability", "FR-044", "unhappy (invalid input)", spell(`{"ability":"luck",` + gp + `}`), []string{a + "spellcasting/ability invalidValue"}},
		{"spellcasting DC 0", "FR-044", "unhappy (boundary)", spell(`{"ability":"wisdom","dc":0,` + gp + `}`), []string{a + "spellcasting/dc outOfRange"}},
		{"second Multiattack", "FR-036", "unhappy (rule interaction)", monActions(monPartJSON("Multiattack", ""), monPartJSON("Multiattack", "")), []string{"/monsters/0/actions/1/name constraintViolation"}},
		{"legendary actions", "FR-040", "happy", monWith(`"experiencePoints":0`, `"experiencePoints":0,"legendaryActions":{"uses":3,"usesInLair":4,"actions":[`+monPartJSON("Move", "")+`]}`), nil},
		{"empty legendary actions", "FR-040", "unhappy (empty)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"legendaryActions":{"uses":3,"actions":[]}`), []string{"/monsters/0/legendaryActions/actions constraintViolation"}},
		{"zero legendary uses", "FR-040", "unhappy (boundary)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"legendaryActions":{"uses":0,"actions":[`+monPartJSON("Move", "")+`]}`), []string{"/monsters/0/legendaryActions/uses outOfRange"}},
		{"zero lair legendary uses", "FR-040", "unhappy (boundary)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"legendaryActions":{"uses":1,"usesInLair":0,"actions":[`+monPartJSON("Move", "")+`]}`), []string{"/monsters/0/legendaryActions/usesInLair outOfRange"}},
		{"legendary part problem", "FR-040", "unhappy (invalid input)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"legendaryActions":{"uses":1,"actions":[`+monPartJSON(" ", "")+`]}`), []string{"/monsters/0/legendaryActions/actions/0/name invalidValue"}},
		{"reaction and bonus action", "FR-033", "happy", monWith(`"experiencePoints":0`, `"experiencePoints":0,"traits":[`+monPartJSON("T", "")+`],"bonusActions":[`+monPartJSON("B", "")+`],"reactions":[`+monPartJSON("R", "")+`]`), nil},
		{"part list not an array", "FR-033", "unhappy (invalid input)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"traits":{}`), []string{"/monsters/0/traits wrongType"}},
	})
}

// Requirement: monSpec FR-098, FR-099, FR-102 and table 4.6c - the Multiattack plan: its shape, the rules inside it
// and the resolution of its names, with the problem kind and path of the table.
// Path: per case.
func TestDecodeMonsterPlan(t *testing.T) {
	t.Parallel()
	const pl = "/monsters/0/actions/0/plan/"
	slam := monPartJSON("Slam", "")
	visage := monPartJSON("Visage", "")
	hasten := monPartJSON("Hasten", "")
	multi := func(plan string) string { return monPartJSON("Multiattack", `,"plan":`+plan) }
	doc := func(plan string, others ...string) string {
		return monActions(append([]string{multi(plan)}, others...)...)
	}
	group := func(count int, parts ...string) string {
		return `{"count":` + string(rune('0'+count)) + `,"parts":["` + strings.Join(parts, `","`) + `"]}`
	}
	variant := func(extra string, groups ...string) string {
		return `{` + extra + `"groups":[` + strings.Join(groups, ",") + `]}`
	}
	plan := func(variants ...string) string { return `{"variants":[` + strings.Join(variants, ",") + `]}` }
	runDocCases(t, []docCase{
		{"any combination", monSpec + " FR-098", "happy", doc(plan(variant("", group(3, "Slam", "Visage"))), slam, visage), nil},
		{"replace a named part", "FR-098", "happy", doc(plan(`{"groups":[`+group(1, "Slam")+`,`+group(1, "Visage")+`],"replacements":[{"count":1,"scope":{"kind":"one_attack_part","part":"Visage"},"options":[{"part":"Slam"}],"limit":"once"}]}`), slam, visage), nil},
		{"additional use", "FR-098", "happy", doc(plan(`{"groups":[`+group(2, "Slam")+`],"additionalUses":[{"options":["Visage"]}]}`), slam, visage), nil},
		{"conditional variant", "FR-098", "happy", doc(plan(variant("", group(2, "Slam")), variant(`"requiresUsed":"Hasten",`, group(3, "Slam"))), slam, hasten), nil},
		{"part or spell option", "FR-098", "happy", doc(plan(`{"groups":[`+group(2, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"any_attack"},"options":[{"part":"Visage"},{"part":"Slam","spell":{"name":"Scorching Ray","castAtLevel":3}}],"limit":"anyNumber"}]}`), slam, visage), nil},
		{"no variants", "FR-098", "unhappy (empty)", doc(`{"variants":[]}`), []string{pl + "variants constraintViolation"}},
		{"variants missing", "FR-098", "unhappy (boundary)", doc(`{}`), []string{pl + "variants missingField"}},
		{"plan not an object", "FR-098", "unhappy (invalid input)", doc(`[]`), []string{"/monsters/0/actions/0/plan wrongType"}},
		{"conditional without sibling", "FR-098", "unhappy (rule interaction)", doc(plan(variant(`"requiresUsed":"Hasten",`, group(3, "Slam"))), slam, hasten), []string{pl + "variants/0/requiresUsed constraintViolation"}},
		{"overlapping groups", "FR-098", "unhappy (rule interaction)", doc(plan(variant("", group(1, "Slam"), group(1, "Slam", "Visage"))), slam, visage), []string{pl + "variants/0/groups/1 constraintViolation"}},
		{"repeated name", "FR-098", "unhappy (invalid input)", doc(plan(variant("", group(1, "Slam", "Slam"))), slam), []string{pl + "variants/0/groups/0/parts/1 duplicate"}},
		{"repeated additional use", "FR-098", "unhappy (invalid input)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"additionalUses":[{"options":["Visage","Visage"]}]}`), slam, visage), []string{pl + "variants/0/additionalUses/0/options/1 duplicate"}},
		{"repeated replacement option", "FR-098", "unhappy (invalid input)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"any_attack"},"options":[{"part":"Visage"},{"part":"Visage"}],"limit":"once"}]}`), slam, visage), []string{pl + "variants/0/replacements/0/options/1/part duplicate"}},
		{"scope part not in a group", "FR-098", "unhappy (rule interaction)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"one_attack_part","part":"Tail"},"options":[{"part":"Visage"}],"limit":"once"}]}`), slam, visage), []string{pl + "variants/0/replacements/0/scope/part constraintViolation"}},
		{"zero group count", "FR-098", "unhappy (boundary)", doc(plan(`{"groups":[{"count":0,"parts":["Slam"]}]}`), slam), []string{pl + "variants/0/groups/0/count outOfRange"}},
		{"no groups", "FR-098", "unhappy (empty)", doc(plan(`{"groups":[]}`), slam), []string{pl + "variants/0/groups outOfRange"}},
		{"spell level 10", "FR-098", "unhappy (boundary)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"any_attack"},"options":[{"part":"Visage","spell":{"name":"S","castAtLevel":10}}],"limit":"once"}]}`), slam, visage), []string{pl + "variants/0/replacements/0/options/0/spell/castAtLevel outOfRange"}},
		{"replacement count 0", "FR-098", "unhappy (boundary)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":0,"scope":{"kind":"any_attack"},"options":[{"part":"Visage"}],"limit":"once"}]}`), slam, visage), []string{pl + "variants/0/replacements/0/count outOfRange"}},
		{"unknown scope kind", "FR-098", "unhappy (invalid input)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"every"},"options":[{"part":"Visage"}],"limit":"once"}]}`), slam, visage), []string{pl + "variants/0/replacements/0/scope/kind invalidValue"}},
		{"unknown limit", "FR-098", "unhappy (invalid input)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"any_attack"},"options":[{"part":"Visage"}],"limit":"twice"}]}`), slam, visage), []string{pl + "variants/0/replacements/0/limit invalidValue"}},
		{"plan on another part", "FR-098", "unhappy (invalid input)", monActions(monPartJSON("Bite", `,"plan":`+plan(variant("", group(1, "Bite"))))), []string{"/monsters/0/actions/0/plan constraintViolation"}},
		{"plan on a trait named Multiattack", "FR-036", "unhappy (rule interaction)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"traits":[`+multi(plan(variant("", group(1, "Nope"))))+`]`), []string{"/monsters/0/traits/0/plan constraintViolation"}},
		{"plan on a bonus action named Multiattack", "FR-036", "unhappy (rule interaction)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"bonusActions":[`+multi(plan(variant("", group(1, "Nope"))))+`]`), []string{"/monsters/0/bonusActions/0/plan constraintViolation"}},
		{"unresolved: a bonus action", "FR-099", "unhappy (missing content)", monWith(`"experiencePoints":0`, `"experiencePoints":0,"actions":[`+multi(plan(variant("", group(1, "Slam"))))+`],"bonusActions":[`+slam+`]`), []string{pl + "variants/0/groups/0/parts/0 constraintViolation"}},
		{"unresolved: unknown name", "FR-099", "unhappy (missing content)", doc(plan(variant("", group(1, "Slam")))), []string{pl + "variants/0/groups/0/parts/0 constraintViolation"}},
		{"unresolved: shared name", "FR-099", "unhappy (missing content)", doc(plan(variant("", group(1, "Slam"))), slam, slam), []string{pl + "variants/0/groups/0/parts/0 constraintViolation"}},
		{"unresolved: additional use", "FR-099", "unhappy (missing content)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"additionalUses":[{"options":["Gone"]}]}`), slam), []string{pl + "variants/0/additionalUses/0/options/0 constraintViolation"}},
		{"unresolved: replacement option", "FR-099", "unhappy (missing content)", doc(plan(`{"groups":[`+group(1, "Slam")+`],"replacements":[{"count":1,"scope":{"kind":"one_attack_part","part":"Slam"},"options":[{"part":"Gone"}],"limit":"once"}]}`), slam), []string{pl + "variants/0/replacements/0/options/0/part constraintViolation"}},
		{"requiresUsed in a bonus action", "FR-099", "happy", monWith(`"experiencePoints":0`, `"experiencePoints":0,"actions":[`+multi(plan(variant("", group(2, "Slam")), variant(`"requiresUsed":"Hasten",`, group(3, "Slam"))))+`,`+slam+`],"bonusActions":[`+hasten+`]`), nil},
		{"requiresUsed unresolved", "FR-099", "unhappy (missing content)", doc(plan(variant("", group(2, "Slam")), variant(`"requiresUsed":"Hasten",`, group(3, "Slam"))), slam), []string{pl + "variants/1/requiresUsed constraintViolation"}},
		{"the plan names are not checked after another part failed", "FR-099", "unhappy (invalid input)", doc(plan(variant("", group(1, "Slam"))), monPartJSON(" ", "")), []string{"/monsters/0/actions/1/name invalidValue"}},
	})
}

// Requirement: monSpec FR-048, FR-051 - a document with every member of a stat block decodes to the monster the
// constructors build from the same values.
// Path: happy.
func TestDecodeMonsterEveryMember(t *testing.T) {
	t.Parallel()
	const doc = `{"schemaVersion":1,"monsters":[{"id":"fixture_beast","name":"Fixture Beast","source":{"document":"Fixture","section":"Animals","page":344},` +
		`"sizes":["Medium","Small"],"swarmOf":"Tiny","type":"Beast","tags":["Tag"],"alignment":"Unaligned","armorClass":14,` +
		`"initiative":{"modifier":-2,"score":8},"hitPoints":{"average":10,"count":3,"faces":10,"modifier":-4},` +
		`"speeds":[{"modes":["walk"],"feet":30},{"modes":["climb","fly"],"feet":20,"hover":true,"qualifier":"choice"}],` +
		`"abilities":{"str":{"score":21,"save":5},"dex":{"score":10,"save":0},"con":{"score":10,"save":0},"int":{"score":18,"save":8},"wis":{"score":10,"save":0},"cha":{"score":1,"save":-5}},` +
		`"skills":{"Stealth":4},"vulnerabilities":[{"damageType":"Piercing","qualifier":"from weapons"}],"resistances":[{"text":"Chosen"}],` +
		`"immunities":[{"damageType":"Poison"},{"condition":"Charmed","qualifier":"with a spell"}],"gear":["Item"],` +
		`"senses":{"special":[{"kind":"Darkvision","feet":60,"qualifier":"q"}],"passivePerception":12},` +
		`"languages":{"phrases":["Deep Speech"],"telepathyFeet":120,"telepathyQualifier":"q"},"challengeRating":"5","experiencePoints":1800,"experiencePointsInLair":2300,` +
		`"traits":[{"name":"Magic","usage":{"kind":"per_day","count":3,"inLairCount":4},"text":"T.","spellcasting":{"ability":"wisdom","dc":15,"attackBonus":7,"components":"c","groups":[{"frequency":{"kind":"per_day_each","count":2},"spells":[{"name":"Light","note":"n","castAtLevel":2}]}]}}],` +
		`"actions":[{"name":"Multiattack","usage":{"kind":"unlimited"},"text":"T.","plan":{"variants":[{"groups":[{"count":2,"parts":["Bite"]}],"additionalUses":[{"options":["Breath"]}],"replacements":[{"count":1,"scope":{"kind":"one_attack_part","part":"Bite"},"options":[{"part":"Breath","spell":{"name":"Ray","castAtLevel":3}}],"limit":"anyNumber"}]}]}},` +
		`{"name":"Bite","usage":{"kind":"other","text":"o"},"text":"T.","attack":{"kind":"melee","bonus":9,"qualifier":"q","reach":5},"savingThrow":{"ability":"constitution","dc":12,"target":"one creature","prefix":"P"},` +
		`"clauses":[{"label":"hit","text":"x","damage":[{"number":5,"dice":{"count":1,"faces":6,"modifier":2},"damageType":"Fire","underwaterResistanceDenied":true},{"number":1,"text":"described"}]},{"label":"success","text":"y","successRider":{"kind":"immune_to_part","hours":24}}]},` +
		`{"name":"Breath","usage":{"kind":"recharge","low":5,"high":6},"text":"T.","attack":{"kind":"ranged","bonus":-1,"range":{"normal":80,"long":320}}}],` +
		`"bonusActions":[{"name":"Dash","usage":{"kind":"recharge_after_rest"},"text":"T."}],"reactions":[{"name":"Parry","usage":{"kind":"unlimited"},"text":"T."}],` +
		`"legendaryActions":{"uses":3,"usesInLair":4,"actions":[{"name":"Move","usage":{"kind":"unlimited"},"text":"T."}]}}]}`
	got := monDecode(t, doc)
	exp := func(name string, f content.StatBlockPartFields) content.StatBlockPart { return monPart(t, name, f) }
	spellGroup := must(content.NewMonsterSpellGroup(must(content.NewSpellFrequencyPerDayEach(2)), []content.MonsterSpellEntry{must(content.NewMonsterSpellEntry("Light", "n", 2))}))
	magic := exp("Magic", content.StatBlockPartFields{
		Usage: must(content.NewPartPerDay(3, 4)),
		Spellcasting: must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
			Ability: ability.Wisdom, SaveDC: 15, HasSaveDC: true, AttackBonus: 7, HasAttackBonus: true, Components: "c", Groups: []content.MonsterSpellGroup{spellGroup},
		})),
	})
	opt := must(content.NewReplacementOption("Breath", "Ray", 3))
	plan := must(content.NewMultiattackPlan([]content.MultiattackVariant{must(content.NewMultiattackVariant("",
		[]content.AttackSlotGroup{must(content.NewAttackSlotGroup(2, []string{"Bite"}))},
		[]content.AdditionalUse{must(content.NewAdditionalUse([]string{"Breath"}))},
		[]content.ReplacementRule{must(content.NewReplacementRule(1, must(content.NewReplacementScopeOneAttackPart("Bite")), []content.ReplacementOption{opt}, content.ReplaceAnyNumber))},
	))}))
	fire := must(content.NewStatBlockDamage(5, dice.MustExpression(1, dice.D6(), 2), must(content.NewSpecificDamageType(content.DamageFire)), true))
	described := must(content.NewStatBlockDamage(1, dice.Expression{}, must(content.NewDescribedDamageType("described")), false))
	bite := exp("Bite", content.StatBlockPartFields{
		Usage:       must(content.NewPartUsageOther("o")),
		Attack:      must(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Bonus: 9, Qualifier: "q", Reach: 5, HasReach: true})),
		SavingThrow: must(content.NewStatBlockSavingThrow(ability.Constitution, 12, "one creature", "P")),
		Clauses: []content.EffectClause{
			must(content.NewEffectClause(content.ClauseHit, "x", nil, []content.StatBlockDamage{fire, described})),
			must(content.NewEffectClause(content.ClauseSuccess, "y", must(content.NewImmuneToPartForHours(24)), nil)),
		},
	})
	breath := exp("Breath", content.StatBlockPartFields{
		Usage:  must(content.NewPartRecharge(5, 6)),
		Attack: must(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackRanged, Bonus: -1, Range: must(content.NewStatBlockAttackRange(80, 320))})),
	})
	fly := must(content.NewSpeedEntry([]content.SpeedMode{content.SpeedClimb, content.SpeedFly}, 20, true, "choice"))
	scores := must(ability.NewScores(map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(21), ability.Dexterity: ability.MustScore(10), ability.Constitution: ability.MustScore(10),
		ability.Intelligence: ability.MustScore(18), ability.Wisdom: ability.MustScore(10), ability.Charisma: ability.MustScore(1),
	}))
	saves := monSaves(0)
	saves[ability.Strength], saves[ability.Intelligence], saves[ability.Charisma] = 5, 8, -5
	f := content.MonsterFields{
		Sizes: []content.CreatureSize{content.SizeMedium, content.SizeSmall}, Swarm: must(content.NewSwarmComposition(content.SizeTiny)),
		Type: content.CreatureTypeBeast, Tags: []string{"Tag"}, Alignment: content.AlignmentUnaligned, ArmorClass: 14,
		Initiative: must(content.NewStatBlockInitiative(-2, 8)), HitPoints: must(content.NewHitPointsEntry(10, dice.MustExpression(3, dice.D10(), -4))),
		Speeds: []content.SpeedEntry{monSpeed(t, 30), fly}, Abilities: must(content.NewAbilityBlock(scores, saves)),
		Skills:          map[content.Skill]int{content.SkillStealth: 4},
		Vulnerabilities: []content.DefenseEntry{must(content.NewDamageTypeDefense(content.DamagePiercing, "from weapons"))},
		Resistances:     []content.DefenseEntry{must(content.NewDescribedDefense("Chosen"))},
		Immunities:      []content.DefenseEntry{must(content.NewDamageTypeDefense(content.DamagePoison, "")), must(content.NewConditionDefense(content.ConditionCharmed, "with a spell"))},
		Gear:            []string{"Item"}, Senses: []content.SenseEntry{must(content.NewSenseEntry(content.SenseDarkvision, 60, "q"))}, PassivePerception: 12,
		Languages:       must(must(content.NewLanguagesEntry([]string{"Deep Speech"})).WithTelepathy(120, "q")),
		ChallengeRating: content.MustChallengeRating("5"), ExperiencePoints: 1800, ExperiencePointsInLair: 2300,
		Traits:           []content.StatBlockPart{magic},
		Actions:          []content.StatBlockPart{exp("Multiattack", content.StatBlockPartFields{Plan: plan}), bite, breath},
		BonusActions:     []content.StatBlockPart{exp("Dash", content.StatBlockPartFields{Usage: content.PartRechargeAfterRest{}})},
		Reactions:        []content.StatBlockPart{exp("Parry", content.StatBlockPartFields{})},
		LegendaryActions: must(content.NewLegendaryActions(3, 4, []content.StatBlockPart{exp("Move", content.StatBlockPartFields{})})),
	}
	want := must(content.NewMonster("fixture_beast", "Fixture Beast", content.MustSourceCitation("Fixture", content.MonsterAnimalsSection, 344), f))
	if !got.Equal(want) {
		t.Error("decoded monster differs from the one built from the same values [FR-048]")
	}
	if sc, ok := got.Spellcasting(); !ok || len(sc.Groups()) != 1 {
		t.Error("the decoded Spellcasting is missing [FR-044]")
	}
}

// Requirement: monSpec section 7 and schema FR-079 to FR-083, FR-101 - the Loader over the real decoder counts one
// entry of kind monster and the problems of a rejected monster document by problem kind, with no new signal; a
// monster registers no reference site, so a set of monsters validates.
// Path: happy and unhappy (invalid input).
func TestLoaderCountsMonsters(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	loader := mustLoader(t, content.JSONDecoder{}, content.WithTelemetry(rec.Telemetry()))
	res, err := loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(monDoc(monJSON))})
	loaded, ok := res.(content.LoadSetLoaded)
	if !ok || err != nil {
		t.Fatalf("LoadSet = (%T, %v), want LoadSetLoaded", res, err)
	}
	var kinds []string
	for _, m := range rec.Meter.MeasurementsNamed(content.MetricEntriesLoaded) {
		kinds = append(kinds, attrsOf(m.Attrs)[content.AttrKind].(string))
	}
	if want := []string{content.ValueKindMonster}; !slices.Equal(kinds, want) {
		t.Errorf("entries_loaded kinds = %v, want %v", kinds, want)
	}
	if _, err := content.ValidateSet(loaded.Set()); err != nil {
		t.Errorf("ValidateSet over a monster set = %v, want nil [FR-049]", err)
	}
	res, err = loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(monWith(`"armorClass":12`, `"armorClass":-1`))})
	if rej, ok := res.(content.LoadSetRejected); !ok || err != nil || len(rej.Problems()) != 1 {
		t.Fatalf("LoadSet(bad) = (%T, %v), want one rejected problem", res, err)
	}
	var counted []string
	for _, m := range rec.Meter.MeasurementsNamed(content.MetricDecodeErrors) {
		counted = append(counted, attrsOf(m.Attrs)[content.AttrProblemKind].(string))
	}
	if want := []string{content.ValueProblemOutOfRange}; !slices.Equal(counted, want) {
		t.Errorf("decode_errors kinds = %v, want %v", counted, want)
	}
}
