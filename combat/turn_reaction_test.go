package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func trkOA(provoker combat.CombatantID, means combat.DepartureMeans, v combat.Visibility) combat.OpportunityAttackReaction {
	return combat.OpportunityAttackReaction{Provoker: provoker, Means: means, Visibility: v}
}

// Requirement: per case (table 4.6a, FR-064 to FR-071, FR-118)
// Path: per case.
func TestOpportunityAttackReaction(t *testing.T) {
	t.Parallel()
	base := trkABC(t)
	disengaged := trkTake(t, base, trkA, trkDisenga)
	spentB, _ := trkApplied(t)(base.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
	means := []combat.DepartureMeans{combat.DepartureBySpeed, combat.DepartureByAction, combat.DepartureByBonusAction, combat.DepartureByReaction}
	for _, m := range means {
		t.Run("accepted by "+m.String(), func(t *testing.T) {
			t.Parallel()
			tr, ev := trkApplied(t)(base.TakeReaction(trkB, trkAvail, trkOA(trkA, m, combat.VisibilitySeen)))
			e := ev[0].(combat.ReactionTaken)
			rec, ok := e.Record().(combat.OpportunityAttackRecord)
			if !tr.ReactionSpent(trkB) || !ok || rec != (combat.OpportunityAttackRecord{Observer: trkB, Provoker: trkA, Means: m, Timing: combat.AttackTimingRightBeforeLeavingReach}) {
				t.Errorf("record = %+v, spent %v [FR-068, FR-071]", e.Record(), tr.ReactionSpent(trkB))
			}
			if got := e.Combatants(); !slices.Equal(got, []combat.CombatantID{trkB, trkA}) || e.Kind() != "reaction_taken" {
				t.Errorf("Combatants = %v, want observer then provoker [FR-130]", got)
			}
			// FR-067: the active turn is untouched.
			if tr.Active() != base.Active() || tr.Round() != base.Round() || tr.Resources() != base.Resources() {
				t.Errorf("the Reaction changed the active turn [FR-067]")
			}
		})
		t.Run("refused when disengaged by "+m.String(), func(t *testing.T) {
			t.Parallel()
			if r := trkReason(t)(disengaged.TakeReaction(trkB, trkAvail, trkOA(trkA, m, combat.VisibilitySeen))); r != (combat.TurnProvokerDisengagedRejection{}) {
				t.Errorf("reason = %#v [FR-069]", r)
			}
			if disengaged.ReactionSpent(trkB) {
				t.Error("a refusal spent the Reaction")
			}
		})
	}
	cases := []struct {
		name string
		tr   combat.TurnTracker
		id   combat.CombatantID
		av   combat.ConditionActionAvailability
		c    combat.ReactionChoice
		want combat.TurnRejection
	}{
		{"not seen", base, trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilityUnseen), combat.TurnProvokerNotSeenRejection{}},
		{"Reaction spent", spentB, trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilitySeen), combat.TurnReactionAlreadyTakenRejection{}},
		{"disengaged and unseen, spent", func() combat.TurnTracker {
			n, _ := trkApplied(t)(disengaged.TakeReaction(trkB, trkAvail, combat.SpellReaction{Spell: "shield"}))
			return n
		}(), trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilityUnseen), combat.TurnReactionAlreadyTakenRejection{}},
		{"disengaged and unseen, unspent", disengaged, trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilityUnseen), combat.TurnProvokerDisengagedRejection{}},
		{"incapacitated and spent", spentB, trkB, trkNone, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilitySeen), combat.TurnIncapacitatedRejection{}},
		{"unknown provoker", base, trkB, trkAvail, trkOA("x", combat.DepartureBySpeed, combat.VisibilitySeen), combat.TurnUnknownCombatantRejection{ID: "x"}},
		{"unknown reactor beats unknown provoker", base, "y", trkAvail, trkOA("x", combat.DepartureBySpeed, combat.VisibilitySeen), combat.TurnUnknownCombatantRejection{ID: "y"}},
		{"a provoker who is not the active combatant cannot be disengaged", disengaged, trkC, trkAvail, trkOA(trkB, combat.DepartureBySpeed, combat.VisibilitySeen), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			step, err := c.tr.TakeReaction(c.id, c.av, c.c)
			if c.want == nil {
				trkApplied(t)(step, err)
				return
			}
			if r := trkReason(t)(step, err); r != c.want {
				t.Errorf("reason = %#v, want %#v", r, c.want)
			}
		})
	}
	// The active combatant may take a Reaction on its own turn after using action and Bonus Action (FR-065).
	own := trkTake(t, base, trkA, trkAttack)
	trkApplied(t)(own.TakeReaction(trkA, trkAvail, combat.FeatureReaction{Feature: "f", Granted: combat.NoStandardAction{}}))
}

// Requirement: doc/specs/initiative-and-actions.md FR-064, FR-065, FR-066, FR-067 - a combatant that is not active may take a Reaction on another's turn; the active combatant may after using its action and Bonus Action; a second is refused until its own turn starts; the active turn is unchanged.
// Path: per case.
func TestReactionBasics(t *testing.T) {
	t.Parallel()
	fr := combat.FeatureReaction{Feature: "f", Granted: combat.NoStandardAction{}}
	tr := trkTake(t, trkABC(t), trkA, trkAttack)
	bg, _ := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "b"}, combat.NoStandardAction{})
	tr, _ = trkApplied(t)(tr.TakeBonusAction(trkA, trkAvail, combat.EntitledToBonusAction{Grant: bg}))
	own, _ := trkApplied(t)(tr.TakeReaction(trkA, trkAvail, fr))
	if !own.ReactionSpent(trkA) || own.Resources() != tr.Resources() {
		t.Errorf("own-turn Reaction: spent %v resources %+v [FR-065, FR-067]", own.ReactionSpent(trkA), own.Resources())
	}
	if r := trkReason(t)(own.TakeReaction(trkA, trkAvail, fr)); r != (combat.TurnReactionAlreadyTakenRejection{}) {
		t.Errorf("second Reaction: %#v [FR-066]", r)
	}
	later, _ := trkAdvance(t, own)
	later = advN(t, later, 2) // A's next turn
	if later.ReactionSpent(trkA) {
		t.Error("the Reaction is not regained at the owner's turn [FR-035]")
	}
}

// Requirement: per case (table 4.6b, FR-072 to FR-074, FR-121, FR-122, FR-134, FR-161)
// Path: per case.
func TestReadiedAndOtherReactions(t *testing.T) {
	t.Parallel()
	trigger := func(id string) combat.ReadiedActionReaction {
		return combat.ReadiedActionReaction{Trigger: combat.MustReadyTriggerID(id)}
	}
	// B readies on its turn; A (next in line) reacts to it... A acts first, so use C as the observer of a trigger on A's turn.
	mk := func(t *testing.T, speeds combat.MovementSpeeds, resp combat.ReadiedResponse) combat.TurnTracker {
		t.Helper()
		tr := trkTracker(t, trkProfile(t, trkA, content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, trkB, content.SizeMedium, speeds), trkProfile(t, trkC, content.SizeMedium, trkSpeeds(t, 30)))
		tr, _ = trkAdvance(t, tr) // B's turn
		tr = trkTake(t, tr, trkB, combat.ReadyChoice{Action: trkReadied(t, "go", resp)})
		tr, _ = trkAdvance(t, tr) // C's turn: B holds the action
		return tr
	}
	moveResp := func(t *testing.T, segs ...combat.MoveSegment) combat.ReadiedMoveResponse {
		t.Helper()
		r, err := combat.NewReadiedMoveResponse(segs)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	t.Run("a matching trigger spends the Reaction and removes the readied action", func(t *testing.T) {
		t.Parallel()
		tr := mk(t, trkSpeeds(t, 30), combat.ReadiedActionResponse{Choice: trkAttack})
		after, ev := trkApplied(t)(tr.TakeReaction(trkB, trkAvail, trigger("go")))
		rec := ev[0].(combat.ReactionTaken).Record().(combat.ReadiedActionRecord)
		if _, held := after.ReadiedActionOf(trkB); held || !after.ReactionSpent(trkB) || rec.Trigger != "go" || rec.Outcome != (combat.ReadiedChoiceOutcome{Choice: trkAttack}) || rec.Held != (combat.HoldsNoSpell{}) {
			t.Errorf("held %v spent %v record %+v [FR-073, FR-122]", held, after.ReactionSpent(trkB), rec)
		}
		if after.Resources() != tr.Resources() || after.Active() != tr.Active() {
			t.Error("the readied action changed the active turn [FR-067]")
		}
	})
	t.Run("a readied Dash is only recorded", func(t *testing.T) {
		t.Parallel()
		tr := mk(t, trkSpeeds(t, 30), combat.ReadiedActionResponse{Choice: combat.DashChoice{Speed: combat.SpeedNameWalk}})
		after, _ := trkApplied(t)(tr.TakeReaction(trkB, trkAvail, trigger("go")))
		if after.Resources() != tr.Resources() {
			t.Errorf("resources = %+v, want unchanged [FR-134]", after.Resources())
		}
	})
	t.Run("the wrong trigger and no readied action", func(t *testing.T) {
		t.Parallel()
		tr := mk(t, trkSpeeds(t, 30), combat.ReadiedActionResponse{Choice: trkAttack})
		want := combat.TurnNoMatchingReadiedActionRejection{Trigger: "Go"}
		if r := trkReason(t)(tr.TakeReaction(trkB, trkAvail, trigger("Go"))); r != want {
			t.Errorf("wrong case: %#v, want %#v [FR-072, FR-137]", r, want)
		}
		if r := trkReason(t)(tr.TakeReaction(trkC, trkAvail, trigger("go"))); r != (combat.TurnNoMatchingReadiedActionRejection{Trigger: "go"}) {
			t.Errorf("no readied action: %#v [FR-072]", r)
		}
	})
	t.Run("a readied move is judged at the trigger", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name   string
			speeds combat.MovementSpeeds
			segs   []combat.MoveSegment
			update int // set B's walking Speed to this after Ready when >= 0
			want   combat.TurnRejection
		}{
			{"exactly Speed", trkSpeeds(t, 30), []combat.MoveSegment{trkWalk(t, 30)}, -1, nil},
			{"one over", trkSpeeds(t, 30), []combat.MoveSegment{trkWalk(t, 31)}, -1, combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}},
			{"difficult, exact", trkSpeeds(t, 30), []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 15, Mode: combat.MovementModeWalk, Terrain: combat.TerrainDifficult})}, -1, nil},
			{"difficult, over", trkSpeeds(t, 30), []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 16, Mode: combat.MovementModeWalk, Terrain: combat.TerrainDifficult})}, -1, combat.TurnInsufficientMovementRejection{ShortfallFeet: 2}},
			{"Speed 0", trkSpeeds(t, 0), []combat.MoveSegment{trkWalk(t, 5)}, -1, combat.TurnNoMovementRemainingRejection{}},
			{"profile updated lower", trkSpeeds(t, 30), []combat.MoveSegment{trkWalk(t, 25)}, 20, combat.TurnInsufficientMovementRejection{ShortfallFeet: 5}},
			{"fly in response", trkSpeeds(t, 30, fly(40)), []combat.MoveSegment{trkMode(t, 40, combat.MovementModeFly)}, -1, nil},
			{"fly segment, no Fly Speed", trkSpeeds(t, 30), []combat.MoveSegment{trkMode(t, 10, combat.MovementModeFly)}, -1, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameFly}},
			{"ends in an occupied space", trkSpeeds(t, 30), []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, End: combat.SegmentEndOccupiedByCreature})}, -1, combat.TurnEndsInOccupiedSpaceRejection{SegmentIndex: 0}},
			{"blocked by a creature", trkSpeeds(t, 30), []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeMedium, Relation: combat.OccupantOther, Availability: trkAvail}}})}, -1, combat.TurnBlockedByCreatureRejection{}},
		}
		for _, c := range cases {
			tr := mk(t, c.speeds, moveResp(t, c.segs...))
			if c.update >= 0 {
				tr, _ = trkApplied(t)(tr.UpdateMovementSpeeds(trkB, trkSpeeds(t, c.update)))
			}
			step, err := tr.TakeReaction(trkB, trkAvail, trigger("go"))
			if c.want == nil {
				after, ev := trkApplied(t)(step, err)
				rec := ev[0].(combat.ReactionTaken).Record().(combat.ReadiedActionRecord)
				mo := rec.Outcome.(combat.ReadiedMoveOutcome)
				if !after.ReactionSpent(trkB) || mo.TotalFeet() <= 0 || len(mo.Charges()) != 1 || after.Resources() != tr.Resources() {
					t.Errorf("%s: spent %v outcome %+v resources %+v [FR-074]", c.name, after.ReactionSpent(trkB), mo, after.Resources())
				}
				if _, held := after.ReadiedActionOf(trkB); held {
					t.Errorf("%s: the readied action is still held", c.name)
				}
				continue
			}
			if r := trkReason(t)(step, err); r != c.want {
				t.Errorf("%s: reason %#v, want %#v [FR-074]", c.name, r, c.want)
			}
			if tr.ReactionSpent(trkB) {
				t.Errorf("%s: a refusal spent the Reaction", c.name)
			}
			if _, held := tr.ReadiedActionOf(trkB); !held {
				t.Errorf("%s: a refusal dropped the readied action", c.name)
			}
		}
	})
	t.Run("a readied move leaves the active combatant's movement alone", func(t *testing.T) {
		t.Parallel()
		tr := mk(t, trkSpeeds(t, 30), moveResp(t, trkWalk(t, 30)))
		tr = trkMove(t, tr, trkC, trkWalk(t, 10))
		after, _ := trkApplied(t)(tr.TakeReaction(trkB, trkAvail, trigger("go")))
		if after.Resources().MovedFeet() != 10 || after.Resources().DashExtraFeet() != 0 {
			t.Errorf("resources = %+v, want MovedFeet 10 [FR-074, FR-067]", after.Resources())
		}
	})
	t.Run("a feature Reaction and a Spell Reaction are only recorded", func(t *testing.T) {
		t.Parallel()
		base := trkABC(t)
		fr := combat.FeatureReaction{Feature: "uncanny_dodge", Granted: combat.GrantsDash{Speed: combat.SpeedNameSwim}}
		tr, ev := trkApplied(t)(base.TakeReaction(trkB, trkAvail, fr))
		if rec := ev[0].(combat.ReactionTaken).Record(); rec != (combat.FeatureReactionRecord{Feature: "uncanny_dodge", Granted: combat.GrantsDash{Speed: combat.SpeedNameSwim}}) || tr.Resources() != base.Resources() {
			t.Errorf("record %+v resources %+v, want the feature recorded with no SpeedUnavailable and no Dash [FR-121, FR-134]", rec, tr.Resources())
		}
		active, _ := trkApplied(t)(base.TakeReaction(trkA, trkAvail, combat.FeatureReaction{Feature: "f", Granted: combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDisengage}}))
		if active.Resources().Disengage() != combat.NotDisengaged {
			t.Error("a feature Reaction marked the active combatant Disengaged [FR-134]")
		}
		_, ev = trkApplied(t)(base.TakeReaction(trkC, trkAvail, combat.SpellReaction{Spell: "counterspell"}))
		if rec := ev[0].(combat.ReactionTaken).Record(); rec != (combat.SpellReactionRecord{Spell: "counterspell"}) {
			t.Errorf("record = %+v [FR-161]", rec)
		}
		if r := trkReason(t)(tr.TakeReaction(trkB, trkAvail, fr)); r != (combat.TurnReactionAlreadyTakenRejection{}) {
			t.Errorf("second: %#v", r)
		}
		active2, _ := trkApplied(t)(base.TakeReaction(trkA, trkAvail, combat.FeatureReaction{Feature: "f", Granted: combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDodge}}))
		if active2.IsDodging(trkA) {
			t.Error("a feature Reaction recorded Dodge [FR-134]")
		}
	})
	t.Run("malformed reactions are invalid input", func(t *testing.T) {
		t.Parallel()
		base := trkABC(t)
		for name, c := range map[string]combat.ReactionChoice{"nil": nil, "no provoker": trkOA("", combat.DepartureBySpeed, combat.VisibilitySeen),
			"no means": trkOA(trkA, 0, combat.VisibilitySeen), "no visibility": trkOA(trkA, combat.DepartureBySpeed, 0),
			"no trigger": combat.ReadiedActionReaction{}, "blank trigger": combat.ReadiedActionReaction{Trigger: " "},
			"feature no id": combat.FeatureReaction{Granted: combat.NoStandardAction{}}, "feature no grant": combat.FeatureReaction{Feature: "f"},
			"spell no id": combat.SpellReaction{}} {
			if _, err := base.TakeReaction(trkB, trkAvail, c); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: %v, want invalid input", name, err)
			}
		}
		if _, err := base.TakeReaction(trkB, 0, combat.SpellReaction{Spell: "s"}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("no availability: %v", err)
		}
	})
}
