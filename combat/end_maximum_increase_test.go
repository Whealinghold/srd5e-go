package combat_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rrEnded ends an increase and returns the applied result or fails the test.
func rrEnded(tb testing.TB, st combat.HitPointStatus, l combat.HitPointMaximumIncreases, id string) combat.EndMaximumIncreaseApplied {
	tb.Helper()
	res, err := combat.EndMaximumIncrease(gid("hero"), st, l, meEID(id))
	if err != nil {
		tb.Fatalf("EndMaximumIncrease: %v", err)
	}
	a, ok := res.(combat.EndMaximumIncreaseApplied)
	if !ok {
		tb.Fatalf("EndMaximumIncrease = %T, want EndMaximumIncreaseApplied", res)
	}
	return a
}

// Requirement: per case (table 4.5, ending an increase; Source for every row: SRD 5.2.1, "Hit Points" p. 183 and "Hit Point Maximum of 0" p. 17)
// Path: per case.
func TestEndMaximumIncrease(t *testing.T) {
	t.Parallel()
	aid5, aid10 := rrIncrease(t, "a5", "aid", 5), rrIncrease(t, "a10", "aid", 10)
	cases := []struct {
		name, requirement, path string
		status                  combat.HitPointStatus
		ledger                  combat.HitPointMaximumIncreases
		end                     string
		maximum, current, lost  int
		events                  []string
		left                    int
	}{
		{"end Aid, current inside", "FR-028; table 4.5", "happy", pcStatus(t, 15, 25, 0), rrLedger(t, aid5), "a5", 20, 15, 0,
			[]string{"hit_point_maximum_increase_ended", "hit_point_maximum_reduced"}, 0},
		{"end Aid, current above", "FR-028; table 4.5", "unhappy (boundary)", pcStatus(t, 25, 25, 0), rrLedger(t, aid5), "a5", 20, 20, 5,
			[]string{"hit_point_maximum_increase_ended", "hit_point_maximum_reduced"}, 0},
		{"end the stronger of one spell", "FR-025, FR-028; table 4.5", "unhappy (rule interaction)", pcStatus(t, 30, 30, 0), rrLedger(t, aid10, aid5), "a10", 25, 25, 5,
			[]string{"hit_point_maximum_increase_ended", "hit_point_maximum_reduced"}, 1},
		{"end the weaker", "FR-025, FR-028; table 4.5", "unhappy (rule interaction)", pcStatus(t, 30, 30, 0), rrLedger(t, aid10, aid5), "a5", 30, 30, 0, nil, 1},
		{"maximum drained meanwhile, Q-9", "FR-028; table 4.5", "unhappy (boundary)", pcStatus(t, 4, 4, 0), rrLedger(t, aid5), "a5", 0, 0, 4,
			[]string{"hit_point_maximum_increase_ended", "hit_point_maximum_reduced", "died"}, 0},
		{"Temporary Hit Points are not lost", "FR-028", "unhappy (rule interaction)", pcStatus(t, 25, 25, 6), rrLedger(t, aid5), "a5", 20, 20, 5,
			[]string{"hit_point_maximum_increase_ended", "hit_point_maximum_reduced"}, 0},
		{"a Dying creature stays Dying", "FR-028", "unhappy (state error)", dyingStatus(t, 25, 1, 1), rrLedger(t, aid5), "a5", 20, 0, 0,
			[]string{"hit_point_maximum_increase_ended", "hit_point_maximum_reduced"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			a := rrEnded(t, c.status, c.ledger, c.end)
			hp := a.Status().HitPoints()
			if hp.Maximum() != c.maximum || hp.Current() != c.current || hp.Temporary() != c.status.HitPoints().Temporary() {
				t.Errorf("status = %+v, want %d of %d [%s; %s]", hp, c.current, c.maximum, c.requirement, c.path)
			}
			if got := rrKinds(a.Events()); !slices.Equal(got, append([]string{}, c.events...)) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, c.requirement)
			}
			if a.Ledger().Len() != c.left {
				t.Errorf("ledger holds %d entries, want %d", a.Ledger().Len(), c.left)
			}
			if len(c.events) > 0 {
				ended := a.Events()[0].(combat.HitPointMaximumIncreaseEnded)
				if ended.Maximum() != c.maximum || ended.HitPointsLost() != c.lost || ended.Drop() != 5 {
					t.Errorf("HitPointMaximumIncreaseEnded = max %d drop %d lost %d, want max %d drop %d lost %d", ended.Maximum(), ended.Drop(), ended.HitPointsLost(),
						c.maximum, 5, c.lost)
				}
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-028 - the death of a creature whose maximum reaches 0 is Dead with cause HitPointMaximumZero (SRD 5.2.1,
// "Hit Point Maximum of 0", p. 17); ending an increase is not damage, so no damage event and no Death Saving Throw failure.
// Path: unhappy (boundary).
func TestEndMaximumIncreaseToZeroIsNotDamage(t *testing.T) {
	t.Parallel()
	a := rrEnded(t, pcStatus(t, 4, 4, 0), rrLedger(t, rrIncrease(t, "a", "aid", 5)), "a")
	if d, ok := a.Status().LifeState().(combat.Dead); !ok || d.Cause() != combat.HitPointMaximumZero {
		t.Errorf("life state = %#v, want Dead (HitPointMaximumZero)", a.Status().LifeState())
	}
	for _, e := range a.Events() {
		if e.Kind() == "damage_taken" || e.Kind() == "death_saving_throw_made" {
			t.Errorf("event %s: ending an increase is not damage", e.Kind())
		}
	}
	// A full maximum of the largest int minus nothing: a drop that exceeds the maximum stops at 0, not below.
	big := rrEnded(t, pcStatus(t, 3, 3, 0), rrLedger(t, rrIncrease(t, "a", "aid", math.MaxInt)), "a")
	if big.Status().HitPoints().Maximum() != 0 {
		t.Errorf("a drop past the maximum left %d, want 0", big.Status().HitPoints().Maximum())
	}
}

// Requirement: per case (FR-029, table 4.5 state errors: an id not in the ledger is NotPresent, a Dead creature is Rejected with the ledger unchanged; Dead is checked first)
// Path: per case.
func TestEndMaximumIncreaseRefusals(t *testing.T) {
	t.Parallel()
	held := rrLedger(t, rrIncrease(t, "a", "aid", 5))
	st := pcStatus(t, 15, 25, 0)
	if res, err := combat.EndMaximumIncrease(gid("hero"), st, held, meEID("nope")); err != nil || res != (combat.EndMaximumIncreaseNotPresent{}) {
		t.Errorf("an unknown id: %v, %v, want EndMaximumIncreaseNotPresent", res, err)
	}
	if res, err := combat.EndMaximumIncrease(gid("hero"), st, combat.HitPointMaximumIncreases{}, meEID("a")); err != nil || res != (combat.EndMaximumIncreaseNotPresent{}) {
		t.Errorf("the zero ledger: %v, %v, want EndMaximumIncreaseNotPresent", res, err)
	}
	for _, id := range []string{"a", "nope"} {
		res, err := combat.EndMaximumIncrease(gid("hero"), deadStatus(t, combat.MassiveDamage), held, meEID(id))
		if r, ok := res.(combat.EndMaximumIncreaseRejected); err != nil || !ok || r.Rejection != combat.CreatureIsDead {
			t.Errorf("Dead creature, id %q: %v, %v, want EndMaximumIncreaseRejected (CreatureIsDead)", id, res, err)
		}
	}
	for name, call := range map[string]func() (combat.EndHitPointMaximumIncreaseResult, error){
		"zero subject": func() (combat.EndHitPointMaximumIncreaseResult, error) {
			return combat.EndMaximumIncrease("", st, held, meEID("a"))
		},
		"zero status": func() (combat.EndHitPointMaximumIncreaseResult, error) {
			return combat.EndMaximumIncrease(gid("hero"), combat.HitPointStatus{}, held, meEID("a"))
		},
	} {
		if res, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: %v, %v, want a nil result and an invalid input error", name, res, err)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-045 - EndMaximumIncrease changes none of its inputs, gives equal results for equal inputs and returns a copy of its events.
// Path: unhappy (immutability).
func TestEndMaximumIncreaseImmutable(t *testing.T) {
	t.Parallel()
	st, ledger := pcStatus(t, 25, 25, 2), rrLedger(t, rrIncrease(t, "a", "aid", 5))
	before, beforeLedger := st, ledger
	first, second := rrEnded(t, st, ledger, "a"), rrEnded(t, st, ledger, "a")
	if !reflect.DeepEqual(st, before) || !ledger.Equal(beforeLedger) || !reflect.DeepEqual(first, second) || ledger.Len() != 1 {
		t.Error("EndMaximumIncrease changed an input or gave unequal results for equal inputs")
	}
	ev := first.Events()
	ev[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
}
