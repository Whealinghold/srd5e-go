package combat_test

import (
	"fmt"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// A Raise Dead cast on a creature dead for three days returns it with 1 Hit Point and an ordeal of -4 to D20 Tests that one Long Rest shrinks to -3.
func ExampleReviveBySpell() {
	dead, _ := combat.NewDead(combat.DeathSavingThrowFailures)
	status, _ := combat.RestoreHitPointStatus(combat.MustHitPoints(0, 20, 0), dead, combat.FallsUnconscious, combat.NoAffinities())
	res := combat.ReviveBySpell(combat.ReviveBySpellCommand{
		Spell: combat.RaiseDead, Subject: combat.MustCombatantID("hero"), Status: status, Consent: combat.SpiritAccepts,
		Circumstances: combat.MustDeathCircumstances(3*24*time.Hour, false, false, combat.Intact),
	})
	applied := res.(combat.ReviveBySpellApplied)
	fmt.Println(applied.Revival.Status().HitPoints().Current(), applied.Ordeal.Modifier().Value(), applied.Ordeal.AfterRest(character.LongRest).Modifier().Value())
	// Output: 1 -4 -3
}

// Aid cast with a level 3 slot raises the Hit Point maximum and the current Hit Points by 10; two casts of Aid take the larger, not the sum.
func ExampleIncreaseMaximum() {
	status := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	aid3, _ := combat.AidIncrease("aid-1", 3)
	res, _ := combat.IncreaseMaximum(combat.MustCombatantID("hero"), status, combat.HitPointMaximumIncreases{}, aid3)
	applied := res.(combat.IncreaseMaximumApplied)
	fmt.Println(applied.Status().HitPoints().Maximum(), applied.Status().HitPoints().Current())

	aid2, _ := combat.AidIncrease("aid-2", 2)
	res, _ = combat.IncreaseMaximum(combat.MustCombatantID("hero"), applied.Status(), applied.Ledger(), aid2)
	fmt.Println(res.(combat.IncreaseMaximumApplied).Status().HitPoints().Maximum())
	// Output:
	// 30 30
	// 30
}

// Death Ward leaves a creature that damage would drop to 0 Hit Points at 1 Hit Point, and the ward is spent.
func ExampleApplyDamageWithDeathWard() {
	status := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
	tr, _ := status.ApplyDamage(combat.MustCombatantID("hero"), []combat.Damage{combat.MustDamage(15, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit, nil)
	status = tr.(combat.HitPointTransitionApplied).Status()
	res, _ := combat.ApplyDamageWithDeathWard(combat.MustCombatantID("hero"), status, []combat.Damage{combat.MustDamage(50, content.DamageFire)}, nil,
		combat.OtherSource, combat.NormalHit, nil, combat.WardActive)
	guarded := res.(combat.GuardedDamageApplied)
	fmt.Println(guarded.Status().HitPoints().Current(), guarded.Ward())
	// Output: 1 spent
}
