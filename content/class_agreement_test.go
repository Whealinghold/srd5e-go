package content_test

import (
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-classes.md FR-007 and doc/specs/content-schema-and-decoder.md FR-094 - for
// every row of the part tables the decoder and the constructor give the same
// verdict: a part the constructor builds decodes, a part it rejects is a
// problem.
// Path: per case.
func TestAgreementClassParts(t *testing.T) {
	t.Parallel()
	str, dex := ability.Strength, ability.Dexterity
	var cases []struct {
		name     string
		base     string
		old, new string
		ctor     error
	}
	add := func(name, base, old, repl string, ctor error) {
		cases = append(cases, struct {
			name     string
			base     string
			old, new string
			ctor     error
		}{name, base, old, repl, ctor})
	}
	w, cw := wizardClassJSON(), columnClassJSON()
	prim := `{"kind":"allOf","abilities":["intelligence"]}`
	for _, c := range []struct {
		json string
		err  error
	}{
		{`{"kind":"allOf","abilities":["strength"]}`, errOf(content.NewAllOf(str))},
		{`{"kind":"allOf","abilities":[]}`, errOf(content.NewAllOf())},
		{`{"kind":"allOf","abilities":["strength","strength"]}`, errOf(content.NewAllOf(str, str))},
		{`{"kind":"anyOf","abilities":["strength","dexterity"]}`, errOf(content.NewAnyOf(str, dex))},
		{`{"kind":"anyOf","abilities":["strength"]}`, errOf(content.NewAnyOf(str))},
		{`{"kind":"anyOf","abilities":["strength","strength"]}`, errOf(content.NewAnyOf(str, str))},
		{`{"kind":"anyOf","abilities":["strength","dexterity","constitution","intelligence","wisdom","charisma"]}`, errOf(content.NewAnyOf(ability.All()...))},
	} {
		add("primary "+c.json, w, prim, c.json, c.err)
	}
	for _, c := range []struct {
		json string
		err  error
	}{
		{`{"picks":1,"options":{"kind":"any"}}`, errOf(content.NewSkillChoiceAny(1))},
		{`{"picks":0,"options":{"kind":"any"}}`, errOf(content.NewSkillChoiceAny(0))},
		{`{"picks":1,"options":{"kind":"listed","skills":["Arcana"]}}`, errOf(content.NewSkillChoiceListed(1, []content.Skill{content.SkillArcana}))},
		{`{"picks":2,"options":{"kind":"listed","skills":["Arcana"]}}`, errOf(content.NewSkillChoiceListed(2, []content.Skill{content.SkillArcana}))},
		{`{"picks":1,"options":{"kind":"listed","skills":["Arcana","Arcana"]}}`, errOf(content.NewSkillChoiceListed(1, []content.Skill{content.SkillArcana, content.SkillArcana}))},
		{`{"picks":1,"options":{"kind":"listed","skills":[]}}`, errOf(content.NewSkillChoiceListed(1, nil))},
	} {
		add("skills "+c.json, w, `{"picks":2,"options":{"kind":"listed","skills":["Arcana","History","Insight"]}}`, c.json, c.err)
	}
	for _, d := range []string{"D6", "D8", "D10", "D12", "D4", "d6", "D20", ""} {
		_, err := content.ParseHitPointDie(d)
		add("die "+d, w, `"D6"`, `"`+d+`"`, err)
	}
	for _, c := range []struct {
		json string
		err  error
	}{
		{`{"category":"Martial","requiresAnyProperty":["Light"]}`, errOf(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{content.WeaponPropertyLight}))},
		{`{"category":"Martial","requiresAnyProperty":[]}`, errOf(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{}))},
		{`{"category":"Martial","requiresAnyProperty":["Light","Light"]}`, errOf(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{content.WeaponPropertyLight, content.WeaponPropertyLight}))},
		{`{"category":"Martial"}`, errOf(content.NewWeaponProficiency(content.WeaponTrainingMartial, nil))},
	} {
		add("weapon "+c.json, w, `{"category":"Simple"}`, c.json, c.err)
	}
	for _, c := range []struct {
		json string
		err  error
	}{
		{`2`, errOf(content.NewColumnCount(2))}, {`0`, errOf(content.NewColumnCount(0))}, {`-1`, errOf(content.NewColumnCount(-1))},
		{`2147483647`, errOf(content.NewColumnCount(content.ClassMaxCount))}, {`2147483648`, errOf(content.NewColumnCount(over()))},
	} {
		add("count "+c.json, cw, `"charges":2`, `"charges":`+c.json, c.err)
	}
	for _, c := range []struct {
		json string
		err  error
	}{
		{`{"count":1,"sides":6}`, errOf(content.NewColumnDice(1, 6))}, {`{"count":0,"sides":6}`, errOf(content.NewColumnDice(0, 6))},
		{`{"count":1,"sides":1}`, errOf(content.NewColumnDice(1, 1))}, {`{"count":1,"sides":2}`, errOf(content.NewColumnDice(1, 2))},
		{`{"count":1,"sides":1000}`, errOf(content.NewColumnDice(1, 1000))}, {`{"count":1,"sides":1001}`, errOf(content.NewColumnDice(1, 1001))},
	} {
		add("dice "+c.json, cw, `"die":{"count":1,"sides":6}`, `"die":`+c.json, c.err)
	}
	for _, n := range []string{"0", "1", "-1"} {
		v, _ := strconv.Atoi(n)
		add("bonus "+n, cw, `"bonus":1`, `"bonus":`+n, errOf(content.NewColumnBonus(v)))
		add("feet "+n, cw, `"speed":10`, `"speed":`+n, errOf(content.NewColumnFeetBonus(v)))
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := len(decodeProblems(classDoc(mutate(c.base, c.old, c.new)))) == 0
			if want := c.ctor == nil; got != want {
				t.Errorf("decoder accepted = %v, constructor accepted = %v [%s FR-007, per case]", got, want, classSpec)
			}
		})
	}
}
