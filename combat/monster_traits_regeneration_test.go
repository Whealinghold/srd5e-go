package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func mtState(tb testing.TB, t combat.RegenerationTrait) combat.RegenerationState {
	tb.Helper()
	return luMust(combat.NewRegenerationState(t))
}

func mtTrollStatus(tb testing.TB, current int) combat.HitPointStatus {
	tb.Helper()
	return status(tb, current, 84, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
}

func healed(tb testing.TB, turn combat.RegenerationTurn) combat.RegenerationHealed {
	tb.Helper()
	h, ok := turn.(combat.RegenerationHealed)
	if !ok {
		tb.Fatalf("turn = %#v, want healed", turn)
	}
	return h
}

func skipped(tb testing.TB, turn combat.RegenerationTurn, want combat.RegenerationSkipReason) combat.RegenerationSkipped {
	tb.Helper()
	s, ok := turn.(combat.RegenerationSkipped)
	if !ok || s.Reason() != want {
		tb.Fatalf("turn = %#v, want skipped %v", turn, want)
	}
	return s
}

// Requirement: doc/specs/monster-traits.md FR-040 - a state holds a Regeneration, not suppressed; a zero trait is rejected.
// Path: unhappy (invalid input).
func TestNewRegenerationState(t *testing.T) {
	t.Parallel()
	s := mtState(t, mtTroll(t))
	if s.Suppressed() || s.IsZero() || s.Trait() != mtTroll(t) {
		t.Errorf("state = %#v", s)
	}
	if _, err := combat.NewRegenerationState(combat.RegenerationTrait{}); !isInvalid(err) {
		t.Errorf("zero trait err = %v", err)
	}
	if !(combat.RegenerationState{}).IsZero() {
		t.Error("the zero state is valid")
	}
}

// Requirement: doc/specs/monster-traits.md FR-042, table RG-1 - the healing rows: heals, capped at the maximum, at the maximum regains 0 with no event, healing at the turn start leaves the state unchanged.
// Path: per case.
func TestRegenerationHeals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		current      int
		want, events int
	}{{"heals", 40, 55, 2 - 1}, {"capped", 80, 84, 1}, {"one short", 83, 84, 1}, {"at maximum", 84, 84, 0}}
	for _, c := range cases {
		s := mtState(t, mtTroll(t))
		h := healed(t, s.AtStartOfTurn(mtTrollStatus(t, c.current), mtID()))
		if got := h.Transition().Status().HitPoints().Current(); got != c.want || len(h.Transition().Events()) != c.events || h.State() != s {
			t.Errorf("%s: %d hit points, %d events, state %v", c.name, got, len(h.Transition().Events()), h.State())
		}
	}
	if h := healed(t, mtState(t, mtTroll(t)).AtStartOfTurn(mtTrollStatus(t, 80), mtID())); h.Transition().HitPointsRegained() != 4 {
		t.Errorf("regained %d, want 4", h.Transition().HitPointsRegained())
	}
}

// Requirement: doc/specs/monster-traits.md FR-041, FR-044, table RG-1 - Acid and Fire suppress once with one event, Cold does not, a second suppression adds no event, and the turn start after skips once then heals.
// Path: per case.
func TestRegenerationSuppression(t *testing.T) {
	t.Parallel()
	s := mtState(t, mtTroll(t))
	for _, c := range []struct {
		name  string
		dealt []content.DamageType
		want  bool
	}{{"Fire", damageTypes(content.DamageFire), true}, {"Acid and Slashing", damageTypes(content.DamageAcid, content.DamageSlashing), true},
		{"Cold", damageTypes(content.DamageCold), false}, {"nothing", nil, false}, {"invalid type", damageTypes(0), false}} {
		step := s.AfterDamage(mtID(), c.dealt)
		if step.State().Suppressed() != c.want || len(step.Events()) != map[bool]int{true: 1, false: 0}[c.want] {
			t.Errorf("%s: %v %v", c.name, step.State(), step.Events())
		}
		if !c.want && step.State() != s {
			t.Errorf("%s: state changed", c.name)
		}
	}
	first := s.AfterDamage(mtID(), damageTypes(content.DamageFire))
	if e := first.Events()[0]; e.Kind() != "regeneration_suppressed" || !slices.Equal(e.Combatants(), []combat.CombatantID{mtID()}) {
		t.Errorf("event = %v", e)
	}
	again := first.State().AfterDamage(mtID(), damageTypes(content.DamageFire))
	if len(again.Events()) != 0 || again.State() != first.State() {
		t.Error("a second suppression added an event")
	}
	ev := first.Events()
	ev[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
	skip := skipped(t, first.State().AtStartOfTurn(mtTrollStatus(t, 40), mtID()), combat.RegenerationSkipSuppressed)
	if skip.State().Suppressed() {
		t.Error("the flag was not cleared")
	}
	if h := healed(t, skip.State().AtStartOfTurn(mtTrollStatus(t, 40), mtID())); h.Transition().Status().HitPoints().Current() != 55 {
		t.Error("the turn after did not heal 15")
	}
	// The first form is never suppressed.
	oni := mtState(t, mtOni(t))
	if step := oni.AfterDamage(mtID(), damageTypes(content.DamageFire)); step.State() != oni || len(step.Events()) != 0 {
		t.Error("an Oni was suppressed")
	}
	if step := (combat.RegenerationState{}).AfterDamage(mtID(), damageTypes(content.DamageFire)); len(step.Events()) != 0 {
		t.Error("zero state emitted an event")
	}
	if step := s.AfterDamage("", damageTypes(content.DamageFire)); len(step.Events()) != 0 || step.State().Suppressed() {
		t.Error("zero subject suppressed")
	}
}

// Requirement: doc/specs/monster-traits.md FR-043, table RG-1 - the skips: Oni Dead, Oni at 0, Oni at 1, Troll Dead, a swarm, a Stable Troll qualifies; zero inputs.
// Path: per case.
func TestRegenerationQualifies(t *testing.T) {
	t.Parallel()
	oni := mtState(t, mtOni(t))
	skipped(t, oni.AtStartOfTurn(deadStatus(t, combat.HitPointMaximumZero), mtID()), combat.RegenerationSkipNoHitPoints)
	stable := status(t, 0, 66, 0, combat.Stable{}, combat.DiesInstantly, combat.NoAffinities())
	skipped(t, oni.AtStartOfTurn(stable, mtID()), combat.RegenerationSkipNoHitPoints)
	mid := status(t, 40, 66, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
	if h := healed(t, oni.AtStartOfTurn(mid, mtID())); h.Transition().Status().HitPoints().Current() != 50 {
		t.Error("Oni at 40 did not reach 50")
	}
	one := status(t, 1, 66, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
	if h := healed(t, oni.AtStartOfTurn(one, mtID())); h.Transition().Status().HitPoints().Current() != 11 {
		t.Error("Oni at 1 did not reach 11")
	}
	troll := mtState(t, mtTroll(t))
	dead := deadStatus(t, combat.HitPointMaximumZero)
	skipped(t, troll.AtStartOfTurn(dead, mtID()), combat.RegenerationSkipNoHitPoints)
	if h := healed(t, troll.AtStartOfTurn(status(t, 0, 84, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), mtID())); h.Transition().Status().LifeState() != (combat.Alive{}) {
		t.Error("a Stable Troll did not recover")
	}
	swarm := luMust(combat.NewSwarmTrait(mtRef(t, 1)))
	marked := luMust(mtOni(t).WithSwarm(swarm))
	skipped(t, mtState(t, marked).AtStartOfTurn(one, mtID()), combat.RegenerationSkipSwarmCannotRegain)
	skipped(t, (combat.RegenerationState{}).AtStartOfTurn(one, mtID()), combat.RegenerationSkipNoHitPoints)
	skipped(t, oni.AtStartOfTurn(combat.HitPointStatus{}, mtID()), combat.RegenerationSkipNoHitPoints)
	skipped(t, oni.AtStartOfTurn(one, ""), combat.RegenerationSkipNoHitPoints)
	for i, w := range []string{"suppressed", "no_hit_points", "swarm_cannot_regain", "unknown"} {
		if combat.RegenerationSkipReason(i+1).String() != w {
			t.Errorf("skip reason %d", i+1)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-045, table RG-1 - a Troll dropped to 0 is Stable at 0 with no Died and FellUnconscious once; the unconscious and massive-damage forms too.
// Path: per case.
func TestDeferRegenerationDeath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		rule   combat.ZeroHitPointsRule
		damage int
		cause  string
	}{{"Troll dropped, DiesInstantly", combat.DiesInstantly, 84, "monster_death"},
		{"unconscious Troll to Dying", combat.FallsUnconscious, 84, "dying"},
		{"massive damage", combat.FallsUnconscious, 168, "massive_damage"}}
	for _, c := range cases {
		before := status(t, 84, 84, 0, combat.Alive{}, c.rule, combat.NoAffinities())
		hit := mtHit(t, before, combat.NormalHit, mtSlash(c.damage))
		out, ok := combat.DeferRegenerationDeath(mtTroll(t), mtID(), hit).(combat.RegenerationDeferred)
		if !ok {
			t.Fatalf("%s: not deferred, status %v", c.name, hit.Status().LifeState())
		}
		hp := out.Status().HitPoints()
		kinds := mtKindsOf(out.Events())
		fell := 0
		for _, k := range kinds {
			if k == "fell_unconscious" {
				fell++
			}
		}
		if out.Status().LifeState() != (combat.Stable{}) || hp.Current() != 0 || hp.Maximum() != 84 || out.Status().ZeroHitPointsRule() != c.rule ||
			slices.Contains(kinds, "died") || fell != 1 || kinds[len(kinds)-1] != "regeneration_death_deferred" {
			t.Errorf("%s: status %v events %v", c.name, out.Status().LifeState(), kinds)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-045, FR-047 - no deferral for the first form, a hit that is no drop, a maximum drained to 0, Exhaustion, a Dying creature damaged again, zero inputs.
// Path: per case.
func TestRegenerationNotDeferred(t *testing.T) {
	t.Parallel()
	troll := mtTroll(t)
	alive := mtZombie(t, 84, 84, 0, combat.DiesInstantly)
	drop := mtHit(t, alive, combat.NormalHit, mtSlash(84))
	notDrop := mtHit(t, alive, combat.NormalHit, mtSlash(3))
	drained := applied(t, luMust(alive.ReduceMaximum(mtID(), 84)), nil)
	exhausted := applied(t, luMust(alive.RecordExhaustionDeath(mtID())), nil)
	hitDying := mtHit(t, dyingStatus(t, 84, 0, 0), combat.NormalHit, mtSlash(3))
	cases := []struct {
		name  string
		trait combat.RegenerationTrait
		who   combat.CombatantID
		tr    combat.HitPointTransitionApplied
		want  combat.RegenerationNotDeferredReason
	}{
		{"first form", mtOni(t), mtID(), drop, combat.RegenerationNoDeferral},
		{"zero trait", combat.RegenerationTrait{}, mtID(), drop, combat.RegenerationNoDeferral},
		{"zero subject", troll, "", drop, combat.RegenerationNoDeferral},
		{"no drop", troll, mtID(), notDrop, combat.RegenerationNoDropToZero},
		{"maximum drained", troll, mtID(), drained, combat.RegenerationNoDropToZero},
		{"Exhaustion", troll, mtID(), exhausted, combat.RegenerationNoDropToZero},
		{"damage at 0", troll, mtID(), hitDying, combat.RegenerationNoDropToZero},
	}
	for _, c := range cases {
		got, ok := combat.DeferRegenerationDeath(c.trait, c.who, c.tr).(combat.RegenerationNotDeferred)
		if !ok || got.Reason != c.want {
			t.Errorf("%s: %#v, want %v", c.name, got, c.want)
		}
	}
	forged := combat.HitPointTransitionApplied{}
	if got := combat.DeferRegenerationDeath(troll, mtID(), forged).(combat.RegenerationNotDeferred); got.Reason != combat.RegenerationNoDropToZero {
		t.Errorf("zero transition: %v", got)
	}
	for i, w := range []string{"no_deferral", "no_drop_to_zero", "cause_not_deferrable", "unknown"} {
		if combat.RegenerationNotDeferredReason(i+1).String() != w {
			t.Errorf("reason %d", i+1)
		}
	}
}

// Requirement: doc/specs/monster-traits.md FR-046, table RG-1 - a deferred Troll heals and recovers at its turn start; one that took Acid or Fire dies of Monster Death.
// Path: per case.
func TestDeferredRegenerationThenTurnStart(t *testing.T) {
	t.Parallel()
	alive := status(t, 84, 84, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
	deferred := combat.DeferRegenerationDeath(mtTroll(t), mtID(), mtHit(t, alive, combat.NormalHit, mtSlash(84))).(combat.RegenerationDeferred)
	s := mtState(t, mtTroll(t))
	h := healed(t, s.AtStartOfTurn(deferred.Status(), mtID()))
	if h.Transition().Status().HitPoints().Current() != 15 || h.Transition().Status().LifeState() != (combat.Alive{}) || !slices.Equal(mtKindsOf(toEvents(h.Transition().Events())), []string{"hit_points_restored", "recovered"}) {
		t.Errorf("heal = %v", h.Transition().Status())
	}
	acid := s.AfterDamage(mtID(), damageTypes(content.DamageAcid)).State()
	die, ok := acid.AtStartOfTurn(deferred.Status(), mtID()).(combat.RegenerationLetDie)
	if !ok || die.Status().LifeState() != combat.LifeState(luMust(combat.NewDead(combat.MonsterDeath))) || die.State().Suppressed() {
		t.Fatalf("turn = %#v", die)
	}
	ev := die.Events()
	if len(ev) != 1 || ev[0].(combat.Died).Cause() != combat.MonsterDeath || ev[0].Kind() != "died" {
		t.Errorf("events = %v", ev)
	}
	ev[0] = nil
	if die.Events()[0] == nil {
		t.Error("Events is not a copy")
	}
	if die.Status().HitPoints().Current() != 0 {
		t.Error("a dead creature has Hit Points")
	}
	// A Dying creature that does not regenerate dies as well.
	dyingTroll := status(t, 0, 84, 0, combat.Dying{}, combat.FallsUnconscious, combat.NoAffinities())
	if _, ok := acid.AtStartOfTurn(dyingTroll, mtID()).(combat.RegenerationLetDie); !ok {
		t.Error("a suppressed Dying Troll did not die")
	}
}

func toEvents(in []combat.HitPointEvent) []combat.Event {
	out := make([]combat.Event, len(in))
	for i, e := range in {
		out[i] = e
	}
	return out
}
