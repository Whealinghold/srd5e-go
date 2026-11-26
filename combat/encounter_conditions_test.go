package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

func exh(n int) combat.Condition {
	return combat.ExhaustionCondition{Level: combat.MustExhaustionLevel(n)}
}

// exhLevel is the Exhaustion level a combatant holds.
func exhLevel(c combat.Combatant) int {
	for _, x := range c.Conditions().Explicit() {
		if e, ok := x.(combat.ExhaustionCondition); ok {
			return e.Level.Value()
		}
	}
	return 0
}

// Requirement: per case
// Path: per case.
func TestExhaustionAndImpliedConditionCases(t *testing.T) {
	t.Parallel()
	type row struct {
		name, requirement, path string
		before                  combat.Combatant
		run                     func(e combat.Encounter) (encTr, error)
		kinds                   []string
		level                   int
		rejection               combat.EncounterRejection
	}
	a := func(hp combat.HitPointStatus, conds ...combat.Condition) combat.Combatant {
		return encC(t, "a", combat.SideA, hp, conds...)
	}
	pc := pcStatus(t, 10, 20, 0)
	apply := func(c combat.Condition) func(combat.Encounter) (encTr, error) {
		return func(e combat.Encounter) (encTr, error) { return e.ApplyCondition(cid("a"), c) }
	}
	remove := func(k content.ConditionKind) func(combat.Encounter) (encTr, error) {
		return func(e combat.Encounter) (encTr, error) { return e.RemoveCondition(cid("a"), k) }
	}
	reduce := func(n int) func(combat.Encounter) (encTr, error) {
		return func(e combat.Encounter) (encTr, error) { return e.ReduceExhaustion(cid("a"), n) }
	}
	rows := []row{
		{"first level", "FR-036; p. 181", "happy", a(pc), apply(exh(1)), []string{"condition_applied"}, 1, nil},
		{"raise from 1", "FR-036; p. 181", "unhappy (rule interaction)", a(pc, exh(1)), apply(exh(1)), []string{"condition_applied"}, 2, nil},
		{"level 5 to 6", "FR-037; p. 181", "unhappy (boundary)", a(pc, exh(5)), apply(exh(1)), []string{"condition_applied", "died"}, 6, nil},
		{"remove lowers one level", "FR-041; p. 181", "unhappy (boundary)", a(pc, exh(3)), remove(content.ConditionExhaustion), []string{"condition_level_reduced"}, 2, nil},
		{"remove at level 1", "FR-041; p. 181", "unhappy (boundary)", a(pc, exh(1)), remove(content.ConditionExhaustion), []string{"condition_removed"}, 0, nil},
		{"reduce from 2 by 1", "FR-063; p. 181", "unhappy (boundary)", a(pc, exh(2)), reduce(1), []string{"condition_level_reduced"}, 1, nil},
		{"reduce from 1 by 1", "FR-063; p. 181", "unhappy (boundary)", a(pc, exh(1)), reduce(1), []string{"condition_removed"}, 0, nil},
		{"reduce by more than the level", "FR-063; p. 181", "unhappy (boundary)", a(pc, exh(2)), reduce(3), []string{"condition_removed"}, 0, nil},
		{"reduce with none", "FR-063; p. 181", "unhappy (state)", a(pc), reduce(1), nil, 0, combat.DoesNotHaveConditionRejection{ID: cid("a"), Kind: content.ConditionExhaustion}},
		{"remove with none", "FR-040; p. 181", "unhappy (state)", a(pc), remove(content.ConditionExhaustion), nil, 0, combat.DoesNotHaveConditionRejection{ID: cid("a"), Kind: content.ConditionExhaustion}},
		{"apply at 6", "FR-064; p. 181", "unhappy (boundary)", a(deadStatus(t, combat.Exhaustion), exh(6)), apply(exh(1)), nil, 6, combat.CombatantIsDeadRejection{ID: cid("a")}},
		{"reduce at 6", "FR-064; p. 181", "unhappy (boundary)", a(deadStatus(t, combat.Exhaustion), exh(6)), reduce(1), nil, 6, combat.CombatantIsDeadRejection{ID: cid("a")}},
		{"remove Prone while Unconscious", "FR-065; p. 191", "unhappy (state)", a(pc, combat.Unconscious{}), remove(content.ConditionProne), nil, 0,
			combat.ConditionImpliedByRejection{ID: cid("a"), Kind: content.ConditionProne}},
		{"already has a condition", "FR-035; p. 179", "unhappy (rule interaction)", a(pc, combat.Poisoned{}), apply(combat.Poisoned{}), nil, 0, combat.AlreadyHasConditionRejection{ID: cid("a"), Kind: content.ConditionPoisoned}},
		{"does not have the condition", "FR-040; p. 179", "unhappy (state)", a(pc), remove(content.ConditionBlinded), nil, 0, combat.DoesNotHaveConditionRejection{ID: cid("a"), Kind: content.ConditionBlinded}},
		{"apply a condition", "FR-034; p. 179", "happy", a(pc), apply(combat.Blinded{}), []string{"condition_applied"}, 0, nil},
		{"remove a condition", "FR-039; p. 179", "happy", a(pc, combat.Blinded{}), remove(content.ConditionBlinded), []string{"condition_removed"}, 0, nil},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			e := encWith(t, r.before)
			tr, err := r.run(e)
			if r.rejection != nil {
				got := encNo(t, e, tr, err)
				if g, w := got.String(), r.rejection.String(); g != w {
					t.Errorf("rejection = %#v, want %#v [%s]", got, r.rejection, r.requirement)
				}
				if ib, ok := got.(combat.ConditionImpliedByRejection); ok && !slices.Equal(ib.ImpliedBy(), []content.ConditionKind{content.ConditionUnconscious}) {
					t.Errorf("implied by %v, want [unconscious] [FR-065]", ib.ImpliedBy())
				}
				return
			}
			after, events := encOK(t, tr, err)
			if !slices.Equal(encKinds(events), r.kinds) {
				t.Errorf("events = %v, want %v [%s]", encKinds(events), r.kinds, r.requirement)
			}
			if got := exhLevel(encMember(t, after, "a")); got != r.level {
				t.Errorf("Exhaustion level = %d, want %d [%s]", got, r.level, r.requirement)
			}
			encOthersEqual(t, e, after, "a")
			checkEnvelopes(t, events)
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-037, FR-068, FR-069 - Exhaustion level 6 wraps ConditionApplied (level 6) then Died (exhaustion), records the death on the combatant, and damage and healing for it are then refused.
// Path: unhappy (boundary).
func TestExhaustionDeathIsWrappedAndRecorded(t *testing.T) {
	t.Parallel()
	e := encWith(t, encC(t, "a", combat.SideA, pcStatus(t, 10, 20, 0), exh(5)))
	tr, err := e.ApplyCondition(cid("a"), exh(1))
	after, events := encOK(t, tr, err)
	applied := events[0].(combat.ConditionEventOccurred).Event().(combat.ConditionApplied)
	if applied.Condition().(combat.ExhaustionCondition).Level.Value() != 6 {
		t.Errorf("ConditionApplied carries %v, want level 6", applied.Condition())
	}
	if d := events[1].(combat.HitPointEventOccurred).Event().(combat.Died); d.Cause() != combat.Exhaustion {
		t.Errorf("Died cause = %v", d.Cause())
	}
	if cause, ok := encMember(t, after, "a").Death(); !ok || cause != combat.Exhaustion {
		t.Errorf("Death = %v, %v [FR-068]", cause, ok)
	}
	d1, err := after.ApplyDamage(cid("a"), encSlash(1), nil, combat.OtherSource, combat.NormalHit)
	h1, err2 := after.ApplyHealing(cid("a"), 1)
	if encNo(t, after, d1, err) == nil || encNo(t, after, h1, err2) == nil {
		t.Errorf("damage and healing for a creature dead of Exhaustion must be refused [FR-069]")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-038, table 4.9.1 - an Immune combatant is refused ImmuneToCondition, and Immune is checked before already-has (a condition suppressed by Immunity).
// Path: unhappy (rule interaction).
func TestImmuneToConditionBeforeAlreadyHas(t *testing.T) {
	t.Parallel()
	set, err := combat.NewConditionSet([]combat.Condition{combat.Poisoned{}}, []content.ConditionKind{content.ConditionPoisoned})
	if err != nil {
		t.Fatal(err)
	}
	immune := encWith(t, mustCombatant(t, "a", combat.SideA, pcStatus(t, 10, 20, 0), set))
	tr, err := immune.ApplyCondition(cid("a"), combat.Poisoned{})
	if r := encNo(t, immune, tr, err); r != (combat.ImmuneToConditionRejection{ID: cid("a"), Kind: content.ConditionPoisoned}) || r.String() != "immune_to_condition" {
		t.Errorf("rejection = %#v, want ImmuneToCondition [FR-038, FR-078]", r)
	}
	if got := (combat.AlreadyHasConditionRejection{}).String(); got != "already_has_condition" {
		t.Errorf("name = %q", got)
	}
	// Dead and immune: dead wins.
	deadImmune := encWith(t, mustCombatant(t, "a", combat.SideA, deadStatus(t, combat.MonsterDeath), set))
	tr, err = deadImmune.ApplyCondition(cid("a"), combat.Poisoned{})
	if r := encNo(t, deadImmune, tr, err); r != (combat.CombatantIsDeadRejection{ID: cid("a")}) {
		t.Errorf("rejection = %#v, want CombatantIsDead first", r)
	}
}

func mustCombatant(tb testing.TB, id string, side combat.Side, hp combat.HitPointStatus, set combat.ConditionSet) combat.Combatant {
	tb.Helper()
	c, err := combat.NewCombatant(combat.CombatantRef{ID: cid(id), Kind: combat.MonsterCombatant{ID: "goblin"}}, side, hp, set)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// Requirement: doc/specs/encounter-aggregate.md FR-066, table 4.6.1 - removing Unconscious ends the implied Incapacitated and leaves the combatant Prone as an explicit condition, with no ConditionApplied for it.
// Path: unhappy (rule interaction).
func TestRemovingUnconsciousLeavesProne(t *testing.T) {
	t.Parallel()
	e := encWith(t, encC(t, "a", combat.SideA, pcStatus(t, 10, 20, 0), combat.Unconscious{}))
	if c := encMember(t, e, "a"); !encHas(c, content.ConditionIncapacitated) || !encHas(c, content.ConditionProne) {
		t.Fatalf("Unconscious must imply Incapacitated and Prone [FR-066]")
	}
	tr, err := e.RemoveCondition(cid("a"), content.ConditionUnconscious)
	after, events := encOK(t, tr, err)
	c := encMember(t, after, "a")
	if encHas(c, content.ConditionIncapacitated) || !encExplicit(c, content.ConditionProne) || !slices.Equal(encKinds(events), []string{"condition_removed"}) {
		t.Errorf("after removing Unconscious: Incapacitated %v Prone %v events %v [FR-066]", encHas(c, content.ConditionIncapacitated), encExplicit(c, content.ConditionProne), encKinds(events))
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-034, FR-039, FR-063 and testing.md section 4 - an invalid condition, an invalid kind and levels below 1 are errors before any rejection.
// Path: unhappy (invalid input).
func TestConditionCommandsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	e := endedEncounter(t, encABC(t))
	for name, f := range map[string]func() (encTr, error){
		"nil condition":   func() (encTr, error) { return e.ApplyCondition(cid("a"), nil) },
		"zero exhaustion": func() (encTr, error) { return e.ApplyCondition(cid("a"), combat.ExhaustionCondition{}) },
		"invalid kind":    func() (encTr, error) { return e.RemoveCondition(cid("a"), 0) },
		"levels 0":        func() (encTr, error) { return e.ReduceExhaustion(cid("a"), 0) },
		"levels -1":       func() (encTr, error) { return e.ReduceExhaustion(cid("a"), -1) },
	} {
		if res, err := f(); err == nil || res != nil {
			t.Errorf("%s: want an ErrInvalidInput error before the ended rejection, got %v, %v", name, res, err)
		}
	}
}
