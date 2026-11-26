package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// The pilot templates of active-effects.md Q11 are fixtures: built here from the exported
// constructors, never from SRD text.

// aeMust returns v, or panics with err: the fixtures are built from constants that are valid.
func aeMust[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func aeEffectID(s string) combat.ActiveEffectID { return combat.MustActiveEffectID(s) }

func aeRounds(tb testing.TB, n int) combat.EffectDuration {
	tb.Helper()
	return aeMust(combat.NewEffectLastsRounds(n))
}

func aeUntil(tb testing.TB, owner string, b content.TurnBoundary) combat.EffectDuration {
	tb.Helper()
	return aeMust(combat.NewEffectLastsUntilTurnBoundary(combat.MustCombatantID(owner), b))
}

func aeStage(tb testing.TB, p combat.EffectStageParams) combat.EffectStage {
	tb.Helper()
	return aeMust(combat.NewEffectStage(p))
}

func aeTemplate(tb testing.TB, p combat.EffectTemplateParams) combat.EffectTemplate {
	tb.Helper()
	return aeMust(combat.NewEffectTemplate(p))
}

func aeRepeatSave(tb testing.TB, a ability.Ability, dc int) combat.RepeatSave {
	tb.Helper()
	return aeMust(combat.NewRepeatSave(a, d20.DifficultyClass(dc)))
}

func aeTrigger(tb testing.TB, whose combat.EffectTurnOwner, b content.TurnBoundary, actions ...combat.EffectAction) combat.EffectTrigger {
	tb.Helper()
	return aeMust(combat.NewEffectTrigger(whose, b, actions...))
}

func aeDeal(tb testing.TB, expr string, t content.DamageType, a combat.EffectDamageAudience) combat.DealDamage {
	tb.Helper()
	return aeMust(combat.NewDealDamage(dice.MustParseExpression(expr), t, a))
}

func aeSaveOf(tb testing.TB, a ability.Ability, dc int, s combat.EffectSaveSuccess) combat.EffectDamageSave {
	tb.Helper()
	return aeMust(combat.NewEffectDamageSave(a, d20.DifficultyClass(dc), s))
}

// aeHoldPerson: 10 rounds, Paralyzed, the target repeats a Wisdom save at the end of each of its turns.
func aeHoldPerson(tb testing.TB, dc int) combat.EffectTemplate {
	tb.Helper()
	save := aeTrigger(tb, combat.EffectTurnOfSubject, content.BoundaryEnd, aeRepeatSave(tb, ability.Wisdom, dc))
	return aeTemplate(tb, combat.EffectTemplateParams{Stages: []combat.EffectStage{aeStage(tb, combat.EffectStageParams{
		Duration: aeRounds(tb, 10), Conditions: []content.ConditionKind{content.ConditionParalyzed}, Triggers: []combat.EffectTrigger{save}})}})
}

// aeAcidArrow: until the end of the target's next turn, then 2d4 acid damage to it.
func aeAcidArrow(tb testing.TB, target string) combat.EffectTemplate {
	tb.Helper()
	end := aeDeal(tb, "2d4", content.DamageAcid, combat.EffectAudienceSubject)
	return aeTemplate(tb, combat.EffectTemplateParams{Stages: []combat.EffectStage{aeStage(tb, combat.EffectStageParams{
		Duration: aeUntil(tb, target, content.BoundaryEnd), OnEnd: []combat.EffectAction{end}})}})
}

// aeSleep: one minute overall; Incapacitated until the end of the target's next turn, then a save
// (the gate), then Unconscious for the rest of the spell; damage ends either stage.
func aeSleep(tb testing.TB, target string, dc int) combat.EffectTemplate {
	tb.Helper()
	first := aeStage(tb, combat.EffectStageParams{Duration: aeUntil(tb, target, content.BoundaryEnd),
		Conditions: []content.ConditionKind{content.ConditionIncapacitated}, EndsOnDamage: true, Gate: aeRepeatSave(tb, ability.Wisdom, dc)})
	second := aeStage(tb, combat.EffectStageParams{Duration: combat.EffectLastsRemainder{},
		Conditions: []content.ConditionKind{content.ConditionUnconscious}, EndsOnDamage: true})
	return aeTemplate(tb, combat.EffectTemplateParams{Stages: []combat.EffectStage{first, second}, Overall: aeRounds(tb, 10)})
}

// aeFlamingSphere: 10 rounds; a creature in the area at the end of its turn takes 2d6 fire damage
// (a Dexterity save for half).
func aeFlamingSphere(tb testing.TB, dc int) combat.EffectTemplate {
	tb.Helper()
	hit := aeDeal(tb, "2d6", content.DamageFire, combat.EffectAudienceTurnTaker).WithSave(aeSaveOf(tb, ability.Dexterity, dc, combat.SaveSuccessHalfDamage))
	trig := aeTrigger(tb, combat.EffectTurnOfCreatureInArea, content.BoundaryEnd, hit)
	return aeTemplate(tb, combat.EffectTemplateParams{Stages: []combat.EffectStage{aeStage(tb, combat.EffectStageParams{
		Duration: aeRounds(tb, 10), Triggers: []combat.EffectTrigger{trig}})}})
}

// aeDelayedBlast: 12d6 stored, 1d6 more at the end of each of the caster's turns, then it
// explodes on the area when the effect ends.
func aeDelayedBlast(tb testing.TB) combat.EffectTemplate {
	tb.Helper()
	grow := aeMust(combat.NewGrowStoredDamage(dice.MustParseExpression("1d6")))
	trig := aeTrigger(tb, combat.EffectTurnOfCaster, content.BoundaryEnd, grow)
	boom := aeMust(combat.NewDealStoredDamage(content.DamageFire, combat.EffectAudienceAreaMembers))
	return aeTemplate(tb, combat.EffectTemplateParams{Stages: []combat.EffectStage{aeStage(tb, combat.EffectStageParams{
		Duration: aeRounds(tb, 10), Triggers: []combat.EffectTrigger{trig}, OnEnd: []combat.EffectAction{boom}})},
		StoredDamage: dice.MustParseExpression("12d6")})
}

// aeMark is a Hunter's Mark or Hex: it combines per caster and survives its subject's death.
func aeMark(tb testing.TB, rounds int) combat.EffectTemplate {
	tb.Helper()
	return aeTemplate(tb, combat.EffectTemplateParams{Stages: []combat.EffectStage{aeStage(tb, combat.EffectStageParams{Duration: aeRounds(tb, rounds)})},
		CombiningSubject: combat.CombiningOnCaster, SurvivesSubjectDeath: true})
}

func aeCreature(tb testing.TB, id string) combat.CreatureEffectSubject {
	tb.Helper()
	return aeMust(combat.NewCreatureEffectSubject(combat.MustCombatantID(id)))
}

func aePlace(tb testing.TB, id string) combat.PlaceEffectSubject {
	tb.Helper()
	return aeMust(combat.NewPlaceEffectSubject(combat.EffectPlaceID(id)))
}

func aeSpellOrigin(tb testing.TB, spell string, level int) combat.SpellEffectOrigin {
	tb.Helper()
	return aeMust(combat.NewSpellEffectOrigin(content.SpellID(spell), level))
}

func aeBinding(tb testing.TB, caster, spell string) combat.ConcentrationBinding {
	tb.Helper()
	return aeMust(combat.NewConcentrationBinding(combat.MustCombatantID(caster), character.NewConcentrationSubjectSpell(content.SpellID(spell))))
}

// aeSpec is one effect to start; the zero fields mean the common case (a spell cast at level 1).
type aeSpec struct {
	id, spell, caster string
	subject           combat.EffectSubject
	template          combat.EffectTemplate
	concentration     bool
	conditions        []content.ConditionKind
	active            string
}

func (s aeSpec) command(tb testing.TB, reg combat.ActiveEffects) combat.StartActiveEffectCommand {
	tb.Helper()
	cmd := combat.StartActiveEffectCommand{Effects: reg, ID: aeEffectID(s.id), Origin: aeSpellOrigin(tb, s.spell, 1),
		Caster: combat.MustCombatantID(s.caster), Subject: s.subject, Template: s.template, RequiresConcentration: s.concentration,
		Conditions: s.conditions, Active: combat.CombatantID(s.active)}
	if s.concentration {
		cmd.Binding = aeBinding(tb, s.caster, s.spell)
	}
	return cmd
}

// aeStart starts one effect and returns the registry; the start must be accepted.
func aeStart(tb testing.TB, reg combat.ActiveEffects, s aeSpec) combat.ActiveEffects {
	tb.Helper()
	res, err := combat.NewStartActiveEffect().Run(context.Background(), s.command(tb, reg))
	if err != nil {
		tb.Fatal(err)
	}
	started, ok := res.(combat.StartActiveEffectStarted)
	if !ok {
		tb.Fatalf("start %s = %#v, want StartActiveEffectStarted", s.id, res)
	}
	return started.Effects()
}

// aeSeq starts each spec in order on an empty registry.
func aeSeq(tb testing.TB, specs ...aeSpec) combat.ActiveEffects {
	tb.Helper()
	var reg combat.ActiveEffects
	for _, s := range specs {
		reg = aeStart(tb, reg, s)
	}
	return reg
}

func aeEffect(tb testing.TB, reg combat.ActiveEffects, id string) combat.ActiveEffect {
	tb.Helper()
	e, ok := reg.Find(aeEffectID(id))
	if !ok {
		tb.Fatalf("registry has no effect %q", id)
	}
	return e
}

func aeIDs(reg combat.ActiveEffects) []string {
	out := []string{}
	for _, e := range reg.Effects() {
		out = append(out, string(e.ID()))
	}
	return out
}

// aeAdvance runs AdvanceActiveEffects with a fixed roller and the active combatant.
func aeAdvance(tb testing.TB, reg combat.ActiveEffects, tick combat.EffectClockTick, active string, faces ...int) combat.AdvanceActiveEffectsApplied {
	tb.Helper()
	res := aeAdvanceCmd(tb, combat.AdvanceActiveEffectsCommand{Effects: reg, Tick: tick, Active: combat.CombatantID(active)}, faces...)
	applied, ok := res.(combat.AdvanceActiveEffectsApplied)
	if !ok {
		tb.Fatalf("advance = %#v, want AdvanceActiveEffectsApplied", res)
	}
	return applied
}

func aeAdvanceCmd(tb testing.TB, cmd combat.AdvanceActiveEffectsCommand, faces ...int) combat.AdvanceActiveEffectsResult {
	tb.Helper()
	uc, err := combat.NewAdvanceActiveEffects(dicetestFixed(faces...))
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatal(err)
	}
	return res
}

func aeStartTick(tb testing.TB, id string) combat.EffectClockTick {
	tb.Helper()
	return aeMust(combat.NewTurnStartReached(combat.MustCombatantID(id)))
}

func aeEndTick(tb testing.TB, id string) combat.EffectClockTick {
	tb.Helper()
	return aeMust(combat.NewTurnEndReached(combat.MustCombatantID(id)))
}

func aeElapsed(tb testing.TB, n int) combat.EffectClockTick {
	tb.Helper()
	return aeMust(combat.NewTimeElapsed(n))
}

// aeRoll is an attempt that rolls a d20 with no modifier.
func aeRoll() d20.SaveAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}
}

// aeSaves is the attempts of one creature at one ability.
func aeSaves(c string, a ability.Ability, attempt d20.SaveAttempt) map[combat.CombatantID]map[ability.Ability]d20.SaveAttempt {
	return map[combat.CombatantID]map[ability.Ability]d20.SaveAttempt{combat.MustCombatantID(c): {a: attempt}}
}

func aeKinds(events []combat.ActiveEffectEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}
