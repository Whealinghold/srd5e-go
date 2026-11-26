package combat_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func advanceOil(tb testing.TB, c combat.OilCoat, d time.Duration) combat.OilCoat {
	tb.Helper()
	n, err := c.Advance(d)
	if err != nil {
		tb.Fatal(err)
	}
	return n
}

// Requirement: per case
// Path: per case.
func TestOilCoatAdvance(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		steps                   []time.Duration
		wantState               combat.OilCoatState
		wantElapsed             time.Duration
	}{
		{"fresh coat is wet", itemUseSpec + "FR-016", "happy", nil, combat.OilCoatWet, 0},
		{"59 seconds stays wet", itemUseSpec + "FR-016 table 4.3a", "unhappy (boundary)", []time.Duration{59 * time.Second}, combat.OilCoatWet, 59 * time.Second},
		{"60 seconds dries", itemUseSpec + "FR-016 table 4.3a", "unhappy (boundary)", []time.Duration{60 * time.Second}, combat.OilCoatDried, time.Minute},
		{"two 30 second advances dry", itemUseSpec + "FR-016 table 4.3a", "unhappy (boundary)", []time.Duration{30 * time.Second, 30 * time.Second}, combat.OilCoatDried, time.Minute},
		{"zero advance changes nothing", itemUseSpec + "FR-016", "unhappy (empty)", []time.Duration{0}, combat.OilCoatWet, 0},
		{"dried stays dried", itemUseSpec + "FR-016", "unhappy (state error)", []time.Duration{time.Hour, time.Second}, combat.OilCoatDried, time.Hour + time.Second},
		{"the sum saturates", itemUseSpec + "FR-016", "unhappy (boundary)", []time.Duration{math.MaxInt64, time.Hour}, combat.OilCoatDried, math.MaxInt64},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			coat := combat.NewOilCoat()
			for _, s := range c.steps {
				coat = advanceOil(t, coat, s)
			}
			if coat.State() != c.wantState || coat.Elapsed() != c.wantElapsed {
				t.Errorf("coat = %v after %v, want %v after %v [%s]", coat.State(), coat.Elapsed(), c.wantState, c.wantElapsed, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-016, FR-018, table 4.3a - Advance(-1 s) returns an error matching srderr.ErrInvalidInput carrying the field and value; a burned-off coat does not change on Advance except its time; a second failed Oil save gives a fresh coat, never a second.
// Path: unhappy (invalid input, state error).
func TestOilCoatAdvanceRejectsNegativeAndKeepsBurnedOff(t *testing.T) {
	t.Parallel()
	_, err := combat.NewOilCoat().Advance(-time.Second)
	if !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldElapsed, Value: int64(-time.Second)}) {
		t.Errorf("Advance(-1s) err = %v, want elapsed -1s", err)
	}
	ign, _ := combat.NewOilCoat().OnFireDamage(1)
	burned := ign.(combat.OilIgnited).Coat()
	if got := advanceOil(t, burned, time.Hour); got.State() != combat.OilCoatBurnedOff {
		t.Errorf("burned-off coat after Advance = %v", got.State())
	}
	wet := advanceOil(t, combat.NewOilCoat(), 30*time.Second)
	if wet.Elapsed() != 30*time.Second || combat.NewOilCoat() == wet || combat.NewOilCoat().Elapsed() != 0 {
		t.Errorf("a fresh coat must be the zero coat, not the 30 second one: %+v", wet)
	}
	if (combat.OilCoat{}) != combat.NewOilCoat() {
		t.Error("the zero OilCoat is not a fresh wet coat")
	}
}

// Requirement: per case
// Path: per case.
func TestOilCoatOnFireDamage(t *testing.T) {
	t.Parallel()
	dried := advanceOil(t, combat.NewOilCoat(), time.Minute)
	ign, _ := combat.NewOilCoat().OnFireDamage(2)
	burned := ign.(combat.OilIgnited).Coat()
	cases := []struct {
		name, requirement, path string
		coat                    combat.OilCoat
		damage                  int
		ignited                 bool
		reason                  combat.OilNotIgnitedReason
	}{
		{"1 Fire damage on wet oil", itemUseSpec + "FR-017 table 4.3a", "unhappy (boundary)", combat.NewOilCoat(), 1, true, 0},
		{"large Fire damage on wet oil", itemUseSpec + "FR-017", "happy", combat.NewOilCoat(), 40, true, 0},
		{"0 Fire damage on wet oil", itemUseSpec + "FR-018 table 4.3a", "unhappy (boundary)", combat.NewOilCoat(), 0, false, combat.OilNotIgnitedNoFireDamageTaken},
		{"Fire damage on dried oil", itemUseSpec + "FR-018 table 4.3a", "unhappy (state error)", dried, 5, false, combat.OilNotIgnitedDried},
		{"a second Fire damage after ignition", itemUseSpec + "FR-018 table 4.3a", "unhappy (state error)", burned, 5, false, combat.OilNotIgnitedBurnedOff},
		{"dried with no damage", itemUseSpec + "FR-018", "unhappy (rule interaction)", dried, 0, false, combat.OilNotIgnitedDried},
		{"the largest damage", itemUseSpec + "FR-017", "unhappy (boundary)", combat.NewOilCoat(), math.MaxInt, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.coat.OnFireDamage(c.damage)
			if err != nil {
				t.Fatalf("OnFireDamage: %v [%s]", err, c.requirement)
			}
			switch v := got.(type) {
			case combat.OilIgnited:
				if !c.ignited || v.Damage() != combat.MustDamage(5, content.DamageFire) || v.Coat().State() != combat.OilCoatBurnedOff || v.Coat().Elapsed() != c.coat.Elapsed() {
					t.Errorf("ignited = %+v [%s]", v, c.requirement)
				}
			case combat.OilNotIgnited:
				if c.ignited || v.Reason() != c.reason || v.Coat() != c.coat {
					t.Errorf("not ignited = %v with coat %+v, want %v and the coat unchanged [%s]", v.Reason(), v.Coat(), c.reason, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-017 - a negative Fire damage taken returns an error matching srderr.ErrInvalidInput with the field and value, and the reasons print their snake_case values.
// Path: unhappy (invalid input).
func TestOilCoatOnFireDamageRejectsNegative(t *testing.T) {
	t.Parallel()
	for _, n := range []int{-1, math.MinInt} {
		if _, err := combat.NewOilCoat().OnFireDamage(n); !errors.Is(err, &combat.ItemUseError{Field: combat.ItemUseFieldDamageTaken, Value: int64(n)}) {
			t.Errorf("OnFireDamage(%d) err = %v", n, err)
		}
	}
	for r, want := range map[combat.OilNotIgnitedReason]string{combat.OilNotIgnitedDried: "dried", combat.OilNotIgnitedBurnedOff: "burned_off",
		combat.OilNotIgnitedNoFireDamageTaken: "no_fire_damage_taken", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("reason %d = %q, want %q", r, r.String(), want)
		}
	}
	for s, want := range map[combat.OilCoatState]string{combat.OilCoatWet: "wet", combat.OilCoatDried: "dried", combat.OilCoatBurnedOff: "burned_off", -1: "unknown", 7: "unknown"} {
		if s.String() != want {
			t.Errorf("state %d = %q, want %q", s, s.String(), want)
		}
	}
}
