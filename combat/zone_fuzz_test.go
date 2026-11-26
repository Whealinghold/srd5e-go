package combat_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzNewPersistentZoneID checks that the id constructor never panics, refuses
// exactly the blank strings and keeps the text it accepts.
func FuzzNewPersistentZoneID(f *testing.F) {
	for _, s := range []string{"", " ", "zone", "spirit_guardians", "\x00", "é"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		id, err := combat.NewPersistentZoneID(in)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewPersistentZoneID(%q) error %v does not match ErrInvalidInput", in, err)
			}
			if strings.TrimSpace(in) != "" {
				t.Fatalf("NewPersistentZoneID(%q) refused a non-blank id", in)
			}
			return
		}
		if id.String() != in {
			t.Fatalf("NewPersistentZoneID(%q) = %q, want the input", in, id)
		}
	})
}

// FuzzNewTurnMarker checks that the marker constructor accepts exactly the rounds
// from 1 up and never panics for any integer.
func FuzzNewTurnMarker(f *testing.F) {
	for _, n := range []int{0, 1, -1, 2, 1 << 30, -(1 << 30)} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, round int) {
		m, err := combat.NewTurnMarker(round, combat.MustCombatantID("x"))
		if round < 1 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewTurnMarker(%d) error = %v, want ErrInvalidInput", round, err)
			}
			return
		}
		if err != nil || m.Round() != round {
			t.Fatalf("NewTurnMarker(%d) = %v, %v", round, m, err)
		}
	})
}

// FuzzNewCasterMove checks the move constructor: a positive distance is accepted
// and a distance of 0 or below is refused, for any integer.
func FuzzNewCasterMove(f *testing.F) {
	for _, n := range []int{0, 1, 30, -30} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, feet int) {
		m, err := combat.NewCasterMove(feet, content.BonusAction)
		if feet <= 0 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewCasterMove(%d) error = %v, want ErrInvalidInput", feet, err)
			}
			return
		}
		if err != nil || m.Feet() != feet || m.How() != content.BonusAction {
			t.Fatalf("NewCasterMove(%d) = %v, %v", feet, m, err)
		}
	})
}

// FuzzNewRoundsDuration checks the rounds constructor: one round and up are accepted,
// anything below one is refused, for any integer.
func FuzzNewRoundsDuration(f *testing.F) {
	for _, n := range []int{0, 1, 600, -1} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, rounds int) {
		d, err := combat.NewRoundsDuration(rounds)
		if rounds < 1 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewRoundsDuration(%d) error = %v, want ErrInvalidInput", rounds, err)
			}
			return
		}
		if err != nil || d.Rounds() != rounds {
			t.Fatalf("NewRoundsDuration(%d) = %v, %v", rounds, d, err)
		}
	})
}

// FuzzNewDriftMove checks the drift constructor for any integer distance.
func FuzzNewDriftMove(f *testing.F) {
	for _, n := range []int{0, 10, -10} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, feet int) {
		m, err := combat.NewDriftMove(feet)
		if feet <= 0 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewDriftMove(%d) error = %v, want ErrInvalidInput", feet, err)
			}
			return
		}
		if err != nil || m.Feet() != feet {
			t.Fatalf("NewDriftMove(%d) = %v, %v", feet, m, err)
		}
	})
}

// FuzzNewCreatureTravelled checks the travel constructor: a non-negative distance is
// accepted and a negative one refused, for any integer.
func FuzzNewCreatureTravelled(f *testing.F) {
	for _, n := range []int{0, 5, -1} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, feet int) {
		ev, err := combat.NewCreatureTravelled(combat.NewZoneCreatureFacts(combat.MustCombatantID("x"), nil, false, false), feet)
		if feet < 0 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewCreatureTravelled(%d) error = %v, want ErrInvalidInput", feet, err)
			}
			return
		}
		if err != nil || ev.FeetInside() != feet {
			t.Fatalf("NewCreatureTravelled(%d) = %v, %v", feet, ev, err)
		}
	})
}
