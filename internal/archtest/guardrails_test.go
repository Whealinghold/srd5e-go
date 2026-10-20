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
		{"embed file under another name carries the package comment", with(map[string]string{"content/data/x.json": "{}", "content/data/bundled_source.go": dataGo, "NOTICE.md": notice("`content/data/`")}), ""},
		{"embed file under another name has no package comment", with(map[string]string{"content/data/x.json": "{}", "content/data/bundled_source.go": "package data\n\nimport \"embed\"\n\n// FS holds the JSON.\n//\n//go:embed *.json\nvar FS embed.FS\n", "NOTICE.md": notice("`content/data/`")}), archtest.RulePackageDoc},
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

// Requirement: architecture.md section 1 (content/contenttest row); plan-phase-2.md step 0.
// Path: content/contenttest may import content and srderr only.
func TestContentTestPolicy(t *testing.T) {
	t.Parallel()
	ct := func(imp string) map[string]string {
		return map[string]string{
			"content/doc.go":              "// Package content is a fixture.\npackage content\n",
			"content/contenttest/doc.go":  "// Package contenttest is a fixture.\npackage contenttest\n",
			"content/contenttest/fake.go": "package contenttest\n\nimport _ \"" + imp + "\"\n",
			"character/doc.go":            "// Package character is a fixture.\npackage character\n",
		}
	}
	runGuardCases(t, []guardCase{
		{"imports content", ct("example.com/m/content"), ""},
		{"imports srderr", ct("example.com/m/srderr"), ""},
		{"imports a context", ct("example.com/m/character"), archtest.RuleImport},
		{"imports d20", ct("example.com/m/d20"), archtest.RuleImport},
		{"imports dice", ct("example.com/m/dice"), archtest.RuleImport},
		{"imports another test package", ct("example.com/m/dice/dicetest"), archtest.RuleImport},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-148 - the identity files
// of package content import only the standard library and srderr, never context
// or telemetry, and never refer to PortError.
// Path: per case.
func TestIdentityFiles(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleIdentity
	id := func(name, body string) map[string]string {
		return map[string]string{
			"content/doc.go":  "// Package content is a fixture.\npackage content\n",
			"content/" + name: "package content\n" + body,
		}
	}
	runGuardCases(t, []guardCase{
		{"stdlib and srderr", id("id.go", "\nimport (\n\t\"strconv\"\n\t\"example.com/m/srderr\"\n)\n\nvar _ = strconv.Itoa\nvar _ = srderr.ErrInvalidInput\n"), ""},
		{"imports context", id("id.go", "\nimport _ \"context\"\n"), rule},
		{"imports telemetry", id("key.go", "\nimport _ \"example.com/m/telemetry\"\n"), rule},
		{"imports dice", id("kind.go", "\nimport _ \"example.com/m/dice\"\n"), rule},
		{"names PortError", id("entry.go", "\nimport \"example.com/m/srderr\"\n\nvar _ *srderr.PortError\n"), rule},
		{"context allowed outside the identity files", id("source.go", "\nimport _ \"context\"\n"), ""},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-128, FR-130 - every
// closed vocabulary type, CreatureSpace and typed id is declared exactly once
// in the module, in package content.
// Path: per case.
func TestVocabularySingleDeclaration(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleVocabulary
	content := func(files map[string]string) map[string]string {
		files["content/doc.go"] = "// Package content is a fixture.\npackage content\n"
		return files
	}
	runGuardCases(t, []guardCase{
		{"declared once in content", content(map[string]string{"content/skill.go": "package content\n\n// Skill is a skill.\ntype Skill int\n"}), ""},
		{"a typed id declared once", content(map[string]string{"content/ids.go": "package content\n\n// SpellID is an id.\ntype SpellID string\n"}), ""},
		{"declared twice in content", content(map[string]string{
			"content/a.go": "package content\n\n// Skill is a skill.\ntype Skill int\n",
			"content/b.go": "package content\n\n// Skill is a skill.\ntype Skill string\n",
		}), rule},
		{"declared in a type group twice", content(map[string]string{
			"content/a.go": "package content\n\ntype (\n\t// Language is one.\n\tLanguage int\n\t// DamageType is one.\n\tDamageType int\n)\n",
			"content/b.go": "package content\n\n// DamageType is one.\ntype DamageType int\n",
		}), rule},
		{"redeclared in another context", content(map[string]string{
			"content/a.go": "package content\n\n// Skill is a skill.\ntype Skill int\n",
			"d20/skill.go": "package d20\n\n// Skill is a skill.\ntype Skill int\n",
		}), rule},
		{"declared only outside content", content(map[string]string{"d20/skill.go": "package d20\n\n// Alignment is one.\ntype Alignment int\n"}), rule},
		{"an unrelated type name", content(map[string]string{"d20/skill.go": "package d20\n\n// Ability is one.\ntype Ability int\n"}), ""},
		{"a method or field named like a vocabulary type", content(map[string]string{"d20/x.go": "package d20\n\n// T has a field.\ntype T struct{ Skill int }\n"}), ""},
		{"a test file may reuse the name", content(map[string]string{
			"content/a.go":      "package content\n\n// Skill is a skill.\ntype Skill int\n",
			"content/a_test.go": "package content\n\ntype Skill int\n",
		}), ""},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-130 - the check
// reports a vocabulary type that has no declaration, so deleting one cannot
// pass the real-module test.
// Path: unhappy (missing content).
func TestMissingVocabulary(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(overlay(map[string]string{"content/doc.go": "// Package content is a fixture.\npackage content\n", "content/s.go": "package content\n\n// Skill is a skill.\ntype Skill int\n"}))
	if err != nil {
		t.Fatal(err)
	}
	missing := archtest.MissingVocabulary(m)
	if slices.Contains(missing, "Skill") || !slices.Contains(missing, "DamageType") || len(missing) != len(archtest.VocabularyTypes())-1 {
		t.Errorf("MissingVocabulary = %v, want every type but Skill", missing)
	}
}

// Requirement: doc/specs/content-source-port.md FR-033, FR-049 - the value-type
// files of the Source port import none of context, embed, testing, contenttest
// or data and never refer to PortError; contenttest/fake_source.go imports only
// content and the standard library.
// Path: per case.
func TestSourceFiles(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleSourceFiles
	in := func(dir, name, body string) map[string]string {
		return map[string]string{
			dir + "/doc.go":  "// Package " + dir[strings.LastIndex(dir, "/")+1:] + " is a fixture.\npackage " + dir[strings.LastIndex(dir, "/")+1:] + "\n",
			dir + "/" + name: body,
		}
	}
	val := func(name, imports string) map[string]string {
		return in("content", name, "package content\n\nimport (\n"+imports+")\n")
	}
	fake := func(imports string) map[string]string {
		return in("content/contenttest", "fake_source.go", "package contenttest\n\nimport (\n"+imports+")\n")
	}
	runGuardCases(t, []guardCase{
		{"catalogue with stdlib and srderr", val("catalogue.go", "\t\"errors\"\n\t_ \"example.com/m/srderr\"\n\t_ \"slices\"\n"), ""},
		{"lookup result imports context", val("lookup_result.go", "\t_ \"context\"\n"), rule},
		{"query imports embed", val("query.go", "\t_ \"embed\"\n"), rule},
		{"catalogue imports testing", val("catalogue.go", "\t_ \"testing\"\n"), rule},
		{"catalogue imports contenttest", val("catalogue.go", "\t_ \"example.com/m/content/contenttest\"\n"), rule},
		{"catalogue imports data", val("catalogue.go", "\t_ \"example.com/m/content/data\"\n"), rule},
		{"query names PortError", in("content", "query.go", "package content\n\nimport \"example.com/m/srderr\"\n\nvar _ *srderr.PortError\n"), rule},
		{"context is allowed in other content files", val("source.go", "\t_ \"context\"\n"), ""},
		{"fake imports content", fake("\t_ \"example.com/m/content\"\n\t_ \"slices\"\n"), ""},
		{"fake imports srderr", fake("\t_ \"example.com/m/content\"\n\t_ \"example.com/m/srderr\"\n"), rule},
		{"fake imports data", fake("\t_ \"example.com/m/content/data\"\n"), rule},
		{"other contenttest files may import srderr", in("content/contenttest", "contract.go", "package contenttest\n\nimport _ \"example.com/m/srderr\"\n"), ""},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-073, FR-075 - only the files named decode*.go of
// package content import encoding/json; no other production file does.
// Path: per case.
func TestJSONConfined(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleJSONConfined
	in := func(files map[string]string) map[string]string {
		files["content/doc.go"] = "// Package content is a fixture.\npackage content\n"
		// A decode.go makes FR-099 apply, so the fixture names the 32-bit run.
		for _, p := range []string{".github/workflows/ci.yml", "AGENTS.md", "doc/specs/release-0-1-0.md"} {
			files[p] = "GOARCH=386 go test ./content/...\n"
		}
		return files
	}
	imp := "package content\n\nimport _ \"encoding/json\"\n"
	runGuardCases(t, []guardCase{
		{"decode.go", in(map[string]string{"content/decode.go": imp}), ""},
		{"decode_value.go", in(map[string]string{"content/decode_value.go": imp}), ""},
		{"another file of content", in(map[string]string{"content/problem.go": imp}), rule},
		{"a name that only contains decode", in(map[string]string{"content/jsondecode.go": imp}), rule},
		{"a test file of content", in(map[string]string{"content/problem_test.go": "package content_test\n\nimport _ \"encoding/json\"\n"}), ""},
		{"another context", in(map[string]string{"d20/x.go": "package d20\n\nimport _ \"encoding/json\"\n"}), rule},
		{"a decode file of another package", in(map[string]string{"d20/decode.go": "package d20\n\nimport _ \"encoding/json\"\n"}), rule},
		{"the data package", in(map[string]string{"content/data/doc.go": "// Package data is a fixture.\npackage data\n", "content/data/x.go": "package data\n\nimport _ \"encoding/json\"\n"}), rule},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-141 - in package content only load_usecase.go
// refers to PortError or to the srderr guard and wrap helpers.
// Path: per case.
func TestContentPortUse(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleContentPortUse
	in := func(name, body string) map[string]string {
		return map[string]string{
			"content/doc.go":    "// Package content is a fixture.\npackage content\n",
			"content/source.go": "package content\n\n// Port names the port.\nconst Port = \"content\"\n",
			"content/" + name:   "package content\n\nimport \"example.com/m/srderr\"\n\n" + body,
		}
	}
	runGuardCases(t, []guardCase{
		{"load_usecase.go names PortError and Guard", in("load_usecase.go", "var _ *srderr.PortError\n\nvar _ = srderr.GuardValue[int]\n"), ""},
		{"another file names PortError", in("decode.go", "var _ *srderr.PortError\n"), rule},
		{"another file calls Guard", in("set.go", "var _ = srderr.Guard\n"), rule},
		{"another file calls WrapPort", in("set.go", "var _ = srderr.WrapPort\n"), rule},
		{"another file calls ContractViolation", in("set.go", "var _ = srderr.ContractViolation\n"), rule},
		{"another file uses ErrInvalidInput only", in("set.go", "var _ = srderr.ErrInvalidInput\n"), ""},
		{"a comment may mention PortError", in("set.go", "// Unlike a PortError, this is input failing validation.\nvar _ = srderr.ErrInvalidInput\n"), ""},
		{"a test file of content", in("set_test.go", "var _ *srderr.PortError\n"), ""},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-075 - package content imports none of os, net and
// io/fs (A3).
// Path: per case.
func TestContentDeniesFileAndNetworkPackages(t *testing.T) {
	t.Parallel()
	in := func(imp string) map[string]string {
		return map[string]string{
			"content/doc.go": "// Package content is a fixture.\npackage content\n",
			"content/x.go":   "package content\n\nimport _ \"" + imp + "\"\n",
		}
	}
	runGuardCases(t, []guardCase{
		{"os", in("os"), archtest.RuleStdlib},
		{"net", in("net"), archtest.RuleStdlib},
		{"net/http", in("net/http"), archtest.RuleStdlib},
		{"io/fs", in("io/fs"), archtest.RuleStdlib},
		{"io", in("io"), ""},
		{"bytes", in("bytes"), ""},
	})
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-099 - once the decoder exists, the CI workflow,
// AGENTS.md and the release-0-1-0 spec each name the 32-bit run.
// Path: per case.
func TestThirtyTwoBit(t *testing.T) {
	t.Parallel()
	const rule = archtest.RuleThirtyTwoBit
	const cmd = "GOARCH=386 go test ./content/..."
	base := func() map[string]string {
		return map[string]string{
			"content/doc.go":    "// Package content is a fixture.\npackage content\n",
			"content/decode.go": "package content\n",
		}
	}
	with := func(files map[string]string) map[string]string {
		out := base()
		for p, s := range files {
			out[p] = s
		}
		return out
	}
	all := map[string]string{".github/workflows/ci.yml": "- run: " + cmd + "\n", "AGENTS.md": "just test-386 # " + cmd + "\n", "doc/specs/release-0-1-0.md": "CI runs `" + cmd + "`.\n"}
	without := func(name string) map[string]string {
		out := map[string]string{}
		for p, s := range all {
			if p == name {
				s = "nothing about it\n"
			}
			out[p] = s
		}
		return with(out)
	}
	runGuardCases(t, []guardCase{
		{"before the decoder exists", map[string]string{"content/doc.go": "// Package content is a fixture.\npackage content\n"}, ""},
		{"all three name it", with(all), ""},
		{"the three files are missing", base(), rule},
		{"ci.yml lacks it", without(".github/workflows/ci.yml"), rule},
		{"AGENTS.md lacks it", without("AGENTS.md"), rule},
		{"the release spec lacks it", without("doc/specs/release-0-1-0.md"), rule},
		{"only the release spec is missing", with(map[string]string{".github/workflows/ci.yml": all[".github/workflows/ci.yml"], "AGENTS.md": all["AGENTS.md"]}), rule},
	})
}
