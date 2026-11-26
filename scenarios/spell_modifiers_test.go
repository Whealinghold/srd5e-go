package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/spell-granted-modifiers.md section 8 (scenario test), FR-021, FR-030, FR-044, FR-051, FR-040, FR-061, FR-062 - a hasted character with Shield of Faith, a Bless die and a Bestow Curse on an ogre
// marked by Hunter's Mark: the Survival check to find the ogre has Advantage, the bonus die is rolled and lifts an attack roll to a hit, the ogre's Shield spell turns that hit into a miss, a natural 20 is a Critical
// Hit that doubles the curse's extra d8 (Hunter's Mark adds its 1d6 Force before it), and Haste doubles the hero's Speed. Public packages only.
// Path: happy.
func TestSpellGrantedModifiersScenario(t *testing.T) {
	t.Parallel()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	hero, ogre := combat.MustCombatantID("hero"), combat.MustCombatantID("ogre")
	src := func(spell string, caster, target combat.CombatantID, order int, choice combat.SpellModifierChoice) combat.SpellModifierSource {
		s, err := combat.NewSpellModifierSource(content.SpellID(spell), caster, target, order, choice)
		must(err)
		return s
	}
	onHero := []combat.SpellModifierSource{
		src("haste", hero, hero, 0, combat.NoChoice{}), src("shield_of_faith", hero, hero, 1, combat.NoChoice{}),
		src("hunters_mark", hero, ogre, 2, combat.NoChoice{}),
		src("bestow_curse", hero, ogre, 3, combat.ChosenCurse{Effect: combat.CurseExtraDamage{}}),
	}

	// The hero looks for the marked ogre: Hunter's Mark gives Advantage on Wisdom (Survival).
	find, err := combat.QueryRollContributions(onHero, combat.AbilityCheckContext{Creature: hero, Ability: ability.Wisdom, Skill: content.SkillSurvival, Seeking: ogre})
	must(err)
	if mode := modifier.Resolve(find.Circumstances()...); mode != modifier.WithAdvantage {
		t.Errorf("the check to find the mark has %v, want Advantage", mode)
	}

	// Armor Class and Speed of the hero: Shield of Faith +2 and Haste +2 and doubled Speed.
	acIn, err := combat.QueryArmorClass(onHero, hero, false)
	must(err)
	total := 14
	for _, b := range acIn.FlatBonuses() {
		total += b.Modifier().Value()
	}
	walk, err := combat.NewMovementSpeeds(30)
	must(err)
	fast, err := combat.QuerySpeeds(onHero, hero, walk)
	must(err)
	if total != 18 || fast.Walk() != 60 {
		t.Errorf("hero Armor Class %d and Speed %d, want 18 and 60", total, fast.Walk())
	}

	// The Bless die of the hero (the rider of its Bless source on an attack roll) is rolled before the d20.
	blessed, err := combat.QueryRollContributions([]combat.SpellModifierSource{src("bless", ogre, hero, 4, combat.NoChoice{})},
		combat.AttackRollContext{Attacker: hero, Target: ogre})
	must(err)
	bless := blessed.BonusDice()[0]
	use, err := combat.NewRollSpellBonusDice(dicetest.NewFixedRoller(3))
	must(err)
	res, err := use.Run(context.Background(), combat.RollSpellBonusDiceCommand{Dice: []combat.BonusDie{bless}})
	must(err)
	bonus := res.(combat.SpellBonusDiceRolled).Modifiers()
	roll, err := d20.NewRoll(modifier.Normal, []int{10})
	must(err)
	attack, err := d20.ResolveAttack(roll, d20.NewRollModifiers(modifier.New(4)).WithOthers(bonus...), 16, 20)
	must(err)
	if !attack.Hit() || attack.Total() != 17 {
		t.Fatalf("attack total %d hit=%v, want 17 and a hit against Armor Class 16", attack.Total(), attack.Hit())
	}

	// The ogre casts Shield in reply: Armor Class 16 + 5 and the same attack roll is judged again.
	ogreSpells := []combat.SpellModifierSource{src("shield", ogre, ogre, 0, combat.NoChoice{})}
	ogreAC, err := combat.QueryArmorClass(ogreSpells, ogre, false)
	must(err)
	newAC := 16
	for _, b := range ogreAC.FlatBonuses() {
		newAC += b.Modifier().Value()
	}
	again, err := combat.ReevaluateAttack(attack, d20.ArmorClass(newAC))
	must(err)
	if again.Hit() || newAC != 21 {
		t.Errorf("after Shield the attack hit=%v against %d, want a miss against 21", again.Hit(), newAC)
	}

	// A natural 20 is a Critical Hit whatever the Armor Class; the curse's extra d8 is doubled by the attack's rule.
	crit, err := d20.NewRoll(modifier.Normal, []int{20})
	must(err)
	critical, err := combat.ReevaluateAttack(mustAttack(t, crit), d20.ArmorClass(newAC))
	must(err)
	riders, err := combat.QueryDamageRiders(onHero, hero, ogre, combat.AttackRollHitEvent{})
	must(err)
	extra, ok := (combat.AttackDamageComposer{}).Dice(riders[1], critical.IsCriticalHit())
	if !critical.Hit() || !critical.IsCriticalHit() || len(riders) != 2 || !ok || extra.Notation() != "2d8" || riders[1].Type() != content.DamageNecrotic {
		t.Errorf("critical = %v/%v, riders = %v, extra = %v", critical.Hit(), critical.IsCriticalHit(), riders, extra)
	}
}

func mustAttack(t *testing.T, roll d20.Roll) d20.AttackRollResult {
	t.Helper()
	a, err := d20.ResolveAttack(roll, d20.NewRollModifiers(modifier.New(4)), 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
