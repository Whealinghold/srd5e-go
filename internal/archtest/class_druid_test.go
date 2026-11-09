package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// druidFiles are the files of package character that hold the Druid code. The Druid is domain only (class-druid.md FR-164, FR-165): no use
// case, no telemetry and no port.
var druidFiles = []string{
	"druid_level", "druid_reason", "druid_enums", "druid_tables", "druid_state", "druid_event", "druid_rules", "druid_gain_level", "druid_choice",
	"druid_use", "druid_trigger", "druid_creation_choices",
}

// Requirement: doc/specs/class-druid.md FR-166, FR-173 - the Druid files define none of the types other specs own (SubclassChosen, RestApplied, FightingStyleChoice, WeaponMasteryChoice, SpellSlotPool,
// SlotOperationRejection, Concentration, ChallengeRating), none of the names FR-173 forbids (ConcentrationEffect, AttackKind, ActionCost, PotentSpellcasting, FeatChoiceSlot, SpellFacts), declare the
// named services, results, commands and events and no generic variant name, none of their exported names equals an export of another package of the module, and the feature state key is named
// druid, which no other production file uses.
// Path: unhappy (name collision).
func TestDruidNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range druidFiles {
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
	if len(own) < 150 {
		t.Fatalf("only %d exported names found in the Druid files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "RestApplied", "FightingStyleChoice", "WeaponMasteryChoice", "SpellSlotPool", "SlotOperationRejection", "Concentration", "ChallengeRating", "CharacterID",
		"ConcentrationEffect", "AttackKind", "ActionCost", "PotentSpellcasting", "FeatChoiceSlot", "SpellFacts"} {
		if own[owned] {
			t.Errorf("the Druid files define %s, which another spec owns or FR-173 forbids", owned)
		}
	}
	for _, generic := range []string{"Available", "Unavailable", "Applied", "Rejected", "Accepted", "Eligible", "Readable", "ContentMissing", "NotApplicable", "Spend", "Regain"} {
		if own[generic] {
			t.Errorf("the Druid files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"DruidLevel", "DruidRejection", "PrimalOrder", "ElementalFury", "LandType", "DruidActionCost", "RestWindowPermission", "BeastFormFacts", "KnownForms", "DruidItemKey", "EquipmentDisposition",
		"DruidFeatureTables", "DruidFeatureState", "DruidFeatureStateKey", "DruidCharacterFacts", "DruidCreationChoices", "DruidRules", "DruidEvent", "GainDruidLevel", "ApplyDruidChoice", "UseDruidFeature",
		"ApplyDruidTrigger", "GainDruidLevelApplied", "GainDruidLevelRejected", "ApplyDruidChoiceApplied", "ApplyDruidChoiceRejected", "UseDruidFeatureUsed", "UseDruidFeatureRejected",
		"ApplyDruidTriggerApplied", "DruidSlotChangeSpend", "DruidSlotChangeRegain", "BeastFormCastCheckBlocked", "PotentBonusApplied", "RangeChangeIncreased", "PrimalStrikeOutcomeDamage",
		"FormEligibilityIneligible", "DruidicPresenceRequiresCheck", "DruidicDetection", "WildShapeAssumed", "WildShapeEnded", "WildCompanionCast", "WildCompanionDismissed", "WildShapeUsesChanged",
		"SpellSlotsExchanged", "LandsAidUsed", "NaturalRecoveryUsed", "NaturesSanctuaryCreated", "NaturesSanctuaryMoved", "NaturesSanctuaryEnded", "PrimalStrikeUsed", "DruidLevelGained", "PrimalOrderChosen",
		"ElementalFuryChosen", "LandTypeChosen", "KnownFormsChosen", "KnownFormReplaced", "DruidChoiceReplaceForm", "DruidUseAssumeWildShape", "DruidUseNatureMagician", "DruidTriggerRestFinished",
		"DruidFeatureAttribute", "NewDruidFeatureState", "NewDruidFeatureTables", "DruidTablesFromClass", "BeastFormFactsFromMonster",
	} {
		if !own[want] {
			t.Errorf("the Druid files declare no %s", want)
		}
	}
	count := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			ast.Inspect(f.AST, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					return true
				}
				idx, ok := call.Fun.(*ast.IndexExpr)
				if !ok {
					return true
				}
				if sel, ok := idx.X.(*ast.Ident); !ok || sel.Name != "MustFeatureStateKey" {
					return true
				}
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && strings.Trim(lit.Value, `"`) == "druid" {
					count++
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Errorf("key name druid is used %d times in package character, want 1", count)
	}
}

// Requirement: doc/specs/class-druid.md FR-147, FR-160, FR-164, FR-165 - the Druid files import neither telemetry, context nor sync, read no clock (no call of time.Now, time.Since or time.Until), start no
// goroutine, import no combat package and name no ContentSource, so they are deterministic, side-effect free pure functions of their arguments that use no dice and emit no signal.
// Path: unhappy (architecture).
func TestDruidIsPure(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			if !slices.Contains(druidFiles, strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")) {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if strings.HasSuffix(imp.Path, "/combat") || imp.Path == "context" || imp.Path == "sync" || imp.Path == "math/rand" || imp.Path == "math/rand/v2" || strings.Contains(imp.Path, "telemetry") ||
					strings.HasSuffix(imp.Path, "/dice/randroller") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.Ident:
					if n.Name == "ContentSource" || n.Name == "Roller" {
						t.Errorf("%s names %s", f.Path, n.Name)
					}
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
	if seen != len(druidFiles) {
		t.Errorf("found %d of %d files", seen, len(druidFiles))
	}
}
