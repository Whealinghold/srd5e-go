package content

import (
	"errors"
	"testing"
)

// FuzzParseUnexported feeds one input to the unexported text parsers.
//
// Requirement: doc/rules/testing.md section 10 - every parser has a Fuzz
// target; none panics, an accepted enum name round-trips through String, a
// rejected one is a *ParseError, and an accepted strict dice text is its own
// notation.
// Path: unhappy (invalid input).
func FuzzParseUnexported(f *testing.F) {
	for _, s := range []string{"", "GP", "Day", "Simple", "Melee", "Arrow", "Light", "village", "Artisan's Tools", "1d8", "d8", "2d6+1", "-5", "+7", "99999999999999999999", "\x00", "\xff"} {
		f.Add(s)
	}
	enums := map[string]func(string) (string, error){
		"Coin":           func(s string) (string, error) { v, err := parseCoin(s); return v.String(), err },
		"CostRate":       func(s string) (string, error) { v, err := parseCostRate(s); return v.String(), err },
		"Availability":   func(s string) (string, error) { v, err := parseAvailability(s); return v.String(), err },
		"WeaponCategory": func(s string) (string, error) { v, err := parseWeaponCategory(s); return v.String(), err },
		"WeaponType":     func(s string) (string, error) { v, err := parseWeaponType(s); return v.String(), err },
		"AmmunitionType": func(s string) (string, error) { v, err := parseAmmunitionType(s); return v.String(), err },
		"ArmorCategory":  func(s string) (string, error) { v, err := parseArmorCategory(s); return v.String(), err },
		"ToolGroup":      func(s string) (string, error) { v, err := parseToolGroup(s); return v.String(), err },
		"CraftSetName":   func(s string) (string, error) { v, err := parseCraftSetName(s); return v.String(), err },
		"Ability":        func(s string) (string, error) { v, err := parseAbilityExact(s); return v.String(), err },
	}
	f.Fuzz(func(t *testing.T, in string) {
		for vocabulary, parse := range enums {
			got, err := parse(in)
			if err == nil {
				if got != in {
					t.Fatalf("%s: %q parsed but String() = %q", vocabulary, in, got)
				}
				continue
			}
			var pe *ParseError
			if !errors.As(err, &pe) || pe.Value != in {
				t.Fatalf("%s: %q rejected with %v, want *ParseError holding the input", vocabulary, in, err)
			}
		}
		if e, ok := parseStrictDice(in); ok && e.Notation() != in {
			t.Fatalf("parseStrictDice accepted %q as %q", in, e.Notation())
		}
		if v := parseExponent(in); v < -maxExponent || v > maxExponent {
			t.Fatalf("parseExponent(%q) = %d out of range", in, v)
		}
	})
}
