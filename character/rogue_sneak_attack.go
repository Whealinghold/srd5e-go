package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// sneakAllyRangeFeet is how close an ally must be to the target to stand in for Advantage (SRD 5.2.1, "Level 1: Sneak
// Attack", p. 62).
const sneakAllyRangeFeet = 5

// AllyNearTarget is an ally of the Rogue near the target of an attack: how far from the target it is and whether it has the
// Incapacitated condition. The caller builds it from the battlefield (SRD 5.2.1, "Level 1: Sneak Attack", p. 62). Immutable,
// comparable with ==.
type AllyNearTarget struct {
	feet          int
	incapacitated bool
}

// NewAllyNearTarget returns an ally feet from the target. It returns an error matching srderr.ErrInvalidInput for a
// negative distance.
func NewAllyNearTarget(feet int, incapacitated bool) (AllyNearTarget, error) {
	if feet < 0 {
		return AllyNearTarget{}, invalidInput("negative ally distance")
	}
	return AllyNearTarget{feet: feet, incapacitated: incapacitated}, nil
}

// DistanceFeet returns the distance from the target in feet.
func (a AllyNearTarget) DistanceFeet() int { return a.feet }

// Incapacitated reports whether the ally has the Incapacitated condition.
func (a AllyNearTarget) Incapacitated() bool { return a.incapacitated }

func (a AllyNearTarget) qualifies() bool { return a.feet <= sneakAllyRangeFeet && !a.incapacitated }

// SneakAttackReason is why Sneak Attack does not apply. The values are in the order the checks run, which is the order the
// reasons of one result are reported in. The zero value is not a reason.
type SneakAttackReason int

const (
	// SneakAttackWeaponNotFinesseOrRanged: the weapon has neither the Finesse property nor is Ranged.
	SneakAttackWeaponNotFinesseOrRanged SneakAttackReason = iota + 1
	// SneakAttackNoAdvantageOrQualifyingAlly: no Advantage on the attack roll and no qualifying ally.
	SneakAttackNoAdvantageOrQualifyingAlly
	// SneakAttackAlreadyUsedThisTurn: Sneak Attack was already used this turn.
	SneakAttackAlreadyUsedThisTurn
	// SneakAttackAttackMissed: the attack roll did not hit.
	SneakAttackAttackMissed
)

// String returns the reason as snake_case, or "unknown".
func (r SneakAttackReason) String() string {
	switch r {
	case SneakAttackWeaponNotFinesseOrRanged:
		return "weapon_not_finesse_or_ranged"
	case SneakAttackNoAdvantageOrQualifyingAlly:
		return "no_advantage_or_qualifying_ally"
	case SneakAttackAlreadyUsedThisTurn:
		return "already_used_this_turn"
	case SneakAttackAttackMissed:
		return "attack_missed"
	}
	return "unknown"
}

// SneakAttackResult is the outcome of SneakAttack.Evaluate: the closed set SneakAttackApplies and SneakAttackDoesNotApply.
//
//sumtype:decl
type SneakAttackResult interface{ isSneakAttackResult() }

// SneakAttackApplies means the extra damage is due: its dice (never a rolled total, so a Critical Hit can roll them twice)
// and its damage type, the weapon's. Immutable.
type SneakAttackApplies struct {
	dice       SneakAttackDice
	damageType content.DamageType
}

// Dice returns the Sneak Attack dice to roll.
func (r SneakAttackApplies) Dice() SneakAttackDice { return r.dice }

// DamageType returns the damage type of the extra damage, the weapon's.
func (r SneakAttackApplies) DamageType() content.DamageType { return r.damageType }

// SneakAttackDoesNotApply means no extra damage: every reason that failed, in check order. Immutable.
type SneakAttackDoesNotApply struct{ reasons []SneakAttackReason }

// Reasons returns every failing reason as a new slice, in check order.
func (r SneakAttackDoesNotApply) Reasons() []SneakAttackReason {
	return append([]SneakAttackReason(nil), r.reasons...)
}

func (SneakAttackApplies) isSneakAttackResult()      {}
func (SneakAttackDoesNotApply) isSneakAttackResult() {}

// SneakAttackInput is what SneakAttack.Evaluate reads. Construct it with a keyed literal.
type SneakAttackInput struct {
	// Dice are the Sneak Attack dice of the Rogue level (RogueProgression.SneakAttackDice).
	Dice SneakAttackDice
	// Attack is the attack roll's result; only a hit qualifies.
	Attack d20.AttackRollResult
	// Finesse reports that the weapon has the Finesse property.
	Finesse bool
	// Ranged reports that the weapon is a Ranged weapon.
	Ranged bool
	// Mode is the net roll mode of the attack roll, already cancelled by modifier.Resolve.
	Mode modifier.RollMode
	// Allies are the allies near the target.
	Allies []AllyNearTarget
	// UsedThisTurn reports that Sneak Attack was already used this turn.
	UsedThisTurn bool
	// DamageType is the weapon's damage type.
	DamageType content.DamageType
}

// SneakAttack is the Sneak Attack rule (SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62). It is a pure function of its
// input; the zero value is ready to use.
type SneakAttack struct{}

// Evaluate decides whether Sneak Attack applies to an attack. It applies once per turn to a hit with a Finesse or Ranged
// weapon when the roll has Advantage, or when it has no Disadvantage and an ally who is not Incapacitated is within 5 feet
// of the target. Otherwise the result lists every failing reason in check order. in is not changed.
func (SneakAttack) Evaluate(in SneakAttackInput) SneakAttackResult {
	var reasons []SneakAttackReason
	if !in.Finesse && !in.Ranged {
		reasons = append(reasons, SneakAttackWeaponNotFinesseOrRanged)
	}
	if in.Mode != modifier.WithAdvantage && (in.Mode == modifier.WithDisadvantage || !anyAllyQualifies(in.Allies)) {
		reasons = append(reasons, SneakAttackNoAdvantageOrQualifyingAlly)
	}
	if in.UsedThisTurn {
		reasons = append(reasons, SneakAttackAlreadyUsedThisTurn)
	}
	if !in.Attack.Hit() {
		reasons = append(reasons, SneakAttackAttackMissed)
	}
	if len(reasons) > 0 {
		return SneakAttackDoesNotApply{reasons: reasons}
	}
	return SneakAttackApplies{dice: in.Dice, damageType: in.DamageType}
}

func anyAllyQualifies(allies []AllyNearTarget) bool {
	for _, a := range allies {
		if a.qualifies() {
			return true
		}
	}
	return false
}
