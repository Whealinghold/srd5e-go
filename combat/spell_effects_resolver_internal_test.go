package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// forbidRolls is a spellRolls that fails the test when anything is rolled.
type forbidRolls struct{ t *testing.T }

func (f forbidRolls) rollAmount(dice.Expression) ([]int, int, error) {
	f.t.Helper()
	f.t.Error("a die was rolled")
	return nil, 0, nil
}

func (f forbidRolls) attackRoll([]modifier.RollCircumstance, d20.RollModifiers, d20.ArmorClass, int) (d20.AttackRollResult, error) {
	f.t.Error("an attack was rolled")
	return d20.AttackRollResult{}, nil
}

func (f forbidRolls) rollSpellDamage(DamageSpec, int, bool) ([]Damage, [][]int, error) {
	f.t.Error("damage was rolled")
	return nil, nil, nil
}

func (f forbidRolls) resolveSave(d20.SaveAttempt, d20.DifficultyClass) (UnarmedStrikeSave, error) {
	f.t.Error("a save was rolled")
	return UnarmedStrikeSaveForced{}, nil
}

// Requirement: spell-effects.md FR-022, R24 - an AttackPlanAutomaticMiss is a miss with no d20 rolled that applies OnMiss and records the reason; CircumstanceCollector returns it only for weapon attacks today (underwater), so the branch is reached here with the plan injected.
// Path: unhappy (rule interaction).
func TestAutomaticMissAppliesOnMissWithoutRolling(t *testing.T) {
	t.Parallel()
	frightened, err := content.NewSpellAppliesCondition(content.ConditionFrightened, content.ForSpellDuration{})
	if err != nil {
		t.Fatal(err)
	}
	amount, _ := content.NewRolledSpellAmount(dice.MustParseExpression("1d6"))
	scalable, _ := content.NewScalableAmount(amount, nil)
	fire, _ := content.NewFixedDamageType(content.DamageFire)
	hitDamage, _ := content.NewSpellDealsDamage(scalable, fire, content.AddsNothing)
	res, err := content.NewAttackRollResolution(content.SpellAttackRanged, []content.SpellOutcome{hitDamage}, []content.SpellOutcome{frightened})
	if err != nil {
		t.Fatal(err)
	}
	p := &spellPlan{c: spellCast{caster: "caster"}}
	target := TargetEffects{target: CreatureTargetRef{ID: "a"}}
	var inst ResolvedInstance
	plan := AttackPlanAutomaticMiss{Reason: AutomaticMissReasonUnderwaterBeyondNormalRange}
	if err := p.attackInstance(forbidRolls{t}, spellPartPlan{index: 2}, res, SpellAttackFactsParams{}, plan, 0, &target, &inst); err != nil {
		t.Fatalf("attackInstance: %v", err)
	}
	if reason, miss := inst.AutomaticMiss(); !miss || reason != AutomaticMissReasonUnderwaterBeyondNormalRange || inst.hit || inst.rolled {
		t.Errorf("evidence = %+v, want an automatic miss with no roll", inst)
	}
	if len(target.damage) != 0 || len(target.conditions) != 1 || target.conditions[0].kind != content.ConditionFrightened || target.conditions[0].part != 2 {
		t.Errorf("plan = %+v, want only the OnMiss condition from part 2", target)
	}
}

// Requirement: spell-effects.md FR-043 - n is the larger of 0 and the cast level minus AboveLevel, and 0 for a cast with no slot level.
// Path: unhappy (boundary).
func TestSlotSteps(t *testing.T) {
	t.Parallel()
	cases := []struct{ castAt, above, want int }{{3, 3, 0}, {4, 3, 1}, {9, 3, 6}, {2, 3, 0}, {0, 3, 0}, {0, 0, 0}, {1, 0, 1}}
	for _, c := range cases {
		if got := slotSteps(c.castAt, c.above); got != c.want {
			t.Errorf("slotSteps(%d, %d) = %d, want %d", c.castAt, c.above, got, c.want)
		}
	}
}
