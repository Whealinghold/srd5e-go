package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// encounterDomainFiles and encounterUseCaseFiles are the production files of encounter-aggregate.md.
func encounterDomainFiles() []string {
	return []string{"combat/encounter.go", "combat/encounter_id.go", "combat/encounter_status.go", "combat/encounter_problem.go",
		"combat/encounter_rejection.go", "combat/encounter_event.go", "combat/encounter_combatant.go", "combat/encounter_transition.go",
		"combat/encounter_turns.go", "combat/encounter_hit_points.go", "combat/encounter_conditions.go"}
}

func encounterUseCaseFiles() []string {
	return []string{"combat/encounter_usecase.go", "combat/encounter_usecases.go"}
}

// Requirement: doc/specs/encounter-aggregate.md section 7 (the domain is silent), FR-061 (no port), observability.md section 3 - the encounter domain files import neither telemetry nor context and name no signal constant; no encounter file imports a package the spec does not use (no dice, no downstream context).
// Path: unhappy (forbidden edge).
func TestEncounterFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		domain := slices.Contains(encounterDomainFiles(), f.Path)
		if !domain && !slices.Contains(encounterUseCaseFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				allowed := []string{"content", "d20", "srderr"}
				if !domain {
					allowed = append(allowed, "telemetry")
				}
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which encounter-aggregate.md does not allow", f.Path, p)
				}
				continue
			}
			switch imp.Path {
			case "slices", "strings", "errors":
			case "context":
				if domain {
					t.Errorf("%s (domain) imports context", f.Path)
				}
			default:
				t.Errorf("%s imports %s, which encounter-aggregate.md does not allow", f.Path, imp.Path)
			}
		}
		if !domain {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if want := len(encounterDomainFiles()) + len(encounterUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d encounter files", seen, want)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-089, FR-090, FR-095 - the closed sets have exactly the variants the spec lists: the status, the end reason, the problems, the ten rejections, the transition and the sixteen use case results.
// Path: unhappy (duplicate).
func TestEncounterClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	results := map[string][]string{}
	for _, uc := range strings.Fields(`AdvanceEncounterTurn ApplyDamageToCombatant ApplyHealingToCombatant ApplyConditionToCombatant RemoveConditionFromCombatant
		ReduceCombatantExhaustion ApplyDeathSavingThrowToCombatant StabilizeCombatant RecoverStableCombatant GiveCombatantTemporaryHitPoints
		ReduceCombatantHitPointMaximum EndEncounter CheckCombatantGrapple ReleaseCombatantGrapple EscapeCombatantGrapple`) {
		results["is"+uc+"Result"] = []string{uc + "Applied", uc + "Rejected"}
	}
	results["isStartEncounterResult"] = []string{"StartEncounterApplied", "StartEncounterInvalid"}
	want := map[string][]string{
		"isEndReason":        {"EndReasonMutualAgreement", "EndReasonSideDefeated"},
		"isEncounterStatus":  {"EncounterEndedStatus", "EncounterInProgress"},
		"isEncounterProblem": {"CombatantLimitProblem", "CombatantNotInTurnOrderProblem", "DuplicateCombatantIDProblem", "MissingSideProblem", "NoCombatantsProblem", "TurnOrderEntryWithoutCombatantProblem"},
		"isEncounterRejection": {"AlreadyHasConditionRejection", "CombatantIsDeadRejection", "ConditionImpliedByRejection", "DeathSavingThrowAlreadyMadeRejection",
			"DoesNotHaveConditionRejection", "EncounterAlreadyEndedRejection", "HitPointRuleRejection", "ImmuneToConditionRejection", "TurnOrderChangedRejection", "UnknownCombatantRejection"},
		"isEncounterTransition": {"transitionApplied", "transitionRejected"},
		"transportEnvelope":     {"ConditionEventOccurred", "GrappleEventOccurred", "HitPointEventOccurred", "TurnEventOccurred"},
		"encounterEvent":        {"encounterEventBase"},
	}
	for k, v := range results {
		want[k] = v
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-050, FR-051 - exactly seven events embed the encounter event base, all declared in encounter_event.go; the four transport envelopes are the only ones named ...Occurred; none is named like a request.
// Path: unhappy (duplicate).
func TestEncounterEventFamilyHasExactlySevenMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "encounterEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/encounter_event.go" {
							t.Errorf("%s is declared in %s, want combat/encounter_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"ConditionEventOccurred", "DyingCombatantTurnStarted", "EncounterEnded", "EncounterStarted", "GrappleEventOccurred", "HitPointEventOccurred", "TurnEventOccurred"}
	if !slices.Equal(members, want) {
		t.Errorf("encounter event members = %v, want %v", members, want)
	}
	for _, m := range members {
		occurred := strings.HasSuffix(m, "Occurred")
		if occurred != slices.Contains(markerReceivers(t, "transportEnvelope"), m) {
			t.Errorf("%s: the suffix Occurred and the transport envelope marker disagree (FR-050)", m)
		}
		for _, bad := range []string{"Due", "Required", "Needed"} {
			if strings.HasSuffix(m, bad) {
				t.Errorf("%s is named like a request, not a fact", m)
			}
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-089, FR-090, section 6 (Exports), plan-phase-4.md P7 - package combat exports the encounter types and use cases once each, no generic variant name, no exported domain-level transition, every rejection and problem type ends in Rejection or Problem, and no name repeats a hit point rejection.
// Path: happy.
func TestCombatExportsEncounterTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		Encounter Combatant NewCombatant EncounterID EncounterIDError EncounterIDRejection EncounterIDEmpty ParseEncounterID MustEncounterID Side SideA SideB
		EndReason EndReasonSideDefeated EndReasonMutualAgreement EncounterStatus EncounterInProgress EncounterEndedStatus
		EncounterProblem CombatantLimitProblem NoCombatantsProblem MissingSideProblem DuplicateCombatantIDProblem CombatantNotInTurnOrderProblem TurnOrderEntryWithoutCombatantProblem
		EncounterProblemError ProblemsOf EncounterRejection EncounterAlreadyEndedRejection UnknownCombatantRejection CombatantIsDeadRejection
		AlreadyHasConditionRejection ImmuneToConditionRejection DoesNotHaveConditionRejection ConditionImpliedByRejection HitPointRuleRejection
		TurnOrderChangedRejection DeathSavingThrowAlreadyMadeRejection EncounterEvent EncounterStarted EncounterEnded DyingCombatantTurnStarted
		TurnEventOccurred HitPointEventOccurred ConditionEventOccurred GrappleEventOccurred NewTurnEventOccurred NewHitPointEventOccurred
		NewConditionEventOccurred NewGrappleEventOccurred
		StartEncounter NewStartEncounter StartEncounterCommand NewStartEncounterCommand StartEncounterResult StartEncounterApplied StartEncounterInvalid
		AdvanceEncounterTurn NewAdvanceEncounterTurn AdvanceEncounterTurnCommand NewAdvanceEncounterTurnCommand
		ApplyDamageToCombatant NewApplyDamageToCombatant ApplyDamageToCombatantCommand NewApplyDamageToCombatantCommand
		ApplyHealingToCombatant NewApplyHealingToCombatant ApplyHealingToCombatantCommand NewApplyHealingToCombatantCommand
		ApplyConditionToCombatant NewApplyConditionToCombatant ApplyConditionToCombatantCommand NewApplyConditionToCombatantCommand
		RemoveConditionFromCombatant NewRemoveConditionFromCombatant RemoveConditionFromCombatantCommand NewRemoveConditionFromCombatantCommand
		ReduceCombatantExhaustion NewReduceCombatantExhaustion ReduceCombatantExhaustionCommand NewReduceCombatantExhaustionCommand
		ApplyDeathSavingThrowToCombatant NewApplyDeathSavingThrowToCombatant ApplyDeathSavingThrowToCombatantCommand NewApplyDeathSavingThrowToCombatantCommand
		StabilizeCombatant NewStabilizeCombatant StabilizeCombatantCommand NewStabilizeCombatantCommand
		RecoverStableCombatant NewRecoverStableCombatant RecoverStableCombatantCommand NewRecoverStableCombatantCommand
		GiveCombatantTemporaryHitPoints NewGiveCombatantTemporaryHitPoints GiveCombatantTemporaryHitPointsCommand NewGiveCombatantTemporaryHitPointsCommand
		ReduceCombatantHitPointMaximum NewReduceCombatantHitPointMaximum ReduceCombatantHitPointMaximumCommand NewReduceCombatantHitPointMaximumCommand
		EndEncounter NewEndEncounter EndEncounterCommand NewEndEncounterCommand
		CheckCombatantGrapple NewCheckCombatantGrapple CheckCombatantGrappleCommand NewCheckCombatantGrappleCommand
		ReleaseCombatantGrapple NewReleaseCombatantGrapple ReleaseCombatantGrappleCommand NewReleaseCombatantGrappleCommand
		EscapeCombatantGrapple NewEscapeCombatantGrapple EscapeCombatantGrappleCommand NewEscapeCombatantGrappleCommand`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, uc := range strings.Fields("AdvanceEncounterTurn ApplyDamageToCombatant ApplyHealingToCombatant ApplyConditionToCombatant RemoveConditionFromCombatant ReduceCombatantExhaustion ApplyDeathSavingThrowToCombatant StabilizeCombatant RecoverStableCombatant GiveCombatantTemporaryHitPoints ReduceCombatantHitPointMaximum EndEncounter CheckCombatantGrapple ReleaseCombatantGrapple EscapeCombatantGrapple") {
		for _, suffix := range []string{"Result", "Applied", "Rejected"} {
			if len(have[uc+suffix]) != 1 {
				t.Errorf("combat declares %s%s %d times, want once", uc, suffix, len(have[uc+suffix]))
			}
		}
	}
	for _, hidden := range []string{"EncounterTransition", "TransitionApplied", "TransitionRejected", "EncounterStart", "Started"} {
		if len(have[hidden]) != 0 {
			t.Errorf("combat exports %s, which FR-089 keeps unexported", hidden)
		}
	}
	for _, generic := range []string{"InProgress", "Ended", "SideDefeated", "MutualAgreement", "NoCombatants", "MissingSide", "EncounterAlreadyEnded", "UnknownCombatant", "CombatantIsDead", "TurnOrderChanged"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat declares the generic name %s (FR-090)", generic)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md section 7, FR-057, FR-062, FR-095, plan-phase-4.md P11 - signals.go pins the sixteen operation values and spans, the one new metric and the invalid start result at their catalogue values; signals.md has a row for each span and the metric and lists none of the superseded encounter names (encounter_transitions, encounter_errors, encounter_duration, encounter_events, srd5e.combat.rejection, srd5e.combat.error_kind).
// Path: unhappy (invalid input).
func TestEncounterSignalNamesAndCatalogue(t *testing.T) {
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
	want := map[string]string{"metricEncounterSize": "srd5e.combat.encounter_size", "valueResultInvalidEncounter": "invalid_encounter"}
	doc := catalogue(t)
	for _, op := range strings.Fields(`StartEncounter:start_encounter AdvanceEncounterTurn:advance_encounter_turn ApplyDamageToCombatant:apply_damage_to_combatant
		ApplyHealingToCombatant:apply_healing_to_combatant ApplyConditionToCombatant:apply_condition_to_combatant RemoveConditionFromCombatant:remove_condition_from_combatant
		ReduceCombatantExhaustion:reduce_combatant_exhaustion ApplyDeathSavingThrowToCombatant:apply_death_saving_throw_to_combatant StabilizeCombatant:stabilize_combatant
		RecoverStableCombatant:recover_stable_combatant GiveCombatantTemporaryHitPoints:give_combatant_temporary_hit_points
		ReduceCombatantHitPointMaximum:reduce_combatant_hit_point_maximum EndEncounter:end_encounter CheckCombatantGrapple:check_combatant_grapple
		ReleaseCombatantGrapple:release_combatant_grapple EscapeCombatantGrapple:escape_combatant_grapple`) {
		name, value, _ := strings.Cut(op, ":")
		want["valueOperation"+name] = value
		want["span"+name] = "srd5e.combat." + value
		if !strings.Contains(doc, "| `srd5e.combat."+value+"` |") {
			t.Errorf("signals.md has no row for the span srd5e.combat.%s", value)
		}
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
		if strings.HasPrefix(name, "metric") && !strings.Contains(doc, "| `"+value+"` |") {
			t.Errorf("signals.md has no row for the metric %s", value)
		}
	}
	for _, old := range []string{"encounter_transitions", "encounter_errors", "encounter_duration", "encounter_events", "srd5e.combat.rejection", "srd5e.combat.error_kind"} {
		if strings.Contains(doc, old) {
			t.Errorf("signals.md lists the superseded name %s (P11)", old)
		}
	}
}
