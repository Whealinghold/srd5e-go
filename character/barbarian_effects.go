package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Fast Movement adds 10 feet of Speed (SRD 5.2.1, "Level 5: Fast Movement", p. 29) and Primal
// Champion raises two scores by 4 up to 25 (SRD 5.2.1, "Level 20: Primal Champion", p. 30).
const (
	fastMovementFeet     = 10
	primalChampionBonus  = 4
	primalChampionMaxima = 25
)

// BarbarianUnarmoredDefenseBasis returns the Armor Class basis Unarmored Defense gives: 10 plus
// Dexterity plus Constitution, usable with a Shield (SRD 5.2.1, "Level 1: Unarmored Defense",
// p. 29). The host passes it to the Armor Class calculation when the character holds the
// feature. The definition declares no basis because content is upstream of character.
func BarbarianUnarmoredDefenseBasis() AbilityBasedArmorClassBasis {
	b, _ := NewAbilityBasedArmorClassBasis("barbarian_unarmored_defense", []ability.Ability{ability.Constitution}, true) // the id and the ability are valid constants
	return b
}

// BarbarianEffects are the effects of the Barbarian features a character holds now, declared
// as data for derived stats and combat to apply (SRD 5.2.1, pp. 28-30). Build it with
// NewBarbarianEffects; the zero value is the effects of a character with no Barbarian levels:
// everything absent. It is a value; the slices and maps its methods return are new.
type BarbarianEffects struct {
	stats    BarbarianLevelStats
	subclass content.SubclassID
	state    BarbarianState
	turn     BarbarianTurnState
	facts    BarbarianHostFacts
}

// NewBarbarianEffects returns the effects for the Barbarian of stats' level and subclass in the
// given rage state, turn state and host facts. Zero stats give no effect at all.
func NewBarbarianEffects(stats BarbarianLevelStats, subclass content.SubclassID, state BarbarianState, turn BarbarianTurnState, facts BarbarianHostFacts) BarbarianEffects {
	return BarbarianEffects{stats: stats, subclass: subclass, state: state, turn: turn, facts: facts}
}

func (e BarbarianEffects) level() int { return e.stats.level.n }

func (e BarbarianEffects) raging() bool { return !e.stats.IsZero() && e.state.rageActive }

// DamageResistances returns the damage types the Barbarian has Resistance to: Bludgeoning,
// Piercing and Slashing while a Rage is active, none otherwise.
func (e BarbarianEffects) DamageResistances() []content.DamageType {
	if !e.raging() {
		return []content.DamageType{}
	}
	return []content.DamageType{content.DamageBludgeoning, content.DamagePiercing, content.DamageSlashing}
}

// RageDamageBonusFor returns the Rage Damage bonus for attack: the bonus of the level while a
// Rage is active and the attack uses Strength with a weapon or an Unarmed Strike and deals
// damage; 0 otherwise.
func (e BarbarianEffects) RageDamageBonusFor(attack BarbarianAttackFacts) int {
	if !e.raging() || !attack.UsesStrength || !attack.DealsDamage || (attack.Kind != BarbarianWeaponAttack && attack.Kind != BarbarianUnarmedStrike) {
		return 0
	}
	return e.stats.rageDamage
}

// StrengthAdvantage reports Advantage on Strength checks and Strength saving throws: true while a Rage is active.
func (e BarbarianEffects) StrengthAdvantage() bool { return e.raging() }

// CannotMaintainConcentration reports that the Barbarian can't maintain Concentration: true while a Rage is active.
func (e BarbarianEffects) CannotMaintainConcentration() bool { return e.raging() }

// CannotCastSpells reports that the Barbarian can't cast spells: true while a Rage is active.
func (e BarbarianEffects) CannotCastSpells() bool { return e.raging() }

// Immunities returns the conditions the Barbarian is immune to: Charmed and Frightened while a
// Rage is active, for the Path of the Berserker from level 6; empty otherwise.
func (e BarbarianEffects) Immunities() map[content.ConditionKind]struct{} {
	out := map[content.ConditionKind]struct{}{}
	if e.raging() && e.level() >= mindlessRageLevel && e.subclass == BerserkerSubclassID {
		out[content.ConditionCharmed] = struct{}{}
		out[content.ConditionFrightened] = struct{}{}
	}
	return out
}

// DangerSenseAdvantage reports Advantage on Dexterity saving throws from level 2, unless the
// Barbarian is Incapacitated.
func (e BarbarianEffects) DangerSenseAdvantage() bool {
	return e.level() >= dangerSenseLevel && !e.facts.Incapacitated
}

// RecklessAdvantageOnStrengthAttacks reports Advantage on attack rolls using Strength for attack
// while Reckless Attack is in force (declared and not yet ended by the next turn).
func (e BarbarianEffects) RecklessAdvantageOnStrengthAttacks(attack BarbarianAttackFacts) bool {
	return e.level() >= recklessAttackLevel && e.turn.recklessDeclared && attack.UsesStrength
}

// AttacksAgainstHaveAdvantage reports that attack rolls against the Barbarian have Advantage:
// true while Reckless Attack is in force.
func (e BarbarianEffects) AttacksAgainstHaveAdvantage() bool {
	return e.level() >= recklessAttackLevel && e.turn.recklessDeclared
}

// StrengthSkillSubstitutions returns the skills whose ability checks may be made as Strength
// checks: Acrobatics, Intimidation, Perception, Stealth and Survival from level 3 while a Rage
// is active; none otherwise.
func (e BarbarianEffects) StrengthSkillSubstitutions() []content.Skill {
	if !e.raging() || e.level() < primalKnowledgeLevel {
		return []content.Skill{}
	}
	return []content.Skill{content.SkillAcrobatics, content.SkillIntimidation, content.SkillPerception, content.SkillStealth, content.SkillSurvival}
}

// AttacksPerAttackAction returns the attacks per Attack action: 2 from level 5, else 1; 0 for no Barbarian levels.
func (e BarbarianEffects) AttacksPerAttackAction() int {
	switch {
	case e.stats.IsZero():
		return 0
	case e.level() >= extraAttackLevel:
		return 2
	}
	return 1
}

// SpeedBonusFeet returns the feet Fast Movement adds to Speed: 10 from level 5 while not
// wearing Heavy armor, else 0.
func (e BarbarianEffects) SpeedBonusFeet() int {
	if e.level() >= extraAttackLevel && !e.facts.WearingHeavyArmor {
		return fastMovementFeet
	}
	return 0
}

// InitiativeAdvantage reports Advantage on Initiative rolls from level 7.
func (e BarbarianEffects) InitiativeAdvantage() bool { return e.level() >= feralInstinctLevel }

// IndomitableMightFloor returns the total to use for a Strength check or Strength saving throw
// whose total is total: the Strength score strength when the total is below it, from level 18.
// Other abilities, other levels and totals at or above the score give total unchanged.
func (e BarbarianEffects) IndomitableMightFloor(usesStrength bool, total, strength int) int {
	if usesStrength && e.level() >= indomitableMightLevel && total < strength {
		return strength
	}
	return total
}

// BarbarianAbilityChange is a new ability score and the maximum it may reach, for the host to
// feed to ChangeAbilityScore. Construct it with a keyed literal.
type BarbarianAbilityChange struct {
	Ability  ability.Ability
	NewScore int
	Maximum  int
}

// PrimalChampion returns the ability changes of Primal Champion (SRD 5.2.1, "Level 20: Primal
// Champion", p. 30): Strength and Constitution each increase by 4 to a maximum of 25, in that
// order. A score already above 25 is not lowered. It returns an error matching
// srderr.ErrInvalidInput for a score outside 1 to 30. It never panics.
func PrimalChampion(strength, constitution ability.Score) ([]BarbarianAbilityChange, error) {
	if strength.Value() < ability.MinScore || constitution.Value() < ability.MinScore {
		return nil, invalidInput("Primal Champion with a zero ability score")
	}
	out := []BarbarianAbilityChange{
		{Ability: ability.Strength, NewScore: primalChampionScore(strength.Value()), Maximum: primalChampionMaxima},
		{Ability: ability.Constitution, NewScore: primalChampionScore(constitution.Value()), Maximum: primalChampionMaxima},
	}
	return out, nil
}

func primalChampionScore(v int) int {
	return max(v, min(v+primalChampionBonus, primalChampionMaxima))
}
