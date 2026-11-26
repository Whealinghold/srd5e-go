package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-036, FR-065 - a skill proficiency and a tool proficiency are different by type: a skill never equals a tool, even of the same text, and the zero proficiency, an invalid skill and an invalid tool are invalid.
// Path: unhappy (invalid input).
func TestAssistedProficiency(t *testing.T) {
	t.Parallel()
	if !athletics().Valid() || !thievesTools().Valid() || athletics() == stealth() || athletics() == thievesTools() {
		t.Error("valid proficiencies must differ by value")
	}
	if s, ok := athletics().Skill(); !ok || s != content.SkillAthletics {
		t.Errorf("Skill() = %v, %v", s, ok)
	}
	if _, ok := athletics().Tool(); ok {
		t.Error("a skill proficiency reports a tool")
	}
	if tool, ok := thievesTools().Tool(); !ok || tool != content.MustItemID("thieves_tools") {
		t.Errorf("Tool() = %v, %v", tool, ok)
	}
	if _, ok := thievesTools().Skill(); ok {
		t.Error("a tool proficiency reports a skill")
	}
	for name, p := range map[string]combat.AssistedProficiency{
		"zero": {}, "bad skill": combat.AssistedSkill(0), "undefined skill": combat.AssistedSkill(99), "bad tool": combat.AssistedTool("Not Valid"),
	} {
		if p.Valid() {
			t.Errorf("%s proficiency is valid", name)
		}
		if _, ok := p.Skill(); ok {
			t.Errorf("%s reports a skill", name)
		}
		if _, ok := p.Tool(); ok {
			t.Errorf("%s reports a tool", name)
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-065 - HelpGiverProficiencies holds each skill and tool once, copies what it is given and what it returns, compares as a set, and its zero value is the valid empty set.
// Path: unhappy (immutability).
func TestHelpGiverProficiencies(t *testing.T) {
	t.Parallel()
	skills := []content.Skill{content.SkillAthletics, content.SkillStealth, content.SkillAthletics}
	tool := content.MustItemID("thieves_tools")
	p := proficienciesOf(t, skills, []content.ItemID{tool, tool})
	if !slices.Equal(p.Skills(), []content.Skill{content.SkillAthletics, content.SkillStealth}) || !slices.Equal(p.Tools(), []content.ItemID{tool}) {
		t.Errorf("duplicates did not collapse: %v, %v", p.Skills(), p.Tools())
	}
	skills[0] = content.SkillArcana
	got := p.Skills()
	got[0] = content.SkillArcana
	if !p.Has(athletics()) || p.Has(combat.AssistedSkill(content.SkillArcana)) {
		t.Error("the constructor or an accessor shares its slice")
	}
	tools := p.Tools()
	tools[0] = "changed"
	if !p.Has(thievesTools()) {
		t.Error("Tools() is not a copy")
	}
	if p.Has(combat.AssistedTool(content.MustItemID("athletics"))) || p.Has(combat.AssistedSkill(content.SkillSurvival)) || p.Has(combat.AssistedProficiency{}) {
		t.Error("Has reported a proficiency the creature lacks")
	}
	var zero combat.HelpGiverProficiencies
	if zero.Has(athletics()) || len(zero.Skills())+len(zero.Tools()) != 0 || !zero.Equal(proficienciesOf(t, nil, nil)) {
		t.Error("the zero value must be the empty set")
	}
	reordered := proficienciesOf(t, []content.Skill{content.SkillStealth, content.SkillAthletics}, []content.ItemID{tool})
	smaller := proficienciesOf(t, []content.Skill{content.SkillStealth}, []content.ItemID{tool})
	if smaller.Equal(p) || p.Equal(smaller) {
		t.Error("a subset must not equal the larger set, in either order")
	}
	if !p.Equal(reordered) || p.Equal(zero) || p.Equal(proficienciesOf(t, []content.Skill{content.SkillStealth, content.SkillArcana}, []content.ItemID{tool})) ||
		p.Equal(proficienciesOf(t, []content.Skill{content.SkillStealth, content.SkillAthletics}, nil)) {
		t.Error("Equal must compare skills and tools as sets")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-065 - an invalid skill or tool is a validation error.
// Path: unhappy (invalid input).
func TestHelpGiverProficienciesRejectInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewHelpGiverProficiencies([]content.Skill{0}, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid skill: %v", err)
	}
	if _, err := combat.NewHelpGiverProficiencies(nil, []content.ItemID{""}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid tool: %v", err)
	}
}
