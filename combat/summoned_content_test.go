package combat_test

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice"
)

// sumStatBlockLines returns the lines of the embedded stat block of the bundled spell.
func sumStatBlockLines(tb testing.TB, spell content.SpellID) []string {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatalf("NewBundledSource: %v", err)
	}
	res, err := src.Lookup(context.Background(), spell.Key())
	found, ok := res.(content.LookupFound)
	if err != nil || !ok {
		tb.Fatalf("Lookup(%s) = %v, %v, want the bundled spell", spell, res, err)
	}
	for _, b := range found.Entry.(content.Spell).Description() {
		if sb, ok := b.(content.TextEmbeddedStatBlock); ok {
			return sb.Lines()
		}
	}
	tb.Fatalf("spell %s has no embedded stat block", spell)
	return nil
}

// sumLine returns the first line with the prefix.
func sumLine(tb testing.TB, lines []string, prefix string) string {
	tb.Helper()
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return l
		}
	}
	tb.Fatalf("no line starts with %q in %q", prefix, lines)
	return ""
}

// sumInts returns the numbers of the first match of the pattern in the text.
func sumInts(tb testing.TB, pattern, text string) []int {
	tb.Helper()
	m := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if m == nil {
		tb.Fatalf("%q does not match %q", text, pattern)
	}
	var out []int
	for _, s := range m[1:] {
		n, err := strconv.Atoi(s)
		if err != nil {
			tb.Fatal(err)
		}
		out = append(out, n)
	}
	return out
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-037, Table S rows 12 to 17 (pp. 131 and 166) - the numbers the Draconic Spirit and Otherworldly Steed constructors use equal the numbers of the embedded stat block text the bundled spells Summon Dragon and Find Steed carry: the Armor Class, Hit Points, Speeds, Resistances, Rend, Breath Weapon, Hit Dice, Slam, Healing Touch and the telepathy of 1 mile. The same comparison for Animate Objects (p. 109) and Giant Insect (p. 136) is not made: their spells are not bundled until Phase 5.
// Path: unhappy (drift).
func TestStatBlockNumbersEqualTheBundledText(t *testing.T) {
	t.Parallel()
	facts := sumFacts(t)
	t.Run("Draconic Spirit, p. 166", func(t *testing.T) {
		t.Parallel()
		lines := sumStatBlockLines(t, "summon_dragon")
		ac := sumInts(t, `^AC (\d+) \+`, sumLine(t, lines, "AC"))
		hp := sumInts(t, `^HP (\d+) \+ (\d+) for each spell level above (\d+)`, sumLine(t, lines, "HP"))
		speed := sumInts(t, `^Speed (\d+) ft\., Fly (\d+) ft\., Swim (\d+) ft\.`, sumLine(t, lines, "Speed"))
		rend := sumInts(t, `^Rend\. .*Hit: (\d+)d(\d+) \+ (\d+) \+ the spell’s level Piercing`, sumLine(t, lines, "Rend."))
		breath := sumInts(t, `Failure: (\d+)d(\d+) damage`, sumLine(t, lines, "Breath Weapon."))
		reach := sumInts(t, `reach (\d+) feet`, sumLine(t, lines, "Rend."))
		cone := sumInts(t, `(\d+)-foot Cone`, sumLine(t, lines, "Breath Weapon."))
		if !strings.Contains(sumLine(t, lines, "Multiattack."), "half the spell’s level (round down)") {
			t.Errorf("the Multiattack line changed: %q", sumLine(t, lines, "Multiattack."))
		}
		res := sumLine(t, lines, "Resistances")
		for _, level := range []int{5, 9} {
			b := sumBlock(t)(combat.NewDraconicSpiritStatBlock(level, facts, content.DamageAcid, content.DamageCold))
			rd, br := sumAttack(t, b, combat.SummonedAttackRend), sumAttack(t, b, combat.SummonedAttackBreathWeapon)
			if b.ArmorClass() != ac[0]+level || b.HitPointMaximum() != hp[0]+hp[1]*(level-hp[2]) || b.MultiattackCount() != level/2 {
				t.Errorf("level %d: AC %d HP %d Multiattack %d, want the text's numbers [FR-037]", level, b.ArmorClass(), b.HitPointMaximum(), b.MultiattackCount())
			}
			if b.Speed(content.SpeedWalk) != speed[0] || b.Speed(content.SpeedFly) != speed[1] || b.Speed(content.SpeedSwim) != speed[2] {
				t.Errorf("Speeds = %d %d %d, want %v [FR-037]", b.Speed(content.SpeedWalk), b.Speed(content.SpeedFly), b.Speed(content.SpeedSwim), speed)
			}
			if rd.Damage() != dice.MustExpression(rend[0], mustDie(t, rend[1]), rend[2]+level) || rd.RangeFeet() != reach[0] ||
				br.Damage() != dice.MustExpression(breath[0], mustDie(t, breath[1]), 0) || br.RangeFeet() != cone[0] {
				t.Errorf("Rend %v reach %d, Breath Weapon %v cone %d, want the text's numbers %v %v [FR-037]", rd.Damage(), rd.RangeFeet(), br.Damage(), br.RangeFeet(), rend, breath)
			}
			for _, r := range b.Resistances() {
				if !strings.Contains(res, r.String()) {
					t.Errorf("Resistance %v is not in %q [FR-037]", r, res)
				}
			}
			if got := strings.Count(res, ",") + 1; got != len(b.Resistances()) {
				t.Errorf("%d Resistances in the text, %d in the stat block [FR-037]", got, len(b.Resistances()))
			}
			imm := sumLine(t, lines, "Immunities")
			for _, c := range b.ConditionImmunities() {
				if !strings.Contains(imm, c.String()) {
					t.Errorf("Immunity %v is not in %q [FR-037]", c, imm)
				}
			}
		}
	})
	t.Run("Otherworldly Steed, p. 131", func(t *testing.T) {
		t.Parallel()
		lines := sumStatBlockLines(t, "find_steed")
		ac := sumInts(t, `^AC (\d+) \+ (\d+) per spell level`, sumLine(t, lines, "AC"))
		hp := sumInts(t, `^HP (\d+) \+ (\d+) per spell level`, sumLine(t, lines, "HP"))
		speed := sumInts(t, `^Speed (\d+) ft\., Fly (\d+) ft\. \(requires level (\d+)\+ spell\)`, sumLine(t, lines, "Speed"))
		slam := sumInts(t, `Hit: (\d+)d(\d+) plus the spell’s level of Radiant \(Celestial\), Psychic \(Fey\), or Necrotic \(Fiend\)`, sumLine(t, lines, "Otherworldly Slam."))
		heal := sumInts(t, `equal to (\d+)d(\d+) plus the spell’s level`, sumLine(t, lines, "Healing Touch"))
		glare := sumInts(t, `within (\d+) feet`, sumLine(t, lines, "Fell Glare"))
		step := sumInts(t, `up to (\d+) feet away`, sumLine(t, lines, "Fey Step"))
		touch := sumInts(t, `within (\d+) feet of the steed`, sumLine(t, lines, "Healing Touch"))
		bond := sumInts(t, `within (\d+) feet of it`, sumLine(t, lines, "Life Bond."))
		if !strings.Contains(sumLine(t, lines, "Languages"), "Telepathy 1 mile (works only with you)") {
			t.Errorf("the Languages line changed: %q", sumLine(t, lines, "Languages"))
		}
		if !strings.Contains(lines[1], "Large") {
			t.Errorf("the size line changed: %q", lines[1])
		}
		for _, level := range []int{2, 3, 4, 9} {
			b := sumBlock(t)(combat.NewOtherworldlySteedStatBlock(combat.SteedFiend, level, facts))
			fly := 0
			if level >= speed[2] {
				fly = speed[1]
			}
			if b.ArmorClass() != ac[0]+ac[1]*level || b.HitPointMaximum() != hp[0]+hp[1]*level || b.HitDice() != dice.MustExpression(level, dice.D10(), 0) ||
				b.Speed(content.SpeedWalk) != speed[0] || b.Speed(content.SpeedFly) != fly || !b.HasLifeBond() || b.Telepathy().RangeFeet() != 5280 {
				t.Errorf("level %d: AC %d HP %d Fly %d, want the text's numbers [FR-037]", level, b.ArmorClass(), b.HitPointMaximum(), b.Speed(content.SpeedFly))
			}
			if s := sumAttack(t, b, combat.SummonedAttackOtherworldlySlam); s.Damage() != dice.MustExpression(slam[0], mustDie(t, slam[1]), level) || s.RangeFeet() != 5 {
				t.Errorf("Slam = %v, want %dd%d+%d [FR-037]", s.Damage(), slam[0], slam[1], level)
			}
			if g := sumAttack(t, b, combat.SummonedAttackFellGlare); g.RangeFeet() != glare[0] || g.SaveDC() != 15 {
				t.Errorf("Fell Glare range %d DC %d, want %d and the spell save DC [FR-037]", g.RangeFeet(), g.SaveDC(), glare[0])
			}
			c := sumBlock(t)(combat.NewOtherworldlySteedStatBlock(combat.SteedCelestial, level, facts))
			if h := sumAttack(t, c, combat.SummonedAttackHealingTouch); h.Damage() != dice.MustExpression(heal[0], mustDie(t, heal[1]), level) || h.RangeFeet() != touch[0] {
				t.Errorf("Healing Touch = %v range %d, want %dd%d+%d within %d [FR-037]", h.Damage(), h.RangeFeet(), heal[0], heal[1], level, touch[0])
			}
			f := sumBlock(t)(combat.NewOtherworldlySteedStatBlock(combat.SteedFey, level, facts))
			if s := sumAttack(t, f, combat.SummonedAttackFeyStep); s.RangeFeet() != step[0] {
				t.Errorf("Fey Step range %d, want %d [FR-037]", s.RangeFeet(), step[0])
			}
		}
		if bond[0] != 5 {
			t.Errorf("Life Bond range in the text = %d, want 5 (FR-057)", bond[0])
		}
	})
}

// mustDie returns the die with the sides.
func mustDie(tb testing.TB, sides int) dice.Die {
	tb.Helper()
	d, err := dice.NewDie(sides)
	if err != nil {
		tb.Fatal(err)
	}
	return d
}
