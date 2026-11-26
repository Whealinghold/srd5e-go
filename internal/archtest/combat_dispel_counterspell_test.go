package archtest_test

import (
	"go/ast"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// dispelCounterspellFiles returns the production files of the Dispel Magic, Counterspell
// and Antimagic Field model in combat (dispel-and-counterspell section 5): the dispel,
// counterspell, antimagic, ongoing spell, spell ending event, spell part and spell
// time remaining files.
func dispelCounterspellFiles(t *testing.T) []archtestFile {
	t.Helper()
	prefixes := []string{"combat/dispel_", "combat/counterspell", "combat/antimagic_field", "combat/ongoing_spell",
		"combat/spell_ending_event", "combat/spell_part", "combat/spell_time_remaining"}
	var out []archtestFile
	for _, f := range combatPackage(t).Production() {
		if slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(f.Path, p) }) {
			out = append(out, archtestFile{path: f.Path, ast: f.AST})
		}
	}
	if len(out) < 12 {
		t.Fatalf("found %d Dispel Magic and Counterspell files, want the model's files", len(out))
	}
	return out
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-053 - the domain files import only kernel packages and the published language of the specs in "Depends on" (ids, value objects, events, d20 results); they import no telemetry, and they never name the character concentration rules, which are a service and not published language.
// Path: unhappy (forbidden edge).
func TestDispelCounterspellFilesImportOnlyThePublishedLanguage(t *testing.T) {
	t.Parallel()
	allowed := []string{
		"github.com/Whealinghold/srd5e-go/ability",
		"github.com/Whealinghold/srd5e-go/character",
		"github.com/Whealinghold/srd5e-go/content",
		"github.com/Whealinghold/srd5e-go/d20",
		"github.com/Whealinghold/srd5e-go/modifier",
		"github.com/Whealinghold/srd5e-go/srderr",
	}
	for _, f := range dispelCounterspellFiles(t) {
		for _, imp := range f.ast.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: import %s: %v", f.path, imp.Path.Value, err)
			}
			if strings.HasPrefix(p, "github.com/Whealinghold/srd5e-go/") && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s; the Dispel Magic model may import only the published language and kernel packages [FR-053]", f.path, p)
			}
		}
		ast.Inspect(f.ast, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if ok && sel.Sel.Name == "ConcentrationRules" {
				t.Errorf("%s names character.ConcentrationRules; a combat model reads character's published values only [FR-053]", f.path)
			}
			return true
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-055 - no generic variant name (Applied, Invalid or Rejected) is exported by the Dispel Magic and Counterspell files; each result is named for its operation (architecture.md section 4; check A7 covers the wider generic list).
// Path: unhappy (name collision).
func TestDispelCounterspellVariantNamesAreNotGeneric(t *testing.T) {
	t.Parallel()
	generic := []string{"Applied", "Invalid", "Rejected"}
	for _, f := range dispelCounterspellFiles(t) {
		for _, decl := range f.ast.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if ok && ts.Name.IsExported() && slices.Contains(generic, ts.Name.Name) {
					t.Errorf("%s exports %s; a result variant is named for its operation [FR-055]", f.path, ts.Name.Name)
				}
			}
		}
	}
}
