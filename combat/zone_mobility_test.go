package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// pzMovedByCaster returns mobility MovedByCaster(maxFeet, action, laterOnly).
func pzMovedByCaster(maxFeet int, action content.ZoneActionKind, laterOnly bool) content.MovedByCaster {
	return luMust(content.NewMovedByCaster(maxFeet, action, laterOnly))
}

// pzDrifts returns mobility DriftsAtTurnStart(feet, direction).
func pzDrifts(feet int, dir content.DriftDirectionRule) content.DriftsAtTurnStart {
	return luMust(content.NewDriftsAtTurnStart(feet, dir))
}

// pzMobileZone creates a zone of spell id with mobility m and rules rules.
func pzMobileZone(t *testing.T, id string, m content.ZoneMobility, rules []content.ZoneRuleFact, once bool) combat.PersistentZone {
	t.Helper()
	facts := pzFacts(content.ZoneFactsFields{Rules: rules, Mobility: m, OncePerTurn: once})
	return pzCreate(t, pzSpell(id, 1, facts, nil), nil)
}

// pzMoonbeam is Moonbeam (SRD 5.2.1, pp. 150-151): a cylinder that moves 60 feet
// by a Magic Action on the caster's later turns.
func pzMoonbeam(t *testing.T) combat.PersistentZone {
	t.Helper()
	return pzMobileZone(t, "moonbeam", pzMovedByCaster(60, content.MagicAction, true), []content.ZoneRuleFact{
		pzRule(content.CreatureEnters, 0, content.AnyCreature),
	}, true)
}

// pzMovedAs returns the outcome of moving zone by a caster move at turn.
func pzMovedAs(t *testing.T, zone combat.PersistentZone, turn combat.TurnMarker, feet int, how content.ZoneActionKind) combat.MoveZoneResult {
	t.Helper()
	m := luMust(combat.NewCasterMove(feet, how))
	return combat.MoveZone(zone, m, turn)
}

// Requirement: doc/specs/persistent-spell-zones.md FR-035 and table M (M1 to M4, M13 to M15)
// - a caster move needs a MovedByCaster zone, the right action kind, a distance within
// the maximum, the caster's own turn and, for a later-turn zone, a later turn (SRD 5.2.1,
// "Moonbeam", p. 150; "Conjure Celestial", p. 118).
// Path: per case.
func TestCasterMoveTableM(t *testing.T) {
	t.Parallel()
	moon := pzMoonbeam(t)
	cases := []struct {
		name, requirement, path string
		zone                    combat.PersistentZone
		feet                    int
		how                     content.ZoneActionKind
		turn                    combat.TurnMarker
		want                    combat.MoveRefusal
		moved                   bool
	}{
		{"M1 later turn, 60 feet, magic", "FR-035 Moonbeam later turn", "happy", moon, 60, content.MagicAction, pzTurn(2, "caster"), 0, true},
		{"M2 on the cast turn", "FR-035 NotOnLaterTurn", "unhappy (rule interaction)", moon, 60, content.MagicAction, pzTurn(1, "caster"), combat.RefusalNotOnLaterTurn, false},
		{"M3 61 feet", "FR-035 ExceedsMaxDistance boundary", "unhappy (boundary)", moon, 61, content.MagicAction, pzTurn(2, "caster"), combat.RefusalExceedsMaxDistance, false},
		{"M4 bonus action on magic-action zone", "FR-035 WrongActionKind", "unhappy (invalid input)", moon, 30, content.BonusAction, pzTurn(2, "caster"), combat.RefusalWrongActionKind, false},
		{"M5 flaming sphere, bonus action, cast turn", "FR-035 Flaming Sphere p. 132", "happy", pzFlamingSphere(t), 30, content.BonusAction, pzTurn(1, "caster"), 0, true},
		{"M6 another combatant's turn", "FR-035 NotCastersTurn", "unhappy (state error)", pzFlamingSphere(t), 30, content.BonusAction, pzTurn(2, "other"), combat.RefusalNotCastersTurn, false},
		{"M13 conjure celestial, caster's turn", "FR-035 with caster movement p. 118", "happy", pzConjureCelestial(t), 30, content.WithCasterMovement, pzTurn(2, "caster"), 0, true},
		{"M14 conjure celestial, another's turn", "FR-035 NotCastersTurn", "unhappy (state error)", pzConjureCelestial(t), 30, content.WithCasterMovement, pzTurn(2, "other"), combat.RefusalNotCastersTurn, false},
		{"M15 conjure animals, 31 feet", "FR-035 ExceedsMaxDistance p. 117", "unhappy (boundary)", pzConjureCelestial(t), 31, content.WithCasterMovement, pzTurn(2, "caster"), combat.RefusalExceedsMaxDistance, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := pzMovedAs(t, c.zone, c.turn, c.feet, c.how)
			if c.moved {
				if _, ok := res.(combat.MoveZoneMoved); !ok {
					t.Fatalf("MoveZone = %#v, want MoveZoneMoved [%s]", res, c.requirement)
				}
				return
			}
			if r, ok := res.(combat.MoveZoneRefused); !ok || r.Reason != c.want {
				t.Fatalf("MoveZone = %#v, want refused %v [%s]", res, c.want, c.requirement)
			}
		})
	}
}

// pzFlamingSphere is Flaming Sphere (SRD 5.2.1, p. 132): a bonus-action move of 30
// feet on any turn, and a stop on contact for the turn.
func pzFlamingSphere(t *testing.T) combat.PersistentZone {
	t.Helper()
	return pzMobileZone(t, "flaming_sphere", pzMovedByCaster(30, content.BonusAction, false), []content.ZoneRuleFact{
		pzRule(content.ZoneMovesIntoSpace, 0, content.AnyCreature),
	}, false)
}

// pzConjureCelestial moves 30 feet with the caster's own movement (p. 118).
func pzConjureCelestial(t *testing.T) combat.PersistentZone {
	t.Helper()
	return pzMobileZone(t, "conjure_celestial", pzMovedByCaster(30, content.WithCasterMovement, false), []content.ZoneRuleFact{
		pzRule(content.CreatureEnters, 0, content.AnyCreature),
	}, true)
}

// Requirement: doc/specs/persistent-spell-zones.md FR-033 and table M (M12) - a stationary
// zone refuses every move with NotMovable and no event (SRD 5.2.1, "Spike Growth", p. 164).
// Path: unhappy (state error).
func TestStationaryZoneRefusesEveryMove(t *testing.T) {
	t.Parallel()
	spike := pzMobileZone(t, "spike", content.StationaryZone{}, []content.ZoneRuleFact{pzTravel(0, 5)}, false)
	if r, ok := pzMovedAs(t, spike, pzTurn(2, "caster"), 10, content.BonusAction).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalNotMovable {
		t.Fatalf("caster move on a stationary zone = %#v, want NotMovable [FR-033]", r)
	}
	if r, ok := combat.MoveZone(spike, luMust(combat.NewDriftMove(5)), pzTurn(2, "caster")).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalNotMovable {
		t.Fatalf("drift on a stationary zone = %#v, want NotMovable [FR-033]", r)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-034 and table M (M11) - a zone that
// follows its origin refuses a drift with MovesWithOrigin; a caster move on it is
// NotMovable (SRD 5.2.1, "Emanation", p. 181; "Spirit Guardians", p. 164).
// Path: unhappy (rule interaction).
func TestMovingOriginZoneFollowsItsCaster(t *testing.T) {
	t.Parallel()
	spirit := pzCreate(t, pzSpirit(), nil)
	if r, ok := combat.MoveZone(spirit, luMust(combat.NewDriftMove(5)), pzTurn(2, "caster")).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalMovesWithOrigin {
		t.Fatalf("drift on Spirit Guardians = %#v, want MovesWithOrigin [FR-034]", r)
	}
	if r, ok := pzMovedAs(t, spirit, pzTurn(2, "caster"), 5, content.BonusAction).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalNotMovable {
		t.Fatalf("caster move on Spirit Guardians = %#v, want NotMovable [FR-034]", r)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-035 and table M (M16) - a drift on a
// MovedByCaster zone and a caster move on a DriftsAtTurnStart zone are both NotMovable,
// since each mobility takes only its own kind of move (SRD 5.2.1, "Moonbeam", p. 150;
// "Cloudkill", p. 116).
// Path: unhappy (rule interaction).
func TestMoveKindsDoNotCrossMobilities(t *testing.T) {
	t.Parallel()
	moon := pzMoonbeam(t)
	cloud := pzMobileZone(t, "cloudkill", pzDrifts(10, content.AwayFromCaster), []content.ZoneRuleFact{
		pzRule(content.CreatureEnters, 0, content.AnyCreature),
	}, true)
	if r, ok := combat.MoveZone(moon, luMust(combat.NewDriftMove(10)), pzTurn(2, "caster")).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalNotMovable {
		t.Fatalf("drift on Moonbeam = %#v, want NotMovable [FR-035 M16]", r)
	}
	if r, ok := pzMovedAs(t, cloud, pzTurn(2, "caster"), 10, content.BonusAction).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalNotMovable {
		t.Fatalf("caster move on Cloudkill = %#v, want NotMovable [FR-035 M16]", r)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-036 and table M (M8 to M10) - a
// drifting zone reports its drift on its caster's turn only, and accepts a drift of
// up to its feet (SRD 5.2.1, "Cloudkill", p. 116; "Incendiary Cloud", pp. 142-143).
// Path: per case.
func TestDriftingZoneReportsAndAcceptsDrift(t *testing.T) {
	t.Parallel()
	cloud := pzMobileZone(t, "cloudkill", pzDrifts(10, content.AwayFromCaster), []content.ZoneRuleFact{
		pzRule(content.CreatureEnters, 0, content.AnyCreature),
	}, true)
	cases := []struct {
		name, requirement, path string
		turn                    combat.TurnMarker
		drifts                  bool
	}{
		{"M8 caster's turn", "FR-036 drift on the caster's turn", "happy", pzTurn(2, "caster"), true},
		{"M9 another's turn", "FR-036 no drift on another's turn", "unhappy (boundary)", pzTurn(2, "other"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rep, ok := combat.ZoneStartOfTurn(cloud, c.turn).(combat.StartOfTurnReported)
			if !ok || rep.Drifts != c.drifts {
				t.Fatalf("ZoneStartOfTurn drifts = %v, want %t [%s]", rep, c.drifts, c.requirement)
			}
			if c.drifts && (rep.Drift.Feet() != 10 || rep.Drift.Direction() != content.AwayFromCaster) {
				t.Fatalf("drift = %d feet %v, want 10 away [%s]", rep.Drift.Feet(), rep.Drift.Direction(), c.requirement)
			}
		})
	}
	if r, ok := combat.MoveZone(cloud, luMust(combat.NewDriftMove(11)), pzTurn(2, "caster")).(combat.MoveZoneRefused); !ok || r.Reason != combat.RefusalExceedsMaxDistance {
		t.Fatalf("drift of 11 on a 10-foot drift = %#v, want ExceedsMaxDistance [FR-036 M10]", r)
	}
	if _, ok := combat.MoveZone(cloud, luMust(combat.NewDriftMove(10)), pzTurn(2, "caster")).(combat.MoveZoneMoved); !ok {
		t.Fatalf("drift of 10 on a 10-foot drift not accepted [FR-036]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-037 - an accepted move returns the
// same zone with one ZoneMoved event, and does not touch the ledger or exemptions.
// Path: happy.
func TestAcceptedMoveKeepsLedgerAndEmitsZoneMoved(t *testing.T) {
	t.Parallel()
	zone := pzFlamingSphere(t)
	got, ok := pzMovedAs(t, zone, pzTurn(1, "caster"), 30, content.BonusAction).(combat.MoveZoneMoved)
	if !ok {
		t.Fatalf("move not accepted [FR-037]")
	}
	if !got.Zone.Equal(zone) || got.Event.Kind() != "zone_moved" || got.Event.DistanceFeet() != 30 || got.Event.How() != content.BonusAction {
		t.Fatalf("moved = zone equal %t event %s %d %v, want same zone, zone_moved 30 bonus [FR-037]", got.Zone.Equal(zone), got.Event.Kind(), got.Event.DistanceFeet(), got.Event.How())
	}
	if c := got.Event.Combatants(); len(c) != 1 || c[0] != combat.MustCombatantID("caster") {
		t.Fatalf("ZoneMoved.Combatants() = %v, want the caster [FR-050]", c)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-038 and table M (M7, M18) - a
// creature moved into a stopping zone stops it for that turn; the next turn's marker
// clears the stop; the refusal of a stop comes before the action-kind check (SRD 5.2.1,
// "Flaming Sphere", p. 132).
// Path: per case.
func TestStoppedZoneRefusesUntilNextTurn(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{
		Rules:               []content.ZoneRuleFact{pzRule(content.ZoneMovesIntoSpace, 0, content.AnyCreature)},
		Mobility:            pzMovedByCaster(30, content.BonusAction, false),
		StopsMoverOnContact: true,
	})
	base := pzCreate(t, pzSpell("sphere", 1, facts, nil), nil)
	stopped := pzAssessed(t, combat.AssessZoneTrigger(base, pzTurn(1, "caster"), combat.NewZoneMovedIntoSpace(pzInBody("x"))))
	if !stopped.ZoneStoppedForTurn {
		t.Fatalf("ZoneStoppedForTurn = false, want true [FR-038]")
	}
	cases := []struct {
		name, requirement, path string
		feet                    int
		how                     content.ZoneActionKind
		turn                    combat.TurnMarker
		want                    combat.MoveRefusal
	}{
		{"M7 same turn, five feet", "FR-038 StoppedForTurn", "unhappy (rule interaction)", 5, content.BonusAction, pzTurn(1, "caster"), combat.RefusalStoppedForTurn},
		{"M18 same turn, wrong action kind", "FR-035 StoppedForTurn before WrongActionKind", "unhappy (rule interaction)", 30, content.MagicAction, pzTurn(1, "caster"), combat.RefusalStoppedForTurn},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if r, ok := pzMovedAs(t, stopped.Zone, c.turn, c.feet, c.how).(combat.MoveZoneRefused); !ok || r.Reason != c.want {
				t.Fatalf("MoveZone = %#v, want %v [%s]", r, c.want, c.requirement)
			}
		})
	}
	next := combat.ZoneStartOfTurn(stopped.Zone, pzTurn(2, "caster")).(combat.StartOfTurnReported).Zone
	if _, ok := pzMovedAs(t, next, pzTurn(2, "caster"), 5, content.BonusAction).(combat.MoveZoneMoved); !ok {
		t.Fatalf("move after the next turn's marker not accepted [FR-038 FR-043]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-039 - ZoneMovedIntoSpace applies to the
// creature whatever it does next: it does not count as an entry, so a later
// first entry on the same turn is still due (SRD 5.2.1, "Moonbeam", p. 151).
// Path: happy.
func TestMovedIntoSpaceDoesNotCountAsEntry(t *testing.T) {
	t.Parallel()
	facts := pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{
		pzRule(content.CreatureFirstEntryOnTurn, 0, content.AnyCreature),
		pzRule(content.ZoneMovesIntoSpace, 0, content.AnyCreature),
	}})
	zone := pzCreate(t, pzSpell("z", 1, facts, nil), nil)
	moved := pzAssessed(t, combat.AssessZoneTrigger(zone, pzTurn(1, "caster"), combat.NewZoneMovedIntoSpace(pzInBody("x"))))
	entered := pzAssessed(t, combat.AssessZoneTrigger(moved.Zone, pzTurn(1, "caster"), combat.NewCreatureEntered(pzInBody("x"))))
	if len(entered.Mandatory) != 1 {
		t.Fatalf("entry after a move on one turn dues = %d, want 1 [FR-039]", len(entered.Mandatory))
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-036 and FR-055 - a move or start of
// turn on a turn earlier than the cast is a StaleTurn result, not an error.
// Path: unhappy (invalid input).
func TestMoveAndStartOfTurnRefuseStaleTurns(t *testing.T) {
	t.Parallel()
	zone := pzFlamingSphere(t)
	if _, ok := pzMovedAs(t, zone, pzTurn(1, "caster"), 5, content.BonusAction).(combat.MoveZoneMoved); !ok {
		t.Fatalf("a move on the cast turn was refused [FR-055]")
	}
	cast2 := pzCreate(t, pzSpell("s2", 1, pzFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{pzRule(content.CreatureEnters, 0, content.AnyCreature)},
		Mobility: pzDrifts(10, content.AwayFromCaster)}), nil), func(c *combat.CreateZoneCommand) { c.Cast = pzTurn(3, "caster") })
	if _, ok := combat.MoveZone(cast2, luMust(combat.NewDriftMove(5)), pzTurn(2, "caster")).(combat.MoveZoneStaleTurn); !ok {
		t.Fatalf("MoveZone before the cast turn not StaleTurn [FR-055]")
	}
	if _, ok := combat.ZoneStartOfTurn(cast2, pzTurn(2, "caster")).(combat.StartOfTurnStaleTurn); !ok {
		t.Fatalf("ZoneStartOfTurn before the cast turn not StaleTurn [FR-055]")
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-035 - a distance of 0 or below for
// a caster move, or a drift, is an *ZoneInvalidError (SRD 5.2.1, "Moonbeam", p. 150).
// Path: unhappy (boundary).
func TestMoveDistancesMustBePositive(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewCasterMove(0, content.MagicAction); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewCasterMove(0) error = %v, want ErrInvalidInput [FR-035]", err)
	}
	if _, err := combat.NewDriftMove(-5); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewDriftMove(-5) error = %v, want ErrInvalidInput [FR-035]", err)
	}
}
