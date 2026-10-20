package content_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func featureIDs(es []content.FeatureEntry) []string {
	out := []string{}
	for _, e := range es {
		out = append(out, e.ID().String())
	}
	return out
}

func wantBounds(t *testing.T, what string, f func(level int) error) {
	t.Helper()
	for _, level := range []int{-1, 0, 21, 100} {
		wantBlockError(t, f(level), "ClassDefinition", content.BlockOutOfRange)
	}
	for _, level := range []int{1, 20} {
		if err := f(level); err != nil {
			t.Errorf("%s(%d) = %v, want no error [%s FR-047, unhappy (boundary)]", what, level, err, classSpec)
		}
	}
}

// Requirement: doc/specs/content-classes.md FR-001, FR-047 to FR-050, FR-099,
// FR-074, FR-075 - the accessors of a full caster with a subclass: key, row and
// level bounds (0, 1, 20, 21), entries by level, resolved rows, feat grants,
// column and feature lookup, subclass lookup.
// Path: per case.
func TestClassDefinitionAccessors(t *testing.T) {
	t.Parallel()
	c := decodeClassFixture(t, ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6)))
	if c.Key() != content.NewKey(content.KindCharacterClass, content.MustID("fixture")) || c.ID() != "fixture" || c.Name() != "Fixture" || !c.Source().IsZero() {
		t.Errorf("identity = %v %v %v [happy]", c.Key(), c.ID(), c.Name())
	}
	if c.HitPointDie() != content.HitDieD6 || !slices.Equal(c.SavingThrowProficiencies(), []ability.Ability{ability.Intelligence, ability.Wisdom}) || c.SubclassLevel() != 3 {
		t.Errorf("core = %v %v %d [happy]", c.HitPointDie(), c.SavingThrowProficiencies(), c.SubclassLevel())
	}
	if all, ok := c.PrimaryAbility().(content.AllOf); !ok || !slices.Equal(all.Abilities(), []ability.Ability{ability.Intelligence}) {
		t.Errorf("PrimaryAbility = %v [happy]", c.PrimaryAbility())
	}
	if sc, ok := c.Spellcasting().(content.SlotSpellcastingTrait); !ok || !sc.Cantrips() || len(c.Progression()) != 20 || len(c.Features()) != 10 || len(c.Subclasses()) != 1 || len(c.Columns()) != 0 {
		t.Errorf("shape = %v %d %d %d [happy]", c.Spellcasting(), len(c.Progression()), len(c.Features()), len(c.Subclasses()))
	}
	if len(c.WeaponProficiencies()) != 1 || len(c.ToolProficiencies()) != 0 || len(c.ArmorTraining()) != 0 || c.SkillChoice().Picks() != 2 {
		t.Error("training accessors [happy]")
	}
	if _, hasC := c.StartingEquipment().C(); hasC || c.StartingEquipment().A().Gold() != 5 || c.MulticlassGrants().SkillChoice() != nil {
		t.Error("equipment and grants accessors [happy]")
	}
	// Rows: 0, 1, 20 and 21 (FR-047).
	wantBounds(t, "RowAt", func(l int) error { return errOf(c.RowAt(l)) })
	wantBounds(t, "EntriesGainedAt", func(l int) error { return errOf(c.EntriesGainedAt(l)) })
	wantBounds(t, "RowFeatures", func(l int) error { return errOf(c.RowFeatures(l)) })
	wantBounds(t, "FeatGrantsAt", func(l int) error { return errOf(c.FeatGrantsAt(l)) })
	sub := mustSubclass(t, c)
	wantBounds(t, "FeaturesAtOrBelow", func(l int) error { return errOf(sub.FeaturesAtOrBelow(l)) })
	if r, _ := c.RowAt(1); r.Level() != 1 {
		t.Errorf("RowAt(1).Level = %d [happy]", r.Level())
	}
	if r, _ := c.RowAt(20); r.Level() != 20 || r.ProficiencyBonus() != 6 {
		t.Errorf("RowAt(20) = %+v [happy]", r)
	}
	// Entries by level (FR-048).
	for _, tc := range []struct {
		level int
		want  []string
	}{{1, []string{"fixture_alpha", "fixture_beta"}}, {2, []string{"fixture_gamma"}}, {6, []string{}}, {20, []string{"fixture_theta"}}} {
		got, _ := c.EntriesGainedAt(tc.level)
		if !slices.Equal(featureIDs(got), tc.want) {
			t.Errorf("EntriesGainedAt(%d) = %v, want %v [per case]", tc.level, featureIDs(got), tc.want)
		}
	}
	// Resolved rows: the level 4 entry again at 8, the marker at 6 (FR-048).
	if r, _ := c.RowFeatures(8); !slices.Equal(featureIDs(r.Entries()), []string{"fixture_asi"}) || r.HasSubclassFeature() {
		t.Errorf("RowFeatures(8) = %v [per case]", featureIDs(r.Entries()))
	}
	r6, _ := c.RowFeatures(6)
	if got := r6.Entries(); got == nil || len(got) != 0 || !r6.HasSubclassFeature() {
		t.Errorf("RowFeatures(6) = %v marker %v [per case]", got, r6.HasSubclassFeature())
	}
	if (content.ResolvedRow{}).HasSubclassFeature() || len((content.ResolvedRow{}).Entries()) != 0 {
		t.Error("the zero ResolvedRow is not empty [unhappy (empty case)]")
	}
	// Feat grants (FR-049): General at 4, 8, 12, 16; none at 5; Epic Boon at 19.
	for _, tc := range []struct {
		level int
		want  []content.FeatCategory
	}{{4, []content.FeatCategory{content.FeatGeneral}}, {8, []content.FeatCategory{content.FeatGeneral}}, {5, []content.FeatCategory{}}, {19, []content.FeatCategory{content.FeatEpicBoon}}} {
		got, _ := c.FeatGrantsAt(tc.level)
		if !slices.Equal(got, tc.want) || got == nil {
			t.Errorf("FeatGrantsAt(%d) = %v, want %v [per case]", tc.level, got, tc.want)
		}
	}
	// Lookups (FR-099, FR-074).
	if f, ok := c.Feature("fixture_asi").(content.FeatureEntryFound); !ok || f.Entry().Level() != 4 {
		t.Errorf("Feature(asi) = %v [happy]", c.Feature("fixture_asi"))
	}
	for _, id := range []content.FeatureID{"fixture_nothing", "sub_a_f0", ""} {
		if nf, ok := c.Feature(id).(content.FeatureEntryNotFound); !ok || nf.ID() != id {
			t.Errorf("Feature(%q) = %v, want not found carrying the id [unhappy (missing content)]", id, c.Feature(id))
		}
	}
	if s, ok := c.Subclass("sub_a").(content.SubclassFound); !ok || s.Subclass().ID() != "sub_a" {
		t.Errorf("Subclass(sub_a) = %v [happy]", c.Subclass("sub_a"))
	}
	if nf, ok := c.Subclass("sub_z").(content.SubclassNotFound); !ok || nf.ID() != "sub_z" {
		t.Errorf("Subclass(sub_z) = %v [unhappy (missing content)]", c.Subclass("sub_z"))
	}
	// Subclass features at or below a level (FR-075), table SF shape.
	for _, tc := range []struct {
		level int
		want  int
	}{{1, 0}, {2, 0}, {3, 1}, {5, 1}, {6, 2}, {20, 2}} {
		got, _ := sub.FeaturesAtOrBelow(tc.level)
		if len(got) != tc.want || got == nil {
			t.Errorf("FeaturesAtOrBelow(%d) has %d features, want %d [per case]", tc.level, len(got), tc.want)
		}
	}
	if sub.Name() != "Sub sub_a" || !sub.Source().IsZero() || len(sub.Features()) != 2 {
		t.Errorf("subclass = %+v [happy]", sub)
	}
	// Columns (FR-050).
	cc := decodeClassFixture(t, columnClassJSON())
	if col, err := cc.Column("die"); err != nil || col.Kind() != content.ColumnKindDice {
		t.Errorf("Column(die) = (%v, %v) [happy]", col, err)
	}
	wantBlockError(t, errOf(cc.Column("zzz")), "ClassDefinition", content.BlockInvalidMember)
	wantBlockError(t, errOf(c.Column("die")), "ClassDefinition", content.BlockInvalidMember)
	row, _ := cc.RowAt(1)
	if v, err := row.Value("charges"); err != nil || v != content.ColumnValue(mustValue(content.NewColumnCount(2))) {
		t.Errorf("Value(charges) = (%v, %v) [happy]", v, err)
	}
	if len(cc.Columns()) != 4 || cc.Spellcasting() != nil {
		t.Errorf("columns = %d, spellcasting %v [happy]", len(cc.Columns()), cc.Spellcasting())
	}
}

func mustSubclass(t *testing.T, c content.ClassDefinition) content.SubclassDefinition {
	t.Helper()
	s, ok := c.Subclass("sub_a").(content.SubclassFound)
	if !ok {
		t.Fatal("no subclass sub_a")
	}
	return s.Subclass()
}

// Requirement: doc/specs/content-classes.md FR-008 - every collection is a copy
// on read, and two classes built from the same text are equal.
// Path: unhappy (immutability).
func TestClassDefinitionImmutability(t *testing.T) {
	t.Parallel()
	doc := ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6))
	c := decodeClassFixture(t, doc)
	c.SavingThrowProficiencies()[0] = ability.Strength
	c.WeaponProficiencies()[0] = content.WeaponProficiency{}
	c.Progression()[0] = content.ProgressionRow{}
	c.Features()[0] = content.FeatureEntry{}
	c.Subclasses()[0] = content.SubclassDefinition{}
	c.Columns()
	mustSubclass(t, c).Features()[0] = content.FeatureEntry{}
	again := decodeClassFixture(t, doc)
	if !slices.Equal(c.SavingThrowProficiencies(), again.SavingThrowProficiencies()) || c.Progression()[0].Level() != 1 || c.Features()[0].ID() != "fixture_alpha" ||
		c.Subclasses()[0].ID() != "sub_a" || mustSubclass(t, c).Features()[0].Level() != 3 || c.WeaponProficiencies()[0].Category() != content.WeaponTrainingSimple {
		t.Error("mutating a returned slice changed the class")
	}
	a, b := decodeSet(t, classDoc(doc)), decodeSet(t, classDoc(doc))
	if !a.Equal(b) {
		t.Error("two classes decoded from the same text are not equal [happy]")
	}
	if a.Equal(decodeSet(t, classDoc(ownerClassJSON("fixture", subclassJSON("sub_b", 3, 6))))) {
		t.Error("different classes are equal [unhappy (boundary)]")
	}
}

// fields rebuilds the fields of c, for a constructor test to edit.
func fields(c content.ClassDefinition) content.ClassFields {
	return content.ClassFields{
		PrimaryAbility: c.PrimaryAbility(), HitPointDie: c.HitPointDie(), SavingThrowProficiencies: c.SavingThrowProficiencies(),
		SkillChoice: c.SkillChoice(), WeaponProficiencies: c.WeaponProficiencies(), ToolProficiencies: c.ToolProficiencies(),
		ArmorTraining: c.ArmorTraining(), StartingEquipment: c.StartingEquipment(), MulticlassGrants: c.MulticlassGrants(),
		Spellcasting: c.Spellcasting(), Columns: c.Columns(), Progression: c.Progression(), Features: c.Features(), Subclasses: c.Subclasses(),
	}
}

// Requirement: doc/specs/content-classes.md FR-007, FR-073 - the constructor
// rejects what the decoder rejects, with an error wrapping ErrInvalidInput, in
// the documented order; MustClassDefinition panics on the same input.
// Path: per case.
func TestNewClassDefinition(t *testing.T) {
	t.Parallel()
	base := decodeClassFixture(t, ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6)))
	build := func(id content.ClassID, name string, edit func(*content.ClassFields)) error {
		f := fields(base)
		edit(&f)
		return errOf(content.NewClassDefinition(id, name, content.SourceCitation{}, f))
	}
	nop := func(*content.ClassFields) {}
	feat := func(slug string, level int, facts ...content.FeatureFact) content.FeatureEntry {
		return mustValue(content.NewFeatureEntry(content.FeatureID(slug), "N "+slug, content.FeatureEntryFields{Level: level, Text: "t", Facts: facts}))
	}
	without := func(i int) func(*content.ClassFields) {
		return func(f *content.ClassFields) { f.Features = slices.Delete(slices.Clone(f.Features), i, i+1) }
	}
	row0 := base.Progression()[0]
	cases := []blockCase{
		{"valid", build("fixture", "Fixture", nop), "", 0},
		{"blank name", build("fixture", "", nop), "ClassDefinition", content.BlockInvalidName},
		{"no primary ability", build("fixture", "F", func(f *content.ClassFields) { f.PrimaryAbility = nil }), "ClassDefinition", content.BlockMissing},
		{"no skill choice", build("fixture", "F", func(f *content.ClassFields) { f.SkillChoice = nil }), "ClassDefinition", content.BlockMissing},
		{"invalid primary ability", build("fixture", "F", func(f *content.ClassFields) { f.PrimaryAbility = content.AllOf{} }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid die", build("fixture", "F", func(f *content.ClassFields) { f.HitPointDie = 0 }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid skill choice", build("fixture", "F", func(f *content.ClassFields) { f.SkillChoice = content.SkillChoiceAny{} }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid tool", build("fixture", "F", func(f *content.ClassFields) { f.ToolProficiencies = []content.ClassToolProficiency{nil} }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid spellcasting", build("fixture", "F", func(f *content.ClassFields) { f.Spellcasting = content.SlotSpellcastingTrait{} }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid column", build("fixture", "F", func(f *content.ClassFields) { f.Columns = []content.ClassColumn{{}} }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid row", build("fixture", "F", func(f *content.ClassFields) {
			f.Progression = slices.Clone(f.Progression)
			f.Progression[3] = content.ProgressionRow{}
		}), "ClassDefinition", content.BlockInvalidMember},
		{"invalid feature", build("fixture", "F", func(f *content.ClassFields) { f.Features = []content.FeatureEntry{{}} }), "ClassDefinition", content.BlockInvalidMember},
		{"invalid subclass", build("fixture", "F", func(f *content.ClassFields) { f.Subclasses = []content.SubclassDefinition{{}} }), "ClassDefinition", content.BlockInvalidMember},
		{"no saves", build("fixture", "F", func(f *content.ClassFields) { f.SavingThrowProficiencies = nil }), "ClassDefinition", content.BlockOutOfRange},
		{"repeated save", build("fixture", "F", func(f *content.ClassFields) { f.SavingThrowProficiencies = abilities(ability.Wisdom, ability.Wisdom) }), "ClassDefinition", content.BlockDuplicate},
		{"repeated weapon category", build("fixture", "F", func(f *content.ClassFields) {
			f.WeaponProficiencies = append(f.WeaponProficiencies, f.WeaponProficiencies[0])
		}), "ClassDefinition", content.BlockDuplicate},
		{"repeated armor", build("fixture", "F", func(f *content.ClassFields) {
			f.ArmorTraining = []content.ArmorTrainingCategory{content.ArmorTrainingLight, content.ArmorTrainingLight}
		}), "ClassDefinition", content.BlockDuplicate},
		{"repeated column", build("fixture", "F", func(f *content.ClassFields) {
			col := mustValue(content.NewClassColumn("a", "A", content.ColumnKindCount))
			f.Columns = []content.ClassColumn{col, col}
		}), "ClassDefinition", content.BlockDuplicate},
		{"21 rows", build("fixture", "F", func(f *content.ClassFields) { f.Progression = append(slices.Clone(f.Progression), f.Progression[19]) }), "ClassDefinition", content.BlockOutOfRange},
		{"19 rows", build("fixture", "F", func(f *content.ClassFields) { f.Progression = f.Progression[:19] }), "ClassDefinition", content.BlockOutOfRange},
		{"other class id", build("other", "F", nop), "ClassDefinition", content.BlockConstraint},
		{"rows out of order", build("fixture", "F", func(f *content.ClassFields) {
			f.Progression = slices.Clone(f.Progression)
			f.Progression[0], f.Progression[1] = f.Progression[1], f.Progression[0]
		}), "ClassDefinition", content.BlockInvalidMember},
		{"row below its position", build("fixture", "F", func(f *content.ClassFields) {
			f.Progression = slices.Clone(f.Progression)
			f.Progression[2] = f.Progression[0]
		}), "ClassDefinition", content.BlockInvalidMember},
		{"values for an undeclared column", build("fixture", "F", func(f *content.ClassFields) {
			f.Progression = slices.Clone(f.Progression)
			f.Progression[0] = mustValue(content.NewProgressionRow(1, 2, row0.Features(), row0.Spellcasting(), map[content.ClassColumnID]content.ColumnValue{"zz": content.ColumnNone{}}))
		}), "ClassDefinition", content.BlockConstraint},
		{"no value for a declared column", build("fixture", "F", func(f *content.ClassFields) {
			f.Columns = []content.ClassColumn{mustValue(content.NewClassColumn("a", "A", content.ColumnKindCount))}
		}), "ClassDefinition", content.BlockMissing},
		{"value of the wrong kind", build("fixture", "F", func(f *content.ClassFields) {
			f.Columns = []content.ClassColumn{mustValue(content.NewClassColumn("a", "A", content.ColumnKindDice))}
			f.Progression = slices.Clone(f.Progression)
			for i, r := range f.Progression {
				f.Progression[i] = mustValue(content.NewProgressionRow(r.Level(), r.ProficiencyBonus(), r.Features(), r.Spellcasting(), map[content.ClassColumnID]content.ColumnValue{"a": mustValue(content.NewColumnCount(1))}))
			}
		}), "ClassDefinition", content.BlockConstraint},
		{"pact row on a slot class", build("fixture", "F", func(f *content.ClassFields) {
			f.Spellcasting = mustValue(content.NewPactSpellcastingTrait(ability.Charisma))
		}), "ClassDefinition", content.BlockConstraint},
		{"slot trait without cantrips flag", build("fixture", "F", func(f *content.ClassFields) {
			f.Spellcasting = mustValue(content.NewSlotSpellcastingTrait(ability.Wisdom, false))
		}), "ClassDefinition", content.BlockConstraint},
		{"non-caster with rows", build("fixture", "F", func(f *content.ClassFields) { f.Spellcasting = nil }), "ClassDefinition", content.BlockConstraint},
		{"feature missing from the rows", build("fixture", "F", without(0)), "ClassDefinition", content.BlockConstraint},
		{"no subclass fact", build("fixture", "F", func(f *content.ClassFields) {
			f.Features = slices.Clone(f.Features)
			f.Features[3] = feat("fixture_delta", 3)
			f.Subclasses = nil
		}), "ClassDefinition", content.BlockConstraint},
		{"subclass fact twice", build("fixture", "F", func(f *content.ClassFields) {
			f.Features = slices.Clone(f.Features)
			f.Features[5] = feat("fixture_epsilon", 5, content.GrantsSubclass{})
		}), "ClassDefinition", content.BlockConstraint},
		{"subclass below the subclass level", build("fixture", "F", func(f *content.ClassFields) {
			f.Subclasses = []content.SubclassDefinition{mustValue(content.NewSubclassDefinition("sub_b", "B", content.SourceCitation{}, []content.FeatureEntry{feat("sub_b_f", 2), feat("sub_b_g", 3)}))}
		}), "ClassDefinition", content.BlockConstraint},
		{"subclass with no feature at the level", build("fixture", "F", func(f *content.ClassFields) {
			f.Subclasses = []content.SubclassDefinition{mustValue(content.NewSubclassDefinition("sub_b", "B", content.SourceCitation{}, []content.FeatureEntry{feat("sub_b_f", 6)}))}
		}), "ClassDefinition", content.BlockConstraint},
		{"subclass feature with another prefix", build("fixture", "F", func(f *content.ClassFields) {
			f.Subclasses = []content.SubclassDefinition{mustValue(content.NewSubclassDefinition("sub_b", "B", content.SourceCitation{}, []content.FeatureEntry{feat("other_f", 3)}))}
		}), "ClassDefinition", content.BlockConstraint},
		{"subclass feature with the subclass fact", build("fixture", "F", func(f *content.ClassFields) {
			f.Subclasses = []content.SubclassDefinition{mustValue(content.NewSubclassDefinition("sub_b", "B", content.SourceCitation{}, []content.FeatureEntry{feat("sub_b_f", 3, content.GrantsSubclass{})}))}
		}), "ClassDefinition", content.BlockConstraint},
		{"multiclass armor beyond the core", build("fixture", "F", func(f *content.ClassFields) {
			f.MulticlassGrants = mustValue(content.NewMulticlassGrants(nil, []content.ArmorTrainingCategory{content.ArmorTrainingHeavy}, nil, nil))
		}), "ClassDefinition", content.BlockConstraint},
	}
	for _, c := range cases {
		if c.block == "" {
			if c.err != nil {
				t.Errorf("%s: err = %v [happy]", c.name, c.err)
			}
			continue
		}
		wantBlockError(t, c.err, c.block, c.want)
	}
	var ie *content.IDError
	if err := build("Bad", "F", nop); !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an invalid id = %v, want *IDError [unhappy (invalid input)]", err)
	}
	f := fields(base)
	f.PrimaryAbility = nil
	assertMustPanics(t, func() { content.MustClassDefinition("fixture", "F", content.SourceCitation{}, f) })
	if c := content.MustClassDefinition("fixture", "F", content.SourceCitation{}, fields(base)); c.SubclassLevel() != 3 {
		t.Error("MustClassDefinition lost the subclass level [happy]")
	}
	// The constructor copies its input slices (FR-008).
	g := fields(base)
	built := content.MustClassDefinition("fixture", "F", content.SourceCitation{}, g)
	g.Features[0] = content.FeatureEntry{}
	g.Progression[0] = content.ProgressionRow{}
	g.SavingThrowProficiencies[0] = ability.Strength
	if built.Features()[0].ID() != "fixture_alpha" || built.Progression()[0].Level() != 1 || built.SavingThrowProficiencies()[0] != ability.Intelligence {
		t.Error("mutating the input slices changed the class [unhappy (immutability)]")
	}
}

// Requirement: doc/specs/content-classes.md FR-059 - the decoder alone accepts a
// spell id that names no spell; ValidateSet reports unresolvedReference naming
// the owner, the field and the id for a class feature, a subclass feature and an
// option, and the set resolves once the spells are present.
// Path: unhappy (missing content).
func TestClassSpellReferences(t *testing.T) {
	t.Parallel()
	w := ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6))
	w = mutate(w, alphaTail, withFacts(alphaTail, spellFact(`{"level":1,"spellIds":["spell_a","spell_b"]}`)))
	w = mutate(w, betaTail, withOptions(betaTail, option("o", `,"facts":[`+spellFact(`{"level":1,"spellIds":["spell_c"]}`)+`]`)))
	w = mutate(w, `"id":"sub_a_f1","name":"Name sub_a_f1","level":6,"text":"Text.\nMore."}`, `"id":"sub_a_f1","name":"Name sub_a_f1","level":6,"text":"Text.\nMore.","facts":[`+spellFact(`{"level":6,"spellIds":["spell_d"]}`)+`],"options":[`+option("p", `,"facts":[`+spellFact(`{"level":6,"spellIds":["spell_e"]}`)+`]`)+`]}`)
	set := decodeSet(t, classDoc(w))
	c := decodeClassFixture(t, w)
	refs := c.References()
	var fieldsSeen []string
	for _, r := range refs {
		if r.Target().Kind() != content.KindSpell || r.Constraint() != nil {
			t.Errorf("reference %v: want a spell with no constraint [happy]", r)
		}
		fieldsSeen = append(fieldsSeen, r.Field())
	}
	want := []string{
		"features/0/facts/0/grants/0/spellIds/0", "features/0/facts/0/grants/0/spellIds/1",
		"features/1/options/0/facts/0/grants/0/spellIds/0",
		"subclasses/0/features/1/facts/0/grants/0/spellIds/0",
		"subclasses/0/features/1/options/0/facts/0/grants/0/spellIds/0",
	}
	if !slices.Equal(fieldsSeen, want) {
		t.Errorf("reference fields = %v, want %v [happy]", fieldsSeen, want)
	}
	_, err := content.ValidateSet(set)
	probs := content.ReferenceProblems(err)
	if len(probs) != 5 {
		t.Fatalf("ValidateSet problems = %v, want one per spell id [unhappy (missing content)]", probs)
	}
	for i, p := range probs {
		if p.Kind != content.ProblemUnresolvedReference || p.Owner != c.Key() || p.Field != want[i] || p.Target.Kind() != content.KindSpell {
			t.Errorf("problem %d = %+v, want unresolvedReference at %s [unhappy (missing content)]", i, p, want[i])
		}
	}
	all := stubSet(t, content.KindSpell, "spell_a", "spell_b", "spell_c", "spell_d", "spell_e")
	if _, err := content.ValidateSet(set, all); err != nil {
		t.Errorf("ValidateSet with the spells = %v, want nil [happy]", err)
	}
	if got := decodeClassFixture(t, wizardClassJSON()).References(); got != nil {
		t.Errorf("a class with no spell facts has references %v [unhappy (empty case)]", got)
	}
}

// Requirement: doc/specs/content-classes.md section 8 (telemetry tests) and
// doc/specs/content-schema-and-decoder.md FR-079 to FR-086 - LoadSet over a
// rejected class document ends the span in error and adds one decode_errors
// measurement per problem kind; an accepted one counts one character_class
// entry.
// Path: unhappy (invalid input).
func TestLoadSetClassDocument(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	loader := mustLoader(t, content.JSONDecoder{}, content.WithTelemetry(rec.Telemetry()))
	bad := classDoc(mutate(mutate(mutate(wizardClassJSON(), `"D6"`, `"D4"`), `"armorTraining":[]`, `"armorTraining":["Light","Light"]`), `"level":1,"text"`, `"level":0,"text"`))
	res, err := loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(bad)})
	rejected, ok := res.(content.LoadSetRejected)
	if !ok || err != nil || len(rejected.Problems()) != 3 {
		t.Fatalf("LoadSet(bad) = (%T, %v), want LoadSetRejected with 3 problems", res, err)
	}
	counts := map[any]float64{}
	for _, m := range rec.Meter.MeasurementsNamed(content.MetricDecodeErrors) {
		counts[attrsOf(m.Attrs)[content.AttrProblemKind]] += m.Value
	}
	if len(counts) != 3 || counts["invalid_value"] != 1 || counts["duplicate"] != 1 || counts["out_of_range"] != 1 {
		t.Errorf("decode_errors = %v, want one measurement per problem kind [unhappy (invalid input)]", counts)
	}
	rec2 := telemetrytest.NewRecorder()
	loader2 := mustLoader(t, content.JSONDecoder{}, content.WithTelemetry(rec2.Telemetry()))
	if res, err := loader2.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(classDoc(wizardClassJSON()))}); err != nil || res == nil {
		t.Fatalf("LoadSet(good) = (%v, %v)", res, err)
	}
	got := rec2.Meter.MeasurementsNamed(content.MetricEntriesLoaded)
	if len(got) != 1 || got[0].Value != 1 || attrsOf(got[0].Attrs)[content.AttrKind] != "character_class" {
		t.Errorf("entries_loaded = %+v, want one character_class [happy]", got)
	}
}

// Requirement: doc/specs/content-classes.md FR-070, FR-074 - the subclass
// constructor and lookup: each rejection by block and problem, copy on read, and
// the lookup of any of several subclasses and columns.
// Path: per case.
func TestSubclassDefinitionAndLookups(t *testing.T) {
	t.Parallel()
	feat := func(slug string, level int) content.FeatureEntry {
		return mustValue(content.NewFeatureEntry(content.FeatureID(slug), "N "+slug, content.FeatureEntryFields{Level: level, Text: "t"}))
	}
	src := mustValue(content.NewSourceCitation("D", "S", 4))
	var ie *content.IDError
	if err := errOf(content.NewSubclassDefinition("Bad", "B", content.SourceCitation{}, []content.FeatureEntry{feat("a_b", 3)})); !errors.As(err, &ie) {
		t.Errorf("an invalid id = %v, want *IDError [unhappy (invalid input)]", err)
	}
	wantBlockError(t, errOf(content.NewSubclassDefinition("sub_a", "", content.SourceCitation{}, []content.FeatureEntry{feat("a_b", 3)})), "SubclassDefinition", content.BlockInvalidName)
	wantBlockError(t, errOf(content.NewSubclassDefinition("sub_a", "B", content.SourceCitation{}, nil)), "SubclassDefinition", content.BlockEmpty)
	wantBlockError(t, errOf(content.NewSubclassDefinition("sub_a", "B", content.SourceCitation{}, []content.FeatureEntry{{}})), "SubclassDefinition", content.BlockInvalidMember)
	in := []content.FeatureEntry{feat("sub_a_x", 3)}
	s := mustValue(content.NewSubclassDefinition("sub_a", "B", src, in))
	in[0] = content.FeatureEntry{}
	if s.ID() != "sub_a" || s.Name() != "B" || s.Source() != src || s.Features()[0].ID() != "sub_a_x" {
		t.Errorf("subclass = %+v [unhappy (immutability)]", s)
	}
	// Lookups find each of several subclasses and columns, whatever their order.
	c := decodeClassFixture(t, ownerClassJSON("fixture", subclassJSON("sub_b", 3)+","+subclassJSON("sub_a", 3)))
	for _, id := range []content.SubclassID{"sub_a", "sub_b"} {
		if f, ok := c.Subclass(id).(content.SubclassFound); !ok || f.Subclass().ID() != id {
			t.Errorf("Subclass(%s) = %v [happy]", id, c.Subclass(id))
		}
	}
	cc := decodeClassFixture(t, columnClassJSON())
	for _, id := range []content.ClassColumnID{"charges", "die", "bonus", "speed"} {
		if col, err := cc.Column(id); err != nil || col.ID() != id {
			t.Errorf("Column(%s) = (%v, %v) [happy]", id, col, err)
		}
	}
	for _, f := range c.Features() {
		if got, ok := c.Feature(f.ID()).(content.FeatureEntryFound); !ok || got.Entry().ID() != f.ID() {
			t.Errorf("Feature(%s) = %v [happy]", f.ID(), got)
		}
	}
}
