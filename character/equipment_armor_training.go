package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ArmorTrainingRules is the stateless service that compares the armor worn
// with the Armor Training held (SRD 5.2.1, "Armor Training", p. 177). The zero
// value is ready to use. Its inputs are plain category lists treated as sets.
type ArmorTrainingRules struct{}

// ArmorTrainingEffects is what wearing armor without training costs. It is an
// immutable value; it holds a slice, so compare two with Equal, not ==.
type ArmorTrainingEffects struct {
	untrained []content.ArmorCategory // canonical order: Light, Medium, Heavy, Shield
}

// Assess returns the effects of wearing the categories in worn while trained
// in those in trained: Untrained holds exactly the worn categories that are
// not trained, in the order Light, Medium, Heavy, Shield. Order and duplicates
// in either input are irrelevant, and neither input is altered.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid category
// or for more than one of Light, Medium and Heavy in worn ("only one suit of
// armor at a time", SRD 5.2.1, "One at a Time", p. 92). An empty worn is valid.
func (ArmorTrainingRules) Assess(worn, trained []content.ArmorCategory) (ArmorTrainingEffects, error) {
	if slices.ContainsFunc(worn, func(c content.ArmorCategory) bool { return !c.Valid() }) ||
		slices.ContainsFunc(trained, func(c content.ArmorCategory) bool { return !c.Valid() }) {
		return ArmorTrainingEffects{}, invalidInput("armor category")
	}
	bodies := 0
	for _, c := range []content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy} {
		if slices.Contains(worn, c) {
			bodies++
		}
	}
	if bodies > 1 {
		return ArmorTrainingEffects{}, invalidInput("more than one suit of armor worn")
	}
	var untrained []content.ArmorCategory
	for _, c := range []content.ArmorCategory{content.ArmorLight, content.ArmorMedium, content.ArmorHeavy, content.ArmorShield} {
		if slices.Contains(worn, c) && !slices.Contains(trained, c) {
			untrained = append(untrained, c)
		}
	}
	return ArmorTrainingEffects{untrained: untrained}, nil
}

// MonsterTraining returns a copy of statBlockArmor: a monster has training
// with any armor in its stat block (SRD 5.2.1, "Armor", p. 92).
func (ArmorTrainingRules) MonsterTraining(statBlockArmor []content.ArmorCategory) []content.ArmorCategory {
	return slices.Clone(statBlockArmor)
}

// Untrained returns a fresh slice of the worn categories without training.
func (e ArmorTrainingEffects) Untrained() []content.ArmorCategory {
	return slices.Clone(e.untrained)
}

// Equal reports whether e and o have the same untrained categories.
func (e ArmorTrainingEffects) Equal(o ArmorTrainingEffects) bool {
	return slices.Equal(e.untrained, o.untrained)
}

// WearsUntrainedBodyArmor reports whether a Light, Medium or Heavy armor is
// worn without training; an untrained Shield alone gives false.
func (e ArmorTrainingEffects) WearsUntrainedBodyArmor() bool {
	return slices.ContainsFunc(e.untrained, func(c content.ArmorCategory) bool { return c != content.ArmorShield })
}

// RollCircumstancesFor returns exactly one [modifier.Disadvantage] when body
// armor is worn untrained and involved holds Strength or Dexterity (SRD 5.2.1,
// "Armor Training", p. 177: any D20 Test that involves Strength or
// Dexterity), otherwise nil. An untrained Shield alone gives nil. Dexterity
// covers Initiative.
func (e ArmorTrainingEffects) RollCircumstancesFor(involved []ability.Ability) []modifier.RollCircumstance {
	if e.WearsUntrainedBodyArmor() && (slices.Contains(involved, ability.Strength) || slices.Contains(involved, ability.Dexterity)) {
		return []modifier.RollCircumstance{modifier.Disadvantage{}}
	}
	return nil
}
