package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type damageRow struct {
	name, requirement, path string
	start                   combat.HitPointStatus
	damages                 []combat.Damage
	delivery                combat.DamageDelivery
	kind                    combat.HitKind
	current, temp           int
	life                    combat.LifeState
	events                  []string
}

// Requirement: per case
// Path: per case.
func TestApplyDamage(t *testing.T) {
	t.Parallel()
	const (
		fall   = combat.FallsUnconscious
		other  = combat.OtherSource
		normal = combat.NormalHit
		crit   = combat.CriticalHit
	)
	pc := func(cur, max, temp int) combat.HitPointStatus { return pcStatus(t, cur, max, temp) }
	monster := func(cur, max int) combat.HitPointStatus {
		return status(t, cur, max, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
	}
	stable := func(max int) combat.HitPointStatus {
		return status(t, 0, max, 0, combat.Stable{}, fall, combat.NoAffinities())
	}
	d := func(n int) []combat.Damage { return []combat.Damage{fire(n)} }
	two := func(a, b int) []combat.Damage {
		return []combat.Damage{fire(a), combat.MustDamage(b, content.DamageCold)}
	}
	hit, atZero, hpRule := `hit-points-and-damage.md table 4.6a; SRD 5.2.1, "Hit Points", p. 16`, `hit-points-and-damage.md table 4.7b; SRD 5.2.1, "Damage at 0 Hit Points", p. 18`, `hit-points-and-damage.md table 4.6a; SRD 5.2.1, "Massive Damage", p. 17`
	dmgTaken, tempEnded, bloodied, dropped, fell, died, ko, lostStab := "damage_taken", "temporary_hit_points_ended", "became_bloodied", "dropped_to_zero_hit_points", "fell_unconscious", "died", "knocked_out", "lost_stability"
	cases := []damageRow{
		{"ordinary hit", hit, "happy", pc(20, 20, 0), d(5), other, normal, 15, 0, combat.Alive{}, []string{dmgTaken}},
		{"becomes Bloodied", `FR-101; Glossary "Bloodied", p. 177`, "unhappy (rule interaction)", pc(20, 20, 0), d(10), other, normal, 10, 0, combat.Alive{}, []string{dmgTaken, bloodied}},
		{"already Bloodied", `FR-101`, "unhappy (rule interaction)", pc(10, 20, 0), d(1), other, normal, 9, 0, combat.Alive{}, []string{dmgTaken}},
		{"Temporary Hit Points never change the Bloodied judgement", `FR-101`, "unhappy (rule interaction)", pc(20, 20, 10), d(12), other, normal, 18, 0, combat.Alive{}, []string{dmgTaken, tempEnded}},
		{"exactly 0, no remainder", hit, "happy", pc(5, 20, 0), d(5), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},
		{"drop from not Bloodied emits Bloodied", `FR-049, table 4.9a`, "unhappy (rule interaction)", pc(11, 20, 0), d(11), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, bloodied, dropped, fell}},
		{"worked example, Massive Damage", hpRule, "unhappy (boundary)", pc(6, 12, 0), d(18), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, dropped, died}},
		{"remainder one below the maximum", hpRule, "unhappy (boundary)", pc(6, 12, 0), d(17), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},
		{"remainder 1", hpRule, "unhappy (boundary)", pc(6, 12, 0), d(7), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},
		{"temporary absorbs part", `table 4.6a; "Lose Temporary Hit Points First", p. 18`, "happy", pc(10, 20, 5), d(7), other, normal, 8, 0, combat.Alive{}, []string{dmgTaken, tempEnded}},
		{"damage equals temporary", `FR-084`, "unhappy (boundary)", pc(10, 20, 5), d(5), other, normal, 10, 0, combat.Alive{}, []string{dmgTaken, tempEnded}},
		{"damage below temporary", `FR-084`, "unhappy (boundary)", pc(10, 20, 5), d(4), other, normal, 10, 1, combat.Alive{}, []string{dmgTaken}},
		{"temporary then drop", hpRule, "unhappy (rule interaction)", pc(3, 20, 5), d(12), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, tempEnded, dropped, fell}},
		{"temporary then massive", hpRule, "unhappy (rule interaction)", pc(3, 10, 5), d(18), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, tempEnded, dropped, died}},
		{"minimal maximum", `table 4.6a; "Falling Unconscious", p. 17`, "unhappy (empty and minimal)", pc(1, 1, 0), d(1), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, bloodied, dropped, fell}},
		{"monster at 0", `FR-041; "Monster Death", p. 17`, "happy", monster(5, 20), d(5), other, normal, 0, 0, dead(combat.MonsterDeath), []string{dmgTaken, dropped, died}},
		{"monster treated like a character", `FR-042`, "unhappy (rule interaction)", pc(5, 20, 0), d(5), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},
		{"knock-out", `FR-050; "Knocking Out a Creature", p. 17`, "happy", pc(5, 20, 0), d(5), combat.MeleeAttackKnockOut, normal, 1, 0, combat.Alive{}, []string{dmgTaken, ko}},
		{"knock-out not needed", `FR-051`, "unhappy (boundary)", pc(5, 20, 0), d(4), combat.MeleeAttackKnockOut, normal, 1, 0, combat.Alive{}, []string{dmgTaken}},
		{"knock-out beats massive", `FR-050`, "unhappy (rule interaction)", pc(5, 20, 0), d(100), combat.MeleeAttackKnockOut, normal, 1, 0, combat.Alive{}, []string{dmgTaken, ko}},
		{"knock-out ignored at 0", `FR-051`, "unhappy (state error)", dyingStatus(t, 20, 0, 0), d(3), combat.MeleeAttackKnockOut, normal, 0, 0, dying(0, 1), []string{dmgTaken}},
		{"ranged does not knock out", `FR-052`, "unhappy (rule interaction)", pc(5, 20, 0), d(5), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},
		{"melee without knock-out", `FR-052`, "unhappy (rule interaction)", pc(5, 20, 0), d(5), combat.MeleeAttackWithoutKnockOut, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},
		{"knock-out after temporary absorbs part", `FR-050`, "unhappy (rule interaction)", pc(5, 20, 3), d(8), combat.MeleeAttackKnockOut, normal, 1, 0, combat.Alive{}, []string{dmgTaken, tempEnded, ko}},
		{"knock-out fully absorbed", `FR-050, FR-084`, "unhappy (rule interaction)", pc(5, 20, 10), d(8), combat.MeleeAttackKnockOut, normal, 5, 2, combat.Alive{}, []string{dmgTaken}},
		{"knock-out of a monster", `FR-050`, "unhappy (rule interaction)", monster(5, 20), d(5), combat.MeleeAttackKnockOut, normal, 1, 0, combat.Alive{}, []string{dmgTaken, ko}},
		{"knock-out into Bloodied", `FR-050, table 4.9a`, "unhappy (rule interaction)", pc(15, 20, 0), d(15), combat.MeleeAttackKnockOut, normal, 1, 0, combat.Alive{}, []string{dmgTaken, bloodied, ko}},
		{"two types summed, Massive Damage", `FR-085`, "unhappy (rule interaction)", pc(6, 12, 0), two(10, 8), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, dropped, died}},
		{"two types, no Massive Damage", `FR-085`, "unhappy (rule interaction)", pc(10, 20, 0), two(6, 5), other, normal, 0, 0, dying(0, 0), []string{dmgTaken, dropped, fell}},

		{"damage at 0: any damage", atZero, "happy", dyingStatus(t, 20, 0, 0), d(3), other, normal, 0, 0, dying(0, 1), []string{dmgTaken}},
		{"damage at 0: Critical Hit", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 0), d(3), other, crit, 0, 0, dying(0, 2), []string{dmgTaken}},
		{"damage at 0: Critical Hit at two failures", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 2), d(3), other, crit, 0, 0, dead(combat.DeathSavingThrowFailures), []string{dmgTaken, died}},
		{"damage at 0: third failure", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 2), d(1), other, normal, 0, 0, dead(combat.DeathSavingThrowFailures), []string{dmgTaken, died}},
		{"damage at 0: Critical Hit at one failure is the third", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 1), d(1), other, crit, 0, 0, dead(combat.DeathSavingThrowFailures), []string{dmgTaken, died}},
		{"damage at 0: one below the maximum", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 0), d(19), other, normal, 0, 0, dying(0, 1), []string{dmgTaken}},
		{"damage at 0: equals the maximum", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 0), d(20), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, died}},
		{"damage at 0: exceeds the maximum", atZero, "unhappy (boundary)", dyingStatus(t, 20, 0, 0), d(25), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, died}},
		{"damage at 0: successes kept", atZero, "unhappy (rule interaction)", dyingStatus(t, 20, 2, 0), d(3), other, normal, 0, 0, dying(2, 1), []string{dmgTaken}},
		{"damage at 0: fully absorbed by temporary", `FR-084`, "unhappy (rule interaction)", status(t, 0, 20, 5, dying(0, 0), fall, combat.NoAffinities()), d(3), other, normal, 0, 2, dying(0, 0), []string{dmgTaken}},
		{"damage at 0: partly absorbed", `FR-062`, "unhappy (rule interaction)", status(t, 0, 20, 5, dying(0, 0), fall, combat.NoAffinities()), d(8), other, normal, 0, 0, dying(0, 1), []string{dmgTaken, tempEnded}},
		{"damage at 0: Massive Damage uses the leftover", `FR-064`, "unhappy (boundary)", status(t, 0, 20, 5, dying(0, 0), fall, combat.NoAffinities()), d(24), other, normal, 0, 0, dying(0, 1), []string{dmgTaken, tempEnded}},
		{"damage at 0: Massive Damage after temporary", `FR-064`, "unhappy (boundary)", status(t, 0, 20, 5, dying(0, 0), fall, combat.NoAffinities()), d(25), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, tempEnded, died}},
		{"damage at 0: two types, Critical Hit counts once", `FR-085, FR-063`, "unhappy (rule interaction)", dyingStatus(t, 20, 0, 0), two(3, 4), other, crit, 0, 0, dying(0, 2), []string{dmgTaken}},
		{"damage at 0: two types, summed Massive Damage", `FR-085, FR-064`, "unhappy (rule interaction)", dyingStatus(t, 20, 0, 0), two(12, 8), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, died}},
		{"Stable creature takes damage", `FR-071`, "unhappy (rule interaction)", stable(20), d(3), other, normal, 0, 0, dying(0, 1), []string{dmgTaken, lostStab}},
		{"Stable creature takes a Critical Hit", `FR-071, FR-063`, "unhappy (rule interaction)", stable(20), d(3), other, crit, 0, 0, dying(0, 2), []string{dmgTaken, lostStab}},
		{"Stable creature takes Massive Damage", `FR-071, FR-064`, "unhappy (rule interaction)", stable(20), d(20), other, normal, 0, 0, dead(combat.MassiveDamage), []string{dmgTaken, lostStab, died}},
		{"Stable creature, damage fully absorbed", `FR-084`, "unhappy (rule interaction)", status(t, 0, 20, 5, combat.Stable{}, fall, combat.NoAffinities()), d(3), other, normal, 0, 2, combat.Stable{}, []string{dmgTaken}},
		{"zero damage changes nothing", `FR-048`, "unhappy (empty and minimal)", pc(10, 20, 4), d(0), other, normal, 10, 4, combat.Alive{}, []string{}},
		{"Immunity changes nothing", `FR-048; "Immunity", p. 17`, "unhappy (empty and minimal)",
			status(t, 0, 20, 0, dying(1, 1), fall, affinities(t, nil, nil, damageTypes(content.DamageFire))), d(9), other, crit, 0, 0, dying(1, 1), []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.start
			tr, err := c.start.ApplyDamage(subject(), c.damages, nil, c.delivery, c.kind, nil)
			a := applied(t, tr, err)
			hp := a.Status().HitPoints()
			if hp.Current() != c.current || hp.Temporary() != c.temp || hp.Maximum() != before.HitPoints().Maximum() || a.Status().LifeState() != c.life {
				t.Errorf("status = %+v %v, want current %d temp %d %v [%s]", hp, a.Status().LifeState(), c.current, c.temp, c.life, c.requirement)
			}
			if got := kinds(a.Events()); !slices.Equal(got, c.events) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, c.requirement)
			}
			if c.start != before {
				t.Errorf("input status changed")
			}
			for _, e := range a.Events() {
				if !slices.Equal(e.Combatants(), []combat.CombatantID{subject()}) {
					t.Errorf("%s combatants = %v, want [subject]", e.Kind(), e.Combatants())
				}
			}
			if dt, ok := firstEvent[combat.DamageTaken](a.Events()); ok && dt.Total() != dt.HitPointsLost()+dt.AbsorbedByTemporary() {
				t.Errorf("DamageTaken total %d != lost %d + absorbed %d", dt.Total(), dt.HitPointsLost(), dt.AbsorbedByTemporary())
			}
		})
	}
}

func firstEvent[T combat.HitPointEvent](events []combat.HitPointEvent) (T, bool) {
	for _, e := range events {
		if x, ok := e.(T); ok {
			return x, true
		}
	}
	var zero T
	return zero, false
}

// Requirement: doc/specs/hit-points-and-damage.md FR-102, FR-084, FR-049 - DamageTaken exposes Hit Points lost,
// the amount absorbed by Temporary Hit Points and their sum as plain ints; overkill counts as lost.
// Path: happy.
func TestDamageTakenAmounts(t *testing.T) {
	t.Parallel()
	tr, err := pcStatus(t, 6, 12, 5).ApplyDamage(subject(), []combat.Damage{fire(18)}, nil, combat.OtherSource, combat.NormalHit, nil)
	dt, _ := firstEvent[combat.DamageTaken](applied(t, tr, err).Events())
	if dt.AbsorbedByTemporary() != 5 || dt.HitPointsLost() != 13 || dt.Total() != 18 || len(dt.Resolutions()) != 1 {
		t.Errorf("DamageTaken = absorbed %d lost %d total %d resolutions %d, want 5, 13, 18, 1",
			dt.AbsorbedByTemporary(), dt.HitPointsLost(), dt.Total(), len(dt.Resolutions()))
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-066, FR-075 - damage to a Dead creature is refused with
// CreatureIsDead and the state is unchanged (Glossary "Dead", p. 180).
// Path: unhappy (state error).
func TestApplyDamageToTheDeadIsRefused(t *testing.T) {
	t.Parallel()
	st := deadStatus(t, combat.MassiveDamage)
	tr, err := st.ApplyDamage(subject(), []combat.Damage{fire(5)}, nil, combat.OtherSource, combat.NormalHit, nil)
	if err != nil || tr != (combat.HitPointTransitionRefused{Rejection: combat.CreatureIsDead}) {
		t.Errorf("= %#v, %v, want Refused(CreatureIsDead)", tr, err)
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-085, FR-086, FR-094, FR-114 - ApplyDamage needs a subject,
// a usable status, at least one Damage, a valid delivery and kind, and rejects an invalid Damage or nil
// adjustment with an error matching srderr.ErrInvalidInput; it never panics.
// Path: unhappy (invalid input).
func TestApplyDamageInvalidInput(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 10, 20, 0)
	ok := []combat.Damage{fire(1)}
	cases := []struct {
		name     string
		status   combat.HitPointStatus
		subject  combat.CombatantID
		damages  []combat.Damage
		adj      []combat.DamageAdjustment
		delivery combat.DamageDelivery
		kind     combat.HitKind
		unprev   []combat.Damage
	}{
		{"zero subject", st, "", ok, nil, combat.OtherSource, combat.NormalHit, nil},
		{"zero status", combat.HitPointStatus{}, subject(), ok, nil, combat.OtherSource, combat.NormalHit, nil},
		{"empty damage and unpreventable", st, subject(), nil, nil, combat.OtherSource, combat.NormalHit, nil},
		{"zero delivery", st, subject(), ok, nil, 0, combat.NormalHit, nil},
		{"zero hit kind", st, subject(), ok, nil, combat.OtherSource, 0, nil},
		{"zero Damage", st, subject(), []combat.Damage{{}}, nil, combat.OtherSource, combat.NormalHit, nil},
		{"zero unpreventable Damage", st, subject(), nil, nil, combat.OtherSource, combat.NormalHit, []combat.Damage{{}}},
		{"nil adjustment", st, subject(), ok, []combat.DamageAdjustment{nil}, combat.OtherSource, combat.NormalHit, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := c.status.ApplyDamage(c.subject, c.damages, c.adj, c.delivery, c.kind, c.unprev)
			if tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("= %v, %v, want nil and ErrInvalidInput", tr, err)
			}
		})
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-076, FR-109 - an operation does not mutate its inputs and
// hands out copies of its events and resolutions.
// Path: unhappy (immutability).
func TestApplyDamageDoesNotMutateInputs(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 10, 20, 3)
	damages := []combat.Damage{fire(5), fire(1)}
	adj := []combat.DamageAdjustment{combat.NewFlatAdjustment(1, nil)}
	damagesCopy, adjCopy := slices.Clone(damages), slices.Clone(adj)
	tr, err := st.ApplyDamage(subject(), damages, adj, combat.OtherSource, combat.NormalHit, nil)
	a := applied(t, tr, err)
	if !reflect.DeepEqual(damages, damagesCopy) || !reflect.DeepEqual(adj, adjCopy) || st != pcStatus(t, 10, 20, 3) {
		t.Errorf("inputs changed")
	}
	events := a.Events()
	events[0] = nil
	rs := a.Resolutions()
	rs[0] = combat.DamageResolution{}
	if a.Events()[0] == nil || a.Resolutions()[0] == (combat.DamageResolution{}) {
		t.Errorf("returned slices alias the stored ones")
	}
	again, _ := st.ApplyDamage(subject(), damages, adj, combat.OtherSource, combat.NormalHit, nil)
	if !reflect.DeepEqual(again, tr) {
		t.Errorf("same input, different result (FR-077)")
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-111 - HitKind has exactly two members and a Critical Hit at
// 0 Hit Points adds two failures, not one.
// Path: unhappy (boundary).
func TestHitKindMembers(t *testing.T) {
	t.Parallel()
	if !combat.NormalHit.Valid() || !combat.CriticalHit.Valid() || combat.HitKind(0).Valid() || combat.HitKind(3).Valid() {
		t.Errorf("HitKind validity wrong")
	}
	if combat.DamageDelivery(0).Valid() || combat.DamageDelivery(4).Valid() || !combat.MeleeAttackKnockOut.Valid() {
		t.Errorf("DamageDelivery validity wrong")
	}
	if combat.TemporaryHitPointsChoice(0).Valid() || !combat.KeepCurrent.Valid() || !combat.AcceptNew.Valid() || combat.TemporaryHitPointsChoice(3).Valid() {
		t.Errorf("TemporaryHitPointsChoice validity wrong")
	}
}
