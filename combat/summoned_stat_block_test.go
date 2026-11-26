package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sumExpr is the expression count d die + modifier.
func sumExpr(count int, die dice.Die, modifier int) dice.Expression {
	return dice.MustExpression(count, die, modifier)
}

// sumBlock returns a function that takes the results of a stat block constructor, which must have built one, and returns the
// block: sumBlock(t)(combat.NewX(...)).
func sumBlock(tb testing.TB) func(combat.SummonedStatBlock, combat.SummonRejection, error) combat.SummonedStatBlock {
	tb.Helper()
	return func(b combat.SummonedStatBlock, rej combat.SummonRejection, err error) combat.SummonedStatBlock {
		tb.Helper()
		if err != nil || rej != nil {
			tb.Fatalf("constructor refused: rejection %#v, error %v", rej, err)
		}
		return b
	}
}

// sumAttack returns the attack of the kind, failing when the block has none.
func sumAttack(tb testing.TB, b combat.SummonedStatBlock, k combat.SummonedAttackKind) combat.SummonedAttack {
	tb.Helper()
	a, ok := b.Attack(k)
	if !ok {
		tb.Fatalf("stat block has no %v", k)
	}
	return a
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-029, FR-030, Table S rows 1 to 6 - the Animated Object has Armor Class 15,
// Hit Points 10 / 20 / 40 by size, Speed 30 and a Force Slam that uses the summoner's spell attack bonus and deals 1d4+3, 2d6+3+mod or
// 2d12+3+mod plus 1d4 / 1d6 / 1d12 for each level above 5 (SRD 5.2.1, "Animate Objects", "Animated Object", pp. 108-109).
// Path: per case.
func TestAnimatedObjectStatBlockTableS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		size                    content.CreatureSize
		level, hp               int
		slam                    dice.Expression
	}{
		{"1 Object M, L5", "Table S 1, p. 109", "happy", content.SizeMedium, 5, 10, sumExpr(1, dice.D4(), 3)},
		{"2 Object M, L7", "Table S 2, p. 109", "unhappy (boundary)", content.SizeMedium, 7, 10, sumExpr(3, dice.D4(), 3)},
		{"Tiny counts as Medium or smaller", "FR-029, p. 109", "unhappy (boundary)", content.SizeTiny, 5, 10, sumExpr(1, dice.D4(), 3)},
		{"3 Object L, L5", "Table S 3, p. 109", "happy", content.SizeLarge, 5, 20, sumExpr(2, dice.D6(), 7)},
		{"4 Object L, L8", "Table S 4, p. 109", "unhappy (boundary)", content.SizeLarge, 8, 20, sumExpr(5, dice.D6(), 7)},
		{"5 Object H, L5", "Table S 5, p. 109", "happy", content.SizeHuge, 5, 40, sumExpr(2, dice.D12(), 7)},
		{"6 Object H, L9", "Table S 6, p. 109", "unhappy (boundary)", content.SizeHuge, 9, 40, sumExpr(6, dice.D12(), 7)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := sumBlock(t)(combat.NewAnimatedObjectStatBlock(c.size, c.level, sumFacts(t)))
			slam := sumAttack(t, b, combat.SummonedAttackSlam)
			if b.ArmorClass() != 15 || b.HitPointMaximum() != c.hp || b.Speed(content.SpeedWalk) != 30 || b.Size() != c.size {
				t.Errorf("AC %d HP %d Speed %d size %v, want 15, %d, 30, %v [%s]", b.ArmorClass(), b.HitPointMaximum(), b.Speed(content.SpeedWalk), b.Size(), c.hp, c.size, c.requirement)
			}
			if slam.Damage() != c.slam || slam.DamageType() != content.DamageForce || slam.AttackBonus() != 7 {
				t.Errorf("Slam = %v %v bonus %d, want %v Force bonus 7 [%s]", slam.Damage(), slam.DamageType(), slam.AttackBonus(), c.slam, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-029, FR-031, FR-035, Table S rows 7 and 11 - a Gargantuan object and a level below the spell's own are refused with their rejection and no error; the rejection of the level comes first.
// Path: unhappy (invalid input).
func TestStatBlocksRefuseWithRejections(t *testing.T) {
	t.Parallel()
	_, rej, err := combat.NewAnimatedObjectStatBlock(content.SizeGargantuan, 5, sumFacts(t))
	if _, ok := rej.(combat.ObjectNotEligibleRejection); !ok || err != nil {
		t.Errorf("Gargantuan object = %#v, %v, want ObjectNotEligibleRejection [Table S 7]", rej, err)
	}
	_, rej, err = combat.NewAnimatedObjectStatBlock(content.SizeGargantuan, 4, sumFacts(t))
	if r, ok := rej.(combat.LevelBelowSpellRejection); !ok || r.Minimum != 5 || err != nil {
		t.Errorf("Gargantuan object at level 4 = %#v, %v, want LevelBelowSpellRejection{5} first [Table R]", rej, err)
	}
	for _, c := range []struct {
		name string
		call func() (combat.SummonedStatBlock, combat.SummonRejection, error)
		min  int
	}{
		{"insect", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewGiantInsectStatBlock(combat.InsectWasp, 3, sumFacts(t))
		}, 4},
		{"spirit", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewDraconicSpiritStatBlock(4, sumFacts(t), content.DamageFire, content.DamageFire)
		}, 5},
		{"steed", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewOtherworldlySteedStatBlock(combat.SteedFey, 1, sumFacts(t))
		}, 2},
	} {
		b, rej, err := c.call()
		if r, ok := rej.(combat.LevelBelowSpellRejection); !ok || r.Minimum != c.min || err != nil || !b.IsZero() {
			t.Errorf("%s below its level = %#v, %v, want LevelBelowSpellRejection{%d} and the zero block [Table S 11, 18]", c.name, rej, err, c.min)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-032, Table S rows 8 to 10 - the Giant Insect has Armor Class 11+L, Hit Points 30 plus 10 per level above 4, Speed 40 and Climb 40, a Multiattack of L/2 attacks and a Poison Jab of 1d6+3+L Piercing plus 1d4 Poison, reach 10; the wasp alone flies, the spider alone has Web Bolt and the centipede alone Venomous Spew (SRD 5.2.1, "Giant Insect", pp. 135-136).
// Path: per case.
func TestGiantInsectStatBlockTableS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		form                    combat.InsectForm
		level, ac, hp, multi    int
		fly                     int
		web, spew               bool
	}{
		{"8 spider L4", "Table S 8, p. 136", "happy", combat.InsectSpider, 4, 15, 30, 2, 0, true, false},
		{"9 wasp L5", "Table S 9, p. 136", "unhappy (boundary)", combat.InsectWasp, 5, 16, 40, 2, 40, false, false},
		{"9 centipede L9", "Table S 9, p. 136", "unhappy (boundary)", combat.InsectCentipede, 9, 20, 80, 4, 0, false, true},
		{"10 wasp traits", "Table S 10, p. 136", "unhappy (rule interaction)", combat.InsectWasp, 4, 15, 30, 2, 40, false, false},
		{"10 centipede traits", "Table S 10, p. 136", "unhappy (rule interaction)", combat.InsectCentipede, 4, 15, 30, 2, 0, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := sumBlock(t)(combat.NewGiantInsectStatBlock(c.form, c.level, sumFacts(t)))
			if b.ArmorClass() != c.ac || b.HitPointMaximum() != c.hp || b.MultiattackCount() != c.multi || b.Speed(content.SpeedWalk) != 40 ||
				b.Speed(content.SpeedClimb) != 40 || b.Speed(content.SpeedFly) != c.fly {
				t.Errorf("AC %d HP %d Multiattack %d Speed %d Climb %d Fly %d, want %d %d %d 40 40 %d [%s]", b.ArmorClass(), b.HitPointMaximum(),
					b.MultiattackCount(), b.Speed(content.SpeedWalk), b.Speed(content.SpeedClimb), b.Speed(content.SpeedFly), c.ac, c.hp, c.multi, c.fly, c.requirement)
			}
			jab := sumAttack(t, b, combat.SummonedAttackPoisonJab)
			if jab.Damage() != sumExpr(1, dice.D6(), 3+c.level) || jab.DamageType() != content.DamagePiercing || jab.ExtraDamage() != sumExpr(1, dice.D4(), 0) ||
				jab.ExtraDamageType() != content.DamagePoison || jab.RangeFeet() != 10 || jab.AttackBonus() != 7 {
				t.Errorf("Poison Jab = %v %v + %v %v reach %d, want 1d6+%d Piercing + 1d4 Poison reach 10 [%s]", jab.Damage(), jab.DamageType(), jab.ExtraDamage(),
					jab.ExtraDamageType(), jab.RangeFeet(), 3+c.level, c.requirement)
			}
			web, hasWeb := b.Attack(combat.SummonedAttackWebBolt)
			if hasWeb != c.web || (hasWeb && (web.Damage() != sumExpr(1, dice.D10(), 3+c.level) || web.DamageType() != content.DamageBludgeoning || web.RangeFeet() != 60)) {
				t.Errorf("Web Bolt = %v present %v, want present %v with 1d10+%d Bludgeoning, range 60 [%s]", web, hasWeb, c.web, 3+c.level, c.requirement)
			}
			spew, hasSpew := b.Attack(combat.SummonedAttackVenomousSpew)
			if hasSpew != c.spew || (hasSpew && (spew.SaveDC() != 15 || spew.RangeFeet() != 10 || !spew.Kind().IsBonusAction())) {
				t.Errorf("Venomous Spew = %v present %v, want present %v as a Bonus Action with DC 15 within 10 feet [%s]", spew, hasSpew, c.spew, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-033, Table S rows 12 to 14 - the Draconic Spirit has Armor Class 14+L, Hit Points 50 plus 10 per level above 5, Speeds 30, Fly 60 and Swim 30, Resistance to Acid, Cold, Fire, Lightning and Poison, a Multiattack of L/2 Rend attacks (1d6+4+L) and a Breath Weapon of 2d6 of its own type; the two types are separate choices and a type outside the five is refused for the field named, the Resistance first (SRD 5.2.1, "Draconic Spirit", p. 166).
// Path: per case.
func TestDraconicSpiritStatBlockTableS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level, ac, hp           int
		shared, breath          content.DamageType
		field                   combat.ResistanceField
	}{
		{"12 spirit L5", "Table S 12, p. 166", "happy", 5, 19, 50, content.DamageCold, content.DamageFire, 0},
		{"13 spirit L9", "Table S 13, p. 166", "unhappy (boundary)", 9, 23, 90, content.DamageAcid, content.DamageAcid, 0},
		{"the two types may differ", "FR-033, Q-18, p. 166", "unhappy (rule interaction)", 6, 20, 60, content.DamagePoison, content.DamageLightning, 0},
		{"14 shared force", "Table S 14, p. 166", "unhappy (invalid input)", 5, 0, 0, content.DamageForce, content.DamageFire, combat.SharedResistanceField},
		{"14 breath force", "Table S 14, p. 166", "unhappy (invalid input)", 5, 0, 0, content.DamageFire, content.DamageForce, combat.BreathDamageTypeField},
		{"14 both outside: the Resistance first", "Table R, FR-033", "unhappy (rule interaction)", 5, 0, 0, content.DamageForce, content.DamageForce, combat.SharedResistanceField},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b, rej, err := combat.NewDraconicSpiritStatBlock(c.level, sumFacts(t), c.shared, c.breath)
			if c.field != 0 {
				if r, ok := rej.(combat.ResistanceNotOfferedRejection); !ok || r.Field != c.field || err != nil {
					t.Errorf("rejection = %#v, %v, want ResistanceNotOfferedRejection{%v} [%s]", rej, err, c.field, c.requirement)
				}
				return
			}
			b = sumBlock(t)(b, rej, err)
			rend, breath := sumAttack(t, b, combat.SummonedAttackRend), sumAttack(t, b, combat.SummonedAttackBreathWeapon)
			if b.ArmorClass() != c.ac || b.HitPointMaximum() != c.hp || b.Speed(content.SpeedWalk) != 30 || b.Speed(content.SpeedFly) != 60 ||
				b.Speed(content.SpeedSwim) != 30 || b.MultiattackCount() != c.level/2 {
				t.Errorf("AC %d HP %d Multiattack %d, want %d %d %d [%s]", b.ArmorClass(), b.HitPointMaximum(), b.MultiattackCount(), c.ac, c.hp, c.level/2, c.requirement)
			}
			if rend.Damage() != sumExpr(1, dice.D6(), 4+c.level) || rend.DamageType() != content.DamagePiercing || rend.RangeFeet() != 10 ||
				breath.Damage() != sumExpr(2, dice.D6(), 0) || breath.DamageType() != c.breath || breath.SaveDC() != 15 || breath.RangeFeet() != 30 {
				t.Errorf("Rend %v, Breath Weapon %v %v DC %d, want 1d6+%d and 2d6 %v DC 15 [%s]", rend.Damage(), breath.Damage(), breath.DamageType(), breath.SaveDC(), 4+c.level, c.breath, c.requirement)
			}
			want := []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageLightning, content.DamagePoison}
			if got := b.Resistances(); len(got) != 5 || got[0] != want[0] || got[4] != want[4] || b.Type() != content.CreatureTypeDragon || b.Size() != content.SizeLarge {
				t.Errorf("Resistances = %v type %v, want %v of a Large Dragon [%s]", got, b.Type(), want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-034, FR-057, FR-058, Table S rows 15 to 18 - the Otherworldly Steed is Large with Armor Class 10+L, Hit Points 5+10L with L d10 Hit Dice, Speed 60, Fly 60 only from level 4, Life Bond, telepathy of 1 mile, a Slam of 1d8+L of Radiant, Psychic or Necrotic damage and the one Bonus Action of its type (SRD 5.2.1, "Find Steed", p. 130; "Otherworldly Steed", p. 131).
// Path: per case.
func TestOtherworldlySteedStatBlockTableS(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		steed                   combat.SteedType
		level, ac, hp, fly      int
		typ                     content.DamageType
		bonus                   combat.SummonedAttackKind
	}{
		{"15 steed L2 Fey", "Table S 15, pp. 130-131", "happy", combat.SteedFey, 2, 12, 25, 0, content.DamagePsychic, combat.SummonedAttackFeyStep},
		{"16 steed L4 Celestial", "Table S 16, p. 131", "unhappy (boundary)", combat.SteedCelestial, 4, 14, 45, 60, content.DamageRadiant, combat.SummonedAttackHealingTouch},
		{"16 steed L3 Celestial", "Table S 16, p. 131", "unhappy (boundary)", combat.SteedCelestial, 3, 13, 35, 0, content.DamageRadiant, combat.SummonedAttackHealingTouch},
		{"17 steed L9 Fiend", "Table S 17, p. 131", "unhappy (boundary)", combat.SteedFiend, 9, 19, 95, 60, content.DamageNecrotic, combat.SummonedAttackFellGlare},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := sumBlock(t)(combat.NewOtherworldlySteedStatBlock(c.steed, c.level, sumFacts(t)))
			slam := sumAttack(t, b, combat.SummonedAttackOtherworldlySlam)
			if b.ArmorClass() != c.ac || b.HitPointMaximum() != c.hp || b.HitDice() != sumExpr(c.level, dice.D10(), 0) || b.Speed(content.SpeedWalk) != 60 ||
				b.Speed(content.SpeedFly) != c.fly || b.Size() != content.SizeLarge || b.Type() != c.steed.CreatureType() || !b.HasLifeBond() {
				t.Errorf("AC %d HP %d Hit Dice %v Fly %d, want %d %d %dd10 %d [%s]", b.ArmorClass(), b.HitPointMaximum(), b.HitDice(), b.Speed(content.SpeedFly), c.ac, c.hp, c.level, c.fly, c.requirement)
			}
			if slam.Damage() != sumExpr(1, dice.D8(), c.level) || slam.DamageType() != c.typ || slam.AttackBonus() != 7 {
				t.Errorf("Slam = %v %v bonus %d, want 1d8+%d %v bonus 7 [%s]", slam.Damage(), slam.DamageType(), slam.AttackBonus(), c.level, c.typ, c.requirement)
			}
			for _, k := range []combat.SummonedAttackKind{combat.SummonedAttackFellGlare, combat.SummonedAttackFeyStep, combat.SummonedAttackHealingTouch} {
				if _, has := b.Attack(k); has != (k == c.bonus) {
					t.Errorf("has %v = %v, want only %v [%s]", k, has, c.bonus, c.requirement)
				}
			}
			if k := c.bonus; k == combat.SummonedAttackHealingTouch {
				if heal := sumAttack(t, b, k); heal.Damage() != sumExpr(2, dice.D8(), c.level) || heal.DamageType() != 0 {
					t.Errorf("Healing Touch = %v %v, want 2d8+%d healing [FR-034]", heal.Damage(), heal.DamageType(), c.level)
				}
			}
			if k := c.bonus; k == combat.SummonedAttackFellGlare && sumAttack(t, b, k).SaveDC() != 15 {
				t.Errorf("Fell Glare DC = %d, want the spell save DC 15 [FR-034]", sumAttack(t, b, k).SaveDC())
			}
			if tp := b.Telepathy(); !tp.IsTelepathic() || tp.RangeFeet() != 5280 {
				t.Errorf("Telepathy = %v, want 5280 feet [FR-058]", tp)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-036 - the attack bonus, save DC, spellcasting modifier and Proficiency Bonus come from the facts, so other facts give other numbers and the stat block holds no ability scores.
// Path: happy.
func TestStatBlocksReadTheSummonersFacts(t *testing.T) {
	t.Parallel()
	other := luMust(combat.NewSummonerSpellFacts(11, 19, 5, 5))
	obj := sumBlock(t)(combat.NewAnimatedObjectStatBlock(content.SizeLarge, 5, other))
	if s := sumAttack(t, obj, combat.SummonedAttackSlam); s.AttackBonus() != 11 || s.Damage() != sumExpr(2, dice.D6(), 8) || obj.ProficiencyBonus() != 5 {
		t.Errorf("Slam = bonus %d %v, Proficiency Bonus %d, want 11, 2d6+8 and 5 (the modifier 5 added)", s.AttackBonus(), s.Damage(), obj.ProficiencyBonus())
	}
	spirit := sumBlock(t)(combat.NewDraconicSpiritStatBlock(5, other, content.DamageCold, content.DamageCold))
	if b := sumAttack(t, spirit, combat.SummonedAttackBreathWeapon); b.SaveDC() != 19 || spirit.ProficiencyBonus() != 5 {
		t.Errorf("Breath Weapon DC = %d, Proficiency Bonus %d, want 19 and 5", b.SaveDC(), spirit.ProficiencyBonus())
	}
	if f := sumFacts(t); f.AttackBonus() != 7 || f.SaveDC() != 15 || f.Modifier() != 4 || f.ProficiencyBonus() != 3 || f.IsZero() {
		t.Errorf("facts = %+v, want 7, 15, 4, 3", f)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-035, FR-067 - a cast level outside 1 to 9, a size, form or type that is not defined and the zero facts are invalid input for every stat block constructor (errors.Is srderr.ErrInvalidInput), with no rejection; numbers of the facts outside bounds are invalid too.
// Path: per case.
func TestStatBlockConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	f := sumFacts(t)
	cases := []struct {
		name string
		call func() (combat.SummonedStatBlock, combat.SummonRejection, error)
	}{
		{"object level 0", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewAnimatedObjectStatBlock(content.SizeMedium, 0, f)
		}},
		{"object level 10", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewAnimatedObjectStatBlock(content.SizeMedium, 10, f)
		}},
		{"object size not defined", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewAnimatedObjectStatBlock(0, 5, f)
		}},
		{"object zero facts", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewAnimatedObjectStatBlock(content.SizeMedium, 5, combat.SummonerSpellFacts{})
		}},
		{"insect form not defined", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewGiantInsectStatBlock(0, 5, f)
		}},
		{"insect level 10", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewGiantInsectStatBlock(combat.InsectWasp, 10, f)
		}},
		{"spirit level 0", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewDraconicSpiritStatBlock(0, f, content.DamageFire, content.DamageFire)
		}},
		{"steed type not defined", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewOtherworldlySteedStatBlock(0, 5, f)
		}},
		{"steed zero facts", func() (combat.SummonedStatBlock, combat.SummonRejection, error) {
			return combat.NewOtherworldlySteedStatBlock(combat.SteedFey, 5, combat.SummonerSpellFacts{})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b, rej, err := c.call()
			if !errors.Is(err, srderr.ErrInvalidInput) || rej != nil || !b.IsZero() {
				t.Errorf("= %v, %#v, %v, want the zero block, no rejection and srderr.ErrInvalidInput [FR-035, FR-067]", b, rej, err)
			}
		})
	}
	for _, bad := range [][4]int{{0, 0, 0, 3}, {1, 15, 0, 0}, {1001, 15, 4, 3}, {-1001, 15, 4, 3}, {7, 1001, 4, 3}, {7, 15, 1001, 3}, {7, 15, -1001, 3}, {7, 15, 4, 1001}} {
		if _, err := combat.NewSummonerSpellFacts(bad[0], bad[1], bad[2], bad[3]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewSummonerSpellFacts(%v) error = %v, want srderr.ErrInvalidInput [FR-067]", bad, err)
		}
	}
	if _, err := combat.NewSummonerSpellFacts(1000, 1000, -1000, 1000); err != nil {
		t.Errorf("NewSummonerSpellFacts at the bounds error = %v, want nil", err)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008, FR-061 - every summoned stat block dies instantly at 0 Hit Points, and a stat block copied from a Monster takes its size, type, Armor Class, average Hit Points, walk Speed, Proficiency Bonus and XP (FR-007).
// Path: happy.
func TestStatBlockOfAMonsterAndTheZeroHitPointsRule(t *testing.T) {
	t.Parallel()
	roster := combat.SummonedRoster{}
	a := sumDo(t, roster, "pc", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{{Creature: cid("sk"), Remains: "bones-1", Kind: combat.RemainsBones, Size: content.SizeMedium}}}, sumScene())
	spec := a.Creatures()[0]
	b := spec.StatBlock()
	if b.Monster() != "skeleton" || b.Type() != content.CreatureTypeUndead || b.Size() != content.SizeMedium || b.ArmorClass() != 12 || b.HitPointMaximum() != 13 ||
		b.Speed(content.SpeedWalk) != 30 || b.Speed(content.SpeedClimb) != 20 || b.ExperiencePoints() != 50 || b.ProficiencyBonus() != 2 || b.ZeroHitPointsRule() != combat.DiesInstantly {
		t.Errorf("stat block = %+v, want the skeleton fixture with XP 50, Proficiency Bonus 2 and DiesInstantly [FR-007, FR-008]", b)
	}
	if ref := spec.Creature(); ref.ID != cid("sk") || ref.Kind != (combat.MonsterCombatant{ID: "skeleton"}) {
		t.Errorf("Creature() = %#v, want MonsterCombatant(skeleton) [FR-008]", ref)
	}
	if _, has := spec.Monster(); !has || !b.CanGainLevels() || !b.CanTakeRests() {
		t.Errorf("spec holds the monster %v, CanGainLevels %v, CanTakeRests %v, want true, true, true", has, b.CanGainLevels(), b.CanTakeRests())
	}
	obj := sumBlock(t)(combat.NewAnimatedObjectStatBlock(content.SizeMedium, 5, sumFacts(t)))
	if obj.ExperiencePoints() != 0 || obj.ZeroHitPointsRule() != combat.DiesInstantly {
		t.Errorf("object XP %d, rule %v, want 0 and DiesInstantly [FR-007, FR-008]", obj.ExperiencePoints(), obj.ZeroHitPointsRule())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-065 - a stat block hands out its lists as copies, so changing one changes nothing stored; two builds are equal by value and a different level is not.
// Path: unhappy (immutability).
func TestStatBlockListsAreCopies(t *testing.T) {
	t.Parallel()
	b := sumBlock(t)(combat.NewDraconicSpiritStatBlock(5, sumFacts(t), content.DamageCold, content.DamageFire))
	res, imm, atk := b.Resistances(), b.ConditionImmunities(), b.Attacks()
	res[0], imm[0], atk[0] = content.DamageForce, content.ConditionBlinded, combat.SummonedAttack{}
	again := sumBlock(t)(combat.NewDraconicSpiritStatBlock(5, sumFacts(t), content.DamageCold, content.DamageFire))
	if !b.Equal(again) || b.Resistances()[0] != content.DamageAcid || b.ConditionImmunities()[0] != content.ConditionCharmed || b.Attacks()[0].Kind() != combat.SummonedAttackRend {
		t.Errorf("a returned list changed the stat block, or two builds differ [FR-065]")
	}
	if other := sumBlock(t)(combat.NewDraconicSpiritStatBlock(6, sumFacts(t), content.DamageCold, content.DamageFire)); b.Equal(other) {
		t.Errorf("stat blocks of levels 5 and 6 are equal")
	}
	sim := sumBlock(t)(combat.NewSimulacrumStatBlock(combat.SimulacrumOriginal{Monster: "hero", Size: content.SizeMedium, Type: content.CreatureTypeHumanoid, HitPointMaximum: 11, WithinTenFeet: true}))
	if f := sim.ForbiddenSpells(); len(f) != 1 {
		t.Fatalf("ForbiddenSpells = %v, want simulacrum", f)
	} else {
		f[0] = "x"
	}
	if sim.ForbiddenSpells()[0] != "simulacrum" {
		t.Errorf("ForbiddenSpells changed through a returned slice [FR-065]")
	}
}
