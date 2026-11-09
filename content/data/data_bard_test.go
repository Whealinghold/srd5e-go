package data_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Bard oracle is the "bard" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func bardOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "bard").(content.ClassDefinition)
	if !ok {
		t.Fatal("bard is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-bard.md FR-122 - the Bard's citations (class p. 31,
// Features table p. 31, College of Lore p. 35), Core Traits (Dexterity and Charisma saves, any 3 skills, Simple weapons, three Musical Instruments, Light
// armor), Starting Equipment options A and B (no option C), multiclass grants (Light armor, one Musical Instrument, one skill of any) and the Charisma slot
// spellcasting with a Cantrips column equal the hand-written oracle.
// Path: happy.
func TestBardCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "bard")
	c := bardOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Bard" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage {
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
	if any, ok := c.SkillChoice().(content.SkillChoiceAny); !ok || o.Skills.Kind != "any" || any.Picks() != o.Skills.Picks {
		t.Errorf("SkillChoice = %#v, want %d of any", c.SkillChoice(), o.Skills.Picks)
	}
	var weapons, armor []string
	for _, w := range c.WeaponProficiencies() {
		if len(w.RequiresAnyProperty()) != 0 {
			t.Errorf("%s weapons require a property", w.Category())
		}
		weapons = append(weapons, w.Category().String())
	}
	for _, a := range c.ArmorTraining() {
		armor = append(armor, a.String())
	}
	tools := c.ToolProficiencies()
	choice, ok := tools[0].(content.ClassToolProficiencyChoice)
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || len(tools) != 1 || !ok || choice.Picks() != 3 ||
		!slices.Equal(choice.Categories(), []string{"Musical Instrument"}) || !slices.Equal(o.Tools, []string{"choice 3 of: Musical Instrument"}) {
		t.Errorf("weapons %v armor %v tools %#v; want %v %v and 3 Musical Instruments", weapons, armor, tools, o.Weapons, o.ArmorTraining)
	}
	g := c.MulticlassGrants()
	mcTools := g.ToolProficiencies()
	mcChoice, mcOK := mcTools[0].(content.ClassToolProficiencyChoice)
	mcSkill, skillOK := g.SkillChoice().(content.SkillChoiceAny)
	var mcArmor []string
	for _, a := range g.ArmorTraining() {
		mcArmor = append(mcArmor, a.String())
	}
	if len(g.WeaponProficiencies()) != 0 || len(o.Multiclass.Weapons) != 0 || !slices.Equal(mcArmor, o.Multiclass.ArmorTraining) || len(mcTools) != 1 || !mcOK ||
		mcChoice.Picks() != 1 || !slices.Equal(mcChoice.Categories(), []string{"Musical Instrument"}) || !skillOK || mcSkill.Picks() != 1 ||
		!slices.Equal(o.Multiclass.Tools, []string{"choice 1 of: Musical Instrument"}) || o.Multiclass.Skills != "any, 1 pick" {
		t.Errorf("MulticlassGrants = %#v, want Light armor, 1 Musical Instrument and 1 skill of any", g)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots charisma cantrips" || sc.Ability() != ability.Charisma || !sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Bard has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ, FS and class-bard.md FR-023, FR-125 (SRD 5.2.1, "Bard Features", p. 31) - the Bardic
// Die column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (the Ability Score Improvement references at 8, 12 and 16, the
// Expertise reference at 9, the Subclass feature marker at 6 and 14, the empty cells at 11, 13, 15 and 17), Bardic Die, Cantrips, Prepared Spells and the nine
// Spell Slot counts.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestBardFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "bard")
	c := bardOf(t)
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
			v, err := row.Value("bardic_die")
			if got, ok := v.(content.ColumnDice); err != nil || !ok || got.Count() != 1 || "d"+strconv.Itoa(got.Sides()) != want.BardicDie {
				t.Errorf("row %d bardic_die = %v (%v), want %s", want.Level, v, err, want.BardicDie)
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-085, FR-093, tables F, FA and SRC - the Bard has exactly the 12 feature entries of table F and the
// College of Lore the 4 of "Bard Subclass: College of Lore" (p. 35), each with its level, id, fact (grantsSubclass at 3, grantsFeat General at 4, grantsFeat
// Epic Boon at 19, the Words of Creation always-prepared grant at 20) and the key phrases of its printed text; the subclass level is 3 and the College of Lore
// cites p. 35.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestBardFeaturesAndCollegeOfLore(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "bard")
	c := bardOf(t)
	checkFeatures(t, "bard", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "college_of_lore", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Bard texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestBardTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := bardOf(t)
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

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100 and plan-phase-3.md 5.10 - every spell id the Words of Creation fact names resolves to a bundled
// spell that has Bard in its class list, so the merged set validates.
// Path: unhappy (missing content: no dangling spell id).
func TestWordsOfCreationSpellsResolveToBardSpells(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	seen := 0
	for _, f := range bardOf(t).Features() {
		for _, fact := range f.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				continue
			}
			for _, g := range grants.Grants() {
				for _, id := range g.SpellIDs() {
					seen++
					if !slices.Contains(spellOf(t, src, string(id)).Classes(), content.ClassID("bard")) {
						t.Errorf("%s (granted at %d) is not on the Bard list", id, g.Level())
					}
				}
			}
		}
	}
	if seen != 2 {
		t.Errorf("always-prepared spells named = %d, want 2 (Power Word Heal and Power Word Kill)", seen)
	}
}
