package combat

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// AssistedProficiency is the skill or tool proficiency the Help action's ability check
// help is for (SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182). A skill and a
// tool never match each other. Build it with AssistedSkill or AssistedTool. The zero value
// is invalid. Comparable and safe for concurrent use.
type AssistedProficiency struct {
	skill content.Skill
	tool  content.ItemID
}

// AssistedSkill returns the proficiency in the skill. It is invalid when skill is invalid.
func AssistedSkill(skill content.Skill) AssistedProficiency { return AssistedProficiency{skill: skill} }

// AssistedTool returns the proficiency in the tool. It is invalid when tool is invalid.
func AssistedTool(tool content.ItemID) AssistedProficiency { return AssistedProficiency{tool: tool} }

// Valid reports whether p is exactly one valid skill or one valid tool.
func (p AssistedProficiency) Valid() bool {
	return (p.skill.Valid() && p.tool == "") || (p.tool.Valid() && p.skill == 0)
}

// Skill returns the skill and true when p is a skill proficiency.
func (p AssistedProficiency) Skill() (content.Skill, bool) { return p.skill, p.skill != 0 && p.Valid() }

// Tool returns the tool and true when p is a tool proficiency.
func (p AssistedProficiency) Tool() (content.ItemID, bool) { return p.tool, p.tool != "" && p.Valid() }

// HelpGiverProficiencies is the skills and tools the creature that took Help is proficient
// in. A monster has no skill proficiency list, so the caller builds this from either
// source. Each skill and tool is held once. The zero value holds none and is valid. It
// holds slices, so compare with Equal, not ==; accessors return copies. Safe for
// concurrent use.
type HelpGiverProficiencies struct {
	skills []content.Skill
	tools  []content.ItemID
}

// NewHelpGiverProficiencies returns the proficiencies, dropping repeats and keeping the
// first-seen order. It returns an error matching srderr.ErrInvalidInput for an invalid
// skill or tool, and the LimitExceededError of a list over 256 (plan P10). It copies its slices and never panics.
func NewHelpGiverProficiencies(skills []content.Skill, tools []content.ItemID) (HelpGiverProficiencies, error) {
	if err := errors.Join(overCap("help skills", len(skills), maxInputList), overCap("help tools", len(tools), maxInputList)); err != nil {
		return HelpGiverProficiencies{}, err
	}
	var out HelpGiverProficiencies
	for _, s := range skills {
		if !s.Valid() {
			return HelpGiverProficiencies{}, invalidInput("help giver proficiency has an invalid skill")
		}
		if !slices.Contains(out.skills, s) {
			out.skills = append(out.skills, s)
		}
	}
	for _, t := range tools {
		if !t.Valid() {
			return HelpGiverProficiencies{}, invalidInput("help giver proficiency has an invalid tool")
		}
		if !slices.Contains(out.tools, t) {
			out.tools = append(out.tools, t)
		}
	}
	return out, nil
}

// Skills returns a copy of the skills, each once.
func (h HelpGiverProficiencies) Skills() []content.Skill {
	return append([]content.Skill{}, h.skills...)
}

// Tools returns a copy of the tools, each once.
func (h HelpGiverProficiencies) Tools() []content.ItemID {
	return append([]content.ItemID{}, h.tools...)
}

// Has reports whether p is a skill or tool the creature is proficient in. An invalid
// proficiency is never held.
func (h HelpGiverProficiencies) Has(p AssistedProficiency) bool {
	s, isSkill := p.Skill()
	t, isTool := p.Tool()
	return (isSkill && slices.Contains(h.skills, s)) || (isTool && slices.Contains(h.tools, t))
}

// Equal reports whether h and o hold the same skills and tools, whatever order.
func (h HelpGiverProficiencies) Equal(o HelpGiverProficiencies) bool {
	return sameSet(h.skills, o.skills) && sameSet(h.tools, o.tools)
}

// sameSet reports whether a and b hold the same elements, each at most once.
func sameSet[T comparable](a, b []T) bool {
	return len(a) == len(b) && !slices.ContainsFunc(a, func(x T) bool { return !slices.Contains(b, x) })
}
