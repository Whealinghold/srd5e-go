package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

func msUnlimited(name string) content.StatBlockPart {
	return luMust(content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T."}))
}

// msMultiattack is a monster whose Multiattack makes count attacks with a part.
func msMultiattack(count int) func(*content.MonsterFields) {
	return func(f *content.MonsterFields) {
		group := luMust(content.NewAttackSlotGroup(count, []string{"Claw"}))
		variant := luMust(content.NewMultiattackVariant("", []content.AttackSlotGroup{group}, nil, nil))
		plan := luMust(content.NewMultiattackPlan([]content.MultiattackVariant{variant}))
		f.Actions = []content.StatBlockPart{msUnlimited("Claw"),
			luMust(content.NewStatBlockPart(content.MultiattackName, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Plan: plan}))}
	}
}

func msPartWith(name string, usage content.PartUsage) func(*content.MonsterFields) {
	return func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{luMust(content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: usage, Text: "T."}))}
	}
}

func msSpells(frequency content.SpellFrequency, level int) func(*content.MonsterFields) {
	return func(f *content.MonsterFields) {
		group := luMust(content.NewMonsterSpellGroup(frequency, []content.MonsterSpellEntry{luMust(content.NewMonsterSpellEntry("Fixture Spell", "", level))}))
		sc := luMust(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{Ability: ability.Wisdom, Groups: []content.MonsterSpellGroup{group}}))
		f.Traits = []content.StatBlockPart{luMust(content.NewStatBlockPart("Spellcasting", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Spellcasting: sc}))}
	}
}

func msDamageDice(expression dice.Expression) func(*content.MonsterFields) {
	return func(f *content.MonsterFields) {
		damage := luMust(content.NewStatBlockDamage(0, expression, luMust(content.NewSpecificDamageType(content.DamageFire)), false))
		clause := luMust(content.NewEffectClause(content.ClauseHit, "T.", nil, []content.StatBlockDamage{damage}))
		f.Actions = []content.StatBlockPart{luMust(content.NewStatBlockPart("Burn", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Clauses: []content.EffectClause{clause}}))}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md section 8, plan-phase-4.md P13 - content numbers at both ends of their range are accepted: a Multiattack of 1 and of content.MonsterMaxNumber attacks, a Recharge range of 1-6 and of a single face, an X/Day count and Legendary Actions at the largest number, a spell cast at levels 1 and 9, At Will and N/Day spells, and Hit Point Dice and damage dice of 1 and dice.MaxCount dice.
// Path: per case.
func TestMonsterSetupAcceptsContentNumbersAtTheirRange(t *testing.T) {
	t.Parallel()
	perDay := luMust(content.NewSpellFrequencyPerDay(content.MonsterMaxNumber))
	each := luMust(content.NewSpellFrequencyPerDayEach(1))
	legendary := func(uses, inLair int) func(*content.MonsterFields) {
		return func(f *content.MonsterFields) {
			f.LegendaryActions = luMust(content.NewLegendaryActions(uses, inLair, []content.StatBlockPart{msUnlimited("Move")}))
		}
	}
	cases := []struct {
		name string
		edit func(*content.MonsterFields)
	}{
		{"Multiattack of 1", msMultiattack(1)},
		{"Multiattack of the largest count", msMultiattack(content.MonsterMaxNumber)},
		{"Recharge 1-6", msPartWith("Breath", luMust(content.NewPartRecharge(1, 6)))},
		{"Recharge 6", msPartWith("Breath", luMust(content.NewPartRecharge(6, 6)))},
		{"Recharge 1", msPartWith("Breath", luMust(content.NewPartRecharge(1, 1)))},
		{"1 a day", msPartWith("Bolt", luMust(content.NewPartPerDay(1, 0)))},
		{"largest X/Day and in Lair", msPartWith("Bolt", luMust(content.NewPartPerDay(content.MonsterMaxNumber, content.MonsterMaxNumber)))},
		{"Recharge after a rest", msPartWith("Bolt", content.PartRechargeAfterRest{})},
		{"Legendary Actions of 1", legendary(1, 0)},
		{"Legendary Actions of the largest number", legendary(content.MonsterMaxNumber, content.MonsterMaxNumber)},
		{"spell at level 1, At Will", msSpells(content.SpellFrequencyAtWill{}, 1)},
		{"spell at level 9, largest N/Day", msSpells(perDay, 9)},
		{"spell with no level, N/Day Each", msSpells(each, 0)},
		{"Hit Point Dice of 1", func(f *content.MonsterFields) {
			f.HitPoints = luMust(content.NewHitPointsEntry(1, dice.MustExpression(1, dice.D4(), 0)))
		}},
		{"Hit Point Dice at dice.MaxCount", func(f *content.MonsterFields) {
			f.HitPoints = luMust(content.NewHitPointsEntry(content.MonsterMaxNumber, dice.MustExpression(dice.MaxCount, dice.D20(), 0)))
		}},
		{"damage dice at dice.MaxCount", msDamageDice(dice.MustExpression(dice.MaxCount, dice.D12(), 0))},
		{"damage dice of 1", msDamageDice(dice.MustExpression(1, dice.D4(), 0))},
		{"a speed of the largest number", func(f *content.MonsterFields) {
			f.Speeds = []content.SpeedEntry{msSpeed(content.MonsterMaxNumber, "", content.SpeedWalk)}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if s, err := combat.NewMonsterSetup(msMonster(t, c.edit), combat.PrintedAverageHitPoints{}); err != nil || s.IsZero() {
				t.Errorf("NewMonsterSetup = %v, %v, want it built (plan P13: the number is inside its range)", s, err)
			}
		})
	}
}

// Requirement: doc/specs/monster-combatant-setup.md section 8, plan-phase-4.md P13 - a monster that is missing (the zero Monster) is not usable: ContentNumberOutOfRange, with no size problem since it has no size options.
// Path: unhappy (missing content).
func TestMonsterSetupRejectsTheZeroMonster(t *testing.T) {
	t.Parallel()
	_, err := combat.NewMonsterSetup(content.Monster{}, combat.PrintedAverageHitPoints{})
	var rej *combat.MonsterSetupError
	if !errors.As(err, &rej) || !slices.Equal(rej.Problems(), []combat.MonsterSetupProblem{combat.ContentNumberOutOfRange}) {
		t.Errorf("err = %v, want a rejection with only ContentNumberOutOfRange", err)
	}
}
