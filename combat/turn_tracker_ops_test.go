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

// trkSolo is a tracker of A with the given speeds, then B and C with walk 30.
func trkSolo(tb testing.TB, size content.CreatureSize, a combat.MovementSpeeds) combat.TurnTracker {
	tb.Helper()
	return trkTracker(tb, trkProfile(tb, trkA, size, a),
		trkProfile(tb, trkB, content.SizeMedium, trkSpeeds(tb, 30)), trkProfile(tb, trkC, content.SizeMedium, trkSpeeds(tb, 30)))
}

var (
	trkAttack  = combat.SimpleActionChoice{Action: combat.SimpleStandardActionAttack}
	trkDodge   = combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}
	trkDisenga = combat.SimpleActionChoice{Action: combat.SimpleStandardActionDisengage}
)

// Requirement: per case (table 4.4a, FR-045 to FR-048, FR-118, FR-119)
// Path: per case.
func TestTakeActionRefusals(t *testing.T) {
	t.Parallel()
	base := trkSolo(t, content.SizeMedium, trkSpeeds(t, 30))
	spent := trkTake(t, base, trkA, trkAttack)
	cases := []struct {
		name, requirement, path string
		tr                      combat.TurnTracker
		id                      combat.CombatantID
		av                      combat.ConditionActionAvailability
		choice                  combat.ActionChoice
		want                    combat.TurnRejection
	}{
		{"incapacitated", "FR-047", "unhappy (state error)", base, trkA, trkNone, trkAttack, combat.TurnIncapacitatedRejection{}},
		{"incapacitated beats not active and spent", "FR-048", "unhappy (rule interaction)", spent, trkA, trkNone, trkAttack, combat.TurnIncapacitatedRejection{}},
		{"unknown beats every other reason", "FR-118", "unhappy (rule interaction)", spent, "x", trkNone, trkAttack, combat.TurnUnknownCombatantRejection{ID: "x"}},
		{"not active beats nothing else but spent", "FR-046", "unhappy (state error)", spent, trkB, trkAvail, trkAttack, combat.TurnNotActiveCombatantRejection{}},
		{"active and spent", "FR-045", "unhappy (state error)", spent, trkA, trkAvail, trkDodge, combat.TurnActionAlreadyTakenRejection{}},
		{"Dash by a speed not owned", "FR-052", "unhappy (invalid input)", base, trkA, trkAvail, combat.DashChoice{Speed: combat.SpeedNameFly}, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameFly}},
		{"action spent beats an unowned speed", "FR-118", "unhappy (rule interaction)", spent, trkA, trkAvail, combat.DashChoice{Speed: combat.SpeedNameFly}, combat.TurnActionAlreadyTakenRejection{}},
		{"feature Dash by an unowned speed", "FR-112", "unhappy (invalid input)", base, trkA, trkAvail, combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsDash{Speed: combat.SpeedNameSwim}}, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameSwim}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			snapshot, _ := trkApplied(t)(c.tr.UpdateMovementSpeeds(trkA, mustProfileSpeeds(t, c.tr, trkA)))
			if r := trkReason(t)(c.tr.TakeAction(c.id, c.av, c.choice)); r != c.want {
				t.Errorf("reason = %#v, want %#v [%s]", r, c.want, c.requirement)
			}
			after, _ := trkApplied(t)(c.tr.UpdateMovementSpeeds(trkA, mustProfileSpeeds(t, c.tr, trkA)))
			if !snapshot.Equal(after) {
				t.Error("a refusal changed the tracker [FR-118]")
			}
		})
	}
	// An able, active combatant with an unused action takes it.
	if _, ev := trkApplied(t)(base.TakeAction(trkA, trkAvail, trkAttack)); !slices.Equal(trkKinds(ev), []string{"action_taken"}) {
		t.Errorf("events = %v", trkKinds(ev))
	}
}

func mustProfileSpeeds(tb testing.TB, t combat.TurnTracker, id combat.CombatantID) combat.MovementSpeeds {
	tb.Helper()
	p, _ := t.ProfileOf(id)
	return p.Speeds
}

// Requirement: doc/specs/initiative-and-actions.md FR-045, FR-049, FR-050, FR-053, FR-054, FR-056 - every choice consumes the action and is reported in ActionTaken; Dodge and Disengage record their state; Ready records the readied action; Attack and Help and the rest resolve nothing here.
// Path: per case.
func TestTakeActionEffects(t *testing.T) {
	t.Parallel()
	for s := combat.SimpleStandardAction(1); s.Valid(); s++ {
		t.Run(s.String(), func(t *testing.T) {
			t.Parallel()
			tr, ev := trkApplied(t)(trkABC(t).TakeAction(trkA, trkAvail, combat.SimpleActionChoice{Action: s}))
			if len(ev) != 1 || ev[0].(combat.ActionTaken).Choice() != (combat.SimpleActionChoice{Action: s}) || tr.Resources().Action() != combat.ResourceSpent ||
				ev[0].Kind() != "action_taken" || ev[0].Combatants()[0] != trkA {
				t.Errorf("events = %+v, resources %+v [FR-050]", ev, tr.Resources())
			}
			if wantDodge, wantDis := s == combat.SimpleStandardActionDodge, s == combat.SimpleStandardActionDisengage; tr.IsDodging(trkA) != wantDodge || (tr.Resources().Disengage() == combat.Disengaged) != wantDis {
				t.Errorf("dodging %v disengaged %v [FR-053, FR-054]", tr.IsDodging(trkA), tr.Resources().Disengage())
			}
			if tr.Resources().MovedFeet() != 0 || tr.Resources().DashExtraFeet() != 0 {
				t.Errorf("an action other than Dash moved or added movement: %+v", tr.Resources())
			}
		})
	}
	t.Run("feature and improvised actions consume the action", func(t *testing.T) {
		t.Parallel()
		for _, c := range []combat.ActionChoice{combat.ImprovisedActionChoice{}, combat.FeatureActionChoice{Feature: "f", Granted: combat.NoStandardAction{}},
			combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsSimpleAction{Action: combat.SimpleStandardActionHide}}} {
			tr, ev := trkApplied(t)(trkABC(t).TakeAction(trkA, trkAvail, c))
			if tr.Resources().Action() != combat.ResourceSpent || ev[0].(combat.ActionTaken).Choice() != c {
				t.Errorf("choice %#v: %+v [FR-049]", c, tr.Resources())
			}
		}
	})
	t.Run("Ready records the readied action and a second is refused", func(t *testing.T) {
		t.Parallel()
		ready := combat.ReadyChoice{Action: trkReadied(t, "door", combat.ReadiedActionResponse{Choice: trkAttack})}
		tr := trkTake(t, trkABC(t), trkA, ready)
		got, ok := tr.ReadiedActionOf(trkA)
		if !ok || got.Trigger() != "door" || !got.Equal(ready.Action) {
			t.Errorf("readied = %+v, %v [FR-056]", got, ok)
		}
		if r := trkReason(t)(tr.TakeAction(trkA, trkAvail, ready)); r != (combat.TurnActionAlreadyTakenRejection{}) {
			t.Errorf("second Ready: %#v [FR-056]", r)
		}
	})
	t.Run("Ready accepts a move that is illegal now", func(t *testing.T) {
		t.Parallel()
		mv, err := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 31)})
		if err != nil {
			t.Fatal(err)
		}
		tr := trkTake(t, trkABC(t), trkA, combat.ReadyChoice{Action: trkReadied(t, "t", mv)})
		if a, ok := tr.ReadiedActionOf(trkA); !ok || len(a.Response().(combat.ReadiedMoveResponse).Segments()) != 1 {
			t.Errorf("readied = %+v [FR-057]", a)
		}
	})
}

// Requirement: doc/specs/initiative-and-actions.md FR-059, FR-058, FR-085 - movement, the action, the Bonus Action and the free interaction may come in any order and a move may be split around them; the free interaction is once a turn.
// Path: happy.
func TestActionsInAnyOrder(t *testing.T) {
	t.Parallel()
	tr := trkMove(t, trkSolo(t, content.SizeMedium, trkSpeeds(t, 30)), trkA, trkWalk(t, 10))
	tr, ev := trkApplied(t)(tr.InteractWithObject(trkA))
	if ev[0].Kind() != "object_interacted" || ev[0].Combatants()[0] != trkA {
		t.Errorf("event = %+v", ev[0])
	}
	tr = trkTake(t, tr, trkA, trkAttack)
	tr = trkMove(t, tr, trkA, trkWalk(t, 20))
	if r := trkReason(t)(tr.InteractWithObject(trkA)); r != (combat.TurnNeedsUtilizeActionRejection{}) {
		t.Errorf("second interaction: %#v [FR-058]", r)
	}
	if r := trkReason(t)(tr.InteractWithObject(trkB)); r != (combat.TurnNotActiveCombatantRejection{}) {
		t.Errorf("B interacts: %#v [FR-119]", r)
	}
	if r := trkReason(t)(tr.InteractWithObject("x")); r != (combat.TurnUnknownCombatantRejection{ID: "x"}) {
		t.Errorf("x interacts: %#v [FR-119]", r)
	}
	if r := trkReason(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, 1)})); r != (combat.TurnNoMovementRemainingRejection{MovementLeft: 0}) {
		t.Errorf("a move after 30 ft: %#v", r)
	}
}

// Requirement: per case (table 4.4b, FR-051, FR-052, FR-084, FR-115)
// Path: per case.
func TestDashAddsTheSpeed(t *testing.T) {
	t.Parallel()
	walkFly := trkSpeeds(t, 30, fly(40))
	cases := []struct {
		name, path   string
		speeds       combat.MovementSpeeds
		movedFirst   int
		dash         combat.SpeedName
		walkable     int // the longest walking move accepted afterwards
		wantRejected bool
	}{
		{"ordinary", "happy", trkSpeeds(t, 30), 0, combat.SpeedNameWalk, 60, false},
		{"odd", "happy", trkSpeeds(t, 25), 0, combat.SpeedNameWalk, 50, false},
		{"reduced", "happy", trkSpeeds(t, 15), 0, combat.SpeedNameWalk, 30, false},
		{"Speed 0", "unhappy (boundary)", trkSpeeds(t, 0), 0, combat.SpeedNameWalk, 0, false},
		{"after moving 10", "unhappy (rule interaction)", trkSpeeds(t, 30), 10, combat.SpeedNameWalk, 60, false},
		{"chosen Fly speed", "unhappy (rule interaction)", walkFly, 0, combat.SpeedNameFly, 70, false},
		{"speed not owned", "unhappy (invalid input)", trkSpeeds(t, 30), 0, combat.SpeedNameFly, 30, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := trkSolo(t, content.SizeMedium, c.speeds)
			if c.movedFirst > 0 {
				tr = trkMove(t, tr, trkA, trkWalk(t, c.movedFirst))
			}
			dash := combat.DashChoice{Speed: c.dash}
			step, err := tr.TakeAction(trkA, trkAvail, dash)
			if c.wantRejected {
				if r := trkReason(t)(step, err); r != (combat.TurnSpeedUnavailableRejection{Speed: c.dash}) {
					t.Fatalf("reason = %#v", r)
				}
				if tr.Resources().Action() != combat.ResourceAvailable || tr.Resources().DashExtraFeet() != 0 {
					t.Error("a refused Dash spent the action")
				}
			} else {
				tr, _ = trkApplied(t)(step, err)
			}
			left := c.walkable - c.movedFirst
			if c.walkable == 0 {
				left = 0
			}
			if left > 0 {
				trkApplied(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, left)}))
			}
			if r := trkReason(t)(tr.Move(trkA, []combat.MoveSegment{trkWalk(t, left+1)})); r == nil {
				t.Errorf("a walk of %d ft was accepted, want the pool to end at %d [FR-051, FR-084]", left+1, left)
			}
		})
	}
}

// Requirement: per case (table 4.4c, FR-055, FR-117)
// Path: per case.
func TestDodgeBenefits(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		dodging    bool
		av         combat.ConditionActionAvailability
		speed      int
		stale      bool // set Speed 0 after Dodge
		want       combat.DodgeBenefits
	}{
		{"active", "happy", true, trkAvail, 30, false, combat.DodgeBenefitsActive},
		{"Speed 1", "happy", true, trkAvail, 1, false, combat.DodgeBenefitsActive},
		{"incapacitated", "unhappy (rule interaction)", true, trkNone, 30, false, combat.DodgeBenefitsNone},
		{"Speed 0", "unhappy (rule interaction)", true, trkAvail, 0, false, combat.DodgeBenefitsNone},
		{"both lost", "unhappy (rule interaction)", true, trkNone, 0, false, combat.DodgeBenefitsNone},
		{"not dodging", "unhappy (empty case)", false, trkAvail, 30, false, combat.DodgeBenefitsNone},
		{"Speed set to 0 mid-turn", "unhappy (rule interaction)", true, trkAvail, 30, true, combat.DodgeBenefitsNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := trkSolo(t, content.SizeMedium, trkSpeeds(t, c.speed))
			if c.dodging {
				tr = trkTake(t, tr, trkA, trkDodge)
			}
			if c.stale {
				tr, _ = trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 0)))
			}
			if got, err := tr.DodgeBenefits(trkA, c.av); err != nil || got != c.want {
				t.Errorf("DodgeBenefits = %v, %v; want %v [FR-055]", got, err, c.want)
			}
		})
	}
	// The engine reads only the profile: a stale Speed of 30 keeps the benefits.
	tr := trkTake(t, trkSolo(t, content.SizeMedium, trkSpeeds(t, 30)), trkA, trkDodge)
	if got, _ := tr.DodgeBenefits(trkA, trkAvail); got != combat.DodgeBenefitsActive || got.String() != "active" {
		t.Errorf("stale profile: %v [FR-117]", got)
	}
}

// Requirement: per case (table 4.5a, FR-060 to FR-063, FR-112 to FR-114, FR-158)
// Path: per case.
func TestBonusActions(t *testing.T) {
	t.Parallel()
	grant := func(source combat.BonusActionSource, g combat.GrantedAction) combat.BonusActionEntitlement {
		bg, err := combat.NewBonusActionGrant(source, g)
		if err != nil {
			t.Fatal(err)
		}
		return combat.EntitledToBonusAction{Grant: bg}
	}
	feat := combat.BonusActionFromFeature{Feature: "cunning_action"}
	base := trkSolo(t, content.SizeMedium, trkWalkFly(t))
	t.Run("Dash by Bonus Action leaves the action and adds the speed", func(t *testing.T) {
		t.Parallel()
		tr, ev := trkApplied(t)(base.TakeBonusAction(trkA, trkAvail, grant(feat, combat.GrantsDash{Speed: combat.SpeedNameWalk})))
		if tr.Resources().Action() != combat.ResourceAvailable || tr.Resources().BonusAction() != combat.ResourceSpent || tr.Resources().DashExtraFeet() != 30 {
			t.Errorf("resources = %+v [FR-062, FR-112]", tr.Resources())
		}
		if e := ev[0].(combat.BonusActionUsed); e.Kind() != "bonus_action_used" || e.Source() != feat || e.Granted() != (combat.GrantsDash{Speed: combat.SpeedNameWalk}) || e.Combatants()[0] != trkA {
			t.Errorf("event = %+v [FR-063]", e)
		}
		tr = trkTake(t, tr, trkA, combat.DashChoice{Speed: combat.SpeedNameWalk})
		if tr.Resources().DashExtraFeet() != 60 {
			t.Errorf("Dash extra after both = %d, want 60 [FR-112]", tr.Resources().DashExtraFeet())
		}
	})
	t.Run("Dash by a speed not owned", func(t *testing.T) {
		t.Parallel()
		if r := trkReason(t)(base.TakeBonusAction(trkA, trkAvail, grant(feat, combat.GrantsDash{Speed: combat.SpeedNameSwim}))); r != (combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameSwim}) {
			t.Errorf("reason = %#v [FR-112]", r)
		}
	})
	t.Run("Disengage by Bonus Action marks Disengaged for this turn only", func(t *testing.T) {
		t.Parallel()
		tr, _ := trkApplied(t)(base.TakeBonusAction(trkA, trkAvail, grant(feat, combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDisengage})))
		if tr.Resources().Disengage() != combat.Disengaged || tr.Resources().Action() != combat.ResourceAvailable {
			t.Errorf("resources = %+v [FR-113]", tr.Resources())
		}
		if r := trkReason(t)(tr.TakeReaction(trkB, trkAvail, combat.OpportunityAttackReaction{Provoker: trkA, Means: combat.DepartureBySpeed, Visibility: combat.VisibilitySeen})); r != (combat.TurnProvokerDisengagedRejection{}) {
			t.Errorf("Opportunity Attack: %#v, want ProvokerDisengaged [FR-069]", r)
		}
		tr, _ = trkAdvance(t, tr)
		if tr.Resources().Disengage() != combat.NotDisengaged {
			t.Error("the next turn starts Disengaged [FR-039]")
		}
	})
	t.Run("Dodge by Bonus Action, then Hide by Bonus Action is refused", func(t *testing.T) {
		t.Parallel()
		tr, _ := trkApplied(t)(base.TakeBonusAction(trkA, trkAvail, grant(feat, combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDodge})))
		if !tr.IsDodging(trkA) {
			t.Error("not dodging [FR-113]")
		}
		other := grant(combat.BonusActionFromFeature{Feature: "other"}, combat.NoStandardAction{})
		if r := trkReason(t)(tr.TakeBonusAction(trkA, trkAvail, other)); r != (combat.TurnBonusActionAlreadyTakenRejection{}) {
			t.Errorf("second Bonus Action with a different grant: %#v [FR-061]", r)
		}
	})
	t.Run("Hide by Bonus Action resolves nothing", func(t *testing.T) {
		t.Parallel()
		tr, ev := trkApplied(t)(base.TakeBonusAction(trkA, trkAvail, grant(feat, combat.GrantsSimpleAction{Action: combat.SimpleStandardActionHide})))
		if ev[0].(combat.BonusActionUsed).Granted() != (combat.GrantsSimpleAction{Action: combat.SimpleStandardActionHide}) || tr.IsDodging(trkA) || tr.Resources().Disengage() == combat.Disengaged || tr.Resources().Action() != combat.ResourceAvailable {
			t.Errorf("state = %+v [FR-114]", tr.Resources())
		}
	})
	t.Run("a Bonus Action from an item", func(t *testing.T) {
		t.Parallel()
		src := combat.BonusActionFromItem{Item: "potion_of_healing"}
		tr, ev := trkApplied(t)(base.TakeBonusAction(trkA, trkAvail, grant(src, combat.NoStandardAction{})))
		if ev[0].(combat.BonusActionUsed).Source() != src || tr.Resources().BonusAction() != combat.ResourceSpent {
			t.Errorf("event = %+v [FR-158]", ev[0])
		}
	})
	t.Run("a Disengage by feature action consumes the action", func(t *testing.T) {
		t.Parallel()
		tr := trkTake(t, base, trkA, combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsSimpleAction{Action: combat.SimpleStandardActionDisengage}})
		if tr.Resources().Disengage() != combat.Disengaged || tr.Resources().Action() != combat.ResourceSpent {
			t.Errorf("resources = %+v [FR-049, FR-113]", tr.Resources())
		}
	})
	t.Run("refusals in order", func(t *testing.T) {
		t.Parallel()
		g := grant(feat, combat.NoStandardAction{})
		cases := []struct {
			name string
			tr   combat.TurnTracker
			id   combat.CombatantID
			av   combat.ConditionActionAvailability
			e    combat.BonusActionEntitlement
			want combat.TurnRejection
		}{
			{"no grant", base, trkA, trkAvail, combat.NotEntitledToBonusAction{}, combat.TurnNoBonusActionGrantedRejection{}},
			{"no grant and not active", base, trkB, trkAvail, combat.NotEntitledToBonusAction{}, combat.TurnNotActiveCombatantRejection{}},
			{"incapacitated", base, trkA, trkNone, g, combat.TurnIncapacitatedRejection{}},
			{"unknown", base, "x", trkNone, g, combat.TurnUnknownCombatantRejection{ID: "x"}},
			{"not active", base, trkB, trkAvail, g, combat.TurnNotActiveCombatantRejection{}},
			{"already taken", func() combat.TurnTracker { n, _ := trkApplied(t)(base.TakeBonusAction(trkA, trkAvail, g)); return n }(), trkA, trkAvail, g, combat.TurnBonusActionAlreadyTakenRejection{}},
		}
		for _, c := range cases {
			if r := trkReason(t)(c.tr.TakeBonusAction(c.id, c.av, c.e)); r != c.want {
				t.Errorf("%s: %#v, want %#v [FR-118]", c.name, r, c.want)
			}
		}
		for name, e := range map[string]combat.BonusActionEntitlement{"nil": nil, "zero grant": combat.EntitledToBonusAction{}} {
			if _, err := base.TakeBonusAction(trkA, trkAvail, e); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s entitlement: %v, want invalid input", name, err)
			}
		}
		if _, err := base.TakeBonusAction(trkA, 0, g); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("no availability: %v, want invalid input", err)
		}
	})
}

// Requirement: doc/specs/initiative-and-actions.md FR-046, FR-069, FR-072, FR-157 - the checks of the active combatant, the Disengaged provoker, the readied trigger and the until-next-turn window hold whichever way the ids sort.
// Path: unhappy (boundary).
func TestOperationsByIDOrder(t *testing.T) {
	t.Parallel()
	atB := advN(t, trkABC(t), 1) // B is active; A sorts before it
	if r := trkReason(t)(atB.TakeAction(trkA, trkAvail, trkAttack)); r != (combat.TurnNotActiveCombatantRejection{}) {
		t.Errorf("A acts on B's turn: %#v, want NotActiveCombatant [FR-046]", r)
	}
	if r := trkReason(t)(atB.Move(trkA, []combat.MoveSegment{trkWalk(t, 5)})); r != (combat.TurnNotActiveCombatantRejection{}) {
		t.Errorf("A moves on B's turn: %#v, want NotActiveCombatant [FR-088]", r)
	}
	// B, the active combatant, Disengaged: an Opportunity Attack against A (not active) is accepted, against B refused.
	disengaged := trkTake(t, atB, trkB, trkDisenga)
	if _, ok := mustApplied(disengaged.TakeReaction(trkC, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilitySeen))); !ok {
		t.Error("an Opportunity Attack against A, who is not the active combatant, was refused [FR-069]")
	}
	if r := trkReason(t)(disengaged.TakeReaction(trkC, trkAvail, trkOA(trkB, combat.DepartureBySpeed, combat.VisibilitySeen))); r != (combat.TurnProvokerDisengagedRejection{}) {
		t.Errorf("against the Disengaged B: %#v [FR-069]", r)
	}
	// A readied trigger that sorts after the held one is not matched.
	ready := combat.ReadyChoice{Action: trkReadied(t, "go", combat.ReadiedActionResponse{Choice: trkAttack})}
	held := trkTake(t, trkABC(t), trkA, ready)
	for _, trig := range []string{"gp", "gn", "g", "goo"} {
		if r := trkReason(t)(held.TakeReaction(trkA, trkAvail, combat.ReadiedActionReaction{Trigger: combat.ReadyTriggerID(trig)})); r != (combat.TurnNoMatchingReadiedActionRejection{Trigger: combat.ReadyTriggerID(trig)}) {
			t.Errorf("trigger %q: %#v, want NoMatchingReadiedAction [FR-072]", trig, r)
		}
	}
	// A's until-next-turn use survives B's turn start and ends at A's own.
	until := character.OnceUntilStartOfYourNextTurn{}
	tr, _ := trkApplied(t)(trkABC(t).RecordFeatUse(trkA, "peerless_aim", until))
	tr, _ = trkAdvance(t, tr) // B's turn starts
	if r := trkReason(t)(tr.RecordFeatUse(trkA, "peerless_aim", until)); r != (combat.TurnFeatUseLimitedRejection{Feat: "peerless_aim"}) {
		t.Errorf("A's use after B's turn started: %#v, want FeatUseLimited [FR-157]", r)
	}
	// An unknown provoker or reactor that sorts around the known ids is unknown.
	for _, id := range []combat.CombatantID{"A", "z", "0"} {
		if r := trkReason(t)(trkABC(t).TakeReaction(trkB, trkAvail, trkOA(id, combat.DepartureBySpeed, combat.VisibilitySeen))); r != (combat.TurnUnknownCombatantRejection{ID: id}) {
			t.Errorf("provoker %q: %#v, want UnknownCombatant", id, r)
		}
		if r := trkReason(t)(trkABC(t).TakeAction(id, trkAvail, trkAttack)); r != (combat.TurnUnknownCombatantRejection{ID: id}) {
			t.Errorf("actor %q: %#v, want UnknownCombatant", id, r)
		}
		if r := trkReason(t)(trkABC(t).RecordFeatUse(id, "f", character.OncePerTurn{})); r != (combat.TurnUnknownCombatantRejection{ID: id}) {
			t.Errorf("feat user %q: %#v, want UnknownCombatant", id, r)
		}
	}
}

func mustApplied(s combat.TrackerStep, err error) (combat.TrackerStepApplied, bool) {
	a, ok := s.(combat.TrackerStepApplied)
	return a, ok && err == nil
}
