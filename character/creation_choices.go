package character

import (
	"maps"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// ToolSelection is the tool a player chooses for a Tool Proficiency that asks for
// one (SRD 5.2.1, "Soldier", p. 83). It is a closed set. Variants:
// NoToolSelection, ChosenTool.
type ToolSelection interface{ isToolSelection() }

// NoToolSelection chooses no tool.
type NoToolSelection struct{}

// ChosenTool chooses the tool Tool. Construct it with a keyed literal.
type ChosenTool struct{ Tool content.ItemID }

func (NoToolSelection) isToolSelection() {}
func (ChosenTool) isToolSelection()      {}

// EquipmentOptionLetter is the letter of a Starting Equipment option (SRD 5.2.1,
// "Choose Starting Equipment", p. 20). It matches the uppercase key of the
// content option: a is A, b is B and c is C. The zero value is invalid.
type EquipmentOptionLetter int

// The Starting Equipment option letters.
const (
	// OptionA is option A.
	OptionA EquipmentOptionLetter = iota + 1
	// OptionB is option B.
	OptionB
	// OptionC is option C; only some classes have it.
	OptionC
)

// Valid reports whether l is a, b or c.
func (l EquipmentOptionLetter) Valid() bool { return l >= OptionA && l <= OptionC }

// String returns "a", "b", "c" or "unknown".
func (l EquipmentOptionLetter) String() string {
	if !l.Valid() {
		return "unknown"
	}
	return "abc"[l-1 : l]
}

// SizeChoice is the size chosen when a species offers two (SRD 5.2.1, "Choose a
// Species", p. 20). It is a closed set. Variants: NoSizeChoice, ChosenSize.
type SizeChoice interface{ isSizeChoice() }

// NoSizeChoice chooses no size; enough for a species with one size.
type NoSizeChoice struct{}

// ChosenSize chooses the size Size. Construct it with a keyed literal.
type ChosenSize struct{ Size content.CreatureSize }

func (NoSizeChoice) isSizeChoice() {}
func (ChosenSize) isSizeChoice()   {}

// SpeciesTraitSelection answers a species choice (Draconic Ancestry, Elven
// Lineage and the like): the trait's name and the name of the option chosen.
// Construct it with a keyed literal. It is comparable with ==.
type SpeciesTraitSelection struct{ Trait, Option string }

// SpeciesGrantSelections answers what a species asks of the player when selected
// (SRD 5.2.1, "Human", "Elf", "Gnome", "Tiefling", pp. 84-86): a skill, an Origin
// feat and, for each lineage trait, the spellcasting ability. Build it with
// NewSpeciesGrantSelections; the zero value chooses nothing. It holds a map, so it
// is not comparable with ==; use Equal.
type SpeciesGrantSelections struct {
	skill     content.Skill
	feat      content.FeatID
	abilities map[string]ability.Ability
}

// NewSpeciesGrantSelections returns the selections. A zero skill, an empty feat
// id and a nil map mean "none". abilities is copied. It returns an error matching
// srderr.ErrInvalidInput for a non-zero invalid skill or feat id, or an invalid
// ability; whether an ability is Intelligence, Wisdom or Charisma is for
// CreateCharacter to report. It never panics.
func NewSpeciesGrantSelections(skill content.Skill, feat content.FeatID, abilities map[string]ability.Ability) (SpeciesGrantSelections, error) {
	if (skill != 0 && !skill.Valid()) || (feat != "" && !feat.Valid()) {
		return SpeciesGrantSelections{}, invalidInput("species grant with an invalid skill or feat")
	}
	for _, a := range abilities {
		if !a.Valid() {
			return SpeciesGrantSelections{}, invalidInput("species grant with an invalid ability")
		}
	}
	return SpeciesGrantSelections{skill: skill, feat: feat, abilities: maps.Clone(abilities)}, nil
}

// Skill returns the chosen skill and true, or false for none.
func (s SpeciesGrantSelections) Skill() (content.Skill, bool) { return s.skill, s.skill != 0 }

// Feat returns the chosen Origin feat and true, or false for none.
func (s SpeciesGrantSelections) Feat() (content.FeatID, bool) { return s.feat, s.feat != "" }

// Abilities returns a new map of the spellcasting ability by lineage trait name.
func (s SpeciesGrantSelections) Abilities() map[string]ability.Ability {
	return maps.Clone(s.abilities)
}

// Equal reports whether s and o hold the same selections.
func (s SpeciesGrantSelections) Equal(o SpeciesGrantSelections) bool {
	return s.skill == o.skill && s.feat == o.feat && maps.Equal(s.abilities, o.abilities)
}

// HigherLevelEquipmentChoice is whether the GM grants the Starting Equipment at
// Higher Levels (SRD 5.2.1, "Starting Equipment", p. 24). It is a closed set.
// Variants: DeclineHigherLevelEquipment, GrantHigherLevelEquipment.
type HigherLevelEquipmentChoice interface{ isHigherLevelEquipmentChoice() }

// DeclineHigherLevelEquipment grants only the class and background options. It is
// the default.
type DeclineHigherLevelEquipment struct{}

// GrantHigherLevelEquipment adds the allowance of the starting level.
type GrantHigherLevelEquipment struct{}

func (DeclineHigherLevelEquipment) isHigherLevelEquipmentChoice() {}
func (GrantHigherLevelEquipment) isHigherLevelEquipmentChoice()   {}

// TrinketChoice is whether the new character rolls for a Trinket (SRD 5.2.1,
// "Trinkets", p. 26). It is a closed set. Variants: NoTrinket, RollForTrinket.
type TrinketChoice interface{ isTrinketChoice() }

// NoTrinket rolls nothing. It is the default.
type NoTrinket struct{}

// RollForTrinket rolls one 1d100 on the Trinkets table.
type RollForTrinket struct{}

func (NoTrinket) isTrinketChoice()      {}
func (RollForTrinket) isTrinketChoice() {}

// TrinketRoll is a roll of the 1d100 on the Trinkets table, 1 to 100 (a roll of
// 100 is the table's 00). It is comparable with ==. The zero value is invalid.
type TrinketRoll struct{ n int }

// NewTrinketRoll returns the roll n. It returns an error matching
// srderr.ErrInvalidInput for n below 1 or above 100. It never panics.
func NewTrinketRoll(n int) (TrinketRoll, error) {
	if n < 1 || n > TrinketCount {
		return TrinketRoll{}, invalidInput("trinket roll outside 1 to 100")
	}
	return TrinketRoll{n: n}, nil
}

// Value returns the roll, 1 to 100; 0 for the zero TrinketRoll.
func (r TrinketRoll) Value() int { return r.n }

// IsZero reports whether r is the invalid zero TrinketRoll.
func (r TrinketRoll) IsZero() bool { return r.n == 0 }
