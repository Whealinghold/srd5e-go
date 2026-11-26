package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func ctRelease(tb testing.TB, in combat.ReleaseOccupantsInput) combat.ReleaseOccupantsResult {
	tb.Helper()
	cmd, err := combat.NewReleaseOccupantsCommand(in)
	if err != nil {
		tb.Fatalf("NewReleaseOccupantsCommand: %v", err)
	}
	res, err := combat.NewReleaseOccupants().Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res
}

func ctReleased(tb testing.TB, res combat.ReleaseOccupantsResult) combat.ReleaseOccupantsReleased {
	tb.Helper()
	r, ok := res.(combat.ReleaseOccupantsReleased)
	if !ok {
		tb.Fatalf("result = %#v, want released", res)
	}
	return r
}

// ctDied is the container "m" of the fixture holding "p" (entered by the real rules) when it dies.
func ctDied(tb testing.TB, name string) combat.ReleaseOccupantsInput {
	tb.Helper()
	s, g, c := ctHeld(tb, name)
	return combat.ReleaseOccupantsInput{Containments: s, Grapples: g, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): c},
		Trigger: combat.ContainerDied{Container: gid("m")}}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-057, Q-4, Q-11 - a dead container with ReleaseOnDeath releases every occupant (container died, no Prone), and the grapple of an Engulfed-grappled creature ends first as released, leaving no Grappled condition or record.
// Path: per case.
func TestReleaseOccupantsContainerDiedReleases(t *testing.T) {
	t.Parallel()
	cube := ctReleased(t, ctRelease(t, ctDied(t, "cube")))
	rel := cube.Releases()
	if len(rel) != 1 || rel[0].Cause() != combat.ContainmentLeaveCauseContainerDied || cube.Containments().Len() != 0 || len(rel[0].Conditions().Effective()) != 0 || !rel[0].RestoreBreathing() ||
		len(cube.Trapped()) != 0 || cube.MovementSpentFeet() != 0 {
		t.Fatalf("cube death = %+v", cube)
	}
	if _, hasProne := rel[0].Prone(); hasProne {
		t.Error("a creature released by a dead container is not Prone (Q-4)")
	}
	for _, name := range []string{"mound", "whelm"} {
		in := ctDied(t, name)
		out := ctReleased(t, ctRelease(t, in))
		r := out.Releases()[0]
		if out.Grapples().Len() != 0 || out.Containments().Len() != 0 || len(r.Conditions().Effective()) != 0 {
			t.Errorf("%s death: grapples %d conditions %v, want none left", name, out.Grapples().Len(), r.Conditions().Effective())
		}
		kinds := ctKinds(out.Events())
		if kinds[0] != "condition_removed" || kinds[1] != "grapple_ended" || kinds[len(kinds)-1] != "occupant_released" {
			t.Errorf("%s events = %v, want the end of the grapple first", name, kinds)
		}
		if end := out.Events()[1].(combat.GrappleEnded); end.Cause() != combat.GrappleEndCauseReleased {
			t.Errorf("%s grapple end = %v, want released", name, end.Cause())
		}
		if in.Grapples.Len() != 1 {
			t.Error("the input grapples must not change")
		}
	}
	empty := ctReleased(t, ctRelease(t, combat.ReleaseOccupantsInput{Trigger: combat.ContainerDied{Container: gid("m")}}))
	if len(empty.Releases()) != 0 || len(empty.Events()) != 0 {
		t.Error("a container that held nobody releases nobody")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-057, FR-058, Q-11, table 4.7.1 - a dead container with EscapeFromCorpse traps each occupant for the rule's feet, removes only the Restrained the containment added, and keeps Blinded and Total Cover; the creature climbs out Prone when its movement is at least the feet (equal allowed), one short is refused.
// Path: per case.
func TestReleaseOccupantsCorpseEscape(t *testing.T) {
	t.Parallel()
	in := ctDied(t, "behir")
	died := ctReleased(t, ctRelease(t, in))
	tr := died.Trapped()
	if len(tr) != 1 || tr[0].Feet() != 15 || tr[0].Occupant != gid("p") || len(died.Releases()) != 0 ||
		!slices.Equal(tr[0].Conditions().Effective(), []content.ConditionKind{content.ConditionBlinded}) {
		t.Fatalf("death = trapped %+v", tr)
	}
	if !slices.Equal(ctKinds(died.Events()), []string{"condition_removed", "container_trapped"}) || !slices.Equal(ctKinds(tr[0].Events()), []string{"condition_removed", "container_trapped"}) {
		t.Errorf("events = %v", ctKinds(died.Events()))
	}
	trapped := died.Containments().ContainerOf(gid("p")).(combat.InContainer).Record()
	if !trapped.Phase().IsTrapped() || trapped.Phase().TrappedFeet() != 15 || !slices.Equal(trapped.AddedConditions(), []content.ConditionKind{content.ConditionBlinded}) {
		t.Errorf("record = %+v", trapped)
	}
	if (combat.ContainmentRules{}).CoverBetween(died.Containments(), gid("out"), gid("p")) != combat.ContainmentCoverTotal {
		t.Error("Total Cover stays until the creature escapes (Q-11)")
	}
	if ev := died.Events()[1].(combat.ContainerTrapped); ev.Feet() != 15 {
		t.Errorf("ContainerTrapped feet = %d", ev.Feet())
	}
	// The Kraken's only imposed condition is Restrained, so nothing is left.
	if k := ctReleased(t, ctRelease(t, ctDied(t, "kraken"))).Trapped()[0]; len(k.Conditions().Effective()) != 0 || k.Feet() != 15 {
		t.Errorf("Kraken trapped = %v feet %d", k.Conditions().Effective(), k.Feet())
	}
	// A second death report leaves a trapped creature trapped.
	again := in
	again.Containments, again.OccupantConditions = died.Containments(), nil
	if second := ctReleased(t, ctRelease(t, again)); !second.Containments().Equal(died.Containments()) || len(second.Trapped()) != 0 {
		t.Error("a trapped record is not trapped again")
	}

	// Escaping the corpse.
	conds := tr[0].Conditions()
	escape := func(movement int) combat.ReleaseOccupantsResult {
		return ctRelease(t, combat.ReleaseOccupantsInput{Containments: died.Containments(), OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): conds},
			Trigger: combat.EscapedCorpse{Occupant: gid("p"), MovementRemainingFeet: movement}})
	}
	if res := escape(14); res != (combat.ReleaseOccupantsRejected{Rejection: combat.ContainmentRejectionInsufficientMovement}) {
		t.Errorf("14 feet of 15 = %#v", res)
	}
	out := ctReleased(t, escape(15))
	rel := out.Releases()[0]
	if out.MovementSpentFeet() != 15 || rel.Cause() != combat.ContainmentLeaveCauseEscapedCorpse || out.Containments().Len() != 0 ||
		!slices.Equal(rel.Conditions().Effective(), []content.ConditionKind{content.ConditionProne}) {
		t.Errorf("escape = spent %d cause %v conditions %v, want 15, escaped corpse, Prone", out.MovementSpentFeet(), rel.Cause(), rel.Conditions().Effective())
	}
	if ctReleased(t, escape(40)).MovementSpentFeet() != 15 {
		t.Error("only the climb is spent, not all the movement")
	}
	if res := escape(-5); res != (combat.ReleaseOccupantsRejected{Rejection: combat.ContainmentRejectionInsufficientMovement}) {
		t.Errorf("negative movement = %#v", res)
	}
	living, _, c := ctHeld(t, "behir")
	for name, want := range map[string]combat.ContainmentRejection{"p": combat.ContainmentRejectionNotTrapped, "stranger": combat.ContainmentRejectionNotContained} {
		res := ctRelease(t, combat.ReleaseOccupantsInput{Containments: living, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid(name): c},
			Trigger: combat.EscapedCorpse{Occupant: gid(name), MovementRemainingFeet: 99}})
		if res != (combat.ReleaseOccupantsRejected{Rejection: want}) {
			t.Errorf("%s: %#v, want %v", name, res, want)
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-059 - a grapple review releases (grapple ended) every Engulfed-grappled creature its container no longer holds, removing only the imposed conditions; a creature still held, a swallowed one and one grappled by another monster are left.
// Path: per case.
func TestReleaseOccupantsGrappleReview(t *testing.T) {
	t.Parallel()
	whelm := ctEnterInput(t, "whelm")
	whelm.OccupantSize = content.SizeSmall
	first := ctContained(t, mustEnter(t, whelm, 1))
	whelm.Occupant, whelm.Containments, whelm.Grapples = gid("q"), first.Containments(), first.Grapples()
	second := ctContained(t, mustEnter(t, whelm, 1))
	set, grapples := second.Containments(), second.Grapples()

	// q's grapple ended elsewhere; its Grappled condition is already gone.
	review := grapplesOf(t, grapples.GrappleOf(gid("p")).(combat.InGrapple).Grapple())
	qConds := condSet(t, []combat.Condition{combat.Restrained{}})
	res := ctReleased(t, ctRelease(t, combat.ReleaseOccupantsInput{Containments: set, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("q"): qConds},
		Trigger: combat.GrappleReview{Grapples: review}}))
	rels := res.Releases()
	if len(rels) != 1 || rels[0].Occupant != gid("q") || rels[0].Cause() != combat.ContainmentLeaveCauseGrappleEnded || res.Containments().Len() != 1 || len(rels[0].Conditions().Effective()) != 0 {
		t.Fatalf("review = %+v", res)
	}
	if !slices.Equal(ctKinds(res.Events()), []string{"condition_removed", "occupant_released"}) {
		t.Errorf("events = %v, want no grapple event (the grapple is already gone)", ctKinds(res.Events()))
	}
	if _, hasProne := rels[0].Prone(); hasProne {
		t.Error("a grapple that ended leaves the creature standing")
	}
	// Everything still held: nobody is released. A grapple by another monster does not hold it for this one.
	if kept := ctReleased(t, ctRelease(t, combat.ReleaseOccupantsInput{Containments: set, Trigger: combat.GrappleReview{Grapples: grapples}})); len(kept.Releases()) != 0 {
		t.Error("nothing to release while the container holds both")
	}
	other := grapplesOf(t, held("other", "p", hand(0)), grapples.GrappleOf(gid("q")).(combat.InGrapple).Grapple())
	res = ctReleased(t, ctRelease(t, combat.ReleaseOccupantsInput{Containments: set, OccupantConditions: map[combat.CombatantID]combat.ConditionSet{gid("p"): second.Conditions()},
		Trigger: combat.GrappleReview{Grapples: other}}))
	if len(res.Releases()) != 1 || res.Releases()[0].Occupant != gid("p") {
		t.Errorf("a grapple by another monster = %+v, want p released", res.Releases())
	}
	swallowed := ctHolding(t, "worm", "m", "s")
	if got := ctReleased(t, ctRelease(t, combat.ReleaseOccupantsInput{Containments: swallowed, Trigger: combat.GrappleReview{}})); len(got.Releases()) != 0 {
		t.Error("a swallowed creature is not bound by a grapple")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-060, FR-061, FR-063 - a creature the host removed is dropped with cause occupant removed and no condition touched; releasing one that is not held is refused as not contained; the map of conditions must hold exactly the creatures whose conditions change.
// Path: per case.
func TestReleaseOccupantsGoneAndConditionMap(t *testing.T) {
	t.Parallel()
	set, _, c := ctHeld(t, "cube")
	gone := ctReleased(t, ctRelease(t, combat.ReleaseOccupantsInput{Containments: set, Trigger: combat.OccupantGone{Occupant: gid("p")}}))
	if len(gone.Releases()) != 1 || gone.Releases()[0].Cause() != combat.ContainmentLeaveCauseOccupantRemoved || gone.Containments().Len() != 0 || gone.Releases()[0].Placement() != nil ||
		gone.Releases()[0].Conditions() != (combat.ConditionSet{}) {
		t.Errorf("occupant gone = %+v", gone)
	}
	if res := ctRelease(t, combat.ReleaseOccupantsInput{Containments: set, Trigger: combat.OccupantGone{Occupant: gid("nobody")}}); res != (combat.ReleaseOccupantsRejected{Rejection: combat.ContainmentRejectionNotContained}) {
		t.Errorf("a creature not held = %#v", res)
	}
	extra := map[combat.CombatantID]combat.ConditionSet{gid("p"): c}
	if res := ctRelease(t, combat.ReleaseOccupantsInput{Containments: set, OccupantConditions: extra, Trigger: combat.OccupantGone{Occupant: gid("p")}}); res != (combat.ReleaseOccupantsInvalid{Problem: combat.ContainmentProblemUnknownOccupant}) {
		t.Errorf("conditions for a removed creature = %#v", res)
	}
	for name, c := range map[string]struct {
		have map[combat.CombatantID]combat.ConditionSet
		want combat.ContainmentProblem
	}{
		"missing": {nil, combat.ContainmentProblemMissingOccupantConditions},
		"extra":   {map[combat.CombatantID]combat.ConditionSet{gid("p"): c, gid("zz"): {}}, combat.ContainmentProblemUnknownOccupant},
	} {
		in := combat.ReleaseOccupantsInput{Containments: set, OccupantConditions: c.have, Trigger: combat.ContainerDied{Container: gid("m")}}
		if res := ctRelease(t, in); res != (combat.ReleaseOccupantsInvalid{Problem: c.want}) {
			t.Errorf("%s: %#v, want invalid %v", name, res, c.want)
		}
	}
	for name, in := range map[string]combat.ReleaseOccupantsInput{
		"no trigger": {}, "blank container": {Trigger: combat.ContainerDied{}}, "blank occupant": {Trigger: combat.OccupantGone{}},
		"blank escaped": {Trigger: combat.EscapedCorpse{}}, "pointer trigger": {Trigger: &combat.OccupantGone{Occupant: gid("p")}}} {
		if _, err := combat.NewReleaseOccupantsCommand(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	// The command copies the map; a run twice is deterministic.
	have := map[combat.CombatantID]combat.ConditionSet{gid("p"): c}
	cmd, _ := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{Containments: set, OccupantConditions: have, Trigger: combat.ContainerDied{Container: gid("m")}})
	delete(have, gid("p"))
	a, errA := combat.NewReleaseOccupants().Run(context.Background(), cmd)
	b, errB := combat.NewReleaseOccupants().Run(context.Background(), cmd)
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) || reflect.TypeOf(a) != reflect.TypeOf(combat.ReleaseOccupantsReleased{}) {
		t.Errorf("results %#v / %#v (%v %v)", a, b, errA, errB)
	}
}
