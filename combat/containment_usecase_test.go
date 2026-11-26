package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// containmentRow is one variant of one containment use case.
type containmentRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	events                  []string
	ended                   []string
	occupants               float64 // -1: no sample
}

func containmentRows(t *testing.T) []containmentRow {
	t.Helper()
	ctx := context.Background()
	enter := func(name string, mod func(*combat.EnterContainerInput), faces ...int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			in := ctEnterInput(t, name)
			if mod != nil {
				mod(&in)
			}
			cmd, err := combat.NewEnterContainerCommand(in)
			if err != nil {
				return nil, err
			}
			uc, err := combat.NewEnterContainer(dicetest.NewFixedRoller(faces...), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, cmd)
		}
	}
	exit := func(name string, method combat.ExitContainerMethod, mod func(*combat.ExitContainerInput), faces ...int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			in := ctExitInput(t, name, method)
			if mod != nil {
				mod(&in)
			}
			cmd, err := combat.NewExitContainerCommand(in)
			if err != nil {
				return nil, err
			}
			uc, err := combat.NewExitContainer(dicetest.NewFixedRoller(faces...), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, cmd)
		}
	}
	advance := func(in combat.AdvanceContainerTurnInput) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewAdvanceContainerTurnCommand(in)
			if err != nil {
				return nil, err
			}
			return combat.NewAdvanceContainerTurn(opts...).Run(ctx, cmd)
		}
	}
	regurgitate := func(in combat.ResolveRegurgitationInput, faces ...int) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewResolveRegurgitationCommand(in)
			if err != nil {
				return nil, err
			}
			uc, err := combat.NewResolveRegurgitation(dicetest.NewFixedRoller(faces...), opts...)
			if err != nil {
				return nil, err
			}
			return uc.Run(ctx, cmd)
		}
	}
	release := func(in combat.ReleaseOccupantsInput) func(...combat.Option) (any, error) {
		return func(opts ...combat.Option) (any, error) {
			cmd, err := combat.NewReleaseOccupantsCommand(in)
			if err != nil {
				return nil, err
			}
			return combat.NewReleaseOccupants(opts...).Run(ctx, cmd)
		}
	}
	roll := d20.CheckAttempt(rollOf(0))
	huge := func(in *combat.EnterContainerInput) { in.OccupantSize = content.SizeHuge }
	occupied := func(in *combat.EnterContainerInput, set combat.Containments) { in.Containments = set }
	frogSet, frogConds := ctSwallowed(t, "frog", false)
	wormSet := ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"p": 31})
	cubeSet, _, cubeConds := ctHeld(t, "cube")
	const table = "monster-engulf-and-swallow.md section 7 closed value sets"
	const a, r, i = "applied", "refused", "invalid"
	return []containmentRow{
		{"enter_container contained", table + "; FR-067, FR-069", "happy", "enter_container", enter("cube", nil, 11), a, "contained", []string{"condition_applied", "occupant_contained"}, []string{}, 1},
		{"enter_container resisted", table + "; FR-024", "happy", "enter_container", enter("cube", func(in *combat.EnterContainerInput) { in.RelocationSpaceAvailable = true }, 12), a, "resisted", []string{}, []string{}, 0},
		{"enter_container damaged_only", table + "; FR-032", "unhappy (rule interaction)", "enter_container", enter("cube", huge, 1), a, "damaged_only", []string{}, []string{}, 0},
		{"enter_container container_is_contained", table + "; FR-019", "unhappy (state error)", "enter_container", enter("worm", func(in *combat.EnterContainerInput) {
			occupied(in, ctSet(t, ctRecord(t, "boss", "m", content.SizeLarge, ctTerms(t, "kraken"), 1)))
		}, 1), r, "container_is_contained", []string{}, []string{}, -1},
		{"enter_container occupant_is_container", table + "; FR-019", "unhappy (state error)", "enter_container", enter("worm", func(in *combat.EnterContainerInput) {
			occupied(in, ctSet(t, ctRecord(t, "p", "x", content.SizeSmall, ctTerms(t, "toad"), 1)))
		}, 1), r, "occupant_is_container", []string{}, []string{}, -1},
		{"enter_container occupant_already_contained", table + "; FR-019", "unhappy (state error)", "enter_container", enter("worm", func(in *combat.EnterContainerInput) {
			occupied(in, ctSet(t, ctRecord(t, "z", "p", content.SizeMedium, ctTerms(t, "worm"), 1)))
		}, 1), r, "occupant_already_contained", []string{}, []string{}, -1},
		{"enter_container not_grappled_by_container", table + "; FR-021", "unhappy (state error)", "enter_container", enter("worm", func(in *combat.EnterContainerInput) { in.Grapples = combat.Grapples{} }, 1),
			r, "not_grappled_by_container", []string{}, []string{}, -1},
		{"enter_container occupant_grappled_elsewhere", table + "; FR-021", "unhappy (state error)", "enter_container", enter("mound", func(in *combat.EnterContainerInput) {
			in.Grapples = grapplesOf(t, held("z", "p", hand(0)))
		}, 1), r, "occupant_grappled_elsewhere", []string{}, []string{}, -1},
		{"enter_container too_large", table + "; FR-021", "unhappy (boundary)", "enter_container", enter("mound", func(in *combat.EnterContainerInput) { in.OccupantSize = content.SizeLarge }, 1), r, "too_large", []string{}, []string{}, -1},
		{"enter_container no_room", table + "; FR-021", "unhappy (boundary)", "enter_container", enter("mound", func(in *combat.EnterContainerInput) {
			occupied(in, ctHolding(t, "mound", "m", "z"))
		}, 1), r, "no_room", []string{}, []string{}, -1},
		{"enter_container same_container_and_occupant", table + "; FR-034", "unhappy (invalid input)", "enter_container", enter("cube", func(in *combat.EnterContainerInput) { in.Occupant = in.Container }, 1),
			i, "same_container_and_occupant", []string{}, []string{}, -1},
		{"enter_container circumstance_not_applicable", table + "; FR-027", "unhappy (invalid input)", "enter_container", enter("worm", func(in *combat.EnterContainerInput) {
			in.Circumstance = combat.EntryCircumstanceEnteredContainerSpace
		}, 1), i, "circumstance_not_applicable", []string{}, []string{}, -1},
		{"exit_container exited", table + "; FR-051, FR-069", "happy", "exit_container", exit("cube", ctSelf(roll), nil, 12), a, "exited", []string{"condition_removed", "occupant_released"}, []string{"escaped"}, 0},
		{"exit_container exited by pull", table + "; FR-052, FR-069", "happy", "exit_container", exit("cube", ctPull("pu", roll), nil, 12), a, "exited", []string{"condition_removed", "occupant_released"}, []string{"pulled_out"}, 0},
		{"exit_container held", table + "; FR-051", "unhappy (boundary)", "exit_container", exit("cube", ctSelf(roll), nil, 11), a, "held", []string{}, []string{}, 1},
		{"exit_container not_contained", table + "; FR-053", "unhappy (state error)", "exit_container", exit("cube", ctSelf(roll), func(in *combat.ExitContainerInput) { in.Occupant = gid("z") }, 12), r, "not_contained", []string{}, []string{}, -1},
		{"exit_container no_way_out", table + "; FR-053", "unhappy (state error)", "exit_container", exit("worm", ctSelf(roll), nil, 12), r, "no_way_out", []string{}, []string{}, -1},
		{"exit_container puller_is_contained", table + "; FR-053", "unhappy (state error)", "exit_container", exit("cube", ctPull("p", roll), nil, 12), r, "puller_is_contained", []string{}, []string{}, -1},
		{"exit_container puller_out_of_reach", table + "; FR-053", "unhappy (state error)", "exit_container", exit("cube", combat.ExitByPull{Puller: gid("pu"), Attempt: roll, Actions: combat.ConditionActionAvailabilityAvailable}, nil, 12),
			r, "puller_out_of_reach", []string{}, []string{}, -1},
		{"exit_container no_action", table + "; FR-053", "unhappy (state error)", "exit_container", exit("cube", combat.ExitBySelf{Attempt: roll, Actions: combat.ConditionActionAvailabilityNone}, nil, 12), r, "no_action", []string{}, []string{}, -1},
		{"advance_container_turn advanced", table + "; FR-043", "happy", "advance_container_turn", advance(combat.AdvanceContainerTurnInput{Containments: wormSet, Container: gid("m"), Boundary: combat.ContainerTurnBoundaryStart}), a, "advanced", []string{}, []string{}, 2},
		{"advance_container_turn advanced, disgorged", table + "; FR-044, FR-069", "unhappy (rule interaction)", "advance_container_turn", advance(combat.AdvanceContainerTurnInput{Containments: frogSet, Container: gid("m"),
			Boundary: combat.ContainerTurnBoundaryEnd, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): frogConds}, Survivors: []combat.CombatantID{gid("p")}}),
			a, "advanced", []string{"condition_removed", "condition_removed", "occupant_released", "condition_applied"}, []string{"disgorged"}, 0},
		{"advance_container_turn missing_occupant_conditions", table + "; FR-063", "unhappy (invalid input)", "advance_container_turn", advance(combat.AdvanceContainerTurnInput{Containments: frogSet, Container: gid("m"),
			Boundary: combat.ContainerTurnBoundaryEnd, Survivors: []combat.CombatantID{gid("p")}}), i, "missing_occupant_conditions", []string{}, []string{}, -1},
		{"advance_container_turn unknown_occupant", table + "; FR-063", "unhappy (invalid input)", "advance_container_turn", advance(combat.AdvanceContainerTurnInput{Containments: wormSet, Container: gid("m"),
			Boundary: combat.ContainerTurnBoundaryStart, OccupantConditions: ctConds(t, "p")}), i, "unknown_occupant", []string{}, []string{}, -1},
		{"resolve_regurgitation regurgitated", table + "; FR-048, FR-069", "happy", "resolve_regurgitation", regurgitate(combat.ResolveRegurgitationInput{Containments: wormSet, Container: gid("m"), OccupantConditions: ctConds(t, "p", "q"), Attempt: rollOf(0)}, 20),
			a, "regurgitated", []string{"condition_removed", "condition_removed", "occupant_released", "condition_applied", "condition_removed", "condition_removed", "occupant_released", "condition_applied"}, []string{"regurgitated", "regurgitated"}, 0},
		{"resolve_regurgitation held", table + "; FR-048", "unhappy (boundary)", "resolve_regurgitation", regurgitate(combat.ResolveRegurgitationInput{Containments: wormSet, Container: gid("m"), OccupantConditions: ctConds(t, "p", "q"), Attempt: rollOf(1)}, 20), a, "held", []string{}, []string{}, 2},
		{"resolve_regurgitation not_due", table + "; FR-047", "unhappy (boundary)", "resolve_regurgitation", regurgitate(combat.ResolveRegurgitationInput{Containments: ctHolding(t, "worm", "m", "p"), Container: gid("m"), Attempt: rollOf(0)}), a, "not_due", []string{}, []string{}, 1},
		{"resolve_regurgitation missing_occupant_conditions", table + "; FR-063", "unhappy (invalid input)", "resolve_regurgitation", regurgitate(combat.ResolveRegurgitationInput{Containments: wormSet, Container: gid("m"), Attempt: rollOf(0)}, 20),
			i, "missing_occupant_conditions", []string{}, []string{}, -1},
		{"resolve_regurgitation unknown_occupant", table + "; FR-063", "unhappy (invalid input)", "resolve_regurgitation", regurgitate(combat.ResolveRegurgitationInput{Containments: wormSet, Container: gid("m"), OccupantConditions: ctConds(t, "p", "q", "zz"), Attempt: rollOf(0)}, 20),
			i, "unknown_occupant", []string{}, []string{}, -1},
		{"release_occupants released", table + "; FR-057, FR-069", "happy", "release_occupants", release(combat.ReleaseOccupantsInput{Containments: cubeSet, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): cubeConds}, Trigger: combat.ContainerDied{Container: gid("m")}}),
			a, "released", []string{"condition_removed", "occupant_released"}, []string{"container_died"}, 0},
		{"release_occupants released, occupant removed", table + "; FR-060", "unhappy (state error)", "release_occupants", release(combat.ReleaseOccupantsInput{Containments: cubeSet, Trigger: combat.OccupantGone{Occupant: gid("p")}}),
			a, "released", []string{"occupant_released"}, []string{"occupant_removed"}, 0},
		{"release_occupants not_contained", table + "; FR-061", "unhappy (state error)", "release_occupants", release(combat.ReleaseOccupantsInput{Containments: cubeSet, Trigger: combat.OccupantGone{Occupant: gid("z")}}), r, "not_contained", []string{}, []string{}, -1},
		{"release_occupants not_trapped", table + "; FR-058", "unhappy (state error)", "release_occupants", release(combat.ReleaseOccupantsInput{Containments: cubeSet, Trigger: combat.EscapedCorpse{Occupant: gid("p"), MovementRemainingFeet: 30}}), r, "not_trapped", []string{}, []string{}, -1},
		{"release_occupants insufficient_movement", table + "; FR-058", "unhappy (boundary)", "release_occupants", release(ctCorpseEscape(t, 14)), r, "insufficient_movement", []string{}, []string{}, -1},
		{"release_occupants missing_occupant_conditions", table + "; FR-063", "unhappy (invalid input)", "release_occupants", release(combat.ReleaseOccupantsInput{Containments: cubeSet, Trigger: combat.ContainerDied{Container: gid("m")}}), i, "missing_occupant_conditions", []string{}, []string{}, -1},
		{"release_occupants unknown_occupant", table + "; FR-063", "unhappy (invalid input)", "release_occupants", release(combat.ReleaseOccupantsInput{Containments: cubeSet, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("zz"): {}}, Trigger: combat.OccupantGone{Occupant: gid("p")}}),
			i, "unknown_occupant", []string{}, []string{}, -1},
	}
}

// ctCorpseEscape is a Behir's trapped meal asking to climb out with movement feet left.
func ctCorpseEscape(tb testing.TB, movement int) combat.ReleaseOccupantsInput {
	tb.Helper()
	died := ctReleased(tb, ctRelease(tb, ctDied(tb, "behir")))
	return combat.ReleaseOccupantsInput{Containments: died.Containments(), OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): died.Trapped()[0].Conditions()},
		Trigger: combat.EscapedCorpse{Occupant: gid("p"), MovementRemainingFeet: movement}}
}

// Requirement: per case
// Path: per case.
func TestContainmentUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range containmentRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			rec.Logger.SetMinLevel(telemetry.LevelTrace)
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-072]", got, plain, err)
			}
			outcome := map[string]string{"applied": "success", "refused": "failure", "invalid": "failure"}[c.class]
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status only for invalid [%s]", spans, c.requirement)
			}
			want := map[string]any{"srd5e.combat.operation": c.operation, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [%s]", ops, want, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v", sa, want)
			}
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, sortedCopy(c.events)) {
				t.Errorf("event kinds counted = %v, want %v [FR-069]", kinds, sortedCopy(c.events))
			}
			if ended := values(measured(rec, "srd5e.combat.containment.ended"), "srd5e.combat.containment.end_reason"); !slices.Equal(ended, c.ended) {
				t.Errorf("ended reasons = %v, want %v [FR-069]", ended, c.ended)
			}
			for _, m := range rec.Meter.MeasurementsNamed("srd5e.combat.containment.ended") {
				if m.Value != 1 || m.Unit != "{containment}" {
					t.Errorf("ended = %+v, want an increment of 1 in {containment}", m)
				}
			}
			samples := rec.Meter.MeasurementsNamed("srd5e.combat.containment.occupants")
			switch {
			case c.occupants < 0 && len(samples) != 0:
				t.Errorf("occupants = %+v, want none for a result with no set [FR-069]", samples)
			case c.occupants >= 0 && (len(samples) != 1 || samples[0].Value != c.occupants || !reflect.DeepEqual(attrMap(samples[0].Attrs), map[string]any{"srd5e.combat.operation": c.operation})):
				t.Errorf("occupants = %+v, want %v with the operation [FR-069]", samples, c.occupants)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want 0 for a result [FR-070]", n)
			}
			logs := rec.Logger.Records()
			if len(logs) != 1 || logs[0].Level != telemetry.LevelDebug || logs[0].Message != "containment operation" ||
				!reflect.DeepEqual(attrMap(logs[0].Attrs), map[string]any{"srd5e.combat.operation": c.operation, "srd5e.combat.result": c.result}) {
				t.Errorf("logs = %+v, want one debug 'containment operation' with the operation and result [FR-073]", logs)
			}
			checkNoD20Signals(t, rec)
		})
	}
}

func sortedCopy(s []string) []string {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-072 - no attribute of any containment signal carries a combatant id, a DC, a count of feet or a part name; every value is one of the closed sets of section 7.
// Path: unhappy (invalid input).
func TestContainmentAttributesAreClosedValues(t *testing.T) {
	t.Parallel()
	closed := []string{"enter_container", "exit_container", "advance_container_turn", "resolve_regurgitation", "release_occupants", "applied", "refused", "invalid", "success", "failure",
		"contained", "resisted", "damaged_only", "exited", "held", "advanced", "regurgitated", "not_due", "released",
		"container_is_contained", "occupant_is_container", "occupant_already_contained", "not_grappled_by_container", "occupant_grappled_elsewhere", "too_large", "no_room",
		"not_contained", "no_way_out", "puller_is_contained", "puller_out_of_reach", "no_action", "not_trapped", "insufficient_movement",
		"same_container_and_occupant", "circumstance_not_applicable", "missing_occupant_conditions", "unknown_occupant",
		"escaped", "pulled_out", "disgorged", "container_died", "escaped_corpse", "grapple_ended", "occupant_removed",
		"condition_applied", "condition_removed", "occupant_contained", "occupant_released", "container_trapped"}
	for _, c := range containmentRows(t) {
		rec, opts := recording()
		if _, err := c.run(opts...); err != nil {
			t.Fatal(err)
		}
		for _, m := range rec.Meter.Measurements() {
			for _, a := range m.Attrs {
				if s, ok := a.Value().(string); !ok || !slices.Contains(closed, s) {
					t.Errorf("%s: %s attribute %s = %v is not a closed value [FR-072]", c.name, m.Name, a.Key(), a.Value())
				}
				if !strings.HasPrefix(a.Key(), "srd5e.") && a.Key() != telemetry.AttrOutcome {
					t.Errorf("%s: attribute key %s is not a srd5e signal key", c.name, a.Key())
				}
			}
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-065, FR-071, errors-and-results.md FR-031 - a roller that fails, panics or answers another expression is a *srderr.PortError (dice_roller_error) with a nil result, recorded once on the span and once in srd5e.combat.errors, with no operations, containment or d20 signal, and nothing applied.
// Path: unhappy (failure of collaborator).
func TestContainmentRollerFailureIsRecordedOnce(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	rollers := map[string]combat.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) }),
		"wrong expression": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
			return dice.NewResult(dice.MustParseExpression("1d6"), []dice.DieRoll{dice.MustDieRoll(dice.D6(), 3)})
		}),
	}
	ctx := context.Background()
	uses := map[string]func(combat.Roller, ...combat.Option) (any, error){
		"enter_container": func(r combat.Roller, o ...combat.Option) (any, error) {
			uc, _ := combat.NewEnterContainer(r, o...)
			cmd, _ := combat.NewEnterContainerCommand(ctEnterInput(t, "cube"))
			return uc.Run(ctx, cmd)
		},
		"exit_container": func(r combat.Roller, o ...combat.Option) (any, error) {
			uc, _ := combat.NewExitContainer(r, o...)
			cmd, _ := combat.NewExitContainerCommand(ctExitInput(t, "cube", ctSelf(rollOf(0))))
			return uc.Run(ctx, cmd)
		},
		"resolve_regurgitation": func(r combat.Roller, o ...combat.Option) (any, error) {
			uc, _ := combat.NewResolveRegurgitation(r, o...)
			cmd, _ := combat.NewResolveRegurgitationCommand(combat.ResolveRegurgitationInput{Containments: ctDamaged(t, ctHolding(t, "worm", "m", "p"), map[string]int{"p": 31}),
				Container: gid("m"), OccupantConditions: ctConds(t, "p"), Attempt: rollOf(0)})
			return uc.Run(ctx, cmd)
		},
	}
	for op, run := range uses {
		for name, roller := range rollers {
			t.Run(op+" "+name, func(t *testing.T) {
				t.Parallel()
				rec, opts := recording()
				res, err := run(roller, opts...)
				var pe *srderr.PortError
				if !errors.As(err, &pe) || pe.Port != dice.Port || !isNilResult(res) {
					t.Fatalf("= %v, %v, want nil and a dice_roller *PortError", res, err)
				}
				if name == "fails" && !errors.Is(err, cause) || name == "wrong expression" && !errors.Is(err, srderr.ErrContractViolation) {
					t.Errorf("cause not reachable: %v", err)
				}
				spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
				if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
					t.Errorf("spans = %+v, want one ended span with one recorded error", spans)
				}
				errs := measured(rec, "srd5e.combat.errors")
				if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != op {
					t.Errorf("errors = %v, want one with error.type", errs)
				}
				for _, n := range []string{"operations", "events", "containment.ended", "containment.occupants"} {
					if m := rec.Meter.MeasurementsNamed("srd5e.combat." + n); len(m) != 0 {
						t.Errorf("%s recorded %d on an error", n, len(m))
					}
				}
				if logs := rec.Logger.Records(); len(logs) != 0 {
					t.Errorf("logged %v, want nothing for an error (never log and return)", logs)
				}
				checkNoD20Signals(t, rec)
			})
		}
	}
}

func isNilResult(v any) bool { return v == nil || reflect.ValueOf(v).IsNil() }

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-065, testing.md section 4 - a nil roller is rejected by the constructors; the zero use cases and commands and a cancelled ctx are refused before any signal.
// Path: unhappy (invalid input, cancellation).
func TestContainmentUseCasesRejectZeroValuesAndCancelledContext(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if _, err := combat.NewEnterContainer(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewEnterContainer(nil) = %v", err)
	}
	if _, err := combat.NewExitContainer(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewExitContainer(nil) = %v", err)
	}
	if _, err := combat.NewResolveRegurgitation(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewResolveRegurgitation(nil) = %v", err)
	}
	roller := dicetest.NewFixedRoller()
	enter, _ := combat.NewEnterContainer(roller)
	exit, _ := combat.NewExitContainer(roller)
	regurg, _ := combat.NewResolveRegurgitation(roller)
	for name, err := range map[string]error{
		"zero enter use case":  second(combat.EnterContainer{}.Run(ctx, combat.EnterContainerCommand{})),
		"zero enter command":   second(enter.Run(ctx, combat.EnterContainerCommand{})),
		"zero exit use case":   second(combat.ExitContainer{}.Run(ctx, combat.ExitContainerCommand{})),
		"zero exit command":    second(exit.Run(ctx, combat.ExitContainerCommand{})),
		"zero regurgitation":   second(combat.ResolveRegurgitation{}.Run(ctx, combat.ResolveRegurgitationCommand{})),
		"zero regurg command":  second(regurg.Run(ctx, combat.ResolveRegurgitationCommand{})),
		"zero advance command": second(combat.AdvanceContainerTurn{}.Run(ctx, combat.AdvanceContainerTurnCommand{})),
		"zero release command": second(combat.ReleaseOccupants{}.Run(ctx, combat.ReleaseOccupantsCommand{})),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	rec, opts := recording()
	enterCmd, _ := combat.NewEnterContainerCommand(ctEnterInput(t, "cube"))
	exitCmd, _ := combat.NewExitContainerCommand(ctExitInput(t, "cube", ctSelf(rollOf(0))))
	regCmd, _ := combat.NewResolveRegurgitationCommand(combat.ResolveRegurgitationInput{Container: gid("m"), Attempt: rollOf(0)})
	advCmd, _ := combat.NewAdvanceContainerTurnCommand(combat.AdvanceContainerTurnInput{Container: gid("m"), Boundary: combat.ContainerTurnBoundaryEnd})
	relCmd, _ := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{Trigger: combat.OccupantGone{Occupant: gid("p")}})
	enter, _ = combat.NewEnterContainer(roller, opts...)
	exit, _ = combat.NewExitContainer(roller, opts...)
	regurg, _ = combat.NewResolveRegurgitation(roller, opts...)
	for name, err := range map[string]error{
		"enter":       second(enter.Run(cancelled, enterCmd)),
		"exit":        second(exit.Run(cancelled, exitCmd)),
		"regurgitate": second(regurg.Run(cancelled, regCmd)),
		"advance":     second(combat.NewAdvanceContainerTurn(opts...).Run(cancelled, advCmd)),
		"release":     second(combat.NewReleaseOccupants(opts...).Run(cancelled, relCmd)),
	} {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s: err = %v, want context.Canceled", name, err)
		}
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a cancelled context must record nothing")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-065, FR-066, FR-073 - the zero-value use cases that need no port work with no options and are silent.
// Path: unhappy (determinism).
func TestContainmentZeroValueUseCasesWork(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	adv, _ := combat.NewAdvanceContainerTurnCommand(combat.AdvanceContainerTurnInput{Containments: ctHolding(t, "worm", "m", "p"), Container: gid("m"), Boundary: combat.ContainerTurnBoundaryStart})
	var a combat.AdvanceContainerTurn
	res, err := a.Run(ctx, adv)
	if err != nil || len(res.(combat.AdvanceContainerTurnAdvanced).DamageDue()) != 1 {
		t.Errorf("zero AdvanceContainerTurn = %#v, %v", res, err)
	}
	rel, _ := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{Containments: ctHolding(t, "worm", "m", "p"), Trigger: combat.OccupantGone{Occupant: gid("p")}})
	var r combat.ReleaseOccupants
	if res, err := r.Run(ctx, rel); err != nil || len(res.(combat.ReleaseOccupantsReleased).Releases()) != 1 {
		t.Errorf("zero ReleaseOccupants = %#v, %v", res, err)
	}
}
