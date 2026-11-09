package archtest_test

import (
	"go/ast"
	"os"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

func loadModule(t *testing.T) archtest.Module {
	t.Helper()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return m
}

// exportedNames returns the exported top-level names the production files of pkg
// declare, optionally only those of the files named in only.
func exportedNames(pkg archtest.Package, only map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, f := range pkg.Production() {
		if only != nil && !only[f.Path] {
			continue
		}
		for _, d := range f.AST.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					out[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							out[s.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								out[n.Name] = true
							}
						}
					}
				}
			}
		}
	}
	return out
}

// Requirement: doc/specs/character-aggregate.md FR-098 - no exported name of the
// aggregate equals an export of another package of the module (so none is
// generic or collides with ability.Reason, ability.InvalidError or
// AbilityScoreRejection). The idiomatic New and MustNew are exempt, and so are
// the shared option, port and signal files, whose names are package-scoped.
// Path: unhappy (name collision).
func TestAggregateNamesAreUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	aggregate := map[string]bool{}
	for _, f := range []string{
		"identifier", "experience_amount", "experience_points", "heroic_inspiration", "transition", "feature_states", "event",
		"class_creation_choices", "bonus_feats", "rule_options", "character", "creation", "experience", "advancement_class",
		"level_up", "ability_score_change", "award_experience", "change_ability_score", "level_up_character",
	} {
		aggregate["character/"+f+".go"] = true
	}
	var own map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, aggregate)
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	if len(own) < 50 {
		t.Fatalf("only %d exported names found in the aggregate files", len(own))
	}
	for n := range own {
		if n == "New" || n == "MustNew" {
			continue
		}
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-099 - a use case file of the
// aggregate calls no other use case: no Run call except the port calls it needs,
// and no constructor of another use case.
// Path: unhappy (nested telemetry).
func TestAggregateUseCasesDoNotCallEachOther(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{"character/level_up_character.go": true, "character/award_experience.go": true, "character/change_ability_score.go": true}
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !files[f.Path] {
				continue
			}
			seen++
			ast.Inspect(f.AST, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fn := call.Fun.(type) {
				case *ast.Ident:
					if slices.Contains(constructors, fn.Name) {
						t.Errorf("%s calls %s", f.Path, fn.Name)
					}
				case *ast.SelectorExpr:
					if fn.Sel.Name == "Run" {
						t.Errorf("%s calls a Run method", f.Path)
					}
				}
				return true
			})
		}
	}
	if seen != len(files) {
		t.Errorf("found %d of %d use case files", seen, len(files))
	}
}
