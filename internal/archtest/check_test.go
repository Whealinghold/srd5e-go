package archtest_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

const (
	goMod = "module example.com/m\n\ngo 1.24\n"
	// doc builds a documented package clause.
	docDice = "// Package dice is a fixture.\npackage dice\n"
)

func file(src string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(src)} }

// clean is a module that satisfies every rule; each negative case overlays it.
func clean() fstest.MapFS {
	return fstest.MapFS{
		"go.mod":               file(goMod),
		"doc.go":               file("// Package srd5e is a fixture.\npackage srd5e\n"),
		"srderr/doc.go":        file("// Package srderr is a fixture.\npackage srderr\n"),
		"srderr/guard.go":      file("package srderr\n\n// Guard recovers.\nfunc Guard() { _ = recover() }\n"),
		"dice/doc.go":          file(docDice),
		"dice/dice.go":         file("package dice\n\nimport \"example.com/m/srderr\"\n\n// Roll rolls.\nfunc Roll() { srderr.Guard() }\n\n// Port names the port.\nconst Port = \"dice_roller\"\n\n// ErrX is a sentinel.\nvar ErrX = error(nil)\n"),
		"dice/dicetest/doc.go": file("// Package dicetest is a fixture.\npackage dicetest\n"),
		"d20/doc.go":           file("// Package d20 is a fixture.\npackage d20\n"),
		"d20/x_test.go":        file("package d20_test\n\nimport _ \"example.com/m/dice/dicetest\"\n"),
	}
}

func rules(t *testing.T, fsys fstest.MapFS) []string {
	t.Helper()
	m, err := archtest.Load(fsys)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var out []string
	for _, v := range archtest.Check(m, archtest.DefaultPolicy()) {
		out = append(out, v.Rule)
	}
	return out
}

func TestCleanFixtureHasNoViolations(t *testing.T) {
	t.Parallel()
	if got := rules(t, clean()); len(got) != 0 {
		t.Fatalf("clean fixture: got violations %v", got)
	}
}

// Each case overlays the clean module and must raise exactly its rule.
func TestEachRuleFires(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"unlisted package", map[string]string{"widget/doc.go": "// Package widget is a fixture.\npackage widget\n"}, archtest.RuleUnlisted},
		{"kernel imports context", map[string]string{"dice/up.go": "package dice\n\nimport _ \"example.com/m/d20\"\n"}, archtest.RuleKernelUp},
		{"import not in allow-list", map[string]string{"dice/up.go": "package dice\n\nimport _ \"example.com/m/ability\"\n"}, archtest.RuleImport},
		{"production imports test support", map[string]string{"d20/up.go": "package d20\n\nimport _ \"example.com/m/dice/dicetest\"\n"}, archtest.RuleIsolation},
		{"internal import", map[string]string{"dice/x_test.go": "package dice_test\n\nimport _ \"example.com/m/internal/archtest\"\n"}, archtest.RuleInternal},
		{"stdlib deny-list", map[string]string{"srderr/x.go": "package srderr\n\nimport _ \"os\"\n"}, archtest.RuleStdlib},
		{"stdlib deny-list nested net", map[string]string{"srderr/x.go": "package srderr\n\nimport _ \"net/http\"\n"}, archtest.RuleStdlib},
		{"rand v2 outside dice", map[string]string{"srderr/x.go": "package srderr\n\nimport _ \"math/rand/v2\"\n"}, archtest.RuleStdlib},
		{"third-party import", map[string]string{"dice/x.go": "package dice\n\nimport _ \"github.com/stretchr/testify/assert\"\n"}, archtest.RuleThirdParty},
		{"third-party in test", map[string]string{"dice/x_test.go": "package dice_test\n\nimport _ \"github.com/stretchr/testify/assert\"\n"}, archtest.RuleThirdParty},
		{"go.mod require", map[string]string{"go.mod": goMod + "\nrequire example.org/x v1.0.0\n"}, archtest.RuleGoMod},
		{"init function", map[string]string{"dice/x.go": "package dice\n\nfunc init() {}\n"}, archtest.RuleInit},
		{"package-level var", map[string]string{"dice/x.go": "package dice\n\nvar counter int\n"}, archtest.RuleVar},
		{"time.Now call", map[string]string{"dice/x.go": "package dice\n\nimport \"time\"\n\nfunc f() { _ = time.Now() }\n"}, archtest.RuleEffect},
		{"go statement", map[string]string{"dice/x.go": "package dice\n\nfunc f() { go f() }\n"}, archtest.RuleEffect},
		{"fmt.Println", map[string]string{"dice/x.go": "package dice\n\nimport \"fmt\"\n\nfunc f() { fmt.Println() }\n"}, archtest.RuleEffect},
		{"println builtin", map[string]string{"dice/x.go": "package dice\n\nfunc f() { println() }\n"}, archtest.RuleEffect},
		{"panic outside Must", map[string]string{"dice/x.go": "package dice\n\nfunc f(err error) { panic(err) }\n"}, archtest.RulePanic},
		{"panic with string", map[string]string{"dice/x.go": "package dice\n\nfunc MustX() { panic(\"boom\") }\n"}, archtest.RulePanic},
		{"recover outside guard", map[string]string{"dice/x.go": "package dice\n\nfunc f() { _ = recover() }\n"}, archtest.RulePanic},
		{"undocumented exported func", map[string]string{"dice/x.go": "package dice\n\nfunc Exported() {}\n"}, archtest.RuleDoc},
		{"doc not starting with name", map[string]string{"dice/x.go": "package dice\n\n// Does a thing.\nfunc Exported() {}\n"}, archtest.RuleDoc},
		{"undocumented exported method", map[string]string{"dice/x.go": "package dice\n\n// T is a type.\ntype T struct{}\n\n// M is documented.\nfunc (T) M() {}\n\nfunc (T) Bad() {}\n"}, archtest.RuleDoc},
		{"undocumented type in group", map[string]string{"dice/x.go": "package dice\n\ntype (\n\t// A is documented.\n\tA int\n\tB int\n)\n"}, archtest.RuleDoc},
		{"missing package doc", map[string]string{"dice/doc.go": "package dice\n"}, archtest.RulePackageDoc},
		{"package doc wrong name", map[string]string{"dice/doc.go": "// Package wrong is a fixture.\npackage dice\n"}, archtest.RulePackageDoc},
		{"stutter", map[string]string{"dice/x.go": "package dice\n\n// DiceRoll stutters.\ntype DiceRoll int\n"}, archtest.RuleStutter},
		{"consumer name", map[string]string{"README.md": "built for " + strings.ToUpper("whealing") + "HOLD\n"}, archtest.RuleConsumer},
		{"consumer name in hidden dir", map[string]string{".github/workflows/ci.yml": "name: " + strings.ToLower("whealing") + "hold\n"}, archtest.RuleConsumer},
		{"consumer name in nested AGENTS.md", map[string]string{"doc/AGENTS.md": "built for " + strings.ToLower("whealing") + "hold\n"}, archtest.RuleConsumer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := clean()
			for p, src := range tc.files {
				fsys[p] = file(src)
			}
			if tc.want == archtest.RuleInternal {
				fsys["internal/archtest/doc.go"] = file("// Package archtest is a fixture.\npackage archtest\n")
			}
			got := rules(t, fsys)
			found := false
			for _, r := range got {
				found = found || r == tc.want
			}
			if !found {
				t.Fatalf("want rule %q among %v", tc.want, got)
			}
		})
	}
}

// Allowed forms must stay silent so the rules do not over-fire.
func TestAllowedFormsAreSilent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files map[string]string
	}{
		{"panic in Must and default", map[string]string{"dice/x.go": "package dice\n\n// MustX panics.\nfunc MustX(err error) { panic(err) }\n\nfunc f(n int) {\n\tswitch n {\n\tdefault:\n\t\tpanic(\"unreachable\"[:0] + \"\")\n\t}\n}\n"}},
		{"time.Now as value", map[string]string{"dice/x.go": "package dice\n\nimport \"time\"\n\nvar _ = time.Now\n"}},
		{"principal type equals package", map[string]string{"dice/x.go": "package dice\n\n// Dice is the principal type.\ntype Dice int\n"}},
		{"grouped consts documented once", map[string]string{"dice/x.go": "package dice\n\n// Sizes of dice.\nconst (\n\tSmall = 1\n\tLarge = 2\n)\n"}},
		{"rand v2 in dice", map[string]string{"dice/x.go": "package dice\n\nimport _ \"math/rand/v2\"\n"}},
		{"root AGENTS.md may name the organisation", map[string]string{"AGENTS.md": "hosted by " + strings.ToLower("whealing") + "hold\n"}},
		{"module path in text", map[string]string{"README.md": "import example.com/m/dice\n"}},
		{"testdata ignored", map[string]string{"dice/testdata/bad.go": "this is not go"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := clean()
			for p, src := range tc.files {
				fsys[p] = file(src)
			}
			if got := rules(t, fsys); len(got) != 0 {
				t.Fatalf("got violations %v", got)
			}
		})
	}
}

func TestImportCycleIsReported(t *testing.T) {
	t.Parallel()
	fsys := clean()
	fsys["dice/up.go"] = file("package dice\n\nimport _ \"example.com/m/ability\"\n")
	fsys["ability/doc.go"] = file("// Package ability is a fixture.\npackage ability\n\nimport _ \"example.com/m/dice\"\n")
	var found bool
	for _, r := range rules(t, fsys) {
		found = found || r == archtest.RuleCycle
	}
	if !found {
		t.Fatal("cycle dice -> ability -> dice not reported")
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files fstest.MapFS
	}{
		{"missing go.mod", fstest.MapFS{}},
		{"no module line", fstest.MapFS{"go.mod": file("go 1.24\n")}},
		{"unparsable go file", fstest.MapFS{"go.mod": file(goMod), "x.go": file("not go")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := archtest.Load(tc.files); err == nil {
				t.Fatal("Load succeeded; want an error")
			}
		})
	}
}

func TestViolationString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		v    archtest.Violation
		want string
	}{
		{archtest.Violation{Rule: "r", File: "a/b.go", Line: 3, Msg: "m"}, "a/b.go:3: r: m"},
		{archtest.Violation{Rule: "r", File: "a", Msg: "m"}, "a: r: m"},
		{archtest.Violation{Rule: "r", Msg: "m"}, "module: r: m"},
	}
	for _, tc := range tests {
		if got := tc.v.String(); got != tc.want {
			t.Errorf("String() = %q, want %q", got, tc.want)
		}
	}
}
