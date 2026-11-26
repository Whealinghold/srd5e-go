package scenarios_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// grpD2Spell is a synthetic spell of the level on the given class lists, with a Magic action casting time (the bundle holds no Cleric cantrips).
func grpD2Spell(t *testing.T, id string, level int, classes ...string) content.Spell {
	t.Helper()
	quoted := make([]string, len(classes))
	for i, c := range classes {
		quoted[i] = `"` + c + `"`
	}
	doc := fmt.Sprintf(`{"schemaVersion":1,"spells":[{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":[%s],"castingTimes":[{"kind":"magicAction"}],"ritual":false,"range":{"kind":"self"},"components":{"verbal":true,"somatic":false},"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}]}`,
		id, level, strings.Join(quoted, ","))
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		t.Fatalf("fixture spell: %v", err)
	}
	return set.Entries()[0].(content.Spell)
}

// grpD2Feat is a synthetic feat of the category; the Fighting Style feats carry the Fighting Style Feature prerequisite as the SRD prints it.
func grpD2Feat(id string, cat content.FeatCategory, repeatable bool, prereqs ...content.Prerequisite) content.Feat {
	f := content.FeatFields{Category: cat, Benefits: []content.FeatBenefit{content.MustFeatBenefit("Benefit", "A benefit.")}, Repeatable: repeatable}
	if repeatable {
		f.RepeatableText = "You can take this feat more than once."
	}
	if len(prereqs) > 0 {
		f.Prerequisites = prereqs
	}
	return content.MustFeat(content.FeatID(id), id, content.SourceCitation{}, f)
}

func grpD2StyleFeat(id string) content.Feat {
	return grpD2Feat(id, content.FeatFightingStyle, false, must2(content.NewFeaturePrerequisite("Fighting Style Feature")))
}
