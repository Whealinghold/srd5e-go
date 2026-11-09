package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-045, FR-046, FR-117, FR-107, table 4.4c (ammunition
// rows) - Expend removes one piece and fails on an empty supply; RecoverAfterFight is half
// rounded down; invalid numbers are input errors; the Must helper panics.
// Path: per case.
func TestAmmunitionSupply(t *testing.T) {
	t.Parallel()
	arrow := content.AmmunitionArrow
	for _, c := range []struct {
		name         string
		pieces, want int
	}{{"one piece", 5, 4}, {"last piece", 1, 0}} {
		after, err := character.MustAmmunitionSupply(c.pieces, arrow).Expend()
		if err != nil || after.Pieces() != c.want || after.Type() != arrow {
			t.Errorf("%s: %v, %v", c.name, after, err)
		}
	}
	empty, err := character.MustAmmunitionSupply(0, arrow).Expend()
	if !errors.Is(err, character.ErrSupplyEmpty) || empty != (character.AmmunitionSupply{}) {
		t.Errorf("empty: %v, %v", empty, err)
	}
	for used, want := range map[int]int{0: 0, 1: 0, 2: 1, 3: 1, 9: 4} {
		got, err := character.MustAmmunitionSupply(0, arrow).RecoverAfterFight(used)
		if err != nil || got != want {
			t.Errorf("RecoverAfterFight(%d) = %d, %v; want %d", used, got, err, want)
		}
	}
	if _, err := character.MustAmmunitionSupply(0, arrow).RecoverAfterFight(-1); err == nil {
		t.Error("RecoverAfterFight(-1) must fail")
	} else {
		wantInvalid(t, err, "RecoverAfterFight(-1)")
	}
	_, err = character.NewAmmunitionSupply(-1, arrow)
	wantInvalid(t, err, "supply -1")
	_, err = character.NewAmmunitionSupply(1, 0)
	wantInvalid(t, err, "supply of no type")
	defer func() {
		if recover() == nil {
			t.Error("MustAmmunitionSupply did not panic")
		}
	}()
	character.MustAmmunitionSupply(-1, arrow)
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-121, FR-107, FR-041, table 4.4c (Opportunity reach
// rows) - a reach is at least 1 foot; a Reach weapon adds 5 feet for Opportunity Attacks and
// another weapon leaves the attacker's reach alone.
// Path: per case.
func TestReachInFeetAndOpportunityAttackReach(t *testing.T) {
	t.Parallel()
	ws := weaponFixtures(t)
	if _, err := character.NewReachInFeet(0); err == nil {
		t.Error("reach 0 must fail")
	}
	_, err := character.NewReachInFeet(-3)
	wantInvalid(t, err, "reach -3")
	if r := character.MustReachInFeet(1); r.Feet() != 1 {
		t.Errorf("reach 1 = %d", r.Feet())
	}
	for _, c := range []struct {
		name, weapon string
		reach, want  int
	}{
		{"Reach weapon", "glaive", 5, 10},
		{"Reach weapon, reach 10", "whip", 10, 15},
		{"No Reach property", "longsword", 5, 5},
		{"No Reach property, reach 10", "longsword", 10, 10},
		{"zero reads as 5", "glaive", 0, 10},
	} {
		if got := character.OpportunityAttackReach(ws[c.weapon], reachOf(c.reach)).Feet(); got != c.want {
			t.Errorf("%s: reach = %d, want %d", c.name, got, c.want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustReachInFeet did not panic")
		}
	}()
	character.MustReachInFeet(0)
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-124, FR-106 - WithAmmunition returns a new wielding and
// leaves the receiver without a supply.
// Path: unhappy (immutability).
func TestWeaponWieldingWithAmmunition(t *testing.T) {
	t.Parallel()
	base := character.WeaponWielding{Grip: character.GripTwoHanded}
	with := base.WithAmmunition(character.MustAmmunitionSupply(3, content.AmmunitionBolt))
	if _, ok := base.Ammunition(); ok {
		t.Error("the receiver gained a supply")
	}
	if s, ok := with.Ammunition(); !ok || s.Pieces() != 3 || with.Grip != character.GripTwoHanded {
		t.Errorf("with = %v, %v", s, ok)
	}
}
