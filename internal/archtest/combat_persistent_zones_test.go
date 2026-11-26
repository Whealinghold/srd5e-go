package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// persistentZoneFiles returns the production files of the persistent zone model
// in combat (persistent-spell-zones section 5): zone_*.go and persistent_zone.go.
func persistentZoneFiles(t *testing.T) []archtestFile {
	t.Helper()
	var out []archtestFile
	for _, f := range combatPackage(t).Production() {
		if strings.HasPrefix(f.Path, "combat/zone_") || f.Path == "combat/persistent_zone.go" {
			out = append(out, archtestFile{path: f.Path, ast: f.AST})
		}
	}
	if len(out) < 10 {
		t.Fatalf("found %d persistent zone files, want the zone model's files", len(out))
	}
	return out
}

// archtestFile is one production file of the persistent zone model.
type archtestFile struct {
	path string
	ast  *ast.File
}

// Requirement: doc/specs/persistent-spell-zones.md FR-049 - the zone files import only the
// kernel srderr and the content context from this module (no telemetry, character
// services, toolbox or encounter code; architecture.md section 1 and 2).
// Path: unhappy (forbidden edge).
func TestPersistentZoneFilesImportOnlyContentAndSrderr(t *testing.T) {
	t.Parallel()
	allowed := []string{"github.com/Whealinghold/srd5e-go/content", "github.com/Whealinghold/srd5e-go/srderr"}
	for _, f := range persistentZoneFiles(t) {
		for _, imp := range f.ast.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: import %s: %v", f.path, imp.Path.Value, err)
			}
			if strings.HasPrefix(p, "github.com/Whealinghold/srd5e-go/") && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s; the persistent zone model may import only content and srderr [FR-049]", f.path, p)
			}
		}
	}
}

// Requirement: doc/specs/persistent-spell-zones.md FR-051 - the zone files name no spell: no
// string literal is a spell id from table Z, so every behaviour comes from the zone
// facts (spell-effect-data.md section 5.1).
// Path: unhappy (rule interaction).
func TestPersistentZoneFilesNameNoSpell(t *testing.T) {
	t.Parallel()
	spells := []string{"spirit_guardians", "cloudkill", "incendiary_cloud", "insect_plague", "moonbeam",
		"flaming_sphere", "spike_growth", "wall_of_fire", "wall_of_thorns", "black_tentacles", "blade_barrier",
		"web", "sleet_storm", "stinking_cloud", "conjure_woodland_beings", "conjure_celestial", "conjure_animals",
		"guardian_of_faith", "grease", "gust_of_wind", "control_water", "forbiddance", "zone_of_truth",
		"prismatic_wall", "symbol"}
	for _, f := range persistentZoneFiles(t) {
		ast.Inspect(f.ast, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			if v, err := strconv.Unquote(lit.Value); err == nil && slices.Contains(spells, v) {
				t.Errorf("%s contains the spell id literal %q [FR-051]", f.path, v)
			}
			return true
		})
	}
}
