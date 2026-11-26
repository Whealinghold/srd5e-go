package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// trkSolo30 is a Medium A with walk 30, fly 40 and climb 20.
func trkSolo30(tb testing.TB) combat.TurnTracker {
	tb.Helper()
	return trkSolo(tb, content.SizeMedium, trkSpeeds(tb, 30, fly(40), climb(20)))
}

func trkClimb(tb testing.TB, feet int, use combat.SpeedUse) combat.MoveSegment {
	tb.Helper()
	return trkSegment(tb, combat.MoveSegmentInput{DistanceFeet: feet, Mode: combat.MovementModeClimb, SpeedUse: use})
}

// Requirement: per case (tables 4.7e and 4.7f, FR-083 to FR-086, FR-089, FR-115)
// Path: per case.
func TestMoveJudgement(t *testing.T) {
	t.Parallel()
	enemyL := combat.Occupant{Size: content.SizeLarge, Relation: combat.OccupantOther, Availability: trkAvail}
	occupiedEnd := combat.MoveSegmentInput{DistanceFeet: 10, Mode: combat.MovementModeWalk, End: combat.SegmentEndOccupiedByCreature}
	cases := []struct {
		name, requirement, path string
		prep                    func(*testing.T, combat.TurnTracker) combat.TurnTracker
		segs                    func(*testing.T) []combat.MoveSegment
		wantMoved               int // when accepted
		want                    combat.TurnRejection
	}{
		{"exactly all 30 ft", "FR-086", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 30)} }, 30, nil},
		{"one under", "FR-086", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 29)} }, 29, nil},
		{"one over", "FR-086", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 31)} }, 0, combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}},
		{"difficult, exact", "FR-086", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 15, Mode: combat.MovementModeWalk, Terrain: combat.TerrainDifficult})}
		}, 30, nil},
		{"dragging a Grappled creature, fits", "FR-077", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 15, Mode: combat.MovementModeWalk, ExtraCostPerFoot: 1})}
		}, 30, nil},
		{"dragging, over", "FR-077", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 16, Mode: combat.MovementModeWalk, ExtraCostPerFoot: 1})}
		}, 0, combat.TurnInsufficientMovementRejection{ShortfallFeet: 2}},
		{"split, fits", "FR-085", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 10), trkWalk(t, 20)} }, 30, nil},
		{"split, the second part too long, the whole move not applied", "FR-086", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 10), trkWalk(t, 21)} }, 0, combat.TurnInsufficientMovementRejection{SegmentIndex: 1, ShortfallFeet: 1}},
		{"mixed modes", "FR-077", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkWalk(t, 10), trkClimb(t, 5, combat.SpeedUseWalking)}
		}, 20, nil},
		{"ends in an occupied space", "FR-081", "unhappy (state error)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkSegment(t, occupiedEnd)} }, 0, combat.TurnEndsInOccupiedSpaceRejection{}},
		{"passes through a Tiny occupant and ends free", "FR-081", "unhappy (state error)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{{Size: content.SizeTiny, Relation: combat.OccupantOther, Availability: trkAvail}}}), trkWalk(t, 5)}
		}, 5 + 5 + 5 - 5, nil},
		{"an earlier segment ending in an occupied space is only passing through", "FR-081", "unhappy (state error)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, occupiedEnd), trkWalk(t, 5)}
		}, 15, nil},
		{"blocked on the way", "FR-080", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{enemyL}})}
		}, 0, combat.TurnBlockedByCreatureRejection{}},
		{"fly after walking; too long", "FR-115", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkWalk(t, 20), trkMode(t, 30, combat.MovementModeFly)}
		}, 0, combat.TurnInsufficientMovementRejection{SegmentIndex: 1, ShortfallFeet: 10}},
		{"fly after walking, fits", "FR-115", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkWalk(t, 10), trkMode(t, 30, combat.MovementModeFly)}
		}, 40, nil},
		{"fly first, then walk", "FR-115", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 30, combat.MovementModeFly), trkWalk(t, 5)}
		}, 0, combat.TurnNoMovementRemainingRejection{SegmentIndex: 1, MovementLeft: 0}},
		{"Climb Speed creature climbs", "FR-115", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkClimb(t, 15, combat.SpeedUseSpecial)}
		}, 15, nil},
		{"Climb Speed, too far", "FR-115", "unhappy (boundary)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkClimb(t, 21, combat.SpeedUseSpecial)}
		}, 0, combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}},
		{"chooses not to use the Climb Speed", "FR-115", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkClimb(t, 10, combat.SpeedUseWalking)}
		}, 20, nil},
		{"walk then climb by the Climb Speed", "FR-115", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkWalk(t, 10), trkClimb(t, 10, combat.SpeedUseSpecial)}
		}, 20, nil},
		{"SRD example: fly 10, walk 10, fly 20", "FR-083", "happy", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 10, combat.MovementModeFly), trkWalk(t, 10), trkMode(t, 20, combat.MovementModeFly)}
		}, 40, nil},
		{"at the limit: fly 1 more", "FR-083", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, tr, trkA, trkMode(t, 40, combat.MovementModeFly))
		}, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 1, combat.MovementModeFly)}
		}, 0, combat.TurnNoMovementRemainingRejection{MovementLeft: 0}},
		{"at the limit: walk 1 more (30 - 40)", "FR-083", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, tr, trkA, trkMode(t, 40, combat.MovementModeFly))
		}, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 1)} }, 0, combat.TurnNoMovementRemainingRejection{MovementLeft: -10}},
		{"walked to the limit, then fly 10", "FR-083", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, tr, trkA, trkWalk(t, 30))
		},
			func(*testing.T) []combat.MoveSegment {
				return []combat.MoveSegment{trkMode(t, 10, combat.MovementModeFly)}
			}, 40, nil},
		{"walked to the limit, then fly 11", "FR-083", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, tr, trkA, trkWalk(t, 30))
		},
			func(*testing.T) []combat.MoveSegment {
				return []combat.MoveSegment{trkMode(t, 11, combat.MovementModeFly)}
			}, 0, combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}},
		{"with Dash: walk 30 after 30", "FR-084", "unhappy (rule interaction)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkTake(t, trkMove(t, tr, trkA, trkWalk(t, 30)), trkA, combat.DashChoice{Speed: combat.SpeedNameWalk})
		}, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 30)} }, 60, nil},
		{"with Dash: fly 1 after walking 60", "FR-084", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, trkTake(t, trkMove(t, tr, trkA, trkWalk(t, 30)), trkA, combat.DashChoice{Speed: combat.SpeedNameWalk}), trkA, trkWalk(t, 30))
		}, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 10, combat.MovementModeFly)}
		}, 70, nil},
		{"with Dash: fly 11 after walking 60", "FR-084", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, trkTake(t, trkMove(t, tr, trkA, trkWalk(t, 30)), trkA, combat.DashChoice{Speed: combat.SpeedNameWalk}), trkA, trkWalk(t, 30))
		}, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 11, combat.MovementModeFly)}
		}, 0, combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}},
		{"reason order: speed before blocked", "FR-118", "unhappy (rule interaction)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			n, _ := trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 30)))
			return n
		}, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 5, combat.MovementModeFly), trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{enemyL}})}
		}, 0, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameFly}},
		{"reason order: blocked before affordability", "FR-118", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 31, Mode: combat.MovementModeWalk, Occupants: []combat.Occupant{enemyL}})}
		}, 0, combat.TurnBlockedByCreatureRejection{}},
		{"reason order: affordability before the occupied end", "FR-118", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 31, Mode: combat.MovementModeWalk, End: combat.SegmentEndOccupiedByCreature})}
		}, 0, combat.TurnInsufficientMovementRejection{ShortfallFeet: 1}},
		{"climb by the Climb Speed without a Climb Speed", "FR-082", "unhappy (invalid input)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			n, _ := trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 30)))
			return n
		}, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkClimb(t, 5, combat.SpeedUseSpecial)}
		}, 0, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameClimb}},
		{"swim by the Swim Speed without a Swim Speed", "FR-082", "unhappy (invalid input)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkSegment(t, combat.MoveSegmentInput{DistanceFeet: 5, Mode: combat.MovementModeSwim, SpeedUse: combat.SpeedUseSpecial})}
		}, 0, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameSwim}},
		{"burrow without a Burrow Speed", "FR-082", "unhappy (invalid input)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 5, combat.MovementModeBurrow)}
		}, 0, combat.TurnSpeedUnavailableRejection{Speed: combat.SpeedNameBurrow}},
		{"a crawl uses the walking Speed", "FR-115", "unhappy (rule interaction)", nil, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 15, combat.MovementModeCrawl)}
		}, 30, nil},
		{"speed lowered to 0 after 10", "FR-116", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			tr = trkMove(t, tr, trkA, trkWalk(t, 10))
			n, _ := trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 0)))
			return n
		}, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 5)} }, 0, combat.TurnNoMovementRemainingRejection{MovementLeft: -10}},
		{"speed lowered to 25 after 10", "FR-116", "unhappy (boundary)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			tr = trkMove(t, tr, trkA, trkWalk(t, 10))
			n, _ := trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 25)))
			return n
		}, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 15)} }, 25, nil},
		{"Speed 0 walking, fly still available", "FR-089", "unhappy (rule interaction)", func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			n, _ := trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 0, fly(40))))
			return n
		}, func(*testing.T) []combat.MoveSegment {
			return []combat.MoveSegment{trkMode(t, 10, combat.MovementModeFly)}
		}, 10, nil},
		{"stale profile: the engine reads only the profile", "FR-117", "unhappy (state error)", nil, func(*testing.T) []combat.MoveSegment { return []combat.MoveSegment{trkWalk(t, 10)} }, 10, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := trkSolo30(t)
			if c.prep != nil {
				tr = c.prep(t, tr)
			}
			segs := c.segs(t)
			before := tr.Resources()
			step, err := tr.Move(trkA, segs)
			if c.want != nil {
				if r := trkReason(t)(step, err); r != c.want {
					t.Fatalf("reason = %#v, want %#v [%s]", r, c.want, c.requirement)
				}
				if tr.Resources() != before {
					t.Error("a refusal changed the tracker [FR-086]")
				}
				return
			}
			after, ev := trkApplied(t)(step, err)
			if got := after.Resources().MovedFeet() - before.MovedFeet(); got != c.wantMoved-before.MovedFeet() && c.wantMoved != 0 {
				t.Errorf("moved %d more, want MovedFeet %d [%s]", got, c.wantMoved, c.requirement)
			}
			if e := ev[0].(combat.MovementSpent); len(e.Charges()) != len(segs) || e.MovedFeet() != after.Resources().MovedFeet() || e.Kind() != "movement_spent" {
				t.Errorf("event = %+v, want one charge per segment [FR-152]", e)
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-088, FR-087, FR-119 - a Move by a combatant that is not active is refused NotActiveCombatant, an unknown id UnknownCombatant, and an empty list or zero segment is an invalid-input error.
// Path: unhappy (state error).
func TestMoveGuards(t *testing.T) {
	t.Parallel()
	tr := trkSolo30(t)
	if r := trkReason(t)(tr.Move(trkB, []combat.MoveSegment{trkWalk(t, 5)})); r != (combat.TurnNotActiveCombatantRejection{}) {
		t.Errorf("B moves: %#v", r)
	}
	if r := trkReason(t)(tr.Move("x", []combat.MoveSegment{trkWalk(t, 5)})); r != (combat.TurnUnknownCombatantRejection{ID: "x"}) {
		t.Errorf("x moves: %#v", r)
	}
	for name, segs := range map[string][]combat.MoveSegment{"nil": nil, "empty": {}, "zero segment": {{}}} {
		if _, err := tr.Move(trkA, segs); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, want invalid input", name, err)
		}
	}
}

// Requirement: per case (table 4.7g, FR-090, FR-091, FR-085, FR-117)
// Path: per case.
func TestChangePosture(t *testing.T) {
	t.Parallel()
	stand := func(cost int) combat.PostureChange {
		return combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: cost}}
	}
	cases := []struct {
		name, requirement, path string
		speeds                  combat.MovementSpeeds
		prep                    func(*testing.T, combat.TurnTracker) combat.TurnTracker
		change                  combat.PostureChange
		id                      combat.CombatantID
		want                    combat.TurnRejection
		wantMoved               int
		kind                    string
	}{
		{"typical stand-up", "FR-091", "happy", trkSpeeds(t, 30), nil, stand(15), trkA, nil, 15, "stood_up"},
		{"rounded down cost", "FR-091", "happy", trkSpeeds(t, 25), nil, stand(12), trkA, nil, 12, "stood_up"},
		{"cost 0", "FR-091", "unhappy (boundary)", trkSpeeds(t, 1), nil, stand(0), trkA, nil, 0, "stood_up"},
		{"CrawlOnly wins over a profile Speed", "FR-091", "unhappy (boundary)", trkSpeeds(t, 30), nil, combat.StandUpChange{Movement: combat.CrawlOnly{}}, trkA, combat.TurnSpeedIsZeroRejection{}, 0, ""},
		{"CrawlOnly with Speed 0", "FR-091", "unhappy (boundary)", trkSpeeds(t, 0), nil, combat.StandUpChange{Movement: combat.CrawlOnly{}}, trkA, combat.TurnSpeedIsZeroRejection{}, 0, ""},
		{"not enough left", "FR-091", "unhappy (boundary)", trkSpeeds(t, 30), func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, tr, trkA, trkWalk(t, 20))
		}, stand(15), trkA, combat.TurnInsufficientMovementRejection{ShortfallFeet: 5}, 0, ""},
		{"exactly enough", "FR-091", "unhappy (boundary)", trkSpeeds(t, 30), func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkMove(t, tr, trkA, trkWalk(t, 15))
		}, stand(15), trkA, nil, 30, "stood_up"},
		{"after Dash the cost stays half of Speed", "FR-091", "unhappy (rule interaction)", trkSpeeds(t, 30), func(t *testing.T, tr combat.TurnTracker) combat.TurnTracker {
			t.Helper()
			return trkTake(t, tr, trkA, combat.DashChoice{Speed: combat.SpeedNameWalk})
		}, stand(15), trkA, nil, 15, "stood_up"},
		{"a stale profile Speed of 0 decides the movement left", "FR-091", "unhappy (state error)", trkSpeeds(t, 0), nil, stand(15), trkA, combat.TurnInsufficientMovementRejection{ShortfallFeet: 15}, 0, ""},
		{"not the active combatant (stand)", "FR-119", "unhappy (state error)", trkSpeeds(t, 30), nil, stand(15), trkB, combat.TurnNotActiveCombatantRejection{}, 0, ""},
		{"not the active combatant (drop)", "FR-119", "unhappy (state error)", trkSpeeds(t, 30), nil, combat.DropProneChange{}, trkB, combat.TurnNotActiveCombatantRejection{}, 0, ""},
		{"unknown id", "FR-119", "unhappy (state error)", trkSpeeds(t, 30), nil, combat.DropProneChange{}, "x", combat.TurnUnknownCombatantRejection{ID: "x"}, 0, ""},
		{"drop Prone", "FR-090", "happy", trkSpeeds(t, 30), nil, combat.DropProneChange{}, trkA, nil, 0, "dropped_prone"},
		{"drop Prone with Speed 0", "FR-090", "happy", trkSpeeds(t, 0), nil, combat.DropProneChange{}, trkA, combat.TurnSpeedIsZeroRejection{}, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := trkSolo(t, content.SizeMedium, c.speeds)
			if c.prep != nil {
				tr = c.prep(t, tr)
			}
			before := tr.Resources()
			step, err := tr.ChangePosture(c.id, c.change)
			if c.want != nil {
				if r := trkReason(t)(step, err); r != c.want || tr.Resources() != before {
					t.Errorf("reason = %#v, want %#v [%s]", r, c.want, c.requirement)
				}
				return
			}
			after, ev := trkApplied(t)(step, err)
			if len(ev) != 1 || ev[0].Kind() != c.kind || after.Resources().MovedFeet() != c.wantMoved || after.Resources().Action() != before.Action() {
				t.Errorf("events %v moved %d, want %s and MovedFeet %d with no action spent [%s]", trkKinds(ev), after.Resources().MovedFeet(), c.kind, c.wantMoved, c.requirement)
			}
			if su, ok := ev[0].(combat.StoodUp); ok && (su.CostFeet() != c.wantMoved-before.MovedFeet() || su.MovedFeet() != c.wantMoved) {
				t.Errorf("StoodUp = %+v", su)
			}
		})
	}
	t.Run("the cost is spent: later moves draw on it", func(t *testing.T) {
		t.Parallel()
		tr := trkSolo(t, content.SizeMedium, trkSpeeds(t, 30))
		after, _ := trkApplied(t)(tr.ChangePosture(trkA, stand(15)))
		after = trkMove(t, after, trkA, trkWalk(t, 15))
		if r := trkReason(t)(after.Move(trkA, []combat.MoveSegment{trkWalk(t, 1)})); r != (combat.TurnNoMovementRemainingRejection{}) {
			t.Errorf("reason = %#v [FR-085, FR-091]", r)
		}
	})
	t.Run("speed changed mid-turn, the host supplies the new value", func(t *testing.T) {
		t.Parallel()
		tr := trkSolo(t, content.SizeMedium, trkSpeeds(t, 30))
		tr, _ = trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 15)))
		trkApplied(t)(tr.ChangePosture(trkA, stand(7)))
		tr, _ = trkApplied(t)(tr.UpdateMovementSpeeds(trkA, trkSpeeds(t, 0)))
		if r := trkReason(t)(tr.ChangePosture(trkA, combat.StandUpChange{Movement: combat.CrawlOnly{}})); r != (combat.TurnSpeedIsZeroRejection{}) {
			t.Errorf("reason = %#v [FR-091]", r)
		}
	})
	t.Run("malformed changes are invalid input", func(t *testing.T) {
		t.Parallel()
		tr := trkSolo(t, content.SizeMedium, trkSpeeds(t, 30))
		for name, c := range map[string]combat.PostureChange{"nil": nil, "no movement": combat.StandUpChange{}, "negative cost": stand(-1)} {
			if _, err := tr.ChangePosture(trkA, c); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: %v, want invalid input", name, err)
			}
		}
	})
}

// Requirement: per case (table 4.11a, FR-155)
// Path: per case.
func TestAdditionalAction(t *testing.T) {
	t.Parallel()
	anyBut := func(a ...combat.StandardAction) combat.AdditionalActionGrant {
		s, err := combat.NewAnyActionBut(a...)
		if err != nil {
			t.Fatal(err)
		}
		g, err := combat.NewAdditionalActionGrant("action_surge", s)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	only := func(a ...combat.StandardAction) combat.AdditionalActionGrant {
		s, err := combat.NewOnlyActions(a...)
		if err != nil {
			t.Fatal(err)
		}
		g, err := combat.NewAdditionalActionGrant("haste", s)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	magic := combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}
	hide := combat.SimpleActionChoice{Action: combat.SimpleStandardActionHide}
	dash := combat.DashChoice{Speed: combat.SpeedNameWalk}
	improvised := combat.ImprovisedActionChoice{}
	cases := []struct {
		name  string
		grant combat.AdditionalActionGrant
		seq   []combat.ActionChoice
		want  []bool // accepted?
	}{
		{"Action Surge: Attack then Magic", anyBut(combat.StandardActionMagic), []combat.ActionChoice{trkAttack, magic}, []bool{true, true}},
		{"Action Surge: Magic then Attack", anyBut(combat.StandardActionMagic), []combat.ActionChoice{magic, trkAttack}, []bool{true, true}},
		{"Action Surge: Magic then Magic", anyBut(combat.StandardActionMagic), []combat.ActionChoice{magic, magic}, []bool{true, false}},
		{"Action Surge: Attack, Attack, Dodge", anyBut(combat.StandardActionMagic), []combat.ActionChoice{trkAttack, trkAttack, trkDodge}, []bool{true, true, false}},
		{"Action Surge permits an improvised action", anyBut(combat.StandardActionMagic), []combat.ActionChoice{trkAttack, improvised}, []bool{true, true}},
		{"Haste: Magic then Dodge", only(combat.StandardActionAttack, combat.StandardActionDash, combat.StandardActionDisengage, combat.StandardActionHide, combat.StandardActionUtilize), []combat.ActionChoice{magic, trkDodge}, []bool{true, false}},
		{"Haste: Dodge then Dash", only(combat.StandardActionAttack, combat.StandardActionDash, combat.StandardActionDisengage, combat.StandardActionHide, combat.StandardActionUtilize), []combat.ActionChoice{trkDodge, dash}, []bool{true, true}},
		{"Haste: improvised then Attack", only(combat.StandardActionAttack, combat.StandardActionDash, combat.StandardActionDisengage, combat.StandardActionHide, combat.StandardActionUtilize), []combat.ActionChoice{improvised, trkAttack}, []bool{true, true}},
		{"Haste: Hide then Hide", only(combat.StandardActionHide), []combat.ActionChoice{hide, hide}, []bool{true, true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, ev := trkApplied(t)(trkABC(t).GrantAdditionalAction(trkA, c.grant))
			if len(ev) != 1 || ev[0].Kind() != "additional_action_granted" || !ev[0].(combat.AdditionalActionGranted).Grant().Equal(c.grant) {
				t.Fatalf("events = %v [FR-155]", trkKinds(ev))
			}
			for i, choice := range c.seq {
				step, err := tr.TakeAction(trkA, trkAvail, choice)
				if c.want[i] {
					tr, _ = trkApplied(t)(step, err)
					continue
				}
				if r := trkReason(t)(step, err); r != (combat.TurnActionAlreadyTakenRejection{}) {
					t.Errorf("choice %d: %#v, want ActionAlreadyTaken [FR-155]", i, r)
				}
			}
		})
	}
	t.Run("grant refusals and clearing", func(t *testing.T) {
		t.Parallel()
		g := anyBut()
		tr, _ := trkApplied(t)(trkABC(t).GrantAdditionalAction(trkA, g))
		if gr, used := tr.AdditionalAction(); !gr.Equal(g) || used != combat.ResourceAvailable {
			t.Errorf("AdditionalAction = %+v, %v", gr, used)
		}
		for _, c := range []struct {
			name string
			tr   combat.TurnTracker
			id   combat.CombatantID
			want combat.TurnRejection
		}{
			{"second grant", tr, trkA, combat.TurnAdditionalActionAlreadyGrantedRejection{}},
			{"not active", trkABC(t), trkB, combat.TurnNotActiveCombatantRejection{}},
			{"unknown", trkABC(t), "x", combat.TurnUnknownCombatantRejection{ID: "x"}},
		} {
			if r := trkReason(t)(c.tr.GrantAdditionalAction(c.id, g)); r != c.want {
				t.Errorf("%s: %#v, want %#v [FR-118]", c.name, r, c.want)
			}
		}
		if _, err := trkABC(t).GrantAdditionalAction(trkA, combat.AdditionalActionGrant{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero grant: %v, want invalid input", err)
		}
		next := advN(t, tr, 3)
		if gr, _ := next.AdditionalAction(); !gr.IsZero() {
			t.Error("the grant survived the turn [FR-040, FR-034]")
		}
		trkApplied(t)(next.GrantAdditionalAction(trkA, g))
	})
	t.Run("Ready with the extra action: only one readied action is held", func(t *testing.T) {
		t.Parallel()
		ready := combat.ReadyChoice{Action: trkReadied(t, "t", combat.ReadiedActionResponse{Choice: trkAttack})}
		tr := trkTake(t, trkABC(t), trkA, ready)
		tr, _ = trkApplied(t)(tr.GrantAdditionalAction(trkA, anyBut()))
		if r := trkReason(t)(tr.TakeAction(trkA, trkAvail, ready)); r != (combat.TurnActionAlreadyTakenRejection{}) {
			t.Errorf("second Ready: %#v [FR-056]", r)
		}
	})
	t.Run("the additional action does not need the combatant to be able, but taking the action does", func(t *testing.T) {
		t.Parallel()
		tr, _ := trkApplied(t)(trkABC(t).GrantAdditionalAction(trkA, anyBut()))
		if r := trkReason(t)(tr.TakeAction(trkA, trkNone, trkAttack)); r != (combat.TurnIncapacitatedRejection{}) {
			t.Errorf("incapacitated: %#v [FR-155]", r)
		}
	})
}
