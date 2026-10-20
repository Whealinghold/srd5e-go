package content_test

import (
	"fmt"
	"strings"
	"testing"
)

const cp = "/classes/0/"

// allRows lists one problem per row of the 20-row progression.
func allRows(suffix string) []string {
	var out []string
	for i := range 20 {
		out = append(out, fmt.Sprintf("%sprogression/%d/%s", cp, i, suffix))
	}
	return out
}

// Requirement: per case
// Path: per case.
func TestDecodeClassHeaderAndCore(t *testing.T) {
	t.Parallel()
	c := func(old, repl string) string { return classDoc(mutate(wizardClassJSON(), old, repl)) }
	runDocCases(t, []docCase{
		{"valid", classSpec + " FR-002", "happy", classDoc(wizardClassJSON()), nil},
		{"missing hitPointDie", classSpec + " FR-002", "unhappy (missing content)", c(`"hitPointDie":"D6",`, ``), []string{cp + "hitPointDie missingField"}},
		{"missing columns", classSpec + " FR-002", "unhappy (missing content)", c(`"columns":[],`, ``), []string{cp + "columns missingField"}},
		{"missing subclasses", classSpec + " FR-002", "unhappy (missing content)", c(`,"subclasses":[]`, ``), []string{cp + "subclasses missingField"}},
		{"missing toolProficiencies", classSpec + " FR-002", "unhappy (missing content)", c(`"toolProficiencies":[],`, ``), []string{cp + "toolProficiencies missingField"}},
		{"unknown key", classSpec + " FR-002", "unhappy (invalid input)", c(`"name":"Fixture",`, `"name":"Fixture","extra":1,`), []string{cp + "extra unknownField"}},
		{"source", classSpec + " FR-009", "happy", c(`"name":"Fixture",`, `"name":"Fixture","source":{"document":"D","section":"S","page":3},`), nil},
		{"progression is a string", classSpec + " FR-006", "unhappy (rule interaction)", classDoc(strings.Replace(wizardClassJSON(), `"progression":[`+strings.Join(classRows("fixture", slotRowJSON, `{}`), ",")+`]`, `"progression":"x"`, 1)), []string{cp + "progression wrongType"}},
		{"feature is not an object", classSpec + " FR-006", "unhappy (rule interaction)", c(`"features":[{"id":"fixture_alpha"`, `"features":[3,{"id":"fixture_alpha"`), []string{cp + "features/0 wrongType"}},
		{"unknown primary kind", classSpec + " FR-011", "unhappy (invalid input)", c(`"allOf"`, `"either"`), []string{cp + "primaryAbility/kind invalidValue"}},
		{"allOf empty", classSpec + " FR-012", "unhappy (boundary)", c(`"abilities":["intelligence"]`, `"abilities":[]`), []string{cp + "primaryAbility/abilities outOfRange"}},
		{"allOf repeated", classSpec + " FR-012", "unhappy (boundary)", c(`"abilities":["intelligence"]`, `"abilities":["intelligence","intelligence"]`), []string{cp + "primaryAbility/abilities/1 duplicate"}},
		{"empty primary kind", classSpec + " FR-011", "unhappy (boundary)", c(`"allOf"`, `""`), []string{cp + "primaryAbility/kind invalidValue"}},
		{"high primary kind", classSpec + " FR-011", "unhappy (boundary)", c(`"allOf"`, `"zzz"`), []string{cp + "primaryAbility/kind invalidValue"}},
		{"empty skill options kind", classSpec + " FR-019", "unhappy (boundary)", c(`"listed"`, `""`), []string{cp + "skillChoice/options/kind invalidValue"}},
		{"high skill options kind", classSpec + " FR-019", "unhappy (boundary)", c(`"listed"`, `"zzz"`), []string{cp + "skillChoice/options/kind invalidValue"}},
		{"empty tool kind", classSpec + " FR-023", "unhappy (boundary)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"","name":"x"}]`), []string{cp + "toolProficiencies/0/kind invalidValue"}},
		{"high tool kind", classSpec + " FR-023", "unhappy (boundary)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"zzz","name":"x"}]`), []string{cp + "toolProficiencies/0/kind invalidValue"}},
		{"empty spellcasting kind", classSpec + " FR-028", "unhappy (boundary)", c(`"kind":"slots"`, `"kind":""`), []string{cp + "spellcasting/kind invalidValue"}},
		{"high spellcasting kind", classSpec + " FR-028", "unhappy (boundary)", c(`"kind":"slots"`, `"kind":"zzz"`), []string{cp + "spellcasting/kind invalidValue"}},
		{"two invalid weapon categories", classSpec + " FR-021", "unhappy (boundary)", c(`[{"category":"Simple"}]`, `[{"category":"x"},{"category":"y"}]`), []string{cp + "weaponProficiencies/0/category invalidValue", cp + "weaponProficiencies/1/category invalidValue"}},
		{"weapon without a category", classSpec + " FR-021", "unhappy (missing content)", c(`[{"category":"Simple"}]`, `[{}]`), []string{cp + "weaponProficiencies/0/category missingField"}},
		{"anyOf of one", classSpec + " FR-013", "unhappy (boundary)", c(`{"kind":"allOf","abilities":["intelligence"]}`, `{"kind":"anyOf","abilities":["strength"]}`), []string{cp + "primaryAbility/abilities outOfRange"}},
		{"anyOf of two", classSpec + " FR-013", "happy", c(`{"kind":"allOf","abilities":["intelligence"]}`, `{"kind":"anyOf","abilities":["strength","dexterity"]}`), nil},
		{"anyOf repeated", classSpec + " FR-013", "unhappy (boundary)", c(`{"kind":"allOf","abilities":["intelligence"]}`, `{"kind":"anyOf","abilities":["strength","strength"]}`), []string{cp + "primaryAbility/abilities/1 duplicate"}},
		{"ability spelling", classSpec + " FR-015", "unhappy (invalid input)", c(`"abilities":["intelligence"]`, `"abilities":["Intelligence"]`), []string{cp + "primaryAbility/abilities/0 invalidValue"}},
		{"primary is an array", classSpec + " FR-011", "unhappy (invalid input)", c(`"primaryAbility":{"kind":"allOf","abilities":["intelligence"]}`, `"primaryAbility":[]`), []string{cp + "primaryAbility wrongType"}},
		{"die D12", classSpec + " FR-017", "happy", c(`"D6"`, `"D12"`), nil},
		{"die lower case", classSpec + " FR-017", "unhappy (boundary)", c(`"D6"`, `"d8"`), []string{cp + "hitPointDie invalidValue"}},
		{"die D4", classSpec + " FR-017", "unhappy (boundary)", c(`"D6"`, `"D4"`), []string{cp + "hitPointDie invalidValue"}},
		{"die number", classSpec + " FR-017", "unhappy (invalid input)", c(`"D6"`, `8`), []string{cp + "hitPointDie wrongType"}},
		{"no saves", classSpec + " FR-018", "unhappy (boundary)", c(`["intelligence","wisdom"]`, `[]`), []string{cp + "savingThrowProficiencies outOfRange"}},
		{"seven saves", classSpec + " FR-018", "unhappy (boundary)", c(`["intelligence","wisdom"]`, `["strength","dexterity","constitution","intelligence","wisdom","charisma","charisma"]`), []string{cp + "savingThrowProficiencies outOfRange"}},
		{"six saves", classSpec + " FR-018", "unhappy (boundary)", c(`["intelligence","wisdom"]`, `["strength","dexterity","constitution","intelligence","wisdom","charisma"]`), nil},
		{"repeated save", classSpec + " FR-018", "unhappy (boundary)", c(`["intelligence","wisdom"]`, `["wisdom","wisdom"]`), []string{cp + "savingThrowProficiencies/1 duplicate"}},
		{"picks zero", classSpec + " FR-019", "unhappy (boundary)", c(`"picks":2`, `"picks":0`), []string{cp + "skillChoice/picks outOfRange"}},
		{"picks beyond the list", classSpec + " FR-019", "unhappy (boundary)", c(`"picks":2`, `"picks":4`), []string{cp + "skillChoice/options/skills outOfRange"}},
		{"picks equal the list", classSpec + " FR-019", "unhappy (boundary)", c(`"picks":2`, `"picks":3`), nil},
		{"any skills", classSpec + " FR-019", "happy", c(`{"kind":"listed","skills":["Arcana","History","Insight"]}`, `{"kind":"any"}`), nil},
		{"unknown skill", classSpec + " FR-019", "unhappy (invalid input)", c(`"Arcana"`, `"Magic"`), []string{cp + "skillChoice/options/skills/0 invalidValue"}},
		{"repeated skill", classSpec + " FR-019", "unhappy (boundary)", c(`"History"`, `"Arcana"`), []string{cp + "skillChoice/options/skills/1 duplicate"}},
		{"empty skills", classSpec + " FR-019", "unhappy (empty case)", c(`["Arcana","History","Insight"]`, `[]`), []string{cp + "skillChoice/options/skills outOfRange"}},
		{"unknown skill options kind", classSpec + " FR-019", "unhappy (invalid input)", c(`"listed"`, `"some"`), []string{cp + "skillChoice/options/kind invalidValue"}},
		{"weapon category twice", classSpec + " FR-021", "unhappy (boundary)", c(`[{"category":"Simple"}]`, `[{"category":"Simple"},{"category":"Simple"}]`), []string{cp + "weaponProficiencies/1 duplicate"}},
		{"weapon filter", classSpec + " FR-021", "happy", c(`[{"category":"Simple"}]`, `[{"category":"Martial","requiresAnyProperty":["Finesse","Light"]}]`), nil},
		{"weapon empty filter", classSpec + " FR-021", "unhappy (empty case)", c(`[{"category":"Simple"}]`, `[{"category":"Martial","requiresAnyProperty":[]}]`), []string{cp + "weaponProficiencies/0/requiresAnyProperty outOfRange"}},
		{"weapon repeated property", classSpec + " FR-021", "unhappy (boundary)", c(`[{"category":"Simple"}]`, `[{"category":"Martial","requiresAnyProperty":["Light","Light"]}]`), []string{cp + "weaponProficiencies/0/requiresAnyProperty/1 duplicate"}},
		{"weapon category spelling", classSpec + " FR-021", "unhappy (invalid input)", c(`"Simple"`, `"simple"`), []string{cp + "weaponProficiencies/0/category invalidValue"}},
		{"armor repeated", classSpec + " FR-022", "unhappy (boundary)", c(`"armorTraining":[]`, `"armorTraining":["Light","Light"]`), []string{cp + "armorTraining/1 duplicate"}},
		{"armor spelling", classSpec + " FR-022", "unhappy (invalid input)", c(`"armorTraining":[]`, `"armorTraining":["light"]`), []string{cp + "armorTraining/0 invalidValue"}},
		{"armor four", classSpec + " FR-022", "happy", c(`"armorTraining":[]`, `"armorTraining":["Light","Medium","Heavy","Shield"]`), nil},
		{"tools", classSpec + " FR-023", "happy", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"specific","name":"Kit"},{"kind":"choice","categories":["Instrument"],"picks":3}]`), nil},
		{"tool picks zero", classSpec + " FR-023", "unhappy (boundary)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"choice","categories":["Instrument"],"picks":0}]`), []string{cp + "toolProficiencies/0/picks outOfRange"}},
		{"tool no categories", classSpec + " FR-023", "unhappy (empty case)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"choice","categories":[],"picks":1}]`), []string{cp + "toolProficiencies/0/categories outOfRange"}},
		{"tool repeated category", classSpec + " FR-023", "unhappy (boundary)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"choice","categories":["A","A"],"picks":1}]`), []string{cp + "toolProficiencies/0/categories/1 duplicate"}},
		{"tool blank name", classSpec + " FR-023", "unhappy (invalid input)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"specific","name":" "}]`), []string{cp + "toolProficiencies/0/name invalidValue"}},
		{"tool unknown kind", classSpec + " FR-023", "unhappy (invalid input)", c(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"any","name":"x"}]`), []string{cp + "toolProficiencies/0/kind invalidValue"}},
		{"equipment option C", classSpec + " FR-024", "happy", c(`"B":{"items":[],"gold":50}`, `"B":{"items":[],"gold":50},"C":{"items":[],"gold":1}`), nil},
		{"equipment option D", classSpec + " FR-024", "unhappy (invalid input)", c(`"B":{"items":[],"gold":50}`, `"B":{"items":[],"gold":50},"D":{"items":[],"gold":1}`), []string{cp + "startingEquipment/D unknownField"}},
		{"equipment missing B", classSpec + " FR-024", "unhappy (missing content)", c(`,"B":{"items":[],"gold":50}`, ``), []string{cp + "startingEquipment/B missingField"}},
		{"equipment quantity zero", classSpec + " FR-024", "unhappy (boundary)", c(`"quantity":1`, `"quantity":0`), []string{cp + "startingEquipment/A/items/0/quantity outOfRange"}},
		{"equipment negative gold", classSpec + " FR-024", "unhappy (boundary)", c(`"gold":5`, `"gold":-1`), []string{cp + "startingEquipment/A/gold outOfRange"}},
		{"equipment zero gold", classSpec + " FR-024", "unhappy (boundary)", c(`"gold":5`, `"gold":0`), nil},
		{"equipment empty items", classSpec + " FR-024", "unhappy (empty case)", c(`"B":{"items":[],"gold":50}`, `"B":{"items":[],"gold":75}`), nil},
		{"spellcasting unknown kind", classSpec + " FR-028", "unhappy (invalid input)", c(`"kind":"slots"`, `"kind":"ritual"`), []string{cp + "spellcasting/kind invalidValue"}},
		{"spellcasting ability spelling", classSpec + " FR-028", "unhappy (invalid input)", c(`"ability":"intelligence"`, `"ability":"Int"`), []string{cp + "spellcasting/ability invalidValue"}},
		{"spellcasting cantrips flag missing", classSpec + " FR-028", "unhappy (missing content)", c(`,"cantrips":true}`, `}`), []string{cp + "spellcasting/cantrips missingField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeClassMulticlassGrants(t *testing.T) {
	t.Parallel()
	const none = `"multiclassGrants":{"weaponProficiencies":[],"armorTraining":[],"toolProficiencies":[]}`
	core := func(old, repl string) string { return mutate(wizardClassJSON(), old, repl) }
	grants := func(base, body string) string {
		return classDoc(mutate(base, none, `"multiclassGrants":{`+body+`}`))
	}
	armorCore := core(`"armorTraining":[]`, `"armorTraining":["Light"]`)
	weaponCore := core(`[{"category":"Simple"}]`, `[{"category":"Martial","requiresAnyProperty":["Light","Finesse"]}]`)
	toolCore := core(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"specific","name":"Kit"},{"kind":"choice","categories":["Instrument"],"picks":2}]`)
	const emptyW, emptyA, emptyT = `"weaponProficiencies":[]`, `"armorTraining":[]`, `"toolProficiencies":[]`
	m := cp + "multiclassGrants/"
	runDocCases(t, []docCase{
		{"within the core armor", classSpec + " FR-026", "happy", grants(armorCore, emptyW+`,"armorTraining":["Light"],`+emptyT), nil},
		{"armor beyond the core", classSpec + " FR-026", "unhappy (rule interaction)", grants(armorCore, emptyW+`,"armorTraining":["Heavy"],`+emptyT), []string{m + "armorTraining/0 constraintViolation"}},
		{"weapon within the filter", classSpec + " FR-026", "happy", grants(weaponCore, `"weaponProficiencies":[{"category":"Martial","requiresAnyProperty":["Light"]}],`+emptyA+`,`+emptyT), nil},
		{"weapon beyond the filter", classSpec + " FR-026", "unhappy (rule interaction)", grants(weaponCore, `"weaponProficiencies":[{"category":"Martial","requiresAnyProperty":["Heavy"]}],`+emptyA+`,`+emptyT), []string{m + "weaponProficiencies/0 constraintViolation"}},
		{"weapon unfiltered beyond a filter", classSpec + " FR-026", "unhappy (rule interaction)", grants(weaponCore, `"weaponProficiencies":[{"category":"Martial"}],`+emptyA+`,`+emptyT), []string{m + "weaponProficiencies/0 constraintViolation"}},
		{"weapon category beyond the core", classSpec + " FR-026", "unhappy (rule interaction)", grants(weaponCore, `"weaponProficiencies":[{"category":"Simple"}],`+emptyA+`,`+emptyT), []string{m + "weaponProficiencies/0 constraintViolation"}},
		{"weapon of an unfiltered core", classSpec + " FR-026", "happy", grants(core(`[{"category":"Simple"}]`, `[{"category":"Martial"}]`), `"weaponProficiencies":[{"category":"Martial","requiresAnyProperty":["Light"]}],`+emptyA+`,`+emptyT), nil},
		{"tool choice within the core", classSpec + " FR-026", "happy", grants(toolCore, emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Instrument","Kit"],"picks":2},{"kind":"specific","name":"Kit"}]`), nil},
		{"weapon of another category on an unfiltered core", classSpec + " FR-026", "unhappy (rule interaction)", grants(wizardClassJSON(), `"weaponProficiencies":[{"category":"Martial"}],`+emptyA+`,`+emptyT), []string{m + "weaponProficiencies/0 constraintViolation"}},
		{"simple weapons on an unfiltered martial core", classSpec + " FR-026", "unhappy (rule interaction)", grants(core(`[{"category":"Simple"}]`, `[{"category":"Martial"}]`), `"weaponProficiencies":[{"category":"Simple"}],`+emptyA+`,`+emptyT), []string{m + "weaponProficiencies/0 constraintViolation"}},
		{"tool choice with fewer picks than the core", classSpec + " FR-026", "unhappy (boundary)", grants(toolCore, emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Instrument"],"picks":1}]`), nil},
		{"tool choice of one on a specific-only core", classSpec + " FR-026", "happy", grants(core(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"specific","name":"Kit"}]`), emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Kit"],"picks":1}]`), nil},
		{"tool picks from the largest core choice, not the last", classSpec + " FR-026", "happy", grants(core(`"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"choice","categories":["Instrument"],"picks":2},{"kind":"choice","categories":["Kit"],"picks":1}]`), emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Instrument"],"picks":2}]`), nil},
		{"tool choice with too many picks", classSpec + " FR-026", "unhappy (rule interaction)", grants(toolCore, emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Instrument"],"picks":3}]`), []string{m + "toolProficiencies/0 constraintViolation"}},
		{"tool category beyond the core", classSpec + " FR-026", "unhappy (rule interaction)", grants(toolCore, emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Gaming"],"picks":1}]`), []string{m + "toolProficiencies/0 constraintViolation"}},
		{"specific tool beyond the core", classSpec + " FR-026", "unhappy (rule interaction)", grants(toolCore, emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"specific","name":"Other"}]`), []string{m + "toolProficiencies/0 constraintViolation"}},
		{"tool grant on a class with no tools", classSpec + " FR-026", "unhappy (rule interaction)", grants(wizardClassJSON(), emptyW+`,`+emptyA+`,"toolProficiencies":[{"kind":"choice","categories":["Instrument"],"picks":1}]`), []string{m + "toolProficiencies/0 constraintViolation"}},
		{"skills within the list", classSpec + " FR-026", "happy", grants(wizardClassJSON(), emptyW+`,`+emptyA+`,`+emptyT+`,"skillChoice":{"picks":1,"options":{"kind":"listed","skills":["Arcana"]}}`), nil},
		{"skill outside the list", classSpec + " FR-026", "unhappy (rule interaction)", grants(wizardClassJSON(), emptyW+`,`+emptyA+`,`+emptyT+`,"skillChoice":{"picks":1,"options":{"kind":"listed","skills":["Stealth"]}}`), []string{m + "skillChoice constraintViolation"}},
		{"skill picks beyond the core", classSpec + " FR-026", "unhappy (rule interaction)", grants(wizardClassJSON(), emptyW+`,`+emptyA+`,`+emptyT+`,"skillChoice":{"picks":3,"options":{"kind":"listed","skills":["Arcana","History","Insight"]}}`), []string{m + "skillChoice constraintViolation"}},
		{"any skill on a listed core", classSpec + " FR-026", "unhappy (rule interaction)", grants(wizardClassJSON(), emptyW+`,`+emptyA+`,`+emptyT+`,"skillChoice":{"picks":1,"options":{"kind":"any"}}`), []string{m + "skillChoice constraintViolation"}},
		{"any skill on an any core", classSpec + " FR-026", "happy", grants(core(`{"kind":"listed","skills":["Arcana","History","Insight"]}`, `{"kind":"any"}`), emptyW+`,`+emptyA+`,`+emptyT+`,"skillChoice":{"picks":1,"options":{"kind":"any"}}`), nil},
		{"listed skills on an any core", classSpec + " FR-026", "happy", grants(core(`{"kind":"listed","skills":["Arcana","History","Insight"]}`, `{"kind":"any"}`), emptyW+`,`+emptyA+`,`+emptyT+`,"skillChoice":{"picks":1,"options":{"kind":"listed","skills":["Stealth"]}}`), nil},
		{"grants missing a part", classSpec + " FR-026", "unhappy (missing content)", grants(wizardClassJSON(), emptyW+`,`+emptyA), []string{m + "toolProficiencies missingField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeClassSpellcastingRows(t *testing.T) {
	t.Parallel()
	w := wizardClassJSON()
	s := func(old, repl string) string { return classDoc(mutate(w, old, repl)) }
	pact := pactClassJSON()
	pc := func(old, repl string) string { return classDoc(mutate(pact, old, repl)) }
	sp := cp + "progression/0/spellcasting"
	runDocCases(t, []docCase{
		{"nine slots", classSpec + " FR-031", "happy", classDoc(w), nil},
		{"eight slots", classSpec + " FR-031", "unhappy (boundary)", s(`[1,0,0,0,0,0,0,0,0]`, `[1,0,0,0,0,0,0,0]`), []string{sp + "/spellSlots outOfRange"}},
		{"ten slots", classSpec + " FR-031", "unhappy (boundary)", s(`[1,0,0,0,0,0,0,0,0]`, `[1,0,0,0,0,0,0,0,0,0]`), []string{sp + "/spellSlots outOfRange"}},
		{"negative slot", classSpec + " FR-031", "unhappy (boundary)", s(`[1,0,0,0,0,0,0,0,0]`, `[-1,0,0,0,0,0,0,0,0]`), []string{sp + "/spellSlots/0 outOfRange"}},
		{"cantrips missing", classSpec + " FR-031", "unhappy (missing content)", s(`"cantrips":3,"preparedSpells":1`, `"preparedSpells":1`), []string{sp + "/cantrips missingField"}},
		{"cantrips on a class without the flag", classSpec + " FR-031", "unhappy (rule interaction)", s(`"cantrips":true`, `"cantrips":false`), allRows("spellcasting/cantrips constraintViolation")},
		{"pact row on a slot class", classSpec + " FR-030", "unhappy (rule interaction)", s(`{"cantrips":3,"preparedSpells":1,"spellSlots":[1,0,0,0,0,0,0,0,0]}`, `{"cantrips":3,"preparedSpells":1,"spellSlots":1,"slotLevel":1}`), []string{sp + " constraintViolation"}},
		{"slot row on a pact class", classSpec + " FR-030", "unhappy (rule interaction)", pc(`{"cantrips":2,"preparedSpells":1,"spellSlots":1,"slotLevel":1}`, `{"preparedSpells":1,"spellSlots":[1,0,0,0,0,0,0,0,0]}`), []string{sp + " constraintViolation"}},
		{"row part on a non-caster", classSpec + " FR-030", "unhappy (rule interaction)", classDoc(strings.Replace(w, `"spellcasting":`+slotTrait+`,`, ``, 1)), allRows("spellcasting constraintViolation")},
		{"caster row without the part", classSpec + " FR-030", "unhappy (missing content)", s(`,"spellcasting":{"cantrips":3,"preparedSpells":1,"spellSlots":[1,0,0,0,0,0,0,0,0]}`, ``), []string{sp + " missingField"}},
		{"pact class", classSpec + " FR-032", "happy", classDoc(pact), nil},
		{"slot level 0", classSpec + " FR-032", "unhappy (boundary)", pc(`"slotLevel":1}`, `"slotLevel":0}`), []string{sp + "/slotLevel outOfRange"}},
		{"slot level 10", classSpec + " FR-032", "unhappy (boundary)", pc(`"slotLevel":1}`, `"slotLevel":10}`), []string{sp + "/slotLevel outOfRange"}},
		{"slot level 9", classSpec + " FR-032", "unhappy (boundary)", pc(`"slotLevel":1}`, `"slotLevel":9}`), nil},
		{"pact slots zero", classSpec + " FR-032", "unhappy (boundary)", pc(`"spellSlots":1,"slotLevel":1`, `"spellSlots":0,"slotLevel":1`), []string{sp + "/spellSlots outOfRange"}},
		{"pact row missing a field", classSpec + " FR-032", "unhappy (missing content)", pc(`"cantrips":2,"preparedSpells":1,`, `"cantrips":2,`), []string{sp + "/preparedSpells missingField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeClassColumnsAndRows(t *testing.T) {
	t.Parallel()
	w := wizardClassJSON()
	cc := columnClassJSON()
	c := func(old, repl string) string { return classDoc(mutate(w, old, repl)) }
	col := func(old, repl string) string { return classDoc(mutate(cc, old, repl)) }
	rowsOf := func() string { return strings.Join(classRows("fixture", slotRowJSON, `{}`), ",") }
	rowsMinus := strings.Join(classRows("fixture", slotRowJSON, `{}`)[:19], ",")
	rowsPlus := rowsOf() + "," + classRows("fixture", slotRowJSON, `{}`)[19]
	v := cp + "progression/0/values/"
	runDocCases(t, []docCase{
		{"columns of every kind", classSpec + " FR-036", "happy", classDoc(cc), nil},
		{"duplicate column id", classSpec + " FR-036", "unhappy (boundary)", col(`{"id":"die"`, `{"id":"charges"`), []string{cp + "columns/1/id duplicate"}},
		{"blank header", classSpec + " FR-036", "unhappy (invalid input)", col(`"header":"Die"`, `"header":" "`), []string{cp + "columns/1/header invalidValue"}},
		{"unknown column kind", classSpec + " FR-036", "unhappy (invalid input)", col(`"kind":"dice"`, `"kind":"text"`), []string{cp + "columns/1/kind invalidValue"}},
		{"invalid column id", classSpec + " FR-036", "unhappy (invalid input)", col(`"id":"die"`, `"id":"Die"`), []string{cp + "columns/1/id invalidValue"}},
		{"count 0", classSpec + " FR-040", "unhappy (boundary)", col(`"charges":2`, `"charges":0`), nil},
		{"count -1", classSpec + " FR-040", "unhappy (boundary)", col(`"charges":2`, `"charges":-1`), []string{v + "charges outOfRange"}},
		{"count as a string", classSpec + " FR-040", "unhappy (invalid input)", col(`"charges":2`, `"charges":"2"`), []string{v + "charges invalidValue"}},
		{"count as an object", classSpec + " FR-040", "unhappy (invalid input)", col(`"charges":2`, `"charges":{"count":1,"sides":6}`), []string{v + "charges wrongType"}},
		{"count fraction", classSpec + " FR-040", "unhappy (invalid input)", col(`"charges":2`, `"charges":2.5`), []string{v + "charges wrongType"}},
		{"dice as an integer", classSpec + " FR-040", "unhappy (invalid input)", col(`"die":{"count":1,"sides":6}`, `"die":6`), []string{v + "die wrongType"}},
		{"dice count 0", classSpec + " FR-040", "unhappy (boundary)", col(`"count":1`, `"count":0`), []string{v + "die/count outOfRange"}},
		{"dice one side", classSpec + " FR-040", "unhappy (boundary)", col(`"sides":6`, `"sides":1`), []string{v + "die/sides outOfRange"}},
		{"dice two sides", classSpec + " FR-040", "unhappy (boundary)", col(`"sides":6`, `"sides":2`), nil},
		{"dice 1000 sides", classSpec + " FR-040", "unhappy (boundary)", col(`"sides":6`, `"sides":1000`), nil},
		{"dice 1001 sides", classSpec + " FR-040", "unhappy (boundary)", col(`"sides":6`, `"sides":1001`), []string{v + "die/sides outOfRange"}},
		{"dice missing sides", classSpec + " FR-040", "unhappy (missing content)", col(`"die":{"count":1,"sides":6}`, `"die":{"count":1}`), []string{v + "die/sides missingField"}},
		{"bonus 0", classSpec + " FR-040", "unhappy (boundary)", col(`"bonus":1`, `"bonus":0`), nil},
		{"bonus -1", classSpec + " FR-040", "unhappy (boundary)", col(`"bonus":1`, `"bonus":-1`), []string{v + "bonus outOfRange"}},
		{"feet bonus", classSpec + " FR-040", "happy", col(`"speed":10`, `"speed":0`), nil},
		{"feet bonus negative", classSpec + " FR-040", "unhappy (boundary)", col(`"speed":10`, `"speed":-5`), []string{v + "speed outOfRange"}},
		{"none for count", classSpec + " FR-040", "unhappy (empty case)", col(`"charges":2`, `"charges":"—"`), nil},
		{"none for dice", classSpec + " FR-040", "unhappy (empty case)", col(`"die":{"count":1,"sides":6}`, `"die":"—"`), nil},
		{"hyphen for none", classSpec + " FR-040", "unhappy (invalid input)", col(`"charges":2`, `"charges":"-"`), []string{v + "charges invalidValue"}},
		{"missing column value", classSpec + " FR-039", "unhappy (missing content)", col(`"charges":2,`, ``), []string{v + "charges missingField"}},
		{"undeclared column value", classSpec + " FR-039", "unhappy (invalid input)", col(`"charges":2,`, `"luck":3,"charges":2,`), []string{v + "luck unknownField"}},
		{"values is an array", classSpec + " FR-039", "unhappy (invalid input)", c(`"values":{}`, `"values":[]`), []string{cp + "progression/0/values wrongType"}},
		{"19 rows", classSpec + " FR-037", "unhappy (boundary)", classDoc(strings.Replace(w, rowsOf(), rowsMinus, 1)), []string{cp + "progression outOfRange"}},
		{"21 rows", classSpec + " FR-037", "unhappy (boundary)", classDoc(strings.Replace(w, rowsOf(), rowsPlus, 1)), []string{cp + "progression outOfRange"}},
		{"level out of order", classSpec + " FR-037", "unhappy (invalid input)", c(`{"level":4,`, `{"level":5,`), []string{cp + "progression/3/level invalidValue"}},
		{"row level 0", classSpec + " FR-037", "unhappy (boundary)", c(`{"level":4,`, `{"level":0,`), []string{cp + "progression/3/level outOfRange"}},
		{"row level 21", classSpec + " FR-037", "unhappy (boundary)", c(`{"level":4,`, `{"level":21,`), []string{cp + "progression/3/level outOfRange"}},
		{"bonus 0 is rejected", classSpec + " FR-038", "unhappy (boundary)", c(`"proficiencyBonus":2`, `"proficiencyBonus":0`), []string{cp + "progression/0/proficiencyBonus outOfRange"}},
		{"bonus 1 is accepted", classSpec + " FR-038", "unhappy (boundary)", c(`"proficiencyBonus":2`, `"proficiencyBonus":1`), nil},
		{"two subclass markers", classSpec + " FR-043", "unhappy (boundary)", c(`{"kind":"subclassFeature"}`, `{"kind":"subclassFeature"},{"kind":"subclassFeature"}`), []string{cp + "progression/5/features/1 duplicate"}},
		{"unknown row feature kind", classSpec + " FR-043", "unhappy (invalid input)", c(`{"kind":"subclassFeature"}`, `{"kind":"other"}`), []string{cp + "progression/5/features/0/kind invalidValue"}},
		{"high row feature kind", classSpec + " FR-043", "unhappy (boundary)", c(`{"kind":"subclassFeature"}`, `{"kind":"zzz"}`), []string{cp + "progression/5/features/0/kind invalidValue"}},
		{"wrong kind with feature fields", classSpec + " FR-043", "unhappy (invalid input)", c(`{"kind":"subclassFeature"}`, `{"kind":"other","featureId":"fixture_gamma","annotation":"x"}`), []string{cp + "progression/5/features/0/kind invalidValue"}},
		{"none followed by text", classSpec + " FR-040", "unhappy (boundary)", col(`"charges":2`, `"charges":"—x"`), []string{v + "charges invalidValue"}},
		{"row feature kind is a number", classSpec + " FR-043", "unhappy (invalid input)", c(`{"kind":"subclassFeature"}`, `{"kind":3}`), []string{cp + "progression/5/features/0/kind wrongType"}},
		{"empty cell", classSpec + " FR-043", "unhappy (empty case)", c(`"features":[{"featureId":"fixture_gamma"}]`, `"features":[]`), []string{cp + "features/2/level constraintViolation"}},
		{"annotation", classSpec + " FR-043", "happy", c(`{"featureId":"fixture_gamma"}`, `{"featureId":"fixture_gamma","annotation":"one use"}`), nil},
		{"blank annotation", classSpec + " FR-043", "unhappy (invalid input)", c(`{"featureId":"fixture_gamma"}`, `{"featureId":"fixture_gamma","annotation":""}`), []string{cp + "progression/1/features/0/annotation invalidValue"}},
		{"unresolved feature", classSpec + " FR-044", "unhappy (missing content)", c(`{"featureId":"fixture_gamma"}`, `{"featureId":"fixture_nothing"}`), []string{cp + "progression/1/features/0/featureId unresolvedReference", cp + "features/2/level constraintViolation"}},
		{"entry later than the row", classSpec + " FR-044", "unhappy (boundary)", c(`{"featureId":"fixture_gamma"}`, `{"featureId":"fixture_epsilon"}`), []string{cp + "progression/1/features/0/featureId constraintViolation", cp + "features/2/level constraintViolation"}},
		{"entry never listed", classSpec + " FR-045", "unhappy (rule interaction)", c(`{"featureId":"fixture_alpha"},`, ``), []string{cp + "features/0/level constraintViolation"}},
		{"entry listed at an older row only", classSpec + " FR-045", "unhappy (rule interaction)", c(`{"featureId":"fixture_theta"}`, `{"featureId":"fixture_alpha"}`), []string{cp + "features/9/level constraintViolation"}},
	})
}
