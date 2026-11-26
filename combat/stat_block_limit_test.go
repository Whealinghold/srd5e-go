package combat_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// sbStartFrom starts the Multiattack of m with the limited-use state of the small monster of the same id: a stat block over
// the part cap cannot make a state of its own.
func sbStartFrom(tb testing.TB, m content.Monster) (combat.MultiattackStartOutcome, error) {
	tb.Helper()
	return combat.StartMultiattack(m, sbSelf(), luStart(tb, luMonster(tb, nil), combat.OutsideLair))
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - StartMultiattack takes a stat block of at most 64 parts whose plans hold at most 64 variants, 64 attack groups and 64 replacement rules, and UseStatBlock one of at most 64 parts; one more of any is a LimitExceededError naming it and nothing changes.
// Path: unhappy (boundary).
func TestStartMultiattackCaps(t *testing.T) {
	t.Parallel()
	start := func(actions []content.StatBlockPart) error {
		_, err := sbStartFrom(t, luMonster(t, func(f *content.MonsterFields) { f.Actions = actions }))
		return err
	}
	t.Run("variants", func(t *testing.T) {
		t.Parallel()
		runCap(t, "multiattack variants", 64, func(n int) error { return start(msPlanActions(t, n, 1, 0)) })
	})
	t.Run("attack groups", func(t *testing.T) {
		t.Parallel()
		// 8 variants of 8 groups is 64 groups and 13 of 5 is 65.
		if err := start(msPlanActions(t, 8, 8, 0)); err != nil {
			t.Errorf("64 groups: error = %v, want none", err)
		}
		wantLimit(t, start(msPlanActions(t, 13, 5, 0)), "multiattack groups", 64)
	})
	t.Run("replacement rules", func(t *testing.T) {
		t.Parallel()
		runCap(t, "multiattack replacements", 64, func(n int) error { return start(msPlanActions(t, 1, 1, n)) })
	})
	t.Run("parts", func(t *testing.T) {
		t.Parallel()
		// msPlanActions(1, 1, 0) is 3 parts (Spit, Claw 0 and the Multiattack).
		runCap(t, "monster parts", 64, func(n int) error {
			return start(append(luActions(n-3, content.PartUnlimited{}), msPlanActions(t, 1, 1, 0)...))
		})
	})
	t.Run("UseStatBlock", func(t *testing.T) {
		t.Parallel()
		state := luStart(t, luMonster(t, nil), combat.OutsideLair)
		req := combat.StatBlockUseRequest{Cost: combat.StatBlockCostAction, Part: luRef(t, combat.MonsterPartActions, 0)}
		runCap(t, "monster parts", 64, func(n int) error {
			_, err := combat.UseStatBlock(luMonster(t, func(f *content.MonsterFields) { f.Actions = luActions(n, content.PartUnlimited{}) }), sbSelf(), req, state)
			return err
		})
	})
}

// sbStepMonster is a monster whose Multiattack allows 100 attacks with Claw and replacing an attack with Spit any number of times.
func sbStepMonster(tb testing.TB) content.Monster {
	tb.Helper()
	rules := []content.ReplacementRule{sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceAnyNumber, sbOption("Spit"))}
	ma := sbMultiattack(sbVariant("", sbGroups(sbGroup(100, "Claw")), nil, rules...))
	return sbMonster(tb, &ma, sbPlain("Claw", "Spit"), nil)
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - one MultiattackProgress holds at most 64 steps (attacks, additional uses and replacements together) and 64 replacements; the 65th is a LimitExceededError from RecordAttack, RecordAdditionalUse or RecordReplacement and the progress is unchanged.
// Path: unhappy (boundary).
func TestMultiattackStepCaps(t *testing.T) {
	t.Parallel()
	m := sbStepMonster(t)
	state := luStart(t, m, combat.OutsideLair)
	claw, spit := sbAction(t, m, "Claw"), sbAction(t, m, "Spit")
	progress := func() combat.MultiattackProgress {
		out, err := combat.StartMultiattack(m, sbSelf(), state)
		started, ok := out.(combat.MultiattackStarted)
		if err != nil || !ok {
			t.Fatalf("StartMultiattack = %#v, %v", out, err)
		}
		return started.Progress()
	}
	// step records one step of kind and returns the progress after it and the error.
	record := func(p combat.MultiattackProgress, kind string) (combat.MultiattackProgress, error) {
		var out combat.MultiattackStepOutcome
		var err error
		switch kind {
		case "attack":
			out, err = p.RecordAttack(claw, state)
		case "additional use":
			out, err = p.RecordAdditionalUse(spit, state)
		default:
			out, err = p.RecordReplacement(sbOption("Spit"), claw, state)
		}
		if rec, ok := out.(combat.MultiattackStepRecorded); ok && err == nil {
			return rec.Transition.Progress(), nil
		}
		return p, err
	}
	t.Run("steps", func(t *testing.T) {
		t.Parallel()
		runCap(t, "multiattack steps", 64, func(n int) error {
			p, err := progress(), error(nil)
			for range n {
				if p, err = record(p, "attack"); err != nil {
					break
				}
			}
			if err != nil && len(p.Attacks()) != 64 {
				t.Errorf("attacks after the refused step = %d, want 64 (unchanged)", len(p.Attacks()))
			}
			return err
		})
	})
	t.Run("a replacement after 64 attacks", func(t *testing.T) {
		t.Parallel()
		p := progress()
		for range 64 {
			p, _ = record(p, "attack")
		}
		for _, kind := range []string{"additional use", "replacement"} {
			if _, err := record(p, kind); err == nil {
				t.Errorf("%s as the 65th step: error = nil, want LimitExceededError", kind)
			} else {
				wantLimit(t, err, "multiattack steps", 64)
			}
		}
	})
	t.Run("replacements", func(t *testing.T) {
		t.Parallel()
		runCap(t, "multiattack replacements", 64, func(n int) error {
			p, err := progress(), error(nil)
			for range n {
				if p, err = record(p, "replacement"); err != nil {
					break
				}
			}
			return err
		})
	})
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - the replacement search of a step is bounded: 19 identical once-only rules and a 20th replacement is a search of 19 factorial picks, which the node budget ends, so the step is refused (MultiattackReplacementLimitReached) instead of never returning.
// Path: unhappy (rule interaction).
func TestReplacementSearchIsBounded(t *testing.T) {
	t.Parallel()
	m := sbMultiattackOnlyOnce(t, 19)
	state := luStart(t, m, combat.OutsideLair)
	out, err := combat.StartMultiattack(m, sbSelf(), state)
	started, ok := out.(combat.MultiattackStarted)
	if err != nil || !ok {
		t.Fatalf("StartMultiattack = %#v, %v", out, err)
	}
	p, claw := started.Progress(), sbAction(t, m, "Claw")
	for i := range 19 {
		step, err := p.RecordReplacement(sbOption("Spit"), claw, state)
		rec, ok := step.(combat.MultiattackStepRecorded)
		if err != nil || !ok {
			t.Fatalf("replacement %d = %#v, %v, want it recorded", i+1, step, err)
		}
		p = rec.Transition.Progress()
	}
	step, err := p.RecordReplacement(sbOption("Spit"), claw, state)
	refused, ok := step.(combat.MultiattackNotInAllowance)
	if err != nil || !ok || refused.Reason != combat.MultiattackReplacementLimitReached {
		t.Errorf("the 20th replacement = %#v, %v, want NotInAllowance(ReplacementLimitReached)", step, err)
	}
}

// sbMultiattackOnlyOnce is a monster whose only replacement rules are n copies of a once-only rule replacing one attack with Spit.
func sbMultiattackOnlyOnce(tb testing.TB, n int) content.Monster {
	tb.Helper()
	rules := make([]content.ReplacementRule, n)
	for i := range rules {
		rules[i] = sbRule(1, content.ReplacementScopeAnyAttack{}, content.ReplaceOnce, sbOption("Spit"))
	}
	ma := sbMultiattack(sbVariant("", sbGroups(sbGroup(100, "Claw")), nil, rules...))
	return sbMonster(tb, &ma, sbPlain("Claw", "Spit"), nil)
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - ResolveStatBlockSaveEffect takes at most 256 targets; the 257th is a LimitExceededError and the input is unchanged.
// Path: unhappy (boundary).
func TestResolveStatBlockSaveEffectTargetCap(t *testing.T) {
	t.Parallel()
	part := sbMephitBreath()
	runCap(t, "stat block save targets", 256, func(n int) error {
		targets := make([]combat.CombatantID, n)
		for i := range targets {
			targets[i] = combat.MustCombatantID(fmt.Sprintf("target_%d", i))
		}
		in := combat.StatBlockSaveEffectInput{Part: part, Ref: luRef(t, combat.MonsterPartActions, 0), Source: sbSelf(), Targets: targets}
		out, err := combat.ResolveStatBlockSaveEffect(in)
		if err == nil && len(out) != n {
			t.Errorf("results = %d, want %d", len(out), n)
		}
		if len(in.Targets) != n || in.Targets[0] != combat.MustCombatantID("target_0") {
			t.Error("the input changed")
		}
		return err
	})
}

// Requirement: doc/specs/monster-stat-block-actions.md Limits (plan P10) - RollStatBlockDamage takes at most 64 damage instances; the 65th is an Invalid result named limit_exceeded (Err is a LimitExceededError, class invalid, no errors count) found before any roll, and the command is unchanged.
// Path: unhappy (boundary).
func TestRollStatBlockDamageInstanceCap(t *testing.T) {
	t.Parallel()
	build := func(n int) combat.RollStatBlockDamageCommand {
		damage := make([]content.StatBlockDamage, n)
		for i := range damage {
			damage[i] = sbDamage(1, dice.Expression{}, content.DamageFire)
		}
		return combat.RollStatBlockDamageCommand{Damage: damage, Choices: sbNumbers(t)}
	}
	t.Run("at the cap", func(t *testing.T) {
		t.Parallel()
		res, err := sbRollUse(t, dicetest.NewMinimumRoller()).Run(context.Background(), build(64))
		if got := sbRolled(t, res, err); len(got) != 64 {
			t.Errorf("damage = %d instances, want 64", len(got))
		}
	})
	t.Run("cap plus one", func(t *testing.T) {
		t.Parallel()
		rec, opts := recording()
		roller := dicetest.NewMinimumRoller()
		cmd := build(65)
		res, err := sbRollUse(t, roller, opts...).Run(context.Background(), cmd)
		inv, ok := res.(combat.StatBlockDamageInvalid)
		if err != nil || !ok || inv.Err() == nil || len(roller.Requests()) != 0 || len(cmd.Damage) != 65 {
			t.Fatalf("Run = %#v, %v, requests %v, want Invalid with LimitExceeded and no roll", res, err, roller.Requests())
		}
		wantLimit(t, inv.Err(), "stat block damage instances", 64)
		spans := rec.Tracer.SpansNamed("srd5e.combat.roll_stat_block_damage")
		want := map[string]any{"srd5e.combat.operation": "roll_stat_block_damage", "srd5e.outcome": "failure", "srd5e.combat.result": "limit_exceeded", "srd5e.combat.result_class": "invalid"}
		if len(spans) != 1 || !mapsEqual(attrMap(spans[0].Attrs), want) || len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")) != 0 {
			t.Errorf("spans = %+v, want one span with %v and no errors count", spans, want)
		}
	})
	t.Run("a content number is still its own result", func(t *testing.T) {
		t.Parallel()
		cmd := combat.RollStatBlockDamageCommand{Damage: []content.StatBlockDamage{{}}, Choices: sbNumbers(t)}
		res, err := sbRollUse(t, dicetest.NewMinimumRoller()).Run(context.Background(), cmd)
		if inv, ok := res.(combat.StatBlockDamageInvalid); err != nil || !ok || inv.Err() != nil {
			t.Errorf("Run = %#v, %v, want Invalid without LimitExceeded", res, err)
		}
	})
}
