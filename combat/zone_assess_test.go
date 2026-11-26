package combat_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// pzWeb is Web (SRD 5.2.1, p. 174): the first entry on a turn, and starting a turn
// inside; no once-per-turn ledger.
func pzWeb() content.ZoneFacts {
	return pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRule(content.CreatureFirstEntryOnTurn, 0, content.AnyCreature),
		pzRule(content.CreatureStartsTurnInside, 0, content.AnyCreature),
	}, Movement: luMust(content.NewZoneMovementFacts(true, false, 0))})
}

// pzEndsInside is a zone with one rule: ends its turn inside, part 0, no ledger.
func pzEndsInside(once bool, scope content.ZoneTargetScope) content.ZoneFacts {
	return pzFacts(content.ZoneFactsFields{
		Rules:       []content.ZoneRuleFact{pzRule(content.CreatureEndsTurnInside, 0, scope)},
		OncePerTurn: once,
	})
}

// pzAssessEvent assesses event on a zone at turn and returns the due result.
func pzAssessEvent(t *testing.T, zone combat.PersistentZone, turn combat.TurnMarker, ev combat.ZoneTriggerEvent) combat.AssessZoneTriggerDue {
	t.Helper()
	return pzAssessed(t, combat.AssessZoneTrigger(zone, turn, ev))
}

// Requirement: doc/specs/persistent-spell-zones.md FR-011 - a creature is inside a rule
// exactly when its regions contain the rule's region; an empty set is outside
// (SRD 5.2.1, "Area of Effect", p. 177; "Wall of Fire", p. 172).
// Path: per case.
func TestRuleRegionMembership(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, "side")}})
	zone := pzCreate(t, pzSpell("wall", 1, facts, nil), nil)
	cases := []struct {
		name, requirement, path string
		regions                 []string
		due                     bool
	}{
		{"in the side region", "FR-011 inside the rule's region", "happy", []string{"side"}, true},
		{"in the body only", "FR-011 another region does not count", "unhappy (boundary)", []string{"body"}, false},
		{"outside every region", "FR-011 empty set is outside", "unhappy (boundary)", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ev := combat.NewCreatureEndedTurn(pzCreature("x", false, false, c.regions...))
			got := pzAssessEvent(t, zone, pzTurn(2, "x"), ev)
			if (len(got.Mandatory) == 1) != c.due {
				t.Fatalf("due = %d, want due %t [%s]", len(got.Mandatory), c.due, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-012 and L6 - a designated creature
// never has an effect due, whatever trigger, region or ledger state.
// Path: unhappy (rule interaction).
func TestDesignatedCreatureIsNeverAffected(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), func(c *combat.CreateZoneCommand) {
		c.Designated = []combat.CombatantID{combat.MustCombatantID("ally")}
	})
	got := pzAssessEvent(t, zone, pzTurn(2, "ally"), combat.NewCreatureEndedTurn(pzInBody("ally")))
	if len(got.Mandatory) != 0 || len(got.Skipped) != 1 || got.Skipped[0].Reason != combat.SkipDesignatedExempt {
		t.Fatalf("designated creature = due %d skipped %v, want skip DesignatedExempt [FR-012]", len(got.Mandatory), got.Skipped)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-013 - EnemiesOfCaster skips a
// creature that is not an enemy (NotAnEnemy); CreaturesCasterCanSee skips one the
// caster cannot see (NotSeenByCaster); AnyCreature skips neither (SRD 5.2.1,
// "Guardian of Faith", p. 137; "Conjure Woodland Beings", p. 118).
// Path: per case.
func TestTargetScopeSkipsByEnemyAndSight(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		scope                   content.ZoneTargetScope
		enemy, seen             bool
		want                    combat.ZoneSkipReason
		due                     bool
	}{
		{"enemy, enemies scope", "FR-013 enemy is acted on", "happy", content.EnemiesOfCaster, true, true, 0, true},
		{"ally, enemies scope", "FR-013 NotAnEnemy", "unhappy (rule interaction)", content.EnemiesOfCaster, false, true, combat.SkipNotAnEnemy, false},
		{"unseen, seen scope", "FR-013 NotSeenByCaster", "unhappy (rule interaction)", content.CreaturesCasterCanSee, true, false, combat.SkipNotSeenByCaster, false},
		{"unseen ally, any scope", "FR-013 AnyCreature skips neither", "happy", content.AnyCreature, false, false, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			zone := pzCreate(t, pzSpell("s", 1, pzEndsInside(false, c.scope), nil), nil)
			got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzCreature("x", c.enemy, c.seen, "body")))
			if (len(got.Mandatory) == 1) != c.due {
				t.Fatalf("due = %d, want %t [%s]", len(got.Mandatory), c.due, c.requirement)
			}
			if !c.due && (len(got.Skipped) != 1 || got.Skipped[0].Reason != c.want) {
				t.Fatalf("skipped = %v, want %v [%s]", got.Skipped, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-014 - skip reasons apply in a fixed
// order and only the first is reported per rule; a creature failing several tests
// reports the earliest (SRD 5.2.1, "Spirit Guardians", p. 164).
// Path: unhappy (rule interaction).
func TestSkipReasonsFollowFixedOrder(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpell("s", 1, pzEndsInside(false, content.EnemiesOfCaster), nil), func(c *combat.CreateZoneCommand) {
		c.Designated = nil
	})
	// Outside the region, not an enemy, not seen: NotInside is earlier than the scopes.
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzCreature("x", false, false)))
	if len(got.Skipped) != 1 || got.Skipped[0].Reason != combat.SkipNotInside {
		t.Fatalf("skipped = %v, want only NotInside (first in order) [FR-014]", got.Skipped)
	}
	// Designated and outside: DesignatedExempt is earlier than NotInside.
	designating := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEndsTurnInside, 0, content.AnyCreature)}, Designation: content.DesignationAtCastingOnly})
	exempt := pzCreate(t, pzSpell("s", 1, designating, nil), func(c *combat.CreateZoneCommand) {
		c.Designated = []combat.CombatantID{combat.MustCombatantID("x")}
	})
	got = pzAssessEvent(t, exempt, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzCreature("x", false, false)))
	if len(got.Skipped) != 1 || got.Skipped[0].Reason != combat.SkipDesignatedExempt {
		t.Fatalf("skipped = %v, want only DesignatedExempt [FR-014]", got.Skipped)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-015 - a rule whose trigger the event
// does not satisfy is ignored without a skip reason.
// Path: happy.
func TestUnmatchedTriggerIsIgnoredWithoutSkip(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureStartedTurn(pzInBody("x")))
	if len(got.Mandatory) != 0 || len(got.Optional) != 0 || len(got.Skipped) != 0 {
		t.Fatalf("start of turn on an enters-and-ends zone = %+v, want nothing [FR-015]", got)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-016 - a creature that starts its turn
// inside is not matched by an enters trigger; a zone that moves onto a creature is
// ZoneMovedIntoSpace, never CreatureEntered (SRD 5.2.1, "Spirit Guardians", p. 164).
// Path: unhappy (rule interaction).
func TestStartedInsideAndMovedOntoAreNotEntries(t *testing.T) {
	t.Parallel()
	enters := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}})
	zone := pzCreate(t, pzSpell("e", 1, enters, nil), nil)
	if got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureStartedTurn(pzInBody("x"))); len(got.Mandatory) != 0 {
		t.Fatalf("starts-inside matched an enters rule [FR-016]")
	}
	if got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewZoneMovedIntoSpace(pzInBody("x"))); len(got.Mandatory) != 0 {
		t.Fatalf("moved-into matched an enters rule [FR-016]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-017 - two rules of one zone that match
// one event and name one part give one effect (SRD 5.2.1, "Wall of Fire", p. 172).
// Path: unhappy (rule interaction).
func TestTwoRulesOnOnePartMerge(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, "body"),
		pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, "side"),
	}})
	zone := pzCreate(t, pzSpell("wall", 1, facts, nil), nil)
	x := pzCreature("x", false, false, "body", "side")
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(x))
	if len(got.Mandatory) != 1 || len(got.Events) != 1 {
		t.Fatalf("merged rules gave %d dues and %d events, want 1 and 1 [FR-017]", len(got.Mandatory), len(got.Events))
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-018 and FR-021 - a due effect carries
// the zone, spell, cast turn, caster, creature, part, trigger, units (1 for every
// trigger but TravelsWithin) and the CasterChoice flag; dues follow rule order.
// Path: happy.
func TestDueEffectCarriesItsFacts(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRule(content.CreatureEndsTurnInside, 1, content.AnyCreature),
		pzRule(content.CreatureEndsTurnInside, 0, content.AnyCreature),
	}})
	zone := pzCreate(t, pzSpell("ordered", 2, facts, nil), func(c *combat.CreateZoneCommand) {
		c.ID = luMust(combat.NewPersistentZoneID("zone-a"))
		c.Cast = pzTurn(1, "caster")
	})
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))
	if len(got.Mandatory) != 2 {
		t.Fatalf("dues = %d, want 2 [FR-018]", len(got.Mandatory))
	}
	first := got.Mandatory[0]
	if first.PartIndex() != 1 || got.Mandatory[1].PartIndex() != 0 {
		t.Fatalf("part order = %d, %d, want 1 then 0 (declaration order) [FR-018]", first.PartIndex(), got.Mandatory[1].PartIndex())
	}
	if first.ZoneID() != luMust(combat.NewPersistentZoneID("zone-a")) || first.Spell() != "ordered" ||
		first.CastMarker() != pzTurn(1, "caster") || first.Turn() != pzTurn(2, "x") ||
		first.Combatants()[1] != combat.MustCombatantID("caster") || first.Combatants()[0] != combat.MustCombatantID("x") ||
		first.Trigger() != content.CreatureEndsTurnInside || first.Units() != 1 || first.CasterChoice() {
		t.Fatalf("due = %+v, fields not as the zone and event give [FR-018 FR-021]", first)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-019 - CreatureTravelled gives units of
// FeetInside divided by UnitFeet (whole units, no carry); fewer than one unit is
// skipped LessThanOneUnit; a negative distance is an *ZoneInvalidError (SRD 5.2.1, "Spike
// Growth", p. 164; "Breaking Up Your Move", p. 14).
// Path: per case.
func TestTravelsWithinUnitsAreWholeUnits(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzTravel(0, 5)}})
	zone := pzCreate(t, pzSpell("spike", 1, facts, nil), nil)
	cases := []struct {
		name, requirement, path string
		feet, units             int
	}{
		{"zero feet", "FR-019 less than one unit", "unhappy (boundary)", 0, 0},
		{"four feet", "FR-019 less than one unit", "unhappy (boundary)", 4, 0},
		{"five feet", "FR-019 one whole unit", "happy", 5, 1},
		{"nine feet", "FR-019 remainder not carried", "unhappy (boundary)", 9, 1},
		{"ten feet", "FR-019 two whole units", "unhappy (boundary)", 10, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ev, err := combat.NewCreatureTravelled(pzCreature("x", false, false, "body"), c.feet)
			if err != nil {
				t.Fatalf("NewCreatureTravelled(%d) = %v [%s]", c.feet, err, c.requirement)
			}
			got := pzAssessEvent(t, zone, pzTurn(2, "x"), ev)
			if c.units == 0 {
				if len(got.Mandatory) != 0 || len(got.Skipped) != 1 || got.Skipped[0].Reason != combat.SkipLessThanOneUnit {
					t.Fatalf("skipped = %v, want LessThanOneUnit [%s]", got.Skipped, c.requirement)
				}
				return
			}
			if len(got.Mandatory) != 1 || got.Mandatory[0].Units() != c.units {
				t.Fatalf("units = %v, want %d [%s]", got.Mandatory, c.units, c.requirement)
			}
		})
	}
	if _, err := combat.NewCreatureTravelled(pzCreature("x", false, false), -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewCreatureTravelled(-1) error = %v, want ErrInvalidInput [FR-019]", err)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-020 - the cause of movement does not
// change a TravelsWithin result: the event has no cause field, so forced and own
// movement of the same distance give the same result (SRD 5.2.1, "Spike Growth", p. 164).
// Path: unhappy (rule interaction).
func TestTravelCauseDoesNotChangeTheResult(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzTravel(0, 5)}})
	zone := pzCreate(t, pzSpell("spike", 1, facts, nil), nil)
	a, _ := combat.NewCreatureTravelled(pzCreature("x", false, false, "body"), 10)
	b, _ := combat.NewCreatureTravelled(pzCreature("x", false, false, "body"), 10)
	if !reflect.DeepEqual(pzAssessEvent(t, zone, pzTurn(2, "x"), a), pzAssessEvent(t, zone, pzTurn(2, "x"), b)) {
		t.Fatalf("the same travel gave different results [FR-020]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-022 and FR-023 - an optional effect is
// returned in Optional, not Mandatory, and the ledger is not updated by assessing it;
// ApplyOptionalZoneEffect records it and refuses a repeat on the same turn (SRD 5.2.1,
// "Conjure Woodland Beings", p. 118: "you can force"; Q-2).
// Path: unhappy (rule interaction).
func TestOptionalEffectIsRecordedOnlyWhenApplied(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, Scope: content.AnyCreature, CasterChoice: true})),
	}, OncePerTurn: true})
	zone := pzCreate(t, pzSpell("woodland", 1, facts, nil), nil)
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))
	if len(got.Mandatory) != 0 || len(got.Optional) != 1 || got.Zone.Ledger().Size() != 0 {
		t.Fatalf("optional assess = mandatory %d optional %d ledger %d, want 0 1 0 [FR-023]", len(got.Mandatory), len(got.Optional), got.Zone.Ledger().Size())
	}
	applied, ok := combat.ApplyOptionalZoneEffect(zone, pzTurn(2, "x"), got.Optional[0]).(combat.OptionalZoneEffectApplied)
	if !ok || applied.Zone.Ledger().Size() != 1 {
		t.Fatalf("ApplyOptionalZoneEffect = %#v, want applied with one ledger entry [FR-023]", applied)
	}
	again := combat.ApplyOptionalZoneEffect(applied.Zone, pzTurn(2, "x"), got.Optional[0])
	if r, ok := again.(combat.OptionalZoneEffectRefused); !ok || r.Reason != combat.SkipAlreadyAffectedThisTurn {
		t.Fatalf("second ApplyOptionalZoneEffect = %#v, want refused AlreadyAffectedThisTurn [FR-023]", again)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-024 - AssessZoneTrigger records each
// mandatory due effect in the ledger it returns and emits one ZoneTriggered per due
// effect, mandatory or optional.
// Path: happy.
func TestAssessRecordsAndEmitsOneEventPerDue(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))
	if at, ok := got.Zone.Ledger().Applied(combat.MustCombatantID("x")); !ok || at != pzTurn(2, "x") {
		t.Fatalf("ledger for x = %v, %t, want turn (2,x) [FR-024]", at, ok)
	}
	if len(got.Events) != 1 || got.Events[0].Kind() != "zone_triggered" {
		t.Fatalf("events = %v, want one zone_triggered [FR-024]", got.Events)
	}
	if c := got.Events[0].Combatants(); len(c) != 1 || c[0] != combat.MustCombatantID("x") {
		t.Fatalf("ZoneTriggered.Combatants() = %v, want the creature [FR-050]", c)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-025 and table L (L1 to L6) - a
// once-per-turn zone applies at most once per creature per turn across all rules
// and triggers (SRD 5.2.1, "Cloudkill", p. 116; "Symbol", p. 167).
// Path: per case.
func TestOncePerTurnLedgerTableL(t *testing.T) {
	t.Parallel()
	x := pzInBody("x")
	caster := pzInBody("caster")
	cases := []struct {
		name, requirement, path string
		events                  []combat.ZoneTriggerEvent
		turns                   []combat.TurnMarker
		wantDue                 []int
	}{
		{"L1 ends its turn inside", "FR-025 one application", "happy",
			[]combat.ZoneTriggerEvent{combat.NewCreatureEndedTurn(x)}, []combat.TurnMarker{pzTurn(2, "x")}, []int{1}},
		{"L2 enters then ends turn", "FR-025 AlreadyAffectedThisTurn", "unhappy (rule interaction)",
			[]combat.ZoneTriggerEvent{combat.NewCreatureEntered(x), combat.NewCreatureEndedTurn(x)},
			[]combat.TurnMarker{pzTurn(2, "x"), pzTurn(2, "x")}, []int{1, 0}},
		{"L3 moved onto, then ends turn", "FR-026 a different marker", "unhappy (boundary)",
			[]combat.ZoneTriggerEvent{combat.NewZoneMovedIntoSpace(x), combat.NewCreatureEndedTurn(x)},
			[]combat.TurnMarker{pzTurn(2, "caster"), pzTurn(2, "x")}, []int{1, 1}},
		{"L4 moved onto, then enters, both on the caster's turn", "FR-025 second skipped", "unhappy (rule interaction)",
			[]combat.ZoneTriggerEvent{combat.NewZoneMovedIntoSpace(x), combat.NewCreatureEntered(x)},
			[]combat.TurnMarker{pzTurn(2, "caster"), pzTurn(2, "caster")}, []int{1, 0}},
		{"L5 ends turn on two rounds", "FR-026 a later round allows", "unhappy (boundary)",
			[]combat.ZoneTriggerEvent{combat.NewCreatureEndedTurn(x), combat.NewCreatureEndedTurn(x)},
			[]combat.TurnMarker{pzTurn(2, "x"), pzTurn(3, "x")}, []int{1, 1}},
		{"L1 for the caster too", "FR-025 a caster inside is not the origin when it moves", "happy",
			[]combat.ZoneTriggerEvent{combat.NewCreatureEndedTurn(caster)}, []combat.TurnMarker{pzTurn(2, "x")}, []int{0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			zone := pzCreate(t, pzSpirit(), nil)
			for i, ev := range c.events {
				got := pzAssessEvent(t, zone, c.turns[i], ev)
				if len(got.Mandatory) != c.wantDue[i] {
					t.Fatalf("event %d dues = %d, want %d [%s]", i, len(got.Mandatory), c.wantDue[i], c.requirement)
				}
				zone = got.Zone
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-027 - a zone without OncePerTurn does not
// limit applications: every matching event applies (SRD 5.2.1, "Spike Growth", p. 164).
// Path: happy.
func TestNonOncePerTurnZoneAppliesOnEveryEvent(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpell("spike", 1, pzEndsInside(false, content.AnyCreature), nil), nil)
	x := pzInBody("x")
	for i := 0; i < 2; i++ {
		got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(x))
		if len(got.Mandatory) != 1 {
			t.Fatalf("application %d dues = %d, want 1 [FR-027]", i, len(got.Mandatory))
		}
		zone = got.Zone
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-028 and table L (L10, L11) - a first
// entry is matched once per turn; leaving and re-entering skips NotFirstEntryThisTurn;
// starting the turn inside is a different trigger (SRD 5.2.1, "Web", p. 174).
// Path: per case.
func TestFirstEntryPerTurnTableL(t *testing.T) {
	t.Parallel()
	web := pzSpell("web", 1, pzWeb(), nil)
	x := pzInBody("x")
	t.Run("L10 enters, leaves, enters again on the same turn", func(t *testing.T) {
		t.Parallel()
		zone := pzCreate(t, web, nil)
		first := pzAssessEvent(t, zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
		if len(first.Mandatory) != 1 {
			t.Fatalf("first entry dues = %d, want 1 [FR-028]", len(first.Mandatory))
		}
		second := pzAssessEvent(t, first.Zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
		if len(second.Mandatory) != 0 || len(second.Skipped) != 1 || second.Skipped[0].Reason != combat.SkipNotFirstEntryThisTurn {
			t.Fatalf("second entry = %+v, want NotFirstEntryThisTurn [FR-028]", second.Skipped)
		}
	})
	t.Run("L11 starts its turn inside after entering", func(t *testing.T) {
		t.Parallel()
		zone := pzCreate(t, web, nil)
		entered := pzAssessEvent(t, zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
		started := pzAssessEvent(t, entered.Zone, pzTurn(2, "x"), combat.NewCreatureStartedTurn(x))
		if len(started.Mandatory) != 1 {
			t.Fatalf("start-inside dues = %d, want 1 [FR-016 FR-028]", len(started.Mandatory))
		}
	})
}

// Requirement: doc/specs/persistent-spell-zones.md FR-028 - a creature that enters and leaves
// on the same turn keeps that turn's first entry: Table E has no departure event, so
// leaving is the absence of any report, and a re-entry that turn is skipped while the
// next turn's entry is due (SRD 5.2.1, "Web", p. 174).
// Path: unhappy (boundary).
func TestEnteredAndLeftOnTheSameTurnKeepsTheEntry(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpell("web", 1, pzWeb(), nil), nil)
	x := pzInBody("x")
	entered := pzAssessEvent(t, zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
	if len(entered.Mandatory) != 1 {
		t.Fatalf("first entry dues = %d, want 1 [FR-028]", len(entered.Mandatory))
	}
	// Table E has no departure event. The creature is outside when its turn ends; that
	// report matches no rule of Web, so it changes nothing.
	left := pzAssessEvent(t, entered.Zone, pzTurn(1, "x"), combat.NewCreatureEndedTurn(pzCreature("x", true, true)))
	if len(left.Mandatory) != 0 || len(left.Optional) != 0 || len(left.Skipped) != 0 {
		t.Fatalf("an ends-turn report matched Web = %+v, want nothing [FR-015 FR-028]", left)
	}
	again := pzAssessEvent(t, left.Zone, pzTurn(1, "x"), combat.NewCreatureEntered(x))
	if len(again.Mandatory) != 0 || len(again.Skipped) != 1 || again.Skipped[0].Reason != combat.SkipNotFirstEntryThisTurn {
		t.Fatalf("re-entry after leaving on the same turn = %+v, want NotFirstEntryThisTurn [FR-028]", again.Skipped)
	}
	next := pzAssessEvent(t, again.Zone, pzTurn(2, "x"), combat.NewCreatureEntered(x))
	if len(next.Mandatory) != 1 {
		t.Fatalf("entry on the next turn dues = %d, want 1 [FR-028]", len(next.Mandatory))
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-029 and table L (L7, L13) - a ledger
// is per zone: two zones of one spell give a due from each, carrying the same spell,
// different zone ids and cast turns; two spells add (SRD 5.2.1, "Combining Spell
// Effects", p. 106; Q-5).
// Path: per case.
func TestLedgerIsPerZoneTableL(t *testing.T) {
	t.Parallel()
	x := pzInBody("x")
	spirit := pzSpirit()
	cases := []struct {
		name, requirement, path string
		spellB                  content.Spell
		sameSpell               bool
	}{
		{"L7 two zones of one spell", "FR-029 no dedupe across zones", "unhappy (rule interaction)", spirit, true},
		{"L13 two zones of different spells", "FR-029 effects add", "unhappy (rule interaction)", pzSpell("other", 1, pzSpiritFacts(), nil), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := pzCreate(t, spirit, nil)
			b := pzCreate(t, c.spellB, func(cmd *combat.CreateZoneCommand) {
				cmd.ID = luMust(combat.NewPersistentZoneID("second"))
				cmd.Cast = pzTurn(1, "other-caster")
			})
			da := pzAssessEvent(t, a, pzTurn(2, "x"), combat.NewCreatureEndedTurn(x)).Mandatory
			db := pzAssessEvent(t, b, pzTurn(2, "x"), combat.NewCreatureEndedTurn(x)).Mandatory
			if len(da) != 1 || len(db) != 1 {
				t.Fatalf("dues = %d and %d, want 1 and 1 (no dedupe) [%s]", len(da), len(db), c.requirement)
			}
			if da[0].ZoneID() == db[0].ZoneID() || da[0].CastMarker() == db[0].CastMarker() {
				t.Fatalf("zone ids or cast turns not distinct [%s]", c.requirement)
			}
			if (da[0].Spell() == db[0].Spell()) != c.sameSpell {
				t.Fatalf("spell ids equal = %t, want %t [%s]", da[0].Spell() == db[0].Spell(), c.sameSpell, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-030 and table L (L8, L9) -
// RecordZoneCastApplication uses up the turn's application, so a later move on the same
// turn is skipped; a zone without OncePerTurn refuses the record (SRD 5.2.1,
// "Incendiary Cloud", p. 142; Q-6).
// Path: per case.
func TestRecordCastApplicationTableL(t *testing.T) {
	t.Parallel()
	t.Run("L8 recorded, then the same turn's move is skipped", func(t *testing.T) {
		t.Parallel()
		zone := pzCreate(t, pzSpirit(), nil)
		rec, ok := combat.RecordZoneCastApplication(zone, combat.MustCombatantID("x"), pzTurn(1, "caster")).(combat.RecordCastApplicationRecorded)
		if !ok {
			t.Fatalf("RecordZoneCastApplication not recorded [FR-030]")
		}
		got := pzAssessEvent(t, rec.Zone, pzTurn(1, "caster"), combat.NewZoneMovedIntoSpace(pzInBody("x")))
		if len(got.Mandatory) != 0 || len(got.Skipped) != 1 || got.Skipped[0].Reason != combat.SkipAlreadyAffectedThisTurn {
			t.Fatalf("move after the appearance = %+v, want AlreadyAffectedThisTurn [FR-030]", got.Skipped)
		}
	})
	t.Run("L9 on a zone without OncePerTurn", func(t *testing.T) {
		t.Parallel()
		zone := pzCreate(t, pzSpell("spike", 1, pzEndsInside(false, content.AnyCreature), nil), nil)
		if _, ok := combat.RecordZoneCastApplication(zone, combat.MustCombatantID("x"), pzTurn(1, "caster")).(combat.RecordCastApplicationNotOncePerTurn); !ok {
			t.Fatalf("RecordZoneCastApplication on a non-once zone not NotOncePerTurn [FR-030]")
		}
	})
}

// Requirement: doc/specs/persistent-spell-zones.md FR-032 and FR-055 - a turn earlier than
// the zone's cast turn, or than a turn the ledger already holds for the creature, is
// a StaleTurn result and changes nothing (SRD 5.2.1, "Combat Step by Step", p. 13).
// Path: per case.
func TestStaleTurnsAreRefused(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(2, "caster") })
	x := pzInBody("x")
	ledgered := pzAssessed(t, combat.AssessZoneTrigger(zone, pzTurn(4, "x"), combat.NewCreatureEndedTurn(x))).Zone
	cases := []struct {
		name, requirement, path string
		zone                    combat.PersistentZone
		turn                    combat.TurnMarker
	}{
		{"before the cast turn", "FR-055 round below the cast round", "unhappy (invalid input)", zone, pzTurn(1, "x")},
		{"before a turn the ledger holds", "FR-032 round below the ledger for the creature", "unhappy (invalid input)", ledgered, pzTurn(3, "x")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := combat.AssessZoneTrigger(c.zone, c.turn, combat.NewCreatureEndedTurn(x)).(combat.AssessZoneTriggerStaleTurn); !ok {
				t.Fatalf("AssessZoneTrigger not StaleTurn [%s]", c.requirement)
			}
			if _, ok := combat.RecordZoneCastApplication(c.zone, x.Combatants()[0], c.turn).(combat.RecordCastApplicationStaleTurn); !ok {
				t.Fatalf("RecordZoneCastApplication not StaleTurn [%s]", c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-008 - when a MovesWithOrigin zone's
// caster is in its own Emanation it is excluded (OriginCreature) unless the caster
// chose CreatorIncludesOrigin; a stationary zone's caster is an ordinary creature
// (SRD 5.2.1, "Emanation", p. 181; "Spirit Guardians", p. 164; "Cloudkill", p. 116).
// Path: per case.
func TestCasterOriginRuleOfEmanations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spell                   content.Spell
		includes                bool
		due                     bool
	}{
		{"moving emanation excludes its caster", "FR-008 OriginCreature", "unhappy (rule interaction)", pzSpirit(), false, false},
		{"moving emanation includes its caster on choice", "FR-008 CreatorIncludesOrigin", "happy", pzSpirit(), true, true},
		{"stationary sphere: caster is ordinary", "FR-008 a Sphere affects its caster", "happy",
			pzSpell("cloud", 1, pzEndsInside(true, content.AnyCreature), nil), false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			zone := pzCreate(t, c.spell, func(cmd *combat.CreateZoneCommand) { cmd.CreatorIncludesOrigin = c.includes })
			got := pzAssessEvent(t, zone, pzTurn(2, "caster"), combat.NewCreatureEndedTurn(pzInBody("caster")))
			if (len(got.Mandatory) == 1) != c.due {
				t.Fatalf("caster due = %d, want %t [%s]", len(got.Mandatory), c.due, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-052 - an operation does not alter its
// inputs and gives equal results for equal inputs.
// Path: unhappy (immutability).
func TestAssessDoesNotAlterItsInput(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	before := zone
	first := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))
	second := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))
	if !zone.Equal(before) || zone.Ledger().Size() != 0 {
		t.Fatalf("AssessZoneTrigger changed its input zone [FR-052]")
	}
	if len(first.Mandatory) != len(second.Mandatory) || first.Zone.Ledger().Size() != second.Zone.Ledger().Size() {
		t.Fatalf("equal inputs gave different results [FR-052]")
	}
}

// optionalRule returns a CasterChoice rule on part with trigger, in the named region.
func optionalRule(trig content.ZoneTrigger, part int, region string) content.ZoneRuleFact {
	return luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: trig, PartIndex: part, Scope: content.AnyCreature,
		Region: luMust(content.NewZoneRegionID(region)), CasterChoice: true}))
}

// Requirement: doc/specs/persistent-spell-zones.md FR-017 - two optional rules that match one event and name one part give one optional effect, and an optional rule whose part is already a mandatory due gives none (no second skip either).
// Path: unhappy (rule interaction).
func TestOptionalRulesOnOnePartMerge(t *testing.T) {
	t.Parallel()
	x := pzCreature("x", false, false, "body", "side")
	t.Run("two optional rules, one part", func(t *testing.T) {
		t.Parallel()
		facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
			optionalRule(content.CreatureEndsTurnInside, 0, "body"), optionalRule(content.CreatureEndsTurnInside, 0, "side"),
		}})
		got := pzAssessEvent(t, pzCreate(t, pzSpell("optional", 1, facts, nil), nil), pzTurn(2, "x"), combat.NewCreatureEndedTurn(x))
		if len(got.Optional) != 1 || len(got.Events) != 1 || len(got.Skipped) != 0 {
			t.Errorf("optional %d, events %d, skipped %d, want 1, 1, 0 [FR-017]", len(got.Optional), len(got.Events), len(got.Skipped))
		}
	})
	t.Run("a mandatory rule then an optional rule on the same part", func(t *testing.T) {
		t.Parallel()
		facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
			pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, "body"), optionalRule(content.CreatureEndsTurnInside, 0, "side"),
		}})
		got := pzAssessEvent(t, pzCreate(t, pzSpell("mixed", 1, facts, nil), nil), pzTurn(2, "x"), combat.NewCreatureEndedTurn(x))
		if len(got.Mandatory) != 1 || len(got.Optional) != 0 || len(got.Events) != 1 {
			t.Errorf("mandatory %d, optional %d, events %d, want 1, 0, 1 [FR-017]", len(got.Mandatory), len(got.Optional), len(got.Events))
		}
	})
}

// Requirement: doc/specs/persistent-spell-zones.md FR-023 - ApplyOptionalZoneEffect refuses a due effect of this zone, spell, cast and caster whose part has no CasterChoice rule with SkipDueNotForThisZone, and records nothing.
// Path: unhappy (invalid input).
func TestApplyOptionalZoneEffectRefusesAPartWithoutAnOptionalRule(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRule(content.CreatureEndsTurnInside, 0, content.AnyCreature), optionalRule(content.CreatureEndsTurnInside, 1, "body"),
	}})
	zone := pzCreate(t, pzSpell("two", 2, facts, nil), nil)
	got := pzAssessEvent(t, zone, pzTurn(2, "x"), combat.NewCreatureEndedTurn(pzInBody("x")))
	if len(got.Mandatory) != 1 || len(got.Optional) != 1 {
		t.Fatalf("assess = mandatory %d optional %d, want 1 and 1", len(got.Mandatory), len(got.Optional))
	}
	// The mandatory due of part 0 names this zone, spell, cast and caster, but part 0 has no optional rule.
	res := combat.ApplyOptionalZoneEffect(zone, pzTurn(2, "x"), got.Mandatory[0])
	if r, ok := res.(combat.OptionalZoneEffectRefused); !ok || r.Reason != combat.SkipDueNotForThisZone {
		t.Errorf("ApplyOptionalZoneEffect = %#v, want refused DueNotForThisZone [FR-023]", res)
	}
	if _, ok := combat.ApplyOptionalZoneEffect(zone, pzTurn(2, "x"), got.Optional[0]).(combat.OptionalZoneEffectApplied); !ok {
		t.Error("the optional part was refused")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-028 - only a zone with a FirstEntryOnTurn rule records entries: an entry into a zone without one leaves the ledger empty, so it does not consume the ledger cap.
// Path: unhappy (boundary).
func TestEntryIsRecordedOnlyForAFirstEntryZone(t *testing.T) {
	t.Parallel()
	x := pzInBody("x")
	plain := pzCreate(t, pzSpell("plain", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}}), nil), nil)
	got := pzAssessEvent(t, plain, pzTurn(1, "x"), combat.NewCreatureEntered(x))
	if len(got.Mandatory) != 1 || got.Zone.Ledger().Size() != 0 {
		t.Errorf("plain zone: dues %d, ledger %d, want 1 and 0 [FR-028]", len(got.Mandatory), got.Zone.Ledger().Size())
	}
	web := pzAssessEvent(t, pzCreate(t, pzSpell("web", 1, pzWeb(), nil), nil), pzTurn(1, "x"), combat.NewCreatureEntered(x))
	if web.Zone.Ledger().Size() != 1 {
		t.Errorf("first-entry zone: ledger %d, want 1 [FR-028]", web.Zone.Ledger().Size())
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-014 - a nil ZoneTriggerEvent does not panic: it is AssessZoneTriggerStaleTurn (the invalid class) on an active zone, the ended answer on an ended one, and the zone is unchanged.
// Path: unhappy (invalid input).
func TestAssessZoneTriggerNilEvent(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	if _, ok := combat.AssessZoneTrigger(zone, pzTurn(2, "x"), nil).(combat.AssessZoneTriggerStaleTurn); !ok {
		t.Error("a nil event on an active zone is not AssessZoneTriggerStaleTurn")
	}
	ended := combat.EndZone(zone, combat.ZoneEndCasterEnded).(combat.EndZoneEnded).Zone
	if _, ok := combat.AssessZoneTrigger(ended, pzTurn(2, "x"), nil).(combat.AssessZoneTriggerEnded); !ok {
		t.Error("a nil event on an ended zone is not AssessZoneTriggerEnded")
	}
}
