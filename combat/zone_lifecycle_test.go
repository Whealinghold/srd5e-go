package combat_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// pzGuardian is Guardian of Faith (SRD 5.2.1, p. 137): an Emanation of 10 feet on
// an object, enemies only, vanishing after 60 damage.
func pzGuardian(t *testing.T) combat.PersistentZone {
	t.Helper()
	facts := pzFacts(content.ZoneFactsFields{
		Rules:        []content.ZoneRuleFact{pzRuleIn(content.CreatureFirstEntryOnTurn, 0, content.EnemiesOfCaster, "body")},
		DamageBudget: 60,
	})
	return pzCreate(t, pzSpell("guardian_of_faith", 1, facts, luMust(content.NewSpellDurationTimed(8, content.DurationUnitHours, content.LimitUpTo))), nil)
}

// pzTenRounds is a rounds zone of 10 rounds cast in round 1.
func pzTenRounds(t *testing.T) combat.PersistentZone {
	t.Helper()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}})
	return pzCreate(t, pzSpell("ten", 1, facts, luMust(content.NewSpellDurationTimed(10, content.DurationUnitRounds, content.LimitConcentrationUpTo))), nil)
}

// Requirement: doc/specs/persistent-spell-zones.md FR-040 - EndZone on an active zone gives
// EndZoneEnded with the reason and one ZoneEnded event, keeping the ledger and facts; the
// reasons are ConcentrationEnded, DurationElapsed, dispelled, dispersed, BudgetSpent and
// CasterEnded (SRD 5.2.1, "Cloudkill", p. 116; "Concentration", p. 179; "Web", p. 174).
// Path: per case.
func TestEndZoneRecordsEachReason(t *testing.T) {
	t.Parallel()
	reasons := []combat.ZoneEndReason{combat.ZoneEndConcentrationEnded, combat.ZoneEndDurationElapsed,
		combat.ZoneEndDispelled, combat.ZoneEndDispersed, combat.ZoneEndBudgetSpent, combat.ZoneEndCasterEnded}
	for _, r := range reasons {
		t.Run(r.String(), func(t *testing.T) {
			t.Parallel()
			created := pzCreate(t, pzSpirit(), nil)
			res, ok := combat.EndZone(created, r).(combat.EndZoneEnded)
			if !ok {
				t.Fatalf("EndZone = %T, want EndZoneEnded [FR-040]", res)
			}
			if !res.Zone.Ended() || res.Zone.EndReason() != r || res.Event.Reason() != r || res.Event.Kind() != "zone_ended" {
				t.Fatalf("ended zone reason %v event %v kind %q, want %v [FR-040]", res.Zone.EndReason(), res.Event.Reason(), res.Event.Kind(), r)
			}
			if !res.Zone.Ledger().Equal(created.Ledger()) || !res.Zone.Facts().Equal(created.Facts()) {
				t.Fatalf("EndZone changed the ledger or facts [FR-040]")
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-040 - EndZone on an ended zone gives
// EndZoneAlreadyEnded, unchanged, with no event (class refused).
// Path: unhappy (state error).
func TestEndZoneTwiceIsRefused(t *testing.T) {
	t.Parallel()
	ended := combat.EndZone(pzCreate(t, pzSpirit(), nil), combat.ZoneEndDispelled).(combat.EndZoneEnded).Zone
	if _, ok := combat.EndZone(ended, combat.ZoneEndCasterEnded).(combat.EndZoneAlreadyEnded); !ok {
		t.Fatalf("EndZone on an ended zone not EndZoneAlreadyEnded [FR-040]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-041 - a rounds zone ends with
// DurationElapsed when the round reaches the cast round plus its rounds: round 9 and 10
// leave a 10-round zone cast in round 1 unchanged; round 11 ends it (Q-8). A round below
// the cast round is an *ZoneInvalidError (SRD 5.2.1, "Duration", p. 106).
// Path: per case.
func TestRoundsZoneEndsAtItsLastRound(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		round                   int
		ended                   bool
	}{
		{"round 9", "FR-041 before the last round", "unhappy (boundary)", 9, false},
		{"round 10", "FR-041 round before the fencepost", "unhappy (boundary)", 10, false},
		{"round 11", "FR-041 cast round plus rounds ends it", "unhappy (boundary)", 11, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := combat.ZoneRoundStarted(pzTenRounds(t), c.round)
			if err != nil {
				t.Fatalf("ZoneRoundStarted(%d) error = %v [%s]", c.round, err, c.requirement)
			}
			if res.Zone.Ended() != c.ended || (len(res.Events) == 1) != c.ended {
				t.Fatalf("ZoneRoundStarted(%d) ended %t events %d, want ended %t [%s]", c.round, res.Zone.Ended(), len(res.Events), c.ended, c.requirement)
			}
			if c.ended && res.Zone.EndReason() != combat.ZoneEndDurationElapsed {
				t.Fatalf("end reason = %v, want DurationElapsed [FR-041]", res.Zone.EndReason())
			}
		})
	}
	if _, err := combat.ZoneRoundStarted(pzCreate(t, pzSpell("c", 1, pzSpiritFacts(), nil), func(c *combat.CreateZoneCommand) {
		c.Cast = pzTurn(2, "caster")
	}), 1); err == nil {
		t.Fatalf("ZoneRoundStarted before the cast round accepted [FR-041]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-041 - a rounds zone with a count near
// the largest int, cast in round 2, is not ended by the next rounds: the cast round plus the
// count would wrap around int (SRD 5.2.1, "Duration", p. 106).
// Path: unhappy (boundary).
func TestHugeRoundsZoneIsNotEndedEarly(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}})
	dur := luMust(content.NewSpellDurationTimed(math.MaxInt, content.DurationUnitRounds, content.LimitConcentrationUpTo))
	zone := pzCreate(t, pzSpell("huge", 1, facts, dur), func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(2, "caster") })
	for _, round := range []int{3, 4, math.MaxInt - 1} {
		res, err := combat.ZoneRoundStarted(zone, round)
		if err != nil || res.Zone.Ended() || len(res.Events) != 0 {
			t.Fatalf("ZoneRoundStarted(%d) ended a zone with %d rounds cast in round 2 [FR-041]", round, math.MaxInt)
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-041 - a host-timed or until-dispelled
// zone is never ended by a round.
// Path: unhappy (rule interaction).
func TestHostTimedAndUntilDispelledZonesIgnoreRounds(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)}})
	days := pzCreate(t, pzSpell("days", 1, facts, luMust(content.NewSpellDurationTimed(7, content.DurationUnitDays, content.LimitUpTo))), nil)
	until := pzCreate(t, pzSpell("until", 1, facts, nil), nil)
	for _, z := range []combat.PersistentZone{days, until} {
		res, err := combat.ZoneRoundStarted(z, 100000)
		if err != nil || res.Zone.Ended() || len(res.Events) != 0 {
			t.Fatalf("round 100000 ended a host-timed or until-dispelled zone [FR-041]")
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-042 and table M-adjacent boundary
// rows - a budget zone ends with BudgetSpent when the total reaches its budget of 60
// (59 does not, 60 does, 61 does); a zone with no budget refuses (RecordDamageNoBudget);
// a negative amount is an *ZoneInvalidError (SRD 5.2.1, "Guardian of Faith", p. 137).
// Path: per case.
func TestDamageBudgetBoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		amount                  int
		ended                   bool
	}{
		{"59 of 60", "FR-042 under the budget", "unhappy (boundary)", 59, false},
		{"exactly 60", "FR-042 the budget spent", "unhappy (boundary)", 60, true},
		{"61 in one step", "FR-042 over the budget", "unhappy (boundary)", 61, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := combat.RecordZoneDamage(pzGuardian(t), c.amount)
			if err != nil {
				t.Fatalf("RecordZoneDamage(%d) error = %v [%s]", c.amount, err, c.requirement)
			}
			rec, ok := res.(combat.RecordDamageRecorded)
			if !ok || rec.Zone.Ended() != c.ended {
				t.Fatalf("RecordZoneDamage(%d) = %#v, want ended %t [%s]", c.amount, res, c.ended, c.requirement)
			}
			if c.ended && (len(rec.Events) != 1 || rec.Zone.EndReason() != combat.ZoneEndBudgetSpent) {
				t.Fatalf("budget end events %d reason %v, want one BudgetSpent [FR-042]", len(rec.Events), rec.Zone.EndReason())
			}
		})
	}
	if _, ok := mustRecord(t, pzCreate(t, pzSpirit(), nil), 5).(combat.RecordDamageNoBudget); !ok {
		t.Fatalf("damage on a zone with no budget not RecordDamageNoBudget [FR-042]")
	}
	if _, err := combat.RecordZoneDamage(pzGuardian(t), -1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("RecordZoneDamage(-1) error = %v, want ErrInvalidInput [FR-042]", err)
	}
}

// mustRecord records amount on zone and fails the test on an error.
func mustRecord(t *testing.T, zone combat.PersistentZone, amount int) combat.RecordDamageResult {
	t.Helper()
	res, err := combat.RecordZoneDamage(zone, amount)
	if err != nil {
		t.Fatalf("RecordZoneDamage(%d) error = %v", amount, err)
	}
	return res
}

// Requirement: doc/specs/persistent-spell-zones.md FR-043 - ZoneStartOfTurn ends nothing;
// on another's turn it changes nothing (SRD 5.2.1, "Cloudkill", p. 116).
// Path: happy.
func TestStartOfTurnEndsNothing(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	rep, ok := combat.ZoneStartOfTurn(zone, pzTurn(2, "other")).(combat.StartOfTurnReported)
	if !ok || rep.Zone.Ended() || rep.Drifts || !rep.Zone.Equal(zone) {
		t.Fatalf("ZoneStartOfTurn on another's turn = %#v, want the zone unchanged with no drift [FR-043]", rep)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-022 - an operation on an ended zone
// leaves it unchanged with no events, by family: AssessZoneTrigger and ApplyOptionalZoneEffect
// AssessZoneTriggerEnded; MoveZone MoveZoneRefused(ZoneHasEnded); RecordZoneCastApplication
// RecordCastApplicationZoneEnded; RecordZoneDamage RecordDamageZoneEnded (all refused);
// ZoneStartOfTurn and ZoneRoundStarted report it unchanged with nothing due.
// Path: unhappy (state error).
func TestEveryFamilyRefusesAnEndedZone(t *testing.T) {
	t.Parallel()
	ended := combat.EndZone(pzCreate(t, pzSpirit(), nil), combat.ZoneEndDispelled).(combat.EndZoneEnded).Zone
	x := pzInBody("x")
	if _, ok := combat.AssessZoneTrigger(ended, pzTurn(2, "x"), combat.NewCreatureEndedTurn(x)).(combat.AssessZoneTriggerEnded); !ok {
		t.Fatalf("AssessZoneTrigger on an ended zone not AssessZoneTriggerEnded [FR-022]")
	}
	if _, ok := combat.ApplyOptionalZoneEffect(ended, pzTurn(2, "x"), combat.ZoneEffectDue{}).(combat.AssessZoneTriggerEnded); !ok {
		t.Fatalf("ApplyOptionalZoneEffect on an ended zone not AssessZoneTriggerEnded [FR-022]")
	}
	if r, ok := combat.MoveZone(ended, luMust(combat.NewDriftMove(5)), pzTurn(2, "x")).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalZoneHasEnded {
		t.Fatalf("MoveZone on an ended zone = %#v, want ZoneHasEnded [FR-022]", r)
	}
	if _, ok := combat.RecordZoneCastApplication(ended, x.Combatants()[0], pzTurn(2, "x")).(combat.RecordCastApplicationZoneEnded); !ok {
		t.Fatalf("RecordZoneCastApplication on an ended zone not refused [FR-022]")
	}
	if _, ok := mustRecord(t, ended, 1).(combat.RecordDamageZoneEnded); !ok {
		t.Fatalf("RecordZoneDamage on an ended zone not refused [FR-022]")
	}
	rep := combat.ZoneStartOfTurn(ended, pzTurn(2, "caster")).(combat.StartOfTurnReported)
	if rep.Drifts || !rep.Zone.Equal(ended) {
		t.Fatalf("ZoneStartOfTurn on an ended zone = %#v, want unchanged, no drift [FR-022]", rep)
	}
	round, err := combat.ZoneRoundStarted(ended, 50)
	if err != nil || len(round.Events) != 0 || !round.Zone.Equal(ended) {
		t.Fatalf("ZoneRoundStarted on an ended zone = %#v, %v, want unchanged [FR-022]", round, err)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-055 and FR-032 - the operations taking
// a turn refuse a turn earlier than the cast turn with their StaleTurn result, and
// ApplyOptionalZoneEffect does the same for the creature's ledger (table of families).
// Path: unhappy (invalid input).
func TestApplyOptionalRefusesStaleTurn(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		luMust(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEndsTurnInside, Scope: content.AnyCreature, CasterChoice: true})),
	}})
	zone := pzCreate(t, pzSpell("w", 1, facts, nil), func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(3, "caster") })
	if _, ok := combat.ApplyOptionalZoneEffect(zone, pzTurn(2, "x"), combat.ZoneEffectDue{}).(combat.AssessZoneTriggerStaleTurn); !ok {
		t.Fatalf("ApplyOptionalZoneEffect before the cast turn not StaleTurn [FR-055]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-007 - designations are fixed for the
// zone's life: no operation changes them.
// Path: happy.
func TestDesignationsAreFixedForTheZoneLife(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), func(c *combat.CreateZoneCommand) {
		c.Designated = []combat.CombatantID{combat.MustCombatantID("ally")}
	})
	before := zone.Exemptions().Creatures()
	ended := combat.EndZone(zone, combat.ZoneEndCasterEnded).(combat.EndZoneEnded).Zone
	if !reflect.DeepEqual(ended.Exemptions().Creatures(), before) {
		t.Fatalf("designations changed across EndZone [FR-007]")
	}
}
