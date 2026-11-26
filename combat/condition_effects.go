package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ConditionEffects answers the questions the rules ask of a ConditionSet: what a
// condition does to a roll, a Speed, a target or an action. It is an empty struct,
// so the zero value is usable and a consumer can declare a small interface over
// the queries it uses. Every query is pure and deterministic, reads no clock,
// randomness or I/O, never changes its argument and returns copies. A query that
// takes a distance or a Speed returns an error matching srderr.ErrInvalidInput for
// a negative one and the zero value with it; no query panics. Safe for
// concurrent use.
type ConditionEffects struct{}

// ConditionAttackFacts are the facts about one attack roll the conditions read.
// Construct it with a keyed literal. Only SightRelationCanSee and LineOfSightClear
// count as seeing and as within line of sight; any other value counts as not.
type ConditionAttackFacts struct {
	// AttackerID and TargetID name the two creatures.
	AttackerID, TargetID CombatantID
	// DistanceFeet is the distance between them in whole feet, 0 or more.
	DistanceFeet int
	// AttackerSeesTarget and TargetSeesAttacker say whether either can somehow see the other.
	AttackerSeesTarget, TargetSeesAttacker SightRelation
	// AttackerFearSource says whether the attacker's source of fear is within line of sight;
	// it is read only when the attacker is Frightened.
	AttackerFearSource LineOfSight
}

// EffectiveSpeed is a Speed after the conditions. Construct it with a keyed literal.
type EffectiveSpeed struct {
	// Value is the Speed in feet, never below 0.
	Value int
	// Increase says whether a Speed of 0 can be raised.
	Increase SpeedIncrease
}

// ProneMovement is what a Prone creature can do with movement. Variants:
// CrawlOnly, CrawlOrRightItself.
type ProneMovement interface{ proneMovement() }

// CrawlOnly: the creature can only crawl, because its Speed is 0.
type CrawlOnly struct{}

// CrawlOrRightItself: the creature can crawl or right itself by spending CostFeet
// feet of movement, half its Speed rounded down. Construct with a keyed literal.
type CrawlOrRightItself struct{ CostFeet int }

func (CrawlOnly) proneMovement()          {}
func (CrawlOrRightItself) proneMovement() {}

// ConditionFlags are the facts the conditions report about a creature.
type ConditionFlags struct {
	Actions       ConditionActionAvailability
	Concentration ConditionConcentration
	Speech        Speech
	HeldItems     HeldItems
	Awareness     Awareness
	Resistance    AllDamageResistance
}

// nearBy is the distance in feet that counts as "within 5 feet" and below.
const nearBy = 5

// holds reports whether an effect gated by when applies to an attack roll.
// own is true for the effects of the attacker's conditions.
func (s ConditionSet) holds(when EffectCondition, own bool, f ConditionAttackFacts) bool {
	switch when {
	case EffectConditionAlways:
		return true
	case EffectConditionAttackerWithin5Feet:
		return !own && f.DistanceFeet <= nearBy
	case EffectConditionAttackerBeyond5Feet:
		return !own && f.DistanceFeet > nearBy
	case EffectConditionSourceInLineOfSight:
		return own && f.AttackerFearSource == LineOfSightClear
	case EffectConditionTargetIsNotGrappler:
		g, _ := s.slots[content.ConditionGrappled].(Grappled)
		return own && g.Grappler != f.TargetID
	case EffectConditionUnlessSeen:
		if own {
			return f.TargetSeesAttacker != SightRelationCanSee
		}
		return f.AttackerSeesTarget != SightRelationCanSee
	}
	return false
}

// AttackRoll returns the Advantage and Disadvantage the conditions of both
// creatures give an attack roll: the attacker's own and those against the target.
// It returns an error for a negative DistanceFeet.
func (ConditionEffects) AttackRoll(attacker, target ConditionSet, facts ConditionAttackFacts) (RollContributions, error) {
	if facts.DistanceFeet < 0 {
		return RollContributions{}, invalidInput("negative distance")
	}
	var out []RollContribution
	attacker.each(func(k content.ConditionKind, e ConditionEffect) {
		if o, ok := e.(OwnAttackRolls); ok && attacker.holds(o.When, true, facts) {
			out = append(out, RollContribution{k, o.Circumstance})
		}
	})
	target.each(func(k content.ConditionKind, e ConditionEffect) {
		if a, ok := e.(AttackRollsAgainst); ok && target.holds(a.When, false, facts) {
			out = append(out, RollContribution{k, a.Circumstance})
		}
	})
	return newRollContributions(out), nil
}

// CriticalHit says whether a hit on a creature with these conditions is a Critical
// Hit by them: Paralyzed and Unconscious make it one from within 5 feet. It
// returns an error for a negative attackerDistanceFeet.
func (ConditionEffects) CriticalHit(target ConditionSet, attackerDistanceFeet int) (AutomaticCritical, error) {
	if attackerDistanceFeet < 0 {
		return 0, invalidInput("negative distance")
	}
	res := AutomaticCriticalNotAutomatic
	target.each(func(_ content.ConditionKind, e ConditionEffect) {
		if c, ok := e.(AutomaticCriticalHit); ok && attackerDistanceFeet <= c.WithinFeet {
			res = AutomaticCriticalAutomatic
		}
	})
	return res, nil
}

// AbilityCheck returns the Disadvantage the conditions give the creature's own
// ability checks. fearSource says whether its source of fear is in line of sight.
func (ConditionEffects) AbilityCheck(self ConditionSet, fearSource LineOfSight) RollContributions {
	var out []RollContribution
	self.each(func(k content.ConditionKind, e ConditionEffect) {
		if c, ok := e.(OwnAbilityChecks); ok &&
			(c.When == EffectConditionAlways || c.When == EffectConditionSourceInLineOfSight && fearSource == LineOfSightClear) {
			out = append(out, RollContribution{k, c.Circumstance})
		}
	})
	return newRollContributions(out)
}

// AbilityCheckEffect says whether the creature automatically fails an ability check
// that requires a sense.
func (ConditionEffects) AbilityCheckEffect(self ConditionSet, requires CheckRequirement) AbilityCheckEffect {
	res := AbilityCheckEffectUnaffected
	self.each(func(_ content.ConditionKind, e ConditionEffect) {
		if a, ok := e.(AutoFailAbilityChecks); ok &&
			(a.Requiring == SenseSight && requires == CheckRequirementSight ||
				a.Requiring == SenseHearing && requires == CheckRequirementHearing) {
			res = AbilityCheckEffectAutomaticFailure
		}
	})
	return res
}

// SocialCheck returns the Advantage the creature that charmed charmed has on an
// ability check to interact socially with it, and nothing for any other creature
// or for an item charmer.
func (ConditionEffects) SocialCheck(charmed ConditionSet, interactingCreature CombatantID) RollContributions {
	var out []RollContribution
	by, _ := charmed.slots[content.ConditionCharmed].(Charmed)
	charmer, isCreature := by.Charmer.(CharmedByCombatant)
	charmed.each(func(k content.ConditionKind, e ConditionEffect) {
		if _, ok := e.(SocialAdvantageForSource); ok && isCreature && charmer.ID == interactingCreature {
			out = append(out, RollContribution{k, modifier.Advantage{}})
		}
	})
	return newRollContributions(out)
}

// SavingThrow returns what the conditions do to a saving throw with ability a;
// automatic failure takes precedence over Disadvantage.
func (ConditionEffects) SavingThrow(self ConditionSet, a ability.Ability) SavingThrowEffect {
	res := SavingThrowEffectUnaffected
	self.each(func(_ content.ConditionKind, e ConditionEffect) {
		switch v := e.(type) {
		case SavingThrowsAutoFail:
			for _, f := range v.Abilities {
				if f == a {
					res = SavingThrowEffectAutomaticFailure
				}
			}
		case SavingThrowDisadvantage:
			if v.Ability == a && res == SavingThrowEffectUnaffected {
				res = SavingThrowEffectDisadvantage
			}
		}
	})
	return res
}

// Initiative returns the circumstances the conditions give an Initiative roll:
// Disadvantage when Incapacitated, Advantage when Invisible.
func (ConditionEffects) Initiative(self ConditionSet) RollContributions {
	var out []RollContribution
	self.each(func(k content.ConditionKind, e ConditionEffect) {
		if c, ok := e.(InitiativeCircumstance); ok {
			out = append(out, RollContribution{k, c.Circumstance})
		}
	})
	return newRollContributions(out)
}

// Speed returns one Speed of baseSpeedFeet after the conditions: 0 when any
// condition sets Speed to 0, else the Speed less 5 feet per Exhaustion level,
// never below 0. Apply it to each Speed a creature has. It returns an error for a
// negative baseSpeedFeet.
func (ConditionEffects) Speed(self ConditionSet, baseSpeedFeet int) (EffectiveSpeed, error) {
	if baseSpeedFeet < 0 {
		return EffectiveSpeed{}, invalidInput("negative speed")
	}
	zero, reduction, level := false, 0, self.effectiveExhaustionLevel()
	self.each(func(_ content.ConditionKind, e ConditionEffect) {
		switch v := e.(type) {
		case SpeedZero:
			zero = true
		case SpeedReducedBy:
			reduction += v.FeetPerLevel * level
		}
	})
	if zero {
		return EffectiveSpeed{0, SpeedIncreaseForbidden}, nil
	}
	return EffectiveSpeed{max(baseSpeedFeet-reduction, 0), SpeedIncreaseAllowed}, nil
}

// StandUp says what a Prone creature with the base Speed can do: crawl, or also
// right itself for half its effective Speed rounded down. It returns an error for
// a negative baseSpeedFeet.
func (c ConditionEffects) StandUp(self ConditionSet, baseSpeedFeet int) (ProneMovement, error) {
	sp, err := c.Speed(self, baseSpeedFeet)
	if err != nil {
		return nil, err
	}
	if sp.Value == 0 {
		return CrawlOnly{}, nil
	}
	return CrawlOrRightItself{sp.Value / 2}, nil
}

// CanTarget says whether the actor may attack target or target it with an
// effect: a creature charmed by a creature may not attack that charmer or target it
// with a damaging effect. Any other target, a non-damaging effect and an item
// charmer are allowed.
func (ConditionEffects) CanTarget(actor ConditionSet, target CombatantID, damage EffectDamage) ConditionPermission {
	res := ConditionPermissionAllowed
	by, _ := actor.slots[content.ConditionCharmed].(Charmed)
	charmer, isCreature := by.Charmer.(CharmedByCombatant)
	actor.each(func(_ content.ConditionKind, e ConditionEffect) {
		if _, ok := e.(CharmerRestriction); ok && isCreature && charmer.ID == target && damage == EffectDamageDamaging {
			res = ConditionPermissionForbidden
		}
	})
	return res
}

// IsValidTarget says whether a creature with these conditions can be the target of
// an effect that needs its target seen: an Invisible one cannot unless the
// effect's creator can somehow see it.
func (ConditionEffects) IsValidTarget(target ConditionSet, effectCreatorSeesTarget SightRelation) ConditionPermission {
	res := ConditionPermissionAllowed
	target.each(func(_ content.ConditionKind, e ConditionEffect) {
		if _, ok := e.(Concealed); ok && effectCreatorSeesTarget != SightRelationCanSee {
			res = ConditionPermissionForbidden
		}
	})
	return res
}

// CanWillinglyMoveCloserTo says whether the creature may move closer to other: not
// when other is its source of fear.
func (ConditionEffects) CanWillinglyMoveCloserTo(self ConditionSet, other CombatantID) ConditionPermission {
	res := ConditionPermissionAllowed
	f, _ := self.slots[content.ConditionFrightened].(Frightened)
	self.each(func(_ content.ConditionKind, e ConditionEffect) {
		if _, ok := e.(CannotApproachSource); ok && f.Source == other {
			res = ConditionPermissionForbidden
		}
	})
	return res
}

// D20TestReduction returns how much Exhaustion lowers the total of a D20 Test: 2
// per level, 0 without Exhaustion.
func (ConditionEffects) D20TestReduction(self ConditionSet) int {
	total, level := 0, self.effectiveExhaustionLevel()
	self.each(func(_ content.ConditionKind, e ConditionEffect) {
		if r, ok := e.(D20TestReducedBy); ok {
			total += r.PerLevel * level
		}
	})
	return total
}

// IsIncapacitated reports whether Incapacitated holds, directly or by implication.
func (ConditionEffects) IsIncapacitated(self ConditionSet) bool {
	return self.effectiveMask()&bit(content.ConditionIncapacitated) != 0
}

// IsDeadFromExhaustion reports whether the creature's Exhaustion level is 6; the
// host then records the death with RecordExhaustionDeath.
func (ConditionEffects) IsDeadFromExhaustion(self ConditionSet) bool {
	return self.effectiveExhaustionLevel() == maxExhaustionLevel
}

// ActiveFlags returns the facts the conditions report about the creature; an empty
// set gives the neutral value of each.
func (ConditionEffects) ActiveFlags(self ConditionSet) ConditionFlags {
	f := ConditionFlags{ConditionActionAvailabilityAvailable, ConditionConcentrationIntact, SpeechAble,
		HeldItemsKept, AwarenessAware, AllDamageResistanceNone}
	self.each(func(_ content.ConditionKind, e ConditionEffect) {
		switch e.(type) {
		case NoActions:
			f.Actions = ConditionActionAvailabilityNone
		case ConcentrationBroken:
			f.Concentration = ConditionConcentrationBroken
		case CannotSpeak:
			f.Speech = SpeechUnable
		case DropsHeldItems:
			f.HeldItems = HeldItemsDropped
		case Unaware:
			f.Awareness = AwarenessUnaware
		case DamageResistanceAll:
			f.Resistance = AllDamageResistanceAll
		}
	})
	return f
}
