package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: per case
// Path: per case.
func TestPossessionSpellClassification(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path, id string
		prevention     combat.PossessionPrevention
		preventionOK   bool
		ending         combat.PossessionEnding
		endingOK       bool
	}{
		{"prevention", "happy", "protection_from_evil_and_good", combat.ProtectionFromEvilAndGood, true, 0, false},
		{"ending", "happy", "dispel_evil_and_good", 0, false, combat.DispelEvilAndGood, true},
		{"a curse spell", "unhappy (boundary)", "remove_curse", 0, false, 0, false},
		{"unknown id", "unhappy (invalid input)", "no_such_spell", 0, false, 0, false},
		{"the empty id", "unhappy (invalid input)", "", 0, false, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: FR-037, FR-039, table 4.6a; SRD 5.2.1, "Possession", p. 186
			p, pok := combat.PossessionPreventionFromSpell(content.SpellID(c.id))
			e, eok := combat.PossessionEndingFromSpell(content.SpellID(c.id))
			if p != c.prevention || pok != c.preventionOK || e != c.ending || eok != c.endingOK {
				t.Errorf("%q: prevention %v,%v ending %v,%v [%s]", c.id, p, pok, e, eok, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-039 - PossessionEnding.PossessingEffect exists for an end the possessing effect itself defines; the String values are pinned.
// Path: happy.
func TestPossessionEnumsString(t *testing.T) {
	t.Parallel()
	if combat.PossessingEffect == combat.DispelEvilAndGood || combat.PossessingEffect.String() != "possessing_effect" ||
		combat.DispelEvilAndGood.String() != "dispel_evil_and_good" || combat.PossessionEnding(0).String() != "unknown" ||
		combat.ProtectionFromEvilAndGood.String() != "protection_from_evil_and_good" || combat.PossessionPrevention(0).String() != "unknown" {
		t.Error("possession enum names changed")
	}
}
