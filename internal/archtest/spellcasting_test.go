package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// spellcastingDomainFiles are the files of package character that hold the spellcasting
// rules; none imports telemetry or context. The use case files hold every telemetry call.
var spellcastingDomainFiles = []string{"spellcasting_class", "spellcasting_stats", "spell_slot_tables", "spell_slots", "spell_lists", "class_spellcasting",
	"spell_preparation", "spellcasting_state", "spellcasting_event", "spellcasting_shapes"}

var spellcastingUseCaseFiles = []string{"calculate_spellcasting_usecase", "create_spellcasting_usecase", "spell_preparation_usecase", "spell_slot_usecase"}

// spellcastingRequiredNames are the names FR-136 requires package character to export.
var spellcastingRequiredNames = []string{"Spellbook", "CantripsKnown", "ClassSpellcasting", "SpellcastingFeature", "SpellcastingFocusKind", "CastSource", "SpellSlotChoice",
	"Concentration", "CastingInProgress", "SpellcastingSummary", "SpellSlotPools", "SlotPoolView", "FeatureSpellAbility", "SlotExpenditure", "SlotRecovery", "SpellcastingLevelChange",
	"PreparationOutcome", "SpellcastingEvent", "SpellSlotExpended", "SpellSlotsRestored", "PreparedSpellsChanged", "SpellcastingState", "ConcentrationSubject",
	"CalculateSpellcastingResult", "CreateSpellcastingResult", "CreateSpellcastingContentMissing", "ChooseAdditionalSpellsResult", "ChooseAdditionalSpellsContentMissing",
	"ChangePreparedSpellsResult", "ChangePreparedSpellsContentMissing", "ExpendSpellSlotResult", "RegainSpellSlotResult", "RestoreSpellSlotsResult", "AdvanceSpellcastingResult",
	"CreateSpellcastingCommand", "ChooseAdditionalSpellsCommand", "ChangePreparedSpellsCommand", "ExpendSpellSlotCommand", "RegainSpellSlotCommand", "RestoreSpellSlotsCommand",
	"AdvanceSpellcastingCommand", "CalculateSpellcastingCommand", "PreparationRejection", "SlotOperationRejection", "SpellcastingAdvanceRejection"}

// Requirement: doc/specs/spellcasting.md FR-084, FR-085, FR-088, FR-143 - the spellcasting domain files
// import neither telemetry nor context nor sync nor time and start no goroutine, and the use case files
// call no other use case.
// Path: unhappy (architecture).
func TestSpellcastingDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender", "NewGainHeroicInspiration",
		"NewExpendHeroicInspiration", "NewCalculateSpellcasting", "NewCreateSpellcasting", "NewChooseAdditionalSpells", "NewChangePreparedSpells", "NewExpendSpellSlot", "NewRegainSpellSlot",
		"NewRestoreSpellSlots", "NewAdvanceSpellcasting"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(spellcastingDomainFiles, name), slices.Contains(spellcastingUseCaseFiles, name)
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
	if want := len(spellcastingDomainFiles) + len(spellcastingUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/spellcasting.md FR-136 - package character exports every type a spellcasting
// command or result names, and none of the spellcasting files exports a name another package of the module
// also exports.
// Path: unhappy (name collision).
func TestSpellcastingExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(spellcastingDomainFiles, spellcastingUseCaseFiles) {
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
	for _, n := range spellcastingRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s (FR-136)", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
