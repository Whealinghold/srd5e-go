package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// holdOn is a valid hold of kind with the id on who; a Hunting Trap names the trap.
func holdOn(tb testing.TB, id string, kind combat.HoldKind, who, trap string) combat.GearHold {
	tb.Helper()
	in := holdIn(id, kind)
	in.Target = gid(who)
	if kind == combat.HoldKindHuntingTrap {
		in.GearID = gearID(trap)
	}
	return gearHold(tb, in)
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-026, FR-034 - IsOn, IsDeployedBy and IsCombatant match exactly one creature, not every creature that sorts before or after it.
// Path: unhappy (boundary).
func TestGearIdentityPredicatesMatchOnlyTheNamedCreature(t *testing.T) {
	t.Parallel()
	hold := holdOn(t, "h", combat.HoldKindChain, "m", "")
	gear := gearOf(t, gearOne(t, combat.DeployedGearKindCaltrops))
	stamp := gearStamp(1, "m")
	for _, other := range []string{"a", "z"} {
		if hold.IsOn(gid(other)) || stamp.IsCombatant(gid(other)) || gear.IsDeployedBy(gid(other)) {
			t.Errorf("%q matched a hold on m, a turn of m or gear of owner", other)
		}
	}
	if !hold.IsOn(gid("m")) || !stamp.IsCombatant(gid("m")) || !gear.IsDeployedBy(gid("owner")) {
		t.Error("the named creature must match")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-027, FR-034 - HoldOf and Find answer for a hold or gear that is not first in the set, and Equal is false when the other set has more members.
// Path: happy; unhappy (boundary).
func TestGearSetsLookUpAndCompareBeyondTheFirstMember(t *testing.T) {
	t.Parallel()
	first, second := holdOn(t, "h1", combat.HoldKindChain, "a", ""), holdOn(t, "h2", combat.HoldKindRope, "b", "")
	both, one := gearHolds(t, first, second), gearHolds(t, first)
	if got, ok := both.HoldOf(hid("h2")); !ok || got != second {
		t.Errorf("HoldOf(h2) = %+v, %v; want the second hold", got, ok)
	}
	if _, ok := both.HoldOf(hid("none")); ok {
		t.Error("HoldOf an unknown id must be false")
	}
	if one.Equal(both) || both.Equal(one) {
		t.Error("a set and a larger set must not be equal")
	}

	gears := gearDeploy(t, gearOne(t, combat.DeployedGearKindCaltrops), "g2", combat.DeployedGearKindBallBearings)
	if got, ok := gears.Find(gearID("g2")); !ok || got.ID() != "g2" {
		t.Errorf("Find(g2) = %+v, %v; want the second gear", got, ok)
	}
	if _, ok := gears.Find(gearID("none")); ok {
		t.Error("Find an unknown id must be false")
	}
	small := gearOne(t, combat.DeployedGearKindCaltrops)
	if small.Equal(gears) || gears.Equal(small) {
		t.Error("a set and a larger set must not be equal")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-031, FR-036 - RestrainsTarget and AttackRollCircumstances read only holds on the target, and the circumstance only from Manacles.
// Path: unhappy (boundary).
func TestGearHoldsReadOnlyHoldsOnTheTarget(t *testing.T) {
	t.Parallel()
	around := func(kind combat.HoldKind) combat.GearHolds {
		return gearHolds(t, holdOn(t, "lo", combat.HoldKindManacles, "a", ""), holdOn(t, "hi", combat.HoldKindManacles, "z", ""), holdOn(t, "mid", kind, "m", "trap"))
	}
	if gearHolds(t, holdOn(t, "lo", combat.HoldKindChain, "a", ""), holdOn(t, "hi", combat.HoldKindChain, "z", "")).RestrainsTarget(gid("m")) {
		t.Error("holds on other creatures must not restrain m")
	}
	for _, kind := range []combat.HoldKind{combat.HoldKindNet, combat.HoldKindChain, combat.HoldKindRope, combat.HoldKindHuntingTrap} {
		if got := around(kind).AttackRollCircumstances(gid("m")); len(got) != 0 {
			t.Errorf("%v on m with Manacles on others gave %v, want none", kind, got)
		}
	}
	got := around(combat.HoldKindManacles).AttackRollCircumstances(gid("m"))
	if len(got) != 1 || got[0] != (modifier.Disadvantage{}) {
		t.Errorf("Manacles on m gave %v, want Disadvantage", got)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-031, FR-035 - Ending a hold, or reanchoring Manacles, reports the end of restraint only when no other hold on that creature restrains, wherever that hold sits in the set.
// Path: unhappy (boundary).
func TestGearRestraintEndsOnlyWhenNoOtherHoldOnTheTargetRestrains(t *testing.T) {
	t.Parallel()
	chain, rope := holdOn(t, "chain", combat.HoldKindChain, "m", ""), holdOn(t, "rope", combat.HoldKindRope, "m", "")
	set := gearHolds(t, chain, rope)
	for _, id := range []string{"chain", "rope"} {
		tr, err := set.End(hid(id), combat.HoldEndCauseEscaped)
		if holdsApplied(t, tr, err).RestraintEnds() {
			t.Errorf("ending %s while the other restrains must not end restraint", id)
		}
	}
	bystanders := gearHolds(t, holdOn(t, "lo", combat.HoldKindChain, "a", ""), chain, holdOn(t, "hi", combat.HoldKindChain, "z", ""))
	tr, err := bystanders.End(hid("chain"), combat.HoldEndCauseEscaped)
	if !holdsApplied(t, tr, err).RestraintEnds() {
		t.Error("holds on other creatures must not keep m restrained")
	}

	cuffs := holdOn(t, "cuffs", combat.HoldKindManacles, "m", "")
	for name, holds := range map[string]combat.GearHolds{
		"after":     gearHolds(t, cuffs, chain),
		"before":    gearHolds(t, chain, cuffs),
		"bystander": gearHolds(t, holdOn(t, "lo", combat.HoldKindChain, "a", ""), cuffs, holdOn(t, "hi", combat.HoldKindChain, "z", "")),
	} {
		tr, err := holds.Reanchor(hid("cuffs"), combat.HoldRestraintDoesNotRestrain)
		got := holdsApplied(t, tr, err)
		if want := name == "bystander"; got.RestraintEnds() != want {
			t.Errorf("%s: reanchor RestraintEnds = %v, want %v", name, got.RestraintEnds(), want)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-035 - Reanchor is available for Manacles only: a Rope or Hunting Trap hold is refused with methodNotAvailable.
// Path: unhappy (state error).
func TestGearReanchorRefusesRopeAndHuntingTrap(t *testing.T) {
	t.Parallel()
	for _, kind := range []combat.HoldKind{combat.HoldKindNet, combat.HoldKindChain, combat.HoldKindRope, combat.HoldKindHuntingTrap} {
		holds := gearHolds(t, holdOn(t, "h", kind, "m", "trap"))
		tr, err := holds.Reanchor(hid("h"), combat.HoldRestraintDoesNotRestrain)
		if r, ok := tr.(combat.GearHoldsRejected); err != nil || !ok || r.Reason() != combat.GearRejectionMethodNotAvailable {
			t.Errorf("%v: Reanchor = %#v, %v; want methodNotAvailable", kind, tr, err)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-006, FR-010, FR-012, FR-015 - Deploy, Recover, Spring and Reset find a gear that is not first in the set, and Deploy refuses an id in use by one.
// Path: happy; unhappy (state error).
func TestGearOperationsReachGearBeyondTheFirst(t *testing.T) {
	t.Parallel()
	traps := gearDeploy(t, gearOne(t, combat.DeployedGearKindHuntingTrap), "g2", combat.DeployedGearKindHuntingTrap)
	tr, err := traps.Deploy(gearID("g2"), combat.DeployedGearKindCaltrops, gid("owner"), 0, combat.GroundLevelLevel)
	if r, ok := tr.(combat.DeployedGearsRejected); err != nil || !ok || r.Reason() != combat.GearRejectionIDInUse {
		t.Errorf("Deploy of an id in use = %#v, %v; want idInUse", tr, err)
	}
	sprung := gearsApplied(t, mustSpring(t, traps, "g2"), nil).Gears()
	if got := gearsApplied(t, sprung.Reset(gearID("g2"), combat.GearHolds{}), nil).Gears(); got.Len() != 2 {
		t.Errorf("Reset of the second gear left %d gears, want 2", got.Len())
	}
	caltrops := gearDeploy(t, gearOne(t, combat.DeployedGearKindCaltrops), "g2", combat.DeployedGearKindCaltrops)
	if got := gearsApplied(t, caltrops.Recover(gearID("g2")), nil).Gears(); got.Len() != 1 {
		t.Errorf("Recover of the second gear left %d gears, want 1", got.Len())
	}
}

func mustSpring(tb testing.TB, set combat.DeployedGears, id string) combat.DeployedGearsTransition {
	tb.Helper()
	tr, err := set.Spring(gearID(id), gid("walker"))
	if err != nil {
		tb.Fatal(err)
	}
	return tr
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-015 - Reset is refused only while a hold of that very trap exists: holds of other Hunting Traps, whichever way their ids sort, do not occupy it.
// Path: unhappy (boundary).
func TestGearResetIgnoresHoldsOfOtherTraps(t *testing.T) {
	t.Parallel()
	holds := gearHolds(t, holdOn(t, "lo", combat.HoldKindHuntingTrap, "a", "a"), holdOn(t, "hi", combat.HoldKindHuntingTrap, "b", "z"), holdOn(t, "chain", combat.HoldKindChain, "c", ""))
	if _, ok := gearSprung(t).Reset(gearID("g"), holds).(combat.DeployedGearsApplied); !ok {
		t.Error("Reset must succeed when only other traps hold creatures")
	}
	own := gearHolds(t, holdOn(t, "own", combat.HoldKindHuntingTrap, "a", "g"))
	if r, ok := gearSprung(t).Reset(gearID("g"), own).(combat.DeployedGearsRejected); !ok || r.Reason() != combat.GearRejectionTrapOccupied {
		t.Errorf("Reset of an occupied trap = %#v, want trapOccupied", r)
	}
}
