package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// levelAdvancementDomainFiles are the files of package character that hold the level
// advancement and multiclassing rules; none imports telemetry or context. The use case
// files hold every telemetry call.
var levelAdvancementDomainFiles = []string{"ability_score_increase", "subclass_selection", "level_advancement_event", "multiclass_training",
	"extra_attack_combination", "advancement_class_status", "level_advancement_preview"}

var levelAdvancementUseCaseFiles = []string{"preview_level_advancement_usecase", "choose_subclass_usecase"}

// levelAdvancementRequiredNames are the names FR-103 requires package character to export.
var levelAdvancementRequiredNames = []string{"AbilityScoreIncrease", "AbilityScoreIncreaseRejection", "AbilityScoreIncreaser", "IncreaseExceedsMaximum", "ChosenSubclass",
	"SubclassStateKey", "SubclassSelection", "SubclassRejected", "SubclassRejectionReason", "LevelAdvancementEvent", "SubclassChosen", "LevelAdvancementPreviewer",
	"LevelAdvancementPreview", "LevelAdvancementGain", "LevelAdvancementBlock", "AdvancementClassStatus", "AdvancementClassStatusExisting", "AdvancementClassStatusNew",
	"PendingAdvancementChoice", "PendingSubclassChoice", "PendingFeatGrant", "MulticlassTraining", "ExtraAttackCombination", "PreviewLevelAdvancement",
	"PreviewLevelAdvancementCommand", "PreviewLevelAdvancementResult", "PreviewLevelAdvancementAvailable", "PreviewLevelAdvancementUnavailable",
	"PreviewLevelAdvancementContentMissing", "ChooseSubclass", "ChooseSubclassCommand", "ChooseSubclassResult", "ChooseSubclassSucceeded", "ChooseSubclassRejected",
	"ChooseSubclassContentMissing"}

// levelAdvancementForbiddenNames are types other specs own; this spec must not redefine them (FR-104).
var levelAdvancementForbiddenNames = []string{"LevelAdvancementStep", "CharacterEvent", "AbilityScoreImprovementChoices", "SingleAbilityIncrease", "FeatGrant",
	"ClassDefinition", "SubclassDefinition", "MissingContent", "SubclassChoice", "Applied", "Rejected", "Succeeded", "ContentMissing"}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-103, FR-117 - the level
// advancement domain files import neither telemetry nor context nor sync nor time and
// start no goroutine, and the use case files call no other use case.
// Path: unhappy (architecture).
func TestLevelAdvancementDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender",
		"NewGainHeroicInspiration", "NewExpendHeroicInspiration", "NewCalculateSpellcasting", "NewCreateSpellcasting", "NewChooseAdditionalSpells", "NewChangePreparedSpells",
		"NewExpendSpellSlot", "NewRegainSpellSlot", "NewRestoreSpellSlots", "NewAdvanceSpellcasting", "NewPreviewLevelAdvancement", "NewChooseSubclass"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(levelAdvancementDomainFiles, name), slices.Contains(levelAdvancementUseCaseFiles, name)
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
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && (sel.Sel.Name == "Run" || sel.Sel.Name == "Do") {
						t.Errorf("%s calls a %s method", f.Path, sel.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if want := len(levelAdvancementDomainFiles) + len(levelAdvancementUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-103, FR-104 - package
// character exports every type of the spec, none of the level advancement files exports a
// name another package of the module also exports, and none redefines a type another
// spec owns or a generic variant name.
// Path: unhappy (name collision).
func TestLevelAdvancementExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(levelAdvancementDomainFiles, levelAdvancementUseCaseFiles) {
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
	for _, n := range levelAdvancementRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s (FR-103)", n)
		}
	}
	for _, n := range levelAdvancementForbiddenNames {
		if own[n] {
			t.Errorf("character exports %s, a name another spec owns or a generic name (FR-104)", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
