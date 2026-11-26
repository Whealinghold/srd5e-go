package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// trkUnpack reads the tracker, events and reason out of any turn use case result.
func trkUnpack(t *testing.T, res any) (combat.TurnTracker, []combat.TurnEvent, combat.TurnRejection) {
	t.Helper()
	if a, ok := res.(interface {
		Tracker() combat.TurnTracker
		Events() []combat.TurnEvent
	}); ok {
		return a.Tracker(), a.Events(), nil
	}
	f := reflect.ValueOf(res).FieldByName("Reason")
	if !f.IsValid() {
		t.Fatalf("result %#v is neither applied nor rejected", res)
	}
	return combat.TurnTracker{}, nil, f.Interface().(combat.TurnRejection)
}

type trkUseRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	direct                  func() (combat.TrackerStep, error)
	class, result           string
	events                  []string
	moved                   map[string]float64 // movement mode to the sample of turn_movement_spent
}

func trkUseRows(t *testing.T) []trkUseRow {
	t.Helper()
	ctx := context.Background()
	base := trkSolo30(t)
	spec := "doc/specs/initiative-and-actions.md "
	attack := trkAttack
	attackTaken := trkTake(t, base, trkA, attack)
	grant, _ := combat.NewAdditionalActionGrant("surge", combat.AnyActionBut{})
	bg, _ := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "b"}, combat.NoStandardAction{})
	segs := []combat.MoveSegment{trkWalk(t, 10), trkMode(t, 5, combat.MovementModeFly), trkWalk(t, 5)}
	oa := trkOA(trkA, combat.DepartureBySpeed, combat.VisibilitySeen)
	resp, _ := combat.NewReadiedMoveResponse([]combat.MoveSegment{trkWalk(t, 10), trkMode(t, 5, combat.MovementModeFly)})
	readyMoveTr := func() combat.TurnTracker {
		tr := trkTracker(t, trkProfile(t, trkA, content.SizeMedium, trkSpeeds(t, 30)), trkProfile(t, trkB, content.SizeMedium, trkWalkFly(t)))
		tr, _ = trkAdvance(t, tr)
		tr = trkTake(t, tr, trkB, combat.ReadyChoice{Action: trkReadied(t, "go", resp)})
		tr, _ = trkAdvance(t, tr)
		return tr
	}()
	stand := combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: 15}}
	row := func(name, requirement, path, op, class, result string, events []string, moved map[string]float64,
		run func(opts ...combat.Option) (any, error), direct func() (combat.TrackerStep, error)) trkUseRow {
		return trkUseRow{name, spec + requirement, path, op, run, direct, class, result, events, moved}
	}
	take := func(tr combat.TurnTracker, id combat.CombatantID, av combat.ConditionActionAvailability, c combat.ActionChoice) trkUseRow {
		return row("", "", "", "take_action", "", "", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, err := combat.NewTakeActionCommand(tr, id, av, c)
				if err != nil {
					t.Fatal(err)
				}
				return combat.NewTakeAction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return tr.TakeAction(id, av, c) })
	}
	named := func(r trkUseRow, name, requirement, path, class, result string, events []string) trkUseRow {
		r.name, r.requirement, r.path, r.class, r.result, r.events = name, spec+requirement, path, class, result, events
		return r
	}
	return []trkUseRow{
		row("advance_tracker_turn applied", "FR-099 FR-100 FR-149", "happy", "advance_tracker_turn", "applied", "applied", []string{"tracker_turn_ended", "tracker_turn_started"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{})
				return combat.NewAdvanceTrackerTurn(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.Advance(combat.TurnAdvance{}) }),
		row("advance_tracker_turn unknown combatant is invalid", "FR-042 FR-100 FR-119 FR-141", "unhappy (invalid input)", "advance_tracker_turn", "invalid", "unknown_combatant", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{Updates: []combat.SpeedUpdate{{Combatant: "x", Speeds: trkSpeeds(t, 1)}}})
				return combat.NewAdvanceTrackerTurn(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.Advance(combat.TurnAdvance{Updates: []combat.SpeedUpdate{{Combatant: "x", Speeds: trkSpeeds(t, 1)}}})
			}),
		row("update_movement_speeds applied", "FR-099 FR-116 FR-142 FR-149", "happy", "update_movement_speeds", "applied", "applied", []string{"movement_speeds_updated"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewUpdateMovementSpeedsCommand(base, trkB, trkSpeeds(t, 10))
				return combat.NewUpdateMovementSpeeds(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.UpdateMovementSpeeds(trkB, trkSpeeds(t, 10)) }),
		row("update_movement_speeds unknown", "FR-116 FR-119 FR-142", "unhappy (invalid input)", "update_movement_speeds", "invalid", "unknown_combatant", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewUpdateMovementSpeedsCommand(base, "x", trkSpeeds(t, 10))
				return combat.NewUpdateMovementSpeeds(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.UpdateMovementSpeeds("x", trkSpeeds(t, 10)) }),
		named(take(base, trkA, trkAvail, attack), "take_action applied", "FR-099 FR-100 FR-143 FR-149", "happy", "applied", "applied", []string{"action_taken"}),
		named(take(attackTaken, trkA, trkAvail, trkDodge), "take_action refused action_already_taken", "FR-045 FR-100", "unhappy (state error)", "refused", "action_already_taken", nil),
		named(take(base, trkB, trkAvail, attack), "take_action refused not_active_combatant", "FR-046 FR-100", "unhappy (state error)", "refused", "not_active_combatant", nil),
		named(take(base, trkA, trkNone, attack), "take_action refused incapacitated", "FR-047 FR-100", "unhappy (state error)", "refused", "incapacitated", nil),
		named(take(base, "x", trkAvail, attack), "take_action unknown combatant is invalid", "FR-119 FR-100", "unhappy (invalid input)", "invalid", "unknown_combatant", nil),
		named(take(base, trkA, trkAvail, combat.DashChoice{Speed: combat.SpeedNameSwim}), "take_action refused speed_unavailable", "FR-052 FR-100", "unhappy (invalid input)", "refused", "speed_unavailable", nil),
		row("take_bonus_action applied", "FR-063 FR-144", "happy", "take_bonus_action", "applied", "applied", []string{"bonus_action_used"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewTakeBonusActionCommand(base, trkA, trkAvail, combat.EntitledToBonusAction{Grant: bg})
				return combat.NewTakeBonusAction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.TakeBonusAction(trkA, trkAvail, combat.EntitledToBonusAction{Grant: bg})
			}),
		row("take_bonus_action refused no_bonus_action_granted", "FR-060 FR-100", "unhappy (state error)", "take_bonus_action", "refused", "no_bonus_action_granted", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewTakeBonusActionCommand(base, trkA, trkAvail, combat.NotEntitledToBonusAction{})
				return combat.NewTakeBonusAction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.TakeBonusAction(trkA, trkAvail, combat.NotEntitledToBonusAction{})
			}),
		row("take_reaction applied", "FR-068 FR-071 FR-145", "happy", "take_reaction", "applied", "applied", []string{"reaction_taken"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewTakeReactionCommand(base, trkB, trkAvail, oa)
				return combat.NewTakeReaction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.TakeReaction(trkB, trkAvail, oa) }),
		row("take_reaction refused provoker_not_seen", "FR-070 FR-100", "unhappy (invalid input)", "take_reaction", "refused", "provoker_not_seen", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewTakeReactionCommand(base, trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilityUnseen))
				return combat.NewTakeReaction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.TakeReaction(trkB, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilityUnseen))
			}),
		row("take_reaction readied move records its movement", "FR-074 FR-104", "happy", "take_reaction", "applied", "applied", []string{"reaction_taken"}, map[string]float64{"walk": 10, "fly": 5},
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewTakeReactionCommand(readyMoveTr, trkB, trkAvail, combat.ReadiedActionReaction{Trigger: "go"})
				return combat.NewTakeReaction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return readyMoveTr.TakeReaction(trkB, trkAvail, combat.ReadiedActionReaction{Trigger: "go"})
			}),
		row("take_reaction unknown provoker is invalid", "FR-119 FR-100", "unhappy (invalid input)", "take_reaction", "invalid", "unknown_combatant", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewTakeReactionCommand(base, trkB, trkAvail, trkOA("x", combat.DepartureBySpeed, combat.VisibilitySeen))
				return combat.NewTakeReaction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.TakeReaction(trkB, trkAvail, trkOA("x", combat.DepartureBySpeed, combat.VisibilitySeen))
			}),
		row("move applied: one sample per mode", "FR-104 FR-146 FR-149", "happy", "move", "applied", "applied", []string{"movement_spent"}, map[string]float64{"walk": 15, "fly": 5},
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewMoveCommand(base, trkA, segs)
				return combat.NewMove(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.Move(trkA, segs) }),
		row("move refused insufficient_movement records no movement", "FR-086 FR-104", "unhappy (boundary)", "move", "refused", "insufficient_movement", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewMoveCommand(base, trkA, []combat.MoveSegment{trkWalk(t, 31)})
				return combat.NewMove(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.Move(trkA, []combat.MoveSegment{trkWalk(t, 31)}) }),
		row("move unknown combatant is invalid", "FR-119 FR-100", "unhappy (invalid input)", "move", "invalid", "unknown_combatant", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewMoveCommand(base, "x", segs)
				return combat.NewMove(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.Move("x", segs) }),
		row("interact_with_object applied", "FR-058 FR-147", "happy", "interact_with_object", "applied", "applied", []string{"object_interacted"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewInteractWithObjectCommand(base, trkA)
				return combat.NewInteractWithObject(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.InteractWithObject(trkA) }),
		row("interact_with_object refused needs_utilize_action", "FR-058 FR-100", "unhappy (state error)", "interact_with_object", "refused", "needs_utilize_action", nil, nil,
			func(opts ...combat.Option) (any, error) {
				used, _ := trkApplied(t)(base.InteractWithObject(trkA))
				cmd, _ := combat.NewInteractWithObjectCommand(used, trkA)
				return combat.NewInteractWithObject(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				used, _ := trkApplied(t)(base.InteractWithObject(trkA))
				return used.InteractWithObject(trkA)
			}),
		row("change_posture stand-up records posture movement", "FR-091 FR-104 FR-148", "happy", "change_posture", "applied", "applied", []string{"stood_up"}, map[string]float64{"posture": 15},
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewChangePostureCommand(base, trkA, stand)
				return combat.NewChangePosture(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.ChangePosture(trkA, stand) }),
		row("change_posture drop Prone records no movement", "FR-090 FR-104", "happy", "change_posture", "applied", "applied", []string{"dropped_prone"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewChangePostureCommand(base, trkA, combat.DropProneChange{})
				return combat.NewChangePosture(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.ChangePosture(trkA, combat.DropProneChange{}) }),
		row("change_posture refused speed_is_zero", "FR-091 FR-100", "unhappy (boundary)", "change_posture", "refused", "speed_is_zero", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewChangePostureCommand(base, trkA, combat.StandUpChange{Movement: combat.CrawlOnly{}})
				return combat.NewChangePosture(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.ChangePosture(trkA, combat.StandUpChange{Movement: combat.CrawlOnly{}})
			}),
		row("grant_additional_action applied", "FR-155 FR-164", "happy", "grant_additional_action", "applied", "applied", []string{"additional_action_granted"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewGrantAdditionalActionCommand(base, trkA, grant)
				return combat.NewGrantAdditionalAction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.GrantAdditionalAction(trkA, grant) }),
		row("grant_additional_action refused not_active_combatant", "FR-155 FR-164", "unhappy (state error)", "grant_additional_action", "refused", "not_active_combatant", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewGrantAdditionalActionCommand(base, trkB, grant)
				return combat.NewGrantAdditionalAction(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.GrantAdditionalAction(trkB, grant) }),
		row("record_feat_use applied", "FR-157 FR-164", "happy", "record_feat_use", "applied", "applied", []string{"feat_use_recorded"}, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewRecordFeatUseCommand(base, trkB, "savage_attacker", character.OncePerTurn{})
				return combat.NewRecordFeatUse(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				return base.RecordFeatUse(trkB, "savage_attacker", character.OncePerTurn{})
			}),
		row("record_feat_use refused feat_use_limited", "FR-157 FR-164", "unhappy (state error)", "record_feat_use", "refused", "feat_use_limited", nil, nil,
			func(opts ...combat.Option) (any, error) {
				used, _ := trkApplied(t)(base.RecordFeatUse(trkB, "savage_attacker", character.OncePerTurn{}))
				cmd, _ := combat.NewRecordFeatUseCommand(used, trkB, "savage_attacker", character.OncePerTurn{})
				return combat.NewRecordFeatUse(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) {
				used, _ := trkApplied(t)(base.RecordFeatUse(trkB, "savage_attacker", character.OncePerTurn{}))
				return used.RecordFeatUse(trkB, "savage_attacker", character.OncePerTurn{})
			}),
		row("record_feat_use unknown combatant is invalid", "FR-157 FR-164", "unhappy (invalid input)", "record_feat_use", "invalid", "unknown_combatant", nil, nil,
			func(opts ...combat.Option) (any, error) {
				cmd, _ := combat.NewRecordFeatUseCommand(base, "x", "f", character.OncePerTurn{})
				return combat.NewRecordFeatUse(opts...).Run(ctx, cmd)
			}, func() (combat.TrackerStep, error) { return base.RecordFeatUse("x", "f", character.OncePerTurn{}) }),
	}
}

// Requirement: per case (FR-099, FR-100, FR-103, FR-104, FR-107, FR-132, FR-149, FR-164). Each use case returns what its tracker operation returns, one to one; records the shared signals with its class and result name; counts its events by kind; records turn_movement_spent only for an accepted move, stand-up or readied move; sets error status only for an invalid result and never increments srd5e.combat.errors; and returns equal results with the no-op bundle.
// Path: per case.
func TestTurnUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range trkUseRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := c.run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-107]", got, plain, err)
			}
			// FR-149: the same as a direct call.
			step, err := c.direct()
			if err != nil {
				t.Fatal(err)
			}
			gt, gev, greason := trkUnpack(t, got)
			switch s := step.(type) {
			case combat.TrackerStepApplied:
				if greason != nil || !gt.Equal(s.Tracker()) || !reflect.DeepEqual(gev, s.Events()) {
					t.Errorf("use case result differs from the direct call [FR-149]")
				}
			case combat.TrackerStepRejected:
				if greason != s.Reason {
					t.Errorf("reason = %#v, direct call %#v [FR-149]", greason, s.Reason)
				}
			}
			outcome := map[string]string{"applied": "success", "refused": "failure", "invalid": "failure"}[c.class]
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status only for invalid [%s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [FR-099]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v", ops, want)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s [FR-105]", dur)
			}
			wantKinds := slices.Clone(c.events)
			slices.Sort(wantKinds)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, wantKinds) {
				t.Errorf("events counted = %v, want %v [FR-103]", counted, wantKinds)
			}
			spent := map[string]float64{}
			for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.turn_movement_spent") {
				spent[attrMap(m.Attrs)["srd5e.combat.movement_mode"].(string)] += m.Value
			}
			if len(c.moved) == 0 && len(spent) != 0 || len(c.moved) != 0 && !reflect.DeepEqual(spent, c.moved) {
				t.Errorf("turn_movement_spent = %v, want %v [FR-104]", spent, c.moved)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result [FR-099]", n)
			}
			for _, banned := range []string{"turns_started", "rounds_started", "turn_actions"} {
				if m := rec.Meter.MeasurementsNamed("srd5e.combat." + banned); len(m) != 0 {
					t.Errorf("%s recorded [FR-103]", banned)
				}
			}
			trkHygiene(t, rec)
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-097, FR-133 - every use case returns an invalid-input error, before any signal, for its zero command, and ctx.Err() unwrapped for a cancelled ctx.
// Path: unhappy (invalid input, cancellation).
func TestTurnUseCasesGuards(t *testing.T) {
	t.Parallel()
	base := trkSolo30(t)
	cmds := map[string]func(ctx context.Context, opts []combat.Option) error{}
	reg := func(name string, run func(ctx context.Context, opts []combat.Option) (any, error)) {
		cmds[name] = func(ctx context.Context, opts []combat.Option) error {
			res, err := run(ctx, opts)
			if err == nil || res != nil && !reflect.ValueOf(res).IsNil() {
				t.Errorf("%s: res %v err %v, want an error and a nil result", name, res, err)
			}
			return err
		}
	}
	// A valid command whose zero value is also run to prove the guard.
	reg("advance", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{})
		if _, err := combat.NewAdvanceTrackerTurn(o...).Run(context.Background(), combat.AdvanceTrackerTurnCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero advance command: %v", err)
		}
		r, err := combat.NewAdvanceTrackerTurn(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("update", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewUpdateMovementSpeedsCommand(base, trkA, trkSpeeds(t, 1))
		if _, err := combat.NewUpdateMovementSpeeds(o...).Run(context.Background(), combat.UpdateMovementSpeedsCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero update command: %v", err)
		}
		r, err := combat.NewUpdateMovementSpeeds(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("take action", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewTakeActionCommand(base, trkA, trkAvail, trkAttack)
		if _, err := combat.NewTakeAction(o...).Run(context.Background(), combat.TakeActionCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero action command: %v", err)
		}
		r, err := combat.NewTakeAction(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("take bonus action", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewTakeBonusActionCommand(base, trkA, trkAvail, combat.NotEntitledToBonusAction{})
		if _, err := combat.NewTakeBonusAction(o...).Run(context.Background(), combat.TakeBonusActionCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero Bonus Action command: %v", err)
		}
		r, err := combat.NewTakeBonusAction(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("take reaction", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewTakeReactionCommand(base, trkB, trkAvail, combat.SpellReaction{Spell: "s"})
		if _, err := combat.NewTakeReaction(o...).Run(context.Background(), combat.TakeReactionCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero Reaction command: %v", err)
		}
		r, err := combat.NewTakeReaction(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("move", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewMoveCommand(base, trkA, []combat.MoveSegment{trkWalk(t, 5)})
		if _, err := combat.NewMove(o...).Run(context.Background(), combat.MoveCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero move command: %v", err)
		}
		r, err := combat.NewMove(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("interact", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewInteractWithObjectCommand(base, trkA)
		if _, err := combat.NewInteractWithObject(o...).Run(context.Background(), combat.InteractWithObjectCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero interact command: %v", err)
		}
		r, err := combat.NewInteractWithObject(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("posture", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewChangePostureCommand(base, trkA, combat.DropProneChange{})
		if _, err := combat.NewChangePosture(o...).Run(context.Background(), combat.ChangePostureCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero posture command: %v", err)
		}
		r, err := combat.NewChangePosture(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("grant", func(ctx context.Context, o []combat.Option) (any, error) {
		g, _ := combat.NewAdditionalActionGrant("f", combat.AnyActionBut{})
		cmd, _ := combat.NewGrantAdditionalActionCommand(base, trkA, g)
		if _, err := combat.NewGrantAdditionalAction(o...).Run(context.Background(), combat.GrantAdditionalActionCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero grant command: %v", err)
		}
		r, err := combat.NewGrantAdditionalAction(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	reg("feat use", func(ctx context.Context, o []combat.Option) (any, error) {
		cmd, _ := combat.NewRecordFeatUseCommand(base, trkA, "f", character.OncePerTurn{})
		if _, err := combat.NewRecordFeatUse(o...).Run(context.Background(), combat.RecordFeatUseCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero feat command: %v", err)
		}
		r, err := combat.NewRecordFeatUse(o...).Run(ctx, cmd)
		return nilIf(r == nil, r), err
	})
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, run := range cmds {
		rec, opts := recording()
		if err := run(cancelled, opts); !errors.Is(err, context.Canceled) {
			t.Errorf("%s: err = %v, want context.Canceled", name, err)
		}
		if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
			t.Errorf("%s: a cancelled call recorded signals", name)
		}
	}
}

// nilIf returns nil when isNil; it avoids a typed nil interface in the guard test.
func nilIf(isNil bool, v any) any {
	if isNil {
		return nil
	}
	return v
}

// Requirement: doc/specs/initiative-and-actions.md FR-133, FR-119 - each command constructor returns an invalid-input error for a malformed command and never for an id that is merely not in the tracker.
// Path: unhappy (invalid input).
func TestTurnCommandConstructorsReject(t *testing.T) {
	t.Parallel()
	base := trkSolo30(t)
	zero := combat.TurnTracker{}
	g, _ := combat.NewAdditionalActionGrant("f", combat.AnyActionBut{})
	bg, _ := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "b"}, combat.NoStandardAction{})
	seg := []combat.MoveSegment{trkWalk(t, 5)}
	errs := map[string]error{}
	put := func(name string, err error) { errs[name] = err }
	_, e := combat.NewAdvanceTrackerTurnCommand(zero, combat.TurnAdvance{})
	put("advance zero tracker", e)
	_, e = combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{Updates: []combat.SpeedUpdate{{Combatant: ""}}})
	put("advance blank update", e)
	_, e = combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{Updates: []combat.SpeedUpdate{{Combatant: trkA}, {Combatant: trkA}}})
	put("advance repeated update", e)
	_, e = combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{SharedSpaceOccupants: []content.CreatureSize{0}})
	put("advance bad size", e)
	_, e = combat.NewUpdateMovementSpeedsCommand(zero, trkA, combat.MovementSpeeds{})
	put("update zero tracker", e)
	_, e = combat.NewUpdateMovementSpeedsCommand(base, "", combat.MovementSpeeds{})
	put("update blank id", e)
	_, e = combat.NewTakeActionCommand(zero, trkA, trkAvail, trkAttack)
	put("action zero tracker", e)
	_, e = combat.NewTakeActionCommand(base, trkA, 0, trkAttack)
	put("action no availability", e)
	_, e = combat.NewTakeActionCommand(base, trkA, trkAvail, nil)
	put("action nil choice", e)
	_, e = combat.NewTakeActionCommand(base, "", trkAvail, trkAttack)
	put("action blank id", e)
	_, e = combat.NewTakeBonusActionCommand(base, trkA, trkAvail, nil)
	put("Bonus Action nil entitlement", e)
	_, e = combat.NewTakeBonusActionCommand(base, trkA, trkAvail, combat.EntitledToBonusAction{})
	put("Bonus Action zero grant", e)
	_, e = combat.NewTakeBonusActionCommand(base, trkA, 0, combat.NotEntitledToBonusAction{})
	put("Bonus Action no availability", e)
	_, e = combat.NewTakeReactionCommand(base, trkA, trkAvail, trkOA(trkA, combat.DepartureBySpeed, combat.VisibilitySeen))
	put("Reaction provoker is the observer", e)
	_, e = combat.NewTakeReactionCommand(base, trkA, trkAvail, nil)
	put("Reaction nil choice", e)
	_, e = combat.NewTakeReactionCommand(base, trkA, 0, combat.SpellReaction{Spell: "s"})
	put("Reaction no availability", e)
	_, e = combat.NewMoveCommand(base, trkA, nil)
	put("move no segments", e)
	_, e = combat.NewMoveCommand(base, trkA, []combat.MoveSegment{{}})
	put("move zero segment", e)
	_, e = combat.NewInteractWithObjectCommand(zero, trkA)
	put("interact zero tracker", e)
	_, e = combat.NewInteractWithObjectCommand(base, "")
	put("interact blank id", e)
	_, e = combat.NewChangePostureCommand(base, trkA, nil)
	put("posture nil", e)
	_, e = combat.NewChangePostureCommand(base, trkA, combat.StandUpChange{})
	put("posture stand without movement", e)
	_, e = combat.NewChangePostureCommand(base, trkA, combat.StandUpChange{Movement: combat.CrawlOrRightItself{CostFeet: -1}})
	put("posture negative cost", e)
	_, e = combat.NewGrantAdditionalActionCommand(base, trkA, combat.AdditionalActionGrant{})
	put("grant zero", e)
	_, e = combat.NewGrantAdditionalActionCommand(zero, trkA, g)
	put("grant zero tracker", e)
	_, e = combat.NewRecordFeatUseCommand(base, trkA, "f", character.NoUsageLimit{})
	put("feat use none", e)
	_, e = combat.NewRecordFeatUseCommand(base, trkA, "f", character.UseRenewedBy{})
	put("feat use renewed by", e)
	_, e = combat.NewRecordFeatUseCommand(base, trkA, "F F", character.OncePerTurn{})
	put("feat use bad id", e)
	_, e = combat.NewRecordFeatUseCommand(base, trkA, "f", nil)
	put("feat use nil", e)
	_, e = combat.NewRecordFeatUseCommand(base, "", "f", character.OncePerTurn{})
	put("feat use blank combatant", e)
	for name, err := range errs {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, want invalid input", name, err)
		}
	}
	// Unknown ids are valid commands.
	for name, e := range map[string]error{
		"action": func() error { _, err := combat.NewTakeActionCommand(base, "x", trkAvail, trkAttack); return err }(),
		"move":   func() error { _, err := combat.NewMoveCommand(base, "x", seg); return err }(),
		"bonus": func() error {
			_, err := combat.NewTakeBonusActionCommand(base, "x", trkAvail, combat.EntitledToBonusAction{Grant: bg})
			return err
		}(),
		"posture": func() error {
			_, err := combat.NewChangePostureCommand(base, "x", combat.DropProneChange{})
			return err
		}(),
	} {
		if e != nil {
			t.Errorf("%s with an unknown id: %v, want a valid command [FR-119]", name, e)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-096 - commands copy their lists: changing the caller's slices after construction changes nothing.
// Path: unhappy (immutability).
func TestTurnCommandsCopyLists(t *testing.T) {
	t.Parallel()
	base := trkSolo30(t)
	segs := []combat.MoveSegment{trkWalk(t, 10)}
	cmd, err := combat.NewMoveCommand(base, trkA, segs)
	if err != nil {
		t.Fatal(err)
	}
	segs[0] = trkWalk(t, 31)
	res, err := combat.NewMove().Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.(combat.MoveApplied); !ok {
		t.Errorf("result = %#v: the command shares the caller's segments", res)
	}
	updates := []combat.SpeedUpdate{{Combatant: trkB, Speeds: trkSpeeds(t, 5)}}
	acmd, _ := combat.NewAdvanceTrackerTurnCommand(base, combat.TurnAdvance{Updates: updates})
	updates[0].Combatant = "x"
	ares, _ := combat.NewAdvanceTrackerTurn().Run(context.Background(), acmd)
	if _, ok := ares.(combat.AdvanceTrackerTurnApplied); !ok {
		t.Errorf("result = %#v: the command shares the caller's updates", ares)
	}
}
