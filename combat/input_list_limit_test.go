package combat_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/magical-effects.md Limits (plan P10) - a carry-over rule reads at most 256 condition kinds and at most 256 effect ids.
// Path: unhappy (boundary).
func TestNewCarryOverExceptCaps(t *testing.T) {
	t.Parallel()
	t.Run("conditions", func(t *testing.T) {
		t.Parallel()
		runCap(t, "carry-over conditions", 256, func(n int) error {
			// A valid kind repeated n times is one kind; the cap is on the list length.
			ks := make([]content.ConditionKind, n)
			for i := range ks {
				ks[i] = content.ConditionPoisoned
			}
			_, err := combat.NewCarryOverExcept(ks, nil)
			return err
		})
	})
	t.Run("effect ids", func(t *testing.T) {
		t.Parallel()
		runCap(t, "carry-over effect ids", 256, func(n int) error {
			ids := make([]combat.OngoingEffectID, n)
			for i := range ids {
				ids[i] = meEID(fmt.Sprintf("e%d", i))
			}
			_, err := combat.NewCarryOverExcept(nil, ids)
			return err
		})
	})
}

// Requirement: doc/specs/magical-effects.md Limits (plan P10) - an illusion deceives at most 256 senses.
// Path: unhappy (boundary).
func TestNewIllusionTraitsCap(t *testing.T) {
	t.Parallel()
	runCap(t, "illusion senses", 256, func(n int) error {
		ss := make([]combat.IllusionSenseID, n)
		for i := range ss {
			ss[i] = combat.IllusionSenseID(fmt.Sprintf("s%d", i))
		}
		_, err := combat.NewIllusionTraits(ss, combat.NotInSpace{})
		return err
	})
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - a tie-break names at most 256 ids.
// Path: unhappy (boundary).
func TestNewTieBreakCap(t *testing.T) {
	t.Parallel()
	runCap(t, "tie-break ids", 256, func(n int) error {
		_, err := combat.NewTieBreak(capIDs(n))
		return err
	})
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - a swap-initiative command holds at most 256 standings.
// Path: unhappy (boundary).
func TestNewSwapInitiativeCommandCap(t *testing.T) {
	t.Parallel()
	runCap(t, "standings", 256, func(n int) error {
		ss := make([]combat.InitiativeStanding, n)
		for i, id := range capIDs(n) {
			ss[i] = combat.InitiativeStanding{Combatant: id, Count: combat.InitiativeCount(n - i)}
		}
		_, err := combat.NewSwapInitiativeCommand(ss, gid("x"), gid("y"), character.InitiativeSwap{}, trkAvail, trkAvail, combat.AllyConsentWilling)
		return err
	})
}

// Requirement: doc/specs/initiative-and-actions.md Limits (plan P10) - a move-along request holds at most 256 segments.
// Path: unhappy (boundary).
func TestNewMoveAlongCap(t *testing.T) {
	t.Parallel()
	runCap(t, "move segments", 256, func(n int) error {
		ss := make([]combat.MoveSegment, n)
		for i := range ss {
			ss[i] = trkWalk(t, 5)
		}
		_, err := combat.NewMoveAlong(ss)
		return err
	})
}

// Requirement: doc/specs/dispel-and-counterspell.md Limits (plan P10) - a registry holds at most 256 ongoing spells, and AntimagicField.Blocks and the AssessAntimagicActivity use case read at most 256 targets inside the aura.
// Path: unhappy (boundary).
func TestRegistryAndAntimagicCaps(t *testing.T) {
	t.Parallel()
	t.Run("registry", func(t *testing.T) {
		t.Parallel()
		runCap(t, "ongoing spells", 256, func(n int) error {
			held := make([]combat.HeldSpell, n)
			for i := range held {
				held[i] = dspHold(dspTarget(t, "goblin"), dspSpell(t, dspSpec{effect: fmt.Sprintf("e%d", i), spell: "bless", level: 1}))
			}
			_, err := combat.NewOngoingSpellRegistry(held)
			return err
		})
	})
	targets := func(n int) []combat.SpellTargetRef {
		out := make([]combat.SpellTargetRef, n)
		for i, id := range capIDs(n) {
			out[i] = combat.CreatureTargetRef{ID: id}
		}
		return out
	}
	t.Run("Blocks", func(t *testing.T) {
		t.Parallel()
		runCap(t, "antimagic targets inside", 256, func(n int) error {
			_, err := combat.AntimagicField{}.Blocks(combat.ActivityCastSpell, false, targets(n))
			return err
		})
	})
	t.Run("use case", func(t *testing.T) {
		t.Parallel()
		runCap(t, "antimagic targets inside", 256, func(n int) error {
			_, err := combat.NewAssessAntimagicActivity().Run(context.Background(), combat.AssessAntimagicActivityCommand{Activity: combat.ActivityCastSpell, TargetsInside: targets(n)})
			return err
		})
	})
}

// Requirement: doc/specs/revival-and-restoration-spells.md Limits (plan P10) - ReviveBySpell reads at most 256 ElapsedConditions, 256 ElapsedEffects and 256 Poisons.Effects; one more of any is ReviveBySpellLimitExceeded before anything else, and through the wired use case a refused result.
// Path: per case.
func TestReviveBySpellListCaps(t *testing.T) {
	t.Parallel()
	conds := func(n int) []content.ConditionKind {
		out := make([]content.ConditionKind, n)
		for i := range out {
			out[i] = content.ConditionBlinded
		}
		return out
	}
	effects := func(n int) []combat.OngoingEffectID {
		out := make([]combat.OngoingEffectID, n)
		for i := range out {
			out[i] = meEID(fmt.Sprintf("x%d", i))
		}
		return out
	}
	for name, edit := range map[string]func(c *combat.ReviveBySpellCommand, n int){
		"elapsed conditions": func(c *combat.ReviveBySpellCommand, n int) { c.ElapsedConditions = conds(n) },
		"elapsed effects":    func(c *combat.ReviveBySpellCommand, n int) { c.ElapsedEffects = effects(n) },
		"poison effects":     func(c *combat.ReviveBySpellCommand, n int) { c.Poisons.Effects = effects(n) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			at := rrCommand(t, combat.Resurrection)
			edit(&at, 256)
			if _, ok := combat.ReviveBySpell(at).(combat.ReviveBySpellApplied); !ok {
				t.Errorf("256 %s: result is not applied", name)
			}
			over := rrCommand(t, combat.Resurrection)
			edit(&over, 257)
			if _, ok := combat.ReviveBySpell(over).(combat.ReviveBySpellLimitExceeded); !ok {
				t.Errorf("257 %s: result is not ReviveBySpellLimitExceeded", name)
			}
			res, err := combat.NewReviveCreatureBySpell().Run(context.Background(), combat.ReviveCreatureBySpellCommand(over))
			if _, ok := res.(combat.ReviveBySpellLimitExceeded); !ok || err != nil {
				t.Errorf("wired Run = %#v, %v, want ReviveBySpellLimitExceeded and no error", res, err)
			}
		})
	}
}
