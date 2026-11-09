package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// ResolvedTool is the facts of one tool that can be proficient: its proficiency
// id, its Ability, its Utilize actions and its group (SRD 5.2.1, "Tools",
// pp. 93-94). A variant takes the facts from its family but keeps its own id.
// Built by [ResolveTool]; the zero value is invalid ([ResolvedTool.IsZero]). It
// holds a slice, so it is not comparable with ==. Safe for concurrent use.
type ResolvedTool struct {
	id      content.ItemID
	ability ability.Ability
	utilize []content.ToolUtilizeAction
	group   content.ToolGroup
}

// IsZero reports whether t is the invalid zero value.
func (t ResolvedTool) IsZero() bool { return t.id == "" }

// ProficiencyID returns the id proficiency is recorded under: a variant's own id.
func (t ResolvedTool) ProficiencyID() content.ItemID { return t.id }

// Ability returns the Ability of the tool (SRD 5.2.1, "Tools", p. 93).
func (t ResolvedTool) Ability() ability.Ability { return t.ability }

// Utilize returns the Utilize actions in printed order, as a new slice.
func (t ResolvedTool) Utilize() []content.ToolUtilizeAction { return slices.Clone(t.utilize) }

// Group returns whether the tool is Artisan's Tools or an Other Tool.
func (t ResolvedTool) Group() content.ToolGroup { return t.group }

// ToolResolution is the closed set of outcomes of [ResolveTool]. Variants:
// ToolResolved, ToolFamilyNeedsVariant, ToolFamilyMissing.
type ToolResolution interface{ isToolResolution() }

// ToolResolved holds the facts of a tool that can be proficient.
type ToolResolved struct{ tool ResolvedTool }

// Tool returns the resolved tool.
func (r ToolResolved) Tool() ResolvedTool { return r.tool }

// ToolFamilyNeedsVariant says the tool is a family (Gaming Set, Musical
// Instrument), which is not a proficiency: a variant must be named.
type ToolFamilyNeedsVariant struct{ family content.ItemID }

// FamilyID returns the family id.
func (r ToolFamilyNeedsVariant) FamilyID() content.ItemID { return r.family }

// ToolFamilyMissing says a variant's family is not in the catalogue exactly once
// as a family, so its facts are unknown. Both ids are empty when the tool was nil.
type ToolFamilyMissing struct{ variant, family content.ItemID }

// VariantID returns the variant's id.
func (r ToolFamilyMissing) VariantID() content.ItemID { return r.variant }

// FamilyID returns the family id the variant names.
func (r ToolFamilyMissing) FamilyID() content.ItemID { return r.family }

func (ToolResolved) isToolResolution()           {}
func (ToolFamilyNeedsVariant) isToolResolution() {}
func (ToolFamilyMissing) isToolResolution()      {}

// ResolveTool returns the facts of tool for proficiency and Utilize. A
// standalone tool's are its own. A variant's Ability, Utilize actions and group
// are its family's, found by id in catalogue, while its proficiency id stays its
// own (SRD 5.2.1, "Tools", pp. 93-94). A family on its own returns
// [ToolFamilyNeedsVariant]. A variant whose family is absent, is not a family,
// or is held twice in catalogue returns [ToolFamilyMissing], whatever the
// catalogue order; so does a nil tool. It rolls nothing, returns no error and
// never panics.
func ResolveTool(tool content.Tool, catalogue []content.Tool) ToolResolution {
	switch t := tool.(type) {
	case content.StandaloneTool:
		return ToolResolved{tool: ResolvedTool{id: t.ID(), ability: t.Ability(), utilize: t.Utilize(), group: t.Group()}}
	case content.ToolFamily:
		return ToolFamilyNeedsVariant{family: t.ID()}
	case content.ToolVariant:
		var named []content.Tool
		for _, c := range catalogue {
			if c != nil && c.ID() == t.VariantOf() {
				named = append(named, c)
			}
		}
		if len(named) != 1 {
			return ToolFamilyMissing{variant: t.ID(), family: t.VariantOf()}
		}
		f, isFamily := named[0].(content.ToolFamily)
		if !isFamily {
			return ToolFamilyMissing{variant: t.ID(), family: t.VariantOf()}
		}
		return ToolResolved{tool: ResolvedTool{id: t.ID(), ability: f.Ability(), utilize: f.Utilize(), group: f.Group()}}
	}
	return ToolFamilyMissing{}
}
