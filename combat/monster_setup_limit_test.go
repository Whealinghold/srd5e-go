package combat_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// msPlanActions are the Actions of a monster whose Multiattack has variants variants, each of groups groups of one attack
// with its own part and replacements rules replacing an attack with Spit, and the parts those name.
func msPlanActions(tb testing.TB, variants, groups, replacements int) []content.StatBlockPart {
	tb.Helper()
	actions := []content.StatBlockPart{msUnlimited("Spit")}
	gs := make([]content.AttackSlotGroup, groups)
	for i := range gs {
		name := fmt.Sprintf("Claw %d", i)
		actions = append(actions, msUnlimited(name))
		gs[i] = luMust(content.NewAttackSlotGroup(1, []string{name}))
	}
	rs := make([]content.ReplacementRule, replacements)
	for i := range rs {
		rs[i] = luMust(content.NewReplacementRule(1, content.ReplacementScopeAnyAttack{},
			[]content.ReplacementOption{luMust(content.NewReplacementOption("Spit", "", 0))}, content.ReplaceOnce))
	}
	vs := make([]content.MultiattackVariant, variants)
	for i := range vs {
		requires := ""
		if i > 0 {
			requires = "Spit"
		}
		vs[i] = luMust(content.NewMultiattackVariant(requires, gs, nil, rs))
	}
	return append(actions, luMust(content.NewStatBlockPart(content.MultiattackName, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Plan: luMust(content.NewMultiattackPlan(vs))})))
}

// Requirement: doc/specs/monster-combatant-setup.md Limits (plan P10) - NewMonsterSetup and SetUpMonsterCombatant take a stat block of at most 64 parts, 64 spells, 64 spell groups, 64 Multiattack variants, 64 attack groups and 64 replacement rules; one more of any is the problem SetupLimitExceeded (the shared limit_exceeded), never ContentNumberOutOfRange, and nothing is rolled.
// Path: per case.
func TestMonsterSetupCaps(t *testing.T) {
	t.Parallel()
	names := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("spell_%d", i)
		}
		return out
	}
	cases := []struct {
		name string
		edit func(n int) func(*content.MonsterFields)
	}{
		{"parts", func(n int) func(*content.MonsterFields) {
			return func(f *content.MonsterFields) { f.Actions = luActions(n, content.PartUnlimited{}) }
		}},
		{"spells", func(n int) func(*content.MonsterFields) {
			return func(f *content.MonsterFields) {
				f.Traits = []content.StatBlockPart{luSpellcasting(luSpells(content.SpellFrequencyAtWill{}, names(n)...))}
			}
		}},
		{"spell groups", func(n int) func(*content.MonsterFields) {
			return func(f *content.MonsterFields) {
				groups := make([]content.MonsterSpellGroup, n)
				for i := range groups {
					groups[i] = luSpells(content.SpellFrequencyAtWill{}, "light")
				}
				f.Traits = []content.StatBlockPart{luSpellcasting(groups...)}
			}
		}},
		{"variants", func(n int) func(*content.MonsterFields) {
			return func(f *content.MonsterFields) { f.Actions = msPlanActions(t, n, 1, 0) }
		}},
		{"attack groups", func(n int) func(*content.MonsterFields) {
			// 8 variants of 8 groups is 64; 13 of 5 is 65.
			return func(f *content.MonsterFields) {
				if n == 64 {
					f.Actions = msPlanActions(t, 8, 8, 0)
				} else {
					f.Actions = msPlanActions(t, 13, 5, 0)
				}
			}
		}},
		{"replacement rules", func(n int) func(*content.MonsterFields) {
			return func(f *content.MonsterFields) { f.Actions = msPlanActions(t, 1, 1, n) }
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			atCap := msMonster(t, c.edit(64))
			if s, err := combat.NewMonsterSetup(atCap, combat.PrintedAverageHitPoints{}); err != nil || s.IsZero() {
				t.Errorf("64 %s: err = %v, want a setup", c.name, err)
			}
			over := msMonster(t, c.edit(65))
			_, err := combat.NewMonsterSetup(over, combat.PrintedAverageHitPoints{})
			var rej *combat.MonsterSetupError
			if !errors.As(err, &rej) || !slices.Equal(rej.Problems(), []combat.MonsterSetupProblem{combat.SetupLimitExceeded}) || combat.SetupLimitExceeded.String() != "limit_exceeded" {
				t.Fatalf("65 %s: err = %v, want only SetupLimitExceeded", c.name, err)
			}
			roller := dicetest.NewMinimumRoller()
			u := luMust(combat.NewSetUpMonsterCombatant(roller))
			res, err := u.Run(t.Context(), combat.SetUpMonsterCombatantCommand{Monster: over, HitPoints: combat.RollHitPointDice{}})
			inv, ok := res.(combat.SetUpMonsterCombatantInvalid)
			if err != nil || !ok || !slices.Equal(inv.Problems(), []combat.MonsterSetupProblem{combat.SetupLimitExceeded}) || len(roller.Requests()) != 0 {
				t.Errorf("65 %s: Run = %#v, %v, requests %v, want Invalid(SetupLimitExceeded) and no roll", c.name, res, err, roller.Requests())
			}
		})
	}
}
