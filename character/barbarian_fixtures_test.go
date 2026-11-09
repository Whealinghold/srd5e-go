package character_test

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

// brbDef is the bundled Barbarian ClassDefinition (content/data/class_barbarian.json).
func brbDef(tb testing.TB) content.ClassDefinition {
	tb.Helper()
	src, err := data.NewBundledSource()
	if err != nil {
		tb.Fatal(err)
	}
	res, err := src.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID("barbarian")))
	if err != nil {
		tb.Fatal(err)
	}
	return res.(content.LookupFound).Entry.(content.ClassDefinition)
}

func brbProgression(tb testing.TB) character.BarbarianProgression {
	tb.Helper()
	return must(character.NewBarbarianProgression(brbDef(tb)))
}

// brbStats are the stats of the Barbarian level n.
func brbStats(tb testing.TB, n int) character.BarbarianLevelStats {
	tb.Helper()
	return brbProgression(tb).StatsAt(character.MustBarbarianLevel(n))
}

// brbChar is a character with the given class levels, enough Experience Points for level 20, flat scores of 10 and a stored maximum of 200.
func brbChar(t *testing.T, entries ...character.ClassLevel) character.Character {
	t.Helper()
	return restored(t, 355000, 200, flatScores(t, 10), entries...)
}

// brbBarbarian is a Barbarian of level n and nothing else.
func brbBarbarian(t *testing.T, n int) character.Character {
	t.Helper()
	return brbChar(t, cl("barbarian", n))
}

// brbWith returns c holding the BarbarianState state.
func brbWith(t *testing.T, c character.Character, state character.BarbarianState) character.Character {
	t.Helper()
	return must(character.WithFeatureState(c, character.BarbarianStateKey(), state))
}

// brbStateOf reads the BarbarianState c stores.
func brbStateOf(t *testing.T, c character.Character) character.BarbarianState {
	t.Helper()
	s, ok, err := character.ReadFeatureState(c.FeatureStates(), character.BarbarianStateKey())
	if err != nil || !ok {
		t.Fatalf("BarbarianState = (%v, %v, %v), want a stored state", s, ok, err)
	}
	return s
}

// brbRaging returns the state of a Barbarian of level n that entered a Rage this turn with expended uses already spent.
func brbRaging(t *testing.T, n, expended int) character.BarbarianState {
	t.Helper()
	return brbEnter(t, n, character.MustBarbarianState(expended, 0), "")
}

// brbEnter enters a Rage on a fresh turn and returns the resulting state.
func brbEnter(t *testing.T, n int, state character.BarbarianState, subclass content.SubclassID) character.BarbarianState {
	t.Helper()
	res := must(character.BarbarianRage{}.Enter(brbID(t), brbStats(t, n), state, character.NewBarbarianTurn(), character.BarbarianHostFacts{}, subclass, 30))
	entered, ok := res.(character.RageEntered)
	if !ok {
		t.Fatalf("Enter = %#v, want RageEntered", res)
	}
	return entered.State()
}

// brbRagingLater returns a Rage of level n that began on an earlier turn (the Barbarian's turn ended once).
func brbRagingLater(t *testing.T, n int) character.BarbarianState {
	t.Helper()
	state := brbRaging(t, n, 0)
	res := must(character.BarbarianRage{}.End(brbID(t), brbStats(t, n), state, character.RageTurnEnded{}))
	return res.(character.RageEndApplied).State()
}

// brbID is a character id for the pure services.
func brbID(t *testing.T) character.Identifier {
	t.Helper()
	return brbBarbarian(t, 1).ID()
}

// brbStrengthAttack is a Strength weapon attack that deals damage and hits.
func brbStrengthAttack() character.BarbarianAttackFacts {
	return character.BarbarianAttackFacts{UsesStrength: true, Kind: character.BarbarianWeaponAttack, DealsDamage: true, DamageType: content.DamageSlashing, Hit: true}
}

// brbSave is a Constitution saving throw with the dice rolled under mode, a modifier of mods and the Difficulty Class dc.
func brbSave(tb testing.TB, mode modifier.RollMode, dice []int, mods, dc int) d20.CheckResult {
	tb.Helper()
	return fgtCheck(tb, d20.SavingThrow, mode, dice, mods, dc)
}

// brbMod is the ability modifier of the score.
func brbMod(tb testing.TB, score int) ability.Modifier {
	tb.Helper()
	return must(ability.NewScore(score)).Modifier()
}
