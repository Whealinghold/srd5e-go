package combat

import "github.com/Whealinghold/srd5e-go/d20"

// ParryFacts are what the host knows of an attack against a creature with Parry: the
// attack roll that hit it, whether it was a melee attack roll and whether the creature is
// holding a weapon. Construct with a keyed literal; the zero value is an attack that
// missed.
type ParryFacts struct {
	Attack            d20.AttackRollResult
	IsMeleeAttackRoll bool
	HoldingWeapon     bool
}

// ParryUnavailableReason says why Parry cannot be used. The zero value is invalid.
// Comparable and safe for concurrent use.
type ParryUnavailableReason int

const (
	// ParryNoTrait: the trait is the zero value.
	ParryNoTrait ParryUnavailableReason = iota + 1
	// ParryAttackMissed: the attack did not hit.
	ParryAttackMissed
	// ParryNotMeleeAttackRoll: the attack roll was not a melee attack roll.
	ParryNotMeleeAttackRoll
	// ParryNotHoldingWeapon: the creature is not holding a weapon.
	ParryNotHoldingWeapon
)

// String returns "no_trait", "attack_missed", "not_melee_attack_roll",
// "not_holding_weapon" or "unknown".
func (r ParryUnavailableReason) String() string {
	return enumName([]string{"no_trait", "attack_missed", "not_melee_attack_roll", "not_holding_weapon"}, int(r))
}

// ParryOffer is the closed set of answers of AssessParry. Variants: ParryAvailable,
// ParryUnavailable.
type ParryOffer interface{ isParryOffer() }

// ParryAvailable: the creature may take its Reaction to Parry.
type ParryAvailable struct{}

// ParryUnavailable: Parry cannot be used. Construct with a keyed literal.
type ParryUnavailable struct{ Reason ParryUnavailableReason }

func (ParryAvailable) isParryOffer()   {}
func (ParryUnavailable) isParryOffer() {}

// AssessParry says whether Parry can answer the attack of facts (SRD 5.2.1, "Bandit",
// p. 262): only a hit by a melee attack roll while the creature holds a weapon. When not,
// the reason is the first of ParryNoTrait, ParryAttackMissed, ParryNotMeleeAttackRoll and
// ParryNotHoldingWeapon. It does not spend the Reaction: the host takes it first
// (TakeReaction with a FeatureReaction) and then calls RespondToParry. It never fails.
func AssessParry(t ParryTrait, facts ParryFacts) ParryOffer {
	switch {
	case t.IsZero():
		return ParryUnavailable{ParryNoTrait}
	case !facts.Attack.Hit():
		return ParryUnavailable{ParryAttackMissed}
	case !facts.IsMeleeAttackRoll:
		return ParryUnavailable{ParryNotMeleeAttackRoll}
	case !facts.HoldingWeapon:
		return ParryUnavailable{ParryNotHoldingWeapon}
	}
	return ParryAvailable{}
}

// ParryResponse is the result of RespondToParry: the attack roll resolved again against the
// raised Armor Class, whether the hit became a miss, and the event.
type ParryResponse struct {
	result     d20.AttackRollResult
	becameMiss bool
	event      ParryUsed
}

// Result returns the attack roll resolved against the raised Armor Class, with the same roll
// and modifiers; the original result is unchanged.
func (r ParryResponse) Result() d20.AttackRollResult { return r.result }

// BecameMiss reports whether the attack that hit now misses.
func (r ParryResponse) BecameMiss() bool { return r.becameMiss }

// Event returns the ParryUsed event.
func (r ParryResponse) Event() ParryUsed { return r.event }

// RespondToParry resolves attack again against the Armor Class of the attack raised by the
// trait's bonus, keeping the roll and the modifiers (SRD 5.2.1, "Marilith", p. 306). subject
// is the parrying creature. A natural 20 stays a hit and a Critical Hit and a natural 1 stays
// a miss: those rules ignore the Armor Class (SRD 5.2.1, "Rolling 20 or 1", p. 7).
//
// It returns an error matching srderr.ErrInvalidInput for a zero trait or subject, or an
// attack that did not hit (the zero result included), and the d20 error for an attack that
// cannot be resolved again. It never panics.
func RespondToParry(t ParryTrait, subject CombatantID, attack d20.AttackRollResult) (ParryResponse, error) {
	if t.IsZero() || subject.IsZero() || !attack.Hit() {
		return ParryResponse{}, invalidInput("parry answers a hit by a parrying creature with the trait")
	}
	raised, err := d20.ResolveAttack(attack.Roll(), attack.Modifiers(), d20.ArmorClass(satAdd(int(attack.Target()), t.bonus)), attack.CriticalRange())
	if err != nil {
		return ParryResponse{}, err
	}
	miss := !raised.Hit()
	return ParryResponse{raised, miss, ParryUsed{monsterTraitEventBase{subject}, t.bonus, miss}}, nil
}
