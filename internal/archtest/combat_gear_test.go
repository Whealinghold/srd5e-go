package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// gearDomainFiles and gearUseCaseFiles are the production files of deployed-and-restraining-gear.md.
func gearDomainFiles() []string {
	return []string{"combat/gear_kind.go", "combat/gear_problem.go", "combat/gear_event.go", "combat/gear_deployed.go", "combat/gear_entry.go",
		"combat/gear_hold.go", "combat/gear_net.go", "combat/gear_binding.go", "combat/gear_release.go"}
}

func gearUseCaseFiles() []string {
	return []string{"combat/gear_usecase.go", "combat/deploy_gear.go", "combat/maintain_deployed_gear.go", "combat/resolve_gear_entry.go",
		"combat/throw_net.go", "combat/bind_creature.go", "combat/release_from_hold.go"}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-069, observability.md section 3 - the gear domain files import only the kernel, content, d20 and the standard library's slices and strings and name no signal constant (no logging, metric or span call, no observability import); the use-case files add dice, modifier, srderr and telemetry.
// Path: unhappy (forbidden edge).
func TestGearFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"ability", "content", "d20", "modifier", "srderr"}
		domain := slices.Contains(gearDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(gearUseCaseFiles(), f.Path):
			allowed = append(allowed, "dice", "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok && !slices.Contains(allowed, p) {
				t.Errorf("%s imports %s, which deployed-and-restraining-gear.md does not allow", f.Path, p)
			}
			if domain && (strings.HasSuffix(imp.Path, "/telemetry") || strings.HasSuffix(imp.Path, "/dice") || strings.HasSuffix(imp.Path, "/context")) {
				t.Errorf("%s imports %s: the gear domain is silent, with no dice, port or telemetry (FR-069)", f.Path, imp.Path)
			}
		}
		if !domain {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (strings.HasPrefix(id.Name, "metric") || strings.HasPrefix(id.Name, "span") || strings.HasPrefix(id.Name, "attr")) {
				t.Errorf("%s names the signal constant %s: the domain is silent (FR-069)", f.Path, id.Name)
			}
			return true
		})
	}
	if want := len(gearDomainFiles()) + len(gearUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d gear files", seen, want)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-059, FR-060, FR-065 - the closed sets have exactly the variants the spec lists: the six results, the effects, outcomes, assessments, states, maintenance, release actors, transitions and tests.
// Path: unhappy (duplicate).
func TestGearClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isDeployGearResult":           {"DeployGearInvalid", "DeployGearPlaced", "DeployGearRejected"},
		"isMaintainDeployedGearResult": {"MaintainDeployedGearApplied", "MaintainDeployedGearRejected"},
		"isResolveGearEntryResult": {"ResolveGearEntryDamaged", "ResolveGearEntryFailed", "ResolveGearEntryInvalid", "ResolveGearEntryRejected",
			"ResolveGearEntrySaved"},
		"isThrowNetResult":          {"ThrowNetAutoSucceeded", "ThrowNetHeld", "ThrowNetRejected", "ThrowNetSaved"},
		"isBindCreatureResult":      {"BindCreatureBound", "BindCreatureFailed", "BindCreatureRejected"},
		"isReleaseFromHoldResult":   {"ReleaseFromHoldHeld", "ReleaseFromHoldRejected", "ReleaseFromHoldReleased"},
		"isGearEffect":              {"GearCaughtEffect", "GearConditionEffect", "GearDamageEffect", "GearSpeedZeroEffect"},
		"isGearEntryOutcome":        {"GearEntryDamaged", "GearEntryFailed", "GearEntrySaved"},
		"isGearEntryAssessment":     {"GearEntryDamageDue", "GearEntryRefused", "GearEntryRollRequired"},
		"isDeployedGearState":       {"GearArmed", "GearInForce", "GearSprung", "OilBurning", "OilUnlit"},
		"isPouredOilState":          {"OilBurning", "OilUnlit"},
		"isGearMaintenance":         {"ExpireBurningOil", "LightOil", "RecoverGear", "ResetHuntingTrap"},
		"isReleaseActor":            {"ActorIsHeld", "ActorIsHelper"},
		"isDeployedGearsTransition": {"DeployedGearsApplied", "DeployedGearsRejected"},
		"isGearHoldsTransition":     {"GearHoldsApplied", "GearHoldsRejected"},
		"isGearTest":                {"GearTestForced", "GearTestRolled"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-063 - exactly the nine gear events embed the gear event base, all declared in gear_event.go.
// Path: unhappy (duplicate).
func TestGearEventFamilyHasExactlyNineMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "gearEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/gear_event.go" {
							t.Errorf("%s is declared in %s, want combat/gear_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"DeployedGearPlaced", "DeployedGearRecovered", "GearHoldEnded", "GearHoldRestraintChanged", "GearHoldStarted", "HuntingTrapReset",
		"HuntingTrapSprung", "PouredOilBurnedOut", "PouredOilLit"}
	if !slices.Equal(members, want) {
		t.Errorf("gear event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-061 - the four rolling gear use case constructors take a Roller and options only and the two others options only.
// Path: unhappy (forbidden edge).
func TestGearConstructorsTakeOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	want := map[string]string{"NewResolveGearEntry": "Roller,...Option", "NewThrowNet": "Roller,...Option", "NewBindCreature": "Roller,...Option",
		"NewReleaseFromHold": "Roller,...Option", "NewDeployGear": "...Option", "NewMaintainDeployedGear": "...Option"}
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

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-064, FR-066, FR-067, FR-072, plan-phase-4.md P11 - signals.go pins the six operation values and spans, the three metrics and the three attribute keys at their catalogue values, and signals.md has a row for each metric and span.
// Path: unhappy (invalid input).
func TestGearSignalNamesAndCatalogue(t *testing.T) {
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
		"metricGearEntries": "srd5e.combat.gear.entries", "metricGearHoldEnded": "srd5e.combat.gear.hold_ended", "metricGearHoldsInForce": "srd5e.combat.gear.holds_in_force",
		"attrGearKind": "srd5e.combat.gear.kind", "attrGearEntry": "srd5e.combat.gear.entry", "attrGearEndCause": "srd5e.combat.gear.end_cause",
		"valueGearEntrySaveFailed": "save_failed", "valueGearEntrySavePassed": "save_passed", "valueGearEntryDamaged": "damaged",
		"valueGearEntryAlreadyTriggered": "already_triggered", "valueGearEntryNotActive": "not_active", "valueGearEntryNotApplicable": "not_applicable",
		"valueResultPlaced": "placed", "valueResultSaved": "saved", "valueResultAutoSucceeded": "auto_succeeded", "valueResultBound": "bound", "valueResultReleased": "released",
	}
	doc := catalogue(t)
	for _, op := range strings.Fields("DeployGear:deploy_gear MaintainDeployedGear:maintain_deployed_gear ResolveGearEntry:resolve_gear_entry ThrowNet:throw_net BindCreature:bind_creature ReleaseFromHold:release_from_hold") {
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
}

// Requirement: doc/specs/deployed-and-restraining-gear.md FR-070, section 6 (Exports) - package combat exports the gear types and use cases once each and no generic variant name.
// Path: happy.
func TestCombatExportsGearTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		DeployedGearKind HoldKind BindingKind GroundLevel AreaTrigger HoldRestraint HoldReleaseMethod HoldEndCause Willingness ThrowerSight DeployReach ReachWithin ReachUnstated
		DeployedGearProfile DeployedGearProfileFor HoldProfile HoldProfileFor HoldCheck DeployedGearID NewDeployedGearID MustDeployedGearID HoldID NewHoldID MustHoldID TurnStamp NewTurnStamp MustTurnStamp
		DeployedGearState PouredOilState GearInForce GearArmed GearSprung OilUnlit OilBurning DeployedGear DeployedGears NewDeployedGears DeployedGearsTransition DeployedGearsApplied DeployedGearsRejected
		EntryRecord EntryLedger NewEntryLedger GearEffect GearDamageEffect GearConditionEffect GearSpeedZeroEffect GearCaughtEffect GearEntryOutcome GearEntrySaved GearEntryFailed GearEntryDamaged
		GearEntryAssessment GearEntryRollRequired GearEntryDamageDue GearEntryRefused AreaEntryRules DCOverride NoDCOverride DCOverrideOf GearHoldInput GearHold NewGearHold GearHolds NewGearHolds
		GearHoldsTransition GearHoldsApplied GearHoldsRejected NetProfile NetRules NetThrowRangeFeet BindingScene BindingRules ReleaseActor ActorIsHeld ActorIsHelper ReleaseRequest HoldReleasePlan HoldRules
		GearRejection GearProblem GearError GearEvent DeployedGearPlaced DeployedGearRecovered PouredOilLit PouredOilBurnedOut HuntingTrapSprung HuntingTrapReset GearHoldStarted GearHoldRestraintChanged GearHoldEnded
		GearTest GearTestRolled GearTestForced GearMaintenance LightOil RecoverGear ResetHuntingTrap ExpireBurningOil
		DeployGear NewDeployGear DeployGearCommand NewDeployGearCommand DeployGearResult DeployGearPlaced DeployGearRejected DeployGearInvalid
		MaintainDeployedGear NewMaintainDeployedGear MaintainDeployedGearCommand NewMaintainDeployedGearCommand MaintainDeployedGearResult MaintainDeployedGearApplied MaintainDeployedGearRejected
		ResolveGearEntry NewResolveGearEntry ResolveGearEntryInput ResolveGearEntryCommand NewResolveGearEntryCommand ResolveGearEntryResult ResolveGearEntrySaved ResolveGearEntryFailed ResolveGearEntryDamaged ResolveGearEntryRejected ResolveGearEntryInvalid
		ThrowNet NewThrowNet ThrowNetInput ThrowNetCommand NewThrowNetCommand ThrowNetResult ThrowNetHeld ThrowNetSaved ThrowNetAutoSucceeded ThrowNetRejected
		BindCreature NewBindCreature BindCreatureInput BindCreatureCommand NewBindCreatureCommand BindCreatureResult BindCreatureBound BindCreatureFailed BindCreatureRejected
		ReleaseFromHold NewReleaseFromHold ReleaseFromHoldInput ReleaseFromHoldCommand NewReleaseFromHoldCommand ReleaseFromHoldResult ReleaseFromHoldReleased ReleaseFromHoldHeld ReleaseFromHoldRejected`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Net", "Placed", "Saved", "Held", "Bound", "Released", "Failed"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-070)", generic)
		}
	}
}
