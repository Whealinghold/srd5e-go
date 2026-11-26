package combat_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// resRow is one signal row: a use case run on a fixed script, the operation it records and the result class and name.
type resRow struct {
	name, requirement, path string
	operation               string
	run                     func(opts ...combat.Option) (any, error)
	class, result           string
	deaths                  []string
}

func rowAttack(tb testing.TB, p combat.PerformAttackParams, faces ...int) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewPerformAttackCommand(p)
		if err != nil {
			tb.Fatal(err)
		}
		uc, err := combat.NewPerformAttack(dicetest.NewFixedRoller(faces...), opts...)
		if err != nil {
			tb.Fatal(err)
		}
		return uc.Run(context.Background(), cmd)
	}
}

func rowUnarmed(tb testing.TB, p combat.PerformUnarmedStrikeParams, faces ...int) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewPerformUnarmedStrikeCommand(p)
		if err != nil {
			tb.Fatal(err)
		}
		uc, err := combat.NewPerformUnarmedStrike(dicetest.NewFixedRoller(faces...), opts...)
		if err != nil {
			tb.Fatal(err)
		}
		return uc.Run(context.Background(), cmd)
	}
}

func rowTake(tb testing.TB, e combat.Encounter, l combat.ActionLedger, who string, req combat.TurnRequest) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewTakeCombatantActionCommand(e, l, cid(who), req)
		if err != nil {
			tb.Fatal(err)
		}
		return combat.NewTakeCombatantAction(opts...).Run(context.Background(), cmd)
	}
}

func rowMove(tb testing.TB, e combat.Encounter, who string, req combat.MovementRequest) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewMoveCombatantCommand(e, cid(who), req)
		if err != nil {
			tb.Fatal(err)
		}
		return combat.NewMoveCombatant(opts...).Run(context.Background(), cmd)
	}
}

func rowAdvance(tb testing.TB, e combat.Encounter, updates []combat.SpeedUpdate) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewAdvanceCombatTurnCommand(e, combat.ActionLedger{}, updates, nil)
		if err != nil {
			tb.Fatal(err)
		}
		return combat.NewAdvanceCombatTurn(opts...).Run(context.Background(), cmd)
	}
}

func rowThrow(tb testing.TB, e combat.Encounter, who string, faces ...int) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewRollCombatantDeathSavingThrowCommand(e, cid(who), nil, nil)
		if err != nil {
			tb.Fatal(err)
		}
		uc, err := combat.NewRollCombatantDeathSavingThrow(dicetest.NewFixedRoller(faces...), opts...)
		if err != nil {
			tb.Fatal(err)
		}
		return uc.Run(context.Background(), cmd)
	}
}

func rowStabilize(tb testing.TB, p combat.AttemptToStabilizeCombatantParams, faces ...int) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewAttemptToStabilizeCombatantCommand(p)
		if err != nil {
			tb.Fatal(err)
		}
		uc, err := combat.NewAttemptToStabilizeCombatant(dicetest.NewFixedRoller(faces...), opts...)
		if err != nil {
			tb.Fatal(err)
		}
		return uc.Run(context.Background(), cmd)
	}
}

func rowDeal(tb testing.TB, e combat.Encounter, target string, amount int) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		cmd, err := combat.NewDealDamageToCombatantCommand(e, cid(target), []combat.Damage{resDamage(amount, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
		if err != nil {
			tb.Fatal(err)
		}
		uc, err := combat.NewDealDamageToCombatant(dicetest.NewFixedRoller(), opts...)
		if err != nil {
			tb.Fatal(err)
		}
		return uc.Run(context.Background(), cmd)
	}
}

func resRows(t *testing.T) []resRow {
	t.Helper()
	healthy := encABC(t)
	monsterHP := func(hp int) combat.Combatant {
		return encC(t, "b", combat.SideB, status(t, hp, hp, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities()))
	}
	deadB, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	charmed, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: cid("b")}}), resPC(t, "b", combat.SideB))
	cover := meleeParams()
	cover.Cover = []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	attackP := func(e combat.Encounter, attacker, target string, cost combat.AttackCost) combat.PerformAttackParams {
		return resParams(t, e, attacker, target, cost)
	}
	notPossible := attackP(healthy, "a", "b", resBegin())
	notPossible.Attack = attackCommand(t, cover)
	killer, _ := encStart(t, resPC(t, "a", combat.SideA), monsterHP(7))
	grapple := resStrikeParams(t, healthy, "a", "b", grappleOption(t, ability.Strength, strengthSave(t)))
	dying := resDyingEncounter(t, 0, 0)
	thrown := resAs[combat.RollCombatantDeathSavingThrowCompleted](t, func() combat.RollCombatantDeathSavingThrowResult {
		r, _, err := resThrow(t, dying, "a", nil, nil, 12)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}()).Encounter()
	dyingSecond, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB))
	stabilizable, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB, combat.Unconscious{}))
	immune, _ := encStart(t, resImmune(t, "a", combat.SideA, content.ConditionProne), resPC(t, "b", combat.SideB))
	grappled, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Grappled{Grappler: cid("g")}), resPC(t, "b", combat.SideB))
	prone, _ := encStart(t, resPC(t, "a", combat.SideA, combat.Prone{}), resPC(t, "b", combat.SideB))
	dodge := combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionDodge}}
	return []resRow{
		{"attack completed", resSpec + "FR-099, FR-100", "happy", "perform_attack", rowAttack(t, attackP(healthy, "a", "b", resBegin()), 14, 4), "applied", "completed", nil},
		{"attack kills a monster", resSpec + "FR-102", "happy", "perform_attack", rowAttack(t, attackP(killer, "a", "b", resBegin()), 14, 4), "applied", "completed", []string{"monster_death"}},
		{"attack: ledger refused", resSpec + "FR-100", "unhappy (state)", "perform_attack", rowAttack(t, attackP(healthy, "a", "b", combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}), 14, 4), "refused", "ledger_refused", nil},
		{"attack: turn refused", resSpec + "FR-100", "unhappy (state)", "perform_attack", rowAttack(t, attackP(healthy, "b", "a", resBegin()), 14, 4), "refused", "turn_refused", nil},
		{"attack: targeting forbidden", resSpec + "FR-100", "unhappy (rule interaction)", "perform_attack", rowAttack(t, attackP(charmed, "a", "b", resBegin()), 14, 4), "refused", "targeting_forbidden", nil},
		{"attack: not possible", resSpec + "FR-100", "unhappy (state)", "perform_attack", rowAttack(t, notPossible, 14, 4), "refused", "attack_not_possible", nil},
		{"attack: unknown combatant", resSpec + "FR-100, FR-101", "unhappy (invalid input)", "perform_attack", rowAttack(t, attackP(healthy, "ghost", "b", resBegin()), 14, 4), "invalid", "unknown_combatant", nil},
		{"attack: dead target", resSpec + "FR-100", "unhappy (state)", "perform_attack", rowAttack(t, attackP(deadB, "a", "b", resBegin()), 14, 4), "refused", "combatant_is_dead", nil},
		{"attack: encounter ended", resSpec + "FR-100", "unhappy (state)", "perform_attack", rowAttack(t, attackP(resEnded(t, healthy), "a", "b", resBegin()), 14, 4), "refused", "encounter_already_ended", nil},
		{"unarmed strike completed", resSpec + "FR-099", "happy", "perform_unarmed_strike", rowUnarmed(t, grapple, 5), "applied", "completed", nil},
		{"unarmed strike: unknown target", resSpec + "FR-100", "unhappy (invalid input)", "perform_unarmed_strike",
			rowUnarmed(t, resStrikeParams(t, healthy, "a", "ghost", grappleOption(t, ability.Strength, strengthSave(t))), 5), "invalid", "unknown_combatant", nil},
		{"action completed", resSpec + "FR-099", "happy", "take_combatant_action", rowTake(t, healthy, combat.ActionLedger{}, "a", dodge), "applied", "completed", nil},
		{"action: turn refused", resSpec + "FR-100", "unhappy (state)", "take_combatant_action", rowTake(t, healthy, combat.ActionLedger{}, "b", dodge), "refused", "turn_refused", nil},
		{"action: ledger refused", resSpec + "FR-100", "unhappy (state)", "take_combatant_action", rowTake(t, healthy, combat.ActionLedger{}, "a", combat.HandleWeapon{Change: combat.WeaponChangeDrop, Timing: combat.WeaponChangeAfterAttack}),
			"refused", "ledger_refused", nil},
		{"move completed", resSpec + "FR-099", "happy", "move_combatant", rowMove(t, healthy, "a", resAlong(t, 10)), "applied", "completed", nil},
		{"move: movement restricted", resSpec + "FR-100", "unhappy (rule interaction)", "move_combatant", rowMove(t, grappled, "a", resAlong(t, 10)), "refused", "movement_restricted", nil},
		{"move: posture mismatch", resSpec + "FR-100", "unhappy (state)", "move_combatant", rowMove(t, prone, "a", combat.DropProne{}), "refused", "posture_mismatch", nil},
		{"move: immune to Prone", resSpec + "FR-100, FR-062", "unhappy (state)", "move_combatant", rowMove(t, immune, "a", combat.DropProne{}), "refused", "immune_to_condition", nil},
		{"advance completed", resSpec + "FR-099", "happy", "advance_combat_turn", rowAdvance(t, healthy, nil), "applied", "completed", nil},
		{"advance: unknown update", resSpec + "FR-100", "unhappy (invalid input)", "advance_combat_turn", rowAdvance(t, healthy, []combat.SpeedUpdate{{Combatant: cid("ghost"), Speeds: trkSpeeds(t, 30)}}), "invalid", "unknown_combatant", nil},
		{"throw completed", resSpec + "FR-099", "happy", "roll_combatant_death_saving_throw", rowThrow(t, dying, "a", 12), "applied", "completed", nil},
		{"throw: not Dying", resSpec + "FR-100", "unhappy (state)", "roll_combatant_death_saving_throw", rowThrow(t, healthy, "a", 12), "refused", "not_dying", nil},
		{"throw: out of turn", resSpec + "FR-100", "unhappy (state)", "roll_combatant_death_saving_throw", rowThrow(t, dyingSecond, "b", 12), "refused", "death_saving_throw_out_of_turn", nil},
		{"throw: second in the turn", resSpec + "FR-100", "unhappy (state)", "roll_combatant_death_saving_throw", rowThrow(t, thrown, "a", 12), "refused", "death_saving_throw_already_made", nil},
		{"stabilize completed", resSpec + "FR-099", "happy", "attempt_to_stabilize_combatant", rowStabilize(t, resStabilizeParams(stabilizable), 12), "applied", "completed", nil},
		{"stabilize: patient not Dying", resSpec + "FR-100", "unhappy (state)", "attempt_to_stabilize_combatant", rowStabilize(t, resStabilizeParams(healthy), 12), "refused", "not_dying", nil},
		{"damage completed", resSpec + "FR-099", "happy", "deal_damage_to_combatant", rowDeal(t, healthy, "b", 5), "applied", "completed", nil},
		{"damage: dead target", resSpec + "FR-100", "unhappy (state)", "deal_damage_to_combatant", rowDeal(t, deadB, "b", 5), "refused", "combatant_is_dead", nil},
		{"damage kills", resSpec + "FR-102", "happy", "deal_damage_to_combatant", rowDeal(t, killer, "b", 9), "applied", "completed", []string{"monster_death"}},
	}
}

// resEventKinds returns the sorted Kinds of the events of a result.
func resEventKinds(v any) []string {
	var out []string
	if e, ok := v.(interface{ Events() []combat.Event }); ok {
		out = encKinds(e.Events())
	}
	slices.Sort(out)
	if out == nil {
		out = []string{}
	}
	return out
}

// resHygiene asserts no log, no misuse, no id as an attribute value, no d20 signal and no error counted.
func resHygiene(t *testing.T, rec *telemetrytest.Recorder, allowErrors bool) {
	t.Helper()
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
	if got := rec.Logger.Records(); len(got) != 0 {
		t.Errorf("logged %v, want nothing [FR-101, FR-104]", got)
	}
	for _, m := range rec.Meter.Measurements() {
		if strings.HasPrefix(m.Name, "srd5e.d20_test.") || (!allowErrors && m.Name == "srd5e.combat.errors") {
			t.Errorf("recorded %s [FR-007, FR-101]", m.Name)
		}
		for _, a := range m.Attrs {
			if s, ok := a.Value().(string); ok && slices.Contains([]string{"a", "b", "c", "ghost", "g", "enc-1"}, s) {
				t.Errorf("%s attribute %s carries the id %q [FR-106]", m.Name, a.Key(), s)
			}
		}
	}
}

// Requirement: per case
// Path: per case.
func TestResolutionUseCaseSignals(t *testing.T) {
	t.Parallel()
	for _, c := range resRows(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			got, err := c.run(opts...)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			plain, err := c.run()
			if err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("the recording bundle and Noop disagree [FR-105]")
			}
			outcome := "success"
			if c.class != "applied" {
				outcome = "failure"
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + c.operation)
			wantErr := c.class == "invalid"
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != wantErr || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span, error status %v, no recorded error [%s]", spans, wantErr, c.requirement)
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
			if kinds := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(kinds, resEventKinds(got)) {
				t.Errorf("event kinds counted = %v, want %v [FR-099]", kinds, resEventKinds(got))
			}
			if deaths := values(measured(rec, "srd5e.combat.deaths"), "srd5e.combat.death_cause"); !slices.Equal(deaths, append([]string{}, c.deaths...)) {
				t.Errorf("deaths = %v, want %v, one per wrapped Died [FR-102]", deaths, c.deaths)
			}
			resHygiene(t, rec, false)
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-095, FR-104, FR-071, FR-083 - a roller that fails, panics or answers another expression is a *srderr.PortError (dice_roller_error) returned unchanged with no result, recorded once on the span and once in srd5e.combat.errors, with no operations sample and no log, and the host's encounter and ledger are as they were (the same call with a working roller still gives the same result).
// Path: per case.
func TestResolutionUseCasesPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	healthy := encABC(t)
	dyingE := resDyingEncounter(t, 0, 0)
	stabilizable, _ := encStart(t, resPC(t, "a", combat.SideA), resDying(t, "b", combat.SideB, combat.Unconscious{}))
	good := func(e dice.Expression) (dice.Result, error) {
		rolls := make([]dice.DieRoll, e.Count())
		for i := range rolls {
			rolls[i] = dice.MustDieRoll(e.Die(), min(e.Die().Sides(), 15))
		}
		return dice.NewResult(e, rolls)
	}
	broken := map[string]func() combat.Roller{
		"fails": func() combat.Roller {
			return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
		},
		"panics": func() combat.Roller {
			return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(cause) })
		},
		"answers another expression": func() combat.Roller {
			return dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) {
				return dice.NewResult(dice.MustParseExpression("1d3"), []dice.DieRoll{dice.MustDieRoll(dice.D3(), 2)})
			})
		},
	}
	grapple := resStrikeParams(t, healthy, "a", "b", grappleOption(t, ability.Strength, strengthSave(t)))
	runs := map[string]func(r combat.Roller, opts ...combat.Option) (any, error){
		"perform_attack": func(r combat.Roller, opts ...combat.Option) (any, error) {
			uc, _ := combat.NewPerformAttack(r, opts...)
			cmd, _ := combat.NewPerformAttackCommand(resParams(t, healthy, "a", "b", resBegin()))
			return uc.Run(context.Background(), cmd)
		},
		"perform_unarmed_strike": func(r combat.Roller, opts ...combat.Option) (any, error) {
			uc, _ := combat.NewPerformUnarmedStrike(r, opts...)
			cmd, _ := combat.NewPerformUnarmedStrikeCommand(grapple)
			return uc.Run(context.Background(), cmd)
		},
		"roll_combatant_death_saving_throw": func(r combat.Roller, opts ...combat.Option) (any, error) {
			uc, _ := combat.NewRollCombatantDeathSavingThrow(r, opts...)
			cmd, _ := combat.NewRollCombatantDeathSavingThrowCommand(dyingE, cid("a"), nil, nil)
			return uc.Run(context.Background(), cmd)
		},
		"attempt_to_stabilize_combatant": func(r combat.Roller, opts ...combat.Option) (any, error) {
			uc, _ := combat.NewAttemptToStabilizeCombatant(r, opts...)
			cmd, _ := combat.NewAttemptToStabilizeCombatantCommand(resStabilizeParams(stabilizable))
			return uc.Run(context.Background(), cmd)
		},
	}
	for op, run := range runs {
		for name, mk := range broken {
			t.Run(op+" "+name, func(t *testing.T) {
				t.Parallel()
				rec, opts := recording()
				res, err := run(mk(), opts...)
				var pe *srderr.PortError
				if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port {
					t.Fatalf("= %v, %v, want nil and a dice_roller *PortError [FR-095]", res, err)
				}
				switch name {
				case "fails":
					if !errors.Is(err, cause) {
						t.Errorf("the cause is not reachable: %v", err)
					}
				case "panics":
					if !errors.Is(err, srderr.ErrPortPanic) {
						t.Errorf("a panic is not an ErrPortPanic: %v", err)
					}
				default:
					if !errors.Is(err, srderr.ErrContractViolation) {
						t.Errorf("a wrong answer is not an ErrContractViolation: %v", err)
					}
				}
				spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
				if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
					t.Errorf("spans = %+v, want one ended span with one recorded error [FR-104]", spans)
				}
				if errs := measured(rec, "srd5e.combat.errors"); len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != op {
					t.Errorf("errors = %v, want exactly one with error.type [FR-104]", errs)
				}
				if m := rec.Meter.MeasurementsNamed("srd5e.combat.operations"); len(m) != 0 {
					t.Errorf("operations recorded %v on a failure", m)
				}
				resHygiene(t, rec, true)
				// The host's state is as it was: a working roller gives a result.
				if again, err := run(dicetest.CallbackRoller(good)); err != nil || again == nil {
					t.Errorf("a working roller after the failure gives %v, %v [FR-095]", again, err)
				}
			})
		}
	}
}

// Requirement: doc/specs/testing.md section 4 (cancellation), combat-action-resolution.md FR-002 - an already-cancelled context is returned unwrapped by every use case before any signal or die.
// Path: unhappy (invalid input).
func TestResolutionUseCasesCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	healthy := encABC(t)
	rec, opts := recording()
	roller := dicetest.NewFixedRoller(14, 4)
	attack, _ := combat.NewPerformAttackCommand(resParams(t, healthy, "a", "b", resBegin()))
	pa, _ := combat.NewPerformAttack(roller, opts...)
	strike, _ := combat.NewPerformUnarmedStrikeCommand(resStrikeParams(t, healthy, "a", "b", grappleOption(t, ability.Strength, strengthSave(t))))
	pu, _ := combat.NewPerformUnarmedStrike(roller, opts...)
	take, _ := combat.NewTakeCombatantActionCommand(healthy, combat.ActionLedger{}, cid("a"), combat.SpendInteraction{})
	move, _ := combat.NewMoveCombatantCommand(healthy, cid("a"), combat.DropProne{})
	adv, _ := combat.NewAdvanceCombatTurnCommand(healthy, combat.ActionLedger{}, nil, nil)
	throw, _ := combat.NewRollCombatantDeathSavingThrowCommand(resDyingEncounter(t, 0, 0), cid("a"), nil, nil)
	rt, _ := combat.NewRollCombatantDeathSavingThrow(roller, opts...)
	stab, _ := combat.NewAttemptToStabilizeCombatantCommand(resStabilizeParams(healthy))
	as, _ := combat.NewAttemptToStabilizeCombatant(roller, opts...)
	deal, _ := combat.NewDealDamageToCombatantCommand(healthy, cid("b"), []combat.Damage{resDamage(3, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
	dd, _ := combat.NewDealDamageToCombatant(roller, opts...)
	for name, run := range map[string]func() (any, error){
		"perform_attack":                    func() (any, error) { return pa.Run(ctx, attack) },
		"perform_unarmed_strike":            func() (any, error) { return pu.Run(ctx, strike) },
		"take_combatant_action":             func() (any, error) { return combat.NewTakeCombatantAction(opts...).Run(ctx, take) },
		"move_combatant":                    func() (any, error) { return combat.NewMoveCombatant(opts...).Run(ctx, move) },
		"advance_combat_turn":               func() (any, error) { return combat.NewAdvanceCombatTurn(opts...).Run(ctx, adv) },
		"roll_combatant_death_saving_throw": func() (any, error) { return rt.Run(ctx, throw) },
		"attempt_to_stabilize_combatant":    func() (any, error) { return as.Run(ctx, stab) },
		"deal_damage_to_combatant":          func() (any, error) { return dd.Run(ctx, deal) },
	} {
		if res, err := run(); res != nil || !errors.Is(err, context.Canceled) {
			t.Errorf("%s = %v, %v, want nil and context.Canceled", name, res, err)
		}
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 || len(roller.Requests()) != 0 {
		t.Errorf("a cancelled call left spans, measurements or dice behind")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-002, FR-096, architecture.md section 4 - a zero use case or a zero command is an error matching ErrInvalidInput, never a panic or a signal; a nil roller is refused by every constructor that takes one.
// Path: unhappy (invalid input).
func TestResolutionUseCasesRefuseZeroValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	rec, opts := recording()
	for name, f := range map[string]func() error{
		"PerformAttack zero": func() error { _, err := combat.PerformAttack{}.Run(ctx, combat.PerformAttackCommand{}); return err },
		"PerformUnarmedStrike zero": func() error {
			_, err := combat.PerformUnarmedStrike{}.Run(ctx, combat.PerformUnarmedStrikeCommand{})
			return err
		},
		"TakeCombatantAction zero command": func() error {
			_, err := combat.NewTakeCombatantAction(opts...).Run(ctx, combat.TakeCombatantActionCommand{})
			return err
		},
		"MoveCombatant zero command": func() error {
			_, err := combat.NewMoveCombatant(opts...).Run(ctx, combat.MoveCombatantCommand{})
			return err
		},
		"AdvanceCombatTurn zero command": func() error {
			_, err := combat.NewAdvanceCombatTurn(opts...).Run(ctx, combat.AdvanceCombatTurnCommand{})
			return err
		},
		"RollCombatantDeathSavingThrow zero": func() error {
			_, err := combat.RollCombatantDeathSavingThrow{}.Run(ctx, combat.RollCombatantDeathSavingThrowCommand{})
			return err
		},
		"AttemptToStabilizeCombatant zero": func() error {
			_, err := combat.AttemptToStabilizeCombatant{}.Run(ctx, combat.AttemptToStabilizeCombatantCommand{})
			return err
		},
		"DealDamageToCombatant zero": func() error {
			_, err := combat.DealDamageToCombatant{}.Run(ctx, combat.DealDamageToCombatantCommand{})
			return err
		},
		"NewPerformAttack nil":                 func() error { _, err := combat.NewPerformAttack(nil); return err },
		"NewPerformUnarmedStrike nil":          func() error { _, err := combat.NewPerformUnarmedStrike(nil); return err },
		"NewRollCombatantDeathSavingThrow nil": func() error { _, err := combat.NewRollCombatantDeathSavingThrow(nil); return err },
		"NewAttemptToStabilizeCombatant nil":   func() error { _, err := combat.NewAttemptToStabilizeCombatant(nil); return err },
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if len(rec.Tracer.Spans()) != 0 {
		t.Errorf("a zero command recorded a span")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-004, FR-005, FR-092 - with the same command and fixed dice a use case returns an equal result on every call, and the command and the encounter that came in are not changed by it.
// Path: unhappy (determinism).
func TestResolutionUseCasesAreDeterministicAndDoNotAlterTheirInput(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	p := resParams(t, e, "a", "b", resBegin())
	cmd, err := combat.NewPerformAttackCommand(p)
	if err != nil {
		t.Fatal(err)
	}
	run := func() combat.PerformAttackResult {
		uc, _ := combat.NewPerformAttack(dicetest.NewFixedRoller(14, 4))
		res, err := uc.Run(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	first, second := run(), run()
	if !reflect.DeepEqual(first, second) || !e.Equal(encABC(t)) || resHP(t, e, "b") != 20 {
		t.Errorf("two runs differ, or the encounter that came in changed [FR-004, FR-005]")
	}
	done := resAs[combat.PerformAttackCompleted](t, first)
	events := done.Events()
	events[0] = nil
	res := done.Resolutions()
	res[0] = combat.DamageResolution{}
	if done.Events()[0] == nil || done.Resolutions()[0].Dealt() != 7 {
		t.Errorf("a returned slice is not a copy [FR-004]")
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-098, plan-phase-4.md P10 - a call that would produce more than 4096 events is an invalid result carrying limit_exceeded and changes nothing; exactly 4096 events complete (the damage events, the grappler's Speed review and a grapple check for each grappled creature), one more does not.
// Path: unhappy (boundary).
func TestDealDamageToCombatantEventCap(t *testing.T) {
	t.Parallel()
	crowd := func(n int) combat.Encounter {
		cs := []combat.Combatant{resPC(t, "g", combat.SideB)}
		for i := range n {
			cs = append(cs, resPC(t, fmt.Sprintf("t%04d", i), combat.SideA, combat.Grappled{Grappler: cid("g")}))
		}
		return resStartWith(t, nil, cs...)
	}
	// Each grappled creature adds one grapple check to the fixed events of the damage and the grappler's own Speed review.
	res, err := resDeal(t, crowd(1), "g", []combat.Damage{resDamage(20, content.DamageSlashing)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	base := len(resAs[combat.DealDamageToCombatantCompleted](t, res).Events()) - 1
	for _, c := range []struct {
		name    string
		grapped int
		events  int
		ok      bool
	}{{"at the cap", 4096 - base, 4096, true}, {"cap plus one", 4096 - base + 1, 0, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := crowd(c.grapped)
			res, err := resDeal(t, e, "g", []combat.Damage{resDamage(20, content.DamageSlashing)}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if c.ok {
				if got := len(resAs[combat.DealDamageToCombatantCompleted](t, res).Events()); got != c.events {
					t.Errorf("%d events, want %d", got, c.events)
				}
				return
			}
			inv := resAs[combat.DealDamageToCombatantInvalid](t, res)
			if inv.Problem() != (combat.LimitExceededProblem{}) || !inv.Encounter().Equal(e) || len(inv.Events()) != 0 {
				t.Errorf("= %#v, want limit_exceeded with the encounter unchanged and no event", inv.Problem())
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-098, plan-phase-4.md P10 - the cap of 4096 events per call holds for an attack too: exactly 4096 events complete, one more is an invalid limit_exceeded with the encounter and ledger unchanged.
// Path: unhappy (boundary).
func TestPerformAttackEventCap(t *testing.T) {
	t.Parallel()
	crowd := func(n int) combat.Encounter {
		cs := []combat.Combatant{resPC(t, "a", combat.SideA), encPC(t, "g", combat.SideB, 5, 5)}
		for i := range n {
			cs = append(cs, resPC(t, fmt.Sprintf("t%04d", i), combat.SideA, combat.Grappled{Grappler: cid("g")}))
		}
		return resStartWith(t, nil, cs...)
	}
	done, _ := resAttack(t, resParams(t, crowd(1), "a", "g", resBegin()), 14, 4)
	base := len(done.Events()) - 1
	atCap, _ := resAttack(t, resParams(t, crowd(4096-base), "a", "g", resBegin()), 14, 4)
	if len(atCap.Events()) != 4096 {
		t.Errorf("%d events at the cap, want 4096", len(atCap.Events()))
	}
	e := crowd(4096 - base + 1)
	res, _, err := resPerform(t, resParams(t, e, "a", "g", resBegin()), 14, 4)
	if err != nil {
		t.Fatal(err)
	}
	inv := resAs[combat.PerformAttackInvalid](t, res)
	if inv.Problem() != (combat.LimitExceededProblem{}) || !inv.Encounter().Equal(e) || !inv.Ledger().Equal(combat.ActionLedger{}) || len(inv.Events()) != 0 {
		t.Errorf("= %#v, want limit_exceeded with nothing changed", inv.Problem())
	}
}
