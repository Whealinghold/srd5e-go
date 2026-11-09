package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A level 5 Rogue attacks with a Shortbow at Advantage, gets Sneak Attack of 3d6, selects Poison with a Poisoner's Kit, and the DC (8 + Dexterity
// modifier 3 + Proficiency Bonus 3), the Constitution save, the Poisoned condition and the 2d6 left to roll match the printed rules; the same Rogue
// does not get Sneak Attack on a second attack in the turn, and a level 20 Rogue turns a failed save into a 20 once until a rest (SRD 5.2.1,
// "Level 1: Sneak Attack", "Level 5: Cunning Strike", "Level 20: Stroke of Luck", pp. 61-63). Cites class-rogue.md FR-011, FR-036, FR-048, FR-052, FR-056 to FR-058, FR-086, FR-088.
func TestRogueSneakAttackWithPoison(t *testing.T) {
	t.Parallel()
	res, err := data.MustBundledSource().Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("rogue")))
	if err != nil {
		t.Fatal(err)
	}
	progression := must(character.NewRogueProgression(res.(content.LookupFound).Entry.(content.ClassDefinition)))
	level := character.MustRogueLevel(5)
	roll := d20.MustRoll(modifier.WithAdvantage, 4, 17)
	attack := must(d20.ResolveAttack(roll, d20.NewRollModifiers(modifier.New(6)), d20.ArmorClass(15), 20))
	sneak := character.SneakAttack{}.Evaluate(character.SneakAttackInput{
		Dice: progression.SneakAttackDice(level), Attack: attack, Ranged: true, Mode: modifier.WithAdvantage, DamageType: content.DamagePiercing,
	})
	applies, ok := sneak.(character.SneakAttackApplies)
	if !ok || applies.Dice().Count() != 3 || applies.DamageType() != content.DamagePiercing {
		t.Fatalf("Sneak Attack = %#v, want 3d6 Piercing", sneak)
	}
	dexterity := must(ability.NewScore(16)).Modifier()
	bonus := character.ProficiencyBonusFor(must(character.NewLevel(5)))
	sel, err := character.NewCunningStrikeSelection(character.CunningStrikeInput{
		Level: level, SneakAttack: sneak, Effects: []character.CunningStrikeEffect{character.CunningStrikePoison}, PoisonersKit: true,
		DexterityModifier: dexterity, ProficiencyBonus: bonus, TargetSize: content.SizeMedium, SpeedFeet: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	poison, ok := sel.Outcomes()[0].(character.CunningStrikeConditionOnFailedSave)
	if !ok || poison.DC != 14 || poison.SaveAbility != ability.Constitution || poison.Condition != content.ConditionPoisoned || sel.RemainingDice().Count() != 2 || sel.Cost().Count() != 1 {
		t.Errorf("Poison = %#v, cost %d, left %d", sel.Outcomes()[0], sel.Cost().Count(), sel.RemainingDice().Count())
	}
	again := character.SneakAttack{}.Evaluate(character.SneakAttackInput{Dice: applies.Dice(), Attack: attack, Ranged: true, Mode: modifier.WithAdvantage, UsedThisTurn: true})
	if n, ok := again.(character.SneakAttackDoesNotApply); !ok || n.Reasons()[0] != character.SneakAttackAlreadyUsedThisTurn {
		t.Errorf("a second Sneak Attack in the turn = %#v", again)
	}

	id := character.MustIdentifier("rogue-20")
	failed := character.StrokeOfLuckRolledTest{Roll: d20.MustRoll(modifier.Normal, 3), Outcome: d20.Failure}
	spent, ok := character.StrokeOfLuck{}.Use(id, character.MustRogueLevel(20), character.InitialRogueState(), failed).(character.StrokeOfLuckSpent)
	if !ok || spent.Roll().Natural() != 20 {
		t.Fatalf("Stroke of Luck = %#v", spent)
	}
	if _, ok := (character.StrokeOfLuck{}).Use(id, character.MustRogueLevel(20), spent.State(), failed).(character.StrokeOfLuckNotAvailable); !ok {
		t.Error("Stroke of Luck was used twice")
	}
	if _, ok := (character.StrokeOfLuck{}).Use(id, character.MustRogueLevel(20), spent.State().AfterRest(character.ShortRest), failed).(character.StrokeOfLuckSpent); !ok {
		t.Error("Stroke of Luck is not back after a Short Rest")
	}
}
