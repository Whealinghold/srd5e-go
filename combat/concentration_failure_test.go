package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/combat-action-resolution.md FR-083, FR-095, FR-104, FR-131, 4.13.1 "roller returns an error" - a roller that fails, panics or answers another expression at the Concentration save, or at the save of a stat block effect, is a *srderr.PortError (dice_roller_error) returned with no result, recorded once on the span and once in srd5e.combat.errors, with no operations sample, and the host's encounter and SpellcastingState are as they were (the same call with a working roller still gives its result).
// Path: per case.
func TestSaveRollerFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
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
	in := concTracks(t, true, 0, 0)
	e := concTarget(t, 20, 20)
	damageCmd := func() combat.DealDamageToCombatantCommand {
		cmd, err := combat.NewDealDamageToCombatantCommand(e, cid("b"), []combat.Damage{combat.MustDamage(9, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
		if err != nil {
			t.Fatal(err)
		}
		cmd, err = cmd.WithTargetConcentration(in)
		if err != nil {
			t.Fatal(err)
		}
		return cmd
	}()
	saveParams := sbSaveParams(t, sbEncounter(t), nil, 10)
	saveCmd, err := combat.NewResolveStatBlockSaveCommand(saveParams)
	if err != nil {
		t.Fatal(err)
	}
	runs := map[string]func(r combat.Roller, opts ...combat.Option) (any, error){
		"deal_damage_to_combatant": func(r combat.Roller, opts ...combat.Option) (any, error) {
			uc, _ := combat.NewDealDamageToCombatant(r, opts...)
			return uc.Run(context.Background(), damageCmd)
		},
		"resolve_stat_block_save_effect": func(r combat.Roller, opts ...combat.Option) (any, error) {
			uc, _ := combat.NewResolveStatBlockSave(r, opts...)
			return uc.Run(context.Background(), saveCmd)
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
					t.Fatalf("= %v, %v, want nil and a dice_roller *PortError [FR-083, FR-095]", res, err)
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
				if _, going := in.State().Concentration(); !going {
					t.Errorf("the host's SpellcastingState lost its Concentration [FR-083]")
				}
				if again, err := run(dicetest.CallbackRoller(good)); err != nil || again == nil {
					t.Errorf("a working roller after the failure gives %v, %v", again, err)
				}
			})
		}
	}
}

// Requirement: testing.md section 4 (cancellation), combat-action-resolution.md FR-002 and FR-009 - an already-cancelled context is returned unwrapped by the stat block save use case before any signal or die, and a zero use case or command is an error matching srderr.ErrInvalidInput, never a panic.
// Path: unhappy (state).
func TestResolveStatBlockSaveCancelledAndZero(t *testing.T) {
	t.Parallel()
	cmd, err := combat.NewResolveStatBlockSaveCommand(sbSaveParams(t, sbEncounter(t), nil, 10))
	if err != nil {
		t.Fatal(err)
	}
	roller := newFixed(5)
	rec, opts := recording()
	uc, _ := combat.NewResolveStatBlockSave(roller, opts...)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := uc.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled ctx gives %v, %v, want nil and context.Canceled unwrapped", res, err)
	}
	if len(roller.Requests()) != 0 || len(rec.Tracer.SpansNamed("srd5e.combat.resolve_stat_block_save_effect")) != 0 {
		t.Errorf("a cancelled call rolled or recorded something")
	}
	if _, err := (combat.ResolveStatBlockSave{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Run(context.Background(), combat.ResolveStatBlockSaveCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: %v, want ErrInvalidInput", err)
	}
}
