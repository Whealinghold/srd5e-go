package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rosTurns advances the encounter n times and returns the ids whose turn started, with the round of each.
func rosTurns(tb testing.TB, e combat.Encounter, n int) []string {
	tb.Helper()
	var out []string
	for range n {
		var events []combat.Event
		e, events = advance(tb, e)
		for _, ev := range events {
			if s, ok := ev.(combat.TurnEventOccurred); ok {
				if st, ok := s.Event().(combat.TrackerTurnStarted); ok {
					out = append(out, string(st.Combatants()[0]))
				}
			}
		}
	}
	return out
}

// rosJoined returns the CombatantJoined event of an applied add.
func rosJoined(tb testing.TB, events []combat.Event) combat.CombatantJoined {
	tb.Helper()
	if len(events) != 1 {
		tb.Fatalf("events = %v, want exactly one CombatantJoined [FR-008]", encKinds(events))
	}
	j, ok := events[0].(combat.CombatantJoined)
	if !ok {
		tb.Fatalf("event = %#v, want CombatantJoined", events[0])
	}
	return j
}

// Requirement: per case (table 4.2a: FR-006, FR-008, FR-009, FR-010, FR-011, FR-027, FR-038)
// Path: per case.
func TestAddCombatantCases(t *testing.T) {
	t.Parallel()
	rows := []struct {
		name, requirement, path string
		check                   func(t *testing.T)
	}{
		{"summoner is active", "FR-006, FR-008", "happy", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			after, events := rosOK(t)(rosAdd(t, e, "x", combat.SideA, rosAfter("b")))
			j := rosJoined(t, events)
			if rosOrder(after) != "a,b,x,c,d" || rosList(after) != "a,b,x,c,d" || j.Count() != 15 || !slices.Equal(j.PrecededBy(), []combat.CombatantID{"b"}) {
				t.Errorf("order %s, count %d, preceded by %v, want a,b,x,c,d at 15 after b", rosOrder(after), j.Count(), j.PrecededBy())
			}
			if after.Tracker().Active().Combatant != cid("b") || after.Tracker().Round() != 2 || after.Status() != e.Status() {
				t.Errorf("the active combatant, round or status changed [FR-005, FR-008]")
			}
			if got := rosTurns(t, after, 3); !slices.Equal(got, []string{"x", "c", "d"}) {
				t.Errorf("next turns %v, want x first [FR-006]", got)
			}
		}},
		{"latecomer placed before the active combatant", "FR-006", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			after, events := rosOK(t)(rosAdd(t, rosEnc(t), "x", combat.SideB, rosAt(20, rosBehind)))
			j := rosJoined(t, events)
			if rosOrder(after) != "x,a,b,c,d" || j.Count() != 20 || len(j.PrecededBy()) != 0 {
				t.Errorf("order %s, count %d, preceded by %v, want x first at 20", rosOrder(after), j.Count(), j.PrecededBy())
			}
			if after.Tracker().Active().Combatant != cid("b") || after.Tracker().Round() != 2 {
				t.Errorf("round and active combatant changed [FR-006]")
			}
			if got := rosTurns(t, after, 4); !slices.Equal(got, []string{"c", "d", "x", "a"}) {
				t.Errorf("next turns %v, want x only after the wrap [FR-006]", got)
			}
		}},
		{"duplicate id", "FR-009", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			e, _ := rosOK(t)(rosAdd(t, rosEnc(t), "x", combat.SideA, rosAfter("a")))
			tr, err := rosAdd(t, e, "x", combat.SideA, rosAfter("a"))
			if r := rosNo(t, e, tr, err); r != (combat.RosterDuplicateCombatantRejection{ID: "x"}) {
				t.Errorf("rejection = %#v, want RosterDuplicateCombatantRejection{x}", r)
			}
		}},
		{"unknown anchor", "FR-009", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			tr, err := rosAdd(t, e, "x", combat.SideA, rosAfter("z"))
			if r := rosNo(t, e, tr, err); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
				t.Errorf("rejection = %#v, want RosterUnknownCombatantRejection{z}", r)
			}
		}},
		{"ended and duplicate", "FR-009", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			e := rosEnded(t, rosEnc(t))
			tr, err := rosAdd(t, e, "a", combat.SideA, rosAfter("z"))
			if r := rosNo(t, e, tr, err); r != (combat.RosterEncounterEndedRejection{}) {
				t.Errorf("rejection = %#v, want RosterEncounterEndedRejection: ended comes before duplicate and unknown", r)
			}
		}},
		{"duplicate beats unknown anchor", "FR-009", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			tr, err := rosAdd(t, e, "a", combat.SideA, rosAfter("z"))
			if r := rosNo(t, e, tr, err); r != (combat.RosterDuplicateCombatantRejection{ID: "a"}) {
				t.Errorf("rejection = %#v, want the duplicate first", r)
			}
		}},
		{"profile id differs", "FR-010", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			if _, err := combat.NewAddCombatantToEncounterCommand(e, rosM(t, "x", combat.SideA), rosProfile(t, "y"), rosAfter("a"), combat.JoinReasonArrived); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("command: err = %v, want ErrInvalidInput", err)
			}
			if _, err := e.AddCombatant(rosM(t, "x", combat.SideA), rosProfile(t, "y"), rosAfter("a"), combat.JoinReasonArrived); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("method: err = %v, want ErrInvalidInput", err)
			}
		}},
		{"dead and Dying joiners", "FR-011", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			for name, c := range map[string]combat.Combatant{
				"dead":  encC(t, "x", combat.SideB, deadStatus(t, combat.MonsterDeath)),
				"dying": encC(t, "x", combat.SideB, dyingStatus(t, 20, 1, 1), combat.Unconscious{}),
			} {
				e := rosEnc(t)
				tr, err := e.AddCombatant(c, rosProfile(t, "x"), rosAt(10, rosBefore), combat.JoinReasonArrived)
				after, events := rosOK(t)(tr, err)
				if got, ok := after.Find(cid("x")); !ok || !got.Equal(c) || rosOrder(after) != "a,b,x,c,d" {
					t.Errorf("%s: the joiner changed or is misplaced: %s", name, rosOrder(after))
				}
				if len(events) != 1 {
					t.Errorf("%s: events %v, want only CombatantJoined", name, encKinds(events))
				}
				_, turn := advance(t, after)
				if name == "dying" && !slices.Equal(encKinds(turn), []string{"tracker_turn_ended", "tracker_turn_started", "dying_combatant_turn_started"}) {
					t.Errorf("dying: advance gave %v, want DyingCombatantTurnStarted when x's turn starts [FR-011]", encKinds(turn))
				}
			}
		}},
		{"other side joins", "FR-011", "happy", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			after, events := rosOK(t)(rosAdd(t, e, "x", combat.SideB, rosAfter("b")))
			if rosJoined(t, events).Side() != combat.SideB {
				t.Errorf("the event reports side %v, want side B", rosJoined(t, events).Side())
			}
			for _, id := range []string{"a", "b", "c", "d"} {
				if !encMember(t, after, id).Equal(encMember(t, e, id)) {
					t.Errorf("%s changed when x joined [FR-005]", id)
				}
			}
		}},
		{"anchor is the new id", "FR-009", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			tr, err := rosAdd(t, e, "x", combat.SideA, rosAfter("x"))
			if r := rosNo(t, e, tr, err); r != (combat.RosterUnknownCombatantRejection{ID: "x"}) {
				t.Errorf("rejection = %#v, want RosterUnknownCombatantRejection{x}", r)
			}
		}},
		{"dead anchor", "FR-009, FR-027", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			cs := rosBase(t)
			cs[0] = encC(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath))
			e := rosStart(t, 5, cs...)
			after, events := rosOK(t)(rosAdd(t, e, "x", combat.SideA, rosAfter("a")))
			if rosOrder(after) != "a,x,b,c,d" || rosJoined(t, events).Count() != 15 {
				t.Errorf("order %s, want x after the dead a at 15", rosOrder(after))
			}
		}},
		{"stale tracker after add", "FR-038", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			stale := e.Tracker()
			after, _ := rosOK(t)(rosAdd(t, e, "x", combat.SideA, rosAfter("a")))
			tr, err := after.ApplyTurnChange(stale, nil)
			if r := encNo(t, after, tr, err); r != (combat.TurnOrderChangedRejection{}) {
				t.Errorf("rejection = %#v, want TurnOrderChanged: take the tracker from the new encounter", r)
			}
			if tr, err := after.ApplyTurnChange(after.Tracker(), nil); err != nil || tr.Rejection() != nil {
				t.Errorf("the tracker of the new encounter is refused: %v, %v", err, tr.Rejection())
			}
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			r.check(t)
		})
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-008, FR-012, FR-037 - an add returns a new encounter (the receiver is unchanged), with the combatants list in the tracker's order, the status unchanged and one CombatantJoined carrying the encounter id, the round, the combatant, its side, the reason, its count and the entry before it.
// Path: happy.
func TestAddCombatantEventAndImmutability(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	before := rosEnc(t)
	x := rosM(t, "x", combat.SideB)
	tr, err := e.AddCombatant(x, rosProfile(t, "x"), rosAt(10, rosBehind), combat.JoinReasonSummoned)
	after, events := rosOK(t)(tr, err)
	if !e.Equal(before) {
		t.Errorf("AddCombatant changed its receiver [FR-037]")
	}
	j := rosJoined(t, events)
	if j.EncounterID() != eid() || j.Round() != 2 || j.Kind() != "combatant_joined" || !slices.Equal(rosIDs(j), []string{"x"}) || j.Side() != combat.SideB ||
		j.Reason() != combat.JoinReasonSummoned || j.Count() != 10 || !slices.Equal(j.PrecededBy(), []combat.CombatantID{"c"}) {
		t.Errorf("event = %#v, want x joining side B at count 10 after c in round 2 as summoned [FR-012]", j)
	}
	if got, ok := after.Find(cid("x")); !ok || !got.Equal(x) {
		t.Errorf("the joiner stored is %v, want it unchanged [FR-011]", got)
	}
	rosHolds(t, after)
	if got := tr.Events(); len(got) > 0 {
		got[0] = nil
		if tr.Events()[0] == nil {
			t.Errorf("Events returns the stored slice, not a copy [FR-035]")
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-013, FR-014, FR-015 - a summoned creature is placed after its summoner with the summoner's count even while the summoner is Incapacitated; a creature that rolls its own Initiative is placed by the count the host rolled with InitiativeCount.FromRoll; no roster operation fires on its own, so the roster stays the same through turns.
// Path: happy.
func TestSummonedAndRolledPlacements(t *testing.T) {
	t.Parallel()
	cs := rosBase(t)
	cs[1] = rosM(t, "b", combat.SideA, combat.Incapacitated{})
	e := rosStart(t, 5, cs...)
	after, events := rosOK(t)(e.AddCombatant(rosM(t, "x", combat.SideA), rosProfile(t, "x"), rosAfter("b"), combat.JoinReasonSummoned))
	if rosOrder(after) != "a,b,x,c,d" || rosJoined(t, events).Count() != rosCountOf(t, e, "b") || rosJoined(t, events).Reason() != combat.JoinReasonSummoned {
		t.Errorf("order %s: a summoned creature of an Incapacitated summoner shares its count and follows it [FR-013]", rosOrder(after))
	}
	roll, err := d20.NewRoll(modifier.Normal, []int{12})
	if err != nil {
		t.Fatal(err)
	}
	count := combat.InitiativeCount(0).FromRoll(roll, modifier.New(2), modifier.New(0))
	rolled, events := rosOK(t)(e.AddCombatant(rosM(t, "y", combat.SideB), rosProfile(t, "y"), rosAt(count, rosBehind), combat.JoinReasonArrived))
	if count != 14 || rosOrder(rolled) != "a,b,y,c,d" || rosJoined(t, events).Count() != 14 {
		t.Errorf("count %d, order %s, want y rolled at 14 between b and c [FR-014]", count, rosOrder(rolled))
	}
	run := rosEnc(t)
	list := rosList(run)
	for range 12 {
		run, _ = advance(t, run)
	}
	if rosList(run) != list {
		t.Errorf("the roster changed through twelve turns: %s [FR-015]", rosList(run))
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-009, FR-010 - an add with an invalid encounter, a zero or forged combatant, a profile without a size or a position or reason that is not defined is invalid input to the method, never a panic or a result.
// Path: unhappy (invalid input).
func TestAddCombatantInvalidInput(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	x := rosM(t, "x", combat.SideA)
	noSize := combat.CombatantProfile{ID: cid("x"), Speeds: trkSpeeds(t, 30)}
	cases := map[string]func() (rosTr, error){
		"zero encounter": func() (rosTr, error) {
			return combat.Encounter{}.AddCombatant(x, rosProfile(t, "x"), rosAfter("a"), combat.JoinReasonArrived)
		},
		"zero combatant": func() (rosTr, error) {
			return e.AddCombatant(combat.Combatant{}, rosProfile(t, "x"), rosAfter("a"), combat.JoinReasonArrived)
		},
		"profile no size": func() (rosTr, error) { return e.AddCombatant(x, noSize, rosAfter("a"), combat.JoinReasonArrived) },
		"nil position":    func() (rosTr, error) { return e.AddCombatant(x, rosProfile(t, "x"), nil, combat.JoinReasonArrived) },
		"blank anchor": func() (rosTr, error) {
			return e.AddCombatant(x, rosProfile(t, "x"), combat.RosterAfterCombatant{}, combat.JoinReasonArrived)
		},
		"undefined tie": func() (rosTr, error) {
			return e.AddCombatant(x, rosProfile(t, "x"), combat.RosterAtInitiative{Count: 3}, combat.JoinReasonArrived)
		},
		"undefined reason":  func() (rosTr, error) { return e.AddCombatant(x, rosProfile(t, "x"), rosAfter("a"), 0) },
		"reason out of set": func() (rosTr, error) { return e.AddCombatant(x, rosProfile(t, "x"), rosAfter("a"), 3) },
	}
	for name, f := range cases {
		if _, err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	for name, f := range map[string]func() error{
		"command zero encounter": func() error {
			_, err := combat.NewAddCombatantToEncounterCommand(combat.Encounter{}, x, rosProfile(t, "x"), rosAfter("a"), combat.JoinReasonArrived)
			return err
		},
		"command nil position": func() error {
			_, err := combat.NewAddCombatantToEncounterCommand(e, x, rosProfile(t, "x"), nil, combat.JoinReasonArrived)
			return err
		},
		"command undefined reason": func() error {
			_, err := combat.NewAddCombatantToEncounterCommand(e, x, rosProfile(t, "x"), rosAfter("a"), 0)
			return err
		},
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

// rosGrapple builds the base encounter where each id of holders is Grappled by the grappler.
func rosGrapple(tb testing.TB, grappler string, holders ...string) combat.Encounter {
	tb.Helper()
	cs := rosBase(tb)
	for i, c := range cs {
		id := string(c.Ref().ID)
		if slices.Contains(holders, id) {
			cs[i] = rosM(tb, id, c.Side(), combat.Grappled{Grappler: cid(grappler)}, combat.Poisoned{})
		}
	}
	return rosStart(tb, 5, cs...)
}

// Requirement: per case (table 4.3a: FR-016, FR-018 to FR-024)
// Path: per case.
func TestRemoveCombatantCases(t *testing.T) {
	t.Parallel()
	remove := func(t *testing.T, e combat.Encounter, id string, cause combat.RemovalCause) (combat.Encounter, []combat.Event) {
		t.Helper()
		return rosOK(t)(e.RemoveCombatant(cid(id), cause))
	}
	refuse := func(t *testing.T, e combat.Encounter, id string) combat.RosterRejection {
		t.Helper()
		tr, err := e.RemoveCombatant(cid(id), combat.RemovalCauseFled)
		return rosNo(t, e, tr, err)
	}
	rows := []struct {
		name, requirement, path string
		check                   func(t *testing.T)
	}{
		{"middle, not active", "FR-016", "happy", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			after, events := remove(t, e, "c", combat.RemovalCauseFled)
			if rosOrder(after) != "a,b,d" || rosList(after) != "a,b,d" || after.Tracker().Active().Combatant != cid("b") || after.Tracker().Round() != 2 {
				t.Errorf("order %s, active %v, round %d, want a,b,d with b active in round 2", rosOrder(after), after.Tracker().Active().Combatant, after.Tracker().Round())
			}
			if _, ok := after.Find(cid("c")); ok {
				t.Errorf("c is still a combatant")
			}
			if len(events) != 1 || events[0].Kind() != "combatant_removed" {
				t.Errorf("events = %v, want only combatant_removed", encKinds(events))
			}
			rosHolds(t, after)
		}},
		{"first, already passed", "FR-016", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			after, _ := remove(t, rosEnc(t), "a", combat.RemovalCauseDisappeared)
			if rosOrder(after) != "b,c,d" || after.Tracker().Active().Combatant != cid("b") {
				t.Errorf("order %s, active %v, want b,c,d with b still active", rosOrder(after), after.Tracker().Active().Combatant)
			}
		}},
		{"last", "FR-016, FR-020", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			after, _ := remove(t, rosEnc(t), "d", combat.RemovalCauseFled)
			if rosOrder(after) != "a,b,c" {
				t.Errorf("order %s, want a,b,c", rosOrder(after))
			}
		}},
		{"the active combatant", "FR-019", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "b"); r != (combat.RosterCombatantIsActiveRejection{ID: "b"}) {
				t.Errorf("rejection = %#v, want RosterCombatantIsActiveRejection{b}", r)
			}
		}},
		{"sole remaining member is active", "FR-020, FR-024", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			e, _ := remove(t, rosEnc(t), "a", combat.RemovalCauseFled)
			if r := refuse(t, e, "b"); r != (combat.RosterCombatantIsActiveRejection{ID: "b"}) {
				t.Errorf("rejection = %#v, want the active check before would-empty", r)
			}
		}},
		{"last of its side, not active", "FR-020, FR-024", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			e, _ := remove(t, rosEnc(t), "a", combat.RemovalCauseFled)
			e, _ = advance(t, e) // c is active
			if r := refuse(t, e, "b"); r != (combat.RosterWouldEmptySideRejection{Side: combat.SideA}) {
				t.Errorf("rejection = %#v, want RosterWouldEmptySideRejection{side A}", r)
			}
		}},
		{"two combatants, one per side", "FR-020", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			e := rosStart(t, 0, rosM(t, "a", combat.SideA), rosM(t, "c", combat.SideB))
			if r := refuse(t, e, "c"); r != (combat.RosterWouldEmptySideRejection{Side: combat.SideB}) {
				t.Errorf("rejection = %#v, want RosterWouldEmptySideRejection{side B}", r)
			}
		}},
		{"only combatant of a side, side empties", "FR-020", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			e, _ := remove(t, rosEnc(t), "c", combat.RemovalCauseFled)
			if r := refuse(t, e, "d"); r != (combat.RosterWouldEmptySideRejection{Side: combat.SideB}) {
				t.Errorf("rejection = %#v, want RosterWouldEmptySideRejection{side B}", r)
			}
			if _, ok := e.Status().(combat.EncounterInProgress); !ok {
				t.Errorf("status = %#v, want the encounter still in progress: it does not end by itself [FR-020]", e.Status())
			}
			ended := runEnd(t, e, combat.EndReasonSideDefeated{Side: combat.SideB})
			if _, ok := ended.(combat.EndEncounterApplied); !ok {
				t.Errorf("end with a side defeated = %#v, want applied", ended)
			}
		}},
		{"unknown id", "FR-024", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "z"); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
				t.Errorf("rejection = %#v, want RosterUnknownCombatantRejection{z}", r)
			}
		}},
		{"ended", "FR-024", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			e := rosEnded(t, rosEnc(t))
			for _, id := range []string{"c", "z", "b"} {
				if r := refuse(t, e, id); r != (combat.RosterEncounterEndedRejection{}) {
					t.Errorf("remove %s = %#v, want RosterEncounterEndedRejection first", id, r)
				}
			}
		}},
		{"dead leaver", "FR-018", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			cs := rosBase(t)
			cs[2] = encC(t, "c", combat.SideB, deadStatus(t, combat.MonsterDeath))
			after, events := remove(t, rosStart(t, 5, cs...), "c", combat.RemovalCauseDisappeared)
			if !slices.Equal(encKinds(events), []string{"combatant_removed"}) || rosOrder(after) != "a,b,d" {
				t.Errorf("events %v, order %s, want only combatant_removed and no Died or Hit Point event [FR-018]", encKinds(events), rosOrder(after))
			}
		}},
		{"leaver holds a grapple", "FR-021", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			e := rosGrapple(t, "c", "d")
			after, events := remove(t, e, "c", combat.RemovalCauseFled)
			if got, want := encKinds(events), []string{"condition_removed", "grapple_ended", "combatant_removed"}; !slices.Equal(got, want) {
				t.Fatalf("events = %v, want %v", got, want)
			}
			end := events[1].(combat.GrappleEventOccurred).Event().(combat.GrappleEnded)
			if !slices.Equal(rosIDs(events[1]), []string{"d", "c"}) || end.Cause() != combat.GrappleEndCauseOutOfRange {
				t.Errorf("GrappleEnded about %v cause %v, want (d, c) out of range [FR-021]", rosIDs(events[1]), end.Cause())
			}
			d := encMember(t, after, "d")
			if encExplicit(d, content.ConditionGrappled) || !encExplicit(d, content.ConditionPoisoned) {
				t.Errorf("d: Grappled kept or other conditions lost [FR-021]")
			}
			checkEnvelopes(t, events[:2])
		}},
		{"leaver is grappled", "FR-021", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			e := rosGrapple(t, "d", "c")
			_, events := remove(t, e, "c", combat.RemovalCauseFled)
			if got, want := encKinds(events), []string{"condition_removed", "grapple_ended", "combatant_removed"}; !slices.Equal(got, want) {
				t.Fatalf("events = %v, want %v", got, want)
			}
			if !slices.Equal(rosIDs(events[1]), []string{"c", "d"}) {
				t.Errorf("GrappleEnded about %v, want (c, d)", rosIDs(events[1]))
			}
		}},
		{"two grapples held, turn order", "FR-021", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			e := rosGrapple(t, "c", "d", "a")
			_, events := remove(t, e, "c", combat.RemovalCauseFled)
			if got, want := encKinds(events), []string{"condition_removed", "grapple_ended", "condition_removed", "grapple_ended", "combatant_removed"}; !slices.Equal(got, want) {
				t.Fatalf("events = %v, want %v", got, want)
			}
			if !slices.Equal(rosIDs(events[0]), []string{"a"}) || !slices.Equal(rosIDs(events[2]), []string{"d"}) {
				t.Errorf("endings for %v then %v, want a then d (turn order)", rosIDs(events[0]), rosIDs(events[2]))
			}
		}},
		{"leaver holds a grapple and is grappled", "FR-021", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			cs := rosBase(t)
			cs[2] = rosM(t, "c", combat.SideB, combat.Grappled{Grappler: cid("a")})
			cs[3] = rosM(t, "d", combat.SideB, combat.Grappled{Grappler: cid("c")})
			_, events := remove(t, rosStart(t, 5, cs...), "c", combat.RemovalCauseFled)
			if len(events) != 5 || !slices.Equal(rosIDs(events[0]), []string{"c"}) || !slices.Equal(rosIDs(events[2]), []string{"d"}) {
				t.Errorf("events = %v about %v, want the leaver's own grapple first, then d's [FR-021]", encKinds(events), rosIDs(events[0]))
			}
		}},
		{"Charmed by the leaver", "FR-022", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			cs := rosBase(t)
			cs[3] = rosM(t, "d", combat.SideB, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: cid("c")}})
			e := rosStart(t, 5, cs...)
			after, events := remove(t, e, "c", combat.RemovalCauseFled)
			if !slices.Equal(encKinds(events), []string{"combatant_removed"}) || !encMember(t, after, "d").Equal(encMember(t, e, "d")) {
				t.Errorf("events %v: the Charmed of d must be kept and give no event [FR-022]", encKinds(events))
			}
		}},
		{"spent Reaction, Dodge, readied action", "FR-016, FR-017, FR-007, FR-048", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			e := rosBusyEnc(t)
			if !e.Tracker().ReactionSpent(cid("c")) || !e.Tracker().IsDodging(cid("c")) {
				t.Fatalf("fixture: c holds no spent Reaction or Dodge")
			}
			after, _ := remove(t, e, "c", combat.RemovalCauseFled)
			if _, ok := after.Tracker().ReadiedActionOf(cid("d")); !ok {
				t.Errorf("the readied action of another combatant was dropped [FR-048]")
			}
			back, _ := rosOK(t)(rosAdd(t, after, "c", combat.SideB, rosAt(10, rosBefore)))
			if back.Tracker().ReactionSpent(cid("c")) || back.Tracker().IsDodging(cid("c")) {
				t.Errorf("c came back with its old Reaction or Dodge [FR-017]")
			}
			if _, ok := back.Tracker().ReadiedActionOf(cid("c")); ok {
				t.Errorf("c came back with a readied action [FR-017]")
			}
		}},
		{"cause carried", "FR-023, FR-026", "happy", func(t *testing.T) {
			t.Helper()
			for _, cause := range []combat.RemovalCause{combat.RemovalCauseFled, combat.RemovalCauseSurrendered, combat.RemovalCauseDisappeared, combat.RemovalCauseDeparted} {
				after, events := remove(t, rosEnc(t), "c", cause)
				rm, ok := events[len(events)-1].(combat.CombatantRemoved)
				if !ok || rm.Cause() != cause || rm.Side() != combat.SideB || rm.EncounterID() != eid() || rm.Round() != 2 || !slices.Equal(rosIDs(rm), []string{"c"}) {
					t.Errorf("cause %v: event = %#v, want the cause, side B, round 2 about c [FR-026]", cause, rm)
				}
				if rosOrder(after) != "a,b,d" {
					t.Errorf("cause %v changed the result: order %s [FR-023]", cause, rosOrder(after))
				}
			}
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			r.check(t)
		})
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-024 - removal is refused in the order ended, unknown, active, would empty a side: each pair of conditions that hold together reports the earlier.
// Path: unhappy (rule interaction).
func TestRemovalRefusalPrecedence(t *testing.T) {
	t.Parallel()
	ended := rosEnded(t, rosEnc(t))
	two := rosStart(t, 0, rosM(t, "a", combat.SideA), rosM(t, "c", combat.SideB)) // a active; c is the last of side B
	cases := []struct {
		name string
		e    combat.Encounter
		id   string
		want combat.RosterRejection
	}{
		{"ended before unknown", ended, "z", combat.RosterEncounterEndedRejection{}},
		{"ended before active", ended, "b", combat.RosterEncounterEndedRejection{}},
		{"ended before would empty", rosEnded(t, two), "c", combat.RosterEncounterEndedRejection{}},
		{"unknown before active", rosEnc(t), "z", combat.RosterUnknownCombatantRejection{ID: "z"}},
		{"active before would empty", two, "a", combat.RosterCombatantIsActiveRejection{ID: "a"}},
		{"would empty alone", two, "c", combat.RosterWouldEmptySideRejection{Side: combat.SideB}},
	}
	for _, c := range cases {
		tr, err := c.e.RemoveCombatant(cid(c.id), combat.RemovalCauseFled)
		if r := rosNo(t, c.e, tr, err); r != c.want {
			t.Errorf("%s: rejection = %#v, want %#v [FR-024]", c.name, r, c.want)
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-016, FR-037, FR-048, FR-049 - a removal returns a new encounter and leaves the receiver unchanged; it equals the encounter started from the combatants that stay (no record of the leaver and no containment state is kept); readied actions of the others are untouched.
// Path: unhappy (immutability).
func TestRemovalKeepsNoRecordOfTheLeaver(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	before := rosEnc(t)
	after, _ := rosOK(t)(e.RemoveCombatant(cid("c"), combat.RemovalCauseDeparted))
	if !e.Equal(before) {
		t.Errorf("RemoveCombatant changed its receiver [FR-037]")
	}
	rest := []combat.Combatant{rosM(t, "a", combat.SideA), rosM(t, "b", combat.SideA), rosM(t, "d", combat.SideB)}
	fresh := rosStart(t, 4, rest...) // b is active in round 2
	if fresh.Tracker().Active().Combatant != cid("b") || fresh.Tracker().Round() != 2 {
		t.Fatalf("fixture: round %d, active %v", fresh.Tracker().Round(), fresh.Tracker().Active().Combatant)
	}
	if !after.Equal(fresh) {
		t.Errorf("the encounter after the removal differs from one started from the combatants that stay: a record of the leaver is kept [FR-049]")
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-009, FR-010, FR-024 - a removal with an invalid encounter, a blank id or a cause that is not defined is invalid input, never a panic or a result.
// Path: unhappy (invalid input).
func TestRemoveCombatantInvalidInput(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	for name, f := range map[string]func() error{
		"zero encounter": func() error {
			_, err := combat.Encounter{}.RemoveCombatant(cid("c"), combat.RemovalCauseFled)
			return err
		},
		"blank id":         func() error { _, err := e.RemoveCombatant("", combat.RemovalCauseFled); return err },
		"undefined cause":  func() error { _, err := e.RemoveCombatant(cid("c"), 0); return err },
		"cause out of set": func() error { _, err := e.RemoveCombatant(cid("c"), 5); return err },
		"command zero": func() error {
			_, err := combat.NewRemoveCombatantFromEncounterCommand(combat.Encounter{}, combat.ActionLedger{}, cid("c"), combat.RemovalCauseFled)
			return err
		},
		"command blank id": func() error {
			_, err := combat.NewRemoveCombatantFromEncounterCommand(e, combat.ActionLedger{}, "", combat.RemovalCauseFled)
			return err
		},
		"command no cause": func() error {
			_, err := combat.NewRemoveCombatantFromEncounterCommand(e, combat.ActionLedger{}, cid("c"), 0)
			return err
		},
		"reposition zero":   func() error { _, err := combat.Encounter{}.RepositionCombatant(cid("c"), rosAfter("a")); return err },
		"reposition blank":  func() error { _, err := e.RepositionCombatant("", rosAfter("a")); return err },
		"reposition no pos": func() error { _, err := e.RepositionCombatant(cid("c"), nil); return err },
		"reposition self":   func() error { _, err := e.RepositionCombatant(cid("c"), rosAfter("c")); return err },
		"command self": func() error {
			_, err := combat.NewRepositionCombatantInEncounterCommand(e, cid("c"), rosAfter("c"))
			return err
		},
		"command no pos": func() error { _, err := combat.NewRepositionCombatantInEncounterCommand(e, cid("c"), nil); return err },
		"command zero enc": func() error {
			_, err := combat.NewRepositionCombatantInEncounterCommand(combat.Encounter{}, cid("c"), rosAfter("a"))
			return err
		},
		"command blank": func() error {
			_, err := combat.NewRepositionCombatantInEncounterCommand(e, "", rosAfter("a"))
			return err
		},
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: per case (table 4.4a: FR-003, FR-028 to FR-032)
// Path: per case.
func TestRepositionCombatantCases(t *testing.T) {
	t.Parallel()
	move := func(t *testing.T, e combat.Encounter, id string, pos combat.RosterPosition) (combat.Encounter, combat.CombatantRepositioned) {
		t.Helper()
		after, events := rosOK(t)(e.RepositionCombatant(cid(id), pos))
		if len(events) != 1 {
			t.Fatalf("events = %v, want exactly one CombatantRepositioned", encKinds(events))
		}
		rep, ok := events[0].(combat.CombatantRepositioned)
		if !ok {
			t.Fatalf("event = %#v, want CombatantRepositioned", events[0])
		}
		rosHolds(t, after)
		return after, rep
	}
	refuse := func(t *testing.T, e combat.Encounter, id string, pos combat.RosterPosition) combat.RosterRejection {
		t.Helper()
		tr, err := e.RepositionCombatant(cid(id), pos)
		return rosNo(t, e, tr, err)
	}
	rows := []struct {
		name, requirement, path string
		check                   func(t *testing.T)
	}{
		{"mount after rider (active)", "FR-028, FR-031", "happy", func(t *testing.T) {
			t.Helper()
			after, rep := move(t, rosEnc(t), "d", rosAfter("b"))
			if rosOrder(after) != "a,b,d,c" || rosCountOf(t, after, "d") != 15 || rep.Count() != 15 || !slices.Equal(rep.PrecededBy(), []combat.CombatantID{"b"}) ||
				rep.PreviousCount() != 5 || !slices.Equal(rep.PreviousPrecededBy(), []combat.CombatantID{"c"}) {
				t.Errorf("order %s, event %#v, want d after b at 15 (was 5 after c)", rosOrder(after), rep)
			}
			if got := rosTurns(t, after, 2); !slices.Equal(got, []string{"d", "c"}) {
				t.Errorf("next turns %v, want d right after b [FR-031]", got)
			}
		}},
		{"back to its own count", "FR-028", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			moved, _ := move(t, rosEnc(t), "d", rosAfter("b"))
			after, rep := move(t, moved, "d", rosAt(5, rosBehind))
			if rosOrder(after) != "a,b,c,d" || rosCountOf(t, after, "d") != 5 || rep.PreviousCount() != 15 {
				t.Errorf("order %s, count %d, want a,b,c,d at 5", rosOrder(after), rosCountOf(t, after, "d"))
			}
		}},
		{"no change", "FR-032", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			e := rosEnc(t)
			after, rep := move(t, e, "c", rosAt(10, rosBefore))
			if !after.Equal(e) || rep.Count() != rep.PreviousCount() || !slices.Equal(rep.PrecededBy(), rep.PreviousPrecededBy()) {
				t.Errorf("a no-op move changed the encounter or reported different previous values [FR-032]")
			}
		}},
		{"moves across the active combatant", "FR-031", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			after, _ := move(t, rosEnc(t), "a", rosAt(3, rosBehind))
			if rosOrder(after) != "b,c,d,a" || after.Tracker().Active().Combatant != cid("b") || after.Tracker().Round() != 2 {
				t.Errorf("order %s, active %v, want b,c,d,a with b still active in round 2", rosOrder(after), after.Tracker().Active().Combatant)
			}
			if got := rosTurns(t, after, 3); !slices.Equal(got, []string{"c", "d", "a"}) {
				t.Errorf("next turns %v: the engine follows the order, so a acts again this round [FR-031]", got)
			}
		}},
		{"subject is active", "FR-029", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "b", rosAfter("a")); r != (combat.RosterCombatantIsActiveRejection{ID: "b"}) {
				t.Errorf("rejection = %#v, want RosterCombatantIsActiveRejection{b}", r)
			}
		}},
		{"anchor is the active combatant", "FR-029", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			after, _ := move(t, rosEnc(t), "a", rosAfter("b"))
			if rosOrder(after) != "b,a,c,d" {
				t.Errorf("order %s, want an active anchor allowed", rosOrder(after))
			}
		}},
		{"unknown subject", "FR-029", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "z", rosAfter("a")); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
				t.Errorf("rejection = %#v", r)
			}
		}},
		{"unknown anchor", "FR-029", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "c", rosAfter("z")); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
				t.Errorf("rejection = %#v", r)
			}
		}},
		{"unknown subject and anchor", "FR-029", "unhappy (invalid input)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "z", rosAfter("y")); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
				t.Errorf("rejection = %#v, want the subject first", r)
			}
		}},
		{"unknown anchor and active subject", "FR-029", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnc(t), "b", rosAfter("z")); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
				t.Errorf("rejection = %#v, want the unknown anchor before the active subject", r)
			}
		}},
		{"keeps state", "FR-030", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			e := rosBusyEnc(t) // d is active; c holds a spent Reaction and Dodge
			after, _ := move(t, e, "c", rosAfter("a"))
			if !after.Tracker().ReactionSpent(cid("c")) || !after.Tracker().IsDodging(cid("c")) {
				t.Errorf("c lost its spent Reaction or Dodge in the move [FR-030]")
			}
			pb, _ := e.Tracker().ProfileOf(cid("c"))
			if pa, _ := after.Tracker().ProfileOf(cid("c")); pa != pb {
				t.Errorf("c lost its speeds [FR-030]")
			}
			if _, ok := after.Tracker().ReadiedActionOf(cid("d")); !ok || !encMember(t, after, "c").Equal(encMember(t, e, "c")) {
				t.Errorf("another entry or the combatant itself changed [FR-028]")
			}
		}},
		{"ended", "FR-029", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			if r := refuse(t, rosEnded(t, rosEnc(t)), "c", rosAfter("a")); r != (combat.RosterEncounterEndedRejection{}) {
				t.Errorf("rejection = %#v", r)
			}
		}},
		{"subject in a tie group, before", "FR-003, FR-032", "unhappy (boundary)", func(t *testing.T) {
			t.Helper()
			after, rep := move(t, rosEnc(t), "a", rosAt(15, rosBefore))
			if rosOrder(after) != "a,b,c,d" || rosCountOf(t, after, "a") != 15 || rep.Count() != rep.PreviousCount() || len(rep.PrecededBy()) != 0 || len(rep.PreviousPrecededBy()) != 0 {
				t.Errorf("order %s, event %#v, want unchanged with equal previous values", rosOrder(after), rep)
			}
		}},
		{"subject in a tie group, after", "FR-003, FR-028", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			after, _ := move(t, rosEnc(t), "a", rosAt(15, rosBehind))
			if rosOrder(after) != "b,a,c,d" {
				t.Errorf("order %s, want b,a,c,d: a is placed against b only", rosOrder(after))
			}
		}},
		{"other subject joins a tie group, before", "FR-003, FR-028", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			after, _ := move(t, rosEnc(t), "c", rosAt(15, rosBefore))
			if rosOrder(after) != "c,a,b,d" || rosCountOf(t, after, "c") != 15 {
				t.Errorf("order %s, want c,a,b,d", rosOrder(after))
			}
		}},
		{"other subject joins a tie group, after", "FR-003, FR-028", "unhappy (rule interaction)", func(t *testing.T) {
			t.Helper()
			after, _ := move(t, rosEnc(t), "c", rosAt(15, rosBehind))
			if rosOrder(after) != "a,b,c,d" || rosCountOf(t, after, "c") != 15 {
				t.Errorf("order %s, want a,b,c,d at 15", rosOrder(after))
			}
		}},
		{"dead anchor", "FR-028", "unhappy (state)", func(t *testing.T) {
			t.Helper()
			cs := rosBase(t)
			cs[0] = encC(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath))
			after, _ := move(t, rosStart(t, 5, cs...), "d", rosAfter("a"))
			if rosOrder(after) != "a,d,b,c" || rosCountOf(t, after, "d") != 15 {
				t.Errorf("order %s, want a,d,b,c with d at 15", rosOrder(after))
			}
		}},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			r.check(t)
		})
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-029, FR-037 - a reposition refuses in the order ended, unknown subject, unknown anchor, active subject, and returns a new encounter leaving the receiver unchanged.
// Path: unhappy (immutability).
func TestRepositionPrecedenceAndImmutability(t *testing.T) {
	t.Parallel()
	e := rosEnc(t)
	before := rosEnc(t)
	ended := rosEnded(t, e)
	for name, c := range map[string]struct {
		e    combat.Encounter
		id   string
		pos  combat.RosterPosition
		want combat.RosterRejection
	}{
		"ended before unknown":         {ended, "z", rosAfter("y"), combat.RosterEncounterEndedRejection{}},
		"ended before active":          {ended, "b", rosAfter("a"), combat.RosterEncounterEndedRejection{}},
		"subject before active anchor": {e, "z", rosAfter("b"), combat.RosterUnknownCombatantRejection{ID: "z"}},
	} {
		tr, err := c.e.RepositionCombatant(cid(c.id), c.pos)
		if r := rosNo(t, c.e, tr, err); r != c.want {
			t.Errorf("%s: rejection = %#v, want %#v", name, r, c.want)
		}
	}
	if _, _ = rosOK(t)(e.RepositionCombatant(cid("d"), rosAfter("a"))); !e.Equal(before) {
		t.Errorf("RepositionCombatant changed its receiver [FR-037]")
	}
}
