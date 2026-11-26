package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// revivalRestorationFiles are the production files of revival-and-restoration-spells.md: pure functions and one event family, no use case file.
func revivalRestorationFiles() []string {
	return []string{"combat/revival_spell.go", "combat/death_circumstances.go", "combat/revive_by_spell.go", "combat/revive_ordeal.go", "combat/caster_strain.go",
		"combat/hit_point_maximum_increase.go", "combat/increase_maximum.go", "combat/end_maximum_increase.go", "combat/restore_maximum.go", "combat/restoration_event.go",
		"combat/restoration_range_error.go", "combat/lesser_restoration.go", "combat/greater_restoration.go", "combat/restorative_healing.go",
		"combat/mass_heal_allotment.go", "combat/death_ward.go"}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-046, section 6 and section 7 - the revival and restoration files are silent domain code: they import of this module only
// character, content, dice, modifier and srderr (no telemetry, d20, toolbox, *test package), take no context and name no signal constant or Instrumenter.
// Path: unhappy (forbidden edge).
func TestRevivalRestorationFilesAreSilentDomainCode(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	allowed := []string{"character", "content", "dice", "modifier", "srderr"}
	seen := 0
	for _, f := range combatPackage(t).Production() {
		if !slices.Contains(revivalRestorationFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which revival-and-restoration-spells.md FR-046 does not allow", f.Path, p)
			}
			if imp.Path == "context" {
				t.Errorf("%s imports context, but the operations are pure functions with no ctx (section 6)", f.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr") || id.Name == "Instrumenter") {
				t.Errorf("%s names %s: the spec owns no signal and no use case (section 7)", f.Path, id.Name)
			}
			return true
		})
	}
	if seen != len(revivalRestorationFiles()) {
		t.Fatalf("found %d of %d revival and restoration files", seen, len(revivalRestorationFiles()))
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-046 - exactly HitPointMaximumIncreased, HitPointMaximumIncreaseEnded and DeathWardSpent embed the restoration event base, all declared in restoration_event.go.
// Path: unhappy (duplicate).
func TestRestorationEventFamilyHasExactlyThreeMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "restorationEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/restoration_event.go" {
							t.Errorf("%s is declared in %s, want combat/restoration_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	if want := []string{"DeathWardSpent", "HitPointMaximumIncreaseEnded", "HitPointMaximumIncreased"}; !slices.Equal(members, want) {
		t.Errorf("restoration event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/revival-and-restoration-spells.md FR-047, FR-048 - package combat declares the exported names of the spec once each and no generic variant name (Applied, Rejected, Succeeded alone).
// Path: unhappy (duplicate).
func TestCombatExportsRevivalRestorationTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		RevivalSpell Revivify RaiseDead Resurrection TrueResurrection RevivalSpellOf RevivalSpellFacts BodyState Intact MissingNonIntegralParts MissingIntegralParts NoBody
		DeathCircumstances NewDeathCircumstances MustDeathCircumstances PoisonsAtDeath RevivalIneligibility TimeWindowExceeded DiedOfOldAge WasUndead NoBodyToTouch
		IntegralPartsMissing NameNotSpoken ReviveBySpellCommand ReviveBySpell ReviveCreatureBySpellResult ReviveBySpellApplied ReviveBySpellIneligible ReviveBySpellNotRevived
		ReviveBySpellInvalidMaximum ReviveBySpellLimitExceeded ReviveOrdeal NewReviveOrdeal MustReviveOrdeal FreshOrdeal CasterStrain StrainNone Strained RestorationRangeError
		HitPointMaximumIncrease NewHitPointMaximumIncrease AidIncrease HeroesFeastIncrease HitPointMaximumIncreases NewHitPointMaximumIncreases IncreaseMaximum
		IncreaseHitPointMaximumResult IncreaseMaximumApplied IncreaseMaximumRejected IncreaseMaximumDuplicate IncreaseMaximumLimitExceeded EndMaximumIncrease EndHitPointMaximumIncreaseResult
		EndMaximumIncreaseApplied EndMaximumIncreaseNotPresent EndMaximumIncreaseRejected RestoreMaximum RestoreHitPointMaximumResult RestoreMaximumApplied RestoreMaximumRejected
		RestoreMaximumInvalidAmount ApplyLesserRestoration CastLesserRestorationResult LesserRestorationApplied LesserRestorationNothingToEnd LesserRestorationWrongKind
		GreaterRestorationChoice EndOneExhaustionLevel EndCharmed EndPetrified EndCurse EndAbilityScoreReduction EndHitPointMaximumReduction GreaterRestorationCommand
		ApplyGreaterRestoration CastGreaterRestorationResult GreaterRestorationApplied GreaterRestorationNothingToEnd GreaterRestorationNotACurse GreaterRestorationHostResolved
		GreaterRestorationRejected GreaterRestorationInvalid RestorativeSpell Heal PowerWordHeal MassHeal ApplyRestorativeHealing CastRestorativeHealingResult
		RestorativeHealingApplied RestorativeHealingRejected RestorativeHealingInvalidLevel MassHealAllotment NewMassHealAllotment DeathWard WardActive WardSpent
		DeathWardTrigger DropToZero InstantKill GuardDamageWithDeathWardResult GuardedDamageApplied GuardedDamageRefused GuardedDamageNotWarded ApplyDamageWithDeathWard
		NegateInstantKill RestorationEvent HitPointMaximumIncreased HitPointMaximumIncreaseEnded DeathWardSpent`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, name := range []string{"Applied", "Rejected", "Succeeded", "ReviveBySpellUseCase"} {
		if len(have[name]) != 0 {
			t.Errorf("combat declares %s, which is a generic variant name or belongs to another spec", name)
		}
	}
}
