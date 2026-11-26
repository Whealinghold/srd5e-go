package combat_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
)

// rtlEffects is the baseline of table 4.2: the contagion c1, the curse c2 and the spell s1.
func rtlEffects() combat.OngoingEffects {
	return meEffects(meEffect("c1", combat.OngoingEffectMagicalContagion, meLabeled("rot")), meCurse("c2"),
		meEffect("s1", combat.OngoingEffectSpell, meSpell("bless")))
}

func rtlIDs(ids ...string) []combat.OngoingEffectID {
	var out []combat.OngoingEffectID
	for _, id := range ids {
		out = append(out, meEID(id))
	}
	return out
}

// rtlCommand is the baseline of table 4.1: a Dead creature with maximum 20, spirit accepts, Poisoned and Blinded,
// the three effects of table 4.2 and nothing elapsed.
func rtlCommand(tb testing.TB, hitPoints int) combat.ReviveCreatureCommand {
	tb.Helper()
	return combat.ReviveCreatureCommand{Subject: gid("hero"), Status: deadStatus(tb, combat.DeathSavingThrowFailures), HitPoints: hitPoints,
		Consent: combat.SpiritAccepts, Conditions: meConditions(combat.Poisoned{}, combat.Blinded{}), Effects: rtlEffects()}
}

func rtlApplied(tb testing.TB, c combat.ReviveCreatureCommand) combat.ReviveCreatureApplied {
	tb.Helper()
	res, ok := combat.Revive(c).(combat.ReviveCreatureApplied)
	if !ok {
		tb.Fatalf("Revive = %T, want ReviveCreatureApplied", combat.Revive(c))
	}
	return res
}

func rtlKinds(events []combat.Event) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// rtlCurrent reads the current Hit Points of a status.
func rtlCurrent(s combat.HitPointStatus) int { return s.HitPoints().Current() }

// Requirement: per case
// Path: per case.
func TestReviveHitPoints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		maximum, value          int
		want                    string
		current                 int
	}{
		{"one Hit Point", "FR-003, table 4.1; SRD 5.2.1, \"Dead\", p. 180", "happy", 20, 1, "applied", 1},
		{"a typical value", "FR-003, table 4.1; SRD 5.2.1, \"Dead\", p. 180", "happy", 20, 7, "applied", 7},
		{"equal to the maximum", "FR-003, table 4.1; SRD 5.2.1, \"Dead\", p. 180", "unhappy (boundary)", 20, 20, "applied", 20},
		{"one above the maximum", "FR-005, table 4.1; SRD 5.2.1, \"Hit Points\", p. 183", "unhappy (boundary)", 20, 21, "applied", 20},
		{"the largest integer", "FR-005; SRD 5.2.1, \"Hit Points\", p. 183", "unhappy (boundary)", 20, math.MaxInt, "applied", 20},
		{"zero", "FR-004, table 4.1", "unhappy (boundary)", 20, 0, "below_one", 0},
		{"negative", "FR-004, table 4.1", "unhappy (invalid input)", 20, -3, "below_one", 0},
		{"the smallest integer", "FR-004", "unhappy (invalid input)", 20, math.MinInt, "below_one", 0},
		{"maximum below 1", "FR-006, table 4.1; SRD 5.2.1, \"Hit Points\", p. 183", "unhappy (boundary)", 0, 5, "maximum_below_one", 0},
		{"maximum of 1 and a value of 1", "FR-006, FR-003", "unhappy (boundary)", 1, 1, "applied", 1},
		{"maximum of 1 and a larger value", "FR-005", "unhappy (boundary)", 1, 9, "applied", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/returning-to-life.md (the case's requirement)
			cmd := rtlCommand(t, c.value)
			cause := combat.DeathSavingThrowFailures
			if c.maximum == 0 {
				cause = combat.HitPointMaximumZero
			}
			cmd.Status = status(t, 0, c.maximum, 0, dead(cause), combat.FallsUnconscious, combat.NoAffinities())
			res := combat.Revive(cmd)
			var got string
			switch res.(type) {
			case combat.ReviveCreatureApplied:
				got = "applied"
			case combat.ReviveCreatureHitPointsBelowOne:
				got = "below_one"
			case combat.ReviveCreatureMaximumBelowOne:
				got = "maximum_below_one"
			default:
				got = "other"
			}
			if got != c.want {
				t.Fatalf("Revive(max %d, value %d) = %T, want %s [%s; %s]", c.maximum, c.value, res, c.want, c.requirement, c.path)
			}
			if got == "applied" {
				if _, ok := res.Status().LifeState().(combat.Alive); !ok || rtlCurrent(res.Status()) != c.current || res.Status().HitPoints().Maximum() != c.maximum {
					t.Errorf("status = %+v / %T, want Alive at %d of %d [%s]", res.Status().HitPoints(), res.Status().LifeState(), c.current, c.maximum, c.requirement)
				}
				return
			}
			if !reflect.DeepEqual(res.Status(), cmd.Status) || !res.Conditions().Equal(cmd.Conditions) || !res.Effects().Equal(cmd.Effects) {
				t.Errorf("a refusal changed the creature's state [%s]", c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/returning-to-life.md FR-001 - a creature that is Alive, Dying or Stable (or the zero status, or a zero subject) is not Dead: ReviveCreatureNotDead, status, conditions and effects unchanged (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (state error).
func TestReviveNotDead(t *testing.T) {
	t.Parallel()
	stable := status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities())
	cases := map[string]combat.HitPointStatus{"alive": pcStatus(t, 12, 20, 0), "dying": dyingStatus(t, 20, 1, 1), "stable": stable, "zero status": {}}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := rtlCommand(t, 5)
			cmd.Status = s
			res, ok := combat.Revive(cmd).(combat.ReviveCreatureNotDead)
			if !ok || !reflect.DeepEqual(res.Status(), s) || !res.Conditions().Equal(cmd.Conditions) || !res.Effects().Equal(cmd.Effects) {
				t.Errorf("Revive(%s) = %#v, want ReviveCreatureNotDead with the state unchanged", name, res)
			}
		})
	}
	cmd := rtlCommand(t, 5)
	cmd.Subject = ""
	if _, ok := combat.Revive(cmd).(combat.ReviveCreatureNotDead); !ok {
		t.Errorf("a zero subject must give ReviveCreatureNotDead, got %T", combat.Revive(cmd))
	}
}

// Requirement: doc/specs/returning-to-life.md FR-002 - SpiritRefuses (and a consent that is not SpiritAccepts, the zero value or an unknown number) gives ReviveCreatureRefused with the state unchanged and no event (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (rule interaction).
func TestReviveSpiritRefuses(t *testing.T) {
	t.Parallel()
	for name, consent := range map[string]combat.RevivalConsent{"refuses": combat.SpiritRefuses, "zero": 0, "unknown": 99, "negative": -1} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cmd := rtlCommand(t, 5)
			cmd.Consent = consent
			res, ok := combat.Revive(cmd).(combat.ReviveCreatureRefused)
			if !ok || !reflect.DeepEqual(res.Status(), cmd.Status) || !res.Conditions().Equal(cmd.Conditions) || !res.Effects().Equal(cmd.Effects) {
				t.Errorf("Revive(consent %v) = %#v, want ReviveCreatureRefused with the state unchanged", consent, res)
			}
		})
	}
	// A refusal is decided before the Hit Point checks.
	cmd := rtlCommand(t, 0)
	cmd.Consent = combat.SpiritRefuses
	if _, ok := combat.Revive(cmd).(combat.ReviveCreatureRefused); !ok {
		t.Errorf("a refusing spirit with 0 Hit Points = %T, want ReviveCreatureRefused", combat.Revive(cmd))
	}
	// A creature that is not Dead is decided before the consent.
	cmd = rtlCommand(t, 5)
	cmd.Status, cmd.Consent = pcStatus(t, 3, 20, 0), combat.SpiritRefuses
	if _, ok := combat.Revive(cmd).(combat.ReviveCreatureNotDead); !ok {
		t.Errorf("a living creature and a refusing spirit = %T, want ReviveCreatureNotDead", combat.Revive(cmd))
	}
}

// Requirement: doc/specs/returning-to-life.md FR-007, FR-006 - revival is possible after every DeathCause; HitPointMaximumZero is rejected only while the maximum is still below 1 (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (rule interaction).
func TestReviveAfterEveryDeathCause(t *testing.T) {
	t.Parallel()
	for _, cause := range []combat.DeathCause{combat.MonsterDeath, combat.HitPointMaximumZero, combat.MassiveDamage, combat.DeathSavingThrowFailures, combat.Exhaustion} {
		t.Run(cause.String(), func(t *testing.T) {
			t.Parallel()
			cmd := rtlCommand(t, 4)
			cmd.Status = status(t, 0, 20, 0, dead(cause), combat.FallsUnconscious, combat.NoAffinities())
			res := rtlApplied(t, cmd)
			if rtlCurrent(res.Status()) != 4 {
				t.Errorf("after %v: current = %d, want 4", cause, rtlCurrent(res.Status()))
			}
		})
	}
	cmd := rtlCommand(t, 4)
	cmd.Status = status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
	if _, ok := combat.Revive(cmd).(combat.ReviveCreatureMaximumBelowOne); !ok {
		t.Errorf("HitPointMaximumZero with maximum 0 = %T, want ReviveCreatureMaximumBelowOne", combat.Revive(cmd))
	}
}

// Requirement: doc/specs/returning-to-life.md FR-008 - the returned status keeps the ZeroHitPointsRule and the DamageAffinities and has 0 Temporary Hit Points; it is Alive (SRD 5.2.1, "Dead", p. 180; hit-points-and-damage.md FR-087).
// Path: happy.
func TestReviveKeepsRuleAndAffinitiesAndClearsTemporaryHitPoints(t *testing.T) {
	t.Parallel()
	aff := affinities(t, nil, damageTypes(content.DamageFire), damageTypes(content.DamagePoison))
	cmd := rtlCommand(t, 6)
	cmd.Status = status(t, 0, 20, 5, dead(combat.MassiveDamage), combat.DiesInstantly, aff)
	got := rtlApplied(t, cmd).Status()
	if got.ZeroHitPointsRule() != combat.DiesInstantly || !reflect.DeepEqual(got.Affinities(), aff) {
		t.Errorf("rule %v / affinities %+v, want DiesInstantly and %+v", got.ZeroHitPointsRule(), got.Affinities(), aff)
	}
	if got.HitPoints().Temporary() != 0 || rtlCurrent(got) != 6 {
		t.Errorf("Hit Points = %+v, want 6 current and 0 Temporary", got.HitPoints())
	}
	if _, ok := got.LifeState().(combat.Alive); !ok {
		t.Errorf("life state = %T, want Alive", got.LifeState())
	}
}

// Requirement: per case
// Path: per case.
func TestReviveWhatReturns(t *testing.T) {
	t.Parallel()
	P, B, C, Par, Inc := content.ConditionPoisoned, content.ConditionBlinded, content.ConditionCharmed, content.ConditionParalyzed, content.ConditionIncapacitated
	cases := []struct {
		name, requirement, path string
		set                     combat.ConditionSet
		elapsedC                []content.ConditionKind
		elapsedE                []string
		wantC                   []content.ConditionKind
		wantE                   []string
	}{
		{"all ongoing", "FR-009, FR-010, table 4.2", "happy", meConditions(combat.Poisoned{}, combat.Blinded{}), nil, nil,
			[]content.ConditionKind{B, P}, []string{"c1", "c2", "s1"}},
		{"a condition lapsed", "FR-009, table 4.2", "unhappy (rule interaction)", meConditions(combat.Poisoned{}, combat.Blinded{}), []content.ConditionKind{P}, nil,
			[]content.ConditionKind{B}, []string{"c1", "c2", "s1"}},
		{"a curse lapsed", "FR-010, table 4.2", "unhappy (rule interaction)", meConditions(combat.Poisoned{}, combat.Blinded{}), nil, []string{"c2"},
			[]content.ConditionKind{B, P}, []string{"c1", "s1"}},
		{"a contagion lapsed", "FR-010, table 4.2", "unhappy (rule interaction)", meConditions(combat.Poisoned{}, combat.Blinded{}), nil, []string{"c1"},
			[]content.ConditionKind{B, P}, []string{"c2", "s1"}},
		{"a spell lapsed", "FR-010 (other categories follow the same rule)", "unhappy (rule interaction)", meConditions(combat.Poisoned{}, combat.Blinded{}), nil, []string{"s1"},
			[]content.ConditionKind{B, P}, []string{"c1", "c2"}},
		{"everything lapsed", "FR-009, FR-010, table 4.2", "unhappy (boundary)", meConditions(combat.Poisoned{}, combat.Blinded{}), []content.ConditionKind{P, B}, []string{"c1", "c2", "s1"},
			nil, nil},
		{"elapsed names an absent condition", "FR-015, table 4.2", "unhappy (invalid input)", meConditions(combat.Poisoned{}, combat.Blinded{}), []content.ConditionKind{C}, nil,
			[]content.ConditionKind{B, P}, []string{"c1", "c2", "s1"}},
		{"elapsed names an absent effect", "FR-015", "unhappy (invalid input)", meConditions(combat.Poisoned{}, combat.Blinded{}), nil, []string{"zz"},
			[]content.ConditionKind{B, P}, []string{"c1", "c2", "s1"}},
		{"elapsed names an implied-only kind", "FR-015, table 4.2; conditions.md FR-033", "unhappy (rule interaction)", meConditions(combat.Paralyzed{}), []content.ConditionKind{Inc}, nil,
			[]content.ConditionKind{Inc, Par}, []string{"c1", "c2", "s1"}},
		{"elapsed names an invalid kind", "FR-015", "unhappy (invalid input)", meConditions(combat.Poisoned{}), []content.ConditionKind{0, 99}, nil,
			[]content.ConditionKind{P}, []string{"c1", "c2", "s1"}},
		{"the same kind named twice", "FR-009", "unhappy (invalid input)", meConditions(combat.Poisoned{}, combat.Blinded{}), []content.ConditionKind{P, P}, nil,
			[]content.ConditionKind{B}, []string{"c1", "c2", "s1"}},
		{"elapsed names Exhaustion", "FR-015, FR-011, table 4.2", "unhappy (rule interaction)", meConditions(condLevel(3)), []content.ConditionKind{content.ConditionExhaustion}, nil,
			[]content.ConditionKind{content.ConditionExhaustion}, []string{"c1", "c2", "s1"}},
		{"Unconscious lapsed leaves the creature Prone", "FR-009; conditions.md FR-031", "unhappy (rule interaction)", meConditions(combat.Unconscious{}), []content.ConditionKind{content.ConditionUnconscious}, nil,
			[]content.ConditionKind{content.ConditionProne}, []string{"c1", "c2", "s1"}},
		{"Prone and Unconscious both lapsed, Prone named first", "FR-009; conditions.md FR-031", "unhappy (rule interaction)", meConditions(combat.Prone{}, combat.Unconscious{}),
			[]content.ConditionKind{content.ConditionProne, content.ConditionUnconscious}, nil, nil, []string{"c1", "c2", "s1"}},
		{"nothing held, empty inputs", "FR-009, table 4.2", "unhappy (empty and minimal)", combat.ConditionSet{}, []content.ConditionKind{}, []string{},
			nil, []string{"c1", "c2", "s1"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/returning-to-life.md (the case's requirement); SRD 5.2.1, "Dead", p. 180
			cmd := rtlCommand(t, 5)
			cmd.Conditions, cmd.ElapsedConditions, cmd.ElapsedEffects = c.set, c.elapsedC, rtlIDs(c.elapsedE...)
			res := rtlApplied(t, cmd)
			gotC := res.Conditions().Effective()
			if !slices.Equal(gotC, c.wantC) && (len(gotC) != 0 || len(c.wantC) != 0) {
				t.Errorf("conditions = %v, want %v [%s; %s]", gotC, c.wantC, c.requirement, c.path)
			}
			var gotE []string
			for _, e := range res.Effects().All() {
				gotE = append(gotE, e.ID().String())
			}
			wantE := c.wantE
			if c.name == "everything lapsed" {
				wantE = nil
			}
			if !slices.Equal(gotE, wantE) {
				t.Errorf("effects = %v, want %v [%s; %s]", gotE, wantE, c.requirement, c.path)
			}
		})
	}
	// An empty creature (no effects at all) returns with none.
	cmd := rtlCommand(t, 5)
	cmd.Effects = combat.OngoingEffects{}
	if res := rtlApplied(t, cmd); res.Effects().Len() != 0 {
		t.Errorf("effects = %d, want none", res.Effects().Len())
	}
}

// Requirement: per case
// Path: per case.
func TestReviveExhaustion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level, want             int
	}{
		{"none", "FR-012, table 4.3", "unhappy (empty and minimal)", 0, 0},
		{"level 1", "FR-011, table 4.3", "unhappy (boundary)", 1, 0},
		{"level 2", "FR-011, table 4.3", "happy", 2, 1},
		{"level 5", "FR-011, table 4.3", "happy", 5, 4},
		{"level 6, died of Exhaustion", "FR-011, FR-007, table 4.3", "unhappy (boundary)", 6, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/returning-to-life.md (the case's requirement); SRD 5.2.1, "Dead", p. 180; "Exhaustion", p. 181
			cmd := rtlCommand(t, 5)
			cmd.Conditions = combat.ConditionSet{}
			if c.level > 0 {
				cmd.Conditions = meConditions(condLevel(c.level))
			}
			if c.level == 6 {
				cmd.Status = status(t, 0, 20, 0, dead(combat.Exhaustion), combat.FallsUnconscious, combat.NoAffinities())
			}
			res := rtlApplied(t, cmd)
			got := 0
			for _, cond := range res.Conditions().Explicit() {
				if e, ok := cond.(combat.ExhaustionCondition); ok {
					got = e.Level.Value()
				}
			}
			if got != c.want {
				t.Errorf("Exhaustion after level %d = %d, want %d [%s; %s]", c.level, got, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/returning-to-life.md FR-013 - one CreatureReturnedToLife with the creature and its new Hit Points comes first, then the ConditionRemoved of each condition dropped, then the Exhaustion event (ConditionLevelReduced above level 1, ConditionRemoved at level 1); a revival with nothing dropped emits only the first (SRD 5.2.1, "Dead", p. 180).
// Path: happy.
func TestReviveEventOrder(t *testing.T) {
	t.Parallel()
	cmd := rtlCommand(t, 25)
	cmd.Conditions = meConditions(combat.Poisoned{}, combat.Blinded{}, condLevel(3))
	cmd.ElapsedConditions = []content.ConditionKind{content.ConditionPoisoned, content.ConditionBlinded}
	events := rtlApplied(t, cmd).Events()
	if want := []string{"creature_returned_to_life", "condition_removed", "condition_removed", "condition_level_reduced"}; !slices.Equal(rtlKinds(events), want) {
		t.Fatalf("events = %v, want %v", rtlKinds(events), want)
	}
	combattest.CheckEvent(t, events[0], "creature_returned_to_life", gid("hero"))
	if e := events[0].(combat.CreatureReturnedToLife); e.HitPoints() != 20 {
		t.Errorf("event Hit Points = %d, want 20 (capped)", e.HitPoints())
	}
	for i, k := range []content.ConditionKind{content.ConditionBlinded, content.ConditionPoisoned} {
		if events[i+1].(combat.ConditionRemoved).ConditionKind() != k {
			t.Errorf("event %d removed %v, want %v", i+1, events[i+1].(combat.ConditionRemoved).ConditionKind(), k)
		}
	}
	if lvl := events[3].(combat.ConditionLevelReduced).Level().Value(); lvl != 2 {
		t.Errorf("reduced to level %d, want 2", lvl)
	}
	for _, e := range events {
		combattest.CheckEvent(t, e, e.Kind(), gid("hero"))
	}

	cmd.Conditions = meConditions(condLevel(1))
	cmd.ElapsedConditions = nil
	events = rtlApplied(t, cmd).Events()
	if want := []string{"creature_returned_to_life", "condition_removed"}; !slices.Equal(rtlKinds(events), want) ||
		events[1].(combat.ConditionRemoved).ConditionKind() != content.ConditionExhaustion {
		t.Errorf("level 1 events = %v, want %v", rtlKinds(events), want)
	}

	cmd.Conditions = combat.ConditionSet{}
	if got := rtlKinds(rtlApplied(t, cmd).Events()); !slices.Equal(got, []string{"creature_returned_to_life"}) {
		t.Errorf("events with nothing dropped = %v, want only the return", got)
	}
}

// Requirement: doc/specs/returning-to-life.md FR-009, conditions.md FR-031 - removing a lapsed Unconscious condition leaves the creature Prone with no event for it, and a lapsed Prone named alongside it is removed after it.
// Path: unhappy (rule interaction).
func TestReviveDropsUnconsciousBeforeProne(t *testing.T) {
	t.Parallel()
	cmd := rtlCommand(t, 5)
	cmd.Conditions = meConditions(combat.Prone{}, combat.Unconscious{})
	cmd.ElapsedConditions = []content.ConditionKind{content.ConditionProne, content.ConditionUnconscious}
	events := rtlApplied(t, cmd).Events()
	if want := []string{"creature_returned_to_life", "condition_removed", "condition_removed"}; !slices.Equal(rtlKinds(events), want) {
		t.Fatalf("events = %v, want %v", rtlKinds(events), want)
	}
	if first := events[1].(combat.ConditionRemoved).ConditionKind(); first != content.ConditionUnconscious {
		t.Errorf("first condition dropped = %v, want Unconscious", first)
	}
}

// Requirement: doc/specs/returning-to-life.md FR-014 - Revive does not alter its inputs, returns copied slices and gives equal results for equal inputs (SRD 5.2.1, "Dead", p. 180).
// Path: unhappy (immutability).
func TestReviveImmutabilityAndDeterminism(t *testing.T) {
	t.Parallel()
	cmd := rtlCommand(t, 9)
	cmd.Conditions = meConditions(combat.Poisoned{}, condLevel(2))
	cmd.ElapsedConditions = []content.ConditionKind{content.ConditionPoisoned}
	cmd.ElapsedEffects = rtlIDs("c1")
	before := cmd
	before.ElapsedConditions, before.ElapsedEffects = slices.Clone(cmd.ElapsedConditions), slices.Clone(cmd.ElapsedEffects)
	first, second := rtlApplied(t, cmd), rtlApplied(t, cmd)
	if !reflect.DeepEqual(cmd, before) {
		t.Errorf("Revive changed its command: %+v, was %+v", cmd, before)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("equal inputs gave different results: %+v and %+v", first, second)
	}
	list := first.Events()
	list[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events returned a view of the result's own list")
	}
	effects := first.Effects().All()
	effects[0] = combat.OngoingEffect{}
	if first.Effects().All()[0].IsZero() {
		t.Error("Effects().All returned a view of the result's own list")
	}
	// The input effects keep what the host reported elapsed.
	if cmd.Effects.Len() != 3 {
		t.Errorf("the input effects now hold %d, want 3", cmd.Effects.Len())
	}
}

// Requirement: doc/specs/returning-to-life.md section 5 - ReviveCreatureResult is a closed set of five variants; every variant reports the status, conditions and effects, and RevivalConsent names its values.
// Path: happy.
func TestReviveResultVariants(t *testing.T) {
	t.Parallel()
	good := rtlCommand(t, 3)
	refuses, living, zeroHP, noMax := good, good, rtlCommand(t, 0), good
	refuses.Consent = combat.SpiritRefuses
	living.Status = pcStatus(t, 4, 20, 0)
	noMax.Status = status(t, 0, 0, 0, dead(combat.HitPointMaximumZero), combat.FallsUnconscious, combat.NoAffinities())
	for name, cmd := range map[string]combat.ReviveCreatureCommand{"applied": good, "refused": refuses, "not dead": living, "below one": zeroHP, "maximum": noMax} {
		res := combat.Revive(cmd)
		switch res.(type) {
		case combat.ReviveCreatureApplied, combat.ReviveCreatureRefused, combat.ReviveCreatureNotDead, combat.ReviveCreatureHitPointsBelowOne, combat.ReviveCreatureMaximumBelowOne:
		default:
			t.Errorf("%s: unexpected variant %T", name, res)
		}
		if res.Status().IsZero() || res.Effects().Len() != cmd.Effects.Len() {
			t.Errorf("%s: the result lost the creature's state", name)
		}
	}
	for c, want := range map[combat.RevivalConsent]string{combat.SpiritAccepts: "accepts", combat.SpiritRefuses: "refuses", 0: "unknown", 9: "unknown"} {
		if c.String() != want {
			t.Errorf("RevivalConsent(%d).String() = %q, want %q", int(c), c.String(), want)
		}
	}
}

// FuzzRevive feeds Revive host numbers and kinds. It asserts properties: it never panics, an applied result is Alive
// at 1 to the maximum with 0 Temporary Hit Points, a refusal changes nothing, and equal inputs give equal results.
func FuzzRevive(f *testing.F) {
	f.Add(7, 0, int8(1), int8(2), 5, "c1")
	f.Add(0, 0, int8(0), int8(0), 0, "")
	f.Add(math.MaxInt, 6, int8(2), int8(99), math.MaxInt, "zz")
	f.Add(math.MinInt, 1, int8(-1), int8(-1), math.MinInt, " ")
	f.Fuzz(func(t *testing.T, hitPoints, level int, consent, kind int8, maximum int, effect string) {
		cause := combat.DeathCause(int(kind)%5 + 1)
		if !cause.Valid() {
			cause = combat.MassiveDamage
		}
		if maximum < 0 {
			maximum = 0
		}
		if maximum == 0 {
			cause = combat.HitPointMaximumZero
		}
		hp, err := combat.NewHitPoints(0, maximum, 0)
		if err != nil {
			t.Skip()
		}
		st, err := combat.RestoreHitPointStatus(hp, dead(cause), combat.FallsUnconscious, combat.NoAffinities())
		if err != nil {
			t.Skip()
		}
		set := combat.ConditionSet{}
		if level >= 1 && level <= 6 {
			set = meConditions(condLevel(level))
		}
		cmd := combat.ReviveCreatureCommand{Subject: gid("hero"), Status: st, HitPoints: hitPoints, Consent: combat.RevivalConsent(consent), Conditions: set,
			Effects: rtlEffects(), ElapsedConditions: []content.ConditionKind{content.ConditionKind(kind)}, ElapsedEffects: []combat.OngoingEffectID{combat.OngoingEffectID(effect)}}
		res := combat.Revive(cmd)
		if !reflect.DeepEqual(res, combat.Revive(cmd)) {
			t.Fatal("equal inputs gave different results")
		}
		if _, ok := res.(combat.ReviveCreatureApplied); ok {
			s := res.Status()
			if _, alive := s.LifeState().(combat.Alive); !alive || s.HitPoints().Current() < 1 || s.HitPoints().Current() > maximum || s.HitPoints().Temporary() != 0 {
				t.Fatalf("applied status = %+v / %T", s.HitPoints(), s.LifeState())
			}
			return
		}
		if !reflect.DeepEqual(res.Status(), st) {
			t.Fatalf("a refusal changed the status: %T", res)
		}
	})
}
