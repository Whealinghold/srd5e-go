package archtest_test

import (
	"go/ast"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// equipmentFiles returns the production files of package character that hold
// the equipment rule maths.
func equipmentFiles(t *testing.T) []archtest.File {
	t.Helper()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var out []archtest.File
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if strings.HasPrefix(f.Path, "character/equipment_") {
				out = append(out, f)
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no character/equipment_*.go files found")
	}
	return out
}

// Requirement: equipment-rules FR-058, FR-059, FR-060, FR-070 (architecture.md
// sections 3, 9; observability.md section 3).
// Path: happy.
func TestEquipmentFilesArePureAndLeaf(t *testing.T) {
	t.Parallel()
	allowed := []string{"ability", "modifier", "srderr", "content"}
	banned := []string{"os", "net", "context", "time", "math/rand", "math/rand/v2", "sync", "reflect"}
	for _, f := range equipmentFiles(t) {
		for _, imp := range f.Imports() {
			switch {
			case slices.Contains(banned, imp.Path), strings.HasSuffix(imp.Path, "test"), strings.Contains(imp.Path, "telemetry"):
				t.Errorf("%s imports %s", f.Path, imp.Path)
			case strings.HasPrefix(imp.Path, "github.com/Whealinghold/srd5e-go/"):
				if rel := strings.TrimPrefix(imp.Path, "github.com/Whealinghold/srd5e-go/"); !slices.Contains(allowed, rel) {
					t.Errorf("%s imports %s; only %v are allowed", f.Path, imp.Path, allowed)
				}
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if g, ok := n.(*ast.GoStmt); ok {
				t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(g.Pos()))
			}
			return true
		})
	}
}

// Requirement: equipment-rules FR-005, FR-019 - ArmorRules offers exactly
// BaseArmorClass, SpeedPenaltyFeet and StealthCircumstances, and LoadSpeedLimited
// has no member taking a Speed.
// Path: unhappy (rule interaction).
func TestEquipmentExportedMembers(t *testing.T) {
	t.Parallel()
	members := map[string][]string{}
	for _, f := range equipmentFiles(t) {
		for _, d := range f.AST.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || !fd.Name.IsExported() {
				continue
			}
			recv := fd.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if id, ok := recv.(*ast.Ident); ok {
				members[id.Name] = append(members[id.Name], fd.Name.Name)
			}
		}
	}
	slices.Sort(members["ArmorRules"])
	if want := []string{"BaseArmorClass", "SpeedPenaltyFeet", "StealthCircumstances"}; !slices.Equal(members["ArmorRules"], want) {
		t.Errorf("ArmorRules members = %v, want %v", members["ArmorRules"], want)
	}
	if want := []string{"LimitFeet"}; !slices.Equal(members["LoadSpeedLimited"], want) {
		t.Errorf("LoadSpeedLimited members = %v, want %v", members["LoadSpeedLimited"], want)
	}
}
