package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// AttackInputs are the inputs the conditions add to one attack roll (combat-action-resolution.md FR-027 to
// FR-033): the circumstances, the modifiers, the automatic Critical Hit and whether the attacker may target
// the creature. The zero value adds nothing and allows the target.
type AttackInputs struct {
	circumstances     []modifier.RollCircumstance
	modifiers         []modifier.Modifier
	automaticCritical AutomaticCritical
	forbidden         bool
}

// Circumstances returns a copy of the Advantage and Disadvantage the conditions and Dodge give the roll,
// individually and in condition-kind order, Dodge last, so the kernel's one combination rule resolves them.
func (i AttackInputs) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(i.circumstances)
}

// Modifiers returns a copy of the modifiers: the Exhaustion reduction of the attacker, when it has one.
func (i AttackInputs) Modifiers() []modifier.Modifier { return slices.Clone(i.modifiers) }

// AutomaticCritical says whether a hit is a Critical Hit by the target's conditions.
func (i AttackInputs) AutomaticCritical() AutomaticCritical { return i.automaticCritical }

// TargetingForbidden reports whether the attacker may not attack the target (Charmed by it).
func (i AttackInputs) TargetingForbidden() bool { return i.forbidden }

// Additions returns the inputs as the Additions of ResolveAttackCommand.WithAdditions.
func (i AttackInputs) Additions() Additions {
	return Additions{Circumstances: i.Circumstances(), Modifiers: i.Modifiers(), AutomaticCritical: i.automaticCritical}
}

// SaveAdjustments are what the target's conditions do to its saving throw (combat-action-resolution.md
// FR-045): a failure without a die, circumstances and modifiers. The use case folds them into the
// SaveAttempt of the command; the attempt never enters a domain service. The zero value adjusts nothing.
type SaveAdjustments struct {
	automaticFailure bool
	circumstances    []modifier.RollCircumstance
	modifiers        []modifier.Modifier
}

// Forced returns d20.Automatic and true when the conditions make the save fail without a die.
func (s SaveAdjustments) Forced() (d20.ForcedFailureReason, bool) {
	return d20.Automatic, s.automaticFailure
}

// Circumstances returns a copy of the circumstances to add to a rolled save.
func (s SaveAdjustments) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(s.circumstances)
}

// Modifiers returns a copy of the modifiers to add to a rolled save.
func (s SaveAdjustments) Modifiers() []modifier.Modifier { return slices.Clone(s.modifiers) }

// AttackConditionInputs asks ConditionEffects every question the conditions answer about an attack or
// a saving throw it causes, and states the answer as inputs (combat-action-resolution.md FR-006, FR-027
// to FR-035, FR-045, FR-125). It is the only code that asks for attacks and saves, and takes no position,
// allegiance or distance but the fields of an AttackScene. It is pure and stateless; the zero value is
// usable.
type AttackConditionInputs struct{}

// dodgeHolds reports whether Dodge benefits apply to the target: the tracker says they are active and its
// conditions do not force its Speed to 0, which the tracker's profile Speed cannot show (FR-125).
func dodgeHolds(target ConditionSet, dodge DodgeBenefits, targetWalkingSpeedFeet int) (bool, error) {
	zero, err := speedForbidden(target, targetWalkingSpeedFeet)
	return dodge == DodgeBenefitsActive && !zero, err
}

// Attack returns the inputs of an attack roll by attacker against target in the scene. dodge is the
// tracker's DodgeBenefits for the target and targetWalkingSpeedFeet the walking Speed of its profile.
// ConditionEffects.AttackRoll is asked once, with both sets, so each contribution appears once. It
// returns an error matching srderr.ErrInvalidInput for a negative Speed.
func (AttackConditionInputs) Attack(attacker, target ConditionSet, attackerID, targetID CombatantID, scene AttackScene,
	dodge DodgeBenefits, targetWalkingSpeedFeet int) (AttackInputs, error) {
	var out AttackInputs
	contributions, err := (ConditionEffects{}).AttackRoll(attacker, target, scene.facts(attackerID, targetID))
	if err != nil {
		return AttackInputs{}, err
	}
	for _, c := range contributions.Items() {
		out.circumstances = append(out.circumstances, c.Circumstance())
	}
	holds, err := dodgeHolds(target, dodge, targetWalkingSpeedFeet)
	if err != nil {
		return AttackInputs{}, err
	}
	if holds && scene.targetSeesAttacker == SightRelationCanSee {
		out.circumstances = append(out.circumstances, modifier.Disadvantage{})
	}
	if n := (ConditionEffects{}).D20TestReduction(attacker); n > 0 {
		out.modifiers = append(out.modifiers, modifier.New(-n))
	}
	if out.automaticCritical, err = (ConditionEffects{}).CriticalHit(target, scene.distanceFeet); err != nil {
		return AttackInputs{}, err
	}
	// The condition forbids attacking the charmer whatever the attack deals (FR-033).
	out.forbidden = (ConditionEffects{}).CanTarget(attacker, targetID, EffectDamageDamaging) == ConditionPermissionForbidden
	return out, nil
}

// Save returns what the conditions of the target do to its saving throw with the ability: automatic
// failure first, else Disadvantage; the Dodge Advantage on a Dexterity save; the Exhaustion reduction.
func (AttackConditionInputs) Save(target ConditionSet, save ability.Ability, dodge DodgeBenefits, targetWalkingSpeedFeet int) (SaveAdjustments, error) {
	var out SaveAdjustments
	switch (ConditionEffects{}).SavingThrow(target, save) {
	case SavingThrowEffectAutomaticFailure:
		out.automaticFailure = true
	case SavingThrowEffectDisadvantage:
		out.circumstances = append(out.circumstances, modifier.Disadvantage{})
	case SavingThrowEffectUnaffected:
	}
	holds, err := dodgeHolds(target, dodge, targetWalkingSpeedFeet)
	if err != nil {
		return SaveAdjustments{}, err
	}
	if holds && save == ability.Dexterity {
		out.circumstances = append(out.circumstances, modifier.Advantage{})
	}
	if n := (ConditionEffects{}).D20TestReduction(target); n > 0 {
		out.modifiers = append(out.modifiers, modifier.New(-n))
	}
	return out, nil
}

// Check returns what the conditions of a creature do to its own ability check: the automatic failure of a
// check that needs a sense the creature lacks, else the Disadvantage of its conditions, and the Exhaustion
// reduction (combat-action-resolution.md FR-073). fearSource says whether the creature's source of fear is
// in line of sight; requires what the check needs of its senses.
func (AttackConditionInputs) Check(self ConditionSet, fearSource LineOfSight, requires CheckRequirement) SaveAdjustments {
	var out SaveAdjustments
	if (ConditionEffects{}).AbilityCheckEffect(self, requires) == AbilityCheckEffectAutomaticFailure {
		out.automaticFailure = true
	}
	for _, c := range (ConditionEffects{}).AbilityCheck(self, fearSource).Items() {
		out.circumstances = append(out.circumstances, c.Circumstance())
	}
	if n := (ConditionEffects{}).D20TestReduction(self); n > 0 {
		out.modifiers = append(out.modifiers, modifier.New(-n))
	}
	return out
}

// ExhaustionReduction returns how much the Exhaustion of the creature lowers the total of any d20 it rolls: a
// Death Saving Throw asks no ability, so only this applies (SRD 5.2.1, "Exhaustion [Condition]", p. 181).
func (AttackConditionInputs) ExhaustionReduction(self ConditionSet) int {
	return (ConditionEffects{}).D20TestReduction(self)
}
