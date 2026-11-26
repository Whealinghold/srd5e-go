package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func trkAdvance(tb testing.TB, t combat.TurnTracker, a ...combat.TurnAdvance) (combat.TurnTracker, []combat.TurnEvent) {
	tb.Helper()
	var adv combat.TurnAdvance
	if len(a) > 0 {
		adv = a[0]
	}
	return trkApplied(tb)(t.Advance(adv))
}

// Requirement: doc/specs/initiative-and-actions.md FR-028, FR-029, FR-031, FR-033, FR-032, FR-141 - advancing ends the active turn and starts the next in the same round; the last combatant ending starts the next round with the first; the order is the same every round; the active and next combatants are exposed; the tracker passed in is unchanged.
// Path: happy.
func TestAdvanceWalksTheRounds(t *testing.T) {
	t.Parallel()
	start := trkABC(t)
	snapshot := trkABC(t)
	tr := start
	type step struct {
		active, next string
		round        int
		kinds        []string
	}
	want := []step{
		{"b", "c", 1, []string{"tracker_turn_ended", "tracker_turn_started"}},
		{"c", "a", 1, []string{"tracker_turn_ended", "tracker_turn_started"}},
		{"a", "b", 2, []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started"}},
		{"b", "c", 2, []string{"tracker_turn_ended", "tracker_turn_started"}},
	}
	for i, w := range want {
		var ev []combat.TurnEvent
		before := tr
		tr, ev = trkAdvance(t, tr)
		if string(tr.Active().Combatant) != w.active || string(tr.Next().Combatant) != w.next || tr.Round() != w.round || !slices.Equal(trkKinds(ev), w.kinds) {
			t.Errorf("step %d: active %v next %v round %d events %v, want %+v", i, tr.Active(), tr.Next(), tr.Round(), trkKinds(ev), w)
		}
		if !before.Equal(before) || before.Round() == tr.Round() && before.Active() == tr.Active() {
			t.Errorf("step %d: the tracker did not move", i)
		}
		if !tr.Order().Equal(start.Order()) {
			t.Errorf("step %d: the order changed [FR-031]", i)
		}
	}
	if !start.Equal(snapshot) {
		t.Error("Advance changed the tracker it was called on [FR-032]")
	}
	_, ev := trkAdvance(t, start)
	if e := ev[0].(combat.TrackerTurnEnded); e.Round() != 1 || e.Combatants()[0] != trkA {
		t.Errorf("TrackerTurnEnded = %+v", e)
	}
	if e := ev[1].(combat.TrackerTurnStarted); e.Round() != 1 || e.Combatants()[0] != trkB {
		t.Errorf("TrackerTurnStarted = %+v", e)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-030, FR-135, FR-141, table 4.3c - one combatant starts a new round every time and an expiring readied action follows the turn start.
// Path: unhappy (empty case).
func TestAdvanceWithOneCombatant(t *testing.T) {
	t.Parallel()
	tr := trkTracker(t, trkProfile(t, trkA, content.SizeMedium, trkSpeeds(t, 30)))
	ready := combat.ReadyChoice{Action: trkReadied(t, "door", combat.ReadiedActionResponse{Choice: combat.ImprovisedActionChoice{}})}
	tr = trkTake(t, tr, trkA, ready)
	for round := 2; round <= 3; round++ {
		var ev []combat.TurnEvent
		tr, ev = trkAdvance(t, tr)
		want := []string{"tracker_turn_ended", "tracker_round_started", "tracker_turn_started"}
		if round == 2 {
			want = append(want, "readied_action_expired")
		}
		if tr.Round() != round || !slices.Equal(trkKinds(ev), want) {
			t.Errorf("round %d: events %v, want %v [FR-030, FR-135]", tr.Round(), trkKinds(ev), want)
		}
	}
}

func trkReadied(tb testing.TB, trigger string, r combat.ReadiedResponse) combat.ReadiedAction {
	tb.Helper()
	a, err := combat.NewReadiedAction(combat.MustReadyTriggerID(trigger), r, nil)
	if err != nil {
		tb.Fatal(err)
	}
	return a
}

// Requirement: per case (table 4.3a, FR-034 to FR-041, FR-141)
// Path: per case.
func TestTurnResourcesAndRegain(t *testing.T) {
	t.Parallel()
	t.Run("a new turn starts fresh and unused movement is lost", func(t *testing.T) {
		t.Parallel()
		tr := trkMove(t, trkABC(t), trkA, trkWalk(t, 30))
		tr = trkTake(t, tr, trkA, combat.DashChoice{Speed: combat.SpeedNameWalk})
		tr, _ = trkApplied(t)(tr.InteractWithObject(trkA))
		tr, _ = trkAdvance(t, tr)
		r := tr.Resources()
		if r.MovedFeet() != 0 || r.DashExtraFeet() != 0 || r.Action() != combat.ResourceAvailable || r.BonusAction() != combat.ResourceAvailable ||
			r.Interaction() != combat.ResourceAvailable || r.Disengage() != combat.NotDisengaged {
			t.Errorf("resources of B = %+v, want fresh [FR-034, FR-040]", r)
		}
		// Back at A: nothing carried over.
		tr, _ = trkAdvance(t, tr)
		tr, _ = trkAdvance(t, tr)
		if tr.Active().Combatant != trkA || tr.Resources().MovedFeet() != 0 || tr.Resources().DashExtraFeet() != 0 || tr.Resources().Action() != combat.ResourceAvailable {
			t.Errorf("A's second turn = %+v, want fresh [FR-040]", tr.Resources())
		}
		if _, err := tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 30)}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("special speed does not add to the walking allowance", func(t *testing.T) {
		t.Parallel()
		tr := trkABC(t)
		trkApplied(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 30)}))
		if r := trkReason(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 30), trkWalk(t, 1)})); r != (combat.TurnNoMovementRemainingRejection{SegmentIndex: 1, MovementLeft: 0}) {
			t.Errorf("reason = %#v [FR-034]", r)
		}
		trkApplied(t)(tr.Move(trkA, []combat.MoveSegment{trkMode(t, 40, combat.MovementModeFly)}))
	})
	t.Run("Reaction regained at its owner's turn only and kept across the round boundary", func(t *testing.T) {
		t.Parallel()
		tr := advN(t, trkABC(t), 2) // C is active in round 1
		tr, _ = trkApplied(t)(tr.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
		tr, _ = trkAdvance(t, tr) // A, round 2
		if !tr.ReactionSpent(trkB) || tr.Round() != 2 {
			t.Errorf("B's Reaction spent = %v in round %d, want spent across the round boundary [FR-036]", tr.ReactionSpent(trkB), tr.Round())
		}
		tr, _ = trkAdvance(t, tr) // B
		if tr.ReactionSpent(trkB) {
			t.Error("B's Reaction is still spent at its own turn start [FR-035]")
		}
	})
	t.Run("Dodge ends when its owner's next turn starts", func(t *testing.T) {
		t.Parallel()
		tr := trkTake(t, trkABC(t), trkA, combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge})
		for i, want := range []bool{true, true, false} {
			tr, _ = trkAdvance(t, tr)
			if tr.IsDodging(trkA) != want {
				t.Errorf("advance %d: dodging = %v, want %v [FR-037]", i+1, tr.IsDodging(trkA), want)
			}
		}
	})
	t.Run("Disengage lasts for the rest of the turn and not the next", func(t *testing.T) {
		t.Parallel()
		tr := trkTake(t, trkABC(t), trkA, combat.SimpleActionChoice{Action: combat.SimpleStandardActionDisengage})
		if tr.Resources().Disengage() != combat.Disengaged {
			t.Fatal("not disengaged")
		}
		tr, _ = trkAdvance(t, tr)
		if tr.Resources().Disengage() != combat.NotDisengaged {
			t.Error("B starts disengaged [FR-039]")
		}
	})
	t.Run("a readied action expires at its owner's next turn without spending the Reaction", func(t *testing.T) {
		t.Parallel()
		tr := trkTake(t, trkABC(t), trkA, combat.ReadyChoice{Action: trkReadied(t, "door", combat.ReadiedActionResponse{Choice: combat.ImprovisedActionChoice{}})})
		var ev []combat.TurnEvent
		for i := 0; i < 3; i++ {
			tr, ev = trkAdvance(t, tr)
		}
		if _, held := tr.ReadiedActionOf(trkA); held || tr.ReactionSpent(trkA) {
			t.Errorf("readied action held = %v, Reaction spent = %v, want expired and unspent [FR-038]", held, tr.ReactionSpent(trkA))
		}
		last := ev[len(ev)-1].(combat.ReadiedActionExpired)
		if last.Trigger() != "door" || last.Combatants()[0] != trkA || last.Held() != (combat.HoldsNoSpell{}) {
			t.Errorf("expiry = %+v", last)
		}
	})
	t.Run("ending a turn having used nothing is fine", func(t *testing.T) {
		t.Parallel()
		tr, ev := trkAdvance(t, trkABC(t))
		if len(ev) != 2 || tr.Active().Combatant != trkB {
			t.Errorf("events %v [FR-041]", trkKinds(ev))
		}
	})
}

// Requirement: per case (table 4.3b, FR-043, FR-141)
// Path: per case.
func TestAdvanceInASharedSpace(t *testing.T) {
	t.Parallel()
	sz := content.SizeTiny
	_ = sz
	cases := []struct {
		name, path string
		ending     content.CreatureSize
		others     []content.CreatureSize
		prone      bool
	}{
		{"same size", "happy", content.SizeMedium, []content.CreatureSize{content.SizeMedium}, true},
		{"smaller than the other", "happy", content.SizeSmall, []content.CreatureSize{content.SizeLarge}, true},
		{"top of the scale", "happy", content.SizeHuge, []content.CreatureSize{content.SizeGargantuan}, true},
		{"larger than the other", "unhappy (boundary)", content.SizeLarge, []content.CreatureSize{content.SizeMedium}, false},
		{"larger than Tiny", "unhappy (boundary)", content.SizeMedium, []content.CreatureSize{content.SizeTiny}, false},
		{"Tiny is always exempt", "unhappy (boundary)", content.SizeTiny, []content.CreatureSize{content.SizeTiny, content.SizeMedium}, false},
		{"not sharing", "unhappy (empty case)", content.SizeMedium, nil, false},
		{"two sharers, not larger than one", "unhappy (rule interaction)", content.SizeLarge, []content.CreatureSize{content.SizeMedium, content.SizeHuge}, true},
		{"larger than both", "unhappy (rule interaction)", content.SizeLarge, []content.CreatureSize{content.SizeMedium, content.SizeSmall}, false},
		{"Tiny among them", "unhappy (rule interaction)", content.SizeLarge, []content.CreatureSize{content.SizeTiny, content.SizeMedium}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := trkTracker(t, trkProfile(t, trkA, c.ending, trkSpeeds(t, 30)), trkProfile(t, trkB, content.SizeMedium, trkSpeeds(t, 30)))
			_, ev := trkAdvance(t, tr, combat.TurnAdvance{SharedSpaceOccupants: c.others})
			want := []string{"tracker_turn_ended", "tracker_turn_started"}
			if c.prone {
				want = []string{"tracker_turn_ended", "fell_prone_in_shared_space", "tracker_turn_started"}
			}
			if !slices.Equal(trkKinds(ev), want) {
				t.Errorf("events = %v, want %v [FR-043]", trkKinds(ev), want)
			}
		})
	}
	tr := trkABC(t)
	if _, err := tr.Advance(combat.TurnAdvance{SharedSpaceOccupants: []content.CreatureSize{0}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("undefined size: %v, want invalid input", err)
	}
	if _, err := (combat.TurnTracker{}).Advance(combat.TurnAdvance{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero tracker: %v, want invalid input", err)
	}
}

// Requirement: per case (table 4.3c, FR-042, FR-135, FR-141)
// Path: per case.
func TestAdvanceEventOrder(t *testing.T) {
	t.Parallel()
	upd := func(id combat.CombatantID, walk int) combat.SpeedUpdate {
		return combat.SpeedUpdate{Combatant: id, Speeds: trkSpeeds(t, walk)}
	}
	t.Run("last in order, shared space, readied action and updates for B then A", func(t *testing.T) {
		t.Parallel()
		tr := trkTake(t, trkABC(t), trkA, combat.ReadyChoice{Action: trkReadied(t, "t", combat.ReadiedActionResponse{Choice: combat.ImprovisedActionChoice{}})})
		tr, _ = trkAdvance(t, tr)
		tr, _ = trkAdvance(t, tr)
		_, ev := trkAdvance(t, tr, combat.TurnAdvance{Updates: []combat.SpeedUpdate{upd(trkB, 10), upd(trkA, 20)},
			SharedSpaceOccupants: []content.CreatureSize{content.SizeMedium}})
		want := []string{"tracker_turn_ended", "fell_prone_in_shared_space", "movement_speeds_updated", "movement_speeds_updated",
			"tracker_round_started", "tracker_turn_started", "readied_action_expired"}
		if !slices.Equal(trkKinds(ev), want) {
			t.Fatalf("events = %v, want %v [FR-135]", trkKinds(ev), want)
		}
		if ev[2].Combatants()[0] != trkA || ev[3].Combatants()[0] != trkB {
			t.Errorf("updates are not in turn order: %v %v [FR-135]", ev[2].Combatants(), ev[3].Combatants())
		}
		if ev[2].(combat.MovementSpeedsUpdated).Speeds().Walk() != 20 {
			t.Error("the update of A lost its speeds")
		}
	})
	t.Run("an unknown id is refused as the first in the supplied order and nothing is applied", func(t *testing.T) {
		t.Parallel()
		tr := trkABC(t)
		snapshot := trkABC(t)
		r := trkReason(t)(tr.Advance(combat.TurnAdvance{Updates: []combat.SpeedUpdate{upd(trkB, 1), upd("x", 1), upd("y", 1)}}))
		if r != (combat.TurnUnknownCombatantRejection{ID: "x"}) || !tr.Equal(snapshot) {
			t.Errorf("reason = %#v, want UnknownCombatant(x) with the tracker unchanged [FR-042]", r)
		}
	})
	t.Run("updates persist to the next turn and later ones win", func(t *testing.T) {
		t.Parallel()
		tr, _ := trkAdvance(t, trkABC(t), combat.TurnAdvance{Updates: []combat.SpeedUpdate{upd(trkB, 20), upd(trkB, 25)}})
		if p, _ := tr.ProfileOf(trkB); p.Speeds.Walk() != 25 {
			t.Errorf("B walks %d, want the later update 25 [FR-042]", p.Speeds.Walk())
		}
		if r := trkReason(t)(tr.Move(trkB, []combat.MoveSegment{trkWalk(t, 26)})); r != (combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}) {
			t.Errorf("reason = %#v, want a shortfall of 1 [FR-116]", r)
		}
		tr, _ = trkAdvance(t, tr)
		tr, _ = trkAdvance(t, tr)
		tr, _ = trkAdvance(t, tr)
		tr, _ = trkAdvance(t, tr)
		if p, _ := tr.ProfileOf(trkB); p.Speeds.Walk() != 25 {
			t.Error("the update did not persist [FR-116]")
		}
	})
}

// Requirement: doc/specs/initiative-and-actions.md FR-157 - OncePerTurn lasts until the current turn ends whoever's it is; OnceUntilStartOfYourNextTurn until that combatant's next turn starts.
// Path: unhappy (state error).
func TestFeatUseWindows(t *testing.T) {
	t.Parallel()
	per, until := character.OncePerTurn{}, character.OnceUntilStartOfYourNextTurn{}
	tr, _ := trkApplied(t)(trkABC(t).RecordFeatUse(trkB, "savage_attacker", per))
	if r := trkReason(t)(tr.RecordFeatUse(trkB, "savage_attacker", per)); r != (combat.TurnFeatUseLimitedRejection{Feat: "savage_attacker"}) {
		t.Errorf("second use: %#v, want FeatUseLimited", r)
	}
	tr, _ = trkApplied(t)(tr.RecordFeatUse(trkB, "other_feat", per))
	tr, _ = trkApplied(t)(tr.RecordFeatUse(trkC, "peerless_aim", until))
	tr, _ = trkAdvance(t, tr) // A's turn ends: every per-turn use is cleared, C's until-next-turn use is not
	trkApplied(t)(tr.RecordFeatUse(trkB, "savage_attacker", per))
	if r := trkReason(t)(tr.RecordFeatUse(trkC, "peerless_aim", until)); r == nil {
		t.Error("C's use was cleared before its turn")
	}
	tr, _ = trkAdvance(t, tr) // B ends, C starts
	trkApplied(t)(tr.RecordFeatUse(trkC, "peerless_aim", until))
}

// Requirement: doc/specs/initiative-and-actions.md FR-157 - the until-next-turn use of another combatant is refused until that combatant's turn starts.
// Path: unhappy (state error).
func TestFeatUseUntilNextTurn(t *testing.T) {
	t.Parallel()
	until := character.OnceUntilStartOfYourNextTurn{}
	tr, _ := trkApplied(t)(trkABC(t).RecordFeatUse(trkB, "peerless_aim", until))
	if r := trkReason(t)(tr.RecordFeatUse(trkB, "peerless_aim", until)); r == nil {
		t.Fatal("second use accepted")
	}
	tr, _ = trkAdvance(t, tr) // B's turn starts
	trkApplied(t)(tr.RecordFeatUse(trkB, "peerless_aim", until))
}

// Requirement: doc/specs/initiative-and-actions.md FR-157, FR-119 - an unknown id is UnknownCombatant, no limit other than the two turn limits and no invalid feat id is accepted.
// Path: unhappy (invalid input).
func TestRecordFeatUseValidation(t *testing.T) {
	t.Parallel()
	tr := trkABC(t)
	if r := trkReason(t)(tr.RecordFeatUse("x", "f", character.OncePerTurn{})); r != (combat.TurnUnknownCombatantRejection{ID: "x"}) {
		t.Errorf("unknown id: %#v", r)
	}
	for name, limit := range map[string]character.FeatUsageLimit{"none": character.NoUsageLimit{}, "nil": nil} {
		if _, err := tr.RecordFeatUse(trkA, "f", limit); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, want invalid input", name, err)
		}
	}
	if _, err := tr.RecordFeatUse(trkA, "Bad Feat", character.OncePerTurn{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("bad feat id: %v, want invalid input", err)
	}
	ev := func() combat.FeatUseRecorded {
		_, e := trkApplied(t)(tr.RecordFeatUse(trkA, "f", character.OncePerTurn{}))
		return e[0].(combat.FeatUseRecorded)
	}()
	if ev.Feat() != "f" || ev.Limit() != (character.OncePerTurn{}) || ev.Combatants()[0] != trkA || ev.Kind() != "feat_use_recorded" {
		t.Errorf("event = %+v", ev)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-154, FR-011 - trackers compare by value through Equal over every field; two trackers differing in any one field are unequal; set and map insertion order is irrelevant.
// Path: happy.
func TestTurnTrackerEquality(t *testing.T) {
	t.Parallel()
	base := trkABC(t)
	if !base.Equal(trkABC(t)) {
		t.Fatal("two trackers built alike are unequal")
	}
	ready := combat.ReadyChoice{Action: trkReadied(t, "t", combat.ReadiedActionResponse{Choice: combat.ImprovisedActionChoice{}})}
	grant, err := combat.NewAdditionalActionGrant("action_surge", combat.AnyActionBut{})
	if err != nil {
		t.Fatal(err)
	}
	differ := map[string]func(combat.TurnTracker) combat.TurnTracker{
		"round":  func(tr combat.TurnTracker) combat.TurnTracker { return advN(t, tr, 3) },
		"active": func(tr combat.TurnTracker) combat.TurnTracker { return advN(t, tr, 1) },
		"profile": func(tr combat.TurnTracker) combat.TurnTracker {
			n, _ := trkApplied(t)(tr.UpdateMovementSpeeds(trkB, trkSpeeds(t, 5)))
			return n
		},
		"resources": func(tr combat.TurnTracker) combat.TurnTracker { return trkMove(t, tr, trkA, trkWalk(t, 5)) },
		"reaction": func(tr combat.TurnTracker) combat.TurnTracker {
			n, _ := trkApplied(t)(tr.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
			return n
		},
		"dodging": func(tr combat.TurnTracker) combat.TurnTracker {
			return trkTake(t, tr, trkA, combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge})
		},
		"readied": func(tr combat.TurnTracker) combat.TurnTracker { return trkTake(t, tr, trkA, ready) },
		"grant": func(tr combat.TurnTracker) combat.TurnTracker {
			n, _ := trkApplied(t)(tr.GrantAdditionalAction(trkA, grant))
			return n
		},
		"feat use": func(tr combat.TurnTracker) combat.TurnTracker {
			n, _ := trkApplied(t)(tr.RecordFeatUse(trkA, "f", character.OncePerTurn{}))
			return n
		},
	}
	for name, f := range differ {
		if got := f(base); got.Equal(base) || base.Equal(got) || !got.Equal(f(base)) {
			t.Errorf("a tracker differing in %s compares equal to the base, or two alike ones differ [FR-154]", name)
		}
	}
	// Insertion order of the sets is irrelevant.
	x, _ := trkApplied(t)(base.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
	x, _ = trkApplied(t)(x.TakeReaction(trkC, trkAvail, combat.SpellReaction{Spell: "shield"}))
	y, _ := trkApplied(t)(base.TakeReaction(trkC, trkAvail, combat.SpellReaction{Spell: "shield"}))
	y, _ = trkApplied(t)(y.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
	if !x.Equal(y) {
		t.Error("insertion order of the spent Reactions matters [FR-154]")
	}
	// The same readied move with different segments is different.
	mv := func(feet int) combat.ReadyChoice {
		r, err := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, feet)})
		if err != nil {
			t.Fatal(err)
		}
		return combat.ReadyChoice{Action: trkReadied(t, "t", r)}
	}
	if trkTake(t, base, trkA, mv(5)).Equal(trkTake(t, base, trkA, mv(10))) || !trkTake(t, base, trkA, mv(5)).Equal(trkTake(t, base, trkA, mv(5))) {
		t.Error("readied moves compare wrongly [FR-154]")
	}
	if !(combat.TurnTracker{}).IsZero() || base.IsZero() || !(combat.TurnTracker{}).Equal(combat.TurnTracker{}) {
		t.Error("IsZero or the zero tracker's Equal is wrong")
	}
}

func advN(tb testing.TB, t combat.TurnTracker, n int) combat.TurnTracker {
	tb.Helper()
	for range n {
		t, _ = trkAdvance(tb, t)
	}
	return t
}

// Requirement: doc/specs/initiative-and-actions.md FR-140, FR-096 - the events and trackers of a step are copies; the receiver of every operation is unchanged by it and by a refusal.
// Path: unhappy (immutability).
func TestTrackerStepCopiesAndImmutability(t *testing.T) {
	t.Parallel()
	tr := trkABC(t)
	snapshot := trkABC(t)
	step, err := tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 10)})
	if err != nil {
		t.Fatal(err)
	}
	ev := step.(combat.TrackerStepApplied).Events()
	ev[0] = nil
	if step.(combat.TrackerStepApplied).Events()[0] == nil {
		t.Error("Events shares its slice [FR-128]")
	}
	moved := step.(combat.TrackerStepApplied).Events()[0].(combat.MovementSpent)
	charges := moved.Charges()
	charges[0].Feet = 999
	if moved.Charges()[0].Feet != 10 || moved.TotalFeet() != 10 || moved.MovedFeet() != 10 {
		t.Errorf("MovementSpent = %+v, want a copy of one charge of 10", moved)
	}
	trkReason(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 999)}))
	trkReason(t)(tr.TakeAction(trkB, trkAvail, combat.ImprovisedActionChoice{}))
	if !tr.Equal(snapshot) {
		t.Error("an operation or a refusal changed the tracker [FR-096]")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-119 - the queries of a tracker are total: the zero tracker knows no id; HeldSpellOf and DodgeBenefits return an invalid-input error for an unknown id.
// Path: unhappy (invalid input).
func TestTrackerQueries(t *testing.T) {
	t.Parallel()
	tr := trkABC(t)
	if _, ok := tr.ProfileOf("x"); ok {
		t.Error("ProfileOf(x) found")
	}
	if p, ok := tr.ProfileOf(trkB); !ok || p.ID != trkB || p.Size != content.SizeMedium {
		t.Errorf("ProfileOf(b) = %+v, %v", p, ok)
	}
	if _, err := tr.HeldSpellOf("x"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("HeldSpellOf(x) = %v", err)
	}
	if h, err := tr.HeldSpellOf(trkA); err != nil || h != (combat.HoldsNoSpell{}) {
		t.Errorf("HeldSpellOf(a) = %v, %v", h, err)
	}
	if _, err := tr.DodgeBenefits("x", trkAvail); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("DodgeBenefits(x) = %v", err)
	}
	if _, err := tr.DodgeBenefits(trkA, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("DodgeBenefits with no availability = %v", err)
	}
	if r := trkReason(t)((combat.TurnTracker{}).TakeAction(trkA, trkAvail, combat.ImprovisedActionChoice{})); r != (combat.TurnUnknownCombatantRejection{ID: trkA}) {
		t.Errorf("zero tracker: %#v, want UnknownCombatant", r)
	}
	g, used := tr.AdditionalAction()
	if !g.IsZero() || used != 0 {
		t.Errorf("AdditionalAction = %+v, %v, want none", g, used)
	}
	if o := tr.Order(); o.Len() != 3 || !slices.Equal(trkOrderIDs(o), []string{"a", "b", "c"}) {
		t.Errorf("Order = %v", o.Standings())
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-154, FR-119 - trackers differing only in whether the additional action is spent are unequal both ways round; an availability that is not defined is invalid input for every operation that takes one.
// Path: unhappy (boundary).
func TestTrackerEqualityOfSpentGrantAndUndefinedAvailability(t *testing.T) {
	t.Parallel()
	g, err := combat.NewAdditionalActionGrant("surge", combat.AnyActionBut{})
	if err != nil {
		t.Fatal(err)
	}
	granted, _ := trkApplied(t)(trkABC(t).GrantAdditionalAction(trkA, g))
	spent, _ := trkApplied(t)(granted.TakeAction(trkA, trkAvail, trkAttack))
	_, used := granted.AdditionalAction()
	if _, extraUsed := spent.AdditionalAction(); used == extraUsed || granted.Equal(spent) || spent.Equal(granted) {
		t.Errorf("trackers differing only in the spent additional action compare equal (%v, %v)", used, extraUsed)
	}
	base := trkABC(t)
	for _, av := range []combat.ConditionActionAvailability{-1, 0, 3} {
		if _, err := base.TakeAction(trkA, av, trkAttack); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("TakeAction with availability %d: %v, want invalid input", av, err)
		}
		if _, err := base.DodgeBenefits(trkA, av); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("DodgeBenefits with availability %d: %v, want invalid input", av, err)
		}
		if _, err := base.TakeReaction(trkB, av, combat.SpellReaction{Spell: "s"}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("TakeReaction with availability %d: %v, want invalid input", av, err)
		}
	}
}
