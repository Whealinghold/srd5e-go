package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func encSlash(n int) []combat.Damage {
	return []combat.Damage{combat.MustDamage(n, content.DamageSlashing)}
}

func dyingT(tb testing.TB, cur, max, temp, successes, failures int) combat.HitPointStatus {
	tb.Helper()
	d, err := combat.NewDying(successes, failures)
	if err != nil {
		tb.Fatal(err)
	}
	return status(tb, cur, max, temp, d, combat.FallsUnconscious, combat.NoAffinities())
}

// Requirement: per case
// Path: per case.
func TestApplyDamageCases(t *testing.T) {
	t.Parallel()
	pc := func(cur, max, temp int) combat.HitPointStatus { return pcStatus(t, cur, max, temp) }
	monster := func(cur, max int, rule combat.ZeroHitPointsRule) combat.HitPointStatus {
		return status(t, cur, max, 0, combat.Alive{}, rule, combat.NoAffinities())
	}
	stable := status(t, 0, 20, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities())
	aff, err := combat.NewDamageAffinities([]combat.DamageScope{combat.AllDamage{}}, []content.DamageType{content.DamageFire}, nil)
	if err != nil {
		t.Fatal(err)
	}
	immunePoison := affinities(t, nil, nil, damageTypes(content.DamagePoison))
	type row struct {
		name, requirement, path string
		before                  combat.Combatant
		damages                 []combat.Damage
		adj                     []combat.DamageAdjustment
		delivery                combat.DamageDelivery
		hit                     combat.HitKind
		kinds                   []string
		cur, temp               int
		life                    string
		unconscious, knocked    bool
	}
	a := func(hp combat.HitPointStatus, conds ...combat.Condition) combat.Combatant {
		return encC(t, "a", combat.SideA, hp, conds...)
	}
	u := condConds(combat.Unconscious{})
	rows := []row{
		{"ordinary damage", "FR-021; p. 16", "happy", a(pc(10, 20, 0)), encSlash(5), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken"}, 5, 0, "alive", false, false},
		{"exactly to 0, character", "FR-023; p. 17", "unhappy (boundary)", a(pc(6, 12, 0)), encSlash(6), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied"}, 0, 0, "dying", true, false},
		{"overkill below the maximum", "FR-023; p. 17", "unhappy (boundary)", a(pc(6, 12, 0)), encSlash(17), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied"}, 0, 0, "dying", true, false},
		{"massive damage, remainder equals the maximum", "FR-025; p. 17", "unhappy (boundary)", a(pc(6, 12, 0)), encSlash(18), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died"}, 0, 0, "dead:massive_damage", false, false},
		{"massive damage, remainder above the maximum", "FR-025; p. 17", "unhappy (boundary)", a(pc(6, 12, 0)), encSlash(19), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died"}, 0, 0, "dead:massive_damage", false, false},
		{"monster with DiesInstantly to 0", "FR-024; p. 17", "happy", a(monster(7, 7, combat.DiesInstantly)), encSlash(7), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died"}, 0, 0, "dead:monster_death", false, false},
		{"monster treated like a character", "FR-024; p. 17", "unhappy (rule interaction)", a(monster(7, 7, combat.FallsUnconscious)), encSlash(7), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "became_bloodied", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied"}, 0, 0, "dying", true, false},
		{"Temporary Hit Points absorb part", "FR-022; p. 18", "happy", a(pc(10, 20, 5)), encSlash(7), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "temporary_hit_points_ended"}, 8, 0, "alive", false, false},
		{"Temporary Hit Points absorb all", "FR-091; p. 18", "unhappy (boundary)", a(pc(10, 20, 5)), encSlash(5), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "temporary_hit_points_ended"}, 10, 0, "alive", false, false},
		{"damage below the Temporary Hit Points", "FR-091; p. 18", "unhappy (boundary)", a(pc(10, 20, 5)), encSlash(3), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken"}, 10, 2, "alive", false, false},
		{"absorbed while Dying at 0", "FR-091; p. 18", "unhappy (rule interaction)", a(dyingT(t, 0, 20, 5, 0, 0), combat.Unconscious{}), encSlash(3), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken"}, 0, 2, "dying", true, false},
		{"Resistance to all, Vulnerability to Fire, aura", "FR-022; p. 17", "unhappy (rule interaction)",
			a(status(t, 40, 40, 0, combat.Alive{}, combat.FallsUnconscious, aff)), []combat.Damage{combat.MustDamage(28, content.DamageFire)},
			[]combat.DamageAdjustment{combat.NewFlatAdjustment(-5, nil)}, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "became_bloodied"}, 18, 0, "alive", false, false},
		{"zero damage", "FR-028; p. 16", "unhappy (boundary)", a(pc(10, 20, 0)), encSlash(0), nil, combat.OtherSource, combat.NormalHit,
			[]string{}, 10, 0, "alive", false, false},
		{"Immune damage type", "FR-028; p. 17", "unhappy (boundary)", a(status(t, 12, 12, 0, combat.Alive{}, combat.FallsUnconscious, immunePoison)),
			[]combat.Damage{combat.MustDamage(9, content.DamagePoison)}, nil, combat.OtherSource, combat.NormalHit,
			[]string{}, 12, 0, "alive", false, false},
		{"Unconscious from a spell, then dropped to 0", "FR-027; p. 179", "unhappy (rule interaction)", a(pc(5, 20, 0), combat.Unconscious{}), encSlash(5), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious"}, 0, 0, "dying", true, false},
		{"damage at 0, already Unconscious", "FR-026; p. 18", "unhappy (rule interaction)", a(dyingT(t, 0, 20, 0, 0, 0), combat.Unconscious{}), encSlash(1), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken"}, 0, 0, "dying", true, false},
		{"third failure by damage", "FR-026; p. 18", "unhappy (rule interaction)", a(dyingT(t, 0, 20, 0, 0, 2), combat.Unconscious{}), encSlash(1), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "died"}, 0, 0, "dead:death_saving_throw_failures", true, false},
		{"critical hit at 0", "FR-026; p. 18", "unhappy (rule interaction)", a(dyingT(t, 0, 20, 0, 0, 0), combat.Unconscious{}), encSlash(1), nil, combat.OtherSource, combat.CriticalHit,
			[]string{"damage_taken"}, 0, 0, "dying", true, false},
		{"damage to a Stable creature", "FR-072; p. 18", "unhappy (rule interaction)", a(stable, combat.Unconscious{}), encSlash(3), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "lost_stability"}, 0, 0, "dying", true, false},
		{"knock-out", "FR-070; p. 17", "unhappy (rule interaction)", a(pc(6, 12, 0)), encSlash(6), nil, combat.MeleeAttackKnockOut, combat.NormalHit,
			[]string{"damage_taken", "knocked_out", "condition_applied"}, 1, 0, "alive", true, true},
		{"knock-out requested, does not reach 0", "FR-070; p. 17", "unhappy (boundary)", a(pc(10, 12, 0)), encSlash(3), nil, combat.MeleeAttackKnockOut, combat.NormalHit,
			[]string{"damage_taken"}, 7, 0, "alive", false, false},
		{"knock-out of a creature that already holds Unconscious", "FR-067, FR-098; p. 179", "unhappy (rule interaction)", a(pc(6, 12, 0), combat.Unconscious{}), encSlash(6), nil, combat.MeleeAttackKnockOut, combat.NormalHit,
			[]string{"damage_taken", "knocked_out"}, 1, 0, "alive", true, false},
		{"Immune to Unconscious, dropped to 0", "FR-067; p. 17", "unhappy (rule interaction)", encC(t, "a", combat.SideA, pc(6, 12, 0)), encSlash(6), nil, combat.OtherSource, combat.NormalHit,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious"}, 0, 0, "dying", false, false},
	}
	// The row above needs an Unconscious Immunity: build it explicitly.
	immuneSet, err := combat.NewConditionSet(nil, []content.ConditionKind{content.ConditionUnconscious})
	if err != nil {
		t.Fatal(err)
	}
	immune, err := combat.NewCombatant(combat.CombatantRef{ID: cid("a"), Kind: combat.MonsterCombatant{}}, combat.SideA, pc(6, 12, 0), immuneSet)
	if err != nil {
		t.Fatal(err)
	}
	rows[len(rows)-1].before = immune
	_ = u
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			e := encWith(t, r.before)
			snapshot := encWith(t, r.before)
			tr, err := e.ApplyDamage(cid("a"), r.damages, r.adj, r.delivery, r.hit)
			after, events := encOK(t, tr, err)
			if !slices.Equal(encKinds(events), r.kinds) {
				t.Errorf("events = %v, want %v [%s]", encKinds(events), r.kinds, r.requirement)
			}
			c := encMember(t, after, "a")
			hp := c.HitPoints().HitPoints()
			if hp.Current() != r.cur || hp.Temporary() != r.temp || encLife(c) != r.life {
				t.Errorf("after: %d (temporary %d) %s, want %d (%d) %s [%s]", hp.Current(), hp.Temporary(), encLife(c), r.cur, r.temp, r.life, r.requirement)
			}
			if got := encExplicit(c, content.ConditionUnconscious); got != r.unconscious || c.KnockedOut() != r.knocked {
				t.Errorf("Unconscious %v knocked out %v, want %v %v [%s]", got, c.KnockedOut(), r.unconscious, r.knocked, r.requirement)
			}
			if n := countExplicit(c, content.ConditionUnconscious); n > 1 {
				t.Errorf("Unconscious held %d times [FR-027]", n)
			}
			if len(tr.Resolutions()) != len(r.damages) {
				t.Errorf("resolutions = %d, want %d [FR-022]", len(tr.Resolutions()), len(r.damages))
			}
			if !e.Equal(snapshot) {
				t.Errorf("the receiver changed [FR-049]")
			}
			encOthersEqualCombatants2(t, e, after, "a")
			checkEnvelopes(t, events)
		})
	}
}

func countExplicit(c combat.Combatant, k content.ConditionKind) int {
	n := 0
	for _, x := range c.Conditions().Explicit() {
		if x.Kind() == k {
			n++
		}
	}
	return n
}

// encOthersEqualCombatants2 asserts every combatant but id is equal and the order is the same.
func encOthersEqualCombatants2(tb testing.TB, before, after combat.Encounter, id string) {
	tb.Helper()
	encOthersEqual(tb, before, after, id)
}

// checkEnvelopes asserts every event is an envelope carrying the encounter id and a round.
func checkEnvelopes(tb testing.TB, events []combat.Event) {
	tb.Helper()
	for _, ev := range events {
		e, ok := ev.(combat.EncounterEvent)
		if !ok || e.EncounterID() != eid() || e.Round() < 1 {
			tb.Errorf("event %T does not carry the encounter id and a round [FR-051]", ev)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-022, FR-021 - the breakdown of the damage reaches the host through the resolutions, field by field, from the hit point rules without being recomputed.
// Path: unhappy (rule interaction).
func TestApplyDamageResolutionsFieldByField(t *testing.T) {
	t.Parallel()
	aff, err := combat.NewDamageAffinities([]combat.DamageScope{combat.AllDamage{}}, []content.DamageType{content.DamageFire}, nil)
	if err != nil {
		t.Fatal(err)
	}
	e := encWith(t, encC(t, "a", combat.SideA, status(t, 40, 40, 0, combat.Alive{}, combat.FallsUnconscious, aff)))
	tr, err := e.ApplyDamage(cid("a"), []combat.Damage{combat.MustDamage(28, content.DamageFire)}, []combat.DamageAdjustment{combat.NewFlatAdjustment(-5, nil)}, combat.OtherSource, combat.NormalHit)
	encOK(t, tr, err)
	res := tr.Resolutions()
	if len(res) != 1 || res[0].Incoming() != 28 || res[0].AfterAdjustments() != 23 || res[0].AfterResistance() != 11 || res[0].AfterVulnerability() != 22 || res[0].Immune() {
		t.Errorf("resolutions = %+v, want 28, 23, 11, 22 [FR-022; worked example p. 17]", res)
	}
	if got := tr.Resolutions(); len(got) > 0 {
		got[0] = combat.DamageResolution{}
		if tr.Resolutions()[0].Incoming() != 28 {
			t.Errorf("Resolutions is not a copy")
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-029, FR-033, FR-069, FR-079 - damage and healing for a dead combatant, dead of Exhaustion included, return CombatantIsDead and change nothing.
// Path: unhappy (state).
func TestDamageAndHealingForTheDeadAreRefused(t *testing.T) {
	t.Parallel()
	for _, hp := range []combat.HitPointStatus{deadStatus(t, combat.MassiveDamage), deadStatus(t, combat.Exhaustion)} {
		e := encWith(t, encC(t, "a", combat.SideA, hp))
		tr, err := e.ApplyDamage(cid("a"), encSlash(4), nil, combat.OtherSource, combat.NormalHit)
		if r := encNo(t, e, tr, err); r != (combat.CombatantIsDeadRejection{ID: cid("a")}) || r.String() != "combatant_is_dead" {
			t.Errorf("damage: rejection = %#v, want CombatantIsDead", r)
		}
		th, err := e.ApplyHealing(cid("a"), 10)
		if r := encNo(t, e, th, err); r != (combat.CombatantIsDeadRejection{ID: cid("a")}) {
			t.Errorf("healing: rejection = %#v, want CombatantIsDead", r)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-071 - each command field reaches the hit point rules unchanged: the transition equals the one of HitPointStatus.ApplyDamage for the same inputs, and invalid inputs are errors.
// Path: unhappy (invalid input).
func TestApplyDamagePassesItsInputsThroughAndRejectsInvalidOnes(t *testing.T) {
	t.Parallel()
	hp := pcStatus(t, 9, 20, 3)
	e := encWith(t, encC(t, "a", combat.SideA, hp))
	adj := []combat.DamageAdjustment{combat.NewFlatAdjustment(1, nil), combat.HalfDamage{}}
	want, werr := hp.ApplyDamage(cid("a"), encSlash(7), adj, combat.MeleeAttackWithoutKnockOut, combat.CriticalHit, nil)
	if werr != nil {
		t.Fatal(werr)
	}
	tr, err := e.ApplyDamage(cid("a"), encSlash(7), adj, combat.MeleeAttackWithoutKnockOut, combat.CriticalHit)
	after, _ := encOK(t, tr, err)
	if got := encMember(t, after, "a").HitPoints(); got != want.(combat.HitPointTransitionApplied).Status() {
		t.Errorf("status = %+v, want the hit point rules' %+v [FR-071]", got, want.(combat.HitPointTransitionApplied).Status())
	}
	for name, f := range map[string]func() (any, error){
		"no damage": func() (any, error) { return e.ApplyDamage(cid("a"), nil, nil, combat.OtherSource, combat.NormalHit) },
		"bad damage": func() (any, error) {
			return e.ApplyDamage(cid("a"), []combat.Damage{{}}, nil, combat.OtherSource, combat.NormalHit)
		},
		"nil adj": func() (any, error) {
			return e.ApplyDamage(cid("a"), encSlash(1), []combat.DamageAdjustment{nil}, combat.OtherSource, combat.NormalHit)
		},
		"delivery": func() (any, error) { return e.ApplyDamage(cid("a"), encSlash(1), nil, 0, combat.NormalHit) },
		"kind":     func() (any, error) { return e.ApplyDamage(cid("a"), encSlash(1), nil, combat.OtherSource, 0) },
		"zero id":  func() (any, error) { return e.ApplyDamage("", encSlash(1), nil, combat.OtherSource, combat.NormalHit) },
		"zero enc": func() (any, error) {
			return (combat.Encounter{}).ApplyDamage(cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
		},
	} {
		if res, err := f(); err == nil || res != nil && !isNilTransition(res) {
			t.Errorf("%s: want an ErrInvalidInput error, got %v, %v", name, res, err)
		}
	}
}

func isNilTransition(v any) bool { return v == nil }
