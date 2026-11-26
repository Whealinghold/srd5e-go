package combat_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestAidIncrease(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level, want             int
		bad                     bool
	}{
		{"level 2", `FR-023; SRD 5.2.1, "Aid", p. 107`, "happy", 2, 5, false},
		{"level 3", `FR-023; SRD 5.2.1, "Aid", p. 107`, "happy", 3, 10, false},
		{"level 9", `FR-023; SRD 5.2.1, "Aid", p. 107`, "unhappy (boundary)", 9, 40, false},
		{"level 1", `FR-023; table 4.5 level 1 / 10`, "unhappy (invalid input)", 1, 0, true},
		{"level 10", `FR-023; table 4.5 level 1 / 10`, "unhappy (invalid input)", 10, 0, true},
		{"level 0", `FR-023`, "unhappy (invalid input)", 0, 0, true},
		{"level -3", `FR-023`, "unhappy (invalid input)", -3, 0, true},
		{"the largest level", `FR-023`, "unhappy (invalid input)", math.MaxInt, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.AidIncrease(meEID("aid1"), c.level)
			if c.bad {
				var re *combat.RestorationRangeError
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &re) || re.Name != "cast_level" || !got.IsZero() {
					t.Errorf("AidIncrease(level %d) = %+v, %v, want a cast_level range error [%s; %s]", c.level, got, err, c.requirement, c.path)
				}
				return
			}
			if err != nil || got.Amount() != c.want || got.Spell() != "aid" || got.ID() != meEID("aid1") {
				t.Errorf("AidIncrease(level %d) = %+v, %v, want %d from aid [%s; %s]", c.level, got, err, c.want, c.requirement, c.path)
			}
		})
	}
	if _, err := combat.AidIncrease("", 2); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("AidIncrease with a blank effect id: %v, want an invalid input error", err)
	}
}

// Requirement: per case
// Path: per case.
func TestHeroesFeastIncrease(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		roll                    dice.Result
		want                    int
		bad                     bool
	}{
		{"a 7", `FR-024; SRD 5.2.1, "Heroes' Feast", p. 140`, "happy", rrRoll(t, 2, 10, 0, 3, 4), 7, false},
		{"the lowest roll", `FR-024`, "unhappy (boundary)", rrRoll(t, 2, 10, 0, 1, 1), 2, false},
		{"the highest roll", `FR-024`, "unhappy (boundary)", rrRoll(t, 2, 10, 0, 10, 10), 20, false},
		{"3d10", `FR-024; table 4.5`, "unhappy (invalid input)", rrRoll(t, 3, 10, 0, 1, 2, 3), 0, true},
		{"2d10+1", `FR-024; table 4.5`, "unhappy (invalid input)", rrRoll(t, 2, 10, 1, 1, 2), 0, true},
		{"2d8", `FR-024`, "unhappy (invalid input)", rrRoll(t, 2, 8, 0, 1, 2), 0, true},
		{"1d10", `FR-024`, "unhappy (invalid input)", rrRoll(t, 1, 10, 0, 4), 0, true},
		{"the zero result", `FR-024`, "unhappy (empty and minimal)", dice.Result{}, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.HeroesFeastIncrease(meEID("feast1"), c.roll)
			if c.bad {
				if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
					t.Errorf("HeroesFeastIncrease(%v) = %+v, %v, want an invalid input error [%s; %s]", c.roll, got, err, c.requirement, c.path)
				}
				return
			}
			if err != nil || got.Amount() != c.want || got.Spell() != "heroes_feast" {
				t.Errorf("HeroesFeastIncrease(%v) = %+v, %v, want %d [%s; %s]", c.roll, got, err, c.want, c.requirement, c.path)
			}
		})
	}
	if _, err := combat.HeroesFeastIncrease("", rrRoll(t, 2, 10, 0, 3, 4)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank effect id: %v, want an invalid input error", err)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-023 - NewHitPointMaximumIncrease rejects an amount below 1 (a range error), a blank effect id and an
// invalid spell id with an error matching srderr.ErrInvalidInput; the zero increase is invalid.
// Path: unhappy (invalid input).
func TestNewHitPointMaximumIncrease(t *testing.T) {
	t.Parallel()
	good, err := combat.NewHitPointMaximumIncrease(meEID("e1"), "aid", 1)
	if err != nil || good.Amount() != 1 || good.ID() != meEID("e1") || good.Spell() != "aid" || good.IsZero() {
		t.Fatalf("NewHitPointMaximumIncrease(1) = %+v, %v", good, err)
	}
	for _, c := range []struct {
		name   string
		id     combat.OngoingEffectID
		spell  content.SpellID
		amount int
	}{{"amount 0", "e1", "aid", 0}, {"amount -1", "e1", "aid", -1}, {"smallest amount", "e1", "aid", math.MinInt}, {"blank id", " ", "aid", 5},
		{"empty id", "", "aid", 5}, {"empty spell", "e1", "", 5}, {"bad spell", "e1", "No Spell!", 5}} {
		got, err := combat.NewHitPointMaximumIncrease(c.id, c.spell, c.amount)
		if !errors.Is(err, srderr.ErrInvalidInput) || !got.IsZero() {
			t.Errorf("%s: NewHitPointMaximumIncrease = %+v, %v, want an invalid input error", c.name, got, err)
		}
	}
	if !(combat.HitPointMaximumIncrease{}).IsZero() {
		t.Error("the zero increase is not IsZero")
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-025 - the effective total is the sum over distinct spells of the largest amount of each spell
// (SRD 5.2.1, "Combining Spell Effects", p. 106).
// Path: unhappy (rule interaction).
func TestHitPointMaximumIncreasesTotal(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		entries []combat.HitPointMaximumIncrease
		want    int
	}{
		{"the zero ledger", nil, 0},
		{"one entry", []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 5)}, 5},
		{"two casts of one spell take the larger", []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 5), rrIncrease(t, "b", "aid", 10)}, 10},
		{"the larger first", []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 10), rrIncrease(t, "b", "aid", 5)}, 10},
		{"equal casts", []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 5), rrIncrease(t, "b", "aid", 5)}, 5},
		{"two spells add", []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 5), rrIncrease(t, "b", "heroes_feast", 7)}, 12},
		{"a sum past the largest int stops there", []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", math.MaxInt), rrIncrease(t, "b", "heroes_feast", 7)}, math.MaxInt},
	}
	for _, c := range cases {
		if got := rrLedger(t, c.entries...).Total(); got != c.want {
			t.Errorf("%s: Total = %d, want %d", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md section 4.5, FR-029 and plan P10 - the ledger copies its entries on construction and returns a
// copy of them, rejects a zero entry and two entries with one effect id, holds at most 64 entries (64 accepted, 65 a range error), and the zero ledger is empty.
// Path: unhappy (boundary).
func TestNewHitPointMaximumIncreases(t *testing.T) {
	t.Parallel()
	var empty combat.HitPointMaximumIncreases
	if empty.Len() != 0 || len(empty.Entries()) != 0 || empty.Total() != 0 || !empty.Equal(rrLedger(t)) {
		t.Errorf("the zero ledger: Len %d, entries %v, Total %d, want empty", empty.Len(), empty.Entries(), empty.Total())
	}
	src := []combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 5), rrIncrease(t, "b", "heroes_feast", 7)}
	l := rrLedger(t, src...)
	src[0] = combat.HitPointMaximumIncrease{}
	got := l.Entries()
	got[1] = combat.HitPointMaximumIncrease{}
	if l.Len() != 2 || l.Total() != 12 || l.Entries()[0].ID() != "a" || l.Entries()[1].ID() != "b" {
		t.Errorf("the ledger changed with its source or a returned slice: %v", l.Entries())
	}
	if _, err := combat.NewHitPointMaximumIncreases([]combat.HitPointMaximumIncrease{{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero entry: %v, want an invalid input error", err)
	}
	if _, err := combat.NewHitPointMaximumIncreases([]combat.HitPointMaximumIncrease{rrIncrease(t, "a", "aid", 5), rrIncrease(t, "a", "heroes_feast", 7)}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a duplicate effect id: %v, want an invalid input error", err)
	}
	many := func(n int) []combat.HitPointMaximumIncrease {
		var out []combat.HitPointMaximumIncrease
		for i := range n {
			out = append(out, rrIncrease(t, fmt.Sprintf("e%d", i), "aid", 5))
		}
		return out
	}
	if l64, err := combat.NewHitPointMaximumIncreases(many(64)); err != nil || l64.Len() != 64 {
		t.Errorf("64 entries: Len %d, %v, want accepted", l64.Len(), err)
	}
	var re *combat.RestorationRangeError
	if _, err := combat.NewHitPointMaximumIncreases(many(65)); !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &re) || re.Name != "increase_entries" || re.Value != 65 {
		t.Errorf("65 entries: %v, want a range error for increase_entries = 65", err)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-015, FR-045 - the Equal of a ledger compares its entries in order.
// Path: unhappy (boundary).
func TestHitPointMaximumIncreasesEqual(t *testing.T) {
	t.Parallel()
	a, b := rrIncrease(t, "a", "aid", 5), rrIncrease(t, "b", "heroes_feast", 7)
	if !rrLedger(t, a, b).Equal(rrLedger(t, a, b)) || rrLedger(t, a, b).Equal(rrLedger(t, b, a)) || rrLedger(t, a).Equal(rrLedger(t, a, b)) {
		t.Error("Equal does not compare the entries in order")
	}
	if !reflect.DeepEqual(rrLedger(t, a).Entries(), []combat.HitPointMaximumIncrease{a}) || !slices.Equal(rrLedger(t, a).Entries(), []combat.HitPointMaximumIncrease{a}) {
		t.Error("Entries does not return the entries")
	}
}
