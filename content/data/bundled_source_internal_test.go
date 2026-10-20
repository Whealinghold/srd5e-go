package data

import (
	"errors"
	"io/fs"
	"maps"
	"testing"
	"testing/fstest"
)

// openFails is a file system whose directory lists but whose files cannot be opened.
type openFails struct{ fstest.MapFS }

func (openFails) Open(string) (fs.File, error)    { return nil, errors.New("open failed") }
func (openFails) ReadFile(string) ([]byte, error) { return nil, errors.New("read failed") }

// readFails is a file system whose root cannot be listed.
type readFails struct{ fstest.MapFS }

func (readFails) ReadDir(string) ([]fs.DirEntry, error) { return nil, errors.New("list failed") }
func (readFails) Open(string) (fs.File, error)          { return nil, errors.New("open failed") }

// Requirement: doc/specs/content-source-port.md FR-062 and doc/plans/plan-phase-2.md 5.6 - the index read from a
// fixture file system: an empty directory is an empty index (the real embed can never be empty), directories are
// skipped, and every file is read by its name.
// Path: unhappy (empty case) / happy.
func TestReadDocumentsFromAFixtureFS(t *testing.T) {
	t.Parallel()
	if got := readDocuments(fstest.MapFS{}); len(got) != 0 {
		t.Errorf("readDocuments(empty) = %v, want an empty index", got)
	}
	fsys := fstest.MapFS{"a.json": {Data: []byte("A")}, "b.json": {Data: []byte("B")}, "sub/c.json": {Data: []byte("C")}}
	if got, want := readDocuments(fsys), map[string]string{"a.json": "A", "b.json": "B"}; !maps.Equal(got, want) {
		t.Errorf("readDocuments = %v, want %v (the subdirectory is skipped)", got, want)
	}
}

// Requirement: doc/specs/content-source-port.md FR-061 - an index that cannot be read becomes documents with empty
// text, which the decoder rejects, so a broken index is a bad bundle and never a silently smaller one.
// Path: unhappy (infrastructure failure of the embed; fixture only).
func TestReadDocumentsFailuresBecomeEmptyTexts(t *testing.T) {
	t.Parallel()
	files := fstest.MapFS{"a.json": {Data: []byte("A")}}
	if got, want := readDocuments(openFails{files}), map[string]string{"a.json": ""}; !maps.Equal(got, want) {
		t.Errorf("readDocuments(unreadable file) = %v, want %v", got, want)
	}
	if got, want := readDocuments(readFails{files}), map[string]string{".": ""}; !maps.Equal(got, want) {
		t.Errorf("readDocuments(unlistable root) = %v, want %v", got, want)
	}
}
