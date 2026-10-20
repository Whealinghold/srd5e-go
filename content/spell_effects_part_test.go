package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

func fireDice(notation string) content.SpellOutcome {
	return damageOf(amount(rolled(notation), nil), fixedType(content.DamageFire))
}

// Requirement: spell-effect-data.md FR-042 - a resolution owns its outcome
// lists: later changes to the input or to a returned list do not reach it.
// Path: unhappy (immutability).
func TestResolutionsOwnTheirOutcomes(t *testing.T) {
	t.Parallel()
	one, two := fireDice("1d6"), fireDice("2d6")

	hit, miss := []content.SpellOutcome{one}, []content.SpellOutcome{one}
	a := must(content.NewAttackRollResolution(content.SpellAttackMelee, hit, miss))
	hit[0], miss[0] = two, two
	a.OnHit()[0], a.OnMiss()[0] = two, two
	if a.OnHit()[0] != one || a.OnMiss()[0] != one {
		t.Errorf("attack outcomes changed through the input or a getter: %v, %v", a.OnHit(), a.OnMiss())
	}

	fail := []content.SpellOutcome{one}
	s := must(content.NewSavingThrowResolution(ability.Dexterity, fail, content.NothingOnSuccess))
	fail[0] = two
	s.OnFailure()[0] = two
	if s.OnFailure()[0] != one {
		t.Errorf("saving throw outcomes changed through the input or a getter: %v", s.OnFailure())
	}

	auto := []content.SpellOutcome{one}
	r := must(content.NewAutomaticResolution(auto))
	auto[0] = two
	r.Outcomes()[0] = two
	if r.Outcomes()[0] != one {
		t.Errorf("automatic outcomes changed through the input or a getter: %v", r.Outcomes())
	}
}

// Requirement: spell-effect-data.md FR-042 - resolutions are equal only when
// every field is equal, including each outcome list.
// Path: happy.
func TestResolutionEquality(t *testing.T) {
	t.Parallel()
	one, two := fireDice("1d6"), fireDice("2d6")
	newAttack := func(k content.SpellAttackKind, hit, miss []content.SpellOutcome) content.AttackRollResolution {
		return must(content.NewAttackRollResolution(k, hit, miss))
	}
	base := newAttack(content.SpellAttackMelee, []content.SpellOutcome{one}, []content.SpellOutcome{one})
	if !base.Equal(newAttack(content.SpellAttackMelee, []content.SpellOutcome{one}, []content.SpellOutcome{one})) {
		t.Error("equal attack resolutions are not equal")
	}
	for name, o := range map[string]content.AttackRollResolution{
		"kind":    newAttack(content.SpellAttackRanged, []content.SpellOutcome{one}, []content.SpellOutcome{one}),
		"on hit":  newAttack(content.SpellAttackMelee, []content.SpellOutcome{two}, []content.SpellOutcome{one}),
		"on miss": newAttack(content.SpellAttackMelee, []content.SpellOutcome{one}, []content.SpellOutcome{two}),
		"no miss": newAttack(content.SpellAttackMelee, []content.SpellOutcome{one}, nil),
	} {
		if base.Equal(o) {
			t.Errorf("attack resolutions differing in %s are equal", name)
		}
	}

	newSave := func(a ability.Ability, e content.SuccessEffect, f content.SpellOutcome) content.SavingThrowResolution {
		return must(content.NewSavingThrowResolution(a, []content.SpellOutcome{f}, e))
	}
	sv := newSave(ability.Dexterity, content.NothingOnSuccess, one)
	if !sv.Equal(newSave(ability.Dexterity, content.NothingOnSuccess, one)) {
		t.Error("equal saving throw resolutions are not equal")
	}
	for name, o := range map[string]content.SavingThrowResolution{
		"ability":    newSave(ability.Wisdom, content.NothingOnSuccess, one),
		"on success": newSave(ability.Dexterity, content.HalfDamageOnSuccess, one),
		"on failure": newSave(ability.Dexterity, content.NothingOnSuccess, two),
	} {
		if sv.Equal(o) {
			t.Errorf("saving throw resolutions differing in %s are equal", name)
		}
	}
}

// Requirement: spell-effect-data.md FR-042 - a part equals another only with
// the same instances and a resolution of the same variant and value.
// Path: happy.
func TestSpellEffectPartEquality(t *testing.T) {
	t.Parallel()
	one := fireDice("1d6")
	p := part(automatic(one), once())
	if !p.Equal(part(automatic(one), once())) {
		t.Error("equal parts are not equal")
	}
	if p.Equal(part(automatic(one), must(content.NewPartInstances(2, nil)))) {
		t.Error("parts with other instances are equal")
	}
	if p.Equal(part(attack(one), once())) || part(attack(one), once()).Equal(p) {
		t.Error("parts with other resolution variants are equal")
	}
	if part(save(ability.Dexterity, content.NothingOnSuccess, one), once()).Equal(p) {
		t.Error("a saving throw part equals an automatic part")
	}
}

// Requirement: spell-effect-data.md FR-041 - growth that fits in an int is
// exact, growth that does not is an error whichever part overflows, and a
// negative base modifier does not make small growth overflow.
// Path: unhappy (boundary).
func TestScalingArithmeticLimits(t *testing.T) {
	t.Parallel()
	maxInt := int(^uint(0) >> 1)

	edge := amount(fixed(1), perSlot(1, 0, 1))
	got, err := edge.At(maxInt-1, 1) // 1 + (maxInt-1) is exactly maxInt
	if err != nil || got != content.SpellAmount(fixed(maxInt)) {
		t.Errorf("At(maxInt-1) = %v, %v; want the fixed amount maxInt", got, err)
	}
	if _, err := edge.At(maxInt, 1); err == nil {
		t.Error("At(maxInt) = no error, want out of range")
	}

	// Only the modifier overflows: the dice part adds nothing.
	_, err = amount(rolled("1d6+1"), perSlot(1, 0, 1)).At(maxInt, 1)
	wantProblem(t, err, "ScalableAmount", content.BlockOutOfRange)

	neg := amount(rolled("1d6-3"), perSlot(1, 0, 1))
	r, err := neg.At(2, 1)
	if err != nil || r != content.SpellAmount(rolled("1d6-1")) {
		t.Errorf("1d6-3 grown by 2 = %v, %v; want 1d6-1", r, err)
	}
}
