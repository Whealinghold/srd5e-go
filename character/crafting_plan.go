package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// hoursPerCraftingDay is the working day of crafting (SRD 5.2.1, "Time", p. 103).
const hoursPerCraftingDay = 8

// ItemCraftingInput is what [PlanItemCrafting] reads. Construct it with a keyed
// literal. A zero Scribing means no scribing table, a zero Allowance the
// standard allowance of one helper and a zero GMAssignedTool none.
type ItemCraftingInput struct {
	Item           content.EquipmentEntry
	Tools          []content.Tool
	Scribing       ScrollScribing
	Party          CraftingParty
	Allowance      AssistantAllowance
	GMAssignedTool content.ItemID
}

// ItemCraftingPlan is the closed set of outcomes of [PlanItemCrafting]. Variants:
// ItemCraftingReady, ItemCraftingBlocked, ItemCraftingNotCraftable,
// ItemCraftingUseScribing, ItemCraftingNeedsGMTool, ItemCraftingGMToolInvalid.
type ItemCraftingPlan interface{ isItemCraftingPlan() }

// ItemCraftingReady says the party can craft the item.
type ItemCraftingReady struct {
	requirements CraftingRequirements
	tools        []content.ItemID
	workers      int
	hours        WorkHours
}

// Requirements returns the requirements unchanged: raw materials, days, basis.
func (r ItemCraftingReady) Requirements() CraftingRequirements { return r.requirements }

// UsableTools returns the tools every worker is proficient with, ordered by id, as a new slice.
func (r ItemCraftingReady) UsableTools() []content.ItemID { return slices.Clone(r.tools) }

// Workers returns the number of characters working: the crafter and the helpers.
func (r ItemCraftingReady) Workers() int { return r.workers }

// WorkHours returns each worker's hours: 8 per day divided by the workers, an
// exact fraction never rounded.
func (r ItemCraftingReady) WorkHours() WorkHours { return r.hours }

// ItemCraftingBlocked says the party lacks something; it lists everything at once.
type ItemCraftingBlocked struct {
	requirements CraftingRequirements
	shortfalls   []ToolCraftingShortfall
}

// Requirements returns the requirements of the item.
func (b ItemCraftingBlocked) Requirements() CraftingRequirements { return b.requirements }

// Shortfalls returns the shortfalls in order (tools, too many helpers, helpers
// not allowed), as a new slice.
func (b ItemCraftingBlocked) Shortfalls() []ToolCraftingShortfall { return slices.Clone(b.shortfalls) }

// ItemCraftingNotCraftable says the item has no crafting requirements; the party was not read.
type ItemCraftingNotCraftable struct{ reason NotCraftableReason }

// Reason returns why the item cannot be crafted.
func (n ItemCraftingNotCraftable) Reason() NotCraftableReason { return n.reason }

// ItemCraftingUseScribing says the item is a Spell Scroll, which has its own
// prerequisites ([QualifyScribe], [BeginScribing]); the party was not read.
type ItemCraftingUseScribing struct{}

// ItemCraftingNeedsGMTool says the item lists no tool and the Game Master has
// not assigned one; the party was not read.
type ItemCraftingNeedsGMTool struct{}

// ItemCraftingGMToolInvalid says the assigned tool is absent from the tools or
// is a family, which is not a proficiency; the party was not read.
type ItemCraftingGMToolInvalid struct{ tool content.ItemID }

// ToolID returns the assigned tool id.
func (g ItemCraftingGMToolInvalid) ToolID() content.ItemID { return g.tool }

func (ItemCraftingReady) isItemCraftingPlan()         {}
func (ItemCraftingBlocked) isItemCraftingPlan()       {}
func (ItemCraftingNotCraftable) isItemCraftingPlan()  {}
func (ItemCraftingUseScribing) isItemCraftingPlan()   {}
func (ItemCraftingNeedsGMTool) isItemCraftingPlan()   {}
func (ItemCraftingGMToolInvalid) isItemCraftingPlan() {}

// PlanItemCrafting says whether the party can craft the item (SRD 5.2.1,
// "Crafting Nonmagical Items" and "Brewing Potions of Healing", p. 103). It
// takes cost, days and tools from [Crafting.RequirementsFor] and never
// recomputes them. The crafter and every helper must be proficient with the one
// tool used; any alternative every worker has suffices. Helpers beyond the
// allowance, and any helper for an item with a fixed crafting override (Potion
// of Healing), are shortfalls. The Game Master's tool is read only for an item
// that lists none. An entry [Crafting.RequirementsFor] cannot read returns
// [ItemCraftingNotCraftable] with NotCraftableNoCraftingRule. It is pure,
// returns no error and never panics.
func PlanItemCrafting(in ItemCraftingInput) ItemCraftingPlan {
	outcome, err := Crafting{}.RequirementsFor(in.Item, in.Tools, in.Scribing)
	if err != nil {
		return ItemCraftingNotCraftable{reason: NotCraftableNoCraftingRule}
	}
	var req CraftingRequirements
	switch o := outcome.(type) {
	case NotCraftable:
		return ItemCraftingNotCraftable{reason: o.Reason}
	case CraftingRequirements:
		req = o
	}
	if req.Basis() == CraftingScrollRow {
		return ItemCraftingUseScribing{}
	}
	alternatives := req.Tools()
	slices.Sort(alternatives)
	if len(alternatives) == 0 && req.Basis() == CraftingGenericRule {
		if in.GMAssignedTool == "" {
			return ItemCraftingNeedsGMTool{}
		}
		if !assignableTool(in.GMAssignedTool, in.Tools) {
			return ItemCraftingGMToolInvalid{tool: in.GMAssignedTool}
		}
		alternatives = []content.ItemID{in.GMAssignedTool}
	}
	workers := in.Party.workers()
	shortfalls := toolShortfalls(workers, alternatives)
	if limit := in.Allowance.Max(); len(in.Party.helpers) > limit {
		shortfalls = append(shortfalls, ToolShortfallTooManyHelpers{count: len(in.Party.helpers), limit: limit})
	}
	if req.Basis() == CraftingItemOverride && len(in.Party.helpers) > 0 {
		shortfalls = append(shortfalls, ToolShortfallHelpersNotAllowed{basis: req.Basis()})
	}
	if len(shortfalls) > 0 {
		return ItemCraftingBlocked{requirements: req, shortfalls: shortfalls}
	}
	return ItemCraftingReady{
		requirements: req,
		tools:        sharedTools(workers, alternatives),
		workers:      len(workers),
		hours:        WorkHours{numerator: hoursPerCraftingDay * req.Days(), denominator: len(workers)},
	}
}

// assignableTool reports whether id is a tool in tools that can be proficient: not a family.
func assignableTool(id content.ItemID, tools []content.Tool) bool {
	return slices.ContainsFunc(tools, func(t content.Tool) bool {
		_, family := t.(content.ToolFamily)
		return t != nil && t.ID() == id && !family
	})
}

// sharedTools returns the alternatives every worker is proficient with; with no
// alternatives no tool is required and it returns nil.
func sharedTools(workers []CraftingWorker, alternatives []content.ItemID) []content.ItemID {
	var out []content.ItemID
	for _, a := range alternatives {
		if !slices.ContainsFunc(workers, func(w CraftingWorker) bool { return !w.tools.Contains(a) }) {
			out = append(out, a)
		}
	}
	return out
}

// toolShortfalls lists a shortfall for each worker with none of the alternatives
// (crafter first), else one for no shared tool. No alternatives means no tool is required.
func toolShortfalls(workers []CraftingWorker, alternatives []content.ItemID) []ToolCraftingShortfall {
	if len(alternatives) == 0 {
		return nil
	}
	var out []ToolCraftingShortfall
	for i, w := range workers {
		if !slices.ContainsFunc(alternatives, w.tools.Contains) {
			role := CraftingRoleHelper
			if i == 0 {
				role = CraftingRoleCrafter
			}
			out = append(out, ToolShortfallWorker{role: role, worker: w.id, alternatives: slices.Clone(alternatives)})
		}
	}
	if len(out) == 0 && len(sharedTools(workers, alternatives)) == 0 {
		out = append(out, ToolShortfallNoSharedTool{alternatives: slices.Clone(alternatives)})
	}
	return out
}
