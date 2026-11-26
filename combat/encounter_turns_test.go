package combat_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
)

// advance advances e by one turn with no updates and returns the new encounter and events.
func advance(tb testing.TB, e combat.Encounter) (combat.Encounter, []combat.Event) {
	tb.Helper()
	tr, err := e.AdvanceTurn(combat.TurnAdvance{})
	return encOK(tb, tr, err)
}

// Requirement: doc/specs/encounter-aggregate.md table 4.3.1, FR-016, FR-017 - turn and round arithmetic with envelopes: the wrap to round 2, one past it, two full rounds and the minimal encounter, with the round of each wrapped event.
// Path: per case.
func TestAdvanceTurnArithmetic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		combatants              int
		advances                int
		active                  string
		round                   int
		last                    []string
		lastRounds              []int
	}{
		{"first advance", "FR-016; p. 13", "happy", 3, 1, "b", 1, []string{"tracker_turn_ended", "tracker_turn_started"}, []int{1, 1}},
		{"second advance", "FR-016; p. 13", "happy", 3, 2, "c", 1, []string{"tracker_turn_ended", "tracker_turn_started"}, []int{1, 1}},
		{"wrap to round 2", "FR-017; p. 13", "unhappy (boundary)", 3, 3, "a", 2, []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started"}, []int{1, 2, 2}},
		{"one past the wrap", "FR-017; p. 13", "unhappy (boundary)", 3, 4, "b", 2, []string{"tracker_turn_ended", "tracker_turn_started"}, []int{2, 2}},
		{"two full rounds", "FR-017; p. 13", "unhappy (boundary)", 3, 6, "a", 3, []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started"}, []int{2, 3, 3}},
		{"minimal encounter wrap", "FR-017; p. 13", "unhappy (empty and minimal)", 2, 2, "a", 2, []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started"}, []int{1, 2, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := encABC(t)
			if c.combatants == 2 {
				e = encABCless(t)
			}
			var events []combat.Event
			for range c.advances {
				before := e
				e, events = advance(t, e)
				if before.Tracker().Equal(e.Tracker()) {
					t.Fatalf("the tracker did not change [FR-016]")
				}
			}
			if e.Tracker().Active().Combatant != cid(c.active) || e.Tracker().Round() != c.round {
				t.Errorf("active %v round %d, want %s round %d [%s]", e.Tracker().Active().Combatant, e.Tracker().Round(), c.active, c.round, c.requirement)
			}
			if !slices.Equal(encKinds(events), c.last) {
				t.Errorf("events = %v, want %v [%s]", encKinds(events), c.last, c.requirement)
			}
			for i, ev := range events {
				env, ok := ev.(combat.TurnEventOccurred)
				if !ok || env.Round() != c.lastRounds[i] || env.EncounterID() != eid() {
					t.Errorf("event %d = %#v, want a TurnEventOccurred of round %d", i, ev, c.lastRounds[i])
				}
			}
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-015, FR-020 - an advance never changes the size or order of the combatants and does not skip a dead, Unconscious or 0 Hit Point combatant: every combatant gets a TrackerTurnStarted each round.
// Path: unhappy (rule interaction).
func TestAdvanceTurnKeepsTheOrderAndDoesNotSkipTheDead(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "b", combat.SideB, deadStatus(t, combat.MonsterDeath)),
		encC(t, "c", combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}))
	want := ids(e.Combatants())
	started := []combat.CombatantID{}
	for range 6 {
		var events []combat.Event
		e, events = advance(t, e)
		for _, ev := range events {
			if s, ok := ev.(combat.TurnEventOccurred); ok && s.Kind() == "tracker_turn_started" {
				started = append(started, s.Combatants()...)
			}
		}
		if got := ids(e.Combatants()); !slices.Equal(got, want) {
			t.Fatalf("combatants = %v, want %v [FR-015]", got, want)
		}
	}
	if want := []combat.CombatantID{"b", "c", "a", "b", "c", "a"}; !slices.Equal(started, want) {
		t.Errorf("turns started = %v, want %v [FR-020]", started, want)
	}
}

// Requirement: doc/specs/encounter-aggregate.md table 4.3.2, FR-018, FR-019 - DyingCombatantTurnStarted is produced for a Dying combatant whose turn starts, whatever its kind, and for no Alive, Stable or Dead one (dead of Exhaustion included).
// Path: per case.
func TestDyingCombatantTurnStartedMatrix(t *testing.T) {
	t.Parallel()
	monsterDying := status(t, 0, 7, 0, func() combat.Dying { d, _ := combat.NewDying(0, 0); return d }(), combat.FallsUnconscious, combat.NoAffinities())
	cases := []struct {
		name, requirement, path string
		hp                      combat.HitPointStatus
		conds                   []combat.Condition
		want                    bool
	}{
		{"player character at 0", "FR-018; p. 17", "happy", dyingStatus(t, 12, 0, 0), condConds(combat.Unconscious{}), true},
		{"player character at 1", "FR-019; p. 17", "unhappy (boundary)", pcStatus(t, 1, 12, 0), nil, false},
		{"stable", "FR-019; p. 188", "unhappy (rule interaction)", status(t, 0, 12, 0, combat.Stable{}, combat.FallsUnconscious, combat.NoAffinities()), condConds(combat.Unconscious{}), false},
		{"dead", "FR-019; p. 180", "unhappy (state)", deadStatus(t, combat.MassiveDamage), nil, false},
		{"monster with DiesInstantly at 0", "FR-019; p. 17", "unhappy (rule interaction)", status(t, 0, 7, 0, dead(combat.MonsterDeath), combat.DiesInstantly, combat.NoAffinities()), nil, false},
		{"monster treated like a character", "FR-018; p. 17", "unhappy (rule interaction)", monsterDying, condConds(combat.Unconscious{}), true},
		{"dead of Exhaustion", "FR-019; p. 181", "unhappy (rule interaction)", deadStatus(t, combat.Exhaustion), condConds(condLevel(6)), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			target := encC(t, "c", combat.SideA, c.hp, c.conds...)
			e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encPC(t, "b", combat.SideB, 20, 20), target)
			e, _ = advance(t, e)
			_, events := advance(t, e)
			got := slices.Contains(encKinds(events), "dying_combatant_turn_started")
			if got != c.want {
				t.Errorf("DyingCombatantTurnStarted = %v, want %v [%s]", got, c.want, c.requirement)
			}
			if c.want {
				if k := encKinds(events); k[len(k)-1] != "dying_combatant_turn_started" {
					t.Errorf("events = %v, want DyingCombatantTurnStarted last [FR-018]", k)
				}
			}
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-018, table 4.3.1 - when a readied action of the Dying combatant expires, ReadiedActionExpired comes before DyingCombatantTurnStarted, which is last, and nothing is inserted between the wrapped tracker events.
// Path: unhappy (rule interaction).
func TestDyingEventComesAfterAReadiedActionExpiry(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), encC(t, "c", combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}), encPC(t, "b", combat.SideB, 20, 20))
	e, _ = advance(t, e) // c is active
	move, err := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 5)})
	if err != nil {
		t.Fatal(err)
	}
	step, err := e.Tracker().TakeAction(cid("c"), trkAvail, combat.ReadyChoice{Action: trkReadied(t, "t", move)})
	tracker, tev := trkApplied(t)(step, err)
	tr, err := e.ApplyTurnChange(tracker, tev)
	e, _ = encOK(t, tr, err)
	e, _ = advance(t, e) // b
	e, _ = advance(t, e) // a
	_, events := advance(t, e)
	want := []string{"tracker_turn_ended", "tracker_turn_started", "readied_action_expired", "dying_combatant_turn_started"}
	if !slices.Equal(encKinds(events), want) {
		t.Errorf("events = %v, want %v [FR-018]", encKinds(events), want)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-099 - the per-turn Death Saving Throw marker is cleared for the combatant whose turn began only, at AdvanceTurn.
// Path: unhappy (rule interaction).
func TestAdvanceTurnClearsTheMarkerOfTheActiveCombatantOnly(t *testing.T) {
	t.Parallel()
	mk := func(id string) combat.Combatant {
		return encC(t, id, combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}).WithDeathSavingThrowMade(true)
	}
	e, _ := encStart(t, mk("a"), mk("b"), encPC(t, "z", combat.SideB, 20, 20))
	if encMember(t, e, "b").DeathSavingThrowMade() != true || encMember(t, e, "a").DeathSavingThrowMade() {
		t.Fatalf("start must clear a's marker only")
	}
	e, _ = advance(t, e)
	if encMember(t, e, "b").DeathSavingThrowMade() || encMember(t, e, "a").DeathSavingThrowMade() {
		t.Errorf("the marker of b was not cleared when its turn began, or a's came back [FR-099]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-085, FR-046, FR-078, table 4.9.1 - AdvanceTurn passes updated speeds and shared-space occupants to the tracker, wraps FellProneInSharedSpace without applying Prone, and reports the smallest unknown id of the updates before the tracker is called.
// Path: unhappy (rule interaction).
func TestAdvanceTurnUpdatesAndUnknownIDs(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	speeds := trkSpeeds(t, 10)
	tr, err := e.AdvanceTurn(combat.TurnAdvance{Updates: []combat.SpeedUpdate{{Combatant: cid("c"), Speeds: speeds}}, SharedSpaceOccupants: []content.CreatureSize{content.SizeMedium}})
	after, events := encOK(t, tr, err)
	want := []string{"tracker_turn_ended", "fell_prone_in_shared_space", "movement_speeds_updated", "tracker_turn_started"}
	if !slices.Equal(encKinds(events), want) {
		t.Errorf("events = %v, want %v [FR-085]", encKinds(events), want)
	}
	if p, _ := after.Tracker().ProfileOf(cid("c")); p.Speeds != speeds {
		t.Errorf("the updated speeds did not reach the tracker")
	}
	if encExplicit(encMember(t, after, "a"), content.ConditionProne) {
		t.Errorf("the encounter applied Prone for FellProneInSharedSpace [FR-085]")
	}
	for _, order := range [][]string{{"z", "b"}, {"b", "z"}} {
		var ups []combat.SpeedUpdate
		for _, id := range order {
			ups = append(ups, combat.SpeedUpdate{Combatant: cid(id), Speeds: speeds})
		}
		tr, err := e.AdvanceTurn(combat.TurnAdvance{Updates: ups})
		if r := encNo(t, e, tr, err); r != (combat.UnknownCombatantRejection{ID: cid("z")}) {
			t.Errorf("rejection = %#v, want the unknown z [FR-078]", r)
		}
	}
	if _, err := e.AdvanceTurn(combat.TurnAdvance{SharedSpaceOccupants: []content.CreatureSize{0}}); err == nil {
		t.Errorf("a size that is not defined must be an error")
	}
}

// Requirement: doc/specs/encounter-aggregate.md table 4.3.3, FR-096, FR-097 - ApplyTurnChange replaces the tracker and wraps the given events in order under their own kind, adds no event and changes no combatant; it rejects, in order, an ended encounter, an unknown id (the smallest) and a different turn order.
// Path: per case.
func TestApplyTurnChange(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	actTracker, actEvents := trkApplied(t)(e.Tracker().TakeAction(cid("a"), trkAvail, combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}))
	t.Run("action spent", func(t *testing.T) {
		t.Parallel()
		tr, err := e.ApplyTurnChange(actTracker, actEvents)
		after, events := encOK(t, tr, err)
		if !after.Tracker().Equal(actTracker) || !slices.Equal(encKinds(events), []string{"action_taken"}) {
			t.Errorf("tracker replaced = %v, events %v [FR-096]", after.Tracker().Equal(actTracker), encKinds(events))
		}
		combattest.CheckEvent(t, events[0], "action_taken", cid("a"))
		if env := events[0].(combat.TurnEventOccurred); env.Round() != 1 || env.EncounterID() != eid() || env.Event().Kind() != "action_taken" {
			t.Errorf("envelope = %#v", env)
		}
		encOthersEqualCombatants(t, e, after)
	})
	t.Run("no events", func(t *testing.T) {
		t.Parallel()
		tr, err := e.ApplyTurnChange(actTracker, nil)
		after, events := encOK(t, tr, err)
		if len(events) != 0 || !after.Tracker().Equal(actTracker) {
			t.Errorf("events %v, want none with the tracker replaced [FR-096]", encKinds(events))
		}
		tr, err = e.ApplyTurnChange(actTracker, []combat.TurnEvent{})
		if _, evs := encOK(t, tr, err); len(evs) != 0 {
			t.Errorf("an empty list gave events")
		}
	})
	t.Run("several events in order", func(t *testing.T) {
		t.Parallel()
		step, err := e.Tracker().Move(cid("a"), []combat.MoveSegment{trkWalk(t, 10)})
		tk, tev := trkApplied(t)(step, err)
		step, err = tk.InteractWithObject(cid("a"))
		tk, tev2 := trkApplied(t)(step, err)
		tr, err := e.ApplyTurnChange(tk, append(tev, tev2...))
		_, events := encOK(t, tr, err)
		if !slices.Equal(encKinds(events), []string{"movement_spent", "object_interacted"}) {
			t.Errorf("events = %v, want the two in order [FR-096]", encKinds(events))
		}
	})
	t.Run("a Dying active combatant adds no event", func(t *testing.T) {
		t.Parallel()
		d, _ := encStart(t, encC(t, "a", combat.SideA, dyingStatus(t, 20, 0, 0), combat.Unconscious{}), encPC(t, "b", combat.SideB, 20, 20))
		tk, tev := trkApplied(t)(d.Tracker().TakeAction(cid("a"), trkAvail, combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}))
		tr, err := d.ApplyTurnChange(tk, tev)
		if _, events := encOK(t, tr, err); !slices.Equal(encKinds(events), []string{"action_taken"}) {
			t.Errorf("events = %v, want ActionTaken only [FR-096]", encKinds(events))
		}
	})
	t.Run("different order", func(t *testing.T) {
		t.Parallel()
		other := trkTracker(t, trkProfile(t, cid("b"), content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, cid("a"), content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, cid("c"), content.SizeMedium, trkSpeeds(t, 30)))
		tr, err := e.ApplyTurnChange(other, nil)
		if r := encNo(t, e, tr, err); r != (combat.TurnOrderChangedRejection{}) || r.String() != "turn_order_changed" {
			t.Errorf("rejection = %#v, want TurnOrderChanged [FR-097]", r)
		}
	})
	t.Run("unknown id, and the smallest of two", func(t *testing.T) {
		t.Parallel()
		step, err := e.Tracker().UpdateMovementSpeeds(cid("a"), trkSpeeds(t, 20))
		tk, _ := trkApplied(t)(step, err)
		for _, evs := range [][]combat.TurnEvent{{unknownSpeeds(t, "z")}, {unknownSpeeds(t, "z"), unknownSpeeds(t, "b")}, {unknownSpeeds(t, "b"), unknownSpeeds(t, "z")}} {
			tr, err := e.ApplyTurnChange(tk, evs)
			if r := encNo(t, e, tr, err); r != (combat.UnknownCombatantRejection{ID: cid("z")}) {
				t.Errorf("rejection = %#v, want unknown z, the only id the encounter lacks [FR-097]", r)
			}
		}
	})
	t.Run("the first of two unknown ids in code unit order", func(t *testing.T) {
		t.Parallel()
		step, err := e.Tracker().UpdateMovementSpeeds(cid("a"), trkSpeeds(t, 20))
		tk, _ := trkApplied(t)(step, err)
		tr, err := e.ApplyTurnChange(tk, []combat.TurnEvent{unknownSpeeds(t, "z"), unknownSpeeds(t, "y")})
		if r := encNo(t, e, tr, err); r != (combat.UnknownCombatantRejection{ID: cid("y")}) {
			t.Errorf("rejection = %#v, want y, the smaller [FR-097]", r)
		}
	})
	t.Run("precedence", func(t *testing.T) {
		t.Parallel()
		other := trkTracker(t, trkProfile(t, cid("b"), content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, cid("a"), content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, cid("c"), content.SizeMedium, trkSpeeds(t, 30)))
		unknown := []combat.TurnEvent{unknownSpeeds(t, "z")}
		tr, err := e.ApplyTurnChange(other, unknown)
		if r := encNo(t, e, tr, err); r != (combat.UnknownCombatantRejection{ID: cid("z")}) {
			t.Errorf("an unknown id and a different order: %#v, want UnknownCombatant [FR-097 step 2 before 3]", r)
		}
		ended := endedEncounter(t, e)
		tr, err = ended.ApplyTurnChange(other, unknown)
		if r := encNo(t, ended, tr, err); r != (combat.EncounterAlreadyEndedRejection{}) {
			t.Errorf("ended, unknown and different order: %#v, want EncounterAlreadyEnded [FR-097 step 1]", r)
		}
	})
	t.Run("invalid input", func(t *testing.T) {
		t.Parallel()
		for name, f := range map[string]func() (any, error){
			"zero tracker":   func() (any, error) { return e.ApplyTurnChange(combat.TurnTracker{}, nil) },
			"nil event":      func() (any, error) { return e.ApplyTurnChange(actTracker, []combat.TurnEvent{nil}) },
			"zero encounter": func() (any, error) { return (combat.Encounter{}).ApplyTurnChange(actTracker, nil) },
		} {
			if _, err := f(); err == nil {
				t.Errorf("%s: want an ErrInvalidInput error", name)
			}
		}
	})
}

// unknownSpeeds is a MovementSpeedsUpdated event about the id.
func unknownSpeeds(tb testing.TB, id string) combat.TurnEvent {
	tb.Helper()
	tk := trkTracker(tb, trkProfile(tb, cid(id), content.SizeMedium, trkSpeeds(tb, 30)))
	step, err := tk.UpdateMovementSpeeds(cid(id), trkSpeeds(tb, 20))
	_, evs := trkApplied(tb)(step, err)
	return evs[0]
}

// encOthersEqualCombatants asserts the combatants of both encounters are equal.
func encOthersEqualCombatants(tb testing.TB, a, b combat.Encounter) {
	tb.Helper()
	if !slices.EqualFunc(a.Combatants(), b.Combatants(), combat.Combatant.Equal) {
		tb.Errorf("the combatants changed [FR-096]")
	}
}

// endedEncounter ends e by agreement.
func endedEncounter(tb testing.TB, e combat.Encounter) combat.Encounter {
	tb.Helper()
	cmd, err := combat.NewEndEncounterCommand(e, combat.EndReasonMutualAgreement{})
	if err != nil {
		tb.Fatal(err)
	}
	res, err := combat.NewEndEncounter().Run(context.Background(), cmd)
	a, ok := res.(combat.EndEncounterApplied)
	if err != nil || !ok {
		tb.Fatalf("EndEncounter = %#v, %v", res, err)
	}
	return a.Encounter()
}

// Requirement: doc/specs/encounter-aggregate.md table 4.3.3 (every event kind), FR-096 - the tracker events of Advance, TakeAction, Move, InteractWithObject, TakeReaction and UpdateMovementSpeeds are each wrapped under their own kind and none is dropped, in the order given.
// Path: happy.
func TestApplyTurnChangeWrapsEveryKindUnderItsOwnKind(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	tr := e.Tracker()
	var events []combat.TurnEvent
	collect := func(step combat.TrackerStep, err error) {
		t.Helper()
		next, evs := trkApplied(t)(step, err)
		tr, events = next, append(events, evs...)
	}
	collect(tr.TakeAction(cid("a"), trkAvail, combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}))
	collect(tr.Move(cid("a"), []combat.MoveSegment{trkWalk(t, 5)}))
	collect(tr.InteractWithObject(cid("a")))
	collect(tr.UpdateMovementSpeeds(cid("b"), trkSpeeds(t, 20)))
	collect(tr.TakeReaction(cid("b"), trkAvail, combat.SpellReaction{Spell: "shield"}))
	collect(tr.Advance(combat.TurnAdvance{SharedSpaceOccupants: []content.CreatureSize{content.SizeMedium}}))
	for range 2 {
		collect(tr.Advance(combat.TurnAdvance{}))
	}
	if len(slices.Compact(slices.Sorted(slices.Values(kindsOfTurn(events))))) < 9 {
		t.Fatalf("the fixture covers too few kinds: %v", kindsOfTurn(events))
	}
	trk, err := e.ApplyTurnChange(tr, events)
	_, wrapped := encOK(t, trk, err)
	if !slices.Equal(encKinds(wrapped), kindsOfTurn(events)) {
		t.Errorf("wrapped kinds = %v, want %v [FR-096]", encKinds(wrapped), kindsOfTurn(events))
	}
	for i, w := range wrapped {
		env := w.(combat.TurnEventOccurred)
		if env.Event().Kind() != events[i].Kind() || !slices.Equal(env.Combatants(), events[i].Combatants()) {
			t.Errorf("event %d does not report the wrapped kind and combatants", i)
		}
	}
	// The rounds: events before the wrap carry round 1, the round started and what follows carry round 2.
	last := wrapped[len(wrapped)-1].(combat.TurnEventOccurred)
	if last.Round() != 2 || wrapped[0].(combat.TurnEventOccurred).Round() != 1 {
		t.Errorf("rounds: first %d last %d, want 1 and 2 [FR-051]", wrapped[0].(combat.TurnEventOccurred).Round(), last.Round())
	}
}

func kindsOfTurn(events []combat.TurnEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}
