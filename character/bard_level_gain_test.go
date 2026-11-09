package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func brdDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src := must(data.NewBundledSource())
	res := must(src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("bard"))))
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

// Requirement: doc/specs/class-bard.md FR-069, FR-097, FR-115 (SRD 5.2.1, "Bard Features", p. 31; "College of Lore", p. 35) - the features a level gains are the class features of exactly that level, then the held subclass's features of that level: Bonus Proficiencies and Cutting Words at 3, Magical Discoveries at 6, Peerless Skill at 14; none is owed from a lower level.
// Path: per case (the levels with subclass features and one without): happy / unhappy (boundary).
func TestBardFeaturesGainedAt(t *testing.T) {
	t.Parallel()
	def := brdDef(t)
	type ids = []content.FeatureID
	for _, c := range []struct {
		level int
		held  content.SubclassID
		want  ids
	}{
		{1, "", ids{"bard_bardic_inspiration", "bard_spellcasting"}},
		{2, lore, ids{"bard_expertise", "bard_jack_of_all_trades"}},
		{3, "", ids{"bard_bard_subclass"}},
		{3, lore, ids{"bard_bard_subclass", "college_of_lore_bonus_proficiencies", "college_of_lore_cutting_words"}},
		{4, lore, ids{"bard_ability_score_improvement"}},
		{6, lore, ids{"college_of_lore_magical_discoveries"}},
		{6, "", ids{}},
		{14, lore, ids{"college_of_lore_peerless_skill"}},
		{20, lore, ids{"bard_words_of_creation"}},
	} {
		got, err := character.BardFeaturesGainedAt(def, c.held, character.MustBardLevel(c.level))
		if err != nil || got == nil || !slices.Equal(got, c.want) {
			t.Errorf("level %d, held %q: %v, %v; want %v", c.level, c.held, got, err, c.want)
		}
	}
	for name, call := range map[string]func() ([]content.FeatureID, error){
		"unknown subclass": func() ([]content.FeatureID, error) {
			return character.BardFeaturesGainedAt(def, "college_of_nothing", character.MustBardLevel(3))
		},
		"zero level": func() ([]content.FeatureID, error) {
			return character.BardFeaturesGainedAt(def, "", character.BardLevel{})
		},
		"zero definition": func() ([]content.FeatureID, error) {
			return character.BardFeaturesGainedAt(content.ClassDefinition{}, "", character.MustBardLevel(3))
		},
		"another class": func() ([]content.FeatureID, error) {
			return character.BardFeaturesGainedAt(clcDef(t), "", character.MustBardLevel(3))
		},
	} {
		if got, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
			t.Errorf("%s: %v, %v; want nil and ErrInvalidInput", name, got, err)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-066, FR-067, FR-068 (SRD 5.2.1, "Level 3: Bard Subclass", p. 32) - a chosen subclass must be one the Bard definition lists (an unknown id is a problem, not a missing-content outcome), must not be empty, is refused below level 3 and when one is held; every problem is listed together.
// Path: per case (unhappy: missing content, boundary, state error).
func TestBardChosenSubclassProblems(t *testing.T) {
	t.Parallel()
	def := brdDef(t)
	for _, c := range []struct {
		level        int
		held, chosen content.SubclassID
		want         []bp
	}{
		{3, "", lore, []bp{}},
		{3, "", "college_of_nothing", []bp{character.BardSubclassUnknown}},
		{3, "", "", []bp{character.BardSubclassMissing}},
		{2, "", lore, []bp{character.BardSubclassTooEarly}},
		{3, lore, lore, []bp{character.BardSubclassAlreadyChosen}},
		{1, lore, "x", []bp{character.BardSubclassTooEarly, character.BardSubclassAlreadyChosen, character.BardSubclassUnknown}},
	} {
		if got := character.BardChosenSubclassProblems(def, character.MustBardLevel(c.level), c.held, c.chosen); !slices.Equal(got, c.want) || got == nil {
			t.Errorf("level %d, held %q, chosen %q: %v, want %v", c.level, c.held, c.chosen, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-079 (SRD 5.2.1, "Level 6: Magical Discoveries", p. 35) - one Magical Discoveries spell may be replaced per level gained by a spell not held, on the Cleric, Druid or Wizard list and within the slots; a second replacement, a spell not held, a held spell as the new one, a Bard-only spell and a level 4 spell at Bard level 6 are each refused and listed together.
// Path: per case (unhappy: boundary, state error, invalid input).
func TestMagicalDiscoveriesReplacementProblems(t *testing.T) {
	t.Parallel()
	level6 := character.MustBardLevel(6)
	held := []content.SpellID{"bless", "shield"}
	bless, cure := brdSpell(t, "bless", 1, "cleric"), brdSpell(t, "cure_wounds", 1, "cleric", "bard")
	druid, bardOnly, high := brdSpell(t, "thorn_whip", 0, "druid"), brdSpell(t, "vicious_mockery", 0, "bard"), brdSpell(t, "polymorph", 4, "druid", "wizard")
	rep := func(out content.SpellID, in content.Spell) character.MagicalDiscoveriesReplacement {
		return character.MagicalDiscoveriesReplacement{Out: out, In: in}
	}
	type rs = []character.MagicalDiscoveriesReplacement
	for _, c := range []struct {
		name string
		in   rs
		want []bp
	}{
		{"none", nil, []bp{}},
		{"one valid", rs{rep("bless", cure)}, []bp{}},
		{"a cantrip", rs{rep("shield", druid)}, []bp{}},
		{"a spell of the highest level with slots", rs{rep("shield", brdSpell(t, "revivify", 3, "cleric"))}, []bp{}},
		{"two", rs{rep("bless", cure), rep("shield", druid)}, []bp{character.BardReplacementTooMany}},
		{"not held", rs{rep("fireball", cure)}, []bp{character.BardReplacedNotHeld}},
		{"new one held", rs{rep("shield", bless)}, []bp{character.BardSpellDuplicate}},
		{"bard only", rs{rep("bless", bardOnly)}, []bp{character.BardSpellNotOnList}},
		{"no slots", rs{rep("bless", high)}, []bp{character.BardSpellWithoutSlots}},
		{"everything", rs{rep("x", bardOnly), rep("bless", high)}, []bp{character.BardReplacementTooMany, character.BardReplacedNotHeld, character.BardSpellNotOnList, character.BardSpellWithoutSlots}},
	} {
		if got := character.MagicalDiscoveriesReplacementProblems(level6, held, c.in); !slices.Equal(got, c.want) || got == nil {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
	if got := character.MagicalDiscoveriesReplacementProblems(character.MustBardLevel(5), held, rs{rep("bless", high)}); !slices.Contains(got, character.BardSpellWithoutSlots) {
		t.Errorf("level 5 with a level 4 spell: %v", got)
	}
}

// Requirement: doc/specs/class-bard.md FR-098 (SRD 5.2.1, "Bard Features", p. 31) - a level gain with incomplete or invalid choices lists every problem found, each once, not just the first: level 2 without Expertise, level 3 without a subclass or skills, level 4 without a feat, level 6 without Magical Discoveries; a complete choice has none; a choice the level does not owe is ignored.
// Path: per case (unhappy: invalid input; happy).
func TestBardLevelChoiceProblems(t *testing.T) {
	t.Parallel()
	def := brdDef(t)
	profs := must(character.NewSkillProficiencies([]content.Skill{content.SkillAthletics, content.SkillAcrobatics, content.SkillArcana}, nil))
	two := []content.Skill{content.SkillAthletics, content.SkillAcrobatics}
	three := []content.Skill{content.SkillHistory, content.SkillStealth, content.SkillArcana}
	valid3 := []content.Skill{content.SkillHistory, content.SkillStealth, content.SkillInsight}
	spells := []content.Spell{brdSpell(t, "bless", 1, "cleric"), brdSpell(t, "thorn_whip", 0, "druid")}
	for _, c := range []struct {
		name    string
		level   int
		held    content.SubclassID
		choices character.BardLevelChoices
		want    []bp
	}{
		{"level 1 owes nothing", 1, "", character.BardLevelChoices{}, []bp{}},
		{"level 2 complete", 2, "", character.BardLevelChoices{Expertise: two}, []bp{}},
		{"level 2 without Expertise", 2, "", character.BardLevelChoices{}, []bp{character.BardExpertiseCountWrong}},
		{"level 3 without anything", 3, "", character.BardLevelChoices{}, []bp{character.BardSubclassMissing}},
		{"level 3 complete", 3, "", character.BardLevelChoices{Subclass: lore, BonusSkills: valid3}, []bp{}},
		{"level 3 with a short skill list", 3, "", character.BardLevelChoices{Subclass: lore, BonusSkills: three[:2]}, []bp{character.BardSkillCountWrong}},
		{"level 3 already proficient", 3, "", character.BardLevelChoices{Subclass: lore, BonusSkills: three}, []bp{character.BardSkillAlreadyProficient}},
		{"level 3 unknown subclass", 3, "", character.BardLevelChoices{Subclass: "college_of_nothing"}, []bp{character.BardSubclassUnknown}},
		{"level 3 subclass held", 3, lore, character.BardLevelChoices{BonusSkills: valid3}, []bp{character.BardSubclassAlreadyChosen, character.BardSubclassMissing}},
		{"level 4 without a feat", 4, lore, character.BardLevelChoices{}, []bp{character.BardFeatMissing}},
		{"level 4 with a feat", 4, lore, character.BardLevelChoices{Feat: "ability_score_improvement"}, []bp{}},
		{"level 19 without a boon", 19, lore, character.BardLevelChoices{}, []bp{character.BardFeatMissing}},
		{"level 6 lore without spells", 6, lore, character.BardLevelChoices{}, []bp{character.BardSpellCountWrong}},
		{"level 6 lore with spells", 6, lore, character.BardLevelChoices{Discoveries: spells}, []bp{}},
		{"level 6 without a subclass", 6, "", character.BardLevelChoices{}, []bp{}},
		{"level 5 ignores extras", 5, lore, character.BardLevelChoices{Feat: "x", Expertise: []content.Skill{0}}, []bp{}},
		{"level 9 repeats listed once", 9, lore, character.BardLevelChoices{Expertise: []content.Skill{content.SkillHistory, content.SkillHistory}}, []bp{character.BardExpertiseNotProficient, character.BardExpertiseDuplicate}},
	} {
		got := character.BardLevelChoiceProblems(def, character.MustBardLevel(c.level), c.held, profs, c.choices)
		if !slices.Equal(got, c.want) || got == nil {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
