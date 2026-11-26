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
func TestRestore(t *testing.T) {
	t.Parallel()
	at := func(n int) combat.HealingCeiling {
		c, err := combat.NewHealingCeilingAt(n)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	healing := `SRD 5.2.1, "Healing", p. 17 (table 4.5a)`
	cases := []struct {
		name, requirement, path string
		start                   combat.HitPointStatus
		amount                  int
		ceiling                 combat.HealingCeiling
		current, temp, regained int
		life                    combat.LifeState
		events                  []string
	}{
		{"worked example", healing, "happy", pcStatus(t, 14, 20, 0), 8, nil, 20, 0, 6, combat.Alive{}, []string{"hit_points_restored"}},
		{"exactly to the maximum", healing, "unhappy (boundary)", pcStatus(t, 14, 20, 0), 6, nil, 20, 0, 6, combat.Alive{}, []string{"hit_points_restored"}},
		{"one short", healing, "unhappy (boundary)", pcStatus(t, 14, 20, 0), 5, nil, 19, 0, 5, combat.Alive{}, []string{"hit_points_restored"}},
		{"already full", `FR-036`, "unhappy (boundary)", pcStatus(t, 20, 20, 0), 3, nil, 20, 0, 0, combat.Alive{}, []string{}},
		{"zero healing at 0 stays Dying", `FR-037; "Falling Unconscious", p. 17`, "unhappy (boundary)", dyingStatus(t, 20, 1, 1), 0, nil, 0, 0, 0, dying(1, 1), []string{}},
		{"one Hit Point at 0", `FR-038, FR-060`, "unhappy (boundary)", dyingStatus(t, 20, 1, 2), 1, nil, 1, 0, 1, combat.Alive{}, []string{"hit_points_restored", "recovered"}},
		{"healing a Stable creature", `FR-074`, "unhappy (rule interaction)", status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), 4, nil, 4, 0, 4, combat.Alive{}, []string{"hit_points_restored", "recovered"}},
		{"temporary Hit Points stay", `FR-032`, "unhappy (rule interaction)", pcStatus(t, 10, 20, 7), 3, nil, 13, 7, 3, combat.Alive{}, []string{"hit_points_restored"}},
		{"ceiling below the maximum", `FR-115`, "unhappy (boundary)", pcStatus(t, 10, 20, 0), 9, at(15), 15, 0, 5, combat.Alive{}, []string{"hit_points_restored"}},
		{"ceiling at the current value", `FR-115`, "unhappy (boundary)", pcStatus(t, 10, 20, 0), 9, at(10), 10, 0, 0, combat.Alive{}, []string{}},
		{"ceiling below the current value never lowers", `FR-115`, "unhappy (boundary)", pcStatus(t, 10, 20, 0), 9, at(4), 10, 0, 0, combat.Alive{}, []string{}},
		{"ceiling above the maximum", `FR-115`, "unhappy (boundary)", pcStatus(t, 10, 20, 0), 99, at(99), 20, 0, 10, combat.Alive{}, []string{"hit_points_restored"}},
		{"ceiling 0 at 0 Hit Points leaves Dying", `FR-115`, "unhappy (boundary)", dyingStatus(t, 20, 0, 1), 5, at(0), 0, 0, 0, dying(0, 1), []string{}},
		{"no ceiling is the default", `FR-115`, "happy", pcStatus(t, 10, 20, 0), 5, combat.NoHealingCeiling{}, 15, 0, 5, combat.Alive{}, []string{"hit_points_restored"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := c.start.Restore(subject(), c.amount, c.ceiling)
			a := applied(t, tr, err)
			hp := a.Status().HitPoints()
			if hp.Current() != c.current || hp.Temporary() != c.temp || a.HitPointsRegained() != c.regained || a.Status().LifeState() != c.life {
				t.Errorf("= current %d temp %d regained %d %v, want %d %d %d %v [%s]", hp.Current(), hp.Temporary(), a.HitPointsRegained(),
					a.Status().LifeState(), c.current, c.temp, c.regained, c.life, c.requirement)
			}
			if got := kinds(a.Events()); !slices.Equal(got, c.events) {
				t.Errorf("events = %v, want %v", got, c.events)
			}
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-037, FR-039, FR-094, FR-115 - negative healing, a zero
// subject or status and an unknown ceiling are invalid input; healing a Dead creature is refused.
// Path: unhappy (invalid input).
func TestRestoreInvalidAndDead(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 10, 20, 0)
	for name, call := range map[string]func() (combat.HitPointTransition, error){
		"negative":     func() (combat.HitPointTransition, error) { return st.Restore(subject(), -1, nil) },
		"zero subject": func() (combat.HitPointTransition, error) { return st.Restore("", 1, nil) },
		"zero status":  func() (combat.HitPointTransition, error) { return combat.HitPointStatus{}.Restore(subject(), 1, nil) },
		"unknown ceiling": func() (combat.HitPointTransition, error) {
			return st.Restore(subject(), 1, unknownCeiling{})
		},
	} {
		tr, err := call()
		if tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, %v, want nil and ErrInvalidInput", name, tr, err)
		}
	}
	tr, err := deadStatus(t, combat.MassiveDamage).Restore(subject(), 5, nil)
	if err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("dead = %v, %v, want Refused(CreatureIsDead)", tr, err)
	}
	if _, err := combat.NewHealingCeilingAt(-1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewHealingCeilingAt(-1) error = %v, want ErrInvalidInput", err)
	}
	if c, err := combat.NewHealingCeilingAt(0); err != nil || c.HitPoints() != 0 {
		t.Errorf("NewHealingCeilingAt(0) = %v, %v", c, err)
	}
}

// unknownCeiling is a HealingCeiling the package does not know: only a type in
// package combat can implement it, so this embeds a real one.
type unknownCeiling struct{ combat.NoHealingCeiling }

// Requirement: per case
// Path: per case.
func TestGainTemporary(t *testing.T) {
	t.Parallel()
	dontStack := `SRD 5.2.1, "They Don't Stack", p. 18`
	cases := []struct {
		name, requirement, path string
		start                   combat.HitPointStatus
		amount                  int
		choice                  combat.TemporaryHitPointsChoice
		temp                    int
		events                  []string
	}{
		{"none held", `FR-028; "Temporary Hit Points", p. 18`, "happy", pcStatus(t, 10, 20, 0), 5, combat.KeepCurrent, 5, []string{"temporary_hit_points_gained"}},
		{"10 held, 12 offered, accept", dontStack, "unhappy (rule interaction)", pcStatus(t, 10, 20, 10), 12, combat.AcceptNew, 12, []string{"temporary_hit_points_gained"}},
		{"10 held, 12 offered, keep", dontStack, "unhappy (rule interaction)", pcStatus(t, 10, 20, 10), 12, combat.KeepCurrent, 10, []string{}},
		{"10 held, 8 offered, accept", dontStack, "unhappy (rule interaction)", pcStatus(t, 10, 20, 10), 8, combat.AcceptNew, 8, []string{"temporary_hit_points_gained"}},
		{"equal amount, accept", dontStack, "unhappy (boundary)", pcStatus(t, 10, 20, 10), 10, combat.AcceptNew, 10, []string{}},
		{"equal amount, keep", dontStack, "unhappy (boundary)", pcStatus(t, 10, 20, 10), 10, combat.KeepCurrent, 10, []string{}},
		{"at full Hit Points", `FR-031`, "unhappy (rule interaction)", pcStatus(t, 20, 20, 0), 3, combat.AcceptNew, 3, []string{"temporary_hit_points_gained"}},
		{"at 0 Hit Points stays Dying", `FR-033`, "unhappy (state error)", dyingStatus(t, 20, 1, 1), 4, combat.AcceptNew, 4, []string{"temporary_hit_points_gained"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := c.start.GainTemporary(subject(), c.amount, c.choice)
			a := applied(t, tr, err)
			if a.Status().HitPoints().Temporary() != c.temp || a.Status().HitPoints().Current() != c.start.HitPoints().Current() ||
				a.Status().LifeState() != c.start.LifeState() || a.HitPointsRegained() != 0 {
				t.Errorf("status = %+v %v [%s]", a.Status().HitPoints(), a.Status().LifeState(), c.requirement)
			}
			if got := kinds(a.Events()); !slices.Equal(got, c.events) {
				t.Errorf("events = %v, want %v", got, c.events)
			}
		})
	}
	st := pcStatus(t, 10, 20, 0)
	for name, call := range map[string]func() (combat.HitPointTransition, error){
		"zero amount":     func() (combat.HitPointTransition, error) { return st.GainTemporary(subject(), 0, combat.AcceptNew) },
		"negative amount": func() (combat.HitPointTransition, error) { return st.GainTemporary(subject(), -3, combat.AcceptNew) },
		"zero choice":     func() (combat.HitPointTransition, error) { return st.GainTemporary(subject(), 3, 0) },
		"zero subject":    func() (combat.HitPointTransition, error) { return st.GainTemporary("", 3, combat.AcceptNew) },
	} {
		if tr, err := call(); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, %v, want ErrInvalidInput (FR-030, FR-094)", name, tr, err)
		}
	}
	if tr, err := deadStatus(t, combat.MassiveDamage).GainTemporary(subject(), 3, combat.AcceptNew); err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("dead = %v, %v, want Refused(CreatureIsDead) (FR-075)", tr, err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-034, FR-098 - ending Temporary Hit Points sets them to 0
// and emits the event; with none held nothing changes; a Dead creature is refused (SRD 5.2.1, "Duration", p. 18).
// Path: per case.
func TestEndTemporary(t *testing.T) {
	t.Parallel()
	tr, err := pcStatus(t, 10, 20, 7).EndTemporary(subject())
	a := applied(t, tr, err)
	if a.Status().HitPoints().Temporary() != 0 || !slices.Equal(kinds(a.Events()), []string{"temporary_hit_points_ended"}) {
		t.Errorf("held = %+v %v", a.Status().HitPoints(), kinds(a.Events()))
	}
	st := pcStatus(t, 10, 20, 0)
	tr, err = st.EndTemporary(subject())
	if a = applied(t, tr, err); a.Status() != st || len(a.Events()) != 0 {
		t.Errorf("none held = %v, want unchanged and no event", a)
	}
	if tr, err := deadStatus(t, combat.MassiveDamage).EndTemporary(subject()); err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("dead = %v, %v", tr, err)
	}
	if tr, err := st.EndTemporary(""); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero subject = %v, %v", tr, err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-093 (amount 0 changes nothing, no event) and the maximum-reduction
// rows, per case.
// Path: per case.
func TestReduceMaximum(t *testing.T) {
	t.Parallel()
	req := `SRD 5.2.1, "Hit Point Maximum of 0", p. 17`
	cases := []struct {
		name, path     string
		start          combat.HitPointStatus
		amount         int
		current, max   int
		life           combat.LifeState
		events         []string
		maximumPrevMax [2]int
	}{
		{"lowers the maximum and the current", "happy", pcStatus(t, 20, 20, 0), 5, 15, 15, combat.Alive{}, []string{"hit_point_maximum_reduced"}, [2]int{20, 15}},
		{"current already below the new maximum", "unhappy (boundary)", pcStatus(t, 4, 20, 0), 5, 4, 15, combat.Alive{}, []string{"hit_point_maximum_reduced"}, [2]int{20, 15}},
		{"amount 0 changes nothing", "unhappy (boundary)", pcStatus(t, 20, 20, 0), 0, 20, 20, combat.Alive{}, []string{}, [2]int{}},
		{"to exactly 0 kills", "unhappy (boundary)", pcStatus(t, 20, 20, 0), 20, 0, 0, dead(combat.HitPointMaximumZero), []string{"hit_point_maximum_reduced", "died"}, [2]int{20, 0}},
		{"past 0 floors at 0 and kills", "unhappy (boundary)", pcStatus(t, 20, 20, 0), 99, 0, 0, dead(combat.HitPointMaximumZero), []string{"hit_point_maximum_reduced", "died"}, [2]int{20, 0}},
		{"a Dying creature keeps its state", "unhappy (rule interaction)", dyingStatus(t, 20, 1, 1), 5, 0, 15, dying(1, 1), []string{"hit_point_maximum_reduced"}, [2]int{20, 15}},
		{"to one", "unhappy (boundary)", pcStatus(t, 20, 20, 0), 19, 1, 1, combat.Alive{}, []string{"hit_point_maximum_reduced"}, [2]int{20, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := c.start.ReduceMaximum(subject(), c.amount)
			a := applied(t, tr, err)
			hp := a.Status().HitPoints()
			if hp.Current() != c.current || hp.Maximum() != c.max || a.Status().LifeState() != c.life {
				t.Errorf("status = %+v %v, want current %d max %d %v [%s]", hp, a.Status().LifeState(), c.current, c.max, c.life, req)
			}
			if got := kinds(a.Events()); !slices.Equal(got, c.events) {
				t.Errorf("events = %v, want %v", got, c.events)
			}
			if r, ok := firstEvent[combat.HitPointMaximumReduced](a.Events()); ok && [2]int{r.PreviousMaximum(), r.Maximum()} != c.maximumPrevMax {
				t.Errorf("event = %d to %d, want %v", r.PreviousMaximum(), r.Maximum(), c.maximumPrevMax)
			}
		})
	}
	st := pcStatus(t, 20, 20, 0)
	if tr, err := st.ReduceMaximum(subject(), -1); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative = %v, %v, want ErrInvalidInput (FR-047)", tr, err)
	}
	if tr, err := deadStatus(t, combat.Exhaustion).ReduceMaximum(subject(), 3); err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("dead = %v, %v (FR-075)", tr, err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-116 - RestoreMaximum raises a reduced maximum to the stated
// normal one and never lowers it; current Hit Points, LifeState and Temporary Hit Points stay; below 1 is invalid
// input; a Dead creature is refused (SRD 5.2.1, "Long Rest", p. 185).
// Path: per case.
func TestRestoreMaximum(t *testing.T) {
	t.Parallel()
	st := status(t, 4, 15, 2, combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
	tr, err := st.RestoreMaximum(subject(), 20)
	a := applied(t, tr, err)
	r, _ := firstEvent[combat.HitPointMaximumRestored](a.Events())
	if hp := a.Status().HitPoints(); hp.Maximum() != 20 || hp.Current() != 4 || hp.Temporary() != 2 || r.PreviousMaximum() != 15 || r.Maximum() != 20 || len(a.Events()) != 1 {
		t.Errorf("raised = %+v %v", hp, kinds(a.Events()))
	}
	for _, n := range []int{15, 10} {
		tr, err = st.RestoreMaximum(subject(), n)
		if a = applied(t, tr, err); a.Status() != st || len(a.Events()) != 0 {
			t.Errorf("RestoreMaximum(%d) changed the status or emitted: %v", n, a.Events())
		}
	}
	for _, n := range []int{0, -1} {
		if tr, err := st.RestoreMaximum(subject(), n); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("RestoreMaximum(%d) = %v, %v, want ErrInvalidInput", n, tr, err)
		}
	}
	if tr, err := deadStatus(t, combat.Exhaustion).RestoreMaximum(subject(), 30); err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("dead = %v, %v", tr, err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-095, FR-075 - RecordExhaustionDeath makes a creature that is
// not Dead Dead with cause Exhaustion and emits Died; a Dead creature is refused (SRD 5.2.1, Rules Glossary,
// "Exhaustion", p. 181).
// Path: per case.
func TestRecordExhaustionDeath(t *testing.T) {
	t.Parallel()
	for name, st := range map[string]combat.HitPointStatus{
		"alive": pcStatus(t, 10, 20, 3), "dying": dyingStatus(t, 20, 1, 1),
		"stable": status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()),
	} {
		tr, err := st.RecordExhaustionDeath(subject())
		a := applied(t, tr, err)
		d, _ := firstEvent[combat.Died](a.Events())
		if a.Status().LifeState() != dead(combat.Exhaustion) || a.Status().HitPoints().Current() != 0 || len(a.Events()) != 1 || d.Cause() != combat.Exhaustion {
			t.Errorf("%s = %v %v", name, a.Status().LifeState(), kinds(a.Events()))
		}
	}
	if tr, err := deadStatus(t, combat.MassiveDamage).RecordExhaustionDeath(subject()); err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("dead = %v, %v", tr, err)
	}
	if tr, err := (combat.HitPointStatus{}).RecordExhaustionDeath(subject()); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero status = %v, %v", tr, err)
	}
}
