package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The 17 feats of SRD 5.2.1 that have behaviour (pp. 87-88). Their ids are the
// snake_case of the printed names.
const (
	featAlert                     content.FeatID = "alert"
	featSavageAttacker            content.FeatID = "savage_attacker"
	featMagicInitiate             content.FeatID = "magic_initiate"
	featSkilled                   content.FeatID = "skilled"
	featAbilityScoreImprovement   content.FeatID = "ability_score_improvement"
	featGrappler                  content.FeatID = "grappler"
	featArchery                   content.FeatID = "archery"
	featDefense                   content.FeatID = "defense"
	featGreatWeaponFighting       content.FeatID = "great_weapon_fighting"
	featTwoWeaponFighting         content.FeatID = "two_weapon_fighting"
	featBoonOfCombatProwess       content.FeatID = "boon_of_combat_prowess"
	featBoonOfDimensionalTravel   content.FeatID = "boon_of_dimensional_travel"
	featBoonOfFate                content.FeatID = "boon_of_fate"
	featBoonOfIrresistibleOffense content.FeatID = "boon_of_irresistible_offense"
	featBoonOfNightSpirit         content.FeatID = "boon_of_the_night_spirit"
	featBoonOfSpellRecall         content.FeatID = "boon_of_spell_recall"
	featBoonOfTruesight           content.FeatID = "boon_of_truesight"
)

// choiceShape is the kind of FeatChoices a feat takes (table C).
type choiceShape int

const (
	shapeNone choiceShape = iota + 1
	shapeMagicInitiate
	shapeSkilled
	shapeImprovement
	shapeSingle
)

// featShape is what table C says about a feat: the choices it takes, the
// abilities a single increase may name, the maximum score, and the triggers that
// renew its limited use (none when it has no limited use).
type featShape struct {
	choices   choiceShape
	abilities []ability.Ability
	maximum   int
	renewedBy UseRenewedBy
}

// shapeOf returns the shape of feat id, and false for a feat with no behaviour.
func shapeOf(id content.FeatID) (featShape, bool) {
	all := ability.All()
	strDex := []ability.Ability{ability.Strength, ability.Dexterity}
	switch id {
	case featAlert, featSavageAttacker, featArchery, featDefense, featGreatWeaponFighting, featTwoWeaponFighting:
		return featShape{choices: shapeNone}, true
	case featMagicInitiate:
		return featShape{choices: shapeMagicInitiate, renewedBy: UseRenewedBy{mask: 1 << RenewOnLongRestFinished}}, true
	case featSkilled:
		return featShape{choices: shapeSkilled}, true
	case featAbilityScoreImprovement:
		return featShape{choices: shapeImprovement, maximum: DefaultScoreCap}, true
	case featGrappler:
		return featShape{choices: shapeSingle, abilities: strDex, maximum: DefaultScoreCap}, true
	case featBoonOfCombatProwess, featBoonOfDimensionalTravel, featBoonOfNightSpirit, featBoonOfTruesight:
		return featShape{choices: shapeSingle, abilities: all, maximum: HighestScoreCap}, true
	case featBoonOfFate:
		return featShape{choices: shapeSingle, abilities: all, maximum: HighestScoreCap,
			renewedBy: UseRenewedBy{mask: 1<<RenewOnInitiativeRolled | 1<<RenewOnShortRestFinished | 1<<RenewOnLongRestFinished}}, true
	case featBoonOfIrresistibleOffense:
		return featShape{choices: shapeSingle, abilities: strDex, maximum: HighestScoreCap}, true
	case featBoonOfSpellRecall:
		return featShape{choices: shapeSingle, abilities: []ability.Ability{ability.Intelligence, ability.Wisdom, ability.Charisma}, maximum: HighestScoreCap}, true
	}
	return featShape{}, false
}

// matches reports whether choices has the type the shape takes.
func (s featShape) matches(choices FeatChoices) bool {
	switch choices.(type) {
	case NoFeatChoices:
		return s.choices == shapeNone
	case SingleAbilityIncrease:
		return s.choices == shapeSingle
	case AbilityScoreImprovementChoices:
		return s.choices == shapeImprovement
	case MagicInitiateChoices:
		return s.choices == shapeMagicInitiate
	case SkilledChoices:
		return s.choices == shapeSkilled
	}
	return false
}

// givesIncrease reports whether the feat raises ability scores, which cannot be
// undone.
func (s featShape) givesIncrease() bool {
	return s.choices == shapeImprovement || s.choices == shapeSingle
}
