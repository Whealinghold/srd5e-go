package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// weaponDomainFiles are the files of package character that hold the weapon and
// mastery rules; the use case file and signals.go are not among them.
var weaponDomainFiles = []string{"weapon_proficiency", "weapon_mastery_choice", "weapon_wielding", "weapon_attack_plan", "weapon_light_attack", "weapon_mastery_offer"}

var weaponUseCaseFiles = []string{"topple_save_usecase"}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-108, FR-109, FR-102 - the weapon domain files import
// no context, time, math/rand, sync, telemetry or combat package and start no goroutine; the Topple
// use case file imports no combat package and builds or calls no other use case, d20.Resolver
// included.
// Path: unhappy (architecture).
func TestWeaponRulesAreSilentAndTheUseCaseDoesNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender", "NewGainHeroicInspiration", "NewExpendHeroicInspiration", "NewResolver"}
	methods := []string{"Run", "TakeRest", "SpendHitPointDie", "Gain", "Expend"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(weaponDomainFiles, name), slices.Contains(weaponUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				banned := strings.HasSuffix(imp.Path, "/combat") || strings.Contains(imp.Path, "/combat/")
				if domain {
					banned = banned || slices.Contains([]string{"context", "time", "sync", "math/rand", "math/rand/v2"}, imp.Path) || strings.Contains(imp.Path, "telemetry")
				}
				if banned {
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
						if useCase && slices.Contains(constructors, fn.Name) {
							t.Errorf("%s calls %s", f.Path, fn.Name)
						}
					case *ast.SelectorExpr:
						if useCase && (slices.Contains(constructors, fn.Sel.Name) || slices.Contains(methods, fn.Sel.Name)) {
							t.Errorf("%s calls %s", f.Path, fn.Sel.Name)
						}
					}
				}
				return true
			})
		}
	}
	if seen != len(weaponDomainFiles)+len(weaponUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(weaponDomainFiles)+len(weaponUseCaseFiles))
	}
}

// Requirement: doc/specs/weapon-properties-and-mastery.md FR-110, FR-112 - every exported identifier of the weapon
// files is documented, none duplicates an export of another package of the module, none is a
// generic variant name and none names a consumer.
// Path: unhappy (name collision).
func TestWeaponNamesAreUniqueDocumentedAndGeneral(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(weaponDomainFiles, weaponUseCaseFiles) {
		files["character/"+f+".go"] = true
	}
	var own map[string]bool
	others := map[string][]string{}
	undocumented := []string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, files)
			for _, f := range pkg.Production() {
				if !files[f.Path] {
					continue
				}
				for _, d := range f.AST.Decls {
					switch d := d.(type) {
					case *ast.FuncDecl:
						if d.Name.IsExported() && d.Doc == nil {
							undocumented = append(undocumented, d.Name.Name)
						}
					case *ast.GenDecl:
						for _, s := range d.Specs {
							if ts, ok := s.(*ast.TypeSpec); ok && ts.Name.IsExported() && d.Doc == nil && ts.Doc == nil {
								undocumented = append(undocumented, ts.Name.Name)
							}
						}
					}
				}
			}
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	if len(own) < 80 {
		t.Fatalf("only %d exported names found in the weapon files", len(own))
	}
	for _, n := range undocumented {
		t.Errorf("character.%s has no doc comment", n)
	}
	generic := []string{"Applied", "Rejected", "NotFound", "Accepted", "Blocked", "Allowed"}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
		if slices.Contains(generic, n) {
			t.Errorf("character.%s is a generic variant name", n)
		}
	}
}
