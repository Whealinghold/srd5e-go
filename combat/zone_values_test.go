package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/persistent-spell-zones.md FR-001 - PersistentZoneID is a
// non-blank string; a blank value gives an *ZoneInvalidError (architecture.md section 3).
// Path: per case.
func TestNewPersistentZoneIDRefusesBlank(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		in                      string
		wantErr                 bool
	}{
		{"empty", "FR-001 blank value refused", "unhappy (invalid input)", "", true},
		{"spaces only", "FR-001 blank value refused", "unhappy (invalid input)", "  \t ", true},
		{"one letter", "FR-001 non-blank accepted", "happy", "z", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NewPersistentZoneID(c.in)
			if c.wantErr {
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Fatalf("NewPersistentZoneID(%q) error = %v, want ErrInvalidInput [%s]", c.in, err, c.requirement)
				}
				var inv *combat.ZoneInvalidError
				if !errors.As(err, &inv) {
					t.Fatalf("NewPersistentZoneID(%q) error %T is not *ZoneInvalidError [%s]", c.in, err, c.requirement)
				}
				return
			}
			if err != nil || got.String() != c.in {
				t.Fatalf("NewPersistentZoneID(%q) = %q, %v [%s]", c.in, got, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-001 - the zero PersistentZoneID is
// invalid, the same as a blank constructed id.
// Path: unhappy (invalid input).
func TestZeroPersistentZoneIDIsBlank(t *testing.T) {
	t.Parallel()
	var zero combat.PersistentZoneID
	if _, err := combat.NewPersistentZoneID(zero.String()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewPersistentZoneID(zero) error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-026 - a TurnMarker is equal when
// round and active combatant are equal (SRD 5.2.1, "Combat Step by Step", p. 13).
// Path: unhappy (boundary).
func TestTurnMarkerEqualityNeedsRoundAndActive(t *testing.T) {
	t.Parallel()
	base := pzTurn(2, "x")
	cases := []struct {
		name, requirement, path string
		other                   combat.TurnMarker
		same                    bool
	}{
		{"same round and combatant", "FR-026 equal turns", "happy", pzTurn(2, "x"), true},
		{"other combatant, same round", "FR-026 a different combatant's turn", "unhappy (boundary)", pzTurn(2, "y"), false},
		{"same combatant, later round", "FR-026 the same combatant's later turn", "unhappy (boundary)", pzTurn(3, "x"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := base == c.other; got != c.same {
				t.Fatalf("turn %v == %v = %t, want %t [%s]", base, c.other, got, c.same, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-032 - NewTurnMarker with a round
// below 1 gives an *ZoneInvalidError; round 1 is the first valid round.
// Path: per case.
func TestNewTurnMarkerRoundBoundary(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		round                   int
		wantErr                 bool
	}{
		{"round 0", "FR-032 round below 1 refused (L12)", "unhappy (boundary)", 0, true},
		{"round -1", "FR-032 negative round refused", "unhappy (boundary)", -1, true},
		{"round 1", "FR-032 first round accepted", "happy", 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m, err := combat.NewTurnMarker(c.round, combat.MustCombatantID("x"))
			if c.wantErr {
				if !errors.Is(err, srderr.ErrInvalidInput) {
					t.Fatalf("NewTurnMarker(%d) error = %v, want ErrInvalidInput [%s]", c.round, err, c.requirement)
				}
				return
			}
			if err != nil || m.Round() != c.round || m.Combatants()[0] != combat.MustCombatantID("x") {
				t.Fatalf("NewTurnMarker(%d) = %v, %v [%s]", c.round, m, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-031 - the ledger holds one
// marker per creature (the latest); a creature recorded twice counts once.
// Path: unhappy (boundary).
func TestLedgerSizeNeverExceedsCreaturesSeen(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), nil)
	x := pzInBody("x")
	for round := 1; round <= 5; round++ {
		res := combat.AssessZoneTrigger(zone, pzTurn(round, "caster"), combat.NewCreatureEndedTurn(x))
		zone = pzAssessed(t, res).Zone
	}
	if got := zone.Ledger().Size(); got != 1 {
		t.Fatalf("ledger Size after 5 turns for one creature = %d, want 1 [FR-031]", got)
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-001 and FR-035 - a creature
// facts value copies its regions and reports them unchanged.
// Path: happy.
func TestZoneCreatureFactsCopyTheirRegions(t *testing.T) {
	t.Parallel()
	regions := []content.ZoneRegionID{luMust(content.NewZoneRegionID("body")), luMust(content.NewZoneRegionID("side"))}
	f := combat.NewZoneCreatureFacts(combat.MustCombatantID("x"), regions, true, false)
	regions[0] = luMust(content.NewZoneRegionID("other"))
	got := f.Regions()
	if len(got) != 2 || got[0].String() != "body" || got[1].String() != "side" {
		t.Fatalf("Regions() = %v, want [body side] after the input changed [FR-052]", got)
	}
	got[0] = luMust(content.NewZoneRegionID("other"))
	if f.Regions()[0].String() != "body" {
		t.Fatalf("Regions() returned an alias of the stored regions [FR-052]")
	}
	if !f.IsEnemyOfCaster() || f.SeenByCaster() || f.Combatants()[0] != combat.MustCombatantID("x") {
		t.Fatalf("facts accessors = enemy %t seen %t creature %v, want true false x", f.IsEnemyOfCaster(), f.SeenByCaster(), f.Combatants()[0])
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-052 - returned exemption lists
// are copies, and the zone's exemptions cannot be changed through them.
// Path: unhappy (immutability).
func TestExemptionsReturnACopy(t *testing.T) {
	t.Parallel()
	zone := pzCreate(t, pzSpirit(), func(c *combat.CreateZoneCommand) {
		c.Designated = []combat.CombatantID{combat.MustCombatantID("ally")}
	})
	ids := zone.Exemptions().Creatures()
	ids[0] = combat.MustCombatantID("enemy")
	if !zone.Exemptions().Contains(combat.MustCombatantID("ally")) || zone.Exemptions().Contains(combat.MustCombatantID("enemy")) {
		t.Fatalf("Creatures() aliased the stored exemptions [FR-052]")
	}
}

// Requirement: none - pins the snake_case diagnostic names of the zone enums and the
// "invalid" name of an out-of-range value (not display text, architecture.md section 6);
// each row names the spec id of its enum. The result-classification id is not cited here.
// Path: per case.
func TestZoneEnumNames(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got, want               string
	}{
		{"end reason", "FR-040 end reasons", "happy", combat.ZoneEndBudgetSpent.String(), "budget_spent"},
		{"end reason first value", "FR-040 the first end reason is named", "unhappy (boundary)", combat.ZoneEndConcentrationEnded.String(), "concentration_ended"},
		{"end reason last value", "FR-040 the last end reason is named", "unhappy (boundary)", combat.ZoneEndCasterEnded.String(), "caster_ended"},
		{"skip reason", "FR-014 skip reasons", "happy", combat.SkipNotFirstEntryThisTurn.String(), "not_first_entry_this_turn"},
		{"skip reason first value", "FR-014 the first skip reason is named", "unhappy (boundary)", combat.SkipDesignatedExempt.String(), "designated_exempt"},
		{"skip reason last value", "FR-014 the last skip reason is named", "unhappy (boundary)", combat.SkipLessThanOneUnit.String(), "less_than_one_unit"},
		{"move refusal first value", "FR-035 the first refusal is named", "unhappy (boundary)", combat.RefusalZoneHasEnded.String(), "zone_has_ended"},
		{"move refusal", "FR-035 refusal order", "happy", combat.RefusalNotOnLaterTurn.String(), "not_on_later_turn"},
		{"end reason zero", "FR-040 zero value", "unhappy (invalid input)", combat.ZoneEndReason(0).String(), "invalid"},
		{"skip reason out of range", "FR-014 skip reasons", "unhappy (invalid input)", combat.ZoneSkipReason(99).String(), "invalid"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Fatalf("name = %q, want %q [%s]", c.got, c.want, c.requirement)
			}
		})
	}
}
