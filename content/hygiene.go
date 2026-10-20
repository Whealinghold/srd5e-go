package content

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// rejectedRune reports whether r may not appear in decoded text (schema
// FR-027, FR-028, plan item P14), so display text cannot spoof or smuggle:
//
//   - the control characters of Unicode category Cc (U+0000 to U+001F and
//     U+007F to U+009F), except the line feed U+000A where allowLF is set (a
//     text keeps its line feeds, FR-029);
//   - the line and paragraph separators U+2028 and U+2029;
//   - the bidirectional formatting characters U+061C (arabic letter mark),
//     U+200E and U+200F (left-to-right and right-to-left marks), U+202A to
//     U+202E (embeddings and overrides) and U+2066 to U+2069 (isolates).
//
// Zero-width and other format characters are not rejected.
func rejectedRune(r rune, allowLF bool) bool {
	switch {
	case r == '\n':
		return !allowLF
	case unicode.IsControl(r):
		return true
	case r == 0x2028 || r == 0x2029 || r == 0x061C || r == 0x200E || r == 0x200F:
		return true
	case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return false
}

// cleanCharacters reports whether s is valid UTF-8 and holds no rejected
// character. A Go string cannot hold a lone surrogate, so the decoder maps
// one to the same verdict as invalid UTF-8.
func cleanCharacters(s string, allowLF bool) bool {
	if !utf8.ValidString(s) {
		return false
	}
	return !strings.ContainsFunc(s, func(r rune) bool { return rejectedRune(r, allowLF) })
}

// validName is the rule of schema FR-027: a name is not empty, has no leading
// or trailing white space and holds only clean characters (no line feed).
func validName(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && cleanCharacters(s, false) // equivalent mutants: a string is never below "", so != and > agree, and TrimSpace never grows s
}

// blankText reports whether s has no non-white-space character (FR-028).
func blankText(s string) bool { return strings.TrimSpace(s) == "" } // equivalent mutant: a string is never below "", so == and <= agree

// validText is the rule of schema FR-028: a text has a non-white-space
// character and holds only clean characters, line feeds allowed.
func validText(s string) bool { return !blankText(s) && cleanCharacters(s, true) }

// validLine is the rule of a single-line, non-blank string (a source document
// or section, a background's feat qualifier): a non-white-space character and
// only clean characters, no line feed.
func validLine(s string) bool { return !blankText(s) && cleanCharacters(s, false) }
