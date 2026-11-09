package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

var rangerDomainFiles = []string{
	"ranger_level", "ranger_reason", "ranger_progression", "ranger_state", "ranger_event", "ranger_rules", "ranger_choices", "ranger_creation_choices",
}

var rangerUseCaseFiles = []string{
	"ranger_usecase", "use_choose_ranger_option", "use_resolve_ranger_features", "use_tireless",
}

// Requirement: doc/specs/class-ranger.md FR-112, FR-113 - the Ranger files define none of the types other specs own, declare the named results, services and events with no generic variant name, none of their exported names
// equals an export of another package of the module, and the feature state key is named ranger_state, which no other production file uses.
// Path: unhappy (name collision).
func TestRangerNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(rangerDomainFiles, rangerUseCaseFiles, []string{"ranger_signals"}) {
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
		t.Fatalf("only %d exported names found in the Ranger files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"Applied", "Rejected", "ContentMissing", "RestApplied", "SubclassChosen", "SubclassRejected", "WeaponMasteryChoice", "FightingStyleChoice", "ClassDataInvalid", "MissingContent", "GrantedCantrip", "ResistanceGrant"} {
		if own[owned] {
			t.Errorf("the Ranger files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "NoEffect", "NotARanger", "ContentMissing", "ClassDataInvalid", "Available", "Spent", "Accepted", "Summary", "Started", "Resolved", "NotMarked"} {
		if own[generic] {
			t.Errorf("the Ranger files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"RangerLevel", "RangerProgression", "RangerState", "RangerStateKey", "RangerStateParts", "RangerCreationChoices", "RangerReason", "RangerRiderReason", "RangerEvent", "RangerFeature", "RangerFightingStyle", "RangerFightingStyleFeat",
		"DruidicWarrior", "DeftExplorerChoice", "RangerChoice", "RangerChoiceRules", "RangerChoiceFacts", "RangerChoiceAccepted", "RangerChoiceRefused", "RangerFacts", "RangerActionCost", "RangerReplacementMoment", "SpeedInput", "SpeedProfile",
		"RangerSenseGrant", "RangerResistanceGrant", "RangerConditionGrant", "RangerGrantExpiry", "MarkStatus", "AttackRider", "RiderDamageType", "AttackerKey", "MultiattackDefenseRecord", "HuntersLoreResult", "RangerRiderEligibility",
		"UseFavoredEnemy", "UseNaturesVeil", "UseSuperiorHuntersDefense", "ApplyRangerRest", "TirelessTemporaryHitPoints", "TirelessUses", "ChooseRangerOption", "ResolveRangerFeatures", "UseTireless",
		"ChooseRangerOptionAccepted", "ChooseRangerOptionRejected", "ChooseRangerOptionNotARanger", "ResolveRangerFeaturesResolved", "ResolveRangerFeaturesNotARanger", "UseTirelessAccepted", "UseTirelessRejected", "UseTirelessNotARanger",
		"FavoredEnemyUseAccepted", "FavoredEnemyUseRejected", "NaturesVeilAccepted", "NaturesVeilRejected", "SuperiorHuntersDefenseAccepted", "SuperiorHuntersDefenseRejected",
		"FavoredEnemyCastSpent", "TirelessUsed", "NaturesVeilUsed", "ExhaustionReductionGranted", "RangerChoiceMade", "RangerChoiceReplaced",
	} {
		if !own[want] {
			t.Errorf("the Ranger files declare no %s", want)
		}
	}
	count := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Value == `"ranger_state"` {
					count++
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Errorf("key name ranger_state is used %d times in package character, want 1", count)
	}
}

// Requirement: doc/specs/class-ranger.md FR-111, FR-104, shared-types-and-ownership.md FR-044 - the Ranger domain files import neither telemetry, context, time nor sync, start no goroutine, read no ContentSource and use no dice roller, so they are pure
// functions of their arguments (the one Ranger feature that rolls, Tireless, has its use case); the use case files call no other use case (neither a constructor nor a Run).
// Path: unhappy (architecture).
func TestRangerDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewRestTaker", "NewUseSecondWind", "NewChooseRangerOption", "NewResolveRangerFeatures", "NewUseTireless"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(rangerDomainFiles, name), slices.Contains(rangerUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry")) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
				if strings.HasSuffix(imp.Path, "/dice/randroller") || strings.HasSuffix(imp.Path, "/dice/dicetest") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.Ident:
					if domain && (n.Name == "ContentSource" || n.Name == "Roller") {
						t.Errorf("%s names %s", f.Path, n.Name)
					}
					if useCase && n.Name == "ContentSource" {
						t.Errorf("%s names ContentSource", f.Path)
					}
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Run" {
						t.Errorf("%s calls a Run method", f.Path)
					}
				}
				return true
			})
		}
	}
	if seen != len(rangerDomainFiles)+len(rangerUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(rangerDomainFiles)+len(rangerUseCaseFiles))
	}
}
