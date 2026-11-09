package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/magic-item-effects.md FR-045 (SRD 5.2.1, "Staff of the Magi", p. 247) - absorption adds the spell level while the total is at
// most the maximum of 50, with a StaffSpellAbsorbed event, and a total over 50 overflows reporting the total (49 + 3 gives 52); a spell level below 1
// is invalid input.
// Path: unhappy (boundary).
func TestStaffSpellAbsorptionAbsorb(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name           string
		current, level int
		total          int
		overflow       bool
	}{
		{"empty staff", 0, 1, 1, false},
		{"49 + 1 reaches 50", 49, 1, 50, false},
		{"50 + 1 overflows", 50, 1, 51, true},
		{"49 + 3 explodes at 52", 49, 3, 52, true},
		{"48 + 2 reaches 50", 48, 2, 50, false},
		{"47 + 9 overflows", 47, 9, 56, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			charges := character.MustItemCharges(50, c.current)
			res := must((character.StaffSpellAbsorption{}).Absorb(holder, inst("magi-1"), charges, c.level))
			switch r := res.(type) {
			case character.StaffSpellAbsorptionAbsorbed:
				ev := r.Event()
				if c.overflow || r.Charges().Current() != c.total || r.Charges().Maximum() != 50 || ev.Level() != c.level || ev.CharacterID() != holder || ev.InstanceID() != inst("magi-1") {
					t.Errorf("absorbed = %+v %+v", r.Charges(), ev)
				}
			case character.StaffSpellAbsorptionOverflowed:
				if !c.overflow || r.Total() != c.total {
					t.Errorf("overflowed total = %d", r.Total())
				}
			}
		})
	}
	good := character.MustItemCharges(50, 5)
	for name, f := range map[string]func() error{
		"level 0":        func() error { return second((character.StaffSpellAbsorption{}).Absorb(holder, inst("m"), good, 0)) },
		"negative level": func() error { return second((character.StaffSpellAbsorption{}).Absorb(holder, inst("m"), good, -1)) },
		"zero holder": func() error {
			return second((character.StaffSpellAbsorption{}).Absorb(character.Identifier{}, inst("m"), good, 1))
		},
		"blank instance": func() error { return second((character.StaffSpellAbsorption{}).Absorb(holder, "", good, 1)) },
		"zero charges": func() error {
			return second((character.StaffSpellAbsorption{}).Absorb(holder, inst("m"), character.ItemCharges{}, 1))
		},
	} {
		wantInvalid(t, f(), name)
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-046, FR-047 (SRD 5.2.1, "Staff of Power", p. 246; "Staff of the Magi", p. 247; "Round Down", p. 5) -
// Retributive Strike: self 16 per charge, the area 4 (Power) or 6 (Magi) per charge and half rounded down on a success, as Force; 0 charges gives zeros;
// a negative count or an invalid staff is invalid input.
// Path: unhappy (boundary).
func TestRetributiveStrikeForCharges(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		staff                   character.RetributiveStrikeStaff
		charges                 int
		self, failed, succeeded int
	}{
		{character.StrikeStaffOfPower, 0, 0, 0, 0},
		{character.StrikeStaffOfPower, 1, 16, 4, 2},
		{character.StrikeStaffOfPower, 7, 112, 28, 14},
		{character.StrikeStaffOfPower, 50, 800, 200, 100},
		{character.StrikeStaffOfTheMagi, 0, 0, 0, 0},
		{character.StrikeStaffOfTheMagi, 1, 16, 6, 3},
		{character.StrikeStaffOfTheMagi, 7, 112, 42, 21},
		{character.StrikeStaffOfTheMagi, 50, 800, 300, 150},
		{character.StrikeStaffOfPower, 3, 48, 12, 6},
		{character.StrikeStaffOfTheMagi, 52, 832, 312, 156},
	} {
		d := must((character.RetributiveStrike{}).ForCharges(c.staff, c.charges))
		if d.Self() != c.self || d.AreaFailed() != c.failed || d.AreaSucceeded() != c.succeeded || d.DamageType() != content.DamageForce {
			t.Errorf("%v %d = %d %d %d", c.staff, c.charges, d.Self(), d.AreaFailed(), d.AreaSucceeded())
		}
	}
	// half rounded down: an odd area damage
	if d := must((character.RetributiveStrike{}).ForCharges(character.StrikeStaffOfPower, 1)); d.AreaSucceeded() != 2 {
		t.Error("rounding")
	}
	wantInvalid(t, second((character.RetributiveStrike{}).ForCharges(character.StrikeStaffOfPower, -1)), "negative")
	wantInvalid(t, second((character.RetributiveStrike{}).ForCharges(0, 1)), "zero staff")
	wantInvalid(t, second((character.RetributiveStrike{}).ForCharges(character.StrikeStaffOfTheMagi+1, 1)), "foreign staff")
	if character.RetributiveStrikeSaveDC != 17 {
		t.Error("DC")
	}
}
