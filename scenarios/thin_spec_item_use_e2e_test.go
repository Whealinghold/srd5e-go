package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/item-use-effects.md FR-004, FR-006, FR-009, FR-013 (SRD 5.2.1, "Alchemist's Fire", p. 94) - Alchemist's Fire thrown beyond its range is not possible and rolls no die; thrown in range at a target that makes the DC 13 save it has no effect and starts no fire, so the host's RollBurningDamage for that target is rejected as not burning.
// Path: unhappy (failure carried through state).
func TestAlchemistsFireSavedAgainstLeavesTheTargetNotBurning(t *testing.T) {
	t.Parallel()
	ctx, fighter, goblin := context.Background(), combat.MustCombatantID("fighter"), combat.MustCombatantID("goblin")
	throw := func(distance, face int) combat.ResolveThrownItemResult {
		t.Helper()
		cmd, err := combat.NewResolveThrownItemCommand(combat.ResolveThrownItemInput{Kind: combat.ThrownAlchemistsFire, Thrower: fighter,
			DexterityModifier: ability.MustScore(16).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
			Target:       combat.ThrownAtCreature{ID: goblin, Size: content.SizeSmall, Type: content.CreatureTypeHumanoid},
			DistanceFeet: distance, Sight: combat.ThrownSightSeen, Cover: combat.ThrownCoverTargetable, Attempt: d20.RollAttempt{}})
		if err != nil {
			t.Fatal(err)
		}
		roller := dicetest.NewFixedRoller(face)
		use, err := combat.NewResolveThrownItem(roller)
		if err != nil {
			t.Fatal(err)
		}
		res, err := use.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		if distance > 20 && len(roller.Requests()) != 0 {
			t.Errorf("an impossible throw rolled %d dice, want none", len(roller.Requests()))
		}
		return res
	}
	if far, ok := throw(500, 12).(combat.ThrownItemNotPossible); !ok || far.Reason != combat.ThrownReasonBeyondRange {
		t.Errorf("a throw far beyond range = %#v, want not possible, beyond range", throw(500, 12))
	}
	saved, ok := throw(20, 15).(combat.ThrownItemResolved) // 15 reaches DC 13
	if !ok {
		t.Fatalf("throw = %#v, want resolved", throw(20, 15))
	}
	none, isNone := saved.Effect().(combat.ThrownNoEffect)
	if !isNone || none.Reason() != combat.ThrownNoEffectSaveSucceeded {
		t.Fatalf("effect = %#v, want no effect, save succeeded", saved.Effect())
	}
	next, err := combat.NewRollBurningDamage(dicetest.NewFixedRoller(2))
	if err != nil {
		t.Fatal(err)
	}
	turn, err := next.Run(ctx, combat.RollBurningDamageCommand{State: combat.BurningStateNotBurning})
	if err != nil {
		t.Fatal(err)
	}
	if rej, ok := turn.(combat.BurningDamageRejected); !ok || rej.Rejection != combat.BurningRejectionNotBurning {
		t.Errorf("burning damage on a target that is not burning = %#v, want rejected, not burning", turn)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-051 (SRD 5.2.1, "Healer's Kit", p. 97) - a Healer's Kit used on an Alive ally is refused, keeps its use, and the same kit then stabilizes a Dying ally: the kit value carried between the calls has 9 uses left only after the real use.
// Path: unhappy (state error), then happy.
func TestHealersKitRefusedOnAnAliveAllyKeepsItsUse(t *testing.T) {
	t.Parallel()
	medic, ally := combat.MustCombatantID("medic"), combat.MustCombatantID("ally")
	kit := combat.FreshHealersKit()
	refused, err := kit.UseToStabilize(medic, ally, itemStatus(t, 12, combat.Alive{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := refused.(combat.HealersKitRefused); !ok {
		t.Fatalf("an Alive patient = %#v, want HealersKitRefused", refused)
	}
	dying, err := combat.NewDying(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	used, err := kit.UseToStabilize(medic, ally, itemStatus(t, 0, dying))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := used.(combat.HealersKitUsed)
	if !ok || got.Kit().UsesRemaining() != 9 || got.Status().LifeState() != (combat.Stable{}) {
		t.Errorf("the Dying patient = %#v, want Stable and 9 uses left (the refusal spent none)", used)
	}
}
