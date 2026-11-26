package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// monsterTraitFiles are the production files of monster-traits.md.
func monsterTraitFiles() []string {
	var out []string
	for _, n := range []string{"trait", "recognise", "event", "legendary_resistance", "circumstances", "undead_fortitude", "regeneration", "death_burst", "situational", "parry"} {
		out = append(out, "combat/monster_traits_"+n+".go")
	}
	return out
}

// Requirement: doc/specs/monster-traits.md FR-092, observability.md section 3 - the monster_traits_*.go files import only ability, content, d20, modifier and the standard library's slices, strconv and strings; never telemetry, context, dice, character or srderr, and name no signal constant.
// Path: unhappy (forbidden edge).
func TestMonsterTraitFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	allowed := []string{"ability", "content", "d20", "modifier"}
	std := []string{"slices", "strconv", "strings"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(monsterTraitFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which monster-traits.md does not allow", f.Path, p)
				}
				continue
			}
			if !slices.Contains(std, imp.Path) {
				t.Errorf("%s imports %s, which monster-traits.md does not allow", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the spec is silent (FR-092)", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if seen != len(monsterTraitFiles()) {
		t.Fatalf("found %d of %d monster trait files", seen, len(monsterTraitFiles()))
	}
}

// Requirement: doc/specs/monster-traits.md FR-090 - exactly the five trait events embed the family base, all declared in monster_traits_event.go.
// Path: unhappy (duplicate).
func TestMonsterTraitEventFamilyHasExactlyFiveMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "monsterTraitEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/monster_traits_event.go" {
							t.Errorf("%s is declared in %s, want combat/monster_traits_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"LegendaryResistanceUsed", "ParryUsed", "RegenerationDeathDeferred", "RegenerationSuppressed", "UndeadFortitudeSaved"}
	if !slices.Equal(members, want) {
		t.Errorf("trait event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/monster-traits.md section 5 - the closed sets have exactly the variants the spec lists; every trait embeds traitBase.
// Path: unhappy (duplicate).
func TestMonsterTraitClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isSaveFailure":                {"ForcedSaveFailure", "RolledSaveFailure"},
		"isLegendaryResistanceOutcome": {"LegendaryResistanceApplied", "LegendaryResistanceRefused"},
		"isUndeadFortitudeCheck":       {"UndeadFortitudeNotTriggered", "UndeadFortitudeSaveRequired"},
		"isUndeadFortitudeSettlement":  {"UndeadFortitudeFell", "UndeadFortitudeSurvived"},
		"isRegenerationTurn":           {"RegenerationHealed", "RegenerationLetDie", "RegenerationSkipped"},
		"isRegenerationDeferral":       {"RegenerationDeferred", "RegenerationNotDeferred"},
		"isDeathBurstResult":           {"DeathBurstLimitExceeded", "DeathBurstNotTriggered", "DeathBurstTriggered"},
		"isSwarmChange":                {"SwarmHealing", "SwarmMaximumReduction", "SwarmTemporaryHitPoints"},
		"isParryOffer":                 {"ParryAvailable", "ParryUnavailable"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
	var traits []string
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					for _, fld := range st.Fields.List {
						if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "traitBase" {
							traits = append(traits, ts.Name.Name)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(traits)
	wantTraits := []string{"AmorphousTrait", "DeathBurstTrait", "FlybyTrait", "LegendaryResistanceTrait", "MagicResistanceTrait", "PackTacticsTrait", "ParryTrait",
		"RegenerationTrait", "SiegeMonsterTrait", "SunlightPenaltyTrait", "SwarmTrait", "UndeadFortitudeTrait"}
	if !slices.Equal(traits, wantTraits) {
		t.Errorf("traits = %v, want %v", traits, wantTraits)
	}
}

// Requirement: doc/specs/monster-traits.md FR-093 - package combat exports the trait types, rules and events once each and no generic variant name.
// Path: happy.
func TestCombatExportsMonsterTraitTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		MonsterTrait LegendaryResistanceTrait MagicResistanceTrait UndeadFortitudeTrait RegenerationTrait DeathBurstTrait SwarmTrait PackTacticsTrait SunlightPenaltyTrait
		FlybyTrait AmorphousTrait SiegeMonsterTrait ParryTrait NewLegendaryResistanceTrait NewMagicResistanceTrait NewUndeadFortitudeTrait NewRegenerationTrait NewDeathBurstTrait
		NewSwarmTrait NewPackTacticsTrait NewSunlightPenaltyTrait NewFlybyTrait NewAmorphousTrait NewSiegeMonsterTrait NewParryTrait SunlightPenaltyScope
		SunlightPenaltyAbilityChecksAndAttackRolls SunlightPenaltyAllD20Tests TraitRejection TraitSectionMismatch TraitTextMismatch TraitUsageMismatch TraitParameterOutOfRange
		TraitUnknownDamageType TraitNoSavingThrow UnrecognisedTrait RecognisedTraits RecogniseTraits SaveFailure RolledSaveFailure ForcedSaveFailure LegendaryResistanceRefusal
		LegendaryResistanceOutcome LegendaryResistanceApplied LegendaryResistanceRefused UseLegendaryResistance SaveSourceKind SpellOrMagicalEffect OtherSaveSource
		MagicResistanceCircumstances UndeadFortitudeFacts NewUndeadFortitudeFacts UndeadFortitudeExemption UndeadFortitudeCheck UndeadFortitudeNotTriggered
		UndeadFortitudeSaveRequired AssessUndeadFortitude UndeadFortitudeSettlement UndeadFortitudeSurvived UndeadFortitudeFell SettleUndeadFortitude RegenerationState
		NewRegenerationState RegenerationStep RegenerationSkipReason RegenerationTurn RegenerationHealed RegenerationSkipped RegenerationLetDie RegenerationNotDeferredReason
		RegenerationDeferral RegenerationDeferred RegenerationNotDeferred DeferRegenerationDeath DeathBurstResult DeathBurstTriggered DeathBurstNotTriggered TriggerDeathBurst
		SwarmRules SwarmChange SwarmHealing SwarmTemporaryHitPoints SwarmMaximumReduction SwarmRefusal CheckSwarmChange AllyFact PackTacticsCircumstances
		SunlightPenaltyCircumstances FlybyExemptsFromOpportunityAttack DamageRecipient DamageRecipientCreature DamageRecipientObjectOrStructure ApplySiegeMonster ParryFacts
		ParryUnavailableReason ParryOffer ParryAvailable ParryUnavailable AssessParry ParryResponse RespondToParry MonsterTraitEvent LegendaryResistanceUsed UndeadFortitudeSaved
		RegenerationSuppressed RegenerationDeathDeferred ParryUsed`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Refused", "Applied", "Unavailable", "Available", "Triggered", "Healed", "Skipped", "Deferred"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-093)", generic)
		}
	}
}
