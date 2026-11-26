package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func endedOf(t *testing.T, res combat.EndActiveEffectsResult) combat.EndActiveEffectsEnded {
	t.Helper()
	e, ok := res.(combat.EndActiveEffectsEnded)
	if !ok {
		t.Fatalf("result = %#v, want EndActiveEffectsEnded", res)
	}
	return e
}

func holdSpec(id, target string) aeSpec {
	return aeSpec{id: id, spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2(target), template: aeHoldPerson(t0(), 14),
		conditions: []content.ConditionKind{content.ConditionParalyzed}}
}

// t0 is a testing.TB for fixtures built outside a test body; the fixtures never fail.
func t0() testing.TB { return &testing.T{} }

func endReasons(events []combat.ActiveEffectEvent) []combat.EffectEndReason {
	var out []combat.EffectEndReason
	for _, e := range events {
		out = append(out, e.(combat.ActiveEffectEnded).Reason())
	}
	return out
}

// Requirement: doc/specs/active-effects.md FR-041, FR-045 (SRD 5.2.1, "Concentration", p. 179) - ConcentrationLost ends every effect bound to that caster and subject in registry order with one ActiveEffectEnded each (reason concentrationLost) and plans the removal of their conditions; a cause that matches no effect returns NothingEnded, refused.
// Path: happy.
func TestConcentrationLostEndsEveryBoundEffect(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, holdSpec("h1", "orc"), holdSpec("h2", "goblin"), bless("other", "cleric", "orc", 0))
	cause := aeMust(combat.NewConcentrationLost("wiz", character.NewConcentrationSubjectSpell("hold_person")))
	out := endedOf(t, aeEnd(t, reg, cause))
	if !slices.Equal(aeIDs(out.Effects()), []string{"other"}) || !slices.Equal(endReasons(out.Events()), []combat.EffectEndReason{combat.EffectConcentrationLost, combat.EffectConcentrationLost}) {
		t.Errorf("registry %v reasons %v, want only the unbound effect left and two concentrationLost [FR-041]", aeIDs(out.Effects()), endReasons(out.Events()))
	}
	want := []combat.EffectConditionPlan{{Effect: "h1", Target: "orc", Kind: content.ConditionParalyzed}, {Effect: "h2", Target: "goblin", Kind: content.ConditionParalyzed}}
	if !slices.Equal(out.ConditionsToRemove(), want) {
		t.Errorf("removals = %v, want %v", out.ConditionsToRemove(), want)
	}
	combattest.CheckEvent(t, out.Events()[0], "active_effect_ended", "orc", "wiz")
	if out.Events()[0].EffectID() != "h1" || out.Events()[1].EffectID() != "h2" {
		t.Error("events are not in registry order")
	}
	for name, c := range map[string]combat.EndingCause{
		"another caster": aeMust(combat.NewConcentrationLost("cleric", character.NewConcentrationSubjectSpell("hold_person"))),
		"another spell":  aeMust(combat.NewConcentrationLost("wiz", character.NewConcentrationSubjectSpell("bane"))),
	} {
		none, ok := aeEnd(t, reg, c).(combat.EndActiveEffectsNothingEnded)
		if !ok || !none.Effects().Equal(reg) {
			t.Errorf("%s: = %#v, want NothingEnded with the registry unchanged", name, none)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-033 (SRD 5.2.1, "Sleep", p. 163) - damage ends an effect whose stage ends on damage when the total is at least 1 (absorbed damage counts: the total of DamageTaken); a total of 0 or less, another creature, or a stage without the flag does nothing.
// Path: unhappy (boundary).
func TestSubjectTookDamage(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("orc"), template: aeSleep(t, "orc", 13), conditions: []content.ConditionKind{content.ConditionIncapacitated}},
		aeSpec{id: "blessed", spell: "bless", caster: "c", subject: aeCreature2("orc"), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)})
	hit := func(subject string, total int) combat.EndActiveEffectsResult {
		return aeEnd(t, reg, aeMust(combat.NewSubjectTookDamage(combat.MustCombatantID(subject), total)))
	}
	for _, total := range []int{-1, 0} {
		if _, ok := hit("orc", total).(combat.EndActiveEffectsNothingEnded); !ok {
			t.Errorf("total %d ended an effect [FR-033]", total)
		}
	}
	if _, ok := hit("goblin", 5).(combat.EndActiveEffectsNothingEnded); !ok {
		t.Error("damage to another creature ended an effect")
	}
	for _, total := range []int{1, 7} {
		out := endedOf(t, hit("orc", total))
		if !slices.Equal(aeIDs(out.Effects()), []string{"blessed"}) || endReasons(out.Events())[0] != combat.EffectSubjectTookDamage ||
			len(out.ConditionsToRemove()) != 1 {
			t.Errorf("total %d: registry %v, want only the effect without the flag left", total, aeIDs(out.Effects()))
		}
	}
	if _, err := combat.NewSubjectTookDamage("", 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero creature err = %v", err)
	}
}

// Requirement: doc/specs/active-effects.md FR-044, FR-046, FR-038 - RemovedByEffect ends exactly the named effects (reason removedByEffect) and runs their onEnd; an id twice in one command ends once; an unknown id is rejected listing every unknown id with nothing ended.
// Path: unhappy (invalid input).
func TestRemovedByEffect(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "bf", spell: "delayed_blast_fireball", caster: "wiz", subject: aePlace(t, "bead"), template: aeDelayedBlast(t)}, bless("b", "c", "t", 0))
	out := endedOf(t, aeEnd(t, reg, combat.NewRemovedByEffect("bf", "bf")))
	if !slices.Equal(aeIDs(out.Effects()), []string{"b"}) || len(out.Events()) != 1 || endReasons(out.Events())[0] != combat.EffectRemovedByEffect {
		t.Errorf("registry %v events %v, want one end of bf [FR-046]", aeIDs(out.Effects()), aeKinds(out.Events()))
	}
	if plans := out.DamagePlans(); len(plans) != 1 || plans[0].Amount() != dice.MustParseExpression("12d6") {
		t.Errorf("plans = %v, want the stored damage to explode [FR-038]", plans)
	}
	rej, ok := aeEnd(t, reg, combat.NewRemovedByEffect("bf", "x", "y")).(combat.EndActiveEffectsRejected)
	if !ok || len(rej.Rejection().Problems()) != 2 || rej.Rejection().Problems()[0].Effect != "x" || rej.Rejection().Problems()[1].Effect != "y" {
		t.Errorf("rejection = %#v, want x and y listed", rej)
	}
	if _, ok := aeEnd(t, reg, combat.NewRemovedByEffect()).(combat.EndActiveEffectsNothingEnded); !ok {
		t.Error("removing nothing ended something")
	}
	ids := []combat.ActiveEffectID{"bf"}
	cause := combat.NewRemovedByEffect(ids...)
	ids[0] = "b"
	if got := endedOf(t, aeEnd(t, reg, cause)); !slices.Equal(aeIDs(got.Effects()), []string{"b"}) {
		t.Error("a caller's slice changed the cause")
	}
}

// Requirement: doc/specs/active-effects.md FR-057 - a result's events, plans and registry are copies and the input registry is not mutated by an end.
// Path: unhappy (immutability).
func TestEndResultsAreCopies(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, holdSpec("h1", "orc"))
	before := reg.Effects()
	out := endedOf(t, aeEnd(t, reg, combat.NewRemovedByEffect("h1")))
	out.Events()[0] = nil
	out.ConditionsToRemove()[0] = combat.EffectConditionPlan{}
	out.DamagePlans()
	out.ConditionsToApply()
	if out.Events()[0] == nil || out.ConditionsToRemove()[0].Target != "orc" {
		t.Error("a caller's slice changed the result")
	}
	if !slices.EqualFunc(before, reg.Effects(), combat.ActiveEffect.Equal) || reg.Len() != 1 {
		t.Error("an end changed the input registry")
	}
	combattest.CheckEventsCopy(t, func() []combat.Event { return eventsToBase(out.Events()) })
}

func eventsToBase(events []combat.ActiveEffectEvent) []combat.Event {
	out := make([]combat.Event, len(events))
	for i, e := range events {
		if e != nil {
			out[i] = e
		}
	}
	return out
}
