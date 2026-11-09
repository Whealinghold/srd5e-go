package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// fighterDomainFiles are the files of package character that hold the Fighter domain code; the use case files, feature_signals.go and
// fighter_signals.go are not among them.
var fighterDomainFiles = []string{
	"fighter_level", "fighter_reason", "fighter_progression", "fighter_state", "fighter_second_wind", "fighter_tactical_mind",
	"fighter_indomitable", "fighter_action_surge", "fighter_effects", "fighter_champion", "fighter_choice_rules", "fighter_event", "fighter_usecase",
}

var fighterUseCaseFiles = []string{"use_second_wind", "use_tactical_mind", "use_indomitable", "use_action_surge", "apply_fighter_option_choice"}

// Requirement: doc/specs/class-fighter.md FR-147, FR-154, FR-161 - the Fighter files define none of the types other specs own (SubclassChosen,
// FightingStyleChoice, WeaponMasteryChoice, WeaponMasteryAccess, TakenFeats, FeatTakingRules, RestApplied, MissingContent, HeroicInspiration, Roll,
// ProficiencyBonus, ClassLevels), declare the named domain-service types of FR-161 and no generic variant name, and none of their exported names
// equals an export of another package of the module.
// Path: unhappy (name collision).
func TestFighterNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(fighterDomainFiles, fighterUseCaseFiles) {
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
		t.Fatalf("only %d exported names found in the Fighter files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "FightingStyleChoice", "WeaponMasteryChoice", "WeaponMasteryAccess", "TakenFeats", "FeatTakingRules", "RestApplied", "MissingContent", "HeroicInspiration", "Roll", "ProficiencyBonus", "ClassLevels"} {
		if own[owned] {
			t.Errorf("the Fighter files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Available", "Unavailable", "AttackKind", "Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"} {
		if own[generic] {
			t.Errorf("the Fighter files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"SecondWindAvailability", "SecondWindAvailable", "SecondWindUnavailable", "SecondWindResolution", "TacticalMindAvailability", "TacticalMindAvailable", "TacticalMindUnavailable",
		"FighterChoice", "FighterChoiceFightingStyle", "FighterChoiceWeaponMasteries", "FighterChoiceAdditionalFightingStyle", "TacticalMindResolution", "TacticalMindResolutionSucceeded",
		"TacticalMindResolutionStillFailed", "IndomitableAvailability", "IndomitableAvailable", "IndomitableUnavailable", "IndomitableResolution", "IndomitableResolutionSucceeded",
		"IndomitableResolutionStillFailed", "ActionSurgeEvaluation", "ActionSurgeGranted", "ActionSurgeRefused", "FighterChoiceVerdict", "FighterChoiceAccepted", "FighterChoiceRefused",
		"ChampionAttackKind", "FighterTurnOwnership", "FighterPerTurnUse", "FighterCombatPresence", "ChampionAdvantageSource", "ChampionInitiativeAdvantage", "ChampionSkillCheckAdvantage",
		"FighterEvent", "UseSecondWind", "UseTacticalMind", "UseIndomitable", "UseActionSurge", "ApplyFighterOptionChoice",
	} {
		if !own[want] {
			t.Errorf("the Fighter files declare no %s", want)
		}
	}
}

// Requirement: doc/specs/class-fighter.md FR-146, FR-149, FR-150, shared-types-and-ownership.md FR-044 - the Fighter domain files import neither telemetry nor context, read no
// clock and start no goroutine, so they are pure functions of their arguments; the five use case files call no other use case (neither a constructor nor a Run).
// Path: unhappy (architecture).
func TestFighterDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewExpendFeatUse", "NewRenewFeatUses", "NewRestTaker", "NewHitDieSpender",
		"NewGainHeroicInspiration", "NewExpendHeroicInspiration", "NewUseSecondWind", "NewUseTacticalMind", "NewUseIndomitable", "NewUseActionSurge", "NewApplyFighterOptionChoice"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(fighterDomainFiles, name), slices.Contains(fighterUseCaseFiles, name)
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
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Run" {
						t.Errorf("%s calls a Run method", f.Path)
					}
				}
				return true
			})
		}
	}
	if seen != len(fighterDomainFiles)+len(fighterUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(fighterDomainFiles)+len(fighterUseCaseFiles))
	}
}
