package d20_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func abilityMods(v int) d20.RollModifiers { return d20.NewRollModifiers(modifier.New(v)) }

func normalRoll(face int) d20.Roll { return d20.MustRoll(modifier.Normal, face) }

func TestResolveCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		kind        d20.Kind
		roll        d20.Roll
		mods        d20.RollModifiers
		dc          d20.DifficultyClass
		wantTotal   int
		wantOutcome d20.Outcome
	}{
		{"total equals DC", d20.AbilityCheck, normalRoll(10), abilityMods(5), 15, 15, d20.Success},
		{"one below DC", d20.SavingThrow, normalRoll(8), abilityMods(1), 10, 9, d20.Failure},
		{"natural 20 gets no special treatment", d20.AbilityCheck, normalRoll(20), abilityMods(-15), 30, 5, d20.Failure},
		{"natural 1 gets no special treatment", d20.SavingThrow, normalRoll(1), abilityMods(20), 10, 21, d20.Success},
		{"total above 30 is unclamped", d20.AbilityCheck, normalRoll(20), abilityMods(15), 30, 35, d20.Success},
		{"total below zero is unclamped", d20.AbilityCheck, normalRoll(1), abilityMods(-5).WithOthers(modifier.New(-2)), 5, -6, d20.Failure},
		{"advantage uses the higher die", d20.AbilityCheck, d20.MustRoll(modifier.WithAdvantage, 18, 3), abilityMods(0), 15, 18, d20.Success},
		{"disadvantage uses the lower die", d20.AbilityCheck, d20.MustRoll(modifier.WithDisadvantage, 18, 3), abilityMods(0), 15, 3, d20.Failure},
		{"proficiency and others add", d20.SavingThrow, normalRoll(5), abilityMods(2).WithProficiencyBonus(modifier.New(3)).WithOthers(modifier.New(-1)), 9, 9, d20.Success},
		{"zero and negative DC", d20.AbilityCheck, normalRoll(1), abilityMods(-1), 0, 0, d20.Success},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := d20.ResolveCheck(tc.kind, tc.roll, tc.mods, tc.dc)
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind() != tc.kind || got.Roll() != tc.roll || !got.Modifiers().Equal(tc.mods) || got.Target() != tc.dc {
				t.Errorf("result does not carry its inputs: %+v", got)
			}
			if got.Total() != tc.wantTotal || got.Outcome() != tc.wantOutcome {
				t.Errorf("total %d outcome %v, want %d %v", got.Total(), got.Outcome(), tc.wantTotal, tc.wantOutcome)
			}
		})
	}
}

func TestResolveCheckRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		kind d20.Kind
		roll d20.Roll
		want d20.Reason
	}{
		{"attack roll", d20.AttackRoll, normalRoll(10), d20.ReasonKindMismatch},
		{"zero kind", 0, normalRoll(10), d20.ReasonUnknownKind},
		{"undefined kind", 7, normalRoll(10), d20.ReasonUnknownKind},
		{"zero roll", d20.AbilityCheck, d20.Roll{}, d20.ReasonZeroRoll},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := d20.ResolveCheck(tc.kind, tc.roll, abilityMods(0), 10)
			if !errors.Is(err, &d20.InvalidError{Reason: tc.want}) || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("err = %v, want reason %v", err, tc.want)
			}
		})
	}
}

func TestResolveAttack(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		roll         d20.Roll
		mods         d20.RollModifiers
		ac           d20.ArmorClass
		critRange    int
		wantTotal    int
		wantOutcome  d20.Outcome
		wantCritical bool
	}{
		{"total equals AC hits", normalRoll(12), abilityMods(3), 15, 20, 15, d20.Success, false},
		{"one below AC misses", normalRoll(11), abilityMods(3), 15, 20, 14, d20.Failure, false},
		{"natural 20 hits and crits at any AC", normalRoll(20), abilityMods(0), 30, 20, 20, d20.Success, true},
		{"natural 20 hits with a hugely negative total", normalRoll(20), abilityMods(-30), 30, 20, -10, d20.Success, true},
		{"natural 1 misses at any total", normalRoll(1), abilityMods(20), 5, 20, 21, d20.Failure, false},
		{"advantage both 1 misses", d20.MustRoll(modifier.WithAdvantage, 1, 1), abilityMods(20), 5, 20, 21, d20.Failure, false},
		{"disadvantage both 20 crits", d20.MustRoll(modifier.WithDisadvantage, 20, 20), abilityMods(-3), 30, 20, 17, d20.Success, true},
		{"disadvantage 20 and 1 uses the 1", d20.MustRoll(modifier.WithDisadvantage, 20, 1), abilityMods(10), 5, 20, 11, d20.Failure, false},
		{"19 is a plain attack by default", normalRoll(19), abilityMods(0), 25, 20, 19, d20.Failure, false},
		{"19 crits in range 19", normalRoll(19), abilityMods(0), 25, 19, 19, d20.Success, true},
		{"18 does not crit in range 19", normalRoll(18), abilityMods(0), 25, 19, 18, d20.Failure, false},
		{"18 crits in range 18", normalRoll(18), abilityMods(0), 25, 18, 18, d20.Success, true},
		{"17 does not crit in range 18", normalRoll(17), abilityMods(0), 25, 18, 17, d20.Failure, false},
		{"17 can still hit by total in range 18", normalRoll(17), abilityMods(8), 25, 18, 25, d20.Success, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := d20.ResolveAttack(tc.roll, tc.mods, tc.ac, tc.critRange)
			if err != nil {
				t.Fatal(err)
			}
			if got.Total() != tc.wantTotal || got.Outcome() != tc.wantOutcome || got.IsCriticalHit() != tc.wantCritical {
				t.Errorf("total %d outcome %v crit %v", got.Total(), got.Outcome(), got.IsCriticalHit())
			}
			if got.Hit() != (tc.wantOutcome == d20.Success) || got.CriticalRange() != tc.critRange {
				t.Errorf("Hit() %v CriticalRange() %d", got.Hit(), got.CriticalRange())
			}
			if got.Roll() != tc.roll || !got.Modifiers().Equal(tc.mods) || got.Target() != tc.ac {
				t.Error("result does not carry its inputs")
			}
		})
	}
}

func TestResolveAttackRejects(t *testing.T) {
	t.Parallel()
	for _, cr := range []int{-1, 0, 17, 21} {
		_, err := d20.ResolveAttack(normalRoll(10), abilityMods(0), 10, cr)
		if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonCriticalRange}) {
			t.Errorf("range %d: err = %v", cr, err)
		}
	}
	_, err := d20.ResolveAttack(d20.Roll{}, abilityMods(0), 10, d20.DefaultCriticalRange)
	if !errors.Is(err, &d20.InvalidError{Reason: d20.ReasonZeroRoll}) {
		t.Errorf("zero roll: err = %v", err)
	}
}

func TestAdjustedBy(t *testing.T) {
	t.Parallel()
	t.Run("check re-resolves", func(t *testing.T) {
		t.Parallel()
		base, _ := d20.ResolveCheck(d20.SavingThrow, normalRoll(8), abilityMods(1), 12)
		up, err := base.AdjustedBy(3)
		if err != nil || up.Total() != 12 || up.Outcome() != d20.Success || up.Modifiers().Count() != 2 {
			t.Errorf("adjusted = %+v, %v", up, err)
		}
		if base.Total() != 9 || base.Outcome() != d20.Failure {
			t.Error("original changed")
		}
		zero, err := base.AdjustedBy(0)
		if err != nil || zero.Total() != 9 || zero.Modifiers().Count() != 2 {
			t.Errorf("zero delta = %+v, %v", zero, err)
		}
		if _, err := (d20.CheckResult{}).AdjustedBy(1); err == nil {
			t.Error("zero result adjusted without error")
		}
	})
	t.Run("natural 20 stays a critical hit", func(t *testing.T) {
		t.Parallel()
		base, _ := d20.ResolveAttack(normalRoll(20), abilityMods(0), 30, 20)
		down, err := base.AdjustedBy(-50)
		if err != nil || !down.Hit() || !down.IsCriticalHit() || down.Total() != -30 {
			t.Errorf("adjusted = %+v, %v", down, err)
		}
	})
	t.Run("natural 1 stays a miss and the range is kept", func(t *testing.T) {
		t.Parallel()
		base, _ := d20.ResolveAttack(normalRoll(1), abilityMods(0), 5, 18)
		up, err := base.AdjustedBy(50)
		if err != nil || up.Hit() || up.CriticalRange() != 18 {
			t.Errorf("adjusted = %+v, %v", up, err)
		}
	})
	t.Run("attack total crosses the AC", func(t *testing.T) {
		t.Parallel()
		base, _ := d20.ResolveAttack(normalRoll(10), abilityMods(0), 12, 20)
		up, _ := base.AdjustedBy(2)
		if base.Hit() || !up.Hit() {
			t.Errorf("base hit %v, adjusted hit %v", base.Hit(), up.Hit())
		}
		if _, err := (d20.AttackRollResult{}).AdjustedBy(1); err == nil {
			t.Error("zero result adjusted without error")
		}
	})
}

func TestResultEqual(t *testing.T) {
	t.Parallel()
	a, _ := d20.ResolveCheck(d20.AbilityCheck, normalRoll(10), abilityMods(1), 12)
	b, _ := d20.ResolveCheck(d20.AbilityCheck, normalRoll(10), abilityMods(1), 12)
	c, _ := d20.ResolveCheck(d20.AbilityCheck, normalRoll(10), abilityMods(1), 13)
	if !a.Equal(b) || a.Equal(c) {
		t.Error("CheckResult.Equal wrong")
	}
	x, _ := d20.ResolveAttack(normalRoll(10), abilityMods(1), 12, 20)
	y, _ := d20.ResolveAttack(normalRoll(10), abilityMods(1), 12, 20)
	z, _ := d20.ResolveAttack(normalRoll(10), abilityMods(1), 12, 19)
	if !x.Equal(y) || x.Equal(z) {
		t.Error("AttackRollResult.Equal wrong")
	}
}

func TestFailWithoutRoll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		kind   d20.Kind
		reason d20.ForcedFailureReason
		want   d20.Reason // 0 means allowed
	}{
		{"save chosen to fail", d20.SavingThrow, d20.ChosenToFail, 0},
		{"save automatic", d20.SavingThrow, d20.Automatic, 0},
		{"check automatic", d20.AbilityCheck, d20.Automatic, 0},
		{"check chosen to fail", d20.AbilityCheck, d20.ChosenToFail, d20.ReasonForcedFailureNotAllowed},
		{"attack automatic", d20.AttackRoll, d20.Automatic, d20.ReasonForcedFailureNotAllowed},
		{"attack chosen", d20.AttackRoll, d20.ChosenToFail, d20.ReasonForcedFailureNotAllowed},
		{"unknown kind", 0, d20.Automatic, d20.ReasonUnknownKind},
		{"unknown reason", d20.SavingThrow, 0, d20.ReasonUnknownReason},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := d20.FailWithoutRoll(tc.kind, tc.reason)
			if tc.want != 0 {
				if !errors.Is(err, &d20.InvalidError{Reason: tc.want}) {
					t.Errorf("err = %v, want reason %v", err, tc.want)
				}
				return
			}
			if err != nil || got.Kind() != tc.kind || got.Reason() != tc.reason {
				t.Errorf("got %+v, %v", got, err)
			}
			if got != mustFail(t, tc.kind, tc.reason) {
				t.Error("equal results differ under ==")
			}
		})
	}
}

func mustFail(t *testing.T, k d20.Kind, r d20.ForcedFailureReason) d20.ForcedFailureResult {
	t.Helper()
	f, err := d20.FailWithoutRoll(k, r)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// The natural 20 and natural 1 rules hold for every modifier and AC.
func FuzzResolveAttack(f *testing.F) {
	f.Add(20, 0, 10, 20)
	f.Add(1, 5, 1, 18)
	f.Add(19, -3, 25, 19)
	f.Fuzz(func(t *testing.T, face, mod, ac, critRange int) {
		roll, err := d20.NewRoll(modifier.Normal, []int{face})
		if err != nil {
			return
		}
		got, err := d20.ResolveAttack(roll, abilityMods(mod), d20.ArmorClass(ac), critRange)
		if err != nil {
			if critRange >= 18 && critRange <= 20 {
				t.Fatalf("valid range %d rejected: %v", critRange, err)
			}
			return
		}
		switch {
		case face == 20 && (!got.Hit() || !got.IsCriticalHit()):
			t.Fatalf("natural 20 did not crit: %+v", got)
		case face == 1 && (got.Hit() || got.IsCriticalHit()):
			t.Fatalf("natural 1 hit: %+v", got)
		case got.IsCriticalHit() && !got.Hit():
			t.Fatalf("critical miss: %+v", got)
		}
	})
}
