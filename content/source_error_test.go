package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-source-port.md FR-018, FR-037, FR-039 -
// SourceError unwraps to the exact Err instance, is found by errors.Is and
// errors.As as a pointer for both Query variants, and Error and Unwrap do not
// panic for a nil Query or nil Err.
// Path: per case.
func TestSourceError(t *testing.T) {
	t.Parallel()
	backend := errors.New("backend")
	cases := []struct {
		name, path string
		err        *content.SourceError
		wantErr    error
		wantMsg    string
	}{
		{"lookup", "unhappy (collaborator failure)", &content.SourceError{Query: content.LookupQuery{Key: plain(content.KindSpell, "a").Key()}, Err: backend}, backend, "content: source lookup failed: backend"},
		{"list kind", "unhappy (collaborator failure)", &content.SourceError{Query: content.ListKindQuery{Kind: content.KindFeat}, Err: backend}, backend, "content: source list_kind failed: backend"},
		{"nil query", "unhappy (invalid design)", &content.SourceError{Err: backend}, backend, "content: source invalid query failed: backend"},
		{"nil err", "unhappy (invalid design)", &content.SourceError{Query: content.ListKindQuery{}}, nil, "content: source list_kind failed"},
		{"both nil", "unhappy (invalid design)", &content.SourceError{}, nil, "content: source invalid query failed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if !errors.Is(c.err.Unwrap(), c.wantErr) {
				t.Errorf("Unwrap() = %v, want the same instance %v [%s]", c.err.Unwrap(), c.wantErr, c.path)
			}
			if got := c.err.Error(); got != c.wantMsg {
				t.Errorf("Error() = %q, want %q [%s]", got, c.wantMsg, c.path)
			}
			if c.err.Error() == "" {
				t.Error("Error() is empty")
			}
			var wrapped error = c.err
			var se *content.SourceError
			if !errors.As(wrapped, &se) || se != c.err {
				t.Errorf("errors.As did not find the *SourceError")
			}
			if c.wantErr != nil && !errors.Is(wrapped, c.wantErr) {
				t.Errorf("errors.Is does not reach the backend error")
			}
		})
	}
}
