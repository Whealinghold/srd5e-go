package archtest_test

import (
	"go/ast"
	"os"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// TestMagicItemMarkerHasOneImplementer pins content-magic-items FR-003 on the
// real tree: the marker interface MagicItem has exactly one method,
// IsMagicItem, MagicItemEntry is the only type of the module with that method,
// and no member anywhere is named isMagicItem (callers test the type
// assertion, not a flag).
func TestMagicItemMarkerHasOneImplementer(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var implementers []string
	sawInterface := false
	for _, pkg := range m.Packages {
		for _, f := range pkg.Files {
			if f.Test {
				continue
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch d := n.(type) {
				case *ast.FuncDecl:
					if d.Name.Name == "isMagicItem" {
						t.Errorf("%s declares isMagicItem; use the type assertion", f.Path)
					}
					if d.Recv != nil && d.Name.Name == "IsMagicItem" {
						implementers = append(implementers, f.Path)
					}
				case *ast.TypeSpec:
					if it, ok := d.Type.(*ast.InterfaceType); ok && d.Name.Name == "MagicItem" {
						sawInterface = true
						if len(it.Methods.List) != 1 || it.Methods.List[0].Names[0].Name != "IsMagicItem" {
							t.Errorf("MagicItem in %s has more than the one marker method", f.Path)
						}
					}
				case *ast.Field:
					for _, name := range d.Names {
						if name.Name == "isMagicItem" {
							t.Errorf("%s has a member named isMagicItem", f.Path)
						}
					}
				}
				return true
			})
		}
	}
	if !sawInterface {
		t.Error("the MagicItem interface is not declared")
	}
	if len(implementers) != 1 || implementers[0] != "content/magic_item.go" {
		t.Errorf("types with IsMagicItem declared in %v; want exactly MagicItemEntry in content/magic_item.go", implementers)
	}
}
