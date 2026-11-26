package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-087 - NewHitPointStatus starts at full Hit Points, with no
// Temporary Hit Points and Alive, and rejects a maximum below 1 with MaximumBelowOne and a missing rule.
// Path: per case.
func TestNewHitPointStatus(t *testing.T) {
	t.Parallel()
	s, err := combat.NewHitPointStatus(20, combat.DiesInstantly, combat.NoAffinities())
	hp := s.HitPoints()
	if err != nil || hp != combat.MustHitPoints(20, 20, 0) || s.LifeState() != (combat.Alive{}) || s.ZeroHitPointsRule() != combat.DiesInstantly ||
		s.Affinities() != combat.NoAffinities() || s.IsZero() || s.IsUnconscious() || s.MakesDeathSavingThrows() {
		t.Fatalf("= %+v, %v", s, err)
	}
	if combat.MustHitPointStatus(20, combat.DiesInstantly, combat.NoAffinities()) != s {
		t.Errorf("MustHitPointStatus differs")
	}
	for _, m := range []int{0, -1, math.MinInt} {
		_, err := combat.NewHitPointStatus(m, combat.FallsUnconscious, combat.NoAffinities())
		var se *combat.HitPointStatusError
		if !errors.As(err, &se) || !errors.Is(err, srderr.ErrInvalidInput) || !slices.Equal(se.Problems(), []combat.HitPointStatusProblem{combat.MaximumBelowOne}) {
			t.Errorf("maximum %d error = %v, want MaximumBelowOne", m, err)
		}
	}
	if _, err := combat.NewHitPointStatus(5, 0, combat.NoAffinities()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no rule error = %v, want ErrInvalidInput", err)
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustHitPointStatus panic = %v", r)
		}
	}()
	combat.MustHitPointStatus(0, combat.DiesInstantly, combat.NoAffinities())
}

// Requirement: doc/specs/hit-points-and-damage.md FR-087 to FR-090, FR-105, FR-106 - RestoreHitPointStatus
// accepts the consistent combinations and returns a *HitPointStatusError listing every problem, in enum order,
// for the rest.
// Path: per case.
func TestRestoreHitPointStatus(t *testing.T) {
	t.Parallel()
	fall := combat.FallsUnconscious
	hp := combat.MustHitPoints
	P := func(ps ...combat.HitPointStatusProblem) []combat.HitPointStatusProblem { return ps }
	cases := []struct {
		name, requirement, path string
		hp                      combat.HitPoints
		life                    combat.LifeState
		problems                []combat.HitPointStatusProblem
	}{
		{"alive", `FR-087`, "happy", hp(5, 20, 3), combat.Alive{}, nil},
		{"knocked out is alive at 1", `FR-090`, "unhappy (boundary)", hp(1, 20, 0), combat.Alive{}, nil},
		{"dying at 0", `FR-089`, "happy", hp(0, 20, 0), dying(1, 2), nil},
		{"stable at 0", `FR-089`, "happy", hp(0, 20, 0), combat.Stable{}, nil},
		{"dead at 0", `FR-089`, "happy", hp(0, 20, 0), dead(combat.MassiveDamage), nil},
		{"dead by maximum zero", `FR-106`, "unhappy (boundary)", hp(0, 0, 0), dead(combat.HitPointMaximumZero), nil},
		{"dead with Hit Points", `FR-089`, "unhappy (invalid input)", hp(1, 20, 0), dead(combat.MassiveDamage), P(combat.DeadWithHitPoints)},
		{"dying with Hit Points", `FR-089`, "unhappy (invalid input)", hp(1, 20, 0), dying(0, 0), P(combat.DyingWithHitPoints)},
		{"stable with Hit Points", `FR-089`, "unhappy (invalid input)", hp(1, 20, 0), combat.Stable{}, P(combat.StableWithHitPoints)},
		{"alive at 0", `FR-090`, "unhappy (invalid input)", hp(0, 20, 0), combat.Alive{}, P(combat.AliveAtZeroHitPoints)},
		{"maximum 0 and Dying", `FR-106`, "unhappy (invalid input)", hp(0, 0, 0), dying(0, 0), P(combat.HitPointMaximumZeroNotDead)},
		{"maximum 0 and Stable", `FR-106`, "unhappy (invalid input)", hp(0, 0, 0), combat.Stable{}, P(combat.HitPointMaximumZeroNotDead)},
		{"maximum 0 and Alive", `FR-106`, "unhappy (invalid input)", hp(0, 0, 0), combat.Alive{}, P(combat.AliveAtZeroHitPoints, combat.HitPointMaximumZeroNotDead)},
		{"maximum 0 and another death cause", `FR-106`, "unhappy (invalid input)", hp(0, 0, 0), dead(combat.MassiveDamage), P(combat.HitPointMaximumZeroNotDead)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.RestoreHitPointStatus(c.hp, c.life, fall, combat.NoAffinities())
			if c.problems == nil {
				if err != nil || s.HitPoints() != c.hp || s.LifeState() != c.life {
					t.Fatalf("= %+v, %v [%s]", s, err, c.requirement)
				}
				return
			}
			var se *combat.HitPointStatusError
			if !errors.As(err, &se) || !errors.Is(err, srderr.ErrInvalidInput) || !slices.Equal(se.Problems(), c.problems) || !s.IsZero() {
				t.Fatalf("= %+v, %v, want problems %v [%s]", s, err, c.problems, c.requirement)
			}
			if p := se.Problems(); len(p) > 0 {
				p[0] = 0
				if se.Problems()[0] == 0 {
					t.Errorf("Problems is not a copy")
				}
			}
		})
	}
	for name, call := range map[string]func() (combat.HitPointStatus, error){
		"nil life": func() (combat.HitPointStatus, error) {
			return combat.RestoreHitPointStatus(hp(5, 5, 0), nil, fall, combat.NoAffinities())
		},
		"no rule": func() (combat.HitPointStatus, error) {
			return combat.RestoreHitPointStatus(hp(5, 5, 0), combat.Alive{}, 0, combat.NoAffinities())
		},
		"dead no cause": func() (combat.HitPointStatus, error) {
			return combat.RestoreHitPointStatus(hp(0, 5, 0), combat.Dead{}, fall, combat.NoAffinities())
		},
	} {
		if s, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || !s.IsZero() {
			t.Errorf("%s = %+v, %v, want zero and ErrInvalidInput", name, s, err)
		}
	}
	_, a := combat.RestoreHitPointStatus(hp(1, 5, 0), combat.Stable{}, fall, combat.NoAffinities())
	_, b := combat.RestoreHitPointStatus(hp(2, 9, 0), combat.Stable{}, fall, combat.NoAffinities())
	_, d := combat.RestoreHitPointStatus(hp(0, 5, 0), combat.Alive{}, fall, combat.NoAffinities())
	if !errors.Is(a, b) || errors.Is(a, d) || a.Error() == "" {
		t.Errorf("error identity wrong")
	}
	for p, want := range map[combat.HitPointStatusProblem]string{combat.MaximumBelowOne: "maximum_below_one", combat.DeadWithHitPoints: "dead_with_hit_points",
		combat.DyingWithHitPoints: "dying_with_hit_points", combat.StableWithHitPoints: "stable_with_hit_points", combat.AliveAtZeroHitPoints: "alive_at_zero_hit_points",
		combat.HitPointMaximumZeroNotDead: "hit_point_maximum_zero_not_dead", 0: "unknown"} {
		if p.String() != want {
			t.Errorf("%d.String() = %q, want %q", p, p.String(), want)
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-070, FR-096 - a Stable and a Dying creature are Unconscious
// through the 0-Hit-Point rules, an Alive or Dead one is not; a Death Saving Throw is due only for Dying, whatever
// the kind of creature.
// Path: per case.
func TestStatusPredicates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                string
		s                   combat.HitPointStatus
		unconscious, saving bool
	}{
		{"alive", pcStatus(t, 5, 20, 0), false, false},
		{"knocked out (alive at 1)", pcStatus(t, 1, 20, 0), false, false},
		{"dying", dyingStatus(t, 20, 0, 0), true, true},
		{"dying monster treated like a character", status(t, 0, 20, 0, dying(0, 0), combat.FallsUnconscious, combat.NoAffinities()), true, true},
		{"stable", status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), true, false},
		{"dead", deadStatus(t, combat.MonsterDeath), false, false},
	}
	for _, c := range cases {
		if c.s.IsUnconscious() != c.unconscious || c.s.MakesDeathSavingThrows() != c.saving {
			t.Errorf("%s: unconscious %v, saving %v, want %v and %v", c.name, c.s.IsUnconscious(), c.s.MakesDeathSavingThrows(), c.unconscious, c.saving)
		}
	}
	if !(combat.HitPointStatus{}).IsZero() || (combat.HitPointStatus{}).IsUnconscious() {
		t.Errorf("zero status wrong")
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-105, testing.md section 10 - RestoreHitPointStatus never
// panics for any numbers: it returns a status or a typed error, and a returned status satisfies the invariants.
// Path: unhappy (invalid input).
func FuzzRestoreHitPointStatus(f *testing.F) {
	for _, s := range [][4]int{{5, 20, 0, 0}, {0, 20, 0, 1}, {0, 0, 0, 3}, {3, 3, 3, 2}, {math.MaxInt, math.MaxInt, math.MaxInt, 0}} {
		f.Add(s[0], s[1], s[2], s[3])
	}
	f.Fuzz(func(t *testing.T, current, maximum, temp, which int) {
		hp, err := combat.NewHitPoints(current, maximum, temp)
		if err != nil {
			return
		}
		lives := []combat.LifeState{combat.Alive{}, dying(1, 1), combat.Stable{}, dead(combat.MassiveDamage), dead(combat.HitPointMaximumZero)}
		life := lives[((which%len(lives))+len(lives))%len(lives)]
		s, err := combat.RestoreHitPointStatus(hp, life, combat.FallsUnconscious, combat.NoAffinities())
		if err != nil {
			var se *combat.HitPointStatusError
			if !errors.As(err, &se) || len(se.Problems()) == 0 {
				t.Fatalf("untyped or empty error: %v", err)
			}
			return
		}
		if _, alive := s.LifeState().(combat.Alive); alive != (hp.Current() > 0) {
			t.Errorf("accepted %+v with %v", hp, s.LifeState())
		}
		if hp.Maximum() == 0 && s.LifeState() != combat.LifeState(dead(combat.HitPointMaximumZero)) {
			t.Errorf("maximum 0 accepted with %v", s.LifeState())
		}
	})
}

// Requirement: doc/specs/hit-points-and-damage.md FR-118 - the host mappings of the species handoffs hold on the
// combat side: a Temporary Hit Points grant against 3 held follows the host's choice (5 accepted, 3 kept), and the
// events tell the host whether damage killed outright (Died), dropped the creature without killing it
// (DroppedToZeroHitPoints without Died) or knocked it out (KnockedOut, no drop) (SRD 5.2.1, "Instant Death", p. 17;
// "They Don't Stack", p. 18).
// Path: unhappy (rule interaction).
func TestHostMappingsOfSpeciesHandoffs(t *testing.T) {
	t.Parallel()
	for choice, want := range map[combat.TemporaryHitPointsChoice]int{combat.AcceptNew: 5, combat.KeepCurrent: 3} {
		tr, err := pcStatus(t, 10, 20, 3).GainTemporary(subject(), 5, choice)
		if got := applied(t, tr, err).Status().HitPoints().Temporary(); got != want {
			t.Errorf("choice %d: temporary = %d, want %d", choice, got, want)
		}
	}
	cases := []struct {
		name    string
		st      combat.HitPointStatus
		dmg     int
		del     combat.DamageDelivery
		died    bool
		dropped bool
	}{
		{"monster death", status(t, 5, 20, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities()), 5, combat.OtherSource, true, true},
		{"Massive Damage", pcStatus(t, 6, 12, 0), 18, combat.OtherSource, true, true},
		{"drop without dying", pcStatus(t, 5, 20, 0), 5, combat.OtherSource, false, true},
		{"knock-out", pcStatus(t, 5, 20, 0), 5, combat.MeleeAttackKnockOut, false, false},
	}
	for _, c := range cases {
		tr, err := c.st.ApplyDamage(subject(), []combat.Damage{fire(c.dmg)}, nil, c.del, combat.NormalHit, nil)
		ev := applied(t, tr, err).Events()
		_, died := firstEvent[combat.Died](ev)
		_, dropped := firstEvent[combat.DroppedToZeroHitPoints](ev)
		if died != c.died || dropped != c.dropped {
			t.Errorf("%s: died %v dropped %v, want %v and %v", c.name, died, dropped, c.died, c.dropped)
		}
	}
}
