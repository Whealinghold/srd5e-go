package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// wishDomainFiles are the files of package character that hold the Wish rules: pure domain code. wish_usecase.go and
// wish_signals.go are the use case and its signals.
var wishDomainFiles = []string{"wish_state", "wish_choice", "wish_rules", "wish_event"}

// wishRequiredNames are the types and functions the spec requires package character to export (FR-067).
var wishRequiredNames = []string{"WishResolver", "NewWishResolver", "ResolveWishCommand", "ResolveWishResult", "ResolveWishDuplicated", "ResolveWishApplied", "ResolveWishRejected",
	"RollWishStressDamageCommand", "RollWishStressDamageResult", "RollWishStressDamageRolled", "RollWishStressDamageNone", "WishChoice", "DuplicateSpell", "ObjectCreation",
	"InstantHealth", "ResistanceGrant", "SpellImmunityGrant", "SuddenLearning", "RollRedo", "ReshapeReality", "WishStatement", "NewWishStatement", "SpellImmunitySubject",
	"ImmuneToSpell", "ImmuneToOtherEffect", "WishEffect", "WishDuplicate", "WishObject", "WishInstantHealth", "WishResistance", "WishSpellImmunity", "WishSuddenLearning",
	"WishRollRedo", "WishReshapeReality", "WishStress", "WishWeakness", "NoWishWeakness", "WishWeakened", "WishCastDamage", "WishCastDamageInactive", "WishCastDamageActive",
	"WishCastingAbility", "CanCastWish", "CannotCastWishAgain", "WishStressKey", "WishStressRoll", "NewWishStressRoll", "WishRecoveryDay", "UnpreventableDamage",
	"ResolveWishRejection", "ResolveWishRejectionKind", "WishEvent", "WishDuplicated", "WishStressEndured", "WishWeaknessEnded", "WishCastDamageEnded", "WishStressDamageRolled",
	"WishTransition", "CheckWish", "ApplyWish", "WishRequiresStress", "IncurWishStress", "WishEffectiveScores", "WishAfterDay", "WishAfterRest", "WishLevelOfCast", "WishDamageDice",
	"WishCastOrigin", "WishSpellID", "WishLevel"}

// Requirement: doc/specs/wish-spell.md FR-067, section 8 "architecture" - package character exports every type the spec names, no Wish file
// uses a generic variant name, and none of their exports equals an export of another package of the module.
// Path: unhappy (name collision).
func TestWishNamesAreExportedAndUnique(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range append([]string{"wish_usecase", "wish_signals"}, wishDomainFiles...) {
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
	for _, n := range wishRequiredNames {
		if !own[n] {
			t.Errorf("character does not export %s from the Wish files", n)
		}
	}
	for n := range own {
		if slices.Contains([]string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"}, n) {
			t.Errorf("character.%s is a generic variant name", n)
		}
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
}

// Requirement: doc/specs/wish-spell.md FR-066, section 7 and architecture.md section 3 - the Wish domain files are pure: they import neither
// telemetry nor context nor sync nor time nor combat, start no goroutine and read no clock; only wish_usecase.go and wish_signals.go import
// telemetry.
// Path: unhappy (architecture).
func TestWishDomainIsPure(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			if !slices.Contains(wishDomainFiles, name) {
				continue
			}
			seen++
			for _, imp := range f.Imports() {
				if imp.Path == "context" || imp.Path == "time" || imp.Path == "sync" || strings.Contains(imp.Path, "telemetry") || strings.HasSuffix(imp.Path, "/combat") {
					t.Errorf("%s imports %s", f.Path, imp.Path)
				}
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				if g, ok := n.(*ast.GoStmt); ok {
					t.Errorf("%s: go statement at %v", f.Path, f.Fset.Position(g.Pos()))
				}
				return true
			})
		}
	}
	if seen != len(wishDomainFiles) {
		t.Errorf("found %d of %d files", seen, len(wishDomainFiles))
	}
}
