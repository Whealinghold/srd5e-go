package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzLimitedUseConstructors feeds the constructors of the limited-use values and the roll map of
// StartOfOwnTurn host numbers and text. It asserts properties: nothing panics, a rejected value is the zero
// value with an error matching srderr.ErrInvalidInput, an accepted value reads back what was given, and a
// roll of a d6 recharges a Recharge 5-6 part exactly for the faces 5 and 6.
func FuzzLimitedUseConstructors(f *testing.F) {
	f.Add(2, 0, 0, 0, 1, "b", 5, 6)
	f.Add(-1, -1, -1, -1, 0, "", 0, 0)
	f.Add(math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt, "x y", math.MinInt, math.MaxInt)
	f.Add(5, 99, 1, 1, 2, "drake", 3, 20)
	f.Fuzz(func(t *testing.T, section, index, group, spell, round int, who string, face, sides int) {
		if r, err := combat.NewLimitedUsePartRef(combat.MonsterPartSection(section), index); err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !r.IsZero() {
				t.Fatalf("rejected part ref: %v, %v", r, err)
			}
		} else if int(r.Section()) != section || r.Index() != index {
			t.Fatalf("accepted part ref reads back %v/%d, want %d/%d", r.Section(), r.Index(), section, index)
		}
		if r, err := combat.NewSpellUseRef(group, spell); err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || r != (combat.SpellUseRef{}) {
				t.Fatalf("rejected spell ref: %v, %v", r, err)
			}
		} else if r.GroupIndex() != group || r.SpellIndex() != spell {
			t.Fatalf("accepted spell ref reads back %d/%d", r.GroupIndex(), r.SpellIndex())
		}
		if id, err := combat.NewCombatantID(who); err == nil {
			if w, err := combat.NewLegendaryActionWindow(id, round); err != nil {
				if !errors.Is(err, srderr.ErrInvalidInput) || !w.IsZero() {
					t.Fatalf("rejected window: %v, %v", w, err)
				}
			} else if w.Round() != round || w.IsZero() {
				t.Fatalf("accepted window reads back round %d", w.Round())
			}
		}
		s := luSpent(t, luStart(t, luDrake(t), combat.OutsideLair), luSpray(t))
		die, err := dice.NewDie(sides)
		if err != nil {
			return
		}
		roll, err := dice.NewDieRoll(die, face)
		if err != nil {
			return
		}
		tr, err := s.StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{luSpray(t): roll})
		if sides != 6 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("a d%d roll was accepted: %v", sides, err)
			}
			return
		}
		if err != nil {
			t.Fatalf("a d6 roll was rejected: %v", err)
		}
		if ready := luReadiness(t, tr.State(), luSpray(t)) == combat.RechargeReady; ready != (face >= 5) {
			t.Fatalf("face %d: ready = %v, want %v", face, ready, face >= 5)
		}
	})
}
