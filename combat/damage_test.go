package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-015, FR-086 - a Damage carries exactly one Damage Type,
// accepts an amount of 0 or more and rejects a negative amount or an invalid type with ErrInvalidInput (SRD 5.2.1,
// "Damage Rolls", p. 16; "Damage Types", pp. 16-17).
// Path: per case.
func TestNewDamage(t *testing.T) {
	t.Parallel()
	d, err := combat.NewDamage(7, content.DamageCold)
	if err != nil || d.Amount() != 7 || d.Type() != content.DamageCold || combat.MustDamage(7, content.DamageCold) != d {
		t.Errorf("= %+v, %v", d, err)
	}
	if z, err := combat.NewDamage(0, content.DamageFire); err != nil || z.Amount() != 0 {
		t.Errorf("zero amount = %+v, %v, want valid", z, err)
	}
	for name, c := range map[string]struct {
		amount int
		typ    content.DamageType
	}{"negative": {-1, content.DamageFire}, "zero type": {1, 0}, "unknown type": {1, 99}} {
		if _, err := combat.NewDamage(c.amount, c.typ); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s error = %v, want ErrInvalidInput", name, err)
		}
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustDamage panic = %v", r)
		}
	}()
	combat.MustDamage(-1, content.DamageFire)
}

// Requirement: doc/specs/hit-points-and-damage.md FR-113 - NewFlatAdjustment defaults a nil scope to AllDamage and
// keeps a given one; the zero FlatAdjustment covers all damage.
// Path: unhappy (boundary).
func TestFlatAdjustmentScope(t *testing.T) {
	t.Parallel()
	if a := combat.NewFlatAdjustment(3, nil); a.Amount() != 3 || a.Scope() != (combat.AllDamage{}) {
		t.Errorf("nil scope = %+v", a)
	}
	s := combat.DamageOfType{Type: content.DamageFire}
	if a := combat.NewFlatAdjustment(-2, s); a.Amount() != -2 || a.Scope() != combat.DamageScope(s) {
		t.Errorf("scoped = %+v", a)
	}
	if (combat.FlatAdjustment{}).Scope() != (combat.AllDamage{}) {
		t.Errorf("zero value scope is not AllDamage")
	}
}
