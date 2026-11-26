package combat_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	rrSpec = "doc/specs/revival-and-restoration-spells.md "
	rrDay  = 24 * time.Hour
)

// rrCircs builds death circumstances of the given time and body, failing the test on a bad one.
func rrCircs(tb testing.TB, since time.Duration, oldAge, undead bool, body combat.BodyState) combat.DeathCircumstances {
	tb.Helper()
	c, err := combat.NewDeathCircumstances(since, oldAge, undead, body)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// rrEffects is the baseline of table 4.2: the curse c1, the contagion c2, the spell s1 and the poison p1.
func rrEffects() combat.OngoingEffects {
	return meEffects(meCurse("c1"), meEffect("c2", combat.OngoingEffectMagicalContagion, meLabeled("rot")),
		meEffect("s1", combat.OngoingEffectSpell, meSpell("bless")), meEffect("p1", combat.OngoingEffectMagicalEffect, meLabeled("venom")))
}

// rrCommand is the baseline of table 4.2: a Dead creature of maximum 20, Poisoned and Blinded, the four
// effects, p1 a poison and the Poisoned condition from poison, the spirit accepting, an intact body just dead.
func rrCommand(tb testing.TB, spell combat.RevivalSpell) combat.ReviveBySpellCommand {
	tb.Helper()
	return combat.ReviveBySpellCommand{Spell: spell, Subject: gid("hero"), Status: deadStatus(tb, combat.DeathSavingThrowFailures),
		Conditions: meConditions(combat.Poisoned{}, combat.Blinded{}), Effects: rrEffects(), Consent: combat.SpiritAccepts,
		Poisons: combat.PoisonsAtDeath{Effects: []combat.OngoingEffectID{meEID("p1")}, PoisonedConditionFromPoison: true}}
}

// rrApplied returns the applied result of a cast or fails the test.
func rrApplied(tb testing.TB, c combat.ReviveBySpellCommand) combat.ReviveBySpellApplied {
	tb.Helper()
	res, ok := combat.ReviveBySpell(c).(combat.ReviveBySpellApplied)
	if !ok {
		tb.Fatalf("ReviveBySpell = %T, want ReviveBySpellApplied", combat.ReviveBySpell(c))
	}
	return res
}

func rrEffectIDs(e combat.OngoingEffects) []string {
	out := []string{}
	for _, x := range e.All() {
		out = append(out, x.ID().String())
	}
	return out
}

func rrConditionKinds(s combat.ConditionSet) []content.ConditionKind {
	out := []content.ConditionKind{}
	for _, c := range s.Explicit() {
		out = append(out, c.Kind())
	}
	return out
}

func rrKinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

func rrSpellID(s string) content.SpellID { return content.MustSpellID(s) }

// rrIncrease builds an increase of an effect for a spell.
func rrIncrease(tb testing.TB, id, spell string, amount int) combat.HitPointMaximumIncrease {
	tb.Helper()
	i, err := combat.NewHitPointMaximumIncrease(meEID(id), rrSpellID(spell), amount)
	if err != nil {
		tb.Fatal(err)
	}
	return i
}

func rrLedger(tb testing.TB, entries ...combat.HitPointMaximumIncrease) combat.HitPointMaximumIncreases {
	tb.Helper()
	l, err := combat.NewHitPointMaximumIncreases(entries)
	if err != nil {
		tb.Fatal(err)
	}
	return l
}

// rrRoll builds a roll of count dice of the given sides with the given faces and modifier.
func rrRoll(tb testing.TB, count, sides, mod int, faces ...int) dice.Result {
	tb.Helper()
	die, err := dice.NewDie(sides)
	if err != nil {
		tb.Fatal(err)
	}
	expr, err := dice.NewExpression(count, die, mod)
	if err != nil {
		tb.Fatal(err)
	}
	var rolls []dice.DieRoll
	for _, f := range faces {
		roll, err := dice.NewDieRoll(die, f)
		if err != nil {
			tb.Fatal(err)
		}
		rolls = append(rolls, roll)
	}
	r, err := dice.NewResult(expr, rolls)
	if err != nil {
		tb.Fatal(err)
	}
	return r
}
