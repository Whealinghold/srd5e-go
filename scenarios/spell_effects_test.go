package scenarios_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// spellFixture builds a synthetic spell (not SRD text) with the effects model.
func spellFixture(t *testing.T, id string, level int, model content.SpellEffectModel) content.Spell {
	t.Helper()
	return must(content.NewSpell(content.SpellID(id), id, content.SourceCitation{}, content.SpellFields{
		Level: content.SpellLevel(level), School: content.SchoolEvocation, Classes: []content.ClassID{"wizard"},
		CastingTimes: must(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction())),
		Range:        content.NewSpellRangeSelf(), Components: content.NewSpellComponents(true, true),
		Duration:    content.NewSpellDurationInstantaneous(),
		Description: []content.TextBlock{content.MustTextParagraph("Text.")}, Effects: model,
	}))
}

// Requirement: doc/specs/spell-effects.md section 8 (scenario), FR-025, FR-026, FR-064, FR-065 - a spell cast at a level 4 slot resolves for two targets, one failed and one successful save; the PlannedDamage goes to ApplyDamage with the HalfDamage adjustment for the saver, and the final Hit Points are asserted (SRD 5.2.1, "Saving Throws", "Half Damage", pp. 16 and 106). The cast report is built by hand here; TestCastSpellReportFeedsResolveSpellEffects below takes it from the real CastSpell.
// Path: happy.
func TestSpellEffectsPlanFeedsApplyDamage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d6 := must(content.NewRolledSpellAmount(dice.MustParseExpression("8d6")))
	step := must(content.NewPerSlotLevelAmount(3, must(content.NewAmountStep(1, 0))))
	amount := must(content.NewScalableAmount(d6, step))
	fire := must(content.NewFixedDamageType(content.DamageFire))
	dmg := must(content.NewSpellDealsDamage(amount, fire, content.AddsNothing))
	save := must(content.NewSavingThrowResolution(ability.Dexterity, []content.SpellOutcome{dmg}, content.HalfDamageOnSuccess))
	part := must(content.NewSpellEffectPart(save, must(content.NewPartInstances(1, nil))))
	area := must(content.NewTargetsInArea(must(content.NewSphereArea(20)), content.SubjectCreature, nil))
	spell := spellFixture(t, "burst", 3, must(content.NewModelledEffects(content.CoverageComplete, area, []content.SpellEffectPart{part})))

	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	dc := stats.SaveDC()
	report := must(character.NewCastReport(character.CastReportParts{Spell: "burst", Source: character.CastSourceSlot, CastAtLevel: 4, SaveDC: &dc}))
	roll := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	target := func(id string) combat.SpellTarget {
		return must(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: combat.MustCombatantID(id)},
			Saves: map[ability.Ability]d20.SaveAttempt{ability.Dexterity: roll}}))
	}
	uc := must(combat.NewResolveSpellEffects(dicetest.NewFixedRoller(3, 3, 3, 3, 3, 3, 3, 3, 3, 5, 19)))
	res, err := uc.Run(ctx, combat.ResolveSpellEffectsCommand{Spell: spell, Report: report, Caster: combat.MustCombatantID("wizard"),
		Targets: []combat.SpellTarget{target("failer"), target("saver")}})
	if err != nil {
		t.Fatal(err)
	}
	plan := res.(combat.SpellEffectsResolved)

	hp := func(i int) int {
		status := combat.MustHitPointStatus(40, combat.FallsUnconscious, combat.NoAffinities())
		id := plan.Targets()[i].Target().(combat.CreatureTargetRef).ID
		pd := plan.Targets()[i].Damage()[0]
		cmd := must(combat.NewApplyDamageCommand(id, status, pd.Damages(), pd.Adjustments(), combat.OtherSource, pd.HitKind(), nil))
		out, err := combat.NewApplyDamage().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return out.(combat.DamageApplied).Status().HitPoints().Current()
	}
	// 9d6 of 3 each is 27: the failer takes all of it, the saver half rounded down.
	if got := hp(0); got != 40-27 {
		t.Errorf("the failer has %d Hit Points, want 13", got)
	}
	if got := hp(1); got != 40-13 {
		t.Errorf("the saver has %d Hit Points, want 27", got)
	}
}

// Requirement: doc/specs/spell-effects.md FR-095, FR-023, Q18 - a flat bonus the host adds to the Damage of a PlannedDamage before ApplyDamage is added once and is not doubled by a Critical Hit, which doubled only the dice (SRD 5.2.1, "Critical Hits", p. 16).
// Path: unhappy (rule interaction).
func TestHostRiderIsNotDoubledOnACriticalHit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d10 := must(content.NewRolledSpellAmount(dice.MustParseExpression("1d10")))
	fire := must(content.NewFixedDamageType(content.DamageFire))
	dmg := must(content.NewSpellDealsDamage(must(content.NewScalableAmount(d10, nil)), fire, content.AddsNothing))
	attack := must(content.NewAttackRollResolution(content.SpellAttackRanged, []content.SpellOutcome{dmg}, nil))
	part := must(content.NewSpellEffectPart(attack, must(content.NewPartInstances(1, nil))))
	one := must(content.NewExactlyTargets(1, nil))
	chosen := must(content.NewTargetsChosen(content.SubjectCreature, one, nil, content.MayIncludeCaster))
	spell := spellFixture(t, "bolt", 1, must(content.NewModelledEffects(content.CoverageComplete, chosen, []content.SpellEffectPart{part})))
	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	bonus := stats.AttackBonus()
	report := must(character.NewCastReport(character.CastReportParts{Spell: "bolt", Source: character.CastSourceSlot, CastAtLevel: 1, AttackBonus: &bonus}))
	facts := must(combat.NewSpellAttackFacts(combat.SpellAttackFactsParams{ArmorClass: 12, Distance: 30, Range: must(combat.NewAttackRange(120, 0))}))
	target := must(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: combat.MustCombatantID("dummy")}, AttackFacts: facts}))
	uc := must(combat.NewResolveSpellEffects(dicetest.NewFixedRoller(20, 3, 4)))
	res, err := uc.Run(ctx, combat.ResolveSpellEffectsCommand{Spell: spell, Report: report, Caster: combat.MustCombatantID("wizard"),
		AttackModifiers: stats.AttackRollModifiers(), Targets: []combat.SpellTarget{target}})
	if err != nil {
		t.Fatal(err)
	}
	pd := res.(combat.SpellEffectsResolved).Targets()[0].Damage()[0]
	if pd.HitKind() != combat.CriticalHit || pd.Damages()[0].Amount() != 7 {
		t.Fatalf("planned %v kind %v, want a critical 3 + 4", pd.Damages(), pd.HitKind())
	}
	const rider = 3 // a host-side class rider, once per damage roll
	boosted := combat.MustDamage(pd.Damages()[0].Amount()+rider, pd.Damages()[0].Type())
	status := combat.MustHitPointStatus(40, combat.FallsUnconscious, combat.NoAffinities())
	cmd := must(combat.NewApplyDamageCommand(combat.MustCombatantID("dummy"), status, []combat.Damage{boosted}, pd.Adjustments(), combat.OtherSource, pd.HitKind(), nil))
	out, err := combat.NewApplyDamage().Run(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.(combat.DamageApplied).Status().HitPoints().Current(); got != 40-7-rider {
		t.Errorf("Hit Points = %d, want %d: the rider is added once", got, 40-7-rider)
	}
}

// castingFacts is a context in which the caster can speak, gesture and cast.
func castingFacts(t *testing.T) character.CastingContext {
	t.Helper()
	return must(character.NewCastingContext(character.CastingFacts{CanCastSpells: true, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, HasComponentPouch: true, TriggerMet: true}))
}

// Requirement: doc/specs/spell-effects.md section 8 (scenario) and D-2 - a Wizard 7 casts a synthetic spell with the real CastSpell at a level 4 slot, the CastReport it returns (not one built by hand) feeds ResolveSpellEffects for two targets, one failed and one successful save, and the PlannedDamage goes to ApplyDamage with HalfDamage for the saver; one level 4 slot is spent and the final Hit Points are asserted (SRD 5.2.1, "Casting Spells", pp. 104-106; "Half Damage", p. 16).
// Path: happy.
func TestCastSpellReportFeedsResolveSpellEffects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	d6 := must(content.NewRolledSpellAmount(dice.MustParseExpression("8d6")))
	amount := must(content.NewScalableAmount(d6, must(content.NewPerSlotLevelAmount(3, must(content.NewAmountStep(1, 0))))))
	dmg := must(content.NewSpellDealsDamage(amount, must(content.NewFixedDamageType(content.DamageFire)), content.AddsNothing))
	save := must(content.NewSavingThrowResolution(ability.Dexterity, []content.SpellOutcome{dmg}, content.HalfDamageOnSuccess))
	part := must(content.NewSpellEffectPart(save, must(content.NewPartInstances(1, nil))))
	area := must(content.NewTargetsInArea(must(content.NewSphereArea(20)), content.SubjectCreature, nil))
	spell := spellFixture(t, "burst", 3, must(content.NewModelledEffects(content.CoverageComplete, area, []content.SpellEffectPart{part})))

	spells := bundledSpells(t)
	spells[spell.ID()] = spell
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: spellLevels(t, "wizard", 7), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("burst"), Spellbook: character.NewSpellbook(ids("burst"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state := created.(character.CreateSpellcastingCreated).State()
	level4 := character.MustSlotLevel(4)
	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(7)))
	cast, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Class: character.CasterWizard, Stats: &stats, Spell: spell, Mode: character.CastModeNormal,
		Slot: character.NewSpellSlotChoiceSpellcasting(level4), Context: castingFacts(t)})
	if err != nil {
		t.Fatal(err)
	}
	performed, ok := cast.(character.CastSpellCast)
	if !ok {
		t.Fatalf("CastSpell = %#v, want CastSpellCast", cast)
	}
	if at, _ := performed.Report().CastAtLevel(); at != 4 || performed.State().Slots(character.PoolSpellcasting).Remaining(level4) != state.Slots(character.PoolSpellcasting).Remaining(level4)-1 {
		t.Fatalf("cast at level %d, level 4 slots left %d (from %d), want 4 and one fewer", at, performed.State().Slots(character.PoolSpellcasting).Remaining(level4), state.Slots(character.PoolSpellcasting).Remaining(level4))
	}

	roll := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	target := func(id string) combat.SpellTarget {
		return must(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: combat.MustCombatantID(id)},
			Saves: map[ability.Ability]d20.SaveAttempt{ability.Dexterity: roll}}))
	}
	uc := must(combat.NewResolveSpellEffects(dicetest.NewFixedRoller(3, 3, 3, 3, 3, 3, 3, 3, 3, 5, 19)))
	res, err := uc.Run(ctx, combat.ResolveSpellEffectsCommand{Spell: spell, Report: performed.Report(), Caster: combat.MustCombatantID("wizard"),
		Targets: []combat.SpellTarget{target("failer"), target("saver")}})
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := res.(combat.SpellEffectsResolved)
	if !ok || plan.EffectiveLevel() != 4 {
		t.Fatalf("result = %#v, want SpellEffectsResolved at level 4", res)
	}
	hp := func(i int) int {
		status := combat.MustHitPointStatus(40, combat.FallsUnconscious, combat.NoAffinities())
		id := plan.Targets()[i].Target().(combat.CreatureTargetRef).ID
		pd := plan.Targets()[i].Damage()[0]
		out, err := combat.NewApplyDamage().Run(ctx, must(combat.NewApplyDamageCommand(id, status, pd.Damages(), pd.Adjustments(), combat.OtherSource, pd.HitKind(), nil)))
		if err != nil {
			t.Fatal(err)
		}
		return out.(combat.DamageApplied).Status().HitPoints().Current()
	}
	// 9d6 of 3 each is 27 (8d6 and one more die for the level 4 slot): all of it, then half rounded down.
	if got := hp(0); got != 40-27 {
		t.Errorf("the failer has %d Hit Points, want 13", got)
	}
	if got := hp(1); got != 40-13 {
		t.Errorf("the saver has %d Hit Points, want 27", got)
	}
}

// Requirement: doc/specs/spell-effects.md section 8 (scenario, second row) and table N7 - a Wizard 5 casts a synthetic cantrip shaped like Fire Bolt with the real CastSpell (no slot, ScalingLevel 5), the report's spell attack bonus decides a hit and the dice come from the character-level table: 2d10 at level 5 (SRD 5.2.1, "Fire Bolt", p. 132; "Cantrips", p. 104).
// Path: happy.
func TestCastCantripAtCharacterLevelFiveFeedsResolveSpellEffects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	table := func(n int) content.SpellAmount {
		return must(content.NewRolledSpellAmount(dice.MustParseExpression(fmt.Sprintf("%dd10", n))))
	}
	levels := must(content.NewAmountLevelTable([]content.AmountLevelEntry{{Level: 1, Value: table(1)}, {Level: 5, Value: table(2)}, {Level: 11, Value: table(3)}, {Level: 17, Value: table(4)}}))
	amount := must(content.NewScalableAmount(table(1), must(content.NewByCharacterLevelAmount(levels))))
	dmg := must(content.NewSpellDealsDamage(amount, must(content.NewFixedDamageType(content.DamageFire)), content.AddsNothing))
	attack := must(content.NewAttackRollResolution(content.SpellAttackRanged, []content.SpellOutcome{dmg}, nil))
	part := must(content.NewSpellEffectPart(attack, must(content.NewPartInstances(1, nil))))
	chosen := must(content.NewTargetsChosen(content.SubjectCreatureOrObject, must(content.NewExactlyTargets(1, nil)), nil, content.MayIncludeCaster))
	spell := spellFixture(t, "ember_bolt", 0, must(content.NewModelledEffects(content.CoverageComplete, chosen, []content.SpellEffectPart{part})))

	spells := bundledSpells(t)
	spells[spell.ID()] = spell
	created, err := character.NewCreateSpellcasting().Do(ctx, character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: spellLevels(t, "wizard", 5), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("ember_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("magic_missile"), Spellbook: character.NewSpellbook(ids("magic_missile"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state := created.(character.CreateSpellcastingCreated).State()
	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	cast, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Class: character.CasterWizard, Stats: &stats, Spell: spell, Mode: character.CastModeNormal, Context: castingFacts(t)})
	if err != nil {
		t.Fatal(err)
	}
	performed, ok := cast.(character.CastSpellCast)
	if !ok {
		t.Fatalf("CastSpell = %#v, want CastSpellCast", cast)
	}
	report := performed.Report()
	scaling, hasScaling := report.ScalingLevel()
	bonus, hasBonus := report.AttackBonus()
	if _, _, spent := report.Slot(); spent || !hasScaling || scaling != 5 || !hasBonus || report.Source() != character.CastSourceCantrip {
		t.Fatalf("report = %+v, want a cantrip with scaling level 5, a spell attack bonus and no slot", report)
	}

	facts := must(combat.NewSpellAttackFacts(combat.SpellAttackFactsParams{ArmorClass: 12, Distance: 30, Range: must(combat.NewAttackRange(120, 0))}))
	target := must(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: combat.MustCombatantID("dummy")}, AttackFacts: facts}))
	uc := must(combat.NewResolveSpellEffects(dicetest.NewFixedRoller(15, 4, 6)))
	res, err := uc.Run(ctx, combat.ResolveSpellEffectsCommand{Spell: spell, Report: report, Caster: combat.MustCombatantID("wizard"),
		AttackModifiers: d20.NewRollModifiers(modifier.New(bonus.Value())), Targets: []combat.SpellTarget{target}})
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := res.(combat.SpellEffectsResolved)
	if !ok {
		t.Fatalf("result = %#v, want SpellEffectsResolved", res)
	}
	pd := plan.Targets()[0].Damage()
	if len(pd) != 1 || pd[0].HitKind() != combat.NormalHit || pd[0].Damages()[0].Amount() != 10 {
		t.Errorf("planned damage = %v, want one normal hit of 2d10 = 4 + 6", pd)
	}
}
