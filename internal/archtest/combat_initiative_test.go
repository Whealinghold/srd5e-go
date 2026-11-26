package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// turnDomainFiles and turnUseCaseFiles are the production files of initiative-and-actions.md.
func turnDomainFiles() []string {
	return []string{"combat/initiative_count.go", "combat/turn_order.go", "combat/movement_speeds.go", "combat/movement_segment.go",
		"combat/turn_actions.go", "combat/ready.go", "combat/turn_rejection.go", "combat/turn_event.go", "combat/turn_resources.go",
		"combat/turn_tracker.go", "combat/turn_tracker_ops.go"}
}

func turnUseCaseFiles() []string {
	return []string{"combat/turn_usecase.go", "combat/turn_usecases.go", "combat/roll_initiative.go", "combat/establish_initiative_order.go", "combat/swap_initiative.go"}
}

// Requirement: doc/specs/initiative-and-actions.md FR-094, FR-097, observability.md section 3 - the domain files import only the kernel, content and character (no telemetry, context or signal constant); the use-case files add telemetry, the dice port and context; neither imports a downstream context or a goroutine.
// Path: unhappy (forbidden edge).
func TestTurnFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "character", "content", "d20", "dice", "modifier", "srderr"}
		domain := slices.Contains(turnDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(turnUseCaseFiles(), f.Path):
			allowed = append(allowed, "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which initiative-and-actions.md does not allow", f.Path, p)
			}
			if domain && (strings.HasSuffix(imp.Path, "/telemetry") || imp.Path == "context" || strings.HasSuffix(imp.Path, "/dice")) {
				t.Errorf("%s imports %s: the turn domain is silent, pure and rolls nothing (FR-094)", f.Path, imp.Path)
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.GoStmt:
				t.Errorf("%s starts a goroutine (FR-094)", f.Path)
			case *ast.Ident:
				if domain && (strings.HasPrefix(v.Name, "metric") || strings.HasPrefix(v.Name, "span") || strings.HasPrefix(v.Name, "attr") || strings.HasPrefix(v.Name, "msg")) {
					t.Errorf("%s names the signal constant %s: the domain is silent (FR-094)", f.Path, v.Name)
				}
			}
			return true
		})
	}
	if want := len(turnDomainFiles()) + len(turnUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d turn files", seen, want)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-123 - TurnTracker has no exported constructor, no exported field, and only turn_tracker.go builds a TurnTracker literal.
// Path: unhappy (invalid input).
func TestTurnTrackerHasNoExportedConstructor(t *testing.T) {
	t.Parallel()
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.FuncDecl:
				if v.Recv == nil && v.Type.Results != nil && v.Name.IsExported() {
					for _, r := range v.Type.Results.List {
						if id, ok := r.Type.(*ast.Ident); ok && id.Name == "TurnTracker" {
							t.Errorf("%s: %s returns a TurnTracker; a tracker comes only from EstablishInitiativeOrder (FR-123)", f.Path, v.Name.Name)
						}
					}
				}
			case *ast.CompositeLit:
				if id, ok := v.Type.(*ast.Ident); ok && id.Name == "TurnTracker" && f.Path != "combat/turn_tracker.go" {
					t.Errorf("%s builds a TurnTracker literal (FR-123)", f.Path)
				}
			case *ast.TypeSpec:
				if st, ok := v.Type.(*ast.StructType); ok && v.Name.Name == "TurnTracker" {
					for _, fld := range st.Fields.List {
						for _, name := range fld.Names {
							if name.IsExported() {
								t.Errorf("TurnTracker has the exported field %s (FR-123)", name.Name)
							}
						}
					}
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-044, FR-126, FR-140, FR-150, FR-151, FR-161 - the closed sets of the spec have exactly the variants it lists.
// Path: unhappy (duplicate).
func TestTurnClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isActionChoice":                   {"DashChoice", "FeatureActionChoice", "ImprovisedActionChoice", "ReadyChoice", "SimpleActionChoice"},
		"isReadyableChoice":                {"DashChoice", "FeatureActionChoice", "ImprovisedActionChoice", "SimpleActionChoice"},
		"isGrantedAction":                  {"GrantsDash", "GrantsSimpleAction", "NoStandardAction"},
		"isReactionChoice":                 {"FeatureReaction", "OpportunityAttackReaction", "ReadiedActionReaction", "SpellReaction"},
		"isReactionRecord":                 {"FeatureReactionRecord", "OpportunityAttackRecord", "ReadiedActionRecord", "SpellReactionRecord"},
		"isReadiedOutcome":                 {"ReadiedChoiceOutcome", "ReadiedMoveOutcome"},
		"isReadiedResponse":                {"ReadiedActionResponse", "ReadiedMoveResponse"},
		"isHeldSpellInput":                 {"HoldsNoSpell", "HoldsSpell"},
		"isBonusActionSource":              {"BonusActionFromFeature", "BonusActionFromItem"},
		"isBonusActionEntitlement":         {"EntitledToBonusAction", "NotEntitledToBonusAction"},
		"isPostureChange":                  {"DropProneChange", "StandUpChange"},
		"isAdditionalActionScope":          {"AnyActionBut", "OnlyActions"},
		"isActivationCost":                 {"ActivatesByAction", "ActivatesByBonusAction", "ActivationCostDescribed"},
		"isTrackerStep":                    {"TrackerStepApplied", "TrackerStepRejected"},
		"isInitiativeBreakdown":            {"RolledInitiative", "ScoredInitiative"},
		"isRollInitiativeResult":           {"RollInitiativeRolled"},
		"isEstablishInitiativeOrderResult": {"InitiativeOrderEstablished", "InitiativeTieBreakRejected", "InitiativeTiesUnresolved"},
		"isSwapInitiativeResult":           {"SwapInitiativeApplied", "SwapInitiativeRejected"},
		"isTurnRejection": strings.Fields(`TurnActionAlreadyTakenRejection TurnAdditionalActionAlreadyGrantedRejection TurnAllyNotWillingRejection
			TurnBlockedByCreatureRejection TurnBonusActionAlreadyTakenRejection TurnEndsInOccupiedSpaceRejection TurnFeatUseLimitedRejection
			TurnIncapacitatedRejection TurnInsufficientMovementRejection TurnNeedsUtilizeActionRejection TurnNoBonusActionGrantedRejection
			TurnNoMatchingReadiedActionRejection TurnNoMovementRemainingRejection TurnNotActiveCombatantRejection TurnProvokerDisengagedRejection
			TurnProvokerNotSeenRejection TurnReactionAlreadyTakenRejection TurnSpeedIsZeroRejection TurnSpeedUnavailableRejection TurnUnknownCombatantRejection`),
	}
	for _, uc := range strings.Fields("AdvanceTrackerTurn UpdateMovementSpeeds TakeAction TakeBonusAction TakeReaction Move InteractWithObject ChangePosture GrantAdditionalAction RecordFeatUse") {
		want["is"+uc+"Result"] = []string{uc + "Applied", uc + "Rejected"}
	}
	for marker, names := range want {
		slices.Sort(names)
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-125, FR-128 - exactly the sixteen events embed turnEventBase, all declared in turn_event.go, and none implements Event on its own.
// Path: unhappy (duplicate).
func TestTurnEventFamilyHasExactlySixteenMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "turnEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/turn_event.go" {
							t.Errorf("%s is declared in %s, want combat/turn_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := strings.Fields(`TrackerRoundStarted TrackerTurnStarted TrackerTurnEnded AdditionalActionGranted FeatUseRecorded InitiativeSwapped ActionTaken
		BonusActionUsed ReactionTaken MovementSpent ObjectInteracted DroppedProne StoodUp FellProneInSharedSpace ReadiedActionExpired MovementSpeedsUpdated`)
	slices.Sort(want)
	if !slices.Equal(members, want) {
		t.Errorf("turn event members = %v, want %v", members, want)
	}
	if got := markerReceivers(t, "event"); slices.ContainsFunc(want, func(n string) bool { return slices.Contains(got, n) }) {
		t.Errorf("a turn event implements Event directly: %v", got)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-092, FR-126, FR-103 - package combat declares no creature-size type, exports none of Applied, Rejected, Refused or TurnStep, and defines no per-turn counter.
// Path: unhappy (name collision).
func TestTurnNamesAndNoPerTurnCounters(t *testing.T) {
	t.Parallel()
	d := declared(combatPackage(t))
	for _, banned := range []string{"CreatureSize", "Applied", "Rejected", "Refused", "TurnStep", "NewTurnTracker"} {
		if got := d[banned]; len(got) != 0 {
			t.Errorf("combat declares %s in %v (FR-092, FR-126)", banned, got)
		}
	}
	if got := d["TrackerStep"]; len(got) != 1 || got[0] != "combat/turn_tracker.go" {
		t.Errorf("TrackerStep is declared in %v, want combat/turn_tracker.go", got)
	}
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				s, _ := strconv.Unquote(lit.Value)
				for _, banned := range []string{"turns_started", "rounds_started", "turn_actions"} {
					if strings.Contains(s, banned) {
						t.Errorf("%s contains %q: turns, rounds and actions are counted by srd5e.combat.events (FR-103)", f.Path, banned)
					}
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-097, FR-127 - RollInitiative takes a Roller and options; every other turn use case takes options only.
// Path: unhappy (forbidden edge).
func TestTurnConstructorsTakeOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	want := map[string]string{"NewRollInitiative": "Roller,...Option"}
	for _, uc := range strings.Fields("EstablishInitiativeOrder AdvanceTrackerTurn UpdateMovementSpeeds TakeAction TakeBonusAction TakeReaction Move InteractWithObject ChangePosture GrantAdditionalAction RecordFeatUse SwapInitiative") {
		want["New"+uc] = "...Option"
	}
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || want[fn.Name.Name] == "" {
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
			if got := strings.Join(params, ","); got != want[fn.Name.Name] {
				t.Errorf("%s takes (%s), want (%s)", fn.Name.Name, got, want[fn.Name.Name])
			}
		}
	}
	if found != len(want) {
		t.Errorf("found %d of %d constructors", found, len(want))
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-149 - the Run of each tracker-backed use case calls exactly one method of cmd.tracker and holds no branch other than the error check.
// Path: happy.
func TestTrackerUseCasesAddNoRule(t *testing.T) {
	t.Parallel()
	checked := 0
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/turn_usecases.go" {
			continue
		}
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != "Run" {
				continue
			}
			checked++
			calls, branches := 0, 0
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.SelectorExpr:
					if x, ok := v.X.(*ast.SelectorExpr); ok && x.Sel.Name == "tracker" {
						if id, ok := x.X.(*ast.Ident); ok && id.Name == "cmd" {
							calls++
						}
					}
				case *ast.IfStmt:
					branches++
				case *ast.SwitchStmt, *ast.ForStmt, *ast.RangeStmt:
					branches += 10
				}
				return true
			})
			if calls != 1 || branches != 1 {
				t.Errorf("Run of %s has %d tracker reads and %d branches, want one operation and the error check only (FR-149)", recvName(fn), calls, branches)
			}
		}
	}
	if checked != 10 {
		t.Errorf("checked %d Run methods, want 10", checked)
	}
}

func recvName(fn *ast.FuncDecl) string {
	if id, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
		return id.Name
	}
	return "?"
}

// Requirement: doc/specs/initiative-and-actions.md FR-099, FR-108, plan-phase-4.md P11 - signals.go pins the thirteen operation values and spans, the four metrics, the three attribute keys and the new result values at their catalogue values, and signals.md has a row for each metric, attribute and span.
// Path: unhappy (invalid input).
func TestTurnSignalNamesAndCatalogue(t *testing.T) {
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
		"metricInitiativeCounts": "srd5e.combat.initiative_counts", "metricInitiativeParticipants": "srd5e.combat.initiative_participants",
		"metricInitiativeTies": "srd5e.combat.initiative_ties", "metricTurnMovementSpent": "srd5e.combat.turn_movement_spent",
		"attrInitiativeMethod": "srd5e.combat.initiative_method", "attrTieOutcome": "srd5e.combat.tie_outcome", "attrMovementMode": "srd5e.combat.movement_mode",
		"valueMovementModePosture": "posture", "valueResultEstablished": "established", "valueResultTiesUnresolved": "ties_unresolved",
		"valueResultTieBreakRejected": "tie_break_rejected",
	}
	doc := catalogue(t)
	for _, op := range strings.Fields(`RollInitiative:roll_initiative EstablishInitiativeOrder:establish_initiative_order AdvanceTrackerTurn:advance_tracker_turn
		UpdateMovementSpeeds:update_movement_speeds TakeAction:take_action TakeBonusAction:take_bonus_action TakeReaction:take_reaction Move:move
		InteractWithObject:interact_with_object ChangePosture:change_posture GrantAdditionalAction:grant_additional_action RecordFeatUse:record_feat_use
		SwapInitiative:swap_initiative`) {
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
		if strings.HasPrefix(name, "metric") || strings.HasPrefix(name, "attr") {
			if !strings.Contains(doc, "| `"+value+"` |") {
				t.Errorf("signals.md has no row for %s", value)
			}
		}
	}
}
