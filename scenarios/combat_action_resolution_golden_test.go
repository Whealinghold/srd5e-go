package scenarios_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The flags are defined here, in the test (go.md section 8): -update rewrites the golden file, -transcript is how the runner asks the
// child process for the transcript. A golden file is never rewritten without -update.
var (
	update         = flag.Bool("update", false, "rewrite the golden files of this package")
	transcriptMode = flag.Bool("transcript", false, "print the transcript of table 8.1 between markers (used by the subprocess runner)")
)

const (
	transcriptBegin = "-----BEGIN TRANSCRIPT-----\n"
	transcriptEnd   = "-----END TRANSCRIPT-----\n"
	goldenPath      = "testdata/golden/combat.txt"
)

// extractTranscript returns the text between the markers of the child's stdout.
func extractTranscript(out string) (string, error) {
	_, rest, ok := strings.Cut(out, transcriptBegin)
	if !ok {
		return "", errors.New("begin marker missing")
	}
	body, _, ok := strings.Cut(rest, transcriptEnd)
	if !ok {
		return "", errors.New("end marker missing")
	}
	return body, nil
}

// Requirement: doc/specs/combat-action-resolution.md FR-109 - the scripted round of table 8.1 runs in a separate process (the test binary
// re-executed with os/exec, test code only), exits 0, and its transcript equals the golden file scenarios/testdata/golden/combat.txt byte
// for byte. Only the -update flag rewrites the file.
// Path: happy.
func TestCombatActionResolutionGolden(t *testing.T) {
	t.Parallel()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCombatActionResolutionScenario$", "-test.count=1", "-transcript")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("child process: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	got, err := extractTranscript(stdout.String())
	if err != nil {
		t.Fatalf("child stdout: %v\n%s", err, stdout.String())
	}
	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file: %v (run with -update to create it)", err)
	}
	if got != string(want) {
		t.Errorf("transcript differs from %s (run with -update to accept):\ngot:\n%s\nwant:\n%s", goldenPath, got, want)
	}
}

// Requirement: per case
// Path: per case.
func TestExtractTranscript(t *testing.T) {
	t.Parallel()
	const fr109 = "doc/specs/combat-action-resolution.md FR-109 - a child whose stdout lacks a marker is rejected, so empty output cannot match an empty golden file"
	cases := []struct {
		name, requirement, path, out, want string
		fail                               bool
	}{
		{"ok", fr109, "happy", "x\n" + transcriptBegin + "1: a\n" + transcriptEnd + "PASS\n", "1: a\n", false},
		{"no begin", fr109, "unhappy (malformed child output)", "1: a\n" + transcriptEnd, "", true},
		{"no end", fr109, "unhappy (malformed child output)", transcriptBegin + "1: a\n", "", true},
		{"empty", fr109, "unhappy (malformed child output)", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractTranscript(c.out)
			if (err != nil) != c.fail || got != c.want {
				t.Errorf("extractTranscript = %q, %v; want %q, fail=%v", got, err, c.want, c.fail)
			}
		})
	}
}
