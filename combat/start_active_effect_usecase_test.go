package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func runStart(t *testing.T, cmd combat.StartActiveEffectCommand, opts ...combat.Option) (combat.StartActiveEffectResult, error) {
	t.Helper()
	return combat.NewStartActiveEffect(opts...).Run(context.Background(), cmd)
}

func holdPersonSpec() aeSpec {
	return aeSpec{id: "hp", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"),
		template: aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{
			aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeMust(combat.NewEffectLastsRounds(10)), Conditions: []content.ConditionKind{content.ConditionParalyzed}}))}})),
		conditions: []content.ConditionKind{content.ConditionParalyzed}, active: "wiz"}
}

// Requirement: doc/specs/active-effects.md FR-013 (SRD 5.2.1, "Duration", p. 106) - starting adds one effect at the end of the registry at stage 0 with its clock set from the stage duration, anchored at the active combatant, and emits one ActiveEffectStarted; it applies no condition, rolls no die and changes nothing in its input.
// Path: happy.
func TestStartAddsAtTheEnd(t *testing.T) {
	t.Parallel()
	first := aeSeq(t, bless("a", "c1", "t", 0))
	cmd := holdPersonSpec().command(t, first)
	cmd.Conditions = []content.ConditionKind{content.ConditionParalyzed}
	res, err := runStart(t, cmd)
	started, ok := res.(combat.StartActiveEffectStarted)
	if err != nil || !ok {
		t.Fatalf("start = %#v, %v", res, err)
	}
	if !slices.Equal(aeIDs(started.Effects()), []string{"a", "hp"}) || first.Len() != 1 {
		t.Errorf("registry = %v, input has %d, want [a hp] and 1 [FR-013]", aeIDs(started.Effects()), first.Len())
	}
	e := aeEffect(t, started.Effects(), "hp")
	if e.Stage() != 0 || e.RemainingRounds() != 10 || !e.AnchoredTo("wiz") || !slices.Equal(e.Conditions(), []content.ConditionKind{content.ConditionParalyzed}) {
		t.Errorf("effect = %#v", e)
	}
	events := started.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v, want one", aeKinds(events))
	}
	combattest.CheckEvent(t, events[0], "active_effect_started", "orc", "wiz")
	if events[0].EffectID() != "hp" || len(started.ConditionsToApply())+len(started.ConditionsToRemove())+len(started.DamagePlans()) != 0 {
		t.Error("a start plans conditions or damage")
	}
	combattest.CheckEventsCopy(t, func() []combat.Event {
		out := []combat.Event{}
		for _, e := range started.Events() {
			out = append(out, e)
		}
		return out
	})
}

// Requirement: doc/specs/active-effects.md FR-013, FR-054 - outside combat the effect has no anchor; an effect on a place has the caster only in its events and holds no condition; an effect on its own caster lists the creature once.
// Path: unhappy (state error).
func TestStartOutsideCombatOnAPlaceAndOnTheCaster(t *testing.T) {
	t.Parallel()
	spec := holdPersonSpec()
	spec.active = ""
	if e := aeEffect(t, aeSeq(t, spec), "hp"); e.Anchored() {
		t.Error("an effect started outside combat is anchored [FR-013]")
	}
	place := aeSpec{id: "sp", spell: "flaming_sphere", caster: "wiz", subject: aePlace(t, "sphere"), template: aeFlamingSphere(t, 13), conditions: []content.ConditionKind{content.ConditionPoisoned}}
	res, _ := runStart(t, place.command(t, combat.ActiveEffects{}))
	ev := res.(combat.StartActiveEffectStarted).Events()[0]
	combattest.CheckEvent(t, ev, "active_effect_started", "wiz")
	if got := aeEffect(t, res.(combat.StartActiveEffectStarted).Effects(), "sp").Conditions(); len(got) != 0 {
		t.Errorf("an effect on a place holds %v", got)
	}
	self := aeSpec{id: "me", spell: "bless", caster: "wiz", subject: aeCreature2("wiz"), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)}
	res, _ = runStart(t, self.command(t, combat.ActiveEffects{}))
	combattest.CheckEvent(t, res.(combat.StartActiveEffectStarted).Events()[0], "active_effect_started", "wiz")
}

// Requirement: doc/specs/active-effects.md FR-013 - a command that is not well formed (a blank id, no caster, no template, an unusable origin or subject, a condition that is not one or is Exhaustion) is an error matching ErrInvalidInput and changes nothing.
// Path: unhappy (invalid input).
func TestStartMalformedCommands(t *testing.T) {
	t.Parallel()
	good := holdPersonSpec().command(t, combat.ActiveEffects{})
	cases := map[string]func(c *combat.StartActiveEffectCommand){
		"blank id":          func(c *combat.StartActiveEffectCommand) { c.ID = " " },
		"no caster":         func(c *combat.StartActiveEffectCommand) { c.Caster = "" },
		"no template":       func(c *combat.StartActiveEffectCommand) { c.Template = combat.EffectTemplate{} },
		"nil origin":        func(c *combat.StartActiveEffectCommand) { c.Origin = nil },
		"zero spell origin": func(c *combat.StartActiveEffectCommand) { c.Origin = combat.SpellEffectOrigin{} },
		"zero other origin": func(c *combat.StartActiveEffectCommand) { c.Origin = combat.OtherEffectOrigin{} },
		"nil subject":       func(c *combat.StartActiveEffectCommand) { c.Subject = nil },
		"zero creature":     func(c *combat.StartActiveEffectCommand) { c.Subject = combat.CreatureEffectSubject{} },
		"zero place":        func(c *combat.StartActiveEffectCommand) { c.Subject = combat.PlaceEffectSubject{} },
		"zero condition":    func(c *combat.StartActiveEffectCommand) { c.Conditions = []content.ConditionKind{0} },
		"exhaustion": func(c *combat.StartActiveEffectCommand) {
			c.Conditions = []content.ConditionKind{content.ConditionExhaustion}
		},
	}
	for name, edit := range cases {
		cmd := good
		edit(&cmd)
		if res, err := runStart(t, cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: = %#v, %v, want an ErrInvalidInput error", name, res, err)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-055, FR-057 - a cancelled context returns ctx.Err() before any signal; a result returns copies; the zero use case works.
// Path: unhappy (cancellation, immutability).
func TestStartCancelledContextAndZeroUseCase(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rec, opts := recording()
	if res, err := combat.NewStartActiveEffect(opts...).Run(ctx, holdPersonSpec().command(t, combat.ActiveEffects{})); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v, %v", res, err)
	}
	if len(rec.Tracer.Spans()) != 0 || len(rec.Meter.Measurements()) != 0 {
		t.Error("a cancelled call recorded a signal")
	}
	var zero combat.StartActiveEffect
	res, err := zero.Run(context.Background(), holdPersonSpec().command(t, combat.ActiveEffects{}))
	if _, ok := res.(combat.StartActiveEffectStarted); err != nil || !ok {
		t.Errorf("zero use case = %v, %v", res, err)
	}
	started := res.(combat.StartActiveEffectStarted)
	started.Events()[0] = nil
	started.ConditionsToApply()
	if started.Events()[0] == nil {
		t.Error("a caller's slice changed the result")
	}
}
