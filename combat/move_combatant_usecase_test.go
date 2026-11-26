package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-action-resolution.md FR-025, FR-056, FR-059, FR-063, 4.9.1 ordinary move - moving 30 feet spends the movement and produces MovementSpent only: no Reaction is taken, no Opportunity Attack and no Prone, and every other combatant is as it was.
// Path: happy.
func TestMoveCombatantMoves(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	got := resMove(t, e, "a", resAlong(t, 30))
	if want := []string{"movement_spent"}; !slices.Equal(resKinds(got.Events()), want) {
		t.Errorf("events = %v, want %v", resKinds(got.Events()), want)
	}
	for _, id := range []string{"a", "b", "c"} {
		if !encMember(t, e, id).Equal(encMember(t, got.Encounter(), id)) || got.Encounter().Tracker().ReactionSpent(cid(id)) {
			t.Errorf("combatant %s changed or spent its Reaction [FR-025]", id)
		}
	}
	if got.Encounter().Tracker().Resources() == e.Tracker().Resources() {
		t.Errorf("the movement was not recorded in the tracker")
	}
}

// Requirement: per case
// Path: per case.
func TestMoveCombatantRestrictions(t *testing.T) {
	t.Parallel()
	crawl, walk := trkMode(t, 10, combat.MovementModeCrawl), trkWalk(t, 10)
	along := func(segs ...combat.MoveSegment) combat.MovementRequest {
		m, err := combat.NewMoveAlong(segs)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	speedZero := combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionSpeedZeroedByCondition}
	crawlOnly := combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionProneAllowsOnlyCrawl}
	cases := []struct {
		name, requirement, path string
		conds                   []combat.Condition
		req                     combat.MovementRequest
		want                    combat.ResolutionRejection
	}{
		{"Grappled, profile Speed 30", resSpec + "FR-057, 4.9.1", "unhappy (rule interaction)", []combat.Condition{combat.Grappled{Grappler: cid("g")}}, along(walk), speedZero},
		{"Restrained, profile Speed 30", resSpec + "FR-057, 4.9.1", "unhappy (rule interaction)", []combat.Condition{combat.Restrained{}}, along(walk), speedZero},
		{"Prone walking", resSpec + "FR-058, 4.9.1", "unhappy (rule interaction)", []combat.Condition{combat.Prone{}}, along(walk), crawlOnly},
		{"Prone, crawl then walk", resSpec + "FR-058, 4.9.1", "unhappy (rule interaction)", []combat.Condition{combat.Prone{}}, along(crawl, walk), crawlOnly},
		{"Prone, already Prone drop", resSpec + "FR-060, 4.9.1", "unhappy (state)", []combat.Condition{combat.Prone{}}, combat.DropProne{}, combat.PostureMismatchRejection{Mismatch: combat.PostureMismatchAlreadyProne}},
		{"Grappled, profile Speed 30, drop", resSpec + "FR-124, 4.9.1", "unhappy (rule interaction)", []combat.Condition{combat.Grappled{Grappler: cid("g")}}, combat.DropProne{}, speedZero},
		{"standing stands up", resSpec + "FR-061, 4.9.1", "unhappy (state)", nil, standUp(t, 30), combat.PostureMismatchRejection{Mismatch: combat.PostureMismatchNotProne}},
		{"Unconscious cannot right itself", resSpec + "FR-061, 4.9.1", "unhappy (boundary)", []combat.Condition{combat.Unconscious{}}, standUp(t, 30),
			combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionCannotRightItself}},
		{"Prone with Speed 0 cannot right itself", resSpec + "FR-061, 4.9.1", "unhappy (boundary)", []combat.Condition{combat.Prone{}}, standUp(t, 0),
			combat.MovementRestrictedRejection{Restriction: combat.MovementRestrictionCannotRightItself}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, _ := encStart(t, resPC(t, "a", combat.SideA, c.conds...), resPC(t, "b", combat.SideB))
			res, err := resMoveRun(t, e, "a", c.req)
			if err != nil {
				t.Fatal(err)
			}
			resWant(t, resAs[combat.MoveCombatantRejected](t, res), e, combat.ActionLedger{}, c.want, nil)
		})
	}
	// Exhaustion reduces the Speed and does not zero it: the move is not stopped here.
	e, _ := encStart(t, resPC(t, "a", combat.SideA, exhaustion(1)), resPC(t, "b", combat.SideB))
	resMove(t, e, "a", along(walk))
	// A Prone creature crawls at double cost: 10 feet of crawling spend 20 feet of movement.
	prone, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Prone{}), resPC(t, "b", combat.SideB))
	got := resMove(t, prone, "a", along(crawl))
	if spent := got.Events()[0].(combat.TurnEventOccurred).Event().(combat.MovementSpent); spent.TotalFeet() != 20 {
		t.Errorf("crawling 10 feet cost %d, want 20 [4.9.1 Prone move]", spent.TotalFeet())
	}
}

func standUp(tb testing.TB, base int) combat.MovementRequest {
	tb.Helper()
	s, err := combat.NewStandUp(base)
	if err != nil {
		tb.Fatal(err)
	}
	return s
}

// Requirement: doc/specs/combat-action-resolution.md FR-060, FR-010 step 5 before step 6, 4.9.1 drop Prone - dropping Prone applies Prone through the encounter and spends no action and no movement; a profile Speed of 0 gives the tracker's refusal before the conditions' (also when Grappled), and a creature that is not active is refused by the tracker first.
// Path: per case.
func TestMoveCombatantDropsProne(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	got := resMove(t, e, "a", combat.DropProne{})
	if want := []string{"dropped_prone", "condition_applied"}; !slices.Equal(resKinds(got.Events()), want) || !resHas(t, got.Encounter(), "a", content.ConditionProne) ||
		got.Encounter().Tracker().Resources() != e.Tracker().Resources() {
		t.Errorf("events %v, Prone %v: want %v, Prone, and no action or movement spent [FR-060]", resKinds(got.Events()), resHas(t, got.Encounter(), "a", content.ConditionProne), want)
	}
	zeroSpeed := resStartWith(t, map[string]int{"a": 0}, resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	grappledZero := resStartWith(t, map[string]int{"a": 0}, resPC(t, "a", combat.SideA, combat.Grappled{Grappler: cid("g")}), resPC(t, "b", combat.SideB))
	cases := []struct {
		name, requirement string
		e                 combat.Encounter
		who               string
		want              combat.ResolutionRejection
	}{
		{"profile Speed 0", resSpec + "FR-124, 4.9.1", zeroSpeed, "a", combat.TrackerRejection{Rejection: combat.TurnSpeedIsZeroRejection{}}},
		{"Grappled with profile Speed 0: tracker first", resSpec + "FR-010, FR-124, 4.9.1", grappledZero, "a", combat.TrackerRejection{Rejection: combat.TurnSpeedIsZeroRejection{}}},
		{"not the active combatant", resSpec + "FR-056, 4.9.1", e, "b", combat.TrackerRejection{Rejection: combat.TurnNotActiveCombatantRejection{}}},
	}
	for _, c := range cases {
		res, err := resMoveRun(t, c.e, c.who, combat.DropProne{})
		if err != nil {
			t.Fatal(err)
		}
		resWant(t, resAs[combat.MoveCombatantRejected](t, res), c.e, combat.ActionLedger{}, c.want, nil)
	}
	// Immunity to Prone: the tracker accepted the posture change and the root refused the condition, so nothing is committed (FR-062).
	immune, _ := encStart(t, resImmune(t, "a", combat.SideA, content.ConditionProne), resPC(t, "b", combat.SideB))
	res, err := resMoveRun(t, immune, "a", combat.DropProne{})
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.MoveCombatantRejected](t, res), immune, combat.ActionLedger{},
		combat.EncounterStateRejection{Rejection: combat.ImmuneToConditionRejection{ID: cid("a"), Kind: content.ConditionProne}}, nil)
}

// Requirement: doc/specs/combat-action-resolution.md FR-061, 4.9.1 stand up - standing up costs half the Speed rounded down (15, 12, 0) and removes the explicit Prone; with too little movement left the tracker refuses.
// Path: per case.
func TestMoveCombatantStandsUp(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		base int
		cost int
	}{{"Speed 30", 30, 15}, {"Speed 25 rounds down", 25, 12}, {"Speed 1", 1, 0}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := resStartWith(t, map[string]int{"a": c.base}, resPC(t, "a", combat.SideA, combat.Prone{}), resPC(t, "b", combat.SideB))
			got := resMove(t, e, "a", standUp(t, c.base))
			stood := got.Events()[0].(combat.TurnEventOccurred).Event().(combat.StoodUp)
			if want := []string{"stood_up", "condition_removed"}; !slices.Equal(resKinds(got.Events()), want) || stood.CostFeet() != c.cost || resHas(t, got.Encounter(), "a", content.ConditionProne) {
				t.Errorf("events %v, cost %d, Prone %v; want %v, %d and standing [%sFR-061]", resKinds(got.Events()), stood.CostFeet(), resHas(t, got.Encounter(), "a", content.ConditionProne), want, c.cost, resSpec)
			}
		})
	}
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resPC(t, "b", combat.SideB))
	spent := resMove(t, e, "a", resAlong(t, 20)).Encounter()
	dropped := resMove(t, spent, "a", combat.DropProne{}).Encounter()
	res, err := resMoveRun(t, dropped, "a", standUp(t, 30))
	if err != nil {
		t.Fatal(err)
	}
	rej := resAs[combat.MoveCombatantRejected](t, res)
	if _, ok := rej.Rejection().(combat.TrackerRejection).Rejection.(combat.TurnInsufficientMovementRejection); !ok || !rej.Encounter().Equal(dropped) {
		t.Errorf("rejection = %#v, want the tracker's insufficient movement with the encounter unchanged", rej.Rejection())
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-010, FR-011, FR-012, FR-143, FR-096 - the stops of a move: an ended encounter, an unknown combatant (invalid), a dead one; and the request constructors refuse a MoveAlong not built by NewMoveAlong, a nil request and a negative stand-up Speed.
// Path: per case.
func TestMoveCombatantStopsAndCommand(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	for who, want := range map[string]combat.ResolutionRejection{
		"b": combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}},
	} {
		res, err := resMoveRun(t, e, who, resAlong(t, 10))
		if err != nil {
			t.Fatal(err)
		}
		resWant(t, resAs[combat.MoveCombatantRejected](t, res), e, combat.ActionLedger{}, want, nil)
	}
	res, err := resMoveRun(t, e, "ghost", resAlong(t, 10))
	if inv := resAs[combat.MoveCombatantInvalid](t, res); err != nil || inv.Problem() != (combat.UnknownCombatantProblem{ID: cid("ghost")}) {
		t.Errorf("= %#v, %v, want an invalid result naming ghost", res, err)
	}
	ended := resEnded(t, e)
	res, err = resMoveRun(t, ended, "a", resAlong(t, 10))
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.MoveCombatantRejected](t, res), ended, combat.ActionLedger{}, combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil)
	for name, req := range map[string]combat.MovementRequest{"nil": nil, "zero MoveAlong": combat.MoveAlong{}} {
		if _, err := combat.NewMoveCombatantCommand(e, cid("a"), req); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := combat.NewMoveCombatantCommand(combat.Encounter{}, cid("a"), combat.DropProne{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero encounter gives %v, want ErrInvalidInput", err)
	}
}
