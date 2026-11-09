package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Druid oracle is the "druid" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func druidOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "druid").(content.ClassDefinition)
	if !ok {
		t.Fatal("druid is not a content.ClassDefinition")
	}
	return c
}

// tablesMatch reports whether the embedded tables equal the oracle tables: names, headers and every cell as printed.
func tablesMatch(got []content.FeatureTable, want []oracleTable) bool {
	if len(got) != len(want) {
		return false
	}
	for i, g := range got {
		w := want[i]
		if g.Name() != w.Name || !slices.Equal(g.Columns(), w.Columns) || len(g.Rows()) != len(w.Rows) {
			return false
		}
		for r, row := range g.Rows() {
			for c, cell := range row {
				text := ""
				switch v := cell.(type) {
				case content.TableCellText:
					text = v.Text()
				case content.TableCellInt:
					text = strconv.Itoa(v.Value())
				}
				if len(w.Rows[r]) != len(row) || text != w.Rows[r][c] {
					return false
				}
			}
		}
	}
	return true
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-druid.md - the Druid's
// citations (class p. 41, Features table p. 41, Circle of the Land p. 46), Core Traits, training (Simple weapons, the Herbalism Kit,
// Light armor and Shields), Starting Equipment options A and B (no option C), multiclass grants (Light armor and Shields, no skill, no
// tool) and the Wisdom slot spellcasting with a Cantrips column equal the hand-written oracle.
// Path: happy.
func TestDruidCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "druid")
	c := druidOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Druid" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage {
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
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || !slices.Equal(rogueToolNames(c.ToolProficiencies()), o.Tools) ||
		len(g.WeaponProficiencies()) != 0 || len(o.Multiclass.Weapons) != 0 || !slices.Equal(mcArmor, o.Multiclass.ArmorTraining) ||
		len(g.ToolProficiencies()) != 0 || len(o.Multiclass.Tools) != 0 || g.SkillChoice() != nil || o.Multiclass.Skills != "none" {
		t.Errorf("weapons %v armor %v tools %v multiclass %#v; want %v %v %v and Light and Shield only", weapons, armor, rogueToolNames(c.ToolProficiencies()), g, o.Weapons, o.ArmorTraining, o.Tools)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots wisdom cantrips" || sc.Ability() != ability.Wisdom || !sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Druid has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ, FS and class-druid.md FR-058, FR-072 (SRD 5.2.1, "Druid
// Features", p. 41) - the Wild Shape column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (the Ability
// Score Improvement references at 8, 12 and 16, the Subclass feature marker at 6, 10 and 14, the empty cells at 9, 11, 13 and 17), Wild
// Shape (a printed dash at level 1), Cantrips, Prepared Spells and the nine Spell Slot counts.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestDruidFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "druid")
	c := druidOf(t)
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
			v, err := row.Value("wild_shape")
			got := "—"
			if n, isCount := v.(content.ColumnCount); isCount {
				got = strconv.Itoa(n.Value())
			}
			if err != nil || got != want.WildShape {
				t.Errorf("row %d wild_shape = %q (%v), want %q", want.Level, got, err, want.WildShape)
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-066, FR-085, FR-093, tables F, FA, OP, TB and SRC - the Druid has exactly the 13 feature
// entries of table F and the Circle of the Land the 5 of "Druid Subclass: Circle of the Land" (p. 46), each with its level, id, fact (grantsSubclass at
// 3, grantsFeat General at 4, grantsFeat Epic Boon at 19, Speak with Animals always prepared at 1), option ids (magician and warden; potent_spellcasting
// and primal_strike; the four lands), embedded tables (Beast Shapes, Nature's Ward) and the key phrases of its printed text and of each option's text; the
// subclass level is 3 and the Circle of the Land cites p. 46. Each land option carries its Circle Spells as an always prepared fact (the printed tables).
// Path: per case (tables F, FA, OP, TB): happy / unhappy (boundary: none missing, none extra).
func TestDruidFeaturesAndCircleOfTheLand(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "druid")
	c := druidOf(t)
	checkFeatures(t, "druid", c.Features(), o.Features)
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
	checkFeatures(t, "circle_of_the_land", sub.Features(), o.SubclassFeatures)
	lands := 0
	for _, opt := range sub.Features()[0].Options() {
		lands++
		var got []string
		for _, fact := range opt.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				t.Errorf("land %s fact = %T, want AlwaysPreparedSpells", opt.ID(), fact)
				continue
			}
			for _, g := range grants.Grants() {
				ids := make([]string, len(g.SpellIDs()))
				for i, id := range g.SpellIDs() {
					ids[i] = string(id)
				}
				got = append(got, strconv.Itoa(g.Level())+": "+strings.Join(ids, ", "))
			}
		}
		if strings.Join(got, "; ") != o.LandSpells[string(opt.ID())] {
			t.Errorf("land %s Circle Spells = %q, want %q", opt.ID(), strings.Join(got, "; "), o.LandSpells[string(opt.ID())])
		}
	}
	if lands != len(o.LandSpells) {
		t.Errorf("%d lands, want %d", lands, len(o.LandSpells))
	}
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Druid texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestDruidTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := druidOf(t)
	texts := []string{c.Name()}
	for _, f := range c.Features() {
		texts = append(texts, f.Name(), f.Text())
		for _, o := range f.Options() {
			texts = append(texts, o.Name(), o.Text())
		}
	}
	for _, f := range c.Subclasses()[0].Features() {
		texts = append(texts, f.Name(), f.Text())
		for _, o := range f.Options() {
			texts = append(texts, o.Name(), o.Text())
		}
	}
	for _, s := range texts {
		for _, term := range bannedTerms() {
			if containsWord(s, term) {
				t.Errorf("text contains the banned term %q: %.60q", term, s)
			}
		}
	}
}

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100 and plan-phase-3.md 5.10 - every spell id the Druidic fact (Speak with Animals) and the
// four Circle Spells options name resolves to a bundled spell, so the merged set validates; Speak with Animals is on the Druid list.
// Path: unhappy (missing content: no dangling spell id).
func TestDruidSpellIDsResolve(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	c := druidOf(t)
	var ids []content.SpellID
	collect := func(facts []content.FeatureFact) {
		for _, fact := range facts {
			if grants, ok := fact.(content.AlwaysPreparedSpells); ok {
				for _, g := range grants.Grants() {
					ids = append(ids, g.SpellIDs()...)
				}
			}
		}
	}
	for _, f := range slices.Concat(c.Features(), c.Subclasses()[0].Features()) {
		collect(f.Facts())
		for _, opt := range f.Options() {
			collect(opt.Facts())
		}
	}
	if len(ids) != 25 {
		t.Errorf("%d spell ids named, want 1 + 4 x 6", len(ids))
	}
	for _, id := range ids {
		spellOf(t, src, string(id))
	}
	if !slices.Contains(spellOf(t, src, "speak_with_animals").Classes(), content.ClassID("druid")) {
		t.Error("speak_with_animals is not on the Druid list")
	}
}
