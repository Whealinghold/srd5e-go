package archtest

import (
	"go/token"
	"slices"
	"strings"
)

func isTestSupport(p Policy, rel string) bool { return p.Entries[rel].Kind == KindSupport }

func isContext(p Policy, rel string) bool { return p.Entries[rel].Kind == KindContext }

// checkImports covers A1 (allow-list), A2 (test-support isolation), A3
// (stdlib deny-list and third-party imports) for one package.
func checkImports(m Module, p Policy, pkg Package, entry Entry, listed bool) []Violation {
	var vs []Violation
	for _, f := range pkg.Files {
		for _, imp := range f.Imports() {
			path, pos := imp.Path, imp.Pos
			rel, local := m.Local(path)
			switch {
			case local:
				vs = append(vs, checkLocalImport(p, f, pos, pkg, entry, listed, rel)...)
			case isThirdParty(path):
				vs = append(vs, at(f, pos, RuleThirdParty, "import %q is third-party; only the standard library is allowed", path))
			case !f.Test && listed && (entry.Kind == KindKernel || entry.Kind == KindContext):
				if deniedStdlib(path) && !stdlibExceptions(pkg.Dir, path) {
					vs = append(vs, at(f, pos, RuleStdlib, "%s must not import %q", pkg.Dir, path))
				}
			}
		}
	}
	return vs
}

func checkLocalImport(p Policy, f File, pos token.Pos, pkg Package, entry Entry, listed bool, rel string) []Violation {
	var vs []Violation
	if (rel == "internal" || strings.HasPrefix(rel, "internal/")) && rel != pkg.Dir {
		return []Violation{at(f, pos, RuleInternal, "%s imports %s; internal packages are never imported", f.Path, rel)}
	}
	if f.Test {
		return nil
	}
	if isTestSupport(p, rel) {
		vs = append(vs, at(f, pos, RuleIsolation, "non-test file imports test-support package %s; only _test.go files may", rel))
	}
	if listed && entry.Kind == KindKernel && isContext(p, rel) {
		vs = append(vs, at(f, pos, RuleKernelUp, "kernel package %s imports context %s", pkg.Dir, rel))
	}
	if listed && rel != pkg.Dir && !slices.Contains(entry.Allow, rel) {
		vs = append(vs, at(f, pos, RuleImport, "%s may not import %s (allowed: %v)", pkg.Dir, rel, entry.Allow))
	}
	return vs
}

// isThirdParty reports whether a non-module import path looks like a
// third-party module: its first element contains a dot.
func isThirdParty(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return strings.Contains(first, ".")
}

// checkCycles reports a cycle in the module-local import graph of non-test
// files. The compiler already rejects cycles; this keeps the fixtures honest
// and the failure message architectural.
func checkCycles(m Module) []Violation {
	graph := map[string][]string{}
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			for _, imp := range f.Imports() {
				if rel, ok := m.Local(imp.Path); ok && rel != pkg.Dir {
					graph[pkg.Dir] = append(graph[pkg.Dir], rel)
				}
			}
		}
	}
	const (
		visiting = 1
		done     = 2
	)
	state := map[string]int{}
	var vs []Violation
	var visit func(string, []string)
	visit = func(n string, stack []string) {
		state[n] = visiting
		for _, next := range graph[n] {
			switch state[next] {
			case visiting:
				vs = append(vs, Violation{Rule: RuleCycle, Msg: "import cycle: " + strings.Join(append(slices.Clone(stack), n, next), " -> ")})
			case 0:
				visit(next, append(slices.Clone(stack), n))
			}
		}
		state[n] = done
	}
	for _, pkg := range m.Packages {
		if state[pkg.Dir] == 0 {
			visit(pkg.Dir, nil)
		}
	}
	return vs
}
