package content_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: spell-effect-data.md FR-044, FR-027, table E10 - a damage type
// selection is fixed or a set of at least two distinct types, in any order.
// Path: per case.
func TestDamageTypeSelection(t *testing.T) {
	t.Parallel()
	_, err := content.NewFixedDamageType(0)
	wantProblem(t, err, "FixedDamageType", content.BlockInvalidMember)
	if f, err := content.NewFixedDamageType(content.DamageForce); err != nil || f.Type() != content.DamageForce {
		t.Errorf("NewFixedDamageType(Force) = %v, %v", f, err)
	}
	for _, in := range [][]content.DamageType{nil, {content.DamageFire}} {
		_, err := content.NewChosenDamageType(in)
		wantProblem(t, err, "ChosenDamageType", content.BlockOutOfRange)
	}
	_, err = content.NewChosenDamageType([]content.DamageType{content.DamageFire, content.DamageFire})
	wantProblem(t, err, "ChosenDamageType", content.BlockDuplicate)
	_, err = content.NewChosenDamageType([]content.DamageType{content.DamageFire, 99})
	wantProblem(t, err, "ChosenDamageType", content.BlockInvalidMember)
	a := must(content.NewChosenDamageType([]content.DamageType{content.DamageFire, content.DamageCold}))
	b := must(content.NewChosenDamageType([]content.DamageType{content.DamageCold, content.DamageFire}))
	if a != b || !slices.Equal(a.Types(), []content.DamageType{content.DamageCold, content.DamageFire}) {
		t.Errorf("sets %v and %v differ or are not in vocabulary order", a.Types(), b.Types())
	}
}

// Requirement: spell-effect-data.md FR-046 - a condition duration is closed;
// a timed duration has an amount of at least 1 and a unit.
// Path: unhappy (boundary).
func TestConditionDurations(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1} {
		_, err := content.NewTimedConditionDuration(n, content.DurationUnitRounds)
		wantProblem(t, err, "TimedConditionDuration", content.BlockOutOfRange)
	}
	_, err := content.NewTimedConditionDuration(1, 0)
	wantProblem(t, err, "TimedConditionDuration", content.BlockInvalidMember)
	d, err := content.NewTimedConditionDuration(1, content.DurationUnitMinutes)
	if err != nil || d.Amount() != 1 || d.Unit() != content.DurationUnitMinutes {
		t.Errorf("NewTimedConditionDuration(1, minutes) = %v, %v", d, err)
	}
	u, err := content.NewUntilNextTurnBoundary(content.OwnerCaster, content.BoundaryEnd) // Ray of Sickness, p. 158
	if err != nil || u.Owner() != content.OwnerCaster || u.Boundary() != content.BoundaryEnd {
		t.Errorf("NewUntilNextTurnBoundary = %v, %v", u, err)
	}
	for _, args := range [][2]int{{0, 1}, {1, 0}} {
		_, err := content.NewUntilNextTurnBoundary(content.TurnOwner(args[0]), content.TurnBoundary(args[1]))
		wantProblem(t, err, "UntilNextTurnBoundary", content.BlockInvalidMember)
	}
}

// Requirement: spell-effect-data.md FR-028 - a condition outcome accepts each
// of the fifteen conditions with any duration, and rejects an invalid one.
// Path: per case.
func TestConditionOutcomeAcceptsAllFifteenConditions(t *testing.T) {
	t.Parallel()
	for c := content.ConditionBlinded; c <= content.ConditionUnconscious; c++ {
		o, err := content.NewSpellAppliesCondition(c, content.ForSpellDuration{})
		if err != nil || o.Kind() != c || o.Duration() != content.ConditionDuration(content.ForSpellDuration{}) {
			t.Errorf("NewSpellAppliesCondition(%v) = %v, %v", c, o, err)
		}
	}
	_, err := content.NewSpellAppliesCondition(0, content.ForSpellDuration{})
	wantProblem(t, err, "SpellAppliesCondition", content.BlockInvalidMember)
	_, err = content.NewSpellAppliesCondition(content.ConditionPoisoned, nil)
	wantProblem(t, err, "ConditionDuration", content.BlockMissing)
	_, err = content.NewSpellAppliesCondition(content.ConditionPoisoned, content.TimedConditionDuration{})
	wantProblem(t, err, "TimedConditionDuration", content.BlockOutOfRange)
}

// Requirement: spell-effect-data.md FR-047 - the four outcome variants hold
// their fields; an invalid amount, selection or addition is rejected.
// Path: per case.
func TestOutcomeVariants(t *testing.T) {
	t.Parallel()
	a := amount(rolled("2d8"), nil)
	d := must(content.NewSpellDealsDamage(a, fixedType(content.DamageFire), content.AddsSpellcastingAbilityModifier))
	if d.Amount() != a || d.Type() != content.DamageTypeSelection(fixedType(content.DamageFire)) || d.AddsModifier() != content.AddsSpellcastingAbilityModifier {
		t.Errorf("SpellDealsDamage = %v", d)
	}
	r := must(content.NewSpellRestoresHitPoints(a, content.AddsSpellcastingAbilityModifier))
	g := must(content.NewSpellGrantsTemporaryHitPoints(a, content.AddsNothing))
	if r.Amount() != a || r.AddsModifier() != content.AddsSpellcastingAbilityModifier || g.Amount() != a || g.AddsModifier() != content.AddsNothing {
		t.Errorf("healing outcomes = %v, %v", r, g)
	}
	var zero content.ScalableAmount
	_, err := content.NewSpellDealsDamage(zero, fixedType(content.DamageFire), content.AddsNothing)
	wantProblem(t, err, "SpellAmount", content.BlockMissing)
	_, err = content.NewSpellDealsDamage(a, nil, content.AddsNothing)
	wantProblem(t, err, "DamageTypeSelection", content.BlockMissing)
	_, err = content.NewSpellDealsDamage(a, content.FixedDamageType{}, content.AddsNothing)
	wantProblem(t, err, "FixedDamageType", content.BlockInvalidMember)
	_, err = content.NewSpellDealsDamage(a, fixedType(content.DamageFire), 0)
	wantProblem(t, err, "SpellDealsDamage", content.BlockInvalidMember)
	_, err = content.NewSpellRestoresHitPoints(a, 0)
	wantProblem(t, err, "SpellRestoresHitPoints", content.BlockInvalidMember)
	_, err = content.NewSpellGrantsTemporaryHitPoints(a, 0)
	wantProblem(t, err, "SpellGrantsTemporaryHitPoints", content.BlockInvalidMember)
	for i, m := range []content.ModifierAddition{content.AddsNothing, content.AddsSpellcastingAbilityModifier, 0} {
		if m.Valid() != (i < 2) {
			t.Errorf("ModifierAddition(%d).Valid() = %v", m, m.Valid())
		}
	}
}

// Requirement: spell-effect-data.md FR-042, FR-026 - the three resolutions
// hold their fields; the outcome lists that cannot be empty are rejected when
// empty; OnMiss may be empty.
// Path: per case.
func TestResolutionVariantsAndEmptyLists(t *testing.T) {
	t.Parallel()
	dmg := damageOf(amount(rolled("1d10"), nil), fixedType(content.DamageFire))
	miss := damageOf(amount(rolled("1d4"), nil), fixedType(content.DamageFire))
	hit := []content.SpellOutcome{dmg}

	ar, err := content.NewAttackRollResolution(content.SpellAttackMelee, hit, []content.SpellOutcome{miss})
	if err != nil || ar.Kind() != content.SpellAttackMelee || !slices.Equal(ar.OnHit(), hit) || len(ar.OnMiss()) != 1 {
		t.Errorf("attack = %v, %v", ar, err)
	}
	if _, err := content.NewAttackRollResolution(content.SpellAttackRanged, hit, nil); err != nil { // OnMiss may be empty
		t.Errorf("empty OnMiss: %v", err)
	}
	_, err = content.NewAttackRollResolution(content.SpellAttackRanged, nil, nil)
	wantProblem(t, err, "AttackRollResolution", content.BlockEmpty)
	_, err = content.NewAttackRollResolution(0, hit, nil)
	wantProblem(t, err, "AttackRollResolution", content.BlockInvalidMember)
	_, err = content.NewAttackRollResolution(content.SpellAttackRanged, hit, []content.SpellOutcome{nil})
	wantProblem(t, err, "SpellOutcome", content.BlockMissing)

	sv, err := content.NewSavingThrowResolution(ability.Wisdom, hit, content.NothingOnSuccess)
	if err != nil || sv.Ability() != ability.Wisdom || sv.OnSuccess() != content.NothingOnSuccess || !slices.Equal(sv.OnFailure(), hit) {
		t.Errorf("save = %v, %v", sv, err)
	}
	_, err = content.NewSavingThrowResolution(ability.Wisdom, nil, content.NothingOnSuccess)
	wantProblem(t, err, "SavingThrowResolution", content.BlockEmpty)
	_, err = content.NewSavingThrowResolution(0, hit, content.NothingOnSuccess)
	wantProblem(t, err, "SavingThrowResolution", content.BlockInvalidMember)
	_, err = content.NewSavingThrowResolution(ability.Wisdom, hit, 0)
	wantProblem(t, err, "SavingThrowResolution", content.BlockInvalidMember)

	au, err := content.NewAutomaticResolution(hit)
	if err != nil || !slices.Equal(au.Outcomes(), hit) {
		t.Errorf("automatic = %v, %v", au, err)
	}
	_, err = content.NewAutomaticResolution(nil)
	wantProblem(t, err, "AutomaticResolution", content.BlockEmpty)
	_, err = content.NewAutomaticResolution([]content.SpellOutcome{})
	wantProblem(t, err, "AutomaticResolution", content.BlockEmpty)
}

// Requirement: spell-effect-data.md FR-043 - HalfDamageOnSuccess needs a
// SpellDealsDamage among the failure outcomes.
// Path: unhappy (invalid input).
func TestHalfDamageNeedsDamage(t *testing.T) {
	t.Parallel()
	condition := must(content.NewSpellAppliesCondition(content.ConditionParalyzed, content.ForSpellDuration{}))
	heal := must(content.NewSpellRestoresHitPoints(amount(rolled("1d4"), nil), content.AddsNothing))
	dmg := damageOf(amount(rolled("1d4"), nil), fixedType(content.DamageCold))
	for _, fail := range [][]content.SpellOutcome{{condition}, {heal, condition}} {
		_, err := content.NewSavingThrowResolution(ability.Constitution, fail, content.HalfDamageOnSuccess)
		wantProblem(t, err, "SavingThrowResolution", content.BlockConstraint)
	}
	if _, err := content.NewSavingThrowResolution(ability.Constitution, []content.SpellOutcome{condition, dmg}, content.HalfDamageOnSuccess); err != nil {
		t.Errorf("damage among the failures: %v", err)
	}
	if _, err := content.NewSavingThrowResolution(ability.Constitution, []content.SpellOutcome{condition}, content.NothingOnSuccess); err != nil {
		t.Errorf("NothingOnSuccess with no damage: %v", err)
	}
}

// Requirement: spell-effect-data.md FR-007 - outcome lists are copied on
// construction and on read.
// Path: unhappy (immutability).
func TestOutcomeListsAreCopied(t *testing.T) {
	t.Parallel()
	dmg := damageOf(amount(rolled("1d10"), nil), fixedType(content.DamageFire))
	other := damageOf(amount(rolled("9d10"), nil), fixedType(content.DamageFire))
	in := []content.SpellOutcome{dmg}
	r := must(content.NewAttackRollResolution(content.SpellAttackRanged, in, in))
	p := part(r, once())
	in[0] = other
	r.OnHit()[0], r.OnMiss()[0] = other, other
	got := p.Resolution().(content.AttackRollResolution)
	got.OnHit()[0] = other
	if !slices.Equal(got.OnHit(), []content.SpellOutcome{dmg}) || !slices.Equal(got.OnMiss(), []content.SpellOutcome{dmg}) {
		t.Error("a list changed after the caller mutated the input or a returned slice [FR-007]")
	}
	au := must(content.NewAutomaticResolution(in))
	in[0] = dmg
	au.Outcomes()[0] = other
	if !slices.Equal(au.Outcomes(), []content.SpellOutcome{other}) {
		t.Errorf("AutomaticResolution outcomes = %v, want the first input to be kept", au.Outcomes())
	}
	sv := must(content.NewSavingThrowResolution(ability.Dexterity, []content.SpellOutcome{dmg}, content.NothingOnSuccess))
	sv.OnFailure()[0] = other
	if sv.OnFailure()[0] != content.SpellOutcome(dmg) {
		t.Error("SavingThrowResolution.OnFailure returned its own slice")
	}
}

// Requirement: spell-effect-data.md FR-042 - a part holds a valid resolution
// and valid instances; equality compares both.
// Path: unhappy (invalid input).
func TestSpellEffectPart(t *testing.T) {
	t.Parallel()
	_, err := content.NewSpellEffectPart(nil, once())
	wantProblem(t, err, "PartResolution", content.BlockMissing)
	_, err = content.NewSpellEffectPart(content.AutomaticResolution{}, once())
	wantProblem(t, err, "AutomaticResolution", content.BlockEmpty)
	_, err = content.NewSpellEffectPart(content.AttackRollResolution{}, once())
	wantProblem(t, err, "AttackRollResolution", content.BlockInvalidMember)
	_, err = content.NewSpellEffectPart(content.SavingThrowResolution{}, once())
	wantProblem(t, err, "SavingThrowResolution", content.BlockInvalidMember)
	_, err = content.NewSpellEffectPart(automatic(damageOf(amount(rolled("1d4"), nil), fixedType(content.DamageCold))), content.PartInstances{})
	wantProblem(t, err, "PartInstances", content.BlockOutOfRange)

	dmg := damageOf(amount(rolled("1d4"), nil), fixedType(content.DamageCold))
	a := part(automatic(dmg), once())
	if !a.Equal(part(automatic(dmg), once())) {
		t.Error("equal parts are not equal")
	}
	three := must(content.NewPartInstances(3, nil))
	other := damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageCold))
	for name, o := range map[string]content.SpellEffectPart{
		"instances":  part(automatic(dmg), three),
		"outcome":    part(automatic(other), once()),
		"resolution": part(attack(dmg), once()),
		"save":       part(save(ability.Dexterity, content.NothingOnSuccess, dmg), once()),
	} {
		if a.Equal(o) || o.Equal(a) {
			t.Errorf("parts differing in %s are equal", name)
		}
	}
	if a.Instances() != once() {
		t.Errorf("Instances() = %v", a.Instances())
	}
	atk, sv := part(attack(dmg), once()), part(save(ability.Dexterity, content.NothingOnSuccess, dmg), once())
	if !atk.Equal(part(attack(dmg), once())) || !sv.Equal(part(save(ability.Dexterity, content.NothingOnSuccess, dmg), once())) {
		t.Error("equal attack or save parts are not equal")
	}
	if atk.Equal(part(must(content.NewAttackRollResolution(content.SpellAttackMelee, []content.SpellOutcome{dmg}, nil)), once())) {
		t.Error("melee and ranged parts are equal")
	}
	if sv.Equal(part(save(ability.Wisdom, content.NothingOnSuccess, dmg), once())) || sv.Equal(part(save(ability.Dexterity, content.HalfDamageOnSuccess, dmg), once())) {
		t.Error("saves with another ability or success effect are equal")
	}
	var zero content.SpellEffectPart
	if !zero.Equal(content.SpellEffectPart{}) || zero.Equal(a) {
		t.Error("zero part equality is wrong")
	}
}
