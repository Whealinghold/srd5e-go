package content_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type blockCase struct {
	name  string
	err   error
	block string
	want  content.BlockProblem
}

func abilities(as ...ability.Ability) []ability.Ability { return as }

// Requirement: doc/specs/content-classes.md FR-011 to FR-015, Cases AR - IsMetBy
// for every row, the printed order, copy on read and the boundaries of both
// constructors.
// Path: per case.
func TestAbilityRequirement(t *testing.T) {
	t.Parallel()
	str, dex, con, wis, cha := ability.Strength, ability.Dexterity, ability.Constitution, ability.Wisdom, ability.Charisma
	allOf := func(as ...ability.Ability) content.AbilityRequirement { return mustValue(content.NewAllOf(as...)) }
	anyOf := func(as ...ability.Ability) content.AbilityRequirement { return mustValue(content.NewAnyOf(as...)) }
	cases := []struct {
		name    string
		req     content.AbilityRequirement
		meeting []ability.Ability
		want    bool
	}{
		{"single, met", allOf(str), abilities(str), true},
		{"single, not met", allOf(str), nil, false},
		{"both required, one met", allOf(dex, wis), abilities(dex), false},
		{"both required, both met", allOf(dex, wis), abilities(wis, dex), true},
		{"either, first", anyOf(str, dex), abilities(str), true},
		{"either, second", anyOf(str, dex), abilities(dex), true},
		{"either, neither", anyOf(str, dex), abilities(con), false},
		{"either, none", anyOf(str, dex), nil, false},
		{"extra ignored", allOf(cha), ability.All(), true},
	}
	for _, c := range cases {
		if got := c.req.IsMetBy(c.meeting); got != c.want {
			t.Errorf("%s: IsMetBy = %v, want %v [%s FR-014, per case]", c.name, got, c.want, classSpec)
		}
	}
	if (content.AllOf{}).IsMetBy(ability.All()) || (content.AnyOf{}).IsMetBy(ability.All()) {
		t.Error("a zero requirement is met [unhappy (empty case)]")
	}
	r := allOf(dex, wis)
	got := r.Abilities()
	got[0] = cha
	if !slices.Equal(r.Abilities(), abilities(dex, wis)) {
		t.Error("mutating Abilities() changed the requirement [unhappy (immutability)]")
	}
	in := abilities(dex, wis)
	r = allOf(in...)
	in[0] = cha
	if !slices.Equal(r.Abilities(), abilities(dex, wis)) {
		t.Error("mutating the input changed the requirement [unhappy (immutability)]")
	}
	six := ability.All()
	for _, c := range []struct {
		name  string
		err   error
		block string
		want  content.BlockProblem
	}{
		{"AllOf none", errOf(content.NewAllOf()), "AllOf", content.BlockOutOfRange},
		{"AllOf seven", errOf(content.NewAllOf(append(six, str)...)), "AllOf", content.BlockOutOfRange},
		{"AllOf repeated", errOf(content.NewAllOf(str, str)), "AllOf", content.BlockDuplicate},
		{"AllOf invalid", errOf(content.NewAllOf(ability.Ability(0))), "AllOf", content.BlockInvalidMember},
		{"AnyOf one", errOf(content.NewAnyOf(str)), "AnyOf", content.BlockOutOfRange},
		{"AnyOf none", errOf(content.NewAnyOf()), "AnyOf", content.BlockOutOfRange},
		{"AnyOf repeated", errOf(content.NewAnyOf(str, str)), "AnyOf", content.BlockDuplicate},
		{"AnyOf invalid", errOf(content.NewAnyOf(str, ability.Ability(99))), "AnyOf", content.BlockInvalidMember},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
	for _, ok := range []error{errOf(content.NewAllOf(six...)), errOf(content.NewAnyOf(str, dex))} {
		if ok != nil {
			t.Errorf("valid requirement rejected: %v [happy]", ok)
		}
	}
}

// mustValue returns v and panics on a non-nil error, for fixtures.
func mustValue[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// errOf drops the value of a constructor result.
func errOf[T any](_ T, err error) error { return err }

// Requirement: doc/specs/content-classes.md FR-019, FR-021 to FR-024 - the
// training and equipment constructors: happy values, each rejection by block
// and problem, copy on read.
// Path: per case.
func TestClassTrainingConstructors(t *testing.T) {
	t.Parallel()
	arcana, history := content.SkillArcana, content.SkillHistory
	item := mustValue(content.NewClassEquipmentItem("Staff", 1))
	opt := mustValue(content.NewClassEquipmentOption([]content.ClassEquipmentItem{item}, 5))
	for _, c := range []blockCase{
		{"listed picks 0", errOf(content.NewSkillChoiceListed(0, []content.Skill{arcana})), "SkillChoiceListed", content.BlockOutOfRange},
		{"listed too few", errOf(content.NewSkillChoiceListed(2, []content.Skill{arcana})), "SkillChoiceListed", content.BlockOutOfRange},
		{"listed duplicate", errOf(content.NewSkillChoiceListed(1, []content.Skill{arcana, arcana})), "SkillChoiceListed", content.BlockDuplicate},
		{"listed invalid", errOf(content.NewSkillChoiceListed(1, []content.Skill{0})), "SkillChoiceListed", content.BlockInvalidMember},
		{"listed picks beyond cap", errOf(content.NewSkillChoiceListed(over(), []content.Skill{arcana})), "SkillChoiceListed", content.BlockOutOfRange},
		{"any picks 0", errOf(content.NewSkillChoiceAny(0)), "SkillChoiceAny", content.BlockOutOfRange},
		{"any picks beyond cap", errOf(content.NewSkillChoiceAny(over())), "SkillChoiceAny", content.BlockOutOfRange},
		{"weapon invalid category", errOf(content.NewWeaponProficiency(0, nil)), "WeaponProficiency", content.BlockInvalidMember},
		{"weapon empty filter", errOf(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{})), "WeaponProficiency", content.BlockOutOfRange},
		{"weapon repeated filter", errOf(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{content.WeaponPropertyLight, content.WeaponPropertyLight})), "WeaponProficiency", content.BlockDuplicate},
		{"specific blank", errOf(content.NewClassToolProficiencySpecific(" ")), "ClassToolProficiencySpecific", content.BlockInvalidName},
		{"choice none", errOf(content.NewClassToolProficiencyChoice(nil, 1)), "ClassToolProficiencyChoice", content.BlockEmpty},
		{"choice blank", errOf(content.NewClassToolProficiencyChoice([]string{" "}, 1)), "ClassToolProficiencyChoice", content.BlockInvalidMember},
		{"choice repeated", errOf(content.NewClassToolProficiencyChoice([]string{"A", "A"}, 1)), "ClassToolProficiencyChoice", content.BlockDuplicate},
		{"choice picks 0", errOf(content.NewClassToolProficiencyChoice([]string{"A"}, 0)), "ClassToolProficiencyChoice", content.BlockOutOfRange},
		{"item blank name", errOf(content.NewClassEquipmentItem("", 1)), "ClassEquipmentItem", content.BlockInvalidName},
		{"item quantity 0", errOf(content.NewClassEquipmentItem("A", 0)), "ClassEquipmentItem", content.BlockOutOfRange},
		{"option invalid item", errOf(content.NewClassEquipmentOption([]content.ClassEquipmentItem{{}}, 0)), "ClassEquipmentOption", content.BlockInvalidMember},
		{"option negative gold", errOf(content.NewClassEquipmentOption(nil, -1)), "ClassEquipmentOption", content.BlockOutOfRange},
		{"equipment two c", errOf(content.NewClassStartingEquipment(opt, opt, opt, opt)), "ClassStartingEquipment", content.BlockOutOfRange},
		{"grants invalid weapon", errOf(content.NewMulticlassGrants([]content.WeaponProficiency{{}}, nil, nil, nil)), "MulticlassGrants", content.BlockInvalidMember},
		{"grants repeated weapon", errOf(content.NewMulticlassGrants([]content.WeaponProficiency{mustValue(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil)), mustValue(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil))}, nil, nil, nil)), "MulticlassGrants", content.BlockDuplicate},
		{"grants repeated armor", errOf(content.NewMulticlassGrants(nil, []content.ArmorTrainingCategory{content.ArmorTrainingLight, content.ArmorTrainingLight}, nil, nil)), "MulticlassGrants", content.BlockDuplicate},
		{"grants invalid tool", errOf(content.NewMulticlassGrants(nil, nil, []content.ClassToolProficiency{content.ClassToolProficiencyChoice{}}, nil)), "MulticlassGrants", content.BlockInvalidMember},
		{"grants nil tool", errOf(content.NewMulticlassGrants(nil, nil, []content.ClassToolProficiency{nil}, nil)), "MulticlassGrants", content.BlockInvalidMember},
		{"grants invalid skills", errOf(content.NewMulticlassGrants(nil, nil, nil, content.SkillChoiceAny{})), "MulticlassGrants", content.BlockInvalidMember},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
	// Happy values and copy on read.
	skills := []content.Skill{arcana, history}
	listed := mustValue(content.NewSkillChoiceListed(1, skills))
	skills[0] = content.SkillStealth
	got := listed.Skills()
	got[1] = content.SkillStealth
	if listed.Picks() != 1 || !slices.Equal(listed.Skills(), []content.Skill{arcana, history}) {
		t.Errorf("SkillChoiceListed = %v after mutation [unhappy (immutability)]", listed.Skills())
	}
	if mustValue(content.NewSkillChoiceAny(3)).Picks() != 3 {
		t.Error("SkillChoiceAny picks lost [happy]")
	}
	w := mustValue(content.NewWeaponProficiency(content.WeaponTrainingMartial, []content.WeaponProperty{content.WeaponPropertyLight}))
	if w.Category() != content.WeaponTrainingMartial || !slices.Equal(w.RequiresAnyProperty(), []content.WeaponProperty{content.WeaponPropertyLight}) {
		t.Errorf("WeaponProficiency = %+v [happy]", w)
	}
	if mustValue(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil)).RequiresAnyProperty() != nil {
		t.Error("an unfiltered training has a filter [happy]")
	}
	c := mustValue(content.NewClassToolProficiencyChoice([]string{"A", "B"}, 2))
	if !slices.Equal(c.Categories(), []string{"A", "B"}) || c.Picks() != 2 || mustValue(content.NewClassToolProficiencySpecific("Kit")).Name() != "Kit" {
		t.Errorf("tool proficiencies = %+v [happy]", c)
	}
	if item.Name() != "Staff" || item.Quantity() != 1 || opt.Gold() != 5 || len(opt.Items()) != 1 {
		t.Errorf("equipment = %+v %+v [happy]", item, opt)
	}
	eq := mustValue(content.NewClassStartingEquipment(opt, opt))
	if _, hasC := eq.C(); hasC || eq.A().Gold() != 5 || eq.B().Gold() != 5 {
		t.Errorf("equipment A/B = %+v [happy]", eq)
	}
	eq = mustValue(content.NewClassStartingEquipment(opt, opt, opt))
	if c, hasC := eq.C(); !hasC || c.Gold() != 5 {
		t.Errorf("equipment C = %+v [happy]", eq)
	}
	g := mustValue(content.NewMulticlassGrants([]content.WeaponProficiency{w}, []content.ArmorTrainingCategory{content.ArmorTrainingShield}, []content.ClassToolProficiency{c}, listed))
	if len(g.WeaponProficiencies()) != 1 || len(g.ArmorTraining()) != 1 || len(g.ToolProficiencies()) != 1 || g.SkillChoice() == nil {
		t.Errorf("grants = %+v [happy]", g)
	}
}

// Requirement: doc/specs/content-classes.md FR-019, FR-021, FR-022 - the three
// training vocabularies parse exactly and case sensitively.
// Path: per case.
func TestClassVocabularies(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		ok   []string
		bad  []string
		p    func(string) (string, error)
	}{
		{"WeaponTrainingCategory", []string{"Simple", "Martial"}, []string{"simple", "Martial ", "", "Ranged"}, func(s string) (string, error) {
			v, err := content.ParseWeaponTrainingCategory(s)
			return v.String(), err
		}},
		{"ArmorTrainingCategory", []string{"Light", "Medium", "Heavy", "Shield"}, []string{"light", "None", " Shield"}, func(s string) (string, error) {
			v, err := content.ParseArmorTrainingCategory(s)
			return v.String(), err
		}},
		{"ClassColumnKind", []string{"count", "dice", "bonus", "feetBonus"}, []string{"Count", "feet", "feetbonus"}, func(s string) (string, error) {
			v, err := content.ParseClassColumnKind(s)
			return v.String(), err
		}},
	} {
		for _, s := range c.ok {
			if got, err := c.p(s); err != nil || got != s {
				t.Errorf("%s(%q) = (%q, %v), want it back [happy]", c.name, s, got, err)
			}
		}
		for _, s := range c.bad {
			if _, err := c.p(s); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s(%q) = %v, want ErrInvalidInput [unhappy (boundary)]", c.name, s, err)
			}
		}
	}
	for _, v := range []interface{ Valid() bool }{content.WeaponTrainingCategory(0), content.WeaponTrainingCategory(3), content.ArmorTrainingCategory(0), content.ArmorTrainingCategory(5), content.ClassColumnKind(0), content.ClassColumnKind(5)} {
		if v.Valid() {
			t.Errorf("%v is valid [unhappy (boundary)]", v)
		}
	}
	if content.WeaponTrainingCategory(0).String() != "unknown" || content.ArmorTrainingCategory(9).String() != "unknown" || content.ClassColumnKind(9).String() != "unknown" {
		t.Error("an undefined value does not print unknown [unhappy (boundary)]")
	}
}

// Requirement: doc/specs/content-classes.md FR-028, FR-031, FR-032 - the
// spellcasting trait and row constructors.
// Path: per case.
func TestClassSpellcastingConstructors(t *testing.T) {
	t.Parallel()
	nine := make([]int, 9)
	for _, c := range []blockCase{
		{"slot trait invalid ability", errOf(content.NewSlotSpellcastingTrait(0, true)), "SlotSpellcastingTrait", content.BlockInvalidMember},
		{"pact trait invalid ability", errOf(content.NewPactSpellcastingTrait(0)), "PactSpellcastingTrait", content.BlockInvalidMember},
		{"eight slots", errOf(content.NewSlotSpellcastingRow(1, nine[:8])), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"ten slots", errOf(content.NewSlotSpellcastingRow(1, append(slices.Clone(nine), 0))), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"negative prepared", errOf(content.NewSlotSpellcastingRow(-1, nine)), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"negative cantrips", errOf(content.NewSlotSpellcastingRowWithCantrips(-1, 1, nine)), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"negative slot", errOf(content.NewSlotSpellcastingRow(1, []int{0, 0, 0, 0, 0, 0, 0, 0, -1})), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"pact slot level 0", errOf(content.NewPactSpellcastingRow(1, 1, 1, 0)), "PactSpellcastingRow", content.BlockOutOfRange},
		{"pact slot level 10", errOf(content.NewPactSpellcastingRow(1, 1, 1, 10)), "PactSpellcastingRow", content.BlockOutOfRange},
		{"pact slots 0", errOf(content.NewPactSpellcastingRow(1, 1, 0, 1)), "PactSpellcastingRow", content.BlockOutOfRange},
		{"pact negative cantrips", errOf(content.NewPactSpellcastingRow(-1, 1, 1, 1)), "PactSpellcastingRow", content.BlockOutOfRange},
		{"pact negative prepared", errOf(content.NewPactSpellcastingRow(1, -1, 1, 1)), "PactSpellcastingRow", content.BlockOutOfRange},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
	slot := mustValue(content.NewSlotSpellcastingTrait(ability.Intelligence, true))
	pact := mustValue(content.NewPactSpellcastingTrait(ability.Charisma))
	if slot.Ability() != ability.Intelligence || !slot.Cantrips() || pact.Ability() != ability.Charisma {
		t.Errorf("traits = %+v %+v [happy]", slot, pact)
	}
	row := mustValue(content.NewSlotSpellcastingRowWithCantrips(3, 4, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}))
	if n, ok := row.Cantrips(); n != 3 || !ok || row.PreparedSpells() != 4 || row.SpellSlots() != [9]int{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		t.Errorf("slot row = %+v [happy]", row)
	}
	if n, ok := mustValue(content.NewSlotSpellcastingRow(4, nine)).Cantrips(); n != 0 || ok {
		t.Error("a row built without cantrips has a Cantrips column [happy]")
	}
	p := mustValue(content.NewPactSpellcastingRow(2, 3, 4, 5))
	if p.Cantrips() != 2 || p.PreparedSpells() != 3 || p.SpellSlots() != 4 || p.SlotLevel() != 5 {
		t.Errorf("pact row = %+v [happy]", p)
	}
	if mustValue(content.NewPactSpellcastingRow(0, 0, 1, 9)).SlotLevel() != 9 {
		t.Error("slot level 9 rejected [unhappy (boundary)]")
	}
}

// Requirement: doc/specs/content-classes.md FR-036, FR-040, FR-043, Cases CV -
// column and row constructors, the kinds a value fits and the "—" text.
// Path: per case.
func TestClassColumnsAndRows(t *testing.T) {
	t.Parallel()
	for _, c := range []blockCase{
		{"count -1", errOf(content.NewColumnCount(-1)), "ColumnCount", content.BlockOutOfRange},
		{"count beyond cap", errOf(content.NewColumnCount(over())), "ColumnCount", content.BlockOutOfRange},
		{"dice count 0", errOf(content.NewColumnDice(0, 6)), "ColumnDice", content.BlockOutOfRange},
		{"dice one side", errOf(content.NewColumnDice(1, 1)), "ColumnDice", content.BlockOutOfRange},
		{"dice beyond the cap", errOf(content.NewColumnDice(over(), 6)), "ColumnDice", content.BlockOutOfRange},
		{"bonus -1", errOf(content.NewColumnBonus(-1)), "ColumnBonus", content.BlockOutOfRange},
		{"feet -1", errOf(content.NewColumnFeetBonus(-1)), "ColumnFeetBonus", content.BlockOutOfRange},
		{"column blank header", errOf(content.NewClassColumn("a", " ", content.ColumnKindCount)), "ClassColumn", content.BlockBlankText},
		{"column multi-line header", errOf(content.NewClassColumn("a", "A\nB", content.ColumnKindCount)), "ClassColumn", content.BlockInvalidCharacter},
		{"column kind", errOf(content.NewClassColumn("a", "A", 0)), "ClassColumn", content.BlockInvalidMember},
		{"row feature blank annotation", errOf(content.NewRowClassFeature("a", " ")), "RowClassFeature", content.BlockBlankText},
		{"row level 0", errOf(content.NewProgressionRow(0, 2, nil, nil, nil)), "ProgressionRow", content.BlockOutOfRange},
		{"row level 21", errOf(content.NewProgressionRow(21, 2, nil, nil, nil)), "ProgressionRow", content.BlockOutOfRange},
		{"row bonus 0", errOf(content.NewProgressionRow(1, 0, nil, nil, nil)), "ProgressionRow", content.BlockOutOfRange},
		{"row invalid feature", errOf(content.NewProgressionRow(1, 2, []content.RowFeature{content.RowClassFeature{}}, nil, nil)), "ProgressionRow", content.BlockInvalidMember},
		{"row two markers", errOf(content.NewProgressionRow(1, 2, []content.RowFeature{content.RowSubclassFeature{}, content.RowSubclassFeature{}}, nil, nil)), "ProgressionRow", content.BlockDuplicate},
		{"row invalid spellcasting", errOf(content.NewProgressionRow(1, 2, nil, content.PactSpellcastingRow{}, nil)), "ProgressionRow", content.BlockInvalidMember},
		{"row invalid column id", errOf(content.NewProgressionRow(1, 2, nil, nil, map[content.ClassColumnID]content.ColumnValue{"Bad": content.ColumnNone{}})), "ProgressionRow", content.BlockInvalidMember},
		{"row invalid value", errOf(content.NewProgressionRow(1, 2, nil, nil, map[content.ClassColumnID]content.ColumnValue{"a": content.ColumnDice{}})), "ProgressionRow", content.BlockInvalidMember},
		{"row nil value", errOf(content.NewProgressionRow(1, 2, nil, nil, map[content.ClassColumnID]content.ColumnValue{"a": nil})), "ProgressionRow", content.BlockInvalidMember},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
	if _, err := content.NewClassColumn("Bad", "A", content.ColumnKindCount); err == nil {
		t.Error("an invalid column id is accepted [unhappy (invalid input)]")
	}
	col := mustValue(content.NewClassColumn("rages", "Rages", content.ColumnKindDice))
	if col.ID() != "rages" || col.Header() != "Rages" || col.Kind() != content.ColumnKindDice {
		t.Errorf("column = %+v [happy]", col)
	}
	d := mustValue(content.NewColumnDice(2, 8))
	if d.Count() != 2 || d.Sides() != 8 || mustValue(content.NewColumnCount(0)).Value() != 0 || mustValue(content.NewColumnBonus(0)).Value() != 0 || mustValue(content.NewColumnFeetBonus(10)).Feet() != 10 {
		t.Error("column values lost their data [happy]")
	}
	if content.ColumnNoneText != "—" {
		t.Errorf("ColumnNoneText = %q, want U+2014 [happy]", content.ColumnNoneText)
	}
	f := mustValue(content.NewRowClassFeature("a_b", "one use"))
	if f.FeatureID() != "a_b" || f.Annotation() != "one use" {
		t.Errorf("row feature = %+v [happy]", f)
	}
	values := map[content.ClassColumnID]content.ColumnValue{"a": content.ColumnNone{}, "b": mustValue(content.NewColumnCount(3))}
	feats := []content.RowFeature{f, content.RowSubclassFeature{}}
	spell := mustValue(content.NewPactSpellcastingRow(1, 1, 1, 1))
	row := mustValue(content.NewProgressionRow(5, 3, feats, spell, values))
	values["c"] = content.ColumnNone{}
	feats[0] = content.RowSubclassFeature{}
	got := row.Features()
	got[0] = nil
	if _, err := row.Value("c"); err == nil {
		t.Error("a column added to the input map appeared in the row [unhappy (immutability)]")
	}
	if v, err := row.Value("b"); err != nil || v != mustValue(content.NewColumnCount(3)) {
		t.Errorf("Value(b) = (%v, %v) [happy]", v, err)
	}
	wantBlockError(t, errOf(row.Value("zz")), "ProgressionRow", content.BlockInvalidMember)
	if row.Level() != 5 || row.ProficiencyBonus() != 3 || row.Spellcasting() != spell || len(row.Features()) != 2 || row.Features()[0] != content.RowFeature(f) {
		t.Errorf("row = %+v [happy]", row)
	}
}

// Requirement: doc/specs/content-classes.md FR-052, FR-053, Cases ID - the feature
// id derivation for every row, and the typed ids of options and columns.
// Path: per case.
func TestDeriveFeatureID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, owner, feature string
		level                int
		want                 string
	}{
		{"plain", "barbarian", "Rage", 0, "barbarian_rage"},
		{"spaces", "bard", "Jack of All Trades", 0, "bard_jack_of_all_trades"},
		{"apostrophe deleted", "monk", "Monk's Focus", 0, "monk_monks_focus"},
		{"typographic apostrophe", "monk", "Monk’s Focus", 0, "monk_monks_focus"},
		{"plural possessive", "rogue", "Thieves' Cant", 0, "rogue_thieves_cant"},
		{"hyphen", "thief", "Second-Story Work", 0, "thief_second_story_work"},
		{"subclass owner", "fiend_patron", "Dark One's Own Luck", 0, "fiend_patron_dark_ones_own_luck"},
		{"duplicate name, first", "barbarian", "Improved Brutal Strike", 13, "barbarian_improved_brutal_strike_level_13"},
		{"duplicate name, second", "barbarian", "Improved Brutal Strike", 17, "barbarian_improved_brutal_strike_level_17"},
		{"edges and runs trimmed", "x", "  --A!!  B-- ", 0, "x_a_b"},
		{"character above the apostrophe", "x", "a…b", 0, "x_a_b"},
		{"range ends", "x", "az09", 0, "x_az09"},
		{"characters beside the ranges", "x", "a`b{c/d:e", 0, "x_a_b_c_d_e"},
		{"apostrophes only inside", "x", "a''b’'c", 0, "x_abc"},
		{"digits kept", "x", "Level 2 Thing", 0, "x_level_2_thing"},
		{"64 characters", "ab", strings.Repeat("c", 61), 0, "ab_" + strings.Repeat("c", 61)},
	}
	for _, c := range cases {
		got, err := content.DeriveFeatureID(c.owner, c.feature, c.level)
		if err != nil || got.String() != c.want || !got.Valid() {
			t.Errorf("%s: DeriveFeatureID = (%q, %v), want %q [%s FR-052, happy]", c.name, got, err, c.want, classSpec)
		}
	}
	long := []struct {
		name  string
		owner string
		name2 string
		level int
		block string
		want  content.BlockProblem
	}{
		{"65 characters", "ab", strings.Repeat("c", 62), 0, "", 0},
		{"no letters", "ab", "!!!", 0, "DeriveFeatureID", content.BlockInvalidName},
		{"empty name", "ab", "", 0, "DeriveFeatureID", content.BlockInvalidName},
		{"level 21", "ab", "x", 21, "DeriveFeatureID", content.BlockOutOfRange},
		{"level -1", "ab", "x", -1, "DeriveFeatureID", content.BlockOutOfRange},
		{"invalid owner", "Ab", "x", 0, "", 0},
	}
	for _, c := range long {
		_, err := content.DeriveFeatureID(c.owner, c.name2, c.level)
		if c.block == "" {
			var ie *content.IDError
			if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: err = %v, want *IDError [%s FR-052, unhappy (boundary)]", c.name, err, classSpec)
			}
			continue
		}
		wantBlockError(t, err, c.block, c.want)
	}
	if _, err := content.DeriveFeatureID("ab", "x", 20); err != nil {
		t.Errorf("level 20 rejected: %v [unhappy (boundary)]", err)
	}
	if id := content.MustClassColumnID("rages"); id.String() != "rages" || !id.Valid() || content.ClassColumnID("Bad").Valid() || content.ClassColumnID("").Valid() {
		t.Error("ClassColumnID validity [happy]")
	}
	if id := content.MustFeatureOptionID("agonizing_blast"); id.String() != "agonizing_blast" || !id.Valid() || content.FeatureOptionID("Bad").Valid() {
		t.Error("FeatureOptionID validity [happy]")
	}
	for _, bad := range []string{"", "A", "1a", "a b", strings.Repeat("a", 65)} {
		if _, err := content.ParseClassColumnID(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ParseClassColumnID(%q) = %v [unhappy (boundary)]", bad, err)
		}
		if _, err := content.ParseFeatureOptionID(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ParseFeatureOptionID(%q) = %v [unhappy (boundary)]", bad, err)
		}
	}
	assertMustPanics(t, func() { content.MustClassColumnID("Bad") })
	assertMustPanics(t, func() { content.MustFeatureOptionID("Bad") })
}

// Requirement: doc/specs/content-classes.md FR-040 - a column value fits only a
// column of its own kind, "—" fits every kind, and the constructor of the class
// agrees: every kind of value in every kind of column, and the cap of each count.
// Path: per case.
func TestClassColumnValueKinds(t *testing.T) {
	t.Parallel()
	base := decodeClassFixture(t, plainClassJSON())
	values := map[content.ClassColumnKind]content.ColumnValue{
		content.ColumnKindCount: mustValue(content.NewColumnCount(1)), content.ColumnKindDice: mustValue(content.NewColumnDice(1, 6)),
		content.ColumnKindBonus: mustValue(content.NewColumnBonus(1)), content.ColumnKindFeetBonus: mustValue(content.NewColumnFeetBonus(10)),
	}
	for columnKind := content.ColumnKindCount; columnKind.Valid(); columnKind++ {
		for valueKind, v := range values {
			for _, vv := range []content.ColumnValue{v, content.ColumnNone{}} {
				f := fields(base)
				f.Columns = []content.ClassColumn{mustValue(content.NewClassColumn("a", "A", columnKind))}
				f.Progression = slices.Clone(f.Progression)
				for i, r := range f.Progression {
					f.Progression[i] = mustValue(content.NewProgressionRow(r.Level(), r.ProficiencyBonus(), r.Features(), r.Spellcasting(), map[content.ClassColumnID]content.ColumnValue{"a": vv}))
				}
				_, none := vv.(content.ColumnNone)
				want := none || valueKind == columnKind
				if _, err := content.NewClassDefinition("fixture", "F", content.SourceCitation{}, f); (err == nil) != want {
					t.Errorf("%T in a %v column: err = %v, want accepted = %v [per case]", vv, columnKind, err, want)
				}
			}
		}
	}
	for _, ok := range []error{
		errOf(content.NewColumnDice(content.ClassMaxCount, 2)), errOf(content.NewColumnDice(1, 1000)),
		errOf(content.NewProgressionRow(1, content.ClassMaxCount, nil, nil, nil)),
	} {
		if ok != nil {
			t.Errorf("boundary value rejected: %v [unhappy (boundary)]", ok)
		}
	}
	wantBlockError(t, errOf(content.NewProgressionRow(1, over(), nil, nil, nil)), "ProgressionRow", content.BlockOutOfRange)
}

// Requirement: doc/specs/content-classes.md FR-031, FR-032 - the count bounds of
// the spellcasting rows: 0 and the cap are accepted, one beyond the cap is not,
// for every count.
// Path: unhappy (boundary).
func TestClassSpellcastingRowBounds(t *testing.T) {
	t.Parallel()
	const max = content.ClassMaxCount
	slots := func(n int) []int { return []int{0, 0, 0, 0, 0, 0, 0, 0, n} }
	for _, ok := range []error{
		errOf(content.NewSlotSpellcastingRow(0, slots(0))),
		errOf(content.NewSlotSpellcastingRowWithCantrips(max, max, slots(max))),
		errOf(content.NewPactSpellcastingRow(0, 0, 1, 1)),
		errOf(content.NewPactSpellcastingRow(max, max, max, 9)),
	} {
		if ok != nil {
			t.Errorf("boundary row rejected: %v [unhappy (boundary)]", ok)
		}
	}
	for _, c := range []blockCase{
		{"cantrips beyond the cap", errOf(content.NewSlotSpellcastingRowWithCantrips(over(), 0, slots(0))), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"prepared beyond the cap", errOf(content.NewSlotSpellcastingRow(over(), slots(0))), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"slots beyond the cap", errOf(content.NewSlotSpellcastingRow(0, slots(over()))), "SlotSpellcastingRow", content.BlockOutOfRange},
		{"pact cantrips beyond the cap", errOf(content.NewPactSpellcastingRow(over(), 0, 1, 1)), "PactSpellcastingRow", content.BlockOutOfRange},
		{"pact prepared beyond the cap", errOf(content.NewPactSpellcastingRow(0, over(), 1, 1)), "PactSpellcastingRow", content.BlockOutOfRange},
		{"pact slots beyond the cap", errOf(content.NewPactSpellcastingRow(0, 0, over(), 1)), "PactSpellcastingRow", content.BlockOutOfRange},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
}

// Requirement: doc/specs/content-classes.md FR-019, FR-021, FR-023, FR-024 - the
// count bounds of the training and equipment constructors accept 0 or 1 and the
// cap, and the distinctness rules hold whatever the order.
// Path: unhappy (boundary).
func TestClassTrainingBounds(t *testing.T) {
	t.Parallel()
	const max = content.ClassMaxCount
	simple := mustValue(content.NewWeaponProficiency(content.WeaponTrainingSimple, nil))
	martial := mustValue(content.NewWeaponProficiency(content.WeaponTrainingMartial, nil))
	for _, ok := range []error{
		errOf(content.NewSkillChoiceAny(max)), errOf(content.NewSkillChoiceAny(1)),
		errOf(content.NewClassToolProficiencyChoice([]string{"A"}, max)),
		errOf(content.NewClassEquipmentItem("A", max)), errOf(content.NewClassEquipmentItem("A", 1)),
		errOf(content.NewClassEquipmentOption(nil, 0)), errOf(content.NewClassEquipmentOption(nil, max)),
		errOf(content.NewMulticlassGrants([]content.WeaponProficiency{martial, simple}, nil, nil, nil)),
		errOf(content.NewMulticlassGrants([]content.WeaponProficiency{simple, martial}, nil, nil, nil)),
	} {
		if ok != nil {
			t.Errorf("boundary value rejected: %v [unhappy (boundary)]", ok)
		}
	}
	for _, c := range []blockCase{
		{"item beyond the cap", errOf(content.NewClassEquipmentItem("A", over())), "ClassEquipmentItem", content.BlockOutOfRange},
		{"gold beyond the cap", errOf(content.NewClassEquipmentOption(nil, over())), "ClassEquipmentOption", content.BlockOutOfRange},
		{"tool picks beyond the cap", errOf(content.NewClassToolProficiencyChoice([]string{"A"}, over())), "ClassToolProficiencyChoice", content.BlockOutOfRange},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
}

// Requirement: doc/specs/content-classes.md FR-025 - an item name is stored as
// given: a parenthetical stays in the name and a choice line is kept verbatim
// beside its quantity.
// Path: happy (stored as printed), and the decoder keeps the same name.
func TestClassEquipmentItemNamesStoredVerbatim(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Arcane Focus (Quarterstaff)", "Musical Instrument of your choice", "Dagger"} {
		it, err := content.NewClassEquipmentItem(name, 2)
		if err != nil || it.Name() != name || it.Quantity() != 2 {
			t.Errorf("NewClassEquipmentItem(%q) = %q x%d, %v; want the name unchanged [happy]", name, it.Name(), it.Quantity(), err)
		}
	}
	doc := classDoc(mutate(wizardClassJSON(), `{"name":"Staff","quantity":1}`, `{"name":"Arcane Focus (Quarterstaff)","quantity":1}`))
	if got := decodeProblems(doc); len(got) != 0 {
		t.Errorf("decoder rejected a parenthetical item name: %q [happy]", got)
	}
}
