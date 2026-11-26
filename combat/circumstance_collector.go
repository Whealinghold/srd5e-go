package combat

import "github.com/Whealinghold/srd5e-go/modifier"

// AttackPlan is the closed set of answers of CircumstanceCollector.Collect: what
// the facts of an attack decide before any die is rolled. Variants:
// AttackPlanProceeds, AttackPlanNotPossible, AttackPlanAutomaticMiss.
type AttackPlan interface{ isAttackPlan() }

// AttackPlanProceeds: the attack is rolled. Circumstances are the Advantage and
// Disadvantage the facts give (the caller's own are added by the use case).
// Construct with a keyed literal.
type AttackPlanProceeds struct {
	Circumstances           []modifier.RollCircumstance
	MissOverride            MissOverride
	RevealsAttackerLocation bool
}

// AttackPlanNotPossible: the attack cannot be made, no die is rolled.
type AttackPlanNotPossible struct{ Reason AttackNotPossibleReason }

// AttackPlanAutomaticMiss: the attack misses without a roll.
type AttackPlanAutomaticMiss struct {
	Reason                  AutomaticMissReason
	RevealsAttackerLocation bool
}

func (AttackPlanProceeds) isAttackPlan()      {}
func (AttackPlanNotPossible) isAttackPlan()   {}
func (AttackPlanAutomaticMiss) isAttackPlan() {}

// CircumstanceCollector is the stateless domain service that decides whether an
// attack can be made and what Advantage and Disadvantage the facts give it. It
// takes the Advantage and Disadvantage as data, so it never queries conditions.
// The zero value is ready to use. Safe for concurrent use.
type CircumstanceCollector struct{}

// Collect decides an attack of form at distance feet against a target behind
// cover (SRD 5.2.1, "Cover", "Range", "Reach", "Ranged Attacks in Close Combat",
// "Unseen Attackers and Targets", "Underwater Combat", pp. 14-16).
//
// Not possible, before any die, and first reported in this order: Total Cover;
// a ranged attack beyond its range (AttackRange rng, beyond a single range or
// beyond long range); a melee attack beyond reach. Then an automatic miss: a
// ranged weapon attack underwater beyond normal range. Otherwise the attack
// proceeds with Disadvantage for an unseen target, from long range, from an
// enemy within 5 feet (ranged attacks, not melee), and underwater (a melee weapon
// attack of an attacker without a Swim Speed and without Piercing damage in
// damage; a ranged weapon attack within normal range), and Advantage when the
// target cannot see the attacker. A target the attacker cannot see and that is
// not at the targeted location is a MissOverride. An invalid form or a distance
// below 0 is read as 0.
func (CircumstanceCollector) Collect(form AttackForm, distance int, rng AttackRange, reach Reach, cover Cover,
	enemies []NearbyEnemy, sit AttackSituation, damage DamageSpec) AttackPlan {
	distance = max(distance, 0)
	if form == UnarmedStrikeAttack { // an Unarmed Strike ignores any supplied reach
		reach = Reach{}
	}
	ranged := form.Delivery() == AttackDeliveryRanged
	band := bandWithinNormal
	switch {
	case cover == TotalCover:
		return AttackPlanNotPossible{AttackNotPossibleReasonTotalCover}
	case ranged:
		if band = rng.bandAt(distance); band == bandBeyondLong {
			return AttackPlanNotPossible{AttackNotPossibleReasonBeyondRange}
		}
	case distance > reach.Feet():
		return AttackPlanNotPossible{AttackNotPossibleReasonBeyondReach}
	}
	weapon := form.Source() == AttackSourceWeapon
	if ranged && weapon && sit.Underwater && band == bandBeyondNormal {
		return AttackPlanAutomaticMiss{AutomaticMissReasonUnderwaterBeyondNormalRange, sit.AttackerHidden}
	}
	plan := AttackPlanProceeds{MissOverride: MissOverrideNone, RevealsAttackerLocation: sit.AttackerHidden}
	disadvantage := func() { plan.Circumstances = append(plan.Circumstances, modifier.Disadvantage{}) }
	if u, ok := sit.Visibility.(TargetUnseen); ok {
		disadvantage()
		if !u.TargetAtTargetedLocation {
			plan.MissOverride = MissOverrideTargetNotAtLocation
		}
	}
	if sit.TargetCannotSeeAttacker {
		plan.Circumstances = append(plan.Circumstances, modifier.Advantage{})
	}
	if band == bandBeyondNormal {
		disadvantage()
	}
	if ranged {
		for _, e := range enemies {
			if e.threatensInCloseCombat() {
				disadvantage()
				break
			}
		}
	}
	if weapon && sit.Underwater {
		if ranged || (!sit.SwimSpeed && !damage.dealsPiercing()) {
			disadvantage()
		}
	}
	return plan
}
