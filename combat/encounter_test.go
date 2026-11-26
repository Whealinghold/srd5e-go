package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func fourCombatants(tb testing.TB) []combat.Combatant {
	tb.Helper()
	return []combat.Combatant{
		encPC(tb, "a", combat.SideA, 20, 20), encPC(tb, "b", combat.SideB, 20, 20),
		encPC(tb, "c", combat.SideA, 20, 20), encPC(tb, "d", combat.SideB, 20, 20),
	}
}

func ids(es []combat.Combatant) []string {
	var out []string
	for _, c := range es {
		out = append(out, string(c.Ref().ID))
	}
	return out
}

// Requirement: doc/specs/encounter-aggregate.md FR-007, FR-008, FR-053 - starting stores the combatants in the tracker's turn order, whatever order they were passed in, and produces EncounterStarted then the established events wrapped.
// Path: happy.
func TestStartStoresCombatantsInTurnOrderAndProducesTheStartEvents(t *testing.T) {
	t.Parallel()
	four := fourCombatants(t)
	est := encEstablished(t, "c", "a", "d", "b")
	e, events, err := (combat.Encounter{}).Start(eid(), est, four)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if got := ids(e.Combatants()); !slices.Equal(got, []string{"c", "a", "d", "b"}) {
		t.Errorf("combatants = %v, want the tracker order c a d b [FR-007]", got)
	}
	if want := []string{"encounter_started", "tracker_round_started", "tracker_turn_started"}; !slices.Equal(encKinds(events), want) {
		t.Errorf("events = %v, want %v [FR-008]", encKinds(events), want)
	}
	started := events[0].(combat.EncounterStarted)
	if started.EncounterID() != eid() || started.Round() != 1 {
		t.Errorf("EncounterStarted carries %v round %d", started.EncounterID(), started.Round())
	}
	combattest.CheckEvent(t, started, "encounter_started", cid("c"), cid("a"), cid("d"), cid("b"))
	combattest.CheckEvent(t, events[2], "tracker_turn_started", cid("c"))
	if e.ID() != eid() || e.Status() != (combat.EncounterInProgress{}) || e.IsZero() || !e.Tracker().Equal(est.Tracker()) {
		t.Errorf("the encounter does not hold the id, status and tracker it was started with")
	}
	if r := e.Tracker().Round(); r != 1 || e.Tracker().Active().Combatant != cid("c") {
		t.Errorf("round %d active %v, want round 1 with c active", r, e.Tracker().Active().Combatant)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-009, FR-053 - the tracker order C, A, B is kept; the same start with the combatants passed as A, B, C stores C, A, B and equals the one built from C, A, B.
// Path: unhappy (rule interaction).
func TestStartKeepsTheTrackerOrderWhateverTheInputOrder(t *testing.T) {
	t.Parallel()
	a, b, c := encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20), encPC(t, "c", combat.SideA, 20, 20)
	est := encEstablished(t, "c", "a", "b")
	fromCAB, _, err1 := (combat.Encounter{}).Start(eid(), est, []combat.Combatant{c, a, b})
	fromABC, _, err2 := (combat.Encounter{}).Start(eid(), est, []combat.Combatant{a, b, c})
	if err1 != nil || err2 != nil {
		t.Fatalf("Start: %v, %v", err1, err2)
	}
	if got := ids(fromABC.Combatants()); !slices.Equal(got, []string{"c", "a", "b"}) {
		t.Errorf("combatants = %v, want c a b", got)
	}
	if !fromABC.Equal(fromCAB) {
		t.Errorf("two starts differing only in the input order are not equal [FR-053]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-010 - a tracker whose order came from a tie-break is accepted as is and keeps its relative order.
// Path: unhappy (boundary).
func TestStartAcceptsATrackerWhoseOrderCameFromATieBreak(t *testing.T) {
	t.Parallel()
	same := combat.InitiativeCount(10)
	standings := []combat.InitiativeStanding{{Combatant: cid("a"), Count: same}, {Combatant: cid("b"), Count: same}}
	profiles := []combat.CombatantProfile{trkProfile(t, cid("a"), 3, trkSpeeds(t, 30)), trkProfile(t, cid("b"), 3, trkSpeeds(t, 30))}
	tb, err := combat.NewTieBreak([]combat.CombatantID{cid("b"), cid("a")})
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, []combat.TieBreak{tb})
	if err != nil {
		t.Fatal(err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	est, ok := res.(combat.InitiativeOrderEstablished)
	if err != nil || !ok {
		t.Fatalf("establish = %#v, %v", res, err)
	}
	e, _, err := (combat.Encounter{}).Start(eid(), est, []combat.Combatant{encPC(t, "a", combat.SideA, 5, 5), encPC(t, "b", combat.SideB, 5, 5)})
	if err != nil || !slices.Equal(ids(e.Combatants()), []string{"b", "a"}) || e.Tracker().Active().Combatant != cid("b") {
		t.Errorf("Start = %v, %v, want the tie-break order b a with b active", ids(e.Combatants()), err)
	}
}

// Requirement: per case
// Path: per case.
func TestStartRejectsInputThatBreaksTheInvariants(t *testing.T) {
	t.Parallel()
	a, b, c := encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20), encPC(t, "c", combat.SideA, 20, 20)
	cases := []struct {
		name, requirement, path string
		order                   []string
		list                    []combat.Combatant
		want                    []combat.EncounterProblem
	}{
		{"no combatants", encSpec + "FR-011, FR-013", "unhappy (empty and minimal)", []string{"a"}, nil,
			[]combat.EncounterProblem{combat.NoCombatantsProblem{}, combat.TurnOrderEntryWithoutCombatantProblem{ID: cid("a")}}},
		{"empty, not nil", encSpec + "FR-011", "unhappy (empty and minimal)", []string{"a"}, []combat.Combatant{},
			[]combat.EncounterProblem{combat.NoCombatantsProblem{}, combat.TurnOrderEntryWithoutCombatantProblem{ID: cid("a")}}},
		{"one side only", encSpec + "FR-011", "unhappy (empty and minimal)", []string{"a", "c"}, []combat.Combatant{a, c},
			[]combat.EncounterProblem{combat.MissingSideProblem{Side: combat.SideB}}},
		{"side A missing", encSpec + "FR-011", "unhappy (empty and minimal)", []string{"b"}, []combat.Combatant{b},
			[]combat.EncounterProblem{combat.MissingSideProblem{Side: combat.SideA}}},
		{"duplicate on one side", encSpec + "FR-012", "unhappy (invalid input)", []string{"a", "b"}, []combat.Combatant{a, a, b},
			[]combat.EncounterProblem{combat.DuplicateCombatantIDProblem{ID: cid("a")}}},
		{"duplicate across sides", encSpec + "FR-012", "unhappy (invalid input)", []string{"a", "b"}, []combat.Combatant{a, b, encPC(t, "a", combat.SideB, 3, 3)},
			[]combat.EncounterProblem{combat.DuplicateCombatantIDProblem{ID: cid("a")}}},
		{"combatant not in the order", encSpec + "FR-013", "unhappy (invalid input)", []string{"a", "b"}, []combat.Combatant{a, b, c},
			[]combat.EncounterProblem{combat.CombatantNotInTurnOrderProblem{ID: cid("c")}}},
		{"order entry without a combatant", encSpec + "FR-013", "unhappy (invalid input)", []string{"a", "b", "c"}, []combat.Combatant{a, b},
			[]combat.EncounterProblem{combat.TurnOrderEntryWithoutCombatantProblem{ID: cid("c")}}},
		{"several at once, in the order found", encSpec + "FR-013", "unhappy (rule interaction)", []string{"a", "z"}, []combat.Combatant{a, a, c},
			[]combat.EncounterProblem{combat.MissingSideProblem{Side: combat.SideB}, combat.DuplicateCombatantIDProblem{ID: cid("a")},
				combat.CombatantNotInTurnOrderProblem{ID: cid("c")}, combat.TurnOrderEntryWithoutCombatantProblem{ID: cid("z")}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e, events, err := (combat.Encounter{}).Start(eid(), encEstablished(t, c.order...), c.list)
			if !errors.Is(err, srderr.ErrInvalidInput) || !e.IsZero() || events != nil {
				t.Fatalf("Start = %v, %v, %v, want ErrInvalidInput, a zero encounter and no events [%s]", e, events, err, c.requirement)
			}
			if got := combat.ProblemsOf(err); !deepEqual(got, c.want) {
				t.Errorf("ProblemsOf = %#v, want %#v [%s]", got, c.want, c.requirement)
			}
			j, ok := err.(interface{ Unwrap() []error })
			if !ok || len(j.Unwrap()) != len(c.want) {
				t.Fatalf("Start error is not the join of one error per problem [FR-013]")
			}
			for i, each := range j.Unwrap() {
				var pe *combat.EncounterProblemError
				if !errors.As(each, &pe) || !deepEqual(pe.Problem(), c.want[i]) || pe.Error() == "" || !errors.Is(each, srderr.ErrInvalidInput) {
					t.Errorf("problem %d is not an EncounterProblemError holding %#v", i, c.want[i])
				}
			}
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-013 - a blank id, a zero tracker and a zero Combatant are invalid input that Start reports as an error matching ErrInvalidInput with no typed problem; MustStart panics exactly then and ProblemsOf of nil or of another error is empty.
// Path: unhappy (invalid input).
func TestStartRejectsUntypedInvalidInputAndMustStartPanicsOnlyThen(t *testing.T) {
	t.Parallel()
	ok := []combat.Combatant{encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20)}
	est := encEstablished(t, "a", "b")
	cases := []struct {
		name string
		id   combat.EncounterID
		est  combat.InitiativeOrderEstablished
		list []combat.Combatant
	}{
		{"blank id", "", est, ok},
		{"zero tracker", eid(), combat.InitiativeOrderEstablished{}, ok},
		{"zero combatant", eid(), est, []combat.Combatant{ok[0], {}, ok[1]}},
	}
	for _, c := range cases {
		_, _, err := (combat.Encounter{}).Start(c.id, c.est, c.list)
		if !errors.Is(err, srderr.ErrInvalidInput) || len(combat.ProblemsOf(err)) != 0 {
			t.Errorf("%s: Start error = %v, want ErrInvalidInput with no problems", c.name, err)
		}
		func() {
			defer func() {
				if e, _ := recover().(error); !errors.Is(e, srderr.ErrInvalidInput) {
					t.Errorf("%s: MustStart panicked with %v, want the Start error", c.name, e)
				}
			}()
			(combat.Encounter{}).MustStart(c.id, c.est, c.list)
		}()
	}
	e, events := (combat.Encounter{}).MustStart(eid(), est, ok)
	if e.IsZero() || len(events) == 0 {
		t.Errorf("MustStart of valid input = %v, %v", e, events)
	}
	if combat.ProblemsOf(nil) != nil || combat.ProblemsOf(errors.New("other")) != nil {
		t.Errorf("ProblemsOf of an error without problems is not empty")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-014, FR-052 - the encounter does not change if the caller later mutates the list passed to Start or the slice Combatants returns.
// Path: unhappy (immutability).
func TestStartTakesItsOwnCopyAndAccessorsReturnCopies(t *testing.T) {
	t.Parallel()
	list := []combat.Combatant{encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20)}
	e, _, err := (combat.Encounter{}).Start(eid(), encEstablished(t, "a", "b"), list)
	if err != nil {
		t.Fatal(err)
	}
	want := encABCless(t)
	list[0], list[1] = combat.Combatant{}, combat.Combatant{}
	got := e.Combatants()
	got[0] = combat.Combatant{}
	if !e.Equal(want) {
		t.Errorf("the encounter changed when the input list or a returned slice was mutated")
	}
	if c, ok := e.Find(cid("zzz")); ok || !c.IsZero() {
		t.Errorf("Find of an unknown id = %v, %v", c, ok)
	}
}

func encABCless(tb testing.TB) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encPC(tb, "a", combat.SideA, 20, 20), encPC(tb, "b", combat.SideB, 20, 20))
	return e
}

// Requirement: doc/specs/encounter-aggregate.md FR-018, FR-099 - when the first combatant is Dying the start ends with DyingCombatantTurnStarted, and the per-turn Death Saving Throw marker of that combatant is cleared at start.
// Path: unhappy (rule interaction).
func TestStartWithADyingFirstCombatant(t *testing.T) {
	t.Parallel()
	dyingA := encC(t, "a", combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}).WithDeathSavingThrowMade(true)
	e, events := encStart(t, dyingA, encPC(t, "b", combat.SideB, 20, 20))
	want := []string{"encounter_started", "tracker_round_started", "tracker_turn_started", "dying_combatant_turn_started"}
	if !slices.Equal(encKinds(events), want) {
		t.Errorf("events = %v, want %v", encKinds(events), want)
	}
	combattest.CheckEvent(t, events[3], "dying_combatant_turn_started", cid("a"))
	if encMember(t, e, "a").DeathSavingThrowMade() {
		t.Errorf("the marker of the combatant whose turn started was not cleared [FR-099]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-013 - Start never panics and returns the joined problems or a valid encounter for any combatant list.
// Path: unhappy (invalid input).
func FuzzEncounterStart(f *testing.F) {
	for _, s := range []string{"", "abc", "aab", "zzz", "\x00\xff", "aabbcc"} {
		f.Add(s)
	}
	est := encEstablished(f, "a", "b", "c")
	f.Fuzz(func(t *testing.T, data string) {
		var list []combat.Combatant
		for _, b := range []byte(data) {
			side := combat.Side(b>>6) + 1
			ref := combat.CombatantRef{ID: combat.CombatantID(string(rune('a' + b%5))), Kind: combat.MonsterCombatant{}}
			c, err := combat.NewCombatant(ref, side, pcStatus(t, 5, 5, 0), combat.ConditionSet{})
			if err != nil {
				continue
			}
			list = append(list, c)
		}
		e, _, err := (combat.Encounter{}).Start(eid(), est, list)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("Start error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if e.IsZero() || len(e.Combatants()) != 3 {
			t.Fatalf("a valid start gave %v", ids(e.Combatants()))
		}
	})
}

// Requirement: doc/specs/encounter-aggregate.md FR-009, FR-083, plan-phase-4.md P14 - the root validates the parts of a tracker and of a Combatant (encounter_validate_internal_test.go) but not the consistency between a Combatant's life state, markers and conditions: a Dying combatant without Unconscious and a knocked-out marker without Unconscious are accepted as the host's input (this pins the limit of the check).
// Path: unhappy (invalid input).
func TestStartDoesNotRevalidateWhatTheSpecLeavesToTheHost(t *testing.T) {
	t.Parallel()
	dyingNoUnconscious := encC(t, "a", combat.SideA, dyingStatus(t, 20, 0, 0))
	markedNoUnconscious := encPC(t, "b", combat.SideB, 5, 5).WithKnockedOut(true).WithDeathSavingThrowMade(true)
	e, events, err := (combat.Encounter{}).Start(eid(), encEstablished(t, "a", "b"), []combat.Combatant{dyingNoUnconscious, markedNoUnconscious})
	if err != nil || e.IsZero() {
		t.Fatalf("Start = %v, %v, want the inconsistent but valid parts accepted [FR-083]", e, err)
	}
	if !slices.Contains(encKinds(events), "dying_combatant_turn_started") {
		t.Errorf("a Dying combatant without Unconscious still produces its turn fact [FR-018: the rule reads LifeState]")
	}
	if !encMember(t, e, "b").KnockedOut() || !encMember(t, e, "b").DeathSavingThrowMade() {
		t.Errorf("the markers of a combatant whose turn did not start must be kept as given")
	}
}
