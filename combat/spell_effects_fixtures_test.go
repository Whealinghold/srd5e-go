package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

const spellEffectsSpec = "doc/specs/spell-effects.md "

// The spells of these tests are synthetic: made-up fixtures that carry the numbers the spec's case
// tables quote from SRD 5.2.1 (pp. 107-158), not spell text.

func seDice(notation string) content.RolledSpellAmount {
	return luMust(content.NewRolledSpellAmount(dice.MustParseExpression(notation)))
}

func seFixedAmount(n int) content.FixedSpellAmount { return luMust(content.NewFixedSpellAmount(n)) }

func seAmount(base content.SpellAmount, scaling content.AmountScaling) content.ScalableAmount {
	return luMust(content.NewScalableAmount(base, scaling))
}

func sePerSlot(above, extraDice, extraFlat int) content.PerSlotLevelAmount {
	return luMust(content.NewPerSlotLevelAmount(above, luMust(content.NewAmountStep(extraDice, extraFlat))))
}

// seCantripDice scales a die from 1 to 4 dice at character levels 1, 5, 11 and 17.
func seCantripDice(die string) content.ByCharacterLevelAmount {
	return luMust(content.NewByCharacterLevelAmount(luMust(content.NewAmountLevelTable([]content.AmountLevelEntry{
		{Level: 1, Value: seDice("1" + die)}, {Level: 5, Value: seDice("2" + die)},
		{Level: 11, Value: seDice("3" + die)}, {Level: 17, Value: seDice("4" + die)}}))))
}

func seDamage(a content.ScalableAmount, sel content.DamageTypeSelection, adds content.ModifierAddition) content.SpellDealsDamage {
	return luMust(content.NewSpellDealsDamage(a, sel, adds))
}

func seFixedType(t content.DamageType) content.FixedDamageType {
	return luMust(content.NewFixedDamageType(t))
}

func seInstances(base int, s content.CountScaling) content.PartInstances {
	return luMust(content.NewPartInstances(base, s))
}

func sePart(r content.PartResolution, i content.PartInstances) content.SpellEffectPart {
	return luMust(content.NewSpellEffectPart(r, i))
}

func seSave(a ability.Ability, onFailure []content.SpellOutcome, onSuccess content.SuccessEffect) content.SavingThrowResolution {
	return luMust(content.NewSavingThrowResolution(a, onFailure, onSuccess))
}

func seAttack(kind content.SpellAttackKind, onHit, onMiss []content.SpellOutcome) content.AttackRollResolution {
	return luMust(content.NewAttackRollResolution(kind, onHit, onMiss))
}

func seAuto(o ...content.SpellOutcome) content.AutomaticResolution {
	return luMust(content.NewAutomaticResolution(o))
}

func seModel(c content.EffectCoverage, t content.SpellTargeting, parts ...content.SpellEffectPart) content.ModelledEffects {
	return luMust(content.NewModelledEffects(c, t, parts))
}

func seChosen(subject content.TargetSubject, c content.TargetCount, r content.CreatureTypeRestriction, p content.SelfTargetPolicy) content.TargetsChosen {
	return luMust(content.NewTargetsChosen(subject, c, r, p))
}

func seExactly(n int) content.ExactlyTargets { return luMust(content.NewExactlyTargets(n, nil)) }

func seArea(subject content.TargetSubject, r content.CreatureTypeRestriction) content.TargetsInArea {
	return luMust(content.NewTargetsInArea(luMust(content.NewSphereArea(20)), subject, r))
}

func seSpell(id string, level int, model content.SpellEffectModel, edit func(*content.SpellFields)) content.Spell {
	f := content.SpellFields{
		Level: content.SpellLevel(level), School: content.SchoolEvocation, Classes: []content.ClassID{"wizard"},
		CastingTimes: luMust(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction())),
		Range:        content.NewSpellRangeSelf(), Components: content.NewSpellComponents(true, true),
		Duration:    content.NewSpellDurationInstantaneous(),
		Description: []content.TextBlock{content.MustTextParagraph("Text.")}, Effects: model,
	}
	if edit != nil {
		edit(&f)
	}
	return luMust(content.NewSpell(content.SpellID(id), id, content.SourceCitation{}, f))
}

var (
	seFire    = seFixedType(content.DamageFire)
	seNothing = content.NothingOnSuccess
)

// seFireball: a level 3 sphere, a Dexterity save, 8d6 Fire and 1d6 more per slot above 3, half on a success.
func seFireball() content.Spell {
	return seSpell("fireball", 3, seModel(content.CoverageComplete, seArea(content.SubjectCreatureOrObject, nil),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("8d6"), sePerSlot(3, 1, 0)), seFire, content.AddsNothing)}, content.HalfDamageOnSuccess), seInstances(1, nil))), nil)
}

// seFlameStrike: a level 5 cylinder, a Dexterity save, 5d6 Fire and 5d6 Radiant, half on a success.
func seFlameStrike() content.Spell {
	return seSpell("flame_strike", 5, seModel(content.CoverageComplete, seArea(content.SubjectCreature, nil),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{
			seDamage(seAmount(seDice("5d6"), sePerSlot(5, 1, 0)), seFire, content.AddsNothing),
			seDamage(seAmount(seDice("5d6"), nil), seFixedType(content.DamageRadiant), content.AddsNothing)}, content.HalfDamageOnSuccess), seInstances(1, nil))), nil)
}

// seHarm: a level 6 single target, a Constitution save, 14d6 Necrotic and half on a success.
func seHarm() content.Spell {
	return seSpell("harm", 6, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seSave(ability.Constitution, []content.SpellOutcome{seDamage(seAmount(seDice("14d6"), nil), seFixedType(content.DamageNecrotic), content.AddsNothing)}, content.HalfDamageOnSuccess), seInstances(1, nil))), nil)
}

// seAcidSplash: a cantrip, a Dexterity save, Acid damage from a table, nothing on a success.
func seAcidSplash() content.Spell {
	return seSpell("acid_splash", 0, seModel(content.CoverageComplete,
		seChosen(content.SubjectCreature, luMust(content.NewUpToTargets(2, nil)), nil, content.MayIncludeCaster),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("1d6"), seCantripDice("d6")), seFixedType(content.DamageAcid), content.AddsNothing)}, seNothing), seInstances(1, nil))), nil)
}

// seHoldPerson: a level 2 spell for up to one Humanoid and one more per slot above 2, a Wisdom save, Paralyzed.
func seHoldPerson() content.Spell {
	humanoids := luMust(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
	up := luMust(content.NewUpToTargets(1, luMust(content.NewPerSlotLevelCount(2, 1))))
	para := luMust(content.NewSpellAppliesCondition(content.ConditionParalyzed, content.ForSpellDuration{}))
	return seSpell("hold_person", 2, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, up, humanoids, content.ExcludesCaster),
		sePart(seSave(ability.Wisdom, []content.SpellOutcome{para}, seNothing), seInstances(1, nil))), nil)
}

// seFireBolt: a cantrip ranged spell attack for Fire damage from a table of dice.
func seFireBolt() content.Spell {
	return seSpell("fire_bolt", 0, seModel(content.CoverageComplete, seChosen(content.SubjectCreatureOrObject, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d10"), seCantripDice("d10")), seFire, content.AddsNothing)}, nil), seInstances(1, nil))), nil)
}

// seEldritchBlast: a cantrip with one ranged spell attack per beam, beams from a table, 1d10 Force each.
func seEldritchBlast() content.Spell {
	beams := luMust(content.NewByCharacterLevelCount(luMust(content.NewCountLevelTable([]content.CountLevelEntry{
		{Level: 1, Count: 1}, {Level: 5, Count: 2}, {Level: 11, Count: 3}, {Level: 17, Count: 4}}))))
	return seSpell("eldritch_blast", 0, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, content.TargetsByInstance{}, nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d10"), nil), seFixedType(content.DamageForce), content.AddsNothing)}, nil), seInstances(1, beams))), nil)
}

// seMagicMissile: a level 1 spell of three automatic darts (one more per slot above 1), 1d4+1 Force each.
func seMagicMissile() content.Spell {
	return seSpell("magic_missile", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, content.TargetsByInstance{}, nil, content.MayIncludeCaster),
		sePart(seAuto(seDamage(seAmount(seDice("1d4+1"), nil), seFixedType(content.DamageForce), content.AddsNothing)),
			seInstances(3, luMust(content.NewPerSlotLevelCount(1, 1))))), nil)
}

// seCureWounds: a level 1 touch healing 2d8 plus the spellcasting ability modifier, 2d8 more per slot above 1.
func seCureWounds() content.Spell {
	heal := luMust(content.NewSpellRestoresHitPoints(seAmount(seDice("2d8"), sePerSlot(1, 2, 0)), content.AddsSpellcastingAbilityModifier))
	return seSpell("cure_wounds", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAuto(heal), seInstances(1, nil))), nil)
}

// seFalseLife: a level 1 self spell granting 2d4+4 Temporary Hit Points, 5 more per slot above 1.
func seFalseLife() content.Spell {
	temp := luMust(content.NewSpellGrantsTemporaryHitPoints(seAmount(seDice("2d4+4"), sePerSlot(1, 0, 5)), content.AddsNothing))
	return seSpell("false_life", 1, seModel(content.CoverageComplete, content.TargetsSelf{}, sePart(seAuto(temp), seInstances(1, nil))), nil)
}

// seChromaticOrb: a level 1 ranged spell attack for 3d8 of a Damage Type the caster chooses.
func seChromaticOrb() content.Spell {
	types := luMust(content.NewChosenDamageType([]content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire,
		content.DamageLightning, content.DamagePoison, content.DamageThunder}))
	return seSpell("chromatic_orb", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("3d8"), sePerSlot(1, 1, 0)), types, content.AddsNothing)}, nil), seInstances(1, nil))), nil)
}

// seIceKnife: a level 1 ranged spell attack against one creature (never an object).
func seIceKnife() content.Spell {
	return seSpell("ice_knife", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d10"), nil), seFixedType(content.DamagePiercing), content.AddsNothing)}, nil), seInstances(1, nil))), nil)
}

// seWardenBolt: a synthetic cantrip-free ritual spell with a 2d6 automatic damage outcome that adds the ability modifier.
func seWardenBolt() content.Spell {
	return seSpell("warden_bolt", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.ExcludesCaster),
		sePart(seAuto(seDamage(seAmount(seDice("2d6"), nil), seFire, content.AddsSpellcastingAbilityModifier)), seInstances(1, nil))),
		func(f *content.SpellFields) { f.Ritual = true })
}

// seSavingStats are the stats of a level 5 wizard-like caster: +3 ability modifier, +3 Proficiency Bonus.
func seSavingStats() character.SpellcastingStats {
	return character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
}

// seReport is the cast report of spellID cast at slot level castAt (0: none) with ScalingLevel scaling (0: none).
func seReport(spellID string, castAt, scaling int) character.CastReport {
	st := seSavingStats()
	dc, bonus := st.SaveDC(), st.AttackBonus()
	return luMust(character.NewCastReport(character.CastReportParts{Spell: content.SpellID(spellID), Source: character.CastSourceSlot,
		CastAtLevel: castAt, ScalingLevel: scaling, SaveDC: &dc, AttackBonus: &bonus}))
}

var seCaster = gid("caster")

// seTarget returns the creature target id with the Wisdom, Dexterity and Constitution saves attempted by rolling with +0.
func seTarget(id string, edit func(*combat.SpellTargetParams)) combat.SpellTarget {
	roll := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	p := combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: gid(id)}, CreatureType: content.CreatureTypeHumanoid,
		Saves: map[ability.Ability]d20.SaveAttempt{ability.Dexterity: roll, ability.Wisdom: roll, ability.Constitution: roll}}
	if edit != nil {
		edit(&p)
	}
	return luMust(combat.NewSpellTarget(p))
}

// seFacts are plain attack facts against Armor Class 12 at 30 feet with a 120 foot range.
func seFacts(edit func(*combat.SpellAttackFactsParams)) combat.SpellAttackFacts {
	p := combat.SpellAttackFactsParams{ArmorClass: 12, Distance: 30, Range: luMust(combat.NewAttackRange(120, 0))}
	if edit != nil {
		edit(&p)
	}
	return luMust(combat.NewSpellAttackFacts(p))
}

func seWithFacts(id string) combat.SpellTarget {
	return seTarget(id, func(p *combat.SpellTargetParams) { p.AttackFacts = seFacts(nil) })
}

// seCommand is the command to cast spell with report, aimed at targets.
func seCommand(spell content.Spell, report character.CastReport, targets ...combat.SpellTarget) combat.ResolveSpellEffectsCommand {
	return combat.ResolveSpellEffectsCommand{Spell: spell, Report: report, Caster: seCaster, AttackModifiers: seSavingStats().AttackRollModifiers(),
		AbilityModifier: seSavingStats().AbilityModifier().Value(), Targets: targets}
}

func seUse(tb testing.TB, r combat.Roller, opts ...combat.Option) combat.ResolveSpellEffects {
	tb.Helper()
	return luMust(combat.NewResolveSpellEffects(r, opts...))
}

// seRun runs cmd with the roller and returns the result.
func seRun(tb testing.TB, cmd combat.ResolveSpellEffectsCommand, r combat.Roller, opts ...combat.Option) (combat.ResolveSpellEffectsResult, error) {
	tb.Helper()
	return seUse(tb, r, opts...).Run(context.Background(), cmd)
}

// seResolved runs cmd and fails the test unless the result is SpellEffectsResolved.
func seResolved(tb testing.TB, cmd combat.ResolveSpellEffectsCommand, r combat.Roller, opts ...combat.Option) combat.SpellEffectsResolved {
	tb.Helper()
	res, err := seRun(tb, cmd, r, opts...)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	got, ok := res.(combat.SpellEffectsResolved)
	if !ok {
		tb.Fatalf("result = %#v, want SpellEffectsResolved", res)
	}
	return got
}

// seRejected runs cmd and returns the reasons of the rejection.
func seRejected(tb testing.TB, cmd combat.ResolveSpellEffectsCommand, r combat.Roller) []combat.SpellEffectsRejectionKind {
	tb.Helper()
	res, err := seRun(tb, cmd, r)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	got, ok := res.(combat.SpellEffectsRejected)
	if !ok {
		tb.Fatalf("result = %#v, want SpellEffectsRejected", res)
	}
	return got.Rejection().Reasons()
}

// seRef is the creature target ref of id.
func seRef(id string) combat.SpellTargetRef { return combat.CreatureTargetRef{ID: gid(id)} }

// seSix returns n copies of v, for the faces of n dice.
func seFaces(v, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}
	return out
}
