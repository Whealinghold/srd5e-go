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
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ctHeld is a container "m" of the fixture holding "p" after an entry that added Restrained (and Blinded for a
// fixture that imposes it): the containments, grapples and the occupant's conditions.
func ctHeld(tb testing.TB, name string) (combat.Containments, combat.Grapples, combat.ConditionSet) {
	tb.Helper()
	terms := ctTerms(tb, name)
	res := mustEnter(tb, ctEnterInput(tb, name), 1)
	c := ctContained(tb, res)
	if terms.Mode() == combat.ContainmentModeEngulfedGrappled {
		return c.Containments(), c.Grapples(), c.Conditions()
	}
	return c.Containments(), c.Grapples(), c.Conditions()
}

func ctExitInput(tb testing.TB, name string, method combat.ExitContainerMethod) combat.ExitContainerInput {
	tb.Helper()
	s, g, c := ctHeld(tb, name)
	return combat.ExitContainerInput{Containments: s, Grapples: g, Occupant: gid("p"), OccupantConditions: c, Method: method}
}

func ctExit(tb testing.TB, in combat.ExitContainerInput, faces ...int) (combat.ExitContainerResult, *dicetest.FixedRoller) {
	tb.Helper()
	cmd, err := combat.NewExitContainerCommand(in)
	if err != nil {
		tb.Fatalf("NewExitContainerCommand: %v", err)
	}
	roller := dicetest.NewFixedRoller(faces...)
	uc, err := combat.NewExitContainer(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res, roller
}

func ctSelf(attempt d20.CheckAttempt) combat.ExitBySelf {
	return combat.ExitBySelf{Attempt: attempt, Actions: combat.ConditionActionAvailabilityAvailable}
}

func ctPull(puller string, attempt d20.CheckAttempt) combat.ExitByPull {
	return combat.ExitByPull{Puller: gid(puller), PullerWithin5Feet: true, Attempt: attempt, Actions: combat.ConditionActionAvailabilityAvailable}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-051, FR-055, FR-079 - a Strength check against the self-escape DC releases the creature (escaped, nearest unoccupied space) on a success, removing only the conditions the containment added and restoring breathing, and leaves it held on a failure; both spend the action.
// Path: per case.
func TestExitContainerBySelf(t *testing.T) {
	t.Parallel()
	in := ctExitInput(t, "cube", ctSelf(rollOf(0)))
	res, roller := ctExit(t, in, 12)
	out, ok := res.(combat.ExitContainerExited)
	if !ok || !out.ActionSpent() {
		t.Fatalf("result = %#v, want exited with the action spent", res)
	}
	rel := out.Release()
	if rel.Cause() != combat.ContainmentLeaveCauseEscaped || out.Containments().Len() != 0 || len(rel.Conditions().Effective()) != 0 || !rel.RestoreBreathing() ||
		len(out.DamageDue()) != 0 || reflect.TypeOf(rel.Placement()) != reflect.TypeOf(combat.NearestUnoccupiedSpace{}) {
		t.Errorf("exit = %+v", out)
	}
	if reqs := roller.Requests(); len(reqs) != 1 || reqs[0].Notation() != "1d20" {
		t.Errorf("dice = %v", reqs)
	}
	if rolled := out.Check().(combat.ContainmentTestRolled); rolled.Result().Kind() != d20.AbilityCheck || rolled.Result().Target() != 12 {
		t.Errorf("check = %+v, want an ability check against 12", rolled.Result())
	}
	if !slices.Equal(ctKinds(out.Events()), []string{"condition_removed", "occupant_released"}) {
		t.Errorf("events = %v", ctKinds(out.Events()))
	}
	// One below the DC: held, nothing changes, the action is spent.
	res, _ = ctExit(t, in, 11)
	failed, ok := res.(combat.ExitContainerHeld)
	if !ok || !failed.ActionSpent() || !failed.Containments().Equal(in.Containments) {
		t.Errorf("result = %#v, want held with the set unchanged", res)
	}
	// An automatic failure draws no die.
	forced := ctExitInput(t, "cube", ctSelf(d20.AutomaticFailure{}))
	res, roller = ctExit(t, forced)
	if h, ok := res.(combat.ExitContainerHeld); !ok || len(roller.Requests()) != 0 {
		t.Errorf("automatic failure = %#v with %d dice", res, len(roller.Requests()))
	} else if _, isForced := h.Check().(combat.ContainmentTestForced); !isForced {
		t.Error("the check must be the forced failure")
	}
	// Advantage rolls two dice.
	adv := ctExitInput(t, "cube", ctSelf(rollOf(0, modifier.Advantage{})))
	if res, roller := ctExit(t, adv, 3, 14); reflect.TypeOf(res) != reflect.TypeOf(combat.ExitContainerExited{}) || roller.Requests()[0].Notation() != "2d20" {
		t.Errorf("advantage: %#v %v", res, roller.Requests())
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-052, FR-075 - a pull needs the pull DC and a puller within 5 feet outside the container; a success releases the creature as pulled out and reports the puller's damage due, a failure no damage.
// Path: per case.
func TestExitContainerByPull(t *testing.T) {
	t.Parallel()
	in := ctExitInput(t, "cube", ctPull("pu", rollOf(0)))
	res, _ := ctExit(t, in, 12)
	out := res.(combat.ExitContainerExited)
	due := out.DamageDue()
	if out.Release().Cause() != combat.ContainmentLeaveCausePulledOut || len(due) != 1 || due[0].Target != gid("pu") || due[0].Container != gid("m") ||
		due[0].Cause() != combat.ContainmentDamageCausePull || due[0].Halved() || due[0].Terms()[0].Expression().Notation() != "3d6" {
		t.Errorf("exit = release %v damage %+v", out.Release().Cause(), due)
	}
	res, _ = ctExit(t, in, 11)
	if h, ok := res.(combat.ExitContainerHeld); !ok || !h.ActionSpent() || h.Containments().Len() != 1 {
		t.Errorf("a failed pull = %#v, want held and no damage", res)
	}
	// The Whelm's occupant leaves by a pull and loses its grapple, escaped.
	whelm := ctExitInput(t, "whelm", ctPull("pu", rollOf(0)))
	res, _ = ctExit(t, whelm, 14)
	w := res.(combat.ExitContainerExited)
	if w.Grapples().Len() != 0 || len(w.DamageDue()) != 0 || len(w.Release().Conditions().Effective()) != 0 {
		t.Errorf("Whelm pull = grapples %d conditions %v", w.Grapples().Len(), w.Release().Conditions().Effective())
	}
	if !slices.Equal(ctKinds(w.Events()), []string{"condition_removed", "grapple_ended", "condition_removed", "occupant_released"}) {
		t.Errorf("events = %v, want the end of the grapple first", ctKinds(w.Events()))
	}
	if end := w.Events()[1].(combat.GrappleEnded); end.Cause() != combat.GrappleEndCauseEscaped {
		t.Errorf("grapple end cause = %v, want escaped", end.Cause())
	}
	if !w.Release().RestoreBreathing() {
		t.Error("the Whelm suffocates what it holds unless it breathes water")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-053, FR-054, table 4.7.1 - an exit is refused with no die, in the order not contained, no way out, puller contained, puller out of reach, no action; a swallow has no way out and an Engulfed-grappled creature leaves by its grapple.
// Path: per case.
func TestExitContainerRefusals(t *testing.T) {
	t.Parallel()
	avail, none := combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone
	roll := d20.CheckAttempt(rollOf(0))
	cases := []struct {
		name, path string
		fixture    string
		method     combat.ExitContainerMethod
		mod        func(*combat.ExitContainerInput)
		want       combat.ContainmentRejection
	}{
		{"not held", "unhappy (state error)", "cube", ctSelf(roll), func(in *combat.ExitContainerInput) { in.Occupant = gid("nobody") }, combat.ContainmentRejectionNotContained},
		{"a swallow has no self escape", "unhappy (state error)", "worm", ctSelf(roll), nil, combat.ContainmentRejectionNoWayOut},
		{"a swallow has no pull", "unhappy (state error)", "worm", ctPull("pu", roll), nil, combat.ContainmentRejectionNoWayOut},
		{"the Whelm has no self escape", "unhappy (state error)", "whelm", ctSelf(roll), nil, combat.ContainmentRejectionNoWayOut},
		{"the mound has no exit at all", "unhappy (state error)", "mound", ctPull("pu", roll), nil, combat.ContainmentRejectionNoWayOut},
		{"the cube's pull from inside the cube", "unhappy (state error)", "cube", ctPull("q", roll), func(in *combat.ExitContainerInput) {
			in.Containments = ctHolding(t, "cube", "m", "p", "q")
		}, combat.ContainmentRejectionPullerIsContained},
		{"a creature pulling itself", "unhappy (state error)", "cube", ctPull("p", roll), nil, combat.ContainmentRejectionPullerIsContained},
		{"a puller beyond 5 feet", "unhappy (boundary)", "cube", combat.ExitByPull{Puller: gid("pu"), Attempt: roll, Actions: avail}, nil, combat.ContainmentRejectionPullerOutOfReach},
		{"the held creature cannot act", "unhappy (state error)", "cube", combat.ExitBySelf{Attempt: roll, Actions: none}, nil, combat.ContainmentRejectionNoAction},
		{"the puller cannot act", "unhappy (state error)", "cube", combat.ExitByPull{Puller: gid("pu"), PullerWithin5Feet: true, Attempt: roll, Actions: none}, nil, combat.ContainmentRejectionNoAction},
		{"out of reach before no action", "unhappy (rule interaction)", "cube", combat.ExitByPull{Puller: gid("pu"), Attempt: roll, Actions: none}, nil, combat.ContainmentRejectionPullerOutOfReach},
		{"no way out before a contained puller", "unhappy (rule interaction)", "worm", combat.ExitByPull{Puller: gid("p"), Attempt: roll, Actions: none}, nil, combat.ContainmentRejectionNoWayOut},
		{"a puller of another container", "happy", "cube", ctPull("o", roll), func(in *combat.ExitContainerInput) {
			in.Containments = ctSet(t, in.Containments.All()[0], ctRecord(t, "other", "o", content.SizeSmall, ctTerms(t, "toad"), 1))
		}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := ctExitInput(t, c.fixture, c.method)
			if c.mod != nil {
				c.mod(&in)
			}
			before := in.Containments.All()
			res, roller := ctExit(t, in, 20)
			if c.want == 0 {
				if _, ok := res.(combat.ExitContainerExited); !ok {
					t.Fatalf("result = %#v, want an exit", res)
				}
				return
			}
			rej, ok := res.(combat.ExitContainerRejected)
			if !ok || rej.Rejection != c.want || rej.ActionSpent() || len(roller.Requests()) != 0 || !slices.Equal(in.Containments.All(), before) {
				t.Errorf("result = %#v with %d dice, want %v, no die and the action kept", res, len(roller.Requests()), c.want)
			}
		})
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-055 - a release removes only the kinds the record added: a Restrained another source holds stays.
// Path: unhappy (rule interaction).
func TestExitContainerKeepsAConditionAnotherSourceHolds(t *testing.T) {
	t.Parallel()
	// The creature was already Restrained when it was held, so the containment added none.
	in := ctEnterInput(t, "cube")
	in.OccupantConditions = condSet(t, []combat.Condition{combat.Restrained{}})
	held := ctContained(t, mustEnter(t, in, 1))
	if rec := held.Containments().ContainerOf(gid("p")).(combat.InContainer).Record(); len(rec.AddedConditions()) != 0 {
		t.Fatalf("added = %v", rec.AddedConditions())
	}
	res, _ := ctExit(t, combat.ExitContainerInput{Containments: held.Containments(), Grapples: held.Grapples(), Occupant: gid("p"),
		OccupantConditions: held.Conditions(), Method: ctSelf(rollOf(0))}, 20)
	out := res.(combat.ExitContainerExited)
	if !slices.Equal(out.Release().Conditions().Effective(), []content.ConditionKind{content.ConditionRestrained}) || !slices.Equal(ctKinds(out.Events()), []string{"occupant_released"}) {
		t.Errorf("conditions %v events %v, want Restrained kept and no removal", out.Release().Conditions().Effective(), ctKinds(out.Events()))
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-053, testing.md section 4 - the command constructor rejects a blank occupant, a method that is not self or pull, a pull without a puller, an attempt that is not a roll or an automatic failure and an invalid action availability; the exit is deterministic.
// Path: unhappy (invalid input).
func TestExitContainerInvalidInputAndDeterminism(t *testing.T) {
	t.Parallel()
	good := ctExitInput(t, "cube", ctSelf(rollOf(0)))
	avail := combat.ConditionActionAvailabilityAvailable
	for name, method := range map[string]combat.ExitContainerMethod{
		"nil method":         nil,
		"self, nil attempt":  combat.ExitBySelf{Actions: avail},
		"self, pointer":      combat.ExitBySelf{Attempt: &d20.AutomaticFailure{}, Actions: avail},
		"self, zero actions": combat.ExitBySelf{Attempt: rollOf(0)},
		"self, actions past": combat.ExitBySelf{Attempt: rollOf(0), Actions: 9},
		"pull, no puller":    combat.ExitByPull{Attempt: rollOf(0), Actions: avail},
		"pull, nil attempt":  combat.ExitByPull{Puller: gid("pu"), Actions: avail},
		"pull, zero actions": combat.ExitByPull{Puller: gid("pu"), Attempt: rollOf(0)},
	} {
		in := good
		in.Method = method
		if _, err := combat.NewExitContainerCommand(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	blank := good
	blank.Occupant = ""
	if _, err := combat.NewExitContainerCommand(blank); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank occupant: %v", err)
	}
	attempt := rollOf(0, modifier.Advantage{})
	in := ctExitInput(t, "cube", ctSelf(attempt))
	cmd, _ := combat.NewExitContainerCommand(in)
	attempt.Circumstances[0] = modifier.Disadvantage{}
	uc, _ := combat.NewExitContainer(dicetest.NewFixedRoller(5, 14))
	a, err1 := uc.Run(context.Background(), cmd)
	uc2, _ := combat.NewExitContainer(dicetest.NewFixedRoller(5, 14))
	b, err2 := uc2.Run(context.Background(), cmd)
	if err1 != nil || err2 != nil || !reflect.DeepEqual(a, b) || reflect.TypeOf(a) != reflect.TypeOf(combat.ExitContainerExited{}) {
		t.Errorf("results %#v / %#v (%v %v): the command must copy the attempt and be deterministic", a, b, err1, err2)
	}
}
