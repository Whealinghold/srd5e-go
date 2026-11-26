package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// Requirement: doc/specs/item-use-effects.md FR-050, table 4.7a - HealersKit accepts 0 to 10 uses and returns an error matching srderr.ErrInvalidInput outside it; a fresh kit holds 10; the zero kit is empty; MustHealersKit panics with that error.
// Path: unhappy (boundary).
func TestHealersKitUses(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 1, 9, 10} {
		if k, err := combat.NewHealersKit(n); err != nil || k.UsesRemaining() != n || combat.MustHealersKit(n) != k {
			t.Errorf("NewHealersKit(%d) = %v, %v", n, k, err)
		}
	}
	for _, n := range []int{-1, 11, math.MaxInt, math.MinInt} {
		if _, err := combat.NewHealersKit(n); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldUsesRemaining, Value: int64(n)}) {
			t.Errorf("NewHealersKit(%d) err = %v", n, err)
		}
	}
	if combat.FreshHealersKit().UsesRemaining() != 10 || (combat.HealersKit{}).UsesRemaining() != 0 || combat.FreshHealersKit() != combat.MustHealersKit(10) {
		t.Error("fresh or zero kit holds the wrong uses")
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), &combat.ItemUseError{Field: combat.ItemUseFieldUsesRemaining, Value: 11}) {
			t.Errorf("MustHealersKit(11) panicked with %v", r)
		}
	}()
	combat.MustHealersKit(11)
}

func useKit(tb testing.TB, kit combat.HealersKit, st combat.HitPointStatus) combat.HealersKitOutcome {
	tb.Helper()
	out, err := kit.UseToStabilize(gid("medic"), gid("hero"), st)
	if err != nil {
		tb.Fatal(err)
	}
	return out
}

// Requirement: per case
// Path: per case.
func TestHealersKitUseToStabilize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		kit                     combat.HealersKit
		status                  combat.HitPointStatus
		want                    string // used, empty or a HitPointRejection
		uses                    int
	}{
		{"fresh kit, Dying patient", itemUseSpec + "FR-051 table 4.7a", "happy", combat.FreshHealersKit(), dyingStatus(t, 20, 1, 2), "used", 9},
		{"last use", itemUseSpec + "FR-051 table 4.7a", "unhappy (boundary)", combat.MustHealersKit(1), dyingStatus(t, 20, 0, 0), "used", 0},
		{"empty kit", itemUseSpec + "FR-052 table 4.7a", "unhappy (state error)", combat.MustHealersKit(0), dyingStatus(t, 20, 0, 0), "empty", 0},
		{"empty kit, Dead patient", itemUseSpec + "FR-052", "unhappy (rule interaction)", combat.HealersKit{}, deadStatus(t, combat.MassiveDamage), "empty", 0},
		{"Stable patient", itemUseSpec + "FR-053 table 4.7a", "unhappy (state error)", combat.FreshHealersKit(), stablePC(t), "not_dying", 10},
		{"Alive patient", itemUseSpec + "FR-053 table 4.7a", "unhappy (state error)", combat.FreshHealersKit(), pcStatus(t, 5, 20, 0), "not_dying", 10},
		{"Dead patient", itemUseSpec + "FR-053 table 4.7a", "unhappy (state error)", combat.FreshHealersKit(), deadStatus(t, combat.MassiveDamage), "creature_is_dead", 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out := useKit(t, c.kit, c.status)
			switch v := out.(type) {
			case combat.HealersKitUsed:
				if c.want != "used" || v.Kit().UsesRemaining() != c.uses || v.Cost() != combat.ItemActionCostUtilizeAction || v.Status().LifeState() != (combat.Stable{}) {
					t.Fatalf("used: kit %d, cost %v, status %v [%s]", v.Kit().UsesRemaining(), v.Cost(), v.Status().LifeState(), c.requirement)
				}
				ev := v.Events()
				if len(ev) != 2 {
					t.Fatalf("events = %d, want became_stable and healers_kit_used", len(ev))
				}
				combattest.CheckEvent(t, ev[0], "became_stable", gid("hero"))
				combattest.CheckEvent(t, ev[1], "healers_kit_used", gid("medic"), gid("hero"))
				ev[0] = nil
				if v.Events()[0] == nil {
					t.Error("Events returns the stored slice [FR-075]")
				}
			case combat.HealersKitEmpty:
				if c.want != "empty" {
					t.Errorf("empty, want %s [%s]", c.want, c.requirement)
				}
			case combat.HealersKitRefused:
				if v.Rejection.String() != c.want {
					t.Errorf("refused %v, want %s [%s]", v.Rejection, c.want, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-051, FR-054, FR-075 - a Dying patient's tallies are reset by the kit; the patient's status is not altered in place; a self-use names the creature once; the command arguments must be real.
// Path: unhappy (immutability, invalid input).
func TestHealersKitUseToStabilizeArguments(t *testing.T) {
	t.Parallel()
	st := dyingStatus(t, 20, 2, 2)
	out := useKit(t, combat.FreshHealersKit(), st)
	if got := out.(combat.HealersKitUsed).Status().LifeState(); got != (combat.Stable{}) {
		t.Errorf("life = %v, want Stable with the tallies reset", got)
	}
	if st.LifeState() != dying(2, 2) {
		t.Errorf("the patient's status changed to %v", st.LifeState())
	}
	self, err := combat.FreshHealersKit().UseToStabilize(gid("hero"), gid("hero"), st)
	if err != nil || !slices.Equal(self.(combat.HealersKitUsed).Events()[1].Combatants(), []combat.CombatantID{gid("hero")}) {
		t.Errorf("self use: %v, %v", self, err)
	}
	bad := []struct {
		name          string
		user, patient combat.CombatantID
		st            combat.HitPointStatus
		field         combat.ItemUseField
	}{
		{"user", "", "p", st, combat.ItemUseFieldUser}, {"patient", "u", "", st, combat.ItemUseFieldPatient}, {"status", "u", "p", combat.HitPointStatus{}, combat.ItemUseFieldStatus},
	}
	for _, b := range bad {
		if _, err := combat.FreshHealersKit().UseToStabilize(b.user, b.patient, b.st); !errors.Is(err, &combat.ItemUseError{Field: b.field}) {
			t.Errorf("zero %s: %v", b.name, err)
		}
	}
}
