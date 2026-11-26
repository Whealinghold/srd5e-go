package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// luActions returns n Actions, each with usage.
func luActions(n int, usage content.PartUsage) []content.StatBlockPart {
	out := make([]content.StatBlockPart, n)
	for i := range out {
		out[i] = luPart(fmt.Sprintf("Action %d", i), usage)
	}
	return out
}

// Requirement: doc/specs/monster-limited-use.md Limits (plan P10) - StartLimitedUse takes at most 64 parts, 64 spell groups and 64 spells per stat block; one more of any is a LimitExceededError.
// Path: unhappy (boundary).
func TestStartLimitedUseCaps(t *testing.T) {
	t.Parallel()
	start := func(m content.Monster) error {
		_, err := combat.StartLimitedUse(m, combat.MustCombatantID(luSelf), combat.OutsideLair)
		return err
	}
	t.Run("parts", func(t *testing.T) {
		t.Parallel()
		runCap(t, "monster parts", 64, func(n int) error {
			return start(luMonster(t, func(f *content.MonsterFields) { f.Actions = luActions(n, content.PartUnlimited{}) }))
		})
	})
	t.Run("parts are counted over every section", func(t *testing.T) {
		t.Parallel()
		err := start(luMonster(t, func(f *content.MonsterFields) {
			f.Traits, f.Actions, f.BonusActions = luActions(20, content.PartUnlimited{}), luActions(20, content.PartUnlimited{}), luActions(20, content.PartUnlimited{})
			f.Reactions = luActions(4, content.PartUnlimited{})
			f.LegendaryActions = luMust(content.NewLegendaryActions(3, 0, luActions(1, content.PartUnlimited{})))
		}))
		wantLimit(t, err, "monster parts", 64)
	})
	t.Run("spells", func(t *testing.T) {
		t.Parallel()
		runCap(t, "monster spells", 64, func(n int) error {
			names := make([]string, n)
			for i := range names {
				names[i] = fmt.Sprintf("spell_%d", i)
			}
			return start(luMonster(t, func(f *content.MonsterFields) {
				f.Traits = []content.StatBlockPart{luSpellcasting(luSpells(content.SpellFrequencyAtWill{}, names...))}
			}))
		})
	})
	t.Run("spells are counted over every group", func(t *testing.T) {
		t.Parallel()
		err := start(luMonster(t, func(f *content.MonsterFields) {
			f.Traits = []content.StatBlockPart{luSpellcasting(luSpells(content.SpellFrequencyAtWill{}, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
				"a1", "b1", "c1", "d1", "e1", "f1", "g1", "h1", "i1", "j1", "k1", "l1", "m1", "n1", "o1", "p1", "q1", "r1", "s1", "t1", "u1", "v1", "w1", "x1", "y1", "z1", "a2", "b2", "c2", "d2", "e2", "f2", "g2", "h2", "i2", "j2"),
				luSpells(luPerDayEach(1), "a", "b", "c"))}
		}))
		wantLimit(t, err, "monster spells", 64)
	})
	t.Run("spell groups", func(t *testing.T) {
		t.Parallel()
		runCap(t, "monster spell groups", 64, func(n int) error {
			groups := make([]content.MonsterSpellGroup, n)
			for i := range groups {
				groups[i] = luSpells(content.SpellFrequencyAtWill{}, "light")
			}
			return start(luMonster(t, func(f *content.MonsterFields) { f.Traits = []content.StatBlockPart{luSpellcasting(groups...)} }))
		})
	})
}

// Requirement: doc/specs/monster-limited-use.md Limits (plan P10) - RechargeRollsNeeded returns at most 64 parts (a stat block holds at most 64) and StartOfOwnTurn takes at most 64 rolls; a map of 65 is a LimitExceededError and the state is unchanged.
// Path: unhappy (boundary).
func TestStartOfOwnTurnRollCap(t *testing.T) {
	t.Parallel()
	m := luMonster(t, func(f *content.MonsterFields) { f.Actions = luActions(64, luRecharge(5, 6)) })
	s := luStart(t, m, combat.OutsideLair)
	rolls := map[combat.LimitedUsePartRef]dice.DieRoll{}
	for i := 0; i < 64; i++ {
		ref := luRef(t, combat.MonsterPartActions, i)
		s = luSpent(t, s, ref)
		rolls[ref] = luRoll(6)
	}
	if got := len(s.RechargeRollsNeeded()); got != 64 {
		t.Fatalf("RechargeRollsNeeded() = %d parts, want 64", got)
	}
	runCap(t, "recharge rolls", 64, func(n int) error {
		in := map[combat.LimitedUsePartRef]dice.DieRoll{}
		for ref, r := range rolls {
			in[ref] = r
		}
		if n > 64 {
			in[luRef(t, combat.MonsterPartActions, 64)] = luRoll(1)
		}
		before := s
		_, err := s.StartOfOwnTurn(in)
		if !s.Equal(before) || len(s.Entries()) != 64 {
			t.Error("StartOfOwnTurn changed the receiver")
		}
		return err
	})
}
