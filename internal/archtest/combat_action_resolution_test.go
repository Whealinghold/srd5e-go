package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// actionResolutionDomainFiles are the silent domain files of combat-action-resolution.md (step 10a): the value objects, the stateless services and
// the accumulator of one call. They import neither telemetry nor context and name no use-case type.
func actionResolutionDomainFiles() []string {
	return []string{"combat/resolution_rejection.go", "combat/resolution_outcome.go", "combat/resolution_event.go", "combat/attack_cost.go",
		"combat/attack_scene.go", "combat/action_ledger.go", "combat/turn_request.go", "combat/resolution_precedence.go", "combat/movement_checks.go",
		"combat/damage_handoff.go", "combat/unarmed_effects.go", "combat/attack_condition_inputs.go", "combat/consequence_rules.go", "combat/resolution.go",
		"combat/concentration_input.go", "combat/parry_choice.go"}
}

// actionResolutionUseCaseFiles are the use-case files of combat-action-resolution.md (step 10a).
func actionResolutionUseCaseFiles() []string {
	return []string{"combat/resolution_usecase.go", "combat/perform_attack_usecase.go", "combat/perform_unarmed_strike_usecase.go",
		"combat/take_combatant_action_usecase.go", "combat/move_combatant_usecase.go", "combat/advance_combat_turn_usecase.go",
		"combat/roll_combatant_death_saving_throw_usecase.go", "combat/attempt_to_stabilize_combatant_usecase.go",
		"combat/deal_damage_to_combatant_usecase.go", "combat/concentration_usecase.go", "combat/parry_usecase.go",
		"combat/resolve_stat_block_save_usecase.go"}
}

// Requirement: doc/specs/combat-action-resolution.md FR-001, FR-121, observability.md section 3 - the domain files of the spec are silent and pure:
// they import neither telemetry, context, dice nor d20's attempt types, name no signal constant, and mention no use-case-layer type (the
// two attack results, SaveAttempt, CheckAttempt, RollAttempt, the Roller port) except the files that fold or hand off a d20 value.
// Path: unhappy (forbidden edge).
func TestActionResolutionDomainFilesAreSilentAndPure(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	banned := []string{"ResolveAttackResult", "ResolveUnarmedStrikeResult", "SaveAttempt", "CheckAttempt", "RollAttempt", "Roller", "attackFlow"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(actionResolutionDomainFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if strings.HasPrefix(imp.Path, mod) && strings.TrimPrefix(imp.Path, mod) == "telemetry" || imp.Path == "context" {
				t.Errorf("%s imports %s: the domain files are silent and synchronous (FR-001)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if slices.Contains(banned, id.Name) {
				t.Errorf("%s names %s, a use-case-layer type (FR-121)", f.Path, id.Name)
			}
			for _, prefix := range []string{"metric", "span", "attr", "msg"} {
				if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
					t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, id.Name)
				}
			}
			return true
		})
	}
	if seen != len(actionResolutionDomainFiles()) {
		t.Fatalf("found %d of %d domain files", seen, len(actionResolutionDomainFiles()))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-006, FR-007 - ConditionEffects is asked only by this spec's domain services for the
// attacks and saves it causes (its files and the root's own), and no use-case file names a use case type declared in another file.
// Path: unhappy (forbidden edge).
func TestActionResolutionUseCasesAskNoConditionRulesAndCallNoUseCase(t *testing.T) {
	t.Parallel()
	useCases := []string{"PerformAttack", "PerformUnarmedStrike", "TakeCombatantAction", "MoveCombatant", "AdvanceCombatTurn",
		"RollCombatantDeathSavingThrow", "AttemptToStabilizeCombatant", "DealDamageToCombatant", "ResolveStatBlockSave"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(actionResolutionUseCaseFiles(), f.Path) {
			continue
		}
		seen++
		own := strings.TrimSuffix(strings.TrimPrefix(f.Path, "combat/"), "_usecase.go")
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if id.Name == "ConditionEffects" {
				t.Errorf("%s asks ConditionEffects directly: only the domain services of this spec do (FR-006)", f.Path)
			}
			for _, uc := range useCases {
				if id.Name == uc && !strings.EqualFold(strings.ReplaceAll(own, "_", ""), uc) {
					t.Errorf("%s names the use case %s of another file (FR-007)", f.Path, uc)
				}
			}
			return true
		})
	}
	if seen != len(actionResolutionUseCaseFiles()) {
		t.Fatalf("found %d of %d use-case files", seen, len(actionResolutionUseCaseFiles()))
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-127, FR-094, section 5 - each result has exactly its three variants, and the closed sets of
// rejections, refusals, problems, outcomes, requests and events have exactly the variants of the spec at this revision.
// Path: unhappy (duplicate).
func TestActionResolutionClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isResolutionRejection": {"AttackNotPossibleRejection", "DeathSavingThrowOutOfTurnRejection", "EncounterStateRejection", "LedgerRejection", "MovementRestrictedRejection",
			"PostureMismatchRejection", "StatBlockRejection", "TargetingForbiddenRejection", "TrackerRejection"},
		"isLedgerRefusal": {"LedgerAbilityCheckHelpRefused", "LedgerAttackRefused", "LedgerAttackRollHelpRefused", "LedgerNoAttackActionInProgress",
			"LedgerMultiattackRefused", "LedgerWeaponChangeRefused"},
		"isResolutionProblem": {"LimitExceededProblem", "UnknownCombatantProblem"},
		"isAttackCost":        {"AttackAlreadyPaid", "BeginsAttackAction", "BonusActionCost", "ContinuesAttackAction", "ReactionCost", "RecordsMultiattack"},
		"isReplacementCost":   {"BeginsAttackActionByReplacing", "ContinuesAttackActionByReplacing"},
		"isAttackActionState": {"AttackActionInProgress", "NoAttackActionInProgress"},
		"isTurnRequest": {"BeginMultiattack", "HandleWeapon", "RecordMultiattackReplacement", "RecordMultiattackUse", "ReplaceAttack", "SpendAction", "SpendBonusAction",
			"SpendInteraction", "SpendReaction", "TakeHelpAction", "UseStatBlockPart"},
		"isConcentrationInput":            {"NoConcentrationTracking", "TracksConcentration"},
		"isConcentrationCheckOutcome":     {"ConcentrationCheckFailed", "ConcentrationEndedWithoutSave", "ConcentrationHeld", "ConcentrationNotChecked"},
		"isParryChoice":                   {"NoParry", "ParryIfAble"},
		"isParryOutcome":                  {"ParryApplied", "ParryNotOffered", "ParryNotUsed"},
		"isMultiattackState":              {"MultiattackInProgress", "NoMultiattack"},
		"isHelpRequest":                   {"AbilityCheckHelpRequest", "AttackRollHelpRequest"},
		"isMovementRequest":               {"DropProne", "MoveAlong", "StandUp"},
		"isTestResolution":                {"TestForcedFailure", "TestRolled"},
		"isAttackPerformedOutcome":        {"AttackAutomaticallyMissedOutcome", "AttackHitOutcome", "AttackMissOutcome"},
		"isUnarmedStrikePerformedOutcome": {"UnarmedDamageOptionPerformed", "UnarmedSavePerformed"},
		"isUnarmedEffectOutcome":          {"UnarmedEffectApplied", "UnarmedEffectBlocked", "UnarmedEffectNotTriggered", "UnarmedEffectPushRecorded"},
		"resolutionEvent":                 {"resolutionEventBase"},
	}
	for _, uc := range []string{"PerformAttack", "PerformUnarmedStrike", "TakeCombatantAction", "MoveCombatant", "AdvanceCombatTurn",
		"RollCombatantDeathSavingThrow", "AttemptToStabilizeCombatant", "DealDamageToCombatant", "ResolveStatBlockSave"} {
		want["is"+uc+"Result"] = []string{uc + "Completed", uc + "Invalid", uc + "Rejected"}
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-034, FR-119 - AttackConditionInputs takes no position, allegiance or proficiency: no parameter
// of its public methods is named for one, and the only facts of the scene it reads are those of an AttackScene.
// Path: unhappy (rule interaction).
func TestAttackConditionInputsTakesOnlySceneFacts(t *testing.T) {
	t.Parallel()
	methods := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			if id, ok := fn.Recv.List[0].Type.(*ast.Ident); !ok || id.Name != "AttackConditionInputs" {
				continue
			}
			methods++
			for _, p := range fn.Type.Params.List {
				for _, n := range p.Names {
					for _, bad := range []string{"allegiance", "side", "position", "proficiency", "distance"} {
						if strings.Contains(strings.ToLower(n.Name), bad) {
							t.Errorf("AttackConditionInputs.%s has the parameter %s: only an AttackScene carries such facts (FR-034, FR-119)", fn.Name.Name, n.Name)
						}
					}
				}
			}
		}
	}
	if methods != 4 {
		t.Errorf("AttackConditionInputs has %d public methods, want Attack, Save, Check and ExhaustionReduction", methods)
	}
}
