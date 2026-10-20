package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

func equalFixtures(t *testing.T) (content.Species, content.Background, content.Feat) {
	t.Helper()
	sp := content.MustSpecies("fixture", "Fixture", content.SourceCitation{}, content.SpeciesFields{
		CreatureType: content.CreatureTypeHumanoid, Sizes: []content.CreatureSize{content.SizeMedium}, Speed: 30,
		Traits: []content.NamedText{content.MustNamedText("T", "x"), content.MustNamedText("U", "y")},
		Choices: []content.SpeciesChoice{content.MustSpeciesChoice("T", []content.ChoiceOption{
			content.MustTextOption("A", "x", []content.SpellID{}),
		})},
	})
	bg := content.MustBackground("fixture", "Fixture", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores: []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution},
		Feat:          content.MustBackgroundFeatReference("feat_a", ""), SkillProficiencies: []content.Skill{content.SkillAthletics, content.SkillStealth},
		ToolProficiency: tool("Tool"),
		EquipmentOptions: content.NewEquipmentOptions(
			content.MustEquipmentOption([]content.EquipmentItem{content.MustEquipmentItem("Rope", 2)}, 10),
			content.MustEquipmentOption([]content.EquipmentItem{}, 50)),
	})
	ft := content.MustFeat("feat_a", "Feat A", content.SourceCitation{}, content.FeatFields{
		Category: content.FeatOrigin, Benefits: []content.FeatBenefit{content.MustFeatBenefit("", "x")},
	})
	return sp, bg, ft
}

// Requirement: go.md section 5 - a type holding a slice provides Equal; an entry built from empty lists equals
// the same entry decoded from "[]" or from absent members (nil and empty are the same "none").
// Path: happy (constructor versus decoder), unhappy (empty versus nil).
func TestConstructedEntriesEqualDecodedWithEmptyLists(t *testing.T) {
	t.Parallel()
	sp, bg, ft := equalFixtures(t)
	doc := `{"schemaVersion":1,"species":[` + mutate(speciesJSON(choiceJSON(`{"kind":"text","name":"A","text":"x","spellIds":[]}`)), `"speed":30`, `"speed":30`) +
		`],"backgrounds":[` + backgroundJSON + `],"feats":[` + featJSON + `]}`
	if !decodeSet(t, doc).Equal(content.MustSet(sp, bg, ft)) {
		t.Error("decoded set differs from the constructed one, want Equal with empty lists")
	}
	nilSp := content.MustSpecies("fixture", "Fixture", content.SourceCitation{}, content.SpeciesFields{
		CreatureType: content.CreatureTypeHumanoid, Sizes: []content.CreatureSize{content.SizeMedium}, Speed: 30,
		Traits:  []content.NamedText{content.MustNamedText("T", "x"), content.MustNamedText("U", "y")},
		Choices: []content.SpeciesChoice{content.MustSpeciesChoice("T", []content.ChoiceOption{content.MustTextOption("A", "x", nil)})},
	})
	if !content.MustSet(sp).Equal(content.MustSet(nilSp)) || !sp.Equal(nilSp) {
		t.Error("empty and nil spellIds differ, want equal")
	}
	noChoices := content.SpeciesFields{
		CreatureType: content.CreatureTypeHumanoid, Sizes: []content.CreatureSize{content.SizeMedium}, Speed: 30,
		Traits: []content.NamedText{content.MustNamedText("T", "x")},
	}
	a, b := noChoices, noChoices
	a.Choices = []content.SpeciesChoice{}
	if !content.MustSpecies("s", "S", content.SourceCitation{}, a).Equal(content.MustSpecies("s", "S", content.SourceCitation{}, b)) {
		t.Error("empty and nil choices differ, want equal")
	}
	if !content.MustEquipmentOption([]content.EquipmentItem{}, 5).Equal(content.MustEquipmentOption(nil, 5)) {
		t.Error("empty and nil items differ, want equal")
	}
}

// Requirement: go.md section 5 - Equal compares every member, lists in order, and variants of a closed set
// are different when their variant differs.
// Path: unhappy (one member differs).
func TestEqualDetectsEachDifference(t *testing.T) {
	t.Parallel()
	sp, bg, ft := equalFixtures(t)
	if !sp.Equal(sp) || !bg.Equal(bg) || !ft.Equal(ft) {
		t.Fatal("an entry is not Equal to itself")
	}
	rope := content.MustEquipmentItem("Rope", 2)
	other := content.MustEquipmentItem("Rope", 3)
	opt := func(items []content.EquipmentItem, gold int) content.EquipmentOption {
		return content.MustEquipmentOption(items, gold)
	}
	dmg := content.MustDamageTypeOption("A", content.DamageFire)
	txt := content.MustTextOption("A", "x", []content.SpellID{"spell_a"})
	ch := func(o ...content.ChoiceOption) content.SpeciesChoice { return content.MustSpeciesChoice("T", o) }
	cases := []struct {
		name string
		same bool
	}{
		{"items differ", opt([]content.EquipmentItem{rope}, 1).Equal(opt([]content.EquipmentItem{other}, 1))},
		{"item count differs", opt([]content.EquipmentItem{rope}, 1).Equal(opt(nil, 1))},
		{"gold differs", opt(nil, 1).Equal(opt(nil, 2))},
		{"options A differ", content.NewEquipmentOptions(opt(nil, 1), opt(nil, 2)).Equal(content.NewEquipmentOptions(opt(nil, 3), opt(nil, 2)))},
		{"options B differ", content.NewEquipmentOptions(opt(nil, 1), opt(nil, 2)).Equal(content.NewEquipmentOptions(opt(nil, 1), opt(nil, 3)))},
		{"text option spells differ", txt.Equal(content.MustTextOption("A", "x", []content.SpellID{"spell_b"}))},
		{"text option name differs", txt.Equal(content.MustTextOption("B", "x", []content.SpellID{"spell_a"}))},
		{"text option text differs", txt.Equal(content.MustTextOption("A", "y", []content.SpellID{"spell_a"}))},
		{"choice trait differs", ch(txt).Equal(content.MustSpeciesChoice("U", []content.ChoiceOption{txt}))},
		{"choice variant differs", ch(dmg).Equal(ch(txt))},
		{"choice variant differs, text first", ch(txt).Equal(ch(dmg))},
		{"choice option differs", ch(dmg).Equal(ch(content.MustDamageTypeOption("A", content.DamageCold)))},
		{"choice text option differs", ch(txt).Equal(ch(content.MustTextOption("A", "y", nil)))},
		{"choice option count differs", ch(dmg).Equal(ch(dmg, content.MustDamageTypeOption("B", content.DamageCold)))},
	}
	for _, c := range cases {
		if c.same {
			t.Errorf("%s: Equal = true, want false", c.name)
		}
	}
	if !ch(txt, content.MustTextOption("B", "x", nil)).Equal(ch(txt, content.MustTextOption("B", "x", nil))) || !ch(dmg).Equal(ch(dmg)) {
		t.Error("equal choices are not Equal")
	}
}

func tool(name string) content.SpecificToolProficiency {
	p, err := content.NewSpecificToolProficiency(name)
	if err != nil {
		panic(err)
	}
	return p
}

// Requirement: go.md section 5 - Feat.Equal and Background.Equal compare every member, including the
// prerequisite variants and the tool proficiency.
// Path: unhappy (one member differs).
func TestFeatAndBackgroundEqualDetectDifferences(t *testing.T) {
	t.Parallel()
	_, bg, ft := equalFixtures(t)
	lvl, _ := content.NewLevelPrerequisite(4)
	lvl2, _ := content.NewLevelPrerequisite(5)
	abil, _ := content.NewAbilityScorePrerequisite([]ability.Ability{ability.Strength}, 13)
	abil2, _ := content.NewAbilityScorePrerequisite([]ability.Ability{ability.Strength}, 15)
	feature, _ := content.NewFeaturePrerequisite("Spellcasting")
	feat := func(name string, benefit string, rep bool, repText string, ps ...content.Prerequisite) content.Feat {
		return content.MustFeat("feat_a", name, content.SourceCitation{}, content.FeatFields{
			Category: content.FeatOrigin, Prerequisites: ps, Repeatable: rep, RepeatableText: repText,
			Benefits: []content.FeatBenefit{content.MustFeatBenefit("", benefit)},
		})
	}
	base := feat("Feat A", "x", false, "", lvl, abil)
	if !base.Equal(feat("Feat A", "x", false, "", lvl, abil)) {
		t.Error("equal feats are not Equal")
	}
	for name, other := range map[string]content.Feat{
		"name":            feat("Feat B", "x", false, "", lvl, abil),
		"benefit":         feat("Feat A", "y", false, "", lvl, abil),
		"repeatable":      feat("Feat A", "x", true, "r", lvl, abil),
		"level":           feat("Feat A", "x", false, "", lvl2, abil),
		"ability minimum": feat("Feat A", "x", false, "", lvl, abil2),
		"variant":         feat("Feat A", "x", false, "", feature, abil),
		"variant order":   feat("Feat A", "x", false, "", abil, lvl),
		"count":           feat("Feat A", "x", false, "", lvl),
		"none":            feat("Feat A", "x", false, ""),
		"category":        content.MustFeat("feat_a", "Feat A", content.SourceCitation{}, content.FeatFields{Category: content.FeatGeneral, Benefits: ft.Benefits()}),
	} {
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("feat differing in %s is Equal", name)
		}
	}
	bgWith := func(mod func(*content.BackgroundFields)) content.Background {
		f := content.BackgroundFields{
			AbilityScores: bg.AbilityScores(), Feat: bg.Feat(), SkillProficiencies: bg.SkillProficiencies(),
			ToolProficiency: bg.ToolProficiency(), EquipmentOptions: bg.EquipmentOptions(),
		}
		mod(&f)
		return content.MustBackground("fixture", "Fixture", content.SourceCitation{}, f)
	}
	if !bg.Equal(bgWith(func(*content.BackgroundFields) {})) {
		t.Error("equal backgrounds are not Equal")
	}
	for name, other := range map[string]content.Background{
		"abilities": bgWith(func(f *content.BackgroundFields) {
			f.AbilityScores = []ability.Ability{ability.Strength, ability.Dexterity, ability.Wisdom}
		}),
		"feat": bgWith(func(f *content.BackgroundFields) { f.Feat = content.MustBackgroundFeatReference("feat_a", "Word") }),
		"skills": bgWith(func(f *content.BackgroundFields) {
			f.SkillProficiencies = []content.Skill{content.SkillAthletics, content.SkillArcana}
		}),
		"tool": bgWith(func(f *content.BackgroundFields) { f.ToolProficiency = tool("Other") }),
		"equipment": bgWith(func(f *content.BackgroundFields) {
			f.EquipmentOptions = content.NewEquipmentOptions(content.EquipmentOption{}, content.EquipmentOption{})
		}),
	} {
		if bg.Equal(other) {
			t.Errorf("background differing in %s is Equal", name)
		}
	}
}

// distinct fails unless a and b are unequal in both argument orders. Checking
// both orders matters: a comparison written as <= or >= instead of == is wrong
// in exactly one of them.
func distinct[T interface{ Equal(T) bool }](t *testing.T, field string, a, b T) {
	t.Helper()
	if a.Equal(b) || b.Equal(a) {
		t.Errorf("%s differs but Equal = true [isolated difference]", field)
	}
}

// Requirement: go.md section 5 - Equal compares every member, and one differing member is enough, whichever
// way round the operands are.
// Path: unhappy (exactly one member differs, both orders).
func TestEqualIsolatesEachMember(t *testing.T) {
	t.Parallel()
	cite := func(page int) content.SourceCitation {
		c, err := content.NewSourceCitation("Doc", "Section", page)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	item := func(n string, q int) []content.EquipmentItem {
		return []content.EquipmentItem{content.MustEquipmentItem(n, q)}
	}

	distinct(t, "item gold", content.MustEquipmentOption(nil, 1), content.MustEquipmentOption(nil, 2))
	distinct(t, "item name", content.MustEquipmentOption(item("A", 1), 1), content.MustEquipmentOption(item("B", 1), 1))
	distinct(t, "text option name", content.MustTextOption("A", "x", nil), content.MustTextOption("B", "x", nil))
	distinct(t, "text option text", content.MustTextOption("A", "x", nil), content.MustTextOption("A", "y", nil))
	text := content.MustTextOption("A", "x", nil)
	distinct(t, "choice trait", content.MustSpeciesChoice("T", []content.ChoiceOption{text}), content.MustSpeciesChoice("U", []content.ChoiceOption{text}))

	nt := func(n string) []content.NamedText { return []content.NamedText{content.MustNamedText(n, "x")} }
	sp := func(id content.SpeciesID, name string, src content.SourceCitation, ct content.CreatureType, sizeText string, speed int,
		sizes []content.CreatureSize, traits []content.NamedText, choices []content.SpeciesChoice) content.Species {
		return content.MustSpecies(id, name, src, content.SpeciesFields{
			CreatureType: ct, Sizes: sizes, SizeText: sizeText, Speed: speed, Traits: traits, Choices: choices,
		})
	}
	med, small := []content.CreatureSize{content.SizeMedium}, []content.CreatureSize{content.SizeSmall}
	base := func() content.Species {
		return sp("a", "A", cite(1), content.CreatureTypeHumanoid, "s", 30, med, nt("T"), nil)
	}
	distinct(t, "species id", base(), sp("b", "A", cite(1), content.CreatureTypeHumanoid, "s", 30, med, nt("T"), nil))
	distinct(t, "species name", base(), sp("a", "B", cite(1), content.CreatureTypeHumanoid, "s", 30, med, nt("T"), nil))
	distinct(t, "species source", base(), sp("a", "A", cite(2), content.CreatureTypeHumanoid, "s", 30, med, nt("T"), nil))
	distinct(t, "species creature type", base(), sp("a", "A", cite(1), content.CreatureTypeAberration, "s", 30, med, nt("T"), nil))
	distinct(t, "species size text", base(), sp("a", "A", cite(1), content.CreatureTypeHumanoid, "t", 30, med, nt("T"), nil))
	distinct(t, "species speed", base(), sp("a", "A", cite(1), content.CreatureTypeHumanoid, "s", 35, med, nt("T"), nil))
	distinct(t, "species sizes", base(), sp("a", "A", cite(1), content.CreatureTypeHumanoid, "s", 30, small, nt("T"), nil))
	distinct(t, "species traits", base(), sp("a", "A", cite(1), content.CreatureTypeHumanoid, "s", 30, med, nt("U"), nil))
	withChoice := sp("a", "A", cite(1), content.CreatureTypeHumanoid, "s", 30, med, nt("T"),
		[]content.SpeciesChoice{content.MustSpeciesChoice("T", []content.ChoiceOption{text})})
	distinct(t, "species choices", base(), withChoice)

	benefit := []content.FeatBenefit{content.MustFeatBenefit("", "x")}
	ft := func(id content.FeatID, name string, src content.SourceCitation, cat content.FeatCategory, rep bool, repText string) content.Feat {
		return content.MustFeat(id, name, src, content.FeatFields{Category: cat, Benefits: benefit, Repeatable: rep, RepeatableText: repText})
	}
	fb := func() content.Feat { return ft("feat_a", "A", cite(1), content.FeatOrigin, true, "r") }
	distinct(t, "feat id", fb(), ft("feat_b", "A", cite(1), content.FeatOrigin, true, "r"))
	distinct(t, "feat name", fb(), ft("feat_a", "B", cite(1), content.FeatOrigin, true, "r"))
	distinct(t, "feat source", fb(), ft("feat_a", "A", cite(2), content.FeatOrigin, true, "r"))
	distinct(t, "feat category", fb(), ft("feat_a", "A", cite(1), content.FeatGeneral, true, "r"))
	distinct(t, "feat repeatable", fb(), ft("feat_a", "A", cite(1), content.FeatOrigin, false, ""))
	distinct(t, "feat repeatable text", fb(), ft("feat_a", "A", cite(1), content.FeatOrigin, true, "s"))

	opts := content.NewEquipmentOptions(content.MustEquipmentOption(nil, 1), content.MustEquipmentOption(nil, 2))
	bgf := func(feat content.FeatID, tl string, abil []ability.Ability, skills []content.Skill, eq content.EquipmentOptions) content.BackgroundFields {
		return content.BackgroundFields{
			AbilityScores: abil, Feat: content.MustBackgroundFeatReference(feat, ""), SkillProficiencies: skills,
			ToolProficiency: tool(tl), EquipmentOptions: eq,
		}
	}
	abil := []ability.Ability{ability.Strength, ability.Dexterity, ability.Constitution}
	skills := []content.Skill{content.SkillAthletics, content.SkillStealth}
	bg := func(id content.BackgroundID, name string, src content.SourceCitation, f content.BackgroundFields) content.Background {
		return content.MustBackground(id, name, src, f)
	}
	bb := func() content.Background { return bg("a", "A", cite(1), bgf("feat_a", "T", abil, skills, opts)) }
	distinct(t, "background id", bb(), bg("b", "A", cite(1), bgf("feat_a", "T", abil, skills, opts)))
	distinct(t, "background name", bb(), bg("a", "B", cite(1), bgf("feat_a", "T", abil, skills, opts)))
	distinct(t, "background source", bb(), bg("a", "A", cite(2), bgf("feat_a", "T", abil, skills, opts)))
	distinct(t, "background feat", bb(), bg("a", "A", cite(1), bgf("feat_b", "T", abil, skills, opts)))
	distinct(t, "background tool", bb(), bg("a", "A", cite(1), bgf("feat_a", "U", abil, skills, opts)))
	distinct(t, "background abilities", bb(), bg("a", "A", cite(1), bgf("feat_a", "T", []ability.Ability{ability.Strength, ability.Dexterity, ability.Wisdom}, skills, opts)))
	distinct(t, "background skills", bb(), bg("a", "A", cite(1), bgf("feat_a", "T", abil, []content.Skill{content.SkillAthletics, content.SkillArcana}, opts)))
	distinct(t, "background equipment", bb(), bg("a", "A", cite(1), bgf("feat_a", "T", abil, skills,
		content.NewEquipmentOptions(content.MustEquipmentOption(nil, 1), content.MustEquipmentOption(nil, 3)))))

	lvl, _ := content.NewLevelPrerequisite(4)
	abl, _ := content.NewAbilityScorePrerequisite([]ability.Ability{ability.Strength}, 13)
	withPre := func(p content.Prerequisite) content.Feat {
		return content.MustFeat("feat_a", "A", cite(1), content.FeatFields{Category: content.FeatOrigin, Benefits: benefit, Prerequisites: []content.Prerequisite{p}})
	}
	distinct(t, "single prerequisite variant", withPre(lvl), withPre(abl))
}
