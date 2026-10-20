package content_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Compile-time FR-114 assertions: every id with a kind is a KeyedID, and
// SubclassID and FeatureID are TypedIDs only.
var (
	_ content.KeyedID = content.SpeciesID("")
	_ content.KeyedID = content.BackgroundID("")
	_ content.KeyedID = content.ClassID("")
	_ content.KeyedID = content.FeatID("")
	_ content.KeyedID = content.SpellID("")
	_ content.KeyedID = content.ItemID("")
	_ content.KeyedID = content.MonsterID("")
	_ content.KeyedID = content.MagicItemID("")
	_ content.TypedID = content.SubclassID("")
	_ content.TypedID = content.FeatureID("")
)

// Requirement: doc/specs/content-schema-and-decoder.md FR-114 - each typed id
// has ParseXID, MustXID, String, Valid and, when it addresses a top-level entry,
// Key and Missing with the kind of the mapping table; the parser agrees with
// ParseID on every input and MustXID panics with the ParseID error.
// Path: per case.
func TestTypedIDs(t *testing.T) {
	t.Parallel()
	type id = content.TypedID
	cases := []struct {
		name  string
		kind  content.Kind // 0: no key
		parse func(string) (id, error)
		must  func(string) id
		conv  func(string) id // plain conversion, bypassing the parser
	}{
		{"SpeciesID", content.KindSpecies, wrap(content.ParseSpeciesID), wrapMust(content.MustSpeciesID), func(s string) id { return content.SpeciesID(s) }},
		{"BackgroundID", content.KindBackground, wrap(content.ParseBackgroundID), wrapMust(content.MustBackgroundID), func(s string) id { return content.BackgroundID(s) }},
		{"ClassID", content.KindCharacterClass, wrap(content.ParseClassID), wrapMust(content.MustClassID), func(s string) id { return content.ClassID(s) }},
		{"SubclassID", 0, wrap(content.ParseSubclassID), wrapMust(content.MustSubclassID), func(s string) id { return content.SubclassID(s) }},
		{"FeatID", content.KindFeat, wrap(content.ParseFeatID), wrapMust(content.MustFeatID), func(s string) id { return content.FeatID(s) }},
		{"SpellID", content.KindSpell, wrap(content.ParseSpellID), wrapMust(content.MustSpellID), func(s string) id { return content.SpellID(s) }},
		{"ItemID", content.KindEquipment, wrap(content.ParseItemID), wrapMust(content.MustItemID), func(s string) id { return content.ItemID(s) }},
		{"MonsterID", content.KindMonster, wrap(content.ParseMonsterID), wrapMust(content.MustMonsterID), func(s string) id { return content.MonsterID(s) }},
		{"FeatureID", 0, wrap(content.ParseFeatureID), wrapMust(content.MustFeatureID), func(s string) id { return content.FeatureID(s) }},
		{"MagicItemID", content.KindMagicItem, wrap(content.ParseMagicItemID), wrapMust(content.MustMagicItemID), func(s string) id { return content.MagicItemID(s) }},
	}
	inputs := []string{"a", "acid_arrow_2", strings.Repeat("a", 64), "", strings.Repeat("a", 65), strings.Repeat("A", 65), "Acid", "1a", "a b", "é"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, in := range inputs {
				want, wantErr := content.ParseID(in)
				got, err := c.parse(in)
				if (err == nil) != (wantErr == nil) || got.String() != want.String() || got.Valid() != (wantErr == nil) {
					t.Errorf("Parse(%q) = (%q, %v), want agreement with ParseID (%q, %v) [unhappy (rule interaction)]", in, got, err, want, wantErr)
				}
				var ie, wie *content.IDError
				if wantErr != nil && (!errors.As(err, &ie) || !errors.As(wantErr, &wie) || ie.Problem != wie.Problem || ie.Value != in || !errors.Is(err, srderr.ErrInvalidInput)) {
					t.Errorf("Parse(%q) error = %v, want the ParseID error %v", in, err, wantErr)
				}
				panicked := panicValue(func() { c.must(in) })
				if (panicked != nil) != (wantErr != nil) {
					t.Errorf("Must(%q) panicked = %v, ParseID error = %v", in, panicked, wantErr)
				}
				if pe, ok := panicked.(error); wantErr != nil && (!ok || !errors.Is(pe, wie)) {
					t.Errorf("Must(%q) panic = %v, want the ParseID error", in, panicked)
				}
				if c.conv(in).Valid() != (wantErr == nil) {
					t.Errorf("Valid() on a converted %q = %v, want %v", in, c.conv(in).Valid(), wantErr == nil)
				}
			}
			valid := c.conv("a")
			k, keyed := valid.(content.KeyedID)
			if (c.kind != 0) != keyed {
				t.Fatalf("KeyedID = %v, want %v [happy]", keyed, c.kind != 0)
			}
			if !keyed {
				return
			}
			wantKey := content.NewKey(c.kind, content.MustID("a"))
			if k.Key() != wantKey || k.Missing() != content.NewMissingContent(c.kind, content.MustID("a")) || k.Missing().Key() != wantKey {
				t.Errorf("Key() = %v, Missing() = %v, want kind %v", k.Key(), k.Missing(), c.kind)
			}
			bad := c.conv("A").(content.KeyedID)
			if !bad.Key().IsZero() || !bad.Missing().Key().IsZero() {
				t.Errorf("an invalid converted id must have the zero Key [unhappy (invalid input)]")
			}
		})
	}
}

func wrap[T content.TypedID](f func(string) (T, error)) func(string) (content.TypedID, error) {
	return func(s string) (content.TypedID, error) { v, err := f(s); return v, err }
}

func wrapMust[T content.TypedID](f func(string) T) func(string) content.TypedID {
	return func(s string) content.TypedID { return f(s) }
}

// FuzzParseTypedIDs checks that every typed id parser agrees with ParseID.
//
// Requirement: doc/specs/content-schema-and-decoder.md FR-114 - each ParseXID
// never panics, accepts exactly what ParseID accepts and round-trips.
// Path: unhappy (invalid input).
func FuzzParseTypedIDs(f *testing.F) {
	for _, s := range []string{"", "a", "a_b1", "1a", "A", "a-b", "a\x00", "a\xff", strings.Repeat("a", 65)} {
		f.Add(s)
	}
	parsers := map[string]func(string) (string, error){
		"Species":    func(s string) (string, error) { v, err := content.ParseSpeciesID(s); return v.String(), err },
		"Background": func(s string) (string, error) { v, err := content.ParseBackgroundID(s); return v.String(), err },
		"Class":      func(s string) (string, error) { v, err := content.ParseClassID(s); return v.String(), err },
		"Subclass":   func(s string) (string, error) { v, err := content.ParseSubclassID(s); return v.String(), err },
		"Feat":       func(s string) (string, error) { v, err := content.ParseFeatID(s); return v.String(), err },
		"Spell":      func(s string) (string, error) { v, err := content.ParseSpellID(s); return v.String(), err },
		"Item":       func(s string) (string, error) { v, err := content.ParseItemID(s); return v.String(), err },
		"Monster":    func(s string) (string, error) { v, err := content.ParseMonsterID(s); return v.String(), err },
		"Feature":    func(s string) (string, error) { v, err := content.ParseFeatureID(s); return v.String(), err },
		"MagicItem":  func(s string) (string, error) { v, err := content.ParseMagicItemID(s); return v.String(), err },
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, want := content.ParseID(s)
		for name, parse := range parsers {
			got, err := parse(s)
			var ie *content.IDError
			if (err != nil) != (want != nil) || (err != nil && (!errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput))) || (err == nil && got != s) {
				t.Fatalf("%s: %q gave %q, %v; ParseID gave %v", name, s, got, err, want)
			}
		}
	})
}
