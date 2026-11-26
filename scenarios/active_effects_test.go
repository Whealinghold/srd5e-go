package scenarios_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/active-effects.md section 8 (scenario test), FR-013, FR-015, FR-029, FR-041, FR-045 - a Cleric casts Bless on two allies and a second Cleric on one: the combined query shows one effect on the shared ally; a Wizard casts Hold Person, the target fails its repeated save at the end of its turn and then succeeds, the registry loses the effect, and the planned condition removal goes to RemoveConditionFromSet; then the Wizard loses Concentration. Public packages only.
// Path: happy.
func TestActiveEffectsScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	stage := func(d combat.EffectDuration, conds []content.ConditionKind, trig ...combat.EffectTrigger) combat.EffectTemplate {
		s, err := combat.NewEffectStage(combat.EffectStageParams{Duration: d, Conditions: conds, Triggers: trig})
		must(err)
		tpl, err := combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{s}})
		must(err)
		return tpl
	}
	rounds10, err := combat.NewEffectLastsRounds(10)
	must(err)
	creature := func(id string) combat.CreatureEffectSubject {
		s, err := combat.NewCreatureEffectSubject(combat.MustCombatantID(id))
		must(err)
		return s
	}
	var reg combat.ActiveEffects
	start := func(id, spell, caster, target string, tpl combat.EffectTemplate, concentration bool, conds ...content.ConditionKind) {
		t.Helper()
		origin, err := combat.NewSpellEffectOrigin(content.SpellID(spell), 2)
		must(err)
		cmd := combat.StartActiveEffectCommand{Effects: reg, ID: combat.MustActiveEffectID(id), Origin: origin, Caster: combat.MustCombatantID(caster),
			Subject: creature(target), Template: tpl, RequiresConcentration: concentration, Conditions: conds, Active: combat.MustCombatantID(caster)}
		if concentration {
			cmd.Binding, err = combat.NewConcentrationBinding(cmd.Caster, character.NewConcentrationSubjectSpell(content.SpellID(spell)))
			must(err)
		}
		res, err := combat.NewStartActiveEffect().Run(ctx, cmd)
		must(err)
		started, ok := res.(combat.StartActiveEffectStarted)
		if !ok {
			t.Fatalf("start %s = %#v", id, res)
		}
		reg = started.Effects()
	}
	blessing := stage(rounds10, nil)
	start("bless-1a", "bless", "cleric1", "ally1", blessing, true)
	start("bless-1b", "bless", "cleric1", "ally2", blessing, true)
	start("bless-2", "bless", "cleric2", "ally2", blessing, true)
	applying := func(target string) []string {
		var out []string
		for _, a := range (combat.EffectCombination{}).Applying(reg, creature(target)) {
			out = append(out, string(a.Effect().ID()))
		}
		return out
	}
	if got := applying("ally1"); !slices.Equal(got, []string{"bless-1a"}) {
		t.Errorf("Bless on ally1 = %v", got)
	}
	if got := applying("ally2"); !slices.Equal(got, []string{"bless-2"}) {
		t.Errorf("Bless on ally2 = %v, want only the latest casting to apply, one d4", got)
	}

	repeat, err := combat.NewRepeatSave(ability.Wisdom, 14)
	must(err)
	trigger, err := combat.NewEffectTrigger(combat.EffectTurnOfSubject, content.BoundaryEnd, repeat)
	must(err)
	hold := stage(rounds10, []content.ConditionKind{content.ConditionParalyzed}, trigger)
	start("hold", "hold_person", "wizard", "orc", hold, true, content.ConditionParalyzed)
	set, err := combat.NewConditionSet([]combat.Condition{combat.Paralyzed{}}, nil)
	must(err)

	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	saves := map[combat.CombatantID]map[ability.Ability]d20.SaveAttempt{"orc": {ability.Wisdom: attempt}}
	tick, err := combat.NewTurnEndReached("orc")
	must(err)
	advance, err := combat.NewAdvanceActiveEffects(dicetest.NewFixedRoller(9, 16))
	must(err)
	var applied combat.AdvanceActiveEffectsApplied
	for i, wantLeft := range []int{4, 3} {
		res, err := advance.Run(ctx, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: tick, Saves: saves})
		must(err)
		applied = res.(combat.AdvanceActiveEffectsApplied)
		reg = applied.Effects()
		if reg.Len() != wantLeft || (i == 0) != (len(applied.ConditionsToRemove()) == 0) {
			t.Fatalf("save %d: %d effects, removals %v, want %d effects", i+1, reg.Len(), applied.ConditionsToRemove(), wantLeft)
		}
	}
	plan := applied.ConditionsToRemove()[0]
	if plan.Kind != content.ConditionParalyzed || plan.Target != "orc" {
		t.Fatalf("plan = %+v", plan)
	}
	rm, err := combat.NewRemoveConditionFromSetCommand(set, plan.Target, plan.Kind)
	must(err)
	removed, err := combat.NewRemoveConditionFromSet().Run(ctx, rm)
	must(err)
	if cleared, ok := removed.(combat.ConditionCleared); !ok || len(cleared.Set().Effective()) != 0 {
		t.Errorf("removal = %#v, want the orc free", removed)
	}

	lost, err := combat.NewConcentrationLost("cleric1", character.NewConcentrationSubjectSpell("bless"))
	must(err)
	res, err := combat.NewEndActiveEffects().Run(ctx, combat.EndActiveEffectsCommand{Effects: reg, Cause: lost})
	must(err)
	ended, ok := res.(combat.EndActiveEffectsEnded)
	if !ok || ended.Effects().Len() != 1 || len(ended.Events()) != 2 {
		t.Errorf("Concentration lost = %#v, want the two Blesses of cleric1 gone and cleric2's left", res)
	}
}
