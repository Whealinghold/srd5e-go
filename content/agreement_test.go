package content_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// jsonString quotes s as a JSON string literal without touching invalid UTF-8
// (which encoding/json would replace), so a row can carry raw bad bytes.
func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case c < 0x20:
			fmt.Fprintf(&b, `\u%04x`, c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// verdict reports whether a probe found no problem.
func accepted(ps []content.Problem) bool { return len(ps) == 0 }

func mustPanicsWith(f func()) (ok bool) {
	defer func() { ok = recover() != nil }()
	f()
	return false
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-094, FR-030, FR-114 - the decoder's id rule and every
// ParseID and typed-id parser give the same verdict for every row, and
// MustID panics iff ParseID errs.
// Path: per case.
func TestAgreementIDs(t *testing.T) {
	t.Parallel()
	valid := []string{"a", "a1", "a_b", "a" + strings.Repeat("b", 63), "acid_arrow_2", "z9_"}
	invalid := []string{"", "A", "2a", "_a", "a-b", " a", "a ", "a b", "a\nb", "é", "aB", "a" + strings.Repeat("b", 64),
		strings.Repeat("A", 65), "a\x00", "a\xff", "   ", "a\u202eb", "1", "-"}
	parsers := map[string]func(string) error{
		"ParseID":           func(s string) error { _, err := content.ParseID(s); return err },
		"ParseSpeciesID":    func(s string) error { _, err := content.ParseSpeciesID(s); return err },
		"ParseBackgroundID": func(s string) error { _, err := content.ParseBackgroundID(s); return err },
		"ParseClassID":      func(s string) error { _, err := content.ParseClassID(s); return err },
		"ParseSubclassID":   func(s string) error { _, err := content.ParseSubclassID(s); return err },
		"ParseFeatID":       func(s string) error { _, err := content.ParseFeatID(s); return err },
		"ParseSpellID":      func(s string) error { _, err := content.ParseSpellID(s); return err },
		"ParseItemID":       func(s string) error { _, err := content.ParseItemID(s); return err },
		"ParseMonsterID":    func(s string) error { _, err := content.ParseMonsterID(s); return err },
		"ParseFeatureID":    func(s string) error { _, err := content.ParseFeatureID(s); return err },
		"ParseMagicItemID":  func(s string) error { _, err := content.ParseMagicItemID(s); return err },
	}
	check := func(id string, want bool) {
		t.Helper()
		label := strconv.QuoteToASCII(id)
		if len(label) > 30 {
			label = label[:30] + "..."
		}
		if got := accepted(content.ProbeField("id", jsonString(id))); got != want {
			t.Errorf("decoder id %s accepted = %v, want %v", label, got, want)
		}
		for name, parse := range parsers {
			if got := parse(id) == nil; got != want {
				t.Errorf("%s(%s) accepted = %v, want %v", name, label, got, want)
			}
		}
		if panicked := mustPanicsWith(func() { content.MustID(id) }); panicked == want {
			t.Errorf("MustID(%s) panicked = %v, want %v", label, panicked, !want)
		}
	}
	for _, id := range valid {
		check(id, true)
	}
	for _, id := range invalid {
		check(id, false)
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-094, FR-027, FR-028, FR-116, FR-151 and doc/plans/plan-phase-2.md P14 -
// the decoder's name and text rules and the building-block constructors give the
// same verdict for every row.
// Path: per case.
func TestAgreementNamesAndTexts(t *testing.T) {
	t.Parallel()
	names := []struct {
		s    string
		want bool
	}{
		{"N", true}, {"Acid Arrow", true}, {"Thieves’ Cant", true}, {"a\u200bb", true}, {"\U0001F600", true},
		{"", false}, {" ", false}, {" N", false}, {"N ", false}, {"N\n", false}, {"a\tb", false}, {"a\nb", false}, {"a\u202eb", false},
		{"a\x00b", false}, {"a\xffb", false}, {"a\u2028b", false}, {"a\u0085b", false}, {"\u00a0N", false},
	}
	for _, c := range names {
		verdicts := map[string]bool{
			"decoder name":        accepted(content.ProbeField("name", jsonString(c.s))),
			"NewNamedText":        func() bool { _, err := content.NewNamedText(c.s, "t"); return err == nil }(),
			"NewTextEntry":        func() bool { _, err := content.NewTextEntry(c.s, "t"); return err == nil }(),
			"SpecificTool":        func() bool { _, err := content.NewSpecificToolProficiency(c.s); return err == nil }(),
			"FeaturePrerequisite": func() bool { _, err := content.NewFeaturePrerequisite(c.s); return err == nil }(),
		}
		for who, got := range verdicts {
			if got != c.want {
				t.Errorf("%s(%q) accepted = %v, want %v", who, c.s, got, c.want)
			}
		}
	}
	texts := []struct {
		s    string
		want bool
	}{
		{"t", true}, {"a\nb", true}, {"  x  ", true}, {"é", true}, {"a\u200bb", true}, {"line one\nline two\n", true},
		{"", false}, {" \n ", false}, {"\t", false}, {"a\tb", false}, {"a\rb", false}, {"a\u202eb", false}, {"a\x00", false}, {"a\xff", false},
		{"a\u2029b", false}, {"a\u061cb", false}, {"a\u2066b", false}, {"a\u007fb", false},
	}
	for _, c := range texts {
		verdicts := map[string]bool{
			"decoder text":     accepted(content.ProbeField("text", jsonString(c.s))),
			"NewNamedText":     func() bool { _, err := content.NewNamedText("n", c.s); return err == nil }(),
			"NewTextParagraph": func() bool { _, err := content.NewTextParagraph(c.s); return err == nil }(),
			"NewTextEntry":     func() bool { _, err := content.NewTextEntry("n", c.s); return err == nil }(),
			"ChoiceTool":       func() bool { _, err := content.NewChoiceToolProficiency(c.s); return err == nil }(),
		}
		for who, got := range verdicts {
			if got != c.want {
				t.Errorf("%s(%q) accepted = %v, want %v", who, c.s, got, c.want)
			}
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-094, FR-034, FR-115 - the decoder's source object and
// NewSourceCitation agree for every row, and the decoded citation equals the constructed one.
// Path: per case.
func TestAgreementSource(t *testing.T) {
	t.Parallel()
	rows := []struct {
		document, section string
		page              int64
		want              bool
	}{
		{"SRD 5.2.1", "Damage Rolls", 16, true},
		{"d", "s", 1, true},
		{"d", "s", content.SourceMaxPage, true},
		{"d", "s", 0, false},
		{"d", "s", -1, false},
		{"d", "s", content.SourceMaxPage + 1, false},
		{"d", "  ", 1, false},
		{"  ", "s", 1, false},
		{"", "s", 1, false},
		{"d", "", 1, false},
		{"d\n", "s", 1, false},
		{"d", "a\nb", 1, false},
		{"d\xff", "s", 1, false},
		{"d", "s\u202e", 1, false},
		{"d\x00", "s", 1, false},
		{"SRD 5.2.1", "Thieves’ Cant", 20, true},
	}
	for _, r := range rows {
		obj := `{"document":` + jsonString(r.document) + `,"section":` + jsonString(r.section) + `,"page":` + strconv.FormatInt(r.page, 10) + `}`
		problems, cite, ok := content.ProbeSource(obj)
		if ok != r.want || accepted(problems) != r.want {
			t.Errorf("decoder source %s: ok=%v problems=%v, want accepted=%v", obj, ok, problems, r.want)
		}
		if int64(int(r.page)) != r.page {
			continue // the page does not fit this platform's int; the decoder row above covers it
		}
		built, err := content.NewSourceCitation(r.document, r.section, int(r.page))
		if (err == nil) != r.want {
			t.Errorf("NewSourceCitation(%q, %q, %d) error = %v, want accepted=%v", r.document, r.section, r.page, err, r.want)
		}
		if r.want && cite != built {
			t.Errorf("decoded citation %v != constructed %v", cite, built)
		}
		if panicked := mustPanicsWith(func() { content.MustSourceCitation(r.document, r.section, int(r.page)) }); panicked == r.want {
			t.Errorf("MustSourceCitation(%q, %q, %d) panicked = %v", r.document, r.section, r.page, panicked)
		}
	}
}

// vocabulary is one closed enum for the agreement test.
type vocabulary struct {
	name    string
	members []string
	parse   func(string) error
	probe   func(string) []content.Problem
}

func vocab[T interface {
	~int
	Valid() bool
	String() string
}](name string, parse func(string) (T, error)) vocabulary {
	var members []string
	for v := T(1); v.Valid(); v++ {
		members = append(members, v.String())
	}
	return vocabulary{
		name:    name,
		members: members,
		parse:   func(s string) error { _, err := parse(s); return err },
		probe:   func(js string) []content.Problem { return content.ProbeEnum(js, parse) },
	}
}

func vocabularies() []vocabulary {
	return []vocabulary{
		vocab("Skill", content.ParseSkill), vocab("DamageType", content.ParseDamageType), vocab("CreatureType", content.ParseCreatureType),
		vocab("CreatureSize", content.ParseCreatureSize), vocab("ConditionKind", content.ParseConditionKind), vocab("Language", content.ParseLanguage),
		vocab("RechargeTrigger", content.ParseRechargeTrigger), vocab("WeaponProperty", content.ParseWeaponProperty),
		vocab("MasteryProperty", content.ParseMasteryProperty), vocab("FeatCategory", content.ParseFeatCategory),
		vocab("HitPointDie", content.ParseHitPointDie), vocab("LightLevel", content.ParseLightLevel), vocab("DarknessKind", content.ParseDarknessKind),
		vocab("Obscurement", content.ParseObscurement), vocab("SpecialSense", content.ParseSpecialSense), vocab("Alignment", content.ParseAlignment),
		vocab("DurationUnit", content.ParseDurationUnit),
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-094, FR-025, FR-105 - for every vocabulary the decoder
// and ParseX accept exactly the members and reject near misses alike.
// Path: per case.
func TestAgreementVocabularies(t *testing.T) {
	t.Parallel()
	for _, v := range vocabularies() {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			if len(v.members) == 0 {
				t.Fatalf("no members found")
			}
			for _, m := range v.members {
				if v.parse(m) != nil || !accepted(v.probe(jsonString(m))) {
					t.Errorf("member %q: ParseX or the decoder rejected it", m)
				}
				for _, near := range []string{strings.ToLower(m) + "x", " " + m, m + " ", strings.ToUpper(m) + "!", m + "\x00", m[:len(m)-1] + "\xff"} {
					if v.parse(near) == nil || accepted(v.probe(jsonString(near))) {
						t.Errorf("near miss %q accepted by ParseX or the decoder", near)
					}
				}
			}
			for _, s := range []string{"", "x", "Unknown", "\u202e"} {
				if (v.parse(s) == nil) != accepted(v.probe(jsonString(s))) {
					t.Errorf("%q: ParseX and the decoder disagree", s)
				}
			}
			// A non-string is wrongType, a different class of problem than a bad member.
			ps := v.probe("1")
			if len(ps) != 1 || ps[0].Kind() != content.ProblemWrongType {
				t.Errorf("a number = %v, want wrongType", ps)
			}
			ps = v.probe(jsonString("zzz"))
			if len(ps) != 1 || ps[0].Kind() != content.ProblemInvalidValue || !errors.Is(v.parse("zzz"), srderr.ErrInvalidInput) {
				t.Errorf("a non-member = %v, want invalidValue and an ErrInvalidInput parse error", ps)
			}
		})
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-094, FR-021 - the integer rule reads whole numbers by
// their value on every platform and every spelling.
// Path: per case.
func TestAgreementIntegerSpellings(t *testing.T) {
	t.Parallel()
	rows := []struct {
		lit  string
		want content.ProblemKind // 0 accepted
	}{
		{"2", 0}, {"2.0", 0}, {"2e0", 0}, {"20e-1", 0}, {"-0", content.ProblemOutOfRange}, {"2.5", content.ProblemWrongType},
		{"\"2\"", content.ProblemWrongType}, {"true", content.ProblemWrongType}, {"null", content.ProblemWrongType},
		{"[]", content.ProblemWrongType}, {"12345678901234567890", content.ProblemOutOfRange}, {"1e999999999", content.ProblemOutOfRange},
		{"1e-999999999", content.ProblemWrongType}, {"9007199254740991", content.ProblemOutOfRange}, {"9007199254740992", content.ProblemOutOfRange},
		{"0", content.ProblemOutOfRange}, {"3", 0}, {"1", 0}, {"20", 0}, {"21", content.ProblemOutOfRange},
	}
	for _, r := range rows {
		ps := content.ProbeField("integer:1:20", r.lit)
		switch {
		case r.want == 0 && len(ps) != 0, r.want != 0 && (len(ps) != 1 || ps[0].Kind() != r.want):
			t.Errorf("integer %s -> %v, want kind %v", r.lit, ps, r.want)
		}
	}
	if ps := content.ProbeField("integer:-9007199254740991:9007199254740991", "9007199254740991"); len(ps) != 0 {
		t.Errorf("2^53 - 1 within a wide range = %v, want accepted", ps)
	}
	if ps := content.ProbeField("integer:-9007199254740991:9007199254740991", "-9007199254740992"); len(ps) != 1 || ps[0].Kind() != content.ProblemOutOfRange {
		t.Errorf("-2^53 = %v, want outOfRange", ps)
	}
}
