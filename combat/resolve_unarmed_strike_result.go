package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
)

// UnarmedStrikeSave is the saving throw of a Grapple or Shove as d20 resolved it.
// Variants: UnarmedStrikeSaveRolled, UnarmedStrikeSaveForced.
type UnarmedStrikeSave interface {
	isUnarmedStrikeSave()
	// Outcome returns d20.Success when the target saved, d20.Failure otherwise.
	Outcome() d20.Outcome
}

// UnarmedStrikeSaveRolled: the save was rolled.
type UnarmedStrikeSaveRolled struct{ result d20.CheckResult }

// Result returns the saving throw d20.ResolveCheck resolved, unchanged.
func (s UnarmedStrikeSaveRolled) Result() d20.CheckResult { return s.result }

// Outcome returns the outcome of the roll.
func (s UnarmedStrikeSaveRolled) Outcome() d20.Outcome { return s.result.Outcome() }

// UnarmedStrikeSaveForced: the save failed without a die, because the target chose
// to fail or fails automatically.
type UnarmedStrikeSaveForced struct{ result d20.ForcedFailureResult }

// Result returns the forced failure d20.FailWithoutRoll returned.
func (s UnarmedStrikeSaveForced) Result() d20.ForcedFailureResult { return s.result }

// Outcome returns d20.Failure.
func (UnarmedStrikeSaveForced) Outcome() d20.Outcome { return d20.Failure }

func (UnarmedStrikeSaveRolled) isUnarmedStrikeSave() {}
func (UnarmedStrikeSaveForced) isUnarmedStrikeSave() {}

// ResolveUnarmedStrikeResult is the closed set of outcomes of
// ResolveUnarmedStrike.Run. Variants: UnarmedStrikeDamageResolved,
// UnarmedStrikeGrappleResolved, UnarmedStrikeShoveResolved,
// UnarmedStrikeNotPossible.
type ResolveUnarmedStrikeResult interface{ isResolveUnarmedStrikeResult() }

// UnarmedStrikeDamageResolved: the Damage option's attack roll was made. A miss is
// a normal outcome, not an error.
type UnarmedStrikeDamageResolved struct {
	attack        d20.AttackRollResult
	hit, critical bool
	armorClass    d20.ArmorClass
	cover         Cover
	damage        []Damage
}

// Attack returns the attack roll exactly as d20.ResolveAttack returned it, with
// the effective Armor Class as its target.
func (r UnarmedStrikeDamageResolved) Attack() d20.AttackRollResult { return r.attack }

// Hit reports whether the strike hit.
func (r UnarmedStrikeDamageResolved) Hit() bool { return r.hit }

// CriticalHit reports a Critical Hit by the roll or by the automatic input.
func (r UnarmedStrikeDamageResolved) CriticalHit() bool { return r.critical }

// ArmorClass returns the target's Armor Class before cover.
func (r UnarmedStrikeDamageResolved) ArmorClass() d20.ArmorClass { return r.armorClass }

// Cover returns the applicable cover (0 for none).
func (r UnarmedStrikeDamageResolved) Cover() Cover { return r.cover }

// HitKind returns CriticalHit for a Critical Hit and NormalHit otherwise.
func (r UnarmedStrikeDamageResolved) HitKind() HitKind {
	if r.critical {
		return CriticalHit
	}
	return NormalHit
}

// Damage returns a copy of the damage of a hit, Bludgeoning equal to 1 plus the
// ability modifier (minimum 0), and nil for a miss.
func (r UnarmedStrikeDamageResolved) Damage() []Damage { return slices.Clone(r.damage) }

// UnarmedStrikeGrappleResolved: the target's save against the Grapple DC was
// resolved. The result only describes the effect; the host passes a succeeded
// Grapple to BeginGrapple.
type UnarmedStrikeGrappleResolved struct {
	dc          d20.DifficultyClass
	saveAbility ability.Ability
	save        UnarmedStrikeSave
	cover       Cover
}

// DC returns the Grapple DC.
func (r UnarmedStrikeGrappleResolved) DC() d20.DifficultyClass { return r.dc }

// SaveAbility returns the ability the target chose for its save.
func (r UnarmedStrikeGrappleResolved) SaveAbility() ability.Ability { return r.saveAbility }

// Save returns the saving throw.
func (r UnarmedStrikeGrappleResolved) Save() UnarmedStrikeSave { return r.save }

// Cover returns the applicable cover.
func (r UnarmedStrikeGrappleResolved) Cover() Cover { return r.cover }

// Grappled reports whether the target failed its save and so is Grappled.
func (r UnarmedStrikeGrappleResolved) Grappled() bool { return r.save.Outcome() == d20.Failure }

// UnarmedStrikeShoveResolved: the target's save against the Shove DC was resolved.
type UnarmedStrikeShoveResolved struct {
	dc          d20.DifficultyClass
	saveAbility ability.Ability
	save        UnarmedStrikeSave
	effect      ShoveEffect
	cover       Cover
}

// DC returns the Shove DC.
func (r UnarmedStrikeShoveResolved) DC() d20.DifficultyClass { return r.dc }

// SaveAbility returns the ability the target chose for its save.
func (r UnarmedStrikeShoveResolved) SaveAbility() ability.Ability { return r.saveAbility }

// Save returns the saving throw.
func (r UnarmedStrikeShoveResolved) Save() UnarmedStrikeSave { return r.save }

// Effect returns the effect the attacker declared.
func (r UnarmedStrikeShoveResolved) Effect() ShoveEffect { return r.effect }

// Cover returns the applicable cover.
func (r UnarmedStrikeShoveResolved) Cover() Cover { return r.cover }

// Applied reports whether the target failed its save, so the Effect applies. The
// host applies it; the result moves no creature and applies no condition.
func (r UnarmedStrikeShoveResolved) Applied() bool { return r.save.Outcome() == d20.Failure }

// UnarmedStrikeNotPossible: the Unarmed Strike cannot be made with that option, and
// no die was rolled. A refusal, not an error.
type UnarmedStrikeNotPossible struct {
	reason AttackNotPossibleReason
	cover  Cover
}

// Reason returns why the strike cannot be made.
func (r UnarmedStrikeNotPossible) Reason() AttackNotPossibleReason { return r.reason }

// Cover returns the applicable cover.
func (r UnarmedStrikeNotPossible) Cover() Cover { return r.cover }

func (UnarmedStrikeDamageResolved) isResolveUnarmedStrikeResult()  {}
func (UnarmedStrikeGrappleResolved) isResolveUnarmedStrikeResult() {}
func (UnarmedStrikeShoveResolved) isResolveUnarmedStrikeResult()   {}
func (UnarmedStrikeNotPossible) isResolveUnarmedStrikeResult()     {}
