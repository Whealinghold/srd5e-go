package combat_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// zoneCreatures returns n distinct combatant ids for a zone ledger.
func zoneCreatures(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("m%03d", i)
	}
	return out
}

// zoneWithLedger returns a Spirit Guardians zone whose ledger holds the first n creatures, each recorded as affected on
// the cast turn.
func zoneWithLedger(t *testing.T, n int) combat.PersistentZone {
	t.Helper()
	zone := pzCreate(t, pzSpirit(), nil)
	for _, id := range zoneCreatures(n) {
		rec, ok := combat.RecordZoneCastApplication(zone, combat.MustCombatantID(id), pzTurn(1, "caster")).(combat.RecordCastApplicationRecorded)
		if !ok {
			t.Fatalf("recording %s: not recorded", id)
		}
		zone = rec.Zone
	}
	return zone
}

// Requirement: doc/specs/persistent-spell-zones.md Limits (plan P10) - the ledger of a zone holds at most 256 creatures: recording, assessing and applying an optional effect for a 256th creature work, a 257th is the limit_exceeded variant of the call and leaves the zone unchanged.
// Path: per case.
func TestZoneLedgerCreatureCap(t *testing.T) {
	t.Parallel()
	turn := pzTurn(1, "caster")
	at, over := zoneWithLedger(t, 255), zoneWithLedger(t, 256)
	last, extra := combat.MustCombatantID(zoneCreatures(256)[255]), combat.MustCombatantID("extra")
	t.Run("record at the cap", func(t *testing.T) {
		t.Parallel()
		if _, ok := combat.RecordZoneCastApplication(at, last, turn).(combat.RecordCastApplicationRecorded); !ok {
			t.Error("the 256th creature was not recorded")
		}
	})
	t.Run("record cap plus one", func(t *testing.T) {
		t.Parallel()
		if _, ok := combat.RecordZoneCastApplication(over, extra, turn).(combat.RecordCastApplicationLimitExceeded); !ok {
			t.Error("the 257th creature was not refused with limit_exceeded")
		}
	})
	t.Run("assess at the cap", func(t *testing.T) {
		t.Parallel()
		if _, ok := combat.AssessZoneTrigger(at, turn, combat.NewCreatureEndedTurn(pzInBody(string(last)))).(combat.AssessZoneTriggerDue); !ok {
			t.Error("the 256th creature was not assessed")
		}
	})
	t.Run("assess cap plus one", func(t *testing.T) {
		t.Parallel()
		if _, ok := combat.AssessZoneTrigger(over, turn, combat.NewCreatureEndedTurn(pzInBody("extra"))).(combat.AssessZoneTriggerLimitExceeded); !ok {
			t.Error("the 257th creature was not refused with limit_exceeded")
		}
	})
	t.Run("optional effect at the cap and cap plus one", func(t *testing.T) {
		t.Parallel()
		facts := pzFacts(content.ZoneFactsFields{OncePerTurn: true, Rules: []content.ZoneRuleFact{
			luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, Scope: content.AnyCreature, CasterChoice: true})),
		}})
		zone := pzCreate(t, pzSpell("choice", 1, facts, nil), nil)
		for _, id := range zoneCreatures(255) {
			zone = combat.RecordZoneCastApplication(zone, combat.MustCombatantID(id), turn).(combat.RecordCastApplicationRecorded).Zone
		}
		due := func(id string) combat.ZoneEffectDue {
			return pzAssessed(t, combat.AssessZoneTrigger(zone, turn, combat.NewCreatureEndedTurn(pzInBody(id)))).Optional[0]
		}
		full, ok := combat.ApplyOptionalZoneEffect(zone, turn, due("x1")).(combat.OptionalZoneEffectApplied)
		if !ok {
			t.Fatal("the 256th creature was not applied")
		}
		if _, ok := combat.ApplyOptionalZoneEffect(full.Zone, turn, due("x2")).(combat.AssessZoneTriggerLimitExceeded); !ok {
			t.Error("the 257th creature was not refused with limit_exceeded")
		}
	})
}

// Requirement: doc/specs/persistent-spell-zones.md FR-040 - EndZone refuses a reason that is not one of the six ZoneEndReason values, the zero value included, with EndZoneInvalidReason, and the zone is unchanged; the wired EndPersistentZone returns an error matching srderr.ErrInvalidInput for it.
// Path: per case.
func TestEndZoneRefusesAnInvalidReason(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	for _, r := range []combat.ZoneEndReason{0, -1, combat.ZoneEndCasterEnded + 1} {
		t.Run(r.String(), func(t *testing.T) {
			t.Parallel()
			if _, ok := combat.EndZone(zone, r).(combat.EndZoneInvalidReason); !ok {
				t.Errorf("EndZone(%d) is not EndZoneInvalidReason", int(r))
			}
			_, err := combat.NewEndPersistentZone().Run(context.Background(), combat.EndPersistentZoneCommand{Request: combat.EndZoneRequest{Zone: zone, Reason: r}})
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("Run error = %v, want ErrInvalidInput", err)
			}
		})
	}
	if zone.Ended() {
		t.Error("the zone changed")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-001 and FR-032 - CreateZone refuses a blank zone id and a zero cast TurnMarker, each as its own reason listed after the others.
// Path: per case.
func TestCreateZoneRefusesZeroIDAndCastTurn(t *testing.T) {
	t.Parallel()
	base := combat.CreateZoneCommand{ID: luMust(combat.NewPersistentZoneID("zone")), Caster: combat.MustCombatantID("caster"),
		Spell: pzSpirit(), Shape: luMust(content.NewSphereArea(20)), Cast: pzTurn(1, "caster")}
	for _, c := range []struct {
		name string
		edit func(*combat.CreateZoneCommand)
		want []combat.CreateZoneRejection
	}{
		{"zero id", func(c *combat.CreateZoneCommand) { c.ID = "" }, []combat.CreateZoneRejection{combat.CreateMissingZoneID}},
		{"blank id", func(c *combat.CreateZoneCommand) { c.ID = "  " }, []combat.CreateZoneRejection{combat.CreateMissingZoneID}},
		{"zero cast turn", func(c *combat.CreateZoneCommand) { c.Cast = combat.TurnMarker{} }, []combat.CreateZoneRejection{combat.CreateMissingCastTurn}},
		{"both", func(c *combat.CreateZoneCommand) { c.ID, c.Cast = "", combat.TurnMarker{} },
			[]combat.CreateZoneRejection{combat.CreateMissingZoneID, combat.CreateMissingCastTurn}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := base
			c.edit(&cmd)
			rej, ok := combat.CreateZone(cmd).(combat.CreateZoneRejected)
			if !ok || fmt.Sprint(rej.Reasons) != fmt.Sprint(c.want) {
				t.Errorf("CreateZone = %#v, want rejected %v", rej, c.want)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-023 - ApplyOptionalZoneEffect refuses a due effect that is not of the zone, of its cast or of an optional rule of its own with SkipDueNotForThisZone, and records nothing.
// Path: per case.
func TestApplyOptionalRefusesAForeignDue(t *testing.T) {
	t.Parallel()
	choice := luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, Scope: content.AnyCreature, CasterChoice: true}))
	facts := pzFacts(content.ZoneFactsFields{OncePerTurn: true, Rules: []content.ZoneRuleFact{choice}})
	spell := pzSpell("choice", 1, facts, nil)
	zone := pzCreate(t, spell, nil)
	other := pzCreate(t, spell, func(c *combat.CreateZoneCommand) { c.ID = "other" })
	laterCast := pzCreate(t, spell, func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(2, "caster") })
	otherCaster := pzCreate(t, spell, func(c *combat.CreateZoneCommand) { c.Caster = combat.MustCombatantID("someone") })
	mandatory := pzCreate(t, pzSpirit(), func(c *combat.CreateZoneCommand) { c.ID = "zone" })
	turn := pzTurn(2, "caster")
	dueOf := func(z combat.PersistentZone) combat.ZoneEffectDue {
		return pzAssessed(t, combat.AssessZoneTrigger(z, turn, combat.NewCreatureEndedTurn(pzInBody("x")))).Optional[0]
	}
	for _, c := range []struct {
		name string
		due  combat.ZoneEffectDue
	}{
		{"another zone", dueOf(other)}, {"another cast", dueOf(laterCast)}, {"another caster", dueOf(otherCaster)},
		{"another spell with no such optional rule", func() combat.ZoneEffectDue {
			d := pzAssessed(t, combat.AssessZoneTrigger(mandatory, turn, combat.NewCreatureEndedTurn(pzInBody("x")))).Mandatory[0]
			return d
		}()},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := combat.ApplyOptionalZoneEffect(zone, turn, c.due).(combat.OptionalZoneEffectRefused)
			if !ok || got.Reason != combat.SkipDueNotForThisZone {
				t.Errorf("ApplyOptionalZoneEffect = %#v, want refused SkipDueNotForThisZone", got)
			}
		})
	}
	if _, ok := combat.ApplyOptionalZoneEffect(zone, turn, dueOf(zone)).(combat.OptionalZoneEffectApplied); !ok {
		t.Error("the zone's own due effect was not applied")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-035 and FR-036 - MoveZone refuses a CasterMove or DriftMove of 0 feet or below (the zero value of either) with RefusalDistanceNotPositive.
// Path: per case.
func TestMoveZoneRefusesANonPositiveDistance(t *testing.T) {
	t.Parallel()
	moonbeam := pzCreate(t, pzSpell("moonbeam", 1, pzFacts(content.ZoneFactsFields{
		Mobility: luMust(content.NewMovedByCaster(60, content.MagicAction, false)),
		Rules:    []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)},
	}), nil), nil)
	cloud := pzCreate(t, pzSpell("cloud", 1, pzFacts(content.ZoneFactsFields{
		Mobility: luMust(content.NewDriftsAtTurnStart(10, content.AwayFromCaster)),
		Rules:    []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)},
	}), nil), nil)
	for _, c := range []struct {
		name string
		zone combat.PersistentZone
		move combat.ZoneMove
	}{
		{"caster move of 0", moonbeam, combat.CasterMove{}},
		{"drift of 0", cloud, combat.DriftMove{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := combat.MoveZone(c.zone, c.move, pzTurn(2, "caster")).(combat.MoveZoneRefused)
			if !ok || got.Reason != combat.RefusalDistanceNotPositive || got.Reason.String() != "distance_not_positive" {
				t.Errorf("MoveZone = %#v, want refused RefusalDistanceNotPositive", got)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-017 - under OncePerTurn, two rules that match one event and name one part give one due effect and no skip: the second rule is not reported as AlreadyAffectedThisTurn.
// Path: unhappy (rule interaction).
func TestOncePerTurnDuplicatePartIsNotASkip(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpell("wall", 1, pzFacts(content.ZoneFactsFields{OncePerTurn: true, Rules: []content.ZoneRuleFact{
		pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, "body"),
		pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, "side"),
	}}), nil), nil)
	got := pzAssessed(t, combat.AssessZoneTrigger(zone, pzTurn(1, "caster"), combat.NewCreatureEndedTurn(pzCreature("x", true, true, "body", "side"))))
	if len(got.Mandatory) != 1 || len(got.Skipped) != 0 {
		t.Errorf("assessed = %d due, skipped %v, want 1 due and no skip", len(got.Mandatory), got.Skipped)
	}
	again := pzAssessed(t, combat.AssessZoneTrigger(got.Zone, pzTurn(1, "caster"), combat.NewCreatureEndedTurn(pzCreature("x", true, true, "body", "side"))))
	if len(again.Mandatory) != 0 || len(again.Skipped) != 2 {
		t.Errorf("a second event = %d due, skipped %v, want the real AlreadyAffectedThisTurn skips", len(again.Mandatory), again.Skipped)
	}
}

// zoneRow is one spell of table Z of persistent-spell-zones.md: its shape, mobility, rules and facts.
type zoneRow struct {
	name, page    string
	shape         content.AreaShape
	mobility      content.ZoneMobility
	rules         []content.ZoneRuleFact
	once, at      bool
	budget        int
	terrain, slow bool
	extra         int
}

// Requirement: doc/specs/persistent-spell-zones.md table Z - each of the 25 SRD zone spells builds its ZoneFacts, creates a zone that keeps them, and each trigger of its rules makes exactly one effect due for a creature in the rule's regions (mandatory, or optional for a CasterChoice rule), with no skip, and a part named by several rules is due once (FR-017).
// Path: per case.
func TestTableZSpells(t *testing.T) {
	t.Parallel()
	rule := func(trig content.ZoneTrigger, scope content.ZoneTargetScope, region string, choice bool) content.ZoneRuleFact {
		f := content.ZoneRuleFields{Trigger: trig, Scope: scope, CasterChoice: choice}
		if trig == content.TravelsWithin {
			f.UnitFeet = 5
		}
		if region != "" {
			f.Region = luMust(content.NewZoneRegionID(region))
		}
		return luMust(content.NewZoneRuleFact(f))
	}
	any, enemies, seen := content.AnyCreature, content.EnemiesOfCaster, content.CreaturesCasterCanSee
	enters, first, ends, starts, moves, travels := content.CreatureEnters, content.CreatureFirstEntryOnTurn,
		content.CreatureEndsTurnInside, content.CreatureStartsTurnInside, content.ZoneMovesIntoSpace, content.TravelsWithin
	sphere := func(r int) content.AreaShape { return luMust(content.NewSphereArea(r)) }
	cube := func(s int) content.AreaShape { return luMust(content.NewCubeArea(s)) }
	cylinder := func(r, h int) content.AreaShape { return luMust(content.NewCylinderArea(r, h)) }
	emanation := func(d int) content.AreaShape { return luMust(content.NewEmanationArea(d)) }
	line := func(l, w int) content.AreaShape { return luMust(content.NewLineArea(l, w)) }
	withOrigin := content.MovesWithOrigin{}
	drift := func(d content.DriftDirectionRule) content.ZoneMobility {
		return luMust(content.NewDriftsAtTurnStart(10, d))
	}
	byCaster := func(feet int, a content.ZoneActionKind, later bool) content.ZoneMobility {
		return luMust(content.NewMovedByCaster(feet, a, later))
	}
	three := func(scope content.ZoneTargetScope, choice bool) []content.ZoneRuleFact {
		return []content.ZoneRuleFact{rule(moves, scope, "", choice), rule(enters, scope, "", choice), rule(ends, scope, "", choice)}
	}
	cloud := []content.ZoneRuleFact{rule(enters, any, "", false), rule(ends, any, "", false), rule(moves, any, "", false)}
	rows := []zoneRow{
		{name: "Spirit Guardians", page: "pp. 164-165", shape: emanation(15), mobility: withOrigin, rules: cloud, once: true, at: true, slow: true},
		{name: "Cloudkill", page: "p. 116", shape: sphere(20), mobility: drift(content.AwayFromCaster), rules: cloud, once: true},
		{name: "Incendiary Cloud", page: "pp. 142-143", shape: sphere(20), mobility: drift(content.AwayFromCasterChosen), rules: cloud, once: true},
		{name: "Insect Plague", page: "p. 143", shape: sphere(20), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(ends, any, "", false)}, once: true, terrain: true},
		{name: "Moonbeam", page: "pp. 150-151", shape: cylinder(5, 40), mobility: byCaster(60, content.MagicAction, true), rules: cloud, once: true},
		{name: "Flaming Sphere", page: "p. 132", shape: sphere(5), mobility: byCaster(30, content.BonusAction, false),
			rules: []content.ZoneRuleFact{rule(ends, any, "within5", false), rule(moves, any, "", false)}},
		{name: "Spike Growth", page: "p. 164", shape: sphere(20), rules: []content.ZoneRuleFact{rule(travels, any, "", false)}, terrain: true},
		{name: "Wall of Fire", page: "p. 172", shape: line(60, 1), rules: []content.ZoneRuleFact{rule(first, any, "body", false), rule(ends, any, "body", false), rule(ends, any, "side", false)}},
		{name: "Wall of Thorns", page: "p. 173", shape: line(60, 1), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(ends, any, "", false)}, once: true, extra: 3},
		{name: "Black Tentacles", page: "p. 113", shape: cube(20), rules: []content.ZoneRuleFact{rule(enters, any, "", false), rule(ends, any, "", false)}, once: true, terrain: true},
		{name: "Blade Barrier", page: "p. 113", shape: line(100, 2), rules: []content.ZoneRuleFact{rule(enters, any, "", false), rule(ends, any, "", false)}, once: true},
		{name: "Web", page: "p. 174", shape: cube(20), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(starts, any, "", false)}, terrain: true},
		{name: "Sleet Storm", page: "p. 163", shape: cylinder(20, 40), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(starts, any, "", false)}, terrain: true},
		{name: "Stinking Cloud", page: "p. 165", shape: sphere(20), rules: []content.ZoneRuleFact{rule(starts, any, "", false)}},
		{name: "Conjure Woodland Beings", page: "pp. 118-119", shape: emanation(10), mobility: withOrigin, rules: three(seen, true), once: true},
		{name: "Conjure Celestial", page: "p. 118", shape: cylinder(10, 40), mobility: byCaster(30, content.WithCasterMovement, false), rules: three(seen, true), once: true},
		{name: "Conjure Animals", page: "pp. 117-118", shape: emanation(10), mobility: byCaster(30, content.WithCasterMovement, false), rules: three(seen, true), once: true},
		{name: "Guardian of Faith", page: "p. 137", shape: emanation(10), rules: []content.ZoneRuleFact{rule(first, enemies, "", false), rule(starts, enemies, "", false)}, budget: 60},
		{name: "Grease", page: "p. 137", shape: cube(10), rules: []content.ZoneRuleFact{rule(enters, any, "", false), rule(ends, any, "", false)}, terrain: true},
		{name: "Gust of Wind", page: "p. 138", shape: line(60, 10), rules: []content.ZoneRuleFact{rule(ends, any, "", false)}},
		{name: "Control Water", page: "pp. 119-120", shape: cube(50), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(ends, any, "", false)}},
		{name: "Forbiddance", page: "p. 133", shape: cube(40), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(ends, any, "", false)}},
		{name: "Zone of Truth", page: "p. 175", shape: sphere(15), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(starts, any, "", false)}},
		{name: "Prismatic Wall", page: "p. 155", shape: line(90, 30), rules: []content.ZoneRuleFact{rule(enters, any, "within20", false), rule(starts, any, "within20", false)}, at: true},
		{name: "Symbol", page: "p. 167", shape: sphere(60), rules: []content.ZoneRuleFact{rule(first, any, "", false), rule(ends, any, "", false)}, once: true},
	}
	if len(rows) != 25 {
		t.Fatalf("table Z has %d rows, want 25", len(rows))
	}
	for _, r := range rows {
		t.Run(strings.ReplaceAll(r.name, " ", "_"), func(t *testing.T) {
			t.Parallel()
			designation := content.NoDesignation
			if r.at {
				designation = content.DesignationAtCastingOnly
			}
			facts := pzFacts(content.ZoneFactsFields{Rules: r.rules, Mobility: r.mobility, OncePerTurn: r.once,
				Designation: designation, DamageBudget: r.budget,
				Movement: luMust(content.NewZoneMovementFacts(r.terrain, r.slow, r.extra))})
			zone := pzCreate(t, pzSpell(strings.ToLower(strings.ReplaceAll(r.name, " ", "_")), 1, facts, nil), func(c *combat.CreateZoneCommand) { c.Shape = r.shape })
			if !zone.Facts().Equal(facts) || zone.Shape() != r.shape {
				t.Errorf("%s (SRD 5.2.1 %s): the zone does not keep its facts and shape", r.name, r.page)
			}
			regions := map[string]bool{string(content.DefaultZoneRegion): true}
			for _, ru := range r.rules {
				regions[string(ru.Region())] = true
			}
			var names []string
			for n := range regions {
				names = append(names, n)
			}
			creature := pzCreature("x", true, true, names...)
			turn := pzTurn(1, "caster")
			seenTriggers := map[content.ZoneTrigger]bool{}
			for _, ru := range r.rules {
				if seenTriggers[ru.Trigger()] {
					continue
				}
				seenTriggers[ru.Trigger()] = true
				var ev combat.ZoneTriggerEvent
				switch ru.Trigger() {
				case content.CreatureEnters, content.CreatureFirstEntryOnTurn:
					ev = combat.NewCreatureEntered(creature)
				case content.CreatureEndsTurnInside:
					ev = combat.NewCreatureEndedTurn(creature)
				case content.CreatureStartsTurnInside:
					ev = combat.NewCreatureStartedTurn(creature)
				case content.ZoneMovesIntoSpace:
					ev = combat.NewZoneMovedIntoSpace(creature)
				case content.TravelsWithin:
					ev = luMust(combat.NewCreatureTravelled(creature, 5))
				}
				got, ok := combat.AssessZoneTrigger(zone, turn, ev).(combat.AssessZoneTriggerDue)
				if !ok {
					t.Fatalf("%s: AssessZoneTrigger is not Due", r.name)
				}
				want := 1
				if len(got.Mandatory)+len(got.Optional) != want || len(got.Skipped) != 0 {
					t.Errorf("%s, trigger %v: %d mandatory, %d optional, skipped %v, want exactly one due effect and no skip",
						r.name, ru.Trigger(), len(got.Mandatory), len(got.Optional), got.Skipped)
				}
				if ru.CasterChoice() && len(got.Optional) != 1 || !ru.CasterChoice() && len(got.Mandatory) != 1 {
					t.Errorf("%s, trigger %v: the effect is not %s", r.name, ru.Trigger(), map[bool]string{true: "optional", false: "mandatory"}[ru.CasterChoice()])
				}
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md Limits (plan P10) - CreateZone reads at most 64 rules and at most 256 designated creatures; one more of either is CreateZoneRejected with CreateLimitExceeded alone, and the at-cap command creates the zone.
// Path: per case.
func TestCreateZoneRuleAndDesignatedCaps(t *testing.T) {
	t.Parallel()
	rulesZone := func(n int) content.Spell {
		rules := make([]content.ZoneRuleFact, n)
		for i := range rules {
			rules[i] = pzRuleIn(content.CreatureEndsTurnInside, 0, content.AnyCreature, fmt.Sprintf("r%d", i))
		}
		return pzSpell("many", 1, pzFacts(content.ZoneFactsFields{Rules: rules}), nil)
	}
	designated := func(n int) []combat.CombatantID {
		out := make([]combat.CombatantID, n)
		for i := range out {
			out[i] = combat.MustCombatantID(fmt.Sprintf("d%d", i))
		}
		return out
	}
	create := func(spell content.Spell, ds []combat.CombatantID) combat.CreatePersistentZoneResult {
		return combat.CreateZone(combat.CreateZoneCommand{ID: luMust(combat.NewPersistentZoneID("zone")), Caster: combat.MustCombatantID("caster"),
			Spell: spell, Shape: luMust(content.NewSphereArea(20)), Designated: ds, Cast: pzTurn(1, "caster")})
	}
	for _, c := range []struct {
		name string
		res  combat.CreatePersistentZoneResult
		over bool
	}{
		{"64 rules", create(rulesZone(64), nil), false},
		{"65 rules", create(rulesZone(65), nil), true},
		{"256 designated", create(pzSpirit(), designated(256)), false},
		{"257 designated", create(pzSpirit(), designated(257)), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if !c.over {
				if _, ok := c.res.(combat.CreateZoneCreated); !ok {
					t.Errorf("result = %#v, want CreateZoneCreated", c.res)
				}
				return
			}
			rej, ok := c.res.(combat.CreateZoneRejected)
			if !ok || len(rej.Reasons) != 1 || rej.Reasons[0] != combat.CreateLimitExceeded {
				t.Errorf("result = %#v, want rejected with CreateLimitExceeded alone", c.res)
			}
		})
	}
}
