package combat_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attacks.md FR-132, FR-033, FR-046, FR-101, FR-134 - the attack command constructor returns the
// joined problems of every violated field, in order, and an error if and only if a field is violated.
// Path: per case.
func TestNewResolveAttackCommand(t *testing.T) {
	t.Parallel()
	dist, improv, spell, crit := combat.ResolveAttackFieldDistance, combat.ResolveAttackFieldImprovisedProficiencyBonus,
		combat.ResolveAttackFieldSpellAttackProficiencyBonus, combat.ResolveAttackFieldCriticalThreshold
	form, rng := combat.ResolveAttackFieldForm, combat.ResolveAttackFieldRange
	improvised, _ := combat.NewWeaponUse(0, 0, nil, combat.WieldedAsDesigned)
	martial, _ := combat.NewWeaponUse(content.WeaponMartial, content.WeaponMelee, nil, combat.WieldedAsDesigned)
	with := func(edit func(*combat.ResolveAttackParams)) combat.ResolveAttackParams {
		p := meleeParams()
		edit(&p)
		return p
	}
	cases := []struct {
		name, requirement, path string
		p                       combat.ResolveAttackParams
		want                    []combat.ResolveAttackField
	}{
		{"a valid melee attack", "FR-132", "happy", meleeParams(), nil},
		{"a valid ranged attack", "FR-132", "happy", rangedParams(), nil},
		{"distance 0", "FR-033", "unhappy (boundary)", with(func(p *combat.ResolveAttackParams) { p.Distance = 0 }), nil},
		{"negative distance", "FR-033", "unhappy (boundary)", with(func(p *combat.ResolveAttackParams) { p.Distance = -1 }), []combat.ResolveAttackField{dist}},
		{"improvised weapon with a Proficiency Bonus", "FR-046", "unhappy (invalid input)", with(func(p *combat.ResolveAttackParams) { p.Weapon = improvised }), []combat.ResolveAttackField{improv}},
		{"improvised weapon without one", "FR-046", "happy", with(func(p *combat.ResolveAttackParams) { p.Weapon = improvised; p.Modifiers = mods(3, 0) }), nil},
		{"non-improvised weapon with one", "FR-100", "happy", with(func(p *combat.ResolveAttackParams) { p.Weapon = martial }), nil},
		{"no weapon use supplied, with one", "FR-050", "happy", meleeParams(), nil},
		{"spell attack without a Proficiency Bonus", "FR-101", "unhappy (invalid input)", with(func(p *combat.ResolveAttackParams) { p.Form = combat.MeleeSpellAttack; p.Modifiers = mods(3, 0) }), []combat.ResolveAttackField{spell}},
		{"spell attack with +3", "FR-101", "happy", with(func(p *combat.ResolveAttackParams) { p.Form = combat.MeleeSpellAttack; p.Modifiers = mods(3, 3) }), nil},
		{"threshold 18", "FR-134", "unhappy (boundary)", with(func(p *combat.ResolveAttackParams) { p.CriticalThreshold = 18 }), nil},
		{"threshold 20", "FR-134", "unhappy (boundary)", with(func(p *combat.ResolveAttackParams) { p.CriticalThreshold = 20 }), nil},
		{"threshold 17", "FR-134", "unhappy (boundary)", with(func(p *combat.ResolveAttackParams) { p.CriticalThreshold = 17 }), []combat.ResolveAttackField{crit}},
		{"threshold 21", "FR-134", "unhappy (boundary)", with(func(p *combat.ResolveAttackParams) { p.CriticalThreshold = 21 }), []combat.ResolveAttackField{crit}},
		{"zero form", "FR-087", "unhappy (invalid input)", with(func(p *combat.ResolveAttackParams) { p.Form = 0 }), []combat.ResolveAttackField{form}},
		{"ranged form without a range", "FR-033", "unhappy (invalid input)", with(func(p *combat.ResolveAttackParams) { p.Form = combat.RangedWeaponAttack }), []combat.ResolveAttackField{rng}},
		{"every field wrong", "FR-132", "unhappy (invalid input)", combat.ResolveAttackParams{Form: combat.RangedSpellAttack, Distance: -1, CriticalThreshold: 3, Modifiers: mods(0, 0)},
			[]combat.ResolveAttackField{dist, spell, crit, rng}},
		{"the zero params", "FR-132", "unhappy (empty and minimal case)", combat.ResolveAttackParams{}, []combat.ResolveAttackField{form}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewResolveAttackCommand(c.p)
			if len(c.want) == 0 {
				if err != nil || cmd.Form() != c.p.Form {
					t.Errorf("NewResolveAttackCommand = %v, %v, want a command [%s]", cmd, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || cmd.Form() != 0 {
				t.Fatalf("NewResolveAttackCommand = %v, %v, want the zero command and ErrInvalidInput [%s]", cmd, err, c.requirement)
			}
			for _, field := range []combat.ResolveAttackField{dist, improv, spell, crit, form, rng} {
				has := false
				for _, w := range c.want {
					has = has || w == field
				}
				if errors.Is(err, combat.ResolveAttackProblem{Field: field}) != has {
					t.Errorf("problem %v present = %v, want %v [%s]", field, !has, has, c.requirement)
				}
			}
			// The problems come in the order of the field constants.
			if joined, ok := err.(interface{ Unwrap() []error }); ok {
				var order []combat.ResolveAttackField
				for _, e := range joined.Unwrap() {
					var pr combat.ResolveAttackProblem
					errors.As(e, &pr)
					order = append(order, pr.Field)
				}
				if !reflect.DeepEqual(order, c.want) {
					t.Errorf("problems = %v, want %v in this order [%s]", order, c.want, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-132 - MustResolveAttackCommand panics with the constructor's error and only then;
// the field names are snake_case.
// Path: unhappy (invalid input).
func TestMustResolveAttackCommand(t *testing.T) {
	t.Parallel()
	if got := combat.MustResolveAttackCommand(meleeParams()).Form(); got != combat.MeleeWeaponAttack {
		t.Errorf("Form = %v", got)
	}
	defer func() {
		if err, _ := recover().(error); !errors.Is(err, combat.ResolveAttackProblem{Field: combat.ResolveAttackFieldForm}) {
			t.Errorf("panic = %v, want the form problem", err)
		}
	}()
	names := []string{combat.ResolveAttackFieldDistance.String(), combat.ResolveAttackFieldImprovisedProficiencyBonus.String(),
		combat.ResolveAttackFieldSpellAttackProficiencyBonus.String(), combat.ResolveAttackFieldCriticalThreshold.String(),
		combat.ResolveAttackFieldForm.String(), combat.ResolveAttackFieldRange.String(), combat.ResolveAttackField(0).String(),
		combat.ResolveAttackProblem{Field: combat.ResolveAttackFieldRange}.Error()}
	want := []string{"distance", "improvised_proficiency_bonus", "spell_attack_proficiency_bonus", "critical_threshold", "form", "range",
		"unknown", "combat: invalid attack command: range"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
	combat.MustResolveAttackCommand(combat.ResolveAttackParams{})
}

// Requirement: doc/specs/attacks.md FR-080, FR-107 - the command copies the caller's slices, so changing them afterwards
// changes nothing.
// Path: unhappy (immutability).
func TestAttackCommandCopiesItsSlices(t *testing.T) {
	t.Parallel()
	p := meleeParams()
	p.Cover = []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	p.Circumstances = []modifier.RollCircumstance{disadvantage}
	cmd := attackCommand(t, p)
	p.Cover[0].Cover = combat.TotalCover
	p.Circumstances[0] = advantage
	res, _ := runAttack(t, cmd, 10, 10)
	r, ok := res.(combat.AttackResolved)
	if !ok || r.Cover() != combat.HalfCover || r.Attack().Roll().Mode() != modifier.WithDisadvantage {
		t.Errorf("= %#v, want Half Cover and Disadvantage as first given", res)
	}
}

// Requirement: doc/specs/attacks.md FR-135 - WithAdditions appends circumstances, appends modifiers to the others, ORs the
// automatic Critical Hit, carries every other field and leaves the original alone; it returns an error where the
// constructor would.
// Path: per case.
func TestResolveAttackCommandWithAdditions(t *testing.T) {
	t.Parallel()
	base := meleeParams()
	base.Circumstances = []modifier.RollCircumstance{advantage}
	base.CriticalThreshold = 19
	base.Damage = slashing(1, 8)
	cmd := attackCommand(t, base)
	got, err := cmd.WithAdditions(combat.Additions{Circumstances: []modifier.RollCircumstance{disadvantage},
		Modifiers: []modifier.Modifier{modifier.New(1), modifier.New(-4)}, AutomaticCritical: combat.AutomaticCriticalAutomatic})
	if err != nil {
		t.Fatal(err)
	}
	// 19 kept on the two d20s [adv, dis cancel to one]: a hit and a Critical Hit; the roll total is 3+2+1-4 = 2 plus the die.
	res, reqs := runAttack(t, got, 19, 5, 6)
	r := res.(combat.AttackResolved)
	if !reflect.DeepEqual(reqs, []string{"1d20", "2d8"}) || !r.CriticalHit() || r.Attack().Total() != 21 ||
		r.Attack().Modifiers().Count() != 4 || !r.Attack().Modifiers().Equal(mods(3, 2, 1, -4)) ||
		!reflect.DeepEqual(r.Circumstances(), []modifier.RollCircumstance{advantage, disadvantage}) || r.Attack().CriticalRange() != 19 {
		t.Errorf("= %v with requests %v, want one 1d20 (cancelled), the added modifiers and circumstance, an automatic Critical Hit, threshold 19", r, reqs)
	}
	// The original is untouched.
	orig, reqs := runAttack(t, cmd, 18, 5, 4)
	if o := orig.(combat.AttackResolved); o.Attack().Modifiers().Count() != 2 || o.CriticalHit() || len(o.Circumstances()) != 1 || len(reqs) != 2 {
		t.Errorf("original = %v, want it unchanged", o)
	}
	// Automatic is never cleared and zero additions add nothing.
	auto := base
	auto.AutomaticCritical = combat.AutomaticCriticalAutomatic
	same, err := attackCommand(t, auto).WithAdditions(combat.Additions{AutomaticCritical: combat.AutomaticCriticalNotAutomatic})
	if res, _ := runAttack(t, same, 12, 5, 6, 6); err != nil || !res.(combat.AttackResolved).CriticalHit() {
		t.Errorf("an addition of NotAutomatic cleared the automatic Critical Hit: %v, %v", res, err)
	}
	// The additions' slices are copied.
	adds := []modifier.Modifier{modifier.New(2)}
	c2, _ := cmd.WithAdditions(combat.Additions{Modifiers: adds})
	adds[0] = modifier.New(50)
	if res, _ := runAttack(t, c2, 10, 1, 1); res.(combat.AttackResolved).Attack().Total() != 17 {
		t.Errorf("total = %d, want 10+3+2+2", res.(combat.AttackResolved).Attack().Total())
	}
	// An invalid command is an error, as the constructor's.
	if _, err := (combat.ResolveAttackCommand{}).WithAdditions(combat.Additions{}); !errors.Is(err, combat.ResolveAttackProblem{Field: combat.ResolveAttackFieldForm}) {
		t.Errorf("WithAdditions on the zero command error = %v, want the form problem", err)
	}
}

// Requirement: doc/specs/attacks.md FR-132 and testing.md §10 - NewResolveAttackCommand never panics and accepts exactly a
// valid form, a non-negative distance and a threshold of 0 or 18 to 20.
// Path: unhappy (invalid input).
func FuzzNewResolveAttackCommand(f *testing.F) {
	f.Add(1, 5, 20, 3, true)
	f.Add(0, -1, 17, 0, false)
	f.Add(4, math.MaxInt, 21, -9, true)
	f.Fuzz(func(t *testing.T, form, distance, threshold, abilityMod int, hasPB bool) {
		pb := 0
		if hasPB {
			pb = 2
		}
		p := combat.ResolveAttackParams{Form: combat.AttackForm(form), Distance: distance, CriticalThreshold: threshold,
			Modifiers: mods(abilityMod, pb), Range: combat.MustAttackRange(30, 0)}
		cmd, err := combat.NewResolveAttackCommand(p)
		valid := combat.AttackForm(form).Valid() && distance >= 0 && (threshold == 0 || (threshold >= 18 && threshold <= 20)) &&
			(combat.AttackForm(form).Source() != combat.AttackSourceSpell || hasPB)
		if (err == nil) != valid || (err == nil && cmd.Form() != p.Form) {
			t.Errorf("NewResolveAttackCommand(%+v) = %v, %v, want valid %v", p, cmd, err, valid)
		}
	})
}
