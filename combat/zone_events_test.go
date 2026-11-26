package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Each zone event variant is a combat Event, named in the past tense with a unique
// snake_case kind (persistent-spell-zones FR-050).
var (
	_ combat.Event     = combat.ZoneCreated{}
	_ combat.Event     = combat.ZoneTriggered{}
	_ combat.Event     = combat.ZoneMoved{}
	_ combat.Event     = combat.ZoneEnded{}
	_ combat.ZoneEvent = combat.ZoneEnded{}
)

// Requirement: doc/specs/persistent-spell-zones.md FR-050 - every zone event kind is unique
// in the package, and each event lists the creatures the spec names: the caster for
// created, moved and ended, the affected creature for triggered.
// Path: happy.
func TestZoneEventKindsAreUniqueAndCombatantsAsSpecified(t *testing.T) {
	t.Parallel()
	created := pzCreate(t, pzSpirit(), nil)
	createdEv := pzCreatedEvent(t)
	triggered := pzAssessed(t, combat.AssessZoneTrigger(created, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))).Events[0]
	ended := combat.EndZone(created, combat.ZoneEndDispelled).(combat.EndZoneEnded).Event
	moved := combat.MoveZone(pzFlamingSphere(t), luMust(combat.NewCasterMove(5, content.BonusAction)), pzTurn(1, "caster")).(combat.MoveZoneMoved).Event

	events := []combat.Event{createdEv, triggered, moved, ended}
	kinds := make([]string, 0, len(events))
	for _, e := range events {
		kinds = append(kinds, e.Kind())
	}
	if !slices.Equal(kinds, []string{"zone_created", "zone_triggered", "zone_moved", "zone_ended"}) {
		t.Fatalf("kinds = %v, want four unique snake_case kinds [FR-050]", kinds)
	}
	if c := triggered.Combatants(); len(c) != 1 || c[0] != combat.MustCombatantID("x") {
		t.Fatalf("triggered combatants = %v, want the affected creature [FR-050]", c)
	}
	for _, e := range []combat.Event{createdEv, moved, ended} {
		if c := e.Combatants(); len(c) != 1 || c[0] != combat.MustCombatantID("caster") {
			t.Fatalf("%s combatants = %v, want the caster [FR-050]", e.Kind(), c)
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-050 - Combatants returns a fresh slice:
// mutating it changes nothing the event holds (FR-052).
// Path: unhappy (immutability).
func TestZoneEventCombatantsAreCopies(t *testing.T) {
	t.Parallel()
	created := pzCreatedEvent(t)
	ids := created.Combatants()
	ids[0] = combat.MustCombatantID("someone")
	if created.Combatants()[0] != combat.MustCombatantID("caster") {
		t.Fatalf("Combatants() aliases the event's caster [FR-052]")
	}
}

// pzCreatedEvent returns the ZoneCreated event of a zone created from pzSpirit.
func pzCreatedEvent(t *testing.T) combat.ZoneCreated {
	t.Helper()
	res, ok := combat.CreateZone(combat.CreateZoneCommand{
		ID: luMust(combat.NewPersistentZoneID("zone")), Caster: combat.MustCombatantID("caster"),
		Spell: pzSpirit(), Shape: luMust(content.NewSphereArea(20)), Cast: pzTurn(1, "caster"),
	}).(combat.CreateZoneCreated)
	if !ok {
		t.Fatalf("CreateZone did not create the zone")
	}
	return res.Event
}
