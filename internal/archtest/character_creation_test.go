package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// ccDomainFiles are the files of package character that hold the character creation rules: pure
// domain code that imports neither telemetry nor context nor sync nor time. ccUseCaseFiles hold
// every telemetry call and every content or dice port call.
var (
	ccDomainFiles  = []string{"ability_score_assignment", "background_ability_increase", "languages", "creation_choices", "creation_problem", "creation_record", "create_character_command", "creation_rules"}
	ccUseCaseFiles = []string{"create_character_usecase", "roll_ability_scores_usecase", "roll_languages_usecase"}
)

// ccRequiredNames are the types and functions the spec requires package character to export (section 6, "Exports").
var ccRequiredNames = []string{"CreateCharacter", "NewCreateCharacter", "CreateCharacterCommand", "CreateCharacterResult", "CreateCharacterSucceeded", "CreateCharacterRejected",
	"NewCreateCharacterRejected", "CreateCharacterContentMissing", "CreationRecord", "CreateCharacterProblem", "CreateCharacterProblemKind", "CreationStep", "StartingEquipmentGrant",
	"RollAbilityScores", "NewRollAbilityScores", "RollAbilityScoresCommand", "RollAbilityScoresResult", "RollAbilityScoresSucceeded", "RolledAbilityScores", "RolledAbilityScore",
	"RollLanguages", "NewRollLanguages", "RollLanguagesCommand", "RollLanguagesResult", "RollLanguagesSucceeded", "RollLanguagesDuplicate", "Languages", "NewLanguages", "StandardLanguageRoll",
	"AbilityScoreAssignment", "NewAbilityScoreAssignment", "MustAbilityScoreAssignment", "AbilityScoreAssignmentProblem", "StandardArrayScores", "PointCostOf", "PointCostBudget",
	"BackgroundAbilityIncrease", "PlusTwoPlusOne", "NewPlusTwoPlusOne", "MustPlusTwoPlusOne", "PlusOneToAll", "BackgroundIncreaseResult", "BackgroundIncreaseApplied", "BackgroundIncreaseRefused",
	"ToolSelection", "NoToolSelection", "ChosenTool", "EquipmentOptionLetter", "SizeChoice", "NoSizeChoice", "ChosenSize", "SpeciesTraitSelection", "SpeciesGrantSelections",
	"NewSpeciesGrantSelections", "HigherLevelEquipmentChoice", "DeclineHigherLevelEquipment", "GrantHigherLevelEquipment", "TrinketChoice", "NoTrinket", "RollForTrinket", "TrinketRoll", "NewTrinketRoll"}

// Requirement: doc/specs/character-creation.md FR-113, FR-114 and section 8 "architecture" - the creation domain files import neither telemetry
// nor context nor sync nor time and start no goroutine; no creation use case file names another use case or constructs one, or calls a Run
// or Do method (CreateCharacter composes pure domain services only).
// Path: unhappy (architecture, nested use cases).
func TestCharacterCreationDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	others := []string{"StartAtHigherLevel", "RollHigherLevelStartingMoney", "RollTrinket", "TakeFeat", "LevelUpCharacter", "NewStartAtHigherLevel", "NewRollHigherLevelStartingMoney",
		"NewRollTrinket", "NewTakeFeat", "NewLevelUpCharacter", "ChooseSubclass", "NewChooseSubclass", "PreviewLevelAdvancement", "NewPreviewLevelAdvancement"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(ccDomainFiles, name), slices.Contains(ccUseCaseFiles, name)
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
				case *ast.Ident:
					if useCase && slices.Contains(others, n.Name) {
						t.Errorf("%s names the use case %s at %v", f.Path, n.Name, f.Fset.Position(n.Pos()))
					}
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && (sel.Sel.Name == "Run" || sel.Sel.Name == "Do") {
						t.Errorf("%s calls a %s method at %v", f.Path, sel.Sel.Name, f.Fset.Position(n.Pos()))
					}
				}
				return true
			})
		}
	}
	if want := len(ccDomainFiles) + len(ccUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/character-creation.md section 6 "Exports" and section 8 "architecture" - package character exports every type of the
// spec, none of the creation files exports a generic variant name or a name another package of the module also exports, and the internal
// CharacterCreationRules service is not exported.
// Path: unhappy (name collision).
func TestCharacterCreationExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(ccDomainFiles, ccUseCaseFiles, []string{"creation_signals"}) {
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
	for _, n := range ccRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s", n)
		}
	}
	for n := range own {
		if slices.Contains([]string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound", "CharacterCreationRules", "CreationRules"}, n) {
			t.Errorf("character exports %s, a generic name or the internal service", n)
		}
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-120 and character-creation.md FR-082, FR-091 - CreateCharacter gets
// the allowance money through TotalGold and never reads the Trinkets table: the creation files call TotalGold and name neither EntryFor nor
// TrinketTable, because the record holds the roll only.
// Path: unhappy (architecture).
func TestCharacterCreationUsesTotalGoldAndNeverReadsTheTrinketTable(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	totalGold := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			if !slices.Contains(ccDomainFiles, name) && !slices.Contains(ccUseCaseFiles, name) {
				continue
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					switch id.Name {
					case "TotalGold":
						totalGold++
					case "EntryFor", "TrinketTable", "NewTrinketTable":
						t.Errorf("%s names %s", f.Path, id.Name)
					}
				}
				return true
			})
		}
	}
	if totalGold == 0 {
		t.Error("no creation file calls TotalGold")
	}
}
