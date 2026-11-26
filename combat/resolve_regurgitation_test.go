package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ctDamaged records damage from inside for each occupant: pairs of (occupant, amount).
func ctDamaged(tb testing.TB, s combat.Containments, damage map[string]int) combat.Containments {
	tb.Helper()
	for who, n := range damage {
		tr, err := s.RecordDamageFromInside(gid(who), n)
		if err != nil {
			tb.Fatal(err)
		}
		if applied, ok := tr.(combat.ContainmentTransitionApplied); ok {
			s = applied.Containments()
		}
	}
	return s
}

func ctRegurgitate(tb testing.TB, in combat.ResolveRegurgitationInput, faces ...int) (combat.ResolveRegurgitationResult, *dicetest.FixedRoller) {
	tb.Helper()
	cmd, err := combat.NewResolveRegurgitationCommand(in)
	if err != nil {
		tb.Fatalf("NewResolveRegurgitationCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewResolveRegurgitation(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res, roller
}

func ctConds(tb testing.TB, ids ...string) map[combat.CombatantID]combat.ConditionSet {
	tb.Helper()
	m := map[combat.CombatantID]combat.ConditionSet{}
	for _, id := range ids {
		m[gid(id)] = condSet(tb, []combat.Condition{combat.Blinded{}, combat.Restrained{}})
	}
	return m
}

func ctTallies(s combat.Containments) []int {
	var out []int
	for _, r := range s.All() {
		out = append(out, r.DamageFromInside())
	}
	return out
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-047, FR-049, FR-050, Q-1, table 4.7.3 - the threshold is per occupant (29 of 30 is not due, 30 is, 20 plus 20 is not), a swallow with no regurgitation terms is never due, a trapped record never counts, no die is rolled when nothing is due, and every outcome resets the container's tallies.
// Path: per case.
func TestResolveRegurgitationDue(t *testing.T) {
	t.Parallel()
	toad := ctHolding(t, "toad", "m", "p")
	cases := []struct {
		name, path string
		set        combat.Containments
		dice       int
	}{
		{"29 of 30", "unhappy (boundary)", ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"p": 29}), 0},
		{"30 of 30", "unhappy (boundary)", ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"p": 30}), 1},
		{"31 of 30", "unhappy (boundary)", ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"q": 31}), 1},
		{"20 and 20 are not summed", "unhappy (rule interaction)", ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"p": 20, "q": 20}), 0},
		{"no regurgitation terms", "unhappy (boundary)", ctDamaged(t, toad, map[string]int{"p": 100}), 0},
		{"nothing recorded", "unhappy (empty and minimal)", ctHolding(t, "worm", "m", "p"), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := combat.ResolveRegurgitationInput{Containments: c.set, Container: gid("m"), OccupantConditions: ctConds(t, "p", "q"), Attempt: rollOf(0)}
			res, roller := ctRegurgitate(t, in, 20)
			if len(roller.Requests()) != c.dice {
				t.Fatalf("dice = %d, want %d", len(roller.Requests()), c.dice)
			}
			var tallies combat.Containments
			switch v := res.(type) {
			case combat.ResolveRegurgitationNotDue:
				tallies = v.Containments()
			case combat.ResolveRegurgitationHeld:
				tallies = v.Containments()
			case combat.ResolveRegurgitationRegurgitated:
				tallies = v.Containments()
			default:
				t.Fatalf("result = %#v", res)
			}
			if c.dice == 0 && reflect.TypeOf(res) != reflect.TypeOf(combat.ResolveRegurgitationNotDue{}) {
				t.Errorf("result = %#v, want not due", res)
			}
			if !slices.Equal(ctTallies(tallies), make([]int, tallies.Len())) {
				t.Errorf("tallies after = %v, want all reset (FR-049)", ctTallies(tallies))
			}
		})
	}
	// A trapped record never counts, and cannot be recorded.
	trapped := ctSet(t, ctRecord(t, "m", "p", content.SizeMedium, ctTerms(t, "worm"), 1, func(in *combat.ContainmentInput) {
		in.Phase, in.DamageFromInside = mustTrapped(t, 20), 40
	}))
	if tr, err := trapped.RecordDamageFromInside(gid("p"), 40); err != nil || tr != (combat.ContainmentTransitionRejected{Rejection: combat.ContainmentRejectionNotContained}) {
		t.Errorf("recording on a trapped record = %#v, %v", tr, err)
	}
	res, roller := ctRegurgitate(t, combat.ResolveRegurgitationInput{Containments: trapped, Container: gid("m"), Attempt: rollOf(0)})
	if _, ok := res.(combat.ResolveRegurgitationNotDue); !ok || len(roller.Requests()) != 0 {
		t.Errorf("a trapped record with a tally = %#v, want not due and no die", res)
	}
	// Only the called container's tallies are reset.
	two := ctSet(t, ctRecord(t, "m", "p", content.SizeMedium, ctTerms(t, "worm"), 1, func(in *combat.ContainmentInput) { in.DamageFromInside = 9 }),
		ctRecord(t, "n", "q", content.SizeMedium, ctTerms(t, "worm"), 1, func(in *combat.ContainmentInput) { in.DamageFromInside = 9 }))
	res, _ = ctRegurgitate(t, combat.ResolveRegurgitationInput{Containments: two, Container: gid("m"), Attempt: rollOf(0)})
	if !slices.Equal(ctTallies(res.(combat.ResolveRegurgitationNotDue).Containments()), []int{0, 9}) {
		t.Errorf("tallies = %v, want only m's reset", ctTallies(res.(combat.ResolveRegurgitationNotDue).Containments()))
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-048, FR-056, FR-079, table 4.7.3 - the container's Constitution save against the terms' DC: a failure (total 20 against 21) releases every occupant, not only the damaging one, Prone, landing within the terms' distance; a success (21) keeps them; a forced failure draws no die.
// Path: per case.
func TestResolveRegurgitationSave(t *testing.T) {
	t.Parallel()
	set := ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"p": 31})
	in := combat.ResolveRegurgitationInput{Containments: set, Container: gid("m"), OccupantConditions: ctConds(t, "p", "q"), Attempt: rollOf(0)}
	res, roller := ctRegurgitate(t, in, 20)
	out, ok := res.(combat.ResolveRegurgitationRegurgitated)
	if !ok || out.Containments().Len() != 0 || len(out.Releases()) != 2 || len(roller.Requests()) != 1 {
		t.Fatalf("save failure = %#v, want both occupants released", res)
	}
	if rolled := out.Check().(combat.ContainmentTestRolled); rolled.Result().Kind() != d20.SavingThrow || rolled.Result().Target() != 21 || rolled.Result().Total() != 20 {
		t.Errorf("check = %+v, want a saving throw of 20 against 21", rolled.Result())
	}
	for i, rel := range out.Releases() {
		if rel.Cause() != combat.ContainmentLeaveCauseRegurgitated || rel.Occupant != gid([]string{"p", "q"}[i]) || !slices.Equal(rel.Conditions().Effective(), []content.ConditionKind{content.ConditionProne}) ||
			rel.Placement() != (combat.LandingWithin{Feet: 5}) {
			t.Errorf("release %d = %+v", i, rel)
		}
	}
	if len(out.Events()) != 8 || out.Events()[0].Kind() != "condition_removed" {
		t.Errorf("events = %v", ctKinds(out.Events()))
	}
	// One more than the DC is a success.
	in.Attempt = rollOf(1)
	res, _ = ctRegurgitate(t, in, 20)
	kept, ok := res.(combat.ResolveRegurgitationHeld)
	if !ok || kept.Containments().Len() != 2 || !slices.Equal(ctTallies(kept.Containments()), []int{0, 0}) {
		t.Errorf("save success = %#v, want held with the tallies reset", res)
	}
	// A forced failure draws no die.
	in.Attempt = d20.ChooseToFail{}
	res, roller = ctRegurgitate(t, in)
	if _, ok := res.(combat.ResolveRegurgitationRegurgitated); !ok || len(roller.Requests()) != 0 {
		t.Errorf("a forced failure = %#v with %d dice", res, len(roller.Requests()))
	}
	// The Behir lands its meal within 10 feet; a creature immune to Prone is reported.
	behir := ctDamaged(t, ctHolding(t, "behir", "m", "p"), map[string]int{"p": 30})
	immune := map[combat.CombatantID]combat.ConditionSet{gid("p"): condSet(t, []combat.Condition{combat.Blinded{}, combat.Restrained{}}, content.ConditionProne)}
	res, _ = ctRegurgitate(t, combat.ResolveRegurgitationInput{Containments: behir, Container: gid("m"), OccupantConditions: immune, Attempt: rollOf(0)}, 1)
	rel := res.(combat.ResolveRegurgitationRegurgitated).Releases()[0]
	if p, ok := rel.Prone(); rel.Placement() != (combat.LandingWithin{Feet: 10}) || !ok || p != combat.ConditionImpositionImmune || len(rel.Conditions().Effective()) != 0 {
		t.Errorf("Behir release = placement %v prone %v/%v conditions %v", rel.Placement(), p, ok, rel.Conditions().Effective())
	}
	if set.All()[0].DamageFromInside() != 31 {
		t.Error("the input set must not change")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-063 - when a save is due the map of conditions holds exactly the container's occupants; a missing or extra id is an Invalid result with no die and no tally reset.
// Path: per case.
func TestResolveRegurgitationConditionMap(t *testing.T) {
	t.Parallel()
	set := ctDamaged(t, ctHolding(t, "worm", "m", "p", "q"), map[string]int{"p": 31})
	for name, c := range map[string]struct {
		have map[combat.CombatantID]combat.ConditionSet
		want combat.ContainmentProblem
	}{
		"missing": {ctConds(t, "p"), combat.ContainmentProblemMissingOccupantConditions},
		"no map":  {nil, combat.ContainmentProblemMissingOccupantConditions},
		"extra":   {ctConds(t, "p", "q", "zz"), combat.ContainmentProblemUnknownOccupant},
	} {
		res, roller := ctRegurgitate(t, combat.ResolveRegurgitationInput{Containments: set, Container: gid("m"), OccupantConditions: c.have, Attempt: rollOf(0)}, 20)
		if res != (combat.ResolveRegurgitationInvalid{Problem: c.want}) || len(roller.Requests()) != 0 {
			t.Errorf("%s: result = %#v with %d dice", name, res, len(roller.Requests()))
		}
	}
	for name, in := range map[string]combat.ResolveRegurgitationInput{
		"blank container": {Attempt: rollOf(0)}, "no attempt": {Container: gid("m")}, "pointer attempt": {Container: gid("m"), Attempt: &d20.ChooseToFail{}}} {
		if _, err := combat.NewResolveRegurgitationCommand(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	// The command copies the map: changing it afterwards does not change the run.
	have := ctConds(t, "p", "q")
	cmd, _ := combat.NewResolveRegurgitationCommand(combat.ResolveRegurgitationInput{Containments: set, Container: gid("m"), OccupantConditions: have, Attempt: rollOf(0)})
	delete(have, gid("p"))
	uc, _ := combat.NewResolveRegurgitation(dicetest.NewFixedRoller(1))
	if res, err := uc.Run(context.Background(), cmd); err != nil || reflect.TypeOf(res) != reflect.TypeOf(combat.ResolveRegurgitationRegurgitated{}) {
		t.Errorf("result = %#v, %v", res, err)
	}
}
