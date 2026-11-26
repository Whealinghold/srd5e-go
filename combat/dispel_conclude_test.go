package combat_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// dspApplied concludes the plan of registry with checks and returns the applied result,
// failing the test when the conclusion is invalid.
func dspApplied(t *testing.T, r combat.OngoingSpellRegistry, plan combat.DispelMagicPlan, checks dspChecks) combat.DispelMagicApplied {
	t.Helper()
	res := combat.DispelMagic{}.Conclude(r, plan, dspByEffect(t, checks))
	applied, ok := res.(combat.DispelMagicApplied)
	if !ok {
		t.Fatalf("Conclude = %T, want DispelMagicApplied", res)
	}
	return applied
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-021 - a total equal to the DC succeeds and ends the spell, one below fails; a natural 20 with a failed total fails and a natural 1 with a passing total succeeds (SRD 5.2.1, "Compare the Total to a Target Number", p. 6).
// Path: per case.
func TestDispelMagicConcludeCheckTotalsAgainstDC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		natural, flat           int
		wantEnded               bool
	}{
		{"total equal to the DC succeeds", "FR-021", "unhappy (boundary)", 14, 0, true},
		{"total one below the DC fails", "FR-021", "unhappy (boundary)", 13, 0, false},
		{"natural 20 with a failed total fails", "FR-021", "unhappy (rule interaction)", 20, -10, false},
		{"natural 1 with a passing total succeeds", "FR-021", "unhappy (rule interaction)", 1, 13, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			goblin := dspTarget(t, "goblin")
			r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 4})))
			plan := dspPlan(t, r, goblin, 3)
			got := len(dspApplied(t, r, plan, dspChecks{"x": dspCheck(t, c.natural, c.flat, 14)}).Ended()) == 1
			if got != c.wantEnded {
				t.Errorf("ended = %v, want %v [%s]", got, c.wantEnded, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-022 - an invalid set of checks is refused with its first problem in plan order, and nothing changes (invalid input, rule interaction).
// Path: per case.
func TestDispelMagicConcludeReportsTheFirstProblem(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	fifteen := dspCheck(t, 15, 0, 14)
	cases := []struct {
		name, requirement, path string
		checks                  dspChecks
		want                    combat.DispelMagicProblem
	}{
		{"a needed check is missing", "FR-022", "unhappy (invalid input)", dspChecks{}, combat.DispelMissingCheck},
		{"a check for a spell that ends without one", "FR-022", "unhappy (invalid input)", dspChecks{"x": fifteen, "cantrip": fifteen}, combat.DispelUnexpectedCheck},
		{"a check at a DC other than the step's", "FR-022", "unhappy (invalid input)", dspChecks{"x": dspCheck(t, 15, 0, 15)}, combat.DispelCheckTargetMismatch},
		{"a check that is a saving throw", "FR-022", "unhappy (invalid input)", dspChecks{"x": dspSavingThrow(t, 15, 14)}, combat.DispelNotAnAbilityCheck},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := dspRegistry(t,
				dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 4})),
				dspHold(goblin, dspSpell(t, dspSpec{effect: "cantrip", spell: "light", level: 0})),
			)
			plan := dspPlan(t, r, goblin, 3)
			res := combat.DispelMagic{}.Conclude(r, plan, dspByEffect(t, c.checks))
			invalid, ok := res.(combat.DispelMagicInvalid)
			if !ok || invalid.Problem() != c.want {
				t.Fatalf("Conclude = %#v, want invalid %v [%s]", res, c.want, c.requirement)
			}
		})
	}
}

// dspSavingThrow resolves a saving throw with a natural roll against dc.
func dspSavingThrow(t *testing.T, natural int, dc d20.DifficultyClass) d20.CheckResult {
	t.Helper()
	roll, err := d20.NewRoll(modifier.Normal, []int{natural})
	if err != nil {
		t.Fatal(err)
	}
	res, err := d20.ResolveCheck(d20.SavingThrow, roll, d20.NewRollModifiers(modifier.New(0)), dc)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-022 - a missing check reported first is reported ahead of a later wrong kind (first problem in plan order).
// Path: unhappy (rule interaction).
func TestDispelMagicConcludeFirstProblemIsInPlanOrder(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "first", spell: "bless", level: 4})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "second", spell: "haste", level: 4})),
	)
	plan := dspPlan(t, r, goblin, 3)
	res := combat.DispelMagic{}.Conclude(r, plan, dspByEffect(t, dspChecks{"second": dspSavingThrow(t, 15, 14)}))
	invalid, ok := res.(combat.DispelMagicInvalid)
	if !ok || invalid.Problem() != combat.DispelMissingCheck {
		t.Fatalf("Conclude = %#v, want missing_check for the first spell [FR-022]", res)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-059 - a plan concluded against a registry that no longer holds its spells is refused (invalid input).
// Path: unhappy (invalid input).
func TestDispelMagicConcludeRefusesAPlanFromAnotherRegistry(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	made := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1})))
	plan := dspPlan(t, made, goblin, 3)
	res := combat.DispelMagic{}.Conclude(dspRegistry(t), plan, nil)
	invalid, ok := res.(combat.DispelMagicInvalid)
	if !ok || invalid.Problem() != combat.DispelPlanMismatch {
		t.Fatalf("Conclude = %#v, want plan_mismatch [FR-059]", res)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-023 - with nothing ended the result is applied with an equal registry and no events; blocked spells survive after failed checks (SRD 5.2.1, "Dispel Magic", p. 124).
// Path: happy.
func TestDispelMagicConcludeNothingEndedKeepsRegistry(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "blocked", spell: "forcecage", level: 2, restriction: combat.DispelNeverDispellable{}})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "failed", spell: "bless", level: 4})),
	)
	plan := dspPlan(t, r, goblin, 3)
	applied := dspApplied(t, r, plan, dspChecks{"failed": dspCheck(t, 2, 0, 14)})
	if !applied.Registry().Equal(r) {
		t.Error("registry changed although nothing ended [FR-023]")
	}
	if len(applied.Events()) != 0 || len(applied.Ended()) != 0 {
		t.Error("events or ended spells reported although nothing ended [FR-023]")
	}
	survived := applied.Survived()
	if len(survived) != 2 || survived[0].Effect().ID() != dspEffectID(t, "failed") || survived[1].Effect().ID() != dspEffectID(t, "blocked") {
		t.Errorf("survived = %v, want the failed check then the blocked spell [FR-023]", survived)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-024 and FR-009 - each ended spell gives one event in plan order, and a parent ends with cause allPartsEnded directly after its last part (SRD 5.2.1, "Guards and Wards", p. 138).
// Path: happy.
func TestDispelMagicConcludeEndsParentAfterItsLastPart(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	part := func(effect string) combat.HeldSpell {
		return dspHold(goblin, dspSpell(t, dspSpec{effect: effect, spell: "guards_and_wards", level: 2, partOf: "gw"}))
	}
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "gw", spell: "guards_and_wards", level: 6, restriction: combat.DispelNeverDispellable{}})),
		part("gw1"),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "bless", spell: "bless", level: 1})),
		part("gw2"),
	)
	plan := dspPlan(t, r, goblin, 3)
	applied := dspApplied(t, r, plan, nil)
	kinds := make([]string, 0)
	for _, e := range applied.Events() {
		kinds = append(kinds, effectCauseName(t, e))
	}
	want := []string{"gw1:dispelled", "bless:dispelled", "gw2:dispelled", "gw:all_parts_ended"}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("events = %v, want %v [FR-024]", kinds, want)
	}
	if got := applied.Registry().SpellsOn(goblin); len(got) != 0 {
		t.Errorf("registry after = %d spells, want none: the parent ended with its parts [FR-009]", len(got))
	}
}

// effectCauseName names an ended spell event as "effect:cause" for comparison.
func effectCauseName(t *testing.T, e combat.SpellEndingEvent) string {
	t.Helper()
	ended, ok := e.(combat.OngoingSpellEnded)
	if !ok {
		t.Fatalf("event %T is not OngoingSpellEnded", e)
	}
	return ended.EffectID().String() + ":" + ended.Cause().String()
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-009 - a parent is not ended while a part of it remains (the parent's later part is kept).
// Path: unhappy (rule interaction).
func TestDispelMagicConcludeKeepsParentWhilePartsRemain(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "gw", spell: "guards_and_wards", level: 6, restriction: combat.DispelNeverDispellable{}})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "gw1", spell: "guards_and_wards", level: 2, partOf: "gw"})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "gw2", spell: "guards_and_wards", level: 5, partOf: "gw"})),
	)
	plan := dspPlan(t, r, goblin, 3)
	applied := dspApplied(t, r, plan, dspChecks{"gw2": dspCheck(t, 2, 0, 15)})
	if got := len(applied.Events()); got != 1 {
		t.Fatalf("events = %d, want only gw1 dispelled [FR-009]", got)
	}
	if got := len(applied.Registry().SpellsOn(goblin)); got != 2 {
		t.Errorf("registry after = %d spells, want the parent and gw2 [FR-009]", got)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-025 - an OngoingSpellEnded carries the target, effect, spell, level, cause, conditions to remove and the Concentration flag, and removes no condition itself (SRD 5.2.1, "Concentration", p. 179).
// Path: happy.
func TestDispelMagicConcludeEventCarriesTheEndedSpell(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	cond := content.ConditionKind(1)
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1, caster: "cleric", concentration: true, conditions: []content.ConditionKind{cond}})))
	applied := dspApplied(t, r, dspPlan(t, r, goblin, 3), nil)
	ended, ok := applied.Events()[0].(combat.OngoingSpellEnded)
	if !ok {
		t.Fatalf("event %T, want OngoingSpellEnded [FR-025]", applied.Events()[0])
	}
	if ended.Kind() != "ongoing_spell_ended" || ended.Cause() != combat.SpellEndDispelled || ended.SpellID() != "bless" || ended.CastAtLevel().Int() != 1 {
		t.Errorf("event = %v %v %v %v, want dispelled bless at level 1 [FR-025]", ended.Kind(), ended.Cause(), ended.SpellID(), ended.CastAtLevel())
	}
	if got := ended.ConditionsToRemove(); len(got) != 1 || got[0] != cond {
		t.Errorf("ConditionsToRemove = %v, want %v [FR-025]", got, cond)
	}
	if !ended.EndsCasterConcentration() || len(ended.Combatants()) != 2 {
		t.Errorf("EndsCasterConcentration or caster wrong [FR-025, FR-026]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-026 - the caster's Concentration ends only for a Concentration spell with a known caster (SRD 5.2.1, "Concentration", p. 179).
// Path: per case.
func TestDispelMagicConcludeEndsConcentrationOnlyWithAKnownCaster(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spec                    dspSpec
		want                    bool
	}{
		{"concentration spell with a caster", "FR-026", "happy", dspSpec{spell: "haste", concentration: true, caster: "wizard"}, true},
		{"non-concentration spell", "FR-026", "unhappy (boundary)", dspSpec{spell: "bless", caster: "cleric"}, false},
		{"concentration spell with an unknown caster", "FR-026", "unhappy (boundary)", dspSpec{spell: "haste", concentration: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			goblin := dspTarget(t, "goblin")
			spec := c.spec
			spec.effect, spec.level = "x", 1
			r := dspRegistry(t, dspHold(goblin, dspSpell(t, spec)))
			applied := dspApplied(t, r, dspPlan(t, r, goblin, 3), nil)
			ended := applied.Events()[0].(combat.OngoingSpellEnded)
			if ended.EndsCasterConcentration() != c.want {
				t.Errorf("EndsCasterConcentration = %v, want %v [%s]", ended.EndsCasterConcentration(), c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-027 - Conclude changes neither input and returns equal results for equal inputs (determinism, immutability).
// Path: unhappy (immutability).
func TestDispelMagicConcludeIsPure(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1})))
	plan := dspPlan(t, r, goblin, 3)
	first := dspApplied(t, r, plan, nil)
	second := dspApplied(t, r, plan, nil)
	if !first.Registry().Equal(second.Registry()) || len(first.Events()) != len(second.Events()) {
		t.Error("equal inputs gave different conclusions [FR-027]")
	}
	if !r.Equal(dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1})))) {
		t.Error("Conclude changed the registry it was given [FR-027]")
	}
	events := first.Events()
	events[0] = nil
	if first.Events()[0] == nil {
		t.Error("changing a returned event list changed the conclusion [FR-027]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-023 - a spell that ended leaves the result registry and the ended list, while the others keep their order (SRD 5.2.1, "Dispel Magic", p. 124).
// Path: happy.
func TestDispelMagicConcludeRemovesOnlyTheEndedSpells(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "ends", spell: "bless", level: 1})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "stays", spell: "forcecage", level: 2, restriction: combat.DispelNeverDispellable{}})),
	)
	applied := dspApplied(t, r, dspPlan(t, r, goblin, 3), nil)
	left := applied.Registry().SpellsOn(goblin)
	if len(left) != 1 || left[0].Effect().ID() != dspEffectID(t, "stays") {
		t.Errorf("registry after = %d spells, want only the blocked one [FR-023]", len(left))
	}
	if got := applied.Ended(); len(got) != 1 || got[0].Effect().ID() != dspEffectID(t, "ends") {
		t.Errorf("ended = %v, want the one dispelled spell [FR-023]", got)
	}
}
