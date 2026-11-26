package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// UnarmedStrikeModifiers returns the roll modifiers of an Unarmed Strike: the
// ability modifier used (Strength unless the caller supplies another) and the
// Proficiency Bonus, which is always added without a proficiency check, and no
// others (SRD 5.2.1, "Unarmed Strike", p. 190). It is pure and never panics.
func UnarmedStrikeModifiers(abilityModifier ability.Modifier, proficiencyBonus character.ProficiencyBonus) d20.RollModifiers {
	return d20.NewRollModifiers(modifier.New(abilityModifier.Value())).
		WithProficiencyBonus(modifier.New(proficiencyBonus.Value()))
}

// IsNoMoreThanOneSizeLarger reports whether target is not more than one size
// larger than attacker, in the order Tiny to Gargantuan (SRD 5.2.1, "Creature
// Size", p. 14). It is false when either size is not valid.
func IsNoMoreThanOneSizeLarger(attacker, target content.CreatureSize) bool {
	return attacker.Valid() && target.Valid() && target <= attacker+1
}

// unarmedStrikeDC returns 8 plus the ability modifier plus the Proficiency Bonus,
// the DC of a Shove (SRD 5.2.1, "Unarmed Strike", p. 190), without cap or floor;
// the sum saturates at the limits of int.
func unarmedStrikeDC(abilityModifier ability.Modifier, proficiencyBonus character.ProficiencyBonus) d20.DifficultyClass {
	return GrappleRules{}.GrappleDC(abilityModifier.Value(), proficiencyBonus.Value())
}

// planUnarmedStrike decides whether an Unarmed Strike with the option of p can be made,
// before any die, in the order Total Cover, beyond 5 feet, target too large (Grapple and
// Shove), no free hand (Grapple); otherwise it proceeds (SRD 5.2.1, "Unarmed Strike",
// p. 190; "Cover", p. 15). The Grapple checks are GrappleRules.Possibility mapped one
// to one.
func planUnarmedStrike(p ResolveUnarmedStrikeParams, cover Cover) AttackPlan {
	switch {
	case cover == TotalCover:
		return AttackPlanNotPossible{AttackNotPossibleReasonTotalCover}
	case p.Distance > unarmedStrikeReachFeet:
		return AttackPlanNotPossible{AttackNotPossibleReasonBeyondReach}
	}
	switch p.Option.(type) {
	case UnarmedStrikeGrappleOption:
		if n, ok := (GrappleRules{}).Possibility(p.AttackerSize, p.TargetSize, p.Hand).(GrappleNotPossible); ok {
			if n.Reason == GrappleNotPossibleReasonTargetTooLarge {
				return AttackPlanNotPossible{AttackNotPossibleReasonTargetTooLarge}
			}
			return AttackPlanNotPossible{AttackNotPossibleReasonNoFreeHand}
		}
	case UnarmedStrikeShoveOption:
		if !IsNoMoreThanOneSizeLarger(p.AttackerSize, p.TargetSize) {
			return AttackPlanNotPossible{AttackNotPossibleReasonTargetTooLarge}
		}
	}
	return AttackPlanProceeds{MissOverride: MissOverrideNone}
}

// unarmedStrikeReachFeet is the reach of an Unarmed Strike (SRD 5.2.1, "Unarmed
// Strike", p. 190); its command has no reach field.
const unarmedStrikeReachFeet = 5
