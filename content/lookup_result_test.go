package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-source-port.md FR-015, FR-016 - LookupResult is
// closed over LookupFound and LookupNotFound (an exhaustive switch with no
// default), the variants compare by value, and LookupNotFound.Missing() is the
// MissingContent of its key.
// Path: happy.
func TestLookupResultVariants(t *testing.T) {
	t.Parallel()
	key := plain(content.KindSpell, "a").Key()
	results := []content.LookupResult{
		content.LookupFound{Entry: plain(content.KindSpell, "a")},
		content.LookupNotFound{Key: key},
	}
	var found, notFound int
	for _, r := range results {
		switch v := r.(type) {
		case content.LookupFound:
			found++
			if v.Entry.Key() != key {
				t.Errorf("LookupFound entry key = %v, want %v", v.Entry.Key(), key)
			}
		case content.LookupNotFound:
			notFound++
			if v.Key != key || v.Missing() != content.NewMissingContent(content.KindSpell, key.ID()) {
				t.Errorf("LookupNotFound = %+v, Missing = %v, want key %v", v, v.Missing(), key)
			}
		}
	}
	if found != 1 || notFound != 1 {
		t.Errorf("switch saw %d found and %d not found, want 1 each", found, notFound)
	}
	if results[0] != (content.LookupFound{Entry: plain(content.KindSpell, "a")}) || results[0] == results[1] {
		t.Error("variants do not compare by value")
	}
	if (content.LookupNotFound{Key: key}) == (content.LookupNotFound{Key: plain(content.KindFeat, "a").Key()}) {
		t.Error("LookupNotFound for different keys compare equal")
	}
}
