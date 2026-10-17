package archtest_test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

const (
	telemetryDoc = "// Package telemetry is a fixture.\npackage telemetry\n"
	d20Doc       = "// Package d20 is a fixture.\npackage d20\n"
)

// useCase is a use-case file by role: a type with an exported method, in a
// context package that imports telemetry.
func useCase(body string) string {
	return "package d20\n\nimport (\n\t\"example.com/m/srderr\"\n\t_ \"example.com/m/telemetry\"\n)\n\n// R resolves.\ntype R struct{}\n\n// Resolve resolves.\nfunc (R) Resolve() error {\n" + body + "\n\treturn nil\n}\n\nvar _ = srderr.Guard\n"
}

// overlay returns the clean module with files added, plus a telemetry package.
func overlay(files map[string]string) fstest.MapFS {
	fsys := clean()
	fsys["telemetry/doc.go"] = file(telemetryDoc)
	for p, src := range files {
		fsys[p] = file(src)
	}
	return fsys
}

type guardCase struct {
	name  string
	files map[string]string
	rule  string // "" means the overlay must raise nothing
}

func runGuardCases(t *testing.T, tests []guardCase) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := rules(t, overlay(tc.files))
			if tc.rule == "" && len(got) != 0 {
				t.Fatalf("want silence, got %v", got)
			}
			if tc.rule != "" && !slices.Contains(got, tc.rule) {
				t.Fatalf("want rule %q among %v", tc.rule, got)
			}
		})
	}
}

func TestPortUse(t *testing.T) {
	t.Parallel()
	const rule = archtest.RulePortUse
	runGuardCases(t, []guardCase{
		{"value-object file wraps a port", map[string]string{"d20/rules.go": "package d20\n\nimport \"example.com/m/srderr\"\n\nfunc f() error { return srderr.WrapPort(\"p\", \"x\", nil) }\n"}, rule},
		{"value-object file names PortError", map[string]string{"d20/rules.go": "package d20\n\nimport \"example.com/m/srderr\"\n\nfunc f(e *srderr.PortError) {}\n"}, rule},
		{"kernel file calls GuardValue", map[string]string{"srderr/x.go": "package srderr\n", "modifier/x.go": "package modifier\n\nimport \"example.com/m/srderr\"\n\nvar _ = srderr.GuardValue[int]\n", "modifier/doc.go": "// Package modifier is a fixture.\npackage modifier\n"}, rule},
		{"aliased import", map[string]string{"d20/rules.go": "package d20\n\nimport e \"example.com/m/srderr\"\n\nfunc f() { e.Guard() }\n"}, rule},
		{"ContractViolation", map[string]string{"d20/rules.go": "package d20\n\nimport \"example.com/m/srderr\"\n\nvar _ = srderr.ContractViolation\n"}, rule},
		{"type with method but no telemetry is not a use case", map[string]string{"d20/rules.go": "package d20\n\nimport \"example.com/m/srderr\"\n\n// T is a type.\ntype T struct{}\n\n// M is a method.\nfunc (T) M() { srderr.Guard() }\n"}, rule},
		{"use-case file may wrap ports", map[string]string{"d20/resolver.go": useCase("\t_ = srderr.WrapPort")}, ""},
		{"srderr itself", map[string]string{"srderr/wrap.go": "package srderr\n\n// WrapPort wraps.\nfunc WrapPort() {}\n\nfunc g() { WrapPort() }\n"}, ""},
		{"port owner package", map[string]string{"dice/x.go": "package dice\n\nimport \"example.com/m/srderr\"\n\nfunc f() { srderr.Guard() }\n"}, ""},
		{"other package Guard", map[string]string{"d20/x.go": "package d20\n\nimport \"example.com/m/dice\"\n\nvar _ = dice.Guard\n"}, ""},
		{"local func named Guard", map[string]string{"d20/x.go": "package d20\n\n// Guard is local.\nfunc Guard() {}\n\nfunc f() { Guard() }\n"}, ""},
		{"comment mentions PortError", map[string]string{"d20/x.go": "package d20\n\n// f returns a *srderr.PortError.\nfunc f() {}\n"}, ""},
		{"test file", map[string]string{"d20/x_test.go": "package d20\n\nimport \"example.com/m/srderr\"\n\nvar _ = srderr.Guard\n"}, ""},
		{"support package", map[string]string{"dice/dicetest/x.go": "package dicetest\n\nimport \"example.com/m/srderr\"\n\nvar _ = srderr.Guard\n"}, ""},
	})
}

func TestVariantSets(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleVariant
	sealed := func(doc string) string {
		return "package d20\n\n" + doc + "type Result interface{ result() }\n\n// A is a variant.\ntype A struct{}\n\nfunc (A) result() {}\n\n// B is a variant.\ntype B struct{}\n\nfunc (*B) result() {}\n"
	}
	runGuardCases(t, []guardCase{
		{"generic type", map[string]string{"d20/x.go": "package d20\n\n// Applied is generic.\ntype Applied struct{}\n"}, rule},
		{"generic const", map[string]string{"d20/x.go": "package d20\n\n// Rejected is generic.\nconst Rejected = 1\n"}, rule},
		{"generic grouped const", map[string]string{"d20/x.go": "package d20\n\nconst (\n\t// Found is generic.\n\tFound = iota\n)\n"}, rule},
		{"doc misses one implementer", map[string]string{"d20/x.go": sealed("// Result is closed: A only.\n")}, rule},
		{"doc misses pointer implementer", map[string]string{"d20/x.go": sealed("// Result is closed: A and nothing else.\n")}, rule},
		{"no doc at all", map[string]string{"d20/x.go": sealed("")}, rule},
		{"word boundary", map[string]string{"d20/x.go": sealed("// Result is closed: As and Bs.\n")}, rule},
		{"implementer in another file", map[string]string{"d20/x.go": "package d20\n\n// Result is closed: A.\ntype Result interface{ result() }\n\n// A is a variant.\ntype A struct{}\n\nfunc (A) result() {}\n", "d20/y.go": "package d20\n\n// C is a variant.\ntype C struct{}\n\nfunc (C) result() {}\n"}, rule},
		{"fully documented", map[string]string{"d20/x.go": sealed("// Result is closed: exactly [A] and [B].\n")}, ""},
		{"generic type in kernel", map[string]string{"dice/x.go": "package dice\n\n// Applied is fine in the kernel.\ntype Applied struct{}\n"}, ""},
		{"near-miss names", map[string]string{"d20/x.go": "package d20\n\n// AppliedEffect is fine.\ntype AppliedEffect struct {\n\tFound bool\n}\n\n// NotFoundError is fine.\ntype NotFoundError struct{}\n\n// Applied is a method here.\nfunc (AppliedEffect) Applied() {}\n\nconst applied = 1\n"}, ""},
		{"test declaration", map[string]string{"d20/x_test.go": "package d20\n\ntype Applied struct{}\n"}, ""},
		{"exported-only interface", map[string]string{"d20/x.go": "package d20\n\n// I has exported methods.\ntype I interface{ M() }\n\n// T implements I.\ntype T struct{}\n\n// M is a method.\nfunc (T) M() {}\n"}, ""},
		{"no implementers", map[string]string{"d20/x.go": "package d20\n\n// S is sealed.\ntype S interface{ s() }\n"}, ""},
		{"partial implementer", map[string]string{"d20/x.go": "package d20\n\n// S is sealed.\ntype S interface {\n\ts()\n\tt()\n}\n\n// T has only one marker.\ntype T struct{}\n\nfunc (T) s() {}\n"}, ""},
	})
}

func TestErrorShape(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleErrorShape
	port := func(name, value string) string {
		return "package " + name + "\n\n// Port names the port.\nconst Port = " + value + "\n"
	}
	runGuardCases(t, []guardCase{
		{"Message field on an error", map[string]string{"d20/x.go": "package d20\n\n// RollError fails.\ntype RollError struct{ Message string }\n"}, rule},
		{"DisplayText field on a result", map[string]string{"d20/x.go": "package d20\n\n// FooResult is a result.\ntype FooResult struct{ DisplayText string }\n"}, rule},
		{"Text method on a sealed implementer", map[string]string{"d20/x.go": "package d20\n\n// Outcome is closed: [Rejected2].\ntype Outcome interface{ outcome() }\n\n// Rejected2 is a variant.\ntype Rejected2 struct{}\n\nfunc (Rejected2) outcome() {}\n\n// Text must not exist.\nfunc (Rejected2) Text() string { return \"\" }\n"}, rule},
		{"Description method on an error type", map[string]string{"d20/x.go": "package d20\n\n// BadError is an error.\ntype BadError struct{}\n\n// Error is the log text.\nfunc (BadError) Error() string { return \"\" }\n\n// Description must not exist.\nfunc (BadError) Description() string { return \"\" }\n"}, rule},
		{"duplicate port", map[string]string{"d20/p.go": port("d20", `"dice_roller"`)}, rule},
		{"uppercase port", map[string]string{"ability/p.go": port("ability", `"Dice"`), "ability/doc.go": "// Package ability is a fixture.\npackage ability\n"}, rule},
		{"hyphen port", map[string]string{"d20/p.go": port("d20", `"dice-roller"`)}, rule},
		{"empty port", map[string]string{"d20/p.go": port("d20", `""`)}, rule},
		{"computed port", map[string]string{"d20/p.go": port("d20", `"a" + "b"`)}, rule},
		{"content value object", map[string]string{"content/doc.go": "// Package content is a fixture.\npackage content\n", "content/spell.go": "package content\n\n// SpellResult holds SRD text.\ntype SpellResult struct{ Description string }\n"}, ""},
		{"telemetry descriptor", map[string]string{"telemetry/meter.go": "package telemetry\n\n// Descriptor describes a signal.\ntype Descriptor struct{ Description string }\n"}, ""},
		{"support package", map[string]string{"telemetry/telemetrytest/doc.go": "// Package telemetrytest is a fixture.\npackage telemetrytest\n", "telemetry/telemetrytest/l.go": "package telemetrytest\n\n// LogResult is a record.\ntype LogResult struct{ Message string }\n"}, ""},
		{"unexported field", map[string]string{"d20/x.go": "package d20\n\n// RollError fails.\ntype RollError struct{ message string }\n"}, ""},
		{"plain type", map[string]string{"d20/x.go": "package d20\n\n// Config configures.\ntype Config struct{ Text string }\n"}, ""},
		{"two distinct ports", map[string]string{"d20/p.go": port("d20", `"d20_thing"`)}, ""},
	})
}

func TestEmbeddedData(t *testing.T) {
	t.Parallel()
	const dataGo = "// Package data is a fixture.\npackage data\n\nimport \"embed\"\n\n// FS holds the JSON.\n//\n//go:embed *.json\nvar FS embed.FS\n"
	notice := func(scope string) string {
		return "# Notice\n\n## Scope\n\n" + scope + "\n\n## Attribution\n\nSee `elsewhere/` here.\n"
	}
	content := map[string]string{"content/doc.go": "// Package content is a fixture.\npackage content\n"}
	with := func(extra map[string]string) map[string]string {
		for k, v := range content {
			extra[k] = v
		}
		return extra
	}
	runGuardCases(t, []guardCase{
		{"notice misses the path", with(map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "NOTICE.md": notice("Covers nothing.")}), archtest.RuleNoticePaths},
		{"notice names an extra path", with(map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "NOTICE.md": notice("Covers `content/data/` and `foo/`.")}), archtest.RuleNoticePaths},
		{"path without trailing slash counts as missing", with(map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "NOTICE.md": notice("Covers `content/data`.")}), archtest.RuleNoticePaths},
		{"notice is exact", with(map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "NOTICE.md": notice("Covers `content/data/`; see `NOTICE.md`.")}), ""},
		{"no bundled content", with(map[string]string{"content/data/.gitkeep": "", "content/data/README.md": "x", "NOTICE.md": notice("Covers nothing.")}), ""},
		{"no content/data directory", map[string]string{"NOTICE.md": notice("Covers nothing.")}, ""},
		{"non-json file", with(map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "content/data/notes.txt": "x", "NOTICE.md": notice("`content/data/`")}), archtest.RuleEmbed},
		{"two go files", with(map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "content/data/more.go": "package data\n", "NOTICE.md": notice("`content/data/`")}), archtest.RuleEmbed},
		{"json without go file", with(map[string]string{"content/data/x.json": "{}", "NOTICE.md": notice("`content/data/`")}), archtest.RuleEmbed},
		{"go file without go:embed", with(map[string]string{"content/data/x.json": "{}", "content/data/data.go": "// Package data is a fixture.\npackage data\n", "NOTICE.md": notice("`content/data/`")}), archtest.RuleEmbed},
		{"embed outside content/data", map[string]string{"dice/x.go": "package dice\n\nimport _ \"embed\"\n"}, archtest.RuleEmbed},
		{"embed in a test file", map[string]string{"dice/x_test.go": "package dice\n\nimport _ \"embed\"\n"}, ""},
		{"data imports a context", with(map[string]string{"character/doc.go": "// Package character is a fixture.\npackage character\n", "content/data/data.go": "// Package data is a fixture.\npackage data\n\nimport _ \"example.com/m/character\"\n"}), archtest.RuleImport},
		{"content imports data", with(map[string]string{"content/data/doc.go": "// Package data is a fixture.\npackage data\n", "content/x.go": "package content\n\nimport _ \"example.com/m/content/data\"\n"}), archtest.RuleImport},
		{"d20 imports data", map[string]string{"content/data/doc.go": "// Package data is a fixture.\npackage data\n", "d20/x.go": "package d20\n\nimport _ \"example.com/m/content/data\"\n"}, archtest.RuleImport},
		{"test file imports data", map[string]string{"content/data/x.json": "{}", "content/data/doc.go": dataGo, "NOTICE.md": notice("`content/data/`"), "d20/x_test.go": "package d20_test\n\nimport _ \"example.com/m/content/data\"\n"}, ""},
	})
}

func TestDocsOnly(t *testing.T) {
	t.Parallel()
	runGuardCases(t, []guardCase{
		{"declaration in the root", map[string]string{"doc.go": "// Package srd5e is a fixture.\npackage srd5e\n\n// X is not allowed.\nconst X = 1\n"}, archtest.RuleDocsOnly},
		{"declaration in scenarios", map[string]string{"scenarios/doc.go": "// Package scenarios is a fixture.\npackage scenarios\n\nfunc helper() {}\n"}, archtest.RuleDocsOnly},
		{"scenarios test file", map[string]string{"scenarios/doc.go": "// Package scenarios is a fixture.\npackage scenarios\n", "scenarios/a_test.go": "package scenarios\n\nfunc helper() {}\n"}, ""},
		{"scenarios external test only", map[string]string{"scenarios/a_test.go": "package scenarios_test\n\nfunc helper() {}\n"}, ""},
	})
}

func TestSentinelVars(t *testing.T) {
	t.Parallel()
	runGuardCases(t, []guardCase{
		{"lowercase err var", map[string]string{"dice/x.go": "package dice\n\nvar errX = error(nil)\n"}, archtest.RuleVar},
		{"Err var with a literal", map[string]string{"dice/x.go": "package dice\n\nvar ErrCount = 5\n"}, archtest.RuleVar},
		{"Err var of another type", map[string]string{"dice/x.go": "package dice\n\nvar ErrCount int\n"}, archtest.RuleVar},
		{"Err sentinel typed error", map[string]string{"dice/x.go": "package dice\n\n// ErrY is a sentinel.\nvar ErrY error = nil\n"}, ""},
	})
}

func TestHiddenEffectsFire(t *testing.T) {
	t.Parallel()
	for _, call := range []string{"os.Exit(1)", "time.Sleep(0)", "_ = time.After(0)", "_ = time.Tick(0)", "_ = time.Since(time.Time{})", "fmt.Printf(\"\")"} {
		runGuardCases(t, []guardCase{{call, map[string]string{"dice/x.go": "package dice\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"time\"\n)\n\nvar _, _, _ = fmt.Sprint, os.Getpid, time.Now\n\nfunc f() { " + call + " }\n"}, archtest.RuleEffect}})
	}
	runGuardCases(t, []guardCase{
		{"recover in a support package", map[string]string{"dice/dicetest/g.go": "package dicetest\n\nfunc g() { _ = recover() }\n"}, ""},
		{"sync in dice", map[string]string{"dice/x.go": "package dice\n\nimport _ \"sync\"\n"}, archtest.RuleStdlib},
		{"adapter imported by a context", map[string]string{"telemetry/slogtelemetry/doc.go": "// Package slogtelemetry is a fixture.\npackage slogtelemetry\n", "d20/x.go": "package d20\n\nimport _ \"example.com/m/telemetry/slogtelemetry\"\n"}, archtest.RuleImport},
	})
}

// ruleFiles is a module with two valid rule files, a matching AGENTS.md and
// fake tag values.
func ruleFiles() map[string]string {
	return map[string]string{
		"doc/rules/architecture.md": "# A\n\nRULE-TAG: `architecture-0123456789ab`\n",
		"doc/rules/go.md":           "# G\n\nRULE-TAG: `go-abcdef012345`\n",
		"AGENTS.md":                 "# Agents\n\n## Read the rules first\n\n- `doc/rules/architecture.md`\n- `doc/rules/go.md`\n\n## Other\n",
	}
}

func TestRuleTags(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleRuleTag
	mod := func(edit func(map[string]string)) map[string]string {
		f := ruleFiles()
		edit(f)
		return f
	}
	runGuardCases(t, []guardCase{
		{"clean rule files", ruleFiles(), ""},
		{"non-md file in doc/rules", mod(func(f map[string]string) { f["doc/rules/notes.txt"] = "RULE-TAG: junk\n" }), ""},
		{"spec file with its own tag", mod(func(f map[string]string) { f["doc/specs/s.md"] = "RULE-TAG: `s-0000000000aa`\n" }), ""},
		{"tag-like word in prose", mod(func(f map[string]string) { f["README.md"] = "the RULE-TAG: label\n" }), ""},
		{"zero tag lines", mod(func(f map[string]string) { f["doc/rules/go.md"] = "# G\n" }), rule},
		{"two tag lines", mod(func(f map[string]string) { f["doc/rules/go.md"] += "RULE-TAG: `go-abcdef012345`\n" }), rule},
		{"indented tag line is not counted", mod(func(f map[string]string) { f["doc/rules/go.md"] = "  RULE-TAG: `go-abcdef012345`\n" }), rule},
		{"stem mismatch", mod(func(f map[string]string) { f["doc/rules/go.md"] = "RULE-TAG: `golang-abcdef012345`\n" }), rule},
		{"uppercase hex", mod(func(f map[string]string) { f["doc/rules/go.md"] = "RULE-TAG: `go-ABCDEF012345`\n" }), rule},
		{"eleven digits", mod(func(f map[string]string) { f["doc/rules/go.md"] = "RULE-TAG: `go-abcdef01234`\n" }), rule},
		{"thirteen digits", mod(func(f map[string]string) { f["doc/rules/go.md"] = "RULE-TAG: `go-abcdef0123456`\n" }), rule},
		{"no backticks", mod(func(f map[string]string) { f["doc/rules/go.md"] = "RULE-TAG: go-abcdef012345\n" }), rule},
		{"value shared by two rule files", mod(func(f map[string]string) { f["doc/rules/go.md"] = "RULE-TAG: `architecture-0123456789ab`\n" }), rule},
		{"value copied into AGENTS.md", mod(func(f map[string]string) { f["AGENTS.md"] += "go-abcdef012345\n" }), rule},
		{"value copied into .github", mod(func(f map[string]string) { f[".github/pr.md"] = "go-abcdef012345\n" }), rule},
		{"value copied into a Go comment", mod(func(f map[string]string) { f["dice/x.go"] = "package dice\n\n// go-abcdef012345\n" }), rule},
		{"value repeated in its own file", mod(func(f map[string]string) { f["doc/rules/go.md"] += "again go-abcdef012345\n" }), rule},
		{"rule file not listed", mod(func(f map[string]string) { f["doc/rules/testing.md"] = "RULE-TAG: `testing-00ff00ff00ff`\n" }), rule},
		{"listed file missing", mod(func(f map[string]string) {
			f["AGENTS.md"] += "x\n"
			f["AGENTS.md"] = strings.Replace(f["AGENTS.md"], "## Other", "- `doc/rules/gone.md`\n\n## Other", 1)
		}), rule},
	})
}

func TestEachNamedRuleExists(t *testing.T) {
	t.Parallel()
	for _, r := range []string{archtest.RulePortUse, archtest.RuleVariant, archtest.RuleErrorShape, archtest.RuleNoticePaths, archtest.RuleEmbed, archtest.RuleRuleTag, archtest.RuleDocsOnly} {
		if r == "" || strings.Contains(r, "_") {
			t.Errorf("rule name %q must be non-empty kebab-case", r)
		}
	}
}

func TestConsumerNameInGithubDir(t *testing.T) {
	t.Parallel()
	fsys := clean()
	fsys[".github/workflows/ci.yml"] = file("name: " + strings.ToLower("whealing") + "hold\n")
	if !slices.Contains(rules(t, fsys), archtest.RuleConsumer) {
		t.Fatal("consumer name in .github/ not reported")
	}
}
