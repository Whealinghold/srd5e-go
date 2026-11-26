package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

const (
	// srdAttack cites the pages the attack tables share.
	srdAttack = `"Attack Rolls", p. 7`
	srdCover  = `"Cover", p. 15`
	srdRange  = `"Range", p. 15`
)

var (
	advantage    = modifier.RollCircumstance(modifier.Advantage{})
	disadvantage = modifier.RollCircumstance(modifier.Disadvantage{})
)

// abilityMod returns the ability modifier n (-5 to +10), through the score that gives it.
func abilityMod(n int) ability.Modifier { return ability.MustScore(max(10+2*n, 1)).Modifier() }

// profBonus returns the Proficiency Bonus of the level that gives it (+2 at level 1, +3 at 5).
func profBonus(n int) character.ProficiencyBonus {
	lvl := character.MustLevel(1)
	if n == 3 {
		lvl = character.MustLevel(5)
	}
	return character.ProficiencyBonusFor(lvl)
}

// mods returns roll modifiers with the ability modifier m and, when pb is not 0, the Proficiency Bonus pb.
func mods(m, pb int, others ...int) d20.RollModifiers {
	r := d20.NewRollModifiers(modifier.New(m))
	if pb != 0 {
		r = r.WithProficiencyBonus(modifier.New(pb))
	}
	for _, o := range others {
		r = r.WithOthers(modifier.New(o))
	}
	return r
}

// meleeParams is a melee weapon attack against AC 15 at 5 feet with +5 (ability 3, proficiency 2) and no damage.
func meleeParams() combat.ResolveAttackParams {
	return combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: 15, Modifiers: mods(3, 2), Distance: 5}
}

// rangedParams is a ranged weapon attack against AC 15 at 30 feet, range 80/320, +5.
func rangedParams() combat.ResolveAttackParams {
	return combat.ResolveAttackParams{Form: combat.RangedWeaponAttack, ArmorClass: 15, Modifiers: mods(3, 2), Distance: 30,
		Range: combat.MustAttackRange(80, 320)}
}

// attackCommand builds the command or fails the test.
func attackCommand(tb testing.TB, p combat.ResolveAttackParams) combat.ResolveAttackCommand {
	tb.Helper()
	c, err := combat.NewResolveAttackCommand(p)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// slashing is a spec of one die of sides sides, Slashing, with the ability modifier on it.
func slashing(count, sides int) combat.DamageSpec {
	return combat.MustDamageSpec(combat.MustDiceDamage(count, dice.MustDie(sides), content.DamageSlashing, 0, true))
}

// runAttack runs the command with a fixed roller of faces and returns the result and the requests.
func runAttack(tb testing.TB, cmd combat.ResolveAttackCommand, faces ...int) (combat.ResolveAttackResult, []string) {
	tb.Helper()
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewResolveAttack(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return res, notations(roller)
}

// unarmedParams is a Damage-option Unarmed Strike against AC 15 with Strength +3 and Proficiency Bonus +2.
func unarmedParams(o combat.UnarmedStrikeOption) combat.ResolveUnarmedStrikeParams {
	return combat.ResolveUnarmedStrikeParams{Option: o, AbilityModifier: abilityMod(3), ProficiencyBonus: profBonus(2),
		AttackerSize: content.SizeMedium, TargetSize: content.SizeMedium, Hand: combat.HandAvailabilityHandFree, Distance: 5}
}

// unarmedCommand builds the command or fails the test.
func unarmedCommand(tb testing.TB, p combat.ResolveUnarmedStrikeParams) combat.ResolveUnarmedStrikeCommand {
	tb.Helper()
	c, err := combat.NewResolveUnarmedStrikeCommand(p)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// runUnarmed runs the command with a fixed roller of faces and returns the result and the requests.
func runUnarmed(tb testing.TB, cmd combat.ResolveUnarmedStrikeCommand, faces ...int) (combat.ResolveUnarmedStrikeResult, []string) {
	tb.Helper()
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewResolveUnarmedStrike(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return res, notations(roller)
}

// notations returns the dice expressions r was asked for, as notations.
func notations(r *dicetest.FixedRoller) []string {
	var out []string
	for _, e := range r.Requests() {
		out = append(out, e.Notation())
	}
	return out
}
