package archtest_test

import (
	"go/ast"
	"go/token"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// attackDomainFiles are the pure domain files of attacks.md: silent and importing no telemetry.
func attackDomainFiles() []string {
	return []string{"combat/attack_form.go", "combat/cover.go", "combat/attack_range.go", "combat/attack_situation.go",
		"combat/attack_problem.go", "combat/circumstance_collector.go", "combat/damage_spec.go", "combat/attack_damage_composer.go",
		"combat/weapon_use.go", "combat/unarmed_strike_rules.go", "combat/resolve_attack_command.go", "combat/resolve_attack_result.go",
		"combat/resolve_unarmed_strike_command.go", "combat/resolve_unarmed_strike_result.go"}
}

// attackUseCaseFiles are the two use-case files of attacks.md.
func attackUseCaseFiles() []string {
	return []string{"combat/resolve_attack.go", "combat/resolve_unarmed_strike.go"}
}

// usedNames returns the identifiers and selector names a file uses (comments are not code).
func usedNames(f archtest.File) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(f.AST, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			out[id.Name] = true
		}
		return true
	})
	return out
}

func attackFiles(t *testing.T) map[string]archtest.File {
	t.Helper()
	out := map[string]archtest.File{}
	for _, f := range combatPackage(t).Production() {
		if slices.Contains(attackDomainFiles(), f.Path) || slices.Contains(attackUseCaseFiles(), f.Path) {
			out[f.Path] = f
		}
	}
	if want := len(attackDomainFiles()) + len(attackUseCaseFiles()); len(out) != want {
		t.Fatalf("found %d of %d attack files", len(out), want)
	}
	return out
}

// Requirement: doc/specs/attacks.md FR-119, FR-109, FR-123 - no file of attacks.md mentions ConditionEffects, ConditionSet or
// the d20 use case (Resolver, NewResolver, Command), and the domain files import no telemetry, dice, content-source or test
// package: attack resolution is pure with respect to conditions and composes the d20 functions only.
// Path: unhappy (forbidden edge).
func TestAttackFilesAreConditionPureAndNeverCallTheD20UseCase(t *testing.T) {
	t.Parallel()
	const mod = "github.com/Whealinghold/srd5e-go/"
	for path, f := range attackFiles(t) {
		for name := range usedNames(f) {
			if slices.Contains([]string{"ConditionEffects", "ConditionSet", "NewResolver", "Resolver", "ForcedFailure", "RolledAttack", "RolledCheck"}, name) {
				t.Errorf("%s names %s (attacks.md FR-119, FR-109)", path, name)
			}
		}
		for _, imp := range f.Imports() {
			p, ok := strings.CutPrefix(imp.Path, mod)
			switch {
			case !ok:
			case p == "telemetry" && slices.Contains(attackDomainFiles(), path):
				t.Errorf("%s imports telemetry; the attack domain is silent (observability.md O1)", path)
			case strings.HasSuffix(p, "test") || p == "content/data":
				t.Errorf("%s imports %s", path, p)
			}
		}
	}
}

// Requirement: doc/specs/attacks.md FR-109, FR-127 - neither use case mentions the other, and attackFlow is an unexported struct
// holding only a Roller: no telemetry, no use case.
// Path: unhappy (rule interaction).
func TestAttackUseCasesDoNotCallEachOtherAndTheFlowIsPlain(t *testing.T) {
	t.Parallel()
	files := attackFiles(t)
	banned := map[string][]string{
		"combat/resolve_attack.go":         {"ResolveUnarmedStrike", "NewResolveUnarmedStrike", "ResolveUnarmedStrikeCommand"},
		"combat/resolve_unarmed_strike.go": {"ResolveAttack", "NewResolveAttack", "ResolveAttackCommand"},
	}
	for path, names := range banned {
		m := usedNames(files[path])
		for _, n := range names {
			if m[n] {
				t.Errorf("%s names %s: a use case never calls another (shared-types-and-ownership FR-044)", path, n)
			}
		}
	}
	found := false
	ast.Inspect(files["combat/resolve_attack.go"].AST, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "attackFlow" {
			return true
		}
		found = true
		st, ok := ts.Type.(*ast.StructType)
		if !ok || len(st.Fields.List) != 1 || len(st.Fields.List[0].Names) != 1 || st.Fields.List[0].Names[0].Name != "roller" {
			t.Errorf("attackFlow must be a struct holding only a roller")
		}
		return false
	})
	if !found {
		t.Errorf("attackFlow is not declared in combat/resolve_attack.go")
	}
	for _, f := range combatPackage(t).Production() {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if fd, ok := n.(*ast.FuncDecl); ok && fd.Recv != nil && fd.Name.IsExported() && len(fd.Recv.List) == 1 {
				if id, ok := fd.Recv.List[0].Type.(*ast.Ident); ok && id.Name == "attackFlow" {
					t.Errorf("%s: attackFlow has the exported method %s", f.Path, fd.Name.Name)
				}
			}
			return true
		})
	}
}

// Requirement: doc/specs/attacks.md FR-123 - combat declares none of Proficiency, AttackConclusion, AutomaticCriticalHitPolicy,
// Dc or DamageType, and no exported top-level name of combat duplicates an exported one of d20 or content.
// Path: unhappy (duplicate).
func TestCombatDoesNotDuplicateUpstreamNames(t *testing.T) {
	t.Parallel()
	mine := declared(combatPackage(t))
	for _, name := range []string{"Proficiency", "AttackConclusion", "AutomaticCriticalHitPolicy", "Dc", "DC", "DamageType", "CreatureSize"} {
		if len(mine[name]) != 0 {
			t.Errorf("combat declares %s in %v (attacks.md FR-123)", name, mine[name])
		}
	}
	// Names that repeat by design: the use-case option shape of every context, and the use case ResolveAttack of
	// attacks.md FR-115 that shares its name with the d20 function it composes (a package qualifier tells them apart).
	// HalfDamage is the hit-points adjustment of an earlier step.
	allowed := []string{"Option", "Roller", "WithTelemetry", "WithClock", "ResolveAttack", "HalfDamage"}
	for _, p := range loadModule(t).Packages {
		if p.Dir != "d20" && p.Dir != "content" {
			continue
		}
		for name := range declared(p) {
			if token.IsExported(name) && len(mine[name]) != 0 && !slices.Contains(allowed, name) {
				t.Errorf("combat declares %s, which package %s already exports (attacks.md FR-123)", name, p.Dir)
			}
		}
	}
}
