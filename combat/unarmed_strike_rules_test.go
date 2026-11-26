package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/attacks.md FR-124, FR-065, FR-066 - UnarmedStrikeModifiers builds the ability modifier slot and
// the Proficiency Bonus slot, once each, and no others (SRD 5.2.1, "Unarmed Strike", p. 190).
// Path: per case.
func TestUnarmedStrikeModifiers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		ability, pb             int
	}{
		{"Strength +3, Proficiency +2", "FR-124", "happy", 3, 2},
		{"Strength -1, Proficiency +2", "FR-124", "unhappy (boundary)", -1, 2},
		{"Strength 0, Proficiency +3", "FR-124", "unhappy (boundary)", 0, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.UnarmedStrikeModifiers(abilityMod(c.ability), profBonus(c.pb))
			pb, has := got.ProficiencyBonus()
			if got.Ability() != modifier.New(c.ability) || !has || pb != modifier.New(c.pb) || len(got.Others()) != 0 || got.Count() != 2 {
				t.Errorf("= %+v, want ability %d, Proficiency Bonus %d and nothing else [%s]", got, c.ability, c.pb, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-078, FR-075 - "no more than one size larger" means the target is not more than one
// entry after the attacker in Tiny, Small, Medium, Large, Huge, Gargantuan (SRD 5.2.1, "Creature Size", p. 14).
// Path: per case.
func TestIsNoMoreThanOneSizeLarger(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		attacker, target        content.CreatureSize
		want                    bool
	}{
		{"same size", "FR-078", "happy", content.SizeMedium, content.SizeMedium, true},
		{"one larger", "FR-078", "unhappy (boundary)", content.SizeMedium, content.SizeLarge, true},
		{"two larger", "FR-078", "unhappy (boundary)", content.SizeMedium, content.SizeHuge, false},
		{"smaller", "FR-078", "unhappy (boundary)", content.SizeHuge, content.SizeTiny, true},
		{"Tiny to Small", "FR-078", "unhappy (boundary)", content.SizeTiny, content.SizeSmall, true},
		{"Tiny to Medium", "FR-078", "unhappy (boundary)", content.SizeTiny, content.SizeMedium, false},
		{"Huge to Gargantuan", "FR-078", "unhappy (boundary)", content.SizeHuge, content.SizeGargantuan, true},
		{"invalid attacker", "FR-078", "unhappy (invalid input)", 0, content.SizeSmall, false},
		{"invalid target", "FR-078", "unhappy (invalid input)", content.SizeLarge, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.IsNoMoreThanOneSizeLarger(c.attacker, c.target); got != c.want {
				t.Errorf("IsNoMoreThanOneSizeLarger(%v, %v) = %v, want %v [%s]", c.attacker, c.target, got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-063, FR-064 - an Unarmed Strike reaches every distance up to 5 feet, a target
// closer than 5 feet (adjacent at 0, or at 3) is not refused.
// Path: unhappy (boundary).
func TestUnarmedStrikeIsPossibleBelowItsReach(t *testing.T) {
	t.Parallel()
	for _, d := range []int{0, 3, 5} {
		p := unarmedParams(combat.UnarmedStrikeDamageOption{ArmorClass: 15})
		p.Distance = d
		if res, _ := runUnarmed(t, unarmedCommand(t, p), 12); unarmedKind(res) != "damage" {
			t.Errorf("distance %d: result = %#v, want the strike made", d, res)
		}
	}
}

func unarmedKind(r combat.ResolveUnarmedStrikeResult) string {
	if _, ok := r.(combat.UnarmedStrikeDamageResolved); ok {
		return "damage"
	}
	return "other"
}
