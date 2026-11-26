package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// meRow is one magical-effects use case run on fixed inputs: the operation it records, the owner's own result it returns
// unchanged, the class and srd5e.combat.result of the section 7 table and the event kinds it counts.
type meRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	owner                   any
}

func meUseCaseRows(t *testing.T) []meRow {
	t.Helper()
	ctx := context.Background()
	hero, mage := gid("hero"), gid("mage")
	curses := meEffects(meBless(), meCurse("c1"))
	dead := meMust(combat.NewDead(combat.MassiveDamage))
	shifted := meShifted(t)
	shift := func(rule combat.CarryOverRule) combat.ShiftShapeCommand {
		return combat.ShiftShapeCommand{Creature: hero, State: combat.TrueForm(), Conditions: meBaseline(), Effects: curses, Effect: meSpell("polymorph"), Rule: rule}
	}
	revert := func(rule combat.CarryOverRule) combat.RevertToTrueFormCommand {
		return combat.RevertToTrueFormCommand{Creature: hero, State: shifted, Conditions: meBaseline(), Effects: curses, Rule: rule}
	}
	tele := func(f combat.TeleportFields) combat.TeleportCommand {
		f.Teleporter = mage
		return meMust(combat.NewTeleportCommand(f))
	}
	seen := tele(combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightRequired, DestinationSeen: true, Policy: combat.Accompany, Touched: []combat.CombatantID{gid("a")}})
	near := tele(combat.TeleportFields{Destination: combat.OccupiedByCreature, Sight: combat.DestinationSightNotRequired})
	unseen := tele(combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightRequired})
	badSet := tele(combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, Touched: []combat.CombatantID{mage}})
	remove := func(id string, means combat.CurseRemovalMeans) combat.RemoveCombatantCurseCommand {
		return combat.RemoveCombatantCurseCommand{Creature: hero, Effects: curses, Curse: meEID(id), Means: means}
	}
	order := func(h combat.TurnHolder, pending, chosen []combat.PendingEffectID) combat.ResolveEffectOrderCommand {
		return combat.ResolveEffectOrderCommand{Holder: h, Pending: pending, Chosen: chosen}
	}
	membership := func(f combat.AreaLocationFacts) combat.AssessAreaLocationCommand {
		return combat.AssessAreaLocationCommand{Request: combat.AssessMembershipRequest{Facts: f}}
	}
	cr := combat.CurseRemoval{}
	ss := combat.ShapeShifting{}
	se := combat.SimultaneousEffects{}
	am := combat.AreaMembership{}
	const fr = "magical-effects.md FR-148, FR-144, section 7; "
	return []meRow{
		{"remove a curse", meSpec + fr + "FR-030", "happy", "remove_combatant_curse",
			func(o ...combat.Option) (any, error) {
				return combat.NewRemoveCombatantCurse(o...).Run(ctx, remove("c1", combat.RemoveCurseSpell))
			}, "applied", "completed", []string{"curse_removed"}, cr.Remove(hero, curses, meEID("c1"), combat.RemoveCurseSpell)},
		{"remove a curse that is not there", meSpec + fr + "FR-031", "unhappy (state error)", "remove_combatant_curse",
			func(o ...combat.Option) (any, error) {
				return combat.NewRemoveCombatantCurse(o...).Run(ctx, remove("nope", combat.RemoveCurseSpell))
			}, "refused", "curse_not_present", nil, cr.Remove(hero, curses, meEID("nope"), combat.RemoveCurseSpell)},
		{"remove an effect that is not a curse", meSpec + fr + "FR-032", "unhappy (state error)", "remove_combatant_curse",
			func(o ...combat.Option) (any, error) {
				return combat.NewRemoveCombatantCurse(o...).Run(ctx, remove("bless", combat.GreaterRestorationSpell))
			}, "refused", "curse_not_a_curse", nil, cr.Remove(hero, curses, meEID("bless"), combat.GreaterRestorationSpell)},
		{"shift into a form", meSpec + fr + "FR-050", "happy", "shift_combatant_shape",
			func(o ...combat.Option) (any, error) {
				return combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.ShiftRequest{Command: shift(nil)}})
			}, "applied", "completed", []string{"shape_shifted"}, meMust(ss.Shift(shift(nil)))},
		{"shift with an exception naming a condition not held", meSpec + fr + "FR-049", "unhappy (state error)", "shift_combatant_shape",
			func(o ...combat.Option) (any, error) {
				return combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.ShiftRequest{Command: shift(meExcept(nil, "nope"))}})
			}, "refused", "shift_exception_not_held", nil, meMust(ss.Shift(shift(meExcept(nil, "nope"))))},
		{"revert to true form", meSpec + fr + "FR-052", "happy", "shift_combatant_shape",
			func(o ...combat.Option) (any, error) {
				return combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.RevertRequest{Command: revert(nil)}})
			}, "applied", "completed", []string{"reverted_to_true_form"}, meMust(ss.Revert(revert(nil)))},
		{"revert with an exception naming an effect not held", meSpec + fr + "FR-052, FR-049", "unhappy (state error)", "shift_combatant_shape",
			func(o ...combat.Option) (any, error) {
				return combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.RevertRequest{Command: revert(meExcept(nil, "nope"))}})
			}, "refused", "shift_exception_not_held", nil, meMust(ss.Revert(revert(meExcept(nil, "nope"))))},
		{"revert on death", meSpec + fr + "FR-054", "happy", "shift_combatant_shape",
			func(o ...combat.Option) (any, error) {
				return combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.RevertOnDeathRequest{Creature: hero, State: shifted, Life: dead}})
			}, "applied", "completed", []string{"reverted_to_true_form"}, ss.RevertOnDeath(hero, shifted, dead)},
		{"revert on death of a creature that is not Dead", meSpec + fr + "FR-055", "unhappy (state error)", "shift_combatant_shape",
			func(o ...combat.Option) (any, error) {
				return combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.RevertOnDeathRequest{Creature: hero, State: shifted, Life: combat.Alive{}}})
			}, "refused", "revert_not_dead", nil, ss.RevertOnDeath(hero, shifted, combat.Alive{})},
		{"teleport to a seen, free destination with a touched creature", meSpec + fr + "FR-056, FR-069", "happy", "teleport_combatant",
			func(o ...combat.Option) (any, error) {
				return combat.NewTeleportCombatant(o...).Run(ctx, combat.TeleportCombatantCommand{Command: seen})
			}, "applied", "completed", []string{"creature_teleported"}, meMust(combat.Teleportation{}.Resolve(seen))},
		{"teleport to an occupied destination", meSpec + fr + "FR-057", "unhappy (rule interaction)", "teleport_combatant",
			func(o ...combat.Option) (any, error) {
				return combat.NewTeleportCombatant(o...).Run(ctx, combat.TeleportCombatantCommand{Command: near})
			}, "applied", "completed", []string{"creature_teleported"}, meMust(combat.Teleportation{}.Resolve(near))},
		{"teleport to a destination that must be seen and is not", meSpec + fr + "FR-059", "unhappy (state error)", "teleport_combatant",
			func(o ...combat.Option) (any, error) {
				return combat.NewTeleportCombatant(o...).Run(ctx, combat.TeleportCombatantCommand{Command: unseen})
			}, "refused", "teleport_destination_not_seen", nil, meMust(combat.Teleportation{}.Resolve(unseen))},
		{"teleport with the teleporter in the touched set", meSpec + fr + "FR-068", "unhappy (invalid input)", "teleport_combatant",
			func(o ...combat.Option) (any, error) {
				return combat.NewTeleportCombatant(o...).Run(ctx, combat.TeleportCombatantCommand{Command: badSet})
			}, "invalid", "teleport_invalid_touched_set", nil, meMust(combat.Teleportation{}.Resolve(badSet))},
		{"order simultaneous effects", meSpec + fr + "FR-092", "happy", "resolve_effect_order",
			func(o ...combat.Option) (any, error) {
				return combat.NewResolveEffectOrder(o...).Run(ctx, order(combat.TurnHolderPlayer, mePending("a", "b"), mePending("b", "a")))
			}, "applied", "completed", nil, se.Order(combat.TurnHolderPlayer, mePending("a", "b"), mePending("b", "a"))},
		{"order a single pending effect", meSpec + fr + "FR-094", "unhappy (empty and minimal)", "resolve_effect_order",
			func(o ...combat.Option) (any, error) {
				return combat.NewResolveEffectOrder(o...).Run(ctx, order(combat.TurnHolderGameMaster, mePending("a"), nil))
			}, "applied", "completed", nil, se.Order(combat.TurnHolderGameMaster, mePending("a"), nil)},
		{"order with an id omitted", meSpec + fr + "FR-093", "unhappy (invalid input)", "resolve_effect_order",
			func(o ...combat.Option) (any, error) {
				return combat.NewResolveEffectOrder(o...).Run(ctx, order(combat.TurnHolderPlayer, mePending("a", "b"), mePending("a")))
			}, "invalid", "order_rejected", nil, se.Order(combat.TurnHolderPlayer, mePending("a", "b"), mePending("a"))},
		{"order with a pending id twice", meSpec + fr + "FR-095", "unhappy (invalid input)", "resolve_effect_order",
			func(o ...combat.Option) (any, error) {
				return combat.NewResolveEffectOrder(o...).Run(ctx, order(combat.TurnHolderPlayer, mePending("a", "a"), mePending("a", "a")))
			}, "invalid", "order_rejected", nil, se.Order(combat.TurnHolderPlayer, mePending("a", "a"), mePending("a", "a"))},
		{"order over the cap of pending ids", meSpec + fr + "plan P10 (257 pending ids)", "unhappy (boundary)", "resolve_effect_order",
			func(o ...combat.Option) (any, error) {
				return combat.NewResolveEffectOrder(o...).Run(ctx, order(combat.TurnHolderPlayer, mePendingN(257), nil))
			}, "invalid", "limit_exceeded", nil, se.Order(combat.TurnHolderPlayer, mePendingN(257), nil)},
		{"a location inside the shape with an open line", meSpec + fr + "FR-020", "happy", "assess_area_location",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessAreaLocation(o...).Run(ctx, membership(meOther(combat.InsideShape, combat.HalfCover)))
			}, "applied", "completed", nil, am.Assess(meOther(combat.InsideShape, combat.HalfCover))},
		{"a location outside the shape", meSpec + fr + "FR-019", "unhappy (rule interaction)", "assess_area_location",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessAreaLocation(o...).Run(ctx, membership(meOther(combat.OutsideShape, combat.HalfCover)))
			}, "applied", "area_location_excluded", nil, am.Assess(meOther(combat.OutsideShape, combat.HalfCover))},
		{"a location whose every line is Total Cover", meSpec + fr + "FR-021", "unhappy (rule interaction)", "assess_area_location",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessAreaLocation(o...).Run(ctx, membership(meOther(combat.InsideShape, combat.TotalCover)))
			}, "applied", "area_location_excluded", nil, am.Assess(meOther(combat.InsideShape, combat.TotalCover))},
		{"a location inside the shape with no lines", meSpec + fr + "FR-023", "unhappy (empty and minimal)", "assess_area_location",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessAreaLocation(o...).Run(ctx, membership(meOther(combat.InsideShape)))
			}, "invalid", "area_no_lines_considered", nil, am.Assess(meOther(combat.InsideShape))},
		{"no facts at all", meSpec + fr + "FR-023", "unhappy (invalid input)", "assess_area_location",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessAreaLocation(o...).Run(ctx, membership(nil))
			}, "invalid", "area_no_lines_considered", nil, am.Assess(nil)},
		{"resolve an origin at an unseen point behind an obstruction", meSpec + fr + "FR-018", "happy", "assess_area_location",
			func(o ...combat.Option) (any, error) {
				return combat.NewAssessAreaLocation(o...).Run(ctx, combat.AssessAreaLocationCommand{Request: combat.ResolveOriginRequest{Placement: combat.UnseenPointBehindObstruction}})
			}, "applied", "completed", nil, combat.AreaOrigin{}.Resolve(combat.UnseenPointBehindObstruction)},
	}
}

// Requirement: per case
// Path: per case.
func TestMagicalEffectsUseCases(t *testing.T) {
	t.Parallel()
	for _, c := range meUseCaseRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if plain, err := c.run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("the recording bundle and Noop disagree [%s]", c.requirement)
			}
			if !reflect.DeepEqual(got, c.owner) {
				t.Errorf("result = %#v, want the owner's %#v unchanged [%s]", got, c.owner, c.requirement)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			wantErr := c.class == "invalid"
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != wantErr || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status %v, no recorded error [%s]", spans, wantErr, c.requirement)
			}
			outcome := "success"
			if c.class != "applied" {
				outcome = "failure"
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [%s]", sa, want, c.requirement)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, append([]string{}, c.events...)) {
				t.Errorf("event kinds counted = %v, want %v [%s]", kinds, c.events, c.requirement)
			}
			resHygiene(t, rec, false)
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-148, FR-144 and doc/rules/testing.md section 4 (cancellation) - an already-cancelled context is returned unwrapped by each magical-effects use case before any signal.
// Path: unhappy (state error).
func TestMagicalEffectsUseCasesCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec, opts := recording()
	_, e1 := combat.NewRemoveCombatantCurse(opts...).Run(ctx, combat.RemoveCombatantCurseCommand{})
	_, e2 := combat.NewShiftCombatantShape(opts...).Run(ctx, combat.ShiftCombatantShapeCommand{})
	_, e3 := combat.NewTeleportCombatant(opts...).Run(ctx, combat.TeleportCombatantCommand{})
	_, e4 := combat.NewResolveEffectOrder(opts...).Run(ctx, combat.ResolveEffectOrderCommand{})
	_, e5 := combat.NewAssessAreaLocation(opts...).Run(ctx, combat.AssessAreaLocationCommand{})
	for i, err := range []error{e1, e2, e3, e4, e5} {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("use case %d: err = %v, want context.Canceled unwrapped", i, err)
		}
	}
	if len(rec.Meter.Measurements()) != 0 || len(rec.Tracer.SpansNamed("srd5e.combat.remove_combatant_curse")) != 0 {
		t.Errorf("a cancelled call recorded a signal")
	}
}

// Requirement: doc/specs/magical-effects.md section 5 (Failures), FR-148 - a nil or unknown request, a zero creature or effect, a command not built by NewTeleportCommand and an undefined origin placement are errors matching srderr.ErrInvalidInput, recorded on the span with error status and never counted as a result.
// Path: unhappy (invalid input).
func TestMagicalEffectsUseCasesInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, c := range []struct {
		name, operation string
		run             func(opts ...combat.Option) error
	}{
		{"shape change with no request", "shift_combatant_shape", func(o ...combat.Option) error {
			_, err := combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{})
			return err
		}},
		{"shift with the zero command", "shift_combatant_shape", func(o ...combat.Option) error {
			_, err := combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.ShiftRequest{}})
			return err
		}},
		{"revert with the zero command", "shift_combatant_shape", func(o ...combat.Option) error {
			_, err := combat.NewShiftCombatantShape(o...).Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.RevertRequest{}})
			return err
		}},
		{"teleport with the zero command", "teleport_combatant", func(o ...combat.Option) error {
			_, err := combat.NewTeleportCombatant(o...).Run(ctx, combat.TeleportCombatantCommand{})
			return err
		}},
		{"area location with no request", "assess_area_location", func(o ...combat.Option) error {
			_, err := combat.NewAssessAreaLocation(o...).Run(ctx, combat.AssessAreaLocationCommand{})
			return err
		}},
		{"origin with an undefined placement", "assess_area_location", func(o ...combat.Option) error {
			_, err := combat.NewAssessAreaLocation(o...).Run(ctx, combat.AssessAreaLocationCommand{Request: combat.ResolveOriginRequest{Placement: 0}})
			return err
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			if err := c.run(opts...); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput", err)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Errorf("spans = %+v, want one span with error status and the error recorded", spans)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.operations")); n != 0 {
				t.Errorf("%d operations counted, want none for a returned error", n)
			}
			assertInvalidInputCounted(t, rec, c.operation)
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-148, FR-070, FR-096 and doc/rules/testing.md section 4 (immutability) - a use case alters none of the lists it is given.
// Path: unhappy (immutability).
func TestMagicalEffectsUseCasesLeaveTheirInputsAlone(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pending, chosen := mePending("a", "b", "c"), mePending("c", "a", "b")
	if _, err := combat.NewResolveEffectOrder().Run(ctx, combat.ResolveEffectOrderCommand{Holder: combat.TurnHolderPlayer, Pending: pending, Chosen: chosen}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(pending, mePending("a", "b", "c")) || !slices.Equal(chosen, mePending("c", "a", "b")) {
		t.Errorf("the lists changed: %v %v", pending, chosen)
	}
	effects := meEffects(meBless(), meCurse("c1"))
	if _, err := combat.NewRemoveCombatantCurse().Run(ctx, combat.RemoveCombatantCurseCommand{Creature: gid("hero"), Effects: effects, Curse: meEID("c1"), Means: combat.RemoveCurseSpell}); err != nil {
		t.Fatal(err)
	}
	if !effects.Equal(meEffects(meBless(), meCurse("c1"))) {
		t.Error("the effects list changed")
	}
}
