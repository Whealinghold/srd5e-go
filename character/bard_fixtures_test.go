package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// brdBard is a Bard of level n with a Charisma score and a full Bardic Inspiration pool sized from it.
func brdBard(t *testing.T, n, charisma int) character.Character {
	t.Helper()
	scores := must(flatScores(t, 10).With(ability.Charisma, ability.MustScore(charisma)))
	c := restored(t, 355000, 200, scores, cl("bard", n))
	return must(character.WithFeatureState(c, character.BardicInspirationPoolKey(), character.InitialBardicInspirationPool(scores.Modifier(ability.Charisma))))
}

// brdWithPool returns c holding a pool of remaining uses out of maximum.
func brdWithPool(t *testing.T, c character.Character, maximum, remaining int) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.BardicInspirationPoolKey(), must(character.NewBardicInspirationPool(maximum, remaining))))
}

// brdPoolOf reads the pool c stores.
func brdPoolOf(t *testing.T, c character.Character) character.BardicInspirationPool {
	t.Helper()
	p, ok, err := character.ReadFeatureState(c.FeatureStates(), character.BardicInspirationPoolKey())
	if err != nil || !ok {
		t.Fatalf("BardicInspirationPool = (%v, %v, %v), want a stored state", p, ok, err)
	}
	return p
}

// brdMod is the Charisma modifier of a score.
func brdMod(t *testing.T, score int) ability.Modifier {
	t.Helper()
	return ability.MustScore(score).Modifier()
}

// brdCheck is a rolled ability check or saving throw: the natural d20, a flat modifier and the DC.
func brdCheck(t *testing.T, kind d20.Kind, natural, mod, dc int) d20.CheckResult {
	t.Helper()
	return must(d20.ResolveCheck(kind, d20.MustRoll(modifier.Normal, natural), d20.NewRollModifiers(modifier.New(mod)), d20.DifficultyClass(dc)))
}

// brdAbility is an adjustable ability check.
func brdAbility(t *testing.T, natural, mod, dc int) character.AdjustableCheck {
	t.Helper()
	return character.AdjustableCheck{Result: brdCheck(t, d20.AbilityCheck, natural, mod, dc)}
}

// brdSave is an adjustable saving throw.
func brdSave(t *testing.T, natural, mod, dc int) character.AdjustableCheck {
	t.Helper()
	return character.AdjustableCheck{Result: brdCheck(t, d20.SavingThrow, natural, mod, dc)}
}

// brdAttack is an adjustable attack roll.
func brdAttack(t *testing.T, natural, mod, ac int) character.AdjustableAttack {
	t.Helper()
	return character.AdjustableAttack{Result: must(d20.ResolveAttack(d20.MustRoll(modifier.Normal, natural), d20.NewRollModifiers(modifier.New(mod)), d20.ArmorClass(ac), d20.DefaultCriticalRange))}
}

const abilityCharisma = ability.Charisma

func scoreOf(n int) ability.Score { return ability.MustScore(n) }
