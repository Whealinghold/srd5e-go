package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"testing"
)

// attackHelpFiles are the production files of attack-and-help-actions.md: pure domain files, no use case.
func attackHelpFiles() []string {
	return []string{"combat/attack_means.go", "combat/attack_allowance.go", "combat/attack_action_progress.go",
		"combat/attack_action_outcome.go", "combat/attack_action_weapon_change.go", "combat/attack_action_event.go",
		"combat/help_proficiency.go", "combat/help_benefit.go", "combat/help_benefits.go", "combat/help_event.go"}
}

// attackHelpTypes returns the exported and unexported type names declared by the attack and Help files.
func attackHelpTypes(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(attackHelpFiles(), f.Path) {
			continue
		}
		seen++
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				out[ts.Name.Name] = true
			}
			return true
		})
	}
	if seen != len(attackHelpFiles()) {
		t.Fatalf("found %d of %d attack and Help files", seen, len(attackHelpFiles()))
	}
	return out
}

// Requirement: doc/specs/attack-and-help-actions.md FR-057, FR-058, FR-061, FR-067, section 7 - the attack and Help files are silent
// domain code: they import only content and modifier of this module (no telemetry, dice, srderr, character, d20) and no sync,
// time or math/rand; they declare no command, use case, port or Run method and name no signal constant or weapon-properties type.
// Path: unhappy (forbidden edge).
func TestAttackHelpFilesAreSilentDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	attackHelpTypes(t)
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(attackHelpFiles(), f.Path) {
			continue
		}
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains([]string{"content", "modifier"}, p) {
				t.Errorf("%s imports %s, which attack-and-help-actions.md FR-057 does not allow", f.Path, p)
			}
			if !strings.Contains(imp.Path, ".") && slices.Contains([]string{"sync", "time", "math/rand", "context"}, imp.Path) {
				t.Errorf("%s imports %s (FR-057, FR-061)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				for _, banned := range []string{"WeaponTurnState", "WeaponWielding", "TurnTracker", "InteractWithObject"} {
					if n.Name == banned {
						t.Errorf("%s names %s (FR-031, FR-067)", f.Path, n.Name)
					}
				}
				if strings.HasPrefix(n.Name, "metric") || strings.HasPrefix(n.Name, "span") || strings.HasPrefix(n.Name, "attr") {
					t.Errorf("%s names the signal constant %s: the spec is silent by design (section 7)", f.Path, n.Name)
				}
			case *ast.TypeSpec:
				for _, suffix := range []string{"Command", "UseCase", "Roller", "Port"} {
					if n.Name.IsExported() && strings.HasSuffix(n.Name.Name, suffix) {
						t.Errorf("%s declares %s: this spec defines no command, use case or port (FR-058)", f.Path, n.Name.Name)
					}
				}
			case *ast.FuncDecl:
				if n.Recv != nil && n.Name.Name == "Run" {
					t.Errorf("%s declares a Run method: no use case here (FR-058)", f.Path)
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-057 - no file of package combat outside the attack and Help files mentions
// their types (the files of combat-action-resolution.md, listed by actionResolutionDomainFiles and actionResolutionUseCaseFiles, and combat/item_use_thrown.go, whose ThrownItemRules.Replacement returns an AttackReplacement by item-use-effects.md FR-004, and combat/roster_usecase.go, whose RemoveCombatantFromEncounterApplied returns the HelpBenefitDropped events of HelpBenefits.DropFor by combatant-roster-changes.md FR-025, are the only consumers).
// Path: unhappy (architecture).
func TestOnlyTheAttackHelpFilesMentionTheirTypes(t *testing.T) {
	t.Parallel()
	mine := attackHelpTypes(t)
	for _, f := range combatPackage(t).Production() {
		if slices.Contains(attackHelpFiles(), f.Path) || slices.Contains(actionResolutionDomainFiles(), f.Path) || slices.Contains(actionResolutionUseCaseFiles(), f.Path) || f.Path == "combat/item_use_thrown.go" || f.Path == "combat/roster_usecase.go" {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && mine[id.Name] && id.IsExported() {
				t.Errorf("%s mentions %s of attack-and-help-actions.md (FR-057)", f.Path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-024, FR-031 - no public member of AttackActionProgress is named for an
// Opportunity Attack or a Reaction, and none takes or returns a type of initiative-and-actions.md (a tracker, a turn, movement,
// an interaction).
// Path: unhappy (rule interaction).
func TestAttackActionProgressPublicMembers(t *testing.T) {
	t.Parallel()
	banned := []string{"Opportunity", "Reaction", "Tracker", "Turn", "Movement", "Move", "Interact", "Nick", "Extra"}
	members := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if id, ok := recv.(*ast.Ident); !ok || id.Name != "AttackActionProgress" {
				continue
			}
			members++
			text := fn.Name.Name
			for _, fl := range []*ast.FieldList{fn.Type.Params, fn.Type.Results} {
				if fl == nil {
					continue
				}
				ast.Inspect(fl, func(n ast.Node) bool {
					if id, ok := n.(*ast.Ident); ok {
						text += " " + id.Name
					}
					return true
				})
			}
			for _, b := range banned {
				if strings.Contains(text, b) {
					t.Errorf("AttackActionProgress.%s mentions %q (FR-024, FR-031, FR-063)", fn.Name.Name, b)
				}
			}
		}
	}
	if members < 8 {
		t.Errorf("found %d public members of AttackActionProgress, want at least 8", members)
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-001, FR-059, FR-066, FR-070, FR-071, section 6 - package combat exports the
// attack and Help types, variants, groups, refusal enums, results and events once each, the six outcome groups and the four
// refusal enums with their values, and exports no source, scope or combine type or function for attack counts.
// Path: happy.
func TestCombatExportsAttackHelpTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		AttackMeans AttackMeansPactWeapon AttackMeansOtherWeapon AttackMeansUnarmedStrike
		AttackReplacement AttackReplacementKind AttackReplacementSource NewAttackReplacement ReplacementByFeature ReplacementByItem
		AttackEntry MadeAttack ReplacedAttack AttackAllowance NewAttackAllowance BaselineAttackAllowance
		AttackActionProgress StartAttackActionProgress NewAttackActionProgress
		AttackActionWeaponChange AttackActionWeaponChangeTiming WeaponChangeKind
		AttackRefusal AttackRefusalLimitReached WeaponChangeRefusal WeaponChangeRefusalNoAttackMade WeaponChangeRefusalNoAttackRemaining
		WeaponChangeRefusalAlreadyUsedForThisAttack AbilityCheckHelpRefusal AbilityCheckHelpRefusalCannotHelpSelf
		AbilityCheckHelpRefusalHelperLacksProficiency AbilityCheckHelpRefusalAssistanceNotPossible AttackRollHelpRefusal
		AttackRollHelpRefusalEnemyNotWithinFiveFeet AttackRollHelpRefusalTargetNotAnEnemy
		AttackRecordOutcome AttackActionAttackRecorded AttackActionAttackRefused
		WeaponChangeOutcome AttackActionWeaponChangeRecorded AttackActionWeaponChangeRefused
		AbilityCheckHelpOutcome AbilityCheckHelpGiven AbilityCheckHelpRefused
		AttackRollHelpOutcome AttackRollHelpGiven AttackRollHelpRefused
		AbilityCheckHelpUseOutcome AbilityCheckHelpUsed NoAbilityCheckHelp
		AttackRollHelpUseOutcome AttackRollHelpUsed NoAttackRollHelp
		HelpBenefitsExpiry HelpBenefitsDrop HelpBenefits NewHelpBenefits HelpBenefit AbilityCheckHelp NewAbilityCheckHelp
		AttackRollHelp NewAttackRollHelp AssistedProficiency AssistedSkill AssistedTool HelpGiverProficiencies NewHelpGiverProficiencies
		AssistanceJudgement AttackAssistTarget AlliedHelpers NewAlliedHelpers
		AttackActionEvent AttackMadeInAttackAction AttackReplacedInAttackAction WeaponChangedInAttackAction
		NewAttackMadeInAttackAction NewAttackReplacedInAttackAction NewWeaponChangedInAttackAction
		HelpEvent HelpGiven HelpBenefitUsed HelpBenefitExpired HelpBenefitDropped
		NewHelpGiven NewHelpBenefitUsed NewHelpBenefitExpired NewHelpBenefitDropped`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for name := range have {
		if strings.Contains(name, "Combine") || strings.Contains(name, "ExtraAttack") || strings.Contains(name, "AttackCount") {
			t.Errorf("combat declares %s: attack counts are an input, not computed here (FR-066)", name)
		}
	}
	for _, name := range []string{"Applied", "Rejected", "Found"} {
		if len(have[name]) != 0 {
			t.Errorf("combat declares the generic name %s (FR-059)", name)
		}
	}
	// FR-070: each refusal enum is exported once with exactly its values (a refusal never carries another).
	values := map[string]int{}
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, s := range gd.Specs {
				for _, n := range s.(*ast.ValueSpec).Names {
					for _, prefix := range []string{"AttackRefusal", "WeaponChangeRefusal", "AbilityCheckHelpRefusal", "AttackRollHelpRefusal"} {
						if strings.HasPrefix(n.Name, prefix) {
							values[prefix]++
						}
					}
				}
			}
		}
	}
	for prefix, n := range map[string]int{"AttackRefusal": 1, "WeaponChangeRefusal": 3, "AbilityCheckHelpRefusal": 4, "AttackRollHelpRefusal": 3} {
		if values[prefix] != n {
			t.Errorf("%s has %d values, want %d (FR-070)", prefix, values[prefix], n)
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-060, FR-068 - exactly AttackMadeInAttackAction, AttackReplacedInAttackAction and
// WeaponChangedInAttackAction embed the Attack action event base, and exactly HelpGiven, HelpBenefitUsed, HelpBenefitExpired and
// HelpBenefitDropped embed the Help event base, each declared once in its family's file.
// Path: unhappy (duplicate).
func TestAttackHelpEventFamiliesHaveTheirMembers(t *testing.T) {
	t.Parallel()
	families := map[string][]string{
		"attackActionEventBase": {"AttackMadeInAttackAction", "AttackReplacedInAttackAction", "WeaponChangedInAttackAction"},
		"helpEventBase":         {"HelpBenefitDropped", "HelpBenefitExpired", "HelpBenefitUsed", "HelpGiven"},
	}
	files := map[string]string{"attackActionEventBase": "combat/attack_action_event.go", "helpEventBase": "combat/help_event.go"}
	got := map[string][]string{}
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, fld := range st.Fields.List {
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && files[id.Name] != "" {
						got[id.Name] = append(got[id.Name], ts.Name.Name)
						if f.Path != files[id.Name] {
							t.Errorf("%s is declared in %s, want %s", ts.Name.Name, f.Path, files[id.Name])
						}
					}
				}
			}
			return true
		})
	}
	for base, want := range families {
		slices.Sort(got[base])
		if !slices.Equal(got[base], want) {
			t.Errorf("members embedding %s = %v, want %v", base, got[base], want)
		}
	}
}
