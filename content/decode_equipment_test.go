package content_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// The fixtures are synthetic: made-up ids, names and numbers, never SRD data.

const (
	weaponJSON = `{"id":"club_x","name":"Club X","category":"Simple","weaponType":"Melee","damage":{"dice":"1d6"},"damageType":"Bludgeoning","properties":[],"mastery":"Sap","weightPounds":2,"cost":{"amount":1,"coin":"SP"}}`
	armorJSON  = `{"id":"armor_x","name":"Armor X","category":"Light","armorClass":{"base":11,"dexterity":"full"},"stealthDisadvantage":false,"weightPounds":10,"cost":{"amount":10,"coin":"GP"}}`
	shieldJSON = `{"id":"shield_x","name":"Shield X","category":"Shield","armorClass":{"bonus":2},"stealthDisadvantage":false,"weightPounds":6,"cost":{"amount":10,"coin":"GP"}}`

	toolJSON    = `{"id":"tool_x","name":"Tool X","toolGroup":"Artisan's Tools","ability":"dexterity","utilize":[{"description":"Do a thing","dc":10}],"craft":[{"kind":"item","id":"club_x"}],"weightPounds":5,"cost":{"amount":5,"coin":"GP"}}`
	familyJSON  = `{"id":"family_x","name":"Family X","toolGroup":"Other Tools","ability":"charisma","utilize":[{"description":"Do a thing","dc":10}],"variants":["variant_x"],"costVaries":true,"weightVaries":true}`
	variantJSON = `{"id":"variant_x","name":"Variant X","variantOf":"family_x","cost":{"amount":1,"coin":"GP"}}`

	plainJSON       = `{"id":"gear_x","name":"Gear X","gearKind":"plain","weightPounds":1,"cost":{"amount":2,"coin":"SP"},"text":"Text."}`
	magicJSON       = `{"id":"potion_x","name":"Potion X","gearKind":"magic","cost":{"amount":50,"coin":"GP"},"text":"Text.","craftingOverride":{"rawMaterials":{"amount":25,"coin":"GP"},"days":1,"toolIds":["tool_x"]}}`
	gearFamilyJSON  = `{"id":"ammo_x","name":"Ammo X","gearKind":"family","costVaries":true,"weightVaries":true,"text":"Text.","variants":["arrows_x"]}`
	arrowsJSON      = `{"id":"arrows_x","name":"Arrows X","gearKind":"ammunitionVariant","variantOf":"ammo_x","weightPounds":1,"cost":{"amount":1,"coin":"GP"},"ammunition":{"amount":20,"storage":"Quiver"}}`
	focusFamilyJSON = `{"id":"focus_x","name":"Focus X","gearKind":"family","costVaries":true,"text":"Text.","variants":["staff_x"]}`
	focusJSON       = `{"id":"staff_x","name":"Staff X","gearKind":"focusVariant","variantOf":"focus_x","weightPounds":4,"cost":{"amount":5,"coin":"GP"},"alsoCountsAs":"club_x","note":"held"}`
	scrollFamJSON   = `{"id":"scroll_x","name":"Scroll X","gearKind":"scrollFamily","costVaries":true,"text":"Text.","variants":["scroll_0"]}`
	scrollVarJSON   = `{"id":"scroll_0","name":"Scroll 0","gearKind":"scrollVariant","variantOf":"scroll_x","cost":{"amount":30,"coin":"GP"},"spellLevel":0}`
	packJSON        = `{"id":"pack_x","name":"Pack X","gearKind":"pack","weightPounds":10,"cost":{"amount":5,"coin":"GP"},"text":"Text.","contents":[{"id":"gear_x","quantity":2,"unit":"days"},{"id":"club_x","quantity":1}]}`

	mountJSON     = `{"id":"mount_x","name":"Mount X","carryingCapacityPounds":480,"cost":{"amount":50,"coin":"GP"}}`
	tackJSON      = `{"id":"tack_x","name":"Tack X","weightPounds":25,"cost":{"amount":10,"coin":"GP"}}`
	vehicleJSON   = `{"id":"boat_x","name":"Boat X","speedMph":1.5,"crew":1,"passengers":3,"cargoTons":0.5,"armorClass":11,"hitPoints":50,"damageThreshold":5,"weightPounds":100,"cost":{"amount":50,"coin":"GP"}}`
	lifestyleJSON = `{"id":"squalid_x","name":"Squalid X","cost":{"amount":1,"coin":"SP","per":"Day"},"text":"Text."}`
	foodJSON      = `{"id":"ale_x","name":"Ale X","cost":{"amount":4,"coin":"CP"}}`
	hirelingJSON  = `{"id":"hire_x","name":"Hire X","cost":{"amount":2,"coin":"CP","per":"Mile"}}`
	serviceJSON   = `{"id":"svc_x","name":"Svc X","spellLevelMin":0,"spellLevelMax":9,"availability":"City only","cost":{"amount":50,"coin":"GP"}}`
	scrollCostFmt = `{"id":"cost_N","name":"Cost N","spellLevel":N,"days":1,"cost":{"amount":15,"coin":"GP"}}`
)

func eqDoc(parts ...string) string { // eqDoc("weapons", w1, w2, "armor", a1)
	var arrays []string
	for i := 0; i < len(parts); {
		j := i + 1
		for j < len(parts) && !isArrayKey(parts[j]) {
			j++
		}
		arrays = append(arrays, `"`+parts[i]+`":[`+strings.Join(parts[i+1:j], ",")+`]`)
		i = j
	}
	return `{"schemaVersion":1,` + strings.Join(arrays, ",") + `}`
}

func isArrayKey(s string) bool {
	switch s {
	case "weapons", "armor", "tools", "adventuringGear", "mounts", "tackHarnessAndDrawnVehicles", "airborneAndWaterborneVehicles",
		"lifestyleExpenses", "foodDrinkAndLodging", "hirelings", "spellcastingServices", "spellScrollCosts", "feats", "backgrounds", "species", "classes":
		return true
	}
	return false
}

func scrollCosts(levels ...int) []string {
	var out []string
	for _, l := range levels {
		out = append(out, strings.ReplaceAll(scrollCostFmt, "N", strconv.Itoa(l)))
	}
	return out
}

// allEquipment is a document with an entry in each of the twelve arrays, in
// which every reference resolves.
func allEquipment() string {
	return eqDoc("weapons", weaponJSON, "armor", armorJSON, shieldJSON,
		"tools", toolJSON, familyJSON, variantJSON,
		"adventuringGear", plainJSON, magicJSON, gearFamilyJSON, arrowsJSON, focusFamilyJSON, focusJSON, scrollFamJSON, scrollVarJSON, packJSON,
		"mounts", mountJSON, "tackHarnessAndDrawnVehicles", tackJSON, "airborneAndWaterborneVehicles", vehicleJSON,
		"lifestyleExpenses", lifestyleJSON, "foodDrinkAndLodging", foodJSON, "hirelings", hirelingJSON,
		"spellcastingServices", serviceJSON, "spellScrollCosts", strings.Join(scrollCosts(0, 1, 2, 3, 4, 5, 6, 7, 8, 9), ","))
}

// Requirement: equipSpec FR-001, FR-008, FR-066 (wire), FR-079 (shape) - a
// document with an entry in each of the twelve arrays decodes to equipment
// entries of the right types, and every reference of it resolves.
// Path: happy.
func TestDecodeEquipmentDocument(t *testing.T) {
	t.Parallel()
	set, err := content.JSONDecoder{}.Decode([]byte(allEquipment()))
	if err != nil {
		t.Fatalf("Decode: %v", content.Problems(err))
	}
	counts := map[content.EquipmentCategory]int{}
	for _, e := range set.Entries() {
		entry, ok := e.(content.EquipmentEntry)
		if !ok || e.Key().Kind() != content.KindEquipment {
			t.Fatalf("entry %v is not an EquipmentEntry of kind equipment", e.Key())
		}
		counts[entry.EquipmentCategory()]++
	}
	want := map[content.EquipmentCategory]int{
		content.EquipmentWeapon: 1, content.EquipmentArmor: 2, content.EquipmentTool: 3, content.EquipmentAdventuringGear: 9,
		content.EquipmentMount: 1, content.EquipmentTackHarnessOrDrawnVehicle: 1, content.EquipmentAirborneOrWaterborneVehicle: 1,
		content.EquipmentLifestyleExpense: 1, content.EquipmentFoodDrinkOrLodging: 1, content.EquipmentHireling: 1,
		content.EquipmentSpellcastingService: 1, content.EquipmentSpellScrollCost: 10,
	}
	if !equalCounts(counts, want) {
		t.Errorf("entries per category = %v, want %v [FR-001, FR-002]", counts, want)
	}
	if _, err := content.ValidateSet(set); err != nil {
		t.Errorf("ValidateSet over the document: %v [FR-067]", content.ReferenceProblems(err))
	}
	again, err := content.JSONDecoder{}.Decode([]byte(allEquipment()))
	if err != nil || !again.Equal(set) {
		t.Errorf("a second decode differs: %v [FR-068 determinism]", err)
	}
	// Spot checks of the values behind the types.
	lookup := func(id string) content.Entry {
		for _, e := range set.Entries() {
			if e.Key() == content.NewKey(content.KindEquipment, content.MustID(id)) {
				return e
			}
		}
		t.Fatalf("no entry %s", id)
		return nil
	}
	if g, ok := lookup("potion_x").(content.MagicGear); !ok {
		t.Error("potion_x is not MagicGear")
	} else if o, ok := g.CraftingOverride(); !ok || o.Days() != 1 || o.RawMaterials() != content.MustCost(25, content.CoinGP, 0) || !slices.Equal(o.ToolIDs(), []content.ItemID{"tool_x"}) {
		t.Errorf("override = %+v, %v [FR-094]", o, ok)
	}
	if v, ok := lookup("boat_x").(content.AirborneOrWaterborneVehicle); !ok || v.SpeedMph() != 1.5 {
		t.Errorf("boat_x = %+v", lookup("boat_x"))
	}
	if h, ok := lookup("hire_x").(content.Hireling); !ok {
		t.Error("hire_x is not a Hireling")
	} else if per, has := h.Price().Cost().Per(); !has || per != content.CostPerMile {
		t.Error("the hireling's rate is not per Mile [FR-012]")
	}
	if a, ok := lookup("armor_x").(content.Armor); !ok {
		t.Error("armor_x is not Armor")
	} else if r, ok := a.ArmorClass().(content.BaseRule); !ok || r.Base() != 11 || r.Dexterity() != content.DexterityBonus(content.FullDexterity{}) {
		t.Errorf("armor_x rule = %#v [FR-093]", a.ArmorClass())
	}
	if w, ok := lookup("club_x").(content.Weapon); !ok || w.Properties() != nil || w.Mastery() != content.MasterySap {
		t.Errorf("club_x = %+v", lookup("club_x"))
	}
}

func equalCounts(a, b map[content.EquipmentCategory]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Requirement: equipSpec FR-008 - an absent array means no entries, an array
// may be present and empty, and any other root key is unknownField.
// Path: unhappy (boundary).
func TestDecodeEquipmentRoot(t *testing.T) {
	t.Parallel()
	runDocCases(t, []docCase{
		{"no arrays", equipSpec + " FR-008", "happy", `{"schemaVersion":1}`, nil},
		{"every array present and empty", "FR-008", "happy", eqDoc("weapons", "armor", "tools", "adventuringGear", "mounts", "tackHarnessAndDrawnVehicles",
			"airborneAndWaterborneVehicles", "lifestyleExpenses", "foodDrinkAndLodging", "hirelings", "spellcastingServices", "spellScrollCosts"), nil},
		{"unregistered root key", "FR-008", "unhappy (boundary)", `{"schemaVersion":1,"shields":[]}`, []string{"/shields unknownField"}},
		{"an array of the wrong type", "FR-008", "unhappy (invalid input)", `{"schemaVersion":1,"tools":{}}`, []string{"/tools wrongType"}},
		{"duplicate id in one array", "FR-004", "unhappy (invalid input)", eqDoc("weapons", weaponJSON, weaponJSON), []string{"/weapons/1/id duplicate"}},
		{"duplicate id across arrays", "FR-004", "unhappy (invalid input)", eqDoc("weapons", weaponJSON, "tools", mutate(toolJSON, `"tool_x"`, `"club_x"`)), []string{"/tools/0/id duplicate"}},
		{"duplicates in array order", "FR-008, FR-016", "unhappy (rule interaction)",
			eqDoc("hirelings", hirelingJSON, hirelingJSON, "weapons", weaponJSON, weaponJSON), []string{"/weapons/1/id duplicate", "/hirelings/1/id duplicate"}},
		{"a weapon and an armor with one id", "FR-004", "unhappy (invalid input)", eqDoc("weapons", weaponJSON, "armor", mutate(armorJSON, `"armor_x"`, `"club_x"`)), []string{"/armor/0/id duplicate"}},
	})
}

// Requirement: equipSpec FR-068 - a document with one fault in each of the
// twelve arrays is rejected with every fault listed in root order.
// Path: unhappy (invalid input).
func TestDecodeEquipmentOneFaultEach(t *testing.T) {
	t.Parallel()
	doc := eqDoc("weapons", mutate(weaponJSON, `"Simple"`, `"Exotic"`),
		"armor", mutate(armorJSON, `"Light"`, `"Heavy Armor"`),
		"tools", mutate(toolJSON, `"dc":10`, `"dc":0`),
		"adventuringGear", mutate(plainJSON, `"plain"`, `"pouch"`),
		"mounts", mutate(mountJSON, `480`, `0`),
		"tackHarnessAndDrawnVehicles", mutate(tackJSON, `"GP"`, `"GC"`),
		"airborneAndWaterborneVehicles", mutate(vehicleJSON, `1.5`, `1.25`),
		"lifestyleExpenses", mutate(lifestyleJSON, `"Day"`, `"Hour"`),
		"foodDrinkAndLodging", mutate(foodJSON, `4`, `-1`),
		"hirelings", mutate(hirelingJSON, `"Mile"`, `"Hour"`),
		"spellcastingServices", mutate(serviceJSON, `"City only"`, `"Anywhere"`),
		"spellScrollCosts", mutate(scrollCosts(3)[0], `"days":1`, `"days":0`))
	want := []string{
		"/weapons/0/category invalidValue", "/armor/0/category invalidValue", "/tools/0/utilize/0/dc outOfRange", "/adventuringGear/0/gearKind invalidValue",
		"/mounts/0/carryingCapacityPounds outOfRange", "/tackHarnessAndDrawnVehicles/0/cost/coin invalidValue",
		"/airborneAndWaterborneVehicles/0/speedMph outOfRange", "/lifestyleExpenses/0/cost/per invalidValue", "/foodDrinkAndLodging/0/cost/amount outOfRange",
		"/hirelings/0/cost/per invalidValue", "/spellcastingServices/0/availability invalidValue", "/spellScrollCosts/0/days outOfRange",
	}
	got := decodeProblems(doc)
	if !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q [FR-068]", got, want)
	}
	if again := decodeProblems(doc); !slices.Equal(got, again) {
		t.Error("the same text gave a different result [FR-068 determinism]")
	}
}

func weaponWith(old, repl string) string { return eqDoc("weapons", mutate(weaponJSON, old, repl)) }

// Requirement: per case
// Path: per case.
func TestDecodeWeapon(t *testing.T) {
	t.Parallel()
	const p = "/weapons/0/"
	props := func(list string) string { return weaponWith(`"properties":[]`, `"properties":[`+list+`]`) }
	rng := `"range":{"normal":20,"long":60}`
	runDocCases(t, []docCase{
		{"valid", equipSpec + " FR-106", "happy", eqDoc("weapons", weaponJSON), nil},
		{"no weight", "FR-106", "happy", weaponWith(`"weightPounds":2,`, ``), nil},
		{"fixed damage 1", "FR-102, FR-110", "happy", weaponWith(`{"dice":"1d6"}`, `{"fixed":1}`), nil},
		{"missing category", "FR-109", "unhappy (missing content)", weaponWith(`"category":"Simple",`, ``), []string{p + "category missingField"}},
		{"missing weaponType", "FR-109", "unhappy (missing content)", weaponWith(`"weaponType":"Melee",`, ``), []string{p + "weaponType missingField"}},
		{"missing damage", "FR-109", "unhappy (missing content)", weaponWith(`"damage":{"dice":"1d6"},`, ``), []string{p + "damage missingField"}},
		{"missing damageType", "FR-109", "unhappy (missing content)", weaponWith(`"damageType":"Bludgeoning",`, ``), []string{p + "damageType missingField"}},
		{"missing properties", "FR-109", "unhappy (missing content)", weaponWith(`"properties":[],`, ``), []string{p + "properties missingField"}},
		{"missing mastery", "FR-109", "unhappy (missing content)", weaponWith(`"mastery":"Sap",`, ``), []string{p + "mastery missingField"}},
		{"missing cost", "FR-109, FR-080", "unhappy (missing content)", weaponWith(`,"cost":{"amount":1,"coin":"SP"}`, ``), []string{p + "cost missingField"}},
		{"category Exotic", "FR-109", "unhappy (invalid input)", weaponWith(`"Simple"`, `"Exotic"`), []string{p + "category invalidValue"}},
		{"weaponType Thrown", "FR-109", "unhappy (invalid input)", weaponWith(`"Melee"`, `"Thrown"`), []string{p + "weaponType invalidValue"}},
		{"damageType Psychics", "FR-109", "unhappy (invalid input)", weaponWith(`"Bludgeoning"`, `"Psychics"`), []string{p + "damageType invalidValue"}},
		{"mastery Pierce", "FR-109", "unhappy (invalid input)", weaponWith(`"Sap"`, `"Pierce"`), []string{p + "mastery invalidValue"}},
		{"category of the wrong type", "FR-109", "unhappy (invalid input)", weaponWith(`"Simple"`, `1`), []string{p + "category wrongType"}},
		{"weight below 0", "FR-080", "unhappy (boundary)", weaponWith(`"weightPounds":2`, `"weightPounds":-0.25`), []string{p + "weightPounds outOfRange"}},
		{"weight of the wrong type", "FR-080", "unhappy (invalid input)", weaponWith(`"weightPounds":2`, `"weightPounds":"2"`), []string{p + "weightPounds wrongType"}},
		{"weight too large for a float", "FR-080", "unhappy (boundary)", weaponWith(`"weightPounds":2`, `"weightPounds":1e999`), []string{p + "weightPounds outOfRange"}},
		{"weightVaries is not a weapon field", "FR-106", "unhappy (invalid input)", weaponWith(`"weightPounds":2,`, `"weightVaries":true,`), []string{p + "weightVaries unknownField"}},
		{"weightNote is not a weapon field", "FR-106", "unhappy (invalid input)", weaponWith(`"weightPounds":2,`, `"weightPounds":2,"weightNote":"x",`), []string{p + "weightNote unknownField"}},
		{"unknown field", "FR-109", "unhappy (invalid input)", weaponWith(`"mastery":"Sap",`, `"mastery":"Sap","finesse":true,`), []string{p + "finesse unknownField"}},
		{"costVaries and cost on a weapon", "FR-014", "unhappy (rule interaction)", weaponWith(`"weightPounds":2,`, `"weightPounds":2,"costVaries":true,`), []string{p + "costVaries constraintViolation"}},
		{"costVaries on a weapon", "FR-014", "unhappy (rule interaction)", weaponWith(`,"cost":{"amount":1,"coin":"SP"}`, `,"costVaries":true`), []string{p + "costVaries constraintViolation"}},
		{"damage of the wrong type", "FR-110", "unhappy (boundary)", weaponWith(`{"dice":"1d6"}`, `6`), []string{p + "damage wrongType"}},
		{"damage with dice and fixed", "FR-110", "unhappy (boundary)", weaponWith(`{"dice":"1d6"}`, `{"dice":"1d6","fixed":1}`), []string{p + "damage constraintViolation"}},
		{"damage with neither", "FR-110", "unhappy (boundary)", weaponWith(`{"dice":"1d6"}`, `{}`), []string{p + "damage missingField"}},
		{"dice xd8", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"xd8"`), []string{p + "damage/dice invalidValue"}},
		{"dice d8, which ParseExpression reads", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"d8"`), []string{p + "damage/dice invalidValue"}},
		{"dice 1d8+1", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"1d8+1"`), []string{p + "damage/dice invalidValue"}},
		{"dice 1d", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"1d"`), []string{p + "damage/dice invalidValue"}},
		{"dice 0d6", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"0d6"`), []string{p + "damage/dice invalidValue"}},
		{"dice with a leading zero", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"01d6"`), []string{p + "damage/dice invalidValue"}},
		{"dice 1d6+0", "FR-110", "unhappy (boundary)", weaponWith(`"1d6"`, `"1d6+0"`), []string{p + "damage/dice invalidValue"}},
		{"dice of the wrong type", "FR-110", "unhappy (invalid input)", weaponWith(`"1d6"`, `6`), []string{p + "damage/dice wrongType"}},
		{"fixed 0", "FR-110", "unhappy (boundary)", weaponWith(`{"dice":"1d6"}`, `{"fixed":0}`), []string{p + "damage/fixed outOfRange"}},
		{"fixed fractional", "FR-110", "unhappy (invalid input)", weaponWith(`{"dice":"1d6"}`, `{"fixed":1.5}`), []string{p + "damage/fixed wrongType"}},
		{"unknown damage key", "FR-110", "unhappy (invalid input)", weaponWith(`{"dice":"1d6"}`, `{"dice":"1d6","modifier":1}`), []string{p + "damage/modifier unknownField"}},
		// FR-111 properties.
		{"property Range", "FR-111", "unhappy (invalid input)", props(`{"property":"Range"}`), []string{p + "properties/0/property invalidValue"}},
		{"missing property", "FR-111", "unhappy (missing content)", props(`{}`), []string{p + "properties/0/property missingField"}},
		{"property Light twice", "FR-111", "unhappy (invalid input)", props(`{"property":"Light"},{"property":"Light"}`), []string{p + "properties/1 duplicate"}},
		{"Thrown without range", "FR-111", "unhappy (missing content)", props(`{"property":"Thrown"}`), []string{p + "properties/0/range missingField"}},
		{"Thrown with range", "FR-111", "happy", props(`{"property":"Thrown",` + rng + `}`), nil},
		{"Finesse with range", "FR-111", "unhappy (rule interaction)", props(`{"property":"Finesse",` + rng + `}`), []string{p + "properties/0/range constraintViolation"}},
		{"range normal 0", "FR-111", "unhappy (boundary)", props(`{"property":"Thrown","range":{"normal":0,"long":10}}`), []string{p + "properties/0/range/normal outOfRange"}},
		{"range long equals normal", "FR-111", "unhappy (boundary)", props(`{"property":"Thrown","range":{"normal":20,"long":20}}`), []string{p + "properties/0/range/long constraintViolation"}},
		{"range long below normal", "FR-111", "unhappy (boundary)", props(`{"property":"Thrown","range":{"normal":20,"long":10}}`), []string{p + "properties/0/range/long constraintViolation"}},
		{"range missing a part", "FR-111", "unhappy (missing content)", props(`{"property":"Thrown","range":{"normal":20}}`), []string{p + "properties/0/range/long missingField"}},
		{"range of the wrong type", "FR-111", "unhappy (invalid input)", props(`{"property":"Thrown","range":3}`), []string{p + "properties/0/range wrongType"}},
		{"Ammunition without ammunition", "FR-111", "unhappy (missing content)", props(`{"property":"Ammunition",` + rng + `}`), []string{p + "properties/0/ammunition missingField"}},
		{"Ammunition Stone", "FR-111", "unhappy (invalid input)", props(`{"property":"Ammunition",` + rng + `,"ammunition":"Stone"}`), []string{p + "properties/0/ammunition invalidValue"}},
		{"Ammunition Bolt", "FR-111", "happy", props(`{"property":"Ammunition",` + rng + `,"ammunition":"Bolt"}`), nil},
		{"Light with ammunition", "FR-111", "unhappy (rule interaction)", props(`{"property":"Light","ammunition":"Bolt"}`), []string{p + "properties/0/ammunition constraintViolation"}},
		{"Versatile without damage", "FR-111", "unhappy (missing content)", props(`{"property":"Versatile"}`), []string{p + "properties/0/damage missingField"}},
		{"Versatile 1d8", "FR-111", "happy", props(`{"property":"Versatile","damage":"1d8"}`), nil},
		{"Versatile damage malformed", "FR-111", "unhappy (invalid input)", props(`{"property":"Versatile","damage":"d8"}`), []string{p + "properties/0/damage invalidValue"}},
		{"Light with damage", "FR-111", "unhappy (rule interaction)", props(`{"property":"Light","damage":"1d8"}`), []string{p + "properties/0/damage constraintViolation"}},
		{"Light with condition", "FR-111", "unhappy (rule interaction)", props(`{"property":"Light","condition":"unless mounted"}`), []string{p + "properties/0/condition constraintViolation"}},
		{"Two-Handed", "FR-111", "happy", props(`{"property":"Two-Handed"}`), nil},
		{"Two-Handed unless mounted", "FR-111", "happy", props(`{"property":"Two-Handed","condition":"unless mounted"}`), nil},
		{"Two-Handed condition other", "FR-111", "unhappy (invalid input)", props(`{"property":"Two-Handed","condition":"never"}`), []string{p + "properties/0/condition invalidValue"}},
		{"Two-Handed condition of the wrong type", "FR-111", "unhappy (invalid input)", props(`{"property":"Two-Handed","condition":1}`), []string{p + "properties/0/condition wrongType"}},
		{"unknown property key", "FR-111", "unhappy (invalid input)", props(`{"property":"Light","reach":5}`), []string{p + "properties/0/reach unknownField"}},
		{"property that is not an object", "FR-111", "unhappy (invalid input)", props(`"Light"`), []string{p + "properties/0 wrongType"}},
		{"properties of the wrong type", "FR-111", "unhappy (invalid input)", weaponWith(`"properties":[]`, `"properties":{}`), []string{p + "properties wrongType"}},
		{"Ammunition with a bad range and bad ammunition", "FR-111", "unhappy (rule interaction)", props(`{"property":"Ammunition","range":{"normal":0,"long":1},"ammunition":"Stone"}`),
			[]string{p + "properties/0/range/normal outOfRange", p + "properties/0/ammunition invalidValue"}},
		{"a repeat with a bad field", "FR-111", "unhappy (rule interaction)", props(`{"property":"Light"},{"property":"Light","damage":"1d8"}`),
			[]string{p + "properties/1 duplicate", p + "properties/1/damage constraintViolation"}},
	})
}
