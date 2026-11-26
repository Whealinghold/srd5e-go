package combat_test

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func condApplied(c combat.Condition) combat.ConditionApplied {
	e, err := combat.NewConditionApplied(subject(), c)
	if err != nil {
		panic(err)
	}
	return e
}

func condRemoved(k content.ConditionKind) combat.ConditionRemoved {
	e, err := combat.NewConditionRemoved(subject(), k)
	if err != nil {
		panic(err)
	}
	return e
}

func condLowered(n int) combat.ConditionLevelReduced {
	e, err := combat.NewConditionLevelReduced(subject(), combat.MustExhaustionLevel(n))
	if err != nil {
		panic(err)
	}
	return e
}

// Requirement: per case
// Path: per case.
func TestApply(t *testing.T) {
	t.Parallel()
	petrifiedPoisoned := condSet(t, condConds(combat.Petrified{}, combat.Poisoned{}))
	cases := []struct {
		name, requirement, path string
		set                     combat.ConditionSet
		apply                   combat.Condition
		want                    string
		wantEffective           []content.ConditionKind
		wantEvents              []combat.ConditionEvent
	}{
		{"Blinded to an empty set", `FR-016, FR-017; "Condition", p. 179`, "happy", combat.ConditionSet{}, combat.Blinded{}, "ConditionAdded", condKinds(cB), []combat.ConditionEvent{condApplied(combat.Blinded{})}},
		{"Charmed to an empty set", `FR-016, FR-017`, "happy", combat.ConditionSet{}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, "ConditionAdded", condKinds(cC), []combat.ConditionEvent{condApplied(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}})}},
		{"Frightened to an empty set", `FR-016, FR-017`, "happy", combat.ConditionSet{}, combat.Frightened{Source: idA()}, "ConditionAdded", condKinds(cF), []combat.ConditionEvent{condApplied(combat.Frightened{Source: idA()})}},
		{"Grappled to an empty set", `FR-016, FR-017`, "happy", combat.ConditionSet{}, combat.Grappled{Grappler: idA()}, "ConditionAdded", condKinds(cG), []combat.ConditionEvent{condApplied(combat.Grappled{Grappler: idA()})}},
		{"Unconscious brings Incapacitated and Prone but one event", `FR-016, FR-026`, "happy", combat.ConditionSet{}, combat.Unconscious{}, "ConditionAdded", condKinds(cI, cPr, cU), []combat.ConditionEvent{condApplied(combat.Unconscious{})}},
		{"a kind already held", `FR-018; "Condition", p. 179`, "unhappy (rule interaction)", condSet(t, condConds(combat.Blinded{})), combat.Blinded{}, "ConditionAlreadyHeld", condKinds(cB), nil},
		{"a second source of fear keeps the first", `FR-019 (D-9)`, "unhappy (rule interaction)", condSet(t, condConds(combat.Frightened{Source: idA()})), combat.Frightened{Source: idB()}, "ConditionAlreadyHeld", condKinds(cF), nil},
		{"a second charmer keeps the first", `FR-019 (D-9)`, "unhappy (rule interaction)", condSet(t, condConds(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}})), combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idB()}}, "ConditionAlreadyHeld", condKinds(cC), nil},
		{"a second grappler keeps the first", `FR-019 (D-9)`, "unhappy (rule interaction)", condSet(t, condConds(combat.Grappled{Grappler: idA()})), combat.Grappled{Grappler: idB()}, "ConditionAlreadyHeld", condKinds(cG), nil},
		{"an immunity to the kind", `FR-021; "Immunity", pp. 17, 183`, "unhappy (rule interaction)", condSet(t, nil, cB), combat.Blinded{}, "ConditionImmune", nil, nil},
		{"an immunity to another kind", `FR-022; "Immunity", p. 17`, "unhappy (boundary)", condSet(t, nil, cD), combat.Blinded{}, "ConditionAdded", condKinds(cB), []combat.ConditionEvent{condApplied(combat.Blinded{})}},
		{"Poisoned while Petrified", `FR-023, FR-125; "Petrified", p. 186`, "unhappy (rule interaction)", condSet(t, condConds(combat.Petrified{})), combat.Poisoned{}, "ConditionImmune", condKinds(cI, cPe), nil},
		{"Poisoned held but suppressed by Petrified is immune, not already held", `FR-024, FR-125`, "unhappy (rule interaction)", petrifiedPoisoned, combat.Poisoned{}, "ConditionImmune", condKinds(cI, cPe), nil},
		{"Incapacitated while Paralyzed", `FR-027, FR-125; "Paralyzed", p. 186`, "unhappy (rule interaction)", condSet(t, condConds(combat.Paralyzed{})), combat.Incapacitated{}, "ConditionAlreadyHeld", condKinds(cI, cPa), nil},
		{"Prone while Unconscious", `FR-027`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{})), combat.Prone{}, "ConditionAlreadyHeld", condKinds(cI, cPr, cU), nil},
		{"Poisoned both stored-immune and derived-immune", `FR-021, FR-023, FR-124`, "unhappy (rule interaction)", condSet(t, condConds(combat.Petrified{}), cPo), combat.Poisoned{}, "ConditionImmune", condKinds(cI, cPe), nil},
		{"Exhaustion immunity", `FR-025; "Immunity", p. 183`, "unhappy (rule interaction)", condSet(t, condConds(condLevel(2)), cE), condLevel(1), "ConditionImmune", nil, nil},
		{"Exhaustion 6 and immune is immune, not at maximum", `FR-125; "Immunity", p. 183`, "unhappy (rule interaction)", condSet(t, condConds(condLevel(6)), cE), condLevel(1), "ConditionImmune", nil, nil},
		{"Exhaustion on an empty set is level 1", `FR-037, FR-123; "Exhaustion", p. 181`, "happy", combat.ConditionSet{}, condLevel(3), "ConditionAdded", condKinds(cE), []combat.ConditionEvent{condApplied(condLevel(1))}},
		{"Exhaustion 1 to 2", `FR-038`, "happy", condSet(t, condConds(condLevel(1))), condLevel(1), "ConditionAdded", condKinds(cE), []combat.ConditionEvent{condApplied(condLevel(2))}},
		{"Exhaustion 2 to 3 whatever the argument", `FR-038, FR-123`, "unhappy (rule interaction)", condSet(t, condConds(condLevel(2))), condLevel(3), "ConditionAdded", condKinds(cE), []combat.ConditionEvent{condApplied(condLevel(3))}},
		{"Exhaustion 5 to 6", `FR-038`, "unhappy (boundary)", condSet(t, condConds(condLevel(5))), condLevel(1), "ConditionAdded", condKinds(cE), []combat.ConditionEvent{condApplied(condLevel(6))}},
		{"Exhaustion at 6 stays 6", `FR-039, FR-125`, "unhappy (boundary)", condSet(t, condConds(condLevel(6))), condLevel(1), "ExhaustionAtMaximum", condKinds(cE), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.set
			res := c.set.Apply(subject(), c.apply)
			if got := reflect.TypeOf(res).Name(); got != c.want {
				t.Fatalf("Apply = %s, want %s [conditions.md %s]", got, c.want, c.requirement)
			}
			if got := res.Set().Effective(); !slices.Equal(got, c.wantEffective) {
				t.Errorf("effective after Apply = %v, want %v", got, c.wantEffective)
			}
			if got := condEvents(res); !slices.Equal(got, c.wantEvents) {
				t.Errorf("events = %v, want %v", got, c.wantEvents)
			}
			if !c.set.Equal(before) {
				t.Errorf("Apply changed the set it was applied to [FR-020]")
			}
			if c.want != "ConditionAdded" && !res.Set().Equal(c.set) {
				t.Errorf("a refused Apply changed the set [FR-115]")
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-019, FR-016 - after a refused second source the original source is kept, and
// the condition stored is the one applied, source included.
// Path: unhappy (rule interaction).
func TestApplyKeepsTheOriginalSource(t *testing.T) {
	t.Parallel()
	s := condSet(t, condConds(combat.Frightened{Source: idA()}))
	if got := s.Apply(subject(), combat.Frightened{Source: idB()}).Set().Explicit(); !slices.Equal(got, condConds(combat.Frightened{Source: idA()})) {
		t.Errorf("explicit after a second source = %v, want the first source", got)
	}
	added := combat.ConditionSet{}.Apply(subject(), combat.Frightened{Source: idB()}).Set().Explicit()
	if !slices.Equal(added, condConds(combat.Frightened{Source: idB()})) {
		t.Errorf("explicit after Apply = %v, want the applied source", added)
	}
}

// Requirement: doc/specs/conditions.md FR-017 - applying each of the fifteen kinds to an empty set gives a set whose
// only explicit condition is that one.
// Path: unhappy (empty and minimal).
func TestApplyEachKindToAnEmptySet(t *testing.T) {
	t.Parallel()
	for _, c := range allFifteen() {
		res, ok := combat.ConditionSet{}.Apply(subject(), c).(combat.ConditionAdded)
		if !ok {
			t.Errorf("Apply(%v) was not ConditionAdded", c)
			continue
		}
		if got := res.Set().Explicit(); len(got) != 1 || got[0].Kind() != c.Kind() {
			t.Errorf("Apply(%v) explicit = %v, want only that kind", c, got)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestRemove(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		set                     combat.ConditionSet
		remove                  content.ConditionKind
		want                    string
		wantEffective           []content.ConditionKind
		wantExplicit            []combat.Condition
		wantEvents              []combat.ConditionEvent
	}{
		{"an explicit condition", `FR-031; "Condition", p. 179`, "happy", condSet(t, condConds(combat.Blinded{}, combat.Poisoned{})), cB, "ConditionCleared", condKinds(cPo), condConds(combat.Poisoned{}), []combat.ConditionEvent{condRemoved(cB)}},
		{"a kind the creature does not have", `FR-032`, "unhappy (state error)", condSet(t, condConds(combat.Blinded{})), cD, "ConditionRemovalNotPresent", condKinds(cB), condConds(combat.Blinded{}), nil},
		{"from an empty set", `FR-032`, "unhappy (empty and minimal)", combat.ConditionSet{}, cB, "ConditionRemovalNotPresent", nil, nil, nil},
		{"a kind held only by implication", `FR-033, FR-125; "Paralyzed", p. 186`, "unhappy (state error)", condSet(t, condConds(combat.Paralyzed{})), cI, "ConditionRemovalImpliedBy", condKinds(cI, cPa), condConds(combat.Paralyzed{}), nil},
		{"an explicit and implied kind stays effective", `FR-034; "Paralyzed", p. 186`, "unhappy (rule interaction)", condSet(t, condConds(combat.Paralyzed{}, combat.Incapacitated{})), cI, "ConditionCleared", condKinds(cI, cPa), condConds(combat.Paralyzed{}), []combat.ConditionEvent{condRemoved(cI)}},
		{"Unconscious leaves the creature Prone", `FR-029, FR-127; "Unconscious", p. 191`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{})), cU, "ConditionCleared", condKinds(cPr), condConds(combat.Prone{}), []combat.ConditionEvent{condRemoved(cU)}},
		{"Unconscious while already Prone", `FR-029`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{}, combat.Prone{})), cU, "ConditionCleared", condKinds(cPr), condConds(combat.Prone{}), []combat.ConditionEvent{condRemoved(cU)}},
		{"Unconscious with Paralyzed keeps Incapacitated", `FR-030; pp. 186, 191`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{}, combat.Paralyzed{})), cU, "ConditionCleared", condKinds(cI, cPa, cPr), condConds(combat.Paralyzed{}, combat.Prone{}), []combat.ConditionEvent{condRemoved(cU)}},
		{"Unconscious suppressed by an immunity leaves no Prone", `FR-029, FR-125`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{}), cU), cU, "ConditionCleared", nil, nil, []combat.ConditionEvent{condRemoved(cU)}},
		{"Unconscious when immune to Prone leaves no Prone", `FR-029, FR-124`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{}), cPr), cU, "ConditionCleared", nil, nil, []combat.ConditionEvent{condRemoved(cU)}},
		{"an explicit condition suppressed by an immunity", `FR-024, FR-125`, "unhappy (rule interaction)", condSet(t, condConds(combat.Petrified{}, combat.Poisoned{})), cPo, "ConditionCleared", condKinds(cI, cPe), condConds(combat.Petrified{}), []combat.ConditionEvent{condRemoved(cPo)}},
		{"a kind below Exhaustion leaves Exhaustion alone", `FR-031, FR-036`, "unhappy (rule interaction)", condSet(t, condConds(combat.Blinded{}, condLevel(3))), cB, "ConditionCleared", condKinds(cE), condConds(condLevel(3)), []combat.ConditionEvent{condRemoved(cB)}},
		{"a kind above Exhaustion leaves Exhaustion alone", `FR-031, FR-036`, "unhappy (rule interaction)", condSet(t, condConds(combat.Poisoned{}, condLevel(3))), cPo, "ConditionCleared", condKinds(cE), condConds(condLevel(3)), []combat.ConditionEvent{condRemoved(cPo)}},
		{"Unconscious suppressed by an immunity leaves no Prone beside another condition", `FR-029, FR-125`, "unhappy (rule interaction)", condSet(t, condConds(combat.Unconscious{}, combat.Blinded{}), cU), cU, "ConditionCleared", condKinds(cB), condConds(combat.Blinded{}), []combat.ConditionEvent{condRemoved(cU)}},
		{"Exhaustion at 3 lowers to 2", `FR-036; "Exhaustion", p. 181`, "unhappy (boundary)", condSet(t, condConds(condLevel(3))), cE, "ExhaustionLowered", condKinds(cE), condConds(condLevel(2)), []combat.ConditionEvent{condLowered(2)}},
		{"Exhaustion at 2 lowers to 1", `FR-036`, "unhappy (boundary)", condSet(t, condConds(condLevel(2))), cE, "ExhaustionLowered", condKinds(cE), condConds(condLevel(1)), []combat.ConditionEvent{condLowered(1)}},
		{"Exhaustion at 1 ends", `FR-036`, "unhappy (boundary)", condSet(t, condConds(condLevel(1))), cE, "ConditionCleared", nil, nil, []combat.ConditionEvent{condRemoved(cE)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.set
			res := c.set.Remove(subject(), c.remove)
			if got := reflect.TypeOf(res).Name(); got != c.want {
				t.Fatalf("Remove = %s, want %s [conditions.md %s]", got, c.want, c.requirement)
			}
			if got := res.Set().Effective(); !slices.Equal(got, c.wantEffective) {
				t.Errorf("effective after Remove = %v, want %v", got, c.wantEffective)
			}
			if got := res.Set().Explicit(); !slices.Equal(got, c.wantExplicit) {
				t.Errorf("explicit after Remove = %v, want %v", got, c.wantExplicit)
			}
			if got := condEvents(res); !slices.Equal(got, c.wantEvents) {
				t.Errorf("events = %v, want %v", got, c.wantEvents)
			}
			if !c.set.Equal(before) {
				t.Errorf("Remove changed the set it was removed from [FR-035]")
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-033 - ConditionRemovalImpliedBy names the implying kinds in kind order, as a
// copy.
// Path: unhappy (state error).
func TestRemoveImpliedByNamesTheImplyingKinds(t *testing.T) {
	t.Parallel()
	res, ok := condSet(t, condConds(combat.Stunned{}, combat.Paralyzed{})).Remove(subject(), cI).(combat.ConditionRemovalImpliedBy)
	if !ok {
		t.Fatal("not ConditionRemovalImpliedBy")
	}
	if got := res.ImpliedBy(); !slices.Equal(got, condKinds(cPa, cS)) {
		t.Errorf("ImpliedBy() = %v, want Paralyzed then Stunned", got)
	}
	res.ImpliedBy()[0] = cB
	if got := res.ImpliedBy(); got[0] != cPa {
		t.Errorf("ImpliedBy aliases its storage")
	}
}

// Requirement: doc/specs/conditions.md FR-033 - a kind that two implied kinds of one condition both name is reported with
// that condition once: Unconscious implies Incapacitated and Prone.
// Path: unhappy (state error).
func TestRemoveImpliedByNamesAConditionOnce(t *testing.T) {
	t.Parallel()
	for _, k := range condKinds(cI, cPr) {
		res, ok := condSet(t, condConds(combat.Unconscious{})).Remove(subject(), k).(combat.ConditionRemovalImpliedBy)
		if !ok || !slices.Equal(res.ImpliedBy(), condKinds(cU)) {
			t.Errorf("Remove(%v) from Unconscious = %#v, want ConditionRemovalImpliedBy by Unconscious once", k, res)
		}
	}
}

// Requirement: doc/specs/conditions.md FR-127 - removing Unconscious emits only ConditionRemoved while the returned
// set holds Prone, so replaying the events alone does not show Prone.
// Path: unhappy (rule interaction).
func TestUnconsciousRemovalEventsAloneDoNotShowProne(t *testing.T) {
	t.Parallel()
	res := condSet(t, condConds(combat.Unconscious{})).Remove(subject(), cU)
	if !slices.Contains(res.Set().Effective(), cPr) {
		t.Fatalf("the returned set does not hold Prone: %v", res.Set().Effective())
	}
	replayed := combat.ConditionSet{}
	for _, e := range condEvents(res) {
		switch ev := e.(type) {
		case combat.ConditionApplied:
			replayed = replayed.Apply(subject(), ev.Condition()).Set()
		}
	}
	if slices.Contains(replayed.Effective(), cPr) {
		t.Errorf("replaying the events built Prone; the spec says the set, not the events, shows it")
	}
}

// Requirement: per case
// Path: per case.
func TestReduceExhaustion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		set                     combat.ConditionSet
		levels                  int
		want                    string
		wantExplicit            []combat.Condition
		wantEvents              []combat.ConditionEvent
		dead                    bool
	}{
		{"Long Rest at 3", `FR-043; "Long Rest", p. 185`, "happy", condSet(t, condConds(condLevel(3))), 1, "ExhaustionReduced", condConds(condLevel(2)), []combat.ConditionEvent{condLowered(2)}, false},
		{"two levels at 5", `FR-043`, "happy", condSet(t, condConds(condLevel(5))), 2, "ExhaustionReduced", condConds(condLevel(3)), []combat.ConditionEvent{condLowered(3)}, false},
		{"from 6 to 5 is no longer dead", `FR-047`, "unhappy (boundary)", condSet(t, condConds(condLevel(6))), 1, "ExhaustionReduced", condConds(condLevel(5)), []combat.ConditionEvent{condLowered(5)}, false},
		{"level 1 reaches 0", `FR-044; "Exhaustion", p. 181`, "unhappy (boundary)", condSet(t, condConds(condLevel(1))), 1, "ExhaustionEnded", nil, []combat.ConditionEvent{condRemoved(cE)}, false},
		{"exactly to 0", `FR-044`, "unhappy (boundary)", condSet(t, condConds(condLevel(3))), 3, "ExhaustionEnded", nil, []combat.ConditionEvent{condRemoved(cE)}, false},
		{"more than present", `FR-044`, "unhappy (boundary)", condSet(t, condConds(condLevel(2))), 5, "ExhaustionEnded", nil, []combat.ConditionEvent{condRemoved(cE)}, false},
		{"no Exhaustion", `FR-046`, "unhappy (state error)", combat.ConditionSet{}, 1, "ExhaustionReductionNotPresent", nil, nil, false},
		{"Exhaustion suppressed by an immunity is still reduced", `FR-125`, "unhappy (rule interaction)", condSet(t, condConds(condLevel(3)), cE), 1, "ExhaustionReduced", condConds(condLevel(2)), []combat.ConditionEvent{condLowered(2)}, false},
		{"other conditions are kept", `FR-043`, "happy", condSet(t, condConds(combat.Blinded{}, condLevel(2))), 1, "ExhaustionReduced", condConds(combat.Blinded{}, condLevel(1)), []combat.ConditionEvent{condLowered(1)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.set
			res, err := c.set.ReduceExhaustion(subject(), c.levels)
			if err != nil {
				t.Fatal(err)
			}
			if got := reflect.TypeOf(res).Name(); got != c.want {
				t.Fatalf("ReduceExhaustion = %s, want %s [conditions.md %s]", got, c.want, c.requirement)
			}
			if got := res.Set().Explicit(); !slices.Equal(got, c.wantExplicit) {
				t.Errorf("explicit after = %v, want %v", got, c.wantExplicit)
			}
			if got := condEvents(res); !slices.Equal(got, c.wantEvents) {
				t.Errorf("events = %v, want %v", got, c.wantEvents)
			}
			if !c.set.Equal(before) {
				t.Errorf("ReduceExhaustion changed the set")
			}
			var fx combat.ConditionEffects
			if fx.IsDeadFromExhaustion(res.Set()) {
				t.Errorf("IsDeadFromExhaustion after reducing = true")
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-045, FR-125 - levels below 1 return an error matching ErrInvalidInput and the
// nil result, checked before any other state, even with no Exhaustion; a zero target is invalid too.
// Path: unhappy (invalid input).
func TestReduceExhaustionRejectsBadInput(t *testing.T) {
	t.Parallel()
	for _, levels := range []int{0, -1, math.MinInt} {
		for name, s := range map[string]combat.ConditionSet{"no Exhaustion": {}, "Exhaustion 3": condSet(t, condConds(condLevel(3)))} {
			if res, err := s.ReduceExhaustion(subject(), levels); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
				t.Errorf("%s, levels %d: ReduceExhaustion = %v, %v, want nil and ErrInvalidInput", name, levels, res, err)
			}
		}
	}
	if res, err := condSet(t, condConds(condLevel(3))).ReduceExhaustion("", 1); !errors.Is(err, srderr.ErrInvalidInput) || res != nil {
		t.Errorf("zero target: ReduceExhaustion = %v, %v, want nil and ErrInvalidInput", res, err)
	}
}

// Requirement: doc/specs/conditions.md FR-036, D-15 - Remove of Exhaustion gives the same set and events as
// ReduceExhaustion(1) at every level, with its own result variants.
// Path: unhappy (rule interaction).
func TestRemoveExhaustionMatchesReduceByOne(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 6; n++ {
		s := condSet(t, condConds(condLevel(n)))
		rm := s.Remove(subject(), cE)
		rd, err := s.ReduceExhaustion(subject(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if !rm.Set().Equal(rd.Set()) || !slices.Equal(condEvents(rm), condEvents(rd)) {
			t.Errorf("level %d: Remove and ReduceExhaustion(1) disagree", n)
		}
	}
}

// Requirement: doc/specs/conditions.md FR-115 - a change's events slice is a copy: changing it leaves the result unchanged.
// Path: unhappy (immutability).
func TestChangeEventsAreCopies(t *testing.T) {
	t.Parallel()
	res := combat.ConditionSet{}.Apply(subject(), combat.Blinded{})
	ev := condEvents(res)
	ev[0] = nil
	if condEvents(res)[0] == nil {
		t.Errorf("Events aliases the stored slice")
	}
	var results = []any{combat.ConditionAlreadyHeld{}, combat.ConditionImmune{}, combat.ExhaustionAtMaximum{}, combat.ConditionRemovalNotPresent{},
		combat.ConditionRemovalImpliedBy{}, combat.ExhaustionReductionNotPresent{}}
	for _, r := range results {
		if condEvents(r) != nil {
			t.Errorf("%T carries events", r)
		}
	}
}

// Requirement: doc/specs/conditions.md FR-125, FR-110 - a zero target or an invalid kind or condition changes nothing.
// Path: unhappy (invalid input).
func TestOperationsOnInvalidArgumentsChangeNothing(t *testing.T) {
	t.Parallel()
	s := condSet(t, condConds(combat.Blinded{}))
	if r, ok := s.Apply("", combat.Prone{}).(combat.ConditionAlreadyHeld); !ok || !r.Set().Equal(s) {
		t.Errorf("Apply with a zero target = %#v", r)
	}
	if r, ok := s.Remove("", cB).(combat.ConditionRemovalNotPresent); !ok || !r.Set().Equal(s) {
		t.Errorf("Remove with a zero target = %#v", r)
	}
	if r, ok := s.Remove(subject(), 0).(combat.ConditionRemovalNotPresent); !ok || !r.Set().Equal(s) {
		t.Errorf("Remove of kind 0 = %#v", r)
	}
}
