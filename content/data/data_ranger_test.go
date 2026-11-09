package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Ranger oracle is the "ranger" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func rangerOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "ranger").(content.ClassDefinition)
	if !ok {
		t.Fatal("ranger is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-ranger.md - the Ranger's
// citations (class p. 57, Features table p. 58, Hunter p. 61), Core Traits, training (Simple and Martial weapons, Light and Medium armor and
// Shields), Starting Equipment options A and B (no option C), multiclass grants (Martial weapons, Light and Medium armor and Shields, one listed skill) and the
// Wisdom slot spellcasting with no Cantrips column equal the hand-written oracle.
// Path: happy.
func TestRangerCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "ranger")
	c := rangerOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Ranger" || !c.Source().CitesSRD521() || o.FeaturesTablePage != 58 {
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
	mcSkills, mcSkillsOK := g.SkillChoice().(content.SkillChoiceListed)
	for _, w := range g.WeaponProficiencies() {
		mcWeapons = append(mcWeapons, w.Category().String())
	}
	for _, a := range g.ArmorTraining() {
		mcArmor = append(mcArmor, a.String())
	}
	if !slices.Equal(weapons, o.Weapons) || !slices.Equal(armor, o.ArmorTraining) || len(c.ToolProficiencies()) != len(o.Tools) ||
		!slices.Equal(mcWeapons, o.Multiclass.Weapons) || !slices.Equal(mcArmor, o.Multiclass.ArmorTraining) ||
		len(g.ToolProficiencies()) != 0 || !mcSkillsOK || mcSkills.Picks() != 1 || !slices.Equal(mcSkills.Skills(), rogueSkills(t, o.Skills.Options)) || o.Multiclass.Skills != "listed, 1 pick" {
		t.Errorf("weapons %v armor %v tools %v multiclass %#v; want %v %v %v and Martial weapons with Light, Medium and Shield", weapons, armor, c.ToolProficiencies(), g, o.Weapons, o.ArmorTraining, o.Tools)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots wisdom" || sc.Ability() != ability.Wisdom || sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Ranger has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ, FS and class-ranger.md FR-009, FR-010, FR-012 (SRD 5.2.1, "Ranger
// Features", p. 58) - the Favored Enemy column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (the Ability
// Score Improvement references at 4, 8, 12 and 16, the Subclass feature marker at 7, 15 and 20, the empty cells at 13 and 17), Favored Enemy
// (a printed dash at levels 1 and 2), no Cantrips column, Prepared Spells and the nine Spell Slot counts.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestRangerFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "ranger")
	c := rangerOf(t)
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
			v, err := row.Value("favored_enemy")
			n, isCount := v.(content.ColumnCount)
			if err != nil || !isCount || n.Value() != want.FavoredEnemy {
				t.Errorf("row %d favored_enemy = %v (%v), want %d", want.Level, v, err, want.FavoredEnemy)
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-085, FR-093, tables F, FA, OP and SRC - the Ranger has exactly the 17 feature entries of table F and
// the Hunter the 5 of "Ranger Subclass: Hunter" (p. 61), each with its level, id, fact (grantsSubclass at 3, grantsFeat General at 4, grantsFeat Fighting Style at 2,
// grantsFeat Epic Boon at 19, the Hunter's Mark grant of Favored Enemy), option ids (colossus_slayer and horde_breaker; escape_the_horde and multiattack_defense) and
// the key phrases of its printed text and of each option's text; the subclass level is 3 and the Hunter cites p. 61.
// Path: per case (tables F, FA, OP): happy / unhappy (boundary: none missing, none extra).
func TestRangerFeaturesAndHunter(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "ranger")
	c := rangerOf(t)
	checkFeatures(t, "ranger", c.Features(), o.Features)
	if c.SubclassLevel() != 3 || len(c.Subclasses()) != 1 {
		t.Fatalf("SubclassLevel = %d, subclasses = %d; want 3 and 1", c.SubclassLevel(), len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "hunter", sub.Features(), o.SubclassFeatures)
	for _, f := range sub.Features() {
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
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Ranger texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestRangerTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := rangerOf(t)
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

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100, class-ranger.md FR-031 and plan-phase-3.md 5.10 - the one spell id the Favored Enemy fact names
// resolves to a bundled spell (so the merged set validates) and that spell has Ranger in its class list (SRD 5.2.1, "Ranger Spell List", p. 60).
// Path: unhappy (missing content: no dangling spell id).
func TestRangerGrantedSpellResolvesAndIsOnTheRangerList(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	c := rangerOf(t)
	seen := 0
	for _, f := range slices.Concat(c.Features(), c.Subclasses()[0].Features()) {
		for _, fact := range f.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				continue
			}
			for _, g := range grants.Grants() {
				for _, id := range g.SpellIDs() {
					seen++
					if !slices.Contains(spellOf(t, src, string(id)).Classes(), content.ClassID("ranger")) || id != "hunters_mark" || g.Level() != 1 {
						t.Errorf("grant %s at %d is not Hunter's Mark at 1 on the Ranger list", id, g.Level())
					}
				}
			}
		}
	}
	if seen != 1 {
		t.Errorf("granted spells = %d, want 1", seen)
	}
}
