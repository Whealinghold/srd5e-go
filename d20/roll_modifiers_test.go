package d20_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func mods(vals ...int) []modifier.Modifier {
	out := make([]modifier.Modifier, len(vals))
	for i, v := range vals {
		out[i] = modifier.New(v)
	}
	return out
}

func TestRollModifiersTotalAndCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		build     d20.RollModifiers
		wantTotal int
		wantCount int
	}{
		{"zero value", d20.RollModifiers{}, 0, 1},
		{"ability only", d20.NewRollModifiers(modifier.New(3)), 3, 1},
		{"with proficiency", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)), 5, 2},
		{"proficiency of zero still counts", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(0)), 3, 2},
		{"proficiency does not stack", d20.NewRollModifiers(modifier.New(1)).WithProficiencyBonus(modifier.New(2)).WithProficiencyBonus(modifier.New(3)), 4, 2},
		{"others in any mix", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)).WithOthers(mods(1, -2)...), 4, 4},
		{"negative total", d20.NewRollModifiers(modifier.New(-5)).WithOthers(mods(-2)...), -7, 2},
		{"others appended in two calls", d20.NewRollModifiers(modifier.New(0)).WithOthers(mods(1)...).WithOthers(mods(2)...), 3, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.build.Total().Value(); got != tc.wantTotal {
				t.Errorf("Total() = %d, want %d", got, tc.wantTotal)
			}
			if got := tc.build.Count(); got != tc.wantCount {
				t.Errorf("Count() = %d, want %d", got, tc.wantCount)
			}
		})
	}
}

func TestRollModifiersAccessors(t *testing.T) {
	t.Parallel()
	m := d20.NewRollModifiers(modifier.New(3))
	if _, ok := m.ProficiencyBonus(); ok {
		t.Error("ProficiencyBonus() reports one on a fresh value")
	}
	m = m.WithProficiencyBonus(modifier.New(2)).WithOthers(mods(1, 4)...)
	if got := m.Ability().Value(); got != 3 {
		t.Errorf("Ability() = %d", got)
	}
	if p, ok := m.ProficiencyBonus(); !ok || p.Value() != 2 {
		t.Errorf("ProficiencyBonus() = %v, %v", p, ok)
	}
	if got := m.Others(); !slices.Equal(got, mods(1, 4)) {
		t.Errorf("Others() = %v", got)
	}
}

func TestRollModifiersAreImmutable(t *testing.T) {
	t.Parallel()
	src := mods(1, 2)
	m := d20.NewRollModifiers(modifier.New(0)).WithOthers(src...)
	src[0] = modifier.New(99)
	m.Others()[1] = modifier.New(99)
	if got := m.Total().Value(); got != 3 {
		t.Errorf("Total() = %d after mutating inputs, want 3", got)
	}
	base := d20.NewRollModifiers(modifier.New(1))
	_ = base.WithOthers(mods(5)...)
	_ = base.WithProficiencyBonus(modifier.New(5))
	if base.Total().Value() != 1 || base.Count() != 1 {
		t.Error("With* changed the receiver")
	}
}

func TestRollModifiersEqual(t *testing.T) {
	t.Parallel()
	a := d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)).WithOthers(mods(1, 2)...)
	tests := []struct {
		name string
		b    d20.RollModifiers
		want bool
	}{
		{"same", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)).WithOthers(mods(1, 2)...), true},
		{"other order", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)).WithOthers(mods(2, 1)...), false},
		{"other ability", d20.NewRollModifiers(modifier.New(4)).WithProficiencyBonus(modifier.New(2)).WithOthers(mods(1, 2)...), false},
		{"other proficiency", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(1)).WithOthers(mods(1, 2)...), false},
		{"no proficiency", d20.NewRollModifiers(modifier.New(3)).WithOthers(mods(1, 2)...), false},
		{"fewer others", d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)).WithOthers(mods(1)...), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := a.Equal(tc.b); got != tc.want {
				t.Errorf("Equal = %v, want %v", got, tc.want)
			}
		})
	}
}
