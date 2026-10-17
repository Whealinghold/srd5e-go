package archtest

import (
	"io/fs"
	"path"
	"strings"
)

// consumerNames returns the names no file may mention. The organisation that
// owns the module appears only in the module path; it is spelled in two parts
// so that this file does not match its own rule.
func consumerNames() []string { return []string{"whealing" + "hold"} }

// exemptFromNames lists the files allowed to name the organisation: only the
// root AGENTS.md, which states the rule and why the module owner is named.
func exemptFromNames(p string) bool { return p == "AGENTS.md" }

// checkConsumerNames reports every mention of a consumer name in a text file,
// ignoring occurrences inside the module path itself (A10).
func checkConsumerNames(m Module, names []string) []Violation {
	var vs []Violation
	_ = fs.WalkDir(m.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entries are skipped; Load already read the Go files
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if exemptFromNames(p) || path.Base(p) == "go.mod" {
			return nil
		}
		raw, err := fs.ReadFile(m.FS, p)
		if err != nil {
			return nil
		}
		text := strings.ToLower(strings.ReplaceAll(string(raw), m.Path, ""))
		for _, n := range names {
			if strings.Contains(text, n) {
				vs = append(vs, Violation{Rule: RuleConsumer, File: p, Msg: "mentions consumer name " + n})
			}
		}
		return nil
	})
	return vs
}
