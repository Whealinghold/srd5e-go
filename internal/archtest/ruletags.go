package archtest

import (
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const tagLabel = "RULE-TAG:"

// checkRuleTags covers A13: every doc/rules/*.md file carries exactly one
// well-formed tag line, no tag value appears anywhere else in the repository,
// and every rule file is listed in AGENTS.md (and every listed file exists).
// It reports nothing when doc/rules/ holds no Markdown file; module_test.go
// pins that the real module has them.
func checkRuleTags(m Module) []Violation {
	rules, _ := fs.Glob(m.FS, "doc/rules/*.md")
	if len(rules) == 0 {
		return nil
	}
	var vs []Violation
	values := map[string]string{} // rule file -> tag value
	backticked := regexp.MustCompile("`([^`]+)`")
	for _, p := range rules {
		raw, _ := fs.ReadFile(m.FS, p)
		var lines []string
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, tagLabel) {
				lines = append(lines, strings.TrimRight(line, "\r"))
			}
		}
		if len(lines) != 1 {
			vs = append(vs, Violation{Rule: RuleRuleTag, File: p, Msg: "needs exactly one line starting " + tagLabel + "; found " + strconv.Itoa(len(lines))})
			continue
		}
		stem := strings.TrimSuffix(path.Base(p), ".md")
		form := regexp.MustCompile("^" + tagLabel + " `" + regexp.QuoteMeta(stem) + "-[0-9a-f]{12}`$")
		if !form.MatchString(lines[0]) {
			vs = append(vs, Violation{Rule: RuleRuleTag, File: p, Msg: "tag line must be " + tagLabel + " `" + stem + "-<12 lowercase hex digits>`"})
		}
		if sub := backticked.FindStringSubmatch(lines[0]); sub != nil {
			values[p] = sub[1]
		}
	}
	vs = append(vs, checkTagUniqueness(m, values)...)
	return append(vs, checkRuleList(m, rules)...)
}

// checkTagUniqueness reports a tag value found anywhere in the repository
// beyond its own tag line (other files, other lines, other rule files).
func checkTagUniqueness(m Module, values map[string]string) []Violation {
	var vs []Violation
	_ = fs.WalkDir(m.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		raw, err := fs.ReadFile(m.FS, p)
		if err != nil {
			return nil
		}
		text := string(raw)
		for rule, v := range values {
			allowed := 0
			if rule == p {
				allowed = 1
			}
			if strings.Count(text, v) > allowed {
				vs = append(vs, Violation{Rule: RuleRuleTag, File: rule, Msg: "tag value " + v + " also appears in " + p + "; a tag is unique to its file and one line"})
			}
		}
		return nil
	})
	return vs
}

// checkRuleList compares doc/rules/*.md with the backticked paths in the
// "Read the rules first" section of AGENTS.md.
func checkRuleList(m Module, rules []string) []Violation {
	raw, _ := fs.ReadFile(m.FS, "AGENTS.md")
	var listed []string
	in := false
	rulePath := regexp.MustCompile("`(doc/rules/[^`/]+\\.md)`")
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "## ") {
			in = strings.TrimSpace(line[3:]) == "Read the rules first"
			continue
		}
		if in {
			for _, sub := range rulePath.FindAllStringSubmatch(line, -1) {
				listed = append(listed, sub[1])
			}
		}
	}
	var vs []Violation
	for _, p := range rules {
		if !slices.Contains(listed, p) {
			vs = append(vs, Violation{Rule: RuleRuleTag, File: p, Msg: "rule file is not in the AGENTS.md \"Read the rules first\" list"})
		}
	}
	for _, p := range listed {
		if _, err := fs.Stat(m.FS, p); err != nil {
			vs = append(vs, Violation{Rule: RuleRuleTag, File: "AGENTS.md", Msg: "lists " + p + ", which does not exist"})
		}
	}
	return vs
}
