package character_test

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// clcDef is the bundled Cleric ClassDefinition (content/data/class_cleric.json).
func clcDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatal(err)
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("cleric")))
	if err != nil {
		tb.Fatal(err)
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

func clcProgression(tb testing.TB) character.ClericProgression {
	tb.Helper()
	return must(character.NewClericProgression(clcDef(tb)))
}

// clcLifeDomain is the bundled Life Domain subclass.
func clcLifeDomain(tb testing.TB) *content.SubclassDefinition {
	tb.Helper()
	sub := clcDef(tb).Subclasses()[0]
	return &sub
}

// clcChar is a character with the given class levels, enough Experience Points for level 20, flat scores of 10 and a stored maximum of 200.
func clcChar(t *testing.T, entries ...character.ClassLevel) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), entries...)
}

// clcCleric is a Cleric of level n and nothing else.
func clcCleric(t *testing.T, n int) character.Character {
	t.Helper()
	return clcChar(t, cl("cleric", n))
}

// clcWith returns c holding the ClericResources and, when given, the ClericChoices.
func clcWith(t *testing.T, c character.Character, res character.ClericResources) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.ClericResourcesKey(), res))
}

func clcWithChoices(t *testing.T, c character.Character, ch character.ClericChoices) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.ClericChoicesKey(), ch))
}

// clcResourcesOf reads the ClericResources c stores.
func clcResourcesOf(t *testing.T, c character.Character) character.ClericResources {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.ClericResourcesKey())
	if err != nil || !ok {
		t.Fatalf("ClericResources = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// clcUses builds resources of maximum with remaining uses and Divine Intervention available.
func clcUses(maximum, remaining int) character.ClericResources {
	return must(character.NewClericResources(maximum, remaining, character.DivineInterventionAvailable{}))
}

// clcStrikes is a ClericChoices holding the Blessed Strikes option (and the Protector order).
func clcStrikes(o character.BlessedStrikesOption) character.ClericChoices {
	return must(character.NewClericChoices(character.DivineOrderProtector, "", o))
}

// clcSpell decodes a synthetic spell with the id, level, class list and raw casting times JSON (a Magic action when empty).
func clcSpell(tb testing.TB, id string, level int, classes []string, times string) content.Spell {
	tb.Helper()
	if times == "" {
		times = "[" + timeAction + "]"
	}
	quoted := make([]string, len(classes))
	for i, c := range classes {
		quoted[i] = `"` + c + `"`
	}
	doc := fmt.Sprintf(`{"schemaVersion":1,"spells":[{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":[%s],"castingTimes":%s,"ritual":false,"range":{"kind":"self"},"components":{"verbal":true,"somatic":false},"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}]}`,
		id, level, strings.Join(quoted, ","), times)
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		tb.Fatalf("fixture spell: %v", err)
	}
	return set.Entries()[0].(content.Spell)
}

// clcClericSpell is a Cleric-list spell with a Magic action casting time.
func clcClericSpell(tb testing.TB, id string, level int) content.Spell {
	tb.Helper()
	return clcSpell(tb, id, level, []string{"cleric", "wizard"}, "")
}

// clcDefFrom decodes the bundled Cleric file after mutate changed its text.
func clcDefFrom(tb testing.TB, mutate func(string) string) content.ClassDefinition {
	tb.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_cleric.json")
	if err != nil {
		tb.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode([]byte(mutate(string(raw))))
	if err != nil {
		tb.Fatalf("mutated Cleric file: %v", err)
	}
	return set.Entries()[0].(content.ClassDefinition)
}

func contentSpellID(s string) content.SpellID { return content.SpellID(s) }
