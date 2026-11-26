package archtest_test

import (
	"go/ast"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

const combatDoc = "// Package combat is a fixture.\npackage combat\n"

// useCaseSrc is a clean use case Do in package combat: command, closed result with an applied and a
// rejected variant, and the rejection the rejected variant holds.
const useCaseSrc = `package combat

import "context"

// DoCommand is the input of Do.Run.
type DoCommand struct{}

// DoResult is the closed set of outcomes of Do.Run. Variants: DoApplied, DoRejected.
type DoResult interface{ isDoResult() }

// DoApplied is applied.
type DoApplied struct{}

// DoRejection is why a Do is refused.
type DoRejection int

// DoRejected is refused.
type DoRejected struct{ Rejection DoRejection }

func (DoApplied) isDoResult()  {}
func (DoRejected) isDoResult() {}

// Do is a use case.
type Do struct{}

// Run runs it.
func (Do) Run(ctx context.Context, cmd DoCommand) (DoResult, error) { return DoApplied{}, nil }
`

// combatRules runs every rule against the clean module plus the files and keeps the combat scans' rules.
func combatRules(t *testing.T, files map[string]string) []string {
	t.Helper()
	files["combat/doc.go"] = combatDoc
	var out []string
	for _, r := range rules(t, overlay(files)) {
		if strings.HasPrefix(r, "combat-") {
			out = append(out, r)
		}
	}
	return out
}

func replaceUseCase(old, new string) string {
	if !strings.Contains(useCaseSrc, old) {
		panic("fixture lacks " + old)
	}
	return strings.Replace(useCaseSrc, old, new, 1)
}

// Requirement: doc/specs/combat-foundations.md FR-022, FR-023 - the naming scan fires on a generic name,
// a use case whose command or result lacks its <UseCase> prefix, a result that is not a closed interface,
// a Rejected or Invalid variant without a Rejection, Problem or error field, a NotPossible variant without a
// Reason, and a variant shared by two use cases, and reports (no panic) a Run whose first result is not a type of the package; and stays silent on the clean forms.
// Path: per case.
func TestCombatNamingScan(t *testing.T) {
	t.Parallel()
	const naming = archtest.RuleCombatNaming
	more := "\n// DoInvalid is bad input.\ntype DoInvalid struct{ %s }\n\nfunc (DoInvalid) isDoResult() {}\n"
	cases := []struct {
		name, requirement, path string
		src                     string
		want                    string // "" means no combat rule fires
	}{
		{"a use case named by the rule", "FR-022", "happy", useCaseSrc, ""},
		{"generic type name", "FR-022", "unhappy (name collision)", useCaseSrc + "\n// Applied is generic.\ntype Applied struct{}\n", naming},
		{"generic constant name", "FR-022", "unhappy (name collision)", useCaseSrc + "\n// Found is generic.\nconst Found = 1\n", naming},
		{"generic function name", "FR-022", "unhappy (name collision)", useCaseSrc + "\n// NotFound is generic.\nfunc NotFound() {}\n", naming},
		{"result without the use case prefix", "FR-022", "unhappy (name collision)",
			replaceUseCase("(DoResult, error)", "(OtherResult, error)") + "\n// OtherResult is closed. Variants: none.\ntype OtherResult interface{ isOtherResult() }\n", naming},
		{"command without the use case prefix", "FR-022", "unhappy (name collision)",
			replaceUseCase("cmd DoCommand", "cmd Input") + "\n// Input is the input.\ntype Input struct{}\n", naming},
		{"run with one parameter", "FR-022", "unhappy (invalid input)", replaceUseCase("ctx context.Context, cmd DoCommand", "ctx context.Context"), naming},
		{"result that is not closed", "FR-022", "unhappy (invalid input)", replaceUseCase("interface{ isDoResult() }", "interface{ IsDo() }"), naming},
		{"rejected variant without a rejection", "FR-023", "unhappy (invalid input)", replaceUseCase("struct{ Rejection DoRejection }", "struct{}"), naming},
		{"rejected variant holding a rejection through an embedded struct", "FR-023", "happy",
			replaceUseCase("struct{ Rejection DoRejection }", "struct{ reason }") + "\ntype reason struct{ r DoRejection }\n", ""},
		{"rejected variant holding a list of problems", "FR-023", "happy",
			replaceUseCase("struct{ Rejection DoRejection }", "struct{ Problems []DoProblem }") + "\n// DoProblem is one problem.\ntype DoProblem int\n", ""},
		{"invalid variant holding the joined error", "FR-023", "happy", useCaseSrc + strings.Replace(more, "%s", "err error", 1), ""},
		{"invalid variant with no field", "FR-023", "happy", useCaseSrc + strings.Replace(more, "%s", "", 1), ""},
		{"invalid variant holding an unrelated field", "FR-023", "unhappy (invalid input)", useCaseSrc + strings.Replace(more, "%s", "count int", 1), naming},
		{"not possible variant holding a reason", "FR-023", "happy",
			useCaseSrc + "\n// DoNotPossibleReason is why.\ntype DoNotPossibleReason int\n\n// DoNotPossible is a refusal.\ntype DoNotPossible struct{ Reason DoNotPossibleReason }\n\nfunc (DoNotPossible) isDoResult() {}\n", ""},
		{"not possible variant without a reason", "FR-023", "unhappy (invalid input)",
			useCaseSrc + "\n// DoNotPossible is a refusal.\ntype DoNotPossible struct{}\n\nfunc (DoNotPossible) isDoResult() {}\n", naming},
		{"variant shared by two use cases", "FR-023", "unhappy (rule interaction)",
			useCaseSrc + "\n// Other is a second use case.\ntype Other struct{}\n\n// OtherCommand is its input.\ntype OtherCommand struct{}\n\n// OtherResult is closed. Variants: DoApplied.\ntype OtherResult interface{ isOtherResult() }\n\nfunc (DoApplied) isOtherResult() {}\n\n// Run runs it.\nfunc (Other) Run(ctx context.Context, cmd OtherCommand) (OtherResult, error) { return DoApplied{}, nil }\n", naming},
		// A Run whose first result is not a type of the package is reported by the shape check, not a panic.
		{"run returning a builtin first result", "FR-022", "unhappy (invalid input)",
			useCaseSrc + "\n// Odd is a use case.\ntype Odd struct{}\n\n// Run runs it.\nfunc (Odd) Run(ctx context.Context, cmd DoCommand) (bool, error) { return false, nil }\n", naming},
		{"run returning a qualified first result", "FR-022", "unhappy (invalid input)",
			useCaseSrc + "\n// Odd is a use case.\ntype Odd struct{}\n\n// Run runs it.\nfunc (Odd) Run(ctx context.Context, cmd DoCommand) (context.CancelFunc, error) { return nil, nil }\n", naming},
		{"rejected type of a domain result is not a use case variant", "FR-023", "happy",
			useCaseSrc + "\n// ThingResult is a domain answer. Variants: ThingRejected.\ntype ThingResult interface{ isThingResult() }\n\n// ThingRejected is refused.\ntype ThingRejected struct{}\n\nfunc (ThingRejected) isThingResult() {}\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, map[string]string{"combat/do.go": c.src})
			switch {
			case c.want == "" && len(got) != 0:
				t.Errorf("want silence, got %v [%s]", got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// envelopeSrc declares the transportEnvelope marker, a minimal Event and the named envelopes; each entry
// of skip names one of them to leave out.
func envelopeSrc(mutate func(name, decl string) string) string {
	var b strings.Builder
	b.WriteString("package combat\n\n// Event is an event.\ntype Event interface {\n\tKind() string\n\tCombatants() []string\n}\n\ntype transportEnvelope interface{ transportEnvelope() }\n")
	for _, name := range []string{"TurnEventOccurred", "HitPointEventOccurred", "ConditionEventOccurred", "GrappleEventOccurred"} {
		decl := "\n// " + name + " wraps an event.\ntype " + name + " struct{ wrapped Event }\n\n" +
			"// Kind is the wrapped kind.\nfunc (e " + name + ") Kind() string { return e.wrapped.Kind() }\n\n" +
			"// Combatants are the wrapped combatants.\nfunc (e " + name + ") Combatants() []string { return e.wrapped.Combatants() }\n\n" +
			"func (" + name + ") transportEnvelope() {}\n"
		b.WriteString(mutate(name, decl))
	}
	return b.String()
}

// Requirement: doc/specs/combat-foundations.md FR-038 - the marked set equals the envelopes the owning spec
// names; an envelope delegates Kind and Combatants; a type named like an envelope is marked.
// Path: per case.
func TestCombatEnvelopeScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatEnvelope
	keep := func(_, decl string) string { return decl }
	cases := []struct {
		name, requirement, path string
		src                     string
		want                    string
	}{
		{"the four named envelopes", "FR-038", "happy", envelopeSrc(keep), ""},
		{"a package without the marker is not scanned for missing envelopes", "FR-038", "happy", "package combat\n", ""},
		{"a named envelope that is not marked", "FR-038", "unhappy (rule interaction)", envelopeSrc(func(n, d string) string {
			if n == "TurnEventOccurred" {
				return strings.Replace(d, "func (TurnEventOccurred) transportEnvelope() {}\n", "", 1)
			}
			return d
		}), rule},
		{"a named envelope that is missing", "FR-038", "unhappy (rule interaction)", envelopeSrc(func(n, d string) string {
			if n == "GrappleEventOccurred" {
				return ""
			}
			return d
		}), rule},
		{"a marked type that is not named", "FR-038", "unhappy (rule interaction)",
			envelopeSrc(keep) + "\n// OtherEvent wraps.\ntype OtherEvent struct{ wrapped Event }\n\n// Kind is a kind.\nfunc (e OtherEvent) Kind() string { return e.wrapped.Kind() }\n\n// Combatants are combatants.\nfunc (e OtherEvent) Combatants() []string { return nil }\n\nfunc (OtherEvent) transportEnvelope() {}\n", rule},
		{"an unmarked type named like an envelope", "FR-038", "unhappy (name collision)",
			envelopeSrc(keep) + "\n// ShapeOccurred is not marked.\ntype ShapeOccurred struct{}\n", rule},
		{"an envelope whose Kind is a literal", "FR-038", "unhappy (invalid input)", envelopeSrc(func(n, d string) string {
			if n == "HitPointEventOccurred" {
				return strings.Replace(d, "return e.wrapped.Kind()", "return \"hit_point_event\"", 1)
			}
			return d
		}), rule},
		{"an envelope whose Combatants is nil", "FR-038", "unhappy (invalid input)", envelopeSrc(func(n, d string) string {
			if n == "ConditionEventOccurred" {
				return strings.Replace(d, "return e.wrapped.Combatants()", "return nil", 1)
			}
			return d
		}), rule},
		{"an envelope without Combatants", "FR-038", "unhappy (invalid input)", envelopeSrc(func(n, d string) string {
			if n == "TurnEventOccurred" {
				i := strings.Index(d, "// Combatants are")
				j := strings.Index(d, "func (TurnEventOccurred) transportEnvelope")
				return d[:i] + d[j:]
			}
			return d
		}), rule},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combatRules(t, map[string]string{"combat/envelope.go": c.src})
			switch {
			case c.want == "" && len(got) != 0:
				t.Errorf("want silence, got %v [%s]", got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// upstream fixtures: a d20 domain service, a character use case and a content port.
func upstreamFiles() map[string]string {
	return map[string]string{
		"d20/resolver.go":      "package d20\n\nimport \"context\"\n\n// Resolver is a domain service.\ntype Resolver struct{}\n\n// Resolve resolves.\nfunc (Resolver) Resolve(ctx context.Context) {}\n",
		"d20/value.go":         "package d20\n\n// DifficultyClass is a value object.\ntype DifficultyClass int\n",
		"character/doc.go":     "// Package character is a fixture.\npackage character\n",
		"character/levelup.go": "package character\n\nimport \"context\"\n\n// LevelUpCharacter is a use case.\ntype LevelUpCharacter struct{}\n\n// Run runs it.\nfunc (LevelUpCharacter) Run(ctx context.Context) {}\n",
		"content/doc.go":       "// Package content is a fixture.\npackage content\n",
		"content/source.go":    "package content\n\nimport \"context\"\n\n// Source is a port.\ntype Source interface{ Lookup(ctx context.Context) }\n",
	}
}

// Requirement: doc/specs/combat-foundations.md FR-026 - a combat file names no use case of character and no
// d20 service outside a use-case file; the published language, a port and a d20 service in a use-case file
// pass; toolbox and internal/ imports fail through A1 and A2.
// Path: per case.
func TestCombatImportScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatImport
	useCaseFile := func(body string) string {
		return "package combat\n\nimport (\n\t\"example.com/m/d20\"\n\t\"example.com/m/character\"\n\t\"example.com/m/content\"\n\t_ \"example.com/m/telemetry\"\n)\n\n// Op is a use case.\ntype Op struct{}\n\n// Apply applies it.\nfunc (Op) Apply() {}\n\nvar _ = d20.DifficultyClass(0)\nvar _ = character.LevelUpCharacter{}\nvar _ content.Source\n" + body
	}
	domainFile := func(body string) string {
		return "package combat\n\nimport (\n\t\"example.com/m/d20\"\n\t\"example.com/m/character\"\n\t\"example.com/m/content\"\n)\n\nvar _ = d20.DifficultyClass(0)\nvar _ = character.LevelUpCharacter{}\nvar _ content.Source\n" + body
	}
	cases := []struct {
		name, requirement, path string
		src                     string
		want                    string
	}{
		{"published language and a port in a domain file", "FR-026", "happy", "package combat\n\nimport (\n\t\"example.com/m/d20\"\n\t\"example.com/m/content\"\n)\n\nvar _ = d20.DifficultyClass(0)\nvar _ content.Source\n", ""},
		{"a d20 service in a use-case file", "FR-026", "happy", "package combat\n\nimport (\n\t\"example.com/m/d20\"\n\t_ \"example.com/m/telemetry\"\n)\n\n// Op is a use case.\ntype Op struct{}\n\n// Apply applies it.\nfunc (Op) Apply() {}\n\nvar _ d20.Resolver\n", ""},
		{"a d20 service in a domain file", "FR-026", "unhappy (forbidden edge)", strings.Replace(domainFile("var _ d20.Resolver\n"), "var _ = character.LevelUpCharacter{}\n", "", 1), rule},
		{"a character use case in a domain file", "FR-026", "unhappy (forbidden edge)", domainFile(""), rule},
		{"a character use case in a use-case file", "FR-026", "unhappy (forbidden edge)", useCaseFile(""), rule},
		{"an import of toolbox", "FR-026", "unhappy (cycle)", "package combat\n\nimport _ \"example.com/m/toolbox\"\n", archtest.RuleImport},
		{"an import of internal", "FR-026", "unhappy (forbidden edge)", "package combat\n\nimport _ \"example.com/m/internal/archtest\"\n", archtest.RuleInternal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			files := upstreamFiles()
			files["combat/uses.go"] = c.src
			files["combat/doc.go"] = combatDoc
			var got []string
			for _, r := range rules(t, overlay(files)) {
				if strings.HasPrefix(r, "combat-") || r == archtest.RuleImport || r == archtest.RuleInternal {
					got = append(got, r)
				}
			}
			switch {
			case c.want == "" && len(got) != 0:
				t.Errorf("want silence, got %v [%s]", got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

const rankTable = "# Decisions\n\n### 4.11 Spec rank table\n\n| Rank | Specs |\n|---|---|\n| 0 | `shared-types-and-ownership` |\n| 1 | `errors-and-results` |\n| 20 | `combat-foundations` |\n| 21 | `hit-points-and-damage` |\n| 41 | `toolbox-poison` |\n\n## 5. Model\n"

func specWith(depends string) string {
	return "# Spec\n\n| | |\n|---|---|\n| Status | implemented |\n| Depends on | " + depends + " |\n"
}

// Requirement: doc/specs/combat-foundations.md FR-026 and shared-types-and-ownership.md FR-056 - the "Depends
// on" row of a combat spec lists only specs of strictly lower rank: never an equal or higher rank, a toolbox
// spec or a spec missing from the rank table; every ranked combat spec has a file with the row.
// Path: per case.
func TestCombatSpecRankScan(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleCombatSpecRank
	base := func(foundations, hp string) map[string]string {
		return map[string]string{
			"doc/specs/shared-types-and-ownership.md": rankTable,
			"doc/specs/combat-foundations.md":         foundations,
			"doc/specs/hit-points-and-damage.md":      hp,
		}
	}
	ok := specWith("`errors-and-results.md`")
	cases := []struct {
		name, requirement, path string
		docs                    map[string]string
		want                    string
	}{
		{"lower ranks only", "FR-026", "happy", base(ok, specWith("`combat-foundations.md`, `errors-and-results.md`, `shared-types-and-ownership.md`")), ""},
		{"a spec of equal rank", "FR-026", "unhappy (cycle)", base(specWith("`combat-foundations.md`"), ok), rule},
		{"a toolbox spec", "FR-026", "unhappy (forbidden edge)", base(specWith("`toolbox-poison.md`"), ok), rule},
		{"a higher combat spec", "FR-026", "unhappy (cycle)", base(specWith("`hit-points-and-damage.md`"), ok), rule},
		{"a spec missing from the table", "FR-026", "unhappy (invalid input)", base(specWith("`unranked.md`"), ok), rule},
		{"prose in the cell is not a dependency", "FR-026", "happy", base(specWith("`errors-and-results.md` (toolbox-poison reads it)"), ok), ""},
		{"a ranked spec without a file", "FR-026", "unhappy (invalid input)", map[string]string{"doc/specs/shared-types-and-ownership.md": rankTable, "doc/specs/combat-foundations.md": ok}, rule},
		{"a spec without the row", "FR-026", "unhappy (invalid input)", base("# Spec\n", ok), rule},
		{"the rank table is absent", "FR-026", "happy", map[string]string{"doc/specs/combat-foundations.md": specWith("`combat-foundations.md`")}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			files := map[string]string{}
			for k, v := range c.docs {
				files[k] = v
			}
			got := combatRules(t, files)
			switch {
			case c.want == "" && len(got) != 0:
				t.Errorf("want silence, got %v [%s]", got, c.requirement)
			case c.want != "" && !slices.Contains(got, c.want):
				t.Errorf("want %s among %v [%s]", c.want, got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-022, FR-038 - the scans are not vacuous on the real module:
// it holds many use cases with a Run method, and exactly the four named envelopes carry the marker.
// Path: unhappy (boundary).
func TestCombatScansSeeTheRealModule(t *testing.T) {
	t.Parallel()
	pkg := combatPackage(t)
	runs, markers := 0, []string{}
	for _, f := range pkg.Production() {
		for _, d := range f.AST.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil {
				continue
			}
			recv := ""
			switch r := fd.Recv.List[0].Type.(type) {
			case *ast.Ident:
				recv = r.Name
			case *ast.StarExpr:
				recv = r.X.(*ast.Ident).Name
			}
			switch {
			case fd.Name.Name == "Run" && ast.IsExported(recv):
				runs++
			case fd.Name.Name == "transportEnvelope":
				markers = append(markers, recv)
			}
		}
	}
	slices.Sort(markers)
	want := []string{"ConditionEventOccurred", "GrappleEventOccurred", "HitPointEventOccurred", "TurnEventOccurred"}
	if !slices.Equal(markers, want) {
		t.Errorf("marked envelopes = %v, want %v", markers, want)
	}
	if runs < 50 {
		t.Errorf("found %d use cases with a Run method, want at least 50", runs)
	}
}
