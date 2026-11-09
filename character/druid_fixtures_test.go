package character_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// druDef is the bundled Druid ClassDefinition (content/data/class_druid.json).
func druDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatal(err)
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("druid")))
	if err != nil {
		tb.Fatal(err)
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

// druTables are the tables built from the bundled Druid class data.
func druTables(tb testing.TB) character.DruidFeatureTables {
	tb.Helper()
	t, err := character.DruidTablesFromClass(druDef(tb))
	if err != nil {
		tb.Fatal(err)
	}
	return t
}

var druTestID = character.MustIdentifier("druid-1")

// druFacts are the facts of a Druid of the level; circle gives the Circle of the Land. The Proficiency Bonus is of a total level
// of the Druid level, which is not the case in the multiclass tests that pass their own.
func druFacts(tb testing.TB, level int, circle bool) character.DruidCharacterFacts {
	tb.Helper()
	return druFactsBonus(tb, level, circle, character.ProficiencyBonusFor(character.MustLevel(level)))
}

func druFactsBonus(tb testing.TB, level int, circle bool, bonus character.ProficiencyBonus) character.DruidCharacterFacts {
	tb.Helper()
	sub := content.SubclassID("")
	if circle {
		sub = character.CircleOfTheLandSubclassID
	}
	return must(character.NewDruidCharacterFacts(druTestID, level, sub, bonus))
}

// druState restores a state with the fields for the Druid of the level.
func druState(tb testing.TB, level int, circle bool, f character.DruidStateFields) character.DruidFeatureState {
	tb.Helper()
	return must(character.NewDruidFeatureState(f, druFacts(tb, level, circle), druTables(tb)))
}

// druForm is a Beast form with the monster id, Challenge Rating text and Fly Speed.
func druForm(id, cr string, fly bool) character.BeastFormFacts {
	return must(character.NewBeastFormFacts(content.MonsterID(id), content.CreatureTypeBeast, content.MustChallengeRating(cr), fly))
}

func druIDs(ids ...string) []content.MonsterID {
	out := make([]content.MonsterID, len(ids))
	for i, id := range ids {
		out[i] = content.MonsterID(id)
	}
	return out
}

// druSlots is the spell slot state of a Druid of the level with the given counts of expended slots per slot level.
func druSlots(t *testing.T, level int, expended map[int]int) character.SpellcastingState {
	t.Helper()
	s := castingState(t, classLevelsOf(t, "druid", level), spellCatalogue(t))
	for l := 1; l <= 9; l++ {
		for range expended[l] {
			r, ok := s.ExpendSlot(character.PoolSpellcasting, slotLevel(l)).(character.SlotExpenditureExpended)
			if !ok { // expend up to the slots there are
				break
			}
			s = r.State()
		}
	}
	return s
}

// druSpell decodes a synthetic spell with the id, level, class list, range JSON and components JSON.
func druSpell(tb testing.TB, id string, level int, classes []string, rangeJSON, components string) content.Spell {
	tb.Helper()
	if rangeJSON == "" {
		rangeJSON = `{"kind":"self"}`
	}
	if components == "" {
		components = verbalOnly()
	}
	quoted := make([]string, len(classes))
	for i, c := range classes {
		quoted[i] = `"` + c + `"`
	}
	doc := fmt.Sprintf(`{"schemaVersion":1,"spells":[{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":[%s],"castingTimes":[%s],"ritual":false,"range":%s,"components":%s,"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}]}`,
		id, level, strings.Join(quoted, ","), timeAction, rangeJSON, components)
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		tb.Fatalf("fixture spell: %v", err)
	}
	return set.Entries()[0].(content.Spell)
}

// druMonster decodes a synthetic monster with the type, Challenge Rating and speeds JSON.
func druMonster(tb testing.TB, kind, cr, speeds string) content.Monster {
	tb.Helper()
	const abilities = `"abilities":{"str":{"score":10,"save":0},"dex":{"score":10,"save":0},"con":{"score":10,"save":0},"int":{"score":10,"save":0},"wis":{"score":10,"save":0},"cha":{"score":10,"save":0}}`
	doc := fmt.Sprintf(`{"schemaVersion":1,"monsters":[{"id":"fixture_beast","name":"Fixture Beast","source":{"document":"Fixture","section":"Monsters A-Z","page":258},`+
		`"sizes":["Medium"],"type":%q,"alignment":"Unaligned","armorClass":12,"initiative":{"modifier":1,"score":11},`+
		`"hitPoints":{"average":9,"count":2,"faces":8,"modifier":0},"speeds":%s,%s,`+
		`"senses":{"passivePerception":10},"languages":{},"challengeRating":%q,"experiencePoints":0}]}`, kind, speeds, abilities, cr)
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		tb.Fatalf("fixture monster: %v", err)
	}
	return set.Entries()[0].(content.Monster)
}

// druDefFrom decodes the bundled Druid file after mutate changed its JSON document (the first class, as a generic map).
func druDefFrom(tb testing.TB, mutate func(class map[string]any)) content.ClassDefinition {
	tb.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_druid.json")
	if err != nil {
		tb.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		tb.Fatal(err)
	}
	mutate(doc["classes"].([]any)[0].(map[string]any))
	out, err := json.Marshal(doc)
	if err != nil {
		tb.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode(out)
	if err != nil {
		tb.Fatalf("mutated Druid file: %v", err)
	}
	return set.Entries()[0].(content.ClassDefinition)
}

// druFeature returns the feature object of the class (or its subclass) with the id.
func druFeature(tb testing.TB, class map[string]any, id string) map[string]any {
	tb.Helper()
	lists := [][]any{class["features"].([]any), class["subclasses"].([]any)[0].(map[string]any)["features"].([]any)}
	for _, list := range lists {
		for _, f := range list {
			if m := f.(map[string]any); m["id"] == id {
				return m
			}
		}
	}
	tb.Fatalf("no feature %s", id)
	return nil
}
