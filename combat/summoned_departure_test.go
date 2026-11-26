package combat_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// sumDeparted returns the departure events of an applied transition.
func sumDeparted(a combat.SummonedApplied) []combat.SummonedCreatureDeparted {
	var out []combat.SummonedCreatureDeparted
	for _, e := range a.Events() {
		if d, ok := e.(combat.SummonedCreatureDeparted); ok {
			out = append(out, d)
		}
	}
	return out
}

// sumLeaves returns the leave instructions of an applied transition.
func sumLeaves(a combat.SummonedApplied) []combat.SummonedRosterLeave {
	var out []combat.SummonedRosterLeave
	for _, act := range a.RosterActions() {
		out = append(out, act.(combat.SummonedRosterLeave))
	}
	return out
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-006, FR-019, FR-020, FR-024, FR-025, FR-026 - a summoned creature at 0 Hit Points leaves the roster and is reported with SummonedCreatureDeparted (Combatants [creature, summoner]) and a SummonedRosterLeave of RemovalCauseDisappeared (the SRD's "disappears", "reverts", "melts away"): a Giant Insect, Draconic Spirit, Otherworldly Steed or familiar drops what it carried; an Animated Object reverts to the object; a simulacrum melts to snow and makes no Death Saving Throws; an undead creature is unlinked and stays as a body with no roster instruction (SRD 5.2.1, pp. 109, 130, 135, 163, 166; "Monster Death", p. 17).
// Path: per case.
func TestACreatureAtZeroHitPointsDeparts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		id                      string
		kind                    combat.SummonKind
		drops, object, snow     bool
		body                    bool
	}{
		{"giant insect", "FR-019, p. 135", "happy", "i1", combat.SummonGiantInsect, true, false, false, false},
		{"draconic spirit", "FR-019, p. 166", "happy", "d1", combat.SummonDraconicSpirit, true, false, false, false},
		{"otherworldly steed", "FR-019, p. 130", "happy", "st", combat.SummonOtherworldlySteed, true, false, false, false},
		{"familiar", "FR-019, FR-049, p. 130", "happy", "fam", combat.SummonFamiliar, true, false, false, false},
		{"animated object", "FR-020, p. 109", "happy", "o1", combat.SummonAnimatedObject, false, true, false, false},
		{"simulacrum", "FR-024, p. 163", "happy", "sim", combat.SummonSimulacrum, false, false, true, false},
		{"animated undead", "FR-025, p. 17", "happy", "u1", combat.SummonAnimatedUndead, false, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roster := sumWorld(t)
			a := sumApplied(t, sumChange(t, roster, combat.ReportZeroHitPoints{Creature: cid(c.id)}, ""))
			deps := sumDeparted(a)
			if len(deps) != 1 || len(a.Events()) != 1 {
				t.Fatalf("events = %v, want one SummonedCreatureDeparted [%s]", sumKinds(a.Events()), c.requirement)
			}
			d := deps[0]
			if d.SummonKind() != c.kind || d.Cause() != combat.SummonEndDroppedToZeroHitPoints || d.PendingRemoval() || d.DropsCarriedItems() != c.drops ||
				d.RevertsToObject() != c.object || d.RevertsToSnow() != c.snow || d.LeavesBody() != c.body || !slices.Equal(d.Combatants(), []combat.CombatantID{cid(c.id), "pc"}) {
				t.Errorf("departure = %#v, want kind %v, drops %v, object %v, snow %v, body %v [%s]", d, c.kind, c.drops, c.object, c.snow, c.body, c.requirement)
			}
			leaves := sumLeaves(a)
			if c.body {
				if len(leaves) != 0 {
					t.Errorf("leaves = %v, want none: the body stays and dies under Monster Death [FR-025]", leaves)
				}
			} else if want := []combat.SummonedRosterLeave{{Creature: cid(c.id), Cause: combat.RemovalCauseDisappeared}}; !reflect.DeepEqual(leaves, want) {
				t.Errorf("leaves = %v, want %v [FR-006, FR-026]", leaves, want)
			}
			if _, still := a.Roster().Link(cid(c.id)); still || a.Roster().Len() != roster.Len()-1 {
				t.Errorf("the creature is still linked [FR-019]")
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-021, Table D - the overflow of an Animated Object is the sum of the post-pipeline damage of the resolutions minus the Hit Points lost minus the damage Temporary Hit Points absorbed, never negative (SRD 5.2.1, "Animate Objects", p. 109).
// Path: per case.
func TestAnAnimatedObjectsOverflow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		damage                  []int
		absorbed, lost          int
		overflow                int
	}{
		{"exact", "Table D, p. 109", "unhappy (boundary)", []int{10}, 0, 10, 0},
		{"one over", "Table D, p. 109", "unhappy (boundary)", []int{11}, 0, 10, 1},
		{"well over", "Table D, p. 109", "happy", []int{25}, 0, 10, 15},
		{"two instances", "Table D, p. 109", "unhappy (rule interaction)", []int{7, 8}, 0, 10, 5},
		{"temporary first", "Table D, p. 109", "unhappy (rule interaction)", []int{12}, 3, 6, 3},
		{"survives", "Table D, p. 109", "unhappy (boundary)", []int{4}, 0, 4, 0},
		{"more taken than dealt is never negative", "FR-021", "unhappy (boundary)", []int{3}, 5, 5, 0},
		{"no damage at all", "FR-021", "unhappy (empty case)", nil, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var res []combat.DamageResolution
			if c.damage != nil {
				res = sumResolutions(t, c.damage...)
			}
			taken := luMust(combat.NewDamageTaken(cid("o1"), res, c.absorbed, c.lost))
			if got := combat.OverflowDamage(res, taken); got != c.overflow {
				t.Errorf("OverflowDamage = %d, want %d [%s]", got, c.overflow, c.requirement)
			}
			if c.overflow == 0 && c.name == "survives" {
				return
			}
			a := sumApplied(t, sumChange(t, sumWorld(t), combat.ReportZeroHitPoints{Creature: cid("o1"), Damage: res, Taken: taken}, ""))
			if got := sumDeparted(a)[0]; got.Overflow() != c.overflow || !got.RevertsToObject() {
				t.Errorf("event overflow = %d reverts %v, want %d and true [FR-020, FR-021]", got.Overflow(), got.RevertsToObject(), c.overflow)
			}
		})
	}
	huge := luMust(combat.NewDamageTaken(cid("o1"), nil, math.MaxInt, math.MaxInt))
	if got := combat.OverflowDamage(sumResolutions(t, 5), huge); got != 0 {
		t.Errorf("OverflowDamage with saturated amounts = %d, want 0 [plan P10: overflow]", got)
	}
	a := sumApplied(t, sumChange(t, sumWorld(t), combat.ReportZeroHitPoints{Creature: cid("i1"), Damage: sumResolutions(t, 50)}, ""))
	if sumDeparted(a)[0].Overflow() != 0 {
		t.Errorf("an insect has overflow [FR-021]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-026, FR-077, Table P (active combatant drops, other combatant drops) - a departing creature that is the active combatant leaves the link at once but its departure is pending: PendingRemoval and AfterTurn are true, so the host ends the turn before removing it; otherwise both are false.
// Path: per case.
func TestTheActiveCombatantsDepartureIsPending(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		active                  string
		pending                 bool
	}{
		{"active combatant drops", "Table P, p. 135", "unhappy (state error)", "i1", true},
		{"other combatant drops", "Table P, p. 135", "happy", "pc", false},
		{"nobody is active", "FR-077", "unhappy (empty case)", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roster := sumWorld(t)
			a := sumApplied(t, sumChange(t, roster, combat.ReportZeroHitPoints{Creature: cid("i1")}, c.active))
			if d := sumDeparted(a)[0]; d.PendingRemoval() != c.pending {
				t.Errorf("PendingRemoval = %v, want %v [%s]", d.PendingRemoval(), c.pending, c.requirement)
			}
			if got, want := sumLeaves(a), []combat.SummonedRosterLeave{{Creature: cid("i1"), Cause: combat.RemovalCauseDisappeared, AfterTurn: c.pending}}; !reflect.DeepEqual(got, want) {
				t.Errorf("leaves = %v, want %v [FR-077]", got, want)
			}
			if _, linked := a.Roster().Link("i1"); linked {
				t.Errorf("a pending departure keeps the link: no further command or turn plan is possible [FR-077]")
			}
			if _, ok := a.Roster().TurnPlan("i1"); ok {
				t.Errorf("a pending departure still has a turn plan [FR-077]")
			}
			if _, ok := sumChange(t, a.Roster(), sumCommand(combat.SummonGiantInsect, 0, "i1"), c.active).(combat.SummonedRejected); !ok {
				t.Errorf("a command reached a departed creature [FR-077]")
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-026 - a departure of a creature the roster does not hold, one already departed or one in the pocket dimension returns SummonedUnchanged, state and events empty, and the roster is as it was.
// Path: per case.
func TestADepartureOfAnUnknownCreatureChangesNothing(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	gone := sumApplied(t, sumChange(t, roster, combat.ReportZeroHitPoints{Creature: cid("i1")}, "")).Roster()
	pocket := sumApplied(t, sumChange(t, roster, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	for name, c := range map[string]struct {
		roster combat.SummonedRoster
		id     string
	}{"unknown": {roster, "ghost"}, "already departed": {gone, "i1"}, "in the pocket dimension": {pocket, "fam"}, "in an empty roster": {combat.SummonedRoster{}, "i1"}} {
		before := c.roster
		if got := sumChange(t, c.roster, combat.ReportZeroHitPoints{Creature: cid(c.id)}, "").(combat.SummonedUnchanged); got != (combat.SummonedUnchanged{}) || !before.Equal(c.roster) {
			t.Errorf("%s: transition = %#v, want SummonedUnchanged [FR-026]", name, got)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-022, FR-027, Table P (spirit killed spell runs; spell ends after spirit gone; spell ends spirit present; objects at spell end; simulacrum) - the end of a spell departs the present creatures of that summoner and spell whose kind lasts for it (cause SpellEnded) and ends the summoner's shared Resistance, whether or not the spirit is still present; a spirit dropping to 0 Hit Points leaves the grant; a simulacrum ends on dispelling only; the Instantaneous kinds are not touched (SRD 5.2.1, "Summon Dragon", p. 166; "Animate Objects", p. 109; "Simulacrum", p. 162).
// Path: per case.
func TestTheEndOfASpell(t *testing.T) {
	t.Parallel()
	end := func(spell string, cause combat.SummonSpellEnd) combat.ReportSpellEnded {
		return combat.ReportSpellEnded{Summoner: cid("pc"), Spell: content.SpellID(spell), Cause: cause}
	}
	world := sumWorld(t)
	killed := sumApplied(t, sumChange(t, world, combat.ReportZeroHitPoints{Creature: cid("d1")}, "")).Roster()
	grant := combat.SharedResistanceGrant{Summoner: cid("pc"), Spell: "summon_dragon", Damage: content.DamageCold}
	if got := killed.SharedResistances(); !slices.Equal(got, []combat.SharedResistanceGrant{grant}) {
		t.Fatalf("grants after the spirit died = %v, want the grant kept [FR-027, Table P]", got)
	}
	if got := world.SharedResistances(); !slices.Equal(got, []combat.SharedResistanceGrant{grant}) {
		t.Errorf("grants = %v, want the grant recorded [FR-027]", got)
	}
	cases := []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		change                  combat.ReportSpellEnded
		departed                []string
		events                  []string
		unchanged               bool
	}{
		{"spell ends, spirit present", "Table P, p. 166", "happy", world, end("summon_dragon", combat.SummonSpellEndConcentrationEnded), []string{"d1"}, []string{"summoned_creature_departed", "shared_resistance_ended"}, false},
		{"spell ends after the spirit is gone", "Table P, p. 166", "unhappy (rule interaction)", killed, end("summon_dragon", combat.SummonSpellEndDurationElapsed), nil, []string{"shared_resistance_ended"}, false},
		{"objects at spell end", "Table P, p. 109", "unhappy (rule interaction)", world, end("animate_objects", combat.SummonSpellEndConcentrationEnded), []string{"o1", "o2"}, []string{"summoned_creature_departed", "summoned_creature_departed"}, false},
		{"insect at spell end", "FR-022, p. 135", "happy", world, end("giant_insect", combat.SummonSpellEndConcentrationEnded), []string{"i1"}, []string{"summoned_creature_departed"}, false},
		{"a simulacrum when dispelled", "FR-022, p. 162", "unhappy (rule interaction)", world, end("simulacrum", combat.SummonSpellEndDispelled), []string{"sim"}, []string{"summoned_creature_departed"}, false},
		{"a simulacrum when Concentration ends", "FR-022, p. 162", "unhappy (rule interaction)", world, end("simulacrum", combat.SummonSpellEndConcentrationEnded), nil, nil, true},
		{"a simulacrum when the duration runs out", "FR-022, p. 162", "unhappy (rule interaction)", world, end("simulacrum", combat.SummonSpellEndDurationElapsed), nil, nil, true},
		{"Find Steed is Instantaneous", "FR-022, p. 130", "unhappy (rule interaction)", world, end("find_steed", combat.SummonSpellEndDispelled), nil, nil, true},
		{"Animate Dead is Instantaneous", "FR-022, p. 108", "unhappy (rule interaction)", world, end("animate_dead", combat.SummonSpellEndDispelled), nil, nil, true},
		{"Find Familiar is Instantaneous", "FR-022, p. 130", "unhappy (rule interaction)", world, end("find_familiar", combat.SummonSpellEndDispelled), nil, nil, true},
		{"another summoner's spell", "FR-022", "unhappy (state error)", world, combat.ReportSpellEnded{Summoner: cid("other"), Spell: "giant_insect", Cause: combat.SummonSpellEndDispelled}, nil, nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumChange(t, c.roster, c.change, "")
			if c.unchanged {
				if _, ok := got.(combat.SummonedUnchanged); !ok {
					t.Errorf("transition = %#v, want SummonedUnchanged [%s]", got, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			var ids []string
			for _, l := range sumLeaves(a) {
				ids = append(ids, string(l.Creature))
			}
			if !slices.Equal(ids, c.departed) || !slices.Equal(sumKinds(a.Events()), c.events) {
				t.Errorf("leaves %v events %v, want %v and %v [%s]", ids, sumKinds(a.Events()), c.departed, c.events, c.requirement)
			}
			for _, d := range sumDeparted(a) {
				if d.Cause() != combat.SummonEndSpellEnded || d.RevertsToObject() != (d.SummonKind() == combat.SummonAnimatedObject) || d.Overflow() != 0 {
					t.Errorf("departure = %#v, want SpellEnded, reverting to an object only for an object, with no overflow [FR-020, FR-022]", d)
				}
			}
			if len(a.Roster().SharedResistances()) != 0 && c.change.Spell == "summon_dragon" {
				t.Errorf("the shared Resistance outlived its spell [FR-027]")
			}
		})
	}
	a := sumApplied(t, sumChange(t, world, end("summon_dragon", combat.SummonSpellEndConcentrationEnded), ""))
	if e := a.Events()[1].(combat.SharedResistanceEnded); e.Grant() != grant || !slices.Equal(e.Combatants(), []combat.CombatantID{"pc"}) {
		t.Errorf("SharedResistanceEnded = %#v, want the grant and [pc] [FR-027]", e)
	}
	if again := sumChange(t, a.Roster(), end("summon_dragon", combat.SummonSpellEndDispelled), ""); !reflect.DeepEqual(again, combat.SummonedUnchanged{}) {
		t.Errorf("a second end of the spell = %#v, want SummonedUnchanged [FR-026]", again)
	}
	for _, c := range []combat.SummonSpellEnd{combat.SummonSpellEndConcentrationEnded, combat.SummonSpellEndDurationElapsed, combat.SummonSpellEndDispelled, 0} {
		if (c.String() == "unknown") == c.Valid() {
			t.Errorf("SummonSpellEnd %d String %q Valid %v disagree", int(c), c.String(), c.Valid())
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-023 - the summoner's death departs an Otherworldly Steed (cause SummonerDied, it drops what it carried) and no other kind: a familiar, simulacrum or undead creature persists (SRD 5.2.1, "Find Steed", p. 130).
// Path: per case.
func TestTheSummonersDeathDepartsOnlyASteed(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	a := sumApplied(t, sumChange(t, roster, combat.ReportSummonerDied{Summoner: cid("pc")}, "st"))
	d := sumDeparted(a)
	if len(d) != 1 || d[0].SummonKind() != combat.SummonOtherworldlySteed || d[0].Cause() != combat.SummonEndSummonerDied || !d[0].DropsCarriedItems() || !d[0].PendingRemoval() {
		t.Errorf("departures = %#v, want the steed, SummonerDied, dropping its items, pending as the active combatant [FR-023]", d)
	}
	if a.Roster().Len() != roster.Len()-1 {
		t.Errorf("roster has %d links, want the steed only gone [FR-023]", a.Roster().Len())
	}
	if _, ok := sumChange(t, a.Roster(), combat.ReportSummonerDied{Summoner: cid("pc")}, "").(combat.SummonedUnchanged); !ok {
		t.Errorf("a second death changed the roster [FR-026]")
	}
	if _, ok := sumChange(t, roster, combat.ReportSummonerDied{Summoner: cid("nobody")}, "").(combat.SummonedUnchanged); !ok {
		t.Errorf("the death of a stranger changed the roster [FR-023]")
	}
	if a2 := sumApplied(t, sumChange(t, roster, combat.ReportSummonerDied{Summoner: cid("pc")}, "")); sumDeparted(a2)[0].PendingRemoval() {
		t.Errorf("a steed that is not active has a pending removal")
	}
}
