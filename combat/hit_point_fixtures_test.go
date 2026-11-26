package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// subject is the creature every hit point test is about.
func subject() combat.CombatantID { return combat.MustCombatantID("hero") }

// status builds a status with the given parts, failing the test on a bad one.
func status(tb testing.TB, current, maximum, temp int, life combat.LifeState, rule combat.ZeroHitPointsRule, aff combat.DamageAffinities) combat.HitPointStatus {
	tb.Helper()
	s, err := combat.RestoreHitPointStatus(combat.MustHitPoints(current, maximum, temp), life, rule, aff)
	if err != nil {
		tb.Fatalf("RestoreHitPointStatus: %v", err)
	}
	return s
}

// pcStatus is an Alive creature that falls unconscious, with no affinities.
func pcStatus(tb testing.TB, current, maximum, temp int) combat.HitPointStatus {
	tb.Helper()
	return status(tb, current, maximum, temp, combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities())
}

// dyingStatus is a Dying creature with the given tallies and maximum.
func dyingStatus(tb testing.TB, maximum, successes, failures int) combat.HitPointStatus {
	tb.Helper()
	d, err := combat.NewDying(successes, failures)
	if err != nil {
		tb.Fatal(err)
	}
	return status(tb, 0, maximum, 0, d, combat.FallsUnconscious, combat.NoAffinities())
}

func deadStatus(tb testing.TB, cause combat.DeathCause) combat.HitPointStatus {
	tb.Helper()
	return status(tb, 0, 20, 0, dead(cause), combat.FallsUnconscious, combat.NoAffinities())
}

func fire(amount int) combat.Damage { return combat.MustDamage(amount, content.DamageFire) }

func damageTypes(ts ...content.DamageType) []content.DamageType { return ts }

func affinities(tb testing.TB, res []combat.DamageScope, vuln, immune []content.DamageType) combat.DamageAffinities {
	tb.Helper()
	a, err := combat.NewDamageAffinities(res, vuln, immune)
	if err != nil {
		tb.Fatal(err)
	}
	return a
}

// applied returns the applied transition or fails the test.
func applied(tb testing.TB, tr combat.HitPointTransition, err error) combat.HitPointTransitionApplied {
	tb.Helper()
	if err != nil {
		tb.Fatalf("operation error: %v", err)
	}
	a, ok := tr.(combat.HitPointTransitionApplied)
	if !ok {
		tb.Fatalf("transition = %#v, want applied", tr)
	}
	return a
}

// kinds lists the Kind of each event.
func kinds(events []combat.HitPointEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// dying builds Dying(s, f).
func dying(s, f int) combat.Dying {
	d, err := combat.NewDying(s, f)
	if err != nil {
		panic(err)
	}
	return d
}

// dead builds Dead(cause).
func dead(cause combat.DeathCause) combat.Dead {
	d, err := combat.NewDead(cause)
	if err != nil {
		panic(err)
	}
	return d
}
