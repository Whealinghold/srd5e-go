package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// speciesDomainFiles are the files of package character that hold the species trait rules;
// none imports telemetry or context. The use case files hold every telemetry call.
var speciesDomainFiles = []string{"species_ancestry", "species_choices", "species_traits", "species_use", "species_uses", "species_effects", "species_passives",
	"species_granted_spells", "species_stones_endurance", "species_relentless_endurance", "species_halfling_luck", "species_high_elf_cantrip", "species_event", "species_rejection"}

var speciesUseCaseFiles = []string{"expend_species_trait_use_usecase", "renew_species_trait_uses_usecase", "use_relentless_endurance_usecase", "roll_halfling_luck_usecase",
	"roll_stones_endurance_usecase", "replace_high_elf_cantrip_usecase"}

// speciesRequiredNames are types the spec requires package character to export (section 6).
var speciesRequiredNames = []string{"SpeciesTraits", "SpeciesTraitChoices", "DragonbornChoices", "ElfChoices", "GnomeChoices", "GoliathChoices", "TieflingChoices", "HumanChoices",
	"NoSpeciesChoices", "DraconicAncestry", "ElvenLineage", "GnomishLineage", "GiantAncestry", "FiendishLegacy", "SpeciesChoiceProblem", "SpeciesUse", "SpeciesUseRequest",
	"SpeciesWindow", "SpeciesTraitEffect", "GiantAncestryEffect", "BreathWeaponEffect", "TemporaryHitPointsGrant", "SpeciesAdvantage", "SpeciesPassives", "SpeciesGrantedSpells",
	"SpeciesUses", "SpeciesEffects", "StonesEnduranceRule", "RelentlessEndurance", "HalflingLuck", "HighElfCantripSwap", "ZeroHitPointsOutcome", "SpeciesTraitEvent",
	"SpeciesTraitRejection", "SpeciesBehaviourUnknown", "ExpendSpeciesTraitUse", "ExpendSpeciesTraitUseResult", "RenewSpeciesTraitUses", "RenewSpeciesTraitUsesResult",
	"UseRelentlessEndurance", "UseRelentlessEnduranceResult", "RollHalflingLuck", "RollHalflingLuckResult", "RollStonesEndurance", "RollStonesEnduranceResult",
	"ReplaceHighElfCantrip", "ReplaceHighElfCantripResult", "ReplaceHighElfCantripContentMissing"}

// speciesForbiddenNames are generic variant names (architecture.md section 4).
var speciesForbiddenNames = []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"}

// Requirement: doc/specs/species-trait-behaviour.md FR-064, FR-065 and section 8 "architecture" - the species domain files import neither telemetry nor context nor sync nor time, start no goroutine and declare no type alias; the use case files call no other use case.
// Path: unhappy (architecture).
func TestSpeciesDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewCalculateDerivedStats", "NewRestTaker", "NewGainHeroicInspiration",
		"NewExpendHeroicInspiration", "NewTakeFeat", "NewExpendFeatUse", "NewRenewFeatUses", "NewExpendSpeciesTraitUse", "NewRenewSpeciesTraitUses", "NewUseRelentlessEndurance",
		"NewRollHalflingLuck", "NewRollStonesEndurance", "NewReplaceHighElfCantrip"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(speciesDomainFiles, name), slices.Contains(speciesUseCaseFiles, name)
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
	if want := len(speciesDomainFiles) + len(speciesUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/species-trait-behaviour.md section 6 "Exports", FR-063 - package character exports every type of the spec, none of the species files exports a name another package of the module also exports, and none uses a generic variant name.
// Path: unhappy (name collision).
func TestSpeciesExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(speciesDomainFiles, speciesUseCaseFiles) {
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
	for _, n := range speciesRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s", n)
		}
	}
	for _, n := range speciesForbiddenNames {
		if own[n] {
			t.Errorf("character exports %s, a generic name", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
