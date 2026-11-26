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

// rrWarded applies damage through a ward and returns the applied result or fails the test.
func rrWarded(tb testing.TB, st combat.HitPointStatus, damages []combat.Damage, delivery combat.DamageDelivery, ward combat.DeathWard) combat.GuardedDamageApplied {
	tb.Helper()
	res, err := combat.ApplyDamageWithDeathWard(gid("hero"), st, damages, nil, delivery, combat.NormalHit, nil, ward)
	if err != nil {
		tb.Fatalf("ApplyDamageWithDeathWard: %v", err)
	}
	a, ok := res.(combat.GuardedDamageApplied)
	if !ok {
		tb.Fatalf("ApplyDamageWithDeathWard = %T, want GuardedDamageApplied", res)
	}
	return a
}

func rrCold(n int) combat.Damage { return combat.MustDamage(n, content.DamageCold) }

// Requirement: per case (table 4.8; baseline: maximum 20, FallsUnconscious, ward WardActive; Source for every row: SRD 5.2.1, "Death Ward" p. 122)
// Path: per case.
func TestApplyDamageWithDeathWard(t *testing.T) {
	t.Parallel()
	monster := func(current, temp int) combat.HitPointStatus {
		return status(t, current, 20, temp, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities())
	}
	cases := []struct {
		name, requirement, path string
		status                  combat.HitPointStatus
		damages                 []combat.Damage
		delivery                combat.DamageDelivery
		ward                    combat.DeathWard
		current, temp           int
		life                    string
		wantWard                combat.DeathWard
		events                  []string
	}{
		{"ordinary hit", "FR-041", "happy", pcStatus(t, 20, 20, 0), []combat.Damage{fire(5)}, combat.OtherSource, combat.WardActive, 15, 0, "alive", combat.WardActive,
			[]string{"damage_taken"}},
		{"exactly to 0", "FR-042", "happy", pcStatus(t, 5, 20, 0), []combat.Damage{fire(5)}, combat.OtherSource, combat.WardActive, 1, 0, "alive", combat.WardSpent,
			[]string{"damage_taken", "death_ward_spent"}},
		{"well past 0, no Massive Damage", "FR-042", "unhappy (boundary)", pcStatus(t, 5, 20, 0), []combat.Damage{fire(100)}, combat.OtherSource, combat.WardActive, 1, 0, "alive", combat.WardSpent,
			[]string{"damage_taken", "death_ward_spent"}},
		{"one short of 0", "FR-041", "unhappy (boundary)", pcStatus(t, 5, 20, 0), []combat.Damage{fire(4)}, combat.OtherSource, combat.WardActive, 1, 0, "alive", combat.WardActive,
			[]string{"damage_taken"}},
		{"temporary Hit Points absorb part", "FR-042", "unhappy (rule interaction)", pcStatus(t, 5, 20, 3), []combat.Damage{fire(8)}, combat.OtherSource, combat.WardActive, 1, 0, "alive", combat.WardSpent,
			[]string{"damage_taken", "temporary_hit_points_ended", "death_ward_spent"}},
		{"fully absorbed", "FR-041", "unhappy (rule interaction)", pcStatus(t, 5, 20, 10), []combat.Damage{fire(8)}, combat.OtherSource, combat.WardActive, 5, 2, "alive", combat.WardActive,
			[]string{"damage_taken"}},
		{"monster", "FR-042", "unhappy (rule interaction)", monster(5, 0), []combat.Damage{fire(5)}, combat.OtherSource, combat.WardActive, 1, 0, "alive", combat.WardSpent,
			[]string{"damage_taken", "death_ward_spent"}},
		{"monster well past 0", "FR-042", "unhappy (rule interaction)", monster(5, 0), []combat.Damage{fire(500)}, combat.MeleeAttackWithoutKnockOut, combat.WardActive, 1, 0, "alive", combat.WardSpent,
			[]string{"damage_taken", "death_ward_spent"}},
		{"knock-out", "FR-041", "unhappy (rule interaction)", pcStatus(t, 5, 20, 0), []combat.Damage{fire(5)}, combat.MeleeAttackKnockOut, combat.WardActive, 1, 0, "alive", combat.WardActive,
			[]string{"damage_taken", "knocked_out"}},
		{"already at 0", "FR-041", "unhappy (state error)", dyingStatus(t, 20, 0, 0), []combat.Damage{fire(3)}, combat.OtherSource, combat.WardActive, 0, 0, "dying", combat.WardActive,
			[]string{"damage_taken"}},
		{"two types summed", "FR-042", "unhappy (rule interaction)", pcStatus(t, 6, 20, 0), []combat.Damage{fire(10), rrCold(8)}, combat.OtherSource, combat.WardActive, 1, 0, "alive", combat.WardSpent,
			[]string{"damage_taken", "death_ward_spent"}},
		{"Immunity", "FR-041", "unhappy (empty and minimal)",
			status(t, 5, 20, 0, combat.Alive{}, combat.FallsUnconscious, affinities(t, nil, nil, damageTypes(content.DamageFire))), []combat.Damage{fire(5)}, combat.OtherSource,
			combat.WardActive, 5, 0, "alive", combat.WardActive, nil},
		{"second drop with a spent ward", "FR-044", "unhappy (rule interaction)", pcStatus(t, 5, 20, 0), []combat.Damage{fire(5)}, combat.OtherSource, combat.WardSpent, 0, 0, "dying", combat.WardSpent,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious"}},
		{"a spent ward lets a Dying creature fail", "FR-044", "unhappy (state error)", dyingStatus(t, 20, 0, 1), []combat.Damage{fire(3)}, combat.OtherSource, combat.WardSpent, 0, 0, "dying", combat.WardSpent,
			[]string{"damage_taken"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: doc/specs/revival-and-restoration-spells.md (the case's requirement)
			a := rrWarded(t, c.status, c.damages, c.delivery, c.ward)
			hp := a.Status().HitPoints()
			var life string
			switch a.Status().LifeState().(type) {
			case combat.Alive:
				life = "alive"
			case combat.Dying:
				life = "dying"
			default:
				life = "other"
			}
			if hp.Current() != c.current || hp.Temporary() != c.temp || life != c.life || a.Ward() != c.wantWard {
				t.Errorf("status = %d current, %d temporary, %s, ward %v; want %d, %d, %s, %v [%s; %s]", hp.Current(), hp.Temporary(), life, a.Ward(), c.current, c.temp, c.life, c.wantWard, c.requirement, c.path)
			}
			if got := rrKinds(a.Events()); !slices.Equal(got, append([]string{}, c.events...)) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, c.requirement)
			}
			if c.wantWard == combat.WardSpent && c.ward == combat.WardActive {
				if ev, ok := a.Events()[len(a.Events())-1].(combat.DeathWardSpent); !ok || ev.Reason() != combat.DropToZero {
					t.Errorf("last event = %#v, want DeathWardSpent (DropToZero)", a.Events()[len(a.Events())-1])
				}
			}
		})
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-042 - the saved creature neither takes Massive Damage nor fails a Death Saving Throw, and the damage pipeline of the rerun
// (the post-pipeline amounts, with the Resistance applied) is the one reported; Bloodied is reported by the rerun.
// Path: unhappy (rule interaction).
func TestApplyDamageWithDeathWardRerunDetails(t *testing.T) {
	t.Parallel()
	a := rrWarded(t, pcStatus(t, 8, 20, 0), []combat.Damage{fire(40)}, combat.OtherSource, combat.WardActive)
	if got := rrKinds(a.Events()); !slices.Equal(got, []string{"damage_taken", "death_ward_spent"}) {
		t.Errorf("events = %v, want damage_taken then death_ward_spent", got)
	}
	if d, ok := a.Events()[0].(combat.DamageTaken); !ok || d.Total() != 40 || len(a.Resolutions()) != 1 {
		t.Errorf("DamageTaken = %#v with %d resolutions, want the 40 damage of the call", a.Events()[0], len(a.Resolutions()))
	}
	if life, ok := a.Status().LifeState().(combat.Alive); !ok || life != (combat.Alive{}) {
		t.Errorf("life state = %#v, want Alive", a.Status().LifeState())
	}
	bloodied := rrWarded(t, pcStatus(t, 11, 20, 0), []combat.Damage{fire(11)}, combat.OtherSource, combat.WardActive)
	if got := rrKinds(bloodied.Events()); !slices.Equal(got, []string{"damage_taken", "became_bloodied", "death_ward_spent"}) {
		t.Errorf("events = %v, want a Bloodied creature saved with became_bloodied before death_ward_spent", got)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-041, FR-044 - a refused call spends nothing: a Dead creature gives GuardedDamageRefused (CreatureIsDead) for both ward states
// and for NegateInstantKill, and the input status is as passed; a call that errors spends nothing either.
// Path: unhappy (state error).
func TestDeathWardRefusalSpendsNothing(t *testing.T) {
	t.Parallel()
	st := deadStatus(t, combat.MassiveDamage)
	for _, ward := range []combat.DeathWard{combat.WardActive, combat.WardSpent} {
		res, err := combat.ApplyDamageWithDeathWard(gid("hero"), st, []combat.Damage{fire(3)}, nil, combat.OtherSource, combat.NormalHit, nil, ward)
		if err != nil || res != (combat.GuardedDamageRefused{Rejection: combat.CreatureIsDead}) {
			t.Errorf("damage to a Dead creature, ward %v = %v, %v, want GuardedDamageRefused (CreatureIsDead)", ward, res, err)
		}
		res, err = combat.NegateInstantKill(gid("hero"), st, ward)
		if err != nil || res != (combat.GuardedDamageRefused{Rejection: combat.CreatureIsDead}) {
			t.Errorf("NegateInstantKill on a Dead creature, ward %v = %v, %v, want GuardedDamageRefused (CreatureIsDead)", ward, res, err)
		}
	}
	if !reflect.DeepEqual(st, deadStatus(t, combat.MassiveDamage)) {
		t.Error("a refusal changed the status")
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-043, FR-044 - NegateInstantKill with an active ward leaves the status unchanged, spends the ward and emits DeathWardSpent
// (InstantKill); with a spent ward it is GuardedDamageNotWarded and changes nothing (SRD 5.2.1, "Death Ward", p. 122).
// Path: unhappy (rule interaction).
func TestNegateInstantKill(t *testing.T) {
	t.Parallel()
	st := pcStatus(t, 5, 20, 2)
	res, err := combat.NegateInstantKill(gid("hero"), st, combat.WardActive)
	a, ok := res.(combat.GuardedDamageApplied)
	if err != nil || !ok || !reflect.DeepEqual(a.Status(), st) || a.Ward() != combat.WardSpent || len(a.Resolutions()) != 0 {
		t.Fatalf("NegateInstantKill(active) = %#v, %v, want the status unchanged and the ward spent", res, err)
	}
	if ev := a.Events(); len(ev) != 1 || ev[0].(combat.DeathWardSpent).Reason() != combat.InstantKill {
		t.Errorf("events = %v, want one DeathWardSpent (InstantKill)", rrKinds(ev))
	}
	if res, err := combat.NegateInstantKill(gid("hero"), st, combat.WardSpent); err != nil || res != (combat.GuardedDamageNotWarded{}) {
		t.Errorf("NegateInstantKill(spent) = %v, %v, want GuardedDamageNotWarded", res, err)
	}
	var zero combat.DeathWard
	if zero != combat.WardActive {
		t.Error("the zero DeathWard is not WardActive")
	}
	dying := dyingStatus(t, 20, 1, 1)
	if a, ok := func() (combat.GuardedDamageApplied, bool) {
		r, _ := combat.NegateInstantKill(gid("hero"), dying, combat.WardActive)
		x, ok := r.(combat.GuardedDamageApplied)
		return x, ok
	}(); !ok || !reflect.DeepEqual(a.Status(), dying) {
		t.Errorf("a Dying creature: %#v, want the ward spent and the status unchanged", a)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-044, table 4.8 maximum reduced to 0 - the ward guards ApplyDamageWithDeathWard only: ReduceMaximum to 0 kills the creature
// (Dead, HitPointMaximumZero) and the ward, a value the host holds, is not an input of it and stays WardActive.
// Path: unhappy (rule interaction).
func TestDeathWardDoesNotGuardMaximumReduction(t *testing.T) {
	t.Parallel()
	ward := combat.WardActive
	tr, err := pcStatus(t, 5, 20, 0).ReduceMaximum(gid("hero"), 20)
	a := applied(t, tr, err)
	if d, ok := a.Status().LifeState().(combat.Dead); !ok || d.Cause() != combat.HitPointMaximumZero || ward != combat.WardActive {
		t.Errorf("life state = %#v, ward %v, want Dead (HitPointMaximumZero) with the ward untouched", a.Status().LifeState(), ward)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-047 - an unknown ward value, a zero subject or status and no damage are errors matching srderr.ErrInvalidInput with a nil result.
// Path: unhappy (invalid input).
func TestDeathWardInvalidInput(t *testing.T) {
	t.Parallel()
	st, dmg := pcStatus(t, 5, 20, 0), []combat.Damage{fire(3)}
	for name, call := range map[string]func() (combat.GuardDamageWithDeathWardResult, error){
		"damage, unknown ward": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.ApplyDamageWithDeathWard(gid("hero"), st, dmg, nil, combat.OtherSource, combat.NormalHit, nil, 7)
		},
		"damage, negative ward": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.ApplyDamageWithDeathWard(gid("hero"), st, dmg, nil, combat.OtherSource, combat.NormalHit, nil, -1)
		},
		"damage, no damage": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.ApplyDamageWithDeathWard(gid("hero"), st, nil, nil, combat.OtherSource, combat.NormalHit, nil, combat.WardActive)
		},
		"damage, zero subject": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.ApplyDamageWithDeathWard("", st, dmg, nil, combat.OtherSource, combat.NormalHit, nil, combat.WardActive)
		},
		"damage, zero status": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.ApplyDamageWithDeathWard(gid("hero"), combat.HitPointStatus{}, dmg, nil, combat.OtherSource, combat.NormalHit, nil, combat.WardActive)
		},
		"damage, invalid delivery": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.ApplyDamageWithDeathWard(gid("hero"), st, dmg, nil, 0, combat.NormalHit, nil, combat.WardActive)
		},
		"negate, unknown ward": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.NegateInstantKill(gid("hero"), st, 7)
		},
		"negate, zero subject": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.NegateInstantKill("", st, combat.WardActive)
		},
		"negate, zero status": func() (combat.GuardDamageWithDeathWardResult, error) {
			return combat.NegateInstantKill(gid("hero"), combat.HitPointStatus{}, combat.WardActive)
		},
	} {
		if res, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: %v, %v, want a nil result and an invalid input error", name, res, err)
		}
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-045 - ApplyDamageWithDeathWard changes none of its inputs, gives equal results for equal inputs and returns copies of its events and resolutions.
// Path: unhappy (immutability).
func TestApplyDamageWithDeathWardImmutable(t *testing.T) {
	t.Parallel()
	st, dmg := pcStatus(t, 5, 20, 0), []combat.Damage{fire(9)}
	beforeStatus, beforeDmg := st, slices.Clone(dmg)
	first, second := rrWarded(t, st, dmg, combat.OtherSource, combat.WardActive), rrWarded(t, st, dmg, combat.OtherSource, combat.WardActive)
	if !reflect.DeepEqual(st, beforeStatus) || !slices.Equal(dmg, beforeDmg) || !reflect.DeepEqual(first, second) {
		t.Error("ApplyDamageWithDeathWard changed an input or gave unequal results for equal inputs")
	}
	ev, res := first.Events(), first.Resolutions()
	ev[0], res[0] = nil, combat.DamageResolution{}
	if first.Events()[0] == nil || first.Resolutions()[0] == (combat.DamageResolution{}) {
		t.Error("Events or Resolutions is not a copy")
	}
	for w, want := range map[combat.DeathWard]string{combat.WardActive: "active", combat.WardSpent: "spent", 5: "unknown", -1: "unknown"} {
		if w.String() != want {
			t.Errorf("DeathWard(%d).String() = %q, want %q", int(w), w.String(), want)
		}
	}
}
