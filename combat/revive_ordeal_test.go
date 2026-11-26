package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (table 4.3; Source for every row: SRD 5.2.1, "Raise Dead" p. 157)
// Path: per case.
func TestReviveOrdealAfterRest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		start                   int
		rest                    character.RestKind
		want                    int
	}{
		{"first Long Rest", "FR-017; table 4.3", "happy", 4, character.LongRest, 3},
		{"second Long Rest", "FR-017; table 4.3", "happy", 3, character.LongRest, 2},
		{"third Long Rest", "FR-017; table 4.3", "happy", 2, character.LongRest, 1},
		{"fourth Long Rest", "FR-017; table 4.3", "happy", 1, character.LongRest, 0},
		{"already clear", "FR-017; table 4.3", "unhappy (boundary)", 0, character.LongRest, 0},
		{"Short Rest", "FR-017; table 4.3", "unhappy (rule interaction)", 4, character.ShortRest, 4},
		{"a Long Rest interrupted into Short Rest benefits", "FR-017; table 4.3", "unhappy (rule interaction)", 4, character.ShortRest, 4},
		{"an invalid rest kind", "FR-017", "unhappy (invalid input)", 4, 0, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.MustReviveOrdeal(c.start).AfterRest(c.rest)
			if got != combat.MustReviveOrdeal(c.want) || got.Penalty() != c.want || got.Modifier() != modifier.New(-c.want) {
				t.Errorf("ordeal %d after %v = %d (modifier %v), want %d [%s; %s]", c.start, c.rest, got.Penalty(), got.Modifier(), c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-016 and table 4.3 - the zero ReviveOrdeal is no ordeal (penalty 0, modifier 0), FreshOrdeal is 4
// (modifier -4), the constructor accepts 0 to 4 and returns an error matching srderr.ErrInvalidInput outside it, and equal penalties are equal with ==.
// Path: unhappy (invalid input).
func TestNewReviveOrdeal(t *testing.T) {
	t.Parallel()
	var zero combat.ReviveOrdeal
	if zero.Penalty() != 0 || zero.Modifier() != modifier.New(0) || zero != combat.MustReviveOrdeal(0) {
		t.Errorf("zero ordeal = %d / %v, want none", zero.Penalty(), zero.Modifier())
	}
	if f := combat.FreshOrdeal(); f.Penalty() != 4 || f.Modifier() != modifier.New(-4) || f != combat.MustReviveOrdeal(4) {
		t.Errorf("FreshOrdeal = %d / %v, want 4 / -4", f.Penalty(), f.Modifier())
	}
	for n := 0; n <= 4; n++ {
		if o, err := combat.NewReviveOrdeal(n); err != nil || o.Penalty() != n {
			t.Errorf("NewReviveOrdeal(%d) = %v, %v", n, o, err)
		}
	}
	for _, n := range []int{5, -1, math.MaxInt} {
		o, err := combat.NewReviveOrdeal(n)
		var re *combat.RestorationRangeError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &re) || re.Name != "ordeal_penalty" || re.Value != int64(n) || o != (combat.ReviveOrdeal{}) {
			t.Errorf("NewReviveOrdeal(%d) = %v, %v, want a range error and the zero value", n, o, err)
		}
	}
	defer func() {
		if err, ok := recover().(error); !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an error matching ErrInvalidInput", err)
		}
	}()
	combat.MustReviveOrdeal(5)
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-018, table 4.3 revived while penalised - a successful Raise Dead or Resurrection gives
// the fresh ordeal (4) and Revivify and True Resurrection give none; a creature that held an ordeal of 2 and is revived again is at 4, not 6 or 2 (the value
// is the fresh one, never a sum).
// Path: unhappy (rule interaction).
func TestReviveBySpellOrdeal(t *testing.T) {
	t.Parallel()
	for spell, want := range map[combat.RevivalSpell]combat.ReviveOrdeal{combat.Revivify: {}, combat.RaiseDead: combat.FreshOrdeal(),
		combat.Resurrection: combat.FreshOrdeal(), combat.TrueResurrection: {}} {
		if got := rrApplied(t, rrCommand(t, spell)).Ordeal; got != want {
			t.Errorf("%v: Ordeal = %d, want %d", spell, got.Penalty(), want.Penalty())
		}
	}
	held := combat.MustReviveOrdeal(2)
	if got := rrApplied(t, rrCommand(t, combat.RaiseDead)).Ordeal; got == held || got.Penalty() != 4 {
		t.Errorf("revived while holding %d: Ordeal = %d, want the fresh 4", held.Penalty(), got.Penalty())
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-019 - the ordeal is one more modifier on every D20 Test, summed with the Exhaustion reduction
// (SRD 5.2.1, "Raise Dead" p. 157; "Exhaustion" p. 181): an ordeal of 4 and Exhaustion 2 add -4 and -4 to a roll, never the larger of the two.
// Path: unhappy (rule interaction).
func TestReviveOrdealStacksWithExhaustion(t *testing.T) {
	t.Parallel()
	set := meConditions(combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(2)})
	fx := combat.ConditionEffects{}
	mods := d20.NewRollModifiers(modifier.New(3)).WithOthers(combat.FreshOrdeal().Modifier(), modifier.New(-fx.D20TestReduction(set)))
	if got := mods.Total(); got != modifier.New(3-4-4) {
		t.Errorf("RollModifiers.Total = %v, want -5 (ability +3, ordeal -4, Exhaustion -4)", got)
	}
}
