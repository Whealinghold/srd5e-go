package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// scrollUseFiles are the files of package character that hold the Spell Scroll use rules: pure
// domain code, no use case and no signal.
var scrollUseFiles = []string{"spell_scroll", "spell_scroll_table", "spell_scroll_reader", "spell_scroll_assessment", "spell_scroll_check", "spell_scroll_scribing", "spell_scroll_copying"}

// scrollUseRequiredNames are the types and functions the spec requires package character to export (FR-035).
var scrollUseRequiredNames = []string{"SpellScroll", "NewSpellScroll", "SpellScrollOrigin", "FoundScroll", "ScribedScrollOrigin", "ScribedScrollStats", "NewScribedScrollStats",
	"SpellScrollRow", "SpellScrollTableRow", "ScrollReaderProfile", "NewScrollReaderProfile", "MustScrollReaderProfile", "ScrollClassRoute", "AnyScrollRoute", "ScrollCheckOption",
	"ScrollCastTerms", "ScrollCasterLevel", "ScrollLowestCasterLevel", "ScribedCantripCasterLevel", "ScribeFacts", "ScribeProblem", "ScrollDisposition", "DispositionRetained",
	"DispositionConsumed", "DispositionErased", "ScrollCastProgress", "ProgressNotStarted", "ProgressInterrupted", "ProgressCompleted", "ScrollNotCopyableReason", "NotAWizardSpell",
	"Cantrip", "ScrollAssessmentResult", "ScrollReady", "ScrollCheckRequired", "ScrollNotReadable", "ScrollNotOnSpellList", "ScrollAlreadyUsed", "ScrollCheckResolutionResult",
	"ScrollCheckPassed", "ScrollSpellErased", "ScribeAssessmentResult", "ScribeEligible", "ScribeIneligible", "ScrollCopyAssessmentResult", "ScrollCopyCheckRequired",
	"ScrollNotCopyable", "ScrollCopyResolved", "AssessScrollReading", "ResolveScrollCheck", "SettleScrollReading", "AssessScribing", "AssessScrollCopying", "ResolveScrollCopy",
	"ErrScrollCheckMismatch"}

// Requirement: doc/specs/spell-scroll-use.md FR-035, section 8 "architecture" - package character exports every type the spec names,
// none of the Spell Scroll files declares an event type or a generic variant name, and none of their exports equals an export of
// another package of the module.
// Path: unhappy (name collision).
func TestSpellScrollUseNamesAreExportedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range scrollUseFiles {
		files["character/"+f+".go"] = true
	}
	var own map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, files)
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	for _, n := range scrollUseRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s from the Spell Scroll files", n)
		}
	}
	for n := range own {
		if strings.HasSuffix(n, "Event") || slices.Contains([]string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"}, n) {
			t.Errorf("character.%s is an event type or a generic variant name", n)
		}
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}

// Requirement: doc/specs/spell-scroll-use.md FR-034, section 7 and architecture.md section 3 - the Spell Scroll files are pure: they import
// neither telemetry nor context nor sync nor time nor combat, start no goroutine and read no clock.
// Path: unhappy (architecture).
func TestSpellScrollUseDomainIsPure(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !slices.Contains(scrollUseFiles, strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")) {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry") || strings.HasSuffix(imp.Path, "/combat") || strings.HasSuffix(imp.Path, "/dice") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
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
	if seen != len(scrollUseFiles) {
		t.Errorf("found %d of %d files", seen, len(scrollUseFiles))
	}
}
