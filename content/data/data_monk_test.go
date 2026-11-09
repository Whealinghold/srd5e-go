package data_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Monk oracle is the "monk" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func monkOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "monk").(content.ClassDefinition)
	if !ok {
		t.Fatal("monk is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC - the Monk's citations
// (class p. 49, Features table p. 50, Warrior of the Open Hand p. 52), Core Traits, training (Simple weapons and Martial
// weapons with the Light property, one tool of Artisan's Tools or Musical Instrument, no armor), Starting Equipment
// options A and B (no option C), multiclass grants (none) and absent spellcasting equal the hand-written oracle.
// Path: happy.
func TestMonkCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "monk")
	c := monkOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Monk" || !c.Source().CitesSRD521() {
		t.Errorf("citation = %q p. %d, want %q p. %d", c.Source().Section(), c.Source().Page(), o.Name, o.ClassPage)
	}
	all, ok := c.PrimaryAbility().(content.AllOf)
	if !ok || o.PrimaryAbility.Kind != "allOf" || len(all.Abilities()) != 2 ||
		all.Abilities()[0] != mustAbility(t, o.PrimaryAbility.Abilities[0]) || all.Abilities()[1] != mustAbility(t, o.PrimaryAbility.Abilities[1]) {
		t.Errorf("PrimaryAbility = %#v, want allOf %v", c.PrimaryAbility(), o.PrimaryAbility.Abilities)
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
		if want := o.WeaponProperty[w.Category().String()]; want != "" {
			if props := w.RequiresAnyProperty(); len(props) != 1 || props[0].String() != want {
				t.Errorf("%s weapons require %v, want %s", w.Category(), props, want)
			}
		} else if len(w.RequiresAnyProperty()) != 0 {
			t.Errorf("%s weapons require %v, want no property", w.Category(), w.RequiresAnyProperty())
		}
	}
	if !slices.Equal(weapons, o.Weapons) || len(c.ArmorTraining()) != 0 || len(o.ArmorTraining) != 0 {
		t.Errorf("weapons %v armor %v; want %v and none", weapons, c.ArmorTraining(), o.Weapons)
	}
	tools := c.ToolProficiencies()
	choice, ok := tools[0].(content.ClassToolProficiencyChoice)
	if len(tools) != 1 || !ok || choice.Picks() != 1 || !slices.Equal(choice.Categories(), []string{"Artisan’s Tools", "Musical Instrument"}) ||
		!slices.Equal(o.Tools, []string{"choice 1 of: Artisan’s Tools, Musical Instrument"}) {
		t.Errorf("ToolProficiencies = %#v, want one pick of Artisan’s Tools or Musical Instrument", tools)
	}
	g := c.MulticlassGrants()
	if len(g.WeaponProficiencies()) != 0 || len(g.ArmorTraining()) != 0 || len(g.ToolProficiencies()) != 0 || g.SkillChoice() != nil || o.Multiclass.Skills != "none" {
		t.Errorf("MulticlassGrants = %#v, want none", g)
	}
	if c.Spellcasting() != nil || o.Spellcasting != "absent" {
		t.Errorf("Spellcasting = %#v, want absent", c.Spellcasting())
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Monk has no option C")
	}
	for label, got := range map[string]content.ClassEquipmentOption{"A": eq.A(), "B": eq.B()} {
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF and class-monk.md section 4.2 (SRD 5.2.1,
// "Monk Features", p. 50) - the three columns (Martial Arts dice, Focus Points count, Unarmored Movement feet) and the 20
// printed rows: Proficiency Bonus, the Class Features cell as printed (the Subclass feature marker at 6, 11 and 17), and
// the three column values, a dash at level 1 for Focus Points and Unarmored Movement.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level).
func TestMonkFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "monk")
	c := monkOf(t)
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
			v, err := row.Value("martial_arts")
			if got, ok := v.(content.ColumnDice); err != nil || !ok || strconv.Itoa(got.Count())+"d"+strconv.Itoa(got.Sides()) != want.MartialArts {
				t.Errorf("row %d martial_arts = %v (%v), want %s", want.Level, v, err, want.MartialArts)
			}
			for id, wantN := range map[string]int{"focus_points": want.FocusPoints, "unarmored_movement": want.UnarmoredMovement} {
				v, err := row.Value(content.ClassColumnID(id))
				if err != nil {
					t.Fatal(err)
				}
				switch got := v.(type) {
				case content.ColumnNone:
					if wantN != 0 {
						t.Errorf("row %d %s = none, want %d", want.Level, id, wantN)
					}
				case content.ColumnCount:
					if got.Value() != wantN || wantN == 0 {
						t.Errorf("row %d %s = %d, want %d", want.Level, id, got.Value(), wantN)
					}
				case content.ColumnFeetBonus:
					if got.Feet() != wantN || wantN == 0 {
						t.Errorf("row %d %s = %d ft, want %d", want.Level, id, got.Feet(), wantN)
					}
				default:
					t.Errorf("row %d %s = %T", want.Level, id, v)
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

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables F, FA and SRC - the Monk has exactly the 22
// feature entries of table F and the Warrior of the Open Hand the 4 of "Monk Subclass: Warrior of the Open Hand" (p. 52),
// each with its level, id, fact and the key phrases of its printed text; the subclass level is 3 and the subclass cites p. 52.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestMonkFeaturesAndOpenHand(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "monk")
	c := monkOf(t)
	checkFeatures(t, "monk", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "warrior_of_the_open_hand", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Monk texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestMonkTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := monkOf(t)
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
