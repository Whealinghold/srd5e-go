package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// pzMovementZone creates a stationary zone with movement facts dt, speed halved and extra.
func pzMovementZone(t *testing.T, id string, dt, halved bool, extra int) combat.PersistentZone {
	t.Helper()
	facts := pzFacts(content.ZoneFactsFields{
		Rules:    []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)},
		Movement: luMust(content.NewZoneMovementFacts(dt, halved, extra)),
	})
	return pzCreate(t, pzSpell(id, 1, facts, nil), nil)
}

// Requirement: doc/specs/persistent-spell-zones.md FR-044 - only active zones where the
// creature is in the body region and is not designated count (SRD 5.2.1, "Spirit
// Guardians", p. 164: "Any other creature's Speed is halved in the Emanation").
// Path: per case.
func TestMovementEffectsCountOnlyActiveBodyUndesignatedZones(t *testing.T) {
	t.Parallel()
	halving := pzMovementZone(t, "halving", false, true, 0)
	ended := combat.EndZone(halving, combat.ZoneEndDispelled).(combat.EndZoneEnded).Zone
	designated := pzCreate(t, pzSpell("halving_d", 1, pzFacts(content.ZoneFactsFields{
		Rules:       []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)},
		Movement:    luMust(content.NewZoneMovementFacts(false, true, 0)),
		Designation: content.DesignationAtCastingOnly,
	}), nil), func(c *combat.CreateZoneCommand) { c.Designated = []combat.CombatantID{combat.MustCombatantID("x")} })
	cases := []struct {
		name, requirement, path string
		zones                   []combat.PersistentZone
		creature                combat.ZoneCreatureFacts
		halved                  bool
	}{
		{"body, active, not designated", "FR-044 the Speed is halved", "happy", []combat.PersistentZone{halving}, pzInBody("x"), true},
		{"ended zone", "FR-044 an ended zone is not considered", "unhappy (state error)", []combat.PersistentZone{ended}, pzInBody("x"), false},
		{"side region only", "FR-044 the body region only", "unhappy (boundary)", []combat.PersistentZone{halving}, pzCreature("x", false, false, "side"), false},
		{"designated creature", "FR-044 an exempt creature is not affected", "unhappy (rule interaction)", []combat.PersistentZone{designated}, pzInBody("x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.ZoneMovementEffectsFor(c.zones, c.creature)
			if got.SpeedHalved() != c.halved || got.None() == c.halved {
				t.Fatalf("ZoneMovementEffectsFor halved = %t, want %t [%s]", got.SpeedHalved(), c.halved, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-045 and FR-046 - Difficult Terrain and
// a halved Speed are flags: two zones do not stack them (SRD 5.2.1, "Difficult Terrain",
// p. 181, "isn't cumulative"; Q-7).
// Path: unhappy (rule interaction).
func TestDifficultTerrainAndSpeedDoNotStack(t *testing.T) {
	t.Parallel()
	a := pzMovementZone(t, "dt_a", true, true, 0)
	b := pzMovementZone(t, "dt_b", true, true, 0)
	got := combat.ZoneMovementEffectsFor([]combat.PersistentZone{a, b}, pzInBody("x"))
	if !got.DifficultTerrain() || !got.SpeedHalved() {
		t.Fatalf("two zones gave difficult %t halved %t, want true true [FR-045 FR-046]", got.DifficultTerrain(), got.SpeedHalved())
	}
	if got.ExtraCostPerFoot() != 1 {
		t.Fatalf("two Difficult Terrain zones cost %d extra per foot, want 1 (not cumulative) [FR-045]", got.ExtraCostPerFoot())
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-047 - the extra cost per foot is the
// larger of 1 (Difficult Terrain) and the extra feet, never their sum: Wall of Thorns
// (extra 3) with Spike Growth (Difficult Terrain) costs 4 per foot, which is 3 extra
// (SRD 5.2.1, "Wall of Thorns", p. 173; "Difficult Terrain", p. 181).
// Path: unhappy (boundary).
func TestExtraCostIsTheLargerNotTheSum(t *testing.T) {
	t.Parallel()
	thorns := pzMovementZone(t, "thorns", false, false, 3)
	spike := pzMovementZone(t, "spike", true, false, 0)
	cases := []struct {
		name, requirement, path string
		zones                   []combat.PersistentZone
		extra                   int
	}{
		{"thorns alone", "FR-047 extra three", "happy", []combat.PersistentZone{thorns}, 3},
		{"spike growth alone", "FR-047 Difficult Terrain costs one", "happy", []combat.PersistentZone{spike}, 1},
		{"thorns and spike growth", "FR-047 4 per foot, not 5", "unhappy (boundary)", []combat.PersistentZone{thorns, spike}, 3},
		{"no zone", "FR-048 empty", "unhappy (empty and minimal)", nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.ZoneMovementEffectsFor(c.zones, pzInBody("x"))
			if got.ExtraCostPerFoot() != c.extra {
				t.Fatalf("ExtraCostPerFoot = %d, want %d [%s]", got.ExtraCostPerFoot(), c.extra, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-047 - the largest extra feet among the
// zones is the merged figure (0 when none).
// Path: happy.
func TestExtraFeetPerFootIsTheMaximum(t *testing.T) {
	t.Parallel()
	small := pzMovementZone(t, "small", false, false, 1)
	large := pzMovementZone(t, "large", false, false, 3)
	if got := combat.ZoneMovementEffectsFor([]combat.PersistentZone{small, large}, pzInBody("x")).ExtraFeetPerFoot(); got != 3 {
		t.Fatalf("ExtraFeetPerFoot = %d, want 3 [FR-047]", got)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-048 - a creature outside every zone, or
// inside only ended or designating zones, gets the zero ZoneMovementEffects (none).
// Path: unhappy (empty and minimal).
func TestCreatureWithNoCountedZoneGetsNone(t *testing.T) {
	t.Parallel()
	zone := pzMovementZone(t, "none", true, true, 2)
	got := combat.ZoneMovementEffectsFor([]combat.PersistentZone{zone}, pzCreature("x", false, false))
	if !got.None() || got.DifficultTerrain() || got.SpeedHalved() || got.ExtraFeetPerFoot() != 0 {
		t.Fatalf("creature outside every zone = %+v, want none [FR-048]", got)
	}
}
