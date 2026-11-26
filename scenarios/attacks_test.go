package scenarios_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// attackName switches exhaustively over the closed ResolveAttackResult set, as a consumer would (FR-108).
func attackName(r combat.ResolveAttackResult) string {
	switch r.(type) {
	case combat.AttackResolved:
		return "resolved"
	case combat.AttackAutomaticallyMissed:
		return "automatic miss"
	case combat.AttackNotPossible:
		return "not possible"
	}
	panic(fmt.Errorf("unreachable %T", r))
}

// unarmedName switches exhaustively over the closed ResolveUnarmedStrikeResult set (FR-108).
func unarmedName(r combat.ResolveUnarmedStrikeResult) string {
	switch r.(type) {
	case combat.UnarmedStrikeDamageResolved:
		return "damage"
	case combat.UnarmedStrikeGrappleResolved:
		return "grapple"
	case combat.UnarmedStrikeShoveResolved:
		return "shove"
	case combat.UnarmedStrikeNotPossible:
		return "not possible"
	}
	panic(fmt.Errorf("unreachable %T", r))
}

// Requirement: doc/specs/attacks.md section 8 scenario 1, FR-108, FR-002, FR-056, FR-057, FR-024, FR-015 - through the public
// API only: a ranged weapon attack at long range behind Half Cover rolls two d20s (Disadvantage), hits the effective AC of 17
// with a Critical Hit, and rolls its dice, the extra dice too, twice; the damage then goes to ApplyDamage. The plan for the
// weapon comes from character.PlanWeaponAttack and is mapped into the attack command.
// Path: happy.
func TestRangedCriticalHitBehindHalfCoverAtLongRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec := telemetrytest.NewRecorder()
	tel := combat.WithTelemetry(rec.Telemetry())

	// character: plan a Shortsword melee attack (Dexterity 16, proficient) and map its data into the command.
	finesse := must2(content.NewPlainUse(content.WeaponPropertyFinesse))
	d6 := must2(content.NewWeaponDiceDamage(dice.MustExpression(1, dice.D6(), 0), content.DamagePiercing))
	shortsword := weapon(t, "shortsword", content.WeaponMartial, d6, content.MasteryVex, finesse)
	planned := character.PlanWeaponAttack(character.WeaponAttackInput{
		Weapon: shortsword, Mode: character.MeleeAttackMode{}, Strength: must2(ability.NewScore(10)), Dexterity: must2(ability.NewScore(16)),
		Preference: character.MustChosenAbility(ability.Dexterity), Wielding: character.WeaponWielding{Grip: character.GripOneHanded},
		Proficiencies: character.NewWeaponProficiencies(character.CategoryProficiency(content.WeaponMartial)),
	}).(character.WeaponAttackPlanned).Plan
	rolls := d20.NewRollModifiers(modifier.New(planned.AbilityModifier().Value()))
	if planned.Proficiency() == character.WeaponProficient {
		rolls = rolls.WithProficiencyBonus(modifier.New(character.ProficiencyBonusFor(character.MustLevel(1)).Value()))
	}
	faces := planned.Damage().(content.WeaponDiceDamage)
	main := combat.MustDiceDamage(faces.Dice().Count(), faces.Dice().Die(), faces.DamageType(), 0, planned.DamageModifier() == character.AddModifierAlways)
	sneak := combat.MustDiceDamage(2, dice.D6(), content.DamagePoison, 0, false) // extra dice of another source (FR-057)
	cmd, err := combat.NewResolveAttackCommand(combat.ResolveAttackParams{
		Form: combat.RangedWeaponAttack, ArmorClass: 15, Modifiers: rolls, Distance: 100, Range: combat.MustAttackRange(80, 320),
		Cover:  []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}},
		Damage: combat.MustDamageSpec(main, sneak),
	})
	if err != nil {
		t.Fatal(err)
	}

	// 100 feet is beyond the normal range 80: two d20s, the lower kept (20, 20 gives a natural 20). Damage: 2d6 (1d6 doubled)
	// then 4d6 (2d6 doubled).
	roller := dicetest.NewFixedRoller(20, 20, 2, 3, 1, 2, 3, 4)
	attack, err := combat.NewResolveAttack(roller, tel)
	if err != nil {
		t.Fatal(err)
	}
	res, err := attack.Run(ctx, cmd)
	if err != nil || attackName(res) != "resolved" {
		t.Fatalf("= %v, %v", res, err)
	}
	r := res.(combat.AttackResolved)
	if !r.CriticalHit() || r.EffectiveArmorClass() != 17 || r.Attack().Roll().Mode() != modifier.WithDisadvantage ||
		r.Attack().Total() != 20+3+2 || r.Delivery() != combat.AttackDeliveryRanged {
		t.Fatalf("attack = hit %v crit %v effective %d mode %v total %d", r.Hit(), r.CriticalHit(), r.EffectiveArmorClass(), r.Attack().Roll().Mode(), r.Attack().Total())
	}
	got := r.Damage()
	if len(got) != 2 || got[0] != combat.MustDamage(5+3, content.DamagePiercing) || got[1] != combat.MustDamage(1+2+3+4, content.DamagePoison) {
		t.Fatalf("damage = %v, want 5+3 Piercing (2d6 and the ability modifier once) and 10 Poison", got)
	}
	for _, m := range rec.Meter.Measurements() {
		if len(m.Name) > 15 && m.Name[:15] == "srd5e.d20_test." {
			t.Errorf("a combat call recorded the d20 signal %s", m.Name)
		}
	}

	// The Damage then goes through the damage pipeline of the target.
	status := combat.MustHitPointStatus(30, combat.DiesInstantly, combat.NoAffinities())
	target := combat.MustCombatantID("goblin")
	dmg, err := combat.NewApplyDamageCommand(target, status, got, nil, combat.OtherSource, r.HitKind(), nil)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := combat.NewApplyDamage(tel).Run(ctx, dmg)
	if err != nil || applied.(combat.DamageApplied).Status().HitPoints().Current() != 30-8-10 {
		t.Errorf("ApplyDamage = %v, %v, want 12 Hit Points left", applied, err)
	}
}

// Requirement: doc/specs/attacks.md section 8 scenario 2, FR-092, FR-094, FR-128, FR-108 - through the public API only: a
// Grapple option against a Medium target behind Half Cover resolves the target's Dexterity save with +2 added once; a
// failed save is handed to BeginGrapple, and the same strike with the attacker's hands full is not possible.
// Path: happy.
func TestGrappleBehindHalfCoverAgainstADexteritySave(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cover := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	params := combat.ResolveUnarmedStrikeParams{
		Option: combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Dexterity,
			Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(1))}},
		AbilityModifier: must2(ability.NewScore(16)).Modifier(), ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)),
		AttackerSize: content.SizeMedium, TargetSize: content.SizeMedium, Hand: combat.HandAvailabilityHandFree, Distance: 5, Cover: cover,
	}
	cmd, err := combat.NewResolveUnarmedStrikeCommand(params)
	if err != nil {
		t.Fatal(err)
	}
	// Strength 16 gives +3, Proficiency +2: DC 13. The save d20 of 9 plus 1 plus the cover's 2 is 12: it fails.
	strike, _ := combat.NewResolveUnarmedStrike(dicetest.NewFixedRoller(9))
	res, err := strike.Run(ctx, cmd)
	if err != nil || unarmedName(res) != "grapple" {
		t.Fatalf("= %v, %v", res, err)
	}
	g := res.(combat.UnarmedStrikeGrappleResolved)
	save := g.Save().(combat.UnarmedStrikeSaveRolled).Result()
	if g.DC() != 13 || save.Total() != 12 || !g.Grappled() || save.Modifiers().Count() != 2 {
		t.Fatalf("grapple = DC %d total %d grappled %v, want 13, 12 and a failed save", g.DC(), save.Total(), g.Grappled())
	}
	// The host hands the succeeded Grapple to BeginGrapple.
	begun, err := combat.NewBeginGrapple().Run(ctx, combat.BeginGrappleCommand{Grapples: combat.Grapples{}, Grappler: combat.MustCombatantID("a"), Target: combat.MustCombatantID("b"),
		Part: combat.MustGraspingHand(0), EscapeDC: g.DC(), RangeFeet: 5})
	if _, ok := begun.(combat.BeginGrappleApplied); err != nil || !ok {
		t.Fatalf("BeginGrapple = %v, %v, want it applied", begun, err)
	}
	// With no hand free the Grapple option is not possible and no die is rolled.
	params.Hand = combat.HandAvailabilityNoHandFree
	roller := dicetest.NewFixedRoller(9)
	strike, _ = combat.NewResolveUnarmedStrike(roller)
	refused, err := strike.Run(ctx, must2(combat.NewResolveUnarmedStrikeCommand(params)))
	if err != nil || unarmedName(refused) != "not possible" || refused.(combat.UnarmedStrikeNotPossible).Reason() != combat.AttackNotPossibleReasonNoFreeHand ||
		len(roller.Requests()) != 0 {
		t.Errorf("= %v, %v, want the not-possible refusal with no die", refused, err)
	}
}
