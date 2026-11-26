package combat

import (
	"slices"
	"strings"
	"unicode"

	"github.com/Whealinghold/srd5e-go/character"
)

// MonsterSpellRestrictions are the two restrictions a spell entry's printed note can
// state: "self only" and "no Concentration" (SRD 5.2.1, "Spellcasting", p. 257;
// "Adult Brass Dragon", p. 269). The zero value states none. Comparable and safe for
// concurrent use.
type MonsterSpellRestrictions struct{ selfOnly, waivesConcentration bool }

// Of returns the restrictions the note states: SelfOnly when it contains "self only"
// and WaivesConcentration when it contains "no Concentration", each matched without
// regard to case, with runs of white space as one space and at word boundaries
// ("selfonly" and "no Concentrations" state nothing). Any other text, and an empty
// note, states nothing. It never fails.
func (MonsterSpellRestrictions) Of(note string) MonsterSpellRestrictions {
	n := normaliseText(note)
	return MonsterSpellRestrictions{selfOnly: containsPhrase(n, "self only"), waivesConcentration: containsPhrase(n, "no concentration")}
}

// SelfOnly reports whether the spell affects only the monster.
func (r MonsterSpellRestrictions) SelfOnly() bool { return r.selfOnly }

// WaivesConcentration reports whether the spell needs no Concentration to maintain.
func (r MonsterSpellRestrictions) WaivesConcentration() bool { return r.waivesConcentration }

// normaliseText lower-cases s, trims it and collapses each run of white space to one
// space.
func normaliseText(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

// containsPhrase reports whether s (already normalised) holds phrase as whole words.
func containsPhrase(s, phrase string) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], phrase)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(phrase)
		before := start == 0 || !isWordByte(s[start-1])
		after := end == len(s) || !isWordByte(s[end])
		if before && after {
			return true
		}
		from = start + 1
	}
}

// isWordByte reports whether b is part of a word. A non-ASCII byte counts as part of
// one, so a letter outside ASCII never forms a boundary.
func isWordByte(b byte) bool {
	return b >= 0x80 || unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
}

// ComponentWaiver is the closed set of readings of a printed components statement
// such as "requiring no Material components" (SRD 5.2.1, "Spellcasting", p. 257).
// Variants: ComponentsWaived, ComponentStatementUnrecognized. Consumers switch over it
// exhaustively. Comparable.
type ComponentWaiver interface{ isComponentWaiver() }

// ComponentsWaived is a statement that waives a unique set of component kinds, the
// empty set for an empty statement. The zero value waives none.
type ComponentsWaived struct{ verbal, somatic, material bool }

func (ComponentsWaived) isComponentWaiver() {}

// Waives reports whether the statement waives kind.
func (w ComponentsWaived) Waives(kind character.ComponentKind) bool {
	switch kind {
	case character.ComponentVerbal:
		return w.verbal
	case character.ComponentSomatic:
		return w.somatic
	case character.ComponentMaterial:
		return w.material
	}
	return false
}

// Kinds returns the waived kinds in the order Verbal, Somatic, Material, as a new
// slice.
func (w ComponentsWaived) Kinds() []character.ComponentKind {
	var out []character.ComponentKind
	for _, k := range []character.ComponentKind{character.ComponentVerbal, character.ComponentSomatic, character.ComponentMaterial} {
		if w.Waives(k) {
			out = append(out, k)
		}
	}
	return out
}

// ComponentStatementUnrecognized is a statement outside the grammar of
// ParseComponentWaiver; a cast under it is refused, never guessed.
type ComponentStatementUnrecognized struct{ text string }

func (ComponentStatementUnrecognized) isComponentWaiver() {}

// Statement returns the statement as given.
func (u ComponentStatementUnrecognized) Statement() string { return u.text }

// ParseComponentWaiver reads a printed components statement. It normalises the text
// (trim, lower-case, collapse white space, strip one trailing period) and then
// requires the grammar "requiring no " followed by "spell components" (Verbal,
// Somatic and Material) or a list of "verbal", "somatic" and "material" joined by
// ", ", " or ", " and ", ", or " or ", and " and followed by " components"; each
// listed kind is waived and a repeated kind is unrecognised. An empty statement
// waives nothing. Any other text, including a missing "requiring " prefix or a
// trailing clause, is ComponentStatementUnrecognized. It never fails and never
// panics.
func ParseComponentWaiver(statement string) ComponentWaiver {
	s := strings.TrimSuffix(normaliseText(statement), ".")
	if s == "" {
		return ComponentsWaived{}
	}
	rest, ok := strings.CutPrefix(s, "requiring no ")
	if !ok {
		return ComponentStatementUnrecognized{text: statement}
	}
	if rest == "spell components" {
		return ComponentsWaived{true, true, true}
	}
	body, ok := strings.CutSuffix(rest, " components")
	if !ok {
		return ComponentStatementUnrecognized{text: statement}
	}
	body = strings.NewReplacer(", or ", ",", ", and ", ",", " or ", ",", " and ", ",", ", ", ",").Replace(body)
	var w ComponentsWaived
	var seen []string
	for _, name := range strings.Split(body, ",") {
		if slices.Contains(seen, name) {
			return ComponentStatementUnrecognized{text: statement}
		}
		seen = append(seen, name)
		switch name {
		case "verbal":
			w.verbal = true
		case "somatic":
			w.somatic = true
		case "material":
			w.material = true
		default:
			return ComponentStatementUnrecognized{text: statement}
		}
	}
	return w
}
