package scenarios_test

import (
	"context"
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

// Requirement: doc/specs/persistent-spell-zones.md FR-054 (with FR-004, FR-018, FR-024, FR-026 and spell-effects.md FR-098) - a Spirit Guardians zone of caster C, cast on C's turn in round 1, assesses two due effects for creature X: X enters on C's turn, then X ends its own turn inside. Each due effect is resolved by ResolveSpellEffects restricted to its part and then applied with ApplyDamage, so the Hit Points fall once per due effect and a successful save halves its own damage only (SRD 5.2.1, "Spirit Guardians", pp. 164-165; "Saving Throws", p. 16). The zone itself rolls, saves and applies nothing. C's own turn end is not reported: only X's arrival and X's turn end are triggers.
// Path: happy.
func TestZoneEffectDueResolvesOnePartAndAppliesDamage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	caster := combat.MustCombatantID("caster")
	x := combat.MustCombatantID("x")
	body := must(content.NewZoneRegionID("body"))

	// Spirit Guardians facts: enters and ends-turn-inside rules on the one part, once per turn, moving with the caster.
	facts := must(content.NewZoneFacts(content.ZoneFactsFields{
		Rules: []content.ZoneRuleFact{
			must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, PartIndex: 0, Scope: content.AnyCreature})),
			must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, PartIndex: 0, Scope: content.AnyCreature})),
		},
		Mobility:    content.MovesWithOrigin{},
		OncePerTurn: true,
	}))
	// A synthetic damage part: 1d6 radiant, a Wisdom save, half damage on a success.
	d6 := must(content.NewRolledSpellAmount(dice.MustParseExpression("1d6")))
	radiant := must(content.NewFixedDamageType(content.DamageRadiant))
	dmg := must(content.NewSpellDealsDamage(must(content.NewScalableAmount(d6, nil)), radiant, content.AddsNothing))
	save := must(content.NewSavingThrowResolution(ability.Wisdom, []content.SpellOutcome{dmg}, content.HalfDamageOnSuccess))
	part := must(content.NewSpellEffectPart(save, must(content.NewPartInstances(1, nil))))
	area := must(content.NewTargetsInArea(must(content.NewEmanationArea(15)), content.SubjectCreature, nil))
	model := must(content.NewModelledEffects(content.CoverageComplete, area, []content.SpellEffectPart{part}))
	model = must(model.WithZoneFacts(facts))
	spell := must(content.NewSpell(content.SpellID("spirit_guardians"), "Spirit Guardians", content.SourceCitation{}, content.SpellFields{
		Level: content.SpellLevel(3), School: content.SchoolEvocation, Classes: []content.ClassID{"cleric"},
		CastingTimes: must(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction())),
		Range:        content.NewSpellRangeSelf(), Components: content.NewSpellComponents(true, true),
		Duration:    must(content.NewSpellDurationTimed(10, content.DurationUnitMinutes, content.LimitConcentrationUpTo)),
		Description: []content.TextBlock{content.MustTextParagraph("Synthetic fixture.")}, Effects: model,
	}))

	// Create the zone on C's turn in round 1.
	cast := must(combat.NewTurnMarker(1, caster))
	created, ok := combat.CreateZone(combat.CreateZoneCommand{
		ID: must(combat.NewPersistentZoneID("spirit")), Caster: caster, Spell: spell,
		Shape: must(content.NewEmanationArea(15)), Cast: cast,
	}).(combat.CreateZoneCreated)
	if !ok {
		t.Fatal("CreateZone was not created")
	}
	zone := created.Zone

	// X enters on C's turn, then ends its own turn inside: two due effects.
	inBody := combat.NewZoneCreatureFacts(x, []content.ZoneRegionID{body}, true, true)
	enter, ok := combat.AssessZoneTrigger(zone, cast, combat.NewCreatureEntered(inBody)).(combat.AssessZoneTriggerDue)
	if !ok || len(enter.Mandatory) != 1 {
		t.Fatalf("entering assessed %#v, want one due effect", enter)
	}
	zone = enter.Zone
	ownTurn := must(combat.NewTurnMarker(1, x))
	endTurn, ok := combat.AssessZoneTrigger(zone, ownTurn, combat.NewCreatureEndedTurn(inBody)).(combat.AssessZoneTriggerDue)
	if !ok || len(endTurn.Mandatory) != 1 {
		t.Fatalf("ending the turn assessed %#v, want one due effect", endTurn)
	}
	zone = endTurn.Zone

	dueEnter, dueEnd := enter.Mandatory[0], endTurn.Mandatory[0]
	if dueEnter.Trigger() != content.CreatureEnters || dueEnd.Trigger() != content.CreatureEndsTurnInside {
		t.Fatalf("triggers = %v, %v; want enters, then ends turn inside", dueEnter.Trigger(), dueEnd.Trigger())
	}
	for _, due := range []combat.ZoneEffectDue{dueEnter, dueEnd} {
		if due.ZoneID() != zone.ID() || due.Spell() != spell.ID() || due.CastMarker() != cast || due.PartIndex() != 0 || due.Units() != 1 {
			t.Fatalf("due effect %#v does not carry the zone, spell, cast marker, part and units [FR-018]", due)
		}
	}

	// Each due effect resolves only its part (RA-2), with a fresh roll: 1d6 first, then X's Wisdom save.
	// The DC is 14 (8 + proficiency 3 + Wisdom 3), so a save of 5 fails and 19 succeeds.
	stats := character.Stats(character.CasterCleric, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	dc := stats.SaveDC()
	report := must(character.NewCastReport(character.CastReportParts{Spell: "spirit_guardians", Source: character.CastSourceSlot, CastAtLevel: 3, SaveDC: &dc}))
	target := must(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: x},
		Saves: map[ability.Ability]d20.SaveAttempt{ability.Wisdom: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}}}))
	resolve := func(due combat.ZoneEffectDue, rolls ...int) combat.PlannedDamage {
		t.Helper()
		uc := must(combat.NewResolveSpellEffects(dicetest.NewFixedRoller(rolls...)))
		out, err := uc.Run(ctx, combat.ResolveSpellEffectsCommand{Spell: spell, Report: report, Caster: caster,
			Targets: []combat.SpellTarget{target}, PartIndexes: []int{due.PartIndex()}, InstanceCount: due.Units()})
		if err != nil {
			t.Fatal(err)
		}
		plan, ok := out.(combat.SpellEffectsResolved)
		if !ok {
			t.Fatalf("ResolveSpellEffects = %T, want SpellEffectsResolved", out)
		}
		return plan.Targets()[0].Damage()[0]
	}
	// Apply the planned damage to X and return X's Hit Points afterwards.
	apply := func(status combat.HitPointStatus, pd combat.PlannedDamage) combat.HitPointStatus {
		t.Helper()
		cmd := must(combat.NewApplyDamageCommand(x, status, pd.Damages(), pd.Adjustments(), combat.OtherSource, pd.HitKind(), nil))
		out, err := combat.NewApplyDamage().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return out.(combat.DamageApplied).Status()
	}

	status := combat.MustHitPointStatus(40, combat.FallsUnconscious, combat.NoAffinities())
	// Entering: the save of 5 fails, so all 4 radiant damage applies.
	status = apply(status, resolve(dueEnter, 4, 5))
	if got := status.HitPoints().Current(); got != 36 {
		t.Fatalf("after entering X has %d Hit Points, want 36", got)
	}
	// Ending its turn inside: the save of 19 succeeds, so 6 damage halves to 3.
	status = apply(status, resolve(dueEnd, 6, 19))
	if got := status.HitPoints().Current(); got != 33 {
		t.Errorf("after ending its turn inside X has %d Hit Points, want 33 [FR-054]", got)
	}
}
