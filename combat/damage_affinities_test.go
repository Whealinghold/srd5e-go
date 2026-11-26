package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/hit-points-and-damage.md FR-091, FR-021, FR-112 - affinities accept empty slices,
// collapse duplicates, return fresh ordered views, compare equal whatever the order, and the constructors give
// the empty, Resistance-to-all and Immunity-to-all values; the zero value is NoAffinities (SRD 5.2.1, "No
// Stacking", p. 17).
// Path: unhappy (empty and minimal).
func TestDamageAffinities(t *testing.T) {
	t.Parallel()
	if (combat.DamageAffinities{}) != combat.NoAffinities() || !combat.NoAffinities().Equal(combat.DamageAffinities{}) {
		t.Errorf("zero value is not NoAffinities")
	}
	if got, err := combat.NewDamageAffinities(nil, nil, nil); err != nil || got != combat.NoAffinities() {
		t.Errorf("empty = %+v, %v", got, err)
	}
	a := affinities(t, []combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}, combat.AllDamage{}, combat.DamageOfType{Type: content.DamageFire}, combat.DamageOfType{Type: content.DamageAcid}},
		[]content.DamageType{content.DamageCold, content.DamageCold}, []content.DamageType{content.DamageThunder, content.DamageAcid, content.DamageThunder})
	b := affinities(t, []combat.DamageScope{combat.DamageOfType{Type: content.DamageAcid}, combat.DamageOfType{Type: content.DamageFire}, combat.AllDamage{}},
		[]content.DamageType{content.DamageCold}, []content.DamageType{content.DamageAcid, content.DamageThunder})
	if !a.Equal(b) || a != b {
		t.Errorf("same sets in another order are not equal")
	}
	if got := a.Resistances(); !slices.Equal(got, []combat.DamageScope{combat.AllDamage{}, combat.DamageOfType{Type: content.DamageAcid}, combat.DamageOfType{Type: content.DamageFire}}) {
		t.Errorf("Resistances = %v", got)
	}
	if got := a.Immunities(); !slices.Equal(got, []content.DamageType{content.DamageAcid, content.DamageThunder}) {
		t.Errorf("Immunities = %v", got)
	}
	v := a.Vulnerabilities()
	v[0] = content.DamageFire
	if !slices.Equal(a.Vulnerabilities(), []content.DamageType{content.DamageCold}) {
		t.Errorf("Vulnerabilities view is not a copy")
	}
	if a.Equal(affinities(t, nil, []content.DamageType{content.DamageCold}, nil)) {
		t.Errorf("different sets compare equal")
	}
	if got := combat.ResistantToAllDamage().Resistances(); !slices.Equal(got, []combat.DamageScope{combat.AllDamage{}}) || len(combat.ResistantToAllDamage().Immunities()) != 0 {
		t.Errorf("ResistantToAllDamage = %v", got)
	}
	if got := combat.ImmuneToAllDamage().Immunities(); len(got) != 13 || got[0] != content.DamageAcid || got[12] != content.DamageThunder {
		t.Errorf("ImmuneToAllDamage = %v", got)
	}
	if c, d := combat.ImmuneToAllDamage(), combat.ImmuneToAllDamage(); c != d {
		t.Errorf("each call must return an equal independent value")
	}
}

// Requirement: doc/specs/hit-points-and-damage.md FR-091, FR-094 - an invalid Damage Type or a missing scope is
// invalid input.
// Path: unhappy (invalid input).
func TestNewDamageAffinitiesInvalid(t *testing.T) {
	t.Parallel()
	bad := map[string]func() (combat.DamageAffinities, error){
		"resistance type": func() (combat.DamageAffinities, error) {
			return combat.NewDamageAffinities([]combat.DamageScope{combat.DamageOfType{}}, nil, nil)
		},
		"nil scope": func() (combat.DamageAffinities, error) {
			return combat.NewDamageAffinities([]combat.DamageScope{nil}, nil, nil)
		},
		"vulnerability": func() (combat.DamageAffinities, error) {
			return combat.NewDamageAffinities(nil, []content.DamageType{0}, nil)
		},
		"immunity": func() (combat.DamageAffinities, error) {
			return combat.NewDamageAffinities(nil, nil, []content.DamageType{14})
		},
	}
	for name, call := range bad {
		if got, err := call(); !errors.Is(err, srderr.ErrInvalidInput) || got != combat.NoAffinities() {
			t.Errorf("%s = %+v, %v, want zero and ErrInvalidInput", name, got, err)
		}
	}
}
