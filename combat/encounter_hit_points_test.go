package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestDeathSavingThrowStabilizeAndRecoverCases(t *testing.T) {
	t.Parallel()
	u := condConds(combat.Unconscious{})
	dying := func(s, f int) combat.Combatant { return encC(t, "a", combat.SideA, dyingStatus(t, 20, s, f), u...) }
	stable := encC(t, "a", combat.SideA, status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), u...)
	type row struct {
		name, requirement, path string
		before                  combat.Combatant
		run                     func(e combat.Encounter) (encTr, error)
		kinds                   []string
		life                    string
		cur                     int
		unconscious, flag       bool
	}
	save := func(natural, total int) func(combat.Encounter) (encTr, error) {
		return func(e combat.Encounter) (encTr, error) { return e.ApplyDeathSavingThrow(cid("a"), natural, total) }
	}
	rows := []row{
		{"success", "FR-073; p. 17", "happy", dying(0, 0), save(12, 12), []string{"death_saving_throw_made"}, "dying", 0, true, true},
		{"natural 1", "FR-073; p. 18", "unhappy (boundary)", dying(0, 1), save(1, 12), []string{"death_saving_throw_made", "died"}, "dead:death_saving_throw_failures", 0, true, true},
		{"natural 20", "FR-073; p. 18", "unhappy (boundary)", dying(0, 1), save(20, 4), []string{"death_saving_throw_made", "hit_points_restored", "recovered", "condition_removed"}, "alive", 1, false, true},
		{"third success", "FR-073; p. 17", "unhappy (boundary)", dying(2, 0), save(10, 10), []string{"death_saving_throw_made", "became_stable"}, "stable", 0, true, true},
		{"Exhaustion lowers the total", "FR-073; p. 181", "unhappy (rule interaction)", dying(0, 0), save(11, 7), []string{"death_saving_throw_made"}, "dying", 0, true, true},
		{"stabilize a Dying character", "FR-074; p. 18", "happy", dying(1, 1), func(e combat.Encounter) (encTr, error) { return e.Stabilize(cid("a")) }, []string{"became_stable"}, "stable", 0, true, false},
		{"recover a Stable character", "FR-075; p. 18", "happy", stable, func(e combat.Encounter) (encTr, error) { return e.RecoverStable(cid("a")) },
			[]string{"hit_points_restored", "recovered", "condition_removed"}, "alive", 1, false, false},
		{"temporary Hit Points at 0", "FR-076; p. 18", "unhappy (rule interaction)", dying(0, 0), func(e combat.Encounter) (encTr, error) {
			return e.GainTemporaryHitPoints(cid("a"), 5, combat.AcceptNew)
		}, []string{"temporary_hit_points_gained"}, "dying", 0, true, false},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			e := encWith(t, r.before)
			tr, err := r.run(e)
			after, events := encOK(t, tr, err)
			if !slices.Equal(encKinds(events), r.kinds) {
				t.Errorf("events = %v, want %v [%s]", encKinds(events), r.kinds, r.requirement)
			}
			c := encMember(t, after, "a")
			if got := c.HitPoints().HitPoints().Current(); got != r.cur || encLife(c) != r.life || encExplicit(c, content.ConditionUnconscious) != r.unconscious || c.DeathSavingThrowMade() != r.flag {
				t.Errorf("after: %d %s unconscious %v flag %v, want %d %s %v %v [%s]", got, encLife(c), encExplicit(c, content.ConditionUnconscious), c.DeathSavingThrowMade(), r.cur, r.life, r.unconscious, r.flag, r.requirement)
			}
			encOthersEqual(t, e, after, "a")
			checkEnvelopes(t, events)
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-073, FR-099, FR-100, table 4.8.1 - a second throw in one turn is refused with DeathSavingThrowAlreadyMade after NotDying within the command; PrecheckDeathSavingThrow gives what Apply gives and changes nothing; the next turn clears the marker.
// Path: unhappy (state).
func TestDeathSavingThrowOncePerTurnAndPrecheck(t *testing.T) {
	t.Parallel()
	u := condConds(combat.Unconscious{})
	mk := func(hp combat.HitPointStatus, conds ...combat.Condition) combat.Encounter {
		return encWith(t, encC(t, "a", combat.SideA, hp, conds...))
	}
	// A is active at start, so build the marker through a first throw.
	e := mk(dyingStatus(t, 20, 0, 0), u...)
	tr, err := e.ApplyDeathSavingThrow(cid("a"), 12, 12)
	once, _ := encOK(t, tr, err)
	tr, err = once.ApplyDeathSavingThrow(cid("a"), 15, 15)
	r := encNo(t, once, tr, err)
	if r != (combat.DeathSavingThrowAlreadyMadeRejection{ID: cid("a")}) || r.String() != "death_saving_throw_already_made" {
		t.Errorf("second throw rejection = %#v [FR-099]", r)
	}
	if got := once.PrecheckDeathSavingThrow(cid("a")); got != r {
		t.Errorf("precheck = %#v, want %#v [FR-100]", got, r)
	}
	if got := e.PrecheckDeathSavingThrow(cid("a")); got != nil {
		t.Errorf("precheck of an accepted throw = %#v, want nil", got)
	}
	// Order inside the command: NotDying comes before the marker.
	alive := mk(pcStatus(t, 8, 20, 0))
	tr, err = alive.ApplyDeathSavingThrow(cid("a"), 15, 15)
	if r := encNo(t, alive, tr, err); r != (combat.HitPointRuleRejection{ID: cid("a"), Reason: combat.NotDying}) || r.String() != "not_dying" {
		t.Errorf("save while Alive = %#v, want HitPointRule(NotDying) [FR-073]", r)
	}
	// The next turn clears the flag; the throw is accepted and sets it again.
	two := encABCDying(t)
	tr, err = two.ApplyDeathSavingThrow(cid("a"), 12, 12)
	made1, _ := encOK(t, tr, err)
	for range 3 {
		made1, _ = advance(t, made1)
	}
	if encMember(t, made1, "a").DeathSavingThrowMade() {
		t.Fatalf("the marker was not cleared after a full round [FR-099]")
	}
	tr, err = made1.ApplyDeathSavingThrow(cid("a"), 12, 12)
	again, _ := encOK(t, tr, err)
	if !encMember(t, again, "a").DeathSavingThrowMade() {
		t.Errorf("the accepted throw did not set the marker again")
	}
}

func encABCDying(tb testing.TB) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encC(tb, "a", combat.SideA, dyingStatus(tb, 20, 0, 0), combat.Unconscious{}), encPC(tb, "b", combat.SideB, 20, 20), encPC(tb, "c", combat.SideA, 20, 20))
	return e
}

// Requirement: doc/specs/encounter-aggregate.md FR-100 - PrecheckDeathSavingThrow returns, in order, EncounterAlreadyEnded, UnknownCombatant, CombatantIsDead, HitPointRule(NotDying) and DeathSavingThrowAlreadyMade, and nil when a throw would be accepted; each equals what ApplyDeathSavingThrow rejects with.
// Path: unhappy (state).
func TestPrecheckDeathSavingThrowOrder(t *testing.T) {
	t.Parallel()
	dead := encWith(t, encC(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath)))
	alive := encWith(t, encPC(t, "a", combat.SideA, 8, 20))
	ended := endedEncounter(t, dead)
	cases := []struct {
		name string
		e    combat.Encounter
		id   string
		want combat.EncounterRejection
	}{
		{"ended before unknown and dead", ended, "zzz", combat.EncounterAlreadyEndedRejection{}},
		{"unknown", alive, "zzz", combat.UnknownCombatantRejection{ID: cid("zzz")}},
		{"dead before not dying", dead, "a", combat.CombatantIsDeadRejection{ID: cid("a")}},
		{"not dying", alive, "a", combat.HitPointRuleRejection{ID: cid("a"), Reason: combat.NotDying}},
		{"invalid encounter names no combatant", combat.Encounter{}, "a", combat.UnknownCombatantRejection{ID: cid("a")}},
		{"zero id names no combatant", alive, "", combat.UnknownCombatantRejection{}},
	}
	for _, c := range cases {
		before := c.e
		if got := c.e.PrecheckDeathSavingThrow(cid2(c.id)); got != c.want {
			t.Errorf("%s: precheck = %#v, want %#v [FR-100]", c.name, got, c.want)
		}
		if !c.e.Equal(before) {
			t.Errorf("%s: precheck changed the encounter", c.name)
		}
		if c.e.IsZero() || c.id == "" {
			continue
		}
		tr, err := c.e.ApplyDeathSavingThrow(cid(c.id), 12, 12)
		if r := encNo(t, c.e, tr, err); r != c.want {
			t.Errorf("%s: apply = %#v, want the precheck's %#v", c.name, r, c.want)
		}
	}
}

func cid2(s string) combat.CombatantID { return combat.CombatantID(s) }

// Requirement: doc/specs/encounter-aggregate.md FR-073, FR-074, FR-075, FR-076, FR-077, table 4.8.1 - stabilizing a Stable or a non Dying combatant returns HitPointRule(NotDying), recovering a Dying one HitPointRule(NotStable); a natural outside 1 to 20, Temporary Hit Points below 1 and a negative maximum reduction are errors.
// Path: unhappy (state, invalid input).
func TestHitPointOperationRefusalsAndErrors(t *testing.T) {
	t.Parallel()
	stable := encWith(t, encC(t, "a", combat.SideA, status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), combat.Unconscious{}))
	dying := encABCDying(t)
	tr, err := stable.Stabilize(cid("a"))
	if r := encNo(t, stable, tr, err); r != (combat.HitPointRuleRejection{ID: cid("a"), Reason: combat.NotDying}) {
		t.Errorf("stabilize a Stable = %#v", r)
	}
	tr, err = dying.RecoverStable(cid("a"))
	if r := encNo(t, dying, tr, err); r != (combat.HitPointRuleRejection{ID: cid("a"), Reason: combat.NotStable}) || r.String() != "not_stable" {
		t.Errorf("recover a Dying = %#v", r)
	}
	for name, f := range map[string]func() (encTr, error){
		"natural 0":      func() (encTr, error) { return dying.ApplyDeathSavingThrow(cid("a"), 0, 5) },
		"natural 21":     func() (encTr, error) { return dying.ApplyDeathSavingThrow(cid("a"), 21, 5) },
		"temporary 0":    func() (encTr, error) { return dying.GainTemporaryHitPoints(cid("a"), 0, combat.KeepCurrent) },
		"choice invalid": func() (encTr, error) { return dying.GainTemporaryHitPoints(cid("a"), 3, 0) },
		"maximum -1":     func() (encTr, error) { return dying.ReduceHitPointMaximum(cid("a"), -1) },
		"zero id":        func() (encTr, error) { return dying.Stabilize("") },
		"zero encounter": func() (encTr, error) { return (combat.Encounter{}).RecoverStable(cid("a")) },
	} {
		if res, err := f(); err == nil || res != nil {
			t.Errorf("%s: want an ErrInvalidInput error, got %v, %v", name, res, err)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-077, FR-068, table 4.8.1 - reducing the maximum to 0 kills the combatant with the wrapped Died (hit_point_maximum_zero) after HitPointMaximumReduced, and the death is derived from the status.
// Path: unhappy (boundary).
func TestReduceHitPointMaximumToZeroKills(t *testing.T) {
	t.Parallel()
	e := encWith(t, encPC(t, "a", combat.SideA, 5, 5))
	tr, err := e.ReduceHitPointMaximum(cid("a"), 5)
	after, events := encOK(t, tr, err)
	if !slices.Equal(encKinds(events), []string{"hit_point_maximum_reduced", "died"}) {
		t.Errorf("events = %v", encKinds(events))
	}
	if cause, ok := encMember(t, after, "a").Death(); !ok || cause != combat.HitPointMaximumZero {
		t.Errorf("Death = %v, %v, want hit_point_maximum_zero [FR-068]", cause, ok)
	}
	// A death stays after later transitions.
	tr, err = after.ApplyHealing(cid("a"), 3)
	rej := encNo(t, after, tr, err)
	if _, ok := encMember(t, after, "a").Death(); !ok || rej == nil {
		t.Errorf("a death must stay [FR-068]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-068 - each of the five death causes sets the death of the combatant.
// Path: unhappy (state).
func TestEveryDeathCauseSetsTheDeath(t *testing.T) {
	t.Parallel()
	for _, cause := range []combat.DeathCause{combat.MonsterDeath, combat.HitPointMaximumZero, combat.MassiveDamage, combat.DeathSavingThrowFailures, combat.Exhaustion} {
		e := encWith(t, encC(t, "a", combat.SideA, deadStatus(t, cause)))
		if got, ok := encMember(t, e, "a").Death(); !ok || got != cause {
			t.Errorf("Death = %v, %v, want %v", got, ok, cause)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-073, FR-076, FR-077, FR-078 - an invalid input value is an error before any rejection, on an ended encounter too; the boundary values (natural 1 and 20, Temporary Hit Points of 1, a maximum reduction of 0) are accepted.
// Path: unhappy (boundary, invalid input).
func TestInvalidValuesAreErrorsBeforeRejectionsAndBoundariesAreAccepted(t *testing.T) {
	t.Parallel()
	ended := endedEncounter(t, encABCDying(t))
	for name, f := range map[string]func() (encTr, error){
		"natural 21":          func() (encTr, error) { return ended.ApplyDeathSavingThrow(cid("a"), 21, 5) },
		"natural 0":           func() (encTr, error) { return ended.ApplyDeathSavingThrow(cid("a"), 0, 5) },
		"temporary amount 0":  func() (encTr, error) { return ended.GainTemporaryHitPoints(cid("a"), 0, combat.AcceptNew) },
		"temporary choice":    func() (encTr, error) { return ended.GainTemporaryHitPoints(cid("a"), 1, 0) },
		"maximum -1":          func() (encTr, error) { return ended.ReduceHitPointMaximum(cid("a"), -1) },
		"escape outcome":      func() (encTr, error) { return ended.EscapeGrapple(cid("a"), 3) },
		"grapple distance -1": func() (encTr, error) { return ended.CheckGrapple(cid("a"), -1, 0) },
		"grapple range -1":    func() (encTr, error) { return ended.CheckGrapple(cid("a"), 0, -1) },
	} {
		if res, err := f(); err == nil || res != nil {
			t.Errorf("%s on an ended encounter: want an ErrInvalidInput error, got %v, %v", name, res, err)
		}
	}
	live := encABCDying(t)
	for name, f := range map[string]func() (encTr, error){
		"natural 1":      func() (encTr, error) { return live.ApplyDeathSavingThrow(cid("a"), 1, 1) },
		"natural 20":     func() (encTr, error) { return live.ApplyDeathSavingThrow(cid("a"), 20, 1) },
		"temporary 1":    func() (encTr, error) { return live.GainTemporaryHitPoints(cid("a"), 1, combat.AcceptNew) },
		"maximum 0":      func() (encTr, error) { return live.ReduceHitPointMaximum(cid("a"), 0) },
		"grapple 0 of 0": func() (encTr, error) { return live.CheckGrapple(cid("a"), 0, 0) },
	} {
		tr, err := f()
		if err != nil {
			t.Errorf("%s: err = %v, want accepted", name, err)
			continue
		}
		_ = tr
	}
}
