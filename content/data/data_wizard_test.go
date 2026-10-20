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
	if len(c.Columns()) != 0 || len(c.Subclasses()) != 0 {
		t.Errorf("columns %v and subclasses %v, want none until Evoker lands", c.Columns(), c.Subclasses())
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
