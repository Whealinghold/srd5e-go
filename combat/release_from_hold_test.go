package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// relHolds is a set with the hold "h" of kind on "victim", the Manacles loose and the others as holdIn makes them.
func relHolds(tb testing.TB, kind combat.HoldKind) combat.GearHolds {
	tb.Helper()
	in := holdIn("h", kind)
	if kind == combat.HoldKindManacles || kind == combat.HoldKindRope {
		in.Restraint = combat.HoldRestraintDoesNotRestrain
	}
	return gearHolds(tb, gearHold(tb, in))
}

// Requirement: per case
// Path: per case.
func TestReleaseFromHoldRun(t *testing.T) {
	t.Parallel()
	esc, burst, pick, key := combat.HoldReleaseMethodEscape, combat.HoldReleaseMethodBurst, combat.HoldReleaseMethodPickLock, combat.HoldReleaseMethodKey
	cases := []struct {
		name, requirement, path string
		kind                    combat.HoldKind
		method                  combat.HoldReleaseMethod
		change                  func(*combat.ReleaseFromHoldInput)
		faces                   []int
		want                    string // released, held or a rejection
		cause                   combat.HoldEndCause
		spent                   bool
		dice                    []string
		damage                  bool
	}{
		{"net escape, total equals the DC", gearSpec + "FR-041 FR-054", "unhappy (boundary)", combat.HoldKindNet, esc, nil, []int{10}, "released", combat.HoldEndCauseEscaped, true, []string{"1d20"}, false},
		{"net escape, one below the DC", gearSpec + "FR-041 FR-056", "unhappy (boundary)", combat.HoldKindNet, esc, nil, []int{9}, "held", 0, true, []string{"1d20"}, false},
		{"net escape by a helper within reach", gearSpec + "FR-041", "happy", combat.HoldKindNet, esc, func(in *combat.ReleaseFromHoldInput) { in.Actor = combat.ActorIsHelper{WithinReach: true} }, []int{10}, "released", combat.HoldEndCauseEscaped, true, []string{"1d20"}, false},
		{"net escape by a helper beyond 5 feet", gearSpec + "FR-041", "unhappy (boundary)", combat.HoldKindNet, esc, func(in *combat.ReleaseFromHoldInput) { in.Actor = combat.ActorIsHelper{} }, nil, "helper_out_of_reach", 0, false, []string{}, false},
		{"chain escape", gearSpec + "FR-051", "happy", combat.HoldKindChain, esc, nil, []int{18}, "released", combat.HoldEndCauseEscaped, true, []string{"1d20"}, false},
		{"chain burst", gearSpec + "FR-051", "happy", combat.HoldKindChain, burst, nil, []int{20}, "released", combat.HoldEndCauseBurst, true, []string{"1d20"}, false},
		{"chain burst one below", gearSpec + "FR-051", "unhappy (boundary)", combat.HoldKindChain, burst, nil, []int{19}, "held", 0, true, []string{"1d20"}, false},
		{"manacles burst", gearSpec + "FR-051", "happy", combat.HoldKindManacles, burst, func(in *combat.ReleaseFromHoldInput) { in.Attempt = rollWith(5) }, []int{20}, "released", combat.HoldEndCauseBurst, true, []string{"1d20"}, false},
		{"rope escape", gearSpec + "FR-051", "happy", combat.HoldKindRope, esc, nil, []int{15}, "released", combat.HoldEndCauseEscaped, true, []string{"1d20"}, false},
		{"rope burst costs an action", gearSpec + "FR-052 FR-058", "unhappy (rule interaction)", combat.HoldKindRope, burst, nil, []int{20}, "released", combat.HoldEndCauseBurst, true, []string{"1d20"}, false},
		{"pick the manacles' lock", gearSpec + "FR-055", "happy", combat.HoldKindManacles, pick, func(in *combat.ReleaseFromHoldInput) { in.HasThievesTools = true }, []int{15}, "released", combat.HoldEndCauseUnlocked, true, []string{"1d20"}, false},
		{"picking a lock without tools", gearSpec + "FR-055", "unhappy (state error)", combat.HoldKindManacles, pick, nil, nil, "tools_missing", 0, false, []string{}, false},
		{"the key takes no check and no action", gearSpec + "FR-055 FR-058", "happy", combat.HoldKindManacles, key, nil, nil, "released", combat.HoldEndCauseUnlocked, false, []string{}, false},
		{"the key by a helper", gearSpec + "FR-055", "unhappy (rule interaction)", combat.HoldKindManacles, key, func(in *combat.ReleaseFromHoldInput) { in.Actor = combat.ActorIsHelper{} }, nil, "released", combat.HoldEndCauseUnlocked, false, []string{}, false},
		{"no action, no die", gearSpec + "FR-053", "unhappy (state error)", combat.HoldKindChain, esc, func(in *combat.ReleaseFromHoldInput) { in.Actions = combat.ConditionActionAvailabilityNone }, nil, "no_action", 0, false, []string{}, false},
		{"no action, not even the key", gearSpec + "FR-053", "unhappy (state error)", combat.HoldKindManacles, key, func(in *combat.ReleaseFromHoldInput) { in.Actions = combat.ConditionActionAvailabilityNone }, nil, "no_action", 0, false, []string{}, false},
		{"a helper cannot burst a chain", gearSpec + "FR-052", "unhappy (rule interaction)", combat.HoldKindChain, burst, func(in *combat.ReleaseFromHoldInput) { in.Actor = combat.ActorIsHelper{WithinReach: true} }, nil, "helper_not_allowed", 0, false, []string{}, false},
		{"a net cannot be burst", gearSpec + "FR-050", "unhappy (invalid input)", combat.HoldKindNet, burst, nil, nil, "method_not_available", 0, false, []string{}, false},
		{"unknown hold", gearSpec + "FR-029", "unhappy (state error)", combat.HoldKindNet, esc, func(in *combat.ReleaseFromHoldInput) { in.HoldID = "zz" }, nil, "unknown_hold", 0, false, []string{}, false},
		{"trap escape", gearSpec + "FR-051", "happy", combat.HoldKindHuntingTrap, esc, nil, []int{13}, "released", combat.HoldEndCauseEscaped, true, []string{"1d20"}, false},
		{"trap escape fails and deals 1 Piercing", gearSpec + "FR-056", "unhappy (boundary)", combat.HoldKindHuntingTrap, esc, nil, []int{12}, "held", 0, true, []string{"1d20"}, true},
		{"trap escape forced failure still deals damage", gearSpec + "FR-057", "unhappy (rule interaction)", combat.HoldKindHuntingTrap, esc, func(in *combat.ReleaseFromHoldInput) { in.Attempt = d20.AutomaticFailure{} }, nil, "held", 0, true, []string{}, true},
		{"chain forced failure deals no damage", gearSpec + "FR-056 FR-057", "unhappy (rule interaction)", combat.HoldKindChain, esc, func(in *combat.ReleaseFromHoldInput) { in.Attempt = d20.AutomaticFailure{} }, nil, "held", 0, true, []string{}, false},
		{"trap freed by a helper within reach", gearSpec + "FR-052", "happy", combat.HoldKindHuntingTrap, esc, func(in *combat.ReleaseFromHoldInput) { in.Actor = combat.ActorIsHelper{WithinReach: true} }, []int{13}, "released", combat.HoldEndCauseEscaped, true, []string{"1d20"}, false},
		{"advantage rolls two dice", gearSpec + "FR-054", "unhappy (rule interaction)", combat.HoldKindNet, esc, func(in *combat.ReleaseFromHoldInput) { in.Attempt = rollWith(0, modifier.Advantage{}) }, []int{2, 10}, "released", combat.HoldEndCauseEscaped, true, []string{"2d20"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			holds := relHolds(t, c.kind)
			in := releaseIn(holds)
			in.Method = c.method
			if c.change != nil {
				c.change(&in)
			}
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := releaseRun(t, in, roller)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if !slices.Equal(requested(roller), c.dice) {
				t.Errorf("dice requested = %v, want %v [%s]", requested(roller), c.dice, c.requirement)
			}
			switch c.want {
			case "released":
				r, ok := res.(combat.ReleaseFromHoldReleased)
				if !ok {
					t.Fatalf("result = %#v, want released [%s]", res, c.requirement)
				}
				ev := r.Events()
				if _, still := r.Holds().HoldOf("h"); still || r.Holds().Len() != 0 || len(ev) != 1 || ev[0].(combat.GearHoldEnded).Cause() != c.cause || r.ActionSpent() != c.spent {
					t.Errorf("result = %#v, want the hold ended as %v, action spent %v", r, c.cause, c.spent)
				}
				combattest.CheckEvent(t, ev[0], "gear_hold_ended", gid("victim"))
				if _, rolled := r.Test(); rolled != (len(c.dice) > 0 || c.faces != nil || c.method != key) {
					t.Errorf("Test ok = %v, want %v", rolled, c.method != key)
				}
				if r.RestraintEnds() != (holds.RestrainsTarget(gid("victim"))) {
					t.Errorf("RestraintEnds = %v for a hold that restrains = %v", r.RestraintEnds(), holds.RestrainsTarget(gid("victim")))
				}
			case "held":
				h, ok := res.(combat.ReleaseFromHoldHeld)
				if !ok {
					t.Fatalf("result = %#v, want held [%s]", res, c.requirement)
				}
				d, hasDamage := h.Damage()
				if h.Test().Outcome() != d20.Failure || !h.ActionSpent() || hasDamage != c.damage || (c.damage && d != combat.MustDamage(1, content.DamagePiercing)) {
					t.Errorf("held = %+v, damage %v %v; want action spent and damage %v", h, d, hasDamage, c.damage)
				}
			default:
				if r, ok := res.(combat.ReleaseFromHoldRejected); !ok || r.Reason().String() != c.want {
					t.Errorf("result = %#v, want rejected %s [%s]", res, c.want, c.requirement)
				}
			}
			if holds.Len() != 1 {
				t.Error("Run must not change the holds it was given")
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-029, FR-042 - releasing the only restraining hold reports RestraintEnds; releasing a hold beside another restraining one does not (SRD 5.2.1, "Restrained", p. 187).
// Path: unhappy (rule interaction).
func TestReleaseFromHoldRestraintEnds(t *testing.T) {
	t.Parallel()
	holds := gearHolds(t, gearHold(t, holdIn("h", combat.HoldKindNet)), gearHold(t, holdIn("chain", combat.HoldKindChain)))
	res, err := releaseRun(t, releaseIn(holds), dicetest.NewFixedRoller(20))
	if r := res.(combat.ReleaseFromHoldReleased); err != nil || r.RestraintEnds() || r.Holds().Len() != 1 {
		t.Errorf("result = %#v, %v; want the Net gone and the restraint kept", res, err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-061, FR-071 - a failing, panicking or wrong roller propagates a *srderr.PortError with no result; the constructor returns a GearError for a blank hold id (blank_id), a method, actor or actions that is not valid or an attempt that is nil or ChooseToFail (invalid_value); NewReleaseFromHold rejects a nil roller; Run returns ErrInvalidInput for the zero command or use case and ctx.Err() for a cancelled ctx.
// Path: unhappy (failure of collaborator, invalid input, cancellation).
func TestReleaseFromHoldFailureAndInvalidInput(t *testing.T) {
	t.Parallel()
	adv := releaseIn(relHolds(t, combat.HoldKindNet))
	adv.Attempt = rollWith(0, modifier.Advantage{})
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := releaseRun(t, adv, roller)
		envPortFailure(t, err)
		if res != nil {
			t.Errorf("%s: result = %#v, want none", name, res)
		}
	}
	base := releaseIn(relHolds(t, combat.HoldKindNet))
	for name, c := range map[string]struct {
		change func(*combat.ReleaseFromHoldInput)
		want   combat.GearProblem
	}{
		"blank hold id":   {func(in *combat.ReleaseFromHoldInput) { in.HoldID = "" }, combat.GearProblemBlankID},
		"invalid method":  {func(in *combat.ReleaseFromHoldInput) { in.Method = 0 }, combat.GearProblemInvalidValue},
		"nil actor":       {func(in *combat.ReleaseFromHoldInput) { in.Actor = nil }, combat.GearProblemInvalidValue},
		"invalid actions": {func(in *combat.ReleaseFromHoldInput) { in.Actions = 0 }, combat.GearProblemInvalidValue},
		"nil attempt":     {func(in *combat.ReleaseFromHoldInput) { in.Attempt = nil }, combat.GearProblemInvalidValue},
	} {
		in := base
		c.change(&in)
		if _, err := combat.NewReleaseFromHoldCommand(in); !errors.Is(err, &combat.GearError{Problem: c.want}) {
			t.Errorf("%s: err = %v, want a %v GearError", name, err, c.want)
		}
	}
	if _, err := combat.NewReleaseFromHold(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v, want ErrInvalidInput", err)
	}
	roller := dicetest.NewFixedRoller(20)
	use, _ := combat.NewReleaseFromHold(roller)
	if _, err := use.Run(context.Background(), combat.ReleaseFromHoldCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	cmd, _ := combat.NewReleaseFromHoldCommand(base)
	if _, err := (combat.ReleaseFromHold{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: err = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil || len(roller.Requests()) != 0 {
		t.Errorf("cancelled ctx: %v, %v; want context.Canceled and no roll", res, err)
	}
	circumstances := []modifier.RollCircumstance{modifier.Advantage{}}
	in := base
	in.Attempt = d20.RollAttempt{Circumstances: circumstances}
	cmd, _ = combat.NewReleaseFromHoldCommand(in)
	circumstances[0] = modifier.Disadvantage{}
	pair := dicetest.NewFixedRoller(1, 20)
	copied, _ := combat.NewReleaseFromHold(pair)
	if res, err := copied.Run(context.Background(), cmd); err != nil || !slices.Equal(requested(pair), []string{"2d20"}) {
		t.Errorf("Run = %v, %v, dice %v; want a command that copied its advantage (2d20)", res, err, requested(pair))
	}
}
