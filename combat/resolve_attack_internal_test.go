package combat

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func fixedFlow(faces ...int) (attackFlow, *dicetest.FixedRoller) {
	r := dicetest.NewFixedRoller(faces...)
	return attackFlow{r}, r
}

func requests(r *dicetest.FixedRoller) []string {
	var out []string
	for _, e := range r.Requests() {
		out = append(out, e.Notation())
	}
	return out
}

// Requirement: doc/specs/attacks.md FR-127 - attackFlow.resolveAttack and resolveUnarmedStrike return what the use cases'
// Run returns for equal dice; the helper holds only a roller.
// Path: happy.
func TestAttackFlowEqualsTheUseCases(t *testing.T) {
	t.Parallel()
	mods := d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2))
	spec := MustDamageSpec(MustDiceDamage(1, dice.D8(), content.DamageSlashing, 0, true))
	ac, err := NewResolveAttackCommand(ResolveAttackParams{Form: MeleeWeaponAttack, ArmorClass: 15, Modifiers: mods, Distance: 5, Damage: spec})
	if err != nil {
		t.Fatal(err)
	}
	flow, _ := fixedFlow(14, 6)
	got, err := flow.resolveAttack(ac)
	uc, _ := NewResolveAttack(dicetest.NewFixedRoller(14, 6))
	want, err2 := uc.Run(context.Background(), ac)
	if err != nil || err2 != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("resolveAttack = %v, %v; Run = %v, %v", got, err, want, err2)
	}
	for _, o := range []UnarmedStrikeOption{
		UnarmedStrikeDamageOption{ArmorClass: 15},
		UnarmedStrikeGrappleOption{SaveAbility: ability.Strength, Attempt: d20.RollAttempt{Modifiers: mods}},
		UnarmedStrikeShoveOption{Effect: GiveProne, SaveAbility: ability.Dexterity, Attempt: d20.ChooseToFail{}},
	} {
		cmd, err := NewResolveUnarmedStrikeCommand(ResolveUnarmedStrikeParams{Option: o, AbilityModifier: ability.MustScore(16).Modifier(),
			ProficiencyBonus: character.ProficiencyBonusFor(character.MustLevel(1)), AttackerSize: content.SizeMedium, TargetSize: content.SizeMedium,
			Hand: HandAvailabilityHandFree, Distance: 5})
		if err != nil {
			t.Fatal(err)
		}
		flow, _ := fixedFlow(11)
		got, err := flow.resolveUnarmedStrike(cmd)
		uc, _ := NewResolveUnarmedStrike(dicetest.NewFixedRoller(11))
		want, err2 := uc.Run(context.Background(), cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%T: resolveUnarmedStrike = %v, %v; Run = %v, %v", o, got, err, want, err2)
		}
	}
}

// Requirement: doc/specs/attacks.md FR-133 - ResolveSave and ResolveCheck over one routine: a RollAttempt resolves the mode
// once, draws that many d20s and compares with d20.ResolveCheck of the right kind; ChooseToFail and AutomaticFailure draw no
// die and return the forced failure; no modifier is added.
// Path: per case.
func TestAttackFlowSaveAndCheck(t *testing.T) {
	t.Parallel()
	mods := d20.NewRollModifiers(modifier.New(2)).WithOthers(modifier.New(1))
	cases := []struct {
		name, requirement, path string
		run                     func(attackFlow) (UnarmedStrikeSave, error)
		faces                   []int
		requests                []string
		kind                    d20.Kind
		natural, total          int
		forced                  d20.ForcedFailureReason
	}{
		{"save with Disadvantage keeps the lower of two d20", "FR-133", "unhappy (rule interaction)",
			func(f attackFlow) (UnarmedStrikeSave, error) {
				return f.resolveSave(d20.RollAttempt{Modifiers: mods, Circumstances: []modifier.RollCircumstance{modifier.Disadvantage{}}}, 12)
			}, []int{15, 6}, []string{"2d20"}, d20.SavingThrow, 6, 9, 0},
		{"check, normal, is one d20 of kind AbilityCheck", "FR-133", "happy",
			func(f attackFlow) (UnarmedStrikeSave, error) {
				return f.resolveCheck(d20.RollAttempt{Modifiers: mods}, 12)
			}, []int{9}, []string{"1d20"}, d20.AbilityCheck, 9, 12, 0},
		{"save chosen to fail", "FR-133, FR-095", "unhappy (rule interaction)",
			func(f attackFlow) (UnarmedStrikeSave, error) { return f.resolveSave(d20.ChooseToFail{}, 12) },
			nil, nil, d20.SavingThrow, 0, 0, d20.ChosenToFail},
		{"save automatic failure", "FR-133", "unhappy (rule interaction)",
			func(f attackFlow) (UnarmedStrikeSave, error) { return f.resolveSave(d20.AutomaticFailure{}, 12) },
			nil, nil, d20.SavingThrow, 0, 0, d20.Automatic},
		{"check automatic failure", "FR-133", "unhappy (rule interaction)",
			func(f attackFlow) (UnarmedStrikeSave, error) { return f.resolveCheck(d20.AutomaticFailure{}, 12) },
			nil, nil, d20.AbilityCheck, 0, 0, d20.Automatic},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			flow, r := fixedFlow(c.faces...)
			got, err := c.run(flow)
			if err != nil || !reflect.DeepEqual(requests(r), c.requests) {
				t.Fatalf("= %v, %v requests %v, want requests %v [%s]", got, err, requests(r), c.requests, c.requirement)
			}
			switch v := got.(type) {
			case UnarmedStrikeSaveRolled:
				if v.Result().Kind() != c.kind || v.Result().Roll().Natural() != c.natural || v.Result().Total() != c.total || !v.Result().Modifiers().Equal(mods) {
					t.Errorf("check = %+v, want kind %v natural %d total %d with the attempt's modifiers only [%s]", v.Result(), c.kind, c.natural, c.total, c.requirement)
				}
			case UnarmedStrikeSaveForced:
				if v.Result().Kind() != c.kind || v.Result().Reason() != c.forced {
					t.Errorf("forced = %+v, want %v/%v [%s]", v.Result(), c.kind, c.forced, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-133, FR-081 - an attempt that is none of the known types is an ErrInvalidInput error
// with no die, and a failing roller is a *srderr.PortError for the dice port.
// Path: unhappy (invalid input, failure of collaborator).
func TestAttackFlowRejectsWhatItCannotResolve(t *testing.T) {
	t.Parallel()
	flow, r := fixedFlow(10)
	if _, err := flow.resolveSave(nil, 10); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("resolveSave(nil) error = %v, want ErrInvalidInput", err)
	}
	if _, err := flow.resolveCheck(nil, 10); !errors.Is(err, srderr.ErrInvalidInput) || len(r.Requests()) != 0 {
		t.Errorf("resolveCheck(nil) error = %v requests %v, want ErrInvalidInput and no die", err, r.Requests())
	}
	cause := errors.New("down")
	failing := attackFlow{dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })}
	var pe *srderr.PortError
	if _, err := failing.resolveSave(d20.RollAttempt{}, 10); !errors.As(err, &pe) || !errors.Is(err, cause) {
		t.Errorf("resolveSave with a failing roller error = %v, want a *PortError wrapping the cause", err)
	}
	if _, err := failing.resolveCheck(d20.RollAttempt{}, 10); !errors.As(err, &pe) {
		t.Errorf("resolveCheck with a failing roller error = %v, want a *PortError", err)
	}
}
