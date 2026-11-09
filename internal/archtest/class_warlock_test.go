package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

var warlockDomainFiles = []string{
	"warlock_level", "warlock_reason", "warlock_state", "warlock_event", "warlock_features", "warlock_invocations",
	"warlock_pact_state", "warlock_pact_blade", "warlock_pact_chain", "warlock_pact_tome", "warlock_gaze", "warlock_lifedrinker",
}

var warlockUseCaseFiles = []string{"warlock_usecase", "use_choose_warlock_invocations", "use_dark_ones_own_luck", "use_choose_warlock_mystic_arcanum", "use_cast_invocation_spell",
	"use_choose_warlock_familiar_form", "use_conjure_book_of_shadows", "use_bond_pact_weapon", "use_lifedrinker"}

// Requirement: doc/specs/class-warlock.md FR-147, FR-161, FR-164 - the Warlock files define none of the types other specs own, declare the named results, services and events with no generic variant name, none of their exported names equals an export of
// another package of the module, and the feature state key is named warlock, which no other production file of the package uses as a literal in warlock_state.go.
// Path: unhappy (name collision).
func TestWarlockNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(warlockDomainFiles, warlockUseCaseFiles, []string{"warlock_signals"}) {
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
		t.Fatalf("only %d exported names found in the Warlock files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "RestApplied", "MissingContent", "ProficiencyBonus", "ClassLevels", "SpellSlotPool", "SlotOperationRejection", "SpellcastingState", "FeatTakingRules", "SkillProficiencies", "Subclass", "ClassDataInvalid", "PactMagicSlots", "SpellSaveDc", "CastMode"} {
		if own[owned] {
			t.Errorf("the Warlock files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Granted", "Applied", "Rejected", "Restored", "Changed", "NotAWarlock", "ContentMissing", "InvalidChoices", "Pool", "Grant", "NoUsesLeft", "AlreadyCast", "NotAHit"} {
		if own[generic] {
			t.Errorf("the Warlock files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"WarlockLevel", "WarlockProfile", "WarlockState", "WarlockStateKey", "WarlockStateParts", "WarlockCreationChoices", "WarlockRejection", "WarlockRejectionKind", "WarlockEvent", "WarlockTransition",
		"WarlockChanged", "WarlockRefused", "WarlockSlotChange", "WarlockSlotChangeExpendPact", "WarlockSlotChangeRegainPact", "WarlockSpellGrant", "WarlockTurnFacts", "WarlockAttackKind", "WarlockRestorableFeature",
		"CantripFacts", "CantripEffect", "CantripKind", "InvocationPick", "InvocationProblem", "InvocationProblemKind", "InvocationSelection", "InvocationEffect", "UnmetPrerequisite", "SelectInvocations",
		"InvocationEligibility", "GrantInvocationSpell", "DeriveInvocationEffects", "ChooseInvocations", "CompleteMagicalCunning", "UseContactPatron", "ChooseMysticArcanum", "CastMysticArcanum", "UseEldritchSmite",
		"ResolveDarkOnesBlessing", "ChooseFiendishResilience", "AddDarkOnesOwnLuck", "UseHurlThroughHell", "RestoreHurlThroughHell", "ApplyWarlockRest", "WarlockAlwaysPrepared", "MagicalCunningRegain",
		"EldritchSpearRangeIncrease", "EldritchSmiteDamage", "PactWeaponAttackCount", "LifedrinkerHealing", "ChooseWarlockInvocations", "UseDarkOnesOwnLuck", "ChooseWarlockInvocationsApplied",
		"ChooseWarlockInvocationsRejected", "ChooseWarlockInvocationsContentMissing", "UseDarkOnesOwnLuckApplied", "UseDarkOnesOwnLuckRejected", "InvocationsChosen", "MysticArcanumChosen", "MagicalCunningUsed",
		"ContactPatronUsed", "MysticArcanumCast", "InvocationSpellCastWithoutSlot", "DarkOnesBlessingGained", "FiendishResilienceChosen", "DarkOnesOwnLuckUsed", "HurlThroughHellUsed", "HurlThroughHellRestored",
		"EldritchSmiteUsed", "WarlockFeaturesRestored", "WarlockPactState", "WarlockPactStateKey", "WarlockPactTransition", "WarlockPactChanged", "WarlockPactRefused", "FormPactWeaponBond", "EndPactWeaponBond",
		"PactWeaponAttackOptions", "ChoosePactWeaponDamageType", "SelectFamiliarForm", "ForgoAttackForFamiliar", "CommandFamiliarAttack", "ChooseFamiliarSpeed", "ChangeFamiliarDamageType", "FamiliarSaveDc",
		"GrantFamiliarResistance", "ConjureBook", "ApplyBookOfShadows", "EndBookOfShadows", "WriteGiftOfTheProtectorsName", "EraseGiftOfTheProtectorsName", "ResolveGiftOfTheProtectors",
		"UseGazeOfTwoMinds", "ApplyLifedrinker", "ChooseWarlockMysticArcanum", "CastInvocationSpell", "ChooseWarlockFamiliarForm", "ConjureBookOfShadows", "BondPactWeapon", "UseLifedrinker",
	} {
		if !own[want] {
			t.Errorf("the Warlock files declare no %s", want)
		}
	}
	count := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok && lit.Value == `"warlock"` && strings.HasSuffix(f.Path, "warlock_state.go") {
					count++
				}
				return true
			})
		}
	}
	if count != 1 {
		t.Errorf(`the literal "warlock" is used %d times in warlock_state.go, want 1 (the feature state key)`, count)
	}
}

// Requirement: doc/specs/class-warlock.md FR-160, FR-163, shared-types-and-ownership.md FR-044 - the Warlock domain files import neither telemetry, context nor sync, name no ContentSource, start no goroutine and read no clock, so
// they are pure functions of their arguments; no Warlock file imports combat or a test package; the use case files call no other use case, neither a constructor nor a Run.
// Path: unhappy (architecture).
func TestWarlockDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewRestTaker", "NewUseSecondWind", "NewResolver", "NewChooseWarlockInvocations", "NewUseDarkOnesOwnLuck", "NewChooseWarlockMysticArcanum", "NewCastInvocationSpell", "NewChooseWarlockFamiliarForm", "NewConjureBookOfShadows", "NewBondPactWeapon", "NewUseLifedrinker"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(warlockDomainFiles, name), slices.Contains(warlockUseCaseFiles, name)
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
	if seen != len(warlockDomainFiles)+len(warlockUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(warlockDomainFiles)+len(warlockUseCaseFiles))
	}
}
