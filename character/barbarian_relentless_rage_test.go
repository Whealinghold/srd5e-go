package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/class-barbarian.md FR-065, FR-067, FR-087 and RR-08, RR-09 - Relentless Rage is unavailable with the first reason of: no Barbarian levels, level 11 not reached, no active Rage, not
// dropped to 0 Hit Points; it is available at level 11 with a Rage and the drop.
// Path: per case (unhappy: state error, boundary).
func TestRelentlessRageAvailability(t *testing.T) {
	t.Parallel()
	dropped := character.BarbarianHostFacts{DroppedToZeroHitPoints: true}
	for _, c := range []struct {
		name  string
		level int
		rage  bool
		facts character.BarbarianHostFacts
		want  character.BarbarianReason
	}{
		{"RR-08 inactive", 11, false, dropped, character.BarbarianRageNotActive}, {"RR-09 level 10", 10, true, dropped, character.BarbarianFeatureNotGained},
		{"not dropped", 11, true, character.BarbarianHostFacts{}, character.BarbarianNotDroppedToZero}, {"level 11", 11, true, dropped, 0}, {"level 20", 20, true, dropped, 0},
		{"level beats Rage", 10, false, dropped, character.BarbarianFeatureNotGained}, {"Rage beats the drop", 11, false, character.BarbarianHostFacts{}, character.BarbarianRageNotActive},
	} {
		state := character.InitialBarbarianState()
		if c.rage {
			state = brbRaging(t, c.level, 0)
		}
		res := character.RelentlessRage{}.Availability(brbStats(t, c.level), state, c.facts)
		if c.want == 0 {
			if res != (character.RelentlessRageAvailable{}) {
				t.Errorf("%s: %#v, want available", c.name, res)
			}
		} else if res != (character.RelentlessRageUnavailable{Reason: c.want}) {
			t.Errorf("%s: %#v, want unavailable %v", c.name, res, c.want)
		}
	}
	if res := (character.RelentlessRage{}).Availability(character.BarbarianLevelStats{}, brbRaging(t, 11, 0), dropped); res != (character.RelentlessRageUnavailable{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("zero stats = %#v", res)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-065, FR-066 and RR-01 to RR-05, RR-12 to RR-15 - Resolve sets Hit Points to twice the Barbarian level on a success (the total meeting the DC), none on a failure, counts the use either
// way and raises the next DC by 5; exactly the DC succeeds and one below fails; a natural 20 or 1 has no special effect.
// Path: per case (RR rows; unhappy: boundary).
func TestRelentlessRageResolve(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label       string
		level, uses int
		natural     int
		mods        int
		dc          int
		success     bool
		hit         int
	}{
		{"RR-01", 11, 0, 15, 0, 10, true, 22}, {"RR-02", 11, 0, 5, 0, 10, false, 0}, {"RR-03", 11, 1, 16, 0, 15, true, 22}, {"RR-04", 14, 2, 8, 0, 20, false, 0},
		{"RR-05", 20, 0, 15, 0, 10, true, 40}, {"RR-12", 11, 0, 8, 2, 10, true, 22}, {"RR-13", 11, 0, 7, 2, 10, false, 0}, {"RR-14", 11, 1, 13, 2, 15, true, 22},
		{"RR-15", 11, 1, 12, 2, 15, false, 0}, {"RR-10 natural 20", 11, 0, 20, -11, 10, false, 0}, {"RR-11 natural 1", 11, 0, 1, 11, 10, true, 22},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			state := brbEnter(t, c.level, character.MustBarbarianState(0, c.uses), "")
			check := brbSave(t, modifier.Normal, []int{c.natural}, c.mods, c.dc)
			res := must(character.RelentlessRage{}.Resolve(brbStats(t, c.level), state, check))
			var out interface {
				State() character.BarbarianState
				Check() d20.CheckResult
			}
			hp := 0
			switch r := res.(type) {
			case character.RelentlessRageResolutionSucceeded:
				out, hp = r, r.HitPoints()
			case character.RelentlessRageResolutionStillFailed:
				out = r
			}
			if (hp > 0) != c.success || hp != c.hit || out.State().RelentlessRageUses() != c.uses+1 || out.State().RelentlessRageDC() != c.dc+5 || out.Check().Roll().Natural() != c.natural {
				t.Errorf("= %#v (state %v), want success %v and %d Hit Points", res, out.State(), c.success, c.hit)
			}
			if _, ok := res.(character.RelentlessRageResolutionSucceeded); ok != c.success {
				t.Errorf("success = %v, want %v", ok, c.success)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-065, FR-067 - Resolve refuses, with an error matching srderr.ErrInvalidInput, a use that Availability says unavailable, a check that is not a saving throw and a check against another DC.
// Path: per case (unhappy: invalid input).
func TestRelentlessRageResolveRefusals(t *testing.T) {
	t.Parallel()
	state := brbRaging(t, 11, 0)
	stats := brbStats(t, 11)
	good := brbSave(t, modifier.Normal, []int{12}, 0, 10)
	if _, err := (character.RelentlessRage{}).Resolve(stats, state, good); err != nil {
		t.Fatal(err)
	}
	_, err := character.RelentlessRage{}.Resolve(stats, character.InitialBarbarianState(), good)
	wantInvalid(t, err, "no Rage")
	_, err = character.RelentlessRage{}.Resolve(brbStats(t, 10), brbRaging(t, 10, 0), good)
	wantInvalid(t, err, "level 10")
	_, err = character.RelentlessRage{}.Resolve(stats, state, fgtCheck(t, d20.AbilityCheck, modifier.Normal, []int{12}, 0, 10))
	wantInvalid(t, err, "not a saving throw")
	_, err = character.RelentlessRage{}.Resolve(stats, state, brbSave(t, modifier.Normal, []int{12}, 0, 15))
	wantInvalid(t, err, "another DC")
}

// Requirement: doc/specs/class-barbarian.md FR-071 - Persistent Rage regains every expended Rage use once between Long Rests from level 15, with the RagesRegained event; a second regaining before a Long Rest is refused and
// a Long Rest makes it available again; it is refused below level 15 and with no Barbarian levels.
// Path: per case (unhappy: state error, boundary).
func TestPersistentRageRegainOnInitiative(t *testing.T) {
	t.Parallel()
	id, stats := brbID(t), brbStats(t, 15)
	spent := character.MustBarbarianState(5, 0)
	first, ok := must(character.PersistentRage{}.RegainOnInitiative(id, stats, spent)).(character.RagesRegainedOnInitiative)
	if !ok || first.State().RagesExpended() != 0 || !first.State().PersistentRageUsed() {
		t.Fatalf("first regaining = %#v", first)
	}
	ev, isEvent := first.Events()[0].(character.RagesRegained)
	if len(first.Events()) != 1 || !isEvent || ev.CharacterID() != id {
		t.Errorf("events = %#v", first.Events())
	}
	first.Events()[0] = nil
	if first.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
	again := brbWithRagesExpended(t, first.State(), 3)
	if got := must(character.PersistentRage{}.RegainOnInitiative(id, stats, again)); got != (character.PersistentRageRefused{Reason: character.BarbarianPersistentRageSpent}) {
		t.Errorf("second regaining = %#v", got)
	}
	if _, ok := must(character.PersistentRage{}.RegainOnInitiative(id, stats, again.AfterRest(character.ShortRest))).(character.PersistentRageRefused); !ok {
		t.Error("a Short Rest must not make Persistent Rage available")
	}
	if _, ok := must(character.PersistentRage{}.RegainOnInitiative(id, stats, again.AfterRest(character.LongRest))).(character.RagesRegainedOnInitiative); !ok {
		t.Error("a Long Rest must make Persistent Rage available")
	}
	if got := must(character.PersistentRage{}.RegainOnInitiative(id, brbStats(t, 14), spent)); got != (character.PersistentRageRefused{Reason: character.BarbarianFeatureNotGained}) {
		t.Errorf("level 14 = %#v", got)
	}
	if got := must(character.PersistentRage{}.RegainOnInitiative(id, character.BarbarianLevelStats{}, spent)); got != (character.PersistentRageRefused{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("zero stats = %#v", got)
	}
	_, err := character.PersistentRage{}.RegainOnInitiative(character.Identifier{}, stats, spent)
	wantInvalid(t, err, "zero id")
}

// brbWithRagesExpended returns state with n Rage uses expended and everything else as it was, by spending uses through the rules.
func brbWithRagesExpended(t *testing.T, state character.BarbarianState, n int) character.BarbarianState {
	t.Helper()
	for state.RagesExpended() < n {
		// a Rage entered and ended expends one use and leaves the other fields alone
		entered := must(character.BarbarianRage{}.Enter(brbID(t), brbStats(t, 15), state, character.NewBarbarianTurn(), character.BarbarianHostFacts{}, "", 30)).(character.RageEntered)
		state = must(character.BarbarianRage{}.End(brbID(t), brbStats(t, 15), entered.State(), character.RageHeavyArmorDonned{})).(character.RageEndApplied).State()
	}
	return state
}
