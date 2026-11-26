package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// environmentFiles are the production files of environment-hazards-and-objects.md: domain files (hazard_*.go and
// object_*.go) and use-case files (one per operation, plus the shared environment_usecase.go).
func environmentFiles(t *testing.T) (domain, useCase map[string]bool) {
	t.Helper()
	domain, useCase = map[string]bool{}, map[string]bool{}
	for _, f := range combatPackage(t).Production() {
		switch name := f.Path[len("combat/"):]; name {
		case "roll_burning_damage.go", "extinguish_burning.go", "resolve_fall.go", "land_after_fall.go", "resolve_dehydration_day_end.go",
			"resolve_malnutrition_day_end.go", "advance_breath.go", "resolve_suffocation_turn_end.go", "restore_breathing.go",
			"remove_exhaustion_levels.go", "roll_object_hit_points.go", "apply_damage_to_object.go", "environment_usecase.go":
			useCase[f.Path] = true
		default:
			if strings.HasPrefix(name, "hazard_") || strings.HasPrefix(name, "object_") {
				domain[f.Path] = true
			}
		}
	}
	return domain, useCase
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-116, FR-118, FR-123, FR-124 - the hazard and object domain files import only
// content, d20, ability, dice and srderr of this module (no telemetry, no character, no later combat spec's code); the use-case files add
// telemetry; neither names a MagicItemResilience type, UnderwaterStatus or a magic item package.
// Path: unhappy (forbidden edge).
func TestEnvironmentFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	domain, useCase := environmentFiles(t)
	if len(domain) < 12 || len(useCase) != 13 {
		t.Fatalf("found %d domain and %d use-case environment files", len(domain), len(useCase))
	}
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"content", "d20", "ability", "dice", "srderr"}
		switch {
		case useCase[f.Path]:
			allowed = append(allowed, "telemetry")
		case !domain[f.Path]:
			continue
		}
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which environment-hazards-and-objects.md FR-116 does not allow", f.Path, p)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "MagicItemResilience") || id.Name == "UnderwaterStatus") {
				t.Errorf("%s names %s (FR-123, FR-124)", f.Path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-116, observability.md section 3 - the domain files name no signal constant:
// they are silent.
// Path: unhappy (forbidden edge).
func TestEnvironmentDomainFilesAreSilent(t *testing.T) {
	t.Parallel()
	domain, _ := environmentFiles(t)
	for _, f := range combatPackage(t).Production() {
		if !domain[f.Path] {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the domain is silent (FR-116)", f.Path, id.Name)
			}
			return true
		})
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-105 - exactly the eleven environment events embed one of the two environment
// event bases, all declared once in hazard_event.go.
// Path: unhappy (duplicate).
func TestEnvironmentEventFamilyHasExactlyElevenMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && (id.Name == "creatureEnvironmentEventBase" || id.Name == "objectEnvironmentEventBase") {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/hazard_event.go" {
							t.Errorf("%s is declared in %s, want combat/hazard_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"BeganChoking", "BreathHeldStarted", "BreathRestored", "BurningEnded", "DehydrationRecovered", "MalnutritionRecovered",
		"ObjectCollapsed", "ObjectDamageDeflected", "ObjectDamaged", "ObjectDestroyed", "RanOutOfBreath"}
	if !slices.Equal(members, want) {
		t.Errorf("environment event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-099, FR-119, plan-phase-4.md P11 - the signals file pins the twelve operation
// values and spans, the two own metrics and the hazard kind key at their catalogue values, with no superseded name.
// Path: unhappy (invalid input).
func TestEnvironmentSignalNames(t *testing.T) {
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
		"metricHazardDamageRolled": "srd5e.combat.hazard.damage_rolled", "metricHazardExhaustionGained": "srd5e.combat.hazard.exhaustion_gained",
		"attrHazardKind": "srd5e.combat.hazard.kind",
	}
	for _, op := range strings.Fields(`RollBurningDamage:roll_burning_damage ExtinguishBurning:extinguish_burning ResolveFall:resolve_fall
		LandAfterFall:land_after_fall ResolveDehydrationDayEnd:resolve_dehydration_day_end ResolveMalnutritionDayEnd:resolve_malnutrition_day_end
		AdvanceBreath:advance_breath ResolveSuffocationTurnEnd:resolve_suffocation_turn_end RestoreBreathing:restore_breathing
		RemoveExhaustionLevels:remove_exhaustion_levels RollObjectHitPoints:roll_object_hit_points ApplyDamageToObject:apply_damage_to_object`) {
		name, value, _ := strings.Cut(op, ":")
		want["valueOperation"+name] = value
		want["span"+name] = "srd5e.combat." + value
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	for name, value := range got {
		if slices.Contains(supersededSignals(), value) {
			t.Errorf("signals.go constant %s holds the superseded name %s", name, value)
		}
	}
}

// Requirement: doc/specs/environment-hazards-and-objects.md FR-104, FR-113, FR-114, FR-117 - package combat exports the hazard and object
// types, results, events and use cases once each, declares no bare Burning or breathing name and no building or vehicle type, and no
// generic variant name.
// Path: happy.
func TestCombatExportsEnvironmentTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		HazardKind HazardBurning HazardDehydration HazardFalling HazardMalnutrition HazardSuffocation HazardExhaustionOutcome HazardNoExhaustion
		HazardExhaustionGained HazardExhaustionAtMaximum HazardImmune HazardRecord HazardRecordRejection HazardRecordError NewHazardRecord MustHazardRecord
		EmptyHazardRecord BurningState BurningStateNotBurning BurningStateBurning BurningRejection BurningRejectionNotBurning
		BurningRejectionExtinguishBlockedByImmunity ExtinguishMethod ProneApplicationOutcome ProneApplied ProneNotApplied
		FallDistance NewFallDistance MustFallDistance LiquidLanding NoLandingReaction LandingCheckMade FallSurface FallOntoGround FallIntoLiquid
		FallLandingCheck FallLandingOption WaterAmount NewWaterAmount FoodAmount NewFoodAmount WaterNeedsPerDay FoodNeedsPerDay
		FoodSavingThrow FoodSaveNotRolled FoodSaveRolled BreathCapacity SuffocationCause BreathState BreathNormal BreathHolding NewBreathHolding
		BreathSuffocating NewBreathSuffocating BreathChange BeginHoldingBreath HoldBreathFor BeginChoking BreathRejection
		ExhaustionReductionOutcome ExhaustionReductionMade ExhaustionReductionNotAttempted ExhaustionRemovalCause ExhaustionRemovalRejection
		EnvironmentEvent BurningEnded DehydrationRecovered MalnutritionRecovered BreathHeldStarted RanOutOfBreath BeganChoking BreathRestored
		ObjectDamaged ObjectDamageDeflected ObjectDestroyed ObjectCollapsed
		ObjectSubstance ObjectArmorClassTable ObjectDurability ObjectHitPointsEntry ObjectHitPointsLookup ObjectHitPointsFound
		ObjectHitPointsSplitRequired ObjectHitPointsTable ObjectAbilityScores NoAbilityScores AssignedAbilityScores NewAssignedAbilityScores
		ObjectAffinities ObjectStatus NewObjectStatus DamageThreshold NewDamageThreshold ThresholdOutcome DamageThresholdGate ObjectHarm
		ObjectDamageInstance NewObjectDamageInstance ObjectBrokenAutomatically AutomaticBreakBasis ObjectDamageRejection CollapseRuling
		DoesNotCollapse CollapsesWhole CompositeStatus CompositeSection CompositeObject NewCompositeObject
		RollBurningDamage RollBurningDamageCommand RollBurningDamageResult BurningDamageRolled BurningDamageRejected NewRollBurningDamage
		ExtinguishBurning ExtinguishBurningCommand ExtinguishBurningResult BurningExtinguished BurningExtinguishRejected NewExtinguishBurning
		ResolveFall ResolveFallCommand ResolveFallResult FallDamageRolled FallWithoutDamage NewResolveFall
		LandAfterFall LandAfterFallCommand LandAfterFallResult FallLandedProne FallLandedStanding NewLandAfterFall
		ResolveDehydrationDayEnd ResolveDehydrationDayEndCommand ResolveDehydrationDayEndResult DehydrationDayEnded NewResolveDehydrationDayEnd
		ResolveMalnutritionDayEnd ResolveMalnutritionDayEndCommand ResolveMalnutritionDayEndResult MalnutritionDayEnded MalnutritionSaveRequired NewResolveMalnutritionDayEnd
		AdvanceBreath AdvanceBreathCommand AdvanceBreathResult BreathAdvanced BreathAdvanceRejected NewAdvanceBreath
		ResolveSuffocationTurnEnd ResolveSuffocationTurnEndCommand ResolveSuffocationTurnEndResult SuffocationTurnEnded NewResolveSuffocationTurnEnd
		RestoreBreathing RestoreBreathingCommand RestoreBreathingResult BreathingRestored NewRestoreBreathing
		RemoveExhaustionLevels RemoveExhaustionLevelsCommand RemoveExhaustionLevelsResult ExhaustionLevelsRemoved RemoveExhaustionLevelsRejected NewRemoveExhaustionLevels
		RollObjectHitPoints RollObjectHitPointsCommand RollObjectHitPointsResult ObjectHitPointsRolled ObjectHitPointsRollSplitRequired NewRollObjectHitPoints
		ApplyDamageToObject ApplyDamageToObjectCommand ApplyDamageToObjectResult ObjectDamageApplied ObjectDamageRejected NewApplyDamageToObject`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, banned := range []string{"NotBurning", "Burning", "Breathing", "HoldingBreath", "Suffocating", "Doused", "Applied", "Rejected", "Found", "NotFound", "Succeeded",
		"Building", "Vehicle", "ObjectBuilding", "ObjectVehicle", "NewBuilding", "NewVehicle"} {
		if len(have[banned]) != 0 {
			t.Errorf("combat declares the name %s (FR-113, FR-114, FR-117)", banned)
		}
	}
}
