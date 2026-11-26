package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// conditionFiles are the production files of conditions.md: domain files and use-case files.
func conditionFiles(t *testing.T) (domain, useCase map[string]bool) {
	t.Helper()
	domain = map[string]bool{}
	useCase = map[string]bool{}
	for _, f := range combatPackage(t).Production() {
		switch name := f.Path[len("combat/"):]; name {
		case "apply_condition_to_set.go", "remove_condition_from_set.go", "reduce_exhaustion_on_set.go", "condition_usecase.go":
			useCase[f.Path] = true
		case "exhaustion_level.go", "roll_contributions.go":
			domain[f.Path] = true
		default:
			if strings.HasPrefix(name, "condition") {
				domain[f.Path] = true
			}
		}
	}
	return domain, useCase
}

// Requirement: doc/specs/conditions.md FR-131 - the condition domain files import only content, modifier, ability and
// srderr of this module (no telemetry, no character, no later combat spec's code); the use-case files add telemetry.
// Path: unhappy (forbidden edge).
func TestConditionFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	domain, useCase := conditionFiles(t)
	if len(domain) < 8 || len(useCase) != 4 {
		t.Fatalf("found %d domain and %d use-case condition files", len(domain), len(useCase))
	}
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"content", "modifier", "ability", "srderr"}
		switch {
		case useCase[f.Path]:
			allowed = append(allowed, "telemetry")
		case !domain[f.Path]:
			continue
		}
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which conditions.md FR-131 does not allow", f.Path, p)
			}
		}
	}
}

// Requirement: doc/specs/conditions.md FR-140 - the three result sets are declared in condition_result.go, and no
// condition domain file names a use-case type or command.
// Path: unhappy (forbidden edge).
func TestConditionResultsLiveInOneFileAndDomainNamesNoUseCase(t *testing.T) {
	t.Parallel()
	d := declared(combatPackage(t))
	for _, name := range []string{"ApplyConditionToSetResult", "ConditionAdded", "ConditionAlreadyHeld", "ConditionImmune", "ExhaustionAtMaximum",
		"RemoveConditionFromSetResult", "ConditionCleared", "ExhaustionLowered", "ConditionRemovalNotPresent", "ConditionRemovalImpliedBy",
		"ReduceExhaustionOnSetResult", "ExhaustionReduced", "ExhaustionEnded", "ExhaustionReductionNotPresent"} {
		if !slices.Equal(d[name], []string{"combat/condition_result.go"}) {
			t.Errorf("%s is declared in %v, want only combat/condition_result.go", name, d[name])
		}
	}
	useCaseTypes := []string{"ApplyConditionToSet", "ApplyConditionToSetCommand", "RemoveConditionFromSet", "RemoveConditionFromSetCommand",
		"ReduceExhaustionOnSet", "ReduceExhaustionOnSetCommand", "conditionUse"}
	domain, _ := conditionFiles(t)
	for _, f := range combatPackage(t).Production() {
		if !domain[f.Path] {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(useCaseTypes, id.Name) {
				t.Errorf("%s refers to the use-case name %s", f.Path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/conditions.md FR-139, FR-129 - exactly ConditionApplied, ConditionRemoved and ConditionLevelReduced
// embed the condition event base, and they are declared once, in condition_event.go.
// Path: unhappy (duplicate).
func TestConditionEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "conditionEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/condition_event.go" {
							t.Errorf("%s is declared in %s, want combat/condition_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"ConditionApplied", "ConditionLevelReduced", "ConditionRemoved"}; !slices.Equal(members, want) {
		t.Errorf("condition event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/conditions.md FR-119, FR-122 and plan-phase-4.md P11 - the signals file holds the surviving
// srd5e.combat.condition.active and .kind names and none of the superseded condition names.
// Path: unhappy (invalid input).
func TestConditionSignalNames(t *testing.T) {
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
	if got["metricConditionActive"] != "srd5e.combat.condition.active" || got["attrConditionKind"] != "srd5e.combat.condition.kind" {
		t.Errorf("condition signal constants = %q, %q", got["metricConditionActive"], got["attrConditionKind"])
	}
	for name, value := range got {
		if slices.Contains(supersededSignals(), value) {
			t.Errorf("signals.go constant %s holds the superseded name %s", name, value)
		}
	}
}

// Requirement: doc/specs/conditions.md section 6 - package combat exports the condition types, queries, results, events and
// use cases once each.
// Path: happy.
func TestCombatExportsConditionTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		Condition Blinded Charmed Deafened ExhaustionCondition Frightened Grappled Incapacitated Invisible Paralyzed Petrified Poisoned Prone
		Restrained Stunned Unconscious Charmer CharmedByCombatant CharmedByItem NewCharmedByItem ExhaustionLevel NewExhaustionLevel MustExhaustionLevel
		ConditionSet NewConditionSet MustConditionSet ConditionDefinition ConditionDefinitionOf ConditionEffect ConditionEffects
		AutoFailAbilityChecks AttackRollsAgainst OwnAttackRolls OwnAbilityChecks SavingThrowsAutoFail SavingThrowDisadvantage SpeedZero SpeedReducedBy
		D20TestReducedBy AutomaticCriticalHit NoActions ConcentrationBroken CannotSpeak InitiativeCircumstance CharmerRestriction SocialAdvantageForSource
		CannotApproachSource Concealed DamageResistanceAll ImmuneTo DropsHeldItems Unaware
		RollContribution RollContributions ConditionAttackFacts EffectiveSpeed ProneMovement CrawlOnly CrawlOrRightItself ConditionFlags
		Sense CheckRequirement SightRelation LineOfSight EffectDamage ConditionPermission SavingThrowEffect AbilityCheckEffect AutomaticCritical SpeedIncrease
		ConditionConcentration Speech ConditionActionAvailability HeldItems Awareness AllDamageResistance EffectCondition
		ConditionEvent ConditionApplied ConditionRemoved ConditionLevelReduced NewConditionApplied NewConditionRemoved NewConditionLevelReduced
		ApplyConditionToSet ApplyConditionToSetCommand NewApplyConditionToSet NewApplyConditionToSetCommand
		RemoveConditionFromSet RemoveConditionFromSetCommand NewRemoveConditionFromSet NewRemoveConditionFromSetCommand
		ReduceExhaustionOnSet ReduceExhaustionOnSetCommand NewReduceExhaustionOnSet NewReduceExhaustionOnSetCommand`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
}
