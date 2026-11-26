package combat_test

import (
	"context"
	"errors"
	"math"
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

// Requirement: per case
// Path: per case.
func TestThrowNetRun(t *testing.T) {
	t.Parallel()
	existing := gearHolds(t, gearHold(t, holdIn("old", combat.HoldKindNet)))
	cases := []struct {
		name, requirement, path string
		change                  func(*combat.ThrowNetInput)
		faces                   []int
		want                    string // held, saved, auto or a rejection
		dice                    []string
	}{
		{"failed save holds the target", gearSpec + "FR-037", "happy", func(*combat.ThrowNetInput) {}, []int{12}, "held", []string{"1d20"}},
		{"total equal to the DC saves", gearSpec + "FR-037 FR-024", "unhappy (boundary)", func(*combat.ThrowNetInput) {}, []int{13}, "saved", []string{"1d20"}},
		{"a modifier saves", gearSpec + "FR-037", "happy", func(in *combat.ThrowNetInput) { in.Attempt = rollWith(3) }, []int{10}, "saved", []string{"1d20"}},
		{"disadvantage keeps the lower die", gearSpec + "FR-037", "unhappy (rule interaction)", func(in *combat.ThrowNetInput) { in.Attempt = rollWith(0, modifier.Disadvantage{}) }, []int{18, 4}, "held", []string{"2d20"}},
		{"chosen failure draws no die", gearSpec + "FR-037", "unhappy (rule interaction)", func(in *combat.ThrowNetInput) { in.Attempt = d20.ChooseToFail{} }, nil, "held", []string{}},
		{"large target saves", gearSpec + "FR-038", "unhappy (boundary)", func(in *combat.ThrowNetInput) { in.TargetSize = content.SizeLarge }, []int{1}, "held", []string{"1d20"}},
		{"huge target succeeds with no die", gearSpec + "FR-038", "unhappy (boundary)", func(in *combat.ThrowNetInput) { in.TargetSize = content.SizeHuge }, nil, "auto", []string{}},
		{"gargantuan target ignores its attempt", gearSpec + "FR-038", "unhappy (boundary)", func(in *combat.ThrowNetInput) {
			in.TargetSize, in.Attempt = content.SizeGargantuan, d20.AutomaticFailure{}
		}, nil, "auto", []string{}},
		{"15 feet is in reach", gearSpec + "FR-035", "unhappy (boundary)", func(in *combat.ThrowNetInput) { in.DistanceFeet = 15 }, []int{1}, "held", []string{"1d20"}},
		{"16 feet is out of reach", gearSpec + "FR-035", "unhappy (boundary)", func(in *combat.ThrowNetInput) { in.DistanceFeet = 16 }, nil, "out_of_reach", []string{}},
		{"target not seen", gearSpec + "FR-035", "unhappy (rule interaction)", func(in *combat.ThrowNetInput) { in.Sight, in.DistanceFeet = combat.ThrowerSightCannotSeeTarget, 99 }, nil, "target_not_seen", []string{}},
		{"target already held by a net", gearSpec + "FR-039", "unhappy (state error)", func(in *combat.ThrowNetInput) { in.Holds, in.HoldID = existing, "new" }, nil, "target_already_held", []string{}},
		{"already held, huge target", gearSpec + "FR-039", "unhappy (rule interaction)", func(in *combat.ThrowNetInput) { in.Holds, in.HoldID, in.TargetSize = existing, "new", content.SizeHuge }, nil, "target_already_held", []string{}},
		{"hold id in use", gearSpec + "FR-027", "unhappy (state error)", func(in *combat.ThrowNetInput) {
			in.Holds, in.HoldID, in.Target = existing, "old", "somebody"
		}, nil, "id_in_use", []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := netIn()
			c.change(&in)
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := netRun(t, in, roller)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if !slices.Equal(requested(roller), c.dice) {
				t.Errorf("dice requested = %v, want %v [%s]", requested(roller), c.dice, c.requirement)
			}
			switch c.want {
			case "held":
				h, ok := res.(combat.ThrowNetHeld)
				if !ok {
					t.Fatalf("result = %#v, want held [%s]", res, c.requirement)
				}
				hold, found := h.Holds().HoldOf(in.HoldID)
				if !found || hold.Kind() != combat.HoldKindNet || hold.Restraint() != combat.HoldRestraintRestrains || !hold.IsOn(gid("victim")) || hold.EscapeDifficultyClass() != 10 ||
					hold.Status() != (combat.NetProfile{}).Status() || !h.RestraintBegins() || h.Test().Outcome() != d20.Failure {
					t.Errorf("hold = %+v, begins %v; want a restraining Net with escape DC 10", hold, h.RestraintBegins())
				}
				combattest.CheckEvent(t, h.Events()[0], "gear_hold_started", gid("victim"))
			case "saved":
				if s, ok := res.(combat.ThrowNetSaved); !ok || s.Test().Outcome() != d20.Success {
					t.Errorf("result = %#v, want saved [%s]", res, c.requirement)
				}
			case "auto":
				if _, ok := res.(combat.ThrowNetAutoSucceeded); !ok {
					t.Errorf("result = %#v, want auto succeeded [%s]", res, c.requirement)
				}
			default:
				if r, ok := res.(combat.ThrowNetRejected); !ok || r.Reason().String() != c.want {
					t.Errorf("result = %#v, want rejected %s [%s]", res, c.want, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-033, FR-036 - the Net's escape Difficulty Class is the profile's 10 unless the command overrides it, and any integer is held as given; the save DC is the command's, with no cap or floor.
// Path: unhappy (boundary).
func TestThrowNetEscapeOverrideAndSaveDC(t *testing.T) {
	t.Parallel()
	for _, dc := range []d20.DifficultyClass{0, -3, 15, math.MaxInt, math.MinInt} {
		in := netIn()
		in.EscapeDCOverride = combat.DCOverrideOf(dc)
		res, err := netRun(t, in, dicetest.NewFixedRoller(1))
		hold, _ := res.(combat.ThrowNetHeld).Holds().HoldOf("net")
		if err != nil || hold.EscapeDifficultyClass() != dc {
			t.Errorf("override %d: escape DC = %d, %v", dc, hold.EscapeDifficultyClass(), err)
		}
	}
	for dc, faces := range map[d20.DifficultyClass]int{math.MinInt: 1, 21: 20, 20: 20, math.MaxInt: 20} {
		in := netIn()
		in.SaveDC = dc
		res, _ := netRun(t, in, dicetest.NewFixedRoller(faces))
		_, saved := res.(combat.ThrowNetSaved)
		if want := dc <= d20.DifficultyClass(faces); saved != want {
			t.Errorf("DC %d, face %d: saved = %v, want %v", dc, faces, saved, want)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-028, FR-061, FR-062 - a Net on a creature a Chain already restrains does not begin a second restraint; a failing, panicking or wrong roller propagates a *srderr.PortError with no result and the holds unchanged; Run does not change its input.
// Path: unhappy (rule interaction, failure of collaborator, immutability).
func TestThrowNetRestraintAndFailure(t *testing.T) {
	t.Parallel()
	chained := gearHolds(t, gearHold(t, holdIn("chain", combat.HoldKindChain)))
	in := netIn()
	in.Holds = chained
	res, err := netRun(t, in, dicetest.NewFixedRoller(1))
	if h := res.(combat.ThrowNetHeld); err != nil || h.RestraintBegins() || h.Holds().Len() != 2 || chained.Len() != 1 {
		t.Errorf("result = %#v, %v; want a second hold without a second restraint and the input unchanged", res, err)
	}
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := netRun(t, withAdvantage(netIn()), roller)
		envPortFailure(t, err)
		if res != nil {
			t.Errorf("%s: result = %#v, want none", name, res)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-059, FR-071 - NewThrowNetCommand returns a GearError for a blank hold id or zero thrower or target (blank_id), a negative distance (negative_distance), a size or sight that is not valid or an attempt that is nil (invalid_value); NewThrowNet rejects a nil roller; Run returns ErrInvalidInput for the zero command or use case and ctx.Err() for a cancelled ctx before any roll.
// Path: unhappy (invalid input, cancellation).
func TestThrowNetInvalidInput(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		change func(*combat.ThrowNetInput)
		want   combat.GearProblem
	}{
		"blank hold id":     {func(in *combat.ThrowNetInput) { in.HoldID = "" }, combat.GearProblemBlankID},
		"zero thrower":      {func(in *combat.ThrowNetInput) { in.Thrower = "" }, combat.GearProblemBlankID},
		"zero target":       {func(in *combat.ThrowNetInput) { in.Target = "" }, combat.GearProblemBlankID},
		"negative distance": {func(in *combat.ThrowNetInput) { in.DistanceFeet = -1 }, combat.GearProblemNegativeDistance},
		"invalid size":      {func(in *combat.ThrowNetInput) { in.TargetSize = 0 }, combat.GearProblemInvalidValue},
		"invalid sight":     {func(in *combat.ThrowNetInput) { in.Sight = 0 }, combat.GearProblemInvalidValue},
		"nil attempt":       {func(in *combat.ThrowNetInput) { in.Attempt = nil }, combat.GearProblemInvalidValue},
	} {
		in := netIn()
		c.change(&in)
		if _, err := combat.NewThrowNetCommand(in); !errors.Is(err, &combat.GearError{Problem: c.want}) {
			t.Errorf("%s: err = %v, want a %v GearError", name, err, c.want)
		}
	}
	if _, err := combat.NewThrowNet(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v, want ErrInvalidInput", err)
	}
	roller := dicetest.NewFixedRoller(1)
	use, _ := combat.NewThrowNet(roller)
	if _, err := use.Run(context.Background(), combat.ThrowNetCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	cmd, _ := combat.NewThrowNetCommand(netIn())
	if _, err := (combat.ThrowNet{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: err = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil || len(roller.Requests()) != 0 {
		t.Errorf("cancelled ctx: %v, %v; want context.Canceled and no roll", res, err)
	}
}
