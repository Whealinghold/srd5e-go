package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// CraftingWorker is one character who works on a crafted item: an id and the
// tools they are proficient with. Immutable. The zero value is invalid.
type CraftingWorker struct {
	id    Identifier
	tools ToolProficiencies
}

// NewCraftingWorker returns the worker. The id is validated by [NewCraftingParty].
func NewCraftingWorker(id Identifier, tools ToolProficiencies) CraftingWorker {
	return CraftingWorker{id: id, tools: tools}
}

// ID returns the character's id.
func (w CraftingWorker) ID() Identifier { return w.id }

// Tools returns the tool proficiencies.
func (w CraftingWorker) Tools() ToolProficiencies { return w.tools }

// CraftingParty is the crafter and anyone who helps them (SRD 5.2.1, "Time",
// p. 103). Immutable. The zero value is invalid ([CraftingParty.IsZero]); every
// plan with it is blocked on the missing crafter.
type CraftingParty struct {
	crafter CraftingWorker
	helpers []CraftingWorker
}

// NewCraftingParty returns the party, copying helpers. It returns an error
// matching srderr.ErrInvalidInput when a worker has a zero id or an id repeats
// (a helper equal to the crafter or to another helper). It never panics.
func NewCraftingParty(crafter CraftingWorker, helpers ...CraftingWorker) (CraftingParty, error) {
	all := append([]CraftingWorker{crafter}, helpers...)
	for i, w := range all {
		if w.id.IsZero() {
			return CraftingParty{}, invalidInput("crafting party: worker without an id")
		}
		if slices.ContainsFunc(all[:i], func(o CraftingWorker) bool { return o.id == w.id }) {
			return CraftingParty{}, invalidInput("crafting party: repeated worker id")
		}
	}
	return CraftingParty{crafter: crafter, helpers: slices.Clone(helpers)}, nil
}

// MustCraftingParty is NewCraftingParty for known-valid input. It panics with
// the error NewCraftingParty returns, and only then.
func MustCraftingParty(crafter CraftingWorker, helpers ...CraftingWorker) CraftingParty {
	p, err := NewCraftingParty(crafter, helpers...)
	if err != nil {
		panic(err)
	}
	return p
}

// IsZero reports whether p is the invalid zero party.
func (p CraftingParty) IsZero() bool { return p.crafter.id.IsZero() }

// Crafter returns the crafter.
func (p CraftingParty) Crafter() CraftingWorker { return p.crafter }

// Helpers returns the helpers as a new slice.
func (p CraftingParty) Helpers() []CraftingWorker { return slices.Clone(p.helpers) }

// workers returns the crafter then the helpers.
func (p CraftingParty) workers() []CraftingWorker {
	return append([]CraftingWorker{p.crafter}, p.helpers...)
}

// CraftingRole says whether a worker is the crafter or a helper. The zero value
// is invalid.
type CraftingRole int

// The roles.
const (
	CraftingRoleCrafter CraftingRole = iota + 1
	CraftingRoleHelper
)

// String returns "crafter", "helper" or "unknown".
func (r CraftingRole) String() string {
	switch r {
	case CraftingRoleCrafter:
		return "crafter"
	case CraftingRoleHelper:
		return "helper"
	}
	return "unknown"
}

// ToolCraftingShortfall is the closed set of things a party lacks to craft an
// item. Variants: ToolShortfallWorker, ToolShortfallNoSharedTool,
// ToolShortfallTooManyHelpers, ToolShortfallHelpersNotAllowed.
type ToolCraftingShortfall interface{ isToolCraftingShortfall() }

// ToolShortfallWorker says a worker is proficient with none of the tools that
// would do.
type ToolShortfallWorker struct {
	role         CraftingRole
	worker       Identifier
	alternatives []content.ItemID
}

// Role returns whether the worker is the crafter or a helper.
func (s ToolShortfallWorker) Role() CraftingRole { return s.role }

// WorkerID returns the worker's id.
func (s ToolShortfallWorker) WorkerID() Identifier { return s.worker }

// Alternatives returns the tools of which proficiency in any one would do, as a new slice.
func (s ToolShortfallWorker) Alternatives() []content.ItemID { return slices.Clone(s.alternatives) }

// ToolShortfallNoSharedTool says every worker has proficiency with some
// alternative but no single alternative is shared by all.
type ToolShortfallNoSharedTool struct{ alternatives []content.ItemID }

// Alternatives returns the tools of which proficiency in any one would do, as a new slice.
func (s ToolShortfallNoSharedTool) Alternatives() []content.ItemID {
	return slices.Clone(s.alternatives)
}

// ToolShortfallTooManyHelpers says there are more helpers than the Game Master allows.
type ToolShortfallTooManyHelpers struct{ count, limit int }

// Count returns the number of helpers.
func (s ToolShortfallTooManyHelpers) Count() int { return s.count }

// Limit returns the number of helpers allowed.
func (s ToolShortfallTooManyHelpers) Limit() int { return s.limit }

// ToolShortfallHelpersNotAllowed says the item is crafted by one character
// alone (interpretation: the potion brewing text names no helper rule).
type ToolShortfallHelpersNotAllowed struct{ basis CraftingBasis }

// Basis returns the crafting basis that forbids helpers.
func (s ToolShortfallHelpersNotAllowed) Basis() CraftingBasis { return s.basis }

func (ToolShortfallWorker) isToolCraftingShortfall()            {}
func (ToolShortfallNoSharedTool) isToolCraftingShortfall()      {}
func (ToolShortfallTooManyHelpers) isToolCraftingShortfall()    {}
func (ToolShortfallHelpersNotAllowed) isToolCraftingShortfall() {}
