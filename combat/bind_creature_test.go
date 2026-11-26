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

// Requirement: per case
// Path: per case.
func TestBindCreatureRun(t *testing.T) {
	t.Parallel()
	cuffs, rope := combat.BindingKindManacles, combat.BindingKindRope
	cases := []struct {
		name, requirement, path string
		change                  func(*combat.BindCreatureInput)
		faces                   []int
		want                    string // bound, failed or a rejection
		escapeDC                d20.DifficultyClass
		dice                    []string
	}{
		{"chain, total equals the DC", gearSpec + "FR-047 FR-054", "unhappy (boundary)", func(*combat.BindCreatureInput) {}, []int{13}, "bound", 18, []string{"1d20"}},
		{"chain, one below the DC", gearSpec + "FR-047 FR-054", "unhappy (boundary)", func(*combat.BindCreatureInput) {}, []int{12}, "failed", 0, []string{"1d20"}},
		{"manacles bind", gearSpec + "FR-047", "happy", func(in *combat.BindCreatureInput) { in.Kind, in.HoldID = cuffs, "cuffs" }, []int{13}, "bound", 20, []string{"1d20"}},
		{"rope, DC 10", gearSpec + "FR-047", "unhappy (boundary)", func(in *combat.BindCreatureInput) { in.Kind, in.HoldID = rope, "rope" }, []int{10}, "bound", 15, []string{"1d20"}},
		{"rope, a modifier", gearSpec + "FR-047", "happy", func(in *combat.BindCreatureInput) { in.Kind, in.Attempt = rope, rollWith(4) }, []int{6}, "bound", 15, []string{"1d20"}},
		{"rope, one below", gearSpec + "FR-047", "unhappy (boundary)", func(in *combat.BindCreatureInput) { in.Kind = rope }, []int{9}, "failed", 0, []string{"1d20"}},
		{"advantage rolls two dice", gearSpec + "FR-054", "unhappy (rule interaction)", func(in *combat.BindCreatureInput) { in.Attempt = rollWith(0, modifier.Advantage{}) }, []int{2, 19}, "bound", 18, []string{"2d20"}},
		{"automatic failure draws no die", gearSpec + "FR-057", "unhappy (rule interaction)", func(in *combat.BindCreatureInput) { in.Attempt = d20.AutomaticFailure{} }, nil, "failed", 0, []string{}},
		{"willing target", gearSpec + "FR-045", "unhappy (rule interaction)", func(in *combat.BindCreatureInput) { in.Willingness = combat.WillingnessWilling }, nil, "target_willing", 0, []string{}},
		{"chain at 6 feet", gearSpec + "FR-049", "unhappy (boundary)", func(in *combat.BindCreatureInput) { in.DistanceFeet = 6 }, nil, "out_of_reach", 0, []string{}},
		{"manacles on a Large creature", gearSpec + "FR-048", "unhappy (boundary)", func(in *combat.BindCreatureInput) { in.Kind, in.TargetSize = cuffs, content.SizeLarge }, nil, "size_not_allowed", 0, []string{}},
		{"target not held down", gearSpec + "FR-046", "unhappy (state error)", func(in *combat.BindCreatureInput) { in.TargetConditions = combat.ConditionSet{} }, nil, "target_not_held_down", 0, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := bindIn(t)
			c.change(&in)
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := bindRun(t, in, roller)
			if err != nil {
				t.Fatalf("Run: %v [%s]", err, c.requirement)
			}
			if !slices.Equal(requested(roller), c.dice) {
				t.Errorf("dice requested = %v, want %v [%s]", requested(roller), c.dice, c.requirement)
			}
			switch c.want {
			case "bound":
				b, ok := res.(combat.BindCreatureBound)
				if !ok {
					t.Fatalf("result = %#v, want bound [%s]", res, c.requirement)
				}
				hold, found := b.Holds().HoldOf(in.HoldID)
				if !found || hold.Kind() != in.Kind.HoldKind() || hold.EscapeDifficultyClass() != c.escapeDC || hold.Restraint() != combat.HoldRestraintRestrains || !hold.IsOn(gid("victim")) ||
					!b.RestraintBegins() || b.Test().Outcome() != d20.Success {
					t.Errorf("hold = %+v, want a restraining %v hold with escape DC %d", hold, in.Kind, c.escapeDC)
				}
				combattest.CheckEvent(t, b.Events()[0], "gear_hold_started", gid("victim"))
			case "failed":
				if f, ok := res.(combat.BindCreatureFailed); !ok || f.Test().Outcome() != d20.Failure {
					t.Errorf("result = %#v, want failed [%s]", res, c.requirement)
				}
			default:
				if r, ok := res.(combat.BindCreatureRejected); !ok || r.Reason().String() != c.want {
					t.Errorf("result = %#v, want rejected %s [%s]", res, c.want, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-040 - the restraint of a new hold comes from the command: a Chain or Rope restrains only when the host states the legs are bound, Manacles only when attached to a fixed chain or hook; the holds in the input are unchanged (SRD 5.2.1, "Chain", "Manacles", "Rope").
// Path: unhappy (rule interaction, immutability).
func TestBindCreatureRestraintComesFromTheCommand(t *testing.T) {
	t.Parallel()
	existing := gearHolds(t, gearHold(t, holdIn("net", combat.HoldKindNet)))
	for _, r := range []combat.HoldRestraint{combat.HoldRestraintRestrains, combat.HoldRestraintDoesNotRestrain} {
		for _, k := range []combat.BindingKind{combat.BindingKindChain, combat.BindingKindManacles, combat.BindingKindRope} {
			in := bindIn(t)
			in.Kind, in.Restraint = k, r
			res, err := bindRun(t, in, dicetest.NewFixedRoller(20))
			b := res.(combat.BindCreatureBound)
			hold, _ := b.Holds().HoldOf(in.HoldID)
			if err != nil || hold.Restraint() != r || b.RestraintBegins() != (r == combat.HoldRestraintRestrains) || b.Holds().RestrainsTarget(gid("victim")) != (r == combat.HoldRestraintRestrains) {
				t.Errorf("%v with %v: restraint %v, begins %v", k, r, hold.Restraint(), b.RestraintBegins())
			}
		}
	}
	in := bindIn(t)
	in.Holds = existing
	res, _ := bindRun(t, in, dicetest.NewFixedRoller(20))
	if b := res.(combat.BindCreatureBound); b.RestraintBegins() || b.Holds().Len() != 2 || existing.Len() != 1 {
		t.Error("a second restraint must not begin and the input holds must be unchanged")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-061, FR-071 - a failing, panicking or wrong roller propagates a *srderr.PortError with no result; the constructor returns a GearError for a blank hold id or zero binder or target (blank_id), a negative distance, a kind, size, willingness or restraint that is not valid or an attempt that is nil or ChooseToFail (invalid_value); NewBindCreature rejects a nil roller; Run returns ErrInvalidInput for the zero command or use case and ctx.Err() for a cancelled ctx.
// Path: unhappy (failure of collaborator, invalid input, cancellation).
func TestBindCreatureFailureAndInvalidInput(t *testing.T) {
	t.Parallel()
	for name, roller := range map[string]combat.Roller{"fails": envFailingRoller(errors.New("down")), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := bindRun(t, withBindAdvantage(bindIn(t)), roller)
		envPortFailure(t, err)
		if res != nil {
			t.Errorf("%s: result = %#v, want none", name, res)
		}
	}
	for name, c := range map[string]struct {
		change func(*combat.BindCreatureInput)
		want   combat.GearProblem
	}{
		"blank hold id":     {func(in *combat.BindCreatureInput) { in.HoldID = "" }, combat.GearProblemBlankID},
		"zero binder":       {func(in *combat.BindCreatureInput) { in.Binder = "" }, combat.GearProblemBlankID},
		"zero target":       {func(in *combat.BindCreatureInput) { in.Target = "" }, combat.GearProblemBlankID},
		"negative distance": {func(in *combat.BindCreatureInput) { in.DistanceFeet = -1 }, combat.GearProblemNegativeDistance},
		"invalid kind":      {func(in *combat.BindCreatureInput) { in.Kind = 0 }, combat.GearProblemInvalidValue},
		"invalid size":      {func(in *combat.BindCreatureInput) { in.TargetSize = 0 }, combat.GearProblemInvalidValue},
		"invalid willing":   {func(in *combat.BindCreatureInput) { in.Willingness = 0 }, combat.GearProblemInvalidValue},
		"invalid restraint": {func(in *combat.BindCreatureInput) { in.Restraint = 0 }, combat.GearProblemInvalidValue},
		"nil attempt":       {func(in *combat.BindCreatureInput) { in.Attempt = nil }, combat.GearProblemInvalidValue},
	} {
		in := bindIn(t)
		c.change(&in)
		if _, err := combat.NewBindCreatureCommand(in); !errors.Is(err, &combat.GearError{Problem: c.want}) {
			t.Errorf("%s: err = %v, want a %v GearError", name, err, c.want)
		}
	}
	if _, err := combat.NewBindCreature(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v, want ErrInvalidInput", err)
	}
	roller := dicetest.NewFixedRoller(20)
	use, _ := combat.NewBindCreature(roller)
	if _, err := use.Run(context.Background(), combat.BindCreatureCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	cmd, _ := combat.NewBindCreatureCommand(bindIn(t))
	if _, err := (combat.BindCreature{}).Run(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: err = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := use.Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil || len(roller.Requests()) != 0 {
		t.Errorf("cancelled ctx: %v, %v; want context.Canceled and no roll", res, err)
	}
	circumstances := []modifier.RollCircumstance{modifier.Advantage{}}
	in := bindIn(t)
	in.Attempt = d20.RollAttempt{Circumstances: circumstances}
	cmd, _ = combat.NewBindCreatureCommand(in)
	circumstances[0] = modifier.Disadvantage{}
	pair := dicetest.NewFixedRoller(1, 20)
	copied, _ := combat.NewBindCreature(pair)
	if res, err := copied.Run(context.Background(), cmd); err != nil || !slices.Equal(requested(pair), []string{"2d20"}) {
		t.Errorf("Run = %v, %v, dice %v; want a command that copied its advantage (2d20)", res, err, requested(pair))
	} else if _, ok := res.(combat.BindCreatureBound); !ok {
		t.Errorf("result = %#v, want bound with the higher die", res)
	}
}
