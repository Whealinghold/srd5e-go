package content_test

import (
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

const miJSON = `{"id":"rod_x","name":"Rod X","category":"Rod","rarity":{"kind":"single","rarity":"Uncommon"},"description":[{"kind":"paragraph","text":"Text."}]}`

func miDoc(entry ...string) string {
	return `{"schemaVersion":1,"magicItems":[` + strings.Join(entry, ",") + `]}`
}

// miWith is a document of the fixture entry with one replacement.
func miWith(old, repl string) string { return miDoc(mutate(miJSON, old, repl)) }

func miDecode(t *testing.T, doc string) content.MagicItemEntry {
	t.Helper()
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %q", decodeProblems(doc))
	}
	for _, e := range set.Entries() {
		if m, ok := e.(content.MagicItemEntry); ok {
			return m
		}
	}
	t.Fatal("no magic item in the set")
	return content.MagicItemEntry{}
}

// Requirement: miSpec FR-037, FR-040, FR-043 - the root array magicItems is
// registered (absent means none, a non-array is wrongType), the smallest
// entry decodes to a valid entry of kind magicItem, and decoding is
// deterministic.
// Path: unhappy (boundary).
func TestDecodeMagicItemRoot(t *testing.T) {
	t.Parallel()
	runDocCases(t, []docCase{
		{"absent array", miSpec + " FR-037", "happy", `{"schemaVersion":1}`, nil},
		{"empty array", "FR-037", "unhappy (boundary)", `{"schemaVersion":1,"magicItems":[]}`, nil},
		{"object", "FR-037", "unhappy (boundary)", `{"schemaVersion":1,"magicItems":{}}`, []string{"/magicItems wrongType"}},
		{"null", "FR-037", "unhappy (boundary)", `{"schemaVersion":1,"magicItems":null}`, []string{"/magicItems wrongType"}},
		{"element not an object", "FR-038", "unhappy (invalid input)", `{"schemaVersion":1,"magicItems":[1]}`, []string{"/magicItems/0 wrongType"}},
	})
	e := miDecode(t, miDoc(miJSON))
	if e.Key().Kind() != content.KindMagicItem || e.CategoryNote() != "" || e.Attunement() != (content.NoAttunement{}) || len(e.Description()) != 1 {
		t.Errorf("minimal entry = %+v [FR-040]", e)
	}
	if !e.Equal(miDecode(t, miDoc(miJSON))) {
		t.Error("two decodes differ [FR-043]")
	}
}

// Requirement: per case (miSpec "Cases: Decoder rows")
// Path: per case.
func TestDecodeMagicItemRows(t *testing.T) {
	t.Parallel()
	const p = "/magicItems/0/"
	two := `{"kind":"byVariant","options":[{"rarity":"Rare","variant":"+1"},{"rarity":"Very Rare","variant":"+2"}]}`
	byVariant := func(opts string) string {
		return miWith(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"byVariant","options":[`+opts+`]}`)
	}
	runDocCases(t, []docCase{
		{"category missing", miSpec + " FR-012", "unhappy (missing content)", miWith(`"category":"Rod",`, ``), []string{p + "category missingField"}},
		{"category plural", "FR-012", "unhappy (invalid input)", miWith(`"Rod"`, `"Wondrous Items"`), []string{p + "category invalidValue"}},
		{"category case", "FR-012", "unhappy (invalid input)", miWith(`"Rod"`, `"armor"`), []string{p + "category invalidValue"}},
		{"category not a string", "FR-012", "unhappy (invalid input)", miWith(`"Rod"`, `7`), []string{p + "category wrongType"}},
		{"category wondrous item", "FR-012", "happy", miWith(`"Rod"`, `"Wondrous Item"`), nil},
		{"note", "FR-013", "happy", miWith(`"category":"Rod",`, `"category":"Rod","categoryNote":"Any Medium or Heavy, Except Hide Armor",`), nil},
		{"note blank", "FR-013", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","categoryNote":" ",`), []string{p + "categoryNote invalidValue"}},
		{"note null", "FR-013", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","categoryNote":null,`), []string{p + "categoryNote wrongType"}},
		{"rarity missing", "FR-017", "unhappy (missing content)", miWith(`"rarity":{"kind":"single","rarity":"Uncommon"},`, ``), []string{p + "rarity missingField"}},
		{"rarity word", "FR-017", "unhappy (invalid input)", miWith(`"Uncommon"`, `"Very rare"`), []string{p + "rarity/rarity invalidValue"}},
		{"rarity word missing", "FR-017", "unhappy (missing content)", miWith(`,"rarity":"Uncommon"`, ``), []string{p + "rarity/rarity missingField"}},
		{"rarity kind", "FR-017", "unhappy (invalid input)", miWith(`"single"`, `"range"`), []string{p + "rarity/kind invalidValue"}},
		{"rarity kind missing", "FR-017", "unhappy (missing content)", miWith(`"kind":"single",`, ``), []string{p + "rarity/kind missingField"}},
		{"rarity kind not a string", "FR-017", "unhappy (invalid input)", miWith(`"single"`, `1`), []string{p + "rarity/kind wrongType"}},
		{"rarity as string", "FR-017, FR-042", "unhappy (rule interaction)", miWith(`{"kind":"single","rarity":"Uncommon"}`, `"Rare"`), []string{p + "rarity wrongType"}},
		{"varies", "FR-017", "happy", miWith(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"varies"}`), nil},
		{"varies with a rarity", "FR-017", "unhappy (invalid input)", miWith(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"varies","rarity":"Rare"}`), []string{p + "rarity/rarity unknownField"}},
		{"single with options", "FR-017", "unhappy (invalid input)", miWith(`"rarity":"Uncommon"`, `"rarity":"Uncommon","options":[]`), []string{p + "rarity/options unknownField"}},
		{"one option", "FR-017", "unhappy (boundary)", byVariant(`{"rarity":"Rare","variant":"+1"}`), []string{p + "rarity/options outOfRange"}},
		{"no options", "FR-017", "unhappy (boundary)", byVariant(``), []string{p + "rarity/options outOfRange"}},
		{"options missing", "FR-017", "unhappy (missing content)", miWith(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"byVariant"}`), []string{p + "rarity/options missingField"}},
		{"two options", "FR-016", "unhappy (boundary)", miWith(`{"kind":"single","rarity":"Uncommon"}`, two), nil},
		{"repeated variant", "FR-017", "unhappy (invalid input)", byVariant(`{"rarity":"Rare","variant":"+1"},{"rarity":"Rare","variant":"+1"}`), []string{p + "rarity/options/1/variant duplicate"}},
		{"blank variant", "FR-017", "unhappy (invalid input)", byVariant(`{"rarity":"Rare","variant":""},{"rarity":"Rare","variant":"+1"}`), []string{p + "rarity/options/0/variant invalidValue"}},
		{"bad option rarity", "FR-017", "unhappy (invalid input)", byVariant(`{"rarity":"Mythic","variant":"+1"},{"rarity":"Rare","variant":"+2"}`), []string{p + "rarity/options/0/rarity invalidValue"}},
		{"option not an object", "FR-017", "unhappy (invalid input)", byVariant(`1,{"rarity":"Rare","variant":"+2"}`), []string{p + "rarity/options/0 wrongType"}},
		{"option with an unknown key", "FR-017", "unhappy (invalid input)", byVariant(`{"rarity":"Rare","variant":"+1","x":1},{"rarity":"Rare","variant":"+2"}`), []string{p + "rarity/options/0/x unknownField"}},
		{"attunement required", "FR-019", "happy", miWith(`"category":"Rod",`, `"category":"Rod","attunement":{},`), nil},
		{"attunement by", "FR-019", "happy", miWith(`"category":"Rod",`, `"category":"Rod","attunement":{"by":"a Wizard"},`), nil},
		{"attunement blank", "FR-019", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","attunement":{"by":" "},`), []string{p + "attunement/by invalidValue"}},
		{"attunement boolean", "FR-019", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","attunement":true,`), []string{p + "attunement wrongType"}},
		{"attunement null", "FR-019", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","attunement":null,`), []string{p + "attunement wrongType"}},
		{"attunement other key", "FR-019", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","attunement":{"class":"x"},`), []string{p + "attunement/class unknownField"}},
		{"description empty", "FR-041", "unhappy (boundary)", miWith(`[{"kind":"paragraph","text":"Text."}]`, `[]`), []string{p + "description outOfRange"}},
		{"description missing", "FR-041", "unhappy (missing content)", miWith(`,"description":[{"kind":"paragraph","text":"Text."}]`, ``), []string{p + "description missingField"}},
		{"description not an array", "FR-041", "unhappy (invalid input)", miWith(`[{"kind":"paragraph","text":"Text."}]`, `"x"`), []string{p + "description wrongType"}},
		{"unknown key", "FR-038", "unhappy (invalid input)", miWith(`"category":"Rod",`, `"category":"Rod","charges":3,`), []string{p + "charges unknownField"}},
		{"name blank", "FR-004", "unhappy (invalid input)", miWith(`"Rod X"`, `" "`), []string{p + "name invalidValue"}},
		{"id with an uppercase letter", "FR-002", "unhappy (invalid input)", miWith(`"rod_x"`, `"Rod_x"`), []string{p + "id invalidValue"}},
		{"id of 65 characters", "FR-006", "unhappy (boundary)", miWith(`"rod_x"`, `"a`+strings.Repeat("b", 64)+`"`), []string{p + "id invalidValue"}},
		{"source", "FR-010", "happy", miWith(`"category":"Rod",`, `"source":{"document":"SRD 5.2.1","section":"Magic Items A–Z","page":209},"category":"Rod",`), nil},
		{"duplicate id", "FR-007", "unhappy (invalid input)", miDoc(miJSON, miJSON), []string{"/magicItems/1/id duplicate"}},
		{"same id, two kinds", "FR-007", "unhappy (rule interaction)", `{"schemaVersion":1,"weapons":[` + mutate(weaponJSON, `"club_x"`, `"rod_x"`) + `],"magicItems":[` + miJSON + `]}`, nil},
		{"three problems in field order", "FR-042", "unhappy (invalid input)",
			miDoc(`{"id":"rod_x","name":"Rod X","rarity":{"kind":"range"},"description":[]}`),
			[]string{p + "category missingField", p + "rarity/kind invalidValue", p + "description outOfRange"}},
		{"problems by ascending index, duplicates last", "FR-042, FR-007", "unhappy (rule interaction)",
			miDoc(mutate(miJSON, `"Rod"`, `"x"`), miJSON, mutate(miJSON, `"rod_x"`, `"rod_y"`), mutate(miJSON, `"Rod"`, `"y"`)),
			[]string{"/magicItems/0/category invalidValue", "/magicItems/3/category invalidValue", "/magicItems/1/id duplicate", "/magicItems/3/id duplicate"}},
	})
}

// Requirement: miSpec FR-012 to FR-019, FR-063 (header forms), FR-022 - the
// decoder builds the typed values of the header forms and the blocks.
// Path: per case.
func TestDecodeMagicItemValues(t *testing.T) {
	t.Parallel()
	entry := func(mods ...string) content.MagicItemEntry {
		d := miJSON
		for i := 0; i+1 < len(mods); i += 2 {
			d = mutate(d, mods[i], mods[i+1])
		}
		return miDecode(t, miDoc(d))
	}
	if r, ok := entry().Rarity().(content.SingleRarity); !ok || r.Rarity() != content.RarityUncommon {
		t.Errorf("single rarity = %#v", entry().Rarity())
	}
	if _, ok := entry(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"varies"}`).Rarity().(content.RarityVaries); !ok {
		t.Error("varies did not decode to RarityVaries")
	}
	by := entry(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"byVariant","options":[{"rarity":"Rare","variant":"Silver or Brass"},{"rarity":"Very Rare","variant":"Bronze"}]}`)
	rv, ok := by.Rarity().(content.RarityByVariant)
	if !ok {
		t.Fatalf("rarity = %T", by.Rarity())
	}
	got := rv.Options()
	if len(got) != 2 || got[0].Variant() != "Silver or Brass" || got[0].Rarity() != content.RarityRare || got[1].Rarity() != content.RarityVeryRare {
		t.Errorf("options = %v", got)
	}
	if a := entry(`"category":"Rod",`, `"category":"Rod","attunement":{},`).Attunement(); a != (content.AttunementRequired{}) {
		t.Errorf("attunement = %#v", a)
	}
	by2, _ := content.NewAttunementRequiredBy("a Cleric, Druid, or Paladin")
	if a := entry(`"category":"Rod",`, `"category":"Rod","attunement":{"by":"a Cleric, Druid, or Paladin"},`).Attunement(); a != content.Attunement(by2) {
		t.Errorf("attunement = %#v", a)
	}
	if n := entry(`"category":"Rod",`, `"category":"Rod","categoryNote":"Battleaxe, Greataxe, or Halberd",`).CategoryNote(); n != "Battleaxe, Greataxe, or Halberd" {
		t.Errorf("note = %q", n)
	}
	src := entry(`"category":"Rod",`, `"source":{"document":"SRD 5.2.1","section":"S","page":209},"category":"Rod",`).Source()
	if src != content.MustSourceCitation("SRD 5.2.1", "S", 209) {
		t.Errorf("source = %v", src)
	}
}

// Requirement: miSpec FR-044 (schema FR-094) - the decoder and the
// constructors give the same verdict on the category and rarity words, the
// option counts and the variants.
// Path: per case.
func TestDecodeMagicItemAgreesWithConstructors(t *testing.T) {
	t.Parallel()
	for _, w := range []string{"Armor", "Wondrous Item", "Wondrous Items", "armor", "", " Rod", "Gadget"} {
		_, perr := content.ParseMagicItemCategory(w)
		if got := len(decodeProblems(miWith(`"Rod"`, jsonString(w)))) == 0; got != (perr == nil) {
			t.Errorf("category %q: decoder accepts %v, ParseMagicItemCategory accepts %v [FR-044]", w, got, perr == nil)
		}
	}
	for _, w := range []string{"Common", "Very Rare", "Very rare", "Mythic", "", "Artifact"} {
		_, perr := content.ParseItemRarity(w)
		if got := len(decodeProblems(miWith(`"Uncommon"`, jsonString(w)))) == 0; got != (perr == nil) {
			t.Errorf("rarity %q: decoder accepts %v, ParseItemRarity accepts %v [FR-044]", w, got, perr == nil)
		}
	}
	for _, variants := range [][]string{{"+1"}, {"+1", "+2"}, {"+1", "+1"}, {"+1", " "}, {"", "+1"}, {"a", "b", "c"}} {
		wire := ""
		var opts []content.RarityOption
		for i, v := range variants {
			if i > 0 {
				wire += ","
			}
			wire += `{"rarity":"Rare","variant":` + jsonString(v) + `}`
			opts = append(opts, content.RarityOption{})
			if o, err := content.NewRarityOption(content.RarityRare, v); err == nil {
				opts[i] = o
			}
		}
		_, cerr := content.NewRarityByVariant(opts)
		doc := miWith(`{"kind":"single","rarity":"Uncommon"}`, `{"kind":"byVariant","options":[`+wire+`]}`)
		if got := len(decodeProblems(doc)) == 0; got != (cerr == nil) {
			t.Errorf("variants %q: decoder accepts %v, constructor accepts %v [FR-044]", variants, got, cerr == nil)
		}
	}
	for _, by := range []string{"a Wizard", " ", "", "a\tb"} {
		_, cerr := content.NewAttunementRequiredBy(by)
		if got := len(decodeProblems(miWith(`"category":"Rod",`, `"category":"Rod","attunement":{"by":`+jsonString(by)+`},`))) == 0; got != (cerr == nil) {
			t.Errorf("attunement by %q: decoder accepts %v, constructor accepts %v [FR-044]", by, got, cerr == nil)
		}
	}
}

// Requirement: miSpec FR-041 (schema FR-150 to FR-152) - the five block kinds
// decode to their variants in order; a table title is optional and a cell may
// be empty.
// Path: happy.
func TestDecodeTextBlocksHappy(t *testing.T) {
	t.Parallel()
	blocks := `[{"kind":"paragraph","text":"P."},{"kind":"entry","name":"Splash","text":"E."},{"kind":"bulletList","items":["a","b"]},` +
		`{"kind":"table","title":"T","header":["h1","h2"],"rows":[["1",""],["2","—"]]},{"kind":"table","header":["h"],"rows":[["x"]]},` +
		`{"kind":"embeddedStatBlock","lines":["Giant X","AC 12"]}]`
	e := miDecode(t, miWith(`[{"kind":"paragraph","text":"Text."}]`, blocks))
	want := []content.TextBlock{
		content.MustTextParagraph("P."), content.MustTextEntry("Splash", "E."), content.MustTextBulletList([]string{"a", "b"}),
		content.MustTitledTextTable("T", []string{"h1", "h2"}, [][]string{{"1", ""}, {"2", "—"}}),
		content.MustTextTable([]string{"h"}, [][]string{{"x"}}),
		content.MustTextEmbeddedStatBlock([]string{"Giant X", "AC 12"}),
	}
	if got := e.Description(); !textBlocksEqual(got, want) {
		t.Errorf("blocks = %#v, want %#v", got, want)
	}
}

// Requirement: miSpec FR-041 (schema FR-152) - the problems of a block, at the
// block's path.
// Path: per case.
func TestDecodeTextBlocksProblems(t *testing.T) {
	t.Parallel()
	const p = "/magicItems/0/description/0"
	blk := func(b string) string { return miWith(`{"kind":"paragraph","text":"Text."}`, b) }
	runDocCases(t, []docCase{
		{"not an object", "FR-152", "unhappy (invalid input)", blk(`"x"`), []string{p + " wrongType"}},
		{"missing kind", "FR-152", "unhappy (missing content)", blk(`{"text":"x"}`), []string{p + "/kind missingField"}},
		{"kind not a string", "FR-152", "unhappy (invalid input)", blk(`{"kind":1,"text":"x"}`), []string{p + "/kind wrongType"}},
		{"unknown kind, other keys not examined", "FR-152", "unhappy (invalid input)", blk(`{"kind":"quote","text":"x","who":1}`), []string{p + "/kind invalidValue"}},
		{"kind case", "FR-152", "unhappy (invalid input)", blk(`{"kind":"Paragraph","text":"x"}`), []string{p + "/kind invalidValue"}},
		{"paragraph unknown key", "FR-152", "unhappy (invalid input)", blk(`{"kind":"paragraph","text":"x","y":1}`), []string{p + "/y unknownField"}},
		{"paragraph text missing", "FR-152", "unhappy (missing content)", blk(`{"kind":"paragraph"}`), []string{p + "/text missingField"}},
		{"paragraph blank", "FR-152", "unhappy (invalid input)", blk(`{"kind":"paragraph","text":" "}`), []string{p + "/text invalidValue"}},
		{"paragraph text not a string", "FR-152", "unhappy (invalid input)", blk(`{"kind":"paragraph","text":3}`), []string{p + "/text wrongType"}},
		{"entry name with a space", "FR-152", "unhappy (invalid input)", blk(`{"kind":"entry","name":" a","text":"x"}`), []string{p + "/name invalidValue"}},
		{"entry blank text", "FR-152", "unhappy (invalid input)", blk(`{"kind":"entry","name":"a","text":""}`), []string{p + "/text invalidValue"}},
		{"entry name missing", "FR-152", "unhappy (missing content)", blk(`{"kind":"entry","text":"x"}`), []string{p + "/name missingField"}},
		{"entry text missing", "FR-152", "unhappy (missing content)", blk(`{"kind":"entry","name":"a"}`), []string{p + "/text missingField"}},
		{"entry both bad", "FR-152", "unhappy (invalid input)", blk(`{"kind":"entry","name":"","text":""}`), []string{p + "/name invalidValue", p + "/text invalidValue"}},
		{"list empty", "FR-152", "unhappy (boundary)", blk(`{"kind":"bulletList","items":[]}`), []string{p + "/items outOfRange"}},
		{"list missing", "FR-152", "unhappy (missing content)", blk(`{"kind":"bulletList"}`), []string{p + "/items missingField"}},
		{"list blank item", "FR-152", "unhappy (invalid input)", blk(`{"kind":"bulletList","items":["a"," "]}`), []string{p + "/items/1 invalidValue"}},
		{"list non-string item", "FR-152", "unhappy (invalid input)", blk(`{"kind":"bulletList","items":[1]}`), []string{p + "/items/0 wrongType"}},
		{"stat block lines missing", "FR-152", "unhappy (missing content)", blk(`{"kind":"embeddedStatBlock"}`), []string{p + "/lines missingField"}},
		{"stat block empty", "FR-152", "unhappy (boundary)", blk(`{"kind":"embeddedStatBlock","lines":[]}`), []string{p + "/lines outOfRange"}},
		{"stat block blank line", "FR-152", "unhappy (invalid input)", blk(`{"kind":"embeddedStatBlock","lines":[""]}`), []string{p + "/lines/0 invalidValue"}},
		{"table header empty", "FR-152", "unhappy (boundary)", blk(`{"kind":"table","header":[],"rows":[["x"]]}`), []string{p + "/header outOfRange"}},
		{"table rows empty", "FR-152", "unhappy (boundary)", blk(`{"kind":"table","header":["h"],"rows":[]}`), []string{p + "/rows outOfRange"}},
		{"table header blank", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":[" "],"rows":[["x"]]}`), []string{p + "/header/0 invalidValue"}},
		{"table title blank", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","title":" ","header":["h"],"rows":[["x"]]}`), []string{p + "/title invalidValue"}},
		{"table ragged row", "FR-152", "unhappy (boundary)", blk(`{"kind":"table","header":["h","i"],"rows":[["x","y"],["x"]]}`), []string{p + "/rows/1 constraintViolation"}},
		{"table wide row", "FR-152", "unhappy (boundary)", blk(`{"kind":"table","header":["h"],"rows":[["x","y"]]}`), []string{p + "/rows/0 constraintViolation"}},
		{"table blank cell", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":["h"],"rows":[[" "]]}`), []string{p + "/rows/0/0 invalidValue"}},
		{"table non-string cell", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":["h"],"rows":[[1]]}`), []string{p + "/rows/0/0 wrongType"}},
		{"table row not an array", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":["h"],"rows":["x"]}`), []string{p + "/rows/0 wrongType"}},
		{"table row null", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":["h"],"rows":[null]}`), []string{p + "/rows/0 wrongType"}},
		{"table row object", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":["h"],"rows":[{}]}`), []string{p + "/rows/0 wrongType"}},
		{"table empty row", "FR-152", "unhappy (boundary)", blk(`{"kind":"table","header":["h"],"rows":[[]]}`), []string{p + "/rows/0 constraintViolation"}},
		{"table bad row hides the width check", "FR-152", "unhappy (rule interaction)", blk(`{"kind":"table","header":["h"],"rows":[[1,"y"]]}`), []string{p + "/rows/0/0 wrongType"}},
		{"table bad header hides the width check", "FR-152", "unhappy (rule interaction)", blk(`{"kind":"table","header":[""],"rows":[["x","y"]]}`), []string{p + "/header/0 invalidValue"}},
		{"table unknown key", "FR-152", "unhappy (invalid input)", blk(`{"kind":"table","header":["h"],"rows":[["x"]],"cols":1}`), []string{p + "/cols unknownField"}},
		{"second block bad", "FR-152", "unhappy (invalid input)", miWith(`[{"kind":"paragraph","text":"Text."}]`, `[{"kind":"paragraph","text":"a"},{"kind":"paragraph","text":""}]`), []string{"/magicItems/0/description/1/text invalidValue"}},
	})
}
