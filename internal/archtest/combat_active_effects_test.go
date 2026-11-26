package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// activeEffectsDomainFiles and activeEffectsUseCaseFiles are the production files of active-effects.md.
func activeEffectsDomainFiles() []string {
	return []string{"combat/active_effect_identity.go", "combat/active_effect_duration.go", "combat/active_effect_clock.go", "combat/active_effect_template.go",
		"combat/active_effect.go", "combat/active_effect_event.go", "combat/active_effect_plan.go", "combat/active_effect_ending.go", "combat/active_effect_work.go",
		"combat/active_effect_start.go", "combat/active_effect_end_rules.go", "combat/active_effect_advance.go"}
}

func activeEffectsUseCaseFiles() []string {
	return []string{"combat/active_effects_usecase.go", "combat/start_active_effect_usecase.go", "combat/advance_active_effects_usecase.go",
		"combat/end_active_effects_usecase.go", "combat/resolve_effect_damage_usecase.go"}
}

// activeEffectUseCases are the identifiers of the four use cases, which no file of another use case may name (FR-056).
func activeEffectUseCases() map[string][]string {
	return map[string][]string{
		"combat/start_active_effect_usecase.go":    {"AdvanceActiveEffects", "NewAdvanceActiveEffects", "EndActiveEffects", "NewEndActiveEffects", "ResolveEffectDamage", "NewResolveEffectDamage"},
		"combat/advance_active_effects_usecase.go": {"StartActiveEffect", "NewStartActiveEffect", "EndActiveEffects", "NewEndActiveEffects", "ResolveEffectDamage", "NewResolveEffectDamage"},
		"combat/end_active_effects_usecase.go":     {"StartActiveEffect", "NewStartActiveEffect", "AdvanceActiveEffects", "NewAdvanceActiveEffects", "ResolveEffectDamage", "NewResolveEffectDamage"},
		"combat/resolve_effect_damage_usecase.go":  {"StartActiveEffect", "NewStartActiveEffect", "AdvanceActiveEffects", "NewAdvanceActiveEffects", "EndActiveEffects", "NewEndActiveEffects"},
	}
}

// Requirement: doc/specs/active-effects.md FR-053, FR-056 - the domain files import only the kernel, content, character, d20 and the standard library's errors, fmt, math, slices and strings, never telemetry, context or a roller; they name no signal constant, no Roller and no dice.Roll; no file of the four use cases names another use case or any other use case of the package.
// Path: unhappy (forbidden edge).
func TestActiveEffectsFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "character", "content", "d20", "dice", "modifier", "srderr"}
		domain := slices.Contains(activeEffectsDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(activeEffectsUseCaseFiles(), f.Path):
			allowed = append(allowed, "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) || (domain && p == "telemetry") {
					t.Errorf("%s imports %s, which active-effects.md does not allow", f.Path, p)
				}
				continue
			}
			switch imp.Path {
			case "errors", "fmt", "math", "slices", "strings":
			case "context":
				if domain {
					t.Errorf("%s imports context: the domain files are pure (FR-053)", f.Path)
				}
			default:
				t.Errorf("%s imports %s, which active-effects.md does not allow", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if id, ok := x.X.(*ast.Ident); ok && domain && id.Name == "dice" && x.Sel.Name == "Roll" {
					t.Errorf("%s calls dice.Roll: the domain requests no die itself (FR-053)", f.Path)
				}
			case *ast.Ident:
				if slices.Contains(activeEffectUseCases()[f.Path], x.Name) {
					t.Errorf("%s names %s: no use case calls another (FR-056)", f.Path, x.Name)
				}
				if !domain {
					return true
				}
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(x.Name, prefix) && len(x.Name) > len(prefix) && x.Name[len(prefix)] >= 'A' && x.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, x.Name)
					}
				}
				if x.Name == "Roller" || x.Name == "attackFlow" || slices.Contains(otherUseCases(), x.Name) {
					t.Errorf("%s names %s: the domain holds no roller and calls no use case (FR-053, FR-056)", f.Path, x.Name)
				}
			}
			return true
		})
	}
	if want := len(activeEffectsDomainFiles()) + len(activeEffectsUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d active effects files", seen, want)
	}
}

// Requirement: doc/specs/active-effects.md FR-003, FR-004, FR-005, FR-040, FR-054, FR-055 - each closed set has exactly the variants the spec lists.
// Path: unhappy (duplicate).
func TestActiveEffectsClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isEffectOrigin":               {"OtherEffectOrigin", "SpellEffectOrigin"},
		"isEffectSubject":              {"CreatureEffectSubject", "PlaceEffectSubject"},
		"isEffectDuration":             {"EffectLastsIndefinitely", "EffectLastsRemainder", "EffectLastsRounds", "EffectLastsUntilTurnBoundary"},
		"isEffectClockTick":            {"CombatantLeft", "EndOfCombatReached", "RoundStartReached", "TimeElapsed", "TurnEndReached", "TurnStartReached"},
		"isEffectAction":               {"DealDamage", "DealStoredDamage", "GrowStoredDamage", "RepeatSave"},
		"isEndingCause":                {"ConcentrationLost", "Dismissed", "MoveEffectSubject", "RemovedByEffect", "SubjectDied", "SubjectTookDamage"},
		"activeEffectEvent":            {"activeEffectEventBase"},
		"isStartActiveEffectResult":    {"StartActiveEffectRejected", "StartActiveEffectStarted"},
		"isAdvanceActiveEffectsResult": {"AdvanceActiveEffectsApplied", "AdvanceActiveEffectsRejected"},
		"isEndActiveEffectsResult":     {"EndActiveEffectsEnded", "EndActiveEffectsMoved", "EndActiveEffectsNothingEnded", "EndActiveEffectsRejected"},
		"isResolveEffectDamageResult":  {"ResolveEffectDamageRejected", "ResolveEffectDamageResolved"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/active-effects.md FR-056, section 6 - NewAdvanceActiveEffects and NewResolveEffectDamage take a Roller and options only; NewStartActiveEffect and NewEndActiveEffects take options only.
// Path: unhappy (forbidden edge).
func TestActiveEffectsConstructorsTakeOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	want := map[string]string{"NewAdvanceActiveEffects": "Roller,...Option", "NewResolveEffectDamage": "Roller,...Option", "NewStartActiveEffect": "...Option", "NewEndActiveEffects": "...Option"}
	found := map[string]int{}
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || want[fn.Name.Name] == "" {
				continue
			}
			found[fn.Name.Name]++
			var params []string
			for _, p := range fn.Type.Params.List {
				switch x := p.Type.(type) {
				case *ast.Ident:
					params = append(params, x.Name)
				case *ast.Ellipsis:
					params = append(params, "..."+x.Elt.(*ast.Ident).Name)
				}
			}
			if got := strings.Join(params, ","); got != want[fn.Name.Name] {
				t.Errorf("%s takes (%s), want (%s)", fn.Name.Name, got, want[fn.Name.Name])
			}
		}
	}
	for name := range want {
		if found[name] != 1 {
			t.Errorf("found %d %s, want 1", found[name], name)
		}
	}
}

// Requirement: doc/specs/active-effects.md section 7, FR-058 - signals.go pins the four operations and spans, the three metrics, three attribute keys, the save kinds and the eight result values at their catalogue values, and signals.md has a row for each metric, attribute and span.
// Path: unhappy (invalid input).
func TestActiveEffectsSignalNamesAndCatalogue(t *testing.T) {
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
		"valueOperationStartActiveEffect": "start_active_effect", "spanStartActiveEffect": "srd5e.combat.start_active_effect",
		"valueOperationAdvanceActiveEffects": "advance_active_effects", "spanAdvanceActiveEffects": "srd5e.combat.advance_active_effects",
		"valueOperationEndActiveEffects": "end_active_effects", "spanEndActiveEffects": "srd5e.combat.end_active_effects",
		"valueOperationResolveEffectDamage": "resolve_effect_damage", "spanResolveEffectDamage": "srd5e.combat.resolve_effect_damage",
		"metricActiveEffectsEnded": "srd5e.combat.active_effects.ended", "metricActiveEffectsSaves": "srd5e.combat.active_effects.saves",
		"metricActiveEffectsActive": "srd5e.combat.active_effects.active", "attrActiveEffectsRejection": "srd5e.combat.active_effects.rejection",
		"attrActiveEffectsEndReason": "srd5e.combat.active_effects.end_reason", "attrActiveEffectsSaveKind": "srd5e.combat.active_effects.save_kind",

		"valueResultEffectStarted": "started", "valueResultEffectsAdvanced": "advanced", "valueResultEffectsEnded": "ended", "valueResultEffectMoved": "moved",
		"valueResultEffectsNothingEnded": "nothing_ended", "valueResultEffectDamageResolved": "effect_damage_resolved",
		"valueResultEffectsRejectedInvalid": "rejected_invalid", "valueResultEffectsRejectedRefused": "rejected_refused",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	doc := catalogue(t)
	for _, name := range []string{"srd5e.combat.start_active_effect", "srd5e.combat.advance_active_effects", "srd5e.combat.end_active_effects", "srd5e.combat.resolve_effect_damage",
		"srd5e.combat.active_effects.ended", "srd5e.combat.active_effects.saves", "srd5e.combat.active_effects.active", "srd5e.combat.active_effects.rejection",
		"srd5e.combat.active_effects.end_reason", "srd5e.combat.active_effects.save_kind"} {
		if !strings.Contains(doc, "| `"+name+"` |") {
			t.Errorf("signals.md has no row for %s", name)
		}
	}
}

// Requirement: doc/specs/active-effects.md section 6 (Exports), FR-055, FR-059, testing plan FR-053 to FR-059 - package combat declares the exported names of the active effects once each, no generic variant name, and no exported name ends in Due or Required.
// Path: happy.
func TestCombatExportsActiveEffectsTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		ActiveEffectID NewActiveEffectID MustActiveEffectID EffectPlaceID NewEffectPlaceID EffectOrigin SpellEffectOrigin NewSpellEffectOrigin OtherEffectOrigin
		NewOtherEffectOrigin EffectSubject CreatureEffectSubject NewCreatureEffectSubject PlaceEffectSubject NewPlaceEffectSubject ConcentrationBinding NewConcentrationBinding
		EffectDuration EffectLastsRounds NewEffectLastsRounds EffectLastsUntilTurnBoundary NewEffectLastsUntilTurnBoundary EffectLastsIndefinitely EffectLastsRemainder
		RoundsInTimeSpan EffectDurationFromSpell EffectDurationFromCondition EffectClockTick TurnStartReached NewTurnStartReached TurnEndReached NewTurnEndReached
		RoundStartReached TimeElapsed NewTimeElapsed EndOfCombatReached CombatantLeft NewCombatantLeft ClockTickFromTurnEvent EffectPotency EffectCombiningSubject
		CombiningOnTarget CombiningOnCaster EffectTurnOwner EffectDamageAudience EffectSaveSuccess EffectDamageSave NewEffectDamageSave EffectAction RepeatSave NewRepeatSave
		DealDamage NewDealDamage GrowStoredDamage NewGrowStoredDamage DealStoredDamage NewDealStoredDamage EffectTrigger NewEffectTrigger EffectModelProblem EffectModelError
		EffectStageParams EffectStage NewEffectStage EffectTemplateParams EffectTemplate NewEffectTemplate ActiveEffect ActiveEffects NewActiveEffects ApplyingEffect
		EffectCombination ActiveEffectEvent ActiveEffectStarted ActiveEffectEnded EffectSaveKind ActiveEffectRepeatSaveMade ActiveEffectStageAdvanced ActiveEffectStoredDamageGrew
		ActiveEffectSubjectMoved EffectEndReason EndingCause ConcentrationLost NewConcentrationLost Dismissed NewDismissed SubjectTookDamage NewSubjectTookDamage SubjectDied
		NewSubjectDied RemovedByEffect NewRemovedByEffect MoveEffectSubject NewMoveEffectSubject EffectConditionPlan EffectDamagePlan ActiveEffectRejectionKind
		ActiveEffectProblem ActiveEffectRejection StartActiveEffect NewStartActiveEffect StartActiveEffectCommand StartActiveEffectResult StartActiveEffectStarted
		StartActiveEffectRejected AdvanceActiveEffects NewAdvanceActiveEffects AdvanceActiveEffectsCommand AdvanceActiveEffectsResult AdvanceActiveEffectsApplied
		AdvanceActiveEffectsRejected EndActiveEffects NewEndActiveEffects EndActiveEffectsCommand EndActiveEffectsResult EndActiveEffectsEnded EndActiveEffectsMoved
		EndActiveEffectsNothingEnded EndActiveEffectsRejected ResolveEffectDamage NewResolveEffectDamage ResolveEffectDamageCommand ResolveEffectDamageResult
		ResolveEffectDamageResolved ResolveEffectDamageRejected EffectDamageTarget EffectDamageAssignment`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-055)", generic)
		}
	}
	for _, name := range want {
		if strings.HasSuffix(name, "Due") || strings.HasSuffix(name, "Required") {
			t.Errorf("the active effects export %s, which ends in Due or Required (FR-059)", name)
		}
	}
}
