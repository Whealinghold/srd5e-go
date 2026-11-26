package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestApplyDeathSave(t *testing.T) {
	t.Parallel()
	const p17, p18 = `SRD 5.2.1, "Death Saving Throws", p. 17`, `SRD 5.2.1, "Rolling a 1 or 20", p. 18`
	made, died, stableEv, restored, recovered := "death_saving_throw_made", "died", "became_stable", "hit_points_restored", "recovered"
	cases := []struct {
		name, requirement, path string
		before                  [2]int
		natural, total          int
		after                   combat.LifeState
		current                 int
		outcome                 combat.DeathSavingThrowOutcome
		events                  []string
	}{
		{"Exhaustion turns a success into a failure", `table 4.7a; "Exhaustion", p. 181`, "unhappy (rule interaction)", [2]int{0, 0}, 11, 7, dying(0, 1), 0, combat.DeathSavingThrowFailure, []string{made}},
		{"total of 10 exactly after reduction", `FR-054`, "unhappy (boundary)", [2]int{0, 0}, 14, 10, dying(1, 0), 0, combat.DeathSavingThrowSuccess, []string{made}},
		{"natural 20 ignores a low total", `FR-056`, "unhappy (boundary)", [2]int{0, 2}, 20, 4, combat.Alive{}, 1, combat.DeathSavingThrowRegainedHitPoint, []string{made, restored, recovered}},
		{"natural 1 with a high total", `FR-055`, "unhappy (boundary)", [2]int{0, 0}, 1, 12, dying(0, 2), 0, combat.DeathSavingThrowTwoFailures, []string{made}},
		{"minimum failure range", p17, "unhappy (boundary)", [2]int{0, 0}, 2, 2, dying(0, 1), 0, combat.DeathSavingThrowFailure, []string{made}},
		{"highest failure", p17, "unhappy (boundary)", [2]int{0, 0}, 9, 9, dying(0, 1), 0, combat.DeathSavingThrowFailure, []string{made}},
		{"lowest success", p17, "unhappy (boundary)", [2]int{0, 0}, 10, 10, dying(1, 0), 0, combat.DeathSavingThrowSuccess, []string{made}},
		{"high success", p17, "happy", [2]int{1, 1}, 19, 19, dying(2, 1), 0, combat.DeathSavingThrowSuccess, []string{made}},
		{"natural 1", p18, "unhappy (boundary)", [2]int{0, 0}, 1, 1, dying(0, 2), 0, combat.DeathSavingThrowTwoFailures, []string{made}},
		{"natural 1 at one failure", p18, "unhappy (boundary)", [2]int{0, 1}, 1, 1, dead(combat.DeathSavingThrowFailures), 0, combat.DeathSavingThrowTwoFailures, []string{made, died}},
		{"natural 1 at two failures", p18, "unhappy (boundary)", [2]int{0, 2}, 1, 1, dead(combat.DeathSavingThrowFailures), 0, combat.DeathSavingThrowTwoFailures, []string{made, died}},
		{"natural 20", p18, "unhappy (boundary)", [2]int{1, 2}, 20, 20, combat.Alive{}, 1, combat.DeathSavingThrowRegainedHitPoint, []string{made, restored, recovered}},
		{"natural 20 records no success", `FR-056`, "unhappy (boundary)", [2]int{2, 0}, 20, 20, combat.Alive{}, 1, combat.DeathSavingThrowRegainedHitPoint, []string{made, restored, recovered}},
		{"third success", `FR-057; "Three Successes/Failures", p. 17`, "unhappy (boundary)", [2]int{2, 2}, 10, 10, combat.Stable{}, 0, combat.DeathSavingThrowSuccess, []string{made, stableEv}},
		{"third failure", `FR-058`, "unhappy (boundary)", [2]int{2, 2}, 9, 9, dead(combat.DeathSavingThrowFailures), 0, combat.DeathSavingThrowFailure, []string{made, died}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			st := dyingStatus(t, 20, c.before[0], c.before[1])
			tr, err := st.ApplyDeathSave(subject(), c.natural, c.total)
			a := applied(t, tr, err)
			if a.Status().LifeState() != c.after || a.Status().HitPoints().Current() != c.current {
				t.Errorf("status = %v with %d, want %v with %d [%s]", a.Status().LifeState(), a.Status().HitPoints().Current(), c.after, c.current, c.requirement)
			}
			if got := kinds(a.Events()); !slices.Equal(got, c.events) {
				t.Errorf("events = %v, want %v", got, c.events)
			}
			if m, _ := firstEvent[combat.DeathSavingThrowMade](a.Events()); m.Outcome() != c.outcome {
				t.Errorf("outcome = %v, want %v", m.Outcome(), c.outcome)
			}
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md table 4.7a "interleaved", FR-059 - successes and failures
// accumulate without being consecutive: 12, 4, 15, 3, 18 make the creature Stable after the fifth roll.
// Path: unhappy (rule interaction).
func TestDeathSavesInterleaved(t *testing.T) {
	t.Parallel()
	st := dyingStatus(t, 20, 0, 0)
	for i, n := range []int{12, 4, 15, 3, 18} {
		tr, err := st.ApplyDeathSave(subject(), n, n)
		x := applied(t, tr, err)
		st = x.Status()
		if i < 4 && !st.MakesDeathSavingThrows() {
			t.Fatalf("not Dying after roll %d", i+1)
		}
	}
	if st.LifeState() != (combat.Stable{}) {
		t.Errorf("after five rolls = %v, want Stable", st.LifeState())
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-053, FR-061, FR-068, FR-073, FR-075 - a Death Saving
// Throw, stabilizing and stable recovery are accepted only in the right state; otherwise a refusal with the
// state unchanged; a natural outside 1 to 20 is invalid input.
// Path: per case.
func TestDeathOperationsRefusals(t *testing.T) {
	t.Parallel()
	alive := pcStatus(t, 10, 20, 0)
	stable := status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities())
	deadSt := deadStatus(t, combat.MassiveDamage)
	dyingSt := dyingStatus(t, 20, 0, 0)
	is := func(r combat.HitPointRejection) combat.HitPointTransition {
		return combat.HitPointTransitionRefused{Rejection: r}
	}
	cases := []struct {
		name string
		call func() (combat.HitPointTransition, error)
		want combat.HitPointTransition
	}{
		{"save when alive", func() (combat.HitPointTransition, error) { return alive.ApplyDeathSave(subject(), 12, 12) }, is(combat.NotDying)},
		{"save when stable", func() (combat.HitPointTransition, error) { return stable.ApplyDeathSave(subject(), 12, 12) }, is(combat.NotDying)},
		{"save when dead", func() (combat.HitPointTransition, error) { return deadSt.ApplyDeathSave(subject(), 12, 12) }, is(combat.CreatureIsDead)},
		{"stabilize alive", func() (combat.HitPointTransition, error) { return alive.Stabilize(subject()) }, is(combat.NotDying)},
		{"stabilize stable", func() (combat.HitPointTransition, error) { return stable.Stabilize(subject()) }, is(combat.NotDying)},
		{"stabilize dead", func() (combat.HitPointTransition, error) { return deadSt.Stabilize(subject()) }, is(combat.CreatureIsDead)},
		{"recover alive", func() (combat.HitPointTransition, error) { return alive.RecoverStable(subject()) }, is(combat.NotStable)},
		{"recover dying", func() (combat.HitPointTransition, error) { return dyingSt.RecoverStable(subject()) }, is(combat.NotStable)},
		{"recover dead", func() (combat.HitPointTransition, error) { return deadSt.RecoverStable(subject()) }, is(combat.CreatureIsDead)},
	}
	for _, c := range cases {
		if got, err := c.call(); err != nil || got != c.want {
			t.Errorf("%s = %v, %v, want %v", c.name, got, err, c.want)
		}
	}
	for _, n := range []int{0, 21, -1} {
		if tr, err := dyingSt.ApplyDeathSave(subject(), n, 10); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("natural %d = %v, %v, want ErrInvalidInput (FR-061)", n, tr, err)
		}
	}
	for name, call := range map[string]func() (combat.HitPointTransition, error){
		"save zero subject":      func() (combat.HitPointTransition, error) { return dyingSt.ApplyDeathSave("", 12, 12) },
		"stabilize zero status":  func() (combat.HitPointTransition, error) { return combat.HitPointStatus{}.Stabilize(subject()) },
		"recover zero subject":   func() (combat.HitPointTransition, error) { return stable.RecoverStable("") },
		"stabilize zero subject": func() (combat.HitPointTransition, error) { return dyingSt.Stabilize("") },
		"save zero status": func() (combat.HitPointTransition, error) {
			return combat.HitPointStatus{}.ApplyDeathSave(subject(), 12, 12)
		},
	} {
		if tr, err := call(); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, %v, want ErrInvalidInput", name, tr, err)
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-067, FR-073, FR-074 - stabilizing a Dying creature makes it
// Stable with the tallies reset; recovering a Stable creature gives 1 Hit Point and Alive, after which healing
// can no longer recover it (SRD 5.2.1, "Stabilizing a Character", p. 18).
// Path: happy.
func TestStabilizeAndRecover(t *testing.T) {
	t.Parallel()
	tr, err := dyingStatus(t, 20, 2, 1).Stabilize(subject())
	a := applied(t, tr, err)
	if a.Status().LifeState() != (combat.Stable{}) || !slices.Equal(kinds(a.Events()), []string{"became_stable"}) || !a.Status().IsUnconscious() {
		t.Errorf("stabilize = %v %v", a.Status().LifeState(), kinds(a.Events()))
	}
	tr, err = a.Status().RecoverStable(subject())
	b := applied(t, tr, err)
	if b.Status().LifeState() != (combat.Alive{}) || b.Status().HitPoints().Current() != 1 || b.HitPointsRegained() != 1 ||
		!slices.Equal(kinds(b.Events()), []string{"hit_points_restored", "recovered"}) {
		t.Errorf("recover = %v %v", b.Status().LifeState(), kinds(b.Events()))
	}
	tr, err = b.Status().RecoverStable(subject())
	if err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.NotStable}) {
		t.Errorf("recover again = %v, %v, want NotStable", tr, err)
	}
}
