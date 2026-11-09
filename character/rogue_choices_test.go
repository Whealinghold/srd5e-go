package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-rogue.md FR-021, FR-085, FR-090 (SRD 5.2.1, pp. 61-63) - RequiredFor lists the choices a Rogue level
// gain requires, in the 21 labels of the spec (both level 1 entries and levels 2 to 20), and nothing else; an undefined entry or the
// zero level lists nothing.
// Path: per case (21 labels): happy / unhappy (empty case).
func TestRogueChoicesRequiredFor(t *testing.T) {
	t.Parallel()
	skills, exp, lang, mastery, equip := character.RogueChoiceSkills, character.RogueChoiceExpertise, character.RogueChoiceLanguage, character.RogueChoiceWeaponMastery, character.RogueChoiceStartingEquipment
	sub, feat := character.RogueChoiceSubclass, character.RogueChoiceFeat
	type K = []character.RogueChoiceKind
	for _, c := range []struct {
		entry character.RogueEntry
		level int
		want  K
	}{
		{character.RogueFirstClass, 1, K{skills, exp, lang, mastery, equip}},
		{character.RogueMulticlass, 1, K{skills, exp, lang, mastery}},
		{character.RogueFirstClass, 2, K{}}, {character.RogueFirstClass, 3, K{sub}}, {character.RogueMulticlass, 3, K{sub}},
		{character.RogueFirstClass, 4, K{feat}}, {character.RogueFirstClass, 5, K{}}, {character.RogueFirstClass, 6, K{exp}}, {character.RogueMulticlass, 6, K{exp}},
		{character.RogueFirstClass, 7, K{}}, {character.RogueFirstClass, 8, K{feat}}, {character.RogueFirstClass, 9, K{}}, {character.RogueFirstClass, 10, K{feat}},
		{character.RogueFirstClass, 11, K{}}, {character.RogueFirstClass, 12, K{feat}}, {character.RogueFirstClass, 13, K{}}, {character.RogueFirstClass, 14, K{}},
		{character.RogueFirstClass, 15, K{}}, {character.RogueFirstClass, 16, K{feat}}, {character.RogueFirstClass, 17, K{}}, {character.RogueFirstClass, 18, K{}},
		{character.RogueFirstClass, 19, K{feat}}, {character.RogueMulticlass, 19, K{feat}}, {character.RogueFirstClass, 20, K{}}, {character.RogueMulticlass, 20, K{}},
	} {
		t.Run(c.entry.String()+" "+strconv.Itoa(c.level), func(t *testing.T) {
			t.Parallel()
			if got := (character.RogueChoices{}).RequiredFor(c.entry, rl(c.level)); !slices.Equal(got, c.want) {
				t.Errorf("RequiredFor = %v, want %v", got, c.want)
			}
		})
	}
	if got := (character.RogueChoices{}).RequiredFor(0, rl(1)); len(got) != 0 {
		t.Errorf("undefined entry: %v", got)
	}
	if got := (character.RogueChoices{}).RequiredFor(character.RogueFirstClass, character.RogueLevel{}); len(got) != 0 {
		t.Errorf("zero level: %v", got)
	}
	for k, want := range map[character.RogueChoiceKind]string{skills: "skills", exp: "expertise", lang: "language", mastery: "weapon_mastery", equip: "starting_equipment", sub: "subclass", feat: "feat", 0: "unknown", 8: "unknown"} {
		if k.String() != want {
			t.Errorf("kind %d = %q, want %q", k, k.String(), want)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-018, FR-019 (SRD 5.2.1, "Core Rogue Traits", "As a Multiclass Character", p. 61) - a first
// class entry takes exactly 4 distinct skills from the Rogue list and a multiclass entry exactly 1; every problem is reported once.
// Path: per case (counts 0, 1, 3, 4, 5; a repeat; a skill outside the list): happy / unhappy (invalid input, boundary).
func TestRogueSkillChoice(t *testing.T) {
	t.Parallel()
	p := rogProgression(t)
	first, multi := character.RogueFirstClass, character.RogueMulticlass
	S := func(s ...content.Skill) []content.Skill { return s }
	acro, athl, dece, insi, inti := content.SkillAcrobatics, content.SkillAthletics, content.SkillDeception, content.SkillInsight, content.SkillIntimidation
	count, dup, notIn := character.RogueSkillWrongCount, character.RogueSkillDuplicate, character.RogueSkillNotInRogueList
	for _, c := range []struct {
		name     string
		entry    character.RogueEntry
		chosen   []content.Skill
		problems []character.RogueSkillChoiceProblem
	}{
		{"first class, 4", first, S(acro, athl, dece, insi), nil},
		{"first class, 0", first, nil, []character.RogueSkillChoiceProblem{count}},
		{"first class, 3", first, S(acro, athl, dece), []character.RogueSkillChoiceProblem{count}},
		{"first class, 5", first, S(acro, athl, dece, insi, inti), []character.RogueSkillChoiceProblem{count}},
		{"first class, a repeat", first, S(acro, acro, dece, insi), []character.RogueSkillChoiceProblem{dup}},
		{"first class, Arcana", first, S(acro, athl, dece, skillArcana), []character.RogueSkillChoiceProblem{notIn}},
		{"first class, three problems", first, S(skillArcana, skillArcana), []character.RogueSkillChoiceProblem{count, dup, notIn}},
		{"multiclass, 1", multi, S(skillSleight), nil},
		{"multiclass, 0", multi, nil, []character.RogueSkillChoiceProblem{count}},
		{"multiclass, 2", multi, S(acro, athl), []character.RogueSkillChoiceProblem{count}},
		{"multiclass, Arcana", multi, S(skillArcana), []character.RogueSkillChoiceProblem{notIn}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := p.SkillChoice(c.entry, c.chosen)
			if c.problems != nil {
				if probs := rogProblems[character.RogueSkillChoiceProblem](t, err); !slices.Equal(probs, c.problems) {
					t.Errorf("problems = %v, want %v", probs, c.problems)
				}
				return
			}
			if err != nil || !slices.Equal(got.Skills(), c.chosen) {
				t.Errorf("choice = %v, %v; want %v", got.Skills(), err, c.chosen)
			}
		})
	}
	_, err := p.SkillChoice(0, S(acro))
	wantInvalid(t, err, "undefined entry")
	got := must(p.SkillChoice(first, S(acro, athl, dece, insi)))
	got.Skills()[0] = skillArcana
	if got.Skills()[0] != acro {
		t.Error("Skills shares storage")
	}
}

// Requirement: doc/specs/class-rogue.md FR-020 (SRD 5.2.1, "Core Rogue Traits", p. 61) - a first class Rogue records exactly one
// Starting Equipment option, A or B; a missing choice is a failure and is never defaulted.
// Path: per case (A, B, none, undefined): happy / unhappy (invalid input).
func TestRogueStartingEquipmentChoice(t *testing.T) {
	t.Parallel()
	for _, o := range []character.RogueStartingEquipmentOption{character.RogueEquipmentOptionA, character.RogueEquipmentOptionB} {
		if c, err := character.NewRogueStartingEquipmentChoice(o); err != nil || c.Option() != o {
			t.Errorf("option %d = %v, %v", o, c, err)
		}
	}
	for _, o := range []character.RogueStartingEquipmentOption{0, 3, -1} {
		c, err := character.NewRogueStartingEquipmentChoice(o)
		wantInvalid(t, err, "option "+strconv.Itoa(int(o)))
		if c.Option() != 0 {
			t.Errorf("option %d produced %v", o, c)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-031, FR-032 (SRD 5.2.1, "Level 1: Thieves' Cant", p. 62) - the one other language is not
// Thieves' Cant itself and is in the permitted set; a language the character already knows is accepted.
// Path: per case (Thieves' Cant, an id outside the set, a valid language): happy / unhappy (invalid input).
func TestRogueLanguageChoice(t *testing.T) {
	t.Parallel()
	permitted := []content.Language{content.LanguageCommon, content.LanguageElvish, content.LanguageDeepSpeech}
	for _, c := range []struct {
		name     string
		chosen   content.Language
		permit   []content.Language
		problems []character.RogueLanguageChoiceProblem
	}{
		{"a permitted language", content.LanguageElvish, permitted, nil},
		{"a rare language", content.LanguageDeepSpeech, permitted, nil},
		{"Thieves' Cant itself", content.LanguageThievesCant, append(slices.Clone(permitted), content.LanguageThievesCant), []character.RogueLanguageChoiceProblem{character.RogueLanguageIsThievesCant}},
		{"outside the permitted set", content.LanguageDraconic, permitted, []character.RogueLanguageChoiceProblem{character.RogueLanguageNotPermitted}},
		{"Thieves' Cant outside the set", content.LanguageThievesCant, permitted, []character.RogueLanguageChoiceProblem{character.RogueLanguageIsThievesCant, character.RogueLanguageNotPermitted}},
		{"the zero language", 0, permitted, []character.RogueLanguageChoiceProblem{character.RogueLanguageNotPermitted}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewRogueLanguageChoice(c.chosen, c.permit)
			if c.problems != nil {
				if probs := rogProblems[character.RogueLanguageChoiceProblem](t, err); !slices.Equal(probs, c.problems) {
					t.Errorf("problems = %v, want %v", probs, c.problems)
				}
				return
			}
			if err != nil || got.Language() != c.chosen {
				t.Errorf("choice = %v, %v", got.Language(), err)
			}
		})
	}
}
