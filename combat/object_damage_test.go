package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func slash(n int) combat.Damage { return combat.MustDamage(n, content.DamageSlashing) }

func instance(t *testing.T, damages ...combat.Damage) combat.ObjectDamageInstance {
	t.Helper()
	i, err := combat.NewObjectDamageInstance(damages, nil)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func damageObject(t *testing.T, s combat.ObjectStatus, harm combat.ObjectHarm) combat.ApplyDamageToObjectResult {
	t.Helper()
	res, err := combat.NewApplyDamageToObject().Run(context.Background(), combat.ApplyDamageToObjectCommand{Status: s, Harm: harm})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-080 - a Damage Threshold is 1 or more; 0 and negative are ErrInvalidInput errors.
// Path: unhappy (invalid input).
func TestNewDamageThreshold(t *testing.T) {
	t.Parallel()
	if th, err := combat.NewDamageThreshold(1); err != nil || th.Amount() != 1 {
		t.Errorf("threshold 1 = %v, %v", th, err)
	}
	for _, n := range []int{0, -1, math.MinInt} {
		if _, err := combat.NewDamageThreshold(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("threshold %d = %v, want ErrInvalidInput", n, err)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-081 to FR-084, FR-112 - the gate compares the summed incoming amounts of one instance with the threshold: no threshold is not applicable, below is not met, equal or above is met; it knows nothing of objects.
// Path: per case.
func TestDamageThresholdGate(t *testing.T) {
	t.Parallel()
	ten, _ := combat.NewDamageThreshold(10)
	cases := []struct {
		name, requirement, path string
		threshold               combat.DamageThreshold
		damages                 []combat.Damage
		want                    combat.ThresholdOutcome
	}{
		{"no threshold", "FR-112", "unhappy (empty and minimal)", combat.DamageThreshold{}, []combat.Damage{slash(1)}, combat.ThresholdNotApplicable},
		{"one below", "FR-081", "unhappy (boundary)", ten, []combat.Damage{slash(9)}, combat.ThresholdNotMet},
		{"equal", "FR-082", "unhappy (boundary)", ten, []combat.Damage{slash(10)}, combat.ThresholdMet},
		{"one above", "FR-082", "unhappy (boundary)", ten, []combat.Damage{slash(11)}, combat.ThresholdMet},
		{"zero amount", "FR-081", "unhappy (boundary)", ten, []combat.Damage{slash(0)}, combat.ThresholdNotMet},
		{"no damage at all", "FR-081", "unhappy (empty and minimal)", ten, nil, combat.ThresholdNotMet},
		{"components add up", "FR-084", "unhappy (rule interaction)", ten, []combat.Damage{slash(6), combat.MustDamage(5, content.DamageFire)}, combat.ThresholdMet},
		{"components each below", "FR-084", "unhappy (rule interaction)", ten, []combat.Damage{slash(4), combat.MustDamage(5, content.DamageFire)}, combat.ThresholdNotMet},
		{"sum saturates", "FR-084", "unhappy (boundary)", ten, []combat.Damage{slash(math.MaxInt), slash(math.MaxInt)}, combat.ThresholdMet},
	}
	for _, c := range cases {
		if got := (combat.DamageThresholdGate{}).Evaluate(c.threshold, c.damages); got != c.want {
			t.Errorf("%s = %v, want %v [%s]", c.name, got, c.want, c.requirement)
		}
	}
	for o, n := range map[combat.ThresholdOutcome]string{combat.ThresholdNotApplicable: "not_applicable", combat.ThresholdMet: "met", combat.ThresholdNotMet: "not_met", 0: "unknown"} {
		if o.String() != n {
			t.Errorf("outcome %d = %q, want %q", int(o), o, n)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-075 to FR-086, FR-109, FR-124 (table 4.9c) - damage to an object: the threshold is tested on the incoming amounts before Resistance or Vulnerability, a met instance is taken whole through the damage pipeline, Immunity still applies, Hit Points stop at 0 and the object is destroyed once, and the events come in the order of FR-109.
// Path: per case.
func TestApplyDamageToObject(t *testing.T) {
	t.Parallel()
	std := (combat.ObjectAffinities{}).Standard()
	resSlash, _ := combat.NewDamageAffinities([]combat.DamageScope{combat.DamageOfType{Type: content.DamageSlashing}}, nil, nil)
	vulnSlash, _ := combat.NewDamageAffinities(nil, []content.DamageType{content.DamageSlashing}, nil)
	vulnFire := (combat.ObjectAffinities{}).StandardWith(mustAffinities(t, nil, []content.DamageType{content.DamageFire}, nil))
	resFire := (combat.ObjectAffinities{}).StandardWith(mustAffinities(t, []combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}}, nil, nil))
	fire := func(n int) combat.Damage { return combat.MustDamage(n, content.DamageFire) }
	poison := func(n int) combat.Damage { return combat.MustDamage(n, content.DamagePoison) }
	cases := []struct {
		name, requirement, path string
		hp, threshold           int
		aff                     combat.DamageAffinities
		damages                 []combat.Damage
		outcome                 combat.ThresholdOutcome
		taken                   int
		events                  []string
	}{
		{"one below", "FR-081", "happy", 30, 10, std, []combat.Damage{slash(9)}, combat.ThresholdNotMet, 0, []string{"object_damage_deflected"}},
		{"equal", "FR-082", "unhappy (boundary)", 30, 10, std, []combat.Damage{slash(10)}, combat.ThresholdMet, 10, []string{"object_damaged"}},
		{"one above takes all", "FR-082", "unhappy (boundary)", 30, 10, std, []combat.Damage{slash(11)}, combat.ThresholdMet, 11, []string{"object_damaged"}},
		{"zero amount", "FR-081", "unhappy (boundary)", 30, 10, std, []combat.Damage{slash(0)}, combat.ThresholdNotMet, 0, []string{"object_damage_deflected"}},
		{"Resistance after the test", "FR-083", "unhappy (rule interaction)", 30, 10, (combat.ObjectAffinities{}).StandardWith(resSlash), []combat.Damage{slash(12)}, combat.ThresholdMet, 6, []string{"object_damaged"}},
		{"Vulnerability does not lift a miss", "FR-083", "unhappy (rule interaction)", 30, 10, (combat.ObjectAffinities{}).StandardWith(vulnSlash), []combat.Damage{slash(9)}, combat.ThresholdNotMet, 0, []string{"object_damage_deflected"}},
		{"Resistance does not drop a hit", "FR-083", "unhappy (rule interaction)", 30, 10, (combat.ObjectAffinities{}).StandardWith(resSlash), []combat.Damage{slash(10)}, combat.ThresholdMet, 5, []string{"object_damaged"}},
		{"components add up", "FR-084", "unhappy (rule interaction)", 30, 10, std, []combat.Damage{slash(6), fire(5)}, combat.ThresholdMet, 11, []string{"object_damaged"}},
		{"components each below", "FR-084", "unhappy (rule interaction)", 30, 10, std, []combat.Damage{slash(4), fire(5)}, combat.ThresholdNotMet, 0, []string{"object_damage_deflected"}},
		{"Immune type in a met instance", "FR-085", "unhappy (rule interaction)", 30, 10, std, []combat.Damage{poison(8), slash(4)}, combat.ThresholdMet, 4, []string{"object_damaged"}},
		{"no threshold", "FR-079", "happy", 30, 0, std, []combat.Damage{slash(1)}, combat.ThresholdNotApplicable, 1, []string{"object_damaged"}},
		{"Immunity only", "FR-073, FR-079", "unhappy (rule interaction)", 30, 0, std, []combat.Damage{poison(5)}, combat.ThresholdNotApplicable, 0, []string{}},
		{"Vulnerability ruling", "FR-074", "unhappy (rule interaction)", 30, 0, vulnFire, []combat.Damage{fire(4)}, combat.ThresholdNotApplicable, 8, []string{"object_damaged"}},
		{"underwater Fire Resistance", "FR-124", "unhappy (rule interaction)", 30, 0, resFire, []combat.Damage{fire(10), slash(10)}, combat.ThresholdNotApplicable, 15, []string{"object_damaged"}},
		{"destroys", "FR-075", "happy", 30, 0, std, []combat.Damage{slash(40)}, combat.ThresholdNotApplicable, 30, []string{"object_damaged", "object_destroyed"}},
		{"exactly destroys", "FR-076", "unhappy (boundary)", 30, 0, std, []combat.Damage{slash(30)}, combat.ThresholdNotApplicable, 30, []string{"object_damaged", "object_destroyed"}},
		{"zero damage, no threshold", "FR-109", "unhappy (empty and minimal)", 30, 0, std, []combat.Damage{slash(0)}, combat.ThresholdNotApplicable, 0, []string{}},
		{"maximum damage", "FR-076", "unhappy (boundary)", 30, 0, std, []combat.Damage{slash(math.MaxInt), slash(math.MaxInt)}, combat.ThresholdNotApplicable, 30, []string{"object_damaged", "object_destroyed"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := envObject(t, c.hp, c.threshold, c.aff)
			harm := instance(t, c.damages...)
			res := damageObject(t, s, harm)
			got, ok := res.(combat.ObjectDamageApplied)
			if !ok {
				t.Fatalf("= %#v, want applied", res)
			}
			ev := []string{}
			for _, e := range got.Events() {
				ev = append(ev, e.Kind())
			}
			if got.Threshold() != c.outcome || c.hp-got.Status().HitPoints().Current() != c.taken || !equalStrings(ev, c.events) {
				t.Errorf("= %v taken %d events %v, want %v %d %v [%s]", got.Threshold(), c.hp-got.Status().HitPoints().Current(), ev, c.outcome, c.taken, c.events, c.requirement)
			}
			if s.HitPoints().Current() != c.hp {
				t.Error("the input status changed")
			}
			if len(got.Events()) > 0 && c.taken > 0 {
				if d, ok := got.Events()[0].(combat.ObjectDamaged); !ok || d.Amount() != c.taken {
					t.Errorf("ObjectDamaged = %v, want amount %d", got.Events()[0], c.taken)
				}
			}
			if c.outcome != combat.ThresholdNotMet && len(got.Resolutions()) != len(c.damages) || c.outcome == combat.ThresholdNotMet && len(got.Resolutions()) != 0 {
				t.Errorf("resolutions = %d, want %d", len(got.Resolutions()), len(c.damages))
			}
		})
	}
}

func mustAffinities(t *testing.T, res []combat.DamageScope, vuln, imm []content.DamageType) combat.DamageAffinities {
	t.Helper()
	a, err := combat.NewDamageAffinities(res, vuln, imm)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-077, FR-086, FR-109 - a destroyed object refuses all harm without events; breaking an intact object destroys it with one ObjectDestroyed, ignoring the threshold and the affinities, and deals no damage.
// Path: unhappy (state error).
func TestApplyDamageToObjectDestroyedAndBroken(t *testing.T) {
	t.Parallel()
	std := (combat.ObjectAffinities{}).Standard()
	dead := envObject(t, 0, 0, std)
	for name, harm := range map[string]combat.ObjectHarm{"damage": instance(t, slash(5)), "break": combat.ObjectBrokenAutomatically{Basis: combat.AutomaticBreakGMAllowed}} {
		if res := damageObject(t, dead, harm); res != (combat.ObjectDamageRejected{Rejection: combat.ObjectDamageRejectionAlreadyDestroyed}) {
			t.Errorf("%s on a destroyed object = %#v, want AlreadyDestroyed", name, res)
		}
	}
	intact := envObject(t, 30, 50, std)
	for _, basis := range []combat.AutomaticBreakBasis{combat.AutomaticBreakFragileNonmagical, combat.AutomaticBreakGMAllowed} {
		got := damageObject(t, intact, combat.ObjectBrokenAutomatically{Basis: basis}).(combat.ObjectDamageApplied)
		if !got.Status().IsDestroyed() || len(got.Events()) != 1 || got.Events()[0] != (combat.ObjectDestroyed{}) || len(got.Resolutions()) != 0 || got.Threshold() != combat.ThresholdNotApplicable {
			t.Errorf("%v break = %+v, want destroyed with one ObjectDestroyed and no damage resolution", basis, got)
		}
		if got.Status().HitPoints().Maximum() != 30 {
			t.Error("the maximum changed")
		}
	}
	if combat.AutomaticBreakFragileNonmagical.String() != "fragile_nonmagical" || combat.AutomaticBreakGMAllowed.String() != "gm_allowed" || combat.AutomaticBreakBasis(0).String() != "unknown" ||
		combat.ObjectDamageRejectionAlreadyDestroyed.String() != "already_destroyed" || combat.ObjectDamageRejection(0).String() != "unknown" {
		t.Error("names are wrong")
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-078, FR-096 - an empty or malformed instance is an ErrInvalidInput error, the constructor copies and the accessors copy, an unknown harm, a zero status and a cancelled context are errors.
// Path: unhappy (invalid input).
func TestObjectDamageInstanceAndHarmInvalidInput(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		d []combat.Damage
		a []combat.DamageAdjustment
	}{"empty": {nil, nil}, "zero Damage": {[]combat.Damage{{}}, nil}, "nil adjustment": {[]combat.Damage{slash(1)}, []combat.DamageAdjustment{nil}}} {
		if _, err := combat.NewObjectDamageInstance(c.d, c.a); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	damages := []combat.Damage{slash(3)}
	adj := []combat.DamageAdjustment{combat.HalfDamage{}}
	i, err := combat.NewObjectDamageInstance(damages, adj)
	if err != nil {
		t.Fatal(err)
	}
	damages[0], adj[0] = slash(9), nil
	i.Damages()[0] = slash(7)
	i.Adjustments()[0] = nil
	if i.Damages()[0] != slash(3) || i.Adjustments()[0] != (combat.HalfDamage{}) {
		t.Error("the instance aliases a slice")
	}
	uc := combat.NewApplyDamageToObject()
	s := envObject(t, 10, 0, combat.NoAffinities())
	bad := map[string]combat.ApplyDamageToObjectCommand{"nil harm": {Status: s}, "zero instance": {Status: s, Harm: combat.ObjectDamageInstance{}},
		"break without basis": {Status: s, Harm: combat.ObjectBrokenAutomatically{}}, "break with an unknown basis": {Status: s, Harm: combat.ObjectBrokenAutomatically{Basis: 9}}, "unknown harm": {Status: s, Harm: badHarm{}}}
	for name, cmd := range bad {
		if _, err := uc.Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := uc.Run(context.Background(), combat.ApplyDamageToObjectCommand{Harm: i}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero status = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(cancelled, combat.ApplyDamageToObjectCommand{Status: s, Harm: i}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}

type badHarm struct{ combat.ObjectHarm }

// Requirement: doc/specs/environment-hazards-and-objects.md FR-079, FR-097 - the same status and harm give an equal result every time, and flat adjustments and Half Damage of the instance go through the same pipeline as for a creature.
// Path: unhappy (determinism).
func TestApplyDamageToObjectUsesThePipeline(t *testing.T) {
	t.Parallel()
	s := envObject(t, 30, 0, (combat.ObjectAffinities{}).Standard())
	i, _ := combat.NewObjectDamageInstance([]combat.Damage{slash(9)}, []combat.DamageAdjustment{combat.NewFlatAdjustment(3, nil), combat.HalfDamage{}})
	first := damageObject(t, s, i).(combat.ObjectDamageApplied)
	second := damageObject(t, s, i).(combat.ObjectDamageApplied)
	if first.Status() != second.Status() || first.Status().HitPoints().Current() != 24 || first.Resolutions()[0].AfterAdjustments() != 6 {
		t.Errorf("= %d Hit Points, resolution %+v, want (9+3)/2 = 6 taken", first.Status().HitPoints().Current(), first.Resolutions())
	}
	first.Resolutions()[0] = combat.DamageResolution{}
	first.Events()[0] = nil
	if first.Resolutions()[0] == (combat.DamageResolution{}) || first.Events()[0] == nil {
		t.Error("Resolutions or Events is not a copy")
	}
}
