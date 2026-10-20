package content_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func with(array, base, old, repl string) string { return eqDoc(array, mutate(base, old, repl)) }

// Requirement: per case
// Path: per case.
func TestDecodeArmor(t *testing.T) {
	t.Parallel()
	const p = "/armor/0/"
	a := func(old, repl string) string { return with("armor", armorJSON, old, repl) }
	s := func(old, repl string) string { return with("armor", shieldJSON, old, repl) }
	const base14 = `"armorClass":{"base":14,"dexterity":"capped","maximum":2}`
	runDocCases(t, []docCase{
		{"valid body armor", equipSpec + " FR-105", "happy", eqDoc("armor", armorJSON), nil},
		{"valid Shield", "FR-024, FR-105", "happy", eqDoc("armor", shieldJSON), nil},
		{"strength requirement 30", "FR-109", "unhappy (boundary)", a(`"stealthDisadvantage"`, `"strengthRequirement":30,"stealthDisadvantage"`), nil},
		{"capped Dexterity", "FR-112", "happy", a(`"armorClass":{"base":11,"dexterity":"full"}`, base14), nil},
		{"no Dexterity", "FR-112", "happy", a(`"full"`, `"none"`), nil},
		{"missing category", "FR-109", "unhappy (missing content)", a(`"category":"Light",`, ``), []string{p + "category missingField"}},
		{"missing armorClass", "FR-109", "unhappy (missing content)", a(`"armorClass":{"base":11,"dexterity":"full"},`, ``), []string{p + "armorClass missingField"}},
		{"missing stealthDisadvantage", "FR-109", "unhappy (missing content)", a(`"stealthDisadvantage":false,`, ``), []string{p + "stealthDisadvantage missingField"}},
		{"missing weightPounds", "FR-109", "unhappy (missing content)", a(`"weightPounds":10,`, ``), []string{p + "weightPounds missingField"}},
		{"missing cost", "FR-109, FR-080", "unhappy (missing content)", a(`,"cost":{"amount":10,"coin":"GP"}`, ``), []string{p + "cost missingField"}},
		{"category Heavy Armor", "FR-109", "unhappy (invalid input)", a(`"Light"`, `"Heavy Armor"`), []string{p + "category invalidValue"}},
		{"strength as text", "FR-109", "unhappy (invalid input)", a(`"stealthDisadvantage"`, `"strengthRequirement":"13","stealthDisadvantage"`), []string{p + "strengthRequirement wrongType"}},
		{"strength 0", "FR-109", "unhappy (boundary)", a(`"stealthDisadvantage"`, `"strengthRequirement":0,"stealthDisadvantage"`), []string{p + "strengthRequirement outOfRange"}},
		{"strength 31", "FR-109", "unhappy (boundary)", a(`"stealthDisadvantage"`, `"strengthRequirement":31,"stealthDisadvantage"`), []string{p + "strengthRequirement outOfRange"}},
		{"stealth as text", "FR-109", "unhappy (invalid input)", a(`false`, `"yes"`), []string{p + "stealthDisadvantage wrongType"}},
		{"weight below 0", "FR-080", "unhappy (boundary)", a(`10,`, `-1,`), []string{p + "weightPounds outOfRange"}},
		{"armorClass empty", "FR-112", "unhappy (missing content)", a(`{"base":11,"dexterity":"full"}`, `{}`), []string{p + "armorClass missingField"}},
		{"armorClass with base and bonus", "FR-112", "unhappy (rule interaction)", a(`"dexterity":"full"}`, `"dexterity":"full","bonus":2}`), []string{p + "armorClass constraintViolation"}},
		{"capped without maximum", "FR-112", "unhappy (missing content)", a(`"full"`, `"capped"`), []string{p + "armorClass/maximum missingField"}},
		{"full with maximum", "FR-112", "unhappy (rule interaction)", a(`"full"`, `"full","maximum":2`), []string{p + "armorClass/maximum constraintViolation"}},
		{"none with maximum", "FR-112", "unhappy (rule interaction)", a(`"full"`, `"none","maximum":2`), []string{p + "armorClass/maximum constraintViolation"}},
		{"capped maximum -1", "FR-112", "unhappy (boundary)", a(`"full"`, `"capped","maximum":-1`), []string{p + "armorClass/maximum outOfRange"}},
		{"capped maximum 0", "FR-112", "unhappy (boundary)", a(`"full"`, `"capped","maximum":0`), nil},
		{"dexterity other", "FR-112", "unhappy (invalid input)", a(`"full"`, `"half"`), []string{p + "armorClass/dexterity invalidValue"}},
		{"dexterity of the wrong type", "FR-112", "unhappy (invalid input)", a(`"full"`, `1`), []string{p + "armorClass/dexterity wrongType"}},
		{"base without dexterity", "FR-112", "unhappy (missing content)", a(`,"dexterity":"full"`, ``), []string{p + "armorClass/dexterity missingField"}},
		{"base 0", "FR-112", "unhappy (boundary)", a(`"base":11`, `"base":0`), []string{p + "armorClass/base outOfRange"}},
		{"base 30", "FR-112", "unhappy (boundary)", a(`"base":11`, `"base":30`), nil},
		{"base 31", "FR-112", "unhappy (boundary)", a(`"base":11`, `"base":31`), []string{p + "armorClass/base outOfRange"}},
		{"bonus with dexterity", "FR-112", "unhappy (rule interaction)", s(`"bonus":2`, `"bonus":2,"dexterity":"full"`), []string{p + "armorClass/dexterity constraintViolation"}},
		{"bonus with maximum and bad bonus", "FR-112", "unhappy (rule interaction)", s(`"bonus":2`, `"bonus":0,"maximum":2`), []string{p + "armorClass/bonus outOfRange", p + "armorClass/maximum constraintViolation"}},
		{"bonus 0", "FR-112", "unhappy (boundary)", s(`"bonus":2`, `"bonus":0`), []string{p + "armorClass/bonus outOfRange"}},
		{"bonus 10", "FR-112", "unhappy (boundary)", s(`"bonus":2`, `"bonus":10`), nil},
		{"bonus 11", "FR-112", "unhappy (boundary)", s(`"bonus":2`, `"bonus":11`), []string{p + "armorClass/bonus outOfRange"}},
		{"Light with a bonus", "FR-024, FR-112", "unhappy (rule interaction)", a(`{"base":11,"dexterity":"full"}`, `{"bonus":2}`), []string{p + "armorClass constraintViolation"}},
		{"Shield with a base", "FR-024, FR-112", "unhappy (rule interaction)", s(`{"bonus":2}`, `{"base":11,"dexterity":"full"}`), []string{p + "armorClass constraintViolation"}},
		{"armorClass of the wrong type", "FR-112", "unhappy (invalid input)", a(`{"base":11,"dexterity":"full"}`, `11`), []string{p + "armorClass wrongType"}},
		{"unknown armorClass key", "FR-112", "unhappy (invalid input)", a(`"dexterity":"full"`, `"dexterity":"full","cap":1`), []string{p + "armorClass/cap unknownField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeTool(t *testing.T) {
	t.Parallel()
	const p = "/tools/0/"
	tl := func(old, repl string) string { return with("tools", toolJSON, old, repl) }
	fam := func(old, repl string) string { return with("tools", familyJSON, old, repl) }
	vr := func(old, repl string) string { return eqDoc("tools", familyJSON, mutate(variantJSON, old, repl)) }
	runDocCases(t, []docCase{
		{"valid standalone", equipSpec + " FR-029", "happy", eqDoc("tools", toolJSON), nil},
		{"family and variant", "FR-029, FR-030", "happy", eqDoc("tools", familyJSON, variantJSON), nil},
		{"family with a fixed cost", "FR-014", "happy", fam(`"costVaries":true`, `"cost":{"amount":1,"coin":"GP"}`), nil},
		{"empty craft list", "FR-033", "happy", tl(`"craft":[{"kind":"item","id":"club_x"}]`, `"craft":[]`), nil},
		{"no craft field", "FR-033", "happy", tl(`"craft":[{"kind":"item","id":"club_x"}],`, ``), nil},
		{"no weight", "FR-013", "happy", tl(`"weightPounds":5,`, ``), nil},
		{"dc 1", "FR-029", "unhappy (boundary)", tl(`"dc":10`, `"dc":1`), nil},
		{"missing utilize", "FR-029", "unhappy (missing content)", tl(`"utilize":[{"description":"Do a thing","dc":10}],`, ``), []string{p + "utilize missingField"}},
		{"empty utilize", "FR-029", "unhappy (boundary)", tl(`[{"description":"Do a thing","dc":10}]`, `[]`), []string{p + "utilize outOfRange"}},
		{"utilize of the wrong type", "FR-029", "unhappy (invalid input)", tl(`[{"description":"Do a thing","dc":10}]`, `{}`), []string{p + "utilize wrongType"}},
		{"dc 0", "FR-029", "unhappy (boundary)", tl(`"dc":10`, `"dc":0`), []string{p + "utilize/0/dc outOfRange"}},
		{"utilize element of the wrong type", "FR-029", "unhappy (invalid input)", tl(`[{"description":"Do a thing","dc":10}]`, `[3]`), []string{p + "utilize/0 wrongType"}},
		{"dc as text", "FR-029", "unhappy (invalid input)", tl(`"dc":10`, `"dc":"10"`), []string{p + "utilize/0/dc wrongType"}},
		{"dc fractional", "FR-029", "unhappy (invalid input)", tl(`"dc":10`, `"dc":10.5`), []string{p + "utilize/0/dc wrongType"}},
		{"blank description", "FR-029", "unhappy (invalid input)", tl(`"Do a thing"`, `" "`), []string{p + "utilize/0/description invalidValue"}},
		{"missing description", "FR-029", "unhappy (missing content)", tl(`"description":"Do a thing",`, ``), []string{p + "utilize/0/description missingField"}},
		{"missing dc", "FR-029", "unhappy (missing content)", tl(`,"dc":10`, ``), []string{p + "utilize/0/dc missingField"}},
		{"missing ability", "FR-029", "unhappy (missing content)", tl(`"ability":"dexterity",`, ``), []string{p + "ability missingField"}},
		{"ability spelled with a capital", "FR-029", "unhappy (invalid input)", tl(`"dexterity"`, `"Dexterity"`), []string{p + "ability invalidValue"}},
		{"missing toolGroup", "FR-113", "unhappy (missing content)", tl(`"toolGroup":"Artisan's Tools",`, ``), []string{p + "toolGroup missingField"}},
		{"toolGroup Artisan Tools", "FR-113", "unhappy (invalid input)", tl(`"Artisan's Tools"`, `"Artisan Tools"`), []string{p + "toolGroup invalidValue"}},
		{"toolGroup of the wrong type", "FR-113", "unhappy (invalid input)", tl(`"Artisan's Tools"`, `3`), []string{p + "toolGroup wrongType"}},
		{"craft kind other", "FR-031", "unhappy (invalid input)", tl(`"kind":"item"`, `"kind":"all"`), []string{p + "craft/0/kind invalidValue"}},
		{"craft kind missing", "FR-031", "unhappy (missing content)", tl(`"kind":"item",`, ``), []string{p + "craft/0/kind missingField"}},
		{"craft set Shields", "FR-031", "unhappy (invalid input)", tl(`{"kind":"item","id":"club_x"}`, `{"kind":"set","set":"Shields"}`), []string{p + "craft/0/set invalidValue"}},
		{"craft set with except", "FR-031", "happy", tl(`{"kind":"item","id":"club_x"}`, `{"kind":"set","set":"MeleeWeapons","except":["club_x"]}`), nil},
		{"craft set without set", "FR-031", "unhappy (missing content)", tl(`{"kind":"item","id":"club_x"}`, `{"kind":"set"}`), []string{p + "craft/0/set missingField"}},
		{"craft item without id", "FR-031", "unhappy (missing content)", tl(`,"id":"club_x"`, ``), []string{p + "craft/0/id missingField"}},
		{"craft item with an invalid id", "FR-031", "unhappy (invalid input)", tl(`"club_x"}]`, `"Bad Id"}]`), []string{p + "craft/0/id invalidValue"}},
		{"except on an item", "FR-031", "unhappy (rule interaction)", tl(`"id":"club_x"`, `"id":"club_x","except":[]`), []string{p + "craft/0/except unknownField"}},
		{"except of the wrong type", "FR-031", "unhappy (invalid input)", tl(`{"kind":"item","id":"club_x"}`, `{"kind":"set","set":"HeavyArmor","except":"a"}`), []string{p + "craft/0/except wrongType"}},
		{"except repeats an id", "FR-031", "unhappy (invalid input)", tl(`{"kind":"item","id":"club_x"}`, `{"kind":"set","set":"HeavyArmor","except":["a","a"]}`), []string{p + "craft/0/except/1 duplicate"}},
		{"craft of the wrong type", "FR-031", "unhappy (invalid input)", tl(`[{"kind":"item","id":"club_x"}]`, `{}`), []string{p + "craft wrongType"}},
		{"craft element of the wrong type", "FR-031", "unhappy (invalid input)", tl(`[{"kind":"item","id":"club_x"}]`, `[3]`), []string{p + "craft/0 wrongType"}},
		{"costVaries on a standalone tool", "FR-014", "unhappy (rule interaction)", tl(`"cost":{"amount":5,"coin":"GP"}`, `"costVaries":true`), []string{p + "costVaries constraintViolation"}},
		{"family with cost and costVaries", "FR-013", "unhappy (rule interaction)", fam(`"costVaries":true`, `"costVaries":true,"cost":{"amount":1,"coin":"GP"}`), []string{p + "costVaries constraintViolation"}},
		{"family with costVaries false", "FR-013", "unhappy (invalid input)", fam(`"costVaries":true`, `"costVaries":false`), []string{p + "costVaries invalidValue"}},
		{"family with costVaries of the wrong type", "FR-013", "unhappy (invalid input)", fam(`"costVaries":true`, `"costVaries":"yes"`), []string{p + "costVaries wrongType"}},
		{"family with neither price", "FR-013", "unhappy (missing content)", fam(`"costVaries":true,`, ``), []string{p + "cost missingField"}},
		{"family with empty variants", "FR-030", "unhappy (boundary)", fam(`["variant_x"]`, `[]`), []string{p + "variants outOfRange"}},
		{"family with a repeated variant", "FR-030", "unhappy (invalid input)", fam(`["variant_x"]`, `["variant_x","variant_x"]`), []string{p + "variants/1 duplicate"}},
		{"family with a variant id that is not an id", "FR-030", "unhappy (invalid input)", fam(`["variant_x"]`, `["Bad Id"]`), []string{p + "variants/0 invalidValue"}},
		{"variant without cost", "FR-030", "unhappy (missing content)", vr(`,"cost":{"amount":1,"coin":"GP"}`, ``), []string{"/tools/1/cost missingField"}},
		{"variant with costVaries", "FR-030, FR-014", "unhappy (rule interaction)", vr(`"cost":{"amount":1,"coin":"GP"}`, `"costVaries":true`), []string{"/tools/1/costVaries constraintViolation"}},
		{"variant with ability", "FR-029", "unhappy (rule interaction)", vr(`"variantOf"`, `"ability":"dexterity","variantOf"`), []string{"/tools/1/ability constraintViolation"}},
		{"variant with utilize", "FR-029", "unhappy (rule interaction)", vr(`"variantOf"`, `"utilize":[],"variantOf"`), []string{"/tools/1/utilize constraintViolation"}},
		{"variant with craft", "FR-029", "unhappy (rule interaction)", vr(`"variantOf"`, `"craft":[],"variantOf"`), []string{"/tools/1/craft constraintViolation"}},
		{"variant with toolGroup", "FR-113", "unhappy (rule interaction)", vr(`"variantOf"`, `"toolGroup":"Other Tools","variantOf"`), []string{"/tools/1/toolGroup constraintViolation"}},
		{"variant with variants", "FR-030", "unhappy (rule interaction)", vr(`"variantOf"`, `"variants":["a"],"variantOf"`), []string{"/tools/1/variants constraintViolation"}},
		{"variant with an invalid variantOf", "FR-030", "unhappy (invalid input)", vr(`"family_x"`, `"Bad Id"`), []string{"/tools/1/variantOf invalidValue"}},
		{"weightNote without weightPounds", "FR-115", "unhappy (rule interaction)", tl(`"weightPounds":5,`, `"weightNote":"full",`), []string{p + "weightNote constraintViolation"}},
		{"blank weightNote", "FR-115", "unhappy (invalid input)", tl(`"weightPounds":5,`, `"weightPounds":5,"weightNote":" ",`), []string{p + "weightNote invalidValue"}},
		{"weightNote of the wrong type", "FR-115", "unhappy (invalid input)", tl(`"weightPounds":5,`, `"weightPounds":5,"weightNote":3,`), []string{p + "weightNote wrongType"}},
		{"weightNote with weightPounds", "FR-115", "happy", tl(`"weightPounds":5,`, `"weightPounds":5,"weightNote":"full",`), nil},
		{"weightVaries false", "FR-013", "unhappy (invalid input)", tl(`"weightPounds":5,`, `"weightVaries":false,`), []string{p + "weightVaries invalidValue"}},
		{"weightVaries of the wrong type", "FR-013", "unhappy (invalid input)", tl(`"weightPounds":5,`, `"weightVaries":1,`), []string{p + "weightVaries wrongType"}},
		{"weightPounds with weightVaries", "FR-013", "unhappy (rule interaction)", tl(`"weightPounds":5,`, `"weightPounds":5,"weightVaries":true,`), []string{p + "weightVaries constraintViolation"}},
		{"note is a gear field only", "FR-115", "unhappy (invalid input)", tl(`"toolGroup"`, `"note":"x","toolGroup"`), []string{p + "note unknownField"}},
		{"text is not a tool field", "FR-116", "unhappy (invalid input)", tl(`"toolGroup"`, `"text":"x","toolGroup"`), []string{p + "text unknownField"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeGear(t *testing.T) {
	t.Parallel()
	const p = "/adventuringGear/0/"
	g := func(old, repl string) string { return with("adventuringGear", plainJSON, old, repl) }
	m := func(old, repl string) string { return with("adventuringGear", magicJSON, old, repl) }
	f := func(old, repl string) string { return with("adventuringGear", gearFamilyJSON, old, repl) }
	am := func(old, repl string) string { return with("adventuringGear", arrowsJSON, old, repl) }
	fc := func(old, repl string) string { return with("adventuringGear", focusJSON, old, repl) }
	sf := func(old, repl string) string { return with("adventuringGear", scrollFamJSON, old, repl) }
	sv := func(old, repl string) string { return with("adventuringGear", scrollVarJSON, old, repl) }
	pk := func(old, repl string) string { return with("adventuringGear", packJSON, old, repl) }
	runDocCases(t, []docCase{
		{"plain", equipSpec + " FR-039", "happy", eqDoc("adventuringGear", plainJSON), nil},
		{"magic with an override", "FR-039, FR-094", "happy", eqDoc("adventuringGear", magicJSON), nil},
		{"magic without an override", "FR-094", "happy", m(`,"craftingOverride":{"rawMaterials":{"amount":25,"coin":"GP"},"days":1,"toolIds":["tool_x"]}`, ``), nil},
		{"family", "FR-039", "happy", eqDoc("adventuringGear", gearFamilyJSON), nil},
		{"ammunition variant", "FR-041", "happy", eqDoc("adventuringGear", arrowsJSON), nil},
		{"focus variant", "FR-040", "happy", eqDoc("adventuringGear", focusJSON), nil},
		{"focus variant without alsoCountsAs", "FR-040", "happy", fc(`"alsoCountsAs":"club_x",`, ``), nil},
		{"scroll family and variant", "FR-081", "happy", eqDoc("adventuringGear", scrollFamJSON, scrollVarJSON), nil},
		{"pack", "FR-042", "happy", eqDoc("adventuringGear", packJSON), nil},
		{"weight with a note", "FR-115, FR-035", "happy", g(`"weightPounds":1`, `"weightPounds":5,"weightNote":"full"`), nil},
		{"no weight", "FR-035", "happy", g(`"weightPounds":1,`, ``), nil},
		{"weight 0.25", "FR-015", "unhappy (boundary)", g(`"weightPounds":1`, `"weightPounds":0.25`), nil},
		{"thousands separator is not JSON", "FR-035", "unhappy (invalid input)", g(`"amount":2`, `"amount":1000`), nil},
		// gearKind.
		{"missing gearKind", "FR-039", "unhappy (missing content)", g(`"gearKind":"plain",`, ``), []string{p + "gearKind missingField"}},
		{"gearKind pouch", "FR-039", "unhappy (invalid input)", g(`"plain"`, `"pouch"`), []string{p + "gearKind invalidValue"}},
		{"gearKind of the wrong type", "FR-039", "unhappy (invalid input)", g(`"plain"`, `1`), []string{p + "gearKind wrongType"}},
		{"gearKind pouch hides the fields", "FR-039", "unhappy (invalid input)", g(`"text":"Text."`, `"text":"Text.","contents":[]`), []string{p + "contents constraintViolation"}},
		// text and note.
		{"plain without text", "FR-037", "unhappy (missing content)", g(`,"text":"Text."`, ``), []string{p + "text missingField"}},
		{"plain with blank text", "FR-037", "unhappy (invalid input)", g(`"Text."`, `" "`), []string{p + "text invalidValue"}},
		{"variant without text", "FR-037", "happy", fc(`"weightPounds"`, `"weightPounds"`), nil},
		{"variant with blank text", "FR-037", "unhappy (invalid input)", fc(`"weightPounds"`, `"text":" ","weightPounds"`), []string{p + "text invalidValue"}},
		{"blank note", "FR-115", "unhappy (invalid input)", g(`"text"`, `"note":"","text"`), []string{p + "note invalidValue"}},
		{"note of the wrong type", "FR-115", "unhappy (invalid input)", g(`"text"`, `"note":3,"text"`), []string{p + "note wrongType"}},
		{"weightNote without weightPounds", "FR-115", "unhappy (rule interaction)", g(`"weightPounds":1,`, `"weightNote":"full",`), []string{p + "weightNote constraintViolation"}},
		{"blank weightNote", "FR-115", "unhappy (invalid input)", g(`"weightPounds":1,`, `"weightPounds":1,"weightNote":" ",`), []string{p + "weightNote invalidValue"}},
		{"utilize is not a gear field", "FR-119", "unhappy (invalid input)", g(`"text"`, `"utilize":[],"text"`), []string{p + "utilize unknownField"}},
		// price.
		{"plain without a price", "FR-013", "unhappy (missing content)", g(`"cost":{"amount":2,"coin":"SP"},`, ``), []string{p + "cost missingField"}},
		{"plain with costVaries", "FR-014", "unhappy (rule interaction)", g(`"cost":{"amount":2,"coin":"SP"}`, `"costVaries":true`), []string{p + "costVaries constraintViolation"}},
		{"family with cost and costVaries", "FR-013", "unhappy (rule interaction)", f(`"costVaries":true`, `"costVaries":true,"cost":{"amount":1,"coin":"GP"}`), []string{p + "costVaries constraintViolation"}},
		{"family with a fixed cost", "FR-014", "happy", f(`"costVaries":true`, `"cost":{"amount":1,"coin":"GP"}`), nil},
		{"cost per Hour", "FR-012", "unhappy (invalid input)", g(`"coin":"SP"`, `"coin":"SP","per":"Hour"`), []string{p + "cost/per invalidValue"}},
		{"cost of the wrong type", "FR-080", "unhappy (invalid input)", g(`{"amount":2,"coin":"SP"}`, `2`), []string{p + "cost wrongType"}},
		{"amount as text", "FR-080", "unhappy (invalid input)", g(`"amount":2`, `"amount":"5"`), []string{p + "cost/amount wrongType"}},
		{"amount fractional", "FR-080", "unhappy (invalid input)", g(`"amount":2`, `"amount":2.5`), []string{p + "cost/amount wrongType"}},
		{"amount -1", "FR-080", "unhappy (boundary)", g(`"amount":2`, `"amount":-1`), []string{p + "cost/amount outOfRange"}},
		{"amount 0", "FR-080", "unhappy (boundary)", g(`"amount":2`, `"amount":0`), nil},
		{"amount 2147483647", "FR-080", "unhappy (boundary)", g(`"amount":2`, `"amount":2147483647`), nil},
		{"amount 2147483648", "FR-080", "unhappy (boundary)", g(`"amount":2`, `"amount":2147483648`), []string{p + "cost/amount outOfRange"}},
		{"cost without coin", "FR-080", "unhappy (missing content)", g(`,"coin":"SP"`, ``), []string{p + "cost/coin missingField"}},
		{"cost without amount", "FR-080", "unhappy (missing content)", g(`"amount":2,`, ``), []string{p + "cost/amount missingField"}},
		{"coin GC", "FR-080", "unhappy (invalid input)", g(`"SP"`, `"GC"`), []string{p + "cost/coin invalidValue"}},
		{"coin lower case", "FR-080", "unhappy (invalid input)", g(`"SP"`, `"sp"`), []string{p + "cost/coin invalidValue"}},
		{"unknown cost key", "FR-080", "unhappy (invalid input)", g(`"coin":"SP"`, `"coin":"SP","unit":1`), []string{p + "cost/unit unknownField"}},
		// fields a kind lacks.
		{"contents on plain", "FR-039", "unhappy (rule interaction)", g(`"text":"Text."`, `"text":"Text.","contents":[]`), []string{p + "contents constraintViolation"}},
		{"variants on a focus variant", "FR-039", "unhappy (rule interaction)", fc(`"weightPounds"`, `"variants":["a"],"weightPounds"`), []string{p + "variants constraintViolation"}},
		{"variantOf on a family", "FR-039", "unhappy (rule interaction)", f(`"text"`, `"variantOf":"a","text"`), []string{p + "variantOf constraintViolation"}},
		{"alsoCountsAs on plain", "FR-040", "unhappy (rule interaction)", g(`"text"`, `"alsoCountsAs":"club_x","text"`), []string{p + "alsoCountsAs constraintViolation"}},
		{"ammunition on a focus variant", "FR-039", "unhappy (rule interaction)", fc(`"weightPounds"`, `"ammunition":{},"weightPounds"`), []string{p + "ammunition constraintViolation"}},
		{"spellLevel on plain", "FR-114", "unhappy (rule interaction)", g(`"text"`, `"spellLevel":0,"text"`), []string{p + "spellLevel constraintViolation"}},
		{"craftingOverride on plain", "FR-094", "unhappy (rule interaction)", g(`"text"`, `"craftingOverride":{},"text"`), []string{p + "craftingOverride constraintViolation"}},
		// families.
		{"family with empty variants", "FR-039", "unhappy (boundary)", f(`["arrows_x"]`, `[]`), []string{p + "variants outOfRange"}},
		{"family without variants", "FR-039", "unhappy (missing content)", f(`,"variants":["arrows_x"]`, ``), []string{p + "variants missingField"}},
		{"family with a repeated variant", "FR-039", "unhappy (invalid input)", f(`["arrows_x"]`, `["a","a"]`), []string{p + "variants/1 duplicate"}},
		{"variant without variantOf", "FR-039", "unhappy (missing content)", am(`"variantOf":"ammo_x",`, ``), []string{p + "variantOf missingField"}},
		{"variantOf that is not an id", "FR-039", "unhappy (invalid input)", am(`"ammo_x"`, `"Bad Id"`), []string{p + "variantOf invalidValue"}},
		{"alsoCountsAs that is not an id", "FR-040", "unhappy (invalid input)", fc(`"club_x"`, `"Bad Id"`), []string{p + "alsoCountsAs invalidValue"}},
		// ammunition.
		{"ammunition missing", "FR-041", "unhappy (missing content)", am(`,"ammunition":{"amount":20,"storage":"Quiver"}`, ``), []string{p + "ammunition missingField"}},
		{"ammunition amount 0", "FR-041", "unhappy (boundary)", am(`"amount":20`, `"amount":0`), []string{p + "ammunition/amount outOfRange"}},
		{"ammunition amount 1", "FR-041", "unhappy (boundary)", am(`"amount":20`, `"amount":1`), nil},
		{"ammunition amount as text", "FR-041", "unhappy (invalid input)", am(`"amount":20`, `"amount":"20"`), []string{p + "ammunition/amount wrongType"}},
		{"ammunition storage empty", "FR-041", "unhappy (invalid input)", am(`"Quiver"`, `""`), []string{p + "ammunition/storage invalidValue"}},
		{"ammunition storage missing", "FR-041", "unhappy (missing content)", am(`,"storage":"Quiver"`, ``), []string{p + "ammunition/storage missingField"}},
		{"ammunition of the wrong type", "FR-041", "unhappy (invalid input)", am(`{"amount":20,"storage":"Quiver"}`, `20`), []string{p + "ammunition wrongType"}},
		// packs.
		{"contents missing", "FR-042", "unhappy (missing content)", pk(`,"contents":[{"id":"gear_x","quantity":2,"unit":"days"},{"id":"club_x","quantity":1}]`, ``), []string{p + "contents missingField"}},
		{"contents empty", "FR-042", "unhappy (boundary)", pk(`[{"id":"gear_x","quantity":2,"unit":"days"},{"id":"club_x","quantity":1}]`, `[]`), []string{p + "contents outOfRange"}},
		{"quantity 0", "FR-042", "unhappy (boundary)", pk(`"quantity":2`, `"quantity":0`), []string{p + "contents/0/quantity outOfRange"}},
		{"quantity 1", "FR-042", "unhappy (boundary)", pk(`"quantity":2`, `"quantity":1`), nil},
		{"blank unit", "FR-042", "unhappy (invalid input)", pk(`"days"`, `" "`), []string{p + "contents/0/unit invalidValue"}},
		{"content id that is not an id", "FR-042", "unhappy (invalid input)", pk(`"gear_x"`, `"Bad Id"`), []string{p + "contents/0/id invalidValue"}},
		{"content without id", "FR-042", "unhappy (missing content)", pk(`"id":"gear_x",`, ``), []string{p + "contents/0/id missingField"}},
		{"content without quantity", "FR-042", "unhappy (missing content)", pk(`"quantity":2,`, ``), []string{p + "contents/0/quantity missingField"}},
		{"content that is not an object", "FR-042", "unhappy (invalid input)", pk(`{"id":"club_x","quantity":1}`, `"club_x"`), []string{p + "contents/1 wrongType"}},
		// scrolls.
		{"scrollVariant without spellLevel", "FR-114", "unhappy (missing content)", sv(`,"spellLevel":0`, ``), []string{p + "spellLevel missingField"}},
		{"scrollVariant spellLevel 9", "FR-114", "unhappy (boundary)", sv(`"spellLevel":0`, `"spellLevel":9`), nil},
		{"scrollVariant spellLevel 10", "FR-114", "unhappy (boundary)", sv(`"spellLevel":0`, `"spellLevel":10`), []string{p + "spellLevel outOfRange"}},
		{"scrollVariant spellLevel -1", "FR-114", "unhappy (boundary)", sv(`"spellLevel":0`, `"spellLevel":-1`), []string{p + "spellLevel outOfRange"}},
		{"scrollVariant spellLevel fractional", "FR-114", "unhappy (invalid input)", sv(`"spellLevel":0`, `"spellLevel":0.5`), []string{p + "spellLevel wrongType"}},
		{"scrollVariant with a weight", "FR-081", "unhappy (rule interaction)", sv(`"spellLevel"`, `"weightPounds":1,"spellLevel"`), []string{p + "weightPounds constraintViolation"}},
		{"scrollVariant with Varies weight", "FR-081", "unhappy (rule interaction)", sv(`"spellLevel"`, `"weightVaries":true,"spellLevel"`), []string{p + "weightVaries constraintViolation"}},
		{"scrollFamily with a fixed cost", "FR-081", "unhappy (rule interaction)", sf(`"costVaries":true`, `"cost":{"amount":1,"coin":"GP"}`), []string{p + "cost constraintViolation"}},
		{"scrollFamily with a weight", "FR-081", "unhappy (rule interaction)", sf(`"text"`, `"weightPounds":1,"text"`), []string{p + "weightPounds constraintViolation"}},
		// crafting override.
		{"override days 0", "FR-117", "unhappy (boundary)", m(`"days":1`, `"days":0`), []string{p + "craftingOverride/days outOfRange"}},
		{"override without rawMaterials", "FR-117", "unhappy (missing content)", m(`"rawMaterials":{"amount":25,"coin":"GP"},`, ``), []string{p + "craftingOverride/rawMaterials missingField"}},
		{"override without days", "FR-117", "unhappy (missing content)", m(`"days":1,`, ``), []string{p + "craftingOverride/days missingField"}},
		{"override without toolIds", "FR-117", "unhappy (missing content)", m(`,"toolIds":["tool_x"]`, ``), []string{p + "craftingOverride/toolIds missingField"}},
		{"override with empty toolIds", "FR-117", "unhappy (empty case)", m(`["tool_x"]`, `[]`), nil},
		{"override rawMaterials bad coin", "FR-117", "unhappy (invalid input)", m(`"coin":"GP"},"days"`, `"coin":"GC"},"days"`), []string{p + "craftingOverride/rawMaterials/coin invalidValue"}},
		{"override of the wrong type", "FR-117", "unhappy (invalid input)", m(`{"rawMaterials":{"amount":25,"coin":"GP"},"days":1,"toolIds":["tool_x"]}`, `1`), []string{p + "craftingOverride wrongType"}},
		{"override tool id repeated", "FR-117", "unhappy (invalid input)", m(`["tool_x"]`, `["a","a"]`), []string{p + "craftingOverride/toolIds/1 duplicate"}},
	})
}

// Requirement: per case
// Path: per case.
func TestDecodeTablesOfPrices(t *testing.T) {
	t.Parallel()
	mt := func(old, repl string) string { return with("mounts", mountJSON, old, repl) }
	tk := func(old, repl string) string { return with("tackHarnessAndDrawnVehicles", tackJSON, old, repl) }
	vh := func(old, repl string) string { return with("airborneAndWaterborneVehicles", vehicleJSON, old, repl) }
	ls := func(old, repl string) string { return with("lifestyleExpenses", lifestyleJSON, old, repl) }
	fd := func(old, repl string) string { return with("foodDrinkAndLodging", foodJSON, old, repl) }
	hr := func(old, repl string) string { return with("hirelings", hirelingJSON, old, repl) }
	runDocCases(t, []docCase{
		{"mount", equipSpec + " FR-046", "happy", eqDoc("mounts", mountJSON), nil},
		{"mount capacity 0", "FR-091", "unhappy (boundary)", mt(`480`, `0`), []string{"/mounts/0/carryingCapacityPounds outOfRange"}},
		{"mount capacity 1", "FR-091", "unhappy (boundary)", mt(`480`, `1`), nil},
		{"mount capacity as text", "FR-091", "unhappy (invalid input)", mt(`480`, `"480"`), []string{"/mounts/0/carryingCapacityPounds wrongType"}},
		{"mount without capacity", "FR-091", "unhappy (missing content)", mt(`"carryingCapacityPounds":480,`, ``), []string{"/mounts/0/carryingCapacityPounds missingField"}},
		{"mount cost amount -1", "FR-080", "unhappy (boundary)", mt(`"amount":50`, `"amount":-1`), []string{"/mounts/0/cost/amount outOfRange"}},
		{"mount without cost", "FR-080", "unhappy (missing content)", mt(`,"cost":{"amount":50,"coin":"GP"}`, ``), []string{"/mounts/0/cost missingField"}},
		{"note on a mount", "FR-115", "unhappy (invalid input)", mt(`"carryingCapacityPounds"`, `"note":"x","carryingCapacityPounds"`), []string{"/mounts/0/note unknownField"}},
		{"weight on a mount", "FR-046", "unhappy (invalid input)", mt(`"carryingCapacityPounds"`, `"weightPounds":1,"carryingCapacityPounds"`), []string{"/mounts/0/weightPounds unknownField"}},
		{"tack", "FR-047", "happy", eqDoc("tackHarnessAndDrawnVehicles", tackJSON), nil},
		{"tack without weight", "FR-047", "unhappy (empty case)", tk(`"weightPounds":25,`, ``), nil},
		{"tack with a per Day rate", "FR-012", "happy", tk(`"coin":"GP"`, `"coin":"GP","per":"Day"`), nil},
		{"tack weight -1", "FR-080", "unhappy (boundary)", tk(`25`, `-1`), []string{"/tackHarnessAndDrawnVehicles/0/weightPounds outOfRange"}},
		{"vehicle", "FR-048", "happy", eqDoc("airborneAndWaterborneVehicles", vehicleJSON), nil},
		{"vehicle with no optional column", "FR-048", "unhappy (boundary)", vh(`"passengers":3,"cargoTons":0.5,`, ``), nil},
		{"vehicle without damageThreshold", "FR-048", "unhappy (boundary)", vh(`"damageThreshold":5,`, ``), nil},
		{"vehicle weight -1", "FR-080", "unhappy (boundary)", vh(`"weightPounds":100`, `"weightPounds":-1`), []string{"/airborneAndWaterborneVehicles/0/weightPounds outOfRange"}},
		{"speed 1.25", "FR-091", "unhappy (boundary)", vh(`1.5`, `1.25`), []string{"/airborneAndWaterborneVehicles/0/speedMph outOfRange"}},
		{"speed 0.5", "FR-091", "unhappy (boundary)", vh(`1.5`, `0.5`), nil},
		{"speed 0", "FR-091", "unhappy (boundary)", vh(`1.5`, `0`), []string{"/airborneAndWaterborneVehicles/0/speedMph outOfRange"}},
		{"speed of 3 written 3.0", "FR-091", "unhappy (boundary)", vh(`1.5`, `3.0`), nil},
		{"speed as text", "FR-091", "unhappy (invalid input)", vh(`1.5`, `"fast"`), []string{"/airborneAndWaterborneVehicles/0/speedMph wrongType"}},
		{"missing speed", "FR-091", "unhappy (missing content)", vh(`"speedMph":1.5,`, ``), []string{"/airborneAndWaterborneVehicles/0/speedMph missingField"}},
		{"cargo 0.25", "FR-091", "unhappy (boundary)", vh(`0.5`, `0.25`), []string{"/airborneAndWaterborneVehicles/0/cargoTons outOfRange"}},
		{"cargo as text", "FR-091", "unhappy (invalid input)", vh(`0.5`, `"half"`), []string{"/airborneAndWaterborneVehicles/0/cargoTons wrongType"}},
		{"crew as text", "FR-091", "unhappy (invalid input)", vh(`"crew":1`, `"crew":"10"`), []string{"/airborneAndWaterborneVehicles/0/crew wrongType"}},
		{"crew 0", "FR-091", "unhappy (boundary)", vh(`"crew":1`, `"crew":0`), []string{"/airborneAndWaterborneVehicles/0/crew outOfRange"}},
		{"passengers 0", "FR-091", "unhappy (boundary)", vh(`"passengers":3`, `"passengers":0`), []string{"/airborneAndWaterborneVehicles/0/passengers outOfRange"}},
		{"armorClass 0", "FR-091", "unhappy (boundary)", vh(`"armorClass":11`, `"armorClass":0`), []string{"/airborneAndWaterborneVehicles/0/armorClass outOfRange"}},
		{"hitPoints 0", "FR-091", "unhappy (boundary)", vh(`"hitPoints":50`, `"hitPoints":0`), []string{"/airborneAndWaterborneVehicles/0/hitPoints outOfRange"}},
		{"damageThreshold 0", "FR-091", "unhappy (boundary)", vh(`"damageThreshold":5`, `"damageThreshold":0`), []string{"/airborneAndWaterborneVehicles/0/damageThreshold outOfRange"}},
		{"several vehicle faults", "FR-091", "unhappy (rule interaction)", vh(`"crew":1,"passengers":3`, `"crew":0,"passengers":0`),
			[]string{"/airborneAndWaterborneVehicles/0/crew outOfRange", "/airborneAndWaterborneVehicles/0/passengers outOfRange"}},
		{"lifestyle expense", "FR-054", "happy", eqDoc("lifestyleExpenses", lifestyleJSON), nil},
		{"lifestyle with a zero cost per Day", "FR-054", "unhappy (boundary)", ls(`"amount":1`, `"amount":0`), nil},
		{"lifestyle without text", "FR-116", "unhappy (missing content)", ls(`,"text":"Text."`, ``), []string{"/lifestyleExpenses/0/text missingField"}},
		{"lifestyle with blank text", "FR-116", "unhappy (invalid input)", ls(`"Text."`, `" "`), []string{"/lifestyleExpenses/0/text invalidValue"}},
		{"lifestyle cost per Hour", "FR-012", "unhappy (invalid input)", ls(`"Day"`, `"Hour"`), []string{"/lifestyleExpenses/0/cost/per invalidValue"}},
		{"lifestyle cost per of the wrong type", "FR-012", "unhappy (invalid input)", ls(`"Day"`, `1`), []string{"/lifestyleExpenses/0/cost/per wrongType"}},
		{"text on a hireling", "FR-116", "unhappy (invalid input)", hr(`"cost"`, `"text":"x","cost"`), []string{"/hirelings/0/text unknownField"}},
		{"text on a food row", "FR-116", "unhappy (invalid input)", fd(`"cost"`, `"text":"x","cost"`), []string{"/foodDrinkAndLodging/0/text unknownField"}},
		{"food", "FR-055", "happy", eqDoc("foodDrinkAndLodging", foodJSON), nil},
		{"food cost without coin", "FR-080", "unhappy (missing content)", fd(`,"coin":"CP"`, ``), []string{"/foodDrinkAndLodging/0/cost/coin missingField"}},
		{"hireling", "FR-056", "happy", eqDoc("hirelings", hirelingJSON), nil},
		{"hireling coin GC", "FR-080", "unhappy (invalid input)", hr(`"CP"`, `"GC"`), []string{"/hirelings/0/cost/coin invalidValue"}},
		{"hireling per Mile of the wrong case", "FR-012", "unhappy (invalid input)", hr(`"Mile"`, `"mile"`), []string{"/hirelings/0/cost/per invalidValue"}},
	})
}

func svc(id string, low, high int) string {
	return strings.NewReplacer(`"svc_x"`, `"`+id+`"`, `"spellLevelMin":0`, `"spellLevelMin":`+itoaT(low), `"spellLevelMax":9`, `"spellLevelMax":`+itoaT(high)).Replace(serviceJSON)
}

func itoaT(n int) string {
	if n < 0 {
		return "-" + itoaT(-n)
	}
	if n > 9 {
		return itoaT(n/10) + itoaT(n%10)
	}
	return string(rune('0' + n))
}

// Requirement: per case
// Path: per case.
func TestDecodeServicesAndScrollCosts(t *testing.T) {
	t.Parallel()
	sv := func(old, repl string) string { return with("spellcastingServices", serviceJSON, old, repl) }
	services := func(rows ...string) string { return eqDoc(append([]string{"spellcastingServices"}, rows...)...) }
	costs := func(rows ...string) string { return eqDoc(append([]string{"spellScrollCosts"}, rows...)...) }
	runDocCases(t, []docCase{
		{"one row covering 0 to 9", equipSpec + " FR-057", "happy", services(svc("a", 0, 9)), nil},
		{"seven rows", "FR-057", "happy", services(svc("a", 0, 0), svc("b", 1, 1), svc("c", 2, 2), svc("d", 3, 3), svc("e", 4, 5), svc("f", 6, 8), svc("g", 9, 9)), nil},
		{"rows in any order", "FR-057", "unhappy (boundary)", services(svc("g", 9, 9), svc("a", 0, 8)), nil},
		{"min above max", "FR-057", "unhappy (boundary)", sv(`"spellLevelMin":0,"spellLevelMax":9`, `"spellLevelMin":5,"spellLevelMax":4`), []string{"/spellcastingServices/0/spellLevelMax constraintViolation"}},
		{"max 10", "FR-057", "unhappy (boundary)", sv(`"spellLevelMax":9`, `"spellLevelMax":10`), []string{"/spellcastingServices/0/spellLevelMax outOfRange"}},
		{"min -1", "FR-057", "unhappy (boundary)", sv(`"spellLevelMin":0`, `"spellLevelMin":-1`), []string{"/spellcastingServices/0/spellLevelMin outOfRange"}},
		{"min as text", "FR-057", "unhappy (invalid input)", sv(`"spellLevelMin":0`, `"spellLevelMin":"0"`), []string{"/spellcastingServices/0/spellLevelMin wrongType"}},
		{"missing min", "FR-057", "unhappy (missing content)", sv(`"spellLevelMin":0,`, ``), []string{"/spellcastingServices/0/spellLevelMin missingField"}},
		{"missing max", "FR-057", "unhappy (missing content)", sv(`"spellLevelMax":9,`, ``), []string{"/spellcastingServices/0/spellLevelMax missingField"}},
		{"gap between rows", "FR-057", "unhappy (boundary)", services(svc("a", 0, 3), svc("b", 5, 9)), []string{"/spellcastingServices constraintViolation"}},
		{"overlap between rows", "FR-057", "unhappy (boundary)", services(svc("a", 0, 4), svc("b", 4, 9)), []string{"/spellcastingServices constraintViolation"}},
		{"rows stop short of 9", "FR-057", "unhappy (boundary)", services(svc("a", 0, 8)), []string{"/spellcastingServices constraintViolation"}},
		{"rows start above 0", "FR-057", "unhappy (boundary)", services(svc("a", 1, 9)), []string{"/spellcastingServices constraintViolation"}},
		{"availability Anywhere", "FR-057", "unhappy (invalid input)", sv(`"City only"`, `"Anywhere"`), []string{"/spellcastingServices/0/availability invalidValue"}},
		{"availability missing", "FR-057", "unhappy (missing content)", sv(`"availability":"City only",`, ``), []string{"/spellcastingServices/0/availability missingField"}},
		{"availability Town or city only", "FR-057", "happy", sv(`"City only"`, `"Town or city only"`), nil},
		{"availability Village, town, or city", "FR-057", "happy", sv(`"City only"`, `"Village, town, or city"`), nil},
		{"a bad row is not also a gap", "FR-057", "unhappy (rule interaction)", services(svc("a", 0, 3), mutate(serviceJSON, `"spellLevelMax":9`, `"spellLevelMax":10`)), []string{"/spellcastingServices/1/spellLevelMax outOfRange"}},
		{"ten scroll costs", "FR-064", "happy", costs(scrollCosts(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)...), nil},
		{"two rows with level 3", "FR-064", "unhappy (invalid input)", costs(scrollCosts(3)[0], strings.ReplaceAll(scrollCosts(3)[0], "cost_3", "cost_3b")), []string{"/spellScrollCosts/1/spellLevel duplicate"}},
		{"rows 0 to 8 only", "FR-064", "unhappy (boundary)", costs(scrollCosts(0, 1, 2, 3, 4, 5, 6, 7, 8)...), []string{"/spellScrollCosts constraintViolation"}},
		{"level 10", "FR-064", "unhappy (boundary)", costs(strings.Replace(scrollCosts(3)[0], `"spellLevel":3`, `"spellLevel":10`, 1)), []string{"/spellScrollCosts/0/spellLevel outOfRange"}},
		{"level -1", "FR-064", "unhappy (boundary)", costs(strings.Replace(scrollCosts(3)[0], `"spellLevel":3`, `"spellLevel":-1`, 1)), []string{"/spellScrollCosts/0/spellLevel outOfRange"}},
		{"days 0", "FR-064", "unhappy (boundary)", costs(strings.Replace(scrollCosts(3)[0], `"days":1`, `"days":0`, 1)), []string{"/spellScrollCosts/0/days outOfRange"}},
		{"days missing", "FR-064", "unhappy (missing content)", costs(strings.Replace(scrollCosts(3)[0], `"days":1,`, ``, 1)), []string{"/spellScrollCosts/0/days missingField"}},
		{"level missing", "FR-064", "unhappy (missing content)", costs(strings.Replace(scrollCosts(3)[0], `"spellLevel":3,`, ``, 1)), []string{"/spellScrollCosts/0/spellLevel missingField"}},
		{"cost missing", "FR-064, FR-080", "unhappy (missing content)", costs(strings.Replace(scrollCosts(3)[0], `,"cost":{"amount":15,"coin":"GP"}`, ``, 1)), []string{"/spellScrollCosts/0/cost missingField"}},
	})
}

func refProblems(t *testing.T, doc string) []string {
	t.Helper()
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", content.Problems(err))
	}
	_, err = content.ValidateSet(set)
	var out []string
	for _, p := range content.ReferenceProblems(err) {
		out = append(out, p.Owner.ID().String()+" "+p.Field+" "+p.Kind.String())
	}
	if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ValidateSet error does not wrap ErrInvalidInput: %v", err)
	}
	return out
}

// Requirement: equipSpec FR-067, FR-107 and doc/plans/plan-phase-2.md P4 - every
// cross-reference site of the equipment arrays is registered and checked for
// existence over the whole set: a missing target is unresolvedReference at its
// path, and the category of the target is not checked.
// Path: per case.
func TestEquipmentReferenceSites(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		doc                     string
		want                    []string
	}{
		{"every reference resolves", "FR-067", "happy", allEquipment(), nil},
		{"tool variants element", "FR-030", "unhappy (missing content)", eqDoc("tools", mutate(familyJSON, `["variant_x"]`, `["nothing"]`)), []string{"family_x variants/0 unresolvedReference"}},
		{"tool variantOf", "FR-030", "unhappy (missing content)", eqDoc("tools", mutate(variantJSON, `"family_x"`, `"no_such_family"`)), []string{"variant_x variantOf unresolvedReference"}},
		{"tool variantOf naming a non-family tool", "P4", "happy", eqDoc("weapons", weaponJSON, "tools", toolJSON, mutate(variantJSON, `"family_x"`, `"tool_x"`)), nil},
		{"craft item", "FR-031", "unhappy (missing content)", eqDoc("tools", mutate(toolJSON, `"club_x"`, `"nothing"`)), []string{"tool_x craft/0/id unresolvedReference"}},
		{"craft except", "FR-031", "unhappy (missing content)", eqDoc("tools", mutate(toolJSON, `{"kind":"item","id":"club_x"}`, `{"kind":"set","set":"MeleeWeapons","except":["nothing"]}`)), []string{"tool_x craft/0/except/0 unresolvedReference"}},
		{"craft item naming a non-weapon", "P4", "happy", eqDoc("tools", mutate(toolJSON, `"club_x"`, `"tool_x"`)), nil},
		{"gear variants element", "FR-039", "unhappy (missing content)", eqDoc("adventuringGear", mutate(gearFamilyJSON, `["arrows_x"]`, `["nothing"]`)), []string{"ammo_x variants/0 unresolvedReference"}},
		{"gear variantOf", "FR-039", "unhappy (missing content)", eqDoc("adventuringGear", mutate(arrowsJSON, `"ammo_x"`, `"nothing"`)), []string{"arrows_x variantOf unresolvedReference"}},
		{"gear variantOf naming a non-family", "P4", "happy", eqDoc("adventuringGear", plainJSON, mutate(arrowsJSON, `"ammo_x"`, `"gear_x"`)), nil},
		{"alsoCountsAs", "FR-040", "unhappy (missing content)", eqDoc("adventuringGear", focusFamilyJSON, focusJSON), []string{"staff_x alsoCountsAs unresolvedReference"}},
		{"alsoCountsAs naming a non-weapon", "P4", "happy", eqDoc("adventuringGear", focusFamilyJSON, plainJSON, mutate(focusJSON, `"club_x"`, `"gear_x"`)), nil},
		{"pack content", "FR-042", "unhappy (missing content)", eqDoc("adventuringGear", mutate(packJSON, `"gear_x"`, `"nothing"`), plainJSON, "weapons", weaponJSON), []string{"pack_x contents/0/id unresolvedReference"}},
		{"override tool", "FR-117", "unhappy (missing content)", eqDoc("adventuringGear", mutate(magicJSON, `["tool_x"]`, `["nothing"]`)), []string{"potion_x craftingOverride/toolIds/0 unresolvedReference"}},
		{"override tool naming a non-tool", "P4", "happy", eqDoc("adventuringGear", mutate(magicJSON, `["tool_x"]`, `["gear_x"]`), plainJSON), nil},
		{"two dangling references of one entry, in site order", "FR-067", "unhappy (rule interaction)", eqDoc("adventuringGear", mutate(mutate(focusJSON, `"club_x"`, `"nothing"`), `"focus_x"`, `"nope"`)),
			[]string{"staff_x variantOf unresolvedReference", "staff_x alsoCountsAs unresolvedReference"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := refProblems(t, c.doc); !slices.Equal(got, c.want) {
				t.Errorf("reference problems = %q, want %q [%s, %s]", got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: equipSpec FR-007, FR-090 - decoding twice gives equal sets, and
// the entries of one decode share nothing the caller can mutate.
// Path: unhappy (immutability).
func TestDecodedEquipmentIsIndependent(t *testing.T) {
	t.Parallel()
	doc := []byte(allEquipment())
	set, err := content.JSONDecoder{}.Decode(doc)
	if err != nil {
		t.Fatal(err)
	}
	clear(doc) // the input is not retained
	for _, e := range set.Entries() {
		if p, ok := e.(content.Pack); ok {
			if p.Contents()[0].ID() != "gear_x" {
				t.Errorf("the pack lost its contents after the input was cleared")
			}
		}
	}
}

// Requirement: equipSpec FR-030, FR-039 - a variant listed by two families, and
// a listed variant whose variantOf names another family, are
// constraintViolation at the later variants/<j>; a family listing only its own
// variants is accepted.
// Path: unhappy (rule interaction).
func TestDecodeFamilyLinks(t *testing.T) {
	t.Parallel()
	famY := mutate(familyJSON, `family_x`, `family_y`)                         // lists variant_x too
	famTwo := mutate(familyJSON, `["variant_x"]`, `["variant_x","variant_y"]`) // variant_y at index 1
	varY := mutate(variantJSON, `variant_x`, `variant_y`)
	famZ := mutate(mutate(familyJSON, `family_x`, `family_y`), `variant_x`, `variant_y`)
	gamY := mutate(gearFamilyJSON, `ammo_x`, `ammo_y`)
	runDocCases(t, []docCase{
		{"each family lists its own", "FR-030", "happy", eqDoc("tools", familyJSON, variantJSON, famZ, mutate(varY, `family_x`, `family_y`)), nil},
		{"tool listed by two families", "FR-030", "unhappy (rule interaction)", eqDoc("tools", familyJSON, famY, variantJSON), []string{"/tools/1/variants/0 constraintViolation"}},
		{"duplicate reported at the later index", "FR-030", "unhappy (boundary)", eqDoc("tools", famZ, famTwo, variantJSON, mutate(varY, `family_x`, `family_y`)), []string{"/tools/1/variants/1 constraintViolation"}},
		{"tool listed under another family", "FR-030", "unhappy (rule interaction)", eqDoc("tools", famY, variantJSON), []string{"/tools/0/variants/0 constraintViolation"}},
		{"family listed as a variant is not this check's", "FR-030", "happy", eqDoc("tools", familyJSON, variantJSON, mutate(mutate(familyJSON, `family_x`, `family_y`), `["variant_x"]`, `["family_x"]`)), nil},
		{"variant of a later family listed by an earlier one", "FR-030", "unhappy (rule interaction)", eqDoc("tools", mutate(familyJSON, `family_x`, `family_a`), mutate(familyJSON, `variant_x`, `variant_z`), variantJSON), []string{"/tools/0/variants/0 constraintViolation"}},
		{"tool variant before its family", "FR-030", "unhappy (rule interaction)", eqDoc("tools", variantJSON, famY), []string{"/tools/1/variants/0 constraintViolation"}},
		{"tool listed with no variantOf is not this check's", "FR-030", "happy", eqDoc("tools", familyJSON, mutate(toolJSON, `tool_x`, `variant_x`)), nil},
		{"gear listed by two families", "FR-039", "unhappy (rule interaction)", eqDoc("adventuringGear", gearFamilyJSON, gamY, arrowsJSON), []string{"/adventuringGear/1/variants/0 constraintViolation"}},
		{"gear listed under another family", "FR-039", "unhappy (rule interaction)", eqDoc("adventuringGear", gamY, arrowsJSON), []string{"/adventuringGear/0/variants/0 constraintViolation"}},
		{"scroll variant listed under another family", "FR-039", "unhappy (rule interaction)", eqDoc("adventuringGear", mutate(scrollFamJSON, `scroll_x`, `scroll_y`), scrollVarJSON), []string{"/adventuringGear/0/variants/0 constraintViolation"}},
		{"focus variant listed under another family", "FR-039", "unhappy (rule interaction)", eqDoc("adventuringGear", mutate(focusFamilyJSON, `focus_x`, `focus_y`), focusJSON), []string{"/adventuringGear/0/variants/0 constraintViolation"}},
	})
}

// Requirement: equipSpec FR-081, FR-080 - a Spell Scroll row whose own weight or
// cost failed to read reports that problem only, not also the FR-081
// constraintViolation (one bad field never invents a second problem).
// Path: unhappy (invalid input).
func TestDecodeScrollBadFieldsReportOnce(t *testing.T) {
	t.Parallel()
	runDocCases(t, []docCase{
		{"scroll variant with a negative weight", "FR-081", "unhappy (boundary)", eqDoc("adventuringGear", mutate(scrollVarJSON, `"spellLevel"`, `"weightPounds":-1,"spellLevel"`)), []string{"/adventuringGear/0/weightPounds outOfRange"}},
		{"scroll family with a cost of the wrong type", "FR-081", "unhappy (invalid input)", eqDoc("adventuringGear", mutate(scrollFamJSON, `"costVaries":true`, `"cost":{"amount":"x","coin":"GP"}`)), []string{"/adventuringGear/0/cost/amount wrongType"}},
		{"scroll variant with a valid weight", "FR-081", "unhappy (rule interaction)", eqDoc("adventuringGear", mutate(scrollVarJSON, `"spellLevel"`, `"weightPounds":1,"spellLevel"`)), []string{"/adventuringGear/0/weightPounds constraintViolation"}},
	})
}
