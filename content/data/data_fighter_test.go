package data_test

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Fighter oracle is content/testdata/srd/classes/oracle.json, written by hand from the
// printed pages and never generated from content/data (content-classes-data.md FR-093).

type fighterOracleFile struct {
	Classes []fighterOracle `json:"classes"`
}

type oracleTable struct {
	Name    string     `json:"name"`
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

type oracleFeature struct {
	ID      string   `json:"id"`
	Options []string `json:"options"`
	Level   int      `json:"level"`
	Name    string   `json:"name"`
	Fact    string   `json:"fact"`
	Phrases []string `json:"phrases"`
	// Tables are the tables printed inside the feature; a cell is its printed text.
	Tables []oracleTable `json:"tables"`
}

type fighterOracle struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	ClassPage         int    `json:"classPage"`
	FeaturesTablePage int    `json:"featuresTablePage"`
	Subclass          struct {
		ID, Name, Section string
		Page              int
	} `json:"subclass"`
	PrimaryAbility struct {
		Kind      string   `json:"kind"`
		Abilities []string `json:"abilities"`
	} `json:"primaryAbility"`
	HitPointDie  string   `json:"hitPointDie"`
	SavingThrows []string `json:"savingThrows"`
	Skills       struct {
		Kind    string   `json:"kind"`
		Picks   int      `json:"picks"`
		Options []string `json:"options"`
	} `json:"skills"`
	Weapons        []string          `json:"weapons"`
	WeaponProperty map[string]string `json:"weaponProperty"`
	Tools          []string          `json:"tools"`
	ArmorTraining  []string          `json:"armorTraining"`
	Equipment      map[string]struct {
		Items [][2]any `json:"items"`
		Gold  int      `json:"gold"`
	} `json:"equipment"`
	Multiclass struct {
		Weapons       []string `json:"weapons"`
		ArmorTraining []string `json:"armorTraining"`
		Tools         []string `json:"tools"`
		Skills        string   `json:"skills"`
	} `json:"multiclass"`
	Spellcasting string                              `json:"spellcasting"`
	Columns      []struct{ ID, Header, Kind string } `json:"columns"`
	Rows         []struct {
		Level               int      `json:"level"`
		ProficiencyBonus    int      `json:"proficiencyBonus"`
		ClassFeatures       []string `json:"classFeatures"`
		SecondWind          int      `json:"secondWind"`
		BardicDie           string   `json:"bardicDie"`
		ChannelDivinity     string   `json:"channelDivinity"`
		FavoredEnemy        int      `json:"favoredEnemy"`
		WildShape           string   `json:"wildShape"`
		Rages               int      `json:"rages"`
		RageDamage          int      `json:"rageDamage"`
		WeaponMastery       int      `json:"weaponMastery"`
		MartialArts         string   `json:"martialArts"`
		SneakAttack         string   `json:"sneakAttack"`
		FocusPoints         int      `json:"focusPoints"`
		SorceryPoints       int      `json:"sorceryPoints"`
		UnarmoredMovement   int      `json:"unarmoredMovement"`
		Cantrips            int      `json:"cantrips"`
		PreparedSpells      int      `json:"preparedSpells"`
		SpellSlots          [9]int   `json:"spellSlots"`
		EldritchInvocations int      `json:"eldritchInvocations"`
		PactSlots           int      `json:"pactSlots"`
		SlotLevel           int      `json:"slotLevel"`
	} `json:"rows"`
	SubclassFeatureRows []int `json:"subclassFeatureRows"`
	OptionTexts         map[string][]struct {
		ID      string   `json:"id"`
		Phrases []string `json:"phrases"`
	} `json:"optionTexts"`
	// Invocations are the Warlock Eldritch Invocation options with their printed prerequisites.
	Invocations []struct {
		ID, Name, Note string
		MinClassLevel  int      `json:"minClassLevel"`
		Requires       []string `json:"requires"`
		Repeatable     bool     `json:"repeatable"`
	} `json:"invocations"`
	Features         []oracleFeature `json:"features"`
	SubclassFeatures []oracleFeature `json:"subclassFeatures"`
	// LandSpells maps a Circle of the Land land type to its printed Circle Spells, "level: id, id; level: id".
	LandSpells map[string]string `json:"landSpells"`
}

func fighterOracleOf(t *testing.T) fighterOracle {
	t.Helper()
	return classOracleOf(t, "fighter")
}

// classOracleOf returns the hand-written oracle entry of the class id.
func classOracleOf(t *testing.T, id string) fighterOracle {
	t.Helper()
	raw, err := os.ReadFile("../testdata/srd/classes/oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var f fighterOracleFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.Classes {
		if c.ID == id {
			return c
		}
	}
	t.Fatal("the oracle has no " + id)
	return fighterOracle{}
}

func fighterOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "fighter").(content.ClassDefinition)
	if !ok {
		t.Fatal("fighter is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC and CT1 to CT3, MC, SC - the Fighter's
// citations (class p. 47, Champion p. 49, the first pages of their sections), Core Traits, training, Starting
// Equipment options A, B and C, multiclass grants and absent spellcasting equal the hand-written oracle.
// Path: happy.
func TestFighterCoreTraits(t *testing.T) {
	t.Parallel()
	o := fighterOracleOf(t)
	c := fighterOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Fighter" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage {
		t.Errorf("citation = %q p. %d, want %q p. %d", c.Source().Section(), c.Source().Page(), o.Name, o.ClassPage)
	}
	if any, ok := c.PrimaryAbility().(content.AnyOf); !ok || o.PrimaryAbility.Kind != "anyOf" || len(any.Abilities()) != len(o.PrimaryAbility.Abilities) {
		t.Errorf("PrimaryAbility = %#v, want anyOf %v", c.PrimaryAbility(), o.PrimaryAbility.Abilities)
	} else {
		for i, a := range any.Abilities() {
			if mustAbility(t, o.PrimaryAbility.Abilities[i]) != a {
				t.Errorf("primary ability %d = %v, want %s", i, a, o.PrimaryAbility.Abilities[i])
			}
		}
	}
	if die, err := content.ParseHitPointDie(o.HitPointDie); err != nil || c.HitPointDie() != die {
		t.Errorf("HitPointDie = %v, want %s", c.HitPointDie(), o.HitPointDie)
	}
	var saves []ability.Ability
	for _, s := range o.SavingThrows {
		saves = append(saves, mustAbility(t, s))
	}
	if !slices.Equal(c.SavingThrowProficiencies(), saves) {
		t.Errorf("SavingThrowProficiencies = %v, want %v", c.SavingThrowProficiencies(), saves)
	}
	var skills []content.Skill
	for _, s := range o.Skills.Options {
		sk, err := content.ParseSkill(s)
		if err != nil {
			t.Fatal(err)
		}
		skills = append(skills, sk)
	}
	if listed, ok := c.SkillChoice().(content.SkillChoiceListed); !ok || listed.Picks() != o.Skills.Picks || !slices.Equal(listed.Skills(), skills) {
		t.Errorf("SkillChoice = %#v, want %d of %v", c.SkillChoice(), o.Skills.Picks, skills)
	}
	var weapons []string
	for _, w := range c.WeaponProficiencies() {
		weapons = append(weapons, w.Category().String())
		if len(w.RequiresAnyProperty()) != 0 {
			t.Errorf("weapon proficiency %v requires a property, want none", w)
		}
	}
	var armor []string
	for _, a := range c.ArmorTraining() {
		armor = append(armor, a.String())
	}
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || len(c.ToolProficiencies()) != len(o.Tools) {
		t.Errorf("weapons %v armor %v tools %d; want %v %v %v", weapons, armor, len(c.ToolProficiencies()), o.Weapons, o.ArmorTraining, o.Tools)
	}
	g := c.MulticlassGrants()
	var gw, ga []string
	for _, w := range g.WeaponProficiencies() {
		gw = append(gw, w.Category().String())
	}
	for _, a := range g.ArmorTraining() {
		ga = append(ga, a.String())
	}
	if !slices.Equal(gw, o.Multiclass.Weapons) || !slices.Equal(ga, o.Multiclass.ArmorTraining) || len(g.ToolProficiencies()) != len(o.Multiclass.Tools) || g.SkillChoice() != nil || o.Multiclass.Skills != "none" {
		t.Errorf("MulticlassGrants = %#v, want weapons %v armor %v and nothing else", g, o.Multiclass.Weapons, o.Multiclass.ArmorTraining)
	}
	if c.Spellcasting() != nil || o.Spellcasting != "absent" {
		t.Errorf("Spellcasting = %#v, want absent", c.Spellcasting())
	}
	eq := c.StartingEquipment()
	optC, hasC := eq.C()
	if !hasC {
		t.Fatal("the Fighter has an option C")
	}
	for label, got := range map[string]content.ClassEquipmentOption{"A": eq.A(), "B": eq.B(), "C": optC} {
		want := o.Equipment[label]
		if got.Gold() != want.Gold || len(got.Items()) != len(want.Items) {
			t.Errorf("option %s = %d GP, %d items; want %d GP, %d items", label, got.Gold(), len(got.Items()), want.Gold, len(want.Items))
			continue
		}
		for i, it := range got.Items() {
			if it.Name() != want.Items[i][0].(string) || it.Quantity() != int(want.Items[i][1].(float64)) {
				t.Errorf("option %s item %d = %s x%d, want %v", label, i, it.Name(), it.Quantity(), want.Items[i])
			}
		}
	}
}

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF and the class spec class-fighter.md Table A
// (SRD 5.2.1, "Fighter Features", p. 47) - the two count columns and the 20 printed rows: Proficiency Bonus, the
// Class Features cell as printed (annotations, ASI references, the Subclass feature marker at 7, 10, 15 and 18),
// Second Wind and Weapon Mastery.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level).
func TestFighterFeaturesTable(t *testing.T) {
	t.Parallel()
	o := fighterOracleOf(t)
	c := fighterOf(t)
	cols := c.Columns()
	if len(cols) != len(o.Columns) || len(c.Progression()) != 20 || len(o.Rows) != 20 {
		t.Fatalf("columns %d (oracle %d), rows %d (oracle %d)", len(cols), len(o.Columns), len(c.Progression()), len(o.Rows))
	}
	for i, want := range o.Columns {
		if string(cols[i].ID()) != want.ID || cols[i].Header() != want.Header || cols[i].Kind().String() != want.Kind {
			t.Errorf("column %d = %s %q %s, want %s %q %s", i, cols[i].ID(), cols[i].Header(), cols[i].Kind(), want.ID, want.Header, want.Kind)
		}
	}
	for _, want := range o.Rows {
		t.Run("level "+strconv.Itoa(want.Level), func(t *testing.T) {
			t.Parallel()
			row, err := c.RowAt(want.Level)
			if err != nil {
				t.Fatal(err)
			}
			if row.Level() != want.Level || row.ProficiencyBonus() != want.ProficiencyBonus || row.Spellcasting() != nil {
				t.Errorf("row %d = level %d bonus %d spellcasting %v; want bonus %d and none", want.Level, row.Level(), row.ProficiencyBonus(), row.Spellcasting(), want.ProficiencyBonus)
			}
			for id, wantN := range map[string]int{"second_wind": want.SecondWind, "weapon_mastery": want.WeaponMastery} {
				v, err := row.Value(content.ClassColumnID(id))
				if got, ok := v.(content.ColumnCount); err != nil || !ok || got.Value() != wantN {
					t.Errorf("row %d %s = %v (%v), want %d", want.Level, id, v, err, wantN)
				}
			}
			var cell []string
			for _, f := range row.Features() {
				switch v := f.(type) {
				case content.RowSubclassFeature:
					cell = append(cell, "Subclass feature")
				case content.RowClassFeature:
					entry, ok := c.Feature(v.FeatureID()).(content.FeatureEntryFound)
					if !ok {
						t.Fatalf("row %d names the unknown feature %s", want.Level, v.FeatureID())
					}
					name := entry.Entry().Name()
					if v.Annotation() != "" {
						name += " (" + v.Annotation() + ")"
					}
					cell = append(cell, name)
				}
			}
			if !slices.Equal(cell, want.ClassFeatures) {
				t.Errorf("row %d Class Features = %q, want %q", want.Level, cell, want.ClassFeatures)
			}
		})
	}
	var markers []int
	for _, r := range c.Progression() {
		for _, f := range r.Features() {
			if _, ok := f.(content.RowSubclassFeature); ok {
				markers = append(markers, r.Level())
			}
		}
	}
	if !slices.Equal(markers, o.SubclassFeatureRows) {
		t.Errorf("Subclass feature rows = %v, want %v", markers, o.SubclassFeatureRows)
	}
}

func factOf(f content.FeatureEntry) string {
	var out []string
	for _, fact := range f.Facts() {
		switch v := fact.(type) {
		case content.GrantsSubclass:
			out = append(out, "grantsSubclass")
		case content.GrantsFeat:
			out = append(out, "grantsFeat "+v.Category().String())
		case content.AlwaysPreparedSpells:
			var grants []string
			for _, g := range v.Grants() {
				ids := make([]string, len(g.SpellIDs()))
				for i, id := range g.SpellIDs() {
					ids[i] = string(id)
				}
				grants = append(grants, strconv.Itoa(g.Level())+": "+strings.Join(ids, ", "))
			}
			out = append(out, "alwaysPrepared "+strings.Join(grants, "; "))
		default:
			out = append(out, "unexpected")
		}
	}
	return strings.Join(out, ", ")
}

// checkFeatures compares entries with the oracle features: name, level, id prefix, the structured fact, the option ids
// (none when the oracle lists none), no tables, and each oracle phrase present verbatim in the text.
func checkFeatures(t *testing.T, owner string, got []content.FeatureEntry, want []oracleFeature) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s has %d feature entries, want %d", owner, len(got), len(want))
	}
	for i, w := range want {
		f := got[i]
		wantID := owner + "_" + strings.NewReplacer(" ", "_", "’", "").Replace(strings.ToLower(w.Name))
		if w.ID != "" {
			wantID = w.ID
		}
		if f.Name() != w.Name || f.Level() != w.Level || string(f.ID()) != wantID {
			t.Errorf("%s feature %d = %q level %d id %s; want %q level %d id %s", owner, i, f.Name(), f.Level(), f.ID(), w.Name, w.Level, wantID)
		}
		var opts []string
		for _, o := range f.Options() {
			opts = append(opts, string(o.ID()))
		}
		if factOf(f) != w.Fact || !slices.Equal(opts, w.Options) || !tablesMatch(f.Tables(), w.Tables) {
			t.Errorf("%s facts = %q, options %v, tables %v; want %q, options %v, tables %v", f.Name(), factOf(f), opts, f.Tables(), w.Fact, w.Options, w.Tables)
		}
		for _, p := range w.Phrases {
			if !strings.Contains(f.Text(), p) {
				t.Errorf("%s text lacks %q", f.Name(), p)
			}
		}
	}
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables F, FA and SRC - the Fighter has exactly the 15
// feature entries of table F and the Champion the 6 of "Fighter Subclass: Champion" (p. 49), each with its level, id,
// the structured fact of table FA (grantsSubclass, grantsFeat General, Epic Boon and Fighting Style) and the
// key phrases of its printed text; the subclass level is 3 and the Champion cites p. 49.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestFighterFeaturesAndChampion(t *testing.T) {
	t.Parallel()
	o := fighterOracleOf(t)
	c := fighterOf(t)
	checkFeatures(t, "fighter", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	champion := c.Subclasses()[0]
	if string(champion.ID()) != o.Subclass.ID || champion.Name() != o.Subclass.Name || champion.Source().Section() != o.Subclass.Section ||
		champion.Source().Page() != o.Subclass.Page || !champion.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", champion.ID(), champion.Name(), champion.Source().Section(), champion.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "champion", champion.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Fighter texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestFighterTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := fighterOf(t)
	texts := []string{c.Name()}
	for _, f := range c.Features() {
		texts = append(texts, f.Name(), f.Text())
	}
	for _, f := range c.Subclasses()[0].Features() {
		texts = append(texts, f.Name(), f.Text())
	}
	for _, s := range texts {
		for _, term := range bannedTerms() {
			if containsWord(s, term) {
				t.Errorf("text contains the banned term %q: %.60q", term, s)
			}
		}
	}
}
