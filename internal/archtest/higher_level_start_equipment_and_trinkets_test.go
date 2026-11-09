package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// hlsDomainFiles are the files of package character that hold the higher-level start, equipment
// and Trinket rules; none imports telemetry, context, sync or time. The use case files hold every
// telemetry call.
var hlsDomainFiles = []string{"higher_level_step", "higher_level_start_rejection", "higher_level_start", "higher_level_equipment", "trinket"}

var hlsUseCaseFiles = []string{"start_at_higher_level_usecase", "roll_higher_level_starting_money_usecase", "roll_trinket_usecase"}

// hlsRequiredNames are the types the spec requires package character to export (FR-123, section 6).
var hlsRequiredNames = []string{"HigherLevelStart", "HigherLevelStep", "StartingLevelStep", "ConsumedFeatGrant", "HigherLevelStartRejection",
	"HigherLevelStartPlanLengthMismatch", "HigherLevelStartCreationInvalid", "HigherLevelStartLevelUpRejected", "HigherLevelStartSubclassRejected",
	"HigherLevelStartIncreaseRejected", "HigherLevelStartIncreaseNotGranted", "HigherLevelStartSubclassRequired", "HigherLevelEquipment",
	"NoHigherLevelEquipment", "HigherLevelEquipmentAllowance", "HigherLevelEquipmentRules", "RolledGold", "Trinket", "TrinketTable", "NewTrinketTable", "MustTrinketTable",
	"StartAtHigherLevel", "StartAtHigherLevelCommand", "StartAtHigherLevelResult", "StartAtHigherLevelStarted", "StartAtHigherLevelRejected", "StartAtHigherLevelContentMissing",
	"RollHigherLevelStartingMoney", "RollHigherLevelStartingMoneyCommand", "RollHigherLevelStartingMoneyResult", "RollHigherLevelStartingMoneyRolled", "RollHigherLevelStartingMoneyNotApplicable",
	"RollTrinket", "RollTrinketCommand", "RollTrinketResult", "RollTrinketRolled"}

// hlsForbiddenNames are generic variant names and the names of types another spec owns or that
// this spec replaced (FR-115, FR-123).
var hlsForbiddenNames = []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "LevelAdvancementStep", "HigherLevelEquipmentChoice", "TrinketChoice", "TrinketRoll",
	"ItemRarity", "FeatGrant", "ClassDefinition", "MissingContent", "StandardEquipmentOnly", "StartingEquipmentAtHigherLevels", "HigherLevelStarted", "HigherLevelMoneyRolled",
	"HigherLevelMoneyNotApplicable", "TrinketRolled"}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-120, FR-123 and section 8 "architecture" - the domain files import
// neither telemetry nor context nor sync nor time, start no goroutine and declare no type alias; the use case files call no other use case.
// Path: unhappy (architecture, nested telemetry).
func TestHigherLevelStartDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewHitDieSpender", "NewGainHeroicInspiration",
		"NewExpendHeroicInspiration", "NewPreviewLevelAdvancement", "NewChooseSubclass", "NewTakeFeat", "NewChangeMagicInitiateSpell", "NewExpendFeatUse", "NewRenewFeatUses", "NewRollFreeCasting",
		"NewStartAtHigherLevel", "NewRollHigherLevelStartingMoney", "NewRollTrinket"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(hlsDomainFiles, name), slices.Contains(hlsUseCaseFiles, name)
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
				case *ast.TypeSpec:
					if n.Assign.IsValid() {
						t.Errorf("%s: type alias %s", f.Path, n.Name.Name)
					}
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
	if want := len(hlsDomainFiles) + len(hlsUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-115, FR-123 and section 6 "Exports" - package character exports every type
// of the spec, none of these files exports a name another package of the module also exports, and none uses a generic variant name or a type
// another spec owns or this spec replaced.
// Path: unhappy (name collision).
func TestHigherLevelStartExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(hlsDomainFiles, hlsUseCaseFiles) {
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
	for _, n := range hlsRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s", n)
		}
	}
	for _, n := range hlsForbiddenNames {
		if own[n] {
			t.Errorf("character exports %s, a generic name or a type another spec owns", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
