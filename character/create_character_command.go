package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// CreateCharacterCommand is the input of CreateCharacter.Run: every choice of
// "Create Your Character" (SRD 5.2.1, p. 19). Construct it with a keyed literal.
// Run does not alter it and copies every collection.
//
// The zero value of an optional field means the default: ClassChoices is
// NoClassCreationChoices, BackgroundTool NoToolSelection, Size NoSizeChoice,
// BackgroundFeatChoices and SpeciesFeatChoices NoFeatChoices, Trinket NoTrinket,
// Options StandardRuleOptions, HigherLevelEquipment DeclineHigherLevelEquipment and
// Plan empty (a start at level 1). The ids, AbilityScores, BackgroundIncrease, both
// equipment letters and Alignment are required. Languages are the two chosen ones;
// Common is always known.
type CreateCharacterCommand struct {
	CharacterID           Identifier
	ClassID               content.ClassID
	BackgroundID          content.BackgroundID
	SpeciesID             content.SpeciesID
	AbilityScores         AbilityScoreAssignment
	BackgroundIncrease    BackgroundAbilityIncrease
	ClassSkills           []content.Skill
	ClassTools            []ChosenTool
	ClassEquipment        EquipmentOptionLetter
	BackgroundEquipment   EquipmentOptionLetter
	BackgroundTool        ToolSelection
	Size                  SizeChoice
	SpeciesSelections     []SpeciesTraitSelection
	SpeciesGrants         SpeciesGrantSelections
	BackgroundFeatChoices FeatChoices
	SpeciesFeatChoices    FeatChoices
	Languages             []content.Language
	Alignment             content.Alignment
	ClassChoices          ClassCreationChoices
	Trinket               TrinketChoice
	Options               RuleOptions
	Plan                  []StartingLevelStep
	HigherLevelEquipment  HigherLevelEquipmentChoice
}

// normalized returns a copy of c with the defaults filled in and every collection
// copied, and an error matching srderr.ErrInvalidInput for a command that cannot
// be evaluated at all (a zero or invalid required field).
func (c CreateCharacterCommand) normalized() (CreateCharacterCommand, error) {
	out := c
	out.ClassSkills = slices.Clone(c.ClassSkills)
	out.ClassTools = slices.Clone(c.ClassTools)
	out.SpeciesSelections = slices.Clone(c.SpeciesSelections)
	out.Languages = slices.Clone(c.Languages)
	out.Plan = slices.Clone(c.Plan)
	if out.BackgroundTool == nil {
		out.BackgroundTool = NoToolSelection{}
	}
	if out.Size == nil {
		out.Size = NoSizeChoice{}
	}
	if out.BackgroundFeatChoices == nil {
		out.BackgroundFeatChoices = NoFeatChoices{}
	}
	if out.SpeciesFeatChoices == nil {
		out.SpeciesFeatChoices = NoFeatChoices{}
	}
	if out.ClassChoices == nil {
		out.ClassChoices = NoClassCreationChoices{}
	}
	if out.Trinket == nil {
		out.Trinket = NoTrinket{}
	}
	if out.Options.IsZero() {
		out.Options = StandardRuleOptions()
	}
	if out.HigherLevelEquipment == nil {
		out.HigherLevelEquipment = DeclineHigherLevelEquipment{}
	}
	if err := out.shapeError(); err != nil {
		return CreateCharacterCommand{}, err
	}
	return out, nil
}

// shapeError returns the error for a command with a zero or invalid required field.
func (c CreateCharacterCommand) shapeError() error {
	bad := c.CharacterID.IsZero() || !c.ClassID.Valid() || !c.BackgroundID.Valid() || !c.SpeciesID.Valid() ||
		c.AbilityScores.IsZero() || c.BackgroundIncrease == nil || !c.ClassEquipment.Valid() || !c.BackgroundEquipment.Valid() ||
		!c.Alignment.Valid() || slices.ContainsFunc(c.ClassSkills, func(s content.Skill) bool { return !s.Valid() }) ||
		slices.ContainsFunc(c.ClassTools, func(t ChosenTool) bool { return !t.Tool.Valid() }) ||
		slices.ContainsFunc(c.Languages, func(l content.Language) bool { return !l.Valid() })
	if bad {
		return invalidInput("create a character: a zero or invalid required field")
	}
	switch v := c.BackgroundIncrease.(type) {
	case PlusOneToAll:
	case PlusTwoPlusOne:
		_, err := NewPlusTwoPlusOne(v.PlusTwo(), v.PlusOne())
		bad = err != nil
	default:
		bad = true
	}
	switch t := c.BackgroundTool.(type) {
	case NoToolSelection:
	case ChosenTool:
		bad = bad || !t.Tool.Valid()
	default:
		bad = true
	}
	switch s := c.Size.(type) {
	case NoSizeChoice:
	case ChosenSize:
		bad = bad || !s.Size.Valid()
	default:
		bad = true
	}
	switch c.Trinket.(type) {
	case NoTrinket, RollForTrinket:
	default:
		bad = true
	}
	switch c.HigherLevelEquipment.(type) {
	case DeclineHigherLevelEquipment, GrantHigherLevelEquipment:
	default:
		bad = true
	}
	for _, s := range c.Plan {
		switch s.HitPointChoice.(type) {
		case RollHitDie, TakeFixedValue:
		default:
			bad = true
		}
		bad = bad || !s.ClassID.Valid() || (s.SubclassID != "" && !s.SubclassID.Valid())
	}
	if _, err := NewLevel(c.Options.StartingLevel()); err != nil {
		bad = true
	}
	if bad {
		return invalidInput("create a character: an invalid choice value")
	}
	return nil
}
