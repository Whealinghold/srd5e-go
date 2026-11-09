package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func ally(feet int, incapacitated bool) character.AllyNearTarget {
	return must(character.NewAllyNearTarget(feet, incapacitated))
}

// Requirement: doc/specs/class-rogue.md FR-036 to FR-043, FR-045 to FR-049 (SRD 5.2.1, "Level 1: Sneak Attack", pp. 61-62; "They
// Don't Stack", p. 8) - Sneak Attack applies with Advantage and a Finesse or Ranged weapon, or without Disadvantage when an
// active ally is within 5 feet; it carries the weapon's damage type and the dice, never a total; and it does not apply, with every
// failing reason in check order, otherwise.
// Path: per case (the Sneak Attack cases of the spec): happy / unhappy (rule interaction, boundary, state error).
func TestSneakAttackEvaluate(t *testing.T) {
	t.Parallel()
	const (
		weapon = character.SneakAttackWeaponNotFinesseOrRanged
		noAdv  = character.SneakAttackNoAdvantageOrQualifyingAlly
		used   = character.SneakAttackAlreadyUsedThisTurn
		missed = character.SneakAttackAttackMissed
	)
	base := func() character.SneakAttackInput {
		return character.SneakAttackInput{Dice: dice6(1), Attack: rogAttack(t, true), Finesse: true, Mode: modifier.WithAdvantage, DamageType: content.DamageSlashing}
	}
	with := func(f func(*character.SneakAttackInput)) character.SneakAttackInput {
		in := base()
		f(&in)
		return in
	}
	for _, c := range []struct {
		name, path string
		in         character.SneakAttackInput
		dice       int
		damage     content.DamageType
		reasons    []character.SneakAttackReason
	}{
		{"Finesse with Advantage", "happy", base(), 1, content.DamageSlashing, nil},
		{"Ranged with Advantage", "happy", with(func(i *character.SneakAttackInput) { i.Finesse, i.Ranged = false, true }), 1, content.DamageSlashing, nil},
		{"Piercing damage type", "happy", with(func(i *character.SneakAttackInput) { i.DamageType = content.DamagePiercing }), 1, content.DamagePiercing, nil},
		{"straight with an active ally at 5 ft", "happy", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.Normal
			i.Allies = []character.AllyNearTarget{ally(5, false)}
		}), 1, content.DamageSlashing, nil},
		{"one ally Incapacitated, one active", "unhappy (rule interaction)", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.Normal
			i.Allies = []character.AllyNearTarget{ally(5, true), ally(5, false)}
		}), 1, content.DamageSlashing, nil},
		{"level 20 dice", "unhappy (boundary)", with(func(i *character.SneakAttackInput) { i.Dice = dice6(10) }), 10, content.DamageSlashing, nil},
		{"level 18 dice", "unhappy (boundary)", with(func(i *character.SneakAttackInput) { i.Dice = dice6(9) }), 9, content.DamageSlashing, nil},
		{"neither Finesse nor Ranged", "unhappy (rule interaction)", with(func(i *character.SneakAttackInput) { i.Finesse = false }), 0, 0, []character.SneakAttackReason{weapon}},
		{"straight and alone", "unhappy (rule interaction)", with(func(i *character.SneakAttackInput) { i.Mode = modifier.Normal }), 0, 0, []character.SneakAttackReason{noAdv}},
		{"ally at 10 ft", "unhappy (boundary)", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.Normal
			i.Allies = []character.AllyNearTarget{ally(10, false)}
		}), 0, 0, []character.SneakAttackReason{noAdv}},
		{"ally at 6 ft", "unhappy (boundary)", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.Normal
			i.Allies = []character.AllyNearTarget{ally(6, false)}
		}), 0, 0, []character.SneakAttackReason{noAdv}},
		{"ally at 0 ft", "unhappy (boundary)", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.Normal
			i.Allies = []character.AllyNearTarget{ally(0, false)}
		}), 1, content.DamageSlashing, nil},
		{"ally Incapacitated at 5 ft", "unhappy (rule interaction)", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.Normal
			i.Allies = []character.AllyNearTarget{ally(5, true)}
		}), 0, 0, []character.SneakAttackReason{noAdv}},
		{"Disadvantage with an ally at 5 ft", "unhappy (rule interaction)", with(func(i *character.SneakAttackInput) {
			i.Mode = modifier.WithDisadvantage
			i.Allies = []character.AllyNearTarget{ally(5, false)}
		}), 0, 0, []character.SneakAttackReason{noAdv}},
		{"already used this turn", "unhappy (state error)", with(func(i *character.SneakAttackInput) { i.UsedThisTurn = true }), 0, 0, []character.SneakAttackReason{used}},
		{"the attack misses", "unhappy (state error)", with(func(i *character.SneakAttackInput) { i.Attack = rogAttack(t, false) }), 0, 0, []character.SneakAttackReason{missed}},
		{"the zero attack result", "unhappy (invalid input)", with(func(i *character.SneakAttackInput) { i.Attack = d20.AttackRollResult{} }), 0, 0, []character.SneakAttackReason{missed}},
		{"all four failures", "unhappy (rule interaction)", with(func(i *character.SneakAttackInput) {
			i.Finesse, i.Mode, i.UsedThisTurn, i.Attack = false, modifier.Normal, true, rogAttack(t, false)
		}), 0, 0, []character.SneakAttackReason{weapon, noAdv, used, missed}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.SneakAttack{}.Evaluate(c.in)
			if len(c.reasons) == 0 {
				a, ok := got.(character.SneakAttackApplies)
				if !ok || a.Dice().Count() != c.dice || a.DamageType() != c.damage || a.Dice().Expression().Modifier() != 0 {
					t.Errorf("result = %#v, want applies %dd6 %v", got, c.dice, c.damage)
				}
				return
			}
			n, ok := got.(character.SneakAttackDoesNotApply)
			if !ok || !slices.Equal(n.Reasons(), c.reasons) {
				t.Errorf("result = %#v, want does not apply %v", got, c.reasons)
			}
		})
	}
}

// Requirement: doc/specs/class-rogue.md FR-043, FR-101, FR-102, FR-103 - an ally distance below 0 is refused with ErrInvalidInput; the
// reasons render as snake_case; Reasons returns a copy; Evaluate does not change its input and equal inputs give equal results.
// Path: unhappy (immutability, invalid input).
func TestSneakAttackValuesAndImmutability(t *testing.T) {
	t.Parallel()
	_, err := character.NewAllyNearTarget(-1, false)
	wantInvalid(t, err, "negative distance")
	a := ally(5, true)
	if a.DistanceFeet() != 5 || !a.Incapacitated() {
		t.Errorf("ally = %d ft, incapacitated %v", a.DistanceFeet(), a.Incapacitated())
	}
	for r, want := range map[character.SneakAttackReason]string{
		character.SneakAttackWeaponNotFinesseOrRanged: "weapon_not_finesse_or_ranged", character.SneakAttackNoAdvantageOrQualifyingAlly: "no_advantage_or_qualifying_ally",
		character.SneakAttackAlreadyUsedThisTurn: "already_used_this_turn", character.SneakAttackAttackMissed: "attack_missed", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("reason %d = %q, want %q", r, r.String(), want)
		}
	}
	allies := []character.AllyNearTarget{ally(5, false)}
	in := character.SneakAttackInput{Dice: dice6(2), Attack: rogAttack(t, false), Mode: modifier.Normal, Allies: allies, UsedThisTurn: true}
	first := character.SneakAttack{}.Evaluate(in).(character.SneakAttackDoesNotApply)
	reasons := first.Reasons()
	reasons[0] = 0
	second := character.SneakAttack{}.Evaluate(in).(character.SneakAttackDoesNotApply)
	if second.Reasons()[0] != character.SneakAttackWeaponNotFinesseOrRanged || first.Reasons()[0] != character.SneakAttackWeaponNotFinesseOrRanged || allies[0] != ally(5, false) {
		t.Errorf("reasons %v / %v, allies %v: a result or input changed", first.Reasons(), second.Reasons(), allies)
	}
}
