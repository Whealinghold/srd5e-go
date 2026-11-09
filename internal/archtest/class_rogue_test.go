package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// rogueFiles are the files of package character that hold the Rogue code. The Rogue is domain only (class-rogue.md FR-109): no use case,
// no telemetry and no port.
var rogueFiles = []string{
	"rogue_level", "rogue_problems", "rogue_progression", "rogue_sneak_attack", "rogue_cunning_strike", "rogue_features", "rogue_expertise",
	"rogue_choices", "rogue_creation_choices", "rogue_state", "rogue_stroke_of_luck", "rogue_event", "rogue_thief",
}

// Requirement: doc/specs/class-rogue.md FR-104 - the Rogue files define none of the types other specs own (SubclassChosen,
// WeaponMasteryChoice, RestApplied, Roll, ProficiencyBonus, ClassLevels, SkillProficiencies, AttunementLimit, AnyScrollRoute), declare the named
// services and results and no generic variant name, and none of their exported names equals an export of another package of the module.
// Path: unhappy (name collision).
func TestRogueNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range rogueFiles {
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
	if len(own) < 100 {
		t.Fatalf("only %d exported names found in the Rogue files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "WeaponMasteryChoice", "RestApplied", "Roll", "ProficiencyBonus", "ClassLevels", "SkillProficiencies", "AttunementLimit", "AnyScrollRoute", "RogueClassFacts"} {
		if own[owned] {
			t.Errorf("the Rogue files define %s, which another spec owns or which is test-only", owned)
		}
	}
	for _, generic := range []string{"Available", "Unavailable", "Applied", "Rejected", "Accepted", "Spent", "Granted", "NotAvailable"} {
		if own[generic] {
			t.Errorf("the Rogue files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"RogueLevel", "RogueProgression", "RogueState", "RogueStateKey", "RogueChoices", "RogueCreationChoices", "RogueExpertise", "SneakAttack", "SneakAttackApplies", "SneakAttackDoesNotApply",
		"NewCunningStrikeSelection", "CunningStrikeOutcome", "RogueGrantedActions", "SteadyAim", "RogueUncannyDodge", "RogueEvasion", "RogueReliableTalent", "RogueElusive", "StrokeOfLuck",
		"StrokeOfLuckSpent", "StrokeOfLuckNotAvailable", "StrokeOfLuckUsed", "RogueEvent", "ThiefFeatures", "UseMagicDevice", "ThiefReflexes", "SecondStoryWorkGranted", "AttunementLimitSet",
	} {
		if !own[want] {
			t.Errorf("the Rogue files declare no %s", want)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-101, FR-109, FR-112 - the Rogue files import neither telemetry, context nor sync, read no clock (no
// call of time.Now, time.Since or time.Until), start no goroutine and import no combat package, so they are deterministic, side-effect free
// pure functions of their arguments and emit no signal.
// Path: unhappy (architecture).
func TestRogueIsPure(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !slices.Contains(rogueFiles, strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")) {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if strings.HasSuffix(imp.Path, "/combat") || imp.Path == "context" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.CallExpr:
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
						if x, isPkg := sel.X.(*ast.Ident); isPkg && x.Name == "time" && (sel.Sel.Name == "Now" || sel.Sel.Name == "Since" || sel.Sel.Name == "Until") {
							t.Errorf("%s reads the clock with time.%s", f.Path, sel.Sel.Name)
						}
					}
				}
				return true
			})
		}
	}
	if seen != len(rogueFiles) {
		t.Errorf("found %d of %d files", seen, len(rogueFiles))
	}
}
