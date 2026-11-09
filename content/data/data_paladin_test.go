package data_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Paladin oracle is the "paladin" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func paladinOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "paladin").(content.ClassDefinition)
	if !ok {
		t.Fatal("paladin is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-paladin.md - the Paladin's
// citations (class p. 53, Features table p. 53, Oath of Devotion p. 56), Core Traits, training (Simple and Martial weapons, Light, Medium and Heavy armor and
// Shields), Starting Equipment options A and B (no option C), multiclass grants (Martial weapons, Light and Medium armor and Shields, no skill) and the
// Charisma slot spellcasting with no Cantrips column equal the hand-written oracle.
// Path: happy.
func TestPaladinCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "paladin")
	c := paladinOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Paladin" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage {
		t.Errorf("citation = %q p. %d, want %q p. %d", c.Source().Section(), c.Source().Page(), o.Name, o.ClassPage)
	}
	all, ok := c.PrimaryAbility().(content.AllOf)
	if !ok || o.PrimaryAbility.Kind != "allOf" || len(all.Abilities()) != 2 || all.Abilities()[0] != mustAbility(t, o.PrimaryAbility.Abilities[0]) || all.Abilities()[1] != mustAbility(t, o.PrimaryAbility.Abilities[1]) {
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
	var weapons, armor, mcArmor, mcWeapons []string
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
	for _, w := range g.WeaponProficiencies() {
		mcWeapons = append(mcWeapons, w.Category().String())
	}
	for _, a := range g.ArmorTraining() {
		mcArmor = append(mcArmor, a.String())
	}
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || len(c.ToolProficiencies()) != len(o.Tools) ||
		!slices.Equal(mcWeapons, o.Multiclass.Weapons) || !slices.Equal(mcArmor, o.Multiclass.ArmorTraining) ||
		len(g.ToolProficiencies()) != 0 || g.SkillChoice() != nil || o.Multiclass.Skills != "none" {
		t.Errorf("weapons %v armor %v tools %v multiclass %#v; want %v %v %v and Martial weapons with Light, Medium and Shield", weapons, armor, c.ToolProficiencies(), g, o.Weapons, o.ArmorTraining, o.Tools)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots charisma" || sc.Ability() != ability.Charisma || sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Paladin has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ, FS and class-paladin.md FR-009, FR-010, FR-012 (SRD 5.2.1, "Paladin
// Features", p. 53) - the Channel Divinity column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (the Ability
// Score Improvement references at 4, 8, 12 and 16, the Subclass feature marker at 7, 15 and 20, the empty cells at 13 and 17), Channel Divinity
// (a printed dash at levels 1 and 2), no Cantrips column, Prepared Spells and the nine Spell Slot counts.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestPaladinFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "paladin")
	c := paladinOf(t)
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
			_, hasCantrips := sc.Cantrips()
			if !ok || row.Level() != want.Level || row.ProficiencyBonus() != want.ProficiencyBonus || hasCantrips ||
				sc.PreparedSpells() != want.PreparedSpells || sc.SpellSlots() != want.SpellSlots {
				t.Errorf("row %d = bonus %d, %v; want bonus %d, no cantrips, prepared %d, slots %v", want.Level, row.ProficiencyBonus(), row.Spellcasting(),
					want.ProficiencyBonus, want.PreparedSpells, want.SpellSlots)
			}
			v, err := row.Value("channel_divinity")
			got := "—"
			if n, isCount := v.(content.ColumnCount); isCount {
				got = strconv.Itoa(n.Value())
			}
			if err != nil || got != want.ChannelDivinity {
				t.Errorf("row %d channel_divinity = %q (%v), want %q", want.Level, got, err, want.ChannelDivinity)
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-085, FR-093, tables F, FA and SRC - the Paladin has exactly the 17 feature entries of table F and
// the Oath of Devotion the 5 of "Paladin Subclass: Oath of Devotion" (pp. 56-57), each with its level, id, fact (grantsSubclass at 3, grantsFeat General at 4,
// grantsFeat Fighting Style at 2, grantsFeat Epic Boon at 19, the Divine Smite and Find Steed grants, the five Oath of Devotion Spells grants) and the key
// phrases of its printed text; the Paladin has no option lists; the subclass level is 3 and the Oath of Devotion cites p. 56.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestPaladinFeaturesAndOathOfDevotion(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "paladin")
	c := paladinOf(t)
	checkFeatures(t, "paladin", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "oath_of_devotion", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Paladin texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestPaladinTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := paladinOf(t)
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

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100, class-paladin.md FR-023 and plan-phase-3.md 5.10 - every spell id the Paladin's Smite, Faithful
// Steed and Oath of Devotion Spells facts name resolves to a bundled spell (so the merged set validates), and exactly the five Oath spells that are not on the
// printed Paladin list (Beacon of Hope, Freedom of Movement, Guardian of Faith, Commune, Flame Strike; SRD 5.2.1, "Paladin Spell List", pp. 55-56) lack Paladin in
// their class list.
// Path: unhappy (missing content: no dangling spell id; rule interaction: a granted spell off the list).
func TestPaladinGrantedSpellsResolveAndFiveAreOffThePaladinList(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	c := paladinOf(t)
	seen := 0
	var off []string
	for _, f := range slices.Concat(c.Features(), c.Subclasses()[0].Features()) {
		for _, fact := range f.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				continue
			}
			for _, g := range grants.Grants() {
				for _, id := range g.SpellIDs() {
					seen++
					if !slices.Contains(spellOf(t, src, string(id)).Classes(), content.ClassID("paladin")) {
						off = append(off, string(id))
					}
				}
			}
		}
	}
	slices.Sort(off)
	if want := []string{"beacon_of_hope", "commune", "flame_strike", "freedom_of_movement", "guardian_of_faith"}; seen != 12 || !slices.Equal(off, want) {
		t.Errorf("granted spells = %d, off the Paladin list = %v; want 12 and %v", seen, off, want)
	}
}
