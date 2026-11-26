package combat_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attacks.md FR-132, FR-033, FR-134 - the Unarmed Strike command constructor returns the joined
// problems of every violated field, in order, and an error if and only if a field is violated; a Wisdom save is one.
// Path: per case.
func TestNewResolveUnarmedStrikeCommand(t *testing.T) {
	t.Parallel()
	dist, crit, save := combat.ResolveUnarmedStrikeFieldDistance, combat.ResolveUnarmedStrikeFieldCriticalThreshold, combat.ResolveUnarmedStrikeFieldSaveAbility
	opt, attempt, effect := combat.ResolveUnarmedStrikeFieldOption, combat.ResolveUnarmedStrikeFieldAttempt, combat.ResolveUnarmedStrikeFieldEffect
	size, pb := combat.ResolveUnarmedStrikeFieldSize, combat.ResolveUnarmedStrikeFieldProficiencyBonus
	damage := func(threshold int) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeDamageOption{ArmorClass: 15, CriticalThreshold: threshold}
	}
	grapple := func(a ability.Ability, at d20.SaveAttempt) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeGrappleOption{SaveAbility: a, Attempt: at}
	}
	shove := func(e combat.ShoveEffect, a ability.Ability, at d20.SaveAttempt) combat.UnarmedStrikeOption {
		return combat.UnarmedStrikeShoveOption{Effect: e, SaveAbility: a, Attempt: at}
	}
	with := func(o combat.UnarmedStrikeOption, edit func(*combat.ResolveUnarmedStrikeParams)) combat.ResolveUnarmedStrikeParams {
		p := unarmedParams(o)
		if edit != nil {
			edit(&p)
		}
		return p
	}
	cases := []struct {
		name, requirement, path string
		p                       combat.ResolveUnarmedStrikeParams
		want                    []combat.ResolveUnarmedStrikeField
	}{
		{"Damage option", "FR-132", "happy", with(damage(0), nil), nil},
		{"Damage option, threshold 18", "FR-134", "unhappy (boundary)", with(damage(18), nil), nil},
		{"Damage option, threshold 17", "FR-134", "unhappy (boundary)", with(damage(17), nil), []combat.ResolveUnarmedStrikeField{crit}},
		{"Damage option, threshold 21", "FR-134", "unhappy (boundary)", with(damage(21), nil), []combat.ResolveUnarmedStrikeField{crit}},
		{"Grapple with Strength", "FR-132", "happy", with(grapple(ability.Strength, rollOf(0)), nil), nil},
		{"Grapple with Dexterity", "FR-132", "happy", with(grapple(ability.Dexterity, d20.ChooseToFail{}), nil), nil},
		{"Grapple with Wisdom", "FR-132, FR-128", "unhappy (invalid input)", with(grapple(ability.Wisdom, rollOf(0)), nil), []combat.ResolveUnarmedStrikeField{save}},
		{"Grapple without an attempt", "FR-094", "unhappy (invalid input)", with(grapple(ability.Strength, nil), nil), []combat.ResolveUnarmedStrikeField{attempt}},
		{"Shove pushing", "FR-073", "happy", with(shove(combat.PushFiveFeet, ability.Strength, rollOf(0)), nil), nil},
		{"Shove giving Prone", "FR-073", "happy", with(shove(combat.GiveProne, ability.Dexterity, d20.AutomaticFailure{}), nil), nil},
		{"Shove with no effect", "FR-073", "unhappy (invalid input)", with(shove(0, ability.Strength, rollOf(0)), nil), []combat.ResolveUnarmedStrikeField{effect}},
		{"negative distance", "FR-033", "unhappy (boundary)", with(damage(0), func(p *combat.ResolveUnarmedStrikeParams) { p.Distance = -1 }), []combat.ResolveUnarmedStrikeField{dist}},
		{"distance 0", "FR-033", "unhappy (boundary)", with(damage(0), func(p *combat.ResolveUnarmedStrikeParams) { p.Distance = 0 }), nil},
		{"no option", "FR-063", "unhappy (invalid input)", with(nil, nil), []combat.ResolveUnarmedStrikeField{opt}},
		{"Grapple with an unset size", "FR-078", "unhappy (invalid input)", with(grapple(ability.Strength, rollOf(0)), func(p *combat.ResolveUnarmedStrikeParams) { p.TargetSize = 0 }), []combat.ResolveUnarmedStrikeField{size}},
		{"Damage option ignores sizes", "FR-065", "unhappy (rule interaction)", with(damage(0), func(p *combat.ResolveUnarmedStrikeParams) { p.TargetSize, p.AttackerSize = 0, 0 }), nil},
		{"no Proficiency Bonus", "FR-066", "unhappy (invalid input)", with(damage(0), func(p *combat.ResolveUnarmedStrikeParams) { p.ProficiencyBonus = character.ProficiencyBonus{} }), []combat.ResolveUnarmedStrikeField{pb}},
		{"everything wrong", "FR-132", "unhappy (invalid input)", with(shove(0, ability.Wisdom, nil), func(p *combat.ResolveUnarmedStrikeParams) {
			p.Distance, p.TargetSize, p.ProficiencyBonus = -2, 0, character.ProficiencyBonus{}
		}), []combat.ResolveUnarmedStrikeField{dist, save, attempt, size, effect, pb}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewResolveUnarmedStrikeCommand(c.p)
			if len(c.want) == 0 {
				if err != nil || reflect.DeepEqual(cmd, combat.ResolveUnarmedStrikeCommand{}) {
					t.Errorf("NewResolveUnarmedStrikeCommand = %v, %v, want a command [%s]", cmd, err, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput [%s]", err, c.requirement)
			}
			var order []combat.ResolveUnarmedStrikeField
			if joined, ok := err.(interface{ Unwrap() []error }); ok {
				for _, e := range joined.Unwrap() {
					var pr combat.ResolveUnarmedStrikeProblem
					errors.As(e, &pr)
					order = append(order, pr.Field)
				}
			}
			if !reflect.DeepEqual(order, c.want) {
				t.Errorf("problems = %v, want %v in this order [%s]", order, c.want, c.requirement)
			}
			if !errors.Is(err, combat.ResolveUnarmedStrikeProblem{Field: c.want[0]}) {
				t.Errorf("errors.Is does not find %v [%s]", c.want[0], c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-132 - MustResolveUnarmedStrikeCommand panics with the constructor's error and only
// then; the field names are snake_case; the command copies the caller's slices.
// Path: unhappy (invalid input, immutability).
func TestMustResolveUnarmedStrikeCommand(t *testing.T) {
	t.Parallel()
	names := []string{combat.ResolveUnarmedStrikeFieldDistance.String(), combat.ResolveUnarmedStrikeFieldCriticalThreshold.String(),
		combat.ResolveUnarmedStrikeFieldSaveAbility.String(), combat.ResolveUnarmedStrikeFieldOption.String(), combat.ResolveUnarmedStrikeFieldAttempt.String(),
		combat.ResolveUnarmedStrikeFieldEffect.String(), combat.ResolveUnarmedStrikeFieldSize.String(), combat.ResolveUnarmedStrikeFieldProficiencyBonus.String(),
		combat.ResolveUnarmedStrikeField(0).String(), combat.ResolveUnarmedStrikeProblem{Field: combat.ResolveUnarmedStrikeFieldSize}.Error(),
		combat.PushFiveFeet.String(), combat.GiveProne.String(), combat.ShoveEffect(0).String()}
	want := []string{"distance", "critical_threshold", "save_ability", "option", "attempt", "effect", "size", "proficiency_bonus", "unknown",
		"combat: invalid unarmed strike command: size", "push_five_feet", "give_prone", "unknown"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
	cs := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	p := unarmedParams(combat.UnarmedStrikeGrappleOption{SaveAbility: ability.Dexterity, Attempt: rollOf(0)})
	p.Cover = cs
	cmd := combat.MustResolveUnarmedStrikeCommand(p)
	cs[0].Cover = combat.TotalCover
	if res, _ := runUnarmed(t, cmd, 10); res.(combat.UnarmedStrikeGrappleResolved).Cover() != combat.HalfCover {
		t.Errorf("the caller's cover slice reached the command")
	}
	defer func() {
		if err, _ := recover().(error); !errors.Is(err, combat.ResolveUnarmedStrikeProblem{Field: combat.ResolveUnarmedStrikeFieldOption}) {
			t.Errorf("panic = %v, want the option problem", err)
		}
	}()
	combat.MustResolveUnarmedStrikeCommand(combat.ResolveUnarmedStrikeParams{})
}

// Requirement: doc/specs/attacks.md FR-132 and testing.md §10 - NewResolveUnarmedStrikeCommand never panics on any numbers
// and accepts exactly a non-negative distance and a threshold of 0 or 18 to 20 for the Damage option.
// Path: unhappy (invalid input).
func FuzzNewResolveUnarmedStrikeCommand(f *testing.F) {
	f.Add(5, 20, 3)
	f.Add(-1, 17, -9)
	f.Add(math.MaxInt, 0, math.MaxInt)
	f.Fuzz(func(t *testing.T, distance, threshold, mod int) {
		p := unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: d20.ArmorClass(mod), CriticalThreshold: threshold})
		p.Distance = distance
		_, err := combat.NewResolveUnarmedStrikeCommand(p)
		valid := distance >= 0 && (threshold == 0 || (threshold >= 18 && threshold <= 20))
		if (err == nil) != valid {
			t.Errorf("distance %d threshold %d: error = %v, want valid %v", distance, threshold, err, valid)
		}
	})
}
