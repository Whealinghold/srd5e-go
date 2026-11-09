package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// toolCraftingFiles are the files of package character that hold the tool
// proficiency, crafting prerequisite and scribing rules. All are pure domain
// code: none holds a use case or a signal.
var toolCraftingFiles = []string{"tool_proficiencies", "tool_resolution", "tool_check", "tool_utilize", "crafting_party", "crafting_plan",
	"scribing_prepared_spells", "scribing_qualification", "scribing_job"}

func toolCraftingPaths() map[string]bool {
	out := map[string]bool{}
	for _, f := range toolCraftingFiles {
		out["character/"+f+".go"] = true
	}
	return out
}

// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-044, FR-045, FR-046,
// FR-047, FR-049 - the tool and crafting files import only ability, modifier,
// srderr, d20 and content of this module and nothing of context, time, sync,
// os, io, net or math/rand; import no telemetry, dice or *test package; start no
// goroutine; call no use case or logger; and use no any or interface{}.
// Path: unhappy (forbidden edge).
func TestToolCraftingFilesArePureDomainCode(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	allowed := []string{"/ability", "/modifier", "/srderr", "/d20", "/content"}
	banned := []string{"context", "time", "sync", "sync/atomic", "os", "io", "net", "math/rand", "math/rand/v2"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !toolCraftingPaths()[f.Path] {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if slices.Contains(banned, imp.Path) || strings.Contains(imp.Path, "telemetry") || strings.HasSuffix(imp.Path, "test") || strings.Contains(imp.Path, "/combat") || strings.Contains(imp.Path, "/toolbox") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
				if strings.Contains(imp.Path, "srd5e-go/") && !slices.ContainsFunc(allowed, func(s string) bool { return strings.HasSuffix(imp.Path, s) }) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.SelectorExpr:
					if slices.Contains([]string{"Do", "Run", "Log", "Start", "WrapPort"}, n.Sel.Name) {
						t.Errorf("%s names .%s at %v", f.Path, n.Sel.Name, f.Fset.Position(n.Pos()))
					}
				case *ast.InterfaceType:
					if n.Methods == nil || len(n.Methods.List) == 0 {
						t.Errorf("%s: empty interface at %v", f.Path, f.Fset.Position(n.Pos()))
					}
				case *ast.Ident:
					if n.Name == "any" {
						t.Errorf("%s: any at %v", f.Path, f.Fset.Position(n.Pos()))
					}
				}
				return true
			})
		}
	}
	if seen != len(toolCraftingFiles) {
		t.Errorf("found %d of %d files", seen, len(toolCraftingFiles))
	}
}

// Requirement: doc/specs/tool-proficiency-and-crafting.md FR-047 - the files
// export names no other package of the module also exports, no generic variant
// name, and declare no type alias (so none re-exports an upstream type).
// Path: unhappy (name collision).
func TestToolCraftingExportsAreUniqueAndNotAliases(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	var own map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, toolCraftingPaths())
			for _, f := range pkg.Production() {
				if !toolCraftingPaths()[f.Path] {
					continue
				}
				ast.Inspect(f.AST, func(n ast.Node) bool {
					if s, ok := n.(*ast.TypeSpec); ok && s.Assign.IsValid() {
						t.Errorf("%s: type alias %s", f.Path, s.Name.Name)
					}
					return true
				})
			}
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	if len(own) < 40 {
		t.Fatalf("only %d exported names found in the tool and crafting files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
		if slices.Contains([]string{"Applied", "Rejected", "Found", "NotFound", "Succeeded", "ContentMissing"}, n) {
			t.Errorf("character.%s is a generic variant name", n)
		}
	}
}
