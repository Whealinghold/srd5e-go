package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-039, FR-040, FR-041, Q-3 - CoverBetween is Total only for a held creature of a Total Cover container and a source outside it: not the container, not a fellow occupant, not the creature itself; RestrictionsOn reads the terms.
// Path: per case.
func TestContainmentCoverAndRestrictions(t *testing.T) {
	t.Parallel()
	cube, mound := ctHolding(t, "cube", "cube", "p", "q"), ctHolding(t, "mound", "mound", "m")
	var rules combat.ContainmentRules
	cases := []struct {
		name, path     string
		set            combat.Containments
		source, target string
		want           combat.ContainmentCover
	}{
		{"an outsider against a held creature", "happy", cube, "outsider", "p", combat.ContainmentCoverTotal},
		{"the container", "unhappy (rule interaction)", cube, "cube", "p", combat.ContainmentCoverNone},
		{"a fellow occupant", "unhappy (rule interaction)", cube, "q", "p", combat.ContainmentCoverNone},
		{"the creature itself", "unhappy (rule interaction)", cube, "p", "p", combat.ContainmentCoverNone},
		{"a creature not held", "unhappy (empty and minimal)", cube, "p", "outsider", combat.ContainmentCoverNone},
		{"the mound gives no cover", "unhappy (rule interaction)", mound, "outsider", "m", combat.ContainmentCoverNone},
		{"an occupant of another container", "happy", cube, "m", "p", combat.ContainmentCoverTotal},
		{"no containments", "unhappy (empty and minimal)", combat.Containments{}, "a", "b", combat.ContainmentCoverNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := rules.CoverBetween(c.set, gid(c.source), gid(c.target)); got != c.want {
				t.Errorf("CoverBetween(%s, %s) = %v, want %v", c.source, c.target, got, c.want)
			}
		})
	}
	want := combat.ContainedRestrictions{Cover: combat.ContainmentCoverTotal, NoVerbalSpells: true, Suffocation: combat.SuffocationRuleAlways}
	if got := rules.RestrictionsOn(cube, gid("p")); got != want {
		t.Errorf("RestrictionsOn(cube occupant) = %+v, want %+v", got, want)
	}
	if got := rules.RestrictionsOn(ctHolding(t, "tarrasque", "tar", "x"), gid("x")); !got.NoTeleport || got.NoVerbalSpells {
		t.Errorf("RestrictionsOn(tarrasque occupant) = %+v, want no teleport only", got)
	}
	none := combat.ContainedRestrictions{Cover: combat.ContainmentCoverNone, Suffocation: combat.SuffocationRuleNone}
	if got := rules.RestrictionsOn(cube, gid("outsider")); got != none {
		t.Errorf("RestrictionsOn(not held) = %+v, want none %+v", got, none)
	}
	if got := rules.RestrictionsOn(mound, gid("m")); got.Cover != combat.ContainmentCoverNone || got.Suffocation != combat.SuffocationRuleNone {
		t.Errorf("RestrictionsOn(mound occupant) = %+v", got)
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-042, FR-045 - MovesWith and CarriedFree return the occupants in entry order as copies, and an action is locked exactly while the container holds a creature and its terms name that action.
// Path: per case.
func TestContainmentMovementAndLockedAction(t *testing.T) {
	t.Parallel()
	var rules combat.ContainmentRules
	toads := ctHolding(t, "worm", "w", "a", "b", "c")
	for _, f := range []func(combat.Containments, combat.CombatantID) []combat.CombatantID{rules.MovesWith, rules.CarriedFree} {
		got := f(toads, gid("w"))
		if !slices.Equal(got, []combat.CombatantID{gid("a"), gid("b"), gid("c")}) {
			t.Fatalf("occupants = %v, want [a b c] in entry order", got)
		}
		got[0] = gid("z")
		if f(toads, gid("w"))[0] != gid("a") {
			t.Error("the result must be a copy")
		}
		if got := f(toads, gid("none")); got == nil || len(got) != 0 {
			t.Errorf("a container holding nobody = %#v, want an empty non-nil slice", got)
		}
	}
	frog := ctHolding(t, "frog", "f", "x")
	for _, c := range []struct {
		name   string
		set    combat.Containments
		who    string
		action string
		want   bool
	}{
		{"Bite while holding", frog, "f", "Bite", true},
		{"another action", frog, "f", "Multiattack", false},
		{"an empty name", frog, "f", "", false},
		{"holding nobody", frog, "idle", "Bite", false},
		{"terms with no locked action", toads, "w", "Bite", false},
		{"terms with no locked action and an empty name", toads, "w", "", false},
		{"no containments", combat.Containments{}, "f", "Bite", false},
	} {
		if got := rules.IsActionLocked(c.set, gid(c.who), c.action); got != c.want {
			t.Errorf("%s: IsActionLocked = %v, want %v", c.name, got, c.want)
		}
	}
}

func ctProne() []combat.Condition { return []combat.Condition{combat.Prone{}} }

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-055, FR-056, FR-079, Q-13 - ReleaseOnSet removes exactly the conditions the containment added (a Restrained another source holds stays), emits one OccupantReleased, applies Prone for regurgitated, disgorged and escaped corpse only (immunity and an existing Prone reported), reports breathing restored and places the creature.
// Path: per case.
func TestReleaseOnSet(t *testing.T) {
	t.Parallel()
	var rules combat.ContainmentRules
	toad := ctTerms(t, "toad")
	rec := ctRecord(t, "w", "o", content.SizeMedium, toad, 1, func(in *combat.ContainmentInput) { in.AddedConditions = ctBR() })
	held := condSet(t, []combat.Condition{combat.Blinded{}, combat.Restrained{}})
	causes := []struct {
		cause combat.ContainmentLeaveCause
		prone bool
	}{
		{combat.ContainmentLeaveCauseEscaped, false}, {combat.ContainmentLeaveCausePulledOut, false}, {combat.ContainmentLeaveCauseRegurgitated, true},
		{combat.ContainmentLeaveCauseDisgorged, true}, {combat.ContainmentLeaveCauseContainerDied, false}, {combat.ContainmentLeaveCauseEscapedCorpse, true},
		{combat.ContainmentLeaveCauseGrappleEnded, false},
	}
	for _, c := range causes {
		t.Run(c.cause.String(), func(t *testing.T) {
			t.Parallel()
			rel := rules.ReleaseOnSet(held, rec, c.cause)
			p, hasProne := rel.Prone()
			wantEffective := []content.ConditionKind(nil)
			if c.prone {
				wantEffective = []content.ConditionKind{content.ConditionProne}
			}
			if rel.IsZero() || rel.Cause() != c.cause || !slices.Equal(rel.Conditions().Effective(), wantEffective) || hasProne != c.prone ||
				(c.prone && p != combat.ConditionImpositionAdded) || rel.RestoreBreathing() {
				t.Fatalf("release = %+v prone %v/%v effective %v", rel, p, hasProne, rel.Conditions().Effective())
			}
			ev := rel.Events()
			wantKinds := []string{"condition_removed", "condition_removed", "occupant_released"}
			if c.prone {
				wantKinds = append(wantKinds, "condition_applied")
			}
			var kinds []string
			for _, e := range ev {
				kinds = append(kinds, e.Kind())
			}
			if !slices.Equal(kinds, wantKinds) {
				t.Errorf("event kinds = %v, want %v", kinds, wantKinds)
			}
		})
	}
	if _, ok := rules.ReleaseOnSet(held, rec, combat.ContainmentLeaveCauseEscaped).Placement().(combat.NearestUnoccupiedSpace); !ok {
		t.Error("an escape places the creature in the nearest unoccupied space")
	}
	worm := ctRecord(t, "w", "o", content.SizeMedium, ctTerms(t, "worm"), 1)
	if p, ok := rules.ReleaseOnSet(held, worm, combat.ContainmentLeaveCauseRegurgitated).Placement().(combat.LandingWithin); !ok || p.Feet != 5 {
		t.Errorf("regurgitation places the creature within the landing distance: %#v", p)
	}
	if _, ok := rules.ReleaseOnSet(held, rec, combat.ContainmentLeaveCauseRegurgitated).Placement().(combat.NearestUnoccupiedSpace); !ok {
		t.Error("terms with no regurgitation place a regurgitated creature in the nearest space")
	}

	// A Restrained another source holds is not removed; the record added none.
	other := ctRecord(t, "w", "o", content.SizeMedium, toad, 1, func(in *combat.ContainmentInput) {
		in.AddedConditions = []content.ConditionKind{content.ConditionBlinded}
	})
	if got := rules.ReleaseOnSet(held, other, combat.ContainmentLeaveCauseEscaped).Conditions().Effective(); !slices.Equal(got, []content.ConditionKind{content.ConditionRestrained}) {
		t.Errorf("effective conditions after the release = %v, want the other source's Restrained", got)
	}
	if len(held.Effective()) != 2 {
		t.Error("the input set must not change")
	}

	// Prone: immune and already held are reported.
	immune := condSet(t, nil, content.ConditionProne)
	if p, ok := rules.ReleaseOnSet(immune, rec, combat.ContainmentLeaveCauseDisgorged).Prone(); !ok || p != combat.ConditionImpositionImmune {
		t.Errorf("Prone on an immune creature = %v/%v, want immune", p, ok)
	}
	already := condSet(t, ctProne())
	if p, ok := rules.ReleaseOnSet(already, rec, combat.ContainmentLeaveCauseDisgorged).Prone(); !ok || p != combat.ConditionImpositionAlreadyHeld {
		t.Errorf("Prone on a prone creature = %v/%v, want already held", p, ok)
	}

	// Breathing is restored only when the terms had suffocation; a removed creature's conditions are untouched.
	cube := ctRecord(t, "c", "o", content.SizeMedium, ctTerms(t, "cube"), 1)
	if !rules.ReleaseOnSet(held, cube, combat.ContainmentLeaveCauseEscaped).RestoreBreathing() || rules.ReleaseOnSet(held, rec, combat.ContainmentLeaveCauseEscaped).RestoreBreathing() {
		t.Error("breathing is restored exactly when the terms had suffocation")
	}
	gone := rules.ReleaseOnSet(held, rec, combat.ContainmentLeaveCauseOccupantRemoved)
	if gone.Conditions() != held || gone.Placement() != nil || len(gone.Events()) != 1 || gone.Events()[0].Kind() != "occupant_released" {
		t.Errorf("a removed creature = %+v, want its conditions untouched, no placement and one event", gone)
	}
	if !rules.ReleaseOnSet(held, combat.Containment{}, combat.ContainmentLeaveCauseEscaped).IsZero() || !rules.ReleaseOnSet(held, rec, 0).IsZero() ||
		!rules.ReleaseOnSet(held, rec, 9).IsZero() {
		t.Error("an invalid record or cause gives the zero release")
	}
	ev := rules.ReleaseOnSet(held, rec, combat.ContainmentLeaveCauseEscaped).Events()
	ev[0] = nil
	if rules.ReleaseOnSet(held, rec, combat.ContainmentLeaveCauseEscaped).Events()[0] == nil {
		t.Error("Events must return a copy")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-062 - a release cause is one of exactly eight, with the snake_case names of srd5e.combat.containment.end_reason.
// Path: happy.
func TestContainmentLeaveCauseNames(t *testing.T) {
	t.Parallel()
	want := []string{"escaped", "pulled_out", "regurgitated", "disgorged", "container_died", "escaped_corpse", "grapple_ended", "occupant_removed"}
	for i, w := range want {
		c := combat.ContainmentLeaveCause(i + 1)
		if c.String() != w || !c.Valid() {
			t.Errorf("cause %d = %q valid %v, want %q", i+1, c.String(), c.Valid(), w)
		}
	}
	if combat.ContainmentLeaveCause(0).Valid() || combat.ContainmentLeaveCause(9).Valid() || combat.ContainmentLeaveCause(9).String() != "unknown" {
		t.Error("only the eight causes are valid")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-040, FR-042, FR-045 - the answers hold whatever the ids sort as: an outsider sorting before or after the target and its container, a creature held by another container, the occupants of each of two containers, and a container that holds nobody while another holds creatures.
// Path: per case.
func TestContainmentQueriesAcrossSeveralContainers(t *testing.T) {
	t.Parallel()
	var rules combat.ContainmentRules
	cube, frog := ctTerms(t, "cube"), ctTerms(t, "frog")
	set := ctSet(t,
		ctRecord(t, "m", "p", content.SizeMedium, cube, 1), ctRecord(t, "m", "q", content.SizeMedium, cube, 2),
		ctRecord(t, "f", "r", content.SizeSmall, frog, 1))
	for _, source := range []string{"a", "n", "z", "f", "r"} {
		if got := rules.CoverBetween(set, gid(source), gid("p")); got != combat.ContainmentCoverTotal {
			t.Errorf("CoverBetween(%s, p) = %v, want Total for a creature outside the cube", source, got)
		}
	}
	for _, source := range []string{"m", "q", "p"} {
		if got := rules.CoverBetween(set, gid(source), gid("p")); got != combat.ContainmentCoverNone {
			t.Errorf("CoverBetween(%s, p) = %v, want None (the container, a fellow occupant, the creature itself)", source, got)
		}
	}
	if got := rules.CoverBetween(set, gid("q"), gid("r")); got != combat.ContainmentCoverTotal {
		t.Errorf("CoverBetween(q, r) = %v, want Total: q is held by another container", got)
	}
	for container, want := range map[string][]combat.CombatantID{"m": {gid("p"), gid("q")}, "f": {gid("r")}, "a": {}, "z": {}} {
		if got := rules.MovesWith(set, gid(container)); !slices.Equal(got, want) || !slices.Equal(rules.CarriedFree(set, gid(container)), want) {
			t.Errorf("MovesWith(%s) = %v, want %v", container, got, want)
		}
	}
	for container, want := range map[string]bool{"f": true, "a": false, "z": false, "m": false} {
		if got := rules.IsActionLocked(set, gid(container), "Bite"); got != want {
			t.Errorf("IsActionLocked(%s, Bite) = %v, want %v", container, got, want)
		}
	}
	if rules.RestrictionsOn(set, gid("r")).Cover != combat.ContainmentCoverTotal || rules.RestrictionsOn(set, gid("r")).NoVerbalSpells {
		t.Error("restrictions come from the occupant's own container")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-056, FR-079 - a regurgitation distance places only a regurgitated creature within it: every other cause of a Purple Worm's meal is the nearest unoccupied space.
// Path: per case.
func TestPlacementByCause(t *testing.T) {
	t.Parallel()
	rec := ctRecord(t, "w", "o", content.SizeMedium, ctTerms(t, "worm"), 1)
	held := condSet(t, nil)
	for c := combat.ContainmentLeaveCauseEscaped; c <= combat.ContainmentLeaveCauseOccupantRemoved; c++ {
		got := (combat.ContainmentRules{}).ReleaseOnSet(held, rec, c).Placement()
		switch c {
		case combat.ContainmentLeaveCauseRegurgitated:
			if got != (combat.LandingWithin{Feet: 5}) {
				t.Errorf("%v placement = %v, want within 5 feet", c, got)
			}
		case combat.ContainmentLeaveCauseOccupantRemoved:
			if got != nil {
				t.Errorf("%v placement = %v, want none", c, got)
			}
		case combat.ContainmentLeaveCauseEscaped, combat.ContainmentLeaveCausePulledOut, combat.ContainmentLeaveCauseDisgorged, combat.ContainmentLeaveCauseContainerDied,
			combat.ContainmentLeaveCauseEscapedCorpse, combat.ContainmentLeaveCauseGrappleEnded:
			if got != (combat.NearestUnoccupiedSpace{}) {
				t.Errorf("%v placement = %v, want the nearest unoccupied space", c, got)
			}
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-054, FR-057 - an exit never ends a grapple the container did not hold: a creature engulfed by a cube and also grappled by another monster keeps that grapple and its Grappled condition.
// Path: unhappy (rule interaction).
func TestExitLeavesAnotherMonstersGrappleAlone(t *testing.T) {
	t.Parallel()
	in := ctExitInput(t, "cube", ctSelf(rollOf(0)))
	other := held("other", "p", hand(0))
	in.Grapples = grapplesOf(t, other)
	in.OccupantConditions = condSet(t, []combat.Condition{combat.Restrained{}, other.Condition()})
	res, _ := ctExit(t, in, 20)
	out := res.(combat.ExitContainerExited)
	if out.Grapples().Len() != 1 || !slices.Contains(out.Release().Conditions().Effective(), content.ConditionGrappled) {
		t.Errorf("grapples %d conditions %v, want the other monster's grapple kept", out.Grapples().Len(), out.Release().Conditions().Effective())
	}
}
