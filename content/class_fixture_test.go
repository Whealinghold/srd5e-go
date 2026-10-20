package content_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// The class fixtures are synthetic: made-up ids, names and texts shaped like a
// full caster (20 rows, ten feature entries, one subclass fact, no subclasses),
// never SRD data.

const classSpec = "doc/specs/content-classes.md"

type fixtureFeature struct {
	slug  string
	level int
	extra string
}

// fixtureFeatures are the ten class features of the Wizard-shaped fixture.
func fixtureFeatures() []fixtureFeature {
	return []fixtureFeature{
		{"alpha", 1, ""}, {"beta", 1, ""}, {"gamma", 2, ""},
		{"delta", 3, `,"facts":[{"kind":"grantsSubclass"}]`},
		{"asi", 4, `,"facts":[{"kind":"grantsFeat","category":"General"}]`},
		{"epsilon", 5, ""}, {"zeta", 7, ""}, {"eta", 18, ""},
		{"boon", 19, `,"facts":[{"kind":"grantsFeat","category":"Epic Boon"}]`}, {"theta", 20, ""},
	}
}

func featureJSON(id string, level int, extra string) string {
	return fmt.Sprintf(`{"id":"%s","name":"Name %s","level":%d,"text":"Text.\nMore."%s}`, id, id, level, extra)
}

// rowCell is the Class Features cell of a row of the fixture owned by owner:
// the entries of its level, the level 4 entry again at 8, 12 and 16, a
// subclass marker at 6, 10 and 14.
func rowCell(owner string, level int) string {
	var cells []string
	for _, f := range fixtureFeatures() {
		if f.level == level {
			cells = append(cells, `{"featureId":"`+owner+"_"+f.slug+`"}`)
		}
	}
	switch level {
	case 8, 12, 16:
		cells = append(cells, `{"featureId":"`+owner+`_asi"}`)
	case 6, 10, 14:
		cells = append(cells, `{"kind":"subclassFeature"}`)
	}
	return strings.Join(cells, ",")
}

// slotRowJSON is the spellcasting part of a slot row at level.
func slotRowJSON(level int) string {
	return fmt.Sprintf(`{"cantrips":3,"preparedSpells":%d,"spellSlots":[%d,0,0,0,0,0,0,0,0]}`, level, level)
}

// pactRowJSON is the spellcasting part of a Pact Magic row at level.
func pactRowJSON(level int) string {
	return fmt.Sprintf(`{"cantrips":2,"preparedSpells":%d,"spellSlots":1,"slotLevel":%d}`, level, 1+(level-1)/4)
}

// classRows returns the 20 rows of the fixture owned by owner; spell(level) is
// the spellcasting member ("" for none) and values the "values" object.
func classRows(owner string, spell func(int) string, values string) []string {
	rows := make([]string, 20)
	for l := 1; l <= 20; l++ {
		s := ""
		if spell != nil {
			s = `,"spellcasting":` + spell(l)
		}
		rows[l-1] = fmt.Sprintf(`{"level":%d,"proficiencyBonus":%d,"features":[%s]%s,"values":%s}`, l, 2+(l-1)/4, rowCell(owner, l), s, values)
	}
	return rows
}

func classFeaturesJSON(owner string) string {
	var fs []string
	for _, f := range fixtureFeatures() {
		fs = append(fs, featureJSON(owner+"_"+f.slug, f.level, f.extra))
	}
	return strings.Join(fs, ",")
}

const slotTrait = `{"kind":"slots","ability":"intelligence","cantrips":true}`

// buildClass assembles a class of the fixture shape from its varying parts.
func buildClass(owner, columns, trait string, rows []string, subclasses string) string {
	spell := ""
	if trait != "" {
		spell = `"spellcasting":` + trait + `,`
	}
	return `{"id":"` + owner + `","name":"Fixture","primaryAbility":{"kind":"allOf","abilities":["intelligence"]},"hitPointDie":"D6",` +
		`"savingThrowProficiencies":["intelligence","wisdom"],"skillChoice":{"picks":2,"options":{"kind":"listed","skills":["Arcana","History","Insight"]}},` +
		`"weaponProficiencies":[{"category":"Simple"}],"toolProficiencies":[],"armorTraining":[],` +
		`"startingEquipment":{"A":{"items":[{"name":"Staff","quantity":1}],"gold":5},"B":{"items":[],"gold":50}},` +
		`"multiclassGrants":{"weaponProficiencies":[],"armorTraining":[],"toolProficiencies":[]},` + spell +
		`"columns":[` + columns + `],"progression":[` + strings.Join(rows, ",") + `],"features":[` + classFeaturesJSON(owner) + `],"subclasses":[` + subclasses + `]}`
}

// ownerClassJSON is the full-caster fixture owned by owner: no columns, slot
// spellcasting with cantrips, the given subclasses.
func ownerClassJSON(owner, subclasses string) string {
	return buildClass(owner, "", slotTrait, classRows(owner, slotRowJSON, `{}`), subclasses)
}

// wizardClassJSON is the full-caster fixture with no subclasses.
func wizardClassJSON() string { return ownerClassJSON("fixture", "") }

// pactClassJSON is a Pact Magic fixture.
func pactClassJSON() string {
	return buildClass("fixture", "", `{"kind":"pact","ability":"charisma"}`, classRows("fixture", pactRowJSON, `{}`), "")
}

// plainClassJSON is a fixture with no spellcasting at all.
func plainClassJSON() string {
	return buildClass("fixture", "", "", classRows("fixture", nil, `{}`), "")
}

const columnDecls = `{"id":"charges","header":"Charges","kind":"count"},{"id":"die","header":"Die","kind":"dice"},{"id":"bonus","header":"Bonus","kind":"bonus"},{"id":"speed","header":"Speed","kind":"feetBonus"}`

const columnValues = `{"charges":2,"die":{"count":1,"sides":6},"bonus":1,"speed":10}`

// columnClassJSON is a fixture with one column of each kind.
func columnClassJSON() string {
	return buildClass("fixture", columnDecls, "", classRows("fixture", nil, columnValues), "")
}

// subclassJSON is a subclass whose features sit at the given levels.
func subclassJSON(id string, levels ...int) string {
	var fs []string
	for i, l := range levels {
		fs = append(fs, featureJSON(id+"_f"+strconv.Itoa(i), l, ""))
	}
	return `{"id":"` + id + `","name":"Sub ` + id + `","features":[` + strings.Join(fs, ",") + `]}`
}

func classDoc(class string) string { return wrapDoc("classes", class) }

// decodeClassFixture decodes a document holding one class and returns it.
func decodeClassFixture(t *testing.T, class string) content.ClassDefinition {
	t.Helper()
	set := decodeSet(t, classDoc(class))
	for _, e := range set.Entries() {
		if c, ok := e.(content.ClassDefinition); ok {
			return c
		}
	}
	t.Fatalf("no class in %s", class)
	return content.ClassDefinition{}
}

// over returns ClassMaxCount+1. A 32-bit int cannot hold it: there it wraps to a
// negative number, which every count rejects as well, so the tests that use it
// pass at both widths.
func over() int {
	v := int64(content.ClassMaxCount) + 1
	return int(v)
}
