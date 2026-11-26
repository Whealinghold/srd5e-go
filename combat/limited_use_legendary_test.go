package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-limited-use.md FR-016, FR-018 - a Legendary Action window needs a creature and a round of at least 1; two windows with the same creature and round are equal and any other pair is not.
// Path: per case.
func TestNewLegendaryActionWindow(t *testing.T) {
	t.Parallel()
	b := combat.MustCombatantID("b")
	cases := []struct {
		name, requirement, path string
		who                     combat.CombatantID
		round                   int
		wantOK                  bool
	}{
		{"round 1", "FR-016", "unhappy (boundary)", b, 1, true},
		{"round 0", "FR-016 LA-1", "unhappy (invalid input)", b, 0, false},
		{"negative round", "FR-016", "unhappy (invalid input)", b, -1, false},
		{"zero creature", "FR-016", "unhappy (invalid input)", "", 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			w, err := combat.NewLegendaryActionWindow(c.who, c.round)
			if c.wantOK != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || w.IsZero() == c.wantOK {
				t.Errorf("NewLegendaryActionWindow(%q, %d) = %v, %v, want ok %v [%s%s]", c.who, c.round, w, err, c.wantOK, limitedUseSpec, c.requirement)
			}
		})
	}
	w1, w2 := luWindow(t, "b", 1), luWindow(t, "b", 1)
	if w1 != w2 || w1 == luWindow(t, "c", 1) || w1 == luWindow(t, "b", 2) || w1.Round() != 1 || w1.Combatants()[0] != b {
		t.Error("windows with equal creature and round must be equal, others not [FR-018]")
	}
	ids := w1.Combatants()
	ids[0] = "clobbered"
	if w1.Combatants()[0] != b {
		t.Error("Combatants is not a copy")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-015, FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, LA-1 - a Legendary Action is allowed after another creature's turn, costs one use and records its window; it is refused, with the state unchanged, as no Legendary Actions, own turn, incapacitated, window already used or no uses remaining, in that order of precedence (SRD 5.2.1, "Legendary Actions", p. 257; "Incapacitated", p. 184).
// Path: per case.
func TestTakeLegendaryAction(t *testing.T) {
	t.Parallel()
	w1, w2 := luWindow(t, "b", 1), luWindow(t, "c", 1)
	own := luWindow(t, luSelf, 1)
	avail, none := combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone
	noLegendary := luStart(t, luMonster(t, nil), combat.OutsideLair)
	// used builds a state whose windows were taken in order.
	used := func(ws ...combat.LegendaryActionWindow) combat.LimitedUseState {
		s := luStart(t, luDrake(t), combat.OutsideLair)
		for _, w := range ws {
			s = luTake(t, s, w)
		}
		return s
	}
	cases := []struct {
		name, requirement, path string
		state                   combat.LimitedUseState
		window                  combat.LegendaryActionWindow
		availability            combat.ConditionActionAvailability
		wantRefusal             combat.LegendaryActionRefusal // 0: allowed
		wantLeft                int
	}{
		{"taken after another's turn", "FR-015 LA-1", "happy", used(), w1, avail, 0, 2},
		{"last use", "FR-015 LA-1", "unhappy (boundary)", used(w1, w2), luWindow(t, "b", 2), avail, 0, 0},
		{"none left", "FR-019 LA-1", "unhappy (boundary)", used(w1, w2, luWindow(t, "b", 2)), luWindow(t, "c", 2), avail, combat.LegendaryRefusedNoUses, 0},
		{"own turn", "FR-016 LA-1", "unhappy (rule interaction)", used(), own, avail, combat.LegendaryRefusedOwnTurn, 3},
		{"incapacitated", "FR-017 LA-1", "unhappy (state error)", used(), w1, none, combat.LegendaryRefusedIncapacitated, 3},
		{"second in the same window", "FR-018 LA-1", "unhappy (rule interaction)", used(w1), w1, avail, combat.LegendaryRefusedWindowUsed, 2},
		{"creature sorting after the monster", "FR-016 FR-018", "unhappy (boundary)", used(), luWindow(t, "zed", 1), avail, 0, 2},
		{"second window", "FR-018 LA-1", "unhappy (rule interaction)", used(w1), w2, avail, 0, 1},
		{"same creature, next round", "FR-018 LA-1", "unhappy (boundary)", used(w1), luWindow(t, "b", 2), avail, 0, 1},
		{"window used earlier is allowed again after another", "FR-018", "unhappy (rule interaction)", used(w1, w2), w1, avail, 0, 0},
		{"no legendary actions", "FR-020 LA-1", "unhappy (empty and minimal)", noLegendary, w1, avail, combat.LegendaryRefusedNoActions, 0},
		{"precedence: own turn before incapacitated", "FR-021 LA-1", "unhappy (rule interaction)", used(), own, none, combat.LegendaryRefusedOwnTurn, 3},
		{"precedence: incapacitated before window used", "FR-021 LA-1", "unhappy (rule interaction)", used(w1), w1, none, combat.LegendaryRefusedIncapacitated, 2},
		{"precedence: window used before none left", "FR-021 LA-1", "unhappy (rule interaction)", used(w1, w2, luWindow(t, "b", 2)), luWindow(t, "b", 2), avail, combat.LegendaryRefusedWindowUsed, 0},
		{"precedence: no legendary before own turn", "FR-021 LA-1", "unhappy (rule interaction)", noLegendary, own, none, combat.LegendaryRefusedNoActions, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			req := limitedUseSpec + c.requirement
			before := c.state
			out, err := c.state.TakeLegendaryAction(c.window, c.availability)
			if err != nil {
				t.Fatalf("TakeLegendaryAction: %v", err)
			}
			if !c.state.Equal(before) {
				t.Fatal("the receiver changed")
			}
			if c.wantRefusal != 0 {
				ref, ok := out.(combat.LegendaryActionRefused)
				if !ok || ref.Reason != c.wantRefusal {
					t.Errorf("outcome = %#v, want refusal %v [%s]", out, c.wantRefusal, req)
				}
				if c.wantLeft != 0 && luLegendaryOf(t, c.state).Remaining() != c.wantLeft {
					t.Errorf("uses = %d, want %d unchanged", luLegendaryOf(t, c.state).Remaining(), c.wantLeft)
				}
				return
			}
			allowed, ok := out.(combat.LegendaryActionAllowed)
			if !ok {
				t.Fatalf("outcome = %#v, want allowed [%s]", out, req)
			}
			next := allowed.Transition.State()
			if got := luLegendaryOf(t, next).Remaining(); got != c.wantLeft {
				t.Errorf("uses left = %d, want %d [%s]", got, c.wantLeft, req)
			}
			if next.LastLegendaryWindow() != (combat.LegendaryWindowTaken{Window: c.window}) {
				t.Errorf("LastLegendaryWindow = %#v, want %v taken [FR-018]", next.LastLegendaryWindow(), c.window)
			}
			ev := allowed.Transition.Events()
			if e, ok := ev[0].(combat.LegendaryActionTaken); len(ev) != 1 || !ok || e.Window() != c.window || e.Remaining() != c.wantLeft {
				t.Errorf("events = %#v, want one LegendaryActionTaken for %v with %d left [%s]", ev, c.window, c.wantLeft, req)
			}
		})
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-034 - TakeLegendaryAction returns an error matching srderr.ErrInvalidInput for the zero window and for an availability that is not one of the two defined values; no refusal is an error.
// Path: unhappy (invalid input).
func TestTakeLegendaryActionRejectsInvalidArguments(t *testing.T) {
	t.Parallel()
	s := luStart(t, luDrake(t), combat.OutsideLair)
	for name, c := range map[string]struct {
		w combat.LegendaryActionWindow
		a combat.ConditionActionAvailability
	}{
		"zero window":            {combat.LegendaryActionWindow{}, combat.ConditionActionAvailabilityAvailable},
		"zero availability":      {luWindow(t, "b", 1), 0},
		"undefined availability": {luWindow(t, "b", 1), 3},
	} {
		out, err := s.TakeLegendaryAction(c.w, c.a)
		if out != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: TakeLegendaryAction = %#v, %v, want nil and ErrInvalidInput", name, out, err)
		}
	}
	zeroState := combat.LimitedUseState{}
	out, err := zeroState.TakeLegendaryAction(luWindow(t, "b", 1), combat.ConditionActionAvailabilityAvailable)
	if ref, ok := out.(combat.LegendaryActionRefused); err != nil || !ok || ref.Reason != combat.LegendaryRefusedNoActions {
		t.Errorf("zero state: %#v, %v, want refused no_legendary_actions", out, err)
	}
	if zeroState.Legendary() != (combat.LegendaryNone{}) || zeroState.LastLegendaryWindow() != (combat.LegendaryWindowNoneTaken{}) {
		t.Error("the zero state must read as no Legendary Actions and no window taken")
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-022, FR-023, LA-1 - the start of the monster's turn returns the Legendary Action uses to their maximum and adds LegendaryActionUsesRegained with the number regained only when uses were below it; a Short Rest and a Long Rest leave the uses unchanged (SRD 5.2.1, "Legendary Actions", p. 257).
// Path: per case.
func TestLegendaryUsesAtTurnStartAndRests(t *testing.T) {
	t.Parallel()
	taken := func(n int) combat.LimitedUseState {
		s := luStart(t, luDrake(t), combat.OutsideLair)
		for i := range n {
			s = luTake(t, s, luWindow(t, "b", i+1))
		}
		return s
	}
	cases := []struct {
		name, requirement, path string
		taken, wantRegained     int
	}{
		{"turn start regains all", "FR-022 LA-1", "happy", 3, 3},
		{"turn start regains one", "FR-022 LA-1", "unhappy (boundary)", 1, 1},
		{"turn start at the maximum", "FR-022 LA-1", "unhappy (empty and minimal)", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr, err := taken(c.taken).StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{})
			if err != nil {
				t.Fatal(err)
			}
			if got := luLegendaryOf(t, tr.State()); got.Remaining() != 3 || got.Maximum() != 3 {
				t.Errorf("uses = %d of %d, want 3 of 3 [%s%s]", got.Remaining(), got.Maximum(), limitedUseSpec, c.requirement)
			}
			ev := tr.Events()
			if c.wantRegained == 0 {
				if len(ev) != 0 || !tr.State().Equal(taken(0)) {
					t.Errorf("events = %v, want none and an equal state", ev)
				}
				return
			}
			if e, ok := ev[0].(combat.LegendaryActionUsesRegained); len(ev) != 1 || !ok || e.Count() != c.wantRegained {
				t.Errorf("events = %#v, want one LegendaryActionUsesRegained(%d)", ev, c.wantRegained)
			}
		})
	}
	spent := taken(3)
	for name, tr := range map[string]combat.LimitedUseTransition{"short": spent.AfterRest(character.ShortRest), "long": spent.AfterRest(character.LongRest)} {
		if got := luLegendaryOf(t, tr.State()).Remaining(); got != 0 {
			t.Errorf("after a %s rest the uses = %d, want 0 unchanged [FR-023]", name, got)
		}
	}
	turn, err := taken(1).StartOfOwnTurn(map[combat.LimitedUsePartRef]dice.DieRoll{})
	if w := turn.State().LastLegendaryWindow(); err != nil || w != (combat.LegendaryWindowTaken{Window: luWindow(t, "b", 1)}) {
		t.Errorf("the turn start must not reset the recorded window, got %#v", w)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-032 - the closed LegendaryTrack and LegendaryWindowRecord are value objects: equal fields give equal values and any differing field makes them unequal.
// Path: happy.
func TestLegendaryValuesAreComparable(t *testing.T) {
	t.Parallel()
	a := luLegendaryOf(t, luStart(t, luDrake(t), combat.OutsideLair))
	b := luLegendaryOf(t, luStart(t, luDrake(t), combat.OutsideLair))
	c := luLegendaryOf(t, luStart(t, luDrake(t), combat.InLair))
	if a != b || a == c {
		t.Error("LegendaryTracked values: equal fields must be equal and a differing maximum unequal")
	}
	if (combat.LegendaryWindowTaken{Window: luWindow(t, "b", 1)}) == (combat.LegendaryWindowTaken{Window: luWindow(t, "b", 2)}) {
		t.Error("windows of different rounds are equal")
	}
	for i, r := range []combat.LegendaryActionRefusal{0, 1, 2, 3, 4, 5, 6} {
		want := []string{"unknown", "no_legendary_actions", "own_turn", "incapacitated", "window_already_used", "no_uses_remaining", "unknown"}[i]
		if r.String() != want {
			t.Errorf("LegendaryActionRefusal(%d).String() = %q, want %q", r, r.String(), want)
		}
	}
}
