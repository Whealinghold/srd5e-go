package content_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// richClassJSON uses every part of the class shape once: an any-of primary
// ability, every training form, equipment option C, multiclass grants, Pact
// Magic, one column of each kind, a spell fact, options, a table and a
// subclass.
func richClassJSON() string {
	s := buildClass("fixture", columnDecls, `{"kind":"pact","ability":"charisma"}`, classRows("fixture", pactRowJSON, columnValues), subclassJSON("sub_a", 3, 6))
	s = mutate(s, `{"kind":"allOf","abilities":["intelligence"]}`, `{"kind":"anyOf","abilities":["strength","dexterity"]}`)
	s = mutate(s, `{"picks":2,"options":{"kind":"listed","skills":["Arcana","History","Insight"]}}`, `{"picks":1,"options":{"kind":"any"}}`)
	s = mutate(s, `[{"category":"Simple"}]`, `[{"category":"Simple"},{"category":"Martial","requiresAnyProperty":["Light","Finesse"]}]`)
	s = mutate(s, `"toolProficiencies":[]`, `"toolProficiencies":[{"kind":"specific","name":"Kit"},{"kind":"choice","categories":["Instrument","Gaming"],"picks":2}]`)
	s = mutate(s, `"armorTraining":[]`, `"armorTraining":["Light","Shield"]`)
	s = mutate(s, `"B":{"items":[],"gold":50}`, `"B":{"items":[],"gold":50},"C":{"items":[{"name":"Rope","quantity":2},{"name":"Cup","quantity":1}],"gold":7}`)
	s = mutate(s, `"multiclassGrants":{"weaponProficiencies":[],"armorTraining":[],"toolProficiencies":[]}`,
		`"multiclassGrants":{"weaponProficiencies":[{"category":"Simple"}],"armorTraining":["Light"],"toolProficiencies":[{"kind":"specific","name":"Kit"}],"skillChoice":{"picks":1,"options":{"kind":"any"}}}`)
	s = mutate(s, alphaTail, withFacts(alphaTail, spellFact(`{"level":1,"spellIds":["spell_a","spell_b"]},{"level":5,"spellIds":["spell_c"]}`)))
	s = mutate(s, betaTail, withOptions(betaTail, option("o1", `,"pointCost":2,"repeatable":true,"prerequisite":{"minClassLevel":5,"requiresOptionIds":["o2"],"note":"A note."}`)+","+option("o2", "")))
	s = mutate(s, gammaTail, strings.TrimSuffix(gammaTail, "}")+`,"tables":[{"name":"T","columns":["A","B"],"rows":[["x",1],["y",-2]]}]}`)
	s = mutate(s, `{"featureId":"fixture_gamma"}`, `{"featureId":"fixture_gamma","annotation":"one use"}`)
	return s
}

// Requirement: doc/specs/content-classes.md FR-011 to FR-066 - the decoder builds
// exactly what the document says: every part of a class that uses each form once
// equals the same class built with the constructors.
// Path: happy.
func TestDecodeClassBuildsEveryPart(t *testing.T) {
	t.Parallel()
	c := decodeClassFixture(t, richClassJSON())
	eq := func(what string, got, want any) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %#v, want %#v", what, got, want)
		}
	}
	eq("primary ability", c.PrimaryAbility(), content.AbilityRequirement(mustValue(content.NewAnyOf(ability.Strength, ability.Dexterity))))
	eq("skill choice", c.SkillChoice(), content.SkillChoice(mustValue(content.NewSkillChoiceAny(1))))
	eq("weapons", c.WeaponProficiencies(), []content.WeaponProficiency{
		mustValue(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil)),
		mustValue(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{content.WeaponPropertyLight, content.WeaponPropertyFinesse})),
	})
	eq("tools", c.ToolProficiencies(), []content.ClassToolProficiency{
		mustValue(content.NewClassToolProficiencySpecific("Kit")),
		mustValue(content.NewClassToolProficiencyChoice([]string{"Instrument", "Gaming"}, 2)),
	})
	eq("armor", c.ArmorTraining(), []content.ArmorTrainingCategory{content.ArmorTrainingLight, content.ArmorTrainingShield})
	staff := mustValue(content.NewClassEquipmentItem("Staff", 1))
	optA := mustValue(content.NewClassEquipmentOption([]content.ClassEquipmentItem{staff}, 5))
	optB := mustValue(content.NewClassEquipmentOption(nil, 50))
	optC := mustValue(content.NewClassEquipmentOption([]content.ClassEquipmentItem{mustValue(content.NewClassEquipmentItem("Rope", 2)), mustValue(content.NewClassEquipmentItem("Cup", 1))}, 7))
	eq("equipment", c.StartingEquipment(), mustValue(content.NewClassStartingEquipment(optA, optB, optC)))
	eq("multiclass grants", c.MulticlassGrants(), mustValue(content.NewMulticlassGrants(
		[]content.WeaponProficiency{mustValue(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil))},
		[]content.ArmorTrainingCategory{content.ArmorTrainingLight},
		[]content.ClassToolProficiency{mustValue(content.NewClassToolProficiencySpecific("Kit"))},
		mustValue(content.NewSkillChoiceAny(1)))))
	eq("spellcasting", c.Spellcasting(), content.ClassSpellcastingTrait(mustValue(content.NewPactSpellcastingTrait(ability.Charisma))))
	eq("columns", c.Columns(), []content.ClassColumn{
		mustValue(content.NewClassColumn("charges", "Charges", content.ColumnKindCount)), mustValue(content.NewClassColumn("die", "Die", content.ColumnKindDice)),
		mustValue(content.NewClassColumn("bonus", "Bonus", content.ColumnKindBonus)), mustValue(content.NewClassColumn("speed", "Speed", content.ColumnKindFeetBonus)),
	})
	row, _ := c.RowAt(2)
	eq("row", row, mustValue(content.NewProgressionRow(2, 2,
		[]content.RowFeature{mustValue(content.NewRowClassFeature("fixture_gamma", "one use"))},
		mustValue(content.NewPactSpellcastingRow(2, 2, 1, 1)),
		map[content.ClassColumnID]content.ColumnValue{
			"charges": mustValue(content.NewColumnCount(2)), "die": mustValue(content.NewColumnDice(1, 6)),
			"bonus": mustValue(content.NewColumnBonus(1)), "speed": mustValue(content.NewColumnFeetBonus(10)),
		})))
	row6, _ := c.RowAt(6)
	eq("subclass marker row", row6.Features(), []content.RowFeature{content.RowSubclassFeature{}})
	row20, _ := c.RowAt(20)
	eq("last pact row", row20.Spellcasting(), content.SpellcastingRow(mustValue(content.NewPactSpellcastingRow(2, 20, 1, 5))))
	features := c.Features()
	eq("spell fact", features[0].Facts(), []content.FeatureFact{mustValue(content.NewAlwaysPreparedSpells([]content.SpellGrant{
		mustValue(content.NewSpellGrant(1, spells("spell_a", "spell_b"))), mustValue(content.NewSpellGrant(5, spells("spell_c"))),
	}))})
	pre := mustValue(content.NewOptionPrerequisite(5, []content.FeatureOptionID{"o2"}, "A note."))
	eq("options", features[1].Options(), []content.FeatureOption{
		mustValue(content.NewFeatureOption("o1", "Opt o1", content.FeatureOptionFields{Text: "Option text.", Prerequisite: pre, PointCost: 2, Repeatable: true})),
		mustValue(content.NewFeatureOption("o2", "Opt o2", content.FeatureOptionFields{Text: "Option text."})),
	})
	eq("table", features[2].Tables(), []content.FeatureTable{mustValue(content.NewFeatureTable("T", []string{"A", "B"}, [][]content.TableCell{
		{mustValue(content.NewTableCellText("x")), mustValue(content.NewTableCellInt(1))}, {mustValue(content.NewTableCellText("y")), mustValue(content.NewTableCellInt(-2))},
	}))})
	eq("grants subclass", features[3].Facts(), []content.FeatureFact{content.GrantsSubclass{}})
	eq("grants feat", features[4].Facts(), []content.FeatureFact{mustValue(content.NewGrantsFeat(content.FeatGeneral))})
	eq("text", features[0].Text(), "Text.\nMore.")
	sub := mustSubclass(t, c)
	if sub.Features()[1].Level() != 6 || sub.Features()[0].ID() != "sub_a_f0" {
		t.Errorf("subclass features = %+v", sub.Features())
	}
	// A none value decodes to ColumnNone in a column of each kind.
	none := decodeClassFixture(t, mutate(columnClassJSON(), `"charges":2`, `"charges":"—"`))
	r, _ := none.RowAt(1)
	if v, _ := r.Value("charges"); v != content.ColumnValue(content.ColumnNone{}) {
		t.Errorf("none value = %v", v)
	}
}
