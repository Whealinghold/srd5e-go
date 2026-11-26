package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// monsterSetupDomainFiles and monsterSetupUseCaseFiles are the production files of monster-combatant-setup.md.
func monsterSetupDomainFiles() []string {
	return []string{"combat/monster_setup.go", "combat/monster_setup_hit_points.go", "combat/monster_setup_defenses.go", "combat/monster_setup_stats.go", "combat/monster_setup_numbers.go"}
}

func monsterSetupUseCaseFiles() []string { return []string{"combat/set_up_monster_combatant.go"} }

// rankTwentySixFiles are the production files of the other specs of rank 26 (monster-limited-use, magical-effects,
// movement-lifting-and-flight, grid-positions-and-movement, readying-a-spell), which monster-combatant-setup.md FR-045
// forbids the pure setup files to use.
func rankTwentySixFile(path string) bool {
	name := strings.TrimSuffix(strings.TrimPrefix(path, "combat/"), ".go")
	for _, prefix := range []string{"limited_use_", "apply_monster_turn_start", "magical_effect", "grid_", "held_spell", "ready", "movement_burrow", "movement_check",
		"movement_flight", "movement_jump", "movement_landing", "movement_load", "movement_segment"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// Requirement: doc/specs/monster-combatant-setup.md FR-045, FR-028, FR-047 - the pure setup files import only the kernel, content, d20 and the standard library's slices and strings, never telemetry, context or a roller; they name no signal constant, no Roller, no RollInitiative, no dice.Roll and no declaration of a file of another spec of rank 26 or higher; the use-case file adds context, fmt, telemetry and dice.
// Path: unhappy (forbidden edge).
func TestMonsterSetupFilesImportOnlyWhatTheSpecAllows(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	pkg := combatPackage(t)
	higher := map[string]bool{}
	for name, files := range declared(pkg) {
		if slices.ContainsFunc(files, rankTwentySixFile) {
			higher[name] = true
		}
	}
	seen := 0
	for _, f := range pkg.Production() {
		allowed := []string{"ability", "content", "d20", "dice", "modifier", "srderr"}
		domain := slices.Contains(monsterSetupDomainFiles(), f.Path)
		switch {
		case domain:
		case slices.Contains(monsterSetupUseCaseFiles(), f.Path):
			allowed = append(allowed, "telemetry")
		default:
			continue
		}
		seen++
		for _, imp := range f.Imports() {
			if p, ok := strings.CutPrefix(imp.Path, mod); ok {
				if !slices.Contains(allowed, p) || (domain && p == "telemetry") {
					t.Errorf("%s imports %s, which monster-combatant-setup.md does not allow", f.Path, p)
				}
				continue
			}
			switch imp.Path {
			case "slices", "strings":
			case "context", "fmt":
				if domain {
					t.Errorf("%s imports %s: the setup files are pure (FR-045)", f.Path, imp.Path)
				}
			default:
				t.Errorf("%s imports %s, which monster-combatant-setup.md does not allow", f.Path, imp.Path)
			}
		}
		if !domain {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == "dice" && sel.Sel.Name == "Roll" {
					t.Errorf("%s calls dice.Roll: the setup rolls no die (FR-028)", f.Path)
				}
			}
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			for _, prefix := range []string{"metric", "span", "attr", "value", "msg"} {
				if strings.HasPrefix(id.Name, prefix) && len(id.Name) > len(prefix) && id.Name[len(prefix)] >= 'A' && id.Name[len(prefix)] <= 'Z' {
					t.Errorf("%s names the signal constant %s: the domain is silent", f.Path, id.Name)
				}
			}
			switch id.Name {
			case "Roller", "RollInitiative":
				t.Errorf("%s names %s: the setup rolls no die (FR-028)", f.Path, id.Name)
			}
			if higher[id.Name] {
				t.Errorf("%s uses %s, declared by a spec of rank 26 or higher (FR-045)", f.Path, id.Name)
			}
			return true
		})
	}
	if want := len(monsterSetupDomainFiles()) + len(monsterSetupUseCaseFiles()); seen != want {
		t.Fatalf("found %d of %d monster setup files", seen, want)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-007, FR-025, FR-037 - the closed sets have exactly the variants the spec lists.
// Path: unhappy (duplicate).
func TestMonsterSetupClosedSetsHaveExactlyTheirVariants(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"isMonsterHitPointsMethod":      {"RollHitPointDice", "UseAverageHitPoints"},
		"isResolvedMonsterHitPoints":    {"PrintedAverageHitPoints", "RolledMonsterHitPoints"},
		"isMonsterInitiativePlan":       {"FixedMonsterInitiative", "RolledMonsterInitiative"},
		"isSetUpMonsterCombatantResult": {"SetUpMonsterCombatantApplied", "SetUpMonsterCombatantInvalid"},
	}
	for marker, names := range want {
		if got := markerReceivers(t, marker); !slices.Equal(got, names) {
			t.Errorf("%s is implemented by %v, want %v", marker, got, names)
		}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-047 - the setup and its unapplied lists hold no display text (no string field) and no random source.
// Path: unhappy (immutability).
func TestMonsterSetupTypesHoldNoTextAndNoRandomSource(t *testing.T) {
	t.Parallel()
	checked := 0
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok || !slices.Contains([]string{"MonsterSetup", "UnappliedDefense", "UnappliedSpeed"}, ts.Name.Name) {
				return true
			}
			checked++
			ast.Inspect(ts.Type, func(m ast.Node) bool {
				switch x := m.(type) {
				case *ast.Ident:
					if x.Name == "string" {
						t.Errorf("%s holds a string (display text)", ts.Name.Name)
					}
				case *ast.SelectorExpr:
					if x.Sel.Name == "Rand" {
						t.Errorf("%s holds a random source", ts.Name.Name)
					}
				}
				return true
			})
			return true
		})
	}
	if checked != 3 {
		t.Errorf("found %d of the 3 setup types", checked)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-037, FR-043 - the use case constructor takes a Roller and options only.
// Path: unhappy (forbidden edge).
func TestSetUpMonsterCombatantConstructorTakesOnlyRollerAndOptions(t *testing.T) {
	t.Parallel()
	found := 0
	for _, f := range combatPackage(t).Production() {
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "NewSetUpMonsterCombatant" {
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
				t.Errorf("NewSetUpMonsterCombatant takes (%s), want (Roller,...Option)", got)
			}
		}
	}
	if found != 1 {
		t.Errorf("found %d NewSetUpMonsterCombatant, want 1", found)
	}
}

// Requirement: doc/specs/monster-combatant-setup.md section 7, plan-phase-4.md P11 - signals.go pins the operation value and span, the clamp counter, the method attribute and its two values and the five result values at their catalogue values, and signals.md has a row for the span, the counter and the attribute.
// Path: unhappy (invalid input).
func TestMonsterSetupSignalNamesAndCatalogue(t *testing.T) {
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
		"valueOperationSetUpMonsterCombatant": "set_up_monster_combatant", "spanSetUpMonsterCombatant": "srd5e.combat.set_up_monster_combatant",
		"metricMonsterHitPointsClamped": "srd5e.combat.monster_hit_points_clamped", "attrMonsterHitPointsMethod": "srd5e.combat.monster_hit_points_method",
		"valueMonsterHitPointsMethodAverage": "average", "valueMonsterHitPointsMethodRolled": "rolled",
		"valueResultSetUpAverage": "set_up_average", "valueResultSetUpRolled": "set_up_rolled", "valueResultSizeChoiceRequired": "size_choice_required",
		"valueResultSizeNotOffered": "size_not_offered", "valueResultContentNumberOutOfRange": "content_number_out_of_range",
	}
	for name, value := range want {
		if got[name] != value {
			t.Errorf("signals.go %s = %q, want %q", name, got[name], value)
		}
	}
	doc := catalogue(t)
	for _, name := range []string{"srd5e.combat.set_up_monster_combatant", "srd5e.combat.monster_hit_points_clamped", "srd5e.combat.monster_hit_points_method"} {
		if !strings.Contains(doc, "| `"+name+"` |") {
			t.Errorf("signals.md has no row for %s", name)
		}
	}
}

// Requirement: doc/specs/monster-combatant-setup.md FR-046, section 6 (Exports) - package combat exports the setup types, options, rules and use case once each and no generic variant name.
// Path: happy.
func TestCombatExportsMonsterSetupTypes(t *testing.T) {
	t.Parallel()
	want := strings.Fields(`
		MonsterSetup NewMonsterSetup MustMonsterSetup SetupOption WithSize WithZeroHitPointsRule WithInitiativeMode WithExtraAffinities
		MonsterSetupProblem SizeChoiceRequired SizeNotOffered RolledExpressionMismatch ContentNumberOutOfRange SetupArgumentInvalid MonsterSetupError
		MonsterHitPointsMethod UseAverageHitPoints RollHitPointDice ResolvedMonsterHitPoints PrintedAverageHitPoints RolledMonsterHitPoints
		MonsterInitiativeMode MonsterInitiativeRolled MonsterInitiativeFixed MonsterInitiativePlan RolledMonsterInitiative FixedMonsterInitiative
		MonsterDefenseKind DefenseVulnerability DefenseResistance DefenseImmunity UnappliedDefenseReason DefenseQualified DefenseDescribed UnappliedDefense UnappliedSpeed
		SetUpMonsterCombatant NewSetUpMonsterCombatant SetUpMonsterCombatantCommand SetUpMonsterCombatantResult SetUpMonsterCombatantApplied SetUpMonsterCombatantInvalid`)
	have := declared(combatPackage(t))
	for _, name := range want {
		if len(have[name]) != 1 {
			t.Errorf("combat declares %s %d times, want once", name, len(have[name]))
		}
	}
	for _, generic := range []string{"Applied", "Invalid", "Rejected", "Setup"} {
		if len(have[generic]) != 0 {
			t.Errorf("combat exports the generic name %s (FR-046)", generic)
		}
	}
}
