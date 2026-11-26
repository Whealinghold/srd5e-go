package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// limitedUseDomainFiles and limitedUseUseCaseFiles are the production files of monster-limited-use.md.
func limitedUseDomainFiles() []string {
	return []string{"combat/limited_use_part.go", "combat/limited_use_entry.go", "combat/limited_use_legendary.go", "combat/limited_use_event.go", "combat/limited_use_state.go"}
}

func limitedUseUseCaseFiles() []string { return []string{"combat/apply_monster_turn_start.go"} }

// Requirement: doc/specs/monster-limited-use.md FR-040, observability.md section 3 - the limited-use domain files import only character, content, dice and the standard library's slices, never telemetry or context, and name no signal constant; the use-case file adds telemetry (and srderr is reached through the instrumenter only).
// Path: unhappy (forbidden edge).
func TestLimitedUseFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	seen := 0
	for _, f := range combatPackage(t).Production() {
		allowed := []string{"character", "content", "dice"}
		domain := slices.Contains(limitedUseDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(limitedUseUseCaseFiles(), f.Path):
			allowed = append(allowed, "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) {
					t.Errorf("%s imports %s, which monster-limited-use.md does not allow", f.Path, p)
				}
				continue
			}
			switch {
			case imp.Path == "slices", !domain && imp.Path == "context":
			default:
				t.Errorf("%s imports %s, which monster-limited-use.md does not allow", f.Path, imp.Path)
			}
		}
		if !domain {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
					if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
						t.Errorf("%s names the signal constant %s: the domain is silent (FR-040)", f.Path, id.Name)
					}
				}
			}
			return true
		})
	}
	if want := len(limitedUseDomainFiles()) + len(limitedUseUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d limited-use files", seen, want)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-033 - exactly the six limited-use events embed the limited-use event base, all declared in limited_use_event.go.
// Path: unhappy (duplicate).
func TestLimitedUseEventFamilyHasExactlySixMembers(t *testing.T) {
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
					if id, ok := fld.Type.(*ast.Ident); ok && len(fld.Names) == 0 && id.Name == "monsterLimitedUseEventBase" {
						members = append(members, ts.Name.Name)
						if f.Path != "combat/limited_use_event.go" {
							t.Errorf("%s is declared in %s, want combat/limited_use_event.go", ts.Name.Name, f.Path)
						}
					}
				}
			}
			return true
		})
	}
	slices.Sort(members)
	want := []string{"LegendaryActionTaken", "LegendaryActionUsesRegained", "LimitedUsePartSpent", "LimitedUsesRestored", "MonsterLairStatusChanged", "MonsterRechargeRolled"}
	if !slices.Equal(members, want) {
		t.Errorf("limited-use event members = %v, want %v", members, want)
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-032, FR-033, FR-034, FR-050 - the closed sets have exactly the variants the spec lists.
// Path: unhappy (duplicate).
func TestLimitedUseClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isLimitedUseEntry":             {"PerDayEntry", "RechargeEntry", "RestRechargeEntry", "SpellUseEntry"},
		"isLegendaryTrack":              {"LegendaryNone", "LegendaryTracked"},
		"isLegendaryWindowRecord":       {"LegendaryWindowNoneTaken", "LegendaryWindowTaken"},
		"isLimitedUseSubject":           {"PartSubject", "SpellSubject"},
		"isLimitedUseSpendOutcome":      {"LimitedUseSpendRefused", "LimitedUseSpent"},
		"isLegendaryActionOutcome":      {"LegendaryActionAllowed", "LegendaryActionRefused"},
		"isApplyMonsterTurnStartResult": {"MonsterTurnStartApplied"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-035 - the use case constructor takes a Roller and options only.
// Path: unhappy (forbidden edge).
func TestApplyMonsterTurnStartConstructorTakesOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "NewApplyMonsterTurnStart" {
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
			if got := strings.Join(params, ","); got != "Roller,...Option" {
				t.Errorf("NewApplyMonsterTurnStart takes (%s), want (Roller,...Option)", got)
			}
		}
	}
	if found != 1 {
		t.Errorf("found %d NewApplyMonsterTurnStart, want 1", found)
	}
}

// Requirement: doc/specs/monster-limited-use.md section 7, plan-phase-4.md P11 - signals.go pins the operation value and span, the roll counter, the recharge outcome attribute and its two values, and the three result values at their catalogue values, and signals.md has a row for the span, the counter and the attribute.
// Path: unhappy (invalid input).
func TestLimitedUseSignalNamesAndCatalogue(t *testing.T) {
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
		"valueOperationApplyMonsterTurnStart": "apply_monster_turn_start", "spanApplyMonsterTurnStart": "srd5e.combat.apply_monster_turn_start",
		"metricMonsterRechargeRolls": "srd5e.combat.monster_recharge_rolls", "attrRechargeOutcome": "srd5e.combat.recharge_outcome",
		"valueRechargeRegained": "regained", "valueRechargeNotRegained": "not_regained",
		"valueResultTurnStartRecharged": "turn_start_recharged", "valueResultTurnStartNotRecharged": "turn_start_not_recharged", "valueResultTurnStartNoRolls": "turn_start_no_rolls",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	doc := catalogue(t)
	for _, name := range []string{"srd5e.combat.apply_monster_turn_start", "srd5e.combat.monster_recharge_rolls", "srd5e.combat.recharge_outcome"} {
		if !strings.Contains(doc, "| `"+name+"` |") {
			t.Errorf("signals.md has no row for %s", name)
		}
	}
}

// Requirement: doc/specs/monster-limited-use.md FR-041, section 6 (Exports) - package combat exports the limited-use types, rules and use case once each and no generic variant name.
// Path: happy.
func TestCombatExportsLimitedUseTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		LimitedUseState StartLimitedUse LairStatus InLair OutsideLair MonsterPartSection MonsterPartTraits MonsterPartActions MonsterPartBonusActions
		MonsterPartReactions MonsterPartLegendaryActions LimitedUsePartRef NewLimitedUsePartRef SpellUseRef NewSpellUseRef RechargeReadiness RechargeReady
		RechargeSpent RechargeRollResult RechargeRegained RechargeNotRegained LimitedUseAvailability UseAvailable UseExhausted UseNotLimited
		LimitedUseRefusal LimitedUseRefusedExhausted LimitedUseRefusedNotLimited LimitedUseSubject PartSubject SpellSubject
		LimitedUseEntry PerDayEntry RechargeEntry RestRechargeEntry SpellUseEntry LegendaryTrack LegendaryNone LegendaryTracked
		LegendaryActionWindow NewLegendaryActionWindow LegendaryWindowRecord LegendaryWindowNoneTaken LegendaryWindowTaken LegendaryActionRefusal
		LegendaryRefusedNoActions LegendaryRefusedOwnTurn LegendaryRefusedIncapacitated LegendaryRefusedWindowUsed LegendaryRefusedNoUses
		LimitedUseTransition LimitedUseSpendOutcome LimitedUseSpent LimitedUseSpendRefused LegendaryActionOutcome LegendaryActionAllowed LegendaryActionRefused
		MonsterLimitedUseEvent LimitedUsePartSpent MonsterRechargeRolled LimitedUsesRestored LegendaryActionTaken LegendaryActionUsesRegained MonsterLairStatusChanged
		ApplyMonsterTurnStart NewApplyMonsterTurnStart ApplyMonsterTurnStartCommand ApplyMonsterTurnStartResult MonsterTurnStartApplied`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Window", "Entry", "Track", "Subject", "Spent", "Refused", "Allowed", "Available"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-041)", generic)
		}
	}
}
