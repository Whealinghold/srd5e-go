package combat

import "github.com/Whealinghold/srd5e-go/d20"

// parried is the plan of the target's Parry for one hit: what became of it, the target's tracker step to
// commit and the response whose result replaces the attack roll for the rest of the call (FR-161).
type parried struct {
	outcome  ParryOutcome
	step     *TrackerStepApplied
	response *ParryResponse
}

// becameMiss reports whether the Parry turned the hit into a miss.
func (p parried) becameMiss() bool { return p.response != nil && p.response.BecameMiss() }

// planParry decides the target's Parry against an attack that rolled roll: Assess, then the tracker step of
// the Reaction on after (the tracker once the attacker's step is applied), then Respond. Nothing is committed.
// A refusal, an unavailable Parry and NoParry leave the hit unchanged and stop nothing.
func planParry(choice ParryChoice, after TurnTracker, targetID CombatantID, target Combatant, roll d20.AttackRollResult,
	melee bool) (parried, error) {
	c, ok := choice.(ParryIfAble)
	if !ok {
		return parried{outcome: ParryNotOffered{}}, nil
	}
	if offer, unavailable := AssessParry(c.Trait, ParryFacts{Attack: roll, IsMeleeAttackRoll: melee, HoldingWeapon: c.HoldingWeapon}).(ParryUnavailable); unavailable {
		return parried{outcome: ParryNotUsed{Unavailable: offer.Reason}}, nil
	}
	step, err := after.TakeReaction(targetID, availabilityOf(target.conditions), c.Reaction)
	if err != nil {
		return parried{}, err
	}
	applied, ok := step.(TrackerStepApplied)
	if !ok {
		return parried{outcome: ParryNotUsed{Refused: step.(TrackerStepRejected).Reason}}, nil
	}
	resp, err := RespondToParry(c.Trait, targetID, roll)
	if err != nil {
		return parried{}, err
	}
	return parried{ParryApplied{resp}, &applied, &resp}, nil
}
