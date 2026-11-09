package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// explorationFiles are the files of package character that hold the exploration
// rules (travel and light sources); none is a use case.
var explorationFiles = []string{"exploration_error", "travel_pace", "vehicle_speed", "travel_mode", "travel_rules", "mounted_pace_boost",
	"light_emission", "light_source_rules", "fuel_supply"}

// explorationRequiredNames are the names FR-045 and FR-053 require package character to export.
var explorationRequiredNames = []string{"TravelPace", "TravelPaceRow", "TravelMode", "LandTravel", "VehicleTravel", "VehicleSpeed", "TravelRules",
	"MountedPaceBoost", "MountedBoostSpendResult", "MountedBoostSpendAccepted", "MountedBoostSpendRefused", "MountedBoostRefusalReason",
	"MountedBoostRestResult", "MountedBoostRestRestored", "MountedBoostRestUnchanged", "MountedPaceBoostEvent", "MountedPaceBoostSpent",
	"MountedPaceBoostRestored", "LightEmission", "LightShape", "LightSpace", "SrdLightSources", "HoodedLanternHood", "LightSourceRules",
	"FuelKind", "FuelSupply", "FuelBurnResult", "FuelBurnContinues", "FuelBurnExhausted", "LightFuelEvent", "LightFuelExhausted"}

// explorationForbiddenNames are the names another spec owns (FR-043); the exploration
// files must not declare them.
var explorationForbiddenNames = []string{"LightLevel", "LightRules", "Obscurement", "SpecialSense", "SightRules", "HideRules", "SearchRules", "ObjectStatus",
	"HazardKind", "CarryingCapacity", "RestKind", "Skill", "TravelRates", "StandardPaceRates", "SpecialMovementRates", "PaceCheckEffect", "TravelByVehicle",
	"Applied", "Rejected", "Succeeded"}

// Requirement: doc/specs/exploration.md FR-044, FR-054 - the exploration files import
// only content, modifier and srderr from the module and none of context, sync, os,
// math/rand, reflect, telemetry or a *test package, start no goroutine and read no clock.
// Path: unhappy (forbidden edge).
func TestExplorationFilesArePure(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !slices.Contains(explorationFiles, strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")) {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				p := imp.Path
				if slices.Contains([]string{"context", "sync", "os", "math/rand", "math/rand/v2", "reflect"}, p) || strings.Contains(p, "telemetry") || strings.HasSuffix(p, "test") ||
					(strings.Contains(p, "srd5e-go/") && !slices.ContainsFunc([]string{"/content", "/modifier", "/srderr"}, func(s string) bool { return strings.HasSuffix(p, s) })) {
					t.Errorf("%s imports %s", f.Path, p)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement", f.Path)
				case *ast.SelectorExpr:
					if id, ok := n.X.(*ast.Ident); ok && id.Name == "time" && slices.Contains([]string{"Now", "Since", "Until", "Sleep", "After", "Tick", "NewTimer"}, n.Sel.Name) {
						t.Errorf("%s reads the clock: time.%s", f.Path, n.Sel.Name)
					}
				}
				return true
			})
		}
	}
	if seen != len(explorationFiles) {
		t.Errorf("found %d of %d exploration files", seen, len(explorationFiles))
	}
}

// Requirement: doc/specs/exploration.md FR-043, FR-045, FR-051, FR-053 - the exploration
// files export every required name, declare none another spec owns, and export no name
// that another package of the module also exports.
// Path: unhappy (name collision).
func TestExplorationExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range explorationFiles {
		files["character/"+f+".go"] = true
	}
	var own map[string]bool
	others := map[string][]string{}
	for _, pkg := range m.Packages {
		switch pkg.Dir {
		case "character":
			own = exportedNames(pkg, files)
			for _, n := range explorationForbiddenNames {
				if own[n] {
					t.Errorf("an exploration file declares %s (FR-043)", n)
				}
			}
		case "content", "d20", "ability", "dice", "modifier", "srderr", "telemetry":
			for n := range exportedNames(pkg, nil) {
				others[n] = append(others[n], pkg.Dir)
			}
		}
	}
	for _, n := range explorationRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s (FR-045, FR-053)", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
