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
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func rollWith(mod int, circumstances ...modifier.RollCircumstance) d20.RollAttempt {
	return d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(mod)), Circumstances: circumstances}
}

// Requirement: per case
// Path: per case.
func TestResolveFallingOffSave(t *testing.T) {
	t.Parallel()
	spec := "doc/specs/mounted-and-underwater-combat.md "
	moved, riderProne, mountProne := combat.FallingOffTriggerMountMovedAgainstItsWill, combat.FallingOffTriggerRiderKnockedProne, combat.FallingOffTriggerMountKnockedProne
	cases := []struct {
		name, requirement, path string
		trigger                 combat.FallingOffTrigger
		attempt                 d20.SaveAttempt
		faces                   []int
		fell                    bool
		total                   int
		dice                    int // d20s requested: 0 for a forced failure
	}{
		{"total equals the DC", spec + "FR-028 FR-030 table 4.4a", "unhappy (boundary)", moved, rollWith(2), []int{8}, false, 10, 1},
		{"one below the DC", spec + "FR-028 FR-031 table 4.4a", "unhappy (boundary)", moved, rollWith(2), []int{7}, true, 9, 1},
		{"rider knocked Prone, success", spec + "FR-029 FR-030 table 4.4a", "happy", riderProne, rollWith(0), []int{15}, false, 15, 1},
		{"mount knocked Prone, failure", spec + "FR-029 FR-031 table 4.4a", "unhappy (state error)", mountProne, rollWith(0), []int{3}, true, 3, 1},
		{"natural 20 is only a 20", spec + "FR-028 table 4.4a", "unhappy (boundary)", moved, rollWith(-11), []int{20}, true, 9, 1},
		{"natural 1 is only a 1", spec + "FR-028 table 4.4a", "unhappy (boundary)", moved, rollWith(9), []int{1}, false, 10, 1},
		{"Advantage takes the higher", spec + "FR-028 FR-072 table 4.4a", "unhappy (rule interaction)", moved, rollWith(0, modifier.Advantage{}), []int{4, 14}, false, 14, 2},
		{"Disadvantage takes the lower", spec + "FR-028 FR-072 table 4.4a", "unhappy (rule interaction)", moved, rollWith(0, modifier.Disadvantage{}), []int{14, 4}, true, 4, 2},
		{"Advantage and Disadvantage cancel", spec + "FR-028 FR-072", "unhappy (rule interaction)", moved, rollWith(0, modifier.Advantage{}, modifier.Disadvantage{}), []int{4}, true, 4, 1},
		{"chosen failure", spec + "FR-034 table 4.4a", "unhappy (rule interaction)", moved, d20.ChooseToFail{}, nil, true, 0, 0},
		{"automatic failure", spec + "FR-034 table 4.4a", "unhappy (rule interaction)", moved, d20.AutomaticFailure{}, nil, true, 0, 0},
		{"negative modifier, minimum die", spec + "FR-028 table 4.4a", "unhappy (boundary)", moved, rollWith(-5), []int{1}, true, -4, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			status := mntMounted(t, combat.MountControlControlled)
			got, err := mntFallWith(t, roller, status, c.trigger, c.attempt, combat.LandingSpaceUnoccupiedWithin5Feet)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			wantRequests := []string{}
			if c.dice > 0 {
				wantRequests = append(wantRequests, dice.MustExpression(c.dice, dice.D20(), 0).String())
			}
			var requests []string
			for _, r := range roller.Requests() {
				requests = append(requests, r.String())
			}
			if !slices.Equal(slices.Clone(append([]string{}, requests...)), wantRequests) {
				t.Errorf("dice requested = %v, want %v [FR-072]", requests, wantRequests)
			}
			if !c.fell {
				s, ok := got.(combat.ResolveFallingOffStayedMounted)
				if !ok || s.RiderStatus() != status || s.Save().Outcome() != d20.Success || s.Save().(combat.FallingOffSaveRolled).Result().Total() != c.total {
					t.Errorf("result = %#v, want stayed mounted with total %d and the status unchanged [%s]", got, c.total, c.requirement)
				}
				return
			}
			f, ok := got.(combat.ResolveFallingOffFellOff)
			if !ok || f.RiderStatus() != combat.NotMounted() || f.Save().Outcome() != d20.Failure || len(f.Events()) != 1 {
				t.Fatalf("result = %#v, want fell off, NotMounted, one event [%s]", got, c.requirement)
			}
			if r, ok := f.Save().(combat.FallingOffSaveRolled); ok && r.Result().Total() != c.total {
				t.Errorf("total = %d, want %d", r.Result().Total(), c.total)
			}
			ev := f.Events()[0].(combat.FellOffMount)
			if ev.Trigger() != c.trigger || ev.Control() != combat.MountControlControlled || ev.Landing() != combat.LandingSpaceUnoccupiedWithin5Feet {
				t.Errorf("event = %#v, want trigger %v, controlled, space within 5 feet [FR-031]", ev, c.trigger)
			}
			combattest.CheckEvent(t, ev, "fell_off_mount", gid("rider"), gid("horse"))
		})
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-034 - a ChooseToFail resolves with ForcedFailureReason ChosenToFail and an AutomaticFailure with Automatic, both for a saving throw and with no die.
// Path: unhappy (rule interaction).
func TestResolveFallingOffForcedFailureReasons(t *testing.T) {
	t.Parallel()
	for attempt, want := range map[d20.SaveAttempt]d20.ForcedFailureReason{d20.ChooseToFail{}: d20.ChosenToFail, d20.AutomaticFailure{}: d20.Automatic} {
		res, err := mntFall(t, combat.FallingOffTriggerMountMovedAgainstItsWill, attempt)
		f, ok := res.(combat.ResolveFallingOffFellOff)
		if err != nil || !ok {
			t.Fatalf("%T: %#v, %v", attempt, res, err)
		}
		forced := f.Save().(combat.FallingOffSaveForced).Result()
		if forced.Reason() != want || forced.Kind() != d20.SavingThrow {
			t.Errorf("%T: forced = %+v, want a saving throw failing for %v", attempt, forced, want)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-028 - the attempt's RollModifiers reach the resolver unchanged, and the Difficulty Class is 10 whatever the trigger.
// Path: unhappy (rule interaction).
func TestResolveFallingOffUsesTheModifiersAndOneDifficultyClass(t *testing.T) {
	t.Parallel()
	attempt := d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(3)).WithProficiencyBonus(modifier.New(2)).WithOthers(modifier.New(1))}
	for _, trigger := range []combat.FallingOffTrigger{combat.FallingOffTriggerMountMovedAgainstItsWill, combat.FallingOffTriggerRiderKnockedProne, combat.FallingOffTriggerMountKnockedProne} {
		res, err := mntFall(t, trigger, attempt, 10)
		s, ok := res.(combat.ResolveFallingOffStayedMounted)
		if err != nil || !ok {
			t.Fatalf("%v: %#v, %v", trigger, res, err)
		}
		check := s.Save().(combat.FallingOffSaveRolled).Result()
		if !check.Modifiers().Equal(attempt.Modifiers) || check.Total() != 16 || check.Target() != combat.FallingOffDifficultyClass || check.Kind() != d20.SavingThrow {
			t.Errorf("%v: check = %+v, want total 16 against DC 10 with the supplied modifiers", trigger, check)
		}
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-031, FR-033 - the control of an independent ride and the landing space the host reported (none) are carried unchanged into FellOffMount, and the save is still rolled.
// Path: unhappy (state error).
func TestResolveFallingOffCarriesControlAndLanding(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(2)
	res, err := mntFallWith(t, roller, mntMounted(t, combat.MountControlIndependent), combat.FallingOffTriggerMountMovedAgainstItsWill, rollWith(0), combat.LandingSpaceNoUnoccupiedSpaceReported)
	f, ok := res.(combat.ResolveFallingOffFellOff)
	if err != nil || !ok || len(roller.Requests()) != 1 {
		t.Fatalf("Run = %#v, %v with %d requests, want fell off after one roll", res, err, len(roller.Requests()))
	}
	ev := f.Events()[0].(combat.FellOffMount)
	if ev.Control() != combat.MountControlIndependent || ev.Landing() != combat.LandingSpaceNoUnoccupiedSpaceReported {
		t.Errorf("event = %#v, want independent and no space reported", ev)
	}
	if ev.Condition() != content.ConditionProne {
		t.Errorf("condition = %v, want Prone [FR-032]", ev.Condition())
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-035 - a rider who is not mounted is refused with MountingRiderNotMountedRejection, with no die rolled and no event.
// Path: unhappy (state error).
func TestResolveFallingOffNotMounted(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(1)
	res, err := mntFallWith(t, roller, combat.NotMounted(), combat.FallingOffTriggerRiderKnockedProne, rollWith(0), combat.LandingSpaceUnoccupiedWithin5Feet)
	r, ok := res.(combat.ResolveFallingOffRejected)
	if err != nil || !ok || r.Reason() != (combat.MountingRiderNotMountedRejection{}) || len(roller.Requests()) != 0 {
		t.Errorf("Run = %#v, %v with %d requests, want refused rider_not_mounted with no die", res, err, len(roller.Requests()))
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-037 - a roller that fails, panics or answers another expression gives a *srderr.PortError and no result.
// Path: unhappy (collaborator failure).
func TestResolveFallingOffPortFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := mntFallWith(t, roller, mntMounted(t, combat.MountControlControlled), combat.FallingOffTriggerMountMovedAgainstItsWill, rollWith(0, modifier.Advantage{}), combat.LandingSpaceUnoccupiedWithin5Feet)
		envPortFailure(t, err)
		if res != nil {
			t.Errorf("%s: result = %#v, want nil", name, res)
		}
	}
	if _, err := mntFallWith(t, envFailingRoller(cause), mntMounted(t, combat.MountControlControlled), combat.FallingOffTriggerMountMovedAgainstItsWill, rollWith(0), combat.LandingSpaceUnoccupiedWithin5Feet); !errors.Is(err, cause) {
		t.Errorf("error = %v, want the cause kept", err)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-067, FR-052 - NewResolveFallingOffCommand rejects a blank rider, a rider that is the mount of its own MountedOn, an unknown trigger or landing space and an attempt of another type (nil included) with ErrInvalidInput; the circumstances are copied.
// Path: unhappy (invalid input).
func TestNewResolveFallingOffCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	selfMount, _ := combat.MountedOn(gid("rider"), combat.MountControlControlled)
	horse := mntMounted(t, combat.MountControlControlled)
	trig, land, ok := combat.FallingOffTriggerRiderKnockedProne, combat.LandingSpaceUnoccupiedWithin5Feet, d20.SaveAttempt(d20.ChooseToFail{})
	for name, err := range map[string]error{
		"rider is its own mount": second(combat.NewResolveFallingOffCommand(gid("rider"), selfMount, trig, ok, land)),
		"blank rider":            second(combat.NewResolveFallingOffCommand("", horse, trig, ok, land)),
		"trigger 0":              second(combat.NewResolveFallingOffCommand(gid("rider"), horse, 0, ok, land)),
		"landing 0":              second(combat.NewResolveFallingOffCommand(gid("rider"), horse, trig, ok, 0)),
		"nil attempt":            second(combat.NewResolveFallingOffCommand(gid("rider"), horse, trig, nil, land)),
		"pointer attempt":        second(combat.NewResolveFallingOffCommand(gid("rider"), horse, trig, &d20.RollAttempt{}, land)),
	} {
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	circumstances := []modifier.RollCircumstance{modifier.Disadvantage{}}
	cmd, err := combat.NewResolveFallingOffCommand(gid("rider"), horse, trig, d20.RollAttempt{Circumstances: circumstances}, land)
	if err != nil {
		t.Fatal(err)
	}
	circumstances[0] = modifier.Advantage{}
	uc, _ := combat.NewResolveFallingOff(dicetest.NewFixedRoller(4, 14))
	res, _ := uc.Run(context.Background(), cmd)
	if _, fell := res.(combat.ResolveFallingOffFellOff); !fell {
		t.Errorf("result = %#v, want Disadvantage kept (4 fails) after the caller changed its slice [FR-051]", res)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-071 - the constructor takes a dice.Roller; a nil roller is ErrInvalidInput, as are the zero use case and the zero command; a cancelled ctx is returned unwrapped before any die is rolled.
// Path: unhappy (invalid input).
func TestResolveFallingOffGuards(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewResolveFallingOff(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller error = %v, want ErrInvalidInput", err)
	}
	cmd, err := combat.NewResolveFallingOffCommand(gid("rider"), mntMounted(t, combat.MountControlControlled), combat.FallingOffTriggerRiderKnockedProne, d20.ChooseToFail{}, combat.LandingSpaceUnoccupiedWithin5Feet)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (combat.ResolveFallingOff{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case error = %v, want ErrInvalidInput", err)
	}
	uc, _ := combat.NewResolveFallingOff(dicetest.NewFixedRoller(5))
	if _, err := uc.Run(context.Background(), combat.ResolveFallingOffCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command error = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uc.Run(ctx, cmd); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled error = %v, want context.Canceled", err)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md FR-051, FR-055 - the events of a result are a copy; mutating the returned slice changes nothing held.
// Path: unhappy (immutability).
func TestResolveFallingOffEventsAreACopy(t *testing.T) {
	t.Parallel()
	res, _ := mntFall(t, combat.FallingOffTriggerMountMovedAgainstItsWill, d20.AutomaticFailure{})
	f := res.(combat.ResolveFallingOffFellOff)
	events := f.Events()
	events[0] = nil
	if f.Events()[0] == nil || len(f.Events()) != 1 {
		t.Error("Events must return a copy")
	}
}
