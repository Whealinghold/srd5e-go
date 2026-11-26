package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/monster-spellcasting.md FR-013, RS-1 - an entry's note sets self only when it contains "self only" and waives Concentration when it contains "no Concentration", each without regard to case, with white space collapsed and at word boundaries; other text and an empty note set nothing (SRD 5.2.1, "Spellcasting", p. 257; "Green Hag", p. 295; "Adult Brass Dragon", p. 269).
// Path: per case.
func TestRestrictionsOfNote(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path, note string
		self, waives                  bool
	}{
		{"self only (Plane Shift)", "RS-1", "happy", "self only", true, false},
		{"self only with tail", "RS-1", "happy", "self only, and the hag leaves no tracks while Invisible", true, false},
		{"case", "FR-013", "unhappy (boundary)", "Self Only", true, false},
		{"informational: included in AC", "RS-1", "unhappy (rule interaction)", "included in AC", false, false},
		{"informational: cast before combat", "RS-1", "unhappy (rule interaction)", "cast before combat", false, false},
		{"24-hour duration", "FR-013", "unhappy (rule interaction)", "24-hour duration", false, false},
		{"shapechange", "RS-1", "unhappy (rule interaction)", "Beast or Humanoid form only, and no Concentration or Temporary Hit Points required", false, true},
		{"empty note", "FR-013", "unhappy (empty and minimal)", "", false, false},
		{"only white space", "FR-013", "unhappy (empty and minimal)", " \t\n ", false, false},
		{"word boundary: selfonly", "RS-1", "unhappy (boundary)", "selfonly", false, false},
		{"word boundary: no Concentrations", "RS-1", "unhappy (boundary)", "no Concentrations", false, false},
		{"word boundary: myself only", "FR-013", "unhappy (boundary)", "myself only", false, false},
		{"a boundary match after a failed one", "FR-013", "unhappy (boundary)", "myself only and self only", true, false},
		{"collapsed white space", "FR-013", "unhappy (boundary)", "SELF \n\t ONLY", true, false},
		{"both", "FR-013", "unhappy (rule interaction)", "self only, no concentration", true, true},
		{"at the end of the text", "FR-013", "unhappy (boundary)", "no Concentration", false, true},
		{"after a digit", "FR-013", "unhappy (boundary)", "3self only", false, false},
		{"after a non-ASCII letter", "FR-013", "unhappy (boundary)", "éself only", false, false},
		{"hyphenated self-only is not the phrase", "FR-013", "unhappy (boundary)", "self-only", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.MonsterSpellRestrictions{}.Of(c.note)
			if got.SelfOnly() != c.self || got.WaivesConcentration() != c.waives {
				t.Errorf("Of(%q) = self only %v, waives %v, want %v, %v [%s]", c.note, got.SelfOnly(), got.WaivesConcentration(), c.self, c.waives, monsterSpellcastingSpec+c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-017, CS-1 - ParseComponentWaiver normalises (trim, lower-case, collapse white space, strip one trailing period) and accepts "requiring no " plus "spell components" (all three) or a list of verbal, somatic and material joined by ", ", " or ", " and ", ", or " or ", and " and followed by " components"; an empty statement waives nothing; anything else, a repeated kind and a trailing clause included, is unrecognised (SRD 5.2.1, "Spellcasting", p. 257; "Deva", p. 279; "Cloaker", p. 274).
// Path: per case.
func TestParseComponentWaiver(t *testing.T) {
	t.Parallel()
	V, S, M := character.ComponentVerbal, character.ComponentSomatic, character.ComponentMaterial
	cases := []struct {
		name, requirement, path, statement string
		want                               []character.ComponentKind
		unrecognised                       bool
	}{
		{"statement material", "CS-1", "happy", "requiring no Material components", []character.ComponentKind{M}, false},
		{"statement S and M", "CS-1", "happy", "requiring no Somatic or Material components", []character.ComponentKind{S, M}, false},
		{"statement all", "CS-1", "happy", "requiring no spell components", []character.ComponentKind{V, S, M}, false},
		{"no statement", "CS-1", "unhappy (empty and minimal)", "", nil, false},
		{"only white space", "FR-017", "unhappy (empty and minimal)", "  \t ", nil, false},
		{"single verbal is a list of one", "FR-017", "happy", "requiring no Verbal components", []character.ComponentKind{V}, false},
		{"unknown statement", "CS-1", "unhappy (invalid input)", "requiring no Arcane components", nil, true},
		{"verbal or somatic", "CS-1", "happy", "requiring no Verbal or Somatic components", []character.ComponentKind{V, S}, false},
		{"spacing and period", "CS-1", "unhappy (boundary)", "  Requiring  no Material components.  ", []character.ComponentKind{M}, false},
		{"two periods", "FR-017", "unhappy (boundary)", "requiring no Material components..", nil, true},
		{"trailing clause", "CS-1", "unhappy (invalid input)", "requiring no spell components and using Wisdom", nil, true},
		{"repeated kind", "CS-1", "unhappy (boundary)", "requiring no Material or Material components", nil, true},
		{"comma list", "FR-017", "happy", "requiring no Verbal, Somatic components", []character.ComponentKind{V, S}, false},
		{"comma and or", "FR-017", "happy", "requiring no Verbal, Somatic, or Material components", []character.ComponentKind{V, S, M}, false},
		{"comma and and", "FR-017", "happy", "requiring no Verbal, Somatic, and Material components", []character.ComponentKind{V, S, M}, false},
		{"and", "FR-017", "happy", "requiring no Verbal and Material components", []character.ComponentKind{V, M}, false},
		{"missing prefix", "FR-017", "unhappy (invalid input)", "no Material components", nil, true},
		{"missing suffix", "FR-017", "unhappy (invalid input)", "requiring no Material", nil, true},
		{"nothing between", "FR-017", "unhappy (invalid input)", "requiring no components", nil, true},
		{"an empty list member", "FR-017", "unhappy (invalid input)", "requiring no Verbal, , Somatic components", nil, true},
		{"spell components inside a list", "FR-017", "unhappy (invalid input)", "requiring no spell components or Material components", nil, true},
		{"only the prefix", "FR-017", "unhappy (invalid input)", "requiring no ", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch got := combat.ParseComponentWaiver(c.statement).(type) {
			case combat.ComponentsWaived:
				if c.unrecognised || !slices.Equal(got.Kinds(), c.want) {
					t.Errorf("ParseComponentWaiver(%q) waives %v, want %v (unrecognised %v) [%s]", c.statement, got.Kinds(), c.want, c.unrecognised, monsterSpellcastingSpec+c.requirement)
				}
				for _, k := range []character.ComponentKind{V, S, M} {
					if got.Waives(k) != slices.Contains(c.want, k) {
						t.Errorf("Waives(%v) = %v, want %v", k, got.Waives(k), slices.Contains(c.want, k))
					}
				}
			case combat.ComponentStatementUnrecognized:
				if !c.unrecognised || got.Statement() != c.statement {
					t.Errorf("ParseComponentWaiver(%q) is unrecognised with text %q, want waived %v [%s]", c.statement, got.Statement(), c.want, c.requirement)
				}
			}
		})
	}
	if (combat.ComponentsWaived{}).Waives(0) || (combat.ComponentsWaived{}).Waives(character.ComponentKind(99)) {
		t.Errorf("the zero ComponentsWaived waives a kind")
	}
}
