package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func meTeleport(tb testing.TB, f combat.TeleportFields) combat.TeleportCombatantResult {
	tb.Helper()
	if f.Teleporter == "" {
		f.Teleporter = gid("mage")
	}
	cmd, err := combat.NewTeleportCommand(f)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := (combat.Teleportation{}).Resolve(cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return res
}

// Requirement: per case
// Path: per case.
func TestTeleportDestinationAndSight(t *testing.T) {
	t.Parallel()
	const req = meSpec + `table 4.9a; SRD 5.2.1, "Teleportation", p. 190: `
	cases := []struct {
		name, requirement, path string
		dest                    combat.DestinationSpace
		sight                   combat.DestinationSight
		seen                    bool
		want                    string
	}{
		{"clear and seen", req + "FR-056, FR-060", "happy", combat.Unoccupied, combat.DestinationSightRequired, true, "at"},
		{"clear, sight not required, unseen", req + "FR-060", "unhappy (boundary)", combat.Unoccupied, combat.DestinationSightNotRequired, false, "at"},
		{"clear, sight required, unseen", req + "FR-059", "unhappy (state error)", combat.Unoccupied, combat.DestinationSightRequired, false, "not seen"},
		{"occupied by a creature", req + "FR-057", "unhappy (rule interaction)", combat.OccupiedByCreature, combat.DestinationSightNotRequired, false, "near"},
		{"blocked by a solid obstacle", req + "FR-058", "unhappy (rule interaction)", combat.BlockedBySolidObstacle, combat.DestinationSightRequired, true, "near"},
		{"occupied, sight required, unseen", req + "FR-061", "unhappy (rule interaction)", combat.OccupiedByCreature, combat.DestinationSightRequired, false, "not seen"},
		{"blocked, sight required, unseen", req + "FR-061", "unhappy (rule interaction)", combat.BlockedBySolidObstacle, combat.DestinationSightRequired, false, "not seen"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := meTeleport(t, combat.TeleportFields{Destination: c.dest, Sight: c.sight, DestinationSeen: c.seen})
			var got string
			switch res.(type) {
			case combat.TeleportArrivedAtDestination:
				got = "at"
			case combat.TeleportArrivedNearDestination:
				got = "near"
			case combat.TeleportDestinationNotSeen:
				got = "not seen"
			default:
				got = "other"
			}
			if got != c.want {
				t.Errorf("Resolve = %T, want %s [%s; %s]", res, c.want, c.requirement, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-062, FR-063 - a teleportation costs no movement unless the effect states a cost, which is reported as MovementCostFeet; a negative cost is an ErrInvalidInput error from the constructor.
// Path: unhappy (boundary).
func TestTeleportMovementCost(t *testing.T) {
	t.Parallel()
	at := func(cost int) combat.TeleportArrivedAtDestination {
		return meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, MovementCostFeet: cost}).(combat.TeleportArrivedAtDestination)
	}
	if at(0).MovementCostFeet() != 0 || at(30).MovementCostFeet() != 30 {
		t.Error("the cost must be reported as stated, 0 when none")
	}
	near := meTeleport(t, combat.TeleportFields{Destination: combat.BlockedBySolidObstacle, Sight: combat.DestinationSightNotRequired, MovementCostFeet: 5}).(combat.TeleportArrivedNearDestination)
	if near.MovementCostFeet() != 5 {
		t.Errorf("near cost = %d", near.MovementCostFeet())
	}
	_, err := combat.NewTeleportCommand(combat.TeleportFields{Teleporter: gid("mage"), Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, MovementCostFeet: -1})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative cost error = %v", err)
	}
}

// Requirement: per case
// Path: per case.
func TestTeleportTouchedCreatures(t *testing.T) {
	t.Parallel()
	touched := []combat.CombatantID{gid("b"), gid("a")}
	cases := []struct {
		name, requirement, path string
		policy                  combat.TouchedCreaturePolicy
		touched                 []combat.CombatantID
		want                    []combat.CombatantID
	}{
		{"the default leaves a touched creature behind", meSpec + `FR-066; SRD 5.2.1, "Teleportation", p. 190`, "unhappy (rule interaction)", combat.StayBehind, touched, nil},
		{"the zero policy is StayBehind", meSpec + "FR-066", "unhappy (boundary)", 0, touched, nil},
		{"accompany lists every touched creature in the order given", meSpec + "FR-067", "unhappy (rule interaction)", combat.Accompany, touched, touched},
		{"accompany with nobody touched", meSpec + "FR-067", "unhappy (empty and minimal)", combat.Accompany, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, Policy: c.policy, Touched: c.touched}).(combat.TeleportArrivedAtDestination)
			if got := res.TeleportingCreatures(); !slices.Equal(got, c.want) {
				t.Errorf("TeleportingCreatures = %v, want %v [%s; %s]", got, c.want, c.requirement, c.path)
			}
			if slices.Contains(res.TeleportingCreatures(), gid("mage")) {
				t.Error("the teleporter must never be in TeleportingCreatures")
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestTeleportInvalidTouchedSet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		touched []combat.CombatantID
		want    []combat.CombatantID
	}{
		{"holds the teleporter", []combat.CombatantID{gid("a"), gid("mage")}, []combat.CombatantID{gid("mage")}},
		{"repeats an id", []combat.CombatantID{gid("a"), gid("b"), gid("a")}, []combat.CombatantID{gid("a")}},
		{"lists each offender once, in order", []combat.CombatantID{gid("b"), gid("mage"), gid("b"), gid("b"), gid("mage")}, []combat.CombatantID{gid("mage"), gid("b")}},
	}
	for _, c := range cases {
		for _, policy := range []combat.TouchedCreaturePolicy{combat.StayBehind, combat.Accompany} {
			t.Run(c.name+"/"+policy.String(), func(t *testing.T) {
				t.Parallel()
				// Requirement: FR-068, FR-144; combat-foundations.md FR-031; the check also holds when the touched set would stay behind, and comes before the sight check.
				res := meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightRequired, DestinationSeen: false, Policy: policy, Touched: c.touched})
				bad, ok := res.(combat.TeleportInvalidTouchedSet)
				if !ok || !slices.Equal(bad.Offending(), c.want) {
					t.Errorf("Resolve = %#v, want TeleportInvalidTouchedSet %v", res, c.want)
				}
			})
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-069 and table 4.15a - an arrival emits one CreatureTeleported carrying the teleporter, the TeleportingCreatures in order and whether the arrival was at or near the destination; Combatants is the teleporter then the accompanying creatures; a rejection emits none.
// Path: happy.
func TestTeleportEvent(t *testing.T) {
	t.Parallel()
	touched := []combat.CombatantID{gid("b"), gid("a")}
	for dest, arrival := range map[combat.DestinationSpace]combat.TeleportArrival{combat.Unoccupied: combat.ArrivalAtDestination, combat.OccupiedByCreature: combat.ArrivalNearDestination} {
		res := meTeleport(t, combat.TeleportFields{Destination: dest, Sight: combat.DestinationSightNotRequired, Policy: combat.Accompany, Touched: touched})
		var evs []combat.MagicalEffectEvent
		switch r := res.(type) {
		case combat.TeleportArrivedAtDestination:
			evs = r.Events()
		case combat.TeleportArrivedNearDestination:
			evs = r.Events()
		}
		if len(evs) != 1 {
			t.Fatalf("%v: events = %v", dest, evs)
		}
		ev := evs[0].(combat.CreatureTeleported)
		combattest.CheckEvent(t, ev, "creature_teleported", gid("mage"), gid("b"), gid("a"))
		if !slices.Equal(ev.Accompanying(), touched) || ev.Arrival() != arrival {
			t.Errorf("%v: event = %v %v", dest, ev.Accompanying(), ev.Arrival())
		}
	}
	solo := meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired}).(combat.TeleportArrivedAtDestination).Events()[0]
	combattest.CheckEvent(t, solo, "creature_teleported", gid("mage"))
	for _, res := range []combat.TeleportCombatantResult{
		meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightRequired}),
		meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, Touched: []combat.CombatantID{gid("mage")}}),
	} {
		switch res.(type) {
		case combat.TeleportArrivedAtDestination, combat.TeleportArrivedNearDestination:
			t.Errorf("a rejection %T must emit no event", res)
		}
	}
	for a, want := range map[combat.TeleportArrival]string{combat.ArrivalAtDestination: "at_destination", combat.ArrivalNearDestination: "near_destination", 0: "unknown"} {
		if a.String() != want {
			t.Errorf("%d.String() = %q", int(a), a.String())
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-070 - Resolve does not alter the touched set the command was built from, returns defensively copied lists, and gives equal results for equal inputs.
// Path: unhappy (immutability).
func TestTeleportImmutability(t *testing.T) {
	t.Parallel()
	touched := []combat.CombatantID{gid("b"), gid("a")}
	fields := combat.TeleportFields{Teleporter: gid("mage"), Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, Policy: combat.Accompany, Touched: touched}
	cmd := meMust(combat.NewTeleportCommand(fields))
	touched[0] = gid("zzz") // the caller changes its slice afterwards
	res, err := (combat.Teleportation{}).Resolve(cmd)
	if err != nil {
		t.Fatal(err)
	}
	arr := res.(combat.TeleportArrivedAtDestination)
	got := arr.TeleportingCreatures()
	if !slices.Equal(got, []combat.CombatantID{gid("b"), gid("a")}) {
		t.Fatalf("TeleportingCreatures = %v", got)
	}
	got[0] = gid("zzz")
	if arr.TeleportingCreatures()[0] != gid("b") {
		t.Error("the returned list is not a copy")
	}
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{arr.Events()[0]} })
	again, _ := (combat.Teleportation{}).Resolve(cmd)
	if !slices.Equal(again.(combat.TeleportArrivedAtDestination).TeleportingCreatures(), arr.TeleportingCreatures()) {
		t.Error("equal inputs must give equal results")
	}
	bad := meTeleport(t, combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, Touched: []combat.CombatantID{gid("a"), gid("a")}}).(combat.TeleportInvalidTouchedSet)
	bad.Offending()[0] = gid("zzz")
	if bad.Offending()[0] != gid("a") {
		t.Error("Offending is not a copy")
	}
}

// Requirement: doc/specs/magical-effects.md FR-063, FR-144, section 5 - NewTeleportCommand rejects a zero teleporter, an undefined destination, sight or policy and a negative cost with ErrInvalidInput; Resolve rejects a command not built by it.
// Path: unhappy (invalid input).
func TestNewTeleportCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	ok := combat.TeleportFields{Teleporter: gid("mage"), Destination: combat.Unoccupied, Sight: combat.DestinationSightRequired}
	for name, mod := range map[string]func(*combat.TeleportFields){
		"zero teleporter":  func(f *combat.TeleportFields) { f.Teleporter = "" },
		"destination 0":    func(f *combat.TeleportFields) { f.Destination = 0 },
		"destination past": func(f *combat.TeleportFields) { f.Destination = combat.BlockedBySolidObstacle + 1 },
		"sight 0":          func(f *combat.TeleportFields) { f.Sight = 0 },
		"sight past":       func(f *combat.TeleportFields) { f.Sight = combat.DestinationSightNotRequired + 1 },
		"policy negative":  func(f *combat.TeleportFields) { f.Policy = -1 },
		"policy past":      func(f *combat.TeleportFields) { f.Policy = combat.Accompany + 1 },
		"cost below zero":  func(f *combat.TeleportFields) { f.MovementCostFeet = -1 },
	} {
		f := ok
		mod(&f)
		if _, err := combat.NewTeleportCommand(f); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
	if res, err := (combat.Teleportation{}).Resolve(combat.TeleportCommand{}); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero command: %v, %v", res, err)
	}
	if _, err := combat.NewTeleportCommand(ok); err != nil {
		t.Errorf("the baseline fields must be valid: %v", err)
	}
}

// Requirement: none - pins the String values of the teleportation enums.
// Path: happy.
func TestTeleportEnumsString(t *testing.T) {
	t.Parallel()
	got := []string{combat.Unoccupied.String(), combat.OccupiedByCreature.String(), combat.BlockedBySolidObstacle.String(), combat.DestinationSpace(0).String(),
		combat.DestinationSightRequired.String(), combat.DestinationSightNotRequired.String(), combat.DestinationSight(0).String(),
		combat.StayBehind.String(), combat.Accompany.String(), combat.TouchedCreaturePolicy(9).String()}
	want := []string{"unoccupied", "occupied_by_creature", "blocked_by_solid_obstacle", "unknown", "required", "not_required", "unknown", "stay_behind", "accompany", "unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
