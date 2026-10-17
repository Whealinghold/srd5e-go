package archtest

import (
	"bytes"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// noticeCoveredPaths are the paths NOTICE.md may name as covered (A12). A
// self-test keeps it equal to the first sentence of FR-033 of the decisions spec.
func noticeCoveredPaths() []string { return []string{"content/data/"} }

const dataDir = "content/data"

// pathTokens returns the backticked tokens of text that are directory paths
// (they end in "/").
func pathTokens(text string) []string {
	dir := regexp.MustCompile(`^[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*/$`)
	var out []string
	for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(text, -1) {
		if dir.MatchString(m[1]) {
			out = append(out, m[1])
		}
	}
	return out
}

// placeholder reports whether a file name does not count as bundled content.
func placeholder(name string) bool { return name == ".gitkeep" || name == "README.md" }

// dataFiles lists the files below content/data; nil when it does not exist.
func dataFiles(m Module) []string {
	var out []string
	_ = fs.WalkDir(m.FS, dataDir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

// checkEmbedded covers A12: embed confinement, the content/data layout and the
// NOTICE.md covered paths. The layout and notice checks are vacuous until
// content/data holds a file.
func checkEmbedded(m Module) []Violation {
	var vs []Violation
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			for _, imp := range f.Imports() {
				if imp.Path == "embed" && pkg.Dir != dataDir {
					vs = append(vs, at(f, imp.Pos, RuleEmbed, "only %s may import embed", dataDir))
				}
			}
		}
	}
	files := dataFiles(m)
	if len(files) == 0 {
		return vs
	}
	vs = append(vs, checkDataLayout(m, files)...)
	bundled := slices.ContainsFunc(files, func(p string) bool { return !placeholder(path.Base(p)) })
	if bundled {
		vs = append(vs, checkNoticePaths(m)...)
	}
	return vs
}

// checkDataLayout requires content/data to hold only JSON, placeholders, test
// files and exactly one Go file, which carries the go:embed directive.
func checkDataLayout(m Module, files []string) []Violation {
	var vs []Violation
	var goFiles []string
	json := false
	for _, p := range files {
		name := path.Base(p)
		switch {
		case strings.HasSuffix(name, ".json"):
			json = true
		case strings.HasSuffix(name, "_test.go") || placeholder(name):
		case strings.HasSuffix(name, ".go"):
			goFiles = append(goFiles, p)
		default:
			vs = append(vs, Violation{Rule: RuleEmbed, File: p, Msg: "content/data holds only .json files and the go:embed Go file"})
		}
	}
	switch {
	case len(goFiles) > 1:
		vs = append(vs, Violation{Rule: RuleEmbed, File: dataDir, Msg: "content/data must have exactly one non-test Go file; found " + strings.Join(goFiles, ", ")})
	case len(goFiles) == 0 && json:
		vs = append(vs, Violation{Rule: RuleEmbed, File: dataDir, Msg: "content/data holds JSON but no Go file with go:embed"})
	case len(goFiles) == 1:
		src, _ := fs.ReadFile(m.FS, goFiles[0])
		if !bytes.Contains(src, []byte("//go:embed")) {
			vs = append(vs, Violation{Rule: RuleEmbed, File: goFiles[0], Msg: "the Go file in content/data must carry the //go:embed directive"})
		}
	}
	return vs
}

// checkNoticePaths requires the path tokens of the Scope section of NOTICE.md
// to equal noticeCoveredPaths. A missing NOTICE.md or Scope section is another
// spec's concern and reports nothing here.
func checkNoticePaths(m Module) []Violation {
	raw, err := fs.ReadFile(m.FS, "NOTICE.md")
	if err != nil {
		return nil
	}
	var vs []Violation
	var seen []string
	heading, inScope := 0, false
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "## ") {
			if inScope {
				break
			}
			if strings.TrimSpace(line[3:]) == "Scope" {
				heading, inScope = i+1, true
			}
			continue
		}
		if !inScope {
			continue
		}
		for _, tok := range pathTokens(line) {
			seen = append(seen, tok)
			if !slices.Contains(noticeCoveredPaths(), tok) {
				vs = append(vs, Violation{Rule: RuleNoticePaths, File: "NOTICE.md", Line: i + 1, Msg: "Scope names " + tok + ", which is not a covered path"})
			}
		}
	}
	if heading == 0 {
		return nil
	}
	for _, want := range noticeCoveredPaths() {
		if !slices.Contains(seen, want) {
			vs = append(vs, Violation{Rule: RuleNoticePaths, File: "NOTICE.md", Line: heading, Msg: "Scope must name covered path " + want})
		}
	}
	return vs
}
