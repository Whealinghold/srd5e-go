package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const (
	cB, cC, cD    = content.ConditionBlinded, content.ConditionCharmed, content.ConditionDeafened
	cE, cF, cG    = content.ConditionExhaustion, content.ConditionFrightened, content.ConditionGrappled
	cI, cIv, cPa  = content.ConditionIncapacitated, content.ConditionInvisible, content.ConditionParalyzed
	cPe, cPo, cPr = content.ConditionPetrified, content.ConditionPoisoned, content.ConditionProne
	cR, cS, cU    = content.ConditionRestrained, content.ConditionStunned, content.ConditionUnconscious
)

// Requirement: doc/specs/conditions.md FR-011, FR-109 - an empty ConditionSet, the zero value included, has no explicit,
// no effective and no immune conditions.
// Path: unhappy (empty and minimal).
func TestEmptyConditionSet(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]combat.ConditionSet{"zero value": {}, "built empty": condSet(t, nil), "built from empty slices": condSet(t, condConds())} {
		if len(s.Effective()) != 0 || len(s.Explicit()) != 0 || len(s.Immunities()) != 0 || !s.Equal(combat.ConditionSet{}) {
			t.Errorf("%s: effective %v explicit %v immunities %v, want all empty", name, s.Effective(), s.Explicit(), s.Immunities())
		}
	}
}

// Requirement: doc/specs/conditions.md FR-012, FR-141 - NewConditionSet rejects two conditions of one kind (two
// Exhaustion, two Charmed with different charmers), a nil condition and an invalid immunity kind, each with an error
// matching ErrInvalidInput (SRD 5.2.1, "Condition", p. 179).
// Path: unhappy (invalid input).
func TestNewConditionSetRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		explicit []combat.Condition
		immune   []content.ConditionKind
	}{
		{"two Exhaustion", condConds(condLevel(1), condLevel(2)), nil},
		{"two Charmed with different charmers", condConds(combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idA()}}, combat.Charmed{Charmer: combat.CharmedByCombatant{ID: idB()}}), nil},
		{"the same condition twice", condConds(combat.Blinded{}, combat.Blinded{}), nil},
		{"a nil condition", condConds(combat.Blinded{}, nil), nil},
		{"an immunity of kind 0", nil, condKinds(0)},
		{"an immunity of kind 16", nil, condKinds(cB, 16)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := combat.NewConditionSet(c.explicit, c.immune)
			if !errors.Is(err, srderr.ErrInvalidInput) || !s.Equal(combat.ConditionSet{}) {
				t.Errorf("NewConditionSet = %v, %v, want the empty set and ErrInvalidInput", s, err)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-012 - MustConditionSet panics with the error NewConditionSet returns for a
// duplicate kind and returns the set otherwise.
// Path: unhappy (invalid input).
func TestMustConditionSet(t *testing.T) {
	t.Parallel()
	if s := combat.MustConditionSet(condConds(combat.Blinded{}), nil); !slices.Equal(s.Effective(), condKinds(cB)) {
		t.Errorf("MustConditionSet(Blinded) effective = %v", s.Effective())
	}
	defer func() {
		if err, ok := recover().(error); !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an error matching ErrInvalidInput", err)
		}
	}()
	combat.MustConditionSet(condConds(combat.Blinded{}, combat.Blinded{}), nil)
}

// Requirement: doc/specs/conditions.md FR-014, FR-142 - a ConditionSet copies its input and returns copies from Explicit,
// Effective and Immunities, so changing a passed-in or read-out slice never changes the set; Explicit is ordered by kind
// with at most one per kind.
// Path: unhappy (immutability).
func TestConditionSetCopiesInAndOut(t *testing.T) {
	t.Parallel()
	in := condConds(combat.Stunned{}, combat.Blinded{}, combat.Grappled{Grappler: idA()})
	immune := condKinds(cD)
	s := condSet(t, in, immune...)
	in[0], immune[0] = combat.Poisoned{}, cC
	want := condConds(combat.Blinded{}, combat.Grappled{Grappler: idA()}, combat.Stunned{})
	if got := s.Explicit(); !slices.Equal(got, want) {
		t.Errorf("Explicit() = %v, want %v ordered by kind, unaffected by the caller's slice", got, want)
	}
	s.Explicit()[0] = combat.Prone{}
	s.Effective()[0] = cU
	s.Immunities()[0] = cC
	if !slices.Equal(s.Explicit(), want) || !slices.Equal(s.Effective(), condKinds(cB, cG, cI, cS)) || !slices.Equal(s.Immunities(), condKinds(cD)) {
		t.Errorf("a read-out slice changed the set: %v %v %v", s.Explicit(), s.Effective(), s.Immunities())
	}
}

// Requirement: doc/specs/conditions.md FR-015 - two sets with the same explicit conditions and immunities are equal in
// either supply order and sets differing in either are not.
// Path: per case.
func TestConditionSetEqual(t *testing.T) {
	t.Parallel()
	base := condSet(t, condConds(combat.Blinded{}, combat.Poisoned{}), cD, cC)
	cases := []struct {
		name  string
		other combat.ConditionSet
		equal bool
	}{
		{"same parts in another order", condSet(t, condConds(combat.Poisoned{}, combat.Blinded{}), cC, cD), true},
		{"a different explicit condition", condSet(t, condConds(combat.Blinded{}, combat.Prone{}), cD, cC), false},
		{"a missing explicit condition", condSet(t, condConds(combat.Blinded{}), cD, cC), false},
		{"a different immunity", condSet(t, condConds(combat.Blinded{}, combat.Poisoned{}), cD, cB), false},
		{"a missing immunity", condSet(t, condConds(combat.Blinded{}, combat.Poisoned{}), cD), false},
		{"a different source", condSet(t, condConds(combat.Blinded{}, combat.Poisoned{}, combat.Frightened{Source: idA()}), cD, cC), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := base.Equal(c.other); got != c.equal || c.other.Equal(base) != c.equal {
				t.Errorf("Equal = %v, want %v", got, c.equal)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestEffectiveConditionsPipeline(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		explicit                []combat.Condition
		immune                  []content.ConditionKind
		want                    []content.ConditionKind
	}{
		{"Paralyzed implies Incapacitated", `FR-026; "Paralyzed", p. 186`, "happy", condConds(combat.Paralyzed{}), nil, condKinds(cI, cPa)},
		{"Petrified implies Incapacitated", `FR-026; "Petrified", p. 186`, "happy", condConds(combat.Petrified{}), nil, condKinds(cI, cPe)},
		{"Stunned implies Incapacitated", `FR-026; "Stunned", p. 189`, "happy", condConds(combat.Stunned{}), nil, condKinds(cI, cS)},
		{"Unconscious implies Incapacitated and Prone", `FR-026; "Unconscious", p. 191`, "happy", condConds(combat.Unconscious{}), nil, condKinds(cI, cPr, cU)},
		{"Blinded implies nothing", `FR-026; "Blinded", p. 177`, "unhappy (boundary)", condConds(combat.Blinded{}), nil, condKinds(cB)},
		{"Prone implies nothing", `FR-026; "Prone", p. 186`, "unhappy (boundary)", condConds(combat.Prone{}), nil, condKinds(cPr)},
		{"Paralyzed and Stunned give one Incapacitated", `FR-028; pp. 186, 189`, "unhappy (rule interaction)", condConds(combat.Paralyzed{}, combat.Stunned{}), nil, condKinds(cI, cPa, cS)},
		{"Unconscious and Prone explicit", `FR-027, FR-028; pp. 186, 191`, "unhappy (rule interaction)", condConds(combat.Unconscious{}, combat.Prone{}), nil, condKinds(cI, cPr, cU)},
		{"a suppressed explicit condition implies nothing", `FR-124; "Immunity", p. 183`, "unhappy (rule interaction)", condConds(combat.Paralyzed{}), condKinds(cPa), nil},
		{"an implied kind is blocked", `FR-124; "Immunity", p. 183`, "unhappy (rule interaction)", condConds(combat.Stunned{}), condKinds(cI), condKinds(cS)},
		{"Petrified suppresses Poisoned", `FR-024, FR-124; "Petrified", p. 186`, "unhappy (rule interaction)", condConds(combat.Petrified{}, combat.Poisoned{}), nil, condKinds(cI, cPe)},
		{"Petrified itself suppressed gives no immunity to Poisoned", `FR-124; pp. 183, 186`, "unhappy (rule interaction)", condConds(combat.Petrified{}, combat.Poisoned{}), condKinds(cPe), condKinds(cPo)},
		{"an implied Prone is blocked", `FR-124; pp. 183, 191`, "unhappy (rule interaction)", condConds(combat.Unconscious{}), condKinds(cPr), condKinds(cI, cU)},
		{"effective kinds are ordered by kind", `FR-013, FR-130`, "unhappy (determinism)", condConds(combat.Unconscious{}, combat.Blinded{}, condLevel(2)), nil, condKinds(cB, cE, cI, cPr, cU)},
		{"an unrelated immunity blocks nothing", `FR-022; "Immunity", p. 17`, "unhappy (boundary)", condConds(combat.Blinded{}), condKinds(cD), condKinds(cB)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := condSet(t, c.explicit, c.immune...)
			if got := s.Effective(); !slices.Equal(got, c.want) {
				t.Errorf("Effective() = %v, want %v [conditions.md %s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/conditions.md FR-013, FR-108 - the effective conditions do not depend on the order the explicit
// conditions were supplied in.
// Path: unhappy (determinism).
func TestEffectiveConditionsAreOrderIndependent(t *testing.T) {
	t.Parallel()
	fwd := allFifteen()
	rev := slices.Clone(fwd)
	slices.Reverse(rev)
	a, b := condSet(t, fwd), condSet(t, rev)
	if !a.Equal(b) || !slices.Equal(a.Effective(), b.Effective()) || !slices.Equal(a.Explicit(), b.Explicit()) {
		t.Errorf("sets built in two orders differ")
	}
}

// Requirement: doc/specs/conditions.md FR-021, FR-022 - an immunity listed twice is still one immunity: the set stays
// immune and reports it once.
// Path: unhappy (rule interaction).
func TestDuplicateImmunitiesAreOneImmunity(t *testing.T) {
	t.Parallel()
	s := condSet(t, nil, cB, cB, cB)
	if !slices.Equal(s.Immunities(), condKinds(cB)) || !s.Equal(condSet(t, nil, cB)) {
		t.Errorf("Immunities() = %v, want Blinded once", s.Immunities())
	}
	if _, ok := s.Apply(subject(), combat.Blinded{}).(combat.ConditionImmune); !ok {
		t.Errorf("a doubly listed immunity let the condition in")
	}
}

// Requirement: doc/specs/conditions.md FR-011 - a stored immunity listed more than once is still one immunity: the kind is immune, not toggled back off by the repeat.
// Path: unhappy (boundary).
func TestConditionSetRepeatedImmunityStaysImmune(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 2, 3} {
		kinds := slices.Repeat([]content.ConditionKind{cPo}, n)
		s, err := combat.NewConditionSet(condConds(combat.Poisoned{}), kinds)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(s.Immunities(), condKinds(cPo)) || len(s.Effective()) != 0 {
			t.Errorf("%d listings of Poisoned: immunities %v effective %v, want one immunity and no effective Poisoned", n, s.Immunities(), s.Effective())
		}
	}
}
