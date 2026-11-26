package combat_test

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-004 - NewDeployedGearID rejects an empty or blank string with a blank_id GearError matching ErrInvalidInput, keeps a valid id as given, and MustDeployedGearID panics with that error.
// Path: unhappy (invalid input).
func TestNewDeployedGearID(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", " ", "\t\n"} {
		if _, err := combat.NewDeployedGearID(s); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemBlankID}) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewDeployedGearID(%q) error = %v, want blank_id", s, err)
		}
	}
	if id, err := combat.NewDeployedGearID(" a "); err != nil || id != " a " {
		t.Errorf("NewDeployedGearID(\" a \") = %q, %v; want it unchanged", id, err)
	}
	defer func() {
		if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
			t.Errorf("recover() = %v, want an error matching ErrInvalidInput", r)
		}
	}()
	combat.MustDeployedGearID("")
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-015 - NewTurnStamp rejects a round below 1 (negative_round) and a zero creature (blank_id), accepts round 1 and the largest int, and compares by both fields.
// Path: unhappy (boundary).
func TestNewTurnStamp(t *testing.T) {
	t.Parallel()
	for _, r := range []int{0, -1, math.MinInt} {
		if _, err := combat.NewTurnStamp(r, gid("a")); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemNegativeRound}) {
			t.Errorf("round %d: err = %v, want negative_round", r, err)
		}
	}
	if _, err := combat.NewTurnStamp(1, ""); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemBlankID}) {
		t.Errorf("zero creature: err = %v, want blank_id", err)
	}
	s, err := combat.NewTurnStamp(1, gid("a"))
	if err != nil || s.Round() != 1 || !s.IsCombatant(gid("a")) || s.IsCombatant(gid("b")) || s.IsZero() {
		t.Fatalf("NewTurnStamp(1, a) = %+v, %v", s, err)
	}
	if s != gearStamp(1, "a") || s == gearStamp(2, "a") || s == gearStamp(1, "b") {
		t.Error("stamps must compare by round and creature")
	}
	if m, err := combat.NewTurnStamp(math.MaxInt, gid("a")); err != nil || m.Round() != math.MaxInt {
		t.Errorf("NewTurnStamp(MaxInt) = %+v, %v", m, err)
	}
	if !(combat.TurnStamp{}).IsZero() || (combat.TurnStamp{}).IsCombatant("") {
		t.Error("the zero stamp is invalid and nobody's turn")
	}
	defer func() {
		if r, _ := recover().(error); !errors.Is(r, srderr.ErrInvalidInput) {
			t.Errorf("recover() = %v, want ErrInvalidInput", r)
		}
	}()
	combat.MustTurnStamp(0, gid("a"))
}

// FuzzNewTurnStamp checks the constructors never panic and agree with their rules for any input.
func FuzzNewTurnStamp(f *testing.F) {
	f.Add(1, "a")
	f.Add(0, "")
	f.Add(math.MaxInt, " ")
	f.Add(math.MinInt, "a")
	f.Fuzz(func(t *testing.T, round int, who string) {
		s, err := combat.NewTurnStamp(round, combat.CombatantID(who))
		if (err == nil) != (round >= 1 && who != "") {
			t.Fatalf("NewTurnStamp(%d, %q) err = %v", round, who, err)
		}
		if err == nil && (s.Round() != round || s.IsZero()) {
			t.Fatalf("stamp = %+v", s)
		}
		id, err := combat.NewDeployedGearID(who)
		if (err == nil) != (len(who) > 0 && len(strings.TrimSpace(who)) > 0) || err == nil && string(id) != who {
			t.Fatalf("NewDeployedGearID(%q) = %q, %v", who, id, err)
		}
		if _, err := combat.NewHoldID(who); (err == nil) != (strings.TrimSpace(who) != "") {
			t.Fatalf("NewHoldID(%q) err = %v", who, err)
		}
	})
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-005, FR-006, FR-007, FR-008, FR-009, table 4.9.2 - Deploy adds a gear with the initial state of its kind and one DeployedGearPlaced when the id is unused, the distance is within reach and the ground rule holds: Caltrops 5 and 6 feet, Ball Bearings 10 and 11, Oil 5 and 6, a Hunting Trap at any distance; Ball Bearings and Oil refuse ground that is not level and Caltrops and the Hunting Trap accept it; the refusal order is idInUse, outOfReach, groundNotLevel (SRD 5.2.1, pp. 96-98).
// Path: per case.
func TestDeployedGearsDeploy(t *testing.T) {
	t.Parallel()
	const caltrops, bearings, oil, trap = combat.DeployedGearKindCaltrops, combat.DeployedGearKindBallBearings, combat.DeployedGearKindPouredOil, combat.DeployedGearKindHuntingTrap
	level, rough := combat.GroundLevelLevel, combat.GroundLevelNotLevel
	cases := []struct {
		name, path string
		kind       combat.DeployedGearKind
		distance   int
		ground     combat.GroundLevel
		taken      bool
		want       combat.GearRejection
		state      combat.DeployedGearState
	}{
		{"caltrops at 5 feet", "unhappy (boundary)", caltrops, 5, level, false, 0, combat.GearInForce{}},
		{"caltrops at 6 feet", "unhappy (boundary)", caltrops, 6, level, false, combat.GearRejectionOutOfReach, nil},
		{"caltrops at 0 feet", "unhappy (boundary)", caltrops, 0, level, false, 0, combat.GearInForce{}},
		{"caltrops on rough ground", "unhappy (rule interaction)", caltrops, 5, rough, false, 0, combat.GearInForce{}},
		{"ball bearings at 10 feet", "unhappy (boundary)", bearings, 10, level, false, 0, combat.GearInForce{}},
		{"ball bearings at 11 feet", "unhappy (boundary)", bearings, 11, level, false, combat.GearRejectionOutOfReach, nil},
		{"ball bearings on rough ground", "unhappy (state error)", bearings, 10, rough, false, combat.GearRejectionGroundNotLevel, nil},
		{"oil at 5 feet is unlit", "happy", oil, 5, level, false, 0, combat.OilUnlit{}},
		{"oil at 6 feet", "unhappy (boundary)", oil, 6, level, false, combat.GearRejectionOutOfReach, nil},
		{"oil on rough ground", "unhappy (state error)", oil, 5, rough, false, combat.GearRejectionGroundNotLevel, nil},
		{"hunting trap is armed at any distance", "happy", trap, math.MaxInt, level, false, 0, combat.GearArmed{}},
		{"hunting trap on rough ground", "unhappy (rule interaction)", trap, 40, rough, false, 0, combat.GearArmed{}},
		{"id in use wins over reach", "unhappy (state error)", caltrops, 99, level, true, combat.GearRejectionIDInUse, nil},
		{"reach wins over ground", "unhappy (rule interaction)", bearings, 99, rough, false, combat.GearRejectionOutOfReach, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var set combat.DeployedGears
			if c.taken {
				set = gearOne(t, trap)
			}
			id := gearID("new")
			if c.taken {
				id = gearID("g")
			}
			tr, err := set.Deploy(id, c.kind, gid("owner"), c.distance, c.ground)
			if err != nil {
				t.Fatal(err)
			}
			if c.want != 0 {
				if r, ok := tr.(combat.DeployedGearsRejected); !ok || r.Reason() != c.want {
					t.Errorf("transition = %#v, want rejected %v [%s FR-009]", tr, c.want, gearSpec)
				}
				if set.Len() != map[bool]int{true: 1, false: 0}[c.taken] {
					t.Error("a refusal must leave the set unchanged")
				}
				return
			}
			a := tr.(combat.DeployedGearsApplied)
			g, ok := a.Gears().Find(id)
			if !ok || g.Kind() != c.kind || g.State() != c.state || !g.IsDeployedBy(gid("owner")) || g.IsDeployedBy(gid("other")) || g.ID() != id {
				t.Errorf("gear = %+v, want kind %v state %#v [%s FR-005]", g, c.kind, c.state, gearSpec)
			}
			ev := a.Events()
			if len(ev) != 1 {
				t.Fatalf("events = %v, want one", ev)
			}
			placed := ev[0].(combat.DeployedGearPlaced)
			combattest.CheckEvent(t, placed, "deployed_gear_placed", gid("owner"))
			if placed.GearID() != id || placed.GearKind() != c.kind {
				t.Errorf("event = %+v", placed)
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-006, FR-071 - Deploy returns a GearError, never a refusal, for a negative distance (negative_distance), a blank id or zero deployer (blank_id) and a kind or ground that is not valid (invalid_value).
// Path: unhappy (invalid input).
func TestDeployedGearsDeployInvalid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		id       combat.DeployedGearID
		kind     combat.DeployedGearKind
		deployer combat.CombatantID
		distance int
		ground   combat.GroundLevel
		want     combat.GearProblem
	}{
		{"negative distance", "g", combat.DeployedGearKindCaltrops, "a", -1, combat.GroundLevelLevel, combat.GearProblemNegativeDistance},
		{"smallest distance", "g", combat.DeployedGearKindCaltrops, "a", math.MinInt, combat.GroundLevelLevel, combat.GearProblemNegativeDistance},
		{"blank id", " ", combat.DeployedGearKindCaltrops, "a", 0, combat.GroundLevelLevel, combat.GearProblemBlankID},
		{"zero deployer", "g", combat.DeployedGearKindCaltrops, "", 0, combat.GroundLevelLevel, combat.GearProblemBlankID},
		{"invalid kind", "g", 0, "a", 0, combat.GroundLevelLevel, combat.GearProblemInvalidValue},
		{"invalid ground", "g", combat.DeployedGearKindCaltrops, "a", 0, 0, combat.GearProblemInvalidValue},
	}
	for _, c := range cases {
		tr, err := combat.DeployedGears{}.Deploy(c.id, c.kind, c.deployer, c.distance, c.ground)
		if tr != nil || !errors.Is(err, &combat.GearError{Problem: c.want}) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: Deploy = %v, %v; want a %v GearError [%s FR-006]", c.name, tr, err, c.want, gearSpec)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-010 - Recover removes Caltrops or Ball Bearings with one DeployedGearRecovered, refuses Oil and a Hunting Trap with notRecoverable and an unknown id with unknownGear; the 10 minutes are profile data (SRD 5.2.1, "Caltrops", "Ball Bearings", p. 96).
// Path: happy; unhappy (state error).
func TestDeployedGearsRecover(t *testing.T) {
	t.Parallel()
	for _, k := range []combat.DeployedGearKind{combat.DeployedGearKindCaltrops, combat.DeployedGearKindBallBearings} {
		a := gearsApplied(t, gearOne(t, k).Recover(gearID("g")), nil)
		ev := a.Events()
		if a.Gears().Len() != 0 || len(ev) != 1 || ev[0].(combat.DeployedGearRecovered).GearKind() != k || ev[0].(combat.DeployedGearRecovered).GearID() != gearID("g") {
			t.Errorf("Recover(%v) = %+v, want the gear removed with one DeployedGearRecovered", k, a)
		}
		combattest.CheckEvent(t, ev[0], "deployed_gear_recovered", gid("owner"))
	}
	for _, k := range []combat.DeployedGearKind{combat.DeployedGearKindPouredOil, combat.DeployedGearKindHuntingTrap} {
		set := gearOne(t, k)
		if r, ok := set.Recover(gearID("g")).(combat.DeployedGearsRejected); !ok || r.Reason() != combat.GearRejectionNotRecoverable {
			t.Errorf("Recover(%v) must be refused with notRecoverable", k)
		}
	}
	if r, ok := (combat.DeployedGears{}).Recover(gearID("g")).(combat.DeployedGearsRejected); !ok || r.Reason() != combat.GearRejectionUnknownGear {
		t.Error("an unknown id must be refused with unknownGear")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-011, FR-012, table 4.9.2 - Light makes Unlit Oil burn through the end of the turn two rounds later of the lighter (lit at (1, A) burns through (3, A)) with one PouredOilLit and refuses Oil already lit, gear that is not Oil and an unknown id; Expire burns out Oil through the ended turn or through an earlier round, in deployment order, and leaves the rest, which an empty set returns as it is; the round saturates at the largest int (SRD 5.2.1, "Oil", p. 98).
// Path: per case.
func TestDeployedGearsLightAndExpire(t *testing.T) {
	t.Parallel()
	lit := gearBurning(t, 1, "a")
	g := gearOf(t, lit)
	if o, ok := g.State().(combat.OilBurning); !ok || o.BurnsThrough() != gearStamp(3, "a") {
		t.Fatalf("state = %#v, want burning through (3, a) [FR-011]", g.State())
	}
	tr, err := gearOne(t, combat.DeployedGearKindPouredOil).Light(gearID("g"), gearStamp(1, "a"))
	ev := gearsApplied(t, tr, err).Events()
	pol := ev[0].(combat.PouredOilLit)
	combattest.CheckEvent(t, pol, "poured_oil_lit", gid("owner"), gid("a"))
	if pol.BurnsThrough() != gearStamp(3, "a") || pol.GearID() != gearID("g") {
		t.Errorf("event = %+v", pol)
	}
	for name, c := range map[string]struct {
		set  combat.DeployedGears
		id   combat.DeployedGearID
		want combat.GearRejection
	}{
		"already lit":   {lit, "g", combat.GearRejectionAlreadyLit},
		"not oil":       {gearOne(t, combat.DeployedGearKindCaltrops), "g", combat.GearRejectionNotPouredOil},
		"unknown":       {lit, "x", combat.GearRejectionUnknownGear},
		"not oil, trap": {gearOne(t, combat.DeployedGearKindHuntingTrap), "g", combat.GearRejectionNotPouredOil},
	} {
		tr, err := c.set.Light(c.id, gearStamp(1, "a"))
		if r, ok := tr.(combat.DeployedGearsRejected); err != nil || !ok || r.Reason() != c.want {
			t.Errorf("%s: Light = %#v, %v; want %v", name, tr, err, c.want)
		}
	}
	if _, err := lit.Light(gearID("g"), combat.TurnStamp{}); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("Light with the zero stamp: err = %v, want invalid_value", err)
	}
	max, err := gearOne(t, combat.DeployedGearKindPouredOil).Light(gearID("g"), gearStamp(math.MaxInt, "a"))
	if o := gearOf(t, gearsApplied(t, max, err).Gears()).State().(combat.OilBurning); o.BurnsThrough().Round() != math.MaxInt {
		t.Errorf("round past the largest int = %d, want it saturated", o.BurnsThrough().Round())
	}

	expire := []struct {
		name, path string
		at         combat.TurnStamp
		burnt      bool
	}{
		{"expire at (3, a) burns it out", "unhappy (boundary)", gearStamp(3, "a"), true},
		{"expire at (3, b) leaves it", "unhappy (boundary)", gearStamp(3, "b"), false},
		{"expire at (2, a) leaves it", "unhappy (boundary)", gearStamp(2, "a"), false},
		{"expire at (4, b) burns it out", "unhappy (boundary)", gearStamp(4, "b"), true},
	}
	for _, c := range expire {
		got, err := lit.Expire(c.at)
		if err != nil {
			t.Fatal(err)
		}
		if (got.Gears().Len() == 0) != c.burnt || len(got.Events()) != map[bool]int{true: 1, false: 0}[c.burnt] {
			t.Errorf("%s: gears %d, events %d [%s FR-012]", c.name, got.Gears().Len(), len(got.Events()), gearSpec)
		}
		if c.burnt {
			out := got.Events()[0].(combat.PouredOilBurnedOut)
			combattest.CheckEvent(t, out, "poured_oil_burned_out", gid("owner"), gid("a"))
		} else if !got.Gears().Equal(lit) {
			t.Error("gear that has not burned out must be unchanged")
		}
	}
	empty, err := combat.DeployedGears{}.Expire(gearStamp(1, "a"))
	if err != nil || empty.Gears().Len() != 0 || len(empty.Events()) != 0 {
		t.Errorf("Expire on the empty set = %+v, %v", empty, err)
	}
	if _, err := lit.Expire(combat.TurnStamp{}); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("Expire with the zero stamp: err = %v, want invalid_value", err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-012 - Expire burns out several Oils in deployment order, one PouredOilBurnedOut each, and keeps gear that is not burning oil.
// Path: unhappy (rule interaction).
func TestDeployedGearsExpireManyInOrder(t *testing.T) {
	t.Parallel()
	set := gearDeploy(t, combat.DeployedGears{}, "z-oil", combat.DeployedGearKindPouredOil)
	set = gearDeploy(t, set, "caltrops", combat.DeployedGearKindCaltrops)
	set = gearDeploy(t, set, "a-oil", combat.DeployedGearKindPouredOil)
	for _, id := range []string{"z-oil", "a-oil"} {
		tr, err := set.Light(gearID(id), gearStamp(1, "a"))
		set = gearsApplied(t, tr, err).Gears()
	}
	got, err := set.Expire(gearStamp(9, "x"))
	if err != nil {
		t.Fatal(err)
	}
	ids := []combat.DeployedGearID{}
	for _, e := range got.Events() {
		ids = append(ids, e.(combat.PouredOilBurnedOut).GearID())
	}
	if !slices.Equal(ids, []combat.DeployedGearID{"z-oil", "a-oil"}) || got.Gears().Len() != 1 {
		t.Errorf("burned out %v with %d gears left, want [z-oil a-oil] in deployment order and the Caltrops left", ids, got.Gears().Len())
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-013, FR-022, table 4.9.1 (reset occupied / free), Q-11 - Reset arms a sprung Hunting Trap with one HuntingTrapReset when no hold in the set has the trap's id and refuses trapOccupied when one has; an armed trap, other gear and an unknown id are refused notSprung or unknownGear; Spring closes an armed trap with one HuntingTrapSprung and refuses a sprung trap and other gear (SRD 5.2.1, "Hunting Trap", p. 97).
// Path: unhappy (state error).
func TestDeployedGearsResetAndSpring(t *testing.T) {
	t.Parallel()
	sprung := gearSprung(t)
	caught := gearHolds(t, gearHold(t, combat.GearHoldInput{ID: hid("h"), Kind: combat.HoldKindHuntingTrap, Target: gid("walker"),
		Restraint: combat.HoldRestraintDoesNotRestrain, GearID: gearID("g"), MovementLimitFeet: 3}))
	other := gearHolds(t, gearHold(t, holdIn("h", combat.HoldKindNet)))
	reset := gearsApplied(t, sprung.Reset(gearID("g"), other), nil)
	combattest.CheckEvent(t, reset.Events()[0], "hunting_trap_reset", gid("owner"))
	if gearOf(t, reset.Gears()).State() != (combat.GearArmed{}) || reset.Events()[0].(combat.HuntingTrapReset).GearID() != gearID("g") {
		t.Error("a free sprung trap must be armed again")
	}
	if gearsApplied(t, sprung.Reset(gearID("g"), combat.GearHolds{}), nil).Gears().Len() != 1 {
		t.Error("reset with no holds must succeed")
	}
	for name, c := range map[string]struct {
		set   combat.DeployedGears
		holds combat.GearHolds
		id    combat.DeployedGearID
		want  combat.GearRejection
	}{
		"occupied":     {sprung, caught, "g", combat.GearRejectionTrapOccupied},
		"armed":        {gearOne(t, combat.DeployedGearKindHuntingTrap), combat.GearHolds{}, "g", combat.GearRejectionNotSprung},
		"not a trap":   {gearOne(t, combat.DeployedGearKindCaltrops), combat.GearHolds{}, "g", combat.GearRejectionNotSprung},
		"unknown gear": {sprung, combat.GearHolds{}, "x", combat.GearRejectionUnknownGear},
	} {
		if r, ok := c.set.Reset(c.id, c.holds).(combat.DeployedGearsRejected); !ok || r.Reason() != c.want {
			t.Errorf("%s: Reset must be refused with %v", name, c.want)
		}
	}

	armed := gearOne(t, combat.DeployedGearKindHuntingTrap)
	tr, err := armed.Spring(gearID("g"), gid("walker"))
	a := gearsApplied(t, tr, err)
	combattest.CheckEvent(t, a.Events()[0], "hunting_trap_sprung", gid("owner"), gid("walker"))
	if gearOf(t, a.Gears()).State() != (combat.GearSprung{}) || a.Events()[0].(combat.HuntingTrapSprung).GearID() != gearID("g") {
		t.Error("an armed trap must be sprung")
	}
	for name, c := range map[string]struct {
		set  combat.DeployedGears
		id   combat.DeployedGearID
		want combat.GearRejection
	}{
		"already sprung": {sprung, "g", combat.GearRejectionTrapAlreadySprung},
		"not a trap":     {gearOne(t, combat.DeployedGearKindCaltrops), "g", combat.GearRejectionTriggerNotApplicable},
		"unknown":        {armed, "x", combat.GearRejectionUnknownGear},
	} {
		tr, err := c.set.Spring(c.id, gid("walker"))
		if r, ok := tr.(combat.DeployedGearsRejected); err != nil || !ok || r.Reason() != c.want {
			t.Errorf("%s: Spring = %#v, %v; want %v", name, tr, err, c.want)
		}
	}
	if _, err := armed.Spring(gearID("g"), ""); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemBlankID}) {
		t.Errorf("Spring with a zero creature: err = %v, want blank_id", err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-014 - NewDeployedGears rejects a zero gear and duplicate ids with ErrInvalidInput, copies its input, returns views that are copies, compares equal whatever the insertion order and iterates in deployment order.
// Path: unhappy (immutability, invalid input).
func TestNewDeployedGears(t *testing.T) {
	t.Parallel()
	set := gearDeploy(t, gearOne(t, combat.DeployedGearKindCaltrops), "h", combat.DeployedGearKindBallBearings)
	gears := set.Gears()
	if _, err := combat.NewDeployedGears([]combat.DeployedGear{gears[0], gears[0]}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("duplicate ids: err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewDeployedGears([]combat.DeployedGear{{}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero gear: err = %v, want ErrInvalidInput", err)
	}
	if empty, err := combat.NewDeployedGears(nil); err != nil || empty.Len() != 0 || !empty.Equal(combat.DeployedGears{}) {
		t.Errorf("nil input = %+v, %v; want the empty set", empty, err)
	}
	rebuilt, err := combat.NewDeployedGears(gears)
	if err != nil || !rebuilt.Equal(set) {
		t.Fatalf("rebuilt = %+v, %v", rebuilt, err)
	}
	gears[0], gears[1] = gears[1], gears[0]
	swapped, err := combat.NewDeployedGears(gears)
	if err != nil || !swapped.Equal(set) || !set.Equal(swapped) {
		t.Error("sets must compare equal whatever the order")
	}
	if got := set.Gears(); got[0].ID() != "g" || got[1].ID() != "h" {
		t.Errorf("gears = %v, want deployment order g, h", got)
	}
	if !rebuilt.Equal(set) || swapped.Gears()[0].ID() != "h" {
		t.Error("the input slice must have been copied")
	}
	view := set.Gears()
	view[0] = combat.DeployedGear{}
	if set.Gears()[0].IsZero() {
		t.Error("a returned view must be a copy")
	}
	if set.Equal(gearOne(t, combat.DeployedGearKindCaltrops)) || set.Equal(gearDeploy(t, gearOne(t, combat.DeployedGearKindCaltrops), "h", combat.DeployedGearKindHuntingTrap)) {
		t.Error("sets of different gears must differ")
	}
	if _, ok := set.Find("zz"); ok {
		t.Error("Find must report an absent gear")
	}
}
