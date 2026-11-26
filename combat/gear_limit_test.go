package combat_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// gearsOf returns n distinct Caltrops.
func gearsOf(tb testing.TB, n int) []combat.DeployedGear {
	tb.Helper()
	// Deploy refuses a 257th gear, so the gears come from sets of at most 128.
	var out []combat.DeployedGear
	for start := 0; start < n; start += 128 {
		var set combat.DeployedGears
		for i := start; i < min(n, start+128); i++ {
			tr, err := set.Deploy(gearID(fmt.Sprintf("g%d", i)), combat.DeployedGearKindCaltrops, gid("owner"), 0, combat.GroundLevelLevel)
			if err != nil {
				tb.Fatal(err)
			}
			set = tr.(combat.DeployedGearsApplied).Gears()
		}
		out = append(out, set.Gears()...)
	}
	return out
}

// holdsOf returns n Net holds on n distinct targets.
func holdsOf(tb testing.TB, n int) []combat.GearHold {
	tb.Helper()
	out := make([]combat.GearHold, n)
	for i := range out {
		in := holdIn(fmt.Sprintf("h%d", i), combat.HoldKindNet)
		in.Target = gid(fmt.Sprintf("v%d", i))
		out[i] = gearHold(tb, in)
	}
	return out
}

// wantGearLimit fails unless err is a GearError with the problem limit_exceeded matching srderr.ErrInvalidInput.
func wantGearLimit(tb testing.TB, err error) {
	tb.Helper()
	var ge *combat.GearError
	if !errors.As(err, &ge) || ge.Problem != combat.GearProblemLimitExceeded || ge.Problem.String() != "limit_exceeded" || !errors.Is(err, srderr.ErrInvalidInput) {
		tb.Errorf("error = %v, want a GearError with GearProblemLimitExceeded", err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md Limits (plan P10) - DeployedGears holds at most 256 gears: NewDeployedGears reads at most 256 and Deploy on a set of 256 is a GearError with GearProblemLimitExceeded, the set unchanged.
// Path: per case.
func TestDeployedGearsCap(t *testing.T) {
	t.Parallel()
	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		runCap(t, "deployed gears", 256, func(n int) error {
			_, err := combat.NewDeployedGears(gearsOf(t, n))
			return err
		})
	})
	t.Run("deploy", func(t *testing.T) {
		t.Parallel()
		full, err := combat.NewDeployedGears(gearsOf(t, 256))
		if err != nil {
			t.Fatal(err)
		}
		tr, err := full.Deploy(gearID("extra"), combat.DeployedGearKindCaltrops, gid("owner"), 0, combat.GroundLevelLevel)
		wantGearLimit(t, err)
		if tr != nil || full.Len() != 256 {
			t.Errorf("transition %v, set of %d, want none and 256", tr, full.Len())
		}
		almost, _ := combat.NewDeployedGears(gearsOf(t, 255))
		if _, err := almost.Deploy(gearID("extra"), combat.DeployedGearKindCaltrops, gid("owner"), 0, combat.GroundLevelLevel); err != nil {
			t.Errorf("the 256th gear: error = %v", err)
		}
	})
}

// Requirement: doc/specs/deployed-and-restraining-gear.md Limits (plan P10) - GearHolds holds at most 256 holds: NewGearHolds reads at most 256, Begin on a set of 256 is GearProblemLimitExceeded, and so are the BindCreature and ThrowNet commands built over a full set.
// Path: per case.
func TestGearHoldsCap(t *testing.T) {
	t.Parallel()
	full := gearHolds(t, holdsOf(t, 256)...)
	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		runCap(t, "gear holds", 256, func(n int) error {
			_, err := combat.NewGearHolds(holdsOf(t, n))
			return err
		})
	})
	t.Run("begin", func(t *testing.T) {
		t.Parallel()
		_, err := full.Begin(gearHold(t, holdIn("extra", combat.HoldKindNet)))
		wantGearLimit(t, err)
		almost := gearHolds(t, holdsOf(t, 255)...)
		if _, err := almost.Begin(gearHold(t, holdIn("extra", combat.HoldKindNet))); err != nil {
			t.Errorf("the 256th hold: error = %v", err)
		}
	})
	t.Run("bind creature command", func(t *testing.T) {
		t.Parallel()
		in := bindIn(t)
		in.Holds = full
		_, err := combat.NewBindCreatureCommand(in)
		wantGearLimit(t, err)
		in.Holds = gearHolds(t, holdsOf(t, 255)...)
		if _, err := combat.NewBindCreatureCommand(in); err != nil {
			t.Errorf("255 holds: error = %v", err)
		}
	})
	t.Run("throw net command", func(t *testing.T) {
		t.Parallel()
		in := netIn()
		in.Holds = full
		_, err := combat.NewThrowNetCommand(in)
		wantGearLimit(t, err)
		in.Holds = gearHolds(t, holdsOf(t, 255)...)
		if _, err := combat.NewThrowNetCommand(in); err != nil {
			t.Errorf("255 holds: error = %v", err)
		}
	})
}

// Requirement: doc/specs/deployed-and-restraining-gear.md Limits (plan P10) - an EntryLedger holds at most 1024 records: NewEntryLedger reads at most 1024, and Record on a full ledger drops the oldest record and keeps the newest, so a host that never prunes is bounded.
// Path: per case.
func TestEntryLedgerCap(t *testing.T) {
	t.Parallel()
	records := func(n int) []combat.EntryRecord {
		out := make([]combat.EntryRecord, n)
		for i := range out {
			out[i] = combat.EntryRecord{GearID: gearID("g"), Creature: gid("c"), Stamp: gearStamp(i+1, "c")}
		}
		return out
	}
	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		runCap(t, "entry records", 1024, func(n int) error {
			_, err := combat.NewEntryLedger(records(n))
			return err
		})
	})
	t.Run("record drops the oldest", func(t *testing.T) {
		t.Parallel()
		all := records(1025)
		ledger, err := combat.NewEntryLedger(all[:1024])
		if err != nil {
			t.Fatal(err)
		}
		next := ledger.Record(all[1024])
		if next.Len() != 1024 || next.Has(all[0]) || !next.Has(all[1]) || !next.Has(all[1024]) || ledger.Len() != 1024 || !ledger.Has(all[0]) {
			t.Errorf("after the 1025th record: %d records, oldest kept %v, newest kept %v; the old ledger changed %v",
				next.Len(), next.Has(all[0]), next.Has(all[1024]), ledger.Len() != 1024 || !ledger.Has(all[0]))
		}
		if again := next.Record(all[1024]); !again.Equal(next) {
			t.Error("recording a record the ledger holds changed it")
		}
	})
}
