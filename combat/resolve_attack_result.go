package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// ResolveAttackResult is the closed set of outcomes of ResolveAttack.Run.
// Variants: AttackResolved, AttackAutomaticallyMissed, AttackNotPossible.
type ResolveAttackResult interface{ isResolveAttackResult() }

// AttackResolved: the attack roll was made. It carries the roll as d20 resolved it
// and what the final hit, Critical Hit and damage are once a forced miss and an
// automatic Critical Hit are applied. A miss is a normal outcome, not an error.
type AttackResolved struct {
	attack        d20.AttackRollResult
	override      MissOverride
	hit, critical bool
	armorClass    d20.ArmorClass
	effective     d20.ArmorClass
	cover         Cover
	circumstances []modifier.RollCircumstance
	delivery      AttackDelivery
	damage        []Damage
	reveals       bool
}

// Attack returns the attack roll exactly as d20.ResolveAttack returned it, with the
// effective Armor Class as its target.
func (r AttackResolved) Attack() d20.AttackRollResult { return r.attack }

// MissOverride says whether a forced miss overrode the roll.
func (r AttackResolved) MissOverride() MissOverride { return r.override }

// Hit reports the final hit: the roll hit and no forced miss applies.
func (r AttackResolved) Hit() bool { return r.hit }

// CriticalHit reports the final Critical Hit: a hit that is a Critical Hit by the
// roll or by the automatic Critical Hit input.
func (r AttackResolved) CriticalHit() bool { return r.critical }

// ArmorClass returns the target's Armor Class before cover.
func (r AttackResolved) ArmorClass() d20.ArmorClass { return r.armorClass }

// EffectiveArmorClass returns the Armor Class plus the cover bonus.
func (r AttackResolved) EffectiveArmorClass() d20.ArmorClass { return r.effective }

// Cover returns the applicable cover (0 for none).
func (r AttackResolved) Cover() Cover { return r.cover }

// Circumstances returns a copy of every Advantage and Disadvantage that went into
// the roll mode: those the facts gave, then the caller's.
func (r AttackResolved) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(r.circumstances)
}

// Delivery returns the attack's delivery, so the host can choose MeleeAttackKnockOut.
func (r AttackResolved) Delivery() AttackDelivery { return r.delivery }

// HitKind returns CriticalHit for a Critical Hit and NormalHit otherwise; it is
// meaningful for a hit.
func (r AttackResolved) HitKind() HitKind {
	if r.critical {
		return CriticalHit
	}
	return NormalHit
}

// Damage returns a copy of the damage of a hit, one Damage per component, and nil
// for a miss or an attack with no DamageSpec.
func (r AttackResolved) Damage() []Damage { return slices.Clone(r.damage) }

// RevealsAttackerLocation reports whether the attack gives away a hidden attacker.
func (r AttackResolved) RevealsAttackerLocation() bool { return r.reveals }

// AttackAutomaticallyMissed: the attack missed without a roll or damage.
type AttackAutomaticallyMissed struct {
	reason  AutomaticMissReason
	cover   Cover
	reveals bool
}

// Reason returns why no roll was made.
func (r AttackAutomaticallyMissed) Reason() AutomaticMissReason { return r.reason }

// Cover returns the applicable cover.
func (r AttackAutomaticallyMissed) Cover() Cover { return r.cover }

// RevealsAttackerLocation reports whether the attack gives away a hidden attacker.
func (r AttackAutomaticallyMissed) RevealsAttackerLocation() bool { return r.reveals }

// AttackNotPossible: the attack cannot be made, and no die was rolled. A refusal,
// not an error.
type AttackNotPossible struct {
	reason AttackNotPossibleReason
	cover  Cover
}

// Reason returns why the attack cannot be made.
func (r AttackNotPossible) Reason() AttackNotPossibleReason { return r.reason }

// Cover returns the applicable cover.
func (r AttackNotPossible) Cover() Cover { return r.cover }

func (AttackResolved) isResolveAttackResult()            {}
func (AttackAutomaticallyMissed) isResolveAttackResult() {}
func (AttackNotPossible) isResolveAttackResult()         {}
