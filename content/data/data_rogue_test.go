package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Rogue oracle is the "rogue" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func rogueOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "rogue").(content.ClassDefinition)
	if !ok {
		t.Fatal("rogue is not a content.ClassDefinition")
	}
	return c
}

func rogueSkills(t *testing.T, names []string) []content.Skill {
	t.Helper()
	var out []content.Skill
	for _, s := range names {
		sk, err := content.ParseSkill(s)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sk)
	}
	return out
}

func rogueToolNames(tools []content.ClassToolProficiency) []string {
	var out []string
	for _, p := range tools {
		if s, ok := p.(content.ClassToolProficiencySpecific); ok {
			out = append(out, "specific "+s.Name())
		}
	}
	return out
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-rogue.md FR-009 -
// the Rogue's citations (class p. 61, Features table p. 62, Thief p. 64), Core Traits, training (Simple weapons and
// Martial weapons with Finesse or Light, Thieves' Tools, Light armor), Starting Equipment options A and B (no option C),
// multiclass grants (Light armor, one listed skill, Thieves' Tools) and absent spellcasting equal the hand-written oracle.
// Path: happy.
func TestRogueCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "rogue")
	c := rogueOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Rogue" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage+1 {
		t.Errorf("citation = %q p. %d, want %q p. %d", c.Source().Section(), c.Source().Page(), o.Name, o.ClassPage)
	}
	all, ok := c.PrimaryAbility().(content.AllOf)
	if !ok || o.PrimaryAbility.Kind != "allOf" || len(all.Abilities()) != 1 || all.Abilities()[0] != mustAbility(t, o.PrimaryAbility.Abilities[0]) {
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
	want := rogueSkills(t, o.Skills.Options)
	if listed, ok := c.SkillChoice().(content.SkillChoiceListed); !ok || listed.Picks() != o.Skills.Picks || !slices.Equal(listed.Skills(), want) {
		t.Errorf("SkillChoice = %#v, want %d of %v", c.SkillChoice(), o.Skills.Picks, want)
	}
	var weapons []string
	for _, w := range c.WeaponProficiencies() {
		weapons = append(weapons, w.Category().String())
		var props []string
		for _, p := range w.RequiresAnyProperty() {
			props = append(props, p.String())
		}
		if got := strings.Join(props, " or "); got != o.WeaponProperty[w.Category().String()] {
			t.Errorf("%s weapons require %q, want %q", w.Category(), got, o.WeaponProperty[w.Category().String()])
		}
	}
	var armor []string
	for _, a := range c.ArmorTraining() {
		armor = append(armor, a.String())
	}
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || !slices.Equal(rogueToolNames(c.ToolProficiencies()), o.Tools) {
		t.Errorf("weapons %v armor %v tools %v; want %v %v %v", weapons, armor, rogueToolNames(c.ToolProficiencies()), o.Weapons, o.ArmorTraining, o.Tools)
	}
	g := c.MulticlassGrants()
	var ga []string
	for _, a := range g.ArmorTraining() {
		ga = append(ga, a.String())
	}
	mc, mcOK := g.SkillChoice().(content.SkillChoiceListed)
	if len(g.WeaponProficiencies()) != 0 || len(o.Multiclass.Weapons) != 0 || !slices.Equal(ga, o.Multiclass.ArmorTraining) ||
		!slices.Equal(rogueToolNames(g.ToolProficiencies()), o.Multiclass.Tools) || !mcOK || mc.Picks() != 1 || !slices.Equal(mc.Skills(), want) ||
		o.Multiclass.Skills != "listed, 1 pick" {
		t.Errorf("MulticlassGrants = %#v, want Light armor, Thieves' Tools and 1 listed skill", g)
	}
	if c.Spellcasting() != nil || o.Spellcasting != "absent" {
		t.Errorf("Spellcasting = %#v, want absent", c.Spellcasting())
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Rogue has no option C")
	}
	for label, got := range map[string]content.ClassEquipmentOption{"A": eq.A(), "B": eq.B()} {
		w := o.Equipment[label]
		if got.Gold() != w.Gold || len(got.Items()) != len(w.Items) {
			t.Errorf("option %s = %d GP, %d items; want %d GP, %d items", label, got.Gold(), len(got.Items()), w.Gold, len(w.Items))
			continue
		}
		for i, it := range got.Items() {
			if it.Name() != w.Items[i][0].(string) || it.Quantity() != int(w.Items[i][1].(float64)) {
				t.Errorf("option %s item %d = %s x%d, want %v", label, i, it.Name(), it.Quantity(), w.Items[i])
			}
		}
	}
}

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF and class-rogue.md FR-009, FR-010, FR-011, FR-012
// (SRD 5.2.1, "Rogue Features", p. 62) - the Sneak Attack dice column and the 20 printed rows: Proficiency Bonus, the
// Class Features cell as printed (the Expertise and Ability Score Improvement references, the Subclass feature marker at
// 9, 13 and 17) and the Sneak Attack dice.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level).
func TestRogueFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "rogue")
	c := rogueOf(t)
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
			v, err := row.Value("sneak_attack")
			if got, ok := v.(content.ColumnDice); err != nil || !ok || strconv.Itoa(got.Count())+"d"+strconv.Itoa(got.Sides()) != want.SneakAttack {
				t.Errorf("row %d sneak_attack = %v (%v), want %s", want.Level, v, err, want.SneakAttack)
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
					cell = append(cell, entry.Entry().Name())
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

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables F, FA and SRC - the Rogue has exactly the 18
// feature entries of table F and the Thief the 5 of "Rogue Subclass: Thief" (p. 64), each with its level, id, fact and
// the key phrases of its printed text; the subclass level is 3 and the Thief cites p. 64.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestRogueFeaturesAndThief(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "rogue")
	c := rogueOf(t)
	checkFeatures(t, "rogue", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "thief", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Rogue texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestRogueTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := rogueOf(t)
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
