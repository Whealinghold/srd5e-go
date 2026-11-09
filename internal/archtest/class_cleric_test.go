package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

var clericDomainFiles = []string{
	"cleric_level", "cleric_reason", "cleric_progression", "cleric_state", "cleric_rule_data", "cleric_event", "cleric_divine_order", "cleric_creation_choices",
	"cleric_divine_spark", "cleric_turn_undead", "cleric_preserve_life", "cleric_blessed_strikes", "cleric_divine_intervention", "cleric_life_domain", "cleric_advance", "cleric_usecase",
}

var clericUseCaseFiles = []string{"use_cleric_channel_divinity", "use_divine_intervention"}

// Requirement: doc/specs/class-cleric.md FR-136, FR-139 - the Cleric files define none of the types other specs own, declare the named results, services and events with no generic variant name, none of their
// exported names equals an export of another package of the module, and the two feature state keys are named cleric_choices and cleric_resources, which no other production file uses.
// Path: unhappy (name collision).
func TestClericNamesAreUniqueAndSpecific(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	files := map[string]bool{}
	for _, f := range slices.Concat(clericDomainFiles, clericUseCaseFiles, []string{"cleric_signals"}) {
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
		t.Fatalf("only %d exported names found in the Cleric files", len(own))
	}
	for n := range own {
		if dirs := others[n]; len(dirs) > 0 {
			t.Errorf("character.%s duplicates an export of %v", n, dirs)
		}
	}
	for _, owned := range []string{"SubclassChosen", "RestApplied", "MissingContent", "ProficiencyBonus", "ClassLevels", "SpellSaveDC", "HitPointMaximum", "IsBloodied"} {
		if own[owned] {
			t.Errorf("the Cleric files define %s, which another spec owns", owned)
		}
	}
	for _, generic := range []string{"Available", "Unavailable", "Recharging", "Protector", "Thaumaturge", "DivineStrike", "PotentSpellcasting", "Applied", "Rejected", "NotACleric", "NotApplicable", "ContentMissing"} {
		if own[generic] {
			t.Errorf("the Cleric files declare the generic name %s", generic)
		}
	}
	for _, want := range []string{
		"ClericLevel", "ClericProgression", "ClericChoices", "ClericResources", "ClericCreationChoices", "ClericRuleData", "ExtraDivineInterventionGrant", "ClericEvent", "ClericAdvanced",
		"DivineOrderProtector", "DivineOrderThaumaturge", "BlessedStrikesDivineStrike", "BlessedStrikesPotentSpellcasting", "DivineInterventionAvailable", "DivineInterventionUnavailableUntilLongRest",
		"DivineInterventionRecharging", "DivineSpark", "TurnUndead", "SearUndead", "PreserveLife", "SupremeHealing", "DivineIntervention", "LifeDomainSpells", "ChooseDivineOrder", "ChooseBlessedStrikes",
		"AdvanceCleric", "ResolveBlessedStrikes", "ResolveClericHealing", "ApplyClericRest", "UseClericChannelDivinity", "UseDivineIntervention", "UseClericChannelDivinityApplied",
		"UseClericChannelDivinityRejected", "UseClericChannelDivinityNotACleric", "UseDivineInterventionApplied", "UseDivineInterventionRejected", "UseDivineInterventionNotACleric",
		"ChooseDivineOrderNotACleric", "AdvanceClericNotACleric", "ResolveBlessedStrikesNotApplicable", "ThaumaturgeCheckBonus",
	} {
		if !own[want] {
			t.Errorf("the Cleric files declare no %s", want)
		}
	}
	// FR-139: the key names are literals in the production files of package character, one file each.
	count := map[string]int{}
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
				if lit, ok := call.Args[0].(*ast.BasicLit); ok {
					count[strings.Trim(lit.Value, `"`)]++
				}
				return true
			})
		}
	}
	for _, k := range []string{"cleric_choices", "cleric_resources"} {
		if count[k] != 1 {
			t.Errorf("key name %q is used %d times in package character, want 1", k, count[k])
		}
	}
}

// Requirement: doc/specs/class-cleric.md FR-123, FR-126, shared-types-and-ownership.md FR-044 - the Cleric domain files import neither telemetry, context, time nor sync, start no goroutine and read no
// ContentSource, so they are pure functions of their arguments; the two use case files call no other use case (neither a constructor nor a Run).
// Path: unhappy (architecture).
func TestClericDomainIsPureAndUseCasesDoNotNest(t *testing.T) {
	t.Parallel()
	m := loadModule(t)
	constructors := []string{"NewLevelUpCharacter", "NewAwardExperience", "NewChangeAbilityScore", "NewTakeFeat", "NewRestTaker", "NewUseSecondWind", "NewUseClericChannelDivinity", "NewUseDivineIntervention"}
	seen := 0
	for _, pkg := range m.Packages {
		if pkg.Dir != "character" {
			continue
		}
		for _, f := range pkg.Production() {
			name := strings.TrimSuffix(strings.TrimPrefix(f.Path, "character/"), ".go")
			domain, useCase := slices.Contains(clericDomainFiles, name), slices.Contains(clericUseCaseFiles, name)
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
				case *ast.Ident:
					if n.Name == "ContentSource" {
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
	if seen != len(clericDomainFiles)+len(clericUseCaseFiles) {
		t.Errorf("found %d of %d files", seen, len(clericDomainFiles)+len(clericUseCaseFiles))
	}
}
