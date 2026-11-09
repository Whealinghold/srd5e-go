package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

func mustAbility(t *testing.T, name string) ability.Ability {
	t.Helper()
	a, err := ability.ParseAbility(name)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// Requirement: doc/specs/content-classes.md FR-016 to FR-029 (Wizard rows of the oracle tables CT1 to CT3, MC, SC of
// content-classes-data.md) and doc/specs/content-thin-slice.md section 1 ("Wizard") - the Wizard core traits: primary
// ability, Hit Point Die, saving throws, skills, weapon, armor and tool training, starting equipment, no multiclass
// grants, Intelligence slot spellcasting with cantrips and no class-specific columns.
// Path: happy.
func TestWizardCoreTraits(t *testing.T) {
	t.Parallel()
	o := readOracle[wizardOracle](t, "wizard")
	c := wizardOf(t, newSlice(t))
	if all, ok := c.PrimaryAbility().(content.AllOf); !ok || !slices.Equal(all.Abilities(), []ability.Ability{mustAbility(t, o.PrimaryAbility)}) {
		t.Errorf("PrimaryAbility = %#v, want AllOf %s", c.PrimaryAbility(), o.PrimaryAbility)
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
	weapons := c.WeaponProficiencies()
	if cat, err := content.ParseWeaponTrainingCategory(o.Weapons[0]); err != nil || len(weapons) != 1 || weapons[0].Category() != cat || len(weapons[0].RequiresAnyProperty()) != 0 {
		t.Errorf("WeaponProficiencies = %#v, want every %s weapon", weapons, o.Weapons[0])
	}
	if len(c.ArmorTraining()) != 0 || len(c.ToolProficiencies()) != 0 || len(o.ArmorTraining) != 0 {
		t.Errorf("armor %v and tools %v, want none", c.ArmorTraining(), c.ToolProficiencies())
	}
	g := c.MulticlassGrants()
	if len(g.WeaponProficiencies())+len(g.ArmorTraining())+len(g.ToolProficiencies()) != 0 || g.SkillChoice() != nil {
		t.Errorf("MulticlassGrants = %#v, want none", g)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || sc.Ability() != mustAbility(t, o.SpellcastingAbility) || !sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want slots with %s and cantrips", c.Spellcasting(), o.SpellcastingAbility)
	}
	if len(c.Columns()) != 0 || len(c.Subclasses()) != 1 {
		t.Errorf("columns %v and subclasses %v, want none and the Evoker", c.Columns(), c.Subclasses())
	}
}

// Requirement: doc/specs/content-classes.md FR-024, FR-025 - the starting equipment options of the Wizard are stored as
// printed: counts split off ("2 Daggers" is Dagger x2), a parenthetical kept in the name, GP as given.
// Path: happy / unhappy (boundary: singular names and parentheticals).
func TestWizardStartingEquipment(t *testing.T) {
	t.Parallel()
	o := readOracle[wizardOracle](t, "wizard")
	got := wizardOf(t, newSlice(t)).StartingEquipment()
	type option struct {
		label string
		got   content.ClassEquipmentOption
		want  equipmentOracle
	}
	for _, c := range []option{{"A", got.A(), o.EquipmentA}, {"B", got.B(), o.EquipmentB}} {
		if c.got.Gold() != c.want.Gold || len(c.got.Items()) != len(c.want.Items) {
			t.Errorf("option %s = %d GP, %d items; want %d GP, %d items", c.label, c.got.Gold(), len(c.got.Items()), c.want.Gold, len(c.want.Items))
			continue
		}
		for i, it := range c.got.Items() {
			if it.Name() != c.want.Items[i][0].(string) || it.Quantity() != int(c.want.Items[i][1].(float64)) {
				t.Errorf("option %s item %d = %s x%d, want %v", c.label, i, it.Name(), it.Quantity(), c.want.Items[i])
			}
		}
	}
	if _, hasC := got.C(); hasC {
		t.Error("the Wizard has no option C")
	}
}

// Requirement: doc/specs/content-classes.md FR-044, FR-045, FR-046, FR-051, FR-052, FR-056, FR-057, FR-060, FR-102
// (tables F, RR, SF, FA of content-classes-data.md) - the Wizard has exactly the ten feature entries at their levels,
// with the structured facts of table FA and no other fact.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra).
func TestWizardFeatures(t *testing.T) {
	t.Parallel()
	o := readOracle[wizardOracle](t, "wizard")
	c := wizardOf(t, newSlice(t))
	feats := c.Features()
	if len(feats) != len(o.Features) || len(feats) != 10 {
		t.Fatalf("Wizard has %d feature entries, want 10", len(feats))
	}
	for i, want := range o.Features {
		f := feats[i]
		wantID := "wizard_" + strings.NewReplacer(" ", "_").Replace(strings.ToLower(want.Name))
		if f.Name() != want.Name || f.Level() != want.Level || string(f.ID()) != wantID {
			t.Errorf("feature %d = %q level %d id %s; want %q level %d id %s", i, f.Name(), f.Level(), f.ID(), want.Name, want.Level, wantID)
		}
		var gotFact string
		for _, fact := range f.Facts() {
			switch v := fact.(type) {
			case content.GrantsSubclass:
				gotFact += "grantsSubclass"
			case content.GrantsFeat:
				gotFact += "grantsFeat " + v.Category().String()
			default:
				t.Errorf("feature %s has the unexpected fact %#v", f.Name(), fact)
			}
		}
		if gotFact != want.Fact || len(f.Options()) != 0 || len(f.Tables()) != 0 {
			t.Errorf("feature %s facts = %q, options %d, tables %d; want %q and no options or tables", f.Name(), gotFact, len(f.Options()), len(f.Tables()), want.Fact)
		}
	}
	if c.SubclassLevel() != 3 {
		t.Errorf("SubclassLevel = %d, want 3", c.SubclassLevel())
	}
}

// Requirement: doc/specs/content-classes.md FR-031 to FR-034, FR-037, FR-038, FR-043, FR-046 (tables FS, SQ, RR, SF) -
// the 20 progression rows equal the printed Wizard Features table: Proficiency Bonus, Class Features cell (including
// the Ability Score Improvement references at 8, 12 and 16 and the subclass markers at 6, 10 and 14), Cantrips,
// Prepared Spells and the nine spell slot counts, with a printed dash stored as 0.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestWizardFeaturesTable(t *testing.T) {
	t.Parallel()
	o := readOracle[wizardOracle](t, "wizard")
	c := wizardOf(t, newSlice(t))
	if len(c.Progression()) != 20 || len(o.PrintedRows) != 20 {
		t.Fatalf("rows = %d (oracle %d), want 20", len(c.Progression()), len(o.PrintedRows))
	}
	for i, printed := range o.PrintedRows {
		t.Run("level "+strconv.Itoa(i+1), func(t *testing.T) {
			t.Parallel()
			tok := strings.Fields(printed)
			n := len(tok)
			level, _ := strconv.Atoi(tok[0])
			bonus, _ := strconv.Atoi(strings.TrimPrefix(tok[1], "+"))
			cell := strings.Join(tok[2:n-11], " ")
			cantrips, _ := strconv.Atoi(tok[n-11])
			prepared, _ := strconv.Atoi(tok[n-10])
			var slots [9]int
			for j := range slots {
				if v := tok[n-9+j]; v != "—" {
					slots[j], _ = strconv.Atoi(v)
				}
			}
			row, err := c.RowAt(level)
			if err != nil {
				t.Fatal(err)
			}
			sc, ok := row.Spellcasting().(content.SlotSpellcastingRow)
			gotCantrips, hasCantrips := sc.Cantrips()
			if !ok || row.Level() != level || row.ProficiencyBonus() != bonus || !hasCantrips || gotCantrips != cantrips || sc.PreparedSpells() != prepared || sc.SpellSlots() != slots {
				t.Errorf("row %d = bonus %d, %v; want bonus %d, cantrips %d, prepared %d, slots %v", level, row.ProficiencyBonus(), row.Spellcasting(), bonus, cantrips, prepared, slots)
			}
			resolved, err := c.RowFeatures(level)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, e := range resolved.Entries() {
				names = append(names, e.Name())
			}
			gotCell := strings.Join(names, ", ")
			switch {
			case resolved.HasSubclassFeature():
				gotCell = "Subclass feature"
			case gotCell == "":
				gotCell = "—"
			}
			if gotCell != cell {
				t.Errorf("row %d Class Features = %q, want %q", level, gotCell, cell)
			}
		})
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-062 and content-classes.md FR-054 - the Wizard Spellcasting text
// names the recommended cantrips Light, Mage Hand and Ray of Frost as text only: the Wizard holds no reference site
// and the slice holds no such spell.
// Path: unhappy (invalid input: prose is not data).
func TestWizardSpellcastingNamesAreText(t *testing.T) {
	t.Parallel()
	o := readOracle[wizardOracle](t, "wizard")
	src := newSlice(t)
	c := wizardOf(t, src)
	var text string
	for _, f := range c.Features() {
		if f.Name() == "Spellcasting" {
			text = f.Text()
		}
	}
	for _, p := range o.SpellcastingPhrases {
		if !strings.Contains(text, p) {
			t.Errorf("Spellcasting text lacks %q", p)
		}
	}
	if refs := c.References(); len(refs) != 0 {
		t.Errorf("Wizard references = %v, want none", refs)
	}
}

// The Wizard class oracle is the "wizard" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093). Step 9e replaces the slice entry with the
// full Wizard; step 9f adds the Evoker subclass (p. 82).

func wizardClassOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	return wizardOf(t, newSlice(t))
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-wizard.md FR-019 to FR-021 -
// the Wizard's citation (class p. 77 with its Features table on the same page), Core Traits (Intelligence, D6, Intelligence and Wisdom
// saves, 2 of 7 listed skills, Simple weapons, no armor or tools), Starting Equipment options A and B (no option C), no multiclass
// grants and slot spellcasting with Intelligence and cantrips equal the hand-written oracle.
// Path: happy.
func TestWizardClassOracleCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "wizard")
	c := wizardClassOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Wizard" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage {
		t.Errorf("citation = %q p. %d, want %q p. %d", c.Source().Section(), c.Source().Page(), o.Name, o.ClassPage)
	}
	if all, ok := c.PrimaryAbility().(content.AllOf); !ok || o.PrimaryAbility.Kind != "allOf" || !slices.Equal(all.Abilities(), []ability.Ability{mustAbility(t, o.PrimaryAbility.Abilities[0])}) {
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
		if len(w.RequiresAnyProperty()) != 0 {
			t.Errorf("%s weapons require a property", w.Category())
		}
		weapons = append(weapons, w.Category().String())
	}
	g := c.MulticlassGrants()
	if !slices.Equal(weapons, o.Weapons) || len(c.ArmorTraining()) != len(o.ArmorTraining) || len(c.ToolProficiencies()) != len(o.Tools) ||
		len(g.WeaponProficiencies())+len(g.ArmorTraining())+len(g.ToolProficiencies()) != 0 || g.SkillChoice() != nil || o.Multiclass.Skills != "none" {
		t.Errorf("weapons %v armor %v tools %v multiclass %#v; want %v and nothing else", weapons, c.ArmorTraining(), c.ToolProficiencies(), g, o.Weapons)
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || o.Spellcasting != "slots intelligence cantrips" || sc.Ability() != ability.Intelligence || !sc.Cantrips() {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Wizard has no option C")
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
	if len(c.Columns()) != len(o.Columns) || len(c.Subclasses()) != 1 || o.Subclass.ID != "evoker" {
		t.Errorf("columns %v and subclasses %v, want no columns and the Evoker", c.Columns(), c.Subclasses())
	}
}

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ and class-wizard.md FR-018 (SRD 5.2.1, "Wizard Features", p. 77) -
// the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (including the Ability Score Improvement references at 4, 8, 12 and 16
// and the Subclass feature marker at 6, 10 and 14), Cantrips, Prepared Spells and the nine Spell Slot counts, with a printed dash stored as 0.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestWizardClassOracleFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "wizard")
	c := wizardClassOf(t)
	if len(c.Progression()) != 20 || len(o.Rows) != 20 {
		t.Fatalf("rows = %d (oracle %d), want 20", len(c.Progression()), len(o.Rows))
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

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables F, FA and SRC - the Wizard has exactly the 10 feature entries of table F, each with its level, id, fact (grantsSubclass at 3, grantsFeat
// General at 4, grantsFeat Epic Boon at 19) and the key phrases of its printed text; the subclass level is 3; and the texts hold no term of another edition.
// Path: per case (tables F, FA): happy / unhappy (boundary: none missing, none extra; invalid input: non-SRD terms).
func TestWizardClassOracleFeatures(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "wizard")
	c := wizardClassOf(t)
	checkFeatures(t, "wizard", c.Features(), o.Features)
	if c.SubclassLevel() != 3 {
		t.Errorf("SubclassLevel = %d, want 3", c.SubclassLevel())
	}
	for _, f := range c.Features() {
		for _, term := range bannedTerms() {
			if containsWord(f.Text(), term) || containsWord(f.Name(), term) {
				t.Errorf("%s contains the banned term %q", f.Name(), term)
			}
		}
	}
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables F, FA and SRC - the Wizard has exactly one subclass, the Evoker, citing
// "Wizard Subclass: Evoker" on p. 82 with the 5 features of table FA (Evocation Savant and Potent Cantrip at 3, Sculpt Spells at 6, Empowered
// Evocation at 10, Overchannel at 14), each with the key phrases of its printed text and no term of another edition.
// Path: per case (table FA): happy / unhappy (boundary: none missing, none extra; invalid input: non-SRD terms).
func TestWizardEvokerOracle(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "wizard")
	c := wizardClassOf(t)
	if len(c.Subclasses()) != 1 {
		t.Fatalf("subclasses = %d, want 1", len(c.Subclasses()))
	}
	sub := c.Subclasses()[0]
	if string(sub.ID()) != o.Subclass.ID || sub.Name() != o.Subclass.Name || sub.Source().Section() != o.Subclass.Section ||
		sub.Source().Page() != o.Subclass.Page || !sub.Source().CitesSRD521() {
		t.Errorf("subclass = %s %q %q p. %d; want %s %q %q p. %d", sub.ID(), sub.Name(), sub.Source().Section(), sub.Source().Page(),
			o.Subclass.ID, o.Subclass.Name, o.Subclass.Section, o.Subclass.Page)
	}
	checkFeatures(t, "evoker", sub.Features(), o.SubclassFeatures)
	for _, f := range sub.Features() {
		for _, term := range bannedTerms() {
			if containsWord(f.Text(), term) || containsWord(f.Name(), term) {
				t.Errorf("%s contains the banned term %q", f.Name(), term)
			}
		}
	}
}
