package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var fx combat.ConditionEffects

func attackFacts(distance int, sees, seen combat.SightRelation) combat.ConditionAttackFacts {
	return combat.ConditionAttackFacts{AttackerID: idA(), TargetID: idB(), DistanceFeet: distance,
		AttackerSeesTarget: sees, TargetSeesAttacker: seen, AttackerFearSource: combat.LineOfSightBlocked}
}

// Requirement: per case
// Path: per case.
func TestAttackRollsAgainstACreature(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	can, cannot := combat.SightRelationCanSee, combat.SightRelationCannotSee
	cases := []struct {
		name, requirement, path string
		target                  []combat.Condition
		distance                int
		sees                    combat.SightRelation
		want                    []contribution
		crit                    combat.AutomaticCritical
	}{
		{"Blinded", `FR-050; p. 177`, "happy", condConds(combat.Blinded{}), 30, cannot, []contribution{{cB, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Invisible", `FR-070; p. 184`, "happy", condConds(combat.Invisible{}), 30, cannot, []contribution{{cIv, dis}}, combat.AutomaticCriticalNotAutomatic},
		{"Invisible, attacker can see", `FR-072; p. 184`, "unhappy (rule interaction)", condConds(combat.Invisible{}), 30, can, nil, combat.AutomaticCriticalNotAutomatic},
		{"Paralyzed adjacent", `FR-076, FR-077; p. 186`, "happy", condConds(combat.Paralyzed{}), 5, cannot, []contribution{{cPa, adv}}, combat.AutomaticCriticalAutomatic},
		{"Paralyzed, 6 feet", `FR-076, FR-078; p. 186`, "unhappy (boundary)", condConds(combat.Paralyzed{}), 6, cannot, []contribution{{cPa, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Paralyzed, 0 feet", `FR-077; p. 186`, "unhappy (boundary)", condConds(combat.Paralyzed{}), 0, cannot, []contribution{{cPa, adv}}, combat.AutomaticCriticalAutomatic},
		{"Petrified adjacent", `FR-080, FR-083; p. 186`, "unhappy (rule interaction)", condConds(combat.Petrified{}), 5, cannot, []contribution{{cPe, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Prone, 5 feet", `FR-089; p. 186`, "unhappy (boundary)", condConds(combat.Prone{}), 5, cannot, []contribution{{cPr, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Prone, 6 feet", `FR-090; p. 186`, "unhappy (boundary)", condConds(combat.Prone{}), 6, cannot, []contribution{{cPr, dis}}, combat.AutomaticCriticalNotAutomatic},
		{"Prone, 0 feet", `FR-089; p. 186`, "unhappy (boundary)", condConds(combat.Prone{}), 0, cannot, []contribution{{cPr, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Restrained", `FR-092; p. 187`, "happy", condConds(combat.Restrained{}), 30, cannot, []contribution{{cR, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Stunned adjacent", `FR-097, FR-098; p. 189`, "unhappy (rule interaction)", condConds(combat.Stunned{}), 5, cannot, []contribution{{cS, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"Unconscious adjacent", `FR-089, FR-101, FR-103; p. 191`, "happy", condConds(combat.Unconscious{}), 5, cannot, []contribution{{cPr, adv}, {cU, adv}}, combat.AutomaticCriticalAutomatic},
		{"Unconscious, 6 feet", `FR-090, FR-101, FR-103; p. 191`, "unhappy (rule interaction)", condConds(combat.Unconscious{}), 6, cannot, []contribution{{cPr, dis}, {cU, adv}}, combat.AutomaticCriticalNotAutomatic},
		{"no conditions", `FR-109; p. 179`, "unhappy (empty and minimal)", nil, 5, cannot, nil, combat.AutomaticCriticalNotAutomatic},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			target := condSet(t, c.target)
			got, err := fx.AttackRoll(combat.ConditionSet{}, target, attackFacts(c.distance, c.sees, cannot))
			if err != nil || !slices.Equal(contributionsOf(got), c.want) {
				t.Errorf("AttackRoll = %v, %v, want %v [conditions.md %s]", contributionsOf(got), err, c.want, c.requirement)
			}
			if crit, err := fx.CriticalHit(target, c.distance); err != nil || crit != c.crit {
				t.Errorf("CriticalHit at %d feet = %v, %v, want %v [conditions.md %s]", c.distance, crit, err, c.crit, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestOwnAttackRolls(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	can, cannot := combat.SightRelationCanSee, combat.SightRelationCannotSee
	clear, blocked := combat.LineOfSightClear, combat.LineOfSightBlocked
	cases := []struct {
		name, requirement, path string
		attacker                []combat.Condition
		targetID                combat.CombatantID
		seen                    combat.SightRelation
		fear                    combat.LineOfSight
		want                    []contribution
	}{
		{"Blinded", `FR-051; p. 177`, "happy", condConds(combat.Blinded{}), idB(), cannot, blocked, []contribution{{cB, dis}}},
		{"Frightened, source in sight", `FR-057; p. 182`, "happy", condConds(combat.Frightened{Source: idB()}), idB(), cannot, clear, []contribution{{cF, dis}}},
		{"Frightened, source out of sight", `FR-058; p. 182`, "unhappy (boundary)", condConds(combat.Frightened{Source: idB()}), idB(), cannot, blocked, nil},
		{"Grappled, another target", `FR-061; p. 182`, "happy", condConds(combat.Grappled{Grappler: idA()}), idB(), cannot, blocked, []contribution{{cG, dis}}},
		{"Grappled, the grappler", `FR-062; p. 182`, "unhappy (boundary)", condConds(combat.Grappled{Grappler: idB()}), idB(), cannot, blocked, nil},
		{"Invisible", `FR-071; p. 184`, "happy", condConds(combat.Invisible{}), idB(), cannot, blocked, []contribution{{cIv, adv}}},
		{"Invisible, target can see", `FR-072; p. 184`, "unhappy (rule interaction)", condConds(combat.Invisible{}), idB(), can, blocked, nil},
		{"Poisoned", `FR-084; p. 186`, "happy", condConds(combat.Poisoned{}), idB(), cannot, blocked, []contribution{{cPo, dis}}},
		{"Prone", `FR-088; p. 186`, "happy", condConds(combat.Prone{}), idB(), cannot, blocked, []contribution{{cPr, dis}}},
		{"Restrained", `FR-093; p. 187`, "happy", condConds(combat.Restrained{}), idB(), cannot, blocked, []contribution{{cR, dis}}},
		{"conditions with no effect on attacks", `FR-109`, "unhappy (empty and minimal)", condConds(combat.Deafened{}, combat.Stunned{}), idB(), cannot, blocked, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			facts := combat.ConditionAttackFacts{AttackerID: idA(), TargetID: c.targetID, DistanceFeet: 10,
				AttackerSeesTarget: cannot, TargetSeesAttacker: c.seen, AttackerFearSource: c.fear}
			got, err := fx.AttackRoll(condSet(t, c.attacker), combat.ConditionSet{}, facts)
			if err != nil || !slices.Equal(contributionsOf(got), c.want) {
				t.Errorf("AttackRoll = %v, %v, want %v [conditions.md %s]", contributionsOf(got), err, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-110 - a negative distance or Speed returns an error matching ErrInvalidInput and the
// zero value; 0 is accepted.
// Path: unhappy (invalid input).
func TestQueriesRejectNegativeInput(t *testing.T) {
	t.Parallel()
	s := condSet(t, condConds(combat.Paralyzed{}))
	if got, err := fx.AttackRoll(s, s, attackFacts(-1, 0, 0)); !errors.Is(err, srderr.ErrInvalidInput) || len(got.Items()) != 0 {
		t.Errorf("AttackRoll(-1) = %v, %v", got, err)
	}
	if got, err := fx.CriticalHit(s, -1); !errors.Is(err, srderr.ErrInvalidInput) || got != 0 {
		t.Errorf("CriticalHit(-1) = %v, %v", got, err)
	}
	if got, err := fx.Speed(s, -5); !errors.Is(err, srderr.ErrInvalidInput) || got != (combat.EffectiveSpeed{}) {
		t.Errorf("Speed(-5) = %v, %v", got, err)
	}
	if got, err := fx.StandUp(s, -5); !errors.Is(err, srderr.ErrInvalidInput) || got != nil {
		t.Errorf("StandUp(-5) = %v, %v", got, err)
	}
	if _, err := fx.AttackRoll(s, s, attackFacts(0, 0, 0)); err != nil {
		t.Errorf("AttackRoll(0) = %v, want accepted", err)
	}
	if got, err := fx.Speed(combat.ConditionSet{}, 0); err != nil || got != (combat.EffectiveSpeed{Value: 0, Increase: combat.SpeedIncreaseAllowed}) {
		t.Errorf("Speed(0) = %v, %v", got, err)
	}
}

// Requirement: per case
// Path: per case.
func TestAbilityChecks(t *testing.T) {
	t.Parallel()
	dis := modifier.Disadvantage{}
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		requires                combat.CheckRequirement
		fear                    combat.LineOfSight
		wantEffect              combat.AbilityCheckEffect
		wantContrib             []contribution
	}{
		{"Blinded, sight", `FR-048; p. 177`, "happy", condConds(combat.Blinded{}), combat.CheckRequirementSight, combat.LineOfSightBlocked, combat.AbilityCheckEffectAutomaticFailure, nil},
		{"Blinded, no sense", `FR-049; p. 177`, "unhappy (boundary)", condConds(combat.Blinded{}), combat.CheckRequirementNone, combat.LineOfSightBlocked, combat.AbilityCheckEffectUnaffected, nil},
		{"Blinded, hearing", `FR-049; p. 177`, "unhappy (boundary)", condConds(combat.Blinded{}), combat.CheckRequirementHearing, combat.LineOfSightBlocked, combat.AbilityCheckEffectUnaffected, nil},
		{"Deafened, hearing", `FR-055; p. 181`, "happy", condConds(combat.Deafened{}), combat.CheckRequirementHearing, combat.LineOfSightBlocked, combat.AbilityCheckEffectAutomaticFailure, nil},
		{"Deafened, sight", `FR-056; p. 181`, "unhappy (boundary)", condConds(combat.Deafened{}), combat.CheckRequirementSight, combat.LineOfSightBlocked, combat.AbilityCheckEffectUnaffected, nil},
		{"Blinded and Deafened, sight", `FR-048, FR-055; pp. 177, 181`, "unhappy (rule interaction)", condConds(combat.Blinded{}, combat.Deafened{}), combat.CheckRequirementSight, combat.LineOfSightBlocked, combat.AbilityCheckEffectAutomaticFailure, nil},
		{"Blinded and Deafened, hearing", `FR-048, FR-055; pp. 177, 181`, "unhappy (rule interaction)", condConds(combat.Blinded{}, combat.Deafened{}), combat.CheckRequirementHearing, combat.LineOfSightBlocked, combat.AbilityCheckEffectAutomaticFailure, nil},
		{"Poisoned", `FR-085; p. 186`, "happy", condConds(combat.Poisoned{}), combat.CheckRequirementNone, combat.LineOfSightBlocked, combat.AbilityCheckEffectUnaffected, []contribution{{cPo, dis}}},
		{"Frightened, source in sight", `FR-057; p. 182`, "happy", condConds(combat.Frightened{Source: idA()}), combat.CheckRequirementNone, combat.LineOfSightClear, combat.AbilityCheckEffectUnaffected, []contribution{{cF, dis}}},
		{"Frightened, source out of sight", `FR-058; p. 182`, "unhappy (boundary)", condConds(combat.Frightened{Source: idA()}), combat.CheckRequirementNone, combat.LineOfSightBlocked, combat.AbilityCheckEffectUnaffected, nil},
		{"no conditions", `FR-109; p. 179`, "unhappy (empty and minimal)", nil, combat.CheckRequirementSight, combat.LineOfSightClear, combat.AbilityCheckEffectUnaffected, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := condSet(t, c.conds)
			if got := fx.AbilityCheckEffect(s, c.requires); got != c.wantEffect {
				t.Errorf("AbilityCheckEffect = %v, want %v [conditions.md %s]", got, c.wantEffect, c.requirement)
			}
			if got := contributionsOf(fx.AbilityCheck(s, c.fear)); !slices.Equal(got, c.wantContrib) {
				t.Errorf("AbilityCheck = %v, want %v [conditions.md %s]", got, c.wantContrib, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestSocialCheckAndCharmerRestriction(t *testing.T) {
	t.Parallel()
	item, _ := combat.NewCharmedByItem("sword")
	byA := condSet(t, condConds(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}))
	byItem := condSet(t, condConds(combat.Charmed{Charmer: item}))
	cases := []struct {
		name, requirement, path string
		charmed                 combat.ConditionSet
		other                   combat.CombatantID
		social                  []contribution
		damaging, notDamaging   combat.ConditionPermission
	}{
		{"the charmer", `FR-052, FR-054; p. 178`, "happy", byA, idA(), []contribution{{cC, modifier.Advantage{}}}, combat.ConditionPermissionForbidden, combat.ConditionPermissionAllowed},
		{"another creature", `FR-053, FR-054; p. 178`, "unhappy (boundary)", byA, idB(), nil, combat.ConditionPermissionAllowed, combat.ConditionPermissionAllowed},
		{"an item charmer", `FR-141; p. 208`, "unhappy (rule interaction)", byItem, idA(), nil, combat.ConditionPermissionAllowed, combat.ConditionPermissionAllowed},
		{"not charmed", `FR-109`, "unhappy (empty and minimal)", combat.ConditionSet{}, idA(), nil, combat.ConditionPermissionAllowed, combat.ConditionPermissionAllowed},
		{"charmed but immune", `FR-124; p. 183`, "unhappy (rule interaction)", condSet(t, condConds(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}), cC), idA(), nil, combat.ConditionPermissionAllowed, combat.ConditionPermissionAllowed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := contributionsOf(fx.SocialCheck(c.charmed, c.other)); !slices.Equal(got, c.social) {
				t.Errorf("SocialCheck = %v, want %v [conditions.md %s]", got, c.social, c.requirement)
			}
			if got := fx.CanTarget(c.charmed, c.other, combat.EffectDamageDamaging); got != c.damaging {
				t.Errorf("CanTarget(damaging) = %v, want %v [conditions.md %s]", got, c.damaging, c.requirement)
			}
			if got := fx.CanTarget(c.charmed, c.other, combat.EffectDamageNotDamaging); got != c.notDamaging {
				t.Errorf("CanTarget(not damaging) = %v, want %v [conditions.md %s]", got, c.notDamaging, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestSavingThrows(t *testing.T) {
	t.Parallel()
	auto, dis, none := combat.SavingThrowEffectAutomaticFailure, combat.SavingThrowEffectDisadvantage, combat.SavingThrowEffectUnaffected
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		str, dex, other         combat.SavingThrowEffect
	}{
		{"Paralyzed", `FR-074, FR-075; p. 186`, "happy", condConds(combat.Paralyzed{}), auto, auto, none},
		{"Petrified", `FR-081; p. 186`, "happy", condConds(combat.Petrified{}), auto, auto, none},
		{"Stunned", `FR-096; p. 189`, "happy", condConds(combat.Stunned{}), auto, auto, none},
		{"Unconscious", `FR-102; p. 191`, "happy", condConds(combat.Unconscious{}), auto, auto, none},
		{"Restrained", `FR-094, FR-095; p. 187`, "happy", condConds(combat.Restrained{}), none, dis, none},
		{"Paralyzed and Restrained", `FR-106; pp. 186, 187`, "unhappy (rule interaction)", condConds(combat.Paralyzed{}, combat.Restrained{}), auto, auto, none},
		{"Restrained and Paralyzed in the other order", `FR-106`, "unhappy (determinism)", condConds(combat.Restrained{}, combat.Paralyzed{}), auto, auto, none},
		{"no conditions", `FR-109; p. 179`, "unhappy (empty and minimal)", nil, none, none, none},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := condSet(t, c.conds)
			for _, a := range ability.All() {
				want := c.other
				if a == ability.Strength {
					want = c.str
				}
				if a == ability.Dexterity {
					want = c.dex
				}
				if got := fx.SavingThrow(s, a); got != want {
					t.Errorf("SavingThrow(%v) = %v, want %v [conditions.md %s]", a, got, want, c.requirement)
				}
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestSpeed(t *testing.T) {
	t.Parallel()
	allowed, forbidden := combat.SpeedIncreaseAllowed, combat.SpeedIncreaseForbidden
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		immune                  []content.ConditionKind
		base, want              int
		increase                combat.SpeedIncrease
	}{
		{"Grappled", `FR-060, FR-128; p. 182`, "happy", condConds(combat.Grappled{Grappler: idA()}), nil, 30, 0, forbidden},
		{"Paralyzed", `FR-073, FR-128; p. 186`, "happy", condConds(combat.Paralyzed{}), nil, 30, 0, forbidden},
		{"Petrified", `FR-079, FR-128; p. 186`, "happy", condConds(combat.Petrified{}), nil, 30, 0, forbidden},
		{"Restrained", `FR-091, FR-128; p. 187`, "happy", condConds(combat.Restrained{}), nil, 30, 0, forbidden},
		{"Unconscious", `FR-100, FR-128; p. 191`, "happy", condConds(combat.Unconscious{}), nil, 30, 0, forbidden},
		{"Stunned only", `FR-098, FR-128; p. 189`, "unhappy (rule interaction)", condConds(combat.Stunned{}), nil, 30, 30, allowed},
		{"Exhaustion 1", `FR-042; p. 181`, "happy", condConds(condLevel(1)), nil, 30, 25, allowed},
		{"Exhaustion 5", `FR-042; p. 181`, "unhappy (boundary)", condConds(condLevel(5)), nil, 30, 5, allowed},
		{"Exhaustion 6", `FR-042; p. 181`, "unhappy (boundary)", condConds(condLevel(6)), nil, 30, 0, allowed},
		{"Exhaustion above base", `FR-107; p. 181`, "unhappy (boundary)", condConds(condLevel(4)), nil, 15, 0, allowed},
		{"Speed 0 beats Exhaustion", `FR-107; pp. 181, 182`, "unhappy (rule interaction)", condConds(combat.Grappled{Grappler: idA()}, condLevel(2)), nil, 30, 0, forbidden},
		{"Exhaustion suppressed by an immunity", `FR-124`, "unhappy (rule interaction)", condConds(condLevel(3)), condKinds(cE), 30, 30, allowed},
		{"base 0", `FR-110`, "unhappy (boundary)", nil, nil, 0, 0, allowed},
		{"no conditions", `FR-109`, "unhappy (empty and minimal)", nil, nil, 30, 30, allowed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := fx.Speed(condSet(t, c.conds, c.immune...), c.base)
			if err != nil || got != (combat.EffectiveSpeed{Value: c.want, Increase: c.increase}) {
				t.Errorf("Speed(%d) = %v, %v, want %d %v [conditions.md %s]", c.base, got, err, c.want, c.increase, c.requirement)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestStandUp(t *testing.T) {
	t.Parallel()
	prone := condConds(combat.Prone{})
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		base                    int
		want                    combat.ProneMovement
	}{
		{"Speed 30", `FR-086; p. 186`, "happy", prone, 30, combat.CrawlOrRightItself{CostFeet: 15}},
		{"Speed 25 rounds down", `FR-086; p. 187`, "unhappy (boundary)", prone, 25, combat.CrawlOrRightItself{CostFeet: 12}},
		{"Speed 1", `FR-086; p. 187`, "unhappy (boundary)", prone, 1, combat.CrawlOrRightItself{CostFeet: 0}},
		{"Speed 0", `FR-087; p. 186`, "unhappy (boundary)", prone, 0, combat.CrawlOnly{}},
		{"Prone and Grappled", `FR-087, FR-107; pp. 182, 186`, "unhappy (rule interaction)", condConds(combat.Prone{}, combat.Grappled{Grappler: idA()}), 30, combat.CrawlOnly{}},
		{"Prone and Exhaustion 2", `FR-107`, "unhappy (rule interaction)", condConds(combat.Prone{}, condLevel(2)), 30, combat.CrawlOrRightItself{CostFeet: 10}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := fx.StandUp(condSet(t, c.conds), c.base)
			if err != nil || got != c.want {
				t.Errorf("StandUp(%d) = %#v, %v, want %#v [conditions.md %s]", c.base, got, err, c.want, c.requirement)
			}
			switch got.(type) {
			case combat.CrawlOnly, combat.CrawlOrRightItself:
			default:
				t.Errorf("StandUp returned %T, want one of the two ProneMovement variants", got)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-041 to FR-042, FR-040, FR-047 (SRD 5.2.1, "Exhaustion", p. 181) - the D20 Test
// reduction is 2 per level, the Speed reduction 5 feet per level and death comes at level 6 only.
// Path: per case.
func TestExhaustionByLevel(t *testing.T) {
	t.Parallel()
	for n := 0; n <= 6; n++ {
		var conds []combat.Condition
		if n > 0 {
			conds = condConds(condLevel(n))
		}
		s := condSet(t, conds)
		if got := fx.D20TestReduction(s); got != 2*n {
			t.Errorf("level %d: D20TestReduction = %d, want %d", n, got, 2*n)
		}
		if got, _ := fx.Speed(s, 40); got.Value != 40-5*n {
			t.Errorf("level %d: Speed(40) = %d, want %d", n, got.Value, 40-5*n)
		}
		if got := fx.IsDeadFromExhaustion(s); got != (n == 6) {
			t.Errorf("level %d: IsDeadFromExhaustion = %v, want %v", n, got, n == 6)
		}
	}
	if fx.D20TestReduction(condSet(t, condConds(condLevel(4)), cE)) != 0 || fx.IsDeadFromExhaustion(condSet(t, condConds(condLevel(6)), cE)) {
		t.Errorf("Exhaustion suppressed by an immunity still counts")
	}
}

// Requirement: per case
// Path: per case.
func TestInitiativeMovementTargetsAndFlags(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	if got := contributionsOf(fx.Initiative(condSet(t, condConds(combat.Incapacitated{})))); !slices.Equal(got, []contribution{{cI, dis}}) {
		t.Errorf("Initiative(Incapacitated) = %v, want Disadvantage [FR-067; p. 184]", got)
	}
	if got := contributionsOf(fx.Initiative(condSet(t, condConds(combat.Invisible{})))); !slices.Equal(got, []contribution{{cIv, adv}}) {
		t.Errorf("Initiative(Invisible) = %v, want Advantage [FR-068; p. 184]", got)
	}
	if got := fx.Initiative(condSet(t, condConds(combat.Stunned{}, combat.Invisible{}))); got.Mode() != modifier.Normal || len(got.Items()) != 2 {
		t.Errorf("Initiative(Stunned, Invisible) = %v (%v), want one of each cancelling [FR-067, FR-068]", contributionsOf(got), got.Mode())
	}
	if got := fx.Initiative(combat.ConditionSet{}); len(got.Items()) != 0 {
		t.Errorf("Initiative(no conditions) = %v, want none [FR-109]", contributionsOf(got))
	}

	frightened := condSet(t, condConds(combat.Frightened{Source: idA()}))
	if fx.CanWillinglyMoveCloserTo(frightened, idA()) != combat.ConditionPermissionForbidden || fx.CanWillinglyMoveCloserTo(frightened, idB()) != combat.ConditionPermissionAllowed {
		t.Errorf("CanWillinglyMoveCloserTo: the source of fear must be forbidden and another creature allowed [FR-059; p. 182]")
	}
	if fx.CanWillinglyMoveCloserTo(combat.ConditionSet{}, idA()) != combat.ConditionPermissionAllowed {
		t.Errorf("CanWillinglyMoveCloserTo with no conditions must be allowed [FR-109]")
	}

	invisible := condSet(t, condConds(combat.Invisible{}))
	if fx.IsValidTarget(invisible, combat.SightRelationCannotSee) != combat.ConditionPermissionForbidden ||
		fx.IsValidTarget(invisible, combat.SightRelationCanSee) != combat.ConditionPermissionAllowed ||
		fx.IsValidTarget(combat.ConditionSet{}, combat.SightRelationCannotSee) != combat.ConditionPermissionAllowed {
		t.Errorf("IsValidTarget: Invisible and cannot see must be forbidden, can see and visible creatures allowed [FR-069, FR-126; p. 184]")
	}

	for name, c := range map[string]struct {
		conds []combat.Condition
		want  combat.ConditionFlags
		incap bool
	}{
		"no conditions": {nil, combat.ConditionFlags{Actions: combat.ConditionActionAvailabilityAvailable, Concentration: combat.ConditionConcentrationIntact,
			Speech: combat.SpeechAble, HeldItems: combat.HeldItemsKept, Awareness: combat.AwarenessAware, Resistance: combat.AllDamageResistanceNone}, false},
		"Incapacitated": {condConds(combat.Incapacitated{}), combat.ConditionFlags{Actions: combat.ConditionActionAvailabilityNone, Concentration: combat.ConditionConcentrationBroken,
			Speech: combat.SpeechUnable, HeldItems: combat.HeldItemsKept, Awareness: combat.AwarenessAware, Resistance: combat.AllDamageResistanceNone}, true},
		"Unconscious": {condConds(combat.Unconscious{}), combat.ConditionFlags{Actions: combat.ConditionActionAvailabilityNone, Concentration: combat.ConditionConcentrationBroken,
			Speech: combat.SpeechUnable, HeldItems: combat.HeldItemsDropped, Awareness: combat.AwarenessUnaware, Resistance: combat.AllDamageResistanceNone}, true},
		"Petrified": {condConds(combat.Petrified{}), combat.ConditionFlags{Actions: combat.ConditionActionAvailabilityNone, Concentration: combat.ConditionConcentrationBroken,
			Speech: combat.SpeechUnable, HeldItems: combat.HeldItemsKept, Awareness: combat.AwarenessAware, Resistance: combat.AllDamageResistanceAll}, true},
		"Blinded": {condConds(combat.Blinded{}), combat.ConditionFlags{Actions: combat.ConditionActionAvailabilityAvailable, Concentration: combat.ConditionConcentrationIntact,
			Speech: combat.SpeechAble, HeldItems: combat.HeldItemsKept, Awareness: combat.AwarenessAware, Resistance: combat.AllDamageResistanceNone}, false},
	} {
		s := condSet(t, c.conds)
		if got := fx.ActiveFlags(s); got != c.want {
			t.Errorf("%s: ActiveFlags = %+v, want %+v [FR-064 to FR-066, FR-082, FR-099, FR-104, FR-109]", name, got, c.want)
		}
		if got := fx.IsIncapacitated(s); got != c.incap {
			t.Errorf("%s: IsIncapacitated = %v, want %v [FR-064]", name, got, c.incap)
		}
	}
}

// Requirement: doc/specs/conditions.md FR-111, FR-112 - queries do not alter the set, are deterministic and return copies:
// repeating a query gives the same answer and changing a returned collection changes nothing.
// Path: unhappy (immutability).
func TestQueriesDoNotAlterTheSet(t *testing.T) {
	t.Parallel()
	s := condSet(t, allFifteen(), cD)
	before := s
	facts := attackFacts(5, combat.SightRelationCannotSee, combat.SightRelationCannotSee)
	a, _ := fx.AttackRoll(s, s, facts)
	b, _ := fx.AttackRoll(s, s, facts)
	if !slices.Equal(contributionsOf(a), contributionsOf(b)) || a.Mode() != b.Mode() {
		t.Errorf("repeating AttackRoll gave a different answer")
	}
	a.Items()[0] = combat.RollContribution{}
	sp1, _ := fx.Speed(s, 30)
	sp2, _ := fx.Speed(s, 30)
	f1, f2 := fx.ActiveFlags(s), fx.ActiveFlags(s)
	if sp1 != sp2 || s != before || f1 != f2 {
		t.Errorf("queries are not deterministic or changed the set")
	}
}

// Requirement: doc/specs/conditions.md FR-057, FR-058, FR-070, FR-072, FR-126, section 5 (facts) - an unset sight relation,
// line of sight, requirement or damage kind counts as not seeing, not in line of sight, no requirement and not damaging.
// Path: unhappy (invalid input).
func TestUnsetFactsCountAsNotSeeingAndNotInLineOfSight(t *testing.T) {
	t.Parallel()
	frightened := condSet(t, condConds(combat.Frightened{Source: idA()}))
	if got, _ := fx.AttackRoll(frightened, combat.ConditionSet{}, combat.ConditionAttackFacts{AttackerID: idA(), TargetID: idB()}); len(got.Items()) != 0 {
		t.Errorf("AttackRoll with an unset line of sight = %v, want none", contributionsOf(got))
	}
	if got := fx.AbilityCheck(frightened, 0); len(got.Items()) != 0 {
		t.Errorf("AbilityCheck with an unset line of sight = %v, want none", contributionsOf(got))
	}
	invisible := condSet(t, condConds(combat.Invisible{}))
	facts := combat.ConditionAttackFacts{AttackerID: idA(), TargetID: idB()}
	if got, _ := fx.AttackRoll(combat.ConditionSet{}, invisible, facts); got.Mode() != modifier.WithDisadvantage {
		t.Errorf("attack on Invisible with an unset sight relation = %v, want Disadvantage", got.Mode())
	}
	if got, _ := fx.AttackRoll(invisible, combat.ConditionSet{}, facts); got.Mode() != modifier.WithAdvantage {
		t.Errorf("attack by Invisible with an unset sight relation = %v, want Advantage", got.Mode())
	}
	if fx.IsValidTarget(invisible, 0) != combat.ConditionPermissionForbidden {
		t.Errorf("IsValidTarget with an unset sight relation must be forbidden")
	}
	if fx.AbilityCheckEffect(condSet(t, condConds(combat.Blinded{})), 0) != combat.AbilityCheckEffectUnaffected {
		t.Errorf("AbilityCheckEffect with an unset requirement must be unaffected")
	}
	charmed := condSet(t, condConds(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}))
	if fx.CanTarget(charmed, idA(), 0) != combat.ConditionPermissionAllowed {
		t.Errorf("CanTarget with an unset damage kind must be allowed")
	}
}

// Requirement: doc/specs/conditions.md FR-052, FR-053, FR-054, FR-059, FR-061, FR-062, FR-141, FR-048, FR-055, FR-056 - the
// charmer, the source of fear, the grappler and the sense are matched by identity, whichever way their ids and values order,
// and a condition that does not apply restricts nothing.
// Path: unhappy (boundary).
func TestIdentityMatchesDoNotDependOnOrdering(t *testing.T) {
	t.Parallel()
	byB := condSet(t, condConds(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idB()}}))
	if fx.CanTarget(byB, idA(), combat.EffectDamageDamaging) != combat.ConditionPermissionAllowed || len(fx.SocialCheck(byB, idA()).Items()) != 0 {
		t.Errorf("a creature charmed by b may attack a and gives a no social Advantage")
	}
	if fx.CanTarget(byB, idB(), combat.EffectDamageDamaging) != combat.ConditionPermissionForbidden || len(fx.SocialCheck(byB, idB()).Items()) != 1 {
		t.Errorf("a creature charmed by b may not attack b, and b has social Advantage")
	}
	item, _ := combat.NewCharmedByItem("sword")
	if fx.CanTarget(condSet(t, condConds(combat.Charmed{Charmer: item})), "", combat.EffectDamageDamaging) != combat.ConditionPermissionAllowed ||
		len(fx.SocialCheck(condSet(t, condConds(combat.Charmed{Charmer: item})), "").Items()) != 0 {
		t.Errorf("an item charmer matched the zero id")
	}
	blinded := condSet(t, condConds(combat.Blinded{}))
	if fx.CanTarget(blinded, idA(), combat.EffectDamageDamaging) != combat.ConditionPermissionAllowed {
		t.Errorf("a creature that is not charmed was restricted")
	}
	if fx.CanWillinglyMoveCloserTo(condSet(t, condConds(combat.Frightened{Source: idB()})), idA()) != combat.ConditionPermissionAllowed {
		t.Errorf("a creature frightened by b may move closer to a")
	}
	grappledByB := condSet(t, condConds(combat.Grappled{Grappler: idB()}))
	facts := combat.ConditionAttackFacts{AttackerID: idA(), TargetID: idA(), AttackerFearSource: combat.LineOfSightBlocked}
	if got, _ := fx.AttackRoll(grappledByB, combat.ConditionSet{}, facts); got.Mode() != modifier.WithDisadvantage {
		t.Errorf("a creature grappled by b attacking a = %v, want Disadvantage", got.Mode())
	}
	if fx.AbilityCheckEffect(condSet(t, condConds(combat.Deafened{})), combat.CheckRequirementNone) != combat.AbilityCheckEffectUnaffected {
		t.Errorf("a Deafened creature fails a check that needs no sense")
	}
}
