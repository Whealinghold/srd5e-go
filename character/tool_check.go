package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ToolCheckInput is what [EvaluateToolCheck] reads. Construct it with a keyed
// literal. A zero SkillUsed means no skill takes part; a zero Ability means the
// tool's own Ability.
type ToolCheckInput struct {
	Tool      ResolvedTool
	Tools     ToolProficiencies
	Skills    SkillProficiencies
	SkillUsed content.Skill
	Ability   ability.Ability
}

// ToolCheckBenefits is what proficiency adds to one ability check made with a
// tool: a multiple of the Proficiency Bonus (0, 1 or 2), Advantage when also
// proficient in the skill used, and the Ability to roll. Built by
// [EvaluateToolCheck]; the zero value is a check with no benefit and no Ability.
// Comparable and safe for concurrent use.
type ToolCheckBenefits struct {
	toolID         content.ItemID
	toolProficient bool
	multiple       int
	advantage      bool
	ability        ability.Ability
}

// EvaluateToolCheck returns the benefits of an ability check with a tool (SRD
// 5.2.1, "Tool Proficiency", p. 93; "Equipment Proficiencies", p. 9; "The Bonus
// Doesn't Stack", p. 8). Tool proficiency gives the Proficiency Bonus once;
// proficiency in the skill used gives it once too, so both together still give
// one; Expertise in the skill used gives twice (a tool alone never doubles, and
// Expertise in another skill is ignored). Advantage needs proficiency in both
// the tool and the skill used. A non-zero Ability that is not valid is treated
// as none given. It is pure, returns no error, and never alters its input.
func EvaluateToolCheck(in ToolCheckInput) ToolCheckBenefits {
	toolProf := in.Tools.Contains(in.Tool.ProficiencyID())
	skillProf := in.Skills.Proficient(in.SkillUsed) || in.Skills.HasExpertise(in.SkillUsed)
	multiple := 0
	if toolProf || skillProf {
		multiple = 1
	}
	if in.Skills.HasExpertise(in.SkillUsed) {
		multiple = 2
	}
	ab := in.Tool.Ability()
	if in.Ability.Valid() {
		ab = in.Ability
	}
	return ToolCheckBenefits{toolID: in.Tool.ProficiencyID(), toolProficient: toolProf, multiple: multiple, advantage: toolProf && skillProf, ability: ab}
}

// ProficiencyID returns the id of the tool the benefits were evaluated for.
func (b ToolCheckBenefits) ProficiencyID() content.ItemID { return b.toolID }

// ToolProficient reports whether the creature is proficient with the tool.
func (b ToolCheckBenefits) ToolProficient() bool { return b.toolProficient }

// ProficiencyBonusMultiple returns how many times the Proficiency Bonus is added: 0, 1 or 2.
func (b ToolCheckBenefits) ProficiencyBonusMultiple() int { return b.multiple }

// UsesProficiency reports whether the check uses a proficiency, the fact other
// rules read for "a check that uses a skill or tool proficiency" (SRD 5.2.1,
// "Proficiency", p. 186).
func (b ToolCheckBenefits) UsesProficiency() bool { return b.multiple > 0 }

// Ability returns the Ability to roll: the one given, else the tool's own.
func (b ToolCheckBenefits) Ability() ability.Ability { return b.ability }

// Circumstances returns the roll circumstances: one Advantage when the creature
// is proficient in the tool and the skill used, else none. It is a new slice. No
// D20 Test is resolved here; pass it to [modifier.Resolve] with any Disadvantage.
func (b ToolCheckBenefits) Circumstances() []modifier.RollCircumstance {
	if b.advantage {
		return []modifier.RollCircumstance{modifier.Advantage{}}
	}
	return []modifier.RollCircumstance{}
}

// RollModifiers returns the modifiers of the check: the ability modifier m, the
// Proficiency Bonus multiple times bonus (absent at multiple 0) in its single
// slot, and others unchanged.
func (b ToolCheckBenefits) RollModifiers(m ability.Modifier, bonus ProficiencyBonus, others ...modifier.Modifier) d20.RollModifiers {
	r := d20.NewRollModifiers(modifier.New(m.Value()))
	if b.multiple > 0 {
		r = r.WithProficiencyBonus(modifier.New(b.multiple * bonus.Value()))
	}
	return r.WithOthers(others...)
}
