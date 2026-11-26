package archtest_test

import (
	"go/ast"
	"go/token"
	"io/fs"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

func combatPackage(t *testing.T) archtest.Package {
	t.Helper()
	for _, p := range loadModule(t).Packages {
		if p.Dir == "combat" {
			return p
		}
	}
	t.Fatal("package combat not found")
	return archtest.Package{}
}

// declared returns every top-level name the production files declare (types,
// funcs without a receiver, consts, vars), with the file that declares it.
func declared(pkg archtest.Package) map[string][]string {
	out := map[string][]string{}
	for _, f := range pkg.Production() {
		for _, d := range f.AST.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					out[d.Name.Name] = append(out[d.Name.Name], f.Path)
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						out[s.Name.Name] = append(out[s.Name.Name], f.Path)
					case *ast.ValueSpec:
						for _, n := range s.Names {
							out[n.Name] = append(out[n.Name], f.Path)
						}
					}
				}
			}
		}
	}
	return out
}

// Requirement: doc/specs/combat-foundations.md FR-001, FR-025 - each reserved
// name is declared exactly once, in this spec's files. (The Go identifiers of
// the unexported helpers are the lower-case forms; the compiler already refuses
// a duplicate, so this pins where each lives.)
// Path: unhappy (duplicate).
func TestCombatReservedNamesDeclaredOnce(t *testing.T) {
	t.Parallel()
	d := declared(combatPackage(t))
	reserved := map[string]string{
		"CombatantID": "combat/combatant_id.go", "CombatantIDError": "combat/combatant_id.go",
		"CombatantIDRejection": "combat/combatant_id.go", "CombatantKind": "combat/combatant_kind.go",
		"PlayerCharacterCombatant": "combat/combatant_kind.go", "MonsterCombatant": "combat/combatant_kind.go",
		"CombatantRef": "combat/combatant_ref.go", "Event": "combat/combat_event.go",
		"combatOperation": "combat/combat_operation.go", "combatResultClass": "combat/result_class.go",
		"instrumenter": "combat/instrumenter.go", "transportEnvelope": "combat/combat_event.go",
		"RuleOptions": "combat/rule_options.go", "PlayingOnAGrid": "combat/rule_options.go",
	}
	for name, file := range reserved {
		if got := d[name]; len(got) != 1 || got[0] != file {
			t.Errorf("%s is declared in %v, want exactly %s", name, got, file)
		}
	}
	if got := d["CombatSignals"]; len(got) != 0 {
		t.Errorf("CombatSignals is a block of constants in signals.go, not a type: declared in %v", got)
	}
}

// Requirement: doc/specs/combat-foundations.md FR-009 - no file of combat
// imports a randomness or uuid package, and no function returns a CombatantID
// except NewCombatantID and MustCombatantID.
// Path: unhappy (determinism).
func TestCombatDoesNotGenerateIDs(t *testing.T) {
	t.Parallel()
	for _, f := range combatPackage(t).Production() {
		for _, imp := range f.Imports() {
			p := imp.Path
			if strings.Contains(p, "rand") || strings.Contains(p, "uuid") {
				t.Errorf("%s imports %s", f.Path, p)
			}
		}
		for _, d := range f.AST.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Type.Results == nil || fn.Name.Name == "NewCombatantID" || fn.Name.Name == "MustCombatantID" {
				continue
			}
			for _, r := range fn.Type.Results.List {
				if id, ok := r.Type.(*ast.Ident); ok && id.Name == "CombatantID" {
					t.Errorf("%s: %s returns a CombatantID", f.Path, fn.Name.Name)
				}
			}
		}
	}
}

// Requirement: doc/specs/combat-foundations.md FR-027, FR-039 - srd5e.combat.
// names are string constants in combat/signals.go only, with a legal prefix;
// signals.go holds neither srd5e.outcome nor error.type nor a d20 key.
// Path: unhappy (invalid input).
func TestCombatSignalNames(t *testing.T) {
	t.Parallel()
	prefixes := []string{"metric", "span", "msg", "attr", "value"}
	for _, f := range combatPackage(t).Production() {
		isSignals := f.Path == "combat/signals.go"
		ast.Inspect(f.AST, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, _ := strconv.Unquote(lit.Value)
			switch {
			case strings.HasPrefix(s, "srd5e.combat.") && !isSignals:
				t.Errorf("%s: literal %q outside signals.go", f.Path, s)
			case isSignals && (s == "srd5e.outcome" || s == "error.type" || strings.HasPrefix(s, "srd5e.d20_test.")):
				t.Errorf("signals.go re-declares %q", s)
			}
			return true
		})
		if !isSignals {
			continue
		}
		for _, d := range f.AST.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			if g.Tok != token.CONST {
				t.Errorf("signals.go declares a %s, want constants only", g.Tok)
				continue
			}
			for _, s := range g.Specs {
				v := s.(*ast.ValueSpec)
				if v.Type != nil {
					t.Errorf("signals.go constant %s is typed", v.Names[0].Name)
				}
				for _, n := range v.Names {
					if n.IsExported() || !hasAnyPrefix(n.Name, prefixes) {
						t.Errorf("signals.go constant %s must be unexported with a prefix of %v", n.Name, prefixes)
					}
				}
			}
		}
	}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// Requirement: doc/specs/combat-foundations.md FR-040 - every valueOperation<X>
// constant has a span<X> constant of srd5e.combat. plus its value, and the reverse.
// Path: unhappy (invalid input).
func TestCombatOperationSpanPairs(t *testing.T) {
	t.Parallel()
	consts := map[string]string{}
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/signals.go" {
			continue
		}
		for _, d := range f.AST.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.CONST {
				for _, s := range g.Specs {
					v := s.(*ast.ValueSpec)
					if lit, ok := v.Values[0].(*ast.BasicLit); ok {
						consts[v.Names[0].Name], _ = strconv.Unquote(lit.Value)
					}
				}
			}
		}
	}
	for name, value := range consts {
		if x, ok := strings.CutPrefix(name, "valueOperation"); ok {
			if span, ok := consts["span"+x]; !ok || span != "srd5e.combat."+value {
				t.Errorf("%s = %q has no span%s = %q", name, value, x, "srd5e.combat."+value)
			}
		}
		if x, ok := strings.CutPrefix(name, "span"); ok && name != "spanPrefix" {
			if op, ok := consts["valueOperation"+x]; !ok || value != "srd5e.combat."+op {
				t.Errorf("%s = %q has no valueOperation%s", name, value, x)
			}
		}
	}
}

// supersededSignals are the names earlier combat specs defined and
// combat-foundations section 5 replaces (and the attribute and signal names
// the plan P11 lists); the catalogue must not list any.
func supersededSignals() []string {
	return []string{
		"srd5e.combat.use_case", "srd5e.operation", "srd5e.combat.condition.operation",
		"srd5e.combat.condition.apply", "srd5e.combat.condition.remove", "srd5e.combat.condition.reduce_exhaustion",
		"srd5e.combat.condition.operations", "srd5e.combat.condition.duration", "srd5e.combat.condition.outcome",
		"srd5e.combat.attack.resolve", "srd5e.combat.attack.resolved", "srd5e.combat.attack.duration", "srd5e.combat.attack.errors",
		"srd5e.combat.unarmed_strike.resolve", "srd5e.combat.unarmed_strike.resolved",
		"srd5e.combat.unarmed_strike.duration", "srd5e.combat.unarmed_strike.errors",
		"srd5e.combat.encounter_transitions", "srd5e.combat.encounter_errors", "srd5e.combat.encounter_duration",
		"srd5e.combat.encounter_events", "srd5e.combat.rejection", "srd5e.combat.error_kind", "srd5e.combat.refusal",
	}
}

// mentions reports whether doc contains name not followed by a name character.
func mentions(doc, name string) bool {
	for rest := doc; ; {
		i := strings.Index(rest, name)
		if i < 0 {
			return false
		}
		rest = rest[i+len(name):]
		if rest == "" || !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789_", rune(rest[0])) {
			return true
		}
	}
}

func catalogue(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(loadModule(t).FS, "doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Requirement: doc/specs/combat-foundations.md FR-037, plan-phase-4.md P11 - the
// catalogue lists no superseded name, and each shared signal has exactly one row.
// Path: unhappy (duplicate).
func TestCombatCatalogue(t *testing.T) {
	t.Parallel()
	doc := catalogue(t)
	for _, name := range supersededSignals() {
		if mentions(doc, name) {
			t.Errorf("signals.md lists the superseded name %s", name)
		}
	}
	for _, name := range []string{"srd5e.combat.operations", "srd5e.combat.operation_duration", "srd5e.combat.errors", "srd5e.combat.events"} {
		rows := 0
		for _, line := range strings.Split(doc, "\n") {
			if strings.HasPrefix(line, "| `"+name+"` |") {
				rows++
			}
		}
		if rows != 1 {
			t.Errorf("signals.md has %d rows for %s, want 1", rows, name)
		}
	}
}

// Requirement: doc/specs/combat-foundations.md FR-037 - every signal name
// constant in combat/signals.go has a catalogue row.
// Path: unhappy (invalid input).
func TestCombatCatalogueMatchesConstants(t *testing.T) {
	t.Parallel()
	doc := catalogue(t)
	for _, f := range combatPackage(t).Production() {
		if f.Path != "combat/signals.go" {
			continue
		}
		for _, d := range f.AST.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range g.Specs {
				v := s.(*ast.ValueSpec)
				name := v.Names[0].Name
				if !strings.HasPrefix(name, "metric") && !strings.HasPrefix(name, "span") || name == "spanPrefix" {
					continue
				}
				val, _ := strconv.Unquote(v.Values[0].(*ast.BasicLit).Value)
				if !strings.Contains(doc, "| `"+val+"` |") {
					t.Errorf("signals.md has no row for %s (%s)", val, name)
				}
			}
		}
	}
}

// Requirement: doc/specs/combat-foundations.md FR-037 - the checker sees a
// superseded name and does not mistake a longer name for it.
// Path: unhappy (boundary).
func TestMentions(t *testing.T) {
	t.Parallel()
	if !mentions("| `srd5e.operation` |", "srd5e.operation") || !mentions("x srd5e.operation", "srd5e.operation") {
		t.Error("mentions missed an exact name")
	}
	if mentions("srd5e.operations", "srd5e.operation") || mentions("none", "srd5e.operation") {
		t.Error("mentions matched a longer name or an absent one")
	}
}

// Requirement: doc/specs/combat-foundations.md FR-012 - a CombatantKind variant has the single exported field ID and
// no exported method; each violating fixture fails (the real package is covered by the real-module Check test).
// Path: per case.
func TestCombatantKindCarriesIdentityOnly(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatKindIdentity
	kind := func(body string) map[string]string {
		return map[string]string{"combat/combatant_kind.go": "package combat\n\n" + body + "\n\nfunc (V) combatantKind() {}\n"}
	}
	runRuleCases(t, rule, []ruleCase{
		{"the ID only", "FR-012", "happy", kind("// V is a variant.\ntype V struct{ ID string }"), ""},
		{"an unexported field beside ID", "FR-012", "happy", kind("// V is a variant.\ntype V struct {\n\tID string\n\tn   int\n}"), ""},
		{"a second exported field", "FR-012", "unhappy (invalid input)", kind("// V is a variant.\ntype V struct {\n\tID   string\n\tSide int\n}"), rule},
		{"no ID field", "FR-012", "unhappy (invalid input)", kind("// V is a variant.\ntype V struct{ Name string }"), rule},
		{"an embedded exported type", "FR-012", "unhappy (invalid input)", kind("// V is a variant.\ntype V struct {\n\tID string\n\tHitPoints\n}"), rule},
		{"an exported method", "FR-012", "unhappy (invalid input)", kind("// V is a variant.\ntype V struct{ ID string }\n\n// Alive is a behaviour.\nfunc (V) Alive() bool { return true }"), rule},
	})
}

// Requirement: doc/specs/combat-foundations.md FR-014 - combatant_kind.go, combatant_ref.go and combatant_id.go import
// the standard library, character, content and srderr only; the real package passes and each violating fixture fails.
// Path: per case.
func TestCombatantReferenceFilesImports(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatReferenceDeps
	ref := func(file, imports string) map[string]string {
		return map[string]string{file: "package combat\n\nimport (\n" + imports + ")\n"}
	}
	runRuleCases(t, rule, []ruleCase{
		{"the allowed imports", "FR-014", "happy", ref("combat/combatant_ref.go", "\t\"strings\"\n\n\t\"example.com/m/character\"\n\t\"example.com/m/content\"\n\t\"example.com/m/srderr\"\n"), ""},
		{"d20 in the kind file", "FR-014", "unhappy (forbidden edge)", ref("combat/combatant_kind.go", "\t\"example.com/m/d20\"\n"), rule},
		{"a kernel package in the id file", "FR-014", "unhappy (forbidden edge)", ref("combat/combatant_id.go", "\t\"example.com/m/dice\"\n"), rule},
		{"a third-party package", "FR-014", "unhappy (forbidden edge)", ref("combat/combatant_ref.go", "\t\"github.com/other/pkg\"\n"), rule},
		{"another file may import d20", "FR-014", "happy", ref("combat/other.go", "\t\"example.com/m/d20\"\n"), ""},
	})
}
