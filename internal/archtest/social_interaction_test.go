package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// influenceDomainFiles are the files of package character that hold the social
// interaction rules; the use case file and signals.go are not among them.
var influenceDomainFiles = []string{"attitude", "influence_approach", "influence_target", "dc_basis", "influence_rules"}

var influenceUseCaseFiles = []string{"resolve_influence_usecase"}

// influenceD20Names are the identifiers of package d20 the Influence use case file may
// name: the spec's FR-038 list, plus the constants and constructor of the same functions
// (AbilityCheck, Automatic, Success, NewRollModifiers) and the shared attribute key.
var influenceD20Names = []string{"NewRoll", "Roll", "RollModifiers", "NewRollModifiers", "Kind", "AbilityCheck", "Outcome", "Success",
	"ResolveCheck", "FailWithoutRoll", "ForcedFailureReason", "Automatic", "CheckResult", "ForcedFailureResult", "DifficultyClass", "AttrRollMode"}

// Requirement: doc/specs/social-interaction.md FR-038, FR-044, FR-045 - the social
// interaction domain files import only ability, modifier, content, d20 and srderr (and
// the standard library apart from context, time, sync and math/rand), start no goroutine
// and import no telemetry; the use case file imports no combat package, names only the
// listed d20 identifiers (never d20.Resolver), constructs or calls no other use case and
// makes no log call.
// Path: unhappy (forbidden edge).
func TestInfluenceFilesRespectTheirImportsAndDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	moduleImports := []string{"/ability", "/modifier", "/content", "/d20"}
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender",
		"NewGainHeroicInspiration", "NewExpendHeroicInspiration", "NewToppleSaveResolver", "NewResolver", "NewInfluenceResolver"}
	methods := []string{"Run", "TakeRest", "SpendHitPointDie", "Gain", "Expend", "ResolveToppleSave", "ResolveInfluence", "Log"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(influenceDomainFiles, name), slices.Contains(influenceUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				banned := strings.HasSuffix(imp.Path, "/combat") || strings.Contains(imp.Path, "/combat/") || strings.HasSuffix(imp.Path, "test")
				if domain {
					banned = banned || slices.Contains([]string{"context", "time", "sync", "math/rand", "math/rand/v2"}, imp.Path) || strings.Contains(imp.Path, "telemetry") || strings.HasSuffix(imp.Path, "/dice")
				}
				if banned {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
				if domain && strings.Contains(imp.Path, "srd5e-go/") && !slices.ContainsFunc(moduleImports, func(s string) bool { return strings.HasSuffix(imp.Path, s) }) && !strings.HasSuffix(imp.Path, "/srderr") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.CallExpr:
					switch fn := n.Fun.(type) {
					case *ast.Ident:
						if useCase && slices.Contains(constructors, fn.Name) && fn.Name != "NewInfluenceResolver" {
							t.Errorf("%s calls %s", f.Path, fn.Name)
						}
					case *ast.SelectorExpr:
						if useCase && (slices.Contains(constructors, fn.Sel.Name) || slices.Contains(methods, fn.Sel.Name)) {
							t.Errorf("%s calls %s", f.Path, fn.Sel.Name)
						}
					}
				case *ast.SelectorExpr:
					if id, ok := n.X.(*ast.Ident); ok && id.Name == "d20" && useCase && !slices.Contains(influenceD20Names, n.Sel.Name) {
						t.Errorf("%s names d20.%s", f.Path, n.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if seen != len(influenceDomainFiles)+len(influenceUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(influenceDomainFiles)+len(influenceUseCaseFiles))
	}
}

// Requirement: doc/specs/social-interaction.md FR-039 - the social interaction files
// export names no other package of the module also exports (so none re-exports or
// aliases an upstream type), and declare no type alias.
// Path: unhappy (name collision).
func TestInfluenceExportsAreUniqueAndNotAliases(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(influenceDomainFiles, influenceUseCaseFiles) {
		files["character/"+f+".go"] = true
	}
	var own map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, files)
			for _, f := range pkg.Production() {
				if !files[f.Path] {
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
		t.Fatalf("only %d exported names found in the social interaction files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
