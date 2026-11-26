package combat_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func deployRun(tb testing.TB, set combat.DeployedGears, id string, kind combat.DeployedGearKind, distance int, ground combat.GroundLevel, opts ...combat.Option) (combat.DeployGearResult, error) {
	tb.Helper()
	cmd, err := combat.NewDeployGearCommand(set, gearID(id), kind, gid("owner"), distance, ground)
	if err != nil {
		tb.Fatal(err)
	}
	return combat.NewDeployGear(opts...).Run(context.Background(), cmd)
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-005, FR-009, FR-059, FR-060, FR-062 - DeployGear returns DeployGearPlaced with the new set and one DeployedGearPlaced, DeployGearRejected with idInUse, outOfReach or groundNotLevel, and DeployGearInvalid with negative_distance for a negative distance; the input set is unchanged and equal commands give equal results (SRD 5.2.1, pp. 96-98).
// Path: happy; unhappy (state error, invalid input, immutability).
func TestDeployGearRun(t *testing.T) {
	t.Parallel()
	set := gearOne(t, combat.DeployedGearKindCaltrops)
	res, err := deployRun(t, set, "new", combat.DeployedGearKindBallBearings, 10, combat.GroundLevelLevel)
	p, ok := res.(combat.DeployGearPlaced)
	if err != nil || !ok || p.Gears().Len() != 2 || len(p.Events()) != 1 || set.Len() != 1 {
		t.Fatalf("Run = %#v, %v; want placed with two gears and one event", res, err)
	}
	ev := p.Events()
	ev[0] = nil
	if p.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	for name, c := range map[string]struct {
		id       string
		distance int
		ground   combat.GroundLevel
		want     combat.GearRejection
	}{
		"id in use":    {"g", 0, combat.GroundLevelLevel, combat.GearRejectionIDInUse},
		"out of reach": {"n", 11, combat.GroundLevelLevel, combat.GearRejectionOutOfReach},
		"not level":    {"n", 10, combat.GroundLevelNotLevel, combat.GearRejectionGroundNotLevel},
	} {
		res, err := deployRun(t, set, c.id, combat.DeployedGearKindBallBearings, c.distance, c.ground)
		if r, ok := res.(combat.DeployGearRejected); err != nil || !ok || r.Reason() != c.want {
			t.Errorf("%s: Run = %#v, %v; want rejected %v", name, res, err, c.want)
		}
	}
	for _, d := range []int{-1, math.MinInt} {
		res, err := deployRun(t, set, "n", combat.DeployedGearKindBallBearings, d, combat.GroundLevelLevel)
		if i, ok := res.(combat.DeployGearInvalid); err != nil || !ok || i.Problem() != combat.GearProblemNegativeDistance {
			t.Errorf("distance %d: Run = %#v, %v; want invalid negative_distance", d, res, err)
		}
	}
	again, _ := deployRun(t, set, "new", combat.DeployedGearKindBallBearings, 10, combat.GroundLevelLevel)
	if !again.(combat.DeployGearPlaced).Gears().Equal(p.Gears()) {
		t.Error("equal commands must give equal results")
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-071 - NewDeployGearCommand returns a GearError for a blank id or zero deployer (blank_id) and a kind or ground that is not valid (invalid_value); Run returns ErrInvalidInput for the zero command and ctx.Err() for a cancelled ctx.
// Path: unhappy (invalid input, cancellation).
func TestNewDeployGearCommand(t *testing.T) {
	t.Parallel()
	ok, level := combat.DeployedGearKindCaltrops, combat.GroundLevelLevel
	for name, c := range map[string]struct {
		id     combat.DeployedGearID
		kind   combat.DeployedGearKind
		who    combat.CombatantID
		ground combat.GroundLevel
		want   combat.GearProblem
	}{
		"blank id": {"", ok, "a", level, combat.GearProblemBlankID}, "zero deployer": {"g", ok, "", level, combat.GearProblemBlankID},
		"invalid kind": {"g", 0, "a", level, combat.GearProblemInvalidValue}, "invalid ground": {"g", ok, "a", 0, combat.GearProblemInvalidValue},
	} {
		if _, err := combat.NewDeployGearCommand(combat.DeployedGears{}, c.id, c.kind, c.who, 0, c.ground); !errors.Is(err, &combat.GearError{Problem: c.want}) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	if _, err := combat.NewDeployGear().Run(context.Background(), combat.DeployGearCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command: err = %v, want ErrInvalidInput", err)
	}
	cmd, _ := combat.NewDeployGearCommand(combat.DeployedGears{}, "g", ok, "a", 0, level)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res, err := combat.NewDeployGear().Run(ctx, cmd); !errors.Is(err, context.Canceled) || res != nil {
		t.Errorf("cancelled ctx: %v, %v; want context.Canceled", res, err)
	}
}
