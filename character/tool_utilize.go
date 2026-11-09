package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// UtilizeActions returns the Utilize actions the tool lists, in printed order
// (a variant's are its family's), as a new slice (SRD 5.2.1, "Tools", p. 93).
func UtilizeActions(tool ResolvedTool) []content.ToolUtilizeAction { return tool.Utilize() }

// ToolUtilizeDCResult is the closed set of outcomes of [UtilizeDC]. Variants:
// ToolUtilizeDC, ToolUtilizeActionNotListed.
type ToolUtilizeDCResult interface{ isToolUtilizeDCResult() }

// ToolUtilizeCheckResult is the closed set of outcomes of [UtilizeCheck].
// Variants: ToolUtilizeCheck, ToolUtilizeActionNotListed, ToolUtilizeBenefitsMismatch.
type ToolUtilizeCheckResult interface{ isToolUtilizeCheckResult() }

// ToolUtilizeDC is the Difficulty Class of one listed Utilize action.
type ToolUtilizeDC struct{ dc d20.DifficultyClass }

// DifficultyClass returns the DC.
func (r ToolUtilizeDC) DifficultyClass() d20.DifficultyClass { return r.dc }

// ToolUtilizeActionNotListed says the tool does not list the action, so no DC is
// known here: the host or Game Master supplies one.
type ToolUtilizeActionNotListed struct{ action content.ToolUtilizeAction }

// Action returns the action that was asked for.
func (r ToolUtilizeActionNotListed) Action() content.ToolUtilizeAction { return r.action }

// ToolUtilizeCheck is an ability check ready for the host to resolve with
// d20.Resolver: the Ability, the DC of the action and the benefits.
type ToolUtilizeCheck struct {
	ability  ability.Ability
	dc       d20.DifficultyClass
	benefits ToolCheckBenefits
}

// Ability returns the Ability to roll.
func (c ToolUtilizeCheck) Ability() ability.Ability { return c.ability }

// DifficultyClass returns the DC of the action.
func (c ToolUtilizeCheck) DifficultyClass() d20.DifficultyClass { return c.dc }

// Benefits returns the proficiency benefits of the check.
func (c ToolUtilizeCheck) Benefits() ToolCheckBenefits { return c.benefits }

// ToolUtilizeBenefitsMismatch says the benefits were evaluated for another tool
// than the one being utilized, so the pairing was not scored.
type ToolUtilizeBenefitsMismatch struct{ tool, benefitsTool content.ItemID }

// ToolID returns the proficiency id of the tool being utilized.
func (m ToolUtilizeBenefitsMismatch) ToolID() content.ItemID { return m.tool }

// BenefitsToolID returns the proficiency id the benefits were evaluated for.
func (m ToolUtilizeBenefitsMismatch) BenefitsToolID() content.ItemID { return m.benefitsTool }

func (ToolUtilizeDC) isToolUtilizeDCResult()                  {}
func (ToolUtilizeActionNotListed) isToolUtilizeDCResult()     {}
func (ToolUtilizeCheck) isToolUtilizeCheckResult()            {}
func (ToolUtilizeActionNotListed) isToolUtilizeCheckResult()  {}
func (ToolUtilizeBenefitsMismatch) isToolUtilizeCheckResult() {}

// UtilizeDC returns the DC of one Utilize action of the tool: one action per
// Utilize, never a combined DC (SRD 5.2.1, "Tools", p. 93). An action the tool
// does not list returns [ToolUtilizeActionNotListed]; no DC is guessed. It
// returns no error and never panics.
func UtilizeDC(tool ResolvedTool, action content.ToolUtilizeAction) ToolUtilizeDCResult {
	if !slices.Contains(tool.utilize, action) {
		return ToolUtilizeActionNotListed{action: action}
	}
	return ToolUtilizeDC{dc: d20.DifficultyClass(action.DC())}
}

// UtilizeCheck returns the ability check for utilizing the tool with the action:
// [ToolUtilizeCheck] with the benefits' Ability, the action's DC and the
// benefits; [ToolUtilizeActionNotListed] for an unlisted action; or
// [ToolUtilizeBenefitsMismatch] when the benefits were evaluated for another
// tool, so a wrong pairing is never silently scored (SRD 5.2.1, "Tools" and
// "Utilize [Action]", pp. 93, 191). It rolls nothing and returns no error.
func UtilizeCheck(tool ResolvedTool, action content.ToolUtilizeAction, benefits ToolCheckBenefits) ToolUtilizeCheckResult {
	dc, ok := UtilizeDC(tool, action).(ToolUtilizeDC)
	if !ok {
		return ToolUtilizeActionNotListed{action: action}
	}
	if benefits.ProficiencyID() != tool.ProficiencyID() {
		return ToolUtilizeBenefitsMismatch{tool: tool.ProficiencyID(), benefitsTool: benefits.ProficiencyID()}
	}
	return ToolUtilizeCheck{ability: benefits.Ability(), dc: dc.dc, benefits: benefits}
}
