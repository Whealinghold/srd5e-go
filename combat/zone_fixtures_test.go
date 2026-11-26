package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// pzPart is one damage part with a Dexterity save: a synthetic model part.
func pzPart() content.SpellEffectPart {
	return sePart(seSave(ability.Dexterity, []content.SpellOutcome{
		seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)}, content.HalfDamageOnSuccess),
		seInstances(1, nil))
}

// pzParts returns n model parts.
func pzParts(n int) []content.SpellEffectPart {
	out := make([]content.SpellEffectPart, n)
	for i := range out {
		out[i] = pzPart()
	}
	return out
}

// pzTurn returns the turn of active in round.
func pzTurn(round int, active string) combat.TurnMarker {
	return luMust(combat.NewTurnMarker(round, combat.MustCombatantID(active)))
}

// pzRule returns a rule on part with trigger and scope, in the body region.
func pzRule(trig content.ZoneTrigger, part int, scope content.ZoneTargetScope) content.ZoneRuleFact {
	return luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: trig, PartIndex: part, Scope: scope}))
}

// pzRuleIn returns a rule on part with trigger, scope and a named region.
func pzRuleIn(trig content.ZoneTrigger, part int, scope content.ZoneTargetScope, region string) content.ZoneRuleFact {
	return luMust(content.NewZoneRuleFact(content.ZoneRuleFields{
		Trigger: trig, PartIndex: part, Scope: scope, Region: luMust(content.NewZoneRegionID(region))}))
}

// pzTravel returns a TravelsWithin rule on part, one unit per unitFeet.
func pzTravel(part, unitFeet int) content.ZoneRuleFact {
	return luMust(content.NewZoneRuleFact(content.ZoneRuleFields{
		Trigger: content.TravelsWithin, PartIndex: part, Scope: content.AnyCreature, UnitFeet: unitFeet}))
}

// pzFacts returns valid zone facts for f.
func pzFacts(f content.ZoneFactsFields) content.ZoneFacts { return luMust(content.NewZoneFacts(f)) }

// pzSpell returns a spell with parts model parts and zone facts facts, and the
// duration dur (nil means until dispelled, a duration a zone can carry).
func pzSpell(id string, parts int, facts content.ZoneFacts, dur content.SpellDuration) content.Spell {
	if dur == nil {
		dur = content.NewSpellDurationUntilDispelled()
	}
	model := luMust(seModel(content.CoverageComplete, seArea(content.SubjectCreature, nil), pzParts(parts)...).WithZoneFacts(facts))
	return seSpell(id, 3, model, func(f *content.SpellFields) { f.Duration = dur })
}

// pzSpiritFacts are the facts of Spirit Guardians (SRD 5.2.1, pp. 164-165): an
// Emanation that moves with its caster, once per turn, designations at casting.
func pzSpiritFacts() content.ZoneFacts {
	return pzFacts(content.ZoneFactsFields{
		Rules: []content.ZoneRuleFact{
			pzRule(content.CreatureEnters, 0, content.AnyCreature),
			pzRule(content.CreatureEndsTurnInside, 0, content.AnyCreature),
			pzRule(content.ZoneMovesIntoSpace, 0, content.AnyCreature),
		},
		Mobility:    content.MovesWithOrigin{},
		OncePerTurn: true,
		Designation: content.DesignationAtCastingOnly,
	})
}

// pzSpirit is a Spirit Guardians spell: concentration for up to 10 minutes.
func pzSpirit() content.Spell {
	return pzSpell("spirit_guardians", 1, pzSpiritFacts(), luMust(content.NewSpellDurationTimed(10, content.DurationUnitMinutes, content.LimitConcentrationUpTo)))
}

// pzCreate creates a zone for spell, with defaults the edit function may change,
// and fails the test when the spell is refused.
func pzCreate(t *testing.T, spell content.Spell, edit func(*combat.CreateZoneCommand)) combat.PersistentZone {
	t.Helper()
	cmd := combat.CreateZoneCommand{
		ID: luMust(combat.NewPersistentZoneID("zone")), Caster: combat.MustCombatantID("caster"),
		Spell: spell, Shape: luMust(content.NewSphereArea(20)), Cast: pzTurn(1, "caster"),
	}
	if edit != nil {
		edit(&cmd)
	}
	res := combat.CreateZone(cmd)
	created, ok := res.(combat.CreateZoneCreated)
	if !ok {
		t.Fatalf("CreateZone = %T, want CreateZoneCreated", res)
	}
	return created.Zone
}

// pzCreature returns the facts of creature id in the named regions.
func pzCreature(id string, enemy, seen bool, regions ...string) combat.ZoneCreatureFacts {
	rs := make([]content.ZoneRegionID, 0, len(regions))
	for _, r := range regions {
		rs = append(rs, luMust(content.NewZoneRegionID(r)))
	}
	return combat.NewZoneCreatureFacts(combat.MustCombatantID(id), rs, enemy, seen)
}

// pzInBody is a creature inside the zone's body region, seen by and hostile to the caster.
func pzInBody(id string) combat.ZoneCreatureFacts { return pzCreature(id, true, true, "body") }

// pzAssessed returns the single mandatory due effect of an assessed zone, failing the test otherwise.
func pzAssessed(t *testing.T, res combat.AssessZoneTriggerResult) combat.AssessZoneTriggerDue {
	t.Helper()
	due, ok := res.(combat.AssessZoneTriggerDue)
	if !ok {
		t.Fatalf("AssessZoneTrigger = %T, want AssessZoneTriggerDue", res)
	}
	return due
}
