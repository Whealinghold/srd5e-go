package archtest_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// spellEffectFileBase reports whether base names a file of the spell effect
// model (spell-effect-data.md FR-039: files named spell_effects*.go).
func spellEffectFileBase(base string) bool {
	return strings.HasPrefix(base, "spell_effects") && strings.HasSuffix(base, ".go") && !strings.HasSuffix(base, "_test.go")
}

// spellEffectAllowedImports are what the model files may import (FR-039): the
// kernel packages the spec names and the standard-library packages of pure
// code. Nothing from combat or character, no context, clock, random or I/O.
func spellEffectAllowedImports() []string {
	return []string{
		"math", "slices",
		"github.com/Whealinghold/srd5e-go/dice",
		"github.com/Whealinghold/srd5e-go/ability",
		"github.com/Whealinghold/srd5e-go/srderr",
	}
}

// spellNames are spells the specs name; no model type may be named after one
// (FR-002).
func spellNames() []string {
	return []string{
		"Fireball", "FireBolt", "MagicMissile", "HoldPerson", "CureWounds", "EldritchBlast", "ChromaticOrb",
		"AcidSplash", "FalseLife", "Heal", "Harm", "HealingWord", "RayOfSickness", "BurningHands", "FlameStrike",
		"IceKnife", "BestowCurse", "Hex", "HuntersMark", "Bane", "Eyebite", "Command", "TrueStrike", "SacredFlame",
		"ScorchingRay", "ChainLightning", "FaerieFire", "FingerOfDeath", "SpiritGuardians", "Cloudkill", "Moonbeam",
		"SpikeGrowth", "WallOfThorns", "GuardianOfFaith", "FlamingSphere",
	}
}

// spellEffectViolations applies FR-002 and FR-039 to the model files: no
// function type, any or interface{} in a struct field or in a parameter or
// result of an exported function or method; no type named after a spell or
// with a Behaviour or Strategy suffix; no import outside the allow-list; no go
// statement; no use of a Spell type.
func spellEffectViolations(files []archtest.File) []string {
	var out []string
	for _, f := range files {
		base := path.Base(f.Path)
		for _, imp := range f.Imports() {
			if !slices.Contains(spellEffectAllowedImports(), imp.Path) {
				out = append(out, base+": imports "+imp.Path)
			}
		}
		badType := func(where string, e ast.Expr) {
			ast.Inspect(e, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.FuncType:
					out = append(out, base+": "+where+" has a function type")
				case *ast.InterfaceType:
					if len(v.Methods.List) == 0 {
						out = append(out, base+": "+where+" has interface{}")
					}
				case *ast.Ident:
					if v.Name == "any" {
						out = append(out, base+": "+where+" has any")
					}
				}
				return true
			})
		}
		for _, d := range f.AST.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				for _, fl := range []*ast.FieldList{d.Type.Params, d.Type.Results} {
					if fl == nil {
						continue
					}
					for _, p := range fl.List {
						badType("func "+d.Name.Name, p.Type)
					}
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					ts, ok := s.(*ast.TypeSpec)
					if !ok {
						continue
					}
					for _, n := range spellNames() {
						if strings.Contains(ts.Name.Name, n) {
							out = append(out, base+": type "+ts.Name.Name+" is named after a spell")
						}
					}
					if strings.HasSuffix(ts.Name.Name, "Behaviour") || strings.HasSuffix(ts.Name.Name, "Strategy") {
						out = append(out, base+": type "+ts.Name.Name+" is a behaviour or strategy")
					}
					if st, ok := ts.Type.(*ast.StructType); ok {
						for _, fld := range st.Fields.List {
							badType("a field of "+ts.Name.Name, fld.Type)
						}
					}
				}
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.GoStmt:
				out = append(out, base+": go statement")
			case *ast.Ident:
				if v.Name == "Spell" {
					out = append(out, base+": uses Spell")
				}
			}
			return true
		})
	}
	return out
}

func parseFile(t *testing.T, name, src string) archtest.File {
	t.Helper()
	fset := token.NewFileSet()
	af, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return archtest.File{Path: "content/" + name, AST: af, Fset: fset}
}

// Requirement: spell-effect-data.md FR-002, FR-039 - the rule itself: a
// violating overlay is reported, a clean file is not.
// Path: unhappy (invalid input).
func TestSpellEffectRuleFindsViolations(t *testing.T) {
	t.Parallel()
	bad := parseFile(t, "spell_effects_bad.go", `package content
import ("time"; "github.com/Whealinghold/srd5e-go/combat")
type Fireball struct{ run func() }
type HoldBehaviour struct{ v any; w interface{} }
type Spell struct{}
func NewThing(f func(int), a any) (func(), error) { go f(1); return nil, nil }
`)
	clean := parseFile(t, "spell_effects_ok.go", `package content
import "slices"
type Marker interface{ isMarker() }
type Thing struct{ v []int }
func NewThing(v []int) (Thing, error) { return Thing{slices.Clone(v)}, nil }
func helper(f func()) {}
`)
	got := spellEffectViolations([]archtest.File{bad})
	for _, want := range []string{
		"imports time", "imports github.com/Whealinghold/srd5e-go/combat", "type Fireball is named after a spell",
		"a field of Fireball has a function type", "type HoldBehaviour is a behaviour or strategy", "a field of HoldBehaviour has any",
		"a field of HoldBehaviour has interface{}", "func NewThing has a function type", "func NewThing has any", "go statement", "uses Spell",
	} {
		if !slices.ContainsFunc(got, func(s string) bool { return strings.HasSuffix(s, want) }) {
			t.Errorf("no violation ending %q in %v", want, got)
		}
	}
	if got := spellEffectViolations([]archtest.File{clean}); len(got) != 0 {
		t.Errorf("clean file reported: %v", got)
	}
}

// Requirement: spell-effect-data.md FR-002, FR-039 - the model files of the
// real tree are plain data in package content: no function type, any or
// interface{}, no spell-named type, only allowed imports; every exported type
// they declare is declared once in the module and none has a generic variant
// name; there is at least one such file.
// Path: happy.
func TestSpellEffectModelFilesOfTheModule(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var files []archtest.File
	declared := map[string]int{}
	var mine []string
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			isModel := pkg.Dir == "content" && spellEffectFileBase(path.Base(f.Path))
			if isModel {
				files = append(files, f)
			}
			for _, d := range f.AST.Decls {
				if g, ok := d.(*ast.GenDecl); ok {
					for _, s := range g.Specs {
						if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.IsExported() {
							declared[ts.Name.Name]++
							if isModel {
								mine = append(mine, ts.Name.Name)
							}
						}
					}
				}
			}
		}
	}
	if len(files) == 0 {
		t.Fatal("no content/spell_effects*.go file found")
	}
	if got := spellEffectViolations(files); len(got) != 0 {
		t.Errorf("model files break FR-002 or FR-039: %v", got)
	}
	generic := []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"}
	for _, name := range mine {
		if declared[name] != 1 || slices.Contains(generic, name) {
			t.Errorf("exported type %s is declared %d times or is a generic variant name", name, declared[name])
		}
	}
	for _, want := range []string{"AreaShape", "ConeArea", "CubeArea", "CylinderArea", "EmanationArea", "LineArea", "SphereArea", "TurnOwner"} {
		if declared[want] != 1 {
			t.Errorf("%s is declared %d times in the module, want exactly once", want, declared[want])
		}
	}
	if declared["TurnHolder"] != 0 {
		t.Error("a TurnHolder type shares the module with TurnOwner")
	}
}
