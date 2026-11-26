package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"
)

// summonedDomainFiles and summonedUseCaseFiles are the production files of summoned-and-companion-creatures.md.
func summonedDomainFiles() []string {
	return []string{"combat/summoned_kind.go", "combat/summoned_rejection.go", "combat/summoned_stat_block.go", "combat/summoned_order.go", "combat/summoned_event.go",
		"combat/summoned_request.go", "combat/summoned_roster.go", "combat/summoned_summon.go", "combat/summoned_change.go"}
}

func summonedUseCaseFiles() []string {
	return []string{"combat/summon_creature_usecase.go", "combat/apply_summoned_change_usecase.go"}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-072, FR-074, FR-075, observability.md section 3 - the summoned domain files import only character, content, dice, slices and time (no telemetry, srderr, context, fmt, sync or randomness), name no signal constant and no roster, mount or turn use case; the use-case files add context and telemetry (and, for SummonCreature, srderr and strings), roll no die, hold no d20 Resolver and call no other use case.
// Path: unhappy (forbidden edge).
func TestSummonedFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		domain := slices.Contains(summonedDomainFiles(), f.Path)
		if !domain && !slices.Contains(summonedUseCaseFiles(), f.Path) {
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				allowed := []string{"content", "character", "dice"}
				if !domain {
					allowed = []string{"content", "telemetry", "srderr"}
				}
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which summoned-and-companion-creatures.md does not allow", f.Path, p)
				}
				continue
			}
			switch imp.Path {
			case "slices", "time":
			case "context", "strings":
				if domain {
					t.Errorf("%s (domain) imports %s (FR-075)", f.Path, imp.Path)
				}
			default:
				t.Errorf("%s imports %s, which summoned-and-companion-creatures.md does not allow (FR-075)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.Ident:
				if domain && (strings.HasPrefix(v.Name, "metric") || strings.HasPrefix(v.Name, "span") || strings.HasPrefix(v.Name, "attr") || strings.HasPrefix(v.Name, "valueOperation")) {
					t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, v.Name)
				}
				if slices.Contains([]string{"AddCombatantToEncounter", "RemoveCombatantFromEncounter", "RepositionCombatantInEncounter", "MountCreature", "AdvanceCombatTurn", "Resolver", "Roller"}, v.Name) {
					t.Errorf("%s names %s: the summoned use cases return roster instructions as data, roll nothing and call no other use case (FR-075, FR-077)", f.Path, v.Name)
				}
			case *ast.SelectorExpr:
				if !domain && v.Sel.Name == "Run" {
					t.Errorf("%s calls a Run method: no use case calls another (FR-075)", f.Path)
				}
			}
			return true
		})
	}
	if want := len(summonedDomainFiles()) + len(summonedUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d summoned files", seen, want)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-065, FR-069, FR-072, FR-073, section 5 - the closed sets have exactly the variants the spec lists (plus the four changes, the two rejections and the one cycle rejection recorded in its Implementation status), and every rejection type ends in Rejection.
// Path: unhappy (duplicate).
func TestSummonedClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isSummonRequest": strings.Fields(`AnimateObjectsRequest GiantInsectRequest SummonDragonRequest FindSteedRequest AnimateDeadRequest CreateUndeadRequest
			FindFamiliarRequest SimulacrumRequest ReassertControlRequest`),
		"isSummonedChange": strings.Fields(`IssueCommand ReportZeroHitPoints ReportSpellEnded ReportSummonerDied ReportLongRestFinished ElapseControlTime DismissFamiliar
			RecallFamiliar ShareFamiliarSenses DeliverTouch RepairSimulacrum ApplySteedLifeBond ReportTurnStarted ReportTurnEnded ReportTaskComplete RequestFamiliarAction`),
		"isSummonRejection": strings.Fields(`SummonCreatureIDInUseRejection SummonLinkCycleRejection LevelBelowSpellRejection NotNightRejection FormNotEligibleRejection
			SteedNotDisappearedRejection TargetNotEligibleRejection ObjectNotEligibleRejection OriginalNotEligibleRejection DuplicateTargetRejection TooManyCreaturesRejection
			CompositionNotAllowedRejection ControlLapsedRejection NothingToAnimateRejection NothingToReassertRejection ResistanceNotOfferedRejection SpaceUnavailableRejection
			SummonLimitExceededRejection CommandNotYourCreatureRejection CommandOutOfRangeRejection CannotGiveVerbalCommandRejection FamiliarCannotAttackRejection
			FamiliarAlreadyDismissedRejection FamiliarNotDismissedRejection SensesNotSharableRejection TouchNotDeliverableRejection SimulacrumTooFarRejection
			SimulacrumHealingRefusedRejection SimulacrumRepairAmountRejection SimulacrumRepairNeedsLongRestRejection LifeBondOutOfReachRejection
			TelepathyOnlyWithSummonerRejection`),
		"isSummonedTransition":        {"SummonedApplied", "SummonedRejected", "SummonedUnchanged"},
		"isSummonedOrder":             {"GeneralOrder", "SpecificOrder"},
		"isSummonedTurnPlan":          {"TurnPlanDefaultDodge", "TurnPlanOrdered"},
		"isSteedChoice":               {"SteedChoiceDifferent", "SteedChoiceSame"},
		"isCommandChannel":            {"ChannelBonusActionMental", "ChannelObeysIndependently", "ChannelVerbalNoAction"},
		"isSummonedRosterAction":      {"SummonedRosterJoin", "SummonedRosterLeave"},
		"isSummonCreatureResult":      {"SummonCreatureApplied", "SummonCreatureContentMissing", "SummonCreatureRejected"},
		"isApplySummonedChangeResult": {"ApplySummonedChangeApplied", "ApplySummonedChangeRejected", "ApplySummonedChangeUnchanged"},
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
	for _, name := range markerReceivers(t, "isSummonRejection") {
		if !strings.HasSuffix(name, "Rejection") {
			t.Errorf("%s does not end in Rejection (FR-073)", name)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-069 - exactly the twelve events of the spec embed the summoned event base, all declared once in summoned_event.go; none implements Event on its own.
// Path: unhappy (duplicate).
func TestSummonedEventFamilyHasExactlyTwelveMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "summonEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/summoned_event.go" {
							t.Errorf("%s is declared in %s, want combat/summoned_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := strings.Fields(`SummonedCreatureJoined SummonedCreatureCommanded SummonedCreatureDeparted SharedResistanceEnded SummonControlReasserted SummonControlLapsed
		FamiliarFormChanged FamiliarDismissedTemporarily FamiliarReappeared FamiliarSightLinkStarted FamiliarSightLinkEnded SimulacrumRepaired`)
	slices.Sort(want)
	if !slices.Equal(members, want) {
		t.Errorf("summoned event members = %v, want %v", members, want)
	}
	if got := markerReceivers(t, "summonEvent"); !slices.Equal(got, []string{"summonEventBase"}) {
		t.Errorf("summonEvent is implemented by %v, want only the base", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-072, FR-075, section 6 (Exports) - package combat exports the two use cases, their commands and results, the roster, the stat block constructors, the requests, the changes and the events once each, with no generic name, and its package comment names each of them.
// Path: happy.
func TestCombatExportsSummonedTypesAndDocumentsThem(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		SummonCreature NewSummonCreature SummonCreatureCommand NewSummonCreatureCommand SummonCreatureResult SummonCreatureApplied SummonCreatureRejected
		SummonCreatureContentMissing ApplySummonedChange NewApplySummonedChange ApplySummonedChangeCommand NewApplySummonedChangeCommand ApplySummonedChangeResult
		ApplySummonedChangeApplied ApplySummonedChangeRejected ApplySummonedChangeUnchanged SummonedRoster SummonedLink SummonedCreatureSpec SummonedTransition
		SummonedApplied SummonedRejected SummonedUnchanged SummonRequest SummonScene SummonedChange SummonRejection SummonEvent SummonKind PlacementRule
		CommandChannel SummonEndCause FamiliarOrigin SteedType SteedMode InsectForm SummonedStatBlock SummonedAttack SummonerSpellFacts NewSummonerSpellFacts
		NewAnimatedObjectStatBlock NewGiantInsectStatBlock NewDraconicSpiritStatBlock NewOtherworldlySteedStatBlock NewSimulacrumStatBlock SimulacrumOriginal
		SummonedOrder SummonedTurnPlan ControlWindow FamiliarAttackGrant SharedResistanceGrant SummonedRosterAction SummonedRosterJoin SummonedRosterLeave
		BonusActionGrantFor SteedModeFor OverflowDamage MonsterIDsNeeded ContentSource FamiliarState
		SummonedCreatureJoined SummonedCreatureCommanded SummonedCreatureDeparted SharedResistanceEnded SummonControlReasserted SummonControlLapsed
		FamiliarFormChanged FamiliarDismissedTemporarily FamiliarReappeared FamiliarSightLinkStarted FamiliarSightLinkEnded SimulacrumRepaired
		SummonCreatureIDInUseRejection LevelBelowSpellRejection TooManyCreaturesRejection FamiliarCannotAttackRejection SummonLimitExceededRejection`)
	enumMembers := strings.Fields(`NewSummonCreature NewApplySummonedChange NewSummonerSpellFacts`)
	have := declared(combatPackage(t))
	doc := ""
	for _, f := range combatPackage(t).Production() {
		if f.Path == "combat/doc.go" && f.AST.Doc != nil {
			doc = f.AST.Doc.Text()
		}
	}
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
		if !strings.Contains(doc, "["+name+"]") && !strings.HasPrefix(name, "New") && !slices.Contains(enumMembers, name) {
			t.Errorf("the package comment of combat/doc.go does not name [%s] (FR-072)", name)
		}
	}
	for _, generic := range []string{"Applied", "Rejected", "Unchanged", "ContentMissing"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (architecture.md A7)", generic)
		}
	}
}
