package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// statBlockDomainFiles and statBlockUseCaseFiles are the production files of monster-stat-block-actions.md.
func statBlockDomainFiles() []string {
	return []string{"combat/stat_block_event.go", "combat/stat_block_use.go", "combat/stat_block_multiattack.go", "combat/stat_block_attack.go",
		"combat/stat_block_save_effect.go", "combat/stat_block_save_immunities.go"}
}

func statBlockUseCaseFiles() []string { return []string{"combat/roll_stat_block_damage.go"} }

// Requirement: doc/specs/monster-stat-block-actions.md FR-050, observability.md section 3 - the stat_block_*.go files import only content, d20, ability, modifier, srderr and the standard library's maps, math, math/bits, slices, strings and time, never telemetry, context, dice or character, and name no signal constant; the use-case file adds context, fmt, dice and telemetry.
// Path: unhappy (forbidden edge).
func TestStatBlockFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "content", "d20", "modifier", "srderr"}
		std := []string{"maps", "math", "math/bits", "slices", "strings", "time"}
		domain := slices.Contains(statBlockDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(statBlockUseCaseFiles(), f.Path):
			allowed = append(allowed, "dice", "telemetry")
			std = append(std, "context", "fmt")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which monster-stat-block-actions.md does not allow", f.Path, p)
				}
				continue
			}
			if !slices.Contains(std, imp.Path) {
				t.Errorf("%s imports %s, which monster-stat-block-actions.md does not allow", f.Path, imp.Path)
			}
		}
		if !domain {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the domain is silent (FR-050)", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if want := len(statBlockDomainFiles()) + len(statBlockUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d stat-block files", seen, want)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-048 - exactly the six stat-block action events embed the family base, all declared in stat_block_event.go.
// Path: unhappy (duplicate).
func TestStatBlockActionEventFamilyHasExactlySixMembers(t *testing.T) {
	t.Parallel()
	var members []string
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, fld := range st.Fields.List {
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "statBlockActionEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/stat_block_event.go" {
							t.Errorf("%s is declared in %s, want combat/stat_block_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"MultiattackAdditionalUseRecorded", "MultiattackAttackRecorded", "MultiattackAttackReplaced", "SaveImmunityExpired", "SaveImmunityGranted", "StatBlockPartUsed"}
	if !slices.Equal(members, want) {
		t.Errorf("stat-block action event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md section 5 - the closed sets have exactly the variants the spec lists (RollStatBlockDamageResult adds StatBlockDamageInvalid for plan P13).
// Path: unhappy (duplicate).
func TestStatBlockClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isStatBlockUseRefusal": {"StatBlockLegendaryUseRefused", "StatBlockPartInWrongSection", "StatBlockPartNotFound", "StatBlockPartUsageExhausted", "StatBlockReactionTriggerNotReported"},
		"isStatBlockUseOutcome": {"StatBlockUseAllowed", "StatBlockUseRefused"}, "isMultiattackStartOutcome": {"MultiattackNotAvailable", "MultiattackStarted"},
		"isMultiattackStepOutcome":    {"MultiattackNotInAllowance", "MultiattackStepRecorded", "MultiattackStepUsageRefused"},
		"isStatBlockSaveTargetResult": {"StatBlockSaveEffectApplies", "StatBlockSaveTargetImmune", "StatBlockSaveTargetRefused"},
		"isRollStatBlockDamageResult": {"StatBlockDamageInvalid", "StatBlockDamageRolled"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-044 - the use case constructor takes a Roller and options only.
// Path: unhappy (forbidden edge).
func TestRollStatBlockDamageConstructorTakesOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "NewRollStatBlockDamage" {
				continue
			}
			found++
			var params []string
			for _, p := range fn.Type.Params.List {
				switch x := p.Type.(type) {
				case *ast.Ident:
					params = append(params, x.Name)
				case *ast.Ellipsis:
					params = append(params, "..."+x.Elt.(*ast.Ident).Name)
				}
			}
			if got := strings.Join(params, ","); got != "Roller,...Option" {
				t.Errorf("NewRollStatBlockDamage takes (%s), want (Roller,...Option)", got)
			}
		}
	}
	if found != 1 {
		t.Errorf("found %d NewRollStatBlockDamage, want 1", found)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md section 7, plan-phase-4.md P11 - signals.go pins the operation value and span, the notation attribute and its three values, and the two result values at their catalogue values, and signals.md has a row for the span and the attribute.
// Path: unhappy (invalid input).
func TestStatBlockSignalNamesAndCatalogue(t *testing.T) {
	t.Parallel()
	got := map[string]string{}
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/signals.go" {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if v, ok := n.(*ast.ValueSpec); ok && len(v.Values) == 1 {
				if lit, ok := v.Values[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					got[v.Names[0].Name], _ = strconv.Unquote(lit.Value)
				}
			}
			return true
		})
	}
	want := map[string]string{
		"valueOperationRollStatBlockDamage": "roll_stat_block_damage", "spanRollStatBlockDamage": "srd5e.combat.roll_stat_block_damage",
		"attrDamageNotation": "srd5e.combat.damage_notation", "valueDamageNotationNumber": "number", "valueDamageNotationDice": "dice", "valueDamageNotationMixed": "mixed",
		"valueResultStatBlockDamageRolled": "stat_block_damage_rolled", "valueResultStatBlockDamageFixed": "stat_block_damage_fixed",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	doc := catalogue(t)
	for _, name := range []string{"srd5e.combat.roll_stat_block_damage", "srd5e.combat.damage_notation"} {
		if !strings.Contains(doc, "| `"+name+"` |") {
			t.Errorf("signals.md has no row for %s", name)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-051 - package combat exports the stat-block action types, rules and use case once each and no generic variant name.
// Path: happy.
func TestCombatExportsStatBlockActionTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		StatBlockActionCost StatBlockCostPassive StatBlockCostAction StatBlockCostBonusAction StatBlockCostReaction StatBlockCostLegendaryAction StatBlockActionCostOf
		ReactionTriggerReport ReactionTriggerOccurred ReactionTriggerNotOccurred StatBlockUseRequest StatBlockUseRefusal StatBlockPartNotFound StatBlockPartInWrongSection
		StatBlockReactionTriggerNotReported StatBlockLegendaryUseRefused StatBlockPartUsageExhausted StatBlockUseTransition StatBlockUseOutcome StatBlockUseAllowed
		StatBlockUseRefused UseStatBlock StatBlockFeatureIDReason FeatureIDInvalidName FeatureIDTooLong FeatureIDDuplicate StatBlockFeatureIDRejected StatBlockFeatureID
		StatBlockFeatureIDProblems StatBlockActionEvent StatBlockPartUsed MultiattackAttackRecorded MultiattackAdditionalUseRecorded MultiattackAttackReplaced
		SaveImmunityGranted SaveImmunityExpired MultiattackAllowanceRefusal MultiattackAttackNotAllowed MultiattackAllowanceSpent MultiattackAdditionalUseNotListed
		MultiattackAdditionalUseAlreadyMade MultiattackReplacementNotListed MultiattackReplacementLimitReached MultiattackNotEnoughAttacksToReplace
		MultiattackPlanProblemKind MultiattackPlanPartMissing MultiattackPlanPartShared MultiattackPlanSpellMissing MultiattackPlanProblem MultiattackPlanProblemsFor
		MultiattackOption WithGrantedAction WithUsedThisTurn MultiattackProgress MultiattackTransition MultiattackStartOutcome MultiattackStarted MultiattackNotAvailable
		MultiattackStepOutcome MultiattackStepRecorded MultiattackNotInAllowance MultiattackStepUsageRefused StartMultiattack DamageNotationChoice DamageNotationNumber
		DamageNotationDice DamageNotationChoices NewDamageNotationChoices AllDamageNotation StatBlockAttackPlan AttackPlanOption WithDelivery PlanStatBlockAttack
		AppliedClausesForAttack StatBlockSaveEffectPlan PlanStatBlockSaveEffect SaveFailureStep SaveFirstFailure SaveSecondFailure SaveSubsequentFailures
		SaveEffectDamageAdjustment SaveEffectDamageUnchanged SaveEffectDamageHalved StatBlockSaveRefusal StatBlockSaveMissing StatBlockSaveNotThisSave
		StatBlockSaveTargetResult StatBlockSaveEffectApplies StatBlockSaveTargetImmune StatBlockSaveTargetRefused StatBlockSaveEffectInput ResolveStatBlockSaveEffect
		SaveImmunityWindow NewSaveImmunityWindow SaveImmunities SaveImmunityTransition RollStatBlockDamage NewRollStatBlockDamage RollStatBlockDamageCommand
		RollStatBlockDamageResult StatBlockDamageRolled StatBlockDamageInvalid`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Window", "Progress", "Plan", "Immunities", "Rolled", "Invalid", "Applies", "Recorded", "Allowed"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-051)", generic)
		}
	}
}
