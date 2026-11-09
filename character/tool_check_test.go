package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: per case tool-proficiency-and-crafting.md FR-007 to FR-013,
// FR-016 (SRD 5.2.1, "Tool Proficiency", p. 93; "Equipment Proficiencies", p. 9;
// "Expertise", p. 182; "The Bonus Doesn't Stack", p. 8) - the "Cases: check
// benefits" table with Herbalism Kit: multiple, Advantage, tool benefit.
// Path: per case (happy; unhappy boundary, rule interaction, empty and minimal).
func TestEvaluateToolCheck(t *testing.T) {
	t.Parallel()
	nature, arcana := content.SkillNature, content.SkillArcana
	cases := []struct {
		name, requirement, path string
		tool                    bool
		prof, exp               []content.Skill
		used                    content.Skill
		wantMultiple            int
		wantAdvantage           bool
	}{
		{"tool proficient, no skill", "FR-007", "happy", true, nil, nil, 0, 1, false},
		{"no proficiency, no skill", "FR-008", "unhappy (boundary)", false, nil, nil, 0, 0, false},
		{"no proficiency, skill not proficient", "FR-008", "unhappy (boundary)", false, nil, nil, nature, 0, false},
		{"tool and skill proficient give one bonus and Advantage", "FR-009", "unhappy (rule interaction)", true, []content.Skill{nature}, nil, nature, 1, true},
		{"tool and Expertise give two and Advantage", "FR-009, FR-011", "unhappy (rule interaction)", true, []content.Skill{nature}, []content.Skill{nature}, nature, 2, true},
		{"skill only, proficient", "FR-010", "unhappy (rule interaction)", false, []content.Skill{nature}, nil, nature, 1, false},
		{"skill only, Expertise", "FR-010, FR-011", "unhappy (rule interaction)", false, []content.Skill{nature}, []content.Skill{nature}, nature, 2, false},
		{"Expertise in another skill is ignored", "FR-011", "unhappy (rule interaction)", true, []content.Skill{arcana}, []content.Skill{arcana}, nature, 1, false},
		{"skill proficiency held but no skill used", "FR-012", "unhappy (empty and minimal)", true, []content.Skill{nature}, []content.Skill{nature}, 0, 1, false},
		{"skill proficiency held, no skill used, no tool", "FR-012", "unhappy (empty and minimal)", false, []content.Skill{nature}, []content.Skill{nature}, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tools := character.ToolProficiencies{}
			if c.tool {
				tools = character.NewToolProficiencies("herbalism_kit")
			}
			b := character.EvaluateToolCheck(character.ToolCheckInput{Tool: resolvedTool(herbalismKit()), Tools: tools, Skills: skillsOf(c.prof, c.exp), SkillUsed: c.used})
			circ := b.Circumstances()
			if b.ProficiencyBonusMultiple() != c.wantMultiple || b.UsesProficiency() != (c.wantMultiple > 0) || b.ToolProficient() != c.tool ||
				(len(circ) == 1) != c.wantAdvantage || (len(circ) == 1 && circ[0] != modifier.Advantage{}) || len(circ) > 1 {
				t.Errorf("benefits = multiple %d, uses %v, tool %v, circumstances %v [%s]", b.ProficiencyBonusMultiple(), b.UsesProficiency(), b.ToolProficient(), circ, c.requirement)
			}
			if b.Ability() != ability.Intelligence || b.ProficiencyID() != "herbalism_kit" {
				t.Errorf("ability %v, tool %q", b.Ability(), b.ProficiencyID())
			}
		})
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-002, FR-003 (SRD
// 5.2.1, "Tools", p. 93) - proficiency is per variant, a family id gives none for
// a variant, and a monster is proficient with its stat block's tool.
// Path: per case (happy; unhappy rule interaction).
func TestEvaluateToolCheckSeparateVariants(t *testing.T) {
	t.Parallel()
	catalogue := []content.Tool{gamingSet()}
	cards := resolvedTool(variantTool("gaming_set_playing_cards", "gaming_set"), catalogue...)
	dice := resolvedTool(variantTool("gaming_set_dice", "gaming_set"), catalogue...)
	cases := []struct {
		name  string
		tool  character.ResolvedTool
		tools character.ToolProficiencies
		want  int
	}{
		{"dice proficiency, cards check", cards, character.NewToolProficiencies("gaming_set_dice"), 0},
		{"family proficiency, dice check", dice, character.NewToolProficiencies("gaming_set"), 0},
		{"dice proficiency, dice check", dice, character.NewToolProficiencies("gaming_set_dice"), 1},
		{"monster with Thieves' Tools", resolvedTool(thievesTools()), character.MonsterToolProficiencies("thieves_tools"), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := character.EvaluateToolCheck(character.ToolCheckInput{Tool: c.tool, Tools: c.tools}).ProficiencyBonusMultiple(); got != c.want {
				t.Errorf("multiple = %d, want %d", got, c.want)
			}
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-013 (SRD 5.2.1, "Tools",
// Ability, p. 93) - a given Ability replaces the tool's own without changing
// proficiency or Advantage; a non-zero invalid Ability counts as none given.
// Path: unhappy (rule interaction).
func TestEvaluateToolCheckAbilityOverride(t *testing.T) {
	t.Parallel()
	in := character.ToolCheckInput{Tool: resolvedTool(thievesTools()), Tools: character.NewToolProficiencies("thieves_tools"),
		Skills: skillsOf([]content.Skill{content.SkillSleightOfHand}, nil), SkillUsed: content.SkillSleightOfHand}
	base := character.EvaluateToolCheck(in)
	in.Ability = ability.Strength
	over := character.EvaluateToolCheck(in)
	if over.Ability() != ability.Strength || base.Ability() != ability.Dexterity {
		t.Errorf("abilities = %v then %v, want Dexterity then Strength", base.Ability(), over.Ability())
	}
	if over.ProficiencyBonusMultiple() != base.ProficiencyBonusMultiple() || len(over.Circumstances()) != len(base.Circumstances()) {
		t.Error("the Ability changed proficiency or Advantage")
	}
	in.Ability = ability.Ability(99)
	if got := character.EvaluateToolCheck(in).Ability(); got != ability.Dexterity {
		t.Errorf("an invalid Ability gave %v, want the tool's Dexterity", got)
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-014 (SRD 5.2.1, "Add
// Modifiers", p. 6; "The Bonus Doesn't Stack", p. 8) - the Proficiency Bonus slot
// is absent at multiple 0, once the bonus at 1, twice at 2; others pass through.
// Path: per case (happy; unhappy boundary).
func TestToolCheckRollModifiers(t *testing.T) {
	t.Parallel()
	pb := character.ProficiencyBonusFor(character.MustLevel(1))
	nature := content.SkillNature
	cases := []struct {
		name      string
		tools     character.ToolProficiencies
		skills    character.SkillProficiencies
		wantBonus int
		wantSlot  bool
	}{
		{"multiple 0", character.ToolProficiencies{}, skillsOf(nil, nil), 0, false},
		{"multiple 1", character.NewToolProficiencies("herbalism_kit"), skillsOf(nil, nil), 2, true},
		{"multiple 2", character.NewToolProficiencies("herbalism_kit"), skillsOf([]content.Skill{nature}, []content.Skill{nature}), 4, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := character.EvaluateToolCheck(character.ToolCheckInput{Tool: resolvedTool(herbalismKit()), Tools: c.tools, Skills: c.skills, SkillUsed: nature})
			r := b.RollModifiers(modifierOf(t, 3), pb, modifier.New(1))
			slot, has := r.ProficiencyBonus()
			if r.Ability().Value() != 3 || has != c.wantSlot || slot.Value() != c.wantBonus || !slices.Equal(r.Others(), []modifier.Modifier{modifier.New(1)}) {
				t.Errorf("modifiers = ability %d, slot %v (%v), others %v", r.Ability().Value(), has, slot, r.Others())
			}
			if want := d20.NewRollModifiers(modifier.New(3)); c.wantSlot == false && !r.Equal(want.WithOthers(modifier.New(1))) {
				t.Error("multiple 0 must leave only the ability and the others")
			}
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-015 (SRD 5.2.1, "They Don't
// Stack", p. 8) - Advantage alone resolves to advantage and with a Disadvantage to
// normal; the list is a copy and holds at most one Advantage.
// Path: unhappy (rule interaction).
func TestToolCheckCircumstancesResolve(t *testing.T) {
	t.Parallel()
	nature := content.SkillNature
	b := character.EvaluateToolCheck(character.ToolCheckInput{Tool: resolvedTool(herbalismKit()), Tools: character.NewToolProficiencies("herbalism_kit"),
		Skills: skillsOf([]content.Skill{nature}, nil), SkillUsed: nature})
	if got := modifier.Resolve(b.Circumstances()...); got != modifier.WithAdvantage {
		t.Errorf("Resolve(Advantage) = %v", got)
	}
	if got := modifier.Resolve(append(b.Circumstances(), modifier.Disadvantage{})...); got != modifier.Normal {
		t.Errorf("Resolve(Advantage, Disadvantage) = %v, want normal", got)
	}
	c := b.Circumstances()
	c[0] = modifier.Disadvantage{}
	if again := b.Circumstances(); len(again) != 1 || again[0] != (modifier.Advantage{}) {
		t.Errorf("circumstances after mutating a returned slice = %v", again)
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-017 - the evaluation is
// deterministic, leaves its inputs alone and gives equal comparable values.
// Path: unhappy (immutability).
func TestEvaluateToolCheckIsDeterministic(t *testing.T) {
	t.Parallel()
	tools := character.NewToolProficiencies("herbalism_kit", "thieves_tools")
	skills := skillsOf([]content.Skill{content.SkillNature}, nil)
	in := character.ToolCheckInput{Tool: resolvedTool(herbalismKit()), Tools: tools, Skills: skills, SkillUsed: content.SkillNature}
	a, b := character.EvaluateToolCheck(in), character.EvaluateToolCheck(in)
	if a != b {
		t.Errorf("two evaluations differ: %#v and %#v", a, b)
	}
	if !tools.Equal(character.NewToolProficiencies("herbalism_kit", "thieves_tools")) || !skills.Proficient(content.SkillNature) {
		t.Error("the evaluation changed its inputs")
	}
}
