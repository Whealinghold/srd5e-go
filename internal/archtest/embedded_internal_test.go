package archtest

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// firstSentence returns the first sentence of the FR-033 row of the decisions
// spec, the text noticeCoveredPaths copies.
func firstSentence(t *testing.T, spec string) string {
	t.Helper()
	for _, line := range strings.Split(spec, "\n") {
		if strings.HasPrefix(line, "| FR-033 |") {
			cell := strings.Split(line, "|")[2]
			s, _, _ := strings.Cut(cell, ". ")
			return s
		}
	}
	t.Fatal("FR-033 row not found in shared-types-and-ownership.md")
	return ""
}

func TestNoticeCoveredPathsMatchDecisionsSpec(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("../../doc/specs/shared-types-and-ownership.md")
	if err != nil {
		t.Fatal(err)
	}
	got := pathTokens(firstSentence(t, string(raw)))
	if !slices.Equal(got, noticeCoveredPaths()) {
		t.Fatalf("FR-033 first sentence names %v; noticeCoveredPaths is %v", got, noticeCoveredPaths())
	}
}

func TestSelfTestDetectsDrift(t *testing.T) {
	t.Parallel()
	spec := "| FR-033 | `NOTICE.md` covers exactly two paths: `content/data/` and `extra/`. Later `doc/research/`. | x | y |"
	if slices.Equal(pathTokens(firstSentence(t, spec)), noticeCoveredPaths()) {
		t.Fatal("a drifted FR-033 sentence must not equal the constant")
	}
}

func TestPathTokens(t *testing.T) {
	t.Parallel()
	got := pathTokens("`a/` `NOTICE.md` `b/c/` `d e/` text `f`")
	if want := []string{"a/", "b/c/"}; !slices.Equal(got, want) {
		t.Fatalf("pathTokens = %v, want %v", got, want)
	}
}
