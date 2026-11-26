package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attacks.md FR-033, FR-107 - a dice component needs 1 to 500 dice (so a Critical Hit can double
// them), a die and a Damage Type; the error joins every violated field and matches srderr.ErrInvalidInput.
// Path: per case.
func TestNewDiceDamage(t *testing.T) {
	t.Parallel()
	count, die, typ := combat.DamageComponentFieldDiceCount, combat.DamageComponentFieldDie, combat.DamageComponentFieldType
	cases := []struct {
		name, requirement, path string
		count                   int
		die                     dice.Die
		typ                     content.DamageType
		fields                  []combat.DamageComponentField
	}{
		{"1d6 slashing", "FR-033", "happy", 1, dice.D6(), content.DamageSlashing, nil},
		{"500 dice", "FR-033", "unhappy (boundary)", 500, dice.D6(), content.DamageSlashing, nil},
		{"501 dice cannot double", "FR-033", "unhappy (boundary)", 501, dice.D6(), content.DamageSlashing, []combat.DamageComponentField{count}},
		{"zero dice", "FR-033", "unhappy (boundary)", 0, dice.D6(), content.DamageSlashing, []combat.DamageComponentField{count}},
		{"negative dice", "FR-033", "unhappy (invalid input)", -1, dice.D6(), content.DamageSlashing, []combat.DamageComponentField{count}},
		{"zero die", "FR-033", "unhappy (invalid input)", 1, dice.Die{}, content.DamageSlashing, []combat.DamageComponentField{die}},
		{"no Damage Type", "FR-033", "unhappy (invalid input)", 1, dice.D6(), 0, []combat.DamageComponentField{typ}},
		{"everything wrong", "FR-107", "unhappy (invalid input)", 0, dice.Die{}, 0, []combat.DamageComponentField{count, die, typ}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := combat.NewDiceDamage(c.count, c.die, c.typ, -2, true)
			if len(c.fields) == 0 {
				if err != nil || d.Count() != c.count || d.Die() != c.die || d.Type() != c.typ || d.Modifier() != -2 || !d.AddsAbilityModifier() {
					t.Errorf("NewDiceDamage = %+v, %v [%s]", d, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || d != (combat.DiceDamage{}) {
				t.Fatalf("NewDiceDamage = %+v, %v, want the zero value and ErrInvalidInput [%s]", d, err, c.requirement)
			}
			for _, f := range c.fields {
				if !errors.Is(err, combat.DamageComponentProblem{Field: f}) {
					t.Errorf("error lacks %v [%s]", f, c.requirement)
				}
			}
			for _, f := range []combat.DamageComponentField{count, die, typ} {
				want := false
				for _, w := range c.fields {
					want = want || w == f
				}
				if errors.Is(err, combat.DamageComponentProblem{Field: f}) != want {
					t.Errorf("problem %v present = %v, want %v [%s]", f, !want, want, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-033, FR-107 - a fixed amount is 0 or more and has a Damage Type; MustX panic with
// the constructor's error.
// Path: per case.
func TestNewFixedDamage(t *testing.T) {
	t.Parallel()
	d, err := combat.NewFixedDamage(0, content.DamageBludgeoning, 3, false)
	if err != nil || d.Amount() != 0 || d.Type() != content.DamageBludgeoning || d.Modifier() != 3 || d.AddsAbilityModifier() {
		t.Errorf("NewFixedDamage(0) = %+v, %v, want an amount of 0 to be valid (FR-033)", d, err)
	}
	_, err = combat.NewFixedDamage(-1, 0, 0, false)
	if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, combat.DamageComponentProblem{Field: combat.DamageComponentFieldAmount}) ||
		!errors.Is(err, combat.DamageComponentProblem{Field: combat.DamageComponentFieldType}) {
		t.Errorf("NewFixedDamage(-1, none) error = %v, want amount and type problems", err)
	}
	for name, f := range map[string]func(){
		"MustFixedDamage": func() { combat.MustFixedDamage(-1, content.DamageFire, 0, false) },
		"MustDiceDamage":  func() { combat.MustDiceDamage(0, dice.D6(), content.DamageFire, 0, false) },
		"MustDamageSpec":  func() { combat.MustDamageSpec() },
	} {
		func() {
			defer func() {
				if err, _ := recover().(error); !errors.Is(err, srderr.ErrInvalidInput) {
					t.Errorf("%s panic = %v, want an ErrInvalidInput error", name, err)
				}
			}()
			f()
		}()
	}
}

// Requirement: doc/specs/attacks.md FR-062, FR-096, FR-107 - a spec has at least one component and at most one that adds
// the ability modifier; the zero spec deals no damage; the components are copied in and out.
// Path: per case.
func TestNewDamageSpec(t *testing.T) {
	t.Parallel()
	slash := combat.MustDiceDamage(1, dice.D6(), content.DamageSlashing, 0, true)
	fire := combat.MustDiceDamage(1, dice.D4(), content.DamageFire, 0, false)
	fire2 := combat.MustFixedDamage(2, content.DamageFire, 0, true)
	comp, add := combat.DamageSpecFieldComponents, combat.DamageSpecFieldAddsAbilityModifier
	cases := []struct {
		name, requirement, path string
		components              []combat.DamageComponent
		fields                  []combat.DamageSpecField
	}{
		{"one component", "FR-062", "happy", []combat.DamageComponent{slash}, nil},
		{"two types, one flagged", "FR-061, FR-096", "happy", []combat.DamageComponent{slash, fire}, nil},
		{"none flagged", "FR-096", "unhappy (boundary)", []combat.DamageComponent{fire}, nil},
		{"empty", "FR-062", "unhappy (empty and minimal case)", nil, []combat.DamageSpecField{comp}},
		{"both flagged", "FR-096", "unhappy (invalid input)", []combat.DamageComponent{slash, fire2}, []combat.DamageSpecField{add}},
		{"a nil component", "FR-107", "unhappy (invalid input)", []combat.DamageComponent{slash, nil}, []combat.DamageSpecField{comp}},
		{"a zero-value component", "FR-107", "unhappy (invalid input)", []combat.DamageComponent{combat.DiceDamage{}}, []combat.DamageSpecField{comp}},
		{"a zero-value fixed component", "FR-107", "unhappy (invalid input)", []combat.DamageComponent{combat.FixedDamage{}}, []combat.DamageSpecField{comp}},
		{"empty and no component to flag", "FR-107", "unhappy (invalid input)", []combat.DamageComponent{}, []combat.DamageSpecField{comp}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.NewDamageSpec(c.components...)
			if len(c.fields) == 0 {
				if err != nil || s.IsZero() || len(s.Components()) != len(c.components) {
					t.Errorf("NewDamageSpec = %+v, %v [%s]", s, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !s.IsZero() {
				t.Fatalf("NewDamageSpec = %+v, %v, want the zero spec and ErrInvalidInput [%s]", s, err, c.requirement)
			}
			for _, f := range c.fields {
				if !errors.Is(err, combat.DamageSpecProblem{Field: f}) {
					t.Errorf("error lacks %v [%s]", f, c.requirement)
				}
			}
		})
	}
	// Immutability: neither the input slice nor a returned slice reaches the spec.
	in := []combat.DamageComponent{slash, fire}
	s := combat.MustDamageSpec(in...)
	in[0] = fire2
	got := s.Components()
	got[1] = fire2
	if again := s.Components(); again[0] != combat.DamageComponent(slash) || again[1] != combat.DamageComponent(fire) {
		t.Errorf("spec components = %v, want them unchanged by a caller's slice [FR-080]", again)
	}
	if !(combat.DamageSpec{}).IsZero() {
		t.Errorf("the zero DamageSpec is not zero")
	}
}

// Requirement: doc/specs/attacks.md FR-107 - the field names are the snake_case words of their constants.
// Path: unhappy (invalid input).
func TestDamageProblemNames(t *testing.T) {
	t.Parallel()
	got := []string{combat.DamageComponentFieldDiceCount.String(), combat.DamageComponentFieldDie.String(), combat.DamageComponentFieldType.String(),
		combat.DamageComponentFieldAmount.String(), combat.DamageComponentField(0).String(), combat.DamageSpecFieldComponents.String(),
		combat.DamageSpecFieldAddsAbilityModifier.String(), combat.DamageSpecField(9).String(),
		combat.DamageComponentProblem{Field: combat.DamageComponentFieldDie}.Error() + combat.DamageSpecProblem{Field: combat.DamageSpecFieldComponents}.Error()}
	want := []string{"dice_count", "die", "type", "amount", "unknown", "components", "adds_ability_modifier", "unknown",
		"combat: invalid damage component: diecombat: invalid damage spec: components"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// Requirement: doc/specs/attacks.md FR-033, FR-107 and testing.md §10 - the damage constructors never panic on any numbers.
// Path: unhappy (invalid input).
func FuzzNewDiceDamage(f *testing.F) {
	for _, s := range [][3]int{{1, 6, 0}, {0, 0, 0}, {500, 100, 12}, {501, -1, -1}, {math.MaxInt, math.MaxInt, 99}} {
		f.Add(s[0], s[1], s[2])
	}
	f.Fuzz(func(t *testing.T, count, sides, typ int) {
		die, err := dice.NewDie(sides)
		if err != nil {
			die = dice.Die{}
		}
		if d, err := combat.NewDiceDamage(count, die, content.DamageType(typ), count, true); err == nil && (d.Count() < 1 || d.Count() > 500) {
			t.Errorf("accepted %d dice", d.Count())
		}
		if d, err := combat.NewFixedDamage(count, content.DamageType(typ), sides, false); err == nil && d.Amount() < 0 {
			t.Errorf("accepted a negative amount %d", d.Amount())
		}
		_, _ = combat.NewDamageSpec(combat.DiceDamage{}, combat.FixedDamage{})
	})
}
