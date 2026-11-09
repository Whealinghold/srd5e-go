package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// magicItemDomainFiles are the files of package character that hold the magic item
// rules; none imports telemetry, context, sync or time.
var magicItemDomainFiles = []string{"magic_item_instance", "magic_item_event", "magic_item_values", "magic_item_facts_rules", "magic_item_activation", "item_charges",
	"short_rest_focus", "attunement_set", "attunement_rules", "attunement_operations", "magic_item_identification", "worn_items", "magic_item_crafting",
	"sentient_traits", "sentient_conflict", "sentient_save_roll"}

var magicItemUseCaseFiles = []string{"magic_item_attunement_usecase", "magic_item_roll_usecase", "plan_magic_item_crafting_usecase", "generate_sentient_traits_usecase", "sentient_usecase"}

// magicItemRequiredNames are the names FR-119 requires package character to export.
var magicItemRequiredNames = []string{"AttuneToMagicItem", "EndMagicItemAttunement", "RechargeMagicItem", "RollPotionMiscibility", "CheckCraftingMaterials", "PlanMagicItemCrafting",
	"GenerateSentientItemTraits", "ResolveSentientItemConflict", "AttemptSentientItemControl", "RepeatSentientControlSave",
	"AttuneToMagicItemResult", "AttuneToMagicItemApplied", "AttuneToMagicItemRejected", "EndMagicItemAttunementResult", "EndMagicItemAttunementApplied", "EndMagicItemAttunementRejected",
	"RechargeMagicItemResult", "RechargeMagicItemApplied", "RechargeMagicItemNotDue", "RollPotionMiscibilityResult", "RollPotionMiscibilityRolled", "RollPotionMiscibilityRejected",
	"CheckCraftingMaterialsResult", "CheckCraftingMaterialsAvailable", "CheckCraftingMaterialsUnavailable", "PlanMagicItemCraftingResult", "PlanMagicItemCraftingPlanned",
	"PlanMagicItemCraftingRejected", "PlanMagicItemCraftingRarityUnresolved", "PlanMagicItemCraftingContentMissing", "PlanMagicItemCraftingContentIncomplete",
	"GenerateSentientItemTraitsResult", "GenerateSentientItemTraitsGenerated", "GenerateSentientItemTraitsRejected", "ResolveSentientItemConflictResult",
	"ResolveSentientItemConflictResisted", "ResolveSentientItemConflictDemandsMade", "ResolveSentientItemConflictRejected", "AttemptSentientItemControlResult",
	"AttemptSentientItemControlTaken", "AttemptSentientItemControlResisted", "AttemptSentientItemControlRejected", "RepeatSentientControlSaveResult",
	"RepeatSentientControlSaveEnded", "RepeatSentientControlSaveContinues",
	"MagicItemEvent", "MagicItemAttuned", "MagicItemAttunementEnded", "MagicItemConsumed", "MagicItemChargesSpent", "MagicItemRecharged", "SentientControlTaken", "SentientControlEnded",
	"WornItemsSlotOccupied", "WornItemsDoesNotFit", "ItemSpellcastingAbilityNotAvailable", "MagicItemInsufficientCharges", "MagicItemAlreadyConsumed", "MagicItemCommandWordAccepted",
	"MagicItemCommandWordFailed", "MagicItemScrollNotReadable", "MagicItemBaseItemNotAllowed", "MagicItemPropertiesLearned", "MagicItemPropertiesNotLearned", "MagicItemCraftingShortfall",
	"ShortRestFocusRejected", "ItemChargesRejected", "RarityNotResolved", "AlignedPurposeRejected", "SentientTraitsRejected", "SentientDemandsRejected", "AttunableItemNotAttunable",
	"SentientSave", "SentientSaveRolled", "SentientSaveForced", "MagicItemCrafting", "MagicItemCraftingOutcome", "MagicItemCraftingRequirements", "MagicItemUsesEquipmentRules", "MagicItemNotCraftable"}

// magicItemForbiddenNames are the generic names FR-119 forbids.
var magicItemForbiddenNames = []string{"Applied", "Rejected", "SlotOccupied", "DoesNotFit", "AbilityNotAvailable", "InsufficientCharges", "ChargesSpent", "ScrollNotReadable",
	"BaseItemNotAllowed", "CommandWordAccepted", "CommandWordFailed", "PropertiesLearned", "CraftingShortfall"}

// Requirement: doc/specs/magic-item-rules.md FR-105, FR-123 - the magic item domain files import neither telemetry nor context nor sync nor time and
// start no goroutine; the use case files call no other use case; the unexported helper sentientSaveRoll declares no exported identifier.
// Path: unhappy (architecture).
func TestMagicItemDomainIsSilentAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewAttuneToMagicItem", "NewEndMagicItemAttunement", "NewRechargeMagicItem", "NewRollPotionMiscibility", "NewCheckCraftingMaterials",
		"NewPlanMagicItemCrafting", "NewGenerateSentientItemTraits", "NewResolveSentientItemConflict", "NewAttemptSentientItemControl", "NewRepeatSentientControlSave",
		"NewToppleSaveResolver", "NewCalculateSpellcasting", "NewCreateSpellcasting", "NewCalculateDerivedStats", "NewRestTaker"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(magicItemDomainFiles, name), slices.Contains(magicItemUseCaseFiles, name)
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
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Do" {
						t.Errorf("%s calls a Do method", f.Path)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok && useCase && sel.Sel.Name == "Resolve" {
						t.Errorf("%s calls a Resolve method (the d20 use case)", f.Path)
					}
				}
				return true
			})
			if name == "sentient_save_roll" {
				for n := range exportedNames(pkg, map[string]bool{f.Path: true}) {
					t.Errorf("sentient_save_roll.go exports %s", n)
				}
			}
		}
	}
	if want := len(magicItemDomainFiles) + len(magicItemUseCaseFiles); seen != want {
		t.Errorf("found %d of %d files", seen, want)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-119 - package character exports every name the magic item use cases and results need, none
// of the generic names, and none of the magic item files exports a name another package of the module also exports.
// Path: unhappy (name collision).
func TestMagicItemExportsAreOwnedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(magicItemDomainFiles, magicItemUseCaseFiles) {
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
	for _, n := range magicItemRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s (FR-119)", n)
		}
	}
	for _, n := range magicItemForbiddenNames {
		if own[n] {
			t.Errorf("character exports the generic name %s (FR-119)", n)
		}
	}
	if own["sentientSaveRoll"] {
		t.Error("sentientSaveRoll is exported")
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}
