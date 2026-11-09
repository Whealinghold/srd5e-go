package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Sorcerer oracle is the "sorcerer" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func sorcererOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "sorcerer").(content.ClassDefinition)
	if !ok {
		t.Fatal("sorcerer is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-sorcerer.md - the Sorcerer's
// citations (class p. 64, Features table p. 65, Draconic Sorcery p. 69), Core Traits (Constitution and Charisma saves, 2 listed skills, Simple weapons, no
// armor training), Starting Equipment options A and B (no option C), multiclass grants (nothing but the Hit Point Die) and the
// Charisma slot spellcasting with a Cantrips column equal the hand-written oracle.
// Path: happy.
func TestSorcererCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "sorcerer")
	c := sorcererOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Sorcerer" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage+1 {
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
	var weapons, armor, mcArmor []string
	for _, w := range c.WeaponProficiencies() {
		if len(w.RequiresAnyProperty()) != 0 {
			t.Errorf("%s weapons require a property", w.Category())
		}
		weapons = append(weapons, w.Category().String())
	}
	for _, a := range c.ArmorTraining() {
		armor = append(armor, a.String())
	}
	g := c.MulticlassGrants()
	for _, a := range g.ArmorTraining() {
		mcArmor = append(mcArmor, a.String())
	}
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || len(c.ToolProficiencies()) != len(o.Tools) ||
		len(g.WeaponProficiencies()) != 0 || len(o.Multiclass.Weapons) != 0 || !slices.Equal(mcArmor, o.Multiclass.ArmorTraining) ||
		len(g.ToolProficiencies()) != 0 || g.SkillChoice() != nil || o.Multiclass.Skills != "none" {
		t.Errorf("weapons %v armor %v tools %v multiclass %#v; want %v %v %v and Light, Medium and Shield", weapons, armor, c.ToolProficiencies(), g, o.Weapons, o.ArmorTraining, o.Tools)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots charisma cantrips" || sc.Ability() != ability.Charisma || !sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Sorcerer has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ, FS and class-sorcerer.md FR-017, FR-102 (SRD 5.2.1, "Sorcerer
// Features", p. 65) - the Sorcery Points column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (the Ability
// Score Improvement references at 4, 8, 12 and 16, Metamagic at 10 and 17, the Subclass feature marker at 6, 14 and 18, the empty cells at 9, 11, 13 and 15), Sorcery Points
// (a printed dash at level 1), Cantrips, Prepared Spells and the nine Spell Slot counts.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestSorcererFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "sorcerer")
	c := sorcererOf(t)
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
			sc, ok := row.Spellcasting().(content.SlotSpellcastingRow)
			cantrips, hasCantrips := sc.Cantrips()
			if !ok || row.Level() != want.Level || row.ProficiencyBonus() != want.ProficiencyBonus || !hasCantrips || cantrips != want.Cantrips ||
				sc.PreparedSpells() != want.PreparedSpells || sc.SpellSlots() != want.SpellSlots {
				t.Errorf("row %d = bonus %d, %v; want bonus %d, cantrips %d, prepared %d, slots %v", want.Level, row.ProficiencyBonus(), row.Spellcasting(),
					want.ProficiencyBonus, want.Cantrips, want.PreparedSpells, want.SpellSlots)
			}
			v, err := row.Value("sorcery_points")
			got := 0
			if n, isCount := v.(content.ColumnCount); isCount {
				got = n.Value()
			}
			if _, none := v.(content.ColumnNone); err != nil || got != want.SorceryPoints || (got == 0 && !none) {
				t.Errorf("row %d sorcery_points = %v (%v), want %d (0 is the printed dash)", want.Level, v, err, want.SorceryPoints)
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
			if !slices.Equal(cell, want.ClassFeatures) && (len(cell) != 0 || len(want.ClassFeatures) != 0) {
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-085, FR-093, tables F, FA, OP and SRC - the Sorcerer has exactly the 10 feature entries of
// table F and Draconic Sorcery the 5 of "Sorcerer Subclass: Draconic Sorcery" (p. 69), each with its level, id, fact (grantsSubclass at 3, grantsFeat General at 4,
// grantsFeat Epic Boon at 19, the four Draconic Spells grants), the ten Metamagic option ids, the Creating Spell Slots table and the
// key phrases of its printed text and of each option's text; the subclass level is 3 and Draconic Sorcery cites p. 69.
// Path: per case (tables F, FA, OP): happy / unhappy (boundary: none missing, none extra).
func TestSorcererFeaturesAndLifeDomain(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "sorcerer")
	c := sorcererOf(t)
	checkFeatures(t, "sorcerer", c.Features(), o.Features)
	for _, f := range c.Features() {
		for _, want := range o.OptionTexts[string(f.ID())] {
			var text string
			for _, opt := range f.Options() {
				if string(opt.ID()) == want.ID {
					text = opt.Text()
				}
			}
			for _, p := range want.Phrases {
				if !strings.Contains(text, p) {
					t.Errorf("%s option %s text lacks %q", f.Name(), want.ID, p)
				}
			}
		}
	}
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "draconic_sorcery", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Sorcerer texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestSorcererTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := sorcererOf(t)
	texts := []string{c.Name()}
	for _, f := range c.Features() {
		texts = append(texts, f.Name(), f.Text())
		for _, o := range f.Options() {
			texts = append(texts, o.Name(), o.Text())
		}
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

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100, plan-phase-3.md 5.10 and class-sorcerer.md FR-103 - the ten spell ids the Draconic Spells
// fact names resolve to bundled spells (so the merged set validates), and exactly Command, Arcane Eye, Legend Lore and Summon Dragon are not on the Sorcerer
// list, so a Sorcerer casts them only because the feature always prepares them.
// Path: unhappy (missing content: no dangling spell id; rule interaction: a granted spell off the list).
func TestDraconicSpellsResolveAndFourAreOffTheSorcererList(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	seen := 0
	var offList []string
	for _, f := range sorcererOf(t).Subclasses()[0].Features() {
		for _, fact := range f.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				continue
			}
			for _, g := range grants.Grants() {
				for _, id := range g.SpellIDs() {
					seen++
					if !slices.Contains(spellOf(t, src, string(id)).Classes(), content.ClassID("sorcerer")) {
						offList = append(offList, string(id))
					}
				}
			}
		}
	}
	slices.Sort(offList)
	if want := []string{"arcane_eye", "command", "legend_lore", "summon_dragon"}; seen != 10 || !slices.Equal(offList, want) {
		t.Errorf("granted spells = %d, off the Sorcerer list = %v; want 10 and %v", seen, offList, want)
	}
}

// Requirement: doc/specs/content-classes-data.md FR-061 to FR-065, table OP and class-sorcerer.md FR-059 (SRD 5.2.1, "Metamagic Options", pp. 66-67) - the ten
// Metamagic options carry the printed Sorcery Point costs (1 for all but Heightened Spell and Quickened Spell, 2 for those), none repeatable and none with a prerequisite.
// Path: per case (table OP): happy / unhappy (boundary: costs 1 and 2).
func TestSorcererMetamagicOptionCosts(t *testing.T) {
	t.Parallel()
	want := map[string]int{"careful_spell": 1, "distant_spell": 1, "empowered_spell": 1, "extended_spell": 1, "heightened_spell": 2,
		"quickened_spell": 2, "seeking_spell": 1, "subtle_spell": 1, "transmuted_spell": 1, "twinned_spell": 1}
	var options []content.FeatureOption
	for _, f := range sorcererOf(t).Features() {
		options = append(options, f.Options()...)
	}
	if len(options) != len(want) {
		t.Fatalf("options = %d, want %d", len(options), len(want))
	}
	for _, o := range options {
		t.Run(string(o.ID()), func(t *testing.T) {
			t.Parallel()
			_, hasPrerequisite := o.Prerequisite()
			if o.PointCost() != want[string(o.ID())] || o.Repeatable() || hasPrerequisite {
				t.Errorf("%s = cost %d, repeatable %v, prerequisite %v; want cost %d only", o.ID(), o.PointCost(), o.Repeatable(), hasPrerequisite, want[string(o.ID())])
			}
		})
	}
}
