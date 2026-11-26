package scenarios_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
)

// grpFWaiverKinds is the order ComponentsWaived.Kinds lists the kinds in.
var grpFWaiverKinds = struct{ v, s, m character.ComponentKind }{character.ComponentVerbal, character.ComponentSomatic, character.ComponentMaterial}

// Requirement: doc/specs/monster-spellcasting.md FR-017 (Table CS-1) - a printed components statement is trimmed, lower-cased, stripped of whitespace runs and one trailing period, and must read "requiring no " plus "spell components" or a list of verbal, somatic and material joined by commas, "or" and "and"; each listed kind is waived, an empty statement waives nothing, and anything else (an unknown kind, a trailing clause, a missing prefix, a repeated kind) is unrecognized and carries the text as given.
// Path: per case.
func TestGrpFComponentWaiverGrammar(t *testing.T) {
	t.Parallel()
	v, s, m := grpFWaiverKinds.v, grpFWaiverKinds.s, grpFWaiverKinds.m
	cases := []struct {
		name      string
		statement string
		waived    []character.ComponentKind // nil with unrecognized false: waives nothing
		bad       bool
	}{
		{name: "empty statement waives nothing", statement: ""},
		{name: "blank statement waives nothing", statement: "   \t "},
		{name: "a single kind", statement: "requiring no Verbal components", waived: []character.ComponentKind{v}},
		{name: "material only", statement: "requiring no Material components", waived: []character.ComponentKind{m}},
		{name: "two kinds with or", statement: "requiring no Verbal or Somatic components", waived: []character.ComponentKind{v, s}},
		{name: "two kinds with and", statement: "requiring no Somatic and Material components", waived: []character.ComponentKind{s, m}},
		{name: "a list with a comma and or", statement: "requiring no Verbal, Somatic, or Material components", waived: []character.ComponentKind{v, s, m}},
		{name: "a list with a comma and and", statement: "requiring no Verbal, Somatic, and Material components", waived: []character.ComponentKind{v, s, m}},
		{name: "a plain comma list", statement: "requiring no Somatic, Material components", waived: []character.ComponentKind{s, m}},
		{name: "all spell components", statement: "requiring no spell components", waived: []character.ComponentKind{v, s, m}},
		{name: "spacing, case and one trailing period", statement: "  Requiring  no Material components.  ", waived: []character.ComponentKind{m}},
		{name: "listed out of order still reports in V, S, M order", statement: "requiring no Material or Verbal components", waived: []character.ComponentKind{v, m}},
		{name: "an unknown kind", statement: "requiring no Arcane components", bad: true},
		{name: "a trailing clause", statement: "requiring no spell components and using Wisdom", bad: true},
		{name: "a repeated kind", statement: "requiring no Material or Material components", bad: true},
		{name: "a missing requiring prefix", statement: "no Verbal components", bad: true},
		{name: "a missing components suffix", statement: "requiring no Verbal", bad: true},
		{name: "two trailing periods", statement: "requiring no Verbal components..", bad: true},
		{name: "spell components inside a list", statement: "requiring no spell components or Verbal components", bad: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch w := combat.ParseComponentWaiver(c.statement).(type) {
			case combat.ComponentsWaived:
				if c.bad {
					t.Fatalf("%q = waived %v, want unrecognized", c.statement, w.Kinds())
				}
				if !slices.Equal(w.Kinds(), c.waived) {
					t.Errorf("%q waives %v, want %v", c.statement, w.Kinds(), c.waived)
				}
				for _, k := range []character.ComponentKind{v, s, m} {
					if w.Waives(k) != slices.Contains(c.waived, k) {
						t.Errorf("%q Waives(%v) = %v, want %v", c.statement, k, w.Waives(k), slices.Contains(c.waived, k))
					}
				}
			case combat.ComponentStatementUnrecognized:
				if !c.bad {
					t.Fatalf("%q = unrecognized, want waived %v", c.statement, c.waived)
				}
				if w.Statement() != c.statement {
					t.Errorf("Statement() = %q, want the text as given %q", w.Statement(), c.statement)
				}
			default:
				t.Fatalf("%q = %#v, not one of the two readings", c.statement, w)
			}
		})
	}
}

// grpFPartRig is the Spellcasting monster of the other monster scenarios and a Misty Step part that prints its own components statement; silenced and bound say whether the monster can speak and make gestures.
func grpFPartRig(t *testing.T, statement string, canSpeak, canGesture bool, actions combat.ConditionActionAvailability) (grpCRig, combat.MonsterPartCast) {
	t.Helper()
	rig := grpCNewRig(t)
	rig.mc = grpCMust(combat.NewMonsterCastContext(canSpeak, canGesture, actions))
	own := grpCMust(combat.NewPartSpellcasting(combat.PartSpellcastingFields{
		Ability: ability.Intelligence, SaveDC: 15, HasSaveDC: true, Components: statement}))
	return rig, grpCMust(combat.NewMonsterPartCast(grpCMistyStepAt, 0, own))
}

// Requirement: doc/specs/monster-spellcasting.md FR-017, FR-018, FR-019, FR-031 - a Fireball cast by a part that prints a components statement: a Verbal or Somatic component the statement waives is not checked (a silenced or bound monster still casts), one it does not waive refuses with its own reason, a Material component never refuses, an unrecognized statement refuses and is never guessed, and the refusals come in the order of the spec (no actions, then the statement, then Verbal, then Somatic).
// Path: per case.
func TestGrpFComponentStatementsDecideWhatASilencedMonsterCasts(t *testing.T) {
	t.Parallel()
	avail, none := combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone
	cases := []struct {
		name       string
		statement  string
		speak      bool
		gesture    bool
		actions    combat.ConditionActionAvailability
		wantCasts  bool
		wantReason combat.MonsterCastRefusal
	}{
		{"no statement and free to speak and move", "", true, true, avail, true, 0},
		{"no statement, silenced", "", false, true, avail, false, combat.MonsterCastRefusalVerbalMissing},
		{"no statement, no hands", "", true, false, avail, false, combat.MonsterCastRefusalSomaticMissing},
		{"no statement, silenced and bound: Verbal is reported first", "", false, false, avail, false, combat.MonsterCastRefusalVerbalMissing},
		{"Verbal waived, silenced", "requiring no Verbal components", false, true, avail, true, 0},
		{"Verbal waived, silenced and bound: Somatic is still checked", "requiring no Verbal components", false, false, avail, false, combat.MonsterCastRefusalSomaticMissing},
		{"Somatic waived, no hands", "requiring no Somatic components", true, false, avail, true, 0},
		{"Somatic waived, silenced: Verbal is still checked", "requiring no Somatic components", false, true, avail, false, combat.MonsterCastRefusalVerbalMissing},
		{"Material waived does not help a silenced monster", "requiring no Material components", false, true, avail, false, combat.MonsterCastRefusalVerbalMissing},
		{"Verbal and Somatic waived by a list", "requiring no Verbal and Somatic components", false, false, avail, true, 0},
		{"all waived by the comma and or list", "requiring no Verbal, Somatic, or Material components", false, false, avail, true, 0},
		{"no spell components, silenced and bound", "requiring no spell components", false, false, avail, true, 0},
		{"spacing and a period do not change the reading", "  Requiring  no spell components.  ", false, false, avail, true, 0},
		{"an unknown kind refuses even when the monster is free", "requiring no Arcane components", true, true, avail, false, combat.MonsterCastRefusalComponentsUnrecognized},
		{"a trailing clause refuses", "requiring no spell components and using Wisdom", true, true, avail, false, combat.MonsterCastRefusalComponentsUnrecognized},
		{"a repeated kind refuses", "requiring no Material or Material components", true, true, avail, false, combat.MonsterCastRefusalComponentsUnrecognized},
		{"the statement is checked before Verbal", "requiring no Arcane components", false, true, avail, false, combat.MonsterCastRefusalComponentsUnrecognized},
		{"no actions come before the statement", "requiring no Arcane components", true, true, none, false, combat.MonsterCastRefusalCannotTakeActions},
		{"no actions refuse a monster that waives everything", "requiring no spell components", true, true, none, false, combat.MonsterCastRefusalCannotTakeActions},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig, part := grpFPartRig(t, c.statement, c.speak, c.gesture, c.actions)
			out, err := rig.cast(part, "fireball")
			if c.wantCasts {
				done := rig.performed(out, err)
				if done.Transition.LimitedUse().AvailabilityOf(grpCMistyStepAt) != combat.UseAvailable { // 3/Day, one spent
					t.Errorf("the part after a cast = %v, want two of three uses left", done.Transition.LimitedUse().AvailabilityOf(grpCMistyStepAt))
				}
				return
			}
			if got := rig.refusal(out, err); got != c.wantReason {
				t.Errorf("refusal = %v, want %v", got, c.wantReason)
			}
			if _, held := rig.state.Concentration(); held || rig.uses.AvailabilityOf(grpCMistyStepAt) != combat.UseAvailable {
				t.Errorf("a refused cast changed the states the caller holds")
			}
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-019 - a Material component never refuses a monster cast, with or without a statement and including one with a gold cost, and the cast needs no free hand beyond the Somatic rule.
// Path: unhappy (boundary).
func TestGrpFMaterialComponentNeverRefusesAMonsterCast(t *testing.T) {
	t.Parallel()
	for _, statement := range []string{"", "requiring no Material components", "requiring no spell components"} {
		// Revivify needs a diamond worth 300 gp; the monster speaks and gestures, so only the Material component is in question.
		rig, part := grpFPartRig(t, statement, true, true, combat.ConditionActionAvailabilityAvailable)
		rig.performed(rig.cast(part, "revivify", grpCID("hero")))
	}
}
