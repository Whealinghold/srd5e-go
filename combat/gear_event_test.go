package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// gearEvents returns one event of each of the nine kinds, produced by the real rules.
func gearEvents(t *testing.T) []combat.GearEvent {
	t.Helper()
	var out []combat.GearEvent
	dep := func(tr combat.DeployedGearsTransition, err error) {
		out = append(out, gearsApplied(t, tr, err).Events()...)
	}
	hld := func(tr combat.GearHoldsTransition, err error) combat.GearHoldsApplied {
		a := holdsApplied(t, tr, err)
		out = append(out, a.Events()...)
		return a
	}
	dep(combat.DeployedGears{}.Deploy(gearID("x"), combat.DeployedGearKindCaltrops, gid("owner"), 0, combat.GroundLevelLevel))
	dep(gearOne(t, combat.DeployedGearKindCaltrops).Recover(gearID("g")), nil)
	dep(gearOne(t, combat.DeployedGearKindPouredOil).Light(gearID("g"), gearStamp(1, "a")))
	burned, _ := gearBurning(t, 1, "a").Expire(gearStamp(3, "a"))
	out = append(out, burned.Events()...)
	dep(gearOne(t, combat.DeployedGearKindHuntingTrap).Spring(gearID("g"), gid("walker")))
	dep(gearSprung(t).Reset(gearID("g"), combat.GearHolds{}), nil)
	cuffs := holdIn("m", combat.HoldKindManacles)
	cuffs.Restraint = combat.HoldRestraintDoesNotRestrain
	began := hld(combat.GearHolds{}.Begin(gearHold(t, cuffs)))
	hld(began.Holds().Reanchor("m", combat.HoldRestraintRestrains))
	hld(began.Holds().End("m", combat.HoldEndCauseUnlocked))
	return out
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-063 - the nine events are GearEvent and Event members with the kinds deployed_gear_placed, deployed_gear_recovered, poured_oil_lit, poured_oil_burned_out, hunting_trap_sprung, hunting_trap_reset, gear_hold_started, gear_hold_restraint_changed and gear_hold_ended, unique, past tense, with Combatants a fresh copy with the deployer first and no duplicates, and equal by value.
// Path: unhappy (immutability, duplicate).
func TestGearEventFamily(t *testing.T) {
	t.Parallel()
	events := gearEvents(t)
	kinds := []string{"deployed_gear_placed", "deployed_gear_recovered", "poured_oil_lit", "poured_oil_burned_out", "hunting_trap_sprung",
		"hunting_trap_reset", "gear_hold_started", "gear_hold_restraint_changed", "gear_hold_ended"}
	if len(events) != len(kinds) {
		t.Fatalf("got %d events, want %d", len(events), len(kinds))
	}
	wantWho := [][]combat.CombatantID{{gid("owner")}, {gid("owner")}, {gid("owner"), gid("a")}, {gid("owner"), gid("a")}, {gid("owner"), gid("walker")},
		{gid("owner")}, {gid("victim")}, {gid("victim")}, {gid("victim")}}
	wide := make([]combat.Event, len(events))
	for i, e := range events {
		var _ combat.Event = e
		combattest.CheckEvent(t, e, kinds[i], wantWho[i]...)
		wide[i] = e
	}
	combattest.CheckEventKinds(t, wide)
	again := gearEvents(t)
	for i := range events {
		if events[i] != again[i] {
			t.Errorf("%s: events built from equal inputs must be equal", kinds[i])
		}
	}
	// One creature that deployed and lit the oil is named once.
	solo, _ := gearOne(t, combat.DeployedGearKindPouredOil).Light(gearID("g"), combat.MustTurnStamp(1, gid("owner")))
	if got := gearsApplied(t, solo, nil).Events()[0].Combatants(); len(got) != 1 {
		t.Errorf("Combatants = %v, want the deployer once", got)
	}
}
