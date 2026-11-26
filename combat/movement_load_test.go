package combat_test

import (
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

type allSpeeds struct{ walk, burrow, climb, fly, swim int }

func speedsOf(tb testing.TB, s allSpeeds) combat.MovementSpeeds {
	tb.Helper()
	var specials []combat.MovementSpeedEntry
	for _, e := range []combat.MovementSpeedEntry{
		{Name: combat.SpeedNameBurrow, Feet: s.burrow}, {Name: combat.SpeedNameClimb, Feet: s.climb},
		{Name: combat.SpeedNameFly, Feet: s.fly}, {Name: combat.SpeedNameSwim, Feet: s.swim}} {
		if e.Feet >= 0 {
			specials = append(specials, e)
		}
	}
	return trkSpeeds(tb, s.walk, specials...)
}

// none marks a special speed the creature does not have.
const none = -1

// Requirement: per case
// Path: per case.
func TestLimitSpeedsByLoad(t *testing.T) {
	t.Parallel()
	const p, d, o = `SRD 5.2.1, "Carrying Capacity", p. 178`, `SRD 5.2.1, "Changes to Your Speeds", p. 188`, `unhappy (rule interaction)`
	all := allSpeeds{30, 15, 20, 40, 30}
	cases := []struct {
		name, requirement, path string
		in                      allSpeeds
		scope                   combat.LoadLimitScope
		want                    allSpeeds
	}{
		{"walking above cap (movement-lifting-and-flight.md FR-001)", p, "happy", allSpeeds{30, none, none, none, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{5, none, none, none, none}},
		{"special speeds follow the decrease (FR-002)", d, o, all, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{5, 0, 0, 15, 5}},
		{"walk 0 with fly: decrease 0 (FR-002, FR-058)", d, o, allSpeeds{0, none, none, 40, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{0, none, none, 40, none}},
		{"already at the cap (FR-058)", p, "unhappy (boundary)", allSpeeds{5, none, none, 40, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{5, none, none, 40, none}},
		{"one over the cap (FR-058)", p, "unhappy (boundary)", allSpeeds{6, none, none, 40, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{5, none, none, 39, none}},
		{"one under the cap (FR-058)", p, "unhappy (boundary)", allSpeeds{4, none, none, 40, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{4, none, none, 40, none}},
		{"below the cap (FR-058)", p, "unhappy (boundary)", allSpeeds{3, none, none, 40, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{3, none, none, 40, none}},
		{"walk 0 alone (FR-058)", p, "unhappy (boundary)", allSpeeds{0, none, none, none, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{0, none, none, none, none}},
		{"walk only keeps the special speeds (FR-057)", p, o, all, combat.LoadLimitScopeWalkOnly, allSpeeds{5, 15, 20, 40, 30}},
		{"walk only below the cap (FR-057)", p, o, allSpeeds{4, none, none, 40, none}, combat.LoadLimitScopeWalkOnly, allSpeeds{4, none, none, 40, none}},
		{"the largest speeds (overflow limit)", p, "unhappy (boundary)", allSpeeds{math.MaxInt, none, none, math.MaxInt, none}, combat.LoadLimitScopeAllSpeedsByEqualAmount, allSpeeds{5, none, none, 5, none}},
	}
	limit := moveLimit(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := speedsOf(t, c.in)
			if got, want := combat.LimitSpeedsByLoad(in, limit, c.scope), speedsOf(t, c.want); got != want {
				t.Errorf("LimitSpeedsByLoad(%+v, %v) = %+v, want %+v [%s]", c.in, c.scope, got, want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-005 - LimitSpeedsByLoad returns a new value and leaves its input as it was.
// Path: unhappy (immutability).
func TestLimitSpeedsByLoadLeavesInputUnchanged(t *testing.T) {
	t.Parallel()
	in := speedsOf(t, allSpeeds{30, none, none, 40, none})
	before := in
	for _, scope := range []combat.LoadLimitScope{combat.LoadLimitScopeAllSpeedsByEqualAmount, combat.LoadLimitScopeWalkOnly} {
		combat.LimitSpeedsByLoad(in, moveLimit(t), scope)
		if in != before {
			t.Errorf("input changed under scope %v: %+v, want %+v", scope, in, before)
		}
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-001, FR-059 and the plan's named boundaries for step 7c - a load at the Carry
// maximum, one pound above it, at the Drag/Lift/Push maximum and one above it is assessed by the character context, and only the
// middle band is a LoadSpeedLimited that limits a Speed; a Strength of the maximum score and the zero score are the extremes.
// Path: unhappy (boundary).
func TestLoadThresholdsLimitTheSpeedOnlyBetweenCarryAndDragLiftPush(t *testing.T) {
	t.Parallel()
	speeds := speedsOf(t, allSpeeds{30, none, none, 40, none})
	cases := []struct {
		name     string
		size     content.CreatureSize
		strength ability.Score
		load     int
		limited  bool
		beyond   bool
	}{
		{"at the Carry maximum", content.SizeMedium, moveScore(t, 10), 150, false, false},
		{"one pound above Carry", content.SizeMedium, moveScore(t, 10), 151, true, false},
		{"at Drag/Lift/Push", content.SizeMedium, moveScore(t, 10), 300, true, false},
		{"one pound above Drag/Lift/Push", content.SizeMedium, moveScore(t, 10), 301, false, true},
		{"one pound below Carry", content.SizeMedium, moveScore(t, 10), 149, false, false},
		{"Strength 30 at Carry", content.SizeMedium, moveScore(t, ability.MaxScore), 30 * 15, false, false},
		{"Strength 30 one above Carry", content.SizeMedium, moveScore(t, ability.MaxScore), 30*15 + 1, true, false},
		{"Strength 1 one above Drag/Lift/Push", content.SizeMedium, moveScore(t, 1), 31, false, true},
		{"the zero score carries nothing", content.SizeMedium, ability.Score{}, 1, false, true},
		{"the zero score and no load", content.SizeMedium, ability.Score{}, 0, false, false},
		{"Tiny at half-pound Carry", content.SizeTiny, moveScore(t, 10), 75, false, false},
		{"Tiny one above half-pound Carry", content.SizeTiny, moveScore(t, 10), 76, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a, err := character.NewCarryingCapacity(c.size, c.strength).Assess(c.load)
			if err != nil {
				t.Fatalf("Assess(%d): %v", c.load, err)
			}
			l, limited := a.(character.LoadSpeedLimited)
			_, beyond := a.(character.LoadBeyondDragLiftPush)
			if limited != c.limited || beyond != c.beyond {
				t.Fatalf("Assess(%d) = %#v, want limited %v, beyond %v", c.load, a, c.limited, c.beyond)
			}
			if limited {
				if got := combat.LimitSpeedsByLoad(speeds, l, combat.LoadLimitScopeAllSpeedsByEqualAmount); got.Walk() != l.LimitFeet() {
					t.Errorf("limited walking Speed = %d, want %d", got.Walk(), l.LimitFeet())
				}
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestApplyArmorSpeedPenalty(t *testing.T) {
	t.Parallel()
	const p, d = `SRD 5.2.1, "Armor", Strength, p. 92`, `doc/specs/movement-lifting-and-flight.md FR-060`
	cases := []struct {
		name, requirement, path string
		in                      allSpeeds
		strength                int
		want                    allSpeeds
	}{
		{"below the requirement", p, "unhappy (boundary)", allSpeeds{30, none, none, 40, none}, 12, allSpeeds{20, none, none, 30, none}},
		{"at the requirement", p, "unhappy (boundary)", allSpeeds{30, none, none, 40, none}, 13, allSpeeds{30, none, none, 40, none}},
		{"floor at 0", p, "unhappy (boundary)", allSpeeds{5, none, none, none, none}, 12, allSpeeds{0, none, none, none, none}},
		{"the largest speed", d, "unhappy (boundary)", allSpeeds{math.MaxInt, none, none, math.MaxInt, none}, 1, allSpeeds{math.MaxInt - 10, none, none, math.MaxInt - 10, none}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.ApplyArmorSpeedPenalty(speedsOf(t, c.in), moveArmor(t, 13), moveScore(t, c.strength))
			if want := speedsOf(t, c.want); got != want {
				t.Errorf("ApplyArmorSpeedPenalty(%+v, Strength %d) = %+v, want %+v [%s]", c.in, c.strength, got, want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/movement-lifting-and-flight.md FR-060, FR-002 - armor without a Strength requirement gives no penalty, and
// armor first then a heavy load decreases walk 30, fly 40 by 10 and then by 25 to walk 5, fly 15.
// Path: unhappy (rule interaction).
func TestArmorThenHeavyLoad(t *testing.T) {
	t.Parallel()
	in := speedsOf(t, allSpeeds{30, none, none, 40, none})
	if got := combat.ApplyArmorSpeedPenalty(in, moveArmor(t, 0), moveScore(t, 1)); got != in {
		t.Errorf("armor with no requirement changed the speeds to %+v", got)
	}
	armored := combat.ApplyArmorSpeedPenalty(in, moveArmor(t, 13), moveScore(t, 12))
	got := combat.LimitSpeedsByLoad(armored, moveLimit(t), combat.LoadLimitScopeAllSpeedsByEqualAmount)
	if want := speedsOf(t, allSpeeds{5, none, none, 15, none}); got != want {
		t.Errorf("armor then load = %+v, want %+v", got, want)
	}
}
