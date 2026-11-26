package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

const (
	uwIn     = combat.UnderwaterStatusUnderwater
	uwOut    = combat.UnderwaterStatusNotUnderwater
	granted  = combat.UnderwaterResistanceGranted
	withheld = combat.UnderwaterResistanceWithheldByEffect
)

// uwTaken is what one instance of damage costs a creature with the affinities, after the flat adjustment, through the damage pipeline.
func uwTaken(tb testing.TB, aff combat.DamageAffinities, d combat.Damage, flat int) int {
	tb.Helper()
	st, err := combat.NewHitPointStatus(1000, combat.FallsUnconscious, aff)
	if err != nil {
		tb.Fatal(err)
	}
	var adj []combat.DamageAdjustment
	if flat != 0 {
		adj = []combat.DamageAdjustment{combat.NewFlatAdjustment(flat, nil)}
	}
	tr, err := st.ApplyDamage(subject(), []combat.Damage{d}, adj, combat.OtherSource, combat.NormalHit, nil)
	return applied(tb, tr, err).Resolutions()[0].Dealt()
}

// Requirement: per case
// Path: per case.
func TestUnderwaterFireResistance(t *testing.T) {
	t.Parallel()
	spec := "doc/specs/mounted-and-underwater-combat.md "
	none := combat.NoAffinities()
	resistFire := affinities(t, []combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}}, nil, nil)
	resistAll := combat.ResistantToAllDamage()
	vulnFire := affinities(t, nil, damageTypes(content.DamageFire), nil)
	immuneFire := affinities(t, nil, nil, damageTypes(content.DamageFire))
	srdExample := affinities(t, []combat.DamageScope{combat.AllDamage{}}, damageTypes(content.DamageFire), nil)
	dmg := func(n int, ty content.DamageType) combat.Damage { return combat.MustDamage(n, ty) }
	cases := []struct {
		name, requirement, path string
		base                    combat.DamageAffinities
		status                  combat.UnderwaterStatus
		grant                   combat.UnderwaterResistanceGrant
		damage                  combat.Damage
		flat, want              int
	}{
		{"underwater halves Fire", spec + "FR-039 table 4.5a", "happy", none, uwIn, granted, fire(15), 0, 7},
		{"not underwater", spec + "FR-040 table 4.5a", "unhappy (boundary)", none, uwOut, granted, fire(15), 0, 15},
		{"not underwater, grant withheld", spec + "FR-040 table 4.5a", "unhappy (boundary)", none, uwOut, withheld, fire(15), 0, 15},
		{"Cold unaffected", spec + "FR-041 table 4.5a", "unhappy (rule interaction)", none, uwIn, granted, dmg(15, content.DamageCold), 0, 15},
		{"Acid unaffected", spec + "FR-041 table 4.5a", "unhappy (rule interaction)", none, uwIn, granted, dmg(15, content.DamageAcid), 0, 15},
		{"Lightning unaffected", spec + "FR-041 table 4.5a", "unhappy (rule interaction)", none, uwIn, granted, dmg(15, content.DamageLightning), 0, 15},
		{"already resists Fire", spec + "FR-042 table 4.5a", "unhappy (rule interaction)", resistFire, uwIn, granted, fire(15), 0, 7},
		{"already resists all damage", spec + "FR-042 table 4.5a", "unhappy (rule interaction)", resistAll, uwIn, granted, fire(15), 0, 7},
		{"Vulnerability to Fire", spec + "FR-043 table 4.5a", "unhappy (rule interaction)", vulnFire, uwIn, granted, fire(15), 0, 14},
		{"Immunity to Fire", spec + "FR-043 table 4.5a", "unhappy (rule interaction)", immuneFire, uwIn, granted, fire(15), 0, 0},
		{"grant withheld by the effect", spec + "FR-044 table 4.5a", "unhappy (rule interaction)", none, uwIn, withheld, fire(15), 0, 15},
		{"withheld, but resists Fire anyway", spec + "FR-044 table 4.5a", "unhappy (rule interaction)", resistFire, uwIn, withheld, fire(15), 0, 7},
		{"one Fire damage", spec + "FR-048 table 4.5a", "unhappy (boundary)", none, uwIn, granted, fire(1), 0, 0},
		{"zero Fire damage", spec + "FR-048 table 4.5a", "unhappy (empty and minimal)", none, uwIn, granted, fire(0), 0, 0},
		{"odd amount rounds down", spec + "FR-048 table 4.5a", "unhappy (boundary)", none, uwIn, granted, fire(9), 0, 4},
		{"adapted worked example", spec + "FR-049 table 4.5a", "unhappy (rule interaction)", vulnFire, uwIn, granted, fire(28), -5, 22},
		{"SRD worked example, not underwater", spec + "FR-070 table 4.5a", "unhappy (rule interaction)", srdExample, uwOut, granted, fire(28), -5, 22},
		{"SRD worked example, underwater: no stacking", spec + "FR-070 table 4.5a", "unhappy (rule interaction)", srdExample, uwIn, granted, fire(28), -5, 22},
		{"status zero adds nothing", spec + "FR-040", "unhappy (invalid input)", none, 0, granted, fire(15), 0, 15},
		{"grant zero adds nothing", spec + "FR-044", "unhappy (invalid input)", none, uwIn, 0, fire(15), 0, 15},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			aff := combat.UnderwaterFireResistanceAffinities(c.base, c.status, c.grant)
			if got := uwTaken(t, aff, c.damage, c.flat); got != c.want {
				t.Errorf("damage taken = %d, want %d [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-041, table 4.5b - underwater adds Resistance to Fire only: 15 damage of each of the thirteen Damage Types costs 7 for Fire and 15 for the other twelve.
// Path: per case.
func TestUnderwaterResistsOnlyFire(t *testing.T) {
	t.Parallel()
	for ty := content.DamageAcid; ty <= content.DamageThunder; ty++ {
		t.Run(ty.String(), func(t *testing.T) {
			t.Parallel()
			want := 15
			if ty == content.DamageFire {
				want = 7
			}
			aff := combat.UnderwaterFireResistanceAffinities(combat.NoAffinities(), uwIn, granted)
			if got := uwTaken(t, aff, combat.MustDamage(15, ty), 0); got != want {
				t.Errorf("15 %v damage underwater costs %d, want %d [FR-041]", ty, got, want)
			}
		})
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-040, FR-042, FR-043, FR-044 - when nothing is added, the result equals the base by value (not underwater, grant withheld, Resistance already held to Fire or to all damage), and vulnerabilities and immunities are carried through.
// Path: unhappy (rule interaction).
func TestUnderwaterAffinitiesEqualBaseWhenNothingIsAdded(t *testing.T) {
	t.Parallel()
	resistFire := affinities(t, []combat.DamageScope{combat.DamageOfType{Type: content.DamageFire}}, nil, nil)
	base := affinities(t, nil, damageTypes(content.DamageCold), damageTypes(content.DamageFire))
	for name, got := range map[string]combat.DamageAffinities{
		"not underwater":       combat.UnderwaterFireResistanceAffinities(base, uwOut, granted),
		"withheld":             combat.UnderwaterFireResistanceAffinities(base, uwIn, withheld),
		"already resists Fire": combat.UnderwaterFireResistanceAffinities(resistFire, uwIn, granted),
		"resists all damage":   combat.UnderwaterFireResistanceAffinities(combat.ResistantToAllDamage(), uwIn, granted),
	} {
		want := map[string]combat.DamageAffinities{"not underwater": base, "withheld": base, "already resists Fire": resistFire, "resists all damage": combat.ResistantToAllDamage()}[name]
		if !got.Equal(want) {
			t.Errorf("%s: affinities changed: %+v, want %+v", name, got, want)
		}
	}
	got := combat.UnderwaterFireResistanceAffinities(base, uwIn, granted)
	if len(got.Vulnerabilities()) != 1 || len(got.Immunities()) != 1 || len(got.Resistances()) != 1 {
		t.Errorf("underwater affinities = %+v, want the vulnerability and immunity kept and Fire Resistance added", got)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-045, FR-046, FR-047 - the Resistance is derived on each call: the base is not altered, leaving the water returns the base, two identical calls are equal, the returned Resistances are a copy, and each creature gets its own result.
// Path: unhappy (immutability).
func TestUnderwaterAffinitiesAreDerivedPerCall(t *testing.T) {
	t.Parallel()
	base := affinities(t, nil, damageTypes(content.DamageCold), nil)
	before := base
	wet := combat.UnderwaterFireResistanceAffinities(base, uwIn, granted)
	dry := combat.UnderwaterFireResistanceAffinities(base, uwOut, granted)
	if base != before || !dry.Equal(base) || !wet.Equal(combat.UnderwaterFireResistanceAffinities(base, uwIn, granted)) || wet.Equal(base) {
		t.Errorf("base %+v wet %+v dry %+v: want the base unchanged, dry == base, wet repeatable and different", base, wet, dry)
	}
	rs := wet.Resistances()
	rs[0] = combat.AllDamage{}
	if got := wet.Resistances(); len(got) != 1 || got[0] != (combat.DamageOfType{Type: content.DamageFire}) {
		t.Errorf("Resistances = %v after the returned slice was changed, want Fire only", got)
	}
	d := fire(15)
	if a, b := uwTaken(t, wet, d, 0), uwTaken(t, dry, d, 0); a != 7 || b != 15 || d != fire(15) {
		t.Errorf("one damage to two creatures took %d and %d, want 7 and 15 with the damage unchanged [FR-047]", a, b)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md section 3 (UnderwaterStatus, UnderwaterResistanceGrant) - the zero value of each is invalid: no constant equals 0.
// Path: unhappy (invalid input).
func TestUnderwaterConstantsAreNotZero(t *testing.T) {
	t.Parallel()
	for _, v := range []int{int(uwIn), int(uwOut), int(granted), int(withheld)} {
		if v < 1 {
			t.Errorf("constant %d, want 1 or more so the zero value is invalid", v)
		}
	}
}
