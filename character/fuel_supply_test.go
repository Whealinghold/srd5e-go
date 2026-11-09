package character_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

var pcID = character.MustIdentifier("pc-1")

func fresh(t *testing.T, k character.FuelKind) character.FuelSupply {
	t.Helper()
	s, err := character.NewFreshFuelSupply(pcID, k)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func burn(t *testing.T, s character.FuelSupply, d time.Duration) character.FuelBurnResult {
	t.Helper()
	r, err := s.Burn(d)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func spentTorch(t *testing.T) character.FuelSupply {
	t.Helper()
	return burn(t, fresh(t, character.FuelTorch), time.Hour).(character.FuelBurnExhausted).Supply()
}

// Requirement: doc/specs/exploration.md FR-036, FR-052, cases 4.5d - fresh supplies hold
// 1 hour, 1 hour and 6 hours (SRD 5.2.1, "Candle", p. 96; "Torch", p. 100; "Oil", pp. 98-99).
// Path: per case.
func TestNewFreshFuelSupply(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		kind character.FuelKind
		want time.Duration
	}{{character.FuelCandle, time.Hour}, {character.FuelTorch, time.Hour}, {character.FuelOilFlask, 6 * time.Hour}} {
		s := fresh(t, c.kind)
		if s.Remaining() != c.want || s.Kind() != c.kind || s.CharacterID() != pcID || c.kind.Capacity() != c.want {
			t.Errorf("%v: remaining %v", c.kind, s.Remaining())
		}
	}
	if _, err := character.NewFreshFuelSupply(pcID, character.FuelKind(0)); !errors.Is(err, &character.ExplorationError{Input: character.ExplorationFuelKind}) {
		t.Errorf("invalid kind err = %v", err)
	}
	if character.FuelKind(0).Capacity() != 0 || (character.FuelOilFlask + 1).Valid() || character.FuelKind(0).Valid() {
		t.Error("an undefined kind must be invalid with no capacity")
	}
	if character.FuelCandle.String() != "candle" || character.FuelTorch.String() != "torch" || character.FuelOilFlask.String() != "oil_flask" || character.FuelKind(0).String() != "unknown" {
		t.Error("FuelKind strings")
	}
}

// Requirement: doc/specs/exploration.md FR-037 to FR-039, FR-042, cases 4.5d - a burn
// continues below the remaining time, exhausts at it, never burns more than is held,
// and a spent supply burns nothing with no second event (SRD 5.2.1, "Torch", p. 100;
// "Oil", p. 99).
// Path: per case.
func TestFuelSupplyBurn(t *testing.T) {
	t.Parallel()
	min := time.Minute
	cases := []struct {
		name, path string
		kind       character.FuelKind
		spent      bool
		burn       time.Duration
		continues  bool
		left       time.Duration // remaining when continuing
		burned     time.Duration // burnedFor when exhausted
		event      bool
	}{
		{"Torch 30 minutes", "happy", character.FuelTorch, false, 30 * min, true, 30 * min, 0, false},
		{"Torch 59 minutes", "unhappy (boundary)", character.FuelTorch, false, 59 * min, true, 1 * min, 0, false},
		{"Torch exactly 1 hour", "unhappy (boundary)", character.FuelTorch, false, 60 * min, false, 0, 60 * min, true},
		{"Torch 61 minutes", "unhappy (boundary)", character.FuelTorch, false, 61 * min, false, 0, 60 * min, true},
		{"Oil 5 h 59 min", "unhappy (boundary)", character.FuelOilFlask, false, 359 * min, true, 1 * min, 0, false},
		{"Oil exactly 6 hours", "unhappy (boundary)", character.FuelOilFlask, false, 6 * time.Hour, false, 0, 6 * time.Hour, true},
		{"Oil 7 hours", "unhappy (boundary)", character.FuelOilFlask, false, 7 * time.Hour, false, 0, 6 * time.Hour, true},
		{"zero burn", "unhappy (boundary)", character.FuelCandle, false, 0, true, time.Hour, 0, false},
		{"zero burn on a spent supply", "unhappy (boundary)", character.FuelTorch, true, 0, false, 0, 0, false},
		{"already spent", "unhappy (state error)", character.FuelTorch, true, 10 * min, false, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := fresh(t, c.kind)
			if c.spent {
				s = spentTorch(t)
			}
			before := s
			switch r := burn(t, s, c.burn).(type) {
			case character.FuelBurnContinues:
				if !c.continues || r.Supply().Remaining() != c.left || r.Supply().Kind() != s.Kind() || r.Supply().CharacterID() != pcID {
					t.Errorf("continues with %v, want continues=%v left %v", r.Supply().Remaining(), c.continues, c.left)
				}
			case character.FuelBurnExhausted:
				ev, raised := r.Event()
				if c.continues || r.BurnedFor() != c.burned || r.Supply().Remaining() != 0 || raised != c.event ||
					(raised && (ev.CharacterID() != pcID || ev.Kind() != s.Kind())) {
					t.Errorf("exhausted: burned %v event %v", r.BurnedFor(), raised)
				}
				var _ character.LightFuelEvent = ev
			}
			if s != before {
				t.Error("Burn changed its receiver")
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-040 - a negative duration is rejected first,
// on a fresh and on a spent supply.
// Path: unhappy (invalid input).
func TestFuelSupplyBurnNegative(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]character.FuelSupply{"fresh": fresh(t, character.FuelCandle), "spent": spentTorch(t)} {
		r, err := s.Burn(-time.Second)
		if r != nil || !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ExplorationError{Input: character.ExplorationBurnDuration}) {
			t.Errorf("%s: Burn(-1s) = %v, %v", name, r, err)
		}
	}
}

// Requirement: doc/specs/exploration.md FR-041, cases 4.5d - burn time is additive over
// calls (SRD 5.2.1, "Oil", "Fuel", p. 99).
// Path: unhappy (rule interaction).
func TestFuelSupplyBurnIsAdditive(t *testing.T) {
	t.Parallel()
	s := fresh(t, character.FuelOilFlask)
	for i, h := range []time.Duration{2, 3} {
		c, ok := burn(t, s, h*time.Hour).(character.FuelBurnContinues)
		if !ok {
			t.Fatalf("burn %d must continue", i+1)
		}
		s = c.Supply()
	}
	if s.Remaining() != time.Hour {
		t.Fatalf("remaining = %v", s.Remaining())
	}
	if ex, ok := burn(t, s, time.Hour).(character.FuelBurnExhausted); !ok || ex.BurnedFor() != time.Hour {
		t.Error("the third burn must exhaust the flask")
	}
	s = fresh(t, character.FuelOilFlask)
	s = burn(t, s, 4*time.Hour).(character.FuelBurnContinues).Supply()
	if ex, ok := burn(t, s, 4*time.Hour).(character.FuelBurnExhausted); !ok || ex.BurnedFor() != 2*time.Hour {
		t.Error("overshooting burns must burn only the 2 hours left")
	}
}

// Requirement: doc/specs/exploration.md FR-052, FR-055, cases 4.5d - Restore accepts zero
// to the capacity and rejects outside it; a restore at capacity equals Fresh; ids differ.
// Path: per case.
func TestRestoreFuelSupply(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		kind       character.FuelKind
		remaining  time.Duration
		bad        bool
	}{
		{"below zero", "unhappy (invalid input)", character.FuelTorch, -time.Second, true},
		{"zero", "unhappy (boundary)", character.FuelTorch, 0, false},
		{"part-burned candle", "happy", character.FuelCandle, 30 * time.Minute, false},
		{"torch capacity", "unhappy (boundary)", character.FuelTorch, time.Hour, false},
		{"oil capacity", "unhappy (boundary)", character.FuelOilFlask, 6 * time.Hour, false},
		{"torch one second over", "unhappy (boundary)", character.FuelTorch, time.Hour + time.Second, true},
		{"oil one second over", "unhappy (boundary)", character.FuelOilFlask, 6*time.Hour + time.Second, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s, err := character.RestoreFuelSupply(pcID, c.kind, c.remaining)
			if c.bad {
				if !errors.Is(err, srderr.ErrInvalidInput) || !errors.Is(err, &character.ExplorationError{Input: character.ExplorationFuelRemaining}) {
					t.Errorf("err = %v", err)
				}
				return
			}
			if err != nil || s.Remaining() != c.remaining {
				t.Errorf("Restore = %v, %v", s.Remaining(), err)
			}
			if c.remaining == c.kind.Capacity() && s != fresh(t, c.kind) {
				t.Error("a restore at capacity must equal Fresh")
			}
		})
	}
	if _, err := character.RestoreFuelSupply(character.Identifier{}, character.FuelTorch, 0); !errors.Is(err, &character.ExplorationError{Input: character.ExplorationCharacterID}) {
		t.Errorf("zero id err = %v", err)
	}
	if _, err := character.RestoreFuelSupply(pcID, character.FuelKind(0), 0); !errors.Is(err, &character.ExplorationError{Input: character.ExplorationFuelKind}) {
		t.Errorf("invalid kind err = %v", err)
	}
	other, _ := character.NewFreshFuelSupply(character.MustIdentifier("pc-2"), character.FuelTorch)
	if other == fresh(t, character.FuelTorch) {
		t.Error("FR-055: supplies of two ids must differ")
	}
}
