package character_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// advSpec describes a synthetic class built from the bundled Wizard document:
// its id, Hit Point Die, Primary Ability, multiclass grants and subclasses. The
// Features table (Ability Score Improvement at 4, 8, 12 and 16, Epic Boon at 19,
// the subclass at 3) is the Wizard's, so every fixture class has the same shape.
type advSpec struct {
	id        string
	die       string
	kind      string      // "allOf" or "anyOf"
	abilities []string    // lower-case ability names
	grants    string      // the multiclassGrants JSON object, "" for none
	subclass  []string    // subclass ids
	weapons   string      // the core weaponProficiencies JSON array, "" for Simple weapons
	armor     string      // the core armorTraining JSON array, "" for none
	replace   [][2]string // literal text edits applied last
}

var (
	primaryPattern    = regexp.MustCompile(`(?s)"primaryAbility": \{.*?\}`)
	weaponsPattern    = regexp.MustCompile(`(?s)"weaponProficiencies": \[\n        \{.*?\n      \]`)
	subclassesPattern = regexp.MustCompile(`(?s)"subclasses": \[.*?\n      \]`)
	grantsPattern     = regexp.MustCompile(`(?s)"multiclassGrants": \{.*?\n      \}`)
)

func quoted(ss []string) string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = `"` + s + `"`
	}
	return strings.Join(out, ",")
}

// advClass decodes the fixture class of spec.
func advClass(t *testing.T, s advSpec) content.ClassDefinition {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "class_wizard.json")
	if err != nil {
		t.Fatal(err)
	}
	// The bundled Wizard now carries the Evoker; the fixture class starts from none and adds its own.
	text := subclassesPattern.ReplaceAllString(string(raw), `"subclasses": []`)
	text = idPattern.ReplaceAllString(text, `"`+s.id+`$1"`)
	text = diePattern.ReplaceAllString(text, `"hitPointDie": "`+s.die+`"`)
	text = primaryPattern.ReplaceAllString(text, `"primaryAbility": {"kind":"`+s.kind+`","abilities":[`+quoted(s.abilities)+`]}`)
	if s.grants != "" {
		text = grantsPattern.ReplaceAllString(text, `"multiclassGrants": `+s.grants)
	}
	if s.weapons != "" {
		text = weaponsPattern.ReplaceAllLiteralString(text, `"weaponProficiencies": `+s.weapons)
	}
	if s.armor != "" {
		text = strings.Replace(text, `"armorTraining": []`, `"armorTraining": `+s.armor, 1)
	}
	var subs []string
	for _, sub := range s.subclass {
		subs = append(subs, `{"id":"`+sub+`","name":"Sub `+sub+`","features":[`+
			`{"id":"`+sub+`_first","name":"First","level":3,"text":"Text."},`+
			`{"id":"`+sub+`_second","name":"Second","level":6,"text":"Text."}]}`)
	}
	for _, r := range s.replace {
		text = strings.ReplaceAll(text, r[0], r[1])
	}
	text = strings.Replace(text, `"subclasses": []`, `"subclasses": [`+strings.Join(subs, ",")+`]`, 1)
	set, err := content.JSONDecoder{}.Decode([]byte(text))
	if err != nil {
		t.Fatalf("fixture class %s: %v", s.id, err)
	}
	for _, e := range set.Entries() {
		if def, ok := e.(content.ClassDefinition); ok {
			return def
		}
	}
	t.Fatalf("fixture class %s: no class decoded", s.id)
	return content.ClassDefinition{}
}

// The sample classes of the level advancement tables. Their Primary Abilities
// follow SRD 5.2.1, Class Overview table, p. 19.
func advWizard(t *testing.T, subclasses ...string) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "wizard", die: "D6", kind: "allOf", abilities: []string{"intelligence"}, subclass: subclasses})
}

func advFighter(t *testing.T, subclasses ...string) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "fighter", die: "D10", kind: "anyOf", abilities: []string{"strength", "dexterity"}, subclass: subclasses})
}

func advBarbarian(t *testing.T) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "barbarian", die: "D12", kind: "allOf", abilities: []string{"strength"}})
}

func advDruid(t *testing.T) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "druid", die: "D8", kind: "allOf", abilities: []string{"wisdom"}})
}

func advMonk(t *testing.T) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "monk", die: "D8", kind: "allOf", abilities: []string{"dexterity", "wisdom"}})
}

func advCleric(t *testing.T, subclasses ...string) content.ClassDefinition {
	t.Helper()
	return advClass(t, advSpec{id: "cleric", die: "D8", kind: "allOf", abilities: []string{"wisdom"}, subclass: subclasses})
}

// requirementsOf returns the Primary Abilities of the given classes.
func requirementsOf(defs ...content.ClassDefinition) map[content.ClassID]content.AbilityRequirement {
	m := map[content.ClassID]content.AbilityRequirement{}
	for _, d := range defs {
		m[d.ID()] = d.PrimaryAbility()
	}
	return m
}

func previewOf(t *testing.T, c character.Character, class content.ClassDefinition, current ...content.ClassDefinition) character.LevelAdvancementPreview {
	t.Helper()
	p, err := character.LevelAdvancementPreviewer{}.Preview(c, class, requirementsOf(current...))
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	return p
}

// holding returns c with the subclass of class stored.
func holding(t *testing.T, c character.Character, class, subclass string) character.Character {
	t.Helper()
	key, err := character.SubclassStateKey{}.ForClass(content.ClassID(class))
	if err != nil {
		t.Fatal(err)
	}
	next, err := character.WithFeatureState(c, key, must(character.NewChosenSubclass(content.SubclassID(subclass))))
	if err != nil {
		t.Fatal(err)
	}
	return next
}
