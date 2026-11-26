package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpAFireBead starts a delayed fire effect on a place (a synthetic Time Span effect that is not Concentration): when it ends
// it deals 8d6 fire to the creatures the host lists, a Dexterity save at DC 15 for half. It dismisses the effect with its
// caster and returns the damage plan the ending produced.
func grpAFireBead(t *testing.T) combat.EffectDamagePlan {
	t.Helper()
	ctx := context.Background()
	save := grpAMust(combat.NewEffectDamageSave(ability.Dexterity, 15, combat.SaveSuccessHalfDamage))
	blast := grpAMust(combat.NewDealDamage(dice.MustParseExpression("8d6"), content.DamageFire, combat.EffectAudienceAreaMembers)).WithSave(save)
	stage := grpAMust(combat.NewEffectStage(combat.EffectStageParams{Duration: grpAMust(combat.NewEffectLastsRounds(10)), OnEnd: []combat.EffectAction{blast}}))
	template := grpAMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{stage}}))
	place := grpAMust(combat.NewPlaceEffectSubject(grpAMust(combat.NewEffectPlaceID("pit"))))
	started, err := combat.NewStartActiveEffect().Run(ctx, combat.StartActiveEffectCommand{
		ID: combat.MustActiveEffectID("bead"), Origin: grpAMust(combat.NewOtherEffectOrigin("fire_bead")), Caster: combat.MustCombatantID("wizard"),
		Subject: place, Template: template, Active: combat.MustCombatantID("wizard")})
	if err != nil {
		t.Fatal(err)
	}
	reg := started.(combat.StartActiveEffectStarted).Effects()
	dismissed := grpAMust(combat.NewDismissed(combat.MustCombatantID("wizard"), false, combat.MustActiveEffectID("bead")))
	res, err := combat.NewEndActiveEffects().Run(ctx, combat.EndActiveEffectsCommand{Effects: reg, Cause: dismissed})
	if err != nil {
		t.Fatal(err)
	}
	ended, ok := res.(combat.EndActiveEffectsEnded)
	if !ok || ended.Effects().Len() != 0 || len(ended.DamagePlans()) != 1 {
		t.Fatalf("dismissing the bead = %#v, want it ended with one damage plan", res)
	}
	return ended.DamagePlans()[0]
}

// Requirement: doc/specs/active-effects.md FR-035, FR-036, FR-038, FR-039 - the wizard dismisses a fire bead: its ending plans 8d6 fire for the creatures in the area; the damage is rolled once and the orc, which fails its Dexterity save, takes all 28 while the goblin, which saves, takes half; each target's save is rolled with its own attempt; nothing was rolled or applied by the plan.
// Path: happy.
func TestActiveEffectsDamageResolutionScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	plan := grpAFireBead(t)
	if plan.Effect() != combat.MustActiveEffectID("bead") || plan.Audience() != combat.EffectAudienceAreaMembers || plan.DamageType() != content.DamageFire || plan.Amount().String() != "8d6" {
		t.Fatalf("plan = %+v, want the bead's 8d6 fire for the creatures in the area", plan)
	}
	if s, ok := plan.Save(); !ok || s.Ability() != ability.Dexterity || s.DifficultyClass() != 15 || s.OnSuccess() != combat.SaveSuccessHalfDamage {
		t.Errorf("save = %+v, want a Dexterity DC 15 save for half", s)
	}
	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	roller := dicetest.NewFixedRoller(3, 4, 5, 6, 1, 2, 3, 4 /* 8d6 = 28 */, 5, 18 /* orc, goblin d20 */)
	uc, err := combat.NewResolveEffectDamage(roller)
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(ctx, combat.ResolveEffectDamageCommand{Plan: plan, Targets: []combat.EffectDamageTarget{{Target: id("orc"), Attempt: attempt}, {Target: id("goblin"), Attempt: attempt}}})
	if err != nil {
		t.Fatal(err)
	}
	resolved, ok := res.(combat.ResolveEffectDamageResolved)
	if !ok || resolved.Rolled() != 28 || len(resolved.Assignments()) != 2 {
		t.Fatalf("resolution = %#v, want 28 rolled and two assignments", res)
	}
	orc, goblin := resolved.Assignments()[0], resolved.Assignments()[1]
	if orc.Target != id("orc") || orc.Damage.Amount() != 28 || orc.Damage.Type() != content.DamageFire || orc.Adjustment != nil || orc.Save.Outcome() != d20.Failure {
		t.Errorf("orc = %+v, want 28 fire, no adjustment, a failed save", orc)
	}
	if goblin.Target != id("goblin") || goblin.Damage.Amount() != 28 || goblin.Adjustment != combat.DamageAdjustment(combat.HalfDamage{}) || goblin.Save.Outcome() != d20.Success {
		t.Errorf("goblin = %+v, want the same 28 marked for half damage after a passed save", goblin)
	}
	if reqs := roller.Requests(); len(reqs) != 3 || reqs[0] != dice.MustParseExpression("8d6") {
		t.Errorf("requests = %v, want the damage once, then one save per target", reqs)
	}
}

// Requirement: doc/specs/active-effects.md FR-039, FR-036, FR-035 - the resolution is refused before any die for an empty target list, a target listed twice, or a target with no save attempt; an unusable plan or a blank target is invalid input; a failing roller is one wrapped port error; a cancelled context is returned unchanged.
// Path: unhappy (invalid input).
func TestActiveEffectsDamageRefusalsScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	plan := grpAFireBead(t)
	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	roller := dicetest.NewFixedRoller(1)
	uc, err := combat.NewResolveEffectDamage(roller)
	if err != nil {
		t.Fatal(err)
	}
	run := func(targets ...combat.EffectDamageTarget) combat.ResolveEffectDamageResult {
		t.Helper()
		res, err := uc.Run(ctx, combat.ResolveEffectDamageCommand{Plan: plan, Targets: targets})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	for name, c := range map[string]struct {
		res  combat.ResolveEffectDamageResult
		want combat.ActiveEffectRejectionKind
	}{
		"no targets":                    {run(), combat.EffectRejectNoTargets},
		"a target listed twice":         {run(combat.EffectDamageTarget{Target: id("orc"), Attempt: attempt}, combat.EffectDamageTarget{Target: id("orc"), Attempt: attempt}), combat.EffectRejectDuplicateTarget},
		"a target with no save attempt": {run(combat.EffectDamageTarget{Target: id("orc")}), combat.EffectRejectMissingSaveAttempt},
	} {
		rej, ok := c.res.(combat.ResolveEffectDamageRejected)
		if !ok || len(rej.Rejection().Reasons()) != 1 || rej.Rejection().Reasons()[0] != c.want {
			t.Errorf("%s: result = %#v, want rejected %v", name, c.res, c.want)
		}
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("refusals rolled %v, want no die", roller.Requests())
	}
	if _, err := uc.Run(ctx, combat.ResolveEffectDamageCommand{Targets: []combat.EffectDamageTarget{{Target: id("orc"), Attempt: attempt}}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero plan err = %v, want ErrInvalidInput", err)
	}
	if _, err := uc.Run(ctx, combat.ResolveEffectDamageCommand{Plan: plan, Targets: []combat.EffectDamageTarget{{Target: "", Attempt: attempt}}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank target err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewResolveEffectDamage(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil roller err = %v, want ErrInvalidInput", err)
	}
	cause := errors.New("dice source down")
	broken, err := combat.NewResolveEffectDamage(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }))
	if err != nil {
		t.Fatal(err)
	}
	_, err = broken.Run(ctx, combat.ResolveEffectDamageCommand{Plan: plan, Targets: []combat.EffectDamageTarget{{Target: id("orc"), Attempt: attempt}}})
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, cause) {
		t.Errorf("a failing roller err = %v, want one *PortError of the dice port wrapping the cause", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := uc.Run(cancelled, combat.ResolveEffectDamageCommand{Plan: plan, Targets: []combat.EffectDamageTarget{{Target: id("orc"), Attempt: attempt}}}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context err = %v, want context.Canceled", err)
	}
}
