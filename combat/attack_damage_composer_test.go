package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestAttackDamageComposerCompose(t *testing.T) {
	t.Parallel()
	const p16 = `"Damage Rolls", "Critical Hits", p. 16`
	slashing, fire, bludgeoning := content.DamageSlashing, content.DamageFire, content.DamageBludgeoning
	d := func(n, sides int, typ content.DamageType, mod int, add bool) combat.DamageComponent {
		return combat.MustDiceDamage(n, dice.MustDie(sides), typ, mod, add)
	}
	f := func(amount int, typ content.DamageType, mod int, add bool) combat.DamageComponent {
		return combat.MustFixedDamage(amount, typ, mod, add)
	}
	cases := []struct {
		name, requirement, path string
		components              []combat.DamageComponent
		rolls                   [][]int
		ability                 int
		critical                bool
		want                    []combat.Damage
	}{
		{"1d4 first, +2, roll 3", "FR-051, FR-053, FR-060", "happy", []combat.DamageComponent{d(1, 4, slashing, 0, true)}, [][]int{{3}}, 2, false, []combat.Damage{combat.MustDamage(5, slashing)}},
		{"fixed 1 without the flag, +2", "FR-054", "unhappy (rule interaction)", []combat.DamageComponent{f(1, bludgeoning, 0, false)}, [][]int{nil}, 2, false, []combat.Damage{combat.MustDamage(1, bludgeoning)}},
		{"2d6 first, +0, rolls 3 and 4", "FR-060", "happy", []combat.DamageComponent{d(2, 6, slashing, 0, true)}, [][]int{{3, 4}}, 0, false, []combat.Damage{combat.MustDamage(7, slashing)}},
		{"penalty -3 on 1d4 roll 2 floors at 0", "FR-055", "unhappy (boundary)", []combat.DamageComponent{d(1, 4, slashing, 0, true)}, [][]int{{2}}, -3, false, []combat.Damage{combat.MustDamage(0, slashing)}},
		{"penalty -2 on 1d4 roll 2 is exactly 0", "FR-055", "unhappy (boundary)", []combat.DamageComponent{d(1, 4, slashing, 0, true)}, [][]int{{2}}, -2, false, []combat.Damage{combat.MustDamage(0, slashing)}},
		{"penalty -1 on 1d4 roll 2 is 1", "FR-055", "unhappy (boundary)", []combat.DamageComponent{d(1, 4, slashing, 0, true)}, [][]int{{2}}, -1, false, []combat.Damage{combat.MustDamage(1, slashing)}},
		{"critical 1d4 +2, rolls 3 and 4", "FR-056", "happy", []combat.DamageComponent{d(1, 4, slashing, 0, true)}, [][]int{{3, 4}}, 2, true, []combat.Damage{combat.MustDamage(9, slashing)}},
		{"critical 1d8 +3, rolls 5 and 6, modifier not doubled", "FR-058", "unhappy (boundary)", []combat.DamageComponent{d(1, 8, slashing, 0, true)}, [][]int{{5, 6}}, 3, true, []combat.Damage{combat.MustDamage(14, slashing)}},
		{"critical with extra dice", "FR-057", "unhappy (rule interaction)", []combat.DamageComponent{d(1, 6, slashing, 0, true), d(2, 6, slashing, 0, false)}, [][]int{{1, 2}, {3, 4, 5, 6}}, 0, true, []combat.Damage{combat.MustDamage(3, slashing), combat.MustDamage(18, slashing)}},
		{"critical fixed amount is unchanged", "FR-059", "unhappy (boundary)", []combat.DamageComponent{f(5, bludgeoning, 0, false)}, [][]int{nil}, 0, true, []combat.Damage{combat.MustDamage(5, bludgeoning)}},
		{"critical 1d4 -5, rolls 1 and 1", "FR-055, FR-056", "unhappy (boundary)", []combat.DamageComponent{d(1, 4, slashing, 0, true)}, [][]int{{1, 1}}, -5, true, []combat.Damage{combat.MustDamage(0, slashing)}},
		{"two types, flag on the first", "FR-061, FR-096", "happy", []combat.DamageComponent{d(1, 6, slashing, 0, true), d(1, 4, fire, 0, false)}, [][]int{{3}, {2}}, 1, false, []combat.Damage{combat.MustDamage(4, slashing), combat.MustDamage(2, fire)}},
		{"two types, no flag", "FR-096", "unhappy (boundary)", []combat.DamageComponent{d(1, 6, slashing, 0, false), d(1, 4, fire, 0, false)}, [][]int{{3}, {2}}, 1, false, []combat.Damage{combat.MustDamage(3, slashing), combat.MustDamage(2, fire)}},
		{"penalty on the first does not reduce the second", "FR-097, FR-098", "unhappy (boundary)", []combat.DamageComponent{d(1, 6, slashing, -4, false), d(1, 4, fire, 0, false)}, [][]int{{2}, {3}}, 0, false, []combat.Damage{combat.MustDamage(0, slashing), combat.MustDamage(3, fire)}},
		{"ability penalty flagged on the first only", "FR-098, FR-055", "unhappy (boundary)", []combat.DamageComponent{d(1, 6, slashing, 0, true), d(1, 4, fire, 0, false)}, [][]int{{2}, {3}}, -3, false, []combat.Damage{combat.MustDamage(0, slashing), combat.MustDamage(3, fire)}},
		{"a component modifier is added once", "FR-097", "unhappy (rule interaction)", []combat.DamageComponent{d(1, 6, slashing, 2, true)}, [][]int{{3}}, 1, false, []combat.Damage{combat.MustDamage(6, slashing)}},
		{"fixed with the flag: 1 plus the modifier", "FR-054, FR-067", "unhappy (boundary)", []combat.DamageComponent{f(1, bludgeoning, 0, true)}, [][]int{nil}, 3, true, []combat.Damage{combat.MustDamage(4, bludgeoning)}},
		{"saturates at int", "FR-055", "unhappy (boundary)", []combat.DamageComponent{f(math.MaxInt, bludgeoning, math.MaxInt, true)}, [][]int{nil}, math.MaxInt, false, []combat.Damage{combat.MustDamage(math.MaxInt, bludgeoning)}},
		{"saturates at the lowest int", "FR-055", "unhappy (boundary)", []combat.DamageComponent{f(0, bludgeoning, math.MinInt, true)}, [][]int{nil}, math.MinInt, true, []combat.Damage{combat.MustDamage(0, bludgeoning)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := (combat.AttackDamageComposer{}).Compose(combat.MustDamageSpec(c.components...), c.rolls, c.ability, c.critical)
			if err != nil || !slices.Equal(got, c.want) {
				t.Errorf("Compose = %v, %v, want %v [%s; %s]", got, err, c.want, c.requirement, p16)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-056, FR-059 - the dice asked for are the component's own, doubled on a Critical Hit,
// and a fixed amount has none (SRD 5.2.1, "Critical Hits", p. 16).
// Path: per case.
func TestAttackDamageComposerDice(t *testing.T) {
	t.Parallel()
	var c combat.AttackDamageComposer
	three := combat.MustDiceDamage(3, dice.D8(), content.DamageFire, 0, false)
	for _, tc := range []struct {
		critical bool
		want     string
	}{{false, "3d8"}, {true, "6d8"}} {
		if e, ok := c.Dice(three, tc.critical); !ok || e.Notation() != tc.want {
			t.Errorf("Dice(3d8, critical %v) = %v, %v, want %s", tc.critical, e, ok, tc.want)
		}
	}
	if e, ok := c.Dice(combat.MustFixedDamage(5, content.DamageFire, 0, false), true); ok || !e.IsZero() {
		t.Errorf("Dice(fixed) = %v, %v, want none", e, ok)
	}
	if e, ok := c.Dice(combat.MustDiceDamage(500, dice.D4(), content.DamageFire, 0, false), true); !ok || e.Count() != 1000 {
		t.Errorf("Dice(500d4, critical) = %v, %v, want the 1000-die maximum", e, ok)
	}
}

// Requirement: doc/specs/attacks.md FR-080, FR-112 - rolls that do not match the components or the dice asked for are an
// ErrInvalidInput error, not a result, and a spec with no components composes to nothing.
// Path: per case.
func TestAttackDamageComposerMismatch(t *testing.T) {
	t.Parallel()
	spec := combat.MustDamageSpec(combat.MustDiceDamage(2, dice.D6(), content.DamageFire, 0, false))
	for name, rolls := range map[string][][]int{
		"no rolls":                nil,
		"too few faces":           {{1}},
		"too many faces":          {{1, 2, 3}},
		"a critical needs double": {{1, 2}},
		"two components of rolls": {{1, 2}, {3}},
	} {
		_, err := (combat.AttackDamageComposer{}).Compose(spec, rolls, 0, name == "a critical needs double")
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if got, err := (combat.AttackDamageComposer{}).Compose(combat.DamageSpec{}, nil, 3, false); err != nil || len(got) != 0 {
		t.Errorf("Compose(no spec) = %v, %v, want nothing", got, err)
	}
	bad := combat.MustDamageSpec(combat.MustFixedDamage(1, content.DamageFire, 0, false))
	if _, err := (combat.AttackDamageComposer{}).Compose(bad, [][]int{nil}, 0, false); err != nil {
		t.Errorf("Compose(valid fixed) error = %v", err)
	}
}
