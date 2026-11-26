package scenarios_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func itemStatus(t *testing.T, current int, life combat.LifeState) combat.HitPointStatus {
	t.Helper()
	s, err := combat.RestoreHitPointStatus(combat.MustHitPoints(current, 20, 0), life, combat.FallsUnconscious, combat.NoAffinities())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/item-use-effects.md section 8 (scenario), FR-004, FR-009, FR-013, FR-015, FR-060 (SRD 5.2.1, "Alchemist's Fire", p. 94) - a Fighter's Attack
// action has one attack replaced by Alchemist's Fire: the replacement is recorded by the host, the DC is 13, the target fails its save and takes 1d4 Fire through
// ApplyDamage, starts burning, and RollBurningDamage rolls the next turn's damage, through the public API only.
// Path: happy.
func TestAlchemistsFireReplacesAnAttackAndSetsTheTargetBurning(t *testing.T) {
	t.Parallel()
	ctx, fighter, goblin := context.Background(), combat.MustCombatantID("fighter"), combat.MustCombatantID("goblin")
	allowance, err := combat.NewAttackAllowance(2, 2)
	if err != nil {
		t.Fatal(err)
	}
	progress, err := combat.StartAttackActionProgress(fighter, allowance)
	if err != nil {
		t.Fatal(err)
	}
	recorded, ok := progress.RecordReplacement((combat.ThrownItemRules{}).Replacement(combat.ThrownAlchemistsFire)).(combat.AttackActionAttackRecorded)
	if !ok || recorded.Progress().AttacksRemaining() != 1 {
		t.Fatalf("replacement = %#v, want recorded with one attack left", recorded)
	}

	cmd, err := combat.NewResolveThrownItemCommand(combat.ResolveThrownItemInput{Kind: combat.ThrownAlchemistsFire, Thrower: fighter,
		DexterityModifier: ability.MustScore(16).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
		Target:       combat.ThrownAtCreature{ID: goblin, Size: content.SizeSmall, Type: content.CreatureTypeHumanoid},
		DistanceFeet: 20, Sight: combat.ThrownSightSeen, Cover: combat.ThrownCoverTargetable, Attempt: d20.RollAttempt{}})
	if err != nil {
		t.Fatal(err)
	}
	thrower, err := combat.NewResolveThrownItem(dicetest.NewFixedRoller(12, 3)) // a 12 fails the DC 13 save, then 1d4 = 3
	if err != nil {
		t.Fatal(err)
	}
	res, err := thrower.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	thrown, ok := res.(combat.ThrownItemResolved)
	if !ok {
		t.Fatalf("throw = %#v, want resolved", res)
	}
	save := thrown.Save().(combat.ThrownSaveRolled)
	effect, burning := thrown.Effect().(combat.ThrownDamageAndBurningStarted)
	if save.DifficultyClass() != 13 || save.Succeeded() || !burning || effect.Damage() != combat.MustDamage(3, content.DamageFire) || thrown.Cost() != combat.ItemActionCostReplacesOneAttack {
		t.Fatalf("throw = DC %d, succeeded %v, effect %#v, cost %v, want DC 13, a failed save and 3 Fire damage with burning", save.DifficultyClass(), save.Succeeded(), thrown.Effect(), thrown.Cost())
	}

	hit, err := combat.NewApplyDamageCommand(goblin, itemStatus(t, 7, combat.Alive{}), []combat.Damage{effect.Damage()}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil {
		t.Fatal(err)
	}
	dealt, err := combat.NewApplyDamage().Run(ctx, hit)
	if err != nil {
		t.Fatal(err)
	}
	if hp := dealt.(combat.DamageApplied).Status().HitPoints().Current(); hp != 4 {
		t.Fatalf("goblin has %d Hit Points, want 4", hp)
	}

	next, err := combat.NewRollBurningDamage(dicetest.NewFixedRoller(2))
	if err != nil {
		t.Fatal(err)
	}
	turn, err := next.Run(ctx, combat.RollBurningDamageCommand{State: combat.BurningStateBurning})
	if err != nil {
		t.Fatal(err)
	}
	if d := turn.(combat.BurningDamageRolled).Damage(); d != combat.MustDamage(2, content.DamageFire) {
		t.Errorf("burning damage = %+v, want 2 Fire", d)
	}
}

// Requirement: doc/specs/item-use-effects.md section 8 (scenario), FR-020, FR-021, FR-024, FR-026 (SRD 5.2.1, "Poison, Basic", p. 99) - Basic Poison on three arrows: one
// hits for Piercing damage and is spent, and the other two expire after 1 minute, through the public API only.
// Path: happy; unhappy (boundary).
func TestBasicPoisonOnThreeArrows(t *testing.T) {
	t.Parallel()
	archer := combat.MustCombatantID("archer")
	applied, err := (combat.BasicPoisonRules{}).ApplyToAmmunition(archer, 3)
	if err != nil {
		t.Fatal(err)
	}
	arrows := applied.Coatings()
	if _, ok := arrows[0].ExtraDamage(content.DamagePiercing).(combat.BasicPoisonExtraDamage); !ok {
		t.Fatal("a fresh coating adds no damage")
	}
	hit, err := arrows[0].AfterWeaponDamage(content.DamagePiercing, 5)
	if err != nil || hit.Coating().State() != combat.BasicPoisonDelivered {
		t.Fatalf("hit = %v, %v, want delivered", hit.Coating().State(), err)
	}
	arrows[0] = hit.Coating()
	for i := 1; i < 3; i++ {
		early, _ := arrows[i].Advance(59 * time.Second)
		late, _ := early.Coating().Advance(time.Second)
		if early.Coating().State() != combat.BasicPoisonPotent || late.Coating().State() != combat.BasicPoisonExpired || len(late.Events()) != 1 {
			t.Errorf("arrow %d: %v then %v, want potent then expired", i, early.Coating().State(), late.Coating().State())
		}
	}
	if arrows[0].State() != combat.BasicPoisonDelivered {
		t.Error("the delivered arrow changed")
	}
}

// Requirement: doc/specs/item-use-effects.md section 8 (scenario), FR-051, FR-040, FR-043 (SRD 5.2.1, "Healer's Kit", p. 97; "Potion of Healing", p. 99) - a Dying ally is
// stabilized by a Healer's Kit (a use is spent, no die) and then healed by a Potion of Healing, which makes the ally Alive, through the public API only.
// Path: happy.
func TestHealersKitThenPotionOfHealing(t *testing.T) {
	t.Parallel()
	ctx, medic, ally := context.Background(), combat.MustCombatantID("medic"), combat.MustCombatantID("ally")
	dying, err := combat.NewDying(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	out, err := combat.FreshHealersKit().UseToStabilize(medic, ally, itemStatus(t, 0, dying))
	if err != nil {
		t.Fatal(err)
	}
	stabilized, ok := out.(combat.HealersKitUsed)
	if !ok || stabilized.Kit().UsesRemaining() != 9 || stabilized.Status().LifeState() != (combat.Stable{}) {
		t.Fatalf("kit = %#v, want Stable and 9 uses left", out)
	}
	cmd, err := combat.NewUsePotionOfHealingCommand(medic, ally, stabilized.Status(), combat.AdministerPotion{DistanceFeet: 5})
	if err != nil {
		t.Fatal(err)
	}
	drinker, err := combat.NewUsePotionOfHealing(dicetest.NewFixedRoller(3, 4)) // 2d4+2 = 9
	if err != nil {
		t.Fatal(err)
	}
	res, err := drinker.Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	healed, ok := res.(combat.PotionOfHealingUsed)
	if !ok || healed.HitPointsRegained() != 9 || healed.Status().LifeState() != (combat.Alive{}) || healed.Status().HitPoints().Current() != 9 {
		t.Fatalf("potion = %#v, want 9 Hit Points regained and Alive", res)
	}
}
