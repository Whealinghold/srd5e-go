package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/environment-hazards-and-objects.md FR-067, FR-068, FR-069 (table 4.9b) - the fixed Hit Points and die expression of each Tiny to Large row, the fixed value being the expression's average rounded down, and a split-required result for Huge and Gargantuan.
// Path: per case.
func TestObjectHitPointsTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		size       content.CreatureSize
		fragileHP  int
		fragile    string
		resilientH int
		resilient  string
	}{
		{content.SizeTiny, 2, "1d4", 5, "2d4"}, {content.SizeSmall, 3, "1d6", 10, "3d6"},
		{content.SizeMedium, 4, "1d8", 18, "4d8"}, {content.SizeLarge, 5, "1d10", 27, "5d10"},
	}
	for _, c := range cases {
		for _, row := range []struct {
			d  combat.ObjectDurability
			hp int
			ex string
		}{{combat.ObjectFragile, c.fragileHP, c.fragile}, {combat.ObjectResilient, c.resilientH, c.resilient}} {
			l, err := (combat.ObjectHitPointsTable{}).Lookup(c.size, row.d)
			found, ok := l.(combat.ObjectHitPointsFound)
			want := dice.MustParseExpression(row.ex)
			if err != nil || !ok || found.Entry.HitPoints() != row.hp || found.Entry.Expression() != want || want.Average() != row.hp {
				t.Errorf("%v %v = %#v, %v, want %d from %s", c.size, row.d, l, err, row.hp, row.ex)
			}
		}
	}
	for _, size := range []content.CreatureSize{content.SizeHuge, content.SizeGargantuan} {
		for _, d := range []combat.ObjectDurability{combat.ObjectFragile, combat.ObjectResilient} {
			l, err := (combat.ObjectHitPointsTable{}).Lookup(size, d)
			if err != nil || l != (combat.ObjectHitPointsSplitRequired{Size: size}) {
				t.Errorf("%v %v = %#v, %v, want split required", size, d, l, err)
			}
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-067 - an invalid size or durability is an ErrInvalidInput error.
// Path: unhappy (invalid input).
func TestObjectHitPointsLookupInvalidInput(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		s content.CreatureSize
		d combat.ObjectDurability
	}{{0, combat.ObjectFragile}, {content.SizeMedium, 0}, {7, combat.ObjectResilient}, {content.SizeMedium, 3}} {
		if _, err := (combat.ObjectHitPointsTable{}).Lookup(c.s, c.d); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Lookup(%d, %d) = %v, want ErrInvalidInput", int(c.s), int(c.d), err)
		}
	}
	if combat.ObjectFragile.String() != "fragile" || combat.ObjectResilient.String() != "resilient" || combat.ObjectDurability(0).String() != "unknown" || combat.ObjectDurability(0).Valid() {
		t.Error("durability names or validity are wrong")
	}
}

func rollObject(t *testing.T, r combat.Roller, size content.CreatureSize, d combat.ObjectDurability) (combat.RollObjectHitPointsResult, error) {
	t.Helper()
	uc, err := combat.NewRollObjectHitPoints(r)
	if err != nil {
		t.Fatal(err)
	}
	return uc.Run(context.Background(), combat.RollObjectHitPointsCommand{Size: size, Durability: d})
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-070, FR-071 (table 4.9b) - one roll of the entry's expression gives current and maximum Hit Points both the total and no Temporary Hit Points; Huge and Gargantuan roll nothing.
// Path: per case.
func TestRollObjectHitPoints(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		roller combat.Roller
		total  int
	}{"minimum": {dicetest.NewMinimumRoller(), 4}, "maximum": {dicetest.NewMaximumRoller(), 32}} {
		roller := c.roller.(*dicetest.FixedRoller)
		res, err := rollObject(t, roller, content.SizeMedium, combat.ObjectResilient)
		got, ok := res.(combat.ObjectHitPointsRolled)
		if err != nil || !ok || got.HitPoints() != combat.MustHitPoints(c.total, c.total, 0) || got.Roll().Total() != c.total || got.Entry().HitPoints() != 18 {
			t.Fatalf("%s = %#v, %v, want %d Hit Points", name, res, err, c.total)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustParseExpression("4d8") {
			t.Errorf("%s requests = %v, want one 4d8 (FR-095)", name, reqs)
		}
	}
	roller := dicetest.NewMaximumRoller()
	res, err := rollObject(t, roller, content.SizeHuge, combat.ObjectFragile)
	if err != nil || res != (combat.ObjectHitPointsRollSplitRequired{Size: content.SizeHuge}) || len(roller.Requests()) != 0 {
		t.Errorf("Huge = %#v, %v with %d requests, want split required and no roll", res, err, len(roller.Requests()))
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-094, FR-095 - a roller that fails, panics or answers another expression is a *srderr.PortError; a nil roller, a bad size and a cancelled context are errors.
// Path: unhappy (collaborator failure).
func TestRollObjectHitPointsFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("roller down")
	for name, r := range map[string]combat.Roller{"fails": envFailingRoller(cause), "panics": envPanicRoller(), "wrong expression": envWrongRoller()} {
		res, err := rollObject(t, r, content.SizeSmall, combat.ObjectFragile)
		envPortFailure(t, err)
		if res != nil || name == "fails" && !errors.Is(err, cause) || name == "wrong expression" && !errors.Is(err, srderr.ErrContractViolation) {
			t.Errorf("%s = %v, %v", name, res, err)
		}
	}
	if _, err := combat.NewRollObjectHitPoints(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller = %v", err)
	}
	var zero combat.RollObjectHitPoints
	if _, err := zero.Run(context.Background(), combat.RollObjectHitPointsCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case = %v", err)
	}
	if _, err := rollObject(t, dicetest.NewMinimumRoller(), 0, combat.ObjectFragile); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero size = %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	uc, _ := combat.NewRollObjectHitPoints(dicetest.NewMinimumRoller())
	if _, err := uc.Run(cancelled, combat.RollObjectHitPointsCommand{Size: content.SizeSmall, Durability: combat.ObjectFragile}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
