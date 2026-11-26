package combat_test

import (
	"errors"
	"math"
	"regexp"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// allEvents builds one event of every class for subject.
func allEvents(t *testing.T) map[string]combat.HitPointEvent {
	t.Helper()
	s := subject()
	must := func(e combat.HitPointEvent, err error) combat.HitPointEvent {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	res := []combat.DamageResolution{}
	dt, err := combat.NewDamageTaken(s, res, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]combat.HitPointEvent{
		"damage_taken":                dt,
		"dropped_to_zero_hit_points":  must(combat.NewDroppedToZeroHitPoints(s)),
		"fell_unconscious":            must(combat.NewFellUnconscious(s)),
		"knocked_out":                 must(combat.NewKnockedOut(s)),
		"became_bloodied":             must(combat.NewBecameBloodied(s)),
		"hit_points_restored":         must(combat.NewHitPointsRestored(s, 4)),
		"temporary_hit_points_gained": must(combat.NewTemporaryHitPointsGained(s, 4)),
		"temporary_hit_points_ended":  must(combat.NewTemporaryHitPointsEnded(s)),
		"hit_point_maximum_reduced":   must(combat.NewHitPointMaximumReduced(s, 20, 15)),
		"hit_point_maximum_restored":  must(combat.NewHitPointMaximumRestored(s, 15, 20)),
		"death_saving_throw_made":     must(combat.NewDeathSavingThrowMade(s, combat.DeathSavingThrowSuccess)),
		"became_stable":               must(combat.NewBecameStable(s)),
		"lost_stability":              must(combat.NewLostStability(s)),
		"recovered":                   must(combat.NewRecovered(s)),
		"died":                        must(combat.NewDied(s, combat.MassiveDamage)),
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-103, FR-104, FR-117, combat-foundations.md FR-015 to FR-019 -
// each HitPointEvent has the constant snake_case Kind of its class, unique in the package, a Combatants slice that
// is exactly [subject] and a fresh copy, and the public constructors build the same events the operations emit.
// Path: per case.
func TestHitPointEventContract(t *testing.T) {
	t.Parallel()
	events := allEvents(t)
	if len(events) != 15 {
		t.Fatalf("%d event classes, want the 15 of FR-103", len(events))
	}
	var all []combat.Event
	for kind, e := range events {
		combattest.CheckEvent(t, e, kind, subject())
		all = append(all, e)
		if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(e.Kind()) {
			t.Errorf("kind %q is not snake_case", e.Kind())
		}
	}
	combattest.CheckEventKinds(t, all)
}

// Requirement: doc/specs/hit-points-and-damage.md FR-117, FR-094 - event constructors reject a zero subject and
// invalid data with ErrInvalidInput and never panic.
// Path: unhappy (invalid input).
func TestHitPointEventConstructorsReject(t *testing.T) {
	t.Parallel()
	s := subject()
	calls := map[string]error{}
	add := func(name string, err error) { calls[name] = err }
	_, err := combat.NewDamageTaken("", nil, 0, 0)
	add("damage taken, zero subject", err)
	_, err = combat.NewDamageTaken(s, nil, -1, 0)
	add("damage taken, negative absorbed", err)
	_, err = combat.NewDamageTaken(s, nil, 0, -1)
	add("damage taken, negative lost", err)
	_, err = combat.NewDroppedToZeroHitPoints("")
	add("dropped", err)
	_, err = combat.NewFellUnconscious("")
	add("fell unconscious", err)
	_, err = combat.NewKnockedOut("")
	add("knocked out", err)
	_, err = combat.NewBecameBloodied("")
	add("bloodied", err)
	_, err = combat.NewTemporaryHitPointsEnded("")
	add("temporary ended", err)
	_, err = combat.NewBecameStable("")
	add("stable", err)
	_, err = combat.NewLostStability("")
	add("lost stability", err)
	_, err = combat.NewRecovered("")
	add("recovered", err)
	_, err = combat.NewHitPointsRestored("", 1)
	add("restored, zero subject", err)
	_, err = combat.NewHitPointsRestored(s, 0)
	add("restored, nothing regained", err)
	_, err = combat.NewTemporaryHitPointsGained("", 1)
	add("gained, zero subject", err)
	_, err = combat.NewTemporaryHitPointsGained(s, 0)
	add("gained, nothing", err)
	_, err = combat.NewHitPointMaximumReduced("", 5, 1)
	add("reduced, zero subject", err)
	_, err = combat.NewHitPointMaximumReduced(s, 5, 5)
	add("reduced, not lower", err)
	_, err = combat.NewHitPointMaximumReduced(s, 5, -1)
	add("reduced, negative", err)
	_, err = combat.NewHitPointMaximumRestored("", 1, 5)
	add("restored maximum, zero subject", err)
	_, err = combat.NewHitPointMaximumRestored(s, 5, 5)
	add("restored maximum, not higher", err)
	_, err = combat.NewHitPointMaximumRestored(s, -1, 5)
	add("restored maximum, negative previous", err)
	_, err = combat.NewDeathSavingThrowMade("", combat.DeathSavingThrowSuccess)
	add("made, zero subject", err)
	_, err = combat.NewDeathSavingThrowMade(s, 0)
	add("made, no outcome", err)
	_, err = combat.NewDied("", combat.MassiveDamage)
	add("died, zero subject", err)
	_, err = combat.NewDied(s, 0)
	add("died, no cause", err)
	for name, err := range calls {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s error = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-102, FR-103, FR-117 - events are immutable values compared by
// value (DamageTaken through Equal), expose their data, and DamageTaken's Total saturates instead of overflowing.
// Path: unhappy (immutability).
func TestHitPointEventValues(t *testing.T) {
	t.Parallel()
	events := allEvents(t)
	if events["recovered"] != events["recovered"] || events["died"] == events["recovered"] {
		t.Errorf("events do not compare by value")
	}
	dt := events["damage_taken"].(combat.DamageTaken)
	again, _ := combat.NewDamageTaken(subject(), nil, 2, 3)
	other, _ := combat.NewDamageTaken(subject(), nil, 2, 4)
	otherSubject, _ := combat.NewDamageTaken("someone", nil, 2, 3)
	r := combat.DamageResolution{}
	withRes, _ := combat.NewDamageTaken(subject(), []combat.DamageResolution{r}, 2, 3)
	if !dt.Equal(again) || dt.Equal(other) || dt.Equal(otherSubject) || dt.Equal(withRes) || dt.Total() != 5 {
		t.Errorf("DamageTaken.Equal or Total wrong")
	}
	res := withRes.Resolutions()
	res[0] = combat.DamageResolution{}
	huge, _ := combat.NewDamageTaken(subject(), nil, math.MaxInt, math.MaxInt)
	if huge.Total() <= 0 {
		t.Errorf("Total overflowed: %d", huge.Total())
	}
	if e := events["hit_point_maximum_reduced"].(combat.HitPointMaximumReduced); e.PreviousMaximum() != 20 || e.Maximum() != 15 {
		t.Errorf("reduced = %d to %d", e.PreviousMaximum(), e.Maximum())
	}
	if e := events["hit_point_maximum_restored"].(combat.HitPointMaximumRestored); e.PreviousMaximum() != 15 || e.Maximum() != 20 {
		t.Errorf("restored = %d to %d", e.PreviousMaximum(), e.Maximum())
	}
	if e := events["hit_points_restored"].(combat.HitPointsRestored); e.Regained() != 4 {
		t.Errorf("regained = %d", e.Regained())
	}
	if e := events["temporary_hit_points_gained"].(combat.TemporaryHitPointsGained); e.Amount() != 4 {
		t.Errorf("amount = %d", e.Amount())
	}
	if e := events["died"].(combat.Died); e.Cause() != combat.MassiveDamage {
		t.Errorf("cause = %v", e.Cause())
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-117 - the event constructors accept the lowest legal data and
// reject the value just below it.
// Path: unhappy (boundary).
func TestHitPointEventConstructorBoundaries(t *testing.T) {
	t.Parallel()
	s := subject()
	ok := map[string]error{}
	_, err := combat.NewDamageTaken(s, nil, 0, 0)
	ok["damage taken zero amounts"] = err
	_, err = combat.NewHitPointsRestored(s, 1)
	ok["restored 1"] = err
	_, err = combat.NewTemporaryHitPointsGained(s, 1)
	ok["gained 1"] = err
	_, err = combat.NewHitPointMaximumReduced(s, 1, 0)
	ok["reduced to 0"] = err
	_, err = combat.NewHitPointMaximumRestored(s, 0, 1)
	ok["restored from 0"] = err
	for _, o := range []combat.DeathSavingThrowOutcome{combat.DeathSavingThrowSuccess, combat.DeathSavingThrowFailure, combat.DeathSavingThrowTwoFailures, combat.DeathSavingThrowRegainedHitPoint} {
		_, err = combat.NewDeathSavingThrowMade(s, o)
		ok["outcome "+o.String()] = err
	}
	for _, c := range []combat.DeathCause{combat.MonsterDeath, combat.Exhaustion} {
		_, err = combat.NewDied(s, c)
		ok["cause "+c.String()] = err
	}
	for name, err := range ok {
		if err != nil {
			t.Errorf("%s: %v, want accepted", name, err)
		}
	}
	_, err = combat.NewHitPointMaximumReduced(s, 5, 6)
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("reduced upwards error = %v", err)
	}
	_, err = combat.NewHitPointMaximumRestored(s, 5, 4)
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("restored downwards error = %v", err)
	}
	_, err = combat.NewDeathSavingThrowMade(s, combat.DeathSavingThrowOutcome(5))
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("outcome 5 error = %v", err)
	}
	_, err = combat.NewDied(s, combat.DeathCause(6))
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("cause 6 error = %v", err)
	}
}
