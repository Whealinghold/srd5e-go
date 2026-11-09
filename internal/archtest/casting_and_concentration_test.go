package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// castingDomainFiles are the files of package character that hold the casting and
// Concentration rules; none imports telemetry, context, sync or time. castingUseCaseFile
// holds every telemetry call.
var castingDomainFiles = []string{"casting_values", "concentration_save_dc", "casting_event", "concentration_rules", "spell_casting"}

const castingUseCaseFile = "casting_usecase"

// castingRequiredNames are the names FR-142 requires package character to export.
var castingRequiredNames = []string{"CastSpell", "CompleteCasting", "BreakConcentration", "BeginConcentration", "CastSpellCommand", "CompleteCastingCommand", "BreakConcentrationCommand",
	"BeginConcentrationCommand", "CastSpellResult", "CompleteCastingResult", "BreakConcentrationResult", "BeginConcentrationResult", "CastSpellCast", "CastSpellStarted", "CastSpellRejected",
	"CompleteCastingCompleted", "CompleteCastingRejected", "BreakConcentrationEnded", "BreakConcentrationCastingFailed", "BreakConcentrationUnchanged", "BeginConcentrationBegan",
	"BeginConcentrationCastingFailed", "CastMode", "CastingContext", "CastingFacts", "ComponentKind", "CastReport", "CastRejection", "CastRejectionKind", "ConcentrationRules",
	"ConcentrationBreakTransition", "CounterCastTransition", "SpellCastTerms", "ConcentrationSaveDC", "ConcentrationSaveDCForDamage", "ConcentrationBreakReason",
	"ConcentrationEndReason", "CastingEvent", "SpellCast", "CastingCountered", "ConcentrationBegan", "ConcentrationEnded", "CastingFailed"}

// castingUnexportedNames are the exported spellings of the internal types FR-128 and Section 5 keep unexported (spellCasting, castOutcome).
var castingUnexportedNames = []string{"SpellCasting", "CastOutcome", "ConcentrationTransitions"}

// Requirement: doc/specs/casting-and-concentration.md FR-128, FR-137 - the casting domain files import neither
// telemetry nor context nor sync nor time and start no goroutine, and the use case file calls no other use case.
// Path: unhappy (architecture).
func TestCastingDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewCastSpell", "NewCompleteCasting", "NewBreakConcentration", "NewBeginConcentration", "NewExpendSpellSlot", "NewCreateSpellcasting", "NewCalculateSpellcasting"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(castingDomainFiles, name), name == castingUseCaseFile
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry")) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Do" {
						t.Errorf("%s calls a Do method", f.Path)
					}
				}
				return true
			})
		}
	}
	if want := len(castingDomainFiles) + 1; seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-128, FR-142 - package character exports every type a casting
// command or result names, keeps the internal service and outcome types unexported, and none of the casting files exports a
// name another package of the module also exports.
// Path: unhappy (name collision).
func TestCastingExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range append(slices.Clone(castingDomainFiles), castingUseCaseFile, "casting_signals") {
		files["character/"+f+".go"] = true
	}
	var own, all map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own, all = exportedNames(pkg, files), exportedNames(pkg, nil)
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	for _, n := range castingRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s (FR-142)", n)
		}
	}
	for _, n := range castingUnexportedNames {
		if all[n] {
			t.Errorf("character exports %s (FR-128)", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
