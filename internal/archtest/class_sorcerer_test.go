package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

var sorcererDomainFiles = []string{
	"sorcerer_level", "sorcerer_reason", "sorcerer_state", "sorcerer_event", "sorcerer_features", "sorcerer_metamagic",
}

var sorcererUseCaseFiles = []string{"sorcerer_usecase", "use_choose_metamagic_options", "use_apply_sorcerer_level_up", "use_create_spell_slot", "use_metamagic"}

// Requirement: doc/specs/class-sorcerer.md FR-130, FR-131 - the Sorcerer files define none of the types other specs own, declare the named results, services and events with no generic variant name, none of their exported names equals an export of
// another package of the module, and the feature state key is named sorcerer, which no other production file of the package uses as a literal.
// Path: unhappy (name collision).
func TestSorcererNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(sorcererDomainFiles, sorcererUseCaseFiles, []string{"sorcerer_signals"}) {
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
	if len(own) < 80 {
		t.Fatalf("only %d exported names found in the Sorcerer files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "RestApplied", "MissingContent", "ProficiencyBonus", "ClassLevels", "SpellSlotPool", "SlotOperationRejection", "SpellcastingState", "FeatTakingRules", "SkillProficiencies", "Subclass", "ClassDataInvalid"} {
		if own[owned] {
			t.Errorf("the Sorcerer files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Granted", "Applied", "Rejected", "Restored", "Changed", "NotASorcerer", "ContentMissing", "InvalidChoices", "Pool", "Grant", "NoSlotToConvert", "CountOutOfRange", "SpellFactUnknown", "CastingTimeNotAction"} {
		if own[generic] {
			t.Errorf("the Sorcerer files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"SorcererLevel", "SorcererProfile", "SorcererState", "SorcererStateKey", "SorcererStateParts", "SorcererCreationChoices", "SorcererRejection", "SorcererRejectionKind", "MetamagicIneligibility", "MetamagicEffect",
		"SorcererEvent", "SorcererTransition", "SorcererChanged", "SorcererRefused", "SorcererSlotTransition", "SorcererSlotsChanged", "SorcererSlotsRefused", "MetamagicSpellFacts", "MetamagicCastContext",
		"MetamagicTransition", "MetamagicApplied", "MetamagicRefused", "InnateSorceryEffects", "SorcererActiveEffects", "SorcererLevelGrants", "ActivateInnateSorcery", "EndInnateSorcery", "StartSorcererTurn",
		"ConvertSpellSlot", "CreateSlot", "SpendCreatedSlot", "ApplySorcererRest", "ChooseMetamagic", "ApplyMetamagic", "CheckQuickenedRestriction", "ChooseElementalAffinity", "ApplyElementalAffinity",
		"UseDragonWings", "UseDragonCompanion", "DraconicResilienceBonus", "DraconicArmorClassBasis", "SorcererLevelGrantsFor", "SubclassFeaturesAvailable", "SorcererFeaturesAt",
		"ChooseMetamagicOptions", "ApplySorcererLevelUp", "CreateSpellSlot", "UseMetamagic", "ChooseMetamagicOptionsApplied", "ChooseMetamagicOptionsRejected", "ChooseMetamagicOptionsContentMissing",
		"ApplySorcererLevelUpApplied", "ApplySorcererLevelUpRejected", "ApplySorcererLevelUpContentMissing", "CreateSpellSlotApplied", "CreateSpellSlotRejected", "CreateSpellSlotContentMissing",
		"UseMetamagicApplied", "UseMetamagicRejected", "UseMetamagicContentMissing", "InnateSorceryActivated", "InnateSorceryEnded", "SpellSlotConvertedToSorceryPoints", "SpellSlotCreated",
		"CreatedSlotSpent", "CreatedSlotsExpired", "MetamagicUsed", "SorceryPointsRestored", "MetamagicOptionsChosen", "ElementalAffinityChosen", "ElementalAffinityApplied", "DragonWingsActivated",
		"DragonWingsEnded", "DragonWingsUseRestored", "DragonCompanionCast",
	} {
		if !own[want] {
			t.Errorf("the Sorcerer files declare no %s", want)
		}
	}
	count := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Value == `"sorcerer"` && strings.HasSuffix(f.Path, "sorcerer_state.go") {
					count++
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Errorf(`the literal "sorcerer" is used %d times in sorcerer_state.go, want 1 (the feature state key)`, count)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-100, FR-101, FR-132, shared-types-and-ownership.md FR-044 - the Sorcerer domain files import neither telemetry, context nor sync, name no ContentSource, start no goroutine and read no clock, so
// they are pure functions of their arguments; no Sorcerer file imports combat or a test package; the use case files call no other use case, neither a constructor nor a Run.
// Path: unhappy (architecture).
func TestSorcererDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewRestTaker", "NewUseSecondWind", "NewResolver", "NewChooseMetamagicOptions", "NewApplySorcererLevelUp", "NewCreateSpellSlot", "NewUseMetamagic"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(sorcererDomainFiles, name), slices.Contains(sorcererUseCaseFiles, name)
			if !domain && !useCase {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if domain && (imp.Path == "context" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry")) {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
				if strings.Contains(imp.Path, "combat") || strings.HasSuffix(imp.Path, "test") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.GoStmt:
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(n.Pos()))
				case *ast.Ident:
					if domain && n.Name == "ContentSource" {
						t.Errorf("%s names ContentSource", f.Path)
					}
				case *ast.CallExpr:
					if id, ok := n.Fun.(*ast.Ident); ok && useCase && slices.Contains(constructors, id.Name) {
						t.Errorf("%s calls %s", f.Path, id.Name)
					}
					if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
						if pkgID, isPkg := sel.X.(*ast.Ident); isPkg && pkgID.Name == "time" {
							t.Errorf("%s uses time.%s", f.Path, sel.Sel.Name)
						}
						if useCase && sel.Sel.Name == "Run" {
							t.Errorf("%s calls a Run method", f.Path)
						}
					}
				}
				return true
			})
		}
	}
	if seen != len(sorcererDomainFiles)+len(sorcererUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(sorcererDomainFiles)+len(sorcererUseCaseFiles))
	}
}
