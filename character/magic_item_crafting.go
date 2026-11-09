package character

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
)

// WorkHours is an exact number of hours as a fraction, never rounded. The zero
// value is invalid. Comparable.
type WorkHours struct{ numerator, denominator int }

// NewWorkHours returns numerator/denominator hours. It returns an error matching
// srderr.ErrInvalidInput for a negative numerator or a denominator below 1.
func NewWorkHours(numerator, denominator int) (WorkHours, error) {
	if numerator < 0 || denominator < 1 {
		return WorkHours{}, invalidInput("work hours: negative numerator or denominator below 1")
	}
	return WorkHours{numerator: numerator, denominator: denominator}, nil
}

// Numerator returns the numerator.
func (h WorkHours) Numerator() int { return h.numerator }

// Denominator returns the denominator, at least 1.
func (h WorkHours) Denominator() int { return h.denominator }

// AssistantAllowance is the most assistants a crafter may have. The zero value is
// the standard allowance, 1.
type AssistantAllowance int

// NewAssistantAllowance returns the allowance of n assistants when the Game
// Master allows more than one. It returns an error matching
// srderr.ErrInvalidInput for n below 2.
func NewAssistantAllowance(n int) (AssistantAllowance, error) {
	if n < 2 {
		return 0, invalidInput("assistant allowance below 2")
	}
	return AssistantAllowance(n), nil
}

// Max returns the number of assistants allowed, 1 for the standard allowance.
func (a AssistantAllowance) Max() int { return max(int(a), 1) }

// CraftingMemberID names a member of a crafting team. The host supplies it. The
// zero value is invalid.
type CraftingMemberID string

// CraftingTeam is the crafter and the assistants. Immutable; the zero value is
// invalid.
type CraftingTeam struct {
	crafter    CraftingMemberID
	assistants []CraftingMemberID
}

// NewCraftingTeam returns the team, copying assistants. It returns an error
// matching srderr.ErrInvalidInput for a blank or repeated member.
func NewCraftingTeam(crafter CraftingMemberID, assistants []CraftingMemberID) (CraftingTeam, error) {
	seen := []CraftingMemberID{crafter}
	for _, a := range assistants {
		if slices.Contains(seen, a) {
			return CraftingTeam{}, invalidInput("crafting team: repeated member")
		}
		seen = append(seen, a)
	}
	if slices.ContainsFunc(seen, func(m CraftingMemberID) bool { return strings.TrimSpace(string(m)) == "" }) {
		return CraftingTeam{}, invalidInput("crafting team: blank member")
	}
	return CraftingTeam{crafter: crafter, assistants: slices.Clone(assistants)}, nil
}

// Crafter returns the crafter.
func (t CraftingTeam) Crafter() CraftingMemberID { return t.crafter }

// Assistants returns a new slice of the assistants.
func (t CraftingTeam) Assistants() []CraftingMemberID { return slices.Clone(t.assistants) }

// Workers returns the number of characters working: the crafter and the assistants.
func (t CraftingTeam) Workers() int { return 1 + len(t.assistants) }

// IsZero reports whether t is the invalid zero team.
func (t CraftingTeam) IsZero() bool { return t.crafter == "" }

// CrafterQualification is what one team member can do: Arcana proficiency, the
// tools they are proficient with and, for the crafter, the spells prepared each
// day. Immutable.
type CrafterQualification struct {
	member                CraftingMemberID
	arcana                bool
	tools                 []content.ItemID
	preparedSpellsEachDay []content.SpellID
}

// NewCrafterQualification returns the qualification, copying the slices. It
// returns an error matching srderr.ErrInvalidInput for a blank member.
func NewCrafterQualification(member CraftingMemberID, arcana bool, tools []content.ItemID, preparedSpellsEachDay []content.SpellID) (CrafterQualification, error) {
	if strings.TrimSpace(string(member)) == "" {
		return CrafterQualification{}, invalidInput("crafter qualification: blank member")
	}
	return CrafterQualification{member: member, arcana: arcana, tools: slices.Clone(tools), preparedSpellsEachDay: slices.Clone(preparedSpellsEachDay)}, nil
}

// Member returns the member.
func (q CrafterQualification) Member() CraftingMemberID { return q.member }

// ShortfallKind is the kind of a crafting shortfall. The zero value is invalid.
type ShortfallKind int

// The kinds.
const (
	ShortfallArcanaProficiency ShortfallKind = iota + 1
	ShortfallToolProficiency
	ShortfallSpellNotPrepared
	ShortfallTooManyAssistants
)

// String returns the snake_case name used as the rejection attribute, or "unknown".
func (k ShortfallKind) String() string {
	names := []string{"arcana", "tool", "spell_not_prepared", "too_many_assistants"}
	if k < 1 || int(k) > len(names) {
		return "unknown"
	}
	return names[k-1]
}

// MagicItemCraftingShortfall is one thing a team lacks. Fields by kind: Member
// for arcana and tool; Tools (the alternatives) for tool; Spell for
// spell-not-prepared. Construct with the functions below.
type MagicItemCraftingShortfall struct {
	Kind   ShortfallKind
	Member CraftingMemberID
	Tools  []content.ItemID
	Spell  content.SpellID
}

// MagicItemCrafting holds the rules of crafting magic items. The zero value is ready to use.
type MagicItemCrafting struct{}

// ToolsFor returns the tools of which the crafter and each assistant need
// proficiency in any one: the category's fixed tools; Crafting.ToolsFor(base)
// for armor and weapon; Tinker's Tools or the base item's tools for a wondrous
// item (Tinker's Tools alone with no base) (SRD 5.2.1, "Magic Item Tools",
// p. 207). It returns an error matching srderr.ErrInvalidInput for a category
// that needs a base item and gets none, or an unknown category.
func (MagicItemCrafting) ToolsFor(category content.MagicItemCategory, base content.EquipmentEntry, tools []content.Tool) ([]content.ItemID, error) {
	switch r := (content.MagicItemToolTable{}).Of(category).(type) {
	case content.FixedTools:
		return r.Tools(), nil
	case content.ByBaseItemTools:
		if base == nil {
			return nil, invalidInput("crafting tools: armor or weapon without a base item")
		}
		return Crafting{}.ToolsFor(base, tools), nil
	case content.TinkersOrBaseItemTools:
		out := []content.ItemID{"tinkers_tools"}
		if base != nil {
			out = append(out, Crafting{}.ToolsFor(base, tools)...)
		}
		return out, nil
	}
	return nil, invalidInput("crafting tools: unknown category")
}

// Shortfalls reports every shortfall at once: for each team member in order (the
// crafter first) Arcana proficiency then tool proficiency in any of toolOptions;
// then each of spells the crafter has not prepared (assistants are not asked);
// then ShortfallTooManyAssistants when the team has more assistants than
// allowance admits. A member with no qualification lacks everything.
func (MagicItemCrafting) Shortfalls(team CraftingTeam, quals []CrafterQualification, toolOptions []content.ItemID, spells []content.SpellID, allowance AssistantAllowance) []MagicItemCraftingShortfall {
	var out []MagicItemCraftingShortfall
	qual := func(m CraftingMemberID) CrafterQualification {
		i := slices.IndexFunc(quals, func(q CrafterQualification) bool { return q.member == m })
		if i < 0 {
			return CrafterQualification{member: m}
		}
		return quals[i]
	}
	for _, m := range append([]CraftingMemberID{team.crafter}, team.assistants...) {
		q := qual(m)
		if !q.arcana {
			out = append(out, MagicItemCraftingShortfall{Kind: ShortfallArcanaProficiency, Member: m})
		}
		if len(toolOptions) > 0 && !slices.ContainsFunc(q.tools, func(t content.ItemID) bool { return slices.Contains(toolOptions, t) }) {
			out = append(out, MagicItemCraftingShortfall{Kind: ShortfallToolProficiency, Member: m, Tools: slices.Clone(toolOptions)})
		}
	}
	prepared := qual(team.crafter).preparedSpellsEachDay
	for _, s := range spells {
		if !slices.Contains(prepared, s) {
			out = append(out, MagicItemCraftingShortfall{Kind: ShortfallSpellNotPrepared, Spell: s})
		}
	}
	if len(team.assistants) > allowance.Max() {
		out = append(out, MagicItemCraftingShortfall{Kind: ShortfallTooManyAssistants})
	}
	return out
}

// IncorporatedItemChoice is how a crafter deals with an incorporated item that
// has a purchase cost. The zero value is no choice.
type IncorporatedItemChoice int

// The choices (SRD 5.2.1, "Time and Cost", p. 207).
const (
	PayPurchaseCost IncorporatedItemChoice = iota + 1
	CraftIncorporatedItem
)

// MagicItemCraftingOutcome is the closed set of crafting requirements.
// Variants: MagicItemCraftingRequirements, MagicItemUsesEquipmentRules,
// MagicItemNotCraftable.
type MagicItemCraftingOutcome interface{ isMagicItemCraftingOutcome() }

// MagicItemCraftingRequirements: the time and cost of crafting. Days are of 8
// hours that need not be consecutive.
type MagicItemCraftingRequirements struct {
	hoursPerWorker WorkHours
	totalHours     int
	costCopper     int64
	choice         IncorporatedItemChoice
	incorporated   CraftingOutcome
}

// WorkHoursPerWorker returns the hours each worker works: the work hours divided
// by the number of characters working, exactly.
func (r MagicItemCraftingRequirements) WorkHoursPerWorker() WorkHours { return r.hoursPerWorker }

// TotalWorkHours returns the work hours of the table row, before sharing.
func (r MagicItemCraftingRequirements) TotalWorkHours() int { return r.totalHours }

// CostCopperPieces returns the raw material cost in Copper Pieces.
func (r MagicItemCraftingRequirements) CostCopperPieces() int64 { return r.costCopper }

// WorkHoursPerDay returns 8.
func (MagicItemCraftingRequirements) WorkHoursPerDay() int { return 8 }

// DaysNeedNotBeConsecutive returns true.
func (MagicItemCraftingRequirements) DaysNeedNotBeConsecutive() bool { return true }

// IncorporatedChoice returns how the incorporated item is handled, or the zero choice.
func (r MagicItemCraftingRequirements) IncorporatedChoice() IncorporatedItemChoice { return r.choice }

// IncorporatedOutcome returns the requirements to craft the incorporated item
// (kept apart, never summed with these), or nil.
func (r MagicItemCraftingRequirements) IncorporatedOutcome() CraftingOutcome { return r.incorporated }

// MagicItemUsesEquipmentRules: the item is a Spell Scroll or a Potion of
// Healing, crafted by the equipment rules.
type MagicItemUsesEquipmentRules struct {
	Item    content.ItemID
	Outcome CraftingOutcome
}

// MagicItemNotCraftableReason says why an item cannot be crafted.
type MagicItemNotCraftableReason int

// MagicItemNotCraftableArtifact: the crafting table has no row for an Artifact.
const MagicItemNotCraftableArtifact MagicItemNotCraftableReason = 1

// MagicItemNotCraftable: the item cannot be crafted. It is an outcome, not an error.
type MagicItemNotCraftable struct{ Reason MagicItemNotCraftableReason }

func (MagicItemCraftingRequirements) isMagicItemCraftingOutcome() {}
func (MagicItemUsesEquipmentRules) isMagicItemCraftingOutcome()   {}
func (MagicItemNotCraftable) isMagicItemCraftingOutcome()         {}

// MagicItemCraftingInput is what RequirementsFor needs. Construct it with a
// keyed literal.
type MagicItemCraftingInput struct {
	Rarity     content.ItemRarity
	Consumable bool
	Basis      content.MagicItemCraftingBasis
	// GearOutcome is Crafting.RequirementsFor of the gear item, for an
	// EquipmentRulesCrafting basis.
	GearOutcome CraftingOutcome
	// Choice, IncorporatedCostCopper and IncorporatedOutcome describe the
	// incorporated item; Choice is the zero value when there is none.
	Choice                 IncorporatedItemChoice
	IncorporatedCostCopper int64
	IncorporatedOutcome    CraftingOutcome
	// Workers is the number of characters working, at least 1.
	Workers int
}

// RequirementsFor returns the time and cost of crafting. An Artifact is
// MagicItemNotCraftable whatever else is true; an item crafted by the equipment
// rules is MagicItemUsesEquipmentRules carrying GearOutcome; otherwise it is the
// table row of the rarity, halved for a consumable, plus the incorporated item's
// cost when the crafter pays it, with the time split among the workers (SRD
// 5.2.1, "Magic Item Crafting Time and Cost", p. 207). It returns an error
// matching srderr.ErrInvalidInput for Workers below 1, a missing GearOutcome or
// IncorporatedOutcome the basis or choice needs, a negative cost or an unknown
// rarity.
func (MagicItemCrafting) RequirementsFor(in MagicItemCraftingInput) (MagicItemCraftingOutcome, error) {
	if in.Workers < 1 || in.IncorporatedCostCopper < 0 {
		return nil, invalidInput("crafting requirements: workers below 1 or negative cost")
	}
	if in.Rarity == content.RarityArtifact {
		return MagicItemNotCraftable{Reason: MagicItemNotCraftableArtifact}, nil
	}
	if gear, ok := in.Basis.(content.EquipmentRulesCrafting); ok {
		if in.GearOutcome == nil {
			return nil, invalidInput("crafting requirements: equipment rules basis without an outcome")
		}
		return MagicItemUsesEquipmentRules{Item: gear.Item(), Outcome: in.GearOutcome}, nil
	}
	found, ok := content.ItemRarityFacts{}.Of(in.Rarity).Crafting().(content.MagicItemCraftingRowFound)
	if !ok {
		return nil, invalidInput("crafting requirements: unknown rarity")
	}
	hours, cost := found.Row().WorkHours(), found.Row().Cost().CopperPieces()
	if in.Consumable {
		hours, cost = hours/2, cost/2
	}
	out := MagicItemCraftingRequirements{totalHours: hours, costCopper: cost, choice: in.Choice, hoursPerWorker: WorkHours{numerator: hours, denominator: in.Workers}}
	switch in.Choice {
	case PayPurchaseCost:
		out.costCopper += in.IncorporatedCostCopper
	case CraftIncorporatedItem:
		if in.IncorporatedOutcome == nil {
			return nil, invalidInput("crafting requirements: crafting the incorporated item without its outcome")
		}
		out.incorporated = in.IncorporatedOutcome
	}
	return out, nil
}

// CraftingMaterials decides whether raw materials can be found. The zero value is ready to use.
type CraftingMaterials struct{}

// MaterialsWaitDays is how long a crafter waits before checking again.
const MaterialsWaitDays = 7

// IsAvailable reports whether the d100 roll finds materials: at most 75 in a
// city, at most 25 in a town or village (SRD 5.2.1, "Time and Cost", p. 207). It
// returns an error matching srderr.ErrInvalidInput for a roll outside 1 to 100 or
// an unknown settlement.
func (CraftingMaterials) IsAvailable(settlement content.SettlementSize, d100 int) (bool, error) {
	if d100 < 1 || d100 > 100 || !settlement.Valid() {
		return false, invalidInput("crafting materials: roll outside 1 to 100 or unknown settlement")
	}
	if settlement == content.SettlementCity {
		return d100 <= 75, nil
	}
	return d100 <= 25, nil
}
