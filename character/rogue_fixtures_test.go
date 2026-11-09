package character_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rogDef is the bundled Rogue ClassDefinition (content/data/class_rogue.json).
func rogDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatal(err)
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("rogue")))
	if err != nil {
		tb.Fatal(err)
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

func rogProgression(tb testing.TB) character.RogueProgression {
	tb.Helper()
	return must(character.NewRogueProgression(rogDef(tb)))
}

func rl(n int) character.RogueLevel { return character.MustRogueLevel(n) }

// rogAttack is an attack roll that hits (hit true, natural 15 against Armor Class 10) or misses (natural 5 against 20).
func rogAttack(tb testing.TB, hit bool) d20.AttackRollResult {
	tb.Helper()
	natural, ac := 15, 10
	if !hit {
		natural, ac = 5, 20
	}
	return must(d20.ResolveAttack(d20.MustRoll(modifier.Normal, natural), d20.NewRollModifiers(modifier.New(0)), d20.ArmorClass(ac), 20))
}

func dice6(n int) character.SneakAttackDice { return must(character.NewSneakAttackDice(n)) }

// problemsOf returns the problems of a *RogueProblemsError[P] in err.
func rogProblems[P comparable](tb testing.TB, err error) []P {
	tb.Helper()
	var pe *character.RogueProblemsError[P]
	if !errors.As(err, &pe) {
		tb.Fatalf("error = %v, want a *RogueProblemsError", err)
	}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		tb.Errorf("error = %v, want ErrInvalidInput", err)
	}
	return pe.Problems()
}

// rogAttackT is a hit for helpers that have no testing.TB.
func rogAttackT() d20.AttackRollResult {
	return must(d20.ResolveAttack(d20.MustRoll(modifier.Normal, 15), d20.NewRollModifiers(modifier.New(0)), d20.ArmorClass(10), 20))
}
