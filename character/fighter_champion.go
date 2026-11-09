package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Champion feature levels (SRD 5.2.1, "Fighter Subclass: Champion", p. 49).
const (
	championSubclassLevel = 3
	championStyleLevel    = 7
	championWarriorLevel  = 10
	championSuperiorLevel = 15
	championSurvivorLevel = 18
)

// ChampionAttackKind is the kind of attack roll a critical-hit threshold is asked
// for. A caller maps its own attack kind to it. The zero value is not a kind.
type ChampionAttackKind int

// The attack kinds.
const (
	// ChampionWeaponAttack: an attack roll with a weapon.
	ChampionWeaponAttack ChampionAttackKind = iota + 1
	// ChampionUnarmedStrike: an attack roll with an Unarmed Strike.
	ChampionUnarmedStrike
	// ChampionOtherAttack: an attack roll that is neither (a spell attack, for one).
	ChampionOtherAttack
)

// String returns "weapon", "unarmed_strike", "other" or "unknown".
func (k ChampionAttackKind) String() string {
	switch k {
	case ChampionWeaponAttack:
		return "weapon"
	case ChampionUnarmedStrike:
		return "unarmed_strike"
	case ChampionOtherAttack:
		return "other"
	}
	return "unknown"
}

// ChampionAdvantageSource is a declared source of Advantage the Champion has, a
// closed set. Variants: ChampionInitiativeAdvantage, ChampionSkillCheckAdvantage.
type ChampionAdvantageSource interface {
	// Circumstance returns the roll circumstance the source gives: Advantage.
	Circumstance() modifier.RollCircumstance
	isChampionAdvantageSource()
}

// ChampionInitiativeAdvantage: Advantage on Initiative rolls.
type ChampionInitiativeAdvantage struct{}

// ChampionSkillCheckAdvantage: Advantage on checks of Skill with Ability. Construct
// it with a keyed literal.
type ChampionSkillCheckAdvantage struct {
	Skill   content.Skill
	Ability ability.Ability
}

// Circumstance returns modifier.Advantage.
func (ChampionInitiativeAdvantage) Circumstance() modifier.RollCircumstance {
	return modifier.Advantage{}
}

// Circumstance returns modifier.Advantage.
func (ChampionSkillCheckAdvantage) Circumstance() modifier.RollCircumstance {
	return modifier.Advantage{}
}

func (ChampionInitiativeAdvantage) isChampionAdvantageSource() {}
func (ChampionSkillCheckAdvantage) isChampionAdvantageSource() {}

// DefyDeath is the Champion's Defy Death, applied to a Death Saving Throw (SRD
// 5.2.1, "Level 18: Survivor", p. 49): Advantage on the save, and a natural 18, 19
// or 20 counts as a 20. It is a value; the zero DefyDeath is the plain save.
type DefyDeath struct {
	advantage       bool
	natural, counts int
}

// AdvantageOnDeathSaves reports whether the Death Saving Throw has Advantage.
func (d DefyDeath) AdvantageOnDeathSaves() bool { return d.advantage }

// Natural returns the natural roll the report was made for.
func (d DefyDeath) Natural() int { return d.natural }

// CountsAs returns the roll the natural roll counts as: 20 for a natural 18 to 20
// with Defy Death, the natural roll otherwise.
func (d DefyDeath) CountsAs() int { return d.counts }

// RegainsOneHitPoint reports whether the roll has the benefit of a 20: the
// character regains 1 Hit Point (SRD 5.2.1, "Death Saving Throws", p. 18). It does
// not change current Hit Points.
func (d DefyDeath) RegainsOneHitPoint() bool { return d.counts == 20 }

// ChampionEffects reports the declared effects of the Champion subclass for a
// Fighter, which other contexts apply. Every query takes the chosen subclass (empty
// for none) and the Fighter level; a Fighter without the Champion has none of them.
// It is stateless; the zero value is ready to use. Safe for concurrent use.
type ChampionEffects struct{}

func isChampion(subclass content.SubclassID) bool { return subclass == ChampionSubclassID }

// CriticalHitThreshold returns the lowest natural d20 roll that is a Critical Hit
// on the attack roll (SRD 5.2.1, "Level 3: Improved Critical", "Level 15: Superior
// Critical", p. 49): 19 for a Champion of Fighter level 3 to 14 on a weapon attack
// roll or Unarmed Strike, 18 from level 15, and 20 otherwise (any other subclass,
// or an attack roll of another kind). It only returns the threshold: it takes no
// roll, and a natural 20 always remains a Critical Hit.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid kind and for
// the Champion at Fighter level below 3 (the subclass is chosen at level 3). It
// never panics.
func (ChampionEffects) CriticalHitThreshold(subclass content.SubclassID, level FighterLevel, kind ChampionAttackKind) (int, error) {
	if kind < ChampionWeaponAttack || kind > ChampionOtherAttack {
		return 0, invalidInput("critical hit threshold for an invalid attack kind " + strconv.Itoa(int(kind)))
	}
	if !isChampion(subclass) {
		return 20, nil
	}
	if level.n < championSubclassLevel {
		return 0, invalidInput("Champion at Fighter level below 3")
	}
	switch {
	case kind == ChampionOtherAttack:
		return 20, nil
	case level.n >= championSuperiorLevel:
		return 18, nil
	}
	return 19, nil
}

// AdvantageSources returns the Advantage the Champion has from Fighter level 3
// (SRD 5.2.1, "Level 3: Remarkable Athlete", p. 49): on Initiative rolls and on
// Strength (Athletics) checks, as a new slice; empty below level 3 or without the
// Champion.
func (ChampionEffects) AdvantageSources(subclass content.SubclassID, level FighterLevel) []ChampionAdvantageSource {
	if !isChampion(subclass) || level.n < championSubclassLevel {
		return []ChampionAdvantageSource{}
	}
	return []ChampionAdvantageSource{ChampionInitiativeAdvantage{}, ChampionSkillCheckAdvantage{Skill: content.SkillAthletics, Ability: ability.Strength}}
}

// CriticalHitMove reports the move after a Critical Hit: TacticalShiftOffered from
// Champion level 3 (up to half Speed without provoking Opportunity Attacks, SRD
// 5.2.1, "Level 3: Remarkable Athlete", p. 49), NoTacticalShift otherwise.
func (ChampionEffects) CriticalHitMove(subclass content.SubclassID, level FighterLevel) TacticalShiftOffer {
	if isChampion(subclass) && level.n >= championSubclassLevel {
		return TacticalShiftOffered{}
	}
	return NoTacticalShift{}
}

// HasSurvivor reports whether the character has the Survivor features, Defy Death
// and Heroic Rally: a Champion of Fighter level 18 or more.
func (ChampionEffects) HasSurvivor(subclass content.SubclassID, level FighterLevel) bool {
	return isChampion(subclass) && level.n >= championSurvivorLevel
}

// DefyDeath reports how Defy Death changes a Death Saving Throw whose natural roll
// is natural: with Survivor, Advantage and a natural 18 to 20 counting as 20;
// without it, the plain roll. It returns an error matching srderr.ErrInvalidInput
// for a natural outside 1 to 20. It never panics.
func (c ChampionEffects) DefyDeath(subclass content.SubclassID, level FighterLevel, natural int) (DefyDeath, error) {
	if natural < 1 || natural > 20 {
		return DefyDeath{}, invalidInput("Death Saving Throw natural roll " + strconv.Itoa(natural) + " outside 1 to 20")
	}
	if !c.HasSurvivor(subclass, level) {
		return DefyDeath{natural: natural, counts: natural}, nil
	}
	counts := natural
	if natural >= 18 {
		counts = 20
	}
	return DefyDeath{advantage: true, natural: natural, counts: counts}, nil
}

// HeroicRally reports the Heroic Rally of a character with Survivor, and false
// when it has none (below Fighter level 18 or without the Champion). The effect
// fires at the start of each of the character's turns, which is the consumer's;
// the report is the per-trigger calculation of HeroicRallyHealing.Healing. It
// returns the errors of HeroicRallyHealing.Healing.
func (c ChampionEffects) HeroicRally(subclass content.SubclassID, level FighterLevel, constitution ability.Modifier, currentHitPoints int, maximum HitPointMaximum) (HeroicRallyOutcome, bool, error) {
	if !c.HasSurvivor(subclass, level) {
		return nil, false, nil
	}
	out, err := HeroicRallyHealing{}.Healing(constitution, currentHitPoints, maximum)
	return out, err == nil, err
}

// FeaturesAt returns the features of the subclass at or below the Fighter level,
// from its SubclassDefinition (Champion: Improved Critical and Remarkable Athlete
// at 3, Additional Fighting Style at 7, Heroic Warrior at 10, Superior Critical at
// 15, Survivor at 18). It returns the error of
// content.SubclassDefinition.FeaturesAtOrBelow.
func (ChampionEffects) FeaturesAt(def content.SubclassDefinition, level FighterLevel) ([]content.FeatureEntry, error) {
	return def.FeaturesAtOrBelow(level.n)
}

// HeroicWarriorOutcome is the closed result of HeroicWarrior.Evaluate. Variants:
// HeroicWarriorGive, HeroicWarriorNotAvailable.
type HeroicWarriorOutcome interface{ isHeroicWarriorOutcome() }

// HeroicWarriorGive: the Champion may give itself Heroic Inspiration. The host
// calls the Heroic Inspiration rules to gain it. The effect fires at the start of
// each of the character's turns, which is the consumer's.
type HeroicWarriorGive struct{}

// Trigger returns TriggerStartOfEachOfYourTurns.
func (HeroicWarriorGive) Trigger() FighterEffectTrigger { return TriggerStartOfEachOfYourTurns }

// HeroicWarriorNotAvailable: Heroic Warrior does not apply. Construct it with a keyed literal.
type HeroicWarriorNotAvailable struct{ Reason FighterReason }

func (HeroicWarriorGive) isHeroicWarriorOutcome()         {}
func (HeroicWarriorNotAvailable) isHeroicWarriorOutcome() {}

// FighterCombatPresence says whether the character is in combat. The zero value is not a value.
type FighterCombatPresence int

// The combat presences.
const (
	// FighterInCombat: during combat.
	FighterInCombat FighterCombatPresence = iota + 1
	// FighterOutOfCombat: outside combat.
	FighterOutOfCombat
)

// String returns "in_combat", "not_in_combat" or "unknown".
func (p FighterCombatPresence) String() string {
	switch p {
	case FighterInCombat:
		return "in_combat"
	case FighterOutOfCombat:
		return "not_in_combat"
	}
	return "unknown"
}

// HeroicWarrior decides the Champion's Heroic Warrior (SRD 5.2.1, "Level 10: Heroic
// Warrior", p. 49). It is stateless and pure; the zero value is ready to use. Safe
// for concurrent use.
type HeroicWarrior struct{}

// Evaluate returns HeroicWarriorGive for a Champion of Fighter level 10 or more who
// is NotInspired and in combat. Otherwise the first reason that holds, in order:
// FighterFeatureNotGained (below level 10 or without the Champion),
// FighterNotInCombat (outside combat, or an invalid presence), FighterAlreadyHasInspiration.
func (HeroicWarrior) Evaluate(subclass content.SubclassID, level FighterLevel, inspiration HeroicInspiration, presence FighterCombatPresence) HeroicWarriorOutcome {
	switch {
	case !isChampion(subclass) || level.n < championWarriorLevel:
		return HeroicWarriorNotAvailable{Reason: FighterFeatureNotGained}
	case presence != FighterInCombat:
		return HeroicWarriorNotAvailable{Reason: FighterNotInCombat}
	case inspiration != NotInspired:
		return HeroicWarriorNotAvailable{Reason: FighterAlreadyHasInspiration}
	}
	return HeroicWarriorGive{}
}

// HeroicRallyOutcome is the closed result of HeroicRallyHealing.Healing. Variants:
// HeroicRallyRegains, HeroicRallyNotEligible.
type HeroicRallyOutcome interface{ isHeroicRallyOutcome() }

// HeroicRallyRegains: the character regains HitPoints Hit Points, uncapped (the
// consumer caps at the maximum).
type HeroicRallyRegains struct{ hitPoints int }

// HitPoints returns 5 plus the Constitution modifier, at least 0.
func (r HeroicRallyRegains) HitPoints() int { return r.hitPoints }

// Trigger returns TriggerStartOfEachOfYourTurns.
func (HeroicRallyRegains) Trigger() FighterEffectTrigger { return TriggerStartOfEachOfYourTurns }

// HeroicRallyNotEligible: nothing is regained. Construct it with a keyed literal.
type HeroicRallyNotEligible struct{ Reason FighterReason }

func (HeroicRallyRegains) isHeroicRallyOutcome()     {}
func (HeroicRallyNotEligible) isHeroicRallyOutcome() {}

// HeroicRallyHealing calculates the Champion's Heroic Rally (SRD 5.2.1, "Level 18:
// Survivor", p. 49). It holds no turn state. It is stateless and pure; the zero
// value is ready to use. Safe for concurrent use.
type HeroicRallyHealing struct{}

// Healing returns HeroicRallyRegains with 5 plus the Constitution modifier (never
// below 0) when the character is Bloodied (IsBloodied) and has at least 1 Hit
// Point, and otherwise HeroicRallyNotEligible with FighterNoHitPoints at 0 Hit
// Points or FighterNotBloodied.
//
// It returns an error matching srderr.ErrInvalidInput for a negative
// currentHitPoints, one above the maximum or the zero maximum. It never panics.
func (HeroicRallyHealing) Healing(constitution ability.Modifier, currentHitPoints int, maximum HitPointMaximum) (HeroicRallyOutcome, error) {
	if currentHitPoints > maximum.Value() {
		return nil, invalidInput("current Hit Points above the Hit Point maximum")
	}
	bloodied, err := IsBloodied(currentHitPoints, maximum)
	if err != nil {
		return nil, err
	}
	switch {
	case currentHitPoints == 0:
		return HeroicRallyNotEligible{Reason: FighterNoHitPoints}, nil
	case !bloodied:
		return HeroicRallyNotEligible{Reason: FighterNotBloodied}, nil
	}
	return HeroicRallyRegains{hitPoints: max(5+constitution.Value(), 0)}, nil
}
