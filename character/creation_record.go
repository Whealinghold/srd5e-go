package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// StartingEquipmentGrant is the starting equipment of a new character: the items of
// the chosen class option, then those of the chosen background option, in that
// order, and the coins as one Cost in Gold Pieces: the gold of both options plus
// the allowance total when the GM grants one (SRD 5.2.1, "Choose Starting
// Equipment", p. 20). Coins stay coins; nothing is bought with them. It is
// immutable. The zero value holds nothing.
type StartingEquipmentGrant struct {
	items []content.EquipmentItem
	gold  content.Cost
}

// Items returns a new slice of the items, class option first.
func (g StartingEquipmentGrant) Items() []content.EquipmentItem { return slices.Clone(g.items) }

// Gold returns the coins, in Gold Pieces; the zero Cost for the zero grant.
func (g StartingEquipmentGrant) Gold() content.Cost { return g.gold }

// CreationRecord is what a created character carries that the Character aggregate
// does not store (SRD 5.2.1, "Create Your Character", p. 19): the choices behind the
// scores, the proficiencies, the Origin feats, the species selections, the
// languages, the alignment, the starting equipment and the trinket roll. It holds
// nothing Character stores, and none of the class or species data, which stay
// readable from the content entries. The host keeps it beside the Character. It is
// immutable; every collection is returned as a copy. It is not comparable with ==;
// the zero value is not a record.
type CreationRecord struct {
	method          AbilityScoreGenerationMethod
	assigned        ability.Scores
	increase        BackgroundAbilityIncrease
	skills          []content.Skill
	overlapping     []content.Skill
	backgroundTool  content.ToolProficiency
	backgroundChose ToolSelection
	classTools      []ChosenTool
	feat            content.FeatID
	required        RequiredSpellList
	size            content.CreatureSize
	selections      []SpeciesTraitSelection
	grants          SpeciesGrantSelections
	languages       Languages
	alignment       content.Alignment
	classChoices    ClassCreationChoices
	equipment       StartingEquipmentGrant
	allowance       HigherLevelEquipment
	allowanceRoll   int
	trinket         TrinketRoll
	feats           TakenFeats
	featProfs       []ProficiencyGrants
	featSpells      []FeatGrantedSpells
	consumed        []ConsumedFeatGrant
	startingLevel   Level
}

// Method returns the ability score generation method.
func (r CreationRecord) Method() AbilityScoreGenerationMethod { return r.method }

// AssignedScores returns the scores as assigned, before the background increase.
func (r CreationRecord) AssignedScores() ability.Scores { return r.assigned }

// Increase returns the background increase applied to them.
func (r CreationRecord) Increase() BackgroundAbilityIncrease { return r.increase }

// Skills returns a new slice of the skill proficiencies: the class skills, the
// background skills and the skill of a species grant, each once, in that order.
func (r CreationRecord) Skills() []content.Skill { return slices.Clone(r.skills) }

// OverlappingSkills returns a new slice of the skills more than one source named.
func (r CreationRecord) OverlappingSkills() []content.Skill { return slices.Clone(r.overlapping) }

// BackgroundTool returns the background's Tool Proficiency as the content states it.
func (r CreationRecord) BackgroundTool() content.ToolProficiency { return r.backgroundTool }

// BackgroundChosenTool returns the tool chosen for the background.
func (r CreationRecord) BackgroundChosenTool() ToolSelection { return r.backgroundChose }

// ClassTools returns a new slice of the tools chosen for the class.
func (r CreationRecord) ClassTools() []ChosenTool { return slices.Clone(r.classTools) }

// BackgroundFeat returns the background's Origin feat and the spell list it requires.
func (r CreationRecord) BackgroundFeat() (content.FeatID, RequiredSpellList) {
	return r.feat, r.required
}

// Size returns the size of the character.
func (r CreationRecord) Size() content.CreatureSize { return r.size }

// SpeciesSelections returns a new slice of the species choices made.
func (r CreationRecord) SpeciesSelections() []SpeciesTraitSelection {
	return slices.Clone(r.selections)
}

// SpeciesGrants returns the species grant selections.
func (r CreationRecord) SpeciesGrants() SpeciesGrantSelections { return r.grants }

// Languages returns the languages known.
func (r CreationRecord) Languages() Languages { return r.languages }

// Alignment returns the alignment.
func (r CreationRecord) Alignment() content.Alignment { return r.alignment }

// ClassChoices returns the class creation choices, as given.
func (r CreationRecord) ClassChoices() ClassCreationChoices { return r.classChoices }

// Equipment returns the starting equipment grant.
func (r CreationRecord) Equipment() StartingEquipmentGrant { return r.equipment }

// Allowance returns the Starting Equipment at Higher Levels allowance, or
// NoHigherLevelEquipment when the GM granted none; its magic item counts are
// recorded as they are.
func (r CreationRecord) Allowance() HigherLevelEquipment { return r.allowance }

// AllowanceRoll returns the roll of the allowance's rolled gold and true, or false
// when none was rolled.
func (r CreationRecord) AllowanceRoll() (int, bool) { return r.allowanceRoll, r.allowanceRoll != 0 }

// Trinket returns the trinket roll and true, or false when none was rolled. The
// text is TrinketTable.EntryFor of the roll and is not copied here.
func (r CreationRecord) Trinket() (TrinketRoll, bool) { return r.trinket, !r.trinket.IsZero() }

// Feats returns the feats the character took at creation.
func (r CreationRecord) Feats() TakenFeats { return r.feats }

// ProficiencyGrants returns a new slice of the proficiencies the Origin feats gave,
// background feat first.
func (r CreationRecord) ProficiencyGrants() []ProficiencyGrants { return slices.Clone(r.featProfs) }

// GrantedSpells returns a new slice of the spells the Origin feats gave, background
// feat first.
func (r CreationRecord) GrantedSpells() []FeatGrantedSpells { return slices.Clone(r.featSpells) }

// ConsumedGrants returns a new slice of the feat grants the plan's increases used,
// in step order.
func (r CreationRecord) ConsumedGrants() []ConsumedFeatGrant { return slices.Clone(r.consumed) }

// StartingLevel returns the level the character started at.
func (r CreationRecord) StartingLevel() Level { return r.startingLevel }
