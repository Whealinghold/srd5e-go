package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ConditionEffect is one rule effect of a condition, as data (SRD 5.2.1, the
// condition entries, pp. 177-191). Variants: AutoFailAbilityChecks,
// AttackRollsAgainst, OwnAttackRolls, OwnAbilityChecks, SavingThrowsAutoFail,
// SavingThrowDisadvantage, SpeedZero, SpeedReducedBy, D20TestReducedBy,
// AutomaticCriticalHit, NoActions, ConcentrationBroken, CannotSpeak,
// InitiativeCircumstance, CharmerRestriction, SocialAdvantageForSource,
// CannotApproachSource, Concealed, DamageResistanceAll, ImmuneTo,
// DropsHeldItems, Unaware. The queries of ConditionEffects interpret them.
type ConditionEffect interface{ conditionEffect() }

// AutoFailAbilityChecks: the creature automatically fails an ability check that
// requires the sense. Construct with a keyed literal.
type AutoFailAbilityChecks struct{ Requiring Sense }

// AttackRollsAgainst: an attack roll against the creature gains the
// circumstance when the condition holds. Construct with a keyed literal.
type AttackRollsAgainst struct {
	Circumstance modifier.RollCircumstance
	When         EffectCondition
}

// OwnAttackRolls: the creature's own attack rolls gain the circumstance when the
// condition holds. Construct with a keyed literal.
type OwnAttackRolls struct {
	Circumstance modifier.RollCircumstance
	When         EffectCondition
}

// OwnAbilityChecks: the creature's own ability checks gain the circumstance when
// the condition holds. Construct with a keyed literal.
type OwnAbilityChecks struct {
	Circumstance modifier.RollCircumstance
	When         EffectCondition
}

// SavingThrowsAutoFail: the creature automatically fails saving throws with these
// abilities. It holds a slice, so it is not comparable with ==. Construct with a
// keyed literal.
type SavingThrowsAutoFail struct{ Abilities []ability.Ability }

// SavingThrowDisadvantage: the creature has Disadvantage on saving throws with
// the ability. Construct with a keyed literal.
type SavingThrowDisadvantage struct{ Ability ability.Ability }

// SpeedZero: the creature's Speed is 0 and cannot be increased.
type SpeedZero struct{}

// SpeedReducedBy: Speed is reduced by FeetPerLevel feet per Exhaustion level.
// Construct with a keyed literal.
type SpeedReducedBy struct{ FeetPerLevel int }

// D20TestReducedBy: the total of a D20 Test is reduced by PerLevel per
// Exhaustion level. Construct with a keyed literal.
type D20TestReducedBy struct{ PerLevel int }

// AutomaticCriticalHit: a hit is a Critical Hit when the attacker is within
// WithinFeet feet. Construct with a keyed literal.
type AutomaticCriticalHit struct{ WithinFeet int }

// NoActions: the creature cannot take actions, Bonus Actions or Reactions.
type NoActions struct{}

// ConcentrationBroken: the creature's Concentration is reported broken.
type ConcentrationBroken struct{}

// CannotSpeak: the creature cannot speak.
type CannotSpeak struct{}

// InitiativeCircumstance: the creature's Initiative roll gains the circumstance.
// Construct with a keyed literal.
type InitiativeCircumstance struct{ Circumstance modifier.RollCircumstance }

// CharmerRestriction: the creature cannot attack its charmer or target it with a
// damaging ability or magical effect.
type CharmerRestriction struct{}

// SocialAdvantageForSource: the charmer has Advantage on an ability check to
// interact socially with the creature.
type SocialAdvantageForSource struct{}

// CannotApproachSource: the creature cannot willingly move closer to its source
// of fear.
type CannotApproachSource struct{}

// Concealed: the creature is not a valid target of an effect that requires its
// creator to see it, unless the creator can somehow see it.
type Concealed struct{}

// DamageResistanceAll: the creature has Resistance to all damage.
type DamageResistanceAll struct{}

// ImmuneTo: the creature is immune to the condition. Construct with a keyed
// literal.
type ImmuneTo struct{ Kind content.ConditionKind }

// DropsHeldItems: the creature drops what it is holding.
type DropsHeldItems struct{}

// Unaware: the creature is unaware of its surroundings.
type Unaware struct{}

func (AutoFailAbilityChecks) conditionEffect()    {}
func (AttackRollsAgainst) conditionEffect()       {}
func (OwnAttackRolls) conditionEffect()           {}
func (OwnAbilityChecks) conditionEffect()         {}
func (SavingThrowsAutoFail) conditionEffect()     {}
func (SavingThrowDisadvantage) conditionEffect()  {}
func (SpeedZero) conditionEffect()                {}
func (SpeedReducedBy) conditionEffect()           {}
func (D20TestReducedBy) conditionEffect()         {}
func (AutomaticCriticalHit) conditionEffect()     {}
func (NoActions) conditionEffect()                {}
func (ConcentrationBroken) conditionEffect()      {}
func (CannotSpeak) conditionEffect()              {}
func (InitiativeCircumstance) conditionEffect()   {}
func (CharmerRestriction) conditionEffect()       {}
func (SocialAdvantageForSource) conditionEffect() {}
func (CannotApproachSource) conditionEffect()     {}
func (Concealed) conditionEffect()                {}
func (DamageResistanceAll) conditionEffect()      {}
func (ImmuneTo) conditionEffect()                 {}
func (DropsHeldItems) conditionEffect()           {}
func (Unaware) conditionEffect()                  {}

// ConditionDefinition lists the rules of one condition as data: the kinds it
// implies and its effects. It is the single source of the condition rules. The
// zero value is invalid. Build one with ConditionDefinitionOf.
type ConditionDefinition struct {
	kind    content.ConditionKind
	implies []content.ConditionKind
	effects []ConditionEffect
}

// Kind returns the condition the definition is for.
func (d ConditionDefinition) Kind() content.ConditionKind { return d.kind }

// Implies returns a copy of the kinds that hold whenever this condition does.
func (d ConditionDefinition) Implies() []content.ConditionKind { return slices.Clone(d.implies) }

// Effects returns a copy of the effects, in the order the SRD entry lists them.
func (d ConditionDefinition) Effects() []ConditionEffect { return slices.Clone(d.effects) }

// ConditionDefinitionOf returns the definition of kind. It returns an error
// matching srderr.ErrInvalidInput for an invalid kind. It never panics.
func ConditionDefinitionOf(kind content.ConditionKind) (ConditionDefinition, error) {
	if !kind.Valid() {
		return ConditionDefinition{}, invalidInput("unknown condition kind")
	}
	return definitionOf(kind), nil
}

// strengthAndDexterity is the pair of abilities several conditions fail.
func strengthAndDexterity() []ability.Ability {
	return []ability.Ability{ability.Strength, ability.Dexterity}
}

// definitionOf is the table of SRD 5.2.1 pp. 177-191, one case per condition. A
// kind that is not valid has no effects.
func definitionOf(kind content.ConditionKind) ConditionDefinition {
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	always := EffectConditionAlways
	d := ConditionDefinition{kind: kind}
	switch kind {
	case content.ConditionBlinded:
		d.effects = []ConditionEffect{AutoFailAbilityChecks{SenseSight},
			AttackRollsAgainst{adv, always}, OwnAttackRolls{dis, always}}
	case content.ConditionCharmed:
		d.effects = []ConditionEffect{CharmerRestriction{}, SocialAdvantageForSource{}}
	case content.ConditionDeafened:
		d.effects = []ConditionEffect{AutoFailAbilityChecks{SenseHearing}}
	case content.ConditionExhaustion:
		d.effects = []ConditionEffect{D20TestReducedBy{2}, SpeedReducedBy{5}}
	case content.ConditionFrightened:
		d.effects = []ConditionEffect{OwnAbilityChecks{dis, EffectConditionSourceInLineOfSight},
			OwnAttackRolls{dis, EffectConditionSourceInLineOfSight}, CannotApproachSource{}}
	case content.ConditionGrappled:
		d.effects = []ConditionEffect{SpeedZero{}, OwnAttackRolls{dis, EffectConditionTargetIsNotGrappler}}
	case content.ConditionIncapacitated:
		d.effects = []ConditionEffect{NoActions{}, ConcentrationBroken{}, CannotSpeak{}, InitiativeCircumstance{dis}}
	case content.ConditionInvisible:
		d.effects = []ConditionEffect{InitiativeCircumstance{adv}, Concealed{},
			AttackRollsAgainst{dis, EffectConditionUnlessSeen}, OwnAttackRolls{adv, EffectConditionUnlessSeen}}
	case content.ConditionParalyzed:
		d.implies = []content.ConditionKind{content.ConditionIncapacitated}
		d.effects = []ConditionEffect{SpeedZero{}, SavingThrowsAutoFail{strengthAndDexterity()},
			AttackRollsAgainst{adv, always}, AutomaticCriticalHit{5}}
	case content.ConditionPetrified:
		d.implies = []content.ConditionKind{content.ConditionIncapacitated}
		d.effects = []ConditionEffect{SpeedZero{}, AttackRollsAgainst{adv, always},
			SavingThrowsAutoFail{strengthAndDexterity()}, DamageResistanceAll{}, ImmuneTo{content.ConditionPoisoned}}
	case content.ConditionPoisoned:
		d.effects = []ConditionEffect{OwnAttackRolls{dis, always}, OwnAbilityChecks{dis, always}}
	case content.ConditionProne:
		d.effects = []ConditionEffect{OwnAttackRolls{dis, always},
			AttackRollsAgainst{adv, EffectConditionAttackerWithin5Feet},
			AttackRollsAgainst{dis, EffectConditionAttackerBeyond5Feet}}
	case content.ConditionRestrained:
		d.effects = []ConditionEffect{SpeedZero{}, AttackRollsAgainst{adv, always},
			OwnAttackRolls{dis, always}, SavingThrowDisadvantage{ability.Dexterity}}
	case content.ConditionStunned:
		d.implies = []content.ConditionKind{content.ConditionIncapacitated}
		d.effects = []ConditionEffect{SavingThrowsAutoFail{strengthAndDexterity()}, AttackRollsAgainst{adv, always}}
	case content.ConditionUnconscious:
		d.implies = []content.ConditionKind{content.ConditionIncapacitated, content.ConditionProne}
		d.effects = []ConditionEffect{DropsHeldItems{}, SpeedZero{}, AttackRollsAgainst{adv, always},
			SavingThrowsAutoFail{strengthAndDexterity()}, AutomaticCriticalHit{5}, Unaware{}}
	}
	return d
}
