package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-002, FR-003, FR-122 - a record takes counts of 0 or more whose level counts sum to at most 6, exposes them, and reports each rejection as a *HazardRecordError matching ErrInvalidInput.
// Path: per case.
func TestNewHazardRecord(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		d, m, s, days           int
		want                    combat.HazardRecordRejection // 0: accepted
	}{
		{"all zero", "FR-002", "happy", 0, 0, 0, 0, 0},
		{"each count set", "FR-002", "happy", 1, 2, 3, 40, 0},
		{"sum exactly six", "FR-002", "unhappy (boundary)", 2, 2, 2, 0, 0},
		{"sum seven", "FR-003", "unhappy (boundary)", 2, 2, 3, 0, combat.HazardRecordLevelSumAboveSix},
		{"negative dehydration", "FR-003", "unhappy (invalid input)", -1, 0, 0, 0, combat.HazardRecordNegativeCount},
		{"negative malnutrition", "FR-003", "unhappy (invalid input)", 0, -1, 0, 0, combat.HazardRecordNegativeCount},
		{"negative suffocation", "FR-003", "unhappy (invalid input)", 0, 0, -1, 0, combat.HazardRecordNegativeCount},
		{"negative days without food", "FR-003", "unhappy (invalid input)", 0, 0, 0, -1, combat.HazardRecordNegativeCount},
		{"huge days are fine", "FR-002", "unhappy (boundary)", 0, 0, 0, math.MaxInt, 0},
		{"huge level count does not overflow the sum", "FR-122", "unhappy (boundary)", math.MaxInt, math.MaxInt, 0, 0, combat.HazardRecordLevelSumAboveSix},
		{"minimum int", "FR-122", "unhappy (invalid input)", math.MinInt, 0, 0, 0, combat.HazardRecordNegativeCount},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewHazardRecord(c.d, c.m, c.s, c.days)
			if c.want == 0 {
				if err != nil || r.DehydrationLevels() != c.d || r.MalnutritionLevels() != c.m || r.SuffocationLevels() != c.s ||
					r.DaysWithoutFood() != c.days || r.ContagionLevels() != 0 {
					t.Errorf("= %+v, %v, want the counts back [%s]", r, err, c.requirement)
				}
				return
			}
			var he *combat.HazardRecordError
			if !errors.As(err, &he) || he.Reason != c.want || !errors.Is(err, srderr.ErrInvalidInput) ||
				!errors.Is(err, &combat.HazardRecordError{Reason: c.want}) || errors.Is(err, &combat.HazardRecordError{Reason: 99}) {
				t.Errorf("err = %v, want a HazardRecordError with %v [%s]", err, c.want, c.requirement)
			}
			if r != (combat.HazardRecord{}) {
				t.Errorf("record = %+v with an error, want the zero record", r)
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-122 - MustHazardRecord panics with the NewHazardRecord error iff it returns one.
// Path: unhappy (invalid input).
func TestMustHazardRecord(t *testing.T) {
	t.Parallel()
	if r := combat.MustHazardRecord(1, 2, 3, 4); r.SuffocationLevels() != 3 {
		t.Errorf("MustHazardRecord = %+v", r)
	}
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an ErrInvalidInput error", err)
		}
	}()
	combat.MustHazardRecord(-1, 0, 0, 0)
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-002, FR-004 - the empty record is all zeros, and records are equal under == exactly when all five values are.
// Path: happy.
func TestHazardRecordEquality(t *testing.T) {
	t.Parallel()
	if combat.EmptyHazardRecord() != (combat.HazardRecord{}) || combat.EmptyHazardRecord() != envRecord(t, 0, 0, 0, 0, 0) {
		t.Error("the empty record is not the zero record")
	}
	base := envRecord(t, 1, 1, 1, 1, 1)
	for name, other := range map[string]combat.HazardRecord{
		"dehydration": envRecord(t, 2, 1, 1, 1, 1), "malnutrition": envRecord(t, 1, 2, 1, 1, 1), "suffocation": envRecord(t, 1, 1, 2, 1, 1),
		"days": envRecord(t, 1, 1, 1, 2, 1), "contagion": envRecord(t, 1, 1, 1, 1, 2),
	} {
		if base == other {
			t.Errorf("records differing in %s are equal", name)
		}
	}
	if base != envRecord(t, 1, 1, 1, 1, 1) {
		t.Error("equal records are unequal")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-115 - WithContagionLevels returns a new record, counts in the 6-level sum, and rejects a negative count or a sum above 6 without changing the receiver.
// Path: per case.
func TestWithContagionLevels(t *testing.T) {
	t.Parallel()
	base := envRecord(t, 2, 2, 1, 0, 0)
	cases := []struct {
		name, requirement, path string
		count                   int
		want                    combat.HazardRecordRejection
	}{
		{"one level", "FR-115", "happy", 1, 0},
		{"to the sum of six", "FR-115", "unhappy (boundary)", 1, 0},
		{"sum seven", "FR-115", "unhappy (boundary)", 2, combat.HazardRecordLevelSumAboveSix},
		{"negative", "FR-115", "unhappy (invalid input)", -1, combat.HazardRecordNegativeCount},
		{"huge", "FR-115", "unhappy (boundary)", math.MaxInt, combat.HazardRecordLevelSumAboveSix},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := base.WithContagionLevels(c.count)
			var he *combat.HazardRecordError
			switch {
			case c.want == 0 && (err != nil || r.ContagionLevels() != c.count || r.DehydrationLevels() != 2):
				t.Errorf("= %+v, %v, want the count set and the rest kept [%s]", r, err, c.requirement)
			case c.want != 0 && (!errors.As(err, &he) || he.Reason != c.want):
				t.Errorf("err = %v, want %v [%s]", err, c.want, c.requirement)
			}
			if base.ContagionLevels() != 0 {
				t.Error("the receiver changed")
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-060, FR-115 - RemovableLevels is max(0, level - dehydration - malnutrition - contagion), 0 for no Exhaustion, and Suffocation levels are not locked (table 4.8a).
// Path: per case.
func TestRemovableLevels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level                   int
		rec                     [4]int // dehydration, malnutrition, suffocation, contagion
		want                    int
	}{
		{"nothing locked", "FR-060", "happy", 3, [4]int{0, 0, 0, 0}, 3},
		{"partly locked", "FR-060", "unhappy (boundary)", 3, [4]int{2, 0, 0, 0}, 1},
		{"all locked", "FR-060", "unhappy (boundary)", 3, [4]int{3, 0, 0, 0}, 0},
		{"locked by both", "FR-060", "unhappy (rule interaction)", 3, [4]int{2, 1, 0, 0}, 0},
		{"locks above the level", "FR-060", "unhappy (boundary)", 1, [4]int{1, 1, 0, 0}, 0},
		{"suffocation is not locked", "FR-060", "unhappy (rule interaction)", 2, [4]int{0, 0, 2, 0}, 2},
		{"contagion locks", "FR-115", "unhappy (rule interaction)", 3, [4]int{1, 0, 0, 1}, 1},
		{"no exhaustion", "FR-060", "unhappy (empty and minimal)", 0, [4]int{0, 0, 0, 0}, 0},
		{"negative level counts as none", "FR-060", "unhappy (invalid input)", -4, [4]int{0, 0, 0, 0}, 0},
		{"from level six", "FR-060", "unhappy (boundary)", 6, [4]int{1, 1, 2, 0}, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := envRecord(t, c.rec[0], c.rec[1], c.rec[2], 0, c.rec[3])
			if got := r.RemovableLevels(c.level); got != c.want {
				t.Errorf("RemovableLevels(%d) = %d, want %d [%s]", c.level, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-122 - NewHazardRecord never panics and accepts exactly the valid counts.
// Path: unhappy (invalid input).
func FuzzNewHazardRecord(f *testing.F) {
	for _, s := range [][4]int{{0, 0, 0, 0}, {2, 2, 2, 5}, {-1, 0, 0, 0}, {math.MaxInt, math.MaxInt, 0, 0}, {math.MinInt, math.MinInt, 0, math.MinInt}, {3, 3, 1, 0}} {
		f.Add(s[0], s[1], s[2], s[3])
	}
	f.Fuzz(func(t *testing.T, d, m, s, days int) {
		r, err := combat.NewHazardRecord(d, m, s, days)
		valid := d >= 0 && m >= 0 && s >= 0 && days >= 0 && d <= 6 && m <= 6-d && s <= 6-d-m
		if (err == nil) != valid {
			t.Fatalf("NewHazardRecord(%d,%d,%d,%d) error = %v, want error %v", d, m, s, days, err, !valid)
		}
		if err != nil && !errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("error %v does not match ErrInvalidInput", err)
		}
		if err == nil && (r.DehydrationLevels() != d || r.DaysWithoutFood() != days) {
			t.Fatalf("record %+v does not hold its counts", r)
		}
	})
}
