package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Cleric oracle is the "cleric" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func clericOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "cleric").(content.ClassDefinition)
	if !ok {
		t.Fatal("cleric is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-cleric.md - the Cleric's
// citations (class p. 36, Features table p. 36, Life Domain p. 40), Core Traits, training (Simple weapons, Light and Medium armor and
// Shields), Starting Equipment options A and B (no option C), multiclass grants (Light and Medium armor and Shields, no skill) and the
// Wisdom slot spellcasting with a Cantrips column equal the hand-written oracle.
// Path: happy.
func TestClericCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "cleric")
	c := clericOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Cleric" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage {
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
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots wisdom cantrips" || sc.Ability() != ability.Wisdom || !sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Cleric has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ, FS and class-cleric.md FR-015, FR-017, FR-038 (SRD 5.2.1, "Cleric
// Features", p. 36) - the Channel Divinity column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (the Ability
// Score Improvement references at 8, 12 and 16, the Subclass feature marker at 6 and 17, the empty cells at 9, 11, 13, 15 and 18), Channel Divinity
// (a printed dash at level 1), Cantrips, Prepared Spells and the nine Spell Slot counts.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestClericFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "cleric")
	c := clericOf(t)
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-085, FR-093, tables F, FA, OP and SRC - the Cleric has exactly the 11 feature entries of
// table F and the Life Domain the 5 of "Cleric Subclass: Life Domain" (p. 40), each with its level, id, fact (grantsSubclass at 3, grantsFeat General at 4,
// grantsFeat Epic Boon at 19, the four Life Domain Spells grants), option ids (protector and thaumaturge; divine_strike and potent_spellcasting) and the
// key phrases of its printed text and of each option's text; the subclass level is 3 and the Life Domain cites p. 40.
// Path: per case (tables F, FA, OP): happy / unhappy (boundary: none missing, none extra).
func TestClericFeaturesAndLifeDomain(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "cleric")
	c := clericOf(t)
	checkFeatures(t, "cleric", c.Features(), o.Features)
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
	checkFeatures(t, "life_domain", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Cleric texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestClericTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := clericOf(t)
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

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100 and plan-phase-3.md 5.10 - every spell id the Life Domain Spells fact names resolves to a
// bundled spell that has Cleric in its class list, so the merged set validates.
// Path: unhappy (missing content: no dangling spell id).
func TestLifeDomainSpellsResolveToClericSpells(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, f := range clericOf(t).Subclasses()[0].Features() {
		for _, fact := range f.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				continue
			}
			for _, g := range grants.Grants() {
				for _, id := range g.SpellIDs() {
					if !slices.Contains(spellOf(t, src, string(id)).Classes(), content.ClassID("cleric")) {
						t.Errorf("%s (granted at %d) is not on the Cleric list", id, g.Level())
					}
				}
			}
		}
	}
}
