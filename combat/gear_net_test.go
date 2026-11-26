package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-035, table 4.9.2 - Possibility refuses targetNotSeen when the thrower cannot see the target and outOfReach beyond 15 feet, 15 is accepted, and targetNotSeen is reported before outOfReach; a negative distance or a sight that is not valid is a GearError, never a refusal (SRD 5.2.1, "Net", p. 98).
// Path: per case.
func TestNetRulesPossibility(t *testing.T) {
	t.Parallel()
	seen, blind := combat.ThrowerSightSeesTarget, combat.ThrowerSightCannotSeeTarget
	cases := []struct {
		name, path string
		distance   int
		sight      combat.ThrowerSight
		want       combat.GearRejection
	}{
		{"15 feet seen", "unhappy (boundary)", 15, seen, 0},
		{"16 feet seen", "unhappy (boundary)", 16, seen, combat.GearRejectionOutOfReach},
		{"adjacent", "happy", 0, seen, 0},
		{"not seen", "unhappy (rule interaction)", 10, blind, combat.GearRejectionTargetNotSeen},
		{"not seen and out of reach", "unhappy (rule interaction)", 99, blind, combat.GearRejectionTargetNotSeen},
		{"largest distance", "unhappy (boundary)", math.MaxInt, seen, combat.GearRejectionOutOfReach},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.NetRules{}.Possibility(c.distance, c.sight)
			if err != nil || got != c.want {
				t.Errorf("Possibility(%d, %v) = %v, %v; want %v [%s FR-035]", c.distance, c.sight, got, err, c.want, gearSpec)
			}
		})
	}
	if _, err := (combat.NetRules{}).Possibility(-1, seen); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemNegativeDistance}) {
		t.Errorf("negative distance: err = %v, want negative_distance", err)
	}
	if _, err := (combat.NetRules{}).Possibility(5, 0); !errors.Is(err, &combat.GearError{Problem: combat.GearProblemInvalidValue}) {
		t.Errorf("invalid sight: err = %v, want invalid_value", err)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-036, table 4.9.2 - SaveDC is 8 plus the Dexterity modifier plus the Proficiency Bonus with no cap or floor (Dex +3 and PB +2 give 13; Dex -1 and PB +2 give 9), saturating at the limits of int.
// Path: happy; unhappy (boundary).
func TestNetRulesSaveDC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		dex, pb, dc int
	}{
		{"Dex +3, PB +2", 3, 2, 13}, {"Dex -1, PB +2", -1, 2, 9}, {"no cap", 40, 40, 88}, {"no floor", -40, 0, -32},
		{"largest", math.MaxInt, math.MaxInt, math.MaxInt}, {"smallest", math.MinInt, math.MinInt, math.MinInt},
	}
	for _, c := range cases {
		if got := (combat.NetRules{}).SaveDC(c.dex, c.pb); got != d20.DifficultyClass(c.dc) {
			t.Errorf("%s: SaveDC(%d, %d) = %d, want %d [%s FR-036]", c.name, c.dex, c.pb, got, c.dc, gearSpec)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-038 - a target of size Huge or Gargantuan succeeds against a Net automatically; Large and smaller save (SRD 5.2.1, "Net", p. 98; "Creature Size", p. 14).
// Path: unhappy (boundary).
func TestNetRulesAutoSucceeds(t *testing.T) {
	t.Parallel()
	for size, want := range map[content.CreatureSize]bool{content.SizeTiny: false, content.SizeSmall: false, content.SizeMedium: false,
		content.SizeLarge: false, content.SizeHuge: true, content.SizeGargantuan: true, 0: false} {
		if got := (combat.NetRules{}).AutoSucceeds(size); got != want {
			t.Errorf("AutoSucceeds(%v) = %v, want %v", size, got, want)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-044 - NetProfile.Status is an undamaged object with Armor Class 10, 5 of 5 Hit Points, no Temporary Hit Points, no Damage Threshold and Immunity to Bludgeoning, Poison and Psychic, the ruling Immunity added to the standard object affinities (SRD 5.2.1, "Net", p. 98).
// Path: happy.
func TestNetProfileStatus(t *testing.T) {
	t.Parallel()
	s := combat.NetProfile{}.Status()
	hp := s.HitPoints()
	if s.IsZero() || s.ArmorClass() != 10 || hp.Current() != 5 || hp.Maximum() != 5 || hp.Temporary() != 0 || s.Threshold().Amount() != 0 || s.IsDestroyed() {
		t.Errorf("status = %+v, want AC 10 and 5 of 5 Hit Points", s)
	}
	want := map[content.DamageType]bool{content.DamageBludgeoning: true, content.DamagePoison: true, content.DamagePsychic: true}
	got := s.Affinities().Immunities()
	if len(got) != 3 || len(s.Affinities().Resistances()) != 0 || len(s.Affinities().Vulnerabilities()) != 0 {
		t.Fatalf("affinities = %+v, want three Immunities only", s.Affinities())
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("immune to %v, want Bludgeoning, Poison and Psychic", d)
		}
	}
	if s != (combat.NetProfile{}).Status() || combat.NetThrowRangeFeet != 15 {
		t.Error("the profile must be a constant value")
	}
}
