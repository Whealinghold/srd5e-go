package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// Requirement: doc/specs/content-thin-slice.md FR-067 and doc/plans/plan-phase-2.md 5.7 - the whole slice through the
// public API only: a document with problems is rejected with every problem, the bundled slice loads, and present and
// absent keys are looked up.
// Path: happy / unhappy (invalid input: a document with problems; missing content: absent keys).
func TestWholeContentSlice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	loader, err := content.NewLoader(content.JSONDecoder{})
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	res, err := loader.LoadSet(ctx, content.LoadSetCommand{Document: []byte(`{"schemaVersion": 2}`)})
	rej, ok := res.(content.LoadSetRejected)
	if err != nil || !ok || len(rej.Problems()) == 0 {
		t.Fatalf("LoadSet(bad document) = (%v, %v), want LoadSetRejected with problems", res, err)
	}
	if got := rej.Problems()[0].Kind(); got != content.ProblemUnsupportedVersion {
		t.Errorf("first problem kind = %v, want %v", got, content.ProblemUnsupportedVersion)
	}

	src, err := data.NewBundledSource()
	if err != nil {
		t.Fatalf("NewBundledSource: %v", err)
	}
	total := 0
	for k := content.KindSpecies; k.Valid(); k++ {
		list, err := src.ListKind(ctx, k)
		if err != nil {
			t.Fatalf("ListKind(%v): %v", k, err)
		}
		total += len(list)
	}
	if total != 17 {
		t.Errorf("bundled entries = %d, want 17", total)
	}

	present := content.NewKey(content.KindSpell, content.MustID("fireball"))
	absent := content.NewKey(content.KindSpell, content.MustID("not_in_the_slice"))
	for key, wantFound := range map[content.Key]bool{present: true, absent: false} {
		got, err := src.Lookup(ctx, key)
		if err != nil {
			t.Fatalf("Lookup(%v): %v", key, err)
		}
		switch r := got.(type) {
		case content.LookupFound:
			if !wantFound || r.Entry.Key() != key {
				t.Errorf("Lookup(%v) found %v", key, r.Entry)
			}
		case content.LookupNotFound:
			if wantFound || r.Key != key {
				t.Errorf("Lookup(%v) not found", key)
			}
		}
	}
}
