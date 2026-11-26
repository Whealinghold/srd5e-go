package combat_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/encounter-aggregate.md FR-055, FR-089, testing.md section 4 - a zero command and a zero encounter are ErrInvalidInput errors and a cancelled context is returned unwrapped, both before any signal; the zero use case works silently; a command takes the encounter and Run does not alter it.
// Path: unhappy (invalid input, cancellation).
func TestEncounterUseCasesRejectZeroAndCancelled(t *testing.T) {
	t.Parallel()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	e := encABC(t)
	rec, opts := recording()
	healCmd, _ := combat.NewApplyHealingToCombatantCommand(e, cid("a"), 1)
	start := combat.NewStartEncounterCommand(eid(), encEstablished(t, "a", "b"), []combat.Combatant{encPC(t, "a", combat.SideA, 1, 1), encPC(t, "b", combat.SideB, 1, 1)})
	if _, err := combat.NewApplyHealingToCombatant(opts...).Run(cancelled, healCmd); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx: err = %v, want context.Canceled", err)
	}
	if _, err := combat.NewStartEncounter(opts...).Run(cancelled, start); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx (start): err = %v, want context.Canceled", err)
	}
	if _, err := combat.NewApplyHealingToCombatant(opts...).Run(context.Background(), combat.ApplyHealingToCombatantCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewStartEncounter(opts...).Run(context.Background(), combat.StartEncounterCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero start command: err = %v, want ErrInvalidInput", err)
	}
	if n := len(rec.Meter.Measurements()) + len(rec.Tracer.Spans()); n != 0 {
		t.Errorf("signals recorded before the checks: %d", n)
	}
	var zero combat.ApplyHealingToCombatant
	res, err := zero.Run(context.Background(), healCmd)
	if _, ok := res.(combat.ApplyHealingToCombatantApplied); !ok || err != nil {
		t.Errorf("the zero use case = %#v, %v, want a working silent use case", res, err)
	}
	// The encounter in the command is not altered by Run.
	snapshot := encABC(t)
	dmg, _ := combat.NewApplyDamageToCombatantCommand(e, cid("a"), encSlash(5), nil, combat.OtherSource, combat.NormalHit)
	r1, _ := combat.NewApplyDamageToCombatant().Run(context.Background(), dmg)
	r2, _ := combat.NewApplyDamageToCombatant().Run(context.Background(), dmg)
	if !e.Equal(snapshot) || !reflect.DeepEqual(r1, r2) {
		t.Errorf("Run altered the command's encounter, or two runs of one command differ [FR-049, FR-055]")
	}
	// Two commands built from one encounter give two independent successors.
	other, _ := combat.NewApplyDamageToCombatantCommand(e, cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
	r3, _ := combat.NewApplyDamageToCombatant().Run(context.Background(), other)
	if r1.(combat.ApplyDamageToCombatantApplied).Encounter().Equal(r3.(combat.ApplyDamageToCombatantApplied).Encounter()) {
		t.Errorf("two different commands gave equal successors")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-071, FR-080, FR-073, FR-076, FR-086, FR-094, section 6 - the command constructors reject a zero encounter, a blank id and every invalid value with an ErrInvalidInput error, so no invalid command can be run.
// Path: unhappy (invalid input).
func TestCommandConstructorsRejectInvalidValues(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	z := combat.Encounter{}
	checks := map[string]error{}
	add := func(name string, err error) { checks[name] = err }
	_, err := combat.NewApplyHealingToCombatantCommand(z, cid("a"), 1)
	add("healing zero encounter", err)
	_, err = combat.NewApplyHealingToCombatantCommand(e, "", 1)
	add("healing blank id", err)
	_, err = combat.NewApplyHealingToCombatantCommand(e, cid("a"), -1)
	add("healing negative", err)
	_, err = combat.NewApplyConditionToCombatantCommand(e, cid("a"), nil)
	add("condition nil", err)
	_, err = combat.NewRemoveConditionFromCombatantCommand(e, cid("a"), 0)
	add("kind", err)
	_, err = combat.NewReduceCombatantExhaustionCommand(e, cid("a"), 0)
	add("levels", err)
	_, err = combat.NewApplyDeathSavingThrowToCombatantCommand(e, cid("a"), 21, 21)
	add("natural", err)
	_, err = combat.NewGiveCombatantTemporaryHitPointsCommand(e, cid("a"), 0, combat.AcceptNew)
	add("temporary amount", err)
	_, err = combat.NewGiveCombatantTemporaryHitPointsCommand(e, cid("a"), 1, 0)
	add("temporary choice", err)
	_, err = combat.NewReduceCombatantHitPointMaximumCommand(e, cid("a"), -1)
	add("maximum", err)
	_, err = combat.NewCheckCombatantGrappleCommand(e, cid("a"), -1, 0)
	add("distance", err)
	_, err = combat.NewCheckCombatantGrappleCommand(e, cid("a"), 0, -1)
	add("range", err)
	_, err = combat.NewEscapeCombatantGrappleCommand(e, cid("a"), 0)
	add("outcome", err)
	_, err = combat.NewStabilizeCombatantCommand(z, cid("a"))
	add("stabilize zero", err)
	_, err = combat.NewRecoverStableCombatantCommand(e, "")
	add("recover blank", err)
	_, err = combat.NewReleaseCombatantGrappleCommand(z, cid("a"))
	add("release zero", err)
	_, err = combat.NewApplyDamageToCombatantCommand(e, cid("a"), nil, nil, combat.OtherSource, combat.NormalHit)
	add("damage none", err)
	_, err = combat.NewApplyDamageToCombatantCommand(z, cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
	add("damage zero encounter", err)
	_, err = combat.NewAdvanceEncounterTurnCommand(z, combat.TurnAdvance{})
	add("advance zero", err)
	_, err = combat.NewAdvanceEncounterTurnCommand(e, combat.TurnAdvance{SharedSpaceOccupants: []content.CreatureSize{0}})
	add("advance size", err)
	_, err = combat.NewApplyConditionToCombatantCommand(z, cid("a"), combat.Blinded{})
	add("condition zero encounter", err)
	_, err = combat.NewRemoveConditionFromCombatantCommand(z, cid("a"), content.ConditionBlinded)
	add("remove zero encounter", err)
	_, err = combat.NewReduceCombatantExhaustionCommand(z, cid("a"), 1)
	add("reduce zero encounter", err)
	_, err = combat.NewApplyDeathSavingThrowToCombatantCommand(z, cid("a"), 10, 10)
	add("save zero encounter", err)
	_, err = combat.NewGiveCombatantTemporaryHitPointsCommand(z, cid("a"), 1, combat.AcceptNew)
	add("temporary zero encounter", err)
	_, err = combat.NewReduceCombatantHitPointMaximumCommand(z, cid("a"), 1)
	add("maximum zero encounter", err)
	_, err = combat.NewCheckCombatantGrappleCommand(z, cid("a"), 1, 1)
	add("check zero encounter", err)
	_, err = combat.NewEscapeCombatantGrappleCommand(z, cid("a"), d20.Success)
	add("escape zero encounter", err)
	for name, err := range checks {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	// The command takes its own copy of the lists.
	list := encSlash(2)
	cmd, err := combat.NewApplyDamageToCombatantCommand(e, cid("a"), list, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	list[0] = combat.MustDamage(19, content.DamageSlashing)
	res, _ := combat.NewApplyDamageToCombatant().Run(context.Background(), cmd)
	if got := res.(combat.ApplyDamageToCombatantApplied).Resolutions()[0].Incoming(); got != 2 {
		t.Errorf("the command did not copy its damage list: incoming %d, want 2 [FR-071]", got)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-080, FR-063, FR-073, FR-076, FR-077, FR-086, FR-094 - the boundary values of every command constructor: a healing amount and a maximum reduction of 0, Exhaustion levels 1 and 2, natural 1 and 20, Temporary Hit Points of 1, a distance and range of 0 are accepted; the values one outside, and an outcome that is neither success nor failure, are errors.
// Path: unhappy (boundary).
func TestCommandConstructorBoundaries(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	ok := map[string]error{}
	add := func(m map[string]error, name string, err error) { m[name] = err }
	_, err := combat.NewApplyHealingToCombatantCommand(e, cid("a"), 0)
	add(ok, "healing 0", err)
	_, err = combat.NewReduceCombatantHitPointMaximumCommand(e, cid("a"), 0)
	add(ok, "maximum 0", err)
	_, err = combat.NewReduceCombatantExhaustionCommand(e, cid("a"), 1)
	add(ok, "levels 1", err)
	_, err = combat.NewReduceCombatantExhaustionCommand(e, cid("a"), 2)
	add(ok, "levels 2", err)
	_, err = combat.NewApplyDeathSavingThrowToCombatantCommand(e, cid("a"), 1, 1)
	add(ok, "natural 1", err)
	_, err = combat.NewApplyDeathSavingThrowToCombatantCommand(e, cid("a"), 20, 25)
	add(ok, "natural 20", err)
	_, err = combat.NewGiveCombatantTemporaryHitPointsCommand(e, cid("a"), 1, combat.KeepCurrent)
	add(ok, "temporary 1", err)
	_, err = combat.NewCheckCombatantGrappleCommand(e, cid("a"), 0, 0)
	add(ok, "distance and range 0", err)
	_, err = combat.NewEscapeCombatantGrappleCommand(e, cid("a"), d20.Failure)
	add(ok, "failure", err)
	for name, err := range ok {
		if err != nil {
			t.Errorf("%s: err = %v, want accepted", name, err)
		}
	}
	bad := map[string]error{}
	_, err = combat.NewApplyDeathSavingThrowToCombatantCommand(e, cid("a"), 0, 0)
	add(bad, "natural 0", err)
	_, err = combat.NewEscapeCombatantGrappleCommand(e, cid("a"), 3)
	add(bad, "outcome 3", err)
	_, err = combat.NewGiveCombatantTemporaryHitPointsCommand(e, cid("a"), 1, 3)
	add(bad, "choice 3", err)
	for name, err := range bad {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}
