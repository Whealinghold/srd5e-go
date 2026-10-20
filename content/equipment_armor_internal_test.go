package content

import "testing"

// Requirement: equipSpec FR-104 - an armor rule is valid only when both its
// base and its Dexterity treatment are valid; the base is checked on its own
// (a zero base with a valid treatment cannot be built outside the package).
// Path: unhappy.
func TestValidArmorRuleChecksBaseAlone(t *testing.T) {
	t.Parallel()
	if validArmorRule(BaseRule{base: 0, dexterity: FullDexterity{}}) {
		t.Error("a base of 0 with a valid Dexterity treatment is a valid rule [FR-104]")
	}
	if !validArmorRule(BaseRule{base: 11, dexterity: FullDexterity{}}) {
		t.Error("a base of 11 with a valid Dexterity treatment is not a valid rule [FR-104]")
	}
}
