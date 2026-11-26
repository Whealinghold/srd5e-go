package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// outOfRange returns the magnitude one above content.MonsterMaxNumber, and its negative one below; at 32-bit width they
// wrap, which is out of range as well.
func outsideNumbers() (above, below int) {
	above, below = content.MonsterMaxNumber, -content.MonsterMaxNumber
	above++
	below--
	return above, below
}

// Requirement: doc/specs/monster-combatant-setup.md section 8, plan-phase-4.md P13 - each content number is checked against its range, inclusive at both ends and false one outside: a printed number, a use count, a Recharge range, an attack count, dice, Hit Points and a spell level.
// Path: per case.
func TestMonsterNumberRanges(t *testing.T) {
	t.Parallel()
	above, below := outsideNumbers()
	big := content.MonsterMaxNumber
	cases := []struct {
		name, path string
		got, want  bool
	}{
		{"printed number at the maximum", "unhappy (boundary)", monsterNumberOK(big), true},
		{"printed number at the minimum", "unhappy (boundary)", monsterNumberOK(-big), true},
		{"printed number zero", "unhappy (boundary)", monsterNumberOK(0), true},
		{"printed number above", "unhappy (boundary)", monsterNumberOK(above), false},
		{"printed number below", "unhappy (boundary)", monsterNumberOK(below), false},

		{"uses of 0", "unhappy (boundary)", monsterUsesOK(0), false},
		{"uses of -1", "unhappy (boundary)", monsterUsesOK(-1), false},
		{"uses of 1", "unhappy (boundary)", monsterUsesOK(1), true},
		{"uses at the maximum", "unhappy (boundary)", monsterUsesOK(big), true},
		{"uses above", "unhappy (boundary)", monsterUsesOK(above), false},

		{"recharge 1-6", "unhappy (boundary)", monsterRechargeOK(1, 6), true},
		{"recharge 6-6", "unhappy (boundary)", monsterRechargeOK(6, 6), true},
		{"recharge 1-1", "unhappy (boundary)", monsterRechargeOK(1, 1), true},
		{"recharge 0-6", "unhappy (boundary)", monsterRechargeOK(0, 6), false},
		{"recharge 1-7", "unhappy (boundary)", monsterRechargeOK(1, 7), false},
		{"recharge 0-0", "unhappy (boundary)", monsterRechargeOK(0, 0), false},
		{"recharge 5-4", "unhappy (invalid input)", monsterRechargeOK(5, 4), false},
		{"recharge 4-5", "happy", monsterRechargeOK(4, 5), true},

		{"attack count 0", "unhappy (boundary)", monsterMultiattackCountOK(0), false},
		{"attack count 1", "unhappy (boundary)", monsterMultiattackCountOK(1), true},
		{"attack count at the maximum", "unhappy (boundary)", monsterMultiattackCountOK(big), true},
		{"attack count above", "unhappy (boundary)", monsterMultiattackCountOK(above), false},

		{"dice count 0", "unhappy (boundary)", monsterDiceOK(0, 6), false},
		{"dice count 1", "unhappy (boundary)", monsterDiceOK(1, 6), true},
		{"dice count at dice.MaxCount", "unhappy (boundary)", monsterDiceOK(dice.MaxCount, 6), true},
		{"dice count above dice.MaxCount", "unhappy (boundary)", monsterDiceOK(dice.MaxCount+1, 6), false},
		{"die of 3 sides", "unhappy (boundary)", monsterDiceOK(1, 3), false},
		{"die of 4 sides", "unhappy (boundary)", monsterDiceOK(1, 4), true},
		{"die of 5 sides", "unhappy (boundary)", monsterDiceOK(1, 5), false},
		{"die of 8 sides", "happy", monsterDiceOK(1, 8), true},
		{"die of 10 sides", "happy", monsterDiceOK(1, 10), true},
		{"die of 12 sides", "happy", monsterDiceOK(1, 12), true},
		{"die of 20 sides", "unhappy (boundary)", monsterDiceOK(1, 20), true},
		{"die of 21 sides", "unhappy (boundary)", monsterDiceOK(1, 21), false},
		{"die of 100 sides", "unhappy (boundary)", monsterDiceOK(1, 100), false},

		{"Hit Points average 0", "unhappy (boundary)", monsterHitPointsOK(0, 1, 6), false},
		{"Hit Points average 1", "unhappy (boundary)", monsterHitPointsOK(1, 1, 6), true},
		{"Hit Points average at the maximum", "unhappy (boundary)", monsterHitPointsOK(big, 1, 6), true},
		{"Hit Points average above", "unhappy (boundary)", monsterHitPointsOK(above, 1, 6), false},
		{"Hit Points with bad dice", "unhappy (invalid input)", monsterHitPointsOK(10, 0, 6), false},

		{"per day 1, none in Lair", "happy", monsterPerDayOK(1, 0), true},
		{"per day 1, 1 in Lair", "happy", monsterPerDayOK(1, 1), true},
		{"per day at the maximum in Lair", "unhappy (boundary)", monsterPerDayOK(big, big), true},
		{"per day 0", "unhappy (boundary)", monsterPerDayOK(0, 0), false},
		{"per day 1, in Lair 0 is none", "unhappy (boundary)", monsterPerDayOK(1, 0), true},
		{"per day ok, in Lair above", "unhappy (boundary)", monsterPerDayOK(1, above), false},
		{"per day ok, in Lair negative", "unhappy (boundary)", monsterPerDayOK(1, -1), false},
		{"per day above, in Lair none", "unhappy (boundary)", monsterPerDayOK(above, 0), false},

		{"speed -1", "unhappy (boundary)", monsterSpeedFeetOK(-1), false},
		{"speed 0", "unhappy (boundary)", monsterSpeedFeetOK(0), true},
		{"speed at the maximum", "unhappy (boundary)", monsterSpeedFeetOK(big), true},
		{"speed above", "unhappy (boundary)", monsterSpeedFeetOK(above), false},

		{"spell level 0", "unhappy (boundary)", monsterSpellLevelOK(0), false},
		{"spell level 1", "unhappy (boundary)", monsterSpellLevelOK(1), true},
		{"spell level 9", "unhappy (boundary)", monsterSpellLevelOK(9), true},
		{"spell level 10", "unhappy (boundary)", monsterSpellLevelOK(10), false},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v (plan P13, %s)", c.name, c.got, c.want, c.path)
		}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md section 8, plan-phase-4.md P13 - a part with no usage, a zero X/Day or a zero Recharge (values content's constructors reject; forged here from content's zero values) is out of range, and the usages that need no number are in range.
// Path: unhappy (invalid input).
func TestMonsterUsageNumbers(t *testing.T) {
	t.Parallel()
	good := []content.PartUsage{content.PartUnlimited{}, content.PartUsageOther{}, content.PartRechargeAfterRest{}}
	for _, u := range good {
		if !monsterUsageOK(u) {
			t.Errorf("monsterUsageOK(%T) = false, want true", u)
		}
	}
	for name, u := range map[string]content.PartUsage{"nil usage": nil, "zero X/Day": content.PartPerDay{}, "zero Recharge": content.PartRecharge{}} {
		if monsterUsageOK(u) {
			t.Errorf("monsterUsageOK(%s) = true, want false", name)
		}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md section 8, plan-phase-4.md P13 - the difference of a printed modifier from its ability modifier is checked to fit an int at the limits of int64 arithmetic.
// Path: unhappy (boundary).
func TestDifferenceFits(t *testing.T) {
	t.Parallel()
	if !differenceFits(0, 0) || !differenceFits(5, -5) || !differenceFits(-5, 5) {
		t.Error("an ordinary difference does not fit")
	}
	if got := differenceFits(content.MonsterMaxNumber, -5); got != (intSize() == 64) {
		t.Errorf("differenceFits(max, -5) = %v at int size %d", got, intSize())
	}
}

// intSize returns the width of int in bits.
func intSize() int {
	const maxUint = ^uint(0)
	return 32 << (maxUint >> 63)
}

// Requirement: doc/specs/monster-combatant-setup.md section 8, plan-phase-4.md P13 - modifiersFit is false for the zero ability block, for a save whose difference from its ability modifier does not fit, and for a skill whose difference does not fit, and true otherwise; the overflow is forged through the fits parameter so the rows run at any int width.
// Path: per case.
func TestModifiersFit(t *testing.T) {
	t.Parallel()
	scores, saves := map[ability.Ability]ability.Score{}, map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	sc, err := ability.NewScores(scores)
	if err != nil {
		t.Fatal(err)
	}
	block, err := content.NewAbilityBlock(sc, saves)
	if err != nil {
		t.Fatal(err)
	}
	zero := func(ability.Ability) int { return 0 }
	always := func(int, int) bool { return true }
	// A forged width: only a printed 7 does not fit.
	forged := func(printed, _ int) bool { return printed != 7 }
	cases := []struct {
		name, path string
		block      content.AbilityBlock
		skills     map[content.Skill]int
		fits       func(int, int) bool
		want       bool
	}{
		{"every difference fits", "happy", block, map[content.Skill]int{content.SkillStealth: 3}, always, true},
		{"no skills", "unhappy (empty case)", block, nil, always, true},
		{"zero ability block", "unhappy (invalid input)", content.AbilityBlock{}, nil, always, false},
		{"a save that overflows", "unhappy (boundary)", block, nil, func(p, m int) bool { return false }, false},
		{"a skill that overflows", "unhappy (boundary)", block, map[content.Skill]int{content.SkillStealth: 7}, forged, false},
		{"a skill that fits beside the forged width", "unhappy (boundary)", block, map[content.Skill]int{content.SkillStealth: 6}, forged, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := modifiersFit(c.block, zero, c.skills, c.fits); got != c.want {
				t.Errorf("modifiersFit = %v, want %v [%s]", got, c.want, c.path)
			}
		})
	}
}
