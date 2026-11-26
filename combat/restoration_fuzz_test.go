package combat_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzNewDeathCircumstances checks the constructor never panics, accepts exactly a non-negative time and a body state of the four,
// returns a range error otherwise, and that every revival spell then answers without panicking.
// Requirement: doc/specs/revival-and-restoration-spells.md FR-008 (testing.md section 10).
// Path: fuzz.
func FuzzNewDeathCircumstances(f *testing.F) {
	for _, s := range []struct {
		ns   int64
		body int
	}{{0, 0}, {-1, 0}, {int64(time.Minute), 3}, {math.MaxInt, 2}, {-1 << 63, 1}, {0, 4}, {0, -1}} {
		f.Add(s.ns, s.body)
	}
	f.Fuzz(func(t *testing.T, ns int64, body int) {
		c, err := combat.NewDeathCircumstances(time.Duration(ns), ns%2 == 0, ns%3 == 0, combat.BodyState(body))
		valid := ns >= 0 && body >= 0 && body <= 3
		var re *combat.RestorationRangeError
		if valid != (err == nil) || (err != nil && (!errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &re))) {
			t.Fatalf("NewDeathCircumstances(%d, %d) error = %v, want an error exactly when invalid", ns, body, err)
		}
		if err != nil && c != (combat.DeathCircumstances{}) {
			t.Fatalf("an error came with %+v", c)
		}
		for _, spell := range []combat.RevivalSpell{combat.Revivify, combat.RaiseDead, combat.Resurrection, combat.TrueResurrection} {
			cmd := combat.ReviveBySpellCommand{Spell: spell, Subject: "hero", Circumstances: c, Consent: combat.SpiritAccepts}
			if combat.ReviveBySpell(cmd) == nil {
				t.Fatal("ReviveBySpell returned nil")
			}
		}
	})
}

// FuzzNewReviveOrdeal checks the constructor accepts exactly 0 to 4 and AfterRest never leaves the range.
// Requirement: doc/specs/revival-and-restoration-spells.md FR-016, FR-017 (testing.md section 10).
// Path: fuzz.
func FuzzNewReviveOrdeal(f *testing.F) {
	for _, n := range []int{0, 4, 5, -1, math.MaxInt, math.MinInt} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		o, err := combat.NewReviveOrdeal(n)
		if (n >= 0 && n <= 4) != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Fatalf("NewReviveOrdeal(%d) error = %v", n, err)
		}
		if err == nil && (o.Penalty() != n || o.Modifier().Value() != -n || o.AfterRest(2).Penalty() != max(0, n-1)) {
			t.Fatalf("NewReviveOrdeal(%d) = %d / %v", n, o.Penalty(), o.Modifier())
		}
	})
}

// FuzzNewHitPointMaximumIncrease checks the constructor never panics, accepts a non-blank id, a valid spell id and an amount of at least 1, and that an accepted
// increase joins a ledger and a creature without panicking.
// Requirement: doc/specs/revival-and-restoration-spells.md FR-023 (testing.md section 10).
// Path: fuzz.
func FuzzNewHitPointMaximumIncrease(f *testing.F) {
	f.Add("e1", "aid", 5)
	f.Add("", "aid", 5)
	f.Add("e1", "", 5)
	f.Add("e1", "aid", 0)
	f.Add("e1", "heroes_feast", math.MaxInt)
	f.Add(" ", "Bad Id", -3)
	f.Fuzz(func(t *testing.T, id, spell string, amount int) {
		inc, err := combat.NewHitPointMaximumIncrease(combat.OngoingEffectID(id), content.SpellID(spell), amount)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !inc.IsZero() {
				t.Fatalf("error %v came with %+v", err, inc)
			}
			return
		}
		st := combat.MustHitPointStatus(20, combat.FallsUnconscious, combat.NoAffinities())
		res, err := combat.IncreaseMaximum("hero", st, combat.HitPointMaximumIncreases{}, inc)
		a, ok := res.(combat.IncreaseMaximumApplied)
		if err != nil || !ok || a.Status().HitPoints().Maximum() < 20 || a.Status().HitPoints().Current() > a.Status().HitPoints().Maximum() {
			t.Fatalf("IncreaseMaximum(%+v) = %v, %v", inc, res, err)
		}
	})
}

// FuzzNewMassHealAllotment checks the constructor accepts exactly non-negative allotments of at most 700 in all and a non-blank id, and never panics.
// Requirement: doc/specs/revival-and-restoration-spells.md FR-040 (testing.md section 10).
// Path: fuzz.
func FuzzNewMassHealAllotment(f *testing.F) {
	f.Add("a", 400, "b", 300)
	f.Add("a", 400, "b", 301)
	f.Add("a", -1, "b", 5)
	f.Add("", 5, "b", 5)
	f.Add("a", math.MaxInt, "a", math.MaxInt)
	f.Fuzz(func(t *testing.T, id1 string, n1 int, id2 string, n2 int) {
		a, err := combat.NewMassHealAllotment(map[combat.CombatantID]int{combat.CombatantID(id1): n1, combat.CombatantID(id2): n2})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || a.Total() != 0 {
				t.Fatalf("error %v came with a total of %d", err, a.Total())
			}
			return
		}
		if a.Total() < 0 || a.Total() > 700 || id1 == "" || id2 == "" {
			t.Fatalf("accepted a total of %d for %q and %q", a.Total(), id1, id2)
		}
	})
}

// FuzzAidIncrease checks Aid accepts exactly a cast level of 2 to 9 and gives 5 for each level above 1.
// Requirement: doc/specs/revival-and-restoration-spells.md FR-023 (testing.md section 10).
// Path: fuzz.
func FuzzAidIncrease(f *testing.F) {
	for _, n := range []int{2, 9, 1, 10, 0, -1, math.MaxInt} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, level int) {
		inc, err := combat.AidIncrease("aid1", level)
		if (level >= 2 && level <= 9) != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Fatalf("AidIncrease(%d) error = %v", level, err)
		}
		if err == nil && inc.Amount() != 5*(level-1) {
			t.Fatalf("AidIncrease(%d) = %d", level, inc.Amount())
		}
	})
}
