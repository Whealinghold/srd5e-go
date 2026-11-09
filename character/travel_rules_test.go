package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func allSkills() []content.Skill {
	var out []content.Skill
	for s := content.SkillAcrobatics; s <= content.SkillSurvival; s++ {
		out = append(out, s)
	}
	return out
}

// Requirement: doc/specs/exploration.md FR-007 to FR-009, cases 4.2a - each pace's game
// effect on the named skills (SRD 5.2.1, "Travel Pace", p. 12).
// Path: per case.
func TestCircumstancesNamedSkills(t *testing.T) {
	t.Parallel()
	dis, adv := []modifier.RollCircumstance{modifier.Disadvantage{}}, []modifier.RollCircumstance{modifier.Advantage{}}
	cases := []struct {
		name, path string
		pace       character.TravelPace
		skill      content.Skill
		want       []modifier.RollCircumstance
	}{
		{"Fast Perception", "happy", character.PaceFast, content.SkillPerception, dis},
		{"Fast Survival", "happy", character.PaceFast, content.SkillSurvival, dis},
		{"Fast Stealth", "happy", character.PaceFast, content.SkillStealth, dis},
		{"Normal Stealth", "happy", character.PaceNormal, content.SkillStealth, dis},
		{"Slow Perception", "happy", character.PaceSlow, content.SkillPerception, adv},
		{"Slow Survival", "happy", character.PaceSlow, content.SkillSurvival, adv},
		{"Normal Perception", "unhappy (boundary)", character.PaceNormal, content.SkillPerception, nil},
		{"Normal Survival", "unhappy (boundary)", character.PaceNormal, content.SkillSurvival, nil},
		{"Slow Stealth", "unhappy (boundary)", character.PaceSlow, content.SkillStealth, nil},
		{"Fast Insight", "unhappy (boundary)", character.PaceFast, content.SkillInsight, nil},
		{"Fast Acrobatics", "unhappy (boundary)", character.PaceFast, content.SkillAcrobatics, nil},
		{"Fast Sleight of Hand", "unhappy (boundary)", character.PaceFast, content.SkillSleightOfHand, nil},
		{"Slow Animal Handling", "unhappy (boundary)", character.PaceSlow, content.SkillAnimalHandling, nil},
		{"Slow Athletics", "unhappy (boundary)", character.PaceSlow, content.SkillAthletics, nil},
		{"Normal Medicine", "unhappy (boundary)", character.PaceNormal, content.SkillMedicine, nil},
		{"invalid pace", "unhappy (invalid input)", character.TravelPace(0), content.SkillStealth, nil},
		{"invalid skill", "unhappy (invalid input)", character.PaceFast, content.Skill(0), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.TravelRules{}.Circumstances(character.LandTravel{Pace: c.pace}, c.skill)
			if len(got) != len(c.want) || (len(got) == 1 && got[0] != c.want[0]) {
				t.Errorf("Circumstances = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-010, FR-011, FR-015, cases 4.2a sweeps - over
// all 18 skills 3, 1 and 2 are non-empty at Fast, Normal and Slow, none at most one
// element, and a vehicle gives none for any skill (SRD 5.2.1, "Travel Pace", "Vehicles", p. 12).
// Path: unhappy (boundary).
func TestCircumstancesSweeps(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		mode character.TravelMode
		want int
	}{
		{character.LandTravel{Pace: character.PaceFast}, 3},
		{character.LandTravel{Pace: character.PaceNormal}, 1},
		{character.LandTravel{Pace: character.PaceSlow}, 2},
		{character.VehicleTravel{Speed: character.MustVehicleSpeed(3)}, 0},
		{character.VehicleTravel{}, 0},
		{nil, 0},
	} {
		n := 0
		for _, s := range allSkills() {
			got := character.TravelRules{}.Circumstances(c.mode, s)
			if len(got) > 1 {
				t.Errorf("%v %v: %d circumstances, want at most 1", c.mode, s, len(got))
			}
			if len(got) > 0 {
				n++
			}
		}
		if n != c.want {
			t.Errorf("%v: %d skills affected, want %d", c.mode, n, c.want)
		}
	}
}

// Requirement: doc/specs/exploration.md FR-048, FR-049 - the returned slice is a fresh
// copy and equal inputs give equal lists.
// Path: unhappy (immutability).
func TestCircumstancesCopyAndDeterminism(t *testing.T) {
	t.Parallel()
	mode := character.LandTravel{Pace: character.PaceFast}
	first := character.TravelRules{}.Circumstances(mode, content.SkillStealth)
	first[0] = modifier.Advantage{}
	_ = append(first, modifier.Advantage{})
	again := character.TravelRules{}.Circumstances(mode, content.SkillStealth)
	if len(again) != 1 || again[0] != (modifier.Disadvantage{}) {
		t.Errorf("a later call returned %v after the first was changed", again)
	}
}

// Requirement: doc/specs/exploration.md FR-012, cases 4.2b - a pace circumstance joins
// the others only through modifier.Resolve (SRD 5.2.1, "They Don't Stack", p. 8).
// Path: per case.
func TestCircumstancesJoinThroughResolve(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		pace       character.TravelPace
		skill      content.Skill
		other      []modifier.RollCircumstance
		want       modifier.RollMode
		dice       int
	}{
		{"pace alone, Fast Perception", "happy", character.PaceFast, content.SkillPerception, nil, modifier.WithDisadvantage, 2},
		{"Slow plus another Advantage", "unhappy (rule interaction)", character.PaceSlow, content.SkillPerception, []modifier.RollCircumstance{modifier.Advantage{}}, modifier.WithAdvantage, 2},
		{"Slow plus Disadvantage", "unhappy (rule interaction)", character.PaceSlow, content.SkillPerception, []modifier.RollCircumstance{modifier.Disadvantage{}}, modifier.Normal, 1},
		{"Fast plus Disadvantage", "unhappy (rule interaction)", character.PaceFast, content.SkillPerception, []modifier.RollCircumstance{modifier.Disadvantage{}}, modifier.WithDisadvantage, 2},
		{"Normal Stealth plus Advantage", "unhappy (rule interaction)", character.PaceNormal, content.SkillStealth, []modifier.RollCircumstance{modifier.Advantage{}}, modifier.Normal, 1},
		{"no pace effect plus Disadvantage", "unhappy (empty and minimal)", character.PaceNormal, content.SkillPerception, []modifier.RollCircumstance{modifier.Disadvantage{}}, modifier.WithDisadvantage, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cs := append(character.TravelRules{}.Circumstances(character.LandTravel{Pace: c.pace}, c.skill), c.other...)
			if got := modifier.Resolve(cs...); got != c.want || got.DiceCount() != c.dice {
				t.Errorf("Resolve = %v (%d dice), want %v (%d)", got, got.DiceCount(), c.want, c.dice)
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-013 - the Perception circumstances, resolved
// by the kernel and fed to CalculateDerivedStats, give Passive Perception 15, 10 and 5
// from a base of 10 (SRD 5.2.1, "Passive Perception", p. 186; "Travel Pace", p. 12).
// Path: per case.
func TestTravelPaceIntoPassivePerception(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		pace character.TravelPace
		want int
	}{{character.PaceSlow, 15}, {character.PaceNormal, 10}, {character.PaceFast, 5}} {
		t.Run(c.pace.String(), func(t *testing.T) {
			t.Parallel()
			in := base(t)
			in.AbilityScores = scoresOf(t, 10, 10, 10, 10, 10, 10)
			in.PerceptionRollMode = modifier.Resolve(character.TravelRules{}.Circumstances(character.LandTravel{Pace: c.pace}, content.SkillPerception)...)
			if got := calculate(t, in).PassivePerception().Value(); got != c.want {
				t.Errorf("Passive Perception at %v = %d, want %d", c.pace, got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-017, cases 4.3a - MilesPerHour is the table
// figure of a pace or the vehicle's Speed, unscaled (SRD 5.2.1, "Travel Pace", p. 12;
// p. 101).
// Path: per case.
func TestTravelRulesMilesPerHour(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		mode       character.TravelMode
		want       float64
	}{
		{"wagon on Normal", "happy", character.LandTravel{Pace: character.PaceNormal}, 3},
		{"Fast", "happy", character.LandTravel{Pace: character.PaceFast}, 4},
		{"Slow", "happy", character.LandTravel{Pace: character.PaceSlow}, 2},
		{"Rowboat", "happy", character.VehicleTravel{Speed: character.MustVehicleSpeed(3)}, 1.5},
		{"Keelboat", "happy", character.VehicleTravel{Speed: character.MustVehicleSpeed(2)}, 1},
		{"Airship", "happy", character.VehicleTravel{Speed: character.MustVehicleSpeed(16)}, 8},
		{"invalid pace", "unhappy (invalid input)", character.LandTravel{}, 0},
		{"zero vehicle", "unhappy (invalid input)", character.VehicleTravel{}, 0},
		{"nil mode", "unhappy (invalid input)", nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (character.TravelRules{}).MilesPerHour(c.mode); got != c.want {
				t.Errorf("MilesPerHour = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/exploration.md FR-014 - TravelMode is closed with two variants
// and a switch over it needs no default.
// Path: happy.
func TestTravelModeIsClosed(t *testing.T) {
	t.Parallel()
	name := func(m character.TravelMode) string {
		switch m.(type) {
		case character.LandTravel:
			return "land"
		case character.VehicleTravel:
			return "vehicle"
		}
		return ""
	}
	if name(character.LandTravel{Pace: character.PaceSlow}) != "land" || name(character.VehicleTravel{}) != "vehicle" {
		t.Error("variants not distinguished")
	}
}

// Requirement: doc/specs/exploration.md FR-018, cases 4.4a - the mounted hour is twice
// the pace's miles per hour, not the day figure doubled (SRD 5.2.1, "Travel Pace", p. 12).
// Path: per case.
func TestMountedHourMiles(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		pace       character.TravelPace
		want       int
	}{
		{"Fast", "happy", character.PaceFast, 8},
		{"Normal", "happy", character.PaceNormal, 6},
		{"Slow", "happy", character.PaceSlow, 4},
		{"invalid", "unhappy (invalid input)", character.TravelPace(0), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := (character.TravelRules{}).MountedHourMiles(c.pace); got != c.want {
				t.Errorf("MountedHourMiles(%v) = %d, want %d", c.pace, got, c.want)
			}
		})
	}
	if (character.TravelRules{}).MountedHourMiles(character.PaceFast) == 2*character.PaceFast.Row().MilesPerDay() {
		t.Error("the mounted hour must not be the day figure doubled")
	}
}
