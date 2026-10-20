package content_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzJSONDecoder checks the properties of schema FR-018, FR-016 and FR-076:
// Decode never panics, gives the same result for the same bytes, returns
// either a nil error or a bounded list of well-formed problems that matches
// ErrInvalidInput with the zero Set, and leaves its input untouched. The seeds
// cover the number spellings, long literals and bulk arrays; each kind row
// adds its own valid and invalid entries as it lands.
//
// Requirement: doc/specs/content-schema-and-decoder.md FR-018, FR-076 and doc/rules/testing.md section 10
// Path: unhappy (invalid input).
func FuzzJSONDecoder(f *testing.F) {
	seeds := []string{
		``, `{}`, `null`, `[]`, `{"schemaVersion":1}`, `{"schemaVersion":2}`, `{"schemaVersion":"1"}`,
		`{"schemaVersion":1.0}`, `{"schemaVersion":1.5}`, `{"schemaVersion":1e0}`, `{"schemaVersion":10E-1}`, `{"schemaVersion":-0}`,
		`{"schemaVersion":1e999999999}`, `{"schemaVersion":1e-999999999}`, `{"schemaVersion":12345678901234567890}`,
		`{"schemaVersion":9007199254740991}`, `{"schemaVersion":9007199254740992}`, `{"schemaVersion":9007199254740993}`,
		`{"schemaVersion":1,"races":[]}`, `{"schemaVersion":1,"spells":{}}`, `{"schemaVersion":1,"spells":[1,null,[],{}]}`,
		`{"schemaVersion":1,"a/b":1,"a~b":2}`, `{"schemaVersion":1,"x":1,"x":2}`, `{"schemaVersion":1,"a\ud800":1}`,
		"{\"schemaVersion\":1,\"a\xff\":1}", "\xef\xbb\xbf{}", `{"schemaVersion":1.` + strings.Repeat("0", 4096) + `}`,
		`{"schemaVersion":1.` + strings.Repeat("0", 4096) + `1}`, `{"schemaVersion":0.` + strings.Repeat("0", 4096) + `1e4097}`,
		`{"schemaVersion":1,"x":` + strings.Repeat("[", 10000) + strings.Repeat("]", 10000) + `}`,
		strings.Repeat("[", 10001) + strings.Repeat("]", 10001),
		`{"schemaVersion":1,"x":[` + strings.Repeat("1,", 10_000_000/2-1) + `1]}`,                            // a flat array of about 10 MB
		`{"schemaVersion":1,"spells":[` + strings.Repeat(`{"slots":[0],"tags":["Acid"]},`, 100_000) + `{}]}`, // 100000 bounded arrays
		`{"schemaVersion":1,"x":[` + strings.Repeat("0,", 100_000) + `0]}`,                                   // a 100k element array
	}
	// One valid and some invalid entries of each registered kind row (step 5a).
	seeds = append(seeds,
		wrapDoc("species", speciesJSON(choiceJSON(damageOpt("A", "Acid")))), wrapDoc("species", speciesJSON(choiceJSON(lineageOpt("L", 1, 3, 5)))),
		wrapDoc("species", speciesJSON(choiceJSON(textOpt("A", "spell_a", "spell_b")))), wrapDoc("species", speciesJSON(choiceJSON(lineageOpt("L", 1, 5, 3)))),
		wrapDoc("species", speciesJSON(choiceJSON(damageOpt("A", "Acid"), textOpt("B")))), wrapDoc("species", speciesJSON(choiceJSON(`{"kind":1}`, `[]`, `null`))),
		wrapDoc("species", `{"id":"x","sizes":[null,[],{}],"speed":1e999999999,"traits":[[],{"name":1}],"choices":{}}`),
		wrapDoc("backgrounds", backgroundJSON), wrapDoc("backgrounds", mutate(backgroundJSON, `"specific"`, `"x"`)),
		wrapDoc("backgrounds", `{"feat":[],"equipmentOptions":{"A":[],"B":{"items":{},"gold":"1"}},"toolProficiency":{"kind":null}}`),
		wrapDoc("feats", featJSON), wrapDoc("feats", mutate(featJSON, `"benefits"`, `"prerequisites":[{"kind":"abilityScore","abilities":["strength","strength"],"minimum":99},{"kind":3}],"benefits"`)),
		wrapDoc("feats", mutate(featJSON, `false`, `true`)), wrapDoc("feats", `{"repeatable":null,"repeatableText":[],"benefits":[{"name":[]}]}`),
	)
	// One valid and some invalid classes (step 5b).
	seeds = append(seeds,
		classDoc(wizardClassJSON()), classDoc(pactClassJSON()), classDoc(plainClassJSON()), classDoc(columnClassJSON()),
		classDoc(ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6))), classDoc(mutate(wizardClassJSON(), `"D6"`, `"D4"`)),
		classDoc(mutate(wizardClassJSON(), `"cantrips":true`, `"cantrips":false`)), classDoc(mutate(pactClassJSON(), `"slotLevel":1}`, `"slotLevel":10}`)),
		classDoc(mutate(columnClassJSON(), `"charges":2`, `"charges":"-"`)), classDoc(mutate(wizardClassJSON(), alphaTail, withOptions(alphaTail, option("a", `,"prerequisite":{"requiresOptionIds":["a"]}`)))),
		classDoc(mutate(wizardClassJSON(), gammaTail, strings.TrimSuffix(gammaTail, "}")+`,"tables":[{"name":"T","columns":["A"],"rows":[[1],["x"],[true]]}]}`)),
		`{"schemaVersion":1,"classes":[{"id":"a","primaryAbility":[],"progression":{},"features":[null,[],{"facts":{}}],"subclasses":[{"features":3}],"columns":[{"kind":1}],"startingEquipment":{"D":1}}]}`,
	)
	// One valid and some invalid equipment documents (step 5c).
	seeds = append(seeds,
		allEquipment(), eqDoc("weapons", mutate(weaponJSON, `"1d6"`, `"d6+1"`)), eqDoc("weapons", mutate(weaponJSON, `"properties":[]`, `"properties":[{"property":"Thrown","range":{"normal":0,"long":0}},{"property":"Thrown"},null]`)),
		eqDoc("armor", mutate(armorJSON, `{"base":11,"dexterity":"full"}`, `{"base":99,"bonus":1,"dexterity":3,"maximum":"x"}`)),
		eqDoc("tools", mutate(familyJSON, `["variant_x"]`, `[null,[],{}]`), mutate(variantJSON, `"variantOf"`, `"ability":1,"craft":{},"variantOf"`)),
		eqDoc("adventuringGear", mutate(packJSON, `"quantity":2`, `"quantity":1e999999999`), mutate(magicJSON, `"days":1`, `"days":[]`), mutate(scrollVarJSON, `"spellLevel":0`, `"spellLevel":1e-9`)),
		eqDoc("airborneAndWaterborneVehicles", mutate(vehicleJSON, `1.5`, `1e999`), mutate(vehicleJSON, `0.5`, `0.`+strings.Repeat("0", 4096)+`5`)),
		eqDoc("spellcastingServices", svc("a", 0, 3), svc("b", 5, 9)), eqDoc(append([]string{"spellScrollCosts"}, scrollCosts(3, 3)...)...),
	)
	// One valid and some invalid magic item documents (step 5d).
	seeds = append(seeds,
		miDoc(miJSON), miDoc(mutate(miJSON, `{"kind":"single","rarity":"Uncommon"}`, `{"kind":"byVariant","options":[{"rarity":"Rare","variant":"+1"},{"rarity":"Rare","variant":"+1"}]}`)),
		miDoc(mutate(miJSON, `"category":"Rod",`, `"category":"Rod","categoryNote":[],"attunement":{"by":1},`)), miDoc(mutate(miJSON, `{"kind":"single","rarity":"Uncommon"}`, `{"kind":[],"options":3}`)),
		miDoc(mutate(miJSON, `{"kind":"paragraph","text":"Text."}`, `{"kind":"table","title":1,"header":[],"rows":[[],[null],"x",["a","b"]]},{"kind":"bulletList","items":[1,""]},{"kind":"embeddedStatBlock","lines":{}},{"kind":"entry"},{"kind":1},null`)),
		`{"schemaVersion":1,"magicItems":[`+strings.Repeat(`{"description":[{"kind":"table","header":["h"],"rows":[["x"]]}]},`, 100_000)+`{}]}`,
	)
	// One valid and some invalid spell documents (step 5g).
	seeds = append(seeds,
		spellDoc(spellJSON), spellWith(`"level":2`, `"level":10`), spellWith(`[{"kind":"magicAction"}]`, `[{"kind":"magicAction","label":"A"},{"kind":"hours","amount":1e999999999}]`),
		spellWith(`"classes":["wizard","fixture_class"]`, `"classes":[null,[],{},"A","A"]`), spellWith(`"components":{`, `"components":{"verbal":1,"material":[],`),
		spellWith(`"range":{"kind":"distance","amount":60,"unit":"feet"}`, `"range":{"kind":"distance","amount":0.5}`), spellWith(`"duration":{`, `"duration":{"kind":"timed","limit":1,`),
		spellFx("2", fxA), spellFx("0", fxB(fxZone)), spellFx("0", fxA), spellFx("2", fxB("")), spellFx("2", `{"zone":[],"parts":{}}`),
		`{"schemaVersion":1,"spells":[`+strings.Repeat(`{"classes":["a"],"description":[{"kind":"paragraph","text":"x"}]},`, 100_000)+`{}]}`,
	)
	// One valid and some invalid monster documents (step 5e).
	seeds = append(seeds,
		monDoc(monJSON), monWith(`"sizes":["Medium"]`, `"sizes":[null,[],{},"Medium","Medium"]`), monWith(`"faces":8`, `"faces":1e999999999`),
		monWith(`"speeds":[{"modes":["walk"],"feet":30}]`, `"speeds":[{"modes":[],"feet":-1,"hover":1},[],null,{"modes":["fly","fly"]}]`),
		monWith(`"languages":{}`, `"languages":{"phrases":{},"telepathyFeet":1.5,"telepathyQualifier":[]}`),
		monActions(monPartJSON("Multiattack", `,"plan":{"variants":[{"requiresUsed":"X","groups":[{"count":1e9,"parts":["A","A"]}],"replacements":[{"scope":{"kind":"one_attack_part"},"options":[{"spell":{}}]}]},[],null]}`)),
		monActions(monPartJSON("Bite", `,"usage":{"kind":"recharge","low":"x"},"clauses":[{"label":"success","successRider":{"kind":"immune_to_part","hours":-1},"damage":[null,{"dice":{}},{"damageType":"Fire","text":"x","underwaterResistanceDenied":[]}]}],"attack":{"range":{"normal":0,"long":-1}},"spellcasting":{"groups":[{"spells":[{"castAtLevel":99}]}]}`)),
		monWith(`"abilities":{`, `"abilities":{"x":null,"str":[],`), monWith(`"senses"`, `"skills":{"a":1,"a":2,"Stealth":"x"},"immunities":[{"condition":"Charmed"},{"damageType":"Fire"},{},{"damageType":"Fire","condition":"Charmed"}],"senses"`),
		`{"schemaVersion":1,"monsters":[`+strings.Repeat(`{"actions":[{"clauses":[{"damage":[{}]}]}]},`, 100_000)+`{}]}`,
	)
	for _, s := range seeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, doc []byte) {
		before := slices.Clone(doc)
		set1, err1 := content.JSONDecoder{}.Decode(doc)
		set2, err2 := content.JSONDecoder{}.Decode(doc)
		if !slices.Equal(doc, before) {
			t.Fatalf("Decode modified its input")
		}
		if (err1 == nil) != (err2 == nil) || !set1.Equal(set2) || !slices.Equal(content.Problems(err1), content.Problems(err2)) {
			t.Fatalf("Decode is not deterministic")
		}
		if err1 == nil {
			_, _ = content.ValidateSet(set1) // the verdict does not matter: it must never panic for a decoded set, whatever its references
			return
		}
		ps := content.Problems(err1)
		if len(ps) == 0 || len(ps) > content.MaxProblems+1 || !errors.Is(err1, srderr.ErrInvalidInput) || !set1.Equal(content.Set{}) {
			t.Fatalf("a rejection must be 1 to %d problems, ErrInvalidInput and the zero Set; got %d problems, %v", content.MaxProblems+1, len(ps), err1)
		}
		for _, p := range ps {
			if p.Kind() < content.ProblemSyntax || p.Kind() > content.ProblemConstraintViolation || !utf8.ValidString(p.Path()) || (p.Path() != "" && p.Path()[0] != '/') {
				t.Fatalf("malformed problem %q", p)
			}
			if got := p.Error(); len(got) > 4*content.MaxRenderedPath+64 || strings.ContainsAny(got, "\n\r") {
				t.Fatalf("Problem.Error() is not bounded and single-line: %q", got)
			}
		}
	})
}
