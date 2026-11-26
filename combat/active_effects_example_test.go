package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A held creature repeats its saving throw at the end of its turn; a success ends the effect and
// the host removes the condition it planned.
func ExampleAdvanceActiveEffects() {
	ctx := context.Background()
	subject, _ := combat.NewCreatureEffectSubject("orc")
	origin, _ := combat.NewSpellEffectOrigin("hold_person", 2)
	repeat, _ := combat.NewRepeatSave(ability.Wisdom, 14)
	trigger, _ := combat.NewEffectTrigger(combat.EffectTurnOfSubject, content.BoundaryEnd, repeat)
	rounds, _ := combat.NewEffectLastsRounds(10)
	stage, _ := combat.NewEffectStage(combat.EffectStageParams{Duration: rounds, Triggers: []combat.EffectTrigger{trigger},
		Conditions: []content.ConditionKind{content.ConditionParalyzed}})
	template, _ := combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{stage}})

	started, _ := combat.NewStartActiveEffect().Run(ctx, combat.StartActiveEffectCommand{ID: "hold", Origin: origin, Caster: "wizard",
		Subject: subject, Template: template, Conditions: []content.ConditionKind{content.ConditionParalyzed}, Active: "wizard"})
	registry := started.(combat.StartActiveEffectStarted).Effects()

	advance, _ := combat.NewAdvanceActiveEffects(dicetest.NewFixedRoller(15))
	tick, _ := combat.NewTurnEndReached("orc")
	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
	res, _ := advance.Run(ctx, combat.AdvanceActiveEffectsCommand{Effects: registry, Tick: tick,
		Saves: map[combat.CombatantID]map[ability.Ability]d20.SaveAttempt{"orc": {ability.Wisdom: attempt}}})
	applied := res.(combat.AdvanceActiveEffectsApplied)
	fmt.Println(applied.Effects().Len(), applied.ConditionsToRemove()[0].Kind)
	// Output: 0 Paralyzed
}
