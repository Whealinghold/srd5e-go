package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-spellcasting.md Limits (plan P10) - NewMonsterSpellcaster takes a trait of at most 64 groups and 64 entries; one more of either is a LimitExceededError.
// Path: unhappy (boundary).
func TestMonsterSpellcasterCaps(t *testing.T) {
	t.Parallel()
	build := func(groups ...content.MonsterSpellGroup) error {
		m := luMonster(t, func(f *content.MonsterFields) { f.Traits = []content.StatBlockPart{luSpellcasting(groups...)} })
		_, err := combat.NewMonsterSpellcaster(m, mscID())
		return err
	}
	t.Run("groups", func(t *testing.T) {
		t.Parallel()
		runCap(t, "spellcasting groups", 64, func(n int) error {
			groups := make([]content.MonsterSpellGroup, n)
			for i := range groups {
				groups[i] = luSpells(content.SpellFrequencyAtWill{}, "light")
			}
			return build(groups...)
		})
	})
	t.Run("entries", func(t *testing.T) {
		t.Parallel()
		runCap(t, "spellcasting entries", 64, func(n int) error {
			names := make([]string, n)
			for i := range names {
				names[i] = fmt.Sprintf("spell_%d", i)
			}
			return build(luSpells(content.SpellFrequencyAtWill{}, names...))
		})
	})
}

// Requirement: doc/specs/monster-spellcasting.md Limits (plan P10) - Cast and CompleteCasting take at most 256 targets; the 257th is a LimitExceededError and both states are unchanged.
// Path: unhappy (boundary).
func TestMonsterCastTargetCap(t *testing.T) {
	t.Parallel()
	rig := mscNewRig(t, mscMonster(t, nil))
	fireball := mscSpells()["fireball"]
	for name, run := range map[string]func(...combat.CombatantID) error{
		"Cast": func(ts ...combat.CombatantID) error { return rig.cast(mscList(t, mscFireball), fireball, ts...).Err },
		"CompleteCasting": func(ts ...combat.CombatantID) error {
			return rig.complete(mscList(t, mscFireball), fireball, ts...).Err
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runCap(t, "monster cast targets", 256, func(n int) error {
				targets := make([]combat.CombatantID, n)
				for i := range targets {
					targets[i] = combat.MustCombatantID(fmt.Sprintf("t%d", i))
				}
				return run(targets...)
			})
		})
	}
}
