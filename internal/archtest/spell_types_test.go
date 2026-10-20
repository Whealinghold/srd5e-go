package archtest_test

import (
	"go/ast"
	"go/token"
	"os"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// spellShapeNames are the exported names content-spells.md FR-114 and FR-119
// list for the spell shape: the closed sets with their variants, the units, the
// limit and the derived list. Each is declared exactly once in the module, as a
// type, a constant or a function.
func spellShapeNames() []string {
	return []string{
		"Spell", "SpellLevel", "SpellLevelError", "SpellSchool", "DistanceUnit", "DurationLimit",
		"CastingTime", "CastingTimeMagicAction", "CastingTimeBonusAction", "CastingTimeReaction", "CastingTimeMinutes", "CastingTimeHours",
		"CastingTimeMode", "SpellCastingTimes", "SpellCastingTimesSingle", "SpellCastingTimesModes",
		"SpellRange", "SpellRangeSelf", "SpellRangeTouch", "SpellRangeDistance", "SpellRangeSight", "SpellRangeUnlimited", "SpellRangeSpecial",
		"SpellDuration", "SpellDurationInstantaneous", "SpellDurationTimed", "SpellDurationUntilDispelled", "SpellDurationUntilDispelledOrTriggered", "SpellDurationSpecial",
		"SpellComponents", "MaterialComponent", "ClassSpellList", "DeriveClassSpellList",
	}
}

// Requirement: content-spells.md FR-114, FR-115 and plan item P7 - the exported
// names of the spell shape exist once each in the module (a type and a function
// cannot share the name ClassSpellList), and the module declares no
// SpellScaling: scaling stays text in higherLevel and cantripUpgrade.
// Path: unhappy (name collision).
func TestSpellShapeNamesAreUniqueAndThereIsNoSpellScaling(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	declared := map[string]int{}
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			for _, d := range f.AST.Decls {
				switch v := d.(type) {
				case *ast.FuncDecl:
					if v.Recv == nil {
						declared[v.Name.Name]++
					}
				case *ast.GenDecl:
					for _, s := range v.Specs {
						switch sp := s.(type) {
						case *ast.TypeSpec:
							declared[sp.Name.Name]++
						case *ast.ValueSpec:
							if v.Tok == token.CONST || v.Tok == token.VAR {
								for _, n := range sp.Names {
									declared[n.Name]++
								}
							}
						}
					}
				}
			}
		}
	}
	for _, name := range spellShapeNames() {
		if declared[name] != 1 {
			t.Errorf("%s is declared %d times in the module, want exactly once", name, declared[name])
		}
	}
	if declared["SpellScaling"] != 0 {
		t.Error("the module declares SpellScaling, which FR-115 forbids")
	}
}

// Requirement: content-spells.md FR-088 and FR-089 - the spell value types and
// their decoder files in package content import neither context, content/data
// nor content/contenttest and start no goroutine, and no production file of the
// module imports a package under tools/.
// Path: unhappy (forbidden import or go statement).
func TestSpellFilesArePureAndToolsIsNotImported(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	banned := []string{"context", m.Path + "/content/data", m.Path + "/content/contenttest"}
	tools := m.Path + "/tools"
	seen := 0
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			for _, imp := range f.Imports() {
				if imp.Path == tools || strings.HasPrefix(imp.Path, tools+"/") {
					t.Errorf("%s imports %s (FR-089)", f.Path, imp.Path)
				}
			}
			base := path.Base(f.Path)
			if pkg.Dir != "content" || (!strings.HasPrefix(base, "spell") && !strings.HasPrefix(base, "decode_spell")) {
				continue
			}
			seen++
			if f.AST.Name.Name != "content" {
				t.Errorf("%s is in package %s, want content (FR-089)", f.Path, f.AST.Name.Name)
			}
			for _, imp := range f.Imports() {
				if slices.Contains(banned, imp.Path) {
					t.Errorf("%s imports %s (FR-088)", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if _, ok := n.(*ast.GoStmt); ok {
					t.Errorf("%s starts a goroutine (FR-088)", f.Path)
				}
				return true
			})
		}
	}
	if seen == 0 {
		t.Error("no spell files found under content/")
	}
}
