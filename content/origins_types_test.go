package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func mustLevelBenefits(t *testing.T, levels ...int) []content.LevelBenefit {
	t.Helper()
	var out []content.LevelBenefit
	for _, l := range levels {
		b, err := content.NewLevelBenefit(l, "text", "spell_a")
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-061, FR-069 - a LevelBenefit has a level of
// 1, 3 or 5, verbatim text and a spell id; each break is an error wrapping ErrInvalidInput.
// Path: per case.
func TestNewLevelBenefit(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		level      int
		spell      content.SpellID
		wantBlock  content.BlockProblem // 0: accepted; ignored when wantID is set
		wantID     content.IDProblem
	}{
		{name: "level 1", level: 1, text: "t", spell: "spell_a"},
		{name: "level 3", level: 3, text: "t", spell: "spell_a"},
		{name: "level 5", level: 5, text: "t", spell: "spell_a"},
		{name: "level 0", level: 0, text: "t", spell: "spell_a", wantBlock: content.BlockInvalidMember},
		{name: "level 2", level: 2, text: "t", spell: "spell_a", wantBlock: content.BlockInvalidMember},
		{name: "level 6", level: 6, text: "t", spell: "spell_a", wantBlock: content.BlockInvalidMember},
		{name: "blank text", level: 1, text: " ", spell: "spell_a", wantBlock: content.BlockBlankText},
		{name: "control character in text", level: 1, text: "a\tb", spell: "spell_a", wantBlock: content.BlockInvalidCharacter},
		{name: "empty spell id", level: 1, text: "t", spell: "", wantID: content.IDEmpty},
		{name: "malformed spell id", level: 1, text: "t", spell: "Spell A", wantID: content.IDInvalidFormat},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b, err := content.NewLevelBenefit(c.level, c.text, c.spell)
			switch {
			case c.wantID != 0:
				var ie *content.IDError
				if !errors.As(err, &ie) || ie.Problem != c.wantID || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("error = %v, want *IDError %v [FR-061, unhappy (invalid input)]", err, c.wantID)
				}
			case c.wantBlock != 0:
				wantBlockError(t, err, "LevelBenefit", c.wantBlock)
			case err != nil || b.Level() != c.level || b.Text() != c.text || b.SpellID() != c.spell:
				t.Errorf("NewLevelBenefit = (%+v, %v), want the arguments back [FR-061, happy]", b, err)
			}
		})
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-061, FR-069 - a LineageOption has exactly three
// benefits for levels 1, 3, 5 in order; the result is a copy.
// Path: per case.
func TestNewLineageOption(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		optName string
		levels  []int
		want    content.BlockProblem // 0: accepted
	}{
		{"levels 1, 3, 5", "L", []int{1, 3, 5}, 0},
		{"no levels", "L", nil, content.BlockOutOfRange},
		{"two levels", "L", []int{1, 3}, content.BlockOutOfRange},
		{"four levels", "L", []int{1, 3, 5, 5}, content.BlockOutOfRange},
		{"repeated level", "L", []int{1, 3, 3}, content.BlockDuplicate},
		{"repeated first level", "L", []int{1, 1, 5}, content.BlockDuplicate},
		{"order 1, 5, 3", "L", []int{1, 5, 3}, content.BlockConstraint},
		{"order 5, 3, 1", "L", []int{5, 3, 1}, content.BlockConstraint},
		{"blank name", " ", []int{1, 3, 5}, content.BlockInvalidName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var bs []content.LevelBenefit
			for _, l := range c.levels {
				bs = append(bs, mustLevelBenefits(t, l)...)
			}
			o, err := content.NewLineageOption(c.optName, bs)
			if c.want != 0 {
				wantBlockError(t, err, "LineageOption", c.want)
				return
			}
			if err != nil {
				t.Fatalf("NewLineageOption = %v, want accepted [FR-061, happy]", err)
			}
			got := o.LevelBenefits()
			got[0] = content.LevelBenefit{}
			bs[1] = content.LevelBenefit{}
			if again := o.LevelBenefits(); again[0].Level() != 1 || again[1].Level() != 3 || o.Name() != "L" {
				t.Errorf("option changed through a returned or an input slice: %+v [immutability]", again)
			}
		})
	}
	if _, err := content.NewLineageOption("L", []content.LevelBenefit{{}, {}, {}}); !errors.Is(err, &content.BlockError{Block: "LineageOption", Problem: content.BlockInvalidMember}) {
		t.Errorf("zero benefits error = %v, want BlockInvalidMember [FR-061, unhappy (invalid input)]", err)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-016, FR-017, FR-020 - the DamageTypeOption and
// TextOption constructors validate and copy.
// Path: per case.
func TestNewDamageAndTextOptions(t *testing.T) {
	t.Parallel()
	d, err := content.NewDamageTypeOption("Name", content.DamageAcid)
	if err != nil || d.Name() != "Name" || d.DamageType() != content.DamageAcid {
		t.Errorf("NewDamageTypeOption = (%+v, %v) [happy]", d, err)
	}
	_, err = content.NewDamageTypeOption(" ", content.DamageAcid)
	wantBlockError(t, err, "DamageTypeOption", content.BlockInvalidName)
	_, err = content.NewDamageTypeOption("Name", 0)
	wantBlockError(t, err, "DamageTypeOption", content.BlockInvalidMember)

	ids := []content.SpellID{"spell_a", "spell_b"}
	o, err := content.NewTextOption("Name", "Text.\nMore.", ids)
	if err != nil || o.Name() != "Name" || o.Text() != "Text.\nMore." || !slices.Equal(o.SpellIDs(), ids) {
		t.Errorf("NewTextOption = (%+v, %v) [happy]", o, err)
	}
	ids[0] = "changed"
	got := o.SpellIDs()
	got[1] = "changed"
	if !slices.Equal(o.SpellIDs(), []content.SpellID{"spell_a", "spell_b"}) {
		t.Errorf("TextOption changed through an input or returned slice [immutability]")
	}
	if none, err := content.NewTextOption("N", "t", nil); err != nil || len(none.SpellIDs()) != 0 {
		t.Errorf("TextOption without spells = (%+v, %v) [empty case]", none, err)
	}
	_, err = content.NewTextOption("", "t", nil)
	wantBlockError(t, err, "TextOption", content.BlockInvalidName)
	_, err = content.NewTextOption("N", " ", nil)
	wantBlockError(t, err, "TextOption", content.BlockBlankText)
	if _, err = content.NewTextOption("N", "t", []content.SpellID{"spell_a", "Bad Id"}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("bad spell id error = %v, want ErrInvalidInput [invalid input]", err)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-016, FR-069 - a SpeciesChoice needs a trait
// name, at least one option, distinct option names and one variant.
// Path: per case.
func TestNewSpeciesChoice(t *testing.T) {
	t.Parallel()
	acid := content.MustTextOption("A", "t", nil)
	cases := []struct {
		name    string
		trait   string
		options []content.ChoiceOption
		want    content.BlockProblem // 0: accepted
	}{
		{"one text option", "T", []content.ChoiceOption{acid}, 0},
		{"damage options", "T", []content.ChoiceOption{mustDamage("A"), mustDamage("B")}, 0},
		{"blank trait name", " ", []content.ChoiceOption{acid}, content.BlockInvalidName},
		{"no options", "T", nil, content.BlockEmpty},
		{"nil option", "T", []content.ChoiceOption{nil}, content.BlockInvalidMember},
		{"zero-value option", "T", []content.ChoiceOption{content.TextOption{}}, content.BlockInvalidMember},
		{"nil text pointer", "T", []content.ChoiceOption{(*content.TextOption)(nil)}, content.BlockInvalidMember},
		{"text pointer", "T", []content.ChoiceOption{&content.TextOption{}}, content.BlockInvalidMember},
		{"valid text pointer", "T", []content.ChoiceOption{ptr(acid2("P"))}, content.BlockInvalidMember},
		{"valid lineage pointer", "T", []content.ChoiceOption{ptr(mustLineage("P"))}, content.BlockInvalidMember},
		{"valid damage pointer", "T", []content.ChoiceOption{ptr(mustDamage("P"))}, content.BlockInvalidMember},
		{"repeated option name", "T", []content.ChoiceOption{mustDamage("A"), mustDamage("A")}, content.BlockDuplicate},
		{"mixed variants", "T", []content.ChoiceOption{mustDamage("A"), acid2("B")}, content.BlockConstraint},
		{"mixed variants, text first", "T", []content.ChoiceOption{acid2("B"), mustDamage("A")}, content.BlockConstraint},
		{"mixed variants, lineage after text", "T", []content.ChoiceOption{acid2("B"), mustLineage("L")}, content.BlockConstraint},
		{"mixed variants, text after lineage", "T", []content.ChoiceOption{mustLineage("L"), acid2("B")}, content.BlockConstraint},
		{"lineage options", "T", []content.ChoiceOption{mustLineage("L"), mustLineage("M")}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch, err := content.NewSpeciesChoice(c.trait, c.options)
			if c.want != 0 {
				wantBlockError(t, err, "SpeciesChoice", c.want)
				return
			}
			if err != nil || ch.TraitName() != c.trait || len(ch.Options()) != len(c.options) {
				t.Fatalf("NewSpeciesChoice = (%+v, %v) [happy]", ch, err)
			}
			c.options[0] = nil
			ch.Options()[0] = nil
			if ch.Options()[0] == nil {
				t.Errorf("choice changed through an input or returned slice [immutability]")
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

func mustDamage(name string) content.DamageTypeOption {
	o, err := content.NewDamageTypeOption(name, content.DamageFire)
	if err != nil {
		panic(err)
	}
	return o
}

func acid2(name string) content.TextOption { return content.MustTextOption(name, "t", nil) }

func baseSpeciesFields(t *testing.T) content.SpeciesFields {
	t.Helper()
	return content.SpeciesFields{
		CreatureType: content.CreatureTypeHumanoid,
		Sizes:        []content.CreatureSize{content.SizeMedium},
		Speed:        30,
		Traits:       []content.NamedText{content.MustNamedText("T", "x"), content.MustNamedText("U", "y")},
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-010, FR-012 to FR-014, FR-060, FR-069 - the
// Species constructor rejects each invalid header, trait list and choice link, in the order of the members.
// Path: per case.
func TestNewSpecies(t *testing.T) {
	t.Parallel()
	choice := func(trait string) content.SpeciesChoice {
		c, err := content.NewSpeciesChoice(trait, []content.ChoiceOption{content.MustTextOption("A", "t", []content.SpellID{"spell_a"})})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	cases := []struct {
		name   string
		id     content.SpeciesID
		edit   func(*content.SpeciesFields)
		wantID bool
		want   content.BlockProblem // 0: accepted
	}{
		{name: "valid", id: "fixture"},
		{name: "two sizes", id: "fixture", edit: func(f *content.SpeciesFields) {
			f.Sizes = []content.CreatureSize{content.SizeSmall, content.SizeMedium}
		}},
		{name: "sizeText and choices", id: "fixture", edit: func(f *content.SpeciesFields) { f.SizeText = "s"; f.Choices = []content.SpeciesChoice{choice("T")} }},
		{name: "speed 0", id: "fixture", edit: func(f *content.SpeciesFields) { f.Speed = 0 }},
		{name: "speed at the cap", id: "fixture", edit: func(f *content.SpeciesFields) { f.Speed = content.SpeciesMaxSpeed }},
		{name: "empty id", id: "", wantID: true},
		{name: "malformed id", id: "Fixture", wantID: true},
		{name: "creature type zero", id: "fixture", edit: func(f *content.SpeciesFields) { f.CreatureType = 0 }, want: content.BlockInvalidMember},
		{name: "no sizes", id: "fixture", edit: func(f *content.SpeciesFields) { f.Sizes = nil }, want: content.BlockOutOfRange},
		{name: "three sizes", id: "fixture", edit: func(f *content.SpeciesFields) {
			f.Sizes = []content.CreatureSize{content.SizeSmall, content.SizeMedium, content.SizeSmall}
		}, want: content.BlockOutOfRange},
		{name: "repeated size", id: "fixture", edit: func(f *content.SpeciesFields) { f.Sizes = []content.CreatureSize{content.SizeSmall, content.SizeSmall} }, want: content.BlockDuplicate},
		{name: "Large", id: "fixture", edit: func(f *content.SpeciesFields) { f.Sizes = []content.CreatureSize{content.SizeLarge} }, want: content.BlockInvalidMember},
		{name: "Tiny", id: "fixture", edit: func(f *content.SpeciesFields) { f.Sizes = []content.CreatureSize{content.SizeTiny} }, want: content.BlockInvalidMember},
		{name: "blank sizeText", id: "fixture", edit: func(f *content.SpeciesFields) { f.SizeText = " " }, want: content.BlockBlankText},
		{name: "negative speed", id: "fixture", edit: func(f *content.SpeciesFields) { f.Speed = -1 }, want: content.BlockOutOfRange},
		{name: "speed above the cap", id: "fixture", edit: func(f *content.SpeciesFields) { f.Speed = plusOne(content.SpeciesMaxSpeed) }, want: content.BlockOutOfRange},
		{name: "no traits", id: "fixture", edit: func(f *content.SpeciesFields) { f.Traits = nil }, want: content.BlockEmpty},
		{name: "zero trait", id: "fixture", edit: func(f *content.SpeciesFields) { f.Traits = []content.NamedText{{}} }, want: content.BlockInvalidMember},
		{name: "repeated trait name", id: "fixture", edit: func(f *content.SpeciesFields) {
			f.Traits = []content.NamedText{content.MustNamedText("T", "x"), content.MustNamedText("T", "y")}
		}, want: content.BlockDuplicate},
		{name: "zero choice", id: "fixture", edit: func(f *content.SpeciesFields) { f.Choices = []content.SpeciesChoice{{}} }, want: content.BlockInvalidMember},
		{name: "choice for a trait the species lacks", id: "fixture", edit: func(f *content.SpeciesFields) { f.Choices = []content.SpeciesChoice{choice("Z")} }, want: content.BlockConstraint},
		{name: "choice for a trait that sorts before the species' traits", id: "fixture", edit: func(f *content.SpeciesFields) { f.Choices = []content.SpeciesChoice{choice("A")} }, want: content.BlockConstraint},
		{name: "trait with a name only", id: "fixture", edit: func(f *content.SpeciesFields) { f.Traits = []content.NamedText{content.MustNamedText("T", "x")} }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := baseSpeciesFields(t)
			if c.edit != nil {
				c.edit(&f)
			}
			src := content.MustSourceCitation("SRD 5.2.1", "Section", 84)
			s, err := content.NewSpecies(c.id, "Fixture", src, f)
			switch {
			case c.wantID:
				var ie *content.IDError
				if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("error = %v, want *IDError [FR-004, unhappy (invalid input)]", err)
				}
				assertMustPanics(t, func() { content.MustSpecies(c.id, "Fixture", src, f) })
			case c.want != 0:
				wantBlockError(t, err, "Species", c.want)
				assertMustPanics(t, func() { content.MustSpecies(c.id, "Fixture", src, f) })
			default:
				if err != nil || s.ID() != c.id || s.Name() != "Fixture" || s.Key() != content.NewKey(content.KindSpecies, content.MustID(string(c.id))) || s.Source() != src {
					t.Errorf("NewSpecies = (%+v, %v) [FR-012, happy]", s, err)
				}
				if s.CreatureType() != f.CreatureType || s.Speed() != f.Speed || s.SizeText() != f.SizeText || !slices.Equal(s.Sizes(), f.Sizes) || !slices.Equal(s.Traits(), f.Traits) || len(s.Choices()) != len(f.Choices) {
					t.Errorf("accessors disagree with the arguments: %+v", s)
				}
			}
		})
	}
	if _, err := content.NewSpecies("fixture", " ", content.SourceCitation{}, baseSpeciesFields(t)); !errors.Is(err, &content.BlockError{Block: "Species", Problem: content.BlockInvalidName}) {
		t.Errorf("blank name error = %v, want BlockInvalidName", err)
	}
	if s, err := content.NewSpecies("fixture", "Fixture", content.SourceCitation{}, baseSpeciesFields(t)); err != nil || !s.Source().IsZero() {
		t.Errorf("species without a source = (%+v, %v), want accepted with the zero citation", s, err)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-056 - a species is immutable: its slices are
// copied on construction and on read.
// Path: unhappy (immutability).
func TestSpeciesIsImmutable(t *testing.T) {
	t.Parallel()
	f := baseSpeciesFields(t)
	s := content.MustSpecies("fixture", "Fixture", content.SourceCitation{}, f)
	f.Sizes[0], f.Traits[0] = content.SizeSmall, content.MustNamedText("Changed", "c")
	s.Sizes()[0], s.Traits()[0] = content.SizeSmall, content.MustNamedText("Changed", "c")
	if s.Sizes()[0] != content.SizeMedium || s.Traits()[0].Name() != "T" {
		t.Errorf("species changed through an input or returned slice: %v, %v", s.Sizes(), s.Traits())
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-021, FR-023 - Species lists exactly one site per
// spell id of its choice options, in order, with the target of kind spell and no constraint.
// Path: happy.
func TestSpeciesReferences(t *testing.T) {
	t.Parallel()
	lineage, err := content.NewLineageOption("L", []content.LevelBenefit{
		content.MustLevelBenefit(1, "a", "spell_a"), content.MustLevelBenefit(3, "b", "spell_b"), content.MustLevelBenefit(5, "c", "spell_c"),
	})
	if err != nil {
		t.Fatal(err)
	}
	f := baseSpeciesFields(t)
	f.Choices = []content.SpeciesChoice{
		content.MustSpeciesChoice("T", []content.ChoiceOption{lineage}),
		content.MustSpeciesChoice("U", []content.ChoiceOption{
			content.MustDamageTypeOption("D", content.DamageAcid),
		}),
		content.MustSpeciesChoice("U", []content.ChoiceOption{content.MustTextOption("X", "t", nil), content.MustTextOption("Y", "t", []content.SpellID{"spell_d", "spell_e"})}),
	}
	refs := content.MustSpecies("fixture", "Fixture", content.SourceCitation{}, f).References()
	var got [][2]string
	for _, r := range refs {
		if r.Target().Kind() != content.KindSpell || r.Constraint() != nil {
			t.Errorf("reference %q targets %v with constraint %v, want a spell and none", r.Field(), r.Target(), r.Constraint())
		}
		got = append(got, [2]string{r.Field(), r.Target().ID().String()})
	}
	want := [][2]string{
		{"choices/0/options/0/levelBenefits/0/spellId", "spell_a"}, {"choices/0/options/0/levelBenefits/1/spellId", "spell_b"},
		{"choices/0/options/0/levelBenefits/2/spellId", "spell_c"},
		{"choices/2/options/1/spellIds/0", "spell_d"}, {"choices/2/options/1/spellIds/1", "spell_e"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("sites = %v, want %v", got, want)
	}
	if none := content.MustSpecies("fixture", "Fixture", content.SourceCitation{}, baseSpeciesFields(t)).References(); len(none) != 0 {
		t.Errorf("a species without choices lists %v, want no sites [empty case]", none)
	}
}

func baseBackgroundFields() content.BackgroundFields {
	return content.BackgroundFields{
		AbilityScores:      []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution},
		Feat:               content.MustBackgroundFeatReference("feat_a", ""),
		SkillProficiencies: []content.Skill{content.SkillAthletics, content.SkillStealth},
		ToolProficiency:    mustSpecificTool("Tool"),
		EquipmentOptions:   content.NewEquipmentOptions(content.EquipmentOption{}, content.EquipmentOption{}),
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-024 to FR-030, FR-069 - the Background
// constructor rejects each invalid member.
// Path: per case.
func TestNewBackground(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		id     content.BackgroundID
		edit   func(*content.BackgroundFields)
		wantID bool
		want   content.BlockProblem // 0: accepted
	}{
		{name: "valid", id: "fixture"},
		{name: "choice tool proficiency", id: "fixture", edit: func(f *content.BackgroundFields) { f.ToolProficiency = mustChoiceTool("Set") }},
		{name: "bad id", id: "A", wantID: true},
		{name: "two abilities", id: "fixture", edit: func(f *content.BackgroundFields) { f.AbilityScores = f.AbilityScores[:2] }, want: content.BlockOutOfRange},
		{name: "four abilities", id: "fixture", edit: func(f *content.BackgroundFields) { f.AbilityScores = append(f.AbilityScores, ability.Wisdom) }, want: content.BlockOutOfRange},
		{name: "repeated ability", id: "fixture", edit: func(f *content.BackgroundFields) { f.AbilityScores[1] = ability.Strength }, want: content.BlockDuplicate},
		{name: "invalid ability", id: "fixture", edit: func(f *content.BackgroundFields) { f.AbilityScores[1] = 0 }, want: content.BlockInvalidMember},
		{name: "zero feat reference", id: "fixture", edit: func(f *content.BackgroundFields) { f.Feat = content.BackgroundFeatReference{} }, wantID: true},
		{name: "one skill", id: "fixture", edit: func(f *content.BackgroundFields) { f.SkillProficiencies = f.SkillProficiencies[:1] }, want: content.BlockOutOfRange},
		{name: "three skills", id: "fixture", edit: func(f *content.BackgroundFields) {
			f.SkillProficiencies = append(f.SkillProficiencies, content.SkillArcana)
		}, want: content.BlockOutOfRange},
		{name: "repeated skill", id: "fixture", edit: func(f *content.BackgroundFields) { f.SkillProficiencies[1] = content.SkillAthletics }, want: content.BlockDuplicate},
		{name: "invalid skill", id: "fixture", edit: func(f *content.BackgroundFields) { f.SkillProficiencies[1] = 0 }, want: content.BlockInvalidMember},
		{name: "no tool proficiency", id: "fixture", edit: func(f *content.BackgroundFields) { f.ToolProficiency = nil }, want: content.BlockMissing},
		{name: "zero specific tool", id: "fixture", edit: func(f *content.BackgroundFields) { f.ToolProficiency = content.SpecificToolProficiency{} }, want: content.BlockInvalidMember},
		{name: "zero choice tool", id: "fixture", edit: func(f *content.BackgroundFields) { f.ToolProficiency = content.ChoiceToolProficiency{} }, want: content.BlockInvalidMember},
		{name: "nil specific tool pointer", id: "fixture", edit: func(f *content.BackgroundFields) { f.ToolProficiency = (*content.SpecificToolProficiency)(nil) }, want: content.BlockInvalidMember},
		{name: "choice tool pointer", id: "fixture", edit: func(f *content.BackgroundFields) { f.ToolProficiency = ptr(mustChoiceTool("Set")) }, want: content.BlockInvalidMember},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := baseBackgroundFields()
			if c.edit != nil {
				c.edit(&f)
			}
			b, err := content.NewBackground(c.id, "Fixture", content.SourceCitation{}, f)
			switch {
			case c.wantID:
				var ie *content.IDError
				if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("error = %v, want *IDError [invalid input]", err)
				}
			case c.want != 0:
				wantBlockError(t, err, "Background", c.want)
				assertMustPanics(t, func() { content.MustBackground(c.id, "Fixture", content.SourceCitation{}, f) })
			default:
				if err != nil || b.ID() != c.id || b.Name() != "Fixture" || b.Key() != content.NewKey(content.KindBackground, content.MustID(string(c.id))) {
					t.Fatalf("NewBackground = (%+v, %v) [happy]", b, err)
				}
				if !slices.Equal(b.AbilityScores(), f.AbilityScores) || !slices.Equal(b.SkillProficiencies(), f.SkillProficiencies) || b.ToolProficiency() != f.ToolProficiency || b.Feat() != f.Feat {
					t.Errorf("accessors disagree with the arguments: %+v", b)
				}
			}
		})
	}
	if _, err := content.NewBackground("fixture", "", content.SourceCitation{}, baseBackgroundFields()); !errors.Is(err, &content.BlockError{Block: "Background", Problem: content.BlockInvalidName}) {
		t.Errorf("empty name error = %v, want BlockInvalidName", err)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-056, FR-025 - a background is immutable and its
// one site is feat/featId, an Origin feat.
// Path: happy and unhappy (immutability).
func TestBackgroundImmutabilityAndSite(t *testing.T) {
	t.Parallel()
	f := baseBackgroundFields()
	b := content.MustBackground("fixture", "Fixture", content.SourceCitation{}, f)
	f.AbilityScores[0], f.SkillProficiencies[0] = ability.Charisma, content.SkillArcana
	b.AbilityScores()[1], b.SkillProficiencies()[1] = ability.Charisma, content.SkillArcana
	if b.AbilityScores()[0] != ability.Strength || b.AbilityScores()[1] != ability.Dexterity || b.SkillProficiencies()[0] != content.SkillAthletics || b.SkillProficiencies()[1] != content.SkillStealth {
		t.Errorf("background changed through an input or returned slice")
	}
	refs := b.References()
	want := content.MustReference("feat/featId", content.NewKey(content.KindFeat, content.MustID("feat_a")), content.TargetHasFeatCategory{Category: content.FeatOrigin})
	if len(refs) != 1 || refs[0] != want {
		t.Errorf("References = %v, want the one Origin-feat site %v", refs, want)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-026, FR-029, FR-030 - the equipment and feat
// reference constructors validate their arguments.
// Path: per case.
func TestEquipmentAndFeatReferenceConstructors(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		qty  int
		want content.BlockProblem
	}{
		{"quantity 1", 1, 0}, {"quantity at the cap", content.MaxEquipmentQuantity, 0},
		{"quantity 0", 0, content.BlockOutOfRange}, {"negative quantity", -1, content.BlockOutOfRange},
		{"quantity above the cap", plusOne(content.MaxEquipmentQuantity), content.BlockOutOfRange},
	} {
		_, err := content.NewEquipmentItem("Rope", c.qty)
		if c.want == 0 && err != nil || c.want != 0 && !errors.Is(err, &content.BlockError{Block: "EquipmentItem", Problem: c.want}) {
			t.Errorf("%s: NewEquipmentItem error = %v, want %v [FR-030]", c.name, err, c.want)
		}
	}
	if _, err := content.NewEquipmentItem(" Rope", 1); !errors.Is(err, &content.BlockError{Block: "EquipmentItem", Problem: content.BlockInvalidName}) {
		t.Errorf("padded item name error = %v, want BlockInvalidName", err)
	}
	item := content.MustEquipmentItem("Book (history)", 1)
	opt, err := content.NewEquipmentOption([]content.EquipmentItem{item}, 8)
	if err != nil || opt.Gold() != 8 || len(opt.Items()) != 1 || opt.Items()[0] != item {
		t.Errorf("NewEquipmentOption = (%+v, %v) [FR-029, happy]", opt, err)
	}
	opt.Items()[0] = content.EquipmentItem{}
	if opt.Items()[0] != item {
		t.Errorf("option changed through a returned slice [immutability]")
	}
	for _, c := range []struct {
		name  string
		items []content.EquipmentItem
		gold  int
		want  content.BlockProblem
	}{
		{"no items, no gold", nil, 0, 0}, {"gold at the cap", nil, content.MaxEquipmentGold, 0},
		{"negative gold", nil, -1, content.BlockOutOfRange}, {"gold above the cap", nil, plusOne(content.MaxEquipmentGold), content.BlockOutOfRange},
		{"zero item", []content.EquipmentItem{{}}, 0, content.BlockInvalidMember},
	} {
		_, err := content.NewEquipmentOption(c.items, c.gold)
		if c.want == 0 && err != nil || c.want != 0 && !errors.Is(err, &content.BlockError{Block: "EquipmentOption", Problem: c.want}) {
			t.Errorf("%s: NewEquipmentOption error = %v, want %v [FR-029]", c.name, err, c.want)
		}
	}
	pair := content.NewEquipmentOptions(opt, content.EquipmentOption{})
	if pair.A().Gold() != 8 || pair.B().Gold() != 0 {
		t.Errorf("EquipmentOptions A/B = %d/%d, want 8/0", pair.A().Gold(), pair.B().Gold())
	}
	if r, err := content.NewBackgroundFeatReference("feat_a", "Word"); err != nil || r.FeatID() != "feat_a" || r.Choice() != "Word" {
		t.Errorf("NewBackgroundFeatReference = (%+v, %v) [FR-026, happy]", r, err)
	}
	if r, err := content.NewBackgroundFeatReference("feat_a", ""); err != nil || r.Choice() != "" {
		t.Errorf("NewBackgroundFeatReference without a choice = (%+v, %v)", r, err)
	}
	if _, err := content.NewBackgroundFeatReference("", "x"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("empty feat id error = %v, want ErrInvalidInput", err)
	}
	if _, err := content.NewBackgroundFeatReference("feat_a", " "); !errors.Is(err, &content.BlockError{Block: "BackgroundFeatReference", Problem: content.BlockBlankText}) {
		t.Errorf("blank choice error = %v, want BlockBlankText [FR-026]", err)
	}
	if _, err := content.NewBackgroundFeatReference("feat_a", "a\nb"); !errors.Is(err, &content.BlockError{Block: "BackgroundFeatReference", Problem: content.BlockInvalidCharacter}) {
		t.Errorf("choice with a line feed error = %v, want BlockInvalidCharacter [FR-026]", err)
	}
}

func baseFeatFields() content.FeatFields {
	return content.FeatFields{
		Category: content.FeatOrigin,
		Benefits: []content.FeatBenefit{content.MustFeatBenefit("", "x")},
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-033, FR-035, FR-039, FR-063, FR-069 - the Feat
// constructor rejects each invalid member and each repeatable-text mismatch.
// Path: per case.
func TestNewFeat(t *testing.T) {
	t.Parallel()
	level := func(n int) content.Prerequisite {
		p, err := content.NewLevelPrerequisite(n)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	cases := []struct {
		name   string
		id     content.FeatID
		edit   func(*content.FeatFields)
		wantID bool
		want   content.BlockProblem // 0: accepted
	}{
		{name: "valid", id: "feat_a"},
		{name: "with prerequisites", id: "feat_a", edit: func(f *content.FeatFields) { f.Prerequisites = []content.Prerequisite{level(4)} }},
		{name: "repeatable with text", id: "feat_a", edit: func(f *content.FeatFields) { f.Repeatable, f.RepeatableText = true, "r" }},
		{name: "bad id", id: "Feat A", wantID: true},
		{name: "category zero", id: "feat_a", edit: func(f *content.FeatFields) { f.Category = 0 }, want: content.BlockInvalidMember},
		{name: "empty prerequisites", id: "feat_a", edit: func(f *content.FeatFields) { f.Prerequisites = []content.Prerequisite{} }, want: content.BlockEmpty},
		{name: "nil prerequisite", id: "feat_a", edit: func(f *content.FeatFields) { f.Prerequisites = []content.Prerequisite{nil} }, want: content.BlockInvalidMember},
		{name: "zero prerequisite", id: "feat_a", edit: func(f *content.FeatFields) { f.Prerequisites = []content.Prerequisite{content.LevelPrerequisite{}} }, want: content.BlockInvalidMember},
		{name: "zero ability prerequisite", id: "feat_a", edit: func(f *content.FeatFields) {
			f.Prerequisites = []content.Prerequisite{content.AbilityScorePrerequisite{}}
		}, want: content.BlockInvalidMember},
		{name: "zero feature prerequisite", id: "feat_a", edit: func(f *content.FeatFields) { f.Prerequisites = []content.Prerequisite{content.FeaturePrerequisite{}} }, want: content.BlockInvalidMember},
		{name: "nil prerequisite pointer", id: "feat_a", edit: func(f *content.FeatFields) {
			f.Prerequisites = []content.Prerequisite{(*content.LevelPrerequisite)(nil)}
		}, want: content.BlockInvalidMember},
		{name: "valid prerequisite pointer", id: "feat_a", edit: func(f *content.FeatFields) {
			f.Prerequisites = []content.Prerequisite{ptr(level(4).(content.LevelPrerequisite))}
		}, want: content.BlockInvalidMember},
		{name: "no benefits", id: "feat_a", edit: func(f *content.FeatFields) { f.Benefits = nil }, want: content.BlockEmpty},
		{name: "zero benefit", id: "feat_a", edit: func(f *content.FeatFields) { f.Benefits = []content.FeatBenefit{{}} }, want: content.BlockInvalidMember},
		{name: "repeatable without text", id: "feat_a", edit: func(f *content.FeatFields) { f.Repeatable = true }, want: content.BlockMissing},
		{name: "text without repeatable", id: "feat_a", edit: func(f *content.FeatFields) { f.RepeatableText = "r" }, want: content.BlockConstraint},
		{name: "blank repeatable text", id: "feat_a", edit: func(f *content.FeatFields) { f.Repeatable, f.RepeatableText = true, " " }, want: content.BlockBlankText},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := baseFeatFields()
			if c.edit != nil {
				c.edit(&f)
			}
			v, err := content.NewFeat(c.id, "Feat", content.SourceCitation{}, f)
			switch {
			case c.wantID:
				var ie *content.IDError
				if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("error = %v, want *IDError", err)
				}
			case c.want != 0:
				wantBlockError(t, err, "Feat", c.want)
				assertMustPanics(t, func() { content.MustFeat(c.id, "Feat", content.SourceCitation{}, f) })
			default:
				if err != nil || v.ID() != c.id || v.Name() != "Feat" || v.Key() != content.NewKey(content.KindFeat, content.MustID(string(c.id))) {
					t.Fatalf("NewFeat = (%+v, %v) [happy]", v, err)
				}
				if v.FeatCategory() != f.Category || v.Repeatable() != f.Repeatable || v.RepeatableText() != f.RepeatableText || len(v.Benefits()) != 1 || len(v.Prerequisites()) != len(f.Prerequisites) {
					t.Errorf("accessors disagree with the arguments: %+v", v)
				}
			}
		})
	}
	if _, err := content.NewFeat("feat_a", "", content.SourceCitation{}, baseFeatFields()); !errors.Is(err, &content.BlockError{Block: "Feat", Problem: content.BlockInvalidName}) {
		t.Errorf("empty name error = %v, want BlockInvalidName", err)
	}
	if v := content.MustFeat("feat_a", "Feat", content.SourceCitation{}, baseFeatFields()); v.Prerequisites() != nil {
		t.Errorf("a feat without prerequisites returned %v, want nil [empty case]", v.Prerequisites())
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-025, FR-056, FR-063 - a Feat implements
// HasFeatCategory, is immutable, and a benefit is a name that may be absent and a verbatim text.
// Path: happy and unhappy (immutability).
func TestFeatImmutabilityAndBenefits(t *testing.T) {
	t.Parallel()
	var _ content.HasFeatCategory = content.Feat{}
	f := baseFeatFields()
	f.Prerequisites = []content.Prerequisite{mustLevel(4)}
	v := content.MustFeat("feat_a", "Feat", content.SourceCitation{}, f)
	f.Benefits[0] = content.FeatBenefit{}
	f.Prerequisites[0] = mustLevel(9)
	v.Benefits()[0] = content.FeatBenefit{}
	v.Prerequisites()[0] = mustLevel(9)
	if v.Benefits()[0].Text() != "x" || v.Prerequisites()[0].(content.LevelPrerequisite).Minimum() != 4 {
		t.Errorf("feat changed through an input or returned slice")
	}
	if b, err := content.NewFeatBenefit("N", "Line.\nTwo."); err != nil || b.Name() != "N" || b.Text() != "Line.\nTwo." {
		t.Errorf("NewFeatBenefit named = (%+v, %v) [FR-040, happy]", b, err)
	}
	if b, err := content.NewFeatBenefit("", "t"); err != nil || b.Name() != "" {
		t.Errorf("NewFeatBenefit unnamed = (%+v, %v) [FR-040, happy]", b, err)
	}
	_, err := content.NewFeatBenefit(" N", "t")
	wantBlockError(t, err, "FeatBenefit", content.BlockInvalidName)
	_, err = content.NewFeatBenefit("N", " ")
	wantBlockError(t, err, "FeatBenefit", content.BlockBlankText)
}

func mustLevel(n int) content.LevelPrerequisite {
	p, err := content.NewLevelPrerequisite(n)
	if err != nil {
		panic(err)
	}
	return p
}

func mustSpecificTool(name string) content.SpecificToolProficiency {
	p, err := content.NewSpecificToolProficiency(name)
	if err != nil {
		panic(err)
	}
	return p
}

func mustChoiceTool(category string) content.ChoiceToolProficiency {
	p, err := content.NewChoiceToolProficiency(category)
	if err != nil {
		panic(err)
	}
	return p
}

// plusOne returns n + 1 computed at run time, so a test of "one above the cap"
// compiles where int is 32 bits wide (the sum then wraps to a negative value,
// which is out of range as well).
func plusOne(n int) int { return int(int64(n) + 1) }

func mustLineage(name string) content.LineageOption {
	return content.MustLineageOption(name, mustLevelBenefitsOf(1, 3, 5))
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-069 and schema FR-093 - every MustX of this step
// returns the value NewX does for valid arguments and panics with its error for invalid ones.
// Path: per case.
func TestMustConstructors(t *testing.T) {
	t.Parallel()
	benefit := content.MustLevelBenefit(1, "t", "spell_a")
	cases := []struct {
		name  string
		valid func()
		bad   func()
	}{
		{"MustDamageTypeOption", func() { content.MustDamageTypeOption("N", content.DamageAcid) }, func() { content.MustDamageTypeOption("", content.DamageAcid) }},
		{"MustLevelBenefit", func() { content.MustLevelBenefit(1, "t", "spell_a") }, func() { content.MustLevelBenefit(2, "t", "spell_a") }},
		{"MustLineageOption", func() {
			content.MustLineageOption("N", []content.LevelBenefit{benefit, content.MustLevelBenefit(3, "t", "spell_a"), content.MustLevelBenefit(5, "t", "spell_a")})
		}, func() { content.MustLineageOption("N", nil) }},
		{"MustTextOption", func() { content.MustTextOption("N", "t", nil) }, func() { content.MustTextOption("N", " ", nil) }},
		{"MustSpeciesChoice", func() { content.MustSpeciesChoice("T", []content.ChoiceOption{content.MustTextOption("N", "t", nil)}) }, func() { content.MustSpeciesChoice("T", nil) }},
		{"MustEquipmentItem", func() { content.MustEquipmentItem("N", 1) }, func() { content.MustEquipmentItem("N", 0) }},
		{"MustEquipmentOption", func() { content.MustEquipmentOption(nil, 1) }, func() { content.MustEquipmentOption(nil, -1) }},
		{"MustBackgroundFeatReference", func() { content.MustBackgroundFeatReference("feat_a", "") }, func() { content.MustBackgroundFeatReference("", "") }},
		{"MustFeatBenefit", func() { content.MustFeatBenefit("", "t") }, func() { content.MustFeatBenefit("", " ") }},
		{"MustEquipmentItemValue", func() { content.MustEquipmentItemValue(0, false) }, func() { content.MustEquipmentItemValue(-1, false) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			c.valid()
			assertMustPanics(t, c.bad)
		})
	}
}

func mustLevelBenefitsOf(levels ...int) []content.LevelBenefit {
	var out []content.LevelBenefit
	for _, l := range levels {
		out = append(out, content.MustLevelBenefit(l, "text", "spell_a"))
	}
	return out
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-056 - the choices of a species are copied on read.
// Path: unhappy (immutability).
func TestSpeciesChoicesAreCopiedOnRead(t *testing.T) {
	t.Parallel()
	f := baseSpeciesFields(t)
	c, err := content.NewSpeciesChoice("T", []content.ChoiceOption{content.MustTextOption("A", "t", nil)})
	if err != nil {
		t.Fatal(err)
	}
	f.Choices = []content.SpeciesChoice{c}
	s := content.MustSpecies("fixture", "Fixture", content.SourceCitation{}, f)
	s.Choices()[0] = content.SpeciesChoice{}
	if got := s.Choices()[0].TraitName(); got != "T" {
		t.Errorf("choice trait name after changing a returned slice = %q, want %q", got, "T")
	}
}
