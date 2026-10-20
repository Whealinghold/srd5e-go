package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
)

// Requirement: doc/specs/content-source-port.md FR-027, FR-047 - InMemorySource
// is registered with the shared contract suite (no failing group: it has no
// backend to fail).
// Path: happy.
func TestInMemorySourceContract(t *testing.T) {
	contenttest.RunSourceContract(t, func(t *testing.T, entries []content.Entry) content.Source {
		t.Helper()
		s, err := content.NewInMemorySource(entries)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}, nil)
}
