package combat_test

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rrIncreased applies an increase and returns the applied result or fails the test.
func rrIncreased(tb testing.TB, st combat.HitPointStatus, l combat.HitPointMaximumIncreases, inc combat.HitPointMaximumIncrease) combat.IncreaseMaximumApplied {
	tb.Helper()
	res, err := combat.IncreaseMaximum(gid("hero"), st, l, inc)
	if err != nil {
		tb.Fatalf("IncreaseMaximum: %v", err)
	}
	a, ok := res.(combat.IncreaseMaximumApplied)
	if !ok {
		tb.Fatalf("IncreaseMaximum = %T, want IncreaseMaximumApplied", res)
	}
	return a
}

// Requirement: per case (table 4.5, maximum 20 and current 15 unless the row says otherwise; Source for every row: SRD 5.2.1, "Aid" p. 107 unless given)
// Path: per case.
func TestIncreaseMaximum(t *testing.T) {
	t.Parallel()
	type step struct {
		id, spell string
		amount    int
	}
	cases := []struct {
		name, requirement, path string
		start                   combat.HitPointStatus
		steps                   []step
		maximum, current, temp  int
		events                  []string
	}{
		{"Aid at level 2", "FR-023, FR-026; table 4.5", "happy", pcStatus(t, 15, 20, 0), []step{{"a", "aid", 5}}, 25, 20, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"Aid at level 3", "FR-026; table 4.5", "happy", pcStatus(t, 15, 20, 0), []step{{"a", "aid", 10}}, 30, 25, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"Heroes' Feast roll 7 (p. 140)", "FR-024, FR-026; table 4.5", "happy", pcStatus(t, 15, 20, 0), []step{{"a", "heroes_feast", 7}}, 27, 22, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"at full Hit Points", "FR-026; table 4.5", "unhappy (boundary)", pcStatus(t, 20, 20, 0), []step{{"a", "aid", 5}}, 25, 25, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"two Aids", "FR-025; table 4.5", "unhappy (rule interaction)", pcStatus(t, 15, 20, 0), []step{{"a", "aid", 5}, {"b", "aid", 10}}, 30, 25, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"weaker second Aid", "FR-025, FR-026; table 4.5", "unhappy (rule interaction)", pcStatus(t, 15, 20, 0), []step{{"a", "aid", 10}, {"b", "aid", 5}}, 30, 25, 0, nil},
		{"equal second Aid", "FR-025, FR-026; table 4.5", "unhappy (boundary)", pcStatus(t, 15, 20, 0), []step{{"a", "aid", 5}, {"b", "aid", 5}}, 25, 20, 0, nil},
		{"Aid and Heroes' Feast", "FR-025; table 4.5", "unhappy (rule interaction)", pcStatus(t, 15, 20, 0), []step{{"a", "aid", 5}, {"b", "heroes_feast", 7}}, 32, 27, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"Dying creature", "FR-027; table 4.5", "unhappy (state error)", dyingStatus(t, 20, 1, 2), []step{{"a", "aid", 5}}, 25, 5, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored", "recovered"}},
		{"Stable creature", "FR-027; table 4.5", "unhappy (state error)", status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()),
			[]step{{"a", "aid", 5}}, 25, 5, 0, []string{"hit_point_maximum_increased", "hit_points_restored", "recovered"}},
		{"Temporary Hit Points kept", "FR-026; table 4.5", "unhappy (rule interaction)", pcStatus(t, 15, 20, 4), []step{{"a", "aid", 5}}, 25, 20, 4,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
		{"the largest maximum stops at the largest int", "FR-026 (overflow)", "unhappy (boundary)", pcStatus(t, 10, math.MaxInt, 0), []step{{"a", "aid", 5}}, math.MaxInt, 15, 0,
			[]string{"hit_point_maximum_increased", "hit_points_restored"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			st, ledger := c.start, combat.HitPointMaximumIncreases{}
			var last combat.IncreaseMaximumApplied
			for _, s := range c.steps {
				last = rrIncreased(t, st, ledger, rrIncrease(t, s.id, s.spell, s.amount))
				st, ledger = last.Status(), last.Ledger()
			}
			hp := st.HitPoints()
			if _, alive := st.LifeState().(combat.Alive); !alive || hp.Maximum() != c.maximum || hp.Current() != c.current || hp.Temporary() != c.temp {
				t.Errorf("status = %+v / %T, want Alive at %d of %d with %d temporary [%s; %s]", hp, st.LifeState(), c.current, c.maximum, c.temp, c.requirement, c.path)
			}
			if got := rrKinds(last.Events()); !slices.Equal(got, append([]string{}, c.events...)) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, c.requirement)
			}
			if ledger.Len() != len(c.steps) {
				t.Errorf("ledger holds %d entries, want %d (every cast joins, even one that changes nothing)", ledger.Len(), len(c.steps))
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-030 - HitPointMaximumIncreased carries the new maximum and the increase, with the creature as its only
// combatant; the events of Restore follow it. An Aid of 5 on 15 of 20 gives maximum 25 and increase 5, and then a restored 5.
// Path: happy.
func TestIncreaseMaximumEvents(t *testing.T) {
	t.Parallel()
	a := rrIncreased(t, pcStatus(t, 15, 20, 0), combat.HitPointMaximumIncreases{}, rrIncrease(t, "a", "aid", 5))
	ev := a.Events()
	inc, ok := ev[0].(combat.HitPointMaximumIncreased)
	if !ok || inc.Maximum() != 25 || inc.Increase() != 5 {
		t.Fatalf("first event = %#v, want HitPointMaximumIncreased(25, 5)", ev[0])
	}
	if restored, ok := ev[1].(combat.HitPointsRestored); !ok || restored.Regained() != 5 {
		t.Errorf("second event = %#v, want HitPointsRestored(5)", ev[1])
	}
	ev[0] = nil
	if a.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}

// Requirement: per case (table 4.5 state errors; FR-029 precedence: Dead is checked first, then the duplicate precondition, then the cap)
// Path: per case.
func TestIncreaseMaximumRefusals(t *testing.T) {
	t.Parallel()
	entry := rrIncrease(t, "a", "aid", 5)
	held := rrLedger(t, entry)
	var full []combat.HitPointMaximumIncrease
	for i := range 64 {
		full = append(full, rrIncrease(t, fmt.Sprintf("e%d", i), "aid", 5))
	}
	almost := rrLedger(t, full[:63]...)
	cases := []struct {
		name, requirement, path string
		status                  combat.HitPointStatus
		ledger                  combat.HitPointMaximumIncreases
		inc                     combat.HitPointMaximumIncrease
		want                    string
	}{
		{"Dead creature", "FR-029; table 4.5", "unhappy (state error)", deadStatus(t, combat.DeathSavingThrowFailures), combat.HitPointMaximumIncreases{}, entry, "rejected"},
		{"Dead and duplicate id", "FR-029 precedence; table 4.5", "unhappy (state error)", deadStatus(t, combat.DeathSavingThrowFailures), held, entry, "rejected"},
		{"duplicate id on an Alive creature", "FR-029; table 4.5", "unhappy (invalid input)", pcStatus(t, 15, 20, 0), held, rrIncrease(t, "a", "heroes_feast", 9), "duplicate"},
		{"a ledger of 63 takes a 64th", "plan P10 (the cap)", "unhappy (boundary)", pcStatus(t, 15, 20, 0), almost, full[63], "applied"},
		{"a ledger of 64 refuses a 65th", "plan P10 (cap plus one)", "unhappy (boundary)", pcStatus(t, 15, 20, 0), rrLedger(t, full...), rrIncrease(t, "extra", "aid", 5), "limit_exceeded"},
		{"Dead and a full ledger", "FR-029 precedence", "unhappy (state error)", deadStatus(t, combat.DeathSavingThrowFailures), rrLedger(t, full...), rrIncrease(t, "extra", "aid", 5), "rejected"},
		{"a duplicate id in a full ledger", "FR-029 precedence", "unhappy (state error)", pcStatus(t, 15, 20, 0), rrLedger(t, full...), full[3], "duplicate"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := combat.IncreaseMaximum(gid("hero"), c.status, c.ledger, c.inc)
			if err != nil {
				t.Fatalf("IncreaseMaximum error: %v", err)
			}
			var got string
			switch r := res.(type) {
			case combat.IncreaseMaximumApplied:
				got = "applied"
			case combat.IncreaseMaximumRejected:
				got = "rejected"
				if r.Rejection != combat.CreatureIsDead {
					t.Errorf("Rejection = %v, want CreatureIsDead", r.Rejection)
				}
			case combat.IncreaseMaximumDuplicate:
				got = "duplicate"
			case combat.IncreaseMaximumLimitExceeded:
				got = "limit_exceeded"
			}
			if got != c.want {
				t.Errorf("IncreaseMaximum = %T, want %s [%s; %s]", res, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-047, FR-045 - IncreaseMaximum returns an error matching srderr.ErrInvalidInput for a zero subject,
// the zero status and the zero increase, and changes none of its inputs, whatever it answers.
// Path: unhappy (invalid input).
func TestIncreaseMaximumInvalidAndImmutable(t *testing.T) {
	t.Parallel()
	st, entry := pcStatus(t, 15, 20, 3), rrIncrease(t, "a", "aid", 5)
	for name, call := range map[string]func() (combat.IncreaseHitPointMaximumResult, error){
		"zero subject": func() (combat.IncreaseHitPointMaximumResult, error) {
			return combat.IncreaseMaximum("", st, combat.HitPointMaximumIncreases{}, entry)
		},
		"zero status": func() (combat.IncreaseHitPointMaximumResult, error) {
			return combat.IncreaseMaximum(gid("hero"), combat.HitPointStatus{}, combat.HitPointMaximumIncreases{}, entry)
		},
		"zero increase": func() (combat.IncreaseHitPointMaximumResult, error) {
			return combat.IncreaseMaximum(gid("hero"), st, combat.HitPointMaximumIncreases{}, combat.HitPointMaximumIncrease{})
		},
	} {
		if res, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: IncreaseMaximum = %v, %v, want a nil result and an invalid input error", name, res, err)
		}
	}
	ledger := rrLedger(t, rrIncrease(t, "old", "heroes_feast", 3))
	before, beforeLedger := st, ledger
	first, _ := combat.IncreaseMaximum(gid("hero"), st, ledger, entry)
	second, _ := combat.IncreaseMaximum(gid("hero"), st, ledger, entry)
	if !reflect.DeepEqual(st, before) || !ledger.Equal(beforeLedger) || !reflect.DeepEqual(first, second) {
		t.Error("IncreaseMaximum changed an input or gave unequal results for equal inputs")
	}
}
