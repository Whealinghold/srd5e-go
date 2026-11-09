package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// deflectTypes reports whether types is a non-empty list of valid damage types and
// whether it includes Bludgeoning, Piercing or Slashing.
func deflectTypes(types []content.DamageType) (valid, physical bool) {
	valid = len(types) > 0
	for _, t := range types {
		valid = valid && t.Valid()
		physical = physical || t == content.DamageBludgeoning || t == content.DamagePiercing || t == content.DamageSlashing
	}
	return valid, physical
}

// DeflectResolution is the outcome of Deflect Attacks: the d10 rolled, the reduction
// and the damage that remains. Only DeflectAttacks.Resolve builds one.
type DeflectResolution struct {
	rolled, reduction, after int
}

// Rolled returns the d10 rolled.
func (r DeflectResolution) Rolled() int { return r.rolled }

// Reduction returns the reduction: the d10 plus the Dexterity modifier plus the Monk
// level, never below 0.
func (r DeflectResolution) Reduction() int { return r.reduction }

// DamageAfter returns the attack's total damage after the reduction, never below 0.
func (r DeflectResolution) DamageAfter() int { return r.after }

// CanRedirect reports that the damage was reduced to 0, so 1 Focus Point can redirect
// some of the attack's force.
func (r DeflectResolution) CanRedirect() bool { return r.after == 0 }

// DeflectAttacks decides and resolves Deflect Attacks and Deflect Energy (SRD 5.2.1,
// "Level 3: Deflect Attacks", p. 51, "Level 13: Deflect Energy", p. 52). It is
// stateless and pure; the zero value is ready to use. Safe for concurrent use.
type DeflectAttacks struct{}

// Availability reports whether the Monk can deflect an attack whose damage includes
// the given types: MonkNotAMonk for zero stats, MonkLevelTooLow below level 3,
// MonkNotEligible for no type, an invalid type, or damage with none of Bludgeoning,
// Piercing or Slashing below level 13.
func (DeflectAttacks) Availability(stats MonkLevelStats, types []content.DamageType) MonkAvailability {
	valid, physical := deflectTypes(types)
	return monkAvailability(
		monkGate(stats, monkDeflectLevel),
		monkIf(!valid || (!physical && stats.level.n < monkDeflectEnergyLevel), MonkNotEligible),
	)
}

// Resolve reduces an attack whose damage includes types and totals total by the d10
// roll rolled plus the Dexterity modifier plus the Monk level; a negative sum is 0.
//
// It returns an error matching srderr.ErrInvalidInput for a rolled outside 1 to 10, a
// negative total, or when Availability says unavailable. It never panics.
func (d DeflectAttacks) Resolve(stats MonkLevelStats, abilities MonkAbilities, types []content.DamageType, total, rolled int) (DeflectResolution, error) {
	if rolled < 1 || rolled > 10 || total < 0 {
		return DeflectResolution{}, invalidInput("Deflect Attacks roll " + strconv.Itoa(rolled) + " outside 1 to 10, or a negative damage total")
	}
	if _, ok := d.Availability(stats, types).(MonkAvailable); !ok {
		return DeflectResolution{}, invalidInput("Deflect Attacks resolved while unavailable")
	}
	reduction := max(rolled+abilities.Dexterity.Value()+stats.level.n, 0)
	return DeflectResolution{rolled: rolled, reduction: reduction, after: max(total-reduction, 0)}, nil
}

// MonkAttackRange says whether the attack Deflect Attacks reduced was melee or ranged.
// The zero value is not a range.
type MonkAttackRange int

// The attack ranges.
const (
	// MonkMeleeAttack: the redirect target is within 5 feet.
	MonkMeleeAttack MonkAttackRange = iota + 1
	// MonkRangedAttack: the redirect target is within 60 feet and not behind Total Cover.
	MonkRangedAttack
)

// String returns "melee", "ranged" or "unknown".
func (r MonkAttackRange) String() string {
	switch r {
	case MonkMeleeAttack:
		return "melee"
	case MonkRangedAttack:
		return "ranged"
	}
	return "unknown"
}

// MonkRedirectTarget is the host's report of the creature the Monk redirects the
// attack's force at. Construct it with a keyed literal.
type MonkRedirectTarget struct {
	// Seen reports that the Monk can see the creature.
	Seen bool
	// DistanceFeet is the distance from the Monk in feet.
	DistanceFeet int
	// BehindTotalCover reports that the creature has Total Cover from the Monk.
	BehindTotalCover bool
}

// MonkRedirectRequest is one redirect of a deflected attack's force. Construct it
// with a keyed literal.
type MonkRedirectRequest struct {
	// DamageAfter is the attack's damage after Deflect Attacks reduced it.
	DamageAfter int
	// Attack is whether the attack was melee or ranged.
	Attack MonkAttackRange
	// Target is the creature the force is redirected at.
	Target MonkRedirectTarget
	// DamageType is the type the attack dealt, which the redirected damage has.
	DamageType content.DamageType
}

// RedirectResolution is the outcome of a redirect: the DC used, the damage on a failed
// save, the damage applied and the new state. Only DeflectRedirect.Resolve builds one.
type RedirectResolution struct {
	dc                    d20.DifficultyClass
	onFailedSave, applied int
	damageType            content.DamageType
	state                 MonkState
}

// DifficultyClass returns the save DC used: the Monk's Focus save DC.
func (r RedirectResolution) DifficultyClass() d20.DifficultyClass { return r.dc }

// DamageOnFailedSave returns two rolls of the Martial Arts die plus the Dexterity modifier, never below 0.
func (r RedirectResolution) DamageOnFailedSave() int { return r.onFailedSave }

// DamageApplied returns the damage the creature takes: DamageOnFailedSave on a failed save, else 0.
func (r RedirectResolution) DamageApplied() int { return r.applied }

// DamageType returns the type of the redirected damage: the type the attack dealt.
func (r RedirectResolution) DamageType() content.DamageType { return r.damageType }

// State returns the new state: 1 Focus Point expended.
func (r RedirectResolution) State() MonkState { return r.state }

// DeflectRedirect decides and resolves the redirect part of Deflect Attacks (SRD 5.2.1,
// "Level 3: Deflect Attacks", p. 51). It is stateless and pure; the zero value is ready
// to use. Safe for concurrent use.
type DeflectRedirect struct{}

func redirectTargetOK(req MonkRedirectRequest) bool {
	t := req.Target
	if !t.Seen || t.DistanceFeet < 0 {
		return false
	}
	switch req.Attack {
	case MonkMeleeAttack:
		return t.DistanceFeet <= 5
	case MonkRangedAttack:
		return t.DistanceFeet <= 60 && !t.BehindTotalCover
	}
	return false
}

// Availability reports whether the Monk can redirect: MonkNotAMonk for zero stats,
// MonkLevelTooLow below level 3, MonkNotEligible when the damage after the reduction
// is above 0, the damage type is invalid or the target is out of range, unseen or (for
// a ranged attack) behind Total Cover, MonkNotEnoughFocusPoints with no Focus Point.
func (DeflectRedirect) Availability(stats MonkLevelStats, state MonkState, req MonkRedirectRequest) MonkAvailability {
	return monkAvailability(
		monkGate(stats, monkDeflectLevel),
		monkIf(req.DamageAfter != 0 || !req.DamageType.Valid() || !redirectTargetOK(req), MonkNotEligible),
		monkIf(state.FocusPoints(stats).Available() < 1, MonkNotEnoughFocusPoints),
	)
}

// Resolve redirects the force: the creature makes a Dexterity saving throw against the
// Monk's Focus save DC (outcome is that save) and takes two rolls of the Martial Arts
// die (rolledTotal, their sum) plus the Dexterity modifier on a failure; 1 Focus Point
// is expended.
//
// It returns an error matching srderr.ErrInvalidInput for a rolledTotal that is not a
// possible total of two Martial Arts dice, an outcome other than d20.Success or
// d20.Failure, or when Availability says unavailable. It never panics.
func (r DeflectRedirect) Resolve(stats MonkLevelStats, state MonkState, abilities MonkAbilities, req MonkRedirectRequest, rolledTotal int, outcome d20.Outcome) (RedirectResolution, error) {
	if (outcome != d20.Success && outcome != d20.Failure) || !monkRolledWithin(rolledTotal, 2, stats.martialArts) {
		return RedirectResolution{}, invalidInput("redirect outcome or Martial Arts roll total " + strconv.Itoa(rolledTotal) + " invalid")
	}
	if _, ok := r.Availability(stats, state, req).(MonkAvailable); !ok {
		return RedirectResolution{}, invalidInput("redirect resolved while unavailable")
	}
	onFailed := max(rolledTotal+abilities.Dexterity.Value(), 0)
	applied := 0
	if outcome == d20.Failure {
		applied = onFailed
	}
	state.focusExpended++
	return RedirectResolution{dc: abilities.SaveDC(), onFailedSave: onFailed, applied: applied, damageType: req.DamageType, state: state}, nil
}
