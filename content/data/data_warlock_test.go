package data_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Warlock oracle is the "warlock" entry of content/testdata/srd/classes/oracle.json, written by hand from the printed
// pages and never generated from content/data (content-classes-data.md FR-093).

func warlockOf(t *testing.T) content.ClassDefinition {
	t.Helper()
	c, ok := lookup(t, newSlice(t), content.KindCharacterClass, "warlock").(content.ClassDefinition)
	if !ok {
		t.Fatal("warlock is not a content.ClassDefinition")
	}
	return c
}

// Requirement: doc/specs/content-classes-data.md FR-085, FR-093, tables SRC, CT1 to CT3, MC, SC and class-warlock.md FR-004 to FR-011 - the Warlock's
// citations (class p. 70, Features table p. 71, Fiend Patron p. 76), Core Traits (Wisdom and Charisma saves, 2 listed skills, Simple weapons, Light
// armor training), Starting Equipment options A and B (no option C), multiclass grants (Light armor only) and the Charisma Pact Magic trait equal the
// hand-written oracle.
// Path: happy.
func TestWarlockCoreTraits(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "warlock")
	c := warlockOf(t)
	if c.Name() != o.Name || c.Source().Page() != o.ClassPage || c.Source().Section() != "Warlock" || !c.Source().CitesSRD521() || o.FeaturesTablePage != o.ClassPage+1 {
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
		t.Errorf("weapons %v armor %v tools %v multiclass %#v; want %v %v %v and Light armor only", weapons, armor, c.ToolProficiencies(), g, o.Weapons, o.ArmorTraining, o.Tools)
	}
	if sc, ok := c.Spellcasting().(content.PactSpellcastingTrait); !ok || o.Spellcasting != "pact charisma" || sc.Ability() != ability.Charisma {
		t.Errorf("Spellcasting = %#v, want %s", c.Spellcasting(), o.Spellcasting)
	}
	eq := c.StartingEquipment()
	if _, hasC := eq.C(); hasC {
		t.Error("the Warlock has no option C")
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

// Requirement: doc/specs/content-classes-data.md FR-093, tables CC, RR, SF, SQ and class-warlock.md section 4.1 (SRD 5.2.1, "Warlock Features", p. 71) -
// the Eldritch Invocations column and the 20 printed rows: Proficiency Bonus, the Class Features cell as printed (annotations of Mystic Arcanum at 11, 13, 15
// and 17, the Ability Score Improvement references at 4, 8, 12 and 16, the Subclass feature marker at 6, 10 and 14, the empty cells at 5, 7 and 18),
// Invocations, Cantrips, Prepared Spells and the Pact Magic Spell Slots and Slot Level.
// Path: per case (the 20 printed rows): happy / unhappy (boundary: first and last level; empty cells).
func TestWarlockFeaturesTable(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "warlock")
	c := warlockOf(t)
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
			sc, ok := row.Spellcasting().(content.PactSpellcastingRow)
			if !ok || row.Level() != want.Level || row.ProficiencyBonus() != want.ProficiencyBonus || sc.Cantrips() != want.Cantrips ||
				sc.PreparedSpells() != want.PreparedSpells || sc.SpellSlots() != want.PactSlots || sc.SlotLevel() != want.SlotLevel {
				t.Errorf("row %d = bonus %d, %v; want bonus %d, cantrips %d, prepared %d, %d slots of level %d", want.Level, row.ProficiencyBonus(), row.Spellcasting(),
					want.ProficiencyBonus, want.Cantrips, want.PreparedSpells, want.PactSlots, want.SlotLevel)
			}
			v, err := row.Value("eldritch_invocations")
			got := 0
			if n, isCount := v.(content.ColumnCount); isCount {
				got = n.Value()
			}
			if err != nil || got != want.EldritchInvocations {
				t.Errorf("row %d eldritch_invocations = %v (%v), want %d", want.Level, v, err, want.EldritchInvocations)
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

// Requirement: doc/specs/content-classes-data.md FR-065, FR-085, FR-093, tables F, FA, OP and SRC - the Warlock has exactly the 9 feature entries of
// table F and Fiend Patron the 5 of "Warlock Subclass: Fiend Patron" (p. 76), each with its level, id, fact (grantsSubclass at 3, grantsFeat General at 4,
// grantsFeat Epic Boon at 19, Contact Other Plane at 9, the four Fiend Spells grants), the 28 invocation option ids and the key phrases of its printed text and of
// the texts of the invocations the oracle quotes; the subclass level is 3 and Fiend Patron cites p. 76.
// Path: per case (tables F, FA, OP): happy / unhappy (boundary: none missing, none extra).
func TestWarlockFeaturesAndFiendPatron(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "warlock")
	c := warlockOf(t)
	checkFeatures(t, "warlock", c.Features(), o.Features)
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
	checkFeatures(t, "fiend_patron", sub.Features(), o.SubclassFeatures)
}

// Requirement: doc/specs/content-classes-data.md FR-061, FR-062, table OP and class-warlock.md FR-044 to FR-048 (SRD 5.2.1, "Eldritch Invocation Options",
// pp. 72-74) - the 28 invocations, in the printed alphabetical order, carry the printed prerequisites (the minimum Warlock level, the invocation required, the
// printed cantrip note) and the Repeatable flag of exactly Agonizing Blast, Eldritch Spear, Lessons of the First Ones and Repelling Blast, and none has a point cost.
// Path: per case (table OP): happy / unhappy (boundary: none missing, none extra, no prerequisite at all for five).
func TestWarlockInvocationOptionsCarryTheirPrerequisites(t *testing.T) {
	t.Parallel()
	o := classOracleOf(t, "warlock")
	var options []content.FeatureOption
	for _, f := range warlockOf(t).Features() {
		options = append(options, f.Options()...)
	}
	if len(options) != 28 || len(o.Invocations) != 28 {
		t.Fatalf("options = %d (oracle %d), want 28", len(options), len(o.Invocations))
	}
	noPrerequisite := 0
	for i, want := range o.Invocations {
		got := options[i]
		t.Run(want.ID, func(t *testing.T) {
			t.Parallel()
			var required []string
			p, has := got.Prerequisite()
			for _, id := range p.RequiresOptionIDs() {
				required = append(required, string(id))
			}
			if string(got.ID()) != want.ID || got.Name() != want.Name || got.Repeatable() != want.Repeatable || got.PointCost() != 0 ||
				p.MinClassLevel() != want.MinClassLevel || p.Note() != want.Note || !slices.Equal(required, want.Requires) &&
				(len(required) != 0 || len(want.Requires) != 0) {
				t.Errorf("option %d = %s %q repeatable %v prerequisite %v (%v); want %s %q repeatable %v level %d requires %v note %q", i, got.ID(), got.Name(),
					got.Repeatable(), p, has, want.ID, want.Name, want.Repeatable, want.MinClassLevel, want.Requires, want.Note)
			}
		})
		if _, has := got.Prerequisite(); !has {
			noPrerequisite++
		}
	}
	if noPrerequisite != 5 {
		t.Errorf("options with no prerequisite = %d, want 5 (Armor of Shadows, Eldritch Mind, Pact of the Blade, Pact of the Chain, Pact of the Tome)", noPrerequisite)
	}
}

// Requirement: doc/specs/content-classes-data.md FR-094 - the Warlock texts hold no term of another edition.
// Path: unhappy (invalid input: non-SRD terms).
func TestWarlockTextsHaveNoBannedTerms(t *testing.T) {
	t.Parallel()
	c := warlockOf(t)
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

// Requirement: doc/specs/content-classes-data.md FR-059, FR-100, plan-phase-3.md 5.10 and class-warlock.md FR-033, FR-034, FR-126 - the eleven spell ids the Contact
// Patron and Fiend Spells facts name resolve to bundled spells (so the merged set validates), and exactly Contact Other Plane and Suggestion are on the Warlock
// list, so a Warlock casts the other nine only because the feature always prepares them.
// Path: unhappy (missing content: no dangling spell id; rule interaction: a granted spell off the list).
func TestWarlockGrantedSpellsResolveAndNineAreOffTheWarlockList(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	c := warlockOf(t)
	features := slices.Concat(c.Features(), c.Subclasses()[0].Features())
	seen := 0
	var onList []string
	for _, f := range features {
		for _, fact := range f.Facts() {
			grants, ok := fact.(content.AlwaysPreparedSpells)
			if !ok {
				continue
			}
			for _, g := range grants.Grants() {
				for _, id := range g.SpellIDs() {
					seen++
					if slices.Contains(spellOf(t, src, string(id)).Classes(), content.ClassID("warlock")) {
						onList = append(onList, string(id))
					}
				}
			}
		}
	}
	slices.Sort(onList)
	if want := []string{"contact_other_plane", "suggestion"}; seen != 11 || !slices.Equal(onList, want) {
		t.Errorf("granted spells = %d, on the Warlock list = %v; want 11 and %v", seen, onList, want)
	}
}
