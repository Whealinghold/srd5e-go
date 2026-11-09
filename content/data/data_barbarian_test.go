package data_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Barbarian oracle is the "barbarian" entry of content/testdata/srd/classes/oracle.json, written by hand from the
// printed pages and never generated from content/data (content-classes-data.md FR-093).

func barbarianOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "barbarian").(content.ClassDefinition)
	if !ok {
		t.Fatal("barbarian is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC and CT1 to CT3, MC, SC - the Barbarian's
// citations (class p. 28, Path of the Berserker p. 30), Core Traits, training, Starting Equipment options A and B
// (no option C), multiclass grants and absent spellcasting equal the hand-written oracle.
// Path: happy.
func TestBarbarianCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "barbarian")
	c := barbarianOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Barbarian" || !c.Source().CitesSRD521() {
		t.Errorf("citation = %q p. %d, want %q p. %d", c.Source().Section(), c.Source().Page(), o.Name, o.ClassPage)
	}
	if all, ok := c.PrimaryAbility().(content.AllOf); !ok || o.PrimaryAbility.Kind != "allOf" || len(all.Abilities()) != 1 || all.Abilities()[0] != mustAbility(t, o.PrimaryAbility.Abilities[0]) {
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
	var weapons, armor, gw, ga []string
	for _, w := range c.WeaponProficiencies() {
		weapons = append(weapons, w.Category().String())
	}
	for _, a := range c.ArmorTraining() {
		armor = append(armor, a.String())
	}
	g := c.MulticlassGrants()
	for _, w := range g.WeaponProficiencies() {
		gw = append(gw, w.Category().String())
	}
	for _, a := range g.ArmorTraining() {
		ga = append(ga, a.String())
	}
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || len(c.ToolProficiencies()) != 0 {
		t.Errorf("weapons %v armor %v tools %d; want %v %v none", weapons, armor, len(c.ToolProficiencies()), o.Weapons, o.ArmorTraining)
	}
	if !slices.Equal(gw, o.Multiclass.Weapons) || !slices.Equal(ga, o.Multiclass.ArmorTraining) || len(g.ToolProficiencies()) != 0 || g.SkillChoice() != nil {
		t.Errorf("MulticlassGrants = %#v, want weapons %v armor %v and nothing else", g, o.Multiclass.Weapons, o.Multiclass.ArmorTraining)
	}
	if c.Spellcasting() != nil || o.Spellcasting != "absent" {
		t.Errorf("Spellcasting = %#v, want absent", c.Spellcasting())
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Barbarian has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF and class-barbarian.md section 4.2.1
// (SRD 5.2.1, "Barbarian Features", p. 28) - the three columns (Rages, Rage Damage, Weapon Mastery) and the 20 printed
// rows: Proficiency Bonus, the Class Features cell as printed (the Subclass feature marker at 6, 10 and 14) and the
// three column values.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level).
func TestBarbarianFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "barbarian")
	c := barbarianOf(t)
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
			for id, wantN := range map[string]int{"rages": want.Rages, "weapon_mastery": want.WeaponMastery} {
				v, err := row.Value(content.ClassColumnID(id))
				if got, ok := v.(content.ColumnCount); err != nil || !ok || got.Value() != wantN {
					t.Errorf("row %d %s = %v (%v), want %d", want.Level, id, v, err, wantN)
				}
			}
			v, err := row.Value("rage_damage")
			if got, ok := v.(content.ColumnBonus); err != nil || !ok || got.Value() != want.RageDamage {
				t.Errorf("row %d rage_damage = %v (%v), want +%d", want.Level, v, err, want.RageDamage)
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

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables F, FA and SRC - the Barbarian has exactly the 20
// feature entries of table F (the two Improved Brutal Strike entries with their level ids) and the Path of the
// Berserker the 4 of "Barbarian Subclass: Path of the Berserker" (p. 30), each with its level, id, fact and the key
// phrases of its printed text; the subclass level is 3 and the subclass cites p. 30.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestBarbarianFeaturesAndBerserker(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "barbarian")
	c := barbarianOf(t)
	checkFeatures(t, "barbarian", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "path_of_the_berserker", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Barbarian texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestBarbarianTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := barbarianOf(t)
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
