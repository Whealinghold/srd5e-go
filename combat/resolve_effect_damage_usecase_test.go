package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// planFrom ends a one-stage effect on the registry and returns the damage plan it called for.
func planFrom(t *testing.T, actions []combat.EffectAction, audience string) combat.EffectDamagePlan {
	t.Helper()
	tpl := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{aeStage(t, combat.EffectStageParams{Duration: aeRounds(t, 1), OnEnd: actions})}}))
	reg := aeSeq(t, aeSpec{id: "e", spell: "bless", caster: "wiz", subject: aeCreature2(audience), template: tpl})
	plans := aeAdvance(t, reg, aeElapsed(t, 1), "").DamagePlans()
	if len(plans) != 1 {
		t.Fatalf("plans = %v, want one", plans)
	}
	return plans[0]
}

func areaPlan(t *testing.T, expr string, save combat.EffectDamageSave, hasSave bool) combat.EffectDamagePlan {
	t.Helper()
	d := aeDeal(t, expr, content.DamageFire, combat.EffectAudienceAreaMembers)
	if hasSave {
		d = d.WithSave(save)
	}
	return planFrom(t, []combat.EffectAction{d}, "p")
}

func runDamage(t *testing.T, roller combat.Roller, cmd combat.ResolveEffectDamageCommand, opts ...combat.Option) (combat.ResolveEffectDamageResult, error) {
	t.Helper()
	uc, err := combat.NewResolveEffectDamage(roller, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return uc.Run(context.Background(), cmd)
}

func targets(attempts map[string]d20.SaveAttempt, order ...string) []combat.EffectDamageTarget {
	var out []combat.EffectDamageTarget
	for _, id := range order {
		out = append(out, combat.EffectDamageTarget{Target: combat.MustCombatantID(id), Attempt: attempts[id]})
	}
	return out
}

// Requirement: doc/specs/active-effects.md FR-036 (SRD 5.2.1, "Saving Throws and Damage", "Half Damage", p. 16) - the damage is rolled once for all targets, one Damage of the plan's type per target in command order; a success on a save of half damage marks HalfDamage, a success on a save of none gives 0 damage; a failure or no save takes full damage; nothing is applied.
// Path: per case.
func TestResolveEffectDamage(t *testing.T) {
	t.Parallel()
	half := areaPlan(t, "2d6", aeSaveOf(t, ability.Dexterity, 13, combat.SaveSuccessHalfDamage), true)
	none := areaPlan(t, "2d6", aeSaveOf(t, ability.Dexterity, 13, combat.SaveSuccessNoDamage), true)
	plain := areaPlan(t, "2d6", combat.EffectDamageSave{}, false)
	attempts := map[string]d20.SaveAttempt{"a": aeRoll(), "b": aeRoll(), "c": d20.ChooseToFail{}}
	cases := []struct {
		name  string
		plan  combat.EffectDamagePlan
		faces []int
		want  []struct {
			amount int
			half   bool
			saved  bool
		}
		dice int
	}{
		{"half damage: one passes, one fails", half, []int{3, 4, 13, 12}, []struct {
			amount int
			half   bool
			saved  bool
		}{{7, true, true}, {7, false, false}}, 3},
		{"none: a success takes nothing", none, []int{3, 4, 20, 1}, []struct {
			amount int
			half   bool
			saved  bool
		}{{0, false, true}, {7, false, false}}, 3},
		{"no save: everyone takes the roll and no d20 is rolled", plain, []int{6, 6}, []struct {
			amount int
			half   bool
			saved  bool
		}{{12, false, false}, {12, false, false}}, 1},
	}
	for _, c := range cases {
		roller := dicetest.NewFixedRoller(c.faces...)
		res, err := runDamage(t, roller, combat.ResolveEffectDamageCommand{Plan: c.plan, Targets: targets(attempts, "a", "b")})
		out, ok := res.(combat.ResolveEffectDamageResolved)
		if err != nil || !ok || len(roller.Requests()) != c.dice || roller.Requests()[0] != dice.MustParseExpression("2d6") {
			t.Errorf("%s: = %#v, %v, dice %v, want resolved with the damage rolled first", c.name, res, err, roller.Requests())
			continue
		}
		as := out.Assignments()
		for i, w := range c.want {
			_, isHalf := as[i].Adjustment.(combat.HalfDamage)
			if as[i].Damage.Amount() != w.amount || as[i].Damage.Type() != content.DamageFire || isHalf != w.half || (as[i].Save != nil) != (c.dice > 1) ||
				(as[i].Save != nil && (as[i].Save.Outcome() == d20.Success) != w.saved) || as[i].Target != combat.CombatantID([]string{"a", "b"}[i]) {
				t.Errorf("%s: assignment %d = %+v, want %+v [FR-036]", c.name, i, as[i], w)
			}
		}
		if out.Rolled() != c.want[1].amount && out.Rolled() != c.want[0].amount {
			t.Errorf("%s: Rolled = %d", c.name, out.Rolled())
		}
	}
	roller := dicetest.NewFixedRoller(2, 2)
	res, _ := runDamage(t, roller, combat.ResolveEffectDamageCommand{Plan: half, Targets: targets(attempts, "c")})
	if a := res.(combat.ResolveEffectDamageResolved).Assignments()[0]; a.Adjustment != nil || a.Damage.Amount() != 4 || len(roller.Requests()) != 1 {
		t.Errorf("a creature that chose to fail took %+v with dice %v, want full damage and no d20", a, roller.Requests())
	}
}

// Requirement: doc/specs/active-effects.md FR-036 - a damage roll below 0 (a negative modifier) is 0 damage, never negative.
// Path: unhappy (boundary).
func TestResolveEffectDamageNeverNegative(t *testing.T) {
	t.Parallel()
	plan := areaPlan(t, "1d4-5", combat.EffectDamageSave{}, false)
	res, err := runDamage(t, dicetest.NewFixedRoller(2), combat.ResolveEffectDamageCommand{Plan: plan, Targets: targets(nil, "a")})
	out := res.(combat.ResolveEffectDamageResolved)
	if err != nil || out.Rolled() != 0 || out.Assignments()[0].Damage.Amount() != 0 {
		t.Errorf("= %+v, %v, want 0 damage", out, err)
	}
}

// Requirement: doc/specs/active-effects.md FR-039, FR-058, FR-063 - an empty target list (refused), a target listed twice (invalid) and a target without a needed attempt (invalid, listing every one) are rejected in that order before any die.
// Path: per case.
func TestResolveEffectDamageRejections(t *testing.T) {
	t.Parallel()
	saveplan := areaPlan(t, "2d6", aeSaveOf(t, ability.Dexterity, 13, combat.SaveSuccessHalfDamage), true)
	noSave := areaPlan(t, "2d6", combat.EffectDamageSave{}, false)
	two := []combat.EffectDamageTarget{{Target: "a"}, {Target: "a"}}
	cases := []struct {
		name string
		cmd  combat.ResolveEffectDamageCommand
		want []combat.ActiveEffectRejectionKind
	}{
		{"no targets", combat.ResolveEffectDamageCommand{Plan: saveplan}, []combat.ActiveEffectRejectionKind{combat.EffectRejectNoTargets}},
		{"a target twice, and missing attempts too", combat.ResolveEffectDamageCommand{Plan: saveplan, Targets: two}, []combat.ActiveEffectRejectionKind{combat.EffectRejectDuplicateTarget}},
		{"a target twice with no save", combat.ResolveEffectDamageCommand{Plan: noSave, Targets: two}, []combat.ActiveEffectRejectionKind{combat.EffectRejectDuplicateTarget}},
		{"missing attempts", combat.ResolveEffectDamageCommand{Plan: saveplan, Targets: []combat.EffectDamageTarget{{Target: "a"}, {Target: "b", Attempt: aeRoll()}, {Target: "c"}}},
			[]combat.ActiveEffectRejectionKind{combat.EffectRejectMissingSaveAttempt, combat.EffectRejectMissingSaveAttempt}},
	}
	for _, c := range cases {
		roller := dicetest.NewMinimumRoller()
		res, err := runDamage(t, roller, c.cmd)
		rej, ok := res.(combat.ResolveEffectDamageRejected)
		var got []combat.ActiveEffectRejectionKind
		if ok {
			for _, p := range rej.Rejection().Problems() {
				got = append(got, p.Reason)
			}
		}
		if err != nil || !ok || !slices.Equal(got, c.want) || len(roller.Requests()) != 0 {
			t.Errorf("%s: = %#v, %v, problems %v, dice %d, want %v [FR-039]", c.name, res, err, got, len(roller.Requests()), c.want)
		}
	}
	missing := cases[3]
	res, _ := runDamage(t, dicetest.NewMinimumRoller(), missing.cmd)
	if p := res.(combat.ResolveEffectDamageRejected).Rejection().Problems(); p[0].Combatant != "a" || p[1].Combatant != "c" || p[0].Ability != ability.Dexterity {
		t.Errorf("missing pairs = %v, want a and c at Dexterity", p)
	}
}

// Requirement: doc/specs/active-effects.md FR-056 - a failing or panicking roller is a *srderr.PortError with no partial result; a cancelled context, a nil roller, the zero use case and a malformed command are errors.
// Path: unhappy (collaborator failure).
func TestResolveEffectDamageFailures(t *testing.T) {
	t.Parallel()
	saveplan := areaPlan(t, "2d6", aeSaveOf(t, ability.Dexterity, 13, combat.SaveSuccessHalfDamage), true)
	cmd := combat.ResolveEffectDamageCommand{Plan: saveplan, Targets: targets(map[string]d20.SaveAttempt{"a": aeRoll()}, "a")}
	cause := errors.New("down")
	calls := 0
	failsOnSave := dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
		calls++
		if e.Die() == dice.D20() {
			return dice.Result{}, cause
		}
		return dicetest.NewMinimumRoller().Roll(e)
	})
	for name, c := range map[string]struct {
		r  combat.Roller
		is error
	}{
		"the damage roll fails": {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }), cause},
		"a save roll fails":     {failsOnSave, cause},
		"a roller panics":       {dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }), srderr.ErrPortPanic},
	} {
		res, err := runDamage(t, c.r, cmd)
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || !errors.Is(err, c.is) {
			t.Errorf("%s: = %v, %v, want a nil result and a PortError", name, res, err)
		}
	}
	if calls != 2 {
		t.Errorf("the failing save was reached after %d rolls, want 2", calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	uc, _ := combat.NewResolveEffectDamage(dicetest.NewMinimumRoller())
	if res, err := uc.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, %v", res, err)
	}
	if _, err := combat.NewResolveEffectDamage(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller err = %v", err)
	}
	var zero combat.ResolveEffectDamage
	if res, err := zero.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	for name, bad := range map[string]combat.ResolveEffectDamageCommand{
		"the zero plan": {Targets: targets(nil, "a")},
		"a zero target": {Plan: saveplan, Targets: []combat.EffectDamageTarget{{Attempt: aeRoll()}}},
	} {
		if res, err := uc.Run(context.Background(), bad); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: = %v, %v, want an ErrInvalidInput error", name, res, err)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-035, FR-057 - a plan exposes copies; its targets name the creature of a subject audience; the command and result are not mutated by a caller's slice.
// Path: unhappy (immutability).
func TestEffectDamagePlanAndResultAreCopies(t *testing.T) {
	t.Parallel()
	plan := planFrom(t, []combat.EffectAction{aeDeal(t, "1d8", content.DamageAcid, combat.EffectAudienceSubject)}, "orc")
	plan.Targets()[0] = "clobbered"
	if !slices.Equal(plan.Targets(), []combat.CombatantID{"orc"}) || plan.IsZero() || !(combat.EffectDamagePlan{}).IsZero() {
		t.Error("a caller's slice changed the plan or IsZero is wrong")
	}
	if _, has := plan.Save(); has {
		t.Error("the plan has a save it was not given")
	}
	in := targets(nil, "orc")
	res, _ := runDamage(t, dicetest.NewMinimumRoller(), combat.ResolveEffectDamageCommand{Plan: plan, Targets: in})
	in[0].Target = "other"
	out := res.(combat.ResolveEffectDamageResolved)
	out.Assignments()[0] = combat.EffectDamageAssignment{}
	if out.Assignments()[0].Target != "orc" {
		t.Error("a caller's slice changed the result")
	}
}
