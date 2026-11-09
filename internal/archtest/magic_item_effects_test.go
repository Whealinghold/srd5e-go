package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// magicItemEffectDomainFiles are the files of package character that hold the magic item effect rules; none imports telemetry, context or sync.
// The use cases of that spec are in magicItemEffectUseCaseFiles, which hold every telemetry call.
var magicItemEffectDomainFiles = []string{"enhancement_bonus", "magic_item_effect", "magic_item_effect_table", "effective_ability_scores", "potion_of_healing", "charged_spell",
	"last_charge", "staff_absorption", "spell_storage", "extradimensional_space", "charged_item_event"}

var magicItemEffectUseCaseFiles = []string{"potion_of_healing_usecase", "last_charge_usecase"}

// magicItemEffectRequiredNames are the names FR-069 requires package character to export.
var magicItemEffectRequiredNames = []string{"RollPotionOfHealing", "ResolveLastCharge", "RollSpellStoringRingLevels", "RollPotionOfHealingCommand", "ResolveLastChargeCommand",
	"RollSpellStoringRingLevelsCommand", "RollPotionOfHealingResult", "RollPotionOfHealingRolled", "RollPotionOfHealingNotListed", "RollPotionOfHealingStandardRolledElsewhere",
	"ResolveLastChargeResult", "ResolveLastChargeResolved", "ResolveLastChargeRejected", "RollSpellStoringRingLevelsResult", "RollSpellStoringRingLevelsRolled",
	"MagicItemEffect", "ItemEffectCondition", "ItemEffectContext", "MagicItemBonus", "ActiveItemEffects", "EffectiveAbilityScores", "AbilityScoreSource", "GiantStrengthType",
	"EnhancementFamily", "EnhancementBonusResult", "EnhancementBonusFound", "EnhancementBonusNotListed", "MagicAmmunition", "AmmunitionState", "MagicItemEffectTable",
	"MagicItemEffectLookup", "MagicItemEffectsFound", "MagicItemEffectsNotModelled", "PotionOfHealingGrade", "PotionOfHealingGradeFound", "PotionOfHealingGradeNotListed",
	"ResistanceTypeTable", "GrantedResistance", "GrantedResistanceRejected", "ChargedSpellOption", "FixedChargeCost", "ChargesPerLevel", "ChargedSpellPlan", "ChargedSpellPlanRejected",
	"ChargedSpellCasting", "ChargedSpellCastApplied", "LastChargeRule", "NoLastChargeEffect", "LastChargeAutomatic", "LastChargeRolled", "LastChargeConsequence", "LastChargeRegain",
	"StaffSpellAbsorption", "StaffSpellAbsorptionAbsorbed", "StaffSpellAbsorptionOverflowed", "RetributiveStrike", "RetributiveStrikeDamage", "SpellStorage", "StoredSpell",
	"SpellStorageStored", "SpellStorageSpellExpended", "SpellStorageNotStorable", "StoredSpellCast", "SpellStorageNotUsable", "SpellStorageNoSuchSpell", "SpellStorageRejected",
	"ExtradimensionalSpace", "BagOfHoldingSpace", "HandyHaversackSpace", "PortableHoleSpace", "SimilarSpace", "SpaceLoad", "AstralGate", "GateArea", "SpaceCheck", "SpaceWithinCapacity",
	"SpaceOverloaded", "ExtradimensionalNesting", "NestingSafe", "NestingDestroysBoth", "AirSupplies", "AirForMinutes", "NoOneBreathing", "ContainerWeight", "FixedWeight",
	"WeighsNextToNothing", "ChargedItemEvent", "MagicItemLastChargeResolved", "RingSpellStored", "RingStoredSpellCast", "StaffSpellAbsorbed"}

// magicItemEffectForbiddenNames are the generic names FR-069 forbids.
var magicItemEffectForbiddenNames = []string{"Applied", "Rejected", "Found", "NotListed", "NotFound", "Rolled", "Resolved", "Overloaded", "Destroyed", "Intact"}

// Requirement: doc/specs/magic-item-effects.md FR-066, FR-067, FR-070 - the effect domain files import neither telemetry nor context nor sync nor
// math/rand and start no goroutine; the use case files call no other use case, and no file of package character imports combat.
// Path: unhappy (architecture).
func TestMagicItemEffectsDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewRollPotionOfHealing", "NewResolveLastCharge", "NewRollSpellStoringRingLevels", "NewRechargeMagicItem", "NewRollPotionMiscibility", "NewCastSpell"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			for _, imp := range f.Imports() {
				if strings.HasSuffix(imp.Path, "/combat") {
					t.Errorf("%s imports combat", f.Path)
				}
			}
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(magicItemEffectDomainFiles, name), slices.Contains(magicItemEffectUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "sync" || strings.HasPrefix(imp.Path, "math/rand") || strings.Contains(imp.Path, "telemetry")) {
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
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Do" {
						t.Errorf("%s calls a Do method", f.Path)
					}
				}
				return true
			})
		}
	}
	if want := len(magicItemEffectDomainFiles) + len(magicItemEffectUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-068, FR-069 - package character exports every name the effect types, use cases and events need, none of
// the generic names, and none of the effect files exports a name another package of the module also exports.
// Path: unhappy (name collision).
func TestMagicItemEffectsExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(magicItemEffectDomainFiles, magicItemEffectUseCaseFiles) {
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
	for _, n := range magicItemEffectRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s (FR-069)", n)
		}
	}
	for _, n := range magicItemEffectForbiddenNames {
		if own[n] {
			t.Errorf("character exports the generic name %s (FR-069)", n)
		}
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
