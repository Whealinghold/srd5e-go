package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func meShift(tb testing.TB, conds combat.ConditionSet, effects combat.OngoingEffects, rule combat.CarryOverRule) combat.ShiftShapeResult {
	tb.Helper()
	res, err := (combat.ShapeShifting{}).Shift(combat.ShiftShapeCommand{Creature: gid("hero"), State: combat.TrueForm(), Conditions: conds,
		Effects: effects, Effect: meSpell("polymorph"), Rule: rule})
	if err != nil {
		tb.Fatal(err)
	}
	return res
}

func meBaselineEffects() combat.OngoingEffects { return meEffects(meBless(), meCurse("c1")) }

// Requirement: per case
// Path: per case.
func TestShiftShapeCarryOver(t *testing.T) {
	t.Parallel()
	const req = meSpec + `table 4.8a; SRD 5.2.1, "Shape-Shifting", p. 187: `
	exhaustion3 := meConditions(combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(3)})
	cases := []struct {
		name, requirement, path string
		conds                   combat.ConditionSet
		effects                 combat.OngoingEffects
		rule                    combat.CarryOverRule
		wantConds               combat.ConditionSet
		wantEffects             []combat.OngoingEffect
		wantOmittedConds        []content.ConditionKind
		wantOmittedEffects      []combat.OngoingEffectID
	}{
		{"default carries everything over", req + "FR-047", "happy", meBaseline(), meBaselineEffects(), combat.CarryOverAll{},
			meBaseline(), []combat.OngoingEffect{meBless(), meCurse("c1")}, nil, nil},
		{"a nil rule is CarryOverAll", req + "FR-047, FR-052", "happy", meBaseline(), meBaselineEffects(), nil,
			meBaseline(), []combat.OngoingEffect{meBless(), meCurse("c1")}, nil, nil},
		{"curse excluded", req + "FR-048", "unhappy (rule interaction)", meBaseline(), meBaselineEffects(), meExcept(nil, "c1"),
			meBaseline(), []combat.OngoingEffect{meBless()}, nil, []combat.OngoingEffectID{"c1"}},
		{"condition excluded", req + "FR-048", "unhappy (rule interaction)", meBaseline(), meBaselineEffects(), meExcept(meKinds(content.ConditionPoisoned)),
			meConditions(combat.Frightened{Source: gid("scary")}), []combat.OngoingEffect{meBless(), meCurse("c1")}, meKinds(content.ConditionPoisoned), nil},
		{"nothing to carry", req + "FR-047", "unhappy (empty and minimal)", combat.ConditionSet{}, meEffects(), combat.CarryOverAll{},
			combat.ConditionSet{}, nil, nil, nil},
		{"nothing named", req + "FR-048", "unhappy (empty and minimal)", meBaseline(), meBaselineEffects(), meExcept(nil),
			meBaseline(), []combat.OngoingEffect{meBless(), meCurse("c1")}, nil, nil},
		{"exception names Exhaustion", req + "FR-048, Q-25", "unhappy (rule interaction)", exhaustion3, meBaselineEffects(), meExcept(meKinds(content.ConditionExhaustion)),
			combat.ConditionSet{}, []combat.OngoingEffect{meBless(), meCurse("c1")}, meKinds(content.ConditionExhaustion), nil},
		{"a condition with a lower kind than Exhaustion is removed whole", req + "FR-048", "unhappy (boundary)", meConditions(combat.Blinded{}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: gid("c")}}, combat.Poisoned{}), meBaselineEffects(), meExcept(meKinds(content.ConditionBlinded, content.ConditionCharmed)),
			meConditions(combat.Poisoned{}), []combat.OngoingEffect{meBless(), meCurse("c1")}, meKinds(content.ConditionBlinded, content.ConditionCharmed), nil},
		{"a repeated name is listed once", req + "FR-048", "unhappy (boundary)", meBaseline(), meBaselineEffects(),
			meExcept(meKinds(content.ConditionPoisoned, content.ConditionPoisoned), "c1", "c1"),
			meConditions(combat.Frightened{Source: gid("scary")}), []combat.OngoingEffect{meBless()}, meKinds(content.ConditionPoisoned), []combat.OngoingEffectID{"c1"}},
		{"Unconscious is removed before a named Prone, leaving neither", req + "FR-048; conditions.md D-7", "unhappy (rule interaction)",
			meConditions(combat.Prone{}, combat.Unconscious{}), meEffects(), meExcept(meKinds(content.ConditionProne, content.ConditionUnconscious)),
			combat.ConditionSet{}, nil, meKinds(content.ConditionUnconscious, content.ConditionProne), nil},
		{"removing only Unconscious leaves Prone", req + "FR-048; conditions.md D-7", "unhappy (rule interaction)",
			meConditions(combat.Unconscious{}), meEffects(), meExcept(meKinds(content.ConditionUnconscious)),
			meConditions(combat.Prone{}), nil, meKinds(content.ConditionUnconscious), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			applied, ok := meShift(t, c.conds, c.effects, c.rule).(combat.ShiftShapeApplied)
			if !ok {
				t.Fatalf("not applied [%s]", c.requirement)
			}
			if !applied.Conditions().Equal(c.wantConds) || !slices.Equal(applied.Effects().All(), c.wantEffects) {
				t.Errorf("conditions %v effects %v [%s; %s]", applied.Conditions().Effective(), applied.Effects().All(), c.requirement, c.path)
			}
			n := applied.NotCarriedOver()
			if !slices.Equal(n.Conditions(), c.wantOmittedConds) || !slices.Equal(n.Effects(), c.wantOmittedEffects) || n.Empty() != (len(c.wantOmittedConds)+len(c.wantOmittedEffects) == 0) {
				t.Errorf("NotCarriedOver = %v, %v want %v, %v", n.Conditions(), n.Effects(), c.wantOmittedConds, c.wantOmittedEffects)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-050 and table 4.15a - a successful shift is Applied, moves the state from TrueForm to ShiftedForm(effect) and emits one ShapeShifted carrying the creature and the effect source.
// Path: happy.
func TestShiftShapeAppliedStateAndEvent(t *testing.T) {
	t.Parallel()
	res := meShift(t, meBaseline(), meBaselineEffects(), nil).(combat.ShiftShapeApplied)
	if !res.State().IsShifted() || res.State().Effect() != meSpell("polymorph") {
		t.Errorf("state = %v", res.State())
	}
	evs := res.Events()
	if len(evs) != 1 {
		t.Fatalf("events = %v", evs)
	}
	ev := evs[0].(combat.ShapeShifted)
	combattest.CheckEvent(t, ev, "shape_shifted", gid("hero"))
	if ev.Effect() != meSpell("polymorph") {
		t.Errorf("event effect = %v", ev.Effect())
	}
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{res.Events()[0]} })
}

// Requirement: doc/specs/magical-effects.md FR-051 (Q-13) - a shift while already shifted succeeds and replaces the effect, keeping the creature shifted.
// Path: unhappy (state error).
func TestShiftShapeWhileShiftedReplacesTheEffect(t *testing.T) {
	t.Parallel()
	first := meMust(combat.ShiftedForm(meSpell("polymorph")))
	res, err := (combat.ShapeShifting{}).Shift(combat.ShiftShapeCommand{Creature: gid("hero"), State: first, Effect: meSpell("shapechange")})
	if err != nil {
		t.Fatal(err)
	}
	applied := res.(combat.ShiftShapeApplied)
	if !applied.State().IsShifted() || applied.State().Effect() != meSpell("shapechange") {
		t.Errorf("state = %v", applied.State())
	}
}

// Requirement: per case
// Path: per case.
func TestShiftShapeExceptionNotHeld(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		conds                   combat.ConditionSet
		rule                    combat.CarryOverRule
		wantConds               []content.ConditionKind
		wantEffects             []combat.OngoingEffectID
	}{
		{"names an absent condition", "FR-049, table 4.8a", "unhappy (state error)", meBaseline(), meExcept(meKinds(content.ConditionBlinded)), meKinds(content.ConditionBlinded), nil},
		{"names an absent effect", "FR-049, table 4.8a", "unhappy (state error)", meBaseline(), meExcept(nil, "nope"), nil, []combat.OngoingEffectID{"nope"}},
		{"names an implied-only kind", "FR-049, table 4.8a; conditions.md FR-032", "unhappy (rule interaction)", meConditions(combat.Paralyzed{}), meExcept(meKinds(content.ConditionIncapacitated)),
			meKinds(content.ConditionIncapacitated), nil},
		{"lists every one not held, held ones included in neither list", "FR-049", "unhappy (invalid input)", meBaseline(),
			meExcept(meKinds(content.ConditionPoisoned, content.ConditionBlinded, content.ConditionDeafened), "c1", "x", "y"),
			meKinds(content.ConditionBlinded, content.ConditionDeafened), []combat.OngoingEffectID{"x", "y"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := meShift(t, c.conds, meBaselineEffects(), c.rule)
			nh, ok := res.(combat.ShiftShapeExceptionNotHeld)
			if !ok {
				t.Fatalf("got %T, want ShiftShapeExceptionNotHeld [%s]", res, c.requirement)
			}
			if !slices.Equal(nh.Conditions(), c.wantConds) || !slices.Equal(nh.Effects(), c.wantEffects) {
				t.Errorf("not held = %v, %v want %v, %v [%s]", nh.Conditions(), nh.Effects(), c.wantConds, c.wantEffects, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-047 to FR-049 (immutability, testing.md section 4) - a shift changes neither its input set nor its input list, and the rule and result views are copies.
// Path: unhappy (immutability).
func TestShiftShapeDoesNotChangeItsInputs(t *testing.T) {
	t.Parallel()
	conds, effects := meBaseline(), meBaselineEffects()
	rule := meExcept(meKinds(content.ConditionPoisoned), "c1")
	res := meShift(t, conds, effects, rule).(combat.ShiftShapeApplied)
	if !conds.Equal(meBaseline()) || !effects.Equal(meBaselineEffects()) {
		t.Error("an input changed")
	}
	rule.Conditions()[0] = content.ConditionBlinded
	rule.Effects()[0] = "zzz"
	if rule.Conditions()[0] != content.ConditionPoisoned || rule.Effects()[0] != "c1" {
		t.Error("a rule view is not a copy")
	}
	n := res.NotCarriedOver()
	n.Conditions()[0] = content.ConditionBlinded
	n.Effects()[0] = "zzz"
	if res.NotCarriedOver().Conditions()[0] != content.ConditionPoisoned || res.NotCarriedOver().Effects()[0] != "c1" {
		t.Error("a NotCarriedOver view is not a copy")
	}
}

// Requirement: doc/specs/magical-effects.md FR-144 and section 5 - a shift with a zero creature or the zero effect source, and a rule that names an invalid condition kind or a blank effect id, is an ErrInvalidInput error with no result.
// Path: unhappy (invalid input).
func TestShiftShapeRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for name, cmd := range map[string]combat.ShiftShapeCommand{
		"zero creature": {Effect: meSpell("polymorph")},
		"zero effect":   {Creature: gid("hero")},
	} {
		if res, err := (combat.ShapeShifting{}).Shift(cmd); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
			t.Errorf("%s: got %v, %v", name, res, err)
		}
	}
	if _, err := combat.NewCarryOverExcept(meKinds(content.ConditionKind(0)), nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid kind: %v", err)
	}
	if _, err := combat.NewCarryOverExcept(nil, []combat.OngoingEffectID{" "}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank effect id: %v", err)
	}
	if _, err := combat.ShiftedForm(combat.EffectSource{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ShiftedForm(zero): %v", err)
	}
	if combat.TrueForm().IsShifted() || !combat.TrueForm().Effect().IsZero() {
		t.Error("the zero state is TrueForm")
	}
}

func meShifted(tb testing.TB) combat.ShapeShiftState {
	tb.Helper()
	return meMust(combat.ShiftedForm(meSpell("polymorph")))
}

// Requirement: doc/specs/magical-effects.md FR-052 and table 4.15a - RevertToTrueForm takes the state, conditions, effects and a CarryOverRule (default CarryOverAll), returns Applied with TrueForm, carries the ongoing effects back by the semantics of FR-047 and FR-048 and emits one RevertedToTrueForm.
// Path: happy.
func TestRevertToTrueForm(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		rule        combat.CarryOverRule
		wantConds   combat.ConditionSet
		wantEffects []combat.OngoingEffect
	}{
		{"default", nil, meBaseline(), []combat.OngoingEffect{meBless(), meCurse("c1")}},
		{"carry all", combat.CarryOverAll{}, meBaseline(), []combat.OngoingEffect{meBless(), meCurse("c1")}},
		{"curse and Poisoned do not carry back", meExcept(meKinds(content.ConditionPoisoned), "c1"), meConditions(combat.Frightened{Source: gid("scary")}), []combat.OngoingEffect{meBless()}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := (combat.ShapeShifting{}).Revert(combat.RevertToTrueFormCommand{Creature: gid("hero"), State: meShifted(t),
				Conditions: meBaseline(), Effects: meBaselineEffects(), Rule: c.rule})
			if err != nil {
				t.Fatal(err)
			}
			applied := res.(combat.RevertToTrueFormApplied)
			if applied.State().IsShifted() || !applied.Conditions().Equal(c.wantConds) || !slices.Equal(applied.Effects().All(), c.wantEffects) {
				t.Errorf("state %v conditions %v effects %v", applied.State(), applied.Conditions().Effective(), applied.Effects().All())
			}
			evs := applied.Events()
			if len(evs) != 1 {
				t.Fatalf("events = %v", evs)
			}
			combattest.CheckEvent(t, evs[0], "reverted_to_true_form", gid("hero"))
			if evs[0].(combat.RevertedToTrueForm).Cause() != combat.RevertCauseEnded {
				t.Errorf("cause = %v", evs[0].(combat.RevertedToTrueForm).Cause())
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-053 - RevertToTrueForm on a creature already in TrueForm is Applied with the same state, conditions and effects and no events, whatever the rule.
// Path: unhappy (empty and minimal).
func TestRevertToTrueFormWhenAlreadyInTrueForm(t *testing.T) {
	t.Parallel()
	res, err := (combat.ShapeShifting{}).Revert(combat.RevertToTrueFormCommand{Creature: gid("hero"), State: combat.TrueForm(),
		Conditions: meBaseline(), Effects: meBaselineEffects(), Rule: meExcept(meKinds(content.ConditionPoisoned), "c1")})
	if err != nil {
		t.Fatal(err)
	}
	applied := res.(combat.RevertToTrueFormApplied)
	if applied.State().IsShifted() || !applied.Conditions().Equal(meBaseline()) || !applied.Effects().Equal(meBaselineEffects()) || len(applied.Events()) != 0 || !applied.NotCarriedOver().Empty() {
		t.Error("a creature in true form must be unchanged with no events")
	}
}

// Requirement: doc/specs/magical-effects.md FR-052, FR-049 - a revert whose rule names what the creature does not hold is RevertToTrueFormExceptionNotHeld, unchanged and with no event; a zero creature is an ErrInvalidInput error.
// Path: unhappy (state error).
func TestRevertToTrueFormRefusalsAndInvalidInput(t *testing.T) {
	t.Parallel()
	res, err := (combat.ShapeShifting{}).Revert(combat.RevertToTrueFormCommand{Creature: gid("hero"), State: meShifted(t),
		Conditions: meBaseline(), Effects: meBaselineEffects(), Rule: meExcept(meKinds(content.ConditionBlinded), "nope")})
	nh, ok := res.(combat.RevertToTrueFormExceptionNotHeld)
	if err != nil || !ok || !slices.Equal(nh.Conditions(), meKinds(content.ConditionBlinded)) || !slices.Equal(nh.Effects(), []combat.OngoingEffectID{"nope"}) {
		t.Errorf("got %#v, %v", res, err)
	}
	if res, err := (combat.ShapeShifting{}).Revert(combat.RevertToTrueFormCommand{State: meShifted(t)}); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero creature: %v, %v", res, err)
	}
}

// Requirement: doc/specs/magical-effects.md FR-054 - RevertOnDeath for a Dead creature in ShiftedForm is Applied with TrueForm and one RevertedToTrueForm with cause died; for a Dead creature in TrueForm it is Applied with no event (SRD 5.2.1, "Shape-Shifting", p. 187).
// Path: unhappy (state error).
func TestRevertOnDeathApplied(t *testing.T) {
	t.Parallel()
	dead := meMust(combat.NewDead(combat.MassiveDamage))
	shifted := (combat.ShapeShifting{}).RevertOnDeath(gid("hero"), meShifted(t), dead).(combat.RevertOnDeathApplied)
	evs := shifted.Events()
	if shifted.State().IsShifted() || len(evs) != 1 {
		t.Fatalf("shifted: state %v events %v", shifted.State(), evs)
	}
	combattest.CheckEvent(t, evs[0], "reverted_to_true_form", gid("hero"))
	if evs[0].(combat.RevertedToTrueForm).Cause() != combat.RevertCauseDied {
		t.Error("cause must be died")
	}
	trueForm := (combat.ShapeShifting{}).RevertOnDeath(gid("hero"), combat.TrueForm(), dead).(combat.RevertOnDeathApplied)
	if trueForm.State().IsShifted() || len(trueForm.Events()) != 0 {
		t.Error("a Dead creature in true form gets no event")
	}
}

// Requirement: doc/specs/magical-effects.md FR-055 - RevertOnDeath for a creature that is not Dead (Alive, Dying, Stable, a nil or zero Dead) is RevertOnDeathNotDead, whatever its form, with no event.
// Path: unhappy (state error).
func TestRevertOnDeathNotDead(t *testing.T) {
	t.Parallel()
	for name, life := range map[string]combat.LifeState{"alive": combat.Alive{}, "dying": combat.Dying{}, "stable": combat.Stable{}, "nil": nil, "zero dead": combat.Dead{}} {
		for _, st := range []combat.ShapeShiftState{combat.TrueForm(), meShifted(t)} {
			if _, ok := (combat.ShapeShifting{}).RevertOnDeath(gid("hero"), st, life).(combat.RevertOnDeathNotDead); !ok {
				t.Errorf("%s (shifted %v): want NotDead", name, st.IsShifted())
			}
		}
	}
	dead := meMust(combat.NewDead(combat.MonsterDeath))
	if _, ok := (combat.ShapeShifting{}).RevertOnDeath("", meShifted(t), dead).(combat.RevertOnDeathNotDead); !ok {
		t.Error("a zero creature is not a Dead creature")
	}
}

// Requirement: doc/specs/magical-effects.md FR-052 - RevertCause and its String values.
// Path: happy.
func TestRevertCauseString(t *testing.T) {
	t.Parallel()
	for c, want := range map[combat.RevertCause]string{combat.RevertCauseEnded: "ended", combat.RevertCauseDied: "died", 0: "unknown"} {
		if c.String() != want {
			t.Errorf("%d.String() = %q", int(c), c.String())
		}
	}
}
