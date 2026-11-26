package combat_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// rosTr is the method set of the unexported transition the roster operations of Encounter return.
type rosTr interface {
	Encounter() combat.Encounter
	Events() []combat.Event
	Rejection() combat.RosterRejection
}

// rosCounts are the Initiative counts of the base order A, B, C, D of the spec (table 4.1a): A and B
// tie at 15 with A first, then C at 10 and D at 5.
var rosCounts = map[string]combat.InitiativeCount{"a": 15, "b": 15, "c": 10, "d": 5}

// rosM is a healthy combatant of the side with the conditions.
func rosM(tb testing.TB, id string, side combat.Side, conds ...combat.Condition) combat.Combatant {
	tb.Helper()
	return encC(tb, id, side, pcStatus(tb, 20, 20, 0), conds...)
}

// rosProfile is the profile of the id: Medium, walk 30.
func rosProfile(tb testing.TB, id string) combat.CombatantProfile {
	tb.Helper()
	return trkProfile(tb, cid(id), content.SizeMedium, trkSpeeds(tb, 30))
}

// rosBase is A and B on side A and C and D on side B, in the order of the spec.
func rosBase(tb testing.TB) []combat.Combatant {
	tb.Helper()
	return []combat.Combatant{rosM(tb, "a", combat.SideA), rosM(tb, "b", combat.SideA), rosM(tb, "c", combat.SideB), rosM(tb, "d", combat.SideB)}
}

// rosEstablish establishes the order of the combatants by rosCounts, A before B.
func rosEstablish(tb testing.TB, cs []combat.Combatant) combat.InitiativeOrderEstablished {
	tb.Helper()
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	var tied []combat.CombatantID
	for _, c := range cs {
		id := string(c.Ref().ID)
		standings = append(standings, combat.InitiativeStanding{Combatant: c.Ref().ID, Count: rosCounts[id]})
		profiles = append(profiles, rosProfile(tb, id))
		if rosCounts[id] == 15 {
			tied = append(tied, c.Ref().ID)
		}
	}
	var breaks []combat.TieBreak
	if len(tied) > 1 {
		b, err := combat.NewTieBreak(tied)
		if err != nil {
			tb.Fatalf("NewTieBreak: %v", err)
		}
		breaks = append(breaks, b)
	}
	cmd, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, breaks)
	if err != nil {
		tb.Fatalf("NewEstablishInitiativeOrderCommand: %v", err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	est, ok := res.(combat.InitiativeOrderEstablished)
	if !ok {
		tb.Fatalf("result = %#v, want InitiativeOrderEstablished", res)
	}
	return est
}

// rosStart starts an encounter of the combatants and advances the turn n times.
func rosStart(tb testing.TB, n int, cs ...combat.Combatant) combat.Encounter {
	tb.Helper()
	e, _, err := (combat.Encounter{}).Start(eid(), rosEstablish(tb, cs), cs)
	if err != nil {
		tb.Fatalf("Start: %v", err)
	}
	for range n {
		e, _ = advance(tb, e)
	}
	return e
}

// rosEnc is the base encounter of the spec: A, B, C, D, with B active in round 2.
func rosEnc(tb testing.TB) combat.Encounter {
	tb.Helper()
	e := rosStart(tb, 5, rosBase(tb)...)
	if e.Tracker().Round() != 2 || e.Tracker().Active().Combatant != cid("b") {
		tb.Fatalf("fixture: round %d, active %v, want round 2 with b active", e.Tracker().Round(), e.Tracker().Active().Combatant)
	}
	return e
}

// rosOrder lists the ids of the turn order, comma separated.
func rosOrder(e combat.Encounter) string {
	var ids []string
	for _, s := range e.Tracker().Order().Standings() {
		ids = append(ids, string(s.Combatant))
	}
	return strings.Join(ids, ",")
}

// rosList lists the ids of the combatants list of the encounter, comma separated.
func rosList(e combat.Encounter) string {
	var ids []string
	for _, c := range e.Combatants() {
		ids = append(ids, string(c.Ref().ID))
	}
	return strings.Join(ids, ",")
}

// rosCountOf returns the count of the id in the order of the encounter.
func rosCountOf(tb testing.TB, e combat.Encounter, id string) combat.InitiativeCount {
	tb.Helper()
	s, err := e.Tracker().Order().StandingOf(cid(id))
	if err != nil {
		tb.Fatalf("StandingOf(%s): %v", id, err)
	}
	return s.Count
}

// rosAfter and rosAt build the two positions.
func rosAfter(id string) combat.RosterPosition { return combat.RosterAfterCombatant{Anchor: cid(id)} }

func rosAt(count combat.InitiativeCount, tie combat.TiePlacement) combat.RosterPosition {
	return combat.RosterAtInitiative{Count: count, Tie: tie}
}

const (
	rosBefore = combat.TiePlacementBeforeTied
	rosBehind = combat.TiePlacementAfterTied
)

// rosAdd adds the combatant x of the side at the position.
func rosAdd(tb testing.TB, e combat.Encounter, id string, side combat.Side, pos combat.RosterPosition) (rosTr, error) {
	tb.Helper()
	return e.AddCombatant(rosM(tb, id, side), rosProfile(tb, id), pos, combat.JoinReasonArrived)
}

// rosOK returns a function that takes the two results of a roster operation and returns the new encounter and events of an applied one.
func rosOK(tb testing.TB) func(rosTr, error) (combat.Encounter, []combat.Event) {
	tb.Helper()
	return func(tr rosTr, err error) (combat.Encounter, []combat.Event) {
		tb.Helper()
		if err != nil {
			tb.Fatalf("roster operation error: %v", err)
		}
		if r := tr.Rejection(); r != nil {
			tb.Fatalf("rejected with %#v, want applied", r)
		}
		return tr.Encounter(), tr.Events()
	}
}

// rosNo returns the rejection of a refused roster operation and asserts the encounter came back equal with no event.
func rosNo(tb testing.TB, before combat.Encounter, tr rosTr, err error) combat.RosterRejection {
	tb.Helper()
	if err != nil {
		tb.Fatalf("roster operation error: %v", err)
	}
	r := tr.Rejection()
	if r == nil {
		tb.Fatalf("applied, want a rejection")
	}
	if !tr.Encounter().Equal(before) || len(tr.Events()) != 0 {
		tb.Errorf("a refusal must carry the unchanged encounter and no events [FR-009]")
	}
	return r
}

// rosHolds asserts the counts along the order do not increase and each id occurs once, with the combatants list in the same order (FR-004, FR-037).
func rosHolds(tb testing.TB, e combat.Encounter) {
	tb.Helper()
	if e.IsZero() {
		tb.Errorf("the encounter is not valid [FR-004]")
	}
	seen := map[combat.CombatantID]bool{}
	var prev combat.InitiativeCount
	for i, s := range e.Tracker().Order().Standings() {
		if i > 0 && s.Count > prev {
			tb.Errorf("count %d follows %d: the counts increase along the order [FR-004]", s.Count, prev)
		}
		if seen[s.Combatant] {
			tb.Errorf("%v occurs twice in the order [FR-004]", s.Combatant)
		}
		seen[s.Combatant], prev = true, s.Count
	}
	if got, want := rosList(e), rosOrder(e); got != want {
		tb.Errorf("combatants list %s, order %s: they differ [FR-037]", got, want)
	}
}

// rosBig builds an encounter of n combatants m0000, m0001, ... alternating sides, with distinct counts, m0000 active, and
// the ids that grapple holds Grappled by the grappler of the first.
func rosBig(tb testing.TB, n int, grappledBy func(i int) (string, bool)) combat.Encounter {
	tb.Helper()
	est, cs := rosBigParts(tb, n, grappledBy)
	e, _, err := startAnySize(eid(), est, cs)
	if err != nil {
		tb.Fatalf("Start: %v", err)
	}
	return e
}

// rosBigParts is the established order and the combatants rosBig starts from.
func rosBigParts(tb testing.TB, n int, grappledBy func(i int) (string, bool)) (combat.InitiativeOrderEstablished, []combat.Combatant) {
	tb.Helper()
	cs := make([]combat.Combatant, n)
	standings := make([]combat.InitiativeStanding, n)
	profiles := make([]combat.CombatantProfile, n)
	for i := range n {
		id := fmt.Sprintf("m%04d", i)
		var conds []combat.Condition
		if by, ok := grappledBy(i); ok {
			conds = append(conds, combat.Grappled{Grappler: cid(by)})
		}
		cs[i] = rosM(tb, id, combat.Side(1+i%2), conds...)
		standings[i] = combat.InitiativeStanding{Combatant: cid(id), Count: combat.InitiativeCount(n - i)}
		profiles[i] = rosProfile(tb, id)
	}
	cmd, err := establishCommandAnySize(standings, profiles)
	if err != nil {
		tb.Fatalf("NewEstablishInitiativeOrderCommand: %v", err)
	}
	res, err := combat.NewEstablishInitiativeOrder().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res.(combat.InitiativeOrderEstablished), cs
}

// rosIDs returns the ids of an event's Combatants as strings.
func rosIDs(e combat.Event) []string {
	var out []string
	for _, id := range e.Combatants() {
		out = append(out, string(id))
	}
	return slices.Clone(out)
}

// rosBusyEnc is the base encounter in round 1 with d active, where c holds a spent Reaction, Dodge and a feat use and d
// holds a readied action: the state a removal or a move must keep or drop (FR-016, FR-030, FR-048).
func rosBusyEnc(tb testing.TB) combat.Encounter {
	tb.Helper()
	e := rosStart(tb, 2, rosBase(tb)...) // c is active
	commit := func(step combat.TrackerStep, err error) {
		tb.Helper()
		tracker, events := trkApplied(tb)(step, err)
		tr, err := e.ApplyTurnChange(tracker, events)
		e, _ = encOK(tb, tr, err)
	}
	commit(e.Tracker().TakeAction(cid("c"), trkAvail, trkDodge))
	commit(e.Tracker().RecordFeatUse(cid("c"), "f", character.OnceUntilStartOfYourNextTurn{}))
	commit(e.Tracker().TakeReaction(cid("c"), trkAvail, combat.OpportunityAttackReaction{Provoker: cid("a"), Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen}))
	e, _ = advance(tb, e) // d is active
	move, err := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(tb, 5)})
	if err != nil {
		tb.Fatal(err)
	}
	commit(e.Tracker().TakeAction(cid("d"), trkAvail, combat.ReadyChoice{Action: trkReadied(tb, "t", move)}))
	return e
}

// rosEnded ends the encounter with both sides agreeing.
func rosEnded(tb testing.TB, e combat.Encounter) combat.Encounter {
	tb.Helper()
	return runEnd(tb, e, combat.EndReasonMutualAgreement{}).(combat.EndEncounterApplied).Encounter()
}

// startAnySize starts an encounter through Start, or through the uncapped start of export_test.go when
// there are more combatants than the cap of 256, which only the tests of the cap of 4096 events need.
func startAnySize(id combat.EncounterID, est combat.InitiativeOrderEstablished, cs []combat.Combatant) (combat.Encounter, []combat.Event, error) {
	if len(cs) > 256 {
		return combat.StartOverTheCap(id, est, cs)
	}
	return (combat.Encounter{}).Start(id, est, cs)
}

// establishCommandAnySize builds the establish command through its constructor, or through the uncapped one of
// export_test.go when there are more standings than the cap of 256.
func establishCommandAnySize(standings []combat.InitiativeStanding, profiles []combat.CombatantProfile) (combat.EstablishInitiativeOrderCommand, error) {
	if len(standings) > 256 {
		return combat.EstablishCommandOverTheCap(standings, profiles), nil
	}
	return combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
}
