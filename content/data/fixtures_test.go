package data_test

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// The oracle of the data pins: SRD 5.2.1 text transcribed from the PDF, kept
// apart from the data files under testdata/srd (content-thin-slice.md section 1,
// "Oracle"; content-licensing-and-notice.md FR-033).
//
//go:embed testdata/srd/thin_slice/*.json
var oracleFS embed.FS

// readOracle decodes testdata/srd/thin_slice/<name>.json into a T.
func readOracle[T any](t *testing.T, name string) T {
	t.Helper()
	raw, err := oracleFS.ReadFile("testdata/srd/thin_slice/" + name + ".json")
	if err != nil {
		t.Fatalf("oracle %s: %v", name, err)
	}
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("oracle %s: %v", name, err)
	}
	return v
}

type citationOracle struct {
	Document string `json:"document"`
	Entries  []struct {
		Kind    string `json:"kind"`
		ID      string `json:"id"`
		Section string `json:"section"`
		Page    int    `json:"page"`
	} `json:"entries"`
}

type traitOracle struct {
	Name    string   `json:"name"`
	Phrases []string `json:"phrases"`
}

type speciesOracle struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	CreatureType string        `json:"creatureType"`
	Sizes        []string      `json:"sizes"`
	SizeText     string        `json:"sizeText"`
	Speed        int           `json:"speed"`
	Traits       []traitOracle `json:"traits"`
}

type speciesOracleFile struct {
	Species           []speciesOracle `json:"species"`
	DraconicAncestors []struct {
		Dragon     string `json:"dragon"`
		DamageType string `json:"damageType"`
	} `json:"draconicAncestors"`
	BreathWeaponDamage []struct {
		Level int    `json:"level"`
		Dice  string `json:"dice"`
	} `json:"breathWeaponDamage"`
	Lineages []struct {
		Name   string   `json:"name"`
		Level1 string   `json:"level1"`
		Level3 string   `json:"level3"`
		Level5 string   `json:"level5"`
		Spells []string `json:"spells"`
	} `json:"lineages"`
	Choices []struct {
		Label      string `json:"label"`
		Species    string `json:"species"`
		Structured bool   `json:"structured"`
		Phrase     string `json:"phrase"`
	} `json:"choices"`
}

type spellOracle struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Level         int      `json:"level"`
	School        string   `json:"school"`
	Classes       []string `json:"classes"`
	CastingTime   string   `json:"castingTime"`
	Range         string   `json:"range"`
	Components    string   `json:"components"`
	Duration      string   `json:"duration"`
	Scaling       string   `json:"scaling"`
	ScalingPhrase string   `json:"scalingPhrase"`
	Phrases       []string `json:"phrases"`
	Entries       []string `json:"entries"`
	Absent        []string `json:"absent"`
}

type spellOracleFile struct {
	Spells []spellOracle `json:"spells"`
}

type wizardOracle struct {
	PrimaryAbility string   `json:"primaryAbility"`
	HitPointDie    string   `json:"hitPointDie"`
	SavingThrows   []string `json:"savingThrows"`
	Skills         struct {
		Picks   int      `json:"picks"`
		Options []string `json:"options"`
	} `json:"skills"`
	Weapons             []string        `json:"weapons"`
	ArmorTraining       []string        `json:"armorTraining"`
	EquipmentA          equipmentOracle `json:"equipmentA"`
	EquipmentB          equipmentOracle `json:"equipmentB"`
	SpellcastingAbility string          `json:"spellcastingAbility"`
	Features            []struct {
		Name  string `json:"name"`
		Level int    `json:"level"`
		Fact  string `json:"fact"`
	} `json:"features"`
	SpellcastingPhrases []string `json:"spellcastingPhrases"`
	PrintedRows         []string `json:"printedRows"`
}

// bannedTerms are the whole words no entry text may contain (thin-slice FR-008).
func bannedTerms() []string { return []string{"race", "races", "subrace", "subraces"} }

// containsWord reports whether s holds term as a whole word, ignoring case.
func containsWord(s, term string) bool {
	s = strings.ToLower(s)
	for i := 0; ; {
		j := strings.Index(s[i:], term)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(term)
		before := start == 0 || !isLetter(s[start-1])
		after := end == len(s) || !isLetter(s[end])
		if before && after {
			return true
		}
		i = start + 1
	}
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' }

// newSlice builds the bundled source and fails the test on a bad bundle.
func newSlice(t *testing.T) *data.BundledSource {
	t.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		t.Fatalf("NewBundledSource() error = %v", err)
	}
	return src
}

// lookup returns the entry of key from src, or fails.
func lookup(t *testing.T, src content.Source, kind content.Kind, id string) content.Entry {
	t.Helper()
	res, err := src.Lookup(context.Background(), content.NewKey(kind, content.MustID(id)))
	if err != nil {
		t.Fatalf("Lookup(%s %s) error = %v", kind, id, err)
	}
	found, ok := res.(content.LookupFound)
	if !ok {
		t.Fatalf("Lookup(%s %s) = %T, want LookupFound", kind, id, res)
	}
	return found.Entry
}

func speciesOf(t *testing.T, src content.Source, id string) content.Species {
	t.Helper()
	s, ok := lookup(t, src, content.KindSpecies, id).(content.Species)
	if !ok {
		t.Fatalf("species %s is not a content.Species", id)
	}
	return s
}

func spellOf(t *testing.T, src content.Source, id string) content.Spell {
	t.Helper()
	s, ok := lookup(t, src, content.KindSpell, id).(content.Spell)
	if !ok {
		t.Fatalf("spell %s is not a content.Spell", id)
	}
	return s
}

func wizardOf(t *testing.T, src content.Source) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, src, content.KindCharacterClass, "wizard").(content.ClassDefinition)
	if !ok {
		t.Fatal("wizard is not a content.ClassDefinition")
	}
	return c
}

// blockTexts returns every string of the blocks: entry names and texts,
// paragraph texts.
func blockTexts(t *testing.T, blocks []content.TextBlock) []string {
	t.Helper()
	var out []string
	for _, b := range blocks {
		switch v := b.(type) {
		case content.TextParagraph:
			out = append(out, v.Text())
		case content.TextEntry:
			out = append(out, v.Name(), v.Text())
		default:
			t.Fatalf("unexpected block %T in a slice spell", b)
		}
	}
	return out
}

// speciesTexts returns every text of a species: size text, traits and the
// text of the lineage options.
func speciesTexts(s content.Species) []string {
	out := []string{s.Name(), s.SizeText()}
	for _, tr := range s.Traits() {
		out = append(out, tr.Name(), tr.Text())
	}
	for _, c := range s.Choices() {
		for _, o := range c.Options() {
			out = append(out, o.Name())
			switch v := o.(type) {
			case content.LineageOption:
				for _, b := range v.LevelBenefits() {
					out = append(out, b.Text())
				}
			case content.TextOption:
				out = append(out, v.Text())
			}
		}
	}
	return out
}

func spellTexts(t *testing.T, s content.Spell) []string {
	t.Helper()
	out := []string{s.Name()}
	out = append(out, blockTexts(t, s.Description())...)
	out = append(out, blockTexts(t, s.HigherLevel())...)
	out = append(out, blockTexts(t, s.CantripUpgrade())...)
	if m, ok := s.Components().Material(); ok {
		out = append(out, m.Description())
	}
	return out
}

func wizardTexts(c content.ClassDefinition) []string {
	out := []string{c.Name()}
	for _, f := range c.Features() {
		out = append(out, f.Name(), f.Text())
	}
	return out
}

// sliceIDs are the 17 entry ids of the slice by kind (thin-slice FR-001 to FR-003).
func sliceIDs() map[content.Kind][]string {
	return map[content.Kind][]string{
		content.KindSpecies:        {"dragonborn", "dwarf", "elf", "human"},
		content.KindCharacterClass: {"wizard"},
		content.KindSpell: {"dancing_lights", "darkness", "detect_magic", "druidcraft", "faerie_fire", "fire_bolt", "fireball",
			"longstrider", "magic_missile", "misty_step", "pass_without_trace", "prestidigitation"},
	}
}

func idsOf(entries []content.Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.Key().ID().String()
	}
	return out
}

func describe(format string, args ...any) string { return fmt.Sprintf(format, args...) }

// equipmentOracle is a printed starting-equipment option: items as [name, quantity] pairs and GP.
type equipmentOracle struct {
	Items [][]any `json:"items"`
	Gold  int     `json:"gold"`
}
