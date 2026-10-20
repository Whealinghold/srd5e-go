package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func rolled(notation string) content.RolledSpellAmount {
	return must(content.NewRolledSpellAmount(dice.MustParseExpression(notation)))
}

func fixed(n int) content.FixedSpellAmount { return must(content.NewFixedSpellAmount(n)) }

func amount(base content.SpellAmount, scaling content.AmountScaling) content.ScalableAmount {
	return must(content.NewScalableAmount(base, scaling))
}

func perSlot(above, extraDice, extraFlat int) content.PerSlotLevelAmount {
	return must(content.NewPerSlotLevelAmount(above, must(content.NewAmountStep(extraDice, extraFlat))))
}

func amountTable(rows ...content.AmountLevelEntry) content.ByCharacterLevelAmount {
	return must(content.NewByCharacterLevelAmount(must(content.NewAmountLevelTable(rows))))
}

func cantripDice(die string) content.ByCharacterLevelAmount {
	return amountTable(
		content.AmountLevelEntry{Level: 1, Value: rolled("1" + die)},
		content.AmountLevelEntry{Level: 5, Value: rolled("2" + die)},
		content.AmountLevelEntry{Level: 11, Value: rolled("3" + die)},
		content.AmountLevelEntry{Level: 17, Value: rolled("4" + die)},
	)
}

func damageOf(a content.ScalableAmount, t content.DamageTypeSelection) content.SpellDealsDamage {
	return must(content.NewSpellDealsDamage(a, t, content.AddsNothing))
}

func fixedType(t content.DamageType) content.FixedDamageType {
	return must(content.NewFixedDamageType(t))
}

func once() content.PartInstances { return must(content.NewPartInstances(1, nil)) }

func part(r content.PartResolution, i content.PartInstances) content.SpellEffectPart {
	return must(content.NewSpellEffectPart(r, i))
}

func exactly(n int, s content.CountScaling) content.ExactlyTargets {
	return must(content.NewExactlyTargets(n, s))
}

func chosen(subject content.TargetSubject, c content.TargetCount, r content.CreatureTypeRestriction) content.TargetsChosen {
	return must(content.NewTargetsChosen(subject, c, r, content.MayIncludeCaster))
}

func modelled(c content.EffectCoverage, t content.SpellTargeting, parts ...content.SpellEffectPart) content.ModelledEffects {
	return must(content.NewModelledEffects(c, t, parts))
}

func attack(hit ...content.SpellOutcome) content.AttackRollResolution {
	return must(content.NewAttackRollResolution(content.SpellAttackRanged, hit, nil))
}

func save(a ability.Ability, onSuccess content.SuccessEffect, fail ...content.SpellOutcome) content.SavingThrowResolution {
	return must(content.NewSavingThrowResolution(a, fail, onSuccess))
}

func automatic(outcomes ...content.SpellOutcome) content.AutomaticResolution {
	return must(content.NewAutomaticResolution(outcomes))
}

func sphere(r int) content.SphereArea { return must(content.NewSphereArea(r)) }

func area(s content.AreaShape) content.TargetsInArea {
	return must(content.NewTargetsInArea(s, content.SubjectCreature, nil))
}

// The pilot spells of spell-effect-data.md table P used by the tests, read from
// the pages cited there.

func fireBolt(c content.EffectCoverage) content.ModelledEffects { // p. 132
	return modelled(c, chosen(content.SubjectCreatureOrObject, exactly(1, nil), nil),
		part(attack(damageOf(amount(rolled("1d10"), cantripDice("d10")), fixedType(content.DamageFire))), once()))
}

func eldritchBlast() content.ModelledEffects { // p. 127
	beams := must(content.NewByCharacterLevelCount(must(content.NewCountLevelTable([]content.CountLevelEntry{
		{Level: 1, Count: 1}, {Level: 5, Count: 2}, {Level: 11, Count: 3}, {Level: 17, Count: 4},
	}))))
	return modelled(content.CoverageComplete, chosen(content.SubjectCreatureOrObject, content.TargetsByInstance{}, nil),
		part(attack(damageOf(amount(rolled("1d10"), nil), fixedType(content.DamageForce))), must(content.NewPartInstances(1, beams))))
}

func magicMissile() content.ModelledEffects { // p. 146
	dart := must(content.NewSpellDealsDamage(amount(rolled("1d4+1"), nil), fixedType(content.DamageForce), content.AddsNothing))
	return modelled(content.CoverageComplete, chosen(content.SubjectCreature, content.TargetsByInstance{}, nil),
		part(automatic(dart), must(content.NewPartInstances(3, must(content.NewPerSlotLevelCount(1, 1))))))
}

func cureWounds() content.ModelledEffects { // p. 121
	heal := must(content.NewSpellRestoresHitPoints(amount(rolled("2d8"), perSlot(1, 2, 0)), content.AddsSpellcastingAbilityModifier))
	return modelled(content.CoverageComplete, chosen(content.SubjectCreature, exactly(1, nil), nil), part(automatic(heal), once()))
}

func falseLife() content.ModelledEffects { // p. 129
	hp := must(content.NewSpellGrantsTemporaryHitPoints(amount(rolled("2d4+4"), perSlot(1, 0, 5)), content.AddsNothing))
	return modelled(content.CoverageComplete, content.TargetsSelf{}, part(automatic(hp), once()))
}

func heal(c content.EffectCoverage) content.ModelledEffects { // p. 139
	hp := must(content.NewSpellRestoresHitPoints(amount(fixed(70), perSlot(6, 0, 10)), content.AddsNothing))
	return modelled(c, chosen(content.SubjectCreature, exactly(1, nil), nil), part(automatic(hp), once()))
}

func fireball(c content.EffectCoverage) content.ModelledEffects { // p. 131
	return modelled(c, area(sphere(20)), part(save(ability.Dexterity, content.HalfDamageOnSuccess,
		damageOf(amount(rolled("8d6"), perSlot(3, 1, 0)), fixedType(content.DamageFire))), once()))
}

func flameStrike() content.ModelledEffects { // p. 132
	cyl := must(content.NewCylinderArea(10, 40))
	return modelled(content.CoverageComplete, area(cyl), part(save(ability.Dexterity, content.HalfDamageOnSuccess,
		damageOf(amount(rolled("5d6"), perSlot(5, 1, 0)), fixedType(content.DamageFire)),
		damageOf(amount(rolled("5d6"), perSlot(5, 1, 0)), fixedType(content.DamageRadiant))), once()))
}

func holdPerson() content.ModelledEffects { // p. 141
	humanoid := must(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
	paralyzed := must(content.NewSpellAppliesCondition(content.ConditionParalyzed, content.ForSpellDuration{}))
	targets := exactly(1, must(content.NewPerSlotLevelCount(2, 1)))
	return modelled(content.CoveragePartial, chosen(content.SubjectCreature, targets, humanoid),
		part(save(ability.Wisdom, content.NothingOnSuccess, paralyzed), once()))
}

func chromaticOrb() content.ModelledEffects { // p. 115
	pick := must(content.NewChosenDamageType([]content.DamageType{
		content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageLightning, content.DamagePoison, content.DamageThunder,
	}))
	return modelled(content.CoveragePartial, chosen(content.SubjectCreatureOrObject, exactly(1, nil), nil),
		part(attack(damageOf(amount(rolled("3d8"), perSlot(1, 1, 0)), pick)), once()))
}

// wantProblem fails the test unless err is a *BlockError for block with the
// problem, matching srderr.ErrInvalidInput.
func wantProblem(tb testing.TB, err error, block string, problem content.BlockProblem) {
	tb.Helper()
	var be *content.BlockError
	if !errors.As(err, &be) || be.Block != block || be.Problem != problem || !errors.Is(err, srderr.ErrInvalidInput) {
		tb.Errorf("error = %v, want *BlockError{%s, %v} wrapping ErrInvalidInput", err, block, problem)
	}
}
