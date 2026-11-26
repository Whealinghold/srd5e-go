package combat_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
)

// sumAllEvents returns the events of real transitions, covering the twelve kinds, with the first event of each kind.
func sumAllEvents(tb testing.TB) map[string]combat.SummonEvent {
	tb.Helper()
	world := sumWorld(tb)
	out := map[string]combat.SummonEvent{}
	collect := func(t combat.SummonedTransition) combat.SummonedApplied {
		a := sumApplied(tb, t)
		for _, e := range a.Events() {
			if _, seen := out[e.Kind()]; !seen {
				out[e.Kind()] = e
			}
		}
		return a
	}
	collect(sumChange(tb, world, sumCommand(combat.SummonAnimatedObject, 0, "o1", "o2"), ""))
	collect(sumChange(tb, world, combat.ReportZeroHitPoints{Creature: cid("i1")}, ""))
	collect(sumChange(tb, world, combat.ReportSpellEnded{Summoner: cid("pc"), Spell: "summon_dragon", Cause: combat.SummonSpellEndDispelled}, ""))
	collect(world.Reassert(cid("pc"), sumReassert(combat.SummonAnimatedUndead, 3, "u1")))
	collect(sumChange(tb, world, combat.ElapseControlTime{Elapsed: 24 * time.Hour}, ""))
	collect(sumFamiliarCast(tb, world, "x", normalForm("cat"), content.CreatureTypeFey, combat.FamiliarOriginFindFamiliarSpell, sumMonsters(tb)))
	pocket := collect(sumChange(tb, world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, ""))
	collect(sumChange(tb, pocket.Roster(), combat.RecallFamiliar{Summoner: cid("pc"), SpaceAvailable: true}, ""))
	linked := collect(sumChange(tb, world, combat.ShareFamiliarSenses{Summoner: cid("pc")}, ""))
	collect(sumChange(tb, linked.Roster(), combat.ReportTurnStarted{Combatant: cid("pc")}, ""))
	collect(sumChange(tb, world, combat.RepairSimulacrum{Summoner: cid("pc"), Creature: cid("sim"), Missing: 3, Restore: 2, LongRestTaken: true}, ""))
	collect(sumTry(tb, combat.SummonedRoster{}, "pc", sumObjects("a"), sumScene()))
	return out
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-069, FR-026 - the events are the twelve members of the closed family SummonEvent, each with a constant unique snake_case Kind and Combatants that are never empty or repeated and are a copy: the creature and then its summoner, or the summoner and then the creatures commanded, or the summoner alone for the end of a shared Resistance.
// Path: per case.
func TestSummonEventsHonourTheEventContract(t *testing.T) {
	t.Parallel()
	want := map[string][]combat.CombatantID{
		"summoned_creature_joined":       {"a", "pc"},
		"summoned_creature_commanded":    {"pc", "o1", "o2"},
		"summoned_creature_departed":     {"i1", "pc"},
		"shared_resistance_ended":        {"pc"},
		"summon_control_reasserted":      {"pc", "u1"},
		"summon_control_lapsed":          {"u1", "pc"},
		"familiar_form_changed":          {"fam", "pc"},
		"familiar_dismissed_temporarily": {"fam", "pc"},
		"familiar_reappeared":            {"fam", "pc"},
		"familiar_sight_link_started":    {"fam", "pc"},
		"familiar_sight_link_ended":      {"fam", "pc"},
		"simulacrum_repaired":            {"sim", "pc"},
	}
	got := sumAllEvents(t)
	if len(got) != len(want) {
		t.Fatalf("%d kinds of event produced, want the twelve of FR-069", len(got))
	}
	var distinct []combat.Event
	for kind, ids := range want {
		e, ok := got[kind]
		if !ok {
			t.Errorf("no event of kind %s was produced [FR-069]", kind)
			continue
		}
		combattest.CheckEvent(t, e, kind, ids...)
		distinct = append(distinct, e)
	}
	combattest.CheckEventKinds(t, distinct)
	joined := sumDo(t, combat.SummonedRoster{}, "pc", sumObjects("a", "b"), sumScene())
	combattest.CheckEventsCopy(t, func() []combat.Event {
		var out []combat.Event
		for _, e := range joined.Events() {
			out = append(out, e)
		}
		return out
	})
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-001, FR-027, FR-069 - SummonedCreatureJoined reports the side, level, placement and, for a Draconic Spirit, the Resistance its summoner shares; a creature of another kind reports none.
// Path: per case.
func TestSummonedCreatureJoinedReportsItsGrant(t *testing.T) {
	t.Parallel()
	spirit := sumDo(t, combat.SummonedRoster{}, "pc", combat.SummonDragonRequest{Creature: cid("d"), Level: 6, SharedResistance: content.DamageAcid, BreathDamageType: content.DamageFire}, sumScene())
	ev := spirit.Events()[0].(combat.SummonedCreatureJoined)
	grant, ok := ev.SharedResistance()
	if want := (combat.SharedResistanceGrant{Summoner: cid("pc"), Spell: "summon_dragon", Damage: content.DamageAcid}); !ok || grant != want || ev.CastLevel() != 6 {
		t.Errorf("grant = %v, %v level %d, want %v at level 6 [FR-027]", grant, ok, ev.CastLevel(), want)
	}
	insect := sumDo(t, combat.SummonedRoster{}, "pc", combat.GiantInsectRequest{Creature: cid("i"), Level: 4, Form: combat.InsectWasp}, sumScene())
	if _, ok := insect.Events()[0].(combat.SummonedCreatureJoined).SharedResistance(); ok {
		t.Errorf("an insect reports a shared Resistance [FR-027]")
	}
	again := sumDo(t, spirit.Roster(), "pc", combat.SummonDragonRequest{Creature: cid("d2"), Level: 5, SharedResistance: content.DamageFire, BreathDamageType: content.DamageFire}, sumScene())
	if got := sumKinds(again.Events()); len(got) != 2 || got[0] != "shared_resistance_ended" || got[1] != "summoned_creature_joined" {
		t.Errorf("events = %v, want the old grant ending before the new spirit joins [FR-027]", got)
	}
	if grants := again.Roster().SharedResistances(); len(grants) != 1 || grants[0].Damage != content.DamageFire {
		t.Errorf("grants = %v, want only the new one", grants)
	}
}
